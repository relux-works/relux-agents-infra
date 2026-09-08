# TASK-260829-1qh0ud revision 7 implementation report

## Exact lineage and scope

- Exact Story base and current `HEAD`: `6d051f54440d36e3ca3d132f8d9d1e78d46289de`.
- Immutable revision 6 candidate reviewed: tree `7b05d0eeb10fac3df1524c6afc897e01b6ef98c7`, patch SHA-256 `43fbb7514f03813e7d8e77b21e187fdbdec11634bccf61879a42906e12eac525`.
- Revision 7 preserves revision 6 record-derived provenance, exact policy equality, restart status, log rotation, lease ownership, and pressure drain/eviction behavior.
- No live model, user-owned runtime, external service, endpoint, or user-owned socket was contacted or mutated. Tests use task-owned local Unix connection pairs and deterministic fake provider observations.

## Concurrency contract

Production call sites: `sharedBrokerServer.handleConnection` status and acquire cases, `observeResourceStatus`, `acquireLease`, and `resourceStatusSnapshot`.

- Admission and diagnostic provider reads now have separate monotonic generations.
- Healthy/busy status polls advance diagnostic freshness only and cannot invalidate a healthy acquire paused before lease reservation.
- Direct pressure observed by status advances admission invalidation, so an older healthy acquire refuses before reservation. Status remains read-only for the pressure latch and eviction timer.
- Immediately before constructing the status frame, `resourceStatusSnapshot` revalidates status and admission generations while snapshotting the pressure latch, broker state, and leases under the same mutex.
- A superseded status decision publishes explicit `unknown/refused` with `resource_observation_stale`; it cannot be laundered into `healthy/admitted` or overwritten by the current latch.
- Existing leases and the single broker-owned runtime remain unchanged in all three new production schedules.

## Production regressions and negative evidence

New real-handler tests:

- `TestSharedBrokerProductionPressureCannotSupersedeStatusBeforePublication`
- `TestSharedBrokerProductionHealthyStatusPollingCannotStarveAdmission`
- `TestSharedBrokerProductionPressuredStatusInvalidatesPendingAdmission`

Evidence commands ran directly in the foreground without `tee` or pipelines.

| Evidence | Exit | Result |
| --- | ---: | --- |
| Two reviewer schedules before the fix | 1 expected-red | Status published healthy after pressure latched; four healthy polls starved the pending acquire. |
| First final-publication mutant attempt | 0 survivor | The original assertion accepted latch-forced pressure and did not prove generation revalidation. This was preserved as an anomaly, then the assertion was hardened to require explicit stale unknown. |
| Final-publication revalidation disabled after hardening | 1 expected-red | The real status/acquire race no longer returned `resource_observation_stale`. |
| Healthy status re-coupled to admission generation | 1 expected-red | Four healthy polls again caused the real pending acquire to refuse stale. |
| Restored three-test production slice | 0 | Publication race, healthy fairness, and pressured invalidation all passed. |
| Full resource/status production slice under `-race -count=1` | 0 | Record provenance, all policy mismatch fields, refusal/recovery, stale ordering, draining, and the three revision-7 schedules passed in `3.032s`. |

## Final validation

All final commands ran after the last production and test edit.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test -race ./internal/infra -run <13 production resource/status tests> -count=1 -v` | 0 | `internal/infra` `3.032s`. |
| `go test ./... -count=1` | 0 | Root `115.107s`; attachments `3.341s`; infra `189.201s`; modelharness `2.274s`; command package has no tests. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build ./...` | 0 | Darwin arm64. |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Unsupported POSIX status shape compiles. |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows status shape compiles. |
| `gofmt -d internal/infra/*.go *.go` | 0 | Empty output. |
| `git diff --check` | 0 | Empty output. |

## Documentation and institutional memory

- `README.md` and `SKILL.md` describe separate diagnostic/admission freshness, pressure-only invalidation, and atomic final status publication.
- `LOGBOOK.md` records the publication race and the observability starvation root causes as separate entries, plus the chosen generation owner contract and negative evidence.

Revision 7 is ready for independent review after the managed developer handoff publishes a fresh immutable Change Request.
