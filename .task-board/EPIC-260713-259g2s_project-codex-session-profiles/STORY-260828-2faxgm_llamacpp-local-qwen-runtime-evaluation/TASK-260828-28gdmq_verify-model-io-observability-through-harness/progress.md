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
- [x] Enumerate what the harness proxies today: child stdout and stderr, emitted JSON events, supervision markers, exit and signal handling
- [x] State explicitly whether prompt and completion bodies travel over stdout/stderr or over the HTTP socket, per runtime, with evidence rather than assumption
- [x] Demonstrate on a live request whether the full prompt, the full completion, tool calls and reasoning content are recoverable from harness-captured output alone
- [x] Show that captured records can be correlated back to the originating request; if no request identity is carried end to end, record that as a named gap
- [x] Cover the failure paths too: a refused request, a condemned worker and a crashed child must each leave attributable evidence
- [x] Record retention, location and size behaviour of whatever is captured, so long unattended operation does not silently drop or unbounded-grow it
- [x] Any capability the harness is believed to have but does not is written down as a blocker, not quietly implemented in passing
- [x] Trace one real Pi session turn end to end and show which engine requests it produced, including compaction and any retry
- [x] State whether Pi-side turn identity is carried into the engine request at all; if nothing links them, that is a named gap, not a workaround to invent here
- [x] Cover the shared-broker case: with several Pi sessions leasing one runtime, show whether captured engine traffic can still be attributed to the session that caused it
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
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Investigation whose main risk is confirming a comfortable assumption; needs the judgement to distinguish what the harness captures from what it is believed to capture, and to leave gaps recorded rather than quietly closed."}
spawn selection rationale for claude-opus-5/high: Investigation whose main risk is confirming a comfortable assumption; needs the judgement to distinguish what the harness captures from what it is believed to capture, and to leave gaps recorded rather than quietly closed.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260828-6a4d27, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260828-6a4d27)
RESULT: the belief under test is false. The harness proxies child stdout/stderr faithfully, but on every managed runtime prompts and completions travel over the HTTP socket, which the harness does not sit on. Live nonce trace across 6 requests (buffered + streaming + tools + 404 + malformed): 0 hits for prompt body, completion text, tool schema, tool call, caller X-Request-Id, and the engine response id in harness-captured output. Only nonces planted in a URL survive, via the access-log request line.

CORRELATION: none end to end. Three concurrent completions produced three byte-identical POST lines. Pi records the engine responseId; the engine never emits it — the join key exists on one side of the boundary only. Shared broker is worse: two Pi sessions on one leased runtime produced two indistinguishable lines in one runtime.log with no lease/session/run marker, and leases/ is emptied on release while broker.log stayed 0 bytes.

FAILURE PATHS: refused request leaves only an access line with the path and status. Condemned worker is the one path with genuinely attributable harness-authored evidence (marker, restart counter, budget exhaustion, exit 1). Crashed child leaves one restart line. Crashed child MID-BODY is worse than absent — the access line reads 200 for a request whose caller received zero bytes, because status is logged at header-write time.

RETENTION: model-harness run opens no sink of its own; Pi exclusive mode does not persist runtime output at all; shared-broker runtime.log is O_APPEND with no rotation, no size cap, and cleanup deliberately preserves it. Behaviour under sustained load untested, stated as untested rather than inferred from the 261 MB fixture.

SCOPE DISCIPLINE: nothing was implemented. 8 named blockers B1-B8 in the report, each with what it would take to close and the decision needed.

TESTS: 4 characterisation/negative tests in internal/modelharness/run_observability_test.go. The supervision gate is bounded from both directions — the near-miss input shares every byte of the marker except its trailing open paren, so head- and tail-truncation mutants both fail it. All 5 mutants killed (gate widened 2 ways, carry dropped, forwarding removed, capture sink added). No delete-only mutant relied on.

VALIDATION (real exit codes): gofmt -l . clean; go vet ./... 0; go build ./... 0; go test ./... -count=1 0 (root 80.3s, infra 133.0s, modelharness 1.45s, attachments 1.38s). Both worktree_isolation validation commands from task-board.config.json run as declared.

UNTESTED, NOT INFERRED: llama-server (B8, not installed on this host); retention under sustained load (B6). Compaction was attempted in the traced session at compact_at_tokens=4870 against a 4861-token context and did NOT fire over four turns on the one-shot --print path; it is reported from production transcript evidence instead, and that non-firing is not claimed as a defect.

HOST CONSTRAINT HONOURED: the 28 GB Qwen model was never loaded. It was only inspected read-only (fds + its runtime log) as a passive natural experiment; not one request was sent to it.

