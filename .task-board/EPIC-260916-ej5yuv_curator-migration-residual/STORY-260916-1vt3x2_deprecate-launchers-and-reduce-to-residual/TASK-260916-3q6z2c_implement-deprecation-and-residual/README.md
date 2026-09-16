# TASK-260916-3q6z2c: implement-deprecation-and-residual

## Description
Implement the accepted plan from the inventory task: deprecation of agents-infra claude|codex (message points at curator run claude_code|codex_cli; behaviour documented), removal of instruction sync, @ rendering, skills fan-out, MCP registry and MCP composition from the launchers and from setup/refresh-links/doctor/verify, residual kept intact with tests, README rewrite, CHANGELOG and version bump.

## Scope
(define task scope)

## Acceptance Criteria
go build/vet/test green in tools/agents-infra; deprecation covered by tests; removed paths have no dead code left; README matches the shipped surface; compose/prepare contracts unchanged (golden tests still pass).
