# relux-agents-infra

Source repo for shared AI agent configurations, instructions, skills, and rules.

Works with:
- **Claude Code** (`~/.claude/`)
- **Codex CLI** (`~/.codex/`)

## Quick Start

```bash
# Bootstrap the launcher, then immediately sync the global runtime
cd /path/to/relux-agents-infra
./setup.sh

# Windows bootstrap
.\setup.ps1

# Bootstrap and also install the optional PDF toolchain
./setup.sh --with-pdf-tools

# Use the installed CLI after bootstrap
agents-infra setup global
agents-infra setup local /path/to/project
agents-infra doctor global
agents-infra doctor local /path/to/project
agents-infra version
```

`setup.sh` and `setup.ps1` are bootstrap wrappers. They delegate into
`scripts/setup.sh` and `scripts/setup.ps1`, build a real `agents-infra` binary
with version metadata, install it into the user-local bin directory, write
install-state metadata, and then immediately run `agents-infra setup global`.

Install-state metadata lives under the standard user config directory:

- macOS: `~/Library/Application Support/agents-infra/install.json`
- Windows: `%AppData%\agents-infra\install.json`

The canonical interface after bootstrap is:
- `agents-infra setup global`
- `agents-infra setup local [PATH]`
- `agents-infra doctor global|local`
- `agents-infra codex [--print-config] [-d] [CODEX_ARGS...]`
- `agents-infra claude [--print-config] [-d] [CLAUDE_ARGS...]`
- `agents-infra version`

Setup syncs the repo into `.agents`, treats `.skills/` as the authoritative
source-managed skill tree, refreshes the managed links it owns inside `skills/`,
and then refreshes symlinks in `.claude/`, `.codex/`, and `.local/bin`.

Author shared changes in this source repo. Do **not** edit `~/.agents/`
directly.
The installed `~/.agents/` copy is runtime state and should not keep git metadata.

For project-local installs, use `agents-infra setup local /abs/path/to/project`.
That creates a local runtime layout under the project root:
- `.agents/`: the installed runtime copy; put the actual contents here
- `.claude/`: thin Claude shim that points into `.agents`
- `.codex/`: thin Codex shim that points into `.agents`
- `.local/bin/`: helper CLIs for the local setup, including `agents-infra`

Project-local setup intentionally does not create `.codex/config.toml`. Codex
model, reasoning effort, service tier, trusted projects, and TUI notices are
owned by the global `~/.codex/config.toml` link by default. This prevents stale
project-local configs from silently overriding the current global model.

During local setup, agents-infra removes only the legacy project-local
`.codex/config.toml` symlink it used to create. A custom project-local config is
left in place because project-specific model/reasoning overrides must be
explicit and intentional, not silently destroyed.

Use `--codex-config` when local setup should make an explicit decision:

- `--codex-config=preserve` keeps custom project-local config files and removes
  only the old managed symlink. This is the default.
- `--codex-config=global` removes `.codex/config.toml`, making the global
  `~/.codex/config.toml` authoritative for the project.
- `--codex-config=local` links `.codex/config.toml` to the installed project
  runtime at `.agents/.configs/codex-config.toml`, making model/reasoning
  settings project-local by explicit choice.

### Provider-specific primary session policies

Projects may set primary Codex and Claude sessions independently in
`.agents/.configs/project-config.toml`. This is optional policy for the
matching `agents-infra` launcher, not a replacement for either provider's own
configuration. A missing table—or a missing Codex individual field—leaves that
dimension to the provider-native project/profile/user/system/default resolution.

Create the local runtime, then either edit the TOML manually or use the setup
flags below:

```bash
PROJECT=/abs/path/to/project
agents-infra setup local "$PROJECT"
```

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

`[agents.codex.primary_session]` must contain at least one of these optional
fields. `model` and `reasoning_effort` are non-empty TOML strings after
trimming; `yolo_mode` is a real TOML boolean, not a quoted string. Codex—not
agents-infra—remains responsible for whether a model is available and whether a
model/effort pair is compatible.

