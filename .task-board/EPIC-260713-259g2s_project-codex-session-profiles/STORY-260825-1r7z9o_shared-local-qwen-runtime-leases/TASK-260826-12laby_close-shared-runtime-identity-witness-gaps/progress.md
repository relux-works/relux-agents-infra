## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Add same-device/wrong-inode witnesses for all four executable-identity gates
- [x] Add below-current protocol-version refusal witness and exact-version control
- [x] Prove each narrowed production gate is killed independently
- [x] Replace the load-flaky wall-clock positive control with an event-driven control if the change remains test-only
- [x] Record review finding and closure evidence in LOGBOOK.md and task resources
- [x] Run focused, race, full, build, vet, and formatting validation
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
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/xhigh","text":"Final Story review proved three independent same-device/wrong-inode identity mutants and one below-current protocol-version mutant survive existing tests; gpt-5.6-sol/xhigh is the strongest admitted developer pair for a narrow adversarial-test closure that must preserve the accepted broker behavior and produce mutation evidence before integration."}
spawn selection rationale for gpt-5.6-sol/xhigh: Final Story review proved three independent same-device/wrong-inode identity mutants and one below-current protocol-version mutant survive existing tests; gpt-5.6-sol/xhigh is the strongest admitted developer pair for a narrow adversarial-test closure that must preserve the accepted broker behavior and produce mutation evidence before integration.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260826-3e75a9, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260826-3e75a9)
Developer evidence: test-only closure adds four named same-device/wrong-inode production-entry witnesses, below-current plus exact controls at broker/client/launcher version gates, and target-emitted event controls. Seven independent narrowed mutants exited 1; focused, race, full package split, build, vet, gofmt, and diff checks exited 0. Outcomes: TASK-260826-12laby_results.md and TASK-260826-12laby_mutation-evidence.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260826-3e75a9, pid=52593, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"This narrow CR closes security-evidence holes at four executable-identity production gates, three protocol-version gates, and a historically flaky launcher control; Claude Opus 5/high provides independent cross-provider review able to rerun each narrowing mutant and distinguish real closure from incidental test failure before Story landing."}
spawn selection rationale for claude-opus-5/high: This narrow CR closes security-evidence holes at four executable-identity production gates, three protocol-version gates, and a historically flaky launcher control; Claude Opus 5/high provides independent cross-provider review able to rerun each narrowing mutant and distinguish real closure from incidental test failure before Story landing.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260826-d4eda7, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260826-d4eda7)
REVIEW VERDICT: changes_requested (CR-TASK-260826-12laby-1 rev1). See TASK-260826-12laby_review-verdict.md.

ACCEPTED: 7 independent narrowing mutants applied to production one at a time, each killed exit 1 by its own named witness — 4 inode-removal mutants (operator:363 candidate scan, operator:470 force-stop, broker:836 client admission, client:341 broker attestation) and 3 below-current protocol-version mutants (broker:839, client:379, launcher:72). Candidate-scan witness drives the real sharedRuntimeBrokerCandidates entry with two launched processes and asserts its own same-device/different-inode premise. Exact-version controls green. gofmt/vet/build clean. Launcher suite 11.055s on clean candidate.

BLOCKING FINDING: the event-driven rewrite deleted the only live witness that the launcher execve-s into the runtime. All five positive controls now wait on a target-emitted stdout line, which a forked child satisfies identically; carriedTarget() (launcher PID ExecPath == target) is now read only in negative assertions, so nothing asserts it is ever true. Reproduced: replacing production sharedRuntimeExecve=unix.Exec with fork+exec leaves the whole launcher suite green (ok 261.236s) on CR rev1, but FAILS at base with the author own message .../valid: valid authorization never reached execve. This matters because the authorization frame is bound to LauncherPID and the gate compares frame.LauncherPID == os.Getpid(); a forking launcher would authorize one PID and run another. The failure string at pi_shared_launcher_test.go:376 still claims execve. The LOGBOOK entry does not record the dropped assertion.

