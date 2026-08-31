# llama.cpp measured against the Python mlx-lm baseline

> **SUPERSEDED FOR DECISION PURPOSES, 2026-08-31.** The migration decision this document
> fed into is now recorded in `.research/260831_local-qwen-runtime-comparison-study.md`
> (checksummed snapshot: `articles/260831_local-qwen-runtime-comparison-study/`):
> **NO-GO on both MLX Swift and llama.cpp; Python `mlx-lm` remains the default.** Two claims
> that originated in this programme are withdrawn there and must not be quoted from here:
> the frame-derived "about 10 % slower decode" (**overturned** — llama.cpp decodes faster)
> and the −5.64 % / "about 9 % more frugal" memory advantage (**withdrawn**, with no
> replacement in either direction). This document remains the measurement record; it is not
> the decision.

> **Revision-5, 2026-08-30, TASK-260829-3k4qrc: the corrected pair was run.** The full
> pinned six-scenario suite ran end to end against `mlx_lm-kv76800-45a472f.server` and
> `llama-server` b10621, sequentially, in one `benchmark-run` invocation, MTP off on
> both, 58 m 55 s wall. All twelve scenario runs succeeded at exact prompt-token parity,
> including the 73,016-token capacity probe. **The gate refused the pair again, exit 4,
> and the term that broke is not the one that was fixed:** `kv=76800` now agrees on both
> sides — STORY-260830-2vrhg1's `--max-kv-size` closed §2's blocker — but `contextPolicy`
> also pins the prefill chunk and the reasoning effort, and `llama-server` reports
> neither on any live endpoint. Probed directly on this build: `/v1/models` `meta` carries
> no `n_batch`/`n_ubatch`, `/props` carries none either, and its
> `default_generation_settings.params.reasoning_format` read `"none"` under
> `--reasoning-effort medium`, so it does not track the launch. Since the gate derives all
> three terms from the live listing and never from argv, the pair is unscoreable on this
> llama.cpp build without either a live effective-configuration report from llama.cpp or a
> weakened admission clause. **Memory is refused on every scenario window and both
> process-wide peaks of both runtimes, including `context_75k`** (the one exception is
> `short_prompt` on the candidate)**:** one `vmmap -summary` against these 26-45 GiB
> targets costs a median 2.2-2.6 s and up to 5.8 s where the 7.0 s bound was calibrated on
> 0.608-0.850 s reads against <=0.9 GiB targets, so 268/288 baseline and 179/200 candidate
> mapped observations fall outside it. The bound was deliberately **not** widened after the
> fact. **Four further windows carry a score without ever facing that gate and are excluded
> from every claim here:** `warmupMemory` and `soakMemory` on both runtimes are built by
> direct `RuntimeMemoryPeak(summarizing:)` in `BenchmarkPass`, never reach
> `BenchmarkFootprintSampler.coveredPeak`, and publish as `measured` despite being the
> worst-covered series in the run (both `soakMemory` windows: 20 stamps, 10.1-15.1 s gaps,
> 19/19 outside both bounds). No scored comparison consumes them — neither name appears in
> `Sources/MLXSwiftRuntimeContract/` — so the pair's admission outcome is unaffected.
> **Decode overturns the withdrawn "about 10 % slower" claim:** on one shared
> generated-event definition llama.cpp decodes faster on both scenarios where decode is
> meaningfully measured — 8.145 vs 6.781 and 7.724 vs 6.671 tok/s — and `long_prompt_8k`
> TTFT reproduces at 0.629x. **The `context_75k` decode figure is withdrawn as a decode
> result** — its budget is 16 completion tokens on both runtimes, so it is 15 tokens over a
> ~1.8 s tail after a 950-1280 s prefill; cite that scenario for capacity, TTFT and prefill
> only. Full number set and the three refusals: board outcome resource
> `TASK-260829-3k4qrc_measured-pair-outcome.md` (local copy
> `.temp/TASK-260829-3k4qrc/rev5/`). The provisional numbers below remain historical and
> are still not a decision set.

> **Revision-4 instrumentation, 2026-08-30:** no benchmark rerun was performed
> in this revision either. The next run must use the merged 76,800-token live KV bound,
> independently timestamped 20 Hz physical-footprint and bounded 0.2 Hz `vmmap`
> observations, synchronous window-boundary samples, and MTP observed off. A
> reused mapped value retains its original timestamp and cannot mint fresh
> evidence. A scored window proves coverage independently for both components,
> under a bound sized to the reader that produces each: 125 ms for the
> in-process Mach series, and 7.0 s for the mapped-file series, derived from the
> 0.61-0.85 s measured cost of one `vmmap -summary` fork at the 5 s cadence.
> Every emitted peak states that mapped-file transients shorter than 7.0 s are
> not observable, and a peak that does not state it cannot be scored. Revisions
> 1-3 claimed 125 ms for the mapped component too, which no window could satisfy
> and which made every memory delta `unmeasured`. Sparse, stale, untimestamped,
> failed, partial, or malformed series are refused. The provisional numbers below remain
> historical and are not a decision set.
>
> Both cache-policy asymmetries must be presented together in that run:
>
> - Python-only `--prompt-cache-size 1 --prompt-cache-bytes 8GB` can reduce TTFT
>   on a repeated prompt and can retain up to the configured cache budget;
>   direction for a cold, unique prompt is unknown.
> - llama.cpp per-slot KV reuse can reduce TTFT when a slot reuses a matching
>   prefix and can retain slot-local KV state; direction without an observed
>   reuse hit is unknown.
>
> Every scenario seals runtime-reported cached-token telemetry. A one-sided hit
> is non-comparable, while absent, partial, or malformed telemetry is `unknown`;
> both conditions are refused rather than scored. Symmetric reported hits or
> misses remain scoreable.

