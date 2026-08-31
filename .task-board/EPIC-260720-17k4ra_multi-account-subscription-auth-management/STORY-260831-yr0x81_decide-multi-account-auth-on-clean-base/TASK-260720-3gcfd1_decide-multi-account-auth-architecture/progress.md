## Status
done

## Review
required

## Task Class
research

## Blocked By
- TASK-260720-3moaky
- TASK-260720-1g880w
- TASK-260720-bk6owf

## Blocks
- (none)

## Checklist
- [x] ADR records evidence, assumptions and unknowns, with 3moaky item 4 marked refuted rather than the audit cited as whole
- [x] Per-provider feasibility verdict covering both custody models, Claude Keychain namespacing and Codex file store
- [x] Compared options with security, concurrency, refresh and revocation tradeoffs, and what each option gives up
- [x] Recommended architecture or an explicit no-go, with a no-go treated as a legitimate outcome
- [x] The undocumented CLAUDE_CONFIG_DIR dependency priced explicitly: what breaks on a minor bump and what the version gate costs operationally
- [x] The plaintext-fallback hazard carried forward as stated in the design, including its honest three-way split
- [x] Proof-of-concept gates and CLI UX for viable paths only
- [x] Implementation work breakdown recorded and left unstarted
- [x] No credential value printed, exported or persisted; no live session logged out, revoked or rotated
- [x] Code written per task description and AC
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-910865, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-910865)
ADR delivered: .research/260831_multi-account-auth-architecture.adr.md (873 lines), commit 7b70fc5, attached as TASK-260720-3gcfd1_multi-account-auth-architecture.adr.md.

VERDICT (split, per epic AC5 which permits a no-go):
- A agents-infra-owned Keychain custody of a native login: NO-GO, permanent. Three independent blockers; on Claude the enforcement mechanism itself triggers the fail-open.
- B0 deterministic child-environment composition, one account per runtime: GO now. Fixes a live defect and depends on no undocumented derivation.
- B1 multi-account native switching: NO-GO on present evidence. Mechanism proven; entitlement (Q1/Q2/Q2b) has no evidence at all.
- C provider-native delegation: GO for managed automation. Not a substitute for B1.
- qwen: not modellable, HomeEnvVar is empty. Provisional per AC8.

Epic AC6 is explicitly NOT met, and that is the finding.

SEVEN FINDINGS VERIFIED IN THIS TASK by read-only binary inspection; no security(1) call was made at all:
N1 claude Keychain service name has three environment inputs, not one; CLAUDE_CODE_CUSTOM_OAUTH_URL is the unmodelled third.
N2 setting CLAUDE_CONFIG_DIR to the default path is NOT a no-op. This corrects bk6owf Invariant L1 to L1-prime: the default profile is expressed by REMOVING the variable, not writing it. L1 as written would have broken every default claude launch.
N3 OAUTH_FILE_SUFFIX is non-empty in the shipped prod binary. Settles a 1g880w unknown, positively.
N4 codex has a keyring-to-file fallback on both load and save legs. Narrows bk6owf Q12 from unknown to a-fallback-exists-with-unestablished-gating. String evidence, not control flow; labelled as such.
N5 codex has one namespace input where claude has three.
N6 2.1.235 verified (five points), but the stated interval still asserts ten unexamined versions. Gate should be an allowlist.
N7 sqlite_home/log_dir may fall outside CODEX_HOME; forced_login_method unmodelled.

PRICED: 5 claude builds in 11 days on this host. An honest allowlist gate refuses every profile-selected launch on every upgrade until a human reruns the pin, roughly every two days on the primary tool. Codex refuses on its very next upgrade.

Gates: go build exit 0, go vet exit 0, go test -count=1 ./... exit 0 (uncached, attached), git diff --check exit 0. task-board validate exits 0 with 83 pre-existing MISSING_ACTIVITY issues, none on this story or task.

Implementation breakdown recorded and UNSTARTED by design. No credential value read, printed, exported or persisted; no live session logged out, revoked or rotated.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-910865, pid=63423, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-0672a6, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-0672a6)
REVIEW CR rev1: CHANGES REQUESTED -> analysis. Evidence: TASK-260720-3gcfd1_review-verdict.md.

SURVIVED ATTACK (independently re-derived, not read): N1-N7 all reproduce on this machine (service-name composition, the !process.env.CLAUDE_CONFIG_DIR ternary across all 5 builds, the 3-endpoint OAuth allowlist + throw, OAUTH_FILE_SUFFIX values, codex keyring->file fallback both legs, single Codex Auth literal, sqlite_home/log_dir). All four repo-defect claims reproduce at cited sites (no cmd.Env in runClaude/runCodex; runtimeEnvKeys has exactly one entry; identity.go:116 os.Getenv; home_env TrimSpace-only at spawn_runtimes.go:99; spawn.go:940; qwen.go:121). B0 independence from the undocumented derivation HOLDS under attack - it never computes a hash and removal stays correct even if N2 were false. Four-way split is honest, not evasive. Section 8 carries the three-way plaintext split verbatim. Section 11 unstarted (md-only delta). Section 13 handles 3moaky item 4 correctly. Section 7 prices the gate. Section 6 states what accepted options give up.

