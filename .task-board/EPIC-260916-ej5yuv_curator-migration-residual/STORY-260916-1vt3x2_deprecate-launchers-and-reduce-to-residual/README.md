# STORY-260916-1vt3x2: deprecate-launchers-and-reduce-to-residual

## Description
Deprecate agents-infra claude|codex (printed deprecation pointing at curator run <env>), remove the migrated responsibilities (instruction sync and @ rendering, skills fan-out, MCP registry and MCP composition of the deprecated launchers), keep the residual (claude-settings.json linking, codex config.toml merge, .rules, pi local-model runtime, lldb-mcp wrapper, attachments manifest contract, task-board compose/prepare contracts), rewrite README; version bump and CHANGELOG.

## Scope
(define story scope)

## Acceptance Criteria
agents-infra claude|codex print a deprecation (exit code documented) and no longer compose MCP or fan out skills; setup/doctor/verify no longer sync instructions or link skills/MCP registry; residual commands unchanged and tested; README states what remains and why; go build/vet/test green; landed via PR with independent review.
