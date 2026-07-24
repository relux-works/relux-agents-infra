# Review verdict: changes requested

Findings:
- README.md says `agents-infra codex --profile fast` opts into Fast.
- Current Codex manual says Fast mode is toggled with `/fast on` / `/fast off` and can be persisted with `service_tier = "fast"` plus `[features].fast_mode = true`.
- `agents-infra codex --print-config --profile fast` shows the profile suppresses project model/effort and passes `--profile fast`; it does not change service tier.
- Source `.configs/codex-config.toml` correctly sets `service_tier = "default"`, and `profiles.fast` remains model/reasoning only.

Impact:
- The implementation satisfies the config sync part, but the documentation currently misstates the Fast opt-in mechanism.

Recommendation:
- Revise the README to describe the actual Fast activation path (`/fast on` or explicit `service_tier = "fast"` + `[features].fast_mode = true`) and keep `--profile fast` only if it is documented as a separate profile selection, not a service-tier toggle.
