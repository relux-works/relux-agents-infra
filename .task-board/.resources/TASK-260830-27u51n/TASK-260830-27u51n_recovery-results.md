# TASK-260830-27u51n recovery results

## Outcome

Recovered the producer handoff after Change Request revision 1 failed its first configured validation command. The repository delta remains scoped to four versioned paths:

- `.instructions/INSTRUCTIONS_WORKFLOW.md` — narrow external-CI local-mirror fallback policy.
- `tools/agents-infra/internal/infra/infra_test.go` — source-to-installed Claude and rendered Codex parity test through the production global `Setup` entry point.
- `README.md` — operator-facing policy summary.
- `LOGBOOK.md` — durable decision and setup-bootstrap finding.

## Acceptance coverage

- Fallback requires verified hosted-provider evidence of an external cause that prevents repository steps from executing and that the agent cannot repair; absent, failed, partial, malformed, or inconclusive status reads do not authorize it.
- Every affected job must be reproduced from the exact clean PR head with matching commands, toolchain, environment/non-secret configuration, services, and target, or a documented equivalent; otherwise it is unverified.
- Evidence requirements include PR-head SHA, hosted job, external cause, platform/target, tool versions, environment assumptions, services, exact commands, and every exit code.
- Repository-caused failures remain blocking. Local evidence cannot forge or replace hosted status, self-review, remote review, merge queues, or branch protection.
- Repository bootstrap refreshed the installed runtime; Agents and Claude workflow bytes equal source, and rendered Codex instructions contain the complete source and required clauses.

## Validation run by this recovery producer

| Command | Exit | Result |
| --- | ---: | --- |
| `go test . -run 'TestModelCheckProductionEntrypoint/deadline_override_terminates_both_owned_process_groups' -count=1` | 0 | Prior CR failure did not reproduce. |
| `go test ./internal/infra -run TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex -count=1` | 0 | Focused production setup/parity test passed before mutation. |
| `go test ./... -count=1` | 0 | All Go packages passed uncached. |
| `go vet ./...` | 0 | Configured second CR gate passed. |
| `go build ./...` | 0 | Full Go build passed. |
| Broadened-trigger mutant focused test | 1 (expected red) | Removing `verified` from the external-cause precondition was rejected by the named test. |
| Focused test after restoring source | 0 | Restored policy passed. |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | Built/installed the current source CLI and refreshed global runtime through the repository bootstrap. |
| `agents-infra verify global` | 0 | Current installed global runtime verified. |
| Source-to-Agents `cmp` | 0 | Installed workflow bytes equal source. |
| Source-to-Claude `cmp` | 0 | Claude instruction surface equals source. |
| Codex required-clause check | 0 | Rendered Codex surface contains all blocking clauses. |
| `gofmt -d internal/infra/infra_test.go` | 0 | No output. |
| `git diff --check` | 0 | No whitespace errors after final README wording adjustment. |

## Recovery anomaly

The pre-bootstrap installed CLI (`v1.6.1-44-gd91d6fc`) returned exit 0 from `setup global` and `verify global` while `~/.agents/.instructions` was absent, so two guessed-path comparisons returned exit 2. Running the repository `./setup.sh` built the current worktree CLI (`v1.6.1-84-g5c9b4e4`), refreshed the supported layout, and made every parity check pass. No runtime file was edited directly.

The earlier producer's attached expected-red artifact was inspected, but this recovery producer also reran an independent broadened-policy mutant and post-restore green control. The original CR failure was preserved as failing evidence; it was not reported as passing.
