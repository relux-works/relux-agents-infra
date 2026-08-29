# TASK-260824-1jjze0 — Blocked rollout evidence

## Outcome

The task-scoped rewrite implementation, inventory, dry-run, production compose
validation, negative tests, and rollback machinery are ready. Recursive apply
was refused before its first source write because the required canonical Qwen
identity is not representable by the installed `mlx_lm.server` deployment.

No automatic runtime migration code was added. No project config under
`/Users/alexis/src` was rewritten.

## Inventory and dry-run

- Root: `/Users/alexis/src`
- Exact suffix: `.agents/.configs/project-config.toml`
- Exclusions: `.git`, `.temp`, dependency/build/cache directories recorded in
  `run/inventory.json`
- Inventory: 121 regular configs across 116 Git worktrees
- Hidden and ignored paths: included by the `os.walk` inventory
- Candidate rewrites: 121
- Production validations: 363 (`121 × openai-infra/anthropic-infra/qwen-infra`)
- Validation result: 363 exit 0, zero failures
- MCP proof: raw MCP blocks and parsed MCP subtrees compared before/after for
  every candidate
- Non-agent config blocks: retained byte-for-byte
- Legacy `[agents...]`: replaced only in candidates by the canonical three
  targets, managed Pi profile, and exact entrypoint map

Authoritative reports:

- `run/inventory.json`
- `run/inventory.txt`
- `run/dry-run-report.json`
- `dry-run-command-02.log`

## Real MLX gate and refusal

Production apply reran the full dry-run, then started the real executable:

```text
/Users/alexis/.local/pipx/venvs/mlx-lm/bin/mlx_lm.server
  --model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit
  --host 127.0.0.1 --port 18011
```

`GET /v1/models` returned the resolved absolute model path:

```text
/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit
```

The contract requires exact profile/target model identity:

```text
Qwen3.8-27B-MLX-8bit
```

The production `waitPiRuntimeReady` path compares model IDs exactly, so the
generated managed profile would fail `runtime_model_unavailable`. A separate
real invocation with the canonical ID as the MLX `--model` selector treated it
as a Hugging Face repository and raised `RepositoryNotFound`.

Apply result:

- command exit: 3 (expected-red refusal)
- status: `refused-before-write`
- writes: `[]`
- real MLX child: terminated and reaped (`-15`)
- port 18011 after gate: no listener (`lsof` exit 1)
- all 121 source hashes after refusal: equal to inventory hashes
- rollback command exit: 2, correctly refused because no successful apply

Authoritative reports:

- `run/applied-rewrite-report.json`
- `run/mlx-selector-probe.json`
- `run/mlx-selector-probe.log`
- `run/rollback-report.json`
- `apply-command-02.log`
- `rollback-command-01.log`

## Script and test evidence

- `rewrite_project_configs.py`: task-only inventory/dry-run/apply/rollback tool
- `test_rewrite_project_configs.py`: four tests covering hidden-path inventory,
  exact MCP preservation, wrong-model narrowing refusal at the production
  `perform_apply -> probe_qwen` call site, backup/hash, apply, and rollback
- final unit command: exit 0, 4 tests passed
- `ruff check`: exit 0
- `python3 -m py_compile`: exit 0
- `go test ./... -count=1`: exit 0
- `go vet ./...`: exit 0
- production `agents-infra` build: exit 0

Earlier non-green commands are retained honestly in the run logs: one malformed
unittest module invocation (exit 1, no tests), one initial test expectation that
did not resolve macOS `/var -> /private/var` (exit 1), one first dry-run exposing
two script orchestration defects (exit 3), and the expected-red MLX/apply and
rollback refusals above.

## Stop-the-line packet

Constraint: canonical target model, Pi profile model, Pi request model, runtime
readiness ID, and MLX load selector are required to be one exact string, but the
real MLX server uses a filesystem path for the local deployment and exposes that
resolved path as its model ID.

Rejected forced fits:

- per-project symlinks named after the canonical ID
- a task-local proxy that rewrites model IDs
- weakening exact readiness to accept any non-empty model list
- claiming parser/compose success proves runtime identity

Viable choices:

1. Recommended: revise the architecture and production implementation to
   represent canonical target identity separately from the provider-native
   request/load/readiness selector, with an explicit fail-closed mapping.
2. Select or extend an MLX-compatible server that supports an explicit stable
   model alias equal to `Qwen3.8-27B-MLX-8bit` on load, requests, and
   `/v1/models`.
3. Change the canonical target model/runtime away from the required MLX target;
   this changes R2 and needs contract approval.

Exact external decision needed: choose whether to preserve the canonical model
identity and add a provider-native selector/alias contract (options 1 or 2), or
revise R2/model runtime identity (option 3). After that dependency is accepted,
rerun this script's dry-run and apply; its hash gate prevents applying stale
candidates.
