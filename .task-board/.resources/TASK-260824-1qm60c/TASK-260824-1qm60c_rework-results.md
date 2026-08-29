# TASK-260824-1qm60c rework implementation and evidence

## Outcome

- Removed the source-managed `[profiles.fast]` definition while retaining `service_tier = "default"`.
- Removed README examples and text that advertised the withdrawn fast profile.
- Added `Setup -> syncRepo -> syncManagedCodexConfig` migration behavior for global and local managed Codex config copies.
- Source remains authoritative for model, reasoning, and service tier. Existing project trust state, TUI notice acknowledgements, and non-fast custom profiles are merged forward. The withdrawn `profiles.fast` entry is never merged.
- Existing custom project `.codex/config.toml`, project primary-session policy, and local Claude settings remain governed by their existing preservation contracts.
- Malformed installed Codex TOML refuses synchronization without replacing that config.

## Negative evidence

Production call site: `infra.Setup` calls `syncRepo`, which calls `syncManagedCodexConfig` for `.configs/codex-config.toml`.

- `TestSetupGlobalMigratesManagedCodexConfigPreservingUserState` seeds `service_tier = "fast"`, `[profiles.fast]`, a trust marker, TUI notice state, and a custom profile before driving the real global `Setup` entry point. It requires Standard/default, removal of fast, and preservation of the user-owned values.
- `TestSetupLocalPreservesExistingNativeAgentConfigsOnResync` attacks the former local skip bypass through the real local `Setup` entry point and requires source defaults plus preserved trust/TUI/custom-profile/primary-session state.
- `TestSetupGlobalRejectsMalformedExistingCodexConfigWithoutReplacingIt` proves malformed installed state is not treated as absence and the existing config is not replaced.

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| Initial combined gofmt/focused-test attempt from module cwd | 2 | Formatting paths were repository-relative; `go test` did not start. Corrected with formatting from repository root. |
| First focused migration test run | 1 | Tests ran and exposed brittle TOML header assertions plus missing fixture directories; corrected. |
| Focused migration/refusal tests after correction | 0 | Global/local migration and malformed-installed refusal passed. |
| Second combined gofmt/focused-test attempt from module cwd | 2 | Same path/cwd command error; `go test` did not start. Corrected immediately. |
| Doctor-assertion focused test compile | 1 | Test used report wrapper fields as scalars; corrected to `Present`/`Value`. |
| Final focused migration tests | 0 | Global/local migration with doctor/verify assertions passed. |
| `go test ./internal/infra -count=1` | 0 | Package passed in 61.297s before final assertion additions. |
| `go test ./... -count=1` (final) | 0 | Main 56.309s; attachments 1.415s; infra 79.748s. |
| `go vet ./...` (final) | 0 | No findings. |
| `go build ./...` (final) | 0 | Module builds. |
| `gofmt -l` empty-output assertion | 0 | Changed Go files formatted. |
| `git diff --check` | 0 | No whitespace errors. |
| Initial developer handoff | 1 | Board correctly refused unchecked item 16; rev1 review verdict already requested rework and routed the task to development, so the evidenced conditional item was checked before retry. |

## Supported setup smoke

The smoke used isolated runtimes under `/tmp/TASK-260824-1qm60c-*`; live `~/.agents` and the main source-repository runtime were not mutated before accepted integration.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Initial smoke binary build from repository root | 1 | Expected module-location error; rebuilt from `tools/agents-infra`. |
| Candidate binary build from `tools/agents-infra` | 0 | Isolated smoke binary created. |
| Fresh isolated `setup global` | 0 | Candidate source installed. |
| Repeat isolated `setup global` over seeded legacy fast/user state | 0 | Supported synchronization path ran. |
| Global managed-config negative/preservation assertions | 0 | Fast tier/profile absent; Standard, trust, TUI state, and custom profile present. |
| Isolated `doctor global` / `verify global` | 0 / 0 | Runtime reported linked global config and verified. |
| Isolated `setup local` from the installed global source | 0 | Supported local synchronization path ran in preserve mode. |
| Local managed-config negative/preservation assertions | 0 | Fast tier/profile absent; Standard, trust/TUI state, custom project config, and primary-session policy present. |
| Primary-policy/custom-config SHA-256 before vs after | 0 | Both hashes byte-identical: `a24868e...7a91` and `edb638c...ab60`. |
| Isolated `doctor local` / `verify local` | 0 / 0 | Doctor reported preserved primary model/yolo policy; runtime verified. |

## Integration boundary

Per the task ordering, live global and source-repository local runtimes must be synchronized only after reviewer acceptance and source integration. The supported path and preservation behavior are covered by production tests and isolated CLI smoke; the orchestrator retains the accepted-integration step.

## Documentation basis

The official Codex configuration reference identifies `service_tier` as the preferred tier, `projects.<path>.trust_level` as project trust state, and `notice.*` as persisted acknowledgement state. This defines the merge boundary used by the migration.
