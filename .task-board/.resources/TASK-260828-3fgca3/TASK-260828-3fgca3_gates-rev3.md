# TASK-260828-3fgca3 — revision 3 gates

Candidate index tree: c856462ee5896599376503bda8dd4493611d89b3
Base: 5a287e4bcb454b53bc432cb7788a03792d1c96f6
Package: tools/mlx-swift-runtime-prototype

Every command run directly as its own process; exit codes as reported.

| gate | command | exit | result |
| --- | --- | ---: | --- |
| package build | swift build --build-tests | 0 | Build complete |
| contract suite | swift test | 0 | 351 tests / 29 suites passed |
| release product | swift build -c release | 0 | Build complete |
| production-entry smoke | scripts/benchmark-gate-smoke.sh (Release product) | 0 | 68 checks, 0 failures |
| real pair at the production entry | benchmark-run over the MLX 8-bit dir vs the 29 GB Q8_0 GGUF under the shipped decision | 0 | accepted=true |
| swift lint | xcrun swift-format lint --strict --recursive Sources Tests | 0 | clean |
| shell lint | shellcheck scripts/benchmark-gate-smoke.sh (default level) | 0 | clean, from 16 findings |
| whitespace | git diff --cached --check | 0 | clean |
| mutants | 12 applied, built, run, reverted | - | 12/12 killed, 0 survivors |

Host: no llama-server, mlx_lm, model-harness or fake-runtime process before or
after; no listener left on 18771-18800. The 28 GB model was never loaded — the
real artifacts were read only to be digested (29,047,084,416 bytes streamed at
8 MiB) and the runtimes throughout are fake-runtime.py stand-ins.
