# TASK-260817-ccpnlm reviewer verdict — changes requested

Verdict branch: `changes_requested -> to-dev`.

## Finding

1. **P1 — Production runtime output fan-in races on the caller writer.** `RunPi` assigns both `runtimeCmd.Stdout` and `runtimeCmd.Stderr` to the same arbitrary `opts.Stderr` (`tools/agents-infra/internal/infra/pi_launch_posix.go:142-143`). `os/exec` copies the two pipes concurrently. The existing production-entry lifecycle test passes a `bytes.Buffer` at `pi_test.go:504-505`; the race detector reports concurrent `bytes.Buffer.ReadFrom` writes and fails the test. This can corrupt captured runtime diagnostics and makes the lifecycle API unsafe for ordinary non-concurrent writers. Minimal reproduction:

   `go test -race ./internal/infra -run '^TestPiLaunchOwnedRuntimeLifecycleAndGlobalStatePreservation$' -count=1`

   Result: exit 1, `WARNING: DATA RACE`, test failed with `race detected during execution of test`.

Required rework: serialize the runtime stdout/stderr fan-in (or otherwise provide concurrency-safe routing) at the production `RunPi` call site, add a fixture that emits both stdout and stderr, and make the minimal race command pass. Do not weaken the test by replacing the buffer with `os.File` or by suppressing race detection.

## Other validation

- `go test ./internal/infra -run 'Test.*Pi' -count=1 -v` — pass, no skips.
- `go test ./... -count=1` — pass.
- `go vet ./...` — pass.
- `go build ./...` — pass.
- `gofmt -l internal/infra/pi_*.go main.go main_test.go` — clean.
- `git diff --check` — pass.
- `task-board validate` — pass.
- `go test -race ./internal/infra -run 'Test.*Pi' -count=1` — fail on the same runtime output race.

Cycle-1 bare-unknown argv bypass is fixed and its production negative/narrowing controls pass. The remaining race prevents acceptance.
