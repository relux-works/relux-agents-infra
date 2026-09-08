# TASK-260829-1q31e0 review verdict — revision 5

## Verdict

**Changes requested** for `CR-TASK-260829-1q31e0-5` revision 5.

Route: `to-dev`. This is ordinary implementation rework, not a Stop-The-Line boundary.

- Base commit: `6d051f54440d36e3ca3d132f8d9d1e78d46289de`
- Candidate tree: `c5579cff255e53f42d99f4f41569c0f8bfe189c5`
- Patch SHA-256: `129b8320d45224dc11aca61833be747bb8459fef5577b569a07fd5dff7530e01`
- Reviewer run: `RUN-260829-905b52`
- Repository delta: `present` (17 paths)

The independently reproduced binary diff digest exactly matches the Change Request. `git diff --check` exits 0. Review and validation ran from an archive of the immutable candidate tree, not from the moving worktree.

## Blocking finding

### P1 — The hard-link provenance is caller-mintable and deletes a foreign file

The revision replaces filename/mode-only ownership with a second hard link under `lifecycle-log-ownership/`, but the new evidence remains fully constructible by the same-UID caller whose files the retention boundary must classify:

- `tools/agents-infra/internal/infra/pi_lifecycle_log_posix.go:199-201` deterministically derives the ownership name from public run-state/name inputs.
- `tools/agents-infra/internal/infra/pi_lifecycle_log_posix.go:204-231` protects the namespace only with mode `0700`; the caller running under the profile owner UID can create entries there.
- `tools/agents-infra/internal/infra/pi_lifecycle_log_posix.go:386-420` treats the caller-created two-link inode and deterministic owner name as proven launcher ownership.
- `tools/agents-infra/internal/infra/pi_lifecycle_log_posix.go:622-629` then unlinks both names.
- `tools/agents-infra/internal/infra/pi_session_log.go:63-75` reaches that pruning path before every real standalone/shared lifecycle-log creation.

The shipped negative test at `pi_lifecycle_log_test.go:203-225` mints only the old filename/mode shape. It never attempts to mint the newly accepted hard-link evidence, although the test helper at lines 552-572 demonstrates that ordinary test/caller code can create it.

An expected-red reviewer probe used the production `openPiSessionLogAt` entry point. It created an expired mode-`0600` lifecycle-shaped foreign file, computed the deterministic owner name, and created the same hard-link pair as the launcher. With `max_count=1`, `max_bytes=256`, and `max_age_seconds=60`, opening the next session deleted the foreign path:

```text
=== RUN   TestReviewPiSessionLogOpenPreservesCallerMintedOwnershipPair
    review_lifecycle_forgery_probe_test.go:41: caller-minted foreign file was deleted through self-minted provenance
--- FAIL: TestReviewPiSessionLogOpenPreservesCallerMintedOwnershipPair (0.03s)
```

This is the standard **forged or self-minted evidence** shape. It violates AC 4 and the README/SKILL claim that unproven foreign entries are never deleted. Positive ownership-mutation, aggregate, race, and soak evidence does not close this deletion-authority bypass.

## Required rework

1. Replace deterministic same-UID-mintable hard-link shape with deletion authority a foreign caller cannot construct, or conservatively preserve files when launcher provenance cannot be distinguished.
2. Add a production-entry negative test that mints the complete currently accepted evidence (public filename, mode, link count, inode identity, and ownership-namespace link) and requires preservation/refusal.
3. Prove the replacement by narrowing its authority, not only by deleting the ownership check; keep the real `openPiSessionLogAt` standalone/shared call site in scope.
4. Reconcile README/SKILL claims and append a corrective `LOGBOOK.md` entry. The reviewer did not edit source because this role is read-only.

## Validation evidence

- Immutable patch identity: reproduced SHA-256 `129b8320...530e01`; `git diff --check` exit 0.
- Existing focused lifecycle/config/status suite, uncached: exit 0. Eight-week result: `managed_count=5/5`, `managed_bytes=405/420`, `expired_count=0`, `within_policy=true`. The print-config tests skipped because the official Pi acceptance asset was unavailable in the immutable archive.
- Focused lifecycle/config/status race suite: exit 0 with the same soak result and the same asset-bound skips.
- Expected-red complete-provenance probe: exit 1, reproducing deletion of the caller-minted foreign file through `openPiSessionLogAt`.
- Full uncached Go suite on the immutable candidate: exit 0 (`tools/agents-infra` 75.072s; `internal/infra` 144.470s).
- `go vet ./...`: exit 0.
- `gofmt -l .` and exact candidate `git diff --check`: exit 0 with no output.
- `GOOS=linux GOARCH=amd64 go build ./...`: exit 0.
- `GOOS=windows GOARCH=amd64 go build ./...`: exit 0.
- No live Pi/model process, service, socket, or endpoint was contacted.

Evidence log SHA-256 values:

- focused existing: `df9cb3f689224a69a0cbc2c8e1ffbe7097119d45aa9146caeb2be39690cb76f5`
- focused race: `71aa4f9b9bea3f7cd3116d418d3dd69f0e8e05ce54e6eb7ffc3e958c00badca9`
- expected-red probe: `c78cd45a141a7f7cd5f38c680a1709c9a705e9a71a52c6fb79b254f3f7d26fa2`
- full Go suite: `4680183a2c00b44bbeab210d291022433a34d8af4546a6eed591577f6eb41a24`
