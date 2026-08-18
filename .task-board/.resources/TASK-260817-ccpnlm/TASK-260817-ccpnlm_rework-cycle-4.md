# TASK-260817-ccpnlm — Developer rework cycle 4

## Scope

Closed the managed Pi session-isolation bypass reported by reviewer cycle 3.

- Managed `--session-dir` now fails with `invalid_provider_arguments` before state or process side effects.
- Managed `--session` and `--fork` reject every selector shape that pinned Pi interprets as a filesystem path: a `/`, a `\`, or a `.jsonl` suffix.
- Plain session-ID selectors, `--continue`, and `--resume` remain supported and use the generated hash-contained `PI_CODING_AGENT_SESSION_DIR`.
- README and Flight Logbook describe the boundary.

Production-entry regression coverage drives `runPi`, attempts global and path-shaped session selectors, verifies the runtime sentinel never executes, verifies no cache state is created, and verifies the global session sentinel remains byte-identical. Narrowing cases cover absolute paths, `.jsonl` paths, and backslash-shaped paths.

## Evidence

Expected-red before the production fix:

- `go test ./internal/infra -run 'TestManagedPiArgvBridge(RejectsIdentityLookalikesAndUnsafeSuffix|KeepsIsolatedSessionModes)$' -count=1` — exit `1`; `BuildManagedPiArguments` admitted `--session-dir /tmp/global-sessions`.

Post-fix validation:

- `go test ./internal/infra -run 'TestManagedPiArgvBridge(RejectsIdentityLookalikesAndUnsafeSuffix|KeepsIsolatedSessionModes)$' -count=1` — exit `0`.
- `go test . -run 'TestRunPi(RejectsManagedSessionPathOverridesBeforeSideEffects|PrintConfigKeepsManagedContinueAndResumeContained)$' -count=1` — exit `0`.
- `go test -race ./internal/infra -run 'Test.*Pi' -count=1` — exit `0`.
- `go test -race . -run 'TestRunPi' -count=1` — exit `0`.
- `go test -race . -run 'TestRunPi(RejectsManagedSessionPathOverridesBeforeSideEffects|PrintConfigKeepsManagedContinueAndResumeContained)$' -count=1` — exit `0`.
- `go test ./... -count=1` — exit `0`.
- `go vet ./...` — exit `0`.
- `go build ./...` — exit `0`.
- `gofmt -l .` — exit `0`, no output.
- `git diff --check` — exit `0`.
- `task-board validate` — exit `0`, board valid.

## Explicit boundary

The launcher still does not claim protection against a malicious reviewed runtime executable or a same-UID process that wins the post-preflight listener race. This rework changes only managed Pi session-path isolation and parser admission.