Task: TASK-260828-2wcrph (story STORY-260828-2faxgm)
Date: 2026-08-29
Author role: developer
Gate: the single-invocation `benchmark-run` from STORY-260827-m30k8z as extended
by TASK-260828-3fgca3, plus one reader fix this task made and proved (§4).

## The result in six sentences

The pinned six-scenario suite ran three times end to end against `llama-server`
and `mlx_lm-relux.server` on the same host, sequentially, with no other model
resident. **Every scenario succeeded on both runtimes, including the capacity
probe at 73,016 prompt tokens, and prompt-token parity was exact — ratio
1.0000 — on all six.** The gate **refused to score any of the three pairs**,
exit 4, on `contextPolicy`: `mlx_lm-relux.server` has no flag that bounds its KV
cache and reports no bound, `llama-server` has no unbounded mode, so the two
pins are `kv=unbounded` against `kv=<n>` and no argv on either side can make
them agree (§2). That refusal is correct rather than a misconfiguration, because
`llama-server` allocates its whole KV arena at load while `mlx_lm.server` grows
its cache per request — measured, 65,536 bytes per token, and the two arenas
this task ran differ by exactly the 4.19 GiB that arithmetic predicts (§5.2).
**All three first-token-derived gate metrics are excluded from this decision**:
llama.cpp spells streamed reasoning `reasoning_content`, which the gate does not
read. The omission has mixed direction — it inflates llama.cpp TTFT, understates
its prefill, and overstates its decode. A replacement probe explicitly requested
streamed usage and received exact parity, 7,784 prompt and 106 completion tokens
from each runtime; it measured Python/llama.cpp decode at 8.449/8.772 tok/s, a
1.038x single-probe observation that is too small and too narrow to establish a
general decode direction (§4.2). **On the owner's two questions the honest
answers are: llama.cpp is modestly faster overall — 0.81x to 0.97x wall clock on
like-for-like scenarios, without attributing that delta to decode — and modestly
more frugal, provably below the baseline at the 73k
capacity probe by about 9%, indeterminate at small working sets, and nowhere
near the order of magnitude the recorded `peakPhysicalFootprintBytes` implies,
because that counter cannot see llama.cpp's weights at all (§5).**

## 0. What is not comparable, stated before any number

Three things travel with this record and are not conclusions of it. They are
TASK-260828-3g87i4's declared non-equivalences, carried into all six run records
by the gate and reproduced here because no report of this decision may omit
them:

1. the MLX build **drops the MTP head** the GGUF carries as `blk.64` — 8
   quantized tensors, 451,319,808 bytes on disk, skipped at load;
2. the **vision tower is placed differently** — 333 bf16 tensors inside the MLX
   safetensors shards against a separate 931,145,984-byte GGUF `mmproj` — and is
   resident in neither on the default text path;
3. GGUF **upcasts norms and 1-D tensors to F32** where the MLX build keeps the
   source bf16: 10,686,464 bytes of extra resident memory, no fidelity
   difference.

Item 1 was observed directly this time rather than taken on trust. `llama-server`
logs `model has unused tensor blk.64.*  -- ignoring` for all thirteen `blk.64`
entries at load, and the runtime consequence is measurable in §5.1: the resident
mapped-file set is the whole GGUF **minus** the MTP head, to within 0.01 GiB.

**MTP was off for every scored pass**, established by the gate from the process
rather than from the launch flag: `speculation=off` in both llama.cpp records of
the two runs taken after §4's fix. No with-MTP number was taken; the flag
`--spec-type draft-mtp` exists on this build and a deployment number for it is
left to whoever wants one, clearly outside this comparison.

Vision residency is not added to or subtracted from any memory figure here. On
the llama.cpp side the `mmproj` is a separate file that was never opened.

## 1. Host, and what else was on it

MacBookPro18,2, Apple M1 Max, 68,719,476,736 bytes of RAM, `sysctl`-read and
pinned into every record as `hostIdentity`.

`llama.cpp 0.3.0`, build **10621**, commit `c1d0e7a00`, Homebrew, reporting
itself as `b10621-c1d0e7a00` in its own `system_fingerprint`.
`mlx_lm-relux.server` from the pinned pipx venv.

