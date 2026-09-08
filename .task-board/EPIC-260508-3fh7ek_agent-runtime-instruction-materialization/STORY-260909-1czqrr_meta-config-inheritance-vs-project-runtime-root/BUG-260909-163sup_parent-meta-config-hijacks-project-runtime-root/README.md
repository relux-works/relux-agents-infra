# BUG-260909-163sup: parent-meta-config-hijacks-project-runtime-root

## Description
installedProjectRuntimeRoot() walks up from the project directory and returns the first ancestor that contains a .agents directory, then materializes the whole provider surface there instead of in the project. A parent directory holding only .agents/.configs is a deliberate meta-config shared by every project beneath it, not a runtime root, so the walk lands on it and PreparePrimarySession fails.

Observed from /Users/alexis/src/local-models (no local .agents of its own), with a meta-config at /Users/alexis/src/.agents/.configs/project-config.toml:

  agents-infra primary-session preparation failed: prepare primary-session project surface: prepare Claude project surface: open /Users/alexis/src/.agents/skills: no such file or directory

Same failure through the anthropic-infra launcher, and task-board sessiond fails transitively because it calls the same PreparePrimarySession. doctor local on the project confirms the config is correctly inherited from the parent (claude_primary_model_source: /Users/alexis/src/.agents/.configs/project-config.toml) while the surface is wrongly targeted at the parent too.

Two distinct defects:
1. Root resolution conflates config inheritance with surface materialization. Inheriting configuration from an ancestor .agents/.configs is correct and intended; relocating the runtime root to that ancestor is not.
2. setupClaude() os.ReadDir(<agentsDir>/skills) hard-fails on a missing skills directory even though it MkdirAll-s the destination claude skills directory just above.

Binary: agents-infra v1.6.1-128-gab60e0d. A partial working-tree change in tools/agents-infra/internal/infra/primary_session_prepare.go gates the walk on verifyRuntimeReceipt; it is uncommitted, unbuilt, and only suppresses the symptom rather than restoring inheritance.

## Scope
tools/agents-infra/internal/infra: installedProjectRuntimeRoot and the project-config inheritance path; setupClaude skills-directory handling; regression tests.

## Acceptance Criteria
A project with no .agents of its own, nested under an ancestor that carries .agents/.configs, materializes its provider surface in the project directory and still inherits the ancestor project-config; primary-session preparation and the anthropic-infra/openai-infra launchers start; setupClaude does not fail on a missing skills directory; negative tests cover the parent-meta-config nesting case and the missing-skills case.
