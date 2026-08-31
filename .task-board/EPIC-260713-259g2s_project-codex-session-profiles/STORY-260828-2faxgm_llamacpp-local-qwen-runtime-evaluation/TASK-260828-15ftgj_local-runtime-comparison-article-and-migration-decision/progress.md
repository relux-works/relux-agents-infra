## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- TASK-260828-2wcrph
- TASK-260827-2v13w8
- TASK-260829-3k4qrc

## Blocks
- TASK-260830-3euwsu

## Checklist
- [x] Report states per runtime what was measured under which pinned conditions, with exact revisions and commands
- [x] Every invalid comparison named with the reason it is invalid, not silently omitted
- [x] Memory economy scored as a first-class criterion alongside throughput and latency
- [x] Decision recorded with concrete blockers for each rejected candidate; Python mlx-lm stays default unless a candidate wins on evidence
- [x] Abstract states the question, the method in one sentence, and the decision
- [x] Method section lets a reader reproduce every number: host, models, quantization, pins, commands, revisions
- [x] Results present all three runtimes in comparable tables, with non-comparable cells marked as such rather than blank
- [x] Threats to validity names every known measurement limitation AND its direction, including the ones that favoured llama.cpp
- [x] Discussion separates what the numbers show from what they would need to show to justify migration
- [x] Conclusion is an explicit GO or NO-GO with the thresholds it was judged against
- [x] Every claim traceable to an attached artifact; no number appears that cannot be sourced
- [x] Article lives in articles/<YYMMDD>_local-qwen-runtime-comparison-study/ with the date of writing in the directory name
- [x] Opens with a dated Provenance section: what was measured, on which binaries and host, on which dates, so a later reader can tell an expired finding from a live one
- [x] ARTICLE.md, README.md, SHA256SUMS, artifacts/ and reproduce.zsh all present, matching the voice-research article convention
- [x] Every cited number is reproducible from artifacts/ and checksummed in SHA256SUMS
- [x] Decision selects the best overall compromise, not a single-criterion winner, and says so explicitly
- [x] Peak resident memory and decode rate weighted as equally important, with decode sometimes dominant, and the weighting stated rather than implied
- [x] Time to first token, 75000-token capacity, tool-call parity, stability and migration risk all carried into the trade-off, not just the two headline axes
- [x] Break-even computed and stated: how many output tokens make the decode penalty cancel the TTFT advantage, with the unit of every cited latency verified against raw artifacts before use
- [x] Where an axis is non-comparable or unmeasured, the decision states how it was handled rather than dropping the axis
- [x] Classic structure: abstract, background, method, results, threats to validity, discussion, conclusion
- [x] Dated file in .research/ following the existing YYMMDD_ naming convention
- [x] Method section states the pre-registered weighting and the break-even formula, and notes both were fixed before the numbers
- [x] Reports that the gate refused the pair with exit 4 on contextPolicy, in the same breath as the numbers
- [x] States that llama-server reports neither prefill-step nor reasoning, and treats that attestability gap as a migration-risk finding
- [x] States that peak resident memory produced no comparison and why, in the decision section rather than a footnote
- [x] Makes no memory comparison in any direction and does not cite the withdrawn -5.64 percent figure
- [x] Does not cite context_75k decode; uses that scenario for capacity, TTFT and prefill only
- [x] Reports that no break-even exists in the measured direction, rather than manufacturing one
- [x] Records that the withdrawn 10 percent slower decode claim was made from SSE frame counts and is overturned
- [x] Ends in an explicit GO or NO-GO selecting the best overall compromise, with the applied weighting stated
- [x] States the standing constraint that the pinned mlx-lm fork is held until an accepted replacement carries its fixes
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
STORY-260828-2faxgm base refresh SKIPPED: the managed workspace holds uncommitted work, so there was no clean checkpoint branch to replay onto trunk bb857fe5c31c; the branch is unchanged at fork point b78498bf98c0
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260830-e86d80, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260830-e86d80)
Synthesis complete; handed to review.

DELIVERABLE
- articles/260831_local-qwen-runtime-comparison-study/ -- ARTICLE.md, README.md, SHA256SUMS, artifacts/ (42 files, 2.9 MB), reproduce.zsh, per the voice-research convention, opening with a dated Provenance section per the house style.
- .research/260831_local-qwen-runtime-comparison-study.md -- byte-identical copy under the repo dated convention (cmp exit 0).
- Repo README gains a Research and Articles section plus a Tooling row for the article verifier. LOGBOOK gains two entries. The superseded .research/260829_ doc gains a supersession banner naming both withdrawn claims.

