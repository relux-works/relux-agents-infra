# TASK-260817-ccpnlm reviewer verdict — changes requested

Verdict branch: `changes_requested -> to-dev`.

## Finding

1. **P1 — Managed session isolation remains bypassable through `--export`.** `piKnownValueOptions` classifies `--export` as a normal forwarded value option (`tools/agents-infra/internal/infra/pi_args.go:51-58`). The generic branch calls `validateManagedPiSessionArgument`, but that guard returns success for every option except `--session-dir`, `--session`, and `--fork` (`pi_args.go:178-188,282-296`). The resulting production plan forwards an arbitrary global Pi session path unchanged. Pinned Pi handles `parsed.export` before its normal session-manager flow and calls `exportFromFile(parsed.export, outputPath)` (`.temp/TASK-260817-ccpnlm/pi-main-pinned.ts:626-637`), directly reading the caller-selected file.

Production defeat probe:

`agents-infra pi --print-config --export <HOME>/.pi/agent/sessions/global.jsonl`

- Exit: `0` with `status:"ok"`.
- Both `launch_variants.interactive.argv` and `managed_host.argv` contain the exact global session path after `--export`.
- The same plan reports a different hash-contained `pi.state.sessions_dir`; the forwarded export source bypasses it.
- Evidence: `.temp/TASK-260817-ccpnlm/export-bypass-cycle4-01.json` and `.temp/TASK-260817-ccpnlm/export-bypass-cycle4-01.stderr`.

This is the standard **bypass path around the check** shape. Cycle 4 protects three path-bearing session options, but the adjacent production export path reaches the same protected global session data without invoking those checks.

Required rework: fail closed on managed `--export`, or accept it only after proving the source path is anchored beneath the generated hash-contained session directory with complete no-follow/read-failure semantics. Add a production-entry negative that attempts to export a sentinel global Pi session and proves refusal before runtime/state side effects plus byte-identical global content. Add a narrowing control for the explicitly supported contained export shape, if export remains supported. Audit the remaining pinned value options for any other direct read/write path that can override the managed agent/session boundary.

## Validation performed

- `go test -race ./internal/infra -run 'Test.*Pi' -count=1` — pass.
- `go test -race . -run 'TestRunPi' -count=1` — pass.
- `go test ./... -count=1` — pass.
- `go vet ./...` — pass.
- `go build ./...` — pass.
- `gofmt -l .` — pass, no output.
- `git diff --check` — pass.
- `task-board validate` — pass.
- Source-built production `--export` defeat probe — gate defeated; global session path admitted into the composed production artifact.

Cycle 4 correctly closes `--session-dir`, path-shaped `--session`, and path-shaped `--fork`, and its new tests pass. Acceptance is withheld because another pinned Pi path-bearing mode still reads outside managed state.
