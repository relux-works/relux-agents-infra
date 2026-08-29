# TASK-260828-3fgca3 — revision 4 (F5): the prompt suite is validated before anything is launched

## The finding, restated

A required `context_75k` scenario carrying `"prefix_repeats": "2027"` — the
repeat count as a JSON **string** — ran a **15-token** prompt instead of 16,232,
and the shipped `benchmark-run` exited **0** with `accepted: true`.

The reader was

```swift
if let repeats = spec["prefix_repeats"] as? Int { ... }
```

`as? Int` is `nil` for an `NSString`, and the old code could not distinguish
that from a field nobody wrote, so it took the absence branch and dropped the
prefix. Both passes launched honestly, both processes were observed from the
kernel, both records sealed over transcripts that recorded their requests
faithfully — the requests were simply the wrong ones. **Sealing is orthogonal to
this class:** the gate seals and scores the wrong request perfectly.

## What was done

`Sources/MLXSwiftRuntimeContract/PromptSuiteSchema.swift` (new). In the contract
library, because the executable target is not unit-testable and this is exactly
the seam a smoke alone would have to carry.

1. **Complete validation before either runtime is launched.** The suite is
   decoded with `JSONDecoder` into `JSONValue`, which separates
   `.string("2027")` from `.int(2027)` from `.bool(true)` from `.double(20.5)`
   exactly — `JSONSerialization` bridges JSON booleans *and* numbers to
   `NSNumber`, which is part of the original defect. Every scenario in the
   document, and every field of every scenario, is checked.
   `BenchmarkScenarios.Suite.init(path:)` runs it, and
   `BenchmarkRunCommand.execute` calls that before the session directory is
   created, before the equivalence verdict is read, and before the first
   `model-harness` spawn.
2. **Absent and present-malformed are two different facts**, the same
   distinction `RuntimeContextWindow` makes for `meta.n_ctx`. A wrong type, an
   out-of-range count, a field belonging to another scenario kind and an
   unrecognised field name all refuse; the refusal quotes the value, so `"2027"`
   and `2027` are distinguishable in the message.
3. **The readers are gone, not hardened.** The drivers take typed
   `PromptSuiteSchema.Scenario` values. There is **no `as?` cast left in
   `BenchmarkScenarios.swift`**. `BenchmarkScenarios.run` also stopped being
   optional-returning: the old signature returned `nil` for an unrecognised
   `kind` and the loop `continue`d, so a misspelled kind silently removed a
   required scenario from a pass.

### The audit, over the surface revision 3's audit missed

| reader | old behaviour | now |
| --- | --- | --- |
| `single.prompt` | `?? ""` — an empty user message, measured and scored | required, non-empty |
| `single.prefix_repeats` | **the finding** | optional; present-and-unusable refuses |
| `max_tokens` (all four kinds) | `?? 256` — a suite asking for 16 got 256 | optional; present-and-unusable refuses |
| `tool.prompt` | `?? ""` | required, non-empty |
| `tool.tools` / `function` / `name` | a scenario-time failure, an hour into the run | validated before launch |
| `tool.parameters.required` | `?? []` — a parity check demanding nothing back | optional; present-and-unusable refuses |
| `multiturn.prefix_repeats` | `?? 0` | optional; present-and-unusable refuses |
| `multiturn.turns` | `?? []` — **zero requests, then `succeeded` with no exchanges** | required, non-empty, each entry non-empty |
| `soak.iterations` | `?? 0` — **zero requests, then `succeeded`** | required, ≥ 1 |
| `soak.prompt_template` | `?? ""` | required, non-empty, must contain `{index}` |
| `kind` | absent/unrecognised: skipped in silence | required, one of four |
| `filler_paragraph` | present-and-empty multiplies to nothing, hollowing out every prefix count with no malformed field anywhere | required, non-empty |
| any unknown key, any level | ignored | refused |
| a field of another kind | ignored | refused |

### The absences that are intentionally supported

Listed by name in `PromptSuiteSchema.supportedAbsences` and asserted whole by a
test, the treatment `RuntimeBenchmark.unpinnableConditions` gets, so adding or
removing one happens in the open:

| field | what an absence means |
| --- | --- |
| `version`, `comment` | documentation. The **values** are deliberately not typed — nothing is read out of them and the shipped suite writes `"version": 1` as a number — while the **keys** are recognised, which is what catches a misspelled field name |
| `<scenario>.max_tokens` | the driver's default, which is the number pinned into both records as `maxOutputTokens` |
| `single.prefix_repeats` | no filler prefix. This is `short_prompt`'s shape in the shipped suite |
| `multiturn.prefix_repeats` | no shared prefix before the first turn |
| `tool.parameters.required` | the tool declares no required arguments |

### Two deliberate non-changes

- **`unpinnableConditions` was not touched.** Still exactly
  `["kv=unread", "prefill-step=unpinned", "reasoning=unpinned"]`.
- **A required scenario removed by `--skip` is still not refused before launch.**
  It would be cheap, but `benchmark-gate-smoke.sh` uses `--skip tool_call` as the
  *only* case that drives admission's `requiredScenarios` call site, and a
  pre-launch refusal would delete that evidence. The pair is still refused one
  layer later, exit 4.

## The correction to revision 3

