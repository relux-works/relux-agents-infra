## Status
done

## Review
light

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Capture sanitized Pi, broker, MLX, memory, and session-timeline evidence without interrupting the live run
- [x] Identify the exact timeout owner and verify effective 75000/50000/8192 threshold rendering
- [x] Implement bounded compaction and exact-session recovery behavior at source
- [x] Run focused regression checks plus synthetic capacity and restore validation
- [x] Record operator monitoring and recovery guidance

## Notes
2026-08-27: Live session left untouched. Root cause is the upstream qwen3_5 ArraysCache Metal buffer-object leak in mlx-lm 0.31.3, followed by the known dead-generation-thread HTTP wedge; Pi timeout is secondary. Research evidence: incident-root-cause.md.
Implemented and committed as d91d6fc. Installed source and patched runtime verified. No subagents or spawned reviewers were used per user direction; validation and review were performed inline.

## Precondition Resources
- [incident-root-cause.md](file://BUG-260827-1jhv2g/incident-root-cause.md) — Sanitized local timeline and upstream root-cause mapping

## Outcome Resources
- [implementation-and-validation.md](file://BUG-260827-1jhv2g/implementation-and-validation.md) — Patched runtime, supervisor, restore, and 50k stress evidence

## Created
2026-08-27T06:27:11Z

## Last Update
2026-08-27T06:59:49Z

## Assigned To
codex
