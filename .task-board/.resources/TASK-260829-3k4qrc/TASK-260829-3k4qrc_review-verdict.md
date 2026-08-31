# TASK-260829-3k4qrc review verdict

Verdict: **changes requested**. Route: `to-dev`.

Reviewed Change Request `CR-TASK-260829-3k4qrc-1` revision 1, exact delta
`f47ffe01bb9f758fd7007aae012bfe76004c278e` to tree
`7ffd62eafc61748be5b642bcbaf2a5763af7b2a1`. The attached patch SHA-256
reproduces as `a221ef780a9fc7e510168e6e6dadd074b6fde8bf6beabb6e06fb39d6924e7fd3`.
All seven changed working-tree files byte-match the candidate blobs.

## Findings requiring rework

### F1 — The current context-policy refusal is correct, but the mismatch is fixable

The production gate correctly refused this exact pair: the sealed baseline pin
is `kv=unbounded;prefill-step=2048;reasoning=medium`, the candidate pin is
`kv=76800;prefill-step=2048;reasoning=medium`, the command exited 4, and no
`decision.json` exists.

That is not an irreducible property of MLX versus llama.cpp. The pinned Python
server exposes no `--max-kv-size`, but the same pinned `mlx-lm` already contains
`BatchGenerator.max_kv_size`, `make_prompt_cache(model, max_kv_size:)`, and
`RotatingKVCache`. The server constructs `BatchGenerator` without that argument.
For this `qwen3_5` model there is an additional integration step:
`qwen3_5.make_cache()` currently returns `KVCache()` for attention layers and
causes the generic `max_kv_size` argument to be ignored. Therefore merely adding
an argv spelling is insufficient, but bounded KV is implementable in the Python
baseline rather than forbidden by the runtime model.

Required rework: add and production-test a real 76,800-token bound through the
pinned Python server and `qwen3_5` cache construction, have the live attestation
derive `kv=76800`, then rerun the complete sequential six-scenario pair. Keep
the current refusal as valid historical evidence, not the final scored number
set. Record this correction in `LOGBOOK.md`.

### F2 — The 5-second sampler has no evidence that it captures a peak

`BenchmarkFootprintSampler.loop()` runs blocking `vmmap -summary` and only then
sleeps five seconds. The effective period is therefore reader duration plus
five seconds. From the sealed run duration and process sample counts, after
subtracting six end-of-scenario samples, the observed effective periodic
interval is approximately 7.76 seconds for Python and 7.32 seconds for
llama.cpp. `beginWindow()` only clears arrays; it takes no synchronous start
sample. `capturePeaks()` takes one synchronous sample at the end.

The added test checks only that a constant equals 5 and exceeds 0.25. It does
not drive `benchmark-run`, create a transient resident-memory spike shorter
than the effective cadence, or prove that the production record captures it.
Candidate `short_prompt` and `tool_call` windows contain only two samples each.
There are no sample timestamps or raw series from which absence of a missed
peak can be established. This is the standard negative shape **capability claim
that does not reproduce**: the artifact calls a sampled maximum a peak without
evidence against the miss path. A faster-spiking runtime can be understated,
creating the new directional bias named in the review brief.

Required rework: make the measurement contract incapable of silently missing
the claimed peak, or name it as a sampled observation and refuse peak scoring.
Add a production-entry negative fixture with a sub-cadence transient allocation
that kills the current implementation, then prove the replacement. Preserve a
bounded observer cost and rerun the full pair after the fix. Record the finding
and resolution in `LOGBOOK.md`.

### F3 — The report does not enumerate every sealed asymmetry

The report's numbered section includes the llama.cpp per-slot KV reuse clause,
but omits the baseline-only sealed declaration
`deployed with --prompt-cache-size 1 --prompt-cache-bytes 8GB`. Since the task
requires every non-comparable dimension to be enumerated rather than dropped,
the final report must present both runtime cache policies and their likely
direction/unknown effect together.

## Verified facts and non-findings

- One foreground invocation ran Python first and llama.cpp second. Sealed
  intervals do not overlap: Python finished at `1788037729.522231`; llama.cpp
  started at `1788037750.100650` (20.58 seconds later). The report states the
  start host had no other loaded model; an idle Ollama service held no model.
- Both records contain all six scenarios, all succeeded, and prompt tokens are
  exactly equal by scenario: 41, 7,784, 313, 7,784, 910, and 73,016. Tool-call
  production paths succeeded and both soak runs completed 20 iterations.
- Both pins record `speculation=off`; llama.cpp also launched with
  `--spec-type none`. No with-MTP number is present.
- The review brief's decode-unit premise does not reproduce. The sealed field is
  `decodeTokensPerSecond`, not milliseconds per token: Python 6.598266 versus
  llama.cpp 7.879572 is +19.4188% output throughput for llama.cpp. Converted to
  latency it is 151.555 versus 126.910 ms/token, or -16.2611% time per token.
  The producer report explicitly labels `tok/s` and says higher is better, so
  its faster direction is correct.
- Long-prompt TTFT is 111.401846 seconds Python versus 69.451095 seconds
  llama.cpp, -37.6571%. The sealed unit is seconds, not milliseconds.
- At 73,016 prompt tokens the recorded sampled resident upper bounds are
  47,791,331,280 bytes Python and 45,097,521,165 bytes llama.cpp, -5.6366%.
  The arithmetic is correct; F2 prevents treating the sampled maxima as proven
  peaks for a decision.
- B7 is recorded in the task report and candidate `LOGBOOK.md` as
  `model-harness` PID 40394 plus Python child 40395 retaining about 31 GiB until
  external process-group signalling. The successor spawn brief also carries
  those exact observed details. The evidence archive does not include a raw
  process-list snapshot from the survivor moment, so this review confirms the
  occurrence was recorded, not an independent reconstruction of it.

## Validation

- Reviewer focused test: `swift test -c release --filter
  productionVMMapCadenceIsBounded` — pass, 1 test / 1 suite.
- Reviewer full suite: `swift test -c release` — pass, 402 tests / 32 suites.
- Producer evidence: Release build, strict swift-format lint, and
  `git diff --check` exit 0; CR validation reran `go test ./... -count=1` and
  `go vet ./...`, both exit 0.
- Green tests do not cure F2: the required transient-spike production path is
  absent, so the positive constant test is not peak-capture evidence.

No repository code was modified by the reviewer. `accept_cr` was not called.
