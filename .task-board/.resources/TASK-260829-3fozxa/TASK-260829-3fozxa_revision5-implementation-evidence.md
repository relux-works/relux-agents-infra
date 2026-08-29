# TASK-260829-3fozxa Revision 5 Implementation Evidence

## Outcome

The independently reviewed revision 4 rotation delta was replayed onto current
trunk `91356833949cb6a30958265514fe5852d97eec1b`. The replay preserves trunk's
persisted `restart_not_before` and `half_open` status/reporting behavior and the
root stdout/stderr capture fix. Revision 4's patch digest was independently
rechecked as
`70e0031bd4ff2fc2bfcf5910cfb71bee13d739b93a16e222f52c325961a46f91`
before replay. All non-logbook paths applied cleanly; `LOGBOOK.md` was merged as
a newest-first append-only union.

Both operator limits remain explicit: `parsePiRuntimeSharing` requires positive
`max_segment_bytes` and `max_segments`; production contains no numeric fallback.
The production path is:

`RunSharedRuntimeBroker -> startUnauthorizedRuntime ->
startUnauthorizedRuntimeWithDependencies -> openSharedRotatingLog ->
newSharedRotatingLogWriter -> sharedFilesystemLogSink.Prune`.

The writer rotates before the first byte beyond the exact cap, retains one
active segment plus at most `max_segments - 1` archives, and deterministically
prunes oldest sequence names. Every managed archive is `Lstat`-validated as a
mode-0600, single-link regular file and must fit the current cap. A lowered cap
therefore fails closed before `Cmd.Start` instead of admitting an oversized
archive from an earlier configuration.

## Acceptance evidence

- Exact boundary: `TestSharedRuntimeLogRotatesBeforeFirstBytePastExactCap`.
- Deterministic oldest-first retention: `TestSharedRuntimeLogPrunesOldestSegmentsDeterministically`.
- Forty-five simulated days with fake clock/sink only: `TestSharedRuntimeLogMultiDayFootprintNeverExceedsConfiguredProduct`; footprint is checked after every day against `max_segment_bytes * max_segments`.
- Missing/zero/overflowing caps: production `RunPi` tests refuse before provider lookup and before cache/runtime-state creation.
- Restart/config-change bypass: `TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart` uses the production filesystem sink plus fake clock and proves `Cmd.Start` receives zero calls.
- No-follow/foreign-file safety: managed archive symlinks refuse and the foreign target remains unchanged.
- Trunk composition: current base, local `main`, and `origin/main` are exact OID `91356833949cb6a30958265514fe5852d97eec1b`; `git rev-list --count HEAD..main` returned `0`.

## Adversarial mutant

The production archive gate was temporarily narrowed from
`stat.Size > maxSegmentBytes` to `stat.Size > maxSegmentBytes*3`. Running
`go test ./internal/infra -run
'^TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart$'
-count=1 -v` exited `1` with
`runtime command start side effect calls=1 want=0`. This is the expected-red
result proving the regression test catches the bypass class. The source was
restored from a task-scoped byte copy, `cmp` succeeded, `git diff --check`
exited `0`, and the exact named test then exited `0`.

## Validation run directly by RUN-260829-959bbe

| Command | Exit | Result |
| --- | ---: | --- |
| Focused rotation/config/restart/status suite, `-count=1 -v` | 0 | Exact-cap, deterministic pruning, 45-day bound, production refusals, restart bypass, no-follow, and trunk-owned recovery status tests pass. |
| Narrowed production archive-size mutant test | 1 | Expected red; bypass reached `Cmd.Start`, so the negative test killed the mutant. |
| Restored production restart-bypass test | 0 | Pass after byte-identical restoration. |
| `go test ./... -count=1` | 0 | Root `87.139s`; `internal/infra` `153.519s`; attachments and modelharness pass. |
| `go vet ./...` | 0 | Pass. |
| `go build ./...` | 0 | Pass for the inspected Darwin/arm64 project target. |
| Initial `gofmt -l internal/infra ../runtime_main_darwin_test.go` | 2 | Operator path error: the file is in the module root, not its parent. This is not reported as a lint pass. |
| Corrected `gofmt -l internal/infra runtime_main_darwin_test.go` | 0 | Empty output; formatting clean. |
| `git diff --check` | 0 | Pass. |

No live model process, service, socket, endpoint, or user-owned harness was
contacted. The implementation is ready for an independent immutable revision 5
review.