**Every one of the three `benchmark-run` invocations was preceded and followed
by a `ps` sweep for `llama-server`, `mlx_lm`, `mlx-swift` and `model-harness run`
processes, and all six sweeps printed `(none)`.** They are in
`.temp/TASK-260828-2wcrph/run-*.log` at the top and bottom of each file. The two
passes inside each invocation are sequential by construction, with a 20 s settle
between them, and the gate refuses a pair whose wall-clock intervals overlap.

One honest note about the host: this workstation was **not** idle. Load average
sat between 4.8 and 8.8 before each run and `hostLoadAverageMax` reached 19.64
inside the llama.cpp capacity scenario. Nothing was killed to make room —
`ollama` was resident throughout at 14 MB with no model loaded, which is its
idle footprint and not a model this comparison had to wait for.

**A process-hygiene failure of my own, recorded because it bears on the
evidence.** Six side probes I launched between runs outlived the shells that
started them — job control does not survive a tool-call boundary here — and two
of them were 29 GB `llama-server` instances. They were found by `ps`, killed,
and the host verified clean **before** the two runs whose numbers appear below.
The affected artifacts are the *probes* taken during that window: three
completions in `probe-slots/` came back `{"code":500,"message":"Compute error."}`
because the host was out of memory, and those responses are not evidence of
anything about llama.cpp. No benchmark record was taken in that window; the
`(none)` sweeps bracketing each run are what establishes that.

## 2. The refusal: llama.cpp and this Python baseline cannot share a KV pin

### 2.1 What was measured

| runtime | reports a bound on `/v1/models`? | flag that states one | `contextPolicy` KV term |
| --- | --- | --- | --- |
| `mlx_lm-relux.server` | **no** — no `meta` block at all | **none exists** | `kv=unbounded`, always |
| `llama-server` b10621 | **yes** — `meta.n_ctx`, measured 8192 / 76800 / 32768 | `--ctx-size` | `kv=<n>`, always |

`mlx_lm-relux.server --help` lists `--chat-template-args`,
`--prefill-step-size`, `--prompt-cache-size`, `--prompt-cache-bytes`,
`--decode-concurrency`, `--prompt-concurrency`, `--draft-model`,
`--num-draft-tokens` and no KV bound option of any spelling. This is not a new
discovery: TASK-260828-2jbufw's own table already read **"KV bound: no flag;
unbounded"** for this runtime. What was new is following it to its conclusion.

### 2.2 What the shipped config said, and why it was wrong

`examples/model-harness.benchmark.toml` told the operator, beside the llama.cpp
profile, to *"match it to the baseline's `--max-kv-size`"*. That instruction is
unfulfillable against the Python incumbent and was written when
`--max-kv-size` was assumed to be an MLX-family flag. It is a **Swift-prototype**
flag: `mlx-swift-runtime-prototype serve --max-kv-size` exists, `mlx_lm.server`
has nothing like it. The comment is corrected in place, and the fiction had also
reached the test fixtures — see §4.3.

### 2.3 The refusal, at the production entry, three times

All three invocations exited **4** with:

```
pinned condition "contextPolicy" differs:
  baseline  "kv=unbounded;prefill-step=2048;reasoning=medium"
  candidate "kv=8192;prefill-step=2048;reasoning=medium"      (session-c8192)
  candidate "kv=76800;prefill-step=2048;reasoning=medium"     (session-c76800)
; these runs are not a comparison
```

No `decision.json` was written by any of them. **Both records and both
attestations were, because admission runs after the passes**, which is why this
report has measurements at all.

### 2.4 Why this is the right refusal and not a configuration mistake

The two runtimes do not merely spell the condition differently; they do not have
the same condition. `llama-server` reserves `n_ctx` tokens of KV at load and
pays for them on a 41-token prompt; `mlx_lm.server` grows its cache with the
request. §5.2 measures the difference and it is 65,536 bytes per token — 4.19
GiB between the two arenas this task ran, before a single token is generated.
Scoring a fixed 76,800-token arena against an unbounded growing cache and
calling the difference "the runtime" is precisely the mistake this pin exists
to prevent.

**The only admissible pair for a llama.cpp candidate is against a baseline that
can state the same bound**, which on this host means the Swift prototype. That
is not this task's comparison, and the brief was explicit that it is not.

## 3. What was measured anyway

Three invocations, all exit 4, all with complete records:

| session | llama.cpp arena | scenarios | note |
| --- | --- | --- | --- |
| `session-c8192-prefix-speculation-fix` | `--ctx-size 8192` | five, `context_75k` skipped | taken with the pre-fix gate; kept as the evidence of §4 |
| `session-c76800` | `--ctx-size 76800` | **all six** | the capacity run |
| `session-c8192` | `--ctx-size 8192` | five, `context_75k` skipped | re-taken with the fixed gate; the 8k-arena numbers below |

### 3.1 Prompt-token parity, checked before any latency number

The brief's first instruction, and the trap that has been sprung three times in
this story. Both runtimes rendered **identical** token counts on every scenario
of every run:

