# Change Request CR-TASK-260830-y6infr-1 revision 2

## Manifest

- Base commit: `4270549dd17c010599e2083bf3ec7672af60ea29` (`main`)
- Candidate patch: `TASK-260830-y6infr_CR-2.patch` (binary `git diff`, base -> working tree)
- Patch byte count: 164654
- Patch SHA-256: `9f754f582272efe2f0fa597bf765590621485b1edafe6f1457db4ea71ea664ba`
- `git diff --check <base> -- .`: exit 0
- Reconstruction: a detached worktree at the base commit applied this exact patch (`git apply --check` then `git apply`, both exit 0), built (`go build ./...` exit 0), and passed the full set of revision-1 guard tests plus every new revision-2 test (`go vet`, targeted `go test` runs, all exit 0). The verification worktree was then removed.

## Reviewer entry points for revision 2

- F1 (production wiring + real observation reader): `tools/agents-infra/internal/infra/agents_management_engine_reader.go` (new), `tools/agents-infra/internal/infra/agents_management_registry.go` (`ResolvePiPluginGraph` nil-fallback), `tools/agents-infra/main.go` (`runPiTurnCLI`, `agents-infra pi turn`).
- F2 (real Process-B evidence): `tools/agents-infra/internal/infra/pi_shared_engine_observation_darwin_test.go` (new), test `TestSharedRuntimeEngineObservationReaderReadsRealBrokerAndProcessASpawnsNeverTouchProcessB`.
- F3 (raw Pi translator lifecycle bypass): `tools/agents-infra/internal/infra/pi_turn_result.go` (`sawFinalAssistantMessage`, tool-event lifecycle guards), tests in `pi_turn_result_test.go`.
- Pure fact-mapping unit coverage (no I/O, static/fake-only family): `tools/agents-infra/internal/infra/agents_management_engine_reader_test.go`.
- CLI argument-validation coverage: `tools/agents-infra/pi_turn_cli_test.go`.

## Evidence

See `TASK-260830-y6infr_results-rev2.md` and `TASK-260830-y6infr_mutants-rev2.log`.
