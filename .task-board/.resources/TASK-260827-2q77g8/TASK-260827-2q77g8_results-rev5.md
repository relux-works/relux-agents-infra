# TASK-260827-2q77g8 — revision 5

Rework for review finding **F1d-R4**: the process-global release delta is proxy
evidence and admits live weights.

## The finding, restated

Revision 4 decided "the weights came back" from

```text
container_deallocated && weight_footprint_bytes > 0
&& baseline_active_bytes - active_bytes >= weight_footprint_bytes
```

Review drove the composed production path with a 6,000-word prompt, which makes
the failed request's own KV state larger than the model. Releasing the request
alone paid the subtraction: `returned_bytes` 608,909,592 against a 262,361,760 B
footprint, a completed rebuild attested, `Memory.clearCache()` performed, the
pending obligation cleared — with post-teardown `active_bytes` at exactly
262,361,760. Every weight still resident. A process-global counter cannot say
whose bytes came back.

## FIX 1 — attribution is read where it lives

`WeightOwnerRegistry` registers this model's whole module tree
(`ModelContext.model.modules()`, 316 objects for the acceptance model) weakly, in
`ModelLoader.load`, immediately after the accepted factory load. Every weight
array of an MLX Swift model is a stored property of one of those objects, so
"none of them is alive" is a claim about *these* weights rather than about the
process's byte total. `weightOwnerCount == 0` — a registry that was never
populated — fails closed, because "zero live owners" would otherwise be true
forever.

## FIX 2 — the byte test is absolute, not a delta

`activeBytes < weightFootprintBytes`. The question is what is still resident,
not what moved: if this model's weight set were held, MLX would call at least
the whole footprint active, so any reading at or above the footprint is refused
however large the process-global drop was. Review's bypass measured
`activeBytes == weightFootprintBytes` and dies here. The revision-4 delta is
kept as a **necessary** condition — the fall must have happened inside this
teardown window — and is never again sufficient.

## FIX 3 — the two conditions review named beside the finding

No generation may be in flight when the reading is taken (a concurrent request's
allocation is equally unattributable), and `activeMemory` must report the same
value for `GenerationBatchRecovery.minimumStableActiveSamples` consecutive polls.
A destruction still running is a falling count, and a single sample taken during
that fall is a partial release read as a finished one.

## FIX 4 — the acceptance suite, and two seams that isolate single clauses

Requirement 2 of the rework: review's long-context inner-retention probe is now
**phase 6e** of `scripts/generation-batch-recovery-smoke.sh`. It asserts the
bypass condition is *present* — `returned_bytes` must clear the footprint, which
is what an earlier revision attested from — and then requires abandonment,
`shared_cache_rebuilds=0`, `shared_cache_rebuilds_abandoned=1`,
`shared_cache_rebuild_pending=true`, no completed-rebuild event and `/health` 503.

A clause with no negative that can isolate it is not proven, and the first cut of
this rework had exactly that defect: the one-byte narrowing mutant reddened the
contract suite and left the whole production suite green, because every existing
seam keeps some object of the model tree alive and ownership refused first. Two
new seams fix it.

- **6f**, `--fault-inject-teardown-retain-weight-modules`: parks a strict subset
  of the tree. The residue lands *below* the footprint — what a released model
  looks like to a counter — and with the long request behind it `returned_bytes`
  clears the footprint too. Every byte-derived clause reads green, and the phase
  asserts each one; only ownership refuses.
- **6g**, `--fault-inject-teardown-retain-weight-arrays`: parks the flattened
  parameter arrays and no object at all. Ownership reports the model released and
  MLX still calls the whole footprint active; only the absolute residue refuses.

Requirement 3, the narrowed unit negative, is
`REVIEW REPRODUCTION F1d-R4: a request larger than the model paid the delta` —
container gone, owners all deallocated, model-sized residue, 608,909,592 B of
non-weight allocation gone since baseline. It must not reach `.rebuilt`, and the
composition test pins that the same reading cannot reach a completed rebuild
through `teardownOutcome`.

Requirement 4: the F1c-R3 behaviour, the outer-container retention timeout and
the TASK-260827-2h39ya `/health` 503 boundary are unchanged, asserted on all four
negative paths, and re-verified by `dead-generation-smoke.sh`.

## Separation, on MLX's own figures

All four rows have `container_deallocated=true` and `/health` 503.

| Run | live owners | observed active | returned | rebuilds / abandoned / pending |
| --- | ---: | ---: | ---: | --- |
| 6a clean | 0 of 316 | 2,720 | 301,226,372 | 1 / 0 / false |
| 6e whole model retained | 316 of 316 | 262,361,760 | **608,909,584** | 0 / 1 / true |
| 6f module subset retained | 158 of 316 | **174,944,928** | 696,326,408 | 0 / 1 / true |
| 6g weight arrays retained | **0 of 316** | 262,361,760 | 608,909,584 | 0 / 1 / true |

6e is the reviewer's bypass reproduced: the delta clause is satisfied and the
runtime refuses anyway. 6f is below the footprint with weights owned. 6g is the
mirror: nothing owned, model resident.

## Mutants — six, all die

| Mutant | Contract suite | Production smoke |
| --- | --- | --- |
| M1 ownership clause deleted (the revision-4 shape) | exit 1 | exit 1, **12 failures, only phase 6f**, attesting `shared_cache_rebuilds=1 cache_bytes=0` with 158 owners live |
| M2 **narrowing**, `<` relaxed to `<=` | exit 1 | exit 1, **12 failures, only phase 6g**, attesting with `observed_active_bytes=262,361,760` |
| M3 production registration call deleted | **exit 0, 197 green** | exit 1, 18 failures; phase 6a fails **closed**, 332,234,364 B of pool still held |
| M4 empty-registry guard removed | exit 1 | not run |
| M5 stability bound relaxed to one sample | exit 1 | not run |
| M6 in-flight veto deleted | exit 1 | not run |

M3 is the load-bearing one: a registry that is perfect and never populated
reports "no live owners", which reads exactly like a released model, and only
driving the production entry point catches it.

Pristine sources were restored and verified byte-for-byte by SHA-256 before the
final verification run.

## Verification actually run — real exit codes

| Command | Exit | Result |
| --- | ---: | --- |
| `xcodebuild build -configuration Release` | 0 | BUILD SUCCEEDED |
| `swift test -c release` | 0 | 197 tests / 19 suites (was 182 / 19) |
| `xcrun swift-format lint --recursive --strict Sources Tests` | 0 | no diagnostics |
| `shellcheck -S warning` on both smokes | 0 | clean |
| `scripts/generation-batch-recovery-smoke.sh` | 0 | 170 checks, 0 failures (was 115) |
| `scripts/dead-generation-smoke.sh` | 0 | 45 checks, 0 failures |
| `git diff --check` | 0 | clean |

## Known limits, stated rather than papered over

MLX's counters are process-global and no clause here can attribute an individual
byte. The gate is built to refuse everything it cannot account for: ownership is
read from the model tree rather than inferred from the allocator, the allocator
must be idle and at rest before it is believed, and anything model-sized is a
refusal. A false abandonment costs a supervision marker and a replacement
process; a false attestation tells an operator the host is free while a condemned
model is still holding it.

## Not done / for the orchestrator

Changes remain **uncommitted** in the story worktree —
`task-board.config.json -> version_control.confirm` is true and integration into
trunk is the orchestrator's step. Python `mlx-lm` remains the default local
runtime; no installed config changed. `/health` and `/v1/models` bodies are
unchanged, verified by the 2h39ya suite still passing.
