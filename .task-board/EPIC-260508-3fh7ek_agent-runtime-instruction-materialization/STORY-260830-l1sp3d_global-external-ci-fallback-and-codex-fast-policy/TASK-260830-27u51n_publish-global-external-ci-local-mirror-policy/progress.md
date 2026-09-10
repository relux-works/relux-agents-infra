## Status
closed

## Review
required

## Task Class
docs

## Estimate
estimated(fibonacci(3))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Versioned global workflow source contains the narrow external-CI fallback
- [x] Exact PR-head local mirror and evidence requirements are explicit
- [x] Code failures hosted status integrity review and protection remain blocking
- [x] Generated Claude and Codex instruction surfaces match the source
- [x] Repository setup or install flow refreshes runtime output without direct edits under user runtime directories
- [x] Focused instruction-generation tests and full relevant Go tests pass
- [x] Code written per task description and AC
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Use the configured Sol high ceiling for a global instruction contract whose failure could alter delivery safety across repositories"}
spawn selection rationale for gpt-5.6-sol/high: Use the configured Sol high ceiling for a global instruction contract whose failure could alter delivery safety across repositories
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-89a5ca, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-89a5ca)
Published narrow external-CI fallback in versioned workflow source. Setup parity test proves installed Claude bytes equal source and rendered Codex contains every blocking clause. Go tests: focused exit 0; internal/infra exit 0; remaining packages exit 0. Build, vet, setup global, verify global, cmp, rg, and diff-check exit 0. Broadened-policy mutant expected-red exit 1, restored focused rerun exit 0. Evidence attached in TASK-260830-27u51n_results.md and task-scoped logs.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-89a5ca, pid=93924, exit=0)
spawn autonomous recovery: run RUN-260829-89a5ca queued successor RUN-260829-28fcb5 (attempt 1/3, model=gpt-5.6-sol): Change Request construction for TASK-260830-27u51n failed: Change Request CR-TASK-260830-27u51n-1 revision 1 validation failed at command 1/2 (1-based) with exit code 1; log resource TASK-260830-27u51n_change-request_rev1-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (codex) (run=RUN-260829-28fcb5)
Recovery run resolved CR revision 1 validation failure. The previously failing model-check deadline subtest passed on direct uncached rerun; configured go test ./... -count=1 and go vet ./... both exit 0, go build exit 0. Repository ./setup.sh rebuilt the current CLI and refreshed global runtime; verify, Agents/Claude cmp, and Codex clause checks exit 0. Fresh broadened-trigger mutant exited 1 as expected and post-restore focused test exited 0. Evidence: TASK-260830-27u51n_recovery-results.md plus task-scoped recovery logs.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-28fcb5, pid=70880, exit=0)
spawn autonomous recovery: run RUN-260829-28fcb5 queued successor RUN-260829-567e6e (attempt 2/3, model=gpt-5.6-sol): Change Request construction for TASK-260830-27u51n failed: Change Request CR-TASK-260830-27u51n-2 revision 2 validation failed at command 1/2 (1-based) with exit code 1; log resource TASK-260830-27u51n_change-request_rev2-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (codex) (run=RUN-260829-567e6e)
Recovery 03: preserved CR rev2 validation exit 1 and waited for competing board-cli/agents-infra suites to drain. All five exact failing process/readiness shapes passed uncached in isolation; configured go test ./... -count=1, go vet ./..., focused setup parity test, go build ./..., gofmt, and diff-check exited 0 on unchanged candidate. Default ./setup.sh exited 1 after Homebrew upgraded LLVM to 23 and the expected lldb-mcp helper disappeared; this is recorded in LOGBOOK and attached honestly. Canonical bootstrap with documented AGENTS_INFRA_SKIP_LLDB_MCP=1, verify global, Agents/Claude byte cmp, and generated Codex policy clause checks exited 0. New outcome: TASK-260830-27u51n_recovery-03-results.md plus task-scoped recovery-03 logs. No production code or timing threshold was changed.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-567e6e, pid=40497, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Independently verify the narrow external-CI fallback cannot launder repository failures or hosted status and that source-to-installed instruction parity is exact"}
spawn selection rationale for gpt-5.6-sol/high: Independently verify the narrow external-CI fallback cannot launder repository failures or hosted status and that source-to-installed instruction parity is exact
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-137cad, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-137cad)
CR revision 3 changes requested: focused parity test has a Claude bypass path. Removing INSTRUCTIONS_WORKFLOW.md from .instructions/INSTRUCTIONS.md leaves the test green, while the Codex master control mutant fails. Rework must assert the full Claude CLAUDE.md -> INSTRUCTIONS.md -> INSTRUCTIONS_WORKFLOW.md production chain and preserve current-origin/main post-integration setup/parity validation. Evidence: TASK-260830-27u51n_review-verdict.md
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-137cad, pid=98790, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Close the reviewer-proven Claude include-chain bypass with a production-wiring assertion and named negative mutant while preserving the narrow policy text"}
spawn selection rationale for gpt-5.6-sol/high: Close the reviewer-proven Claude include-chain bypass with a production-wiring assertion and named negative mutant while preserving the narrow policy text
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-0fd3d0, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-0fd3d0)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-0fd3d0, pid=26250, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Use the configured Sol high ceiling to independently attack the repaired Claude include chain and distinguish deterministic policy gates from concurrent Pi fixture flakes"}
spawn selection rationale for gpt-5.6-sol/high: Use the configured Sol high ceiling to independently attack the repaired Claude include chain and distinguish deterministic policy gates from concurrent Pi fixture flakes
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-977f67, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-977f67)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-977f67, pid=244, exit=0)
2026-09-11 goal audit: closed as superseded duplicate. Global external-CI local-mirror fallback policy landed via STORY-260830-11fnea / PR #23 (4270549); candidate text byte-identical to main INSTRUCTIONS_WORKFLOW.md. Supersession chain recorded in TASK-260830-27u51n_supersession.md.

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260829-89a5ca.log](file://TASK-260830-27u51n/TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260829-89a5ca.log) — System spawn log captured by task-board
- [TASK-260830-27u51n_results.md](file://TASK-260830-27u51n/TASK-260830-27u51n_results.md) — Developer outcome and validation summary
- [TASK-260830-27u51n_go-test-infra-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-test-infra-01.log) — Full internal/infra Go test evidence
- [TASK-260830-27u51n_go-test-remaining-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-test-remaining-01.log) — Remaining Go package test evidence
- [TASK-260830-27u51n_expected-red-mutant-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_expected-red-mutant-01.log) — Negative evidence for broadened external-CI fallback
- [TASK-260830-27u51n_setup-global-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_setup-global-01.log) — Canonical global setup evidence
- [TASK-260830-27u51n_verify-global-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_verify-global-01.log) — Installed runtime verification evidence
- [TASK-260830-27u51n_change-request_rev1.patch](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev1.patch) — Change Request CR-TASK-260830-27u51n-1 revision 1 candidate patch (repository_delta=present, 4 changed paths)
- [TASK-260830-27u51n_change-request_rev1-validation.log](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev1-validation.log) — Change Request CR-TASK-260830-27u51n-1 revision 1 bounded validation log
- [TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260829-28fcb5.log](file://TASK-260830-27u51n/TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260829-28fcb5.log) — System spawn log captured by task-board
- [TASK-260830-27u51n_recovery-results.md](file://TASK-260830-27u51n/TASK-260830-27u51n_recovery-results.md) — Recovery implementation, validation, setup parity, mutant, and anomaly summary
- [TASK-260830-27u51n_go-test-all-recovery-02.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-test-all-recovery-02.log) — Recovery full uncached Go suite evidence
- [TASK-260830-27u51n_expected-red-mutant-recovery-02.log](file://TASK-260830-27u51n/TASK-260830-27u51n_expected-red-mutant-recovery-02.log) — Recovery broadened-policy expected-red mutant and post-restore control
- [TASK-260830-27u51n_setup-runtime-recovery-02.log](file://TASK-260830-27u51n/TASK-260830-27u51n_setup-runtime-recovery-02.log) — Repository bootstrap, runtime repair, verification, and installed parity evidence
- [TASK-260830-27u51n_recovery-failing-test-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_recovery-failing-test-01.log) — Direct uncached rerun of the test that failed CR revision 1
- [TASK-260830-27u51n_focused-policy-test-recovery-02.log](file://TASK-260830-27u51n/TASK-260830-27u51n_focused-policy-test-recovery-02.log) — Focused source-to-output setup test evidence
- [TASK-260830-27u51n_go-vet-all-recovery-02.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-vet-all-recovery-02.log) — Configured Go vet gate evidence
- [TASK-260830-27u51n_go-build-all-recovery-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-build-all-recovery-01.log) — Full Go build evidence
- [TASK-260830-27u51n_tool-readiness-recovery-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_tool-readiness-recovery-01.log) — Recovery tool readiness evidence
- [TASK-260830-27u51n_change-request_rev2.patch](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev2.patch) — Change Request CR-TASK-260830-27u51n-2 revision 2 candidate patch (repository_delta=present, 4 changed paths)
- [TASK-260830-27u51n_change-request_rev2-validation.log](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev2-validation.log) — Change Request CR-TASK-260830-27u51n-2 revision 2 bounded validation log
- [TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260829-567e6e.log](file://TASK-260830-27u51n/TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260829-567e6e.log) — System spawn log captured by task-board
- [TASK-260830-27u51n_recovery-03-results.md](file://TASK-260830-27u51n/TASK-260830-27u51n_recovery-03-results.md) — Recovery implementation, exact isolated reruns, full gates, setup parity, and anomaly summary
- [TASK-260830-27u51n_isolated-model-check-deadline-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-model-check-deadline-03.log) — Uncached exact rerun of CR rev2 model-check tool-pids failure
- [TASK-260830-27u51n_isolated-direct-rpc-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-direct-rpc-03.log) — Uncached exact rerun of CR rev2 direct RPC side-effect failure
- [TASK-260830-27u51n_isolated-readiness-timeout-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-readiness-timeout-03.log) — Uncached exact rerun of CR rev2 readiness 503 timeout failure
- [TASK-260830-27u51n_isolated-readiness-runtime-exit-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-readiness-runtime-exit-03.log) — Uncached exact rerun of CR rev2 early runtime exit failure
- [TASK-260830-27u51n_isolated-shutdown-escalation-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-shutdown-escalation-03.log) — Uncached exact rerun of CR rev2 shutdown escalation failure
- [TASK-260830-27u51n_go-test-all-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-test-all-recovery-03.log) — Recovery full uncached configured Go suite evidence
- [TASK-260830-27u51n_go-vet-all-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-vet-all-recovery-03.log) — Recovery configured Go vet gate evidence
- [TASK-260830-27u51n_focused-policy-test-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_focused-policy-test-recovery-03.log) — Focused source-to-installed Claude and rendered Codex parity test evidence
- [TASK-260830-27u51n_go-build-all-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-build-all-recovery-03.log) — Recovery full Go build evidence
- [TASK-260830-27u51n_setup-runtime-failed-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_setup-runtime-failed-recovery-03.log) — Truthful failed default setup evidence after Homebrew LLVM 23 removed expected lldb-mcp helper
- [TASK-260830-27u51n_setup-runtime-skip-lldb-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_setup-runtime-skip-lldb-recovery-03.log) — Successful canonical setup using documented optional LLDB MCP skip
- [TASK-260830-27u51n_verify-global-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_verify-global-recovery-03.log) — Installed global runtime verification evidence
- [TASK-260830-27u51n_installed-parity-recovery-03.log](file://TASK-260830-27u51n/TASK-260830-27u51n_installed-parity-recovery-03.log) — Installed Agents Claude byte parity and generated Codex clause evidence
- [TASK-260830-27u51n_change-request_rev3.patch](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev3.patch) — Change Request CR-TASK-260830-27u51n-3 revision 3 candidate patch (repository_delta=present, 4 changed paths)
- [TASK-260830-27u51n_change-request_rev3-validation.log](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev3-validation.log) — Change Request CR-TASK-260830-27u51n-3 revision 3 bounded validation log
- [TASK-260830-27u51n_spawn-log_-reviewer--reviewer--codex-_RUN-260830-137cad.log](file://TASK-260830-27u51n/TASK-260830-27u51n_spawn-log_-reviewer--reviewer--codex-_RUN-260830-137cad.log) — System spawn log captured by task-board
- [TASK-260830-27u51n_review-verdict.md](file://TASK-260830-27u51n/TASK-260830-27u51n_review-verdict.md) — Reviewer verdict for CR revision 4: broadened external-CI trigger mutant survives
- [TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260830-0fd3d0.log](file://TASK-260830-27u51n/TASK-260830-27u51n_spawn-log_-implementer--developer--codex-_RUN-260830-0fd3d0.log) — System spawn log captured by task-board
- [TASK-260830-27u51n_rework-04-results.md](file://TASK-260830-27u51n/TASK-260830-27u51n_rework-04-results.md) — Developer rework closing Claude include-chain bypass with negative mutant and validation evidence
- [TASK-260830-27u51n_expected-red-claude-include-mutant-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_expected-red-claude-include-mutant-04.log) — Expected-red Claude workflow include mutant (exit 1)
- [TASK-260830-27u51n_focused-claude-chain-final-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_focused-claude-chain-final-04.log) — Final focused production setup and Claude/Codex wiring test (exit 0)
- [TASK-260830-27u51n_go-test-all-final-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-test-all-final-04.log) — Exact-final-candidate full Go suite truthful timing failure (exit 1)
- [TASK-260830-27u51n_go-test-infra-final-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-test-infra-final-04.log) — Exact-final-candidate infra package truthful pinned-Pi fixture failure (exit 1)
- [TASK-260830-27u51n_isolated-model-check-deadline-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-model-check-deadline-04.log) — Isolated uncached model-check deadline rerun (exit 0)
- [TASK-260830-27u51n_isolated-readiness-bounds-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-readiness-bounds-04.log) — Isolated uncached readiness bounds rerun (exit 0)
- [TASK-260830-27u51n_isolated-direct-rpc-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_isolated-direct-rpc-04.log) — Isolated uncached pinned-Pi direct-RPC rerun (exit 0)
- [TASK-260830-27u51n_go-vet-all-final-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-vet-all-final-04.log) — Final Go vet evidence (exit 0)
- [TASK-260830-27u51n_go-build-all-final-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_go-build-all-final-04.log) — Final Go build evidence (exit 0)
- [TASK-260830-27u51n_setup-global-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_setup-global-04.log) — Canonical repository bootstrap with documented LLDB MCP skip (exit 0)
- [TASK-260830-27u51n_verify-global-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_verify-global-04.log) — Installed global runtime verification (exit 0)
- [TASK-260830-27u51n_installed-runtime-parity-final-04.log](file://TASK-260830-27u51n/TASK-260830-27u51n_installed-runtime-parity-final-04.log) — Installed Agents/Claude byte parity and Claude/Codex production wiring (exit 0)
- [TASK-260830-27u51n_change-request_rev4.patch](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev4.patch) — Change Request CR-TASK-260830-27u51n-4 revision 4 candidate patch (repository_delta=present, 4 changed paths)
- [TASK-260830-27u51n_change-request_rev4-validation.log](file://TASK-260830-27u51n/TASK-260830-27u51n_change-request_rev4-validation.log) — Change Request CR-TASK-260830-27u51n-4 revision 4 bounded validation log
- [TASK-260830-27u51n_spawn-log_-reviewer--reviewer--codex-_RUN-260830-977f67.log](file://TASK-260830-27u51n/TASK-260830-27u51n_spawn-log_-reviewer--reviewer--codex-_RUN-260830-977f67.log) — System spawn log captured by task-board
- [TASK-260830-27u51n_reviewer-focused-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_reviewer-focused-01.log) — Reviewer focused Setup-path control, exit 0
- [TASK-260830-27u51n_reviewer-broadened-policy-mutant-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_reviewer-broadened-policy-mutant-01.log) — Surviving broadened-trigger mutant: focused gate incorrectly exits 0
- [TASK-260830-27u51n_reviewer-go-test-all-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_reviewer-go-test-all-01.log) — Reviewer full uncached Go suite, exit 1 in readiness timing test
- [TASK-260830-27u51n_reviewer-readiness-repro-01.log](file://TASK-260830-27u51n/TASK-260830-27u51n_reviewer-readiness-repro-01.log) — Exact failing readiness production-entry test passed 5/5 isolated
- [TASK-260830-27u51n_supersession.md](file://TASK-260830-27u51n/TASK-260830-27u51n_supersession.md) — Fresh-current-trunk replacement ownership after surviving semantic mutant

## Created
2026-08-29T22:59:15Z

## Last Update
2026-09-11T12:39:38Z

## Assigned To
[reviewer] reviewer (codex)
