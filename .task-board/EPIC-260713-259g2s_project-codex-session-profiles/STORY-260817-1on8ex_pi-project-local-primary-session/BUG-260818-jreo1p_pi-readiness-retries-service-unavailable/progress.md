## Status
to-review

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(2))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Exact HTTP 503 responses are retried only while the owned runtime remains alive, until exact-model readiness or timeout
- [x] HTTP 502 and every other non-200 response remain fatal before Pi spawn
- [x] Production RunPi coverage and a widening mutant bind both the retry and rejection branches
- [x] Documentation, setup, installed verification, live Qwen text/tool smokes, and cleanup evidence agree with the implementation
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
Reproduced at production pi-infra with fully cached Qwen: llama.cpp HTTP 503 during model load caused immediate runtime_readiness_invalid. Fixed waitPiRuntimeReady to retry exact 503 only. Production RunPi test proves 503,503,200 reaches Pi only after request 3 and 502 fails after request 1; widening retry to all 5xx makes the named test fail. Focused/full tests, build, vet, gofmt, bootstrap, project sync/verify, and installed Qwen text/tool smokes exit 0. No staging or commit.
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5","text":"Opus 5 is required for independent review of production readiness lifecycle semantics, exact-status narrowing, and negative-mutant evidence."}
spawn selection rationale for claude-opus-5: Opus 5 is required for independent review of production readiness lifecycle semantics, exact-status narrowing, and negative-mutant evidence.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260818-f9bbb1, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260818-f9bbb1)
Cross-board finding from TASK-260817-300nun review (local-models, RUN-260818-8ca9ef): internal/infra/pi_launch_posix.go:385 calls deadline.Reset(timeout) on every exact-503 poll, so a runtime returning 503 persistently at the 100ms tick never trips startup_timeout_seconds; the configured 120s becomes a per-503 inactivity window rather than a launch bound and pi-infra hangs against a live-but-stuck server (child Signal(0) only bounds the dead-child case). TestPiLaunchReadinessRetriesOnlyServiceUnavailableAtProductionEntry covers 503-then-ready (count 3) and 502-fatal (count 1); persistent-503-must-still-time-out is uncovered, so the missing upper bound is green. The Qwen live smoke was unaffected (readiness reached in ~13.7s) and is accepted, but its artifact describes the fix as retrying until readiness/timeout, which the current code does not guarantee.
REVIEW VERDICT: changes_requested -> to-dev. Confirmed correct: the two named branches are genuinely bound at the production RunPi entry (widening 503->=500 makes bad_gateway subtest fail; deleting the 503 branch makes service_unavailable subtest fail; both reproduced independently). Full module suite green (root 76.5s, attachments 3.6s, infra 112.1s), vet/gofmt clean, single readiness surface, docs agree, and ~/.local/bin/agents-infra is byte-identical to a -trimpath rebuild of current source (sha256 6e74c363c2fcea21b56efe72f1f355738f83f32971b589ac7e2e5265ff4d837d), so the installed runtime carries the fix.

BLOCKING GAP: the two other clauses of the same AC sentence have zero binding coverage. (1) Timeout bound: injecting deadline.Reset(timeout) into the 503 branch makes the retry infinite, and go test ./internal/infra -count=1 still returns ok 70.703s. A permanently-503 llama.cpp would hang RunPi forever holding the profile lock and owned process group. (2) In-loop child liveness: deleting both case <-childWait.done and the child.Signal(0) check leaves the suite green at ok 71.280s; TestPiLaunchRefusesForeignReadyListenerForDeadChild only exercises the post-readiness check because it serves 200. No fixture in the package ever sustains 503. This change is what made that loop long-lived, so both bounds are newly load-bearing and shipped unbound.

