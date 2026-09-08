# TASK-260830-1s3elq: register-local-qwen-as-first-class-runtime

## Description
Register local-qwen as a first-class client-executable runtime for project-management, resolved through the versioned status contract rather than through agents-infra internals.

## Scope
(define task scope)

## Acceptance Criteria
project-management resolves and spawns against local-qwen through the published versioned contract only; no code path reads agents-infra internal state to do it. A runtime that is quarantined or not ready is refused with an error naming the observed status and the reason, and the refusal is proven at the production entry point rather than in a fixture. Registration adds no runtime-specific branch to shared spawn code.
