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
- TASK-260828-2jbufw

## Checklist
- [x] Pinned llama.cpp revision recorded verbatim and reproducible from a written build or install command
- [x] Qwen GGUF weights of the closest available quantization staged under src/local-models with source URL and content hash
- [x] Quantization relationship to the MLX 8-bit group64 baseline stated explicitly, including where it is NOT equivalent
- [x] Weights load and answer one bounded prompt through llama-server before the task is called done
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
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Acquisition task whose real difficulty is judging quantization equivalence honestly rather than staging files; must also sequence a large download around another run holding the host's memory."}
spawn selection rationale for claude-opus-5/high: Acquisition task whose real difficulty is judging quantization equivalence honestly rather than staging files; must also sequence a large download around another run holding the host's memory.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260828-74f439, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260828-74f439)
Install done: Homebrew llama.cpp 0.3.0 = upstream tag v0.3.0, commit c1d0e7a004015f23bc0233470b747b596f29b264, build 10621, released 2026-08-25 and currently the latest upstream release. brew pin applied. Binary confirms qwen35 arch and qwen35 graph_mtp / nextn support.

Weights: orcarouter/Qwen3.8-27B-Uncensored-GGUF is a GATED repo and this host has no HF token, so the first-party Q8_0 is not downloadable. Staging chimingw/Qwen3.8-27B-Uncensored-OrcaRouter-GGUF Q8_0 instead: ungated, produced by llama-quantize directly from the same orcarouter F16 GGUF, and byte-size identical to the gated first-party file at 29047084416 bytes, as are its Q4_K_M/Q5_K_M/Q6_K siblings.

Equivalence work in progress. Established so far by reading the artifacts directly: MLX 8-bit affine g64 with uint8 + bf16 scale + bf16 bias, and GGUF Q8_0 with int8 + fp16 scale over block 32, are both exactly 8.5 bits per weight. The GGUF quantized-tensor set equals the MLX quantized-tensor set plus 8 MTP-block tensors. Two confirmed NON-equivalences so far: first, the MLX 8-bit build DROPPED the MTP head that exists in the BF16 source while the GGUF keeps it as blk.64 nextn; second, MLX carries the vision tower in-file in bf16 while the GGUF text model has zero vision tensors and vision lives in a separate 931 MB f16 mmproj.
RESULT — ready for review.

RUNTIME PIN. Homebrew llama.cpp 0.3.0 = upstream tag v0.3.0, commit c1d0e7a004015f23bc0233470b747b596f29b264, build 10621, released 2026-08-25 and currently the latest upstream release. brew pin llama.cpp applied. Reproducible from the immutable formula at homebrew-core e266bd3ebd650c00225c9252934f3193d39f0767, which pins that exact upstream revision. libllama.dylib carries the qwen35 architecture and llama_model_qwen35 graph_mtp, which this model requires.

WEIGHTS. /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/ with PROVENANCE.md. Q8_0 sha256 31756fca94beca71ea4b8706d6fdc896dab2a3c6376ab0c1863b98512a24f8d6 and mmproj sha256 add205b7bfdb3f71f6da36b0a82aa20928dd829a920878c602628cdfbebc5288, both recomputed locally and matching the publisher manifest. The first-party orcarouter GGUF repo is gated and this host has no HF token, so the ungated chimingw mirror was used; its manifest records llama-quantize Q8_0 from the same orcarouter F16 GGUF and its four quant files are byte-size identical to the gated originals. Byte-identity to the gated file is NOT verifiable and is not claimed.

EQUIVALENCE — COMPARABLE, WITH THREE CONDITIONS. Both schemes cost exactly 8.5 bits per weight: MLX affine g64 pays 8 bits plus bf16 scale and bf16 zero point per 64; Q8_0 pays 8 bits plus fp16 scale per 32, symmetric. Measured against the shared BF16 source of record over 18 tensors, mean relative-RMS ratio 0.766 over the 16 paired tensors — the GGUF is slightly MORE faithful than the MLX build. Quantized-tensor sets match exactly: MLX 498, GGUF 506 = the same 498 plus 8 MTP tensors. Language-model weight bytes with MTP excluded: 28579478528 vs 28584769536, a 0.02 percent difference.

