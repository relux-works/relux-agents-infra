# TASK-260810-3833d1: scaffold-project-local-instruction-space

## Description
Stop copying shared global instruction modules into project-local runtimes while retaining the local instruction directory and provider rendering topology.

## Scope
agents-infra local setup, instruction rendering, setup verification, tests, and operator documentation. Global setup remains unchanged.

## Acceptance Criteria
Local setup creates project-owned .agents/.instructions without shared global policy content; existing handwritten project instructions are preserved; repeated local setup is idempotent; Codex and Claude runtime entrypoints remain valid; global setup behavior is unchanged; setup and verification tests pass.
