# TASK-260829-3fozxa Revision 2 Recovery Evidence

## Scope

Recovery run `RUN-260829-80a79c` inspected the revision 2 restart/config-change fix after automatic Change Request construction failed on an unrelated readiness timing test. No repository source was changed by this recovery run.

## Production-path result

The candidate closes revision 1 finding F1 at:

`startUnauthorizedRuntime` -> `startUnauthorizedRuntimeWithDependencies` -> `openSharedRotatingLog` -> `newSharedRotatingLogWriter` -> `sharedFilesystemLogSink.Prune`.

`sharedFilesystemLogSink.Prune` validates every managed retained archive against the current `max_segment_bytes` before count pruning. It refuses oversized, non-regular, linked, or non-mode-0600 managed archives. The production-start regression test creates a 1-byte active segment plus a 10-byte managed archive under `max_segment_bytes=4` and `max_segments=2`, then proves `Cmd.Start` receives zero calls.

## Commands run directly by this recovery run

| Command | Exit | Result |
| --- | ---: | --- |
| Focused log-rotation/config/production-start suite, `-count=1 -v` | 0 | Pass |
| `go test ./... -count=1` | 1 | Unrelated readiness timing failure; not reported as green |
| Exact readiness test rerun 1, `-count=1 -v` | 1 | Same timing family; both subtests failed |
| Exact readiness test rerun 2, `-count=1 -v` | 1 | Same timing family; both subtests failed |
| Exact readiness test rerun 3, `-count=1 -v` | 1 | Same timing family; both subtests failed |
| Exact readiness test rerun 4, `-count=1 -v` | 0 | Pass after bounded contention wait |
| `go test ./... -count=1` retry 2 | 143 | Root package exposed the model-check timing failure; shell was terminated before `internal/infra` completed, so this is not a full-suite result |
| Exact `TestModelCheckProductionEntrypoint`, `-count=1 -v` | 1 | 13/14 subtests passed; owned-process-group deadline subtest missed `tool-pids` |
| Exact owned-process-group deadline subtest, `-count=1 -v` | 1 | Missed `tool-pids` before the bounded deadline |
| Exact owned-process-group deadline subtest retry, `-count=1 -v` | 0 | Pass after host contention declined |
| `go test . -count=1` | 0 | Pass; root package `91.123s` |
| `go test ./internal/infra -count=1` | 0 | Pass; infra package `149.103s` |
| Remaining module packages, `-count=1` | 0 | Attachments and modelharness pass; command package has no tests |
| `go vet ./...` | 0 | Pass |
| `go build ./...` | 0 | Pass |
| `gofmt -l internal/infra runtime_main_darwin_test.go` | 0 | Pass; no output |
| `git diff --check` | 0 | Pass |

The full-suite failure was:

- `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry/refuses_after_owned_runtime_exits`
- observed `runtime_readiness_timeout`; expected `runtime_exited_early`

Exact reruns also sometimes failed to create the readiness-count file before the hard one-second production timeout. The fourth exact rerun passed after a bounded wait. Host load during diagnosis reached `17.79 / 13.31 / 13.23` while unrelated Go test jobs were active. The separate model-check deadline test repeatedly failed to create `tool-pids` before its deadline under the same contention. These are recorded as failed required gates, not as passes.

## Existing evidence accepted without rerunning

The attached `TASK-260829-3fozxa_revision2-implementation-evidence.md` records a narrowing mutant that weakened the production archive-size gate and made `TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart` fail because `Cmd.Start` became reachable. This recovery run inspected that evidence and the restored candidate but did not rerun the source mutant.

## Remaining handoff gate

Every module package passed in bounded sequential commands after contention declined. The exact aggregate `go test ./... -count=1` remains the automatic Change Request handoff gate; the earlier attached revision 2 evidence also records an exit-0 aggregate run against this exact candidate. The recovery run may hand off only if `task-board handoff` reconstructs the candidate and its configured gate exits `0`.
