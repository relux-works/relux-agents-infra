# TASK-260817-ccpnlm developer rework — cycle 5

## Scope

Closed the reviewer cycle-4 managed Pi session-state bypass through `--export`.

- `BuildManagedPiArguments` now refuses managed `--export` with `invalid_provider_arguments` before managed state, lock, socket, runtime, or Pi side effects.
- The pinned Pi value-option audit found no other direct agent/session-state override outside the already handled `--session-dir`, path-shaped `--session`/`--fork`, and `--export` surfaces. Plain session/fork IDs, `--continue`, and `--resume` remain rooted by `PI_CODING_AGENT_SESSION_DIR`.
- Native Pi passthrough remains unchanged when Pi policy is genuinely absent.
- README and Flight Logbook record the managed behavior and regression closure.

## Negative production evidence

`TestRunPiRejectsManagedSessionPathOverridesBeforeSideEffects/global_session_export` drives the production `runPi` entry point with a sentinel file beneath the normal global Pi session tree. It requires:

- `invalid_provider_arguments`;
- no runtime sentinel launch;
- byte-identical global session content; and
- no managed cache state creation.

The unit-level argument bridge test also refuses managed `--export`, so narrowing the production state-argument gate to the earlier three session options makes the focused suite fail.

## Validation

All commands ran directly as standalone processes.

| Command | Exit code | Result |
| --- | ---: | --- |
| `go test -race ./internal/infra -run 'Test.*Pi' -count=1` | 0 | pass |
| `go test -race . -run 'TestRunPi' -count=1` | 0 | pass |
| `go test ./... -count=1` | 0 | pass |
| `go vet ./...` | 0 | pass |
| `go build ./...` | 0 | pass |
| `gofmt -l .` | 0 | pass; no output |
| `git diff --check` | 0 | pass; no output |
| `task-board validate` | 0 | pass |

Global runtime installation was intentionally not run: downstream `TASK-260817-3a0zr3` owns installation/alias verification, while this developer handoff leaves the source diff reviewable and uncommitted.
