# STORY-260830-112ewt: surface-degraded-child-launch-composition

## Description
Make a failed child-launch composition visible where it matters instead of degrading every spawned child to a bare launch behind a one-line diagnostic, and give operators a way to tell that a newly required profile field has stranded their installed configuration.

## Scope
agents-infra child-launch composition and its spawn-facing diagnostics. Fail-closed config semantics and the absence of automatic migrations are deliberate and stay; this is about visibility and diagnosis, not about relaxing validation.

## Acceptance Criteria
A composition failure is reported so an orchestrator cannot spawn a series of children unaware that each launched without its project MCP configuration, and a configuration stranded by newly required profile fields is diagnosable in one command that names every missing field at once rather than one field per invocation.
