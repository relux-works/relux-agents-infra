#!/usr/bin/env python3
"""Separate two facts the rev1 report had collapsed into one: where the vision
tower LIVES on disk, and whether it is RESIDENT at runtime under the executed
model factory.

  A. file membership -- bytes of `vision_tower.*` / `model.visual.*` inside the
     MLX 8-bit checkpoint, straight from its safetensors index.
  B. production code path -- run the REAL mlx_lm.models.qwen3_5.Model.sanitize
     over the checkpoint's real key set. mlx_lm.utils.load_model calls sanitize
     BEFORE model.load_weights(...) and BEFORE mx.eval(model.parameters()), so
     anything sanitize drops is never evaluated into resident parameters.
     A mutant with the vision branch deleted is run alongside: if the filter did
     not matter, the mutant would drop the same keys. It must not.
  C. measured residency (--measure) -- actually load the text-only model through
     mlx_lm and report MLX active memory plus process RSS, and assert the loaded
     parameter tree contains zero vision tensors.

C is the only one of the three that licenses a memory-comparison number.
A is a file-size fact and must never be added to or subtracted from an RSS
comparison.

REPRODUCIBILITY:

    /Users/alexis/.local/pipx/venvs/mlx-lm/bin/python vision_residency.py            # A + B
    /Users/alexis/.local/pipx/venvs/mlx-lm/bin/python vision_residency.py --measure  # A + B + C

    interpreter : CPython 3.14.7 (pipx venv `mlx-lm`)   numpy 2.5.2   mlx_lm 0.31.3

--measure loads ~28.6 GB and refuses to start below 35 GiB reclaimable.
"""
import inspect
import json
import os
import re
import resource
import struct
import subprocess
import sys
import textwrap

import mlx_lm
from mlx_lm import utils as mlx_utils
from mlx_lm.models import qwen3_5

MLX = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"
VISION_PREFIXES = ("vision_tower", "model.visual")
DTYPE_BYTES = {'BF16': 2, 'F16': 2, 'F32': 4, 'U8': 1, 'I8': 1, 'U32': 4, 'I32': 4, 'U16': 2}

failures = []


def check(name, cond, detail):
    print(f"  [{'PASS' if cond else 'FAIL'}] {name}: {detail}")
    if not cond:
        failures.append(name)


def reclaimable_gib():
    """free + inactive + speculative pages, the same figure load_and_answer.sh uses."""
    out = subprocess.run(["vm_stat"], capture_output=True, text=True, check=True).stdout
    pages = 0
    for label in ("Pages free", "Pages inactive", "Pages speculative"):
        m = re.search(rf"{label}:\s+(\d+)", out)
        pages += int(m.group(1)) if m else 0
    return pages * 16384 / 1073741824


def index_entries():
    """(key -> byte size) for every tensor in the MLX checkpoint, from its shard headers."""
    with open(f"{MLX}/model.safetensors.index.json") as fh:
        wm = json.load(fh)['weight_map']
    sizes = {}
    for shard in sorted(set(wm.values())):
        with open(f"{MLX}/{shard}", 'rb') as f:
            hl = struct.unpack('<Q', f.read(8))[0]
            hdr = json.loads(f.read(hl))
        for k, m in hdr.items():
            if k == '__metadata__':
                continue
            off = m['data_offsets']
            sizes[k] = off[1] - off[0]
    missing = set(wm) - set(sizes)
    if missing:
        raise SystemExit(f"index lists {len(missing)} tensors absent from the shard headers")
    return sizes


# ------------------------------------------------------------------ part A
print("A. FILE MEMBERSHIP -- what is inside the MLX 8-bit checkpoint on disk")
sizes = index_entries()
vis = {k: v for k, v in sizes.items() if k.startswith(VISION_PREFIXES)}
vis_bytes = sum(vis.values())
total_bytes = sum(sizes.values())
print(f"    tensors total            : {len(sizes)}   {total_bytes} bytes")
print(f"    vision_tower*/model.visual*: {len(vis)}   {vis_bytes} bytes")
print(f"    everything else          : {len(sizes) - len(vis)}   {total_bytes - vis_bytes} bytes")
check("vision tower is present in the file", vis_bytes > 0,
      f"{vis_bytes} bytes on disk -- a file-size fact, NOT a residency fact")

# ------------------------------------------------------------------ part B
print("\nB. PRODUCTION CODE PATH -- mlx_lm.models.qwen3_5.Model.sanitize over the real key set")
print(f"    mlx_lm {mlx_lm.__version__} at {os.path.dirname(mlx_lm.__file__)}")


