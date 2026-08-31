# Choosing a Local Qwen Runtime: Python mlx-lm, MLX Swift, and llama.cpp

**A three-runtime comparison study, its two refusals, and the migration decision they support.**

*Measurement campaigns 2026-08-28 (MLX Swift arm) and 2026-08-30 (llama.cpp arm). Written 2026-08-31.*

---

## Provenance — read this first

This paper is a **dated snapshot**. Both runtimes under evaluation are under active
development and will change; every measurement was taken once, against specific binaries,
on specific dates, on one host. The purpose of this section is that a reader a year from
now can tell exactly what was measured, on what, and when — and can distinguish a finding
that has expired from one that still holds.

| Item | Value |
|---|---|
| Measurement campaign — MLX Swift arm | 2026-08-28, 15:50:09 → 16:51:08 UTC. One `benchmark-run` invocation, both passes sequential. Source task `TASK-260827-2v13w8`, story `STORY-260827-m30k8z`. |
| Measurement campaign — llama.cpp arm | 2026-08-30, 18:14:11 → 19:13:06 UTC, 3,535 s wall. One `benchmark-run` invocation, both passes sequential. Source task `TASK-260829-3k4qrc` revision 4, story `STORY-260828-2faxgm`. |
| Host, both campaigns | `MacBookPro18,2`, Apple M1 Max, 68,719,476,736 B RAM, macOS build `25F80`, `arm64`. Pinned into every record as `hostIdentity` `MacBookPro18,2/68719476736/25F80/arm64`. |
| Incumbent — MLX Swift campaign | `mlx_lm.server` 0.32.0 at fork commit `9150698`, `mlx`/`mlx-metal` 0.32.2, CPython 3.14.7, `transformers` 5.16.1. Context policy `kv=unbounded;prefill-step=2048;reasoning=medium`. |
| Incumbent — llama.cpp campaign | `mlx_lm-kv76800-45a472f.server` — `mlx_lm` 0.32.0 at fork commit `45a472f2d0cda166b7ffe1a80fe50dd9621f4303`, `mlx`/`mlx-metal` 0.32.2. Context policy `kv=76800;prefill-step=2048;reasoning=medium`. **The two incumbent builds are not the same binary and not the same configuration; see §4.6.** |
| Candidate — MLX Swift | `mlx-swift` 0.31.6 (`0bb916c67f4b9e5c682cbe02a42c701c93ab5021`), `mlx-swift-lm` 3.31.4 (`bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57`), `swift-transformers` 1.3.3, served by the Release product of `tools/mlx-swift-runtime-prototype`. |
| Candidate — llama.cpp | `llama-server`, Homebrew `llama.cpp` 0.3.0, build **10621**, commit `c1d0e7a00`, self-reporting `b10621-c1d0e7a00`. Observed executable digest `07c17ec0…57232f6`. |
| Model of record | `hf:orcarouter/Qwen3.8-27B-Uncensored-BF16`. MLX arms serve `Qwen3.8-27B-Uncensored-MLX-8bit`, digest `1b10f3fe…88460b`, quantization `8bit/group64/affine`. llama.cpp serves `Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf`, digest `31756fca…24f8d6`, quantization `Q8_0`. Equivalence declaration digest `106edbf4…f09f962`, verdict `comparable` with **three declared non-equivalences** (§3.2). |
| Harness | `model-harness` `v1.6.1-44-gd91d6fc` (Swift arm) and `v1.6.1-103-g4270549` (llama.cpp arm). |
| Gate binary — llama.cpp arm | SHA-256 `36318014afae06bec693779ac3de6426c356ce678ed0882ff3aeb413a089d055`. One process launched, drove, sampled, sealed and judged both passes. |
| Gate binary — MLX Swift arm | SHA-256 `8a517b10e6a74793dd47d33d07b1b08275863f3fb7e8cfb11880a14b71014f91`. On that arm the same file was also the candidate runtime; its attestation's `observedExecutableDigest` equals its `gateBinaryDigest`. |
| Pinned inputs | prompt suite `bba8867f…6275d41`, thresholds `7490c454…4c6ac208`, llama.cpp-arm launcher config `bb1aee25…554df3b`, MLX-Swift-arm launcher config `d063af2e…82a36620`. |
| Repository state at publication | `f6da7e4` on `task-board/story/STORY-260828-2faxgm`, 2026-08-31T01:05:26+03:00. Trunk `main` at `bb857fe5`, 2026-08-30T17:47:59+03:00; the story branch is 8 commits behind trunk on paths this study does not measure. |
| Product code changed by this paper | **None.** This is a synthesis task: it re-runs no measurement and changes no configuration. The deployed `profiles.qwen-local` still points at the Python `mlx-lm` runtime. |
| Raw data | `artifacts/` in this directory, checksummed in `SHA256SUMS`. `artifacts/analysis/recompute.py` regenerates every figure this paper cites from the sealed records; `reproduce.zsh` runs it and diffs the result against `artifacts/analysis/expected-figures.json`. |
| Source documents | §9. Both arms were accepted only after adversarial review — **eleven rounds in total**: five recorded review verdicts on the MLX Swift arm and six on llama.cpp's corrected rerun. |

**Observation versus contract.** Where a runtime documents a behaviour, this paper cites the
document. Where it does not, the behaviour is reported as **observed on these binaries on
these dates**, and that is all it is. In particular, §4.1's finding that `llama-server`
serialises neither its effective prefill chunk nor its effective reasoning effort on any
route is a statement about build `b10621-c1d0e7a00`, established from that build's own
route table and server sources. A later build may change it, and if it does, §7's decision
has a named reopening condition waiting for exactly that.

**What expires and what does not.** The throughput and latency readings expire with the
binaries: re-measure them against any new llama.cpp build or `mlx-lm` revision. The
*structural* results are more durable — that the gate cannot certify this pair at all
(§4.1), that the memory instrument cannot cover a 26–45 GiB target at its calibrated
cadence (§4.4), and that one runtime's cache fired while the other's did not (§4.3.5) are
properties of the instrument and the runtimes' interfaces, not of a particular afternoon.

---

## Abstract

**The question.** Should the deployed local Qwen runtime — Python `mlx-lm` — be replaced by
MLX Swift or by llama.cpp?

**The method.** Two candidate arms were each measured against the incumbent by a single
process that launched both runtimes, drove all six pinned scenarios itself, sampled and
sealed what it measured, and then judged it, under a weighting fixed before the numbers
existed: peak resident memory and decode throughput carry **equal** weight, with time to
first token, 75,000-token capacity, tool-call parity, stability and migration risk weighed
alongside them.

**The decision: NO-GO on both candidates. Python `mlx-lm` remains the default local Qwen
runtime.**

MLX Swift was rejected on a scored, reproduced blocker: its 8k scenario peak footprint is
**1.151×** the incumbent's against a `≤ 1.10` bar, reproduced at 1.144× and 1.151× across
two independent full reruns. llama.cpp is rejected on a different and more interesting
basis. It is **faster on every axis this study could measure** — decode **+20.1 %** and
**+15.8 %** tokens per second on the two scenarios where decode is meaningfully measured,
TTFT at **0.629×** on the 8k prompt, wall clock 0.66–0.87× — and yet the comparison gate
**refused to score the pair, exit 4**, and wrote no decision, because `llama-server`
b10621 reports neither its effective prefill chunk nor its effective reasoning effort on
any live endpoint, so two of the three terms of the pinned `contextPolicy` come back
`not-reported`. Adopting it means giving up the ability to attest a running local
runtime's effective generation configuration — the exact discipline this project spent nine
review rounds building. Separately, **the memory axis produced no comparison at all**: the
mapped-file coverage bound is calibrated about 2.5× below what one `vmmap -summary` costs
against a 26–45 GiB target, so the gate refused memory on every scenario window and both
process-wide peaks of both runtimes. Half the pre-registered weighting is therefore absent,
and it is absent in a direction nobody can currently sign.

**No break-even exists in the measured direction.** The pre-registered crossover formula
requires one runtime to lead TTFT and the other to lead decode. llama.cpp leads both, in
every scenario that measures them, so the crossover length comes out negative
(−15.7 and −1,946.6 output tokens) and there is no positive response length at which the
incumbent catches up. That is reported as an absence, not manufactured into a number.

