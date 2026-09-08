# TASK-260830-woqvhz Recovery Validation — Revision 2

Timestamp: 2026-08-30 09:59:09 +0300
Run: `RUN-260830-fb94e4`
Story worktree HEAD: `3295c7da7151de128f176cf7560a57d54c8f6c0d`

## Recovery cause

Change Request revision 2 validation command 1, `go test ./... -count=1`, exited 1 in the unrelated, separately tracked readiness timing fixture. The preserved board log reports both `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry` subtests red. No timeout, assertion, retry, ordering, product source, or test source was changed in response.

## Commands rerun directly in this recovery

Every command below ran as a standalone process without `tee` or a pipeline.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./... -count=1` | 0 | Exact previously failing CR gate recovered: 5/5 package outcomes; root 85.974s, attachments 1.919s, infra 143.374s, modelharness 1.663s. |
| `go test ./internal/infra -run '^TestSetupGlobal(PublishesExternalCILocalMirrorPolicyToClaudeAndCodex\|RejectsBroadenedRepairableOrMerelyInconvenientExternalCIMirrorTrigger\|RejectsAdditiveRepairableOrMerelyInconvenientExternalCIMirrorPermission\|RejectsAdditiveUnverifiedStatusExternalCIMirrorPermission)$' -count=1 -v` | 0 | Four named production-`Setup` tests passed; package 1.563s. |
| `go test -p 1 ./... -count=1` | 0 | Required serialized uncached suite: 5/5 package outcomes; root 69.793s, attachments 0.588s, infra 136.586s, modelharness 0.656s. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build ./...` | 0 | No diagnostics. |
| `git diff --check` | 0 | No whitespace errors. |
| `gofmt -l tools/agents-infra/internal/infra/infra_test.go` | 0 | No paths printed. |

## Candidate integrity

No repository source file changed during this recovery. The revision-2 candidate remains exactly four paths and `289 insertions, 5 deletions`:

Current `git diff --binary` SHA-256: `594fb5be9963035e28124dcd4d480cbc0d53a135d34c39c30bf819e9cf09b922`, exactly matching the previously attached revision-2 candidate.

- `.instructions/INSTRUCTIONS_WORKFLOW.md` — 8 additions
- `LOGBOOK.md` — 38 additions
- `README.md` — 14 additions
- `tools/agents-infra/internal/infra/infra_test.go` — 229 additions, 5 deletions

The previously attached canonical `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh`, `agents-infra verify global`, 6/6 installed parity, three live expected-red policy mutants, and candidate patch evidence all describe these unchanged bytes. They were accepted from the immediately preceding producer run rather than rerun here. The failed default LLDB path and readiness timing defect remain separately tracked and were not weakened or forced green.
