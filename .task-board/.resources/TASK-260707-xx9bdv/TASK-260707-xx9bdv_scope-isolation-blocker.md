# TASK-260707-xx9bdv Scope-Isolation Blocker

## Required outcome

Publish a new Change Request whose repository delta is empty or contains only
this task's instruction-cleanup scope, while preserving the existing foreign
worktree delta untouched.

## Observed state

- Managed Story worktree: `STORY-260508-1ajnbe`
- Story branch tip: `cf21665dde35274cc14e66e26a93574e0c18c15c`
- Current task Change Request: revision 1, ready, repository delta present,
  eight changed paths
- Foreign Change Request: `TASK-260824-1qm60c` revision 2, accepted,
  repository delta present, eight changed paths
- Foreign task status: `closed`
- Foreign task note: its accepted patch is intentionally not checkpointed in
  this legacy Story and will be replayed through a dedicated delivery Story
- Current foreign worktree patch SHA-256:
  `789172a17237eb91ff6d54d14ee2eb7b4707372ccb6b37046ca9062103790bbf`

The eight foreign paths are:

- `.configs/codex-config.toml`
- `LOGBOOK.md`
- `README.md`
- `SKILL.md`
- `tools/agents-infra/internal/infra/codex_config.go`
- `tools/agents-infra/internal/infra/infra.go`
- `tools/agents-infra/internal/infra/infra_test.go`
- `tools/agents-infra/setup_test.go`

There is no file overlap with the task's source instruction cleanup. The
cleanup already exists in ancestor commit
`d1c8d7d5649c37df394d3401101a9650491b4893`, and the current diff for
`.instructions/INSTRUCTIONS_WORKFLOW.md` is empty.

`LOGBOOK.md` is a workflow overlap: the logbook skill would normally record
this anomaly there, but the file is part of the protected foreign delta.
Writing a new entry would violate the rework requirement to leave that delta
untouched, so the finding is recorded here and in board notes instead.

## Constraint and evidence

Change Request publication snapshots the whole dirty managed Story worktree;
it has no task-path filter. Revision 1 already proves that handing off from
this state captures all eight foreign paths. The active developer run also
holds the Story lease, so another owner cannot reconcile the workspace while
the run remains active.

The owning foreign workflow explicitly excludes checkpointing revision 2 in
this legacy Story. Therefore the clean publication prerequisite cannot be
created inside this developer run without violating ownership or preservation.

## Rejected forced fits

- Publishing again would reproduce the cross-task Change Request.
- Stashing, reverting, resetting, or staging away the foreign delta would
  violate the explicit preservation requirement.
- Committing the foreign delta under this task would misattribute ownership.
- Checkpointing the foreign revision here would contradict its owning task's
  recorded dedicated-Story delivery decision.
- A second ad hoc Git worktree cannot publish a managed Change Request because
  publication is bound to the recorded Story workspace and branch.

## Required orchestration action

Release this run, preserve the foreign Story workspace, and reroute
`TASK-260707-xx9bdv` to a clean managed Story/worktree (or otherwise provide a
board-supported task-scoped publication surface). Then respawn the developer
handoff so task-board can publish the expected empty Change Request.

No repository file was changed by this rework run, and the foreign delta was
left byte-for-byte untouched.
