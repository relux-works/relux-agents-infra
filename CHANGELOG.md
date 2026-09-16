# Changelog

All notable changes to this project are documented here. Version numbers are
injected at install time from git tags (`scripts/setup.sh`, `scripts/setup.ps1`
via `git describe --tags`); there is no version file to bump in-tree. The
release owner publishes the tag.

## [Unreleased] — proposed v2.0.0 (breaking)

Deprecates direct proprietary-provider execution and removes the instruction,
skill, and MCP-registry distribution responsibilities that moved to
Curator-managed homes. This is Slice A of the agents-infra migration: the
residual local runtime, Pi local-model operation, attachments, and the
task-board compose/prepare contracts stay intact.

### Deprecated (exit 1 stubs this release, removed next)

Each entrypoint below prints exactly one migration line to stderr, writes
nothing to stdout, exits 1, and performs no launch, config read, provider
resolution, build, delegation, or filesystem change. Guards run before any
argument parsing — and, for the installed wrappers, before any sibling
lookup, build-directory creation, or toolchain invocation — so
`--print-config`, `--help`, danger flags, malformed provider args, and
`spawn` all receive the same message even when the cached build output is
absent or `go` is broken. There is no `--print-config` exception. The
deprecated wrappers refuse directly in the wrapper itself; the live
`pi-infra`/`qwen-infra` wrappers and the residual `agents-infra` launcher
behavior for non-deprecated subcommands are unchanged.

- `agents-infra codex` → `curator run codex_cli -- <args>`
- `agents-infra claude` → `curator run claude_code -- <args>`
- `agents-infra target openai-infra` → `curator run codex_cli -- <args>`
- `agents-infra target anthropic-infra` → `curator run claude_code -- <args>`
- Installed `openai-infra` / `anthropic-infra` wrappers → same Curator commands
- Installed `openai-dange` / `anthropic-dange` wrappers → same Curator
  commands (danger flags are never forwarded to Curator)
- Local `.local/bin/codex-local` shim → `curator run codex_cli -- <args>`

The canonical `openai-infra` / `anthropic-infra` identifiers remain valid
for the non-launching `compose --mode primary-session --entrypoint …`
contract; deprecation intercepts launching dispatch only.

### Removed from setup / refresh-links / doctor / verify

- Instruction sync and `@include` rendering from ordinary setup and
  refresh-links (global instruction copy, local scaffold creation, provider
  entrypoint generation). The `.instructions/` sources stay in the repo as
  Curator's provenance byte source.
- Repo-skill materialization and the `.skills` → `skills` → provider fan-out,
  including stale/dangling link repair and skill-link validation. Installed
  and user-managed skill content is preserved byte-identical; nothing is
  deleted from native homes.
- Bundled `.configs/codex-mcp-servers.toml` shipping (the file is deleted from
  the repo and excluded from sync). Composition reads only caller-supplied
  global and project registries and fails closed on a missing definition.
- MCP composition from the deprecated launchers. The shared pure builders and
  registry readers stay for `compose`.
- Skill topology requirements from install verification, and instruction /
  skill markers plus the instruction include closure from the usable-source
  contract. A source tree without `.instructions/`, `SKILL.md`, `README.md`,
  or skill trees is now accepted.

### Kept (residual surface)

- `setup` / `refresh-links` / `doctor` / `verify` for the residual runtime:
  `claude-settings.json` linking, Codex `config.toml` merge
  (`--codex-config` preserve/global/local), `.rules` links, helper launchers,
  the local CLI wrapper, and the local `codex-local` shim lifecycle
  (nonempty MCP opt-ins create/preserve it; removal deletes only the
  recognized generated shim).
- `claude_linked`, `codex_linked`, `codex_rendered`,
  `codex_project_rendered`, and `infra_skill_link` doctor fields as legacy
  observations of unmanaged native-home state — never as health requirements.
- Pi local-model runtime in full: broker, profiles, targets, harness,
  `runtime status/stop/quarantine/unquarantine`, `runtime-launch`,
  `model-check` (still Pi-only), and `qwen-infra` / `target qwen-infra`.
- Attachments manifest contract and the `attachments` subcommands.
- `compose` child and `compose --mode primary-session` (including
  `--entrypoint`) contracts unchanged: same schema-1 wire shape, MCP-only
  argv, required env names, provenance, strict errors, and empty arrays.
  Golden tests still pass.
- `prepare --agent codex|claude` schema-1 contract with real evidence. Its
  instruction rendering is an explicit compatibility exception: `prepare` —
  and only `prepare` — renders instruction artifacts from project-owned
  installed inputs (creating minimal scaffold entrypoints on runtimes that
  have none), because the task-board validator still requires real artifacts.
  Full renderer removal is gated on a coordinated consumer migration, not
  part of this release.

### Docs

- README rewritten around the residual surface: what setup installs, the
  deprecation notice with the Curator migration commands and removal
  timeline, caller-supplied MCP registries, and the compose/prepare
  compatibility contracts.
- LLDB: there is no bundled LLDB wrapper or bootstrap in this revision and
  never was — caller-supplied stdio definitions keep working through
  compose; nothing to migrate.
