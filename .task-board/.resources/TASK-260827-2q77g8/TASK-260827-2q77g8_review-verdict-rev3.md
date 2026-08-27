# TASK-260827-2q77g8 review verdict — revision 3

Reviewed Change Request `CR-TASK-260827-2q77g8-3`, revision 3, from base
`08beb4052faa851e686705677e24a20c2397ad87` to candidate tree
`7e3f5976f69e6d3faa30215b19cf841c0f000ae5`
(`repository_delta=present`). The supplied patch SHA-256 was independently
verified as
`a5373357dfef813efef4e5686184ab66e261ef45d6d12a40bef80888ab1591ef`.
All 13 worktree paths matched the candidate tree after review.

Verdict: **changes requested**, route to `to-dev`. This is ordinary autonomous
rework, not a Stop-The-Line boundary.

## Finding

### F1c-R3 — `weak ModelContainer == nil` precedes release of its weight-owning state

Revision 3 removes the process-global half-baseline heuristic, but the new
`WeightReleaseBarrier` is not the deterministic weight-release barrier its
comments and attestation claim:

- `GenerationEngine.deinit` creates a weak reference to `model.container`
  (`GenerationEngine.swift:114-128`).
- `WeightReleaseBarrier.isReleased` equates that weak reference reading `nil`
  with every weight buffer having been returned
  (`BatchLedgerStore.swift:217-269`).
- In pinned `mlx-swift-lm` `bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57`,
  `ModelContainer` is only a wrapper around the stored
  `SerialAccessContainer<ModelContext>` (`ModelContainer.swift:32-55`). The
  actual `LanguageModel` and its weight arrays live in
  `ModelContext.model` (`ModelFactory.swift:63-89`), held by
  `SerialAccessContainer.value` (`SerialAccessContainer.swift:44-51`).
- Swift weak references may become `nil` while destruction of stored state is
  still in progress. The independent scratch probe observed, in order:
  `payload-deinit-start`, `weak-nil`, then `payload-deinit-finish`.

That makes wrapper weak-nil a proxy signal, not proof that the inner model and
its arrays have finished releasing. The new gate is scoped to one wrapper, so
it is no longer a process-global aggregate wearing a new name; it is still an
insufficient lifetime proxy wearing the name "weights released".

#### Production-path narrowed mutant

I copied the exact candidate into disposable `.temp` storage and changed only
the pinned dependency's real `SerialAccessContainer<T>` destruction: when
`T == ModelContext`, its deinit retained `value` for 3 seconds. No request or KV
container was delayed. This narrows precisely the disputed interval: the outer
`ModelContainer` may reach weak-nil while its weight-owning stored state is
still alive.

The mutant Release build succeeded. Every smoke phase before condemned teardown
remained green, including both KV leak checks and the immediate cache-rebuild
and oversize narrowing checks. Phase 6a then produced:

```text
generation_shared_cache_rebuilt release_observed=true
active_bytes_before=262361760 active_bytes=262361760
cache_bytes_before=69556348 cache_bytes=0 shared_cache_rebuilds=1
FAIL TEARDOWN GATE: active_bytes=262361760, ceiling=16777216
```

The complete smoke exited 1 with exactly one failure. The runtime cleared the
pool, lowered `shared_cache_rebuild_pending`, and attested the rebuild while the
whole 262,361,760-byte model was still active. This is the
**capability claim that does not reproduce** / **prove, or report nothing**
negative shape: wrapper destruction was reported as completed weight release.

The shipped `--fault-inject-teardown-retain` negative does not cover this
interval. It retains the *outer* `ModelContainer`, so weak-nil never occurs at
all. It proves the timeout/abandonment branch and cannot distinguish a correct
post-weight-release barrier from one that fires between outer weak-nil and
inner stored-state destruction.

## Required rework

1. Replace wrapper weak-nil with a completion signal ordered after destruction
   of the weight-owning `ModelContext.model` / MLX arrays. Do not attest from a
   proxy that can precede stored-property destruction.
