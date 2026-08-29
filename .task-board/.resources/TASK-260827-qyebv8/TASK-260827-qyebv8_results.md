# TASK-260827-qyebv8 — MLX Swift LM Qwen runtime prototype

**Status (revision 3, run `RUN-260827-a388ff`):** every checklist item is
proven. Prototype built, contract suite green, lifecycle proven end to end, and
the full-model load, streaming, tool-call and refusal smokes run against the
real 28 GB Qwen through `model-harness run`.

This document is the current summary. Two later artifacts carry the detail it
points at and are not superseded:

- `TASK-260827-qyebv8_full-model-smoke.md` — the full-model evidence and the
  `swift build` / Metal shader library finding (revision 2).
- `TASK-260827-qyebv8_review-rework-rev2.md` — the four reviewer findings from
  revision 1 and how each was fixed and attacked (revision 3).

Everything below that previously described the full-model smokes as blocked has
been rewritten; the historical blocker record is kept separately in
`TASK-260827-qyebv8_blocker.md`, which describes a host condition that no longer
holds.

Python `mlx-lm` remains the default local runtime. No installed config was
changed.

---

## 1. What was built

`tools/mlx-swift-runtime-prototype/` — a SwiftPM package producing
`mlx-swift-runtime-prototype`, which serves the configured local Qwen model over
the same OpenAI-compatible surface the Pi profile
`qwen-3.8-27b-mlx-8bit` uses against Python `mlx_lm.server`.

Two targets:

| Target | Role |
| --- | --- |
| `MLXSwiftRuntimeContract` | Pure Swift. Argument parsing, model-directory admission, OpenAI request decoding, every admission gate, the `<think>` reasoning splitter, response/SSE builders, runtime events. No MLX, no network, no model — so all of it is testable without a 29 GB load. |
| `mlx-swift-runtime-prototype` | Executable. MLX Swift LM model loading, generation, the NIO loopback HTTP server, mach memory sampling, signal lifecycle, and the `preflight` subcommand. |

### Pinned upstream revisions (verbatim)

Declared with `exact:` in `Package.swift`; the full transitive graph is pinned by
revision in the committed `Package.resolved`.

| Package | Version | Revision |
| --- | --- | --- |
| `https://github.com/ml-explore/mlx-swift` | 0.31.6 | `0bb916c67f4b9e5c682cbe02a42c701c93ab5021` |
| `https://github.com/ml-explore/mlx-swift-lm` | 3.31.4 | `bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57` |
| `https://github.com/huggingface/swift-transformers` | 1.3.3 | `2fa33e1f5e7131a7fc64c28e6d161dcec0d24820` |
| `https://github.com/apple/swift-nio` | 2.99.0 | `f71c8d2a5e74a2c6d11a0fbe324774b5d6084237` |
| `https://github.com/swiftlang/swift-syntax` (transitive) | 603.0.2 | `79e4b74a295b6eb74a8b585e3a39d29e70c1dbd1` |
| `https://github.com/huggingface/swift-huggingface` (transitive) | 0.9.0 | `b721959445b617d0bf03910b2b4aced345fd93bf` |
| `https://github.com/huggingface/swift-jinja` (transitive) | 2.4.2 | `7d0b8880ef8e567dd4e0089f8b99fb354129017c` |

The two MLX revisions are also compiled into the binary and reported on the
`listening` event and as `system_fingerprint` on every completion, so a captured
transcript always names the code that produced it.

Toolchain: Apple Swift 6.3.3 (swiftlang-6.3.3.1.3), Xcode 26.6 (17F113),
target `arm64-apple-macosx26.0`, on a MacBookPro18,2 with 64 GiB.

---

## 2. Commands and outputs

Every command below was run as a standalone process; the exit code shown is its
real status.

Gate results as of revision 3. `swift build -c release` produces a binary that
**cannot load a model** — see `TASK-260827-qyebv8_full-model-smoke.md` — so it is
a compile gate only; the runnable product comes from `xcodebuild`.