| scenario | python-mlx-lm | llamacpp | skew |
| --- | ---: | ---: | ---: |
| `short_prompt` | 41 | 41 | 1.0000 |
| `long_prompt_8k` | 7,784 | 7,784 | 1.0000 |
| `tool_call` | 313 | 313 | 1.0000 |
| `multiturn_prefix_reuse` | 7,784 | 7,784 | 1.0000 |
| `stability_soak` | 910 | 910 | 1.0000 |
| `context_75k` | **73,016** | **73,016** | 1.0000 |

`--chat-template-args '{"reasoning_effort": "medium"}'` and
`--reasoning-effort medium` render the same prompt. The 41-token
`short_prompt` is the same 41 tokens review measured when it caught the
`xhigh`-versus-`medium` skew, so the reasoning policy is pinned on both sides
and the fourth configuration difference was **not** in the prompt.

### 3.2 What the gate recorded, and which columns of it mean anything

Read §4.2 before this table. **`TTFT`, `prefill tok/s` and `decode tok/s` are
not comparable between these two runtimes**, because the instant the gate calls
"the first token" is a different instant on each: the first *reasoning* token
for `mlx_lm.server`, which publishes `delta.reasoning`, and the first *content*
token — after the entire think block — for `llama-server`, which publishes
`delta.reasoning_content`. The three columns are reproduced as recorded, marked,
because they are what the files contain and someone will read them.

**Wall clock is the one recorded latency number that survives**, since it needs
no first-token detection at all.

`--ctx-size 8192` (`session-c8192`):

| scenario | metric | python-mlx-lm | llamacpp | note |
| --- | --- | ---: | ---: | --- |
| `short_prompt` | wall clock | 5.89s | **5.80s** | 0.98x |
| | TTFT | 0.97s | 3.57s | *not comparable* |
| | prefill tok/s | 42.41 | 11.50 | *not comparable* |
| | decode tok/s | 10.78 | 25.08 | *not comparable* |
| `long_prompt_8k` | wall clock | 86.37s | **75.27s** | **0.87x** |
| | TTFT | 76.29s | 73.97s | *not comparable* |
| | prefill tok/s | 102.04 | 105.23 | *not comparable* |
| | decode tok/s | 10.41 | 80.79 | *not comparable; excluded by §4.2* |
| `tool_call` | wall clock | 9.15s | **6.86s** | 0.75x |
| `multiturn_prefix_reuse` | wall clock | 235.52s | **84.47s** | **0.36x**, see §3.6 |
| `stability_soak` | wall clock | 136.70s | **119.53s** | 0.87x |

`--ctx-size 76800` (`session-c76800`), including the capacity probe:

| scenario | metric | python-mlx-lm | llamacpp | note |
| --- | --- | ---: | ---: | --- |
| `short_prompt` | wall clock | 5.91s | **5.73s** | 0.97x |
| `long_prompt_8k` | wall clock | 94.39s | **76.13s** | **0.81x** |
| | decode tok/s | 10.03 | 81.31 | *not comparable* |
| `tool_call` | wall clock | 10.50s | **6.89s** | 0.66x |
| `multiturn_prefix_reuse` | wall clock | 282.74s | **36.68s** | **0.13x**, see §3.6 |
| `stability_soak` | wall clock | 138.50s | **123.11s** | 0.89x |
| **`context_75k`** | **succeeded** | **yes** | **yes** | both |
| | prompt tokens | 73,016 | 73,016 | 1.0000 |
| | wall clock | 893.83s | **862.97s** | 0.97x |
| | TTFT | 891.96s | *unmeasured* | §4.2 |
| | prefill tok/s | 81.86 | *unmeasured* | §4.2 |

**The 75,000-token capacity criterion is met by both runtimes.** Neither
aborted, neither degraded, and both answered from the same 73,016-token prompt.
The suite's `context_75k` scenario builds 2,027 filler repeats, which is 73,016
tokens under this tokenizer, not 75,000; that is the shipped suite's pinned
value and it matches the accepted MLX-Swift comparison exactly.

### 3.3 The authoritative streamed-usage probe

Measured outside the gate, because the gate cannot measure them on this pair:
one `long_prompt_8k` completion per runtime, back to back on the same host with
no other model runtime resident, same prompt, and `max_tokens` 256. The request
explicitly carried `stream_options: {"include_usage": true}`; the raw stream
from **both** runtimes carried `prompt_tokens: 7784`, `completion_tokens: 106`
and `total_tokens: 7890`. The probe refuses to derive decode when those counts
are absent, malformed or conflicting. SSE frame counts are retained only as
transport diagnostics and are never substituted for token counts.

| metric | python-mlx-lm | llamacpp | candidate / baseline |
| --- | ---: | ---: | ---: |
| prompt tokens from streamed usage | 7,784 | 7,784 | 1.0000 |
| completion tokens from streamed usage | 106 | 106 | 1.0000 |
| first generated token / TTFT | 103.310s | **76.784s** | 0.743x |
| prefill tok/s (7,784 ÷ TTFT) | 75.346 | **101.375** | 1.345x |
| decode tok/s (`(106 - 1) ÷ first-to-last text interval`) | 8.449 | **8.772** | 1.038x |
| SSE frames, diagnostic only | 106 | 106 | not a token count |
| reasoning/content frames, diagnostic only | 89 / 15 | 89 / 14 | not a token count |

