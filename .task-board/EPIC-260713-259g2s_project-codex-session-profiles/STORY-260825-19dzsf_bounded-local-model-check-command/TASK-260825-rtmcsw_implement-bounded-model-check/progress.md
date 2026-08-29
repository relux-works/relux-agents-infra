## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- (none)

## Blocks
- TASK-260825-39ycg2

## Checklist
- [x] Production CLI resolves a configured target and invokes the existing managed runtime lifecycle
- [x] Default deadline and overrides terminate the full process group and prove cleanup
- [x] Raw JSONL/stderr artifacts and sanitized stable summaries are persisted
- [x] Expected tool/text assertions and malformed/failed event cases return meaningful non-zero results
- [x] Production-entrypoint positive and negative tests pass without external model downloads
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
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/xhigh","text":"Lifecycle-sensitive CLI work needs strong implementation and negative-test coverage; gpt-5.6-sol/xhigh is the highest admitted pair and avoids unnecessary internal delegation."}
spawn selection rationale for gpt-5.6-sol/xhigh: Lifecycle-sensitive CLI work needs strong implementation and negative-test coverage; gpt-5.6-sol/xhigh is the highest admitted pair and avoids unnecessary internal delegation.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-27-gb3cb845; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260825-cca110, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260825-cca110)
Implementation plan: 1) map canonical target resolution and managed Pi/runtime lifecycle; 2) add production CLI command and production-entrypoint tests first; 3) implement bounded execution, evidence parsing/sanitization, expectations, and cleanup; 4) run targeted tests, full Go suite, vet/lint-equivalent and build; 5) attach task-scoped evidence and hand off to review.
Implemented production agents-infra model-check via the configured canonical target and existing infra.RunPi lifecycle. Added bounded readiness/session context, process-group cleanup evidence, protected raw artifacts, sanitized JSON/text summaries, stable exit codes, independent expectation negatives, malformed/tool-failure cases, and production-entrypoint cleanup tests. Current-tree gates: production suite exit 0; full go test exit 0; race exit 0; vet exit 0; go build exit 0; Windows cross-build exit 0; gofmt exit 0. Attached task-scoped outcome and test logs; logbook records the process-group finding.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260825-cca110, pid=93169, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"Independent cross-provider review is warranted for a process-lifecycle and evidence-sanitization command; Opus 5/high can attack the timeout, cleanup, and secret-leak gates without the cost of maximum effort."}
spawn selection rationale for claude-opus-5/high: Independent cross-provider review is warranted for a process-lifecycle and evidence-sanitization command; Opus 5/high can attack the timeout, cleanup, and secret-leak gates without the cost of maximum effort.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-27-gb3cb845; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260825-29b9f4, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260825-29b9f4)
Review verdict: changes requested (RUN-260825-29b9f4, CR rev1). Implementation is sound and the full module suite is green (go test . 64.3s, go test ./internal/... 80.7s, model-check suite 8/8). Mutation testing killed 5 of 7 narrowing mutants (deadline mid-session, malformed JSONL line>1, expect-text, tool-failure threshold, secret sanitizer). Two survived: (F1) --prompt is never proven to reach the provider - replacing opts.Prompt with a fixed literal keeps the whole suite green, and the fixture discards the request body; (F2) cleanup_confirmed is an attestation with no negative test - forcing isConfirmedCleanupState to return true keeps the suite green, and the exit-1-on-unconfirmed-cleanup branch never executes. (F3) Three refusals have zero coverage: deadline range, output-dir overwrite refusal, non-managed target - all three verified to work today by manual probe, so this is a regression-guard gap. (F4) minor: fabricated duration_ms=1, misleading event_stream_error on a never-launched run, README omits the exit-code contract. Full detail and mutant table in TASK-260825-rtmcsw_review-verdict.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260825-29b9f4, pid=20164, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/xhigh","text":"Focused rework must kill two reviewer-demonstrated lifecycle mutants and add refusal guards while preserving the accepted design; gpt-5.6-sol/xhigh is the strongest admitted implementation pair for this evidence-critical correction."}
spawn selection rationale for gpt-5.6-sol/xhigh: Focused rework must kill two reviewer-demonstrated lifecycle mutants and add refusal guards while preserving the accepted design; gpt-5.6-sol/xhigh is the strongest admitted implementation pair for this evidence-critical correction.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-27-gb3cb845; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260825-70dbf2, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260825-70dbf2)
Rework resolves review F1-F4: prompt delivery is asserted at the provider request, cleanup false states select exit 1, deadline overwrite and non-managed refusals have production tests, never-launched summaries avoid invented stream errors, measured duration is honest, and README documents exit codes. Five narrowed mutants exited 1. Full main and internal suites, vet, native build, Windows cross-build, and diff check exited 0. Evidence: TASK-260825-rtmcsw_rework-results.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260825-70dbf2, pid=25634, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"Revision 2 claims to kill the exact surviving prompt-delivery and cleanup-attestation mutants; an independent Opus 5/high rerun is the most cost-effective way to verify those gates and the added refusal coverage."}
spawn selection rationale for claude-opus-5/high: Revision 2 claims to kill the exact surviving prompt-delivery and cleanup-attestation mutants; an independent Opus 5/high rerun is the most cost-effective way to verify those gates and the added refusal coverage.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-27-gb3cb845; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260825-75ed44, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260825-75ed44)
Review of CR-TASK-260825-rtmcsw-2 rev 2 (RUN-260825-75ed44): CHANGES REQUESTED. Evidence: TASK-260825-rtmcsw_review-verdict.md. Blocking: (1) TestModelCheckProductionEntrypoint/deadline_override_bounds_runtime_readiness is timing-flaky - 6/6 fail cold, 3/3 fail under CPU load, passes warm; the fixture writes runtime-pids after import http.server + Popen and races the 400ms deadline. Product behaves correctly (status timed_out, runtime cleanup confirmed, exit 2) - fixture defect. (2) processGroupCleanupState (pi_launch_posix.go:488), the only production source of the cleanup_confirmed attestation, has no test: a mutant returning constant "confirmed" survives the whole suite; the existing unit test only hand-feeds synthetic states to the evaluator. Fix with a direct unit test - live pgid => failed, reaped pid => confirmed. (3) main.go structural interface{ExitCode() int} also matches *exec.ExitError, so agents-infra codex/claude/target now exit with the child code (verified 42 vs base 1) and 255 for signal death (os.Exit(-1), outside documented range). Narrow to *infra.ModelCheckFailure. Non-blocking: --approve auto-approval undocumented; no test pins --expect-text to the final assistant response; --deadline 0 silently defaults. Positive controls: narrowing mutant that drops SIGKILL escalation IS caught; redaction and prompt-forwarding proofs are real. Build/vet/gofmt/windows+linux cross-compile clean; full suites green warm.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260825-75ed44, pid=47358, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/xhigh","text":"The third cycle is a narrow but high-impact compatibility repair: constrain exit-code typing, prove process-group cleanup from live OS state, and remove a cold-start race; gpt-5.6-sol/xhigh is justified by the regression risk and mutant-based evidence gate."}
spawn selection rationale for gpt-5.6-sol/xhigh: The third cycle is a narrow but high-impact compatibility repair: constrain exit-code typing, prove process-group cleanup from live OS state, and remove a cold-start race; gpt-5.6-sol/xhigh is justified by the regression risk and mutant-based evidence gate.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-27-gb3cb845; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260825-ef435b, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260825-ef435b)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260825-ef435b, pid=63733, exit=0)
Review-cycle safety note: RUN-260825-75ed44 left eight orphaned busy-loop CPU-load fixture processes in PGID 59922; the orchestrator terminated PIDs 59924-59931 and verified the group empty. Future review must not use unbounded synthetic load loops and must verify cleanup of every reviewer-created process before handoff.
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"Revision 3 repairs a cross-command exit-code regression and adds direct live-process cleanup evidence; Opus 5/high should independently retest those exact production boundaries while avoiding the prior review fixture leak."}
spawn selection rationale for claude-opus-5/high: Revision 3 repairs a cross-command exit-code regression and adds direct live-process cleanup evidence; Opus 5/high should independently retest those exact production boundaries while avoiding the prior review fixture leak.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-27-gb3cb845; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260825-bce1e8, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260825-bce1e8)
Reviewer RUN-260825-bce1e8 accepted CR revision 3. Seven narrowing mutants applied to the candidate tree and reverted; each was caught by a named test (deadline lower bound, final-response accumulation, malformed-JSONL skip, DeadlineExceeded reporting, secret redaction, SIGKILL escalation, structural exit-code interface). Worktree tree OID recomputed after the experiments = bf1f208724002da517e7ff9f89bdafe4c0650e9d, identical to the candidate. Independent runs: build/vet/gofmt clean, darwin+linux+windows cross-build ok, TestModelCheckProductionEntrypoint PASS 14/14 in 13.63s (not skipped), go test ./internal/... ok, go test . ok. Verdict artifact: TASK-260825-rtmcsw_review-verdict-rev3.md. Process note: a resource update overwrote the cycle-2 verdict payload before accept_cr refused that name; its reasoning survives in the RUN-260825-29b9f4 spawn log.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260825-bce1e8, pid=86436, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260825-rtmcsw_spawn-log_-implementer--developer--codex-_RUN-260825-cca110.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_spawn-log_-implementer--developer--codex-_RUN-260825-cca110.log) — System spawn log captured by task-board
- [TASK-260825-rtmcsw_results.md](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_results.md) — Implementation, test matrix, exact gates, failures, and finding
- [TASK-260825-rtmcsw_production-tests.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_production-tests.log) — Production-entrypoint positive and narrowed negative test evidence
- [TASK-260825-rtmcsw_full-go-test.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_full-go-test.log) — Current-tree full Go suite evidence
- [TASK-260825-rtmcsw_change-request_rev1.patch](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_change-request_rev1.patch) — Change Request CR-TASK-260825-rtmcsw-1 revision 1 candidate patch (repository_delta=present, 9 changed paths)
- [TASK-260825-rtmcsw_spawn-log_-reviewer--reviewer--claude-_RUN-260825-29b9f4.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_spawn-log_-reviewer--reviewer--claude-_RUN-260825-29b9f4.log) — System spawn log captured by task-board
- [TASK-260825-rtmcsw_review-verdict.md](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_review-verdict.md) — Reviewer verdict for CR revision 3: ACCEPTED, with 7-mutant gate-attack table and independent verification evidence
- [TASK-260825-rtmcsw_spawn-log_-implementer--developer--codex-_RUN-260825-70dbf2.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_spawn-log_-implementer--developer--codex-_RUN-260825-70dbf2.log) — System spawn log captured by task-board
- [TASK-260825-rtmcsw_rework-results.md](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_rework-results.md) — Developer rework, exact green gates, and narrowed-mutant evidence
- [TASK-260825-rtmcsw_change-request_rev2.patch](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_change-request_rev2.patch) — Change Request CR-TASK-260825-rtmcsw-2 revision 2 candidate patch (repository_delta=present, 10 changed paths)
- [TASK-260825-rtmcsw_spawn-log_-reviewer--reviewer--claude-_RUN-260825-75ed44.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_spawn-log_-reviewer--reviewer--claude-_RUN-260825-75ed44.log) — System spawn log captured by task-board
- [TASK-260825-rtmcsw_spawn-log_-implementer--developer--codex-_RUN-260825-ef435b.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_spawn-log_-implementer--developer--codex-_RUN-260825-ef435b.log) — System spawn log captured by task-board
- [TASK-260825-rtmcsw_rework-cycle3-results.md](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_rework-cycle3-results.md) — Developer cycle 3 fixes, exact green gates, mutant kills, and fixture-cleanup directive
- [TASK-260825-rtmcsw_change-request_rev3.patch](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_change-request_rev3.patch) — Change Request CR-TASK-260825-rtmcsw-3 revision 3 candidate patch (repository_delta=present, 10 changed paths)
- [TASK-260825-rtmcsw_spawn-log_-reviewer--reviewer--claude-_RUN-260825-bce1e8.log](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_spawn-log_-reviewer--reviewer--claude-_RUN-260825-bce1e8.log) — System spawn log captured by task-board
- [TASK-260825-rtmcsw_review-verdict-rev3.md](file://TASK-260825-rtmcsw/TASK-260825-rtmcsw_review-verdict-rev3.md) — Reviewer verdict for CR revision 3: ACCEPTED, with 7-mutant gate-attack table, independent verification, and 6/6 cold-start stability re-check

## Created
2026-08-25T08:52:10Z

## Last Update
2026-08-25T10:55:03Z

## Assigned To
[reviewer] reviewer (claude)
