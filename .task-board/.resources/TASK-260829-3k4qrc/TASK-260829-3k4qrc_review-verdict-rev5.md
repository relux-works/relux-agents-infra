# TASK-260829-3k4qrc revision 5 review verdict

Verdict: **changes requested** — routed to `to-dev`. Two reported-fact defects in
the artifacts the article and the decision will be written from. **No rerun is
needed**; both fixes are documentation-only and the measurements stand.

Reviewed `CR-TASK-260829-3k4qrc-5` revision 5, base `3272e3ae`, candidate tree
`75b41cd2`. Attached patch reproduces SHA-256
`597e0e987399e8339ceb02ee5aa7ef58fd0839ff751bdb13d16e0a14bd2d4954`. The rev5
delta over the already-accepted rev4 tree `bafa676b` is **73 added lines across
3 documentation files** and nothing else — `git diff --stat bafa676b 75b41cd2`
= `.research/260829_…md` +26, `LOGBOOK.md` +24, `tools/mlx-swift-runtime-prototype/README.md`
+23. No source, no test, no script changed, so rev4's validation evidence covers
this tree exactly and I did not re-run the suite. I modified no repository file;
everything I wrote lives under gitignored `.temp/`.

---

## The four verification items — all four confirmed, independently

### 1. Is the `contextPolicy` refusal genuinely irreducible? **Yes. Reproduced with my own probe.**

I did not accept the producer's probe files. I launched `llama-server` myself on
the same build (`0.3.0 (build 10621, commit c1d0e7a00)`) and the same pinned
GGUF, under `--ubatch-size 2048 --batch-size 2048 --reasoning-effort medium
--reasoning on`, and added the one flag the producer's probe was missing:
`--metrics`.

| endpoint | status | carries prefill chunk? | carries effective reasoning effort? |
| --- | ---: | --- | --- |
| `/props` | 200 | no | no |
| `/slots` | 200 | no | no |
| `/metrics` | 200 | no — runtime counters only | no |
| `/v1/models` | 200 | no | no |
| `/models` | 200 | no | no |
| `/health` | 200 | no | no |
| `/v1/props`, `/api/show` | 404 | — | — |

Every `reason`-shaped key on the whole surface, exhaustively:
`reasoning_format` (`"none"` **under `--reasoning-effort medium`** — reproduced,
it does not track the launch), `reasoning_in_content`, and two
`chat_template_caps` booleans `supports_preserve_reasoning` /
`supports_reasoning_effort`. The last is the one false friend the producer's
report does not mention: it is a **capability of the chat template**, not the
effective effort in force, and a gate wired to it would carry a constant.
No `n_batch`/`n_ubatch` anywhere. The default-verbosity startup log prints
neither either.

**The producer's conclusion stands and is not reducible by argv, by a missed
endpoint, or by `--metrics`.** Probe bundle attached.

### 2. Are the readings trustworthy despite the refusal? **Yes — every number reproduces from the raw records.**

Recomputed from `session-rev4/records/*.json`, not from the report's tables:

- Sealed intervals: baseline `1788113670.052570 → 1788115795.126375` = 2125.074 s;
  candidate `1788115816.214009 → 1788117186.393213` = 1370.179 s; separation
  **+21.088 s**, overlap **−21.088 s**. No overlap.
- `speculation: "off"` in both records' pins. `seed 1234`, `temperature 0`,
  `topP 1`, `maxOutputTokens 256`, `promptSuiteDigest bba8867f…` identical on
  both sides.
- Prompt tokens: 41 / 7784 / 313 / 7784 / 910 / **73016**, exactly equal on both
  runtimes for all six scenarios.
- Decode 6.7809/8.1450, 6.6705/7.7235, 8.0269/8.7764 and TTFT 2.5314/2.1429,
  107.2163/67.4273, 1279.8919/950.8526 all reproduce to the printed digit.
- `contextPolicy` in the records is literally
  `kv=76800;prefill-step=2048;reasoning=medium` vs
  `kv=76800;prefill-step=not-reported;reasoning=not-reported`. Driver exit **4**,
  and `find session-rev4 -name 'decision*'` returns nothing — no `decision.json`
  was written. The refused state did not corrupt anything: admission runs after
  the passes, both records and both attestations exist.

