# TASK-260826-h934tg — Reviewer verdict (CR-TASK-260826-h934tg-2 revision 2)

- Verdict: **accepted** via `accept_cr(TASK-260826-h934tg, revision=2)`
- Reviewer run: `RUN-260826-bc3e6b`
- Candidate: base `fd80bd8e0c1de3f372fd1a7527613a5135762de4`, tree
  `7cfbbd938587b2054e112440678d6bb317d834b4`
- Prior verdict: `TASK-260826-h934tg_review-verdict.md` — `changes_requested`,
  blocking finding **F1** (new authorization branch had no negative witness).
- Everything below was rerun by this reviewer on the exact revision-2 candidate.
  Nothing is accepted from previously attached evidence.

## Candidate identity — verified

| Check | Result |
| --- | --- |
| Working tree OID (`read-tree HEAD` + `add -A` + `write-tree`, scratch index) | `7cfbbd938587b2054e112440678d6bb317d834b4` — matches declared candidate |
| Patch resource sha256 | recomputed `7506fc58…8a23` == declared == `shasum` of live `git diff base..candidate` |
| Changed paths vs base | exactly the declared 17; nothing under `.task-board/` |
| `git rev-parse main` | `fd80bd8…` — declared base **is** current main |
| `git merge-base --is-ancestor fd80bd8 HEAD` | exit 0 — main is a real ancestor |
| HEAD | `3a52ec7`, unchanged from revision 1; parents `8f81371` (accepted checkpoint) + `fd80bd8` (main) |

Fresh-main ancestry is unchanged by the rework.

## Revision 1 → revision 2 delta — verified minimal

`git diff 02e41e53 7cfbbd93 --stat` is exactly two files:

```
 LOGBOOK.md                                             |  2 +-
 .../internal/infra/pi_standalone_shared_test.go        | 81 ++++++++++++++
```

- `pi_standalone_shared_test.go`: adds only
  `TestRunPiStandaloneNeverInheritsPrimarySessionProjectTrust`.
- `LOGBOOK.md`: rewrites one EVIDENCE line inside entry `1411` to name the
  witness instead of the general suite — precisely what the revision-1 verdict
  asked for.

**Production code is byte-unchanged** from the already-reviewed revision 1: the
delta touches one `_test.go` file and one Markdown file, nothing else. Every
non-F1 finding of the revision-1 review therefore still holds unmodified
(merge correctness, additive documentation, scope containment, O1 observation).

`### ` LOGBOOK entry counts: main 141, rev1 146, rev2 **146**; set difference in
both directions against both main and rev1 is empty — still the union, nothing
dropped. `gofmt -l tools/agents-infra/` is empty.

## The fixture actually composes the hostile policy — verified, not assumed

The witness builds its config as
`validPiProfileWithArgv(...)` → `strings.Replace("[agents.pi.primary_session]\n",
"[agents.pi.primary_session]\nyolo_mode = true\n", 1)` →
`+ standalonePiPolicyTOML("true", ["read","bash","edit","write"])`.

`validPiProfileTOML` (`pi_test.go:44-46`) does emit `[agents.pi.primary_session]\n`,
so the replacement fires rather than silently no-opping, and
`standalonePiPolicyTOML` emits `yolo_mode = true` for the standalone section.
This is confirmed behaviourally, not by reading: under the M8b mutant the child
argv contains `--approve` — an argument that can only exist if primary-session
`yolo_mode = true` was parsed and applied. A no-op fixture could not produce it.

## The production child's approval posture — verified

Clean candidate, real launch through `RunPi`, argv observed by the helper child:

```
--provider local-provider --model Model --thinking medium
--no-approve --no-extensions --tools read,bash,edit,write
--mode json --no-session --print "primary yolo isolation worker"
```

Exactly one `--no-approve`; no `--approve`, `-a`, or `-na`. The assertion loop
counts `--no-approve` (`!= 1` fails) and fatals on any of `--approve`, `-a`,
`-na`, so both the fail-open and the duplicate-posture shapes are bounded.

## F1 closure — independently reproduced

All mutants applied one at a time to a detached scratch worktree pinned to the
exact candidate tree (`git commit-tree 7cfbbd93 -p HEAD`, worktree since
removed; candidate tree OID re-verified `7cfbbd93` after every restore). The
managed Story worktree was never mutated.

| # | Mutant | Revision 1 | Revision 2 |
| --- | --- | --- | --- |
| M8b | `applyPiPrimarySessionYolo` applied in the standalone branch, result threaded into `BuildStandalonePiArguments`, caller-arg validation narrowed to `nil`, caller args prepended to the owned block | **SURVIVED** the whole suite | **killed** |
| M8c | narrower leak: only the short form `-a` prepended when primary yolo is on | not run | **killed** |

M8b failure, verbatim:

