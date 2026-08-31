# TASK-260829-3k4qrc — revision 4 results

Scope: F3 and F4 from the revision-3 verdict, plus one defect the F4 fix
exposed. **The hour-scale pinned pair was again not run**, per the brief. It is
now worth running: a scored memory dimension is demonstrably reachable at the
production entry.

Base `3272e3ae60b53eb0fb61f6d427a883487711b48f`, candidate tree
`bafa676bada78f5cdb6481326977086a20aaf7e7`, patch SHA-256
`f0011b033f1a86b5c103dc1de5f3930c876528f0b013a90e798511cee45514c3`.

## F3 — the memory dimension had an empty admissible set

Direction taken: the first one the verdict offered. The bound is now **derived
from the cadence the mapped component is actually read at**, not asserted.

| Quantity | Revision 3 | Revision 4 | Basis |
| --- | ---: | ---: | --- |
| `observedMappedFileReadCostSeconds` | — | 1.0 s | measured, rounded up |
| `maximumMappedFileSampleGapSeconds` | 0.125 s | 7.0 s | `samplingInterval + 2 x readCost` |
| `maximumPhysicalFootprintSampleGapSeconds` | 0.125 s | 0.125 s | unchanged |

Reader cost measured on this host through the production `Process`/`Pipe`/
`waitUntilExit` shape, 8 consecutive calls per target: **0.608–0.672 s** against
a ~3 MB target, **0.783–0.850 s** against a ~0.9 GiB target. Cost rises with the
target's mapping count, so the 28 GiB runtimes are not read faster. 1.0 s is the
rounded-up larger figure; the bound adds one cadence plus one further reader
cost of scheduling headroom.

The Mach component keeps its independent 20 Hz series and its own 125 ms bound.
That separation — the good part of revision 3 — is untouched.

What the wider bound costs is **stated, not implied**. Every emitted
`RuntimeMemoryPeak` now carries `mappedFileObservationLimitSeconds` and a
`mappedFileObservabilityNote` saying mapped-file transients shorter than that
cadence are not observable, and `validatedScoredBytes` returns nil for a
`measured` peak that omits them or claims a different cadence. Four decode-time
mutants cover that (limit removed, limit forged to 0.125, note removed, note
replaced) against an untouched control that still scores.

The reason this is sound for *this* measurement: the mapped-file component that
matters is the model weights, resident for the whole run. A 125 ms claim over
them bought nothing real. Sub-cadence *anonymous* growth is the genuine risk and
is covered by the Mach series at 20 Hz.

## F4 — the suite could not tell "refused correctly" from "never scores"

- The control now requires **exit 0**, `accepted=true`, no blockers, and a
  scored memory delta with numeric baseline and candidate on both the process
  and at least one scenario. It fails when memory cannot be measured.
- The mmap-memory assertions dropped `assert_peak_or_coverage_refusal` entirely
  and require `measured` peaks and scored deltas.
- The refusal branch keeps its own separate fixture:
  `benchmark-memory-coverage-refusal-probe` drives the same production sampler
  class with the bound narrowed to 125 ms and requires it to refuse while the
  unnarrowed control on the same shape scores. The override can only narrow —
  `BenchmarkFootprintSampler` clamps it to the contract bound.

Every admitted-path check is back at **exit 0**. Revision 3 had all of them at
exit 3.

## Third defect, found by the F4 tightening

`BenchmarkFootprintSampler.stop()` set one flag for both loops, so the 20 Hz
Mach loop exited immediately while the 5 s `vmmap` loop finished an in-flight
read that landed one reader-cost later. Every process-wide series ended with a
hole exactly one `vmmap` cost wide, and `BenchmarkRunCommand.swift:674-675`
carried it straight into the whole-process memory delta. In the pre-fix smoke,
**20 of 20** sessions had a `partial` process peak on the larger runtime, tail
gaps 0.358–0.494 s against a 125 ms bound.

`stop()` now retires the slow loop first, waits for its read to land with the
fast loop still covering that interval, then retires the fast loop. Costs
nothing — the old code already waited for the same thread.
`benchmark-memory-stop-coverage-probe` reproduces the race deliberately and
fails if it did not occur; with the simultaneous-stop mutant restored it exits 1
with `mach-physical-footprint-sampling-gap`.

A blackout hypothesis (vmmap stalling `proc_pid_rusage` on the same target) was
tested and **rejected** — 6 runs, max gap 0.060–0.063 s, no stall.

## Validation

| Check | Result |
| --- | --- |
| `swift build -c release` | exit 0 |
| `swift test -c release` | exit 0, 410 tests / 32 suites (rev 3: 407) |
| `swift format lint --strict` | exit 0 |
| Four production-entry probes | exit 0 each |
| Teardown mutant probe | exit 1, as required |
| `benchmark-gate-smoke.sh` run 3 | 119 PASS / 1 FAIL, exit 1 |

The single smoke FAIL was the new series-coverage assertion applied to
`warmupMemory`, which is one synchronous point reading rather than a sampled
window and is read by no decision. The assertion was scoped to series and the
exact block replayed against run 3's own artifacts (exit 0). **No production
source changed after run 3**, and a fourth full run was not performed because
the task caps the suite at two full runs.

## Still unknown, carried forward and not closed here

Whether llama.cpp's **streaming** usage frame reports
`prompt_tokens_details.cached_tokens`. The baseline does. If the candidate does
not, `cacheReuseObservation` returns `unknown` and every scenario becomes
non-comparable — the same failure shape as F3 applied to every dimension at
once. One short streamed request against `llama-server` answers it and should
precede the pair.