**This study also overturns two of its own earlier claims.** A previously reported
"llama.cpp is about 10 % slower at decode" was derived from server-sent-event *frame*
counts rather than tokens and is **withdrawn and overturned** — llama.cpp decodes faster.
A previously reported memory advantage for llama.cpp came from an instrument this programme
has since shown was defective, and is **withdrawn with no replacement in either direction**;
the figure is not restated anywhere in this paper.

---

## 1. Introduction

### 1.1 The problem

A local Qwen 27B runtime serves this workstation's agent tooling. The incumbent is Python
`mlx-lm` behind `model-harness`. Two replacements were proposed for different reasons: an
**MLX Swift** in-process runtime, to remove the Python layer and get a supervisable native
process; and **llama.cpp**, because it is the most widely deployed local inference server
and its GGUF Q8_0 weights were already available.

The programme's owner asked two questions, and this paper answers them in that order:
**is a candidate faster, and is it more frugal with memory?** — with the explicit
instruction that a winner on one axis alone is not a winner.

### 1.2 What makes this study unusual

Most runtime comparisons produce a table. This one produced **two refusals and one table**,
and the refusals are the more important output.

The comparison instrument does not accept a document about a measurement. It launches both
runtimes itself, drives every scenario over HTTP itself, samples memory itself, seals the
transcript, and only then judges. Its admission conditions are derived from each *running
process* — never from the launch argv, because argv is not what a process parsed
(`--prefill-step-size 2048 --prefill-step-siz 999` runs at 999 while argv reads 2048). That
discipline was built over **eleven** adversarial review rounds across the two arms — five
on the MLX Swift arm and six on llama.cpp's corrected rerun — three of which found a way to
make the gate return `accepted=true` on something that had never served a token.

The consequence is that **a runtime which will not report its own effective configuration
cannot be scored against one that will**. That is not a bug in this study. It is the study's
central finding about llama.cpp.

### 1.3 What this paper is not

It is not a quality or fidelity comparison: temperature is 0, seed is pinned, and no output
was scored for correctness beyond tool-call structure. It is not a multi-host result: one
M1 Max, one run per scenario per runtime. It is not a speculative-decoding result:
multi-token prediction was off on every pass. And it is not a llama.cpp-versus-MLX-Swift
comparison — §4.6 shows why that table cannot be built from this evidence and refuses to
build it.

---

## 2. Background

### 2.1 The three runtimes

| Runtime | What it is | Role here |
|---|---|---|
| **Python `mlx-lm`** | `mlx_lm.server` on Apple MLX, from a pinned internal fork, deployed behind `model-harness` as `profiles.qwen-local` | **Incumbent.** Measured *as deployed*, including its configured prompt cache. |
| **MLX Swift** | `mlx-swift` + `mlx-swift-lm` compiled into `tools/mlx-swift-runtime-prototype`, an in-process runtime that is also the benchmark gate | **Candidate A**, evaluated 2026-08-28. |
| **llama.cpp** | `llama-server` from Homebrew `llama.cpp` 0.3.0, serving GGUF Q8_0 | **Candidate B**, evaluated 2026-08-30. |

### 2.2 The pinned scenario suite

Six scenarios, one prompt suite digest, identical on every pass of both campaigns:

| Scenario | What it exercises | Prompt tokens |
|---|---|---:|
| `short_prompt` | cold small request; decode-dominated | 41 |
| `long_prompt_8k` | prefill-dominated request | 7,784 |
| `tool_call` | structured tool-call parity | 313 |
| `multiturn_prefix_reuse` | three turns over one shared prefix | 7,784 |
| `stability_soak` | 20 sequential iterations, 64-token budget each | 910 |
| `context_75k` | capacity probe near the pinned window | 73,016 |

The suite's 2,027 filler repeats render **73,016** tokens under this tokenizer, not 75,000.
That is the pinned value on every pass of both campaigns, and it is what "75,000-token
capacity" means throughout this paper.

### 2.3 The prior state of the evidence, and why this paper exists

Both arms reached an accepted result, but three claims made along the way did not survive
review, and a paper that hides its own corrections is worth less than one that shows them.
The withdrawals are listed in full in §4.7 and are not re-used anywhere in this text.

---

## 3. Method

### 3.1 The instrument

Both campaigns used the same production entry point: `benchmark-run`, a single invocation
that owns the whole comparison.

- It launches the baseline through `model-harness`, waits for readiness, drives all six
  scenarios over HTTP, samples memory throughout, and seals a transcript digest.
- It settles for 20 s, launches the candidate, and repeats.
- It writes one **record** per runtime (pins, revisions, per-scenario metrics, raw memory
  sample series, sealed transcript) and one **attestation** per runtime (what the gate
  observed of that process: kernel executable path and digest, process start time, live
  context window, live generation configuration, live speculation state).
- Only then does it **admit** the pair, and only an admitted pair is scored.

`benchmark-compare` exists as a **replay** and is structurally unable to return an
acceptance; its best outcome is a reproduced rejection, exit 3. That cap is the fix for the
forgery class review found three times on the MLX Swift arm.

**Admission conditions** — any one of these refuses the pair rather than scoring it:

1. the two passes' sealed wall-clock intervals must not overlap;
2. every scenario must carry the HTTP exchanges its numbers came from;
3. the record's claimed scenario wall clock must be consistent with the interval the gate
   actually watched the runtime for;
4. the pinned `contextPolicy` — KV window, prefill chunk, reasoning effort — must be equal
   on both sides, **each term derived from the live process, never from argv**;
5. prompt-token skew per scenario must be within `≤ 1.10`;
6. runtime-reported cached-token telemetry must not be one-sided.

**Exit codes.** 0 accepted, 2 usage, 3 rejected-on-score, **4 inadmissible** — refused
before scoring, no `decision.json` written.

### 3.2 Pinned conditions

Identical on both passes of the llama.cpp campaign, verified from the records rather than
from the launch scripts: `seed 1234`, `temperature 0`, `topP 1`, `maxOutputTokens 256`,
prompt suite digest `bba8867f…6275d41`, model of record
`source:hf:orcarouter/Qwen3.8-27B-Uncensored-BF16`, `hostIdentity`
`MacBookPro18,2/68719476736/25F80/arm64`, KV window `76800` on both, `speculation=off` on
both.

**Three declared non-equivalences travel with every number in this paper** and are
conclusions of the weight acquisition task, not of this one:

1. the MLX build **drops the MTP head** the GGUF carries as `blk.64` — 8 quantized tensors,
   451,319,808 B on disk, skipped at load and logged as ignored by `llama-server`;
2. the **vision tower is placed differently** — 333 bf16 tensors inside the MLX safetensors
   shards against a separate 931,145,984 B GGUF `mmproj` — and is resident in neither on
   the default text path;
3. GGUF **upcasts norms and 1-D tensors to F32** where the MLX build keeps the source
   bf16: 10,686,464 B of extra resident memory, no fidelity difference.

### 3.3 The pre-registered weighting

**This section was written before the corrected rerun produced its numbers, specifically so
that the weighting could not be fitted to the result.** It is reproduced here from the task
brief without alteration.

- **Peak resident memory and decode throughput carry equal weight.** Decode dominates on
  some workloads; memory dominates on a 64 GiB machine already serving other work.
- **Time to first token, 75,000-token capacity, tool-call parity, stability and migration
  risk are weighed alongside them**, not as tie-breakers.
- **The deliverable is the best overall compromise, not the winner on any single axis.**

### 3.4 The pre-registered break-even

With response length `L` in output tokens, total response time is

```
T(L) = TTFT + L / r          r = decode throughput, tokens per second
```

so the crossover between baseline `p` and candidate `l` is

```
L* = (TTFT_p − TTFT_l) / (1/r_l − 1/r_p)
```

**A crossover exists only when one runtime leads TTFT and the other leads decode.** If one
leads both, `L*` is non-positive, the candidate's curve lies below the baseline's at every
positive length, and the honest report is that **no break-even exists in the measured
direction**. That case was pre-registered as a possible outcome precisely so that finding it
would not be mistaken for an error, and it is what happened (§4.5).

### 3.5 The scoring bands

From the pinned `benchmark-thresholds.json`, digest `7490c454…4c6ac208`, unchanged across
both campaigns:

