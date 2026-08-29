# TASK-260829-3fozxa Revision 2 Implementation Evidence

## Outcome

Revision 1's restart/config-change bypass is closed with a fail-closed policy.
The production filesystem sink now validates every managed retained archive
against the current `max_segment_bytes` before deterministic count pruning.
An oversized, non-regular, linked, or incorrectly permissioned managed archive
refuses writer construction before the runtime command starts.

## Production path covered

`startUnauthorizedRuntime` -> `startUnauthorizedRuntimeWithDependencies` ->
`openSharedRotatingLog` -> `newSharedRotatingLogWriter` ->
`sharedFilesystemLogSink.Prune`.

`TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart`
creates a 1-byte active segment and a valid mode-0600 10-byte managed archive,
then configures `max_segment_bytes=4` and `max_segments=2`. It uses the
production filesystem sink plus a fake clock and asserts that the injected
`Cmd.Start` boundary receives zero calls. Both files remain unchanged after
refusal.

`TestOpenSharedRotatingLogRefusesManagedArchiveSymlinkWithoutTouchingForeignTarget`
proves the expanded archive-validation pass does not follow a managed-shaped
symlink or alter its foreign target.

## Acceptance mapping

- Both caps remain mandatory operator configuration with no numeric code defaults.
- Exact-cap splitting and oversized input are covered by the shared writer tests.
- Active plus at most `max_segments - 1` archives are retained; every retained
  file is no larger than `max_segment_bytes`.
- Oldest-first pruning remains sequence-name deterministic under equal clocks.
- The 45-day fake-sink/fake-clock simulation remains bounded by
  `max_segment_bytes * max_segments` without wall-clock sleeps.
- Restart after lowering `max_segment_bytes` now refuses an older oversized
  managed archive before runtime launch side effects.

## Adversarial evidence

A temporary narrowing mutant changed the archive size gate from
`size > max_segment_bytes` to `size > 3 * max_segment_bytes`.

- Command: `go test ./internal/infra -run '^TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart$' -count=1 -v`
- Exit: 1 (expected red)
- Failure: `runtime command start side effect calls=1 want=0`

The source was restored without Git checkout/reset, compared byte-for-byte with
the task-scoped pre-mutant copy (`cmp`, exit 0), and the named test was rerun
green (exit 0).

## Validation

| Command | Exit | Result |
| --- | ---: | --- |
| Focused rotation/config/production-path tests, `-count=1 -v` | 0 | Pass |
| Narrowed-gate mutant named test | 1 | Expected red; production start boundary reached |
| Restored named regression test, `-count=1 -v` | 0 | Pass |
| `go test ./... -count=1` (run 1) | 1 | Unrelated readiness timing failure; recorded, not treated as green |
| Exact failing readiness test rerun, `-count=1 -v` | 0 | Pass |
| `go test ./... -count=1` (run 2) | 0 | Pass; root 186.435s, infra 339.288s |
| `go vet ./...` | 0 | Pass |
| `go build ./...` | 0 | Pass |
| `gofmt -l internal/infra` plus empty-output assertion | 0 / 0 | Clean |
| `git diff --check` | 0 | Clean |

Run 1 failed only in
`TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry`:
one subtest missed its readiness-count file and another observed timeout instead
of early exit. The exact test immediately passed on rerun, and the required full
suite subsequently passed. No rotation code was changed in response to this
transient failure.

## Evidence files

- `.temp/TASK-260829-3fozxa/focused-tests-02.log`
- `.temp/TASK-260829-3fozxa/mutant-01-expected-red.log`
- `.temp/TASK-260829-3fozxa/mutant-restored-green-01.log`
- `.temp/TASK-260829-3fozxa/go-test-all-01.log`
- `.temp/TASK-260829-3fozxa/unrelated-readiness-rerun-01.log`
- `.temp/TASK-260829-3fozxa/go-test-all-02.log`
- `.temp/TASK-260829-3fozxa/go-vet-all-01.log`
- `.temp/TASK-260829-3fozxa/go-build-all-01.log`
- `.temp/TASK-260829-3fozxa/gofmt-check-01.log`
- `.temp/TASK-260829-3fozxa/git-diff-check-01.log`

Base HEAD for this Story worktree: `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`.
