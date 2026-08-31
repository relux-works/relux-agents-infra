# TASK-260829-3cwcb6 Rework Revision 3 Results

Date: 2026-08-29  
Role: developer  
Evidence base: `4332e1dddd0164876b4da3ec0340ba9320aec1e9` plus the Story-worktree candidate delta. The worktree was 24 commits behind local `main`; it was not rebased because the assignment reserves integration for the orchestrator.

## Outcome

Revision 2's symmetric memory collection, honest upper-bound metric name, raw
components, fail-closed read states, reasoning-field clock boundary, MTP
direction and audit for defects biased against llama.cpp remain intact.

The two revision-3 findings are corrected:

1. `RuntimeMemoryComponents.validatedResidentMemoryUpperBoundBytes` now
   re-derives `machPhysicalFootprintBytes + residentMappedFileBytesUpperBound`
   with sign and overflow checks. A decoded document whose stored composite
   disagrees with those raw components exposes no validated score.
2. The production smoke now binds every
   `peak_resident_memory_upper_bound_bytes` decision delta to the corresponding
   generated record composite for baseline and candidate, at every scored
   scenario and at process scope. It also independently re-derives each record
   composite from its raw fields.

`RuntimeBenchmark.decide` is named and driven by the permanent decoded-forgery
negative. The old Mach-only reading makes the consumer report the memory axis
unmeasured and block the decision.

## Negative Evidence

### Decoded-component forgery

The permanent Swift test encoded a valid measured peak `(Mach=100,
mapped=2,048, composite=2,148)`, rewrote both stored composite and score to
`100`, decoded it, and drove `RuntimeBenchmark.decide`.

- Before the validation fix: exit 1; `validatedScoredBytes` returned `100`, the
  decision accepted, and all three refusal assertions failed.
- After the fix: exit 0; the decoded peak exposes no score and the decision
  carries the named unmeasured-memory blocker.

### Mach-only narrowing mutant

The exact reviewer mutant changed only the final valid measured return from the
re-derived composite to `peakSample.machPhysicalFootprintBytes`. Record shape,
metric name, raw components and `scoredBytes` remained unchanged.

The full production smoke exited 1. Its new decision-to-record assertion killed
the mutant:

| Candidate value | Bytes |
| --- | ---: |
| Mach physical footprint consumed by mutant decision | 13,648,328 |
| Resident mapped-file upper component | 11,114,906 |
| Generated record composite / `scoredBytes` | 24,763,234 |

The mutant run contained 15 reported failures. Fourteen were the known,
unrelated intermittent candidate attestation-close cascade and do not support
the kill. The one additional failure was the memory decision-to-record mismatch
above. The source was restored from a scratch copy; before/after SHA-256 was
`3a6218589aacb2ab009034d7c67a6295a592d0b18170841f7b14c2e5a77b3a4e`,
and the final Release product was rebuilt after restoration.

## Validation And Exit Codes

| Command / attempt | Exit | Evidence |
| --- | ---: | --- |
| Focused decoded-forgery test before fix | 1 | Three expected regression assertions failed; production decision accepted the forged `100 B` score |
| Focused decoded-forgery test after fix | 0 | One test / one suite passed |
| `swift test -c release` before mutant | 0 | 401 tests / 32 suites passed |
| Full production smoke attempt 1 | 1 | 14 known attestation-close cascade failures; new timing and memory sections passed; not counted as a clean pass |
| Full production smoke clean retry | 0 | `BENCHMARK GATE SMOKE OK (0 failures)` |
| Mach-only mutant `swift build -c release` | 0 | Mutant production binary built |
| Full production smoke against Mach-only mutant | 1 | 15 failures: 14 unrelated attestation-close failures plus the one supporting decision-to-record mismatch |
| Restored final `swift build -c release` | 0 | Final SwiftPM Release binary rebuilt |
| Restored final `swift test -c release` | 0 | 401 tests / 32 suites passed |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | Clean |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | Clean |
| Canonical Xcode macOS arm64 Release build | 0 | `BUILD SUCCEEDED` |
| Final `git diff --check` | 0 | Clean |

## Revision-3 Scope

- `tools/mlx-swift-runtime-prototype/Sources/MLXSwiftRuntimeContract/RuntimeMemoryAccounting.swift`
- `tools/mlx-swift-runtime-prototype/Tests/MLXSwiftRuntimeContractTests/RuntimeBenchmarkTests.swift`
- `tools/mlx-swift-runtime-prototype/scripts/benchmark-gate-smoke.sh`
- `LOGBOOK.md`

The broader directional-bias audit remains in the existing
`TASK-260829-3cwcb6_rework-results.md` outcome. Revision 3 found no reason to
change its directions: Mach-only favours llama.cpp; the conservative composite
has runtime-dependent residual bias; MTP-off and incumbent parity policy are
against llama.cpp; fixed order remains indeterminate.
