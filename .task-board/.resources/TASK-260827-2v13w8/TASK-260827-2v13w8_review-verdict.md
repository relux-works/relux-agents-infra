# TASK-260827-2v13w8 review verdict — changes requested

- Change Request: `CR-TASK-260827-2v13w8-1`, revision 1
- Reviewer run: `RUN-260828-805ed2`
- Candidate: base `3f313d9175f2ada9b9ab3320ab524c0918f9daac`, tree `9ff759db0fd48ed5f824e3d9af44421133db36d7`
- Patch SHA-256 independently verified: `55651146a3c3e6251094053478003cda2de941ef8adf4fcef8fbdabe868b3bd1`
- Verdict branch: **changes requested → `to-dev`**

The installed Python runtime remains the default while this rework is open. The current REJECT report is not accepted as the migration decision because it did not benchmark a supported text-only Swift factory path and its comparison gate admits self-minted evidence.

## Findings

### F1 — Critical — B1 is not established as an upstream-only blocker

The byte arithmetic is correct: `73016² × 24 × 2 = 255,904,140,288`, above the `41,747,087,360` Metal buffer limit. The pinned source checkout is exactly `mlx-swift-lm bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57`; `MLXVLM/Models/Qwen35.swift:982-1043` discards `windowSize` and evaluates the full prompt. The crash log confirms the selected factory was `MLXVLM.VLMModelFactory` and the process trapped.

The report's stronger claim — that nothing in this repository can avoid the unchunked path — is false or at minimum untested:

- `ModelLoader.swift:104-118` chooses `VLMModelFactory` first and reaches `LLMModelFactory` only after VLM refusal.
- The same pinned upstream explicitly registers `qwen3_5` in `MLXLLM/LLMModelFactory.swift:45`.
- `MLXLLM/LLMModel.swift:15-44` implements chunked prefill using `windowSize ?? 512`.
- `MLXLLM/Models/Qwen35.swift:638-677` builds the text model and drops vision weights during sanitize.
- The delivered HTTP contract is text-only: `ChatCompletionRequest.swift:151-169` refuses image/audio content.

This is a supported factory path for the configured text-only surface. Expose/select it, prove it loads this exact model directory, and rerun the full comparison including 75k. B1 and likely the 8k prefill result may change. Until that run exists, “requires upstream Qwen35VL work; no repository workaround” is not a valid decision premise.

Negative shape: **bypass path around the check** — a supported production-adjacent path that avoids the claimed constraint was never exercised.

### F2 — Critical — the comparison gate accepts forged/self-minted run evidence

Production reproduction:

```text
./.build/release/mlx-swift-runtime-prototype benchmark-compare \
  --baseline forged-baseline.json --candidate forged-candidate.json \
  --thresholds examples/benchmark-thresholds.json --output forged-decision.json
exit 0; accepted=true; blockers=[]
```

Both records had empty `revisions`, different `--config` paths, and caller-authored identical `contextPolicy` strings. The production subcommand accepted them. Evidence files are attached beside this verdict.

This works because `runtime-benchmark.py:924-934` self-mints `contextPolicy = "unbounded-kv"` from a constant rather than deriving it from the rendered launch; `RunRecord.command` records only `model-harness run`, not the driver invocation or config bytes/digest; `RuntimeBenchmark.admit` compares the `Pins` values supplied by the caller and never binds revisions or profile configuration to an executed run.

Negative shape: **forged or self-minted evidence**. The existing P1 mutant proves a laundering call-site edit but does not cover an unmodified caller supplying forged records through the real CLI.

Bind each record to the actual rendered profile/config, complete driver argv, runtime revision evidence and scenario suite, or narrow the command's claim so it is not presented as an attestation gate. Add a production-entry negative test that submits caller-minted, correctly shaped records and requires refusal.

### F3 — High — the favourable memory verdict is not like-for-like

The reported `1.094x within` ratio divides whole-pass peaks even though Python completed the 75k workload and Swift aborted before measuring it. The two maxima therefore come from different completed work:

| Memory comparison | Python | Swift | Ratio |
| --- | ---: | ---: | ---: |
| Whole pass, different completed workload | 45.11 GiB | 49.36 GiB | 1.094x |
| Same 8k scenario | 31.47 GiB | 44.01 GiB | **1.399x** |
| Multi-turn peak-so-far | 32.00 GiB | 49.36 GiB | **1.542x** |

