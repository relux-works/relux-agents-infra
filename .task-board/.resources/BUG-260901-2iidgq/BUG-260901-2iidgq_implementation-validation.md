# BUG-260901-2iidgq implementation validation

## Change
Codex and Claude launch planning now use provider-scoped project parsing. Shared TOML/MCP and selected-provider fields remain strict. Full parsing remains unchanged for Pi, setup/verify, and canonical targets. README, shipped SKILL, and LOGBOOK were updated.

## bsim-shaped evidence
A temporary copy of /Users/alexis/src/.agents/.configs/project-config.toml was placed beneath a temporary bsim-shaped project and only the selected Qwen profile publisher line was removed. /Users/alexis/src/bsim and its config were not mutated.

- Installed pre-fix agents-infra Codex compose: exit 1, invalid_project_configuration, field agents.pi.profiles.qwen-3.8-27b-mlx-8bit.publisher.
- Candidate Codex compose through go run: exit 0, status ok.
- Candidate Claude compose through go run: exit 0, status ok.
- Candidate direct Pi compose: exit 1, names agents.pi.profiles.qwen-3.8-27b-mlx-8bit.publisher.
- Candidate canonical qwen-infra compose: exit 1, names the same field with source and remediation.

## Gates run directly
- Focused production-callsite tests with -count=1: exit 0.
- go test ./... -count=1: exit 0.
- go vet ./...: exit 0.
- go build ./...: exit 0.
- git diff --check: exit 0.

Production call sites named by tests: BuildPrimarySessionLaunchPlan to BuildCodexLaunchPlan or BuildClaudeLaunchPlan; strict negatives drive BuildPrimarySessionLaunchPlan for Pi and BuildCanonicalTargetLaunchPlan for qwen-infra.