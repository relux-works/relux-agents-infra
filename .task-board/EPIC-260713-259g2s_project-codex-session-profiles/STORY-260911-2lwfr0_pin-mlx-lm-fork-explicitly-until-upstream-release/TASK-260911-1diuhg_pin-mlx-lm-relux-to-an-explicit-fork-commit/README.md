# TASK-260911-1diuhg: pin-mlx-lm-relux-to-an-explicit-fork-commit

## Description
Replace the editable pipx install of mlx-lm-relux with a non-editable install from an explicit relux-works/mlx-lm commit or tag (currently the intended tree is 45a472f on task/TASK-260830-2hc5r2-bounded-kv, which contains 9150698). Make the agents-infra local-model setup/verify path check that the installed distribution matches that pin and refuse editable/drifted installs. Document the retirement condition.

## Scope
pipx spec + agents-infra setup/verify for the qwen local-model target + README note

## Acceptance Criteria
See story AC; verify local passes on the pinned install and fails on an editable install (negative control).
