# TASK-260829-1qh0ud review verdict

## Verdict

**Changes requested → `to-dev`.** CR `CR-TASK-260829-1qh0ud-1` revision 1 is not accepted.

Review target: base `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`, candidate tree `b4e84702064ead077790cf3d27a844cf4c0a8a51`, repository delta present.

## Finding: status bypass publishes hypothetical recovery as actual admission

Severity: acceptance-blocking correctness defect (AC1 and AC3).

The production status path calls `observeResourceStatus(false)` at `pi_shared_broker_darwin.go:916-919`. Classification applies recovery hysteresis to its local `pressureLatched` argument at `pi_shared_resources.go:171-180` and can therefore produce `healthy` / `admitted`. But the caller explicitly declines to commit that recovered latch at `pi_shared_broker_darwin.go:1040-1046`; it also emits no `resource-recovered` event, so an active pressure-eviction timer is not cancelled.

This creates one wire response with mutually contradictory observable facts: broker state `pressured`, resource state `healthy`, admission `admitted`, while the real broker pressure latch remains true and the pressure drain/eviction path remains armed. A consumer cannot truthfully distinguish pressured from recovered state, and the published admission is not the admission state the broker is enforcing.

Negative-evidence shape: **bypass path around the check**. The acquisition path commits recovery, but the status path reuses classification without committing or faithfully reporting the existing latch. The shipped status production test covers only an initially unlatched `busy` sample; the drain test sets the latch directly and never queries status during recovery.

## Attack evidence

A scratch-only test drove the real `sharedBrokerServer.handleConnection` status entry point; the candidate tree was not modified:

```text
go test ./internal/infra -run '^TestReviewAttackStatusPublishesAdmittedWhilePressureLatchRemains$' -count=1 -v
=== RUN   TestReviewAttackStatusPublishesAdmittedWhilePressureLatchRemains
reviewer_pressure_status_attack_test.go:32: reproduced: wire broker_state=pressured resources=healthy/admitted while internal pressure_latched=true
--- PASS: TestReviewAttackStatusPublishesAdmittedWhilePressureLatchRemains (0.10s)
PASS
```

Scratch source: `.temp/TASK-260829-1qh0ud/attack-module-1/internal/infra/reviewer_pressure_status_attack_test.go`.

The candidate's focused suites were independently rerun uncached and passed:

- focused resource/config/production tests: exit 0, 1.335s;
- focused production resource tests under `-race`: exit 0, 3.106s;
- exact CR validation artifact reports `go test ./... -count=1`, `go vet ./...`: exit 0;
- `git diff --check <base> <candidate>`: exit 0.

Green positive paths do not cover the reproduced status contradiction.

## Required rework

Make status publication and the enforced pressure latch one coherent state model. A status request must not publish recovery/admission that the broker has not applied. Either reconcile recovery atomically with the latch and eviction timer through the broker event/state owner, or report the still-latched pressure/admission until that owner applies recovery. Add a deterministic production-handler negative test starting from a pressure-latched broker with drain/eviction armed, then feed a recovery-threshold observation and prove the status, latch, broker state, timer transition, and subsequent acquisition cannot contradict one another.

The test must drive `handleConnection` status plus the real admission/recovery surface; a helper-only classifier assertion is insufficient.

## Current-trunk composition audit

Published `origin/main` was verified at `675f77ed63376320ed1213f46f9462a299c0abaf` (`Merge pull request #10 from relux-works/codex/infra-restart-status`). Its overlap is semantically compatible but textually adjacent. Rework and final Story integration must preserve all three additive status facts: `RestartNotBefore`, `HalfOpen`, and `Resources`; `applySharedRuntimeLedgerStatus` must keep copying the two persisted restart facts; Darwin and unsupported platform status shapes and README/SKILL documentation must retain both contracts. The trunk advance is not the reason for this verdict and does not require a stop-the-line decision.
