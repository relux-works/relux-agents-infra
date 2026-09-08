# TASK-260830-3euwsu — review verdict revision 2: changes requested (minor)

Reviewed artifact: `.research/260831_engine-adapter-contract-and-canonical-knob-set.spec.md`
at HEAD `6d682b5` (CR base OID), which matches the CR candidate tree —
`repository_delta=empty` is correct, this file was already committed by `040be30`
before this CR was opened.

Note: the outcome resource `TASK-260830-3euwsu_engine-adapter-contract.spec.md`
attached to the board is stale — it still shows the pre-`040be30` text (the
Knob 1 argv-fallback exception is still present in that resource). The actual
committed repository file at HEAD is correct and reflects the fix. This review
is against the real repository file, which governs; the stale resource should
be refreshed with `update_resource` so the board's own record matches the
delivered content, but it is not itself a rework blocker.

## 1. Is the carve-out gone, or narrower?

Gone, not narrowed. Grepped the whole document for every `argv` occurrence
(24 hits). §2 is rewritten with no standing exception and explicitly explains
*why* the removed exception was wrong (superseded citation, root-caused
production bug). Knob 1's `notReported` row now reads "No argv fallback" and
routes to `contextBoundNotHonoured`. Knob 2/3 unchanged (already forbidden).
Knob 10's "Adapter obligation" now states plainly: "This contract currently
allows no argv fallback for any pinned term (§2)." §5's constraint bullet now
covers Knobs 1, 2, and 3. §4 (downstream obligations) and §6 (reopening
conditions) contain no argv mentions at all — no back door there. Confirmed
clean.

## 2. Forward-looking language

`notReported`'s vocabulary-table row (§1) and Knob 10 both gate a future
argv fallback behind two concrete, checkable conditions (no live-report path
+ no `contextBoundNotHonoured`-equivalent refusal guard) plus an explicit,
cited revision — not a vague "TBD" escape hatch. A future revision would have
to name the new engine, prove both conditions against that engine's actual
API surface, and cite the proof, exactly like every other knob in this
document is required to. This reads as an honest door, not a re-entry point.

## 3. Retained adapter obligation

Knob 1's "Adapter obligation" row is untouched by this revision and still
states completion-before-attestation as an obligation with a named failure
consequence ("Conflating those two produces a false `unbounded` pin on a
runtime that is in fact bounded, one request away from proving it"), not as
advisory language. Confirmed.

## 4. Citations — spot-checked five, including the two the brief named

- `TASK-260830-2hc5r2_rework-rev4-results.md:5-7` — **holds.** Lines 5-6
  contain the exact quoted sentence about the three-state rule and the
  `contextBoundNotHonoured` refusal.
- `TASK-260830-2hc5r2_rework-rev6-results.md:18-30` — **holds.** Line 29
  contains the exact quoted sentence ("All six non-value states are
  inadmissible. None falls back to or is decoded from argv."), well within
  the cited range.
- `LOGBOOK.md:310` — **does not hold.** At HEAD (`6d682b5`), line 310 of
  `LOGBOOK.md` is blank. The cited root-cause text ("`RuntimeBenchmark
  .contextPolicy` treated an answered `/v1/models` response without
  `meta.n_ctx`...") is now at line 316, inside entry "0456 — Missing Live KV
  Evidence No Longer Falls Back To Argv". This citation was accurate at
  `040be30` (the rework commit) — line 310 was correct there — but the very
  next commit in this same task's sequence, `6d682b5` ("record logbook entry
  for the argv-fallback correction"), prepended 7 lines to the top of
  `LOGBOOK.md` (confirmed: entries are prepended, IDs descend as you read
  down — 0555, 0551, 0550, 0504, 0456, 0354), shifting every later line down
  by 6 and silently invalidating the citation this same rework had just
  written. This is not a one-off: the prior review round's own
  `LOGBOOK.md:310` citation (written against `0445307`) was accurate at the
  time for the same reason and would have gone stale on the same later
  commit. `LOGBOOK.md:310` appears **three times** in the current spec (§2
  body, Knob 1's `notReported` row, §5's constraint bullet) — all three are
  now wrong in the same way.
- Knob 6's help-text quote — **holds, and is now precisely sourced.**
  Verified directly: `/Users/alexis/src/relux-works/mlx-lm` at commit
  `45a472f2d0cda166b7ffe1a80fe50dd9621f4303` (`git log -1 -- mlx_lm/server.py`
  confirms this is the commit that introduced the line), `mlx_lm/server.py:1902`
  reads exactly `help="Maximum size in bytes of the KV caches"`. Matches the
  spec's quote and location verbatim.
- Knob 2's "44 enumerated routes" claim — **holds.** Study
  `260831_local-qwen-runtime-comparison-study.md:745`: "**NO** — prefill
  chunk and reasoning effort `notReported` on all 44 routes (§4.1)."

4 of 5 spot-checked citations hold exactly. `LOGBOOK.md:310` does not, and
the failure mode is structural rather than incidental: this document cites
specific line numbers into a log file that prepends new entries at the top,
which guarantees drift on every future entry, not just this task's own. The
fix that already happened here (recording the correction as a new logbook
entry) demonstrates the problem live: the citation broke within the same
commit sequence that introduced it.

## 5. Substance is otherwise sound

The argv-fallback exception is genuinely closed, not narrowed. The
`contextBoundNotHonoured` routing, the `unpinnableConditions` framing, and
the "same as Knob 1"/"Knobs 1, 2, and 3" language are all internally
consistent across every section that touches this rule. No admission clause
of the comparative gate is weakened. This was the substantive defect from
round 1 and it is fixed.

## Required rework (small, mechanical — not a re-litigation of round 1)

- Fix the three `LOGBOOK.md:310` references (§2 body, Knob 1's `notReported`
  row, §5). Either correct the line number to the current location, or —
  better, given `LOGBOOK.md` prepends and will drift again — cite the stable
  entry identifier instead of a raw line number (e.g. "`LOGBOOK.md`, entry
  `0456` — 'Missing Live KV Evidence No Longer Falls Back To Argv'"), so a
  future unrelated logbook entry doesn't silently break this citation again.
- Refresh the stale `TASK-260830-3euwsu_engine-adapter-contract.spec.md`
  outcome resource so it matches the committed file at HEAD (currently shows
  the pre-`040be30` text with the argv exception still present).

## Verdict: CHANGES REQUESTED (minor) — route to `analysis`

Not a re-opening of the argv-exception question — that is resolved. This is
a citation-accuracy defect the task's own citation discipline (§0: "A knob
with no citation is not in this document") requires to be fixed, caused by
this task's own commit ordering. Expect this to be a one-line-per-occurrence
fix plus a resource refresh, not another investigation round.
