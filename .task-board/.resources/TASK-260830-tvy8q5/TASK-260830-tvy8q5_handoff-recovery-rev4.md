# TASK-260830-tvy8q5 revision 4 developer evidence

## Change

- Persist `policy_source` in every odd legacy retirement generation.
- Reject odd-generation resume before candidate inspection or mutation unless the current policy source, numeric policy digest, and confirmed full-plan hash all match the persisted authority.
- Validate that odd generations carry policy provenance and even generations carry none; include the added field in the bounded control-document preflight.
- Add the production-entry negative/positive crash test `TestPiLegacyRetirementResumeRejectsChangedPolicySource` and record the root cause in `LOGBOOK.md`.

Production SHA-256: `62f61690851f8a05374571dc39a4200da5f247c879987fb091ced66c07ce6a12` (`pi_lifecycle_legacy.go`).
Test SHA-256: `6ecc06e8b7a645af773f33ebeb6ac854cc7ea803628c9825907262c8af6addbf` (`pi_lifecycle_legacy_test.go`).

## Direct validation

All successful Go gates used a task-scoped `HOME`, `GOCACHE`, `GOMODCACHE`, and `GOPATH`, with `GOSUMDB=off`, `GOTELEMETRY=off`, and a read-only `file://` proxy backed by the repository's existing `.temp/remote-model-harness/setup-smoke-home/go/pkg/mod/cache/download`. No network proxy was used.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Initial rev4 test with `GOPROXY=off` | 1 | Dependency lookup refused before package setup; recorded honestly in `rev4-policy-source-green.log` before the cache route was corrected. This was not a code-test result. |
| `go test ./internal/infra -run '^TestPiLegacyRetirementResumeRejectsChangedPolicySource$' -count=1 -v` | 0 | Exact production-entry source-a/source-b refusal and source-a resume. |
| Same test with only the `legacy.PolicySource != policySource` comparison narrowed away | 1 expected-red | `changed policy source reused odd-generation authority: <nil>`; `rev4-policy-source-narrowed-mutant.log`. Guard restored with an exact patch before all later gates. |
| Focused legacy rev2/rev3/rev4, automatic non-mutation, and deterministic eight-week soak tests | 0 | Eight named tests passed in `5.084s`; `rev4-focused-infra.log`. |
| Same focused infra set under `-race` | 0 | Eight named tests passed in `6.819s`; `rev4-focused-race.log`. |
| CLI status pagination, non-launching exact-plan retirement, and docs tests | 0 | Four named tests passed; `rev4-cli-docs.log`. |
| `go test ./... -count=1` | 0 | Full module suite passed; slowest package `internal/infra` in `161.225s`; `rev4-full-go-test.log`. |
| `go vet ./...` | 0 | `rev4-full-go-vet.log` (empty). |
| Native Darwin/arm64 `go build ./...` | 0 | `rev4-native-build.log` (empty). |
| Linux/amd64 `CGO_ENABLED=0 go build ./...` | 0 | `rev4-linux-amd64-build.log` (empty). |
| `gofmt -l` assertion and `git diff --check` | 0 / 0 | No listed files and no whitespace errors. |

The Windows compile was not rerun in this recovery because the isolated cache does not contain the Windows-only `github.com/natefinch/atomic` module and the assignment forbids network or real-HOME module-cache contact. The prior attached result records a green Windows compile for the candidate. Revision 4 changes only `//go:build !windows` production/test files and `LOGBOOK.md`, so the Windows compilation surface is unchanged; this is reliance on attached evidence, not a newly executed Windows gate.

## No-live-runtime boundary

No Pi executable, installed runtime, model, provider process, service, socket, endpoint, setup/install flow, or real user-HOME runtime/config state was contacted. Tests used only deterministic temporary filesystem state and fake time. Required role skills were read from their installed instruction paths; all Go validation state remained task-scoped and module reads came only from the repository-local file proxy described above.
