# TASK-260830-12n20p — rework recovery validation 02

## Outcome

The revised audit now enumerates all 18 exact selectors from `tools/agents-infra/main.go` and explicitly classifies `model-check`:

- **covered:** reusable runtime/model resolution, single-process launch value, and Pi preflight seam;
- **contract change in skill-agents-management:** none for `model-check` itself;
- **stays in agents-infra:** canonical-target restriction, managed lifecycle execution, deadline and cleanup, raw evidence protection, JSONL lifecycle/tool parsing, text/tool expectations, sanitization, overwrite refusal, and typed CLI exits.

The shipped-side evidence was reread from exact tag `skill-agents-management@v0.3.0` (`3bec0baf9a0c897b0f76e1182e371a25132fa509`) via a task-scoped archive. Neither repository was modified.

## Commands run directly

| Command | Working directory | Exit | Evidence |
| --- | --- | ---: | --- |
| `go test . -run '^TestModelCheckProductionEntrypoint$' -count=1` | `tools/agents-infra` | 0 | `model-check-production-01.log`; production CLI positive/negative matrix passed in 21.507s. |
| `go test ./internal/infra -run '^TestModelCheckCleanupAttestationRefusesUnconfirmedStates$' -count=1` | `tools/agents-infra` | 0 | `model-check-cleanup-attestation-01.log`; cleanup-attestation refusal passed in 0.459s. |
| `go build ./...` | `tools/agents-infra` | 0 | `go-build-01.log` (empty success output). |
| `go vet ./...` | `tools/agents-infra` | 0 | `go-vet-01.log` (empty success output). |
| `go test ./pkg/agentic/... ./pkg/vendorplugin/... -count=1` | exact `skill-agents-management@v0.3.0` archive | 0 | `skill-agents-management-v0.3.0-tests-01.log`; includes all seven systems, `BuildPlan`, `BuildLaunch`, local-models, and Pi preflight tests. |
| `go build ./...` | exact `skill-agents-management@v0.3.0` archive | 0 | `skill-agents-management-v0.3.0-build-01.log` (empty success output). |
| `python3 .temp/TASK-260830-12n20p/validate_audit.py` | repository root | 0 | `audit-census-02.log`: `audit-census-ok: 18 exact selectors; model-check ownership split present`. |
| task-scoped Markdown trailing-whitespace assertion | repository root | 0 | `audit-markdown-02.log`: `audit-markdown-ok: 162 lines`. |
| `git diff --check` | repository root | 0 | Empty success output. |

One initial invocation intended for the model-check test exited 1 before Go started because its log redirection named the wrong relative directory (`../../../.temp/...`). It supplied no test result and was corrected immediately; the corrected standalone command above exited 0.

## Red gate retained honestly

`go test ./internal/infra -run '^TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry$' -count=1` exited 1 on two direct attempts:

- the 1s runtime bound expired before the fixture recorded a 503 request;
- the exit-after fixture reported `runtime_readiness_timeout` instead of `runtime_exited_early`.

Evidence: `pi-readiness-production-01.log`, `pi-readiness-production-02.log`. The same two shapes caused the already-attached Change Request revision-2 full-suite command `go test ./... -count=1` to exit 1. At the later diagnostic checkpoint two foreign Go suites were active; no foreign process was stopped, and no causal claim is made from that observation.

This red readiness test is outside the audit artifact's read-only delta and is not presented as green or waived. The exact production `model-check` entrypoint matrix and cleanup-attestation refusal used for this rework are green.

## Evidence provenance

- Rerun in this successor: every command in the first table and both red focused attempts.
- Accepted from already-attached evidence only: Change Request revision-2 full-suite exit 1 and its full raw output.
- Repository status: tracked worktree clean; only ignored task-scoped `.temp/` evidence changed.
- `LOGBOOK.md` was read. The task's explicit “change nothing in either repository” constraint forbids adding a tracked logbook entry; the model-check ownership decision and red-gate anomaly are persisted in the revised board outcome and this task-scoped outcome instead.