`[agents.claude.primary_session]` accepts optional non-empty `model` and
`yolo_mode` fields. `yolo_mode` is an unquoted TOML boolean. Claude reasoning
remains provider-native. Codex fields never configure `agents-infra claude`,
and Claude fields never configure `agents-infra codex`; `[mcp]` remains the one
intentional provider-shared project section.

When launched from a project directory, `agents-infra codex` walks from the
filesystem root to the current directory and reads each
`.agents/.configs/project-config.toml` it finds. It parses every discovered
file once, retains absolute paths as provenance, and fails before launch when
any discovered file is invalid. `~/.agents/.configs/project-config.toml` is
never a project-policy source. Each primary-session field composes
independently: the nearest file that explicitly supplies that field wins. In
particular, a child `yolo_mode = false` masks a parent's `true`; omission means
inheritance.

Claude model and yolo provenance compose independently with the same
root-to-leaf rule: the nearest explicitly configured field wins, and
`yolo_mode = false` masks an inherited `true`. Claude never inherits Codex
model, effort, or yolo values and does not affect Codex resolution.

For model and reasoning effort, the per-field launch precedence is:

| Priority | Source | Notes |
| --- | --- | --- |
| 1 | Explicit Codex selection | `--model`/`-m`, top-level `-c model=...`, or top-level `-c model_reasoning_effort=...` passed through this launcher. |
| 2 | Effective project primary-session field | Emitted as a Codex CLI/config override only when selected. |
| 3 | Codex native resolution | Project/profile/user/system/default configuration. |

An explicit `--profile` or `-p` suppresses project model and project reasoning
so the profile can resolve them; an explicit model or effort passed alongside a
profile still wins for its own field. Equal duplicate explicit values collapse
to one override; conflicting explicit values fail before Codex is executed.
Model, reasoning, and profile wrapper arguments participate only before `--`.

Yolo is an independent safety decision per provider. For either launcher,
explicit `-d`, `--danger`, `--yolo`, or its native dangerous flag wins over
project policy; otherwise effective project `yolo_mode = true` enables it, and
an explicit project `false` or an absent value emits no dangerous flag. Codex
uses `--dangerously-bypass-approvals-and-sandbox`; Claude uses
`--dangerously-skip-permissions`. Each launch emits that provider's native
dangerous flag at most once, and `--print-config` records the effective source
and whether project policy was suppressed by explicit CLI input. Persistent
yolo applies only to its matching `agents-infra codex` or `agents-infra claude`
primary launch; it never affects `task-board spawn`, task-board manifests, or
child-run selection.

For Claude primary-session policy, `agents-infra claude` applies an effective
project model through native `--model MODEL` and an effective yolo value through
`--dangerously-skip-permissions`. An explicit Claude `--model` or
`--model=MODEL` before `--` wins and suppresses only the project model for that
launch; explicit Claude danger input suppresses only the project yolo policy.
The Claude wrapper does not infer model or yolo values from Codex policy,
profiles, or task-board settings.

Use the supported local setup surface to update a project without replacing
MCP settings, unrelated TOML tables, comments, or unspecified primary fields:

```bash
agents-infra setup local /abs/path/to/project \
  --codex-primary-model gpt-5.6-terra \
  --codex-primary-reasoning-effort xhigh \
  --codex-yolo-mode=false \
  --claude-primary-model claude-opus-4-6 \
  --claude-yolo-mode=false

agents-infra setup local /abs/path/to/project \
  --clear-codex-primary-session

agents-infra setup local /abs/path/to/project \
  --claude-primary-model claude-opus-4-6

agents-infra setup local /abs/path/to/project \
  --clear-claude-primary-session
```

