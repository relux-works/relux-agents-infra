# TASK-260829-1q31e0 architecture revision required

Date: 2026-08-29, Europe/Moscow.

## Orchestrator verdict

The current retention architecture is **not development-ready**. Keep the
implementation Task in `analysis` until the architecture resolves both
acceptance-blocking lifecycle contracts below. The completed solution-architect
run acknowledged the directive but its final outcome still explicitly rejects a
migration path and contains no bounded lock-acquisition or scan-work contract.

## Finding 1: launch and status liveness is unbounded

The design requires create, append, close, recovery, pruning, and read-only
status to take the same profile `retention.lock`, then scan attacker- or
corruption-influenced directory contents. It does not define:

- a bounded lock wait or acquisition deadline;
- a bounded number of directory entries or bytes inspected per operation;
- a continuation/cursor or bounded maintenance path for excess inventory;
- the exact typed refusal or `unknown` result when either budget is exhausted;
- whether append may block indefinitely behind a stuck writer;
- a fairness rule that prevents status/maintenance from starving launch or
  append.

A crashed process normally releases a kernel lock, but a stuck same-UID writer
or a huge foreign/corrupt namespace can still hang launch and read-only status
indefinitely. That contradicts the hours/days/weeks stability objective.

### Required contract

Pin explicit operation budgets for create, append, close, and status. State the
clock and cancellation semantics, lock fairness/priority, maximum scan work per
operation, continuation behavior, and fail-closed result. Budget exhaustion
must preserve every unproven entry, perform no unsafe partial pruning, and
surface a stable typed diagnostic. Add deterministic production-path attacks
for a held lock, repeated status pressure, and inventory beyond the scan budget.

## Finding 2: existing installations have no healthy upgrade path

The design preserves every legacy lifecycle log forever and makes any legacy
presence force `within_policy=false` and `soak_ready=false`. It also explicitly
rejects migration, adoption, movement, and deletion. Therefore an upgraded
profile with old logs can remain permanently unhealthy even after all new
managed entries satisfy policy.

### Required contract

Define an explicit, safe upgrade/operator path. It may be a separate audited
command or a bounded retirement workflow, but it must:

- never silently adopt legacy files as managed ownership evidence;
- inventory and report exact candidates before mutation;
- require explicit operator intent for deletion or archival;
- use descriptor-relative no-follow inspection and preserve ambiguous/foreign
  evidence;
- be resumable and bounded under interruption;
- let a clean upgraded profile eventually reach `within_policy=true` and
  `soak_ready=true` without manual filesystem surgery;
- expose dry-run/result evidence and deterministic migration tests.

If the product intentionally chooses permanent unhealthy status instead, that
is a product decision and must be surfaced as such rather than called
development-ready.

## Re-entry gate

Publish a revised architecture outcome that closes both findings with exact
production call sites, status fields, refusal codes, operator flow, and negative
tests. Only then return the Task to `to-dev`. Do not start implementation and do
not contact a live model, runtime, service, socket, endpoint, or user-owned model
state while resolving this decision.
