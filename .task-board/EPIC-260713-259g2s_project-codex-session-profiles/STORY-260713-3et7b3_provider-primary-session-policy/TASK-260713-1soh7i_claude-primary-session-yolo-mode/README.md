# TASK-260713-1soh7i: claude-primary-session-yolo-mode

## Description
Bring agents-infra claude to parity with agents-infra codex for persistent yolo mode so the claudeD shell alias can become plain "agents-infra claude" (config-driven), exactly like codexD is plain "agents-infra codex".

Current state: [agents.claude.primary_session] in .agents/.configs/project-config.toml supports only model. The Claude launcher (tools/agents-infra/internal/infra/claude_launch.go) expands -d/--danger/--yolo wrapper shortcuts to --dangerously-skip-permissions but has no persistent yolo_mode policy. Codex (codex_launch.go + project_config.go) already has the full contract: yolo_mode boolean in [agents.codex.primary_session], root-to-leaf composition where nearest explicit field wins and explicit false masks inherited true, CLI danger flags suppress project policy, exactly one dangerous flag emitted, resolution provenance rendered in --print-config.

Scope — mirror the Codex yolo contract for Claude:
1. project_config.go: add yolo_mode (*bool) to ClaudePrimarySessionSource, ClaudePrimarySessionPolicy (bool value type with Value/Source/Present), parseClaudePrimarySession (accept yolo_mode as supported field, keep unsupported-field rejection for anything else), composeClaudePrimarySession, cloneClaudePrimarySessionSource.
2. claude_launch.go: parseClaudeWrapperArgs must track dangerRequested/dangerSource for -d/--danger/--yolo shortcuts AND a literal --dangerously-skip-permissions user arg (consume them instead of passing through, mirroring codex parse behavior). Add YoloMode resolution to ClaudePrimarySessionResolution (bool resolution struct mirroring CodexPrimarySessionBoolResolution with not_configured/applied/suppressed_by_explicit_cli application states). resolveClaudePrimarySession emits exactly one --dangerously-skip-permissions in ConfigArgs when effective yolo is true. Default stays safe (false).
3. RenderClaudeLaunchPlan --print-config output: render yolo_mode block (effective_value/effective_source/project_value/project_source/project_application) like the codex renderer.
4. Setup flags in main.go + project_config_setup.go: add --claude-yolo-mode=true|false for setup local, mirroring --codex-yolo-mode semantics (local-only, preserves explicit false, atomic write, conflicts with --clear-claude-primary-session which must clear the whole table including yolo_mode). Update setup usage/help text.
5. Doctor: report claude yolo value and source analogous to codex doctor output (absent -> false from default).
6. Tests: extend claude_launch_test.go, project_config_test.go, project_config_setup_test.go mirroring the codex yolo coverage: config parse (true/false/invalid type), composition/masking across ancestor configs, CLI suppression, exactly-one-flag emission, pass-through after --, render output, setup flag set/clear, doctor.
7. Docs: update README.md provider session policy section and .skills source skill docs (relux-agents-infra skill SKILL.md if it documents the codex-only yolo limitation, e.g. the sentence saying the persistent yolo setting never propagates to Claude must be revised to describe the independent Claude policy) plus any INSTRUCTIONS_TOOLS.md wording about claudeD/codexD launchers if present in this repo source.

Out of scope: task-board spawn policy, run manifests, spawn ceilings, changes to codex behavior, editing ~/.zshrc (orchestrator handles the personal alias after merge).

Semantics must match codex exactly: precedence explicit CLI danger flag > project yolo_mode > safe default false; nearest config wins; yolo_mode=false explicitly masks inherited true; only args before -- participate; malformed config fails before launch.

## Scope
(define task scope)

## Acceptance Criteria

1. [agents.claude.primary_session] accepts yolo_mode boolean; non-boolean values fail with a clear field error before launch.
2. agents-infra claude with project yolo_mode=true launches with exactly one --dangerously-skip-permissions; yolo_mode=false or absent adds none.
3. Nearest-wins composition: child project yolo_mode=false masks ancestor true.
4. -d/--danger/--yolo or literal --dangerously-skip-permissions on the CLI results in exactly one flag and marks project policy suppressed_by_explicit_cli in --print-config.
5. --print-config renders yolo_mode effective/project values with sources without launching.
6. setup local --claude-yolo-mode=true|false updates only that field preserving other TOML bytes; --clear-claude-primary-session removes the table and conflicts with set flags; doctor reports claude yolo value+source.
7. go test ./... passes in tools/agents-infra with new coverage for all of the above.
8. README and skill docs describe the Claude yolo policy; no doc still claims persistent yolo is codex-only.
