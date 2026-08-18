# STORY-260817-1on8ex: pi-project-local-primary-session

## Description
Add Pi as a project-aware primary agent launcher alongside Codex and Claude.

## Scope
Nearest-project Pi policy, non-launching diagnostics, local-model runtime boundary, direct launcher, pi-infra alias, setup verification, tests, and documentation.

## Acceptance Criteria
A project can select a local-model runtime profile through .agents/.configs/project-config.toml; agents-infra pi resolves the nearest effective policy and starts Pi with the selected model without mutating global Pi credentials or settings; --print-config is non-launching; pi-infra preserves cwd and arguments; setup and verify install and validate the alias; tests cover precedence, invalid config, argument overrides, missing dependencies, and alias delegation.
