# TASK-260720-3gcfd1: decide-multi-account-auth-architecture

## Description
Produce a feasibility ADR for multi-account auth management. Compare agents-infra-owned Keychain credentials, agents-infra-managed opaque provider profiles, and provider-native switching/delegation. Select a per-provider model only where evidence supports it; explicitly allow hybrid or no-go outcomes.

## Scope
Architecture decision only after provider-boundary and Keychain-custody research; no implementation and no real credential handling.

## Acceptance Criteria
ADR records evidence, assumptions, unknowns, per-provider feasibility verdict, compared options, security/concurrency/refresh/revocation tradeoffs, recommended architecture or no-go, proof-of-concept gates, CLI UX only for viable paths, and implementation work breakdown that remains unstarted.
