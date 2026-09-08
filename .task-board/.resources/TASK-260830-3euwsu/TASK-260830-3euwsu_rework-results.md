# TASK-260830-3euwsu — rework after changes-requested verdict

Both required fixes from the review verdict applied to `.research/260831_engine-adapter-contract-and-canonical-knob-set.spec.md` (committed `040be30`).

## Fix 1 — removed Knob 1's argv-fallback exception

The old §2 named exactly one standing exception to the Prime Directive: Knob 1's `notReported` state falling back to reading `--max-kv-size` from argv. The reviewer traced this to a root-caused, fixed production bug (`LOGBOOK.md:310`, entry "Missing Live KV Evidence No Longer Falls Back To Argv"): production `benchmark-run` once treated an answered `/v1/models` without `meta.n_ctx` as permission to reuse caller-requested `--max-kv-size`, and this was closed by `TASK-260830-2hc5r2`, confirmed unchanged through its final revision (`TASK-260830-2hc5r2_rework-rev6-results.md:18-30`: "All six non-value states are inadmissible. None falls back to or is decoded from argv.").

Changes:
- §2 rewritten: no standing exception, full stop. The superseded citation (`260828_llamacpp-in-the-benchmark-gate.md:481`, an old MLX-Swift-only design predating `STORY-260830-2vrhg1` and `TASK-260830-2hc5r2`) is now explicitly named as superseded rather than reused as the general rule.
- Knob 1's `notReported` row rewritten: no argv fallback, refused as `contextBoundNotHonoured` / `kv=not-reported`, cited to the rework-rev4-results quote.
- Knob 2's row: "unlike Knob 1" -> "same as Knob 1" (both now forbid argv fallback).
- §1 vocabulary table's `notReported` row: no longer claims an allowed argv fallback exists for "specific terms."
- §5 constraints: extended from "Knobs 2 and 3's argv fallback is forbidden" to "Knobs 1, 2, and 3."
- Knob 10 reconciled: reframed as a forward-looking specification obligation for a hypothetical future engine with (a) no live-report path and (b) no `contextBoundNotHonoured`-equivalent refusal guard — explicitly distinguished from today's `mlx_lm` fork, which has a live-report path. Does not reuse the superseded MLX-Swift-arm citation.

## Fix 2 — sourced Knob 6's help-text quote

The literal string "Maximum size in bytes of the KV caches" was my (the brief author's) addition without provenance, not something the developer's repo-internal citations could support. Verified directly against the pinned fork: `/Users/alexis/src/relux-works/mlx-lm/mlx_lm/server.py:1902`, commit `45a472f2d0cda166b7ffe1a80fe50dd9621f4303`. Knob 6's row now cites that fork path/commit explicitly and notes it is outside this repository.

## Scope discipline

Only the two flagged sections were touched (§2, Knob 1, Knob 6, plus the minimal ripple into §1/§5/Knob 2/Knob 10 needed for internal consistency after removing the exception). No other knob's content, citations, or obligations were rewritten.
