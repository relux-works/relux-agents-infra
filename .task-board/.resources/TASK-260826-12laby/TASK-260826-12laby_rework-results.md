# TASK-260826-12laby — developer rework results

## Outcome

Closed the revision-1 review finding without changing shared-runtime production
behavior. The exact-version launcher control still waits on the target-emitted
stdout event, then kernel-inspects the original launcher PID and requires that
PID's executable path to equal the configured target. This restores the
same-PID `execve` witness without restoring the load-flaky wall-clock poll.

Changed in this rework:

- `tools/agents-infra/internal/infra/pi_shared_launcher_test.go`
- `LOGBOOK.md`

The four same-device/wrong-inode witnesses, three below-current version
witnesses, and all production files are unchanged from revision 1.

## Negative evidence rerun in this developer run

The reviewer's M8 fork+exec mutant was applied to the real production variable
`sharedRuntimeExecve` in `pi_shared_launcher_darwin.go`, one mutant at a time,
with `-count=1`.

| Production call site | Narrowing mutant | Named witness | Exit | Result |
| --- | --- | --- | ---: | --- |
| `RunSharedRuntimeLauncher` -> `sharedRuntimeExecve` | replace same-process `unix.Exec` with fork, child exec, and parent wait | `TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry/exact_protocol_version_control` | 1 | Target event arrived, but kernel identity for launcher PID still named `infra.test`, not the target. |

The expected-red command was run directly and failed in `11.48s`; its real exit
code was 1. Production was then restored from a byte copy, `cmp` succeeded, and
`git diff --exit-code -- internal/infra/pi_shared_launcher_darwin.go` exited 0.
The same named exact-version control on clean `unix.Exec` exited 0 in `1.02s`
(`1.912s` package time).

M1-M7 were not rerun in this rework run. Their independent exit-1 evidence is
carried from `TASK-260826-12laby_mutation-evidence.md` and was independently
rerun and accepted by revision-1 review in
`TASK-260826-12laby_review-verdict.md`. This rework changes only the launcher
positive-control helper and LOGBOOK evidence; none of those seven witnesses or
their production gates changed.

## Validation rerun in this developer run

Every command below was run directly as a standalone foreground process. No
result was piped through `tee`, and every reported green gate exited 0.

| Gate | Command | Exit | Result |
| --- | --- | ---: | --- |
| Repeated launcher controls | `go test ./internal/infra -run '^(TestSharedRuntimeLauncherRejectsAuthorizationChannelGuardBypassesAtProductionEntry|TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry)$' -count=10` | 0 | `45.961s` |
| Focused | `go test ./internal/infra -run 'SharedRuntime|SharedBroker' -count=1` | 0 | `56.795s` |
| Focused race | `go test -race ./internal/infra -run 'SharedRuntime|SharedBroker' -count=1` | 0 | `74.952s` |
| Full root package | `go test . -count=1` | 0 | `139.808s` |
| Full attachments package | `go test ./internal/attachments -count=1` | 0 | `1.815s` |
| Full infra package | `go test ./internal/infra -count=1` | 0 | `181.982s` |
| Build | `go build ./...` | 0 | No output |
| Vet | `go vet ./...` | 0 | No output |
| Formatting | `gofmt -l .` | 0 | No output |
| Patch integrity | `git diff --check` | 0 | No output |

The three package test commands are the bounded full-module equivalent of
`go test ./... -count=1` for this module.

## Architecture and scope

The repair remains test-only and drives the real `runtime-launch` subprocess.
It uses the existing kernel process inspection path after a target-controlled
event; it adds no product seam, flag, fallback, timeout, or production behavior.
