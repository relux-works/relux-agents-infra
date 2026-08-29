# TASK-260828-3g87i4: acquire-llamacpp-and-equivalent-quantization-weights

## Description
Install a pinned llama.cpp build and stage equivalent-quantization Qwen GGUF weights under src/local-models, establishing and recording how the GGUF quantization relates to the MLX 8-bit group64 baseline.

## Scope
(define task scope)

## Acceptance Criteria
A pinned llama.cpp revision is built or installed, equivalent-quantization Qwen GGUF weights are staged under src/local-models, and the relationship between the GGUF quantization and the MLX 8-bit group64 baseline is recorded explicitly, including where it is not equivalent.
