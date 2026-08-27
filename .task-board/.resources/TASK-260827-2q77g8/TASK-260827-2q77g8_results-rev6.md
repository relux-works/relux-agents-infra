# TASK-260827-2q77g8 — revision 6 results

Round-6 rework for **F1e-R5**: a strict subset of copied weight arrays reached
`.rebuilt` with 255,724,192 B of the condemned model still active.

## The directive, and what was done about it

The orchestrator's instruction was to stop moving the threshold and collapse the
class. That is what this revision does, and the answer is uncomfortable enough
that it is stated first rather than buried.

### The maximum admissible residue is now **0 bytes**

`GenerationBatchRecovery.residualNonWeightAllowanceBytes = 0`, and the clause is

```swift
guard observation.activeBytes <= residualNonWeightAllowanceBytes else { return false }
```

### Why nothing weight-sized can hide beneath it

Nothing of *any* size can hide beneath zero, which is the point — the argument
needs no claim about tensor sizes, checkpoint shapes or this model in
particular.

MLX's allocator counters are process-global. `Memory.snapshot().activeMemory` is
one number for the whole process and cannot attribute a byte to an owner. So any
allowance `A > 0` is a promise that nothing weight-sized fits underneath it, and
that promise is unkeepable: there is no reading that separates `A` bytes of
sampler state from `A` bytes of retained weights. Five rounds of review are the
empirical form of the same statement — each round narrowed the band and each
round the reviewer found a production input inside the new one:

| Revision | Residue clause | The input review put inside it |
| --- | --- | --- |
| 3 | wrapper `weak`-`nil` | delayed `SerialAccessContainer` destruction; whole model resident |
| 4 | `baseline - active >= footprint` | 6,000-word prompt; the request's own KV paid the delta |
| 5 | `active < footprint` | strict subset of copied parameter arrays, 255,724,192 B |

At `A = 0` the question does not arise. `activeMemory == 0` means **no MLX
buffer of any kind is alive in the process**, so no *weight* buffer is alive in
it either. It is the one reading a process-global counter can carry an
attribution claim on, because it leaves nothing to attribute.

### The honest consequence: a clean rebuild is essentially never attested

Recorded plainly, as the brief asks. This runtime's own **clean** condemned
teardown leaves **2,720 B** of post-generation MLX state (sampler and RNG
arrays) active — measured, not assumed, in phase 6a of the acceptance suite with
every other clause green:

| Reading | Clean teardown, revision 6 |
| --- | ---: |
| container deallocated | `true` |
| registered / live `Module` owners | `316 / 0` |
| generations in flight / stable samples | `0 / 302` |
| `weight_footprint_bytes` | `262,361,760 B` |
| `returned_bytes` | `303,847,812 B` |
| `observed_active_bytes` | `2,720 B` |
| `residual_non_weight_allowance_bytes` | `0 B` |
| verdict | **abandoned** |

2,720 is not 0, so the clean path abandons like every other condemned teardown.
`.rebuilt` is not dead code — it is reachable, and the unit suite drives it — but
under MLX's process-global counters this runtime does not reach it in
production. **That is the accepted outcome, not a defect to engineer around**,
and the only way to engineer around it is to re-open a band.

### The cost, asserted rather than described

Refusing to attest means refusing to clear, so the pool is left holding the
freed model: phase 6a measures **331,887,724 B** of `cache_bytes` after the
abandonment. That is payable only because the abandonment re-announces the
supervision marker — the supervisor replaces a process condemnation had already
made unusable, and the host gets everything back. A false abandonment costs a
restart that was going to happen anyway; a false attestation tells an operator
the host is free with a condemned model still on it.

**Considered and rejected:** an opportunistic `Memory.clearCache()` on the
abandoned path. It would return the pool on the clean path at the cost of
splitting "cleared" from "attested" into two states an operator has to reason
about — in the exact area where five rounds of review found conflation. Phase 6a
asserts the cost instead, and mutant **P3** proves that assertion is not
vacuous.

## Required rework, item by item

1. **Ownership is a veto, never proof.** Documented in the gate, in the barrier
   and in the README, and enforced by test: the reading that revision 5 accepted
   (owners gone, half-footprint residue) is now an explicit negative, and the
   ownership clause is still proved attributable by moving *only* the owner
   count on a reading the residue clause accepts.
2. **The narrowed production phase exists.** New seam
   `--fault-inject-teardown-retain-weight-array-subset` holds the largest half
   of the flattened parameter arrays by `nbytes`. New acceptance phase **6h**
   reproduces the reviewer's figure to the byte and requires abandonment:

   | Measurement | Phase 6h, revision 6 |
   | --- | ---: |
   | registered / live `Module` owners | `316 / 0` |
   | container deallocated | `true` |
   | generations in flight / stable samples | `0 / 302` |
   | `weight_footprint_bytes` | `262,361,760 B` |
   | `observed_active_bytes` | `255,724,192 B` |
   | `returned_bytes` | `615,547,160 B` |
   | `residual_non_weight_allowance_bytes` | `0 B` |
   | event / rebuilds / abandoned / pending | `abandoned` / `0` / `1` / `true` |
   | `/health` | `503` |

   The phase asserts every other clause green first, so the refusal is
   attributable to the residue alone, and asserts the residue is *below* the
   footprint and *above* a weight-sized floor — otherwise it would be phase 6g
   under another name.
