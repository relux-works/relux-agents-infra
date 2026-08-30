# TASK-260830-2hc5r2 revision 7 fresh-base blocker

## Constraint

The managed Story workspace was not reprovisioned at the recovery brief's required current trunk. The specialist assignment forbids switching, rebasing, merging, or deleting the managed Story branch; workspace refresh remains orchestrator-owned.

## Evidence

- `git rev-parse HEAD` = `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- `git rev-parse main` = `436760d62f4ea451cf49614ff7e40109d96915b3`.
- `git rev-list --count HEAD..main` = `2`; `git rev-list --count main..HEAD` = `0`.
- `git log HEAD..main` names `0f26660` and merge `436760d`; their repository delta is the 31-line trunk `LOGBOOK.md` block the recovery brief requires preserving.
- `task-board worktree status STORY-260830-2vrhg1` reports Story tip `3295c7da...` and Change Request revision 6 as `stale`.
- The backup was materialized from the board as `.temp/TASK-260830-2hc5r2/kv-rev6-task-delta.patch`; SHA-256 is exactly `350123ecbc83a81a0a8b2c2b71c9f92486aeb7c211810ad0b6e990d7793861f5`.
- The working tree already contained all 16 patch paths before this run attempted any repository edit.
- `git apply --check --exclude=LOGBOOK.md ...` exits `1` because all 15 non-logbook hunks are already present.
- `git apply --reverse --check --exclude=LOGBOOK.md ...` exits `0`.
- Exact byte comparison of the current non-logbook `git diff --binary` with the patch's non-logbook section exits `0`.
- `.configs/codex-config.toml` is absent from both the current delta and the 16-path backup.
- Current `LOGBOOK.md` contains the Story's additive 2026-08-30 block only because its base is pre-trunk; it does not yet contain trunk's required 31-line block.

## Failed assumption

The recovery brief states that the fresh workspace provisions at current trunk `436760d`. The managed workspace demonstrably remains at the stale pre-refresh tip `3295c7da` with the prior task delta restored in place.

## Safe option and recommendation

The orchestrator should release this run/workspace, refresh or reprovision the managed Story branch at exact `436760d62f4ea451cf49614ff7e40109d96915b3`, then restore the verified backup. The 15 non-logbook paths must remain byte-identical; resolve only `LOGBOOK.md` by keeping trunk's and the Story's additive blocks under one `## 2026-08-30` heading.

Do not publish revision 7 from the current tip: calling it a fresh-base revision would be false, and adapting the dirty workspace would violate both the recovery brief and the managed-worktree ownership boundary.
