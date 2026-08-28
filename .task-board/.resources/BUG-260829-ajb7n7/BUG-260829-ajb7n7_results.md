# BUG-260829-ajb7n7 developer evidence

## Base and scope
- Story worktree HEAD, main, and refreshed origin/main all resolved to 675f77ed63376320ed1213f46f9462a299c0abaf before edits.
- Changed tools/agents-infra/main_test.go and added the required LOGBOOK.md entry.
- No live Pi or local-model runtime, service, socket, or endpoint was started, probed, or contacted.

## Expected-red reproduction on prior helpers
- go test -run ^TestCaptureStdoutDrainsLargeOutputConcurrently$ -count=1 -timeout=2s . => exit 1 (expected): callback blocked in os.File.Write at 3 MiB + 17 bytes while captureStdout had not reached io.Copy.
- go test -run ^TestCaptureStderrDrainsLargeOutputConcurrently$ -count=1 -timeout=2s . => exit 1 (expected): identical blocked writer/read-start ordering for captureStderr.

## Implementation
- Both wrappers delegate to one capturePipe helper.
- The io.Copy goroutine starts before the producer.
- Idempotent cleanup restores the global descriptor first, closes the writer, waits for the drain result, then closes the reader; panic unwinding uses the same cleanup.
- Tests assert byte identity for separate 3 MiB + 17 byte stdout/stderr payloads plus normal-path and panic-path descriptor restoration.

## Validation
- Focused capture regressions: exit 0, 0.446s.
- Focused capture regressions with -race: exit 0, 3.226s.
- Exact root status test TestRunRuntimeStatusJSONIsAbsentAndSideEffectFree: exit 0, 0.378s.
- Complete root package go test -count=1 .: exit 0, 80.853s.
- go vet .: exit 0.
- go build .: exit 0; generated untracked binary removed afterward.
- git diff --check: exit 0.

Negative gate testing is not applicable: this change is test-only output capture and does not gate, refuse, validate, authorize, or attest production behavior.