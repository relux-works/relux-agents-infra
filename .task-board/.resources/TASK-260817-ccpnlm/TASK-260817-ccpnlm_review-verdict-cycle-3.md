# TASK-260817-ccpnlm reviewer verdict — changes requested

Verdict branch: `changes_requested -> to-dev`.

## Finding

1. **P1 — Managed session isolation is bypassable through forwarded `--session-dir`.** `BuildManagedPiArguments` classifies `--session-dir` as an ordinary value option (`tools/agents-infra/internal/infra/pi_args.go:51-57`) and forwards it unchanged (`pi_args.go:178-185`). `RunPi` later sets the intended isolated `PI_CODING_AGENT_SESSION_DIR` (`pi_launch_posix.go:202-204`), but the pinned Pi documentation states that `--session-dir` overrides that environment variable (`docs/environment-variables.md:82`; `docs/settings.md:243`). A caller can therefore redirect managed session reads/writes into the normal global Pi tree, defeating the task's no-global-Pi-state and isolated-session boundary.

Production defeat probe:

`agents-infra pi --print-config --session-dir <HOME>/.pi/agent/sessions`

- Exit: `0`.
- Both `launch_variants.interactive.argv` and `managed_host.argv` contain the exact global path after `--session-dir`.
- The plan simultaneously reports a different hash-contained `pi.state.sessions_dir`; Pi will ignore that environment-backed location because CLI precedence wins.
- Evidence: `.temp/TASK-260817-ccpnlm/session-dir-bypass-01.json`.

This is the standard **bypass path around the check** shape: isolated environment state is present, but a second production input surface overrides it.

Required rework: fail closed on managed session-location overrides before runtime/state side effects (at minimum `--session-dir`; audit the adjacent path-bearing `--session` and `--fork` surfaces against the same no-global-tree contract). Add a production-entry negative test that attempts a global Pi session path, observes the named refusal before runtime start, and proves global session sentinels remain byte-identical. Include a narrowing control demonstrating that ordinary isolated session modes such as `--continue`/`--resume` still use the generated managed directory.

## Validation performed

- `go test -race ./internal/infra -run 'Test.*Pi' -count=1` — pass.
- `go test ./... -count=1` — pass.
- `go vet ./...` — pass.
- `go build ./...` — pass.
- `task-board validate` — pass.
- Production session-dir defeat probe — gate defeated; exit 0 and global session path present in the composed production artifact.

The cycle-2 output race is fixed and its focused/full race coverage passes. Green validation does not accept the task because the production argv bridge still provides a direct override around the managed state boundary.
