## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(3))

## Blocked By
- TASK-260827-qyebv8

## Blocks
- TASK-260827-2v13w8

## Checklist
- [x] Deterministic fault injection kills or invalidates the Swift generation worker and is reproducible from a recorded command
- [x] Health endpoint returns HTTP 503 while the generation worker is dead, verified by captured response and status code
- [x] Healthy control run on the same build returns HTTP 200 for the same endpoint
- [x] Model-harness detects the failure and performs the configured supervised recovery, with harness log evidence
- [x] Regression is expressed as a repeatable check, not a one-off manual observation
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
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Fault-injection regression on a runtime whose health semantics were just hardened; needs careful reasoning about what constitutes proven worker death versus a merely slow response, on the same codebase this lineage built."}
spawn selection rationale for claude-opus-5/high: Fault-injection regression on a runtime whose health semantics were just hardened; needs careful reasoning about what constitutes proven worker death versus a merely slow response, on the same codebase this lineage built.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260827-f67365, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260827-f67365)
Plan: carry the mlx-lm dead-generation-thread health regression into the MLX Swift runtime.

Swift has no thread that can silently die (errors propagate out of a structured task), so the carried semantic is worker *invalidation*: a generation failure whose signature means the MLX Metal backend — not the request — is broken. Source incident: BUG-260827-1jhv2g, `[metal::malloc] Resource limit (499000) exceeded`.

1. Contract library gains GenerationWorkerHealth (classify + readiness transition) and HealthReport; RuntimeReadiness gains .generationWorkerFailed. /health 503 {"status":"unavailable"} matches the upstream mlx-lm patch wording.
2. Router/RuntimeState are the production call sites; runtime emits a generation_worker_unavailable supervision marker so model-harness fatal_output_substrings restarts it.
3. Deterministic fault seam: serve --fault-inject-generation-error <text>, mirroring upstream mlx-lm which injected RuntimeError("boom") into its own generation loop.
4. E2E on the real binary against the cached 261MB mlx-community/Qwen1.5-0.5B-Chat-4bit model (no 29GB load needed): healthy control 200, fault run 200 then 503, harness restart + recovery, and a NEGATIVE run where a request-scoped error keeps /health at 200 and triggers no restart.
Ready for review.

WHAT SHIPPED
The mlx-lm dead-generation-thread regression (BUG-260827-1jhv2g / BUG-260827-2tul5n) now has a home in the MLX Swift acceptance suite. It does not port literally — Swift errors propagate out of a structured task instead of silently killing a worker — so it lands as invalidation: the runtime condemns itself when a generation fails with a signature naming the backend rather than the request.

The Swift prototype had the same observable bug for a different reason: Router.complete caught the error, answered 500, and left readiness at .ready, so /health kept answering 200 for a runtime whose backend was gone.

New: Sources/MLXSwiftRuntimeContract/GenerationWorkerHealth.swift (classifier + readiness transition + HealthReport + pinned supervision marker), Tests/.../GenerationWorkerHealthTests.swift, scripts/dead-generation-smoke.sh. RuntimeReadiness gained .generationWorkerFailed so /v1/models — what the launcher actually polls — stops advertising too. Production call sites: RuntimeState.recordGenerationFailure, Router.health, Router.complete, RuntimeHTTPHandler.sendStream.

EVIDENCE (all exit 0)
- swift test -c release: 116 tests / 11 suites
- xcrun swift-format lint: clean
- xcodebuild Release: BUILD SUCCEEDED
- scripts/dead-generation-smoke.sh: 35 checks, 0 failures, real Release binary under real model-harness against the cached 261MB Qwen1.5-0.5B-Chat-4bit
- scripts/lifecycle-smoke.sh re-run: 17 checks, 0 failures
- go build/vet + go test ./... -count=1 (configured worktree validation): all ok

WOULD IT HAVE FAILED
Four mutants, all caught. Three on the classifier (condemn-all, narrowed, health-returns-200) redden the contract suite. The fourth is the one that matters: deleting the single line `await recordGenerationFailure(error)` from Router.complete and rebuilding Release makes the smoke exit 1 with 11 failures and reproduces the original incident verbatim — /health answering 200 with the worker condemned. The check is bound to the production call site, not to a helper.