| Command | Exit | Note |
| --- | ---: | --- |
| `swift package resolve` | 0 | Produced the `Package.resolved` above |
| `swift build -c release` | 0 | Compile gate; the product cannot reach the GPU |
| `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData -skipPackagePluginValidation -skipMacroValidation` | 0 | `** BUILD SUCCEEDED **`, 0 error lines; the runnable product |
| `swift test -c release` | 0 | 103 tests in 9 suites |
| `swift format lint --recursive --strict Sources Tests Package.swift` | 0 | `.swift-format` matches upstream mlx-swift-lm (4-space) |
| `bash -n scripts/smoke.sh scripts/lifecycle-smoke.sh scripts/metallib-gate-probe.sh` | 0 | |
| `go build ./...` | 0 | |
| `go vet ./internal/infra/...` | 0 | |
| `gofmt -l internal/infra/infra.go internal/infra/source_sync_build_artifacts_test.go` | 0 | no output |
| `go test ./internal/infra/... -count=1` | 0 | 124.040s, after the install-sync fixes in §2.1 |
| `model-harness render qwen-mlx-swift-prototype --config <task config> --host 127.0.0.1 --port 18017 --json` | 0 | plan below |
| `scripts/smoke.sh` (real 28 GB model, through `model-harness run`) | 0 | **SMOKE OK, 36 PASS / 0 FAIL** |
| `scripts/lifecycle-smoke.sh` | 0 | 17 checks, 0 failures |
| `scripts/metallib-gate-probe.sh` | 0 | 10 checks, 0 failures; forged-evidence probe at the real entry point |
| `model-harness run qwen-mlx-swift-fixture ...` | 143 (SIGTERM, expected) | managed-path lifecycle, §2.3 |
| `mlx-swift-runtime-prototype preflight --model <model> --reasoning-effort medium` | 0 | report below |

Logs: `.temp/TASK-260827-qyebv8/logs/` and
`tools/mlx-swift-runtime-prototype/.temp/qyebv8/`.

### 2.1 An install-sync regression this package caused, and its fix

`go test ./... -count=1` failed on
`TestInstalledBinarySetupLocalPiInfraRepairsModeAndSymlinkDrift`:

```
setup local did not repair alias mode: exit status 1
open .../003/.agents/tools/mlx-swift-runtime-prototype/.build/arm64-apple-macosx/release/
     swift-crypto_Crypto.bundle/PrivacyInfo.xcprivacy: permission denied
```

**Cause.** `shouldSkip` in `internal/infra/infra.go` selects installable source
files from an explicit deny-list of path components; it does not consult
`.gitignore`. Once anyone compiles the new SwiftPM package, `tools/
mlx-swift-runtime-prototype/.build/` exists in the working tree, and `setup
local` tried to copy it into the install root. Some SwiftPM bundle resources are
written read-only, so the copy fails and takes the whole run down.

**Fix, at the source rather than around it.** `.build` joins `.temp` as a skipped
path component:

```go
// SwiftPM writes build products into `.build`. They are machine-local
// artifacts, often hundreds of megabytes, and their bundle resources carry
// restrictive permissions that make a copy fail outright.
case hasPathComponent(rel, ".build"):
    return true
```

A package-local `.gitignore` alone would not have helped, because this sync
never reads one. New coverage in
`internal/infra/source_sync_build_artifacts_test.go`, with two mutants:

| # | Mutation | Exit | Verdict |
| --- | --- | ---: | --- |
| M9 | `.build` exclusion removed entirely | 1 | CAUGHT |
| M10 | exclusion widened to `strings.Contains(rel, "build")` | 1 | CAUGHT |

M10 matters: a substring match would also swallow `tools/build/main.go` and
`tools/.buildkite/`, so the test pins the exact-component semantics, not just
the presence of some exclusion.

**Flaky test observed, unrelated.**
`TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC`
failed on two full-suite runs and passed on two of three isolated re-runs with
identical code. It drives the real pinned Pi binary under a 10 s RPC timeout, on
a host carrying 40+ GiB of wired memory from the resident Qwen runtime. It
passed in the final full-suite run. This looks environmental, not a regression
from this task, but it is recorded rather than dismissed.

### model-harness accepts the prototype as a managed local runtime

```json
{
  "contract": "model-harness.launch-plan",
  "schema_version": 1,
  "profile": "qwen-mlx-swift-prototype",
  "mode": "local",
  "executable": ".../tools/mlx-swift-runtime-prototype/.build/release/mlx-swift-runtime-prototype",
  "argv": ["serve", "--model", "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit",
           "--host", "127.0.0.1", "--port", "18017",
           "--reasoning-effort", "medium", "--default-max-tokens", "2048"],
  "endpoint": "http://127.0.0.1:18017/v1"
}
```

`{host}` / `{port}` substitution works, and the resolved endpoint matches the one
the Pi profile expects. The profile lives in a **task-scoped** config file; the
operator's `/Users/alexis/src/.agents/.configs/model-harness.toml` is untouched.

