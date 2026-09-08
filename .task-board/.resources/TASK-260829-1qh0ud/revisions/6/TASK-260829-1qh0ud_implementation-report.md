# TASK-260829-1qh0ud revision 6 implementation report

## Exact candidate lineage

- Exact base and current `HEAD`: `6d051f54440d36e3ca3d132f8d9d1e78d46289de` (`origin/main`, shared-runtime log rotation accepted on trunk).
- Revision 5 immutable candidate tree reviewed: `bd49d59fed93ad7b11a425cedf8728b78dda16b0`.
- Revision 5 patch SHA-256 reviewed: `ba5072d68fb1e7075df9bda5927f44ee8515113ef0fa2711afecec23c5852718`.
- Revision 6 preserves the full revision 5 pressure/status/log-rotation composition and changes only the rejected provenance/evidence surface plus its documentation.
- No live model, user-owned runtime, external service, user-owned socket, or model endpoint was probed or mutated. All production-path tests use task-owned local Unix sockets and deterministic fake provider observations.

## Record-derived policy provenance fix

Production call site: `SharedRuntimeStatusReport -> sharedRuntimeRecordStatus`.

- A readable broker record now supplies one coherent resource policy provenance: both `sharing.effective` and `resources.policy` come from `record.Sharing`.
- Provider facts remain `unknown/refused` with source `record-derived-unverified`; the record is not promoted to an attested observation.
- A pre-extension record with no sharing field reports `resources.policy.mode = unknown`, reason `resource_pressure_policy_unknown`, and refused admission.
- Semantically partial record policies (`provider` without a table or `disabled` with a table) also fail closed as unknown/refused; they are not laundered into disabled/not-enforced.
- Valid record-effective disabled policy may publish disabled/not-enforced, while valid record-effective provider policy publishes its complete table with unavailable observations and refused admission.

## Field-independent live-broker mismatch evidence

Production call site: `sharedBrokerServer.handleConnection -> acquireLease`.

- The real hello/acquire path now has independent mismatch witnesses for both disabled/provider directions, missing caller evidence, observation path, observation timeout, pressure threshold, recovery threshold, eviction grace, pressure action, unknown action, and busy action.
- Every mismatch must return `shared_runtime_resource_policy_mismatch` before provider observation or lease reservation, with configured/effective provenance.
- Every case verifies zero provider observations, zero new leases, unchanged runtime PID, and no duplicate broker/runtime ownership.

## Negative evidence

Commands were run directly as foreground processes without `tee` or shell pipelines.

| Attack | Exit | Evidence |
| --- | ---: | --- |
| Pre-fix `SharedRuntimeStatusReport` record-provenance test | 1 expected-red | All three initial cases exposed caller-derived disabled/provider policy instead of the record-effective or explicit-unknown policy; package `1.260s` |
| Pressure-threshold-only equality mutant through `handleConnection -> acquireLease` | 1 expected-red | Recovery/path/timeout/grace/all-action single-field cases received real leases; package `0.486s` |
| Restored focused status and mismatch tests | 0 | Both production entry points green; package `0.983s` before the semantic-partial extension |
| Final focused production tests under `-race -count=1` | 0 | Both mismatch directions, missing/partial records, and all independent policy fields green; package `2.132s` |

## Final validation

All final commands ran after the last production-code edit.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test -race ./internal/infra -run '^(TestSharedRuntimeStatusReportRecordDerivedResourcePolicyUsesCoherentProvenance\|TestSharedBrokerProductionResourcePolicyMismatchRefusesBeforeObservationOrLease)$' -count=1` | 0 | `internal/infra` `2.132s` |
| `go test ./... -count=1` | 0 | root `82.385s`; attachments `1.255s`; infra `142.577s`; modelharness `1.383s`; command package has no tests |
| `go vet ./...` | 0 | No diagnostics |
| `go build ./...` | 0 | Darwin arm64 build |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Unsupported POSIX public status shape compiles |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows public status shape compiles |
| `gofmt -d internal/infra/*.go *.go` | 0 | Empty output |
| `git diff --check` | 0 | No whitespace errors |

## Documentation and institutional memory

- `README.md` documents record-derived coherent policy provenance, explicit unknown semantics for missing/partial records, and every field in the exact live-broker mismatch gate.
- `SKILL.md` carries the same operator contract without claiming record evidence is attested.
- `LOGBOOK.md` records the root cause, fix, production evidence, and ownership invariant under the `2026-08-29 2235` entry.

Revision 6 is ready for independent review after the managed developer handoff publishes a fresh immutable Change Request on the exact base above.
