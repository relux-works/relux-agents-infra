# TASK-260829-3fozxa Implementation Evidence

## Result

- Added required `runtime.sharing.max_segment_bytes` and `runtime.sharing.max_segments`; parsing rejects missing, zero, negative, non-integer, and unknown policy members with no numeric fallback.
- Wired `startUnauthorizedRuntime` to one synchronized rotating writer for runtime stdout and stderr.
- Split writes before the first byte beyond the exact segment cap.
- Kept at most `max_segments` total active/archived segments and pruned archives by monotonic creation sequence, oldest first.
- Kept the writer alive until `command.Wait()` because a custom `io.Writer` makes `os/exec` use a parent-owned copy pipe.
- Refused a pre-existing active segment larger than the configured cap rather than discarding it or claiming a false bound.
- Updated all explicit shared-runtime fixtures and the operator README example.

## Acceptance Evidence

- Production config gate: `TestRunPiRejectsMissingSharedRuntimeLogRotationPolicy` drives `RunPi -> loadCompositeProjectConfig -> parsePiRuntimeSharing`; absent caps are refused before provider lookup and before runtime-state creation.
- Production writer call site: `startUnauthorizedRuntime -> openSharedRotatingLog -> newSharedRotatingLogWriter`.
- Exact cap: `TestSharedRuntimeLogRotatesBeforeFirstBytePastExactCap` retains `abcd`, `efgh`, `i` as 4/4/1-byte segments for one cross-boundary write.
- Deterministic pruning: `TestSharedRuntimeLogPrunesOldestSegmentsDeterministically` retains `ghi`, `j` after four logical segments with `max_segments=2`.
- Multi-day bound: `TestSharedRuntimeLogMultiDayFootprintNeverExceedsConfiguredProduct` simulates 45 days with a fake clock and fake sink only; after every day every segment is at most 127 bytes and total output is at most `127 * 5` bytes. No filesystem or sleep is used.
- Negative/error paths cover absent numeric policy, missing dependencies, sink open/rotate/prune failures, pre-existing oversized segments, and writes after close.

## Validation

| Command | Exit | Result |
| --- | ---: | --- |
| `git fetch origin main` | 0 | `HEAD`, protected base, `origin/main`, and `FETCH_HEAD` all resolved to `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`; ahead/behind `0/0`. |
| `go test ./internal/infra -run 'TestParsePiRuntimeSharingIsStrictAndOptIn\|TestSharedRuntime' -count=1` (baseline) | 0 | Relevant pre-change baseline green. |
| `go test ./internal/infra -count=1` (intermediate) | 1 | Expected migration red: existing shared-profile fixtures lacked newly required `max_segment_bytes`; not reported as a passing gate. |
| `go test ./internal/infra -run 'Test(ConnectAndAttestSharedRuntime\|SharedRuntime\|RunPiStandalone\|BuildPiStandalone)' -count=1` | 0 | Previously red shared-runtime families green after explicit fixture migration. |
| `go test ./internal/infra -run 'TestSharedRuntimeLog' -count=1 -coverprofile=../../.temp/TASK-260829-3fozxa/rotation-cover.out` | 0 | Rotation tests green; core function coverage: constructor 88.9%, Write 91.3%, rotate 90.9%, Close 100%. |
| `go test ./... -count=1` (final rerun) | 0 | All Go packages green; `internal/infra` 180.340s. |
| `go vet ./...` | 0 | Canonical Go lint/vet clean. |
| `go build ./...` | 0 | Current macOS project build green. |
| `git diff --check` | 0 | No whitespace errors. |

## Persistent Finding

Recorded the `os/exec` custom-writer ownership invariant and the wait-before-close decision in `LOGBOOK.md` under `2026-08-29 1732`.
