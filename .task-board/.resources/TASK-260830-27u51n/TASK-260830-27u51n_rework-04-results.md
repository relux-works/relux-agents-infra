# TASK-260830-27u51n rework 04 results

## Outcome

Closed the reviewer-proven Claude reachability bypass in the external-CI local
mirror policy parity test. The production `Setup` entry point must now produce
the complete chain below:

1. `.claude/CLAUDE.md` equals the generated Claude entrypoint and loads
   `@instructions/INSTRUCTIONS.md`.
2. `.claude/instructions` links to the managed `.agents/.instructions` tree.
3. The installed `INSTRUCTIONS.md` is byte-equal to the versioned source and
   contains the exact include for `INSTRUCTIONS_WORKFLOW.md`.
4. The installed workflow is byte-equal to the versioned source, while the
   rendered Codex `AGENTS.md` contains the complete policy.

This is a test-only rework on top of the existing policy, README, and logbook
delta. No production policy text or setup implementation was weakened.

## Negative evidence

Removing only
`@~/.agents/.instructions/INSTRUCTIONS_WORKFLOW.md` from the versioned Claude
instruction index caused the focused production setup test to fail with exit
code 1 at the installed-index assertion. The source include was then restored,
and the focused test passed with exit code 0. This attacks the reviewer's exact
bypass shape; deleting the workflow file or the whole test was not used as the
mutant.

Evidence: `expected-red-claude-include-mutant-04.log` and
`focused-claude-chain-final-04.log`.

## Validation

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/infra -run '^TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex$' -count=1` | 0 | Final focused production setup/wiring test passed. |
| Same focused test with only the Claude workflow include removed | 1 | Expected red; failed at installed Claude index reachability assertion. |
| `go test ./... -count=1` on the candidate before the final index byte-equality assertion | 0 | Full uncached Go suite passed. |
| `go test ./... -count=1` on the exact final candidate | 1 | Truthful red under concurrent board validation: model-check PID evidence and two readiness fixtures were not created before their short deadlines. |
| Exact isolated model-check deadline test | 0 | Passed uncached. |
| Exact isolated readiness bounds test | 0 | Both previously failing readiness subtests passed uncached. |
| `go test ./internal/infra -count=1` on the exact final candidate after host contention drained | 1 | All but one test passed; pinned-Pi RPC side-effect fixture did not create its file in time. |
| Exact isolated pinned-Pi direct-RPC test | 0 | Passed uncached. |
| `go vet ./...` | 0 | Passed on the exact final candidate. |
| `go build ./...` | 0 | Passed on the exact final candidate. |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | Canonical repository bootstrap refreshed the global runtime from this source worktree. |
| `agents-infra verify global` | 0 | Installed runtime postcondition passed. |
| Installed Agents/Claude byte parity, Claude entrypoint/include chain, and Codex policy clause gate | 0 | Passed. |
| `git diff --check` | 0 | Passed. |

The non-zero broad-suite runs remain attached as failures, not represented as
passes. Their complete failure sets were rerun through the named production
entry tests and passed uncached in isolation. No timing threshold or unrelated
runtime code was changed to manufacture green evidence.

The Story worktree was at base commit
`5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` with `HEAD..main = 0` when the
rework began. The integration owner should retain the configured clean
PR-head/post-integration setup and parity validation when landing the accepted
Change Request.

## Logs

- `focused-claude-chain-final-04.log`
- `expected-red-claude-include-mutant-04.log`
- `go-test-all-final-04.log`
- `go-test-infra-final-04.log`
- `isolated-model-check-deadline-04.log`
- `isolated-readiness-bounds-04.log`
- `isolated-direct-rpc-04.log`
- `go-vet-all-final-04.log`
- `go-build-all-final-04.log`
- `setup-global-04.log`
- `verify-global-04.log`
- `installed-runtime-parity-final-04.log`
