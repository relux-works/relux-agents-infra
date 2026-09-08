## Status
to-dev

## Assigned To
[reviewer] reviewer (codex)

## Created
2026-07-07T13:09:29Z

## Last Update
2026-08-30T05:36:37Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Setup installs the LLDB MCP prerequisites on macOS so a project-local lldb opt-in works after render and restart
- [x] Install is idempotent: a second run neither reinstalls nor errors
- [x] A prerequisite that cannot be installed fails clearly rather than leaving a silently broken opt-in
- [x] The install path and its exact commands are documented where an operator will find them
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
Started inline per user request. Scope: add macOS/Homebrew LLVM lldb-mcp provisioning to scripts/setup.sh and document the behavior. Existing LLDB MCP registry/code changes are already dirty in this checkout and will be preserved.
Implemented macOS lldb-mcp bootstrap in scripts/setup.sh. Source repo: ~/src/relux-works/relux-agents-infra. Setup command verified: ./scripts/setup.sh. Evidence logs: ~/src/videocall/ios/.temp/tool-readiness/lldb-mcp-agents-infra-setup-source-02.log and -03.log. Go tests: ~/src/videocall/ios/.temp/tool-readiness/lldb-mcp-agents-infra-go-test-02.log. The hook installs Homebrew llvm if needed, writes /opt/homebrew/bin/lldb-mcp managed wrapper, and supports AGENTS_INFRA_SKIP_LLDB_MCP=1.
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Last unrouted parked element in this story; the lldb MCP helper install path with acceptance criteria written from its stated intent."}
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Last unrouted parked element in this story; the lldb MCP helper install path."}
spawn selection rationale for gpt-5.6-sol/high: Last unrouted parked element in this story; the lldb MCP helper install path.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-c04eef, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-c04eef)
Reviewer changes requested: production ./setup.sh exits 0 when Homebrew and lldb-mcp are absent; a second run invokes brew install llvm again and recreates the managed wrapper. Producer evidence does not reproduce in the authoritative Story worktree, and no shell regression test binds either AC. See TASK-260707-1gnk6p_review-verdict.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-c04eef, pid=28206, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Closes a fail-open prerequisite path and a non-idempotent reinstall, both proven at the real setup entry point, on a worktree that is 63 commits behind trunk."}
spawn selection rationale for gpt-5.6-sol/high: Closes a fail-open prerequisite path and a non-idempotent reinstall, both proven at the real setup entry point, on a worktree that is 63 commits behind trunk.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-36cab9, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-36cab9)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-36cab9, pid=62043, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Second-round review of the fail-open prerequisite and non-idempotent reinstall fixes, both of which the first review proved at the real setup entry point."}
spawn selection rationale for gpt-5.6-sol/high: Second-round review of the fail-open prerequisite and non-idempotent reinstall fixes, both of which the first review proved at the real setup entry point.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-3af7ea, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-3af7ea)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-3af7ea, pid=50621, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Closes a wrapper health check that accepts an unreachable exec line as evidence of delegation."}
spawn selection rationale for gpt-5.6-sol/high: Closes a wrapper health check that accepts an unreachable exec line as evidence of delegation.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-af928c, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-af928c)
Cycle-3 rework: scripts/setup.sh now owns the managed LLDB wrapper bytes through one renderer and uses cmp against that exact output, so the reviewer forged wrapper with exit 0 before the expected exec is repaired. Added production-entry negative TestBootstrapSetupRepairsManagedLLDBMCPWrapperWithUnreachableExpectedTarget. Expected red exit 1 before fix; named test, five-test bootstrap suite, full uncached go test ./..., go vet, go build, gofmt, zsh syntax, and diff check all exit 0 after fix. Evidence: TASK-260707-1gnk6p_rework-cycle3.md. Evidence head cf21665d, 63 commits behind local main.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-af928c, pid=2021, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Third-round review of the wrapper health check; the question is whether delegation is now established by behaviour rather than by the presence of a line."}
spawn selection rationale for gpt-5.6-sol/high: Third-round review of the wrapper health check; the question is whether delegation is now established by behaviour rather than by the presence of a line.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-1e97cb, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-1e97cb)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-1e97cb, pid=67487, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Republishes the LLDB setup work from a clean delivery story so the Change Request is isolated by construction, carrying all three findings."}
spawn selection rationale for gpt-5.6-sol/high: Republishes the LLDB setup work from a clean delivery story so the Change Request is isolated by construction, carrying all three findings.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-f7c32b, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-f7c32b)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-f7c32b, pid=2861, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Third review cycle on fail-open setup logic where a forged wrapper previously passed the health check; ceiling pair chosen because the remaining risk is a narrowing mutant that a shallower pass would confirm rather than catch."}
spawn selection rationale for gpt-5.6-sol/high: Third review cycle on fail-open setup logic where a forged wrapper previously passed the health check; ceiling pair chosen because the remaining risk is a narrowing mutant that a shallower pass would confirm rather than catch.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-e8c438, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-e8c438)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-e8c438, pid=15371, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-c04eef.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-c04eef.log) — System spawn log captured by task-board
- [TASK-260707-1gnk6p_review-verdict.md](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-verdict.md) — CR revision 3 changes-requested verdict with production fail-open reproduction and three surviving narrowing mutants
- [TASK-260707-1gnk6p_spawn-log_-implementer--developer--codex-_RUN-260830-36cab9.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_spawn-log_-implementer--developer--codex-_RUN-260830-36cab9.log) — System spawn log captured by task-board
- [TASK-260707-1gnk6p_results.md](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_results.md) — Rework implementation and production-entry validation evidence on Story and current main bases
- [TASK-260707-1gnk6p_change-request_rev1.patch](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_change-request_rev1.patch) — Change Request CR-TASK-260707-1gnk6p-1 revision 1 candidate patch (repository_delta=present, 10 changed paths)
- [TASK-260707-1gnk6p_change-request_rev1-validation.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_change-request_rev1-validation.log) — Change Request CR-TASK-260707-1gnk6p-1 revision 1 bounded validation log
- [TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-3af7ea.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-3af7ea.log) — System spawn log captured by task-board
- [TASK-260707-1gnk6p_review-attack_forged-managed-wrapper.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-attack_forged-managed-wrapper.log) — Cycle-2 production-entry attack proving forged managed-wrapper bypass
- [TASK-260707-1gnk6p_spawn-log_-implementer--developer--codex-_RUN-260830-af928c.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_spawn-log_-implementer--developer--codex-_RUN-260830-af928c.log) — System spawn log captured by task-board
- [TASK-260707-1gnk6p_rework-cycle3.md](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_rework-cycle3.md) — Cycle-3 unreachable-exec negative, exact-byte wrapper fix, and direct validation exit codes
- [TASK-260707-1gnk6p_change-request_rev2.patch](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_change-request_rev2.patch) — Change Request CR-TASK-260707-1gnk6p-2 revision 2 candidate patch (repository_delta=present, 10 changed paths)
- [TASK-260707-1gnk6p_change-request_rev2-validation.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_change-request_rev2-validation.log) — Change Request CR-TASK-260707-1gnk6p-2 revision 2 bounded validation log
- [TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-1e97cb.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-1e97cb.log) — System spawn log captured by task-board
- [TASK-260707-1gnk6p_review-verdict-rev2-cycle3.md](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-verdict-rev2-cycle3.md) — CR revision 2 cycle-3 changes-requested verdict with production-entry gate attacks and sibling-scope contamination evidence
- [TASK-260707-1gnk6p_spawn-log_-implementer--developer--codex-_RUN-260830-f7c32b.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_spawn-log_-implementer--developer--codex-_RUN-260830-f7c32b.log) — System spawn log captured by task-board
- [TASK-260707-1gnk6p_clean-base-reimplementation.md](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_clean-base-reimplementation.md) — Clean-base LLDB MCP setup implementation, negative production-entry evidence, and validation exit codes
- [TASK-260707-1gnk6p_change-request_rev3.patch](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_change-request_rev3.patch) — Change Request CR-TASK-260707-1gnk6p-3 revision 3 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260707-1gnk6p_change-request_rev3-validation.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_change-request_rev3-validation.log) — Change Request CR-TASK-260707-1gnk6p-3 revision 3 bounded validation log
- [TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-e8c438.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_spawn-log_-reviewer--reviewer--codex-_RUN-260830-e8c438.log) — System spawn log captured by task-board
- [TASK-260707-1gnk6p_review-focused-tests-rev3.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-focused-tests-rev3.log) — Uncached focused production-entry LLDB setup tests on CR revision 3
- [TASK-260707-1gnk6p_review-broken-external-rev3.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-broken-external-rev3.log) — Production setup fail-open reproduction with unusable external lldb-mcp
- [TASK-260707-1gnk6p_review-mutant-arg-delegation-rev3.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-mutant-arg-delegation-rev3.log) — Focused suite passing after delegation is narrowed to the single tested argument shape
- [TASK-260707-1gnk6p_review-mutant-swallow-exit-rev3.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-mutant-swallow-exit-rev3.log) — Focused suite passing after generated wrapper swallows helper failures
- [TASK-260707-1gnk6p_review-mutant-same-inode-rewrite-rev3.log](file://TASK-260707-1gnk6p/TASK-260707-1gnk6p_review-mutant-same-inode-rewrite-rev3.log) — Focused suite passing after byte-identical wrapper is rewritten in place
