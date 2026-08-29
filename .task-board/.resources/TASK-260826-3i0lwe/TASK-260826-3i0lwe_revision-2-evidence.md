# TASK-260826-3i0lwe revision-2 evidence

## Delta

- Added a readable stdin witness to the existing shared-runtime standalone production test.
- Added an exclusive-runtime standalone production test with a readable stdin witness.
- Added CLI deadline boundary and sanitized refusal coverage for `(0, 30m]`.
- Made the README standalone configuration example show the required managed Pi compatibility prerequisite.
- Recorded the reviewer F1 root cause and repair in `LOGBOOK.md`.

No production guard was removed or weakened. The revision-2 code delta is tests and documentation; the accepted revision-1 production design remains intact.

## Discriminating calibration

| Production call site | Narrowing mutant | Mutant exit | Observed failure | Restored exit |
| --- | --- | ---: | --- | ---: |
| `RunPi` exclusive branch, `pi_launch_posix.go` | Wire caller stdin for the default exclusive standalone shape | `1` | `TestRunPiStandaloneExclusiveWorkerClosesReadableStdin`, `StdinEOF:false` | `0` |
| `runSharedPiSession`, `pi_shared_client_darwin.go` | Wire caller stdin for the shared standalone shape | `1` | `TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`, `StdinEOF:false` | `0` |

Both mutants were applied one at a time, run uncached with `-count=1`, restored by exact `apply_patch`, and followed by a green named-test control.

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused exclusive/shared stdin tests | `0` | `stdin-focused-baseline-01.log` |
| Targeted CLI deadline/refusal tests | `0` | `cli-focused-baseline-01.log` |
| `go test ./... -count=1` | `0` | `go-test-all-01.log` |
| `go vet ./...` | `0` | `go-vet-all-01.log` |
| Native `go build ./...` | `0` | `go-build-native-01.log` |
| Darwin arm64/amd64 cross-builds | `0` / `0` | `go-build-darwin-*-01.log` |
| Linux amd64 / Windows amd64 cross-builds | `0` / `0` | `go-build-*-amd64-01.log` |
| `gofmt` diff gate | `0` | `gofmt-diff-01.log` |
| `git diff --check` | `0` | `git-diff-check-01.log` |

Raw task-scoped logs remain under `.temp/TASK-260826-3i0lwe/` in the Story worktree. The expected-red mutant logs are reported as failures, not passes.
