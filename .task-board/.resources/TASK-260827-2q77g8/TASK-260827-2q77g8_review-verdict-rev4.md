# TASK-260827-2q77g8 review verdict — revision 4

## Verdict

**Changes requested → `to-dev`.** Revision 4 closes the revision-3 outer-container lifetime defect, but its replacement byte gate can still attest a completed weight release while the retained model weights remain fully active. This is ordinary implementation rework, not a stop-the-line boundary.

Reviewed Change Request: `CR-TASK-260827-2q77g8-4`, revision `4`, candidate tree `85cb9ae65b4d585e0a2b12cc08d28a944b23df93` over base `08beb4052faa851e686705677e24a20c2397ad87`. The supplied binary patch SHA-256 was independently reproduced as `d13ee706b24addd97cd171bfa359793936243e7289751bf01ea9cff9ab7a3b7c`. All 14 candidate path hashes matched the review worktree before the verdict.

## Blocking finding

### F1d-R4 — process-global release delta is proxy evidence and admits live weights

Negative shape: **forged or self-minted evidence**, expressed here as a process-global proxy that unrelated/request-local deallocation can satisfy. The production gate at `GenerationBatchRecovery.weightsReleased(_:)` accepts when:

```text
container_deallocated
&& weight_footprint_bytes > 0
&& baseline_active_bytes - active_bytes >= weight_footprint_bytes
```

The model-load footprint itself was stable and credible in this candidate: `ModelLoader.load` samples immediately around the accepted factory load before readiness admits traffic, the pinned loader evaluates the model before returning, and every real run measured `262,361,760 B`. The defect is the other side of the comparison. `WeightReleaseBarrier.init` samples process-global `MLX.Memory.activeMemory` in `GenerationEngine.deinit`; that baseline can still include allocations from the failed generation. Their later release is indistinguishable from weight release.

I drove the composed production path with the real Release binary, real 0.5B model, real HTTP request, real fault classification, `GenerationEngine.generate` → `RuntimeState.recordGenerationFailure` → `GenerationEngine.deinit` → `WeightReleaseBarrier.waitForRelease` → `GenerationBatchLedgerStore.completeWorkerTeardown`, and the existing `--fault-inject-teardown-retain-weights true` seam. The seam retained `ModelContext.model` below a deallocated container while a deterministic 6,000-word prompt made request-local MLX allocation larger than the model.

Two consecutive runs reproduced the same false attestation:

| Measurement | Run 1 | Run 2 |
| --- | ---: | ---: |
| `weight_footprint_bytes` | `262,361,760 B` | `262,361,760 B` |
| `baseline_active_bytes` | `871,271,352 B` | `871,271,336 B` |
| `returned_bytes` | `608,909,592 B` | `608,909,576 B` |
| post-teardown `active_bytes` | `262,361,760 B` | `262,361,760 B` |
| `container_deallocated` | `true` | `true` |
| teardown event | `generation_shared_cache_rebuilt` | `generation_shared_cache_rebuilt` |
| ledger (`rebuilds` / `abandoned` / `pending`) | `1 / 0 / false` | `1 / 0 / false` |
| `/health` | `503` | `503` |

The post-teardown active bytes exactly equal the measured footprint: all model weights remain active. Nevertheless, request-local release contributed about `608.9 MB` to the process-global subtraction, exceeded the `262.4 MB` footprint, set `release_observed=true`, called `Memory.clearCache()`, and cleared the pending obligation. This is a direct production bypass, not a fabricated observation passed to a helper.

The short-prompt phase 6d does not cover this class: it returned only `40,440,196 B`, below the footprint, so the pristine suite correctly abandoned there. Scaling the same production request changes only the amount of unrelated/request-local memory in the teardown baseline and defeats the gate. Concurrent in-flight allocation has the same attribution problem; the process-global counter cannot identify which owner returned the bytes.

## Required rework

1. Do not let a process-global drop attest release of this model's weights unless the observation excludes, attributes, or conservatively vetoes non-weight allocation. If exact attribution is unavailable, fail closed; a false abandonment is safer than claiming that live weights were returned.
2. Carry the attached long-context inner-retention probe into the maintained acceptance suite. It must require retained active weight residue, `shared_cache_rebuilds=0`, `shared_cache_rebuilds_abandoned=1`, `shared_cache_rebuild_pending=true`, no `generation_shared_cache_rebuilt` event, and `/health=503`.
3. Add a narrowed unit negative where the container is gone, live model-sized residue remains, and a larger non-weight allocation has disappeared since baseline. The observation must not reach `.rebuilt`. With the current four-field observation, this case is indistinguishable from success, so a green helper-only test is insufficient.
4. Preserve the now-correct F1c-R3 behavior, the outer-container retention timeout, and the TASK-260827-2h39ya health boundary.

## Other review results

- Pristine generation-batch recovery smoke: **passed, 115 checks / 0 failures**. Short-prompt 6d failed closed with `262,361,760 B` active, `40,440,196 B` returned, `rebuilds=0`, `abandoned=1`, `pending=true`, `/health=503`.
- Independent F1c-R3 dependency mutant: added only a 3-second `deinit` delay to pinned `SerialAccessContainer<ModelContext>`, rebuilt the real Release product, and reran the full production smoke. The mutant **died with 8 failures** in phase 6a; revision 4 did not emit an early completed-rebuild attestation while the weights remained active.
- Outer-container retention phase 6c: passed fail-closed assertions (`rebuilds=0`, `abandoned=1`, `pending=true`, no completed event, repeated supervision marker, `/health=503`).
- TASK-260827-2h39ya `dead-generation-smoke.sh`: **passed, 0 failures**, including buffered/streaming 503, supervised replacement, benign negative, and over-broad classifier negative.
- Swift Testing: **182 tests in 19 suites passed**.
- Release `xcodebuild`: **BUILD SUCCEEDED**.
- `swift-format lint --strict` and `shellcheck`: **clean**.
- `git diff --check`: clean.

## Reproduction

The evidence archive contains the exact probe and complete runtime artifacts. From the story worktree after extracting the archive under `.temp/TASK-260827-2q77g8/review-rev4`:

```bash
ROOT="$PWD" \
OUT="$PWD/.temp/TASK-260827-2q77g8/review-rev4/repro" \
PORT=18034 \
  .temp/TASK-260827-2q77g8/review-rev4/long-context-inner-retain-probe.sh
```

Correct behavior makes the probe exit `0`. Revision 4 exits `1` with `MEASUREMENT GATE BYPASS` and prints the full measurement record above.

Evidence archive: `TASK-260827-2q77g8_review-evidence-rev4.tar.gz`, SHA-256 `2ea175343e5645c11eaaec92a28b855810ff5dd5b038f4174a4481cb901156a2`.
