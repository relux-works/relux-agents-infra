# STORY-260713-3et7b3: provider-primary-session-policy

## Description
Add independent Codex and Claude primary-session model policies so a project can be opened through either agent system without inheriting the other provider model.

## Scope
Primary-session model policy resolution for Codex and Claude in agents-infra composition and project configuration.

## Acceptance Criteria
A project opened through Codex resolves only the Codex primary-session model policy, and a project opened through Claude resolves only the Claude one; neither inherits the other provider's model. A project configuring one agent system and not the other opens through the configured one and refuses the unconfigured one with an error naming the missing policy rather than falling back to a default. Production entrypoint evidence, not unit fixtures, demonstrates both directions.
