# Reviewer Verdict — Cycle 6

Task: `TASK-260817-2h8hn4`
Verdict: changes requested
Route: `analysis`

## Review scope and evidence

- Re-read the task outcome `TASK-260817-2h8hn4_pi-local-model-launch-contract.md`, the linked three-task plan, Story children/dependencies, and prior cycle findings.
- Verified the current official Pi custom-model, settings, trust, model-selection, and session CLI surfaces from the three task-linked documents on 2026-08-17. The redirect to `earendil-works/pi` is current; the contract's custom `models.json`, project-over-global settings merge, trust flags, model flags, and file-operand claims match those documents.
- Checked current DFlash documentation. It documents launch configuration and supported target/draft pairs but no runtime-owned status/attestation endpoint that independently proves active DFlash state.
- `task-board validate` and `git diff --check` pass. Source and outcome contract hashes match at `f170355332267a620029748eb9b77db1499f9242d9aa51636d491ea14163b9b4`.

## Finding F1 — forged or self-minted evidence

The cycle-6 schema is exact, fresh, and fail-closed on malformed fields, but its authority is circular. The arbitrary project-selected `runtime.executable` or direct-child adapter receives `AGENTS_INFRA_RUNTIME_LAUNCH_NONCE`, knows its own PID and the expected model/capabilities/target/draft from argv/config, and is then allowed to produce the JSON object that the launcher treats as authoritative.

A production-faithful bypass is therefore still admitted:

1. The selected direct-child adapter binds the configured loopback port, so its PID and nonce satisfy every written ownership field.
2. It proxies `/v1/models` and inference traffic to a pre-existing foreign or target-only backend that the launcher did not start or own.
3. It returns the exact required attestation object from local config/environment, including `capabilities:["dflash","text","tools"]` and `dflash.active=true`, although the backend is not owned and DFlash is inactive.
4. The current contract reaches Pi launch: response shape, nonce, PID, timestamp, model, capability order, and target/draft all match. This contradicts the stated no-attach ownership boundary and the requirement that DFlash evidence be authoritative rather than an argv/config echo.

The existing negative scenarios mutate fields after accepting the attestation producer as authority. They do not attack whether that producer is entitled to make the claim. The positive Muse fixture explicitly mints the expected object, so it proves schema reachability, not authority.

## Required rework

1. Define the attestation trust root and ownership mechanism. For example, use an agents-infra-owned/versioned adapter that reserves or owns the public listening socket, owns and reaps the complete backend process tree, and derives model/tool/DFlash state from an authoritative backend observation. If the backend exposes no authoritative DFlash state, launch must refuse; argv/config echo is not evidence.
2. State precisely which runtime/adapter identities are trusted to attest, how project TOML may select them without minting the authority, and what threat boundary remains. If arbitrary project executables are intentionally trusted as the authority, stop claiming the launcher independently proves listener/backend ownership or active DFlash and reconcile that weaker claim with the task DoD.
3. Add a real-entry negative case using a direct-child proxy that returns a perfectly formed current-nonce/current-PID attestation while forwarding to a foreign target-only backend. It must refuse before Pi. Add a narrowing mutant that replaces the authoritative observation with config/env echo and require that named test to fail.
4. Carry the corrected authority requirement into `TASK-260817-ccpnlm` and operator documentation acceptance criteria.

This is ordinary architecture/research rework, not an external blocker. Route the task to `analysis` for another producer/reviewer cycle.
