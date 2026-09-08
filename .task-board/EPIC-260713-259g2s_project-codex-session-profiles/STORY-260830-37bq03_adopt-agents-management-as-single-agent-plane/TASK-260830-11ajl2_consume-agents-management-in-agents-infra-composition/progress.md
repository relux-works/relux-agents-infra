## Status
done

## Review
required

## Task Class
code

## Estimate
estimated(fibonacci(8))

## Blocked By
- (none)

## Blocks
- TASK-260830-3nx7yp
- TASK-260830-8j82uy
- TASK-260830-mf1xwk

## Checklist
- [x] Existing composition, primary-session, targets and model-check tests pass unchanged
- [x] Composition, primary-session launch, targets and model-check behave identically, proven by the existing suite passing unchanged
- [x] No existing test was edited to make the change land; if one had to be, it is reported as a behaviour change instead
- [x] A list of the gaps the first consumer found is attached for the board migration to reuse
- [x] TASK-260830-mf1xwk is not started or modified by this task
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] agents-infra depends on and imports skill-agents-management v0.5.0 and uses it for SystemID normalization and registry cross-check
- [x] The Go floor bump from 1.21.0 to 1.25.5 is stated and its effect on build and CI assumptions checked
- [x] Hardcoded codex, claude and pi branches remain; no branch is removed that v0.5.0 cannot replace
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [ ] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-206aab, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-206aab)
BLOCKED — stop-the-line, no production code changed. Full evidence in TASK-260830-11ajl2_first-consumer-gap-report.md (+ _v050-registry-probe.txt, _baseline-test-suite.log).

Baseline at worktree 9db0526: build exit 0, vet exit 0, go test ./... -count=1 exit 0, 482 test functions green (88/357/13/24).

Why the task cannot land against exact v0.5.0:
1. DIRECTION INVERSION. The shipped pi system plugin and local-models vendor describe agents-infra from the OUTSIDE: pi.ResolveBinary resolves the binary "agents-infra" (probe proved it), pi/args.go emits `pi --profile <p> --`, pi/preflight.go shells out to `agents-infra runtime status --json`, and local-models config Pointer is an agents-infra checkout path + agents.pi.profiles.<name> key. agents-infra consuming these would re-exec itself and shell out to itself. Not a gap a library patch closes.
2. COMPOSITION IS VALIDATE-ONLY. System.ValidateComposition refuses a prefix the caller already built; there is no producer contract. The child_launch_composition.go:186-197 codex/claude branch therefore cannot be removed, only bypassed — which the DoD rejects. Audit verdict for this surface was already "stays here".
3. PLAN IS SINGLE-VARIANT. agentic.Plan has one Mode/Binary/Argv; agents-infra needs interactive + managed-host + managed-client simultaneously. Plan.Nodes is a dependency graph (engine/sidecar), not a variant set. Claude deliberately declares no ManagedSession. This is audit change #3, UNSHIPPED, and no task in STORY-260830-37bq03 owns it (the story shipped #1 via 1jpse1 and #2 via ter72z only).
4. TARGET VOCABULARY HAS NOTHING TO RESOLVE TO. Probe: v0.5.0 has 6 frozen runtimes; lookup of runtime "pi" -> found=false and "local-models" -> found=false. agents-infra pair qwen/pi maps to nothing. Runtime "qwen" is qwen-code x alibaba, which the audit explicitly forbids mapping to qwen-infra.

DoD item 3 ("adding an environment requires no agents-infra change") is false by construction at v0.5.0: a new environment still needs lockXTargetArguments (canonical_target.go:417-427) and buildXPrimarySessionLaunchPlan (primary_session_launch_plan.go:249-256). Driving the admission gate off registry membership would WIDEN it from 3 admitted environments to 7 registered systems — a gate weakening, not a generalization.

Cannot self-serve the standing "fix it in the library and release" instruction: the accepted lockstep plan requires a NEW INDEPENDENT ACCEPTANCE before any release or tag in this sequence, and its Decision pins both consumers to compile exact v0.5.0.

