# TASK-260721-23pal4: disable-fast-tier-by-default

## Description
Make Standard service tier the source-managed Codex default and synchronize the casual-talks local runtime copy without introducing a project-local Codex config.

## Scope
Change only the source-managed Codex service-tier default, directly related README documentation, and installed runtime copies produced by the supported setup flow. Preserve the existing explicit fast profile, user-managed trust/TUI state, unrelated dirty worktree changes, and the absence of casual-talks/.codex/config.toml.

## Acceptance Criteria
- Source .configs/codex-config.toml no longer defaults to service_tier fast and explicitly records the Standard/default tier.
- The existing profiles.fast table remains available and unchanged for explicit profile selection.
- README documents that Standard is the default and Fast is opt-in.
- The effective global ~/.codex/config.toml resolves to service_tier default after verification.
- /Users/alexis/src/casual-talks/.agents/.configs/codex-config.toml resolves to service_tier default after supported local setup, while /Users/alexis/src/casual-talks/.codex/config.toml remains absent.
- Unrelated source and installed runtime state is preserved, and validation evidence is attached to the task.