WHERE IT IS NOT EQUIVALENT. 1. The MLX 8-bit build DROPPED the MTP head that exists in the BF16 source; the GGUF keeps it as blk.64 with nextn_predict_layers=1. llama.cpp can therefore run MTP speculative decoding and the MLX baseline structurally cannot, so any tokens/s measured with MTP enabled is a different algorithm and must not be compared as a runtime difference. Build 10621 skips all 15 blk.64 tensors as unused on the default path, exactly 451319808 bytes, so this is a file-size and capability difference, not a default-path memory difference. 2. Vision placement: MLX carries the tower in-file in bf16 at 921460192 bytes, resident whether used or not; the GGUF text model has zero vision tensors and vision is a separate 931145984-byte mmproj loaded only with --mmproj. A naive RSS comparison silently compares different things. 3. Norms and 1-D tensors are bf16 in MLX and F32 in GGUF — lossless widening, 10686464 extra bytes. Also: the qwen35 converter permutes the GDN value-head axis, so tensor-by-tensor tooling cannot assume matching indices.

LOAD-AND-ANSWER. Passed at 15:08:59 after waiting about 16 minutes from 14:52 for TASK-260827-2v13w8 to release the host. The check refuses to start while anything listens on 18000-18999 or an mlx_lm / prototype / benchmark process is alive, and requires 35 GiB reclaimable; it ran only once the host reported free with 51.7 GiB. No other run process was signalled. Ready in 45 s, curl rc=0, finish_reason stop, content "The capital of Armenia is Yerevan.", MTP not enabled.

NEGATIVE EVIDENCE. This task added no product-code gate; the two gates live in the delivered analysis tooling and are named here. Call site 1: quant_equivalence.py try_align, invoked per tensor from main before any error is reported. Call site 2: the 3x ratio verdict in quant_equivalence.py main, which decides COMPARABLE vs NOT COMPARABLE. test_alignment_guard.py exercises both with mutants that must be rejected: blk.0 GGUF against blk.1 BF16 is refused as MISMATCH rather than force-aligned; a BF16 tensor re-rounded to the E4M3 grid — what an FP8-derived GGUF would look like — is rejected at ratio 3.889 while the real artifact passes at 0.790 and a 1.5x-noise mutant still passes at 1.184, so the gate narrows rather than merely existing. Exit 0, 7/7 cases, re-run byte-identical after lint fixes.

VALIDATION. go build ./... exit 0 and go vet ./... exit 0 in tools/agents-infra. ruff check exit 0 and shellcheck exit 0 on the delivered scripts. quant_equivalence.py exit 0, test_alignment_guard.py exit 0, load_and_answer.sh exit 0. Both scripts re-run after the lint fixes with identical output.

SCOPE. No repository code changed; the only tracked edit is the LOGBOOK.md entry. profiles.qwen-local remains on Python mlx_lm.server and nothing installed here became a default.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260828-74f439, pid=2926, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Independent-provider review of a quantization-equivalence judgement that decides whether a whole later comparison is valid; the numbers favour the challenger, so an independent check matters more than usual."}
spawn selection rationale for gpt-5.6-sol/xhigh: Independent-provider review of a quantization-equivalence judgement that decides whether a whole later comparison is valid; the numbers favour the challenger, so an independent check matters more than usual.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260828-c12ea2, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260828-c12ea2)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260828-c12ea2, pid=84494, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Rework correcting an equivalence judgement in both directions: a residency claim that overstated MLX memory cost, and a provenance claim that understated how well the staged weights are attested."}
spawn selection rationale for claude-opus-5/high: Rework correcting an equivalence judgement in both directions: a residency claim that overstated MLX memory cost, and a provenance claim that understated how well the staged weights are attested.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260828-3ee053, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260828-3ee053)
REV2 — rework of CR revision 1. All five findings addressed; the COMPARABLE-with-conditions verdict stands, corrected where it was wrong and strengthened where the reviewer found better evidence.

