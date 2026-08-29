# TASK-260830-1l74m8: reduce-global-codex-sol-context-to-272k

## Description
Change the managed global Codex CLI config from the 1M Sol context override to a 272K window with 245K auto-compaction headroom.

## Scope
Only Codex config defaults, matching README/docs/tests, installed global sync, and board metadata for this task.

## Acceptance Criteria
.configs/codex-config.toml contains model_context_window=272000 and model_auto_compact_token_limit=245000 before any TOML table; README documents the new values and headroom rationale; installed ~/.codex/config.toml matches source after agents-infra setup global; Python tomllib parses source and installed config; relevant Go tests pass.