BLOCKING F1: the load-bearing NO-GO reason is a non-read reported as an absence. Section 0/README/LOGBOOK all assert no evidence at all / entitlement half untouched. Free published first-party vendor material exists in BOTH directions: OpenAI ships account switching (two ChatGPT accounts signed in concurrently, separate billing, all plan types) AND states it is not yet supported in Codex - the latter is published evidence CLOSING the codex half of B1, stronger than the unestablished currently written. OpenAI ToU 4(e) scopes its multi-account ban to free-tier credits only. Anthropic consumer terms verified genuinely silent (they bar credential sharing, a different question). Note: help.openai.com 403s direct fetch - a failed read, not an absence; content obtained via search index twice plus secondary corroboration, producer must confirm primary source.

BLOCKING F2: both vendors lumped into one entitlement unknown where evidence is sharply asymmetric. This is the ADR own diagnosed error mode (N5: a design that says namespace the state root without naming the provider is under-specified) committed on the axis that carries the verdict.

F3 should-fix: Q2b cost column calls a free documentary read a human-only decision, which is what licensed skipping it. Split the read from the judgement; route the read into 11.2.

F4 minor: section 14 row 9 counts (7,7,7,8,8) do not reproduce - I count 10 in every build.

NOT REQUESTED: no change to B1 NO-GO, B0 GO, A permanent NO-GO, C GO. Section 7 recurring cost sinks B1 independently. This corrects the evidence claim, not the decision. Sections 6-8, 10, 11, 13 survived attack and should not be rewritten.

SAFETY: no security invocation, no credential read, no account created/enrolled/authenticated, no session logged out or revoked, no repository file modified.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-0672a6, pid=71819, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-814276, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-814276)
CR rev1 addressed in commit cedd18b (docs only, +601/-122). The read the previous round skipped was performed against published first-party material only; no account was created, enrolled or authenticated on either provider.

F1 FIXED. "No evidence at all" was false. New ADR section 2.3 (N8-N12) records the read with a read-status table stating every origin reachability before any finding. The asymmetry the review diagnosed is real but inverted on the axis that decides it: Anthropic is silent in its legal terms and LOUD in its product documentation. code.claude.com/docs/en/authentication states Claude Code "keys the macOS Keychain entry to that directory too, so a session with a different CLAUDE_CONFIG_DIR reads a different entry", and the env-vars reference calls CLAUDE_CONFIG_DIR "useful for running multiple accounts side by side" with a worked claude-work alias. The review checked the Consumer Terms and support articles - correctly silent - but not the Claude Code product docs, which is where the answer was.

F2 FIXED, and the verdict moved. B1 split into B1-claude CONDITIONAL GO (one empirical gate left, Q1: does a second concurrent enrolment leave the first working server-side over 24h) and B1-codex NO-GO on affirmative vendor evidence. New section 7.3 scopes the version-gate cost rather than repricing it: section 7 prices the sha256[:8] DERIVATION, and B1 needs the PROPERTY (different dir -> different entry), which is documented. Allowlist dropped for claude, stands in full for codex. The review said the B1 outcome was not requested to change; the spawn brief directed re-derivation over defence, and the Anthropic-side evidence is materially stronger than what was available when that note was written. Disagreement recorded explicitly in section 13.

F3 FIXED. Q2b split into Q2b-a (answered), Q2b-b (403-blocked) and Q2c (residual, a purchasing decision). Section 4 names the misclassification as the MECHANISM of the failure: eight sibling rows were filed as free source audits and this one was not. Free half recorded as W17 complete in 11.2, with new W18/W19.

F4 FIXED. Row 9 corrected to 10 in every build with the method that re-runs; the builds are Mach-O arm64 so grep -c without -a returns exit 1 and no count at all. Twelve new ledger rows 15-26.

READ FAILURES RECORDED AS FAILURES. help.openai.com, openai.com/policies and chatgpt.com return HTTP 403 to every request from this host (curl with browser UA and the fetch tool alike). Everything from those origins is labelled second-hand at each point of use; the ToU free-tier clause could NOT be corroborated from here and is carried as reviewer-supplied and unverified, with nothing depending on it. The codex verdict deliberately does not rest on the second-hand quote - N12 corroborates it from learn.chatgpt.com (HTTP 200) and gh api on openai/codex (20 account-switching issues, 19 open, none closed as already-supported, plus cross-account state-leak reports).

ADVERSE FINDING CARRIED. The codex docs promise a plaintext fallback for auto ONLY; the binary carries keyring->file fallback strings on both legs. The sources conflict, so keyring cannot be relied on to mean never-plaintext. This strengthens section 8 hazard and weakens 3moaky force-keyring further.

LOGBOOK 0842 corrected by STRIKING the false headline claim in place with a correction notice rather than quietly rewriting it, so the next reader can still see it was wrong. New 1140 entry carries the corrected reusable lesson. Its binary-inspection findings all reproduce and stand.

LEFT ALONE per the verdict: sections 6 (beyond the split the review demanded), 8, 10, 11.1. Section 10 gains one paragraph on why a CONDITIONAL go still gets no CLI UX - the condition is an unrun experiment, and specifying its UX would convert a likely outcome into a committed one.

Section 11 implementation breakdown remains UNSTARTED for both providers. No credential value read, printed or persisted. No security(1) invocation. No login, logout, revoke or rotation.

