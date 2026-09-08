# TASK-260830-12n20p review verdict — Change Request revision 3

Verdict: **accepted**.

## Acceptance rationale

The revised outcome satisfies the acceptance criterion. It provides a per-behaviour ownership map with explicit `covered`, `contract change`, and `stays here` outcomes, code citations on both repository sides, a complete `agents-infra` top-level command census, and the previously missing `model-check` classification.

The ownership boundary is supported by the exact shipped `skill-agents-management` contract at tag `v0.3.0` (`3bec0baf9a0c897b0f76e1182e371a25132fa509`), not by its README summary:

- `docs/architecture.md:13-82,183-257` defines the System/Vendor/runtime topology, the Pi/local-model component boundary, and the plan-value-versus-process-execution boundary.
- `pkg/agentic/plan.go:9-26,65-167` proves the current single-process `Plan` shape and the real `BuildPlan` dispatch/gates.
- `pkg/vendorplugin/spawn.go:103-190` proves runtime/model/effort resolution, Pi preflight, and the handoff into `BuildPlan`.
- `pkg/agentic/systems/pi/pi.go:1-138`, `args.go:10-40`, and `preflight.go:17-75` prove that Pi is shipped, Process B remains consumer-owned, turn grammar remains explicitly open, and preflight fails closed.
- `pkg/vendorplugin/vendors/local-models/config.go:30-177` and `vendor.go:8-60` prove the typed local-model catalog/pointer seam and the `local-models` vendor identity.

The agents-infra side was checked from commit `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`:

- `tools/agents-infra/main.go:46-83,94-133,417-628,669-918` accounts for every top-level selector and the real Codex, Claude, Pi, broker, canonical-target, composition, preparation, and model-check entry points.
- `internal/infra/primary_session_launch_plan.go:24-155,204-420` proves the simultaneous interactive/managed-host/managed-client shape that the shipped single `Plan` cannot represent.
- `internal/infra/child_launch_composition.go:11-70,94-198` proves the agents-infra-specific discovery/provenance/envelope surface around plugin grammar validation.
- `internal/infra/pi_plan.go:17-268`, `pi_launch_posix.go:47-304`, and the shared-runtime/standalone files cited by the audit prove the split between reusable engine/sidecar representation gaps and agents-infra-owned process, broker, trust, and materialization mechanics.
- `internal/modelharness/config.go:26-82,98-170,203-280`, `run.go:16-104`, and the production process-start inventory confirm the local/SSH engine-plan gap and consumer-owned execution/supervision.
- `internal/infra/model_check.go:146-713` plus `model_check_main_test.go:90-350` prove that `model-check` reuses an underlying launch value but owns behavior evidence, deadline/cleanup, JSONL interpretation, expectations, artifact safety, sanitization, and typed exits locally.

The audit also names precisely what must change in `skill-agents-management` (general plugin graph/engine node, typed multi-node and named-variant plan contributions, fail-closed graph/plan validation) and what must stay in agents-infra (configuration/provenance UX, process execution, runtime broker, state materialization, authorization, and behavior-check evidence policy).

## Adversarial review

- Enumerated every production process-start surface with `exec.Command*`/`.Start()` and traced it to the map. No unclassified agent/provider launch bypass was found; the separate `cmd/model-harness` run/stress/doctor surface falls under the existing engine-plan/process-ownership row.
- Drove the real built `agents-infra model-check` entry point with `go test . -run '^TestModelCheckProductionEntrypoint$' -count=1`: PASS. This test carries missing tool/text, earlier non-final text, failed tool, malformed stream, readiness/run timeout, invalid deadline, overwrite, and non-managed-target refusal shapes.
- Drove the narrowed cleanup-attestation refusal with `go test ./internal/infra -run '^TestModelCheckCleanupAttestationRefusesUnconfirmedStates$' -count=1`: PASS.
- Ran exact-tag `env -u TASK_BOARD_DIR go test ./pkg/agentic/... ./pkg/vendorplugin/... -count=1`: PASS. These suites exercise the real `BuildPlan`/`BuildLaunch` entry points and include unknown/missing/out-of-vocabulary effort, redirected launch, unresolvable runtime, malformed/absent local config, and Pi preflight refusal/timeout negatives.
- Ran `go test ./internal/modelharness -count=1`: PASS.
- Ran `go build ./...` and `go vet ./...` in agents-infra, plus exact-tag `go build ./...` and focused `go vet` in skill-agents-management: PASS.

The previously reported Pi readiness timing test remains red in producer evidence. It predates and is outside this zero-delta research leaf, is not presented as proof for any classification, and does not contradict the ownership map. No full-suite-green claim is made from it.

## Repository delta

`repository_delta=empty` is the correct outcome for this leaf. The requested deliverable is a written research/audit map stored as a task-scoped board outcome; no repository implementation or documentation edit was requested. The exact CR diff from base `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` to candidate tree `f6c67993fb17426a984262ca7e99e6f5765eecdb` has zero paths, `git diff --check` is clean, and the worktree has no tracked change.

Reviewer-created logs are under `.temp/TASK-260830-12n20p-review/`. No production or tracked repository file was modified during review.