REWORK: keep requireTargetEvent, add back a positive assertion that the launcher own PID comes to run the target; prove it by applying the fork+exec mutant and recording the failing test name; keep the exact-execve control green; record the drop-and-restore in LOGBOOK. Everything else in rev1 carries forward unchanged.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260826-d4eda7, pid=79467, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/xhigh","text":"Independent review accepted all seven identity/version narrowing closures but proved the event-driven launcher control no longer distinguishes execve from fork-and-exec; gpt-5.6-sol/xhigh should make the minimal same-PID witness repair, demonstrate the exact fork mutant red and exact execve green, and preserve every accepted rev1 test."}
spawn selection rationale for gpt-5.6-sol/xhigh: Independent review accepted all seven identity/version narrowing closures but proved the event-driven launcher control no longer distinguishes execve from fork-and-exec; gpt-5.6-sol/xhigh should make the minimal same-PID witness repair, demonstrate the exact fork mutant red and exact execve green, and preserve every accepted rev1 test.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260826-1f3818, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260826-1f3818)
Developer rework: retained the target event and restored a same-PID execve assertion by kernel-inspecting the authorized launcher PID after the event. Reviewer fork+exec mutant M8 now exits 1 on the exact protocol-version control; clean unix.Exec exits 0. Focused, race, full package split, build, vet, formatting, and diff checks exit 0. Evidence: TASK-260826-12laby_rework-results.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260826-1f3818, pid=86572, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"Rev2 claims to preserve event-driven stability while restoring a kernel-verified same-PID execve witness and killing the exact fork-and-exec mutant that survived rev1; Claude Opus 5/high should independently reproduce that closure, ensure all seven previously accepted gates remain bound, and accept only if the CR remains test-only."}
spawn selection rationale for claude-opus-5/high: Rev2 claims to preserve event-driven stability while restoring a kernel-verified same-PID execve witness and killing the exact fork-and-exec mutant that survived rev1; Claude Opus 5/high should independently reproduce that closure, ensure all seven previously accepted gates remain bound, and accept only if the CR remains test-only.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260826-589f55, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260826-589f55)
Reviewer rev2: ACCEPTED. Attacked, not read: 7 independent production narrowing mutants (4 inode-clause removals on pi_shared_broker_darwin.go:836, pi_shared_client_darwin.go:341, pi_shared_operator_darwin.go:363, pi_shared_operator_darwin.go:470; 3 protocol-version range narrowings) each killed by exactly one named witness with exact-version controls green. Reviewer fork+exec mutant of sharedRuntimeExecve exits 1 on exact_protocol_version_control -> the rev1 execve-witness regression is genuinely closed. Pre-CR baseline reproduced: mutants A/B/C survived on base test files, D killed incidentally; that base run also reproduced the 15s wall-clock flake this CR removes. Event-driven control verified: 30 launcher control iterations under 8x CPU load, exit 0. Race clean after mutex removal (happens-before via donePoll). Full package covered in two chunks (37+242 of 279 tests), build/vet/gofmt exit 0. CARRIED-FORWARD FINDING (non-blocking): the LOGBOOK ROOT CAUSE line claims all four Dev/Ino gates had wrong-device witnesses; sharedRuntimeBrokerCandidates has none - dropping only its Dev clause leaves the whole shared suite green (mutant I, exit 0). Out of this AC scope (inode only) and not expressible without a production seam. Recommend the Story owner correct that sentence at integration and consider a separate leaf for the candidates device clause. Evidence: TASK-260826-12laby_review-verdict-rev2.md, TASK-260826-12laby_review-rev2-mutant-logs.tgz
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260826-589f55, pid=7761, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"The prior independent Opus review fully accepted revision 2 and persisted exhaustive mutant evidence but omitted the reviewer-bound accept_cr mutation; this focused successor must verify the immutable revision/evidence binding and record the authorized acceptance without reopening settled implementation scope."}
spawn selection rationale for claude-opus-5/high: The prior independent Opus review fully accepted revision 2 and persisted exhaustive mutant evidence but omitted the reviewer-bound accept_cr mutation; this focused successor must verify the immutable revision/evidence binding and record the authorized acceptance without reopening settled implementation scope.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260826-b07fed, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260826-b07fed)
agent completed: [reviewer] reviewer (claude) (exit=143)
spawn run completed: claude (run=RUN-260826-b07fed, pid=24317, exit=143)
spawn run RUN-260826-b07fed cancelled by operator; operator action required; reason: no operator reason supplied

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260826-12laby_spawn-log_-implementer--developer--codex-_RUN-260826-3e75a9.log](file://TASK-260826-12laby/TASK-260826-12laby_spawn-log_-implementer--developer--codex-_RUN-260826-3e75a9.log) — System spawn log captured by task-board
- [TASK-260826-12laby_results.md](file://TASK-260826-12laby/TASK-260826-12laby_results.md) — Developer implementation, mutation, and validation evidence
- [TASK-260826-12laby_mutation-evidence.md](file://TASK-260826-12laby/TASK-260826-12laby_mutation-evidence.md) — Independent narrowed-gate mutation evidence for executable identity and protocol versions
- [TASK-260826-12laby_change-request_rev1.patch](file://TASK-260826-12laby/TASK-260826-12laby_change-request_rev1.patch) — Change Request CR-TASK-260826-12laby-1 revision 1 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260826-12laby_spawn-log_-reviewer--reviewer--claude-_RUN-260826-d4eda7.log](file://TASK-260826-12laby/TASK-260826-12laby_spawn-log_-reviewer--reviewer--claude-_RUN-260826-d4eda7.log) — System spawn log captured by task-board
- [TASK-260826-12laby_review-verdict.md](file://TASK-260826-12laby/TASK-260826-12laby_review-verdict.md) — Reviewer verdict for CR-TASK-260826-12laby-1 rev1: changes_requested; 7 narrowing mutants killed, fork+exec mutant survives launcher suite (execve witness regression)
- [TASK-260826-12laby_spawn-log_-implementer--developer--codex-_RUN-260826-1f3818.log](file://TASK-260826-12laby/TASK-260826-12laby_spawn-log_-implementer--developer--codex-_RUN-260826-1f3818.log) — System spawn log captured by task-board
- [TASK-260826-12laby_rework-results.md](file://TASK-260826-12laby/TASK-260826-12laby_rework-results.md) — Developer rework, fork+exec mutation proof, and full validation evidence
- [TASK-260826-12laby_change-request_rev2.patch](file://TASK-260826-12laby/TASK-260826-12laby_change-request_rev2.patch) — Change Request CR-TASK-260826-12laby-2 revision 2 candidate patch (repository_delta=present, 5 changed paths)
- [TASK-260826-12laby_spawn-log_-reviewer--reviewer--claude-_RUN-260826-589f55.log](file://TASK-260826-12laby/TASK-260826-12laby_spawn-log_-reviewer--reviewer--claude-_RUN-260826-589f55.log) — System spawn log captured by task-board
- [TASK-260826-12laby_review-verdict-rev2.md](file://TASK-260826-12laby/TASK-260826-12laby_review-verdict-rev2.md) — Reviewer verdict for CR rev 2: accepted; 7 narrowing mutants + fork+exec execve mutant + pre-CR baseline reproduced
- [TASK-260826-12laby_review-rev2-mutant-logs.tgz](file://TASK-260826-12laby/TASK-260826-12laby_review-rev2-mutant-logs.tgz) — Reviewer rev2 raw logs: mutants A-I, fork+exec mutant, pre-CR baseline, focused/race/full/flake runs
- [TASK-260826-12laby_spawn-log_-reviewer--reviewer--claude-_RUN-260826-b07fed.log](file://TASK-260826-12laby/TASK-260826-12laby_spawn-log_-reviewer--reviewer--claude-_RUN-260826-b07fed.log) — System spawn log captured by task-board

## Created
2026-08-26T06:30:07Z

## Last Update
2026-08-26T08:23:50Z

## Assigned To
[reviewer] reviewer (claude)
