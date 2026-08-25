# TASK-260826-1o2gkq — reviewer verdict: ACCEPTED

Reviewer run `RUN-260826-8e0561` (claude-opus-5/high), Change Request
`CR-TASK-260826-1o2gkq-1` revision 1. All checks below were re-run by the
reviewer inside the managed Story worktree at `HEAD = c40f0c2`,
tree `f92c9b2`. The worktree was left clean and unmodified.

## 1. Why an empty `repository_delta` is the correct outcome here

The CR carries zero paths. That is not a producer failure; it is the only
possible signature of a correct result for this leaf.

This task's deliverable is a **DAG-ancestry property plus a proof of
non-change**. Neither is expressible in a tree diff. The one repository object
the producer authored is the merge commit itself, whose entire value is that
`main` became an ancestor while the accepted product tree stayed byte-identical.

Verified that the emptiness is structural rather than an accident of base
selection: diffing the pre-merge accepted tip against `HEAD` yields 39 differing
paths and **every one of them is under `.task-board/`** — checkout artifacts that
managed CR publication drops. So even with base `91adc73`, the published patch
would still be empty.

The inverse is the failure condition: **a non-empty product patch here would
have meant drift from accepted rev5.** Acceptance is therefore conditioned on
proving non-change, which section 2 and 3 do.

## 2. Ancestry and merge shape — verified

| Check | Result |
| --- | --- |
| `git rev-list --parents -n1 HEAD` | `c40f0c2` has exactly two parents: `91adc73` (accepted rev5 tip) and `e70f953` (current main) |
| `git merge-base --is-ancestor main HEAD` | exit 0 — real merge, not a rebase or squash |
| Accepted rev5 candidate tree | `40a83fe` = `git rev-parse 91adc73^{tree}`, cross-checked against the fcu5pe board record |

## 3. Zero product drift from accepted rev5 — verified independently

- Full-tree `git diff --name-only 40a83fe HEAD` → 39 paths, **0 of them outside
  `.task-board/`**. Run without a pathspec, so an exclusion-glob mistake cannot
  hide a path.
- Recomputed the complete blob manifest myself: `git ls-tree -r` on both trees,
  board and `LOGBOOK.md` filtered out → **124 paths on each side, `cmp` identical.**
  `LOGBOOK.md` is separately identical by the full-tree diff above.

Note: the producer's own manifest covers **35 of those 124 paths** — only the
CR-touched ones. That manifest alone would not have detected drift on an
untouched path. Their separate full-tree `git diff --quiet ... ':(exclude)'`
closes the gap, so overall coverage is complete, but the 35-path figure should
not be read as tree-wide coverage.

## 4. Trunk preservation — the attack the producer's evidence did not make

Tree-identity with rev5 proves no drift *from rev5*, but it would pass equally
if the merge had silently **reverted a trunk-only change** by resolving to the
rev5 blob. The producer asserted the `README.md` / `main.go` conflicts were
"mechanical" without demonstrating it. I tested it directly.

For every non-board path `main` changed since merge-base `b3cb845`:

| Outcome | Count | Paths |
| --- | ---: | --- |
| Byte-identical to main in `HEAD` | 11 | `SKILL.md`, `canonical_target.go`, `canonical_target_pi_test.go`, `canonical_target_pi_main_test.go`, `model_check.go`, `model_check_test.go`, `model_check_docs_test.go`, `model_check_main_test.go`, `pi_platform_windows.go`, `pi_run_report.go`, `pi_operator_docs_test.go` |
| Pure superset (0 deletions vs main) | 5 | `LOGBOOK.md` 168+/0-, `main.go` 103+/0-, `pi_test.go` 88+/0-, `pi_plan.go` 22+/0-, `pi_launch_posix.go` 3+/0- |
| Single-line replacement, inspected | 2 | `README.md` 40+/1-, `pi_config.go` 80+/1- |

Both replacements were read line by line and **extend** the main-side line
rather than drop it:

- `README.md`: the `agents-infra` tool-table row keeps every main-side command
  and output and adds `runtime status` / `runtime stop`.
- `pi_config.go`: `rejectUnknownFields(...)` keeps all main-side field names and
  appends `"sharing"`.

`pi_state.go` shows 24 deletions against main, but **main never touched that
file** since merge-base — those are rev5 refactors of merge-base content
(hardcoded component lists replaced by `statePathComponents(paths)`), already
reviewed under CR rev5.

**Conclusion: no trunk content was silently reverted.**

## 5. LOGBOOK.md additive on both sides — verified

- vs main: `git diff --numstat e70f953 HEAD -- LOGBOOK.md` → `168  0`. Zero
  deletions in an LCS diff means main's LOGBOOK is a **line-subsequence** of the
  merged file — every trunk line survives, in order.
- vs the Story side: byte-identical to accepted rev5 (section 3), so the
  accepted side is preserved by construction.

## 6. `integration_base_moved` is genuinely closed — verified

| Check | Result |
| --- | --- |
| `git merge-tree --write-tree main HEAD` | `f92c9b2` — **exactly HEAD's tree**, exit 0 |
| `git merge-tree --write-tree HEAD main` | `f92c9b2`, exit 0 (order-independent) |
| `git merge-base --is-ancestor main HEAD` | exit 0 → integration is a fast-forward |

Merging the candidate with main reproduces the candidate tree with no conflict.
The structural stale-base cause recorded against the prior revisions is gone.

(Method note: `git merge-tree --write-tree --name-only main HEAD` prints main's
tree `363a2d5`, not a merge result — the flag perturbs argument parsing. Anyone
re-running this check should omit `--name-only`.)

