# TASK-260830-tvy8q5 developer outcome

## Implementation

- Added non-launching `agents-infra pi lifecycle status` pagination and `retire-legacy --dry-run|--confirm PLAN_HASH` CLI entry points.
- Added stable full-plan hashing over policy, generations, profile/run/log directory identities, and every candidate identity. Dry-run refuses incomplete scans, unknown evidence, mutation-count overflow, generation overflow, and any plan whose initial or progress odd-generation authority cannot fit the 4096-byte control bound.
- Added exact lowercase confirmation, foreground/retention lock acquisition, odd/even legacy fencing, operation-bound tombstone rename, resumable unlink, and immediate generation/aggregate/directory/descriptor/path revalidation before every mutation.
- Automatic setup, managed launch, and status never retire legacy evidence. Managed launch refuses an odd legacy generation; status health is published only by a fresh complete scan with stable even generations.
- Added deterministic no-sleep hourly, daily, and eight-week retention-plane simulation covering crash backoff/quarantine/half-open recovery, persisted-ledger host reload, stale/renewed leases, corrupt state refusal, backend health loss, and pressure hysteresis.
- Updated README, tool documentation, repository skill contract, and `LOGBOOK.md`.

Production entry points exercised by the negative tests are `runPi -> runPiLifecycleCLI`, `PiLegacyRetirementDryRun`, `PiLegacyRetire -> resumePiLegacyRetirement`, `openPiSessionLog`, `PiLifecycleStatus`, restart-ledger read/write, `sharedBrokerServer.statusSnapshotLocked`, and `classifySharedRuntimeResources`.

## Validation evidence

All commands below were run directly without `tee` or background execution.

- Expected-red pre-implementation focused compile: exit 1 (operator symbols absent).
- Expected-red operator documentation test: exit 1 (required lifecycle contract absent).
- `go test ./internal/infra -run 'TestPiLegacyRetirement|TestPiAutomaticSetupLaunchAndStatusNeverMutateLegacyEvidence|TestPiRetentionPlaneDeterministicEightWeekCrashLeaseReloadPressureSoak' -count=1`: exit 0 on the exact handoff tree.
- `go test . -run 'TestRunPiLifecycle|TestPiOperatorContract|TestReluxAgentsInfraSkill' -count=1`: exit 0.
- `go test ./... -count=1`: exit 0 on the exact handoff tree (`internal/infra` 165.084s).
- `go test -race ./internal/infra -run 'TestPiLegacyRetirement|TestPiAutomaticSetupLaunchAndStatusNeverMutateLegacyEvidence|TestPiRetentionPlaneDeterministicEightWeekCrashLeaseReloadPressureSoak' -count=1`: exit 0.
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- `GOOS=linux GOARCH=amd64 go test -exec=true ./... -run '^$' -count=1`: exit 0 (cross-compile only).
- `GOOS=windows GOARCH=amd64 go test -exec=true ./... -run '^$' -count=1`: exit 0 (cross-compile only).
- `gofmt -l` over every changed Go file: exit 0 with empty output.
- `git diff --check`: exit 0.

## Negative gate strength

- Mutation budget was deliberately narrowed from `max_mutations_per_operation/2` candidates to the raw mutation count. `TestPiLegacyRetirementPlanRefusesNarrowedCandidateBound` then failed with exit 1 because the real dry-run admitted two candidates under a three-mutation budget. Production was restored and the focused tests returned exit 0.
- Immediate file authority was deliberately narrowed to mode/UID at both the held descriptor and descriptor-relative path checks. `TestPiLegacyRetirementPreservesLastGapCandidateSubstitution` then failed with exit 1 because replacement evidence was deleted. Production was restored and the same focused tests returned exit 0.
- A path-only narrowing remained green because the independent held-descriptor ctime/link proof rejected the substitution first; it was not counted as mutant evidence. The test was strengthened to substitute the second candidate after an authorized first mutation, then the combined immediate-authority narrowing above proved the full gate.
- `TestPiLegacyRetirementResumesCrashAfterFencedRename` interrupts after descriptor-relative rename and before progress-generation persistence, proves automatic launch refuses the odd fence, and resumes only with the original exact plan hash.

## Installed parity and no-live-runtime evidence

