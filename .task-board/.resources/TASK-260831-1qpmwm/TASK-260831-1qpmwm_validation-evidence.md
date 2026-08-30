# TASK-260831-1qpmwm — validation evidence

## Story base preflight
- `git fetch origin main` → `origin/main` OID `5feebbb170ea9a9ef884899c846a897d58f02fc5`
- Worktree `HEAD` OID equals `origin/main` OID exactly (zero drift, fast-forward).
- Selected base: `5feebbb170ea9a9ef884899c846a897d58f02fc5`.

## Scope
- Only `task-board.config.json` modified. `git status --short` shows a single `M task-board.config.json`.
- Diff touches only `spawn.ceilings`, `spawn.preferred_agentic_system`, and adds `spawn.workload_classes`.
- `spawn.enabled`, `spawn.max_parallel`, `spawn.worktree_isolation`, `version_control` are byte-identical to before.

## JSON validity
```
$ python3 -m json.tool task-board.config.json > /dev/null && echo json_valid=true
json_valid=true
```

## Policy bytes equal attached authority
Canonicalized (`jq -S`) comparison of `{ceilings, preferred_agentic_system, workload_classes}`
extracted from `task-board.config.json` against the attached `effective-mixed-policy.json`
authority produced an empty diff — exact match (key set, nesting, and values).

## fast_mode absence
`grep -i fast_mode task-board.config.json` → no matches. Codex preflight below independently
confirms `fast_mode: false` with `fast_mode_source: "default"` — i.e. resolved from the
runtime default because the key is absent from policy, not from an explicit setting.

## Preflights (`task-board q 'project_config(view=spawn-preflight, role=ROLE, agent=PROVIDER)'`)

### Codex developer
- `providers.allowed` = `["claude","codex"]`, `providers.target` = `"codex"`
- `resolved_role_ceiling.allowed_models` = `["gpt-5.6-sol"]`
- `resolved_role_ceiling.reasoning_effort` = `"medium"`, `criterion` = `"less_or_equal"`
- `resolved_role_ceiling.fast_mode` = `false`, `fast_mode_source` = `"default"`
- `resolved_role_ceiling.contract_version` = `"spawn-policy-v4"`
- Full output: `.temp/preflight-codex-developer.json`

### Claude developer
- `providers.allowed` = `["claude","codex"]`, `providers.target` = `"claude"`
- `resolved_role_ceiling.allowed_models` = `["claude-sonnet-5"]`
- `resolved_role_ceiling.reasoning_effort` = `"high"`, `criterion` = `"less_or_equal"`
- Full output: `.temp/preflight-claude-developer.json`

### Claude reviewer
- `providers.allowed` = `["claude","codex"]`, `providers.target` = `"claude"`
- `resolved_role_ceiling.allowed_models` = `["claude-sonnet-5"]`
- `resolved_role_ceiling.reasoning_effort` = `"high"`, `criterion` = `"less_or_equal"`
- Full output: `.temp/preflight-claude-reviewer.json`

All three preflights resolve `preferred_agentic_system.mixed = [claude, codex]` as
`providers.allowed`, confirming the mixed provider set applies uniformly for both
provider/role combinations exercised.

## Negative-path evidence (gating behavior at the production call site)

The spawn ceilings are a gate: `task-board spawn`, the real launcher, must
refuse a request that falls outside the new bounds. Two out-of-bound requests
were issued directly against `task-board spawn TASK-260831-1qpmwm --role developer
--background ...` (the production call site) after the new config was written:

1. `--agent codex --model gpt-5.6-sol --reasoning-effort high` (old ceiling
   admitted `high`; new ceiling caps at `medium`, `less_or_equal`):
   ```
   workload_class_pair_unavailable_in_snapshot: selected pair codex/gpt-5.6-sol/high was not available in the verified snapshot
   EXIT=1
   ```
2. `--agent claude --model claude-opus-5 --reasoning-effort high` (old ceiling
   admitted `claude-opus-5`; new ceiling's `allowed_models` is
   `["claude-sonnet-5"]` only):
   ```
   workload_class_pair_unavailable_in_snapshot: selected pair claude/claude-opus-5/high was not available in the verified snapshot
   EXIT=1
   ```

Both requests were refused before any task lookup or launch side effect —
confirmed by `task-board q 'get(TASK-260831-1qpmwm) { status assignee }'`
still reporting `status=development`, `assignee=[implementer] developer (claude)`
unchanged, with no new RUN created. This proves the applied policy actually
narrows the admitted (model, effort) pair set at the real spawn entry point,
not just in the static JSON.

## Diff check
```
$ git diff --stat task-board.config.json
 task-board.config.json | 175 +++++++++++++++++++++++++++++++++++++++++++------
 1 file changed, 154 insertions(+), 21 deletions(-)
```

## Note on task scope (`task-board.config.json` only)
No Go source under `tools/agents-infra` references `preferred_agentic_system`,
`workload_classes`, or `reasoning_effort_criterion` — the spawn config is consumed
entirely by the external `task-board` CLI, not by in-repo code. The existing
`spawn.worktree_isolation.validation.commands` (`go test ./...`, `go vet ./...`)
are therefore orthogonal to this config-only change and were not re-run for this
task, since no Go source changed.
