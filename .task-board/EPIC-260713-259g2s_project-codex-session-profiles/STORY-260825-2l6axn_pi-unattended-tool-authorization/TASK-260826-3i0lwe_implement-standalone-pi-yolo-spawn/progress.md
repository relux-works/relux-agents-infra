## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(13))

## Blocked By
- TASK-260825-lsojra

## Blocks
- TASK-260826-h934tg

## Checklist
- [x] Define a board-agnostic standalone Pi spawn/headless-session config and CLI contract
- [x] Own and validate the exact unattended Pi arguments; refuse conflicting caller flags
- [x] Disable extension discovery and prove replacement extensions cannot shadow authorized tools
- [x] Keep project trust separate from tool authorization and require no human prompt or stdin
- [x] Preserve configured local-Qwen reasoning medium in the standalone launch
- [x] Reuse the shared runtime lease broker while isolating each Pi process, session, and state
- [x] Prove two concurrent workers reuse one runtime and final release does not stop a live peer
- [x] Add no-model Pi tool side-effect and direct-RPC bypass negative tests
- [x] Document standalone usage and explicitly defer task-board adapter integration
- [x] Record the standalone-now board-later decision in LOGBOOK.md
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
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/xhigh","text":"This is a security-sensitive standalone unattended launcher spanning CLI/config validation, Pi tool and extension authorization boundaries, typed process cleanup, and real two-worker one-runtime concurrency; gpt-5.6-sol/xhigh is the strongest admitted developer pair for implementing the complete board-agnostic primitive with adversarial tests while preserving interactive behavior."}
STORY-260825-2l6axn base refresh CONFLICTED against trunk 197238cff1ca and was aborted; the branch is unchanged at fork point e70f953969d4 and this producer reworks on the same branch. Conflict: Auto-merging LOGBOOK.md
spawn selection rationale for gpt-5.6-sol/xhigh: This is a security-sensitive standalone unattended launcher spanning CLI/config validation, Pi tool and extension authorization boundaries, typed process cleanup, and real two-worker one-runtime concurrency; gpt-5.6-sol/xhigh is the strongest admitted developer pair for implementing the complete board-agnostic primitive with adversarial tests while preserving interactive behavior.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260826-e9a9bc, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260826-e9a9bc)
agent completed: [implementer] developer (codex) (exit=-1)
spawn run completed: codex (run=RUN-260826-e9a9bc, pid=68860, exit=-1)
RUN-260826-e9a9bc was operator-cancelled after editing two files on stale base despite merge directives. Exact binary patch preserved as TASK-260826-3i0lwe_pre-merge-worker.patch (sha256 ed12ad4837c8efe56ad9ef157242b33243c28f2613b049ac904bcc936e102f65). Recovery order is mandatory: inspect patch; restore only pi_config.go and project_config.go to a68e428 after verifying patch digest; merge current main 197238c into the Story branch with additive LOGBOOK resolution; prove main ancestry and shared-runtime files present; selectively reapply useful patch hunks; then implement. No repo-wide restore/reset.
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/xhigh","text":"The first implementation run was cancelled after two task-owned edits landed on a stale pre-broker base; this successor must perform surgical patch preservation and path restore, merge current main with additive LOGBOOK resolution, verify shared-runtime ancestry, then complete the security-sensitive standalone YOLO launcher and its concurrency/authorization tests."}
STORY-260825-2l6axn base refresh SKIPPED: the managed workspace holds uncommitted work, so there was no clean checkpoint branch to replay onto trunk 197238cff1ca; the branch is unchanged at fork point e70f953969d4
spawn selection rationale for gpt-5.6-sol/xhigh: The first implementation run was cancelled after two task-owned edits landed on a stale pre-broker base; this successor must perform surgical patch preservation and path restore, merge current main with additive LOGBOOK resolution, verify shared-runtime ancestry, then complete the security-sensitive standalone YOLO launcher and its concurrency/authorization tests.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260826-6398e4, max_parallel=20)
spawn run RUN-260826-e9a9bc cancelled by operator; operator action required; reason: Cancelled before further implementation because the run edited pi_config.go and project_config.go while Story branch still lacked required shared-runtime main commits despite two acknowledged hard-prerequisite directives. Preserve current two-file delta for recovery; successor must patch-capture, restore only these task-owned paths, merge main 197238c additively, then reapply/reassess.
spawn run started: [implementer] developer (codex) (run=RUN-260826-6398e4)
Recovery audit: b3113e4 is a single-parent replay of landed shared-runtime product commit baf18fc onto the research branch. Non-board product paths match current main exactly except this Story own .research document and additive LOGBOOK; implementation may continue. Current main 197238c is still not an ancestor, so a dedicated fresh-base merge leaf is mandatory after implementation review and before story_final integration.
spawn run RUN-260826-6398e4 cancelled by operator; operator action required; reason: Run stopped after sustained no-progress interval. Preserve the Story worktree exactly; the orchestrator will continue from the existing diff and complete validation/review.
agent completed: [implementer] developer (codex) (exit=-1)
spawn run completed: codex (run=RUN-260826-6398e4, pid=74154, exit=-1)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"The implementation and full validation already exist in the managed worktree; gpt-5.6-sol/high is the user-requested strong pair for a bounded exact-diff inspection and producer handoff without reopening architecture."}
STORY-260825-2l6axn base refresh SKIPPED: the managed workspace holds uncommitted work, so there was no clean checkpoint branch to replay onto trunk 197238cff1ca; the branch is unchanged at fork point e70f953969d4
spawn selection rationale for gpt-5.6-sol/high: The implementation and full validation already exist in the managed worktree; gpt-5.6-sol/high is the user-requested strong pair for a bounded exact-diff inspection and producer handoff without reopening architecture.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-33-g197238c; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260826-e9f65e, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260826-e9f65e)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260826-e9f65e, pid=9078, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"This is a security-sensitive unattended execution boundary with extension replacement, raw-RPC bypass, process isolation, crash cleanup, and shared-runtime lifecycle claims; Claude Opus 5/high provides an independent strong review of the Codex-produced exact candidate."}
spawn selection rationale for claude-opus-5/high: This is a security-sensitive unattended execution boundary with extension replacement, raw-RPC bypass, process isolation, crash cleanup, and shared-runtime lifecycle claims; Claude Opus 5/high provides an independent strong review of the Codex-produced exact candidate.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-34-gfd80bd8; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260826-2449c2, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260826-2449c2)
Reviewer RUN-260826-2449c2, CR-TASK-260826-3i0lwe-1 rev 1: CHANGES REQUESTED. Candidate tree verified byte-identical to dcdfe67 before and after review. Full suite green on the candidate (main 81.7s, attachments 2.6s, infra 126.5s); vet/build clean on darwin arm64+amd64, windows/amd64, linux/amd64; gofmt clean. Twelve narrowing mutants on the authorization, argument-ownership, allowlist, mode, print/session, redaction, board-run-id and CLI-positional gates all died. Real pinned Pi 0.84.2 no-model proofs reproduced with live controls: extension discovery genuinely suppressed (control leg writes the marker), direct RPC bash writes a real sentinel past a blocking tool_call hook, production exposes JSON mode only. Concurrency proof genuine: distinct pgids, distinct hash-contained state, one verified shared runtime, crash releases only its own lease, final release reaps. No task-board dependency, no sudo/root, Windows fails closed. BLOCKING F1: the standalone stdin-isolation guard at pi_launch_posix.go:275-277 and pi_shared_client_darwin.go:629-631 has no witness that can fail. Deleting it from both production paths adds ZERO new failures across go test ./... (mutant failure set identical to baseline). The StdinEOF assertion in pi_standalone_shared_test.go:145 is vacuous because the test never sets RunPiOptions.Stdin, so os/exec wires /dev/null either way. No live bypass today (runPiStandaloneCLI omits Stdin) but the guard is load-bearing for exactly the deferred board adapter. Fix: pass a readable Stdin and keep asserting StdinEOF, covering both the exclusive and shared launch paths. N1-N4 non-blocking: entrypoint-name and pi-environment refusals are redundant with profileForbidden, deadline bound untested, README standalone snippet omits the required primary_session.pi_compatibility. Full detail in TASK-260826-3i0lwe_review-verdict.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260826-2449c2, pid=13826, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Revision 1 has one concrete reviewer-proven test-witness defect plus bounded CLI/docs gaps; gpt-5.6-sol/high is the user-requested strong implementation pair for discriminating exclusive/shared stdin mutations and publishing revision 2."}
STORY-260825-2l6axn base refresh SKIPPED: the managed workspace holds uncommitted work, so there was no clean checkpoint branch to replay onto trunk fd80bd8e0c1d; the branch is unchanged at fork point e70f953969d4
spawn selection rationale for gpt-5.6-sol/high: Revision 1 has one concrete reviewer-proven test-witness defect plus bounded CLI/docs gaps; gpt-5.6-sol/high is the user-requested strong implementation pair for discriminating exclusive/shared stdin mutations and publishing revision 2.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-34-gfd80bd8; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260826-462535, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260826-462535)
Revision 2 addressed reviewer F1: exclusive and shared standalone launch tests now pass readable non-empty RunPiOptions.Stdin witnesses and require EOF. One-at-a-time narrowing mutants in pi_launch_posix.go and pi_shared_client_darwin.go each exited 1 with StdinEOF:false; restored controls exited 0. Added CLI (0,30m] boundary/sanitized refusal tests and self-contained README compatibility prerequisite. Full go test ./... -count=1, vet, native/darwin/linux/windows builds, gofmt, and diff checks exited 0. Updated implementation outcome and attached TASK-260826-3i0lwe_revision-2-evidence.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260826-462535, pid=28184, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"Revision 2 changes security-sensitive unattended execution and stdin isolation; Opus high is justified for independent adversarial verification of the exact candidate and mutation evidence."}
spawn selection rationale for claude-opus-5/high: Revision 2 changes security-sensitive unattended execution and stdin isolation; Opus high is justified for independent adversarial verification of the exact candidate and mutation evidence.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-34-gfd80bd8; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260826-292d0a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260826-292d0a)
Reviewer verdict CR-TASK-260826-3i0lwe-2 rev2: ACCEPTED (RUN-260826-292d0a). Evidence: TASK-260826-3i0lwe_review-verdict-rev2.md. Rev1 blocking stdin defect is fixed and independently proven: removing the pi_launch_posix.go guard fails only the exclusive test, removing the pi_shared_client_darwin.go guard fails only the shared concurrent test, both with StdinEOF:false; restored code green. Four narrowing mutants killed (drop --no-extensions; caller-arg refusal narrowed to a reserved-spelling denylist; allowlist membership check removed; standalone run-ID override removed -> falls back to TASK_BOARD_RUN_ID). Verified independently against pinned Pi 0.84.2 that the exact production argv parses (control: unknown flag rejected), that the seven allowlisted names are exactly Pi built-ins, and that global extensions do load without --no-extensions so that test leg is not vacuous. agents-infra pi spawn --print-config exercised end to end: reasoning medium preserved, prompt redacted, yolo_mode=false -> pi_tool_authorization_required on stderr only. Full suites green: internal/infra 121.1s ok, main 75.7s ok, attachments ok; vet/gofmt clean; windows+linux cross-build ok. Non-blocking follow-ups F1 standalone run-state dirs never reclaimed while plan reports persistence=disabled, F2 global-extension test leg lacks an in-test positive control, F3 tool_authorization diagnostics are constants rather than derived from the composed argv.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260826-292d0a, pid=37413, exit=0)

