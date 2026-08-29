# TASK-260827-2v13w8 — MLX Swift runtime migration: benchmark and decision

**Revision 4.** Supersedes revisions 1, 2 and 3. Every measurement below was
taken by **one invocation of one binary** — SHA-256
`8a517b10e6a74793dd47d33d07b1b08275863f3fb7e8cfb11880a14b71014f91` — which
launched both runtimes through `model-harness`, drove every scenario itself,
sampled and timed them itself, sealed what it measured and judged it, without
ever leaving the process. For the candidate pass that same file is also the
runtime that served. There is no argument to the production entry point through
which a measurement can be supplied.

## The decision

**REJECT. Python `mlx-lm` remains the default local Qwen runtime.**
`accepted=false`, **one** blocker. No installed configuration was changed:
`profiles.qwen-local` in the operator's own model-harness config still points at
`mlx_lm.server`.

| Metric | Python | Swift | Swift/Python | Bar | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| 8k scenario-local peak footprint | 31.39 GiB | 36.14 GiB | **1.151x** | `<= 1.10` | outside |

Everything else scored is inside its band, including the whole-process peak
(1.092x), which is comparable because both passes completed the same parity work.

## What changed since revision 3

### R3-A — the forgery class, closed by construction rather than by policing

Round 3 found, for the third time, that the production entry could be made to
return `accepted=true, exit 0` without anything having been served. The third
attack used no tampering at all: review started two placeholder HTTP servers
that answered `GET /v1/models`, had the shipped `benchmark-attest` command
observe them — correctly; the processes were real — typed the measurements into
a record, and got an acceptance in 7.2 seconds.

The common shape across all three rounds was a **seam**: some other program
measured, and the gate was asked to believe a document about it. Revision 4
removes the seam.

- `benchmark-attest` **no longer exists**. No caller can direct this binary to
  observe a process of the caller's choosing. The only code that writes an
  attestation is the code that spawned the process it describes.
- `benchmark-run` is the production entry point for the decision. It launches,
  drives, measures, records, seals and judges in one invocation. Every metric in
  a record comes from an HTTP exchange this process performed, and each
  scenario's transcript — request path, body digests, byte counts, status and
  the wall-clock instants of the exchange — is sealed into the attestation.
- `benchmark-compare` is demoted to **replay** and **cannot return an
  acceptance at all**. Its best outcome is a reproduced rejection, exit 3. That
  cap is not cosmetic: a command that can turn files on disk into
  `accepted=true, exit 0` is the exact bypass review found three times, and a
  replay that can only confirm or refuse cannot be it.

Review's own reproduction is now a case in the acceptance suite. Two live
placeholder runtimes that answer `/v1/models` and serve nothing are driven for
real by the production subcommand, and the result is **exit 4** with a refusal
that names what was missing:

```
PASS  a pass whose runtimes only listed models is inadmissible (exit 4)
PASS  the refusal names what was missing rather than scoring it
PASS  the placeholder pass produced no decision at all
```

### Where the boundary actually is — measured, not asserted

`BenchmarkRunCommand`'s own documentation says a modified build of this binary
can report anything it likes. That sentence was tested rather than left as a
disclaimer. Three limit probes replaced the real scenario driver with fabricated
results inside the production path, keeping the launch, the observation and the
seal genuine:

| Probe | Fabrication | Outcome |
| --- | --- | --- |
| **L-1** | plausible measurements, **no** transcript | refused — `admit` requires every scenario to carry the exchanges it came from |
| **L-2** | measurements **with** a synthetic transcript, wall clock 0.01 s/scenario | refused — `record claims 0.03s of scenario wall clock, but the gate watched the runtime for only 0.0159s` |
| **L-3** | same, wall clock scaled to **0.0005 s/scenario** | **accepted, exit 0** |

L-3 is reported as a survivor. The suite did go red on L-3, but through an
unrelated downstream fixture, and a red suite for the wrong reason is not
detection. The honest statement of the boundary is therefore:

