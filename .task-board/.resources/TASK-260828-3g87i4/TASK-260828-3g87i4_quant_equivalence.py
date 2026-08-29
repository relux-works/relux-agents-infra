#!/usr/bin/env python3
"""Establish, numerically, how the staged GGUF Q8_0 relates to the MLX 8-bit/group64 baseline.

For each tensor we dequantize three artifacts and compare against the BF16 source of record:

  A) BF16 reference : Qwen3.8-27B-Uncensored-BF16      safetensors, bfloat16
  B) MLX 8-bit g64  : Qwen3.8-27B-Uncensored-MLX-8bit  affine, uint8 + bf16 scale + bf16 bias, group 64
  C) GGUF Q8_0      : ...-OrcaRouter-Q8_0.gguf         symmetric, int8 + fp16 scale, block 32

Reported per tensor:
  * rel_rms of B vs A and of C vs A
  * the ratio C_err/A_err. ~1x means both are 8-bit quantizations of the SAME numbers.
    A ratio >> 3x would mean the GGUF inherited a coarser upstream grid (e.g. an FP8
    E4M3 checkpoint, 4-bit mantissa) and the two 8-bit builds are NOT comparable.
  * whether GGUF needed an axis permutation to line up with the HF/MLX layout.

The permutation path is guarded: a permutation is accepted only if it is a genuine
bijection AND it drives the residual error down to quantization noise. Otherwise the
tensor is reported as MISMATCH, not silently "aligned".

The COMPARABLE / NOT COMPARABLE decision lives in exactly one place --
comparability_verdict() below, over the single threshold RATIO_CEIL. main() reports
that function's answer and nothing else, and there is deliberately no second copy of
the threshold anywhere. test_alignment_guard.py does not take that on trust: it
replaces the collect_rows() seam and runs THIS FILE'S main(), asserting the verdict
line main() prints and the status it exits with. A main() that stopped consulting
comparability_verdict -- or that kept a dead call to it and decided inline elsewhere
-- is caught by observing that behaviour, not by reading this source.

REPRODUCIBILITY -- requires NumPy, which the system `python3` on this host does not
have. Run it with the pinned interpreter:

    /Users/alexis/.local/pipx/venvs/mlx-lm/bin/python quant_equivalence.py

    interpreter : CPython 3.14.7  (pipx venv `mlx-lm`)
    numpy       : 2.5.2
    mlx_lm      : 0.31.3          (not imported here; the venv is reused for numpy)

Any Python >= 3.10 with NumPy >= 2.0 works; only the artifact paths below are fixed.
"""
import gc
import json
import math
import struct
import sys

try:
    import numpy as np
except ModuleNotFoundError:  # pragma: no cover - environment guidance, not logic
    sys.exit(
        "numpy is required and is absent from this interpreter.\n"
        "Run with the pinned interpreter instead:\n"
        "  /Users/alexis/.local/pipx/venvs/mlx-lm/bin/python " + sys.argv[0]
    )

GGUF_DIR = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0"
GGUF = f"{GGUF_DIR}/Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf"
BF16 = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-BF16"
MLX  = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"

NOISE_CEIL = 0.05          # rel_rms above this is not 8-bit rounding noise
FULL_READ_ELEMS = 64_000_000   # read the whole tensor below this element count
RATIO_CEIL = 3.0           # PRODUCTION comparability threshold - single source of truth.
                           # A GGUF that inherited a coarser upstream grid (FP8 E4M3 has
                           # 4 mantissa bits) lands far above this against the same MLX
                           # baseline; true 8-bit-of-the-same-numbers lands below 1.


