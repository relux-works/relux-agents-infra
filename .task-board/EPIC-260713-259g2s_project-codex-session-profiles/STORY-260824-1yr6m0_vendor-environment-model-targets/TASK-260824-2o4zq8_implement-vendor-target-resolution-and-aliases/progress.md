## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260824-3rl3ws

## Blocks
- TASK-260824-2a4gk3
- TASK-260824-1jjze0

## Checklist
- [x] Parse exact target/entrypoint fields and compose targets atomically root-to-cwd (Contract Sections 2-3)
- [x] Validate admitted tuples and Qwen Pi profile model/reasoning/endpoint assertions (Sections 2.3 and 7)
- [x] Implement identity-locked target dispatch plus --entrypoint primary-session compose and provenance output (Sections 3-5)
- [x] Install, repair, and verify all three sibling-only vendor aliases on global/local paths (Section 4)
- [x] Drive every Section 7 refusal through production parse/compose/alias/setup/verify paths with narrowing controls
- [x] Prove direct legacy Codex, Claude, Pi, pi-infra precedence is unchanged and update README/skill (Section 6)
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
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Cross-provider parser, identity-locked dispatch, setup aliases, diagnostics, and compatibility tests form a broad high-risk Go change; Sol high is appropriate while matching the owner-selected execution tier."}
spawn selection rationale for gpt-5.6-sol/high: Cross-provider parser, identity-locked dispatch, setup aliases, diagnostics, and compatibility tests form a broad high-risk Go change; Sol high is appropriate while matching the owner-selected execution tier.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-18-g302a445; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260824-d1391b, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260824-d1391b)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260824-d1391b, pid=54266, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"Independent cross-provider review must inspect the committed canonical-target implementation plus its residual CR, challenge fail-closed identity checks, alias safety, compatibility, and validation evidence; Opus 5 high is proportionate to the security-sensitive multi-file surface."}
spawn selection rationale for claude-opus-5/high: Independent cross-provider review must inspect the committed canonical-target implementation plus its residual CR, challenge fail-closed identity checks, alias safety, compatibility, and validation evidence; Opus 5 high is proportionate to the security-sensitive multi-file surface.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-24-gaa38ce9; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260824-36cd5a, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260824-36cd5a)
Review verdict: CHANGES REQUESTED (CR-TASK-260824-2o4zq8-1 rev1). Evidence: TASK-260824-2o4zq8_review-verdict.md. Build/vet/gofmt/git-diff--check clean; go test ./... -count=1 green; 34 canonical/Qwen/target tests PASS with 0 SKIP (LOGBOOK claim reproduces). Five narrowing mutants driven through production entrypoints were all killed. BLOCKING Finding 1 (CONFIRMED): a literal -- disables the Codex and Claude identity lock. lockCodexTargetArguments/lockClaudeTargetArguments return raw on -- as if it were an operand boundary, but parseCodexWrapperArgs (codex_launch.go:563) and parseClaudeWrapperArgs (claude_launch.go:414) consume that -- as a wrapper delimiter and keep parsing the rest as provider flags. Reproduced through runTarget with a recording fake provider, err=nil: openai-infra -- exec -- --profile work launches codex with --profile work (Section 3.6 says a Codex profile selector must ALWAYS fail target_identity_conflict on an alias invocation) while --print-config still reports effective_profile_source=native; anthropic-infra -- -- --model other launches claude with --model claude-opus-5 --effort high --model other, last-wins, so other launches while the plan reports claude-opus-5; same for --effort low and for Codex --model-reasoning-effort low. Pi is unaffected (real message-operand boundary, refuses flag-like operands). No test passes -- through any lock, so the -- branch of all three lock functions is entirely uncovered: bypass-path-around-the-check shape. Finding 2 (low): runTarget prints the whole launch plan to stderr on every hosted launch, not only under --print-config, and the Pi branch never does. Rework: lock across the wrapper --, add production-path negative tests for every affected selector plus a narrowing mutant, and pin the Pi divergence.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260824-36cd5a, pid=49500, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Reviewer reproduced a production delimiter bypass that can launch hosted providers with identity-divergent selectors after literal --; Sol high is appropriate to repair argument-boundary semantics, add production-path negative coverage and mutation controls, and preserve legacy behavior without widening scope."}
spawn selection rationale for gpt-5.6-sol/high: Reviewer reproduced a production delimiter bypass that can launch hosted providers with identity-divergent selectors after literal --; Sol high is appropriate to repair argument-boundary semantics, add production-path negative coverage and mutation controls, and preserve legacy behavior without widening scope.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-24-gaa38ce9; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260824-456538, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260824-456538)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260824-456538, pid=86123, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"A reproduced identity-lock bypass was repaired across wrapper/provider delimiters with new production-path negatives and a narrowing mutant; independent Opus 5 high review must re-attack the exact bypass, inspect stdout/stderr behavior, and verify no regression in hosted or Pi operand semantics."}
spawn selection rationale for claude-opus-5/high: A reproduced identity-lock bypass was repaired across wrapper/provider delimiters with new production-path negatives and a narrowing mutant; independent Opus 5 high review must re-attack the exact bypass, inspect stdout/stderr behavior, and verify no regression in hosted or Pi operand semantics.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-24-gaa38ce9; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260824-274367, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260824-274367)
reviewer RUN-260824-274367 verdict on CR rev2: ACCEPTED. Rev1 blocking finding (post-delimiter Codex/Claude identity-lock bypass) fixed in ba0d95d and independently verified: my own narrowing mutant reddens all 8 production runTarget subtests; live probes of the real codex/claude CLIs confirm the second delimiter is genuinely the provider operand boundary (claude reports claude-opus-5 with post-dash --model NOTAREALMODEL; codex exec -- --profile X errors unexpected argument; codex --cd cannot swallow --). New desync attacks (permission-mode/model/effort/-c/--sandbox/--cd eating the delimiter) all fail closed. Alias-set narrowing mutant kills the setup/verify per-alias controls. Live fail-closed matrix for 7.3/7.7/7.9 incl. unreadable-source-is-not-absent-mapping. Gates re-run here: build, vet, gofmt, both package suites, Qwen 8 PASS 0 SKIP, diff --check clean. Non-blocking: hosted-vs-Pi delimiter divergence is not in README/SKILL. No commit_ack supplied; acceptance evidence is TASK-260824-2o4zq8_review-verdict-rev2.md for the commit-owning mover.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260824-274367, pid=16403, exit=0)

