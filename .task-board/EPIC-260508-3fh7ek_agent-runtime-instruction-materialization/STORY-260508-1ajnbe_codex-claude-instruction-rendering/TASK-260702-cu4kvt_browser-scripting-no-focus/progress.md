## Status
done

## Assigned To
[reviewer] reviewer (codex)

## Created
2026-07-02T09:44:48Z

## Last Update
2026-08-29T23:04:39Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Add browser automation no-focus instruction module
- [x] Include module in Claude/Codex instruction entrypoints
- [x] Update README instruction module list
- [x] Sync global runtime and verify installed instructions
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
Added INSTRUCTIONS_BROWSER_AUTOMATION.md with no-focus-by-default browser scripting policy, linked it from AGENTS.md and INSTRUCTIONS.md, updated README, ran agents-infra setup global --source-dir ~/src/relux-works/relux-agents-infra, and verified installed ~/.agents instructions contain the new module.
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Routing a long-parked to-review element to a verdict; the review backlog is a goal clause and these sit in stories whose worktree lease is free."}
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Routing a long-parked to-review element to a verdict as the review-backlog goal clause requires."}
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Routing a parked to-review element now that acceptance criteria exist to judge it against."}
spawn selection rationale for gpt-5.6-sol/high: Routing a parked to-review element now that acceptance criteria exist to judge it against.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-550a3e, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-550a3e)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-550a3e, pid=50388, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260702-cu4kvt_spawn-log_-reviewer--reviewer--codex-_RUN-260829-550a3e.log](file://TASK-260702-cu4kvt/TASK-260702-cu4kvt_spawn-log_-reviewer--reviewer--codex-_RUN-260829-550a3e.log) — System spawn log captured by task-board
- [TASK-260702-cu4kvt_review-verdict.md](file://TASK-260702-cu4kvt/TASK-260702-cu4kvt_review-verdict.md) — Accepted reviewer verdict with delivery-surface, installed-runtime, negative-path, and test evidence
