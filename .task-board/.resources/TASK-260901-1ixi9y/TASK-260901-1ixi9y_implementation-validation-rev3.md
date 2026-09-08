# TASK-260901-1ixi9y — implementation and validation, revision 3

## Revision 2 defect

Change Request revision 2 special-cased only the first alias-owned `-d` in
`runTarget`. The remaining argv was still passed to Go's `flag.FlagSet`, so a
caller-leading `-d`, `--danger`, `--yolo`, native danger flag, or ordinary
provider flag such as `--model` was rejected with `flag provided but not
defined` before `BuildCanonicalTargetLaunchPlan` could deduplicate danger
selection or launch the provider.

## Exact fix

`runTarget` now recognizes the direct-provider-YOLO path by its alias-owned
leading `-d`. Only on that path it consumes the wrapper-reserved
`--print-config` and first `--` boundary; every other caller token is forwarded
unchanged and in order to `BuildCanonicalTargetLaunchPlan`. Ordinary
`openai-infra` and `anthropic-infra` invocations retain their previous
`flag.FlagSet` parsing contract.

The source-managed setup/verify implementation from the earlier revisions is
preserved: global and local setup install regular executable `openai-dange` and
`anthropic-dange` siblings, verification rejects missing/drifted/symlinked/
non-executable aliases, and setup repairs those states.

## Negative production-path matrix

`TestInstalledLocalDirectProviderYoloAliasesForwardLeadingFlagsAndDeduplicateDanger`
builds the real `agents-infra` binary, drives `setup local`, executes the
installed aliases through their canonical siblings and the production
`runTarget` dispatcher, and records argv at fake `codex` and `claude` provider
executables. For both providers it covers caller-leading `-d`, `--danger`,
`--yolo`, exact `--model`, and all redundant danger spellings together without
an explicit `--`. It requires exactly one provider-native danger flag and an
ordered byte-exact subsequence of every non-danger caller token, including
spaces, tabs, newlines, empty strings, and Unicode. This test fails against CR
revision 2 at `runTarget` before provider execution.

`TestRunTargetCanonicalAliasStillRequiresBoundaryForUnknownWrapperFlag` is the
narrowing negative: without the alias-owned leading `-d`, the canonical alias
still rejects an unknown wrapper flag. This proves the repair did not silently
widen ordinary canonical alias parsing.

## Validation evidence

- Initial formatting/test shell attempt: exit 2 before tests ran because
  repository-relative file paths were mistakenly used from the module
  directory. Credited as no test evidence.
- First focused test attempt: exit 1 at test-package compilation due to a
  missing `slices` import. Fixed and not credited as passing evidence.
- Corrected narrow focused test, uncached: exit 0.
- Final focused alias/setup/verify suite across `.` and `./internal/infra`,
  uncached: exit 0.
- `go test -count=1 ./...`: exit 0.
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- `git diff --check`: exit 0.

Production call site under test:
`setup local -> installed *-dange -> canonical *-infra -> generated agents-infra target -> runTarget -> BuildCanonicalTargetLaunchPlan -> provider exec`.