## Precondition Resources
- [TASK-260824-2o4zq8_vendor-target-contract.md](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_vendor-target-contract.md) — Revision 3 architecture input; implement Sections 2-7 with provider-specific profile provenance and Codex selector locks

## Outcome Resources
- [TASK-260824-2o4zq8_spawn-log_-implementer--developer--codex-_RUN-260824-d1391b.log](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_spawn-log_-implementer--developer--codex-_RUN-260824-d1391b.log) — System spawn log captured by task-board
- [TASK-260824-2o4zq8_results.md](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_results.md) — Implementation, refusal coverage, and validation evidence
- [TASK-260824-2o4zq8_change-request_rev1.patch](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_change-request_rev1.patch) — Change Request CR-TASK-260824-2o4zq8-1 revision 1 candidate patch (repository_delta=present, 1 changed paths)
- [TASK-260824-2o4zq8_spawn-log_-reviewer--reviewer--claude-_RUN-260824-36cd5a.log](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_spawn-log_-reviewer--reviewer--claude-_RUN-260824-36cd5a.log) — System spawn log captured by task-board
- [TASK-260824-2o4zq8_review-verdict.md](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_review-verdict.md) — Reviewer verdict for CR rev1: changes requested; confirmed post-'--' Codex/Claude identity-lock bypass at the production alias entrypoint, with mutant and probe evidence
- [TASK-260824-2o4zq8_spawn-log_-implementer--developer--codex-_RUN-260824-456538.log](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_spawn-log_-implementer--developer--codex-_RUN-260824-456538.log) — System spawn log captured by task-board
- [TASK-260824-2o4zq8_rework-results.md](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_rework-results.md) — Reviewer rework implementation, production negative tests, narrowing mutant, and full validation evidence
- [TASK-260824-2o4zq8_change-request_rev2.patch](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_change-request_rev2.patch) — Change Request CR-TASK-260824-2o4zq8-2 revision 2 candidate patch (repository_delta=present, 1 changed paths)
- [TASK-260824-2o4zq8_spawn-log_-reviewer--reviewer--claude-_RUN-260824-274367.log](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_spawn-log_-reviewer--reviewer--claude-_RUN-260824-274367.log) — System spawn log captured by task-board
- [TASK-260824-2o4zq8_review-verdict-rev2.md](file://TASK-260824-2o4zq8/TASK-260824-2o4zq8_review-verdict-rev2.md) — Reviewer verdict for CR rev2: accepted; post-delimiter identity-lock fix verified against real Codex/Claude CLIs, independent narrowing mutants, live fail-closed matrix, all gates green

## Created
2026-08-24T14:44:35Z

## Last Update
2026-08-24T17:37:09Z

## Assigned To
[reviewer] reviewer (claude)
