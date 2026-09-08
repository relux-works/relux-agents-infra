# TASK-260829-1q31e0 developer handoff

## Outcome

- Added mandatory per-profile `[agents.pi.profiles.<name>.lifecycle_log_retention]` with explicit positive `max_count`, `max_bytes`, and `max_age_seconds`. Missing, zero, negative, or unknown policy fields fail in `RunPi -> loadCompositeProjectConfig -> parsePiProfile` before provider lookup or managed-state creation.
- Added anchored, deterministic pruning for lifecycle JSONL files. It retains the newest owned-file prefix that fits count, bytes, and age; always preserves the active file; revalidates device, inode, size, regular-file type, exact mode 0600, and link count before unlink; and fsyncs the directory after deletion.
- Defined foreign entries as anything outside the exact launcher-owned filename/type/mode/link shape. Foreign names, symlinks, wrong-mode files, hard links, directories, and identity replacements are observed separately and never pruned.
- Reserved each complete JSONL record before write. A record that cannot fit `max_bytes` is dropped atomically and surfaced in run status instead of being partially written.
- Added `lifecycle_logs` to non-launching Pi plans and managed `PiRunReport`: configured policy/source, directory/managed/foreign count and bytes, expired count, active count, oldest/newest timestamps, `within_policy`, cumulative prune totals, dropped records, and retention errors.
- Updated `README.md`, source `SKILL.md`, operator documentation tests, config fixtures, and `LOGBOOK.md`. No installed runtime or live model was touched.

## Deterministic soak evidence

Fake clock plus task-local temporary filesystem simulated 8 weeks at 3 turns/day:

| Metric | Observed | Bound |
| --- | ---: | ---: |
| Managed files | 5 | 5 |
| Managed bytes | 405B | 420B |
| Expired managed files | 0 | 0 |
| Preserved foreign files | 1 | Never pruned |
| `within_policy` | `true` | `true` |

The same suite proves deterministic newest retention, active age exemption, foreign symlink/wrong-mode preservation, identity-swap refusal, partial-crash safety, and whole-record byte refusal through the production retention call sites.

## Validation and real exit codes

All commands ran directly with no `tee` or background process.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run 'TestPiLifecycleLog|TestPiSessionLogEvent|TestPiPrintConfigReportsLifecycleLog|TestRunPiRequiresExplicitPositiveLifecycleLogRetention' -count=1 -v` | 0 | `.temp/TASK-260829-1q31e0/retention-tests-final.log` |
| Same focused suite with `-race` | 0 | `.temp/TASK-260829-1q31e0/retention-race-final.log` |
| `go test ./internal/infra -count=1` | 0 | `.temp/TASK-260829-1q31e0/go-test-infra-final.log` |
| `go test . -count=1` | 0 | `.temp/TASK-260829-1q31e0/go-test-root-final.log` |
| `go test ./internal/attachments ./internal/modelharness ./cmd/model-harness -count=1` | 0 | `.temp/TASK-260829-1q31e0/go-test-supporting-final.log` |
| `go vet ./...` | 0 | `.temp/TASK-260829-1q31e0/go-vet-all-final.log` |
| `go build ./...` | 0 | `.temp/TASK-260829-1q31e0/go-build-all-final.log` |
| Linux amd64 `go test -c ./internal/infra` | 0 | `.temp/TASK-260829-1q31e0/linux-compile-01.log` |
| Windows amd64 `go test -c ./internal/infra` | 0 | `.temp/TASK-260829-1q31e0/windows-compile-01.log` |
| `git diff --check` | 0 | `.temp/TASK-260829-1q31e0/final-preflight.log` |

One post-hardening aggregate `go test ./... -count=1` exited 1 because `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry/refuses_after_owned_runtime_exits` observed `runtime_readiness_timeout` instead of `runtime_exited_early`; all other packages were green in that run. The exact subtest then passed 10/10 with exit 0, and the standalone `internal/infra` package passed exit 0. An earlier uncached aggregate `go test ./... -count=1` also exited 0. This timing anomaly and its evidence are recorded in `LOGBOOK.md`; it is not reported as a passing aggregate gate.

## Workspace base

- Story branch/worktree selected base: `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`.
- Refreshed `origin/main` matched the selected base exactly; ahead/behind counts were `0/0`.
- Evidence: `.temp/TASK-260829-1q31e0/base-verification.log`.
