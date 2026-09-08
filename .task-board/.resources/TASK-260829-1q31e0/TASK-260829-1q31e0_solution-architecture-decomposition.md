# TASK-260829-1q31e0 solution architecture decomposition

Date: 2026-08-29, Europe/Moscow.

## Verdict

Keep one code Task under `STORY-260829-1xvrur`:

- `TASK-260829-1q31e0: bound-pi-lifecycle-log-retention`

This is the smallest development-ready decomposition. Configuration, the dedicated profile aggregate namespace, entry-envelope lifecycle and recovery, the shared scanner, production launch wiring, status, documentation, and adversarial/soak gates are one ownership boundary. Splitting any of those into independently acceptable leaves would permit a production bypass or a status/admission contract mismatch.

## Requirement traceability

| Story requirement | Owning Task acceptance boundary |
| --- | --- |
| AC1: explicit positive overflow-safe count, byte, and age bounds with no numeric defaults | Task AC1 and the `RunPi` configuration refusal test at the production entry point |
| AC2: one profile aggregate root while per-run agent/session state stays isolated | Task AC2; both `RunPi` exclusive/standalone and `runSharedPiSession` must reach the same lifecycle root |
| AC3: strict random-ID envelope deletion authority under the trusted effective-UID boundary | Task AC3-AC4; descriptor-relative traversal, exact inventory, active locks, identity revalidation, exact non-recursive unlink, and preservation of unproven evidence |
| AC4: deterministic bounded pruning and truthful scanner-derived diagnostics | Task AC5-AC6; newest inactive prefix, active exhaustion refusal, managed JSONL and envelope bytes, legacy/foreign/unknown reporting, `within_policy`, and `soak_ready` |
| AC5: adversarial production-entry and long-horizon proof without a live runtime | Task AC7; bypass, narrowing, mutation, forged-close, crash-partial, unreadable, and concurrency attacks plus race, eight-week fake-clock soak, full Go, vet, build, format/diff, and compile gates |

## Architecture and scope checks

- Exact implementation base is `6d051f54440d36e3ca3d132f8d9d1e78d46289de`; the assigned Story worktree is at that commit and zero commits behind local `main`.
- The worktree contains the rejected revision-5 candidate as uncommitted evidence. This analysis did not edit, stage, reset, or otherwise absorb those files.
- The accepted architecture outcome selects `<profile>/lifecycle-logs/entries/<UTC-start>-<128-bit-random-id>/` with `record.json`, `active.lock`, and `log.jsonl`, guarded by one profile `retention.lock`.
- Per-run `agent/`, `sessions/`, `client.lock`, and related state remain under `runs/<run-key>`; the aggregate namespace does not merge session ownership.
- Effective UID is explicitly trusted. Random IDs and strict envelopes defend against accidental collision and scanner overreach, not malicious same-UID mutation.
- Darwin/arm64 is the managed product path. Linux and Windows requirements are compile-safe unsupported behavior, not new runtime implementations.

## Rejected board additions

- No research Task: the architecture outcome already resolves the threat model, deletion authority, recovery semantics, platform boundary, rejected alternatives, and exact adversarial matrix.
- No migration Task: automatic legacy adoption, movement, or deletion is explicitly excluded; legacy evidence is preserved and reported unhealthy.
- No documentation, test, soak, or review Tasks: these are task-local acceptance gates and artifacts, not independently shippable product deliverables.
- No daemon, privileged helper, broker-owner, Session Manager owner, or zero-persistence Task: each is explicitly rejected or out of scope by the accepted architecture decision.

## Dependency and readiness result

No dependency edge is required. The single Task has one coherent deliverable, a Fibonacci estimate of 5, required review, explicit production call sites, negative evidence requirements, platform limits, and a complete validation matrix. A developer can begin from the architecture decision and current-trunk replay resources without an unresolved product or research question.

## Board correction

The Story description, scope, and AC were stale: they named base `675f77ed` and the rejected scan-across-run-directories design. They are updated to exact base `6d051f54...`, the dedicated aggregate namespace, the trusted-same-UID contract, the same-scanner status boundary, and the current adversarial/soak gates.

## Lifecycle verdict

The independent architecture verdict is accepted and the implementation Task is development-ready. The task-specific review precondition requires the Task to remain at `to-dev` after acceptance.

The generic `solution-architect` handoff command cannot represent that route for this historical code Task: it verifies every implementation checklist item and correctly refused while code, tests, lint, build, and soak items remain unchecked. No implementation item was marked complete and no gate was bypassed. The developer must satisfy those items before the eventual code handoff to review.
