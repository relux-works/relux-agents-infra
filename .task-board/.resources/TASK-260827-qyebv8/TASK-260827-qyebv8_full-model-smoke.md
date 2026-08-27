# TASK-260827-qyebv8 — full-model smoke evidence (resume run RUN-260827-5add7a)

The prior run left checklist items 2, 3, 4, 5 and 8 unproven because the host
already held the Python `mlx_lm.server` copy of the same model. The host was
free this run (35.4 GiB free, no `mlx_lm.server` process, `agents-infra runtime
status`: broker absent, leases 0), so the load and generation smokes were run.

They exposed a real defect first, which had nothing to do with memory.

## Finding: `swift build` produces a binary that can never load a model

The first full-model attempt failed at load with:

```
MLX error: Failed to load the default metallib. library not found library not found library not found library not found
  at .../mlx-swift/Source/Cmlx/mlx-c/mlx/c/stream.cpp:115
model-harness: run local profile "qwen-mlx-swift-prototype": exit status 255
```

Root cause is upstream and documented. mlx-swift's own README states verbatim:

> SwiftPM (command line) cannot build the Metal shaders so the ultimate build
> has to be done via Xcode.
> Although `SwiftPM` (command line) cannot build the Metal shaders, `xcodebuild`
> can and it can be used to do command line builds.

Confirmed against the pinned 0.31.6 checkout:

- `Cmlx` declares no `resources:`, so SwiftPM emits no `mlx-swift_Cmlx.bundle`.
- No `.metallib` exists anywhere under `.build` after `swift build -c release`.
- `Package.swift:265` still refers to a `PrepareMetalShaders` step that no
  longer ships in the package.
- `Package.swift:202-203` defines `SWIFTPM_BUNDLE="mlx-swift_Cmlx"` and
  `METAL_PATH="default.metallib"`; `backend/metal/device.cpp` resolves
  `<root>/mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib` across the
  main bundle, all bundles and all frameworks — the four "library not found"
  messages are those four failed lookups.

The prototype README's build instruction (`swift build -c release`) was
therefore wrong: it produced a binary that could never serve a model. Fixed.

The same failed run did vindicate the lifecycle design: the `listening` event
fired and the port bound *before* the load, so the managed readiness poll saw a
live socket answering 503 rather than a refused connection.

## Build path that works

```bash
xcodebuild -downloadComponent MetalToolchain    # ~688 MB, once per host
xcodebuild build \
  -scheme mlx-swift-runtime-prototype \
  -configuration Release \
  -destination 'platform=macOS,arch=arm64' \
  -derivedDataPath ./DerivedData \
  -skipPackagePluginValidation -skipMacroValidation
```

Both skip flags are required for an unattended build:

| Flag | Without it |
| --- | --- |
| `-skipPackagePluginValidation` | `Validate plug-in "CudaBuild" in package "mlx-swift"` fails the build. |
| `-skipMacroValidation` | `Macro "MLXHuggingFaceMacros" from package "mlx-swift-lm" must be enabled before it can be used`. |

Metal Toolchain is a separate Xcode component on this host; without it the
Metal compile step fails with `cannot execute tool 'metal' due to missing Metal
Toolchain`. Installed this run (first-party Apple component for the already
installed Xcode; `metal` reports version 32023.883, target air64-apple-darwin25.5.0).

Products land side by side, which is the layout MLX resolves:

```
DerivedData/Build/Products/Release/mlx-swift-runtime-prototype                                  (79,880,296 B)
DerivedData/Build/Products/Release/mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib    ( 3,817,916 B)
```

## New gate: named refusal instead of an opaque mid-load abort

`MetalShaderLibraryCheck` (Sources/MLXSwiftRuntimeContract/MetalShaderLibraryCheck.swift)
is admitted in `Main.swift` on the `serve` path **before the listener binds**,
and reported as a named `metal_shader_library` stage by `preflight`.

Verified at the real entry point, not just in unit tests — the surviving
`swift build` product was run directly:

