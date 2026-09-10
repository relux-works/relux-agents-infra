# TASK-260911-39bag5: derive-environment-admission-and-dispatch-from-the-agentic-registry

## Description
Remove the last hardcoded agentic-system lists in agents-infra so that admission and dispatch derive from the skill-agents-management registry (v0.5.9, pkg/agentic). Known sites: tools/agents-infra/internal/infra/project_config.go:411 (validateProjectTarget environment must be one of codex, claude-code, pi), canonical_target.go:80-98 and :417-421 (environment/system id switches), primary_session_launch_plan.go:208-254 (per-provider dispatch). The registry is WIDER than what agents-infra can launch (agy, gemini, muse, qwen are registered too), so the launchable set must come from the systems this program actually has a launcher for, expressed once and cross-checked against registry membership, never from registry membership alone. Superseded prior attempt: TASK-260830-11ajl2 (8c7b771) added an identity-only guard that contradicts trunk, do not revive it.

## Scope
tools/agents-infra/internal/infra/{project_config,canonical_target,primary_session_launch_plan,agents_management_registry}.go and their tests; README/SKILL only where the environment wording is documented. No new runtime, no change to which systems can be launched, no go.mod change.

## Acceptance Criteria
A single launchable-systems declaration replaces every literal codex/claude-code/pi list on the admission and dispatch paths; each admitted identifier is cross-checked against agentic.Default at validation time; an unregistered identifier, a registered-but-unlaunchable system (e.g. gemini), and a case/alias variant are each refused fail-closed at the production entry with the existing CLI wording preserved (golden test on the error text); existing project-config fixtures and the installed-binary setup tests pass unchanged; narrowing mutants that drop the registry cross-check or re-add a literal fail a named test.
