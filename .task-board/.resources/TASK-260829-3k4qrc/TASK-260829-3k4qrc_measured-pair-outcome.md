# TASK-260829-3k4qrc — revision 5: the corrected llama.cpp comparison, run

> **Revision 5 is a reporting correction only.** No measurement code changed, no rerun was
> performed, and every number below still comes from the same `session-rev4` artifacts. The
> revision-5 review confirmed the measurements and rejected two reported facts: the refusal
> accounting was incomplete (§4 — four windows carry a score without ever facing the
> coverage gate) and §4.1 attributed a physical cause to an instrument artifact. Both are
> corrected in place below, and the `context_75k` decode figure is withdrawn as a decode
> result (§3.2).

Date: 2026-08-30
Role: developer
Session: `.temp/TASK-260829-3k4qrc/rev4/session-rev4`
Driver exit: **4 — inadmissible. No `decision.json` was written.**

## The result in six sentences

The full pinned six-scenario suite ran end to end against `mlx_lm-kv76800-45a472f.server`
and `llama-server`, sequentially, on this host, in one `benchmark-run` invocation, with
MTP observed off on both sides. **All twelve scenario runs succeeded and prompt-token
parity was exact — ratio 1.0000 — on all six scenarios, including the capacity probe at
73,016 tokens.** The gate refused to score the pair, **exit 4**, on `contextPolicy`
again — but for a different reason than every previous run: the KV term now *agrees*
(`kv=76800` on both), and the refusal moved to the other two terms of the same pin,
because `llama-server` b10621 reports no effective prefill chunk and no reasoning effort
on any live endpoint (§2). **Memory is refused on every scenario window and both process-wide
peaks of both runtimes, including `context_75k`** — the mapped-file coverage bound is
7.0 s and one `vmmap -summary` against these 26–45 GiB targets costs a median 2.2–2.6 s
and up to 5.8 s, so 268 of 288 baseline and 179 of 200 candidate mapped observations fall
outside it (§4). **Four further windows publish a score without ever facing that gate and
are excluded from every claim here** (§4). **On decode the corrected instrumentation overturns the withdrawn "llama.cpp
about 10 percent slower" claim: llama.cpp decodes faster on both scenarios where decode is
meaningfully measured** — +20.1 % and +15.8 % tokens per second — and TTFT and prefill
reproduce revision 1 closely (§3). The `context_75k` decode figure is withdrawn as a
decode result: its budget is 16 completion tokens (§3.2). Two scenarios are separately non-comparable on sealed
cached-token telemetry before `contextPolicy` is even reached, because llama.cpp's
per-slot KV reuse fired and the incumbent's configured prompt cache did not (§5).

## 0. Units, direction, and what is not a result

- **Decode and prefill are tokens per second. Higher is better.**
- **TTFT and wall clock are seconds. Lower is better.**
- **Memory is bytes**, scored as `residentMemoryUpperBoundBytes` = Mach physical
  footprint + the upper edge of the `vmmap -summary` resident `mapped file` bucket.
  Lower is better. `peakPhysicalFootprintBytes` is retained in the records and is
  **not** the scored quantity; on an `mmap`-loading runtime it cannot see the weights.
- Every ratio below is **candidate ÷ baseline** = llama.cpp ÷ Python.
- **No number in §3 is a scored comparison.** The gate refused the pair, so these are
  the two records' own readings printed side by side. They are reported because the
  brief requires the corrected measurements to be produced, and because refusing to
  print them would hide a result rather than qualify it.
- **No `--spec-type draft-mtp` number was taken.** MTP is off in both records, read by
  the gate from each running process, not from the launch flag.

## 1. What ran, and on what

