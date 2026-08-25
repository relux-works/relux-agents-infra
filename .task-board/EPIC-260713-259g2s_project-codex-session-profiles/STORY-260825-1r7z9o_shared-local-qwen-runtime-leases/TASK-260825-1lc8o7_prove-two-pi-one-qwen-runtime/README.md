# TASK-260825-1lc8o7: prove-two-pi-one-qwen-runtime

## Description
Document and prove two parallel Pi agent processes using one verified Qwen MLX runtime.

## Scope
Add operator documentation and a bounded production acceptance harness driven by the real orchestrator spawn surface. Launch two independent tracked Pi agent runs with distinct RUN handles/process groups against the same Qwen profile; capture sanitized evidence that their executions overlap and both complete while exactly one broker-owned MLX server PID exists; then terminate/release both and prove bounded cleanup.

## Acceptance Criteria
README and skill docs expose the orchestrator workflow and safety boundary. Automated negative tests pass. A real smoke records two distinct tracked RUN IDs, two Pi PIDs/session identities, different parent process groups, one Qwen runtime PID, overlapping execution, successful independent outputs, and confirmed cleanup after the final run. A single-shell pi-and-pi demonstration does not satisfy acceptance.
