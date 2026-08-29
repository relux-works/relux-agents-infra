# BUG-260827-1jhv2g: qwen-pi-compaction-request-timeouts

## Description
A long-lived local Qwen Pi session repeatedly times out while summarizing near the compaction threshold; overflow recovery can fail after two compactions, while the same persisted session may continue after another launch.

## Scope
Diagnose Pi session metadata, generated thresholds, client timeout behavior, MLX request timing, prompt-cache memory, runtime handoff, and exact-session restore. Preserve the live session and transcript. Implement the smallest source fix needed for reliable multi-week operation and verify without spawning agents.

## Acceptance Criteria
The incident has timestamped evidence identifying the failing layer; generated Pi configuration uses context 75000 and compact-at 50000; exact-session restore is deterministic; compaction/recovery no longer enters repeated opaque timeouts under the reproduced load or reports a bounded actionable failure; runtime and prompt-cache memory remain bounded; focused regression checks and operator recovery guidance pass.
