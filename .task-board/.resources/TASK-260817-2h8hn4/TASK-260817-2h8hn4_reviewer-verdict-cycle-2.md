# Reviewer verdict — changes requested

Task: `TASK-260817-2h8hn4`  
Verdict branch: `changes_requested -> analysis`

## Evidence reviewed

- Outcome `TASK-260817-2h8hn4_pi-local-model-launch-contract.md` and its byte-identical `.research` source.
- Prior verdict `TASK-260817-2h8hn4_reviewer-verdict-cycle-1.md`.
- Linked story plan `TASK-260817-2h8hn4_pi-story-plan.md` and the three-task dependency chain.
- Official Pi custom-model, settings, and usage documentation.
- Pi production argument parser at the artifact's pinned revision `a1bc0ec79010887210cc7de28714d72c78577dab`: `packages/coding-agent/src/cli/args.ts`.
- Public DFlash supported-target inventory.

The cycle-1 findings about generated-catalog identity, DFlash attestation, and the missing plan resource are materially corrected. Exact managed identity is now the stated rule, the DFlash gate is nonce/PID/freshness/target/draft bound and fail-closed, and the plan is a task-scoped outcome.

## Finding

### F1 — The claimed wrapper separator is a production bypass path

Sections 5 and 12 state that the wrapper preserves one literal `--` in the final Pi argv and that every following token is opaque prompt/operand content which cannot change managed identity. Pi's production parser at the pinned revision has no end-of-options branch. It iterates the entire argv; a literal `--` enters the generic `arg.startsWith("--")` branch as an empty unknown flag, may consume the next non-flag operand as its value, and then continues parsing later `--provider`, `--model`, `--api-key`, `--thinking`, and trust flags normally.

Therefore a real invocation shaped like `agents-infra pi --profile qwen -- prompt --provider openai --model gpt-... --api-key ...` is not protected by the separator after agents-infra forwards the promised literal argv. The wrapper may validate only pre-separator identity while Pi subsequently accepts post-separator native overrides. This directly defeats the managed-profile identity gate and contradicts negative scenario 7.

The same pinned parser recognizes native model/provider options only in spaced form. `--model=value` and `--provider=value` enter the generic unknown-extension-flag branch, so the contract must explicitly normalize those wrapper forms before Pi or reject them; it cannot leave parser parity implicit.

Negative shape: **bypass path around the check**.

Required rework:

1. Remove the assertion that forwarding literal `--` makes the suffix opaque to Pi.
2. Define one executable, fail-closed bridge for wrapper post-separator operands. It must either encode/deliver them through a Pi surface that cannot be reparsed as options, or reject unsupported option-looking payload shapes before any runtime starts. Preserve the exact intended prompt semantics or state the documented restriction.
3. Define whether `--flag=value` is wrapper syntax that is normalized to Pi's spaced argv or an invalid form. Do not describe it as native Pi parser behavior.
4. Add production-entry cases where post-separator `--provider`, `--model`, `--api-key`, `--thinking`, `--approve`, and an ordinary leading-dash prompt attempt to alter the final Pi interpretation. Assert the managed identity and trust boundary, not only agents-infra's pre-separator parse result.
5. Add a narrowing mutant that forwards the literal separator again (or scans only pre-separator tokens) and require the real-entry test to fail.

## Checks that passed

- Official Pi documentation substantiates custom `models.json`, auth availability, project trust/override semantics, model option syntax, and agent/session directory overrides.
- The exact-identity rules reject different endpoint-exposed IDs, patterns, provider mismatches, and Unicode separator lookalikes before runtime start.
- DFlash launch now refuses absent, unreadable, malformed, false, stale/future, replayed, or mismatched attestation; `unknown` is diagnostics-only.
- The linked plan and decision -> implementation -> alias/documentation dependencies are present and proportional.
- `task-board validate`, `git diff --check`, and decision-resource byte parity passed.

## Verdict

Changes requested. Route to `analysis`: this is a recoverable contract/source-parity defect requiring a safe CLI operand rule and revised acceptance scenarios, not an external or human-only stop-the-line blocker.
