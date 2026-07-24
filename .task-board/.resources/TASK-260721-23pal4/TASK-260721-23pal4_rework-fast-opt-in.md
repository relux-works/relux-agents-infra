# Focused rework

The service-tier change and runtime synchronization are correct. Do not redo or alter them.

Correct these claims:
- README currently says `agents-infra codex --profile fast` opts into Fast. This is false because `[profiles.fast]` contains only model and reasoning fields.
- LOGBOOK says Fast remains an explicit profile. Clarify that the profile is retained but is not a service-tier toggle.
- Update `TASK-260721-23pal4_results.md` on the board so it no longer repeats the false profile claim.

Use the current official Codex mechanism: `/fast on` opts into Fast interactively; persistent Fast requires `service_tier = "fast"` with `[features].fast_mode = true`. Keep the source service tier at `default`, keep `[profiles.fast]` unchanged, and do not touch installed runtime copies.
