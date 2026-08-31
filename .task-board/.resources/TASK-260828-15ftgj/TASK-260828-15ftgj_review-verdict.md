# TASK-260828-15ftgj — review verdict: ACCEPTED

**Run:** `RUN-260830-3d329c` (reviewer archetype). **Change Request:** `CR-TASK-260828-15ftgj-1` revision 1.
**Base** `3f313d91` → **candidate tree** `e3fe13b6`. Reviewed 2026-08-31.
**Verdict: ACCEPTED.** Five findings recorded below; none changes a number that carries a
comparison, and none changes the decision. Recorded as follow-ups, not as rework.

**Scope actually delivered by this task** (worktree working tree; the other 130 paths in the
CR delta are already-accepted story-branch work that the CR base predates):
`.research/260831_...md`, `articles/260831_local-qwen-runtime-comparison-study/**` (41 files),
`.research/260829_...md` banner, `LOGBOOK.md`, `README.md`. `git status` confirms **no product
code in this delta** — the article's "Product code changed by this paper: None" holds.

---

## 1. How this was reviewed

The article's own verifier was **not** trusted as evidence. Every figure below was re-extracted
from the sealed records with an independent script, bypassing `recompute.py` entirely, and the
verifier was then attacked with three fresh narrowing mutants that the shipped campaign does
not contain.

I did not re-run the pair (per brief).

## 2. Independently re-derived from the raw artifacts — all exact

Extracted straight from `records/*.json.gz`, `session.json`, `attest/*.json`, `run-rev4.log`
and the Swift `Sources/`, with no use of `recompute.py`:

| Claim | Article | Independently re-derived | |
|---|---|---|---|
| §4.3.3 `short_prompt` TTFT / prefill / decode / wall | 2.5314→2.1429 (0.8465), 16.1969→19.1328 (1.1813), 6.7809→8.1450 (1.2012), 10.4496→9.1055 (0.8714) | identical to 4 dp | ✓ |
| §4.3.3 `long_prompt_8k` same four | 107.2163→67.4273 (0.6289), 72.6009→115.4428 (1.5901), 6.6705→7.7235 (1.1579), 123.0636→81.1127 (0.6591) | identical | ✓ |
| §4.3.3 `context_75k` TTFT / prefill / wall | 1,279.8919→950.8526 (0.7429), 57.0486→76.7900 (1.3460), 0.7432 | identical | ✓ |
| §4.3.2 prompt-token skew, all six | 1.0000 ×6 incl. 73,016 | 41/7784/313/7784/910/73016 both sides | ✓ |
| §4.3.1 pass separation | +21.088 s | 1788115816.214009 − 1788115795.126375 = 21.0876 | ✓ |
| §4.1 refusal text and exit | exit 4, `prefill-step=not-reported;reasoning=not-reported` | verbatim final line of `run-rev4.log`; `run-rev4.exit` = 4; pins confirm | ✓ |
| §4.1 attestations | candidate `notReported` ×2, baseline `reported` 2048/medium | exact in both attestation files | ✓ |
| §4.3.4 speculation | candidate `active:false` reported, baseline `notReported` | exact | ✓ |
| §4.3.5 cache telemetry | base miss ×6 (0 on all 26 turns); cand hit `[5736,7780,7809]`, `[18]×20` | exact, `state` fields included | ✓ |
| §4.4 window table | 13 of 14 partial; one candidate `short_prompt` measured @ 34,731,153,644 B | exact incl. per-window `issues` strings | ✓ |
| §4.4 gap arithmetic | 289 obs / 268-of-288 over bound / 0.102-7.569-9.145 s; 201 / 179-of-200 / 0.508-7.221-10.823 s | recomputed from 36,740 and 23,942 raw stamps — **identical to 3 dp** | ✓ |
| §4.4 Mach excursions | baseline 3 (0.618, 0.273), candidate 8 @ 0.13–0.29 s | 3 → [0.618, 0.273, 0.234]; 8 → [0.292 … 0.136] | ✓ |
| §4.4.1 bypass | `BenchmarkPass.swift` :100/:107 direct `RuntimeMemoryPeak(summarizing:)`; `coveredPeak` private, 3 callers; neither name in `Sources/MLXSwiftRuntimeContract/` | confirmed in source; grep for both names in Contract returns **empty** | ✓ |
| §4.4.3 soak | cand +6,201,119,136 B, base −509,935,592 B, mapped constant 28,239,409,972 B | exact in `session.json` | ✓ |
| §4.2 full MLX Swift table + 1.1512 blocker | 10 rows | every cell exact; `decision.json` `accepted:false`, exactly one blocker naming `long_prompt_8k/peak_physical_footprint_bytes … 1.1512007084931994` | ✓ |
| §4.2 admission asymmetry | both Swift-arm pins `kv=unbounded;prefill-step=2048;reasoning=medium` | identical on both sides — which is *why* that arm scored and this one did not | ✓ |
| §4.5 break-even | −15.73 / −1,946.62, deltas −0.024698 / −0.020440, +0.3884 / +39.7889 s | exact; `exists:false` on both | ✓ |
| §4.6 incumbent drift | 0.660× / 0.686× → "1.46–1.51× faster two days earlier" | 0.660197 / 0.686033 → 1.515 / 1.458 | ✓ |
| §7.2 mapped component | 11 distinct values 2.26–3.64 MB baseline, 3 candidate | 11 values, 2,262,016 → 3,638,272 B; candidate 3 | ✓ |
| Provenance | branch 8 commits behind trunk | `git rev-list --count HEAD..main` = 8 | ✓ |