> This construction closes the **ordinary-caller** forgery class — that was
> R3-A, and it is closed. It does **not** close the **modified-build** class. A
> forger who edits and rebuilds this binary, fabricates the transcript as well
> as the numbers, and scales the fabricated wall clock below the residual
> interval a no-work pass still occupies, obtains an acceptance. What stands
> between a fabrication and an acceptance is arithmetic consistency with an
> observed interval, not any ability to tell a measured number from a typed one.

No further hardening was attempted, and that is a deliberate stop rather than an
omission: every additional clause would raise the cost of the same attack
without changing its class, which is what rounds 1 to 3 already demonstrated
three times.

## Reproducibility — read this before reading the numbers

Revision 3 reported three blockers, all on the 8k scenario. Two of them do not
reproduce.

| 8k metric | rev3 Python | rev4 Python | rev3 Swift | rev4 Swift | rev3 ratio | rev4 ratio |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| TTFT (s) | 76.974 | 92.950 | 93.142 | 92.978 | 1.210x | **1.000x** |
| prefill (tok/s) | 101.126 | 83.744 | 83.572 | 83.718 | 0.826x | **1.000x** |
| scenario peak (B) | 33,875,940,256 | 33,705,153,440 | 38,759,076,720 | 38,801,396,520 | 1.144x | **1.151x** |

The candidate reproduced itself to within **0.2%** on all three. The incumbent
came out **21% slower** on its own 8k prefill than the same runtime, same
config, same prompt policy one revision earlier. Nothing in the record explains
it: the harness now samples host load per scenario, and the 1-minute load
average inside the two 8k windows was **10.18** (Python) against **8.27**
(Swift) — comparable, and not the 3x contamination signature that ruined the
first revision-4 attempt.

Consequences, stated plainly:

1. **The revision-3 throughput blockers are withdrawn.** They were single-run
   measurements and they did not reproduce. Nothing in this decision rests on
   them.
2. **Single-run throughput on this host is worth about ±20%** and should not be
   read more finely than that. The suite runs each scenario once per runtime;
   repeats were not run, and that is a stated gap, not a claim of stability.
3. **The surviving blocker is the stable one.** The 8k footprint ratio
   reproduced at 1.144x and 1.151x across two independent full reruns, on two
   different builds, weeks-apart page-cache states, and both times outside the
   1.10 bar. The decision rests on the metric that repeated.

## Capacity — recorded, not scored

Both runtimes serve the 73,016-token prompt on this host.

| `context_75k` | Python | Swift | Swift/Python |
| --- | ---: | ---: | ---: |
| time to first token (s) | 971.41 | 1504.45 | 1.549x |
| prefill (tok/s) | 75.16 | 48.53 | 0.646x |
| decode (tok/s) | 7.93 | 1.12 | 0.141x |
| scenario peak footprint | 45.20 GiB | 49.36 GiB | 1.092x |

Not scored, because it is a capability question and both runtimes have the
capability; reported, because "both pass" is not "both are equal". The Swift
decode figure here is **16 completion tokens measured under a 1-minute host load
average of 35.21** at 49.4 GiB resident on a 64 GiB machine — memory pressure
the scenario creates for itself. It should be read as "much slower", not as
1.12 tok/s. Revision 3 measured the same scenario at 4.94 tok/s.

## What the candidate wins, reported because a rejection that hides them is an argument

- `short_prompt` on every axis: TTFT 0.945x, prefill 1.058x, decode 1.061x,
  footprint 0.990x.
- 8k decode throughput **1.083x**, and 8k TTFT and prefill at parity (1.000x).
- Soak: median latency **6.62 s** against **7.28 s**, aggregate output
  **9.58 tok/s** against **8.70 tok/s**. Both completed **20/20** with zero
  failures — a bounded pass for both, not a win for either.
- Time to a *served* completion after launch: **7.02 s** against **10.97 s**.
  The Swift runtime answers `/v1/models` with `503` until the weights are
  resident; `mlx_lm.server` answers `200` in 2.9 s with nothing loaded, so the
  faster listing number means less.
- Whole pass wall clock is 1.316x **against** the candidate, and that is almost
  entirely `context_75k`.

## Concrete blockers to reopening the migration

