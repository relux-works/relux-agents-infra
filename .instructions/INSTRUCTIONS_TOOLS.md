# Allowed Tools

Agent may use these tools freely.

## Build / Apple tooling

* `xcodebuild` (CLI builds/tests)
* `swift` (toolchain / SwiftPM)
* `simctl` (simulator management)

---

## Scripting / Automation

### git

* Inspect diffs, config, repo state.
* Agent may check git config to detect default branch and repo location.
* **Never commit or stage files automatically.**

### python / perl / make

* `python` — scripting
* `perl` — quick regex/text processing
* `make` — automation
  * Agent may create a Makefile for its own automation.
  * If a Makefile already exists, extend it (do not replace).

---

## Project-Local MCP Servers

Agents-infra managed MCP servers are project-local opt-in only. Do not enable
them globally by default.

Supported shared MCP server definitions:

| Server | Use | Enablement |
|--------|-----|------------|
| `figma` | Figma Dev Mode / design context | Add `figma` to `.agents/.configs/project-config.toml` |
| `lldb` | LLDB debugging through `lldb-mcp` stdio bridge | Add `lldb` to `.agents/.configs/project-config.toml`; macOS `./setup.sh` installs Homebrew `llvm` and an `lldb-mcp` wrapper when needed |
| `safari` | Safari Technology Preview web inspection through `safaridriver --mcp` | Add `safari` to `.agents/.configs/project-config.toml`; requires Safari Technology Preview with web developer features, remote automation, and external agents enabled |

Project opt-in example:

```toml
[codex.mcp]
enabled_servers = ["figma", "lldb", "safari"]
```

Start Codex through `agents-infra codex` from inside the project tree so the
project-local MCP registry and enablement are composed into Codex `-c`
overrides.

---

## Installing additional CLI tools

* If the agent identifies a **strong, justified need** for additional CLI tools, it may:
  * ask to install them, **or**
  * attempt to install them directly **if permissions allow**.
* Any installs must be:
  * explicitly documented (what/why/how),
  * minimal (avoid tool bloat),
  * reproducible (include exact commands),
  * and safe (no hidden side effects).

---

## Tool Readiness Check (mandatory)

* Before using any tool for real project work, the agent must verify it can run and produce expected output.
* Verification outputs / scratch artifacts must be stored in `.temp/` (since it is not versioned).
* If a tool cannot be used successfully:
  * document the failure (what command failed + error) in `.temp/` logs and/or `.temp/tasks.md`;
  * do not proceed with workflows that assume the tool works.
