# TASK-260826-12laby — Review Verdict: CHANGES REQUESTED

- Change Request: `CR-TASK-260826-12laby-1` revision `1`
- Base OID: `c40f0c26e071a9b466f1b856bebe91f19fb7390b`
- Candidate tree OID: `6a76724e61684e85a64063ddfd6a77d294795936`
- Verdict: **changes_requested** -> `to-dev`
- Reviewed by attacking the gates, not reading them. The working tree was verified
  byte-identical to the candidate tree (`git diff <candidate-tree>` empty) before
  and after every mutant; production files were restored after each run.

## Verdict summary

Six of the seven acceptance items land and are independently proven. One does
not: the rewrite of the launcher positive control from marker-poll to
event-driven **deleted the only live witness that the production launcher
`execve`s into the runtime** instead of forking a child. That is a witness
regression introduced by this CR, in the exact class the task exists to close,
and it is not disclosed in the LOGBOOK entry. Fix that and the CR is acceptable.

## What was proven (accepted work)

Seven independent narrowing mutants were applied to production one at a time,
each with the rest of the tree clean, and each was killed by its own named
witness. No mutant was delete-only; every one narrowed the gate.

| # | Production gate | Narrowing applied | Killed by | Result |
| --- | --- | --- | --- | --- |
| M1 | `pi_shared_operator_darwin.go:363` broker-candidate scan | dropped `identity.Ino != ownIdentity.Ino` | `TestSharedRuntimeBrokerCandidatesRejectSameDeviceWrongInodeAtProductionEntry` | FAIL, exit 1 |
| M2 | `pi_shared_operator_darwin.go:470` force-stop before signal | dropped `brokerIdentity.Ino != ownIdentity.Ino` | `...ForceStopRejectsRecordedBrokerIdentityNarrowingBeforeSignal/broker_executable_same_device_wrong_inode` | FAIL, exit 1 |
| M3 | `pi_shared_broker_darwin.go:836` broker admits client | dropped `clientExec.Ino != ...Ino` | `...SharedBrokerAttestClient.../client_executable_same_device_wrong_inode` | FAIL, exit 1 |
| M4 | `pi_shared_client_darwin.go:341` client attests broker | dropped `peerIdentity.Ino != ownIdentity.Ino` | `...ConnectAndAttestSharedRuntime.../broker_executable_same_device_wrong_inode` | FAIL, exit 1 |
| M5 | `pi_shared_broker_darwin.go:839` | `!=` -> `>` (admits below-current) | `...SharedBrokerAttestClient.../past_protocol_version_range_narrowing` | FAIL, exit 1 |
| M6 | `pi_shared_client_darwin.go:379` | `!=` -> `>` (admits below-current) | `...ConnectAndAttestSharedRuntime.../past_protocol_version_range_narrowing` | FAIL, exit 1 |
| M7 | `pi_shared_launcher_darwin.go:72` | `==` -> `<=` (admits below-current) | `...LauncherComparesEveryAuthorizationValueAtProductionEntry/past_protocol_version` | FAIL, exit 1 |

Notes on the accepted portion:

- All four `Dev`/`Ino` production comparisons in the package now have symmetric
  same-device/wrong-inode coverage. The previous suite had wrong-device
  witnesses only, so M1, M3 and M4 would have survived at base.
- The candidate-scan witness (M1) drives the real production entry
  `sharedRuntimeBrokerCandidates` with two genuinely-launched processes: one from
  the exact executable and one from a same-device byte copy at a different inode.
  The fixture asserts its own premise (`copyIdentity.Dev == ownIdentity.Dev &&
  copyIdentity.Ino != ownIdentity.Ino`) and fails rather than silently degrading
  if the temp dir lands on another device. Both directions are checked: the
  wrong-inode PID must be absent and the exact-executable PID must be present.
- Below-current protocol-version refusal is covered on all three version gates
  (client attestation, broker admission, launcher frame), each alongside an
  exact-version control that stays green. The renamed controls
  (`exact protocol version control`,
  `...ExactProtocolVersionReportsOnlyThePassedGateSet`) correctly state what they hold.
- Dropping the `sync.Mutex` around `sawTarget` is race-safe as written: the poll
  goroutine returns after its single write, and every `carriedTarget()` read is
  preceded in the same goroutine by `run.wait(...)`, which joins `<-run.donePoll`.
- The event-driven control is a large, real win on wall clock: the launcher suite
  runs in **11.055s** on the clean candidate.

## Finding — CONFIRMED, blocking

**The launcher positive control no longer witnesses `execve`; a fork+exec mutant
now survives the entire launcher suite.**

Production contract (`pi_shared_launcher_darwin.go:28,95`):

```go
sharedRuntimeExecve = unix.Exec
...
if err := sharedRuntimeExecve(resolved.Profile.Runtime.Executable, argv, environ); err != nil {
```

The launcher must *replace its own process image*. This is load-bearing, not
cosmetic: the authorization frame is bound to `LauncherPID`, and the launcher gate
compares `frame.LauncherPID == os.Getpid()`. If the launcher forked instead, the
PID the broker authorized and tracks would not be the PID of the runtime that
actually runs — the PID-binding chain would authorize one process and start another.

