# TASK-260825-rtmcsw — Developer Rework Cycle 3

## Outcome

Resolved every blocking finding from reviewer run `RUN-260825-75ed44` without
changing the accepted managed-runtime design:

- `main` now maps only `*infra.ModelCheckFailure` to the model-check exit-code
  taxonomy. Bare provider `*exec.ExitError` values retain the established CLI
  exit `1` instead of leaking child exits or `-1` into `os.Exit`.
- `processGroupCleanupState` is tested against real OS state: a live owned
  process group reports `failed`; the same group after kill and reap reports
  `confirmed` or `confirmed_after_sigkill` according to the cleanup result.
- The slow-readiness production fixture writes its leader PID before heavy
  Python imports and uses a bounded 2s deadline. The formerly flaky production
  subtest passed six uncached repetitions.
- Explicit `--deadline 0` now refuses before launch; omission still receives
  the CLI's safe 5-minute default.
- `--expect-text` is pinned to the final assistant response by a negative case
  where the expected bytes exist in both the prompt and an earlier assistant
  tool-call turn but not the final response.
- README now states that model-check passes Pi `--approve`, so tool calls run
  unattended in the caller project.

Production call sites under test:

- `main -> run -> runModelCheck -> infra.RunModelCheck -> infra.RunPi`
- `main -> run -> runCodex -> exec.Cmd.Run`
- `RunPi -> processGroupCleanupState -> finishPiRunReport -> RunModelCheck`

No external model or runtime download was used.

## Negative Evidence

| Mutant | Named test | Real exit | Result |
| --- | --- | ---: | --- |
| Force `processGroupCleanupState` to return `confirmed` | `TestProcessGroupCleanupStateReflectsLiveAndReapedGroups` | 1 | Killed on live-group `failed` assertion |
| Restore structural `interface{ ExitCode() int }` matching in `main` | `TestMainKeepsProviderChildFailuresAtLegacyExitOne` | 1 | Killed by observed wrapper exit 42 |
| Restore zero-deadline fallback to the default | `TestModelCheckProductionEntrypoint/zero_deadline_refuses_before_launch` | 1 | Killed by an unexpected successful runtime launch |

Each mutated source was restored with an exact reverse patch and compared with
its task-scoped backup; all three `cmp` commands exited `0`.

## Green Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test -count=6 -run 'TestModelCheckProductionEntrypoint/deadline_override_bounds_runtime_readiness$' .` | 0 | 6/6 uncached readiness-deadline runs, 18.880s |
| `go test -count=1 .` | 0 | Full main package, 80.513s |
| `go test -count=1 ./internal/...` | 0 | Attachments 0.994s; infra 84.971s |
| `go test -race -count=1 -run 'TestProcessGroupCleanupStateReflectsLiveAndReapedGroups\|TestModelCheckCleanupAttestationRefusesUnconfirmedStates\|TestMainKeepsProviderChildFailuresAtLegacyExitOne\|TestModelCheckProductionEntrypoint' . ./internal/infra` | 0 | Main 16.867s; infra 1.464s |
| `go vet ./...` | 0 | No diagnostics |
| `go build ./...` | 0 | Native build |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows cross-build |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux cross-build |
| `gofmt -l .` | 0 | Empty output |
| `git diff --check` | 0 | No whitespace errors |

The pre-fix targeted red run exited `1` for exactly the two reproduced defects:
provider child exit 42 leaked from the wrapper, and explicit deadline zero was
silently replaced by 5 minutes. The final targeted rerun exited `0`.

## Review-Fixture Cleanup Directive

The orchestrator reported that reviewer cycle 2 left eight busy-loop fixture
processes in one process group. The orchestrator terminated the exact PIDs and
verified the group empty. This rework did not reproduce that unbounded load
loop; stability evidence uses bounded ordinary fixtures only. The anomaly and
decision are recorded in `LOGBOOK.md`.

## Files In Scope

- `tools/agents-infra/main.go`
- `tools/agents-infra/model_check_main_test.go`
- `tools/agents-infra/internal/infra/model_check.go`
- `tools/agents-infra/internal/infra/pi_test.go`
- `README.md`
- `LOGBOOK.md`

The broader model-check implementation and managed Pi lifecycle changes remain
the existing task scope documented by the prior cycle outcomes.