F3 (residency) — CORRECTED, and now measured rather than asserted. Confirmed the reviewer: mlx_lm.models.qwen3_5.Model.sanitize drops every vision_tower.*/model.visual.* key, and mlx_lm.utils.load_model sanitizes at offset 2524 before load_weights at 5320 and before mx.eval(model.parameters()) at 5403. Went further and measured it: a real text-only mlx_lm.load of the 8-bit build gives 1847 parameter tensors / 28579478528 bytes, mx.get_active_memory() 28579478536, ZERO resident vision tensors, 921460192 bytes never materialised. vision_residency.py also runs a mutant of Model.sanitize with the vision branch deleted over the same key set — it keeps all 333 vision keys, so the filter is load-bearing, not decorative. Report section 5.2 rewritten to separate on-disk placement from runtime residency; 5.4 marks every on-disk-only row; benchmark condition 2 changed from "note the MLX vision tower is resident-but-unused" to "compare measured footprint, not file membership; do NOT adjust either side by 921/931 MB". LOGBOOK corrected in a new 1557 entry (the 1510 entry is left intact per append-only). Also found: peak RSS is the wrong metric here — 22305783808 then 18627723264 bytes across two runs of the identical load, because the weights are mmap-backed; mx.get_active_memory() was stable to the byte.

F4 (first-party byte identity) — CORRECTED IN OUR FAVOUR, and it is stronger than the reviewer stated. https://huggingface.co/api/models/orcarouter/Qwen3.8-27B-Uncensored-GGUF?blobs=true returns HTTP 200 at revision a855f377abf5cbda99a278414466743f427e97c8 and publishes the LFS SHA-256 for BOTH files, not just the text model: Q8_0 31756fca94beca71ea4b8706d6fdc896dab2a3c6376ab0c1863b98512a24f8d6 and mmproj add205b7bfdb3f71f6da36b0a82aa20928dd829a920878c602628cdfbebc5288. Recomputed both locally over the whole files (shasum -a 256, 95 s) — identical. Byte identity to the first-party artifacts is ESTABLISHED under SHA-256. The separate fact is preserved: the gated blob still returns HTTP 401 without a token, reproduced again at rev2. Updated the staged PROVENANCE.md (synced byte-identical to /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/PROVENANCE.md, sha256 19a085db…), the board copy, report section 2, and the logbook.

F1 (FP8 negative did not exercise production) — FIXED AND MUTANT-KILLED. There is now one comparability decision, quant_equivalence.comparability_verdict() over RATIO_CEIL = 3.0; main() calls it and holds no ratio comparison of its own; the module is __main__-guarded so the test imports the real module instead of exec-ing a truncated copy. test_alignment_guard.py declares NO threshold and drives that function. 10/10, exit 0. The reviewer mutant is dead: RATIO_CEIL 3.0 -> 4.0 makes the suite exit 1 with both FP8 cases failing (mutant-ratio-4.0.log). A second mutant — main() deciding inline again — also exits 1 on the two call-site binding cases (mutant-callsite.log). Added cases the reviewer did not ask for: one FP8 tensor among three good ones is still NOT COMPARABLE (a mean would have hidden it), and a NaN ratio is NOT COMPARABLE.

F2 (attestation exited zero for a wrong answer) — FIXED AND MUTANT-KILLED. Reproduced the defect first: the rev1 script with only the binary made injectable exits 0 on "The capital of Armenia is Baku." (mutant-rev1-binseam.log). load_and_answer.sh now exits non-zero on curl failure (7), unexpected JSON shape or zero completion tokens (8), finish_reason != stop (9), empty or wrong answer (10), early server exit (5). The prompt and required answer are hard-coded; no env var can make it accept a wrong answer. test_load_answer_gate.sh drives the real script with fakes, 11 cases, exit 0, stable over five consecutive runs. Mutants: deleting the assertion fails 6 of 11; widening the expected answer from \byerevan\b to \bcapital\b — just enough to admit "…is Baku." — fails exactly one case, so the gate bounds the answer itself.

F5 (not reproducible from its shebang) — FIXED. Pinned interpreter /Users/alexis/.local/pipx/venvs/mlx-lm/bin/python (CPython 3.14.7, numpy 2.5.2, mlx_lm 0.31.3) recorded in every script docstring, with a numpy-missing guard that prints the exact command, plus a new report section 8 giving the exact command for every artifact. Also found a related reproducibility gap the review did not flag: rev1 ran 512 rows per tensor and the log recorded it only as a header line; rev2 runs the 1024-row default. Per-tensor ratios move in the third decimal (token_embd 0.761 -> 0.754), the 0.766 mean does not. Section 4 now states this.

