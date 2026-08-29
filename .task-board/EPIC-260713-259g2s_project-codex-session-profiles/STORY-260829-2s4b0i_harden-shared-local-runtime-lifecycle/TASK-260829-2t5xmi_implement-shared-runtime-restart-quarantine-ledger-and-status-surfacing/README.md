# TASK-260829-2t5xmi: implement-shared-runtime-restart-quarantine-ledger-and-status-surfacing

## Description
Add a persisted cross-broker restart and quarantine ledger, surface lifecycle facts in SharedRuntimeStatus JSON, and close the two real production-seam lease-release gaps.

## Scope
Resolve paths through the existing shared-runtime path contract; implement explicit operator-configured backoff and quarantine; extend existing broker admission and runSharedPiSession subprocess fixtures; use no live model.

## Acceptance Criteria
The ledger survives broker restart; status JSON exposes restart_count, quarantined_until, and last_readiness_match; bounded exponential backoff, stable-run reset, automatic-half-open and manual quarantine are deterministic; real handleConnection and runSharedPiSession release leases after abrupt client death; no numeric code defaults.
