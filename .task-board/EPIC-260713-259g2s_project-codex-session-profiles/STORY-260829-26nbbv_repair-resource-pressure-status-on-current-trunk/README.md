# STORY-260829-26nbbv: repair-resource-pressure-status-on-current-trunk

## Description
Replay resource-pressure status on current infra trunk and fix the review-proven status/admission contradiction.

## Scope
Compose restart_not_before/half_open with resource status from exact 675f77ed; make status publication coherent with pressure latch, broker state, eviction timer, and subsequent acquisition. Static/fake fixtures only.

## Acceptance Criteria
Fresh Story base equals 675f77ed; revision-1 resource contract is preserved; status never reports healthy/admitted while enforced latch remains pressured and eviction is armed; deterministic production-handler negative covers latch, state, timer and subsequent acquisition; RestartNotBefore, HalfOpen and Resources all survive composition; full gates and independent review pass.