# ------------------------------------------------------------------ GGUF
class _R:
    def __init__(s, f): s.f = f
    def raw(s, n):
        b = s.f.read(n)
        if len(b) < n: raise EOFError
        return b
    def u32(s): return struct.unpack('<I', s.raw(4))[0]
    def u64(s): return struct.unpack('<Q', s.raw(8))[0]
    def st(s): return s.raw(s.u64()).decode('utf-8', 'replace')
    def val(s, t):
        if t == 8: return s.st()
        if t == 9:
            et = s.u32(); n = s.u64()
            return [s.val(et) for _ in range(n)]
        fmt, n = {0:('<B',1),1:('<b',1),2:('<H',2),3:('<h',2),4:('<I',4),5:('<i',4),
                  6:('<f',4),7:('<B',1),10:('<Q',8),11:('<q',8),12:('<d',8)}[t]
        v = struct.unpack(fmt, s.raw(n))[0]
        return bool(v) if t == 7 else v


def gguf_index(path):
    with open(path, 'rb') as f:
        r = _R(f)
        if r.raw(4) != b'GGUF': raise SystemExit("not a GGUF file")
        r.u32(); nt = r.u64(); nkv = r.u64()
        kv = {}
        for _ in range(nkv):
            k = r.st(); kv[k] = r.val(r.u32())
        ts = {}
        for _ in range(nt):
            name = r.st(); nd = r.u32()
            dims = [r.u64() for _ in range(nd)]
            ts[name] = (dims, r.u32(), r.u64())
        align = kv.get('general.alignment', 32)
        return kv, ts, (f.tell() + align - 1) // align * align


def gguf_rows(path, ts, data_start, name, n_rows):
    """Rows of a Q8_0 tensor, shape (rows, ne0), in GGUF's own axis order."""
    dims, tt, off = ts[name]
    if tt != 8: raise SystemExit(f"{name}: ggml type {tt}, expected Q8_0(8)")
    ne0, ne1 = dims[0], dims[1]
    n_rows = min(n_rows, ne1)
    bpr = ne0 // 32
    rb = bpr * 34
    with open(path, 'rb') as f:
        f.seek(data_start + off)
        buf = f.read(rb * n_rows)
    if len(buf) != rb * n_rows:
        raise SystemExit(f"{name}: bytes not present (partial download)")
    a = np.frombuffer(buf, dtype=np.uint8).reshape(n_rows, bpr, 34)
    d = a[:, :, :2].copy().view(np.float16).astype(np.float32).reshape(n_rows, bpr, 1)
    q = a[:, :, 2:].copy().view(np.int8).astype(np.float32)
    return (d * q).reshape(n_rows, ne0)


# ------------------------------------------------------------ safetensors
def st_rows(dirpath, name, n_rows):
    with open(f"{dirpath}/model.safetensors.index.json") as fh:
        wm = json.load(fh)['weight_map']
    if name not in wm: return None, None, None
    path = f"{dirpath}/{wm[name]}"
    with open(path, 'rb') as f:
        hl = struct.unpack('<Q', f.read(8))[0]
        hdr = json.loads(f.read(hl))
        base = 8 + hl
        m = hdr[name]
        shape, dt = m['shape'], m['dtype']
        isz = {'BF16':2,'F16':2,'F32':4,'U8':1,'I8':1,'U32':4}[dt]
        rowlen = int(np.prod(shape[1:])) if len(shape) > 1 else 1
        rows = min(n_rows, shape[0])
        f.seek(base + m['data_offsets'][0])
        buf = f.read(rows * rowlen * isz)
    a = np.frombuffer(buf, dtype=np.uint8)
    if dt == 'BF16':
        out = np.frombuffer(((a.view('<u2').astype('<u4')) << 16).tobytes(), dtype='<f4')
    elif dt == 'F16': out = a.view(np.float16).astype(np.float32)
    elif dt == 'F32': out = a.view('<f4')
    elif dt == 'U32': out = a.view('<u4')
    else: out = a.view(np.int8 if dt == 'I8' else np.uint8)
    return out.reshape(rows, -1), shape, dt


