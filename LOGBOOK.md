# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-07-13

### 1804 — Claude Separator Review Correction
- FINDING: The prior separator concern is not a task regression: the detailed contract requires literal native danger input to be consumed while mirroring Codex, and `codex_launch.go:379` uses the same ordering.
- DECISION: Treat `--` as stopping wrapper-shortcut/selection parsing; retain native dangerous-flag de-duplication and provenance parity with Codex.
- STATUS: `TASK-260713-1soh7i` accepted after full Go validation and documentation audit.

### 1803 — Claude Yolo Separator Regression
- REGRESSION: `claude_launch.go:342` recognizes `--dangerously-skip-permissions` before checking `--`; a native argument after the separator suppresses `yolo_mode=false` as explicit CLI input.
- FINDING: `go run . claude --print-config -- --dangerously-skip-permissions` reports `effective_value: true` and `suppressed_by_explicit_cli`; the task requires only pre-separator arguments to participate.
- STATUS: `TASK-260713-1soh7i` routed to rework; add focused coverage for the native flag after `--`.

### 1757 — Claude Persistent Yolo Policy
- DECISION: `[agents.claude.primary_session].yolo_mode` has independent nearest-field precedence; explicit false masks inherited true and explicit Claude danger input suppresses project policy.
- FIX: Claude parse, launch resolution, print-config, setup, doctor, docs, and tests now mirror the Codex yolo contract with `--dangerously-skip-permissions` emitted at most once.
- STATUS: Full Go test, vet, build, 81.0% infra coverage, print-config/doctor smoke, and setup true→false→clear smoke pass.

### 1718 — Provider Session Policy Review Accepted
- MILESTONE: `TASK-260713-1bok5k` verified independent Claude provenance and no Codex model/reasoning/yolo leakage in both target repositories.
- STATUS: Uncached Go tests, vet, build, gofmt, diff-check, print-config, and doctor smokes passed.

### 1713 — Claude Primary Session Isolated from Codex Policy
- DECISION: `[agents.claude.primary_session]` owns only a non-empty Claude model; Codex model, reasoning effort, and yolo remain provider-local.
- FIX: `project_config.go`, `claude_launch.go`, setup, print-config, and doctor compose and report Claude provenance independently.
- STATUS: Full Go test, vet, build, and two-repository Claude print-config/doctor smokes pass with `claude-opus-4-6`.

### 1610 — Primary-Session Operator Contract Documented
- FIX: `README.md` and `SKILL.md` now document the primary-session TOML, independent nearest-field precedence, yolo scope, setup/clear, render, doctor, native fallback, and `.codex/config.toml` coexistence.
- FIX: Corrected the MCP example to `[mcp]`; task-board spawn-ceiling ownership is cross-linked without duplicating its policy.
- STATUS: CLI usage and primary-session Go tests were checked against the active implementation.

### 1608 — MCP Skill Example Used Retired Table Path
- FINDING: `SKILL.md` documented `[codex.mcp]`, while `project_config.go` reads the canonical `[mcp]` table.
- FIX: Primary-session documentation update will correct the MCP example alongside the current launcher contract.

### 1558 — Codex Alias Danger Moved to Project Policy
- FINDING: Active `codexD` is owned only by the user-authored `~/.zshrc:134`; no separate tracked alias definition exists, while `.instructions/INSTRUCTIONS_TOOLS.md:53` owns the shared documentation.
- FIX: Removed the alias-level `-d`; retained `agents-infra codex -d` as the documented explicit ad-hoc full-trust escape hatch.
- STATUS: Fresh zsh smokes in relux-agents-infra and skill-project-management render no wrapper danger expansion and exactly one final native danger argument from each target's `yolo_mode = true` project profile.

### 1537 — Sibling Board Has Pre-Existing Orphan Resources
- FINDING: skill-project-management `task-board validate` reports six tracked orphan resource files/directories dated February–April 2026.
- SCOPE: Target setup did not touch `.task-board`; its config checksum stayed byte-identical and contains no primary-session field.
- STATUS: Left unchanged as unrelated board debt; relux-agents-infra board validation passes.

### 1535 — Target Primary Codex Profiles Active
- MILESTONE: `agents-infra setup local` configured `gpt-5.6-terra`, `xhigh`, and `yolo_mode = true` in relux-agents-infra and skill-project-management.
- STATUS: Both doctor and print-config report target-local provenance, unchanged effective MCP enablement, and exactly one native danger flag in the primary Codex argv.

### 1534 — Concurrent Sibling Writes During Rollout
- ANOMALY: skill-project-management tracked files changed concurrently while target setup ran; `tools/board-cli/cmd/root.go` appeared after the before-state snapshot and board agents remained active on spawn-ceiling work.
- SCOPE: Setup's tracked delta there is the generated marker in `AGENTS.md`; its original bytes remain in `.agents/.instructions/AGENTS.project.md`.
- DECISION: Preservation evidence uses target config, task-board config, MCP registry, and rendered argv checksums instead of an unstable whole-worktree hash; concurrent edits were preserved.