3. **The defect-encoding unit expectations are gone.** `footprint - 1` and the
   half-footprint acceptance are deleted; both are now negatives. The
   fail-closed contract is proved through the production call site
   (`GenerationEngine.deinit` → `GenerationBatchLedgerStore.completeWorkerTeardown`)
   by phases 6a and 6h, and mutant P1 shows the production path reddening at
   that exact call site.
4. **Preserved.** F1d-R4 (6e), F1c-R3 (6d), the outer-container timeout (6c),
   recoverable failure and next-request behaviour (1–5c), and the
   TASK-260827-2h39ya `/health` 503 boundary (`dead-generation-smoke.sh`, 45/45).
5. **LOGBOOK.md** carries the corrective entry (`2026-08-28 1120`), which
   explicitly retracts the 0745 claim that the absolute-residue clause was the
   fix.

## What changed

- `GenerationBatchRecovery.swift` — `residualNonWeightAllowanceBytes = 0`;
  clause 6 rewritten from footprint-relative to allowance-relative; the "what
  this cannot do" note now states the never-attested consequence.
- `BatchLedgerStore.swift` — `observed_active_bytes` and
  `residual_non_weight_allowance_bytes` published on the abandoned and deferred
  events as well as the completed one, so an acceptance run can separate a
  residue refusal from an ownership or clock refusal.
- `RuntimeOptions.swift`, `Main.swift` — fifth retention seam
  `--fault-inject-teardown-retain-weight-array-subset`, refused without an
  injection like its four siblings.
- `generation-batch-recovery-smoke.sh` — phase 6a rewritten from "attests a
  rebuild" to "abandons, and here is why and what it costs"; new phase 6h;
  `CONDEMNED_CACHE_CEILING` removed (the ordering it guarded is moot once the
  condemned path never clears).
- `GenerationEngine.swift` docs, tool `README.md`, `LOGBOOK.md`.

## Verification actually run (real exit codes)

| Gate | Result |
| --- | --- |
| `xcodebuild` Release | exit 0, BUILD SUCCEEDED |
| `swift test -c release` | exit 0, 204 tests / 19 suites |
| `scripts/generation-batch-recovery-smoke.sh` | exit 0, 193 checks, 0 failures, 15 phases |
| `scripts/dead-generation-smoke.sh` | exit 0, 45 checks, 0 failures |
| `swift-format lint --recursive --strict Sources Tests` | exit 0, no diagnostics |
| `shellcheck -S warning` on both smoke scripts | exit 0 |
| `git diff --check` | exit 0 |

Both smokes ran on the real Release binary under the real `model-harness` with
the real 0.5B model, after a full rebuild from restored pristine sources
(`GenerationBatchRecovery.swift` SHA-256
`9ba96ce4d7a2c2003813ad3b7d2f3b1343879be69b2dd9d3776b89183480b81d`,
`BatchLedgerStore.swift` SHA-256
`8d91fc75183fb95ef84798c30eb184920ff75b391cb109c19459289f58ae5c7c`).

## Mutants — seven, and the two that separate the intervals

| Mutant | Change | Result |
| --- | --- | --- |
| **P1** (production) | restore revision 5's gate, `activeBytes < weightFootprintBytes` | smoke exit 1, **32 failures**, including `SUBSET GATE: a rebuild was attested with 255,724,192B of weights still resident` — review's finding measured at the production call site |
| **P2** (production, NARROWING) | allowance raised `0 → 4,096` | smoke exit 1, **19 failures**, all in the clean-path phase; phase 6h's verdict checks stay **green**, proving 6a and 6h cover different intervals and that the bound is proved by tightening, not only by deletion |
| **P3** (production) | `Memory.clearCache()` on the abandoned path, without attesting | smoke exit 1, **exactly 1 failure**, the cost gate — so that assertion is not vacuous |
| M1 (contract, NARROWING) | allowance `0 → 4,096` | `swift test` exit 1; reddens the allowance test and the clean-path residue test |
| M2 (contract) | restore revision 5's gate | `swift test` exit 1; reddens the F1e-R5 reproduction, the allowance bound, the ownership-veto test |
| M3 (contract) | residue clause deleted | `swift test` exit 1; reddens F1d-R4 and F1e-R5 both |
| M4 (contract) | ownership clause deleted | `swift test` exit 1; reddens the ownership-veto and empty-registry tests |

Every mutant was reverted and the tree re-verified by SHA-256 before the final
pristine build and acceptance runs above.
