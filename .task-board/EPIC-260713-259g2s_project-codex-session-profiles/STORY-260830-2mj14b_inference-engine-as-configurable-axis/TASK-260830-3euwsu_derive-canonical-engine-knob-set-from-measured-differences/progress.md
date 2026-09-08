## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- TASK-260828-15ftgj
- TASK-260830-ter72z

## Blocks
- TASK-260830-2cgim0
- TASK-260830-18n40a

## Checklist
- [x] Knob set derived from measured differences, each cited to the task that found it, not invented
- [x] Every knob states the refusal behaviour when an engine cannot express it — silent drop is forbidden
- [x] Covers argv spelling, stream field naming, health and readiness semantics, weight artifact shape, memory accounting, and capability differences such as speculative decoding
- [x] Canonical knob set derived from measured engine differences, each knob cited to the evidence that found it
- [x] Every knob states what happens when an engine cannot express it, distinguishing refusal, silent default and not-reported
- [x] The adapter contract covers argv spelling and parsing precedence, stream field names, health and readiness semantics, weight formats, memory accounting and speculative decoding
- [x] The process-reported versus argv-derived asymmetry is treated as a first-class contract concern, not an implementation detail
- [x] No admission clause of the comparative gate is weakened to make a knob fit
- [x] Specification only: no engine ported and no adapter implemented
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
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the contract must account for a runtime that cannot express two of three pinned terms."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the contract must account for a runtime that cannot express two of three pinned terms.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:6a3e45c201ed0de4aa6693249c4ae30cdc3d96faaae29e56d9e3b85144de8711 rationale="Desk analysis deriving an adapter contract from a landed comparison study."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-a94917, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-a94917)
Specification delivered: .research/260831_engine-adapter-contract-and-canonical-knob-set.spec.md, attached as outcome resource TASK-260830-3euwsu_engine-adapter-contract.spec.md, committed 9f82044. Ten knobs derived from the local-Qwen runtime comparison study (.research/260831_local-qwen-runtime-comparison-study.md) and the tasks it cites (TASK-260830-2hc5r2 progress, .research/260828_llamacpp-in-the-benchmark-gate.md, .research/260828_llamacpp-under-the-managed-harness.md, .research/260829_llamacpp-against-the-python-baseline.md): KV bound, prefill chunk, reasoning effort (the contextPolicy triad), speculative decoding, stream field naming (delta.reasoning vs delta.reasoning_content -- the landmine knob with no refusal path), cache semantics/telemetry, memory/weight-residency accounting, health/readiness semantics, weight artifact shape/equivalence declaration, and argv parsing precedence as a cross-cutting rule. Each knob states its reported/notReported/unread behaviour and cites evidence; Knobs 2 and 3 (prefill chunk, reasoning effort) explicitly forbid an argv fallback on llama-server because that reopens the studys central finding. No admission clause of the comparative gate is weakened. Spec-only: no engine ported, no adapter implemented, no build/test/lint applies to this repo for a markdown-only change (no markdown lint tooling present in relux-agents-infra).
Checklist items 10-14 (code written / tests / negative tests / lint / build) are the generic developer Definition-of-Done template items; this task literal AC and item 9 explicitly scope it to specification-only, zero code. Read against that scope they are vacuously satisfied: zero code was required by the task description/AC and zero was written, so there is no unmet code/test obligation; no gating/refusing/validating/authorizing/attesting behavior was implemented in code by this task, so no negative-test obligation exists either. For lint/build, ran the one relevant validation command for a text-only change: `git diff --check HEAD~2 HEAD` (whitespace/conflict-marker check over both new commits) -- exit 0, no issues. No markdown lint tool exists in relux-agents-infra and no build step consumes .research/*.md, verified via repo root listing (AGENTS.md, setup.sh/setup.ps1, scripts/, no Makefile targeting docs).
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-a94917, pid=59000, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the contract governs every future engine addition."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the contract governs every future engine addition.
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd3e613489c956e6043b696fb6cc3d206d0110fb87f94761c2290445c16d6d1f rationale="Contract review where citation integrity decides whether the derivation is real."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-5be30d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-5be30d)
CHANGES REQUESTED. Knob 1 (KV bound) claims notReported may fall back to argv (--max-kv-size), citing an older MLX-Swift-arm-only design (260828_llamacpp-in-the-benchmark-gate.md:481). LOGBOOK.md:310 shows this exact fallback was found as a ROOT CAUSE bug and fixed; TASK-260830-2hc5r2 rev4-6 (cited elsewhere in the same knob table) confirms production now refuses this case as contextBoundNotHonoured with no argv fallback; the main study (S3.1 cond.4, S4.1 option 2 rejected, S6.3) states no per-term exception. This is the ONE named exception to the Prime Directive (S2) and Knob 10 exists to serve it, so the error is structural, not cosmetic. Secondary: Knob 6s literal --prompt-cache-bytes help-text quote is not grounded to any cited evidence in-repo. Full verdict: TASK-260830-3euwsu_review-verdict.md. Routing to analysis for rework.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-5be30d, pid=64380, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the fix must not leave a narrower version of the same carve-out."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the fix must not leave a narrower version of the same carve-out.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:6a3e45c201ed0de4aa6693249c4ae30cdc3d96faaae29e56d9e3b85144de8711 rationale="Removing a contract exception that standardises a previously fixed bug."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-d40b87, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-d40b87)
Rework pass (RUN-260831-d40b87) closes both required fixes from TASK-260830-3euwsu_review-verdict.md: removed Knob 1's argv-fallback exception (root-caused, already-fixed bug per LOGBOOK.md:310/TASK-260830-2hc5r2), reframed Knob 10, sourced Knob 6's help-text quote to the pinned mlx-lm fork (mlx_lm/server.py:1902, commit 45a472f2d0cda166b7ffe1a80fe50dd9621f4303). Committed 040be30 (spec) + 6d682b5 (logbook).

Checklist items 17-21 rationale (spec-only deliverable, same scope discipline as the first developer pass's notes on items 10-14):
- Item 17 (Implementation matches AC): the corrected spec still derives the knob set from measured differences, cites each to evidence, and states refusal behaviour per knob -- matches AC verbatim.
- Item 18 (Solution fits project architecture): a specification document has no runtime architecture to fit; no violation possible.
- Item 19 (Tests green): no code changed, so no test suite applies. Ran the one relevant validation for a text-only change: `git diff --check 040be30~1 040be30` -- exit 0, no whitespace/conflict-marker issues.
- Item 20 (gate/refusal/validation/authorization/attestation behavior attacked): no such behavior was implemented in code by this task (spec-only, item 9), so there is nothing to attack; vacuously satisfied.
- Item 21 (verdict evidence added, status routed by verdict branches): TASK-260830-3euwsu_review-verdict.md is attached as outcome evidence and both of its required-rework bullets were applied verbatim; status is being routed to to-review per the changes-requested-then-reworked branch.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-d40b87, pid=65194, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the fix must not leave a narrower carve-out."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the fix must not leave a narrower carve-out.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:6a3e45c201ed0de4aa6693249c4ae30cdc3d96faaae29e56d9e3b85144de8711 rationale="Closing an argv carve-out that the shipped gate refuses, with a named smoke test as the evidence."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-84e36f, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-84e36f)
Verified on resume: RUN-260831-d40b87 already applied and committed the exact fix required by this spawn brief (commits 040be30 spec + 6d682b5 logbook) — Knob 1 notReported refuses unconditionally with no argv fallback, S2 names no standing exception, Knob 2/3 consistently forbidden, Knob 10 reframed as forward-looking-only, S5 lists Knobs 1/2/3 together. Re-read the full spec end-to-end (S0-S6) to confirm internal consistency: no remaining reference to an active argv fallback anywhere. Working tree clean, git diff --check HEAD~3 HEAD exit 0. No further code change required; handing off to review.
agent completed: [implementer] developer (claude) (exit=0)
spawn completion blocked: no new or updated task-scoped outcome artifact was attached. Add or update an outcome resource named like TASK-260830-3euwsu_results.md and then set status back to to-review.
spawn run completed: claude (run=RUN-260831-84e36f, pid=69107, exit=0)
No Change Request revision was published for TASK-260830-3euwsu (handoff_unsatisfied): the board is not at to-review
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; a narrower carve-out already survived one pass."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; a narrower carve-out already survived one pass.
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd3e613489c956e6043b696fb6cc3d206d0110fb87f94761c2290445c16d6d1f rationale="Verifying an argv carve-out is unreachable rather than narrower, in a contract that governs every future engine."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-7b6acd, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-7b6acd)
Review rev2: argv carve-out genuinely removed (grepped whole doc, no back door in §4/§6). 4/5 spot-checked citations hold; LOGBOOK.md:310 is stale (blank line at HEAD — content shifted to line 316 by this task's own later commit 6d682b5, since LOGBOOK.md prepends new entries). Appears 3x in spec. Small mechanical fix required, not a re-litigation of the argv question. See TASK-260830-3euwsu_review-verdict-rev2.md.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-7b6acd, pid=69774, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the citation class breaks silently and must be swept, not patched at three sites."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the citation class breaks silently and must be swept, not patched at three sites.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:6a3e45c201ed0de4aa6693249c4ae30cdc3d96faaae29e56d9e3b85144de8711 rationale="Two mechanical fixes: drifting line-number citations and a stale board resource."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-cd0b95, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-cd0b95)
Two mechanical fixes per spawn brief: (1) three LOGBOOK.md:310 line-number citations in the spec replaced with stable entry-ID citation (entry 0456, "Missing Live KV Evidence No Longer Falls Back To Argv") since the logbook prepends and the line drifted during this session; checked remaining spec citations, all other targets are line-stable revision-dated docs so no further fix needed. (2) Board resource TASK-260830-3euwsu_engine-adapter-contract.spec.md was stale (pre-040be30, argv exception still present) — refreshed via update_resource to match repo HEAD. Logbook entry recorded at 1520. Commits: f2034ca (citation fix), e538ae0 (logbook entry).
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-cd0b95, pid=70643, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the sweep claim needs checking, not the settled contract."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the sweep claim needs checking, not the settled contract.
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:cd3e613489c956e6043b696fb6cc3d206d0110fb87f94761c2290445c16d6d1f rationale="Proportionate confirmation of two mechanical fixes on settled material."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-3dd83c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-3dd83c)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-3dd83c, pid=74302, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-a94917.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-a94917.log) — System spawn log captured by task-board
- [TASK-260830-3euwsu_engine-adapter-contract.spec.md](file://TASK-260830-3euwsu/TASK-260830-3euwsu_engine-adapter-contract.spec.md)
- [TASK-260830-3euwsu_change-request_rev1.patch](file://TASK-260830-3euwsu/TASK-260830-3euwsu_change-request_rev1.patch) — Change Request CR-TASK-260830-3euwsu-1 revision 1 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260830-3euwsu_change-request_rev1-validation.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_change-request_rev1-validation.log) — Change Request CR-TASK-260830-3euwsu-1 revision 1 bounded validation log
- [TASK-260830-3euwsu_spawn-log_-reviewer--reviewer--claude-_RUN-260831-5be30d.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_spawn-log_-reviewer--reviewer--claude-_RUN-260831-5be30d.log) — System spawn log captured by task-board
- [TASK-260830-3euwsu_review-verdict.md](file://TASK-260830-3euwsu/TASK-260830-3euwsu_review-verdict.md) — Revision 3 confirmation verdict: both mechanical fixes verified, accepted
- [TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-d40b87.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-d40b87.log) — System spawn log captured by task-board
- [TASK-260830-3euwsu_rework-results.md](file://TASK-260830-3euwsu/TASK-260830-3euwsu_rework-results.md) — Rework summary: removed Knob 1 argv-fallback exception, sourced Knob 6 quote
- [TASK-260830-3euwsu_change-request_rev2.patch](file://TASK-260830-3euwsu/TASK-260830-3euwsu_change-request_rev2.patch) — Change Request CR-TASK-260830-3euwsu-2 revision 2 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260830-3euwsu_change-request_rev2-validation.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_change-request_rev2-validation.log) — Change Request CR-TASK-260830-3euwsu-2 revision 2 bounded validation log
- [TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-84e36f.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-84e36f.log) — System spawn log captured by task-board
- [TASK-260830-3euwsu_spawn-log_-reviewer--reviewer--claude-_RUN-260831-7b6acd.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_spawn-log_-reviewer--reviewer--claude-_RUN-260831-7b6acd.log) — System spawn log captured by task-board
- [TASK-260830-3euwsu_review-verdict-rev2.md](file://TASK-260830-3euwsu/TASK-260830-3euwsu_review-verdict-rev2.md) — Review revision 2 verdict: changes requested (minor) — stale LOGBOOK.md:310 citation
- [TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-cd0b95.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_spawn-log_-implementer--developer--claude-_RUN-260831-cd0b95.log) — System spawn log captured by task-board
- [TASK-260830-3euwsu_change-request_rev3.patch](file://TASK-260830-3euwsu/TASK-260830-3euwsu_change-request_rev3.patch) — Change Request CR-TASK-260830-3euwsu-3 revision 3 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260830-3euwsu_change-request_rev3-validation.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_change-request_rev3-validation.log) — Change Request CR-TASK-260830-3euwsu-3 revision 3 bounded validation log
- [TASK-260830-3euwsu_spawn-log_-reviewer--reviewer--claude-_RUN-260831-3dd83c.log](file://TASK-260830-3euwsu/TASK-260830-3euwsu_spawn-log_-reviewer--reviewer--claude-_RUN-260831-3dd83c.log) — System spawn log captured by task-board
- [TASK-260830-3euwsu_review-verdict-rev3.md](file://TASK-260830-3euwsu/TASK-260830-3euwsu_review-verdict-rev3.md) — Revision 3 confirmation verdict: both mechanical fixes verified, accepted

## Created
2026-08-29T22:23:47Z

## Last Update
2026-08-31T11:07:34Z

## Assigned To
[reviewer] reviewer (claude)
