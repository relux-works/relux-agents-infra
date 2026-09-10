# TASK-260911-39bag5 review verdict — rev1 — ACCEPTED

## What was checked

- Full diff of CR-TASK-260911-39bag5-1 rev1 (base 4126777d..candidate 956864d4) read file-by-file:
  agents_management_registry.go (new launchableSystems declaration + ValidateLaunchableEnvironment/
  ValidateLaunchableProvider cross-checked against agentic.Default), project_config.go,
  canonical_target.go, primary_session_launch_plan.go, agents_management_registry_test.go (new),
  LOGBOOK.md.
- go build ./..., go vet ./... — clean.
- go test ./... (full tools/agents-infra module) — all green (internal/infra 196s, root 115s,
  attachments 2s, modelharness 14s). No flake observed in this run.
- gofmt -l on all touched files — clean.
- Confirmed no go.mod/go.sum change.
- Independently re-applied two of the producer's named narrowing mutants by hand against a scratch
  copy and confirmed each fails its named test, then restored the files and diffed the restored
  working tree against the candidate tree OID (956864d4) to verify byte-for-byte equality (0 diff
  lines after git add -N on the untracked test file):
  1. Dropped the `lookup(...)` call in `validateLaunchableEnvironment`, keeping only declaration
     membership — `TestValidateLaunchableEnvironmentRegistryCrossCheckIsLoadBearing` failed as
     claimed.
  2. Re-added the literal `[]string{"codex", "claude-code", "pi"}` + hardcoded error string in
     `project_config.go` (i.e. reverted `validateProjectTarget` to call the old inline check instead
     of `ValidateLaunchableEnvironment`) — `TestLaunchableSystemSourceHasNoReintroducedLiteralList`
     failed as claimed.
- Verified golden-text preservation by inspection: `validateLaunchableEnvironment` and
  `validateLaunchableProvider` produce byte-identical error strings to the pre-change code
  (`must be one of codex, claude-code, pi` / `unsupported provider %q`) for the same admitted-set
  ordering.
- Verified the AC coverage table and mutant evidence table in
  `TASK-260911-39bag5_results.md` against the actual test file content — accurate, not padded.
- Checked remaining literal `"codex"`/`"claude"`/`"pi"` occurrences left in project_config.go
  (lines 158, 267, 273, 426-433, 453, 509) and canonical_target.go (lines 156, 267, 313, 316):
  these are a genuinely different concept (per-provider project-config table parsing, vendor/
  environment 1:1 pairing switch, hosted-vs-local bookkeeping) than the admission/dispatch sites
  named in the task description (project_config.go:411, canonical_target.go:80-98/417-421,
  primary_session_launch_plan.go:208-254), and were never in the task's named scope. Left alone
  correctly per the task's explicit constraint against changing which systems can be launched.
- Confirmed TASK-260830-11ajl2's superseded identity-only-guard pattern (8c7b771: adds a
  registry cross-check but leaves every hardcoded codex/claude/pi branch in place) was not revived
  — this CR actually removes the hardcoded literals from project_config.go and
  primary_session_launch_plan.go, a materially different and more complete approach.
- LOGBOOK entry documents a genuine finding from the producer's own adversarial mutant testing
  (a coincidental second refusal gate in validateProjectTarget's vendor/environment switch that
  could mask a bypassed first gate) and how the test was corrected to assert gate-specific wording
  instead of mere error presence — good adversarial rigor, independently plausible and consistent
  with the code.

## Verdict

ACCEPTED. All AC rows driven through the named production entries with negative and mutant
coverage; existing CLI wording preserved; installed-binary/project-config fixture tests pass
unchanged; full suite green; no go.mod change; superseded TASK-260830-11ajl2 guard not revived.
