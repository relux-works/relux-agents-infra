# STORY-260830-30vtvt: deliver-lldb-mcp-setup-on-clean-base

## Description
Deliver the LLDB MCP setup prerequisites and wrapper idempotency fixes on a clean base, because the legacy story worktree carries an accepted foreign Change Request that is deliberately not checkpointed there.

## Scope
Delivery vehicle only. It exists because TASK-260824-1qm60c's accepted patch shares the legacy story worktree and is replayed elsewhere; nothing here touches that foreign delta.

## Acceptance Criteria
The LLDB MCP setup work is delivered from a workspace containing only its own scope: missing prerequisites fail clearly, a second run is a no-op, and wrapper health is established by delegation behaviour rather than by the presence of a line.
