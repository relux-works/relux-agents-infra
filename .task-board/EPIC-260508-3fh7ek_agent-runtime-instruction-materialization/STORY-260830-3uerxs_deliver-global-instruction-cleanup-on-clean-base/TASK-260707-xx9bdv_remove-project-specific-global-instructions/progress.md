## Status
done

## Assigned To
[reviewer] reviewer (codex)

## Created
2026-07-07T10:12:37Z

## Last Update
2026-09-01T18:20:00Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Preserve source and installed global instruction diffs under task temp artifacts
- [x] Remove project-specific Tap2Cash/x-platform-airdrop workflow rule from global source instructions
- [x] Delete stale installed global instruction directory and run agents-infra setup global
- [x] Verify global instructions no longer contain project-specific Swipe2Cash/Tap2Cash material
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant

## Notes
Preserved source worktree diff, source/global project-specific hit lists, installed extra global instruction files, and untracked source files under ios/swipe2cash/.temp/global-instructions-reset-260707 before resetting installed global runtime.
Ran agents-infra setup global after deleting ~/.agents/.instructions and rendered global AGENTS files. Verification: agents-infra doctor global passed; global project-specific hit list is empty; installed extra instruction file list is empty; go test ./... passed from tools/agents-infra.
Follow-up review from skill-swift-relux bootstrap: kept reusable infra changes, removed the remaining swipe2cash fixture path from tools/agents-infra/main_test.go, reran go test ./... and agents-infra setup global, and verified global/project-specific hit checks are empty.
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Routing a long-parked to-review element to a verdict; the review backlog is a goal clause and these sit in stories whose worktree lease is free."}
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Next parked to-review element in a story whose lease just freed; draining the review backlog is a goal clause."}
spawn selection rationale for gpt-5.6-sol/high: Next parked to-review element in a story whose lease just freed; draining the review backlog is a goal clause.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-8b8a4f, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-8b8a4f)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-8b8a4f, pid=21595, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Rework of a parked instruction-cleanup task whose review found the required preservation incomplete and the outcome overstating it."}
spawn selection rationale for gpt-5.6-sol/high: Rework of a parked instruction-cleanup task whose review found the required preservation incomplete and the outcome overstating it.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-b1d424, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-b1d424)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-b1d424, pid=81613, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Second-round review of the instruction-cleanup rework whose first verdict found the required preservation incomplete and the outcome overstating it."}
spawn selection rationale for gpt-5.6-sol/high: Second-round review of the instruction-cleanup rework whose first verdict found the required preservation incomplete and the outcome overstating it.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_compose_failed; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=composition_command_failed; bare child launch retained
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-e6d46b, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-e6d46b)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-e6d46b, pid=65451, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Change Request carries another task's delta; the rework must isolate this task's scope while preserving the foreign work-in-progress untouched."}
spawn selection rationale for gpt-5.6-sol/high: Change Request carries another task's delta; the rework must isolate this task's scope while preserving the foreign work-in-progress untouched.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_compose_failed; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=composition_command_failed; bare child launch retained
spawn queued: [implementer] developer (codex) (run=RUN-260829-46515a, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-46515a)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-46515a, pid=96584, exit=0)
No Change Request revision was published for TASK-260707-xx9bdv (handoff_unsatisfied): the board is not at to-review
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Resumes on a clean delivery story after the orchestrator resolved the foreign-changeset planning obstacle the producer correctly refused to decide."}
spawn selection rationale for gpt-5.6-sol/high: Resumes on a clean delivery story after the orchestrator resolved the foreign-changeset planning obstacle the producer correctly refused to decide.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-29fd77, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-29fd77)
Clean-workspace rework plan: reconstruct and verify the exact removed global workflow rule from Git history and board-owned preservation evidence; preserve that source passage, its installed copy, and the four stale installed project-local instruction files in a new task-scoped bundle; move the current installed global instruction tree into the bundle before refresh; run agents-infra setup global from this Story workspace; verify all source/installed/rendered surfaces and run Go validation; publish an empty-scope Change Request via developer handoff.
Clean Story workspace rework executed at 71230fc. Preserved exactly: the full source and installed pre-removal workflow snapshots containing the one removed Tap2Cash bullet, plus all four stale installed project-local instruction files in full. New bundle TASK-260707-xx9bdv_clean-workspace-preservation-v2.tar.gz is 249715 bytes, SHA-256 7789747b840860a25db222c9ca63ac7e7db20e13dba900dc7ec3da51b620f13e; board download round-trip is byte-identical. Moved ~/.agents/.instructions intact into the artifact, ran agents-infra setup global from this workspace, and verify/doctor passed. Strict and flexible source/installed/Codex/Claude alias refusal gates passed; source-installed recursive diff passed. Observed rerun go test ./... -count=1, go vet ./..., go build ./..., gofmt empty assertion, and git diff --check all exited 0. Repository candidate is clean/empty and no old Story worktree was accessed. origin/main advanced by two non-instruction commits during validation; managed branch was left untouched and read-only upstream instruction alias gate passed.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-29fd77, pid=27114, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Reviews the instruction cleanup now delivered from a clean workspace, where the Change Request should be isolated by construction rather than by subtraction."}
spawn selection rationale for gpt-5.6-sol/high: Reviews the instruction cleanup now delivered from a clean workspace, where the Change Request should be isolated by construction rather than by subtraction.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-c779bb, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-c779bb)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-c779bb, pid=12146, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"less_or_equal ceiling: taking the top admitted claude-sonnet-5/high pair because the overlap resolution in .configs/codex-config.toml is a judgement call the reviewer will attack."}
STORY-260830-3uerxs base refresh: the Story branch was replayed onto trunk 5a78932b449f before this final-leaf producer started; the reviewed trunk OID is 5a78932b449f
spawn selection rationale for claude-sonnet-5/high: less_or_equal ceiling: taking the top admitted claude-sonnet-5/high pair because the overlap resolution in .configs/codex-config.toml is a judgement call the reviewer will attack.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:6a3e45c201ed0de4aa6693249c4ae30cdc3d96faaae29e56d9e3b85144de8711 rationale="Following rank-1 implementation recommendation: a base refresh of an already-accepted candidate with one known config overlap needs care, not novel design."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-119-g08a094c; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260902-1d962b, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260902-1d962b)
Rework complete: refreshed accepted CR onto current trunk (5a78932, was 71230fc). No repo delta needed — the actual removal landed in d1c8d7d5 back in July; this task deliverable is the preservation artifact + installed-runtime refresh, both re-verified clean on the new base. codex-config.toml staleness was a base-pin artifact from cumulative sibling-task drift, not a genuine conflict with this task intent — resolved by keeping trunk as-is. go test/go vet/setup global/verify global/doctor global all exit 0. Negative search clean on all 4 surfaces (source, installed, rendered Codex, rendered Claude). See TASK-260707-xx9bdv_v3-refresh-results.md.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260902-1d962b, pid=87862, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"less_or_equal ceiling: taking the top admitted gpt-5.6-sol/medium pair because a delta that collapsed from six paths to zero needs the strongest available review."}
spawn selection rationale for gpt-5.6-sol/medium: less_or_equal ceiling: taking the top admitted gpt-5.6-sol/medium pair because a delta that collapsed from six paths to zero needs the strongest available review.
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/medium pair_source=explicit match=recommended_rank_1 snapshot=sha256:33f2012c0b70f3ab5caea46038998966ff66745dec61f156ff5a45306f77893d rationale="Following rank-1 review recommendation; also routing to codex for provider diversity because the claude producer authored the claim under review."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-124-g5a78932; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260902-350738, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260902-350738)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260902-350738, pid=15028, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260707-xx9bdv_global-instructions-reset-summary.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_global-instructions-reset-summary.md) — Rework summary with preservation, refresh, negative-search, and validation evidence
- [TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260829-8b8a4f.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260829-8b8a4f.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_review-verdict.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_review-verdict.md) — Reviewer accepted verdict for CR revision 2 with artifact, production-surface, runtime-refresh, drift, and independent Go validation evidence
- [TASK-260707-xx9bdv_go-test-readiness-rerun-01.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_go-test-readiness-rerun-01.log) — Exact readiness production test rerun, count=3, current main
- [TASK-260707-xx9bdv_spawn-log_-implementer--developer--codex-_RUN-260829-b1d424.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-implementer--developer--codex-_RUN-260829-b1d424.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_global-instructions-reset-artifact.tar.gz](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_global-instructions-reset-artifact.tar.gz) — Self-contained source and installed global instruction preservation bundle with SHA-256 manifest
- [TASK-260707-xx9bdv_go-test-all-01.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_go-test-all-01.md) — Uncached full Go suite evidence with exact Story worktree revision and exit code
- [TASK-260707-xx9bdv_change-request_rev1.patch](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_change-request_rev1.patch) — Change Request CR-TASK-260707-xx9bdv-1 revision 1 candidate patch (repository_delta=present, 8 changed paths)
- [TASK-260707-xx9bdv_change-request_rev1-validation.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_change-request_rev1-validation.log) — Change Request CR-TASK-260707-xx9bdv-1 revision 1 bounded validation log
- [TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260829-e6d46b.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260829-e6d46b.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_review-verdict-rev1-cycle2.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_review-verdict-rev1-cycle2.md) — Second-cycle reviewer changes-requested verdict: CR revision 1 contains cross-task fast-profile delta
- [TASK-260707-xx9bdv_spawn-log_-implementer--developer--codex-_RUN-260829-46515a.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-implementer--developer--codex-_RUN-260829-46515a.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_scope-isolation-blocker.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_scope-isolation-blocker.md) — Cross-task Change Request isolation blocker, preserved-delta evidence, rejected forced fits, and required reroute
- [TASK-260707-xx9bdv_spawn-log_-implementer--developer--codex-_RUN-260830-29fd77.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-implementer--developer--codex-_RUN-260830-29fd77.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_clean-workspace-preservation-v2.tar.gz](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_clean-workspace-preservation-v2.tar.gz) — Clean-workspace preservation bundle with exact removed passage snapshots, four full stale project-local files, pre/post runtime trees, checksums, and validation evidence
- [TASK-260707-xx9bdv_clean-workspace-results-v2.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_clean-workspace-results-v2.md) — Exact clean-workspace rework outcome with preservation scope, runtime refresh, negative alias gates, and Go validation
- [TASK-260707-xx9bdv_change-request_rev2.patch](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_change-request_rev2.patch) — Change Request CR-TASK-260707-xx9bdv-2 revision 2 candidate patch (repository_delta=present, 6 changed paths)
- [TASK-260707-xx9bdv_change-request_rev2-validation.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_change-request_rev2-validation.log) — Change Request CR-TASK-260707-xx9bdv-2 revision 2 bounded validation log
- [TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260830-c779bb.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260830-c779bb.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_review-verdict-rev2-cycle3.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_review-verdict-rev2-cycle3.md) — Reviewer accepted verdict for CR revision 2, cycle 3, with artifact, production-surface, runtime-refresh, drift, and independent Go validation evidence
- [TASK-260707-xx9bdv_spawn-log_-implementer--developer--claude-_RUN-260902-1d962b.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-implementer--developer--claude-_RUN-260902-1d962b.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_v3-refresh-results.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_v3-refresh-results.md) — Rework: refresh accepted CR onto current trunk (5a78932), codex-config.toml overlap resolution, re-verification and validation evidence
- [TASK-260707-xx9bdv_change-request_rev3.patch](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_change-request_rev3.patch) — Change Request CR-TASK-260707-xx9bdv-3 revision 3 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260707-xx9bdv_change-request_rev3-validation.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_change-request_rev3-validation.log) — Change Request CR-TASK-260707-xx9bdv-3 revision 3 bounded validation log
- [TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260902-350738.log](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_spawn-log_-reviewer--reviewer--codex-_RUN-260902-350738.log) — System spawn log captured by task-board
- [TASK-260707-xx9bdv_review-verdict-rev3-cycle4.md](file://TASK-260707-xx9bdv/TASK-260707-xx9bdv_review-verdict-rev3-cycle4.md) — Independent reviewer verdict for CR revision 3 empty-delta refresh
