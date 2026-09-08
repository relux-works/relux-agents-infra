# Review verdict: CR-TASK-260830-y6infr-1 revision 1

## Verdict

CHANGES REQUESTED. Route `TASK-260830-y6infr` to `to-dev`.

Reviewed base `4270549dd17c010599e2083bf3ec7672af60ea29`, candidate tree `a3f221e21d7cec360431f3adb67dbc56923ab789`, and the exact accepted upstream dependency pseudo-version `v0.5.1-0.20260830114459-046baef11790`. The supplied upstream API patch digest matches `471ebb6ce1b3805bcc8213248ebf0f8ac3c67c7ee215eb279c2f4490c76d96cb`.

## Blocking findings

### F1 — Production consumer and observer graph are not wired to production

`tools/agents-infra/internal/infra/agents_management_process_a.go:28` declares `BuildAndRunPiTurn`, and `agents_management_registry.go:32` declares `BuildPiPluginGraph`, but repository-wide caller search finds both only in `_test.go`. `ResolvePiPluginGraph` and `NewSanitizedEngineObservationAdapter` likewise have no non-test caller. README explicitly says the boundary "is not wired to a CLI verb yet".

The observation side also stops at an injected `SanitizedEngineObservationReader` interface. No non-test type implements `ReadSanitizedEngineObservation`; all evidence is produced by test readers that copy the authoritative query and mint good facts. Consequently the candidate does not inject a concrete versioned sanitized observation at a real agents-infra assembly/call site and does not prove that a bounded underlying read happens before child effects.

Negative shapes: **check present but uncalled from production**, **forged or self-minted evidence**, and **absent evidence treated as satisfied by calling the helper directly**. This leaves AC2 and AC3 unproven and the Description's concrete adapter/Process-A bridge unconsumed.

Required rework: wire the graph and `BuildAndRunPiTurn` from the actual agents-infra production assembly/child-launch path; provide the concrete bounded sanitized observation reader owned by agents-infra; make tests obtain the result through that real entry point; prove one observer read before preflight/child effects and exactly one `pi.ValidateTurnResult` call before persistence/success.

### F2 — The claimed fake Process-B composition is only a marker written by fake Process A

`TestProcessAConsumerUsesSoleClassifierAtRealChildLaunch` creates one shell script named `agents-infra`; that same Process-A script writes `FAKE_PROCESS_B_MARKER` and prints a success document. No fake broker, lease, readiness, Process-B child, lifecycle seam, or release path participates. The assertion that the marker proves "fake Process-B lifecycle was composed" therefore does not reproduce the claimed capability.

Negative shape: **capability claim that does not reproduce** / positive-path-only evidence. AC3 and AC4 require a real agents-infra child-launch call site with fake Process A and fake Process B, including proof that Process-A cleanup never directly signals Process B and only releases its lease.

Required rework: exercise the actual agents-infra-owned fake Process-B lifecycle/broker boundary and the real Process-A launch entry together, then attack cancellation, cleanup, lease release, restart/quarantine ownership, and direct-Process-B-signal bypasses.

### F3 — Raw Pi translator admits invalid/missing lifecycle evidence

`parsePiTurnJSONL` requires only session, `agent_start`, `agent_end`, and no open tool IDs at EOF (`pi_turn_result.go:162-253`). It does not require an authoritative assistant `message_end`. Its tool start/update/end cases (`:231-245`) also omit the lifecycle guard used by message/turn events, so a complete tool lifecycle after `agent_end` is accepted.

A reviewer-only Go overlay, stored under ignored `.temp/`, drove the actual parser and failed as intended:

```text
missing authoritative assistant message admitted as success: text="" toolFailed=false
post-agent tool lifecycle admitted as success: text="accepted" toolFailed=false
```

Command: `go test -overlay ../../.temp/TASK-260830-y6infr-review/overlay.json ./internal/infra -run '^TestReviewerAttack' -count=1` (exit 1 because both invalid streams were admitted). This is a narrowed **bypass path around the check** and violates the required missing-event/lifecycle attack matrix.

Required rework: enforce the pinned Pi v0.84.2 lifecycle grammar, require the final authoritative assistant message for success, reject tool events outside the active agent lifecycle, and add production-translation tests that fail under each narrowed mutant.

## Verification performed

- `git diff --check <base> <candidate>`: pass.
- Upstream patch SHA-256: exact match.
- `go list -m -json github.com/relux-works/skill-agents-management`: exact immutable pseudo-version and commit suffix.
- `go test ./... -count=1`: pass; `internal/infra` completed in 155.494s.
- Focused adversarial/contract suite: pass.
- Focused `go test -race`: pass.
- `go vet ./...`: pass.
- Native `go build ./...`: pass.
- `GOOS=linux GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go build ./...`: pass.
- Changed Go files `gofmt -l`: zero output.
- Initial cross-platform `go test -run '^$'` attempts tried to execute foreign binaries and produced expected `exec format error`; they were corrected to cross-platform compile-only `go build` commands above.
- No live runtime, model, service, socket, endpoint, user configuration, or user HOME was contacted.

The green shipped suite does not override F1–F3 because it lacks a production caller, uses self-minted observation/fake marker evidence, and admits the two narrowed raw-protocol attacks.