## Precondition Resources
- [TASK-260826-3i0lwe_pi-unattended-tool-authorization.md](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_pi-unattended-tool-authorization.md) — Pinned Pi 0.84.2 authorization research and standalone unattended launcher decision
- [TASK-260826-3i0lwe_pre-merge-worker.patch](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_pre-merge-worker.patch) — Preserved two-file cancelled-worker delta; successor must restore only these paths, merge current main, then selectively reapply

## Outcome Resources
- [TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-e9a9bc.log](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-e9a9bc.log) — System spawn log captured by task-board
- [TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-6398e4.log](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-6398e4.log) — System spawn log captured by task-board
- [TASK-260826-3i0lwe_implementation-results.md](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_implementation-results.md) — Standalone Pi implementation and revision-2 mutation/full-gate evidence
- [TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-e9f65e.log](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-e9f65e.log) — System spawn log captured by task-board
- [TASK-260826-3i0lwe_change-request_rev1.patch](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_change-request_rev1.patch) — Change Request CR-TASK-260826-3i0lwe-1 revision 1 candidate patch (repository_delta=present, 35 changed paths)
- [TASK-260826-3i0lwe_spawn-log_-reviewer--reviewer--claude-_RUN-260826-2449c2.log](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_spawn-log_-reviewer--reviewer--claude-_RUN-260826-2449c2.log) — System spawn log captured by task-board
- [TASK-260826-3i0lwe_review-verdict.md](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_review-verdict.md) — Reviewer verdict for CR revision 1: changes requested — standalone stdin-isolation guard has no failing witness; full mutation/attack log and accepted evidence
- [TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-462535.log](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_spawn-log_-implementer--developer--codex-_RUN-260826-462535.log) — System spawn log captured by task-board
- [TASK-260826-3i0lwe_revision-2-evidence.md](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_revision-2-evidence.md) — Revision-2 stdin narrowing calibration, CLI bounds, and full Go gate evidence
- [TASK-260826-3i0lwe_change-request_rev2.patch](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_change-request_rev2.patch) — Change Request CR-TASK-260826-3i0lwe-2 revision 2 candidate patch (repository_delta=present, 35 changed paths)
- [TASK-260826-3i0lwe_spawn-log_-reviewer--reviewer--claude-_RUN-260826-292d0a.log](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_spawn-log_-reviewer--reviewer--claude-_RUN-260826-292d0a.log) — System spawn log captured by task-board
- [TASK-260826-3i0lwe_review-verdict-rev2.md](file://TASK-260826-3i0lwe/TASK-260826-3i0lwe_review-verdict-rev2.md) — Reviewer verdict for CR revision 2: accepted, with stdin-guard mutant proof, gate-narrowing table, pinned-Pi probes, and three non-blocking findings

## Created
2026-08-25T22:25:41Z

## Last Update
2026-08-26T11:02:44Z

## Assigned To
[reviewer] reviewer (claude)
