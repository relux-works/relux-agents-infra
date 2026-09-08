# BUG-260830-1lyo9m: disable-codex-fast-mode-in-infra-board-config

## Description
Remove explicit fast_mode activation from the versioned agents-infra Codex ceiling while preserving the supported false-by-default contract.

## Scope
task-board.config.json only plus board evidence; preserve exclusive Codex gpt-5.6-sol/high and workload recommendations.

## Acceptance Criteria
fast_mode is absent from the agents-infra Codex ceiling; spawn preflight resolves fast_mode=false from default with service tier default; exclusive Codex and exact gpt-5.6-sol/high plus workload recommendations remain unchanged; config validates and focused evidence is attached.