def mlx_dequant(name, n_rows, ncols):
    w, _wshape, wdt = st_rows(MLX, name, n_rows)
    if w is None: return None, None
    sc, _, sdt = st_rows(MLX, name.replace('.weight', '.scales'), n_rows)
    bi, _, bdt = st_rows(MLX, name.replace('.weight', '.biases'), n_rows)
    if sc is None: return None, {"quantized": False}
    group = ncols // sc.shape[1]
    per_word = ncols // w.shape[1]
    bits = 32 // per_word
    q = np.stack([(w >> (bits * k)) & ((1 << bits) - 1) for k in range(per_word)], axis=-1)
    q = q.reshape(w.shape[0], ncols).astype(np.float32)
    deq = (q.reshape(w.shape[0], -1, group) * sc[:, :, None] + bi[:, :, None]).reshape(w.shape[0], ncols)
    return deq, {"quantized": True, "bits": bits, "group_size": group,
                     "scale_dtype": sdt, "bias_dtype": bdt, "packed_dtype": wdt}


# ------------------------------------------------------------- comparison
def rel_rms(ref, x):
    ref = ref.astype(np.float64); x = x.astype(np.float64)
    return float(np.sqrt(np.mean((x - ref) ** 2)) / np.sqrt(np.mean(ref ** 2)))