| Quantity | Band | Direction |
|---|---|---|
| time to first token | `≤ 1.10` candidate/baseline | lower is better |
| prefill throughput | `≥ 0.90` candidate/baseline | higher is better |
| decode throughput | `≥ 0.90` candidate/baseline | higher is better |
| peak footprint | `≤ 1.10` candidate/baseline | lower is better |
| prompt-token skew | `≤ 1.10` | parity check |
| scored scenarios | `short_prompt`, `long_prompt_8k` | |
| parity-success scenarios | `short_prompt`, `long_prompt_8k`, `context_75k`, `tool_call`, `stability_soak` | must succeed on both |

### 3.6 Units, and what is not a result

- **Decode and prefill are tokens per second; higher is better.**
- **TTFT and wall clock are seconds; lower is better.** Verified against the raw record
  field names `timeToFirstTokenSeconds` and `wallClockSeconds` before any figure in this
  paper was written; no latency figure here was taken from a rendered table.
- **Memory is bytes.** On the llama.cpp arm the scored quantity is
  `residentMemoryUpperBoundBytes` = Mach physical footprint + the upper edge of the
  `vmmap -summary` resident `mapped file` bucket. On the MLX Swift arm it is
  `peakPhysicalFootprintBytes`. **These are different instruments and are never compared
  across arms** (§4.6).
- Every ratio is **candidate ÷ baseline**.
- **Nothing in §4.3 is a scored comparison.** The gate refused the llama.cpp pair, so those
  are the two records' own readings printed side by side, under identical prompts, identical
  prompt-token counts, a pinned model, seed and temperature, with speculation off. They are
  reported because refusing to print them would hide a result rather than qualify it.

### 3.7 Reproducing the measurements

The llama.cpp campaign, verbatim from `artifacts/llamacpp-pair/run-rev4.sh`:

```bash
"$GATE" benchmark-run \
    --config  .../TASK-260829-3k4qrc-rev4.benchmark.toml \
    --model   /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
    --candidate-model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf \
    --equivalence equivalence/qwen3-8-27b-uncensored.equivalence.json \
    --prompts     examples/benchmark-prompts.json \
    --thresholds  examples/benchmark-thresholds.json \
    --session     .../session-rev4 \
    --harness     /Users/alexis/.local/bin/model-harness \
    --baseline-runtime  python-mlx-lm --baseline-profile  qwen-benchmark-python \
    --candidate-runtime llamacpp      --candidate-profile qwen-benchmark-llamacpp \
    --port 18341 \
    --python-bin /Users/alexis/.local/pipx/venvs/mlx-lm-kv76800-45a472f/bin/python \
    --baseline-declare  '...prompt cache...' \
    --candidate-declare '...per-slot KV reuse...' \
    --candidate-declare '...whole KV arena allocated at load...' \
    --startup-timeout 900 --request-timeout 10800 --settle-seconds 20
```

Launch argv for each side is in `artifacts/llamacpp-pair/benchmark.toml`; the MLX Swift
campaign's single command and config are in `artifacts/mlx-swift-pair/`. The MLX Swift
candidate must be built with `xcodebuild -configuration Release` — a `swift build` product
cannot compile mlx-swift's Metal shaders and refuses to serve by design.

**Every number cited below is regenerated from the sealed records by**
`artifacts/analysis/recompute.py`, **and `./reproduce.zsh` fails if any of them moves.**

---

## 4. Results

### 4.1 REFUSAL 1 — the gate would not score llama.cpp against the incumbent, exit 4

The full six-scenario suite ran end to end. All twelve scenario runs succeeded. Prompt-token
parity was **exact — ratio 1.0000 — on all six scenarios**, including the 73,016-token
capacity probe. And then the gate refused:

```
pinned condition "contextPolicy" differs:
  baseline  "kv=76800;prefill-step=2048;reasoning=medium"
  candidate "kv=76800;prefill-step=not-reported;reasoning=not-reported"
; these runs are not a comparison
```

**Driver exit 4. No `decision.json` was written.** Both records and both attestations exist,
because admission runs after the passes — which is the only reason this paper has numbers at
all.

**The KV term is fixed and stayed fixed.** Every earlier run in this programme refused on
`kv=unbounded` against `kv=<n>`, because the incumbent had no flag that bounded its KV cache
and `llama-server` has no unbounded mode. `STORY-260830-2vrhg1` delivered `--max-kv-size` in
the pinned `mlx-lm` fork, and this run confirms it end to end: **both records pin
`kv=76800`, derived by the gate from each running process's live `/v1/models` `meta.n_ctx`.**
That blocker is gone. The refusal moved to the other two terms of the same pin.

**What broke, and why it is not a configuration mistake.** The candidate's attestation
records it in the runtime's own terms:

```json
"observedGenerationConfiguration": {
  "prefillStepSize":  { "state": "notReported" },
  "reasoningEffort":  { "state": "notReported" }
}
```

against the baseline's

```json
"observedGenerationConfiguration": {
  "prefillStepSize":  { "state": "reported", "value": "2048" },
  "reasoningEffort":  { "state": "reported", "value": "medium" }
}
```

The process had been launched with `--ubatch-size 2048 --reasoning-effort medium`. It
parsed them. It just does not say so anywhere a caller can read.

Review established this from the pinned build itself rather than from a probe list —
enumerating the complete route set out of `libllama-server-impl.dylib`'s string table, 44
routes including seven the earlier probe never touched, and cross-checking against
llama.cpp server sources:

- **Prefill chunk.** `n_ubatch` and `n_batch` never appear as a JSON key in any server
  handler. Every occurrence in `tools/server/*.cpp` is an internal scheduling variable or an
  `SRV_WRN` log format string. There is no endpoint to miss.
- **Reasoning effort.** `reasoning_effort` exists only as an *inbound request* field, parsed
  into `chat_template_kwargs` and re-emitted into an outbound `chatcmpl_body`. It is never
  part of any server-state response. The `--reasoning-effort` launch flag populates a
  default that no route reports.
- **The false friend.** `/props` `default_generation_settings.params.reasoning_format` read
  `"none"` while the process ran under `--reasoning-effort medium`. A gate wired to it would
  carry a value that does not track the launch — the same `/props`-shaped trap this
  programme already documented for speculation, where `/props` reported `"none"` under
  `--spec-type ngram-mod` while `/slots` flipped to `true`.

**So the refusal is correct and irreducible on this build.** There are exactly three ways
out, and this study took none of them:

1. llama.cpp gains a live effective-configuration report, as the `mlx-lm` fork did for KV.
   This is the only route that produces a scored pair without weakening the gate.
2. The gate reads the two terms from argv for a runtime that cannot report them. This
   reopens the exact defect that the live derivation closed.
3. The decision accepts that this llama.cpp build can only be scored against a baseline that
   also declines to report — which on this host is nothing.

### 4.2 Candidate A — MLX Swift, scored and rejected

The MLX Swift arm *was* admitted and scored, because both runtimes reported the same
`contextPolicy` (`kv=unbounded;prefill-step=2048;reasoning=medium`; the incumbent had no KV
bound flag yet and the Swift prototype reported none either). Its verdict:
**`accepted=false`, one blocker.**

| Metric | Python | MLX Swift | Swift/Python | Band | Verdict |
|---|---:|---:|---:|---|---|
| `long_prompt_8k` scenario peak footprint | 33,705,153,440 B | 38,801,396,520 B | **1.1512×** | `≤ 1.10` | **outside** |
| `short_prompt` scenario peak footprint | 29,399,274,400 B | 29,116,781,352 B | 0.9904× | `≤ 1.10` | within |
| whole-process peak footprint | 48,534,787,000 B | 52,995,462,000 B | 1.0919× | `≤ 1.10` | within |
| `short_prompt` TTFT | 0.9849 s | 0.9307 s | 0.9450× | `≤ 1.10` | within |
| `short_prompt` prefill | 41.630 tok/s | 44.054 tok/s | 1.0582× | `≥ 0.90` | within |
| `short_prompt` decode | 10.271 tok/s | 10.897 tok/s | 1.0609× | `≥ 0.90` | within |
| `long_prompt_8k` TTFT | 92.9499 s | 92.9785 s | 1.0003× | `≤ 1.10` | within |
| `long_prompt_8k` prefill | 83.744 tok/s | 83.718 tok/s | 0.9997× | `≥ 0.90` | within |
| `long_prompt_8k` decode | 9.723 tok/s | 10.533 tok/s | 1.0833× | `≥ 0.90` | within |
| prompt tokens, both scored scenarios | 41 / 7,784 | 41 / 7,784 | 1.0000× | `≤ 1.10` | within |

