# TASK-260830-11ajl2: consume-agents-management-in-agents-infra-composition

## Description
Narrowed on 2026-08-31 after the first-consumer gap report. agents-infra takes the skill-agents-management dependency and consumes it for identity only: SystemID normalization and registry lookup as a cross-check. Composition production, target and alias resolution, and primary-session launch variants stay in agents-infra, matching the audit's own stays-here verdicts. The hardcoded codex, claude and pi branches REMAIN, because v0.5.0 cannot express what they do and the accepted lockstep plan requires both sides to compile exact v0.5.0 with no new release before a fresh human acceptance. The genuine contract gaps are tracked separately.

## Scope
(define task scope)

## Acceptance Criteria
agents-infra depends on and imports skill-agents-management v0.5.0 and uses it for SystemID normalization and registry cross-check. The existing test suite passes unchanged at 482 test functions across four packages, with no test edited to make the change land. The Go floor bump from 1.21.0 to 1.25.5 that the dependency forces is stated and its effect on any build or CI assumption is checked. Composition, primary-session launch, targets and model-check behave identically. Nothing in this task requires a new agents-management release, and no hardcoded branch is removed that v0.5.0 cannot replace.
