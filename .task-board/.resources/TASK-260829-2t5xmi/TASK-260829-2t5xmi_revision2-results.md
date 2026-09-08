# TASK-260829-2t5xmi revision 2 developer evidence

## Outcome

Addressed reviewer finding F1 only. The production Pi config gate now rejects every runtime/shared-supervision seconds value whose effective `time.Duration` would overflow. It also validates coupled handoff and broker-ordering sums, limits doubled lease-stale duration, and clamps exponential backoff before doubling can overflow. No numeric defaults changed and no live model was used.

Production negative path: `RunPi -> loadCompositeProjectConfig -> parsePiRuntime`. `TestRunPiRejectsSupervisionSecondsThatOverflowEffectiveDurations` supplies values just above the explicit bound and proves refusal before provider lookup, runtime launch, shared-runtime directory creation, or restart-ledger mutation.

## Revision 2 files

- `tools/agents-infra/internal/infra/pi_config.go`
- `tools/agents-infra/internal/infra/pi_shared_supervision.go`
- `tools/agents-infra/internal/infra/pi_shared_client_darwin.go`
- `tools/agents-infra/internal/infra/pi_test.go`
- `tools/agents-infra/internal/infra/pi_shared_supervision_test.go`
- `README.md`
- `LOGBOOK.md`

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run 'TestRunPiRejectsSupervisionSecondsThatOverflowEffectiveDurations\|TestSharedRuntimeRestartBackoffClampsBeforeDurationOverflow' -count=1` | 0 | Focused production-gate and clamp tests passed after implementation and again after mutant restoration. |
| Narrowed mutant: change positive-duration refusal from `> maximum` to `> maximum+1`, then `go test ./internal/infra -run '^TestRunPiRejectsSupervisionSecondsThatOverflowEffectiveDurations$' -count=1` | 1 | Expected red. The real `RunPi` test detected admitted `max+1` values across runtime/supervision fields; source restored byte-for-byte from task-scoped copy. |
| `go test ./internal/infra -count=1` | 0 | Uncached package passed in 176.886s, including existing restart/quarantine, half-open, stable reset, manual quarantine, and abrupt-client-death fixtures. |
| `go test . ./cmd/model-harness ./internal/attachments ./internal/modelharness -count=1` | 0 | Remaining module packages passed; root package 117.521s. |
| `go vet ./...` | 0 | Vet clean. |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux compile passed. |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows compile passed. |
| `gofmt -d` on revision 2 Go files | 0 | No formatting diff. |
| `git diff --check` | 0 | No whitespace errors. |

## Constraints preserved

- Existing persisted ledger, status JSON, deterministic quarantine/backoff, half-open/stable reset, manual quarantine, and lease-release behavior remains covered by the uncached package run.
- Fixtures remain static subprocess fixtures; no live-model interaction was introduced or executed.
- Revision 2 adds validation bounds only; it adds or changes no numeric default.