### 1513 — Platform-Backed Project Config Replacement
- DECISION: `project_config_replace_posix.go` uses same-filesystem POSIX rename; `project_config_replace_windows.go` uses `github.com/natefinch/atomic.ReplaceFile` backed by `MoveFileExW(REPLACE_EXISTING|WRITE_THROUGH)`; unsupported targets fail closed.
- FIX: `project_config_setup.go:506` delegates final replacement; focused failure coverage proves original-byte and temporary-file preservation; Windows-only tests cover successful replace and delete-locked failure preservation.
- STATUS: Full/race/vet/coverage/build gates and Windows amd64/arm64 test compilation pass. Local Wine execution was unavailable because the cask requires sudo and the portable archive was Gatekeeper-killed; no platform protections were bypassed.

### 1455 — Windows Atomic Replace Contract Gap
- FINDING: `tools/agents-infra/internal/infra/project_config_setup.go:529` ends the guarded write with `os.Rename`; Go explicitly does not guarantee `Rename` atomicity on non-Unix platforms.
- SCOPE: Windows is supported through `scripts/setup.ps1`; a Windows cross-build proves compilation, not atomic replacement semantics.
- STATUS: `TASK-260713-4ihi4q` review routed to rework for a platform-backed atomic replace and Windows-specific evidence; Unix tests and black-box failure preservation pass.

### 1444 — Global Project-Config Collision Rejected
- REGRESSION: `setup local "$HOME"` accepted primary-session flags and wrote the global runtime config that project discovery intentionally ignores.
- ROOT CAUSE: `project_config_setup.go:56` did not enforce discovery's global-path exclusion, and lexical absolute-path equality missed filesystem aliases.
- FIX: `project_config_setup.go:56` rejects set/clear before sync; `infra.go:1200` compares existing ancestor identity plus unresolved suffix, with Windows case folding.
- STATUS: Exact set/clear, symlink-alias, CLI, full/race/vet/build, and two-repository smoke coverage pass.

### 1423 — Setup Flags After Project Path
- ROOT CAUSE: Go `flag.FlagSet` stops at the first positional argument; documented `setup local PROJECT --flag` calls left trailing flags unparsed and could skip the requested profile mutation.
- FIX: `tools/agents-infra/main.go` extracts the leading local project before flag parsing, rejects extra positionals, and preserves explicit `--codex-yolo-mode=false`; `tools/agents-infra/setup_test.go` covers the path-first form.
- STATUS: Set, partial update, no-flag byte preservation, diagnostics, and clear passed in disposable relux-agents-infra and skill-project-management worktrees.

### 1408 — Placeholder Setup Task Retired
- FIX: Closed `TASK-260713-22gp48` as an accidental placeholder duplicate; `TASK-260713-4ihi4q` remains the sole owner of local primary-session setup flags, atomic TOML merge, preservation, and tests.
- MILESTONE: Regenerated `STORY-260713-3vxko6` plan retains the closed duplicate only for audit and includes target profile rollout `TASK-260713-25bqi7` plus alias cleanup `TASK-260713-1ripj2`.
- STATUS: Active primary-session tasks have complete briefs, dependencies, and checklists; agents-infra Go tests, board validation, and diff checks pass.

### 1340 — agents-infra Owns Project Primary Codex Policy
- DECISION: `[agents.codex.primary_session]` in project `.agents/.configs/project-config.toml` owns optional model, reasoning effort, and `yolo_mode`; nearest ancestor field wins and explicit false masks inherited yolo.
- DECISION: task-board remains a separate consumer of `task-board.config.json -> spawn.ceilings` and does not provide primary-session defaults to agents-infra.
- SCOPE: `STORY-260713-3vxko6` decomposes launch composition, safe local setup, print-config/doctor evidence, docs, and disposable validation of relux-agents-infra plus skill-project-management.
- STATUS: Cross-repository contract and diagrams are linked as task preconditions from `TASK-260713-190sng` in skill-project-management.

## 2026-07-10

### 1155 — Board-Agnostic Image Intake
- DECISION: Image intake belongs in `.scripts/agents-attachments` as `stage-images`, not in board-specific scripts or resources.
- DECISION: Source-to-staged mappings persist redacted source labels plus content hashes; raw ICCID/IMSI/key-like labels are not written into staged filenames.
- SCOPE: `.scripts/agents-attachments`, `.instructions/INSTRUCTIONS_ATTACHMENTS.md`, `SKILL.md`, `README.md`, `tests/test_agents_attachments.py`.