| | |
| --- | --- |
| host | MacBookPro18,2, Apple M1 Max, 68,719,476,736 B RAM, `hostIdentity` `MacBookPro18,2/68719476736/25F80/arm64` |
| baseline | `mlx_lm-kv76800-45a472f.server`, `mlx_lm` 0.32.0 @ `45a472f2d0cda166b7ffe1a80fe50dd9621f4303`, mlx 0.32.2 |
| candidate | `llama-server`, Homebrew llama.cpp 0.3.0, build `b10621-c1d0e7a00` |
| config | `.temp/TASK-260829-3k4qrc/rev4/TASK-260829-3k4qrc-rev4.benchmark.toml` |
| driver | one `benchmark-run`, 3,535 s wall (58 m 55 s), exit 4 |
| equivalence | `equivalence/qwen3-8-27b-uncensored.equivalence.json`, `modelOfRecord` `source:hf:orcarouter/Qwen3.8-27B-Uncensored-BF16` on both records |

### 1.1 Sealed, non-overlapping intervals

| pass | runtime | start (unix) | finish (unix) | duration |
| --- | --- | ---: | ---: | ---: |
| baseline | `python-mlx-lm` | 1788113670.052570 | 1788115795.126375 | 2125.074 s |
| candidate | `llamacpp` | 1788115816.214009 | 1788117186.393213 | 1370.179 s |

Separation between the passes: **21.088 s**. Overlap: **−21.088 s**, i.e. none. A
positive overlap is an admission refusal by name.

### 1.2 Host holds one model at a time — checked, not assumed

`ps` sweeps for `llama-server`, `mlx_lm`, `mlx-swift` and `model-harness run` were taken
before and after the invocation and both printed `(none)`
(`.temp/TASK-260829-3k4qrc/rev4/run-rev4-sweeps.log`). Full raw process lists were
captured on both sides (`raw-processes-before.txt`, `raw-processes-after.txt`). The two
passes are sequential by construction with a 20 s settle between them.

Peak host load during the run: baseline `hostLoadAverageMax` 13.995 inside `context_75k`,
candidate 8.881 inside `long_prompt_8k`. The workstation was not otherwise idle-locked
and nothing was killed to make room.

**Blocker B7 did not recur.** The runner terminates only its own process group and never
matches by process name. After the run, `pgrep -f` against this run's own config path
found nothing; a raw process-list snapshot was pre-staged for the case where it had
(`run-rev4-interval.txt`: `no orphan names this run's config`).

## 2. REFUSAL 1 — `contextPolicy`, exit 4, and the term that broke is not the one that was fixed

```
pinned condition "contextPolicy" differs:
  baseline  "kv=76800;prefill-step=2048;reasoning=medium"
  candidate "kv=76800;prefill-step=not-reported;reasoning=not-reported"
; these runs are not a comparison
```

Driver exit **4**. No `decision.json`. Both records and both attestations were written,
because admission runs after the passes — which is why this report has measurements.

### 2.1 The KV term is fixed and stayed fixed

Every previous run in this story refused on `kv=unbounded` against `kv=<n>`.
STORY-260830-2vrhg1 delivered `--max-kv-size` in the pinned `mlx-lm` fork, and this run
confirms it end to end: **both records pin `kv=76800`, derived by the gate from each
running process's live `/v1/models` `meta.n_ctx`, not from argv.** That blocker is gone.

### 2.2 The term that broke, and why it is not a configuration mistake

`contextPolicy` has three terms. The gate derives all three from the live model listing
and never decodes runtime argv — a deliberate change, because argv is not what the
process parsed (`--prefill-step-size 2048 --prefill-step-siz 999` runs at 999 while argv
reads 2048). `llama-server` reports the first term and neither of the other two.

Measured directly against the same build and the same GGUF, launched with
`--ubatch-size 2048 --reasoning-effort medium`
(`probe-llamacpp-v1-models.json`, `probe-llamacpp-props.json`, `probe-llamacpp-slots.json`):

| endpoint | what it carries | prefill chunk? | reasoning effort? |
| --- | --- | --- | --- |
| `/v1/models` `data[].meta` | `vocab_type`, `n_vocab`, `n_ctx`, `n_ctx_train`, `n_embd`, `n_params`, `size`, `ftype` | **no** | **no** |
| `/props` | `build_info`, `model_path`, `total_slots`, `chat_template`, `modalities`, `default_generation_settings` | **no** | **no** |
| `/props` `default_generation_settings` | `n_ctx`, `params` | **no** | **no** |
| `/props` `default_generation_settings.params` | sampler fields, `reasoning_format`, `reasoning_in_content`, `speculative.types` | **no** | **no** — see below |
| `/slots[]` | `id`, `is_processing`, `n_ctx`, `speculative` | **no** | **no** |