### 3. The memory cause. **Confirmed exactly, from the run's own raw series.**

Inverting the mapped observation stamps myself:

| runtime | mapped observations | gap min / median / max | derived cost (gap − 5.0 s) median / max | gaps over 7.0 s |
| --- | ---: | ---: | ---: | ---: |
| `python-mlx-lm` | 289 | 0.102 / **7.569** / 9.145 s | 2.569 / 4.145 s | **268 / 288** |
| `llamacpp` | 201 | 0.508 / **7.221** / 10.823 s | 2.221 / 5.823 s | **179 / 200** |

Mach series: baseline max gap 0.6179 s with 3 gaps over 125 ms; candidate max
0.2919 s with 8 over. Every figure in §4.1 of the report is correct.

### 4. `context_75k` decode. **Confirmed unreliable. Drop it from the article entirely.**

`completionTokens` is **16 on both runtimes**. The figure is 15 tokens over a
~1.9 s (baseline) / ~1.7 s (candidate) tail after a 950–1280 s prefill. A single
scheduling hiccup moves it several percent. The report footnotes it; a footnote
is not enough for a number that will be read as "+9.3 % at 75k". It should not
appear in the article as a decode result at all — the 75k scenario measured
capacity, TTFT and prefill, and those three are what it should be cited for.

---

## Finding 1 — BLOCKING. "Every memory window was refused except one" is false against the run's own output

`.research/260829_…md`, `LOGBOOK.md` and `tools/mlx-swift-runtime-prototype/README.md`
— all three repo-persisted by this revision — assert:

> "memory refused on every window of both runtimes except `short_prompt` on the candidate"
> "**every memory window of that pair was refused except one**"

The run's own `session-rev4/session.json` emits **four more** `RuntimeMemoryPeak`
windows, and all four are `status: "measured"`, `issues: []`, with a populated
`scoredBytes`:

| window | status | scoredBytes |
| --- | --- | ---: |
| baseline `warmupMemory` | measured | 29,120,518,072 |
| baseline `soakMemory` | measured | 29,827,094,504 |
| candidate `warmupMemory` | measured | 34,248,152,988 |
| candidate `soakMemory` | measured | 44,346,176,540 |

They are `measured` because they **never face the coverage gate**.
`BenchmarkPass.swift:100` and `BenchmarkPass.swift:107` build them by direct
`RuntimeMemoryPeak(summarizing:)` construction. The gate lives in
`BenchmarkFootprintSampler.coveredPeak` (`BenchmarkFootprintSampler.swift:333-345`)
and is reached only from `currentWindowPeak()`, `processPeakSoFar()` and
`capturePeaks()`. `RuntimeMemoryPeak.init(summarizing:)`
(`RuntimeMemoryAccounting.swift:272-317`) sets `.measured` and fills `scoredBytes`
on "all reads complete" alone, with no coverage judgement, and
`validatedScoredBytes` then returns a value for them.

Measured on this run's own stamps, those four windows are the **worst-covered
series in the whole pair**:

| window | mapped stamps | mapped gap min/med/max | over the 7.0 s bound | Mach max gap | over the 125 ms bound |
| --- | ---: | ---: | ---: | ---: | ---: |
| baseline `soakMemory` | 20 | 13.862 / 14.682 / 15.117 s | **19 / 19** | 15.117 s | **19 / 19** |
| candidate `soakMemory` | 20 | 10.141 / 10.916 / 13.129 s | **19 / 19** | 13.129 s | **19 / 19** |
| both `warmupMemory` | 1 | — | — | — | — (one sample; the gate refuses `< 2` as `resident-memory-sampling-coverage-insufficient`) |

