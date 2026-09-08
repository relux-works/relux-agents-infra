# TASK-260829-1rinqw developer handoff

## Outcome

- `SharedRuntimeStatus` now emits `restart_not_before` and `half_open` on Darwin and in the non-Darwin compile-time shapes.
- `SharedRuntimeStatusReport` copies both values directly from `restart-ledger.json`; it never derives backoff from `restart_count`.
- Human-readable `runtime status` output carries the same two facts.
- Pre-extension (missing field), post-extension (RFC3339 field), explicit `null`, and malformed timestamp fixtures are pinned.
- A serving/ready static fixture with a non-zero historical restart count serializes `restart_not_before: null`.
- `last_failure` and `last_failure_at` are explicitly deferred: restart-ledger v1 persists neither reason nor timestamp, so the status schema does not fabricate either field.

## Validator-safe consumer mapping

The consumer must retain field presence. A missing pre-extension `restart_not_before` maps to `vendorplugin.UnknownAfterCheck("agents-infra runtime status --json.restart_not_before")`; malformed timestamps refuse the status read. Only a present, non-null RFC3339 deadline strictly after the observation time maps to:

```go
vendorplugin.LimitedUntil(
    restartNotBefore,
    vendorplugin.Observation{
        Source: "agents-infra runtime status --json.restart_not_before",
        Detail: "shared runtime restart backoff deadline",
        At: checkedAt,
    },
)
```

`LimitedUntil` supplies a non-zero evidence-derived `Until`, plus non-empty `Checked` and `Observed`, so `vendorplugin.Availability.Validate` accepts the verdict. A present `null` or elapsed deadline contributes no backoff verdict. `restart_count` and `half_open` alone never produce Limited; consumers continue with attested broker/runtime facts.

## Production and negative evidence

- Production call site: `SharedRuntimeStatusReport -> applySharedRuntimeLedgerStatus`.
- Malformed `restart_not_before`, `quarantined_until`, and `last_readiness_match` values reach `SharedRuntimeStatusReport` and refuse with `shared_runtime_state_unreadable`.
- Narrowing mutant: replacing the production deadline copy with `nil` made `TestSharedRuntimeStatusReportPublishesPersistedDeadlineWithoutInference` fail, exit 1. The source was restored from a task-scoped copy and verified byte-for-byte; the focused suite then passed.
- No user-owned or real model runtime was accessed. Tests used static JSON and existing fake process fixtures only.

## Validation evidence

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/infra -run '^TestSharedRuntimeStatus' -count=1` (baseline) | 0 | Existing focused baseline passed. |
| Same command after tests, before production fields | 1 | Expected-red compile failure: `SharedRuntimeStatus` lacked the new fields. |
| Same command after implementation/restoration (final) | 0 | All status contract, compatibility, serving-state, and malformed timestamp tests passed. |
| `go test . -run '^(TestRunRuntimeStatusJSONIsAbsentAndSideEffectFree\|TestPiOperatorContractDocumentsCycle10Boundary\|TestReluxAgentsInfraSkillRoutesSafePiWorkflowToSource)$' -count=1` (final) | 0 | CLI JSON and docs/skill contract tests passed. |
| `go test . -count=1` | 0 | Root CLI package passed. |
| `go vet ./...` (final) | 0 | Vet clean. |
| `gofmt -d ...` | 0 | No formatting diff. |
| `go build ./...` | 0 | Native Darwin build passed. |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux compile passed. |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows compile passed. |
| `git diff --check` (final) | 0 | No whitespace errors. |

## Non-green broad-suite evidence

- `go test ./internal/infra -count=1`, attempt 1: exit 1 after 168.375s on unrelated `TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC`; its exact rerun passed, exit 0.
- `go test ./internal/infra -count=1`, attempt 2: exit 1 after 354.180s on unrelated `TestPiLaunchMusePassesExactTargetDraftArgv` and `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry`. Exact rerun cleared the Muse test, while the readiness fixture remained red with missing `readiness-count` and timeout-vs-early-exit mismatches. No changed file is in that launch-readiness path; all task-focused suites are green.
- `GOOS=linux GOARCH=amd64 go test ./... -run '^$'`: exit 1, expected probe error because macOS attempted to execute Linux test binaries (`exec format error`). The correct compile-only `go build ./...` gate passed for Linux.

## Workspace evidence

- Story branch: `task-board/story/STORY-260829-2qtrqa`
- Selected base/tip at start: `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`
- Tool readiness: `.temp/TASK-260829-1rinqw/tool-readiness.log`
- Architectural decision recorded at `LOGBOOK.md` under `2026-08-29 / 1756`.
