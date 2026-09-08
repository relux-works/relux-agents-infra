# TASK-260829-1qh0ud: publish-shared-runtime-resource-pressure-status

## Description
Expose bounded typed loaded-model memory-pressure and inference-busy facts with explicit unknown semantics and drain or eviction policy.

## Scope
Infra observation/status and deterministic fake-provider tests only; do not probe or mutate a user-owned live model.

## Acceptance Criteria
1. Status distinguishes observable healthy busy pressured draining and unknown facts without guessing. 2. Thresholds and eviction/drain sequencing are explicit configuration with no unsafe defaults. 3. Refusal and recovery after pressure clears are deterministic. 4. Lease ownership is preserved and no duplicate runtime is started under pressure. 5. Wire fixtures and consumer handoff are versioned.