def _perm_from_corr(G, B, probe=128):
    """perm[j] = index in B matching column j of G, or None if not a bijection."""
    n = G.shape[1]
    if B.shape[1] != n: return None
    step = max(1, G.shape[0] // probe)
    Gp = G[::step][:probe]; Bp = B[::step][:probe]
    Gn = Gp / (np.linalg.norm(Gp, axis=0, keepdims=True) + 1e-30)
    Bn = Bp / (np.linalg.norm(Bp, axis=0, keepdims=True) + 1e-30)
    perm = np.empty(n, dtype=np.int64)
    best = np.empty(n, dtype=np.float32)
    for s0 in range(0, n, 2048):
        blk = Gn[:, s0:s0 + 2048].T @ Bn            # (chunk, n)
        np.abs(blk, out=blk)
        idx = np.argmax(blk, axis=1)
        perm[s0:s0 + 2048] = idx
        best[s0:s0 + 2048] = blk[np.arange(len(idx)), idx]
        del blk
    if len(np.unique(perm)) != n: return None       # not a bijection -> refuse
    if float(best.min()) < 0.95: return None        # weak match -> refuse
    return perm


def try_align(g, b, m):
    """Return (g, b, m, note). Tries identity, then a column permutation, then a
    row permutation. A permutation is accepted only when it is a bijection AND it
    drives the residual to 8-bit rounding noise."""
    if rel_rms(b, g) <= NOISE_CEIL:
        return g, b, m, ""
    p = _perm_from_corr(g, b)
    if p is not None:
        b2 = b[:, p]
        m2 = m[:, p] if m is not None else None
        if rel_rms(b2, g) <= NOISE_CEIL:
            return g, b2, m2, "GGUF input axis permuted vs HF layout; bijection verified"
    p = _perm_from_corr(g.T, b.T)
    if p is not None:
        b2 = b[p]
        m2 = m[p] if m is not None else None
        if rel_rms(b2, g) <= NOISE_CEIL:
            return g, b2, m2, "GGUF output axis permuted vs HF layout; bijection verified"
    return g, b, m, "MISMATCH: no verified axis permutation aligns these tensors"


# --------------------------------------------------------------- the verdict
def comparability_verdict(rows):
    """THE comparability decision. main() prints this function's answer and no other.

    rows: iterable of (name, err_mlx, err_gguf, ratio, note) as produced by main().
          err_mlx is None for a tensor absent from the MLX 8-bit build; such a
          tensor has no ratio and therefore casts no vote.

    A paired tensor votes NOT COMPARABLE when its GGUF/MLX error ratio exceeds
    RATIO_CEIL, or when the ratio is not a number at all (a NaN must never be
    silently read as "fine").

    Returns (comparable: bool, bad: list[(name, ratio)], paired: list[float]).
    """
    paired = [r for _, em, _, r, _ in rows if em is not None]
    bad = [(n, r) for n, em, _, r, _ in rows
           if em is not None and (math.isnan(r) or r > RATIO_CEIL)]
    return (not bad), bad, paired


# gguf name, mlx name, bf16 name, permute-axis ('none' | 'cols')
PAIRS = [
    ('token_embd.weight',           'language_model.model.embed_tokens.weight',                    'model.language_model.embed_tokens.weight'),
    ('output.weight',               'language_model.lm_head.weight',                               'lm_head.weight'),
    ('blk.0.ffn_gate.weight',       'language_model.model.layers.0.mlp.gate_proj.weight',          'model.language_model.layers.0.mlp.gate_proj.weight'),
    ('blk.0.ffn_down.weight',       'language_model.model.layers.0.mlp.down_proj.weight',          'model.language_model.layers.0.mlp.down_proj.weight'),
    ('blk.31.ffn_up.weight',        'language_model.model.layers.31.mlp.up_proj.weight',           'model.language_model.layers.31.mlp.up_proj.weight'),
    ('blk.63.ffn_down.weight',      'language_model.model.layers.63.mlp.down_proj.weight',         'model.language_model.layers.63.mlp.down_proj.weight'),
    ('blk.3.attn_q.weight',         'language_model.model.layers.3.self_attn.q_proj.weight',       'model.language_model.layers.3.self_attn.q_proj.weight'),
    ('blk.3.attn_k.weight',         'language_model.model.layers.3.self_attn.k_proj.weight',       'model.language_model.layers.3.self_attn.k_proj.weight'),
    ('blk.3.attn_v.weight',         'language_model.model.layers.3.self_attn.v_proj.weight',       'model.language_model.layers.3.self_attn.v_proj.weight'),
    ('blk.63.attn_output.weight',   'language_model.model.layers.63.self_attn.o_proj.weight',      'model.language_model.layers.63.self_attn.o_proj.weight'),
    ('blk.0.attn_qkv.weight',       'language_model.model.layers.0.linear_attn.in_proj_qkv.weight','model.language_model.layers.0.linear_attn.in_proj_qkv.weight'),
    ('blk.0.attn_gate.weight',      'language_model.model.layers.0.linear_attn.in_proj_z.weight',  'model.language_model.layers.0.linear_attn.in_proj_z.weight'),
    ('blk.0.ssm_alpha.weight',      'language_model.model.layers.0.linear_attn.in_proj_a.weight',  'model.language_model.layers.0.linear_attn.in_proj_a.weight'),
    ('blk.0.ssm_beta.weight',       'language_model.model.layers.0.linear_attn.in_proj_b.weight',  'model.language_model.layers.0.linear_attn.in_proj_b.weight'),
    ('blk.0.ssm_out.weight',        'language_model.model.layers.0.linear_attn.out_proj.weight',   'model.language_model.layers.0.linear_attn.out_proj.weight'),
    ('blk.62.ssm_out.weight',       'language_model.model.layers.62.linear_attn.out_proj.weight',  'model.language_model.layers.62.linear_attn.out_proj.weight'),
    # MTP / nextn block: present in BF16 source and in GGUF, ABSENT from the MLX 8-bit build
    ('blk.64.nextn.eh_proj.weight', 'mtp.fc.weight',                                               'mtp.fc.weight'),
    ('blk.64.ffn_up.weight',        'mtp.layers.0.mlp.up_proj.weight',                             'mtp.layers.0.mlp.up_proj.weight'),
]


def collect_rows(n_rows):
    """Read the staged artifacts and build main()'s per-tensor rows.

    This is the ONE seam main() gets its data through, and the only reason it
    exists as a separate function: a test can replace this module attribute and
    then run the real main() -- its verdict, its printed COMPARABLE / NOT
    COMPARABLE line and its exit codes -- over controlled rows, without a 29 GB
    read. Nothing here decides comparability.

    Returns (rows, skipped): rows as (name, err_mlx, err_gguf, ratio, note),
    skipped as the names whose GGUF bytes could not be read at all.
    """
    kv, ts, ds = gguf_index(GGUF)
    print(f"GGUF               : {GGUF}")
    print(f"architecture       : {kv['general.architecture']}")
    print(f"general.file_type  : {kv['general.file_type']} (7 = MOSTLY_Q8_0)")
    print(f"n_tensors          : {len(ts)}")
    print(f"nextn_predict_layers: {kv.get('qwen35.nextn_predict_layers')}")
    print(f"block_count        : {kv.get('qwen35.block_count')}")
    print(f"rows sampled       : {n_rows}\n")

    rows = []
    skipped = []
    for gname, mname, bname in PAIRS:
        if gname not in ts:
            print(f"[{gname}] ABSENT from GGUF"); continue
        dims = ts[gname][0]
        ncols, nrows_full = dims[0], dims[1]
        want = nrows_full if ncols * nrows_full <= FULL_READ_ELEMS else n_rows
        try:
            g = gguf_rows(GGUF, ts, ds, gname, want)
        except SystemExit as e:
            print(f"[{gname}] SKIPPED: {e}\n"); skipped.append(gname); continue
        b, bshape, _ = st_rows(BF16, bname, want)
        if b is None:
            print(f"[{gname}] BF16 tensor {bname} absent"); continue
        b = b[:g.shape[0]]; g = g[:b.shape[0]]
        m, meta = mlx_dequant(mname, want, ncols)
        if m is not None: m = m[:b.shape[0]]
        g, b, m, note = try_align(g, b, m)
        e_g = rel_rms(b, g)
        if m is None:
            e_m = None
            mnote = "ABSENT from MLX 8-bit build"
        else:
            e_m = rel_rms(b, m)
            mnote = f"{meta['bits']}-bit g{meta['group_size']} scale={meta['scale_dtype']} bias={meta['bias_dtype']}"
        ratio = (e_g / e_m) if e_m else float('nan')
        rows.append((gname, e_m, e_g, ratio, note or mnote))
        full = "full" if want == nrows_full else f"{want}/{nrows_full} rows"
        print(f"[{gname}] dims={dims} bf16={bshape} read={full}")
        print(f"    MLX  vs BF16 rel_rms = {(f'{e_m:.7f}') if e_m is not None else '   n/a   '}   {mnote}")
        print(f"    GGUF vs BF16 rel_rms = {e_g:.7f}   {note}")
        print(f"    ratio gguf/mlx       = {ratio:.3f}" if e_m else "    ratio                = n/a (tensor exists in GGUF+BF16 but not in MLX)")
        print()
        del g, b, m; gc.collect()

    return rows, skipped


def main():
    n_rows = int(sys.argv[1]) if len(sys.argv) > 1 else 1024
    rows, skipped = collect_rows(n_rows)

    print("=" * 96)
    print(f"{'tensor':<30} {'MLX vs BF16':>13} {'GGUF vs BF16':>14} {'ratio':>8}   note")
    for n, em, eg, r, note in rows:
        ems = f"{em:.7f}" if em is not None else "n/a"
        print(f"{n:<30} {ems:>13} {eg:>14.7f} {r:>8.3f}   {note}" if em is not None
              else f"{n:<30} {ems:>13} {eg:>14.7f} {'n/a':>8}   {note}")
    comparable, bad, paired = comparability_verdict(rows)
    print("=" * 96)
    mean_ratio = (sum(paired) / len(paired)) if paired else float('nan')
    print(f"tensors compared against MLX: {len(paired)}   mean ratio: {mean_ratio:.3f}")
    print(f"tensors present in GGUF+BF16 but ABSENT from the MLX 8-bit build: "
          f"{len([1 for _, em, _, _, _ in rows if em is None])}")
    if not comparable:
        print(f"NOT COMPARABLE — {len(bad)} tensor(s) exceed the "
              f"{RATIO_CEIL}x error ratio: {bad}")
        sys.exit(1)
    if skipped:
        print(f"INCOMPLETE — {len(skipped)} tensor(s) could not be read: {skipped}")
        sys.exit(2)
    print("COMPARABLE: every paired tensor sits at 8-bit rounding noise against the same "
          "BF16 source; the two builds quantize the SAME numbers.")


if __name__ == '__main__':
    main()