1. **8k scenario-local peak footprint 1.151x** — 36.14 GiB against 31.39 GiB for
   the same 7,784-token prompt, reproduced across two revisions. Needs
   `<= 1.10x`. This is the whole of the rejection.
2. **No prompt cache.** Declared rather than tuned away, because the incumbent
   is measured as deployed with `--prompt-cache-size 1 --prompt-cache-bytes 8GB`
   while the prototype builds a fresh KV cache per request. Visible in
   `multiturn_prefix_reuse` as a 1.353x scenario footprint. Not scored — the two
   runtimes are not doing the same work there — and not a defect, but work the
   migration would have to absorb.
3. **`context_75k` is 1.55x slower end to end.** Not scored and not a bar, but a
   caller would notice it.
4. **Throughput evidence is single-run.** Before any future accept, the scored
   scenarios need repeats; this revision demonstrated why by failing to
   reproduce two of revision 3's three blockers.

None of the four needs upstream work.

## Contract limit carried forward (R2-D)

An MLX error raised on an MLX-owned `asyncEval` thread reaches MLX's global
default handler and **traps the process**. `MLX.withError` in
`GenerationEngine.run` converts only errors delivered to the awaiting task.
So TASK-260827-2h39ya's in-process `503` and TASK-260827-2q77g8's batch-release
attestation are reachable **only** for failures delivered as throws; otherwise
recovery begins when the supervisor observes the process died. This is written
into `GenerationWorkerHealth` and `GenerationBatchRecovery`'s own documentation
and is unchanged by this revision.

## Not diagnosed, and stated as not diagnosed

- **`tool_call` renders 313 Python tokens against 296 Swift** for the same
  request — 1.057x, inside the `maxPromptTokenSkewRatio` band, so not a blocker,
  but a real difference in how the two runtimes serialize the tool schema into
  the template. The scenario is scored for parity only, which both passed.
- **Why the incumbent's 8k prefill fell 21% between revisions** (see
  Reproducibility). The host-load record does not explain it.
- **Soak footprint drift** is +75.7 MB on Swift and −495.4 MB on Python. Both
  bounded; Python's negative drift is consistent with prompt-cache eviction.

## Scenario outcomes

| Scenario | Python | Swift | Python prompt tok | Swift prompt tok |
| --- | --- | --- | ---: | ---: |
| `short_prompt` | ok | ok | 41 | 41 |
| `long_prompt_8k` | ok | ok | 7784 | 7784 |
| `tool_call` | ok | ok | 313 | 296 |
| `multiturn_prefix_reuse` | ok | ok | 7784 | 7784 |
| `stability_soak` | ok | ok | 910 | 910 |
| `context_75k` | ok | ok | 73016 | 73016 |

## Per-scenario measurements

