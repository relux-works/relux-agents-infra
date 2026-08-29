# TASK-260826-12laby — Reviewer Verdict (CR rev 2)

- Verdict: **ACCEPTED**
- Change Request: `CR-TASK-260826-12laby-2` revision `2`
- Base OID `c40f0c26e071a9b466f1b856bebe91f19fb7390b` → candidate tree `5a15d04ef890067f6148d35f8df73be450fb1091`
- Repository delta: `present` (4 test files + LOGBOOK; **no production file changed** — verified by diff)
- Review host: darwin/arm64, go1.25.5, worktree `.temp/STORY-260825-1r7z9o/worktree`
- Working tree restored to the exact candidate tree after every mutant (`git diff <candidate> --stat` empty).

## How this was reviewed

The change was attacked, not read. Every gate the CR claims to witness was
independently narrowed in **production** code, the named witness was required to
fail, and the exact-version control was required to stay green. Production was
restored from a clean copy between every mutant.

## Independent narrowing mutants — all 7 killed, each by exactly one witness

| # | Production gate | Narrowing applied | Named witness that failed | Controls |
|---|---|---|---|---|
| A | `pi_shared_broker_darwin.go:836` client exec identity | dropped `\|\| clientExec.Ino != …Ino` | `TestSharedBrokerAttestClientRejectsEveryGateDeleteAndNarrowWitness/client_executable_same_device_wrong_inode` | 9/9 other subtests PASS |
| B | `pi_shared_client_darwin.go:341` broker exec identity | dropped `\|\| peerIdentity.Ino != ownIdentity.Ino` | `TestConnectAndAttestSharedRuntimeRejectsEveryGateDeleteAndNarrowWitness/broker_executable_same_device_wrong_inode` | 19/19 other subtests + exact-version test PASS |
| C | `pi_shared_operator_darwin.go:363` candidate scan | dropped `\|\| identity.Ino != ownIdentity.Ino` | `TestSharedRuntimeBrokerCandidatesRejectSameDeviceWrongInodeAtProductionEntry` | n/a (single test) |
| D | `pi_shared_operator_darwin.go:470` force-stop before signal | dropped `\|\| brokerIdentity.Ino != ownIdentity.Ino` | `TestSharedRuntimeForceStopRejectsRecordedBrokerIdentityNarrowingBeforeSignal/broker_executable_same_device_wrong_inode` | 4/4 other subtests PASS |
| E | `pi_shared_broker_darwin.go:839` protocol version | `!=` → `>` (accept below-current) | `…AttestClient…/past_protocol_version_range_narrowing` | `exact_protocol_version_control` PASS |
| F | `pi_shared_client_darwin.go:379` protocol version | `!=` → `>` | `…ConnectAndAttest…/past_protocol_version_range_narrowing` | `…ExactProtocolVersionReportsOnlyThePassedGateSet` PASS |
| G | `pi_shared_launcher_darwin.go:72` protocol version | `==` → `<=` | `TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry/past_protocol_version` | `exact_protocol_version_control` + `future_protocol_version` PASS |

Failure messages were checked, not just exit codes. Examples:
- C: `sharedRuntimeBrokerCandidates admitted same-device/wrong-inode candidate=…PID:10525…different-inode-candidate…`
- D: `force stop error={"code":"runtime_shutdown_timeout"…} want broker_stop_identity_mismatch` (kill is a stub; `signalAttempted` and real process liveness are both asserted, so the mutant admitted identity without the gate refusing).

## Reviewer's own execve mutant (H) — the rev-1 finding is genuinely closed

Replaced production `sharedRuntimeExecve = unix.Exec` with a `syscall.ForkExec` +
`Wait4` + `os.Exit(0)` mutant (launcher PID survives, child runs the target).

Result: exit 1 on
`TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry/exact_protocol_version_control` with

```
target event came from a process other than the execve'd launcher:
launcher_pid=11190 live=true exec_path=".../infra.test" target="/tmp/x2799353317/launcher-target"
```

Clean `unix.Exec` exits 0. The live `execve`-on-the-launcher-PID assertion is
restored, which is exactly what revision 1 dropped.

## Pre-CR baseline — the CR's stated value is real

Restored the four test files to base `c40f0c2` and re-ran mutants A–D against the
whole shared suite:

| Mutant | Pre-CR suite exit | Meaning |
|---|---|---|
| A | 0 | survived — genuinely unwitnessed before this CR |
| B | 0 | survived |
| C | 0 | survived |
| D | 1 | killed by `TestSharedRuntimeForceStopRefusesForgedBrokerIdentityWithoutSignal` (incidental foreign-binary test) |

This confirms the LOGBOOK's "three inode-removal mutants survived / force-stop
killed only incidentally" claim.