There is no `n_batch` or `n_ubatch` field anywhere on this runtime's HTTP surface.
`reasoning_format` is not the reasoning-effort policy that renders the prompt — it is the
response-format setting, and it **read `"none"` while the process had been launched with
`--reasoning-effort medium`**. That is precisely the `/props`-shaped trap this story
already documented for speculation, where `/props` kept reporting `"none"` under
`--spec-type ngram-mod` while `/slots` flipped to `true`. A reader wired to
`reasoning_format` would carry a value that does not track the launch.

**So the refusal is correct and irreducible on this build.** No argv on either side
repairs it, exactly as the KV refusal could not be repaired by argv before the fork
gained the flag. The three ways out, none of which this task took:

1. llama.cpp gains a live effective-configuration report, as the `mlx-lm` fork did. This
   is the same shape as the fix that closed the KV term.
2. The gate reads the two values from argv **for a runtime that cannot report them**.
   This reopens the exact defect the live derivation closed, and would be a weakened
   admission clause. The brief forbids it and so does the record.
3. The decision accepts that this llama.cpp build can only be scored against a baseline
   that also declines to report — which on this host is nothing.

Option 1 is the only one that produces a scored pair without weakening the gate. It is a
product/ownership decision about the pinned llama.cpp build, not a measurement, and it is
the exact input this task now needs from review.

## 3. The number set (record readings — refused as a comparison, printed as measurements)

### 3.1 Prompt-token parity, before any latency number

| scenario | python-mlx-lm | llamacpp | skew | verdict |
| --- | ---: | ---: | ---: | --- |
| `short_prompt` | 41 | 41 | 1.0000 | comparable |
| `long_prompt_8k` | 7,784 | 7,784 | 1.0000 | comparable |
| `tool_call` | 313 | 313 | 1.0000 | comparable |
| `multiturn_prefix_reuse` | 7,784 | 7,784 | 1.0000 | comparable |
| `stability_soak` | 910 | 910 | 1.0000 | comparable |
| `context_75k` | **73,016** | **73,016** | 1.0000 | comparable |

### 3.2 Latency and throughput

TTFT, prefill and decode now share one generated-event definition on both runtimes: a
streamed delta counts when any of `content`, `reasoning` or `reasoning_content` carries a
non-empty string. That is the fix which invalidated every earlier llama.cpp gate reading.

| scenario | metric | python-mlx-lm | llamacpp | cand/base |
| --- | --- | ---: | ---: | ---: |
| `short_prompt` | TTFT s | 2.5314 | **2.1429** | 0.8465 |
| | prefill tok/s | 16.1969 | **19.1328** | 1.1813 |
| | **decode tok/s** | 6.7809 | **8.1450** | **1.2012** |
| | wall clock s | 10.4496 | **9.1055** | 0.8714 |
| `long_prompt_8k` | TTFT s | 107.2163 | **67.4273** | 0.6289 |
| | prefill tok/s | 72.6009 | **115.4428** | 1.5901 |
| | **decode tok/s** | 6.6705 | **7.7235** | **1.1579** |
| | wall clock s | 123.0636 | **81.1127** | 0.6591 |
| `tool_call` | wall clock s | 13.1792 | **11.0522** | 0.8386 |
| `stability_soak` | wall clock s | 291.3045 | **221.9062** | 0.7618 |
| `context_75k` | TTFT s | 1279.8919 | **950.8526** | 0.7429 |
| | prefill tok/s | 57.0486 | **76.7900** | 1.3460 |
| | ~~decode tok/s~~ | ~~8.0269~~ | ~~8.7764~~ | **withdrawn †** |
| | wall clock s | 1281.7795 | **952.5634** | 0.7432 |

