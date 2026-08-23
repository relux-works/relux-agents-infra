---
name: relux-agents-infra
description: Shared agent infrastructure repo for Claude Code and Codex CLI. Use when updating global agent instructions, skills, symlink setup, tool configs, rules, or the generic agents attachments manifest contract and helper tooling.
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
agents-infra prepare --agent codex --project /path/to/project --schema-version 1 --json
agents-infra compose --agent codex --project /path/to/project --schema-version 1 --json
pi-infra --print-config
agents-infra version
```

`setup global` and `setup local` need no host-specific source path. They resolve
the source tree from `--source-dir`, then `AGENTS_INFRA_SOURCE_DIR`, then the
`repoPath` recorded in the installer's `install.json`, then the installed
`~/.agents` runtime. An explicit source that is not an agents-infra tree fails
and names what is missing instead of falling back.

A usable source carries the instruction entrypoints, `.configs`, `.rules`, the
`tools/agents-infra` Go module the generated launcher builds, the exact
217-record Pi tree manifest, and every instruction module its entrypoints
include. Anything less is refused before the destination is touched.

For local setup that module is proven by running what the launcher runs, not by
listing its files and not by compiling alone: the generated launcher builds the
module into `PROJECT/.local/bin/.agents-infra-build/` and executes the result on
every invocation, so setup and `verify local` do the same and require the binary
to answer `version` as `agents-infra`. Both therefore need a Go toolchain — the
same one the launcher needs. The launcher never writes into the source checkout.

```bash
agents-infra verify local /path/to/project
```

`verify` is the postcondition setup itself enforces before marking a runtime
complete. Gate on it rather than on `setup`'s exit code: a directory that exists
is not a runtime that works.

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
- `.local/bin/pi-infra` is the exact sibling alias for `agents-infra pi`

Project-local instructions are an overlay, not a copy of global policy:

- `setup local` skips the source `.instructions/` tree
- it creates minimal `.agents/.instructions/AGENTS.md` and
  `.agents/.instructions/INSTRUCTIONS.md` entrypoints only when missing
- it preserves all existing project instruction files on every resync
- add project-specific modules and relative includes under that local directory
- never populate local instructions by copying shared global modules

## Codex config modes

Keep local agent runtime setup separate from Codex model/reasoning config.

Every global or local source sync refreshes the managed
`.agents/.configs/codex-config.toml` defaults while merging existing project
trust decisions, TUI notice acknowledgements, and custom profiles other than
the withdrawn `fast` profile. Source remains authoritative for model,
reasoning, and service tier. Malformed installed TOML refuses synchronization
without replacing that config.

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

- `preserve` (default) preserves custom `.codex/config.toml` files, but removes a managed project-local config from a prior explicit `local` setup, including the old symlink form.
- `global` removes `.codex/config.toml`; use this when a local config unintentionally shadows the global model/settings.
- `local` atomically renders a managed regular `.codex/config.toml` from `.agents/.configs/codex-config.toml`; it omits the user-level-only top-level `profiles` table and preserves all other valid TOML settings. Malformed installed TOML leaves the existing project config unchanged. Use this mode only when project-local config is intentional.

Those modes alone govern project Codex config state. Primary launch preparation
refreshes instructions, skills, and rules while preserving an absent, managed,
custom, or linked `.codex/config.toml` exactly.

Diagnose effective state with:

```bash
agents-infra doctor local /path/to/project
```

Key fields:

- `codex_config_effective: global` means Codex uses the global `~/.codex/config.toml`.
- `codex_config_effective: project-local` means `.codex/config.toml` is active for that project.
- `codex_config_shadowing_global: true` means project-local config overrides the global config; remove it with `--codex-config=global` if unintended.
- `codex_config_generated: true` means the project-local config is the managed project-safe file rendered by agents-infra.
- `codex_config_linked: true` means the config is a symlink to the full installed config, as used by global setup or left by an older local setup.

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

## Canonical vendor targets

Use canonical targets when a project needs an explicit vendor/environment/model
identity behind an installed vendor alias. Define complete atomic targets and
exact mappings in contributing project configs:

```toml
[agents.targets.openai]
vendor = "openai"
environment = "codex"
model = "gpt-5.6-sol"
reasoning = "high"

