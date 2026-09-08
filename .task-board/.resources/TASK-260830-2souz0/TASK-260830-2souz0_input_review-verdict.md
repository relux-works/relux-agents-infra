# TASK-260830-1ugbb1 review verdict — Change Request revision 2

## Verdict

**Changes requested; route to `to-dev`.** Revision 2 fixes the three revision-1 blockers, but cannot be accepted because three new deterministic production-entry probes defeat required strict recovery and status attestation boundaries.

Reviewed Change Request: `CR-TASK-260830-1ugbb1-2`, revision `2`.

- Base OID: `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`
- Candidate tree OID: `825784782e2effe8c555545301d7a9a7aadebbdf`
- Patch SHA-256 independently verified: `5c162f2050c288039651599e5f8c7fc252e1455412e03d145a7441d5faece553`
- Spawn goal: none; this reviewer run is not goal-bound.
- Repository delta: present.

## Blocking findings

### 1. Odd-close recovery accepts forged close-looking evidence

Production call site: `openPiSessionLog` -> `recoverPiLifecycle` -> `recoverPiLifecycleClose` in `tools/agents-infra/internal/infra/pi_session_log.go:1063`.

When `.record.json.tmp` is absent, lines 1078-1082 return success for any strict record whose committed counters match the odd generation. The already-published branch never proves `record.ClosedAt == generation.StartedAt`. A well-formed but unrelated timestamp therefore closes the odd generation, admits a new entry, and increments recovery as if the exact operation completed.

The repository test at `pi_lifecycle_log_test.go:308` currently encodes this false-positive behavior: its `after-record` branch writes a fixed `2026-01-01T00:00:00Z` instead of the returned odd operation's `StartedAt` and expects recovery to succeed.

Negative shape: **forged or self-minted evidence**.

Reviewer probe: `TestReviewerForgedCloseLookingEvidenceIsRefused`.

Observed forbidden result:

```text
forged close-looking record admitted by production recovery: next=true err=<nil>
```

Required rework: distinguish the exact pre-publication record (`ClosedAt == ""`) from the exact completed record (`ClosedAt == generation.StartedAt`); reject and preserve every other value under the same odd generation. Correct the positive crash test and add the forged-close negative.

### 2. Delete recovery deletes a tombstone outside strict filesystem authority

Production call site: `openPiSessionLog` -> `recoverPiLifecycle` -> `recoverPiLifecycleDelete` in `pi_session_log.go:1104`.

The aggregate scan skips the named recovery tombstone. `recoverPiLifecycleDelete` then opens the directory and unlinks recognized child names without validating/revalidating the tombstone's exact mode, trusted effective UID, fd/path device+inode, or the children's regular-file/mode/link/UID/device/inode identities. A tombstone narrowed from mode `0700` to `0755` is fully deleted and new work is admitted.

Negative shape: **bypass path around the check**. Normal managed-envelope scanning validates strict authority; the delete-recovery path bypasses it before mutation.

Reviewer probe: `TestReviewerDeleteRecoveryPreservesNarrowedTombstoneAuthority`.

Observed forbidden result:

```text
narrowed tombstone authority was deleted and new work admitted: next=true err=<nil>
```

Required rework: carry and prove the recovery identity required by the odd delete operation; validate/revalidate the tombstone and every remaining child immediately before each bounded unlink. Preserve mismatched, linked, symlinked, wrong-mode, wrong-UID, replaced, or unreadable evidence and leave generation odd with stable typed unknown. Add directory-narrowing and child-substitution production-recovery negatives.

### 3. Status publishes health for a malformed even generation carrying residual operation authority

Production call site: `PiLifecycleStatus` -> `readPiLifecycleGenerationPair` -> `validatePiLifecycleGeneration` in `pi_session_log.go:1384`.

The even-generation branch rejects non-empty `OperationID`, `OperationKind`, `EntryName`, and `StagingName`, but accepts non-empty `StartedAt` and non-zero `CommittedBefore`, `RecordsBefore`, and `AppendBytes`. That is not a strict completed generation record. Status nevertheless reports a fresh complete scan with `UnknownCount:0`, `WithinPolicy:true`, and `SoakReady:true`.

Negative shape: **forged or self-minted evidence** plus failure to report an unproven property as unknown.

Reviewer probe: `TestReviewerEvenGenerationWithResidualOperationAuthorityIsUnknown`.

Observed forbidden result:

```text
AggregateGeneration:4 ScanComplete:true UnknownCount:0 WithinPolicy:true SoakReady:true err=<nil>
```

Required rework: make generation validation strict by state and operation kind. Even records must reject every residual operation-only field and invalid counters. Odd records must validate their timestamp, operation kind, name grammar, and exact kind-specific field contract. Add status and next-writer negatives for malformed even and odd records.

## Revision-1 rework verified

The following candidate tests passed uncached against the exact revision-2 tree:

- aggregate root and `entries/` status authority narrowing;
- atomic append-record temp recovery;
- atomic odd/even generation temp recovery;
- production-issued final-page continuation health refusal;
- legacy-directory mutation invalidation;
- run-log continuation advancement;
- policy/generation continuation invalidation.

Command:

```bash
go test -count=1 ./internal/infra -run '^(TestPiLifecycleStatusRejectsNarrowedAggregateDirectoryAuthority|TestPiLifecycleOddAppendRecoversAtomicRecordTempPhase|TestPiLifecycleRecoversAtomicGenerationTempPhases|TestPiLifecycleFinalContinuationPageCannotPublishHealth|TestPiLifecycleLegacyContinuationRejectsDirectoryMutation|TestPiLifecycleRunLogContinuationAdvancesThroughProductionScanner|TestPiLifecycleContinuationRejectsPolicyAndGenerationChange)$' -v
```

Result: exit `0`, package time `5.148s`.

## Reviewer validation and provenance

- Host/toolchain readiness: `go version go1.25.5 darwin/arm64`; `GOOS=darwin`, `GOARCH=arm64`.
- Three reviewer negatives ran together with `-count=1`: exit `1`, package time `1.322s`; all three failed because the forbidden state was admitted.
- Probe source: `TASK-260830-1ugbb1_reviewer-negative-probes_rev2.go` outcome resource.
- Producer CR validation log independently inspected: uncached `go test ./... -count=1` passed in `294.777s` for `internal/infra` and `go vet ./...` passed. Those positive gates do not override the three deterministic boundary failures.
- Original implementation-start evidence records all required refs at `5c9b4e4`. At review time, selected Story base and workspace `HEAD` remain `5c9b4e4`, while a fresh `git fetch origin main` and `git ls-remote origin refs/heads/main` resolve current protected main to descendant `3295c7da7151de128f176cf7560a57d54c8f6c0d`. This later advance is not a rejection reason; integration must apply the normal current-trunk overlap/revalidation contract.
- `git diff --stat 5c9b4e4... 8257847...` reports the expected 20-path, 3,686-insertion candidate.
- No live model, runtime, daemon, service, endpoint, or socket was contacted.
- Full/race/soak/cross-compile gates were not redundantly rerun after deterministic blockers. Producer evidence remains useful but cannot establish acceptance.

## Rework handoff

Repair all three boundaries in the single lifecycle authority, land equivalent or stronger production-entry negatives, and update `LOGBOOK.md` with these recovery/status root causes. Then rerun the required focused race, fresh-run soak, uncached package splits, vet, format/diff, Darwin build, and Linux/Windows compile gates before publishing revision 3.
