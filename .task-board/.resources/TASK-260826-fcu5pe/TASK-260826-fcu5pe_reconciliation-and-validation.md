# TASK-260826-fcu5pe reconciliation and validation evidence

## Candidate identity

- Story branch: `task-board/story/STORY-260825-1r7z9o`
- Candidate commit: `94752fdfd60b9b94209884a39b31082834a25e42`
- Candidate tree: `0ad11d216421593781e3f7c67977e5a4a5dfa21d`
- Current main: `e70f953969d46e451892d9f16e7401b879910b6b`
- Story/main unique commit count: `21 / 4`
- Worktree: clean after the candidate commit

The managed final-leaf refresh attempted by task-board conflicted only while
replaying `LOGBOOK.md` and was aborted, leaving the accepted branch unchanged.
The assignment forbids a child from switching, rebasing, or merging the managed
Story branch. Reconciliation is therefore a deliberate tree composition: the
two current-main product commits are already carried by `3d203b1` and
`5e3bbe0`, with shared-runtime adaptation in `a94e67e`; current-main board-state
commits are intentionally excluded from the Story repository candidate.

## Overlap audit

The exact path intersection between `b3cb84550a60..main` and
`b3cb84550a60..HEAD` contains 18 files.

### Byte-identical to current main

`git diff --exit-code main HEAD -- <paths>` exited 0 for all 11 paths:

- `SKILL.md`
- `tools/agents-infra/canonical_target_pi_main_test.go`
- `tools/agents-infra/internal/infra/canonical_target.go`
- `tools/agents-infra/internal/infra/canonical_target_pi_test.go`
- `tools/agents-infra/internal/infra/model_check.go`
- `tools/agents-infra/internal/infra/model_check_test.go`
- `tools/agents-infra/internal/infra/pi_platform_windows.go`
- `tools/agents-infra/internal/infra/pi_run_report.go`
- `tools/agents-infra/model_check_docs_test.go`
- `tools/agents-infra/model_check_main_test.go`
- `tools/agents-infra/pi_operator_docs_test.go`

These preserve current-main bounded model-check, canonical Qwen native
thinking, standalone Pi yolo refusal, Windows support, and documentation tests
byte-for-byte.

### Deliberately composed with accepted broker behavior

| Path | Current-main intent retained | Accepted Story behavior added |
| --- | --- | --- |
| `LOGBOOK.md` | Qwen/model-check history is retained | Broker/reviewer history is retained; commit `94752fd` adds the accepted rev2 discriminating mutant evidence |
| `README.md` | Current Qwen and bounded model-check contract remains | Strict `runtime.sharing`, operator commands, and Tools section are added |
| `tools/agents-infra/internal/infra/pi_config.go` | Native thinking validation and `pi_yolo_mode_unsupported` remain | Strict opt-in sharing schema and deep-clone isolation are added |
| `tools/agents-infra/internal/infra/pi_launch_posix.go` | Yolo refusal remains before executable lookup; current context/readiness/cleanup path remains | Shared dispatch occurs only after config, environment, Pi identity, and runtime identity gates; absent/exclusive mode remains on the existing path |
| `tools/agents-infra/internal/infra/pi_plan.go` | Qwen thinking/model generation and current launch plan remain | Non-inspecting shared-runtime diagnostics and broker-owned runtime provenance are added |
| `tools/agents-infra/internal/infra/pi_test.go` | Current-main Qwen/yolo and exclusive-runtime tests remain | Strict sharing parser and side-effect-free print-config tests are added |
| `tools/agents-infra/main.go` | Bounded model-check command and exit semantics remain | `runtime status|stop|broker|runtime-launch` and typed shared-runtime exits are added |

All other accepted shared-runtime production/test files are Story-only additions
and remain unchanged from the accepted rev2 candidate. `RunPi` reaches
`runSharedPiSession` only after existing static identity gates.
`connectAndAttestSharedRuntime` remains the single production call site for the
13-gate client attestation chain and is reused by operator commands.
`RunSharedRuntimeLauncher` reads descriptor 3, then the production tokenizer
feeds `sharedAuthorizationShapeDecision`; it does not evaluate mutants against
self-minted frames. Protocol version remains exactly 6.

No files under a task-board/Pi adapter surface are changed: the repository has
no such source directory in this candidate, and the Story delta is confined to
the agents-infra Pi runtime, its tests, README/SKILL/logbook, and the accepted
research specification. Standalone Pi yolo policy is unchanged.

## Validation

Every command below ran as a standalone foreground process after reconciliation.
No result was piped through `tee`; every recorded exit is the real process exit.

| Command | Exit | Result |
| --- | ---: | --- |
| fail if `gofmt -l .` is non-empty | 0 | no unformatted Go files |
| `git diff --check` | 0 | clean |
| `go test ./internal/infra -run '^(TestSharedRuntime\|TestSharedAuthorization)' -count=1` | 0 | `16.200s` |
| `go test -race ./internal/infra -run '^(TestSharedRuntime\|TestSharedAuthorization)' -count=1` | 0 | `30.204s` |
| focused oracle/calibration/per-mutant/production-entry/reject-all suite | 0 | `2.445s` |
| focused current-main canonical Qwen/yolo/model-check infra suite | 0 | `1.298s` |
| focused current-main Qwen compose/model-check root suite | 0 | `15.675s` |
| `go build ./...` | 0 | Darwin build passes |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux build passes |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows build passes |
| `go vet ./...` | 0 | configured landing vet command passes |
| `go test ./... -count=1` | 0 | root `131.289s`; attachments `2.823s`; infra `217.763s` |

The focused mutant command includes:

- `TestSharedRuntimeAuthorizationShapeOracleDifferential`
- `TestSharedRuntimeAuthorizationMutantCalibrationAndHarnessNegatives`
- `TestSharedRuntimeEveryShapeMutantAdmitsPlainValidFrameAtProductionEntry`
- `TestSharedRuntimeShapeMutantsDriveProductionLauncherGate`
- `TestSharedRuntimeRejectAllProbeReddensAtProductionEntry`

This binds the negative evidence to the production `runtime-launch` call site,
proves narrowing rather than deletion-only, exercises the harness's own
reject-all negative, and keeps the plain-valid control under every mutant.

## Diagnostic history

- `task-board version` was an unsupported readiness probe and exited 1; the
  supported `task-board --help` readiness probe and all required tools then
  passed.
- One shell-quoting attempt to commit the logbook exited 1 before any command
  ran; a second un-staged commit attempt exited 1. The file was then staged by
  exact path and commit `94752fd` succeeded. These were not validation gates and
  changed no candidate bytes beyond the intended logbook entry.

No required validation remains red.