`FootprintSampler.mark()` returns the cumulative peak since process start, not a scenario-local peak. `RuntimeBenchmark.decide` scores only the whole-process values even when a required parity scenario failed. It is not valid to call the memory axis “inside the band” for a workload the candidate did not finish. Report memory as unknown/non-comparable for the failed pair, or compare scenario-local peaks on successful common scenarios and enforce the intended threshold there.

The summary also labels decimal values `47.3/33.8` as GiB; the actual binary GiB values are `44.01/31.47`.

### F4 — High — favourable latency/load/stability claims are overstated or mislabeled

- Short TTFT is a real observed API latency ratio (`0.751x`), but the requests rendered to 41 Swift prompt tokens versus 79 Python tokens. One observation on materially different token workloads does not establish a runtime “win”; report the observation and confounder.
- Swift `load_seconds=3.289` has no comparable Python load-only measurement. Installed `mlx_lm.server` starts `ResponseGenerator._generate`, which immediately calls `ModelProvider.load_default()` before serving work. It does not lazily begin loading on the first completion. `/v1/models` does answer during the background load, so the readiness defect is real; the causal description and “Swift wins load time” are not.
- `run_soak` labels `completion_tokens / total elapsed` as `decodeTokensPerSecond`; that elapsed includes every request's TTFT/prefill. The `8.99/9.78 tok/s` values are aggregate output throughput, not decode throughput.
- Both two-minute/20-request soaks had zero failures and negative footprint drift. That supports a bounded pass for both, not “Swift wins soak stability.” A larger negative release by Python is not instability.

### F5 — High — real MLX failures bypass the two accepted supervision contracts

Confirmed from `mlx-swift 0bb916c67f4b9e5c682cbe02a42c701c93ab5021`: the default handler calls `fatalError` at `MLX/ErrorHandler.swift:336-346`; the runtime never calls `withErrorHandler` or `withError`. The 75k crash produced no `generation_worker_failed`, health 503, supervision marker or teardown event.

`TASK-260827-2h39ya` and `TASK-260827-2q77g8` remain evidence for injected Swift `throw` paths, but their contracts are unreachable for actual MLX C++ fatal errors under the delivered runtime. The story must say this explicitly. Rework must either install and production-test MLX error handling around generation or scope those contracts honestly to thrown failures; fault injection alone is not proof of allocator supervision.

### F6 — Medium — reproducibility and incumbent facts need correction

The report's “every command” table contains `<task config>`, `<model>`, `--declare ...`, `BINARY=...`, `OUT=...`, and a task-local `capacity-probe.sh` command whose script is not attached. The executed task config is not byte-identical to the committed example (only the Swift executable path differs, but its SHA-256 is different). Record the complete commands and attach/digest every executed config/script.

Incumbent findings independently confirmed:

- Installed `mlx-lm 0.32.0 @ 9150698` has unconditional `/health` 200; checkout `b0a45b8` is not installed and `9150698` is not its ancestor.
- Python `/v1/models` scans cached models and appends the configured model last; `model-harness/internal/modelharness/stress.go:217-218` selects `Data[0].ID`, so stress can benchmark the wrong model.
- `/v1/models` readiness is not weight readiness, but the load starts immediately in the generation thread rather than on the first completion.

## Independent validation

- Candidate tree matches the current tracked worktree; exact CR patch digest verified.
- Swift pins verified from both `Package.resolved` and local checkout HEADs.
- `swift test -c release`: 228 tests in 21 suites, exit 0.
- `benchmark-compare` on attached producer records: exit 3, four blockers reproduced.
- `python3 -m py_compile scripts/runtime-benchmark.py`: exit 0.
- `swift-format lint --strict`: exit 0.
- Exact candidate `git diff --check`: exit 0.

Green positive gates do not resolve F1-F6. Revision 2 needs the supported-factory benchmark, provenance-bound negative gate, corrected memory/metric semantics, exact reproducibility evidence, and an explicit story-level disposition of real MLX traps. New findings must also be added to `LOGBOOK.md` by the producer; the reviewer did not modify the snapshotted worktree because that would stale the Change Request.
