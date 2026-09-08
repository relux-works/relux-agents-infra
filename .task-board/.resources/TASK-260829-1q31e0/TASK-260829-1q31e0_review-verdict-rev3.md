# TASK-260829-1q31e0 independent review verdict — revision 3

## Verdict

**Changes requested** for `CR-TASK-260829-1q31e0-3` revision 3.

Route: `to-dev`. This is ordinary implementation rework, not a Stop-The-Line boundary.

- Base commit: `91356833949cb6a30958265514fe5852d97eec1b`
- Candidate tree: `ed8acf9f7637eedabe7815e63eddee87086bb335`
- Patch SHA-256: `6237f85847d93643090c3cfcc8a631c24046e2715e15186675d5014e765c3fe2`
- Reviewer run: `RUN-260829-356d6e`
- Repository delta: `present` (17 paths)

The attached patch byte-for-byte matches `git diff --binary <base> <candidate>`. `git diff --check` exits 0. Review and probes ran from an archive of the immutable candidate tree, not from the moving worktree.

## Blocking findings

### P1 — An active launcher log can leave managed accounting and then grow past `max_bytes`

Production `piSessionLog.event` calls `prunePiLifecycleLogsLocked` and writes the record when that returns success (`tools/agents-infra/internal/infra/pi_session_log.go:108-143`). The aggregate scanner classifies mode/link/name before it checks the active lock (`pi_lifecycle_log_posix.go:307-323`). If the already-open, launcher-created log changes from mode `0600` to `0644`, the scanner reclassifies it as foreign, omits its bytes and active lock from managed accounting, and returns healthy policy evidence. `event` does not revalidate the open fd or pathname before writing.

The reviewer production-call-site probe opened a real `piSessionLog`, changed only its mode, and wrote eight records through `piSessionLog.event`. It observed:

```text
size=1440 max_bytes=256 dropped_records=0
managed_count=0 managed_bytes=0 active_count=0
foreign_count=1 foreign_bytes=1440 within_policy=true
```

Thus one automatically-created per-turn lifecycle log exceeded the configured byte bound by 5.6x while diagnostics falsely claimed the managed aggregate was healthy. This is the standard **bypass path around the check** plus **prove, or report unknown** failure. It violates AC 3 and AC 5; it also disproves `LOGBOOK.md:16-17`'s claim that all file-locked active logs are detected.

### P1 — Filename and mode are self-mintable deletion authority for a foreign file

The same scanner treats any matching timestamp/nonce filename plus a mode-`0600`, single-link regular-file shape as launcher-owned. A caller can create that shape without the launcher. The reviewer placed such a foreign file in the canonical log directory and called the real `openPiSessionLogAt` creation path with an expired/count-limited policy. The candidate deleted the foreign file before opening the new log.

This is the standard **forged or self-minted evidence** shape. Filename/mode alone cannot support the unconditional README/SKILL claim that foreign entries are never deleted, so AC 4 is unmet.

## Required rework

1. Bind each open session log to its original fd/path identity and revalidate device, inode, type, mode, link count, and path identity before every reservation/write. If ownership becomes unverifiable, drop the whole record, surface an error/unknown footprint, and do not write.
2. Make aggregate scanning detect a file-locked lifecycle-name entry before permissively reclassifying it as foreign. An active launcher log with damaged ownership shape must make retention/status fail closed; it must not disappear from active/managed bytes while writes continue.
3. Replace self-mintable filename/mode deletion authority with durable launcher provenance, or conservatively preserve matching files whose launcher ownership cannot be established. Add a negative production-path test that mints the current accepted shape and requires preservation/refusal.
4. Add the active-mode/link/identity narrowing test at `piSessionLog.event`, asserting the on-disk file never exceeds `max_bytes`, `DroppedRecords` increments, and status is not healthy.
5. Correct README/SKILL and append a newer LOGBOOK entry so the shipped claims match the fail-closed ownership contract.

## Validation evidence

- Reviewer expected-red probe: exit 1; both tests failed with the two behaviors above. Attached as `TASK-260829-1q31e0_review-negative-ownership-probe-rev3.log`.
- Existing focused lifecycle/config/status tests, uncached: exit 0 in 10.889s. The eight-week test still reports `managed_count=5/5`, `managed_bytes=405/420`, `expired_count=0`, `within_policy=true`; it does not mutate ownership after open. Attached as `TASK-260829-1q31e0_review-focused-existing-rev3.log`.
- Producer CR validation records `go test ./... -count=1` exit 0 and `go vet ./...` exit 0 on the exact candidate. The patch digest was independently reproduced, so those attached full-suite results were accepted as tree-bound positive evidence rather than rerun broadly after the decisive negative failure.
- No live Pi/model process, service, socket, or endpoint was used.
- Reviewer did not modify the candidate or its `LOGBOOK.md`; the probe exists only in an ignored scratch archive. This verdict is the persistent finding, and the producer must add the corrective project logbook entry during rework.
