# BUG-260830-1rabvi: no-managed-worktree-refresh-for-moved-base

## Description
A managed Story worktree whose base falls behind trunk has no in-contract way forward. The specialist assignment forbids switching, rebasing, or merging the managed Story branch, and no public 'task-board worktree refresh' command exists; 'worktree abort' releases the lease and preserves the worktree but does not reprovision it at current trunk. Observed twice on TASK-260830-2hc5r2: the producer correctly filed a blocker rather than violating the contract. Evidence: worktree tip 3295c7d vs main 436760d, HEAD..main=2, main..HEAD=0. The orchestrator resolved it manually with stash / merge --ff-only / stash pop / additive conflict resolution, which is deterministic precisely because every managed Story branch carries ahead=0 (deltas are uncommitted working-tree state, not commits). That manual procedure is the missing operation.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
A managed Story worktree whose base is strictly behind trunk can be reprovisioned at current trunk through a public board operation, preserving the uncommitted delta byte-identically and reporting any content conflict explicitly rather than resolving it silently. The operation refuses when the branch carries commits of its own. Specialists remain forbidden from moving the base.
