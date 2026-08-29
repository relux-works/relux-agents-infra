# TASK-260827-2h39ya review verdict

## Verdict

**Changes requested → `to-dev`.** Change Request `CR-TASK-260827-2h39ya-1`
revision 1 is not accepted.

Reviewed immutable delta:

- base OID: `9a55809ecf3bfbb49d9d329b71158145119bfc5f`
- candidate tree OID: `1fd50f20d0f65d2843a294a42263be77b11a05e0`
- patch SHA-256: `6196924ed4887da735335726e07e5f84172833c7595e161b51f70bef0aa5cdac` (verified)

## Blocking finding

### P1 — Generic `Resource limit` substring condemns request-scoped failures

`GenerationWorkerHealth.invalidatingSignatures` lists `"metal::malloc"` and
`"Resource limit"` as two independent substring matches
(`GenerationWorkerHealth.swift:64-78`). The incident evidence establishes the
combined Metal allocator message, not every error containing the generic words
`Resource limit`. Because `Router.recordGenerationFailure` applies this
classifier to every thrown generation error (`Router.swift:71-85`, called from
`Router.swift:160-166` and `HTTPServer.swift:155-165`), an otherwise
request-scoped error whose text contains those words permanently drops the
engine, changes `/health` and `/v1/models` to 503, and emits the fatal supervisor
marker.

This is the negative-evidence **narrowing, not deleting** failure. The shipped
negative case (`dead-generation-smoke.sh:56-59,380-414`) avoids every fatal
substring, so it proves only that one benign phrase stays healthy; it does not
prove that the fatal class is narrow enough. The unit-test cases have the same
gap.

Production-entry-point reproduction on the freshly rebuilt Release binary:

```text
Injected failure: RequestError: Resource limit for this request is 8 tokens
ready=1
health_before=200
chat=500
health_after=503
models_after=503
health body={"detail":"RequestError: Resource limit for this request is 8 tokens","status":"unavailable"}
emitted event={"detail":"RequestError: Resource limit for this request is 8 tokens","event":"generation_worker_failed","marker":"generation_worker_unavailable"}
```

Expected for a request-scoped failure: request 500, `/health` remains 200,
`/v1/models` remains 200, and no supervision marker/restart. Actual behavior
would take a healthy runtime out of rotation and, with the shipped example
policy, restart it.

Required rework:

1. Narrow the incident classifier to backend-specific evidence. At minimum,
   the Metal incident must require its Metal allocator context rather than
   treating generic `Resource limit` as independently fatal; prefer typed
   evidence if MLX exposes it.
2. Add contract and production E2E negative coverage for a request-scoped
   error that contains `Resource limit` but not the Metal allocator signature.
   Require health/models 200, no fatal marker, and no supervised restart.
3. Re-run the 116-test contract suite and the full dead-generation smoke, and
   update `LOGBOOK.md` with this reviewer-found narrowing regression. The
   reviewer did not mutate the logbook because reviewer runs are read-only and
   the CR candidate must remain immutable.

## Validation performed

- `swift test -c release`: 116 tests / 11 suites passed.
- `xcrun swift-format lint --configuration .swift-format --recursive Sources Tests`: clean.
- `bash -n scripts/dead-generation-smoke.sh scripts/lifecycle-smoke.sh`: clean.
- `xcodebuild ... -configuration Release ...`: `BUILD SUCCEEDED` on macOS arm64.
- Official `scripts/dead-generation-smoke.sh` against that rebuilt binary:
  35 checks, 0 failures (healthy 200, invalidated 503, streaming path,
  supervised recovery, benign control).
- Independent adversarial production probe: reproduced the blocking false
  condemnation above.
- `git diff --check` on the exact CR delta: clean.

The intended dead-worker path and supervised recovery work, but the refusal
boundary is unsafe and its negative coverage does not catch the over-broad
class. Revision 1 therefore cannot be accepted.
