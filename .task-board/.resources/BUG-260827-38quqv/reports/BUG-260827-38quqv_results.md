# Global version-control instruction cleanup

## Delivered

- Removed global commit and push authorization prose from `.instructions/INSTRUCTIONS_WORKFLOW.md`.
- Removed global commit-message and attribution policy.
- Delegated tracked commit acknowledgement and timing to `task-board.config.json -> version_control`.
- Preserved repository-specific contribution rules, worktree guidance, dirty-checkout isolation, and destructive-command safeguards.
- Removed the stale commit/push cross-reference from `.instructions/INSTRUCTIONS_TOOLS.md`.
- Replaced the stale project-local global-instruction duplicate with minimal Codex and Claude project overlays.
- Regenerated the tracked root `AGENTS.md` as a project-only overlay.
- Synchronized and verified global and local installed runtimes.

## Evidence

- Focused `go test ./internal/infra -count=1`: passed in 131.570 seconds.
- Full `go test ./... -count=1`: passed.
- `agents-infra verify global`: passed.
- `agents-infra verify local /Users/alexis/src/relux-works/relux-agents-infra`: passed.
- Negative search across source, generated project instructions, and installed global instructions found none of the removed phrases.
- Installed global instructions contain the task-board version-control delegation.
- Root `AGENTS.md` contains only the project overlay and no duplicated global modules.

## Boundary

This cleanup removes relux-agents-infra's global policy only. It does not override or delete contribution rules owned by another repository, including mlx-lm's AI-authorship policy.