† **WITHDRAWN — `context_75k` decode is not a decode result and must not be published as
one.** `completionTokens` is **16 on both runtimes**, so the figure is 15 tokens divided by
a ~1.9 s (baseline) / ~1.7 s (candidate) tail that follows a 950–1280 s prefill; a single
scheduling hiccup moves it several percent. The struck values are retained only so the
withdrawal is auditable against the record. **This scenario is valid for capacity, TTFT and
prefill, and is to be cited for those three only.**

`multiturn_prefix_reuse` timing is deliberately omitted from this table: §5 shows it is
non-comparable on sealed cache telemetry, and printing 0.0069x TTFT beside comparable
rows would invite exactly the misreading the telemetry exists to prevent.

### 3.3 Against the provisional numbers this task was told to reproduce or overturn

| claim under test | provisional (revision 1, invalid for decision) | corrected (this run) | outcome |
| --- | ---: | ---: | --- |
| "llama.cpp about 10 % slower at decode" (frame-derived, withdrawn) | — | llama.cpp **faster** on both scenarios where decode is meaningfully measured, +20.1 % / +15.8 % tok/s (`context_75k` withdrawn, §3.2) | **overturned** |
| decode, `long_prompt_8k`, tok/s | 6.5983 vs 7.8796 (1.1942x) | 6.6705 vs 7.7235 (**1.1579x**) | reproduced, slightly smaller |
| TTFT, `long_prompt_8k`, s | 111.4018 vs 69.4511 (0.6234x) | 107.2163 vs 67.4273 (**0.6289x**) | reproduced |
| prefill, `long_prompt_8k`, tok/s | 69.8732 vs 112.0789 (1.6041x) | 72.6009 vs 115.4428 (**1.5901x**) | reproduced |
| resident bound at 73,016 prompt tokens | 47,791,331,280 B vs 45,097,521,165 B (−5.64 %) | **refused on both runtimes** | **neither — see §4** |

**The decode direction moved toward llama.cpp, which is the direction the bias warning
told me to distrust, so it is worth being explicit about what was and was not done.**
Nothing was retried, reordered or tuned. This is the first and only full pair run in this
revision; the numbers above are from that run; no scenario was re-executed.

## 4. REFUSAL 2 — memory, on every gated window of both runtimes

**`context_75k` memory is not scored on either runtime.** It is explicitly refused, and so
is every other **gated** window of the pair except one. "Gated" is load-bearing here and
§4.0 states exactly what it excludes; the earlier revision of this report said "every
window", which is false against the run's own output.

| window | baseline status | candidate status |
| --- | --- | --- |
| `short_prompt` scenario | partial — `resident-mapped-file-sampling-gap` | **measured**, 34,731,153,644 B |
| `long_prompt_8k` scenario | partial — `resident-mapped-file-sampling-gap` | partial — `resident-mapped-file-sampling-gap` |
| `tool_call` scenario | partial — `resident-mapped-file-sampling-gap` | partial — `resident-mapped-file-sampling-gap` |
| `multiturn_prefix_reuse` scenario | partial — `resident-mapped-file-sampling-gap` | partial — `resident-mapped-file-sampling-gap` |
| `stability_soak` scenario | partial — `resident-mapped-file-sampling-gap` | partial — `mach-physical-footprint-sampling-gap` |
| **`context_75k` scenario** | **partial — `mach-physical-footprint-sampling-gap`** | **partial — `mach-physical-footprint-sampling-gap`** |
| whole-pass process peak | partial — `mach-physical-footprint-sampling-gap` | partial — `mach-physical-footprint-sampling-gap` |

The single `measured` window **among those the gate judged** is on the candidate only, so
**no scenario in this pair has a scored memory value on both sides. The memory axis
produced no comparison at all.** Four further windows carry a score without being judged
at all — §4.0 — and they are excluded from that statement and from every memory claim in
this report.

### 4.0 Four windows publish a score without ever facing the coverage gate

