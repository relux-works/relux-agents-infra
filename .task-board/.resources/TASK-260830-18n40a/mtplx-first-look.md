# MTPLX — first look (not the tracked evaluation)

Source: https://github.com/youssofal/MTPLX, read 2026-08-30. Vendor claims are
quoted, not endorsed. Nothing here was measured on our host.

## What it is

Native macOS app plus CLI for local models on Apple Silicon, with multi-token
prediction. Python 3.11+, builds on MLX. Apache-2.0 "with attribution
requirements". ~1.8k stars, CI badge present, 28 open issues / 46 PRs.

Serves an OpenAI-compatible HTTP API on 127.0.0.1:8000: `/v1/chat/completions`,
`/v1/completions`, `/v1/models`, `/v1/embeddings`, `/v1/rerank`, plus an
Anthropic-compatible `/v1/messages`.

Install: `brew install youssofal/mtplx/mtplx` or `python3 -m pip install mtplx`.
Serve: `mtplx serve --port 8000`.

## The finding that matters most for us

**It does not depend on `mlx-lm`; it implements its own inference path.**

That collides directly with the standing constraint that the pinned `mlx-lm`
fork is held until an accepted replacement carries its fixes. Our bounded-KV
work (STORY-260830-2vrhg1) lives in that fork. Adopting MTPLX would not inherit
it — the KV bound would have to be re-established, and re-proven, against a
different codebase. That is a migration-risk input, not a blocker, but it must
be priced into any GO decision rather than discovered afterwards.

## Fit with our model

- Requires models with built-in MTP heads. "Modern models like Qwen 3.5/3.6/3.8
  ship with built-in MTP heads." Our Qwen 3.8 27B qualifies.
- No silent fallback for models without heads: "an MTP launch is rejected before
  weights load." Fail-closed, which is the behaviour we want.
- Qwen 3.8 27B variants: Bare Speed and Optimized Speed are 4-bit dynamic quant;
  **Optimized Quality is 8-bit dynamic quant**, matching our current 8-bit.
- `--no-mtp` runs autoregressive-only. This is what makes a fair comparison
  possible at all: our gate requires speculation off on both sides, and MTPLX
  can be measured that way and then separately with MTP on.

## Vendor performance claims — unverified, and structurally suspect for us

- "1.6x faster on a 16 GB M4 Mac mini and 2.24x on an M5 Max"
- "14.4 tok/s baseline becomes 23.0 tok/s" — 16 GB M4, 9B model, depth 1
- "227.1 to 296.1, 1.30x" for Forge

None of these are on our host, our model size, our quantization, or our harness.
They are exactly the shape of number our comparison gate exists to refuse:
measured by the interested party on its own instrumentation. They justify
running the evaluation; they cannot substitute for it.

## Correctness claim needing independent proof

"Same model, same output distribution"; at temperature 0.6 / top_p 0.95 it
"behaves exactly like normal decoding, just faster", via "exact rejection
sampling with residual correction" following the Leviathan and Chen theorem.

If true this is a strong result. It is also the single claim whose failure would
be least visible in a throughput benchmark, so the evaluation must test output
distribution equivalence directly, not infer it from tokens per second.

## Memory — the open question

Guidance is coarse: "16 GB of memory runs the 4B and 9B models comfortably";
"Qwen 3.8 Optimized Speed is recommended on Macs with 32 GB or more". No
per-variant footprint for the 27B models is stated.

Our host holds one 28 GB model at a time and peak resident memory is weighted
equally with decode in the pending decision, so an unstated footprint is not a
detail. Speculative decoding also holds draft state in addition to the KV cache,
so the MTP-on configuration should be expected to cost more memory than
MTP-off, and that delta has to be measured rather than assumed.

## Suggested scope when the tracked evaluation runs

1. Measure with `--no-mtp` first, on the same six pinned scenarios, so it enters
   the existing comparison on equal terms.
2. Then measure with MTP on as a separate configuration, reporting the memory
   delta alongside the throughput delta.
3. Test output-distribution equivalence directly.
4. Record the `mlx-lm` independence as an explicit migration-risk item.
