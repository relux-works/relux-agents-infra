## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(13))

## Blocked By
- TASK-260916-38vqh4

## Blocks
- (none)

## Checklist
- [x] agents-infra claude|codex (and openai-infra/anthropic-infra, target-yolo aliases, local shim) print the deprecation line to stderr and exit 1 with no side effects; tests cover it
- [x] Instruction sync/@ rendering for setup/refresh, skills fan-out, bundled MCP servers and MCP composition are removed from setup/refresh-links/doctor/verify and the launchers; v1 compose/prepare contracts consumed by task-board keep passing their golden tests
- [x] Residual kept and tested: claude-settings.json linking, codex config.toml merge, .rules, pi local-model runtime incl. runtime quarantine/unquarantine, broker, runtime-launch, attachments contract
- [x] README rewritten to state what remains and why; CHANGELOG entry and version bump; go build/vet and narrow package tests green (quoted exit codes); handoff via task-board
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"muse-spark-1.3-contributor/max","text":"coding producer per operator directive muse-spark:max"}
spawn selection rationale for muse-spark-1.3-contributor/max: coding producer per operator directive muse-spark:max
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260916-2cd636, max_parallel=8)
spawn run started: [implementer] developer (muse) (run=RUN-260916-2cd636)
Slice A ready for review. Deprecation stubs (exact stderr, exit 1, no side effects) + setup/refresh removals + v1 prepare compat path + README/CHANGELOG. All package tests green (bounded selections, quoted exit codes), 2/2 narrowing mutants killed. Findings: piCatalogPaths/resolvePiNativeExecutable dead on HEAD (pre-existing, untouched); SKILL.md still documents old surface (follow-up); version via release-owner v2.0.0 tag (no version file in repo). Windows executable paths + live provider runs are stated blind spots.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260916-2cd636, pid=67227, exit=0)
spawn autonomous recovery: run RUN-260916-2cd636 queued successor RUN-260916-8aaa25 (attempt 1/3, model=muse-spark-1.3-contributor): Change Request construction for TASK-260916-3q6z2c failed: Change Request CR-TASK-260916-3q6z2c-1 revision 1 validation failed at command 1/1 (1-based) with exit code 1; log resource TASK-260916-3q6z2c_change-request_rev1-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (muse) (run=RUN-260916-8aaa25)
spawn selection rationale tuple: {"role":"developer","pair":"muse-spark-1.3-contributor/max","text":"rework: add CI + remote gate so validation runs on GitHub; muse-spark:max"}
spawn selection rationale for muse-spark-1.3-contributor/max: rework: add CI + remote gate so validation runs on GitHub; muse-spark:max
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260916-16818e, max_parallel=8)
Slice A implemented and verified this session. Deprecation: 8 tests/87 subtests green incl 46 executable entrypoint×shape probes (exit 1, exact stderr, empty stdout, no side effects). Removals: skills fan-out, instruction sync/@ rendering from setup/refresh, bundled registry shipping, launcher MCP composition; renderer kept only behind prepare v1. Residual: settings/config/rules/Pi/runtime/attachments/compose+prepare goldens green. Full module verified per-package: main 96 pass/10 env-skips/0 fail; infra 493/493 green; attachments+modelharness ok. Both narrowing mutants killed. SKILL.md migrated (was left stale). Host anomaly: fresh-binary first-exec stalls minutes in dyld assessment (no syspolicyd; 215d uptime) caused transient group timeouts, all proven flakes via isolated + pristine-HEAD reruns. Results: TASK-260916-3q6z2c_results.md.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260916-8aaa25, pid=83200, exit=0)
spawn autonomous recovery: run RUN-260916-8aaa25 queued successor RUN-260916-e29c9d (attempt 2/3, model=muse-spark-1.3-contributor): Change Request construction for TASK-260916-3q6z2c failed: Change Request CR-TASK-260916-3q6z2c-2 revision 2 validation failed at command 1/1 (1-based) with exit code 127; log resource TASK-260916-3q6z2c_change-request_rev2-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (muse) (run=RUN-260916-16818e)
Rework 1 (RUN-260916-e29c9d): added scripts/remote-gate.sh + .github/workflows/ci.yml; cheap local gates green (build/vet/gofmt/sh -n/yaml/in-process deprecation/compose goldens/both SIGTERM tests isolated, one host-stall flake rerun green); results revision 2 attached.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260916-16818e, pid=40910, exit=0)
spawn autonomous recovery: run RUN-260916-16818e queued successor RUN-260916-7201c8 (attempt 1/3, model=muse-spark-1.3-contributor): Change Request construction for TASK-260916-3q6z2c failed: Change Request CR-TASK-260916-3q6z2c-3 revision 3 validation failed at command 1/1 (1-based) with exit code 1; log resource TASK-260916-3q6z2c_change-request_rev3-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (muse) (run=RUN-260916-e29c9d)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260916-e29c9d, pid=51339, exit=0)
spawn autonomous recovery: run RUN-260916-e29c9d queued successor RUN-260916-f9ef46 (attempt 3/3, model=muse-spark-1.3-contributor): Change Request construction for TASK-260916-3q6z2c failed: Change Request CR-TASK-260916-3q6z2c-4 revision 4 validation failed at command 1/1 (1-based) with exit code 1; log resource TASK-260916-3q6z2c_change-request_rev4-validation.log; retry: fix the failure and complete the producer again; the configured suite will rerun automatically
spawn run started: [implementer] developer (muse) (run=RUN-260916-7201c8)
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260916-7201c8, pid=66879, exit=0)
spawn run started: [implementer] developer (muse) (run=RUN-260916-f9ef46)
agent completed: [implementer] developer (muse) (exit=143)
spawn run RUN-260916-f9ef46 cancelled by operator; operator action required; reason: no operator reason supplied
spawn run completed: muse (run=RUN-260916-f9ef46, pid=87874, exit=143)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-6-astra/low","text":"reviewers run gpt-6-astra:low"}
spawn selection rationale for gpt-6-astra/low: reviewers run gpt-6-astra:low
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=main-7a0a24c; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260916-c350c0, max_parallel=8)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260916-c350c0)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260916-c350c0, pid=89117, exit=0)
spawn autonomous recovery: run RUN-260916-c350c0 queued successor RUN-260916-dd58b8 (attempt 1/3, model=gpt-6-astra): reviewer run RUN-260916-c350c0 remains unsatisfied: reviewer run has no verdict branch while TASK-260916-3q6z2c is reviewing
spawn run RUN-260916-dd58b8 failed because its runner heartbeat expired; operator action required; failure: spawn runner heartbeat expired
spawn selection rationale tuple: {"role":"developer","pair":"muse-spark-1.3-contributor/max","text":"rework after review P1 (wrappers build before refusing); muse-spark:max"}
spawn selection rationale for muse-spark-1.3-contributor/max: rework after review P1 (wrappers build before refusing); muse-spark:max
spawn agent resolution: Agent selection: muse via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: degraded_contract_unavailable; contract=agents-infra.child-launch-composition; provider=muse; schema=1; diagnostic=composition_contract_unavailable; bare child launch retained
spawn queued: [implementer] developer (muse) (run=RUN-260916-d10cf9, max_parallel=8)
spawn run started: [implementer] developer (muse) (run=RUN-260916-d10cf9)
rev6 (rework 4, this session): reviewer P1 fixed — installed deprecated wrappers (openai/anthropic-infra, dange pair, codex-local) and the local cliWrapper now refuse directly before any sibling lookup, mkdir, or go build. Reproduced pre-fix (exit 73 + unexpected-build + build-dir recreation), then verified post-fix: 58/58 hardened executable subtests + 16/16 alias subtests green with failing-go/absent-build fixture and zero snapshot exclusions; both narrowing mutants killed with sibling-pass narrowness; live qwen/pi/version paths intact. Evidence: TASK-260916-3q6z2c_results.md revision 6. Full suite runs on CI via the remote gate after handoff.
agent completed: [implementer] developer (muse) (exit=0)
spawn run completed: muse (run=RUN-260916-d10cf9, pid=99888, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-6-astra/low","text":"reviewers run gpt-6-astra:low; revision 6"}
spawn selection rationale for gpt-6-astra/low: reviewers run gpt-6-astra:low; revision 6
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (codex) (run=RUN-260916-eafce4, max_parallel=8)
spawn run RUN-260916-eafce4 failed; operator action required; failure: queued spawn preparation failed: workload_class_transport_required: pass exactly one of --workload-class or --workload-task-class
spawn selection rationale for gpt-6-astra/low: reviewers run gpt-6-astra:low; revision 6
spawn workload selection: class=review source=explicit policy=spawn.workload_classes pair=codex/gpt-6-astra/low pair_source=explicit match=recommended_rank_1 snapshot=sha256:fb4002876227c0014ec167986468661db08fb763e369aca2c9095779c5170056 rationale="review class; astra low is the only admitted codex pair and matches the operator reviewer policy"
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn queued: [reviewer] reviewer (codex) (run=RUN-260916-c8ac14, max_parallel=8)
spawn run RUN-260916-c8ac14 failed; operator action required; failure: queued spawn preparation failed: workload_class_snapshot_stale: supplied digest does not match current config, registry, schedule/provider, ceiling, or limit evidence; re-query spawn-preflight
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-6-astra/low","text":"reviewers run gpt-6-astra:low; revision 6 (old binary: new build enforces a workload ceremony against the tracked config)"}
spawn selection rationale for gpt-6-astra/low: reviewers run gpt-6-astra:low; revision 6 (old binary: new build enforces a workload ceremony against the tracked config)
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=main-7a0a24c; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260916-5bcb3d, max_parallel=8)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260916-5bcb3d)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260916-5bcb3d, pid=32048, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"gpt-6-astra/low","text":"bound integration run; astra:low"}
spawn selection rationale for gpt-6-astra/low: bound integration run; astra:low
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[muse,codex,claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=main-7a0a24c; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260916-6d7fdc, max_parallel=8)
spawn run started: [implementer] developer (codex) (run=RUN-260916-6d7fdc)