The negative phase is what makes the rest mean anything: a request-scoped failure must return 500 and leave /health at 200, the model advertised, no marker, no restart. A runtime condemning itself on any error would be a worse bug than the one being fixed and would pass every other phase.

TWO THINGS REVIEWERS SHOULD KNOW
1. Health-503 and supervised restart cannot be observed in one process. With fatal_output_substrings attached, model-harness kills the runtime within milliseconds of the marker hitting stdout — correct, and it destroys the 503 window. The smoke runs the same fault twice, unsupervised then supervised, so each phase measures one property instead of measuring which won the race.
2. model-harness does not forward signals to the runtime it spawns. Signalling the harness alone orphans a live listener; an early revision of the smoke passed three phases against the leftover control process before I caught it. Teardown now signals the process group and every phase refuses to start on an occupied port.

SCOPE KEPT
Python mlx-lm remains the default local runtime and rollback path. examples/model-harness.prototype.toml gains the supervision policy but is still an example, not an installed config. Uncommitted per version_control.confirm=true.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260827-f67365, pid=36852, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Independent-provider review of a self-condemnation health contract where the dangerous failure mode is over-condemning; a different model family is less likely to accept the producer's own framing of what counts as backend versus request failure."}
spawn selection rationale for gpt-5.6-sol/xhigh: Independent-provider review of a self-condemnation health contract where the dangerous failure mode is over-condemning; a different model family is less likely to accept the producer's own framing of what counts as backend versus request failure.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260827-79a047, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260827-79a047)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260827-79a047, pid=55466, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Rework of one blocking classifier-breadth finding on code this lineage owns: narrow the fatal signature to Metal-allocator evidence and prove the request-scoped neighbour stays healthy at the production entry point."}
spawn selection rationale for claude-opus-5/high: Rework of one blocking classifier-breadth finding on code this lineage owns: narrow the fatal signature to Metal-allocator evidence and prove the request-scoped neighbour stays healthy at the production entry point.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260827-289f68, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260827-289f68)
Rework of CR-TASK-260827-2h39ya-1 rev 1 (blocking P1: over-broad classifier). Ready for review.

THE FINDING, CONFIRMED
The reviewer was right and the defect was real. `invalidatingSignatures` matched `metal::malloc` and `Resource limit` as two independent substrings, so `RequestError: Resource limit for this request is 8 tokens` -- a request-scoped refusal sharing two generic words with the incident -- condemned a healthy runtime through the production entry point: /health and /v1/models to 503, engine dropped, fatal marker emitted, harness restart.

THE FIX
Signatures are now conjunctive. `BackendFailureSignature` requires ALL of its fragments; `metal-allocator-resource-limit` requires `[metal::malloc]` AND `Resource limit`. The bracketed tag is not a guess -- the vendored MLX emits `[metal::malloc] Resource limit (N) exceeded.` verbatim at mlx/backend/metal/allocator.cpp:141-144. The allocators OTHER throw at allocator.cpp:111-117 (`Attempting to allocate N bytes which is greater than the maximum allowed buffer size`) is deliberately excluded: it refuses one oversized allocation and leaves the pool intact, so carrying the allocator tag is not on its own proof the backend is gone. A signature with no fragments matches nothing rather than everything, because allSatisfy is vacuously true on an empty sequence.

WHY THE OLD NEGATIVE COVERAGE MISSED IT
Revision 1s benign case shared no words with any fatal signature. It proved the gate is not condemn-all and nothing about how wide the fatal class is -- it could not tell a correctly narrowed gate from a deleted one. `scripts/dead-generation-smoke.sh` now runs the negative phase twice: 4a benign, 4b the reviewers exact message. Contract suite gained four narrowing arguments plus three tests.

WOULD IT HAVE FAILED
Four mutants, all caught. The decisive one: rebuild Release with the bare `Resource limit` signature restored and the smoke exits 1 with 7 failures, ALL inside phase 4b, every other phase still green -- the reviewers finding reproduced at the production entry point, with the runtime emitting generation_worker_failed for a request-scoped error and model-harness restarting it twice. Contract mutants: widened (6 issues), over-narrowed via an extra required fragment (5 issues), fail-closed guard deleted (2 issues).

