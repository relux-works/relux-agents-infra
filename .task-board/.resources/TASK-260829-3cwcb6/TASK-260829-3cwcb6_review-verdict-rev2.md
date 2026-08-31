# TASK-260829-3cwcb6 Review Verdict — Revision 2

Verdict: **changes requested** (`to-dev`).

Reviewed immutable delta: `4332e1dddd0164876b4da3ec0340ba9320aec1e9` → tree `695239665a2c648971027cfadfac071c229d3a11` for `CR-TASK-260829-3cwcb6-2`, revision 2.

## What revision 2 fixes correctly

- F1's production collection path is symmetric. `RuntimeMemoryAccounting.forExecutable` returns the same `residentMemoryUpperBound` profile for every executable/model/argv shape, and both passes instantiate `BenchmarkFootprintSampler` from that profile. The result is recorded as `peak_resident_memory_upper_bound_bytes`, explicitly `conservative-upper-bound`, with Mach, literal vmmap token, mapped-file upper component, composite, counts, status and issues retained.
- Warm-up and soak memory appear for baseline and candidate in `session.json`; scenario-local and process-window peaks appear in both records. Read failure, malformed, partial and absence are distinct states and do not expose a normal scored value.
- F2 is present in machine-readable output, not only prose. The production smoke requires the MTP-off direction (`direction is against llama.cpp`) in both records and in `decision.json`; the production `declared(...)` path adds it to each pass before comparison.
- Remaining known limitation directions are explicit: fixed-order is indeterminate, parity policy favours the incumbent, MTP-off is against llama.cpp, and conservative-memory residual direction is runtime-dependent.
- Reasoning deltas use one generated-event definition for `reasoning` and `reasoning_content`; the production smoke reached and passed this focused section.

## Blocking finding F1 — the Mach-only narrowing mutant survives

Production scoring is `BenchmarkRunCommand.run` → `RuntimeBenchmark.decide` → `RuntimeMemoryPeak.validatedScoredBytes` (`RuntimeBenchmark.swift:2131-2157`, `RuntimeMemoryAccounting.swift:154-169`). I narrowed only the final measured return from the composite to `peakSample.machPhysicalFootprintBytes`. The new record shape, accounting name, vmmap collection, raw components and `scoredBytes` persisted unchanged.

Results:

- Unmodified revision 2: `swift test -c release` passed, 400 tests / 32 suites.
- Mach-only mutant: `swift test -c release` also passed, 400 tests / 32 suites.
- The production-entry memory section of `scripts/benchmark-gate-smoke.sh` also passed the mutant:
  - `mmap-loaded and anonymous runtime shapes carry the same scored quantity` — PASS.
  - `both records carry scored resident upper bounds and raw components` — PASS.

The mutant's production artifacts show the bypass directly:

| Candidate process value | Bytes |
| --- | ---: |
| Mach physical footprint | 13,468,104 |
| vmmap mapped-file upper component | 11,114,906 |
| record composite / `scoredBytes` | 24,583,010 |
| value actually emitted in decision delta | 13,468,104 |

The smoke checks the metric name and the record's composite fields, but never asserts that the decision delta consumed that composite. It therefore accepts the old Mach-only reading while all new evidence remains present. This is the required narrowing shape from the round-2 brief and violates the acceptance criterion that every corrected metric carry a production-entry negative refusing the old reading.

Evidence logs:

- `.temp/review-TASK-260829-3cwcb6/swift-test-full-unmodified-01.log`
- `.temp/review-TASK-260829-3cwcb6/swift-test-mutant-mach-only-01.log`
- `.temp/review-TASK-260829-3cwcb6/benchmark-gate-smoke-mutant-mach-only-02.log`
- `.temp/review-TASK-260829-3cwcb6/mutant-scored-values-01.log`

The full smoke attempt recorded 14 earlier failures from the known intermittent candidate attestation-close observation (`gate-smoke-candidate` opened but never closed). That unrelated fail-closed flake does not support this verdict; the later reasoning and memory sections executed, and the memory section's PASS on the mutant is the relevant evidence.

## Blocking finding F1b — decoded raw components do not validate the composite

`RuntimeMemoryComponents` uses synthesized `Codable`. A decoded document can retain a non-zero mapped component while changing `residentMemoryUpperBoundBytes` and `RuntimeMemoryPeak.scoredBytes` to the old Mach-only value. `validatedScoredBytes` only compares those two forged values with each other; it does not re-derive `machPhysicalFootprintBytes + residentMappedFileBytesUpperBound`.

A review-only negative encoded a valid measured peak `(Mach=100, mapped=2,048, composite=2,148)`, changed both composite and score to `100`, decoded it through `JSONDecoder`, and required fail-closed. The test failed because `validatedScoredBytes` returned `100` instead of `nil`.

Evidence: `.temp/review-TASK-260829-3cwcb6/swift-test-forged-components-01.log`.

This contradicts `RuntimeMemoryAccounting.swift:154-156`, whose contract says decoded evidence whose components and score disagree is malformed and blocks.

## Required rework

1. Add a production-entry negative that drives `benchmark-run` with a non-zero mapped component and asserts each process/scenario memory delta's baseline and candidate values equal the corresponding validated composite from the generated records. It must fail when scoring is narrowed to Mach-only while the new record shape remains intact.
2. Re-derive and validate the composite from raw Mach + mapped-file components after decode; an internally inconsistent measured peak must expose no score and must block the record consumer.
3. Keep the current symmetric collection path, explicit upper-bound naming, raw components, fail-closed read states, F2 machine-readable direction, and all remaining limitation directions.
4. Rerun the 400-test Release suite and a clean production smoke. Record the killed Mach-only mutant, not only the positive fixture.
5. Add the review regression to `LOGBOOK.md` during producer rework. The reviewer did not edit the repository logbook because this role is read-only.

Logbook-ready entry:

```markdown
### 1702 — Memory Record Shape Did Not Prove The Decision Used The Composite
- REGRESSION: A narrowing mutant returned Mach physical footprint from `RuntimeMemoryPeak.validatedScoredBytes` while retaining the upper-bound record shape and vmmap components. All 400 contract tests and the production smoke's memory section passed; `decision.json` scored 13,468,104 B while the candidate record composite was 24,583,010 B.
- ROOT CAUSE: production smoke asserted record shape and metric name, not equality between decision deltas and the generated composite. Decoded component validation also trusted a forged composite instead of re-deriving Mach plus mapped bytes.
- STATUS: TASK-260829-3cwcb6 revision 2 changes requested; add a production-entry Mach-only negative and fail-closed decoded-component validation.
```