Rework is additive, no production-path change needed: add production-entry cases for sustained-503 -> runtime_readiness_timeout and child-exit-during-503 -> runtime_exited_early, each asserting PIDs gone and lock released, and confirm both mutants go red. Minor: pi_launch_posix.go:389 formats read=%v with an always-nil readErr; and the live Qwen smoke claim in the results artifact has no attached log. Full detail in BUG-260818-jreo1p_reviewer-verdict.md. Working tree restored byte-for-byte; nothing staged or committed.
CORRECTION to the cross-board note from TASK-260817-300nun (RUN-260818-8ca9ef): the current source does NOT contain deadline.Reset(timeout). pi_launch_posix.go:385-387 is exactly `if resp.StatusCode == http.StatusServiceUnavailable { continue }`, file sha256 fe5c597bc8026411fcf141f74069ee3ff008b0a25dde56cf0269707802213b29, verified again after that note was written. That reviewer almost certainly read the file inside my transient mutant window - deadline.Reset(timeout) at that exact spot is the unbounded-retry mutant I injected and removed during this review (restore cmp-clean against a pre-mutation backup; the 04:13 mtime is that restore, content byte-identical to the producer output). Do not fix a line that does not exist. The conclusion in that note is nonetheless right and duplicates my Finding 1: persistent 503 must still trip startup_timeout_seconds, and that upper bound is currently unbound by tests - as written the code does terminate correctly on timeout, it is only the evidence that is missing. Concurrent reviewers mutating a shared dirty checkout is itself a hazard worth avoiding for this story.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260818-f9bbb1, pid=1374, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/medium","text":"Sol medium is sufficient for the narrow reviewer-directed test-only rework binding sustained-503 timeout, child-exit liveness, process cleanup, and lock release."}
spawn selection rationale for gpt-5.6-sol/medium: Sol medium is sufficient for the narrow reviewer-directed test-only rework binding sustained-503 timeout, child-exit liveness, process cleanup, and lock release.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260818-fe202a, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260818-fe202a)
Rework added production RunPi cases for persistent exact-503 timeout and child exit during 503, with PID reap and lock-release assertions. Isolated mutants are red and restored source is byte-identical; focused/full tests, build, vet, format, setup, global/local verify, print-config, board validation, and cleanup checks exit 0. BLOCKER: three installed Qwen text smokes each exited 1 with runtime_readiness_timeout at the configured 120s; tool smoke was not run because exact-model readiness never succeeded. No port-18011 listener or managed llama process remained, Ollama had no loaded model, and memory was 77% free. Restore the reviewed Qwen deployment within 120s (recommended) or explicitly approve a larger startup_timeout_seconds, then rerun installed text and read-tool smokes. Evidence: BUG-260818-jreo1p_rework-evidence.md; anomaly and isolation decision recorded in LOGBOOK.md.
CORRECTION: orchestrator directives 85b483 and b5a4e1 classify the three new Qwen timeouts as non-blocking because this re-smoke path does not expose the prior task-scoped cached weights. Preserve their exit-1 timeout and exit-0 cleanup evidence, cite the already accepted local-models TASK-260817-300nun live text/tool evidence, and do not supersede it. No further live attempts or profile changes. Rework is ready for review on real isolated red mutants, restored-source cmp, full tests/build/vet, setup/verify, and cleanup gates.
CONTROLLED LIVE GATE: final orchestrator directive 970c32 identified the accepted task HOME cache. Re-ran project-local installed pi-infra under env -i with HOME=/Users/alexis/src/local-models/.temp/TASK-260817-300nun/live-smoke/home, explicit PATH, TMPDIR=/tmp, and /bin/zsh -f. Text smoke exit 0 and JSON assertion proves exact QWEN_TEXT_OK. Tool smoke exit 0 and combined assertion proves exactly one bash call, exact printf QWEN_TOOL_VALUE=42 arguments/result, isError=false, and exact final QWEN_TOOL_OK:42. Post-smoke listener/process cleanup exit 0. Raw logs: BUG-260818-jreo1p_qwen-controlled-smokes.tar.gz. Earlier ordinary-HOME timeouts remain honest bounded-timeout cleanup evidence, not a blocker.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260818-fe202a, pid=9309, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5","text":"Opus 5 is needed for cycle-two adversarial review of the new termination-bound tests, isolated mutant integrity, controlled-cache live smokes, and concurrent dirty-checkout safety."}
spawn selection rationale for claude-opus-5: Opus 5 is needed for cycle-two adversarial review of the new termination-bound tests, isolated mutant integrity, controlled-cache live smokes, and concurrent dirty-checkout safety.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260818-a40b1d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260818-a40b1d)
CYCLE-2 REVIEW VERDICT: accepted. Both AC branches are bound at the production RunPi entry and both new termination bounds are real. Independently reproduced in an isolated rsync copy (.temp/BUG-260818-jreo1p/review2), never in the shared checkout: deadline.Reset(timeout) on exact 503 -> package hits the 60s go-test bound (RED); deleting both in-loop liveness checks -> refuses_after_owned_runtime_exits gets runtime_readiness_timeout instead of runtime_exited_early (RED); widening ==503 to >=500 -> bad_gateway RED; deleting the 503 branch -> service_unavailable_then_ready plus both new subtests RED. Source restored and re-hashed to fe5c597bc8026411fcf141f74069ee3ff008b0a25dde56cf0269707802213b29 after every mutant; pi_test.go f94500ddd04f8275726719a78622c24762e7cbeb6b7ba9891d988bd4b7959d5b; nothing staged or committed; no orphan process or listener.

Green gates re-run independently in the main checkout: go test ./... -count=1 exit 0 (root 122.568s, attachments 6.362s, infra 170.243s), go vet exit 0, gofmt -l no output, verify global exit 0, verify local /Users/alexis/src/local-models exit 0.