EVIDENCE, ALL REAL EXIT CODES
- swift test -c release: exit 0, 119 tests / 11 suites (was 116)
- xcrun swift-format lint: exit 0, no output
- xcodebuild Release: exit 0, BUILD SUCCEEDED
- scripts/dead-generation-smoke.sh, fixed build: exit 0, 45 checks / 0 failures (was 35)
- scripts/dead-generation-smoke.sh, over-broad mutant build: exit 1, 7 failures -- expected red, this is the narrowing proof
- scripts/lifecycle-smoke.sh: exit 0, 17 checks
Real Release binary, real /Users/alexis/.local/bin/model-harness, cached 261MB Qwen1.5-0.5B-Chat-4bit, port 18019.

NOT RE-RUN: the Go gates under tools/agents-infra. This delta is Swift + two shell/markdown files + LOGBOOK.md; no Go file is in it. Stated rather than silently skipped.

LOGBOOK: new entry 2242 -- A Refusal Boundary Wide Enough To Catch Its Own Neighbour, recording the regression, the root cause, and the negative-coverage lesson.

ONE THING FOR WHOEVER RUNS THE SMOKES: scripts/lifecycle-smoke.sh does not absolutize OUT the way dead-generation-smoke.sh does, so a relative OUT= makes the fixture path relative and 12 checks go red for a reason unrelated to the code. Pass an absolute OUT. Left unchanged as out of scope.

