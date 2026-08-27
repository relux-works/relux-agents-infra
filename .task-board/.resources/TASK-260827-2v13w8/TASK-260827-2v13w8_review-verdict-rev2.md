# TASK-260827-2v13w8 Review Verdict — Revision 2

## Verdict

**CHANGES REQUESTED.** Route `TASK-260827-2v13w8` to `to-dev`. Do not accept
`CR-TASK-260827-2v13w8-2` revision 2.

The migration decision itself is conservatively correct: **REJECT MLX Swift as
the default runtime and keep Python `mlx-lm` as the default**. The change request
is not acceptable as benchmark evidence because the admission gate still accepts
fully caller-minted records, the two profiles rendered different prompts, and the
serving measurements were not produced by the final candidate binary.

Reviewed snapshot:

- Base OID: `3f313d9175f2ada9b9ab3320ab524c0918f9daac`
- Candidate tree OID: `aeeb329eb11b2cc02fcd930a8668bda1497d4b53`
- Serving binary SHA-256: `19c54c56f3db6544ec319016483a4a6aa2f538bde9783687a2c0925d58dc90b9`
- Final/judging binary SHA-256: `3e5fdcc522d844a154a7d2b60b473f58d64eda56db348f40cc48baffc3c09430`

## Confirmed fixes and results

1. The text-only factory issue from round 1 is closed. The final binary reported
   `MLXLLM.LLMModelFactory`, completed the five ordinary scenarios, and completed
   an independently driven 73,016-token capacity request with HTTP 200,
   `prompt_tokens=73016`, `completion_tokens=16`, and the process alive afterward.
2. Both rendered launch argv values explicitly pin `--prefill-step-size 2048`.
   `contextPolicy` is re-derived as `kv=unbounded;prefill-step=2048`; the round-1
   512-versus-2048 asymmetry is closed.
3. The reported 8k threshold failures are arithmetically honest:

   | Metric | Swift / Python | Threshold | Result |
   | --- | ---: | ---: | --- |
   | 8k prefill throughput | `0.824x` | `>= 0.90x` | fail |
   | 8k TTFT | `1.208x` | `<= 1.10x` | fail |
   | 8k scenario footprint | `1.159x` | `<= 1.10x` | fail |

   These three failures make REJECT the honest conservative direction even
   before the invalid short-prompt comparison is removed.
4. Tool-call parity and the bounded 20-request soak passed independently on the
   final binary. The independent capacity pass took `1575.334s`, versus the
   producer's approximately `1318s`; capacity is confirmed, while the single-run
   timing variance reinforces that it should not be treated as a precise score.
5. The real oversized-allocation probe supports the bounded `MLX.withError`
   claim: HTTP 500, process survives, health remains 200, batch release is emitted,
   and a following request succeeds.

## Findings requiring rework

### R2-A — Critical: comparison evidence remains fully self-mintable

Production call chain:

- `BenchmarkCompareCommand.run(arguments:)`
- `RuntimeBenchmark.admit(baseline:candidate:thresholds:)`
- `RuntimeBenchmark.admitProvenance(_:)`

Revision 2 correctly refuses the round-1 records that omit `launchProvenance`:
that attack now exits 4. It does not solve the trust boundary. Every provenance
field still lives inside the same caller-authored JSON record. Admission verifies
only internal consistency among caller-controlled values.

I reran the production smoke's fully populated pair. That pair fabricates its
sampled pid, time interval, driver and launcher argv, config and executable
digests, rendered launch argv, revisions, pins, timings, and measurements. The
production `benchmark-compare` binary with SHA-256 `3e5fdcc...` returned exit 0,
`accepted=true`, no blockers, and unit ratios. No runtime or benchmark driver
earned any part of that evidence.

This is the **forged or self-minted evidence** negative shape. The producer's
14-check smoke labels this pair as its accepted control, so the positive-path
control is also the bypass. The mutant campaign narrows individual clauses but
does not test a generalized forgery that supplies every current clause.

Required rework:

- Bind admission to driver-owned materialization or attestation that the record
  author cannot synthesize merely by filling JSON fields; alternatively keep
  measurement and comparison inside one trusted production run rather than
  accepting unsigned external records as earned evidence.
- Add a production-entry negative that populates every current provenance field
  consistently and requires refusal. It must drive
  `BenchmarkCompareCommand.run(arguments:)`, not only call the helper.
