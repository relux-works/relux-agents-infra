# Change Request CR-TASK-260830-y6infr-3 revision 3

## Manifest

- Base commit: `4270549dd17c010599e2083bf3ec7672af60ea29` (`main`)
- Candidate tree (reconstructed): `16fc3dc61ba89edbc88dbf5cc236bd011ab0c151`
- Candidate patch: `TASK-260830-y6infr_change-request_rev3.patch` (binary `git diff`, base -> working tree)
- Patch byte count: 178058
- Patch SHA-256: `2a89bafb752b5103f33329d4e77a386255c696d1f13a732397f3f1a71d651702`
- Round-trip reconstruction: `git read-tree <base>` into a scratch index, `git apply --cached` the exact patch (exit 0), `git write-tree` — reproduces `16fc3dc6...` byte-for-byte against the tree written from the actual working tree (scratch index seeded from the real worktree index so pre-existing gitignored-but-tracked `.task-board/.resources/**/*.log` blobs are preserved, then `git add -A` to capture the current disk state). Both trees are identical.
- `git diff --check <base> -- .`: exit 0.

## Revision 3 closes CR-TASK-260830-y6infr-2 revision 2's two blocking findings

- F1 (Pi JSONL turn/message lifecycle bypassable): `tools/agents-infra/internal/infra/pi_turn_result.go`, `parsePiTurnJSONL` — see `TASK-260830-y6infr_results-rev3.md` for the exact state-machine change and reproduced attacks.
- F2 (fake Process A / fake-backed Process B not composed in one production graph): `tools/agents-infra/internal/infra/pi_shared_engine_observation_darwin_test.go`, `TestSharedRuntimeEngineObservationReaderReadsRealBrokerAndProcessASpawnsNeverTouchProcessB` — rewritten to drive `BuildPiPluginGraph`/`BuildAndRunPiTurn` with the real `SharedRuntimeSanitizedEngineObservationReader` bound to the same fake-backed broker, alongside a real, independently-acquired second shared-runtime lease representing Process A's own lease lifecycle.
- Evidence anomaly (mutant log falsely printed exit=0): reproduced, root-caused, and corrected — see below.

## Reviewer entry points for revision 3

- F1: `tools/agents-infra/internal/infra/pi_turn_result.go:190-276` (`parsePiTurnJSONL`, `turnOpen`/`messageOpen` state), `tools/agents-infra/internal/infra/pi_turn_result_test.go` (`TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations`).
- F2: `tools/agents-infra/internal/infra/pi_shared_engine_observation_darwin_test.go` (rewritten test), `tools/agents-infra/internal/infra/pi_shared_integration_test.go` (`/agents-infra/resources` handler added to the shared fake-runtime source template so the real reader can observe a genuine `provider`-mode resource fact).
- Evidence anomaly: `.temp/TASK-260830-y6infr/mutants.sh` (unchanged `run_mutant` exit-code capture, reproduced correct on this run), `TASK-260830-y6infr_mutants-rev3.log`.

## Evidence

See `TASK-260830-y6infr_results-rev3.md`, `TASK-260830-y6infr_mutants-rev3.log`, and `TASK-260830-y6infr_change-request_rev3-validation.log`.