COMMIT: changes left uncommitted. task-board.config.json version_control.confirm is true, so commit needs owner acknowledgement; desired timing is backdated to the previous day after 20:00 MSK. Uncommitted: LOGBOOK.md (M), .research/260828_model-io-observability-through-harness.md, tools/agents-infra/internal/modelharness/run_observability_test.go.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260828-6a4d27, pid=16410, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Independent-provider review of a negative finding that contradicts an assumed platform capability; the reviewer must be able to falsify the claim rather than agree with a familiar codebase."}
spawn selection rationale for gpt-5.6-sol/xhigh: Independent-provider review of a negative finding that contradicts an assumed platform capability; the reviewer must be able to falsify the claim rather than agree with a familiar codebase.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260828-1362ba, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260828-1362ba)
REVIEW VERDICT rev1: changes requested; route to analysis. Core HTTP-vs-stdio conclusion confirmed independently; B4 and B7 reproduced. Corrections required: B8 Homebrew availability claim does not reproduce, Pi event count is 78 not 77, test comment contradicts B8 unknown status, headline must distinguish Pi-visible messages from uncaptured full HTTP wire bodies, and each blocker needs an explicit closure decision/input. Evidence: TASK-260828-28gdmq_review-verdict.md and TASK-260828-28gdmq_review-evidence.tar.gz.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260828-1362ba, pid=39557, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Reporting-accuracy rework on an investigation whose deliverable is the report: an overstated headline must be narrowed without erasing the evidence that contradicts it, and each blocker needs its owning decision named."}
spawn selection rationale for claude-opus-5/high: Reporting-accuracy rework on an investigation whose deliverable is the report: an overstated headline must be narrowed without erasing the evidence that contradicts it, and each blocker needs its owning decision named.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260828-840e14, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260828-840e14)
rev2 rework — all five reworks applied, plus one self-found evidence gap.

1. Headline narrowed. The verdict no longer claims nothing on the managed path contains a prompt or completion. It now states the two-part result: model-harness-captured engine streams contain neither the assembled HTTP request nor the response body, and Pi-side records cannot be joined to engine-side requests. The partial Pi-side evidence (this task own Pi artifact carries the user message and the assistant completion text) is preserved explicitly, with the 85-chars-vs-4966-tokens gap as the reason it is only partial.

2. B8 reason corrected. brew search llama returns llama.cpp and brew info --json=v2 llama.cpp reports it available with installed: []. The false Homebrew claim is replaced and marked as an earlier-revision error. Conclusion unchanged: llama-server absent from PATH, behaviour unknown. Decision now points at TASK-260828-3g87i4.

3. Event count fixed to 78, verified by re-parsing pi-turn-stdout.json (78 records; census 1+1+1+4+68+1+1+1). Test comment narrowed to the two audited runtimes and restated as the channel-independent fact the test actually establishes.

4. Every blocker B1-B8 now carries one Owning decision plus a Recommendation and, where relevant, an explicit Needed from a human. B2 stays explicitly dependent on B1.

5. Board resource and .research/ copy re-synced and verified byte-identical (sha256 9fc7454a8b53e1e5781eb78933e503b0fe36ffe832870adeeff43239e44ef4e5).

Self-found gap, fixed: rev1 claimed the supervision-marker gate was bounded against truncation from either end. Re-running the mutants disproved it. marker[:14] and marker[:len(marker)-1] were killed, but marker[14:] — a genuine head truncation — SURVIVED with exit 0, because the single near miss diverged only at the marker trailing paren. TestSupervisionMarkerGateRejectsNearMissOutput is now table-driven with two near misses: one sharing the whole head, one sharing the whole tail including the paren. Each asserts at run time that it does not contain the full marker, so a near miss cannot degrade into a positive case. All six mutants now exit 1. Both the survived and killed runs are attached.

Validation, each a standalone process, real exit codes: gofmt -l . 0 no output; go vet ./... 0; go build ./... 0; go test ./internal/modelharness/ -count=1 0; narrow -run over the four observability tests 0 (4 tests, 2 subtests, all PASS); M1-M6 mutation runs exit 1 each.

NOT rerun at rev2 and stated as such rather than reported fresh: full-repo go test ./internal/... (infra takes ~2min). rev2 changes only .research/ prose, LOGBOOK.md and one _test.go. Production code untouched: git diff on internal/modelharness/run.go is empty after every mutant was reverted.

