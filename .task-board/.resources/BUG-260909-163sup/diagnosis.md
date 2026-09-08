# Diagnosis: parent meta-config hijacks the project runtime root

Investigated on 2026-09-09 against `agents-infra v1.6.1-128-gab60e0d`.
The defect is present on `origin/main` (0936a41) unchanged.

## Reproduction

Two directories matter:

- `/Users/alexis/src/local-models` — a real project. It has **no** `.agents`,
  `.claude`, or `.codex` of its own.
- `/Users/alexis/src` — its ancestor. It carries a deliberate **meta-config**
  shared by every repository beneath it:

```
/Users/alexis/src/.agents/
└── .configs/
    ├── project-config.toml          # primary-session policy for all nested projects
    ├── model-harness.toml
    └── project-config.toml.bak-260830
```

Running the launcher from the project:

```
$ agents-infra prepare --agent claude --project . --schema-version 1 --json
{"contract":"agents-infra.primary-session-preparation","status":"error",...,"error":{"code":"render_failed"}}
prepare primary-session project surface: prepare Claude project surface: open /Users/alexis/src/.agents/skills: no such file or directory
```

The same failure reaches the user through `anthropic-infra`:

```
prepare canonical claude target project surface: prepare Claude project surface: open /Users/alexis/src/.agents/skills: no such file or directory
```

and through the task-board Session Manager, which calls the same
`PreparePrimarySession`:

```
agents-infra primary-session preparation failed: prepare primary-session project surface: prepare Claude project surface: open /Users/alexis/src/.agents/skills: no such file or directory
```

## Root cause

`installedProjectRuntimeRoot()` in
`tools/agents-infra/internal/infra/primary_session_prepare.go` walks upward
from the project directory and returns the **first ancestor containing a
`.agents` directory**, stopping at `$HOME`. It performs no check on what that
`.agents` actually is:

```go
agentsDir := filepath.Join(current, ".agents")
info, err := os.Stat(agentsDir)
if err == nil {
    if !info.IsDir() { ... }
    return current, true, nil          // <- any .agents wins
}
```

`PreparePrimarySession` then feeds that ancestor into `LocalLayout` and
materializes the **entire provider surface** there instead of in the project.

This conflates two separate concerns that must not share one resolution:

1. **Config inheritance** — reading `project-config.toml` from an ancestor
   `.agents/.configs`. This already works and is intended. `agents-infra doctor
   local /Users/alexis/src/local-models` reports it correctly:

   ```
   claude_primary_model: claude-fable-5-1
   claude_primary_model_source: /Users/alexis/src/.agents/.configs/project-config.toml
   codex_primary_model_source: /Users/alexis/src/.agents/.configs/project-config.toml
   ```

2. **Surface materialization** — rendering `CLAUDE.md`, linking `instructions`,
   `settings.json`, and fanning out skills. This must always happen in the
   project directory, and currently follows (1) to the ancestor.

## Secondary defect

`setupClaude()` in `tools/agents-infra/internal/infra/infra.go` creates the
destination skills directory but then hard-fails when the *source* skills
directory is absent:

```go
claudeSkillsDir := filepath.Join(layout.ClaudeDir, "skills")
os.MkdirAll(claudeSkillsDir, 0o755)          // destination: created
...
skillsDir := filepath.Join(layout.AgentsDir, "skills")
entries, err := os.ReadDir(skillsDir)        // source: hard error if missing
if err != nil { return err }
```

That is the proximate `open .../skills: no such file or directory`.
`setupCodexWithConfig()` has the identical shape and the same exposure.

## Observable collateral

Because the ancestor became the project root, `/Users/alexis/src/.claude/` and
`/Users/alexis/src/.codex/` were materialized there. `~/src/.claude/CLAUDE.md`
is now loaded as **project instructions for every repository under `~/src`**,
which is not what a meta-config directory should produce.

## What the meta-config is, and is not

The ancestor `.agents/.configs` is a legitimate, intentional pattern: one
primary-session policy shared by every project in the tree. It is **not** a
broken or partial installation, and the fix must not treat it as one.

Note that `agents-infra verify local /Users/alexis/src` reports it as an
unusable runtime (no install receipt, no `.instructions`, no `.rules`, no
`SKILL.md`, no `.skills`). That report is correct *as a runtime check* and is
useful evidence, but "not a runtime" must resolve to "keep walking / use this
only for config", never to "fail the nested project".

## Work already started in the main checkout

The main checkout carries an **uncommitted** change that is a partial fix for
this exact defect, and its intent matches the required contract. Do not
duplicate it blindly, and do not discard it either — reconcile with it.

`primary_session_prepare.go` gates the upward walk on a completed-install
receipt, so a config-only ancestor is skipped rather than adopted:

```go
if len(verifyRuntimeReceipt(layout, agentsDir)) == 0 {
    return current, true, nil
}
```

`primary_session_prepare_test.go` already adds two negative tests:

- `TestPreparePrimarySessionTreatsConfigOnlyAncestorAsNoop`
- `TestPreparePrimarySessionSkipsConfigOnlyAncestorForInstalledRuntime`

Both assert that preparation did not write into the ancestor, not merely that
no error was returned. `main_test.go` adds the receipt to a shared fixture.

`README.md` and `SKILL.md` already state the contract:

> The command walks from `--project` toward the filesystem root and selects the
> nearest project `.agents/` runtime with a valid completed-install receipt. A
> config-only ancestor such as `.agents/.configs/project-config.toml` still
> contributes launch policy during composition, but preparation skips it
> instead of treating it as an installed runtime.

That is the intended behavior. What remains:

1. The secondary `setupClaude` / `setupCodexWithConfig` missing-skills-directory
   defect is **untouched** — `tools/agents-infra/internal/infra/infra.go` is not
   among the modified files. With the receipt gate in place the crash is no
   longer reachable by this path, but the fragility stands and the checklist
   requires it fixed with its own negative test.
2. A positive control proving a nested project still *inherits* the ancestor
   `project-config.toml` while materializing its own surface.
3. Documentation of the meta-config as a supported pattern with a worked
   example (tracked separately as TASK-260909-z10c1u).
4. Building and reinstalling the binary, since the shipped
   v1.6.1-128-gab60e0d predates all of this.

## Checkout state, for the record

The main checkout of `relux-agents-infra` is 8 commits behind `origin/main` and
is dirty (243 paths). `README.md` and `.instructions/INSTRUCTIONS_TESTING.md`
are modified locally *and* changed by the incoming commits, so the main
checkout must not be fast-forwarded or reset. All work belongs in an isolated
worktree cut from `origin/main`.