### Preflight against the exact configured model

`mlx-swift-runtime-prototype preflight` exercises everything that can be checked
without materializing weights. Run against
`/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit`, exit 0:

| Stage | Outcome | Detail |
| --- | --- | --- |
| `base_configuration` | passed | `model_type=qwen3_5 quantization=8bit/group64` |
| `architecture_registry` | passed | `qwen3_5` implemented by **MLXVLM and MLXLLM** |
| `vlm_configuration_decode` | passed | the model's own `config.json` decodes into `MLXVLM.Qwen35Configuration` |
| `tokenizer_load` | passed | swift-transformers `AutoTokenizer` from the model directory |
| `chat_template` | passed | 260 tokens rendered with 1 tool declaration |
| `generation_starts_in_reasoning` | passed | prompt ends `...<|im_start|>assistant\n<think>\n` |
| `tool_call_format` | passed | inferred `xml_function` |

This is the decisive compatibility result: **mlx-swift-lm 3.31.4 implements this
model's architecture**, decodes this model's exact configuration, loads its
tokenizer, and renders its chat template including tool declarations. The
`<think>` finding also empirically confirms the assumption the reasoning splitter
is built on — the template opens a think block, so generation begins *inside*
reasoning and the model emits only the closing `</think>`.

### 2.3 The managed `model-harness run` path, end to end

Run against the zero-memory fixture profile, so it exercises the real managed
path without weights:

```
model-harness pid=60663
readiness poll: bound=1 status=503 body={"data":[],"error":{"code":"model_load_failed",...},"object":"list"}
--- child under the harness ---
60668 mlx-swift-runti
--- SIGTERM to the process group ---
model-harness exited after 0s status=143
port 18019 released
```

Forwarded child stdout, unchanged by the harness:

```json
{"event":"listening","host":"127.0.0.1","port":18019,"mlx_swift":"0.31.6 (0bb916c...)","mlx_swift_lm":"3.31.4 (bd4b743...)","model_id":"..."}
{"event":"model_load_failed","model_path":"...","detail":"no MLX Swift LM factory could load model_type \"not_a_real_architecture\": ..."}
{"event":"shutting_down","signal":"SIGTERM"}
{"event":"stopped"}
```

What this establishes for the managed contract:

- `model-harness run` spawns the prototype as its **direct child** (60668 under
  60663), which is the ownership invariant the Pi launcher and the shared broker
  both depend on.
- The child's stdout and stderr are forwarded unchanged, so the one-line JSON
  events reach the launcher's log.
- `/v1/models` answers through the managed endpoint while not ready, in the
  shape the readiness poller parses.
- SIGTERM to the process group reaches the child, which shuts down cleanly and
  releases the port. Harness status 143 is the harness itself taking the same
  group signal, matching what `terminateProcessGroup` does in production.

Not established: the **ready-state** `200` listing after a real weight load.
That is the one part of the readiness contract still gated on host memory
(§5), and it is covered by unit tests plus mutant M3, not by a live load.

### Lifecycle smoke — 17/17, zero GPU memory

`scripts/lifecycle-smoke.sh` drives startup, readiness, load failure and
shutdown against a fixture directory that passes directory admission but that no
factory can load, so it costs no GPU memory and can run beside the resident
Python runtime.

```
PASS  non-loopback host exits 2
PASS  non-loopback refusal names the required host
PASS  missing model directory exits 2
PASS  missing directory is reported as missing
PASS  unsupported reasoning effort exits 2
PASS  listener bound before the model resolved
PASS  /v1/models answers 503 while not ready
PASS  not-ready listing is an empty OpenAI model list
PASS  the model ID is absent from the not-ready listing
PASS  chat completions answers 503 while not ready
PASS  not-ready completion reports model_not_ready
PASS  unloadable architecture produced a model_load_failed event
PASS  /v1/models stays 503 after a failed load
PASS  failed listing reports model_load_failed and advertises nothing
PASS  SIGTERM exited 0 in 0s
PASS  runtime emitted a stopped event
PASS  port 18018 released
LIFECYCLE SMOKE OK (0 failures)
```

Runtime event stream from that run:

