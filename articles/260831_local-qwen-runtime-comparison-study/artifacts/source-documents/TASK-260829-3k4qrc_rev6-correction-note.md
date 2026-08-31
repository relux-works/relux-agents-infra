# TASK-260829-3k4qrc — revision 6 (CR rev6): the two reported-fact corrections

Date: 2026-08-30
Role: developer
Scope: **documentation and reporting only.** No measurement code changed. No rerun.

The revision-5 review confirmed all four verification items and stated explicitly that
**no rerun is needed** — the measurements stand and both defects are documentation-only.
This revision fixes exactly those two findings plus the one "also fix" item, and changes
nothing else.

---

## Proof this revision is documentation-only

| check | result |
| --- | --- |
| `git diff --name-only 75b41cd2 <this tree>` | 3 files, all `.md`: `.research/260829_llamacpp-against-the-python-baseline.md`, `LOGBOOK.md`, `tools/mlx-swift-runtime-prototype/README.md` |
| `git diff --stat 75b41cd2 <this tree>` | `51 insertions, 17 deletions` across those 3 files |
| non-`.md` files in that diff | **none** (`grep -vE '\.md$'` exits 1) |
| `git diff --stat bafa676b <this tree> -- Sources Tests scripts examples` | **empty — byte-identical** to the tree rev4's validation evidence was gathered on |
| `git diff --stat 75b41cd2 <this tree> -- Sources Tests scripts examples` | **empty — byte-identical** to the reviewed rev5 tree |

The compilable tree has not moved since `bafa676b`. Revalidation was run anyway (below)
rather than inherited.

---

## Finding 1 — refusal accounting corrected, and the +6.2 GB number given its provenance

**Verified independently from `session-rev4/session.json` before writing anything.**
Four `RuntimeMemoryPeak` windows come back `status: "measured"`, `issues: []`, with a
populated `scoredBytes`, and are absent from the old refusal table:

| window | status | issues | `scoredBytes` | stamps |
| --- | --- | --- | ---: | ---: |
| baseline `warmupMemory` | measured | `[]` | 29,120,518,072 | 1 |
| baseline `soakMemory` | measured | `[]` | 29,827,094,504 | 20 |
| candidate `warmupMemory` | measured | `[]` | 34,248,152,988 | 1 |
| candidate `soakMemory` | measured | `[]` | 44,346,176,540 | 20 |

Coverage of those same windows, recomputed from their own raw stamps:

| window | mapped gap min / med / max | over 7.0 s | Mach max gap | over 125 ms |
| --- | ---: | ---: | ---: | ---: |
| baseline `soakMemory` | 13.862 / 14.682 / 15.117 s | **19 / 19** | 15.117 s | **19 / 19** |
| candidate `soakMemory` | 10.141 / 10.916 / 13.129 s | **19 / 19** | 13.129 s | **19 / 19** |
| both `warmupMemory` | single point — `coveredPeak` refuses `< 2` as `resident-memory-sampling-coverage-insufficient` | — | — | — |

Cause reproduced in the source: `BenchmarkPass.swift:100` (`recordWarmupMemory`) and
`BenchmarkPass.swift:107` (`recordSoak`) construct these by direct
`RuntimeMemoryPeak(summarizing:)`. The gate is `BenchmarkFootprintSampler.coveredPeak`
(`BenchmarkFootprintSampler.swift:333-345`), reachable only from `currentWindowPeak()`,
`processPeakSoFar()` and `capturePeaks()`.

Bound confirmed: `grep -rn "soakMemory\|warmupMemory" Sources/MLXSwiftRuntimeContract/`
returns nothing, so no scored comparison consumed them and no admission outcome depends
on them. Reporting defect, not a scoring one — stated that way in every document.

**The +6,201,119,136 B soak climb now travels with its provenance everywhere it appears.**
It is `candidate.soak.resident_memory_upper_bound_drift_bytes`, a two-point first-to-last
delta off the ungated `soakMemory` series. Verified further: the entire climb is the Mach
anonymous component (9,905,647,432 -> 16,106,766,568 B = exactly +6,201,119,136 B) while
the mapped component is constant at 28,239,409,972 B on all 20 samples. Every document now
says it is **not a leak and not a memory regression**.

## Finding 2 — the invented physical cause removed

`session.json` reports `memorySamplesReadFailed: 0` and `memorySamplesMalformed: 0` on
**both** passes. The `readFailureCount: 1` is the synthetic coverage marker `coveredPeak`
appends on refusal (`BenchmarkFootprintSampler.swift:344`). Confirmed exhaustively across
`records/*.json`: `readFailureCount == 1` on **all 26** `partial` windows and `0` on the
single `measured` one (candidate `short_prompt`). The sentence claiming "exactly one failed
read ... a process that had already been torn down" is replaced by what the counter
actually is, and **no physical conclusion is drawn from it**.

## Also fixed — `context_75k` decode withdrawn

`completionTokens` is **16 on both runtimes** (verified in both records). The decode figure
is struck in the §3.2 table and marked `withdrawn`, removed from the §3.3 "overturned" row
and from the six-sentence summary, and marked withdrawn in `LOGBOOK.md` and the research
banner. That scenario remains valid for **capacity, TTFT and prefill only**, stated in each
place. The struck values are retained in the table solely so the withdrawal is auditable.

## What was left exactly as it is

All measurements and raw series; the `contextPolicy` exit-4 refusal and its
irreducibility; decode and TTFT 8.145 vs 6.781 and 7.724 vs 6.671 tok/s; `long_prompt_8k`
TTFT at 0.629x; and the memory refusal as the honest answer for that axis.
