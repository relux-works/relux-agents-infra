# Engine Adapter Contract And Canonical Knob Set

Task: `TASK-260830-3euwsu`
Story: `STORY-260830-2mj14b`
Status: specification only — no engine ported, no adapter implemented
Primary evidence source: `.research/260831_local-qwen-runtime-comparison-study.md`
(hereafter **the study**), plus the tasks it cites

---

## 0. What this document is and is not

This is the canonical knob set and adapter contract for the local Qwen engine
family (`python-mlx-lm`, `mlx-swift`, `llama.cpp`, with `mlx-lm` variants
already distinguished by fork commit) that a `model-harness` engine adapter
must express, derived from differences the runtime comparison study and its
cited tasks actually **measured** — not from a guess at what engines vary in.

It is not:

- an implementation. `TASK-260830-1e9gse` ports three engines onto the plugin
  graph and `TASK-260830-2cgim0` consumes engine kind in `model-harness`
  profiles; both are separate, currently blocked on this task.
- a benchmark. No model is loaded and nothing here re-runs a measurement.
- a weakening of the comparison gate described in the study. Every admission
  clause the study describes (§3.1's six conditions, the `contextPolicy`
  refusal, the per-scenario cache refusal, the memory coverage refusal) is
  treated here as a fixed constraint the adapter must serve, not a target to
  loosen so a knob fits.

Every knob below states **what happens when an engine cannot express it** —
`reported`, `notReported`, or `unread` in the study's own vocabulary (§1) — and
cites the study section or task that found the difference. A knob with no
citation is not in this document.

---

## 1. The shared vocabulary: three states, not two

The study's benchmark gate already settled this taxonomy under adversarial
review (study §4.1, §4.3.5, `.research/260828_llamacpp-in-the-benchmark-gate.md`
lines 313–354), and this contract adopts it unchanged rather than inventing a
parallel one:

| State | Meaning | Adapter consequence |
| --- | --- | --- |
| **`reported`** | The live, running process answered the question about itself, on an endpoint built for the purpose. | Trusted. Becomes the pinned value. |
| **`notReported`** | The live process answered — the endpoint responded, was well-formed — and named no value for this term. A legitimate absence, not a malfunction. | For every knob this contract currently names (§3), refused — inadmissible, exactly like `unread` (§2). No knob below carries an active argv fallback for this state; a fallback could only be introduced by a future revision naming a new, explicitly-cited exception (§2). Never silently promoted to a number without saying `notReported` happened. |
| **`unread`** | The endpoint answered but the gate could not extract a bound: wrong type, non-positive, malformed JSON, non-2xx status, a shape the reader does not recognise. | **Always refused.** `unread` is in every knob's `unpinnableConditions`. It is never treated as `notReported` and never falls through to argv — an `unread` reading is evidence of a bug in the reader or the process, not evidence of absence. |