The table above is the complete list of windows the coverage gate *judged*. It is not the
complete list of `RuntimeMemoryPeak` windows this run emitted. `session.json` emits four
more, and all four come back `status: "measured"`, `issues: []`, with a populated
`scoredBytes`:

| window | status | issues | `scoredBytes` | mapped stamps |
| --- | --- | --- | ---: | ---: |
| baseline `warmupMemory` | **measured** | `[]` | 29,120,518,072 | 1 |
| baseline `soakMemory` | **measured** | `[]` | 29,827,094,504 | 20 |
| candidate `warmupMemory` | **measured** | `[]` | 34,248,152,988 | 1 |
| candidate `soakMemory` | **measured** | `[]` | 44,346,176,540 | 20 |

**They are `measured` because they never reach the gate.** `BenchmarkPass.swift:100`
(`recordWarmupMemory`) and `BenchmarkPass.swift:107` (`recordSoak`) build them by direct
`RuntimeMemoryPeak(summarizing:)`. The gate lives in
`BenchmarkFootprintSampler.coveredPeak` (`BenchmarkFootprintSampler.swift:333-345`) and is
reached only from `currentWindowPeak()`, `processPeakSoFar()` and `capturePeaks()`.
`RuntimeMemoryPeak.init(summarizing:)` (`RuntimeMemoryAccounting.swift:272-317`) sets
`.measured` and fills `scoredBytes` on "all reads complete" alone, with no coverage
judgement, and `validatedScoredBytes` then returns a value for them.

Measured against this run's own stamps, those four are the **worst-covered series in the
whole pair**:

| window | mapped stamps | mapped gap min/med/max | over the 7.0 s bound | Mach max gap | over the 125 ms bound |
| --- | ---: | ---: | ---: | ---: | ---: |
| baseline `soakMemory` | 20 | 13.862 / 14.682 / 15.117 s | **19 / 19** | 15.117 s | **19 / 19** |
| candidate `soakMemory` | 20 | 10.141 / 10.916 / 13.129 s | **19 / 19** | 13.129 s | **19 / 19** |
| baseline `warmupMemory` | 1 | — | — | — | one point; `coveredPeak` refuses `< 2` as `resident-memory-sampling-coverage-insufficient` |
| candidate `warmupMemory` | 1 | — | — | — | same |

So a 15 s-gap series publishes as `measured` with a score, while a 0.05 s series with one
0.62 s hiccup is refused as `partial`. **Bounded, so this is not over-read:**
`RuntimeBenchmark.decide` references neither `soakMemory` nor `warmupMemory` — neither name
appears anywhere in `Sources/MLXSwiftRuntimeContract/` — so **no scored comparison consumed
them** and the pair's admission outcome is unaffected. This is a reporting defect, not a
scoring one. Routing both construction sites through `coveredPeak` is open instrumentation
debt, recorded here rather than fixed, because this revision changes no measurement code.

**Consequence for anything quoted downstream:** any number taken from `soakMemory` or
`warmupMemory` — including the +6,201,119,136 B soak climb in §6 — carries **no coverage
guarantee**, and must be stated with that provenance or left out.

### 4.1 The cause, measured from the run's own raw series

The mapped-file component is refreshed at 0.2 Hz by forking `/usr/bin/vmmap -summary`, so
the interval between two mapped observations is `5.0 s sleep + one read`. Inverting that
over this run's raw series:

| runtime | target | mapped observations | gap min/median/max | derived `vmmap` cost min/median/max | gaps over the 7.0 s bound |
| --- | --- | ---: | ---: | ---: | ---: |
| `python-mlx-lm` | 27–45 GiB anonymous | 289 | 0.102 / 7.569 / 9.145 s | — / 2.569 / 4.145 s | **268 / 288** |
| `llamacpp` | 26.3 GiB mapped + 6–16 GiB anonymous | 201 | 0.508 / 7.221 / 10.823 s | — / 2.221 / 5.823 s | **179 / 200** |

