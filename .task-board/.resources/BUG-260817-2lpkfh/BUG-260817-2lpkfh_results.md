# BUG-260817-2lpkfh — implementation evidence

## Outcome

Managed Pi policy now binds the direct runtime argv to the exact endpoint
declared by `base_url`. A supported profile must contain exactly one spaced
`--host 127.0.0.1` pair and exactly one spaced
`--port <base_url-port>` pair. Missing, duplicate, attached, wildcard, and
divergent forms fail during project-config parsing, before compose diagnostics,
state creation, listener inspection, or process launch.

Production call sites covered:

- in-process CLI entry: `runCompose` -> `BuildPrimarySessionLaunchPlan` ->
  `buildPiPrimarySessionLaunchPlan` -> `parsePiConfig` / `parsePiRuntime`
- installed entry: setup-generated `.local/bin/agents-infra compose --mode
  primary-session --agent pi ...`
- downstream installed entry: `/Users/alexis/src/local-models/.local/bin/agents-infra`

The Qwen and Muse downstream profile validator now asserts one exact host pair
and one exact profile-specific port pair in emitted runtime argv.

## Files changed for this bug

Source repository:

- `tools/agents-infra/internal/infra/pi_config.go`
- `tools/agents-infra/internal/infra/pi_test.go`
- `tools/agents-infra/main_test.go`
- `tools/agents-infra/installed_binary_setup_test.go`
- `README.md`
- `SKILL.md`
- `.research/260817_pi-local-model-launch-contract.md`
- `LOGBOOK.md`

Downstream project:

- `/Users/alexis/src/local-models/.scripts/test-managed-pi-profiles.sh`
- `/Users/alexis/src/local-models/README.md`

## Negative evidence

- Initial focused production/parser regression: exit 1. Both production
  compose mutants were admitted; parser cases also admitted wildcard bind,
  port drift, missing/attached/duplicate endpoint options.
- Same focused regression after the fix: exit 0.
- Port-equality narrowing mutant in the task-scoped copied Go module: exit 1.
  `TestRunComposePiRefusesRuntimeEndpointDivergence/runtime_port_drift` failed
  because production compose admitted the drift. Host validation remained in
  place, so this proves the width of the port binding rather than gate presence.
- Installed downstream wildcard mutant compose: exit 1,
  `invalid_project_configuration`, exact `.runtime.argv` host mismatch.
- Installed downstream port-drift mutant compose: exit 1,
  `invalid_project_configuration`, exact `.runtime.argv` port mismatch.
- Exact loopback production controls pass and preserve the complete literal
  runtime argv.

The first full Pi-scope run after introducing the contract exited 1 because
fake runtime fixtures did not yet express host/port pairs. Fixtures were updated
to retain their test-specific positional arguments while declaring the exact
managed bind. The rerun exited 0.

## Validation

Every command ran directly as a standalone process; exit codes below are the
real command statuses.

| Gate | Exit | Result |
| --- | ---: | --- |
| Focused parser + production compose regression, before fix | 1 | Expected red; wildcard and port drift admitted |
| Focused parser + production compose regression, after fix | 0 | Pass |
| `go test ./... -run Pi -count=1`, first fixture audit | 1 | Expected contract fallout in fake runtime fixtures |
| `go test ./... -run Pi -count=1`, after fixture updates | 0 | Pass |
| Installed local compose regression | 0 | Pass |
| Port-equality narrowing mutant production test | 1 | Expected red; named port-drift test failed |
| `go test ./... -count=1` | 0 | Pass |
| `go vet ./...` | 0 | Pass |
| `go build -o ../../.temp/BUG-260817-2lpkfh/agents-infra .` | 0 | Pass |
| `gofmt -d` on changed Go files | 0 | Empty output |
| `./setup.sh` | 0 | Global binary/runtime installed and verified |
| `agents-infra verify global` | 0 | Pass |
| Source repo `agents-infra setup local` | 0 | Pass |
| Source repo `agents-infra verify local` | 0 | Pass |
| `local-models` `agents-infra setup local` | 0 | Pass |
| `local-models` `agents-infra verify local` | 0 | Pass |
| Qwen `.local/bin/pi-infra --print-config` | 0 | Exact `127.0.0.1:18011` argv/endpoint |
| Muse `.local/bin/pi-infra --print-config --profile muse-glimmer-30b-dflash` | 0 | Exact `127.0.0.1:18012` argv/endpoint |
| `local-models/.scripts/test-managed-pi-profiles.sh` after reinstall | 0 | Pass |
| `zsh -n .scripts/test-managed-pi-profiles.sh` | 0 | Pass |
| Installed downstream wildcard compose mutant | 1 | Expected refusal |
| Installed downstream port-drift compose mutant | 1 | Expected refusal |
| Source `git diff --check` | 0 | Pass |
| `local-models` `git diff --check` | 0 | Pass |
| `task-board validate` | 0 | Pass |

No files were staged or committed. The source checkout and downstream
`local-models` checkout both contained pre-existing story work; it was
preserved.