**A fourth outcome exists at the pair level, not the per-knob level: admission
refusal.** When one or more knobs disagree, or one side is `unread`, or (for
`contextPolicy`'s three terms specifically) the two sides are not all
`reported` together, the gate refuses to score the pair — exit 4,
inadmissible, no decision written (study §3.1, §4.1). This is the study's
central structural finding and this contract preserves it as a hard
requirement on any adapter: **a knob may not be silently dropped to make a
pair scoreable.**

A second, narrower refusal exists at the **scenario** level: a pair can be
admitted (`contextPolicy` agrees) and still have individual scenarios refused
because a per-scenario condition — sealed cache telemetry, in this study's
case — is one-sided (study §4.3.5). The adapter must be able to express both
granularities, because collapsing scenario-level refusal into pair-level
refusal would throw away four otherwise-comparable scenarios to protect two
non-comparable ones, and the reverse — silencing the scenario-level refusal —
would print `llama.cpp`'s 0.726 s TTFT beside the baseline's 105.206 s under a
shared prefix, which the study explicitly declined to do (§4.3.3).

---

## 2. The Prime Directive: derive from the live process, never from argv, with no standing exception

Every knob below is an instance of one rule, stated once here so it is not
repeated ten times: **a pinned condition is derived from what the running
process reports about itself, never from the launch argv it was started
with**, because argv is not proof of what the process parsed
(`--prefill-step-size 2048 --prefill-step-siz 999` runs the process at 999
while argv reads 2048 — study §1.2, §6.3, reproduced through production
`benchmark-run` in `TASK-260830-2hc5r2`'s revision 4 and 5 review rounds).

This document has **no standing exception**. An earlier draft of this
contract carved out Knob 1's `notReported` state as an allowed argv fallback,
citing an old design on the MLX-Swift-only arm
(`.research/260828_llamacpp-in-the-benchmark-gate.md:481`). That citation
describes a revision of `RuntimeContextWindow` that predates
`STORY-260830-2vrhg1` (which gave the `mlx-lm` fork a `--max-kv-size` flag at
all) and predates `TASK-260830-2hc5r2`, which found the identical fallback
live in production `benchmark-run`, root-caused it, and closed it:
`LOGBOOK.md:310` — *"`RuntimeBenchmark.contextPolicy` treated an answered
`/v1/models` response without `meta.n_ctx` as permission to reuse
caller-requested `--max-kv-size`; production `benchmark-run` therefore
accepted a pair whose attestations explicitly said `notReported`."* The fix
(same entry, and confirmed unchanged through the final revision by
`TASK-260830-2hc5r2_rework-rev6-results.md:18-30`, "All six non-value states
are inadmissible. None falls back to or is decoded from argv.") routes an
answered omission to `contextBoundNotHonoured` — refusal — not to an
argv-read value. Standardizing the old fallback as this contract's one named
exception would reopen the exact bug the project already found and fixed, so
Knob 1 carries no fallback below and this section names none.

The remaining rule on argv is therefore narrower than "when may argv be
read" — under the engines this contract currently covers, it is never read
for a pinned `contextPolicy` term. It stays relevant for two reasons this
document still tracks:

- Knob 10's per-engine parsing-precedence registry remains load-bearing for
  any *future* engine or state this contract has not yet enumerated — see
  Knob 10 for the distinction between that forward-looking scope and today's
  three engines.
- If argv were ever read for a pinned term, it must be read through a
  **per-engine parsing-precedence registry**, not as literal tokens (§4,
  Knob 10), because a registry that assumes `--flag value` wins ignores that
  `mlx_lm`'s Python `argparse` resolves unique abbreviations and applies
  last-wins on repetition, while the Swift prototype refuses ambiguous or
  duplicate flags outright rather than picking one (`TASK-260830-2hc5r2`
  progress, revision 5 rework and revision 5 review rounds).

---

## 3. The canonical knob set

Ten items. The first three are the pinned `contextPolicy` triad the study's
gate already enforces as a single admission condition; the remaining seven are
adapter concerns the study surfaced around them.

### Knob 1 — KV / context window bound

| | |
| --- | --- |
| **Argv spelling** | `mlx_lm.server` / `mlx-swift-runtime-prototype`: `--max-kv-size`, absent by default (unbounded on `mlx_lm`, absent-and-unbounded on the Swift prototype). `llama-server`: `--ctx-size` / `-c`, **always finite** — absent falls back to the model's trained context, not to unbounded. |
| **Live report** | `llama-server`: `meta.n_ctx` on `GET /v1/models`, present unconditionally, measured `8192`/`76800`/`32768` across sessions. The pinned `mlx_lm` fork (post `STORY-260830-2vrhg1`): also `meta.n_ctx` on `/v1/models`, but **only after cache construction** — the value is absent on the first `/v1/models` answer after launch and appears only once a real completion has built the `RotatingKVCache` (`TASK-260830-2hc5r2` progress, revision 6 rework: "KV remains live `meta.n_ctx` after cache construction"). `mlx-swift` prototype: reported live in the `kv=unbounded` era (study §4.2). |
| **Citations** | Study §4.1 ("both records pin `kv=76800`, derived by the gate from each running process's live `/v1/models` `meta.n_ctx`"); `.research/260828_llamacpp-under-the-managed-harness.md:96-110`; `TASK-260830-2hc5r2` progress (revision 6 rework). |
| **`reported`** | Trusted, becomes the pin. |
| **`notReported`** | **No argv fallback.** "The runtime answered and named none" is not evidence of `unbounded` — it is refused, `kv=not-reported` in `unpinnableConditions`, exactly like `unread` (`.task-board/.resources/TASK-260830-2hc5r2/TASK-260830-2hc5r2_rework-rev4-results.md:5-7`: *"`notReported` becomes `kv=not-reported`; `unread` remains `kv=unread`; both are inadmissible and neither can fall back to `--max-kv-size`. An answered omission with `--max-kv-size 76800` is refused as `contextBoundNotHonoured`."*). An earlier draft of this contract named this state as the one standing argv-fallback exception, citing a superseded MLX-Swift-only design (`.research/260828_llamacpp-in-the-benchmark-gate.md:481`, predates `STORY-260830-2vrhg1` and `TASK-260830-2hc5r2`); that fallback was found live in production, root-caused as a bug, and closed (`LOGBOOK.md:310`) — see §2. |
| **`unread`** | Non-object `meta`, non-integer/non-positive `n_ctx` → refused, `kv=unread` is in `unpinnableConditions` (`.research/260828_llamacpp-in-the-benchmark-gate.md:327-328`). |
| **Adapter obligation** | An adapter must **not** treat "`/v1/models` answered 200" as sufficient to read the KV pin on the `mlx_lm` fork. It must either drive one real completion first, or poll `/v1/models` again after the first generation, before treating a missing `n_ctx` as `notReported` rather than as "not yet constructed." Conflating those two produces a false `unbounded` pin on a runtime that is in fact bounded, one request away from proving it (§4 Knob 8 covers the general readiness-vs-attestability distinction this is an instance of). |

### Knob 2 — Prefill / prompt-eval chunk size (batch geometry)

| | |
| --- | --- |
| **Argv spelling** | `mlx_lm.server`: `--prefill-step-size`, default `2048`. `mlx-swift`: internal `prefillStepSize`, default `512`. `llama-server`: `--ubatch-size`, default `512` — **not** `--batch-size`/`-b`, which is a different, larger logical-batch unit defaulting to `2048` and is a false-friend name collision with `mlx_lm`'s *value*, not its *parameter*. |
| **Live report** | `mlx_lm` fork (post `STORY-260830-2vrhg1` revision 6): reports via live `meta.runtime_config` after the fix landed. `llama-server` build `b10621-c1d0e7a00`: **never reported, on any of 44 enumerated routes.** `n_ubatch`/`n_batch` appear in `tools/server/*.cpp` only as internal scheduling variables or `SRV_WRN` log format strings — there is no JSON key on any handler, established by pulling the complete route table out of `libllama-server-impl.dylib`'s string table and cross-checking server sources, not from a probe list. |
| **Citations** | `.research/260828_llamacpp-under-the-managed-harness.md:99`; study §4.1 (route enumeration, lines 380–390); `TASK-260830-2hc5r2` progress (revision 6 rework). |
| **`reported`** | Trusted (`mlx_lm` fork, `mlx-swift`). |
| **`notReported`** | On `llama-server` this is the build's **only** possible state for this term — it is structural, not transient. **Argv fallback is explicitly forbidden here**, same as Knob 1: falling back "reopens the exact defect that the live derivation closed" (study §4.1, option 2, named and rejected). |
| **`unread`** | N/A on this term for `llama-server` — there is no endpoint to return a malformed answer from. A future build that adds one inherits the general `unread` rule. |
| **Adapter obligation** | When paired against any engine that reports this term (`mlx_lm` fork, `mlx-swift`), an adapter serving `llama-server` on this build **must cause pair-level admission refusal**, not silently script around it. This is the study's central finding (§4.1, §6.3): it is a migration-risk property of the runtime, not a configuration mistake, and the only clean fix is upstream — `llama-server` gaining a live effective-configuration report (study §4.1 option 1, §7.2 item 4). An adapter that reads argv instead, or wires the false-friend `/props default_generation_settings.params.reasoning_format`-style endpoint (see Knob 3) as a substitute, is a forced fit and must not be built. |

### Knob 3 — Reasoning effort

| | |
| --- | --- |
| **Argv spelling** | `mlx_lm.server`: `--chat-template-args '{"reasoning_effort": "..."}'`. `mlx-swift` and `llama-server`: `--reasoning-effort` — **the same spelling**, but with different reportability (below). |
| **Live report** | `mlx_lm` fork (post revision 6): reports live. `llama-server`: `reasoning_effort` exists **only as an inbound request field** — parsed into `chat_template_kwargs` and re-emitted into an outbound `chatcmpl_body` for that one request. It is never part of any server-state response; the `--reasoning-effort` launch flag populates a default that no route reports. **Named false friend:** `GET /props` → `default_generation_settings.params.reasoning_format` read `"none"` while the process ran under `--reasoning-effort medium` — a value that exists, answers, and does not track the launch. |
| **Citations** | `.research/260828_llamacpp-under-the-managed-harness.md:100`; study §4.1 (lines 386–394, the false-friend paragraph); `.research/260829_llamacpp-against-the-python-baseline.md:473-480` documents the same false-friend shape on the sibling speculation endpoint, corroborating that `/props` is a stale-by-design surface on this build, not a one-off. |
| **`reported`** | Trusted (`mlx_lm` fork, `mlx-swift`). |
| **`notReported`** | Structural on `llama-server`, same as Knob 2 — no endpoint exists. **Argv fallback forbidden**, for the same reason as Knob 2. |
| **`unread`** | N/A — no endpoint. |
| **Adapter obligation** | Identical to Knob 2: causes pair-level admission refusal when paired against a reporting engine. **The false-friend endpoint must never be wired as a substitute reader**, even though it answers a plausible-looking value — a gate wired to `/props reasoning_format` would carry a value that does not track the launch, the exact trap the study already documented once on speculation's `/props` `speculative.types` field (Knob 4) and refused to repeat here. |

### Knob 4 — Speculative decoding state

| | |
| --- | --- |
| **Capability shape** | `llama-server`: `--spec-type <kind>` (e.g. `ngram-mod`, `draft-mtp`), multi-token prediction supported and the served GGUF carries the MTP head (`blk.64`, 8 quantized tensors). `mlx_lm.server`: no speculation flag exercised in this study; the 8-bit MLX build **drops** the MTP head entirely (skipped at load, logged as ignored by `llama-server` when it later loads the GGUF). `mlx-swift`: not characterised by this study; declared single-generation-at-a-time (`GenerationEngine` is an actor). A fourth shape is coming: `TASK-260830-18n40a` evaluates MTPLX, MLX-native MTP, and is explicitly blocked on this task because its shape must fit this contract rather than be bolted on after. |
| **Live report** | `llama-server`: **`GET /slots` → `[0].params.speculative`** (boolean), tracks the launch flag correctly on used *and* unused slots — proved by running `--spec-type ngram-mod` and reading `true` before any request touched that slot. **Named false friend:** `GET /props` → `params["speculative.types"]` stayed `"none"` under **both** a non-speculative and a `--spec-type ngram-mod` launch — it does not track the launch at all and "is still not read" by design. `mlx_lm.server`: exposes **no** speculation state on any endpoint — a legitimate capability absence, not a malfunction. |
| **Citations** | `.research/260828_llamacpp-in-the-benchmark-gate.md:339-354, 632-653` (status-code taxonomy and the `/props` vs `/slots` divergence table); `.research/260829_llamacpp-against-the-python-baseline.md:465-480` (production-entry proof, both pre-fix and post-fix sessions); study §4.3.4, §4.6 (attestation row), §T9. |
| **`reported`** | `llama-server` `/slots` returning a well-formed `speculative` boolean on a named slot. Trusted. |
| **`notReported`** | `mlx_lm.server` (no endpoint at all — legitimate absence); `llama-server` `/slots` returning 200 with an empty slot array ("nothing there to be speculating"). Admitted as `off` **only** when corroborated by "no speculation flag was passed at launch" — never inferred from `/props`. |
| **`unread`** | Status 0, any other 4xx/5xx, a 200 that will not parse, or a non-empty slot array naming no `speculative` key. **Refused** — `speculation=unread` refuses admission, the conservative direction (`.research/260828_llamacpp-in-the-benchmark-gate.md:353, 376`). |
| **Adapter obligation** | Model this as an **open enum**, not a boolean — `off \| ngram \| draft-model \| mtp \| notReported \| unread` — because MTPLX (`TASK-260830-18n40a`) is a third capability shape the study never measured and a boolean would force it into an existing case rather than adding one. `/props`-shaped endpoints must be enumerated per engine as **known false friends** and never wired as a reader for any knob, not just this one — the pattern repeats identically on Knob 3's `reasoning_format`. |

### Knob 5 — Stream / generated-event field naming

| | |
| --- | --- |
| **What it is** | Not a launch-time knob — a wire-protocol knob. Every timing metric (`timeToFirstTokenSeconds`, prefill tok/s, decode tok/s) is derived from the instant a streamed SSE delta first carries generated text, and "generated text" is spelled differently per engine. |
| **Per-engine spelling** | `mlx_lm.server` publishes reasoning tokens under **`delta.reasoning`**. `llama-server` publishes the identical concept under **`delta.reasoning_content`**. Both publish ordinary content under `delta.content` identically. |
| **Citations** | `.research/260829_llamacpp-against-the-python-baseline.md:487-493` ("`mlx_lm.server` publishes `delta.reasoning`; `llama-server` publishes `delta.reasoning_content`."); the corrected shared definition is confirmed live in study §4.3.3 ("a streamed delta counts when any of `content`, `reasoning` or `reasoning_content` carries a non-empty string"). |
| **Why this knob is different from the others** | It has **no refusal path**. Both fields are `reported` — present, well-formed, live — under both engines, so nothing in the gate's admission logic can catch a wrong field mapping. Using only the `mlx_lm`-native spelling (`content`/`reasoning`) against `llama-server` does not refuse; it silently mismeasures, and the errors are **mixed-direction**: TTFT and wall-clock time inflated by the whole `<think>` block, prefill throughput understated, decode throughput **overstated** — and on any completion where the whole token budget is spent thinking, the clock never starts at all (a 16-token completion producing 91 characters entirely under `reasoning_content` and zero characters the old gate could see). This was the study's own "REPORTED, NOT FIXED" finding (`.research/260829_llamacpp-against-the-python-baseline.md` §4.2) and it previously produced the now-withdrawn "llama.cpp is about 10% slower at decode" claim (study §4.7). |
| **Adapter obligation** | Maintain a **per-engine field-name table** for every "did the model produce output" definition, never a single shared constant. Because there is no refusal path to catch a stale or incomplete table, this table must be verified against a **live fixture per engine** as part of the adapter's own test suite (a negative test: assert that a completion whose entire budget is spent in `reasoning_content` on `llama-server` still starts the clock), not merely reviewed by inspection. |

### Knob 6 — Cache / prompt-cache semantics and telemetry

| | |
| --- | --- |
| **Configuration shape** | `mlx_lm` baseline is deployed with `--prompt-cache-size 1 --prompt-cache-bytes 8GB`. `llama-server` needs no equivalent flag — per-slot KV reuse is automatic and cross-request by default. |
| **The confusable flag** | `--prompt-cache-bytes`'s own help text reads "Maximum size in bytes of the KV caches" — verified in the pinned fork at `/Users/alexis/src/relux-works/mlx-lm`, `mlx_lm/server.py:1902` (commit `45a472f2d0cda166b7ffe1a80fe50dd9621f4303`, outside this repository, so not reachable from a repo-internal citation) — which describes a **stored prefix pool**, not the active generation's KV — exactly the kind of confusion an adapter exists to absorb rather than propagate into a comparison. |
| **Measured behaviour** | The baseline's configured cache **did not fire once** across six scenarios and 26 turns — `cached_tokens` is `0` everywhere, corroborated by timing rather than trusted from a reported zero (three `multiturn_prefix_reuse` turns on one shared 7,784-token prefix cost ~three full prefills, 347.6 s, with the third turn still paying 105.2 s). `llama-server`'s per-slot reuse **did** fire, one-sided, in exactly two scenarios: `multiturn_prefix_reuse` (`[5736, 7780, 7809]`) and `stability_soak` (`[18]×20`), warmed by an earlier scenario's traffic before the scenario under test had sent anything. |
| **Telemetry surface** | Both engines expose the **same field name**, `usage.prompt_tokens_details.cached_tokens`, on every response. No adapter-side name translation is needed for this knob — the divergence is behavioural, not nominal. |
| **Citations** | Study §4.3.5 (REFUSAL 2), §T3, §7.2 item 6; `.research/260829_llamacpp-against-the-python-baseline.md:69, 189, 389`. |
| **Admission consequence** | Not a per-engine `reported`/`notReported`/`unread` state — a **scenario-level** admission gate. When `cached_tokens` is one-sided under otherwise-identical prompts, that scenario is refused rather than scored, independent of whether `contextPolicy` admits the pair as a whole. Symmetric miss (both `0`) or symmetric hit stays scoreable. |
| **Adapter obligation** | Carry a `declare` field per side (mirroring the study's own `--baseline-declare` / `--candidate-declare` CLI flags, §3.7) that states the cache configuration in force and whether it is expected to fire, so an incumbent's non-firing cache is recorded as a **declared asymmetry** rather than silently tuned away or hidden. This is the same discipline MLX Swift's blocker list already applies to its own missing prompt cache (study §4.2, blocker 2: "declared as an asymmetry rather than tuned away"). |

### Knob 7 — Memory / weight-residency accounting

| | |
| --- | --- |
| **Why it is not one instrument** | `llama.cpp` `mmap`s its GGUF weights; those pages are resident and file-backed, invisible to a naive anonymous-footprint read. `mlx_lm`/`mlx-swift` allocate weights anonymously, visible to that same naive read. A single Mach-physical-footprint number therefore measures different things on the two engine families. |
| **The scored quantity, as built** | `residentMemoryUpperBoundBytes` = Mach physical footprint **+** the upper edge of `vmmap -summary`'s resident "mapped file" bucket. Both components must **independently** prove sampling coverage before a window may be scored: a 125 ms bound for the in-process Mach series, a 7.0 s bound for the external `vmmap`-derived mapped-file series. |
| **Measured coverage failure** | Against a 26–45 GiB target the external `vmmap -summary` fork costs a median 2.2–2.6 s and up to 5.8 s — about 2.5× the calibration baseline (0.608–0.850 s, calibrated against a ~3 MB and a ~0.9 GiB target). The bound therefore sits below the cadence the reader can deliver, and **268/288** (baseline) and **179/200** (candidate) mapped observations were refused. Zero scenario windows scored on both sides; the memory axis — half the pre-registered weighting — produced **no comparison at all, in either direction.** |
| **Citations** | Study §3.6 (instrument definition), §4.4 (coverage math and gap table), §4.4.1 (four windows that bypass the gate entirely and carry no coverage guarantee), §6.2 item 2, §7.2 item 1 (the named design fix). |
| **Refusal semantics** | The gate **fails safe**, symmetrically across both engines — it does not produce a wrong number, it refuses the window. This is deliberate and the study explicitly refused to fix it by re-deriving the cost bound upward after seeing its own run refused, calling that "confirmation bias with extra steps" (§4.4). |
| **Cross-arm instrument mismatch** | `peakPhysicalFootprintBytes` (the MLX-Swift-arm quantity, Mach-only) and `residentMemoryUpperBoundBytes` (the llama.cpp-arm quantity, Mach + mapped) are **different instruments and must never be compared across arms** — an `mmap`-loading runtime is invisible to the former (study §3.6, §4.6 reason 2). |
| **Adapter obligation** | Every memory figure an adapter emits must be tagged with which instrument produced it. An adapter **must not** narrow per-engine coverage requirements to make one engine's numbers look complete when the other's do not — the correct fix named by the study is at the instrument's design (read the mapped component in-process via `proc_pidinfo`/a `mach_vm_region` walk, collapsing two instruments into one cheap read at one cadence so the 7.0 s bound disappears rather than being retuned), not at the bound's calibration (§7.2 item 1). This fix is out of scope for this task; it is recorded here as a standing constraint on whichever task ports the memory instrument itself. |

### Knob 8 — Health and readiness semantics

| | |
| --- | --- |
| **What the harness expects** | `model-harness stress` polls `GET <endpoint>/models`; a Pi profile declares `readiness_path = "/models"` against a base URL of `.../v1`, so readiness is uniformly `GET /v1/models` answering `200`. |
| **Per-engine behaviour** | `llama-server`: `GET /v1/models` returns `503` before weights are resident, `200` once resident (measured 1.10 s and 1.12 s after launch on two runs), advertising **only the configured file**. `llama-server` additionally serves a real `GET /health` → `200 {"status":"ok"}`. `mlx-swift`: the same resident-gated shape as `llama-server` (readiness genuinely reflects loaded state). `mlx_lm.server`: `/v1/models` answers `200` about a second after launch **with no weights resident yet**, and lists **every model in the local cache**, not just the one being served. `mlx_lm.server`'s `/health` answers unconditionally regardless of true state — a known defect, `BUG-260827-1jhv2g` — and `mlx-swift` exposes no `/health` at all. |
| **Citations** | `.research/260828_llamacpp-under-the-managed-harness.md:112-134` (table and narrative); Knob 1's live-KV-report finding is the sharpest instance of the general problem this knob names. |
| **The distinction that matters** | "The endpoint answered 200" and "the pin this endpoint carries is attestable" are **different predicates**. The pinned `mlx_lm` fork's `/v1/models` can 200 while `meta.n_ctx` is absent (pre-cache-construction, Knob 1) — an operator or an adapter treating readiness as sufficient to read a `contextPolicy` term would derive `unbounded` from a runtime that is in fact bounded, one completion away from proving it. |
| **Adapter obligation** | Treat readiness (harness may route traffic) and attestability (a specific pinned term is currently `reported`) as separate gates. `llama-server`'s and `mlx-swift`'s resident-gated readiness happens to make this distinction moot for KV on those two engines (weights resident implies the bound is knowable); `mlx_lm`'s does not, and an adapter for it must poll again, or drive a warm-up completion, before trusting a `notReported` reading over an `unbounded` default. Do not read `mlx_lm.server`'s unconditional `/health` as a liveness signal for anything beyond "the process is scheduled" — it does not distinguish "loaded and serving" from "crashed and stuck." |

### Knob 9 — Weight artifact shape and model equivalence declaration

| | |
| --- | --- |
| **What it is** | Not a runtime knob — an adapter-boundary concern about what "the same model" means across a safetensors-serving engine and a GGUF-serving engine. |
| **The three declared, measured non-equivalences** | (1) The MLX 8-bit build **drops the MTP head** the GGUF carries as `blk.64` — 8 quantized tensors, 451,319,808 B on disk, skipped at load and logged as ignored by `llama-server`. (2) The **vision tower is placed differently** — 333 bf16 tensors inside the MLX safetensors shards versus a separate 931,145,984 B GGUF `mmproj` file — and is resident in neither on the default text path. (3) GGUF **upcasts norms and 1-D tensors to F32** where the MLX build keeps the source bf16: +10,686,464 B of extra resident memory on the GGUF side, with no fidelity difference. |
| **Citations** | Study §3.2 (equivalence declaration digest `106edbf4…f09f962`, verdict `comparable` **with** these three named exceptions). |
| **Refusal semantics** | An equivalence declaration is not a boolean gate — the study distinguishes `identical` (never claimed here), `comparable-with-N-declared-exceptions` (what was accepted), and `unrelated` (would refuse). A gate wired only to a content hash or artifact size would either over-refuse on these three known, accounted-for divergences, or — worse — silently ignore them and let the third one (+10.7 MB resident) corrupt exactly the memory comparison Knob 7 already struggles to deliver. |
| **Adapter obligation** | Carry a digest-pinned, machine-readable equivalence declaration per model pair, enumerating every known non-equivalence with its measured byte cost where one exists. Adding a new engine (e.g. MTPLX, `TASK-260830-18n40a`) or a new quantization requires a **new** declaration, not a reuse of an existing one across artifacts it was not measured against. |

### Knob 10 — Argv parsing precedence (cross-cutting)

| | |
| --- | --- |
| **What it is** | Not a value knob, and **not currently load-bearing for Knob 1** — §2 corrected Knob 1 to carry no argv fallback, so no live engine in this contract's scope (`python-mlx-lm`, `mlx-swift`, `llama-server`) has a case where this registry gates an admitted pin today. It is retained as a **forward-looking specification obligation**: for any future engine this contract does not yet cover that (a) has no live-report path for a pinned term at all and (b) has no `contextBoundNotHonoured`-equivalent refusal guard to fall back on, a per-engine parsing-precedence registry is the minimum discipline before that engine's argv could ever be trusted for anything — because different engines resolve duplicate or abbreviated flags differently, and an adapter that reads argv as literal tokens inherits whichever engine's parser it implicitly assumed. This must not be conflated with today's `mlx_lm` fork, which has a live-report path (Knob 1) and is not the engine this knob is scoped to protect. |
| **Measured per-engine behaviour** | `python-mlx-lm` (Python `argparse`): **last-wins** on exact-name repetition, **and** resolves unique-prefix abbreviations — `--prefill-step-siz` uniquely abbreviates `--prefill-step-size` and wins over an earlier, fully-spelled occurrence of the same option. `mlx-swift`: **rejects** duplicate flags outright rather than picking one. Anything the registry does not recognise (unknown flag, ambiguous abbreviation matching more than one option): **`unresolved`**. |
| **Citations** | `TASK-260830-2hc5r2` progress — revision 4 review (production `benchmark-run` accepted a duplicate kernel-observed prefill flag pair while pinning the wrong one — a bypass, not a hypothetical); revision 5 rework ("`RuntimeBenchmark` now decodes observed argv through a per-runtime registry: `python-mlx-lm` argparse last-wins; `mlx-swift` duplicate rejection; unknown/ambiguous unresolved"); revision 5 review (found the registry still ignored a *unique abbreviation* case, `--prefill-step-siz`, and pinned the wrong value — closed in the same revision cycle); study §1.2, §6.3 (the `--prefill-step-size 2048 --prefill-step-siz 999` example, reported as a production negative, not a specification claim). |
| **Adapter obligation** | This contract currently allows **no** argv fallback for any pinned term (§2) — the port task (`TASK-260830-1e9gse`) must not wire one for `python-mlx-lm`, `mlx-swift`, or `llama-server` on the strength of this knob. If a future engine addition genuinely meets both conditions in the row above — no live-report path and no `contextBoundNotHonoured`-equivalent refusal guard — and this document is revised to name that engine an exception, the read must go through a **per-engine precedence registry** that models last-wins vs. reject-on-duplicate vs. abbreviation resolution **as measured for that engine's actual parser**, not a generic "last flag wins" assumption, with `unresolved` refusing exactly like `unread` does for a live report. Until such a revision exists, this knob has no active fallback to gate. |

---

## 4. What downstream tasks must satisfy against this contract

- **`TASK-260830-1e9gse` (port three engines onto the plugin graph).** Each
  ported engine must produce, for Knobs 1–4 and 10: which states
  (`reported`/`notReported`/`unread`) it can actually reach on its current
  build, verified against a live fixture the way the study did (route
  enumeration from the binary's own string table or server sources, not a
  probe list) — not assumed from this document's description of the three
  engines already measured. A newly ported engine that turns out to report a
  term this contract currently marks structurally `notReported` for
  `llama-server` (e.g. a `llama-server` build that adds a live prefill-chunk
  endpoint) reopens Knob 2/3's admission refusal per study §4.1 option 1, and
  this document's table must be revised, not silently bypassed.
- **`TASK-260830-2cgim0` (consume engine kind in `model-harness` profiles).**
  Must expose engine kind as the discriminator this contract's per-knob tables
  are keyed on, and must not collapse Knob 4's speculation state into a
  boolean — the open-enum requirement exists specifically because this task's
  sibling, `TASK-260830-18n40a`, adds a third shape.
- **`TASK-260830-18n40a` (evaluate MTPLX as an MLX-native MTP engine).**
  Blocked on this task by design. Its speculation reporting must be classified
  against Knob 4's enum before any comparison against `llama-server` or the
  incumbent is attempted; a boolean "speculating: true/false" model inherited
  from `llama-server`'s `/slots` shape would misrepresent MTP's actual
  capability surface.

---

## 5. Constraints this contract does not relax

- No admission clause of the comparative gate (study §3.1's six conditions,
  the per-scenario cache refusal, the memory coverage refusal) is weakened by
  any knob above. Where a knob cannot be expressed, the documented consequence
  is refusal (pair-level or scenario-level, per knob) or an explicit
  `notReported`/declared-asymmetry — never a silent default and never a
  forced-through score.
- Knobs 1, 2, and 3's argv fallback is explicitly **forbidden**, not merely
  undocumented — an adapter implementation that adds one reopens either the
  study's central finding (Knobs 2/3) or `TASK-260830-2hc5r2`'s root-caused
  and closed production bug (Knob 1, `LOGBOOK.md:310`), and must be treated
  as a regression against this specification, not a convenience.
- Known false-friend endpoints (`/props default_generation_settings.params
  .reasoning_format` for Knob 3, `/props params["speculative.types"]` for
  Knob 4) must be enumerated in the adapter's own source as endpoints that
  exist, answer, and must **not** be read for the knob they resemble.

## 6. Reopening conditions

Each carried over from the study because they are the conditions under which
a knob's stated refusal could change:

1. Knobs 2 and 3 reopen if `llama-server` gains a live effective-configuration
   report for prefill chunk and reasoning effort (study §4.1 option 1, §7.2
   item 4) — the single upstream change that makes the pair scoreable without
   weakening this contract.
2. Knob 7 reopens once the memory instrument is redesigned to read the mapped
   component in-process (study §7.2 item 1) — re-calibrating the existing
   external-`vmmap` bound upward does **not** count as reopening it and must
   not be accepted as a substitute.
3. Knob 4's enum gains a confirmed fourth case once `TASK-260830-18n40a`
   measures MTPLX's actual reporting surface; until then `mtp` is a
   placeholder informed only by the GGUF/MLX MTP-head asymmetry in Knob 9.
