# TASK-260911-2zcdqe rev3 — REFRESH + REPUBLISH rework

rev2 = rev1 rebased onto trunk 969be6b; only LOGBOOK.md combined; product delta unchanged.

## What this rework did

CR-TASK-260911-2zcdqe rev1 was accepted by the reviewer, but `worktree
integrate` refused with `integration_base_moved`: trunk had advanced twice
since (first STORY-260830-2mj14b landed the engine/knobs axis, moving trunk to
`969be6b`; rev2, which had already handled an earlier trunk move, was withdrawn
by the orchestrator before review could catch up). This run refreshed the
stale rev1 candidate onto the current trunk (`969be6b`) and republished it —
no product code was written or changed beyond what rev1 already had reviewed
and accepted.

## Steps taken

1. `task-board worktree refresh-candidate TASK-260911-2zcdqe` reported a
   regular-file conflict in `tools/agents-infra/internal/modelharness/config.go`
   (trunk added `Engine`/`Model`/`Knobs` fields to `Profile`/`Plan`; the
   candidate independently added `PinnedDistribution` to the same structs —
   both additive, no semantic overlap). Resolved by hand-merging both sets of
   fields/logic into one file.
2. `--replay-resolutions` accepts an undocumented schema (reverse-engineered
   from the binary's own strict-JSON `unknown field` errors, same shape LOGBOOK
   entry 1815/rev2 already found): top-level
   `{version, branch_oid, trunk_oid, candidate_tree_oid, resolutions: [...]}`;
   each resolution is `{checkpoint_oid, path, sha256, content}` where `content`
   is base64-encoded replacement bytes and `sha256` is a plain hex digest of
   those same bytes. Submitted the resolved `config.go` this way;
   `refresh-candidate` returned `{"Outcome":"refresh_advanced", ...}` and the
   Story branch tip advanced to a new commit rebased cleanly onto `969be6b`.
3. **Found and fixed a deeper instance of the known tool regression** (LOGBOOK
   1830): the post-refresh *working tree* silently reverted to the raw
   pre-refresh candidate snapshot for every path the candidate itself had
   staged — not just files it never touched. This dropped trunk's new
   `Engine`/`Model`/`Knobs` fields back out of `config.go` (build succeeded
   but `go test ./internal/modelharness/...` failed immediately:
   `plan.Engine undefined`) and dropped trunk's new "Engine, model and knob
   axes" README section. Also re-reverted `tools/agents-infra/internal/modelharness/engine_knobs.go`,
   its test, and `tools/agents-infra/engine_knob_profile_docs_test.go` (pure
   trunk files, untouched by this candidate) exactly like the LOGBOOK-only
   case in rev2. And dropped LOGBOOK.md back to candidate-only content again,
   losing trunk's newest entry.
4. Fixed all of it by hand in the Story worktree (not the retained replay
   worktree): restored the three engine-knobs files verbatim from trunk
   (`git checkout 969be6b -- <path>`, confirmed candidate never touched them);
   rebuilt `config.go` and the README "Pinned fork distribution" section by
   taking the *correctly rebased commit's* content (`git show <new-tip>:<path>`,
   already combined trunk + this candidate's committed conflict resolution
   with no further issue) and reapplying the candidate's own uncommitted
   increment on top (isolated by diffing the old candidate tree against its
   old base); reinserted the missing LOGBOOK trunk entry.
5. Verified the working candidate is byte-identical to accepted rev1 outside
   trunk-driven additions: for all 8 files in the CR's `changed_paths`,
   diffed the current file against rev1's own target git blob (extracted
   from `TASK-260911-2zcdqe_change-request_rev1.patch`'s `index` lines) —
   `config_test.go`, `pinned_distribution.go`, `pinned_distribution_test.go`,
   `run.go`, `run_test.go` are byte-identical; `config.go` and `README.md`
   differ only by trunk's new engine/knob additions (diffed separately against
   `969be6b`, zero unexpected lines).
6. Ran validation.

## Validation run

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `gofmt -l tools/agents-infra/internal/modelharness/config.go` — clean (no
  output).
- `go test ./internal/modelharness/...` — `ok`, 13.5s.
- `go test ./...` (full `tools/agents-infra` suite, no cache) — all green:
  root package 122.598s, `internal/infra` 221.412s, `internal/attachments`
  and `internal/modelharness` cached ok. Exit code 0.

## LOGBOOK proof (mandatory)

```
$ diff <(git show 969be6b:LOGBOOK.md) LOGBOOK.md | grep -c '^<'
0
```

Zero deletions relative to trunk `969be6b`'s LOGBOOK.md — every trunk entry
(including the "1743 — model-harness Environment Axis" entry this refresh
originally dropped, twice) is present verbatim; the diff is additions-only
(this task's 2026-09-11 entries plus the new 1830 finding above).

## Host state (unchanged from rev1 — NOT redone)

Fork pushed at `06e2f0f` on `origin/task/TASK-260830-2hc5r2-bounded-kv`, pipx
`mlx-lm-relux` reinstalled non-editable from it, live
`/Users/alexis/src/.agents/.configs/model-harness.toml` pin updated,
`~/.local/bin/model-harness` redeployed. All already applied and
independently verified by the accepted rev1 review. This rework touched none
of it and did not restart the shared qwen runtime.
