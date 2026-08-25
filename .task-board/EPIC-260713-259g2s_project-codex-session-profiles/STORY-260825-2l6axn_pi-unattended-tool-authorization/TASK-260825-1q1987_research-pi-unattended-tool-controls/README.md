# TASK-260825-1q1987: research-pi-unattended-tool-controls

## Description
Research pinned Pi 0.84.2 and current upstream mechanisms for unattended tool execution, including native flags/config, tool approval flow, extension APIs, custom tool replacement, RPC mode, and minimal patch points.

## Scope
Use primary sources only: upstream repository code, release artifacts, official docs, and issue/discussion records from project maintainers. Reproduce relevant behavior locally without launching the MLX model. Persist findings in .research/260825_pi-unattended-tool-authorization.md and as a task outcome resource.

## Acceptance Criteria
The research names the exact authorization call path, distinguishes project trust from tool execution permission, tests candidate mechanisms against the pinned binary/source, compares security and maintenance costs, and recommends one concrete agents-infra implementation with explicit defaults, opt-in scope, diagnostics, and negative tests.
