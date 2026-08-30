# Review verdict: CR-TASK-260830-y6infr-3 revision 3

## Verdict

ACCEPTED via `accept_cr`.

Reviewed base `4270549dd17c010599e2083bf3ec7672af60ea29`, candidate tree
`16fc3dc61ba89edbc88dbf5cc236bd011ab0c151`, patch SHA-256
`2a89bafb752b5103f33329d4e77a386255c696d1f13a732397f3f1a71d651702`. A
temporary alternate-index reconstruction (`git read-tree HEAD && git add -A &&
git write-tree` under a scratch `GIT_INDEX_FILE`) of the worktree produced the
exact candidate tree.

## F1 — Pi JSONL turn/message lifecycle (independently attacked, not just read)

Wrote my own reviewer-only overlay test (not the shipped fixtures) driving the
real `parsePiTurnJSONL` production entry directly via `go test -overlay` with
8 malformed streams, including all cases named in the spawn brief: missing
`turn_start`, missing `message_start`, duplicate `turn_start` while open,
`message_end` without `message_start`, `turn_end` without `turn_start`,
reordered end-before-start, interleaved second `turn_start` mid-turn, and a
trailing `message_update` after `turn_end`. All 8 were refused (test exit 0 —
every subtest's negative assertion passed). This independently reproduces and
extends the rev2-reviewer's three original attacks against the current code.

## F2 — Fake Process A composed with the real broker-backed observation reader

Read `agents_management_registry.go` and `agents_management_observer.go`: `BuildPiPluginGraph`
wraps the caller-supplied `SanitizedEngineObservationReader` in
`NewSanitizedEngineObservationAdapter` and registers it via
`vendorplugin.NewRegistryWithEngineObservationAdapters` — the ADR's
registry-construction trust seam. `ObserveEngine` calls
`a.reader.ReadSanitizedEngineObservation` directly; no fake/recording reader
sits between it and the injected reader.

Ran `TestSharedRuntimeEngineObservationReaderReadsRealBrokerAndProcessASpawnsNeverTouchProcessB`
myself on darwin/arm64 (real hardware, not CI-assumed): it passes the real
`SharedRuntimeSanitizedEngineObservationReader` — bound to a genuine
(fake-backed) broker established earlier in the same test — directly into
`BuildPiPluginGraph`, then drives `BuildAndRunPiTurn` twice (success and
mid-flight cancellation) through the exact production graph. A second,
independent, real shared-runtime lease (via the production
`acquireSharedRuntimeLease` client) proves Process-A's own lease lifecycle
survives alongside the untouched Process-B peer lease and runtime PID, both
before and after release. Exit 0.

`BuildAndRunPiTurn` (`agents_management_process_a.go:28`) calls
`vendorplugin.BuildLaunch` then `managementpi.ValidateTurnResult(input)`
exactly once — confirmed by grep (single production call site) and by the
shipped `TestConsumerPlaneCannotParseAroundSoleClassifier`/
`TestProcessAConsumerCannotParseAroundSoleClassifier`, both rerun and passing.
Cleanup (`agents_management_process_a_posix.go`) sets `Setpgid: true` on
Process A and signals only `-command.Process.Pid` (its own process group) on
cancellation — never Process B, consistent with the darwin test's untouched
peer-lease assertion.

Attacked the stale/forged/conflicting/unsupported-observation matrix by
rerunning `TestPiPluginGraphBuildLaunchRefusesForgedObservationBeforePreflight`
(wrong contract, stale `ValidUntil`, profile/engine identity drift, malformed
missing fact, unsupported fact) against the real `vendorplugin.BuildLaunch`:
all six refuse with the exact typed sentinel, observation called exactly once,
preflight called zero times. Exit 0.

## Specific claim challenged: mutants.sh exit-code anomaly

Copied the exact shipped `TASK-260830-y6infr_mutants.sh` resource into the
worktree, snapshotted a shasum of every `.go` file under
`internal/infra` before running, then executed the real script unmodified. It
reproduced `TASK-260830-y6infr_mutants-rev3.log` **byte-for-byte** (`diff`
exit 0) — all 18/18 `MUTANT_KILLED` with real `exit=1`, one
`DISCOVERY_NARROWING_ADMITTED` with real `exit=0` as intended. Post-run
shasums of every file were identical to the pre-run snapshot, confirming the
script's own `restore()` left no residue. This independently confirms the
producer's claim: the current script is correct and the rev2 anomalous log
was not produced by this script.

## Baseline (all rerun independently, not accepted from the record)

- `git diff --check <base> <candidate>`: exit 0.
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- `gofmt -l .`: zero output.
- `go list -m -json github.com/relux-works/skill-agents-management`: exact
  pseudo-version `v0.5.1-0.20260830114459-046baef11790`, no `replace`
  directive in `go.mod`.
- Full uncached `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`: exit
  0, every package (`.`, `cmd/model-harness`, `internal/attachments`,
  `internal/infra` 155.167s, `internal/modelharness`).
- `go test ./internal/infra -race -count=1 -run
  'TestPiTurnTranslator|TestSharedRuntimeEngineObservationReader|TestConsumer|TestPiPluginGraph|TestObservationPlane|TestGenericPlanes|TestProcessBLifecycle'`:
  exit 0.
- Cross-builds `GOOS=linux GOARCH=amd64`, `GOOS=windows GOARCH=amd64`,
  `GOOS=darwin GOARCH=amd64` `go build ./...`: exit 0 each.
- Reran the AST-based boundary/identity guards
  (`TestObservationPlaneCannotReachLiveRuntime`,
  `TestConsumerPlaneCannotParseAroundSoleClassifier`,
  `TestGenericPlanesContainNoIdentityBranch`,
  `TestConcreteAssemblyDeclaresIdentityWithoutDispatchingOnIt`,
  `TestProcessBLifecycleStaysOwnedByAgentsInfraAndOffTheGenericPlanes`,
  `TestContractTestsContactNoLiveRuntime`,
  `TestQwenInfraResolvesToPiAndLocalModelsNeverShippedQwen`,
  `TestPlanIsMetamorphicUnderIdentityRenaming`,
  `TestDryRunPlanIsObservationAndPreflightFree`): all pass. These use AST
  reachability (not a fixed filename list) and were themselves the target of
  mutant 1/2 above (called-helper-in-new-file bypass), which I reproduced as
  killed.
- Confirmed `agents-infra pi turn` is a real, dispatch-wired CLI verb
  (`main.go:485-486`) and `agents-infra pi spawn --result-schema 1` installs
  the schema-1 writer before all later validation (`main.go:578-603`),
  closing rev1's F1 "not wired to a CLI verb" finding.
- Spot-checked `pi_environment.go`'s environment denial list against ADR
  §5.1: exact match (`HF_ENDPOINT`, `MODEL_ENDPOINT`, `GGML_BACKEND_PATH`,
  `LLAMA_API_KEY`, and prefixes `DYLD_`/`LD_`/`NODE_`/`BUN_`/`LLAMA_ARG_`).
- No live runtime, model, external service, production socket, or user
  configuration was contacted during review.

## Conclusion

Both rev2 blocking findings are closed with evidence I reproduced myself
against the real production entry points, not the shipped test summaries. The
disputed mutants.sh exit-code claim is settled: the script and log are
correct as shipped. No new blocking finding found after actively attempting
to defeat the lifecycle guard, the composition seam, and the boundary/no-live
guards.
