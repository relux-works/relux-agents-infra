# TASK-260827-2q77g8 — generation-batch failure recovery on the MLX Swift runtime

Revision 3. Carries the *recoverable* half of the `mlx_lm.server` generation
regression into the MLX Swift runtime's acceptance suite. `TASK-260827-2h39ya`
pinned the terminal half (a backend that is gone must stop answering `/health`
200). This task pins the far more common one: a generation that fails mid-batch
on a runtime that is still able to serve the next caller.

Revisions 1 and 2 are described in the review verdicts on this task. This
document leads with what revision 3 changed, because that is the whole of the
delta under review.

## What revision 3 fixes — F1b-R2

Review found that the deferred shared-pool rebuild for a **condemned** worker
could complete without the condemned worker's weights ever having been
released. Three distinct defects in one gate:

| Defect | Revision 2 | Revision 3 |
| --- | --- | --- |
| Threshold | `activeMemory < baseline / 2`. At the 29 GB target model this admits the clear with ~14.5 GB still active and about to fall into the pool. | No threshold. `WeightReleaseBarrier` holds a `weak` reference to the exact `ModelContainer` the condemned engine served from and waits for it to read `nil`. |
| Scope | Process-global `activeMemory`. Any unrelated MLX movement can cross a proportional aggregate without one condemned buffer being freed. | Scoped to this worker's container object. Alive, or destroyed by ARC. No intermediate state. |
| Timeout | `waitForWeightRelease` returned `false`; the caller discarded the Boolean, cleared the pending flag, cleared the pool once, and attested a completed rebuild. | Fails closed. An unobserved release cannot reach the clear at all. |

The **partial-release** class the reviewer named is eliminated by construction
rather than bounded: a weak reference has no half-released reading to admit.

### The fail-closed path

`GenerationBatchRecovery.teardownOutcome(releaseObserved:attempt:maxAttempts:)`
is a pure gate. It cannot return `.rebuilt` without an observed release.
Unobserved attempts return `.retry` up to `workerTeardownAttempts` (3), then
`.abandoned`. An abandoned teardown:

- performs **no** `Memory.clearCache()`
- attests **no** rebuild — `shared_cache_rebuilds` stays where it was
- leaves `shared_cache_rebuild_pending` **raised**, so the pool keeps reporting
  as owed for the life of the process
- counts `shared_cache_rebuilds_abandoned`
- emits `generation_shared_cache_rebuild_abandoned` with `release_observed=false`
- **re-announces the supervision marker**, because a host still holding a
  condemned model must not be left competing with its replacement for that
  memory, and the replacement process is what actually frees it

`GET /debug/generation-state` publishes all three new fields, so the failure is
a readable property of runtime state rather than something to reconstruct from
an event stream.

### Requirement 4 — the 2h39ya boundary

Preserved and asserted, not assumed. Phase 6c checks `/health` is still `503`
after a *failed* teardown, and `dead-generation-smoke.sh` still runs 45 checks
with 0 failures.

## The two teardown paths, on MLX's own figures

Not ledger counters — `Memory.snapshot()` readings published by the endpoint.

| Reading | 6a: release observed | 6c: release never observed |
| --- | ---: | ---: |
| `mlx.active_bytes` | 2,720 | 262,361,760 |
| `mlx.cache_bytes` | 0 | 69,526,636 |
| `shared_cache_rebuilds` | 1 | 0 |
| `shared_cache_rebuilds_abandoned` | 0 | 1 |
| `shared_cache_rebuild_pending` | false | true |

## Production-path negative evidence — requirement 3

`--fault-inject-teardown-retain true` does not ask the runtime to *report* a
timeout. It parks the real `ModelContainer` for the lifetime of the process, so
the weights genuinely are never released and the barrier genuinely never
observes one. The residue is then measured from `mlx.active_bytes` against a
134,217,728 B floor, so a seam that retained nothing cannot pass the phase by
default.

The flag is refused without `--fault-inject-generation-error` (only a condemned
worker has a deferred teardown) and refused for any value other than
`true`/`false`.

**A seam that lets go is not a seam.** The first cut held the container only for
the duration of the wait. By the time the suite read the allocator the reference
had been dropped and the 262 MB residue had already drained into
`buffer_cache_`, so the phase measured `active_bytes=2,720` and failed its own
residue floor. Caught by the floor, which is the point of having one.

