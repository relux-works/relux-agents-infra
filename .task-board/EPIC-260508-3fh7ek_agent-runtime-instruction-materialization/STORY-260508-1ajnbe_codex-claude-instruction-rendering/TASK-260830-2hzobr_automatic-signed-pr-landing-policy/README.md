# TASK-260830-2hzobr: automatic-signed-pr-landing-policy

## Description
Replace the global no-commit policy with automatic signed commit and exact PR landing rules, allow explicit agent co-authorship, regenerate installed instructions, and publish the reviewed change.

## Scope
.instructions/INSTRUCTIONS_WORKFLOW.md; installed global instruction materialization under ~/.agents and ~/.codex; validation; signed PR and exact-head main landing.

## Acceptance Criteria
Global instructions require automatic commits signed by the configured author; each task starts from synchronized local main; work reaches main through reviewed PR and automatic exact fast-forward landing; agent co-authorship is allowed when explicitly identified; generated and installed instructions match source; validation passes except documented pre-existing board defects.