Revision 3's logbook entry and report claimed *"every reading in the invocation
was walked"*. **That claim did not hold.** It walked the readings the gate
performs against a **runtime** — attestations, `/v1/models`, `/slots`, argv,
`sysctl`, subprocess output — and never walked the readings it performs against
its own **input files**. The correction is written in place, in both:

- `LOGBOOK.md`, the 0130 entry, a `CORRECTION` bullet above the `AUDIT` bullet.
- `.research/260828_llamacpp-in-the-benchmark-gate.md` §0R.4, a blockquote at the
  head of the section.

## Evidence

### Negative evidence at the production entry

`scripts/benchmark-gate-smoke.sh` section 5. Each case drives the shipped
`benchmark-run` with a malformed suite and asserts **three** things, because a
refusal that arrives after an hour of model loading is not the fix:

- a nonzero exit,
- **no decision emitted** — not a rejection, not an inadmissibility, nothing,
- **no session directory**, which is created after the suite is read and before
  the first launch, so its absence is evidence that no runtime was started.

| case | result |
| --- | --- |
| `"prefix_repeats": "2027"` on a required scenario | exit 5, no decision, no session directory |
| `prefix_repeat` (a misspelled field name) | exit 5, no decision, no session directory |
| `"iterations": 0` on the soak | exit 5, no decision, no session directory |
| `"filler_paragraph": ""` | exit 5, no decision, no session directory |
| `"kind": "singel"` | exit 5, no decision, no session directory |
| **the shipped `examples/benchmark-prompts.json`** | gets **past** validation and stops on the next input instead — this is what stops the rule being satisfied by refusing everything |

### Mutants: 3 applied, built, run, reverted. 3 killed, 0 survivors.

| Mutant | What it does | `swift test` | smoke |
| --- | --- | --- | --- |
| **M-F5-1** | `positiveInt` returns `nil` without a fault for a non-integer, so present-but-unusable collapses into absence again | exit 1, **15 red** | exit 1 — **the string `"2027"` is measured as a 15-token prompt and the pair is accepted, exit 0** |
| **M-F5-2** *(narrowing)* | `single.prefix_repeats` becomes required | exit 1, 3 red | exit 1 — the control's own `short_prompt` is refused, **exit 5** |
| **M-F5-3** *(narrowing)* | `version`/`comment` stop being recognised keys, so the unknown-key rule over-reaches | exit 1, 9 red | exit 1 — the control suite is refused, **exit 5** |

M-F5-1 is the acceptance question, answered at the production entry: the finding
itself, restored in one line, and the shipped subcommand returns
`accepted: true` for a hollow capacity scenario.

M-F5-2 and M-F5-3 are the narrowing pair and each refuses a suite this
repository actually ships, so the admitted class is pinned in both directions. A
delete-only mutant would say the validation exists; these say what it admits.

**There is no bypass mutant to write for the call site, and that is the point of
the refactor.** The previous shape had a check the driver could stop calling.
There is no unvalidated suite the drivers can accept now: they take
`PromptSuiteSchema.Scenario` values, which only the validator produces. The call
site's reachability is proven instead by the session-directory assertion.

**M11 and M12 re-run against this source** for the G4 bound the story carries:
**18 red** and **16 red**, unchanged from revision 3. This delta touches neither
`Pins.firstMismatch` nor `RuntimeSpeculation`.

### Gates, each run directly as its own process

| Gate | Command | Result |
| --- | --- | --- |
| package build | `swift build --build-tests` | exit **0** |
| contract suite | `swift test` | **379 tests / 30 suites**, exit **0** |
| production-entry smoke | `scripts/benchmark-gate-smoke.sh` against the Release product | **84 checks, 0 failures**, exit **0** |
| Release product | `swift build -c release` | exit **0** |
| Swift lint | `xcrun swift-format lint --strict --recursive Sources Tests` | exit **0** |
| shell lint | `shellcheck scripts/benchmark-gate-smoke.sh` | exit **0** |
| whitespace | `git diff --check` | exit **0** |

Toolchain: Apple Swift 6.3.3.

**Not run, and why.** Revision 3's real 29 GB cross-format pair through
`benchmark-run` is not repeated: it is an hour of exclusive host time, and the
only part of it this delta could have broken — the shipped suite passing
validation — is asserted directly at the production entry (smoke §5). No model
was loaded by anything in this revision.

## Files

| File | Change |
| --- | --- |
| `Sources/MLXSwiftRuntimeContract/PromptSuiteSchema.swift` | **new.** The whole schema: `JSONValue` decoding, the typed `Suite`/`Scenario`/`Body` model, `supportedAbsences`, every fault collected rather than the first thrown |
| `Sources/mlx-swift-runtime-prototype/BenchmarkScenarios.swift` | `Suite` wraps the validated document; all four drivers take typed values; no `as?` cast remains; `run` no longer optional-returning |
| `Sources/mlx-swift-runtime-prototype/BenchmarkRunCommand.swift` | the scenario loop no longer reads `kind` or skips on a failed cast |
| `Tests/MLXSwiftRuntimeContractTests/PromptSuiteSchemaTests.swift` | **new.** 28 tests, mostly negatives, each a document the old readers would have accepted and quietly mismeasured |
| `scripts/benchmark-gate-smoke.sh` | section 5, six production-entry cases |
| `README.md` | the suite-validation contract and the supported-absence table |
| `LOGBOOK.md`, `.research/260828_llamacpp-in-the-benchmark-gate.md` | revision 4, and the correction to revision 3's audit claim |
