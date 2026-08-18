# Reviewer Verdict — Cycle 5

Task: `TASK-260817-2h8hn4`
Verdict: changes requested
Route: `analysis`

## Finding

### F1 — Qwen readiness is not bound to the owned runtime child

The contract promises that managed launch does not attach to an unknown listener and acceptance scenario 12.4 requires a pre-existing listener to prevent Pi launch. The Qwen path cannot enforce that promise:

- section 7 starts the configured runtime child and checks only that the child remains alive;
- it then accepts the exact model ID from `GET <base_url>/models`;
- unlike the Muse DFlash path, Qwen readiness has no fresh nonce, direct-child PID, listener identity, inherited socket, or other proof that the response came from the selected runtime child.

These observations admit two indistinguishable states: the selected child owns the endpoint, or an unrelated pre-existing listener owns it while the selected child stays alive without binding. The latter reaches Pi launch under the written algorithm.

Concrete attack evidence is in `.temp/TASK-260817-2h8hn4/reviewer-foreign-listener-bypass-cycle5-01.log`: an independent loopback listener returned `qwen-3.8-27b` while a different selected-runtime child PID remained alive and owned no listener. Every fact required by the Qwen readiness gate was true, although endpoint ownership was false.

Negative shape: **bypass path around the check**. This is also positive-path-only evidence: the existing Qwen readiness scenarios prove model discovery but do not prove child ownership.

## Required rework

1. Define one production-enforceable child-bound readiness contract for every managed runtime, not only DFlash. A clean option is a fresh launcher nonce plus exact direct-child PID and model identity from an authoritative runtime/adapter endpoint. If a socket-transfer protocol is chosen instead, specify its ownership and lifecycle semantics exactly.
2. Do not treat preflight port absence alone as proof; it has a check-to-bind race and does not establish which process serves the later response.
3. Make Qwen launch refuse absent, unreadable, malformed, stale, replayed, wrong-nonce, wrong-PID, wrong-model, foreign-listener, and child-exited evidence before Pi starts. Keep absence distinct from failed observation.
4. Drive the real future `agents-infra pi` entry point with a foreign listener plus a live non-binding runtime child and require refusal. Narrow the ownership gate to liveness plus `/v1/models`, or remove only the nonce/PID field, and require that named test to fail.
5. Update the exact TOML schema, diagnostics, failure codes, lifecycle, security boundary, rejected alternatives, acceptance scenarios, and downstream implementation/operator acceptance criteria.

## Checks completed

- Re-read the complete cycle-five decision artifact and confirmed byte parity with its task outcome (`10762c3c307cff76c9ee53c2608ed9178831b5a51fd9939f56c95060123cb1e3`).
- Rechecked official Pi models, settings/trust, and usage documentation on current `main`; the documented custom-model, compatibility, trust, model-option, and local-file behavior supports the corresponding contract sections.
- Rechecked the official Pi `v0.84.2` standalone asset checksum and local extracted evidence: darwin-arm64 asset SHA-256 `c996e888...`, native arm64 standalone entrypoint, 217 regular files, and matching canonical manifest evidence.
- Confirmed cycle-four's npm/Node bypass is closed by the standalone-only catalog, loader-environment refusal, canonical absolute execution, and point-of-use full-tree recheck with real-entry negative/narrowing scenarios.
- Confirmed exact managed provider/model identity, fake-separator handling, DFlash nonce/PID attestation, no-global-Pi-state boundary, process cleanup, three-task dependency chain, and task/resource traceability are otherwise explicit and proportional.
- `task-board validate`, `git diff --check`, and decision/outcome parity passed. No product build/test applies to this no-implementation research task.

Official references:

- https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/models.md
- https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md
- https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md
- https://github.com/earendil-works/pi/releases/tag/v0.84.2

## Verdict

Changes requested. Route to `analysis`: the gap is a recoverable launch-contract defect, not an external blocker or human-only decision.
