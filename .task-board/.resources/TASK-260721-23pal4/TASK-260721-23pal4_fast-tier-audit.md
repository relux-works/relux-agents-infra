# Pre-change audit

- Source repo `.configs/codex-config.toml` currently has `service_tier = "fast"`.
- Installed global `~/.codex/config.toml` resolves through `~/.agents/.configs/codex-config.toml` and currently has `service_tier = "default"`.
- `casual-talks/.agents/.configs/codex-config.toml` has `service_tier = "fast"`, but `casual-talks/.codex/config.toml` is absent and `agents-infra doctor local` reports `codex_config_effective: global`.
- The current Codex manual says `service_tier = "fast"` persists Fast mode; `/fast off` persists the Standard selection.
- Installed global config also contains user-managed trusted-project and TUI entries absent from the source template; do not erase those entries during synchronization.
- The source worktree already contains unrelated user changes. Preserve them.
