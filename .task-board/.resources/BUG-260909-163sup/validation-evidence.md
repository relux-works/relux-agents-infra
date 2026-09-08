# Validation evidence — BUG-260909-163sup / TASK-260909-z10c1u

## Why this is a local mirror

GitHub Actions reports no checks on the branch:

```
$ gh pr checks 38
no checks reported on the 'task-board/bug/BUG-260909-163sup' branch
```

The repository owner states the account's GitHub Actions minutes are exhausted,
so hosted CI cannot execute repository steps. That is an external cause the
agent cannot repair, so the configured validation commands were mirrored
locally. This local evidence supplements unavailable hosted execution; it does
not satisfy a required hosted status and does not authorize bypassing review or
branch protection.

## Run identity

| Field | Value |
| --- | --- |
| PR | relux-works/relux-agents-infra#38 |
| PR head | d943c161748d36fadf41440b33d56cb7d9a72234 |
| Base | main @ 0936a41 |
| Checkout | clean linked worktree at .temp/BUG-260909-163sup/worktree, cut from origin/main |
| Platform | macOS 26.5.1, arm64 |
| Toolchain | go1.25.5 darwin/arm64 |
| Commands | the two configured in spawn.worktree_isolation.validation.commands |

No secrets or non-default environment configuration were required by these
commands.

## Results

```
$ cd tools/agents-infra && go vet ./...
VET_OK                                    (exit 0)

$ cd tools/agents-infra && go test ./... -count=1
ok      .../tools/agents-infra                      107.200s
?       .../tools/agents-infra/cmd/model-harness    [no test files]
ok      .../tools/agents-infra/internal/attachments   0.968s
ok      .../tools/agents-infra/internal/infra       208.079s
ok      .../tools/agents-infra/internal/modelharness 13.976s

[exited with code 0]
```

## Control plants

Each product change was reverted independently and the tests that claim to
cover it were required to fail. Both produced the intended failure, not a
skipped test, unrelated setup error, or crash.

Reverting `primary_session_prepare.go` to origin/main:

```
--- FAIL: TestPreparePrimarySessionTreatsConfigOnlyAncestorAsNoop
    prepare Codex project surface: read instruction file
    .../.agents/.instructions/AGENTS.md: no such file or directory
--- FAIL: TestPreparePrimarySessionSkipsConfigOnlyAncestorForInstalledRuntime
    prepare Codex project surface: read instruction file
    .../teams/.agents/.instructions/AGENTS.md: no such file or directory
```

Reverting `infra.go` to origin/main:

```
--- FAIL: TestPreparePrimarySessionRendersSurfaceWithoutManagedSkillsTree/claude
    prepare Claude project surface: open .../.agents/skills: no such file or directory
--- FAIL: TestPreparePrimarySessionRendersSurfaceWithoutManagedSkillsTree/codex
    prepare Codex project surface: open .../.agents/skills: no such file or directory
```

The second control reproduces the reported user-facing error string exactly.

## End-to-end check on the reporting configuration

A binary built from the PR head was run against the real project that reported
the defect (`/Users/alexis/src/local-models`, nested under the `~/src`
meta-config). The installed binary is v1.6.1-128-gab60e0d and still fails.

| Check | Installed v1.6.1-128-gab60e0d | PR head build |
| --- | --- | --- |
| `prepare --agent claude` | `render_failed` | `status: ok`, `local_runtime_present: false`, 0 artifacts |
| `prepare --agent codex` | `render_failed` | `status: ok`, `local_runtime_present: false`, 0 artifacts |
| `claude_primary_model` | `claude-fable-5-1` | `claude-fable-5-1` |
| `claude_primary_model_source` | `~/src/.agents/.configs/project-config.toml` | unchanged |
| `codex_primary_model` | `gpt-5.6-sol` | `gpt-5.6-sol` |
| `codex_primary_model_source` | `~/src/.agents/.configs/project-config.toml` | unchanged |
| Provider surface written into `~/src` | yes | no |

Both halves of the contract hold: configuration is still inherited from the
meta-config, and no surface is materialized into it.

## Follow-up outside this PR

The installed `agents-infra` binary predates this fix, so the machine keeps
failing until it is rebuilt and reinstalled from the landed trunk. That is a
human action taken outside a manager-hosted session.
