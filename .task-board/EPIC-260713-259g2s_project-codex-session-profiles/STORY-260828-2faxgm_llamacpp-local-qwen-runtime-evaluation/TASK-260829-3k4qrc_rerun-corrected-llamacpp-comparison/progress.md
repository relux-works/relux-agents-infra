## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260829-3cwcb6
- TASK-260830-2hc5r2
- TASK-260830-y0oq3h

## Blocks
- TASK-260828-15ftgj

## Checklist
- [x] Both runtimes run sequentially on a host holding no other model, stated in the report
- [x] MTP off for every scored comparison; any with-MTP number reported separately and outside the comparison
- [x] Non-comparable dimensions refused by the gate, and every refusal reported rather than omitted
- [x] Corrected decode and TTFT reproduce or overturn the provisional 10 percent decode deficit
- [x] Corrected memory reproduces or overturns the provisional 9 percent advantage at 75000 tokens
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
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"The measurement the migration decision rests on, run once on accepted bias-free instrumentation across six scenarios and two 28 GB runtimes."}
spawn selection rationale for gpt-5.6-sol/high: The measurement the migration decision rests on, run once on accepted bias-free instrumentation across six scenarios and two 28 GB runtimes.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-d6b68d, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-d6b68d)
agent completed: [implementer] developer (codex) (exit=124)
spawn run completed: codex (run=RUN-260829-d6b68d, pid=25201, exit=124)
spawn run RUN-260829-d6b68d failed; operator action required; failure: run exceeded --timeout 5h0m0s and was terminated by the launcher
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Resumes the decisive measurement after an orchestrator-authored host-protocol deadlock, with the wait rule scoped to run start rather than to the run's own pass sequencing."}
spawn selection rationale for gpt-5.6-sol/high: Resumes the decisive measurement after an orchestrator-authored host-protocol deadlock, with the wait rule scoped to run start rather than to the run's own pass sequencing.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-a88f5c, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-a88f5c)
Recovery finding: accepted composite-memory reader inherited the legacy 250 ms Mach sampling cadence while each sample now launches blocking vmmap -summary. Two full-size attempts produced no baseline record before timeout; the latest reached only 32768/73016 capacity prompt tokens. Minimal fix centralizes a 5 s cadence with synchronous boundary samples; focused test, full Release suite, Release build, strict swift-format, and git diff --check all exit 0 before rerun. RUN-260829-d6b68d termination also left model-harness pid 40394 and Python child 40395 holding about 31 GiB until external process-group signalling, reproducing TASK-260828-28gdmq blocker B7 in production.
Corrected full rerun outcome: one foreground benchmark-run executed Python mlx-lm then llama.cpp sequentially on a host checked once before load with no other large model; MTP was off for all scored measurements. Both runtimes completed all six pinned scenarios with exact prompt-token parity and tool-call success. The admission gate refused scoring (exit 4, no decision.json) because contextPolicy differs: Python kv=unbounded versus llama.cpp kv=76800; every non-comparable dimension is enumerated in TASK-260829-3k4qrc_results.md. Corrected long-8k decode overturns the provisional 10% deficit: llama.cpp 7.880 ms/token vs Python 6.598 ms/token (+19.42% output rate), while TTFT is 69.451 ms vs 111.402 ms (-37.66%). At 73016 prompt tokens, corrected conservative memory reproduces only the direction of the provisional advantage and narrows it: llama.cpp 42.000 GiB vs Python 44.509 GiB (-5.64%, not 9%). Validation exits: focused test 0; Swift release tests 0 (402 tests/32 suites); Swift release build 0; strict swift-format lint 0; git diff --check 0; archive integrity 0. Evidence archive SHA-256 d3ebebea3251063e16ba0426e8e9cb8e7b3911625275b98cb409780d25ac6835.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-a88f5c, pid=90511, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Reviews the decisive measurement whose gate refused to score it; the central question is whether the context-policy mismatch is fixable configuration or an irreducible runtime property."}
spawn selection rationale for gpt-5.6-sol/high: Reviews the decisive measurement whose gate refused to score it; the central question is whether the context-policy mismatch is fixable configuration or an irreducible runtime property.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-f78621, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-f78621)
Reviewer revision 1 verdict: changes requested. Current kv=unbounded vs kv=76800 refusal is correct but fixable in pinned Python mlx-lm by plumbing and implementing bounded qwen3_5 cache construction, then rerunning. The blocking vmmap loop sleeps five seconds after each read and observed effective intervals are about 7.76s/7.32s; no production transient-spike negative test proves peak capture. Final report also omits the baseline-only prompt-cache declaration. Raw decode is tok/s, so +19.42% llama.cpp throughput is directionally correct; TTFT and capacity-memory arithmetic reproduce. See TASK-260829-3k4qrc_review-verdict.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-f78621, pid=58138, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Clears the dangling unaccepted Change Request that deadlocks its own sibling by addressing the sampler and report findings that do not depend on the KV bound."}
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Semantic merge of attestation logic hardened across nine bypass-closing review cycles, plus a peak-memory measurement that must be able to fail."}
spawn selection rationale for gpt-5.6-sol/high: Semantic merge of attestation logic hardened across nine bypass-closing review cycles, plus a peak-memory measurement that must be able to fail.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-62370c, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-62370c)
agent completed: [implementer] developer (codex) (exit=124)
spawn run completed: codex (run=RUN-260830-62370c, pid=30460, exit=124)
spawn run RUN-260830-62370c failed; operator action required; failure: run exceeded --timeout 1h0m0s and was terminated by the launcher
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Continuation of a timed-out run whose hard merge already landed; ceiling pair chosen because the remaining work is a measurement that must be able to fail."}
spawn selection rationale for gpt-5.6-sol/high: Continuation of a timed-out run whose hard merge already landed; ceiling pair chosen because the remaining work is a measurement that must be able to fail.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-92-g3295c7d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-332081, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-332081)
agent completed: [implementer] developer (codex) (exit=124)
spawn run completed: codex (run=RUN-260830-332081, pid=93417, exit=124)
spawn run RUN-260830-332081 failed; operator action required; failure: run exceeded --timeout 1h30m0s and was terminated by the launcher
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f7130ab6b2ae6c4bf7bde632137d1eb646cbd3be7f4039acc56d0cf5f965b005 rationale="Swift benchmark instrumentation fix; mmap-aware memory accounting on an axis weighted equally with decode."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-99-gfe38182; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-01f7fa, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-01f7fa)
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-01f7fa, pid=77240, exit=0)
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:d826fb4554b9825b9cea38eb387c522977ecf69d6e6e80c0a3c63c4696a71d0b rationale="Gate before the measurement the published decision rests on; refusal-path breadth must be established, not assumed."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-bcb7b6, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-bcb7b6)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-bcb7b6, pid=81691, exit=0)
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=codex/gpt-5.6-sol/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:f7130ab6b2ae6c4bf7bde632137d1eb646cbd3be7f4039acc56d0cf5f965b005 rationale="Two blocking measurement defects on the axis weighted equally with decode; both reproduced against exact candidate sources."
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260830-6d8cd4, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260830-6d8cd4)
Revision 3 fixes both review blockers: independent Mach/mapped timestamps now refuse stale mmap coverage, and sealed per-scenario cache-reuse observations now drive comparability. Full Swift suite: 407 tests, exit 0. Release build, swift-format, bash -n, shellcheck, diff check: exit 0. Full production smoke run 1: exit 1 from an overly narrow test assertion; isolated correction exit 0. Full production smoke run 2: 118 checks, exit 0. Both smoke logs and the revision-3 outcome are attached. Per rework brief, the hour-scale pair was not run; revision-1 decode/TTFT/75k memory numbers remain historical and outside the decision.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260830-6d8cd4, pid=19468, exit=0)
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260830-065450, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260830-065450)
agent completed: [reviewer] reviewer (codex) (exit=1)
spawn limit degradation: Provider limit on attempt 1: re-selection against the frozen snapshot chose codex/gpt-5.6-sol; relaunching under the same run
agent completed: [reviewer] reviewer (codex) (exit=1)
spawn limit exhausted: the retry was refused before any subscription group was subtracted (reason provider_limit_retry_bound, attempts 2, evidence RUN-260830-065450); provider reported: ERROR: You've hit your usage limit. Visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at Sep 6th, 2026 12:21 AM.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260830-abcd6d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260830-abcd6d)
Revision 3 review: CHANGES REQUESTED (to-dev). F2 (cache reuse) is genuinely closed and load-bearing under a narrowing mutant. F1 mechanism is correct but calibrated so the scored resident-memory dimension has an EMPTY admissible set: one vmmap -summary costs 0.668-0.687s while the mapped-file coverage bound admits 0.125s, so every window from 0.05s to 7s (and a zero-length one) refuses with resident-mapped-file-sampling-gap. Every memory delta becomes an unmeasured blocker and accepted can never be true; the 75k memory DoD item is unreachable. The smoke absorbs this: all admitted-path checks in the rev3 green run are exit 3 where rev2 was exit 0, so no full pass reaches accepted=true. Do NOT spend the hour on this revision. Full verdict: TASK-260829-3k4qrc_review-verdict-rev3.md
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-abcd6d, pid=99823, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260830-39dc95, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260830-39dc95)
Revision 4 handed to review. Scope was F3 + F4 from the revision-3 verdict; the hour-scale pinned pair was again NOT run, per the brief.