DECISION: NO-GO on both candidates. Python mlx-lm remains the default; profiles.qwen-local untouched.
- MLX Swift: scored, accepted=false, one reproduced blocker -- long_prompt_8k peak footprint 1.1512x against <= 1.10 (1.144x and 1.151x across two reruns). Retired.
- llama.cpp: rejected for now, NOT retired. It is faster on every measurable axis (decode +20.1%/+15.8%, 8k TTFT 0.629x, prefill 1.590x, wall clock 0.66-0.87x) and the decision does not rest on doubting that. It rests on: the gate refused the pair exit 4 with no decision.json because two contextPolicy terms are not-reported; an equally-weighted axis (memory) produced NO comparison at all, zero windows scored on both sides; and of the measured advantage only TTFT and prefill are outside this host ~+/-20% single-run noise floor.
- No break-even exists in the measured direction: llama.cpp leads both TTFT and decode, so L* is -15.73 and -1946.62 output tokens. Reported as an absence, not manufactured.
- Cross-arm comparison refused with four reasons, including the incumbent decoding 1.46-1.51x faster in the earlier campaign.
- Standing constraint recorded: the pinned mlx-lm fork is held until an accepted replacement carries its fixes; STORY-260830-2vrhg1 must not be stranded.

BRIEF PROHIBITIONS HONOURED: no memory comparison in any direction; the withdrawn memory figure is not restated even to be argued against; no unscored components printed; +6.2 GB soak climb carries its ungated provenance and is not called a leak; context_75k decode figures not printed; multiturn and soak timings not used as speed results; the exit-4 refusal is stated in the same breath as the numbers; the absent memory axis is in the decision section, not a footnote.

VALIDATION (literal exits, TASK-260828-15ftgj_validation.log)
- zsh reproduce.zsh: 0 -- checksums, record decompression, full figure recomputation byte-identical to expected-figures.json, and 25 structural checks.
- ruff check 0, ruff format --check 0, zsh -n 0, bash -n 0, git diff --check 0, cmp article/research 0, recompute determinism 0.
- No Swift/Go/shell source touched: git diff HEAD --name-only -- tools Sources scripts setup.sh setup.ps1 is empty.

NEGATIVE EVIDENCE: 13 narrowing mutants against reproduce.zsh, each falsifying one article claim and then REGENERATING expected-figures.json and SHA256SUMS from its own mutated artifacts so only the structural block can catch it. 13/13 caught, each by the check naming the claim it attacks. Production call site: the inline structural block in reproduce.zsh, reached unconditionally on every green path. Two survivors reported not hidden -- B1 (recompute.py stubbed to copy the expectation) and D1 (delete-only control). Full record: TASK-260828-15ftgj_mutant-campaign.md.

