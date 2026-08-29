# TASK-260829-3cwcb6 — Cross-runtime measurement audit

## Scope and conclusion

Audited the production `benchmark-run` path from launcher-profile resolution through live SSE parsing, per-scenario rate construction, memory sampling, record sealing, admission, and decision scoring. The accepted TASK-260828-2wcrph speculation reader remained intact.

Two known metric defects are corrected at their production entry:

1. TTFT/prefill/decode now recognize `content`, `reasoning`, and `reasoning_content` as the same generated-event boundary. Decode ends on the last generated event, not the response tail.
2. Mach `ri_phys_footprint` is not scored for an exact `llama-server` executable or any `.gguf` artifact. All memory values are unmeasured and the existing decision gate refuses them.

The audit found no additional demonstrated numeric defect that makes llama.cpp look worse. It did find two structural policies/limitations with a direction against the challenger; both are carried into every production record rather than hidden.

## Direction table

| Surface | Status | Direction | Record treatment |
| --- | --- | --- | --- |
| Missing `delta.reasoning_content` | Corrected | Mixed: llama.cpp TTFT worse, prefill worse, decode better | One canonical first/last generated-event reader; old values withdrawn |
| Decode ended at usage/`[DONE]` | Corrected | Unknown; hurts a runtime in proportion to its post-token framing delay | Decode interval is first-to-last generated event |
| Mach footprint on mmap-loaded GGUF | Refused as unmeasurable | Old reading strongly favoured llama.cpp | All footprint fields `nil`; limitation says the old reading favours that runtime; decision blocks |
| `/slots` top-level `speculative` shadowed by `params` | Previously corrected, verified retained | Old quiet reading favoured llama.cpp when it drafted | Both placements read; any `true` settles `on`; absent/unread remain distinct and refused |
| Fixed pass order: baseline then candidate | Remaining limitation | Residual heat/pressure can hurt llama.cpp as candidate; host-cache carry-over can favour it; net indeterminate | Built-in declared asymmetry on both records; per-pass load remains diagnostic, not a gate |
| One-way parity policy | Intentional migration policy, not a metric | Favours incumbent: baseline success/candidate failure blocks, reverse does not | Built-in declared asymmetry names the direction and policy boundary |
| MTP disabled for parity | Intentional comparable-algorithm constraint | Against llama.cpp's product advantage; it cannot use a capability the incumbent lacks | Speculation pin/refusal plus trusted model non-equivalence record |
| Python deployed prompt cache retained | Intentional deployed-baseline condition | Favours incumbent if it works; measured run showed llama.cpp slot reuse instead | Caller declaration plus scenario transcript; not presented as a pure decode metric |
| Loopback/client timestamps and SSE chunk grouping | Remaining limitation | Neither established | Usage counts supply tokens; generated-delta events supply endpoints; no direction claimed |

## Production call sites and negatives

- Latency production call site: `BenchmarkRunCommand.execute` → `drive` → `BenchmarkScenarios.run` → `BenchmarkHTTPDriver.stream` → `RuntimeStreamDelta.read`.
  - Production negative: `scripts/benchmark-gate-smoke.sh` launches one stand-in publishing `reasoning` and one publishing `reasoning_content`. Both spend the first three events reasoning and the last event on content. The real `benchmark-run` must accept and both records must carry comparable non-null TTFT/decode. Restoring the old reader leaves the candidate with only the final content event, so decode becomes unmeasured and the entry refuses.
- Memory production call site: `BenchmarkRunCommand.drive` derives `RuntimeMemoryAccounting` from the reviewed profile executable, model artifact, and launch argv, injects it into `BenchmarkFootprintSampler`, and every scenario, soak, warm-up, and process reading routes through that sampler.
  - Production negative: the smoke launches a normally observed candidate whose reviewed argv carries a `.gguf` artifact. `benchmark-run` must exit rejected on `peak_physical_footprint_bytes was not measured`; the candidate record must omit every optional footprint and carry the directional limitation. The old implementation exits accepted with a scored Mach number. Using the ordinary observed executable is deliberate: an earlier copied-interpreter fixture was refused by the independent process-observation gate before memory admission, so it could not prove this production boundary.
- Speculation production call site retained: `BenchmarkRunCommand.speculationAnswer` → `RuntimeSpeculation.read`. Existing unit and smoke negatives cover top-level/`params` disagreement, unread observations, and a drafting runtime that must not score as off.

## Adversarial self-review

- Narrowed reasoning support to `reasoning` only: the `reasoning_content` unit case fails and the production pair loses candidate decode.
- Narrowed memory detection to exact executable only: the `.gguf` classifier test fails; renamed wrappers cannot restore scoring for the shipped artifact shape.
- Bypassed the sampler through soak or warm-up: code-path audit found both former direct `physicalFootprintBytes` calls and routed them through `BenchmarkPass.currentMemoryBytes`; the production record asserts every scenario and process footprint is `nil`.
- Tried to spend unknown memory as zero: no conversion exists; `RuntimeBenchmark.decide` already adds a blocker whenever either scored value is absent.
- Could not make a defensible numeric claim about candidate-second thermal bias from 1-minute load average. It remains explicitly directional-risk/indeterminate, not a correction disguised as a threshold.
- Could not prove a renamed llama.cpp binary serving a non-`.gguf` artifact has the same mmap behavior. That unsupported shape remains outside the classifier; the shipped exact executable and artifact are both covered.

## Evidence

| Command | Exit | Result |
| --- | ---: | --- |
| `swift test --filter RuntimeStreamDeltaTests` | 0 | 5 tests / 2 suites passed on the initial correction |
| `swift test --filter RuntimeMemoryAccountingTests` | 0 | 3 tests / 1 suite passed after widening the fail-closed classifier to reviewed `.gguf` launch artifacts |
| `swift test -c release` | 0 | Final source: 398 tests / 32 suites passed; only three pre-existing deprecation warnings |
| canonical `xcodebuild build ... -configuration Release ...` | 0 | Final macOS arm64 Release product and Metal bundle build remained valid |
| `scripts/benchmark-gate-smoke.sh` attempt 1 | 1 | Honest fixture failure: copied/wrapped executable hit `attestation opened and never closed`; optional JSON `nil` was also incorrectly asserted as a present key |
| `scripts/benchmark-gate-smoke.sh` attempt 2 | 1 | JSON assertion fixed, but the copied-interpreter fixture still hit the independent observation refusal |
| `scripts/benchmark-gate-smoke.sh` final | 0 | `BENCHMARK GATE SMOKE OK (0 failures)`, including both reasoning spellings, mmap-memory refusal, and speculation fail-closed cases |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | Strict Swift formatting clean |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | Production smoke script clean |
| `git diff --check` | 0 | Patch whitespace clean |

The first two smoke failures were not relabelled as passes. They exposed that executable substitution was the wrong negative shape, so the final fixture drives the memory gate through the real launch profile using a reviewed `.gguf` argv token without defeating process observation.

No 28 GiB model rerun was performed in this task; TASK-260829-3k4qrc owns the full measured rerun after review accepts this instrumentation.
