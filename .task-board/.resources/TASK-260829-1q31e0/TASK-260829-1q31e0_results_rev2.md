# TASK-260829-1q31e0 revision 2 developer outcome

## Result

Revision 1's per-run retention bypass is removed on current trunk base `675f77ed63376320ed1213f46f9462a299c0abaf`.

- Explicit `max_count`, `max_bytes`, and `max_age_seconds` remain mandatory and positive; no numeric fallback exists.
- One mode-0600 profile-level retention lock serializes create, record reservation, scan, and prune across the profile `logs/` directory and every exact `runs/<run-key>/logs/` directory.
- Every open lifecycle JSONL holds a file lock. Aggregate pruning detects all active files, refuses a new file when active count/bytes consume the policy, and revalidates device, inode, type, link count, mode, size, path identity, and inactive lock state before unlink.
- Deterministic newest-prefix ordering is global across run directories, with full path as the stable timestamp tie-breaker.
- Foreign names, symlinks, hard links, wrong modes, directories, identity replacements, active files, and unreadable trees are never deleted. Read failures produce `observation="unknown"`, never `absent` or `within_policy=true`.
- `--print-config`, primary-session compose, and managed run reports expose the same profile aggregate: participating directory count, directory/managed/foreign count and bytes, active/expired counts, oldest/newest timestamps, bounds, pruning totals, drops, and errors.
- README, source skill guidance, operator contract tests, and the corrective LOGBOOK entry describe the aggregate ownership boundary.

No live Pi/model process, service, socket, or endpoint was used.

## Negative evidence

The production-shaped test drives the state-selection and log-open seam used by both `RunPi` standalone and `runSharedPiSession`: every simulated turn resolves a fresh `ResolvePiClientStatePaths` run key before `openPiSessionLog`.

A temporary narrowing mutant changed the production scanner back to only `paths.LogsDir`. The exact test failed with exit 1 because status saw `ManagedCount=0` while three run directories retained their logs. The mutant was restored with `apply_patch`; the same test then passed with exit 0. Evidence: `aggregate-narrowing-mutant-01.log`.

Additional negative/refusal coverage proves:

- missing/non-positive policy fields refuse before provider lookup or managed-state mutation;
- a third concurrent active log is refused at `max_count=2` without deleting either active file or a foreign file;
- identity replacement aborts pruning without deleting the replacement, original, active, or foreign files;
- a simulated crash during a multi-delete plan leaves active and foreign files intact;
- an oversized JSONL record is dropped without writing partial bytes;
- an unreadable aggregate directory reports `unknown` and preserves the foreign target.

## Deterministic soak evidence

The eight-week fake-clock test creates a fresh run ID for every simulated turn through the production client-state resolver. Final profile aggregate:

| Metric | Observed | Bound |
| --- | ---: | ---: |
| Simulated duration | 8 weeks | 8 weeks |
| Managed count | 5 | 5 |
| Managed bytes | 405 | 420 |
| Expired count | 0 | 0 |
| Foreign count | 1 preserved | never prune |
| Within policy | true | true |

## Validation

All commands ran directly as standalone processes; no `tee` or status-masking pipeline was used.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run 'TestPi(LifecycleLog|SessionLogEvent|PrintConfigReportsLifecycleLog)' -count=1 -v` | 0 | 9 focused tests; eight-week line recorded in `TASK-260829-1q31e0_retention-tests_rev2.log` |
| narrowed aggregate mutant test | 1 | expected-red; per-run bypass detected |
| restored exact aggregate test | 0 | production scanner restoration verified |
| `go test ./... -count=1` | 0 | root 185.735s; infra 316.986s; attachments/modelharness green |
| `go vet ./...` | 0 | no output |
| focused retention suite with `-race` | 0 | infra 23.666s |
| Linux amd64 compile-only, root + infra | 0 / 0 | artifacts under ignored task temp directory |
| Windows amd64 compile-only, root + infra | 0 / 0 | artifacts under ignored task temp directory |
| focused operator docs test | 0 | 0.580s |
| `gofmt -l <changed Go files>` | 0 | no output |
| `git diff --check` | 0 | no output |

## Replay note

The board-owned revision-1 replay patch applied cleanly to all paths except current-trunk `LOGBOOK.md` and `pi_operator_docs_test.go`, which already contained the landed restart-status work. Those two paths were composed semantically rather than overwritten. The initial full `git apply --check` exit 1 and the exclusion preflight exit 0 are recorded in `tool-failures-01.log`.
