# GGUF Q8_0 vs MLX 8-bit group64 — what is comparable and what is not

TASK-260828-3g87i4 · story STORY-260828-2faxgm · 2026-08-28 · **revision 3**

Everything below was measured against the artifacts on this host, not taken from
model cards. Scripts and raw logs are attached alongside this document; §8 gives
the exact interpreter and command for every one of them.

Revision 3 closes the one finding review left open, and changes nothing else:

* §4.2 / §4.3 — the FP8 negative now drives the **production entry path**.
  Revision 2 called `comparability_verdict()` directly and then inspected source
  strings to claim the call site was bound; review defeated that with a dead
  `if False:` call plus an inline decision at a looser threshold, and the suite
  stayed green while `main()` admitted the FP8 ratio 3.889. Every comparability
  case now injects rows through `collect_rows` and asserts what the real
  `main()` printed and exited with. Review's own bypass is a named mutant case,
  the `RATIO_CEIL` 3.0 → 4.0 kill is kept, and the check count in this report is
  now the count the suite prints (16, was claimed as 10 while 12 ran).

The verdict, the provenance attestation, the vision separation and the
load-and-answer gate are unchanged from revision 2 and were accepted there.

Revision 2 corrects three things review found in revision 1, two of which move
against the earlier text and one of which moves in its favour:

* §5.2 / §5.4 / §6 — "the MLX vision tower is resident whether you use it or not"
  was **wrong**. It is *in the file*; it is not resident under a text-only load.
  Now measured, not asserted (§5.2).
* §2 — "byte identity to the first-party artifact is not verifiable" was **too
  weak**. The gated repo publishes its LFS digests publicly and they match ours
  exactly (§2).
* §4 / §7 — both gates now run against the code that actually decides, and each
  is shown killing a mutant of that code (§4.3, §7.1).

---

## 1. Runtime pin

| | |
|---|---|
| Formula | Homebrew `llama.cpp` 0.3.0, bottle `arm64_tahoe` |
| Upstream tag | `v0.3.0` |
| Upstream commit | `c1d0e7a004015f23bc0233470b747b596f29b264` |
| Build number | **10621** |
| Released | 2026-08-25 — the current latest upstream release |
| homebrew-core tap head at install | `e266bd3ebd650c00225c9252934f3193d39f0767` |
| Installed | `/opt/homebrew/Cellar/llama.cpp/0.3.0` (38 binaries incl. `llama-server`, `llama-bench`, `llama-quantize`) |

Reported by the binary itself:

```
$ llama-server --version
version: 0.3.0 (build 10621, commit c1d0e7a00)
built with AppleClang 21.0.0.21000101 for Darwin arm64
```

Install and hold the pin:

```bash
brew install llama.cpp     # 0.3.0 at homebrew-core e266bd3ebd650c00225c9252934f3193d39f0767
brew pin llama.cpp         # applied; `brew list --pinned` now reports llama.cpp
```

The formula at that tap revision is immutable and pins the upstream revision
explicitly, so the exact build is reconstructible even after homebrew-core moves:

<https://raw.githubusercontent.com/Homebrew/homebrew-core/e266bd3ebd650c00225c9252934f3193d39f0767/Formula/l/llama.cpp.rb>

```ruby
url "https://github.com/ggml-org/llama.cpp.git",
    tag:      "v0.3.0",
    revision: "c1d0e7a004015f23bc0233470b747b596f29b264"
```

Equivalent from source: `git clone … && git checkout c1d0e7a004015f23bc0233470b747b596f29b264 && cmake -B build -DGGML_METAL=ON && cmake --build build -j`.

**This build supports the model.** The publisher requires a llama.cpp new enough
to have the `qwen35` hybrid-GDN architecture and the MTP/`nextn` head. Symbols in
the installed `libllama.dylib` confirm both: architecture strings `qwen35`,
`qwen35moe`; and `llama_model_qwen35::graph_mtp`, `%s.nextn_predict_layers`.

## 2. Weights staged

`/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/` — see
`PROVENANCE.md` in that directory for URLs, commands and hashes.

