# TASK-260830-27u51n recovery 03

## Outcome

The external-CI local-mirror policy candidate remains unchanged in behavior and
is ready for review. CR revision 2's unrelated process/readiness failures did
not reproduce after the competing board-cli and agents-infra suites drained.
No production code or timing threshold was changed.

Versioned scope:

- `.instructions/INSTRUCTIONS_WORKFLOW.md` contains the narrow fallback and all
  blocking/status-integrity clauses.
- `tools/agents-infra/internal/infra/infra_test.go` drives the real global setup
  path and proves source-to-installed Claude byte parity plus rendered Codex
  policy inclusion.
- `README.md` documents the operator contract.
- `LOGBOOK.md` records the policy decision and the LLVM 23 setup anomaly.

## Recovery validation run directly in this worker

| Command / evidence | Exit | Result |
| --- | ---: | --- |
| Exact model-check deadline/tool-pids subtest, uncached | 0 | PASS, package 5.794s |
| Exact pinned-Pi direct-RPC side-effect test, uncached | 0 | PASS, package 1.546s |
| Exact readiness 503 timeout subtest, uncached | 0 | PASS, package 3.518s |
| Exact early-runtime-exit readiness subtest, uncached | 0 | PASS, package 1.941s |
| Exact shutdown escalation test, uncached | 0 | PASS, package 2.659s |
| `cd tools/agents-infra && go test ./... -count=1` | 0 | All packages PASS; root 84.738s, infra 168.472s |
| `cd tools/agents-infra && go vet ./...` | 0 | PASS |
| Focused source-to-Claude/Codex setup parity test | 0 | PASS, package 3.487s |
| `cd tools/agents-infra && go build ./...` | 0 | PASS |
| `gofmt -d` emptiness gate | 0 | No formatting delta |
| `git diff --check` | 0 | PASS |

The already attached
`TASK-260830-27u51n_expected-red-mutant-recovery-02.log` remains the negative
evidence for narrowing the trigger: the broadened-policy mutant exited 1 and
the restored focused control exited 0. This worker did not relabel or rerun
that prior evidence.

## Runtime synchronization

The first direct `./setup.sh` invocation exited 1 before global sync after
Homebrew auto-upgraded LLVM 22 to 23 and the expected
`/opt/homebrew/opt/llvm/bin/lldb-mcp` helper was absent. Homebrew documents LLD
and LLDB as separate formulae in that release. The failure is preserved as
evidence and was not presented as passing.

The canonical bootstrap was then rerun with its documented
`AGENTS_INFRA_SKIP_LLDB_MCP=1` switch because this task does not require the
optional LLDB MCP bootstrap. That invocation exited 0, built and installed the
current source CLI, and ran global setup. `agents-infra verify global` exited 0.
Installed Agents and Claude workflow files compare byte-for-byte with the
versioned source, and the generated Codex `AGENTS.md` contains the fallback,
exact PR-head, repository-failure, non-forgery, review, and protection clauses.

## Evidence notes

CR revision 2's validation log remains attached and records its real exit 1.
Two quiet-window diagnostic attempts in this recovery exited 75 before any Go
suite started because the initial process scanner matched its own shell argv;
the scanner was corrected to inspect the executable/argv fields. These were
preflight refusals, not test results.

Candidate base observed in the Story worktree: `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`.
The worktree was 0 commits behind local `main` at recovery start.