The contract bound is `samplingIntervalSeconds + 2 x observedMappedFileReadCostSeconds`
= `5.0 + 2 x 1.0` = **7.0 s**, and its `observedMappedFileReadCostSeconds` was calibrated
at 0.608–0.850 s against a ~3 MB and a ~0.9 GiB target. **Against a 26–45 GiB target the
same read costs a median 2.2–2.6 s and up to 5.8 s**, so the bound is below the cadence
the reader can deliver on the processes this comparison actually measures, and the window
correctly refuses rather than scoring from a stale value.

The Mach component behaved as designed almost everywhere — 0.060–0.069 s maximum gaps
inside the fast scenario windows, well inside its own 125 ms bound. It exceeded it in a
handful of places under load: a 0.6179 s and a 0.2731 s gap inside `context_75k`
(baseline, host load 13.995), and eight gaps of 0.13–0.29 s across the candidate's
whole-pass series. Those are what refuse the `context_75k` and process-wide windows.

**Correction, revision 5: no read failed in either pass.** An earlier revision of this
report read the per-window `readFailureCount: 1` as an observed read failure and gave it a
physical cause ("the final sample against a process that had already been torn down"). The
run says otherwise: `session.json` reports `memorySamplesReadFailed: 0` and
`memorySamplesMalformed: 0` on **both** passes. That `1` is the **synthetic coverage
marker** `coveredPeak` appends when the gate refuses —
`RuntimeMemoryPeak(summarizing: reads + [.readFailed(issue)])`
(`BenchmarkFootprintSampler.swift:344`). It tracks the refusal, not a read, and the records
show it exactly: `readFailureCount` is `1` on **every one of the 26 windows whose status is
`partial`** — all 6 baseline scenarios, all 6 candidate scenarios, both per-scenario
process-peak-so-far series on each side, and both whole-pass peaks — and `0` on the single
window that came back `measured` (candidate `short_prompt`). **No physical conclusion is
drawn from it, because the data supports none.**

### 4.2 What was deliberately not done

**The bound was not widened.** Re-deriving `observedMappedFileReadCostSeconds` from the
28 GB targets would raise it to roughly 16 s and would probably make these windows score.
Doing that *after* seeing my own run refused, on data whose direction I already knew, is
the confirmation-bias move the brief names explicitly. The calibration gap is real and is
reported here as a finding for review to rule on; it is not something a measurement run
gets to fix on itself.

### 4.3 The unscored components, labelled as such

Not evidence. The peak sample of a refused window is a real reading of one instant, but
the window failed coverage, so a higher transient may have gone unobserved. Recorded so
the next revision can tell whether re-derivation changes the answer or only the status:

| runtime | `context_75k` Mach footprint | `context_75k` mapped upper bound | sum (unscored) |
| --- | ---: | ---: | ---: |
| `python-mlx-lm` | 48,431,289,344 B | 2,343,936 B | 48,433,633,280 B |
| `llamacpp` | 16,162,537,728 B | 28,239,409,972 B | 44,401,947,700 B |

The revision-1 pair at the same prompt was 47,791,331,280 B against 45,097,521,165 B.

## 5. REFUSAL 3 — two scenarios are non-comparable on sealed cache telemetry

Every scenario sealed `usage.prompt_tokens_details.cached_tokens` from both runtimes.
Both runtimes reported it on every scenario; nothing was `unknown` and nothing was
malformed.

| scenario | baseline | candidate | comparability |
| --- | --- | --- | --- |
| `short_prompt` | miss, `[0]` | miss, `[0]` | symmetric — scoreable |
| `long_prompt_8k` | miss, `[0]` | miss, `[0]` | symmetric — scoreable |
| `tool_call` | miss, `[0]` | miss, `[0]` | symmetric — scoreable |
| `multiturn_prefix_reuse` | miss, `[0, 0, 0]` | **hit, `[5736, 7780, 7809]`** | **one-sided — non-comparable** |
| `stability_soak` | miss, `[0]` x 20 | **hit, `[18]` x 20** | **one-sided — non-comparable** |
| `context_75k` | miss, `[0]` | miss, `[0]` | symmetric — scoreable |

Two things follow, both measured rather than assumed:

