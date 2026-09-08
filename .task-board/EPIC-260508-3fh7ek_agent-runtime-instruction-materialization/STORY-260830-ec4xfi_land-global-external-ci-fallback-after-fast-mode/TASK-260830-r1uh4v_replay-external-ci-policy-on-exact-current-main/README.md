# TASK-260830-r1uh4v: replay-external-ci-policy-on-exact-current-main

## Description
Apply the independently validated four-path policy delta to the fresh managed Story workspace from exact protected origin/main, reconcile only proven current-trunk drift, and publish an immutable Change Request.

## Scope
.instructions/INSTRUCTIONS_WORKFLOW.md, README.md, LOGBOOK.md, and tools/agents-infra/internal/infra/infra_test.go only.

## Acceptance Criteria
The selected base equals freshly fetched origin/main and workspace HEAD; the prior patch is replay input rather than authority; broadened-trigger and Claude/Codex include-bypass mutants fail; focused/full tests, vet, build, canonical setup, verify global, and installed parity pass; a task-scoped outcome and exact CR are published for independent review.
