# TASK-260830-11ajl2 — checklist realignment, recorded verbatim

The task description and acceptance criteria were rewritten on 2026-08-31 for option B of
the first-consumer gap report. The **checklist in `progress.md` was not** — it still carried
the four DoD items the spawn brief said were gone:

> This task's description and acceptance criteria are rewritten. DoD items that v0.5.0
> cannot satisfy are gone. **The hardcoded `codex`, `claude` and `pi` branches stay**, and
> that is now the correct outcome rather than a shortfall.

This run aligned the checklist with the ratified AC. Nothing here is a judgement about scope
— that decision was already made — but the removals are recorded verbatim so a reviewer can
restore any of them exactly if this reading is wrong.

## Removed, with the reason from the ratified scope

1. `Hardcoded codex/claude/pi agent branches removed from agents-infra, not merely bypassed`
   — contradicted by the brief: "The hardcoded `codex`, `claude` and `pi` branches stay."
   v0.5.0's composition contract is validate-only and `agentic.Plan` is single-variant, so
   the branches can only be bypassed, which the item itself forbids.

2. `Adding an environment afterwards requires no agents-infra change; prove with a narrowing
   mutant` — false by construction at v0.5.0. A new environment still needs a case in
   `lockCanonicalTargetArguments` (`canonical_target.go:417-427`) and a
   `buildXPrimarySessionLaunchPlan` (`primary_session_launch_plan.go:249-256`), and the
   audit assigns both to agents-infra. The "generalization" that would satisfy it — driving
   admission off registry membership — is a gate **weakening**, from 3 admitted environments
   to 7 registered systems.

3. `agents-infra depends on and imports skill-agents-management; hardcoded codex, claude and
   pi branches are gone` — the first half landed and is now its own item; the second half is
   removal 1 again.

4. `Every contract gap found is fixed in skill-agents-management and released, with the
   version recorded` — forbidden by the brief: "No new agents-management release, and no
   version other than exact `v0.5.0`." The accepted lockstep plan requires a fresh
   independent human acceptance before any release in this sequence. The gaps are recorded
   instead, in `TASK-260830-11ajl2_consumer-gaps.md`, and the structural one is
   `TASK-260831-1pfnxx`.

## Added, from the rewritten acceptance criteria

- `agents-infra depends on and imports skill-agents-management v0.5.0 and uses it for
  SystemID normalization and registry cross-check`
- `The Go floor bump from 1.21.0 to 1.25.5 is stated and its effect on build and CI
  assumptions checked`
- `Hardcoded codex, claude and pi branches remain; no branch is removed that v0.5.0 cannot
  replace`

## Kept unchanged

Every other item, including `Existing composition, primary-session, targets and model-check
tests pass unchanged`, `No existing test was edited to make the change land`, and
`A list of the gaps the first consumer found is attached for the board migration to reuse`.
All three are satisfied and evidenced in `TASK-260830-11ajl2_results.md`.
