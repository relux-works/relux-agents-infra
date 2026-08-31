# TASK-260829-3k4qrc revision 6 review verdict

Verdict: **accepted**. `CR-TASK-260829-3k4qrc-6`, base `3272e3ae`, candidate tree
`98df2a3f`. Attached patch reproduces SHA-256
`7c7f9bc2f0e5f1881067df29cd3410c49cdf0ee02f2af681af3d84c4e9732278`, matching the
CR declaration. Both revision-5 blocking findings are fixed, the fixes are correct
against the run's own raw data, and no measurement code moved. I modified no
repository file; everything I produced lives under gitignored `.temp/`.

## Scope of the delta, verified rather than accepted

| check | result |
| --- | --- |
| `git diff --stat 75b41cd2 98df2a3f` | 3 files, **52 insertions / 17 deletions** |
| non-`.md` paths in that delta | **none** (`grep -vE '\.md$'` exits 1) |
| `git diff --stat bafa676b 98df2a3f -- Sources Tests scripts examples` | **empty — byte-identical** to the tree rev4's validation was gathered on |
| worktree `git diff 98df2a3f` | **empty** — what I reviewed is what will be integrated |

Validation re-run on this tree rather than inherited: `swift build -c release` exit 0,
`swift test -c release` **410 tests / 32 suites** passed, `swift-format lint --strict`
empty, `scripts/benchmark-gate-smoke.sh` **120 PASS / 0 FAIL** (counted from the log,
not from the report's claim). The CR's own validation log is the repo-level Go suite,
exit 0 on `go test ./...` and `go vet ./...`.

---

## The four brief items — confirmed, item 1 by a stronger method than last round

### 1. Is the `contextPolicy` refusal irreducible? **Yes — and now proved from the build, not from a probe list.**

I did not re-probe endpoints one at a time. Probing settles "the seven URLs I tried
did not carry it"; it does not settle "no URL carries it". I enumerated the whole
HTTP surface out of the pinned build's own string table
(`/opt/homebrew/Cellar/llama.cpp/0.3.0/lib/libllama-server-impl.dylib`, the dylib
behind `llama-server` 0.3.0 build 10621 commit `c1d0e7a00` — the exact build in the
record) and cross-checked it against llama.cpp server source.

Complete route set the build registers — **including seven the earlier probe never
touched**: `/props`, `/slots`, `/metrics`, `/health`, `/v1/health`, `/models`,
`/v1/models`, `/models/load`, `/models/sse`, `/models/unload`, `/completion`,
`/completions`, `/v1/completions`, `/chat/completions`, `/v1/chat/completions`,
**`/v1/chat/completions/control`**, `/chat/completions/input_tokens`,
`/v1/chat/completions/input_tokens`, **`/responses`**, `/v1/responses`,
`/responses/input_tokens`, `/v1/responses/input_tokens`, **`/v1/messages`**,
`/v1/messages/count_tokens`, **`/tools`**, **`/v1/stream`**, **`/v1/streams/lookup`**,
`/infill`, `/embedding`, `/embeddings`, `/v1/embeddings`, `/rerank`, `/reranking`,
`/v1/rerank`, `/v1/reranking`, `/tokenize`, `/detokenize`, `/apply-template`,
`/lora-adapters`, `/audio/transcriptions`, `/v1/audio/transcriptions`,
`/cors-proxy`, `/predict`, `/index.html`.

None of them can carry either term, because the values are not serialized anywhere:

- **Prefill chunk.** `n_ubatch` and `n_batch` **never appear as a JSON key in any
  server handler.** Every occurrence in `tools/server/*.cpp` is either an internal
  scheduling variable (`server-context.cpp:3011,3091,3466,3491`) or an `SRV_WRN`
  log format string (`server.cpp:145-148`). The dylib's string table agrees: the
  only `n_ubatch`/`n_batch` strings in the shipped binary are those same log
  formats. There is no endpoint to miss.
- **Reasoning effort.** `reasoning_effort` exists only as an **inbound request
  field**, parsed in `server-common.cpp:1295-1302` into `chat_template_kwargs`, and
  re-emitted into an **outbound** `chatcmpl_body` at `server-chat.cpp:290`. It is
  never part of any server-state response. The `--reasoning-effort` launch flag
  populates a default that no route reports.
- I read the `/props` payload construction line by line
  (`server-context.cpp:4576-4610`): `default_generation_settings` is
  `{params: tparams.to_json(true), n_ctx}` where `tparams.sampling = params.sampling`
  — sampler fields only — plus `total_slots`, `model_alias`, `model_ftype`,
  `model_path`, `modalities`, `media_marker`, endpoint toggles, `ui`,
  `chat_template`, `chat_template_caps`, `bos_token`, `eos_token`, `build_info`,
  `is_sleeping`, `cors_proxy_enabled`. No batch. No effective reasoning effort.
  `chat_template_caps.supports_reasoning_effort` is a template capability, a
  constant, and a gate wired to it would carry a constant — the false friend the
  rev5 review flagged, confirmed here at its construction site.

**The producer's conclusion holds and the refusal is correct.** Reading the two
terms from argv would reopen the exact defect the live derivation closed
(`--prefill-step-size 2048 --prefill-step-siz 999` runs at 999 while argv reads
2048); the gate must not do it, and the report does not ask it to.

### 2. Are the readings trustworthy despite the refusal? **Yes — recomputed from `session-rev4/records/*.json`, not from any table.**

- Sealed intervals: baseline `1788113670.052570 → 1788115795.126375` = **2125.074 s**;
  candidate `1788115816.214009 → 1788117186.393213` = **1370.179 s**; separation
  **+21.087634 s**, overlap **−21.088 s**. No overlap.
- Prompt tokens equal on **all six** scenarios: 41 / 7,784 / 313 / 7,784 / 910 /
  **73,016**, ratio 1.0000 each.
- `speculation: "off"` in both pins. `seed 1234`, `temperature 0`, `topP 1`,
  `maxOutputTokens 256`, `promptSuiteDigest bba8867f…` identical on both sides;
  `modelOfRecord source:hf:orcarouter/Qwen3.8-27B-Uncensored-BF16` on both.
- `contextPolicy` literally `kv=76800;prefill-step=2048;reasoning=medium` vs
  `kv=76800;prefill-step=not-reported;reasoning=not-reported`. **The KV term agrees
  on both sides and is derived from the live listing — that blocker is gone.**
- Decode 6.7809/8.1450 and 6.6705/7.7235; TTFT 2.5314/2.1429, 107.2163/67.4273,
  1279.8919/950.8526; prefill 16.1969/19.1328, 72.6009/115.4428, 57.0486/76.7900 —
  all reproduce to the printed digit.
- Nothing in the refused state corrupted anything: admission runs after the passes,
  both records and both attestations exist, and no `decision.json` was written.

### 3. The memory cause. **Confirmed, from the run's own raw stamps.**

Recomputed by inverting the mapped observation timestamps myself:

| window | mapped gap min / med / max | over 7.0 s | Mach max gap | over 125 ms |
| --- | ---: | ---: | ---: | ---: |
| baseline `soakMemory` | 13.862 / 14.682 / 15.117 s | **19 / 19** | 15.117 s | **19 / 19** |
| candidate `soakMemory` | 10.141 / 10.916 / 13.129 s | **19 / 19** | 13.129 s | **19 / 19** |

`memorySamplesReadFailed: 0` and `memorySamplesMalformed: 0` on **both** passes.
The `readFailureCount: 1` is the synthetic coverage marker, and the records prove it:
`1` on every `partial` window, `0` on the single `measured` one. The arithmetic behind
the 7.0 s bound versus a 2.2–2.6 s median / 5.8 s max `vmmap` cost against 26–45 GiB
targets is correct.

### 4. `context_75k` decode. **Confirmed unreliable — and now correctly withdrawn.**

`completionTokens` is **16 on both runtimes**, verified in both records. 15 tokens over
a ~1.8 s tail after a 950–1280 s prefill is not a decode measurement.

---

## Both revision-5 findings — fixed, and the fixes verified against raw data

**Finding 1 (refusal accounting).** All four ungated windows reproduce exactly:

| window | status | issues | `scoredBytes` | stamps |
| --- | --- | --- | ---: | ---: |
| baseline `warmupMemory` | measured | `[]` | 29,120,518,072 | 1 |
| baseline `soakMemory` | measured | `[]` | 29,827,094,504 | 20 |
| candidate `warmupMemory` | measured | `[]` | 34,248,152,988 | 1 |
| candidate `soakMemory` | measured | `[]` | 44,346,176,540 | 20 |

The bypass is real and the docs now describe it correctly. I traced it in production
rather than trusting the description: `coveredPeak` is reached only from
`currentWindowPeak()` (`:297`), `processPeakSoFar()` (`:303`) and `capturePeaks()`
(`:322-323`); `BenchmarkPass.swift:100` and `:107` construct these four directly.
**The "no scored comparison consumes them" bound holds** — `grep -rn
"soakMemory\|warmupMemory" Sources/MLXSwiftRuntimeContract/` returns nothing; the
names appear only in the harness executable and the session serializer
(`BenchmarkRunPins.swift:463-464`). All three repo docs and the report now scope the
claim to "every scenario window and both process-wide peaks", list the four windows
with their real status, and carry the provenance.

**Finding 2 (invented physical cause).** The sentence is gone and replaced by what the
counter is. No physical conclusion is drawn from it. Correct.

**Also fixed.** `context_75k` decode is struck in the report table, removed from the
"overturned" row and the summary, and marked withdrawn in `LOGBOOK.md`, the research
banner and the README. The struck values are retained only so the withdrawal is
auditable — the right call.

**The +6.2 GB soak climb now travels with its provenance everywhere.** Verified from
the raw series: the entire delta is the Mach anonymous component
(9,905,647,432 → 16,106,766,568 = exactly +6,201,119,136 B) while the mapped
component is constant at 28,239,409,972 B on all 20 samples. Baseline fell by
509,935,592 B. "Not a leak, not a memory regression, not a gated number" is what the
data supports.

## Non-blocking nits, recorded so they can be corrected if quoted

1. **Off-by-one in the refusal enumeration.** `§4.1` of the report and the rev6
   correction note say `readFailureCount == 1` on "**all 26** partial windows … all 6
   candidate scenarios". The pair has **26 gated windows: 25 `partial` + 1 `measured`**
   (candidate `short_prompt` scenario), so it is 5 candidate scenarios, not 6 — as the
   same document's own §4 table states two sections earlier. The substantive claim is
   correct and verified; only the count is wrong, and only in board artifacts, not in
   any repo document or anything article-facing.
2. The correction note states `51 insertions`; the actual `git diff --stat` is **52**.
3. `TASK-260829-3k4qrc_measured-pair-outcome.md:437` forward-references a "§7.1" that
   the document does not contain. The validation it points at is real and attached.
4. There is a **third** ungated `RuntimeMemoryPeak(summarizing:)` at
   `BenchmarkFootprintSampler.swift:364`, inside `sampleCounts()`, which the docs do
   not name. It is correct by design and it *strengthens* Finding 2's fix: because it
   is ungated, the session-level `memorySamplesReadFailed: 0` never sees the synthetic
   marker, which is exactly what makes the correction provable. Worth naming when the
   two `BenchmarkPass` sites are routed through `coveredPeak`, so it is not "fixed" too.

None of these changes a conclusion, a number, or what the article may claim. Blocking a
seventh revision on them would be ceremony, not review.

---

## The judgement you asked for: **(b) — proceed to the article. Do not buy the second pair.**

Not on cost. On the measured fact that **(a) as scoped cannot produce the number the
decision needs.** Verified from the records myself:

| window | baseline refusal | candidate refusal |
| --- | --- | --- |
| **`context_75k`** — the window the decision rests on | `mach-physical-footprint-sampling-gap` | `mach-physical-footprint-sampling-gap` |
| whole-pass process peak | `mach-physical-footprint-sampling-gap` | `mach-physical-footprint-sampling-gap` |
| `stability_soak` | `resident-mapped-file-sampling-gap` | `mach-physical-footprint-sampling-gap` |
| `short_prompt` / `long_prompt_8k` / `tool_call` / `multiturn` | `resident-mapped-file-sampling-gap` | mapped, except `short_prompt` (measured) |

**`context_75k` refuses on the *Mach* bound on both sides.** A self-calibrating mapped
bound does not touch it. The 125 ms Mach bound broke because host load hit **13.995**,
and that load is generated by the benchmark itself — a 73k-token prefill pinning an
M1 Max for 21 minutes. You cannot calibrate that away without either making the bound
load-aware (weakening the instrument precisely where it is protecting you) or running
the 75k probe on a machine that is not running the 75k probe. So (a) buys memory on the
short scenarios — where memory is least interesting — and leaves the 75k comparison
refused exactly as it is today. Two hours plus two agent cycles for the windows nobody
is deciding on.

**What would make me confident a second pair scores** — and I can now put a number on
why the current design is wrong, independent of which way the results went:

The scored quantity is Mach footprint + the `vmmap` mapped upper bound. Across the
whole pass the mapped component takes **11 distinct values spanning 2.26–3.64 MB on the
baseline** — about 0.01 % of a 29 GB footprint — and **3 distinct values on the
candidate** (`0` pre-load, then 28,239,409,972 / 28,668,906,701 — effectively static
once the GGUF is mapped). So a 2.2–5.8 s external `vmmap` fork, at a 0.2 Hz cadence,
policed by a 7.0 s gap bound, exists to track a quantity that is a rounding error on
one runtime and a constant on the other — and it is that fork that refuses 268/288 and
179/200 of the mapped observations.

That is a design defect, not a calibration error, and the fix is the one the previous
round named: **stop forking `/usr/bin/vmmap`.** Read the mapped component in-process
(`proc_pidinfo` / a `mach_vm_region` walk) and both components come from one cheap read
at one cadence; the 7.0 s bound disappears rather than being retuned, and the Mach
bound stops being a separate killer for `context_75k`. That is a real instrumentation
task with a design, and it is the only version of (a) I would back. **Re-deriving
`observedMappedFileReadCostSeconds` upward against the 28 GB targets is not that** —
doing it after seeing your own run refused, on data whose direction you already know,
is the confirmation-bias move the brief names, and the producer was right to refuse it.

### What the article must not claim

1. **No memory comparison in any direction** — not "llama.cpp uses less", not
   "comparable", and **not the revision-1 −5.64 % figure**, which came from the
   instrument this story has since shown was defective.
2. **Not the §4.3 unscored components** (48,433,633,280 B vs 44,401,947,700 B). Single
   instants from windows that failed coverage; the report labels them "not evidence"
   and they must not be laundered into a chart.
3. **Not the +6.2 GB soak climb as a leak or a memory regression.** Two-point delta off
   a window that never faced the gate, entirely in the Mach anonymous component. State
   it with that provenance as an open observation, or leave it out.
4. **Nothing in §3 as a scored comparison.** The gate refused the pair, exit 4, no
   `decision.json`. They are each runtime's own readings under identical prompts,
   identical prompt-token counts, a pinned model, seed 1234, temperature 0 and MTP off
   — say exactly that, and say the gate refused, in the same breath as the numbers.
5. **Not `context_75k` decode.** Cite that scenario for capacity, TTFT and prefill only.
6. **Not `multiturn_prefix_reuse` or `stability_soak` timings as speed results.** Sealed
   `cached_tokens` show llama.cpp hit (`[5736, 7780, 7809]`, `[18]×20`) against a
   baseline miss (`[0]`) — one-sided cache, verified in the records.
7. **It must state, in the decision section and not a footnote, that an equally-weighted
   axis is absent and why:** the mapped bound is calibrated a factor of ~2.5 below what
   a `vmmap -summary` costs against a 26–45 GiB target, and the 125 ms Mach bound cannot
   hold under the host load the 73k prefill itself generates. The gate failed safe. It
   did not produce a wrong number. But the decision is being made on the remaining
   criteria and that has to be visible in the text.

The decode result is the article's headline and it is solid: one shared generated-event
definition, exact prompt-token parity, MTP and speculation off on both sides, sequential
passes with no overlap on a host holding no other model — llama.cpp decoded faster on
both scenarios where decode is meaningfully measured, **8.145 vs 6.781 tok/s (+20.1 %)**
and **7.724 vs 6.671 tok/s (+15.8 %)**, with `long_prompt_8k` TTFT at **0.629x** and
prefill at **1.590x**, both reproducing revision 1. The withdrawn "about 10 % slower"
claim is overturned. Report it as two runtimes' own readings that the gate would not
certify as a comparison, and the article is honest and still has its result.

---

## Definition of Done

| criterion | status |
| --- | --- |
| Both runtimes sequential, host holding no other model, stated | **met** — intervals verified, overlap −21.088 s, `ps` sweeps clean |
| MTP off for every scored comparison | **met** — `speculation: "off"` in both records, read from `/slots` |
| Non-comparable dimensions refused, every refusal reported | **met** — the four ungated windows are now accounted for in all three repo docs and the report |
| Corrected decode/TTFT reproduce or overturn the provisional deficit | **met** — overturned, direction and magnitudes verified from raw records |
| Corrected memory reproduces or overturns the 9 % advantage | **met as a refusal** — the axis produced no comparison, and that is the honest answer |
| Tests / lint / build green | **met** — 410 tests, 120/0 smoke, lint clean, build 0, re-run on this tree |
| Findings recorded in logbook | **met** — and now accurate |
| Gate behaviour attacked, not read | **done by me** — whole-surface route enumeration from the pinned dylib plus source-level proof that neither term is serializable anywhere; coverage arithmetic recomputed from raw stamps; the `coveredPeak` bypass traced to all three ungated construction sites and its "not consumed" bound checked by grep |
