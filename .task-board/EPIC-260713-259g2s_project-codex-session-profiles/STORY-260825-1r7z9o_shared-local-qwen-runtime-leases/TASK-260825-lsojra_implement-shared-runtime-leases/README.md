# TASK-260825-lsojra: implement-shared-runtime-leases

## Description
Implement the reviewed shared local-Qwen runtime owner and multi-client lease protocol.

## Scope
Implement the reviewed terminal-independent broker and automatic child-launch lease integration. Reuse canonical target/profile resolution and managed runtime identity checks. Each tracked spawn gets its own Pi process/session/state, while same-profile spawns across independent spawn-runner process groups reuse one broker-owned MLX runtime. Add crash-safe lease cleanup, status, bounded explicit stop, and typed negative paths.

## Acceptance Criteria
The normal orchestrator tracked-spawn path supports concurrent shared-runtime leases with single-flight startup, verified reuse, final-release cleanup, and operator status/stop. Tests cover independent parents, races, crashes, stale RUN/PID leases, mismatched profile/model/argv, unrelated listeners, and no premature shutdown while another tracked Pi lease is live.
