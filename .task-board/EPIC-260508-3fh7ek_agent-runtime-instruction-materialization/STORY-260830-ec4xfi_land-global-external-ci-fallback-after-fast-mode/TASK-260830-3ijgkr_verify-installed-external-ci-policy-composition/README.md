# TASK-260830-3ijgkr: verify-installed-external-ci-policy-composition

## Description
On the accepted current-trunk Story candidate, independently rerun source-to-generated-to-installed instruction composition and the exact negative trigger matrix before final Story integration.

## Scope
Validation and evidence only unless a concrete parity defect is proven; do not expand policy wording or touch unrelated configuration.

## Acceptance Criteria
Fresh candidate contains the accepted policy and landed fast-mode configuration; source, generated Claude/Codex, ~/.agents, ~/.claude, and ~/.codex surfaces match after canonical setup; broadened trigger and both include-bypass mutants fail; every real validation command and exit is attached; empty or corrective CR is independently accepted before Story integration.
