# TASK-260827-qyebv8 — reviewer rework (revision 3)

Run `RUN-260827-a388ff`. Addresses every finding in
`TASK-260827-qyebv8_review-verdict.md` (revision 1, changes requested).

| Finding | Status |
| --- | --- |
| F1 out-of-range `max_tokens` aborts the runtime | Fixed; proven at the live ready HTTP entry and by 3 narrowing mutants |
| F2 the Metal-library gate accepts a directory as forged evidence | Fixed; proven at the composed executable entry point and by 3 narrowing mutants |
| F3 generated `default.profraw` in the candidate | Removed and ignored |
| F4 stale primary results artifact | Rewritten |

One hardening beyond the four findings is included and called out in §5.

---

## F1 — an out-of-range integral JSON number aborted the runtime

### The defect

`ChatCompletionRequest.optionalInt(_:field:)` accepted any integral `Double` and
converted it with `Int(_:)`. `Int(1e300)` is a Swift **fatal error**, not a
thrown one, so a single ordinary request body killed the managed runtime before
any refusal could be written to the socket. The reviewer's probe exited 133.

That is a bypass path, not a cosmetic one: small non-positive values reached the
`invalidMaxTokens` refusal while a larger invalid value killed the process that
was supposed to issue it.

### The fix

`Sources/MLXSwiftRuntimeContract/ChatCompletionRequest.swift`

```swift
case .some(.double(let value)) where value == value.rounded():
    guard let exact = Int(exactly: value) else {
        throw ChatCompletionDecodingError.numberOutOfRange(field: field, value: value)
    }
    return exact
```

`numberOutOfRange` is a new decoding error, distinct from `wrongType`, so an
unrepresentable value and a fractional one do not collapse into one message.

Production call site: `Router.chatCompletions(body:)` →
`ChatCompletionRequest.decode(from:)` → `optionalInt(_:field:)`. The router
already maps a decoding throw to `400 invalid_request_error / invalid_body`
(`Sources/mlx-swift-runtime-prototype/Router.swift:93-101`), so the fix reaches
the client as a bounded refusal with no further plumbing.

### Proven at the real entry point, on the real model

Three out-of-range literals were posted to the **live, ready** runtime serving
the 28 GB Qwen through `model-harness run`, each followed by a health probe,
because a bounded 400 from a process that then dies would prove nothing:

```
PASS  max_tokens 1e300 refused with 400
PASS  max_tokens 1e300 reports an out-of-range body, not a crash
PASS  runtime still healthy after max_tokens 1e300
PASS  max_tokens -1e300 refused with 400
PASS  max_tokens -1e300 reports an out-of-range body, not a crash
PASS  runtime still healthy after max_tokens -1e300
PASS  max_tokens 9223372036854775808 refused with 400
PASS  max_tokens 9223372036854775808 reports an out-of-range body, not a crash
PASS  runtime still healthy after max_tokens 9223372036854775808
PASS  max_tokens 0 still hits the ordinary bound
PASS  max_tokens 0 reports invalid_max_tokens, not an out-of-range body
```

The last two are the control: the gate must reject *unrepresentable* values, not
merely large ones, and the ordinary positive-integer rule must still own `0`.
These checks are now part of `scripts/smoke.sh`, so they run on every full-model
smoke rather than living in a one-off probe.

### Named tests

`Tests/MLXSwiftRuntimeContractTests/ChatCompletionAdmissionTests.swift`

- `refusesUnrepresentableMaxTokens` — `1e300`, `-1e300`, `1e19`, `-1e19`,
  `9223372036854775808`
- `acceptsRepresentableBoundary` — `Int.max` still decodes
- `acceptsIntegralDoubleMaxTokens` — `64.0`, `1e3` still decode
- `refusesFractionalMaxTokens` — `1.5` is a `wrongType`, not an out-of-range

---

## F2 — the Metal-library gate accepted a directory as forged evidence

### The defect

`MetalShaderLibraryCheck.inspect` used `FileManager.fileExists(atPath:)`, which
answers `true` for a directory. `mkdir default.metallib` at the exact expected
path was therefore classified `.present` and admitted; the reviewer drove the
composed executable and saw it bind port 28117 and emit `listening`.

This is the **forged / self-minted evidence** shape: the object the gate treats
as proof can be created by the thing being gated.

### The fix

New `Sources/MLXSwiftRuntimeContract/FileObjectProbe.swift`. It uses POSIX
`stat` directly rather than Foundation, because the distinctions that matter are
exactly the ones Foundation blurs:

- `stat` follows symlinks, so a symlink to a real library is admitted and a
  dangling one is not;
- its `errno` separates `ENOENT` ("nothing here", proven absence) from `EACCES`
  ("I was not allowed to look", never absence);
