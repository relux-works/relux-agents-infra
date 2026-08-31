# TASK-260828-15ftgj — revision-2 republication verification

Run `RUN-260830-e3d95a`, role `developer`, 2026-08-31.
Workspace `.temp/STORY-260828-2faxgm/worktree`, branch
`task-board/story/STORY-260828-2faxgm`.

**Scope of this run: verification and publication only.** No file inside
`articles/260831_local-qwen-runtime-comparison-study/` or
`.research/260831_local-qwen-runtime-comparison-study.md` was read-modified,
edited, regenerated or re-measured. Every claim below is a read or a comparison.
The orchestrator's hand-off claims were re-established independently rather than
accepted; where a claim was incomplete, that is stated.

---

## 1. The base is current trunk and the branch carries the merge

```
$ git rev-parse --abbrev-ref HEAD
task-board/story/STORY-260828-2faxgm

$ git rev-list --count HEAD..main
0

$ git log --oneline -1
cdcd8f7 Merge branch 'main' into task-board/story/STORY-260828-2faxgm

$ git log -1 --format='%H %P' cdcd8f7
cdcd8f7ae90e83f08bb24740411e2c7b4295af72 f6da7e4bc1ef525cb90919de1b61af133263eb6e bb857fe5c31c1b1fa66f4194d00d7269821c6d76

$ git fetch origin main && git rev-parse origin/main
bb857fe5c31c1b1fa66f4194d00d7269821c6d76

$ git merge-base --is-ancestor origin/main HEAD   # exit 0
$ test "$(git rev-parse cdcd8f7^2)" = "$(git rev-parse origin/main)"   # exit 0
```

`HEAD` is the merge commit. Its second parent **is** the freshly fetched
`origin/main`, so the base is not merely "an ancestor of trunk" — it is exactly
current trunk. The board agrees: `task-board worktree status STORY-260828-2faxgm`
reports `current_base_oid = bb857fe5…`, `branch_tip_oid = cdcd8f7a…`.

The path that refused integration is clean:

```
$ git diff --stat HEAD origin/main -- .configs/codex-config.toml
(empty output — identical)
```

## 2. The article delta is unchanged apart from the LOGBOOK.md merge

The preserved patch was hashed before use:

```
$ shasum -a 256 .temp/TASK-260828-15ftgj/article-delta.patch
136b582fecd51ec0a96df067e3ec30b35c5ca4bbf26fb1786e5b4cff4ae9fa5e   # matches the brief

$ grep -c '^diff --git ' .temp/TASK-260828-15ftgj/article-delta.patch
46
```

The pre-merge candidate was **reconstructed**, not assumed, by replaying the
patch onto the pre-merge commit in a scratch directory outside any repository:

```
$ mkdir -p $S/pre && git archive f6da7e4 | tar -x -C $S/pre
$ cd $S/pre && git apply --binary --whitespace=nowarn <article-delta.patch>   # exit 0
$ find articles -type f | wc -l
42
```

All 46 patch paths were then compared byte-for-byte with `cmp` against the live
worktree:

| Result | Count | Paths |
| --- | ---: | --- |
| byte-identical | 44 | the whole article directory, `.research/260831_…md`, `.research/260829_…md` |
| differ | 2 | `LOGBOOK.md`, `README.md` |

`LOGBOOK.md` is the declared merge conflict (§3). `README.md` was **not**
declared in the brief; it differs because trunk added an
"External-CI local mirror fallback" section that the merge brought in. The
Story's own README contribution survived intact — the 14 changed content lines
in `git diff HEAD -- README.md` are byte-identical to the 14 in the preserved
patch (`cmp` exit 0). So the second difference is a trunk-side addition, not an
article-side loss.

Independent cross-check against the **accepted** revision-1 record rather than
the orchestrator's own patch:

```
$ shasum -a 256 .task-board/.resources/TASK-260828-15ftgj/TASK-260828-15ftgj_change-request_rev1.patch
279ef45a08a1a98ed87474606bed13884766dd8a0b129143aaf04a97c7f6aa64   # matches CR-TASK-260828-15ftgj-1.diff_sha256

$ git archive 3f313d91… | tar -x -C $S2/pre        # CR rev1 base_oid
$ cd $S2/pre && git apply --binary <rev1 patch>    # exit 0
$ diff -r -q $S2/pre/articles/260831_… <worktree>/articles/260831_…   # exit 0
$ cmp $S2/pre/.research/260831_…md <worktree>/.research/260831_…md    # exit 0
$ cmp $S2/pre/.research/260829_…md <worktree>/.research/260829_…md    # exit 0
```

**The article the reviewer accepted and the article now in the worktree are the
same bytes.** Nothing was changed.