- A fresh `CGO_ENABLED=0` binary was built from the exact worktree into `.temp/TASK-260830-tvy8q5/parity/bin/agents-infra`: exit 0.
- The first parity command from the repository root failed with exit 127 because that directory has no `go.mod`; the binary was rebuilt from `tools/agents-infra` with exit 0.
- A fake HOME inside the source tree was correctly refused as recursive self-sync (exit 1). A second isolated setup correctly refused the missing bootstrap-owned `~/.local/bin/agents-infra` (exit 1).
- After installing the fresh binary into the isolated bootstrap slot, `setup global --source-dir <exact-worktree>` exited 0 and `verify global` exited 0 at `/tmp/agents-infra-TASK-260830-tvy8q5-parity.WfjMtn/home/.agents`.
- No command used the user's live HOME, installed runtime, Pi executable, provider endpoint, socket, lease, or network service. CLI tests set `PATH=/definitely/no/provider/bin` and configure a nonexistent runtime while exercising status, pagination, dry-run, and confirmation through the real `runPi` entry point.

## Handoff boundary

Developer implementation, tests, docs, static parity, and evidence are ready for independent review. Independent review, Story PR publication/checks, exact-head landing, and live installed-runtime rollout remain orchestrator/reviewer-owned and are intentionally not claimed here.

The earlier developer handoff failed closed only because an obsolete checklist item required independent review and final Story PR merge before the producer could enter `to-review`. The orchestrator removed that owner-incompatible item. All remaining developer-owned checklist items are now satisfied; review and integration remain in their owning phases.

## Bounded handoff recovery — RUN-260830-190f4c

- Inspected the exact current tracked diff and all four untracked candidate files at Story worktree HEAD `40d4e4f8c17387f04fa99cb26585f396e8f07cc0` on branch `task-board/story/STORY-260830-2ywmg2`. The tracked binary diff SHA-256 is `56a3c76dc133e23ec964c2daa6e4c1b211f1f90f847c339e3d3dcc4250cc3c2b`.
- The worktree HEAD is seven commits behind local `main`; the assignment forbids branch switching/rebasing and delegates trunk integration to the orchestrator. These recovery checks therefore attest only the preserved Story candidate at the exact HEAD above plus its uncommitted delta.
- Re-ran `go test ./internal/infra -run 'TestPiLegacyRetirement|TestPiAutomaticSetupLaunchAndStatusNeverMutateLegacyEvidence|TestPiRetentionPlaneDeterministicEightWeekCrashLeaseReloadPressureSoak' -count=1`: exit 0 (`5.074s`).
- Re-ran `go test . -run 'TestRunPiLifecycle|TestPiOperatorContract|TestReluxAgentsInfraSkill' -count=1`: exit 0 (`0.726s`).
- Re-ran `gofmt -l` over every changed Go file: exit 0 with empty output.
- Re-ran `git diff --check`: exit 0 with empty output.
- Recovery contacted no Pi/runtime/model/process/service/socket/endpoint/network service and performed no setup/install or user-HOME runtime operation. No candidate source, test, documentation, or logbook file changed during recovery.

## Revision 2 external-absence authority repair — RUN-260830-70d0cd

- Exact-diff inspection found the reviewer-proven bypass still present in `PiLegacyRetire -> resumePiLegacyRetirement`: source+tombstone absence advanced retirement even when the odd generation had not persisted `candidate_renamed=true`. The stale recovery-only instruction was not used to hand off a known-red candidate.
- The resume path now treats that unexplained dual absence as typed `lifecycle_log_evidence_unknown` and preserves the odd generation. A crash after unlink remains resumable only when the odd generation persisted operation-bound rename authority before unlink.
- Added `TestPiLegacyRetirementRefusesExternalAbsenceWithoutPersistedRenameAuthority` and `TestPiLegacyRetirementResumesCrashAfterUnlinkWithPersistedRenameAuthority`, both through the production `PiLegacyRetire -> resumePiLegacyRetirement` call site. Updated `LOGBOOK.md` with the root cause, fix, and evidence.
- Deliberately narrowed the new guard so the external-absence branch was admitted. The uncached production-entry negative test failed with exit 1 and showed false `RetiredCount=1`, `WithinPolicy=true`, and `SoakReady=true`. Restoring the guard made the exact test exit 0.
- `go test ./internal/infra -run '^TestPiLegacyRetirementRefusesExternalAbsenceWithoutPersistedRenameAuthority$' -count=1`: exit 0 after restoring production.
- `go test ./internal/infra -run 'TestPiLegacyRetirement|TestPiAutomaticSetupLaunchAndStatusNeverMutateLegacyEvidence|TestPiRetentionPlaneDeterministicEightWeekCrashLeaseReloadPressureSoak' -count=1`: exit 0 (`4.931s`).
- `go test . -run 'TestRunPiLifecycle|TestPiOperatorContract|TestReluxAgentsInfraSkill' -count=1`: exit 0 (`0.529s`).
- `go test -race ./internal/infra -run 'TestPiLegacyRetirement(RefusesExternalAbsenceWithoutPersistedRenameAuthority|ResumesCrashAfterUnlinkWithPersistedRenameAuthority)$' -count=1`: exit 0 (`3.969s`).
- `go build ./...`: exit 0.
- `GOOS=linux GOARCH=amd64 go test -exec=true ./internal/infra -run '^$' -count=1`: exit 0 (compile-only).
- `GOOS=windows GOARCH=amd64 go test -exec=true ./... -run '^$' -count=1`: exit 0 (compile-only).
- `gofmt -l` over all changed Go files: exit 0 with empty output; `git diff --check`: exit 0 with empty output.
- The revision used only deterministic temporary filesystem fixtures and fake clocks. It contacted no live Pi/runtime/model/process/service/socket/endpoint, performed no setup/install, and used no user-HOME runtime state.