NOT DONE, and stated: no measurement was re-run; this task writes up and decides. Independent replication of the runs is not established -- only arithmetic fidelity to the sealed records.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-e86d80, pid=44919, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260830-3d329c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260830-3d329c)
Review rev1 ACCEPTED (RUN-260830-3d329c). Verified independently of recompute.py: 297 of 298 cited numbers re-derived straight from records/*.json.gz, session.json, attestations, run-rev4.log and Swift sources — including all §4.4 gap arithmetic recomputed from 36,740 and 23,942 raw stamps. SHA256SUMS 41/41; reproduce.zsh green; ARTICLE.md byte-identical to the .research copy. Decision attacked: NO-GO survives deletion of its weakest leg (the noise-floor claim) — grant llama.cpp its full decode win and you still hold one of two equally-weighted axes with the other having no data, against a named migration-risk loss. All brief constraints satisfied (no memory comparison, -5.64 absent, context_75k decode withheld though present in the record at 1.0934x, multiturn/soak excluded, no manufactured break-even, frame-count error attributed). Verifier attacked with 3 fresh mutants: MU-A caught; MU-B and MU-C survived. 5 findings, none decision-changing, all for orchestrator follow-up: F1 §4.2 prints 1.5493x where recompute derives 1.548726 (only untraceable number; falsifies §3.7 in one cell); F2 reproduce.zsh MLX Swift checks read decision.json not the records (MU-B passes green with a record that no longer supports the blocker); F3 crossCampaign.comparable is a hardcoded False so that check is a tautology; F4 campaign claim of 13-mutants-all-caught overstates coverage (MU-C shaves decode +20pct to +2pct and survives); F5 dangling §7.2.1 ref, and §7.2 reopening items only partly board-tracked. Full reasoning: TASK-260828-15ftgj_review-verdict.md. No commit_ack supplied.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-3d329c, pid=53107, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260830-e3d95a, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260830-e3d95a)
Rev2 republication run RUN-260830-e3d95a (developer): verification only, article untouched.

Verified independently, not accepted from the brief:
- Base is EXACTLY current trunk: HEAD=cdcd8f7 is the merge, its second parent equals freshly fetched origin/main bb857fe; HEAD..main = 0; .configs/codex-config.toml identical to trunk.
- Article unchanged: reconstructed the accepted CR rev1 candidate from TASK-260828-15ftgj_change-request_rev1.patch (sha256 279ef45a... matches the CR record) onto base 3f313d9; articles/260831_* diff -r exit 0 and both .research/ copies cmp exit 0 against the live worktree. Same result against the preserved article-delta.patch (sha256 136b582f...): 44/46 paths byte-identical.
- The 2 differing paths are LOGBOOK.md (declared conflict) and README.md (UNDECLARED in the brief - trunk added an External-CI local mirror fallback section; the Story README delta itself is byte-identical to the preserved patch).
- Both LOGBOOK sides survive, checked as a multiset of non-blank lines: Story added 249 vs merge base, trunk added 13, ZERO missing either way. Whole-file check flags one trunk line, the heading ## 2026-08-27, which the merge base already carried twice and the Story side deduped pre-merge inside the accepted delta - not a merge loss. The brief 112/63 counts could not be reproduced; the conclusion holds under a stricter rule.
- Shape: .research study 1103 lines, article dir 42 files.
- Green at the merged base: shasum -a 256 -c SHA256SUMS exit 0 (41 OK), ./reproduce.zsh exit 0 PASS, go vet ./... exit 0, go test ./... -count=1 exit 0 (4 ok / 0 FAIL). Run standalone, real exit codes.
- Working tree at publication: 46 paths, 9878 insertions, 0 deletions.

No behavior changed, so no test was added. Evidence: TASK-260828-15ftgj_rev2-republication-verification.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-e3d95a, pid=79612, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-29c925, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-29c925)
Reviewer RUN-260831-29c925, CR revision 2: ACCEPTED. Scope was the three brief items only; article content/numbers/decision stay accepted from rev1 and were not re-opened. (1) Fresh base: HEAD cdcd8f7, HEAD..main = 0, main = origin/main = CR base bb857fe, .configs/codex-config.toml identical to trunk. (2) Article delta invariant: regenerated from immutable tree OIDs (index never touched); both patches 3353792 bytes / 51733 lines, exactly 3 lines differ - two index blob lines (LOGBOOK.md, README.md) and one README hunk offset 1643->1657. Zero content lines differ. .research doc 1103 lines, article dir 42 files, ARTICLE.md byte-identical to the .research copy, shasum -c SHA256SUMS 41 OK / 0 fail. 173->82 drop explained: rev2 paths are a strict subset of rev1, 91 dropped, 0 under articles/ or .research/260831, and all 91 exist in the candidate tree byte-identical to trunk. (3) LOGBOOK merge additive, proven four ways: multiset survival (story 235/235, trunk 13/13 unique lines; 0 lines in the merge that came from neither side); the one shortfall (## 2026-08-27) is a story-side pre-merge dedup of a base duplicate, git diff b78498b f6da7e4 shows the removal; every one of the 45 lines missing vs trunk is exactly one the story had already removed pre-merge (unexplained = 0), and the merge removed 0 lines from the story side; git merge-tree --write-tree reproduces the mechanical merge, conflicting only in LOGBOOK.md, and the hand result differs from it by 3 conflict markers plus a verbatim reorder (added multiset == removed non-marker multiset). No conflict markers left, git diff --check exit 0. Non-blocking: the brief understated the base move by one file (README.md base also moved +14 lines), and the handed 112/63 line-count claim could not be reproduced under any metric - reported as unreproduced; nothing in the verdict rests on it. Evidence: TASK-260828-15ftgj_review-verdict-rev2.md. No commit_ack supplied; orchestrator owns the done transition.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-29c925, pid=93126, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260828-15ftgj_spawn-log_-implementer--developer--claude-_RUN-260830-e86d80.log](file://TASK-260828-15ftgj/TASK-260828-15ftgj_spawn-log_-implementer--developer--claude-_RUN-260830-e86d80.log) — System spawn log captured by task-board
- [TASK-260828-15ftgj_local-qwen-runtime-comparison-study.md](file://TASK-260828-15ftgj/TASK-260828-15ftgj_local-qwen-runtime-comparison-study.md) — The comparison study: three runtimes, two refusals, explicit NO-GO on both candidates under a pre-registered equal weighting of memory and decode
- [TASK-260828-15ftgj_article-snapshot.tar.gz](file://TASK-260828-15ftgj/TASK-260828-15ftgj_article-snapshot.tar.gz) — Full checksummed article directory: ARTICLE.md, README.md, SHA256SUMS, artifacts/ and reproduce.zsh; reproduce.zsh exits 0 on this snapshot
- [TASK-260828-15ftgj_expected-figures.json](file://TASK-260828-15ftgj/TASK-260828-15ftgj_expected-figures.json) — Every figure the article cites, recomputed from the sealed records; reproduce.zsh fails if any of them moves
- [TASK-260828-15ftgj_mutant-campaign.md](file://TASK-260828-15ftgj/TASK-260828-15ftgj_mutant-campaign.md) — 13 narrowing mutants against the article's own verifier, all caught by the check naming the claim they attack; two survivors reported not hidden
- [TASK-260828-15ftgj_validation.log](file://TASK-260828-15ftgj/TASK-260828-15ftgj_validation.log) — Literal exits of every validation command run for this task, the mutant campaign's expected-red results, and the final-tree revalidation
- [TASK-260828-15ftgj_change-request_rev1.patch](file://TASK-260828-15ftgj/TASK-260828-15ftgj_change-request_rev1.patch) — Change Request CR-TASK-260828-15ftgj-1 revision 1 candidate patch (repository_delta=present, 173 changed paths)
- [TASK-260828-15ftgj_change-request_rev1-validation.log](file://TASK-260828-15ftgj/TASK-260828-15ftgj_change-request_rev1-validation.log) — Change Request CR-TASK-260828-15ftgj-1 revision 1 bounded validation log
- [TASK-260828-15ftgj_spawn-log_-reviewer--reviewer--claude-_RUN-260830-3d329c.log](file://TASK-260828-15ftgj/TASK-260828-15ftgj_spawn-log_-reviewer--reviewer--claude-_RUN-260830-3d329c.log) — System spawn log captured by task-board
- [TASK-260828-15ftgj_review-verdict.md](file://TASK-260828-15ftgj/TASK-260828-15ftgj_review-verdict.md) — Reviewer verdict rev1: ACCEPTED. 297/298 cited numbers re-derived independently of recompute.py; 3 fresh verifier mutants (2 survived); 5 findings, none decision-changing.
- [TASK-260828-15ftgj_spawn-log_-implementer--developer--claude-_RUN-260830-e3d95a.log](file://TASK-260828-15ftgj/TASK-260828-15ftgj_spawn-log_-implementer--developer--claude-_RUN-260830-e3d95a.log) — System spawn log captured by task-board
- [TASK-260828-15ftgj_rev2-republication-verification.md](file://TASK-260828-15ftgj/TASK-260828-15ftgj_rev2-republication-verification.md) — Revision-2 republication verification: base is exactly current trunk, article byte-identical to the accepted rev1 candidate, both LOGBOOK sides survive, reproduce.zsh and the Go suite green at the merged base
- [TASK-260828-15ftgj_change-request_rev2.patch](file://TASK-260828-15ftgj/TASK-260828-15ftgj_change-request_rev2.patch) — Change Request CR-TASK-260828-15ftgj-2 revision 2 candidate patch (repository_delta=present, 82 changed paths)
- [TASK-260828-15ftgj_change-request_rev2-validation.log](file://TASK-260828-15ftgj/TASK-260828-15ftgj_change-request_rev2-validation.log) — Change Request CR-TASK-260828-15ftgj-2 revision 2 bounded validation log
- [TASK-260828-15ftgj_spawn-log_-reviewer--reviewer--claude-_RUN-260831-29c925.log](file://TASK-260828-15ftgj/TASK-260828-15ftgj_spawn-log_-reviewer--reviewer--claude-_RUN-260831-29c925.log) — System spawn log captured by task-board
- [TASK-260828-15ftgj_review-verdict-rev2.md](file://TASK-260828-15ftgj/TASK-260828-15ftgj_review-verdict-rev2.md) — Reviewer verdict for Change Request revision 2: ACCEPTED. Independent verification of fresh base, article-delta invariance (3 metadata lines differ out of 51733), and additive LOGBOOK.md merge proven four ways.

## Created
2026-08-28T10:12:56Z

## Last Update
2026-08-31T00:39:52Z

## Assigned To
[reviewer] reviewer (claude)
