# TASK-260830-1xscas: replay-accepted-policy-rework-on-fast-off-trunk

## Description
Apply the validated revision-2 four-path replay input only after proving a freshly fetched protected-trunk base, reconcile current drift, and publish an immutable CR.

## Scope
.instructions/INSTRUCTIONS_WORKFLOW.md, README.md, LOGBOOK.md, tools/agents-infra/internal/infra/infra_test.go only. Never restore task-board.config.json fast_mode.

## Acceptance Criteria
Selected base and workspace equal freshly fetched origin/main; patch input digest is verified; exactly four policy paths change; task-board.config.json stays fast_mode-absent with false/default preflight; additive broadened-trigger, Claude entrypoint, Claude include, and Codex include mutants all fail; focused/full tests, vet, build, canonical setup, global verify, installed parity, independent CR review, PR and merge pass.
