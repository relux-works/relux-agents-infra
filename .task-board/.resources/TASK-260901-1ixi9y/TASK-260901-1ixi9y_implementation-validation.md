# TASK-260901-1ixi9y implementation and validation

## Scope

Global and local setup install regular executable openai-dange and anthropic-dange aliases beside their canonical provider aliases. Each wrapper adds one leading -d and preserves caller cwd and argv bytes/order. VerifyInstalledRuntime rejects missing, drifted, symlinked, and non-executable alias files; Setup repairs each state. README.md, SKILL.md, and LOGBOOK.md document the operator contract and review correction.

## Review rework

Revision 1 was correctly rejected because the installed alias chain reached runTarget with a leading -d, which flag.FlagSet rejected before BuildCanonicalTargetLaunchPlan. The repair in tools/agents-infra/main.go admits only that leading alias prefix, parses wrapper --print-config after it, then restores -d at the front of provider arguments. Other provider options retain the existing explicit -- boundary.

Production proof drives setup local -> installed dange alias -> canonical provider alias -> generated agents-infra target -> runTarget -> BuildCanonicalTargetLaunchPlan -> provider exec stub for both OpenAI and Anthropic. Each provider receives exactly one native danger flag and the caller sentinel. Separate wrapper execution records NUL-delimited argv to prove caller bytes/order and cwd. Negative setup/verify tests mutate the real installed artifacts into missing, drifted, symlinked, and mode-0644 states, require refusal, then require setup repair and successful verification.

## Validation

- Initial isolated installed-path test: exit 1 at compile time because the test used unavailable slices.Count; production was not executed. Replaced with an explicit count loop.
- go test -count=1 . -run ^TestInstalledLocalDirectProviderYoloAliasesReachProvidersWithOneNativeDangerFlag$: exit 0.
- Focused alias/setup/verify go test across ./...: exit 0.
- go test -count=1 ./...: exit 0; top-level 104.975s, infra 189.962s, modelharness 14.557s.
- go vet ./...: exit 0.
- go build ./...: exit 0.
- Focused README/SKILL documentation tests after the logbook correction: exit 0.
- git diff --check: exit 0.

No required validation was skipped.