The frame split confirms that the two transports have nearly identical shape;
it does **not** establish how many tokens each frame carries. The completion
count comes only from streamed usage.

**Two cautions on these numbers.** They are one completion per runtime. The
1.038x decode delta is not a general performance claim: the three retained
llama.cpp runtime logs report 10.51, 10.79 and 10.74 tok/s for their own
106-token generations, while this rerun's server log reports 8.69 tok/s and the
client interval reports 8.772. This probe establishes the count and this one
rate; it does not establish a stable decode advantage in either direction.

**What this means for the recorded numbers.** `decodeTokensPerSecond` for
llama.cpp is `(completion − 1) ÷ (wall − TTFT)`, and with the clock starting
after the think block the denominator is only the content tail. The recorded
80.79 tok/s is therefore an artifact and is excluded. The replacement 8.772
tok/s comes from the authoritative 106-token usage count and the first-to-last
generated-text interval; it is not reconstructed from 106 SSE frames.

### 3.4 Tool-call parity

`tool_call` **succeeded on both runtimes in all three runs**. The gate's parity
check is not a status code: it requires `finish_reason == "tool_calls"`, a call
naming `read_coolant_pressure` exactly, arguments that parse as JSON, and every
key in the declaration's `required` list — here `vehicle_id` — present in them.
Both runtimes satisfied all four, at an identical 313-token prompt.

### 3.5 Long-running stability

`stability_soak`, 20 sequential completions, **succeeded on both runtimes in all
three runs** with no failed iteration. Completion tokens: 1,280 for Python
against 1,248 for llama.cpp, both against a 64-token-per-iteration budget, so
neither runtime was truncating differently. llama.cpp finished the soak in
119.53s against 136.70s at the 8k arena and 123.11s against 138.50s at the
76,800 arena — consistent across both runs. The gate defect and the small,
single-probe decode delta do not support attributing that wall-clock result to
faster decode; prefill and cross-request prefix reuse are separate contributors.

There is one stability observation that is **not** a pass/fail: llama.cpp's
anonymous footprint climbed monotonically across the pass, 1.86 → 2.59 → 3.66 →
4.27 → 11.43 GiB at the 8k arena, and 6.05 → 15.34 GiB at the 76,800 arena. The
soak's own `footprint_drift_bytes` is recorded per iteration in the record's
soak detail. I am not calling this a leak: the sampler reports a per-scenario
window peak and Metal compute buffers are plausibly the whole of it. It is
flagged because a 9.6 GiB climb over five scenarios is the kind of thing that
should be looked at before a deployment decision, and this task did not
establish what it is.

### 3.6 The prompt-cache asymmetry runs the other way

The declaration this run carried on the baseline — *"deployed with
`--prompt-cache-size 1 --prompt-cache-bytes 8GB`; `llama-server` holds no
cross-request prompt cache"* — is the framing
`examples/model-harness.benchmark.toml` uses, and **the measurement contradicts
its second half.**

`multiturn_prefix_reuse` sends three turns over one 7,784-token prefix. A single
prefill of that prefix costs about 75-85s on either runtime.

| run | python-mlx-lm | llamacpp |
| --- | ---: | ---: |
| `--ctx-size 8192` | 235.52s ≈ **three** prefills | 84.47s ≈ **one** prefill |
| `--ctx-size 76800` | 282.74s ≈ **three** prefills | **36.68s — less than one** |

So the incumbent's configured prompt cache **did not deliver prefix reuse on
this scenario**, and `llama-server`'s per-slot KV reuse did. The 36.68s is
below the cost of a single prefill, which means llama.cpp entered
`multiturn_prefix_reuse` with the prefix already warm from `long_prompt_8k`,
the scenario immediately before it — its cache is not merely per-conversation
but **survives across requests and therefore across scenarios**.

Two consequences, neither of which this task can settle:

* the `multiturn_prefix_reuse` figures above are **contaminated by scenario
  order** on the llama.cpp side and should not be read as a clean 0.13x;
* the asymmetry the records declare is stated backwards, and a future run should
  declare it as measured rather than as assumed.

## 4. Three readings the gate could not make on this runtime

All three are the same shape as `-ub`: a reading this gate makes, spelled by
llama.cpp in a way the reader did not know. None was found by reasoning about
the code; all three were found by running it. One is fixed here (§4.1), one is
measured and reported with its corrected numbers (§4.2), and one was a test
fixture describing a launch that cannot exist (§4.3).

### 4.1 FIXED — `/slots` names `speculative` where the reader was not looking

**The finding.** The first run's llama.cpp record pinned
`speculation=unread`. `speculation=unread` is in
`RuntimeBenchmark.unpinnableConditions` and is refused, so a second, independent
refusal was sitting behind the KV one: **on this build the gate could never
establish that MTP was off, which is a precondition of every scored comparison
in this story.**