F3: maximumMappedFileSampleGapSeconds is now derived from the cadence the mapped component is actually read at - samplingIntervalSeconds + 2 * observedMappedFileReadCostSeconds = 7.0s - instead of an unreachable 125ms claim. Reader cost measured on this host through the production Process/Pipe shape: 0.608-0.672s (~3MB target) and 0.783-0.850s (~0.9GiB target), 8 calls each. The Mach component keeps its independent 20Hz series and its own 125ms bound. The blind spot is stated in the contract and carried in every emitted peak (mappedFileObservationLimitSeconds + mappedFileObservabilityNote); validatedScoredBytes returns nil for a measured peak that omits or forges them.

F4: the control now requires exit 0, accepted=true, no blockers and scored memory deltas with numeric baseline and candidate on the process and at least one scenario. mmap-memory assertions dropped the refusal tolerance. The refusal branch has its own fixture: benchmark-memory-coverage-refusal-probe drives the production sampler with the bound narrowed to 125ms and requires refusal while the unnarrowed control scores. Every admitted-path check is back at exit 0; revision 3 had all of them at exit 3.

Third defect, found by the F4 tightening rather than by reading: stop() retired both sampling loops at once, so the 20Hz Mach loop exited while an in-flight vmmap read landed one reader-cost later, leaving a hole at the tail of every process-wide series. 20 of 20 sessions in the pre-fix smoke had a partial process peak on the larger runtime, tail gaps 0.358-0.494s. stop() now retires the slow loop first. Covered by benchmark-memory-stop-coverage-probe, which exits 1 against the restored mutant.