- `st_mode & S_IFMT` names what was actually found, so the refusal can say
  `is a directory` instead of `was not found`.

`inspect` gained a `libraryNotAFile(path:kind:)` outcome; `classify` carries it
into `absent(searched:rejected:)`; the error message appends *"An object occupies
the expected library path without being a library file: … Existence at that path
is not the same as a loadable library."*

An unreadable library path is still routed to `undetermined`, which does **not**
refuse — a failed read must not become a "your build is broken" verdict.

Production call site: `Main.main()` → `MetalShaderLibraryCheck.inspect` →
`classify` → `admit`, before the listener binds
(`Sources/mlx-swift-runtime-prototype/Main.swift:49-56`).

### Proven at the composed executable entry point

`scripts/metallib-gate-probe.sh` is new and reproducible. It copies a
`swift build` product — the build that genuinely lacks the shader library, which
is what makes the gate observable at all — into a staging directory, plants the
forged object beside it, and drives `serve` for real.

```
PASS  a directory at the library path exits 2 (got 2)
PASS  no listener was ever bound with the forged directory
PASS  the refusal names the forged object
PASS  no listening event with the forged directory
PASS  a dangling symlink at the library path exits 2 (got 2)
PASS  no listener was ever bound with the dangling symlink
PASS  a swift-build product with no bundle exits 2 (got 2)
PASS  no listener was ever bound without the bundle
PASS  a regular file at the library path passes the gate and binds
PASS  port 28117 is free at the end of the probe

METALLIB GATE PROBE OK (0 failures)
```

The fourth case is the control that keeps the other nine honest: a gate that
refuses unconditionally would satisfy every refusal above and prove nothing
about the object type it claims to check.

### Named tests

`Tests/MLXSwiftRuntimeContractTests/MetalShaderLibraryCheckTests.swift`

- `forgedLibraryDirectoryIsRefused` — a directory at the library path is
  refused, and the message names it
- `symlinkedLibrary` — dangling symlink refused, resolving symlink admitted
- `unreadableLibraryPathIsNotAbsence` — `EACCES` on the library path is
  `undetermined`, and does not refuse

### Real-path regression check

The hardened gate must not refuse the build that actually works. `preflight`
against the real model on the `xcodebuild` product exits 0 with
`"metal_shader_library": {"outcome": "passed"}`, and the full-model smoke loaded
and generated. The gate rejects non-files, not indirection.

---

## F3 — generated LLVM profile data

`tools/mlx-swift-runtime-prototype/default.profraw` (6 180 616 B) is deleted.
`.gitignore` now carries:

```
# LLVM raw profile output from instrumented local runs
*.profraw
*.profdata
```

`find . -name '*.profraw'` outside `.build/` and `DerivedData/` returns nothing.

---

## F4 — stale evidence handoff

`TASK-260827-qyebv8_results.md` was rewritten:

- the header states the current status and names the two artifacts that carry
  the detail, so there is one status statement rather than two contradictory
  ones;
- §2's gate table carries revision-3 commands, exit codes and counts, and states
  that `swift build` is a compile gate whose product cannot load a model;
- §5 "Blocked: full-model smokes" is replaced by "Full-model smokes — run and
  green" with the measured figures;
- §7 records the `DerivedData` exclusion alongside `.build`;
- `TASK-260827-qyebv8_blocker.md` is explicitly labelled historical.

The two review regressions and their fixes are recorded in `LOGBOOK.md`.

---

## 5. Hardening beyond the four findings

`ModelDirectoryCheck.observe` had the identical forged-evidence shape: it
matched required files by **name in a directory listing**, so `mkdir
config.json` — or a directory named `*.safetensors` — passed admission. It is
the same gate class the reviewer attacked, on the same startup path, so it was
closed in the same change rather than left for a third round.

Required entries and the weight file are now probed with the same
`FileObjects.probe`; a new `notRegularFiles(details:)` observation and matching
error name what was found. Per-entry `EACCES` now yields `unreadable`, which
throws its own error, rather than being laundered into `incomplete`.

Symlinked model trees — how a Hugging Face snapshot lays its files out — are
still admitted; `admitsSymlinkedFiles` covers that so the hardening cannot
silently break a real layout.

Named tests in `ModelDirectoryCheckTests.swift`: `refusesForgedRequiredFile`,
`refusesForgedWeights`, `admitsSymlinkedFiles`, `unreadableEntryIsNotMissing`.

---

## 6. Mutation evidence — six mutants, all caught

Each mutant **narrows** a gate in production code rather than deleting it, so a
delete-only "the gate exists" result is not what is being measured. Every mutant
was applied, the full suite run, then reverted.