| File | Bytes | SHA-256, recomputed locally over the whole file |
|---|---:|---|
| `Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf` | 29 047 084 416 | `31756fca…24f8d6` |
| `mmproj-Qwen3.8-27B-Uncensored-OrcaRouter-F16.gguf` | 931 145 984 | `add205b7…bc5288` |

Downloaded from `chimingw/Qwen3.8-27B-Uncensored-OrcaRouter-GGUF` @ `58ebd123…7af9d`,
because the first-party repo is gated (below). Both hashes match that mirror's
LFS metadata and its `MANIFEST.json`.

### Byte identity to the first-party artifact IS established

`orcarouter/Qwen3.8-27B-Uncensored-GGUF` — same publisher as the BF16 and MLX
builds already on this host — is gated (`gated: auto`) and its **blobs** return
HTTP 401 without a Hugging Face token, which this host does not have. Revision 1
concluded from that that content identity to the first-party file was
unverifiable. That was wrong, and review caught it: downloading the blob is not
required to compare content, because the repo's **metadata** endpoint is public
and publishes the LFS SHA-256 of every file.

<https://huggingface.co/api/models/orcarouter/Qwen3.8-27B-Uncensored-GGUF?blobs=true>
— HTTP 200, revision `a855f377abf5cbda99a278414466743f427e97c8`:

| first-party file | published LFS SHA-256 | our local file |
|---|---|---|
| `Qwen3.8-27B-Uncensored-Q8_0.gguf` (29 047 084 416 B) | `31756fca94beca71ea4b8706d6fdc896dab2a3c6376ab0c1863b98512a24f8d6` | **identical** |
| `mmproj-Qwen3.8-27B-Uncensored-f16.gguf` (931 145 984 B) | `add205b7bfdb3f71f6da36b0a82aa20928dd829a920878c602628cdfbebc5288` | **identical** |

Both staged files are therefore **byte-identical to the first-party artifacts
under SHA-256**, for the text model and for the mmproj. The mirror is a mirror in
the strict sense, not merely a same-recipe rebuild — consistent with its
`MANIFEST.json`, which records `llama-quantize … Q8_0 64` from the same orcarouter
`Qwen3.8-27B-Uncensored-F16` GGUF.

Two facts that must stay apart:

* **content identity** — established, by matching published and recomputed
  SHA-256 digests;
* **access** — the gated blob is still not downloadable from this host without
  authorization (`HTTP 401`, reproduced again at revision 2). Anyone rebuilding
  this staging step from the first-party repo needs a token; from the mirror they
  do not, and they can verify they got the same bytes either way.

## 3. The two quantization schemes are different, at identical bit cost

| | MLX 8-bit baseline | GGUF Q8_0 |
|---|---|---|
| Storage | `uint8`, 4 per `uint32` word | `int8` |
| Scale | **bf16**, one per group | **fp16**, one per block |
| Zero point | **bf16 bias, one per group** | **none — symmetric** |
| Group / block | **64** | **32** |
| Bits per weight | 8 + (16+16)/64 = **8.5** | 8 + 16/32 = **8.5** |
| Reconstruction | `w = q·scale + bias` (affine) | `w = q·scale` (symmetric) |

Same 8.5 bits/weight, reached two different ways. Q8_0 buys a 2× finer block at
the cost of losing the zero point; MLX buys asymmetry at the cost of a coarser
group. Measured below, Q8_0's finer block wins slightly: it is consistently
~24 % *more* accurate than the MLX affine scheme against the same source.

Confirmed from the artifacts: MLX `config.json` → `{"bits": 8, "group_size": 64,
"mode": "affine"}`; scales/biases are `BF16` `[17408, 80]` next to a `U32`
`[17408, 1280]` packed weight for a 5120-wide row. GGUF →
`general.file_type = 7` (MOSTLY_Q8_0), 506 Q8_0 tensors + 360 F32.

## 4. Do they quantize the same numbers? — yes, verified

`quant_equivalence.py` dequantizes the same tensor from all three local artifacts
and measures each 8-bit build against the **BF16 source of record**
(`Qwen3.8-27B-Uncensored-BF16`). Neither 8-bit build is derived from the other,
so the shared BF16 is the only valid referee.

