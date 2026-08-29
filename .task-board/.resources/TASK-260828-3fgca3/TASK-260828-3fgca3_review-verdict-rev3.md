# TASK-260828-3fgca3 review verdict — round 3 / CR revision 4

## Verdict

**Changes requested. Route `TASK-260828-3fgca3` to `to-dev`. Do not accept CR revision 4.**

Revision 4 closes the reported `prefix_repeats` bypass, validates the suite before session creation or runtime launch, preserves F1–F3/G1–G4, and keeps `unpinnableConditions` byte-identical. It nevertheless leaves one nested prompt-suite reader outside the claimed complete schema: a misspelled tool `parameters.required` key is treated as the intentionally supported absence of `required`, weakens the tool parity assertion to demand no arguments, and reaches `accepted: true` through the shipped `benchmark-run`.

This is the same negative-evidence shape as F5: **a malformed read presented as a supported absence/default**, plus a bypass path below the two object levels where `unknownKeys` is actually invoked.

Reviewed candidate:

- Change Request: `CR-TASK-260828-3fgca3-4`, revision `4`
- Base OID: `5a287e4bcb454b53bc432cb7788a03792d1c96f6`
- Candidate tree OID: `3e2ff31b50f541039c06764bfedc3aa4dbf2c15b`
- Patch SHA-256: `3df85e50dc7b7daed25518997c9d8db49c261698299dbea57b6623cebb3ea6ad` (matches the board handoff)

## Blocking finding F6 — nested tool-schema typo becomes “no required arguments”

Production path:

1. `PromptSuiteSchema.validate` rejects unknown keys only at the document and scenario objects (`PromptSuiteSchema.swift:198`, `:277`).
2. `toolDeclarations` reads `parameters["required"]` only when that exact key exists; otherwise `requiredArguments` remains `[]` (`PromptSuiteSchema.swift:486-510`). It does not validate unknown keys in the tool declaration, function object, or parameters object.
3. `BenchmarkScenarios.tool` forwards the declaration verbatim, then treats an empty `requiredArguments` list as a satisfied parity check (`BenchmarkScenarios.swift:187-195`, `:229-240`).

Production-entry attack:

- Start from the repository smoke suite and change only `"required": ["vehicle"]` to the misspelled `"require": ["vehicle"]`.
- The shipped current-source `benchmark-run` launches both runtimes, reports `tool_call: ok` for each, exits `0`, and emits `"accepted": true`.
- The smoke runtime constructs returned arguments from the exact `parameters.required` key (`benchmark-gate-smoke.sh:171-180`). With the typo it returns `{}`; the gate nevertheless reports the tool scenario as succeeded because its extracted requirement list is empty.
- Raw output is attached as `TASK-260828-3fgca3_review-rev3-nested-tool-typo-attack.log`; both sealed records in the task-scoped scratch session record `tool_call.succeeded=true`.

The code and documentation claim a broader invariant than the implementation provides. Revision 4's report says unknown keys are refused “at every level”; the new logbook entry says `parameters.required ?? []` and unknown keys “all refuse now”. Neither claim holds below the scenario object.

## Required rework

- Make the benchmark tool schema unambiguous at the parity-reading boundary. Recommended: require an explicit `parameters.required` array for benchmark tool scenarios, allow `[]` to mean deliberately no required arguments, and remove `tool.parameters.required` from `supportedAbsences`. This closes typo-to-absence without pretending to implement a complete allowlist for arbitrary JSON Schema keywords.
- Validate the tool declaration fields the benchmark itself depends on, including `type == "function"`, a function object/name, a parameters object, and the explicit `required` array. Preserve the rest of the JSON Schema verbatim if it is intentionally opaque.
- Add the exact `required` → `require` production-entry negative. It must exit nonzero, emit no decision, and create no session directory.
- Add the narrowing control for an explicit empty `required: []` if no required arguments remain supported, plus the shipped non-empty control.
- Audit and test the remaining nested tool reader boundaries rather than asserting “unknown keys at every level”. Correct the revision-4 logbook/report audit claim and record this regression in the logbook during rework.

## What revision 4 did close

- Original F5 attack: string-valued `context_75k.prefix_repeats` exits `5`, names the field, emits no decision, and creates no session directory.
- Integer control: `prefix_repeats: 2027` reaches the complete workload and is accepted with `16,232 / 16,232` prompt tokens. One first control run hit the already documented fail-closed Python exec-path race; the records still carried 16,232 tokens, and one bounded retry exited `0` with `accepted: true`.
- Validation order: `BenchmarkScenarios.Suite(path:)` is constructed at `BenchmarkRunCommand.swift:167`; session directories are created at `:187-192`, and runtime launch is inside `drive` beginning at `:499`. The malformed control's absent session directory independently proves validation precedes both launches.
- Revision-3 correction: present in `LOGBOOK.md:28` and `.research/260828_llamacpp-in-the-benchmark-gate.md:301-305`.
- F1–F3/G1–G4: the 84-check production smoke passes with zero failures, including caller-minted equivalence, malformed `n_ctx`, failed `/slots`, KV false-match, `-ub`, trusted equivalence, MTP-off and no-verdict refusals.
- `unpinnableConditions` and downstream admission are unchanged: the rev3 and rev4 blobs are identical for `RuntimeBenchmark.swift` (`509304cf4a1b5ad1899011f612de7886319e00ab`), `RuntimeAttestation.swift` (`8214b9d80824d6c94bfdeefd4553fd126191fc77`), and `BenchmarkRunPins.swift` (`3994fb241376164debebedcafd366b9cdadc36b1`).

## Reviewer validation

- `swift build --build-tests`: passed.
- `swift test`: 379 tests / 30 suites passed.
- `scripts/benchmark-gate-smoke.sh`: 84 checks, 0 failures.
- Original malformed F5 production attack: exit 5, no decision, no session directory.
- Integer F5 production control: exit 0, accepted, 16,232 / 16,232 prompt tokens.
- Nested tool typo production attack: exit 0, accepted — blocking finding reproduced.
- `xcrun swift-format lint --strict --recursive Sources Tests`: passed.
- `shellcheck scripts/benchmark-gate-smoke.sh`: passed.
- Exact CR `git diff --check`: passed.

The clean suite is insufficient for acceptance because it never attacks the nested `parameters.required` reader that controls the parity assertion.
