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
- [x] HF_ENDPOINT and MODEL_ENDPOINT policy enforced before runtime spawn with no value leakage
- [x] Production-entry clean and denial controls plus narrowing mutants pass
- [x] Operator documentation and docs regression gate describe model-origin environment policy
- [x] Bootstrap-installed global launcher and local wrapper behavior are verified
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
- [x] Direct runtime-launch production entry refuses denied model-origin environment before sharedRuntimeExecve

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/medium","text":"The verified follow-up is a bounded environment-policy extension with production negatives, docs, and installed-runtime evidence; Sol medium matches the project execution policy."}
spawn selection rationale for gpt-5.6-sol/medium: The verified follow-up is a bounded environment-policy extension with production negatives, docs, and installed-runtime evidence; Sol medium matches the project execution policy.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260817-3c9bf4, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260817-3c9bf4)
Developer handoff: exact HF_ENDPOINT and MODEL_ENDPOINT deny at RunPi pre-spawn environment gate; values redacted; tokens/cache remain separate. Unit, production-entry, narrowing-mutant, docs, installed global/local, full/race/vet/build/diff/setup/verify evidence is in BUG-260817-2bh9nk_results.md.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-3c9bf4, pid=49391, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5","text":"Independent Opus 5 review is warranted because model-origin environment policy is a supply-chain trust boundary enforced across source, bootstrap, and local wrapper surfaces."}
spawn selection rationale for claude-opus-5: Independent Opus 5 review is warranted because model-origin environment policy is a supply-chain trust boundary enforced across source, bootstrap, and local wrapper surfaces.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260817-afbb20, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260817-afbb20)
Reviewer verdict cycle 1: ACCEPTED. Independent attack evidence in BUG-260817-2bh9nk_review.md. HF-only and MODEL-only narrowing mutants of pi_catalog.go:295 each redden the opposite name in the unit gate, the production RunPi entry test, and (MODEL-only) both real installed launcher surfaces. README and SKILL.md docs mutants each redden their own regression test. Real bootstrap-owned ~/.local/bin/pi-infra probed with a positive clean control: control reaches runtime spawn (marker created), both denied names exit 1 with name-only diagnostics, no value leak, no runtime child. strings(1) on the installed binary reports 0 hits for HF_ENDPOINT and is a false-negative proxy (Go immediate-constant string compare) - only the behavioral probe establishes the property. No bypass found: no env field in PiRuntime config, both managed spawn sites downstream of the gate, no os.Environ re-read, malformed/duplicate/empty-name entries rejected first, --print-config takes no environment. gofmt/vet/build clean; go test ./... -count=1 green on all 3 packages. Non-blocking: officialPiAsset skips without the gitignored .temp asset, but the installed-launcher test covers the gate without it. Reviewer supplied no commit_ack; commit-owning mover still owns the final done transition.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260817-afbb20, pid=57417, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Next parked to-review element in this story now that its lane is free; draining the review backlog is a goal clause."}
spawn selection rationale for gpt-5.6-sol/high: Next parked to-review element in this story now that its lane is free; draining the review backlog is a goal clause.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-b4c8d8, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-b4c8d8)
agent completed: [reviewer] reviewer (codex) (exit=1)
spawn run RUN-260829-b4c8d8 failed without autonomous retry; operator action required; provider failure: provider_capability_unavailable: Codex app-server capability is unavailable; remediation: install or update Codex, then relaunch via `task-board codex` and retry
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Retry after the previous reviewer run failed on a transient Codex app-server capability error rather than on the work."}
spawn selection rationale for gpt-5.6-sol/high: Retry after the previous reviewer run failed on a transient Codex app-server capability error rather than on the work.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_compose_failed; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=composition_command_failed; bare child launch retained
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-1f24ee, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-1f24ee)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-1f24ee, pid=97193, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Closes a model-origin bypass reachable through an independently callable launcher entry; the fix must bind the check to the protected transition rather than to the path that normally precedes it."}
spawn selection rationale for gpt-5.6-sol/high: Closes a model-origin bypass reachable through an independently callable launcher entry; the fix must bind the check to the protected transition rather than to the path that normally precedes it.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_compose_failed; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=composition_command_failed; bare child launch retained
spawn queued: [implementer] developer (codex) (run=RUN-260829-1c4dae, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-1c4dae)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-1c4dae, pid=49882, exit=0)
spawn autonomous recovery: run RUN-260829-1c4dae queued successor RUN-260830-8012e8 (attempt 1/3, model=gpt-5.6-sol): Change Request construction for BUG-260817-2bh9nk failed: Change Request CR-BUG-260817-2bh9nk-1 revision 1 validation failed at command 1/2 (1-based) with exit code 1; log resource BUG-260817-2bh9nk_change-request_rev1-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (codex) (run=RUN-260830-8012e8)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-8012e8, pid=28938, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Second-round review of the model-origin bypass fix; the question is whether the check now binds to the protected transition rather than to the path that normally precedes it."}
spawn selection rationale for gpt-5.6-sol/high: Second-round review of the model-origin bypass fix; the question is whether the check now binds to the protected transition rather than to the path that normally precedes it.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-84-g5c9b4e4; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-7d25eb, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-7d25eb)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-7d25eb, pid=13641, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260817-2bh9nk_spawn-log_-implementer--developer--codex-_RUN-260817-3c9bf4.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_spawn-log_-implementer--developer--codex-_RUN-260817-3c9bf4.log) — System spawn log captured by task-board
- [BUG-260817-2bh9nk_results.md](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_results.md) — Implementation, negative evidence, validation, and installed launcher handoff
- [BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--claude-_RUN-260817-afbb20.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--claude-_RUN-260817-afbb20.log) — System spawn log captured by task-board
- [BUG-260817-2bh9nk_review.md](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_review.md) — Reviewer verdict cycle 1: accepted, with independent narrowing mutants, docs mutants, and a real installed-launcher probe with clean control
- [BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--codex-_RUN-260829-b4c8d8.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--codex-_RUN-260829-b4c8d8.log) — System spawn log captured by task-board
- [BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--codex-_RUN-260829-1f24ee.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--codex-_RUN-260829-1f24ee.log) — System spawn log captured by task-board
- [BUG-260817-2bh9nk_review-verdict-RUN-260829-1f24ee.md](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_review-verdict-RUN-260829-1f24ee.md) — Reviewer changes-requested verdict: shared runtime-launch bypass and rework proof contract
- [BUG-260817-2bh9nk_RUN-260829-1f24ee_production-binary-bypass.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_RUN-260829-1f24ee_production-binary-bypass.log) — Production binary child-exec bypass probe for both model-origin names
- [BUG-260817-2bh9nk_RUN-260829-1f24ee_original-gates.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_RUN-260829-1f24ee_original-gates.log) — Original RunPi denial and clean-control reviewer gates
- [BUG-260817-2bh9nk_RUN-260829-1f24ee_installed-docs-gates.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_RUN-260829-1f24ee_installed-docs-gates.log) — Installed normal launcher and operator documentation reviewer gates
- [BUG-260817-2bh9nk_spawn-log_-implementer--developer--codex-_RUN-260829-1c4dae.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_spawn-log_-implementer--developer--codex-_RUN-260829-1c4dae.log) — System spawn log captured by task-board
- [BUG-260817-2bh9nk_rework-results.md](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_rework-results.md) — Shared runtime-launch boundary fix, actual production-binary negatives, narrowing mutants, caller audit, and current validation evidence
- [BUG-260817-2bh9nk_change-request_rev1.patch](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_change-request_rev1.patch) — Change Request CR-BUG-260817-2bh9nk-1 revision 1 candidate patch (repository_delta=present, 3 changed paths)
- [BUG-260817-2bh9nk_change-request_rev1-validation.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_change-request_rev1-validation.log) — Change Request CR-BUG-260817-2bh9nk-1 revision 1 bounded validation log
- [BUG-260817-2bh9nk_spawn-log_-implementer--developer--codex-_RUN-260830-8012e8.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_spawn-log_-implementer--developer--codex-_RUN-260830-8012e8.log) — System spawn log captured by task-board
- [BUG-260817-2bh9nk_change-request_rev2.patch](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_change-request_rev2.patch) — Change Request CR-BUG-260817-2bh9nk-2 revision 2 candidate patch (repository_delta=present, 4 changed paths)
- [BUG-260817-2bh9nk_change-request_rev2-validation.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_change-request_rev2-validation.log) — Change Request CR-BUG-260817-2bh9nk-2 revision 2 bounded validation log
- [BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--codex-_RUN-260830-7d25eb.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_spawn-log_-reviewer--reviewer--codex-_RUN-260830-7d25eb.log) — System spawn log captured by task-board
- [BUG-260817-2bh9nk_review-verdict.md](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_review-verdict.md) — Reviewer acceptance verdict for Change Request revision 2 with production-entry attack evidence
- [BUG-260817-2bh9nk_RUN-260830-7d25eb_full-suite.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_RUN-260830-7d25eb_full-suite.log) — Independent uncached full Go module suite
- [BUG-260817-2bh9nk_RUN-260830-7d25eb_production-installed-docs.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_RUN-260830-7d25eb_production-installed-docs.log) — Independent production binary, installed launcher, and operator docs gates
- [BUG-260817-2bh9nk_RUN-260830-7d25eb_mutant-hf-only.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_RUN-260830-7d25eb_mutant-hf-only.log) — Expected-red HF-only narrowing mutant at built production binary entry
- [BUG-260817-2bh9nk_RUN-260830-7d25eb_mutant-model-only.log](file://BUG-260817-2bh9nk/BUG-260817-2bh9nk_RUN-260830-7d25eb_mutant-model-only.log) — Expected-red MODEL-only narrowing mutant at built production binary entry

## Created
2026-08-17T20:38:22Z

## Last Update
2026-08-30T00:45:05Z

## Assigned To
[reviewer] reviewer (codex)