ANOMALY worth flagging to the next reviewer. During this rework six load-gate cases "passed" as rejections while actually exiting 3 — the host-contention guard firing on TASK-260827-2v13w8-rev4 smoke holding port 18799 and a neighbouring mlx-swift build, not the answer gate. Asserting merely "non-zero" hid it. The suite now asserts the EXACT exit code, plants its own port/process sentinels and scans only those for the answer cases, and runs three further cases at the UNTOUCHED defaults (planted listener on 18950 named in the log; planted runtime-benchmark.py named in the log; and a control proving both refusals came from the plants). load_and_answer.sh itself keeps the global defaults.

LOAD-AND-ANSWER STATUS — read this carefully. The green real run (load-and-answer-01.log, exit 0 at 15:53, ready ~10s warm, finish_reason stop, correct answer, assertion enforced) was produced BEFORE the two additive contention knobs were added. load-and-answer-asrun.diff is the full diff between that as-run script and the attached artifact: the delta is entirely inside the pre-launch contention guard; the launch, request and answer-assertion path is byte-identical. A fresh real run was attempted at the end of this rework and correctly REFUSED with exit 3 because TASK-260827-2v13w8-rev4 smoke still holds port 18799 after 23+ minutes. I did not signal or kill that process and did not narrow the guard to get past it. The final artifact answer path is separately exercised end to end by the 11-case suite, including a positive control that reaches the assertion and exits 0.

VALIDATION. go build ./... exit 0 and go vet ./... exit 0 in tools/agents-infra. ruff check exit 0 over quant_equivalence.py, test_alignment_guard.py, vision_residency.py, fake_llama_server.py, gguf_inspect.py. shellcheck exit 0 over load_and_answer.sh, test_load_answer_gate.sh, dead_llama_server.sh. quant_equivalence.py exit 0 (mean 0.766). test_alignment_guard.py exit 0 (10/10). vision_residency.py --measure exit 0. test_load_answer_gate.sh exit 0 (11/11, five repeats).

SCOPE. Repository delta is still one file: LOGBOOK.md, now +24 lines (the new 1557 entry). git diff --check exit 0. No product code changed; profiles.qwen-local remains on Python mlx_lm.server and nothing installed here became a default.
REV2 ATTESTATION UPDATE — the real load-and-answer was rerun with the exact delivered artifact at untouched defaults and passed. load-and-answer-03.log, exit 0 at 16:36:07-16:36:31, free 42.9 GiB, ready ~25s warm, curl rc=0, finish_reason stop, content "The capital of Armenia is Yerevan.", OK: bounded answer asserted, MTP not enabled. This supersedes the earlier note about the as-run script: load-and-answer-01.log and load-and-answer-asrun.diff are kept as history only.

Before that it refused correctly. load-and-answer-refused-03.log, exit 3 at 16:35:48, because TASK-260827-2v13w8-rev4 smoke held port 18799 continuously for 33+ minutes. The check waited about 25 minutes for that neighbour to release the host; the process was not signalled, not killed, and the guard was not narrowed to get past it. Both the refusal and the pass are attached, so the guard is evidenced live at its defaults in both directions.

Final sweep on the delivered artifacts: ruff exit 0, shellcheck exit 0, test_alignment_guard.py exit 0 (10/10), test_load_answer_gate.sh exit 0 (11/11), vision_residency.py exit 0, go build and go vet exit 0. All 14 board resources verified sha256-identical to the local artifacts, and the staged PROVENANCE.md next to the weights is byte-identical to the board copy (19a085db…).
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260828-3ee053, pid=33551, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Second-round review by the author of the corrections, checking that a residency claim corrected downward did not erase the real on-disk difference alongside it."}
spawn selection rationale for gpt-5.6-sol/xhigh: Second-round review by the author of the corrections, checking that a residency claim corrected downward did not erase the real on-disk difference alongside it.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260828-fba9f5, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260828-fba9f5)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260828-fba9f5, pid=62934, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Narrow rework binding a negative test to the production entry path after a source-inspection assertion was defeated by a dead-call mutant."}
spawn selection rationale for claude-opus-5/high: Narrow rework binding a negative test to the production entry path after a source-inspection assertion was defeated by a dead-call mutant.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260828-153384, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260828-153384)
rev3 — F1 closed, nothing else reopened.

WHAT WAS WRONG: rev2 claimed main() was bound to comparability_verdict() by inspecting source strings. Review kept a dead `if False: comparability_verdict(rows)` and decided inline at RATIO_CEIL + 1.0 — the threshold is a NAME, so the regex never saw it — and the whole suite exited 0 while main() admitted the FP8 ratio 3.889.

