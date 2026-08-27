# TASK-260827-2h39ya — dead-generation-worker health regression on MLX Swift

Carries the `mlx_lm.server` dead-generation-thread regression into the MLX Swift
runtime's acceptance suite.

## The regression being carried

`BUG-260827-1jhv2g`: mlx-lm's generation thread died on
`[metal::malloc] Resource limit (499000) exceeded` while the HTTP listener
stayed up. `/health` kept answering `200 {"status": "ok"}` for a runtime that
could not produce a token, so callers found out through request timeouts.
Upstream fix (`BUG-260827-2tul5n`): `/health` reports
`503 {"status": "unavailable"}` when `_generation_thread.is_alive()` is false.

## How it lands in Swift

It does not port literally. Swift has no equivalent silent death — an error
thrown inside a structured task propagates to its caller rather than tearing a
worker down. The same regression lands as **invalidation**: the runtime
condemns itself when a generation fails with a signature that names the
backend rather than the request.

The Swift prototype had the same observable bug for a different reason.
`Router.complete` caught a generation error, answered `500`, and left readiness
at `.ready` — so `/health` kept answering `200` for a runtime whose backend was
gone.

### Changes

| File | Change |
| --- | --- |
| `Sources/MLXSwiftRuntimeContract/GenerationWorkerHealth.swift` | New. Classifies a generation failure, owns the readiness transition, pins the supervision marker, and owns the `/health` body via `HealthReport`. |
| `Sources/MLXSwiftRuntimeContract/ModelsListing.swift` | `RuntimeReadiness` gains `.generationWorkerFailed`; a condemned runtime stops advertising on `/v1/models` under its own error code, not `model_load_failed`. |
| `Sources/MLXSwiftRuntimeContract/RuntimeOptions.swift` | `--fault-inject-generation-error <message>` acceptance seam; an empty value is refused at parse time. |
| `Sources/mlx-swift-runtime-prototype/RuntimeState.swift` | `recordGenerationFailure(_:)` — production call site for the transition. Drops the engine; returns whether this call condemned the worker, so the marker is emitted once. |
| `Sources/mlx-swift-runtime-prototype/Router.swift` | `health()` routes through `HealthReport`; `recordGenerationFailure(_:)` emits the marker; `complete()` reports its failures. |
| `Sources/mlx-swift-runtime-prototype/HTTPServer.swift` | The SSE path reports its failures too — a second, independent production call site. |
| `Sources/mlx-swift-runtime-prototype/GenerationEngine.swift` | Honours the fault seam before any MLX call; the injected text is thrown verbatim. |
| `examples/model-harness.prototype.toml` | Supervision policy on the `generation_worker_unavailable` marker, `restart_on_failure = false`. |
| `scripts/dead-generation-smoke.sh` | New end-to-end regression check. |

### Behaviour when the worker is condemned

1. Engine dropped; readiness `generationWorkerFailed`, terminal.
2. `GET /health` → `503 {"status":"unavailable","detail":"<failure>"}`.
3. `GET /v1/models` → `503`, empty `data[]`, code `generation_worker_unavailable`,
   model ID absent.
4. `POST /v1/chat/completions` → `503` at the readiness gate.
5. One stdout line carrying the literal marker `generation_worker_unavailable`,
   which a `model-harness` `fatal_output_substrings` policy turns into a restart.

The runtime never restarts itself. Recovery is the supervisor's job; a runtime
that quietly healed itself would hand the next caller a backend it already
knows is broken.

## What is deliberately NOT condemned

`GenerationWorkerHealth.invalidatingSignatures` is three entries —
`metal::malloc`, `Resource limit`, `Failed to load the default metallib` —
each from a recorded incident. Condemning on every generation error would be a
worse bug than the one being fixed: one malformed request would take a healthy
runtime out of rotation and, under supervision, restart it.

Only a `ready` runtime transitions. A generation killed by `SIGTERM` must not be
reported as a dead worker and must not ask the harness to restart a process that
was asked to stop.

## Evidence

All commands run from `tools/mlx-swift-runtime-prototype`.

| Command | Result |
| --- | --- |
| `swift test -c release` | 116 tests, 11 suites, **exit 0** |
| `xcrun swift-format lint --configuration .swift-format --recursive Sources Tests` | **exit 0**, no diagnostics |
| `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData -skipPackagePluginValidation -skipMacroValidation` | **BUILD SUCCEEDED, exit 0** |
| `scripts/dead-generation-smoke.sh` | 35 checks, 0 failures, **exit 0** |
| `scripts/lifecycle-smoke.sh` (regression check) | 17 checks, 0 failures, **exit 0** |
| `go build ./... && go vet ./...` in `tools/agents-infra` | **exit 0** |
| `go test ./internal/modelharness/` | **ok, exit 0** |
| `model-harness render qwen-mlx-swift-prototype --config <example> --json` | **exit 0**; supervision policy present in the rendered plan |