```json
{"event":"listening","host":"127.0.0.1","port":18018,
 "mlx_swift":"0.31.6 (0bb916c67f4b9e5c682cbe02a42c701c93ab5021)",
 "mlx_swift_lm":"3.31.4 (bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57)","model_id":"..."}
{"event":"model_load_failed","model_path":"...","detail":"no MLX Swift LM factory could load model_type \"not_a_real_architecture\": MLXVLM.VLMModelFactory: unsupportedModelType(\"not_a_real_architecture\") | MLXLLM.LLMModelFactory: unsupportedModelType(\"not_a_real_architecture\")"}
{"event":"shutting_down","signal":"SIGTERM"}
{"event":"stopped"}
```

This satisfies the managed-contract lifecycle expectations directly: the listener
binds *before* the load resolves (so the launcher's poll sees `503`, not a
refused connection), `/v1/models` never advertises the model while it is not
resident, and SIGTERM exits 0 well inside the profile's
`shutdown_timeout_seconds = 10`.

---

## 3. Tests

`swift test -c release` — 103 tests, 9 suites, exit 0.

| Suite | Covers |
| --- | --- |
| `serve argument parsing` | loopback gate, absolute-path gate, port range, duplicate/unknown flags, subcommand set, reasoning-effort domain |
| `model directory admission` | fail-closed admission; `unreadable` is not downgraded to `missing` |
| `chat completion admission` | model identity, role set, bounds, tool declaration shape |
| `unsupported parameter refusal` | fields refused vs. inert forms admitted |
| `models listing readiness gate` | listing derived from readiness, not configuration |
| `reasoning splitter` | `</think>` splitting across arbitrary chunk boundaries |
| `OpenAI response shape` | non-streaming, streaming, usage packet, tool-call encoding, SSE framing |
| `runtime events` | load event fields; unknown memory is `null`, never `0` |

### Negative evidence: narrowing mutants, all caught

Revision 1 ran the ten mutants below. Revision 2 added six more (Metal shader
library gate and the Go install-sync exclusion) and revision 3 added six more
for the two reviewer findings — see
`TASK-260827-qyebv8_review-rework-rev2.md`.

Each mutant weakens one gate in production code — it does not delete it — and
the suite must fail. Every mutant was applied, run, reverted; `tests_ran > 0` was
required so an empty filter could not read as green.

| # | Mutation (production code) | Filter | Exit | Tests run | Verdict |
| --- | --- | --- | ---: | ---: | --- |
| M1 | model identity relaxed to `configuredID.hasPrefix(requested)` | `ChatCompletionAdmissionTests` | 1 | 17 | CAUGHT |
| M1b | model identity relaxed to `requested.contains(configuredID)` | `ChatCompletionAdmissionTests` | 1 | 17 | CAUGHT |
| M1c | model identity relaxed to case-insensitive compare | `ChatCompletionAdmissionTests` | 1 | 17 | CAUGHT |
| M2 | `developer` role silently mapped to `system` | `ChatCompletionAdmissionTests` | 1 | 17 | CAUGHT |
| M3 | `/v1/models` advertises the model while still loading (status stays 503) | `ModelsListingTests` | 1 | 5 | CAUGHT |
| M4 | unreadable model dir downgraded to `missing` | `ModelDirectoryCheckTests` | 1 | 8 | CAUGHT |
| M5 | loopback gate relaxed to `127.` prefix + `localhost` | `RuntimeOptionsTests` | 1 | 15 | CAUGHT |
| M6 | `chat_template_kwargs` dropped instead of refused | `UnsupportedParametersTests` | 1 | 6 | CAUGHT |
| M7 | reasoning splitter stops holding back partial markers | `ReasoningSplitterTests` | 1 | 10 | CAUGHT |
| M8 | reasoning splitter drops its unflushed tail | `ReasoningSplitterTests` | 1 | 10 | CAUGHT |

M1 was **not** caught on the first attempt. The original identity tests only
used unrelated or longer IDs, so a prefix-relaxed gate passed them. Test cases
for proper prefixes, superstrings and case-shifted paths were added, and M1,
M1b and M1c are caught by them. Logs: `logs/mutation-02.log`, `logs/mutation-03.log`.

Production call sites for the gates:
- `Router.chatCompletions(body:)` calls `ChatCompletionAdmission.admit(_:configuration:)`
  before any tokenizer or model work.
- `Router.route(method:path:body:)` calls `ModelsListing.make(modelID:readiness:created:)`
  for `GET /v1/models`, reading the stored readiness from `RuntimeState`.
- `Main.main()` calls `ModelDirectoryCheck.admit(path:observation:)` before the
  listener binds.
