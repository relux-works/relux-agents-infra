# TASK-260830-3nx7yp: support-opencode-as-agent-environment

## Description
Add OpenCode as a supported agent environment in agents-infra composition, instruction materialization and local-model targeting.

## Scope
OpenCode as an agent environment across agents-infra composition, instruction materialization and local-model targeting.

## Acceptance Criteria
OpenCode is selectable as an agent environment through the same configuration surface as the existing environments, with no OpenCode-specific branch in shared composition code. Instruction materialization produces OpenCode's expected entrypoint and include chain, verified on the installed artifact rather than on source bytes. Local-model targeting composes with OpenCode across all three axes independently. Adding it weakens no admission clause of the comparative gate.
