## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Write failing tests for local/global instruction separation and idempotency
- [x] Implement project-local instruction scaffold without shared module copying
- [x] Preserve handwritten project instructions and valid Codex/Claude entrypoints
- [x] Update README and SKILL operator contract
- [x] Run focused and full agents-infra verification
- [x] Document the dirty-checkout to task-scoped worktree to reviewed-patch workflow in global instructions
- [x] Inspect and curate all pre-existing repository diffs into coherent verified commits
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"This is a focused Go setup and instruction-contract review with explicit local/global separation, idempotency, dirty-checkout safety, and commit-boundary checks; Sol medium is sufficient for an independent audit without unnecessary cost."}
spawn selection rationale for gpt-5.6-sol/medium: This is a focused Go setup and instruction-contract review with explicit local/global separation, idempotency, dirty-checkout safety, and commit-boundary checks; Sol medium is sufficient for an independent audit without unnecessary cost.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.0-40-g71fef12; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260810-007ba1, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260810-007ba1)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260810-007ba1, pid=55135, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [implementation-outcome.md](file://TASK-260810-3833d1/implementation-outcome.md) — Implementation, validation, smoke evidence, and commit mapping
- [TASK-260810-3833d1_results.md](file://TASK-260810-3833d1/TASK-260810-3833d1_results.md) — Task-scoped implementation and verification evidence
- [TASK-260810-3833d1_spawn-log_-reviewer--reviewer--codex-_RUN-260810-007ba1.log](file://TASK-260810-3833d1/TASK-260810-3833d1_spawn-log_-reviewer--reviewer--codex-_RUN-260810-007ba1.log) — System spawn log captured by task-board
- [TASK-260810-3833d1_review-verdict.md](file://TASK-260810-3833d1/TASK-260810-3833d1_review-verdict.md) — Accepted reviewer verdict with production-path, negative-shape, and full-suite evidence

## Created
2026-08-10T20:09:43Z

## Last Update
2026-08-10T20:34:56Z

## Assigned To
[reviewer] reviewer (codex)
