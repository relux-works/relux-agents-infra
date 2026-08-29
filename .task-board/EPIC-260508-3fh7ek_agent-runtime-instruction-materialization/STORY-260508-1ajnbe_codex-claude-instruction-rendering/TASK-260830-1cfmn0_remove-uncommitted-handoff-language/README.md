# TASK-260830-1cfmn0: remove-uncommitted-handoff-language

## Description
Remove residual no-commit and uncommitted-handoff wording, preserve automatic signed delivery, install the revised global instructions, and track historical missing resource payloads in GitHub.

## Scope
.instructions/INSTRUCTIONS_WORKFLOW.md; .instructions/INSTRUCTIONS_REMOTE_AGENTS.md; installed global instruction surfaces; GitHub integrity-debt issue; signed PR and exact-head landing.

## Acceptance Criteria
Global instructions contain no generic no-commit or leave-uncommitted directive; owning agents automatically complete signed PR delivery; remote workers return reviewable artifacts without a contradictory commit prohibition; installed ~/.agents and rendered Codex instructions match; a GitHub issue tracks the current missing-resource payload debt with reproduction and acceptance criteria; validation and signed landing pass.