| Scenario | Metric | Python | Swift | Swift/Python |
| --- | --- | ---: | ---: | ---: |
| `short_prompt` | TTFT (s) | 0.985 | 0.931 | 0.945x |
| `short_prompt` | prefill tok/s | 41.630 | 44.054 | 1.058x |
| `short_prompt` | decode tok/s | 10.271 | 10.897 | 1.061x |
| `short_prompt` | wall clock (s) | 6.145 | 5.703 | 0.928x |
| `short_prompt` | scenario peak footprint (B) | 29,399,274,400 | 29,116,781,352 | 0.990x |
| `long_prompt_8k` | TTFT (s) | 92.950 | 92.978 | 1.000x |
| `long_prompt_8k` | prefill tok/s | 83.744 | 83.718 | 1.000x |
| `long_prompt_8k` | decode tok/s | 9.723 | 10.533 | 1.083x |
| `long_prompt_8k` | wall clock (s) | 103.749 | 102.852 | 0.991x |
| `long_prompt_8k` | scenario peak footprint (B) | 33,705,153,440 | 38,801,396,520 | 1.151x |
| `tool_call` | TTFT (s) | unmeasured | unmeasured | n/a |
| `tool_call` | prefill tok/s | unmeasured | unmeasured | n/a |
| `tool_call` | decode tok/s | unmeasured | unmeasured | n/a |
| `tool_call` | wall clock (s) | 11.053 | 9.822 | 0.889x |
| `tool_call` | scenario peak footprint (B) | 30,492,005,280 | 39,657,935,656 | 1.301x |
| `multiturn_prefix_reuse` | TTFT (s) | 95.331 | 93.798 | 0.984x |
| `multiturn_prefix_reuse` | prefill tok/s | 74.767 | 87.232 | 1.167x |
| `multiturn_prefix_reuse` | decode tok/s | unmeasured | unmeasured | n/a |
| `multiturn_prefix_reuse` | wall clock (s) | 317.145 | 291.917 | 0.920x |
| `multiturn_prefix_reuse` | scenario peak footprint (B) | 35,272,446,880 | 47,721,861,928 | 1.353x |
| `stability_soak` | TTFT (s) | unmeasured | unmeasured | n/a |
| `stability_soak` | prefill tok/s | unmeasured | unmeasured | n/a |
| `stability_soak` | decode tok/s | unmeasured | unmeasured | n/a |
| `stability_soak` | wall clock (s) | 147.192 | 131.172 | 0.891x |
| `stability_soak` | scenario peak footprint (B) | 30,436,316,064 | 47,943,324,456 | 1.575x |
| `context_75k` | TTFT (s) | 971.413 | 1,504.453 | 1.549x |
| `context_75k` | prefill tok/s | 75.165 | 48.533 | 0.646x |
| `context_75k` | decode tok/s | 7.933 | 1.117 | 0.141x |
| `context_75k` | wall clock (s) | 973.305 | 1,517.883 | 1.560x |
| `context_75k` | scenario peak footprint (B) | 48,534,787,000 | 52,995,462,000 | 1.092x |

## Whole pass

| Quantity | Python | Swift | Swift/Python |
| --- | ---: | ---: | ---: |
| whole-process peak footprint (B) | 48,534,787,000 | 52,995,462,000 | 1.092x |
| /v1/models lists the model (s) | 2.913 | 5.225 | 1.794x |
| first served completion (s) | 10.971 | 7.018 | 0.640x |
| footprint after warm-up (B) | 29,111,882,656 | 28,980,335,376 | 0.995x |
| whole pass wall clock (s) | 1,570.0 | 2,066.9 | 1.316x |
| host 1-min load average, max | 28.94 | 35.21 | reported, never scored |
| footprint samples ok/failed | 6145/0 | 8079/0 | — |

### Python soak

```json
{
  "aggregate_output_tokens_per_second": 8.696106539410147,
  "first_footprint_bytes": 29798536096,
  "first_latency_seconds": 7.3097243309021,
  "footprint_drift_bytes": -495403008,
  "iterations": 20,
  "last_footprint_bytes": 29303133088,
  "last_latency_seconds": 7.176095008850098,
  "median_latency_seconds": 7.27947211265564
}
```

### Swift soak

```json
{
  "aggregate_output_tokens_per_second": 9.575191167401387,
  "first_footprint_bytes": 47866729256,
  "first_latency_seconds": 7.094546794891357,
  "footprint_drift_bytes": 75726848,
  "iterations": 20,
  "last_footprint_bytes": 47942456104,
  "last_latency_seconds": 6.636031150817871,
  "median_latency_seconds": 6.622905254364014
}
```

## Decision

