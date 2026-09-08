# TASK-260829-1qh0ud independent revision 7 review verdict

## Verdict

**Accepted.** Immutable Change Request `CR-TASK-260829-1qh0ud-7`, revision 7,
satisfies the task acceptance criteria and the revision-7 concurrency rework.
This run found no acceptance-blocking defect.

The authoritative board already reported this revision as accepted by an
earlier reviewer run when this independent rerun started. This verdict does not
infer acceptance from that state or from producer summaries: every result below
was gathered independently by `RUN-260829-a97d98` from the immutable candidate.

## Exact review identity

- Base OID: `6d051f54440d36e3ca3d132f8d9d1e78d46289de`
- Candidate tree OID: `dabd04a99420aceb21005de65221426bba252c37`
- Repository delta: `present`, 22 changed paths
- Patch: `TASK-260829-1qh0ud_change-request_rev7.patch`
- Patch SHA-256: `7e377be3bdbe65516820fcfa39cec620f0ca7afed60d1dcb72d8638410d475f5`
- Review time: `2026-08-30T02:05:52+03:00`

The patch digest matched the board resource. `git diff --check <base> <tree>`
was empty. The candidate was exported directly from the candidate tree into a
task-scoped disposable directory; the exported resource implementation digest
matched `git show <tree>:<path>`.

## Review findings

No blocking findings.

The production state model now has separate diagnostic and admission
generations. A healthy/busy status poll cannot invalidate a pending healthy
lease, while direct pressure evidence still invalidates it before reservation.
The status handler revalidates status and admission freshness while snapshotting
the pressure latch, broker state, and leases under the same mutex immediately
before the response is built. Superseded facts are published as explicit
`unknown/refused`, never as healthy admission.

The remaining contract surfaces compose coherently:

- record-derived status uses persisted effective sharing for resource-policy
  provenance, while pre-extension or partial records report explicit unknown;
- caller/effective resource policy equality is checked before provider I/O or
  lease reservation and covers every independently variable policy field;
- pressure refuses only new leases, preserves existing connection-bound leases
  and the single runtime owner, then starts the configured eviction grace only
  after the final lease releases;
- recovery uses the same broker/runtime, clears the latch through admission,
  and cancels the pressure eviction sequence;
- provider reads are bounded, strict, model-bound, and versioned; status and
  observation consumer fixtures are checked in;
- Darwin publishes the typed resource status, unsupported POSIX and Windows
  shapes compile with the additive field, and protocol version 7 prevents an
  old broker wire shape from being mistaken for the new contract;
- provider policy has no implicit safe-looking default: mode is required,
  provider mode requires the complete table, disabled mode forbids the table,
  and action values are pinned.

No live model, user-owned runtime, external endpoint, or user-owned socket was
contacted or mutated. Production-path tests use deterministic fake provider
observations and task-owned local Unix connection pairs.

## Independent positive validation

All commands ran against the exported immutable candidate, not the mutable
producer worktree.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused 13-test production resource/status slice under `-race -count=1` | 0 | `internal/infra` passed in `4.391s` |
| `go test ./... -count=1` | 0 | root `266.083s`; attachments `4.271s`; infra `521.116s`; modelharness `2.381s`; command package had no tests |
| `go vet ./...` | 0 | no diagnostics |
| `go build ./...` | 0 | Darwin arm64 |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | unsupported POSIX shape compiled |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows shape compiled |
| `gofmt -d internal/infra/*.go *.go` | 0 | empty output |
| `git diff --check <base> <tree>` | 0 | empty output |

## Independent gate-defeat evidence

Each mutant changed only a disposable candidate copy and was rerun uncached
through the real production handler tests.

| Narrowed mutant | Test | Exit | Defeat observed |
| --- | --- | ---: | --- |
| Final publication freshness ignored admission-generation changes | `TestSharedBrokerProductionPressureCannotSupersedeStatusBeforePublication` | 1 expected-red | Superseded status no longer produced the required explicit stale `unknown/refused` response |
| Every diagnostic status poll advanced admission freshness again | `TestSharedBrokerProductionHealthyStatusPollingCannotStarveAdmission` | 1 expected-red | Four healthy polls starved the pending acquire, which returned `shared_runtime_resource_unknown` |
| Policy equality compared only `pressure_threshold_bytes` | `TestSharedBrokerProductionResourcePolicyMismatchRefusesBeforeObservationOrLease/recovery_threshold_only_differs` | 1 expected-red | The mismatch gate admitted the caller and granted the forbidden lease |

These attacks cover the production call sites
`sharedBrokerServer.handleConnection -> acquireLease` and
`sharedBrokerServer.handleConnection(status) -> resourceStatusSnapshot`. They
demonstrate narrowed-gate failures, not delete-only reachability.

## Acceptance-criteria mapping

1. Healthy, busy, pressured, draining, and unknown are typed and source-aware;
   stale/unreadable facts fail closed without proxy inference.
2. Thresholds, hysteresis, provider timeout, busy/unknown actions, pressure
   refusal, and drain/eviction grace are explicit required configuration.
3. Fake-provider production fixtures deterministically prove pressure refusal,
   stale ordering, recovery, and polling fairness.
4. Existing leases and the runtime PID survive pressure; no duplicate runtime
   or lease ownership is created by retry/recovery.
5. Wire protocol 7 plus observation/status v1 fixtures provide the versioned
   consumer handoff.

## Board lifecycle result

This run called the required named acceptance mutation with its own newly
attached evidence. The board refused only because the same immutable revision
was already persisted as `accepted` by reviewer run `RUN-260829-a159a9` before
this rerun started:

`change_request_state_conflict: revision 7 is accepted, which does not admit a transition to accepted`

No alternate identity or retry was used. The element remains at the existing
accepted handoff `to-review`; this reviewer supplied no `commit_ack` and made no
`done` transition. The Orchestrator retains checkpoint/integration ownership.
