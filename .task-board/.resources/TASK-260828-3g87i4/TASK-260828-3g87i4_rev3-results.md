# TASK-260828-3g87i4 — revision 3 results

Scope: the single finding review left open (F1). F2–F5 were accepted in revision 2
and nothing in their scope was touched.

## What was wrong

`test_alignment_guard.py` claimed `quant_equivalence.main()` was bound to the one
comparability decision by inspecting source strings:

* `'comparability_verdict(' in inspect.getsource(main)`
* a regex looking for a bare numeric `> N` comparison inside `main()`

Review defeated both with a mutant that kept `if False: comparability_verdict(rows)`
and let the real CLI path decide inline at `RATIO_CEIL + 1.0`. The threshold is a
*name*, so the regex never saw it; the dead call satisfied the first check. The FP8
ratio 3.889 was admitted by `main()` and the whole suite still exited 0.

## What changed

Production (`quant_equivalence.py`) — one seam, no decision moved:

* `collect_rows(n_rows)` extracted from `main()`. It reads the staged artifacts and
  returns `(rows, skipped)`. It decides nothing.
* `main()` takes its rows from that seam, then prints the table, asks
  `comparability_verdict()` and exits on the answer, exactly as before.
* `mean ratio` now prints `nan` instead of raising `ZeroDivisionError` when no
  tensor is paired against MLX.

Tests (`test_alignment_guard.py`) — rewritten around the production entry path:

* `drive_main(module, rows, skipped)` replaces `collect_rows` and runs the real
  `main()`, capturing its stdout and exit status.
* Every comparability case asserts **what `main()` printed** (`COMPARABLE` /
  `NOT COMPARABLE` / `INCOMPLETE`) **and the status it exited with**.
* No test calls `comparability_verdict()` directly. No test reads production source
  to decide whether the gate is reachable. The two source-string checks are gone.
* Two mutants of the production file are named cases, each loaded as a real module
  and driven through its own `main()`. Each case fails unless the production
  assertion flips to failing against the mutant.

## Evidence

`test_alignment_guard.py` — **exit 0, 16 checks, 0 failures**
(`TASK-260828-3g87i4_guard-tests-04.log`). The count in the report is now the count
the suite prints; revisions 1 and 2 claimed 10 while 12 ran.

Gate 2, all driven through `main()`:

| injected rows | must emit | exit |
|---|---|---|
| real GGUF row (ratio 0.790) | COMPARABLE | 0 |
| BF16 re-rounded to the E4M3 grid (ratio 3.889) | NOT COMPARABLE | 1 |
| one FP8 row among three good ones | NOT COMPARABLE | 1 |
| a NaN ratio | NOT COMPARABLE | 1 |
| row at 1.5x the real noise (ratio 1.184) | COMPARABLE | 0 |
| tensor absent from the MLX build | COMPARABLE | 0 |
| tensor whose GGUF bytes could not be read | INCOMPLETE | 2 |

Gate 3 — the same mutants applied to the real production file in a scratch copy,
with the suite byte-unchanged:

| mutant | log | result |
|---|---|---|
| A: `RATIO_CEIL` 3.0 -> 4.0 | `TASK-260828-3g87i4_mutant-rev3-ratio.log` | **exit 1**, 3 failures incl. the FP8 gate-2 case |
| B: dead `if False: comparability_verdict(rows)` + inline decision at `RATIO_CEIL + 1.0` (review's own bypass) | `TASK-260828-3g87i4_mutant-rev3-bypass.log` | **exit 1**, 3 failures incl. the FP8 gate-2 case |

A control case records why gate 2 must observe behaviour: revision 2's syntactic
inspection, applied to mutant B, still reports it clean.

Each mutant case also asserts its own mutation applied — exactly one site, source
changed, module loads. An unbuildable mutant fails rather than passing silently
(visible in the mutant-B log, where the doubly-bypassed file does not compile).

## Regression check on the refactor

`quant_equivalence.py` full 1024-row run after the extraction:
`TASK-260828-3g87i4_quant-equivalence-04.log`, exit 0, **byte-identical** to the
revision-2 `quant-equivalence-03.log` — 16 paired tensors, mean ratio 0.766,
COMPARABLE, 2 tensors absent from the MLX build.

`ruff check quant_equivalence.py test_alignment_guard.py` — exit 0.

## Not rerun in revision 3

`load_and_answer.sh` and `test_load_answer_gate.sh` are byte-identical to the
artifacts whose green runs are already attached (`load-and-answer-03.log`, exit 0,
`finish_reason: stop`, asserted answer; `load-gate-tests-01.log`, exit 0, 11/11).
Nothing in revision 3 touches either of them.

One rerun of `load_and_answer.sh` was attempted at 17:05 and **refused with exit 3**
(`TASK-260828-3g87i4_load-and-answer-04.log`): TASK-260827-2v13w8-rev4's benchmark held
port 18031 with `mlx_lm-relux.server` and the mlx-swift prototype both live. The
neighbour was not signalled and `CONTENTION_SCAN` / `CONTENTION_PROCS` were not
narrowed to get past it. The host was still held at 17:17, so the check was left
unrerun rather than worked around. The revision-2 real run stands as the evidence for
this artifact.
