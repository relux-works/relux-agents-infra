# TASK-260829-1qh0ud revision 5 implementation report

## Exact replay and composition

- Base: `6d051f54440d36e3ca3d132f8d9d1e78d46289de` (`Merge pull request #12 from relux-works/codex/pi-shared-runtime-log-rotation`).
- Immutable revision-4 patch: `TASK-260829-1qh0ud_change-request_rev4.patch`.
- Verified patch SHA-256: `8274f1ffad0325307b589278971004d031c9343128987e26632172dace6b78bf`.
- Mechanical replay conflicted only in `LOGBOOK.md` and `tools/agents-infra/internal/infra/pi_config.go`. Both were merged semantically: all revision-4 resource-pressure entries/fields remain, and trunk-owned log rotation (`max_segment_bytes`, `max_segments`, secure rotating writer/tests/docs) is preserved.
- Public status continues to compose `restart_not_before`, `half_open`, versioned `resources`, and configured/effective sharing provenance on Darwin, unsupported POSIX, and Windows shapes.

No live model, user-owned runtime, external service, or user-owned socket was probed or mutated. Production-path tests use deterministic fake provider/process fixtures and task-owned local sockets only.

## Revision-5 policy fix

Production call site: `sharedBrokerServer.handleConnection -> acquireLease`.

- `sharedRuntimeResourcePolicyMatches` is the single compatibility decision for lease admission.
- Acquisition requires exact equality of `resource_pressure_mode` and the complete provider policy table fixed by the caller and live broker.
- Any difference refuses before provider observation and lease reservation with code `shared_runtime_resource_policy_mismatch`, reason `configured_resource_pressure_policy_differs`, mismatch field `sharing.resource_pressure`, and both configured/effective sharing values.
- Status-only access remains available so operators can inspect both policy provenances.
- Refusal preserves all existing leases, the broker/runtime PID, and single ownership; no duplicate runtime start path is reached.

Production negative cases:

- provider-configured caller against disabled broker;
- caller pressure threshold `900` against broker threshold `1000`;
- absent caller policy evidence.

All require zero provider observations, zero new leases, and unchanged runtime ownership. A narrowed mutant that compared only provider mode/presence granted the forbidden stricter-threshold lease and made the named production test exit 1. The restored code exits 0. An earlier mutant attempt was interrupted with exit 1 after 69.756s because the test waited for the handler before closing a wrongly granted connection; it is recorded only as a harness-ordering failure, not as gate evidence. The test was corrected to close after reading the response and before waiting for handler termination.

## Validation

All commands ran directly as foreground processes without `tee` or backgrounding.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run '^TestSharedBrokerProductionResourcePolicyMismatchRefusesBeforeObservationOrLease$' -count=1 -v` | 0 | Three mismatch witnesses pass after restore; final package time `0.495s` |
| Same exact test with provider table equality narrowed to presence | 1 expected-red | Stricter-threshold subtest received a real `lease` response; package `1.022s` |
| Focused resource/config/status/drain production slice, uncached | 0 | Resource package slice `1.786s` |
| Focused production resource/status slice under `go test -race`, uncached | 0 | Package `2.959s` |
| `go test ./cmd/model-harness ./internal/attachments ./internal/modelharness -count=1` | 0 | Complete small-package subset; command package has no tests |
| `go test . -count=1` | 0 | Complete root package `99.568s` |
| `go test ./internal/infra -count=1` | 0 | Complete infra package `168.636s` |
| `go test ./internal/infra -run '^TestSharedRuntimeProductionSingleFlightIndependentClientsCrashAndFinalRelease$' -count=1 -v` | 0 | Configured/effective status provenance plus single-owner integration `1.333s` |
| `go vet ./...` | 0 | Final post-edit run, no diagnostics |
| `go build ./...` | 0 | Darwin arm64 build |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Unsupported POSIX surface compiles |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows surface compiles |
| `gofmt -d internal/infra/*.go *.go` | 0 | Empty output |
| `git diff --check` | 0 | No whitespace errors |

The five packages returned by `go list ./...` are all covered by the three uncached test commands above; no package or test was skipped.

## Files and contracts

- Resource-pressure configuration and validation: `tools/agents-infra/internal/infra/pi_config.go`.
- Versioned provider observation/status and strict compatibility owner: `tools/agents-infra/internal/infra/pi_shared_resources.go`.
- Broker admission, pressure latch, drain/eviction, typed refusal: `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go`.
- Consumer error projection: `tools/agents-infra/internal/infra/pi_shared_client_darwin.go`.
- Deterministic wire fixtures and negative production tests: `pi_shared_resources_test.go` plus `testdata/shared-runtime-resource-*.json`.
- Operator/consumer docs: `README.md`, `SKILL.md`.
- Findings and replay decisions: `LOGBOOK.md`.

Revision 5 is ready for independent review against the immutable fresh Change Request produced from this exact base and candidate tree.
