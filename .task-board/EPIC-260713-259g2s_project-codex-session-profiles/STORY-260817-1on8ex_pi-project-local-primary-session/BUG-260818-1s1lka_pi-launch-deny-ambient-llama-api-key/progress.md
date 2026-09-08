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
- [x] Exact LLAMA_API_KEY refusal occurs before managed state and runtime spawn without value leakage
- [x] Production and installed launcher controls plus narrowing evidence prove the gate
- [x] HF_TOKEN, cache variables, and unrelated names remain admitted controls
- [x] Operator docs and bootstrap/local verification pass
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
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/medium","text":"Ambient LLAMA_API_KEY has a documented authentication effect incompatible with these credential-free managed profiles; an exact-name gate and installed-surface tests are bounded work for Sol medium."}
spawn selection rationale for gpt-5.6-sol/medium: Ambient LLAMA_API_KEY has a documented authentication effect incompatible with these credential-free managed profiles; an exact-name gate and installed-surface tests are bounded work for Sol medium.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260817-afdf08, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260817-afdf08)
Implemented exact LLAMA_API_KEY refusal at production RunPi environment gate before state/runtime spawn. Added helper, production, installed global/local, admitted-control, non-leak, docs, and narrowing-mutant evidence. Full tests/vet/build/bootstrap/global+local verification exit 0. Evidence: BUG-260818-1s1lka_results.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-afdf08, pid=13689, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5","text":"Independent Opus 5 review is required because this exact-name security gate sits on the shared production launcher, mutates installed surfaces, and must verify pre-state ordering, non-leak diagnostics, admitted controls, and mutation evidence without trusting the producer summary."}
spawn selection rationale for claude-opus-5: Independent Opus 5 review is required because this exact-name security gate sits on the shared production launcher, mutates installed surfaces, and must verify pre-state ordering, non-leak diagnostics, admitted controls, and mutation evidence without trusting the producer summary.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260817-8fb08d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260817-8fb08d)
reviewer verdict (RUN-260817-8fb08d, claude-opus-5): ACCEPTED. Gate attacked with 5 mutants in disposable copies, not read: narrow (value-empty-only) reddens helper + production RunPi + BOTH installed launcher surfaces with the runtime actually spawning; prefix-broaden reddens LLAMA_API_KEY_SUFFIX clean controls; gate-reorder after CreatePiStateTree reddens the pre-state assertion for llama_api_key; value-leak mutant reddens non-leak assertions. Pristine baseline green in the same copies. Bypass search: single production caller main.go:423 -> RunPi; both spawn sites consume the validated slice; PiProfile carries no env field; --print-config never execs; duplicate/malformed/first-= splits all refused; Windows managed path unreachable. Independent validation in the main checkout: go test ./... -count=1, go vet, go build, gofmt -l, git diff --check, agents-infra verify global, agents-infra verify local all exit 0. Main checkout byte-verified unchanged (pi_catalog.go sha256 7decb2c7...). RESIDUAL GAP (accepted, low severity, no AC falsified): a strings.EqualFold widening mutant SURVIVES the entire internal/infra package -- TestPiExecutionEnvironmentAcceptsExactCleanEnvironment has lowercase controls for hf_endpoint/model_endpoint/ggml_backend_path but none for llama_api_key, while SKILL.md shipped by this change claims lookalikes stay admitted. One-line follow-up: add llama_api_key=case-sensitive-lookalike to that clean list. UNKNOWN, not inferred: the b10470 binary is no longer on this host, so the LLAMA_API_KEY-backs---api-key premise is carried forward from the prior cycle, not re-established here; whether b10470 exposes another ambient-auth env name is likewise unknown. Evidence: BUG-260818-1s1lka_review-verdict.md. Reviewer archetype supplies no commit_ack; routed to-review for the commit-owning mover to commit this scope and make the final done transition with commit_ack=scope_committed.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260817-8fb08d, pid=24538, exit=0)
Post-review correction attached: b10470 is present/executable and reports build 10470 commit 34af94cd9; only the verdict premise-presence paragraph is superseded. Accepted gate verdict stands. Lowercase llama_api_key admitted-control remains non-blocking follow-up.
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Next parked to-review element in this story now that its lane is free; it is the sibling of the ambient-environment class just accepted, so the same bypass shape is worth checking."}
spawn selection rationale for gpt-5.6-sol/high: Next parked to-review element in this story now that its lane is free; it is the sibling of the ambient-environment class just accepted, so the same bypass shape is worth checking.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-6ebf9d, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-6ebf9d)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-6ebf9d, pid=84745, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Additive fixture coverage so a case-broadening mutant on a correct production gate can no longer survive every shipped suite."}
spawn selection rationale for gpt-5.6-sol/high: Additive fixture coverage so a case-broadening mutant on a correct production gate can no longer survive every shipped suite.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-b6b5f9, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-b6b5f9)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-b6b5f9, pid=84697, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Second-round review of additive fixture coverage; the question is whether the case-broadening mutant now fails while genuinely unrelated names are still admitted."}
spawn selection rationale for gpt-5.6-sol/high: Second-round review of additive fixture coverage; the question is whether the case-broadening mutant now fails while genuinely unrelated names are still admitted.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-f28839, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-f28839)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-f28839, pid=76317, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Re-spawned against the published Change Request revision; the previous reviewer run predated the revision and could not legally record the acceptance it reached."}
spawn selection rationale for gpt-5.6-sol/high: Re-spawned against the published Change Request revision; the previous reviewer run predated the revision and could not legally record the acceptance it reached.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-78a722, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-78a722)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-78a722, pid=32123, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260818-1s1lka_spawn-log_-implementer--developer--codex-_RUN-260817-afdf08.log](file://BUG-260818-1s1lka/BUG-260818-1s1lka_spawn-log_-implementer--developer--codex-_RUN-260817-afdf08.log) — System spawn log captured by task-board
- [BUG-260818-1s1lka_results.md](file://BUG-260818-1s1lka/BUG-260818-1s1lka_results.md) — Developer implementation, negative narrowing, and validation evidence
- [BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--claude-_RUN-260817-8fb08d.log](file://BUG-260818-1s1lka/BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--claude-_RUN-260817-8fb08d.log) — System spawn log captured by task-board
- [BUG-260818-1s1lka_review-verdict.md](file://BUG-260818-1s1lka/BUG-260818-1s1lka_review-verdict.md) — CR rev1 accepted verdict by RUN-260830-78a722 with independent production/installed narrowing and EqualFold attacks
- [BUG-260818-1s1lka_post-review-correction.md](file://BUG-260818-1s1lka/BUG-260818-1s1lka_post-review-correction.md) — Primary-session correction of the reviewer runtime-presence claim; acceptance otherwise unchanged
- [BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--codex-_RUN-260830-6ebf9d.log](file://BUG-260818-1s1lka/BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--codex-_RUN-260830-6ebf9d.log) — System spawn log captured by task-board
- [BUG-260818-1s1lka_review-verdict-RUN-260830-6ebf9d.md](file://BUG-260818-1s1lka/BUG-260818-1s1lka_review-verdict-RUN-260830-6ebf9d.md) — Reviewer changes-requested verdict: EqualFold case-broadening survives production and installed suites
- [BUG-260818-1s1lka_review-evidence-RUN-260830-6ebf9d.tar.gz](file://BUG-260818-1s1lka/BUG-260818-1s1lka_review-evidence-RUN-260830-6ebf9d.tar.gz) — Focused, mutant, full-suite, bootstrap, and verify logs for reviewer RUN-260830-6ebf9d
- [BUG-260818-1s1lka_spawn-log_-implementer--developer--codex-_RUN-260830-b6b5f9.log](file://BUG-260818-1s1lka/BUG-260818-1s1lka_spawn-log_-implementer--developer--codex-_RUN-260830-b6b5f9.log) — System spawn log captured by task-board
- [BUG-260818-1s1lka_rework-results-RUN-260830-b6b5f9.md](file://BUG-260818-1s1lka/BUG-260818-1s1lka_rework-results-RUN-260830-b6b5f9.md) — Developer rework summary with EqualFold mutant and restored-source validation
- [BUG-260818-1s1lka_rework-evidence-RUN-260830-b6b5f9.tar.gz](file://BUG-260818-1s1lka/BUG-260818-1s1lka_rework-evidence-RUN-260830-b6b5f9.tar.gz) — Focused, EqualFold mutant, full-suite, build, vet, formatting, and restoration logs
- [BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--codex-_RUN-260830-f28839.log](file://BUG-260818-1s1lka/BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--codex-_RUN-260830-f28839.log) — System spawn log captured by task-board
- [BUG-260818-1s1lka_change-request_rev1.patch](file://BUG-260818-1s1lka/BUG-260818-1s1lka_change-request_rev1.patch) — Change Request CR-BUG-260818-1s1lka-1 revision 1 candidate patch (repository_delta=present, 2 changed paths)
- [BUG-260818-1s1lka_change-request_rev1-validation.log](file://BUG-260818-1s1lka/BUG-260818-1s1lka_change-request_rev1-validation.log) — Change Request CR-BUG-260818-1s1lka-1 revision 1 bounded validation log
- [BUG-260818-1s1lka_review-verdict-RUN-260830-f28839.md](file://BUG-260818-1s1lka/BUG-260818-1s1lka_review-verdict-RUN-260830-f28839.md) — Reviewer workflow changes-requested verdict: candidate passes, but run predates and is not bound to CR rev 1; requires CR-bound re-review
- [BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--codex-_RUN-260830-78a722.log](file://BUG-260818-1s1lka/BUG-260818-1s1lka_spawn-log_-reviewer--reviewer--codex-_RUN-260830-78a722.log) — System spawn log captured by task-board
- [BUG-260818-1s1lka_review-evidence-RUN-260830-78a722.tar.gz](file://BUG-260818-1s1lka/BUG-260818-1s1lka_review-evidence-RUN-260830-78a722.tar.gz) — CR rev1 reviewer evidence: exact patch, pristine production/installed validation, EqualFold and narrow-empty mutant reds
- [BUG-260818-1s1lka_review-verdict-RUN-260830-78a722.md](file://BUG-260818-1s1lka/BUG-260818-1s1lka_review-verdict-RUN-260830-78a722.md) — CR rev1 accepted verdict authored by bound reviewer RUN-260830-78a722

## Created
2026-08-17T22:20:40Z

## Last Update
2026-08-30T01:55:29Z

## Assigned To
[reviewer] reviewer (codex)
