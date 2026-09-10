# TASK-260911-1diuhg review verdict — rev1 — changes_requested

## Verified as solid

- **Real host state**: `pipx list` on this machine shows `mlx-lm-relux` (mlx-lm 0.32.0) installed
  non-editable. `direct_url.json` for the installed dist-info reads
  `{"vcs_info": {"commit_id": "45a472f2d0cda166b7ffe1a80fe50dd9621f4303", ...}}` with no
  `dir_info.editable` — matches the task's pin exactly, confirmed independently, not taken on
  claim.
- **Gate mechanism**: `PinnedDistribution` config (`config.go`), `VerifyPinnedDistribution`
  (`pinned_distribution.go`), wired into the production `Doctor()` function
  (`run.go:323-327`) that backs `model-harness doctor`. Ran `go build ./...`, `go vet ./...`,
  and `go test ./internal/modelharness/... -v` myself: all pass (11/11 pinned-distribution/doctor
  tests green).
- **Narrowing mutant verified independently, not taken on the report's word**: applied
  `meta.VCSInfo.CommitID != pin.Commit` → `meta.VCSInfo.CommitID[:7] != pin.Commit[:7]` by hand,
  reran `TestVerifyPinnedDistributionRefusesCommitSharingOnlyAPrefix` — it failed as expected
  (`error = <nil>, want drifted refusal`), then reverted and confirmed `git diff` on the file is
  empty. The gate is real, not a delete-only mutant dressed up as evidence.
- **LOGBOOK finding**: the producer discovered via `git merge-base` that the pinned commit
  `45a472f` does **not** actually contain `9150698` (the generation-recovery fix the story
  believed it carried) — a real, well-evidenced gap in the story's own premise, correctly
  surfaced rather than silently ignored, and correctly *not* used to block this task's
  mechanism-only scope (the commit-value decision belongs to a human in the separate `mlx-lm`
  fork repo).

## Finding: the gate is never wired to the actual qwen local-model target

Task scope is explicit: *"pipx spec + agents-infra setup/verify **for the qwen local-model
target**"*. The live, currently-deployed config for that target is
`/Users/alexis/src/.agents/.configs/model-harness.toml` (found by grepping for
`profiles.qwen-local` outside the repo — it is the only match, its `executable` points at the
real reinstalled `mlx_lm-relux.server` binary, `--model` points at the real local model path).
This file was **not modified by the change** (mtime 2026-09-02, before this task) and its
`[profiles.qwen-local]` stanza has no `pinned_distribution` sub-table.

Reproduced directly:

```
$ model-harness doctor qwen-local --host 127.0.0.1 --port 18011 \
    --config /Users/alexis/src/.agents/.configs/model-harness.toml
profile=qwen-local mode=local executable=/Users/alexis/.local/bin/mlx_lm-relux.server status=ok
```

No mention of `pinned_distribution` in the output — because the profile doesn't declare it,
`Doctor()` skips `VerifyPinnedDistribution` entirely (`run.go:323`: `if plan.PinnedDistribution
!= nil`). This is the "guard that no production path invokes has never run" shape: the gate is
correct and well-tested in isolation, but for the actual qwen deployment it is currently dead —
an operator could reinstall `mlx-lm-relux` editable tomorrow and `model-harness doctor qwen-local`
would still report `status=ok`.

The results.md "Live CLI smoke" section shows `doctor` reporting the pin check passing/failing,
but against a redacted `.../config.toml` path — not the real deployed file at
`/Users/alexis/src/.agents/.configs/model-harness.toml`. The README's example TOML block uses the
real profile name (`qwen-local`) and the real venv path/commit verbatim, which reads as if it
describes the live config, but it was never actually applied there, and nothing in the README or
results.md flags this as a deliberate, documented operator follow-up (contrast with the pipx
reinstall and "no runtime behaviour change" framing, which *are* explicitly called out as
operator-owned facts already done/deferred).

This directly undercuts the task AC: *"verify local passes on the pinned install and fails on an
editable install (negative control)"* was only demonstrated against a synthetic/temp config, not
the real target the task names in its own Scope line.

## Requested rework

1. Add `[profiles.qwen-local.pinned_distribution]` to
   `/Users/alexis/src/.agents/.configs/model-harness.toml` (package `mlx_lm`, the real
   `site_packages` path, commit `45a472f2d0cda166b7ffe1a80fe50dd9621f4303` — exactly what
   README.md already documents).
2. Re-run `model-harness doctor qwen-local` against that real file and capture the `status=ok`
   output showing `pinned_distribution` is now actually evaluated.
3. Demonstrate the negative control against the real file too: temporarily reinstall
   `mlx-lm-relux` editable (or otherwise perturb the real dist-info) and show `doctor` now
   refuses it, then restore the non-editable pinned install and reconfirm `status=ok`. This
   closes the loop the task was scoped to deliver — right now the negative control only exists
   against fabricated stand-ins.
4. If wiring the live config is intentionally being deferred as an operator action, say so
   explicitly in the README/results with a concrete next step, the same way the pipx reinstall
   and "deploy is an operator step" points are already called out — don't leave it implicit.

repeat-of: none
