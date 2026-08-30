# TASK-260830-2hc5r2: bound-python-baseline-kv-for-comparability

## Description
Implement a real bounded KV window in the pinned Python mlx-lm baseline, including qwen3_5 cache construction, so the benchmark pair stops being refused on contextPolicy and the memory criterion becomes a runtime comparison rather than a policy comparison.

## Scope
The relux-works/mlx-lm fork at /Users/alexis/src/relux-works/mlx-lm plus whatever agents-infra profile spelling is needed to pass the bound. Does not change the deployed default profile.

## Acceptance Criteria
The pinned Python server accepts and honours a 76800-token KV bound through qwen3_5 cache construction, the live attestation derives kv=76800, and the benchmark pair's contextPolicy pins match so the gate no longer refuses on that dimension.
