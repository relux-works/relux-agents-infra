# TASK-260826-vrvauf: set-src-qwen-context-window-50k

## Description
Set the inherited /Users/alexis/src Pi profile context window to 50000 without changing other model policy.

## Scope
/Users/alexis/src/.agents/.configs/project-config.toml only; verify through qwen-infra static resolution.

## Acceptance Criteria
The generalized Qwen profile reports context_window 50000 and all other inherited model, reasoning, endpoint, and yolo settings remain unchanged.
