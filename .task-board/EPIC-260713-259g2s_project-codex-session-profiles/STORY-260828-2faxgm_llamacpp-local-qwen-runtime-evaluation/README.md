# STORY-260828-2faxgm: llamacpp-local-qwen-runtime-evaluation

## Description
Evaluate llama.cpp with equivalent-quantization GGUF weights as a candidate local Qwen inference backend, measured against the pinned Python mlx-lm baseline with the same comparison harness used for the MLX Swift evaluation.

## Scope
Local Qwen inference backend selection in agents-infra. Same host, same comparison harness, same pinned prompt suite and thresholds as the MLX Swift evaluation. Python mlx-lm remains default and rollback until an accepted replacement wins on evidence.

## Acceptance Criteria
llama.cpp is measured against the pinned Python mlx-lm baseline through the same comparison harness, with quantization equivalence explicitly established or declared non-comparable, and a migration decision is recorded with memory economy scored as a first-class criterion.