- `RuntimeOptions.parse(arguments:)` is called from `Main.main()` before anything else.

---

## 4. Named gap list

Recorded, not worked around. Nothing below is silently emulated.

### 4.1 Unsupported request parameters — refused with `400 unsupported_parameter`

| Field | Why refused |
| --- | --- |
| `reasoning_effort` | The Pi profile sets `supports_reasoning_effort = false`. Qwen3.5 takes reasoning effort as a **chat-template kwarg**, chosen at startup via `--reasoning-effort`. There is no per-request path, so accepting the field would report a change that never happened. |
| `chat_template_kwargs` | The `</think>` splitter is configured from the startup template state. A request that set `enable_thinking: false` would have its entire answer filed as `reasoning` and its `content` reported as `null`. |
| `stop` | No decoded stop-string matching is wired up. |
| `logit_bias`, `top_logprobs`, `logprobs: true` | No logit processor or logprob plumbing. |
| `response_format` | No grammar/JSON-schema constrained decoding. |
| `n != 1` | Only one choice is generated. |
| `tool_choice` other than `auto` | The runtime cannot force or forbid a tool call. |

`seed` **is** honoured (forwarded to `GenerateParameters.seed`).
`role: "developer"` is refused with `400 unsupported_role`
(`supports_developer_role = false`).
Non-text content parts are refused (`input = ["text"]`).

### 4.2 Reasoning-content behaviour

- The Qwen3.5 chat template ends its generation prompt with a bare `<think>\n`
  (confirmed by preflight), so generation **starts inside** the reasoning block
  and the model emits only the closing `</think>`.
- MLX Swift LM has **no reasoning support at all**: `MLXLMCommon.Generation` has
  exactly three cases — `chunk`, `info`, `toolCall` — and nothing in
  `MLXLMCommon` mentions thinking or reasoning. The `<think>` split is therefore
  implemented in this prototype (`ReasoningSplitter`), mirroring
  `mlx_lm.generate.TextStateMachine`, which is where Python does the same job.
  **This is a real feature gap between the Python and Swift runtimes.**
- The prototype publishes reasoning under the JSON key **`reasoning`**, matching
  `mlx_lm.server` (`server.py`, `generate_response`), *not* the
  `reasoning_content` key used by some other OpenAI-compatible servers. This is
  deliberate parity with the runtime the Pi profile is configured against today,
  and it is the key any consumer must read.
- Unlike Python's state machine, the prototype does not treat model-emitted
  `<tool_call>` markers as a separate stripping state — `MLXLMCommon.ToolCallProcessor`
  already removes them upstream and emits `Generation.toolCall`.

### 4.3 Architecture, tokenizer and sampler

- **No unsupported-architecture gap for this model.** `qwen3_5` is registered in
  both `MLXVLM.VLMTypeRegistry` and `MLXLLM.LLMTypeRegistry` at mlx-swift-lm
  3.31.4, and this model's `config.json` decodes into `MLXVLM.Qwen35Configuration`.
  Because the model carries a `vision_config`, `VLMModelFactory` is the factory
  that claims it; the prototype tries VLM then LLM, in the same order
  `ModelFactoryRegistry` uses.
- Factories are referenced **directly** rather than through mlx-swift-lm's
  `NSClassFromString` trampolines. In a statically linked executable that never
  names `VLMModelFactory`, the trampoline class can be stripped, and the
  resulting "no model factory available" error is indistinguishable from an
  unsupported architecture. Naming them keeps a link problem from masquerading
  as an architecture gap.
- **Tokenizer:** loads from the model directory via swift-transformers 1.3.3.
  No gap observed.
- **Chat template:** renders via swift-jinja 2.4.2 including tool declarations.
  No gap observed. Template kwargs are reachable only through
  `ChatSession.additionalContext`, which the prototype uses for
  `reasoning_effort`.
- **Sampler:** `GenerateParameters` exposes `temperature`, `topP`, `topK`,
  `minP`, `seed` and repetition/presence/frequency penalties. The prototype wires
  `temperature`, `top_p` and `seed`. `min_p`, `top_k` and the penalty family have
  **no OpenAI request field wired to them** and are left at MLX defaults — they
  are not refused, because a request cannot ask for them.
- **Default temperature differs from Python.** `GenerateParameters` defaults
  `temperature` to `0.6` and `topP` to `1.0`. When a request omits them, the
  Swift and Python runtimes will not sample identically. Any A/B comparison must
  set both explicitly.

