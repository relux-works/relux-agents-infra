# TASK-260824-1jjze0: rewrite-all-local-project-target-configs

## Description
Implement the explicit rollout requirement "rewrite all local project target configs" using contract R2-R3, R5-R6 and Sections 2, 5-7: build and run a task-scoped one-time recursive rewrite over /Users/alexis/src/**/.agents/.configs/project-config.toml, preserving the full MCP configuration exactly and replacing the remaining agent configuration with the three canonical targets and vendor entrypoint mappings.

## Scope
The script lives only in task scratch/output, not in the agents-infra runtime migration path. Discover hidden and ignored configs while excluding .git, .temp, and dependency caches; inventory and dry-run before writes; preserve unrelated user changes and exact MCP TOML semantics; rewrite recursively; validate every changed config with production parsing and alias compose; record hashes, skips, failures, and rollback evidence.

## Acceptance Criteria
Traceability: explicit rollout requirement plus contract R2-R3, R5-R6 and Sections 2, 5-7. A reproducible one-time recursive script inventories every in-scope config, dry-runs cleanly, preserves each MCP section exactly, replaces all other agent configuration with the three exact canonical targets and mappings, and validates every resulting file through production parsing and alias compose. No automatic runtime migration code exists; unrelated user changes are untouched; rollback and per-file results are recorded.
