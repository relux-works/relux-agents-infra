# BUG-260817-2lpkfh — Reviewer Verdict (RUN-260830-ea6684)

Verdict: **accepted**. Accepted handoff to the commit-owning mover; this reviewer supplies no `commit_ack`.

Run `RUN-260830-ea6684` is not goal-bound and was not launched against a Change Request. Review was read-only for repository source. The endpoint-binding implementation is already contained in ancestor commit `97ebda4a67f9e75667a70cc90081a662590babfd`; the reviewed source paths have no current working-tree delta.

## Gate and production reachability

- Production compose reaches `runCompose -> BuildPrimarySessionLaunchPlan -> buildPiPrimarySessionLaunchPlan -> loadCompositeProjectConfig -> parsePiConfig -> parsePiProfile -> validatePiRuntimeEndpointArgv`.
- `parsePiProfile` validates `runtime.argv` against the canonical `base_url` before compose diagnostics, runtime state creation, listener checks, or child launch.
- The accepted control preserves exactly one spaced `--host 127.0.0.1` and exactly one spaced `--port <base_url-port>` pair.
- Focused parser negatives independently refused wildcard bind, port drift, missing/attached endpoint options, and duplicate port.
- Focused production negatives independently drove both in-process compose and the setup-generated installed CLI. Exact control passed; wildcard host and port drift refused.
- Real llama.cpp `b10470` help exposes only `--host`, `--port`, and `--reuse-port` for endpoint binding; no short host/port alias was found. `--reuse-port` does not change the declared host or port and is outside this bug's divergence acceptance criteria.

## Narrowing evidence

A throwaway copy under `.temp/BUG-260817-2lpkfh-review/port-unbound-mutant-root` kept the gate and host equality intact but narrowed only the port comparison from exact equality to presence. With `-count=1`, `TestRunComposePiRefusesRuntimeEndpointDivergence/runtime_port_drift` failed because production compose admitted the divergent port. This proves the port-equality bound rather than only gate presence.

Attached producer/prior-review evidence additionally supplies independently executed host-equality and exactly-once narrowing mutants plus real-runtime environment-precedence evidence. I accepted those facts as attached evidence; I did not rerun those two mutants or the environment precedence probe in this run.

## Validation rerun by this reviewer

| Gate | Result |
| --- | --- |
| Focused parser negatives, uncached | pass |
| In-process + installed production endpoint negatives, uncached | pass |
| Port-equality narrowing mutant | expected fail on `runtime_port_drift` |
| `go test ./... -count=1` | pass; root `219.708s`, infra `376.061s`, attachments `2.342s`, modelharness `2.151s` |
| `go vet ./...` | pass |
| `go build` plus binary help smoke | pass |
| `gofmt -l tools/agents-infra` | empty |
| `agents-infra verify global` | pass |
| Story-worktree local setup from verified installed runtime + `verify local` | pass |
| `local-models` local setup + `verify local` | pass after project-specific config restoration described below |
| Qwen and Muse `pi-infra --print-config` | pass; exact `127.0.0.1:18011` / `127.0.0.1:18012` argv and endpoint |
| `local-models/.scripts/test-managed-pi-profiles.sh` | pass |
| `git diff --check`, cached diff check | pass |
| `task-board validate` | pass |

The producer's task-scoped artifact supplies the earlier successful full `./setup.sh` bootstrap/rebuild. I did not rerun that global mutating bootstrap; I reran the installed global verification, project-local setup/verify, build, and production-entry tests.

## Setup probe anomaly and recovery

The first Story-worktree `verify local` correctly failed because that isolated worktree had never been installed. `setup local --source-dir <same-worktree>` then correctly refused a self-sync. Installing from the verified global runtime succeeded and final worktree verification passed.

Running downstream setup from the generic installed runtime replaced `local-models/.agents/.configs/project-config.toml` with the generic Qwen-only config, so the Muse print-config and profile validator initially failed. This was reviewer-induced generated-state drift, not a defect in the endpoint gate. The exact pre-run project config survived in two independent control fixtures with matching SHA-256 `702cf1c6ccf4a5c12c38775343e7af5d36c42af2a7f954404a22dc76a05be703`; only that generated config was restored through `apply_patch`, its hash was rechecked, and downstream verify, both print-config calls, and the profile validator then passed. No tracked downstream file was changed (the downstream repository has no commits and all content was already untracked).

Story-worktree setup temporarily changed only the generated source comment in tracked `AGENTS.md`; it was restored to its known-clean pre-run `HEAD` value. Final tracked status matches the pre-review list exactly; unrelated staged/unstaged Story work was preserved.

## Acceptance criteria

- Wildcard runtime bind and runtime/base_url port drift refuse through production compose: **met and attacked**.
- Accepted argv expresses the exact `127.0.0.1` base_url endpoint: **met**.
- Installed/production entry paths have negative tests plus a narrowing control: **met and independently rerun**.
- Pi configuration and operator documentation state the invariant: **met**.
- Focused/full Go, vet, build, setup/install, global/local verify, downstream profile validation, formatting, diff, and board gates: **met**.

The existing `LOGBOOK.md` entries `2109 — Managed Pi Endpoint Claim Bound To Runtime Argv` and `2132 — Pi Endpoint Gate Survives Attack; llama Env Override Refuted` already persist the task's root cause, fix, negative evidence, and known non-blocking fixture anomaly; no duplicate logbook entry was added.
