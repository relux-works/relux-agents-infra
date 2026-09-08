## Status
done

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Separate config-inheritance resolution from provider-surface materialization: an ancestor .agents that carries configuration only never becomes a nested project's runtime root
- [x] That same nested project still inherits project-config.toml from the ancestor .agents/.configs, with doctor local reporting the ancestor file as the value source
- [x] setupClaude and setupCodexWithConfig tolerate an absent source skills directory instead of returning an error
- [x] Negative test: a project nested under a config-only ancestor .agents does NOT get its surface written to the ancestor, asserted by absence of ancestor-side artifacts and not merely by absence of an error
- [x] Negative test: an agents dir with no skills directory renders the surface rather than failing, and does not create stray skill links
- [x] Positive control: a project with its own complete .agents still resolves to itself and overrides ancestor config
- [x] Regression witness reproducing the reported failure: prepare --agent claude for a project under a config-only ancestor .agents succeeds
- [x] go test ./... -count=1 and go vet ./... pass in tools/agents-infra
- [x] A nested project with no .agents of its own reports an explicit preparation no-op and runs on the global runtime, and no provider surface is written into the config-only ancestor

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/medium","text":"Sol at medium is the configured implementation rank 1 and the ceiling maximum for Codex here; the diagnosis is attached so no re-investigation budget is needed."}

## Precondition Resources
- [diagnosis.md](file://BUG-260909-163sup/diagnosis.md)

## Outcome Resources
- [validation-evidence.md](file://BUG-260909-163sup/validation-evidence.md) — Local mirror validation, control plants, and end-to-end check on the reporting configuration for PR #38

## Created
2026-09-09T14:49:10Z

## Last Update
2026-09-09T15:10:16Z
