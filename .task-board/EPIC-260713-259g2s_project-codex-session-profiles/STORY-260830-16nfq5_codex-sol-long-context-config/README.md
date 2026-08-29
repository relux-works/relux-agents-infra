# STORY-260830-16nfq5: codex-sol-long-context-config

## Description
Update the shared global Codex CLI defaults to use GPT-5.6 Sol with the explicit long-context metadata requested by the owner.

## Scope
Source config .configs/codex-config.toml plus global setup/install verification only.

## Acceptance Criteria
Shared source config has top-level model = "gpt-5.6-sol", model_context_window = 1000000, and model_auto_compact_token_limit = 900000 before any TOML table; installed ~/.codex/config.toml reflects the same values; TOML parses successfully; unrelated trust/notice entries are preserved.
