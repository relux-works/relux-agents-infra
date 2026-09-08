# TASK-260830-mf1xwk: migrate-task-board-onto-generalized-plugin-graph

## Description
Carry task-board across the generalized plugin graph in lockstep with skill-agents-management, since the board is already a heavy consumer of the two-layer contract and is also the tool executing this migration.

## Scope
skill-project-management as the consumer and skill-agents-management as the contract owner. The board is the tool running this migration, so continuity of its spawn and review routing is a delivery constraint, not a nice-to-have.

## Acceptance Criteria
task-board runs on the generalized plugin graph with its spawn plane, runtime registry, reasoning vocabularies, capability ranking, provider-limit detection and availability all working as before, proven by its own suite and by a real spawn on this board, and at no point during the migration is the board left unable to spawn or route work.
