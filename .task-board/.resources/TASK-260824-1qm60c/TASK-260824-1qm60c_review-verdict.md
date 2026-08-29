# Review verdict — accepted (CR revision 2)

## Verdict

Accepted. CR `CR-TASK-260824-1qm60c-2`, revision `2`, is ready for the
commit-owning orchestrator to checkpoint/integrate.

## Scope and architecture

- `.configs/codex-config.toml` removes `[profiles.fast]` and keeps
  `service_tier = "default"`.
- README no longer offers the withdrawn fast profile.
- The former repeat-sync bypass is closed at the production call path:
  `infra.Setup` -> `syncRepo` -> `syncManagedCodexConfig`.
- Source-controlled model/reasoning/service-tier defaults replace stale managed
  values; installed `projects`, `notice`, and non-fast custom profiles survive.
  Malformed existing TOML refuses sync without being replaced.
- The local primary-session policy and custom project `.codex/config.toml`
  remain user-owned and byte-identical across the local repeat-sync path.

## Gate-defeat evidence

Negative shape attacked: **bypass path around the check**.

I seeded global and local managed configurations with the withdrawn
`service_tier = "fast"` and `[profiles.fast]`, plus a custom profile, trust
marker, and TUI acknowledgement. I drove the real global and local CLI setup
paths in a disposable runtime, then repeated local setup. Both managed copies
reject the fast profile/tier and retain `service_tier = default`, the custom
profile, trust marker, and TUI state. `doctor` and `verify` passed for global
and local modes. The user-managed local project config and primary-session file
had identical SHA-256 values before and after repeat sync.

The malformed-existing-TOML boundary is also covered by
`TestSetupGlobalRejectsMalformedExistingCodexConfigWithoutReplacingIt`: parse
failure is not treated as absence and the existing bytes remain unchanged.

## Validation

- `go test ./... -count=1` — pass.
- Focused source/migration/refusal tests with `-count=1` — pass.
- `go vet ./...` — pass.
- `go build ./...` — pass.
- `gofmt -l` on the changed Go files — clean.
- Exact CR diff whitespace check — clean; worktree equals candidate tree
  `a1f084fb0def43889cc3a517bc00072033c81817`.

Detailed review logs are under `.temp/TASK-260824-1qm60c/`; the persistent
root-cause and migration finding is already recorded in `LOGBOOK.md`.