**Why.** `RuntimeSpeculation.read` did
`let parameters = (slot["params"] as? [String: Any]) ?? slot` — consult `params`,
and fall back to the slot only when `params` is **absent**. Measured on this
build:

* `speculative` is a **top-level** field of a slot;
* `params` appears on a slot **once it has served a request** and carries
  sampling settings — `seed`, `temperature`, `top_k`, `top_p`, `n_predict` — and
  never names `speculative`.

So every slot the soak had touched went to the `continue`, and once traffic had
touched all four the whole array was invisible: `named == false`, `unread`. A
runtime that had answered the question in plain text was refused for a reading
the gate had in its hands. Reproduced directly on the 27B model: an unused slot
is readable, and the same slot after one completion is not
(`.temp/TASK-260828-2wcrph/probe-slots/`).

**The dangerous half.** This was not only a false refusal. With the pre-fix
reader, a slot carrying `{"speculative": true, "params": {"speculative": false}}`
read as `reported(false)` — a **drafting runtime scored as MTP-off**. The
mutant in §4.4 kills on exactly that case.

**The fix, additive.** Both placements are read per slot and neither shadows the
other; any `true` in either still returns `reported(true)` immediately; a slot
naming the field in **neither** place is still `unread`. Nothing that used to be
refused is now admitted.

**The reader was not added on faith.** The `/props`-shaped mistake — a field
that looks right and does not move with the launch — was checked for first. On
build `b10621-c1d0e7a00`, after eight completions had touched every slot:

| launch | top-level `speculative` | `params.speculative` | shipped reader | fixed reader |
| --- | --- | --- | --- | --- |
| `--spec-type none` | `false` on all 4 slots | never present | `reported(false)`¹ | `reported(false)` |
| `--spec-type ngram-mod` | **`true`** on all 4 slots | never present | `reported(true)`¹ | `reported(true)` |

¹ in that probe only one of four slots had acquired a `params` block, so the
shipped reader still found a readable slot. In the benchmark pass, where the
20-iteration soak had touched all four, it did not.

The field tracks the launch, on used and unused slots alike. `/props` reported
`speculative.types: "none"` under **both** launches and is still not read.

**Proved at the production entry.** The two runs taken after the fix both
record `speculation=off` in the llama.cpp record, written by the gate from its
own `/slots` exchange inside its own observation window. The pre-fix session is
kept as `session-c8192-prefix-speculation-fix` with `{"state":"unread"}` in its
attestation.

### 4.2 REPORTED, NOT FIXED — the first-token clock starts in a different place on each runtime

**This is the most consequential finding in the report**, because unlike §4.1 it
does not fail closed: it produces numbers, and the numbers are wrong.

`BenchmarkHTTPDriver.streamed` starts its clock on the first frame carrying
`delta.content` or `delta.reasoning`, reasoning included on purpose because this
model's template opens a `<think>` block and waiting for content would report
the length of the model's thinking as runtime latency. That is the correct
definition. **`mlx_lm.server` publishes `delta.reasoning`; `llama-server`
publishes `delta.reasoning_content`.** So the gate applies the intended
definition to the incumbent and the rejected one to the candidate.

Rerun with authoritative streamed usage, one `long_prompt_8k` completion per
runtime and every frame timestamped:

| | python-mlx-lm | llamacpp |
| --- | ---: | ---: |
| first frame with generated text | 103.310s | 76.784s |
| first frame the **gate** counts | 103.310s (`reasoning`) | **87.259s** (`content`, 10.475s later) |
| last generated-text frame | 115.737s | 88.754s |
| streamed completion tokens | 106 | 106 |

Three metrics are derived from that instant, and each is wrong in a different
direction for llama.cpp:

| recorded metric | derivation | effect on llama.cpp |
| --- | --- | --- |
| `timeToFirstTokenSeconds` | the instant itself | **inflated** by the whole think block |
| `prefillTokensPerSecond` | `promptTokens ÷ ttft` | **understated** |
| `decodeTokensPerSecond` | `(completion − 1) ÷ (wall − ttft)` | **overstated** — the denominator is only the content tail while the numerator still counts every reasoning token; the historical 80.79 is excluded |

And where the whole budget is spent thinking, no `content` frame arrives at all
and the clock never starts: a 16-token completion produced 91 characters of
generated text, **all** under `reasoning_content`, and zero the gate can see.
That is exactly the pattern in the records:

| scenario | max_tokens | llamacpp TTFT |
| --- | ---: | --- |
| `short_prompt` | 256 | 3.57s (inflated) |
| `long_prompt_8k` | 256 | 73.97s (inflated) |
| `multiturn_prefix_reuse` | 64 | **unmeasured** |
| `context_75k` | 16 | **unmeasured** |

This defect has **mixed direction**. It makes llama.cpp look worse on TTFT and
prefill, and better on decode. It is therefore incorrect to say that every
measurement defect in this comparison flattered llama.cpp.

