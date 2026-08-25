# STORY-260825-1r7z9o: shared-local-qwen-runtime-leases

## Description
Allow an orchestrator to launch multiple independent tracked Pi agents that automatically share one agents-infra-owned verified local Qwen runtime.

## Scope
Design and implement a local shared-runtime broker whose lifetime is independent of any invoking terminal or child process. Separate task-board/agent orchestrator spawn-runners in distinct process groups must acquire leases automatically when launching the same managed Qwen profile, reuse one verified loopback model server, keep Pi sessions/state isolated, and release leases on tracked child termination. The broker must reject arbitrary or identity-mismatched listeners and stop/reap the runtime only after the final tracked lease expires or an operator explicitly stops it.

## Acceptance Criteria
An orchestrator can issue two independent tracked agent spawns, each launching its own Pi process/session through the normal spawn path, and both overlap while using exactly one verified Qwen model-server PID. The proof must not rely on one parent shell or a manual pi-and-pi script. Broker lifetime, lease recovery, final-release cleanup, status, explicit stop, and negative identity/concurrency tests are documented and pass.
