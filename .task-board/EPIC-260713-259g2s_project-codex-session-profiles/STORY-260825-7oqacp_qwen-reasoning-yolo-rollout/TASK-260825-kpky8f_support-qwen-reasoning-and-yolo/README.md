# TASK-260825-kpky8f: support-qwen-reasoning-and-yolo

## Description
Establish the pinned Pi runtime native contracts for Qwen reasoning and unattended tool execution, then implement source-of-truth agents-infra support with exact validation, provenance, tests, and documentation.

## Scope
Inspect the installed pinned Pi CLI and repository-owned compatibility assumptions; extend canonical qwen/pi target and managed Pi launch composition so target reasoning=medium maps to the real Pi thinking mechanism. Add agents.pi.primary_session yolo_mode only if it maps to a real native Pi execution policy. Unsupported or contradictory configurations must fail explicitly before launch.

## Acceptance Criteria
A non-launching qwen-infra plan reports reasoning medium and its source; launch argv/config forwards the correct Pi-native reasoning selection; yolo_mode=true has tested real native semantics or is rejected with a precise unsupported-capability error; existing direct Pi behavior and safe defaults remain compatible; focused Go tests and documentation checks pass.
