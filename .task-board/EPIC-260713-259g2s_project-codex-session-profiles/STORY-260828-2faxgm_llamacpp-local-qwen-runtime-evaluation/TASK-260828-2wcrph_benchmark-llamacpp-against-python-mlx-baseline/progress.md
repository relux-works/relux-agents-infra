## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- TASK-260828-2jbufw
- TASK-260828-3fgca3

## Blocks
- TASK-260828-15ftgj
- TASK-260829-3cwcb6

## Checklist
- [x] All pinned scenarios run on the same host with no other model resident, stated in the report
- [x] Prefill, decode, TTFT and peak physical footprint recorded per scenario; resident_bytes not used for sizing
- [x] 75000-token capacity scenario executed and its outcome recorded, including failure mode if it fails
- [x] Tool-call parity and long-running stability recorded, with prompt-token skew checked before any latency number is compared
- [x] Non-comparable pairs refused by the gate rather than scored, and every refusal reported
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
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"The comparison the whole evaluation exists for: six scenarios on the 28 GB model against the Python baseline through the hardened gate, with judgement needed about MTP, quantization asymmetries and what the numbers mean."}
spawn selection rationale for claude-opus-5/high: The comparison the whole evaluation exists for: six scenarios on the 28 GB model against the Python baseline through the hardened gate, with judgement needed about MTP, quantization asymmetries and what the numbers mean.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260828-d1d2bd, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260828-d1d2bd)
RESULT: the pinned six-scenario suite ran three times end to end, llama-server b10621-c1d0e7a00 vs mlx_lm-relux.server, same host, sequential, ps sweeps printed (none) before and after all three. Every scenario succeeded on both runtimes including context_75k at 73,016 prompt tokens; prompt-token parity exact (1.0000) on all six; tool-call parity and the 20-iteration soak passed on both. All three invocations exited 4, UNSCORED.

REFUSAL (correct, not a misconfiguration): contextPolicy differs, kv=unbounded vs kv=8192/kv=76800. mlx_lm-relux.server has no flag that bounds its KV cache and reports no bound; llama-server has no unbounded mode. No argv on either side can make them agree, so llama.cpp can never be scored against this Python incumbent -- the only admissible baseline is the Swift prototype, which has --max-kv-size. examples/model-harness.benchmark.toml told the operator to match the baseline --max-kv-size; that instruction was unfulfillable and is corrected.

THREE llama.cpp readings the gate could not make:
1. FIXED + mutant-proved: RuntimeSpeculation.read consulted slot.params and fell back to the slot only when params was ABSENT. On this build speculative is top-level and params appears only after a slot has served a request, so the 20-iteration soak blinded it: speculation=unread, refused. Worse, {speculative:true, params:{speculative:false}} read as reported(false) -- a drafting runtime scored as MTP-off. Both post-fix runs record speculation=off at the production entry.
2. REPORTED NOT FIXED (the dangerous one -- it emits a number rather than refusing): TTFT/prefill/decode are not comparable across this pair. mlx_lm publishes delta.reasoning, llama-server publishes delta.reasoning_content, so llama.cpp is clocked after its whole think block. Recorded decode 80.79 tok/s vs 8.88 measured = 9.1x overstatement; true decode 8.88 vs 9.85 means llama.cpp is ~10% SLOWER at decode, not 8x faster. Corrected rates measured outside the gate and reported.
3. Memory: ri_phys_footprint cannot see mmap-loaded weights. Same process, same moment: footprint 1.41 GiB, ps RSS 28.09 GiB, vmmap mapped-file resident 26.6 GiB with 0 dirty. maxPeakFootprintRatio would have scored llama.cpp at ~0.08x and called it 12x more frugal.

OWNER QUESTIONS: llama.cpp is modestly faster (0.81x-0.97x wall clock, driven by prefill not decode) and modestly more frugal (provably below the baseline at context_75k, 41.05 vs 45.20 GiB, ~9%; indeterminate at small working sets). Not an order of magnitude either way.