**The blocker is the metric that repeated.** Two of revision 3's three blockers did not
reproduce and were withdrawn; the 8k footprint ratio reproduced at **1.144×** and
**1.151×** across two independent full reruns, on two different builds and weeks-apart
page-cache states, both times outside the bar.

Recorded but not scored, because capacity is a capability question both runtimes pass:

| `context_75k` | Python | MLX Swift | Swift/Python |
|---|---:|---:|---:|
| TTFT | 971.413 s | 1,504.453 s | 1.5493× |
| prefill | 75.165 tok/s | 48.533 tok/s | 0.6457× |
| wall clock | 973.305 s | 1,517.883 s | 1.5595× |
| scenario peak footprint | 48,534,787,000 B | 52,995,462,000 B | 1.0919× |

The Swift `context_75k` decode figure (1.117 tok/s) is **not cited as a decode result**: it
is 16 completion tokens measured at 49.4 GiB resident on a 64 GiB machine under a 1-minute
load average of 35.21 — memory pressure the scenario creates for itself.

**What the candidate won, reported because a rejection that hides them is an argument rather
than a finding:** `short_prompt` on every axis; 8k decode at 1.083× with TTFT and prefill at
parity; time to a *served* completion after launch 7.018 s against 10.971 s; soak 20/20 on
both sides.

**Concrete blockers to reopening MLX Swift**, all four of which are local work rather than
upstream work:

1. **8k scenario peak footprint 1.151×**, reproduced. Needs `≤ 1.10×`. This is the whole of
   the rejection.
2. **No prompt cache.** The prototype builds a fresh KV cache per request while the
   incumbent is measured as deployed with `--prompt-cache-size 1 --prompt-cache-bytes 8GB`.
   Declared as an asymmetry rather than tuned away.
3. **`context_75k` is 1.56× slower end to end.** Not a bar, but a caller would notice it.
4. **Throughput evidence is single-run.** Revision 3 demonstrated why: two of its three
   blockers vanished on rerun.

One contract limit carried forward and not fixed: an MLX error raised on an MLX-owned
`asyncEval` thread reaches MLX's global default handler and **traps the process**, so the
in-process 503 and batch-release recovery paths are reachable only for failures delivered as
throws.

### 4.3 Candidate B — llama.cpp, measured and refused

Everything in this section is **two runtimes' own readings, not a scored comparison**. The
gate refused the pair with exit 4 on `contextPolicy` (§4.1) and wrote no decision. They are
comparable in their inputs — identical prompts, exactly equal prompt-token counts, one model
of record, seed 1234, temperature 0, speculation off, sequential passes on a host holding no
other model — and *not* certified as a comparison by the instrument that took them.

#### 4.3.1 The passes did not overlap, and the host held one model

| Pass | Runtime | Start (unix) | Finish (unix) | Duration |
|---|---|---:|---:|---:|
| baseline | `python-mlx-lm` | 1788113670.052570 | 1788115795.126375 | 2,125.074 s |
| candidate | `llamacpp` | 1788115816.214009 | 1788117186.393213 | 1,370.179 s |

Separation **+21.088 s**; overlap **−21.088 s**. A positive overlap is an admission refusal
by name. `ps` sweeps for `llama-server`, `mlx_lm`, `mlx-swift` and `model-harness run` before
and after the invocation both printed `(none)`, and full raw process lists were captured on
both sides. Peak host load: baseline 13.995 inside `context_75k`, candidate 8.881 inside
`long_prompt_8k`. The workstation was not idle-locked and nothing was killed to make room.

#### 4.3.2 Prompt-token parity, before any latency number

| Scenario | python-mlx-lm | llamacpp | Skew | Verdict |
|---|---:|---:|---:|---|
| `short_prompt` | 41 | 41 | 1.0000 | comparable |
| `long_prompt_8k` | 7,784 | 7,784 | 1.0000 | comparable |
| `tool_call` | 313 | 313 | 1.0000 | comparable |
| `multiturn_prefix_reuse` | 7,784 | 7,784 | 1.0000 | comparable |
| `stability_soak` | 910 | 910 | 1.0000 | comparable |
| `context_75k` | **73,016** | **73,016** | 1.0000 | comparable |

#### 4.3.3 Latency and throughput

TTFT, prefill and decode share one generated-event definition on both runtimes: a streamed
delta counts when any of `content`, `reasoning` or `reasoning_content` carries a non-empty
string. That correction is what invalidated every earlier llama.cpp gate reading, and it is
why §4.7's withdrawal was necessary.

| Scenario | Metric | python-mlx-lm | llamacpp | cand/base |
|---|---|---:|---:|---:|
| `short_prompt` | TTFT s | 2.5314 | **2.1429** | 0.8465 |
| | prefill tok/s | 16.1969 | **19.1328** | 1.1813 |
| | **decode tok/s** | 6.7809 | **8.1450** | **1.2012** |
| | wall clock s | 10.4496 | **9.1055** | 0.8714 |
| `long_prompt_8k` | TTFT s | 107.2163 | **67.4273** | 0.6289 |
| | prefill tok/s | 72.6009 | **115.4428** | 1.5901 |
| | **decode tok/s** | 6.6705 | **7.7235** | **1.1579** |
| | wall clock s | 123.0636 | **81.1127** | 0.6591 |
| `tool_call` | wall clock s | 13.1792 | **11.0522** | 0.8386 |
| `context_75k` | TTFT s | 1,279.8919 | **950.8526** | 0.7429 |
| | prefill tok/s | 57.0486 | **76.7900** | 1.3460 |
| | wall clock s | 1,281.7795 | **952.5634** | 0.7432 |
| | decode tok/s | *not cited* † | *not cited* † | **withdrawn †** |

† **`context_75k` decode is not a decode result and is not published as one.**
`completionTokens` is **16 on both runtimes**, so the figure is 15 tokens over a ~1.9 s
(baseline) / ~1.7 s (candidate) tail following a 950–1,280 s prefill; a single scheduling
hiccup moves it several percent. The two values are in the sealed records and in
`artifacts/source-documents/`, so the withdrawal stays auditable, and they are deliberately
not printed here. **That scenario is cited for capacity, TTFT and prefill only.**

**Two scenarios are absent from this table on purpose.** `multiturn_prefix_reuse` and
`stability_soak` timings are **not speed results** — §4.3.5 shows they are non-comparable on
sealed cache telemetry, and printing llama.cpp's 0.726 s TTFT beside the baseline's
105.206 s would invite exactly the misreading the telemetry exists to prevent.

#### 4.3.4 What both runtimes passed

- **75,000-token capacity: met by both.** 73,016 prompt tokens, identical on both sides,
  both completed, neither aborted or degraded.
- **Tool-call parity: satisfied by both.** The check is not a status code: it requires
  `finish_reason == "tool_calls"`, a call naming `read_coolant_pressure` exactly, arguments
  that parse as JSON, and every key in the declaration's `required` list. Both satisfied all
  four at an identical 313-token prompt.
- **Stability: 20 of 20 iterations on both**, no failed iteration.
- **Speculation off on both.** The candidate's attestation reports `active: false`, read from
  its live `/slots` inside the observation window. The baseline's attestation records
  `notReported` — `mlx_lm.server` exposes no speculation state — and neither pass was
  launched with a draft model or any speculation flag, which is what the recorded
  `speculation=off` pin rests on for that side. **No `--spec-type draft-mtp` number exists
  in this study.**

#### 4.3.5 REFUSAL 2 — two scenarios are non-comparable on sealed cache telemetry

Every scenario sealed `usage.prompt_tokens_details.cached_tokens` from both runtimes.
Nothing was `unknown` and nothing was malformed.