- **`llama-server` holds a cross-request prefix cache and it survives across scenarios.**
  Its first `multiturn_prefix_reuse` turn already found 5,736 of 7,784 prompt tokens
  cached, before that scenario had sent anything — warmed by `long_prompt_8k` before it.
  That is why its TTFT there is 0.726 s against the baseline's 105.206 s, and it is a
  cache result, not a runtime-speed result.
- **The incumbent's configured `--prompt-cache-size 1 --prompt-cache-bytes 8GB` did not
  fire once in this run** — `cached_tokens` is 0 on all six scenarios and all 26 turns.
  That reported zero is corroborated by timing rather than trusted on its own: the
  baseline's `multiturn_prefix_reuse` spent 347.591 s over three turns on one shared
  7,784-token prefix, which is about three full prefills at its measured 72.6 tok/s, and
  its third turn still paid 105.206 s to first token. The asymmetry the shipped config
  declares therefore runs the opposite way from the way it is written, and this run
  measures that rather than assuming it.

Had `contextPolicy` admitted the pair, these two scenarios would still have been refused
per-scenario rather than scored.

## 6. What passed on both runtimes

- **75,000-token capacity: met by both.** 73,016 prompt tokens, identical on both sides,
  both completed, neither aborted or degraded. (The shipped suite's 2,027 filler repeats
  render 73,016 tokens under this tokenizer, not 75,000; that is the pinned value and it
  matches the accepted MLX-Swift comparison.)
- **Tool-call parity: satisfied by both.** The check is not a status code: it requires
  `finish_reason == "tool_calls"`, a call naming `read_coolant_pressure` exactly,
  arguments that parse as JSON, and every key in the declaration's `required` list.
  Both runtimes satisfied all four at an identical 313-token prompt.
- **Stability: 20 of 20 iterations on both**, no failed iteration. Completion tokens
  1,280 baseline against 1,248 candidate under a 64-token-per-iteration budget.
- **MTP off on both**, `speculation=off` in both records, read by the gate from each
  running process's `GET /slots` inside its own observation window.

One stability observation that is not a pass/fail, and **not a leak, not a memory
regression, and not a gated number**: llama.cpp's resident upper bound climbed
**+6,201,119,136 B across the soak** while the baseline's fell by 509,935,592 B.

**Provenance, which must travel with this number wherever it is quoted.** It is a two-point
first-to-last delta —
`session.json → candidate.soak.resident_memory_upper_bound_drift_bytes`, computed from the
first and last samples of the `soakMemory` window. That window is one of the four series in
§4.0 that **never faced the coverage gate**: 20 stamps at a ~10.9 s median cadence, 19 of
19 gaps outside both the 7.0 s mapped bound and the 125 ms Mach bound. Two of its twenty
points, subtracted, in a document that everywhere else refuses memory for insufficient
coverage. The entire climb sits in the Mach anonymous component (9,905,647,432 →
16,106,766,568 B, exactly +6,201,119,136 B) while the mapped component is constant at
28,239,409,972 B on every sample.

It is the same shape as the anonymous climb TASK-260828-2wcrph flagged and this task again
does not establish what it is. Report it only as an open observation carrying the
provenance above, or leave it out.

## 7. Validation run for this revision

Every command below was run as a standalone process; these are the real process exits.
The revision-3 evidence did **not** cover this source — `Sources/` and `scripts/` were
last modified at 19:10 and 19:25 today, after the 15:06 revision-3 smoke — so all of it
was re-run against the working tree that produced the measurements.

| command | exit | result |
| --- | ---: | --- |
| `swift build -c release` | 0 | pre-existing `quantization` deprecation warning only |
| `swift test -c release` | 0 | **410 tests in 32 suites** |
| `scripts/benchmark-gate-smoke.sh` (production entry) | 0 | **120 PASS / 0 FAIL** |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | no output |
| `bash -n scripts/benchmark-gate-smoke.sh` | 0 | |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | |
| `benchmark-run` (the pinned pair) | **4** | **expected-refusal path — inadmissible on `contextPolicy`, no `decision.json`** |