Relative RMS error vs BF16, 18 tensors spanning embeddings, output head, MLP,
full attention, GDN linear attention and the MTP block, early/mid/late layers.
Run at the default sample of 1024 rows per tensor
(`quant-equivalence-04.log`, byte-identical to the revision-2 `quant-equivalence-03.log`; command in §8):

| tensor | MLX vs BF16 | GGUF vs BF16 | ratio | note |
|---|---:|---:|---:|---|
| `token_embd.weight` | 0.0086334 | 0.0065130 | 0.754 | |
| `output.weight` | 0.0070824 | 0.0053527 | 0.756 | |
| `blk.0.ffn_gate.weight` | 0.0068307 | 0.0053992 | 0.790 | |
| `blk.0.ffn_down.weight` | 0.0068060 | 0.0054137 | 0.795 | |
| `blk.31.ffn_up.weight` | 0.0069638 | 0.0054633 | 0.785 | |
| `blk.63.ffn_down.weight` | 0.0074420 | 0.0057384 | 0.771 | |
| `blk.3.attn_q.weight` | 0.0070865 | 0.0054506 | 0.769 | |
| `blk.3.attn_k.weight` | 0.0074301 | 0.0055894 | 0.752 | |
| `blk.3.attn_v.weight` | 0.0071808 | 0.0054798 | 0.763 | |
| `blk.63.attn_output.weight` | 0.0077240 | 0.0059367 | 0.769 | |
| `blk.0.attn_qkv.weight` | 0.0071192 | 0.0053959 | 0.758 | output axis permuted |
| `blk.0.attn_gate.weight` | 0.0072480 | 0.0054387 | 0.750 | output axis permuted |
| `blk.0.ssm_alpha.weight` | 0.0070679 | 0.0053762 | 0.761 | output axis permuted |
| `blk.0.ssm_beta.weight` | 0.0070134 | 0.0053951 | 0.769 | output axis permuted |
| `blk.0.ssm_out.weight` | 0.0074596 | 0.0056278 | 0.754 | input axis permuted |
| `blk.62.ssm_out.weight` | 0.0074733 | 0.0056245 | 0.753 | input axis permuted |
| `blk.64.nextn.eh_proj.weight` | — | 0.0084117 | — | **absent from MLX build** |
| `blk.64.ffn_up.weight` | — | 0.0054082 | — | **absent from MLX build** |

**Mean ratio 0.766 over the 16 paired tensors.** Every paired tensor sits at
8-bit rounding noise against the same BF16 weights. There is no extra lossy hop
on the GGUF side — in particular the GGUF did **not** come through the
`orcarouter/…-FP8` checkpoint, which was the live risk given that the GGUF
model card links the FP8 repo as its base.

The sample size is part of the result: revision 1 ran at 512 rows and reported
mean 0.766 with `token_embd` at 0.761; at 1024 rows `token_embd` is 0.754 and the
mean is still 0.766. Per-tensor figures move in the third decimal with the
sample, the conclusion does not. The row count is printed in every log
(`rows sampled : …`) so a reader can always tell which run they are looking at.

### 4.1 The GDN layout difference is cosmetic

Five tensors initially compared as pure noise (cosine ≈ 0.04). They are not
different weights: llama.cpp's `qwen35` converter reorders the Gated-DeltaNet
value-head axis relative to the HF layout that both BF16 and MLX use. Once the
permutation is recovered they land at exactly the same 0.75–0.77 ratio as
everything else. This matters for anyone diffing tensors by index, and for
nothing else numerically.

### 4.2 Where the verdict is decided

There is exactly one comparability decision in the tool and one threshold behind
it: `quant_equivalence.comparability_verdict()` over `RATIO_CEIL = 3.0`.
`main()` prints that function's answer and exits on it; it holds no ratio
comparison of its own.

That claim is not asserted by reading the source. `main()` takes its per-tensor
rows from one seam, `collect_rows(n_rows)`, and the guard suite replaces that
seam and then runs **this file's real `main()`** — its table, its verdict line,
its exit status. Every case in §4.3 asserts what `main()` printed
(`COMPARABLE` / `NOT COMPARABLE` / `INCOMPLETE`) and the status it exited with.
Nothing calls `comparability_verdict()` directly and nothing greps production
source to decide whether the gate is wired up.

