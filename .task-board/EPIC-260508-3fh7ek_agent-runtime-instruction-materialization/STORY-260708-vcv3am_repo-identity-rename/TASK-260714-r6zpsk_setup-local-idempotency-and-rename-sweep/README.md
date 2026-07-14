# TASK-260714-r6zpsk: setup-local-idempotency-and-rename-sweep

## Description
Make agents-infra setup local idempotent for repeated project-local refreshes without destructive overwrite of local policy/configuration, clean up remaining alexis-agents-infra naming references after the repo rename, and handle the stale old local directory path safely.

## Scope
(define task scope)

## Acceptance Criteria
Repeated agents-infra setup local runs succeed when project-local .agents already contains managed repo skill links and should not fail while removing or recreating an already-correct link. Existing local primary-session policy, MCP opt-in, custom project files, task-board config, and non-generated artifacts are preserved unless an explicit setup flag asks to change them. Remaining alexis-agents-infra mentions in relux-agents-infra source/docs/tests are renamed to relux-agents-infra, except historical data under ignored .temp if intentionally skipped. Verification covers Go tests for agents-infra and a local setup smoke on a temporary target run twice.
