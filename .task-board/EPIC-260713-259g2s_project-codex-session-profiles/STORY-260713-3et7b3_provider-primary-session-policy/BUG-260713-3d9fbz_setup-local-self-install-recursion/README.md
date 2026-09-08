# BUG-260713-3d9fbz: setup-local-self-install-recursion

## Description
agents-infra setup local recurses infinitely when PROJECT_DIR equals --source-dir (installing the agents-infra source repo into itself). It creates nested .agents/.agents/.agents/... directories until mkdir fails with file name too long. Repro: agents-infra setup local /path/to/relux-agents-infra --source-dir /path/to/relux-agents-infra --claude-yolo-mode=true. Workaround: pass --no-sync when only mutating project-config fields on the source repo itself. Fix idea: detect source-dir == project-dir (or source inside project) and skip/deny the sync copy step with a clear error. Found 2026-07-13 while rolling out claude yolo_mode.

## Scope
agents-infra setup local sync step; source-dir versus project-dir resolution.

## Acceptance Criteria
Running setup local with --source-dir resolving to the project directory itself, or to a directory containing it, is refused before any filesystem mutation, with an error naming both resolved paths and the reason. The refusal survives symlinks, relative paths, trailing slashes and case-insensitive path equality on macOS. A regression test drives the production setup entrypoint, not a helper, and is red when the guard is removed. No nested .agents directory is created on the refused path.