At base, that property was witnessed. `carriedTarget()` polls
`inspectSharedProcess(launcherPID).ExecPath == fixture.target`, i.e. the launcher's
*own* PID must come to be running the target. Every positive control required it:

```go
waitForSharedTest(t, sharedLauncherPositiveControlTimeout, func() bool {
    _, markerErr := os.Stat(fixture.marker)
    return markerErr == nil && run.carriedTarget()
}, "valid authorization never reached execve")
```

The CR replaces all five of those call sites with `run.requireTargetEvent(...)`,
which waits only for the target-emitted stdout line. A forked child inherits the
same stdout pipe and writes the same marker file, so it satisfies the new control
identically. After this CR, `carriedTarget()` is read **only** in negative
assertions (`|| run.carriedTarget()` -> fail if true). Nothing in the package
asserts it is ever true, so nothing asserts `execve` semantics.

### Failure scenario, reproduced

Mutant M8, applied to production only (`pi_shared_launcher_darwin.go:28`):

```go
sharedRuntimeExecve = func(path string, argv []string, env []string) error {
    child := exec.Command(path, argv[1:]...)
    child.Env = env
    child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
    if err := child.Start(); err != nil { return err }
    _ = child.Wait()
    os.Exit(0)
    return nil
}
```

| Test file state | Command | Result |
| --- | --- | --- |
| **CR revision 1** | `go test -run 'TestSharedRuntimeLauncher\|TestSharedRuntimeEveryShapeMutant\|TestSharedRuntimeShapeMutants\|TestSharedRuntimeRejectAllProbe' -count=1` | **ok, 261.236s — mutant survives** |
| base `c40f0c2` | same command, only `pi_shared_launcher_test.go` reverted to base | **FAIL: `.../valid (15.01s): valid authorization never reached execve`** |

The base suite kills M8 with the author's own message for this exact class. The CR
suite passes it. Same production mutant, same package, only the test file differs —
this is a regression in the change under review, not a pre-existing gap.

Two aggravating details:

1. The failure string at `pi_shared_launcher_test.go:376` still reads
   `"valid authorization never reached execve"`. The label continues to claim a
   property the assertion behind it can no longer detect — a witness that names a
   class it does not cover.
2. The LOGBOOK entry describes the rewrite as `Launcher positive controls consume a
   target-emitted stdout event instead of a 15s marker-poll deadline` and reports
   ten green repeats. It does not record that an assertion was dropped along the
   way. The green repeats are true and irrelevant to the property that was lost.

This is squarely inside the CR's own scope line: *"make the known positive control
timeout event-driven if this can be done without widening product behavior."*
Product behavior did not widen — the witness did.

### Required rework

Keep the event-driven control (it is correct and 24x faster); restore the property
alongside it. The event and the identity check are orthogonal and compose cleanly —
for example, gate on `targetHit` for liveness and then assert the launcher PID
itself is running the target:

```go
run.requireTargetEvent(t, "...")
if !run.carriedTarget() { t.Fatalf("launcher did not execve into the target: ...") }
```

Note that the poll goroutine now returns on `!observation.live()`, so a naive
composition can read `sawTarget` after the poller gave up. Whatever shape is
chosen, the bar is the same as for every other gate in this task:

1. At least one positive control asserts the launcher's own PID comes to run the
   target — not merely that the target ran.
2. The M8 fork+exec mutant above is applied to production and the named control
   **fails**, with the failing test name recorded.
3. The exact-`execve` control stays green on clean production.
4. The LOGBOOK entry records the assertion that was dropped and restored, so the
   next reader does not have to rediscover it.

## Validation actually run by this review

All in `tools/agents-infra`, macOS/darwin, on the candidate tree.

| Check | Command | Result |
| --- | --- | --- |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./internal/infra/` | exit 0 |
| Format | `gofmt -l ./internal/` | no output, exit 0 |
| New witnesses (focused) | `go test -run '<5 new/changed identity+protocol tests>' -count=1 -v` | **ok, 25.155s**, all subtests PASS |
| Launcher suite, clean | `go test -run 'TestSharedRuntimeLauncher\|...ShapeMutant\|...ShapeMutants\|...RejectAllProbe' -count=1` | **ok, 11.055s** |
| Mutants M1-M7 | one at a time, restored after each | each **FAIL exit 1** on its named witness |
| Mutant M8 | fork+exec, CR tests | **ok, 261.236s — survives** |
| Mutant M8 | fork+exec, base launcher test file | **FAIL** — regression confirmed |

Not rerun by this review, accepted from the producer's attached evidence: the
`-race` focused run, the three-way full-package split, and the ten repeated
positive controls. Those measure stability and are not in dispute; the finding
above is a coverage defect that a green run cannot surface by construction.

## Routing

`to-dev` for rework on the launcher positive control only. Everything else in
revision 1 — all four inode witnesses, all three below-current protocol witnesses,
the candidate-scan production-entry test, the renamed exact-version controls, and
the event-driven mechanism itself — is sound and should be carried forward
unchanged into revision 2.