`shasum -c SHA256SUMS` → **41/41 OK**. `./reproduce.zsh` → green in 1.1 s.
`ARTICLE.md` and `.research/260831_...md` are **byte-identical** (`cmp`), as claimed.

**Numeric sweep.** All 298 numeric tokens in `ARTICLE.md` were matched against
`expected-figures.json` and the raw artifacts. **297 trace. One does not** — finding F1.

## 3. Brief constraints — each checked, each satisfied

- **No memory comparison in any direction; −5.64 % absent.** `grep` for `5.64` in `ARTICLE.md`
  returns nothing. §4.4.2 refuses all three memory statements by name. ✓
- **§3 numbers not presented as a scored comparison; exit 4 stated alongside them.** §3.6's last
  bullet and §4.3's opening paragraph both say it; the refusal is §4.1, a top-level result
  section, not a footnote. ✓
- **`context_75k` decode not cited.** The llama.cpp values **exist in the record** (8.0269 vs
  8.7764, ratio 1.0934×) and are **not printed anywhere in the article** — I checked for the
  literals. Cited for capacity/TTFT/prefill only, as required. ✓
- **`multiturn_prefix_reuse` and `stability_soak` timings not used as speed results.** Absent
  from §4.3.3 with the sealed-cache reason given. The one place their timings appear
  (105.206 s vs 0.726 s) is §4.3.5, as *evidence of the cache asymmetry* — the correct use. ✓
- **Absent equally-weighted axis stated in the decision section.** §7.1 leg 2, not a footnote. ✓
- **No manufactured break-even.** Absence reported with the pre-registered reason. ✓
- **Frame-count error recorded as the orchestrator's.** §4.7 row 1, attributed and overturned. ✓
- **Standing constraint present and correct.** §7.3: pinned fork held until a replacement
  carries `--max-kv-size`. ✓
- **§4.7 genuinely withdraws.** Five rows, including two withdrawals that *hurt* the paper's
  own earlier position and one — the `readFailureCount: 1` misreading — that nobody would
  have caught. ✓

## 4. Attacking the decision itself

**Is NO-GO supported, or the safe default dressed up?** Supported. I tested each leg.

- **Leg 1 (attestation refusal).** Verified structural, not configurational: the process *was*
  launched with `--ubatch-size 2048 --reasoning-effort medium` and parsed them; the refusal is
  that no route serialises them. The three escape routes are enumerated and each declined with
  a stated reason. Migration risk is in the pre-registered weighting, so this leg is admissible
  weight, not post-hoc.
- **Leg 2 (memory absent).** Verified: **zero windows scored on both sides.** The single most
  credible act in the paper is §4.4's refusal to re-derive the cost bound upward after seeing
  its own run refused — the fix would probably have made the windows score, and it was declined
  as confirmation bias on data whose direction was already known.
- **Leg 3 (noise floor) — the one I was asked to break.** It **holds, and it is the weakest.**
  The ±20 % traces to `TASK-260827-2v13w8_results.md`, and that source is not an assertion: the
  incumbent's *own* 8k prefill moved 101.126 → 83.744 tok/s (−17.2 %) and TTFT 76.974 → 92.950 s
  (+20.8 %) between two reruns of the same runtime, same config, comparable host load. Decode
  specifically has no same-config repeat, so applying ±20 % to decode is an extrapolation — but
  it is the *conservative* direction, and the cross-campaign decode swing (0.660×/0.686×, i.e.
  31–34 %) is decode-specific and larger. The article does not over-reach on it: it declines to
  apply it to TTFT/prefill, grants llama.cpp the speed win outright in §6.1 and §7.1, and says
  "this study does not claim it survives repeats" — reporting *unknown* rather than inferring
  stability from a proxy.

**The decisive test: remove leg 3 entirely.** Grant llama.cpp its full +20.1 %/+15.8 % decode as
real and repeatable. You then hold **one** of two equally-weighted axes, with the other having
no data in any direction, against a named migration-risk loss. Migrating on that is precisely
the single-axis win the brief forbade. **NO-GO survives the loss of the leg I was told to attack
hardest**, which is what makes it a decision rather than a preference.

**Is the reverse error being made?** No. §6.5 states three flip conditions *in advance*, §7.2
gives six ordered bounded work items — item 1 is a concrete engineering design (in-process
`proc_pidinfo`/`mach_vm_region` read replacing the `vmmap` fork, which removes the 7.0 s bound
rather than retuning it), and llama.cpp is "explicitly not retired". That is actionable. The one
soft spot is F5 below: the reopening work is concrete in prose but only partly tracked on the
board.

