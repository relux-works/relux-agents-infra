# STORY-260827-m30k8z: mlx-swift-local-qwen-runtime

## Description
Migrate the managed local Qwen execution backend from Python mlx-lm to an MLX Swift LM-based runtime after measured compatibility, capacity, and reliability validation.

## Scope
Local Qwen runtime and model-harness integration in agents-infra. Preserve Pi profile and session semantics, remote/local mode boundaries, and the Python runtime as a rollback path until acceptance.

## Acceptance Criteria
An MLX Swift LM migration path is specified, prototyped, benchmarked against the current Python mlx-lm baseline, and accepted only with tool-calling, reasoning, health, lifecycle, context-capacity, and rollback parity evidence.