```
pi_standalone_shared_test.go:266: standalone worker received non-owned approval
posture "--approve": []string{"--provider","local-provider","--model","Model",
"--thinking","medium","--approve","--no-approve","--no-extensions","--tools",
"read,bash,edit,write","--mode","json","--no-session","--print",
"primary yolo isolation worker"}
--- FAIL: TestRunPiStandaloneNeverInheritsPrimarySessionProjectTrust (0.92s)
```

M8c failure is the same shape with `-a`, so the witness is not spelled to one
literal token.

**The new test is the sole discriminator.** Running the entire `internal/infra`
package under M8b produces exactly one failure:

```
--- FAIL: TestRunPiStandaloneNeverInheritsPrimarySessionProjectTrust (1.22s)
FAIL   …/tools/agents-infra/internal/infra   138.339s
```

That reproduces the revision-1 finding (M8b previously survived the full suite)
and demonstrates the added witness is what closes it, rather than some
pre-existing test that happened to start failing.

Restore check: after each mutant, `git checkout --` on the two named production
paths returned the scratch tree to `7cfbbd93`, and the witness passed again
(`--- PASS … (0.96s)`).

Production call site named and driven: `RunPi` (`pi_launch_posix.go:71-126`) →
`BuildStandalonePiArguments` (`pi_standalone.go:157`). This is the **only**
non-test caller that composes launch argv (`pi_standalone.go:264` passes a
hard-coded `nil` and is the diagnostic path). `argsPlan` is computed once at
`pi_launch_posix.go:122-126`, *before* the shared/exclusive split, and is then
handed to `runSharedPiSession` (line 149) or used directly at line 275. The
exclusive-path witness therefore bounds the argv-composition class for both
standalone paths; a shared-path duplicate would add no discriminating power.

The witness **runs, it does not skip**: `officialPiAsset` resolves inside a real
checkout, and the test reports `--- PASS … (1.67s)` / `(0.96s)` with real work,
never `SKIP`.

## Landing suite — rerun by this reviewer on the exact candidate tree

| Command | Result |
| --- | --- |
| `go vet ./...` | exit 0 |
| `go test ./internal/... -count=1` | exit 0 — attachments 0.669s, infra 115.968s |
| `go test . -count=1` | exit 0 — root 71.206s |
| `GOOS=darwin GOARCH=arm64 go build ./...` | exit 0 |
| `GOOS=darwin GOARCH=amd64 go build ./...` | exit 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | exit 0 |
| `GOOS=linux GOARCH=arm64 go build ./...` | exit 0 |
| `GOOS=windows GOARCH=amd64 go build ./...` | exit 0 |

The configured landing commands (`cd tools/agents-infra && go test ./... -count=1`,
`go vet ./...`) were run as their two constituent package sets to stay inside the
per-call time bound; the union is the configured suite and every part exited 0.

## Producer evidence cross-check

`TASK-260826-h934tg_revision-2-evidence.md` claims no result this reviewer could
not reproduce. Its M8b calibration, restoration hashes, changed-path diff, and
ancestry claim all match independent measurement here. No overclaim found.

## Observation O2 — non-blocking

The fixture's hostile primary-session policy is injected with a `strings.Replace`
on the literal `"[agents.pi.primary_session]\n"`. If `validPiProfileTOML` ever
stops emitting that exact header, the replacement silently becomes a no-op and
the witness degrades to a non-discriminating positive test **without failing**.
A `t.Fatal` when the replacement count is zero (or asserting the parsed policy)
would make that failure loud. Not grounds for rework — the witness discriminates
today, proven by M8b/M8c — recorded so a future fixture refactor does not quietly
reopen F1.

O1 from revision 1 (standalone stdout/stderr inherit a tty via `piProcessWriter`)
is unchanged and remains a declared, non-blocking decision.

## Definition of Done

| Item | State |
| --- | --- |
| Checkpoint accepted implementation before merging | met — `8f81371` is parent 1 |
| Merge mainline, prove main is ancestor of HEAD | met — `--is-ancestor` exit 0 |
| LOGBOOK/overlaps additive, no feature change | met — 146 entries, union preserved; production code byte-unchanged from rev1 |
| Full Go test + vet landing suite on merged tree | met — rerun by reviewer, all green |
| Fresh-base merge evidence + published CR | met — patch sha256 verified against the live diff |
| Negative coverage of gating/authorizing behavior, production call site named | **met — F1 closed**; M8b and M8c both killed, witness is the sole discriminator |
| Lint clean | met — `gofmt -l` empty, `go vet` exit 0 |
| Build not broken | met — 5 cross builds |
| Implementation matches AC | met |
| Solution fits project architecture | met |
| New outcome artifact attached | met — this verdict + `TASK-260826-h934tg_rev2-review-evidence.tar.gz` |

## Routing

`accept_cr(TASK-260826-h934tg, revision=2, evidence=TASK-260826-h934tg_review-verdict-rev2.md)`.
No `commit_ack` supplied — the orchestrator checkpoints or integrates the
accepted revision and makes the `done` transition.
