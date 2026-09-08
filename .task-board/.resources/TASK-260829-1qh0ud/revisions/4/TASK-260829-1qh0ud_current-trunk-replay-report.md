# TASK-260829-1qh0ud current-trunk replay report

## Scope and composition

- Replayed the immutable revision-3 resource-pressure candidate from board resource `TASK-260829-1qh0ud_change-request_rev3.patch` onto exact current trunk `91356833949cb6a30958265514fe5852d97eec1b`.
- Verified source patch SHA-256: `f55348fc64e7e7ef0782dc220ea78ffe21740bbb2bca190ca6b9a756a9da4c49`.
- Every code, test, fixture, README, and skill hunk applied without textual conflict. `LOGBOOK.md` was merged semantically because current trunk contains the later 20:35 output-capture entry; all resource-pressure entries and the later trunk entry are preserved in chronological order.
- The concurrent trunk status additions remain composed: `SharedRuntimeStatusReport` carries `restart_not_before`, `half_open`, and the versioned `resources` status together.
- No live model, user-owned runtime, service, endpoint, or socket was probed or mutated. All behavior evidence uses deterministic fake-provider fixtures through the production broker handler.

## Delivered behavior

- Explicit `disabled` or `provider` resource-pressure configuration with no implicit mode, threshold, timeout, action, or eviction defaults.
- Strict bounded provider observations for independent loaded-model-memory and inference-busy facts, including explicit unknown semantics for read, schema, model, and fact failures.
- Protocol-v7 `agents-infra.pi.shared-runtime.resource-status.v1` consumer handoff with healthy, busy, pressured, draining, and unknown states.
- Pressure refusal preserves existing connection-bound leases and the single broker-owned runtime; idle eviction follows the configured drain/grace sequence.
- Monotonic provider-observation generations prevent stale recovery from clearing newer pressure or granting a lease, including the post-observation lease-reservation gate.
- Read-only status cannot publish hypothetical recovery while the broker still enforces its pressure latch; draining remains authoritative.

Production negative tests drive `sharedBrokerServer.handleConnection` through its real `status` and `acquire` cases:

- `TestSharedBrokerProductionStatusCannotBypassLatchedPressureRecovery`
- `TestSharedBrokerProductionStaleRecoveryObservationCannotClearNewerPressure`
- `TestSharedBrokerProductionSupersededAdmissionCannotGrantBeforeLeaseReservation`
- `TestSharedBrokerProductionStaleStatusObservationCannotLaunderNewerPressure`
- `TestSharedBrokerProductionUnknownResourceObservationRefusesLease`
- `TestSharedBrokerProductionStatusPreservesDrainingOverLatchedPressure`

## Validation and exact exits

| Command | Exit | Result |
| --- | ---: | --- |
| Focused eight production resource/status tests, uncached | 0 | `internal/infra` pass in 1.925s |
| Three stale-generation production tests under `go test -race`, uncached | 0 | `internal/infra` pass in 1.659s |
| `go test ./... -count=1` | 0 | Root 135.782s; attachments 2.545s; infra 248.089s; modelharness 1.402s |
| `go vet ./...` | 0 | No diagnostics |
| `go build ./...` on Darwin arm64 | 0 | Build succeeds |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Build succeeds |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Build succeeds |
| `gofmt -d tools/agents-infra/internal/infra/*.go tools/agents-infra/*.go` | 0 | Empty diff, independently asserted with exit 0 |
| `git diff --check` | 0 | No whitespace errors |

Raw logs are attached separately for the focused tests, race tests, and complete configured test suite. Empty diagnostic logs for vet, builds, gofmt, and diff-check remain under `.temp/TASK-260829-1qh0ud/`.