Revision 2 did grep the source, and review broke it: a mutant that kept a dead
`if False: comparability_verdict(rows)` while the CLI decided inline at
`RATIO_CEIL + 1.0` passed every syntactic check and admitted the FP8 ratio
3.889, with the whole suite still green. A check that cannot observe behaviour
cannot witness it. That exact mutant is now a named case (§4.3, mutant B).

### 4.3 Why you can believe the two verdicts above

`test_alignment_guard.py` — exit 0, **16 checks, 0 failures**
(`guard-tests-04.log`). The count printed by the suite is the count claimed
here; earlier revisions of this report said "10/10" while the suite printed 12.

Gate 1 — `try_align()` must not manufacture an alignment:

| case | expected | result |
|---|---|---|
| true pair aligns without a permutation | accept | rel_rms 0.0053962 ✔ |
| **`blk.0` GGUF vs `blk.1` BF16** — different weights | **refuse** | `MISMATCH`, rel_rms 1.3847 ✔ |
| genuine column permutation | recover | recovered ✔ |
| genuine row permutation | recover | recovered ✔ |

Gate 2 — the production entry path. Each row set is injected through
`collect_rows`, and the assertion is on what `main()` emitted and exited with:

| injected rows | `main()` must emit | exit | result |
|---|---|---|---|
| real GGUF row (ratio 0.790) | COMPARABLE | 0 | ✔ |
| **BF16 re-rounded to the E4M3 grid** (ratio 3.889) | **NOT COMPARABLE** | **1** | ✔ |
| **one FP8 row among three good ones** | **NOT COMPARABLE** | **1** | a mean would have hidden it ✔ |
| **a NaN ratio** | **NOT COMPARABLE** | **1** | not silently "below the ceiling" ✔ |
| row at 1.5× the real quantization noise (ratio 1.184) | COMPARABLE | 0 | gate is not reject-all ✔ |
| tensor absent from the MLX build | COMPARABLE | 0 | casts no vote ✔ |
| a tensor whose GGUF bytes could not be read | **INCOMPLETE** | **2** | a failed read is never an absence ✔ |

Gate 3 — mutants of the production file, each loaded as a real module and driven
through its own `main()`. A mutant that gate 2's assertions do not reject is a
hole in the suite, so each case fails unless the assertion flips to failing:

| mutant of the production code | in-suite result | same mutant applied to the real file |
|---|---|---|
| A: `RATIO_CEIL` 3.0 → 4.0 | mutant `main()` emits COMPARABLE / exit 0 → **killed** ✔ | suite **exit 1**, 3 failures (`mutant-rev3-ratio.log`) |
| B: dead `if False: comparability_verdict(rows)` + inline CLI decision at `RATIO_CEIL + 1.0` — review's own bypass | mutant `main()` emits COMPARABLE / exit 0 → **killed** ✔ | suite **exit 1**, 3 failures (`mutant-rev3-bypass.log`) |
| (control) revision 2's syntactic inspection applied to mutant B | reports it **clean** ✔ | — this is why gate 2 observes behaviour |

Each mutant case also asserts its own mutation actually applied — exactly one
site, source changed, module loads. A mutant that cannot be built proves
nothing, so it fails rather than passing silently: that is what the `mutant B
applied` case reports in the `mutant-rev3-bypass.log` run.

The suite declares no threshold of its own, so it cannot drift away from
production a second time. The permutation search is likewise not free to invent
an alignment: it accepts only a verified bijection whose residual falls to
rounding noise, and it refuses the wrong-layer pair. The verdict is not vacuous
either — it fires on a simulated FP8 upstream while still passing a row 1.5×
noisier than reality, so it discriminates the exact hypothesis it exists to test.

The `collect_rows` extraction is the only production change in revision 3. The
full 1024-row analysis rerun after it (`quant-equivalence-04.log`) is
byte-identical to the revision-2 run: 16 paired tensors, mean ratio 0.766,
COMPARABLE.

## 5. Where the two builds are NOT equivalent

These are real and the benchmark must account for them.

### 5.1 The MTP head — the significant one

The BF16 source ships a Multi-Token-Prediction head (`mtp.fc`, `mtp.layers.0.*`;
`config.json` → `mtp_num_hidden_layers: 1`).