**MLX Swift retirement — proportionate.** The blocker reproduced at **1.144×** and **1.151×**
on two builds weeks apart, both outside a bar pinned before the numbers. Both reproduce from the
artifacts. Retirement with four named local-work blockers to reopening is the right call.

## 5. Attacking the verifier — three fresh mutants, two survive

The shipped campaign's 13 mutants all reproduce as caught. I ran three it does not contain,
using its own methodology (mutate, regenerate `expected-figures.json`, re-checksum, so only the
structural block can catch it), each on a throwaway copy:

| Mutant | Mutation | Result |
|---|---|---|
| **MU-A** | Swift 8k footprint → 1.05× **and** `decision.json` flipped to `accepted=true`, blockers emptied | **caught** — 2 structural failures |
| **MU-B** | Swift 8k footprint → 1.05× in the **record only**, `decision.json` untouched | **SURVIVED, exit 0** |
| **MU-C** | candidate decode 8.145→6.9 and 7.724→6.8, shaving the advantage from +20.1 %/+15.8 % to ~+2 % | **SURVIVED, exit 0** |

## 6. Findings

**F1 — one cited number does not trace (`ARTICLE.md` §4.2 capacity table).** The MLX Swift
`context_75k` TTFT ratio is printed as **`1.5493×`**. `recompute.py` derives **`1.548726`** from
the same records (1504.4526/971.4133), and I get the same independently. The source document
says `1.549x`; the article appears to have padded a fourth digit and got it wrong. Magnitude
0.0006, unscored metric, changes nothing — **but** §3.7 claims "Every number cited below is
regenerated from the sealed records … and `./reproduce.zsh` fails if any of them moves," and for
this one cell that is false: the run is green. *Fix: print `1.5487×` (or `1.549×`, matching §4.6).*

**F2 — `reproduce.zsh`'s MLX Swift checks validate the decision document, not the records.**
All four Swift structural checks read `decision.json` (`accepted`, `blockers`, `deltas`) and
never cross-check it against `records/mlx-swift.json`. MU-B proves it: a record whose 8k
footprint no longer yields 1.1512× still passes `the ratio is 1.1512x` green. This is the
"accepts a document about a measurement" class the paper names as its own central discipline in
§1.2 — reintroduced one layer up, in the reproduction gate. Bounded: `decision.json` is the real
gate's own sealed output from the run that produced the records, and the ratios *are* recomputed
into `candidateOverBaselineRatios`, so the expectation diff catches this unless expectations are
regenerated. Not disclosed in `mutant-campaign.md`. *Fix: derive `outside` from the recomputed
Swift ratios and assert the decision's blocker agrees with it.*

**F3 — one structural check is a tautology.** `crossCampaign.comparable` is a hardcoded `False`
literal at `recompute.py:381`. The check `the two candidate arms are marked non-comparable with
each other` therefore cannot fail for any input and asserts nothing about the data, while
appearing in the green run beside 24 record-derived checks. The *conclusion* is correct and well
argued in §4.6 — it is the check that is empty.

**F4 — the campaign's "13 narrowing mutants, all caught" overstates coverage.** MU-C is a
record-level narrowing mutant, run by the campaign's own stated methodology and with
`recompute.py` untouched, that survives and falsifies a headline the abstract leads with. It is
outside the disclosed B1 (modified-tooling) and D1 (delete-control) classes. §8 is honest that
only five structural claims are re-asserted and decode is not among them, so this is a
completeness caveat on the campaign's claim rather than a hidden one — but the sentence should
say "13 mutants against the five re-asserted structural claims", not imply general coverage.

**F5 — dangling cross-reference.** §9 cites "the mapped-file sampler redesign in §7.2.1"; no
§7.2.1 exists (it is §7.2 item 1). Also: §7.2's six reopening items are the concrete path the
decision's credibility rests on, and only three open items are listed as board-tracked. Worth
opening elements for the sampler redesign, the `coveredPeak` bypass and the 3× repeat.

## 7. Why this is accepted rather than reworked

F1 is a fourth-digit transcription slip in an unscored cell. F2–F4 are completeness gaps in the
paper's *self-verification apparatus*, whose boundary §8 already discloses honestly (B1), and
none of them causes the gate to admit something it must reject — the numbers are all correct as
shipped; I verified 297 of 298 against the raw records myself, outside the tooling. F5 is a typo
and a tracking note.

Against that: the decision is independently sound, survives deletion of its weakest leg, every
brief constraint is satisfied, every refusal is reported as a first-class result rather than
buried, and the paper withdraws four of its own prior claims including two that cost it its own
earlier position. Sending 1,103 lines back for a rounding digit and three verifier notes would
be ceremony.

**Handoff:** F1–F5 are for the orchestrator to carry forward as follow-up work. No `commit_ack`
supplied — the commit-owning mover makes the `done` transition.