CODE: RuntimeAttestation.swift (speculation reader), +5 tests in RuntimeObservationReadingTests, boundedMLXArgv -> boundedSwiftArgv +2 tests in RuntimeBenchmarkContextBoundTests (the old fixture described an mlx_lm.server launch carrying a flag it does not have), examples/model-harness.benchmark.toml, README.md, LOGBOOK.md, .research/260829_llamacpp-against-the-python-baseline.md. unpinnableConditions untouched.

VERIFICATION, each run as its own process: swift build exit 0; swift test 392 tests/30 suites exit 0; xcodebuild -configuration Release exit 0; benchmark-gate-smoke.sh 0 failures exit 0; swift format lint --strict --recursive Sources Tests exit 0; shellcheck scripts/benchmark-gate-smoke.sh exit 0; git diff --check exit 0. Three mutants applied to shipped source, all killed (pre-fix reader; narrowing absence-as-answer; contextPolicy dropped from Pins.firstMismatch).

UNCOMMITTED: version_control.confirm is true, so the worktree changes are left uncommitted for the orchestrator.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260828-d1d2bd, pid=50316, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Independent review of the comparison that decides a runtime migration, where every measurement defect found so far pulled in the same direction and the review's central task is to test that asymmetry."}
spawn selection rationale for gpt-5.6-sol/high: Independent review of the comparison that decides a runtime migration, where every measurement defect found so far pulled in the same direction and the review's central task is to test that asymmetry.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-e00bc4, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-e00bc4)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-e00bc4, pid=13780, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Rework replacing an unsupported decode figure derived from SSE frame counts with either an authoritative token-count measurement or an honest unknown, and correcting a directional-bias claim the reviewer disproved."}
spawn selection rationale for gpt-5.6-sol/high: Rework replacing an unsupported decode figure derived from SSE frame counts with either an authoritative token-count measurement or an honest unknown, and correcting a directional-bias claim the reviewer disproved.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-99f19e, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-99f19e)
REWORK REVISION 2: Replaced unsupported frame-count-derived decode claims with a paired probe that explicitly requested streamed usage. Both runtimes returned 7784 prompt / 106 completion / 7890 total tokens. Python: TTFT 103.310s, prefill 75.346 tok/s, decode 8.449 tok/s. llama.cpp: TTFT 76.784s, prefill 101.375 tok/s, decode 8.772 tok/s. The 1.038x decode delta is one bounded observation, not a general direction. Raw timestamped SSE, summaries, probe and four negative tests attached as TASK-260828-2wcrph_rework-evidence.tgz; standalone verifier independently recomputed both rates. Removed the unsupported prior figures and decodes-far-faster attribution from report/README/logbook. Recorded mixed direction: omitted reasoning_content inflates llama TTFT, understates prefill, overstates decode. All gate latency remains excluded; TASK-260829-3cwcb6 owns production fix/full rerun. Verification exits: probe tests 0; py_compile 0; raw verifier 0; targeted Swift 21/21 and 19/19 exit 0; full Swift 392/30 exit 0; swift build 0; xcodebuild 0; format 0; shellcheck 0; production smoke with Xcode product and absolute OUT 0 (0 failures); git diff --check 0. Smoke invocations without BINARY and with default relative OUT each failed exit 1 and are not counted green.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-99f19e, pid=38533, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Second-round review of a correction that flipped the measured direction in the challenger's favour; symmetry of the paired probe is the acceptance question."}
spawn selection rationale for gpt-5.6-sol/high: Second-round review of a correction that flipped the measured direction in the challenger's favour; symmetry of the paired probe is the acceptance question.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260829-0ca566, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260829-0ca566)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260829-0ca566, pid=33465, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260828-2wcrph_spawn-log_-implementer--developer--claude-_RUN-260828-d1d2bd.log](file://TASK-260828-2wcrph/TASK-260828-2wcrph_spawn-log_-implementer--developer--claude-_RUN-260828-d1d2bd.log) — System spawn log captured by task-board
- [TASK-260828-2wcrph_report.md](file://TASK-260828-2wcrph/TASK-260828-2wcrph_report.md) — Revised llama.cpp vs Python mlx-lm report with authoritative streamed-usage decode evidence and mixed-direction threat correction
- [TASK-260828-2wcrph_evidence.tgz](file://TASK-260828-2wcrph/TASK-260828-2wcrph_evidence.tgz) — Three benchmark-run sessions (records, attestations, runtime logs), run logs with host sweeps, and every probe: /slots placement, memory accounting, SSE first-token timing
- [TASK-260828-2wcrph_tables-c76800.txt](file://TASK-260828-2wcrph/TASK-260828-2wcrph_tables-c76800.txt) — Per-scenario measurements for the full six-scenario run including the 73,016-token capacity probe
- [TASK-260828-2wcrph_tables-c8192.txt](file://TASK-260828-2wcrph/TASK-260828-2wcrph_tables-c8192.txt) — Per-scenario measurements at the matched 8k KV arena
- [TASK-260828-2wcrph_change-request_rev1.patch](file://TASK-260828-2wcrph/TASK-260828-2wcrph_change-request_rev1.patch) — Change Request CR-TASK-260828-2wcrph-1 revision 1 candidate patch (repository_delta=present, 7 changed paths)
- [TASK-260828-2wcrph_spawn-log_-reviewer--reviewer--codex-_RUN-260829-e00bc4.log](file://TASK-260828-2wcrph/TASK-260828-2wcrph_spawn-log_-reviewer--reviewer--codex-_RUN-260829-e00bc4.log) — System spawn log captured by task-board
- [TASK-260828-2wcrph_review-verdict.md](file://TASK-260828-2wcrph/TASK-260828-2wcrph_review-verdict.md) — Reviewer changes-requested verdict: corrected decode uses SSE frame count as an unproved token proxy; rerun and revise required
- [TASK-260828-2wcrph_spawn-log_-implementer--developer--codex-_RUN-260829-99f19e.log](file://TASK-260828-2wcrph/TASK-260828-2wcrph_spawn-log_-implementer--developer--codex-_RUN-260829-99f19e.log) — System spawn log captured by task-board
- [TASK-260828-2wcrph_rework-results.md](file://TASK-260828-2wcrph/TASK-260828-2wcrph_rework-results.md) — Authoritative streamed-usage rerun, corrected decode interpretation, decision boundary, and validation exits
- [TASK-260828-2wcrph_rework-evidence.tgz](file://TASK-260828-2wcrph/TASK-260828-2wcrph_rework-evidence.tgz) — Raw timestamped SSE for both runtimes, authoritative usage summaries, probe, negative tests, and rework report
- [TASK-260828-2wcrph_change-request_rev2.patch](file://TASK-260828-2wcrph/TASK-260828-2wcrph_change-request_rev2.patch) — Change Request CR-TASK-260828-2wcrph-2 revision 2 candidate patch (repository_delta=present, 7 changed paths)
- [TASK-260828-2wcrph_spawn-log_-reviewer--reviewer--codex-_RUN-260829-0ca566.log](file://TASK-260828-2wcrph/TASK-260828-2wcrph_spawn-log_-reviewer--reviewer--codex-_RUN-260829-0ca566.log) — System spawn log captured by task-board
- [TASK-260828-2wcrph_review-verdict-rev2.md](file://TASK-260828-2wcrph/TASK-260828-2wcrph_review-verdict-rev2.md) — Revision 2 acceptance verdict with independent raw-SSE recomputation and symmetry audit

## Created
2026-08-28T10:12:56Z

## Last Update
2026-08-29T11:33:50Z

## Assigned To
[reviewer] reviewer (codex)