FIX: quant_equivalence.py grew exactly one seam, collect_rows(n_rows); main() takes its rows from it and decides exactly as before. test_alignment_guard.py replaces that seam and runs the REAL main(), asserting the verdict line it printed (COMPARABLE / NOT COMPARABLE / INCOMPLETE) AND its exit status, for all seven comparability cases. No test calls comparability_verdict() directly; both source-string checks are deleted.

EVIDENCE (all commands run directly, real exit codes):
- test_alignment_guard.py -> exit 0, 16 checks, 0 failures (guard-tests-04.log). The report now claims the count the suite prints; rev1/rev2 said 10 while 12 ran.
- gate 2 through main(): real row -> COMPARABLE/0; FP8 3.889 -> NOT COMPARABLE/1; one FP8 among three good -> NOT COMPARABLE/1; NaN ratio -> NOT COMPARABLE/1; 1.5x-noise -> COMPARABLE/0; MLX-absent -> COMPARABLE/0; unreadable tensor -> INCOMPLETE/2 (a failed read is not an absence).
- mutant A, RATIO_CEIL 3.0->4.0 applied to the real file, suite byte-unchanged: exit 1, 3 failures incl. the FP8 case (mutant-rev3-ratio.log).
- mutant B, reviewers own dead-call bypass applied to the real file: exit 1, 3 failures incl. the FP8 case (mutant-rev3-bypass.log).
- both mutants are also named in-suite cases, each asserting its own mutation applied at exactly one site and loads; an unbuildable mutant fails rather than passing silently.
- control case records that rev2 syntactic inspection still reports mutant B clean — which is why gate 2 observes behaviour.
- refactor regression: quant_equivalence.py 1024-row rerun exit 0, byte-identical to the rev2 log (quant-equivalence-04.log): 16 paired, mean 0.766, COMPARABLE.
- ruff check on both files: exit 0.

NOT RERUN: load_and_answer.sh / test_load_answer_gate.sh are byte-identical to the accepted rev2 artifacts and rev3 does not touch them. One rerun attempt at 17:05 REFUSED with exit 3 (load-and-answer-04.log) — TASK-260827-2v13w8-rev4 held port 18031 with mlx_lm-relux.server and the mlx-swift prototype live. Host still held at 17:17. Neighbour not signalled, scans not narrowed.