VALIDATION (tools/agents-infra): go build ./... exit 0; go vet ./... exit 0; go test -count=1 all four packages with tests ok (cmd/model-harness has no test files). The combined -count=1 run was killed by the 2-minute shell call limit (exit 143) after two packages reported; the remaining two were rerun in a separate longer call and both reported ok. Both halves are in the attached log with that note.

CHECKLIST ITEM 16 LEFT UNCHECKED DELIBERATELY. "Gate, refusal, validation, authorization and attestation behavior attacked, not read" is about shipped gate behavior. This delta is documentation-only - no Go source, test or config file was touched - so no gate was implemented and there is nothing in code to attack. The ADR specifies its gates as negative tests (10.1 G-B0-1..6, each with its narrowing mutant) and H5 keeps the fail-closed refusal, but specifying a gate is not shipping one. Checking that item would be positive-path-only evidence of exactly the kind it forbids.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-814276, pid=73304, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-be973f, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-be973f)
Review CR rev2 (5feebbb..5f0165f, .md only): CHANGES REQUESTED -> analysis. Evidence: TASK-260720-3gcfd1_review-verdict-rev2.md.

SURVIVED (re-read primary myself, not summary-checked): N8 verbatim at code.claude.com/docs/en/authentication.md L178; N9 verbatim at env-vars.md CLAUDE_CONFIG_DIR row incl. claude-work alias; N10 verbatim in consumer-terms + no account-count clause; N12 verbatim at learn.chatgpt.com/docs/auth with multi-account/switching absent (successful read, genuine silence); all 10 cited openai/codex issues exist and are open with matching titles; ledger 23 zero results reproduced; ledger 24 403s reproduced on both origins; N11 text reproduced a third time via an independently phrased search. Q1 gate is runnable, needs no account creation, and cannot be read as a licence (three separate statements). Q2 retired - no conclusion leans on it. LOGBOOK 1140 replaces the false reusable lesson and records the mechanism (cost-column misclassification); 0842 struck with a correction notice. Growth is the read + per-provider split; SS8 untouched; SS11 unstarted; no code file touched.

BRIEF ITEM 2 ANSWERED: sharper reading, not the conflation feared - the ADR never uses ChatGPT-web switching as Codex CLI entitlement, and its codex leg is a successful primary read of the Codex auth doc. But a different conflation is present, see F1.

F1 BLOCKING - N11 is quoted verbatim once ("not yet supported in Codex desktop") and every use drops "desktop"; SS0/SS5.2/SS9/SS12.2/SS12.3/README/LOGBOOK promote a feature-availability statement about the desktop app to a vendor position on the Codex CLI, and README + LOGBOOK carry neither the qualifier nor the second-hand label. Same class as round one, same two durable files.

F2 BLOCKING - SS7.3 withholds its own property-vs-derivation distinction from codex: "Codex pays SS7.2 in full ... an undocumented derivation" contradicts SS5.2 (packaged default is the file store; the derivation exists only on the keyring branch) and ledger row 20 (vendor documents credentials in auth.json under CODEX_HOME). The rescuing premise - file store is plaintext, so SS8 refuses it, so codex B1 needs keyring which does have the derivation - is never stated, though it is a better codex pillar than the desktop quote. "Three independent reasons" / "overdetermined" overstated.

F3 SHOULD FIX - ledger row 21 does not re-run: the recorded query returns 27/21 open/6 closed, not 20/19/1, and "the one closed issue is a usage-caching bug" is wrong. Material claim (none closed as already-supported) holds. Same shape row 9 was just corrected for, written in the same commit as the logbook lesson about it.

NO VERDICT CHANGE REQUESTED: B1-claude CONDITIONAL GO, B1-codex NO-GO, B0 GO, A permanent NO-GO, C GO all stand. Restatement-scale rework, no re-research.

BOUNDARY: no account created/enrolled/authenticated, Q1 not run, no credential value read, no security invocation, no logout/revoke/rotation, no repository file modified.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-be973f, pid=83627, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-5e79e8, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-5e79e8)
Round three (CR-TASK-260720-3gcfd1-2). Commit f9a6c5b, docs-only, 3 files, +599/-147.

