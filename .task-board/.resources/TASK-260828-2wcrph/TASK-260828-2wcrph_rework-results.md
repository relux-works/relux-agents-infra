# TASK-260828-2wcrph rework results

Date: 2026-08-29
Role: developer
Host: MacBookPro18,2 / Apple M1 Max / 64 GiB

## Outcome

The unsupported paired decode claim is withdrawn. The replacement probe sent
`stream_options: {"include_usage": true}` and retained every timestamped SSE
packet. Both runtimes returned authoritative and exactly equal counts:

| Metric | Python mlx-lm | llama.cpp | Candidate / baseline |
| --- | ---: | ---: | ---: |
| Prompt tokens | 7,784 | 7,784 | 1.0000 |
| Completion tokens | 106 | 106 | 1.0000 |
| Total tokens | 7,890 | 7,890 | 1.0000 |
| TTFT | 103.310s | 76.784s | 0.743x |
| Prefill | 75.346 tok/s | 101.375 tok/s | 1.345x |
| Decode | 8.449 tok/s | 8.772 tok/s | 1.038x |

The runtimes ran sequentially on the same host. A process check before the
first runtime found no model server. llama.cpp was stopped before Python mlx-lm
started. A final process check found no `llama-server`,
`mlx_lm-relux.server --model`, or `mlx-swift-runtime-prototype serve` process.

The 1.038x decode delta is one bounded observation, not a general direction.
The prior three llama.cpp runtime logs report 10.51, 10.79 and 10.74 tok/s for
their own 106-token generations; the rerun server reported 8.69 tok/s while the
client interval reported 8.772 tok/s. No suite-level decode direction is claimed.

## Evidence contract

- `probe_decode_usage.py` derives decode only from `completion_tokens`.
- Empty, malformed, inconsistent, or non-positive usage leaves decode
  `unknown`; four synthetic negative tests exercise those branches.
- SSE frame counts are recorded as transport diagnostics and never used as
  token counts.
- The raw streams each contain one non-empty usage packet at frame 105. Extra
  `prompt_tokens_details` metadata is retained raw; the summary normalizes only
  the three authoritative count fields.
- The first independent verifier incorrectly required raw usage to equal the
  normalized summary byte-for-byte and raised `AssertionError`; a later shell
  command masked that assertion with a trailing successful diagnostic. That
  invocation is not counted as gate evidence. The corrected verifier ran as a
  standalone process, normalized the three count fields, independently
  recomputed both rates, and exited 0.

Raw evidence SHA-256:

- `llamacpp-authoritative-raw.json`:
  `7cff0ebc1771ff6bd9a10c4f61eff9b3494796bcafe105c85efdeae5055973dd`
- `llamacpp-authoritative-summary.json`:
  `a67ae58e677b4ece6125c9dce5d199b6473ef799e1b98c0e17f99808b09a9633`
- `python-mlx-authoritative-raw.json`:
  `e38a7020afd6c343acd61dadec6bdbd3996e03faac9c9ad43cccb955681829c3`
- `python-mlx-authoritative-summary.json`:
  `cc58ad92de6b3bb720575d1724754763617fcfa42f5c8f778061dcb46f62c70a`

## Decision boundary

The known production gate defect remains explicitly excluded. Omitting
`reasoning_content` has mixed direction: it inflates llama.cpp TTFT, understates
prefill, and overstates decode. All gate-recorded llama.cpp TTFT/prefill/decode
values remain outside the migration decision. TASK-260829-3cwcb6 owns the
production fix and full rerun.

The independently confirmed evidence from revision 1 remains unchanged:

- all three production pairs exited 4 on exact `contextPolicy` mismatch and
  produced no scored decision;
- all six scenarios had exact prompt-token parity;
- `context_75k` served 73,016 prompt tokens on each runtime;
- tool-call parity and the 20-iteration soak passed on both;
- wall clock was 0.81x on `long_prompt_8k` and 0.97x at capacity;
- capacity peak physical footprint was 45.20 GiB for Python mlx-lm and a
  conservative 41.05 GiB upper bound for llama.cpp;
- small-working-set memory remains indeterminate.

## Validation

Every counted gate below ran as a standalone process.

| Command | Exit | Result |
| --- | ---: | --- |
| `python3 -m unittest -v test_probe_decode_usage.py` | 0 | 4 tests passed; missing/malformed/inconsistent usage refuses a rate |
| `python3 -m py_compile probe_decode_usage.py test_probe_decode_usage.py` | 0 | Probe and negative tests compile |
| standalone raw-evidence verifier | 0 | Both usage packets verified; decode independently recomputed as 8.771963 / 8.449346 |
| `swift test -c release --filter RuntimeObservationReadingTests` | 0 | 21 tests passed |
| `swift test -c release --filter RuntimeBenchmarkContextBoundTests` | 0 | 19 tests passed |
| `swift test -c release` | 0 | 392 tests / 30 suites passed |
| `swift build -c release` | 0 | Build succeeded; three pre-existing deprecation warnings |
| supported macOS `xcodebuild build ... -configuration Release` | 0 | `BUILD SUCCEEDED`; runnable product plus metallib |
| `swift format lint --strict --recursive Sources Tests` | 0 | Clean |
| `shellcheck scripts/benchmark-gate-smoke.sh` | 0 | Clean |
| `benchmark-gate-smoke.sh` with Xcode product and absolute task-scoped `OUT` | 0 | `BENCHMARK GATE SMOKE OK (0 failures)`; positive and negative production paths exercised |
| unsupported-claim search across report, README and logbook | 0 | Superseded paired figures and `decodes far faster` attribution absent |
| `git diff --check` | 0 | Clean |

Smoke setup failures are reported, not counted as green:

- no `BINARY`: exit 1, required environment missing;
- SwiftPM Release product with default relative `OUT`: exit 1;
- Xcode Release product with default relative `OUT`: exit 1.

The last two failed because current `model-harness` refuses the script's
relative generated config path (`config path must be absolute`). The successful
run matched the previous reviewer invocation by setting an absolute task-scoped
`OUT`; no source behavior was changed to manufacture the pass.