[agents.targets.anthropic]
vendor = "anthropic"
environment = "claude-code"
model = "claude-opus-5"
reasoning = "high"

[agents.targets.qwen]
vendor = "qwen"
environment = "pi"
model = "Qwen3.8-27B-MLX-8bit"
reasoning = "off"
profile = "qwen-3.8-27b-mlx-8bit"
profile_provider = "local-qwen"
endpoint = "http://127.0.0.1:18011/v1"

[agents.entrypoints]
openai-infra = "openai"
anthropic-infra = "anthropic"
qwen-infra = "qwen"
```

Only `openai/codex`, `anthropic/claude-code`, and `qwen/pi` are admitted.
Qwen reuses an existing complete managed Pi profile: model and thinking must
match it, optional provider/endpoint values are assertions, and effective
provider/endpoint provenance remains the profile definition. A nearer target
replaces the same name atomically; a nearer mapping replaces only that alias.

Inspect without launching:

```bash
openai-infra --print-config
anthropic-infra --print-config
qwen-infra --print-config
agents-infra compose --mode primary-session --entrypoint qwen-infra \
  --project "$PWD" --schema-version 1 --json
```

Missing/malformed mappings, invalid tuples/profiles, and conflicting explicit
identity selectors fail before provider/runtime side effects and carry source,
field/identity, and remediation. Exact repeats are accepted. Pi model tokens
are decoded as `model`, `provider/model`, `model:thinking`, or
`provider/model:thinking`; divergent suffix/provider coordinates conflict.
Codex aliases also lock `-c model=...` and `-c
model_reasoning_effort=...` and always reject profile selectors.

Canonical declarations never rewrite config and never become defaults for
direct `agents-infra codex|claude|pi` or `pi-infra`; those commands retain
legacy precedence. Alias schema-v1 plans add `target` and Qwen effective
provider/endpoint fields, while legacy `--agent` plans keep their existing
shape. See [README.md](README.md#canonical-vendor-target-entrypoints) for full
field domains and diagnostics.

Task-board spawn ceilings are documented by the separate
[task-board spawn-ceiling contract](https://github.com/relux-works/skill-project-management/blob/main/.specs/project-agent-selection-policy.md#task-board-spawn-ceiling-contract).
Do not add spawn ceilings, model ranks, or task-board resolver policy to
agents-infra TOML.

## Managed Pi local-model policy

Pi launcher, setup, alias, catalog, and operator-contract changes belong in
this source repository. Never edit `~/.agents`, `~/.local/bin/pi-infra`, or a
project's generated alias as the source of truth. Change `tools/agents-infra`,
`README.md`, and this skill here, then install and verify:

```bash
agents-infra setup global --source-dir /path/to/relux-agents-infra
agents-infra verify global
agents-infra setup local /abs/path/to/project
agents-infra verify local /abs/path/to/project
```

Both setup modes install `pi-infra` beside their exact `agents-infra` target.
The alias preserves caller cwd and argv order/bytes and delegates only as
`agents-infra pi`; it never falls back through `PATH`. Setup repairs alias
drift, while setup's postcondition and `verify` reject missing/changed alias
bytes or mode and a missing, wrong, or unusable sibling target. Both paths must
themselves be regular files; a symlink is drift even when it resolves to
byte-identical content. Setup also installs and repairs the exact sibling-only
`openai-infra`, `anthropic-infra`, and `qwen-infra` aliases, each delegating as
`agents-infra target <exact-entrypoint>` with no embedded target policy. They also
validate the authoritative source/installed manifest at
`tools/agents-infra/internal/infra/pi-v0.84.2-darwin-arm64-tree-manifest.txt`
with SHA-256
`2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b`.

Managed policy lives only in ancestor
`.agents/.configs/project-config.toml` files. Profiles are atomic exact-name
definitions; nearest explicit selection wins, with wrapper `--profile` above
TOML. Exact decoded UTF-8 profile bytes are the logical identity and feed the
SHA-256 state key without normalization or lossy sanitization. State paths are
hash-only beneath the canonical user cache root and are opened through the
anchored no-follow containment contract. Do not add raw profile-name path
components, shared locks for byte-distinct names, a fallback for read/path
failure, or project-controlled catalog evidence.

Use this order for every new or changed deployment:

```bash
cd /abs/path/to/project