Validation: swift build -c release exit 0; swift test -c release exit 0 (410 tests / 32 suites, up from 407); swift format lint --strict exit 0; four production-entry probes exit 0; teardown mutant probe exit 1 as required; gate smoke run 3 = 119 PASS / 1 FAIL exit 1. The single FAIL was the new series-coverage assertion applied to warmupMemory, a single point reading rather than a sampled window, read by no decision; the assertion was scoped to series and the exact block replayed against run 3 artifacts (exit 0). No production source changed after run 3, and a fourth full run was not performed because the task caps the suite at two full runs.

HONESTY ON THE CHECKLIST: items 1-5 (both runtimes sequential on a clean host; MTP off for every scored comparison; non-comparable dimensions refused; corrected decode/TTFT; corrected memory at 75k) were checked by an earlier revision and are NOT satisfied by this one. No rerun was performed in revisions 2, 3 or 4. They remain pending the hour-scale pair, which is now worth running because a scored memory dimension is demonstrably reachable at the production entry.

STILL UNKNOWN, not closed here: whether llama.cpp streaming usage frames report prompt_tokens_details.cached_tokens. The baseline does. If the candidate does not, cacheReuseObservation returns unknown and every scenario becomes non-comparable - the same failure shape as F3 applied to every dimension at once. One short streamed request against llama-server answers it and should precede the pair.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-39dc95, pid=15824, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260830-d9668d, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260830-d9668d)
Revision 4 ACCEPTED (accept_cr, evidence TASK-260829-3k4qrc_review-verdict-rev4.md). F3 and F4 both closed. Reviewer executed rather than read: rev2 transient attack on candidate sources (missed transient is now disclosed, not stale-minted); vmmap cost measured independently 0.663-0.957s, formula 5+2x1.0=7.0 reproduces; broken-reader and narrowed-constant mutants fail all four probes; a 3600s override cannot widen the gate against a real 11.67s hole; full benchmark-gate-smoke.sh run by the reviewer is 120 PASS / 0 FAIL exit 0 with accepted=true and scored memory on process and short_prompt; swift test 410/32 exit 0. Carried-forward unknown CLOSED by live probe against pinned llama-server 0.3.0: streamed usage frame reports prompt_tokens_details.cached_tokens (0 cold, 54 on repeat) when stream_options.include_usage is sent, and the driver always sends it; without stream_options there is no usage frame at all. Instrumentation is trustworthy enough to run the hour-scale pair. SCOPE: this accepts the instrumentation revision only - the task AC (the actual pinned rerun) is still outstanding, so do not close as done on this revision alone. Watch item before the pair: if real vmmap cost on the 28GiB runtimes exceeds 2.0s the mapped gap exceeds the 7.0s bound and memory refuses (fails safe, but visibly).
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-d9668d, pid=59263, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260830-aae9ac, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260830-aae9ac)
Revision 4 measurement run in progress.

Instrument validated on the exact working-tree source before the pair was launched (rev3 evidence did not cover it — Sources changed at 19:10/19:25 today, after the 15:06 rev3 smoke):
- swift test -c release: exit 0, 410 tests / 32 suites
- scripts/benchmark-gate-smoke.sh: exit 0, 120 PASS / 0 FAIL, driven at the production subcommands
- xcrun swift-format lint --strict --recursive Sources Tests: exit 0
- bash -n + shellcheck -S warning on the smoke script: exit 0 / exit 0
- swift build -c release: exit 0

Host state before launch: no llama-server, mlx_lm, mlx-swift or model-harness process resident; ~40 GB available.

The pair now has a pinnable contextPolicy for the first time in this story: STORY-260830-2vrhg1 delivered --max-kv-size in the pinned mlx-lm fork 45a472f, so the baseline can state kv=76800 against llama.cpp kv=76800 instead of the kv=unbounded refusal the earlier runs hit.

Config: .temp/TASK-260829-3k4qrc/rev4/TASK-260829-3k4qrc-rev4.benchmark.toml
Runner: .temp/TASK-260829-3k4qrc/rev4/run-rev4.sh (single foreground benchmark-run owning both passes; terminates only its own process group)
Revision 5: THE CORRECTED PAIR WAS RUN. One foreground benchmark-run, six scenarios x two runtimes, sequential, MTP off on both, 3535 s wall (58m55s). All twelve scenario runs succeeded at exact prompt-token parity 1.0000, including the 73,016-token capacity probe. Driver exit 4 — INADMISSIBLE, no decision.json. Full report: TASK-260829-3k4qrc_results-rev4.md.

