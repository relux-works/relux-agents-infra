# TASK-260911-2zcdqe — rev2 (refresh + republish)

rev2 = rev1 rebased onto trunk 67e9c43; only LOGBOOK.md combined; product delta unchanged.

## What happened

CR-TASK-260911-2zcdqe rev1 was accepted by the reviewer
(`TASK-260911-2zcdqe_review-verdict-rev1.md`), but `worktree integrate` refused
with `integration_base_moved`: trunk had advanced past the recorded checkpoint
(STORY-260830-37bq03 and STORY-260830-112ewt landed, new trunk `67e9c43`), and
trunk's own LOGBOOK.md change overlapped with this CR's LOGBOOK.md change under
the same `## 2026-09-11` header. No product-code conflict was expected or found.

## Steps taken

1. Ran `task-board worktree refresh-candidate TASK-260911-2zcdqe`. It refused
   with a regular-file conflict on LOGBOOK.md and retained an isolated replay
   worktree at `.temp/base-refresh/replay-3305295578/worktree` plus a
   `resolution-template.json`.
2. Combined the two sides of the LOGBOOK.md conflict by hand in that retained
   worktree: kept every trunk entry (1730 "Doctor Now Names Every Missing
   Profile Field In One Run", 1345 "Environment Admission Has a Coincidental
   Second Gate", 1631 "Readiness Test Flake Had A Third, Undocumented Race")
   AND every candidate entry (1640, 1710, both from TASK-260911-1diuhg),
   newest-first under the shared date header, with no leftover conflict-marker
   lines.
3. Reverse-engineered the undocumented `--replay-resolutions` JSON schema by
   iterating on the tool's own strict-JSON error messages (`unknown field`,
   then semantic errors), since neither `--help` nor the repo (this is a
   separate closed-source binary, not vendored here) documents the resolution
   object's fields. Working schema for `changerequest.ReplayResolutions`:
   `{version, branch_oid, trunk_oid, candidate_tree_oid, resolutions: [{checkpoint_oid, path, sha256, content}]}`
   where `content` is the **base64-encoded** replacement file bytes and
   `sha256` is the plain SHA-256 hex digest of those same raw bytes (a
   consistency binding, not a lookup key — supplying only a hash with no
   `content` field produces `replay resolution content digest mismatch`
   against an empty/zero-value comparison).
4. Re-ran `refresh-candidate --replay-resolutions ...` with that resolution.
   It returned `{"Outcome":"refresh_advanced", ...}` and advanced the Story
   branch (new tip `24a3901`, rebased onto `67e9c43`).
5. **Found and fixed a real regression from step 4.** After the refresh, the
   working tree's LOGBOOK.md had silently reverted to the *candidate-only*
   version (1730-recovery/1640/1710) — the three trunk entries I had combined
   into the retained replay worktree were gone. Separately, seven
   `tools/agents-infra/internal/infra/*` files and `main.go` (none touched by
   this task or by rev1) showed as modified/deleted in the working tree,
   reverting trunk's STORY-260830-37bq03/112ewt work
   (`project_config_field_gaps.go` and its test were deleted outright), and 16
   files under the worktree's local `.task-board/` checkout artifact also
   reverted. None of this belongs to this task's scope. I hand-fixed
   LOGBOOK.md again directly in the Story worktree (all six entries, verified
   zero deletions against trunk — see proof below) and ran
   `git checkout -- <path>` for every one of the out-of-scope infra/main.go
   files and `.task-board/` to restore them to match the new HEAD exactly.
   Root cause is presumably that `refresh-candidate` reconstructs the working
   tree from a `candidate_tree_oid` snapshot captured before trunk's new
   commits landed, and that snapshot silently overrides files it didn't need
   a resolution for; this is worth its own defect report against the
   task-board tool, filed separately from this task's scope.
6. Verified the working candidate is otherwise byte-identical to accepted
   rev1 outside LOGBOOK.md: `git diff HEAD` now shows only `LOGBOOK.md` and
   `README.md` changed (plus `.task-board/` activity-ledger churn from the
   board mutations this run itself performed, which is expected and excluded
   per the Story worktree contract — it is a checkout artifact, not the
   board). README.md's final content matches rev1's intent exactly (pin
   `commit = "06e2f0f..."`, corrected retirement condition naming upstream
   PR #1513, "Known gap"/"Operator follow-up" stale sections removed — all
   already landed in the committed `d5e12bd`/`24a3901` history, so the
   remaining uncommitted README delta is small and matches rev1's tail).
7. Ran validation: `go build ./...` (clean) and
   `go test -count=1 ./internal/modelharness/...` (`ok`, 13.198s, no cache).

## LOGBOOK proof (mandatory)

```
$ diff <(git show 67e9c43:LOGBOOK.md) LOGBOOK.md | grep -c '^<'
0
```

Zero deletions relative to trunk `67e9c43`'s LOGBOOK.md — every trunk entry is
still present verbatim; the diff is additions-only (candidate's three
2026-09-11 entries).

## Host state (unchanged from rev1 — NOT redone)

Fork pushed at `06e2f0f` on `origin/task/TASK-260830-2hc5r2-bounded-kv`, pipx
`mlx-lm-relux` reinstalled non-editable from it, live
`/Users/alexis/src/.agents/.configs/model-harness.toml` pin updated,
`~/.local/bin/model-harness` redeployed. All of this was already applied and
verified by the accepted rev1 run; this rework did not touch any of it and did
not restart the shared qwen runtime.