```json
{
  "accepted": false,
  "blockers": [
    "long_prompt_8k/peak_physical_footprint_bytes ratio 1.1512007084931994 is outside the admissible band <= 1.1 (baseline 33705153440.0, candidate 38801396520.0)"
  ],
  "declaredAsymmetries": [
    "prompt cache enabled as deployed: --prompt-cache-size 1 --prompt-cache-bytes 8GB, the incumbent's installed configuration, kept rather than tuned away",
    "/health is a static 200 in the deployed revision 9150698; the generation-thread liveness fix is in the fork checkout but not installed",
    "/v1/models answers 200 about a second after launch with no weights resident; the model loads on first completion",
    "no prompt cache across requests: every request builds a fresh KV cache, so a shared multi-turn prefix is re-prefilled every turn",
    "one generation at a time: GenerationEngine is an actor, so requests serialize even though the profile allows concurrent leases",
    "readiness is gated on a resident model: /v1/models answers 503 until the weights are loaded, and advertises only the configured model",
    "served by MLXLLM.LLMModelFactory (text-only); the vision tower is not loaded and the HTTP contract refuses image and audio content"
  ],
  "deltas": [
    {
      "admissibleRatio": "<= 1.1",
      "baseline": 41,
      "candidate": 41,
      "metric": "prompt_tokens",
      "ratio": 1,
      "scenario": "short_prompt",
      "verdict": "within"
    },
    {
      "admissibleRatio": "<= 1.1",
      "baseline": 0.9848637580871582,
      "candidate": 0.9306838512420654,
      "metric": "time_to_first_token_seconds",
      "ratio": 0.9449874092734175,
      "scenario": "short_prompt",
      "verdict": "within"
    },
    {
      "admissibleRatio": ">= 0.9",
      "baseline": 41.63012362200416,
      "candidate": 44.053627819245506,
      "metric": "prefill_tokens_per_second",
      "ratio": 1.058215157352076,
      "scenario": "short_prompt",
      "verdict": "within"
    },
    {
      "admissibleRatio": ">= 0.9",
      "baseline": 10.271074675413033,
      "candidate": 10.896686081347813,
      "metric": "decode_tokens_per_second",
      "ratio": 1.0609100240924518,
      "scenario": "short_prompt",
      "verdict": "within"
    },
    {
      "admissibleRatio": "<= 1.1",
      "baseline": 29399274400,
      "candidate": 29116781352,
      "metric": "peak_physical_footprint_bytes",
      "ratio": 0.99039115577628,
      "scenario": "short_prompt",
      "verdict": "within"
    },
    {
      "admissibleRatio": "<= 1.1",
      "baseline": 7784,
      "candidate": 7784,
      "metric": "prompt_tokens",
      "ratio": 1,
      "scenario": "long_prompt_8k",
      "verdict": "within"
    },
    {
      "admissibleRatio": "<= 1.1",
      "baseline": 92.94988203048706,
      "candidate": 92.9784939289093,
      "metric": "time_to_first_token_seconds",
      "ratio": 1.0003078207072158,
      "scenario": "long_prompt_8k",
      "verdict": "within"
    },
    {
      "admissibleRatio": ">= 0.9",
      "baseline": 83.74405464492025,
      "candidate": 83.71828442340217,
      "metric": "prefill_tokens_per_second",
      "ratio": 0.999692274017214,
      "scenario": "long_prompt_8k",
      "verdict": "within"
    },
    {
      "admissibleRatio": ">= 0.9",
      "baseline": 9.723226817584736,
      "candidate": 10.533407463530223,
      "metric": "decode_tokens_per_second",
      "ratio": 1.0833242565606154,
      "scenario": "long_prompt_8k",
      "verdict": "within"
    },
    {
      "admissibleRatio": "<= 1.1",
      "baseline": 33705153440,
      "candidate": 38801396520,
      "metric": "peak_physical_footprint_bytes",
      "ratio": 1.1512007084931994,
      "scenario": "long_prompt_8k",
      "verdict": "outside"
    },
    {
      "admissibleRatio": "<= 1.1",
      "baseline": 48534787000,
      "candidate": 52995462000,
      "metric": "peak_physical_footprint_bytes",
      "ratio": 1.0919067595784442,
      "scenario": "process",
      "verdict": "within"
    }
  ]
}
```

## Reproducing this

### Pinned revisions

| Component | Revision |
| --- | --- |
| Python `mlx-lm` | `0.32.0`, direct URL `file:///Users/alexis/src/relux-works/mlx-lm/.git` @ `9150698` |
| `mlx` / `mlx-metal` | `0.32.2` / `0.32.2` |
| `transformers` | `5.16.1` |
| CPython | `3.14.7` |
| `mlx-swift` | `0.31.6` (`0bb916c67f4b9e5c682cbe02a42c701c93ab5021`) |
| `mlx-swift-lm` | `3.31.4` (`bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57`) |
| `swift-transformers` | `1.3.3` (pinned in `Package.swift`) |
| `model-harness` | `v1.6.1-44-gd91d6fc` (`d91d6fc`, 2026-08-27T06:58:43Z) |
| host | `MacBookPro18,2` / 68,719,476,736 B / build `25F80` / `arm64` |

