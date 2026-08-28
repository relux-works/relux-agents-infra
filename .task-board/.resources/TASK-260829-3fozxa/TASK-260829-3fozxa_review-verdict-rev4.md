# TASK-260829-3fozxa Review Verdict — Change Request revision 4

## Verdict

**Accepted.** CR `CR-TASK-260829-3fozxa-4`, revision 4.

- Base OID: `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`
- Candidate tree OID: `6d63fe072ad717eb4d2b42353fe679340269e3ad`
- Reconstructed independently via `git worktree add --detach <base>` +
  `git read-tree -m -u <tree>`; `git write-tree` reproduced
  `6d63fe072ad717eb4d2b42353fe679340269e3ad` exactly.
- 17 changed paths match the CR manifest; `git diff --stat` matches.

## Prior finding F1 (revision 1) — verified fixed

Revision 1 was rejected for a restart/config-change bypass: `Prune` counted
archives but never validated their sizes against the *current* cap, so an
archive written under an older, larger `max_segment_bytes` survived a broker
restart and pushed retained output above `max_segment_bytes * max_segments`.

Revision 4 fixes this in `sharedFilesystemLogSink.Prune`
(`pi_shared_log_rotation_darwin.go:52`): every managed archive is `Lstat`'d and
validated as a mode-0600, single-link regular file no larger than the current
`maxSegmentBytes` *before* count-based pruning; any violation fails closed with
`shared_runtime_state_path_invalid` and removes nothing. This runs from
`newSharedRotatingLogWriter` → `openSharedRotatingLog` →
`startUnauthorizedRuntimeWithDependencies`, i.e. before `command.Start()`
(`pi_shared_broker_darwin.go:369` precedes line 387), so refusal happens before
any runtime-launch side effect.

The required regression test exists and drives the real path:
`TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart`
(`pi_shared_log_rotation_darwin_test.go:18`) plants a pre-existing 10-byte
archive under `max_segment_bytes=4, max_segments=2`, calls
`startUnauthorizedRuntimeWithDependencies` with a `startCommand` stub that
fails the test if invoked, and asserts refusal, zero start calls, and byte-for-
byte file preservation.

**Adversarial check (this review):** widened the archive-size gate from
`stat.Size > maxSegmentBytes` to `stat.Size > maxSegmentBytes*3` (10 ≤ 12, so
the bypass would resurface). The regression test failed exactly as expected
(`runtime command start side effect calls=1 want=0`), proving the test is not
a delete-only mutant target — it detects a narrowed gate. Reverted; tree hash
confirmed unchanged afterward. A `*2` narrowing (10 vs 8) still triggers
refusal by coincidence of the fixture's numbers, so `*3` was required to
demonstrate a live, escaping bypass — recorded here so a future reviewer
doesn't waste a cycle re-deriving the right multiplier.

## Acceptance-criteria verification

- **No numeric code defaults**: `MaxSegmentBytes`/`MaxSegments` route through
  `requiredPositiveInt` (`pi_config.go:715`), which errors on absence, non-
  positive values, and platform-int overflow. Both fields are threaded into
  `rejectUnknownFields`/parsing with no fallback constant anywhere in
  `pi_config.go`, `pi_shared_log_rotation*.go`.
  `TestRunPiRejectsMissingSharedRuntimeLogRotationPolicy` and
  `TestRunPiRejectsZeroAndOverflowingSharedRuntimeLogRotationPolicy` drive
  `RunPi` end-to-end for absent/zero/overflow on both fields — pass.
- **Rotation at exact byte cap**: `sharedRotatingLogWriter.Write` rotates when
  `currentBytes == maxSegmentBytes` before writing the next chunk, splitting a
  crossing write byte-exact.
  `TestSharedRuntimeLogRotatesBeforeFirstBytePastExactCap` proves an exact-cap
  write does *not* rotate early and a crossing write splits `abcd|efgh|i` — pass.
- **Deterministic oldest-first pruning**: archive filenames encode a zero-
  padded 20-digit monotonic sequence (`nextArchivePath`,
  `pi_shared_log_rotation_darwin.go:81`), so `sort.Strings` orders oldest-first
  independent of filesystem timestamp resolution — no wall-clock dependency.
  `TestSharedRuntimeLogPrunesOldestSegmentsDeterministically` — pass.
- **Aggregate bound `max_segment_bytes * max_segments`, active segment
  counted correctly**: production `Prune` retains `maxSegments-1` archives
  plus the 1 active segment = `maxSegments` total, matching the fake sink's
  semantics (which counts the active segment in `len(segments)`).
  `TestSharedRuntimeLogMultiDayFootprintNeverExceedsConfiguredProduct` drives
  45 simulated days via `clock.advance(24*time.Hour)` (no sleep) and asserts
  `sink.footprint() <= maxSegmentBytes*maxSegments` and
  `len(segments) <= maxSegments` after every day — pass.