| Scenario | baseline | candidate | Comparability |
|---|---|---|---|
| `short_prompt` | miss `[0]` | miss `[0]` | symmetric — scoreable |
| `long_prompt_8k` | miss `[0]` | miss `[0]` | symmetric — scoreable |
| `tool_call` | miss `[0]` | miss `[0]` | symmetric — scoreable |
| `multiturn_prefix_reuse` | miss `[0, 0, 0]` | **hit `[5736, 7780, 7809]`** | **one-sided — non-comparable** |
| `stability_soak` | miss `[0]` ×20 | **hit `[18]` ×20** | **one-sided — non-comparable** |
| `context_75k` | miss `[0]` | miss `[0]` | symmetric — scoreable |

Two things follow, both measured rather than assumed.

**`llama-server` holds a cross-request prefix cache and it survives across scenarios.** Its
first `multiturn_prefix_reuse` turn already found 5,736 of 7,784 prompt tokens cached, before
that scenario had sent anything — warmed by `long_prompt_8k` before it.

**The incumbent's configured `--prompt-cache-size 1 --prompt-cache-bytes 8GB` did not fire
once in this run.** `cached_tokens` is 0 on all six scenarios and all 26 turns. That reported
zero is corroborated by timing rather than trusted alone: the baseline spent 347.591 s over
three `multiturn_prefix_reuse` turns on one shared 7,784-token prefix — about three full
prefills at its measured 72.6 tok/s — and its third turn still paid 105.206 s to first token.
**The asymmetry the shipped configuration declares therefore runs in the opposite direction
from the way it is written**, and this run measures that rather than assuming it.

Had `contextPolicy` admitted the pair, these two scenarios would still have been refused
per-scenario rather than scored.

### 4.4 REFUSAL 3 — memory, the axis that produced no comparison

**This is half the pre-registered weighting and it is absent.** It belongs here and in the
decision section, not in a footnote.

The scored memory quantity on this arm is `residentMemoryUpperBoundBytes` = Mach physical
footprint + the upper edge of the `vmmap -summary` resident `mapped file` bucket. Both
components must independently prove sampling coverage before a window may be scored: a
125 ms bound for the in-process Mach series, and a 7.0 s bound for the mapped-file series.

| Window | baseline | candidate |
|---|---|---|
| `short_prompt` scenario | partial — `resident-mapped-file-sampling-gap` | **measured**, 34,731,153,644 B |
| `long_prompt_8k` scenario | partial — mapped gap | partial — mapped gap |
| `tool_call` scenario | partial — mapped gap | partial — mapped gap |
| `multiturn_prefix_reuse` scenario | partial — mapped gap | partial — mapped gap |
| `stability_soak` scenario | partial — mapped gap | partial — `mach-physical-footprint-sampling-gap` |
| **`context_75k` scenario** | **partial — Mach gap** | **partial — Mach gap** |
| whole-pass process peak | partial — Mach gap | partial — Mach gap |

**The single `measured` window among those the gate judged is on the candidate only, so no
scenario in this pair has a scored memory value on both sides.** `recompute.py` computes
that set and it comes back **empty**; `reproduce.zsh` fails if it ever stops being empty.
**The memory axis produced no comparison at all, in any direction.**

**Why, measured from the run's own raw series.** The mapped-file component is refreshed at
0.2 Hz by forking `/usr/bin/vmmap -summary`, so the interval between two mapped observations
is `5.0 s sleep + one read`. Inverting that over the whole-pass series:

| Runtime | Target | Mapped observations | Gap min / median / max | Derived `vmmap` cost | Gaps over the 7.0 s bound |
|---|---|---:|---:|---:|---:|
| `python-mlx-lm` | 27–45 GiB anonymous | 289 | 0.102 / 7.569 / 9.145 s | — / 2.569 / 4.145 s | **268 / 288** |
| `llamacpp` | 26.3 GiB mapped + 6–16 GiB anonymous | 201 | 0.508 / 7.221 / 10.823 s | — / 2.221 / 5.823 s | **179 / 200** |

The contract bound is `samplingIntervalSeconds + 2 × observedMappedFileReadCostSeconds`
= `5.0 + 2 × 1.0` = **7.0 s**, and its cost term was calibrated at 0.608–0.850 s against a
~3 MB and a ~0.9 GiB target. **Against a 26–45 GiB target the same read costs a median
2.2–2.6 s and up to 5.8 s** — roughly 2.5× the calibration — so the bound sits below the
cadence the reader can actually deliver on the processes this comparison measures, and the
window correctly refuses rather than scoring from a stale value.

The Mach component behaved as designed almost everywhere: across the four
non-soak, non-capacity scenario windows on both runtimes its maximum gap is 0.060–0.069 s,
well inside its 125 ms bound. It exceeded that bound only under load — **three times across
the baseline's whole-pass series** (two of them, 0.618 s and 0.273 s, inside `context_75k`
at host load 13.995) and **eight times across the candidate's**, at 0.13–0.29 s. Those
excursions are what refuse `context_75k` and both process-wide windows, and two of the
candidate's are what refuse its `stability_soak` scenario.

**The gate failed safe. It did not produce a wrong number.**

**The bound was deliberately not widened.** Re-deriving the cost term from the 28 GB targets
would raise it to roughly 16 s and would probably make these windows score. Doing that
*after* seeing one's own run refused, on data whose direction one already knows, is
confirmation bias with extra steps. The calibration gap is reported as a finding for review
to rule on.

#### 4.4.1 Four windows publish a score without ever facing the gate

The table above is the complete list of windows the coverage gate *judged*. It is not the
complete list of memory windows the run emitted. Four more come back `status: "measured"`,
`issues: []`, with a populated score:

| Window | Status | Score (B) | Mapped stamps | Mapped gaps over 7.0 s | Mach gaps over 125 ms |
|---|---|---:|---:|---:|---:|
| baseline `warmupMemory` | measured | 29,120,518,072 | 1 | — | — |
| baseline `soakMemory` | measured | 29,827,094,504 | 20 | **19 / 19** | **19 / 19** |
| candidate `warmupMemory` | measured | 34,248,152,988 | 1 | — | — |
| candidate `soakMemory` | measured | 44,346,176,540 | 20 | **19 / 19** | **19 / 19** |

They are `measured` because they never reach the gate: `BenchmarkPass.swift:100`
(`recordWarmupMemory`) and `:107` (`recordSoak`) construct them by direct
`RuntimeMemoryPeak(summarizing:)`, while the gate lives in
`BenchmarkFootprintSampler.coveredPeak` and is reachable only from `currentWindowPeak()`,
`processPeakSoFar()` and `capturePeaks()`. So a 15 s-gap series publishes with a score while
a 0.05 s series with one 0.62 s hiccup is refused.

**Bounded, so this is not over-read:** neither `soakMemory` nor `warmupMemory` appears
anywhere in `Sources/MLXSwiftRuntimeContract/`, so no scored comparison consumed them and the
pair's admission outcome does not depend on them. It is a reporting defect, not a scoring
one — and open instrumentation debt.

**Consequence:** anything quoted from those windows carries **no coverage guarantee**. That
includes the +6.2 GB soak observation in §4.4.3, which is stated with that provenance
attached or not at all.

#### 4.4.2 Numbers this paper will not print as memory results

Three figures exist in the record and are **deliberately not used**, because using them
would launder a refusal into a chart:

1. **The resident-memory advantage reported for llama.cpp at the 73,016-token prompt** in an
   earlier revision. It came from an instrument this programme has since shown was defective,
   so the figure is **withdrawn, with no replacement in either direction**, and is not
   restated here even to be argued against.
2. **The unscored peak components** at `context_75k`. Single instants from windows that failed
   coverage, so a higher transient may have gone unobserved. The source record labels them
   "not evidence"; this paper honours that and does not print them.
3. **Any statement that llama.cpp is more, less, or comparably frugal.** There is no measured
   basis for any of the three.

#### 4.4.3 One open observation, with its provenance attached

llama.cpp's resident upper bound rose **+6,201,119,136 B across the 20-iteration soak** while
the baseline's fell by 509,935,592 B. The entire climb is in the Mach anonymous component
(9,905,647,432 → 16,106,766,568 B, exactly +6,201,119,136 B) while the mapped component is
constant at 28,239,409,972 B on all twenty samples.

**This is not a leak, not a memory regression, and not a gated number.** It is a two-point
first-to-last delta off the `soakMemory` window — one of the four in §4.4.1 that never faced
the coverage gate, with 20 stamps at a ~10.9 s median cadence and 19 of 19 gaps outside both
bounds. It is the same shape as an anonymous climb an earlier task flagged, and this study
again does not establish what it is. It is recorded as an **open observation** because
something that would matter before a deployment should not be dropped merely because it
cannot be scored.

