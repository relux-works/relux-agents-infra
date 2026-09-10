# STORY-260830-37bq03: adopt-agents-management-as-single-agent-plane

## Description
Make agents-infra a consumer of skill-agents-management, so task-board and agents-infra share one plugin plane for agentic systems, vendors and local inference engines instead of each carrying its own.

## Scope
agents-infra composition, primary-session launch, targets and model-harness profiles. skill-agents-management is the source repo for the plugin contracts; fix contract gaps there rather than working around them here.

## Acceptance Criteria
Configured target environment admission and primary-session dispatch derive the admitted agentic-system set from the skill-agents-management registry (no hardcoded codex/claude-code/pi lists in project_config.go, canonical_target.go, primary_session_launch_plan.go); refusals for unregistered or unlaunchable identifiers stay fail-closed with unchanged CLI wording; the lockstep release/rollback plan names the currently pinned agents-management version; negative tests cover an unregistered identifier, a registered-but-unlaunchable system, and the downgrade direction.
