#!/usr/bin/env python3
"""Negative tests for the two gates quant_equivalence.py relies on.

A passing equivalence report is worthless unless these gates would have refused
a bad pair. Each case below is a mutant that MUST be rejected.

  gate 1 -- try_align(): must not manufacture an axis permutation between tensors
            that are simply different weights.
  gate 2 -- the PRODUCTION COMPARABLE / NOT COMPARABLE decision, exercised by
            running quant_equivalence.main() itself. Every gate-2 case replaces
            the collect_rows() seam with controlled rows, runs the real main(),
            and asserts BOTH the verdict line main() prints and the status it
            exits with. Nothing here calls comparability_verdict() directly and
            nothing here reads production source to decide whether the gate is
            wired up: a check that cannot observe behaviour cannot witness it.
  gate 3 -- mutants of the production file, each loaded and driven through its
            own main() the same way. A mutant that the gate-2 assertions do not
            reject is a hole in this suite, so each mutant case FAILS unless the
            production assertion flips to failing against it.

This file defines NO threshold of its own. Two production mutants bound it:
raising RATIO_CEIL 3.0 -> 4.0, and the call-site bypass that keeps a dead
`if False: comparability_verdict(rows)` while the CLI decides inline at a looser
threshold. Both admit the FP8 ratio 3.889 and both must be caught here.

REPRODUCIBILITY -- requires NumPy and must be run from this file's directory
(it imports quant_equivalence.py sitting next to it):

    cd <dir containing quant_equivalence.py>
    /Users/alexis/.local/pipx/venvs/mlx-lm/bin/python test_alignment_guard.py

    interpreter : CPython 3.14.7 (pipx venv `mlx-lm`)   numpy : 2.5.2
"""
import contextlib
import importlib.util
import io
import pathlib
import re
import sys
import types

import numpy as np

# Import the production module for real -- it is __main__-guarded, so importing it
# loads the actual code under test without running the report.
_SRC = pathlib.Path(__file__).resolve().parent / 'quant_equivalence.py'
_PROD_TEXT = _SRC.read_text()
_spec = importlib.util.spec_from_file_location('quant_equivalence', _SRC)
qe = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(qe)

gguf_index, gguf_rows, st_rows = qe.gguf_index, qe.gguf_rows, qe.st_rows
try_align, rel_rms, mlx_dequant = qe.try_align, qe.rel_rms, qe.mlx_dequant
GGUF, BF16 = qe.GGUF, qe.BF16

failures = []
checks = []


def check(name, cond, detail):
    checks.append(name)
    print(f"  [{'PASS' if cond else 'FAIL'}] {name}: {detail}")
    if not cond: failures.append(name)


def e4m3(x):
    """Round to the E4M3 grid: 4 significant bits of mantissa."""
    m, e = np.frexp(x.astype(np.float64))
    return np.ldexp(np.round(m * 16.0) / 16.0, e).astype(np.float32)


def row(name, err_mlx, err_gguf):
    """One row in collect_rows()'s exact shape: (name, err_mlx, err_gguf, ratio, note)."""
    ratio = (err_gguf / err_mlx) if err_mlx else float('nan')
    return (name, err_mlx, err_gguf, ratio, '')


# ------------------------------------------------------ driving production main()
def load_variant(label, text):
    """Load a source variant of the production file as a real, importable module."""
    mod = types.ModuleType(f'quant_equivalence__{label}')
    mod.__file__ = str(_SRC)
    # Executing a mutated copy of the file under test is the point of this helper --
    # the mutant must be a real, runnable module so its main() can be driven.
    exec(compile(text, f'{_SRC}#{label}', 'exec'), mod.__dict__)  # noqa: S102
    return mod


def drive_main(module, rows, skipped=()):
    """Run module.main() FOR REAL over injected rows.

    Only the data source is replaced: main() still builds the table, asks for the
    verdict however it chooses to, prints it, and exits. Returns (exit_code, stdout).
    """
    buf = io.StringIO()
    saved_collect, saved_argv = module.collect_rows, sys.argv
    module.collect_rows = lambda n_rows: (list(rows), list(skipped))
    sys.argv = ['quant_equivalence.py']
    code = 0
    try:
        with contextlib.redirect_stdout(buf):
            module.main()
    except SystemExit as exc:
        code = exc.code if isinstance(exc.code, int) else (0 if exc.code is None else 1)
    finally:
        module.collect_rows, sys.argv = saved_collect, saved_argv
    return code, buf.getvalue()


