# TASK-260828-3g87i4 review verdict — revision 2

Change Request: `CR-TASK-260828-3g87i4-2`, revision `2`  
Verdict: **CHANGES REQUESTED** → `to-dev`

Candidate reviewed exactly as handed off:

- Base OID: `132c5997f9ad8a82358d03d7a08a23eff46bcf9d`
- Candidate tree OID: `11c8e3f066593182cd7cf0bb9775c68fff0e51eb`
- Repository delta: `LOGBOOK.md`, 24 inserted lines
- Patch SHA-256: `35fcfc98ce7824e7842f52d05b17abdfda19e3cf5c497c7ef7031bd7232900fe` (recomputed; matches the Change Request)
- `git diff --check` passed

## Blocking finding

### F1 remains incomplete: a call-site bypass keeps the suite green

Shape: **check present but uncalled from production** / **bypass path around the check**, exposed by a narrowing mutant.

Revision 2 fixes the original defect in part. The attached test imports the real `quant_equivalence.py`, calls its production `comparability_verdict()`, declares no private threshold, and the reviewer reproduced both expected results:

- original suite: exit `0`; real Q8_0 ratio `0.790` → `COMPARABLE`; FP8-grid ratio `3.889` → `NOT COMPARABLE`;
- production `RATIO_CEIL` mutant `3.0 → 4.0`: suite exit `1`; both FP8 cases fail.

The remaining problem is the claimed binding to `main()`. `test_alignment_guard.py` does not drive `main()` with the FP8 row. It calls `comparability_verdict()` directly, then treats two source-string checks as call-site evidence:

- source contains `comparability_verdict(`;
- source contains no numeric `> N` pattern matching its regex.

Reviewer narrowing mutant, made only in ignored scratch:

1. Keep `if False: comparability_verdict(rows)` in `main()`, so the syntactic call check passes.
2. Let the actual CLI path decide inline with `r > RATIO_CEIL + 1.0`, which admits the observed FP8 ratio `3.889`.
3. Run the complete attached `test_alignment_guard.py` unchanged.

Result: **exit `0`, every check PASS**, even though the actual `main()` path now admits the FP8-grid case. The direct function checks keep exercising an unused guard, and the regex does not see a duplicated symbolic threshold. This is the exact production-call-site failure shape the reviewer contract says must be rejected.

Required rework:

1. Make the FP8 negative drive the runtime decision used by `main()` — not a direct call plus source inspection. Inject controlled rows/dependencies into the production entry path and assert its emitted `COMPARABLE` / `NOT COMPARABLE` result and exit status.
2. Keep the existing `RATIO_CEIL 3.0 → 4.0` mutant kill.
3. Add a call-site bypass mutant that keeps a dead/ignored syntactic call while the CLI decides elsewhere; the named test must fail.
4. Correct the report/resource claim `10/10`: the current suite prints 12 named checks.

Because this is ordinary recoverable test/evidence rework, the correct route is `to-dev`, not `blocked`.

## Revision-2 fixes independently accepted

### F2 — load-and-answer now fails closed

- Reviewer reran the exact `load_and_answer.sh` through the attached fake-server harness.
- Confident wrong answer (`Baku`) returned exactly exit `10`.
- Full suite: 11 cases, 0 failures; correct answer positive control returned exit `0`.
- Reviewer widened the production answer regex from `Yerevan` to `capital`; the unchanged suite returned exit `1`, with the wrong-answer case becoming exit `0`. The narrowing mutant is killed.
- Attached real-run evidence shows the final script loaded the staged model through real `llama-server`, received `finish_reason: stop`, asserted `The capital of Armenia is Yerevan.`, and exited `0`. The reviewer did not repeat the 29 GB real load because the final-artifact run is already attached; the negative gate and its narrowing mutant were rerun independently.

### F3 — vision file membership and runtime residency are correctly separated

- Reviewer reran parts A+B against the real MLX checkpoint: 333 vision tensors / `921460192` bytes are present on disk; production `Model.sanitize` drops all 333 before `load_weights` and `mx.eval`; the vision-filter-deleted mutant keeps all 333.
- Attached measured part C reports 1847 resident parameter tensors / `28579478528` bytes and zero resident vision tensors.
- The revision-2 report preserves the real on-disk placement difference: MLX embeds the 921 MB tower; GGUF keeps a separate 931 MB mmproj. It no longer turns that fact into a default-path memory claim.
- The new append-only `LOGBOOK.md` entry explicitly retracts the earlier false residency statement while preserving the historical entry.

### F4 — first-party content identity and gated access are both preserved

- Local `PROVENANCE.md` is byte-identical to the board `TASK-260828-3g87i4_PROVENANCE.md` resource.
- First-party metadata resource is valid JSON at revision `a855f377abf5cbda99a278414466743f427e97c8`, `gated: auto`, and publishes both expected LFS SHA-256 values.
- Reviewer recomputed both complete local files:
  - Q8_0: `31756fca94beca71ea4b8706d6fdc896dab2a3c6376ab0c1863b98512a24f8d6`
  - mmproj: `add205b7bfdb3f71f6da36b0a82aa20928dd829a920878c602628cdfbebc5288`
- PROVENANCE, the board analysis resource, and the new logbook entry state first-party byte identity under SHA-256. All three separately retain that unauthenticated first-party blob download returns HTTP 401.

### F5 — runnable environment is recorded

- Recorded interpreter reproduced: CPython `3.14.7`, NumPy `2.5.2`, `mlx_lm 0.31.3` at `/Users/alexis/.local/pipx/venvs/mlx-lm/bin/python`.
- Both shell artifacts pass `bash -n`.

## Acceptance-criteria evidence retained

- Installed/pinned runtime independently verified: `llama.cpp 0.3.0`, build `10621`, commit prefix `c1d0e7a00`; `brew list --pinned` reports `llama.cpp`. The report records full upstream commit `c1d0e7a004015f23bc0233470b747b596f29b264` and reproducible Homebrew/source paths.
- Staged files exist under `/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/` at exactly `29047084416` and `931145984` bytes; full hashes match PROVENANCE and first-party LFS metadata.
- Attached 1024-row equivalence run reports 16 paired tensors, mean ratio `0.766`, and `COMPARABLE`; reviewer reran its focused 512-row real/FP8 guard rather than the full analysis.
- The report clearly retains the three material non-equivalences needed by the later benchmark: the MLX build dropped the MTP head; vision placement differs on disk but not default text-path residency; GGUF keeps norms/1-D tensors in F32 while MLX keeps them in BF16.
- Verdict remains `COMPARABLE` only with declared conditions: MTP off/separate, measured runtime footprint rather than file membership, and pinned sampling/tokenization.

No candidate repository file was modified by this review. All aliases and mutants were created under ignored `.temp/TASK-260828-3g87i4/`. The `logbook` skill would normally persist the surviving-gate finding, but the reviewer role is read-only; producer rework must append it to `LOGBOOK.md` in revision 3.
