# STORY-260830-1a25ka: land-codex-fast-policy-on-current-trunk

## Description
Land the already validated Codex-only spawn policy on the exact current protected trunk after task-board fast_mode support is installed.

## Scope
agents-infra task-board.config.json only; preserve unrelated dirty checkout state and replay from fetched origin/main.

## Acceptance Criteria
Fresh Story workspace selects fetched origin/main; policy is spawn-policy-v4, exclusive Codex, exact gpt-5.6-sol/high, fast_mode true, all workload classes including unified aligned; schema preflight and independent review pass; delivery uses PR and merge.