Board blockers (independent): set_status to development was REFUSED — blocked by y6infr (development) and mf1xwk (backlog). y6infr owns the pi plugin consumption + pi x local-models runtime that gap 4 needs; its spawn FAILED (exit=1, codex provider limit, retry Sep 6) so it is parked but not progressing. mf1xwk is listed as a blocker of this task although this task brief says it runs BEFORE the board migration — that edge appears backwards. Per the brief, mf1xwk was not started or modified.

DECISION NEEDED, one of: (A) schedule audit change #3 + producer-side composition + pi x local-models runtime and re-accept the lockstep plan for the extra release; (B) narrow this task to what v0.5.0 honestly supports and strike DoD items 1/3/4 — dependency + identity consumption only, hardcoded branches stay per the audit\047s own "stays here" verdicts; (C) fix the mf1xwk edge direction and let y6infr land first, then rescope to codex/claude only. RECOMMENDATION: B, then A. The DoD was written against the v0.2.0 shape the brief describes, which audit TASK-260830-12n20p already overturned.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-206aab, pid=12387, exit=0)
No Change Request revision was published for TASK-260830-11ajl2 (handoff_unsatisfied): the board is not at to-review
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-ca7291, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-ca7291)
DELIVERED (narrowed scope, option B). Commit 8c7b771 on task-board/story/STORY-260830-37bq03; base not moved. Full evidence in TASK-260830-11ajl2_results.md.

WHAT LANDED. agents-infra depends on skill-agents-management v0.5.0 and consumes it for identity only: NormalizeSystemID, SystemID, Registry, Default — four names, nothing else. New internal/infra/agentic_identity.go (103 lines); project_config.go:384 changes by 2 lines; two new test files; README states the Go floor. Hardcoded codex/claude/pi branches all remain (child_launch_composition.go:186-197, primary_session_launch_plan.go:249-256, canonical_target.go:417-427), which is the correct outcome at v0.5.0, not a shortfall.

THE GATE IS A CONJUNCTION. Production call site: validateProjectTarget (project_config.go:384), the sole admission gate for every agents.targets.* entry. It checks the admitted list FIRST and alone, then normalizes, then requires the id to normalize to itself, then requires a registry hit. Deriving admission from Registry.IDs() would widen from 3 admitted environments to 7 registered systems — the gate weakening the first run named. Vendors are deliberately NOT cross-checked: agents-infra qwen = Pi + local Qwen profile, unrelated to the plane qwen-code x alibaba. No lookup table was written: the three environment names ARE the three system ids, and that equality is asserted rather than assumed.

MUTANTS, all exit 1 (TASK-260830-11ajl2_mutants.log). M-a widen to registry membership -> anti-widening test red on all 4 subtests. M-b NARROWING: cross-check codex only -> 4 tests red, so the covered class is bounded, not just the gate present. M-c drop the pi plugin import -> 5 PRE-EXISTING canonical_target tests red, which is what proves the cross-check is load-bearing in production and not only in its own tests. M-d reach past the identity surface (agentic.BuildPlan) -> guard test red.

EVIDENCE, real exit codes, each command a standalone process, GOWORK=off. build 0, vet 0, gofmt -l no output, go test ./... -count=1 exit 0 (agents-infra 109.196s / internal/infra 183.567s / internal/attachments 1.521s / internal/modelharness 15.578s), root package re-run after the README edit exit 0. LINT: this repo configures no golangci-lint or staticcheck — no config, neither binary installed — so the lint gate here is gofmt + go vet, both clean; nothing else was run and nothing else is claimed.

482 INTACT BY NAME, not by count. Baseline regenerated from HEAD into a throwaway tree via git archive and diffed: 0 removed, 9 added (491 total). No existing test file modified — git status shows only README.md, LOGBOOK.md, go.mod, go.sum, project_config.go changed plus 3 added files.

GO FLOOR 1.21.0 -> 1.25.5, forced by the dependency. No CI exists in this repo to break (no .github, no .gitlab-ci.yml, no .circleci). What it does touch is the RUNTIME: the generated launcher runs go build on every invocation, so a host below the floor refuses under GOTOOLCHAIN=local and downloads a toolchain (network) under the default auto — both modes reproduced, not asserted. launcherStartupFailure fails closed and carries the toolchain error verbatim. scripts/setup.sh install_go() checks presence but never version; left as is, installer policy is out of scope. v0.5.0 is public on proxy.golang.org with sum.golang.org hashes, so no credential setup. Binary +79,632 bytes (+0.70%); cobra is not linked.