### 4.4 Prototype scope limits

- **One generation at a time.** `GenerationEngine` is an actor, so requests
  serialize. Each concurrent `ChatSession` would allocate its own KV cache, and
  two 75k-context caches beside a 27B 8-bit model do not fit in this host's
  working set. The Pi profile allows `max_leases = 8`, so a production runtime
  would need a real answer here.
- **No prompt caching across requests.** Python runs with
  `--prompt-cache-size 1 --prompt-cache-bytes 8GB`; the prototype builds a fresh
  KV cache per request. This is the most likely source of a large latency
  regression on multi-turn sessions and must be measured before any migration
  decision.
- **No `/v1/completions`, no embeddings, no `/v1/models/{id}`.**
- **No speculative decoding / MTP drafter**, although mlx-swift-lm supports both.
- **No supervision markers.** The example profile deliberately omits a
  `supervision` table: the Python profile's fatal substring
  `RuntimeError: [metal::malloc] Resource limit (` is a Python-runtime string
  and would never match Swift output. An unproven restart marker would paper
  over exactly the failures this prototype exists to observe.
- **No `prompt_tokens_details.cached_tokens`** in `usage`, since there is no
  prompt cache to report.

---

## 5. Full-model smokes — run and green

No longer blocked. The host was free (no `mlx_lm.server` resident), so the four
items that revision 1 could not prove were proven against the real model.

`scripts/smoke.sh` drives the prototype the way the managed contract does:
`model-harness run` owns the process, readiness is polled with the launcher's
own rules, and shutdown is a SIGTERM to the process group.

**SMOKE OK — 36 PASS / 0 FAIL** (exit 0). Latest run, revision 3:

| Measurement | Value |
| --- | --- |
| Load | 6.738 s |
| Physical footprint after load | 29 633 484 064 B (28 260.7 MiB) |
| First 503 from `/v1/models` | 2 s after start |
| Ready | 8 s after start |
| Streaming | 35 frames / 34 chunks, one finish frame, one usage frame, `[DONE]` |
| Tool call | `finish_reason=tool_calls`, well-formed payload with a `call_` id |
| SIGTERM | exit 143 in 1 s, port released, `stopped` event emitted |

`resident_bytes` is unstable across identical loads (2 650.2 / 6 583.1 /
10 774.2 / 14 056.9 MiB observed) while the physical footprint stays within
16 MiB and MLX's active-bytes figure is byte-identical. **Size hosts from the
physical footprint, not from the resident figure.** Recorded, not worked around.

Full detail, including the `swift build` Metal shader library finding that this
smoke exposed: `TASK-260827-qyebv8_full-model-smoke.md`.

`TASK-260827-qyebv8_blocker.md` is a historical record of the revision-1 host
condition. It no longer describes the current state.

---


## 6. Default runtime unchanged

- `/Users/alexis/src/.agents/.configs/model-harness.toml` — not modified;
  `profiles.qwen-local` still executes `/Users/alexis/.local/bin/mlx_lm-relux.server`.
- `/Users/alexis/src/.agents/.configs/project-config.toml` — not modified; the Pi
  profile `qwen-3.8-27b-mlx-8bit` still launches the Python runtime on port 18011.
- The prototype's own profile lives only in the task-scoped
  `.temp/TASK-260827-qyebv8/model-harness-prototype.toml` and in a documented
  example under `tools/mlx-swift-runtime-prototype/examples/`.
- The prototype binary is not installed to `~/.local/bin`.

Python `mlx-lm` remains the rollback path, unchanged and untouched.

---

## 7. Repository changes

| Path | Change |
| --- | --- |
| `tools/mlx-swift-runtime-prototype/` | New SwiftPM package: sources, tests, scripts, example profile, README, `.swift-format`, `.gitignore`, `Package.resolved` |
| `README.md` | One row added to the Tooling table for the prototype |
| `tools/agents-infra/internal/infra/infra.go` | `shouldSkip` now excludes the `.build` and `DerivedData` path components (§2.1) |
| `tools/agents-infra/internal/infra/source_sync_build_artifacts_test.go` | New: positive and narrowing coverage for those exclusions |

`go build ./...`, `go vet ./internal/infra/...`, `gofmt -l` on both changed Go
files and `go test ./internal/infra/... -count=1` all exit 0.

Nothing is committed: `task-board.config.json` sets
`version_control.confirm = true`, so the changes are left in the story worktree
for the owner to confirm.
