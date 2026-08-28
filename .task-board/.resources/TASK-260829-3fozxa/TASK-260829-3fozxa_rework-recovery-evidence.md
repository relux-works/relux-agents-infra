# TASK-260829-3fozxa Rework Recovery Evidence

## Outcome

Revision 1 finding F1 is closed by a fail-closed restart/config-change policy.
`sharedFilesystemLogSink.Prune` validates every managed retained archive against
the currently configured `max_segment_bytes` before deterministic count
pruning. A successful writer therefore retains one active segment plus at most
`max_segments - 1` archives, and each retained file is at most
`max_segment_bytes` bytes. Oversized, non-regular, linked, or incorrectly
permissioned managed archives refuse writer construction.

Production path under test:

`startUnauthorizedRuntime` -> `startUnauthorizedRuntimeWithDependencies` ->
`openSharedRotatingLog` -> `newSharedRotatingLogWriter` ->
`sharedFilesystemLogSink.Prune`.

`TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart`
uses the production filesystem sink and fake clock with a 1-byte active file,
a 10-byte managed archive, `max_segment_bytes=4`, and `max_segments=2`. Writer
construction refuses with `shared_runtime_state_path_invalid`; the injected
`Cmd.Start` boundary receives zero calls and both existing files are preserved.

This recovery also added
`TestRunPiRejectsZeroAndOverflowingSharedRuntimeLogRotationPolicy`, which drives
the real `RunPi` entry point. Zero and out-of-range TOML values for both caps
refuse as `invalid_project_configuration` before provider lookup or creation of
the runtime cache/state tree. Together with the existing missing-cap test, the
shipped suite now covers absent, zero, and overflowing operator policy at the
production boundary.

## Commands run directly by recovery run RUN-260829-f8bfc7

| Command | Exit | Result |
| --- | ---: | --- |
| Focused rotation/restart/symlink suite, `-count=1 -v` | 0 | Production restart refusal, exact cap, deterministic pruning, 45-day bound, and no-follow cases pass. |
| Initial new zero/overflow production-entry test | 1 | Test assertion was too narrow: TOML overflow correctly refused at the document parser before a field-specific error could be emitted. This was not reported as a product failure or pass. |
| Corrected missing/zero/overflow production-entry suite, `-count=1 -v` | 0 | Both caps refuse before provider lookup and runtime-state mutation. |
| `go test ./... -count=1` | 0 | Root 169.670s; infra 327.101s; attachments/modelharness pass; command package has no tests. |
| `go vet ./...` | 0 | Pass. |
| `go build ./...` | 0 | Pass. |
| `gofmt -l internal/infra runtime_main_darwin_test.go` | 0 | No output. |
| `git diff --check` after the logbook update | 0 | Pass. |

The exact automatic CR validation command had failed in two prior construction
attempts on unrelated timing-sensitive model-check/readiness fixtures while
other Go suites were running and host load was elevated. This run waited for
the foreign infra suite to exit and then ran the exact command as one foreground
process; it passed. No rotation code was changed to accommodate those timing
failures.

## Adversarial evidence provenance

The board-attached
`TASK-260829-3fozxa_revision2-implementation-evidence.md` records the narrowing
mutant that weakened the managed-archive size gate from
`size > max_segment_bytes` to `size > 3 * max_segment_bytes`. The named
production-start regression failed with `Cmd.Start` reached (exit 1), the
source was restored from a task-scoped copy, and the named test reran green.
This recovery inspected and accepted that exact attached evidence; it did not
rerun the source mutant. All validation rows above were run directly by this
recovery.

## Files added by this recovery

- Production-entry zero/overflow negative coverage in
  `tools/agents-infra/internal/infra/pi_test.go`.
- Validation recovery entry in `LOGBOOK.md`.

The existing revision 2 production fix, tests, documentation, and task-owned
revision 1 candidate changes were preserved. Base HEAD remains
`891de4427bb7de6885b8b221f0e2b24a49a8fdc2`.
