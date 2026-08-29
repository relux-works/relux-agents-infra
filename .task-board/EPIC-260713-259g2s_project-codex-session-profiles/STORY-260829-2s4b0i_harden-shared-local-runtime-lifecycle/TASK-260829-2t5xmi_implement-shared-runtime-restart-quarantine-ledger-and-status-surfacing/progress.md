## Status
development

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [ ] Persist restart and quarantine ledger across broker restart using the existing shared-runtime path contract
- [ ] Expose restart_count quarantined_until and last_readiness_match in SharedRuntimeStatus JSON
- [ ] Implement deterministic operator-configured bounded exponential backoff stable-run reset automatic half-open and manual quarantine
- [ ] Prove real handleConnection and runSharedPiSession release leases after abrupt client death
- [ ] Use static subprocess fixtures only and introduce no numeric code defaults or live-model interaction
- [ ] Code written per task description and AC
- [ ] Relevant tests written for new or changed behavior and passing
- [ ] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [ ] Lint clean
- [ ] Relevant build/validation commands run after changes and build not broken
- [ ] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [ ] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/high","text":"Cross-process persisted lifecycle state and abrupt-death lease cleanup are high-risk concurrency work; Sol high is the configured maximum and must satisfy deterministic fake-only recovery gates"}
spawn selection rationale for gpt-5.6-sol/high: Cross-process persisted lifecycle state and abrupt-death lease cleanup are high-risk concurrency work; Sol high is the configured maximum and must satisfy deterministic fake-only recovery gates
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: exclusive[codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260829-e40c5b, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260829-e40c5b)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260829-2t5xmi_spawn-log_-implementer--developer--codex-_RUN-260829-e40c5b.log](file://TASK-260829-2t5xmi/TASK-260829-2t5xmi_spawn-log_-implementer--developer--codex-_RUN-260829-e40c5b.log) — System spawn log captured by task-board

## Created
2026-08-29T10:55:36Z

## Last Update
2026-08-29T12:32:51Z

## Assigned To
[implementer] developer (codex)
