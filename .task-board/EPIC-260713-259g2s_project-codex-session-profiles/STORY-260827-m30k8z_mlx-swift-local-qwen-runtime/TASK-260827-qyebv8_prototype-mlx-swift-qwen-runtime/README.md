# TASK-260827-qyebv8: prototype-mlx-swift-qwen-runtime

## Description
Build a task-scoped MLX Swift LM runtime prototype for the configured local Qwen model and map it onto the existing model-harness and Pi OpenAI-compatible contracts.

## Scope
Pin official MLX Swift and MLX Swift LM dependencies; load the exact configured Qwen model; implement or adapt models, chat completions, streaming, reasoning, tool-call, and lifecycle interfaces without changing the default runtime.

## Acceptance Criteria
The prototype loads the exact local model, completes bounded text and tool-call smokes through the managed contract, records unsupported model or tokenizer behavior explicitly, and leaves the Python mlx-lm runtime as the default rollback path.
