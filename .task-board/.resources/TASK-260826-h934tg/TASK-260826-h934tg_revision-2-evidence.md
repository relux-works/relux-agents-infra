# TASK-260826-h934tg — Revision 2 rework evidence

## Scope

Addressed only blocking review finding F1 from Change Request revision 1.
Production behavior, authorization semantics, tool allowlist, and merge ancestry
remain unchanged. The repository delta from revision 1 is exactly:

- `tools/agents-infra/internal/infra/pi_standalone_shared_test.go`: add the
  production-launch witness
  `TestRunPiStandaloneNeverInheritsPrimarySessionProjectTrust`.
- `LOGBOOK.md`: make entry 1411 cite that discriminating witness and M8b.

The witness drives the real `RunPi` entry point through
`pi_launch_posix.go` into `BuildStandalonePiArguments`. Its composed project
policy explicitly has both `[agents.pi.primary_session] yolo_mode = true` and
an authorized `[agents.pi.standalone_session]`. The launched child must receive
exactly one owned `--no-approve` and must not receive `--approve`, `-a`, or
the caller spelling `-na`.

## M8b narrowing calibration

The production files were copied before mutation and their SHA-256 values were
recorded:

- `pi_launch_posix.go`: `57a54d4a8eba086f9b98ccb92bf14a2d673447f8cab841e880a8f754471631e1`
- `pi_standalone.go`: `e9622b77790cb5c0396f446977dcd7e2d5c6507449474d375ebd7f71ae8dc9d4`

M8b applied primary-session yolo before the standalone branch, passed the
resulting args into standalone composition, and narrowed caller-argument
validation to `nil`. The focused witness was rerun with `-count=1` and exited
1 as required:

```text
standalone worker received non-owned approval posture "--approve":
[... "--approve" "--no-approve" ...]
```

Both production files were restored from the pre-mutation copies. `cmp` exited
0 for each, their SHA-256 values returned to the values above, and
`git diff --quiet HEAD -- <production paths>` exited 0. The restored focused
witness then exited 0.

## Direct validation results

Every gate below ran as a standalone process without `tee` or a status-hiding
pipe.

| Command | Exit | Result |
| --- | ---: | --- |
| Initial focused invocation with wrong log path | 1 | Shell failed before Go ran; preserved in `focused-invocation-failure-01.log` |
| `go test ./internal/infra -run '^TestRunPiStandaloneNeverInheritsPrimarySessionProjectTrust$' -count=1 -v` | 0 | Clean pre-mutant witness passed |
| Same focused command under M8b | 1 | Expected red; M8b killed by inherited `--approve` |
| Same focused command after exact restoration | 0 | Restored production passed |
| `go test ./internal/infra -run '^TestRunPiStandalone(NeverInheritsPrimarySessionProjectTrust\|ConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer\|ExclusiveWorkerClosesReadableStdin)$' -count=1 -v` | 0 | All three production-launch witnesses passed |
| `go test ./... -count=1` | 0 | Root `79.768s`, attachments `1.031s`, infra `130.945s` |
| `go vet ./...` | 0 | Configured vet gate passed |
| `go build ./...` | 0 | Native Darwin/arm64 build passed |
| `GOOS=darwin GOARCH=amd64 go build ./...` | 0 | Cross-build passed |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Cross-build passed |
| `GOOS=linux GOARCH=arm64 go build ./...` | 0 | Cross-build passed |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Cross-build passed |
| `gofmt -d tools/agents-infra/internal/infra/pi_standalone_shared_test.go` plus empty-output check | 0 / 0 | Formatting clean |
| `git diff --check` | 0 | Diff hygiene clean |
| exact expected-vs-actual changed-path diff | 0 | Only the test and LOGBOOK changed from revision 1 |
| `git merge-base --is-ancestor main HEAD` | 0 | Current `main` is a real ancestor |

Configured landing commands were read from `task-board.config.json` and both
ran fresh: `cd tools/agents-infra && go test ./... -count=1`, then
`cd tools/agents-infra && go vet ./...`.

## Candidate identity

- Merge commit: `3a52ec762b93149b6db541612f28bf1a6ccef5ed`
- Parent 1, accepted standalone checkpoint:
  `8f81371d93c75552580bb1530281ea5627f429a1`
- Parent 2 and current `main`:
  `fd80bd8e0c1de3f372fd1a7527613a5135762de4`
- Revision-1 tree before this two-file rework:
  `02e41e53790b42bfe5cb7cc5c9e19d622a507035`

Revision 2 is ready for independent review.
