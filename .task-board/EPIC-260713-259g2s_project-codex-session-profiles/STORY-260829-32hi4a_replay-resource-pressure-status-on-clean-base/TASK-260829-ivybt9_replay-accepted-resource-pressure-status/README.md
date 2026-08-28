# TASK-260829-ivybt9: replay-accepted-resource-pressure-status

## Description
Apply immutable accepted pressure revision 7 patch SHA-256 7e377be3bdbe65516820fcfa39cec620f0ca7afed60d1dcb72d8638410d475f5 to exact current protected base. This replacement lane only; do not mutate historical Task, Stories, worktrees, CRs, move journal, root dirty LOGBOOK, instructions, or unrelated board state.

## Scope
Exactly the 22 paths declared by CR-TASK-260829-1qh0ud-7. Static fixtures and fake providers only; no live model/runtime/daemon/service/endpoint/socket.

## Acceptance Criteria
1. Selected base HEAD local main origin main and GitHub main equal 6d051f54440d36e3ca3d132f8d9d1e78d46289de before replay. 2. Applied patch digest equals 7e377be3 and exactly 22 paths produce candidate tree dabd04a9. 3. Both revision-6 races remain closed and all prior policy/provenance/ownership/restart composition gates pass. 4. Full uncached suite race slice vet format diff and Darwin Linux Windows builds pass. 5. Fresh independent review accepts a new immutable story_final CR. 6. Managed integration completes with no reparent and no historical-lane or root-dirty-state mutation.
