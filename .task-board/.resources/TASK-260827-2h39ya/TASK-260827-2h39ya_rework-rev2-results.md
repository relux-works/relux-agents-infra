# TASK-260827-2h39ya — rework of CR revision 1

Reviewer verdict `CR-TASK-260827-2h39ya-1` rev 1: **changes requested**, one P1 —
`GenerationWorkerHealth.invalidatingSignatures` listed `metal::malloc` and
`Resource limit` as two *independent* substrings, so a request-scoped failure
carrying the generic words condemned a healthy runtime.

## What was wrong

The incident message asserts two facts at once:

    RuntimeError: [metal::malloc] Resource limit (499000) exceeded

that MLX's Metal allocator produced it, **and** that the allocator is exhausted.
Revision 1 split it into OR-matched fragments. That keeps the positive case
green and silently widens the fatal class to anything carrying either half —
including the reviewer's probe, `RequestError: Resource limit for this request
is 8 tokens`, which took `/health` and `/v1/models` to 503, dropped the engine,
emitted the fatal marker, and got the runtime restarted.

The shipped negative coverage could not catch it. Revision 1's benign case
shared no words with any signature, so it proved the gate is not condemn-all and
nothing about how wide the fatal class is. A negative case has to sit next to
the boundary, not somewhere in the safe interior.

## What changed

| File | Change |
| --- | --- |
| `Sources/MLXSwiftRuntimeContract/GenerationWorkerHealth.swift` | `BackendFailureSignature` — a named signature with `requiredFragments` that must **all** match. `metal-allocator-resource-limit` requires `[metal::malloc]` **and** `Resource limit`; `metal-shader-library-unreachable` requires its one full phrase. Empty fragment list matches nothing. |
| `Tests/.../GenerationWorkerHealthTests.swift` | Four narrowing cases added to the request-scoped set, plus three new tests: the paired allocator-context assertion, the empty-signature fail-closed test, and a sweep asserting no shipped signature matches either neighbour. |
| `scripts/dead-generation-smoke.sh` | Negative phase extracted into `negative_phase()`; run twice — 4a benign, **4b the reviewer's exact message**. Each phase also re-checks that the runtime keeps serving afterwards. |
| `README.md` (prototype) | Signature table with the upstream source line for each; both failure directions spelled out. |
| `LOGBOOK.md` | New entry `2242 — A Refusal Boundary Wide Enough To Catch Its Own Neighbour`. |

### Why `[metal::malloc]` bracketed, and why the conjunction is not arbitrary

Not a guess. The vendored MLX in `.build/checkouts/mlx-swift` emits it verbatim:

- `mlx/backend/metal/allocator.cpp:141-144` → `"[metal::malloc] Resource limit (" << resource_limit_ << ") exceeded."` — the pool is exhausted; the next request cannot be served either. **Fatal.**
- `mlx/backend/metal/allocator.cpp:111-117` → `"[metal::malloc] Attempting to allocate N bytes which is greater than the maximum allowed buffer size of M bytes."` — one oversized allocation refused, pool intact. **Request-scoped**, and deliberately excluded. Carrying the allocator tag is not on its own proof that the backend is gone.

So the fatal class is exactly "the Metal allocator reports exhaustion", and both
fragments are load-bearing in opposite directions.

## Would it have failed

| Mutant | Applied to | Result |
| --- | --- | --- |
| Widened: bare `Resource limit` restored as an independent signature | contract suite | **red**, 6 issues across 3 tests |
| Widened, same mutant, **Release rebuilt** | `scripts/dead-generation-smoke.sh` | **exit 1, 7 failures, all inside phase 4b**; every other phase still green |
| Over-narrowed: extra `ArraysCache` fragment required | contract suite | **red**, 5 issues — the incident stops being recognised |
| Fail-closed guard on an empty fragment list deleted | contract suite | **red**, 2 issues |

The second row is the one that matters: it reproduces the reviewer's finding at
the production entry point on a real binary under real `model-harness`, and the
new phase is what turns it red. Because 4a stays green under the same mutant,
the suite now distinguishes *deleted*, *over-broad* and *correctly narrowed* —
revision 1 could only distinguish deleted.

Evidence: `TASK-260827-2h39ya_overbroad-reproduction.txt`.

## Validation (real exit codes)

| Command | Result |
| --- | --- |
| `swift test -c release` | exit 0 — 119 tests, 11 suites (was 116) |
| `xcrun swift-format lint --configuration .swift-format --recursive Sources Tests` | exit 0, no output |
| `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release ... -skipPackagePluginValidation -skipMacroValidation` | exit 0, `** BUILD SUCCEEDED **` |
| `scripts/dead-generation-smoke.sh` (fixed build) | exit 0 — **45 checks, 0 failures** (was 35) |
| `scripts/dead-generation-smoke.sh` (over-broad mutant build) | **exit 1 — 7 failures, all in phase 4b** (expected red) |
| `scripts/lifecycle-smoke.sh` | exit 0 — 17 checks, 0 failures |

Smoke runs used the real Release binary, the real `/Users/alexis/.local/bin/model-harness`,
and the cached 261 MB `mlx-community/Qwen1.5-0.5B-Chat-4bit` on port 18019.

**Not re-run:** the Go gates (`go build` / `go vet` / `go test ./...` under
`tools/agents-infra`). This delta touches only Swift sources, two shell/markdown
files and `LOGBOOK.md`; no Go file is in it. Stated rather than silently skipped.

## Note for whoever runs the smokes next

`scripts/lifecycle-smoke.sh` does not absolutize `OUT` the way
`scripts/dead-generation-smoke.sh` does, so a relative `OUT=` makes the fixture
path relative and the runtime correctly refuses `--model` — 12 checks go red for
a reason that has nothing to do with the code. Pass an absolute `OUT`. Left
unchanged: outside this rework's scope.

## Scope

Python `mlx-lm` remains the default local runtime and the rollback path.
`examples/model-harness.prototype.toml` is still an example, not an installed
config. Uncommitted, per `version_control.confirm = true`.