def emitted_verdict(out):
    """The verdict main() actually printed -- ordered so COMPARABLE cannot match
    the tail of NOT COMPARABLE."""
    if 'NOT COMPARABLE' in out: return 'NOT COMPARABLE'
    if 'INCOMPLETE' in out: return 'INCOMPLETE'
    if 'COMPARABLE' in out: return 'COMPARABLE'
    return 'NO VERDICT'


def main_says(module, rows, want_verdict, want_code, skipped=()):
    """(ok, detail) for 'this module's main() emits want_verdict and exits want_code'."""
    code, out = drive_main(module, rows, skipped)
    got = emitted_verdict(out)
    ok = (got == want_verdict and code == want_code)
    return ok, f"main() emitted {got!r} exit={code}; expected {want_verdict!r} exit={want_code}"


kv, ts, ds = gguf_index(GGUF)
N = 512

print("gate 1 -- try_align must refuse to align genuinely different tensors")
g = gguf_rows(GGUF, ts, ds, 'blk.0.ffn_gate.weight', N)
b_self, _, _ = st_rows(BF16, 'model.language_model.layers.0.mlp.gate_proj.weight', N)
b_other, _, _ = st_rows(BF16, 'model.language_model.layers.1.mlp.gate_proj.weight', N)

_, _, _, note_ok = try_align(g.copy(), b_self.copy(), None)
check("true pair aligns", not note_ok.startswith("MISMATCH"),
      f"note={note_ok!r} rel_rms={rel_rms(b_self, g):.7f}")

_, b_al, _, note_bad = try_align(g.copy(), b_other.copy(), None)
check("wrong-layer pair refused", note_bad.startswith("MISMATCH"),
      f"note={note_bad!r} rel_rms_after={rel_rms(b_al, g):.4f}")

# a real permutation must still be recovered, so the guard is not just "always refuse"
rng = np.random.default_rng(0)
cperm = rng.permutation(g.shape[1])
rperm = rng.permutation(g.shape[0])
_, b_p, _, note_p = try_align(g[:, cperm].copy(), b_self.copy(), None)
check("true column permutation recovered", "input axis permuted" in note_p,
      f"note={note_p!r} rel_rms_after={rel_rms(b_p, g[:, cperm]):.7f}")

_, b_r, _, note_r = try_align(g[rperm].copy(), b_self.copy(), None)
check("true row permutation recovered", "output axis permuted" in note_r,
      f"note={note_r!r} rel_rms_after={rel_rms(b_r, g[rperm]):.7f}")

# --------------------------------------------------------------------------- gate 2
print("\ngate 2 -- production main() must emit NOT COMPARABLE on an FP8-grade upstream loss")

m, meta = mlx_dequant('language_model.model.layers.0.mlp.gate_proj.weight', N,
                      ts['blk.0.ffn_gate.weight'][0][0])
m = m[:b_self.shape[0]]
e_mlx = rel_rms(b_self, m)
e_gguf = rel_rms(b_self, g)
fake = e4m3(b_self)                 # what the GGUF would look like via an FP8 E4M3 checkpoint
e_fake = rel_rms(b_self, fake)

REAL_ROW = row('blk.0.ffn_gate.weight', e_mlx, e_gguf)
FP8_ROW = row('blk.0.ffn_gate.weight', e_mlx, e_fake)
print(f"  (real ratio {e_gguf / e_mlx:.3f} | FP8-mutant ratio {e_fake / e_mlx:.3f} | "
      f"mlx={e_mlx:.7f} gguf={e_gguf:.7f} fp8={e_fake:.7f})")

check("real GGUF row: production main() emits COMPARABLE and exits 0",
      *main_says(qe, [REAL_ROW], 'COMPARABLE', 0))

# THE production negative: the FP8 row must be refused by the runtime path itself.
check("FP8 row: production main() emits NOT COMPARABLE and exits 1",
      *main_says(qe, [FP8_ROW], 'NOT COMPARABLE', 1))

# one bad tensor among good ones -- a verdict that only looked at the mean would pass it
check("one FP8 row among three good ones: production main() still refuses",
      *main_says(qe, [row('good.0', e_mlx, e_gguf), row('good.1', e_mlx, e_gguf),
                      FP8_ROW, row('good.3', e_mlx, e_gguf)], 'NOT COMPARABLE', 1))

# NaN must never read as "fine": a ratio that is not a number is not below a ceiling
check("NaN ratio: production main() refuses",
      *main_says(qe, [('nan.0', e_mlx, float('nan'), float('nan'), '')], 'NOT COMPARABLE', 1))

