# TASK-260916-38vqh4: inventory-and-deprecation-plan

## Description
Research report: inventory every command, package and setup step of tools/agents-infra and scripts/setup.* and classify each as REMOVE (instruction sync, @ rendering, skills fan-out, MCP registry, MCP composition inside the claude|codex launchers), DEPRECATE (the agents-infra claude|codex launchers and the openai-infra|anthropic-infra entrypoints if they are the same surface) or KEEP (claude-settings.json linking, codex config.toml merge, .rules, pi local-model runtime with broker/profiles/targets/harness, lldb-mcp wrapper, attachments manifest contract, compose/prepare child-launch contracts consumed by task-board, doctor/verify residual). For each REMOVE item cite the Curator replacement (profile install / curator run / managed homes) with the B5 evidence. Produce the exact edit plan (files, functions, tests, README sections) for the implementation task.

## Scope
(define task scope)

## Acceptance Criteria
TASK-<id>_report.md: classification table with file:line, replacement evidence, deprecation message text and exit code, README outline, test plan, risks.
