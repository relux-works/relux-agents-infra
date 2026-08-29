# llama.cpp as a benchmark-gate candidate

Task: TASK-260828-3fgca3 (story STORY-260828-2faxgm)
Date: 2026-08-28
Author role: developer
Built against: `main` at `fb85963`, merged into this story branch. The gate is
the single-invocation construction landed by STORY-260827-m30k8z; the
caller-authored record shape does not exist here and was not reintroduced.

## Verdict, in four parts

1. **G2 is closed, and the fix is that the gate asks the process.** The KV
   bound is no longer derived from argv at all for a runtime that reports one.
   `llama-server` reports `n_ctx` on `GET /v1/models`, measured, so its pin is
   that number and can never be `unbounded` ([§2](#2-g2--the-kv-bound-comes-from-the-running-process)).
2. **G1 is closed additively.** `-ub` / `--ubatch-size` are read as the second
   and third spellings of the prefill pin, exactly as `--chat-template-args`
   was added as a second spelling of reasoning effort.
   `unpinnableConditions` was **not** relaxed; it grew by one entry
   ([§3](#3-g1--a-third-spelling-of-the-prefill-pin)).
3. **The acceptance question is answered by a killed mutant, at the production
   entry.** With the pre-fix derivation restored, `benchmark-run` returns
   **exit 0, `accepted=true`** for a candidate running a 32,768-token context
   window compared against a genuinely unbounded baseline. With the fix, the
   same two processes are refused **exit 4** naming both readings
   ([§4](#4-mutants)).
4. **A third gap, not in this task's brief, blocks the real llama.cpp pass and
   is not patched around.** `--model` must be an MLX weight *directory* and the
   `modelPath`/`modelDigest` pins demand both runtimes name the same one; a
   GGUF candidate names a different file. Two production-entry refusals, exit
   5 each, are recorded as **G4** with the decision needed
   ([§5](#5-g4--the-model-pin-has-no-cross-format-meaning-not-in-this-brief)).

## 1. Host and fixtures

Nothing else held this host's memory: `ps` showed no `llama-server`,
`mlx_lm`, `model-harness` or `mlx-swift` process before or after, and no
process belonging to another run was signalled. The **28 GB model was never
loaded**. Every llama.cpp measurement below uses the 676 MB
`Qwen2.5-0.5B-Instruct` Q8_0 fixture staged by TASK-260828-2jbufw, on an
OS-assigned ephemeral port outside `18000-18999`. The G4 probes in §5 are
refused *before* any launch, so they loaded nothing either.

Runtime under test: pinned Homebrew `llama.cpp 0.3.0`, build **10621**, commit
`c1d0e7a00`, reported by the server itself as `build_info: b10621-c1d0e7a00`.

## 2. G2 — the KV bound comes from the running process

### 2.1 What llama.cpp reports, measured

`GET /v1/models` carries a `meta` block the MLX runtimes do not emit at all:

| Launch | `meta.n_ctx` | `meta.n_ctx_train` | `/props` `default_generation_settings.n_ctx` |
| --- | ---: | ---: | ---: |
| `--ctx-size 8192` | **8192** | 32768 | 8192 |
| no context flag | **32768** | 32768 | 32768 |

Two readings of the same fact, from two endpoints, agreeing. There is no
launch of this runtime that is unbounded.

`ModelsListing.make` — the Swift prototype's own `/v1/models` — emits
`id`/`object`/`created`/`owned_by` and no `meta`. `mlx_lm.server` likewise. So
the two MLX runtimes answer the question and name no bound, which is a
different fact from not being asked.

### 2.2 The decision, and why this one

The brief left the choice open: make the derivation runtime-aware, or read the
bound from the running server. **The bound is read from the running server.**

Runtime-awareness was rejected because it has nowhere trustworthy to get the
runtime's identity. The only anchors in a record are `record.runtime` — a
string the record declares — and the launch executable's path. Keying the KV
semantics off either turns "which runtime is this" into a claim, and the pin
exists precisely so that a declared condition is a *reading*. Worse, it does
not remove the false premise; it forks it into a per-runtime table that the
next runtime has to be added to, correctly, or the same class of defect
returns silently.

Reading the process removes the premise instead of maintaining it. It also
fits the construction that landed: the gate binary already performs a
`GET /v1/models` against the process it spawned, inside its own observation
window, at close, and already refuses a pass whose answer it could not read.
The bound rides on that exact exchange.

### 2.3 What was implemented

`RuntimeContextWindow` (`Sources/MLXSwiftRuntimeContract/RuntimeAttestation.swift`)
is the gate's first-hand reading, with three cases kept apart because
collapsing any two of them is the same defect in another shape:

| Case | Means | KV term |
| --- | --- | --- |
| `.reported(n)` | the runtime answered and named its bound | `kv=n` |
| `.notReported` | the runtime answered and named none | `--max-kv-size` value, else `unbounded` |
| `.unread` | the gate got no answer at all | `kv=unread`, refused |

It is a non-optional field of `RuntimeAttestation`, so it lives in the document
the *gate* authored, not the one the record authored, and
`admitProvenance(_:observing:)` re-derives the pin from it. A record still
cannot declare a policy by writing a string.

`BenchmarkRunCommand.servingAnswer` produces the window and `servedModelID`
from **one** exchange: a failed read yields `nil` model *and* `.unread`, never
`.notReported`. `Pins` moved below that call, because the KV bound is the
runtime's answer and not the launch's.

Three further consequences, each deliberate:

* **The value, not the spelling.** `kv=max-kv-size=4096` became `kv=4096`. A
  llama.cpp `n_ctx` of 8192 and an MLX `--max-kv-size 8192` are one reading of
  one condition; under the old rendering they could never compare equal and
  llama.cpp would have been uncomparable rather than wrong.
* **`kv=unread` joined `unpinnableConditions`.** The list grew; nothing was
  removed. A bound the gate failed to read is exactly as unusable as a prefill
  chunk left to a default.
* **A pinned bound the process did not honour is refused by name.**
  `AdmissionError.contextBoundNotHonoured`. Ask for `--ctx-size 8192`, run
  4096, and the pin still agrees on both sides — because the pin takes the
  process's number — so nothing above this clause can see it.

### 2.4 The residual, stated rather than implied

`.notReported` still falls back to argv, so a *bounded* runtime that answered
`/v1/models` and declined to say so would read as unbounded. That is not the
llama.cpp case — it always reports, measured in §2.1 — and closing it in
general would mean refusing `mlx_lm.server`, which reports nothing and is the
incumbent baseline. What *is* closed is the contradiction: a launch carrying
`--ctx-size` / `-c` whose server will not confirm the bound is refused, so a
llama.cpp launch cannot reach the argv fallback at all. `--max-kv-size` is
deliberately not symmetric there, because it belongs to runtimes measured to
report nothing.

## 3. G1 — a third spelling of the prefill pin

`value(of: "--prefill-step-size") ?? value(of: "--ubatch-size") ?? value(of: "-ub")`.
Additive, and `--batch-size` is deliberately **not** read: it is llama.cpp's
*logical* batch, default 2048, where the physical prompt-evaluation chunk is
`--ubatch-size`, default 512. Reading the first as the second would pin a
condition the launch never stated, at four times the value in effect.

`unpinnableConditions` was not relaxed. TASK-260828-2jbufw measured what the
relaxation admits — an unpinned mlx-swift launch (512) against an unpinned
`mlx_lm.server` one (2048), because all three runtimes derive the
byte-identical string — and the test
`doesNotRelaxTheUnpinnableConditions` asserts the list as a whole so a future
edit has to remove a clause in the open.

## 4. Mutants

Eight, each applied to the shipped source, built, run, and reverted. **All
eight killed.** `swift test` is the 302-test contract suite; the smoke is
`scripts/benchmark-gate-smoke.sh`, 46 checks driving the shipped subcommands.

| Mutant | What it does | `swift test` | smoke |
| --- | --- | --- | --- |
| **M1** | the pre-fix derivation: KV off argv, absence as unbounded | exit 1, 10 red | exit 1 — **the false match is accepted, exit 0** |
| **M2** | `.unread` spent as an absence | exit 1, 2 red | — |
| **M3** | *narrowing*: `kv=unbounded` declared unpinnable | exit 1, 12+ red | — |
| **M4** | *the relaxation this task was told not to make* | exit 1, 3 red | exit 1, 2 FAIL |
| **M5** | G1 undone: `-ub`/`--ubatch-size` unread | exit 1, 7 red | exit 1, 4 FAIL |
| **M6** | *narrowing*: every pinned bound must be confirmed | exit 1, 3 red | exit 1, 2 FAIL |
| **M7** | `--batch-size` read as the prompt-evaluation chunk | exit 1, 1 red | — |
| **M8** | **production call site**: the driver stops asking the runtime | **exit 0 — blind** | exit 1, 6 FAIL |

Three of these carry the argument.

**M1 is the acceptance question, and it is a production-entry answer.** With
the pre-fix derivation restored, the smoke's first KV case — two real spawned
processes, one reporting `n_ctx` 32768 and one reporting nothing, driven and
measured and judged by the shipped `benchmark-run` — comes out:

```
FAIL  a 32k context window cannot match an unbounded baseline: expected exit 4, got 0
```

Exit 0 is `accepted=true`. A 32,768-token window scored against no window at
all, green. With the fix the same two processes exit **4**, and the refusal
prints both readings rather than one derived string.

**M8 is the seam the unit suite cannot see.** Deleting the `meta.n_ctx` read
from `BenchmarkRunCommand.servingAnswer` leaves all 302 contract tests
**passing** — they hand the window in directly — and reddens six smoke checks,
including the same false acceptance. A window type that is unit-tested and
never populated from the wire would have promised nothing.

**M3 and M6 are the narrowing pair.** Both make the gate *stricter*: M3 adds
`kv=unbounded` to `unpinnableConditions`, M6 requires every pinned bound to be
confirmed by the process. Both redden the tests that pin the *admitted* class —
`a --max-kv-size launch whose runtime reports nothing is its normal case`,
`a llama.cpp candidate is admitted against a baseline pinned to the same bound`,
and the smoke's accepted pair. A delete-only mutant would have said the clause
exists; these say what it covers, in both directions.

## 5. G4 — the model pin has no cross-format meaning (not in this brief)

A real `llama-server` pass cannot be launched by `benchmark-run` today, and the
reason is not G1 or G2. Two production-entry refusals, both exit **5**, both
taken before anything was launched, so no weights were loaded:

```
$ benchmark-run --model .../Qwen3.8-27B-Uncensored-GGUF-Q8_0 ...
"…/Qwen3.8-27B-Uncensored-GGUF-Q8_0/config.json" could not be read; the model cannot be pinned

$ benchmark-run --model .../Qwen3.8-27B-Uncensored-MLX-8bit --baseline-profile llamacpp-candidate ...
profile "llamacpp-candidate" does not pass
"…/Qwen3.8-27B-Uncensored-MLX-8bit" to the runtime; the modelPath pin would
not be bound to the process under test
```

`BenchmarkRunPins.modelDigest` / `quantizationLabel` read `config.json` and
`model.safetensors.index.json` out of an MLX weight **directory**; the staged
GGUF directory has a `.gguf` file and a `PROVENANCE.md` and neither of those.
And `modelPath` and `modelDigest` are both pins compared for equality, so a
comparison in which one side serves `…/Qwen3.8-27B-Uncensored-MLX-8bit` and the
other serves `…-OrcaRouter-Q8_0.gguf` cannot satisfy them however the digest is
computed.

**This was not patched.** Making `--model` accept a file, or digesting the two
formats into one label, is not a derivation widening like G1 — it is a decision
about what "the same model" means when the two runtimes cannot read the same
weight file. TASK-260828-3g87i4 already established that GGUF Q8_0 and MLX
8-bit quantize the same numbers *and* that the MLX build lost the MTP head, so
the answer is not "they are the same weights" and it is not "they are different
models" either. Adding a flag, a format branch or a per-runtime digest rule
here would be exactly the compensating-hack shape the gate has been rebuilt
three times to remove.

**Decision needed:** what the `modelPath` and `modelDigest` pins mean for a
cross-format comparison — a shared logical model identity with per-runtime
weight digests recorded as a declared asymmetry, or a rule that
cross-format comparisons are out of scope for this gate. Owner: the story.
Nothing in this delta depends on the answer; the KV and prefill work is
complete and correct either way.

## 6. Gates

| Gate | Command | Result |
| --- | --- | --- |
| package build | `swift build --build-tests` | exit **0** |
| contract suite | `swift test` | **302 tests / 25 suites**, exit **0** |
| production-entry smoke | `scripts/benchmark-gate-smoke.sh` | **46 checks, 0 failures**, exit **0** |
| Release product | `xcodebuild build -configuration Release` | **BUILD SUCCEEDED**, exit **0** |
| Swift lint | `swift-format lint --strict --recursive Sources Tests` | exit **0** |
| shell lint | `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | exit **0** |
| whitespace | `git diff --check` | exit **0** |
| mutants | 8 applied, built, run, reverted | **8/8 killed**, 0 survivors |

Not run, and why: the real 28 GB llama.cpp-vs-MLX comparison. It is blocked by
G4 above, not by anything in this delta, and this task's acceptance is about
admissibility rather than about a migration decision.

## 7. Files

| File | Change |
| --- | --- |
| `Sources/MLXSwiftRuntimeContract/RuntimeAttestation.swift` | `RuntimeContextWindow`; `observedContextWindow` on the attestation |
| `Sources/MLXSwiftRuntimeContract/RuntimeBenchmark.swift` | `contextPolicy(derivedFrom:observing:)`; `declaredContextBound`; `contextBoundNotHonoured`; `kv=unread` added to `unpinnableConditions`; `admitProvenance(_:observing:)` |
| `Sources/mlx-swift-runtime-prototype/BenchmarkRunCommand.swift` | `servingAnswer` reads `meta.n_ctx` off the live runtime; pins built after it |
| `Tests/.../RuntimeBenchmarkContextBoundTests.swift` | new, 17 tests, mostly negative |
| `Tests/.../RuntimeBenchmarkTests.swift` | call sites updated; the MLX default window stated once |
| `scripts/benchmark-gate-smoke.sh` | section 3: 8 production-entry KV checks; the stand-in learned `--n-ctx` |
| `examples/model-harness.benchmark.toml` | `profiles.qwen-benchmark-llamacpp` |
| `README.md` | what the gate observes, what it refuses, why the bound is read off the process |
