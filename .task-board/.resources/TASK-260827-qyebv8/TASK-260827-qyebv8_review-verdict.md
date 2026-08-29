# TASK-260827-qyebv8 review verdict

## Verdict

**Changes requested; route to `to-dev`.** Do not accept Change Request
`CR-TASK-260827-qyebv8-1` revision 1.

The main prototype and its attached real-model evidence are substantial, but two
admission paths can be defeated and the candidate contains a generated coverage
artifact. A green suite does not cover these negative shapes.

## Candidate reviewed

- Base OID: `3f313d9175f2ada9b9ab3320ab524c0918f9daac`
- Candidate tree OID: `dae7d453e49ba2d067f0ad6ad88bc0299c871ea8`
- Patch SHA-256: `5e7bd1417c3c37f6415bb1b43ff94cd6135f1eed03d13791e618be4142c1b16a`
- Repository delta: present, 44 changed paths
- `git diff --check`: exit 0

## Findings

### F1 — High: an out-of-range integral JSON number aborts the runtime instead of being refused

Production call site:

`Router.chatCompletions(body:)` → `ChatCompletionRequest.decode(from:)` →
`optionalInt(_:field:)` in
`Sources/MLXSwiftRuntimeContract/ChatCompletionRequest.swift:191`.

The decoder accepts an integral `Double` and calls `Int(value)` without checking
that the value is representable. A request containing `"max_tokens": 1e300`
therefore terminates the process with a Swift fatal error. The reviewer probe
compiled the exact candidate contract sources and exited `133`:

```text
Swift/arm64e-apple-macos.swiftinterface:39435: Fatal error: Double value cannot be converted to Int because the result would be greater than Int.max
PROBE_EXIT=133
```

This is a bypass path around the `invalidMaxTokens` refusal: ordinary small
non-positive values reach it, while a larger invalid numeric value kills the
managed runtime before any HTTP error can be returned.

Required rework:

- make numeric conversion total and throwing (`Int(exactly:)` or an explicit
  representable-range check), including negative out-of-range values;
- add named negative coverage for values outside `Int` and for integral doubles;
- prove the real ready HTTP entry returns a bounded `400` response and remains
  alive after the request; narrow the representable-range gate and require the
  named test to fail.

### F2 — High: the Metal-library gate accepts a directory as forged evidence

Production call site:

`Main.main()` → `MetalShaderLibraryCheck.inspect` → `classify` → `admit`, before
the listener bind.

`inspect` uses only `FileManager.fileExists(atPath:)` for
`.../default.metallib`; it does not require a regular file. A directory created
at that exact path is classified as `.present` and admitted.

The helper-level probe reported:

```text
OBSERVATION=present(path: ".../mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib")
ADMITTED_FAKE_LIBRARY_DIRECTORY
```

The same shape was then driven through the composed executable entry point. A
SwiftPM-built binary was copied beside a directory named `default.metallib` and
started against the zero-weight unsupported-architecture fixture. It emitted the
`listening` event on port 28117, proving the production startup gate admitted the
forged object; it was then SIGTERMed and exited 0.

This is the standard **forged or self-minted evidence** shape. The existing
`bundleWithoutLibraryIsAbsent` test proves only that an empty bundle is refused;
it does not prove the terminal object is a usable library file.

Required rework:

- require the terminal object to be a regular file (and distinguish a failed
  metadata/read from proven absence rather than laundering it into absence);
- add a negative test where `default.metallib` is a directory;
- drive the copied SwiftPM-built executable with that forged layout and require
  exit 2 with no listener; narrow the file-type gate and require that named test
  to fail.

### F3 — Repository hygiene: generated LLVM profile data is in the candidate

`tools/mlx-swift-runtime-prototype/default.profraw` is a 6,180,616-byte LLVM raw
profile (SHA-256
`fa36709de8bc22f4de0f2a7a2d5d5994c67cd5d96b5e66c44e2b5c1df7a2ed72`). It is
one of the 44 Change Request paths and is not matched by the package
`.gitignore`.

Required rework: remove it from revision 2 and ignore task-local LLVM profile
outputs (`*.profraw`, or a tighter equivalent).

### F4 — Evidence handoff is stale

`TASK-260827-qyebv8_results.md` still says the full-model load and generation
smokes are unproven and lists checklist items 2–5/8 as open. The later
`TASK-260827-qyebv8_full-model-smoke.md` proves them. Update the primary results
artifact (or clearly mark it superseded) so the next reviewer does not have two
contradictory task status statements.

The reviewer did not append to tracked `LOGBOOK.md`: changing a tracked file
after the CR snapshot would stale the candidate and violates the reviewer
read-only boundary. Revision 2 should record the two review regressions and
their fixes there.

## Verification

Rerun by this reviewer on the candidate worktree:

| Command | Result |
| --- | --- |
| `swift test -c release` | exit 0; 92 tests in 9 suites |
| `swift format lint --recursive --strict Sources Tests Package.swift` | exit 0 |
| `go test ./internal/infra/... -count=1` | exit 0; 119.808s |
| `go build ./...` | exit 0 |
| `go vet ./internal/infra/...` | exit 0 |
| `gofmt -l internal/infra/infra.go internal/infra/source_sync_build_artifacts_test.go` | exit 0, no output |
| `bash -n scripts/smoke.sh scripts/lifecycle-smoke.sh` | exit 0 |
| exact-tree `git diff --check` | exit 0 |

The reviewer did not repeat the 28 GB real-model load. The producer-attached
`TASK-260827-qyebv8_smoke-full-model.log` (25 PASS / 0 FAIL), preflight JSON,
lifecycle log and managed-run log were inspected and are accepted as evidence
for the successful positive path only. They do not exercise F1 or F2.

