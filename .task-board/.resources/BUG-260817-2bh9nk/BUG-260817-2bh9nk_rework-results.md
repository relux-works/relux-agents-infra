# BUG-260817-2bh9nk Rework Results

## Outcome

- `RunSharedRuntimeLauncher` validates its actual inherited environment immediately before `sharedRuntimeExecve`.
- The internal launcher regression drives the process-shaped `runtime runtime-launch` entry with descriptor 3 and caller-minted authorization evidence.
- `TestProductionRuntimeLaunchRefusesModelOriginEnvironment` additionally builds the real `agents-infra` binary and drives `tools/agents-infra/main.go:628` with caller-owned descriptor 3, the child PID, the resolved runtime key, and a valid exec-plan digest.
- The real binary clean control execs the configured target. Exact `HF_ENDPOINT` and `MODEL_ENDPOINT` cases refuse before target exec and never print their values.
- Existing `RunPi`, operator documentation, bootstrap-installed global alias, and project-local wrapper checks remain intact and green.

## Production Call-Site Audit

- `sharedRuntimeExecve` has one production invocation: `tools/agents-infra/internal/infra/pi_shared_launcher_darwin.go`, after the environment validation.
- `RunSharedRuntimeLauncher` is reached from `tools/agents-infra/main.go` for `runtime runtime-launch`.
- The shared broker starts that same CLI entry with `runtime runtime-launch`; it does not own a second `sharedRuntimeExecve` path.
- Internal test code invokes `RunSharedRuntimeLauncher` through its `TestMain` routing shim. The new main-package regression separately invokes the built production binary, so production CLI argument routing is under test rather than assumed.
- Unsupported POSIX and Windows definitions return unsupported errors and never spawn the runtime.

## Current-Run Validation Evidence

| Gate | Exit | Result |
| --- | ---: | --- |
| First production-binary test attempt | 1 | Fixture failed before launch: missing `.agents/.configs` directory |
| Second production-binary test attempt | 1 | Fixture failed before launch: sharing timeout `35s` was below the parser's `37s` minimum |
| Third production-binary test attempt | 1 | Fixture failed before launch: cache root did not exist |
| Fourth production-binary test attempt | 1 | Fixture failed before launch: macOS rendezvous path was 200 bytes |
| Fifth production-binary test attempt | 1 | HF refusal occurred, but the harness read output before `Wait` and required a marker before the next denied case |
| Restored production-binary test | 0 | Clean target exec plus both exact-name refusals pass through the built CLI |
| Internal launcher production subset | 0 | Caller-minted authorization control and both model-origin refusals pass |
| MODEL_ENDPOINT removed from exact gate | 1 | Expected red: production binary test reports `MODEL_ENDPOINT` admitted |
| HF_ENDPOINT removed from exact gate | 1 | Expected red: production binary test reports `HF_ENDPOINT` admitted |
| Pre-mutant source restoration comparison | 0 | `cmp` matches the saved source after each mutant |
| Production-binary baseline after restoration | 0 | Restored exact gate is green uncached |
| Installed global/local Pi launcher plus docs gates | 0 | Both installation surfaces and operator-contract regressions pass |
| RunPi/environment and shared-launcher focused gates | 0 | Clean controls and denial paths pass |
| `go test ./... -count=1` | 0 | Root `200.977s`, attachments `2.955s`, infra `323.826s`, modelharness `2.285s` |
| `go vet ./...` | 0 | Clean |
| `go build ./...` | 0 | Clean |
| `gofmt -d` changed files | 0 | No diff |
| `git diff --check` | 0 | Clean |

Earlier attached evidence also records a serial full-suite exit 0 and focused race-test exit 0. The table above identifies the commands rerun directly in `RUN-260830-8012e8`; those earlier results were not substituted for current execution.

## Files Changed

- `tools/agents-infra/internal/infra/pi_shared_launcher_darwin.go`
- `tools/agents-infra/internal/infra/pi_shared_launcher_test.go`
- `tools/agents-infra/runtime_main_darwin_test.go`
- `LOGBOOK.md`