So a 15 s-gap series is published as `measured` with a score, while a 0.05 s
series with one 0.62 s hiccup is refused as `partial`. This is the bypass the
revision-4 verdict logged as non-blocking with the exact words *"it becomes a
trap the moment anything scores them."* **This revision is that moment**: §6 of
the report, and the `OPEN:` bullet this CR writes into `LOGBOOK.md`, quote
**+6,201,119,136 B** — which is `session.json → candidate.soak.resident_memory_upper_bound_drift_bytes`,
the first-to-last delta of the ungated candidate `soakMemory` series — beside a
document that everywhere else insists no memory number survived coverage. A
reader cannot tell that this one never faced the gate.

Bounded severity, stated so this is not over-read: `RuntimeBenchmark.decide`
references neither `soakMemory` nor `warmupMemory` — they appear nowhere in
`Sources/MLXSwiftRuntimeContract/` — so **no scored comparison consumed them**
and the pair's admission outcome is unaffected. This is a reporting defect, not
a scoring one. It is still blocking because the whole authority of this report
is that its refusals are exhaustive.

**Fix (docs only):** in all three repo files and in the board report, scope the
claim to the windows it is true of ("every scenario window and both process-wide
peaks"), add `warmupMemory`/`soakMemory` to the §4 table with their real status
and the note that they are constructed outside `coveredPeak` and carry no
coverage judgement, and attach that provenance to the +6.2 GB soak number
wherever it appears.

## Finding 2 — BLOCKING. The report invents a physical cause for an instrument artifact

Report §4.1:

> "Each pass also recorded exactly **one failed read**, counted as a read failure
> and not folded into absence or into malformed data — the final sample against a
> process that had already been torn down."

No read failed in either pass. `session.json` says
`baseline.memorySamplesReadFailed: 0` and `candidate.memorySamplesReadFailed: 0`.

The `readFailureCount: 1` the producer saw is the **synthetic coverage marker**
`coveredPeak` appends when the gate refuses: `RuntimeMemoryPeak(summarizing: reads + [.readFailed(issue)])`
(`BenchmarkFootprintSampler.swift:344`). Proof from the records: `readFailureCount`
is `1` on **every** window whose status is `partial` — all 6 baseline scenarios,
5 candidate scenarios, both process-wide peaks — and `0` on the single window
that came back `measured` (candidate `short_prompt`). It tracks the refusal, not
a read.

This is a fact reported as observed that was inferred from a proxy signal and
given a causal story ("a process that had already been torn down") that nothing
measured. In a story whose stated standard is *"an absence and a failure to read
are different facts"* and *"prove, or report nothing"*, this is the shape that
has to come out of the artifact before it becomes an article.

**Fix (docs only):** delete the sentence or replace it with what the counter
actually is.

---

## The judgement you asked for: **(b) — proceed to the article. Do not spend the second pair.**

Not because two hours is expensive. Because **option (a) as scoped would not
produce the number the decision needs**, and the run already proves it.

Re-deriving `observedMappedFileReadCostSeconds` against a real target would put
the bound near 16 s, which clears every mapped gap observed here (worst was
10.823 s). That fixes the **mapped** refusal. But look at which refusal each
window that matters actually hit:

| window | baseline issue | candidate issue |
| --- | --- | --- |
| `context_75k` — **the window the decision needs** | `mach-physical-footprint-sampling-gap` | `mach-physical-footprint-sampling-gap` |
| whole-pass process peak | `mach-physical-footprint-sampling-gap` | `mach-physical-footprint-sampling-gap` |
| `stability_soak` | `resident-mapped-file-sampling-gap` | `mach-physical-footprint-sampling-gap` |

`context_75k` refuses on **both sides** for the *Mach* bound, not the mapped one.
A self-calibrating mapped bound does not touch it. The Mach series blew its
125 ms bound because the host load average hit **13.995** — and that load is
generated by the benchmark itself, a 73k-token prefill pinning an M1 Max for
21 minutes. You cannot calibrate that away; you would have to either make the
bound load-aware, which is weakening the instrument in the one place it is
actually protecting you, or run the 75k probe on a machine that is not doing the
75k probe.

So (a) buys, at best, memory on the short scenarios — the ones where memory is
least interesting — and leaves the 75k memory comparison refused exactly as it
is now. That is a bad trade for two hours plus two more agent cycles.

**What would make me confident a second pair scores** — and this is the honest
scoping if you ever do want the memory axis, so you can judge it as the real
piece of work it is rather than a constant retune: stop forking `/usr/bin/vmmap`.
The 2.2–5.8 s cost and the whole mapped/Mach cadence split exist only because
the mapped component is read by an external process. Read it in-process
(`proc_pidinfo` / a `mach_vm_region` walk) and both components come from one
cheap read at one cadence, the 7.0 s bound disappears rather than being retuned,
and the Mach bound stops being the thing that separately kills `context_75k`.
That is an instrumentation task with a real design, not a calibration round, and
it is the only version of (a) I would back.

### What the article must not claim

1. **No memory comparison, in any direction, hedged or not.** Not "llama.cpp uses
   less", not "comparable", and specifically **not the revision-1 −5.64 % figure** —
   that came from the instrument this story has since shown was defective.
2. **Not the §4.3 unscored components** (48,433,633,280 B vs 44,401,947,700 B).
   Those are single instants from windows that failed coverage; the report labels
   them "not evidence" and the article must not launder them into a chart.
3. **Not the +6.2 GB soak climb as a leak or a memory regression.** It is a
   two-point drift off a window that never faced the coverage gate (Finding 1).
   Either state it with that provenance as an open observation, or leave it out.
4. **Nothing in §3 as a scored comparison.** The gate refused the pair, exit 4,
   no `decision.json`. They are each runtime's own readings under identical
   prompts, identical prompt-token counts, and a pinned model/seed/temperature —
   say exactly that, and say the gate refused, in the same breath as the numbers.
5. **Not `context_75k` decode** (item 4 above). Cite that scenario for capacity,
   TTFT and prefill only.
6. **Not `multiturn_prefix_reuse` or `stability_soak` timings as speed results.**
   Sealed `cached_tokens` show llama.cpp hit (`[5736, 7780, 7809]`, `[18]×20`) and
   the baseline miss (`[0]`) — one-sided cache, verified in the records.
7. **Must state that an equally-weighted axis is absent, and why**, in the
   decision section rather than a footnote: the mapped-file coverage bound is
   calibrated a factor of ~2.5 below what a `vmmap -summary` costs against a
   26–45 GiB target, and the Mach bound cannot hold 125 ms under the host load
   the 73k prefill itself generates. The gate failed safe; it did not produce a
   wrong number. But the decision is being made on the remaining criteria, and
   that has to be visible in the text, not discoverable in an appendix.

The decode result itself is solid and is the article's headline: on one shared
generated-event definition, at exact prompt-token parity, with MTP off and
speculation off on both sides, llama.cpp decoded faster on both scenarios where
decode is meaningfully measured — 8.145 vs 6.781 tok/s and 7.724 vs 6.671 tok/s —
and `long_prompt_8k` TTFT reproduced revision 1 at 0.629x. The withdrawn "about
10 % slower" claim is overturned. Say it as two runtimes' own readings that the
gate would not certify as a comparison, and the article is honest and still has
its result.

---

## Definition of Done status

- Both runtimes sequential on a host holding no other model, stated — **met**,
  intervals and `ps` sweeps verified.
- MTP off for every scored comparison — **met**, `speculation: "off"` in both records.
- Non-comparable dimensions refused, every refusal reported — **not met**, Finding 1:
  four `measured` ungated windows are omitted from the refusal accounting.
- Corrected decode/TTFT reproduce or overturn the provisional deficit — **met**.
- Corrected memory reproduces or overturns the 9 % advantage — **met as a refusal**;
  the axis produced no comparison, and that is correctly the answer.
- Tests / lint / build — **met**, carried from rev4 on a byte-identical source tree.
- Findings recorded in logbook — **met in form**, but inherits Finding 1's inaccuracy.
- Gate behaviour attacked, not read — **done by me**: live `llama-server` probe
  including `--metrics`, coverage arithmetic recomputed from raw stamps, and the
  `coveredPeak` bypass traced to its production construction sites.