### 4.5 Break-even: there is none in the measured direction

Applying §3.4's pre-registered formula to the two scenarios where decode is meaningfully
measured:

| Scenario | TTFT_p − TTFT_l | 1/r_l − 1/r_p | L* | Crossover? |
|---|---:|---:|---:|---|
| `short_prompt` | +0.3884 s | −0.024698 s/tok | **−15.73 tokens** | **none** |
| `long_prompt_8k` | +39.7889 s | −0.020440 s/tok | **−1,946.62 tokens** | **none** |

**llama.cpp leads on both TTFT and decode in every scenario that measures them, so the two
response-time curves never cross at any positive response length.** `L*` is negative in both
cases; a negative crossover is not a response length and is not presented as one.

What that means concretely — total response time at four response lengths:

| Scenario | L | python-mlx-lm | llamacpp | ratio |
|---|---:|---:|---:|---:|
| `short_prompt` | 16 | 4.891 s | 4.107 s | 0.840 |
| | 64 | 11.970 s | 10.000 s | 0.836 |
| | 256 | 40.284 s | 33.573 s | 0.833 |
| | 1024 | 153.543 s | 127.864 s | 0.833 |
| `long_prompt_8k` | 16 | 109.615 s | 69.499 s | 0.634 |
| | 64 | 116.811 s | 75.714 s | 0.648 |
| | 256 | 145.594 s | 100.573 s | 0.691 |
| | 1024 | 260.729 s | 200.009 s | 0.767 |

The advantage *narrows* with length on the 8k prompt — from 0.634× to 0.767× — because the
prefill saving is a fixed 39.8 s while decode contributes a smaller proportional gain. It
narrows toward parity; it never reverses. **The pre-registered question "how many output
tokens make the decode penalty cancel the TTFT advantage" has no answer here, because there
is no decode penalty.** Reporting its absence is the answer.

### 4.6 All three runtimes, with non-comparable cells marked as such

This is the table the acceptance criteria ask for, and it is mostly refusals. Every cell
that is not a number says why.

| Criterion | Python `mlx-lm` (incumbent) | MLX Swift | llama.cpp |
|---|---|---|---|
| **Scored by the gate?** | reference | **yes** — `accepted=false`, exit 3 | **no** — **exit 4, inadmissible**, no `decision.json` (§4.1) |
| **Decode throughput** | reference | 1.061× / 1.083× (`short` / `8k`), **within band** | **1.201× / 1.158×**, reading only — not certified as a comparison |
| **Time to first token** | reference | 0.945× / 1.000× | **0.847× / 0.629× / 0.743×** (`short` / `8k` / `75k`), reading only |
| **Prefill throughput** | reference | 1.058× / 1.000× | **1.181× / 1.590× / 1.346×**, reading only |
| **Peak resident memory** | reference | **1.151× at 8k — the blocker**; 0.990× short; 1.092× process | **NO COMPARISON.** Refused on every scenario window and both process peaks of both runtimes; zero windows scored on both sides (§4.4) |
| **75,000-token capacity** | 73,016 tokens, completes | completes, **1.549× TTFT, 1.560× wall clock** | completes, **0.743× TTFT, 1.346× prefill**; decode not citable (16 completion tokens) |
| **Tool-call parity** | reference | pass, but renders 296 prompt tokens against 313 — a real template-serialisation difference, inside the skew band | pass at an **exactly equal** 313-token prompt |
| **Stability** | 20/20 | 20/20 | 20/20 |
| **Cross-request prompt cache** | configured `--prompt-cache-size 1 --prompt-cache-bytes 8GB`; **measured as not firing once** in the llama.cpp campaign | none — fresh KV cache per request, declared | **per-slot KV reuse, fires and survives across scenarios** — makes two scenarios non-comparable (§4.3.5) |
| **Attests its effective generation config?** | **yes** — KV window, prefill chunk and reasoning effort all reported live | KV window and prefill chunk reported (`kv=unbounded` era) | **NO** — prefill chunk and reasoning effort `notReported` on all 44 routes (§4.1) |
| **Concurrency** | as deployed | one generation at a time (`GenerationEngine` is an actor), declared | multi-slot, not exercised by this suite |
| **Error containment** | as deployed | **MLX errors on an `asyncEval` thread trap the process**; only thrown errors are recoverable | not characterised by this study |

**The MLX Swift column and the llama.cpp column may not be compared with each other.**
Four independent reasons, each sufficient on its own:

1. **Different incumbent.** The Swift campaign's baseline is `mlx-lm` at `9150698` with
   `kv=unbounded`; the llama.cpp campaign's is the `45a472f` fork with `kv=76800`.
2. **Different memory instrument.** `peakPhysicalFootprintBytes` against the
   mapped-file-inclusive resident upper bound. On an `mmap`-loading runtime the former cannot
   see the weights at all.
3. **Different dates on a host that was not idle-locked** — 2026-08-28 against 2026-08-30.
4. **The incumbent itself moved between them, and by a lot.** Recomputed from the two
   campaigns' own records:

| Scenario | Incumbent decode, Swift campaign | Incumbent decode, llama.cpp campaign | ratio |
|---|---:|---:|---:|
| `short_prompt` | 10.271 tok/s | 6.781 tok/s | **0.660** |
| `long_prompt_8k` | 9.723 tok/s | 6.670 tok/s | **0.686** |

**The same runtime, on the same host, serving the same model, decoded about 1.46–1.51×
faster two days earlier.** The configurations differ (`--max-kv-size 76800` and a different
fork commit) and this study does not establish which term is responsible. Whatever the cause,
it is far larger than either candidate's measured advantage, and it forbids reading any
number across the two campaigns.

### 4.7 What this study withdraws from its own earlier claims

| Claim | Where it came from | Status |
|---|---|---|
| "llama.cpp is about 10 % slower at decode" | Derived by an orchestrator from **server-sent-event frame counts rather than tokens**, before the shared generated-event definition landed | **Withdrawn and overturned.** llama.cpp decodes **faster** on both scenarios where decode is meaningfully measured: +20.1 % and +15.8 % tok/s. Frame-derived throughput is not resurrected anywhere in this paper. |
| "llama.cpp is more frugal at the 73k probe" | The pre-correction memory instrument | **Withdrawn.** No replacement in either direction, and the figure itself is not restated (§4.4.2). |
| `context_75k` decode | The corrected run's own table | **Withdrawn as a decode result** — 16 completion tokens on both sides. That scenario is cited for capacity, TTFT and prefill only, and the figures are not printed. |
| "exactly one failed read, against a process already torn down" | A misreading of `readFailureCount: 1` | **Withdrawn.** `memorySamplesReadFailed: 0` and `memorySamplesMalformed: 0` on both passes; the `1` is the synthetic coverage marker `coveredPeak` appends on refusal, present on every refused window and absent on the one that scored. No physical conclusion is drawn from it. |
| "memory was refused on every window" | An earlier revision of the source report | **Corrected**, not withdrawn: refused on every window **the gate judged**; four windows bypass the gate entirely (§4.4.1). |

The direction of the first two corrections is worth stating plainly: **the decode
correction moved the numbers toward llama.cpp, and the memory correction removed llama.cpp's
advantage.** Neither was retried, reordered or tuned; there was one full pair run in the
corrected revision and the numbers above are from it.

---

## 5. Threats to validity

Each threat names its **direction** — which runtime it favours — because a limitation whose
direction is unstated is decoration.

**T1 — Sampled peak, not proven peak. Direction: unknown, and that is the point.** Even the
one window that scored is a sampled maximum. A higher transient shorter than the sampling
interval is unobservable by construction, and the sampler says so in every emitted window.
This is why §4.4 refuses rather than scoring from a stale value, and why no memory number
here is called a peak.

**T2 — The mapped-file bound is calibrated ~2.5× too tight for these targets. Direction:
neutral between runtimes, adverse to the study.** It refused both sides symmetrically. It is
the reason the memory axis is absent, and re-deriving it after the fact was refused as
confirmation bias. Note that fixing it alone would **not** rescue `context_75k`, which
refuses on the *Mach* bound on both sides — and that bound broke under host load of 13.995
generated by the 73k prefill itself. You cannot calibrate that away without either making the
bound load-aware (weakening the instrument exactly where it protects you) or running the 75k
probe on a machine that is not running the 75k probe.