* **GGUF keeps it.** `qwen35.block_count = 65`, `qwen35.nextn_predict_layers = 1`,
  block `blk.64` carries `nextn.eh_proj/enorm/hnorm/shared_head_norm` plus a full
  attention + FFN layer. 451 319 808 bytes.
* **The MLX 8-bit build dropped it.** Zero `mtp`/`nextn` tensors in
  `model.safetensors.index.json`. The MLX conversion discarded a head that exists
  upstream.

Consequences:

1. llama.cpp can run MTP speculative decoding on these weights. The MLX baseline
   structurally cannot. **Any decode-throughput number produced with llama.cpp's
   MTP/speculative path enabled is not comparable to the MLX baseline** — it is
   a different algorithm, not a faster runtime. Leave MTP off, or report it as a
   separate clearly-labelled configuration.
2. On disk the GGUF carries 451 319 808 bytes the MLX build does not. It is
   **not** resident by default: build 10621 logs `model has unused tensor
   blk.64.* -- ignoring` for all 15 tensors of that block when the MTP path is
   not enabled, and the byte sizes it reports sum to exactly 451 319 808. So
   default-path memory is unaffected; only file size and the MTP capability
   differ.

### 5.2 The vision tower — placement, not residency

Revision 1 said the MLX vision tower is "in the file whether or not you use it"
and then, in §6, turned that into "resident-but-unused". The second half was
**false** and it distorted exactly the criterion the benchmark exists to
measure. Two different facts, kept apart from here on.

**On-disk placement — a file-membership fact.**

* The MLX checkpoint contains 333 `vision_tower.*` tensors totalling
  **921 460 192 bytes**, unquantized bf16, inside the same safetensors shards as
  the language model.
* The GGUF text model has **zero** vision tensors. Vision is the separate
  931 145 984-byte mmproj, passed with `--mmproj` or not at all.

**Runtime residency — a measured fact, and it is zero on both sides.**

`mlx_lm.models.qwen3_5.Model.sanitize` drops every key starting `vision_tower`
or `model.visual`, and `mlx_lm.utils.load_model` calls `sanitize` *before*
`model.load_weights(...)` and *before* `mx.eval(model.parameters())`. Dropped
keys are therefore never evaluated into resident parameters. The adjacent
mlx-swift prototype runs `--model-factory text-only` and drops them too.

Measured on this host — `vision_residency.py --measure`, exit 0
(`vision-residency-02.log`):

| | |
|---|---:|
| tensors in the checkpoint | 2 180 · 29 500 938 720 B |
| of which vision | 333 · 921 460 192 B |
| **parameter tensors after a text-only load** | **1 847 · 28 579 478 528 B** |
| MLX active memory after `mx.eval(parameters)` | 28 579 478 536 B |
| resident vision tensors | **0** |
| bytes that never materialised | 921 460 192 |

So under the factory that is actually executed, the MLX side carries **no**
vision residency, and neither does the llama.cpp side without `--mmproj`.

That the filter is load-bearing rather than incidental is checked, not assumed:
the same script runs a mutant of `Model.sanitize` with its vision branch deleted
over the identical key set, and the mutant keeps all 333 vision keys. If the
production filter stopped working, the check fails.

Consequences for the benchmark:

* **Do not add or subtract 921 MB from any memory comparison.** Neither runtime
  pays it on the default text path. Revision 1's advice to "note that the MLX
  side's vision tower is resident-but-unused" was wrong and is withdrawn.
* The difference that *does* survive is a **file-size** one: 29.50 GB of MLX
  checkpoint on disk against 29.05 GB of GGUF text model plus an optional
  0.93 GB mmproj.
* If a later run enables vision on either side, that is a different
  configuration and both sides must enable it.
* Compare **measured footprint under the executed factory**, not file membership.
  `mx.get_active_memory()` is the stable MLX-side number (28 579 478 536 B across
  runs). Peak process RSS is *not* — it read 22.3 GB and 18.6 GB on two runs of
  the same load, because the weights are mmap-backed and paged lazily. Use an
  allocator-level figure, or `footprint`, and say which.

