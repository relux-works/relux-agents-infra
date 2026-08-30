# TASK-260830-2hc5r2 — revision 2 review verdict

## Verdict

**Changes requested.** Route Change Request `CR-TASK-260830-2hc5r2-2`
revision 2 to `to-dev`.

Reviewed base `f47ffe01bb9f758fd7007aae012bfe76004c278e`, candidate tree
`3cfd196cc588445b550c7c2143959391e54a99be`, and patch SHA-256
`51be834687c5c770f5e35f2859f6deb16187637b2ef362533d67a0421562a929`.
The external source fork was reviewed separately at clean commit
`ec9eea0af1d741cd4eb21c8766478a3e79dd44d6`.

## Blocking finding

### F1 — caller-supplied `--python-bin` can attest a different runtime

Revision 2 records the immutable `mlx-lm` commit by running
`BenchmarkRunPins.pythonRevisions(python:)` against the interpreter supplied by
the caller through `benchmark-run --python-bin`. Nothing in the production
entry point ties that interpreter to the baseline profile's
`provenance.launchExecutable` or to the process that served the benchmark.

The reviewer drove the exact candidate binary's real `benchmark-run` entry
point with:

- baseline profile executable `/usr/bin/python3`, launching the smoke
  `fake-runtime.py` rather than `mlx_lm.server`;
- `--python-bin` pointing at the isolated
  `mlx-lm-kv76800-ec9eea0` interpreter;
- the ordinary live smoke profiles, requests, attestations, records, and
  admission path; no record was edited after the run.

Actual result: `exit 0`, `decision.accepted=true`. The accepted baseline record
simultaneously says:

```text
revisions.mlx_lm_direct_url.commit_id = ec9eea0af1d741cd4eb21c8766478a3e79dd44d6
provenance.launchExecutable = /Applications/Xcode.app/.../Python.app/.../Python
provenance.launchArgv[0] = .../fake-runtime.py
```

The pinned commit therefore describes a caller-selected decoy interpreter, not
the runtime that produced the measurements. This is the standard **forged or
self-minted evidence** shape and also a **bypass path around the check**. It is
the residual form of revision-1 F2: renaming the wrapper and recording an
immutable `direct_url.json` does not make runtime provenance sensitive to the
imported implementation when the provenance reader is independently supplied.

Required rework:

1. Derive Python package provenance from the actual configured launch
   executable/process. For a console-script wrapper, resolve and validate its
   interpreter and inspect that interpreter's installed distribution; an
   equivalent live runtime-reported immutable build identity is also valid.
2. If `--python-bin` remains, fail closed unless it is proven to be the
   interpreter behind the baseline profile executable. An unreadable or
   mismatched relationship must produce no decision.
3. Add a production-entry negative test that launches one runtime while
   supplying a different pinned interpreter through `--python-bin`, and require
   refusal. The current positive profile-string/direct-url evidence does not
   cover this shape.
4. Add the finding and its closure as append-only Logbook evidence during
   rework; the reviewer did not edit the candidate or `LOGBOOK.md`.

## Confirmed implementation evidence

- `qwen3_5.make_cache(max_kv_size:)` constructs `RotatingKVCache(max_size:
  76800, keep: 4)` for full-attention layers and preserves `ArraysCache` for
  linear layers; the no-bound path still constructs `KVCache`.
- Seeded single and batch server paths carry the bound. The revision-1 mixed
  bounded/unbounded draft bypass now fails closed, and `/v1/models` omits
  `meta` for that shape.
- The source fork is clean at immutable commit `ec9eea0...`; the isolated
  install's `direct_url.json` currently names that commit correctly. This fact
  is real but not bound to the measured process by the production gate.
- Reviewer rerun in the source repository: 115 Python model/server tests, one
  skip, exit 0. A first attempt with the isolated benchmark venv ran 83 tests
  but could not import `test_server` because that environment lacks the
  test-only `requests` dependency; it is retained as a red environment attempt
  and not counted as a pass.
- Exact candidate archive: Release build exit 0; 20 focused Swift
  `RuntimeBenchmarkContextBoundTests` pass.
- The tracked producer log contains the raw live response: 73,139 prompt
  tokens, 73,111 cached, 67 completion tokens, `finish_reason=stop`, and the
  exact correct answer naming the coolant loop, intake manifold, and telemetry
  uplink. The same live server then reported `meta.n_ctx=76800`.
- The correctness prompt remained below the bound by 3,661 tokens. This proves
  the benchmark's exact-margin workload survives; it does not prove semantic
  retention after rotation beyond 76,800, and is not represented as such.
- The 16-token and 96-token attempts remain visibly separate: both ended
  `finish_reason=length`; neither is counted as the correctness pass. The smoke
  invocation errors are likewise preserved separately from the final green
  smoke.
- The candidate modifies only the benchmark-only profile spelling and contains
  no `[profiles.qwen-local]`; the server option defaults to `None`, preserving
  the deployed unbounded default behavior.

## Reviewer evidence

- `TASK-260830-2hc5r2_review-forged-python-provenance-rev2.log`
- `TASK-260830-2hc5r2_review-verdict-rev2.md`

The reviewer made no repository code changes.