**Why it is reported and not fixed here.** The unmeasured half fails closed —
`appendDelta` reports `verdict: "unmeasured"` with a blocker, never a number.
The measured half does **not** fail closed, and that is the argument for fixing
it. **TASK-260829-3cwcb6 owns the production fix and full rerun.** This interim
artifact does not change the driver, does not replace the known-wrong gate value
with a transport proxy, and excludes all gate-recorded TTFT/prefill/decode
figures from the decision. §3.3 records only the separately verified probe.

Consequence for this report: **llama.cpp's TTFT on the capacity probe is
unknown, not fast**, and its recorded 80.79 tok/s decode is an artifact and not
a result.

### 4.3 A fixture that described a launch that cannot exist

`RuntimeBenchmarkContextBoundTests.boundedMLXArgv` was
`unboundedMLXArgv + ["--max-kv-size", "8192"]`, and the records built from it
were labelled `python-mlx-lm`. That is an `mlx_lm.server` command line carrying
a flag `mlx_lm.server` does not have. It mattered because the suite's one
positive llama.cpp case, `admitsALlamaCPPCandidateWithNoClauseRelaxed`, then
read as though llama.cpp-against-the-Python-incumbent were admissible.

Renamed to `boundedSwiftArgv`, rebuilt as the Swift profile actually spells it,
and the records relabelled. The admissible pair is llama.cpp against
**mlx-swift**, and the suite now says so.

### 4.4 Mutants

Three, each applied to the shipped source, built, run, reverted. All killed.

| mutant | what it does | result |
| --- | --- | --- |
| **M-2wcrph-1** | restores the pre-fix `params ?? slot` reader | `swift test` **exit 1**, 3 tests red / 4 issues — including `{"speculative": true, "params": {"speculative": false}}` reading as `reported(false)`, the drafting-runtime-scored-as-quiet case |
| **M-2wcrph-2** | *narrowing*: a slot naming the field nowhere is read as "not speculating" | `swift test` **exit 1**, 6 issues across 3 tests, including the pre-existing `slotsWithoutTheFieldAreUnread` |
| **M-2wcrph-3** | *widening*: `contextPolicy` is dropped from `Pins.firstMismatch` — the change that would let this task's pair be scored | `swift test` **exit 1**; `refusesTheLlamaCPPCandidateAgainstThePythonIncumbent` reddens, so the refusal this whole report rests on is pinned |

## 5. Memory — the other difference that is in the metric, not the runtime

The brief predicted a fourth configuration difference. There were two, and
neither is in the prompt, the prefill chunk, the reasoning effort or the model
factory — §4.2 is one and this is the other. **Both are in the measurement
itself.**

### 5.1 `ri_phys_footprint` does not see llama.cpp's weights

Every `peakPhysicalFootprintBytes` in every record is
`proc_pid_rusage(...).ri_phys_footprint`. Sampled on the **same process at the
same moment** as `ps` RSS and `vmmap -summary`, each runtime holding the 27B
model after one completion:

| reading | `llama-server` | `mlx_lm-relux.server` |
| --- | ---: | ---: |
| `ri_phys_footprint` — **what the records carry** | **1.41 GiB** | **27.62 GiB** |
| `ps` RSS | **28.09 GiB** | 24.26 GiB |
| `vmmap` "mapped file", resident | **26.6 GiB**, of which **dirty 0** | — |

`llama-server` `mmap`s the GGUF. Those pages are resident and **clean**, so
`phys_footprint` — which counts dirty anonymous memory — excludes all 26.6 GiB
of them. MLX reads its weights into anonymous memory, so its footprint counts
them, and in fact exceeds its RSS because footprint also counts what the
compressor has taken.

**Had the pair been admitted, `maxPeakFootprintRatio: 1.1` would have scored
llama.cpp at roughly 0.08x the baseline and called it twelve times more
frugal.** That number is an artifact of the metric. This is the same class as
the brief's own warning about `resident_bytes`, arriving from the other
direction: on this pair `phys_footprint` is the reading that lies.

The 26.6 GiB is checkable. The GGUF is 29,036,089,344 bytes = 27.04 GiB, and
llama.cpp declines to load the 451,319,808-byte `blk.64` MTP head = 0.42 GiB.
27.04 − 0.42 = **26.62 GiB**. The mapped-file residency is the whole file minus
exactly the tensors the load log said it was ignoring — which is also the first
runtime confirmation that declared non-equivalence #1 is real in memory and not
only on disk.

### 5.2 The KV arena, derived and then measured

64 layers with `full_attention_interval` 4 gives 16 full-attention layers; 4 KV
heads; `head_dim` 256; K and V; f16:

```
16 x 4 x 256 x 2 x 2 = 65,536 bytes per token
  --ctx-size  8,192 -> 0.50 GiB
  --ctx-size 76,800 -> 4.69 GiB
  predicted difference: 4.19 GiB
```

Observed difference in llama.cpp's `short_prompt` footprint between the two
runs: **4.19 GiB** (1.86 → 6.05 GiB). The arena is paid in full on a 41-token
prompt, it is anonymous so `phys_footprint` *does* see it, and it is why the
`--ctx-size` you choose is a resident cost and not a limit.

### 5.3 What can actually be claimed about frugality