## 7. Gate attacked, not read

Byte-identity of all 124 non-board blobs is itself the decisive attack on *this*
leaf's risk: the merge cannot have weakened a gate it did not touch.

To confirm the accepted negative tests are **live in the merged tree** and not
merely present, I ran a narrowing mutant against the production authorization
comparison in `internal/infra/pi_shared_launcher_darwin.go:73` — widening
`launcher_pid` to also admit `0`:

| Tree | `…ComparesEveryAuthorizationValueAtProductionEntry/launcher_pid_zero` |
| --- | --- |
| Clean | exit 0 |
| Narrowing mutant | **exit 1** on the named witness |

A narrowing mutant, not a delete-only one, so it bounds the class the gate
covers. Driven through the real production entry (`sharedRuntimeExecve` path),
not a helper. File restored byte-identically (blob `ab13a7c`);
`git status --porcelain --untracked-files=all` empty afterwards.

All nine negative subtests of that test passed on **every** run, including the
runs where the positive control failed (section 8).

## 8. Independent validation — and one honest failure

Re-run by the reviewer in the worktree, Go 1.25.5 darwin/arm64:

| Command | Exit | Time |
| --- | ---: | --- |
| `gofmt -l .` | 0 (empty) | — |
| `go vet ./...` | 0 | — |
| `go build ./...` | 0 | — |
| `go test . -count=1` (root, landing gate) | 0 | 139.5s |
| `go test ./internal/attachments -count=1` | 0 | 1.7s |
| focused shared-runtime suite, `-race` | 0 | 56.8s |
| focused shared-runtime suite, no race | **1 pass / 2 fail of 3** | 33.8s pass; 48.7s / 50.7s fail |
| the failing subtest alone, 3 runs | 0 / 0 / 0 | 1.4–2.3s |

**The focused shared-runtime gate is flaky and I reproduced it.** Every failure
is the same subtest —
`TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry/valid`,
`valid authorization never reached execve` — the wall-clock **positive control**
blowing its fixed 15 s budget (`pi_shared_launcher_test.go:23,343`). It passes
in 1.4–2.3 s in isolation and fails only inside the loaded full-suite run; note
the correlation with suite wall time (pass at 33.8 s, fails at 48.7 s / 50.7 s).

This does **not** block acceptance, for two reasons that are provable rather
than assumed:

1. **It cannot have been introduced by this task.** The product and test tree is
   byte-identical to accepted rev5 (section 3), so the exact same bytes flaked
   before this merge.
2. **It is already a known, reviewed, accepted condition** of CR rev3/4/5 —
   recorded in `…_review-verdict-RUN-260826-6697f3.md` §8 ("flakes on a clean
   tree… a pure wall-clock flake, not a kill"), `…-ae2fa5.md` §6.4 ("causes
   `go test ./... -count=1` to fail intermittently on a loaded host for no code
   reason"), `…-c56491.md` §6.1, and recommendation 4 of `…-6b7916.md`.

This task also **cannot** fix it: making the control event-driven would change
accepted rev5 test blobs and violate this task's own byte-identity acceptance
criterion. It must be carried by a task authorized to touch those blobs.

**What is fairly charged to the producer:** reporting `exit 0` for this gate from
a single run, with no note that it is a known-flaky control, is single-sample
evidence for a capability that does not reproduce. The result is not false, but
the confidence attached to it is unearned. Recorded, not blocking.

I re-ran the root and attachments packages myself; I did **not** re-run the full
`go test ./...` infra package (≈397 s, exceeds a single bounded call). I accepted
that leg from the producer's `validation-logs.tar.gz` while independently
characterising the infra package through six targeted runs above.

## 9. Findings carried forward (none blocking)

- **F1 — flaky landing gate (Story/orchestrator).** Replace the 15 s wall-clock
  positive control in `pi_shared_launcher_test.go` with an event-driven wait, or
  retry it before declaring failure. ~2-in-3 failure rate on a loaded host makes
  `go test ./...` unreliable as a landing gate and can mis-score mutants. Needs a
  task authorized to modify accepted rev5 test blobs.
- **F2 — `story_final` CR not published; checklist overstated.** DoD item
  "Publish a new story_final Change Request with fresh-base evidence" is ticked,
  but **no story-scoped CR resource exists** (`.task-board/.resources/` has no
  `STORY-260825-1r7z9o` entry) and the Story records no outcome resources. There
  is no child-side publish verb — `task-board` exposes no `cr`/`publish` command;
  Story-level CR publication happens at Story handoff, which is orchestrator-owned
  and outside a task producer's authority. The producer stated this accurately in
  its board note, so the substance was disclosed and only the tick is overstated.
  **Orchestrator must close this item** — it is the one AC clause this leaf did
  not and structurally could not satisfy.
- **F3 — manifest breadth.** The producer's 35-path blob manifest is not
  tree-wide (124 non-board non-LOGBOOK paths exist). Complete only in combination
  with their full-tree diff. Future base-refresh tasks should assert the full
  tree directly.

## Verdict

**ACCEPTED.** The leaf's actual deliverable — trunk ancestry acquired through a
real merge, zero drift from accepted rev5, additive LOGBOOK, no silently
reverted trunk content, and a fast-forward integration path — is proven, and the
empty repository delta is the correct signature of that result rather than a
missing change. F2 is an open AC clause that only the orchestrator can close;
F1 is inherited, known, and out of this task's authority to fix.