| # | Mutation (production code) | Exit | Caught by |
| --- | --- | ---: | --- |
| M-A | `FileObjects.probe` returns `.regularFile` for every non-regular object (existence-only) | 1 | `forgedLibraryDirectoryIsRefused`, `symlinkedLibrary`, `refusesForgedRequiredFile`, `refusesForgedWeights` |
| M-B | first `stat` failure laundered into `.absent` | 1 | `unreadableEntryIsNotMissing` |
| M-B2 | *every* `stat` failure laundered into `.absent` | 1 | `unreadableLibraryPathIsNotAbsence`, `symlinkedLibrary`, `unreadableEntryIsNotMissing` |
| M-C | `S_IFDIR` classified as `.regularFile` (the exact reviewer defect) | 1 | `forgedLibraryDirectoryIsRefused`, `refusesForgedRequiredFile`, `refusesForgedWeights` |
| M-D | out-of-range `max_tokens` clamped instead of refused | 1 | `refusesUnrepresentableMaxTokens` (10 issues) |
| M-E | `Int(exactly:)` reverted to `Int(_:)` (the exact reviewer defect) | 1 | test **process** died with signal 5 on `-1e300`, reproducing the reviewer's exit 133 |
| M-F | only the positive side of the range gated | 1 | `refusesUnrepresentableMaxTokens` (negative cases) |

M-A and M-C were additionally driven through the **composed executable**:

```
# under M-A
FAIL  a directory at the library path exited 0, expected 2
FAIL  the forged directory bound a listener on 28117
FAIL  a listening event was emitted with the forged directory
FAIL  a dangling symlink at the library path exited 0, expected 2
METALLIB GATE PROBE FAILED (6 failures)

# under M-C
FAIL  a directory at the library path exited 0, expected 2
FAIL  the forged directory bound a listener on 28117
METALLIB GATE PROBE FAILED (4 failures)
```

That is the reviewer's exact observation reproduced on demand, and absent once
the gate is restored.

### A mutant that was not caught on the first attempt

M-B, which launders only the *first* `stat` failure into absence, did not fail
`unreadableLibraryPathIsNotAbsence`: in that fixture the `lstat` fallback still
returned `unreadable`, so the Metal test could not observe the change. It was
caught by the model-directory test instead, and M-B2 was then written to remove
both guards and confirm the Metal test does fail when the whole distinction
goes.

The same pass exposed a test that could not fail for its stated reason:
`unreadableEntryIsNotMissing` originally chmod'ed the model directory to `0o000`,
which makes `contentsOfDirectory` fail first, so the per-entry probe was never
reached. It was rewritten to use a symlink into a `0o000` subdirectory, keeping
the directory listable while stat'ing the entry fails — and it asserts the
listing is still readable, so the precondition itself is checked. Reported
because a test that cannot fail is not evidence.

---

## 7. Gates (real exit codes, standalone processes, no pipes)

| Command | Exit |
| --- | ---: |
| `swift build -c release` | 0 |
| `swift test -c release` | 0 — 103 tests / 9 suites (was 92) |
| `swift format lint --recursive --strict Sources Tests Package.swift` | 0 |
| `xcodebuild build … -skipPackagePluginValidation -skipMacroValidation` | 0 — `** BUILD SUCCEEDED **`, 0 error lines |
| `bash -n scripts/smoke.sh scripts/lifecycle-smoke.sh scripts/metallib-gate-probe.sh` | 0 |
| `mlx-swift-runtime-prototype preflight --model <model>` | 0 |
| `scripts/smoke.sh` (real 28 GB model via `model-harness run`) | 0 — SMOKE OK, 36 PASS / 0 FAIL |
| `scripts/lifecycle-smoke.sh` | 0 — LIFECYCLE SMOKE OK |
| `scripts/metallib-gate-probe.sh` | 0 — 10 PASS / 0 FAIL |
| `go build ./...` | 0 |
| `go vet ./internal/infra/...` | 0 |
| `gofmt -l internal/infra/infra.go internal/infra/source_sync_build_artifacts_test.go` | 0 — no output |
| `go test ./internal/infra/... -count=1` | 0 — 124.040s |

Full-model smoke, revision 3: load 6.738 s, physical footprint 29 633 484 064 B
(28 260.7 MiB), first 503 at 2 s, ready at 8 s, streaming 35 frames / 34 chunks,
`finish_reason=tool_calls` with a well-formed payload, SIGTERM exit 143 in 1 s,
port released, `stopped` event emitted.

---

## 8. Unchanged

Python `mlx-lm` remains the default local runtime. `model-harness.toml` and
`project-config.toml` are untouched, the prototype profile lives only in
`.temp/TASK-260827-qyebv8/` and is passed with `--config`, and no binary is
installed on `PATH`. Nothing is committed: `version_control.confirm = true`.