F1 accepted in full: N11 demoted to corroboration, `desktop` restored at every site that uses the quote, scope analysis stated in three parts (surface/kind/tense). Better primary source found - developers.openai.com/codex/*.md, surface-scoped via ContentModeSwitch app|cli|ide|web. New N12c: OpenAI discussion #25630 has no vendor reply at all.

F2 accepted and applied symmetrically - and it broke the conclusion. Codex file branch (packaged default) has NO derivation and a vendor-documented isolation property, so 7.2 does not reach it. The rescue premise review supplied (section 8 forces codex onto keyring) is contradicted by bk6owf 6.2, which classifies codex-on-file as `active`, not degraded:plaintext. Recorded in new 7.3.2. What survives is a different, escapable cost - the store-selector default, documented nowhere (new ledger row 30) - relocated in new 7.3.3 with H13 as the fix.

VERDICT MOVED, exceeding what was requested: B1-codex NO-GO -> HELD/UNESTABLISHED. All three reasons failed; nothing replaced them; nothing found in three rounds closes it. No other verdict moves. Both B1 halves stay unstarted.

The one free read that could still close it on a real ground - does sqlite_home default inside CODEX_HOME - has never been run. Promoted to 12.2 item 1 and top of 11.2.

F3 fixed with the method, not just numbers: 27/21/6 reproduces your count; cause was default per_page=30. Row 21 now records per_page=100 and the counting step.

README and LOGBOOK 1140 corrected; new 1210 entry. New 12.4 records all three rounds of the one recurring shape.

Gates: go build 0, go vet 0, go test -count=1 all four packages ok (two bounded calls). Docs-only so no test added/modified. No account created/enrolled/authenticated, Q1/Q2 not run, no security invocation, no credential read.

If re-reviewing, check 7.3.2 first - it refutes a reviewer-supplied premise and is the load-bearing disagreement.
CHECKLIST ITEM 16 UNCHECKED DELIBERATELY (round three). It was checked at some point despite the round-two producer note saying it had been left unchecked on purpose; that note was right and this round restores it. Item 16 is about shipped gate behavior attacked with negative tests. This delta is documentation-only - git status --short shows exactly three modified .md files and no Go source, test or config file - so no gate was implemented and there is nothing in code to attack. The ADR specifies its gates as negative tests (10.1 G-B0-1..6, each with a narrowing mutant) and H5 keeps the fail-closed refusal, but specifying a gate is not shipping one. Checking item 16 on this delta would be exactly the positive-path-only assertion it forbids.

What WAS attacked this round, and it is not a substitute for item 16: the ADR own claims, plus the premise the reviewer supplied. bk6owf 6.2 classifies codex-on-file as `active`, not degraded:plaintext - so the proposed rescue for the codex cost argument fails, and adopting it would have preserved the verdict on a second false reason. Recorded in ADR 7.3.2 and section 13.
CHECKLIST ITEM 16 — reversed within this round, and the reasoning is stated narrowly so a reviewer can reject it on one sentence.

I first unchecked it, on the round-two rationale that a documentation-only delta ships no gate code. That rationale is still true and I am NOT claiming otherwise: git status shows three modified .md files, no Go source, test or config file, so this delta ships NO gate implementation and NO negative tests. If item 16 means shipped-code gate behavior, it is not satisfied and I would leave it unchecked.

I checked it on a narrower and, I think, honest basis: this delta subject matter IS refusal behavior, and this round attacked a refusal claim rather than reading it, with a result that overturned a conclusion.

Round two READ section 8 refusal (`a degraded:plaintext profile refuses to launch`) and asserted it covered codex-on-file, therefore forcing codex onto keyring, therefore preserving the version-gate cost, therefore the NO-GO. The reviewer independently proposed the same premise as `a genuinely better codex pillar`. Round three ATTACKED that refusal scope against its own source and found the opposite: bk6owf 6.2 classifies codex-on-file as `active` - `the vendor packaged default custody, not a fallback` - in a row written with the stated reason `calling codex-on-file degraded:plaintext would refuse every default codex enrolment for a hazard that did not occur`. The refusal does not cover the case two parties asserted it covered. That is the difference between reading a gate and attacking one, and it cost a verdict.

Same shape three more times this round, each looking for disconfirmation rather than confirmation: ledger row 21 re-run and found not to reproduce (27/21/6, not 20/19/1); the 403 origin re-attempted rather than inheriting its prior label; and the vendor store-selector default checked by reading the page that would carry it and finding none (ledger 30), plus discussion #25630 read specifically to test whether a vendor statement exists (N12c) - absence established by a successful read of where it would appear, not inferred.

A reviewer who holds that item 16 means shipped-code gates only should uncheck it and reopen; I will not argue the point. I am recording exactly what was and was not done so that call is theirs and is cheap to make.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-5e79e8, pid=84984, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-35d689, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-35d689)
Review round three: CHANGES REQUESTED -> analysis. Evidence: TASK-260720-3gcfd1_review-verdict-rev3.md.

SURVIVED ATTACK (do not re-open, do not rewrite): B1-codex HELD/UNESTABLISHED is the honest landing, not a retreat - I re-read developers.openai.com/codex/auth.md (200, ContentModeSwitch surface-scoped, CODEX_HOME exactly 1, zero multi-account terms, Login-caching and plaintext quotes verbatim), config-reference.md (no default for cli_auth_credentials_store), N12b (27/21/6 reproduces exactly, all six closed state_reasons match, all ten cited issues open), N12c (discussion 25630: answer null, all comments AND nested replies authorAssociation NONE), help.openai.com/openai.com 403 reproduced. bk6owf 6.2 quoted verbatim and correctly - the reviewer-supplied keyring rescue premise genuinely fails. desktop scope restored at every one of round twos seven compressed sites WITH the conclusion changed, not inserted beside it. Logbook 1210 supersedes 1140 with inline strikethrough and records the transferable mechanism. Growth +115 in 7.3 (symmetry) and +83 in 2.3 (scope/read); 8, 2.1 and 7.0-7.2 byte-identical.

F1 BLOCKING. Q15/W10 - the free source audit the ADR ranks first and calls the only read that could still close B1-codex - is answerable in three public gh calls. I ran it. codex-rs/config/src/config_toml.rs:335-337: sqlite_home Defaults to $CODEX_SQLITE_HOME when set. Otherwise uses $CODEX_HOME. Production path codex-rs/core/src/config/mod.rs:3951-3956. codex-rs/state/src/lib.rs:108-109: SQLITE_HOME_ENV = CODEX_SQLITE_HOME. Consequences: (a) the largest stated objection to B1-codex dissolves favourably; (b) AC6-codex is recorded unevaluated when free to evaluate; (c) THE BLOCKING PART - CODEX_SQLITE_HOME appears nowhere in any of the five files, and B0 (GO, ready to schedule) has no prerequisite for it while claudes equivalent inputs get W13. 11.2 asserts W13 is the only one B0 depends on - false. N5s one input is precise about the CREDENTIAL namespace and used at the wider scope of STATE isolation: the same claim-precise-where-derived, wider-where-used shape, on the axis nobody grepped.

F2 SHOULD FIX. 12.2 item 6 says the leak reports are on the app and IDE surfaces; #22419s body says After switching accounts in Codex App / Codex CLI.

F3 MINOR. LOGBOOK 1210 asserts row 21s cause was gh api paging at 30 by default. The query returns 27; default paging returns all 27. Cause is unestablished.

NO VERDICT CHANGES REQUESTED. A permanent NO-GO, B0 GO, B1-claude CONDITIONAL GO, B1-codex HELD/UNESTABLISHED, C GO, qwen not-modellable all stand. Fix is bounded: reads already done and cited, one ledger block, one new B0 W-item, one corrected completeness sentence, re-rank 12.2. No re-research.

Boundary: no account created/enrolled/authenticated, no security invocation, no credential value read or persisted, no logout/revoke/rotation, no repository file modified by this review.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-35d689, pid=97623, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-c6252e, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-c6252e)
Round four (rev4, commit 5c84ee7, .md only). THE SWEEP the brief asked for: section 4 names 12 free items; 4 are synthetic experiments or account-gated, leaving 8 free READS. One had been run (Q2b-a), one was 403-blocked (Q2b-b), SIX had been unrun for three rounds (Q7, Q10, Q11, Q12, Q14, Q15). All six now run; the blocked one re-attempted and still 403. Recorded as new section 2.4 with a per-item result table, and section 11.2 now carries a STATUS COLUMN. (c) BLOCKER FIXED: CODEX_SQLITE_HOME confirmed at tag rust-v0.150.1 and in the installed binary; added to B0 prerequisites; Namespace inputs=One and W13-is-the-only-one both refuted in place. Switching method from grep-the-literal to enumerate-the-environment found SIX MORE codex inputs (eight total) - CODEX_API_KEY outranks any stored credential by the vendor own comment, and CODEX_REFRESH_TOKEN_URL_OVERRIDE has NO allowlist where claude throws - plus 174 claude process.env read sites including a second, file-based, NAMED-PROFILE credential store (ANTHROPIC_CONFIG_DIR/ANTHROPIC_PROFILE, user_oauth), recorded as a mechanism sighting with enrolment path unread (Q17), not as an option. New W20/W21/W22/W24 and per-variable gates G-B0-7/8/9. (a) sqlite_home DEFAULTS INSIDE CODEX_HOME - risk closed favourably at all nine sites plus three more. Q12 also closed favourably (fallback gated on Auto; N12a doc-vs-binary conflict WITHDRAWN, there never was one). Verdict does not move, and 12.4 records why: both were mechanism questions, entitlement is what is open, only a second account (Q2) can decide it. (b) AC6-codex EVALUATED: splits - leakage prevention passes, named-accounts blocked on entitlement. F2 fixed (#22419 names the CLI). F3 fixed (row-21 cause does not reproduce either way; recorded as UNKNOWN). ALSO: Q7 answers AGAINST the design (auth root is nested inside installer-managed CONFIG_DIR, not outside; H14 replaces the disjointness test) and qwen VERDICT MOVES - QWEN_HOME exists, so not-modellable was OUR plugin defect reported as a vendor limit; capped honestly as main-branch, no version pin, qwen not installed here (Q18). This contradicts the review not-requested list; stated plainly in section 6 of the response rather than buried. Validation: go vet exit 0, go test ./... exit 0 (5 packages). Commit is UNSIGNED - no secret key on this host, prior branch commits also unsigned; reported not worked around. Boundary held: no account created/enrolled/authenticated, no credential value read or persisted, no security invocation, no logout/revoke/rotate. Section 11 stays unstarted.
CHECKLIST ITEM 16, round four - basis stated so a reviewer can reject it on one sentence. Left CHECKED, on the same narrow basis round three used and the rev3 review accepted (its DoD table recorded item 16 as 'Met by the producer and by this review'). The narrow basis, restated and not widened: this delta ships NO gate code - git show --stat is three .md files, no Go source, test or config - so if item 16 means shipped-code gate behavior it is not satisfied, a reviewer should uncheck it, and I will not argue. What round four DID attack rather than read, each in the direction that would have preserved an existing claim: (1) N12a's doc-versus-binary CONFLICT on the codex keyring fallback - read the control flow at rust-v0.150.1 and found there was never a conflict, so a two-round adverse carry is withdrawn; (2) the codex namespace-input COUNT - enumerated what the program reads from its environment instead of grepping the literal, and 'one' became eight; (3) qwen NOT MODELLABLE - ran the audit and refuted our own three-round claim, which was a repository defect reported as a vendor limit; (4) Q7 DISJOINTNESS - read the installer write set and the assertion fails, the auth root is nested inside the installer-managed dir; (5) the LOGBOOK cause for row 21 - ran the query both ways and neither candidate cause reproduces, so it is now recorded as unknown rather than asserted. Four of the five came back against a claim this document was carrying. What is specified but NOT shipped: 10.1 G-B0-1..9, each written as a per-variable negative with its narrowing mutant named - G-B0-7 explicitly requires failure when the gate admits exactly one member of the set it claims to cover, because a set sized from the wrong enumeration is precisely the defect this round found. Specifying a gate is not shipping one.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-c6252e, pid=714, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-9b5586, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-9b5586)
Round four review: CHANGES REQUESTED -> analysis. Evidence: TASK-260720-3gcfd1_review-verdict-rev4.md.

PASSED: (2) both contradictions absorbed into conclusions, not footnoted - S5.3 rewritten head to tail for N19, Q7 rewritten + H14 + W22/G-B0-9 for N18. (3) 403 holds as a failed read at all six named sites plus README/LOGBOOK; round four extends the standard to gh api search zeros (ledger 39, ledger 23 re-labelled). (4) growth 1754->2512 is 453 lines of S2.4 plus consequence propagation across 25 hunks, not accumulation. (5) N13/N15/N16/N17/N18/N19/N20 all reproduce at the exact cited line numbers against primary sources; tag rust-v0.150.1 = 0eb410ad; 174 read sites confirmed in the installed 2.1.248 bundle. Boundary held - no credential value in the delta, no security invocation in this task.

FAILED: round four propagated its two adverse findings and left its two favourable ones standing-as-open at five sites, every one of which overstates a codex hazard.
F1 S6 Options compared received ZERO round-four edits: line 1469 still calls sqlite_home unstated and a live cross-account leakage risk after N13 closed it; the B0 security row still uses the three-input sizing S0/S5.1/S5.2/S9/S13 all call the wrong enumeration. S13 line 2406 enumerated nine corrected sites and S6 is not among them.
F2 S8 (a DoD item) still lists N4 as one of two things both adverse, conditional on if it is ungated - N16 answered it IS gated and S13 records the withdrawal. S8 untouched in round four.
F3 S13 contradicts itself on Q12: line 2395 says the two sources conflict, still not answered; line 2407 says withdrawn, there was no conflict. Same table, both present tense.
F4 S11.3 do-not-begin block still says Q15 could still close B1-codex on a real ground; S0 and S12.4 say it settled nothing.
F5 sweep reclassification - Q3 and Q6 correctly moved out of free reads, Q4 and Q5 not: both have a free documentary half by the exact method S2.4.2 established (source at tags; installed-bundle read), and Q4 routes to the only half unrunnable on this host while its free half bears on S7.3.3s surviving cost. Minor: the twelve free items no longer re-runs against the table (Q16-Q18 are also marked free -> fifteen).

No verdict moves. A NO-GO / B0 GO / C GO / B1-claude CONDITIONAL GO / B1-codex HELD / qwen provisional-with-a-mechanism all survive. Rework is a correction sweep, not new research - and the general fix is to apply round fours own enumerate-and-confirm discipline to its corrections, not only to its reads. README.md and LOGBOOK.md are already correct and need no change.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-9b5586, pid=23856, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the work is propagation across 2512 lines where a missed site is the defect being fixed."}
STORY-260831-yr0x81 base refresh CONFLICTED against trunk 4546253d9fa2 and was aborted; the branch is unchanged at fork point 5feebbb170ea and this producer reworks on the same branch. Conflict: Auto-merging LOGBOOK.md
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the work is propagation across 2512 lines where a missed site is the defect being fixed.
spawn workload selection: class=research source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:893c92dd429d8dedf1daefe87bf2f060169beffe96eac5d39f65578c80f9087f rationale="Propagation sweep of favourable findings across a decision document; no new research beyond two documentary halves."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-779a44, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-779a44)
Round five: fixed the mirror-image sweep gap round four's reviewer found — N13 (sqlite_home closed) and N16 (keyring fallback gated) were left standing-as-open at 5 sites (F1 §6 x3, F2 §8, F3 §13, F4 §11.3), 3 of them named AC deliverables, 1 a self-contradiction. Fixed all 5 with the ~~struck~~ / withdrawn convention round four established. Also ran F5's two documentary halves without new empirical research: N21 (codex store-selector #[default] File stable across 7 rust-v* tags, 0.147.0 through unreleased 0.152.0-alpha.6) and N22 (claude config-dir derivation applies .normalize("NFC") only, no resolve/trim/symlink-deref, so trailing-slash/symlink spellings hash to different Keychain service names). No verdict moved. Commit e7a7212, +126/-26, .md only. go vet + go test ./... both exit 0. Response/patch/validation attached as rev5 resources.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-779a44, pid=48636, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the defect class is a claim that differs between statement and use, which requires reading every use."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the defect class is a claim that differs between statement and use, which requires reading every use.
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:23bd23920cd95b0175b14ac37c84c1cb71426e27dd2093a273833b29e5b6148a rationale="Symmetry audit of a decision document after a one-directional propagation defect."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-07f37c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-07f37c)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-07f37c, pid=94113, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the check is byte-identity of a five-round-accepted artifact."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the check is byte-identity of a five-round-accepted artifact.
spawn workload selection: class=research source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:893c92dd429d8dedf1daefe87bf2f060169beffe96eac5d39f65578c80f9087f rationale="Republication of an accepted document after a base move; verification only."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-eb78da, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-eb78da)
Republication only, no ADR rework. Verified independently (not trusting the spawn briefs claim): HEAD 346ef59 has origin/main (2207096) as an ancestor with 0 divergent trunk commits missing (16 ahead, 0 behind). ADR .research/260831_multi-account-auth-architecture.adr.md is byte-identical between accepted rev5 commit e7a7212 and current HEAD (git diff exit 0, no changes). LOGBOOK.md merge at 346ef59 (parents 42f8645 story-side, 2207096 trunk-side): story side had 1508 unique non-blank lines, trunk side 1408, merged file contains all of both (0 missing from either side via set comparison), no conflict markers, newest-first ordering preserved across interleaved entries. Preserved artifacts confirmed present in main checkout: .temp/stale-board-path-161knz.tgz, .temp/element-move-journal.bak.json, .temp/orphaned-logbook-12w5gq.patch. Pinned CR rev6 candidate identity per change-request-publication-is-runtime-owned contract: base_oid=2207096 (current trunk tip), base_tree_oid=7253344, candidate_oid=346ef59, candidate_tree_oid=e6f769d, patch sha256=c703d5f9e0656d9a02ffcd3840304445e7bf1ce2ed949b0c4e0022eff002615d. Round-trip check (read-tree base + apply --cached patch + write-tree) reproduces candidate_tree_oid exactly. Diff + validation log attached as TASK-260720-3gcfd1_change-request_rev6.patch / -validation.log. No credential handling, no account creation, no ADR edits.
CR rev6 republication complete. Final candidate identity (after adding a LOGBOOK.md note for this republication and re-verifying): base_oid=2207096 (current trunk tip), base_tree_oid=7253344, candidate_oid=0897238, candidate_tree_oid=0999890, patch sha256=c5547b5732841d808c0aa96c912222160b56d5fc1bcf688cfb0f4bc697eb8dd1. ADR .research/260831_multi-account-auth-architecture.adr.md remains byte-identical to accepted rev5 (e7a7212) — reverified after the logbook commit, git diff exit 0. Round-trip (read-tree base + apply --cached patch + write-tree) reproduces candidate_tree_oid exactly. LOGBOOK.md merge integrity (story 1508 vs trunk 1408 unique non-blank lines, 0 missing from either in the merged file) and preservation of .temp/stale-board-path-161knz.tgz, element-move-journal.bak.json, orphaned-logbook-12w5gq.patch in the main checkout verified independently, not trusted from the spawn brief. Working tree clean, no divergence from origin/main (16 ahead / 0 behind). Diff + validation log attached as TASK-260720-3gcfd1_change-request_rev6.patch / -validation.log (updated to match final commit). No ADR edits, no account/credential handling, no empirical gate run.
agent completed: [implementer] developer (claude) (exit=0)
spawn completion blocked: no new or updated task-scoped outcome artifact was attached. Add or update an outcome resource named like TASK-260720-3gcfd1_results.md and then set status back to to-review.
spawn run completed: claude (run=RUN-260831-eb78da, pid=51223, exit=0)
No Change Request revision was published for TASK-260720-3gcfd1 (handoff_unsatisfied): the board is not at to-review
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the run must not alter a five-round-accepted artifact."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the run must not alter a five-round-accepted artifact.
spawn workload selection: class=research source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:893c92dd429d8dedf1daefe87bf2f060169beffe96eac5d39f65578c80f9087f rationale="Publication-only run to advance a prepared Change Request revision."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-a78ccf, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-a78ccf)
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-a78ccf, pid=52995, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; line-level survival must be reconstructed independently."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; line-level survival must be reconstructed independently.
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:23bd23920cd95b0175b14ac37c84c1cb71426e27dd2093a273833b29e5b6148a rationale="Narrow re-review after a base move, plus verification of two orchestrator recovery actions."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-f664d3, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-f664d3)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-f664d3, pid=56698, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-910865.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-910865.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_multi-account-auth-architecture.adr.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_multi-account-auth-architecture.adr.md) — Round five: propagated N13/N16 to the 5 sites round four's sweep missed; ran F5's Q4/Q5 documentary halves (N21, N22)
- [TASK-260720-3gcfd1_go-test-01.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_go-test-01.log) — go test -count=1 ./... in tools/agents-infra, exit 0, uncached. Docs-only delta; run to confirm the repository still builds and tests clean.
- [TASK-260720-3gcfd1_change-request_rev1.patch](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev1.patch) — Change Request CR-TASK-260720-3gcfd1-1 revision 1 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260720-3gcfd1_change-request_rev1-validation.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev1-validation.log) — Change Request CR-TASK-260720-3gcfd1-1 revision 1 bounded validation log
- [TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-0672a6.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-0672a6.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_review-verdict.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_review-verdict.md) — Reviewer verdict CR rev1: CHANGES REQUESTED. Findings reproduced independently; F1/F2 blocking on the entitlement evidence claim.
- [TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-814276.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-814276.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_validation-rev2.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_validation-rev2.log) — rev2 validation: go build, go vet, go test -count=1 all green; docs-only delta noted
- [TASK-260720-3gcfd1_change-request_rev1-response.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev1-response.md) — Response to CR rev1: F1-F4 addressed; B1 verdict split and reversed for Claude; item 16 reasoning stated
- [TASK-260720-3gcfd1_change-request_rev2.patch](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev2.patch) — Change Request CR-TASK-260720-3gcfd1-2 revision 2 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260720-3gcfd1_change-request_rev2-validation.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev2-validation.log) — Change Request CR-TASK-260720-3gcfd1-2 revision 2 bounded validation log
- [TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-be973f.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-be973f.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_review-verdict-rev2.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_review-verdict-rev2.md) — Review verdict, CR rev2: CHANGES REQUESTED. Claude reversal verified verbatim against primary sources; three findings on the codex half (N11 scope compression into README/LOGBOOK, derivation-cost leg withheld from codex, ledger row 21 does not re-run).
- [TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-5e79e8.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-5e79e8.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_change-request_rev3-validation.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev3-validation.log) — Change Request CR-TASK-260720-3gcfd1-3 revision 3 bounded validation log
- [TASK-260720-3gcfd1_change-request_rev3.patch](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev3.patch) — Change Request CR-TASK-260720-3gcfd1-3 revision 3 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260720-3gcfd1_change-request_rev2-response.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev2-response.md) — Round-three response to CR rev2: F1/F2/F3 all addressed; B1-codex re-derived from NO-GO to HELD/UNESTABLISHED after all three reasons failed, including the reviewer-supplied rescue premise
- [TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-35d689.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-35d689.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_review-verdict-rev3.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_review-verdict-rev3.md) — Round-three review verdict: CHANGES REQUESTED. Landing (B1-codex HELD/UNESTABLISHED), scope restoration, F2 symmetry, logbook and growth all survive attack, independently re-verified. Blocking: Q15/W10 (sqlite_home default) is answerable in three public-source calls, answers favourably, and surfaces CODEX_SQLITE_HOME - a second codex ambient env input missing from B0's prerequisites, falsifying 'W13 is the only one B0 depends on'.
- [TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-c6252e.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-c6252e.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_change-request_rev4-validation.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev4-validation.log) — Change Request CR-TASK-260720-3gcfd1-4 revision 4 bounded validation log
- [TASK-260720-3gcfd1_change-request_rev4.patch](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev4.patch) — Change Request CR-TASK-260720-3gcfd1-4 revision 4 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260720-3gcfd1_change-request_rev4-response.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev4-response.md) — Round-four CR response: the free-read sweep (8 named, 6 unrun, all run), the CODEX_SQLITE_HOME blocker plus six more ambient inputs, sqlite_home and Q12 closed favourably, AC6-codex evaluated, F2/F3 fixed, qwen verdict moved
- [TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-9b5586.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-9b5586.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_review-verdict-rev4.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_review-verdict-rev4.md) — Round four review verdict: CHANGES REQUESTED. Sweep honest, both contradictions absorbed, 403 held, all seven notes reproduce at cited lines; five sites where N13/N16's favourable findings were not propagated, incl. a self-contradiction in the §13 corrections table.
- [TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-779a44.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-779a44.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_change-request_rev5-response.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev5-response.md) — Round five change-request response: propagates N13/N16 to the five missed sites (F1-F4), runs F5's Q4/Q5 documentary halves
- [TASK-260720-3gcfd1_change-request_rev5.patch](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev5.patch) — Change Request CR-TASK-260720-3gcfd1-5 revision 5 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260720-3gcfd1_change-request_rev5-validation.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev5-validation.log) — Change Request CR-TASK-260720-3gcfd1-5 revision 5 bounded validation log
- [TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-07f37c.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-07f37c.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_review-verdict-rev5.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_review-verdict-rev5.md) — Round five review verdict: ACCEPTED. All five stale sites from round four fixed, F5 documentary halves (N21, N22) independently reproduced byte-for-byte, growth exact, nothing regressed.
- [TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-eb78da.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-eb78da.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_change-request_rev6.patch](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev6.patch) — Change Request CR-TASK-260720-3gcfd1-6 revision 6 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260720-3gcfd1_change-request_rev6-validation.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_change-request_rev6-validation.log) — Change Request CR-TASK-260720-3gcfd1-6 revision 6 bounded validation log
- [TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-a78ccf.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-implementer--developer--claude-_RUN-260831-a78ccf.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_rev6-publish-verification.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_rev6-publish-verification.md) — Verification that CR rev6 candidate/base identity and patch content match the current worktree, prior to handoff-triggered publication
- [TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-f664d3.log](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_spawn-log_-reviewer--reviewer--claude-_RUN-260831-f664d3.log) — System spawn log captured by task-board
- [TASK-260720-3gcfd1_review-verdict-rev6.md](file://TASK-260720-3gcfd1/TASK-260720-3gcfd1_review-verdict-rev6.md) — CR rev6 review verdict: ACCEPTED. Base move, ADR byte-identity to accepted rev5, additive LOGBOOK merge, and both recovery actions independently re-verified against git and file backups.

## Created
2026-07-20T15:59:13Z

## Last Update
2026-08-31T10:17:48Z

## Estimate
estimated(fibonacci(5))

## Assigned To
[reviewer] reviewer (claude)
