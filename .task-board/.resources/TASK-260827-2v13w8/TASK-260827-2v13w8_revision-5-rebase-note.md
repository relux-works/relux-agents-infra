# TASK-260827-2v13w8 — revision 5 refresh note

**The only delta from revision 4 is the trunk refresh and the LOGBOOK.md merge.**
No source file, no script, no example, no test, no measurement, no attestation
and no artifact changed. Review can be scoped to exactly that.

## Why this revision exists

Revision 4 was accepted. Integration then refused with `integration_base_moved`:
trunk advanced by `323c482` ("Record upstream generation-health publication"),
which touches `LOGBOOK.md`, and this Change Request also touches `LOGBOOK.md`.
Path intersection demotes an accepted revision to `stale` because nobody has
reviewed the combination. That is the guard behaving correctly.

## What was done

The Story branch was refreshed onto current trunk `323c482` by **merging trunk
into the branch** (`24a2dac`), not by rebasing it. See "Why a merge and not a
rebase" below — a rebase was tried first, worked, and was then deliberately
undone.

One conflict, in `LOGBOOK.md`, at the hunk directly under the `## 2026-08-27`
heading. Resolved by keeping **both** sides, not by choosing one: trunk's
`1229 — Generation Health Contribution Published` entry is retained verbatim,
and every entry this Story added is retained verbatim. The trunk entry was
placed after `1452` and before `0144`, which is where it sits relative to `0144`
on trunk and keeps the section's newest-first ordering intact.

| Fact | Before | After |
| --- | --- | --- |
| branch tip | `9e4c53a` | `24a2dac` (merge commit) |
| `merge-base(branch, main)` — the `story_final` base | `3f313d9` | `323c482` |
| commits behind trunk | 1 | 0 |
| recorded workspace checkpoint `9e4c53a` reachable from tip | yes | **yes** |

The task's own work was carried across as a temporary commit and restored
afterwards, so it is uncommitted in the worktree exactly as it was for
revision 4.

## Proof that nothing else moved

The revision-4 candidate content and the revision-5 candidate content were each
committed as a temporary snapshot and diffed against each other:

```
$ git diff --name-only refs/task-board/rev4-prerebase-candidate refs/task-board/rev5-candidate-content
LOGBOOK.md
```

One path. Not "a small diff" — one path.

| Object | revision 4 | revision 5 |
| --- | --- | --- |
| candidate snapshot commit | `b06d1a2` | `1bc5f80` |
| candidate tree | `7f65667945a8087e883e6c82eb9fc8b402cce917` | `0d403a3dc883ac2978011dd29c3c9190ddc250e6` |
| `LOGBOOK.md` blob | `468b015e723e0ecd59112318e0232401e8bf547d` | `a4ddd1f514f742f2ed399f4b34db66862553c844` |

Both snapshots are kept reachable as `refs/task-board/rev4-prerebase-candidate`
and `refs/task-board/rev5-candidate-content` in the Story worktree, so the diff
above is reproducible rather than asserted. The `LOGBOOK.md` delta between those
two trees is a pure six-line insertion of trunk's `1229` entry — no deletion, no
reordering, no reflow of any Story entry.

## Why a merge and not a rebase

The rebase was performed first and it was clean: the three checkpointed leaves
replayed as `a08ad38 / e21f514 / 645bf12`, one LOGBOOK conflict, resolved the
same way. It was then **undone**, because `task-board worktree status` reported
a blocker that the merge does not create:

```
blocked:    managed branch no longer contains the recorded checkpoint
```

The workspace record `.temp/worktrees/STORY-260827-m30k8z.json` carries
`checkpoint_oid: 9e4c53a…`, and rebasing rewrote that commit. Adopting an
existing workspace goes through `assertCheckpointReachable`
(`internal/worktree/manager.go:580`), which refuses a branch whose history no
longer contains the recorded checkpoint with `ErrorWorkspaceStale` and states
that "no automatic reset repairs it". The reviewer spawn for this revision
adopts exactly this workspace, so a rebased branch would have blocked review
rather than unblocked integration. Merging reaches the same `story_final` base —
`StoryFinalBase` is `merge-base(branch, trunk)`
(`internal/changerequest/basis.go:32`), which is `323c482` either way — while
leaving `9e4c53a` an ancestor of the tip. Integration is unaffected by the
branch's shape: it builds one commit from the recorded candidate tree with trunk
as its only parent (`internal/integration/integrate.go:264`).

**Tool-contract finding, recorded rather than worked around** (see the logbook
entry of the same date): task-board's own pre-producer base refresh has this
same shape. `refreshSpawnStoryBaseForFinalLeaf`
(`cmd/spawn_workspace.go:208`) rebases the branch via `changerequest.Refresh`
and updates only `CurrentBaseOID`; nothing updates `CheckpointOID`. A successful
built-in `refresh_advanced` on a Story that has already checkpointed a leaf
should therefore leave the same unreachable checkpoint behind. That belongs to
the task-board source repo, not to this one, and was not touched here — this
revision's brief was explicitly "change nothing else".

## Validation actually run at this tree

Run by me in this worktree after the refresh, each command as a standalone
process, real exit status reported:

| Command | Exit |
| --- | ---: |
| `cd tools/agents-infra && go vet ./...` | 0 |
| `cd tools/agents-infra && go test ./... -count=1` | 0 |
| `git diff --check` | 0 |

That is the configured landing suite
(`spawn.worktree_isolation.validation.commands`) in full. `go test` result
lines: `agents-infra` ok 105.466s; `internal/attachments` ok 2.760s;
`internal/infra` ok 186.695s; `internal/modelharness` ok 2.004s;
`cmd/model-harness` no test files. It was run twice — once on the rebased tree
and once again on the merged tree, both exit 0 — so the branch-shape change
carries its own evidence rather than inheriting it.

## What was deliberately NOT re-run

`swift build`, `swift test`, `xcodebuild` Release, `swift-format lint --strict`,
`shellcheck`, `benchmark-gate-smoke.sh`, `lifecycle-smoke.sh`, the
`benchmark-compare` replay, and above all the hour-long `benchmark-run`
measurement pass. Stated plainly rather than implied: **their evidence is
revision 4's and is not re-established here.** The justification is the one-path
diff above — every Swift source, script, example, threshold, prompt and test
those gates read is byte-identical, and the single changed file is a Markdown
logbook that no gate compiles, lints or executes. Re-measuring would also have
violated the refresh brief's requirement that the measurements, attestations and
binary identity remain byte-identical accepted evidence.

## Decision

Unchanged. **REJECT** stands: Python `mlx-lm` remains the default local Qwen
runtime, `benchmark-compare` replay exits 3, and the surviving blocker is the 8k
scenario-local peak footprint at 1.151x against a 1.10 bar. Nothing in this
revision touches that finding, and `profiles.qwen-local` still points at
`mlx_lm.server`.
