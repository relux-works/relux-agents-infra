# TASK-260829-3fozxa: implement-shared-runtime-log-rotation

## Description
Bound runtime.log growth for unattended multi-day and multi-week shared Pi broker operation.

## Scope
Add byte-capped rotation and deterministic pruning using explicit operator-configured max_segment_bytes and max_segments, with fake-clock and fake-sink tests only.

## Acceptance Criteria
The simulated multi-day runtime log footprint never exceeds max_segment_bytes multiplied by max_segments; rotation triggers at the exact byte cap; old segments prune deterministically without wall-clock sleep; no numeric code defaults.
