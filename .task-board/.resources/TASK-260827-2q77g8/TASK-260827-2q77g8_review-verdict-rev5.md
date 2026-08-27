# TASK-260827-2q77g8 review verdict — revision 5

## Verdict

**Changes requested → `to-dev`.** Revision 5 closes the exact F1d-R4 long-prompt bypass, but a narrowed production retention input still reaches `.rebuilt` while 255,724,192 bytes of the condemned model remain active. This is ordinary implementation rework, not a Stop-The-Line boundary: the stated MLX limitation is honest, and the permitted remedy is to abandon conservatively whenever release cannot be proved.

Reviewed Change Request: `CR-TASK-260827-2q77g8-5`, revision `5`, candidate tree `24a6144a9f0968740a87e3cb7cade87ea140a3cb` over base `08beb4052faa851e686705677e24a20c2397ad87`. The supplied patch SHA-256 independently matched `61c96d48c800d8ae083a88a057abd75e47098d623761ded2c8ec14d6b37f6c68`; all 15 candidate paths matched the review worktree and `git diff --check` was clean.

## Round-5 brief results

- **F1d-R4 reproduced and closed.** I reran the real Release binary, real model-harness, real 0.5B model, real HTTP route, and the maintained 6,000-word request. The request-local release again satisfied the old proxy (`returned_bytes=608,909,584 B >= weight_footprint_bytes=262,361,760 B`) while every weight remained active (`active_bytes=262,361,760 B`). Revision 5 failed closed: no completed-rebuild event, `shared_cache_rebuilds=0`, `shared_cache_rebuilds_abandoned=1`, `shared_cache_rebuild_pending=true`, and `/health=503`.
- **The MLX limit is honest.** Its allocator counters are process-global and cannot attribute an individual byte. Conservative abandonment is the correct engineering answer, and false abandonment is an accepted cost under the review brief. The finding below is not the existence of that limit; it is that revision 5 still accepts an unattributable residue below one full footprint.
- **F1c-R3 remains closed.** The below-container retention phase observed `container_deallocated=true`, `active_bytes=262,361,760 B`, and only `39,391,612 B` returned; it abandoned, left the pool pending, emitted no completed-rebuild event, and kept `/health=503`.
- **Outer-container timeout remains closed.** The retained-container phase held `262,361,760 B`, abandoned after the bounded wait, kept `shared_cache_rebuilds=0` and pending=true, re-announced the supervision marker, and kept `/health=503`.
- **TASK-260827-2h39ya remains intact.** The independent dead-generation smoke passed all 45 checks, including buffered and streaming 503, supervised replacement, benign-negative, and over-broad-classifier negative paths.

## Blocking finding

### F1e-R5 — a strict subset of copied weight arrays bypasses both release clauses

Negative shape: **bypass path around the check**, demonstrated by a **narrowed production input** rather than deleting the gate.

`WeightOwnerRegistry` tracks `Module` objects weakly, while the candidate itself documents and exercises the fact that copied `MLXArray` values keep weight buffers alive after every `Module` dies (`BatchLedgerStore.swift:503-513`, `Main.swift:138-142`). The absolute residue gate then accepts every value strictly below one full load footprint (`GenerationBatchRecovery.swift:366-373`). The candidate unit test makes that admission explicit: with identical byte evidence it expects `weightsReleased` to return true as soon as `liveWeightOwners` changes from 1 to 0, even with half the footprint still active (`GenerationBatchRecoveryTests.swift:785-803`); it also accepts `footprint - 1` (`:778-782`).

I attacked the composed production path in a task-scoped COW copy, leaving the candidate checkout untouched. The only mutation narrowed the existing array-retention seam from all flattened parameter arrays to the largest half by `nbytes`. The real model has 628 tensors totalling 261,171,200 bytes; that strict 314-array subset totals 254,541,824 bytes, so it is both significant and below the candidate's 262,361,760-byte footprint. I rebuilt the real Release product and reran the complete 14-phase production smoke with the same long request.

Every phase before 6g stayed green. Phase 6g reached the protected state with:

| Measurement | Narrowed production run |
| --- | ---: |
| registered / live `Module` owners | `316 / 0` |
| `weight_footprint_bytes` | `262,361,760 B` |
| `observed_active_bytes` | `255,724,192 B` |
| `baseline_active_bytes` | `871,607,252 B` |
| `returned_bytes` | `615,883,060 B` |
| generations in flight / stable samples | `0 / 3` |
| container deallocated | `true` |
| event | `generation_shared_cache_rebuilt` |
| rebuilds / abandoned / pending | `1 / 0 / false` |
| post-clear cache bytes | `0 B` |
| `/health` | `503` |

Thus every revision-5 clause read green: the wrapper was gone, all registered `Module` owners were gone, the allocator was idle and stable, active bytes were below the footprint, and returned bytes cleared the footprint. `GenerationEngine.deinit` passed that observation through the production `GenerationBatchLedgerStore.completeWorkerTeardown` call site (`GenerationEngine.swift:129-157`; `BatchLedgerStore.swift:141-149`), which returned `.rebuilt`, called `Memory.clearCache()`, emitted the completed event, and cleared the pending obligation while about 97.5% of the measured footprint was still active.

The maintained all-array phase only tests the exact/full-footprint boundary. The maintained module-subset phase cannot cover this class because keeping a `Module` alive makes ownership refuse first. A strict subset of raw arrays is the missing combination: significant weight residue, zero live registered owners, and residue below the full-footprint threshold.

## Required rework

1. Treat weak `Module` ownership as a veto, not proof that copied weight buffers are gone. Do not let any below-footprint residue reach `.rebuilt` unless the remaining bytes are attributable as non-weight state. Under the stated MLX boundary, fail closed and abandon when that attribution is unavailable; preserving a clean-path rebuild is optional, while a false release attestation is not.
2. Narrow the production raw-array retention phase to a strict subset that leaves zero live `Module` owners, significant active residue below the footprint, a satisfied returned-byte clause, zero in-flight generations, and a stable reading. It must still abandon with no completed-rebuild event, rebuilds=0, abandoned=1, pending=true, and `/health=503`.
3. Replace the unit expectation that accepts `footprint - 1` / half-footprint residue with a negative consistent with the fail-closed contract, then prove it through `GenerationEngine.deinit` → `GenerationBatchLedgerStore.completeWorkerTeardown`, not only through the pure helper.
4. Preserve the now-correct F1d-R4 long-prompt refusal, F1c-R3 refusal, outer-container timeout, recoverable failure/next-request behavior, and TASK-260827-2h39ya health boundary.
5. Record this round's correction in `LOGBOOK.md` during producer rework. The reviewer did not modify the candidate tree.

## Other verification

| Gate | Result |
| --- | --- |
| `swift test -c release` | exit 0; 197 tests / 19 suites |
| pristine Release `xcodebuild` | exit 0; BUILD SUCCEEDED |
| pristine generation-batch recovery smoke | exit 0; 170 checks / 0 failures |
| pristine dead-generation smoke | exit 0; 45 checks / 0 failures |
| `swift-format lint --recursive --strict Sources Tests` | exit 0; no diagnostics |
| `shellcheck -S warning` on both smoke scripts | exit 0; clean |
| narrowed raw-array production run | exit 1; 12 failures, all in phase 6g; false `.rebuilt` measured above |

Evidence archive: `TASK-260827-2q77g8_review-evidence-rev5-round5.tar.gz`.
