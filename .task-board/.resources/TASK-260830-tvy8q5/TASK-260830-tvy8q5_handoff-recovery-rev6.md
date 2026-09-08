# TASK-260830-tvy8q5 revision 6 developer recovery evidence

Date: 2026-08-30
Role: developer
Workspace: managed Story worktree on `task-board/story/STORY-260830-2ywmg2`

## Recovery scope

- Inspected the current tracked diff and all four untracked lifecycle implementation/test files.
- Preserved the existing production implementation.
- Added the reviewer-requested production-entry regression `TestRunPiLifecycleStatusRefusesForeignEvidence` and corrected the corresponding LOGBOOK evidence line.
- Did not run setup, launch, an installed runtime, a provider/model, a socket probe, or any live service/endpoint.
- Go ran with `GOENV=off`, `GOTELEMETRY=off`, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, and worktree-local caches. Test configuration used temporary HOME/PATH values and a deliberately nonexistent runtime path.

## Tool readiness

- `task-board --version`: exit 0; `0.24.3-172-g063197b1`.
- `git --version`: exit 0; `2.53.0`.
- `go version`: exit 0; `go1.25.5 darwin/arm64`.
- `gofmt` resolved at `/opt/homebrew/bin/gofmt`.

## Direct validation

1. Clean production-entry negative:
   `go test . -run '^TestRunPiLifecycleStatusRefusesForeignEvidence$' -count=1`
   - Exit 0.
   - Drives `runPi -> runPiLifecycleCLI -> PiLifecycleOperatorStatus`.
   - Requires `foreign_count=1`, exact foreign bytes, `legacy_count=0`, `unknown_count=0`, `within_policy=false`, `soak_ready=false`, and preserved external evidence.

2. Narrowed gate mutant:
   `go test -overlay=.../foreign-mutant-overlay.json . -run '^TestRunPiLifecycleStatusRefusesForeignEvidence$' -count=1`
   - Exit 1, expected red and reported as failing.
   - The overlay removes only `status.ForeignCount == 0` from the production `WithinPolicy` expression.
   - Failure observed `foreign_count=1` with the mutant incorrectly publishing `within_policy=true` and `soak_ready=true`.

3. Clean CLI lifecycle neighbors:
   `go test . -run '^TestRunPiLifecycle' -count=1`
   - Exit 0.

4. Clean legacy/automatic-path/eight-week deterministic suite:
   `go test . -run '^(TestPiLegacyRetirement|TestPiAutomaticSetupLaunchAndStatusNeverMutateLegacyEvidence|TestPiRetentionPlaneDeterministicEightWeekCrashLeaseReloadPressureSoak)' -count=1`
   - First invocation: exit 1 because its worktree-local module-cache path was one directory too shallow. `GOPROXY=off` refused lookup; no endpoint was contacted.
   - Corrected invocation of the exact same test selection: exit 0 in 5.642s.

5. Formatting and diff validation:
   - `gofmt -d tools/agents-infra/pi_lifecycle_main_test.go`: exit 0, no output.
   - `git diff --check`: exit 0, no output.

The previously attached revision-5/full-candidate evidence remains the authority for the full repository, race, vet, build, Linux/Windows cross-compile, deterministic soak, and static installed-parity gates. This recovery reran only checks covering the test-only revision-6 delta, as required by the bounded handoff instruction.

## Candidate identity before handoff

- Tracked binary diff SHA-256: `c1cc0f4ee09be86b3d2e17ccc4189fd6ef9d5438c6ab8feb1e2d507e7a7f89ce`.
- `pi_lifecycle_legacy.go`: `62f61690851f8a05374571dc39a4200da5f247c879987fb091ced66c07ce6a12`.
- `pi_lifecycle_legacy_test.go`: `6ecc06e8b7a645af773f33ebeb6ac854cc7ea803628c9825907262c8af6addbf`.
- `pi_lifecycle_soak_darwin_test.go`: `1335486eae60dbc2620ce29c73dcb0310633a82eafcb51126b6ec2825dfb6102`.
- `pi_lifecycle_main_test.go`: `3bd22dc9dd41161a6f514ad81f13bf46a2c454064d61cb7d558b65ac7c8dea14`.

Ready for independent review; no review or integration claim is made by this developer evidence.