**T3 — Sealed cache asymmetries on both sides. Direction: favours llama.cpp where it fired,
and the two scenarios it fired in are excluded.** The incumbent was measured *as deployed*
with `--prompt-cache-size 1 --prompt-cache-bytes 8GB`; llama.cpp reuses per-slot KV. Measured
outcome: the incumbent's cache did not fire once, llama.cpp's fired in two scenarios
(`[5736, 7780, 7809]` and `[18]×20`). Those two scenarios are refused rather than scored.
**The four scenarios this paper draws timing conclusions from are symmetric misses on both
sides**, so the cache asymmetry does not contaminate the decode or TTFT result. Also note
what it means for the *incumbent's* numbers: a prompt cache that never fires is a
configuration the incumbent is paying for and not receiving, which flatters neither runtime
but does mean the incumbent's deployed configuration has an unexamined defect.

**T4 — Single host, single run per scenario. Direction: unknown, magnitude large.** The MLX
Swift arm demonstrated the cost of this directly: two of revision 3's three blockers vanished
on rerun, and single-run throughput on this host was assessed as worth about ±20 %. §4.6's
0.660×/0.686× incumbent drift between campaigns is the same phenomenon at a larger scale.
**llama.cpp's measured decode advantage (+15.8 % and +20.1 %) is of the same order as this
noise floor**, and this study does not claim it survives repeats. Its TTFT advantage on the
8k prompt (0.629×) and prefill advantage (1.590×) are well outside it.

**T5 — Speculation and MTP were off on both sides. Direction: removes a llama.cpp
capability from the comparison.** `--spec-type draft-mtp` exists on this build and the GGUF
carries the MTP head that the MLX weights drop. No with-MTP number was taken. A future
evaluation that turns it on is measuring a different thing than this one did, and should say
so.

**T6 — Declared model non-equivalences. Direction: mixed, all small.** Q8_0 against
8bit/group64/affine; the MTP head present in one artifact and skipped in the other; F32
upcast norms costing 10,686,464 B extra resident on the GGUF side. The equivalence
declaration's verdict is `comparable` **with** these three named exceptions, and no memory
figure in this paper is fine enough for the third to matter.

**T7 — Blocker B7: a launcher timeout orphaned processes holding about 31 GiB.** During an
earlier attempt in this campaign, a spawn run exceeding its launcher timeout was terminated
with exit 124 and left `model-harness` plus a Python child reparented to init, holding
roughly 31 GiB until external process-group signalling. It reproduced a second time from a
different run. **Direction: it did not touch the measurements in this paper** — the accepted
run's script terminates only its own process group, never matches by name, and its post-run
orphan check recorded `no orphan names this run's config`, with `ps` sweeps clean on both
sides. It is recorded because it is a live operational hazard on this host, tracked as
`BUG-260830-2950qe`, and because an orphan holding a listening port makes a rerun on the same
port fail with `EADDRINUSE`.

**T8 — Four memory windows bypass the coverage gate.** Described in §4.4.1. Bounded: nothing
in the scoring contract reads them, so no decision depends on them. **Direction: it would
inflate confidence in whichever side quoted them**, which is why the +6.2 GB observation
carries its provenance everywhere it appears and is not called a regression.

**T9 — The attestation asymmetry cuts slightly the other way too.** The baseline reports
`observedSpeculation: notReported` while the candidate reports it live from `/slots`. So on
that one term the incumbent is the runtime that will not say. It does not affect the
`contextPolicy` refusal, which is about prefill chunk and reasoning effort, and neither pass
was launched with any speculation flag — but a paper that names llama.cpp's attestation gap
should name the incumbent's too.

**T10 — This synthesis re-ran nothing.** No measurement in this paper was taken by its
author. Every figure is recomputed from sealed records by `artifacts/analysis/recompute.py`,
and the recomputation is what `reproduce.zsh` verifies. What that establishes is arithmetic
fidelity to the records, not independent replication of the runs.

---

## 6. Discussion

### 6.1 What the numbers show

On the axes that were measured, **llama.cpp is the faster runtime on this host, and not
marginally**. It reached first token 37 % sooner on an 8k prompt, prefilled 1.59× faster,
decoded 16–20 % faster, and finished the whole comparable workload in 0.66–0.87× the wall
clock. It matched the incumbent on capacity at 73,016 tokens with exact prompt-token parity,
matched it on strict tool-call structure at an exactly equal prompt, and matched it on
stability at 20/20.

MLX Swift is a narrower result: parity or a small win on latency and decode, one reproduced
memory blocker at 1.151× against a 1.10 bar, and a materially worse capacity profile
(1.56× wall clock at 73k).

### 6.2 What the numbers would need to show to justify migration

Under §3.3's weighting, a migration needs **four** things, and llama.cpp currently has one
and a half.

1. **A speed advantage that survives repeats.** It has the advantage; §5's T4 says it has not
   been shown to survive repeats. TTFT and prefill are comfortably outside the noise floor;
   decode is not.
2. **A memory result.** It has none, in either direction. This is not "llama.cpp probably
   wins on memory because it `mmap`s its weights" — that intuition is exactly what the
   withdrawn memory figure encoded, and the instrument that produced it was defective.
3. **Attestability.** It fails this outright, and the failure is structural rather than
   configurational (§4.1). This is the finding that most surprised the study.
4. **An explained memory profile under sustained load.** The +6.2 GB anonymous climb across
   a 20-iteration soak is unexplained. It is not a leak on this evidence; it is also not
   something to deploy past without knowing what it is.

### 6.3 The attestation result is the interesting one

This programme spent eleven adversarial review rounds building an instrument that trusts
only what a running process reports about itself, because three separate attacks showed that
anything else can be forged. The final form of that discipline is: **derive every pinned
condition from the live process, never from argv.**

llama.cpp cannot satisfy it. Not because of a missing flag — because the values are never
serialised. `n_ubatch` and `n_batch` exist only as internal scheduling variables and log
format strings; `reasoning_effort` exists only as an inbound request field. There is no
endpoint to add a reader for.

The migration consequence is concrete and outlives this study's timing numbers: **adopting
llama.cpp means the deployed local runtime can no longer attest its own effective generation
configuration.** An operator asking "what prefill chunk and reasoning effort is this process
actually running at" would get an answer from the launch script rather than from the process
— the precise substitution that this project has already been burned by, in the form of
`--prefill-step-size 2048 --prefill-step-siz 999` running at 999 while argv reads 2048.

It is worth being explicit that this is a **migration-risk finding, not a llama.cpp defect
report**. Nothing obliges an inference server to publish its effective batch geometry. It is
a mismatch between what this deployment needs to attest and what this runtime chooses to
expose, and it would be resolved by exactly one upstream addition.

### 6.4 The incumbent did not come out of this clean

Two findings are about the deployed runtime rather than the candidates, and both should
outlive this decision:

- **Its configured prompt cache did not fire once**, on any of six scenarios or 26 turns,
  corroborated by timing rather than trusted from a reported zero. The deployment is paying
  for a cache it is not getting.
- **Its own throughput moved by 1.46–1.51× between two campaigns two days apart**, on the
  same host and model. Whatever the cause — the new `--max-kv-size` bound, the fork commit,
  or host state — a 50 % swing in the reference is larger than either candidate's advantage
  and is not currently understood.

Neither is a reason to migrate. Both are reasons to be careful about how much any
single-run cross-runtime number is worth on this host, and the second is a concrete argument
for the repeats §7.2 asks for.

### 6.5 What would have to be true for the decision to flip

Stated in advance, so a future reader can check the decision against evidence rather than
against argument:

- **If memory later measures in llama.cpp's favour** and the attestation gap is closed, the
  case becomes strong: two equally-weighted axes plus TTFT, capacity and tool-call parity,
  against a migration cost. That is a GO.
- **If memory later measures in llama.cpp's favour but the attestation gap stays open**, the
  decision is a genuine trade — a materially faster and more frugal runtime against the loss
  of live configuration attestation — and it becomes a product decision about how much that
  attestation is worth, not a measurement question.