## Mutants — every new gate was made to fail

| # | Mutant | Layer | Result |
| --- | --- | --- | --- |
| N1 | `teardownOutcome` always `.rebuilt` — the revision-2 shape | contract | exit 1, **7 issues** |
| N2 | **NARROWING**: fail closed while retrying, clear anyway at the bound | contract | exit 1, 3 issues |
| N3 | abandoning also attests a rebuild (self-minted evidence) | contract | exit 1, 4 issues |
| P1 | the revision-2 shape, end to end | **smoke** | exit 1, **6 failures — only phase 6c reddened**, at `active_bytes=262,361,760` beside `cache_bytes=0`: the defect measured directly |
| P2 | `WeightReleaseBarrier.isReleased` always `true` | **smoke** | exit 1, 7 failures; also reddens the 6a ordering gate at 303,848,400 B |
| M8 | delete the production `Memory.clearCache()` (reviewer's round-1 mutant, re-verified after the restructure) | **smoke** | exit 1, 2 failures |

N2 is the narrowing mutant: the gate is not merely present, it holds across the
whole class rather than only at the delete-it boundary. P1 reddening *only*
phase 6c is the corresponding narrowness result at the production path.

Mutants ran in the story worktree with pristine backups restored and verified
byte-for-byte (`shasum -a 256` before and after, `diff` clean); the final
verification below ran on the restored tree.

## Verification actually run

Every command run directly as a standalone process; exit codes are real.

| Command | Exit | Result |
| --- | ---: | --- |
| `xcodebuild build -configuration Release -destination 'platform=macOS,arch=arm64'` | 0 | BUILD SUCCEEDED |
| `swift test -c release` | 0 | **169 tests / 18 suites** (was 154 / 15) |
| `xcrun swift-format lint --recursive Sources Tests` | 0 | **0 diagnostics** |
| `shellcheck -S warning` on both smokes | 0 | no diagnostics |
| `scripts/generation-batch-recovery-smoke.sh` | 0 | **98 checks, 0 failures** (was 81) |
| `scripts/dead-generation-smoke.sh` | 0 | **45 checks, 0 failures** — 2h39ya intact |

Logs: `.temp/FINAL-release-build.log`, `.temp/FINAL-swift-test.log`,
`.temp/FINAL-lint.log`, `.temp/FINAL-batch-recovery.log`,
`.temp/FINAL-dead-generation.log`. Mutant logs: `.temp/mutant-P1-smoke.log`,
`.temp/mutant-P2-smoke.log`, `.temp/mutant-M8-smoke.log`.

## Acceptance criteria

| AC | Evidence |
| --- | --- |
| Deterministic failure injected reproducibly from a recorded command | `--fault-inject-generation-error <msg> --count N --after-tokens 1`, fixed `seed`; phases 1–6c |
| Affected request terminates with an explicit error, not a hang or truncated success | Phase 1: HTTP 500 `generation_failed`, body asserted to contain **no** `choices`. Phase 2 (SSE): terminal error frame and **no** `finish_reason` frame. |
| Invalid batch and cache state released or rebuilt, evidenced by runtime state or logs | `GET /debug/generation-state` — `active=0`, `batches_released=1`, allocator-bound leak and rebuild gates; `generation_batch_released` event |
| A subsequent request completes without restarting a healthy process | Phase 1 request 2 → 200 with real text and usage; **exactly one** `listening` event; no `restarting profile` line |
| Unrecoverable worker death still reports 503 | Phase 6a: `/health` 503 `{"status":"unavailable"}`, next request refused 503, marker emitted, batch still released. Phase 6b: supervised restart, replacement answers 200. Phase 6c: 503 holds even when the teardown itself fails. |

## Scope notes

- Python `mlx-lm` remains the default local runtime. Nothing installed changed;
  `examples/model-harness.prototype.toml` is still an example, and gains only a
  comment explaining that the abandoned-rebuild marker is deliberately fatal.
- `GET /debug/generation-state` gains `shared_cache_rebuilds_abandoned` and
  `shared_cache_rebuild_pending`. `/health` and `/v1/models` bodies are
  unchanged — verified by the 2h39ya suite passing.
- Changes are **uncommitted** in the story worktree:
  `task-board.config.json -> version_control.confirm` is `true`, and integration
  is the orchestrator's step.
