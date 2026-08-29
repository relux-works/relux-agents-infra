# STORY-260829-2s4b0i: harden-shared-local-runtime-lifecycle

## Description
Make the shared Pi local-model runtime externally observable and safe for unattended multi-week operation without interacting with a live model.

## Scope
Persisted restart and quarantine lifecycle state, status JSON surfacing, production-seam lease-release regression tests, and bounded runtime log retention in relux-agents-infra.

## Acceptance Criteria
External consumers can distinguish restart backoff and quarantine from ordinary unavailability; runtime logs have a deterministic bounded footprint; all validation uses fakes and subprocess fixtures with no live model.
