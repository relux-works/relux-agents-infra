# TASK-260817-ccpnlm cycle-6 developer outcome

## Scope

Resolved reviewer cycle-5 signal lifecycle flakiness without weakening the production `RunPi` path or the accepted managed-session isolation gates.

## Changes

- Replaced the external Python Pi signal fixture with the race-instrumented Go test binary.
- The child installs `signal.Notify` before writing its readiness marker, so SIGINT/SIGTERM delivery cannot race fixture initialization.
- Kept the production entry point under test: `RunPi` still launches the child in its own process group, forwards the signal, waits for graceful exit, cleans the runtime group, and releases the profile lock.
- Increased startup budgets to `10s` only for Python fixtures whose assertions are not timeout behavior; production timeout handling and the dedicated timeout-escalation test are unchanged.
- Recorded the root cause and evidence in `LOGBOOK.md`.

## Negative and stability evidence

- Baseline `go test -race ./internal/infra -run 'Test.*Pi' -count=1`: exit 1; reproduced `TestPiLaunchForwardsSignalsThenCleansRuntime/terminated` with `signal: killed` plus Python readiness timeouts.
- `go test -race ./internal/infra -run '^TestPiLaunchForwardsSignalsThenCleansRuntime$' -count=20`: exit 0; 20/20 signal lifecycle repetitions passed.
- `go test -race ./internal/infra -run 'Test.*Pi' -count=1`: exit 0 in three independent consecutive processes (18.776s, 17.685s, 17.829s).
- The signal regression covers both SIGINT and SIGTERM, asserts the exact received signal, then asserts runtime-group cleanup and profile-lock release.
- Existing `TestPiLaunchShutdownTimeoutKillsRuntimeGroupAndReleasesLock` retains timeout-to-SIGKILL escalation evidence.

## Required validation

| Command | Exit | Result |
| --- | ---: | --- |
| `go test -race . -run 'TestRunPi' -count=1` | 0 | pass |
| `go test ./... -count=1` | 0 | pass |
| `go vet ./...` | 0 | pass |
| `go build ./...` | 0 | pass |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | pass |
| `git diff --check` | 0 | pass |
| `task-board validate` | 0 | pass |

Logs are under `.temp/TASK-260817-ccpnlm/`.
