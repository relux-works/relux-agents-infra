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
- [x] One invocation reports every missing required field for the profile, not the first one and then stop
- [x] The error names the field path and what a valid value looks like, as the board-side error contract requires
- [x] Fail-closed validation is preserved; nothing here adds an automatic migration or a silent default
- [x] Regression test drives the real compose path with a config missing several required fields
- [x] One diagnostic invocation reports every missing required profile field (names + config path) in a single run, not only the first
- [x] Negative test: a profile missing all eight late-added fields (max_segment_bytes, max_segments, restart_limit, restart_initial_backoff_seconds, restart_max_backoff_seconds, stable_run_seconds, quarantine_seconds, resource_pressure_mode) yields all eight names; positive control: a valid profile reports none
- [x] Existing single-field refusals and their wording unchanged (golden); full tools/agents-infra suite green
- [x] Task-board half (prominent degraded flag in spawn output) NOT implemented here; it is skill-project-management BUG-260911-2tvq62
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] In a managed Story worktree the candidate is left UNCOMMITTED in the worktree for the handoff to snapshot — never commit on the Story branch. A producer commit moves the branch tip off the recorded checkpoint and the handoff refuses with change_request_candidate_committed_past_checkpoint; repair with `git reset --soft <checkpoint_oid>` before completing again.
- [x] Every command, message, state, or refusal named in the AC is driven through the production entry point by a named committed test, or is declared a stated bound. Report coverage as a ratio — `n of m AC rows driven` — and name the production call site for each. Prose in place of the ratio is not evidence.
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Every gate ships at least one NARROWING mutant — the gate stays present and is weakened to admit exactly one member of the class it must reject, and a named test must fail. A delete-only mutant proves only that the gate exists and is not accepted as evidence.
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Add projectPiProfileFieldGaps lenient presence scanner (new file project_config_field_gaps.go)
- [x] Wire Report.PiProfileFieldGaps into Doctor() and print in doctor CLI output
- [x] Negative test: 8 missing runtime.sharing fields all named in one Doctor() call
- [x] Positive control: valid shared profile reports zero gaps
- [x] Narrowing mutant: drop resource_pressure_mode check -> negative test fails
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches
- [x] A gate that inspects source text is additionally attacked by a mutant that PRESERVES the searched-for token and changes behavior, and the mutant harness executes the behavioral suite, not only the static checker.