The only other files differing between the reviewed rev1 candidate tree and the
current worktree are `.instructions/INSTRUCTIONS_WORKFLOW.md`,
`task-board.config.json` and `tools/agents-infra/internal/infra/infra_test.go`.
Each was confirmed to equal **trunk's** version exactly
(`git show bb857fe:<path>` hash == worktree hash), i.e. all three are the merge
delivering trunk, not the Story mutating anything.

## 3. Both pre-merge LOGBOOK.md sides survive

Both sides were reconstructed from git rather than read from the brief:
trunk side `git show bb857fe:LOGBOOK.md`; Story side = the patch-reconstructed
`$S/pre/LOGBOOK.md`; merge base `b78498bf98c0…` (`git merge-base f6da7e4 bb857fe`).

Survival was checked as a **multiset** of non-blank lines, so a line dropped
from two occurrences to one still registers as missing.

| Side | non-blank lines added vs merge base | missing from merged LOGBOOK.md |
| --- | ---: | ---: |
| Story working tree | 249 | **0** |
| trunk (`bb857fe`) | 13 | **0** |

Whole-file multiset check, not just the added lines:

| Side | total non-blank | missing from merged |
| --- | ---: | ---: |
| Story working tree | 1388 | **0** |
| trunk | 1153 | 1 |

That single line is the heading `## 2026-08-27`. It is **not** a merge loss: the
merge base already carried that heading **twice** (`grep -c '^## 2026-08-27$'`
→ base 2, trunk 2, Story side 1, merged 1). The Story side deduplicated it
before the merge, inside the delta that review already accepted. The merge
preserved the Story side's choice. No log content line is affected.

Merged file: 1672 lines, 1401 non-blank, resolved additively.

## 4. Article shape

```
$ wc -l < .research/260831_local-qwen-runtime-comparison-study.md
1103
$ find articles/260831_local-qwen-runtime-comparison-study -type f | wc -l
42
$ ls articles/260831_local-qwen-runtime-comparison-study
ARTICLE.md  README.md  SHA256SUMS  artifacts  reproduce.zsh
$ cmp articles/260831_…/ARTICLE.md .research/260831_…md     # exit 0
```

Both stated numbers hold: **1103 lines**, **42 files**.

## 5. The article still verifies at the merged base

```
$ cd articles/260831_local-qwen-runtime-comparison-study && shasum -a 256 -c SHA256SUMS
exit=0   (41 files OK; the 42nd file is SHA256SUMS itself)

$ ./reproduce.zsh
exit=0   PASS: local Qwen runtime comparison study reproduced
```

`reproduce.zsh` re-derives every cited figure from the sealed records and
re-asserts the structural claims the decision rests on (campaign exit 4 on
`contextPolicy`, no `decision.json` for the llama.cpp pair, the memory axis
producing no comparison, no positive break-even crossover on either
decode-admissible scenario, `context_75k` excluded from every decode claim, the
MLX Swift rejection resting on the 1.1512x 8k peak-footprint ratio). It re-runs
no measurement.

## 6. Configured validation suite, run at the merged base

`task-board.config.json → spawn.worktree_isolation.validation.commands`:

```
$ cd tools/agents-infra && go vet ./...
exit=0
$ cd tools/agents-infra && go test ./... -count=1
exit=0   (4 packages ok, 0 FAIL)
```

Run as standalone processes; no `tee`, no pipe chain. Exit codes are the real
ones.

## 7. Working tree at publication

```
$ git status --short
 M .research/260829_llamacpp-against-the-python-baseline.md
 M LOGBOOK.md
 M README.md
?? .research/260831_local-qwen-runtime-comparison-study.md
?? articles/

$ git add -N -- . && git diff --stat HEAD | tail -1
46 files changed, 9878 insertions(+)
```

46 paths, insertions only — the same count captured in the preserved patch.

## 8. Notes and non-findings

- No test was added or changed by this run, because no behavior was changed.
  The gate-negative-evidence obligations are unchanged from revision 1, whose
  mutant campaign against the article's own verifier is attached as
  `TASK-260828-15ftgj_mutant-campaign.md`.
- `.ruff_cache/` sits at the worktree root and is not named in `.gitignore`.
  It is nonetheless invisible to git because ruff writes a self-ignoring
  `.gitignore` inside it, so it cannot enter the Change Request snapshot. Left
  as-is: adding a `.gitignore` entry is a repository change outside this run's
  scope. Flagged for whoever owns repo hygiene.
- The brief's "112 non-blank lines from the Story side and 63 from trunk"
  could not be reproduced under any counting rule tried here; the independent
  multiset counts are 249 and 13 added-vs-base. The *conclusion* the brief drew
  — zero missing on both sides — is confirmed, and confirmed more strictly than
  the added-line framing, by the whole-file multiset check in §3.
