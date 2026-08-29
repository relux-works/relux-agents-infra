# TASK-260828-2jbufw review verdict — CR revision 2

Verdict: **ACCEPTED**

Change Request: `CR-TASK-260828-2jbufw-2`, revision `2`
Base: `804500529d613d4a3cff182376c8f7fdc6c26c1f`
Candidate tree: `ca0c15e99da30b0320b07657b17232a4ac816ac4`
Patch SHA-256: `c447707f7f9ec798794760d61b1043506756182a3e2b47106c0a68e67dc84ab8`

## Scope reviewed

This is the requested round-2 review of F1 and F2 only:

- F1: reject a successful stopped-group attestation while a same-group member survives.
- F2: keep POSIX lifecycle coverage while restoring Windows test cross-compilation.

The already-accepted llama-server lifecycle/readiness evidence, B8 answer, and G1/G2 analysis from round 1 were not reopened. The nine changed paths contain no benchmark-driver or benchmark-compare implementation.

Every changed working-tree file hashes to its blob in candidate tree `ca0c15e…`. The downloaded revision-2 patch hashes to the Change Request digest, and the exact base-to-candidate diff passes `git diff --check`.

## F1 — group shutdown attestation

The production call chain is live: `model-harness run` calls `modelharness.Run`, then `run` / `runWithSignals`, then `runOnce` or `runSupervisedAttempt`, and a received supervisor signal calls `shutdownRuntime`. On POSIX, the success record is now gated by both the direct child being reaped and `runProcessGroupStopped` observing `kill(-pgid, 0) == ESRCH`. A failed inspection is not treated as an empty group.

I drove the shipped CLI with an independent review-only fixture. Its same-group grandchild:

- ignored SIGTERM/INT/HUP;
- closed stdin/stdout/stderr inherited from the runtime child;
- held a real loopback listener;
- remained invisible to `exec.Cmd.Wait` after the direct child exited.

The fixture itself fails non-zero if the harness returns exit 0 before both the exact PGID disappears and the port is immediately bindable again. It also keeps a separate process-group sentinel alive to catch signalling wider than the runtime-owned group.

Candidate results:

| Scenario | Elapsed | PGID gone before exit 0 | Port free before exit 0 | Out-of-group sentinel | Escalation |
| --- | ---: | --- | --- | --- | --- |
| SIGTERM-ignoring detached holder | `10.052s` | yes | yes | alive | expected group SIGKILL after 10s grace |
| ordinary clean group | `2ms` | yes | yes | alive | none |

The stubborn log records the child exit while the group remains non-empty, the grace expiry, group kill, and only then `stopped after terminated`. The clean log contains no timeout or kill record and exits promptly.

I then built the production CLI through a narrowing overlay mutant which preserved the group check but incorrectly widened “stopped” to include the first grace-period timeout. Against that mutant, the harness printed a stopped record while PGID `72543` remained live; the reviewer fixture exited `2` with `harness returned 0 while process group ... remained`. The mutant was therefore killed by the negative fixture. Post-run checks established that the candidate and mutant fixture PIDs and PGIDs were all gone.

No shutdown hang, clean-path failure, or cross-group kill reproduced.

## F2 — Windows test compilation without deleting POSIX coverage

`GOOS=windows GOARCH=amd64 go test -c -o <scratch>/modelharness-windows-amd64.test.exe ./internal/modelharness` exits `0`. Windows `go vet` also exits `0`.

The POSIX coverage was scoped, not deleted:

- `run_shutdown_posix_test.go` has `//go:build !windows` and still contains seven lifecycle tests plus its helper-process entry point, including both detached-member negative tests.
- unconstrained `run_shutdown_test.go` retains two cross-platform shutdown contract tests.
- the native full package run executes the POSIX tests successfully.

## Validation

| Check | Result |
| --- | --- |
| Candidate blob identity for all 9 changed paths | pass |
| CR patch SHA-256 | exact match |
| `git diff --check <base> <candidate>` | pass |
| `gofmt -l` on changed Go files | no output |
| `go vet ./internal/modelharness` | pass |
| `GOOS=windows GOARCH=amd64 go vet ./internal/modelharness` | pass |
| Windows test cross-compile | pass |
| `go test -count=1 -v ./internal/modelharness` | pass, `13.512s` |
| shutdown-focused `go test -race -count=1` | pass, `14.414s` |
| `go test -count=1 . ./cmd/...` | pass, root `81.437s`; command package builds |
| independent production-entry group-bypass fixture | pass |
| timeout-as-stopped narrowing mutant | killed |

## Verdict

Revision 2 closes F1 and F2 without overshooting the managed group boundary or deleting POSIX coverage. The ordinary shutdown remains prompt and successful. No changes are requested.
