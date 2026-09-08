# TASK-260829-2t5xmi — Developer Handoff

## Outcome

- Added a mode-0600, runtime-key-scoped `restart-ledger.json` through `ResolveSharedRuntimePaths`.
- Added required operator configuration: `restart_limit`, `restart_initial_backoff_seconds`, `restart_max_backoff_seconds`, `stable_run_seconds`, and `quarantine_seconds`. No numeric production defaults were introduced.
- `RunSharedRuntimeBroker` records failed attempts and exact readiness matches; `acquireSharedRuntimeLease` automatically starts successor brokers after ledger-bound exponential delays and preserves typed quarantine exit 77.
- Restart backoff is capped, stable readiness resets the count, reaching the limit quarantines, expiry admits one half-open attempt, and a failed half-open re-quarantines.
- Added `runtime quarantine` / `runtime unquarantine`; mutations refuse while an active broker owns the election lock.
- `SharedRuntimeStatus` JSON always carries `restart_count`, `quarantined_until`, and `last_readiness_match` (plus `manual_quarantine`).
- Added real production-seam coverage for abrupt AF_UNIX client death in `sharedBrokerServer.handleConnection` and abrupt Pi subprocess death through `RunPi -> runSharedPiSession`.
- Fixtures remain local/static subprocess fixtures; no live-model call was used.

## Validation Evidence

All commands below ran in the assigned Story worktree module.

| Command | Exit | Result |
| --- | ---: | --- |
| Relevant production/CLI suite | 0 | infra 22.095s; root 0.738s |
| Relevant race suite (pre-final wiring) | 0 | 46.504s |
| Final relevant race suite | 0 | 60.601s |
| `go test ./internal/infra -count=1` | 0 | 177.008s |
| Root + remaining package tests | 0 | root 87.402s; attachments 2.468s; modelharness 1.402s; model-harness has no tests |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | Darwin build clean |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | cross-build clean |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | cross-build clean |
| `gofmt -l tools/agents-infra` | 0 | no output |
| `git diff --check` | 0 | clean |

## Negative Evidence

- Narrowed the production backoff cap from `delay > maximum` to `delay >= maximum*2`; `TestSharedRuntimeRestartPolicyIsBoundedStableAndHalfOpen` exited 1 because the third delay widened to 8s instead of the configured 5s. Exact source copy restored; named test returned exit 0.
- Bypassed the active broker-lock refusal in `SetSharedRuntimeManualQuarantine`; `TestSetSharedRuntimeManualQuarantineRefusesActiveBrokerBeforeLedgerMutation` exited 1 because the mutation was admitted. Exact source copy restored; named test returned exit 0.
- `TestSharedRuntimeStatusRefusesMalformedRestartLedger` proves a malformed read is refused as `shared_runtime_state_unreadable`, not treated as absent/zero.
- `TestRunSharedRuntimeBrokerRefusesPersistedManualQuarantineBeforeRuntimeLaunch` and `TestAcquireSharedRuntimeLeaseSurfacesPersistedManualQuarantine` drive the real broker/acquisition entry points.

## Evidence Anomaly

An early baseline was invalid: it ran in the primary checkout and overlapped another suite after a lost session handle. Its exit 1 is recorded in `LOGBOOK.md` and is not cited as candidate validation. Every result above is a later directly tracked run in this Story worktree.
