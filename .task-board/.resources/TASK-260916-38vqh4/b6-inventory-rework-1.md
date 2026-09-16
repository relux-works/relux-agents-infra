# Rework 1 — TASK-260916-38vqh4 (reviewer verdict TASK-260916-38vqh4_review-verdict-rev1.md)

Apply exactly the two P2 corrections and republish the report (update the resource TASK-260916-38vqh4_report.md; no repository edits):
1. Add the four runtime subcommands dispatched at tools/agents-infra/main.go:854–885 — `runtime quarantine`, `runtime unquarantine`, `broker`, `runtime-launch` — to the command inventory as KEEP (pi local-model runtime residual), with file:line.
2. Fix the installed local Codex shim path: LocalLayout sets BinDir to `<project>/.local/bin` (cite internal/infra/infra.go line), so the shim is `<project>/.local/bin/codex`, not `.codex/bin/codex`; correct the setup table and the deprecation section accordingly.
Then tick the checklist and `task-board handoff TASK-260916-38vqh4 --role developer`. Do NOT run `go test ./...` (the full suite stalls this host); no tests are needed for a report correction.
