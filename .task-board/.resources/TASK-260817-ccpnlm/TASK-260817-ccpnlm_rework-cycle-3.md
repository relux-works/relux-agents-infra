# TASK-260817-ccpnlm — developer rework cycle 3

## Outcome

Closed reviewer cycle 2's production output race without weakening the caller-facing writer test.

- `RunPi` now serializes every concurrently live runtime/Pi stdout/stderr write through one shared mutex while preserving each stream's configured destination.
- Runtime child completion is retained as a multi-consumer close latch instead of a consumable result channel.
- Runtime cleanup waits for `Cmd.Wait` to finish and drain output pipes after the process group disappears, before returning to the caller or releasing the profile lock.
- The immediate post-readiness child-exit branch also cleans any surviving runtime descendants.
- `TestPiLaunchSerializesRuntimeOutputFanIn` drives production `RunPi` with an actual runtime writing heavily to stdout and stderr and a non-concurrent `bytes.Buffer`; it asserts both streams survive and passes under the race detector.

## Red-to-green evidence

Every command ran as a standalone process without `tee`.

| Command | Exit | Meaning |
| --- | ---: | --- |
| `go test -race ./internal/infra -run '^TestPiLaunchSerializesRuntimeOutputFanIn$' -count=1` after runtime-only mutex | 1 | Expected development red: exposed concurrent Pi stderr and a return-before-pipe-drain path. |
| Same command after retained child completion but before cross-process serialization | 1 | Expected development red: isolated the remaining runtime/Pi cross-process writer race. |
| `go test -race ./internal/infra -run '^TestPiLaunchSerializesRuntimeOutputFanIn$' -count=1` after the full fix | 0 | Dual-stream production regression green. |
| `go test -race ./internal/infra -run '^TestPiLaunchOwnedRuntimeLifecycleAndGlobalStatePreservation$' -count=1` | 0 | Exact reviewer reproduction green. |

## Final validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run 'Test.*Pi' -count=1` | 0 | `.temp/TASK-260817-ccpnlm/go-test-focused-05.log` |
| `go test -race ./internal/infra -run 'Test.*Pi' -count=1` | 0 | `.temp/TASK-260817-ccpnlm/go-test-race-pi-02.log` |
| `go test ./... -count=1` | 0 | `.temp/TASK-260817-ccpnlm/go-test-full-05.log` |
| `go vet ./...` | 0 | `.temp/TASK-260817-ccpnlm/go-vet-04.log` |
| `go build -o ../../.temp/TASK-260817-ccpnlm/bin/agents-infra .` | 0 | `.temp/TASK-260817-ccpnlm/go-build-04.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `.temp/TASK-260817-ccpnlm/go-build-windows-03.log` |
| `test -z "$(gofmt -l .)"` | 0 | `.temp/TASK-260817-ccpnlm/gofmt-check-02.log` |
| `git diff --check` | 0 | `.temp/TASK-260817-ccpnlm/git-diff-check-06.log` |
| Untracked Pi source/manifest trailing-whitespace scan | 0 | `.temp/TASK-260817-ccpnlm/untracked-whitespace-04.log` |
| `task-board validate` | 0 | `.temp/TASK-260817-ccpnlm/task-board-validate-05.log` |

The full Pi race suite exercised the official 217-record fixture with no skips. The approved malicious-runtime and same-UID bind-race non-claims remain unchanged.
