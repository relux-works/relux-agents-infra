# TASK-260829-1qh0ud revision 4 review verdict

## Verdict

**Changes requested -> `to-dev`.** Change Request `CR-TASK-260829-1qh0ud-4`
revision 4 is not accepted.

Review target:

- base OID: `91356833949cb6a30958265514fe5852d97eec1b`
- candidate tree OID: `98f8f9506c4769001f557b8014b756c4e2b69981`
- repository delta: `present`
- patch SHA-256: `8274f1ffad0325307b589278971004d031c9343128987e26632172dace6b78bf`

## Acceptance-blocking finding: caller pressure policy can be silently narrowed

Severity: correctness and admission-safety defect affecting AC2 and AC3.

The runtime key deliberately excludes sharing policy, so an existing broker may
have a different effective sharing configuration from a new caller. The hello
frame carries the caller's `ConfiguredSharing`, and the broker returns its
`EffectiveSharing`, but neither production side validates resource-pressure
compatibility before acquisition:

- `sharedBrokerServer.attestClient` validates executable, protocol, runtime key,
  and profile digest, but not `hello.ConfiguredSharing` resource policy
  (`pi_shared_broker_darwin.go:958-993`).
- `sharedBrokerServer.acquireLease` classifies only
  `server.resolved.Sharing.ResourcePressure` (`pi_shared_broker_darwin.go:996-1051`).
- `connectAndAttestSharedRuntime` accepts `response.EffectiveSharing` without
  comparing its resource policy to `resolved.Sharing`
  (`pi_shared_client_darwin.go:411-475`).

Therefore an existing broker can silently weaken the caller's explicit gate.
The deterministic narrowing witness used the real
`sharedBrokerServer.handleConnection -> acquireLease` path:

- broker effective provider threshold: `1000` bytes;
- caller configured provider threshold: `900` bytes;
- provider observation: `950` bytes;
- actual result: a lease was granted, although the caller's explicit policy
  classifies the same observation as pressure and requires refusal.

The delete-shaped companion witness also reproduced: a caller configured with
`resource_pressure_mode = "provider"` received a lease from an existing broker
whose effective mode was `disabled`.

Negative-evidence shapes: **narrowed gate** and **bypass path around the check**.
The shipped production tests always make `hello.ConfiguredSharing` equal to the
broker's effective policy, so they cannot expose this mismatch.

Attack output:

```text
=== RUN   TestReviewAttackStricterCallerThresholdCannotBeNarrowedByExistingBroker
reviewer_resource_policy_mismatch_attack_test.go:65: caller threshold=900 was narrowed to effective threshold=1000 at observed bytes=950: lease=<redacted>
--- FAIL: TestReviewAttackStricterCallerThresholdCannotBeNarrowedByExistingBroker (0.01s)
FAIL
```

Scratch-only source and logs were created outside the candidate tree under
`.temp/TASK-260829-1qh0ud/policy-mismatch-attack/`. The CR worktree was not
modified by the reviewer.

## Required rework

Give the broker resource-pressure policy one explicit compatibility boundary
before lease acquisition. Fail closed with a typed refusal when the live
broker's effective policy would weaken the caller's configured policy. At
minimum cover provider-to-disabled downgrade and a wider effective pressure
threshold; strict equality of the resource-pressure mode/table is the simplest
coherent contract. Preserve the existing broker/runtime and do not start a
duplicate runtime on mismatch.

Add deterministic negative production tests that obtain the result through the
real hello/acquire entry point with intentionally different configured and
effective resource policies. Include both the narrowed-threshold witness and
provider-to-disabled downgrade. The test must fail when the compatibility check
is removed or narrowed, and normal status must continue to expose both
configured and effective policy provenance. Record the fix in `LOGBOOK.md` in
the producer run; this read-only reviewer did not mutate the immutable candidate.

## Independent validation

The finding survives otherwise-green validation:

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./... -count=1` | 0 | Full module suite passed; `internal/infra` 225.373s |
| `go vet ./...` | 0 | No diagnostics |
| Focused nine production resource tests under `-race` | 0 | `internal/infra` passed in 2.930s |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Unsupported POSIX surface compiles |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows surface compiles |
| `git diff --check <base> <candidate>` | 0 | No whitespace errors |
| Narrowed-policy production attack | 1 expected-red | Broker granted the forbidden lease |
| Provider-to-disabled production attack | 1 expected-red | Broker granted the forbidden lease |

The existing rev4 latch recovery, stale-generation, draining precedence,
unknown-read, versioned wire-fixture, and cross-platform composition work is
otherwise coherent. The verdict is limited to the uncovered resource-policy
compatibility bypass above.