# Required first step: same static resolution/validation as launch, but no
# process, file, lock, socket, connection, download, or Pi trust mutation.
pi-infra --print-config

# Equivalent machine-readable resolver for a session owner.
agents-infra compose --mode primary-session --agent pi \
  --project "$PWD" --schema-version 1 --json

# Only after reviewing exact provenance, selected profile/model, absolute
# runtime executable, literal argv, loopback URL, hash-only state paths,
# requested/unverified capabilities, and standalone Pi catalog identity.
pi-infra
```

The first ASCII `--` is a wrapper-only operand boundary because the pinned Pi
parser has no end-of-options state. It is not forwarded. Suffix tokens are
preserved only when they begin with neither ASCII `-` nor `@`; ambiguous option
consumption, repeated separators, or option-looking suffixes fail before side
effects. Never simplify this to pass-through `--`, prefix-only scanning, or a
shell command string.

The configured absolute runtime executable plus literal argv is reviewed
trusted policy. agents-infra reproducibly checks/spawns that selection, refuses
any runtime argv that does not contain exactly one spaced `--host 127.0.0.1`
pair and one spaced `--port <base_url-port>` pair, refuses an occupied exact
loopback listener, requires the direct child alive and exact model readiness,
retries only connection failures and HTTP 503 while the model loads,
owns/reaps its process group, isolates Pi state, and performs the deterministic
standalone Pi tree check both initially and immediately before Pi spawn. It
never attaches or silently falls back to another runtime, port, profile, model,
listener, or Muse target-only decoding.

Managed launch refuses the exact llama.cpp model-origin environment names
`HF_ENDPOINT` and `MODEL_ENDPOINT` before runtime spawn and reports only the
name, never the value. Treat tokens and cache-location variables separately
unless their runtime effect is established; do not widen this rule by naming
convention alone.

Exact `GGML_BACKEND_PATH` is also refused before managed state or runtime spawn:
llama.cpp build 10470 passes its inherited value to `dlopen()` during backend
discovery. Other `GGML_*` names remain outside this exact-name policy until a
runtime effect is established; do not turn this into a speculative prefix gate.

Exact `LLAMA_API_KEY` is refused before managed state or runtime spawn because
llama.cpp build 10470 uses it as the environment backing for `--api-key`.
Managed profiles must not inherit ambient runtime authentication absent from
their reviewed configuration, and refusal must name only `LLAMA_API_KEY`, never
its value. Keep `HF_TOKEN`, cache-location variables, `LLAMA_API_KEY` lookalikes,
and unrelated names admitted unless their runtime effect establishes a separate
policy.

Capabilities remain requested/configured, never independently verified:
Qwen `text`/`tools`; Muse `dflash`/`text`/`tools` with
`dflash.status = configured-unverified`. Runtime reports are unverified
diagnostic provenance. Operators must run real Pi text plus tool round trips
for each profile and a runtime-specific Muse benchmark/telemetry check. Do not
invent an attestation API or claim that readiness detects silent DFlash
disablement.

The practical trust boundary explicitly excludes a malicious selected runtime
and a malicious same-UID process that wins the post-preflight bind race.
Preflight plus readiness is not cryptographic listener ownership. Model/runtime
acquisition or conversion, benchmark automation, secure runtime distribution,
backend catalogs, compiled observers, proxy adapters/private-pipe authorities,
and runtime/DFlash cryptographic attestation are outside this contract. Do not
reintroduce the retired cycle-7 backend/observer/proxy/attestation design into
TOML, diagnostics, examples, or acceptance gates. Unknown/unverified is
diagnostics-only, not satisfied evidence.

The exact TOML, compatibility catalog, state-key rules, lifecycle, operator
smoke procedure, full error list, and security limitations are authoritative in
[README.md](README.md#managed-pi-local-model-operator-contract). Start every Pi
incident with `pi-infra --print-config`, then `agents-infra verify local "$PWD"`.

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

Child runners that already own model, safety, prompt, and lifecycle arguments
must use the non-launching, versioned MCP-only contract:

```bash
agents-infra compose --agent codex --project /path/to/project --schema-version 1 --json
agents-infra compose --agent claude --project /path/to/project --schema-version 1 --json
```

The command emits one
`agents-infra.child-launch-composition` version-1 JSON document and never
launches a provider. Codex output contains only `-c mcp_servers.*` pairs;
Claude output is empty or one `--mcp-config` pair. Safe metadata exposes
server names, source paths, and referenced bearer-token environment variable
names, but primary-session policy, provider user args, and environment values
are excluded.

Session managers that own the primary provider process themselves use the
primary-session mode instead:

```bash
agents-infra compose --mode primary-session --agent codex|claude --project /path/to/project --schema-version 1 --json [-- PROVIDER_ARGS...]
```

It emits one `agents-infra.primary-session-launch-plan` version-1 JSON
document: the resolved provider executable, an `interactive` argv identical to
what `agents-infra codex|claude` would launch, a `managed_host` argv
(`codex-app-server` or `claude-pty`) plus a `managed_client` argv for
thread/client tokens (for Codex the classification is total — every
interactive token lands in host argv, client argv, or resolved session
policy, never silently dropped; for Claude the client fragment is always
empty), the resolved model/reasoning/yolo/sandbox/profile/approval/MCP policy
with per-field provenance (including attached `-mVALUE`/`-pVALUE` forms),
required environment variable names (never values; composed MCP bearer-token
references plus a valid Codex `--remote-auth-token-env` name, de-duplicated,
with post-`--` tokens never interpreted), and the contributing
config sources. Codex `--profile` values are additionally validated against
the provider's plain profile-name syntax in every spelling, failing closed
with `invalid_provider_arguments` on names the Codex parser rejects. Pass-through native policy selections (Codex
`--sandbox`/`--ask-for-approval` flags and `-c sandbox_mode=`/`-c
approval_policy=` overrides; Claude `--effort`/`--permission-mode`) are
reflected into `resolved` with provider-faithful precedence and duplicate
handling, and an explicit policy selection suppresses project-config
`yolo_mode` so the bypass flag is never composed next to it. Policy values
are validated against the provider-accepted domains (typed-flag enums versus
config-override variants for Codex, case-sensitive permission-mode choices
for Claude) and fail closed with `invalid_provider_arguments` when the
provider itself would reject them; an unknown Claude `--effort` value, which
Claude ignores with a warning, keeps its argv token but reports
`resolved.reasoning` as provider-native instead of effective. No launch is
performed and no board or goal state is read.

Before a session manager starts or connects to a primary provider host, it must
refresh the same installed project surface as the direct launcher:

```bash
agents-infra prepare --agent codex|claude --project /path/to/project --schema-version 1 --json
```

This board-agnostic, non-launching contract emits one
`agents-infra.primary-session-preparation` version-1 report. With an installed
project `.agents/` runtime it refreshes the provider-specific managed surface:
Codex instructions and skills/rules; or the Claude entrypoint,
instruction/settings links, and managed skills. Codex preparation preserves
the exact pre-existing `.codex/config.toml` state, including absence; only
explicit `setup local --codex-config=preserve|global|local` selects that mode.
The report marks the config artifact `absent` or `preserved`. Without a local
runtime it reports an explicit no-op. The nearest installed ancestor runtime
is selected and the global `~/.agents` runtime is never treated as a project.
Direct `agents-infra codex|claude` launches call the same preparation function
immediately before provider exec; `--print-config` remains read-only.

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
- ship the helper in the Go `agents-infra attachments` subcommand plus the
  generated backwards-compatible `agents-attachments` launcher
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
- [tools/agents-infra/internal/attachments](tools/agents-infra/internal/attachments)