No provider primary-session flags leave `project-config.toml` byte-identical.
Set flags update only the supplied provider fields, so
`--codex-yolo-mode=false` is an explicit value and
`--claude-primary-model` never edits the Codex, MCP, comments, or unrelated
TOML content. Each clear removes only its own table. Clear cannot be combined
with a set flag for the same provider, all primary-session flags are local-only,
and a target that resolves to `~/.agents/.configs/project-config.toml` is
rejected. Validation or atomic-write failures preserve the original file.
Supported Unix targets use an atomic rename; Windows uses
[`atomic.ReplaceFile`](https://pkg.go.dev/github.com/natefinch/atomic#ReplaceFile)
(`MoveFileExW` replace/write-through); unsupported replacement targets fail
closed.

Inspect either provider invocation without launching an agent:

```bash
cd /abs/path/to/project
agents-infra codex --print-config
agents-infra codex --print-config --profile fast
agents-infra codex --print-config --model gpt-5.6-terra -c 'model_reasoning_effort="xhigh"'

agents-infra claude --print-config
agents-infra claude --print-config --model claude-opus-4-6

# Start an actual primary Codex session from the same project.
agents-infra codex
```

`--print-config` prints every discovered project-config path; effective and
project values with their sources; explicit-CLI/profile suppression state where
applicable; wrapper yolo expansion; and the exact provider args to be executed.
It is the first diagnostic when a launch does not use the expected values.

`doctor local` uses the same resolver for persistent configuration evidence:

```bash
agents-infra doctor local /abs/path/to/project
```

For a configured project, its stable primary-session fields include:

```text
codex_primary_config_valid: true
codex_primary_model: gpt-5.6-terra
codex_primary_model_source: /abs/path/to/project/.agents/.configs/project-config.toml
codex_primary_reasoning_effort: xhigh
codex_primary_reasoning_effort_source: /abs/path/to/project/.agents/.configs/project-config.toml
codex_primary_yolo_mode: false
codex_primary_yolo_mode_source: /abs/path/to/project/.agents/.configs/project-config.toml
claude_primary_config_valid: true
claude_primary_model: claude-opus-4-6
claude_primary_model_source: /abs/path/to/project/.agents/.configs/project-config.toml
claude_primary_yolo_mode: false
claude_primary_yolo_mode_source: /abs/path/to/project/.agents/.configs/project-config.toml
```

When a provider model or Codex effort is absent, doctor renders an empty value
with source `native`; absent Codex or Claude yolo renders `false` with source
`default`. Invalid ancestor TOML makes doctor nonzero, reports the exact path
and field, and sets both provider validation flags false without printing
partial provider policy.

Existing `.codex/config.toml` behavior is unchanged: no automatic migration
or deletion occurs, and local config remains an intentional Codex-native
project layer. `codex_config_shadowing_global: true` means it overrides the
global `~/.codex/config.toml`; use `--codex-config=global` to remove an
unwanted local config, or `--codex-config=local` to install the managed local
one. Project primary-session overrides and `.codex/config.toml` can coexist;
use an explicit `--profile` when the profile should control model and effort.

Troubleshooting:

- Unexpected model or effort: run `agents-infra codex --print-config`; check
  `effective_source`, `project_application`, an explicit `-c`/`--model`, and
  profile suppression.
- Unexpected danger flag: inspect that launcher's `wrapper_expansions` and
  `yolo_mode`; set the nearest provider field explicitly to `false` to mask an
  inherited `true`.
- Unexpected Claude model: run `agents-infra claude --print-config`; check its
  `effective_source`, `project_application`, and any explicit `--model`.
- Invalid configuration: use unquoted strings for model/effort and an unquoted
  boolean for yolo; the launcher reports the source path and field.
- Global model appears ignored: run `agents-infra doctor local PROJECT` and
  resolve any `codex_config_shadowing_global: true` state deliberately.

Task-board child spawn ceilings belong to the separate
[task-board spawn-ceiling contract](https://github.com/relux-works/skill-project-management/blob/main/.specs/project-agent-selection-policy.md#task-board-spawn-ceiling-contract).
Agents-infra neither reads nor validates that `task-board.config.json` policy;
it owns only this primary-session TOML.

## Tooling

| Tool | Purpose | Command | Outputs |
|------|---------|---------|---------|
| `./setup.sh` / `./setup.ps1` | Bootstrap the `agents-infra` CLI and sync the global runtime | `./setup.sh`, `.\setup.ps1` | `~/.local/bin/agents-infra`, `~/.agents/`, `~/.claude/`, `~/.codex/`, install-state metadata |
| `agents-infra` | Set up or inspect global/project-local agent runtimes; configure and launch isolated primary Codex and Claude sessions; launch either agent with project-local MCP opt-ins | `agents-infra setup global`, `agents-infra setup local /path/to/project --codex-primary-model MODEL --codex-primary-reasoning-effort EFFORT --codex-yolo-mode=true\|false --claude-primary-model MODEL`, `agents-infra setup local /path/to/project --clear-codex-primary-session`, `agents-infra setup local /path/to/project --clear-claude-primary-session`, `agents-infra doctor local /path/to/project`, `agents-infra codex --print-config`, `agents-infra claude --print-config` | Runtime directories under the target root; printed diagnostics on stdout |
| `agents-attachments` | Resolve generic attachment manifests and stage image inputs for inspection | `agents-attachments list`, `agents-attachments path screenshot.png`, `agents-attachments stage-images ./photo.heic --out-dir .temp/image-intake` | `.temp/agents-attachments-manifest.json`, `.temp/agents-attachments/`, staged images and `image-stage-map.json` under caller-selected `.temp/` |
| `python3` | Run the `agents-attachments` helper and its focused tests | `python3 -m py_compile .scripts/agents-attachments`, `python3 -m unittest tests/test_agents_attachments.py` | Python bytecode cache and task-scoped logs under `.temp/` |
| `sips` / ImageMagick `magick` | Normalize HEIC/HEIF image inputs for staged inspection | `sips -s format png input.heic --out output.png`, `magick input.heic output.png` | Normalized staged images under caller-selected `.temp/` |
| `go` | Build, test, and vet the Go CLI in `tools/agents-infra` | `cd tools/agents-infra && go test ./...`, `cd tools/agents-infra && go vet ./...` | Go test cache; task-scoped logs should be written under `.temp/` |
| `task-board` | Track project work, checklist state, and outcome resources | `task-board q --format compact 'get(TASK-ID) { full }'`, `task-board m 'set_status(TASK-ID, status=development)'` | `.task-board/` and `.task-board/.resources/` |
| `git` | Inspect repo state and validate diff hygiene | `git status --short`, `git diff --check` | No repo artifact; task-scoped command logs should be written under `.temp/` |
| `ssh` / `scp` / `tar` | Validate and document host-agnostic remote agent worker handoff patterns | `ssh "$REMOTE_SSH" 'hostname'`, `scp prompt.md "$REMOTE_SSH:/tmp/run/prompt.md"`, `tar -czf source.tgz .` | Remote task copies and local scratch artifacts under `.temp/remote-agent/` |

## Structure

```
~/.agents/
├── .instructions/          # Global instructions (modular .md files)
│   ├── INSTRUCTIONS.md     # Entry point (loads all modules)
│   ├── AGENTS.md           # Entry point for Codex CLI
│   ├── INSTRUCTIONS_ATTACHMENTS.md
│   ├── INSTRUCTIONS_BROWSER_AUTOMATION.md
│   ├── INSTRUCTIONS_REMOTE_AGENTS.md
│   ├── INSTRUCTIONS_PLATFORM.md
│   ├── INSTRUCTIONS_STRUCTURE.md
│   ├── INSTRUCTIONS_TOOLS.md
│   ├── INSTRUCTIONS_SKILLS.md
│   ├── INSTRUCTIONS_DIAGRAMS.md
│   ├── INSTRUCTIONS_TESTING.md
│   ├── INSTRUCTIONS_WORKFLOW.md
│   ├── INSTRUCTIONS_DOCS.md
│   └── INSTRUCTIONS_STYLE.md
│
├── .skills/                # Source-managed shared skills versioned in this repo
│   ├── algorithmic-art/
│   ├── architecture-diagrams/
│   ├── brand-guidelines/
│   ├── canvas-design/
│   ├── doc-coauthoring/
│   ├── docx/
│   ├── frontend-design/
│   ├── internal-comms/
│   ├── ios-ui-validation/
│   ├── mcp-builder/
│   ├── pdf/
│   ├── pptx/
│   ├── skill-creator/
│   ├── slack-gif-creator/
│   ├── theme-factory/
│   ├── web-artifacts-builder/
│   ├── web-search/
│   ├── webapp-testing/
│   └── xlsx/
│
├── skills/                 # External skills/tooling area in installed runtime; not versioned by this repo
│
├── scripts/                # Cross-platform bootstrap entrypoints
│   ├── setup.sh
│   └── setup.ps1
│
├── .scripts/               # Setup and utility scripts
│   ├── setup-symlinks.sh   # Internal compatibility wrapper over agents-infra
│   └── agents-attachments  # Manifest resolver plus image staging helper
│
├── .configs/               # Tool configurations
│   ├── claude-settings.json    # Claude Code settings (reference)
│   ├── codex-config.toml       # Codex CLI config
│   └── codex-mcp-servers.toml  # Known Codex MCP server definitions
│
├── tools/
│   └── agents-infra/       # Go CLI source
│
└── .rules/                 # Codex CLI rules
    └── default.rules       # Pre-approved commands
```

## Instructions

Modular instruction files in `.instructions/`:

| File | Purpose |
|------|---------|
| `INSTRUCTIONS.md` | Entry point for Claude Code |
| `AGENTS.md` | Entry point for Codex CLI |
| `INSTRUCTIONS_PLATFORM.md` | Target platform preferences (iOS > macOS) |
| `INSTRUCTIONS_STRUCTURE.md` | Project structure conventions |
| `INSTRUCTIONS_ATTACHMENTS.md` | Generic attachment manifest, image staging, inspection, OCR fallback, and redaction workflow |
| `INSTRUCTIONS_BROWSER_AUTOMATION.md` | No-focus browser scripting and authenticated browser-session rules |
| `INSTRUCTIONS_REMOTE_AGENTS.md` | Host-agnostic workflow for using remote Claude/agent workers through isolated project copies and patch handoff |
| `INSTRUCTIONS_TOOLS.md` | Allowed CLI tools |
| `INSTRUCTIONS_SKILLS.md` | Skills system usage |
| `INSTRUCTIONS_DIAGRAMS.md` | C4/PlantUML diagram rules |
| `INSTRUCTIONS_TESTING.md` | Swift Testing, refactoring workflow |
| `INSTRUCTIONS_WORKFLOW.md` | Task tracking, model fallback, autonomous completion, forced-fit escalation, Git, and logging |
| `INSTRUCTIONS_DOCS.md` | Documentation requirements |
| `INSTRUCTIONS_STYLE.md` | Communication style |

## Skills

Each skill follows the structure:

```
skill-name/
├── SKILL.md              # Required: frontmatter + instructions
├── scripts/              # Optional: executable code
├── references/           # Optional: docs/schemas
└── assets/               # Optional: templates/resources
```

### Available Skills

| Skill | Description |
|-------|-------------|
| `ios-ui-validation` | UI testing with screenshot validation, Page Object pattern |
| `skill-creator` | Scaffold new skills |
| `architecture-diagrams` | C4/PlantUML diagrams |
| `frontend-design` | Production-grade frontend interfaces |
| `pdf` | Markdown/HTML to PDF rendering with shared themes |
| `webapp-testing` | Playwright-based web testing |
| `mcp-builder` | Build MCP servers |
| `web-search` | Web search integration |
| `canvas-design` | Visual art in PNG/PDF |
| `algorithmic-art` | p5.js generative art |
| `theme-factory` | Artifact styling toolkit |
| `brand-guidelines` | Anthropic brand colors/typography |
| `internal-comms` | Internal communications templates |
| `slack-gif-creator` | Animated GIFs for Slack |
| `doc-coauthoring` | Documentation co-authoring workflow |
| `web-artifacts-builder` | Multi-component HTML artifacts |

## Optional PDF Toolchain

Install the PDF renderer stack with:

```bash
./setup.sh --with-pdf-tools
```

Or without rerunning the whole bootstrap:

```bash
./.scripts/setup-pdf-tools.sh
./.scripts/setup-pdf-tools.sh --check
```

Managed dependencies:

- `pandoc`
- `weasyprint`
- `poppler` (`pdftotext`, `pdfinfo`)

The shared PDF skill lives at `.skills/pdf/` and includes:

- `scripts/render-pdf.sh`
- `assets/template.html5`
- `assets/themes/prose-classic.css`
- `assets/themes/report-clean.css`

Example:

```bash
./.skills/pdf/scripts/render-pdf.sh notes/report.md \
  -o .temp/report.pdf \
  --theme prose-classic \
  --title "Research Report"
```

Quick preflight and discovery:

```bash
./.scripts/setup-pdf-tools.sh --check
./.skills/pdf/scripts/render-pdf.sh --list-themes
```

## Configs

### Claude Code (`claude-settings.json`)

Reference config with:
- Allowed tools (Bash, Read, Edit, Write, etc.)
- Default model: `sonnet` (currently Sonnet 4.6)
- Enabled plugins: `swift-lsp`

### Codex CLI (`codex-config.toml`)

- Model: `gpt-5.5`
- Reasoning effort: `xhigh`
- Project docs byte limit: `131072`
- The approaching-rate-limit model switch reminder is suppressed with `[notice].hide_rate_limit_model_nudge = true` so Codex does not ask to move to a lower-credit model.
- As of the audited Codex CLI `0.144.1`, the separate safety-buffering chooser (`Retry with a faster model` / `Keep waiting`) has no supported `config.toml` setting for suppression, a default choice, or automatic waiting. It is runtime UI shown before agent instructions can act; terminal key automation or a patched Codex binary is intentionally out of scope.
- Global workflow instructions treat temporary model unavailability as an operational condition: retry the preferred model at least three times, then choose the best viable fallback autonomously and escalate only a real blocker.
- Trusted projects list
- Global setup owns `~/.codex/config.toml`; project-local setup deliberately does not create `.codex/config.toml` so the global model/settings remain authoritative.
- Local setup removes legacy managed project-local config symlinks but preserves custom `.codex/config.toml` files.
- Explicit project-local config is available with `agents-infra setup local /path/to/project --codex-config=local`.
- Enforce global config with `agents-infra setup local /path/to/project --codex-config=global`.
- `agents-infra doctor local` reports `codex_config_shadowing_global: true` when a project-local `.codex/config.toml` is overriding the global config.

### Project-Local MCP Opt-In (Codex + Claude Code)

Agents-infra does not enable MCP servers in the global Codex or Claude Code
config by default. Projects opt in explicitly through a single,
agent-agnostic list in `.agents/.configs/project-config.toml`:

```toml
[mcp]
enabled_servers = ["figma"]
```

There is one list per project, not one per agent — `enabled_servers` decides
which servers are available regardless of whether you launch Codex or Claude
Code. Known MCP server definitions live in `.configs/codex-mcp-servers.toml`
and are synced into project runtimes. Definitions can describe streamable
HTTP servers with `url` or stdio servers with `command` and optional `args`.

Start Codex through `agents-infra codex` from inside the project tree. The
launcher walks upward from the current directory, composes every discovered
`.agents/.configs/project-config.toml`, resolves enabled MCP definitions from
project registries plus the global registry, logs where each part came from,
then starts Codex with the resulting `-c` overrides:

```bash
agents-infra codex
agents-infra codex -d -
agents-infra codex exec "check the Figma node"
agents-infra codex --print-config
```

Start Claude Code the same way through `agents-infra claude` — same
`enabled_servers` list, same registries, same ancestor walk — but rendered as
a single Claude Code `--mcp-config` JSON payload instead of Codex `-c`
overrides (streamable HTTP servers become `{"type":"http","url":...}`, with
`bearer_token_env_var` mapped to an `Authorization: Bearer ${VAR}` header for
Claude Code to expand at launch; stdio servers become
`{"type":"stdio","command":...,"args":[...]}`). That payload is added on top
of whatever MCP servers are already configured at the user/project level —
the launcher does not pass `--strict-mcp-config`, so existing `.mcp.json` /
`claude mcp add` servers keep working unchanged:

```bash
agents-infra claude
agents-infra claude -d
agents-infra claude --print-config
```

`-d` expands to Codex `--dangerously-bypass-approvals-and-sandbox` or Claude
Code `--dangerously-skip-permissions` respectively. Each launcher can also
apply its own persistent `[agents.<provider>.primary_session].yolo_mode`
policy. If no project opt-in is found while walking upward, neither launcher
mounts anything — no `-c` overrides for Codex, no `--mcp-config` flag for
Claude Code.

LLDB MCP is available as an opt-in stdio server:

```toml
[mcp]
enabled_servers = ["lldb"]
```

LLDB's MCP integration uses `lldb-mcp`, which bridges stdio to the LLDB MCP
server socket. On macOS, `./setup.sh` installs Homebrew `llvm` when `lldb-mcp`
is missing and writes a narrow `$(brew --prefix)/bin/lldb-mcp` wrapper that
execs Homebrew's helper without overriding `LLDB_EXE_PATH`. This lets
`lldb-mcp` use the `lldb` binary next to itself by default, matching LLDB's
documented behavior. The wrapper also prunes dead-PID
`~/.lldb/lldb-mcp-*.json` discovery files before launch so stale sockets do not
break the MCP initialize handshake. Set `AGENTS_INFRA_SKIP_LLDB_MCP=1` to skip
that bootstrap. If a project uses an LLDB build with the helper elsewhere,
override the definition in the project-local
`.agents/.configs/codex-mcp-servers.toml`:

```toml
[servers.lldb]
command = "/path/to/lldb-mcp"
```

Safari MCP is available as an opt-in stdio server backed by Safari Technology
Preview's `safaridriver`:

```toml
[mcp]
enabled_servers = ["safari"]
```

The shared definition launches:

```toml
[servers.safari]
command = "/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver"
args = ["--mcp"]
```

Prerequisites:

- Install Safari Technology Preview 247 or newer.
- Enable `Safari Settings > Advanced > Show features for web developers`.
- Enable `Safari Settings > Developer > Enable remote automation and external agents`.

Safari remains project-local opt-in only. Do not add it to a global Codex or
Claude Code MCP config unless the user explicitly wants a user-managed global
server.

During `agents-infra setup local`, a non-empty `enabled_servers` list also
installs `.local/bin/codex-local` as a backward-compatible shim that delegates
to `agents-infra codex`. The project-local `agents-infra` helper preserves the
caller's working directory before it runs the source checkout with `go run`, so
`codex-local --print-config` should report the directory where the user invoked
it, not `.agents/tools/agents-infra`.

User-managed global MCP servers in the base Codex config, or in Claude Code's
own user/project scopes, remain that agent's own responsibility, not
agents-infra project opt-in state. The global Codex model/settings config
remains authoritative.

## Attachments

This repo defines a generic agent attachment contract:

- manifest file name: `agents-attachments-manifest.json`
- env var: `AGENTS_ATTACHMENTS_MANIFEST`
- helper CLI: `agents-attachments`

The repo does not itself ingest chat attachments. A separate runtime or launcher
must materialize files locally, write the manifest, and export the env var before
starting the agent process.

For Codex sessions, the helper can bootstrap a local manifest from rollout
history when `CODEX_THREAD_ID` is available:

```bash
agents-attachments materialize
```

For image intake, stage explicit paths or manifest references before inspection:

```bash
agents-attachments stage-images ./photo.heic screenshot.png --out-dir .temp/image-intake
agents-attachments stage-images --all --manifest .temp/agents-attachments-manifest.json --out-dir .temp/image-intake
```

`stage-images` keeps originals read-only, writes normalized/copied images under
the selected scratch directory, and emits `image-stage-map.json` with redacted
source labels, content hashes, staged filenames, and HEIC normalization details.
HEIC/HEIF inputs normalize to PNG with macOS `sips` first, then ImageMagick
(`magick`, then `convert`) as the portable fallback; missing converters fail
clearly.

Agents should inspect staged images directly through runtime vision first. OCR
is only a bounded fallback when direct inspection is insufficient. Observations
must cite the staged filename, evidence, confidence, uncertainty, and redactions;
do not persist raw ICCID, IMSI, QR payloads, activation codes, tokens, keys, or
password-like values extracted from images.

## Rules

`.rules/default.rules`: pre-approved Codex CLI commands:
- PlantUML download and rendering
- Temporary directory creation

## How It Works

After running `agents-infra setup global`:

```
~/.agents/
├── skills/
│   ├── relux-agents-infra -> ~/.agents
│   ├── skill-creator -> ~/.agents/.skills/skill-creator
│   └── ...

~/.claude/
├── CLAUDE.md           # Loads @instructions/INSTRUCTIONS.md
├── instructions/ -> ~/.agents/.instructions/
└── skills/
    ├── relux-agents-infra -> ~/.agents/skills/relux-agents-infra
    ├── skill-creator/ -> ~/.agents/skills/skill-creator
    └── ...

~/.codex/
├── AGENTS.md           # Rendered from ~/.agents/.instructions/AGENTS.md
├── config.toml -> ~/.agents/.configs/codex-config.toml
├── skills/
│   └── ... -> ~/.agents/skills/...
└── rules/
    └── default.rules -> ~/.agents/.rules/default.rules
```

`~/.agents` is the installed runtime copy. It should not be used as a git checkout.

Meaning of the two skill trees:
- `.skills/` is the authoritative skill content that belongs to this repo, lives under its version control, and is synced into the installed runtime.
- `skills/` is the external runtime area for public skills and tooling. It may contain content that does not belong to `relux-agents-infra`. `setup` only refreshes the managed links it owns there and must not treat that directory as repo-owned content.

Project-local install example:

```
project-root/
├── .agents/
│   ├── .instructions/
│   ├── .configs/
│   ├── .scripts/
│   ├── .skills/
│   └── skills/
├── .claude/
│   ├── CLAUDE.md
│   ├── instructions/ -> .agents/.instructions/
│   └── skills/ -> .agents/skills/...
├── .codex/
│   ├── AGENTS.md       # Rendered Codex instructions
│   └── skills/ -> .agents/skills/...
├── AGENTS.md           # Rendered project-root Codex instructions
└── .local/bin/
    ├── agents-attachments -> .agents/.scripts/agents-attachments
    └── agents-infra       # launcher for the Go CLI
```

In local-project mode, treat `.agents/` as the installed source/runtime-common tree. `.claude/` and `.codex/` are agent-specific runtime outputs. Codex does not expand Claude-style `@...` include indexes, so `setup` materializes `.codex/AGENTS.md` and project-root `AGENTS.md` as flattened markdown. If a hand-written project-root `AGENTS.md` exists, `setup local` preserves it as `.agents/.instructions/AGENTS.project.md` before rendering the Codex-visible file.

## Adding New Skills

1. Create skill in `.skills/<skill-name>/`
2. Add `SKILL.md` with frontmatter
3. Run `agents-infra setup global` to propagate
4. `setup` will refresh the managed link in the installed external `skills/` area without replacing unrelated external skills

Use `./setup.sh` only as bootstrap when the `agents-infra` launcher is missing
or needs reinstalling. On Windows, use `.\setup.ps1` for the same bootstrap flow.

Or use the `skill-creator` skill:

```
/skill-creator
```

## Updating Instructions

Edit files in this source repo, then run `agents-infra setup global` to sync them
into `~/.agents` and refresh the installed runtime state.

## Git

This repo is version-controlled. Commit your changes:

```bash
cd /path/to/relux-agents-infra
git add -A
git commit -m "Update skills/instructions"
git push
agents-infra setup global
```

<!-- relux-ecosystem:start -->

## About Relux Works

This project is part of the open-source ecosystem of
[Relux Works](https://relux.works), an AI-native software development studio.
We build fixed-price MVPs, rescue vibe-coded apps, run local AI inference, and
train teams to work with coding agents. Much of the infrastructure behind that
work is open source.

- Full catalog: [relux.works/en/open-source](https://relux.works/en/open-source/)
- Agentic enablement: [agent harnesses & team training](https://relux.works/en/agentic-enablement/)
- Hire us the agent-native way: point your assistant at `https://api.relux.works/mcp`
- Contact: ivan@relux.works

<!-- relux-ecosystem:end -->