### The end-to-end check

```bash
BINARY=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
HARNESS=/Users/alexis/.local/bin/model-harness \
MODEL=/Users/alexis/.cache/huggingface/hub/models--mlx-community--Qwen1.5-0.5B-Chat-4bit/snapshots/659d8dafc39202a6688bb46242d60440702489b1 \
PORT=18019 OUT=./dead-generation-out \
scripts/dead-generation-smoke.sh
```

Real Release binary, real `model-harness`, real model. Four phases:

| Phase | Establishes |
| --- | --- |
| control | Same build, no injected fault: `/health` `200` before and after a real generation; no marker in its output. |
| fault (unsupervised) | `/health` `200` → completion `500` naming the failure → `/health` `503 unavailable` → still `503` on a later poll → `/v1/models` `503` with the model ID absent → later completions `503` → marker emitted. |
| fault, streaming | The same, driven through the SSE call site: terminal error frame, `/health` `503`. |
| supervision | The same fault with the policy attached: harness names the marker, restarts, replacement answers `200`. |
| negative | A request-scoped failure returns `500` and leaves `/health` `200`, the model advertised, no marker, no restart. |

Captured artifacts under `OUT`: `fault-health-before.json`,
`fault-health-after.json`, `fault-models.json`, `fault-marker.json`,
`supervised-restart.txt`, `benign-health.json`.

Recorded condemnation event:

```json
{"detail":"RuntimeError: [metal::malloc] Resource limit (499000) exceeded","event":"generation_worker_failed","marker":"generation_worker_unavailable"}
```

Recorded supervised recovery:

```
model-harness: restarting profile "prototype-supervised" after supervised runtime failure (1/2): fatal output "generation_worker_unavailable": signal: killed
```

### Would it have failed?

| Mutant | Result |
| --- | --- |
| `classify` returns `.workerInvalidated` for everything (over-broad gate) | contract suite **exit 1**, 12 issues — every request-scoped case reddens |
| `invalidatingSignatures` narrowed, dropping `metal::malloc` / `Resource limit` | contract suite **exit 1**, 4 issues — the incident message no longer condemns |
| `HealthReport` answers `200 {"status":"ok"}` for a condemned worker (the regression itself) | contract suite **exit 1**, 5 issues |
| **Production**: delete `await recordGenerationFailure(error)` from `Router.complete`, rebuild Release | smoke **exit 1**, 11 failures, including `REGRESSION GATE: /health answered 200 with the worker condemned` — the original incident reproduced exactly |

The fourth is the important one: it proves the check is bound to the production
call site and not to a helper that nothing calls. The negative phase proves the
gate is narrow — it is satisfied only by a runtime that distinguishes a broken
backend from a bad request, not by one that condemns itself on any error.

## Design notes worth carrying forward

- **Health-503 and supervised restart cannot be observed in one process.** With
  `fatal_output_substrings` attached, `model-harness` kills the runtime within
  milliseconds of the marker reaching its stdout — correct behaviour, and it
  destroys the 503 window. The smoke runs the same injected fault twice,
  unsupervised then supervised, so each phase measures one property rather than
  measuring which won the race.
- **`model-harness` does not forward signals to the runtime it spawns.**
  Signalling the harness alone orphans a live listener. An early revision of the
  smoke "passed" three phases against the leftover control process. Teardown now
  signals the process group, and every phase refuses to start on an occupied
  port so a stale listener can never satisfy a check.
- **The fault seam is a seam, not a verdict.** `--fault-inject-generation-error`
  throws the given text out of the real generation path; production classifies
  it. That is what makes the negative phase a real negative. Upstream mlx-lm
  proved its own generation-thread recovery the same way, by injecting a
  `RuntimeError` into its live generation loop.
- **Small model on purpose.** The failure path never reaches the weights, so
  paying 29 GB and a six-second load would only mean the check cannot run while
  the default Python runtime is resident. The 261 MB
  `mlx-community/Qwen1.5-0.5B-Chat-4bit` makes it a two-minute check that costs
  no meaningful GPU memory.

## Scope kept

Python `mlx-lm` remains the default local runtime and the rollback path.
`examples/model-harness.prototype.toml` gains the supervision policy but is
still an example, not an installed config. No installed config was changed.
