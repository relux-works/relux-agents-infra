---
name: relux-agents-infra
description: Shared agent infrastructure repo for Claude Code and Codex CLI. Use when updating global agent instructions, skills, symlink setup, tool configs, rules, or the generic agents attachments manifest contract and helper tooling.
triggers:
  - relux-agents-infra
  - agents infra
  - agent infrastructure
  - shared agent config
  - global agent instructions
  - codex config
  - claude settings
  - setup symlinks
  - agents attachments manifest
  - attachments manifest
  - агентская инфра
  - конфиг агентов
  - настройки codex
  - настройки claude
---

# relux-agents-infra

Source repo for the shared agent infrastructure that installs into `~/.agents`, `~/.claude`, and `~/.codex`.

Do not edit `~/.agents` directly when changing shared instructions, configs, or skills.
Work in the source repo, then run `agents-infra setup global` or `./setup.sh`
to sync the installed runtime copy.

Use this repo when you need to:

- update global instructions in `.instructions/`
- add or adjust shared skills in `.skills/`
- change Codex or Claude configuration in `.configs/`
- update the Go CLI in `tools/agents-infra/`
- update symlink/bootstrap logic in `.scripts/setup-symlinks.sh`, `scripts/setup.sh`, `scripts/setup.ps1`, or `setup.sh`
- use `agents-infra setup global|local` to sync and refresh installed links
- maintain the generic `agents-attachments-manifest.json` contract and helper tooling

## Quick start

```bash
cd /path/to/relux-agents-infra
./setup.sh
.\\setup.ps1

# Canonical interface after bootstrap
agents-infra setup global
agents-infra setup local /path/to/project
agents-infra doctor global
agents-infra doctor local /path/to/project
agents-infra version
```

This repo is setup/configuration infrastructure, not the runtime that launches agent sessions.
`~/.agents` is the installed destination, not the place to author shared changes.

`./setup.sh` and `.\setup.ps1` are bootstrap wrappers: they delegate into the
cross-platform scripts under `scripts/`, build the `agents-infra` binary with
embedded version metadata, install it into the user-local bin dir, write
install-state metadata, and then immediately run `agents-infra setup global`.

For project-local setup, install into the target repo so that:
- `.agents/` holds the actual installed runtime contents
- `.claude/` and `.codex/` are just thin shims/symlinks into `.agents`
- `.local/bin/` exposes helper CLIs for that local setup, including `agents-infra`

## Codex config modes

Keep local agent runtime setup separate from Codex model/reasoning config.

Default project-local setup should not create `.codex/config.toml`:

```bash
agents-infra setup local /path/to/project
# same as:
agents-infra setup local /path/to/project --codex-config=preserve
```

Use the explicit modes when the user asks about Codex config/model drift:

```bash
# Remove any project-local Codex config so ~/.codex/config.toml is authoritative.
agents-infra setup local /path/to/project --codex-config=global

# Intentionally make Codex model/reasoning settings project-local.
agents-infra setup local /path/to/project --codex-config=local
```

Mode semantics:

- `preserve` (default) preserves custom `.codex/config.toml` files, but removes the old managed symlink `.codex/config.toml -> .agents/.configs/codex-config.toml`.
- `global` removes `.codex/config.toml`; use this when a local config unintentionally shadows the global model/settings.
- `local` links `.codex/config.toml` to `.agents/.configs/codex-config.toml`; use only when project-local model/reasoning config is intentional.

Diagnose effective state with:

```bash
agents-infra doctor local /path/to/project
```

Key fields:

- `codex_config_effective: global` means Codex uses the global `~/.codex/config.toml`.
- `codex_config_effective: project-local` means `.codex/config.toml` is active for that project.
- `codex_config_shadowing_global: true` means project-local config overrides the global config; remove it with `--codex-config=global` if unintended.
- `codex_config_linked: true` means the project-local config is the managed agents-infra symlink, not a custom file.

## Provider-specific primary-session policy

Project policy for primary `agents-infra codex` and `agents-infra claude`
sessions is optional and belongs only in `.agents/.configs/project-config.toml`.
It is separate from provider-native config and does not choose task-board
child-spawn models.

```toml
[mcp]
enabled_servers = ["figma"]

[agents.codex.primary_session]
model = "gpt-5.6-terra"
reasoning_effort = "xhigh"
yolo_mode = false

[agents.claude.primary_session]
model = "claude-opus-4-6"
yolo_mode = false
```

Each table needs at least one supported field. Model and Codex
`reasoning_effort` values are non-empty strings; both providers accept
`yolo_mode` as an unquoted TOML boolean. Providers remain the authority for
model availability and Codex model/effort compatibility.

