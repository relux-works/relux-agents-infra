## Status
done

## Review
required

## Task Class
code

## Blocked By
- (none)

## Blocks
- STORY-260830-2mj14b

## Checklist
(empty)

## Notes
2026-09-11 goal audit: TASK-260830-s5ro4e lockstep release/rollback plan (rev 12) landed on main via PR #43 (merge a9f727e). TASK-260830-11ajl2 identity-only gate (8c7b771) is SUPERSEDED, not stranded: its TestProductionCodeImportsAgentsManagementForIdentityOnly guard fails on current main by design because trunk adopted the generic Pi adapter, engine reader and pkg/localruntime (PRs #29/#31/#35). Remaining scope: remove the hardcoded codex/claude-code/pi literals in project_config.go:411, canonical_target.go:80-98,417-421 and primary_session_launch_plan.go:208-254 so admission and dispatch flow through the agents-management registry; keep the lockstep plan current with the pinned version (v0.5.9); TASK-260831-1pfnxx is cross-repo (agents-management pi plugin still LookPath(agents-infra)).

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-29T22:28:02Z

## Last Update
2026-09-10T18:40:00Z
