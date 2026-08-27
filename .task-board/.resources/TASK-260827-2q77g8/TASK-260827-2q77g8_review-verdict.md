# TASK-260827-2q77g8 review verdict — changes requested

Reviewed Change Request `CR-TASK-260827-2q77g8-1`, revision 1, from base
`08beb4052faa851e686705677e24a20c2397ad87` to candidate tree
`18b67bc43f85af8d4404a4340f2c0d2d4dce4fbd` (`repository_delta=present`).

Verdict: **changes requested**, route to `to-dev`. This is ordinary autonomous
rework, not an external Stop-The-Line blocker.

## Findings

### F1 — The acceptance suite attests ledger bookkeeping, not resource release

The recovery call sites are present:

- `GenerationEngine.generate` catches every failed generation and calls
  `GenerationBatchLedgerStore.fail`.
- `GenerationBatchLedger.fail` closes the logical slot and increments
  `batchesReleased` / `sharedCacheRebuilds`.
- `GenerationBatchLedgerStore.fail` performs the only production
  `Memory.clearCache()` call.
- `GET /debug/generation-state` exposes both the ledger and MLX allocator
  counters.

However, the smoke asserts only the counters and event minted by the same
ledger. Two independent production mutants survived all 63 checks:

| Mutant | What production stopped doing | Smoke | Authoritative runtime evidence |
| --- | --- | ---: | --- |
| M8 | Removed only `Memory.clearCache()` while leaving ledger updates and events intact | 63/63 pass | Rebuild phase still reported `shared_cache_rebuilds=1`; `cache_bytes=67,955,820` instead of baseline `34,729,220` |
| M9 | Retained every failed `ChatSession`, keeping its KV state alive, while the ledger still closed the slot | 63/63 pass | After one failure `active_bytes=287,527,584` vs baseline `262,361,760` (`+25,165,824`); after two failures `312,693,408` vs `262,361,760` (`+50,331,648`) while ledger reported `active=0` and `batches_released=1/2` |

M9 is the required leak shape: the failed request terminates, the next small
request succeeds, and the ledger claims release while the allocator shows one
KV-sized increment per failure. The current suite therefore does not establish
the acceptance criterion that invalid batch/KV state is released.

The baseline condemned path also exposes a real ordering defect. Its final
state is `active_bytes=2,720`, `cache_bytes=299,129,824`, and
`shared_cache_rebuilds=1`. `Memory.clearCache()` runs inside
`GenerationEngine.generate` failure unwinding, before
`Router.recordGenerationFailure` makes `RuntimeState` drop the engine/model.
Buffers released after that clear repopulate the shared pool. The code claims
to return the pool before a replacement loads, but the post-condemnation state
shows almost the full 261 MB model in cache; the same ordering is unsafe for the
29 GB target model.

Required rework:

1. Bind the smoke to MLX allocator state, not only ledger counters/events.
   A mutant that retains the failed `ChatSession` must fail on retained active
   bytes, and deleting the production `Memory.clearCache()` call must fail on
   retained cache bytes.
2. Order cleanup after the relevant request/engine lifetimes actually end.
   Prove the condemned path from post-failure state or logs before replacement,
   not from a counter incremented before engine destruction.
3. Keep the slot identity/double-close guards; review found no double-counting
   defect in those value-type transitions. The defect is that those transitions
   are treated as proof of the external MLX action.

Negative-evidence shapes: **self-minted evidence**, and **check present but not
bound to the production action**.

### F2 — `metal-allocation-oversize` does not implicate the shared cache

The split between request-local KV state and MLX's process-global buffer cache
is architecturally valid. Collapsing the lifetimes would be wrong. The selected
positive cache-rebuild class is not valid, though.

In the pinned `mlx-swift` 0.31.6 allocator source,
`mlx/backend/metal/allocator.cpp:110-117` rejects
`size > device_->maxBufferLength()` before taking the cache lock or attempting
cache reuse/reclamation at lines 125-137. `clear_cache()` at lines 178-182 only
clears `buffer_cache_`; it cannot change `maxBufferLength()` and cannot make the
same oversized allocation succeed.

The smoke injects that message as text after one normal token. Its next request
uses a smaller normal allocation, so success proves that the worker remained
healthy; it does not prove that clearing the pool recovered the injected
failure. The current classifier pays the cold-cache cost for an error the cache
cannot cause or repair, contradicting its own narrowing rationale.

Required rework:

1. Remove `metal-allocation-oversize` from the shared-cache rebuild class, or
   replace it with a source-backed failure for which cache reclamation can
   materially change the next allocation.
2. Add a positive and a neighbouring negative that discriminate actual cache
   semantics, then reproduce a narrowing mutant at the production call site.
3. Append a corrective `LOGBOOK.md` entry. The existing 2325 entry is
   append-only institutional history and currently records the disproven claim
   that oversize allocation is blocked by freed cached buffers.

## What passed and remains worth keeping

| Check | Result |
| --- | ---: |
| Candidate `xcodebuild` Release | pass |
| `swift test -c release` | 151 tests / 15 suites pass |
| `swift-format lint --recursive Sources Tests` | pass, no diagnostics |
| Baseline generation-batch recovery smoke | 63/63 pass |
| `dead-generation-smoke.sh` regression | 45/45 pass |
| Producer E4 mutant: remove `ledger.fail` production call | reproduced, 17 smoke failures |
| Producer E2 mutant: rebuild shared pool for every failure | reproduced, 2 failures, both in no-rebuild phase |

The 2h39ya health boundary remains intact in both directions: the incident
signature reaches `/health` 503 and supervised restart; the neighbouring
request-scoped `Resource limit` message remains healthy and serving. The Change
Request did not widen or narrow `GenerationWorkerHealth.invalidatingSignatures`.

## Reproduction record

Review mutants ran in a disposable archive of the exact candidate tree under
`.temp/TASK-260827-2q77g8/mutants/candidate/`. Each touched source was restored
from a byte-for-byte copy and verified with `cmp`; the managed Story worktree
was never mutated by review.

All runtime mutants were rebuilt with the documented Release `xcodebuild`
command, then driven through the real Release executable, real
`model-harness`, and the cached 0.5B MLX model. Smoke log SHA-256 digests:

- baseline: `d8e2b3dca801b82ed8992bbfeac385eeee98f0aac42f0313e6957369f6a9a8ef`
- E4: `a8093944279e897d693af99b911146c28ebf53e4fc516b0cb3c00aed1634e50a`
- E2: `47014fe14e49fa36c2f4a3449ef82d9c12bbc6ec909335cb0c91b567b6d86434`
- M8: `653d0fb48feda2c3f39884cbf86ac376d54b0ed88530728a137df966736cd37c`
- M9: `dea3cc96ff00284f586889064d173e82f03d9982be4cc4f8f137677890b45df5`
- dead-generation control: `821655ef3f5acc9a4fd0e7bfb3f4458b9ba052d9e162703901d88fd07f64cfc0`

Review logs and JSON allocator snapshots are attached separately as
`TASK-260827-2q77g8_review-evidence.tar.gz`.