That same pre-CR run also reproduced the wall-clock flake this CR removes:
`TestSharedRuntimeLauncherRejectsAuthorizationChannelGuardBypassesAtProductionEntry/descriptor_three_is_a_socket_rather_than_a_fifo` failed after 15.02s.

## Load-flake control — replaced, and verified under load

30 launcher positive-control iterations (`-count=10` over the three launcher
suites) with 8 concurrent `yes` CPU-load generators: exit 0, 27.1s total. The
15s wall-clock `waitForSharedTest` deadline is gone; the control now consumes a
target-emitted stdout event.

## Concurrency check

The CR removes the `sync.Mutex` guarding `sharedLauncherRun.sawTarget`. Verified
safe: the poll goroutine's write happens-before every `carriedTarget()` read via
`close(donePoll)` / `<-run.donePoll` inside `wait()`, and every one of the 12
`carriedTarget()` call sites is preceded by a `run.wait(...)`. Focused `-race`
run over the launcher/attestation/broker suites: exit 0, 0 data races, 38.9s.

## Validation run on the restored candidate tree

| Check | Command | Result |
|---|---|---|
| Focused | `go test ./internal/infra/ -run <6 witness tests> -count=1 -v` | exit 0, 25.4s, 52 PASS |
| Focused race | `go test ./internal/infra/ -race -count=1 -run <shared suites>` | exit 0, 38.9s, 0 DATA RACE |
| Full package (chunk 1/2, 37 shared tests) | `go test ./internal/infra/ -count=1 -run <mask>` | exit 0, 36.2s |
| Full package (chunk 2/2, 242 remaining tests) | `go test ./internal/infra/ -count=1 -run <mask>` | exit 0, 131.5s |
| Final clean re-run (shared, post-restore) | `go test ./internal/infra/ -count=1 -run <mask>` | exit 0, 59.5s |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | no output |

The package was split only because a single shell call here is time-bounded; all
279 top-level tests in `internal/infra` were covered across the two chunks (37 +
242), with no `-run` mask overlap or omission.

## Finding carried forward (does NOT block acceptance)

**The LOGBOOK ROOT CAUSE line overstates prior coverage for one of the four gates.**

It says: *"Four production `Dev != … || Ino != …` gates had wrong-device
witnesses but no same-device/wrong-inode witness"*.

Three of the four do have a named same-inode/wrong-device witness
(`pi_shared_broker_admission_test.go:107`, `pi_shared_attestation_test.go:213`,
`pi_shared_integration_test.go:992`). The fourth —
`sharedRuntimeBrokerCandidates` (`pi_shared_operator_darwin.go:363`) — has none,
before or after this CR. Reproduction:

```
# drop only the Dev clause from sharedRuntimeBrokerCandidates:
#   if err != nil || identity.Dev != ownIdentity.Dev || identity.Ino != ownIdentity.Ino {
# ->  if err != nil || identity.Ino != ownIdentity.Ino {
go test ./internal/infra/ -count=1 -run '<all 37 shared tests>'   # exit 0 — mutant survives
```

Log: `.temp/TASK-260826-12laby/mutant-I-candidates-dev-only.log`.

Why this is not `changes_requested`:
- it is a statement about *prior* coverage, in the motivation section; every
  `EVIDENCE:` claim in the same entry was independently reproduced and is accurate;
- the gap it misdescribes is pre-existing and outside this task's scope and AC
  (the AC asks only that each gate be killed when its **inode** comparison is
  removed — verified for all four);
- `sharedRuntimeBrokerCandidates` calls package-level `processExecIdentity`
  directly with no dependency seam, so a same-inode/wrong-device witness for it
  is not expressible without a production seam change — which the Story
  deliberately treats as a last resort.

Recommended follow-up for the Story owner: correct that one sentence at
integration (it understates the gap, it does not overstate the fix), and consider
a separate leaf for the `sharedRuntimeBrokerCandidates` device clause.

## Definition of Done

- [x] Same-device/wrong-inode witnesses for all four executable-identity gates — verified by mutants A–D
- [x] Below-current protocol-version refusal witness and exact-version control — verified by mutants E–G
- [x] Each narrowed production gate killed independently — one failing subtest per mutant, controls green
- [x] Load-flaky wall-clock positive control replaced with an event-driven control — verified under 8× CPU load, 30 iterations
- [x] Review finding and closure evidence recorded (this artifact + board notes)
- [x] Focused, race, full, build, vet, formatting validation — all exit 0
- [x] Gating behavior covered by negative tests that fail when the gate admits what it must reject, production call site named
- [x] Implementation matches AC; solution fits project architecture (test-only, accepted shared-runtime production contract preserved)
- [x] Gate behavior attacked, not read
