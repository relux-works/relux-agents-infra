# TASK-260829-1q31e0 revision 3 developer outcome

## Result

Revision 2's independently validated profile-aggregate lifecycle-log retention was replayed onto refreshed current trunk `91356833949cb6a30958265514fe5852d97eec1b`.

- Explicit `max_count`, `max_bytes`, and `max_age_seconds` remain mandatory and positive; no numeric fallback exists.
- One mode-0600 profile lock serializes lifecycle create, record reservation, scan, and deterministic prune across canonical `logs/` plus every exact `runs/<run-key>/logs/` tree.
- Every open lifecycle file holds its own lock. Active files are detected across run directories, exempted from age pruning, and counted against byte/count admission.
- Only exact launcher-owned mode-0600 single-link regular files are candidates. Foreign names, symlinks, hard links, wrong modes, directories, active files, identity replacements, and unreadable subtrees are never deleted.
- Read failures report `observation="unknown"` and never become healthy absence.
- `--print-config`, primary-session compose, and run reports expose the same aggregate footprint, policy, participating-directory count, active/expired facts, pruning totals, drops, and errors.
- The refreshed-main `capturePipe` deadlock repair remains intact. `LOGBOOK.md` preserves both entries in newest-first order, and `main_test.go` composes the lifecycle table with the concurrent capture implementation.

No live Pi/model process, service, socket, or endpoint was used.

## Negative evidence

Production call sites `RunPi` and `runSharedPiSession` both resolve fresh run state through `ResolvePiClientStatePaths` and open logs through `openPiSessionLog`.

A temporary narrowing mutant made `openPiLifecycleLogDirectories` skip every valid `runs/<run-key>/logs/` directory. The production-shaped test `TestPiLifecycleLogRetentionBoundsAggregateAcrossProductionClientStatePaths` failed with exit 1, observing only one participating directory and zero managed files while the profile policy was `max_count=1`, `max_bytes=256`, `max_age_seconds=60`. The source was restored, compared byte-for-byte with its pre-mutant backup (exit 0), and the same test then passed with exit 0.

Additional passing refusal tests cover absent and non-positive policy fields, active-count overflow, foreign and active preservation, identity replacement, partial-crash deletion, atomic whole-record byte refusal, and unreadable aggregate status.

## Deterministic soak

The fake-clock test creates a fresh run ID for every turn over eight simulated weeks.

| Metric | Observed | Bound |
| --- | ---: | ---: |
| Managed count | 5 | 5 |
| Managed bytes | 405 | 420 |
| Expired count | 0 | 0 |
| Foreign count | 1 preserved | never prune |
| Within policy | true | true |

## Validation

Every gate ran as a direct standalone process without `tee` or status-masking pipelines.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused lifecycle/config/status suite, uncached and verbose | 0 | 10 tests; eight-week metrics above |
| Narrowed aggregate production-scanner mutant | 1 | Expected-red bypass detection |
| Byte-for-byte restoration check | 0 | Mutated source equals pre-mutant backup |
| Restored aggregate production-path test | 0 | Scanner restoration verified |
| `go test ./... -count=1` | 0 | root `88.854s`; infra `155.917s`; attachments/modelharness green |
| Focused lifecycle/config/status suite with `-race` | 0 | infra `15.572s` |
| `go vet ./...` | 0 | No output |
| `go build ./...` | 0 | No output |
| Linux amd64 root compile-only | 0 | Artifact in ignored task temp directory |
| Linux amd64 infra compile-only | 0 | Artifact in ignored task temp directory |
| Windows amd64 root compile-only | 0 | Artifact in ignored task temp directory |
| Windows amd64 infra compile-only | 0 | Artifact in ignored task temp directory |
| Focused operator documentation contract tests | 0 | root package `0.383s` |
| `gofmt -l` on all changed Go files | 0 | No files listed |
| `git diff --check` | 0 | No output |
| Reverse-check immutable rev2 patch excluding composed `LOGBOOK.md` | 0 | All other candidate changes reproduce exactly |

One preliminary focused-test invocation failed before Go started because the ignored log destination used `../../../.temp` instead of `../../.temp`; zsh returned exit 1. The corrected invocation is the focused exit-0 row above and its raw output is attached separately.
