# TASK-260829-3fozxa Review Verdict

## Verdict

Accepted: `CR-TASK-260829-3fozxa-5` revision 5 satisfies the task acceptance criteria and is ready for the orchestrator's checkpoint/integration flow.

No review findings remain.

## Immutable candidate

- Base OID: `91356833949cb6a30958265514fe5852d97eec1b`
- Candidate tree OID: `4b620a577bf730bed84d2546fe4019ccd99cb44c`
- Patch SHA-256: `c55a4020210d98a06b62c7dabcd428e37446b4e197a27c9c8e054c15aa10fdeb`
- Repository delta: present
- Reconstruction: the board patch digest matched an independently generated binary diff; a detached disposable clone checked out a synthetic commit over the supplied base and reproduced the exact candidate tree with a clean worktree.

The complete 17-path delta was reviewed:

- `LOGBOOK.md`
- `README.md`
- `tools/agents-infra/internal/infra/pi_config.go`
- `tools/agents-infra/internal/infra/pi_shared_attestation_test.go`
- `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go`
- `tools/agents-infra/internal/infra/pi_shared_integration_test.go`
- `tools/agents-infra/internal/infra/pi_shared_launcher_test.go`
- `tools/agents-infra/internal/infra/pi_shared_log_rotation.go`
- `tools/agents-infra/internal/infra/pi_shared_log_rotation_darwin.go`
- `tools/agents-infra/internal/infra/pi_shared_log_rotation_darwin_test.go`
- `tools/agents-infra/internal/infra/pi_shared_log_rotation_test.go`
- `tools/agents-infra/internal/infra/pi_shared_protocol_test.go`
- `tools/agents-infra/internal/infra/pi_shared_supervision_test.go`
- `tools/agents-infra/internal/infra/pi_standalone_shared_test.go`
- `tools/agents-infra/internal/infra/pi_standalone_test.go`
- `tools/agents-infra/internal/infra/pi_test.go`
- `tools/agents-infra/runtime_main_darwin_test.go`

## Acceptance-criteria review

- `max_segment_bytes` and `max_segments` are required positive platform integers at the production `RunPi -> loadCompositeProjectConfig -> parsePiRuntimeSharing` boundary. Missing, zero, and overflowing values refuse as invalid project configuration before provider lookup or runtime-state creation. No numeric code default exists.
- The production path is `startUnauthorizedRuntime -> startUnauthorizedRuntimeWithDependencies -> openSharedRotatingLog -> newSharedRotatingLogWriter -> sharedFilesystemLogSink`. Writes split before the first byte past the cap; an existing active segment exactly at cap rotates before accepting new data; oversized writes split across capped segments.
- `max_segments` counts the active segment. Production pruning retains at most `max_segments - 1` managed archives, sorts fixed-width sequence names oldest-first, preserves the active file, and ignores unrelated names.
- Startup validates the active segment and every managed archive against the current byte cap. Oversized, symlinked, linked, non-regular, or incorrectly permissioned managed paths fail closed before command start.
- The shipped 45-day fake-clock/fake-sink test has no sleeps and checks the footprint after every simulated day against `max_segment_bytes * max_segments`.
- The rotating writer remains owned until `command.Wait()` drains the `os/exec` copy paths, then closes. Start, rotate, prune, short-write, close, and write-after-close errors are not silently converted to success.

## Adversarial evidence

Independent disposable probes (removed before final validation) drove the production filesystem sink and start path without contacting any model process, service, socket, endpoint, or user harness:

- Pre-existing active file exactly at 4 bytes, one 13-byte crossing/oversized write, four rotations at one identical fake-clock timestamp, deterministic retention of sequences 3 and 4, active content `Q`, foreign-file preservation, and aggregate footprint `9B <= 12B`: pass.
- Active symlink, managed archive hardlink, and managed archive mode `0640`: each refused; foreign targets remained unchanged.
- Production start dependency seam launched only `/bin/sh -c 'printf late-output'`; all 11 bytes were present after wait/close across rotated segments, proving writer lifetime through process output drain.
- A successful harmless process plus an injected closer error returned that close error exactly once.
- Off-by-one mutant: changed production archive allowance from `maxSegments - 1` to `maxSegments`. The production filesystem probe failed with three archives retained where two were allowed (`exit 1`). Restoring the candidate made the same probe pass. The temporary test was deleted and the exact candidate tree was reverified.

This attacks the standard negative shapes **bypass path around the check** and **check present but uncalled from production**, including the revision-1 restart/config-change path.

## Trunk composition

- Current `origin/main` is exactly `91356833949cb6a30958265514fe5852d97eec1b`, equal to the revision-5 base.
- `git merge-tree --write-tree <candidate-commit> origin/main` produced `4b620a577bf730bed84d2546fe4019ccd99cb44c` with no conflict.
- The earlier required anchor `675f77ed63376320ed1213f46f9462a299c0abaf` is an ancestor of the revision-5 base by three commits. Its advance to the current base overlaps the revision-5 path set only at `LOGBOOK.md`; the candidate preserves the intervening root output-capture entry and adds the rotation replay entry above it.
- Persisted `restart_not_before` and `half_open` production/status fields and their tests remain present. Revision 5 changes only shared-supervision fixture policy values in that area, so it does not lose, shadow, or report stale recovery state.
- Merging the candidate with the exact `675f...` anchor also produced the candidate tree without conflict because that anchor is already contained in the candidate history.

## Validation

All commands ran in the pristine reconstructed candidate:

- Focused rotation, retention, restart/config-change, no-follow, missing/zero/overflow gates: `go test ./internal/infra -run '^(TestSharedRuntimeLog|TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart|TestOpenSharedRotatingLogRefusesManagedArchiveSymlinkWithoutTouchingForeignTarget|TestRunPiRejects(Missing|ZeroAndOverflowing)SharedRuntimeLogRotationPolicy)' -count=1 -v` — exit 0.
- Configured full suite: `go test ./... -count=1` — exit 0; root package `82.676s`, `internal/infra` `156.100s`.
- Configured static gate: `go vet ./...` — exit 0.
- Reviewer build gate: `go build ./...` — exit 0.
- `git diff --check <base> <tree>` — exit 0.
- `gofmt -d` over every changed Go path — no output.

Final disposable checkout status was clean and its tree remained exactly `4b620a577bf730bed84d2546fe4019ccd99cb44c`.
