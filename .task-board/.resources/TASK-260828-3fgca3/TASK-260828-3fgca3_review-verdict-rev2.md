# TASK-260828-3fgca3 review verdict — round 2 / CR revision 3

## Verdict

**Changes requested. Route `TASK-260828-3fgca3` to `to-dev`. Do not accept CR revision 3.**

Revision 3 closes F1-F3, preserves `unpinnableConditions`, and passes its clean suite. It nevertheless opens a production-entry acceptance path through malformed benchmark-suite fields. A present malformed `prefix_repeats` value is silently treated as absent, so a scenario named and required as `context_75k` can execute a 15-token prompt and still produce `accepted: true`.

This is the negative-evidence shape **failed or malformed read presented as absence/default**. It also makes the producer's claimed audit of every other runtime/configuration reading incomplete.

Reviewed candidate:

- Change Request: `CR-TASK-260828-3fgca3-3`, revision `3`
- Base OID: `5a287e4bcb454b53bc432cb7788a03792d1c96f6`
- Candidate tree OID: `50c6eb38f2aafc417ee6d80a7075c9d2c5668cc4`
- Patch SHA-256: `3e2d25974fea7a3c2dd835242e0e1f4315b6ba4274773b9e327ce006b2408de2` (matches the board handoff)

## Blocking finding F5 — malformed scenario input weakens a required workload

Production call path:

1. `BenchmarkRunCommand.execute` constructs `BenchmarkScenarios.Suite` at `BenchmarkRunCommand.swift:167`.
2. `BenchmarkRunCommand.drive` iterates required scenario names and invokes `BenchmarkScenarios.run` at `BenchmarkRunCommand.swift:561-570`.
3. `BenchmarkScenarios.single` reads `prefix_repeats` with `if let ... as? Int` at `BenchmarkScenarios.swift:121-129`. A present value of the wrong type takes the same path as an absent optional field.

Production-entry attack against the shipped release binary and the same HTTP stand-in used by the repository smoke gate:

- The prompt suite declares required `context_75k` with `"prefix_repeats": "2027"` (a malformed string).
- `benchmark-run` exits `0` and emits `"accepted": true`.
- The supposedly required `context_75k` workload records only **15 prompt tokens** for both baseline and candidate.
- An otherwise identical control with integer `"prefix_repeats": 2027` also exits `0`, but records **16,232 prompt tokens**.

Thus the malformed field did not yield an unknown/refusal. It silently narrowed the workload by more than three orders of magnitude and allowed a false acceptance under the required scenario name. This is not cured by transcript sealing: the gate faithfully seals and scores the wrong, weakened request.

The same permissive pattern appears elsewhere in `BenchmarkScenarios.swift`: `prompt`, `max_tokens`, and additional `prefix_repeats` reads use `as?` plus optional/default branches (notably lines 121-129, 175, 247-249, and 318). The producer's audit therefore did not establish the claimed invariant for all input readings.

Required rework:

- Validate the complete prompt-suite schema before either runtime is launched.
- Distinguish a legitimately absent optional field from a present malformed value.
- Refuse wrong types, invalid ranges, and scenario-incompatible fields rather than defaulting or skipping them.
- Add a negative test through shipped `benchmark-run` for a string-valued `prefix_repeats` on a required scenario; it must exit nonzero and emit no decision.
- Audit every scenario reader (`prompt`, `prefix_repeats`, `max_tokens`, iteration/turn counts and other optional casts) under the same absence-vs-failure rule, and record which absences are intentionally supported.
- Correct the current logbook/audit claim during rework, because revision 3 did not audit this production input surface successfully.

## Required round-2 attacks

All three previously reported bypasses now refuse through the shipped `benchmark-run` production entry:

- **F1 caller-minted equivalence:** a well-shaped verdict over the real fixture files, carrying one generic note instead of the three required non-equivalences, is refused; the command emits no decision.
- **F2 malformed finite context:** `meta.n_ctx` present as string `"32768"` is recorded as an unread reading, refused, and emits no decision.
- **F3 failed speculation observation:** `GET /slots` returning HTTP 500 is recorded as an unread reading, refused, and emits no decision. HTTP 404 remains the intentionally distinct not-reported case.

The repository smoke run reports 68 checks and 0 failures, including those three attacks.

## Trusted equivalence binding

The binding is outside benchmark caller control in the reviewed production path:

- `TrustedEquivalenceDecisions.shipped` is a compile-time registry of the trusted document digest, source of record, artifact digests, and all three non-equivalences.
- The caller may choose a file path but cannot add a digest to the shipped registry.
- `BenchmarkRunCommand.equivalenceReading(path:)` rejects an unregistered document before launch.
- `RuntimeBenchmark.admitModelIdentity` repeats the lookup against the sealed attestation digest.
- Production callers use the default shipped store; custom stores are only injected by tests.

## `unpinnableConditions` and narrowing mutants

The extracted declaration is byte-identical between revision 2 (`d829b94f0b19da46b5ea93aa044b3801624de1d6`) and revision 3:

- SHA-256 both revisions: `4d7c8e6217e09c126cbce28c19d0b5a39926c8e58ef02b3d10a8962e6b4c3f3b`
- Value: `["kv=unread", "prefill-step=unpinned", "reasoning=unpinned"]`

Mutants were rebuilt from revision-3 sources in private scratch packages:

- A8, adding `kv=unbounded`: killed; 351 tests / 29 suites fail with 80 issues.
- A9, removing `prefill-step=unpinned`: killed; 351 tests / 29 suites fail with 10 issues.

This confirms the producer did not relax or extend that set in revision 3 and that the tests bind both directions. It does not mitigate F5.

## Validation performed by this reviewer

- Exact CR patch digest: matched.
- Candidate tree: working source matches `50c6eb38f2aafc417ee6d80a7075c9d2c5668cc4`; `git diff --check` clean.
- `swift build -c release`: passed.
- `swift test`: passed, 351 tests in 29 suites.
- `scripts/benchmark-gate-smoke.sh`: passed, 68 checks.
- `xcrun swift-format lint --strict --recursive Sources Tests`: passed.
- `shellcheck scripts/benchmark-gate-smoke.sh`: passed.
- A8 and A9 narrowing mutants: both killed.
- F1/F2/F3 production attacks: all refused.
- F5 malformed-suite attack: falsely accepted; integer control demonstrates the lost workload.

The clean suite is therefore insufficient for acceptance: it does not attack the malformed suite-input surface used by the production driver.