Uncommitted per version_control.confirm=true.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260827-289f68, pid=66319, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Second-round review by the provider that produced the P1 finding, so closure is judged against that reviewer's own reproduction and the opposite over-narrowing failure is checked."}
spawn selection rationale for gpt-5.6-sol/xhigh: Second-round review by the provider that produced the P1 finding, so closure is judged against that reviewer's own reproduction and the opposite over-narrowing failure is checked.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260827-7e186e, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260827-7e186e)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260827-7e186e, pid=79668, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260827-2h39ya_spawn-log_-implementer--developer--claude-_RUN-260827-f67365.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_spawn-log_-implementer--developer--claude-_RUN-260827-f67365.log) — System spawn log captured by task-board
- [TASK-260827-2h39ya_results.md](file://TASK-260827-2h39ya/TASK-260827-2h39ya_results.md) — Implementation notes, evidence table, mutant results for the dead-generation-worker health regression
- [TASK-260827-2h39ya_dead-generation-smoke.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_dead-generation-smoke.log) — End-to-end regression suite, 35 checks, exit 0, real binary under real model-harness
- [TASK-260827-2h39ya_production-mutant-smoke.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_production-mutant-smoke.log) — Same suite against a mutant that deletes the production call site: exit 1, 11 failures, original incident reproduced
- [TASK-260827-2h39ya_captured-responses.txt](file://TASK-260827-2h39ya/TASK-260827-2h39ya_captured-responses.txt) — Captured /health and /v1/models bodies across the 200 -> 503 -> 200 transition
- [TASK-260827-2h39ya_supervised-restart.txt](file://TASK-260827-2h39ya/TASK-260827-2h39ya_supervised-restart.txt) — model-harness log line naming the marker as the reason for the supervised restart
- [TASK-260827-2h39ya_condemnation-event.json](file://TASK-260827-2h39ya/TASK-260827-2h39ya_condemnation-event.json) — The generation_worker_failed event the runtime emits when it condemns its worker
- [TASK-260827-2h39ya_swift-test-release.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_swift-test-release.log) — swift test -c release: 116 tests, 11 suites, exit 0
- [TASK-260827-2h39ya_lifecycle-smoke.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_lifecycle-smoke.log) — Existing lifecycle smoke re-run as a regression check: 17 checks, exit 0
- [TASK-260827-2h39ya_change-request_rev1.patch](file://TASK-260827-2h39ya/TASK-260827-2h39ya_change-request_rev1.patch) — Change Request CR-TASK-260827-2h39ya-1 revision 1 candidate patch (repository_delta=present, 14 changed paths)
- [TASK-260827-2h39ya_spawn-log_-reviewer--reviewer--codex-_RUN-260827-79a047.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_spawn-log_-reviewer--reviewer--codex-_RUN-260827-79a047.log) — System spawn log captured by task-board
- [TASK-260827-2h39ya_review-verdict.md](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-verdict.md) — Reviewer verdict for CR revision 1: changes requested due to over-broad Resource limit classifier
- [TASK-260827-2h39ya_review-overbroad-probe.md](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-overbroad-probe.md) — Independent production probe reproducing false worker condemnation for request-scoped Resource limit error
- [TASK-260827-2h39ya_spawn-log_-implementer--developer--claude-_RUN-260827-289f68.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_spawn-log_-implementer--developer--claude-_RUN-260827-289f68.log) — System spawn log captured by task-board
- [TASK-260827-2h39ya_rework-rev2-results.md](file://TASK-260827-2h39ya/TASK-260827-2h39ya_rework-rev2-results.md) — Rework of CR rev 1: conjunctive backend-failure signatures, narrowing evidence, mutant table, real exit codes
- [TASK-260827-2h39ya_overbroad-reproduction.txt](file://TASK-260827-2h39ya/TASK-260827-2h39ya_overbroad-reproduction.txt) — Reviewer P1 reproduced at the production entry point on a rebuilt Release binary, and the same probe under the fix: condemnation+restart vs health 200
- [TASK-260827-2h39ya_dead-generation-smoke_rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_dead-generation-smoke_rev2.log) — Full regression suite on the fixed build: 45 checks, 0 failures, exit 0 (was 35 checks)
- [TASK-260827-2h39ya_overbroad-mutant-smoke.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_overbroad-mutant-smoke.log) — Same suite against a Release binary carrying the over-broad signature: exit 1, 7 failures, all inside the new phase 4b
- [TASK-260827-2h39ya_swift-test-release_rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_swift-test-release_rev2.log) — swift test -c release: 119 tests, 11 suites, exit 0
- [TASK-260827-2h39ya_lifecycle-smoke_rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_lifecycle-smoke_rev2.log) — Lifecycle smoke re-run as a regression check on the reworked build: 17 checks, exit 0
- [TASK-260827-2h39ya_change-request_rev2.patch](file://TASK-260827-2h39ya/TASK-260827-2h39ya_change-request_rev2.patch) — Change Request CR-TASK-260827-2h39ya-2 revision 2 candidate patch (repository_delta=present, 14 changed paths)
- [TASK-260827-2h39ya_spawn-log_-reviewer--reviewer--codex-_RUN-260827-7e186e.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_spawn-log_-reviewer--reviewer--codex-_RUN-260827-7e186e.log) — System spawn log captured by task-board
- [TASK-260827-2h39ya_review-verdict-rev2.md](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-verdict-rev2.md) — Round 2 reviewer verdict for CR revision 2: accepted after independent production boundary attack
- [TASK-260827-2h39ya_review-dead-generation-smoke-rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-dead-generation-smoke-rev2.log) — Reviewer rerun: real Release binary and model-harness, 45 checks, 0 failures
- [TASK-260827-2h39ya_review-swift-test-release-rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-swift-test-release-rev2.log) — Reviewer rerun: Swift contract suite, 119 tests in 11 suites, exit 0
- [TASK-260827-2h39ya_review-xcodebuild-release-rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-xcodebuild-release-rev2.log) — Reviewer rebuild of CR revision 2 macOS arm64 Release product, BUILD SUCCEEDED
- [TASK-260827-2h39ya_review-lifecycle-smoke-rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-lifecycle-smoke-rev2.log) — Reviewer rerun: existing lifecycle smoke, 17 checks, 0 failures
- [TASK-260827-2h39ya_review-scope-entrypoint-audit-rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-scope-entrypoint-audit-rev2.log) — Reviewer audit of production call sites, Go delta absence, restart evidence, and smoke counts
- [TASK-260827-2h39ya_review-wrapper-attempt1-rev2.log](file://TASK-260827-2h39ya/TASK-260827-2h39ya_review-wrapper-attempt1-rev2.log) — Transparent record of reviewer wrapper error before swift test started; superseded by successful rerun

## Created
2026-08-27T10:29:11Z

## Last Update
2026-08-27T19:59:19Z

## Assigned To
[reviewer] reviewer (codex)
