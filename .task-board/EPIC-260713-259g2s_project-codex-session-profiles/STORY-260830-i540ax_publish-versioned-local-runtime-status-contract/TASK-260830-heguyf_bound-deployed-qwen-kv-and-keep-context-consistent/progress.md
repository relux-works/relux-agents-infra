## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(3))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] A configuration where context_window exceeds the KV bound is refused at the production entry point, with an error naming both values
- [x] Raising context_window without raising the bound fails closed rather than silently truncating
- [x] A test drives the real resolution path for both the refusing and the admitting case, not a helper in isolation
- [x] Removing the guard makes the refusal test red
- [x] The prompt-cache-bytes versus max-kv-size distinction is documented so it cannot be mistaken again
- [x] The relationship is documented where an operator setting context_window will see it
- [x] context_window values are unchanged by this task and other profile settings are preserved exactly
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Profile resolution refuses a context_window larger than the KV bound at the production entry with an error naming both values; a consistent pair is admitted (test drives the real resolution path)
- [x] Raising context_window without raising --max-kv-size fails closed (negative test); the deployed qwen-local profile pair (75000 / 76800) is admitted (golden)
- [x] Relationship documented where an operator sets context_window; narrowing mutant that drops the check fails a named test; full tools/agents-infra suite green
- [x] In a managed Story worktree the candidate is left UNCOMMITTED in the worktree for the handoff to snapshot — never commit on the Story branch. A producer commit moves the branch tip off the recorded checkpoint and the handoff refuses with change_request_candidate_committed_past_checkpoint; repair with `git reset --soft <checkpoint_oid>` before completing again.
- [x] Every command, message, state, or refusal named in the AC is driven through the production entry point by a named committed test, or is declared a stated bound. Report coverage as a ratio — `n of m AC rows driven` — and name the production call site for each. Prose in place of the ratio is not evidence.
- [x] Every gate ships at least one NARROWING mutant — the gate stays present and is weakened to admit exactly one member of the class it must reject, and a named test must fail. A delete-only mutant proves only that the gate exists and is not accepted as evidence.
- [x] A gate that inspects source text is additionally attacked by a mutant that PRESERVES the searched-for token and changes behavior, and the mutant harness executes the behavioral suite, not only the static checker.
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"Highest admitted effort on the only admitted model; the guard must be reachable from the real loader, not a helper."}
spawn selection rationale for claude-sonnet-5/high: Highest admitted effort on the only admitted model; the guard must be reachable from the real loader, not a helper.
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:6a3e45c201ed0de4aa6693249c4ae30cdc3d96faaae29e56d9e3b85144de8711 rationale="Enforcing a config relationship whose violation silently truncates context."
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-28242d, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-28242d)
agent completed: [implementer] developer (claude) (exit=1)
spawn run RUN-260831-28242d failed without autonomous retry; operator action required; provider failure: provider_capability_unavailable: Claude authentication is unavailable; remediation: run `claude login` and retry the goal-bound spawn
2026-09-11 goal audit: WIP exists uncommitted in .temp/STORY-260830-i540ax/worktree (pi_kv_bound.go, pi_kv_bound_test.go, pi_config.go, 2026-08-31 14:57). Deployed --max-kv-size 76800 vs context_window 75000 still unenforced on main. Still valid.
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"less_or_equal ceiling: sonnet-5/high is the top admitted pair for a code change"}
spawn selection rationale for claude-sonnet-5/high: less_or_equal ceiling: sonnet-5/high is the top admitted pair for a code change
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Rank-1 implementation pair; code change with negative-test obligations in a managed Story worktree"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-140-g61d832d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-51a2f1, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-51a2f1)
Implemented validatePiModelHarnessKVBound in pi_kv_bound.go, wired into parsePiProfile production entry. Refuses context_window > referenced model-harness --max-kv-size (or absent bound) for model-harness-backed Pi runtimes; SSH mode and non-model-harness runtimes out of scope. Tests drive parseProjectConfig directly: refusal (absent/below/off-by-one), admission (equal/deployed 75000-76800 golden pair), non-model-harness no-op control. Two mutants applied and confirmed red (delete-guard, off-by-one comparison weakening) then reverted. Documented context_window vs --max-kv-size vs --prompt-cache-bytes in SKILL.md and README.md next to existing operator guidance, plus LOGBOOK entry. No live config files modified. Full tools/agents-infra suite green (build, vet, gofmt, race on changed area, full go test ./...). Ready for review.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-51a2f1, pid=45315, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"less_or_equal ceiling: sonnet-5/high is the top admitted pair for an adversarial review"}
spawn selection rationale for claude-sonnet-5/high: less_or_equal ceiling: sonnet-5/high is the top admitted pair for an adversarial review
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:c1d45d75e280f504c00350c1bdccf0443316cf4183fffb8ceca1bdc4e46e9584 rationale="Rank-1 review pair; adversarial review of the KV-bound/context_window consistency gate at the production entry"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260911-afc935, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260911-afc935)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-afc935, pid=75496, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/low","text":"less_or_equal ceiling: sonnet-5/low for a mechanical checkpoint-only run"}
spawn selection rationale for claude-sonnet-5/low: less_or_equal ceiling: sonnet-5/low for a mechanical checkpoint-only run
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/low pair_source=explicit match=not_recommended snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Consciously below rank-1: a single checkpoint command needs no reasoning; low effort is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-20a682, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-20a682)
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-20a682, pid=960, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-heguyf_spawn-log_-implementer--developer--claude-_RUN-260831-28242d.log](file://TASK-260830-heguyf/TASK-260830-heguyf_spawn-log_-implementer--developer--claude-_RUN-260831-28242d.log) — System spawn log captured by task-board
- [TASK-260830-heguyf_spawn-log_-implementer--developer--claude-_RUN-260911-51a2f1.log](file://TASK-260830-heguyf/TASK-260830-heguyf_spawn-log_-implementer--developer--claude-_RUN-260911-51a2f1.log) — System spawn log captured by task-board
- [TASK-260830-heguyf_results.md](file://TASK-260830-heguyf/TASK-260830-heguyf_results.md) — KV-bound guard implementation summary, AC coverage ratio, mutant evidence table, commands run
- [TASK-260830-heguyf_change-request_rev1.patch](file://TASK-260830-heguyf/TASK-260830-heguyf_change-request_rev1.patch) — Change Request CR-TASK-260830-heguyf-1 revision 1 candidate patch (repository_delta=present, 6 changed paths)
- [TASK-260830-heguyf_change-request_rev1-validation.log](file://TASK-260830-heguyf/TASK-260830-heguyf_change-request_rev1-validation.log) — Change Request CR-TASK-260830-heguyf-1 revision 1 bounded validation log
- [TASK-260830-heguyf_spawn-log_-reviewer--reviewer--claude-_RUN-260911-afc935.log](file://TASK-260830-heguyf/TASK-260830-heguyf_spawn-log_-reviewer--reviewer--claude-_RUN-260911-afc935.log) — System spawn log captured by task-board
- [TASK-260830-heguyf_review-verdict-rev1.md](file://TASK-260830-heguyf/TASK-260830-heguyf_review-verdict-rev1.md) — Reviewer verdict for CR-TASK-260830-heguyf-1 revision 1: accepted with adversarial mutant evidence
- [TASK-260830-heguyf_spawn-log_-implementer--developer--claude-_RUN-260911-20a682.log](file://TASK-260830-heguyf/TASK-260830-heguyf_spawn-log_-implementer--developer--claude-_RUN-260911-20a682.log) — System spawn log captured by task-board

## Created
2026-08-30T13:19:53Z

## Last Update
2026-09-10T20:05:00Z

## Assigned To
[implementer] developer (claude)