SEALED, NON-OVERLAPPING INTERVALS
  baseline  python-mlx-lm 1788113670.052570 .. 1788115795.126375 (2125.074 s)
  candidate llamacpp      1788115816.214009 .. 1788117186.393213 (1370.179 s)
  separation 21.088 s; overlap -21.088 s. ps sweeps before and after both printed (none); raw process lists captured both sides. B7 did not recur — runner signals only its own process group; no orphan named this run config.

REFUSAL 1 — contextPolicy, exit 4. THE TERM THAT BROKE IS NOT THE ONE THAT WAS FIXED.
  baseline  kv=76800;prefill-step=2048;reasoning=medium
  candidate kv=76800;prefill-step=not-reported;reasoning=not-reported
STORY-260830-2vrhg1 closed the KV term — both sides now pin kv=76800 from live /v1/models. The refusal moved to the pin two other terms. Probed the same build/GGUF under --ubatch-size 2048 --reasoning-effort medium: /v1/models meta = {vocab_type,n_vocab,n_ctx,n_ctx_train,n_embd,n_params,size,ftype}; /props = {build_info,model_path,total_slots,chat_template,modalities,default_generation_settings{n_ctx,params}}; /slots[] = {id,is_processing,n_ctx,speculative}. NO n_batch/n_ubatch anywhere, and params.reasoning_format read "none" under --reasoning-effort medium so it does not track the launch (the same /props trap already recorded for speculation). The gate derives all three terms from the live listing and never from argv; reading them off argv would reopen the defect that derivation closed, so nothing was weakened.

REFUSAL 2 — MEMORY REFUSED ON EVERY WINDOW OF BOTH RUNTIMES, INCLUDING context_75k.
Only one window scored in the whole pair (candidate short_prompt, 34,731,153,644 B), so NO scenario has a scored memory value on both sides and the memory axis produced no comparison. Cause measured from the run own raw series (gap - 5.0 s sleep): one vmmap -summary against these 26-45 GiB targets costs a median 2.221-2.569 s and up to 5.823 s, where the 7.0 s bound was calibrated on 0.608-0.850 s reads against <=0.9 GiB targets. 268/288 baseline and 179/200 candidate mapped gaps exceeded the bound. Mach series held its 125 ms bound inside fast windows (0.060-0.069 s) and exceeded it only under load (0.618 s and 0.273 s in baseline context_75k at host load 13.995; eight 0.13-0.29 s gaps on the candidate pass series). One failed read per pass, counted, not folded into absence. The reviewer rev4 watch item fired exactly as written. THE BOUND WAS DELIBERATELY NOT WIDENED after seeing my own run refuse.

REFUSAL 3 — two scenarios non-comparable on sealed cached-token telemetry (both runtimes reported it on all six; nothing unknown or malformed): multiturn_prefix_reuse baseline miss [0,0,0] vs candidate HIT [5736,7780,7809]; stability_soak baseline miss [0]x20 vs candidate HIT [18]x20. llama-server reuses KV across scenarios — its first multiturn turn already had 5,736/7,784 cached, warmed by long_prompt_8k — which is why its TTFT there is 0.726 s vs 105.206 s. The incumbent configured prompt cache did NOT fire once (cached_tokens 0 on all six scenarios / 26 turns, corroborated by 347.591 s over three turns ~= three full prefills). The shipped config declares this asymmetry backwards.

THE NUMBER SET (record readings; refused as a comparison; decode/prefill are tok/s higher-better, TTFT/wall are seconds lower-better; ratio = llamacpp/python)
  short_prompt    TTFT 2.5314 / 2.1429 s (0.8465) | prefill 16.1969 / 19.1328 (1.1813) | DECODE 6.7809 / 8.1450 (1.2012) | wall 10.4496 / 9.1055 (0.8714)
  long_prompt_8k  TTFT 107.2163 / 67.4273 s (0.6289) | prefill 72.6009 / 115.4428 (1.5901) | DECODE 6.6705 / 7.7235 (1.1579) | wall 123.0636 / 81.1127 (0.6591)
  tool_call       wall 13.1792 / 11.0522 s (0.8386); parity satisfied on both (finish_reason=tool_calls, correct tool name, JSON args, required key present)
  stability_soak  wall 291.3045 / 221.9062 s (0.7618); 20/20 iterations on both, no failed iteration
  context_75k     TTFT 1279.8919 / 950.8526 s (0.7429) | prefill 57.0486 / 76.7900 (1.3460) | decode 8.0269 / 8.7764 (1.0934, 15 tokens over a ~1.8 s tail — noise-dominated, do not lean on it) | wall 1281.7795 / 952.5634 (0.7432)
  multiturn timing omitted from the comparison: non-comparable on cache telemetry.

