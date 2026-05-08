## Status
done

## Assigned To
(none)

## Created
2026-05-08T13:08:17Z

## Last Update
2026-05-08T13:18:21Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Decision: implementation belongs in tools/agents-infra CLI, specifically setup global/local plus refresh-links. .agents is source/common install tree, not the runtime contract for Codex. Current Codex 0.128.0 treats @~/.agents/... lines literally in prompt-input, so Codex must receive flattened/materialized AGENTS.md.
Implemented in tools/agents-infra. setup global/local and refresh-links now render Codex AGENTS.md files instead of symlinking to @include indexes. Local setup also renders project-root AGENTS.md for Codex project-doc discovery and preserves the previous hand-written source as .agents/.instructions/AGENTS.project.md. Verified go test ./..., ./setup.sh, agents-infra doctor global/local, and codex debug prompt-input with no raw @*.md include lines.
Follow-up included runtime hygiene: setup now skips .task-board/task-board.config.json from syncRepo and cleanup removes stale board artifacts from installed ~/.agents. Verified ~/.agents has no board artifacts after final ./setup.sh.

## Precondition Resources
(none)

## Outcome Resources
(none)
