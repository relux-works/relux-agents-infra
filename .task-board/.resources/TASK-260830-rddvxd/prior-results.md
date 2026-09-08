# TASK-260830-woqvhz Developer Outcome

## Candidate

- Selected base after fresh fetch: `HEAD == origin/main == 3295c7da7151de128f176cf7560a57d54c8f6c0d` (`0/0` ahead/behind).
- Final tracked scope: `.instructions/INSTRUCTIONS_WORKFLOW.md`, `LOGBOOK.md`, `README.md`, `tools/agents-infra/internal/infra/infra_test.go`.
- Final diff: 4 files, 212 insertions, 5 deletions.
- Reviewable patch: `TASK-260830-woqvhz_candidate.patch`.
- Patch SHA-256: `4ae999d4491f5baf82bcd4dc990b7a15ce2b2581d477244e9ee6315c0f4df78e`.

## Implementation

- Replayed the exact revision-4 External-CI mirror policy semantics beneath current trunk's automatic signed-delivery contract without weakening exact-head review, checks, signatures, landing, merge queues, or branch protection.
- Preserved README model/config documentation and added the revision-4 operator summary.
- Preserved the production Claude `CLAUDE.md -> instructions symlink -> INSTRUCTIONS.md -> INSTRUCTIONS_WORKFLOW.md` chain and the rendered Codex `AGENTS.md` chain.
- Strengthened the Setup-path policy assertion from an interior substring to the complete exclusive trigger: `Use a local mirror only when ... verified external cause that the agent cannot repair.`
- Added `TestSetupGlobalRejectsBroadenedRepairableOrMerelyInconvenientExternalCIMirrorTrigger`, which drives production `Setup`, publishes the broadened policy to Agents/Claude/Codex surfaces, retains the formerly asserted hosted-external-cause phrase, and requires semantic validation to reject it.
- Recorded revision-4 decisions, the surviving broadened mutant, current-main replay, LLVM 23 setup handling, and the observed readiness timing anomaly in `LOGBOOK.md`.

## Validation

All commands ran directly as standalone processes. Expected-red mutants are reported as failures, not passes.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run '^(TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex|TestSetupGlobalRejectsBroadenedRepairableOrMerelyInconvenientExternalCIMirrorTrigger)$' -count=1` | 0 | `focused-restored-03.log` |
| Broadened repairable-or-merely-inconvenient live mutant; same focused production Setup test | 1 (expected red) | `broadened-live-mutant-01.log`; missing complete exclusive trigger on installed workflow |
| Claude workflow-include removal; same focused production Setup test | 1 (expected red) | `claude-include-live-mutant-01.log`; installed Claude index lacks workflow include |
| Codex workflow-include removal; same focused production Setup test | 1 (expected red) | `codex-include-live-mutant-01.log`; rendered Codex instructions lack workflow source |
| First serialized uncached `go test -p 1 ./... -count=1` | 0 | `go-test-all-serial-01.log` |
| Second serialized uncached full run after LOGBOOK update | 1 | `go-test-all-serial-02-final.log`; preserved red readiness timing anomaly, not accepted as success |
| Exact red readiness subtest, uncached `-count=5` | 0 | `readiness-repro-01.log`; 5/5 bounded reproduction |
| Authoritative serialized uncached `go test -p 1 ./... -count=1` on unchanged final candidate | 0 | `go-test-all-serial-03-final.log`: root 134.576s, attachments 4.195s, infra 212.743s, modelharness 0.729s |
| `go vet ./...` on unchanged final candidate | 0 | `go-vet-all-02-final.log` |
| `go build ./...` on unchanged final candidate | 0 | `go-build-all-02-final.log` |
| `git diff --check` and exact four-path scope assertion | 0 | `git-diff-check-02-final.log` plus direct scope gate |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | `setup-global-skip-lldb-02-final.log` |
| `agents-infra verify global` | 0 | `verify-global-03-final.log` |
| Installed source-byte, Claude entrypoint/symlink/include, and rendered Codex exclusive-trigger parity | 0 | Direct final parity gate after canonical setup |

## Setup Scope Note

`AGENTS_INFRA_SKIP_LLDB_MCP=1` is the documented temporary lane for the separately owned Homebrew LLVM 23 `lldb-mcp` bootstrap defect. The setup evidence proves installation and instruction parity; it does not claim default LLDB MCP compatibility.

## Toolchain

- `task-board 0.24.3-169-ge4022da4`
- `git 2.53.0`
- `go1.25.5 darwin/arm64`
