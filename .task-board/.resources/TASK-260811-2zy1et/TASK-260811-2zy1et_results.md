# Outcome

- Corrected `README.md` and `SKILL.md` so primary preparation consistently documents exact preservation of absent, managed, custom, and linked `.codex/config.toml` state.
- Confirmed the stale rewrite claims are absent and the preservation wording is present.
- `git diff --check`: passed.
- `go test ./...` in `tools/agents-infra`: passed.
- Installed from source with `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh`.
- `agents-infra verify global`: passed.
- Installed `~/.agents/SKILL.md` is byte-identical to the source skill and contains the corrected contract.
- Source documentation commit: `0f1d889` (`docs(codex): clarify primary config preservation`).

Logs are under `.temp/TASK-260811-2zy1et/`.
