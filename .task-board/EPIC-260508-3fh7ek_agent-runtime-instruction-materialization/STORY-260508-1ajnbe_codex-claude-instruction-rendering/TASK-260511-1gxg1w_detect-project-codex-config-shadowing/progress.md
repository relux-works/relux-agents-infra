## Status
done

## Assigned To
codex

## Created
2026-05-11T20:22:13Z

## Last Update
2026-05-11T20:36:44Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Plan: keep project-local Codex config absent; add doctor diagnostics for any project .codex/config.toml that shadows the global config; update README with the operational contract.
Design update: local agent runtime setup and tool config setup must be separate concerns. Default local setup should render/materialize instructions and link skills/helpers, but Codex model/reasoning config must remain global unless explicitly requested. Next patch should keep doctor diagnostics and avoid deleting custom project-local config without a clear legacy/generated marker.
Implemented narrower behavior: local setup now removes only the legacy managed .codex/config.toml symlink to project .agents/.configs/codex-config.toml; custom project-local configs are preserved and doctor reports codex_config_shadowing_global=true. Verified go test ./..., diff --check, custom-config and legacy-config smoke projects, and multi-tun doctor output.
Installed updated agents-infra via ./setup.sh. Installed doctor now reports codex_config_present/codex_config_linked/codex_config_shadowing_global. Verified global doctor and multi-tun local doctor with installed binary.
Follow-up: add explicit Codex config mode for local setup. Default should keep global config authoritative; an explicit local mode may install project-local config; diagnostics must show which mode is active/effective.
Added explicit --codex-config mode for setup local and refresh-links: preserve (default), global, local. preserve removes only legacy managed symlink and preserves custom overrides; global removes project-local config; local links .codex/config.toml to project .agents/.configs/codex-config.toml. Doctor now reports codex_config_effective plus action hints. Verified go test ./..., diff --check, preserve/global/local smoke, installed ./setup.sh, and multi-tun doctor.
Follow-up: document --codex-config usage in alexis-agents-infra SKILL.md so agents know local runtime setup and Codex model config are separate layers.
Final: documented Codex config modes in SKILL.md, synced global and multi-tun local installed skills, and prepared release tag v1.4.0.

## Precondition Resources
(none)

## Outcome Resources
(none)
