# STORY-260830-3uerxs: deliver-global-instruction-cleanup-on-clean-base

## Description
Deliver the removal of project-specific x-platform-airdrop and Tap2Cash material from the global instructions on a clean base, because the legacy story worktree carries an accepted foreign Change Request that is deliberately not checkpointed there.

## Scope
Delivery vehicle only. It exists because TASK-260824-1qm60c's accepted patch shares the legacy story worktree and is intentionally replayed elsewhere; nothing here touches that foreign delta.

## Acceptance Criteria
The instruction cleanup is delivered from a workspace containing only its own scope, with every removed passage preserved under a task-scoped artifact and the outcome statement matching exactly what was preserved.
