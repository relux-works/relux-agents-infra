# TASK-260824-293a0x: land-reviewed-fast-profile-migration

## Description
Replay the accepted fast-profile removal and setup-preservation migration from CR-TASK-260824-1qm60c-2 in the dedicated delivery Story.

## Scope
Apply the exact accepted patch resource TASK-260824-1qm60c_change-request_rev2.patch from TASK-260824-1qm60c, verify the resulting eight changed paths and candidate tree, rerun the configured Go test/vet gate, and publish the final Story Change Request without unrelated edits.

## Acceptance Criteria
The candidate matches accepted CR revision 2 semantically and remains limited to the eight reviewed paths; tree-bound Go tests and vet pass; reviewer acceptance cites the prior rev2 verdict and new Story-final candidate; worktree integrate lands the source change on main.