Incidentally the mmproj is BF16 despite its `F16` filename
(`general.file_type = 32`), which matches MLX's bf16 vision tower — vision weight
*precision* is equivalent even though placement is not.

### 5.3 Which tensors get quantized at all

The quantized sets line up exactly:

* MLX quantizes **498** tensors: all MLP, GDN and full-attention projections,
  plus `embed_tokens` and `lm_head`.
* GGUF quantizes **506** = the same 498 + the 8 tensors of the MTP block.

Everything else — layernorms, GDN `conv1d`, `A_log`, `dt_bias` — is unquantized
in both. Precision differs: MLX keeps them **bf16** (as in the source), GGUF
upcasts to **F32**. That is a lossless widening, so no fidelity difference, only
10 686 464 bytes of extra resident memory on the GGUF side.

### 5.4 Weight budget

| | bytes |
|---|---:|
| MLX total tensor bytes | 29 500 938 720 |
|  · language model | 28 579 478 528 — **and this is what a text-only load makes resident** |
|  · vision tower (bf16) | 921 460 192 — on disk only; dropped by `sanitize` before load (§5.2) |
|  · MTP head | 0 — dropped at conversion |
| GGUF text model tensor bytes | 29 036 089 344 |
|  · language model excl. MTP | 28 584 769 536 |
|  · MTP block `blk.64` | 451 319 808 — on disk only; skipped at load (§5.1) |
|  · F32 norms and 1-D tensors | 10 686 464 |
| GGUF mmproj (separate file) | 931 145 984 — not loaded without `--mmproj` |

Every row marked *on disk only* is a file-size fact. Only the two
language-model rows belong in a memory comparison.

**Language-model weights, MTP excluded on both sides: 28 579 478 528 vs
28 584 769 536 — a 0.02 % difference.** That is the number that says these are
the same model at the same bit cost.

### 5.5 Smaller mismatches worth stating

* **Tokenizer.** GGUF carries its own GPT-2-style vocab (`tokenizer.ggml.pre = qwen35`,
  248 320 tokens, 247 587 merges) rather than reusing `tokenizer.json`. Token
  counts per prompt should be checked to match, not assumed.
* **Sampling defaults.** Both carry temp 1.0 / top_k 20 / top_p 0.95, but
  llama.cpp and mlx_lm apply repetition/penalty defaults differently. Pin
  sampling explicitly on both sides.
* **Layout.** The GDN value-head permutation of §4 means tensor-by-tensor
  tooling cannot assume matching indices.

## 6. Verdict for the benchmark gate

**Comparable, with three conditions.**

The weights pass: same architecture, same BF16 lineage, same 498-tensor
quantized set, identical 8.5 bits/weight, 0.02 % weight-byte difference, and
every sampled tensor within 8-bit rounding noise of the same source. If anything
the GGUF side is marginally *more* faithful to BF16 (ratio 0.766), which biases
quality comparisons very slightly in llama.cpp's favour — far below the noise of
any throughput measurement, but it should be stated rather than ignored.

Conditions for a valid comparison:

1. **MTP speculative decoding OFF** on the llama.cpp side, or reported as a
   separate configuration. The MLX baseline has no MTP head to match it.
2. **Compare measured footprint, not file membership.** Neither side makes the
   vision tower resident on the default text path — the MLX loader drops it
   before `mx.eval`, and llama.cpp never sees it without `--mmproj` (§5.2,
   measured). So do **not** adjust either side by 921/931 MB. Report an
   allocator-level or `footprint`-level number and name it; peak RSS is unstable
   across runs of the identical load because the weights are mmap-backed. If
   vision is wanted, enable it on both sides and say so.
3. **Pin sampling and verify tokenization** on both sides.

If a later run enables MTP and compares the resulting tokens/s to the MLX
baseline as though it were a runtime difference, that comparison should be
refused. That is the one way this pair genuinely becomes non-comparable.

---

## 7. Load-and-answer check — passed, and it can fail

`load_and_answer.sh`, exit 0, run with **the exact artifact attached here** at its
untouched defaults (`load-and-answer-03.log`, 16:36:07–16:36:31):

