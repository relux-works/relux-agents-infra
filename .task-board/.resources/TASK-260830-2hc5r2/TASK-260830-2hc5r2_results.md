# TASK-260830-2hc5r2 — bounded Python KV baseline

## Result

The pinned Python `mlx-lm` fork now constructs a real `RotatingKVCache` for
Qwen3.5 full-attention layers when `--max-kv-size 76800` is supplied. The
linear-attention `ArraysCache` layers and the unbounded default remain
unchanged.

The live `/v1/models` attestation is not copied from argv. Before any cache was
constructed, the exact configured model entry had no `meta` field despite the
server having been launched with `--max-kv-size 76800`. After the seeded
production generation path created the prompt cache, the same running server
reported:

```json
{"meta":{"n_ctx":76800}}
```

The server derives that value from the actual `RotatingKVCache.max_size`
instances. A production-path negative test supplies an unbounded `KVCache`
under the same configured flag and requires the attested bound to remain absent.

## Source changes

External source fork `/Users/alexis/src/relux-works/mlx-lm` at baseline commit
`b0a45b8fdd3bb5d6c390d65a6f8521c296f980ec`, branch
`fix/generation-health-readiness`:

- `mlx_lm/models/cache.py`
- `mlx_lm/models/qwen3_5.py`
- `mlx_lm/server.py`
- `tests/test_models.py`
- `tests/test_server.py`

Story worktree baseline commit
`f47ffe01bb9f758fd7007aae012bfe76004c278e`:

- benchmark-only Python argv now includes `--max-kv-size 76800`
- benchmark-only llama.cpp argv now includes `--ctx-size 76800`
- live context-policy and negative gate tests cover the matching bound and an
  ignored Python bound
- attestation documentation distinguishes the active bounded Python server
  from the unchanged unbounded deployed default

`examples/model-harness.benchmark.toml` contains no `[profiles.qwen-local]`
section. The deployed default profile was not changed.

## Live 73k+ correctness evidence

Server launch used the exact local model, `--prefill-step-size 2048`,
`--max-kv-size 76800`, seeded requests, and `reasoning_effort=medium`.

1. A first 73,015-token request with `max_tokens=16` exited 0 after 844.365s,
   but ended `finish_reason=length` with `content=null`. This is retained as an
   expected-insufficient attempt and is not counted as correctness evidence.
2. A second 73,015-token request with `max_tokens=96` exited 0 after 858.027s.
   Its reasoning correctly recovered all three systems, but public content was
   truncated after two names. This attempt is also not counted as the final
   correctness proof.
3. A continuation of that same cached conversation had 73,139 prompt tokens,
   of which the running server reported 73,111 as cached. It exited 0 in 9.951s
   with `finish_reason=stop`, 67 completion tokens, and exact public content:

   > The coolant loop, the intake manifold, and the telemetry uplink were serviced.

   Usage was `prompt_tokens=73139`, `cached_tokens=73111`,
   `completion_tokens=67`, `total_tokens=73206`. The server log independently
   showed only 28 suffix tokens processed, confirming that the 73,111-token
   bounded cache, rather than a shortened prompt, supplied the long context.

After this response, the exact live model entry still reported
`meta.n_ctx=76800`. Server and client processes exited 0; port 18041 and matching
process probes were empty after teardown.

## Verification

All commands below were run directly and their real exit codes observed.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Qwen bounded-cache test before implementation | 1 | Expected red: returned `KVCache`, not `RotatingKVCache` |
| `python -m unittest tests.test_models tests.test_server` | 0 | 114 tests, 1 skip |
| `python -m isort --profile black --check-only ...` | 0 | five changed Python/test files |
| `python -m black --check ...` | 0 | five files unchanged |
| `python -m pip install --no-deps --editable .` | 0 | editable wheel built and installed |
| `swift test -c release` | 0 | 403 tests in 32 suites |
| `swift build -c release` | 0 | Release build |
| `swift format lint --strict --recursive Sources Tests` | 0 | strict lint clean |
| `benchmark-gate-smoke.sh` with absolute `BINARY` and `OUT` | 0 | `BENCHMARK GATE SMOKE OK (0 failures)` |
| Story and external-fork `git diff --check` | 0 | no whitespace errors |

Two invocation mistakes remain visible rather than rewritten as passes:

- smoke without `BINARY` exited 1 with the script's required-variable error;
- smoke with the default relative `OUT` exited 1 because model-harness requires
  an absolute config path. The final task-scoped absolute `OUT` run is the green
  gate above.

Plain `isort --check-only` also exited 1 because its default wrapping conflicts
with Black. The final compatible `--profile black` check and Black check both
exit 0.

The pipx test environment gained task-local development dependencies
`requests==2.32.5`, `black==25.1.0`, and `isort==6.0.0`; no project dependency
manifest was changed.
