# TASK-260829-1rinqw: publish-shared-runtime-backoff-deadline

## Description
Add restart_not_before to SharedRuntimeStatus and decide additive half-open and last-failure facts so consumers can distinguish active backoff from a non-zero historical restart count.

## Scope
Shared runtime status wire contract parser docs and static/fake tests; no live runtime access.

## Acceptance Criteria
1. restart_not_before is serialized from the persisted ledger without inference. 2. Serving-after-restart is never mislabeled as active backoff. 3. Half-open and last-failure inclusion or explicit deferral is documented and pinned. 4. Pre/post-extension fixtures and malformed timestamps are covered. 5. Consumer handoff names the exact validator-safe Availability mapping.
