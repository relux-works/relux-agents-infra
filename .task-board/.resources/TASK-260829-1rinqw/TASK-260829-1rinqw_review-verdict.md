# TASK-260829-1rinqw review verdict

## Verdict

Accepted Change Request `CR-TASK-260829-1rinqw-1` revision 1.

Reviewed base `891de4427bb7de6885b8b221f0e2b24a49a8fdc2` against candidate tree `ae609b2fdf83db0b291f80036cd4f72417856184`. The exact binary diff hashes to `2e8b9ac54821720abd441df5fc0a809a20d72b7fc293c0990db93e8065d41089`, matching the handed-off patch digest. The worktree matched the candidate tree before and after review.

No blocking or rework finding was found.

## Acceptance evidence

- `SharedRuntimeStatusReport -> applySharedRuntimeLedgerStatus` copies persisted `restart_not_before` and `half_open` without deriving either from `restart_count`, broker state, or readiness history.
- `restart_not_before` has no `omitempty`, so the post-extension JSON key is always present as an RFC3339 timestamp or `null`. A ready/serving fixture with historical `restart_count=2` and `half_open=true` remains `restart_not_before:null`.
- Pre-extension missing-key and post-extension timestamp fixtures are pinned. Malformed `restart_not_before`, `quarantined_until`, and `last_readiness_match` values are driven through the production `SharedRuntimeStatusReport` entry point and refused as `shared_runtime_state_unreadable`, so read failure is not treated as absence.
- `half_open` is published as additive ledger evidence but explicitly documented as non-gating. `last_failure` and `last_failure_at` are explicitly deferred and absent because restart-ledger v1 persists neither reason nor time; tests refuse fabricated last-failure output.
- The consumer handoff names the exact presence-aware mapping: only a present, non-null future deadline produces `vendorplugin.LimitedUntil(deadline, Observation{Source: "agents-infra runtime status --json.restart_not_before", Detail: "shared runtime restart backoff deadline", At: checkedAt})`. Missing legacy evidence maps to `UnknownAfterCheck`; null or elapsed deadlines, `restart_count`, and `half_open` do not mint Limited. This shape supplies non-zero `Until`, `Checked`, and `Observed` and satisfies `Availability.Validate`.
- Darwin production, unsupported POSIX, unsupported Windows, human-readable CLI, README, and skill surfaces carry a consistent additive contract.

## Gate attack

In an isolated `.temp` copy of the exact candidate, the production copy was narrowed so `restart_not_before` was published only when `restart_count >= 3`. The candidate fixture persists a valid deadline with `restart_count=2`. Running `go test ./internal/infra -run '^TestSharedRuntimeStatusReportPublishesPersistedDeadlineWithoutInference$' -count=1` exited 1 at `pi_shared_supervision_test.go:258`, proving the production call-site test detects a narrower bypass rather than only total deletion. The candidate was not modified.

## Validation

Reviewer-rerun commands:

- `go test ./internal/infra -run '^TestSharedRuntimeStatus' -count=1` — pass.
- `go test . -run '^(TestRunRuntimeStatusJSONIsAbsentAndSideEffectFree|TestPiOperatorContractDocumentsCycle10Boundary|TestReluxAgentsInfraSkillRoutesSafePiWorkflowToSource)$' -count=1` — pass.
- `GOOS=linux GOARCH=amd64 go build ./...` — pass.
- `GOOS=windows GOARCH=amd64 go build ./...` — pass.
- `go vet ./...` — pass.
- `git diff --check <base> <candidate>` — pass.

Accepted from the immutable candidate validation resource rather than rerun broadly: `go test ./... -count=1` passed all packages, including `internal/infra` in 275.040s; the same resource records `go vet ./...` pass. Review used only static/fake fixtures and did not access a live runtime.
