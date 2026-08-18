# Reviewer verdict — changes requested

Task: `TASK-260817-2h8hn4`  
Verdict branch: `changes_requested -> analysis`

## Evidence reviewed

- Outcome `TASK-260817-2h8hn4_pi-local-model-launch-contract.md`.
- Official Pi custom-model documentation: <https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/models.md>.
- Official Pi settings/trust documentation: <https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/settings.md>.
- Official Pi usage/CLI documentation: <https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/usage.md>.
- Official Pi source for `PI_CODING_AGENT_DIR`: <https://github.com/earendil-works/pi/blob/main/packages/coding-agent/src/config.ts>.
- Public DFlash supported-model inventory: <https://github.com/z-lab/dflash>.
- Board children/dependencies and `.planning/260817_131553_story-260817-1on8ex.md`.

The official sources substantiate the document's basic Pi facts: custom models live in the agent-dir `models.json`; project settings override global settings and are trust-gated; the documented model flags and session-directory precedence are accurate; and the agent directory is environment-overridable. The public DFlash inventory does not substantiate the two product labels as checkpoint IDs, so refusing to guess those coordinates is correct.

## Findings

### F1 — CLI model/provider overrides are not executable against the generated catalog

Section 4 says generated `models.json` contains only the selected profile's one provider/model entry. Section 5 independently allows explicit `--provider` and `--model` values to suppress those profile injections, and says a different model is valid whenever `/v1/models` exposes it. Pi's official model contract loads custom selectable models from `models.json`; endpoint discovery alone does not register an omitted custom model.

This leaves two invalid implementation branches: pass an override Pi cannot select from the isolated catalog, or allow a different provider/model to bypass the selected managed-profile identity while still starting its runtime. The positive scenarios exercise only an override equal to the configured provider/model, so the bypass survives.

Negative shape: **bypass path around the check**.

Required rework: choose and specify one exact rule. Either restrict managed-profile provider/model overrides to the exact generated identity and fail closed on any different selection, or define deterministic catalog materialization and metadata for every allowed override. Add production-entry negative cases for a different provider, a different endpoint-exposed model, a pattern, provider/model mismatch, and post-separator lookalikes.

### F2 — DFlash has no authoritative launch-time attestation gate

Sections 8 and 10 require a `*-dflash` profile to fail closed when speculative decoding is disabled or unproved. The exact TOML schema defines no status/capability endpoint, response selector, expected value, log attestation, or other authoritative evidence source. Readiness proves only the target model. Negative scenario 11 then permits either refusal **or** `unknown`, so absent evidence can take a permissive branch even though the profile name promises DFlash.

Negative shapes: **absent evidence treated as satisfied** and **capability claim that does not reproduce**.

Required rework: add an exact, safely bounded attestation contract to the profile schema (or explicitly defer the DFlash profile as unsupported). A launching `*-dflash` profile must refuse when evidence is absent, unreadable, malformed, false, stale, or for a different target/draft. `unknown` is acceptable only for non-launching diagnostics and must not reach Pi launch. Add real-entry tests for each refusal plus one authoritative positive fixture; narrow the attestation selector/value and require the positive test to fail.

### F3 — Produced planning artifact is not linked as an outcome

The checklist marks the planning-artifact requirement complete, but `.planning/260817_131553_story-260817-1on8ex.md` exists and neither this task nor its story exposes it as an outcome resource. This is a concrete mismatch between reported completion and board evidence.

Required rework: attach the existing plan to the appropriate task/story outcome with a task-scoped resource name, or document and prove that it is outside this task's produced planning artifacts before re-checking the item.

## Checks that passed

- The three-task dependency chain is present and proportional: decision -> implementation -> alias/documentation.
- The decision artifact is English, task-scoped, and covers TOML, precedence, lifecycle, security, failures, rejected alternatives, and production-entry acceptance scenarios.
- Global `~/.pi/agent` mutation is explicitly excluded for managed profiles.
- `git diff --check` passed during review.

## Verdict

Changes requested. Route to `analysis` because all findings require contract/research correction before implementation; none is an external blocker or human-only stop-the-line decision.
