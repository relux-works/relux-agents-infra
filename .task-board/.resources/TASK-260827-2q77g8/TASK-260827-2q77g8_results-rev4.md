# TASK-260827-2q77g8 — revision 4

Rework for review finding **F1c-R3**: `WeightReleaseBarrier` treated a `weak`
`ModelContainer` reading `nil` as completed weight release, and Swift `weak`
references can nil while destruction of the stored state below them is still in
progress.

## The finding, reproduced as a shipped negative

Review's narrowed mutant delayed only `SerialAccessContainer<ModelContext>`
destruction by 3 s. Every phase before condemned teardown stayed green and 6a
emitted `release_observed=true shared_cache_rebuilds=1 pending=false
cache_bytes=0` with `active_bytes=262361760`.

That interval is now a production seam, not a scratch mutant.
`--fault-inject-teardown-retain-weights true` parks `ModelContext.model` — the
`LanguageModel` and its arrays, below the container — and lets the
`ModelContainer` be deallocated on schedule. The wrapper genuinely reaches
`weak`-`nil` with the whole model genuinely still active.

## Fix 1 — the release verdict is conjunctive and allocator-bound

`GenerationBatchRecovery.weightsReleased(_:)` is a pure gate over a
`WeightReleaseObservation`. It requires **both**:

- `containerDeallocated` — kept as a **veto**, not an attestation. While the
  wrapper lives the weights are certainly held, and no byte count may out-vote
  that.
- `returnedBytes >= weightFootprintBytes` — MLX's own `activeMemory` must have
  given back at least what *this* model cost to load. `weightFootprintBytes` is
  measured in `ModelLoader.load` as the `activeMemory` delta across the load.

No slack, and none is needed: whatever is legitimately still active at the end
was already active at the baseline, so it cancels out of the subtraction. An
unmeasured footprint (`0`) **fails closed** — a measurement that did not happen
is not a model that cost nothing.

This is not the revision-2 proportional aggregate. At the 29 GB target model it
demands all 29 GB back, where a half-baseline crossing admitted the clear with
~14.5 GB resident.

`teardownOutcome(releaseObserved:attempt:maxAttempts:)` is unchanged; it now
receives a verdict derived from the measurement rather than from a proxy.

## Fix 2 — the measurement is published, so the verdict can be checked

`generation_shared_cache_rebuilt`, `..._deferred` and `..._abandoned` all carry
`container_deallocated`, `weight_footprint_bytes`, `baseline_active_bytes` and
`returned_bytes`. `weight_footprint_bytes` is also on the `ready` event.

## Fix 3 — phase 6d, the narrowed production negative

Phase 6c (container seam) cannot reach this interval: parking the wrapper means
`weak`-`nil` never happens, so a wrapper-only barrier refuses for the wrong
reason. 6c now asserts `container_deallocated=false` and 6d asserts
`container_deallocated=true`, so the two negatives cannot be confused.

Phase 6a gained a MEASUREMENT GATE: the clean teardown must show
`container_deallocated=true`, a non-zero measured footprint, and
`returned_bytes >= weight_footprint_bytes`.

## Separation on MLX's own figures

| Path | container_deallocated | footprint | returned | active_bytes | cache_bytes | rebuilds | abandoned | pending | /health |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | ---: |
| 6a clean teardown | true | 262,361,760 | 302,274,956 | 2,720 | 0 | 1 | 0 | false | 503 |
| 6c container retained | false | 262,361,760 | — | 262,361,760 | 67,953,772 | 0 | 1 | true | 503 |
| 6d weights retained | **true** | 262,361,760 | 39,391,628 | 262,361,760 | 67,770,492 | 0 | 1 | true | 503 |

Row 6d is the reviewer's defect state. Revision 3 reported it as a completed
release; revision 4 abandons.

## Mutants — all die

| Mutant | Where | Result |
| --- | --- | --- |
| **P1** `weightsReleased` returns `containerDeallocated` only (revision 3's shape) | production | smoke exit 1, **7 failures, only phase 6d**: `shared_cache_rebuilds=1`, `cache_bytes=0`, `active_bytes=262,361,760` |
| **N1** *narrowing*: byte test relaxed to `returnedBytes > 0` | production | smoke exit 1, 7 failures in 6d; contract suite exit 1, both NARROWING tests redden |
| **N2** unmeasured-footprint guard deleted | contract | `swift test` exit 1, "an unmeasured footprint attests nothing" reddens |
| **M8** production `Memory.clearCache()` deleted (reviewer's earlier mutant, re-verified after the restructure) | production | smoke exit 1, 2 failures: REBUILD GATE (4,200,448 B below control, want ≥ 16,777,216) and ORDERING GATE (331,915,388 B held) |

P1 and N1 are narrowings, not deletions: the bound is proven by tightening the
gate's shape, not only by removing it. Pristine sources restored and verified
byte-for-byte (`shasum -a 256 -c`) before final verification.

## Gates — real exit codes

| Gate | Exit | Result |
| --- | ---: | --- |
| `xcodebuild` Release (arm64, `-skipPackagePluginValidation -skipMacroValidation`) | 0 | BUILD SUCCEEDED |
| `swift test -c release` | 0 | 182 tests / 19 suites (was 169 / 18) |
| `xcrun swift-format lint --recursive Sources Tests` | 0 | no diagnostics |
| `shellcheck -S warning` on both smokes | 0 | no diagnostics |
| `scripts/generation-batch-recovery-smoke.sh` | 0 | **115 checks**, 0 failures (was 98) |
| `scripts/dead-generation-smoke.sh` (2h39ya boundary) | 0 | 45 checks, 0 failures |

## Requirements 3, 4, 5

- Fail-closed abandonment and the container-retention seam are preserved
  unchanged; they were not the finding.
- The `TASK-260827-2h39ya` `/health` 503 boundary is asserted on both negative
  paths (6c and 6d) and by the unchanged 45-check dead-generation smoke.
- `LOGBOOK.md` 0530 entry appended, explicitly retracting the 0330 claim that
  wrapper `weak`-`nil` means "ARC having destroyed *this* worker's weights" with
  "no intermediate state".

## Not done / for the orchestrator

Changes remain **uncommitted** in the story worktree.
`task-board.config.json -> version_control.confirm` is true and integration into
trunk is the orchestrator's step. No installed config changed; Python `mlx-lm`
remains the default local runtime.
