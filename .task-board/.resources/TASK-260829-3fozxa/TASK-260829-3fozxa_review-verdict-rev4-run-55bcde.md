# TASK-260829-3fozxa Review Verdict — CR revision 4

## Verdict

Changes requested for `CR-TASK-260829-3fozxa-4` revision 4. Route `TASK-260829-3fozxa` to `to-dev`.

The log-rotation implementation fixes the revision-1 restart bypass and passes its focused acceptance checks, but revision 4 is still bound to historical base `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`. It cannot be accepted as the delivery tree against refreshed `main` / `origin/main` `675f77ed63376320ed1213f46f9462a299c0abaf`.

## Immutable candidate identity

- Change Request: `CR-TASK-260829-3fozxa-4`, revision `4`, state `ready`.
- Base OID: `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`.
- Candidate tree OID: `6d63fe072ad717eb4d2b42353fe679340269e3ad`.
- Patch SHA-256: `70e0031bd4ff2fc2bfcf5910cfb71bee13d739b93a16e222f52c325961a46f91`.
- The board-owned patch was applied with `git apply --index` in a detached disposable worktree at the exact base; `git write-tree` reproduced `6d63fe072ad717eb4d2b42353fe679340269e3ad` exactly.
- All 17 manifest paths were reviewed. No review mutation remains; final staged tree is still `6d63fe072ad717eb4d2b42353fe679340269e3ad`.

## Required finding F1 — rev4 is stale against refreshed trunk

Severity: delivery-tree / integration gate failure.

Current local `main` and `origin/main` both resolve to `675f77ed63376320ed1213f46f9462a299c0abaf`. Since the CR base, trunk changed four paths that revision 4 also changes:

- `LOGBOOK.md`
- `README.md`
- `tools/agents-infra/internal/infra/pi_shared_supervision_test.go`
- `tools/agents-infra/runtime_main_darwin_test.go`

The canonical Change Request contract states that an accepted revision whose reviewed trunk advanced on any intersecting `changed_paths` is rejected by integration as `integration_base_moved` and demoted to `stale`; validation and acceptance are tree-bound, so manually testing a different composed tree cannot authorize the historical candidate tree.

The independent composition probes reproduce that condition:

- `git merge-tree --write-tree 675f77e <synthetic-candidate-commit>` exits `1` with a `LOGBOOK.md` conflict.
- Direct replay of the exact board patch onto detached `675f77e` using `git apply --index` exits `1`: `patch failed: LOGBOOK.md:5`.
- A review-only union can be constructed as tree `531c6b4662d380dfe955bd8f22c1e67f46c99d57`, and its code composes semantically, but that is not revision 4's candidate tree and no Change Request evidence is bound to it.

Accepting rev4 would therefore create an accepted record that the next integration gate must immediately stale. This is ordinary rework, not a Stop-The-Line blocker.

### Required rework

1. Refresh/rebase the Story worktree onto current authoritative trunk `675f77ed63376320ed1213f46f9462a299c0abaf` (or the newer exact trunk if it advances again).
2. Preserve both trunk's persisted recovery-state work and the log-rotation delta. Keep `restart_not_before` and `half_open` status/reporting tests alongside the explicit rotation caps.
3. Resolve `LOGBOOK.md` as one newest-first union rather than dropping either side.
4. Publish revision 5 from the refreshed base and rerun the configured validation suite on that exact new tree.

## Implementation and adversarial evidence

Revision 1's required code defect is fixed:

- Production path: `RunSharedRuntimeBroker -> startUnauthorizedRuntime -> startUnauthorizedRuntimeWithDependencies -> openSharedRotatingLog -> newSharedRotatingLogWriter -> sharedFilesystemLogSink.Prune`.
- Every managed archive is now `Lstat`-validated as a mode-0600, single-link regular file and refused when larger than the current `max_segment_bytes`, before `command.Start`.
- Missing, zero, and overflowing caps refuse through the production `RunPi` configuration path before provider lookup or cache/runtime-state creation. No numeric code defaults were found.
- The writer splits at the exact cap, retains at most `max_segments-1` archives plus the active segment, uses monotonic zero-padded sequence names for deterministic oldest-first pruning, and keeps the generic writer alive until `command.Wait` completes.

Independent attacks:

- A temporary production-filesystem probe began with an active file exactly at the 4-byte cap, two equal-timestamp archives, and an unrelated file. The next byte rotated to sequence 3, pruned sequence 1, preserved sequence 2, the active file, and the unrelated file, and retained 9 bytes under the 12-byte bound. No sleep or live model/service was used.
- An overlay narrowing mutant widened the archive-size refusal from `size > cap` to `size > cap*3`. `TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart` failed with `runtime command start side effect calls=1 want=0`, killing the bypass-path mutant. The immutable candidate tree was reverified afterward.
- The composed review tree kept trunk's production `restart_not_before` and `half_open` mappings byte-for-byte. Focused rotation, configuration-refusal, restart-policy, ledger-status, and failed-half-open tests passed together.

## Validation results

- Candidate focused rotation/configuration/restart-path suite: pass.
- Candidate `go vet ./...`: exit 0.
- Candidate `go build ./...`: exit 0.
- Candidate `git diff --check` and Go formatting check: clean. The first platform probe used a non-portable BSD `paste` invocation and emitted usage; corrected `GOOS=darwin`, `GOARCH=arm64` was recorded separately.
- Candidate literal `go test ./... -count=1`: exit 1. Root package timed out at 10 minutes in unchanged `TestRunComposeEmitsOneV1DocumentWithoutProviderExecutable/codex`; `internal/infra` also reported one real-Pi side-effect fixture failure. The infra failure passed an isolated uncached rerun. The root failure hung again in an isolated rerun and was explicitly terminated; it is not reported as green.
- Review-only composed tree `go vet ./...` and `go build ./...`: exit 0.
- Review-only composed infra matrix covering rotation plus persisted restart/status behavior: pass.
- A composed root status test again hung in the unchanged stdout-capture path and was explicitly interrupted; no weakened/skip-based run is reported as a validation pass.

The non-green full command independently prevents an acceptance claim, but F1 is sufficient even if the unrelated root test is repaired: revision 4 still is not the tree that can land on refreshed trunk.

## Repository integrity

The producer workspace was not edited. All reconstruction, probes, mutant overlays, and composition experiments were confined to ignored disposable review paths under `.temp/TASK-260829-3fozxa-review/`. The repository `LOGBOOK.md` was not changed by this reviewer because the reviewer contract is read-only; this task-scoped board outcome is the persistent finding record.
