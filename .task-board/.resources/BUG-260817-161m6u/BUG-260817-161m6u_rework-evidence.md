# BUG-260817-161m6u — Cycle-1 Rework Evidence

## Source Delta

- Added `` `LLAMA_ARG_*` environment names are refused before llama.cpp starts `` to the exact fragments asserted by `TestPiOperatorContractDocumentsCycle10Boundary`.
- No production gate change was needed; reviewer mutation evidence already established that `RunPi` calls `ValidatePiExecutionEnvironment` before managed state creation and llama.cpp spawn.

## Negative Evidence

| Gate | Exit | Result |
| --- | ---: | --- |
| Focused source tests, first assertion spelling | 1 | Honest failure: assertion omitted README Markdown backticks; corrected before proceeding. |
| Focused environment, production-entry, and docs tests | 0 | Gate and exact clean-environment control green. |
| README deny-contract deletion mutant | 1 (expected red) | `TestPiOperatorContractDocumentsCycle10Boundary` failed on the missing exact fragment. |
| Restored README docs test | 0 | Exact contract restored and green. |

## Source Validation

| Command | Exit |
| --- | ---: |
| `go test ./... -count=1` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` (empty output required) | 0 |
| `go build -trimpath` | 0 |
| `git diff --check` | 0 |

## Installed Runtime Refresh and Proof

| Check | Exit | Evidence |
| --- | ---: | --- |
| `./setup.sh` bootstrap | 0 | Rebuilt and installed `~/.local/bin/agents-infra`; SHA-256 changed from `df62cd...f42c3` to `3cd24e...a0d`. |
| Installed `agents-infra verify global` | 0 | Global runtime tree verified after bootstrap. |
| Installed local setup from refreshed `~/.agents` | 0 | Disposable task-local runtime refreshed. |
| Installed `agents-infra verify local` | 0 | Disposable local runtime verified. |
| Installed `pi-infra` with inherited `LLAMA_ARG_MODEL` | 1 (expected refusal) | Refused before state creation; stderr names `LLAMA_ARG_MODEL`, omits the probe value, and the isolated HOME remains empty. |
| Installed binary string parity | 0 | Binary contains `LLAMA_ARG_`, `runtime-affecting`, and `pi_execution_environment_invalid`. |

The installed CLI's human error renderer emits the safe message without the symbolic `PiLaunchError.Code`; therefore the behavioral proof establishes refusal, redaction, and no-state ordering, while binary parity establishes that the installed executable contains the named error-code path. No claim is inferred from `verify global` about executable freshness.

## Expected/Recovered Failures

- `agents-infra setup local ... --source-dir <repo>` exited 1 because the disposable destination is nested under that source; self-sync is correctly refused. Re-running with `AGENTS_INFRA_SOURCE_DIR=~/.agents` exited 0.
- The initial installed-probe harness exited 1 only because it incorrectly required the symbolic code in human stderr. The corrected harness checks the observable CLI contract and exited 0 while preserving the actual Pi process refusal exit 1.
