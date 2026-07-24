# TASK-260721-23pal4 Validation Results

## Delivered scope

- `.configs/codex-config.toml` sets `service_tier = "default"` and leaves `[profiles.fast]` unchanged.
- `README.md` now says Standard is the default; the retained `fast` profile is for explicit model/reasoning selection; `/fast on` is the interactive Fast opt-in; persistent Fast requires `service_tier = "fast"` with `[features].fast_mode = true`.
- `LOGBOOK.md` now clarifies that the retained `fast` profile is not a service-tier toggle.
- `agents-infra setup local -project-dir /Users/alexis/src/casual-talks -source-dir /Users/alexis/src/relux-works/relux-agents-infra -codex-config=global` synced the managed runtime copy and did not create `/Users/alexis/src/casual-talks/.codex/config.toml`.

## Preservation checks

- Source config and the `casual-talks` managed config both report `service_tier = "default"`.
- `agents-infra doctor global` reports `codex_config_effective: global`.
- `agents-infra doctor local /Users/alexis/src/casual-talks` reports `codex_config_present: false` and `codex_config_effective: global`.
- `/Users/alexis/src/casual-talks/.codex/config.toml` remains absent after setup.
- Unrelated source and runtime state was left untouched.

## Validation gates

| Command | Exit code | Notes |
| --- | ---: | --- |
| `agents-infra setup local -project-dir /Users/alexis/src/casual-talks -source-dir /Users/alexis/src/relux-works/relux-agents-infra -codex-config=global` | 0 | Sync completed; no project-local Codex config created. |
| `agents-infra doctor global` | 0 | `codex_config_effective: global` |
| `agents-infra doctor local /Users/alexis/src/casual-talks` | 0 | `codex_config_effective: global`, `codex_config_present: false` |
| `git diff --check` | 0 | Clean |
| `go test ./...` in `tools/agents-infra` | 0 | Passed |
| `go vet ./...` in `tools/agents-infra` | 0 | Passed |
| `go build ./...` in `tools/agents-infra` | 0 | Passed |
