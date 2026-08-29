# TASK-260830-3j93bl: update-global-codex-sol-context-config

## Description
Set the managed global Codex CLI config to GPT-5.6 Sol and explicit 1M/900K context limits, then sync the installed runtime config.

## Scope
Only the shared Codex config source and task-board metadata for this task.

## Acceptance Criteria
.configs/codex-config.toml contains the requested root keys before any TOML table; installed ~/.codex/config.toml resolves to the synced managed config with the same root keys; Python tomllib parses the source and installed config; task notes record any pre-existing validation debt.
