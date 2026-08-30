# TASK-260830-2hc5r2 — revision 6 round 6 review verdict

## Verdict

**Accepted.** I could not construct a ninth bypass on the production path.
Revision 6 closes the abbreviation finding by removing runtime-argv decoding
from context-policy evidence and deriving KV, prefill, and reasoning only from
the running process's exact model entry. Missing and malformed reports remain
inadmissible; they do not fall back to argv or profile values.

Reviewed Change Request `CR-TASK-260830-2hc5r2-6`, revision 6, candidate tree
`44856df645a4710f6361244940a2a15f38ac99e8`. The published patch SHA-256
recomputed as
`be535462877f9572127b0c0200994b9564b0dcd91481b97cf5fb4f0427c6c644`.
The non-board working tree matches that immutable candidate; the 35 board paths
are control-plane drift already identified by Change Request publication and
are excluded from the candidate.

## Gate attacks rerun

The exact candidate Release binary drove the shipped `benchmark-run` entry
through `scripts/benchmark-gate-smoke.sh`: **52 passes, 0 failures, exit 0**.

- Abbreviated Python flag `--prefill-step-size 2048 --prefill-step-siz 999`:
  exit 4; the live baseline pin is
  `kv=76800;prefill-step=999;reasoning=medium`, versus candidate prefill 2048;
  no accepted decision exists.
- Repeated exact Python flag `--prefill-step-size 2048 --prefill-step-size 999`:
  exit 4 with the same live effective `999`; no accepted decision exists.
- Finite launch `--max-kv-size 76800` with omitted `meta.n_ctx`: exit 4,
  `kv=not-reported`; omission cannot mint acceptance.
- Omitted live `meta.runtime_config`: exit 4,
  `prefill-step=not-reported`. Unit coverage separately distinguishes malformed
  values as `unread`; both states are listed in production
  `RuntimeBenchmark.unpinnableConditions` and are refused even when both sides
  would otherwise match.
- Rewritten process argv is reported as its live `999`, not the profile's
  `2048`, and the pair is refused.
- A decoy serving process plus an unrelated `--python-bin` exits 5 before a
  decision: the observed process did not execute the package-owned entry point.

I also launched the actual installed signed fork twice, once with repeated
exact flags and once with the abbreviation. Its own `/v1/models` response
reported `prefill_step_size=999` and `reasoning_effort=medium` both times.
Before cache construction, `n_ctx` remained absent despite the finite launch
flag, proving that the report does not manufacture a live bound from argv.
Both probes were stopped and their ports were clean afterward.

A caller can influence the effective process configuration through supported
CLI semantics, as expected, but that influence changes the process's report
and therefore the derived pin. A caller cannot substitute a profile claim or
unrelated interpreter for the reporting process. An absent or malformed report
has no accepting branch.

## Source and long-context evidence

- Fork commits `45a472f2d0cda166b7ffe1a80fe50dd9621f4303` and parent
  `0a0452a9ca64d5b8ee3786fb23d3f828417f9514` both verify as good SSH-signed
  commits for the configured human identity. The remote task branch equals
  `45a472f2...`.
- The installed benchmark-only distribution's `direct_url.json` pins both
  `commit_id` and `requested_revision` to `45a472f2...`; package provenance
  verifies RECORD hashes at the production call site.
- The revision-6 fork delta is exactly `mlx_lm/server.py` plus
  `tests/test_server.py` (49 insertions, 5 deletions). The production change is
  confined to `/v1/models` reporting of parsed prefill/reasoning values. It
  does not alter cache construction, generation, sampling, tokenization, or
  prompt-cache behavior.
- Signed parent `0a0452a9...` and the source commit used by the 73k run,
  `ec9eea0a...`, have the identical tree
  `7378b59c77a2add1dce15de9bf099b399867b761`. Therefore revision 6's
  reporting-only delta does not invalidate the inherited generation result.
- I inspected the raw tracked run, not only its summary: 73,139 prompt tokens,
  73,111 cached tokens, 67 completion tokens, `finish_reason=stop`, the correct
  coolant-loop/intake-manifold/telemetry-uplink answer, and live
  `meta.n_ctx=76800` afterward. I did not rerun this hour-scale generation.

The current fork still constructs `RotatingKVCache(max_size=76800, keep=4)` for
Qwen3.5 full-attention layers, retains `ArraysCache` for linear layers, and
retains `KVCache` for the unbounded default. Mixed bounded/unbounded caches and
differently bounded rotating caches derive no active bound. The seeded single
path and batch path both carry the configured bound.

Only `qwen-benchmark-python` and `qwen-benchmark-swift` in the benchmark example
carry 76,800. That file declares no `qwen-local` profile, and the deployed
default profile was not changed.

## Validation

Rerun in this reviewer session:

- fork `python -m unittest tests.test_models tests.test_server`: 116 tests,
  1 skipped, exit 0;
- `swift test -c release --filter RuntimeBenchmarkTests`: 73 tests / 5 suites,
  exit 0;
- `swift test -c release`: 287 tests / 24 suites, exit 0;
- strict `swift-format`, `bash -n`, and `shellcheck -S warning`: exit 0;
- exact production smoke: 52 passes / 0 failures, exit 0;
- signed-commit, remote-ref, installed-direct-url, candidate-patch digest, and
  non-board candidate-tree checks: exit 0.

Two initial Python invocations used environments missing `huggingface_hub` or
`requests`; neither reached the full requested suite and neither is counted as
a pass. Repeating with the producer's test-capable task interpreter produced
the 116-test green result above.

The smoke reproduced the already-recorded decoy-child cleanup debt by leaving
one task-owned listener on port 19691 after its refusal. I inspected its exact
argv, terminated only that process, and confirmed the smoke port range clean.
This does not open an acceptance path and is not a new finding for this task.

## Conclusion

The bounded Python baseline is source-implemented, live-attested, comparable
with the candidate at `kv=76800`, and protected against the previously found
forged, absent, duplicate, abbreviated, decoy, unsigned, and mixed-cache
shapes. Revision 6 is accepted for orchestrator checkpoint/integration.
