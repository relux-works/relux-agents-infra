## Status
done

## Review
required

## Task Class
research

## Estimate
estimated(fibonacci(5))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Trace the pinned Pi tool execution and permission path from primary source
- [x] Check current upstream native flags, settings, extensions, RPC, and custom-tool APIs
- [x] Reproduce candidate controls locally without starting the model runtime
- [x] Compare autonomy, security, compatibility, and maintenance tradeoffs
- [x] Recommend a concrete fail-closed agents-infra contract and negative test plan
- [x] Findings written to file
- [x] Key aspects highlighted
- [x] Fact-checking performed — claims verified, sources cited
- [x] Findings linked on the board as a new task-scoped outcome resource
- [x] All questions from task description answered
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Evaluate sudo/admin wrapping versus a narrowly allowlisted privileged helper boundary
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [ ] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn selection rationale tuple: {"role":"researcher","pair":"gpt-5.6-sol/xhigh","text":"This research must reconcile pinned and current upstream Pi authorization internals, extension APIs, RPC behavior, and prompt-injection/security tradeoffs into an implementable unattended-spawn contract; gpt-5.6-sol/xhigh is justified by the breadth of primary-source code tracing and the high cost of falsely claiming tool autonomy."}
spawn selection rationale for gpt-5.6-sol/xhigh: This research must reconcile pinned and current upstream Pi authorization internals, extension APIs, RPC behavior, and prompt-injection/security tradeoffs into an implementable unattended-spawn contract; gpt-5.6-sol/xhigh is justified by the breadth of primary-source code tracing and the high cost of falsely claiming tool autonomy.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] researcher (codex) (run=RUN-260825-499c64, max_parallel=20)
spawn run started: [analyst] researcher (codex) (run=RUN-260825-499c64)
Research artifact attached as TASK-260825-1q1987_pi-unattended-tool-authorization.md; commits f08f6e1 and 750df55. Pinned 0.84.2 no-model flag/extension/replacement/RPC probes each exited 0. Agent-core block test exited 0. Coding-agent block test initially exited 1 before collection because generated provider JSON was absent; official hydration exited 0 and the exact rerun exited 0. Pinned-to-current hook diff exited 0. Staged diff check initially exited 2 for two trailing spaces, then exited 0 after correction. Supplemental marker gate initially exited 1 for a case-mismatched assertion, then exited 0 with the exact marker. Link gate initially exited 1 for private unauthenticated task-board URLs; those were replaced by repo@commit:path:line citations and the full public-link rerun exited 0. Recommendation: explicit tracked-session yolo plus exact native --tools allowlist, --no-extensions, separate --no-approve trust, no raw RPC bash; no root/unrestricted sudo, and no claim of task-board Pi spawn until a shipped Pi adapter exists.
agent completed: [analyst] researcher (codex) (exit=0)
spawn run completed: codex (run=RUN-260825-499c64, pid=67160, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5/high","text":"The research changes the unattended execution and privilege boundary for Pi, distinguishes native tool execution from extensions/RPC and sudo, and must be checked against pinned upstream source plus local probes before implementation; Claude Opus 5/high provides independent provider review at appropriate depth."}
spawn selection rationale for claude-opus-5/high: The research changes the unattended execution and privilege boundary for Pi, distinguishes native tool execution from extensions/RPC and sudo, and must be checked against pinned upstream source plus local probes before implementation; Claude Opus 5/high provides independent provider review at appropriate depth.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-31-ge70f953; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260825-58ee90, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260825-58ee90)
Review verdict: accepted (CR-TASK-260825-1q1987-1 rev1, reviewer RUN-260825-58ee90). Independently re-derived every load-bearing claim from pinned v0.84.2 (914cf147, verified tag) and current 0.84.3 (8fa7eebd) rather than accepting the report: beforeToolCall preflight at agent-loop.ts:600-668 with both call sites at :452/:507; --approve sets only projectTrustOverride (args.ts:205-208, 3 grep hits); --tools filters custom AND builtin in _refreshToolRegistry; --no-extensions drops global and project discovery at resource-loader.ts:451/:555; allowed-tools has exactly one repo hit, docs/skills.md:148; all 31 RPC commands enumerated, none mutates the registry; direct RPC bash bypasses the hook at rpc-mode.ts:559-580; user_bash is fail-OPEN (runner.ts:955-980 catches and continues, UserBashEventResult has no block field); agent.ts/agent-loop.ts/types.ts byte-identical pinned to current; no new approval flag in current. Probe logs re-read, not trusted: policy-probe.ts registers an always-blocking tool_call handler and rpc-direct-bash.log still shows success/exitCode 0, and approve-override.log shows bash source=auto. Worktree HEAD tree equals candidate 1d2845fb and status is clean, so the inspected artifacts are the reviewed candidate. Delta is docs-only (2 files, +649/-0), so no project suite is implicated and none was run or claimed. Negative-shape audit clean: the initial coding-agent exit 1 is retained as a real setup failure, the task-board Pi adapter gap is refused rather than claimed (6.1), and the test plan uses narrowing mutants (11.2 items 14-15). Two non-blocking advisories recorded in the verdict: 9.1 item 3 does not name the source of truth for the known Pi tool-name catalog (fails safe), and the extension-replacement-under-allowlist caveat is code-derived rather than probe-derived (11.2 item 8 already schedules the negative).
Reviewer regression check (docs-only delta, run for completeness rather than because the delta could affect it): tools/agents-infra `go build ./...` exit 0; `go test ./internal/infra/...` ok, 84.0s. Verdict artifact updated to report these as observed results. Worktree remains clean; reviewer made no repository change.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260825-58ee90, pid=47507, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260825-1q1987_spawn-log_-analyst--researcher--codex-_RUN-260825-499c64.log](file://TASK-260825-1q1987/TASK-260825-1q1987_spawn-log_-analyst--researcher--codex-_RUN-260825-499c64.log) — System spawn log captured by task-board
- [TASK-260825-1q1987_pi-unattended-tool-authorization.md](file://TASK-260825-1q1987/TASK-260825-1q1987_pi-unattended-tool-authorization.md) — Primary-source Pi 0.84.2/current unattended tool authorization research, local no-model reproductions, privilege and task-board integration boundaries, recommendation, and negative test plan.
- [TASK-260825-1q1987_change-request_rev1.patch](file://TASK-260825-1q1987/TASK-260825-1q1987_change-request_rev1.patch) — Change Request CR-TASK-260825-1q1987-1 revision 1 candidate patch (repository_delta=present, 2 changed paths)
- [TASK-260825-1q1987_spawn-log_-reviewer--reviewer--claude-_RUN-260825-58ee90.log](file://TASK-260825-1q1987/TASK-260825-1q1987_spawn-log_-reviewer--reviewer--claude-_RUN-260825-58ee90.log) — System spawn log captured by task-board
- [TASK-260825-1q1987_review-verdict.md](file://TASK-260825-1q1987/TASK-260825-1q1987_review-verdict.md) — Reviewer verdict (accepted) for CR-TASK-260825-1q1987-1 rev1: independent re-derivation of every load-bearing Pi authorization claim from pinned v0.84.2 and current upstream source, probe-log re-verification including the RPC direct-bash bypass and fail-open user_bash hook, negative-shape audit, worktree regression check (go build/go test internal/infra exit 0), and two non-blocking advisory notes.

## Created
2026-08-25T20:27:09Z

## Last Update
2026-08-25T22:28:28Z

## Assigned To
[reviewer] reviewer (claude)
