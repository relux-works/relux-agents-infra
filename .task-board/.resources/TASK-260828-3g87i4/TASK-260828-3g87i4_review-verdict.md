# TASK-260828-3g87i4 review verdict

Change Request: `CR-TASK-260828-3g87i4-1`, revision `1`  
Verdict: **CHANGES REQUESTED** → `to-dev`

Candidate reviewed exactly as handed off:

- Base: `132c5997f9ad8a82358d03d7a08a23eff46bcf9d`
- Candidate tree: `d87021ff62751657411ad2c6646c7264affa7270`
- Patch SHA-256: `89c4fe0fcb28c9ab76e078c09ead83869f812b869995ec45a64e1db4c24e33b8` (recomputed; matches the Change Request)
- Repository delta: only `LOGBOOK.md`, 12 inserted lines; `git diff --check` passed

## Blocking findings

### F1 — Critical: the FP8 negative test does not exercise the production verdict

Shape: **check present but uncalled from production**, exposed by a narrowing mutant.

Production decides comparability in `quant_equivalence.py:265` with the literal condition `r > 3.0`. The attached `test_alignment_guard.py:28` declares a separate `RATIO_CEIL = 3.0` and evaluates the synthetic FP8 value against that test-local constant. It never calls the production verdict.

Reviewer reproduction:

1. Original test: FP8 mutant ratio `3.889`, exit `0`, 7/7.
2. Reviewer scratch mutant: production threshold changed only from `3.0` to `4.0`.
3. Under that production mutant, a ratio of `3.889` would be admitted as comparable.
4. The full attached test still exited `0`, 7/7, because it continued to compare against its private `3.0`.

This defeats the exact gate the review brief required. Rework must put the threshold and verdict in one production function used by `main`, then drive that production entry point with the FP8-derived case. A `3.0 → 4.0` mutant must make the named negative test fail. Do not repair this by merely asserting two duplicated constants are equal.

### F2 — High: the load-and-answer attestation exits zero for a wrong answer

Shape: **capability claim that does not reproduce** / positive-path-only attestation.

The production artifact `load_and_answer.sh` uses `set -u`, prints `curl rc`, `finish_reason`, and content, but asserts none of them. Its final `grep | head` pipeline can return success after an invalid response.

Reviewer scratch mutant replaced only the absolute `BIN` with a healthy local HTTP fake. The fake returned:

```text
finish_reason: stop
content: 'This is deliberately the wrong answer.'
```

`load_and_answer.sh` exited `0` (`mutant_exit=0`). Rework must fail closed on non-zero curl, malformed JSON, non-stop completion, empty/wrong bounded answer, and early server exit. Add a negative test that drives the exact script/entry point with a wrong-answer fake and requires non-zero.

The real capability itself is present: the reviewer independently reran the real `llama-server` check and received `The capital of Armenia is Yerevan.`, `finish_reason=stop`, exit `0`. This finding is about the delivered attestation gate, not the model's ability to answer.

### F3 — High: `resident regardless of use` is false for the MLX vision tower

The artifact inventory confirms that the MLX checkpoint contains exactly `921,460,192` bytes of `vision_tower.*` tensors on disk. It does **not** establish that those bytes are resident in a text-only run.

All installed relevant MLX-LM environments (`0.31.3` and both `0.32.0` environments) implement `mlx_lm.models.qwen3_5.Model.sanitize` by dropping `vision_tower.*` and `model.visual.*`. `mlx_lm.utils.load_model` sanitizes before `model.load_weights(...)` and before `mx.eval(model.parameters())`. Therefore the text-only Python baseline does not evaluate the vision tensors into resident model parameters. The adjacent Swift text-only factory likewise reports that it drops vision weights.

The report partly acknowledges this in section 5.2 (`text-only ... does not load it`) but section 6 and the tracked `LOGBOOK.md` entry revert to `resident` / `resident-but-unused`. That contradiction directly distorts the owner's memory-economy criterion.

Rework must distinguish:

- checkpoint/on-disk placement: MLX embeds 921,460,192 bytes; GGUF keeps a separate 931,145,984-byte mmproj;
- runtime residency: determined by the selected model factory and measured physical footprint, not by file membership.

Do not add or subtract 921 MB from an RSS/physical-footprint comparison unless the executed factory actually loads it.

### F4 — Medium: first-party byte identity is verifiable and matches

The staged source is more strongly attested than reported. The first-party gated Hugging Face metadata endpoint with `blobs=true` publicly exposes the LFS SHA-256 for `Qwen3.8-27B-Uncensored-Q8_0.gguf`:

`31756fca94beca71ea4b8706d6fdc896dab2a3c6376ab0c1863b98512a24f8d6`

