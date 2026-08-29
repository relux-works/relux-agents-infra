# TASK-260830-1xscas Review Verdict

## Verdict

Accepted. Change Request `CR-TASK-260830-1xscas-1` revision 1 matches the
validated revision-2 replay, closes both predecessor bypasses, and is suitable
for the orchestrator's checkpoint/integration and canonical PR delivery flow.

## Exact Target, Digest, Scope, And Configuration

- Two fresh `git fetch origin main` runs resolved `origin/main`, local `main`,
  workspace `HEAD`, and CR base to
  `d69a435945758ea1cd5dfa62395ca32498e712c7`.
- The final workspace snapshot tree is
  `ffd719be0ba3e26bb3a1caeeb537dc97c4dc7390`, exactly the immutable CR
  candidate tree.
- The materialized CR patch SHA-256 is
  `388e2cd095f69a613733ab03c91c03a65ed1090b0ae82f3499cf48b0742db3e9`
  and its bytes equal `git diff --binary <base> <candidate>`.
- Exactly four paths change: `.instructions/INSTRUCTIONS_WORKFLOW.md`,
  `README.md`, `LOGBOOK.md`, and
  `tools/agents-infra/internal/infra/infra_test.go`.
- `task-board.config.json` is outside the delta and contains no `fast_mode`
  key. Spawn preflight reports `fast_mode=false`, source `default`.
- `git diff --check` and `gofmt -d` are clean.

## Gate-Defeat Review

The production call site is `infra.Setup(Options{Layout: layout})`, reached by
`TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex` through
`Setup -> RefreshLinks -> setupClaude/setupCodex`. The test owns its expected
Claude entrypoint and complete External-CI policy section independently of the
production constant and installed artifacts.

Reviewer-created disposable candidate copies killed all required negative
shapes uncached:

| Mutant | Result | Protected surface |
| --- | ---: | --- |
| Add contradictory permissive mirror trigger | exit 1 | Exact composed External-CI section |
| Redirect production Claude entrypoint to platform-only instructions | exit 1 | Test-owned consumer entrypoint |
| Remove workflow include from Claude instruction index | exit 1 | Installed Claude index |
| Remove workflow include from Codex instruction index | exit 1 | Rendered Codex instructions |

This closes the predecessor findings `bypass path around the check` and
`forged or self-minted evidence`; no surviving bypass was found.

## Reviewer Validation

- Focused policy tests, uncached: exit 0.
- Full `go test ./... -count=1`: exit 0.
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh`: exit 0.
- `agents-infra verify global`: exit 0.
- Installed Agents/Claude/Codex source, entrypoint, include, rendered-policy,
  and single-heading parity: exit 0.
- Final fresh-base and candidate-tree gate: exit 0.

The review discarded two invalid local probes rather than laundering them into
evidence: forward-applying the patch over the already-applied candidate, and
requiring a literal unrendered Codex index inside the rendered Codex artifact.
Both were replaced with correct byte/tree and consumer-surface oracles.

## Handoff Boundary

Acceptance parks the Task at `to-review`; it does not claim `done`, commit,
push, PR checks, or merge. The commit-owning orchestrator must preserve this
reviewed exact head, perform checkpoint/integration, canonical signed PR
delivery, hosted checks, and landing under the repository policy.