Since llama.cpp's true resident cost is its recorded footprint plus its
mapped-file residency, and I did not sample mapped-file residency *during* the
passes, the honest form is a bound: at most `footprint + 26.6 GiB`, taking the
weights as fully resident. Against the baseline's measured peak:

| run | scenario | python-mlx-lm, measured | llama.cpp upper bound | verdict |
| --- | --- | ---: | ---: | --- |
| c8192 | `short_prompt` | 27.35 GiB | 28.46 GiB | indeterminate |
| c8192 | `long_prompt_8k` | 31.39 GiB | **29.19 GiB** | **llama.cpp lower** |
| c8192 | `multiturn_prefix_reuse` | 31.91 GiB | **30.87 GiB** | **llama.cpp lower** |
| c8192 | `stability_soak` | 28.34 GiB | 38.03 GiB | indeterminate |
| c76800 | `long_prompt_8k` | 31.36 GiB | 33.38 GiB | indeterminate |
| c76800 | **`context_75k`** | **45.20 GiB** | **41.05 GiB** | **llama.cpp lower** |
| c76800 | pass peak | 45.20 GiB | 41.94 GiB | **llama.cpp lower** |

**The answer to the owner's question.** llama.cpp is **modestly** more frugal,
and only where the working set is large: provably below the baseline on the 8k
prompt with a matched 8k arena, and provably below it on the 73,016-token
capacity probe — 41.05 GiB against 45.20 GiB, about 9%. At small working sets it
is indeterminate, and with the 76,800-token arena it is indeterminate almost
everywhere because the arena costs 4.69 GiB whether or not you use it. **It is
not the order-of-magnitude win the raw records imply, and anyone reading
`peakPhysicalFootprintBytes` out of these files without §5.1 will conclude
something false.**

Where the bound is loose the row says *indeterminate*, not *equal*. Closing
those rows means sampling mapped-file residency beside the footprint for the
duration of a pass, which is a change to `BenchmarkFootprintSampler` and is not
in this task.

## 6. Recommendations, in the order I would take them

1. **Complete TASK-260829-3cwcb6: read `reasoning_content` as a second spelling
   of a streamed reasoning delta, then rerun** (§4.2). This is the only finding
   here that puts wrong latency numbers into a record rather than refusing. The
   correction has mixed direction — TTFT and prefill move in llama.cpp's favour,
   decode moves against it — and invalidates prior llama.cpp latency records,
   which must be retaken rather than compared against.
2. **Sample resident mapped-file bytes beside `ri_phys_footprint`** (§5.1).
   Until then no memory number in this gate is comparable across an
   `mmap`-loading runtime and an anonymous-loading one, and
   `maxPeakFootprintRatio` would have scored llama.cpp at about 0.08x the
   baseline. This turns every *indeterminate* in §5.3 into a measurement.
3. **Decide what a llama.cpp-versus-Python comparison is supposed to mean**
   (§2). The gate is right that the two do not share a KV condition. The options
   are (a) accept that this incumbent can only be compared to the Swift
   prototype, (b) score llama.cpp against a Swift baseline pinned to the same
   `--max-kv-size` and treat the Python numbers here as context, or (c) declare
   "bounded arena versus growing cache" an asymmetry rather than a pin — which I
   do not recommend, because it is a first-order memory difference and declaring
   it away is how §5.1's 12x artifact would have reached a report.
4. **Re-declare the prompt-cache asymmetry as measured** (§3.6): `llama-server`
   does hold a cross-request prefix cache, the incumbent's configured one did
   not fire on `multiturn_prefix_reuse`, and llama.cpp's figure for that
   scenario is contaminated by scenario order.
5. **Look at llama.cpp's 9.6 GiB anonymous climb across a pass** (§3.5) before
   any deployment decision.

## 7. Artifacts

Under `.temp/TASK-260828-2wcrph/`:

| path | what |
| --- | --- |
| `TASK-260828-2wcrph.benchmark.toml` | the config all three runs used; two llama.cpp profiles differing only in `--ctx-size` |
| `run-c8192-v2.log`, `run-c76800.log`, `run-c8192.log` | the three invocations, host sweeps included |
| `session-c8192/`, `session-c76800/`, `session-c8192-prefix-speculation-fix/` | records, attestations, runtime logs, session manifests |
| `report.py` | renders a session into the tables above |
| `probe-slots/`, `probe-slots-used.sh`, `SlotsRepro.swift` | §4.1's measurements |
| `probe-memory/`, `probe-memory.sh`, `Footprint.swift` | §5.1's measurements and the SSE shape of §4.2 |
| `probe-ttft.py`, `probe-ttft.sh`, `probe-ttft-*.json` | superseded probe; raw outputs have empty `usage` and cannot establish decode |
| `TASK-260828-2wcrph-rework/probe_decode_usage.py`, `*-authoritative-raw.json`, `*-authoritative-summary.json` | §3.3 rerun; explicit streamed usage, raw timestamped frames, and verified token-count-derived rates |
| `smoke-01.log`, `build-02.log` | `benchmark-gate-smoke.sh` 0 failures; Release build |
