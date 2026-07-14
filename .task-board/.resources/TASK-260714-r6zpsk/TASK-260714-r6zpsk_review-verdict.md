Review verdict: ACCEPTED.

Reviewed implementation:
- tools/agents-infra/internal/infra/infra.go
- tools/agents-infra/internal/infra/infra_test.go

Architecture and AC:
- Local sync preserves existing native Codex and Claude policy files, preserves project-config.toml/MCP opt-in, and skips source-local runtime directories instead of nesting them into the target.
- Canonical relux-agents-infra skill self-link is installed idempotently before stale-link cleanup. Stale aliases do not fan out to Claude/Codex, and permission-denied stale cleanup is tolerated without losing the canonical link.
- No alexis-agents-infra references remain in tracked source/docs/tests outside task-board records. The old filesystem path is absent; a non-git relux-agents-infra-legacy-scratch directory remains as safe retained scratch.

Independent validation:
- go test -count=1 ./... passed from tools/agents-infra.
- Focused new tests passed: stale permission cleanup, source runtime skip, and native config resync preservation.
- Isolated local setup smoke ran twice after seeding custom policy, MCP opt-in, native configs, task-board config, project artifact, and a stale legacy self-link. Preserved artifacts were byte-identical; stale link removed; canonical .agents/.claude/.codex links intact.
- gofmt -d and git diff --check were clean.

Review artifacts and command logs are under .temp/TASK-260714-r6zpsk/.