TWO THINGS FOR THE REVIEWER. (1) The checklist in progress.md was stale relative to the ratified scope — the description and AC were rewritten for option B but the four struck DoD items were still there. They are removed and recorded VERBATIM with reasons in TASK-260830-11ajl2_checklist-realignment.md, along with the three AC-derived items that replaced them; any removal can be restored exactly. (2) TASK-260830-y6infr still blocks this task on the board. That edge was correct for the original scope — y6infr owns the pi plugin consumption and the pi x local-models runtime the old DoD needed — and the narrowed scope needs neither. y6infr is itself parked (codex provider limit, retry Sep 6). Dropping the edge is an orchestrator call and this run did not make it.

SCOPE KEPT. No new agents-management release, no tag, exact v0.5.0 only. No hardcoded branch removed. TASK-260830-mf1xwk untouched. No credential, token, cookie or keychain value printed, exported or persisted. Commits are unsigned because no signing key is configured in this environment (git config has no gpg.format/user.signingkey and HEAD~1 is also unsigned); author identity is the configured alexis <alexis@relux.works>.
HANDOFF ATTEMPTED AND REFUSED BY THE BOARD, NOT BY MISSING EVIDENCE. `task-board handoff TASK-260830-11ajl2 --role developer` reported only: cannot set to to-review — blocked by TASK-260830-y6infr (status: development). Checklist and outcome-resource verification passed; the sole obstacle is the dependency edge.

EXACT DECISION NEEDED (orchestrator, one call): drop the TASK-260830-y6infr -> TASK-260830-11ajl2 blocking edge, or move this task to to-review directly. That edge was correct for the ORIGINAL scope, which needed the pi plugin consumption and the pi x local-models runtime y6infr owns. The scope narrowed to identity-only on 2026-08-31 and needs neither. y6infr is itself parked and not progressing (spawn exit=1, codex provider limit, retry Sep 6), so waiting for it parks completed, green, evidenced work behind an unrelated task.

This run did NOT remove the edge. The previous run reported both the y6infr and the mf1xwk edges; only mf1xwk was fixed in the rewrite, so the remaining edge is treated as the orchestrators to keep or drop, not as an obstacle to route around. Removing a blocking edge to make my own work land is exactly the gate weakening this task spent its evidence budget arguing against.

