# TASK-260829-3fozxa Review Verdict

## Verdict

Changes requested for `CR-TASK-260829-3fozxa-1` revision 1. Route the implementation to `to-dev`.

The exact candidate was reconstructed in a detached disposable review worktree from base `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`. The board patch SHA-256 is `6376a0a3d6e1cf2aec118aaecd5af1faccbf9eefc266c31ce337c010055d3405`; the staged reconstructed tree is exactly `4d3ad50d9920a5b179b6bbf702cc720feac21217`.

## Required Finding

### F1 — Restart/config-change bypass admits retained archives above the configured cap

Severity: acceptance-criteria failure.

Production path: `startUnauthorizedRuntime` -> `openSharedRotatingLog` -> `newSharedRotatingLogWriter` -> `sharedFilesystemLogSink.Prune`.

`newSharedRotatingLogWriter` validates only the active file size (`pi_shared_log_rotation.go:48`) before calling `Prune`. `sharedFilesystemLogSink.Prune` limits archive count but never validates the sizes of archives it retains (`pi_shared_log_rotation_darwin.go:53`). Therefore an archive created under an older, larger `max_segment_bytes` survives a broker/runtime restart after the operator lowers the cap. This is the standard negative shape **bypass path around the check**: the active segment is guarded, while retained segments reach the protected bounded state through the restart path without the same guard.

Independent production-sink probe, run with no model process or service:

- pre-existing active segment: `1B`
- pre-existing valid mode-0600 archive: `10B`
- configured `max_segment_bytes=4`, `max_segments=2`
- observed: `openSharedRotatingLog` succeeded and retained `11B`
- required bound: `4 * 2 = 8B`
- result: probe failed with `restart admitted footprint=11 above max_segment_bytes*max_segments=8` (exit 1)

This violates the primary aggregate-footprint AC and the reviewer brief's broker/runtime restart requirement. The shipped fake sink starts with no retained archives, so its multi-day test cannot expose this path.

Required rework:

1. Fail closed or deterministically remove managed retained archives whose size exceeds the current `max_segment_bytes`; the policy must preserve the no-foreign-file and no-follow guarantees.
2. Add a negative restart/config-change test using the production filesystem sink plus fake clock. It must establish that a retained archive created under a larger prior cap cannot produce a successful writer whose aggregate footprint exceeds the new product bound.
3. Drive the same production call chain used by `startUnauthorizedRuntime`, name it in the test, and prove refusal occurs before runtime launch side effects when refusal is the chosen policy.
4. Add the regression and its resolution to `LOGBOOK.md` in the next candidate revision. This reviewer did not alter the immutable candidate or repository files.

## Adversarial Evidence

- Focused shipped tests: `go test ./internal/infra -run '^(TestSharedRuntimeLog|TestRunPiRejectsMissingSharedRuntimeLogRotationPolicy)' -count=1 -v` — exit 0.
- Full suite: `go test ./... -count=1` — exit 0; root package `96.812s`, `internal/infra` `163.752s`.
- Static checks: `go vet ./...` — exit 0; `go build ./...` — exit 0.
- Narrowing mutant: changed the production rotation prune allowance from `maxSegments` to `maxSegments-1`; `TestSharedRuntimeLogPrunesOldestSegmentsDeterministically` failed with `segments=1 want=2` (exit 1). Candidate source was restored and tree hash reverified.
- Independent overflow probes: out-of-range TOML values for both caps refused as `invalid_project_configuration` before provider lookup and cache creation — pass.
- Independent production-sink boundary probe: pre-existing active file exactly at cap, crossing/oversized write, equal timestamps, oldest-first pruning, active preservation, and unrelated-file preservation — pass.
- Independent no-follow probe: matching archive symlink refused and foreign target remained unchanged — pass.
- Final reconstructed candidate tree after all temporary probes/mutants: `4d3ad50d9920a5b179b6bbf702cc720feac21217`; no unstaged review mutation remained.

The green shipped suite does not supersede F1 because it lacks retained-archive restart state. Revision 1 is not accepted.