- **If memory later measures against llama.cpp**, the decision is settled and the NO-GO
  hardens: an equally-weighted axis lost, plus an adverse migration-risk axis, against a
  speed advantage of which only TTFT and prefill are outside the noise floor.

---

## 7. Conclusion — the decision

### 7.1 NO-GO on both candidates. Python `mlx-lm` remains the default local Qwen runtime.

**Judged against §3.3's pre-registered weighting — peak resident memory and decode
throughput equal, with TTFT, 75,000-token capacity, tool-call parity, stability and migration
risk weighed alongside them — and against §3.5's pinned bands.** This selects the best
overall compromise; it does not select the winner on any single axis, and llama.cpp *is* the
winner on the speed axis.

**MLX Swift — NO-GO, and retired as a candidate.** Scored, `accepted=false`, on a
reproduced blocker: `long_prompt_8k` peak footprint **1.1512×** against a `≤ 1.10` bar,
reproduced at 1.144× and 1.151× across two independent reruns on two builds. Its three
further blockers to reopening — no cross-request prompt cache, 1.56× slower at 73k, and
single-run throughput evidence — are all local work, none upstream. Its `asyncEval` trapping
behaviour is a contract limit that a supervised deployment would have to design around.

**llama.cpp — NO-GO for now, and explicitly not retired.** It is the faster runtime on every
axis this study could measure, and the decision does not rest on doubting that. It rests on
three things:

1. **The gate refused the pair, exit 4, and wrote no decision.** Two of the three
   `contextPolicy` terms come back `not-reported` because build `b10621-c1d0e7a00` serialises
   neither its effective prefill chunk nor its effective reasoning effort on any of its 44
   routes. Adopting it costs the ability to attest a running local runtime's effective
   generation configuration.
2. **An equally-weighted axis produced no comparison at all.** Memory was refused on every
   scenario window and both process-wide peaks of both runtimes; zero windows scored on both
   sides. Migrating on half a weighting, where the missing half is the half that historically
   carried the argument for llama.cpp *and* was the half whose earlier number had to be
   withdrawn, is not a decision — it is a preference.
3. **Of the measured advantage, only part is outside the noise floor.** TTFT (0.629× at 8k)
   and prefill (1.590×) are comfortably outside it. Decode (+15.8 %, +20.1 %) is of the same
   order as the ±20 % single-run variability this host demonstrated, and the incumbent's own
   1.46–1.51× drift between campaigns is larger than either.

**The honest label on this conclusion is: NO-GO on present evidence, with a strong candidate
and a named path to reopening.** It is not "llama.cpp lost". A hedge stated plainly is worth
more than a confident conclusion the data cannot carry.

### 7.2 What would settle it

In priority order, and each is a bounded piece of work:

1. **Fix the memory instrument at its design, not its calibration.** The scored quantity's
   mapped component takes 11 distinct values spanning 2.26–3.64 MB on the baseline — about
   0.01 % of a 29 GB footprint — and 3 distinct values on the candidate, effectively static
   once the GGUF is mapped. A 2.2–5.8 s external `vmmap` fork at 0.2 Hz, policed by a 7.0 s
   bound, exists to track a rounding error on one runtime and a constant on the other, and it
   is that fork which refuses 268/288 and 179/200 of the mapped observations. **Read the
   mapped component in-process** (`proc_pidinfo` / a `mach_vm_region` walk) and both
   components come from one cheap read at one cadence; the 7.0 s bound disappears rather than
   being retuned, and the Mach bound stops being a separate killer for `context_75k`. Merely
   re-deriving the cost term upward is **not** this fix and was refused as confirmation bias.
2. **Then re-run the pair for memory**, and only then.
3. **Repeat the scored scenarios.** Three runs minimum, so the decode delta can be separated
   from the ±20 % this host produces on its own.
4. **Raise the effective-configuration report with llama.cpp upstream** — a live report of
   effective `n_ubatch` and reasoning effort is the single change that makes this pair
   scoreable without weakening the gate.
5. **Explain the +6.2 GB anonymous soak climb** before any deployment decision, whichever way
   the rest goes.
6. **Investigate why the incumbent's configured prompt cache never fired**, and why its
   throughput moved 1.46–1.51× between campaigns. Both are incumbent findings that matter
   regardless of the migration.

### 7.3 The standing constraint

**The pinned `mlx-lm` fork is held until an accepted replacement carries its fixes.** The
bounded-KV work landed as `STORY-260830-2vrhg1` — the `--max-kv-size` flag that finally made
the incumbent's KV window pinnable and closed the refusal that blocked every earlier run in
this programme — is exactly such a fix, and it must not be stranded. Any future migration
plan has to carry it forward or demonstrate that the replacement makes it unnecessary.

No installed configuration was changed by this study: `profiles.qwen-local` still points at
the Python `mlx-lm` runtime.

---

## 8. Reproducing this paper

```bash
cd articles/260831_local-qwen-runtime-comparison-study
./reproduce.zsh
```

It verifies every retained artifact against `SHA256SUMS`, then regenerates every figure this
paper cites from the sealed records and diffs the result against
`artifacts/analysis/expected-figures.json`. A green run ends with
`PASS: local Qwen runtime comparison study reproduced`.

It verifies more than arithmetic. It re-asserts the structural claims this decision rests
on — that the driver exited 4 and wrote no decision, that the two passes did not overlap,
that prompt-token parity is exactly 1.0000 on all six scenarios, that **no memory window is
scored on both sides**, and that **no positive break-even exists** on either
decode-admissible scenario. If any of those stops holding, the run goes red and the
corresponding section of this paper is no longer supported by its own artifacts.

That gate was itself attacked rather than assumed: `artifacts/analysis/mutant-campaign.md`
records **13 narrowing mutants**, each falsifying one claim above and then regenerating both
`expected-figures.json` and `SHA256SUMS` from its own mutated artifacts so that only the
structural block can catch it. All 13 are caught, each by the check that names the claim it
attacks. Two survivors are reported there rather than hidden; the load-bearing one is that
replacing `recompute.py` with a stub that copies the expectation passes cleanly. **A green
run establishes that this paper's figures follow arithmetically from the sealed records as
shipped. It does not establish that a modified snapshot is honest**, and no attempt was made
to close that class, for the same reason the MLX Swift arm gave when it reported the same
boundary: every additional clause raises the cost of the identical attack without changing
its class.

`reproduce.zsh` does **not** re-run the benchmark. Re-running the measurement requires the
model weights, both runtimes, and about an hour of an otherwise-idle M1 Max; §3.7 gives the
exact command.

## 9. Source documents and further evidence

Retained in `artifacts/source-documents/`, and on the task board under their original names:

| Document | What it is |
|---|---|
| `TASK-260829-3k4qrc_measured-pair-outcome.md` | The llama.cpp campaign's full number set, three refusals and validation exits |
| `TASK-260829-3k4qrc_rev6-correction-note.md` | The two reported-fact corrections and their independent verification |
| `TASK-260829-3k4qrc_review-verdict-rev6.md` | Accepting review: whole-route enumeration from the pinned dylib, coverage arithmetic recomputed from raw stamps, and the constraints on what this article may claim |
| `TASK-260827-2v13w8_results.md` | The MLX Swift campaign's decision, mutant campaign and reproducibility analysis |
| `TASK-260827-2v13w8_review-verdict-rev4.md` | Accepting review of the MLX Swift arm |

Earlier programme research, in `.research/`: `260817_pi-local-model-launch-contract.md`,
`260825_pi-unattended-tool-authorization.md`, `260828_model-io-observability-through-harness.md`
(blocker B7's origin), `260828_llamacpp-under-the-managed-harness.md`,
`260828_llamacpp-in-the-benchmark-gate.md`, `260829_llamacpp-against-the-python-baseline.md`
(the provisional, superseded number set — historical, not a decision set).

Open items tracked on the board: `BUG-260830-2950qe` (launcher timeout orphans
`model-harness`); the `coveredPeak` bypass at `BenchmarkPass.swift:100` and `:107`; the
mapped-file sampler redesign in §7.2.1.

---

*This document is published in two places in this repository: as the checksummed article
snapshot `articles/260831_local-qwen-runtime-comparison-study/ARTICLE.md`, and as the dated
research paper `.research/260831_local-qwen-runtime-comparison-study.md`. The two are
byte-identical; the article directory is the one that carries the artifacts and the
reproduction script.*
