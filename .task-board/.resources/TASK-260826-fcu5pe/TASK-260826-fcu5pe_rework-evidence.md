# TASK-260826-fcu5pe rework evidence

## Candidate

- Commit: `63bd5381aafa86212f40bbd3aada0acb71fb4b6d`
- Tree: `685a204677ec4b137415cbded9441d50902fbde6`
- Current main: `e70f953969d46e451892d9f16e7401b879910b6b`
- Worktree: clean after the task-scoped commit.
- Rework delta from reviewed revision 1: five paths, limited to the shared-runtime client, its attestation/integration/launcher tests, and `LOGBOOK.md`.

The accepted current-main reconciliation recorded in
`TASK-260826-fcu5pe_reconciliation-and-validation.md` remains unchanged. This
rework does not touch `pi_config.go`, `pi_launch_posix.go`, `pi_plan.go`,
`main.go`, standalone Pi yolo policy, or any task-board Pi adapter surface.

## Review finding closure

`connectAndAttestSharedRuntime` now appends each `SharedRuntimeGateOutcome` only
after that production check passes. The former hardcoded 13-entry report is
removed. `SharedRuntimeStatusReport` still copies the client result, and the
two-lease production status test now requires the exact ordered 13-gate set,
with `passed` / `attested` on every member.

The client production entry is attacked with 16 divergent evidence shapes:
peer UID, peer zombie state, broker executable device, broker build device,
broker start time, future protocol version, empty runtime key, profile digest,
empty endpoint, runtime executable device, runtime UID, runtime start time,
runtime executable path, runtime argv, runtime zombie state, and model
discovery. Each asserts its typed refusal.

The forked `runtime runtime-launch` production entry is attacked with a socket
at descriptor 3, a CLI runtime key that differs from the recomposed profile,
content after the newline-delimited frame, a frame one byte over the 65536-byte
bound, `launcher_pid=0`, and an empty `exec_plan_digest`. Each guard attack has
a plain-valid control and proves the target is not reached.

## Mutation calibration

The task-scoped scratch driver copied the current module before every mutation,
required compilation, ran the named production-entry test uncached, and treated
only test exit `1` as a kill. Compile failures were rejected as invalid evidence.

- Client/report: 23 delete-or-narrow mutants, each expected-red `go test` exit
  `1`. This includes root/zombie, same-inode wrong-device, zero/empty,
  future-version range, runtime-process subfield, and missing reported-gate
  variants.
- Launcher: 10 delete-or-narrow mutants, each expected-red `go test` exit `1`.
  The descriptor, recomposed-key, trailing-content, and frame-bound variants
  reached the real target with `carried_target=true`; both zero-value comparison
  narrowings were also killed.
- Group driver exits: `client-a=0`, `client-b=0`, `launcher=0` after every
  constituent mutant produced the required exit `1`.

The authoritative logs are attached separately as task-scoped outcome
resources. Earlier scratch attempts containing compile-invalid mutants were
discarded and are not cited as evidence.

## Clean-tree validation

Every command ran as a foreground standalone process without a pipe or `tee`.

| Command | Exit | Result |
| --- | ---: | --- |
| `gofmt -l .` with non-empty output treated as failure | 0 | empty |
| `git diff --check` | 0 | clean |
| `go build ./...` | 0 | Darwin build |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux build |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows build |
| `go vet ./...` | 0 | configured landing validation |
| focused shared-runtime suite | 0 | `28.041s` |
| focused shared-runtime race suite | 0 | `42.899s` |
| oracle/calibration/production-entry mutant suite | 0 | `18.376s` |
| `go test ./... -count=1` | 0 | root `102.807s`; attachments `3.175s`; infra `152.173s` |

The source mutation runs are deliberately red individual gates, not passing
tests: their real exit is `1`, and the one-line rationale is that each narrowed
or deleted guard admits the exact forged evidence its production test requires
the implementation to refuse.
