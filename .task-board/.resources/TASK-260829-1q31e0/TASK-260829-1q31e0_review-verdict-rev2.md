# TASK-260829-1q31e0 review verdict (revision 2)

## Verdict

**accepted** for CR `CR-TASK-260829-1q31e0-2` revision 2, candidate tree
`bcdf1c032276742220b3ae2cb27d51d470796ead`, base `675f77ed63376320ed1213f46f9462a299c0abaf`.

## Identity verification

- `git diff bcdf1c03...` against the worktree, restricted to the 17 changed
  paths (via a scratch index, since `.task-board/` inside the worktree is a
  checkout artifact and was excluded), is empty: the worktree exactly equals
  the declared candidate tree.
- `git diff --binary <base> <candidate> | shasum -a 256` reproduces
  `ec96ceb8ab23a78e4074b96210c9de538ade2d6e1aaaa6cc5cf00cd745bd8bab`, matching
  the CR's declared patch resource hash exactly.
- Base commit `675f77ed` is exactly current `origin/main`-equivalent trunk
  (matches `git log` HEAD at session start); no divergence.

## Revision 1 blocking finding: resolved

Revision 1 pruned only the caller-supplied `paths.LogsDir`, so each distinct
`runs/<run-key>/logs/` directory (one per shared/standalone run ID) got an
independent full allowance — aggregate count/bytes/age were unbounded across
runs, and `--print-config` observed none of them (P1 in
`review-verdict-rev1-precondition.md`).

Revision 2 replaces the single-directory scan with a profile-scoped aggregate
(`pi_lifecycle_log_posix.go`):

- `openPiLifecycleLogDirectories` discovers the profile's `logs/` directory
  *and* every `runs/<64-hex>/logs/` directory by listing `runs/` and filtering
  on the same state-key pattern used to construct run paths — not by trusting
  a caller-supplied list.
- `scanPiLifecycleLogs` sums `managed_count`/`managed_bytes`/`expired_count`
  across all participating directories into one `PiLifecycleLogFootprint`.
- A single profile-root lock file (`lifecycle-logs.lock`, flock `LOCK_EX` for
  prune, `LOCK_SH` for observe) serializes pruning across every run directory,
  making eviction deterministic across concurrent producers.
- "Active" is now determined by a real, held `flock(LOCK_EX)` on the log
  file's own fd (`piLifecycleLogIsActive` does a non-blocking trylock and
  reads `EWOULDBLOCK` as "held elsewhere") rather than by name comparison —
  crash-safe, since a killed process's fd/lock is released by the kernel and
  the orphaned file becomes ordinary retention input again.

## Adversarial verification (this review, not just re-reading the diff)

1. **Exact replay of the rev1-defeating shape, through the real production
   entry-point pattern** (`ResolvePiClientStatePaths` → `CreatePiStateTree` →
   `openPiSessionLogAt`, which is exactly what `RunPi` and
   `runSharedPiSession` do): three distinct standalone run IDs against
   `max_count=1`. Result: `managed_count=1/1`, `within_policy=true` — the
   bypass is closed at the entry point, not just inside a helper that tests
   call directly.
2. **Corrupted `runs/<hash>` entry** (a regular file where a directory is
   expected, injected outside the API): `observePiLifecycleLogFootprint`
   returns `observation="unknown"` with a populated `Error`, not `absent` and
   not `within_policy=true`. Read failures are not laundered into a healthy
   or empty status.
3. **Concurrency**: 12 goroutines opening independent run IDs and writing
   bursts simultaneously against `max_count=3` under `go test -race`, 5
   repetitions. `managed_count` and `managed_bytes` never exceeded policy in
   any run; no race detected.

All three probes, plus the producer's own new tests
(`TestPiLifecycleLogRetentionBoundsAggregateAcrossProductionClientStatePaths`,
`TestPiLifecycleLogRetentionPreservesEveryActiveRunAndRefusesCountOverflow`,
`TestPiLifecycleLogStatusReportsUnknownWhenAggregateDirectoryCannotBeRead`,
`TestPiLifecycleLogRetentionBoundsSimulatedEightWeekFootprint` — now driven
through distinct run IDs via the production state-resolution path, not one
shared directory), were written and run in this review session, then deleted
before recording this verdict; the working tree matches the candidate tree
exactly (see Identity verification).

## Boundary and defensive checks read and confirmed

- `requiredPositiveInt64`/`requiredPositiveDurationSeconds` reject zero and
  non-positive values; `rejectUnknownFields` refuses an unrecognized key
  (e.g. `default_days`) — no numeric defaults are reachable (AC1).
- `addPiLifecycleBytes` guards `math.MaxInt64` overflow on aggregate byte
  sums; the exhaustion check (`retainedCount > policy.MaxCount ||
  retainedBytes > policy.MaxBytes || reserveBytes > policy.MaxBytes-retainedBytes`)
  is ordered so the subtraction cannot underflow.
- `removePiLifecycleLog` re-opens by name, takes a non-blocking exclusive
  trylock (refusing if the file became active), and revalidates
  device/inode/mode/nlink/size twice (by fd and by path) before `unlinkat` —
  the same identity-swap defense as revision 1, now applied per candidate
  path across every participating directory.
- Symlinks, wrong mode, hard-linked, and unrecognized-name entries are
  reported as `foreign` and are excluded from every delete candidate list, in
  both the deterministic-preservation test and the new corrupt-entry probe.

## Build/test evidence

- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `gofmt -l .`: no output.
- `go test ./internal/infra/... -run 'LifecycleLog|SessionLog' -v -count=1`:
  all pass, including the 8-simulated-week soak
  (`managed_count=5/5 managed_bytes=405/420 expired_count=0
  within_policy=true`).
- `go test ./... -count=1`: one failure,
  `TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC`
  (unrelated RPC-bypass fixture, not among the 17 changed paths). Reran in
  isolation 3x with `-count=3`: passed every time. This matches the
  pre-existing timing-sensitivity flake the producer already logged in
  revision 1's LOGBOOK entry ("Process-Lifecycle Timing Fixtures Flaked Once
  And Passed Focused Reruns") and is not a regression from this change.

## Documentation

`README.md` and `SKILL.md` updates accurately describe the profile-aggregate
lock, the two participating directory shapes, active-file locking, and the
`unknown`-on-read-failure contract; `pi_operator_docs_test.go` pins the exact
new phrases. The new `LOGBOOK.md` entry (1914) correctly states the rev1 root
cause and the rev2 fix.

## Conclusion

Revision 2 closes the exact bypass identified in revision 1 review, holds up
under three independent adversarial probes (production-path replay, corrupted
aggregate input, concurrent race), and composes cleanly with current trunk.
Accepting.
