# TASK-260829-1qh0ud revision 7 independent review verdict

## Verdict

**Accepted.** Change Request `CR-TASK-260829-1qh0ud-7` revision 7 satisfies the task acceptance criteria and closes the revision-6 concurrency findings.

Review target: base commit `6d051f54440d36e3ca3d132f8d9d1e78d46289de`, candidate tree `dabd04a99420aceb21005de65221426bba252c37`, repository delta present, patch SHA-256 `7e377be3bdbe65516820fcfa39cec620f0ca7afed60d1dcb72d8638410d475f5`. The digest, object types, 22 changed paths, and clean `git diff --check` were independently verified before testing. Review and validation ran from an archive of the immutable candidate tree, not from the mutable Story worktree.

## Acceptance evidence

- The production status path `sharedBrokerServer.handleConnection -> observeResourceStatus(false) -> resourceStatusSnapshot` separates diagnostic freshness from lease-admission invalidation and atomically revalidates the status and admission generations with broker state, pressure latch, and leases immediately before constructing the wire response. Superseded facts publish `unknown/refused` with `resource_observation_stale`; draining and the current pressure latch cannot be laundered into healthy availability.
- Direct pressure observed by status still advances admission invalidation. Healthy status polls do not advance it and therefore cannot starve an acquire paused before reservation. The acquire path revalidates the admission generation under the same mutex before adding a lease.
- Provider read failures, malformed or partial facts, unavailable broker observations, pre-extension records, and record-derived unverified policies remain explicit unknown/refused or propagate a read error; none are converted into absence or healthy facts. Unsupported platforms retain the public `resources` field and return the typed `shared_runtime_platform_unsupported` boundary. Darwin, Linux, and Windows builds passed.
- Resource-pressure configuration has no inferred threshold or action defaults: provider mode requires the complete policy table; thresholds are positive with recovery strictly below pressure; timeout and eviction grace are bounded; pressure/unknown/busy actions are pinned. Lease acquisition requires exact configured/effective policy equality before observation or reservation.
- Pressure refusal preserves existing connection-bound leases and the broker-owned runtime PID. Recovery at the configured hysteresis boundary grants on the same runtime. Pressure eviction waits for final lease release, then follows the configured grace/drain sequence. No tested refusal, retry, status race, or recovery path starts a duplicate runtime.
- Consumer handoff is versioned as `agents-infra.pi.shared-runtime.resource-observation.v1` and `agents-infra.pi.shared-runtime.resource-status.v1`; the shared wire protocol is version 7 and both JSON fixtures are present and exercised.

No live model, user-owned runtime, external endpoint, or user-owned socket was contacted or mutated. Production-entry tests used deterministic fake provider observations and task-owned Unix connection pairs.

## Reviewer-owned gate attacks

All attacks were applied only to scratch copies of the candidate; the immutable candidate and Story worktree were not modified.

1. **Final-publication narrowing mutant:** removed only the admission-generation half of `resourceStatusSnapshot` freshness. `TestSharedBrokerProductionPressureCannotSupersedeStatusBeforePublication` failed with exit 1 because the superseded response was no longer explicit `unknown/refused`.
2. **Status/admission re-coupling mutant:** made healthy status observations increment admission generation again. `TestSharedBrokerProductionHealthyStatusPollingCannotStarveAdmission` failed with exit 1 after four polls caused `shared_runtime_resource_unknown` instead of a lease.
3. **Policy-equality narrowing mutant:** compared provider policies only by pressure threshold. `TestSharedBrokerProductionResourcePolicyMismatchRefusesBeforeObservationOrLease` failed with exit 1 and granted leases for recovery-threshold, path, timeout, grace, and action mismatches. This proves the production hello/handleConnection/acquire gate is covered field-independently.

These are production-call-site, narrowed-gate attacks covering the bypass-path shape; they are not helper-only or delete-only checks.

## Independent validation

| Command | Exit | Result |
| --- | ---: | --- |
| Focused 13-test production resource/status slice under `-race -count=1` | 0 | Passed in 3.890s, including both exact revision-6 reviewer schedules, policy mismatches, refusal/recovery, stale ordering, draining, and lease/runtime preservation. |
| `cd tools/agents-infra && go test ./... -count=1` | 0 | All packages passed; root 81.606s, attachments 3.026s, infra 159.498s, modelharness 2.996s. |
| `cd tools/agents-infra && go vet ./...` | 0 | No diagnostics. |
| `cd tools/agents-infra && go build ./...` | 0 | Darwin build passed. |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Unsupported POSIX surface compiled. |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows surface compiled. |
| `gofmt -d internal/infra/*.go *.go` | 0 | Empty output. |
| `git diff --check <base> <candidate>` | 0 | Empty output. |

No acceptance-blocking finding remains. The accepted handoff should be recorded with `accept_cr`; the Orchestrator owns checkpoint/integration and the eventual `done` transition.