That is identical to both the mirror LFS digest and the reviewer's full local SHA-256 recomputation. The unauthenticated first-party blob request still returns HTTP `401`, but downloading the blob is not required to compare its published content digest. Byte identity to the first-party Q8_0 artifact is therefore established under SHA-256; it is not unknown.

Primary evidence:

- First-party metadata: <https://huggingface.co/api/models/orcarouter/Qwen3.8-27B-Uncensored-GGUF?blobs=true>
- Mirror revision metadata: <https://huggingface.co/api/models/chimingw/Qwen3.8-27B-Uncensored-OrcaRouter-GGUF/revision/58ebd123013160600229eda180b5b17f3fb7af9d?blobs=true>
- Mirror manifest: <https://huggingface.co/chimingw/Qwen3.8-27B-Uncensored-OrcaRouter-GGUF/raw/58ebd123013160600229eda180b5b17f3fb7af9d/MANIFEST.json>

Update staged `PROVENANCE.md`, the board analysis resource, and the append-only project logbook with this favourable correction. Preserve the separate fact that the gated blob itself is not downloadable without authorization.

### F5 — Medium: the analysis command is not reproducible from its shebang on this host

`#!/usr/bin/env python3` resolves to `/opt/homebrew/bin/python3` 3.14.7 here, which has no NumPy; direct execution fails with `ModuleNotFoundError`. The reviewer reproduced the analysis with the existing `/Users/alexis/.local/pipx/venvs/mlx-lm/bin/python` environment (`numpy 2.5.2`). Record an exact interpreter/dependency command or make the artifact self-describing enough that the next reviewer can run it without discovering a private environment.

## Independently confirmed evidence

- Runtime pin: installed and Homebrew-pinned `llama.cpp 0.3.0`; `llama-server --version` reports build `10621`, commit `c1d0e7a00`. Immutable Homebrew formula `e266bd3e...` pins tag `v0.3.0` and commit `c1d0e7a004015f23bc0233470b747b596f29b264`. The official latest-release endpoint reports the same tag/commit.
- Capability: installed `libllama.dylib` contains `qwen35`, `qwen35::graph_mtp`, and `nextn_predict_layers`; `llama-server --help` exposes `draft-mtp` while the default `--spec-type` is `none`.
- Local staged sizes: `29,047,084,416` and `931,145,984` bytes.
- Full local SHA-256 recomputation: `31756fca...24f8d6` and `add205b7...bc5288`, matching mirror LFS metadata; the text-model hash also matches first-party gated LFS metadata exactly.
- Independent header inventory: GGUF `506` Q8_0 tensors, of which `498` are non-MTP and `8` are MTP; MLX has `498` scale tensors and `498` bias tensors; BF16 has `15` MTP tensors while MLX has `0`.
- Bit cost: MLX affine g64 is `8 + (16 + 16) / 64 = 8.5` bits/weight; Q8_0 is `(16 + 32×8) / 32 = 8.5` bits/weight.
- Favourable fidelity result reproduced from the local artifacts: 16 paired tensors, mean relative-RMS ratio `0.765866767` from printed errors, all 16 GGUF errors lower than MLX (`0.75037...0.79484`). Rounded report value `0.766` is correct.
- MTP default path: the real server log reports all `15` `blk.64.*` tensors unused; their reported sizes sum to exactly `451,319,808` bytes. Default load did not enable speculative decoding.
- Real load-and-answer: reviewer rerun passed; ready after about 10 seconds from warm cache, correct bounded answer, and no server/listener survived teardown.
- Host coexistence: producer script can signal only its own captured `$SRV` PID. The neighboring tracked run completed normally after the producer's check, and no evidence shows another process was killed or signalled. The producer's 51.7 GiB and reviewer's 47.5 GiB preflight checks both found no protected listener/process before starting.

## Required rework and next review

1. Bind FP8 rejection to the production verdict and kill the `3.0 → 4.0` mutant.
2. Make the real load-and-answer script fail closed; kill the wrong-answer fake.
3. Correct vision placement versus residency everywhere, especially the benchmark conditions and `LOGBOOK.md`.
4. Replace the first-party `unverifiable` claim with the matching public LFS digest evidence.
5. Record the exact runnable Python environment/command and rerun the updated analysis and negative checks.
6. Update existing task-scoped resources rather than silently adding conflicting copies, publish Change Request revision 2, and send it through another reviewer cycle.

No candidate repository file was modified by this review. All attack mutants and probes live under the ignored task-scoped `.temp/TASK-260828-3g87i4/` tree. Because the reviewer must not drift the handed candidate, the `logbook` skill's repository write is intentionally deferred to producer rework; the correction is fully specified above.
