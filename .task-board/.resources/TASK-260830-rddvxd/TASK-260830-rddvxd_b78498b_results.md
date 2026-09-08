# TASK-260830-rddvxd Developer Outcome — b78498b Replay

## Candidate

- Fresh fetch proof immediately before snapshot: `HEAD == main == origin/main == FETCH_HEAD == b78498bf98c05175db10bb341aee621e53de4881`; ahead/behind `0/0`.
- Board workspace `initial_base_oid`, `current_base_oid`, `checkpoint_oid`, and `selected_base_oid` all equal `b78498bf98c05175db10bb341aee621e53de4881`.
- Exact tracked scope: `.instructions/INSTRUCTIONS_WORKFLOW.md`, `LOGBOOK.md`, `README.md`, `tools/agents-infra/internal/infra/infra_test.go`.
- Diff: 4 files, 198 insertions, 5 deletions; `git diff --check` exit 0.
- Reviewable patch: `TASK-260830-rddvxd_candidate.patch`.
- Patch SHA-256: `10271f705b6aa34b3f42ca9ba8d54922fe0f6bf65d7e0d1bdd44f85741e62549`.

## Implementation

- Replayed audited revision-4 External-CI local-mirror semantics beneath current trunk's automatic signed-delivery contract without weakening current-main freshness, signatures, exact-head review, checks, landing, merge queues, or branch protection.
- Authorized mirroring only for verified external hosted-CI non-execution the agent cannot repair; absent, failed, partial, malformed, or inconclusive status reads never authorize fallback.
- Required every affected job to run from an exact clean PR head with matching commands, toolchain, environment/non-secret configuration, services, and target, or a documented equivalent.
- Required SHA/job/cause/platform/tool/environment/service/command/exit evidence. Repository-caused failures remain blocking, and local evidence never becomes a hosted status.
- Added production `Setup` coverage for Claude `CLAUDE.md -> instructions symlink -> INSTRUCTIONS.md -> INSTRUCTIONS_WORKFLOW.md` and the rendered Codex `AGENTS.md` surface.
- Added a narrowed broadened-trigger regression that retains the old interior phrase but admits repairable or merely inconvenient disruption; validation rejects it.

## Validation

All gates ran directly as standalone foreground processes. Expected-red mutants are reported as failures.

| Gate | Exit | Result |
| --- | ---: | --- |
| Restored focused production `Setup` pair, final exact-tree rerun | 0 | `ok`, infra 7.232s |
| Broadened repairable/inconvenient live mutant through production `Setup` | 1, expected red | Missing complete exclusive trigger |
| Claude workflow-include removal through production `Setup` | 1, expected red | Installed Claude index lacks workflow include |
| Codex workflow-include removal through production `Setup` | 1, expected red | Rendered Codex instructions lack workflow source |
| `go test -p 1 ./... -count=1`, final exact-tree rerun | 0 | root 133.278s; attachments 1.219s; infra 241.910s; modelharness 0.837s |
| `go vet ./...`, final exact-tree rerun | 0 | No findings |
| `go build ./...`, final exact-tree rerun | 0 | darwin/arm64 build succeeds |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | Current source built, installed, and setup verified |
| `agents-infra verify global` | 0 | Global runtime verified |
| Installed source bytes, Claude chain, and rendered Codex full-source parity | 0 | Complete workflow and exclusive/no-forgery clauses present |
| Fresh fetch/base equality, four-path scope, and `git diff --check` | 0 | Exact current-main candidate |

## Setup Scope Boundary

`AGENTS_INFRA_SKIP_LLDB_MCP=1` is the documented temporary lane for the separately owned Homebrew LLVM 23 `lldb-mcp` bootstrap defect. This evidence proves instruction installation and parity; it does not claim default LLDB MCP compatibility.

## Delivery Boundary

The producer did not stage, commit, integrate, push, forge hosted status, self-review, or bypass protection. Developer handoff publishes the candidate Change Request. Independent review of the exact CR and verification of its eventual base remain reviewer/orchestrator-owned.

## Toolchain

- `agents-infra v1.6.1-96-gb78498b`
- `task-board 0.24.3-172-g063197b1`
- `git 2.53.0`
- `go1.25.5 darwin/arm64`

This resource supersedes stale-base validation summaries for developer evidence on the `b78498b` replay.
