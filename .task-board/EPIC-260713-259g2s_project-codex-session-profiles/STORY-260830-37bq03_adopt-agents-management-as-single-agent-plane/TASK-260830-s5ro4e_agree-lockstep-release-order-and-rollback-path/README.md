# TASK-260830-s5ro4e: agree-lockstep-release-order-and-rollback-path

## Description
task-board is simultaneously a dense consumer of the agents-management plugin contract and the tool through which this migration is executed. A breaking contract change can therefore leave the board unable to spawn or route the very work that performs the migration. Produce the written release order and rollback path before any breaking change lands, covering: which repository releases in which order, which versions are compatible with which, what the board must still be able to do at every intermediate step, and how to return to a working board from each step if it fails.

## Scope
(define task scope)

## Acceptance Criteria
A written plan names, for every step in the order: the repository released, the version, the exact board capabilities that must remain working at that point (at minimum spawn and route), and the concrete command sequence to roll back to the previous working state. The plan identifies every step at which the board would be running against a contract version it does not yet support, and states how that window is avoided or survived. A step whose rollback is untested is marked as such rather than assumed. The plan is reviewed and accepted before the first breaking change lands.