2. Add a production-path negative that allows the outer `ModelContainer` to
   become weak-nil while retaining the real inner weight-owning state. It must
   prove that `Memory.clearCache()`, `release_observed=true`, and the completed
   rebuild transition cannot occur until that inner state is released.
3. Preserve the current fail-closed abandonment path and lifetime-retain seam.
   They passed independent review and are not the finding.
4. Preserve the `TASK-260827-2h39ya` `/health` 503 boundary.
5. Append a corrective Flight Logbook entry. The candidate's 0330 entry states
   that weak-nil means ARC destroyed the weights and has no intermediate state;
   the narrowed production mutant disproves that statement. This reviewer did
   not edit `LOGBOOK.md`, because the reviewer role is read-only and must not
   change the candidate under review.

## Round-3 claims that did pass

### Fail-closed timeout and no later false completion

The shipped phase 6c passed independently: with the real retained container,
`active_bytes=262361760`, `shared_cache_rebuilds=0`,
`shared_cache_rebuilds_abandoned=1`, pending `true`, and `/health=503`.

A focused lifetime probe then sampled the same runtime PID at abandonment and
12 seconds later. Both snapshots were unchanged:

| Snapshot | Active | Cache | Rebuilds | Abandoned | Pending | Health |
| --- | ---: | ---: | ---: | ---: | --- | ---: |
| At abandonment | 262,361,760 B | 68,508,660 B | 0 | 1 | true | 503 |
| 12s later, same PID | 262,361,760 B | 68,508,660 B | 0 | 1 | true | 503 |

`RetainedWeights.shared` holds the real container in a process-static strong
array with no removal path (`BatchLedgerStore.swift:273-296`). The only
production caller of `completeWorkerTeardown` is the one task scheduled by
`GenerationEngine.deinit`, and it breaks after `.abandoned`. No later
production path can convert this abandoned teardown into a completed one.

### 2h39ya boundary

`scripts/dead-generation-smoke.sh` passed all 45 checks. Buffered and streaming
condemnation still move `/health` to 503, later completions are refused, the
supervisor replaces the process, and both benign and over-broad neighbouring
request failures stay healthy.

## Independent verification

| Check | Result |
| --- | --- |
| Exact 13-path worktree content vs candidate tree | match |
| CR patch SHA-256 | match |
| `git diff --check` on exact CR delta | pass |
| macOS arm64 Release `xcodebuild` | pass |
| `swift test -c release` | 169 tests / 18 suites pass |
| `swift-format lint --recursive Sources Tests` | pass, no diagnostics |
| `shellcheck -S warning` on both smokes | pass, no diagnostics |
| pristine generation-batch smoke | 98 checks, 0 failures |
| retained seam, post-abandonment lifetime probe | pass, same PID and state after 12s |
| `dead-generation-smoke.sh` | 45 checks, 0 failures |
| ModelContext-only delayed-destruction mutant build | pass |
| ModelContext-only delayed-destruction mutant smoke | exit 1, exactly 1 expected failure in phase 6a |

Primary evidence SHA-256 values:

- pristine build log: `6a75aad87bef26305fd7828dda5c0c30dd3e247597ec01b61dbcc43ec29c113d`
- contract test log: `54859c9fffe3e6cea2d53e19001ffca39ef64da755ae1a0d62a11152142b4461`
- pristine recovery smoke: `c3513a2ca94df9ce72b595c0f2ee001755472536843cb20bb764f04e8c53d275`
- dead-generation smoke: `edef507f4e1e9fd36112598b15bccf53e58e5e23573adaefb15d45d882330d27`
- Swift weak-lifetime probe: `e0d1580073fa2afcc0cda1bd0bea1af64a2e9a8afd0609d34e6ebec87e970187`
- narrowed production mutant smoke: `2e505ffd6b2df861d3173fd891a5a84a884d9ec7df1fe2528bb400c16c4ad2af`
- retained lifetime probe: `c299ae18a5ff40403ab6e9732df379abe83ec086ab702ce1554e5bf45eaf9f46`

All review mutations and build products live only in disposable `.temp`
storage. The managed Story worktree's 13 review paths still match the candidate
tree exactly.