## Notes
2026-09-11 goal audit: split. Prominence of degraded composition in spawn output is task-board work, mirrored as skill-project-management BUG-260911-2tvq62; this record keeps only the all-missing-fields diagnostic.
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/high","text":"less_or_equal ceiling: sonnet-5/high is the top admitted pair for a code change"}
spawn selection rationale for claude-sonnet-5/high: less_or_equal ceiling: sonnet-5/high is the top admitted pair for a code change
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Rank-1 implementation pair; code change with negative-test obligations in the managed Story worktree"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-140-g61d832d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-b97cf6, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-b97cf6)
Added an additive lenient presence scan (project_config_field_gaps.go: projectPiProfileFieldGaps/missingPiProfileFields) wired into Doctor() as Report.PiProfileFieldGaps, printed by agents-infra doctor local. It reads each ancestor project config with toml.Unmarshal directly (not the strict parser) and reports every absent required Pi profile field with dotted path + config path + expected-value description, instead of stopping at the first one. Existing strict parsers (parsePiRuntimeSharing etc.) are unchanged byte-for-byte -- fail-closed single-field refusal on the production launch path is preserved; this is a read-only diagnostic layered next to it, not a replacement. Negative test removes all 8 late-added runtime.sharing fields from a valid profile and asserts Doctor() names all 8 in one call; positive control asserts a valid profile reports zero gaps. Narrowing mutant (drop one required-field check) kills the negative test, confirmed then reverted. Full go build/vet/gofmt/test ./... green. Task-board half (BUG-260911-2tvq62) not touched.
Removed 2 DoD checklist items as inapplicable to this leaf's explicit scope: (1) 'Degradation is visible without reading a spawn note line by line...' -- that is the task-board-side prominence work, explicitly split out to skill-project-management BUG-260911-2tvq62 per this task's Scope section (mirrored by the remaining item 'Task-board half...NOT implemented here'). (2) 'A gate that inspects source text is additionally attacked by a mutant that PRESERVES the searched-for token...' -- this pattern applies to gates that grep/search raw text for a token; missingPiProfileFields does structured map-key presence checks on already-parsed TOML data, not source-text token search, so no such mutant shape exists for this gate. All other DoD items checked with evidence in the attached BUG-260830-5pmaiz_results.md outcome resource and LOGBOOK.md 2026-09-11 1730 entry.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-b97cf6, pid=90395, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/high","text":"less_or_equal ceiling: sonnet-5/high is the top admitted pair for an adversarial review"}
spawn selection rationale for claude-sonnet-5/high: less_or_equal ceiling: sonnet-5/high is the top admitted pair for an adversarial review
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/high pair_source=explicit match=recommended_rank_1 snapshot=sha256:c1d45d75e280f504c00350c1bdccf0443316cf4183fffb8ceca1bdc4e46e9584 rationale="Rank-1 review pair; adversarial review of the all-missing-fields diagnostic with its negative test"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-140-g61d832d; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260911-11bcfa, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260911-11bcfa)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-11bcfa, pid=37343, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/low","text":"less_or_equal ceiling: sonnet-5/low for a mechanical integrate-only run"}
spawn selection rationale for claude-sonnet-5/low: less_or_equal ceiling: sonnet-5/low for a mechanical integrate-only run
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/low pair_source=explicit match=not_recommended snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Consciously below rank-1: a single integrate command needs no reasoning; low effort is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-c1bed0, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-c1bed0)
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-c1bed0, pid=49104, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/medium","text":"less_or_equal ceiling: sonnet-5/medium for a mechanical refresh/republish"}
STORY-260830-112ewt base refresh SKIPPED: the managed workspace holds uncommitted work, so there was no clean checkpoint branch to replay onto trunk b138ebce0086; the branch is unchanged at fork point 4126777d9813
spawn selection rationale for claude-sonnet-5/medium: less_or_equal ceiling: sonnet-5/medium for a mechanical refresh/republish
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/medium pair_source=explicit match=not_recommended snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Consciously below rank-1: base refresh + republish of an accepted tree with one logbook merge; medium is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-921f49, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-921f49)
Checklist item 29 (token-preserving mutant against a source-text-inspecting gate) is not applicable here: projectPiProfileFieldGaps (project_config_field_gaps.go) scans a parsed TOML config table (map[string]any) for missing keys — it never inspects source code text, greps a file for a token, or runs a static checker over source. The only gate in this task's scope is the field-presence scanner, already covered by the narrowing mutant on resource_pressure_mode (item validated in rev1 review). Checking as a stated bound, not fabricated evidence.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-921f49, pid=54153, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/medium","text":"less_or_equal ceiling: sonnet-5/medium for a republish identity review"}
spawn selection rationale for claude-sonnet-5/medium: less_or_equal ceiling: sonnet-5/medium for a republish identity review
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/medium pair_source=explicit match=not_recommended snapshot=sha256:c1d45d75e280f504c00350c1bdccf0443316cf4183fffb8ceca1bdc4e46e9584 rationale="Consciously below rank-1: identity check of a rebased republish against an existing verdict; medium is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260911-fc50ef, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260911-fc50ef)
Rev2 REJECTED (changes_requested, repeat-of: none): product delta outside LOGBOOK.md verified byte-identical to accepted rev1, base verified = current trunk b138ebc, bounded validation green. But LOGBOOK.md merge silently dropped trunk entry 1345 (Environment Admission Has a Coincidental Second Gate, TASK-260911-39bag5) which exists in b138ebc and in this branch commit 1688f0c but is absent from the candidate tree, no conflict markers. See BUG-260830-5pmaiz_review-verdict-rev2.md. Next revision must recombine LOGBOOK.md to keep all three entries (trunk 1345 + candidate 1730 and 1631) without touching any other file.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-fc50ef, pid=97732, exit=0)
spawn selection rationale for claude-sonnet-5/low: less_or_equal ceiling: sonnet-5/low for a mechanical integrate-only run
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/low pair_source=explicit match=not_recommended snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Consciously below rank-1: a single integrate command needs no reasoning; low effort is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-005b62, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-005b62)
spawn run RUN-260911-005b62 cancelled by operator; operator action required; reason: no operator reason supplied
agent completed: [implementer] developer (claude) (exit=143)
spawn run completed: claude (run=RUN-260911-005b62, pid=2852, exit=143)
spawn selection rationale tuple: {"role":"developer","pair":"claude-sonnet-5/medium","text":"less_or_equal ceiling: sonnet-5/medium for a docs-only rework"}
spawn selection rationale for claude-sonnet-5/medium: less_or_equal ceiling: sonnet-5/medium for a docs-only rework
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/medium pair_source=explicit match=not_recommended snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Consciously below rank-1: single-file logbook recombination fix with a mechanical proof; medium is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-b8093a, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-b8093a)
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-b8093a, pid=4334, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-sonnet-5/medium","text":"less_or_equal ceiling: sonnet-5/medium for a docs-only rework review"}
spawn selection rationale for claude-sonnet-5/medium: less_or_equal ceiling: sonnet-5/medium for a docs-only rework review
spawn workload selection: class=review source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/medium pair_source=explicit match=not_recommended snapshot=sha256:c1d45d75e280f504c00350c1bdccf0443316cf4183fffb8ceca1bdc4e46e9584 rationale="Consciously below rank-1: mechanical verification of a logbook-only fix against an existing verdict; medium is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260911-74799c, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260911-74799c)
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-74799c, pid=13767, exit=0)
spawn selection rationale for claude-sonnet-5/low: less_or_equal ceiling: sonnet-5/low for a mechanical integrate-only run
spawn workload selection: class=implementation source=derived policy=spawn.workload_classes pair=claude/claude-sonnet-5/low pair_source=explicit match=not_recommended snapshot=sha256:321bcdb6b25052021ee23e989517b46b1e84e04bf55d389460146a0c21374783 rationale="Consciously below rank-1: a single integrate command needs no reasoning; low effort is in policy"
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-162-gd5e12bd; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260911-50f4c7, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260911-50f4c7)
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260911-50f4c7, pid=15854, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-b97cf6.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-b97cf6.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_results.md](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_results.md) — Rev 3 rework results: LOGBOOK.md 1345 entry restoration proof + bounded validation
- [BUG-260830-5pmaiz_change-request_rev1.patch](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_change-request_rev1.patch) — Change Request CR-BUG-260830-5pmaiz-1 revision 1 candidate patch (repository_delta=present, 5 changed paths)
- [BUG-260830-5pmaiz_change-request_rev1-validation.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_change-request_rev1-validation.log) — Change Request CR-BUG-260830-5pmaiz-1 revision 1 bounded validation log
- [BUG-260830-5pmaiz_spawn-log_-reviewer--reviewer--claude-_RUN-260911-11bcfa.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-reviewer--reviewer--claude-_RUN-260911-11bcfa.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_review-verdict-rev1.md](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_review-verdict-rev1.md) — Reviewer verdict evidence for CR-BUG-260830-5pmaiz-1 rev1
- [BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-c1bed0.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-c1bed0.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-921f49.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-921f49.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_change-request_rev2.patch](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_change-request_rev2.patch) — Change Request CR-BUG-260830-5pmaiz-2 revision 2 candidate patch (repository_delta=present, 5 changed paths)
- [BUG-260830-5pmaiz_change-request_rev2-validation.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_change-request_rev2-validation.log) — Change Request CR-BUG-260830-5pmaiz-2 revision 2 bounded validation log
- [BUG-260830-5pmaiz_spawn-log_-reviewer--reviewer--claude-_RUN-260911-fc50ef.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-reviewer--reviewer--claude-_RUN-260911-fc50ef.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_review-verdict-rev2.md](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_review-verdict-rev2.md) — Reviewer verdict for CR-BUG-260830-5pmaiz-2 rev2: changes_requested, LOGBOOK merge dropped a trunk entry
- [BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-005b62.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-005b62.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-b8093a.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-b8093a.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_change-request_rev3.patch](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_change-request_rev3.patch) — Change Request CR-BUG-260830-5pmaiz-3 revision 3 candidate patch (repository_delta=present, 5 changed paths)
- [BUG-260830-5pmaiz_change-request_rev3-validation.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_change-request_rev3-validation.log) — Change Request CR-BUG-260830-5pmaiz-3 revision 3 bounded validation log
- [BUG-260830-5pmaiz_spawn-log_-reviewer--reviewer--claude-_RUN-260911-74799c.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-reviewer--reviewer--claude-_RUN-260911-74799c.log) — System spawn log captured by task-board
- [BUG-260830-5pmaiz_review-verdict-rev3.md](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_review-verdict-rev3.md) — Reviewer verdict for CR-BUG-260830-5pmaiz-3 rev3: accepted
- [BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-50f4c7.log](file://BUG-260830-5pmaiz/BUG-260830-5pmaiz_spawn-log_-implementer--developer--claude-_RUN-260911-50f4c7.log) — System spawn log captured by task-board

## Created
2026-08-30T00:00:44Z

## Last Update
2026-09-11T14:57:13Z

## Assigned To
[implementer] developer (claude)