### Digests

| Artifact | SHA-256 |
| --- | --- |
| gate = driver = candidate runtime (Release product) | `8a517b10e6a74793dd47d33d07b1b08275863f3fb7e8cfb11880a14b71014f91` |
| baseline executable, as the kernel ran it (`Python.framework` 3.14.7) | `e89d60f8ccee9db684801330198797723db2ac378cd53e5cfe523128feb76599` |
| launcher config `model-harness.benchmark.toml` | `d063af2ecc77eb9fd440d11ff14cc692b23f48cdf86a0117a08c128b82a36620` |
| prompt suite `examples/benchmark-prompts.json` | `bba8867f4300f343b15960fb2cf8f821c653b96f538cf87287d0dacd36275d41` |
| thresholds `examples/benchmark-thresholds.json` | `7490c45477e05e08d1ef6fc747fa80e2f2f224ca4dfae599c4957a4f4c6ac208` |
| model directory pin (`config.json` + safetensors index) | `1b10f3fe1c1097c909fa35e112b943255c44be4a5f332f45e0af57a96188460b` |
| sealed transcript, baseline pass | `85d1a985a69ec46033fa21f5dff42ab9deaa7bdafa1132fddbe9075277162008` |
| sealed transcript, candidate pass | `2e1b665d80cf83c09d039ffbbbdccd4970d5c658cb67352f32b43dd08cd52bd3` |

Note the third row of the attestation pair: `observedExecutableDigest` for the
candidate equals `gateBinaryDigest`. One file served, observed and judged.

### Build

```bash
cd tools/mlx-swift-runtime-prototype
xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release \
  -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData \
  -skipPackagePluginValidation -skipMacroValidation
```

A `swift build` product cannot compile mlx-swift's Metal shaders and refuses to
serve by design, so the profile must name the `xcodebuild` Release product.

### The measurement — one command

Verbatim, as `TASK-260827-2v13w8_run-benchmark-rev4.sh`:

```bash
./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype benchmark-run \
    --config  <rev4>/model-harness.benchmark.toml \
    --model   /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
    --prompts examples/benchmark-prompts.json \
    --thresholds examples/benchmark-thresholds.json \
    --session <rev4>/session \
    --harness /Users/alexis/.local/bin/model-harness \
    --baseline-runtime python-mlx-lm  --baseline-profile  qwen-benchmark-python \
    --candidate-runtime mlx-swift     --candidate-profile qwen-benchmark-swift \
    --port 18031 \
    --python-bin /Users/alexis/.local/pipx/venvs/mlx-lm-relux/bin/python \
    --candidate-binary ./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
    --baseline-declare  ... (3)  --candidate-declare ... (4)
```

Both passes ran sequentially inside that one process, 20 s apart, on the same
host, model, quantization, prompt suite, context policy and output bounds. The
context policy each pass derived from its own rendered launch argv was
identical: `kv=unbounded;prefill-step=2048;reasoning=medium`.

### The replay

```bash
mlx-swift-runtime-prototype benchmark-compare \
  --baseline  <rev4>/session/records/python-mlx-lm.json \
  --candidate <rev4>/session/records/mlx-swift.json \
  --thresholds examples/benchmark-thresholds.json \
  --attestations <rev4>/session/attest \
  --output <rev4>/replay-decision.json
# exit 3 — admitted, re-scored, identical verdict, and structurally unable to accept
```

## Gates