VS THE PROVISIONAL NUMBERS
  Withdrawn frame-derived "llama.cpp about 10% slower at decode": OVERTURNED. llama.cpp decodes faster on every scenario that measures decode (+20.1%, +15.8%, +9.3%).
  decode long_prompt_8k: rev1 1.1942x -> rev4 1.1579x (reproduced).
  TTFT long_prompt_8k: rev1 0.6234x -> rev4 0.6289x (reproduced).
  prefill long_prompt_8k: rev1 1.6041x -> rev4 1.5901x (reproduced).
  resident bound at 73,016 tokens: rev1 47,791,331,280 vs 45,097,521,165 B -> rev4 REFUSED ON BOTH. Neither reproduced nor overturned.
Nothing was retried, reordered or tuned. One run; these are its numbers.

CHECKLIST HONESTY: items 1-4 are satisfied by this run. Item 5 (corrected memory reproduces or overturns the 9% advantage at 75k) landed on the brief EXPLICITLY-REFUSED branch, not the reproduce/overturn branch — context_75k memory is refused on both runtimes with the cause measured. It was already checked by an earlier revision and I did not re-check it on new evidence.

OPEN, for review to rule on: (1) llama.cpp has no live effective-configuration report, so this pair is unscoreable unless llama.cpp gains one; the other two exits are permanent non-comparability or a weakened admission clause. (2) the 7.0 s mapped bound is calibrated ~2.5x below what a 28 GB target costs; re-deriving it must be decided before a rerun, not after seeing which way the numbers went. (3) examples/model-harness.benchmark.toml declares the cache asymmetry backwards. (4) llama.cpp resident bound climbed +6,201,119,136 B across the soak while the baseline fell 509,935,592 B; unexplained.

VALIDATION (standalone processes, real exits): swift build -c release 0 | swift test -c release 0 (410 tests / 32 suites) | scripts/benchmark-gate-smoke.sh 0 (120 PASS / 0 FAIL) | swift-format lint --strict 0 | bash -n 0 | shellcheck -S warning 0 | git diff --check 0 | benchmark-run 4 (expected-refusal path, inadmissible, no decision.json). No production source was changed by this task; repository delta is docs only (LOGBOOK.md, tools/mlx-swift-runtime-prototype/README.md, .research/260829_llamacpp-against-the-python-baseline.md) on top of the accepted revision-4 instrumentation. Evidence archive SHA-256 103e9a26cbc92d5c2b9547b74167f4518694b174e33d6ec6fd2e862a43d11f70
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-aae9ac, pid=83785, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260830-08b2c8, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260830-08b2c8)
Revision 5 review: CHANGES REQUESTED (docs-only, no rerun needed). Evidence: TASK-260829-3k4qrc_review-verdict-rev5.md, TASK-260829-3k4qrc_reviewer-rev5-probe-evidence.tar.gz.

All four verification items confirmed independently. (1) contextPolicy refusal is irreducible: I relaunched llama-server b10621 myself with --metrics added and probed /props /slots /metrics /v1/models /models /health; no n_batch/n_ubatch anywhere, reasoning_format reads none under --reasoning-effort medium, and chat_template_caps.supports_reasoning_effort is a template capability not the effective value. (2) Readings reproduce to the printed digit from the raw records; sealed intervals do not overlap, prompt tokens equal on all six scenarios, speculation off both sides, exit 4 and no decision.json. (3) Memory arithmetic reproduces exactly: mapped gaps median 7.569/7.221 s, 268/288 and 179/200 over the 7.0 s bound. (4) context_75k decode is 15 tokens over a ~1.8 s tail; drop it from the article, do not footnote it.

Two blocking reported-fact defects, both documentation-only:
F1: the claim persisted into .research, LOGBOOK and tools README that every memory window was refused except one is false. session.json emits four more windows - warmupMemory and soakMemory on both sides - as status=measured with scoredBytes, because BenchmarkPass.swift:100/107 build them by direct RuntimeMemoryPeak(summarizing:) and never reach BenchmarkFootprintSampler.coveredPeak:333-345. Their own stamps are the worst-covered series in the pair (mapped gaps 10.1-15.1 s, 19/19 over bound). The +6,201,119,136 B soak climb this CR writes into LOGBOOK is read off one of them. decide() reads neither, so no scored comparison consumed them - reporting defect, not scoring.
F2: report section 4.1 narrates the synthetic coverage marker as one failed read against a process that had already been torn down. memorySamplesReadFailed is 0 in both passes; readFailureCount=1 appears on every partial window and 0 on the one measured window - it tracks the refusal, not a read.

Recommendation on the pending judgement: option (b), proceed to the article. Option (a) as scoped would not deliver the number the decision needs - context_75k memory refuses on BOTH sides for mach-physical-footprint-sampling-gap, not the mapped bound, blown by a host load of 13.995 the 73k prefill itself generates. Re-deriving the mapped cost fixes the short scenarios and leaves the 75k comparison refused exactly as it is. The only version of (a) worth backing is replacing the vmmap fork with an in-process read, which is an instrumentation task, not a calibration round. Article constraints listed in the verdict.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-08b2c8, pid=98593, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260830-cd38a1, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260830-cd38a1)
CR revision 6 — documentation and reporting only, no measurement code changed, no rerun (rev5 review stated no rerun needed).

