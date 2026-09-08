# TASK-260707-1gnk6p Rework Cycle 3 Evidence

## Outcome

The managed LLDB MCP wrapper idempotency gate no longer accepts line-presence
as delegation evidence. `scripts/setup.sh` now renders one canonical wrapper,
compares the installed regular file byte-for-byte with that renderer, and uses
the same renderer when installing or repairing the wrapper.

`TestBootstrapSetupRepairsManagedLLDBMCPWrapperWithUnreachableExpectedTarget`
drives the production root `./setup.sh` entry point with the reviewer's forged
wrapper:

```sh
#!/bin/sh
# agents-infra managed lldb-mcp wrapper.
exit 0
exec "/expected/homebrew/opt/llvm/bin/lldb-mcp" "$@"
```

Before the production fix, the named test exited 1 because setup preserved the
forged inode. After the fix, it exits 0 because setup rejects and replaces that
wrapper. The existing two-run test remains the positive/idempotency bound: one
`brew install llvm`, no second-run wrapper replacement, and no second-run error.

## Validation

Evidence was gathered at Story head
`cf21665dde35274cc14e66e26a93574e0c18c15c`, 63 commits behind local `main`.
All commands ran directly as standalone processes without `tee` or status-
hiding pipelines.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test . -run '^TestBootstrapSetupRepairsManagedLLDBMCPWrapperWithUnreachableExpectedTarget$' -count=1` before fix | 1 | Expected red: `./setup.sh accepted an unreachable expected helper exec as delegation evidence` |
| Same named test after fix | 0 | Forged production-entry wrapper repaired |
| `go test . -run '^TestBootstrapSetup' -count=1` | 0 | Five root setup tests pass, including refusal, external helper, two-run idempotency, wrong target, and unreachable exec |
| `go test ./... -count=1` | 0 | Full uncached Go suite passes |
| `go vet ./...` | 0 | Clean |
| `go build ./...` | 0 | Build succeeds |
| `gofmt -d bootstrap_lldb_mcp_test.go` | 0 | No output |
| `zsh -n scripts/setup.sh` | 0 | Shell syntax valid |
| `git diff --check` | 0 | No whitespace errors |

## Files

- `scripts/setup.sh`: canonical renderer, exact-byte health check, shared write path.
- `tools/agents-infra/bootstrap_lldb_mcp_test.go`: reviewer-shaped production-entry negative.
- `README.md` and `SKILL.md`: exact managed-content idempotency contract.
- `LOGBOOK.md`: forged-delegation regression, fix, and proof.

No live Homebrew installation or user runtime mutation was needed for this
rework; the production setup entry point ran against task-scoped fake Homebrew,
LLVM, HOME, bin, and config directories.