| Gate | Command | Exit |
| --- | --- | ---: |
| SwiftPM build | `swift build` | 0 |
| contract suite | `swift test` — 285 tests, 24 suites | 0 |
| Release build | `xcodebuild ... -configuration Release` | 0 |
| format | `xcrun swift-format lint --strict --recursive Sources Tests` | 0 |
| shell | `shellcheck -S warning scripts/*.sh` | 0 |
| production-entry gate smoke | `scripts/benchmark-gate-smoke.sh` — 39 checks, 0 failures | 0 |
| lifecycle smoke | `scripts/lifecycle-smoke.sh` — 17 checks, 0 failures | 0 |
| whitespace | `git diff --check` | 0 |
| the decision itself | `benchmark-compare` replay of this session | **3** (expected-red: exit 3 *is* the reproduced rejection) |

**One honest gap.** The production `benchmark-run` invocation was launched
through a `nohup ... &` wrapper that did not capture its exit status, so the
exit code of the run that produced these measurements was **not recorded**. What
is recorded is its `decision.json` (`accepted=false`), the one line that maps it
(`return decision.accepted ? .accepted : .rejected`), the replay at exit 3, and
the gate smoke driving the same production entry to exits 0, 2, 3 and 4 on real
passes. The measurement was not repeated to recover a number that the harness
should have captured the first time.

## Mutants

Nine, at the sites the decision depends on. Full record in
`TASK-260827-2v13w8_mutant-campaign-rev4.json`; each was applied to the source,
built, run against the acceptance suite, and reverted.

| ID | Class | Site | Caught by | Outcome |
| --- | --- | --- | --- | --- |
| **N-P** | narrowing | `BenchmarkRunCommand.drive`, the scenario loop → placeholder path | gate smoke | caught |
| **P-1** | production bypass | `execute`, admit failure → `ExitCode.accepted` | gate smoke | caught |
| **P-2** | production bypass | `execute`, `requiredScenarios: []` | gate smoke | caught |
| **P-3** | production bypass | `BenchmarkCompareCommand.attestation`, malformed → absent | gate smoke | caught |
| **N-1** | narrowing | `admitAttestation`, observing-binary clause made tautological | `swift test` | caught |
| **N-2** | narrowing | `admitTranscriptObservation`, absent seal self-minted | `swift test` | caught |
| **L-1** | limit probe | fabricated results, no transcript | gate smoke | caught |
| **L-2** | limit probe | fabricated results + transcript, 0.01 s/scenario | gate smoke | caught |
| **L-3** | limit probe | same, 0.0005 s/scenario | — | **survived** |

N-P is the mutant this revision was rewritten for: it swaps the real scenario
driver for the placeholder path while leaving the launch, the observation and
the seal genuine, and the production control goes from exit 0 to exit 4. That is
what makes the control a control — the acceptance depends on the scenarios
having been driven, not on the run having been watched.

P-1, P-2 and P-3 all **compile**, leave `RuntimeBenchmark.admit` completely
correct, and are invisible to the contract suite. Only the production-entry
smoke catches them.

L-3 survived and is reported as a survivor; see *Where the boundary actually is*.

## Artifacts

| File | What it is |
| --- | --- |
| `TASK-260827-2v13w8_results.md` | this report |
| `TASK-260827-2v13w8_record-python-mlx-lm-rev4.json` | baseline record, with every scenario's sealed transcript |
| `TASK-260827-2v13w8_record-mlx-swift-rev4.json` | candidate record, likewise |
| `TASK-260827-2v13w8_attestation-python-rev4.json` | what the gate observed of the baseline process |
| `TASK-260827-2v13w8_attestation-mlx-swift-rev4.json` | what the gate observed of the candidate process |
| `TASK-260827-2v13w8_decision-rev4.json` | the verdict the measuring binary returned |
| `TASK-260827-2v13w8_session-rev4.json` | readiness, soak detail, host load maxima, sampler coverage |
| `TASK-260827-2v13w8_run-benchmark-rev4.sh` | the one command, verbatim |
| `TASK-260827-2v13w8_model-harness.benchmark-rev4.toml` | the executed launcher config |
| `TASK-260827-2v13w8_mutant-campaign-rev4.json` | the nine mutants and their outcomes |
| `TASK-260827-2v13w8_gate-benchmark-smoke-rev4.log` | 39 production-entry checks |
| `TASK-260827-2v13w8_benchmark-run-rev4.log` | the measuring run's own output |
