---
name: alexis-agents-infra
description: Shared agent infrastructure repo for Claude Code and Codex CLI. Use when updating global agent instructions, skills, symlink setup, tool configs, rules, or the generic agents attachments manifest contract and helper tooling.
triggers:
  - alexis-agents-infra
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

# alexis-agents-infra

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
cd /path/to/alexis-agents-infra
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
[codex.mcp]
enabled_servers = ["figma"]
```

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

Runtime responsibilities:

- materialize incoming files to local disk
- write `agents-attachments-manifest.json`
- export `AGENTS_ATTACHMENTS_MANIFEST`
- propagate the same manifest/env into spawned child agents

This repo's responsibilities:

- define the contract in `.instructions/INSTRUCTIONS_ATTACHMENTS.md`
- ship the helper in `.scripts/agents-attachments`
- install/symlink the helper via `.scripts/setup-symlinks.sh`

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