FINDING 1 FIXED. Verified against session-rev4 raw data before writing: four windows (warmupMemory + soakMemory on both runtimes) come back status=measured, issues=[], with scoredBytes 29120518072 / 29827094504 / 34248152988 / 44346176540. They never reach the coverage gate — BenchmarkPass.swift:100 and :107 construct them by direct RuntimeMemoryPeak(summarizing:), while the gate is BenchmarkFootprintSampler.coveredPeak (:333-345). Recomputed their coverage: both soakMemory windows are 20 stamps with 13.862/14.682/15.117 s and 10.141/10.916/13.129 s gaps, 19/19 outside BOTH the 7.0 s mapped bound and the 125 ms Mach bound — the worst-covered series in the pair. Both warmupMemory windows are a single point. Bound confirmed: grep of Sources/MLXSwiftRuntimeContract/ returns neither name, so no scored comparison consumed them.
Refusal accounting corrected in all four documents (new report section 4.0 + scoped claims in .research, LOGBOOK.md, tools README). The +6,201,119,136 B soak climb now carries its provenance everywhere it appears and is stated as NOT a leak and NOT a memory regression: it is a two-point first-to-last delta off that ungated series. Additional verified detail — the whole climb is the Mach anonymous component (9905647432 -> 16106766568 B, exactly +6201119136) while the mapped component is constant at 28239409972 B on all 20 samples.

FINDING 2 FIXED. memorySamplesReadFailed=0 and memorySamplesMalformed=0 on BOTH passes. The readFailureCount:1 is the synthetic coverage marker coveredPeak appends on refusal (BenchmarkFootprintSampler.swift:344): confirmed 1 on all 26 partial windows and 0 on the single measured one (candidate short_prompt). Section 4.1 sentence replaced with what the counter is; no physical conclusion drawn.

ALSO FIXED. context_75k decode withdrawn — completionTokens is 16 on both runtimes. Struck in the 3.2 table as withdrawn, removed from the 3.3 overturned row and the six-sentence summary, marked withdrawn in LOGBOOK.md and the research banner. That scenario is cited for capacity, TTFT and prefill only.

UNCHANGED as instructed: all measurements and raw series, the contextPolicy exit-4 refusal, decode/TTFT 8.145 vs 6.781 and 7.724 vs 6.671 tok/s, long_prompt_8k TTFT 0.629x, and the memory refusal as the honest answer for that axis.

DELTA: 3 markdown files over the reviewed rev5 tree 75b41cd2 (.research/260829_llamacpp-against-the-python-baseline.md, LOGBOOK.md, tools/mlx-swift-runtime-prototype/README.md) = 52 insertions, 17 deletions. Zero non-markdown files. Sources/Tests/scripts/examples are byte-identical to bafa676b and 75b41cd2. Final tree 98df2a3f; patch sha256 02a4dd15ceff759e6493d966ab92ee1d1b11ba81c8bdc768ab9b1dbdb405b88b, git apply --check against base 3272e3a exit 0.

VALIDATION re-run for this revision rather than inherited, real exit codes: swift build -c release 0; swift test -c release 0 (410 tests / 32 suites); scripts/benchmark-gate-smoke.sh 0 (120 PASS / 0 FAIL, BENCHMARK GATE SMOKE OK, 12m53s); swift-format lint --strict 0; bash -n 0; shellcheck -S warning 0. Reported honestly: an earlier smoke attempt exited 1 because I left BINARY unset — my invocation error, not a repo failure — and a background wrapper around the good run reported code 1 because its last statement was a grep for FAIL that legitimately found none.

NOT RUN: the pinned pair (benchmark-run) — the rev5 review said no rerun is needed and the brief forbids it. Go tests — no Go source in this delta.