- **No-follow / foreign-file safety across restart**:
  `TestOpenSharedRotatingLogRefusesManagedArchiveSymlinkWithoutTouchingForeignTarget`
  proves a symlinked archive name is refused via `Lstat` (not followed) and the
  foreign symlink target is untouched. The active log path itself already used
  `O_NOFOLLOW` + regular-file/single-link/mode checks pre-CR
  (`openSharedLog`, `pi_shared_broker_darwin.go:423`), unchanged here.
- **Writer lifetime across restart paths**: `startUnauthorizedRuntimeWithDependencies`
  now passes the rotating writer as `Stdout`/`Stderr` (previously a bare
  `*os.File`), which makes `os/exec` copy through a pipe requiring the writer
  to outlive `Start()`. `waitForPiProcessAndClose` closes it only after
  `command.Wait()` returns, propagating a close error only when `Wait` itself
  did not error. This is exercised indirectly by the existing
  `TestPiLaunchSerializesRuntimeOutputFanIn` (unrelated to this CR's diff,
  unchanged file).

## Trunk composition (per rev4 review scope brief)

Current `origin/main` is `675f77ed63376320ed1213f46f9462a299c0abaf`, 2 commits
ahead of this CR's base, adding persisted restart/quarantine status fields in
`pi_shared_operator_darwin.go` plus supervision test/doc changes — none of
which the candidate touches.

`git merge-tree --write-tree --merge-base=891de4427bb7de6885b8b221f0e2b24a49a8fdc2 675f77ed63376320ed1213f46f9462a299c0abaf 6d63fe072ad717eb4d2b42353fe679340269e3ad`
reported exactly one conflict: `LOGBOOK.md` (both sides append dated,
non-overlapping entries — a textual, not semantic, conflict). `README.md`,
`pi_shared_supervision_test.go`, and `runtime_main_darwin_test.go` auto-merged
cleanly; no conflict touched any production file, and no restart/quarantine
status field is shadowed, lost, or reported stale by the candidate.

Verified by hand-resolving the trivial `LOGBOOK.md` conflict (interleaving the
two dated entries in a disposable worktree, not the producer workspace) and
building/testing the composed tree: `go build ./...` and `go vet ./...` exit 0;
rotation tests (`SharedRuntimeLog*`, `StartUnauthorizedRuntimeRefusesOversized*`,
`OpenSharedRotatingLogRefusesManagedArchiveSymlink*`) and restart/status tests
(`SharedRuntimeStatus*`, `SharedRuntimeRestartPolicy*`,
`SharedBrokerServePersistsStableResetAndFailedHalfOpen*`) all pass together in
the composed tree — no interference either direction.

## Full validation (candidate tree, disposable worktree)

- `go build ./...` — exit 0.
- `go vet ./...` — exit 0.
- `gofmt -l .` — no output (clean).
- Focused rotation + config-gate tests (`-run` mask covering all
  `pi_shared_log_rotation*.go` tests plus
  `TestRunPiRejectsMissingSharedRuntimeLogRotationPolicy` and
  `TestRunPiRejectsZeroAndOverflowingSharedRuntimeLogRotationPolicy`) — pass,
  0.81s.
- `go test ./... -count=1`: first run failed two tests neither touched by this
  CR's diff nor related to log rotation —
  `TestPiLaunchSerializesRuntimeOutputFanIn` and
  `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry`
  (readiness-polling timing). Reproduced the same failure on the unmodified
  **base** commit (`891de44`) under identical host load, confirming this is a
  pre-existing host-timing flake, not a regression introduced by this CR (also
  matches the candidate's own `LOGBOOK.md` entry documenting this exact
  flake). A subsequent full run on the candidate tree passed cleanly:
  root package 541.088s, `internal/infra` 253.245s, exit 0.

## Evidence this review attacked, not just read

- Narrowing mutant on `sharedFilesystemLogSink.Prune`'s size gate — killed the
  regression test as required, then reverted (tree hash re-verified clean).
- Confirmed the refusal path precedes `command.Start()` by reading call order,
  not by trusting the test name.
- Reproduced the pre-existing flaky-test failure on the unmodified base commit
  before accepting it as unrelated, rather than assuming the candidate's own
  logbook claim.
- Independently reconstructed the candidate tree from base + tree OID rather
  than trusting the stated patch digest, and independently ran
  `git merge-tree` against exact current trunk rather than trusting the
  candidate's LOGBOOK.md narrative about composition.

No unstaged mutation remains in the producer workspace; all review activity
happened in disposable worktrees under
`.temp/TASK-260829-3fozxa/review/{rev4,base,merged}`, which have been removed.
