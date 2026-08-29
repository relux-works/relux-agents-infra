# TASK-260824-1qm60c implementation and validation

## Delivered scope

- Removed the source-managed `[profiles.fast]` table from `.configs/codex-config.toml` while retaining `service_tier = "default"`.
- Removed the `agents-infra codex --print-config --profile fast` example and the README statement advertising the retained fast profile.
- Added source/config documentation regression tests and changed the production `Setup` test fixture/assertions so global setup must install Standard without `[profiles.fast]`.
- Recorded the live-runtime preservation boundary in `LOGBOOK.md`.

## Production call site and regression binding

- `infra.Setup` is driven by `TestSetupGlobalLinksCodexConfig`; the installed global config must contain `service_tier = "default"` and must not contain `[profiles.fast]`.
- `TestSourceManagedCodexConfigUsesStandardTierWithoutFastProfile` parses the repository source TOML and rejects a reintroduced fast profile or non-Standard tier.
- `TestREADMEDoesNotAdvertiseRemovedCodexFastProfile` rejects the withdrawn command/example wording.

## Supported setup synchronization evidence

- Fresh isolated global `setup`, `doctor`, and `verify` all exited 0. The installed managed config contains Standard/default and no fast profile or persistent fast tier.
- Fresh isolated local `setup`, `doctor`, and `verify` all exited 0. The installed managed config contains Standard/default and no fast profile or persistent fast tier.
- The local target was preseeded with user-managed `.codex/config.toml` trust/TUI state and `.agents/.configs/project-config.toml` primary-session policy. Both files remained byte-identical after setup (`shasum -c`, exit 0).
- Live `~/.agents` and the main source-repository local runtime were not synchronized before review because the task requires synchronization after accepted source integration. Their pre-existing config hashes remained unchanged; this avoids replacing user-owned global state or mutating the accepted-source target prematurely.

## Validation results

| Command | Exit | Result |
| --- | ---: | --- |
| Focused infra test, first invocation | 1 | Test process did not start: invalid log redirection path; corrected and rerun. |
| `go test ./internal/infra -run 'TestSetupGlobalLinksCodexConfig\|TestSetupLocalLocalCodexConfigModeRendersProjectSafeCodexConfig' -count=1` | 0 | Passed. |
| `go test . -run 'TestSourceManagedCodexConfigUsesStandardTierWithoutFastProfile\|TestREADMEDoesNotAdvertiseRemovedCodexFastProfile' -count=1` | 0 | Passed. |
| Isolated global setup, first invocation | 1 | Correct safety refusal: destination nested beneath source; moved outside source and reran. |
| Isolated global `setup` / `doctor` / `verify` | 0 / 0 / 0 | Passed. |
| Local worktree setup attempt | 1 | Correct safety refusal: source contains destination `.agents`; reran against an external isolated project. |
| Isolated local `setup` / `doctor` / `verify` | 0 / 0 / 0 | Passed. |
| Isolated global/local managed-config assertions | 0 / 0 | Standard present; fast profile/tier absent. |
| User state checksum preservation | 0 | Trust/TUI and primary-session policy byte-identical. |
| `go test ./... -count=1` | 0 | Passed. |
| `go vet ./...` | 0 | Passed. |
| `go build ./...` | 0 | Passed. |
| `gofmt -l setup_test.go` empty-output assertion | 0 | Clean. |
| `git diff --check` | 0 | Clean. |

## Handoff boundary

The source change is ready for review. After reviewer acceptance and Story integration, the orchestrator should run supported live global and source-repository local setup/doctor/verify while preserving the live user-managed trust/TUI state documented in `LOGBOOK.md`.