OPEN FOR REVIEW: routing recordWarmupMemory/recordSoak through coveredPeak (or dropping scoredBytes from those windows) is instrumentation debt this revision reports rather than fixes, since it changes no measurement code. Recorded as decision item 0 in report section 9.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-cd38a1, pid=5077, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260830-1806bb, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260830-1806bb)
reviewer rev6 verdict: ACCEPTED (CR-TASK-260829-3k4qrc-6, base 3272e3ae, candidate 98df2a3f, patch sha256 verified). Delta is 3 .md files, 52 insertions/17 deletions; Sources/Tests/scripts/examples byte-identical to bafa676b; worktree == candidate tree. Both rev5 blocking findings fixed and verified against raw records. Item 1 settled by a stronger method than rev5: full route enumeration from the pinned libllama-server-impl.dylib (b10621-c1d0e7a00) plus server-source proof that n_ubatch/n_batch never appear as a JSON key anywhere (log format strings only) and reasoning_effort is inbound-request-only, never in a server-state response; /props payload read at server-context.cpp:4576-4610. contextPolicy refusal is irreducible. All readings reproduce from records: overlap -21.088s, prompt-token parity 1.0000 on all six, speculation off both sides, context_75k completionTokens 16/16. Memory arithmetic reproduces exactly (soakMemory 19/19 over both bounds on both sides; +6,201,119,136 B is entirely Mach anonymous, mapped constant at 28,239,409,972 B). Validation re-run on this tree: build 0, 410 tests/32 suites, lint clean, gate smoke 120 PASS/0 FAIL. Judgement: (b) proceed to the article. context_75k refuses on the Mach bound on BOTH sides, so a self-calibrating mapped bound cannot score it; the 125ms Mach breach is caused by load the 73k prefill itself generates (hostLoadAverageMax 13.995). New supporting measurement: the mapped component takes 11 distinct values spanning 2.26-3.64 MB on the baseline and 3 (effectively static) on the candidate, so the 2.2-5.8s external vmmap fork policies a rounding error / a constant - a design defect, not a calibration error. The only version of (a) worth backing is reading the mapped component in-process (proc_pidinfo / mach_vm_region), not re-deriving the bound upward. Non-blocking nits recorded in the verdict: refusal enumeration says all 26 partial windows when the pair has 25 partial + 1 measured (board artifacts only, not repo docs); correction note says 51 insertions vs actual 52; dangling §7.1 reference; a third ungated RuntimeMemoryPeak(summarizing:) at BenchmarkFootprintSampler.swift:364 in sampleCounts() that is correct by design and must not be routed through coveredPeak.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260830-1806bb, pid=36040, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260829-d6b68d.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260829-d6b68d.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260829-a88f5c.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260829-a88f5c.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_results.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_results.md) — Revision 2 instrumentation closure, literal validation exits, and preserved revision 1 measurements
- [TASK-260829-3k4qrc_evidence.tar.gz](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_evidence.tar.gz) — Sealed records, attestations, runtime logs, failed-attempt cadence evidence, and validation logs
- [TASK-260829-3k4qrc_change-request_rev1.patch](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev1.patch) — Change Request CR-TASK-260829-3k4qrc-1 revision 1 candidate patch (repository_delta=present, 7 changed paths)
- [TASK-260829-3k4qrc_change-request_rev1-validation.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev1-validation.log) — Change Request CR-TASK-260829-3k4qrc-1 revision 1 bounded validation log
- [TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--codex-_RUN-260829-f78621.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--codex-_RUN-260829-f78621.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_review-verdict.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_review-verdict.md) — Revision 1 reviewer verdict: changes requested with raw-number verification, KV fixability analysis, sampler peak-capture attack, and rerun requirements
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-62370c.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-62370c.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-332081.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-332081.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-01f7fa.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-01f7fa.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_benchmark-gate-smoke_rev2.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_benchmark-gate-smoke_rev2.log) — Revision 2 production smoke: exit 0, zero failures, mmap accounting and transient probe green
- [TASK-260829-3k4qrc_change-request_rev2.patch](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev2.patch) — Change Request CR-TASK-260829-3k4qrc-2 revision 2 candidate patch (repository_delta=present, 15 changed paths)
- [TASK-260829-3k4qrc_change-request_rev2-validation.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev2-validation.log) — Change Request CR-TASK-260829-3k4qrc-2 revision 2 bounded validation log
- [TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--codex-_RUN-260830-bcb7b6.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--codex-_RUN-260830-bcb7b6.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_review-verdict-rev2.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_review-verdict-rev2.md) — Revision 2 changes-requested verdict: mmap freshness bypass, uncalled cache-comparability rule, negative attacks, and validation
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-6d8cd4.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--codex-_RUN-260830-6d8cd4.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_results_rev3.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_results_rev3.md) — Revision 3 implementation, negative production evidence, validation exits, and explicit no-rerun scope
- [TASK-260829-3k4qrc_benchmark-gate-smoke-rev3-green.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_benchmark-gate-smoke-rev3-green.log) — Revision 3 full production-entry smoke: exit 0, 118 checks, 0 failures
- [TASK-260829-3k4qrc_benchmark-gate-smoke-rev3-red.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_benchmark-gate-smoke-rev3-red.log) — Preserved first full smoke: exit 1 from an overly narrow smoke assertion; product gate refused the window correctly
- [TASK-260829-3k4qrc_change-request_rev3.patch](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev3.patch) — Change Request CR-TASK-260829-3k4qrc-3 revision 3 candidate patch (repository_delta=present, 18 changed paths)
- [TASK-260829-3k4qrc_change-request_rev3-validation.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev3-validation.log) — Change Request CR-TASK-260829-3k4qrc-3 revision 3 bounded validation log
- [TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--codex-_RUN-260830-065450.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--codex-_RUN-260830-065450.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-abcd6d.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-abcd6d.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_review-verdict-rev3.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_review-verdict-rev3.md) — Revision 3 review verdict: changes requested; scored memory dimension has an empty admissible set
- [TASK-260829-3k4qrc_reviewer-rev3-mapped-probe.json](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_reviewer-rev3-mapped-probe.json) — Reviewer rev3: production benchmark-mapped-file-sampler-probe output showing partial/refused window
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--claude-_RUN-260830-39dc95.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--claude-_RUN-260830-39dc95.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_change-request_rev4.patch](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev4.patch) — Change Request CR-TASK-260829-3k4qrc-4 revision 4 candidate patch (repository_delta=present, 18 changed paths)
- [TASK-260829-3k4qrc_change-request_rev4-validation.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev4-validation.log) — Change Request CR-TASK-260829-3k4qrc-4 revision 4 bounded validation log
- [TASK-260829-3k4qrc_benchmark-gate-smoke-rev4.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_benchmark-gate-smoke-rev4.log) — Revision 4 gate smoke run 3: 119 PASS / 1 FAIL, every admitted-path check at exit 0
- [TASK-260829-3k4qrc_rev4-probe-evidence.tar.gz](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_rev4-probe-evidence.tar.gz) — Revision 4 raw evidence: four probe records, the teardown mutant record, vmmap reader-cost and Mach-blackout measurements, and the pre-fix smoke run
- [TASK-260829-3k4qrc_results_rev4.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_results_rev4.md) — Revision 4 results: F3 derived mapped-file bound, F4 acceptance-path evidence, teardown defect
- [TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-d9668d.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-d9668d.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_review-verdict-rev4.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_review-verdict-rev4.md) — Revision 4 review verdict: accepted. F3/F4 closed, mutants executed, fresh full smoke 120 PASS/0 FAIL, cached-tokens unknown closed by live llama-server probe.
- [TASK-260829-3k4qrc_reviewer-rev4-evidence.tar.gz](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_reviewer-rev4-evidence.tar.gz) — Reviewer revision-4 evidence: fresh full gate smoke (120 PASS/0 FAIL) plus the perturbed first run, accepted decision.json, live llama-server streamed cached-tokens transcripts.
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--claude-_RUN-260830-aae9ac.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--claude-_RUN-260830-aae9ac.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_tables-rev4.txt](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_tables-rev4.txt) — Rendered number set for session-rev4: pins, sealed intervals, prompt-token parity, sealed cache telemetry, per-scenario metrics and per-window memory status.
- [TASK-260829-3k4qrc_rev4-evidence.tar.gz](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_rev4-evidence.tar.gz) — Raw evidence: both run records with timestamped memory series, attestations, runtime logs, session.json, host sweeps, raw process lists, llama.cpp live-surface probes, validation logs. SHA-256 103e9a26cbc92d5c2b9547b74167f4518694b174e33d6ec6fd2e862a43d11f70
- [TASK-260829-3k4qrc_measured-pair-outcome.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_measured-pair-outcome.md) — Revision 6: corrected pair outcome report — refusal accounting now names the four ungated measured windows (new 4.0), the failed-read sentence replaced with the synthetic coverage marker it actually is, context_75k decode withdrawn. Documentation-only; measurements unchanged.
- [TASK-260829-3k4qrc_change-request_rev5.patch](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev5.patch) — Change Request CR-TASK-260829-3k4qrc-5 revision 5 candidate patch (repository_delta=present, 18 changed paths)
- [TASK-260829-3k4qrc_change-request_rev5-validation.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev5-validation.log) — Change Request CR-TASK-260829-3k4qrc-5 revision 5 bounded validation log
- [TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-08b2c8.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-08b2c8.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_review-verdict-rev5.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_review-verdict-rev5.md) — Revision 5 review verdict: changes requested, two reporting defects, plus the (a)/(b) recommendation
- [TASK-260829-3k4qrc_reviewer-rev5-probe-evidence.tar.gz](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_reviewer-rev5-probe-evidence.tar.gz) — Reviewer's own llama-server live-surface probe (build b10621, --metrics included): all endpoint payloads, notes, server log
- [TASK-260829-3k4qrc_spawn-log_-implementer--developer--claude-_RUN-260830-cd38a1.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-implementer--developer--claude-_RUN-260830-cd38a1.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_rev6-correction-note.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_rev6-correction-note.md) — CR rev6 correction note: both blocking findings and the context_75k withdrawal, each verified against session-rev4 raw data before writing.
- [TASK-260829-3k4qrc_change-request_rev6.patch](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev6.patch) — Change Request CR-TASK-260829-3k4qrc-6 revision 6 candidate patch (repository_delta=present, 18 changed paths)
- [TASK-260829-3k4qrc_change-request_rev6-validation.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_change-request_rev6-validation.log) — Change Request CR-TASK-260829-3k4qrc-6 revision 6 bounded validation log
- [TASK-260829-3k4qrc_benchmark-gate-smoke-rev6.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_benchmark-gate-smoke-rev6.log) — Production gate smoke re-run for CR rev6: BENCHMARK GATE SMOKE OK (0 failures), 120 PASS / 0 FAIL, exit 0, 12m53s.
- [TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-1806bb.log](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_spawn-log_-reviewer--reviewer--claude-_RUN-260830-1806bb.log) — System spawn log captured by task-board
- [TASK-260829-3k4qrc_review-verdict-rev6.md](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_review-verdict-rev6.md) — Revision 6 reviewer verdict: accepted. Whole-surface llama-server reportability proof, full recomputation from raw records, (a)/(b) judgement with article constraints.
- [TASK-260829-3k4qrc_reviewer-rev6-evidence.tar.gz](file://TASK-260829-3k4qrc/TASK-260829-3k4qrc_reviewer-rev6-evidence.tar.gz) — Reviewer rev6 evidence: pinned-dylib route enumeration + server-source reportability proof, and full recomputation of intervals, pins, readings, gated/ungated memory windows and mapped-component variability

## Created
2026-08-29T10:39:12Z

## Last Update
2026-08-30T22:05:27Z

## Assigned To
[reviewer] reviewer (claude)