class _Shim:
    """Just enough of Model for the outer sanitize: it delegates to language_model."""
    class _LM:
        @staticmethod
        def sanitize(weights):
            return weights          # isolate the OUTER filter under test
    language_model = _LM()


weights = {k: None for k in sizes}
kept = qwen3_5.Model.sanitize(_Shim(), weights)
kept_vision = [k for k in kept if 'visual' in k or 'vision_tower' in k]
print(f"    keys in                  : {len(weights)}")
print(f"    keys out of sanitize     : {len(kept)}")
print(f"    vision keys surviving    : {len(kept_vision)}")
check("production sanitize drops every vision key", not kept_vision,
      f"{len(vis)} vision keys in, {len(kept_vision)} out")

# mutant: the same function with its vision branch removed. If the branch were
# decorative, the mutant would drop the same keys. It must not.
src = textwrap.dedent(inspect.getsource(qwen3_5.Model.sanitize))
lines = src.splitlines(keepends=True)
mutant_lines, drop_next = [], False
for ln in lines:
    if drop_next:                       # the `continue` belonging to a removed branch
        drop_next = False
        continue
    if 'vision_tower' in ln or 'model.visual' in ln:
        drop_next = ln.rstrip().endswith(':')
        continue
    mutant_lines.append(ln)
mutant_src = ''.join(mutant_lines)
check("mutant actually differs from production", mutant_src != src,
      f"{len(lines) - len(mutant_lines)} vision-filter lines removed from a copy of "
      "the production source")
ns = {}
exec(mutant_src, ns)  # noqa: S102 - deliberate: run the vision-filter-free mutant
mutant_kept = ns['sanitize'](_Shim(), dict(weights))
mutant_vision = [k for k in mutant_kept if 'visual' in k or 'vision_tower' in k]
check("without the vision branch the vision keys SURVIVE", len(mutant_vision) == len(vis),
      f"mutant keeps {len(mutant_vision)} vision keys -- the branch is load-bearing")

# ordering: sanitize must run before load_weights and before mx.eval, or dropping
# the keys would not stop them becoming resident.
lm_src = inspect.getsource(mlx_utils.load_model)
i_san = lm_src.find('model.sanitize(')
i_load = lm_src.find('model.load_weights(')
i_eval = lm_src.find('mx.eval(model.parameters())')
print(f"    load_model offsets: sanitize@{i_san} load_weights@{i_load} mx.eval@{i_eval}")
check("sanitize runs before load_weights", 0 <= i_san < i_load, f"{i_san} < {i_load}")
check("sanitize runs before mx.eval(parameters)", 0 <= i_san < i_eval, f"{i_san} < {i_eval}")

# ------------------------------------------------------------------ part C
if '--measure' in sys.argv:
    print("\nC. MEASURED RESIDENCY -- load the text-only model and read the real footprint")
    free = reclaimable_gib()
    print(f"    reclaimable before load  : {free:.1f} GiB")
    if free < 35:
        sys.exit(f"INSUFFICIENT MEMORY ({free:.1f} GiB < 35) -- refusing to load")

    import mlx.core as mx
    from mlx.utils import tree_flatten
    model, _tok = mlx_lm.load(MLX)
    mx.eval(model.parameters())
    params = dict(tree_flatten(model.parameters()))
    param_bytes = sum(v.nbytes for v in params.values())
    resident_vision = [k for k in params if 'visual' in k or 'vision_tower' in k]
    rss = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    print(f"    parameter tensors        : {len(params)}   {param_bytes} bytes")
    print(f"    mx active memory         : {mx.get_active_memory()} bytes")
    print(f"    mx peak memory           : {mx.get_peak_memory()} bytes")
    print(f"    process peak RSS         : {rss} bytes")
    check("zero vision tensors are resident after a text-only load",
          not resident_vision, f"resident vision tensors: {len(resident_vision)}")
    check("resident parameter bytes exclude the vision tower",
          param_bytes <= total_bytes - vis_bytes + 1_000_000,
          f"{param_bytes} resident vs {total_bytes} on disk "
          f"({total_bytes - param_bytes} bytes never materialised)")
else:
    print("\nC. MEASURED RESIDENCY -- skipped (pass --measure to load ~28.6 GB)")

print(f"\n{len(failures)} failure(s)" + (f": {failures}" if failures else ""))
sys.exit(1 if failures else 0)