Both launchers walk from filesystem root to their current directory, combining
every project config they find except `~/.agents/.configs/project-config.toml`.
The nearest explicit field wins; omitted fields inherit, and `yolo_mode = false`
explicitly masks an inherited `true`. A malformed or invalid discovered config
fails before launch.

For model and reasoning, precedence is explicit wrapper CLI selection
(`--model`/`-m`, top-level `-c model=...`, or top-level
`-c model_reasoning_effort=...`) before project TOML before Codex-native
resolution. `--profile`/`-p` suppresses project model and reasoning but not
explicit values supplied with it; it does not suppress yolo. Equal explicit
duplicates collapse, conflicting values fail, and only arguments before `--`
take part in model/reasoning/profile wrapper resolution.

Yolo defaults to safe for both providers. `-d`, `--danger`, `--yolo`, or the
matching native dangerous flag opt an invocation in; otherwise only effective
`yolo_mode = true` does. The result contains exactly one
`--dangerously-bypass-approvals-and-sandbox` for Codex or
`--dangerously-skip-permissions` for Claude when enabled. Each persistent
setting is limited to its matching primary launch and never propagates to
`task-board spawn`, run manifests, or spawn-ceiling policy.

Use supported setup flags for precise local mutation:

```bash
agents-infra setup local /path/to/project \
  --codex-primary-model gpt-5.6-terra \
  --codex-primary-reasoning-effort xhigh \
  --codex-yolo-mode=false \
  --claude-primary-model claude-opus-4-6 \
  --claude-yolo-mode=false

agents-infra setup local /path/to/project --clear-codex-primary-session
agents-infra setup local /path/to/project --clear-claude-primary-session
```

No primary-session flag preserves project-config bytes. Set flags update only
their supplied field; explicit false is preserved. Clear removes only the
primary-session table and conflicts with set flags. All primary flags are
local-only and reject the global `~/.agents` config path. Parse and atomic-write
failures preserve the original TOML.

Use these operators before diagnosing or changing session behavior:

```bash
cd /path/to/project
agents-infra codex --print-config
agents-infra doctor local "$PWD"
agents-infra codex
```

