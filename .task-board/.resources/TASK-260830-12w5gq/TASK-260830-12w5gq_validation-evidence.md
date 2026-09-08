# TASK-260830-12w5gq — restore-mixed-sol-medium-and-sonnet-five-settings — validation evidence

## Scope

`task-board.config.json` only (`spawn.ceilings`, `spawn.preferred_agentic_system`, `spawn.workload_classes`).

## State found

The working tree already carried the target edit (uncommitted, pre-existing dirty state at
session start). No further code edits were needed; this run validated the existing diff against
the active project policy and against the board CLI's own spawn-preflight resolution.

## Byte-for-byte alignment check

Compared `spawn.ceilings`, `spawn.preferred_agentic_system`, and `spawn.workload_classes` in
`task-board.config.json` against `/Users/alexis/src/relux-works/skill-project-management/task-board.config.json`
(the active project policy) via Python `json.load` equality:

```
ceilings True
preferred_agentic_system True
workload_classes True
```

## JSON validity

```
python3 -c "import json; json.load(open('task-board.config.json')); print('VALID JSON')"
-> VALID JSON
```

## Codex spawn-preflight (developer role)

`task-board q 'project_config(view=spawn-preflight, role=developer, agent=codex)'`

- `admitted_pairs`: `gpt-5.6-sol` with `efforts: [low, medium]`
- `resolution_model`: `gpt-5.6-sol`, `reasoning_effort`: `medium`, `reasoning_effort_criterion`: `less_or_equal`
- `fast_mode`: `false`, `fast_mode_source`: `default` (key absent from config, resolves false)
- `providers.allowed`: `[claude, codex]`, `providers.target`: `codex`

Confirms AC: "Codex admits only gpt-5.6-sol up to medium" and "fast_mode is absent and resolves false".

## Claude spawn-preflight (developer role)

`task-board q 'project_config(view=spawn-preflight, role=developer, agent=claude)'`

- `admitted_pairs`: `claude-sonnet-5` with `efforts: [low, medium, high]`
- `resolution_model`: `claude-sonnet-5`, `reasoning_effort`: `high`, `reasoning_effort_criterion`: `less_or_equal`
- `providers.allowed`: `[claude, codex]`, `providers.target`: `claude`

Confirms AC: "Claude admits only claude-sonnet-5 up to high" and mixed provider admission
(`preferred_agentic_system.mixed: [claude, codex]`).

## Workload classes

All 11 `spawn.workload_classes` entries (`unified`, `architecture`, `implementation`,
`mechanical`, `debugging`, `testing`, `review`, `research`, `documentation`, `migration`,
`operations`) contain exactly the two admitted pairs (`codex/gpt-5.6-sol/medium`,
`claude/claude-sonnet-5/high`), matching the active project policy verbatim, with `unified` as
the first/fallback entry in every class ordering variant used.

## Board structure validation

`task-board validate` reports 83 pre-existing `MISSING_ACTIVITY` findings across unrelated older
board elements (activity-stream gaps predating this task). None reference `task-board.config.json`
or this task's scope; left untouched per the "preserve unrelated dirty checkout" instruction.
