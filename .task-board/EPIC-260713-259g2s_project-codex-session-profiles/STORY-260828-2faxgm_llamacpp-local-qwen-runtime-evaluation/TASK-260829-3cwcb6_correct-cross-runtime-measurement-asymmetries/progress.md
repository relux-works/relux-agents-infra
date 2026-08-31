## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260828-2wcrph

## Blocks
- TASK-260829-3k4qrc

## Checklist
- [x] Reasoning-delta field naming handled for both runtimes so decode and TTFT clock the same boundary
- [x] Memory accounting sees mmap-loaded weights, or the metric is declared unmeasurable for that runtime rather than scored
- [x] Audit for measurement defects biased AGAINST llama.cpp, reported even if none are found
- [x] Every corrected metric carries a production-entry negative that refuses the old reading
- [x] Directional bias of every remaining known limitation stated explicitly in the record
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
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Removes the remaining directional bias from the production gate's own instrumentation and establishes the direction of every residual limitation, which the decision and its article both depend on."}
spawn selection rationale for gpt-5.6-sol/high: Removes the remaining directional bias from the production gate's own instrumentation and establishes the direction of every residual limitation, which the decision and its article both depend on.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-c27b0e, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-c27b0e)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-c27b0e, pid=49834, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Review of the instrumentation the decision rests on, where the central question is whether refusing an invalid memory metric without supplying a valid one leaves the owner's first-class criterion undecidable."}
spawn selection rationale for gpt-5.6-sol/high: Review of the instrumentation the decision rests on, where the central question is whether refusing an invalid memory metric without supplying a valid one leaves the owner's first-class criterion undecidable.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-f96cbf, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-f96cbf)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-f96cbf, pid=34172, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Supplies the valid memory accounting that makes the owner's first-class criterion decidable for both runtimes, and moves the MTP directional limitation from prose into the records the article will be built from."}
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Supplies the valid memory accounting that makes the owner first-class criterion decidable for both runtimes, and moves the MTP directional limitation from prose into the records the article will be built from."}
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Supplies the valid memory accounting that makes the first-class criterion decidable for both runtimes, and moves the MTP directional limitation from prose into the records the article is built from."}
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Supplies valid memory accounting so the first-class criterion is decidable for both runtimes, and moves the MTP directional limitation into the records."}
spawn selection rationale for gpt-5.6-sol/high: Supplies valid memory accounting so the first-class criterion is decidable for both runtimes, and moves the MTP directional limitation into the records.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-1b256a, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-1b256a)
Rework: both runtimes now score peak_resident_memory_upper_bound_bytes from exact Mach footprint plus a conservative vmmap resident mapped-file upper component across warm-up/scenario/soak/process windows. Absent/read-failed/malformed/partial states fail closed; raw components and semantics are record-carried. MTP-off direction is explicitly against llama.cpp in both records and decision. Validation: focused memory tests 0; benchmark tests 0; Release suite 400/32 exit 0; final Release production smoke 0 failures exit 0; Xcode Release build exit 0; swift-format, shellcheck, git diff check exit 0. Outcome: TASK-260829-3cwcb6_rework-results.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-1b256a, pid=85749, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Second-round review whose acceptance question is symmetry: an upper bound for the challenger against an exact figure for the incumbent would mirror the very bias this task removes."}
spawn selection rationale for gpt-5.6-sol/high: Second-round review whose acceptance question is symmetry: an upper bound for the challenger against an exact figure for the incumbent would mirror the very bias this task removes.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-86d73a, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-86d73a)
Revision 2 changes requested. Mach-only narrowing mutant survived all 400 tests and the production memory smoke section; decoded raw components can also forge a Mach-only composite. See TASK-260829-3cwcb6_review-verdict-rev2.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-86d73a, pid=35763, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Binds the new memory metric to what the decision actually consumes and makes a decoded record validate against its own raw components; both are unbound-evidence defects rather than collection defects."}
spawn selection rationale for gpt-5.6-sol/high: Binds the new memory metric to what the decision actually consumes and makes a decoded record validate against its own raw components; both are unbound-evidence defects rather than collection defects.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-58ea35, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-58ea35)
Revision 3 rework: RuntimeMemoryComponents now re-derives Mach + mapped bytes before RuntimeBenchmark.decide can score decoded evidence. Permanent negative forges both derived fields back to Mach-only and proves the consumer blocks. Production smoke binds every scenario/process baseline/candidate delta to generated composites. Final Release suite 401 tests/32 suites exit 0; clean smoke retry exit 0; SwiftPM and Xcode Release builds, swift-format, shellcheck and git diff check exit 0. Exact Mach-only mutant smoke exit 1 on candidate decision 13648328 versus composite 24763234; its other 14 failures are the known unrelated attestation-close cascade and do not support the kill. See TASK-260829-3cwcb6_rework-rev3-results.md and attached raw logs.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-58ea35, pid=95253, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Third-round review confirming the memory metric is now bound to the decision and validated after decode, with the mutant kill separated from an unrelated failure cascade."}
spawn selection rationale for gpt-5.6-sol/high: Third-round review confirming the memory metric is now bound to the decision and validated after decode, with the mutant kill separated from an unrelated failure cascade.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-2d170c, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-2d170c)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-2d170c, pid=89797, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260829-3cwcb6_spawn-log_-implementer--developer--codex-_RUN-260829-c27b0e.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_spawn-log_-implementer--developer--codex-_RUN-260829-c27b0e.log) — System spawn log captured by task-board
- [TASK-260829-3cwcb6_measurement-audit.md](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_measurement-audit.md) — Cross-runtime measurement corrections, directional audit, adversarial self-review, production-entry negatives, and validation exit codes
- [TASK-260829-3cwcb6_change-request_rev1.patch](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_change-request_rev1.patch) — Change Request CR-TASK-260829-3cwcb6-1 revision 1 candidate patch (repository_delta=present, 12 changed paths)
- [TASK-260829-3cwcb6_spawn-log_-reviewer--reviewer--codex-_RUN-260829-f96cbf.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_spawn-log_-reviewer--reviewer--codex-_RUN-260829-f96cbf.log) — System spawn log captured by task-board
- [TASK-260829-3cwcb6_review-verdict.md](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_review-verdict.md) — Reviewer changes-requested verdict: memory axis remains undecidable, MTP direction absent from records, exact-tree tests and narrowing mutants
- [TASK-260829-3cwcb6_spawn-log_-implementer--developer--codex-_RUN-260829-1b256a.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_spawn-log_-implementer--developer--codex-_RUN-260829-1b256a.log) — System spawn log captured by task-board
- [TASK-260829-3cwcb6_rework-results.md](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_rework-results.md) — Rework implementation, directional-bias audit, production-entry negatives, red-to-green attempts, and validation exit codes
- [TASK-260829-3cwcb6_swift-test-release.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_swift-test-release.log) — Release Swift test log: 400 tests in 32 suites, exit 0
- [TASK-260829-3cwcb6_xcodebuild-release.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_xcodebuild-release.log) — macOS arm64 Xcode Release build log: BUILD SUCCEEDED, exit 0
- [TASK-260829-3cwcb6_change-request_rev2.patch](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_change-request_rev2.patch) — Change Request CR-TASK-260829-3cwcb6-2 revision 2 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260829-3cwcb6_change-request_rev2-validation.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_change-request_rev2-validation.log) — Change Request CR-TASK-260829-3cwcb6-2 revision 2 bounded validation log
- [TASK-260829-3cwcb6_spawn-log_-reviewer--reviewer--codex-_RUN-260829-86d73a.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_spawn-log_-reviewer--reviewer--codex-_RUN-260829-86d73a.log) — System spawn log captured by task-board
- [TASK-260829-3cwcb6_review-verdict-rev2.md](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_review-verdict-rev2.md) — Revision 2 reviewer verdict: changes requested; Mach-only narrowing mutant and decoded-component forgery reproduced
- [TASK-260829-3cwcb6_spawn-log_-implementer--developer--codex-_RUN-260829-58ea35.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_spawn-log_-implementer--developer--codex-_RUN-260829-58ea35.log) — System spawn log captured by task-board
- [TASK-260829-3cwcb6_rework-rev3-results.md](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_rework-rev3-results.md) — Revision 3 decoded-composite fix, production decision binding, honest mutant kill, and validation exit codes
- [TASK-260829-3cwcb6_swift-test-release-rev3.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_swift-test-release-rev3.log) — Revision 3 final Release suite: 401 tests in 32 suites, exit 0
- [TASK-260829-3cwcb6_benchmark-gate-smoke-clean-rev3.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_benchmark-gate-smoke-clean-rev3.log) — Revision 3 clean production smoke retry: zero failures, exit 0
- [TASK-260829-3cwcb6_benchmark-gate-smoke-mach-only-mutant.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_benchmark-gate-smoke-mach-only-mutant.log) — Mach-only narrowing mutant killed at production decision-to-record binding; exit 1, with unrelated flake failures explicitly separated
- [TASK-260829-3cwcb6_xcodebuild-release-rev3.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_xcodebuild-release-rev3.log) — Revision 3 macOS arm64 Xcode Release build: BUILD SUCCEEDED, exit 0
- [TASK-260829-3cwcb6_forged-components-red.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_forged-components-red.log) — Pre-fix negative: decoded Mach-only composite was accepted; focused test exit 1 with three issues
- [TASK-260829-3cwcb6_forged-components-green.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_forged-components-green.log) — Post-fix negative: decoded Mach-only composite blocked by decision consumer; focused test exit 0
- [TASK-260829-3cwcb6_benchmark-gate-smoke-attestation-flake-rev3.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_benchmark-gate-smoke-attestation-flake-rev3.log) — First revision 3 production smoke attempt: exit 1 with 14 known attestation-close cascade failures; targeted timing and memory sections passed but run is not counted as clean
- [TASK-260829-3cwcb6_change-request_rev3.patch](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_change-request_rev3.patch) — Change Request CR-TASK-260829-3cwcb6-3 revision 3 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260829-3cwcb6_change-request_rev3-validation.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_change-request_rev3-validation.log) — Change Request CR-TASK-260829-3cwcb6-3 revision 3 bounded validation log
- [TASK-260829-3cwcb6_spawn-log_-reviewer--reviewer--codex-_RUN-260829-2d170c.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_spawn-log_-reviewer--reviewer--codex-_RUN-260829-2d170c.log) — System spawn log captured by task-board
- [TASK-260829-3cwcb6_review-verdict-rev3.md](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_review-verdict-rev3.md) — Revision 3 accepted reviewer verdict with independent forged-components and Mach-only production-entry attacks
- [TASK-260829-3cwcb6_review-rev3-forged-components.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_review-rev3-forged-components.log) — Reviewer rerun of decoded Mach-only forged-components negative: 1 test passed
- [TASK-260829-3cwcb6_review-rev3-swift-test-release.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_review-rev3-swift-test-release.log) — Reviewer full exact-tree Release suite: 401 tests in 32 suites passed
- [TASK-260829-3cwcb6_review-rev3-smoke-pristine.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_review-rev3-smoke-pristine.log) — Reviewer exact-tree production smoke: exit 0 with zero failures
- [TASK-260829-3cwcb6_review-rev3-smoke-mach-only-mutant.log](file://TASK-260829-3cwcb6/TASK-260829-3cwcb6_review-rev3-smoke-mach-only-mutant.log) — Reviewer Mach-only narrowing mutant production smoke: exit 1 on the sole decision-to-record memory mismatch

## Created
2026-08-29T10:39:11Z

## Last Update
2026-08-29T15:17:54Z

## Assigned To
[reviewer] reviewer (codex)
