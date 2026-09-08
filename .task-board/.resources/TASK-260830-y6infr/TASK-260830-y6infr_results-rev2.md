# Revision 2 producer evidence

Closes review findings F1, F2, F3 from `TASK-260830-y6infr_review-verdict.md`
(CR-TASK-260830-y6infr-1 revision 1, base `4270549`, candidate tree `a3f221e`).

## F1 — production wiring and a real (non-test) sanitized observation reader

- Added `SharedRuntimeSanitizedEngineObservationReader` in
  `internal/infra/agents_management_engine_reader.go`: a concrete, non-test
  `SanitizedEngineObservationReader` that performs one bounded
  `SharedRuntimeStatusReport` read (the same read-only production entry
  point `agents-infra runtime status` uses) against agents-infra's own
  Process-B broker, then derives the closed 17-fact set from that live
  status plus the already-resolved `PiProfile`. It never starts, stops,
  signals, or otherwise mutates Process B.
- It is fail-closed by construction: any fact it cannot back with genuine
  data (unmeasured loaded-model memory, unmeasured inference activity, no
  absolute `--model` path in the resolved profile's argv, an unattested or
  not-ready broker) refuses the whole observation rather than inventing a
  value. `resource_pressure_mode=disabled` (a legitimate real configuration)
  therefore genuinely cannot complete an observation today — an honest
  fail-closed outcome, not a bug.
- `ResolvePiPluginGraph` (`agents_management_registry.go`) now fills a nil
  `status`/`observations` argument with real production defaults:
  `localruntime.NewCLIStatusReader()` and the new reader constructed from the
  already-resolved profile. Existing callers that inject fakes are
  unaffected (same signature, same behavior when non-nil).
- `main.go` adds a real, non-test production call site:
  `agents-infra pi turn --prompt TEXT [--deadline D]` (`runPiTurnCLI`)
  resolves the graph and drives `infra.BuildAndRunPiTurn`. Repository-wide
  `grep` confirms `BuildAndRunPiTurn`, `BuildPiPluginGraph`,
  `ResolvePiPluginGraph`, and `NewSanitizedEngineObservationAdapter` each now
  have a non-test caller.
- The existing `TestObservationPlaneCannotReachLiveRuntime` /
  `TestGenericPlanesContainNoIdentityBranch` /
  `TestProcessBLifecycleStaysOwnedByAgentsInfraAndOffTheGenericPlanes` guards
  discover the observation/consumer planes by same-receiver-type call-graph
  traversal from `vendorplugin`'s exact call seeds; the new reader has a
  different receiver type and is reached only through the injected interface
  boundary, so it is correctly excluded from those planes (it is a concrete
  adapter, architecturally the same category as the trusted assembly file,
  not part of the generic launch path) while remaining free to perform real
  I/O. All three guards still pass unmodified.

## F2 — real (fake-backed) Process-B evidence, not a marker file

- `internal/infra/pi_shared_engine_observation_darwin_test.go` (new,
  `//go:build darwin`, intentionally outside the `agents_management_*`/
  `pi_turn_result*` no-live-runtime test family since it is real by design)
  reuses this repo's own existing broker/lease integration harness
  (`startSharedLeaseHelper`, `buildSharedFakeRuntime`,
  `SharedRuntimeStatusReport`) — the same harness the pre-existing
  `pi_shared_integration_test.go` suite already exercises — to start a
  genuine broker holding a genuine lease over a genuine (test-built,
  non-live-model) runtime child process.
- The new real `SharedRuntimeSanitizedEngineObservationReader` is driven
  against that live attested broker and correctly refuses (real negative
  result, not fabricated) because the fixture profile leaves
  `resource_pressure_mode=disabled`.
- `BuildAndRunPiTurn` is then driven twice through the real production
  consumer with a fake Process A (once to success, once cancelled
  mid-flight) while the independently-held broker/lease from the same test
  is still up. Before and after, `SharedRuntimeStatusReport` is read
  directly and asserted byte-for-byte unchanged: same `Broker.State`
  (`serving`), same `Runtime.PID`, same lease count. This proves Process-A's
  cleanup path never reaches, signals, leases, or releases Process B —
  reproduced against real (fake-backed) infrastructure, not inferred from a
  marker file a fake Process-A script wrote to itself.

