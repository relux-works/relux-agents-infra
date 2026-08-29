# Qwen3.8-27B-Uncensored — GGUF Q8_0 (llama.cpp)

Staged by TASK-260828-3g87i4 (story STORY-260828-2faxgm) on 2026-08-28 as the
llama.cpp-side counterpart to `../Qwen3.8-27B-Uncensored-MLX-8bit`.

This is **not** a default for anything. `profiles.qwen-local` stays on Python
`mlx_lm.server`; nothing here changes runtime defaults.

## Files

| File | Bytes | SHA-256 |
|---|---:|---|
| `Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf` | 29 047 084 416 | `31756fca94beca71ea4b8706d6fdc896dab2a3c6376ab0c1863b98512a24f8d6` |
| `mmproj-Qwen3.8-27B-Uncensored-OrcaRouter-F16.gguf` | 931 145 984 | `add205b7bfdb3f71f6da36b0a82aa20928dd829a920878c602628cdfbebc5288` |

Both hashes were recomputed locally over the whole file (`shasum -a 256`, rerun
2026-08-28 at revision 2) and match the mirror's `SHA256SUMS.txt` and
`MANIFEST.json` **and the first-party repo's published LFS digests** — see below.

Despite the `F16` in its name the mmproj is actually **BF16** (`general.file_type = 32`;
110 BF16 + 224 F32 tensors).

## Source

Repository: <https://huggingface.co/chimingw/Qwen3.8-27B-Uncensored-OrcaRouter-GGUF>
Repo revision at download: `58ebd123013160600229eda180b5b17f3fb7af9d`

```bash
DEST=/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0
BASE=https://huggingface.co/chimingw/Qwen3.8-27B-Uncensored-OrcaRouter-GGUF/resolve/main
mkdir -p "$DEST"
curl -SL -o "$DEST/Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf" \
  "$BASE/Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf"
curl -SL -o "$DEST/mmproj-Qwen3.8-27B-Uncensored-OrcaRouter-F16.gguf" \
  "$BASE/AUX/mmproj-Qwen3.8-27B-Uncensored-OrcaRouter-F16.gguf"
```

### Why not the first-party repo — and why the bytes are still first-party

`orcarouter/Qwen3.8-27B-Uncensored-GGUF` is the publisher of the BF16 and MLX
builds already staged here, and would have been the obvious source. It is a
**gated** repo (`gated: auto`); its blobs return HTTP 401 `GatedRepo` and this
host has no Hugging Face token.

`chimingw/...-OrcaRouter-GGUF` is ungated and its `MANIFEST.json` records that
each quant was produced by `llama-quantize` **directly from the same
`Qwen3.8-27B-Uncensored-F16-00001-of-00002.gguf`** published by orcarouter:

```
llama-quantize .../Qwen3.8-27B-Uncensored-F16-00001-of-00002.gguf \
  .../Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf.partial Q8_0 64
```

**The staged files are byte-identical to the first-party artifacts.** Gating
blocks the *blob*, not the *metadata*: the first-party model endpoint is public
and publishes each file's LFS SHA-256.

```bash
curl -s 'https://huggingface.co/api/models/orcarouter/Qwen3.8-27B-Uncensored-GGUF?blobs=true'
# HTTP 200, revision a855f377abf5cbda99a278414466743f427e97c8
#   Qwen3.8-27B-Uncensored-Q8_0.gguf        29047084416  lfs.sha256 31756fca...24f8d6
#   mmproj-Qwen3.8-27B-Uncensored-f16.gguf    931145984  lfs.sha256 add205b7...bc5288

shasum -a 256 Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf \
              mmproj-Qwen3.8-27B-Uncensored-OrcaRouter-F16.gguf
#   31756fca...24f8d6   add205b7...bc5288   -> identical
```

Both digests match exactly, for the text model and for the mmproj. So content
identity to the first-party artifacts is **established under SHA-256**; an
earlier revision of this file called it unverifiable, which was wrong.

What is still true is that the gated blob itself is **not downloadable** from
this host without authorization (`HTTP 401`, reproduced 2026-08-28). Rebuilding
this staging step from the first-party repo requires a token; from the mirror it
does not, and either way you can check you got the same bytes.

Independently of provenance, the staged weights were also verified numerically:
they match the local BF16 source of record at 8-bit rounding noise, mean error
ratio 0.766 against the MLX 8-bit baseline
(see `TASK-260828-3g87i4_quantization-equivalence.md`).

## Runtime

Homebrew `llama.cpp` 0.3.0 — upstream tag `v0.3.0`, commit
`c1d0e7a004015f23bc0233470b747b596f29b264`, build **10621**, released 2026-08-25.
Pinned with `brew pin llama.cpp`.

```bash
llama-server -m Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf \
  --host 127.0.0.1 --port 18901 -c 4096 --jinja
```

Add `--mmproj mmproj-Qwen3.8-27B-Uncensored-OrcaRouter-F16.gguf` for vision;
without it no vision weights are loaded.

This is **not** a memory difference against the MLX build. The MLX checkpoint
embeds its vision tower (921 460 192 bytes) but `mlx_lm` drops those keys in
`Model.sanitize` before `load_weights` and before `mx.eval`, so a text-only load
makes none of them resident — measured at 28 579 478 528 resident parameter
bytes and zero resident vision tensors. Placement differs; residency does not.
See §5.2 of `TASK-260828-3g87i4_quantization-equivalence.md`.
