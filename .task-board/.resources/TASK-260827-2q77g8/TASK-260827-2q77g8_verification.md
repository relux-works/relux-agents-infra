# TASK-260827-2q77g8 — revision 2 (CR rework)

Reworks `CR-TASK-260827-2q77g8-1` after the reviewer's **changes requested**
verdict. Both findings are addressed, and both mutants that survived revision 1
now redden the suite at the reviewer's own measured numbers.

All commands were run in this worktree against the real Release binary under the
real `model-harness`, on `mlx-community/Qwen1.5-0.5B-Chat-4bit`.

## Gate results

| Gate | Command | Exit | Result |
| --- | --- | ---: | --- |
| Release build | `xcodebuild build -scheme … -configuration Release` | 0 | BUILD SUCCEEDED |
| Contract suite | `swift test -c release` | 0 | 154 tests / 15 suites passed (was 151) |
| Recovery smoke | `scripts/generation-batch-recovery-smoke.sh` | 0 | 81 checks, 0 failures (was 63) |
| 2h39ya regression | `scripts/dead-generation-smoke.sh` | 0 | 45 checks, 0 failures |
| Swift lint | `swift format lint --recursive Sources Tests` | 0 | clean |
| Shell lint | `shellcheck -S warning …` | 0 | clean at warning+ |

`shellcheck` at default (info) severity exits 1, as do all five sibling scripts
in `scripts/` — SC2086 on a deliberately word-split PID list and SC2015 on the
`A && pass || fail` idiom. No new lint class was introduced.

## F1 — resource claims are now bound to the allocator, not to the ledger

**Root cause.** `GenerationBatchLedger.fail` incremented `sharedCacheRebuilds`
when it *built the plan*, and the smoke asserted the ledger's own JSON. The
check and the thing checked had the same author.

**Fix.**

- The counter moved to `recordSharedCacheRebuild()`, called one line after
  `Memory.clearCache()` returns and nowhere else.
- The smoke anchors every resource claim to `mlx.active_bytes` /
  `mlx.cache_bytes`, which come from `Memory.snapshot()`.
- New **leak** phase: active bytes must not grow across failed generations.
- The **rebuild** phase measures cache bytes against a **no-rebuild control**
  run — same model, prompt and pinned seed, so only the clear differs.

**Measured on the fixed build:**

| Reading | Value |
| --- | ---: |
| baseline active_bytes (model resident) | 262,361,760 |
| after 1 failure / 2 failures / a success | 262,361,760 (unchanged) |
| control cache_bytes (pool left alone) | 67,968,124 |
| cache_bytes after the rebuild | 39,916,512 |

## F1b — the condemned-path cleanup ordering defect

Review measured 299,129,824 cache bytes on a condemned runtime whose ledger
reported the pool rebuilt. The clear ran while the weights were still held.

Placing it correctly took three attempts, all measured:

| Attempt | active_bytes at the moment of the clear | Verdict |
| --- | ---: | --- |
| inside the failing request | — | request still holds the engine |
| after the response is written | 303,782,980 | still too early |
| `GenerationEngine.deinit` | 303,850,548 | still too early — a `deinit` body runs *before* stored properties are released |
| deinit + bounded wait on observed release | 2,720 | correct |

The teardown now waits on an **observed** fact rather than an assumed ordering:
`deinit` schedules a task that polls `Memory.snapshot().activeMemory` until it
falls below half the condemnation-time reading (bounded to 2 s; a timeout is
reported, not hidden), then clears. Measured after the fix:

```
cache_bytes_before = 329,274,476   cache_bytes = 0   active_bytes = 2,720
```

The whole model goes back to the system. An **ORDERING GATE** in the smoke fails
if a condemned runtime is left holding it, so the ordering is checked rather
than trusted.

## F2 — `metal-allocation-oversize` was never shared-cache pressure

Confirmed from the pinned `mlx-swift` allocator source. `MetalAllocator::malloc`
throws the oversize message as its **first act**, before
`std::unique_lock lk(mutex_)` and before any cache is consulted, testing
`size > device_->maxBufferLength()`. `clear_cache()` empties `buffer_cache_` and
cannot move that limit, so no rebuild makes the request succeed. The old phase
passed only because the request it made afterwards was small enough to succeed
either way.

**Fix.** The pressure class is now `metal-buffer-allocation-failed` —
`[malloc]` **and** `Unable to allocate` — the throw taken when `newBuffer`
returns null. Reaching it means the allocator already ran
`release_cached_buffers(mem_required - gc_limit_)`, a *slice* sized only to get
back under `gc_limit_`, and still failed; `clear_cache()` empties the pool
outright, so it can return what that partial reclaim kept. It does not condemn.
`[malloc]` is not a substring of `[metal::malloc]`, so the pressure and
condemning classes cannot collide through their tags — pinned by a test.

Oversize is now a **narrowing negative** in both suites. Measured: cache_bytes
after an oversize rejection 65,149,052 against a control of 67,968,124 — the
pool was left alone.

A corrective `LOGBOOK.md` entry was appended (2026-08-28 0130); the 2325 entry's
claim about oversize is explicitly retracted there.

## Negative evidence — the reviewer's mutants now die

| Mutant | Revision 1 | Revision 2 |
| --- | --- | --- |
| **M8** delete the production `Memory.clearCache()`, leaving counters and events intact | 63/63 **pass** | **exit 1**, 2 failures: REBUILD GATE (only 313,344 B below control, want ≥ 16,777,216) and ORDERING GATE (331,915,388 B still held) |
| **M9** retain the failed `ChatSession`, ledger still closes the slot | 63/63 **pass** | **exit 1**, 3 failures: LEAK GATE at +25,165,824 B after one failure and +50,331,648 B after two — the reviewer's exact measurements |

Mutants carried forward from revision 1 and re-confirmed this run: always-rebuild,
never-rebuild, disjunctive matcher, unreleased slot, double-close, `generate`
never calling the ledger (unit suite stays green — only the smoke catches it),
and a swallowed generation error.

Two findings worth keeping:

**A unit suite cannot prove the production call site.** Deleting the ledger calls
from `GenerationEngine.generate` — leaving the ledger type perfect and simply
never called — keeps `swift test` at exit 0 with every test green, while the
smoke fails 14 checks across every phase.

**A status-code assertion would have missed the swallowed-error mutant.** Making
`generate` swallow the failure still answered `500`, from a downstream guard. What
caught it was the body assertion that the error names the injected failure, plus
the condemnation phase going `/health` 200 and serving the next request from a
worker that should have been condemned.

## Logs

Under `tools/mlx-swift-runtime-prototype/.temp/` (gitignored):
`xcodebuild-clean.log`, `final-swift-test.log`, `final-lint.log`,
`final-smoke.log`, `final-dead-smoke.log`, `smoke-m8.log`, `smoke-m9.log`,
`smoke-rework-0{1,2,3}.log`, plus per-phase transcripts in `final-out/`,
`m8-out/`, `m9-out/`.