```
== host contention check ==
free+inactive+speculative: 42.9 GiB (floor 35)
== starting llama-server ==
== waiting for readiness (max 600s) ==
ready after ~25s
== one bounded prompt ==
curl rc=0
finish_reason: stop
content: 'The capital of Armenia is Yerevan.'
usage: {'completion_tokens': 10, 'prompt_tokens': 25, 'total_tokens': 35, ...}
OK: bounded answer asserted
== PASS: load-and-answer attested ==
```

Command: `llama-server -m …-Q8_0.gguf --host 127.0.0.1 --port 18901 -c 4096 --jinja`,
one chat completion at `temperature 0`, `max_tokens 96`,
`chat_template_kwargs.enable_thinking = false`. MTP/speculative decoding was
**not** enabled. Readiness in ~25 s here is a warm-page-cache number; the first
cold load of this file took ~45 s. Neither is a benchmark number.

`load-and-answer-01.log` is the equivalent run from earlier in this rework, before
two additive contention knobs were added; `load-and-answer-asrun.diff` shows that
delta is confined to the pre-launch guard. It is kept only as history — the log
above is the attestation for the artifact as delivered.

The guard was exercised for real in between: an attempt at 16:35:48
**refused with exit 3** (`load-and-answer-refused-03.log`) because
TASK-260827-2v13w8-rev4's smoke runtime had held port 18799 for over 33 minutes.
The check waited about 25 minutes for that neighbour to release the host. The
neighbouring process was not signalled, not killed, and the guard was not
narrowed to get past it — §8 says explicitly that if this script exits 3 you wait.

Host contention was honoured, not worked around. The check refuses to start while
anything listens on 18000–18999 or an `mlx_lm` / prototype / benchmark process is
alive, and requires ≥35 GiB reclaimable. At revision 1 it was blocked from 14:52
to 15:08 — about 16 minutes — until TASK-260827-2v13w8 released the machine, and
again for about 25 minutes at revision 2, as above. No other run's process was
signalled on either occasion.

Revision 3 did not rerun this check, and says so rather than implying otherwise.
`load_and_answer.sh` and `test_load_answer_gate.sh` are byte-identical to the
artifacts whose green runs are quoted above and in §7.1; revision 3 touched only
`quant_equivalence.py` and `test_alignment_guard.py`. One rerun was attempted at
17:05 and **refused with exit 3** (`load-and-answer-04.log`) — TASK-260827-2v13w8-rev4's
benchmark held port 18031 with `mlx_lm-relux.server` and the mlx-swift prototype both
live. That neighbour was not signalled and the scans were not narrowed.

### 7.1 The attestation now fails closed

At revision 1 this script printed `curl rc`, `finish_reason` and the content and
asserted **none of them**. Review proved it by swapping in a healthy fake that
answered *"This is deliberately the wrong answer."* — the script exited 0. That
was reproduced again at revision 2 against the rev-1 artifact, with only the
binary made injectable: a fake answering *"The capital of Armenia is Baku."*
still gave `exit 0` (`mutant-rev1-binseam.log`).

The script now exits non-zero on a curl failure (7), a body that is not the
expected JSON shape (8), zero reported completion tokens (8), a `finish_reason`
other than `stop` (9), an empty completion (10), a completion that does not
contain the expected answer (10), and an early server exit (5). The prompt and
the required answer are hard-coded; no environment variable can make the script
accept a wrong answer.

`test_load_answer_gate.sh` drives **the real script** — the exact artifact
attached to this task — swapping only `LLAMA_SERVER_BIN`, exactly as the reviewer
did. Every case asserts the **exact** exit code, not merely "non-zero": a case
that trips the host-contention guard (3) instead of the answer gate is a failure,
not a pass. Exit 0, 11/11, stable over five consecutive runs
(`load-gate-tests-01.log`):

| case | expected exit |
|---|---|
| confident wrong answer (*"…is Baku."*) | 10 |
| `finish_reason: length` with a truncated answer | 9 |
| empty completion | 10 |
| body with no `choices[]` | 8 |
| body that is not JSON at all | 8 |
| correct answer but `completion_tokens: 0` | 8 |
| server that exits immediately | 5 |
| **correct answer** — the gate is not "always reject" | **0** |
| a listener planted on 18950, **default** port scan | 3, naming `:18950` |
| a process planted as `runtime-benchmark.py`, **default** process scan | 3, naming it |
| both plants outside both scans — control for the two above | 0 |

