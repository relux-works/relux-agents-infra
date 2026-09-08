# TASK-260829-1q31e0 revision 4 developer outcome

## Result

Revision 4 closes both revision-3 ownership blockers and adds a concurrently
atomic profile-lock creation protocol discovered during the new contention
test. No live Pi/model process, service, socket, or endpoint was used.

## Implementation

- New lifecycle files are created behind a launcher-owned hard-link provenance
  entry in the profile-private mode-0700 `lifecycle-log-ownership/` namespace.
- Aggregate pruning requires exact lifecycle name, mode 0600, regular-file
  type, link count two, and matching device/inode identity through provenance.
  Filename/mode-only and legacy shapes remain foreign and are preserved.
- Scanner activity detection happens before permissive foreign classification;
  damaged locked/active evidence returns `unknown`.
- `piSessionLog.event` revalidates its original fd, public path, and provenance
  identity/type/mode/link count before reservation and again immediately before
  write. A failed proof drops the complete record and preserves `unknown` in
  the run report after close.
- Shared profile-lock creation uses open-existing, then `O_CREAT|O_EXCL`, then
  one `EEXIST` loser open before `flock`. This removed the macOS concurrent
  `openat(O_CREAT|O_NOFOLLOW)` ENOENT observed under contention.
- README, SKILL, and LOGBOOK now describe the shipped ownership and diagnostic
  contract.

## Negative evidence

- Pre-fix production-entry tests: exit 1. Active mode mutation grew the log to
  1048 bytes with `max_bytes=256`, `dropped_records=0`, while the scanner
  reported it foreign/healthy; a self-minted lifecycle name/mode shape was
  deleted by `openPiSessionLogAt`.
- Final production-entry tests mutate active mode, link count, and public path
  identity; all eight attempted records are dropped, no file grows, and status
  is `unknown`.
- A deterministic post-reservation race displaces the public path before write;
  the second fd/path/provenance validation refuses the record and both inodes
  remain empty.
- Twelve simultaneous production-shaped run IDs against `max_count=4` admit
  exactly four active logs, refuse the rest with typed exhaustion, preserve a
  self-minted foreign lifecycle shape, and remain within aggregate bounds.
- The initial contention implementation exposed a real macOS lock-creation
  race: full suite and focused `-count=50` exited 1 with lock-open ENOENT. After
  the atomic creation fix, the same focused stress passes 50/50.

## Final validation

All commands ran directly as foreground processes without `tee`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run 'TestPi(LifecycleLog|SessionLog|PrintConfig)' -count=1 -v` | 0 | `rev4-focused-tests.log` |
| `go test -race ./internal/infra -run 'TestPi(LifecycleLog|SessionLog|PrintConfig)' -count=1` | 0 | `rev4-focused-race.log` |
| `go test ./internal/infra -run '^TestPiLifecycleLogRetentionSerializesConcurrentProductionRunAdmission$' -count=50` | 0 | direct run; 50/50 |
| `go test ./... -count=1` | 0 | `rev4-full-go-test.log`; root 90.853s, infra 165.013s |
| `go vet ./...` | 0 | `rev4-go-vet.log` |
| `go build ./...` | 0 | `rev4-go-build.log` |
| explicit changed-file `gofmt -l` emptiness gate | 0 | `rev4-gofmt.log` |
| `git diff --check` | 0 | no output |

Eight simulated weeks end at `managed_count=5/5`,
`managed_bytes=405/420`, `expired_count=0`, `foreign_count=1`,
`within_policy=true`.

One earlier full-suite run exited 1 because production code accidentally called
the test-only `piErrorCode`; a later run exited 1 on the newly exposed
concurrent lock-creation ENOENT. Both failures were corrected and rerun; they
are not reported as passes.

## Base and composition handoff

- Evidence commit/base: `91356833949cb6a30958265514fe5852d97eec1b`.
- At final freshness check local/main and origin/main were
  `6d051f5` (three commits ahead) after shared runtime log rotation merged.
- Overlap exists in `README.md`, `LOGBOOK.md`, `pi_config.go`, and `pi_test.go`.
  The assigned Story workspace explicitly forbids developer rebase/merge;
  independent review/orchestrator integration must verify semantic composition
  with that newer trunk.

