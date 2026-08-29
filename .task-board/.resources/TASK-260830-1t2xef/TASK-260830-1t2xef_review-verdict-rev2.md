# TASK-260830-1t2xef review verdict

Verdict: **accepted** for Change Request `CR-TASK-260830-1t2xef-2`, revision 2.

## Candidate identity and scope

- Fresh `git fetch origin main` resolved `origin/main` to `b78498bf98c05175db10bb341aee621e53de4881`, exactly the CR base OID and current worktree `HEAD`.
- Candidate tree is `9a9623b285817fb1ff09096555cc81697e1930b0`.
- The CR patch resource SHA-256 is `bd0a0cfab71eec25c8dbe119fe50e547597e91e19be321ec14b8e6ee5215604f`, matching the supplied digest.
- The downloaded patch is byte-identical to `git diff --binary <base> <candidate-tree>`.
- The exact delta contains only `task-board.config.json`.
- The worktree file blob `fa77f9362f7ae6dba6313fc054c6e25b2c418fbd` matches the blob in the candidate tree after all review probes.

## Policy assertions

The installed `task-board` build (`0.24.3-172-g063197b1`) was pointed explicitly at the candidate with `TASK_BOARD_CONFIG=<worktree>/task-board.config.json`; the authoritative board remained selected through `TASK_BOARD_DIR`. This avoids accidentally validating the control-root checkout's old v2 config while preserving the authoritative board.

- `spawn.ceilings.contract_version`: `spawn-policy-v4`
- effective provider mode: exclusive, allowed providers exactly `[codex]`
- Codex admitted pairs: exactly `gpt-5.6-sol/high`, entries authority
- `resolution_model`: `gpt-5.6-sol`
- `adjustment_confirmation`: `none`
- `fast_mode`: `true`; source `configured`; provenance `spawn.ceilings.codex.fast_mode`
- Claude ceiling: unconfigured/absent
- workload classes: exactly all eleven (`unified`, `architecture`, `implementation`, `mechanical`, `debugging`, `testing`, `review`, `research`, `documentation`, `migration`, `operations`)
- every class has exactly one recommendation and available pair: `codex/gpt-5.6-sol/high`
- schema projection exposes the same eleven-value `workload_class` enum

All eleven targeted `project_config(view=spawn-preflight, role=developer, agent=codex, workload_class=...)` calls passed the exact-provider, pair, fast-mode provenance, determinate-limit-read, and recommendation assertions.

## Gate-defeat evidence

Production entry point: `task-board spawn`; workload enforcement is reached through `runSpawn` and `cmd/workload_projection.go`.

- **Bypass path around the check:** an explicit Claude request was refused with `agent_not_allowed_by_preferred_agentic_system` before task lookup.
- **Narrowed/out-of-set admission:** `codex/gpt-5.6-terra/high` against the exact candidate snapshot was refused with `workload_class_pair_unavailable_in_snapshot`.
- **Absent evidence treated as satisfied:** omitting workload transport was refused with `workload_class_transport_required`.
- Positive control: the admitted pair with the exact fresh snapshot digest passed policy enforcement and reached only the deliberately nonexistent task lookup (`goal_scope_invalid`); no run or model process was launched.
- **Failure to read is not absence:** malformed candidate JSON failed closed with `project config refused` / `unexpected end of JSON input`; it did not fall back to defaults.
- Mutants setting `fast_mode=false` and adding a Claude ceiling both caused the corresponding acceptance assertions to fail. Adding a Claude ceiling still could not bypass exclusive Codex provider enforcement.

No live model runtime was contacted. An auxiliary no-workload mutant showed ceiling selection occurs after task lookup on that path, so it is intentionally not claimed as ceiling-refusal evidence.

## Repository validation

- `jq -e . task-board.config.json`: pass
- `git diff --check <base> <candidate-tree>`: pass
- `cd tools/agents-infra && go test ./... -count=1`: pass (all packages; `internal/infra` 247.337s)
- `cd tools/agents-infra && go vet ./...`: pass
- final base, changed-path, blob, and exact-patch integrity checks: pass

No review findings remain. The configuration-only solution matches the task acceptance criteria and is ready for orchestrator checkpoint/integration.
