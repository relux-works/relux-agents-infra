# Project-local instruction scaffold outcome

Implemented and verified:

- `setup local` skips the shared source `.instructions/` tree.
- Missing project-owned `AGENTS.md` and `INSTRUCTIONS.md` entrypoints are scaffolded under `.agents/.instructions/`.
- Existing project instruction files survive repeated local setup unchanged.
- Handwritten project-root `AGENTS.md` preservation and Codex/Claude runtime rendering remain valid.
- `setup global` still installs and renders shared global instruction modules.
- Global workflow guidance now documents dirty-checkout inspection, task-scoped worktree isolation, reviewed patch integration, main-checkout revalidation, and explicit commit authorization.

Validation:

- Focused local/global setup tests passed.
- `go test ./... -count=1` passed.
- `go vet ./...` passed.
- `go build ./...` passed.
- A real temp-project `setup local` plus `verify local` passed and produced only `AGENTS.md` and `INSTRUCTIONS.md` in the fresh local instruction directory.
- Direct probes found no copied global policy text in the fresh local instruction tree or rendered project Codex entrypoints.
- `git diff --check` passed.

Commits:

- `a51cb4b` — project-owned local instruction scaffold.
- `57a6148` — dirty-checkout and worktree workflow guidance.

Pre-existing repository changes were inspected, validated, and committed separately as `837e8df` and `b10d3b7`.
