# TASK-260827-2q77g8 — Review Verdict, Revision 6

## Verdict

**Accepted.** Candidate tree `b65a1f6089fe6f5db7097658fff9b80b4d1b974f` satisfies the round-6 directive and the task acceptance criteria. No false teardown-release attestation was reproduced on the production path.

## Scope reviewed

- Base: `08beb4052faa851e686705677e24a20c2397ad87`
- Candidate tree: `b65a1f6089fe6f5db7097658fff9b80b4d1b974f`
- All 15 changed working files were byte-compared with the candidate tree and matched.
- The repository was not modified by this reviewer; only ignored `.temp/` evidence and board resources were produced.

The round directive deliberately accepts increased abandonment, a narrow/rare clean rebuild path, and the inability of MLX process-global counters to attribute non-zero bytes. I did not treat those consequences as findings. The review searched only for a production-path false attestation: weight-sized live residue reported as a completed release.

## Gate attack and result

The composed production path is:

`GenerationEngine.generate` → `GenerationBatchLedgerStore.fail` → `GenerationEngine.deinit` → `WeightReleaseBarrier.waitForRelease` → `GenerationBatchRecovery.weightsReleased` → `GenerationBatchLedgerStore.completeWorkerTeardown`.

Caller search found no bypass around this path for a pending condemned teardown. `completeWorkerTeardown` recomputes `releaseObserved` from the full observation; it does not accept caller-minted Boolean evidence. Its `.rebuilt` branch is reachable only when `weightsReleased` accepts. Revision 6 requires `activeBytes <= residualNonWeightAllowanceBytes`, and the allowance is exactly `0`. The production observation reads MLX `Memory.snapshot().activeMemory`, backed by a non-negative `size_t` active-memory count. A live MLX weight buffer therefore produces a positive reading and cannot enter the completed-release branch.

I attempted the strongest maintained production input, F1e-R5 array-subset retention, against a freshly built Xcode product. It produced every non-residue clause as green while keeping weight-sized residue live:

| Observation | Result |
| --- | ---: |
| Container deallocated | `true` |
| Registered/live owners | `316 / 0` |
| Generations in flight | `0` |
| Stable samples | `302` |
| Weight footprint | `262,361,760 B` |
| Returned bytes | `615,547,160 B` |
| Live residue | `255,724,192 B` |
| Allowance | `0 B` |

The residue was both weight-sized and strictly below the footprint, exactly the interval revision 5 admitted. Revision 6 emitted no `generation_shared_cache_rebuilt`, kept `shared_cache_rebuilds=0`, recorded one abandonment, kept the rebuild pending, released the batch, returned HTTP `500` for the affected request, and reported `/health` `503`. This is the required refusal.

I could not construct any production input that reaches `.rebuilt` with weight-sized residue live. The zero allowance collapses the class; no positive-residue interval remains.

## Required regression confirmations

All were rerun on the freshly built production binary with the local Qwen1.5 0.5B MLX model:

| Boundary | Independent result |
| --- | --- |
| Recoverable failure and next request | Faulted request returned explicit `500` with no partial completion; batch released; `/health` stayed `200`; next request succeeded on the original process; exactly one listener bound. |
| Outer-container timeout (6c) | `262,361,760 B` remained active; `release_observed=false`; no rebuild; abandoned and pending; batch released; `/health 503`. |
| F1c-R3 inner retention (6d) | Container deallocated while `262,361,760 B` stayed active; no rebuild; abandoned and pending; batch released; `/health 503`. |
| F1d-R4 long context (6e) | `608,909,584 B` returned against a `262,361,760 B` footprint while the full `262,361,760 B` model remained active; no rebuild; abandoned and pending; batch released; `/health 503`. |
| TASK-260827-2h39ya health boundary | Every unrecoverable condemned-worker production phase above reported `/health 503`; the supervision marker remained present. |

## Validation

- `swift test -c release` — passed, 204 tests in 19 suites.
- `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData -skipPackagePluginValidation -skipMacroValidation` — `BUILD SUCCEEDED`.
- `swift format lint --recursive --strict Sources Tests Package.swift` — passed.
- `bash -n scripts/generation-batch-recovery-smoke.sh` — passed.
- `git diff --check 08beb4052faa851e686705677e24a20c2397ad87 b65a1f6089fe6f5db7097658fff9b80b4d1b974f` — passed.
- Production smoke slices for recovery, 6c, 6d, 6e, and 6h — passed with zero failures.

One initial reviewer-only 6h slice omitted the full suite's earlier `write_long_body` prerequisite, so the request never reached the runtime (`started=0`, `/health 200`). It was discarded as invalid orchestration evidence. The corrected recorded slice explicitly invoked the existing prerequisite and passed; no candidate failure was hidden or accepted from that invalid attempt.

## Acceptance rationale

The affected request fails explicitly, invalid batch state is released, recoverable failures leave the healthy worker serving and allow the next request, condemned failures remain terminal and visible as HTTP 503, and every reproduced prior bypass now fails closed. The implementation fits the existing runtime ownership: request accounting remains in the ledger, serving readiness remains in `RuntimeState`, and deferred teardown is owned by engine deallocation. Revision 6 is accepted without findings.
