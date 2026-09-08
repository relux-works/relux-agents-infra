# TASK-260830-rddvxd: replay-and-verify-global-external-ci-fallback

## Description
Apply the independently audited policy candidate to the fresh Story workspace, repair any trunk drift, and validate both generated agent instruction surfaces.

## Scope
Only .instructions/INSTRUCTIONS_WORKFLOW.md, README.md, LOGBOOK.md, and focused setup/infra tests unless current-trunk reconciliation proves another source file is required.

## Acceptance Criteria
All policy clauses are executable and narrowly triggered; broadened-trigger and include-bypass mutants fail; full infra tests, vet, build, canonical setup with documented LLDB skip, verify global, and installed parity pass; no remote status is forged; exact CR is independently reviewed.
