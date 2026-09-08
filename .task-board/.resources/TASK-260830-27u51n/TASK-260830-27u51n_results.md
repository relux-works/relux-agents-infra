# TASK-260830-27u51n developer results

## Outcome

- Added the narrow external-CI local mirror fallback to `.instructions/INSTRUCTIONS_WORKFLOW.md`.
- Fallback requires a verified external hosted-CI execution cause the agent cannot repair.
- Every affected job must run on an exact clean PR-head checkout with matching commands, toolchain, environment variables, services, and target, or a documented equivalent.
- Evidence must record PR-head SHA, hosted job, external cause, platform/target, tool versions, environment assumptions, services, exact commands, and every exit code.
- Repository-caused failures, hosted-status integrity, remote review, merge queues, and branch protection remain blocking and authoritative.
- Added a production `Setup` parity test for the versioned source, installed Claude module, Claude-linked module, and rendered Codex instructions.
- Updated README documentation and recorded the trust-boundary decision in `LOGBOOK.md`.

## Base and scope

- Story branch: `task-board/story/STORY-260830-l1sp3d`
- Evidence commit: `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`
- Freshly fetched `origin/main`, local `main`, and `HEAD` all resolved to the same SHA; ahead/behind was `0/0` before implementation.
- Versioned changes: `.instructions/INSTRUCTIONS_WORKFLOW.md`, `tools/agents-infra/internal/infra/infra_test.go`, `README.md`, `LOGBOOK.md`.
- Runtime directories were refreshed only through `agents-infra setup global --source-dir <Story worktree>`; no direct runtime edits were made.

## Validation evidence

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused production Setup parity test | 0 | `.temp/TASK-260830-27u51n-go-test-focused-02.log`; package reported `5.680s` |
| Full `go test ./internal/infra -count=1` | 0 | `.temp/TASK-260830-27u51n-go-test-infra-01.log`; package reported `206.795s` |
| Remaining module packages, uncached | 0 | `.temp/TASK-260830-27u51n-go-test-remaining-01.log`; root `123.780s`, attachments `3.147s`, modelharness `1.761s`, model-harness command package had no tests |
| `go build ./...` | 0 | `.temp/TASK-260830-27u51n-go-build-01.log` |
| `go vet ./...` | 0 | `.temp/TASK-260830-27u51n-go-vet-01.log` |
| `agents-infra setup global --source-dir <Story worktree>` | 0 | `.temp/TASK-260830-27u51n-setup-global-01.log` |
| `agents-infra verify global` | 0 | `.temp/TASK-260830-27u51n-verify-global-01.log` |
| Source versus installed `.agents` Claude module `cmp` | 0 | Byte-identical |
| Source versus `.claude/instructions` module `cmp` | 0 | Byte-identical |
| Required policy clauses in rendered `~/.codex/AGENTS.md` | 0 | `rg` found all blocking clauses |
| `git diff --check` | 0 | No whitespace errors |

## Negative evidence

- Temporarily broadened fallback authorization by removing the requirement for a verified external cause the agent cannot repair.
- The exact production-path parity test failed with exit `1`, naming the missing clause. Log: `.temp/TASK-260830-27u51n-go-test-mutant-expected-red-01.log`.
- Restored the source and reran the same focused test; exit `0`. Log: `.temp/TASK-260830-27u51n-go-test-focused-restored-01.log`.

## Execution anomaly

- The first focused-test shell invocation exited `1` before running Go because its output redirection used the nonexistent relative path `../../../.temp/...`. No test result was claimed from that invocation. The corrected absolute-path invocation is the exit-0 focused evidence above.
