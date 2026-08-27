# TASK-260827-2q77g8: verify-generation-batch-failure-recovery-on-mlx-swift

## Description
Carry the mlx-lm generation-batch exception recovery regression into the MLX Swift runtime acceptance suite.

## Scope
In-flight request failure, batch and cache cleanup, worker survival, and the next request after a generation or Metal failure.

## Acceptance Criteria
A deterministic generation-batch failure terminates the affected request with an explicit error, releases or rebuilds invalid batch and cache state, keeps or recreates a serving generation worker, and allows a subsequent request to complete without restarting a healthy process; unrecoverable worker death is reflected by health as 503.