- Keep the round-1 missing-provenance refusal as a separate absent-evidence test.

Independent raw decision: `TASK-260827-2v13w8_review-forged-decision-rev2.json`.

### R2-B — High: the runtimes did not render the same prompt policy

The Swift profile passes `--reasoning-effort medium`. The Python profile passes no
equivalent `--chat-template-args`, so this model's Jinja template defaults to
`xhigh` and injects an additional system instruction.

Independent tokenizer rendering established the cause:

| Prompt | Python profile default | Explicit `medium` | Delta |
| --- | ---: | ---: | ---: |
| `short_prompt` | `79` | `41` | `+38` |
| `long_prompt_8k` | `7821` | `7783` | `+38` |
| `context_75k` | `73053` | `73015` | `+38` |

The runtime records add one token to the long and context render counts but
preserve the same difference (`7822/7784`, `73054/73016`). Thus B4's reported
`79/41 = 1.927x` is a real harness defect, not a runtime regression. Python is
the misconfigured side relative to the stated common `medium` policy.

`RuntimeBenchmark.contextPolicy(derivedFrom:)` binds KV and prefill chunking but
does not bind reasoning/template arguments, so the gate also admits this
asymmetry.

Required rework:

- Pin the same explicit reasoning/template policy in both rendered launch argv
  values and bind it in admission.
- Rerun every scenario for both runtimes. The mismatch affects every prompt, not
  only `short_prompt`; B4 must not be scored against either runtime.

### R2-C — High: the final candidate binary is not the measured serving binary

The raw serving measurements remain valid historical observations about binary
`19c54c...`. Judging the records with `3e5fdcc...` does not by itself change their
arithmetic. It does, however, invalidate the stronger claim that the current
candidate source and final executable were benchmarked: the candidate now builds
`3e5fdcc...`, while the exact `19c54c...` source snapshot/build artifact is not
attached. A byte-identical `ready` event proves startup equivalence only, not the
entire serving path.

Required rework: rerun with the final binary after fixing the common prompt
policy. That single rerun resolves both the evidence comparability and binary
identity gaps. If old measurements are retained as history, label them as results
for `19c54c...`, not the final candidate.

### R2-D — Contract limit: task-local MLX recovery is not process-wide recovery

`GenerationEngine.run` correctly uses `MLX.withError` and checks `ErrorBox` after
streamed items. The disclosed limit is also real: the handler is task-local, so an
MLX error raised on an MLX-owned `asyncEval` thread still reaches the global
default and traps.

Consequences:

- `TASK-260827-2h39ya` (dead-generation health) guarantees health invalidation
  only for failures delivered as throws. A trap cannot emit the in-process 503 or
  `generation_worker_unavailable`; recovery must begin with supervisor detection
  of process death.
- `TASK-260827-2q77g8` (batch failure recovery) can attest batch release and
  teardown only when control unwinds through Swift. A trap skips those events and
  cleanup is performed by process termination; replacement is the recovery path.

Those task contracts must remain explicitly scoped to thrown/task-local failures,
or gain a separate process-death/restart production test. Revision 2 must not be
read as proof that arbitrary MLX backend faults are survivable in-process.

## Validation performed by review

- Exact candidate delta: `git diff 3f313d9175f2ada9b9ab3320ab524c0918f9daac aeeb329eb11b2cc02fcd930a8668bda1497d4b53`
- `git diff --check` on the exact delta: pass.
- `swift test -c release`: 253 tests in 21 suites passed.
- Xcode Release build: passed; final executable digest reproduced as `3e5fdcc...`.
- `xcrun swift-format lint --strict --recursive Sources Tests`: pass.
- `shellcheck -S warning scripts/*.sh`: pass.
- `python3 -m py_compile scripts/runtime-benchmark.py`: pass.
- Focused Go install-sync tests: pass.
- `scripts/benchmark-gate-smoke.sh`: its 14 declared checks pass, while its
  fabricated accepted control demonstrates R2-A.
- Independent final-binary ordinary scenarios: 5/5 passed.
- Independent final-binary 75k capacity scenario: passed in `1575.334s`.
- Independent tokenizer renders: reproduced the exact 79-versus-41 root cause.

The Swift build emits one non-blocking deprecation warning for `quantization` in
`Preflight.swift`; it is not part of this verdict.

Review modified no product code. The only repository documentation mutation is
the mandatory append-only Logbook record of these findings.
