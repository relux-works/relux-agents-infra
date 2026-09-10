## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(3))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Assertion depends on the observable outcome, not on how many polls fit in a wall-clock window
- [x] Reproduce the failure deliberately under load before fixing, so the fix is proven against the real cause
- [x] Do not widen a timeout until it stops failing; state why the chosen bound is principled
- [x] A false CR-validation failure is treated as a defect: it cancels producer runs and consumes recovery attempts on correct work
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] In a managed Story worktree the candidate is left UNCOMMITTED in the worktree for the handoff to snapshot — never commit on the Story branch. A producer commit moves the branch tip off the recorded checkpoint and the handoff refuses with change_request_candidate_committed_past_checkpoint; repair with `git reset --soft <checkpoint_oid>` before completing again.
- [x] Every command, message, state, or refusal named in the AC is driven through the production entry point by a named committed test, or is declared a stated bound. Report coverage as a ratio — `n of m AC rows driven` — and name the production call site for each. Prose in place of the ratio is not evidence.
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Every gate ships at least one NARROWING mutant — the gate stays present and is weakened to admit exactly one member of the class it must reject, and a named test must fail. A delete-only mutant proves only that the gate exists and is not accepted as evidence.
- [x] A gate that inspects source text is additionally attacked by a mutant that PRESERVES the searched-for token and changes behavior, and the mutant harness executes the behavioral suite, not only the static checker.
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
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"less_or_equal ceiling: sonnet-5/high is the top admitted pair for a code change with negative-test obligations"}
spawn selection rationale for claude-sonnet-5/high: less_or_equal ceiling: sonnet-5/high is the top admitted pair for a code change with negative-test obligations
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Rank-1 implementation pair; code task in a managed Story worktree with full Go suite validation"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-140-g61d832d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-c0d114, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-c0d114)
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-c0d114, pid=17646, exit=0)
No Change Request revision was published for BUG-260830-1rths2 (handoff_unsatisfied): the board is not at to-review
spawn autonomous recovery: run RUN-260911-c0d114 queued successor RUN-260911-0aeacb (attempt 1/3, model=claude-sonnet-5): producer run RUN-260911-c0d114 remains unsatisfied: producer run RUN-260911-c0d114 published no Change Request and reached no handoff branch while BUG-260830-1rths2 is backlog: the board is not at to-review
spawn run started: [implementer] developer (claude) (run=RUN-260911-0aeacb)
Fixed both described races: removed the readiness-count file read (asserts wall-clock elapsed>=timeout instead), and made the exitAfter runtime exit immediately (removes the deadline-vs-delayed-exit race). Also found+fixed a third, related race under load-stress verification: the fake runtime pidfile write happened after expensive stdlib imports, so under heavy concurrency the production 1s deadline could fire before the pidfile was written, making assertRecordedPIDsGone fail. Fixed by writing the pidfile before the expensive imports, scoped to this test's fixture only. Verified with 145 top-level concurrent runs (45 + 100) across up to 5 concurrent processes, 0 flakes. Production readiness mutant (halved deadline) is caught by the named subtest; details + evidence in BUG-260830-1rths2_results.md. See results doc for a related pre-existing flake in an unrelated test that is out of scope here.
Checklist item 8 (AC coverage ratio): 2 of 2 AC rows driven through the production entry point. Row 1 - times_out_while_runtime_remains_alive must yield runtime_readiness_timeout via a wall-clock lower bound, not a side-effect file read: driven by RunPi -> waitPiRuntimeReady deadline branch, pi_launch_posix.go:571 (piError(runtime_readiness_timeout)), asserted in pi_test.go TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry/times_out_while_runtime_remains_alive (pi_test.go:1701-1745). Row 2 - refuses_after_owned_runtime_exits must yield runtime_exited_early deterministically, not racing two timed conditions: driven by RunPi -> waitPiRuntimeReady childWait.done/child.Signal branches, pi_launch_posix.go:569 and :573-575 (piError(runtime_exited_early)), asserted in the same test function subtest refuses_after_owned_runtime_exits. Checklist item 11 (source-text-inspecting gate mutant): not applicable — this bug fixes test-fixture timing assertions against production error codes and elapsed wall time; there is no gate here that inspects source text/tokens, so no such mutant applies. Stated bound, not a skipped check.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-0aeacb, pid=45605, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"less_or_equal ceiling: sonnet-5/high is the top admitted pair for an adversarial review"}
spawn selection rationale for claude-sonnet-5/high: less_or_equal ceiling: sonnet-5/high is the top admitted pair for an adversarial review
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:c1d45d75e280f504c00350c1bdccf0443316cf4183fffb8ceca1bdc4e46e9584 rationale="Rank-1 review pair; independent review of a 2-file concurrency-test fix with mutant evidence"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-140-g61d832d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260911-239d97, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260911-239d97)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-239d97, pid=40421, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/low","text":"less_or_equal ceiling: sonnet-5/low for a mechanical checkpoint-only run"}
spawn selection rationale for claude-sonnet-5/low: less_or_equal ceiling: sonnet-5/low for a mechanical checkpoint-only run
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/low pair_source=explicit match=not_recommended snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Consciously below rank-1: a single checkpoint command needs no reasoning; low effort is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-140-g61d832d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-74c22a, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-74c22a)
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-74c22a, pid=78073, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260830-1rths2_spawn-log_-implementer--developer--claude-_RUN-260911-c0d114.log](file://BUG-260830-1rths2/BUG-260830-1rths2_spawn-log_-implementer--developer--claude-_RUN-260911-c0d114.log) — System spawn log captured by task-board
- [BUG-260830-1rths2_spawn-log_-implementer--developer--claude-_RUN-260911-0aeacb.log](file://BUG-260830-1rths2/BUG-260830-1rths2_spawn-log_-implementer--developer--claude-_RUN-260911-0aeacb.log) — System spawn log captured by task-board
- [BUG-260830-1rths2_results.md](file://BUG-260830-1rths2/BUG-260830-1rths2_results.md) — Root cause, fix, and mutant/concurrency evidence for the readiness test flake
- [BUG-260830-1rths2_change-request_rev1.patch](file://BUG-260830-1rths2/BUG-260830-1rths2_change-request_rev1.patch) — Change Request CR-BUG-260830-1rths2-1 revision 1 candidate patch (repository_delta=present, 2 changed paths)
- [BUG-260830-1rths2_change-request_rev1-validation.log](file://BUG-260830-1rths2/BUG-260830-1rths2_change-request_rev1-validation.log) — Change Request CR-BUG-260830-1rths2-1 revision 1 bounded validation log
- [BUG-260830-1rths2_spawn-log_-reviewer--reviewer--claude-_RUN-260911-239d97.log](file://BUG-260830-1rths2/BUG-260830-1rths2_spawn-log_-reviewer--reviewer--claude-_RUN-260911-239d97.log) — System spawn log captured by task-board
- [BUG-260830-1rths2_review-verdict-rev1.md](file://BUG-260830-1rths2/BUG-260830-1rths2_review-verdict-rev1.md) — Reviewer verdict rev1: independent verification of the readiness-test-flake fix (concurrency repro, narrowing mutant, full suite)
- [BUG-260830-1rths2_spawn-log_-implementer--developer--claude-_RUN-260911-74c22a.log](file://BUG-260830-1rths2/BUG-260830-1rths2_spawn-log_-implementer--developer--claude-_RUN-260911-74c22a.log) — System spawn log captured by task-board

## Created
2026-08-30T00:06:04Z

## Last Update
2026-09-10T19:05:00Z

## Assigned To
[implementer] developer (claude)