No production source was changed by this task. It is a measurement revision.

**Revision 5 changed no source, no test and no script** — only this report and the three
repo-persisted documents (`.research/260829_llamacpp-against-the-python-baseline.md`,
`LOGBOOK.md`, `tools/mlx-swift-runtime-prototype/README.md`). The table above therefore
still describes the exact tree that produced the measurements, and is carried forward
rather than re-claimed. The revision-5 documentation delta was re-verified on its own terms
in §7.1.

## 8. Artifacts

Under `.temp/TASK-260829-3k4qrc/rev4/`:

| path | what |
| --- | --- |
| `TASK-260829-3k4qrc-rev4.benchmark.toml` | the config both passes used |
| `run-rev4.sh` | the runner: one foreground `benchmark-run`, own process group only |
| `run-rev4.log`, `run-rev4.exit` | the invocation and its exit code |
| `run-rev4-interval.txt` | driver start/finish/wall/exit and the orphan check |
| `run-rev4-sweeps.log` | pre- and post-run `ps`/`vm_stat`/`uptime` sweeps |
| `raw-processes-before.txt`, `raw-processes-after.txt` | full raw process lists either side of the run |
| `session-rev4/records/*.json` | both run records, with raw timestamped memory series |
| `session-rev4/attest/*.json` | both attestations |
| `session-rev4/logs/*-runtime.log` | both runtime logs |
| `session-rev4/session.json` | lifecycle, soak and memory summaries |
| `TASK-260829-3k4qrc_tables-rev4.txt` | the rendered number set |
| `report_rev4.py` | the renderer, reading only what the gate wrote |
| `memory-coverage-analysis.txt` | per-window Mach and mapped gap distributions |
| `vmmap-cost-derivation.txt` | the `vmmap` cost derivation behind §4.1 |
| `probe-llamacpp-v1-models.json`, `probe-llamacpp-props.json`, `probe-llamacpp-slots.json` | §2.2's live-surface probe |
| `swift-test-release-rev4.log`, `benchmark-gate-smoke-rev4.log`, `swift-format-lint-rev4.log`, `shellcheck-rev4.log` | the validation logs |

## 9. What review has to decide

0. **§4.0 — `warmupMemory` and `soakMemory` bypass the coverage gate.** Both are built by
   direct `RuntimeMemoryPeak(summarizing:)` in `BenchmarkPass` and publish as `measured`
   with a score while being the worst-covered series in the run. Nothing in
   `Sources/MLXSwiftRuntimeContract/` reads them, so no decision was affected, but the two
   construction sites should be routed through `coveredPeak` — or the two windows should
   stop carrying `scoredBytes` — before anything downstream is allowed to score them. This
   revision reports the bypass; it does not change measurement code.
1. **§2 — the `contextPolicy` prefill/reasoning terms.** llama.cpp cannot report them and
   the gate will not read argv. Either the pinned llama.cpp build gains a live effective
   configuration report, or this pair is permanently unscoreable, or an admission clause
   is weakened. This task did not choose.
2. **§4 — the mapped-file coverage bound.** 7.0 s was derived honestly from ≤0.9 GiB
   targets and is too tight for 26–45 GiB ones by a factor of about 2.5. Re-deriving it
   from the real targets is the same class of fix as the one that produced 7.0 s, but it
   must be decided before the next run and not after seeing which way the numbers went.
3. **§5 — the declared cache asymmetry is backwards in the shipped config.** Measured:
   `llama-server` reuses across scenarios, and the incumbent's configured prompt cache did
   not fire at all.
4. **§6 — llama.cpp's +6.2 GB soak climb** is unexplained and should be before any
   deployment decision. It is **not** a leak or a memory regression on this evidence: it is
   a two-point delta off an ungated window (§4.0). Whatever is decided, the number does not
   travel without that provenance.
5. **§3.2 — `context_75k` decode is withdrawn** as a decode result (16 completion tokens on
   both runtimes). That scenario is cited for capacity, TTFT and prefill only.
