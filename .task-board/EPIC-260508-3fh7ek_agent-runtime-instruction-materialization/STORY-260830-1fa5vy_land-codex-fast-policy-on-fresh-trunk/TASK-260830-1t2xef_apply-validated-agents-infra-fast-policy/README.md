# TASK-260830-1t2xef: apply-validated-agents-infra-fast-policy

## Description
Replay the validated task-board.config.json delta on a freshly fetched current trunk using the installed fast_mode-capable task-board.

## Scope
Only task-board.config.json. Use the prior isolated current-trunk candidate as non-authorizing input; independently validate exact provider, pair, fast_mode, workload recommendations, and schema projection.

## Acceptance Criteria
Candidate base equals freshly fetched origin/main; contract_version is spawn-policy-v4; preferred_agentic_system is exclusive codex; sole admitted pair and every workload recommendation are gpt-5.6-sol/high; fast_mode is true; Claude ceiling is absent; project preflight and repository checks pass; independent reviewer accepts exact CR.
