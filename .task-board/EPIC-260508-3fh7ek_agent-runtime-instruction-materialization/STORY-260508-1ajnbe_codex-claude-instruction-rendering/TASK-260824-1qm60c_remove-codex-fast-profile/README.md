# TASK-260824-1qm60c: remove-codex-fast-profile

## Description
Remove the retained Codex [profiles.fast] definition from the relux-agents-infra source configuration after the owner explicitly withdrew the earlier keep-fast-profile decision.

## Scope
Update source-managed Codex configuration and directly related documentation that advertises the fast profile. Preserve service_tier=default, primary-session policy, user-managed trust/TUI state, and unrelated repository work. After accepted source integration, synchronize supported global and source-repository local runtimes through agents-infra setup and verify both.

## Acceptance Criteria
Source .configs/codex-config.toml contains no profiles.fast table or persistent fast service-tier selection; documentation no longer advertises agents-infra codex --profile fast; installed global and source-repository local managed config copies contain no fast profile after supported setup while user-managed config state is preserved; agents-infra doctor and verify pass for global and local modes; repository-focused tests or config validation pass.
