# mlx-swift-runtime-prototype

A task-scoped prototype (TASK-260827-qyebv8) that serves the configured local
Qwen model through **MLX Swift LM** behind the same OpenAI-compatible surface
the Pi profile already uses against Python `mlx_lm.server`.

It exists to answer one question with evidence: *can an MLX Swift LM runtime
stand in for the Python one without changing the managed contract?* It is not a
replacement, it is not installed, and it does not change any default.

> **The default local runtime is unchanged.** `profiles.qwen-local` in the
> operator's `model-harness` config still points at Python `mlx_lm.server`, and
> the Pi profile `qwen-3.8-27b-mlx-8bit` still launches it. Nothing in this
> directory is referenced by an installed config. Python `mlx-lm` remains the
> rollback path.

## Pinned upstream revisions

Dependencies are pinned with `exact:` so every recorded measurement names the
code that produced it. `Package.resolved` is committed and pins the transitive
graph by revision.

| Package | Version | Revision |
| --- | --- | --- |
| `ml-explore/mlx-swift` | 0.31.6 | `0bb916c67f4b9e5c682cbe02a42c701c93ab5021` |
| `ml-explore/mlx-swift-lm` | 3.31.4 | `bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57` |
| `huggingface/swift-transformers` | 1.3.3 | `2fa33e1f5e7131a7fc64c28e6d161dcec0d24820` |
| `apple/swift-nio` | 2.99.0 | `f71c8d2a5e74a2c6d11a0fbe324774b5d6084237` |

The same two MLX revisions are compiled into the binary
(`Sources/mlx-swift-runtime-prototype/PinnedRevisions.swift`) and reported on
the `listening` event and as `system_fingerprint` on every completion.

## Build

> **`swift build` cannot produce a runnable binary.** mlx-swift's own README
> states it verbatim: *"SwiftPM (command line) cannot build the Metal shaders so
> the ultimate build has to be done via Xcode."* A SwiftPM-built product has no
> `mlx-swift_Cmlx.bundle`, so MLX aborts with `Failed to load the default
> metallib` at the first GPU touch. The runtime now refuses to start in that
> state instead of crashing mid-load — see the refusal table below.

The Metal Toolchain is a separately downloaded Xcode component. Install it once:

```bash
xcodebuild -downloadComponent MetalToolchain    # ~688 MB, first time only
```

Then build with `xcodebuild`:

```bash
cd tools/mlx-swift-runtime-prototype
xcodebuild build \
  -scheme mlx-swift-runtime-prototype \
  -configuration Release \
  -destination 'platform=macOS,arch=arm64' \
  -derivedDataPath ./DerivedData \
  -skipPackagePluginValidation -skipMacroValidation
```

`-skipPackagePluginValidation` is required because mlx-swift attaches a
`CudaBuild` build-tool plugin, and `-skipMacroValidation` because mlx-swift-lm
uses an `MLXHuggingFaceMacros` macro. An unattended `xcodebuild` cannot answer
the interactive trust prompts these otherwise raise, and fails the build.

The product and the compiled shader bundle land side by side, which is exactly
the layout MLX resolves `default.metallib` from:

```
DerivedData/Build/Products/Release/mlx-swift-runtime-prototype
DerivedData/Build/Products/Release/mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib
```

Moving the executable away from that directory breaks the lookup.

## Test

The contract suite is pure Swift with no MLX GPU dependency, so it still runs
under SwiftPM:

```bash
swift test -c release           # contract suite, no model load required
```

## Run

```bash
DerivedData/Build/Products/Release/mlx-swift-runtime-prototype serve \
  --model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
  --host 127.0.0.1 --port 18017 \
  --reasoning-effort medium
```

