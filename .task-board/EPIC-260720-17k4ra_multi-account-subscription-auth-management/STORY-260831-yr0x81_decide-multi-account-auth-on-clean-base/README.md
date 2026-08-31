# STORY-260831-yr0x81: decide-multi-account-auth-on-clean-base

## Description
Carries the remaining multi-account auth work away from STORY-260720-161knz, where the completed TASK-260720-3moaky still holds an unaccepted Change Request that refuses every sibling producer (BUG-260830-1dzkcu, third variant). The isolation audit itself is done and its findings stand; only the container is being changed.

## Scope
(define story scope)

## Acceptance Criteria
Keychain custody and refresh semantics are researched per provider, the extensible auth-method lifecycle is designed, and an ADR records the decision with its evidence, assumptions, unknowns, per-provider feasibility verdict, compared options and security consequences. No credential value is ever printed, exported or persisted, and no live authenticated session on this machine is logged out, revoked or rotated to obtain evidence.