## Revision 3 tombstone-retry durability repair — RUN-260830-61e01d

- Exact-diff inspection against the attached revision-2 reviewer verdict confirmed the tombstone-only retry still unlinked without persisting equivalent rename authority. The recovery precondition was stale relative to that blocking verdict, so the known-red candidate was not handed off unchanged.
- Added `TestPiLegacyRetirementResumesTwoProgressWindowCrashes` through `PiLegacyRetire -> resumePiLegacyRetirement`. Before the repair, the uncached test exited 1 because the odd generation retained `candidate_renamed=false` after the second crash.
- The retry now persists `candidate_renamed=true` after proving the exact operation-bound tombstone and before unlink. The revision-2 external dual-absence negative remains typed unknown.
- Exact external-absence/two-crash pair: exit 0 (`0.748s`); focused legacy/automatic/eight-week suite: exit 0 (`5.757s`); root CLI/docs suite: exit 0 (`1.249s`); focused race: exit 0 (`2.120s`).
- `go test ./... -count=1`: exit 0 (`internal/infra` `220.160s`); `go vet ./...`, `go build ./...`, Darwin/Linux/Windows compile-only gates, `gofmt -l`, and `git diff --check`: exit 0.
- Final tracked binary diff SHA-256: `33397162661bbc00bc4b77b1d70c25188035f89e7b3c5e7ae3cab425fdea6f55`; exact untracked file digests are recorded in `TASK-260830-tvy8q5_handoff-recovery-rev3.md`.
- No live Pi/runtime/model/process/service/socket/endpoint/network service, setup/install flow, or live user-HOME runtime state was contacted.

## Revision 5 pre-retirement health repair — RUN-260830-9a8820

- Reconciled the stale recovery note with the latest reviewer verdict and found that the preserved revision-4 candidate still published `within_policy=true` while legacy evidence remained. It was not handed off in that known-red state.
- `PiLifecycleStatus` now requires zero legacy and foreign evidence for `within_policy`; `soak_ready` inherits the same fresh complete-scan gate.
- `TestRunPiLifecycleOperatorIsNonLaunchingAndProjectsExactPlan` drives `runPi -> runPiLifecycleCLI`, requires both health flags false before retirement, and requires health only after the exact confirmed plan retires the candidate.
- The test-first run failed with exit 1 against the permissive expression. A narrowed mutant that retained foreign refusal but removed only the legacy-count condition also failed with exit 1, reporting `LegacyCount=1`, `WithinPolicy=true`, and `SoakReady=true`. Production was restored and the exact test exited 0.
- Focused legacy/crash/resume/automatic-non-mutation/eight-week soak tests exited 0 (`5.698s`); focused lifecycle CLI/docs tests exited 0 (`0.636s`); native `go build ./...`, changed-file `gofmt -l`, and `git diff --check` exited 0 with empty formatter/diff-check output.
- The bounded recovery relied on already attached revision-4 evidence for full repository, race, vet, Linux, and Windows gates; those commands were not rerun after the platform-neutral revision-5 expression/test/logbook delta.
- Exact hashes and full bounded evidence are attached as `TASK-260830-tvy8q5_handoff-recovery-rev5.md`. No live runtime, process, service, socket, endpoint, setup/install flow, network service, or live user-HOME runtime/config state was contacted.
