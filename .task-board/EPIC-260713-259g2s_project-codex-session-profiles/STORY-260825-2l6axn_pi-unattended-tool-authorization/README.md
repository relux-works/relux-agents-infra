# STORY-260825-2l6axn: pi-unattended-tool-authorization

## Description
Establish a standalone unattended Pi launch contract for qwen-infra so independent local-Qwen workers can be spawned now and integrated into task-board later.

## Scope
Implement only the agents-infra/Pi spawn boundary: non-interactive independent worker launches, shared local-model runtime reuse, deterministic tool authorization, no human approval prompts, and no unrestricted root authority. task-board runtime registration and adapter integration are explicitly deferred.

## Acceptance Criteria
qwen-infra can launch multiple independent non-interactive Pi workers against the shared local-Qwen runtime; authorized tool calls execute without UI prompts; project trust is not misrepresented as tool authorization; extension discovery and tool exposure are deterministic and fail closed; no task-board integration is claimed or required in this Story.