## Precondition Resources
- [b6-inventory-report.md](file://TASK-260916-3q6z2c/b6-inventory-report.md)
- [b6-inventory-verdict.md](file://TASK-260916-3q6z2c/b6-inventory-verdict.md)
- [b6-implement-brief.md](file://TASK-260916-3q6z2c/b6-implement-brief.md)
- [remote-gate-curator.sh](file://TASK-260916-3q6z2c/remote-gate-curator.sh)
- [ci-curator.yml](file://TASK-260916-3q6z2c/ci-curator.yml)
- [b6-rework-1.md](file://TASK-260916-3q6z2c/b6-rework-1.md)
- [b6-rework-2.md](file://TASK-260916-3q6z2c/b6-rework-2.md)
- [b6-rework-3.md](file://TASK-260916-3q6z2c/b6-rework-3.md)
- [b6-review-brief.md](file://TASK-260916-3q6z2c/b6-review-brief.md)
- [b6-rework-4.md](file://TASK-260916-3q6z2c/b6-rework-4.md)
- [b6-integrate-instruction.md](file://TASK-260916-3q6z2c/b6-integrate-instruction.md)

## Outcome Resources
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-2cd636.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-2cd636.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_results.md](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_results.md) — Slice A implementation results revisions 1-6 (rev6: installed wrappers refuse before any build)
- [TASK-260916-3q6z2c_change-request_rev1.patch](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev1.patch) — Change Request CR-TASK-260916-3q6z2c-1 revision 1 candidate patch (repository_delta=present, 20 changed paths)
- [TASK-260916-3q6z2c_change-request_rev1-validation.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev1-validation.log) — Change Request CR-TASK-260916-3q6z2c-1 revision 1 bounded validation log
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-8aaa25.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-8aaa25.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-16818e.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-16818e.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_change-request_rev2.patch](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev2.patch) — Change Request CR-TASK-260916-3q6z2c-2 revision 2 candidate patch (repository_delta=present, 21 changed paths)
- [TASK-260916-3q6z2c_change-request_rev2-validation.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev2-validation.log) — Change Request CR-TASK-260916-3q6z2c-2 revision 2 bounded validation log
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-e29c9d.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-e29c9d.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_change-request_rev3.patch](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev3.patch) — Change Request CR-TASK-260916-3q6z2c-3 revision 3 candidate patch (repository_delta=present, 23 changed paths)
- [TASK-260916-3q6z2c_change-request_rev3-validation.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev3-validation.log) — Change Request CR-TASK-260916-3q6z2c-3 revision 3 bounded validation log
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-7201c8.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-7201c8.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_change-request_rev4.patch](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev4.patch) — Change Request CR-TASK-260916-3q6z2c-4 revision 4 candidate patch (repository_delta=present, 29 changed paths)
- [TASK-260916-3q6z2c_change-request_rev4-validation.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev4-validation.log) — Change Request CR-TASK-260916-3q6z2c-4 revision 4 bounded validation log
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-f9ef46.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-f9ef46.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_change-request_rev5.patch](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev5.patch) — Change Request CR-TASK-260916-3q6z2c-5 revision 5 candidate patch (repository_delta=present, 30 changed paths)
- [TASK-260916-3q6z2c_change-request_rev5-validation.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev5-validation.log) — Change Request CR-TASK-260916-3q6z2c-5 revision 5 bounded validation log
- [TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-c350c0.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-c350c0.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-dd58b8.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-dd58b8.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-d10cf9.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--muse-_RUN-260916-d10cf9.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_change-request_rev6.patch](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev6.patch) — Change Request CR-TASK-260916-3q6z2c-6 revision 6 candidate patch (repository_delta=present, 32 changed paths)
- [TASK-260916-3q6z2c_change-request_rev6-validation.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_change-request_rev6-validation.log) — Change Request CR-TASK-260916-3q6z2c-6 revision 6 bounded validation log
- [TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-eafce4.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-eafce4.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-c8ac14.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-c8ac14.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-5bcb3d.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-reviewer--reviewer--codex-_RUN-260916-5bcb3d.log) — System spawn log captured by task-board
- [TASK-260916-3q6z2c_review-verdict-rev6.md](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_review-verdict-rev6.md) — Revision 6 accepted review: prior P1 resolved, targeted checks and exact-tree hosted CI verified
- [TASK-260916-3q6z2c_spawn-log_-implementer--developer--codex-_RUN-260916-6d7fdc.log](file://TASK-260916-3q6z2c/TASK-260916-3q6z2c_spawn-log_-implementer--developer--codex-_RUN-260916-6d7fdc.log) — System spawn log captured by task-board

## Created
2026-09-16T01:55:12Z

## Last Update
2026-09-16T07:34:05Z

## Assigned To
[implementer] developer (codex)
