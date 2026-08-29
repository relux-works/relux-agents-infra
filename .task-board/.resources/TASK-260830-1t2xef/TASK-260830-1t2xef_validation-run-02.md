# TASK-260830-1t2xef validation run 02

## Scope and base

- Repository delta: `task-board.config.json` only (`90` insertions, `8` deletions).
- Fresh fetch command: `git fetch origin main`.
- Selected Story base, `HEAD`, local `main`, fetched `origin/main`, and merge base: `b78498bf98c05175db10bb341aee621e53de4881`.
- Final ahead/behind count: `0 / 0`.
- No commit or integration was performed by the producer.

## Effective policy

- Contract: `spawn-policy-v4`.
- Preferred provider system: exclusive Codex.
- Sole admitted pair: `codex/gpt-5.6-sol/high`.
- Codex `fast_mode`: `true`, provenance `spawn.ceilings.codex.fast_mode`, source `configured`.
- Claude ceiling: absent (`configured=false` in the effective projection).
- All eleven workload classes resolve to one recommendation: `codex/gpt-5.6-sol/high`.
- Project config schema exposes all eleven workload-class values.
- Candidate validation used the installed task-board with `TASK_BOARD_CONFIG=$PWD/task-board.config.json`; this is required because the spawned worktree keeps the authoritative external `TASK_BOARD_DIR`, whose default config path otherwise points at the control root.

## Production entry points and negative evidence

- `task-board q 'project_config(view=spawn-preflight, ...)'` for `implementation`: exit `0`.
- Batched preflight for all eleven workload classes: exit `0`; machine assertions: exit `0`.
- Full `project_config()` projection and exact policy assertions: exit `0`.
- `schema(operation=project_config)` projection: exit `0`.
- Claude provider attack through `project_config(view=spawn-preflight, agent=claude, ...)`: exit `1`, expected refusal `agent_not_allowed_by_preferred_agentic_system`.
- Non-admitted `codex/gpt-5.6-terra/high` attack through the real `task-board spawn` entry point using a nonexistent task ID: exit `1`, expected refusal `workload_class_pair_unavailable_in_snapshot`; no task lookup mutation, RUN creation, or model runtime launch occurred.
- Missing workload transport attack through the real `task-board spawn` entry point using a nonexistent task ID: exit `1`, expected refusal `workload_class_transport_required`; no model runtime launch occurred.

## Repository gates

- `jq -e . task-board.config.json`: exit `0`.
- `git diff --check`: exit `0`.
- `git diff --exit-code -- . ':(exclude)task-board.config.json'`: exit `0`.
- Configured `cd tools/agents-infra && go test ./... -count=1`, first run: exit `1`. One unrelated Pi readiness timing subtest did not observe its temporary counter before the runtime bound.
- Exact failing production-entry test rerun with `-count=1`: exit `0`.
- Full `go test ./internal/infra -count=1` rerun: exit `0`.
- Configured `cd tools/agents-infra && go test ./... -count=1`, clean second run: exit `0`.
- Configured `cd tools/agents-infra && go vet ./...`: exit `0`.
- No live model runtime was contacted by any validation.

## Evidence files

- `.temp/TASK-260830-1t2xef/project-config-after-01.json`
- `.temp/TASK-260830-1t2xef/project-config-assertions-01.log`
- `.temp/TASK-260830-1t2xef/project-config-schema-01.json`
- `.temp/TASK-260830-1t2xef/spawn-preflight-implementation-01.json`
- `.temp/TASK-260830-1t2xef/spawn-preflight-all-workloads-01.json`
- `.temp/TASK-260830-1t2xef/spawn-preflight-all-workloads-assertions-01.log`
- `.temp/TASK-260830-1t2xef/refusal-claude-01.json`
- `.temp/TASK-260830-1t2xef/refusal-nonadmitted-pair-01.log`
- `.temp/TASK-260830-1t2xef/refusal-nonadmitted-effort-01.log`
- `.temp/TASK-260830-1t2xef/go-test-agents-infra-01.log`
- `.temp/TASK-260830-1t2xef/go-test-infra-readiness-rerun-01.log`
- `.temp/TASK-260830-1t2xef/go-test-internal-infra-rerun-01.log`
- `.temp/TASK-260830-1t2xef/go-test-agents-infra-02.log`
- `.temp/TASK-260830-1t2xef/go-vet-agents-infra-01.log`
