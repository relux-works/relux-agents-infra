# TASK-260829-1qh0ud implementation report

## Delivered

- Added strict explicit `resource_pressure_mode = "disabled"|"provider"`; provider mode requires all observation, threshold, hysteresis, eviction, and action fields. No field is defaulted.
- Added a bounded provider observation contract: exact model/schema, strict JSON, 16 KiB maximum body, and 1..30000 ms timeout.
- Added typed independent loaded-model-memory and inference facts plus aggregate `healthy`, `busy`, `pressured`, `draining`, and `unknown` states.
- Bumped the shared wire protocol to v7 and published the versioned `agents-infra.pi.shared-runtime.resource-status.v1` handoff through the production status response and operator JSON.
- Gated production lease acquisition in `sharedBrokerServer.handleConnection -> acquireLease`: observed pressure and unknown evidence refuse a new lease; busy is reported independently; no state is inferred from lease/restart counts.
- Preserved connection-bound leases and the single broker-owned runtime under pressure. Pressure drains only after the final lease release and explicit eviction grace. Recovery at the explicit lower threshold admits against the same runtime PID.
- Added versioned provider/consumer JSON fixtures and deterministic fake-provider tests. No user-owned live model, runtime, endpoint, process, or state was probed or mutated.
- Updated README, source `SKILL.md`, and `LOGBOOK.md`.

## Production negative evidence

- `TestSharedBrokerProductionPressureRefusalPreservesLeaseAndRecoversWithoutDuplicateRuntime` drives the real `sharedBrokerServer.handleConnection -> acquireLease` path, proves pressure refusal, retained ownership, and recovery with the same runtime PID.
- `TestSharedBrokerProductionUnknownResourceObservationRefusesLease` proves a provider read failure remains unknown and cannot acquire.
- `TestParsePiRuntimeResourcePressureRequiresExplicitSafePolicy` narrows thresholds/actions/time bounds and requires every unsafe or incomplete policy to fail parsing.
- `TestSharedRuntimeProviderObservationIsBoundedStrictAndVersioned` rejects wrong schema/model, unknown fields, failed reads, and oversized evidence.
- `TestSharedBrokerPressureDrainPreservesLeaseThenEvictsAfterFinalRelease` proves no pressure eviction while leased and drain only after final release.

## Validation evidence

Green commands (real exit `0`):

- Final focused resource suite: `go test ./internal/infra -run 'Test(...)$' -count=1` — 1.141s.
- Focused production concurrency gate: `go test -race ./internal/infra -run 'Test(SharedBrokerProduction...|SharedBrokerPressure...)$' -count=1` — 2.296s.
- Remaining full infra suite: `go test ./internal/infra -skip '^TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry$' -count=1` — 345.878s.
- Module compile gate: `go test ./... -run '^$' -count=1`.
- `go vet ./...` after final production changes.
- `go build ./...` after final production changes.
- `GOOS=linux GOARCH=amd64 go build ./...` after the common observer change.
- `GOOS=windows GOARCH=amd64 go build ./...` after the common observer change.
- `git diff --check` after final edits.

Red or invalidated commands (reported, not presented as passing):

- First focused resource run exited `1`: the new TOML subtable was inserted before an existing scalar in one fixture. Scalars were reordered and the exact gate then exited `0`.
- One combined formatting/test command exited `2`: repo-relative paths were used from the module directory, so tests did not start. Formatting and the exact test gate were rerun separately; both exited `0`.
- First Windows cross-build exited `1`: the common observer referenced a `!windows` EOF helper. A common strict EOF helper replaced it; Windows, Linux, and host builds then exited `0`.
- `go test ./internal/infra -count=1` exited `1` after 304.617s, and `go test ./... -count=1` exited `1` after the root/attachments packages passed: both failed the pre-existing one-second `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry` timing fixture.
- An isolated `-count=3` run of that readiness test also exited `1` under concurrent host load. During failure, read-only process inspection showed several unrelated Go integration suites from other Story worktrees. No foreign process was stopped. This timing gate is not reported as passing; every other infra test passed in the explicit skip run above.

## Workspace evidence

- Story branch: `task-board/story/STORY-260829-3rs679`.
- Selected protected base and cached `origin/main`: `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`.
- Detailed readiness/base evidence: `.temp/TASK-260829-1qh0ud/tool-readiness-and-base.log`.
