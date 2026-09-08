# TASK-260830-woqvhz Developer Outcome — Revision 2

## Candidate

- Preserved the orchestrator-selected Story base `3295c7da7151de128f176cf7560a57d54c8f6c0d`; local `main` is four commits ahead and was not merged, rebased, or moved because base movement is orchestrator-owned.
- Scope remains exactly four paths: `.instructions/INSTRUCTIONS_WORKFLOW.md`, `LOGBOOK.md`, `README.md`, and `tools/agents-infra/internal/infra/infra_test.go`.
- Diff: 4 files, 289 insertions, 5 deletions.
- Candidate patch: `TASK-260830-woqvhz_candidate-rev2.patch`.
- Patch SHA-256: `594fb5be9963035e28124dcd4d480cbc0d53a135d34c39c30bf819e9cf09b922`.

## Rework

- Replaced presence-only clause validation with extraction and byte-exact comparison of the complete accepted revision-4 External-CI section through the next top-level heading.
- Preserved the exact policy bytes and the production Claude `CLAUDE.md -> instructions symlink -> INSTRUCTIONS.md -> INSTRUCTIONS_WORKFLOW.md` chain plus rendered Codex `AGENTS.md` inclusion.
- Added named production-`Setup` regressions for the reviewer's additive repairable-or-inconvenient permission and a differently-shaped partial-or-inconclusive-status exception. Both retain the complete exclusive trigger and require rejection on Agents, Claude, and Codex installed surfaces.
- Preserved the replacement-shaped broadened-trigger regression.
- Recorded the surviving additive bypass, exact-block fix, and negative evidence in `LOGBOOK.md`.

## Validation

Every gate ran as a standalone process. Non-zero mutant runs are reported as expected failures, not passes.

| Gate | Exit | Exact result |
| --- | ---: | --- |
| Restored focused production-`Setup` tests, uncached verbose | 0 | 4/4 named tests passed |
| Reviewer additive repairable/inconvenient live policy mutant against `TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex` | 1 | Expected red: installed exact policy block differed |
| Distinct additive partial/inconclusive-status live policy mutant against the same production test | 1 | Expected red: installed exact policy block differed |
| Replacement broadened-trigger live policy mutant against the same production test | 1 | Expected red: installed exact policy block differed |
| `go test -p 1 ./... -count=1` | 0 | 5/5 package outcomes green: 4 test-bearing packages passed, 1 package had no test files; root 84.659s, attachments 0.604s, infra 150.754s, modelharness 1.076s |
| `go vet ./...` | 0 | Green |
| `go build ./...` | 0 | Green |
| `git diff --check` | 0 | Green |
| Exact changed-path scope | 0 | 4/4 paths match declared scope |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | Built/installed both CLIs and verified global setup; LLVM 23 default LLDB compatibility remains separately tracked |
| `agents-infra verify global` | 0 | Verified global runtime |
| Installed parity check | 0 | 6/6 checks: Agents workflow bytes, Claude symlink, Claude index bytes, Claude entrypoint bytes, Claude workflow include, rendered Codex workflow bytes |

## Evidence Files

- `TASK-260830-woqvhz_focused-restored-rev2.log`
- `TASK-260830-woqvhz_additive-repairable-live-mutant-rev2.log`
- `TASK-260830-woqvhz_additive-unverified-status-live-mutant-rev2.log`
- `TASK-260830-woqvhz_replacement-live-mutant-rev2.log`
- `TASK-260830-woqvhz_go-test-all-serial-rev2.log`
- `TASK-260830-woqvhz_go-vet-rev2.log`
- `TASK-260830-woqvhz_go-build-rev2.log`
- `TASK-260830-woqvhz_git-diff-check-rev2.log`
- `TASK-260830-woqvhz_setup-global-skip-lldb-rev2.log`
- `TASK-260830-woqvhz_verify-global-rev2.log`
- `TASK-260830-woqvhz_installed-parity-rev2.log`
