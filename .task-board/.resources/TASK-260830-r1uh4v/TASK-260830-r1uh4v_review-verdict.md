# TASK-260830-r1uh4v Review Verdict

## Verdict

Changes requested. CR `CR-TASK-260830-r1uh4v-1` revision 1 must not be accepted.
Two production-composition bypass mutants survive the exact candidate's focused
tests with exit 0.

## Exact Target And Scope

- Fresh `git fetch origin main` during review resolved `origin/main` to
  `fe3818209c9861fcafa1f2e68efe078cc0f96f30`.
- Workspace and Story branch `HEAD` equal that OID.
- The workspace tree resolved through `git stash create` is
  `4f24569fe94add1a31ce62e1dfce619fa46d0815`, exactly the CR candidate tree.
- The exact base-to-candidate delta changes only the four declared paths:
  `.instructions/INSTRUCTIONS_WORKFLOW.md`, `LOGBOOK.md`, `README.md`, and
  `tools/agents-infra/internal/infra/infra_test.go`.
- `git diff --check` exits 0. `task-board.config.json` is outside the delta and
  still contains the landed Codex-only `"fast_mode": true` configuration.

## Findings

### F1 — Additive broadened-trigger bypass remains green

Severity: blocking acceptance.

Negative shape: bypass path around the check; clause-presence is not proof of
the exclusive authorization bound.

The tests at `tools/agents-infra/internal/infra/infra_test.go:25-34` accept any
installed body containing the required phrases. In a disposable copy of the
exact candidate, the reviewer preserved the exact exclusive trigger and added
an immediately following clause authorizing a local mirror for any repairable
or merely inconvenient hosted-CI disruption. Production `infra.Setup` published
the mutated workflow, while both
`TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex` and
`TestSetupGlobalRejectsBroadenedRepairableOrMerelyInconvenientExternalCIMirrorTrigger`
passed uncached with exit 0.

Evidence:

- `.temp/reviewer/mutant-shapes-02.log`
- `.temp/reviewer/mutant-additive-broadened-02.log`
- `.temp/reviewer/mutant-exits-02.log`

Required rework: make the production-composition assertion independently pin
the complete External-CI policy section, including its authorization boundary,
so an additive contradictory trigger fails. Re-run this exact additive mutant
uncached and require exit 1.

### F2 — Claude entrypoint bypass uses a self-minted oracle

Severity: blocking acceptance.

Negative shapes: forged or self-minted evidence; bypass path around the check.

Production `infra.Setup` writes the Claude entrypoint from
`generatedClaudeEntrypoint` (`infra.go:92`). The test compares the installed
entrypoint to that same production constant (`infra_test.go:727-734`) and then
checks the workflow include in an index that the mutated entrypoint no longer
loads. In a disposable exact-candidate copy, changing the production entrypoint
from `@instructions/INSTRUCTIONS.md` to
`@instructions/INSTRUCTIONS_PLATFORM.md` made Claude bypass the workflow while
`TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex` still
passed uncached with exit 0.

Evidence:

- `.temp/reviewer/mutant-shapes-02.log`
- `.temp/reviewer/mutant-claude-entrypoint-bypass-02.log`
- `.temp/reviewer/mutant-exits-02.log`

Required rework: assert the Claude consumer path against an independent test
expectation, not `generatedClaudeEntrypoint` itself, and make this production
constant mutant fail uncached with exit 1.

## Validation And Evidence Audit

Reviewer-rerun:

- Pristine focused production tests: exit 0
  (`.temp/reviewer/pristine-focused-01.log`).
- Additive broadened-trigger mutant: exit 0, unexpected survivor.
- Claude production-entrypoint bypass mutant: exit 0, unexpected survivor.
- Fresh base/HEAD/tree/scope and diff check: recorded in
  `.temp/reviewer/exact-target-01.log`.

Producer evidence accepted as executed, not rerun by the reviewer:

- CR-bounded full `go test ./... -count=1` and `go vet ./...` exit 0.
- Producer full test, vet, build, canonical skip-LLDB setup, global verify, and
  installed parity logs are present. Every file in the attached evidence
  archive matches its manifest (`.temp/reviewer/producer-archive-sha-audit-02.log`).
- The producer's replacement-style broadened trigger and source-index
  Claude/Codex include-removal mutants are genuinely red, but they do not cover
  the two surviving bypasses above.

The `logbook` skill requires a repository `LOGBOOK.md` write. This reviewer run
is read-only by role, so it did not alter the candidate; the regression is
persisted in this board-owned verdict artifact for the next producer.
