# TASK-260721-26pa1j: scale-workflow-to-task-risk

## Description
Add a global workflow instruction that requires process overhead to remain proportional to task scope and risk. Prevent ordinary localized or mechanical changes from automatically triggering full board decomposition, tracked producer/reviewer spawns, exhaustive validation, or repeated ceremony when a minimal inline workflow and narrow checks are sufficient.

## Scope
Edit only the global workflow source module and its generated instruction artifact as required by the agents-infra rendering contract. Preserve all unrelated dirty work. Sync the installed global runtime through the supported agents-infra setup flow. Commit only this task scope with author and committer timestamps set to 2026-07-20 20:30 MSK, then push main as explicitly authorized.

## Acceptance Criteria
Global instructions explicitly state that the presence of a task tracker does not by itself trigger project-management skill or spawned-agent orchestration; low-risk localized and reversible work uses the smallest valid tracked workflow, inline execution, proportional self-review, and narrow checks unless the user, repository, or applicable skill explicitly requires more; prior explicit quality gates are not carried forward after scope narrows without current justification; generated/runtime instructions contain the new rule; unrelated dirty files remain unstaged; the focused commit has both Git dates at 2026-07-20 20:30 +0300 and is pushed to origin/main.
