# BUG-260830-1dzkcu: mutual-sibling-change-request-deadlock

## Description
Two sibling tasks in one Story can each hold an unaccepted Change Request that blocks the other's producer, with no in-contract exit. Observed three times, each variant sharper than the last. First: TASK-260830-2hc5r2 and TASK-260829-3k4qrc deadlocked until the orchestrator reparented one to a new Story. Second: TASK-260829-3k4qrc received a changes-requested verdict, yet CR-TASK-260829-3k4qrc-1 stayed in state 'ready' and refused any new sibling producer with change_request_sibling_producer_blocked. Third, and the sharpest: TASK-260720-3moaky is status 'done' and still holds CR-TASK-260720-3moaky-1 revision 1 in state 'ready', which refuses producer TASK-260720-1g880w in Story STORY-260720-161knz. A completed task should not be able to block a sibling with a Change Request nobody will ever act on, because the element that owns it is finished and will not be reworked. 'task-board worktree' exposes abort, checkpoint, gc, integrate, repair, status and transaction; none withdraws or resolves a Change Request left behind by a done element.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
A Change Request stops blocking sibling producers once it can no longer be acted upon: when its review requested changes, and when its owning element reaches done or closed. The board either resolves such a Change Request automatically at that transition or exposes an operation that withdraws it, and refuses to move an element to done while it holds a Change Request that would outlive it. The board detects a mutual sibling block, reports it with both element IDs and the blocking edge, and documents the supported exit.
