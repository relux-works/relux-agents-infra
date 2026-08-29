# TASK-260828-2jbufw — Rework evidence: F1 group-stop attestation, F2 Windows test compile

Revision: worktree `task-board/story/STORY-260828-2faxgm`, HEAD 8045005 + uncommitted delta.
Scope: shared model-harness lifecycle only. No benchmark-driver or gate work, per the narrowed brief.

## F1 — the harness attested a stopped group it could not see

### Defect

`shutdownRuntime` (`tools/agents-infra/internal/modelharness/run.go`) returned success and
printed `process group N stopped after ...` the instant `exec.Cmd.Wait` reported the direct
child. Wait answers for the one process the harness holds a handle for. A same-group member
that redirected its inherited stdout/stderr away is invisible to it, so the harness wrote the
stopped-group record and exited 0 while that member was alive holding the runtime's port.

This is the attestation half of blocker B7 from TASK-260828-28gdmq. B7 fixed the orphaning
(signal `-pgid`, not the child); the group was still only assumed stopped afterwards. The
harness did not merely orphan — it certified success while doing it.

### Fix

- `runProcessGroupStopped` — new, per platform.
  - POSIX (`run_process_posix.go`): `kill(-pgid, 0)`. `ESRCH` is the only answer that means
    empty; `EPERM` means a live member that is not ours; anything else is a **failed read**
    returned as an error, never as an absence.
  - Windows (`run_process_windows.go`): answers for the direct child only, matching the stated
    no-job-object limitation instead of silently claiming the group contract.
- `shutdownRuntime` — the stopped record and the `0` exit are now gated on child-reaped **and**
  kernel-confirmed empty group. It polls every 50 ms after the child is reaped, escalates
  `SIGKILL` to the whole group when the grace expires, and returns a failure when the group
  state is still unknown after the kill grace.
- `README.md` — the stop contract paragraph now states the group-empty condition and the
  unknown-state failure.

### Regression tests (both fail against the pre-fix revision)

Fixture: `TestPortHolderHelperProcess`, a compiled helper that ignores SIGTERM/INT/HUP, holds a
loopback listener, and is started with stdio redirected away from the pipes Wait watches. All
three properties must hold at once, which is why it is a binary and not a shell line — a shell
`trap '' TERM` around a separate listener binary does not protect the socket holder.

| Test | Entry point | Asserts |
| --- | --- | --- |
| `TestRunSignalledShutdownWaitsForADetachedGroupMember` | `runWithSignals` seam, tuned grace | group gone + port rebindable at the instant of the claim; `is not empty`, `did not exit within`, `stopped after terminated` records |
| `TestModelHarnessRunDoesNotReportStoppedWhileAGroupMemberHoldsThePort` | shipped `model-harness run` binary, launcher config profile, real directed `SIGTERM` | same, against the exit code a supervisor reads |

Both assert **without a retry window**. The harness has already claimed the stop finished; a
wait-and-retry would launder a false claim into a pass.

### Mutants run

| Mutant | `settled()` behaviour | Result |
| --- | --- | --- |
| A — the pre-fix revision | report stopped on `Wait` alone | **killed**, both tests: `detached port holder (pid 3607) was alive at the instant the harness reported the group stopped: kill(pid, 0) = <nil>` |
| B — narrowing | keep the group check, but report stopped when the grace expires instead of escalating | **killed**: harness logged `... is not empty; waiting for the rest of the group` then `stopped` with the holder still alive |

Mutant A is the defect itself, reproduced at both the seam and the production entry point.
Mutant B is the narrowing one: it leaves the gate in place and only widens what counts as empty.

## F2 — POSIX test body broke the Windows test build

`GOOS=windows go test -c ./internal/modelharness` failed at six `syscall.Kill` lines.

- `run_shutdown_test.go` → `run_shutdown_posix_test.go`, behind `//go:build !windows`, the same
  constraint as `run_process_posix.go`. The now-redundant `requirePOSIX` runtime skip is gone.
- New unconstrained `run_shutdown_test.go` keeps a Windows-compilable test surface:
  `shutdownSignals` must include `os.Interrupt` on every platform (without it `signal.Notify`
  installs no handler, which is B7's zero-bytes death), and `runProcessGroupStopped` on an
  unstarted command must report stopped rather than error.

## Gates run (each a standalone process, real exit code)

| Gate | Exit |
| --- | ---: |
| `GOOS=windows go test -c -o /dev/null ./internal/modelharness` (F2, was 6 errors) | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` (no output) | 0 |
| `go test -count=1 ./internal/...` | 0 |
| `go test -count=1 .` (docs tests, after the README edit) | 0 |
| `go test -count=1 ./cmd/...` | 0 (no test files) |

Post-run `ps` shows no leaked fixture processes from this task. The two live
`mlx-swift-runtime-prototype` / `model-harness run qwen-benchmark-python` processes belong to
TASK-260827-2v13w8-rev4 and were left untouched.

## Not in scope, stated rather than silently fixed

The fatal-marker restart path in `runSupervisedAttempt` still kills the group and relaunches
without confirming the group emptied first; a survivor there collides with the replacement
runtime on the same port. Same class as F1, not touched here, not covered by these tests.
