# STORY-260830-37bq03: adopt-agents-management-as-single-agent-plane

## Description
Make agents-infra a consumer of skill-agents-management, so task-board and agents-infra share one plugin plane for agentic systems, vendors and local inference engines instead of each carrying its own.

## Scope
agents-infra composition, primary-session launch, targets and model-harness profiles. skill-agents-management is the source repo for the plugin contracts; fix contract gaps there rather than working around them here.

## Acceptance Criteria
agents-infra resolves agentic systems and model vendors through skill-agents-management rather than its own hardcoded codex/claude/pi handling, task-board keeps consuming the same module, and a new environment or engine is added in one repository only.
