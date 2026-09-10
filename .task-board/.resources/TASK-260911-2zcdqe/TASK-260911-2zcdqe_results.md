# Refresh probe result

Spawn brief: attempt to refresh CR-TASK-260911-2zcdqe rev 2 (ready, base 67e9c43) onto current trunk 969be6b before review.

Command: task-board worktree refresh-candidate TASK-260911-2zcdqe --json

Exit code: 1

Typed error code: INTERNAL_ERROR
Message: "candidate refresh requires a rework revision; TASK-260911-2zcdqe is ready"

Per brief: refresh-candidate refuses because the revision is ready/pending review. No publish attempted, no product/config/fork/pipx/runtime changes made in this run. The orchestrator should route review of CR rev 2 (base 67e9c43) directly instead of a refresh/republish.
