# BUG-260817-161m6u — Developer Outcome

## Implementation

- `ValidatePiExecutionEnvironment` now denies the case-insensitive `LLAMA_ARG_*` namespace alongside the existing `DYLD_*`, `LD_*`, `NODE_*`, and `BUN_*` gates.
- Environment refusals quote the variable name and never include its value.
- Production call site: `RunPi` in `tools/agents-infra/internal/infra/pi_launch_posix.go` invokes the validator before managed state creation and before `runtimeCmd.Start()` spawns llama.cpp; it repeats validation immediately before Pi spawn.
- README documents the managed Pi environment deny contract.

## Tests

- `LLAMA_ARG_MODEL` refusal with exact name-only diagnostic.
- `LLAMA_ARG_CTX_SIZE` refusal as a second absent-option variable.
- Exact clean-environment control.
- Existing duplicate, `DYLD_*`, `LD_*`, `NODE_*`, `BUN_*`, and managed `PI_*` refusal cases retained.
- Production `RunPi` negatives prove denied inherited variables create no managed state and expose no values.

## Validation Evidence

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Focused baseline before change | 0 | `source-focused-baseline-01.log` |
| Focused environment and production tests | 0 | `source-focused-tests-01.log` |
| Narrowing mutant (`LLAMA_ARG_` -> `LLAMA_ARG_M`) | 1 (expected red) | `negative-narrowing-mutant-01.log`; `LLAMA_ARG_CTX_SIZE` was admitted and production `RunPi` reached runtime spawn/readiness |
| Restored focused negative tests | 0 | `negative-narrowing-restored-green-01.log` |
| Full source `go test ./... -count=1` | 0 | `source-go-test-01.log` |
| Source `go vet ./...` | 0 | `source-go-vet-01.log` |
| Source `go build ./...` | 0 | `source-go-build-01.log` |
| Global runtime-tree sync from source (refreshes `~/.agents`, not the bootstrap-owned executable) | 0 | `setup-global-01.log` |
| Installed global runtime-tree verification (does not establish executable freshness) | 0 | `verify-global-01.log` |
| Installed local setup in task-scoped disposable project | 0 | `setup-local-installed-02.log` |
| Installed local verification | 0 | `verify-local-installed-01.log` |
| `gofmt` check | 0 | `gofmt-check-01.log` |
| `git diff --check` | 0 | `git-diff-check-01.log` |

## Cycle-1 Rework

- Added an exact README fragment assertion to `TestPiOperatorContractDocumentsCycle10Boundary`.
- Deleting the documented `LLAMA_ARG_*` deny sentence makes that test fail with exit 1; restoring it returns exit 0.
- Ran repository bootstrap `./setup.sh`, which rebuilt `~/.local/bin/agents-infra`; its SHA-256 changed from `df62cd52b15a3b3f45d6b3b14b0d1b9061e68e9787b7968834a432bf8e6f42c3` to `3cd24eab3fc674d31310fa1cf59a1e723ef61f718c4b466dec742f747eb36a0d`.
- The installed `~/.local/bin/pi-infra` behavioral probe exited 1 for inherited `LLAMA_ARG_MODEL`, named only the variable, omitted the probe value, and created no managed state. The human CLI error intentionally renders the safe message rather than the symbolic code; installed-binary string parity separately confirms `pi_execution_environment_invalid`, `LLAMA_ARG_`, and `runtime-affecting` are present.
- Installed global verification and refreshed installed-local setup/verification all exited 0.

## Notes

- The checkout contained pre-existing story-wide uncommitted/untracked Pi work. The change was applied narrowly in-place because the target Pi files do not exist in `HEAD`; no unrelated changes were reset, staged, or committed.
- Security root cause and gate evidence are recorded in `LOGBOOK.md` at `2026-08-17 2253`.
