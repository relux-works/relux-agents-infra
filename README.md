# alexis-agents-infra

Source repo for shared AI agent configurations, instructions, skills, and rules.

Works with:
- **Claude Code** (`~/.claude/`)
- **Codex CLI** (`~/.codex/`)

## Quick Start

```bash
# Bootstrap the launcher, then immediately sync the global runtime
cd /path/to/alexis-agents-infra
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
- `agents-infra version`

Setup syncs the repo into `.agents`, treats `.skills/` as the authoritative
source-managed skill tree, refreshes the managed links it owns inside `skills/`,
and then refreshes symlinks in `.claude/`, `.codex/`, and `.local/bin`.

Author shared changes in this source repo. Do **not** edit `~/.agents/`
directly.
The installed `~/.agents/` copy is runtime state and should not keep git metadata.

For project-local installs, use `agents-infra setup local /abs/path/to/project`.
That creates a local runtime layout under the project root:
- `.agents/` — the installed runtime copy; put the actual contents here
- `.claude/` — thin Claude shim that points into `.agents`
- `.codex/` — thin Codex shim that points into `.agents`
- `.local/bin/` — helper CLIs for the local setup, including `agents-infra`

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

If a Codex session starts with the wrong model, run:

```bash
agents-infra doctor local /abs/path/to/project
```

`codex_config_shadowing_global: true` means the project still has a local
`.codex/config.toml` that overrides the global config. Remove it if unintended,
or keep it as an explicit project-local override.

## Structure

```
~/.agents/
├── .instructions/          # Global instructions (modular .md files)
│   ├── INSTRUCTIONS.md     # Entry point (loads all modules)
│   ├── AGENTS.md           # Entry point for Codex CLI
│   ├── INSTRUCTIONS_ATTACHMENTS.md
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
│   └── agents-attachments  # Helper for agents-attachments-manifest.json
│
├── .configs/               # Tool configurations
│   ├── claude-settings.json    # Claude Code settings (reference)
│   └── codex-config.toml       # Codex CLI config
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
| `INSTRUCTIONS_TOOLS.md` | Allowed CLI tools |
| `INSTRUCTIONS_SKILLS.md` | Skills system usage |
| `INSTRUCTIONS_DIAGRAMS.md` | C4/PlantUML diagram rules |
| `INSTRUCTIONS_TESTING.md` | Swift Testing, refactoring workflow |
| `INSTRUCTIONS_WORKFLOW.md` | Git, task tracking, logging |
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
- Trusted projects list
- Global setup owns `~/.codex/config.toml`; project-local setup deliberately does not create `.codex/config.toml` so the global model/settings remain authoritative.
- Local setup removes legacy managed project-local config symlinks but preserves custom `.codex/config.toml` files.
- Explicit project-local config is available with `agents-infra setup local /path/to/project --codex-config=local`.
- Enforce global config with `agents-infra setup local /path/to/project --codex-config=global`.
- `agents-infra doctor local` reports `codex_config_shadowing_global: true` when a project-local `.codex/config.toml` is overriding the global config.

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

## Rules

`.rules/default.rules` — pre-approved Codex CLI commands:
- PlantUML download and rendering
- Temporary directory creation

## How It Works

After running `agents-infra setup global`:

```
~/.agents/
├── skills/
│   ├── alexis-agents-infra -> ~/.agents
│   ├── skill-creator -> ~/.agents/.skills/skill-creator
│   └── ...

~/.claude/
├── CLAUDE.md           # Loads @instructions/INSTRUCTIONS.md
├── instructions/ -> ~/.agents/.instructions/
└── skills/
    ├── alexis-agents-infra -> ~/.agents/skills/alexis-agents-infra
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
- `skills/` is the external runtime area for public skills and tooling. It may contain content that does not belong to `alexis-agents-infra`. `setup` only refreshes the managed links it owns there and must not treat that directory as repo-owned content.

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
cd /path/to/alexis-agents-infra
git add -A
git commit -m "Update skills/instructions"
git push
agents-infra setup global
```
