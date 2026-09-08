# TASK-260901-1ixi9y revision 4 implementation and validation

## Revision 3 defect
Revision 3 inferred dange call-site origin from caller-visible leading `-d`. Direct `openai-infra -d` and `anthropic-infra -d` therefore entered the permissive dange parser even though their pre-existing contract required `flag provided but not defined` refusal.

## Exact repair
The source-managed `openai-dange` and `anthropic-dange` wrapper bytes now supply `--agents-infra-direct-provider-yolo-call-site` to their exact sibling canonical aliases. `runTarget` recognizes that marker only in the first internal position, consumes it before provider argv, injects one wrapper-owned `-d`, and sends the resulting argv through authoritative `BuildCanonicalTargetLaunchPlan` danger de-duplication and target locking. Unmarked canonical aliases retain `flag.FlagSet` delimiter and unknown-flag behavior. The marker literal has one source of truth in `infra.DirectProviderYoloCallSiteMarker`.

## Production negative matrix
`TestInstalledLocalProviderAliasesScopeImplicitFlagForwardingToDangeChain` drives `setup local` installed regular-file aliases through the generated canonical wrapper and real `runTarget` dispatcher into fake Codex and Claude executables. Both dange aliases accept caller-leading `-d`, `--danger`, `--yolo`, native danger, and `--model` without an explicit boundary; redundant danger selections become exactly one provider-native danger flag; non-danger empty, whitespace, newline, tab, and Armenian argument bytes preserve order; the internal marker never reaches provider argv. The same installed test invokes `openai-infra -d --model caller-model` and `anthropic-infra -d --model caller-model`, requires the historical parse refusal, and proves neither provider side effect occurred. This matrix fails against CR revision 2 and revision 3.

## Documentation
README, `SKILL.md`, and `LOGBOOK.md` scope implicit provider-flag forwarding to the marked dange chain only. The logbook explicitly records the revision 3 origin-inference defect and revision 4 correction.

## Validation run directly in this developer run
- Focused dispatcher, wrapper, drift/repair tests: `go test . ./internal/infra -count=1 -run ...` — exit 0.
- Installed dual-route production matrix: `go test . -count=1 -run ^TestInstalledLocalProviderAliasesScopeImplicitFlagForwardingToDangeChain$` — exit 0.
- Full uncached suite: `go test ./... -count=1` — exit 0.
- Static analysis: `go vet ./...` — exit 0.
- Build: `go build ./...` — exit 0.
- Whitespace gate: `git diff --check` — exit 0.
