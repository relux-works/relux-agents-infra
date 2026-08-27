# TASK-260827-2q77g8 review verdict — revision 2

Reviewed Change Request `CR-TASK-260827-2q77g8-2`, revision 2, from base
`08beb4052faa851e686705677e24a20c2397ad87` to candidate tree
`3e45efc8f1959a9c0f389cef149e6f9ba07ab374` (`repository_delta=present`).
The supplied patch SHA-256 was independently verified as
`d6a4c81e12b83386bc4c5ae0f49b8e8660dd7c8aedc5d1f7d04cabcf8936eed1`.

Verdict: **changes requested**, route to `to-dev`. This is ordinary autonomous
rework, not a Stop-The-Line boundary.

## Finding

### F1b-R2 — teardown treats a partial observation or timeout as successful release

Revision 2 fixes the fixture's original cleanup order, but the replacement gate
does not establish that the condemned model has actually finished releasing its
buffers:

- `GenerationEngine.deinit` asks for `activeMemory < baseline / 2`
  (`GenerationEngine.swift:99-109`). For the 29 GB target model this admits the
  clear while almost 14.5 GB can still be active and about to fall into
  `buffer_cache_`. The 261 MB fixture happens to drop from the model-sized value
  to 2,720 B in one observed step; that result does not make the proportional
  threshold sound for a much larger, incrementally released model.
- Concurrent requests that already captured the engine do delay its `deinit`,
  which correctly prevents ordinary in-flight requests from racing this clear.
  The remaining signal is still process-global, not tied to this `LoadedModel`:
  unrelated MLX active-memory movement can cross a proportional aggregate
  threshold without proving that every condemned-model buffer is gone.
- `waitForWeightRelease` is bounded to 100 × 20 ms and returns `false` on
  timeout (`BatchLedgerStore.swift:164-173`). The caller discards that Boolean
  and unconditionally calls `completeWorkerTeardown`.
- `completeWorkerTeardown` first clears `rebuildPendingWorkerTeardown`, performs
  one `Memory.clearCache()`, increments the rebuild attestation, and never
  retries (`BatchLedgerStore.swift:120-140`). If the observation timed out, or
  crossed at a half-released state, the remaining buffers later refill the pool
  and the ledger still says the rebuild completed.
- The source comment and producer evidence say a timeout is reported, but the
  `generation_shared_cache_rebuilt` event contains no release-observed/timeout
  field. Its pre/post allocator readings permit later diagnosis; they do not
  prevent or explicitly report the failed precondition.

The 0.5B smoke's fixed 128 MiB ceiling catches a nearly whole 261 MB model left
behind. It has no timeout/partial-release negative and does not establish that
the same rule leaves negligible residue for the 29 GB runtime in scope. This is
the **absent evidence treated as satisfied** shape: failure to observe the
release takes the same success transition as observing it.

Required rework:

1. Couple the deferred clear to a deterministic lifetime/release barrier, or to
   a criterion that establishes essentially all of this worker's condemned
   allocation is gone; do not use a half-baseline crossing as completion.
2. On timeout, fail closed: do not clear the pending flag or attest a completed
   rebuild. Emit an explicit `release_observed=false`/timeout outcome and use a
   bounded retry or a terminal worker/process path that cannot leave the old
   model competing with its replacement.
3. Add production-path negative evidence for both partial release and timeout.
   The gate must fail when substantial residue remains, not only when the entire
   261 MB fixture remains cached.
4. Preserve the current `/health` 503 boundary while reworking teardown.

## Round-1 findings retested

### M8 — delete production `Memory.clearCache()`

Reproduced independently in a new APFS copy-on-write disposable copy of the
exact revision-2 candidate. The ledger counter and event remained intact.

- Release mutant build: pass.
- Smoke: exit 1, 2 failures.
- `REBUILD GATE`: only 2,651,152 B below the no-rebuild control, below the
  required 16,777,216 B margin.
- `ORDERING GATE`: 332,228,732 cache bytes retained after condemnation, above
  the 134,217,728 B ceiling.

M8 now dies. The allocator-bound check defeats the round-1 self-minted-evidence
shape.

### M9 — retain each faulted `ChatSession`

After restoring the M8 source byte-for-byte, the disposable copy retained the
actual `ChatSession` at the injected mid-batch throw.

- Release mutant build: pass.
- Smoke: exit 1, 3 failures.
- One failed session: +25,165,824 active bytes.
- Two failed sessions: +50,331,648 active bytes.
- The successful third request leaves the same +50,331,648 B residue while the
  logical ledger still reports both failed batches released.

M9 now dies at the expected production shape and at the reviewer's original
increments.

## F2 allocator-source verdict

The replacement class `metal-buffer-allocation-failed` is source-backed in the
pinned `mlx-swift` 0.31.6 checkout (`0bb916c67f4b9e5c682cbe02a42c701c93ab5021`):

- `MetalAllocator::malloc` attempts cache reuse, computes `mem_required`, and
  calls `release_cached_buffers(mem_required - gc_limit_)` at allocator.cpp
  125-137 before allocation.
- `[malloc] Unable to allocate N bytes.` is thrown only when the following
  `newBuffer` attempts still return null, at lines 147-156.
- `clear_cache()` calls `buffer_cache_.clear()` at lines 178-182, so it can
  release cached buffers deliberately retained by the earlier partial slice.
- The removed oversize message is still correctly excluded: it throws at lines
  111-117 before the lock/cache path and compares against `maxBufferLength()`.

F2 is resolved. A full pool rebuild can materially change a later retry for the
new class; it cannot repair the removed oversize class.

## Independent verification

| Check | Result |
| --- | --- |
| Exact 13-path worktree content vs candidate tree | match |
| `git diff --check` on exact CR delta | pass |
| macOS arm64 Release `xcodebuild` | pass |
| `swift test -c release` | 154 tests / 15 suites pass |
| `swift-format lint --recursive Sources Tests` | pass, no diagnostics |
| `shellcheck -S warning` on recovery smoke | pass, no diagnostics |
| baseline generation-batch smoke | 81 checks, 0 failures |
| M8 generation-batch smoke | exit 1, 2 expected failures |
| M9 generation-batch smoke | exit 1, 3 expected failures |
| `dead-generation-smoke.sh` | 45 checks, 0 failures; health 503 preserved |

Review logs are under `.temp/TASK-260827-2q77g8/`. Important SHA-256 values:

- baseline smoke: `1c974880be02979ffdd4a8a5d5b7727649b1887c6f9350260f4351fd5af70ff2`
- M8 smoke: `d179580138ecbab14721bb41c80cce782ee8c67c1e5ea4fbfae380c9c1ea9bc0`
- M9 smoke: `e732258ae158a58a497b130c0a01c3c135239f8bfe57428750a47d30185c04e3`
- dead-generation smoke: `c0bfeacfc1d46d876d36cc4c53c55d43435546812f35f78937462887962c4f15`

The managed Story worktree's 13 review paths still match the candidate tree;
all mutants and their build products live only in the disposable `.temp` copy.
