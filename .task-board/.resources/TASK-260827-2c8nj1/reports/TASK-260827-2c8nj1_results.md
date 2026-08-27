# Canonical pull-request delivery

Added the global default branch-to-PR-or-MR-to-review-to-green-checks-to-merge publication workflow. The rule requires a real platform review event, explains GitHub self-approval limits, preserves stricter repository-local rules, and does not grant publication authority or replace task-board commit confirmation.

Validation: built the source agents-infra CLI into an isolated temporary home, then setup global and verify global both exited 0. The policy was present in the installed workflow module and rendered Codex AGENTS.md.