## F3 — raw Pi translator lifecycle bypass

- `parsePiTurnJSONL` (`pi_turn_result.go`) previously accepted a stream with
  no authoritative assistant `message_end` (empty `finalText` silently
  treated as success) and admitted `tool_execution_start/update/end` events
  after `agent_end` — only `message_*`/`turn_*` events carried the
  `!sawAgentStart || sawAgentEnd` lifecycle guard.
- Fixed: added `sawFinalAssistantMessage`, required at EOF; added the same
  lifecycle guard to all three tool-execution event cases.
- `TestPiTurnTranslatorRefusesMissingAuthoritativeMessageAndPostAgentTools`
  (new) directly reproduces both admitted shapes from the reviewer's
  narrowed-mutant finding and fails on the pre-fix code (see mutants log).

## Negative proof (narrowing mutants, all killed)

See `TASK-260830-y6infr_mutants-rev2.log` for full output. Summary:

1. Removing the missing-authoritative-assistant-message guard: admitted the
   attack; `TestPiTurnTranslatorRefusesMissingAuthoritativeMessageAndPostAgentTools`
   failed as expected, confirming the guard (not the test) is load-bearing.
2. Narrowing the lifecycle guard to leave `tool_execution_start`/`end`
   unchecked (message guards intact): admitted a post-`agent_end` tool
   lifecycle; the same test failed as expected.
3. Admitting unmeasured loaded-model memory as `resident-bytes:0`:
   `TestSharedRuntimeEngineFactsRefuseWhenMemoryIsNotMeasured` failed as
   expected.
4. Collapsing `sharedRuntimeAttestedAndReady` to always return true:
   `TestSharedRuntimeAttestedAndReadyRefusesEveryUntrustedShape` failed on
   3 of 4 subtests as expected.

Each mutant was applied, the exact narrow test was run and observed red,
then the source was restored and the build reverified green.

## Validation (real exit codes)

- `gofmt -l .`: zero output (fixed one formatting drift in the new reader
  file before this run).
- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `go test ./internal/infra -count=1`: exit 0 (183.4s).
- `go test . -count=1` (module `main` package): exit 0 (90.9s).
- `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`: exit 0, every
  package (`.`, `cmd/model-harness`, `internal/attachments`,
  `internal/infra`, `internal/modelharness`).
- `go test ./internal/infra -race -run '<every new/changed test family>'
  -count=1`: exit 0 (15.7s).
- `go test . -race -run 'TestRunPiTurnCLI|TestRunPiDispatchesTurn|TestPiTurnSchema' -count=1`:
  exit 0.
- Cross-platform `go build ./...` and `go vet ./...`: darwin/amd64,
  darwin/arm64, linux/amd64, linux/arm64, windows/amd64 — all exit 0.
- `go mod verify`: `all modules verified`.
- No Makefile exists in this repository (confirmed again); no `make`
  targets were skipped.
- No live runtime, model, external process, service, socket, network
  endpoint, or user HOME/configuration outside this repository's own
  bounded test fixtures was contacted. The darwin integration test's
  "runtime" is a test-built Go HTTP stub serving `/v1/models`, launched only
  under this test's own temp `HOME`/project directories.

## Not run, stated plainly

- Independent review, PR, and merge gates belong to the review and delivery
  roles.
- The stable upstream `skill-agents-management` release tag and its final
  pin remain out of scope for this checkpoint (AC7); `TASK-260830-u8nd0b`
  owns that.
- `agents-infra pi turn` was not exercised against a real MLX/local model
  process (no live runtime access is permitted in this task's scope); its
  argument-validation and dispatch-routing paths are covered by
  `pi_turn_cli_test.go`, and its full resolve/observe/launch path is covered
  end-to-end by the darwin integration test using fakes throughout.
