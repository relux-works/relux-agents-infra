# TASK-260830-2hc5r2 rework results

## Outcome

Both reviewer findings are addressed before the comparison rerun.

### F1 — mixed cache attestation fails closed

- Source: `/Users/alexis/src/relux-works/mlx-lm/mlx_lm/server.py`.
- `_cache_max_kv_size` ignores only recognized non-attention `ArraysCache` state.
- A finite attestation now requires every other active cache entry to be a
  `RotatingKVCache` or `BatchRotatingKVCache` with one identical `max_size`.
- An unbounded, mixed, partial, differently bounded, or unrecognized cache
  returns `None`.
- Production-entry negative:
  `TestResponseGeneratorKVBound.test_seeded_single_generation_does_not_attest_mixed_target_and_draft_caches`
  drives `ResponseGenerator._serve_single` with a bounded target and unbounded
  draft, then drives `APIHandler.handle_models_request`; `/v1/models` omits
  `meta`.

### F2 — immutable source and benchmark-only install

- Source repository revision:
  `ec9eea0af1d741cd4eb21c8766478a3e79dd44d6` on
  `fix/generation-health-readiness`.
- The commit contains exactly the five mlx-lm implementation/test files and the
  source repository is clean after the commit.
- Isolated pipx environment:
  `mlx-lm-kv76800-ec9eea0`.
- Benchmark entry point:
  `/Users/alexis/.local/bin/mlx_lm-kv76800-ec9eea0.server`.
- Benchmark provenance interpreter:
  `/Users/alexis/.local/pipx/venvs/mlx-lm-kv76800-ec9eea0/bin/python`.
- Imported package path is inside that isolated environment, not the editable
  source checkout.
- Its `direct_url.json` records the full source OID as both
  `vcs_info.commit_id` and `requested_revision`.
- The benchmark-only `qwen-benchmark-python` profile pins this exact suffixed
  entry point and `--max-kv-size 76800`. The shipped test refuses the previous
  `mlx_lm-relux.server` executable assignment. No `[profiles.qwen-local]` is
  present in the benchmark config.
- The installed entry-point SHA-256 is
  `4dfbd73ede6d4978d74c916fd4ed95c58d14df806156908f105ee0cd096247b0`.
- The built wheel SHA-256 is
  `e65442f98e626a1f4577ef2a36c45991a23a4473e48de323cfbf4e97e40f772b`.

## Validation run in this rework

| Command / gate | Exit | Result |
| --- | ---: | --- |
| Targeted `TestResponseGeneratorKVBound` | 0 | 4 tests pass, including mixed target/draft production negative |
| `python -m unittest tests.test_models tests.test_server` | 0 | 115 tests, 1 skip |
| Black check over five mlx-lm files | 0 | clean |
| isort `--profile black --check-only` over five mlx-lm files | 0 | clean |
| mlx-lm `git diff --check` before commit | 0 | clean |
| `python -m pip wheel --no-deps .` | 0 | `mlx_lm-0.32.0-py3-none-any.whl` built |
| mlx-lm `git diff --check`, status, and revision after commit | 0 | clean tree at `ec9eea0…` |
| pipx PEP 508 install from `git+file://…@ec9eea0…` | 0 | isolated environment installed |
| Isolated `direct_url.json` read and server `--help` | 0 | exact OID recorded; `--max-kv-size` exposed |
| `model-harness render qwen-benchmark-python` | 0 | exact isolated entry point and KV bound rendered |
| Final `swift test -c release` | 0 | 403 tests in 32 suites |
| macOS arm64 Xcode Release build | 0 | `BUILD SUCCEEDED` |
| strict recursive Swift format lint | 0 | clean |
| shellcheck for production benchmark smoke | 0 | clean |
| production `benchmark-gate-smoke.sh` | 0 | `BENCHMARK GATE SMOKE OK (0 failures)` |
| Story worktree `git diff --check` | 0 | clean |

## Red attempts retained honestly

- `python -m build --wheel`: exit 1 because the validation venv has no `build`
  module. The successful `pip wheel` build above is the replacement build gate.
- Bare local VCS spelling passed to pipx: exit 1, package spec parse refusal.
  The PEP 508 direct-reference spelling installed successfully.
- First full Swift rerun: exit 1 because the new negative assertion rejected a
  comment mentioning the deployed wrapper, not only the old executable
  assignment. The assertion was narrowed to the assignment and the full suite
  then passed.
- Read-only `model-harness render qwen-local`: exit 1 because no default config
  exists at the CLI's default path. It is not presented as deployed-profile
  evidence and no default config was created or changed.

## Evidence accepted from the already attached revision-1 run

The 28 GB model and 73k correctness generation were not rerun in this rework.
The existing task outcome and reviewer verdict retain the raw production
response: 73,139 prompt tokens, 73,111 cached tokens, 67 completion tokens,
`finish_reason=stop`, the correct three-system answer, and live
`meta.n_ctx=76800`. The reviewer independently confirmed that evidence. The
immutable commit contains that same cache-construction implementation plus the
fail-closed mixed-cache correction.

The full Python-vs-llama.cpp comparison was deliberately not rerun here. This
rework establishes the two prerequisites the comparison must inherit; the
comparison rerun belongs after review accepts this revision.

## Evidence files

- `TASK-260830-2hc5r2_python-tests-02.log`
- `TASK-260830-2hc5r2_swift-tests-03.log`
- `TASK-260830-2hc5r2_xcode-build-02.log`
- `TASK-260830-2hc5r2_benchmark-smoke-02.log`
