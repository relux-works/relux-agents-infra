## Status
to-review

## Review
none

## Task Class
metadata

## Estimate
estimated(fibonacci(1))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Changed only /Users/alexis/src/.agents/.configs/project-config.toml: context_window 131072 -> 50000. agents-infra doctor accepts the inherited config. qwen-infra compose resolves the parent profile and computes expected models.json SHA ba8643cd646bf496ebca5ac70f1483b86624530311cdbbab9f8038ad383d59b2, different from the stale pre-launch catalog SHA 427eabd5242fc324015ad9c5d1b13fd552b4282033e46148750aab47d02e3c4b. Source inspection confirms managed Pi launch regenerates models.json from profile.ContextWindow before starting the runtime. No relux-agents-infra source-code change was required.

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-25T22:25:42Z

## Last Update
2026-08-25T22:27:35Z

## Assigned To
codex