```
$ .build/release/mlx-swift-runtime-prototype serve --model <qwen> --host 127.0.0.1 --port 18019
EXIT=2
mlx-swift-runtime-prototype: MLX Metal shader library mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib
was not found next to this executable (searched: .../.build/release, .../.build/arm64-apple-macosx/release).
This binary was almost certainly produced by `swift build`, ... Rebuild with `xcodebuild build ...`.
A Metal Toolchain is required: `xcodebuild -downloadComponent MetalToolchain`.

$ lsof -nP -iTCP:18019 -sTCP:LISTEN
(no listener - refused before binding)
```

Before the change this exact command bound the port and then aborted mid-load.

The gate distinguishes absence from a failed read: a search root that exists but
cannot be inspected yields `.undetermined` and does **not** refuse the launch,
because failing to read a directory is not evidence the library is missing from
it. Covered by `unreadableIsNotAbsence` and by `realUnreadableRoot`, which
chmods a real directory to 0o000.

## Full-model smoke — SMOKE OK (0 failures)

`scripts/smoke.sh` through `model-harness run`, against the real model.

| Measure | Value |
| --- | ---: |
| Load time | 6.977 s (6.146 / 6.225 / 6.977 across three runs) |
| Resident (task_info) | 2,650.2 MiB — see note below |
| Physical footprint | 28,261.4 MiB (28,277.4 / 28,261.7 / 28,261.4) |
| MLX active bytes | 29,501,612,496 |
| MLX peak bytes | 29,501,612,496 |
| MLX cache bytes | 6,568 |
| Host memory | 68,719,476,736 |
| First 503 observed at | 2 s |
| Ready at | 8 s |

**`resident_bytes` is not a usable memory measure here.** Across three full
smokes it read 14,056.9 / 10,774.2 / 2,650.2 MiB for an otherwise identical
load, while `physical_footprint_bytes` stayed within 16 MiB of 28,261 MiB and
`mlx_active_bytes` was identical to the byte. The MLX weight buffers live in
unified memory that `task_info` `resident_size` does not account for
consistently. Anyone sizing a host should read the physical footprint or the
MLX active bytes, not the resident figure. Recorded, not worked around.

Model: `/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit`
(`model_type=qwen3_5`, 8bit/group64, factory `MLXVLM.VLMModelFactory`).

Checks that passed:

- readiness: 503 with an empty OpenAI list while loading, then 200 advertising
  exactly the configured model ID
- `model_loaded` event carries load seconds and resident memory
- live refusals: unknown model -> 404 `model_not_found`; `developer` role -> 400
  `unsupported_role`; `reasoning_effort` -> 400 `unsupported_parameter`;
  `chat_template_kwargs` -> 400
- non-streaming: 200, `chat.completion`, `finish_reason=stop`,
  usage 17 + 34 = 51 consistent, marker `TEXT_RESPONSE_OK` present,
  137 chars of reasoning reported separately from content
- streaming: 35 frames / 34 chunks, exactly one `finish_reason` frame, exactly
  one usage frame, `[DONE]` terminator, marker present, no `</think>` leak
- tool call: `finish_reason=tool_calls`, usage 291 + 60 = 351, payload
  `[{"id":"call_2747ff05cb2242bc9ca6fd78808add2b","type":"function",
  "function":{"name":"write_file","arguments":"{\"content\":\"TOOL_ROUNDTRIP_OK\",\"path\":\"/tmp/mlx-swift-smoke.txt\"}"}}]`
- shutdown: SIGTERM to the process group, exit 143 after 1 s, port released,
  `stopped` event emitted

### Reasoning-content behaviour (recorded)

Reasoning is published as `choices[].message.reasoning`, matching
`mlx_lm.server`, which is the runtime the Pi profile is configured against.
The key `reasoning_content` is **absent** — a generic OpenAI client expecting
`reasoning_content` would not see it. `system_fingerprint` is
`mlx-swift-runtime-prototype-3.31.4 (bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57)`.

## Preflight (real model, no weights)

`preflight` exit 0, every stage passed, including the new one:

```
metal_shader_library: passed -> .../Release/mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib
base_configuration: passed   -> model_type=qwen3_5 quantization=8bit/group64
architecture_registry: passed-> qwen3_5 implemented by MLXVLM, MLXLLM
vlm_configuration_decode: passed -> MLXVLM.Qwen35Configuration
tokenizer_load: passed
chat_template: passed        -> 260 tokens with 1 tool declaration
generation_starts_in_reasoning: passed -> prompt ends with an open <think> block
tool_call_format: passed     -> xml_function
```

