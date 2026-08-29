# TASK-260824-1jjze0 — Review verdict: ACCEPTED

Reviewer run: `RUN-260824-608bf2` (not goal-bound). Read-only review; no code
was modified. All evidence below was re-derived independently from the live
filesystem and a freshly built production binary, not taken from the producer
handoff.

## Independent re-verification

| Check | Method | Result |
| --- | --- | ---: |
| In-scope inventory | own `find` over `/Users/alexis/src`, hidden + ignored, minus `.git`/`.temp`/dep caches | 121 |
| Inventory equals producer's | set diff against `run/inventory.json` | 0 only-mine, 0 only-theirs |
| Exclusion audit | classified all 566 discovered configs by matched excluded dir | 445 excluded, **all under `.temp`**; no real project dropped by `target`/`vendor`/`venv`/etc. |
| Production alias compose | `go build` of worktree `tools/agents-infra`, then compose `--mode primary-session --entrypoint {openai,anthropic,qwen}-infra --project <real dir> --schema-version 1 --json` for all 121 | **363/363 exit 0** |
| Section 5 Pi invariants | parsed my own 121 `qwen-infra` plans | 121/121 satisfy `target.model == profile.model`, `resolved.model == provider + "/" + model`, `resolved.endpoint == profile.base_url == pi.runtime.endpoint`, `reasoning == off` |
| On-disk drift vs applied report | SHA-256 of all 121 current files vs `after_sha256` | 0 drift, 0 missing |
| MCP preservation | parsed backup vs current, compared every `mcp` node at any depth | 0 semantic diffs |
| Non-`agents` preservation | full TOML compare of every non-`agents` top-level table | 0 diffs |
| Canonical result exactness | parsed all 121 current files | 121/121 have `agents == {pi, targets, entrypoints}`, the three exact targets, the exact three entrypoint mappings, and **one** identical `agents.pi` blob |
| Unrelated worktree changes | diffed `dirty_worktrees_before` vs `_after` across 116 repos | 0 non-config status changes (`.agents/` is git-excluded in these repos) |
| Backup/rollback readiness | SHA-256 of all 121 backups vs `before_sha256`, on disk **and** in the archive | 0 missing, 0 mismatch in both copies |
| No runtime migration code | `git diff main` for this task; grep for project-config writes in production Go | task delta is `LOGBOOK.md` only; script is untracked; no production write path |
| Build / tests | `go test ./... -count=1`, `go vet ./...` in the worktree | ok / ok (71.9s + 2.7s + 118.7s), vet clean |

## Gates attacked, not read

- **MLX identity gate.** Ran the script's own negative test at its production
  call site (`perform_apply -> probe_qwen`) with a fake server advertising a
  wrong model: exit 3, `status=refused-before-write`, `writes=[]`, source bytes
  unchanged. The gate also fired for real in run `RUN-260824-5a8707`, which
  refused the whole rollout before any write when the short canonical ID was
  not served. That is production refusal evidence, not a helper assertion.
- **Compose-validation gate.** Built my own fixture with an `agents-infra`
  stub that exits 7. `apply` refused with "apply refused: dry-run failed"
  before any write; the fixture config was byte-identical afterwards.
- **MCP-preservation guard.** Constructed four adversarial shapes that a naive
  "drop every `agents` block" rewrite would silently corrupt. Refused:
  `[agents.codex.mcp]` (raw MCP blocks changed), `mcp` inline key inside
  `[agents]` (MCP TOML semantics changed), `[[agents.codex.mcp]]` (raw MCP
  blocks changed).
- **Canonical identity locks with the absolute-path model.** The operator's
  absolute-path model identity contains `/`, which could have broken the Pi
  `provider/model[:thinking]` decode. Verified against production dispatch:
  exact repeat accepted, `local-qwen/<abs path>` accepted, divergent model
  refused `target_identity_conflict`, `<abs path>:high` refused, `--provider
  other` refused. The lock survives the chosen value.

## Findings (recorded, none blocking)

1. **Latent hole in the one-time script — never fired.** `blocks()` retains the
   pre-first-table preamble verbatim, so a legacy `agents.*` **dotted key**
   written before any table header survives the rewrite. Reproduced:
   `agents.codex.primary_session.model = "x"` is retained alongside the
   canonical tables and the result still parses, yielding
   `agents = [codex, pi, targets, entrypoints]`. Verified this never fired: all
   121 delivered files parse to exactly `{pi, targets, entrypoints}`. Robustness
   gap in an already-executed scratch script, not a delivered defect. Fix
   before any re-run.
2. **Timing-flaky test.** The test helper's default `--probe-timeout 2` makes
   `test_apply_backup_hash_and_rollback_round_trip` fail on a cold first run —
   reproduced once (suite 12.9s, exit 3, "did not serve the exact canonical
   model selector") — then pass on 3 consecutive warm suite runs and 3 isolated
   runs. Timing sensitivity, not a logic defect. The real apply used a much
   larger timeout and its probe took 16.6s.
3. **Operational consequence of the AC, flagged for the operator.**
   `[agents.codex.primary_session]` and `[agents.claude.primary_session]` were
   deleted from 119 configs, including **238 `yolo_mode = true` entries**
   (112x `claude-fable-5`, 111x `gpt-5.6-sol/xhigh`, and others). Confirmed
   live: `agents-infra codex|claude --print-config` in a rewritten project now
   reports `yolo_mode: false (default)` and native model. `codexD` / `claudeD`
   therefore lose yolo mode in every project. This is exactly what the task AC
   ("replaces all other agent configuration") and Contract Section 6 authorize,
   so it is not a defect — but it is a real daily-workflow change the operator
   should be aware of.
4. **Pre-existing, not caused by this task.** `agents-infra verify local` now
   fails in every project with "no generated openai-infra/anthropic-infra/
   qwen-infra launcher". That postcondition is unconditional in
   `internal/infra/runtime_receipt.go:183` (`canonicalTargetLauncherFailures`)
   and does not depend on config content — it comes from TASK-260824-2o4zq8 and
   needs `setup.sh` rerun. The three aliases are still absent from PATH, so the
   rewritten configs are correct but not yet reachable through the vendor
   aliases. Belongs to setup/deployment (TASK-260824-2a4gk3).

## AC disposition

Every AC clause is satisfied and independently re-derived: reproducible
one-time script confined to task scratch, complete inventory, clean dry-run,
exact MCP preservation, exact canonical replacement, production parsing plus
alias compose on every resulting file, no automatic runtime migration code,
unrelated user changes untouched, and reversible per-file backups with recorded
hashes.

Verdict: **accepted**. Repository delta for this task is `LOGBOOK.md` only and
its two entries are factually consistent with the evidence. The commit-owning
mover should commit that scope and make the final `done` transition with
`commit_ack=scope_committed`.