`--print-config` is non-launching: it shows discovered paths, field provenance,
CLI/profile suppression where applicable, yolo expansion, and final argv.
Doctor reports each provider's primary-session model and yolo values with their
sources (and Codex reasoning); absent strings use source `native`, and absent
yolo is `false` from `default`. For complete troubleshooting and
`.codex/config.toml` coexistence, see
[README.md](README.md#project-primary-codex-session-policy).

Task-board spawn ceilings are documented by the separate
[task-board spawn-ceiling contract](https://github.com/relux-works/skill-project-management/blob/main/.specs/project-agent-selection-policy.md#task-board-spawn-ceiling-contract).
Do not add spawn ceilings, model ranks, or task-board resolver policy to
agents-infra TOML.

## MCP server policy

MCP servers managed by agents-infra are project-local opt-in. Agents-infra does
not enable MCP servers in the global Codex config.

Reason: MCP servers add tool/context surface area. A project should expose only
the MCPs it actually needs. User-managed global MCP servers may still exist in
Codex's base config, but agents-infra project opt-in should not create global
defaults.

Use this pattern:

- Keep known MCP server definitions in the agents-infra source registry:
  `.configs/codex-mcp-servers.toml`.
- Enable MCP servers per project through:
  `.agents/.configs/project-config.toml`.
- Run `agents-infra setup local ...` after changing project MCP config.
- Start Codex through `agents-infra codex` from inside the project tree.
  It walks upward from the current directory, composes every discovered
  `.agents/.configs/project-config.toml`, resolves enabled MCP definitions from
  project registries plus the global registry, logs provenance for every config
  part, then launches `codex` with the resulting `-c` overrides.
- Use `agents-infra codex --print-config` to inspect the composed config without
  launching Codex.
- Use `agents-infra codex -d ...` as the shorthand for Codex yolo mode
  (`--dangerously-bypass-approvals-and-sandbox`).
- `.local/bin/codex-local` is only a backward-compatible shim; it delegates to
  `agents-infra codex`.
- Project-local helpers must preserve the caller working directory even when
  they run this source checkout via `go run`; `codex-local --print-config`
  should report the directory where it was invoked, not
  `.agents/tools/agents-infra`.

Example project config:

```toml
[mcp]
enabled_servers = ["figma"]
```

For day-to-day use, projects should keep MCP opt-in in
`.agents/.configs/project-config.toml` and users should start Codex through the
agents-infra launcher, not through plain `codex`. The launcher renders the
composed project config and applies it to Codex with `-c` overrides for that
session.

Recommended project flow:

```bash
# From the project root after editing .agents/.configs/project-config.toml
agents-infra setup local "$PWD"

# Inspect the rendered config without launching Codex
agents-infra codex --print-config

# Launch Codex with the rendered project-local MCP config applied
agents-infra codex
agents-infra codex -d -
agents-infra codex exec "inspect the enabled MCP tools"
```

If the user wants the normal `codex` command to always apply project-local MCP
config, add a shell function to `~/.zshrc` or `~/.bashrc`. Use a function rather
than a plain alias so arguments are forwarded correctly:

```bash
codex-raw() {
  command codex "$@"
}

codex() {
  agents-infra codex "$@"
}
```

After reloading the shell, `codex --print-config`, `codex -d -`, and
`codex exec ...` will go through `agents-infra codex`; `codex-raw ...` remains
available when the user explicitly wants the unwrapped Codex CLI. Do not add MCP
servers to global `~/.codex/config.toml` just to make plain `codex` work.

Definitions may be streamable HTTP servers with `url` or stdio servers with
`command` and optional `args`. `lldb` is available as an opt-in stdio definition
using `command = "lldb-mcp"`. On macOS, `./setup.sh` installs Homebrew `llvm`
when needed and writes an `lldb-mcp` wrapper into the Homebrew bin directory.
The wrapper execs Homebrew's helper without overriding `LLDB_EXE_PATH`, so the
helper uses the `lldb` binary next to itself by default, and it prunes dead-PID
`~/.lldb/lldb-mcp-*.json` discovery files before launch. Set
`AGENTS_INFRA_SKIP_LLDB_MCP=1` to skip that bootstrap. Projects may override the
registry locally with an absolute helper path when needed.

`safari` is available as an opt-in stdio definition using Safari Technology
Preview's `safaridriver`:

```toml
[servers.safari]
command = "/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver"
args = ["--mcp"]
```

Safari prerequisites:

- Install Safari Technology Preview 247 or newer.
- Enable `Safari Settings > Advanced > Show features for web developers`.
- Enable `Safari Settings > Developer > Enable remote automation and external agents`.

Projects opt in with `enabled_servers = ["safari"]`. Safari is not enabled
globally by agents-infra.

Expected behavior:

- Plain `codex mcp list` remains empty unless the user explicitly configured
  global MCPs outside agents-infra.
- Project-local MCPs are mounted only when starting Codex through
  `agents-infra codex` from a directory covered by local project config.
- If no local project config is found, agents-infra does not mount an MCP server
  just because it exists in a registry.
- `agents-infra doctor local /path/to/project` reports the opt-in list through
  `codex_mcp_enabled`.

## Attachments Contract

Incoming user files are modeled as a generic manifest, not as board-specific state.

- Manifest file name: `agents-attachments-manifest.json`
- Environment variable: `AGENTS_ATTACHMENTS_MANIFEST`
- Default project-local fallback: `.temp/agents-attachments-manifest.json`
- Helper CLI installed from this repo: `agents-attachments`
- Codex bootstrap helper: `agents-attachments materialize`
- Image staging helper: `agents-attachments stage-images`

Runtime responsibilities:

- materialize incoming files to local disk
- write `agents-attachments-manifest.json`
- export `AGENTS_ATTACHMENTS_MANIFEST`
- propagate the same manifest/env into spawned child agents

This repo's responsibilities:

- define the contract in `.instructions/INSTRUCTIONS_ATTACHMENTS.md`
- ship the helper in `.scripts/agents-attachments`
- install/symlink the helper via `.scripts/setup-symlinks.sh`

Image intake workflow:

- use `agents-attachments stage-images` for explicit local paths or generic
  manifest references before inspecting images
- keep originals read-only and inspect staged files under caller-controlled
  scratch, usually `.temp/image-intake`
- use the generated mapping JSON to audit source-to-staged relationships
- normalize HEIC/HEIF to PNG through macOS `sips` or ImageMagick fallback
- prefer direct runtime vision first; use OCR only as a bounded fallback
- tie observations to staged filenames with evidence, confidence, uncertainty,
  and redaction notes
- redact ICCID, IMSI, QR payloads, activation codes, tokens, keys, passwords,
  and similar secrets before persisting or reporting extracted values
- keep the workflow board-agnostic; do not require task-board IDs, resources,
  statuses, or directory conventions beyond caller-provided scratch paths

## Key Paths

- `.instructions/` — global instruction modules
- `.configs/` — Codex/Claude config files
- `.rules/` — Codex rules
- `.scripts/` — setup and helper tooling
- `.skills/` — source-managed shared skills versioned in this repo
- `skills/` — external skills/tooling area in installed runtimes; not versioned by this repo

## References

- [README.md](README.md)
- [.instructions/INSTRUCTIONS_ATTACHMENTS.md](.instructions/INSTRUCTIONS_ATTACHMENTS.md)
- [.scripts/agents-attachments](.scripts/agents-attachments)