New artifact: TASK-260828-28gdmq_rework-evidence.tar.gz.
rev2 addendum: both declared board validation commands (spawn.worktree_isolation.validation) rerun at this revision — go vet ./... exit 0, go test ./... -count=1 exit 0 (root 86.887s, infra 146.187s, modelharness 1.689s, attachments 1.685s). The report Commands run section was corrected to record that instead of the earlier not-rerun caveat, and the board resource re-synced: sha256 5e44bebfedc61ec6edb512ed790159f14ecdd507c7011324bdca0a8d4b399cc5, verified byte-identical to .research/260828_model-io-observability-through-harness.md. Changes left uncommitted per version_control.confirm=true.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260828-840e14, pid=64064, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Second-round review by the author of the corrections list, checking that a headline narrowed on its instruction did not swing into understatement."}
spawn selection rationale for gpt-5.6-sol/xhigh: Second-round review by the author of the corrections list, checking that a headline narrowed on its instruction did not swing into understatement.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260828-58134b, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260828-58134b)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260828-58134b, pid=85267, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260828-28gdmq_spawn-log_-implementer--developer--claude-_RUN-260828-6a4d27.log](file://TASK-260828-28gdmq/TASK-260828-28gdmq_spawn-log_-implementer--developer--claude-_RUN-260828-6a4d27.log) — System spawn log captured by task-board
- [TASK-260828-28gdmq_model-io-observability-report.md](file://TASK-260828-28gdmq/TASK-260828-28gdmq_model-io-observability-report.md) — Model I/O observability investigation report (rev2: narrowed headline, corrected B8 rationale and event count, explicit owning decision per blocker, both-ends gate bound)
- [TASK-260828-28gdmq_evidence.tar.gz](file://TASK-260828-28gdmq/TASK-260828-28gdmq_evidence.tar.gz)
- [TASK-260828-28gdmq_change-request_rev1.patch](file://TASK-260828-28gdmq/TASK-260828-28gdmq_change-request_rev1.patch) — Change Request CR-TASK-260828-28gdmq-1 revision 1 candidate patch (repository_delta=present, 3 changed paths)
- [TASK-260828-28gdmq_spawn-log_-reviewer--reviewer--codex-_RUN-260828-1362ba.log](file://TASK-260828-28gdmq/TASK-260828-28gdmq_spawn-log_-reviewer--reviewer--codex-_RUN-260828-1362ba.log) — System spawn log captured by task-board
- [TASK-260828-28gdmq_review-verdict.md](file://TASK-260828-28gdmq/TASK-260828-28gdmq_review-verdict.md) — Reviewer changes-requested verdict with independent live reproduction and required research corrections
- [TASK-260828-28gdmq_review-evidence.tar.gz](file://TASK-260828-28gdmq/TASK-260828-28gdmq_review-evidence.tar.gz) — Raw reviewer live HTTP/harness, B4, B7 and validation evidence (sha256 8aa7cf53c9cb932a3f93e1fdc7802d44558eb46ff9aa47487d47507d1d0481ef)
- [TASK-260828-28gdmq_spawn-log_-implementer--developer--claude-_RUN-260828-840e14.log](file://TASK-260828-28gdmq/TASK-260828-28gdmq_spawn-log_-implementer--developer--claude-_RUN-260828-840e14.log) — System spawn log captured by task-board
- [TASK-260828-28gdmq_rework-evidence.tar.gz](file://TASK-260828-28gdmq/TASK-260828-28gdmq_rework-evidence.tar.gz) — rev2 rework evidence: narrow+package modelharness test logs and six mutation runs, including the rev1 head-truncation mutant that survived and motivated the second near miss (sha256 7b53ba86aa5fe7572e3ffb27353a7963121370a328fbe06dd6f66940081630c6)
- [TASK-260828-28gdmq_change-request_rev2.patch](file://TASK-260828-28gdmq/TASK-260828-28gdmq_change-request_rev2.patch) — Change Request CR-TASK-260828-28gdmq-2 revision 2 candidate patch (repository_delta=present, 3 changed paths)
- [TASK-260828-28gdmq_spawn-log_-reviewer--reviewer--codex-_RUN-260828-58134b.log](file://TASK-260828-28gdmq/TASK-260828-28gdmq_spawn-log_-reviewer--reviewer--codex-_RUN-260828-58134b.log) — System spawn log captured by task-board
- [TASK-260828-28gdmq_review-verdict-rev2.md](file://TASK-260828-28gdmq/TASK-260828-28gdmq_review-verdict-rev2.md) — Round 2 reviewer acceptance verdict for Change Request revision 2

## Created
2026-08-28T10:13:59Z

## Last Update
2026-08-28T11:29:42Z

## Assigned To
[reviewer] reviewer (codex)
