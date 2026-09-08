# TASK-260830-12n20p review verdict — Change Request revision 3

Verdict: **accepted**.

## Why this satisfies the task

The revised `TASK-260830-12n20p_agents-infra-contract-audit.md` now satisfies the acceptance criterion: it maps each agents-infra agentic-system/vendor/launch behavior to `covered`, `contract change`, or `stays here`, cites both repositories, explicitly inventories all production `agents-infra` command selectors, and includes the previously missing `model-check` behavior.

I reread the shipped contract from exact `skill-agents-management` tag `v0.3.0` (`3bec0baf9a0c897b0f76e1182e371a25132fa509`), especially `docs/architecture.md:13-82,183-257`, rather than relying on its README summary. The implementation sides match that contract:

- `pkg/agentic/plan.go:9-26,65-167` and `pkg/vendorplugin/spawn.go:103-190` prove the current single-process `Plan`, model/effort/runtime resolution, optional preflight, and `BuildPlan` dispatch.
- `pkg/agentic/systems/pi/pi.go:1-138`, `args.go:10-40`, and `preflight.go:17-75` prove Pi is shipped, the real turn grammar remains explicitly open, preflight fails closed, and Process B stays consumer-owned.
- `pkg/vendorplugin/vendors/local-models/config.go:30-177` and `vendor.go:8-60` prove the typed local-model pointer/catalog seam and `local-models` vendor identity.
- On the agents-infra side, `tools/agents-infra/main.go:46-83,94-133,417-628,669-918` covers the top-level launch inventory; `internal/infra/primary_session_launch_plan.go:24-155,204-420` proves the named host/client variants absent from the shipped single `Plan`; `child_launch_composition.go:11-70,94-198` proves config/provenance/envelope ownership; the Pi, model-harness, broker, standalone, and model-check files cited in the audit prove the remaining engine-plan gaps and consumer-specific process/policy ownership.

The audit explicitly names what must change in `skill-agents-management`: a general plugin graph/engine node, typed multi-node and named-variant plan contributions, and fail-closed graph/plan validation. It separately keeps agents-infra's config/provenance UX, process execution, runtime broker, state materialization, authorization, and behavior-evidence policy here.

## Adversarial evidence

- Enumerated all production `exec.Command*`/`.Start()` sites and traced agent/provider launches to the map. No bypass path or unclassified agent/provider launch was found; the separate `cmd/model-harness` run/stress/doctor surface is covered by the engine-plan/process-ownership row.
- `go test . -run '^TestModelCheckProductionEntrypoint$' -count=1` — PASS through the built CLI, including missing tool/text, earlier non-final text, failed tool, malformed stream, timeout, invalid deadline, overwrite, and non-managed-target negatives.
- `go test ./internal/infra -run '^TestModelCheckCleanupAttestationRefusesUnconfirmedStates$' -count=1` — PASS; narrowed cleanup-attestation refusal.
- Exact-tag `env -u TASK_BOARD_DIR go test ./pkg/agentic/... ./pkg/vendorplugin/... -count=1` — PASS; real `BuildPlan`/`BuildLaunch` entry points and their refusal tests.
- `go test ./internal/modelharness -count=1` — PASS.
- agents-infra `go build ./...` and `go vet ./...`, plus exact-tag skill-agents-management `go build ./...` and focused `go vet` — PASS.

The producer's previously reported Pi readiness timing test remains red. It predates and is outside this zero-delta research leaf, is not used as evidence for a classification, and does not contradict the ownership map; no full-suite-green claim is made from it.

## Empty repository delta

`repository_delta=empty` is correct. This leaf's requested product is the task-scoped written audit stored on the board, not a repository implementation or documentation patch. The exact base-to-candidate diff has zero paths, `git diff --check` is clean, and the worktree has no tracked change.

This uniquely named artifact was created after reviewer run `RUN-260829-7dc6bb` launched because the pre-existing generic verdict resource had no launch-time digest in the legacy run manifest and therefore could not serve as provably reviewer-owned acceptance evidence.
