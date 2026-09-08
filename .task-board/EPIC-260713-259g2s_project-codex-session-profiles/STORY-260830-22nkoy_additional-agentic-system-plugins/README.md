# STORY-260830-22nkoy: additional-agent-environments

## Description
Add OpenCode and Hermes as supported agent environments alongside the existing Claude Code, Codex, Qwen, Gemini, Muse and Agy runtimes, so the environment axis is as extensible as the engine axis.

## Scope
OpenCode and Hermes are added as agentic-system plugins in skill-agents-management's pkg/agentic, not as bespoke branches in agents-infra. agents-infra currently knows only codex, claude and pi directly; the point of this work is that it stops knowing them individually.

## Acceptance Criteria
OpenCode and Hermes are registered agentic-system plugins in skill-agents-management, available to both task-board spawn and agents-infra composition without either repository special-casing them.
