# TASK-260827-2v13w8: benchmark-and-decide-mlx-swift-runtime-migration

## Description
Benchmark the MLX Swift prototype against the pinned Python mlx-lm baseline and produce the migration and rollback decision.

## Scope
Same host, model, quantization, prompts, context policy, output bounds, and model-harness measurement contract. Compare TTFT, prefill throughput, decode throughput, peak memory, 75k context capacity, tool-call parity, and long-running stability.

## Acceptance Criteria
A reproducible comparison report records exact revisions and commands; the Swift runtime becomes eligible for the default profile only when required compatibility and regression tasks pass and performance, memory, and stability are no worse than the accepted thresholds; otherwise the Python runtime remains default with concrete blockers recorded.