NEW EVIDENCE the producer did not have: the installed global runtime was driven through the real production entry. ~/.local/bin/agents-infra pi --version under env -i against Python readiness fixtures with startup_timeout_seconds=3 gives persistent 503 -> exit 1 runtime readiness timed out after 26 polls in 3.17s, and 502 -> exit 1 invalid readiness response: status=502 after 1 poll in 0.26s, runtime PID gone in both. Provenance: the installed binary is byte-identical (58f178e6139cba18b709edb83b642973a14cd30763b4430a185ddf4f0c0682eb) to a -trimpath rebuild of current source with the ldflags scripts/setup.sh:176 stamps; a plain -trimpath rebuild differs by 96 bytes and the symbol-table delta is exactly main.Version/Commit/BuildDate, so cycle ones plain-trimpath method is no longer valid. The project-local /Users/alexis/src/local-models/.local/bin/agents-infra is a shell launcher that go-builds from this source repo on every invocation, so the Qwen smokes necessarily ran current production source. Smoke tarball re-parsed independently: text final assistant text exactly QWEN_TEXT_OK; tool run has exactly one bash toolCall with command exactly printf QWEN_TOOL_VALUE=42, tool_execution_end isError=false with result exactly QWEN_TOOL_VALUE=42, final text exactly QWEN_TOOL_OK:42; stderr shows the reviewed profile on 127.0.0.1:18011.

NON-BLOCKING findings. (1) Narrowing survivor: time.NewTimer(timeout*10) keeps every readiness test green (timeout subtest runs 10.13s instead of 1.13s), so the suite binds terminates on some deadline but not on the configured startup_timeout_seconds; a swap to ShutdownTimeoutSeconds would also ship green. Production behavior is correct - my installed run timed out at 3.17s for a configured 3s. Cheap follow-up: assert elapsed in the timeout subtest. (2) Cosmetic: pi_launch_posix.go:389 formats read=%v with an always-nil readErr, so operators see status=502 read=<nil>; flagged in cycle one, still present. (3) Deleting only one of the two in-loop liveness checks survives because the other covers the same class - redundancy, not a hole; deleting both is RED. None of these admit what the gate must reject.

Honest scope limit: the controlled Qwen smoke does not itself traverse a 503 window - its llama.cpp bound port 18011 only after model loaded at 13.97s. 503 semantics are bound by the Go production-entry tests and by the installed-binary run.

Full detail: BUG-260818-jreo1p_reviewer-verdict-cycle2.md. Reviewer supplied no commit_ack; the commit-owning mover commits this scope and makes the final done transition with commit_ack=scope_committed.
ACCEPTANCE HANDOFF: the done transition was attempted without commit_ack and the board correctly refused it (version-control commit acknowledgement required, desired commit time backdated to the previous day after 20:00 MSK per owner policy). Per the reviewer contract this run supplies no commit_ack, so the task is parked in to-review carrying the cycle-2 accepted verdict. The commit-owning mover commits this scope and re-runs set_status(BUG-260818-jreo1p, status=done, commit_ack=scope_committed). Review sandbox .temp/BUG-260818-jreo1p/review2 and .temp/BUG-260818-jreo1p/installed-check removed after evidence capture; shared checkout unchanged.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260818-a40b1d, pid=21346, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260818-jreo1p_results.md](file://BUG-260818-jreo1p/BUG-260818-jreo1p_results.md) — Production reproduction, exact fix, negative mutant, tests, build, install, and live verification evidence
- [BUG-260818-jreo1p_spawn-log_-reviewer--reviewer--claude-_RUN-260818-f9bbb1.log](file://BUG-260818-jreo1p/BUG-260818-jreo1p_spawn-log_-reviewer--reviewer--claude-_RUN-260818-f9bbb1.log) — System spawn log captured by task-board
- [BUG-260818-jreo1p_reviewer-verdict.md](file://BUG-260818-jreo1p/BUG-260818-jreo1p_reviewer-verdict.md) — Reviewer verdict: changes_requested; two CONFIRMED unbound branches (503 retry timeout bound, in-loop child liveness) proven by suite-green mutants
- [BUG-260818-jreo1p_spawn-log_-implementer--developer--codex-_RUN-260818-fe202a.log](file://BUG-260818-jreo1p/BUG-260818-jreo1p_spawn-log_-implementer--developer--codex-_RUN-260818-fe202a.log) — System spawn log captured by task-board
- [BUG-260818-jreo1p_rework-evidence.md](file://BUG-260818-jreo1p/BUG-260818-jreo1p_rework-evidence.md) — Termination-bound production tests, isolated mutants, full validation, installed setup, controlled Qwen smokes, and cleanup evidence
- [BUG-260818-jreo1p_qwen-controlled-smokes.tar.gz](file://BUG-260818-jreo1p/BUG-260818-jreo1p_qwen-controlled-smokes.tar.gz) — Raw controlled-cache installed Qwen text/tool JSON and runtime stderr logs
- [BUG-260818-jreo1p_spawn-log_-reviewer--reviewer--claude-_RUN-260818-a40b1d.log](file://BUG-260818-jreo1p/BUG-260818-jreo1p_spawn-log_-reviewer--reviewer--claude-_RUN-260818-a40b1d.log) — System spawn log captured by task-board
- [BUG-260818-jreo1p_reviewer-verdict-cycle2.md](file://BUG-260818-jreo1p/BUG-260818-jreo1p_reviewer-verdict-cycle2.md) — Cycle-2 reviewer verdict: accepted, with mutant matrix, installed-runtime 503/502 proof, provenance, and live-smoke re-verification

## Created
2026-08-18T00:50:44Z

## Last Update
2026-08-18T02:06:15Z

## Assigned To
[reviewer] reviewer (claude)
