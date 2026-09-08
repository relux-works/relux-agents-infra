ACCEPTED — CR-TASK-260901-1ixi9y-6 (revision 6).

Verified the revision-5 marker-forgery defect (LOGBOOK 2026-09-01 1300) is fixed:
runTarget/runCanonicalTarget no longer authenticates dange origin from any argv
token. The dange chain is now a distinct dispatched subcommand
(agents-infra target-yolo <entrypoint>), injected server-side as
providerArgs[0]="-d" inside runDirectProviderYoloTarget, never derived from
caller-controlled bytes. openai-infra/anthropic-infra keep runTarget's original
strict flag.FlagSet path untouched.

Attacked the gate directly:
- Confirmed via TestRunTargetCanonicalAliasesRefuseForgedRevision5Marker and the
  installed-binary matrix (TestInstalledLocalProviderAliasesScopeImplicitFlagForwardingToDangeChain)
  that openai-infra/anthropic-infra reject both the retired revision-5 marker
  literal and a marker-like key=value variant with "flag provided but not
  defined", with no provider side effect (record-file absence checked).
- Confirmed canonical aliases still refuse bare -d/--danger
  (TestRunTargetCanonicalAliasesStillRefuseLeadingDangerOutsideDangeRoute),
  preserving the pre-existing contract.
- Confirmed target-yolo entrypoint whitelist rejects qwen-infra and arbitrary
  strings (only openai-infra/anthropic-infra allowed).
- Confirmed exactly one native danger flag reaches provider argv even with
  every redundant caller danger shortcut combined (-d --danger --yolo
  --dangerously-*), and non-danger caller bytes/order (including Armenian
  text, embedded newlines/tabs, empty strings) are preserved byte-for-byte,
  via both the in-process launch-plan test and the installed-binary
  end-to-end matrix driving real generated aliases against fake provider exec
  stubs.
- Confirmed setup/verify install, detect, and repair openai-dange/
  anthropic-dange for missing, drifted, symlinked, and non-executable states
  (POSIX), matching the existing canonical-alias contract exactly.
- gofmt clean, go vet clean, full `go test ./...` green (all 4 packages,
  ~107s/191s/15s/2s), focused re-run of every new/changed test green.

No forced fits, no positive-path-only evidence — every gate claim has a
matching negative test hitting the real production call site (runTarget /
runDirectProviderYoloTarget / installed generated binaries), not a helper
tested in isolation.

DoD: all items satisfied. README/SKILL/LOGBOOK accurately document the
revision-5 defect and revision-6 fix. Changed paths are product source, tests,
and docs only — no scratch artifacts.
