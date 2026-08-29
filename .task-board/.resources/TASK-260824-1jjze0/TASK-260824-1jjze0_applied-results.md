# TASK-260824-1jjze0 — Applied rollout results

## Outcome

The one-time operator rollout rewrote all 121 in-scope
`/Users/alexis/src/**/.agents/.configs/project-config.toml` files. The
inventory included hidden and ignored paths and excluded `.git`, `.temp`, and
the dependency/build/cache directory names recorded in `run/inventory.json`.

The rewrite retained every non-`agents` block byte-for-byte, proved raw and
parsed MCP equality per file, and replaced the full `agents` configuration
with:

- OpenAI / Codex / `gpt-5.6-sol` / `high`
- Anthropic / Claude Code / `claude-opus-5` / `high`
- Qwen / Pi / managed local MLX 8-bit / `off`
- exact `openai-infra`, `anthropic-infra`, and `qwen-infra` mappings

Per the operator architecture decision recorded on the task after the first
fail-closed run, the Qwen target/profile model identity is the real resolved
weights path:

```text
/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit
```

This exactly matches the installed `mlx_lm.server` load, request, and
readiness identity. No runtime migration or startup rewrite code was added.

## Rollout evidence

| Evidence | Result |
| --- | ---: |
| Inventory | 121 regular configs |
| Git worktrees represented | 116 |
| Dry-run candidates | 121 |
| Dry-run production alias composes | 363/363 exit 0 |
| Applied atomic writes | 121 |
| Per-file backups | 121 |
| Post-apply actual-path alias composes | 363/363 exit 0 |
| Validation failures / skips | 0 / 0 |
| Current/candidate/backup hash checks | 121/121 passed |
| Raw MCP backup/current comparisons | 121/121 passed |
| Unrelated `git status` lines | preserved |
| Rollback readiness | passed |

The apply command exited 0. Before its first write, it started the real
`/Users/alexis/.local/pipx/venvs/mlx-lm/bin/mlx_lm.server`, loaded the exact
local weights path, observed that exact ID from `/v1/models`, and sent a
minimal `/v1/chat/completions` request using the same model selector. The
request returned HTTP 200 with one choice. The child was then terminated and
reaped (`-15`); pre/post `lsof` returned exit 1 because no listener remained
on port 18011.

The non-mutating `verify` command exited 0 with:

```text
verification_status=passed writes=121 validations=363
```

## Validation commands

| Command | Exit | Result |
| --- | ---: | --- |
| `python3 test_rewrite_project_configs.py` | 0 | 5 tests passed, including exact-model refusal and post-apply drift refusal at production call sites |
| `ruff check rewrite_project_configs.py test_rewrite_project_configs.py` | 0 | clean |
| `python3 -m py_compile ...` | 0 | clean |
| task script `dry-run` | 0 | 121 candidates; 363 production composes |
| task script `apply` | 0 | real MLX gate then 121 atomic writes |
| task script `verify` | 0 | hashes, MCP, backup, compose, worktree status, rollback readiness |
| `go test ./... -count=1` | 0 | all agents-infra packages passed |
| `go vet ./...` | 0 | clean |
| post-rollout `go build` | 0 | production binary built |

One earlier invocation, `python3 -m unittest .temp/.../test_rewrite_project_configs.py`,
exited 1 before test discovery because `unittest` interpreted the dot-prefixed
filesystem path as an empty module name. It was a command-form error, not a
test failure; the test file was then run directly and passed all five tests.
The two `lsof` exit-1 checks are expected-red absence probes and are not
reported as passing commands.

## Report integrity

| Report | SHA-256 |
| --- | --- |
| `run/inventory.json` | `55242f6a7971bfd34fc6b4bc4cd9e8a6385a482c1c1c246571765753eb21442e` |
| `run/dry-run-report.json` | `5aa2006a40b507e165c97aff441e6a56fca18e1b86deb97a07f00062016805c9` |
| `run/applied-rewrite-report.json` | `ed5892cc32d00ff88956b2b2421f7906a44e343dd84f3b421a7b64ac33c6f111` |
| `run/verification-report.json` | `cc5bd59f6df0af9cb1014621df7e710f9a9a4046e8627c993ee16544a04378ab` |

The evidence archive contains the script, tests, complete inventory, dry-run,
applied rewrite, validation, MLX, backup, candidate, and verification reports.
Backups are retained under `run/backups/`; rollback refuses any file whose
current hash diverges from the recorded applied hash.
