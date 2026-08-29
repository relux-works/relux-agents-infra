# Standalone Pi YOLO implementation results

## Delivered

- Added the board-agnostic `qwen-infra spawn --prompt TEXT` and direct `agents-infra pi spawn` entry points.
- Added an explicit `[agents.pi.standalone_session]` policy requiring `yolo_mode = true` and an exact validated built-in tool allowlist.
- The wrapper owns `--no-approve`, `--no-extensions`, `--tools`, `--mode json`, `--no-session`, and `--print`; caller Pi arguments are refused before executable lookup.
- Standalone stdin is closed, prompt diagnostics are redacted, failures are typed and sanitized, and configured reasoning remains `medium`.
- Each worker gets an independent Pi process group and random hash-contained client state while shared profiles reuse one verified runtime through existing leases.
- Interactive Pi behavior is unchanged. No task-board adapter, root/sudo path, raw RPC mode, or extension discovery was added.
- README, SKILL.md, and LOGBOOK.md document the standalone-now / board-adapter-later boundary.

## Security and lifecycle evidence

- Real pinned Pi no-model probes prove `--no-extensions` blocks project and global replacement extensions.
- A real pinned Pi control proves direct RPC `bash` bypasses the model `tool_call` hook; production standalone argv excludes RPC and pins JSON mode.
- Negative tests refuse missing/false authorization, empty/duplicate/unknown/wildcard allowlists, malformed config, caller authorization/extension/RPC flags, and state mutation before authorization.
- Concurrent lifecycle coverage proves two independent Pi processes and state roots lease one runtime; crashing one worker releases only its lease and the final release reaps the runtime.

## Original validation

- `gofmt` on every changed Go source: PASS
- `git diff --check`: PASS
- `go test ./internal/infra .`: PASS (`159.992s`, `103.036s`)
- `go test ./...`: PASS
- `go vet ./...`: PASS
- `go build ./...`: PASS
- `GOOS=linux GOARCH=amd64 go build ./...`: PASS
- `GOOS=windows GOARCH=amd64 go build ./...`: PASS

The first repository-root Go test and the first cross-platform `go test` commands were harness mistakes and are recorded under `.temp/TASK-260826-3i0lwe/`; corrected commands passed.

## Fresh handoff audit — 2026-08-26

The recovered production diff, documentation, and every new standalone test were inspected. No concrete defect was found and no delivered-tree change was made during this bounded audit.

- `git diff --check`: exit `0`
- `go test ./internal/infra -run 'Test(BuildStandalonePiArgumentsOwnsExactAuthorizationAndMediumReasoning|PinnedPiNoModelManagedFlagsDisableProjectAndGlobalReplacementExtensions|PinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC|RunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer)$' -count=1`: exit `0` (`4.565s` package time)
- `go test . -run 'Test(RunTargetQwenStandalonePrintConfigOwnsAuthorizationAndPreservesReasoning|StandaloneCLIRefusesCallerOverridesWithSanitizedTypedFailure)$' -count=1`: exit `0` (`0.687s` package time)
- `go build ./...`: exit `0`

The production entry points covered by the fresh gates are `RunPi`, `runTarget -> runPiStandaloneCLI -> BuildPiStandaloneLaunchPlan`, and `runSharedPiSession`.

## Revision 2 — reviewer F1 and bounded gaps

- Both standalone production launch paths now receive a readable non-empty `RunPiOptions.Stdin` witness. The exclusive and shared child helpers still observe EOF, proving the existing guards close stdin rather than relying on `os/exec`'s nil-stdin default.
- Narrowing the exclusive guard in `pi_launch_posix.go` made `TestRunPiStandaloneExclusiveWorkerClosesReadableStdin` fail with exit `1` and `StdinEOF:false`; exact restoration returned exit `0`.
- Narrowing the shared guard in `pi_shared_client_darwin.go` made `TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer` fail with exit `1` and `StdinEOF:false`; exact restoration returned exit `0`.
- CLI tests now admit both exact `(0, 30m]` endpoints (`1ns`, `30m`) and refuse `-1ns`, `0`, and `30m1ns` with the sanitized `pi_standalone_deadline_invalid` public envelope.
- The README standalone snippet now includes the required managed Pi `primary_session.profile` and `primary_session.pi_compatibility` prerequisite and points profile/target details to the surrounding operator contract.
- Redundant fail-closed entrypoint and environment gates remain unchanged.

### Revision-2 validation

- Focused exclusive/shared stdin baseline: exit `0` (`3.738s` package time).
- Targeted CLI bounds, refusal, and sanitization: exit `0` (`0.630s` package time).
- Exclusive stdin narrowing mutant: expected exit `1`, named test observed `StdinEOF:false`; restored control exit `0` (`1.333s`).
- Shared stdin narrowing mutant: expected exit `1`, named test observed `StdinEOF:false`; restored control exit `0` (`1.993s`).
- `go test ./... -count=1`: exit `0` (main `80.514s`, attachments `1.291s`, infra `129.473s`).
- `go vet ./...`: exit `0`.
- `go build ./...`: exit `0`.
- `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...`: exit `0`.
- `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...`: exit `0`.
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...`: exit `0`.
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...`: exit `0`.
- `gofmt` diff check and `git diff --check`: exit `0`.

One initial `gofmt` readiness invocation failed with exit `1` because its task-log path used one extra `..`; the command stopped before `gofmt` ran. The corrected readiness command and all listed gates then ran directly with their real exit codes preserved.
