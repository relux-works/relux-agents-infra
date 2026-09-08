# TASK-260830-1ugbb1 developer rework outcome

## Handoff

Revision-1 reviewer findings are addressed and the candidate is ready for review.
No live model, runtime, daemon, service, endpoint, or socket was contacted.

## Base and current-main authority

- The original implementation started only after fetched `origin/main`, GitHub `refs/heads/main`, selected Story base, workspace `HEAD`, local `main`, and `FETCH_HEAD` all equaled protected post-pressure commit `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`; revision-1 board evidence preserves that preflight.
- During this rework, `git fetch origin main` and independent `git ls-remote origin refs/heads/main` both resolved current protected main to descendant `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- The managed Story base and workspace `HEAD` remain `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`. No switch, rebase, merge, or unrelated trunk absorption occurred.
- Exact current-main overlap with the task's 18-path candidate is only `README.md`; lifecycle, pressure, launch, and test sources do not overlap. Orchestrator integration must reconcile that documentation overlap and rerun affected gates.

## Reviewer blockers addressed

1. Lock-free `PiLifecycleStatus` now proves the aggregate root and `entries/` through held descriptors, exact mode/trusted-effective-UID checks, fd/path device+inode equality, and unchanged final authority after both generation rereads. Narrowing either directory to mode `0755` returns typed unknown and cannot publish health.
2. Status pagination now uses production-issued, phase-specific directory cookies for aggregate entries, canonical profile legacy logs, `runs/`, and per-run logs. Tokens bind project/profile, policy digest, both generations, root identity/change time, current directory identity/change time, and parent run cursor where needed. Every page reports page-local counters and lower-bound health refusal; directory, policy, or generation changes invalidate the token. A final continuation page remains `scan_complete=false`, `within_policy=false`, and `soak_ready=false`.
3. Atomic control publication uses strict fixed temp names. Generation temps must be the exact next odd operation or exact current-operation completion. Append/close record temps must match the odd operation's entry, counters, bytes, identity, and close time. Exact temps converge under the same odd generation; ambiguous evidence remains preserved and unknown. Recovery counters now survive subsequent operations.

Dead revision-1 status/paging implementations were removed so one production lifecycle authority owns the decision.

## New adversarial production evidence

- `TestPiLifecycleStatusRejectsNarrowedAggregateDirectoryAuthority`
- `TestPiLifecycleOddAppendRecoversAtomicRecordTempPhase`
- `TestPiLifecycleRecoversAtomicGenerationTempPhases`
- `TestPiLifecycleFinalContinuationPageCannotPublishHealth` now obtains and advances only production-issued tokens; the self-minted token shape was removed.
- `TestPiLifecycleLegacyContinuationRejectsDirectoryMutation`
- `TestPiLifecycleRunLogContinuationAdvancesThroughProductionScanner`
- `TestPiLifecycleContinuationRejectsPolicyAndGenerationChange`

These cover the real `PiLifecycleStatus`, `openPiSessionLog`, generation publication, append recovery, and next-writer recovery call sites.

## Current-source green validation

| Command | Exit | Result |
| --- | ---: | --- |
| Focused lifecycle suite after the final code delta | 0 | `9.943s` |
| Focused lifecycle + accepted pressure production race slice | 0 | `3.038s` |
| `go test -count=1 ./internal/infra -skip '^TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry$'` | 0 | `241.587s` |
| Exact skipped timing fixture, current source | 0 | `2.924s` |
| `go test -count=1 .` | 0 | `187.961s` |
| `go test -count=1 ./internal/attachments` | 0 | `2.600s` |
| `go test -count=1 ./internal/modelharness` | 0 | `1.490s` |
| `go vet ./...` | 0 | clean |
| Darwin `go build ./...` | 0 | clean |
| Linux/amd64 `go build ./...` | 0 | clean |
| Windows/amd64 `go build ./...` | 0 | clean |
| `gofmt -l .` | 0 | empty output |
| `git diff --check` | 0 | clean |

The uncached package split covers every test while keeping each headless process below the command bound.

## Preserved red evidence

- An earlier unfiltered `go test -count=1 ./internal/infra` exited `1` after `282.885s`: the known host-timing fixture observed `runtime_readiness_timeout` instead of `runtime_exited_early`.
- Its immediate isolated `-count=3` rerun also exited `1` after `16.987s`, including two missing short-lived readiness marker observations.
- After host load subsided, the exact fixture on current source exited `0` in `2.924s`; the rest of the current package independently exited `0` in `241.587s`.
- The red runs remain failures and are not relabeled as passing.

## Persistent decisions

`LOGBOOK.md` records the filesystem-authority bypass, operation-bound temp recovery, real directory-cookie pagination, and the protected-main advance/current-main overlap decision.
