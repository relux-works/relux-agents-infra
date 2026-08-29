# TASK-260827-2h39ya review verdict — CR revision 2

## Verdict

**Accepted.** Change Request `CR-TASK-260827-2h39ya-2` revision 2 satisfies the
task acceptance criteria and closes the revision-1 P1 narrowing failure.

Reviewed immutable delta:

- base OID: `9a55809ecf3bfbb49d9d329b71158145119bfc5f`
- candidate tree OID: `a61c1edab35a4a4e09d374d107ac97a5d313d487`
- patch SHA-256: `acc1dcb1667177c02d391a249c8de7999578e6ebc94ec0a8cdd34bbd4a1a6f06`
- repository delta: present, 14 changed paths

The patch digest was reproduced from the exact base/candidate diff before
review. After all validation, an alternate-index snapshot of the uncommitted
worktree reproduced candidate tree `a61c1eda...`, and the patch digest remained
unchanged. The reviewer modified no repository file.

## Round-2 boundary attack

Revision 1 treated `Resource limit` as independently fatal. Revision 2 replaces
that OR-match with `BackendFailureSignature.requiredFragments`; the recorded
Metal exhaustion requires both `[metal::malloc]` and `Resource limit`. Empty
fragment lists match nothing.

I rebuilt the macOS arm64 Release product from revision 2 and ran
`scripts/dead-generation-smoke.sh` against the real binary, real cached
Qwen1.5-0.5B model, and real `model-harness` v1.6.1-44-gd91d6fc. This drives
the production chain rather than calling the classifier directly:

- buffered entry point: `Router.complete` -> `Router.recordGenerationFailure`
  -> `RuntimeState.recordGenerationFailure`;
- streaming entry point: `RuntimeHTTPHandler.sendStream` ->
  `Router.recordGenerationFailure` -> `RuntimeState.recordGenerationFailure`;
- observation surfaces: `Router.health` via `HealthReport.make` and
  `Router.route(/v1/models)` via `ModelsListing.make`;
- recovery surface: the emitted `generation_worker_unavailable` marker consumed
  by `model-harness` supervision.

The result was 45 checks, 0 failures:

1. Healthy control: `/health` returned HTTP 200 before and after a real
   generation.
2. Original `BUG-260827-1jhv2g` incident text —
   `RuntimeError: [metal::malloc] Resource limit (499000) exceeded` — returned
   the provoking request as 500, changed `/health` from 200 to 503
   `{"status":"unavailable"}`, stopped model advertisement, refused later
   completions with 503, and did so through both buffered and streaming paths.
3. With the configured fatal substring, `model-harness` recorded exactly one
   supervised restart (`1/2`) naming `generation_worker_unavailable`; the
   replacement rebound and returned `/health` 200.
4. The revision-1 neighbour —
   `RequestError: Resource limit for this request is 8 tokens` — returned the
   request as 500 while `/health` and `/v1/models` remained 200, emitted no
   marker, caused no restart, and continued serving subsequent requests.

This independently proves both sides requested by the round-2 brief: the fix no
longer over-condemns the generic neighbour, and it did not overshoot into an
over-narrow classifier that misses the original Metal allocator incident. It
also defeats the **check present but uncalled from production**, **bypass path
around the check**, and **narrowing, not deleting** negative shapes for the two
completion paths and the supervised recovery surface.

## Independent validation

| Command / check | Reviewer result |
| --- | --- |
| `swift test -c release` | exit 0; 119 tests in 11 suites |
| `xcrun swift-format lint --configuration .swift-format --recursive Sources Tests` | exit 0; clean |
| `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData -skipPackagePluginValidation -skipMacroValidation` | exit 0; `BUILD SUCCEEDED` |
| `scripts/dead-generation-smoke.sh` on reviewer port 18029 | exit 0; 45 checks, 0 failures |
| `scripts/lifecycle-smoke.sh` with an absolute reviewer `OUT` | exit 0; 17 checks, 0 failures |
| `bash -n scripts/dead-generation-smoke.sh` | exit 0 |
| `git diff --check <base> <candidate>` | clean |

One initial reviewer wrapper invocation did not start `swift test` because its
log path was wrong and it assigned zsh's read-only `status` variable. That
wrapper failure is recorded separately; the corrected command above ran to
completion and is the test evidence used for this verdict.

## Scope and architecture

The exact 14-path CR contains no `.go` file and no path under
`tools/agents-infra/`. The producer's statement that the Go gates were not
re-run for revision 2 is therefore accurate; I did not treat their absence as a
pass and did not re-run unrelated Go packages. The delta is confined to the MLX
Swift runtime contract/executable, its Swift Testing suite, repeatable shell
acceptance check, example supervision policy, and project documentation/logbook.

The solution keeps readiness and engine ownership in `RuntimeState`, centralizes
classification and health response semantics in the contract library, wires all
production generation paths, and leaves recovery to the existing supervisor.
The default Python runtime and installed configuration remain unchanged.

No blocking or non-blocking code finding remains for revision 2.
