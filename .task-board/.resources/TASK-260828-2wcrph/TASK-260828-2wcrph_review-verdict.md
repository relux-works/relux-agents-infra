# Review verdict — TASK-260828-2wcrph

Change Request: `CR-TASK-260828-2wcrph-1`, revision 1  
Reviewed candidate tree: `5471e7986065f59b2188a3d6c197a48736bf1f6e`  
Verdict: **changes requested → analysis**

## Blocking finding

The report's supposedly corrected decode comparison, `8.88 tok/s` for llama.cpp versus `9.85 tok/s` for Python, is not supported by the attached corrected probe.

- `probe-ttft.py:66-72` emits `true_decode_tok_s` only when streamed usage contains `completion_tokens`.
- Both attached `probe-ttft-*.json` files contain `"usage": {}` and neither contains `true_decode_tok_s`.
- The published values reproduce exactly only by substituting non-empty SSE frame counts for token counts: `(103-1)/(83.496277-72.008933)=8.8793` and `(104-1)/(107.716383-97.254640)=9.8454`.
- An SSE frame count is a transport/chunking proxy, not an established token count. This is the `prove, or report unknown` negative-evidence shape.
- The attached llama.cpp runtime logs independently report `10.51`, `10.79`, and `10.74 tok/s` for the 106-token `long_prompt_8k` generation across the three suite runs. Those are not directly interchangeable with the intended client-side metric, but they demonstrate that the evidence does not establish even the claimed direction, “llama.cpp is ~10% slower.”

Consequently these candidate claims are unsupported and must not ship as facts:

- `.research/260829_llamacpp-against-the-python-baseline.md:250-267` (`8.88`, `9.85`, `0.90x`, “real deficit”, `9.1x`)
- `tools/mlx-swift-runtime-prototype/README.md:620-621`
- `LOGBOOK.md:12`

The same report also contradicts itself at `.research/260829_llamacpp-against-the-python-baseline.md:287-288`, attributing the soak result to a runtime that “decodes far faster” after claiming corrected decode is slower. The soak wall-clock result is real, but prefill, prefix-cache reuse, output-token count, and scenario order confound attribution; it does not prove faster decode.

## Required rework

1. Re-run the corrected first-token/decode probe with authoritative completion-token counts for both runtimes (for example streamed usage explicitly requested and verified), retain the raw evidence, and derive the rate only when the count is present. If that cannot be established, report decode as `unknown`; do not infer tokens from SSE frames.
2. Revise the research report, README, and LOGBOOK to remove or replace the unsupported exact decode numbers and the “decodes far faster” attribution.
3. Keep the known gate defect explicitly excluded from any decision. It is acceptable for `TASK-260829-3cwcb6` to own the production fix and rerun, but not for this interim artifact to replace a known-wrong gate number with an unproved proxy number.
4. Attach revised task-scoped report/evidence and publish a new Change Request revision for another reviewer cycle.

## What review independently confirmed

- The three pinned invocations were sequential on the same `MacBookPro18,2`; each run log has `(none)` before and after the model-process sweep. The reviewer audited the attached raw runs and did not rerun the roughly hour-scale 28 GB suite.
- All three production invocations exited `4` on the exact `contextPolicy` mismatch and wrote no scored decision. The context-bound negative tests refuse the Python unbounded baseline against llama.cpp bounds across 512…262144.
- Both runtimes succeeded at the pinned `context_75k` scenario's actual 73,016 prompt tokens with exact prompt-token parity. Wall-clock ratios reproduce from the records, including `0.81x` on `long_prompt_8k` and `0.97x` at capacity.
- The speculation defect and fix reproduce independently: the old placement would ignore top-level `speculative` once `params` exists; used-slot evidence shows top-level `false` under `--spec-type none` and `true` under `ngram-mod`, while `params` is present only on a served slot and does not contain the field. Both post-fix attestations record `observedSpeculation={state:reported,active:false}`; the pre-fix attestation is `unread`. The production call site is `BenchmarkRunCommand.speculationAnswer` → `RuntimeSpeculation.read`.
- The declared model non-equivalences travel with every examined record and attestation.
- The memory defect reproduces: one llama process at one moment reports `ri_phys_footprint=1.41 GiB`, `ps RSS=28.09 GiB`, and `vmmap` mapped-file resident `26.6 GiB`, dirty `0`. The `context_75k` arithmetic reproduces as a conservative `41.05 GiB` llama.cpp upper bound versus `45.20 GiB` Python. Calling small working sets indeterminate is an honest refusal, not a hidden loss.
- A defect biased against llama.cpp was found: omitting `reasoning_content` inflates llama.cpp TTFT and understates its prefill rate, even while it overstates decode. Therefore the premise that all three defects only flatter llama.cpp is false; this defect has mixed directional bias.

## Review validation

- `swift test --filter RuntimeObservationReadingTests`: 21 tests passed.
- `swift test --filter RuntimeBenchmarkContextBoundTests`: 19 tests passed.
- `swift test`: 392 tests / 30 suites passed.
- Production-entry `scripts/benchmark-gate-smoke.sh`: 0 failures after resolving the required `BINARY`; the initial readiness invocation correctly failed before testing because `BINARY` was unset and is preserved separately.
- `git diff --check` on the exact CR range: passed.

Review logs are under `.temp/review-TASK-260828-2wcrph/` in the Story worktree.