# narrowing: a mutant only slightly worse than the real thing must still pass, so the
# gate is not simply refusing everything that is not byte-identical
mild = b_self + (g - b_self) * 1.5
e_mild = rel_rms(b_self, mild)
check("1.5x-noise row: production main() still accepts (gate is not reject-all)",
      *main_says(qe, [row('blk.0.ffn_gate.weight', e_mlx, e_mild)], 'COMPARABLE', 0))

# a tensor absent from the MLX build carries no ratio and must not vote either way
check("MLX-absent row casts no vote",
      *main_says(qe, [REAL_ROW, ('blk.64.ffn_up.weight', None, 0.0054106, float('nan'), 'absent')],
                 'COMPARABLE', 0))

# an unreadable tensor is a failure to read, not an absence: it must not read as success
check("unreadable tensor: production main() emits INCOMPLETE and exits 2, never COMPARABLE",
      *main_says(qe, [REAL_ROW], 'INCOMPLETE', 2, skipped=['blk.3.attn_q.weight']))

# --------------------------------------------------------------------------- gate 3
print("\ngate 3 -- production mutants that admit the FP8 row must make gate 2 fail")


def mutant(label, old, new):
    """Materialise a mutant of the production file.

    Returns (module, applied, why). `applied` is False -- and the caller's check
    fails -- unless the mutation hit exactly one site, changed the source, AND
    produced a module that actually loads. A mutant that cannot be built proves
    nothing, so it must never pass silently.
    """
    hits = _PROD_TEXT.count(old)
    if hits != 1:
        return None, False, f"expected exactly 1 mutation site, found {hits}"
    text = _PROD_TEXT.replace(old, new, 1)
    if text == _PROD_TEXT:
        return None, False, "mutation left the source unchanged"
    try:
        return load_variant(label, text), True, "single-site textual mutation of production source"
    except Exception as exc:                       # noqa: BLE001 - report, never crash the suite
        return None, False, f"mutant source does not load: {type(exc).__name__}: {exc}"


# mutant A -- widen the single production threshold past the observed FP8 ratio
mut_ratio, ratio_applied, why_a = mutant('ratio_ceil_4', 'RATIO_CEIL = 3.0', 'RATIO_CEIL = 4.0')
check("mutant A applied: production RATIO_CEIL 3.0 -> 4.0", ratio_applied, why_a)
if ratio_applied:
    ok_a, detail_a = main_says(mut_ratio, [FP8_ROW], 'NOT COMPARABLE', 1)
    check("mutant A is KILLED: widened RATIO_CEIL fails the FP8 gate-2 assertion",
          not ok_a, f"mutant {detail_a}")

# mutant B -- the reviewer's call-site bypass: a dead syntactic call to the real
# verdict, while the CLI path decides inline at RATIO_CEIL + 1.0 (admits 3.889).
BYPASS_OLD = "    comparable, bad, paired = comparability_verdict(rows)\n"
BYPASS_NEW = (
    "    # MUTANT: dead syntactic call kept so source inspection still sees it,\n"
    "    # while the real CLI path decides inline at a looser threshold.\n"
    "    if False:\n"
    "        comparable, bad, paired = comparability_verdict(rows)\n"
    "    paired = [r for _, em, _, r, _ in rows if em is not None]\n"
    "    bad = [(n, r) for n, em, _, r, _ in rows\n"
    "           if em is not None and (math.isnan(r) or r > RATIO_CEIL + 1.0)]\n"
    "    comparable = not bad\n"
)
mut_bypass, bypass_applied, why_b = mutant('callsite_bypass', BYPASS_OLD, BYPASS_NEW)
check("mutant B applied: dead comparability_verdict call + inline CLI decision",
      bypass_applied, why_b)
if bypass_applied:
    ok_b, detail_b = main_says(mut_bypass, [FP8_ROW], 'NOT COMPARABLE', 1)
    check("mutant B is KILLED: call-site bypass fails the FP8 gate-2 assertion",
          not ok_b, f"mutant {detail_b}")

    # why gate 2 has to observe behaviour: the source-string checks this suite used
    # to rely on still report the bypass mutant as clean.
    mut_main_src = BYPASS_NEW + _PROD_TEXT.split(BYPASS_OLD, 1)[1]
    syntactic_clean = ('comparability_verdict(' in mut_main_src
                       and not re.findall(r'>\s*\d+(?:\.\d+)?\s*(?:or|\))', mut_main_src))
    check("syntactic call-site inspection would have MISSED mutant B",
          syntactic_clean,
          "mutant B keeps the literal call and holds no bare numeric comparison, so "
          "source inspection reports it clean -- only the driven main() catches it")

print(f"\n{len(checks)} checks, {len(failures)} failure(s)" + (f": {failures}" if failures else ""))
sys.exit(1 if failures else 0)
