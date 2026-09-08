# TASK-260829-1qh0ud revision 7 review rework

Start from the immutable revision 6 candidate on its protected Story workspace.
Do not discard the accepted revision-6 provenance and policy-equality fixes.

## Acceptance-blocking findings

1. A status observation can complete `healthy/admitted`, then a newer acquire
   can latch pressure before the status response snapshots and publishes state.
   The real wire response can therefore report `healthy/admitted` while
   `pressure_latched=true`.
2. Status and acquire observations advance the same admission invalidation
   generation. Repeated healthy status polling can supersede every healthy
   acquire at the pre-reservation boundary; the reviewer reproduced four
   refusals from four polls and zero leases.

The exact verdict and attack logs are attached under `revisions/6/`.

## Required implementation

- Make status observation, broker/latch state, and wire publication one coherent
  decision. Immediately before send, atomically prove the observation and
  broker snapshot are current together. On supersession, use a bounded retry or
  an explicit truthful unknown/refused status; never publish stale
  `healthy/admitted`.
- Separate diagnostic observation freshness from lease-admission invalidation,
  or implement a pinned bounded fairness/retry contract that proves status
  polling cannot indefinitely deny a healthy lease. Pressure observations must
  still invalidate unsafe admission before reservation.
- Preserve revision-6 record-derived provenance, full independent sharing-policy
  equality, restart facts, single runtime ownership, drain/eviction semantics,
  and unsupported-platform shapes.
- Add deterministic production `handleConnection` regressions for both exact
  reviewer schedules. Helper-only classifier tests are insufficient.
- Add expected-red mutants that remove the final publication revalidation and
  that re-couple status polling to admission invalidation.
- Record both concurrency findings and the chosen state/generation contract in
  `LOGBOOK.md` without erasing unrelated existing entries.

## Validation and handoff

Prove attacks red before the fix and green after it, then run focused production
tests under `-race`, `go test ./... -count=1`, `go vet ./...`, `go build ./...`,
gofmt/diff integrity, and Darwin/Linux/Windows compile gates. Publish a new
immutable Change Request revision and retain all prior evidence.

Do not contact or mutate a live model runtime, service, socket, endpoint, or
user-owned model state.
