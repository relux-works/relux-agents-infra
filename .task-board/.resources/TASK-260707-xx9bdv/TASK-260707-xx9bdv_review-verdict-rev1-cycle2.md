# TASK-260707-xx9bdv Review Verdict — CR Revision 1, Cycle 2

Verdict: **changes requested** (`to-dev`)

## Finding

### F1 — The Change Request snapshots another task's repository delta

The exact handed candidate (`cf21665dde35274cc14e66e26a93574e0c18c15c`
to `d20a1a815ef13fd778254bbd2eeb0fc02b28586a`) changes eight paths:

- `.configs/codex-config.toml`
- `LOGBOOK.md`
- `README.md`
- `SKILL.md`
- `tools/agents-infra/internal/infra/codex_config.go`
- `tools/agents-infra/internal/infra/infra.go`
- `tools/agents-infra/internal/infra/infra_test.go`
- `tools/agents-infra/setup_test.go`

Those paths implement the Codex fast-profile removal and managed-config
migration from the already accepted but uncheckpointed
`TASK-260824-1qm60c` revision 2. The current candidate adds two later review
entries to that task's `LOGBOOK.md` delta, but it contains no task-specific
instruction cleanup change. In particular:

- `git diff cf21665d d20a1a8 -- .instructions/INSTRUCTIONS_WORKFLOW.md` is empty.
- Cleanup commit `d1c8d7d5649c37df394d3401101a9650491b4893` is already an ancestor of both the
  CR base and current `main`.
- The producer outcome explicitly says this rework introduced no tracked
  worktree changes and left pre-existing unrelated Story changes untouched.

Change Request publication snapshots the whole dirty Story worktree. Accepting
this revision would therefore attest and later checkpoint another task's code
under `TASK-260707-xx9bdv`. This is the negative shape **bypass path around the
check**: Story worktree isolation exists, but the uncheckpointed cross-task
dirty state bypasses task-scope isolation at the protected `accept_cr` /
`worktree checkpoint` path.

Required rework: reconcile the stale Story-worktree delta through its owning
workflow, then publish a new revision whose `repository_delta` is empty (the
expected shape for this artifact/runtime-only rework) or contains only a real
`TASK-260707-xx9bdv` repository change. Do not discard unrelated work ad hoc.

## Functional Evidence That Passed

- The self-contained archive SHA-256 is
  `2611265e4fb08d9c5e4707235106269a484350bc75ae5567b687c211c91a0d4c`.
- All 78 internal `SHA256SUMS` entries verify.
- The four previously missing project-local instruction files are present and
  byte-identical to the recoverable originals under
  `/Users/alexis/src/x-platform-airdrop/.agents/.instructions`.
- Strict and separator/case-flexible searches found no project aliases across
  source instructions, installed instructions, rendered Codex instructions,
  or rendered Claude instructions.
- Source and installed instruction basename sets are identical.
- Fresh `agents-infra verify global` and `agents-infra doctor global` both exit
  0.
- The CR's attached tree-bound validation reports uncached `go test ./...` and
  `go vet ./...` exit 0. This validates the candidate but does not cure its
  cross-task scope.

## Review Boundary

No repository files were modified by this reviewer, no `commit_ack` was
supplied, and `accept_cr` was intentionally not called.