## Lifecycle smoke — LIFECYCLE SMOKE OK (0 failures), exit 0

17/17 against the unloadable fixture (no GPU memory): startup refusals,
listener-before-load, 503 while not ready, `model_not_ready`, a
`model_load_failed` event for an unsupported architecture, 503 preserved after a
failed load, SIGTERM exit 0, stopped event, port released.

## Gate commands and real exit codes

| Command | Exit |
| --- | ---: |
| `swift build -c release` | 0 |
| `swift test -c release` (92 tests / 9 suites) | 0 |
| `swift format lint --strict --recursive Sources Tests` | 0 |
| `xcodebuild build -scheme ... -configuration Release ...` | BUILD SUCCEEDED, 0 `error:` lines |
| `preflight --model <qwen>` | 0 |
| `scripts/smoke.sh` (full model) | 0 — SMOKE OK, 25 PASS / 0 FAIL |
| `scripts/lifecycle-smoke.sh` | 0 — LIFECYCLE SMOKE OK |
| `go test ./internal/infra/ -run TestShouldSkip -count=1` | 0 (4 tests) |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | no files |
| `go test ./internal/infra/... -count=1` | 0 (122.3 s) |

Test count moved 77 -> 92: 15 new cases in the `metal shader library admission`
suite, plus 2 new Go tests for the `DerivedData` exclusion.

## Mutation evidence (gates narrowed, not just deleted)

| Mutant | Change | Result |
| --- | --- | --- |
| M1 | `classify` drops the `.undetermined` guard, so an unreadable root counts as absence | CAUGHT — `unreadableIsNotAbsence` |
| M2 | `admit` stops throwing on a proven `.absent` | CAUGHT — `absentRefuses` |
| M3 | `inspect` accepts the bundle *directory* as proof instead of the metallib *file* | CAUGHT — `bundleWithoutLibraryIsAbsent` |
| M4 | `composeSearchRoots` drops the executable directory, shrinking the search | CAUGHT — `executableDirectoryIsItsOwnRoot` |
| A | Go: delete the `DerivedData` exclusion | CAUGHT — `TestShouldSkipExcludesXcodeDerivedData` |
| B | Go: narrow `hasPathComponent` to `strings.Contains` | CAUGHT — `TestShouldSkipKeepsPathsNamedLikeDerivedData` |

M4 was **not** caught on the first attempt. The original test asserted against
`defaultSearchRoots()`, where a test host's bundle URL and executable directory
are the same path, so removing one of them changed nothing observable. The root
composition was extracted into a pure `composeSearchRoots` and asserted there;
M4 is caught now. Reported because a test that cannot fail is not evidence.

## Second regression found and fixed: DerivedData repeats the .build hazard

Building through `xcodebuild -derivedDataPath ./DerivedData` writes a 2.6 GB
tree into the source directory — the same hazard as the `.build` directory that
broke `setup local` in the prior run, and `shouldSkip` did not exclude it.

- `tools/agents-infra/internal/infra/infra.go`: `DerivedData` joins `.build` and
  `.temp` as a skipped path component.
- `tools/agents-infra/internal/infra/source_sync_build_artifacts_test.go`: two
  new tests, one for the exclusion and one pinning it to a whole-path-component
  match so it cannot swallow `DerivedDataPolicy.go`.
- `tools/mlx-swift-runtime-prototype/.gitignore`: `DerivedData/`.

## Defaults unchanged

`model-harness.toml` and `project-config.toml` untouched. The prototype profile
lives only in `.temp/TASK-260827-qyebv8/model-harness-prototype.toml` and is
passed with `--config`; nothing is installed and no binary is placed on PATH.
Python `mlx-lm` remains the default local runtime and the rollback path.

## Gap list (unchanged from the prior run, plus one new build gap)

New: **a SwiftPM command-line build cannot produce a runnable MLX binary.** The
build must go through `xcodebuild` and needs the separately downloaded Metal
Toolchain plus two validation-skip flags. This is an upstream constraint, now
named and refused at startup rather than worked around.

Still open, unchanged: one generation at a time, no prompt caching across
requests, no `/v1/completions`, no embeddings, no speculative decoding, no
supervision markers, and `reasoning_content` is not published under that name.
