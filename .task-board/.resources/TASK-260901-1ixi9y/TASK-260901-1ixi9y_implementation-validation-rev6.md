# TASK-260901-1ixi9y revision 6 implementation evidence

## Revision-5 defect
CR revision 5 authenticated dange origin by comparing caller-controlled argv[1] with the public --agents-infra-direct-provider-yolo-call-site string. A direct openai-infra or anthropic-infra caller could forge that marker and enter delimiter-free implicit YOLO forwarding without either dange alias in the call chain.

## Exact repair
The production marker and marker branch were removed. Installed openai-dange and anthropic-dange now dispatch to the sibling agents-infra binary through the distinct target-yolo command with the authoritative canonical entrypoint. runTarget returned to its strict FlagSet path and shares only post-parse canonical resolution/launch code. The target-yolo route admits only openai-infra and anthropic-infra, injects one -d selection, and canonical resolution deduplicates all caller danger forms to one provider-native flag.

## Killing negative and production matrix
TestInstalledLocalProviderAliasesScopeImplicitFlagForwardingToDangeChain drives setup local, both installed dange aliases, the generated launcher, target-yolo/runDirectProviderYoloTarget, canonical resolution, and provider exec stubs. It retains the complete -d, --danger, --yolo, native-danger, --model, redundant-danger, empty/space/newline/tab/Unicode argument matrix. The same installed test invokes openai-infra and anthropic-infra with the exact public revision-5 marker and a marker-like assignment and requires flag-provided-but-not-defined refusal before the provider record can be created. TestRunTargetCanonicalAliasesRefuseForgedRevision5Marker independently kills the old runTarget marker branch.

## Validation run directly by revision-6 developer
- go test ./internal/infra -run Test(DirectProviderYolo|SetupRepairsAndVerifyRejectsDirectProviderYoloAliasDrift|SetupLocalCreatesInstalledRuntime|SetupGlobalDoesNotInstallCLIWrapper) -count=1 — exit 0
- go test . -run Test(RunDirectProviderYoloTarget|ParseDirectProviderYoloTargetArgs|RunTargetCanonicalAliases|InstalledLocalProviderAliasesScopeImplicitFlagForwardingToDangeChain) -count=1 — exit 0
- go test ./... -count=1 — exit 0
- go vet ./... — exit 0
- go build ./... — exit 0
- git diff --check — exit 0

## Changed paths
LOGBOOK.md, README.md, SKILL.md, and nine Go product/test files under tools/agents-infra. No .temp-review, board resource, prompt, run log, or agent scratch path is present in git diff --name-only.