# STORY-260901-1br672: provider-local-primary-session-composition

## Description
Make provider launch composition validate only the configuration surface selected by that provider so optional local runtime profiles cannot block unrelated Codex or Claude sessions.

## Scope
Provider-scoped project configuration loading for primary-session composition, strict selected-provider validation, regression fixtures, bsim-shaped smoke evidence, and documentation.

## Acceptance Criteria
Codex-only and Claude-only primary-session plans compose successfully in projects containing absent, incomplete, or unavailable unselected Pi profiles; selecting those Pi/Qwen profiles remains strict and actionable; production-callsite tests and a bsim-shaped installed-binary smoke prove both boundaries.