CHANGE REQUEST rev3: base 132c5997f9ad8a82358d03d7a08a23eff46bcf9d; repository delta LOGBOOK.md only, +39 lines (two entries, 1715 and 1710); patch sha256 edda55429904a6be23944922cbb8b9e4cb7bf68bede167a262481ca4c4c33fa1; git diff --check exit 0. Scripts, logs and the report are board resources, not repository files.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260828-153384, pid=14668, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Third-round review scoped to one question: whether the negative now witnesses the production decision rather than the source text, tested with the reviewer's own bypass mutant."}
spawn selection rationale for gpt-5.6-sol/xhigh: Third-round review scoped to one question: whether the negative now witnesses the production decision rather than the source text, tested with the reviewer's own bypass mutant.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260828-b955ad, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260828-b955ad)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260828-b955ad, pid=50911, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260828-3g87i4_spawn-log_-implementer--developer--claude-_RUN-260828-74f439.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_spawn-log_-implementer--developer--claude-_RUN-260828-74f439.log) — System spawn log captured by task-board
- [TASK-260828-3g87i4_quant_equivalence.py](file://TASK-260828-3g87i4/TASK-260828-3g87i4_quant_equivalence.py) — rev3: main() now takes rows from the collect_rows seam; single comparability decision unchanged
- [TASK-260828-3g87i4_test_alignment_guard.py](file://TASK-260828-3g87i4/TASK-260828-3g87i4_test_alignment_guard.py) — rev3: all comparability cases drive the real main() and assert its emitted verdict + exit status; ratio and call-site-bypass mutants are named cases
- [TASK-260828-3g87i4_quant-equivalence-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_quant-equivalence-01.log) — Full run over 18 tensors, exit 0, mean ratio 0.766
- [TASK-260828-3g87i4_guard-tests-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_guard-tests-01.log) — Guard negative tests, exit 0, 7/7 cases; FP8 mutant rejected at ratio 3.889
- [TASK-260828-3g87i4_weight-budget-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_weight-budget-01.log) — Weight-byte budget per build; language-model weights differ by 0.02 percent with MTP excluded
- [TASK-260828-3g87i4_quantization-equivalence.md](file://TASK-260828-3g87i4/TASK-260828-3g87i4_quantization-equivalence.md) — rev3 equivalence report: production-entry-path evidence in 4.2/4.3, check count corrected to the 16 the suite prints, load-and-answer rerun status recorded in 7
- [TASK-260828-3g87i4_load-and-answer-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_load-and-answer-01.log) — Real llama-server load and one bounded prompt, exit 0 at 15:53, ready ~10s warm, finish_reason stop, correct answer, assertion enforced. See load-and-answer-asrun.diff for which script revision produced it.
- [TASK-260828-3g87i4_llama-server-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_llama-server-01.log) — Raw llama-server log from the final 16:36 run; all 15 blk.64 MTP tensors skipped as unused on the default path
- [TASK-260828-3g87i4_load_and_answer.sh](file://TASK-260828-3g87i4/TASK-260828-3g87i4_load_and_answer.sh) — rev2: fails closed on curl error (7), bad body/zero tokens (8), finish_reason != stop (9), empty or wrong answer (10), early server exit (5). Prompt and required answer are not overridable.
- [TASK-260828-3g87i4_guard-tests-02.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_guard-tests-02.log) — Guard negative tests re-run after lint fixes, exit 0, output byte-identical to rev 01
- [TASK-260828-3g87i4_quant-equivalence-02.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_quant-equivalence-02.log) — Equivalence report re-run after lint fixes, exit 0, summary identical to rev 01
- [TASK-260828-3g87i4_gguf_inspect.py](file://TASK-260828-3g87i4/TASK-260828-3g87i4_gguf_inspect.py) — Minimal GGUF header/tensor-info reader used to inventory the Q8_0 and mmproj files; works on a partially downloaded file
- [TASK-260828-3g87i4_PROVENANCE.md](file://TASK-260828-3g87i4/TASK-260828-3g87i4_PROVENANCE.md) — rev2 copy of the provenance file staged next to the weights: byte identity to the first-party gated artifacts now established under SHA-256 via the public metadata endpoint; vision placement no longer described as a memory difference.
- [TASK-260828-3g87i4_change-request_rev1.patch](file://TASK-260828-3g87i4/TASK-260828-3g87i4_change-request_rev1.patch) — Change Request CR-TASK-260828-3g87i4-1 revision 1 candidate patch (repository_delta=present, 1 changed paths)
- [TASK-260828-3g87i4_spawn-log_-reviewer--reviewer--codex-_RUN-260828-c12ea2.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_spawn-log_-reviewer--reviewer--codex-_RUN-260828-c12ea2.log) — System spawn log captured by task-board
- [TASK-260828-3g87i4_review-verdict.md](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-verdict.md) — Reviewer verdict for Change Request revision 1: changes requested, with production-bound narrowing mutants, artifact verification, and exact rework requirements
- [TASK-260828-3g87i4_spawn-log_-implementer--developer--claude-_RUN-260828-3ee053.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_spawn-log_-implementer--developer--claude-_RUN-260828-3ee053.log) — System spawn log captured by task-board
- [TASK-260828-3g87i4_vision_residency.py](file://TASK-260828-3g87i4/TASK-260828-3g87i4_vision_residency.py) — Separates vision file-membership from runtime residency: safetensors accounting, the real mlx_lm sanitize over the real key set with a vision-branch-deleted mutant, and a measured text-only load. Exit 0.
- [TASK-260828-3g87i4_test_load_answer_gate.sh](file://TASK-260828-3g87i4/TASK-260828-3g87i4_test_load_answer_gate.sh) — Negative tests driving the real load_and_answer.sh with fakes; asserts EXACT exit codes so a host-contention refusal cannot pass as an answer-gate rejection. 11 cases incl. a positive control and three at the untouched contention defaults.
- [TASK-260828-3g87i4_fake_llama_server.py](file://TASK-260828-3g87i4/TASK-260828-3g87i4_fake_llama_server.py) — Healthy llama-server-shaped HTTP fake whose answer is chosen by FAKE_MODE (right/wrong/truncated/empty/malformed/notjson/zerotokens). Injected via LLAMA_SERVER_BIN.
- [TASK-260828-3g87i4_quant-equivalence-03.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_quant-equivalence-03.log) — rev2 equivalence report, exit 0, 1024 rows per tensor, mean ratio 0.766 over 16 paired tensors
- [TASK-260828-3g87i4_guard-tests-03.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_guard-tests-03.log) — rev2 guard negative tests against the production verdict, exit 0, 10/10; FP8 mutant rejected at 3.889 by comparability_verdict itself
- [TASK-260828-3g87i4_mutant-ratio-4.0.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_mutant-ratio-4.0.log) — MUTANT KILL: production RATIO_CEIL moved 3.0 -> 4.0; guard suite exits 1 with both FP8 cases failing. This is the mutant that survived at rev1.
- [TASK-260828-3g87i4_mutant-callsite.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_mutant-callsite.log) — MUTANT KILL: main() decides inline instead of calling comparability_verdict; guard suite exits 1 on both call-site binding cases.
- [TASK-260828-3g87i4_load-gate-tests-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_load-gate-tests-01.log) — rev2 load-and-answer gate negative tests driving the real script, exit 0, 11/11, exact exit codes asserted, stable over five consecutive runs
- [TASK-260828-3g87i4_mutant-load-noassert.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_mutant-load-noassert.log) — MUTANT KILL: answer assertion deleted from load_and_answer.sh; suite exits 1 with 6 of 11 cases failing.
- [TASK-260828-3g87i4_mutant-load-weak.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_mutant-load-weak.log) — NARROWING MUTANT KILL: expected answer widened from \\byerevan\\b to \\bcapital\\b, enough to admit 'the capital of Armenia is Baku.'; exactly one case fails.
- [TASK-260828-3g87i4_mutant-rev1-binseam.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_mutant-rev1-binseam.log) — Reproduction of the reviewer's F2: the rev1 script with only the binary made injectable exits 0 on a confidently wrong answer.
- [TASK-260828-3g87i4_vision-residency-02.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_vision-residency-02.log) — Measured MLX text-only residency: 1847 parameter tensors / 28579478528 bytes, zero resident vision tensors, 921460192 bytes never materialised. Exit 0.
- [TASK-260828-3g87i4_firstparty-lfs-metadata.json](file://TASK-260828-3g87i4/TASK-260828-3g87i4_firstparty-lfs-metadata.json) — HTTP 200 metadata from the GATED first-party repo orcarouter/Qwen3.8-27B-Uncensored-GGUF with blobs=true: publishes the LFS SHA-256 that matches both staged files exactly.
- [TASK-260828-3g87i4_load-and-answer-asrun.diff](file://TASK-260828-3g87i4/TASK-260828-3g87i4_load-and-answer-asrun.diff) — Diff between the script that produced load-and-answer-02.log and the attached artifact: the delta is entirely inside the pre-launch contention guard; the launch, request and answer-assertion path is byte-identical.
- [TASK-260828-3g87i4_dead_llama_server.sh](file://TASK-260828-3g87i4/TASK-260828-3g87i4_dead_llama_server.sh) — A server that exits immediately, as llama.cpp does on a corrupt or missing model; used by the negative suite's early-exit case.
- [TASK-260828-3g87i4_load-and-answer-refused-03.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_load-and-answer-refused-03.log) — Fresh real attempt with the FINAL artifact at untouched defaults: exit 3, guard refused because TASK-260827-2v13w8-rev4 smoke still held port 18799 after 33 min. The neighbour was not signalled and the guard was not narrowed.
- [TASK-260828-3g87i4_load-and-answer-03.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_load-and-answer-03.log) — ATTESTATION for the delivered artifact: real llama-server load and one bounded prompt with the final load_and_answer.sh at untouched defaults, exit 0 at 16:36, ready ~25s, finish_reason stop, correct answer asserted.
- [TASK-260828-3g87i4_change-request_rev2.patch](file://TASK-260828-3g87i4/TASK-260828-3g87i4_change-request_rev2.patch) — Change Request CR-TASK-260828-3g87i4-2 revision 2 candidate patch (repository_delta=present, 1 changed paths)
- [TASK-260828-3g87i4_spawn-log_-reviewer--reviewer--codex-_RUN-260828-fba9f5.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_spawn-log_-reviewer--reviewer--codex-_RUN-260828-fba9f5.log) — System spawn log captured by task-board
- [TASK-260828-3g87i4_review-verdict-rev2.md](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-verdict-rev2.md) — Revision 2 reviewer verdict: changes requested because a narrowed production call-site bypass keeps the FP8 suite green; F2-F5 independently accepted.
- [TASK-260828-3g87i4_spawn-log_-implementer--developer--claude-_RUN-260828-153384.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_spawn-log_-implementer--developer--claude-_RUN-260828-153384.log) — System spawn log captured by task-board
- [TASK-260828-3g87i4_guard-tests-04.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_guard-tests-04.log) — rev3 guard suite: 16 checks, 0 failures, exit 0
- [TASK-260828-3g87i4_quant-equivalence-04.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_quant-equivalence-04.log) — rev3 full 1024-row analysis after the collect_rows refactor: byte-identical to the rev2 run, mean ratio 0.766, COMPARABLE, exit 0
- [TASK-260828-3g87i4_mutant-rev3-ratio.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_mutant-rev3-ratio.log) — production mutant A (RATIO_CEIL 3.0->4.0) applied to the real file: unchanged suite exits 1 with the FP8 gate-2 case failing
- [TASK-260828-3g87i4_mutant-rev3-bypass.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_mutant-rev3-bypass.log) — production mutant B (review's dead comparability_verdict call + inline CLI decision) applied to the real file: unchanged suite exits 1 with the FP8 gate-2 case failing
- [TASK-260828-3g87i4_load-and-answer-04.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_load-and-answer-04.log) — rev3 load-and-answer rerun attempt at 17:05: guard refused with exit 3, host held by TASK-260827-2v13w8-rev4; scans not narrowed
- [TASK-260828-3g87i4_rev3-results.md](file://TASK-260828-3g87i4/TASK-260828-3g87i4_rev3-results.md) — rev3 implementation notes: what F1 was, the collect_rows seam, the 16-check suite, and both production mutant kills
- [TASK-260828-3g87i4_change-request_rev3.patch](file://TASK-260828-3g87i4/TASK-260828-3g87i4_change-request_rev3.patch) — Change Request CR-TASK-260828-3g87i4-3 revision 3 candidate patch (repository_delta=present, 1 changed paths)
- [TASK-260828-3g87i4_spawn-log_-reviewer--reviewer--codex-_RUN-260828-b955ad.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_spawn-log_-reviewer--reviewer--codex-_RUN-260828-b955ad.log) — System spawn log captured by task-board
- [TASK-260828-3g87i4_review-verdict-rev3.md](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-verdict-rev3.md) — Revision 3 reviewer verdict: accepted after independent production-entry bypass and ratio mutant kills
- [TASK-260828-3g87i4_review-rev3-baseline-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-rev3-baseline-01.log) — Reviewer baseline on exact board artifacts: exit 0, 16 checks, 0 failures
- [TASK-260828-3g87i4_review-rev3-mutant-bypass-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-rev3-mutant-bypass-01.log) — Independent dead-call and looser-threshold production mutant: unchanged suite exits 1 at the FP8 main() assertion
- [TASK-260828-3g87i4_review-rev3-mutant-ratio-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-rev3-mutant-ratio-01.log) — Independent RATIO_CEIL 3.0 to 4.0 production mutant: unchanged suite exits 1 at the FP8 main() assertion
- [TASK-260828-3g87i4_review-rev3-mutant-bypass.diff](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-rev3-mutant-bypass.diff) — Exact reviewer call-site bypass mutation applied to the board artifact copy
- [TASK-260828-3g87i4_review-rev3-mutant-ratio.diff](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-rev3-mutant-ratio.diff) — Exact reviewer RATIO_CEIL mutation applied to the board artifact copy
- [TASK-260828-3g87i4_review-rev3-ruff-resource-mode-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-rev3-ruff-resource-mode-01.log) — Ruff on resource-get 0644 materializations: EXE001 only, documenting transport mode loss
- [TASK-260828-3g87i4_review-rev3-ruff-execmode-01.log](file://TASK-260828-3g87i4/TASK-260828-3g87i4_review-rev3-ruff-execmode-01.log) — Ruff on byte-identical producer-mode 0755 copies: all checks passed

## Created
2026-08-28T10:12:55Z

## Last Update
2026-08-28T14:34:35Z

## Assigned To
[reviewer] reviewer (codex)
