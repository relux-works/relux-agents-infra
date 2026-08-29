# TASK-260828-3fgca3 review verdict — round 4 / CR revision 5

## Verdict

**Accepted.** Accept `CR-TASK-260828-3fgca3-5` revision `5` and park
`TASK-260828-3fgca3` at `to-review` for the orchestrator's checkpoint/integration.

Revision 5 closes F6 at the production entry, preserves the previously reviewed
F1–F5 and G1–G4 gates, does not relax `unpinnableConditions`, and states the
tool-schema boundary narrowly enough to match the implementation. I could not
construct a further production-entry bypass.

Reviewed candidate:

- Base OID: `5a287e4bcb454b53bc432cb7788a03792d1c96f6`
- Candidate tree OID: `bf16f38acfefea46c8292e8aca9ee013d19168c5`
- Patch SHA-256: `668b92a77ea3eeb00c06b54c7f909091934f95e3d04e6914016a607b0f426cfb`
- All 21 changed-path working bytes were re-hashed and matched the candidate tree.

## F6 production-entry attack and controls

I ran the shipped Release `mlx-swift-runtime-prototype benchmark-run` through
`scripts/benchmark-gate-smoke.sh` on unused ports `19871–19895`.

- `parameters.required` changed to `parameters.require`: exit `5`; the refusal
  names `scenarios.tool_call.tools[0].function.parameters.required`; no decision
  was emitted; no session directory was created.
- `required` absent: exit `5`; no decision; no session directory.
- non-`function` tool type: exit `5`; no decision; no session directory.
- explicit `required: []`: complete pass, exit `0`, `accepted: true`; both sealed
  records contain `tool_call.succeeded == true`.
- shipped non-empty control (`required: ["vehicle"]`): the ordinary control pass
  reached `accepted: true`; `examples/benchmark-prompts.json` passed prompt-suite
  validation and stopped only on the deliberately unreadable thresholds control.

The complete production smoke finished with **95 checks, 0 failures, exit 0**.
It also re-ran the F1–F5/G1–G4 attacks and controls: caller-minted equivalence,
missing/unreadable equivalence, malformed and mismatched `n_ctx`, `-ub` pinning,
unhonoured context bound, failed `/slots`, MTP-on, silent `/slots`, inherited
speculation, malformed F5 suite shapes, trusted-equivalence routing, and the
same-bound llama.cpp-shaped positive control.

The smoke left no listeners or fake-runtime processes behind. No real model was
loaded.

## Claim audit

The implemented invariant is bounded and matches the revised claim:

- unknown keys are refused at the document object and each kind-scoped scenario
  object;
- inside a tool declaration, only `type`, `function`, `function.name`,
  `function.parameters`, and `function.parameters.required` are validated;
- `required` is mandatory and `[]` is the explicit no-mandatory-arguments form;
- the remaining tool/JSON-Schema subtree is deliberately opaque to the gate and
  is carried as the same `JSONValue` tree into the request, then serialized with
  sorted keys. “Verbatim” is therefore value-preserving, not a claim that source
  whitespace, key order, or lexical JSON bytes are forwarded;
- `PromptSuiteSchema.validatedToolFields`, `unvalidatedByDesign`, and the exact
  five `supportedAbsences` are pinned whole by tests.

The README, research report, test-suite header, and LOGBOOK correction now name
that boundary and the residual: a misspelling in an opaque parameter keyword
that the benchmark does not read can reach the runtime unremarked. No universal
JSON-Schema validation is claimed.

## No-relaxation and mutant evidence

Revision 5 changes only the F6 schema/tests/smoke and their documentation.
Compared with reviewed revision 4, these admission-bearing files are
byte-identical:

- `RuntimeBenchmark.swift`: blob `509304cf4a1b5ad1899011f612de7886319e00ab`
- `RuntimeAttestation.swift`: blob `8214b9d80824d6c94bfdeefd4553fd126191fc77`
- `BenchmarkRunPins.swift`: blob `3994fb241376164debebedcafd366b9cdadc36b1`

`RuntimeBenchmark.unpinnableConditions` remains exactly
`["kv=unread", "prefill-step=unpinned", "reasoning=unpinned"]`, and the whole
list is asserted by `doesNotRelaxTheUnpinnableConditions`.

As a read-only reviewer I did not mutate candidate source. I inspected the
producer's new task-scoped revision-5 mutant report and the named tests, and
accepted its tree-scoped results: M-F6-1 reproduces the typo false acceptance;
M-F6-2 and M-F6-3 narrow the admitted class and are killed; the re-applied
M-F5-1/2/3, M11/M12, and A8/A9 are also killed. My independent production smoke
reproduced the fixed side of each relevant gate on the exact candidate tree.

## Reviewer validation

- `swift build --build-tests`: exit `0`.
- `swift build -c release`: exit `0`.
- `swift test -c release`: **385 tests in 30 suites**, exit `0`.
- `scripts/benchmark-gate-smoke.sh`: **95 checks, 0 failures**, exit `0`.
- `xcrun swift-format lint --strict --recursive Sources Tests`: exit `0`.
- `shellcheck scripts/benchmark-gate-smoke.sh`: exit `0`.
- exact CR `git diff --check`: exit `0`.

One exploratory non-canonical combined lane,
`swift build -c release --build-tests`, exited `1` because SwiftPM compiled the
library without testing enabled (`module ... was not compiled for testing`).
The project's canonical split build/test lanes above all passed; this probe is
not a candidate failure and is recorded here rather than hidden.

