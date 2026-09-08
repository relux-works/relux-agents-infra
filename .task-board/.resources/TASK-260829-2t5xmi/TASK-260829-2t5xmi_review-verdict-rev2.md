# TASK-260829-2t5xmi Review Verdict

## Verdict

**Accepted** for Change Request `CR-TASK-260829-2t5xmi-2` revision 2. No review findings remain.

Reviewed exact candidate:

- Base OID: `ee8ae3c7bd4a9eb7e2e015dafa83500c7093542c`
- Candidate tree OID: `c0fa5d86939e7033774aa5b48dc70589d54c1b4d`
- Repository delta: present, 21 paths
- Patch SHA-256: `0b7b4d2f7c85aa73694dc51c2b69b2ac238ee341c85345c60a60de1ac5618121`
- A temporary-index reconstruction of the review worktree produced the same candidate tree OID; `git diff --check` is clean.

Revision 2 is limited to finding F1 from revision 1: 161 inserted/deleted lines across seven files relative to candidate tree `639dd436400f7b013cea1ad2769a23f4b7d70d87`. It does not change numeric defaults or introduce live-model behavior.

## F1 resolution and gate-defeat evidence

The production config gate now rejects seconds above the explicit overflow-safe `time.Duration` bound of `9223372036` seconds. Doubled lease-stale duration is separately bounded at `4611686018` seconds, and the effective linger + shutdown + handoff-grace sum is validated before runtime use. Startup, shutdown, linger, heartbeat, lease-stale, broker-start, restart initial/max backoff, stable-run, and quarantine conversions were enumerated from their production call sites; all are reached through `parsePiRuntime` / `parsePiRuntimeSharing`, and no alternate production constructor bypass was found.

The uncached negative test `TestRunPiRejectsSupervisionSecondsThatOverflowEffectiveDurations` drove the real `RunPi -> loadCompositeProjectConfig -> parsePiRuntime` path. It supplied `max+1` values for every duration field plus overflowing coupled sums and proved refusal before provider lookup and before creation of cache/runtime/ledger state. All 12 negative cases passed. This directly covers the prior `10_000_000_000`-seconds bypass.

The producer's attached revision-2 evidence additionally narrowed the gate from `> maximum` to `> maximum+1`; the same production-path test failed as required with `-count=1`, and the source was restored byte-for-byte. The reviewer remained read-only and independently reran the real negative path rather than mutating repository code.

`sharedRuntimeRecordFailure` now clamps against `maximum-delay` before doubling. `TestSharedRuntimeRestartBackoffClampsBeforeDurationOverflow` passed with a near-maximum policy and a positive, bounded result.

## Reviewer validation

Reviewer-rerun, uncached checks:

- Overflow/config gate: `TestRunPiRejectsSupervisionSecondsThatOverflowEffectiveDurations`, `TestSharedRuntimeRestartBackoffClampsBeforeDurationOverflow`, and strict sharing parser — passed.
- Abrupt-death production seams: real `handleConnection` and `RunPi -> runSharedPiSession` subprocess fixture tests — passed.
- Restart/quarantine preservation: cross-broker automatic restart, persisted/manual quarantine refusal and surfacing, stable reset, failed half-open, persisted ledger, malformed-ledger refusal, bounded policy, and lifecycle status JSON tests — passed.
- Static gates: `go vet ./internal/infra .`, Darwin `go build ./...`, `gofmt -d` on revision-2 Go files, and `git diff --check` — passed/clean.

The tree-bound Change Request validation attached by the producer ran `go test ./... -count=1` (including `internal/infra` in 154.008s) and `go vet ./...`; both exited 0. No live model was invoked, and the abrupt-client tests use static subprocess fixtures.

`LOGBOOK.md` records the duration-overflow regression, fix, production-path negative test, narrowed mutant, and validation evidence as required.