Code state: committed as 8c7b771 on task-board/story/STORY-260830-37bq03. Nothing is pending or unsaved. Ready for review the moment the edge is resolved.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-ca7291, pid=17178, exit=0)
No Change Request revision was published for TASK-260830-11ajl2 (handoff_unsatisfied): the board is not at to-review
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-781ec1, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-781ec1)
Review ACCEPTED (RUN-260831-781ec1, head 8c7b771). Evidence: TASK-260830-11ajl2_review-verdict.md + TASK-260830-11ajl2_review-suite.log. Gate attacked with 6 mutants in a throwaway copy (widening to registry membership, delete-only, narrowing to codex-only, drop-pi-import, behaviour-via-method, second-file-import) - all but one caught, and the escape is confined and non-dangerous (F1). No end-to-end path to a fourth environment: env gate and the vendor/environment pair switch are independent closed sets. Identity only confirmed by diff: only agentic_identity.go plus 2 lines of project_config.go. Go floor: both GOTOOLCHAIN branches reproduced, one go.mod in repo, no CI, no version pin anywhere. No test edited: 0 removed / 9 added by name across the whole branch. Suite green, gofmt/vet clean, worktree pristine. Checklist decision: all four removals correct, nothing to restore; removal 2 verified against canonical_target.go:415-425, project_config.go:398-421 and primary_session_launch_plan.go:249-256, and its generalisation is exactly the widening mutant. 4 non-blocking findings recorded. Reviewer supplies no commit_ack; commit-owning mover integrates and makes the done transition. Orchestrator items: TASK-260830-y6infr blocking edge is now stale for the narrowed scope.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-781ec1, pid=24223, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260830-11ajl2_spawn-log_-implementer--developer--claude-_RUN-260831-206aab.log](file://TASK-260830-11ajl2/TASK-260830-11ajl2_spawn-log_-implementer--developer--claude-_RUN-260831-206aab.log) — System spawn log captured by task-board
- [TASK-260830-11ajl2_first-consumer-gap-report.md](file://TASK-260830-11ajl2/TASK-260830-11ajl2_first-consumer-gap-report.md) — First-consumer gap report: why agents-infra cannot consume agents-management v0.5.0 for composition/primary-session/targets/model-check; 8 reusable gaps, verified evidence, 3 options
- [TASK-260830-11ajl2_v050-registry-probe.txt](file://TASK-260830-11ajl2/TASK-260830-11ajl2_v050-registry-probe.txt) — Executable evidence: probe compiled against agents-management v0.5.0 showing 7 registered systems vs 3 admitted environments, no pi/local-models runtime, and pi ResolveBinary targeting agents-infra itself
- [TASK-260830-11ajl2_baseline-test-suite.log](file://TASK-260830-11ajl2/TASK-260830-11ajl2_baseline-test-suite.log) — Baseline: GOWORK=off go test ./... -count=1 at worktree 9db0526, exit 0, 482 test functions green
- [TASK-260830-11ajl2_spawn-log_-implementer--developer--claude-_RUN-260831-ca7291.log](file://TASK-260830-11ajl2/TASK-260830-11ajl2_spawn-log_-implementer--developer--claude-_RUN-260831-ca7291.log) — System spawn log captured by task-board
- [TASK-260830-11ajl2_results.md](file://TASK-260830-11ajl2/TASK-260830-11ajl2_results.md) — Identity-only consumption of agents-management v0.5.0: what landed, the conjunctive admission gate and its production call site, 4 mutants, 482 existing tests intact by name, Go floor 1.21.0 to 1.25.5 blast radius
- [TASK-260830-11ajl2_consumer-gaps.md](file://TASK-260830-11ajl2/TASK-260830-11ajl2_consumer-gaps.md) — Reusable v0.5.0 gap and trap list for the board migration and TASK-260831-1pfnxx: 5 contract gaps, 5 traps, 2 mechanical notes
- [TASK-260830-11ajl2_checklist-realignment.md](file://TASK-260830-11ajl2/TASK-260830-11ajl2_checklist-realignment.md) — Verbatim record of the four stale DoD checklist items removed and the three added from the rewritten AC, so any removal can be restored exactly
- [TASK-260830-11ajl2_mutants.log](file://TASK-260830-11ajl2/TASK-260830-11ajl2_mutants.log) — Raw output of the four gate mutants (widen-to-registry, narrow-to-codex-only, drop-pi-import, reach-past-identity-surface), all exit 1
- [TASK-260830-11ajl2_full-suite-after.log](file://TASK-260830-11ajl2/TASK-260830-11ajl2_full-suite-after.log) — GOWORK=off go test ./... -count=1 after the change, exit 0, all four packages green
- [TASK-260830-11ajl2_change.patch](file://TASK-260830-11ajl2/TASK-260830-11ajl2_change.patch) — Complete reviewable diff: go.mod/go.sum, new agentic_identity.go, 2-line project_config.go wiring, two new test files, README Go-floor note
- [TASK-260830-11ajl2_spawn-log_-reviewer--reviewer--claude-_RUN-260831-781ec1.log](file://TASK-260830-11ajl2/TASK-260830-11ajl2_spawn-log_-reviewer--reviewer--claude-_RUN-260831-781ec1.log) — System spawn log captured by task-board
- [TASK-260830-11ajl2_review-verdict.md](file://TASK-260830-11ajl2/TASK-260830-11ajl2_review-verdict.md) — Review verdict: ACCEPTED. Gate attacked with 6 mutants incl. widening/delete/narrowing/plugin-drop, both Go-floor branches reproduced, no-test-edited proven by name-diff, checklist removals decided (all four correct), 4 non-blocking findings
- [TASK-260830-11ajl2_review-suite.log](file://TASK-260830-11ajl2/TASK-260830-11ajl2_review-suite.log) — Independent reviewer re-run: GOWORK=off go test ./... -count=1 at 8c7b771, exit 0, four packages green

## Created
2026-08-29T22:28:23Z

## Last Update
2026-08-31T03:34:01Z

## Assigned To
[reviewer] reviewer (claude)
