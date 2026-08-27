# TASK-260827-qyebv8 review verdict — Change Request revision 2

## Verdict

**Accepted.** Change Request `CR-TASK-260827-qyebv8-2`, revision 2, satisfies
the task acceptance criteria and closes all four findings from revision 1.
No blocking review finding remains.

This reviewer did not modify the candidate. Narrowing mutants and filesystem
attack fixtures were created only under task-scoped `.temp/` paths.

## Candidate reviewed

- Base OID: `3f313d9175f2ada9b9ab3320ab524c0918f9daac`
- Candidate tree OID: `2d1a4a2dea34704a386464696e8fc2b2ad9d718a`
- Patch SHA-256: `79df45b995f4f52fdc6ba799b536af2d638dd933a9835e5c189cc5330c290f0`
- Repository delta: present, 45 changed paths
- Exact-tree `git diff --check`: exit 0
- Every changed worktree file hashed to the candidate blob: 0 mismatches

## Prior findings

### F1 — closed: out-of-range `max_tokens`

The production path is `Router.chatCompletions(body:)` ->
`ChatCompletionRequest.decode(from:)` -> `optionalInt(_:field:)`. Conversion is
now total through `Int(exactly:)`; the router maps the throw to bounded HTTP
`400 invalid_body`.

Reproduced against the freshly rebuilt Xcode product serving the exact 28 GB
Qwen through `model-harness run`:

- `1e300`, `-1e300`, and `9223372036854775808` each returned 400 with the
  representable-range refusal;
- `/health` returned 200 after every request;
- `max_tokens: 0` still reached the separate `invalid_max_tokens` bound.

The same negative probe was run against a task-scoped narrowing mutant that
replaced `Int(exactly:)` with `Int(_)`: baseline exit 0 with the expected throw;
mutant exit 133 with the original fatal conversion. The prior bypass is both
closed at the real HTTP entry point and observable to a narrowing negative test.

### F2 — closed: forged Metal-library directory

The production path is `Main.main()` -> `MetalShaderLibraryCheck.inspect` ->
`classify` -> `admit`, before `RuntimeHTTPServer.start`.

The composed SwiftPM executable probe independently passed all 10 checks:

- directory and dangling-symlink forgeries exited 2;
- neither bound a listener or emitted `listening`;
- the refusal named the directory;
- missing-bundle control exited 2;
- regular-file positive control reached the listener;
- the port was free after synchronous cleanup.

A scratch narrowing mutant changed only `S_IFDIR` classification to
`regularFile`. The same probe passed on baseline (exit 0) and failed on the
mutant (exit 1, forged directory admitted). This reproduces the producer's F2
mutation claim rather than accepting it as self-reported evidence.

### F3 — closed: generated profile data

No `.profraw`, `.profdata`, `.build`, `DerivedData`, or `.temp` path exists in
the candidate tree. Package `.gitignore` covers all four build/profile-output
classes. Reviewer validation caused the instrumented Xcode product to create a
new `default.profraw`; Git reported it as ignored, proving the rule on the real
producer, and the reviewer removed that reproducible local artifact afterward.

### F4 — closed: stale evidence handoff

`TASK-260827-qyebv8_results.md` is now the single current summary. It records
103 tests, the green 36-pass full-model smoke, and identifies
`TASK-260827-qyebv8_blocker.md` as a historical revision-1 host condition that
no longer describes task state. The results and blocker resources are no longer
contradictory.

## Additional `ModelDirectoryCheck.observe` hardening

The unrequested hardening was attacked through `Main.main`, not accepted from
helper tests alone:

- a directory named `config.json` exited 2 and was identified as a directory;
- a directory named `model.safetensors` exited 2 and was identified as a
  directory;
- an unreadable symlink target for required `config.json` exited 2 as
  `Permission denied (errno 13)`, not as missing;
- none of ports 28128, 28129, or 28131 was left listening;
- unit coverage retained the positive symlink-to-regular-file path, and the
  exact real model loaded successfully after the hardened gate.

The additional gate therefore refuses forged objects, preserves unknown/read
failure as a distinct fact, and remains connected to the actual startup entry
point.

## Independent validation

| Command / attack | Result |
| --- | --- |
| `swift test -c release` | exit 0; 103 tests / 9 suites |
| `swift format lint --recursive --strict Sources Tests Package.swift` | exit 0 |
| documented release `xcodebuild build ...` | exit 0; `BUILD SUCCEEDED` |
| `swift build -c release` | exit 0 |
| exact-model `preflight` | exit 0; exact path, pins, Qwen 3.5 registry/config, tokenizer, template, reasoning start, and tool format passed |
| `scripts/smoke.sh` through `model-harness run` | exit 0; 36 PASS / 0 FAIL |
| `scripts/metallib-gate-probe.sh` | exit 0; 10 PASS / 0 FAIL |
| `go test ./internal/infra/... -count=1` | exit 0; 125.883s |
| `go build ./...` / `go vet ./internal/infra/...` | exit 0 / exit 0 |
| changed-file `gofmt -l` / smoke-script `bash -n` | no output / exit 0 |

The live smoke loaded the exact configured model in 6.06s, reported a
28,261.7 MiB physical footprint, exercised non-streaming, streaming,
reasoning separation, tool calls, models/readiness, and lifecycle shutdown,
then synchronously reaped the runtime and released port 28130.

`Package.resolved`, the compiled revision strings, and preflight agree on
`mlx-swift` 0.31.6 / `0bb916c67f4b9e5c682cbe02a42c701c93ab5021` and
`mlx-swift-lm` 3.31.4 / `bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57`.
No default `model-harness.toml`, project config, installed binary, or Pi profile
is changed; Python `mlx-lm` remains the default rollback runtime.

One non-blocking build note remains: `swift build` reports the pinned upstream
`BaseConfiguration.quantization` API as deprecated. The strict formatter is
clean, the documented Xcode product builds and runs, and this diagnostic-only
preflight read does not affect acceptance of the pinned prototype.
