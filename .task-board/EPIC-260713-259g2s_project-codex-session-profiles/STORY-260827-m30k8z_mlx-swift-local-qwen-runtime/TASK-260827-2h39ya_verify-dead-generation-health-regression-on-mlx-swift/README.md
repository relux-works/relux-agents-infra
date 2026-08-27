# TASK-260827-2h39ya: verify-dead-generation-health-regression-on-mlx-swift

## Description
Carry the mlx-lm generation-thread health regression into the MLX Swift runtime acceptance suite.

## Scope
Health and readiness semantics for a runtime whose generation worker has exited or become unable to serve requests.

## Acceptance Criteria
A deterministic fault kills or invalidates the Swift generation worker; the health endpoint returns HTTP 503 instead of 200; model-harness detects the failure and performs the configured supervised recovery; the healthy control remains HTTP 200.
