# TASK-260830-1t2xef producer results

## Repository delta

- Changed only `task-board.config.json`.
- Migrated `spawn.ceilings` to `spawn-policy-v4`.
- Kept `preferred_agentic_system` exclusive to Codex.
- Configured one admitted pair: `gpt-5.6-sol/high`.
- Set Codex `fast_mode` to `true` with configured provenance.
- Removed the Claude ceiling.
- Set all eleven workload classes, including `unified`, to the same sole Codex pair.
- Producer made no commit and performed no integration.

## Fresh-base proof

- `git fetch --no-tags origin main`: exit 0.
- Fresh `FETCH_HEAD`: `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- `origin/main`: `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- Story `selected_base_oid`: `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- Story `upstream_oid`: `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- Worktree `HEAD`, branch tip, and merge-base: `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- Ahead/behind counts were `0/0`; worktree was clean before editing.

## Static and production-policy validation

All accepted gates below ran after the change.

| Command / gate | Exit | Result |
| --- | ---: | --- |
| Exact JSON policy assertion with `jq -e` | 0 | Exact v4 ceiling, sole Codex pair, no Claude key, fast mode, and eleven identical workload recommendations |
| `TASK_BOARD_CONFIG=$PWD/task-board.config.json task-board q --format compact 'project_config(view=spawn-preflight, role=developer, agent=codex, workload_class=implementation)'` | 0 | `explicit_allow_set`, exact `gpt-5.6-sol/high`, `fast_mode=true` from `spawn.ceilings.codex.fast_mode`, sole recommendation |
| `.temp/TASK-260830-1t2xef/validation/assert-preflight.zsh` | 0 | Production preflight assertions passed for all eleven explicit workload classes |
| `schema(operation=project_config)` plus `jq -e` assertion | 0 | `spawn-preflight` view and exact eleven-value `workload_class` enum projected |
| Full `project_config()` projection plus `jq -e` assertion | 0 | Candidate config path, exclusive Codex, only configured Codex ceiling, exact pair, fast-mode provenance, and all workload classes |
| `jq -e . task-board.config.json` | 0 | JSON parses |

Production call site exercised by the task-scoped tests: installed `task-board` query operation `project_config(view=spawn-preflight, role=developer, agent=codex, workload_class=...)`.

The first diagnostic preflight without `TASK_BOARD_CONFIG` exited 0 but correctly showed the control-root v2 config; it was rejected as candidate evidence. The documented `TASK_BOARD_CONFIG=$PWD/task-board.config.json` override was then used for every accepted candidate projection while retaining the authoritative external board.

## Negative gate evidence

Both mutants were copies under `.temp`; the repository candidate was not mutated.

| Expected-red mutant | Exit | Expected-failure rationale |
| --- | ---: | --- |
| Add admitted `gpt-5.6-terra/high`, then run the all-class production preflight assertion | 1 | The exact sole-pair gate rejected a widened admission set |
| Add a valid Claude v4 ceiling, then run the full production projection assertion | 1 | The exact configured-provider gate rejected a Claude ceiling even though preferred runtime remained exclusive Codex |

These are narrowed-gate mutants: they preserve the gate and add one forbidden member, rather than deleting policy wholesale.

## Repository validation

| Command | Exit | Result |
| --- | ---: | --- |
| `cd tools/agents-infra && go test ./... -count=1` | 0 | All packages passed; main package `229.663s`, infra package `369.367s` |
| `cd tools/agents-infra && go vet ./...` | 0 | Clean |
| `git diff --check` | 0 | Clean |
| `git status --short` / `git diff --name-only` | 0 | Only `task-board.config.json` changed |

No command launched or contacted a live model runtime. All task-board policy checks were read-only query operations.

## Readiness note

`task-board version` is not a supported subcommand and returned exit 1 during the initial readiness probe. `task-board --help` returned exit 0 and confirmed the required `q`, `m`, `handoff`, `spawn`, `validate`, and `worktree` surfaces. The mandatory initial status mutation had already returned exit 0. Detailed readiness evidence is retained in `.temp/TASK-260830-1t2xef/validation/tool-readiness-01.log`.
