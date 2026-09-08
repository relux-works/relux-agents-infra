# TASK-260830-18n40a: evaluate-mtplx-as-mlx-native-mtp-engine

## Description
Evaluate MTPLX as a fourth local inference engine: MLX-native with built-in multi-token-prediction speculative decoding, which is precisely the capability the MLX 8-bit build lost and llama.cpp retained.

## Scope
Engine evaluation under the adapter contract. https://github.com/youssofal/MTPLX — MLX-native, Apple Silicon first, OpenAI-compatible on 127.0.0.1:8000, plus Anthropic-compatible /v1/messages; targets Qwen 3.5/3.6/3.8 and Gemma 4; claims 1.6x-2.24x from native MTP with exact rejection sampling.

## Acceptance Criteria
MTPLX enters the existing comparison on identical terms rather than as a separate study: the same six pinned scenarios, the same comparison gate with no admission clause weakened, the same units, and the same weighting where peak resident memory and decode throughput are equal and decode sometimes dominates. It is measured twice — first with --no-mtp so it is comparable against the runtimes already studied with speculation off, then with MTP on as a separate configuration — and the memory delta between the two is reported alongside the throughput delta, because speculative decoding holds draft state in addition to the KV cache. The vendor's own figures (1.6x on an M4 mini, 2.24x on an M5 Max, 14.4 to 23.0 tok/s at depth 1) are recorded as unverified third-party claims and never substituted for a measurement on this host. The claim of identical output distribution under exact rejection sampling is tested directly rather than inferred from throughput, since its failure would be invisible in a speed benchmark. Every non-comparable dimension is refused rather than scored. The evaluation records, as an explicit migration-risk item, that MTPLX does not depend on mlx-lm and therefore does not inherit the pinned fork's bounded-KV fix.
