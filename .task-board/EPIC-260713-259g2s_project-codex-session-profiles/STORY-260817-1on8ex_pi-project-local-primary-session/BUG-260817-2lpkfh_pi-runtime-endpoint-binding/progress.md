## Status
to-review

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(5))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Reject wildcard runtime binds and runtime/base_url port divergence through production compose
- [x] Add production-entry negative tests and a valid narrowing control
- [x] Update Pi configuration contract and README/tool documentation where needed
- [x] Run focused and full Go tests, vet, build, setup/install, global/local verify, and diff checks
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [ ] Implementation matches AC
- [ ] Solution fits project architecture
- [ ] Tests green
- [ ] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [ ] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"gpt-5.6-sol/medium","text":"Endpoint binding is a focused security-sensitive Pi parser and production-compose rework with exact wildcard and port-drift negatives; Sol medium is the configured implementation ceiling."}
spawn selection rationale for gpt-5.6-sol/medium: Endpoint binding is a focused security-sensitive Pi parser and production-compose rework with exact wildcard and port-drift negatives; Sol medium is the configured implementation ceiling.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (codex) (run=RUN-260817-c7dc20, max_parallel=20)
spawn run started: [implementer] developer (codex) (run=RUN-260817-c7dc20)
Managed Pi parser now requires exactly one spaced --host 127.0.0.1 and one --port matching base_url before production compose/diagnostics/launch. In-process and installed production negatives reject wildcard and port-drift; a port-equality narrowing mutant fails the named production test. Full Go tests, vet, build, setup.sh, global/source/local-models setup+verify, both Pi print-config plans, downstream profile validation, formatting, diff, and board gates pass. Evidence: BUG-260817-2lpkfh_results.md. Nothing staged or committed.
agent completed: [implementer] developer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-c7dc20, pid=50195, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"claude-opus-5","text":"Opus 5 is the user-requested top independent reviewer for the security-sensitive Pi endpoint-binding contract now that its session window has reset."}
spawn selection rationale for claude-opus-5: Opus 5 is the user-requested top independent reviewer for the security-sensitive Pi endpoint-binding contract now that its session window has reset.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260817-e5ed53, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260817-e5ed53)
Reviewer RUN-260817-e5ed53 verdict: ACCEPTED. Evidence: BUG-260817-2lpkfh_reviewer-verdict.md.

Gate attacked, not read. Production entry 1 (agents-infra compose --mode primary-session --agent pi): exact-loopback control accepted with literal argv preserved; refused wildcard --host 0.0.0.0, --port drift, attached --host=/--port=, missing --host, missing --port, duplicate --host, --host ::, --host localhost (9 refusals, all invalid_project_configuration on runtime.argv). Production entry 2 (agents-infra pi --print-config): control ok, wildcard and port drift refused.

Bypass attempts that found no hole: gate still fires with Pi absent from PATH (config-parse-time, not behind managed-profile resolution); LLAMA_ARG_HOST/LLAMA_ARG_PORT exist in the real llama-b10470 runtime and env is passed through unfiltered, but the runtime was driven directly and warns "will be overwritten by command line argument" - CLI wins, proven not inferred; flag-value swallowing fails closed via preflight+readiness on the declared port; UNIX-socket bind excluded by exact 127.0.0.1 equality.

Narrowing mutants (no delete-only): unbinding the port value reddens runtime_port_drift; unbinding the host value reddens wildcard_runtime_bind; relaxing exactly-one to at-least-one reddens duplicate_runtime_port. Bounds proven.

Reviewer reruns: go test ./... 0, go vet 0, gofmt -l empty, verify global 0, verify local (source) 0, verify local (local-models) 0. Installed ~/.local/bin/agents-infra carries the gate; local-models .local/bin/agents-infra is a wrapper onto the source repo, so the previous cycle F1 source-contract requirement is satisfied.

Non-blocking follow-ups for the story owner (not defects of this bug): (1) mainTestOfficialPiAsset t.Skipf makes both production-entry negatives silently SKIP and report ok when the gitignored .temp/TASK-260817-2h8hn4 Pi asset is absent - proven by removing the path; helper is not in HEAD and backs 6 tests, so it is an inherited story-wide fixture convention; (2) pi_operator_docs_test.go does not pin the new endpoint-binding documentation sentences; (3) --reuse-port is not refused in runtime argv.

Reviewer archetype supplies no commit_ack. Acceptance evidence is handed to the commit-owning mover, which commits its scope and then makes the final done transition with commit_ack=scope_committed.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260817-e5ed53, pid=61427, exit=0)

## Precondition Resources
- [local-models-pi-endpoint-review.md](file://BUG-260817-2lpkfh/local-models-pi-endpoint-review.md) — Downstream reviewer evidence demonstrating wildcard-bind and port-drift production-compose bypasses.

## Outcome Resources
- [BUG-260817-2lpkfh_spawn-log_-implementer--developer--codex-_RUN-260817-c7dc20.log](file://BUG-260817-2lpkfh/BUG-260817-2lpkfh_spawn-log_-implementer--developer--codex-_RUN-260817-c7dc20.log) — System spawn log captured by task-board
- [BUG-260817-2lpkfh_results.md](file://BUG-260817-2lpkfh/BUG-260817-2lpkfh_results.md) — Endpoint-binding fix, production negatives, narrowing mutant, setup/install, and full validation evidence
- [BUG-260817-2lpkfh_spawn-log_-reviewer--reviewer--claude-_RUN-260817-e5ed53.log](file://BUG-260817-2lpkfh/BUG-260817-2lpkfh_spawn-log_-reviewer--reviewer--claude-_RUN-260817-e5ed53.log) — System spawn log captured by task-board
- [BUG-260817-2lpkfh_reviewer-verdict.md](file://BUG-260817-2lpkfh/BUG-260817-2lpkfh_reviewer-verdict.md) — Reviewer verdict: accepted. Gate attacked through two production entries (10 divergence forms), env-precedence bypass empirically refuted, three narrowing mutants proven to bite, full validation rerun.

## Created
2026-08-17T17:15:38Z

## Last Update
2026-08-17T18:33:38Z

## Assigned To
[reviewer] reviewer (claude)
