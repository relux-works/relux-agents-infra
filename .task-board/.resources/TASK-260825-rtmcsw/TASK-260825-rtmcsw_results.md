# TASK-260825-rtmcsw — Developer Outcome

## Implementation

- Added `agents-infra model-check --target ENTRYPOINT --prompt TEXT --output-dir DIR` with repeatable `--expect-tool` and `--expect-text` flags plus a bounded `--deadline` override.
- The production call site is `main -> run -> runModelCheck -> infra.RunModelCheck -> infra.RunPi`. The command resolves a configured canonical target and refuses targets that do not resolve to an existing managed local Pi profile.
- The default deadline is 5 minutes, overrides are restricted to 1 millisecond through 30 minutes, and the context bounds runtime readiness as well as the Pi session.
- Reused the existing managed Pi/runtime launcher. Both owned process groups now use bounded signal-to-SIGKILL cleanup, including descendants left after the group leader exits, and expose sanitized lifecycle evidence through `PiRunReport`.
- Persists mode-0600 `events.jsonl`, `stderr.log`, `summary.json`, and `summary.txt` in a new explicit mode-0700 output directory. Existing artifact names are never overwritten.
- Parses JSONL into stable event counts, tool calls/failures, bounded and sanitized final response, expectation outcomes, provider/model/target identity, duration/deadline, process/command outcome, event-stream validity, and cleanup state.
- Stable command exit codes: execution failure `1`, timeout `2`, malformed/partial event stream `3`, expectation failure `4`, failed tool execution `5`.
- Raw provider bytes remain only in protected raw artifacts. Production-entrypoint tests prove that a fixture secret is retained in raw JSONL but absent from stdout, stderr, and sanitized summaries.

## Production-entrypoint coverage

`TestModelCheckProductionEntrypoint` builds and drives the real CLI binary against the existing verified pinned Pi asset and a local OpenAI-compatible fixture runtime. It performs no external model download or network access. Cases cover:

1. Happy path with expected `read` tool and expected final text.
2. Mode-0600 raw and summary artifacts plus sanitized terminal output.
3. A final response exceeding 4096 bytes is bounded only in sanitized evidence while raw JSONL is preserved.
4. Missing expected tool and missing expected text as independent negative gates.
5. Failed tool execution.
6. Malformed JSONL, which is refused rather than treated as absent evidence.
7. Deadline during runtime readiness with runtime process-group cleanup.
8. Deadline during an ignore-TERM tool call with direct `ESRCH` proof for Pi/tool and runtime descendants.

## Validation evidence

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test . -run '^TestModelCheckProductionEntrypoint$' -count=1 -v` | 0 | `.temp/STORY-260825-19dzsf/model-check-production-04.log`; all eight cases pass |
| `go test ./internal/infra -run '^(TestPiLaunchReadinessRefusesMalformedMismatchAndDeadChild\|TestPiLaunchReadinessRetriesOnlyServiceUnavailableAtProductionEntry\|TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry\|TestPiLaunchPiSpawnFailureCleansRuntimeAndReleasesLock\|TestPiLaunchPointOfUseCatalogMutationCleansRuntime\|TestPiLaunchForwardsSignalsThenCleansRuntime\|TestPiLaunchShutdownTimeoutKillsRuntimeGroupAndReleasesLock\|TestPiRuntimeReadinessDoesNotFollowRedirect)$' -count=1 -v` | 0 | `.temp/STORY-260825-19dzsf/pi-lifecycle-01.log` |
| `go test ./... -count=1` | 0 | `.temp/STORY-260825-19dzsf/go-test-03.log`; CLI, attachments, and infra packages pass |
| `go test -race ./internal/infra -run '^(TestPiLaunchForwardsSignalsThenCleansRuntime\|TestPiLaunchShutdownTimeoutKillsRuntimeGroupAndReleasesLock\|TestPiRuntimeReadinessDoesNotFollowRedirect)$' -count=1` | 0 | `.temp/STORY-260825-19dzsf/pi-lifecycle-race-01.log` |
| `go vet ./...` | 0 | `.temp/STORY-260825-19dzsf/go-vet-02.log` |
| `go build ./...` | 0 | `.temp/STORY-260825-19dzsf/go-build-02.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `.temp/STORY-260825-19dzsf/go-build-windows-01.log` |
| repository-wide `gofmt -l` cleanliness check | 0 | `.temp/STORY-260825-19dzsf/gofmt-check-02.log` (empty) |
| `git diff --check` | 0 | Run after the current-tree full suite |

## Red and superseded runs

- The test-first production-entrypoint run exited `1` because the new production symbols did not exist yet: `.temp/STORY-260825-19dzsf/model-check-red-01.log`.
- The first implementation attempt exited `1` because the fixture config used an invalid runtime argv shape; it was corrected before any green claim: `.temp/STORY-260825-19dzsf/model-check-green-attempt-01.log`.
- An intermediate production run exited `1` after tabs were accidentally introduced into the embedded Python fixture; the fixture indentation was corrected and the exact suite was rerun green: `.temp/STORY-260825-19dzsf/model-check-production-01.log`.
- `.temp/STORY-260825-19dzsf/go-test-01.log` contains green package output, but its wrapper did not retain the terminal process status. It is not counted as passing evidence; `go-test-02.log` and the later current-tree `go-test-03.log` both captured real exit code `0`.

## Important finding

The managed launcher previously waited for the Pi leader on normal completion but did not prove that the entire Pi process group had disappeared. The fix and direct descendant-cleanup evidence are recorded in `LOGBOOK.md` under “Bounded Checks Need Process-Group Evidence”.
