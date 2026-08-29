# Installed runtime verification

Source integration: `aa38ce9` (`agents-infra v1.6.1-24-gaa38ce9`).

The bootstrap installed the binary built from the integrated source and ran its
global setup. Explicit `doctor global` and `verify global` checks then passed.
The managed global Codex config retained its existing `projects`, `notice`, and
non-`fast` custom profile tables, removed `profiles.fast`, and now uses
`service_tier = "default"`.

The first project-local setup probe correctly refused an explicit source path
that would copy the repository into its own `.agents` destination. Retrying the
supported command without that self-containing override selected the installed
global runtime as its source. Project-local setup, doctor, and verify then
passed. The old local managed Codex config retained every pre-existing
`projects`, `notice`, and non-`fast` custom profile entry, removed
`profiles.fast`, and now uses `service_tier = "default"`. The project-owned
`.agents/.configs/project-config.toml` remained byte-identical and the absent
`.codex/config.toml` remained absent.

Evidence logs are retained locally under `.temp/TASK-260824-293a0x/`:

- `install-global.log`, `doctor-global.log`, `verify-global.log`
- `setup-local-02.log`, `doctor-local-02.log`, `verify-local-02.log`
- `global-codex-before.toml`, `local-codex-before.toml`
- `compare_codex_config.go`, `project-config-before.sha256`
