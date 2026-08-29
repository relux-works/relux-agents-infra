# TASK-260824-2o4zq8: implement-vendor-target-resolution-and-aliases

## Description
Implement TASK-260824-3rl3ws contract Sections 2-7 (R1, R3-R6): canonical target/entrypoint parsing, atomic composition, tuple and Pi-profile validation, decoded composite Pi identity locking, diagnostics, aliases, compatibility, tests, and operator documentation.

## Scope
Go project-config model/resolver; primary-session compose and print output; agents-infra target dispatch; global/local openai-infra, anthropic-infra, qwen-infra setup and verification; focused production-entry negative tests; README and relux-agents-infra skill updates. Do not change direct legacy launcher precedence.

## Acceptance Criteria
Sections 2-7 are implemented at production entrypoints. Canonical aliases resolve configured target identity and provenance; Qwen reuses an existing managed Pi profile and proves the Section 5 qualified-model/profile-provider/endpoint invariants; every Section 7 negative class, including composite model suffix/provider divergence, reasoning-domain bounds, actionable startup error context, and no-config-rewrite/no-side-effect guarantees, fails closed; direct agents-infra codex, claude, pi and pi-infra behavior remains compatible; focused/full Go tests, vet, build, setup/verify, and diff checks pass.
