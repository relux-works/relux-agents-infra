# TASK-260830-tvy8q5 review verdict — revision 4

## Verdict

Changes requested. Route to `to-dev`.

Immutable review surface:

- Change Request: `CR-TASK-260830-tvy8q5-4`, revision 4
- Base OID: `b78498bf98c05175db10bb341aee621e53de4881`
- Candidate tree OID: `36ff2a46233e928553a06bd373200cf63ea8ab39`
- Patch SHA-256: `f88f67f6ca3727ce569e411d1f02c38e3e3d22b5ffc4b14e8b73c4fb049e63f9`
- A reviewer-owned alternate-index snapshot of the managed worktree produced the exact candidate tree OID; `git diff --check` passed and no candidate file drifted.

## F1 — Fresh pre-retirement status publishes `within_policy=true`

Severity: blocking health-attestation defect.

Negative shape: **bypass path around the check**. `PiLifecycleStatus` requires a fresh from-start scan and stable even generations, but its `WithinPolicy` predicate ignores `LegacyCount` and `ForeignCount`. The following `SoakReady` predicate rejects them, so the same response simultaneously advertises healthy `within_policy=true` and unresolved legacy evidence.

Production call site: `tools/agents-infra/internal/infra/pi_session_log.go:2109` through `runPi -> runPiLifecycleCLI -> PiLifecycleOperatorStatus -> PiLifecycleStatus`. The shipped CLI test at `tools/agents-infra/pi_lifecycle_main_test.go:56` checks the legacy count and `soak_ready`, but does not require `within_policy=false`.

This violates acceptance criterion 3 and the public contracts at `SKILL.md:467-470`, `SKILL.md:493-496`, `README.md:1029-1033`, and `README.md:1075-1079`: legacy/foreign evidence is not healthy, and only a fresh complete post-retirement scan may publish `within_policy` and `soak_ready`.

Deterministic production-entry reproduction:

1. Run the existing non-launching CLI fixture with one mode-0600 legacy JSONL file and stable even generations.
2. Strengthen only its status assertion to require `WithinPolicy == false` before dry-run/confirmation.
3. Run uncached through a Go overlay, task-scoped HOME/cache, local file proxy, and no runtime/network path.

Actual result: `ScanComplete=true`, `LegacyCount=1`, `UnknownCount=0`, `WithinPolicy=true`, `SoakReady=false`; the strengthened production-entry test fails. Transcript: `TASK-260830-tvy8q5_reviewer-pre-retirement-health-probe.log` (SHA-256 `4f1413029532d7eac2dc2c00b930a575f88851e937f6918c3a0ce5f31b7efa69`).

Expected revision 5:

- Make legacy and foreign evidence refuse both health attestations on a fresh scan; preserve the explicit counts and documented unknown semantics rather than treating them as absence.
- Strengthen the real CLI production-entry test to require `within_policy=false` and `soak_ready=false` before retirement, then require both true only after exact confirmed retirement.
- Cover foreign evidence through the same production status entry point.
- Prove the new attestation guard with an uncached narrowing mutant that omits only the legacy/foreign condition, then rerun the focused status/retirement/crash/soak tests and the configured full gates.

## Revision-4 repair and reviewer validation

- Revision 4 correctly persists `policy_source`, validates it on odd generations, and compares current source before candidate inspection or mutation.
- Independent narrowed mutant removing only `legacy.PolicySource != policySource` made `TestPiLegacyRetirementResumeRejectsChangedPolicySource` fail with the admitted `err=nil`; SHA-256 `2ed3f2021a00745849cc04238b9afd9a03c26c92b80b1f3c77bcac6143e58088`.
- Twelve focused legacy authority/confirmation/crash/automatic-nonmutation tests passed uncached.
- Status odd/pagination/stale-policy/directory-mutation tests and deterministic eight-week crash/lease/reload/pressure soak passed uncached.
- CLI status/retirement/docs tests passed unchanged; the strengthened pre-retirement health probe failed as described above.
- Attached revision-4 full `go test ./...` and `go vet ./...` evidence is green. Revision 4 differs from revision 3 only in `LOGBOOK.md` and `//go:build !windows` production/test files, so prior exact-candidate cross-platform evidence was not invalidated by the policy-source repair.
- No Pi executable, installed runtime, model/provider process, service, socket, endpoint, setup/install flow, network proxy, or real user-HOME runtime/config state was contacted. Dependency reads used only the repository-local `file://` module proxy. Reviewer source and documentation remained read-only; overlays and transcripts are ignored `.temp/` artifacts.