| Flag | Meaning |
| --- | --- |
| `--model` | Absolute path to the local model directory. Required. |
| `--model-id` | ID advertised on `/v1/models`. Defaults to `--model`, matching what `mlx_lm.server` publishes for a local directory. |
| `--host` | Must be `127.0.0.1`. Any other value is refused. |
| `--port` | Required, 1–65535. |
| `--max-kv-size` | Rotating KV-cache bound. Unset means an unbounded cache. |
| `--default-max-tokens` | Applied when a request omits `max_tokens`. Default 2048. |
| `--reasoning-effort` | Chat-template kwarg: `low`, `medium` or `xhigh`. The template's own default is `xhigh`, which injects an extra system instruction — worth ~38 tokens on every prompt — so any comparison has to state it. |
| `--fault-inject-generation-error` | Acceptance-suite fault seam. Makes generation fail with this exact text. Off unless given. See [Dead generation worker](#dead-generation-worker). |
| `--fault-inject-generation-error-count` | Bounds the seam to the first N generation attempts. Unset means every attempt, which is what the dead-generation suite needs. A bound is what makes *recovery* observable. Requires `--fault-inject-generation-error`. |
| `--fault-inject-generation-error-after-tokens` | Fires the seam only after N tokens have reached the client, so the batch and its KV cache exist at the moment of failure. Default `0` — fire before MLX is touched. Requires `--fault-inject-generation-error`. |
| `--fault-inject-teardown-retain` | `true`/`false`. Parks the condemned worker's `ModelContainer` for the lifetime of the process, so its weights are never released and the teardown's release barrier genuinely never fires. Drives the fail-closed branch of the acceptance suite. Default `false`. Requires `--fault-inject-generation-error`. |
| `--fault-inject-teardown-retain-weights` | `true`/`false`. Parks `ModelContext.model` — the weights *below* the container — while letting the container itself be deallocated on schedule. So the wrapper's `weak` reference really does read `nil` with the whole model still resident, which is the one interval the flag above cannot produce. Default `false`. Requires `--fault-inject-generation-error`. |
| `--fault-inject-teardown-retain-weight-modules` | `true`/`false`. Parks a **strict subset** of the model's module tree — the second half of `model.modules()` — and lets the container, the root model object and the rest of the tree die on schedule. MLX then reports a residue *below* the model's load footprint, which is what a fully released model looks like to a process-global counter, while this model's weights are still owned. The one seam no byte comparison can refuse. Default `false`. Requires `--fault-inject-generation-error`. |
| `--fault-inject-teardown-retain-weight-arrays` | `true`/`false`. Parks the flattened parameter **arrays** and no object of the model tree at all, so every `Module` and the container above them die on schedule while MLX goes on calling the whole footprint active. Ownership then reports the model released and only the absolute-residue reading can refuse. Default `false`. Requires `--fault-inject-generation-error`. |
| `--fault-inject-teardown-retain-weight-array-subset` | `true`/`false`. Review's revision-5 bypass, kept as a maintained input. Same mechanism as the flag above but narrowed to the **largest half by `nbytes`**, so ownership still reports the model released while the residue lands significant and yet *strictly below* the load footprint — measured `255,724,192 B` of `262,361,760 B`. That is the interval any footprint-relative residue check admits, and the reason the allowance is now zero. Default `false`. Requires `--fault-inject-generation-error`. |

`{host}` and `{port}` are the literal tokens `model-harness` substitutes, so the
argv above drops straight into a local profile — see
`examples/model-harness.prototype.toml`.

## Endpoints

| Route | Behaviour |
| --- | --- |
| `GET /v1/models` | `503` with an empty OpenAI list until the weights are resident; then `200` advertising exactly the configured model ID. |
| `GET /health` | `200 {"status":"ok"}` only when the runtime can actually generate. `503` while loading, while shutting down, after a failed load, and after the generation worker is condemned — the last as `{"status":"unavailable"}`. |
| `POST /v1/chat/completions` | Non-streaming JSON, or SSE when `stream: true`. |
| `GET /debug/generation-state` | Read-only batch/cache accounting: in-flight generations, started/completed/failed, batches released, shared-cache rebuilds — completed, abandoned, and still owed — and MLX allocator figures. Always `200`, including once the worker is condemned. See [Generation batch recovery](#generation-batch-recovery). |

Response shape tracks `mlx_lm.server` rather than a generic OpenAI client,
because that is the runtime the Pi profile is configured against today:
reasoning is published under `reasoning` (not `reasoning_content`), and
`system_fingerprint` is present on every packet.

## What it refuses, and why

The prototype refuses requests it cannot honour instead of answering `200` with
the field ignored. A silent drop reports a constraint that was never applied.

| Refusal | Status / code | Reason |
| --- | --- | --- |
| `model` ≠ the loaded model ID | `404` `model_not_found` | The process serves exactly one model. Matching is exact — not prefix, substring, basename or case-insensitive. |
| `role: "developer"` (and any other unknown role) | `400` `unsupported_role` | The Pi profile sets `supports_developer_role = false`; folding it into `system` would change the prompt silently. |
| `reasoning_effort` | `400` `unsupported_parameter` | `supports_reasoning_effort = false`. Reasoning effort is a chat-template kwarg chosen at startup, not a per-request field. |
| `chat_template_kwargs` | `400` `unsupported_parameter` | The `</think>` splitter is configured from the startup template state. A request that flipped `enable_thinking` would have its whole answer filed as `reasoning`. |
| `stop`, `logit_bias`, `top_logprobs`, `response_format`, `n != 1`, `logprobs: true`, `tool_choice` other than `auto` | `400` `unsupported_parameter` | Not implemented; honouring them would change the result. |
| non-text content parts | `400` `invalid_body` | The profile declares `input = ["text"]`. |
| `--host` other than `127.0.0.1` | exit 2 | A model-harness plan can only present loopback. |
| missing / unreadable / incomplete model directory | exit 2 | No downloader is linked, so a bad path has no fallback. An unreadable directory is reported as unreadable, never as absent. |
| `max_tokens` outside the `Int` range (`1e300`, `9223372036854775808`, …) | `400` `invalid_body` | A JSON number too large for `Int` decodes as a `Double`, and converting it with `Int(_:)` is a Swift fatal error. The conversion is total, so an out-of-range value is refused like any other bad field instead of killing the runtime. |
| no `mlx-swift_Cmlx.bundle` next to the executable | exit 2 | A `swift build` product cannot reach the GPU. Refused before the port binds, so the failure is named instead of aborting mid-load. An unreadable search root suppresses the refusal — a directory we could not read is not evidence the library is gone. |
| a `default.metallib` that is not a regular file (directory, dangling symlink) | exit 2 | Existence is not usability, and `FileManager.fileExists` cannot tell them apart. The object must be a regular file, or a symlink resolving to one, or the launch is refused and the message names what is actually there. |

`seed` **is** honoured and forwarded to the sampler.

## Dead generation worker

`200` on `/health` means *this runtime can produce a token*, not *this process
is running*. The difference is a recorded incident, not a hypothetical.

In `mlx_lm.server`, generation runs on a long-lived thread. An uncaught
exception killed that thread while the HTTP listener stayed up, so `/health`
kept answering `200 {"status": "ok"}` for a runtime that could no longer
generate anything, and callers discovered it as a request timeout instead
(`BUG-260827-1jhv2g`: the `qwen3_5` `ArraysCache` leak exhausting MLX's Metal
allocator with `[metal::malloc] Resource limit (499000) exceeded`). Upstream
fixed it in `BUG-260827-2tul5n` by reporting `503 {"status": "unavailable"}`
whenever `_generation_thread.is_alive()` is false.

Swift has no equivalent silent death — an error thrown inside a structured task
propagates to its caller rather than tearing a worker down — so the same
regression lands here as **invalidation**. When a generation fails with a
signature that condemns the backend rather than the request, the runtime:

1. drops the engine and moves to `generationWorkerFailed`, which is terminal;
2. answers `/health` `503 {"status": "unavailable"}` with the failure as `detail`;
3. stops advertising the model on `/v1/models`, which is what the managed
   launcher actually polls, and answers `503`
   `generation_worker_unavailable` there;
4. refuses further completions with `503` at the readiness gate rather than
   handing out an engine already known to be broken;
5. writes one line carrying the literal marker `generation_worker_unavailable`,
   which a `model-harness` `fatal_output_substrings` policy turns into a
   supervised restart. See `examples/model-harness.prototype.toml`.

The runtime never restarts itself. Recovery is the supervisor's job, and a
runtime that quietly healed itself would hand the next caller a backend it
already knows is broken.

### Getting a real MLX failure to that classifier at all

None of the above ran for the one failure it exists for, until this revision.

`MLX/ErrorHandler.swift` dispatches to `fatalError` when no handler is
installed, and the runtime installed none. So MLX's C++ layer did not raise a
Swift error that `GenerationEngine.generate` could catch — it killed the
process. Review confirmed it against a real 73k-token prompt: the runtime
trapped with `Fatal error: [metal::malloc] Attempting to allocate
255904140288 bytes ...` and emitted no `generation_worker_failed`, no `503`, no
supervision marker and no teardown event.

`GenerationEngine.run` now enters `MLX.withError` before generation, checks the
`ErrorBox` after every streamed item, and rethrows whatever MLX reported as
`GenerationError.mlx(_:)` carrying the message verbatim — which is the same
string the `--fault-inject-generation-error` seam feeds the classifier, so both
paths reach it with the same bytes.

Two consequences worth stating plainly:

- The oversized-allocation message (`allocator.cpp:111-117`) is deliberately
  **not** an invalidating signature. It rejects one allocation and leaves the
  pool intact, so it belongs to the request that asked for it: the request gets
  a `500`, `/health` stays `200`, and the process survives. Before this change
  the same condition killed the runtime.
- The handler is a **task local**. An error raised on a thread MLX owns — an
  `asyncEval` worker with no task context — still reaches the global default and
  still traps. The runtime does not install a process-global handler, because
  returning from one is what leaves MLX's C++ running on state it has already
  declared invalid.

### What condemns the worker, and what does not

Only failures that name a broken backend condemn it —
`GenerationWorkerHealth.invalidatingSignatures`, currently two entries, each a
**conjunction** of fragments that must all be present:

| Signature | Requires | Why |
| --- | --- | --- |
| `metal-allocator-resource-limit` | `[metal::malloc]` **and** `Resource limit` | The `BUG-260827-1jhv2g` incident. MLX emits `[metal::malloc] Resource limit (N) exceeded.` verbatim from `mlx/backend/metal/allocator.cpp:141-144`; the pool is exhausted and the next request cannot be served either. |
| `metal-shader-library-unreachable` | `Failed to load the default metallib` | MLX cannot reach the GPU at all. The startup gate refuses that build up front, so seeing it *during* generation means the shader library went away under a running process. |

The list is short on purpose, and conjunctive on purpose. The gate is dangerous
in both directions:

- **Too broad.** Condemning on *every* generation error would be a worse bug
  than the one being fixed: one malformed request would take a healthy runtime
  out of rotation and, under supervision, restart it. Matching `Resource limit`
  on its own does the same thing more quietly —
  `RequestError: Resource limit for this request is 8 tokens` is a
  request-scoped refusal that carries the phrase and nothing else. So does the
  allocator's *other* throw (`allocator.cpp:111-117`,
  `[metal::malloc] Attempting to allocate N bytes which is greater than the
  maximum allowed buffer size`): it rejects one oversized allocation and leaves
  the pool intact.
- **Too narrow.** A signature that matches nothing turns the endpoint back into
  a liveness check for the process.

A signature with no fragments matches nothing rather than everything —
`allSatisfy` is vacuously true on an empty sequence, so failing closed there is
what keeps a bad edit from silently producing a condemn-all gate.

A cancelled generation, a missing usage packet and a rejected parameter are all
request-scoped and leave readiness untouched.

Only a `ready` runtime transitions. A generation that fails because `SIGTERM`
arrived mid-request must not be reported as a dead worker, and must not ask the
harness to restart a process that was deliberately asked to stop.

## Generation batch recovery

The section above is about the failure a runtime cannot survive. This one is
about the far more common one it must: a generation that blows up mid-batch on
a runtime that is still perfectly able to serve the next caller.

In `mlx_lm.server` that case was not survivable either. The exception escaped
the batch loop, killed the generation thread, and took the batch entry and its
KV cache with it into a process that kept listening. Here the error propagates
to its caller, so the runtime *can* recover — and therefore does so
deliberately and observably, rather than by hoping ARC gets to the session
before the next request needs the memory.

When a generation fails, `GenerationBatchRecovery.plan(after:)` decides what
its state owes back:

| | Always | Only when implicated |
| --- | --- | --- |
| **Batch entry + its KV cache** | released | — |
| **MLX's shared buffer pool** | — | dropped via `Memory.clearCache()` |

The split is the whole design. The per-request KV cache belongs to the request,
so it is returned under *every* verdict, including a condemned one — the
supervisor is about to start a replacement that needs the host's memory. The
shared pool is different: it is what makes every *other* request fast, and
clearing it for an ordinary request-scoped error would be a self-inflicted
latency regression paid for a failure that never touched it.

Membership in the pressure class is decided from the pinned allocator source,
not from how allocation-shaped a message reads. The question is narrow: **can
`clear_cache()` change the outcome of the next attempt?**

| Signature | Requires | Why |
| --- | --- | --- |
| `metal-buffer-allocation-failed` | `[malloc]` **and** `Unable to allocate` | The throw taken when `newBuffer` returns null. Reaching it means the allocator already ran `release_cached_buffers(mem_required - gc_limit_)` — a *slice* sized only to get back under `gc_limit_` — and still came up empty. `clear_cache()` empties `buffer_cache_` outright, so it can hand back what that partial reclaim kept. Does **not** condemn: one allocation failed and the backend can still serve. |

Deliberately **not** in the class: `[metal::malloc] Attempting to allocate N
bytes which is greater than the maximum allowed buffer size`. It reads like
allocation pressure and is not. `MetalAllocator::malloc` throws it as its first
act, before `std::unique_lock lk(mutex_)` and before any cache is consulted,
testing `size > device_->maxBufferLength()`; `clear_cache()` cannot move that
limit. Revision 1 shipped it as a pressure class and charged every subsequent
generation a cold pool to recover from a failure the pool can neither cause nor
repair. It is now carried as a narrowing negative in both suites.

Conjunctive, for the same reason `GenerationWorkerHealth`'s signatures are —
the fragments assert two things at once and either half alone is a phrase an
unrelated failure can carry. `[malloc]` is not a substring of `[metal::malloc]`,
so the pressure and condemning classes cannot collide through their tags.

A condemning failure implies a rebuild too, but a **deferred** one. The weights
are still held while the failing request unwinds, so the clear waits for the
engine to be deallocated and for the condemned `ModelContainer` to be *observed*
released; only then is there anything in the pool worth returning. Getting that
ordering wrong is silent, and was the shape of a real defect: the pool was
emptied while the model was still active, and the whole model landed in it a
moment later.

"Observed" is literal, and is the second half of that defect. It is a
conjunction of readings, and every clause in it is there because review defeated
the set without it.

**The container veto.** A `weak` reference to the exact container the condemned
engine was serving from. While that reads non-`nil` the weights are certainly
still held, so it is a cheap and exact veto — but it is not an attestation, and
shipping it as one was a defect of its own. `ModelContainer` is a wrapper: the
weights live below it in `SerialAccessContainer<ModelContext>` →
`ModelContext.model`, and a Swift `weak` reference may read `nil` while
destruction of that stored state is still running. Review demonstrated it twice
— a probe observing `payload-deinit-start`, `weak-nil`, `payload-deinit-finish`
in that order, and then a mutant that delayed *only* the pinned
`SerialAccessContainer<ModelContext>` destruction and made the runtime clear the
pool and attest a completed rebuild with all 262,361,760 bytes of model still
active.

**Ownership, read from the model tree.** Every `Module` in the tree rooted at
`ModelContext.model` is registered at load, weakly, in `WeightOwnerRegistry`.
Every weight array of an MLX Swift model is a stored property of one of those
objects, so "none of them is alive" is a statement about *this model's* weights.
This is the clause that **attributes**, and it exists because MLX's counters
cannot. An earlier revision answered the release question from a process-global
byte delta — `baselineActiveBytes - activeBytes >= weightFootprintBytes` — and
review drove the same production path with a six-thousand-word prompt, which
makes the failed request's own KV state larger than the model. Releasing the
request alone satisfied the subtraction: `608,909,592 B` "returned" against a
`262,361,760 B` footprint, a completed rebuild attested, and post-teardown
`active_bytes` sitting at exactly `262,361,760` — every weight still resident.
A registry that was never populated reports zero live owners forever, so an
unpopulated registry fails the gate closed.

**Absolute residue, against an allowance of zero.** MLX's `activeMemory` must be
at or below `GenerationBatchRecovery.residualNonWeightAllowanceBytes`, and that
constant is **`0`**. The question is what is still resident, not what moved.

An earlier revision compared the residue against `LoadedModel.weightFootprintBytes`
instead, which admits a *band*: everything strictly below one full model. Review
walked a production input into that band — a strict subset of this model's own
parameter arrays, copied out so that all `316` registered `Module` owners died
and ownership reported the model released, sitting at `255,724,192 B` of a
`262,361,760 B` footprint with `returned_bytes` clearing the footprint as well.
Every clause read green and the runtime attested a completed release over
~97.5% of the model.

The band is removed rather than narrowed, because any allowance `A > 0` is a
promise that nothing weight-sized fits underneath it, and a process-global
counter cannot keep that promise: there is no reading that separates `A` bytes
of sampler state from `A` bytes of retained weights. At zero the question does
not arise — `activeMemory == 0` means no MLX buffer of any kind is alive, so no
*weight* buffer is alive either. It is the one reading a process-global counter
can carry an attribution claim on, because it leaves nothing to attribute. The
maximum admissible residue is therefore **0 bytes**, and `--fault-inject-teardown-retain-weight-array-subset`
keeps review's exact bypass in the acceptance suite as a maintained input.

The old delta is kept as a *necessary* condition — the fall has to have happened
inside this teardown window — and is never again sufficient for anything.

**The measured cost, stated rather than engineered around.** This runtime's own
*clean* condemned teardown leaves `2,720 B` of post-generation MLX state
(sampler and RNG arrays) active. That is not zero, so the clean path abandons
too, and a completed rebuild is essentially **never** attested here. The pool is
then left holding the freed model — the acceptance suite measures `331,887,724 B`
of `cache_bytes` — until the supervisor replaces the process the abandonment's
marker already demands. A false abandonment costs a restart that condemnation
had made necessary anyway; a false attestation tells an operator the host is
free with a condemned model still on it.

**Idle and at rest.** The reading is refused while any generation is still in
flight, because a concurrent request's allocation and release cannot be
attributed either, and it is refused until `activeMemory` has reported the same
value for `GenerationBatchRecovery.minimumStableActiveSamples` consecutive
polls. A destruction still running is a falling count, and a single sample taken
during that fall is a partial release read as a finished one.

None of this is the proportional aggregate a still earlier revision used. That
one waited for process-global `activeMemory` to fall below *half* its
condemnation-time value, which at the 29 GB target model admits the clear with
~14.5 GB still active.

An unmeasured footprint — `0` — fails closed rather than admitting everything,
as does an unpopulated owner registry. That is the absence-versus-failure-to-read
rule: a footprint that was never measured is not a model that cost nothing, and
a comparison against it would be satisfied by any reading at all.

What this deliberately does **not** claim: MLX's counters are process-global and
no clause here can attribute an individual byte. The gate is built to refuse
everything it cannot account for — ownership is read from the model tree rather
than inferred from the allocator, the allocator must be idle and at rest before
it is believed, and *any* residue at all is a refusal. Weak `Module` ownership is
a veto and never a proof: copied `MLXArray` values outlive the modules that held
them, which the retention seams below demonstrate directly. A false abandonment
costs a supervision marker and a replacement process; a false attestation tells
an operator the host is free while a condemned model is still holding it.

The wait is bounded, and **fails closed**. An unobserved release cannot reach
the clear at all: after
`GenerationBatchRecovery.workerTeardownAttempts` the runtime performs no
`Memory.clearCache()`, attests no rebuild, leaves `shared_cache_rebuild_pending`
raised, counts the attempt in `shared_cache_rebuilds_abandoned`, emits
`generation_shared_cache_rebuild_abandoned` with `release_observed=false`, and
re-announces the supervision marker — a host holding a condemned model it could
not return must not be left competing with its replacement for that memory. The
earlier revision discarded the timeout and took the success transition either
way, so a failure to observe was reported exactly like an observation.

The readiness transition has exactly one owner, `GenerationWorkerHealth`. The
plan reports the condemning signature but never acts on it, so recovery cannot
resurrect a worker the health gate condemned: a request arriving after
condemnation is still refused `503` at the readiness gate, even when the
injected fault would not have repeated.

`GET /debug/generation-state` publishes the accounting, and outlives the engine
that condemnation drops. `active` is the only non-monotonic counter, which is
the point: a runtime that failed a generation without giving the slot back
shows up there and nowhere else. `batches_released` is tracked separately from
`failed` rather than derived from it — deriving it would make the invariant
unfalsifiable.

The endpoint also publishes MLX's own allocator figures, and **those** are what
the acceptance suite asserts resource claims against. The ledger's counters say
what the runtime believes; `mlx.active_bytes` and `mlx.cache_bytes` say what the
allocator is actually holding. `sharedCacheRebuilds` is incremented one line
after `Memory.clearCache()` returns and nowhere else, so a deleted clear cannot
leave a counter behind to vouch for it.

## Generation-batch-recovery smoke

```bash
BINARY=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
HARNESS=/Users/alexis/.local/bin/model-harness \
MODEL=/Users/alexis/.cache/huggingface/hub/models--mlx-community--Qwen1.5-0.5B-Chat-4bit/snapshots/659d8dafc39202a6688bb46242d60440702489b1 \
PORT=18021 OUT=./batch-recovery-out \
scripts/generation-batch-recovery-smoke.sh
```

193 checks on the real Release binary under the real `model-harness`, in
fifteen phases. Every resource claim is anchored to MLX's allocator figures rather than
to a ledger counter:

| Phase | What it establishes |
| --- | --- |
| recovery | A fault fired *after* real tokens reached the client ends that request `500` — never a truncated `200` — leaves `/health` at `200`, returns the batch slot, and the **next request completes on the same process** with exactly one listener ever bound. |
| streaming | The same through SSE, which is a separate production call site: partial chunks, then a terminal error frame and **no** `finish_reason`, then a clean next stream. |
| multi-fault | `--fault-inject-generation-error-count 2` fails exactly two requests and serves the third, so the seam's own arithmetic is checked rather than assumed. |
| leak | **Allocator-bound.** `mlx.active_bytes` must not grow across failed generations. Catches a runtime that closes the ledger slot while retaining the failed `ChatSession` — `+25,165,824 B` of KV per failure, and completely invisible to the counters. |
| no-rebuild | **Narrowing**, and the control for the next phase: an ordinary failure releases the batch and leaves the shared pool alone. |
| rebuild | **Allocator-bound.** An exhausted allocation also drops the shared pool, measured as `cache_bytes` well below the control run — same model, prompt and pinned seed, so only the clear differs. Catches deletion of the production `Memory.clearCache()` while its counter and event survive. |
| oversize | **Narrowing.** A `maxBufferLength` rejection must **not** drop the pool, because the pool cannot repair it. |
| condemned (unsupervised) | **Narrowing**, and the clean-path cost of the zero allowance. The recorded exhaustion still moves `/health` to `503`, still refuses the next request, still emits the marker — and still releases its batch. The teardown is genuinely clean: container gone, `0` of `316` owners live, nothing in flight, `302` identical samples, `303,847,812 B` returned against a `262,361,760 B` footprint. It **still abandons**, because the `2,720 B` residue is not zero. The phase asserts every other clause green, so the refusal is attributable to the residue alone, and asserts the cost: `331,887,724 B` of pool left held and the marker demanding replacement. A revision that quietly cleared without attesting reddens the cost gate alone. |
| condemned (supervised) | The marker still produces a supervised restart and the replacement answers `200`. |
| condemned (retained teardown) | **Negative, allocator-bound.** `--fault-inject-teardown-retain true` holds the real container, so the release genuinely never happens. The runtime must then report the pool as *owed*: `shared_cache_rebuilds` `0`, `shared_cache_rebuild_pending` `true`, an explicit `release_observed=false` event, the marker re-announced, and `/health` still `503`. The residue is measured from `mlx.active_bytes`, so a seam that retained nothing cannot pass the phase by default. |
| condemned (inner retention) | **Negative, narrowed** — the interval the phase above cannot reach. `--fault-inject-teardown-retain-weights true` parks `ModelContext.model` and lets the `ModelContainer` die on schedule, so `container_deallocated` is `true` while `mlx.active_bytes` still reports the whole model. A runtime that answers the release question from the wrapper alone clears the pool and attests a rebuild here; this phase requires it to abandon instead. Measured: `41,488,772 B` of `262,361,760 B` returned, `shared_cache_rebuilds` `0`, `/health` `503`. |
| condemned (long context) | **Negative**, and review's own reproduction of the process-global-delta bypass. The same inner retention, driven by a six-thousand-word prompt so the failed request's KV state outweighs the model. The phase *asserts the bypass condition is present* — `returned_bytes` `608,909,576` against a `262,361,760 B` footprint, which an earlier revision attested from — and then requires the runtime to abandon with `active_bytes` at `262,361,760`. Without the first assertion the phase would be a slower inner-retention run that proved nothing. |
| condemned (module subset) | **Negative, narrowed**, and the class no byte comparison can reach. `--fault-inject-teardown-retain-weight-modules true` parks a strict subset of the module tree — measured `158` of `316` owners live — so `mlx.active_bytes` lands at `174,944,928`, *below* the `262,361,760 B` footprint, while `returned_bytes` reaches `696,650,012`. Every byte-derived clause of the release gate reads green and the phase asserts each one; only ownership refuses. A runtime that dropped the registry passes every other negative and fails this one alone. |
| condemned (array subset) | **Negative, narrowed**, and review's revision-5 bypass kept as a live production input. `--fault-inject-teardown-retain-weight-array-subset true` holds the largest half of the parameter arrays by `nbytes`, so `container_deallocated` is `true`, `live_weight_owners` is `0` of `316`, nothing is in flight, the reading is at rest, and `returned_bytes` reaches `615,547,160` — while `active_bytes` sits at `255,724,192`, *below* the `262,361,760 B` footprint. That is the interval the weight-array phase cannot produce (its residue is at or above the footprint) and the module-subset phase cannot either (ownership refuses first). Restoring the footprint-relative gate reddens 32 checks here and in the clean-path phase. |
| condemned (weight arrays) | **Negative, narrowed**, and the mirror of the module-subset phase: the production negative for the absolute-residue clause at or above the footprint. `--fault-inject-teardown-retain-weight-arrays true` holds the parameter arrays and nothing that owns them, so `container_deallocated` is `true`, `live_weight_owners` is `0` of `316`, nothing is in flight, the reading is at rest and `returned_bytes` reaches `608,909,584` — every clause except one says released — while `active_bytes` sits at the full `262,361,760`. The phase asserts each satisfied clause, so the refusal is attributable to the residue reading and to nothing else. |

The last seven phases are separate runs for the same reason the dead-generation
suite splits its own: with a policy attached, `model-harness` kills the runtime
within milliseconds of the marker, destroying the `503` window the phase exists
to observe.

The narrowing phases are what make the rest mean anything. Everything in the
first four is satisfied by a runtime that recovered from *every* failure,
including the one that means the backend is gone — and that runtime would hand
the next caller a dead worker while answering `/health` `200`, which is the
exact incident the dead-generation work exists to end.

Unlike the dead-generation smoke, this one **does** reach the weights: the
fault fires after real tokens, which is the only way the batch has anything to
release. Requests pin `seed`, so what the runtime is asked to recover from does
not vary run to run.

## Lifecycle

1. The model directory is admitted before anything binds.
2. The listener binds **before** the model loads, so the managed launcher's
   readiness poll sees `503` (keep waiting) rather than a refused connection.
3. `model_loaded` is written to stdout as one JSON line carrying load seconds,
   resident bytes and physical footprint. A `task_info` failure is reported as
   `null`, never as `0`.
4. `SIGTERM`/`SIGINT` closes the listener and exits `0` within the profile's
   `shutdown_timeout_seconds`.

`model-harness` forwards a managed child's stdout and stderr unchanged, which is
why the runtime reports through one-line JSON events on stdout.

## Smoke

```bash
HARNESS=/path/to/model-harness \
HARNESS_CONFIG=/absolute/path/to/model-harness-prototype.toml \
PORT=18017 OUT=./smoke-out \
scripts/smoke.sh
```

The script renders the plan, starts the runtime through `model-harness run`,
polls `/v1/models` with the launcher's own readiness rules, checks the four
refusals above against the live server, runs bounded non-streaming, streaming
and tool-call completions, then `SIGTERM`s the process group and confirms the
port is released. It exits non-zero if any check fails.

> **Host memory.** The model is ~29 GB of 8-bit weights. The smoke cannot run
> while the Python `mlx_lm.server` runtime for the same model is resident — a
> 64 GiB host cannot hold both. Check `agents-infra runtime status` first.

## Dead-generation-worker smoke

```bash
BINARY=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
HARNESS=/Users/alexis/.local/bin/model-harness \
MODEL=/Users/alexis/.cache/huggingface/hub/models--mlx-community--Qwen1.5-0.5B-Chat-4bit/snapshots/659d8dafc39202a6688bb46242d60440702489b1 \
PORT=18019 OUT=./dead-generation-out \
scripts/dead-generation-smoke.sh
```

Drives the whole path on the real Release binary under the real
`model-harness`, in four phases:

| Phase | What it establishes |
| --- | --- |
| control | The same build, no injected fault: `/health` `200` before and after a real generation, and no marker in its output. |
| fault | **Unsupervised.** `/health` `200` → completion `500` → `/health` `503 unavailable`, still `503` on a later poll, `/v1/models` `503` with the model ID absent, later completions refused `503`. |
| supervision | The same fault **with** the policy attached: `model-harness` names the marker, restarts, and the replacement answers `200`. |
| negative | A request-scoped failure returns `500` and leaves `/health` at `200`, the model advertised, no marker, and no restart. |

The fault and supervision phases are separate runs on purpose. With a policy
attached, the harness kills the runtime within milliseconds of the marker
reaching its stdout — correct behaviour, and it destroys the very `503` window
the fault phase has to observe. Measuring both in one process would only
measure which won the race.

The negative phase is what makes the rest mean anything: a runtime that
condemned itself on any error would pass the first three phases.

Uses a small model on purpose — the failure path never reaches the weights, so
paying 29 GB to observe it would only mean the check cannot run while the
default Python runtime is resident. Runs in about two minutes and needs no GPU
memory beyond the 0.5B model.

## Metal shader library gate probe

```bash
BINARY=$PWD/.build/release/mlx-swift-runtime-prototype \
PORT=28117 OUT=./metallib-gate-out \
scripts/metallib-gate-probe.sh
```

Attacks the startup gate at the composed entry point rather than through a
helper. It deliberately wants a **`swift build`** product — the build that
genuinely lacks the shader library, which is what makes the gate observable —
copies it into a staging directory, plants a directory and then a dangling
symlink where `default.metallib` belongs, and requires `serve` to exit 2 with no
listener ever bound. The last case plants a real file and requires the gate to
*admit*, so a gate that refused everything could not pass the probe.

Runs in seconds and loads no weights.

## Runtime benchmark and migration decision

**One invocation launches both runtimes, drives every scenario against them,
records what it observed and judges the pair.** Not a driver plus a gate — one
process, because every seam between the two turned out to be a place where a
caller could hand the gate a document about work nobody did.

```bash
GATE=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype

"$GATE" benchmark-run \
    --config /absolute/path/to/model-harness.benchmark.toml \
    --model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
    --prompts examples/benchmark-prompts.json \
    --thresholds examples/benchmark-thresholds.json \
    --session ./benchmark-session \
    --harness /Users/alexis/.local/bin/model-harness \
    --baseline-runtime python-mlx-lm --baseline-profile qwen-benchmark-python \
    --candidate-runtime mlx-swift    --candidate-profile qwen-benchmark-swift \
    --port 18031 \
    --python-bin /Users/alexis/.local/pipx/venvs/mlx-lm-relux/bin/python \
    --candidate-binary "$GATE" \
    --baseline-declare "deployed with --prompt-cache-size 1 …" \
    --candidate-declare "no prompt cache across requests …"
```

It writes `records/<runtime>.json`, `attest/<runtime>.attestation.json`,
`logs/<runtime>-runtime.log`, `session.json` and `decision.json` under
`--session`, and exits `0` accepted, `3` rejected, `4` inadmissible, `2` usage,
`5` aborted before a pass could produce anything.

The two passes are sequential, never concurrent — this host holds ~35 GiB free
and one copy of this model is 28 GB, so overlapping runs would be measuring each
other's memory pressure. Both are launched through `model-harness run`, so
ownership, readiness polling and shutdown are the production ones.

### Why it is one invocation

Three review rounds found the same defect in three different constructions, and
each fix raised the cost of the forgery without removing it:

| Revision | The gate checked | What review did |
| --- | --- | --- |
| 1 | each record against itself | minted a self-consistent pair; `accepted=true`, exit 0 |
| 2 | records against provenance the driver recorded off its own launch | minted both; `accepted=true`, exit 0 |
| 3 | passes it observed itself, through `benchmark-attest open`/`close` | started two placeholder HTTP servers that answered `GET /v1/models` and nothing else, had the **production** commands attest them — correctly, the processes were real — typed the measurements, and got `accepted=true` in 7.2 s |

Round 3 used no forgery at all. It used the shipped commands exactly as
documented. The gate proved two processes stayed alive and that some endpoint
named the model; it proved nothing about what ran or what was measured.

So revision 4 removes the seam:

- **`benchmark-attest` no longer exists.** There is no way to ask this binary to
  observe a process the caller chose. An attestation is produced only as a
  by-product of `benchmark-run`, about a process `benchmark-run` spawned.
- **Measurements travel with the exchanges they came from.** Every
  `ScenarioResult` carries a `transcript`: for each request, the path, the
  digest and byte count of the request body, the HTTP status, the digest and
  byte count of the response, and the instants it was sent and finished.
- **The observation seals the record.** `transcriptDigest` is
  `RuntimeBenchmark.transcriptDigest(of:)` over the record the same invocation
  just built — covering the measurements as well as the exchanges, because
  sealing the exchanges alone would leave the reported TTFT free to be anything.
- **`benchmark-compare` can no longer return an acceptance.** It replays an
  archived session — same admission, same scoring — and exits `3` when it
  admits the pair and `4` when it does not. Never `0`. Files on disk can be
  authored; a command that turned authored files into `accepted=true, exit 0` is
  precisely the bypass found three times.

**What this still does not prove.** A modified build of the gate can report
anything; nothing a program says about itself survives the program being
changed. What changed is the *class*: all three findings were ordinary use of
shipped commands, and that is closed. Producing a false acceptance now requires
editing and rebuilding the gate.

That last sentence was measured rather than left as a disclaimer. Three limit
probes replaced the scenario driver with fabricated results while leaving the
launch, the observation and the seal genuine:

| Probe | Fabrication | Outcome |
| --- | --- | --- |
| L-1 | plausible measurements, **no** transcript | refused — admission requires every scenario to carry the exchanges it came from |
| L-2 | measurements **with** a synthetic transcript, 0.01 s per scenario | refused — `record claims 0.03s of scenario wall clock, but the gate watched the runtime for only 0.0159s` |
| L-3 | the same, scaled to 0.0005 s per scenario | **accepted, exit 0** |

So the boundary is exact, and it is narrower than "requires editing the gate"
suggests: what stands between a fabrication and an acceptance is **arithmetic
consistency with an observed interval**, not any ability to tell a measured
number from a typed one. The ordinary-caller class is closed; the modified-build
class is not, and no attempt was made to close it — every additional clause
would raise the cost of the same attack without changing its class, which is
what the three rounds above already demonstrated.

### What one invocation observes

Per pass, all of it read first-hand about a process it spawned:

| Observation | Source |
| --- | --- |
| the runtime pid | the child of the `model-harness` process this invocation spawned, resolved from `sysctl KERN_PROC_ALL` |
| that pid's executable path | `proc_pidpath`, the kernel |
| that pid's process start time | `sysctl KERN_PROC_PID`, the kernel; re-read before close, so a recycled pid is refused |
| the SHA-256 of the executable actually running | the file the kernel named |
| the SHA-256 of the launcher config | the file on disk |
| the model ID the runtime is serving | a live `GET /v1/models` before teardown |
| every request and response | performed by this process, timed by its own clock |
| the runtime's physical footprint | `proc_pid_rusage` on that pid, sampled every 0.25 s |
| when the pass began and ended | the gate's own clock |
| which build made these observations | the gate binary's own SHA-256 |

`examples/model-harness.benchmark.toml` carries both profiles; the Swift
`executable` must be the **`xcodebuild`** Release product, because a
`swift build` binary has no Metal shader library and refuses to serve.
`examples/benchmark-prompts.json` is the pinned prompt suite and its SHA-256 is
recorded in both run records, so a comparison across two different suites is
refused rather than reported.

### What the gate refuses

Both entry points apply the same admission. It refuses:

| Condition | Why |
| --- | --- |
| no scenario in a record ever completed a chat completion | the process answered other endpoints and served nothing; there is no benchmark to judge. **This is review's round-3 reproduction** |
| a scenario claims success and its transcript holds no served completion | a 200 on some other path, or a 200 with an empty body, is not a served completion |
| a required scenario carries no transcript | a measurement that does not carry the requests it came from is a number, not an observation |
| the record does not digest to what the observation sealed | these measurements are not the ones the gate watched being taken |
| the observation seals no transcript | it witnesses that a process existed, which is not a witness to what was measured |
| an exchange sits outside the window it was observed in | the work reported is not the work watched |
| any of the ten pinned fields differs | host, model path, model digest, quantization, prompt-suite digest, context policy, scored output bound, temperature, top-p, seed. The refusal names the field |
| both records name the same runtime | a runtime compared against itself cannot decide a migration |
| a record carries no `provenance` block | it does not decode at all; a hand-minted document never reaches a comparison |
| a record declares no `revisions` | a record that cannot name the code that ran is not evidence about a runtime |
| the declared `contextPolicy` is not what the recorded launch argv derives | the pin would be the caller's claim rather than the run's condition |
| the derived policy leaves the prefill chunk unstated | `mlx_lm.server` defaults it to 2048 and `MLXLMCommon` to 512, so an unstated condition is an unpinned one |
| the derived policy leaves the reasoning effort unstated | this model's chat template defaults `reasoning_effort` to `xhigh` and injects an extra system instruction at that setting. A profile that says nothing renders a *different prompt*: 79 tokens against 41 for the same short messages, a constant +38 on every prompt in the suite. Read from `--reasoning-effort` here and from `--chat-template-args` for `mlx_lm.server` |
| no attestation exists for a record's runtime | nobody observed that pass, so the record is the only witness to itself |
| an attestation exists and cannot be read or decoded | reported as a read failure, which is a different fact from absence and never collapsed into it |
| the attestation was opened and never closed | the gate saw the pass begin and never saw it end |
| the record's pid, executable digest, config path/digest or profile differs from what the gate observed | the record describes a run the gate did not watch |
| the runtime the gate asked was serving another model | a record can name any model; this is the one that answered |
| the observation window does not sit inside the interval the record claims | the certificate is for some other stretch of time |
| the record's scenario wall clock exceeds the watched window | those scenarios did not happen under observation |
| both attestations observed one pid started at the same instant | two runtimes cannot be one process. Pid *and* start time, because sequential passes may legitimately reuse a pid |
| the binary judging the comparison is not the binary that wrote the attestations | a build that did not watch these runs cannot certify that it was the one measured |
| the two records' launcher-config digests differ | both profiles live in one file; two runs configured by different documents are not a comparison |
| the two records share a launch-executable digest | two records naming different runtimes cannot have run the same binary |
| the rendered launch argv never mentions the pinned model path | the pin is not bound to the process that ran |
| the launcher command does not carry the recorded config path and profile | the recorded configuration is not the one the launcher was given |
| any recorded digest is not 64 lowercase hex | the record is not bound to the bytes it names |
| the two wall-clock intervals overlap | this host holds ~35 GiB free and one copy of this model is 28 GB, so overlapping runs were paging against each other |
| a record finished before it started | its interval cannot establish that the runs were sequential |
| a record's scenario wall clock exceeds the pass it claims to fit in | the scenarios it claims could not have run in the interval it claims |
| a required scenario is missing from either record | a scenario one runtime never ran is not a scenario the other one won |
| a record could not be read, or decoded | reported as `unreadable` / `malformed`, never as an empty record |

`scripts/benchmark-gate-smoke.sh` drives all of it through the real subcommands:

```bash
BINARY=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
    scripts/benchmark-gate-smoke.sh
```

**Its accepted control is not fabricated, and this time that claim is about the
measurements.** Round 3 was right that the previous version's was not: its two
attestations were real and every number beside them was invented by the script.
Here the control is one `benchmark-run` invocation that spawns both stand-in
runtimes through `model-harness`, drives every scenario, times and samples them,
and seals and judges what it measured — the same code path the 28 GB comparison
uses. The script supplies configuration and prompts; it cannot supply a
measurement, and four cases assert that no flag accepts one.

It then runs review's own reproduction as a production-entry negative: two
processes that answer `GET /v1/models` and refuse everything else, launched by
`benchmark-run` itself, must exit `4` with the refusal that says nothing was
served. Each remaining negative moves exactly one field of the **real** session
the control produced — one measurement, one pid, one digest, one attestation —
so the refusal it provokes is attributable to that thing.

And, having admitted the pair, it blocks the decision on:

- any scored metric that was **not measured** on either side — unknown is not
  within threshold;
- any scored scenario that did not succeed on **both** runtimes — the numbers a
  failed run leaves behind are the numbers it had when it died;
- a baseline value of zero, which has no ratio — reporting one would print `inf`
  and score it as a pass;
- a parity scenario the baseline won and the candidate lost. A parity scenario
  the **baseline** also lost is reported but is not held against the candidate;
- a scored scenario whose two runtimes rendered prompts more than
  `maxPromptTokenSkewRatio` apart. Its ratios are still reported and given no
  verdict: a faster number on a smaller prompt is not a faster runtime.

Peak memory is scored **per scenario**, from a scenario-local window, not from
the process's running maximum. The whole-process peak is reported against the
same bound only when every parity scenario succeeded on both runtimes;
otherwise the two maxima summarise different completed work and the axis is
reported as `non-comparable`. Review caught the previous shape: a candidate that
*aborted* the 75k probe posted a lower whole-pass maximum than the baseline that
completed it, and the gate scored that 1.094x ratio as "within".

### Measurement choices worth knowing

- **Size comes from the Mach physical footprint**, never from `ps` RSS. Three
  identical loads of this model reported 2 650 / 10 774 / 14 056 MiB resident
  while the physical footprint stayed within 16 MiB and MLX's active-bytes
  figure was byte-identical.
- **Readiness is "the pinned model answered a completion"**, not "`/v1/models`
  returned 200". `mlx_lm.server` answers `/v1/models` about a second after
  launch with no weights resident, and lists every MLX model in the local
  Hugging Face cache with the configured one appended **last** — so a poller
  that takes `data[0].id` gets a different model entirely.
- **Time to first token counts reasoning.** This model's chat template opens a
  `<think>` block, so the first thing generated for any prompt is reasoning.
  Waiting for the first *content* delta would report the length of the model's
  thinking as runtime latency.
- **Sampler settings are sent explicitly** on every request. The two runtimes
  disagree on defaults — `GenerateParameters` starts at `temperature = 0.6`,
  `mlx_lm.server` at `1.0` — so omitting them would compare two samplers.
- **The prefill chunk is pinned by value on both sides.** `mlx_lm.server`
  defaults `--prefill-step-size` to 2048 and `MLXLMCommon.GenerateParameters`
  defaults `prefillStepSize` to 512. Comparing the two as shipped measures a 4x
  difference in chunk size and reports it as a difference between runtimes, so
  `--prefill-step-size` is stated explicitly in both profiles and the gate
  refuses a pair that left it out.
- **The reasoning effort is pinned by value on both sides.** This model's chat
  template defaults `reasoning_effort` to `xhigh` and injects
  `"Reasoning effort is set to xhigh. Please think carefully…"` into the system
  turn at that setting; `medium` injects nothing. So a Swift profile passing
  `--reasoning-effort medium` against a Python profile passing no
  `--chat-template-args` is not two runtimes on one prompt — it is two prompts,
  79 tokens against 41 on the short scenario and a constant +38 on every prompt
  in the suite. Revision 2 of the migration decision reported that as a 1.93x
  runtime skew. Both profiles now state `medium` and the gate refuses a pair
  that left it to the template.
- **The text-only factory is the one under test**, and it is the runtime's
  default. See below.

### Which MLX Swift factory serves this model

Both `MLXLLM.LLMModelFactory` and `MLXVLM.VLMModelFactory` register
`model_type` `qwen3_5` at the pinned `mlx-swift-lm bd4b743`, and the two build
different prompt-evaluation strategies:

| Factory | Model | Prompt evaluation |
| --- | --- | --- |
| `MLXLLM.LLMModelFactory` | `MLXLLM.Qwen35Model` | chunked, `windowSize ?? 512`, via the `LLMModel` extension |
| `MLXVLM.VLMModelFactory` | `MLXVLM.Qwen35VLModel` | one call; `prepare(_:cache:windowSize:)` declares `windowSize _: Int?` and discards it |

`--model-factory` selects the order, and **`text-only` is the default**:

| Value | Order |
| --- | --- |
| `text-only` | `MLXLLM.LLMModelFactory`, then `MLXVLM.VLMModelFactory` |
| `text-only-strict` | `MLXLLM.LLMModelFactory` only — no fallback |
| `vision-first` | `MLXVLM.VLMModelFactory`, then `MLXLLM.LLMModelFactory` |

This executable serves a text-only surface — `ChatCompletionRequest` refuses
image and audio content parts — so the vision tower is weight it loads and can
never reach, and `MLXLLM.Qwen35Model.sanitize` drops those weights outright.
Preferring the vision factory meant paying for a capability the HTTP contract
rejects and losing chunked prefill to buy it. The `ready` event publishes both
the requested `factory_preference` and the `factory` that actually built the
model, so a fallback is visible rather than assumed.

## Known limitations

See the task's gap list for the full record. In short: one generation at a time,
no prompt caching across requests, no `/v1/completions`, no embeddings, and no
speculative decoding.

The one supervision marker that exists is `generation_worker_unavailable`
(above). Nothing else in this runtime announces itself as fatal, so every other
failure mode still reaches an operator as an exit status rather than as a
supervised restart.
