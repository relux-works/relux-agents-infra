# TASK-260829-1qh0ud revision 6 review verdict

## Verdict

**Changes requested → `to-dev`.** CR `CR-TASK-260829-1qh0ud-6` revision 6 is not accepted.

Review target: base `6d051f54440d36e3ca3d132f8d9d1e78d46289de`, candidate tree `7b05d0eeb10fac3df1524c6afc897e01b6ef98c7`, repository delta present, 22 changed paths, patch SHA-256 `43fbb7514f03813e7d8e77b21e187fdbdec11634bccf61879a42906e12eac525`.

## Finding: post-observation status publication bypasses the generation gate

Severity: acceptance-blocking correctness defect (AC1 and AC3).

The production status path obtains `resources` and the broker snapshot in two separate critical sections at `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go:953-956`. `observeResourceStatus(false)` validates its generation while holding `server.mu` at lines 1100-1131, then unlocks and returns. A newer acquire can increment the generation and latch pressure after that unlock but before `statusSnapshot`. The older status response is never revalidated and can publish `resources=healthy/admitted` while the broker's enforced `pressureLatched` is already true.

This is the same protected-state contradiction the status contract must prevent, at a later scheduling boundary than the existing stale-completion test. `TestSharedBrokerProductionStaleStatusObservationCannotLaunderNewerPressure` covers a newer pressure observation completing before the older status observation returns; it does not cover pressure arriving after the status observation returns but before publication.

Negative-evidence shape: **bypass path around the check**. The generation check is present inside observation completion, but the production publish path has an unlocked observe-to-snapshot window after it.

## Attack evidence

A scratch-only copy of the immutable candidate added one synchronization-only hook immediately after the production `observeResourceStatus(false)` call and before the unchanged production `statusSnapshot`. No candidate file was modified. The test drove the real `sharedBrokerServer.handleConnection` status and acquire cases over local Unix connection pairs:

```text
go test ./internal/infra -run '^TestReviewAttackPressureCanSupersedeStatusAfterObservationBeforePublish$' -count=1 -v
=== RUN   TestReviewAttackPressureCanSupersedeStatusAfterObservationBeforePublish
    reviewer_post_observe_status_race_test.go:60: reproduced: wire broker_state=serving resources=healthy/admitted while pressure_latched=true
--- PASS: TestReviewAttackPressureCanSupersedeStatusAfterObservationBeforePublish (0.00s)
PASS
```

Scratch source and log:

- `.temp/TASK-260829-1qh0ud/review-rev6/post-observe-status-race/internal/infra/reviewer_post_observe_status_race_test.go`
- `.temp/TASK-260829-1qh0ud/review-rev6/post-observe-status-race.log`

The revision-6 fixes themselves have meaningful negative evidence:

- narrowing live-broker policy equality to pressure-threshold-only makes the real `handleConnection -> acquireLease` table fail on recovery threshold, path, timeout, grace, and all three actions (`policy-narrowing-mutant.log`, exit 1);
- deriving record resource status from caller configuration makes the real `SharedRuntimeStatusReport` provenance table fail in both mismatch directions and both partial-record cases (`record-provenance-mutant.log`, exit 1).

Those gates are correct, but they do not close the newly reproduced post-observation publication window.

## Finding: status polling can indefinitely starve healthy lease admission

Severity: acceptance-blocking liveness defect (AC3).

Status requests and acquire requests increment the same `resourceObservationGeneration` at `pi_shared_broker_darwin.go:1093-1095`. The acquire path correctly refuses when its generation changes before lease reservation, but any status observation can cause that change. There is no bounded retry, admission priority, or separation between diagnostic and admission generations. Sustained observability can therefore keep an otherwise healthy runtime from granting any lease.

A second scratch-only test used the candidate's existing `beforeLeaseReservation` seam and the real `handleConnection` status/acquire cases; production code was unchanged. For each attempt, a healthy acquire completed observation and paused before reservation, one healthy status poll completed, and the acquire then failed as stale:

```text
go test ./internal/infra -run '^TestReviewAttackHealthyStatusPollingCanStarveHealthyAdmissions$' -count=1 -v
=== RUN   TestReviewAttackHealthyStatusPollingCanStarveHealthyAdmissions
    reviewer_status_starvation_test.go:69: reproduced: 4 healthy status polls superseded 4 healthy admissions; zero leases granted
--- PASS: TestReviewAttackHealthyStatusPollingCanStarveHealthyAdmissions (0.01s)
PASS
```

Negative-evidence shape: **bypass path around the check**. Status observability shares and advances the admission invalidation counter without itself owning admission, so a safe stale-snapshot gate becomes an unbounded denial path.

## Independent validation

All commands ran in the exact Story worktree candidate; no live model, user-owned runtime, service, endpoint, or user-owned socket was contacted.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./... -count=1` | 0 | root 104.758s; attachments 2.143s; infra 178.010s; modelharness 1.162s |
| focused production resource/status suite under `-race -count=1` | 0 | infra 3.450s |
| `go vet ./...` | 0 | no diagnostics |
| `go build ./...` | 0 | Darwin arm64 |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | unsupported POSIX surface compiles |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows surface compiles |
| `gofmt -d internal/infra/*.go *.go` | 0 | empty output |
| `git diff --check <base> <candidate>` | 0 | no whitespace errors |

## Required rework

Make resource generation/latch and the status response one coherent publication decision. Before sending a status response, atomically prove that its resource generation is still current with the broker/latch snapshot it publishes; if superseded, return explicit `unknown/refused` or retry a bounded observation. Do not publish the older healthy/admitted snapshot.

Add a deterministic real `handleConnection` regression for the exact sequence: status observation completes healthy, newer acquire latches pressure, then the older status reaches publication. Prove the wire response cannot be healthy/admitted, the latch remains enforced, no lease is granted by the stale status path, and broker/runtime ownership remains single. A helper-only classifier assertion is insufficient.

Separate diagnostic observation freshness from lease-admission invalidation, or add an explicit bounded retry/fairness contract that prevents status polling from indefinitely superseding healthy acquisitions. Add a deterministic production test with repeated healthy status polls interleaved at the pre-reservation boundary and prove a healthy acquire succeeds within the pinned bound while pressure observations still invalidate unsafe admissions.

Add the post-observation publication race and its resolution to `LOGBOOK.md` in the next immutable producer revision. This reviewer did not edit `LOGBOOK.md` because any repository write would mutate the Change Request candidate under review.
