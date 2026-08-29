# STORY-260830-21k9pi: codex-sol-272k-context-config

## Description
Reduce the shared global Codex CLI GPT-5.6 Sol context override to the short-context pricing boundary with conservative auto-compaction headroom.

## Scope
Source config .configs/codex-config.toml plus README, setup tests, and global install verification only.

## Acceptance Criteria
Shared source config has model_context_window = 272000 and model_auto_compact_token_limit = 245000 as top-level TOML keys before any table; installed ~/.codex/config.toml reflects the same values; GPT-5.6 Sol remains selected; TOML parses; trust/notice entries are preserved.
