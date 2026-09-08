# BUG-260818-1s1lka rework results — RUN-260830-b6b5f9

## Scope

Additive test-fixture coverage only. Production `ValidatePiExecutionEnvironment` at `tools/agents-infra/internal/infra/pi_catalog.go:295` remains unchanged.

- Added `llama_api_key=case-sensitive-lookalike` to `TestPiExecutionEnvironmentAcceptsExactCleanEnvironment`.
- Added the same control to the production `RunPi` clean lifecycle fixture.
- Added the same control to the shared installed bootstrap-global/project-local launcher clean fixture.
- Preserved `LLAMA_API_KEY_SUFFIX`, `HF_TOKEN`, Hugging Face cache variables, `UNRELATED_SERVICE_API_KEY`, and `GGML_METAL_PATH` admitted controls.

## EqualFold broadening attack

Before mutation, the changed focused production fixtures and installed launcher suite both exited 0 with `-count=1`.

Temporarily replaced only the exact `name == "LLAMA_API_KEY"` comparison at `pi_catalog.go:295` with `strings.EqualFold(name, "LLAMA_API_KEY")`.

| Surface | Command scope | Exit | Observed failure |
| --- | --- | ---: | --- |
| Helper | `TestPiExecutionEnvironmentAcceptsExactCleanEnvironment` | 1 | Lowercase `llama_api_key` rejected |
| Production entry | `TestPiLaunchCleanEnvironmentReachesRuntimeBackendInitializationAndPreservesGlobalState` through `RunPi` | 1 | Refused before runtime initialization |
| Installed bootstrap-global alias | installed launcher suite clean control | 1 (suite) | Runtime marker absent; lowercase lookalike denied |
| Installed project-local wrapper | installed launcher suite clean control | 1 (suite) | Runtime marker absent; lowercase lookalike denied |

These are expected-red mutation results and are reported as failures, not passing gates. Logs: `mutant-helper.log`, `mutant-production-runpi.log`, and `mutant-installed-global-local.log`.

The production file was restored through the inverse patch and compared with the pre-mutation copy. Both SHA-256 values are `7decb2c7ab27637cab0bfa1b0941cbfa458d05befb0012bddfedf01ec117fb19`; `cmp` exited 0.

## Restored-source validation

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Focused helper + production `RunPi` controls after restore, uncached | 0 | `pristine-focused-infra-after-mutant.log` |
| Installed bootstrap-global + project-local launcher suite after restore, uncached | 0 | `pristine-installed-after-mutant.log` |
| Full environment refusal/admission `RunPi` focus | 0 | `final-focused-runpi.log` |
| Installed launcher + README/SKILL operator docs focus | 0 | `final-installed-and-docs.log` |
| `go test -p 1 ./... -count=1` | 0 | `go-test-all-serial.log`; root 122.478s, infra 338.586s, all packages green |
| `go vet ./...` | 0 | `go-vet.log` |
| `go build ./...` | 0 | `go-build.log` |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | `gofmt-list.log` is empty |
| `git diff --check HEAD` | 0 | `git-diff-check.log` |

The installed launcher test builds the real CLI, runs isolated `setup global` and `setup local`, and then drives both generated `pi-infra` surfaces. The canonical host `setup.sh` / installed-runtime verify flow was not rerun for this additive test-only delta because it would install unrelated sibling Story changes from the shared worktree; the earlier attached implementation/review evidence already records those commands at exit 0.

## Repository state

Task delta is limited to:

- `tools/agents-infra/internal/infra/pi_test.go`
- `tools/agents-infra/installed_binary_setup_test.go`

Pre-existing `MM` changes in `LOGBOOK.md`, `pi_shared_launcher_darwin.go`, `pi_shared_launcher_test.go`, and `runtime_main_darwin_test.go` belong to sibling Story work and were preserved. The existing `LOGBOOK.md` entry `2026-08-29 0205 — Exact-Name Gates Need A Case Control, Not Just A Suffix Control` already records this regression, so no duplicate logbook entry was added.

One readiness attempt exited 1 before invoking Go because its log redirection used an incorrect relative path. It is preserved in `tool-readiness-failure.log`; corrected `go version` exited 0 (`go1.25.5 darwin/arm64`).
