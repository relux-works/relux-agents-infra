# Codex Model Fallback Policy Audit

## Scope

- Source of truth: `/Users/alexis/src/relux-works/relux-agents-infra`
- Installed runtime destinations: `~/.agents`, `~/.claude`, and `~/.codex`
- Tracked task: `TASK-260710-381t1q`
- Audited Codex CLI: `codex-cli 0.144.1`
- Audited upstream source tag: `rust-v0.144.1` (`44918ea10c0f99151c6710411b4322c2f5c96bea`)

## Runtime Findings

Codex has two separate model-switch interfaces:

1. The approaching-rate-limit nudge is configurable. The supported key is
   `[notice].hide_rate_limit_model_nudge`. The source-managed config already
   sets it to `true` in `.configs/codex-config.toml`, so global setup suppresses
   that lower-credit model chooser.
2. The safety-buffering prompt offers `Retry with a faster model`, `Keep
   waiting`, and `Learn more`. In Codex 0.144.1 it is implemented directly in
   `codex-rs/tui/src/chatwidget/safety_buffering.rs`. The complete generated
   `codex-rs/core/config.schema.json` has no safety-buffering notice, default
   action, auto-wait, or model-retry preference. Therefore agents-infra cannot
   suppress or pre-answer this prompt through a supported Codex config key.

The safety-buffering prompt occurs in the runtime before the selected model
continues, so `AGENTS.md` instructions cannot control the prompt itself.
Automating terminal key presses or patching the installed Codex binary would be
an unsupported forced fit and should not be added to agents-infra.

## Recommended Source Edits

- Add a durable policy to `.instructions/INSTRUCTIONS_WORKFLOW.md`: never ask
  the human whether to wait or downgrade; retry the preferred model several
  times with backoff; after repeated unavailability choose the best alternative
  execution path autonomously; record the fallback; escalate only a real
  blocker.
- Keep `.configs/codex-config.toml` with
  `hide_rate_limit_model_nudge = true` and add a regression assertion that
  global setup installs it.
- Add renderer regression coverage proving the model-availability policy is
  present in generated Codex instructions and remains available to Claude via
  the shared instruction tree.
- Document the supported rate-limit setting and the safety-buffering limitation
  in `README.md`.

## Validation And Install

Run:

```bash
cd tools/agents-infra
go test ./...
go vet ./...

cd ../..
./setup.sh
agents-infra doctor global
```

`./setup.sh` is the canonical bootstrap/install verification path. It builds
and installs `agents-infra`, runs `setup global`, and runs `doctor global`.
Afterward verify that `~/.codex/config.toml` resolves to the source-installed
config, contains `hide_rate_limit_model_nudge = true`, and rendered
`~/.codex/AGENTS.md` contains the autonomous model-availability policy with no
raw `@~/.agents` includes.

## Evidence

- Codex manual helper failed because the response omitted
  `x-content-sha256`; log:
  `.temp/agents-infra-policy-audit/codex-manual-fetch-02.log` in the
  `skill-project-management` checkout.
- Exact upstream source was cloned under
  `.temp/agents-model-fallback-policy/codex-source`.
- Search and source-discovery logs are under
  `.temp/agents-model-fallback-policy/`.