The last three matter because the host is shared. The real guard refuses whenever
anything listens in 18000–18999 or a local model runtime is alive, which is right
for a 29 GB load and fatal for fakes that load nothing — a neighbouring run
holding port 18799 turned six answer-gate cases into exit-3 "passes" during
development. The answer-gate cases therefore narrow `CONTENTION_SCAN` /
`CONTENTION_PROCS` to sentinels the suite plants itself, and the three cases above
run at the untouched defaults. The narrowing cannot hide a broken answer gate:
that gate has no environment override at all.

And the gate is bounded, not merely present:

| mutant of `load_and_answer.sh` | suite result |
|---|---|
| answer assertion deleted, seams kept (`mutant-load-noassert.log`) | **exit 1** — 6 cases fail |
| assertion kept but the expected answer widened from `\byerevan\b` to `\bcapital\b` (`mutant-load-weak.log`) | **exit 1** — the wrong-answer case fails, and only that one |

The narrowing mutant is the informative one: widening what counts as a correct
answer just enough to admit *"The capital of Armenia is Baku."* breaks exactly
one case. The gate bounds the answer itself, not the presence of some string.

### 7.2 What the load log adds

All 15 `blk.64.*` tensors are reported unused and skipped on the default path
(see §5.1). The vision tower is absent as expected — `--mmproj` was not passed.
The embedded chat template works under `--jinja`, and llama.cpp notes the
template supports preserving reasoning (`--reasoning-preserve`), which is
another knob to align with the MLX side before comparing output.

---

## 8. Reproducing every number here

The system `python3` on this host is Homebrew 3.14.7 **without NumPy**, so the
analysis scripts fail on their shebang alone. Review hit this. Use the pinned
interpreter — the existing `mlx-lm` pipx venv, which is also the environment the
MLX baseline itself runs under:

```bash
PY=/Users/alexis/.local/pipx/venvs/mlx-lm/bin/python
# CPython 3.14.7 · numpy 2.5.2 · mlx_lm 0.31.3
cd <directory holding these scripts>          # they import each other by path

"$PY" quant_equivalence.py          > quant-equivalence-04.log    # exit 0, mean ratio 0.766
"$PY" test_alignment_guard.py       > guard-tests-04.log          # exit 0, 16 checks, 0 failures
"$PY" vision_residency.py --measure > vision-residency-02.log     # exit 0, loads ~28.6 GB
./load_and_answer.sh                > load-and-answer-03.log      # exit 0, needs llama.cpp + 35 GiB
./test_load_answer_gate.sh          > load-gate-tests-01.log      # exit 0, 11/11, no model needed
```

Notes:

* `quant_equivalence.py` takes an optional row count: `"$PY" quant_equivalence.py 512`
  reproduces revision 1's table. The default is 1024 and every log states which
  was used.
* `test_alignment_guard.py` imports `quant_equivalence.py` from its own
  directory, so run it from there. To reproduce the two production mutants,
  copy both files into a scratch directory, apply the mutation to the *copy* of
  `quant_equivalence.py` only, and rerun the unchanged suite there — it must
  exit 1 in both cases (`mutant-rev3-ratio.log`, `mutant-rev3-bypass.log`).
* `test_load_answer_gate.sh` needs neither llama.cpp nor the weights — its fakes
  answer instantly. It is the cheap check; run it first. It is also safe to run
  beside another tracked run: it plants and scans only its own sentinels.
* `load_and_answer.sh` is the opposite — it refuses while any other local runtime
  holds the host, and it is meant to. If it exits 3, wait; do not narrow its
  scans to get past a neighbour.
* `vision_residency.py` without `--measure` does parts A and B only and costs
  nothing. Only `--measure` loads the model, and it refuses below 35 GiB
  reclaimable.
* Only NumPy is required by `quant_equivalence.py`; `vision_residency.py`
  additionally needs `mlx_lm` and `mlx`. Any Python ≥ 3.10 with NumPy ≥ 2.0
  works — the venv above is simply the one that already has both on this host.
