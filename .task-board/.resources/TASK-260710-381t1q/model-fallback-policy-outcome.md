# Model Availability And Stop-The-Line Policy Outcome

## Implemented

- `.instructions/INSTRUCTIONS_WORKFLOW.md`
  - Added an autonomous model availability policy: never hand the wait-versus-
    downgrade choice to the human, retry the preferred model at least three
    times with backoff, then select the best viable fallback and escalate only
    a real blocker.
  - Strengthened Stop-The-Line as autonomous-by-default, with objective
    external blocker criteria, explicit forced-fit warning signs, a required
    persisted evidence/alternatives/decision packet, and the iOS Bluetooth
    pre-permission state example.
- `README.md`
  - Documented the supported rate-limit nudge suppression, Codex 0.144.1
    safety-buffering limitation, autonomous fallback behavior, and expanded
    workflow module responsibility.
- `tools/agents-infra/internal/infra/infra_test.go`
  - Added materialization assertions for both policy streams in generated Codex
    instructions and the shared Claude instruction tree.
  - Added a global setup assertion for
    `[notice].hide_rate_limit_model_nudge = true`.

The supported runtime key was already present in
`.configs/codex-config.toml`, so no source config edit was required.

## Runtime Limitation

Codex CLI 0.144.1 has no supported config key for the safety-buffering chooser
that offers `Retry with a faster model` and `Keep waiting`. The upstream source
implements that chooser directly in the TUI, while the generated config schema
contains no suppression, default-action, or auto-wait field. Global
instructions are loaded only after runtime routing reaches the model and cannot
pre-answer that UI. Agents-infra intentionally does not automate terminal key
presses or patch the Codex binary.

The separate approaching-rate-limit model switch nudge is suppressed by the
supported `notice.hide_rate_limit_model_nudge = true` setting.

## Verification

Passed:

```text
go test ./...
go vet ./...
go run . setup global --source-dir ... --home-dir .temp/.../home-02
go run . doctor global --home-dir .temp/.../home-02
./setup.sh
agents-infra doctor global
codex debug prompt-input 'installed policy verification probe'
git diff --check
```

Installed runtime verification:

- `~/.codex/config.toml` links to
  `~/.agents/.configs/codex-config.toml` and contains the suppression key.
- `~/.codex/AGENTS.md` contains both policy streams and no raw
  `@~/.agents` includes.
- `~/.claude/instructions/INSTRUCTIONS_WORKFLOW.md` contains both policy
  streams.
- `cmp` reports no drift for the workflow instructions, Codex config, or
  README between source and `~/.agents`.
- `agents-infra doctor global` reports rendered/linked/installed state healthy.

Logs are under `.temp/agents-model-fallback-policy/`.

## Tooling Notes

- The official Codex manual helper failed because the response omitted
  `x-content-sha256`; the audit continued against the exact official upstream
  `rust-v0.144.1` source tag and generated config schema.
- Codex reports that `--strict-config` is unsupported for `features` and
  `debug`; the supported notice key was instead verified from the exact source
  schema and through the installed runtime link/content checks.
- No files were staged or committed.
