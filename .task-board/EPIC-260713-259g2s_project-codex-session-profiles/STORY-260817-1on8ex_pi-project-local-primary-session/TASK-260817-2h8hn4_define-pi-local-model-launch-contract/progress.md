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
- TASK-260817-ccpnlm

## Checklist
- [x] Verify Pi behavior against the linked official usage, settings, and custom-model documentation.
- [x] Specify exact project TOML, precedence, non-launching diagnostics, and explicit CLI override behavior.
- [x] Define process ownership and lifecycle for Qwen 3.8 27B and Muse Glimmer 30B plus DFlash profiles without mutating global Pi state.
- [x] Document rejected alternatives, fail-closed cases, and executable positive and negative acceptance scenarios.
- [x] Board size is proportional to the spec and is the smallest decomposition that maps every requirement
- [x] Every story and task traces to a concrete spec requirement; justified-gap elements also carry a self-verified gap record
- [x] Beyond-literal-spec elements include a written justification naming the gap and the spec and out-of-scope checks performed before creation
- [x] Research tasks cite an exact question the spec genuinely leaves open
- [x] Dependencies linked
- [x] Tasks are atomic — one clear deliverable each
- [x] Completeness verified — nothing forgotten
- [x] Any planning artifacts actually produced are linked as new task-scoped outcome resources; diagrams are strictly optional, never a standing deliverable
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches
- [x] Managed-profile provider/model overrides are either exact-identity only or backed by a deterministic generated catalog, with mismatch and separator-lookalike negative cases.
- [x] DFlash launch uses an exact authoritative attestation schema and refuses absent, malformed, false, stale, or mismatched evidence; unknown is diagnostics-only.
- [x] Define a production-faithful argv bridge that never forwards a fake Pi separator, rejects or safely encodes post-separator option-looking operands, and explicitly handles --flag=value forms before runtime start.

## Notes
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Focused contract design with three official Pi references and bounded acceptance scenarios; Sol medium is sufficient before separate implementation and review gates."}
spawn selection rationale for gpt-5.6-sol/medium: Focused contract design with three official Pi references and bounded acceptance scenarios; Sol medium is sufficient before separate implementation and review gates.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-08bda1, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-08bda1)
Decision artifact: .research/260817_pi-local-model-launch-contract.md and outcome TASK-260817-2h8hn4_pi-local-model-launch-contract.md. Verified official Pi usage/settings/models at redirected main revision a1bc0ec79010887210cc7de28714d72c78577dab. Chosen boundary: selected profiles use agents-infra-owned isolated Pi state, atomic local-only models.json, loopback managed runtime child, exclusive profile lock, exact model readiness, and fail-closed cleanup; no ~/.pi/agent mutation. Public DFlash inventory does not substantiate the task product labels as checkpoint IDs, so exact runtime target/draft strings remain operator-supplied and are never guessed. Existing three-task chain is the smallest complete decomposition; no new element or diagram justified. Validation: task-board validate and git diff --check pass; logs in .temp/TASK-260817-2h8hn4/.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-08bda1, pid=83947, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Review is bounded to one architecture artifact with explicit positive and negative scenarios; Sol medium is sufficient to challenge the contract before implementation."}
spawn selection rationale for gpt-5.6-sol/medium: Review is bounded to one architecture artifact with explicit positive and negative scenarios; Sol medium is sufficient to challenge the contract before implementation.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-252f62, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-252f62)
Reviewer cycle 1: changes requested -> analysis. F1 generated models.json contains one identity while managed CLI provider/model overrides may select another, creating a catalog/profile bypass. F2 DFlash has no exact authoritative attestation source and scenario 11 permits unknown, so absent evidence may be treated as satisfied. F3 the produced story plan is not linked as an outcome despite the checklist assertion. See TASK-260817-2h8hn4_reviewer-verdict-cycle-1.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-252f62, pid=88363, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Targeted rework is limited to the reviewer findings on override identity, DFlash attestation, and linked plan evidence; Sol medium remains sufficient before a fresh review."}
spawn selection rationale for gpt-5.6-sol/medium: Targeted rework is limited to the reviewer findings on override identity, DFlash attestation, and linked plan evidence; Sol medium remains sufficient before a fresh review.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-f98913, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-f98913)
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-f98913, pid=90677, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Second-cycle review targets the corrected exact-identity override rule, authoritative DFlash attestation, linked plan evidence, and regression scenarios; Sol medium is sufficient for this bounded verdict."}
spawn selection rationale for gpt-5.6-sol/medium: Second-cycle review targets the corrected exact-identity override rule, authoritative DFlash attestation, linked plan evidence, and regression scenarios; Sol medium is sufficient for this bounded verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-3efee7, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-3efee7)
Reviewer cycle 2: changes requested -> analysis. Pi production parseArgs at pinned revision a1bc0ec79010887210cc7de28714d72c78577dab does not implement -- as end-of-options; the promised literal separator is parsed as an empty unknown flag and later provider/model/api-key/thinking/trust flags remain active, bypassing the managed identity boundary. Native equals forms are also not recognized model/provider options. See TASK-260817-2h8hn4_reviewer-verdict-cycle-2.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-3efee7, pid=94795, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Third-cycle rework is narrowly bounded to Pi production argv semantics and a fail-closed operand bridge, with the prior identity and DFlash contract retained; Sol medium is sufficient before another independent review."}
spawn selection rationale for gpt-5.6-sol/medium: Third-cycle rework is narrowly bounded to Pi production argv semantics and a fail-closed operand bridge, with the prior identity and DFlash contract retained; Sol medium is sufficient before another independent review.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-ef6255, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-ef6255)
Rework cycle 3: pinned Pi parseArgs evidence proves literal -- is not an end-of-options separator. Revised the decision artifact to strip the wrapper delimiter, reject post-delimiter -/@ operands and boundary ambiguity before side effects, normalize recognized equals forms to spaced Pi argv, and require real-entry negative plus narrowing-mutant evidence. Updated outcome resource; source/resource SHA-256 parity b1cb5ed759b7a3667b2ab190dbe0a7558fd70f9dfc61ef0e0dbfce8fadbc7bcf. Board validate, git diff check, and Markdown whitespace check pass; no product code/build applies to this no-implementation research task.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-ef6255, pid=96675, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Third-cycle review is bounded to the production-faithful Pi argv bridge plus retained identity, attestation, and evidence gates; Sol medium is sufficient for a final independent architecture verdict."}
spawn selection rationale for gpt-5.6-sol/medium: Third-cycle review is bounded to the production-faithful Pi argv bridge plus retained identity, attestation, and evidence gates; Sol medium is sufficient for a final independent architecture verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-2de4a8, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-2de4a8)
Reviewer cycle 3: changes requested -> analysis. The argv bridge is proven only against Pi commit a1bc0ec79010887210cc7de28714d72c78577dab, but managed launch accepts an arbitrary pi executable without an authoritative parser identity or mismatch/unknown refusal. See TASK-260817-2h8hn4_reviewer-verdict-cycle-3.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-2de4a8, pid=98991, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Cycle-four rework is narrowly scoped to binding the already-designed Pi argv bridge to an authoritative executable identity before any managed side effect; Sol medium is sufficient to amend the contract and production-negative evidence without widening scope."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-four rework is narrowly scoped to binding the already-designed Pi argv bridge to an authoritative executable identity before any managed side effect; Sol medium is sufficient to amend the contract and production-negative evidence without widening scope.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-d5fc0c, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-d5fc0c)
Rework cycle 4: bound the managed Pi argv bridge to a deterministic compile-time compatibility catalog. Project TOML selects but cannot mint an entry. Direct launch statically verifies the canonical published npm package tree, entrypoint, and parser before lock/file/runtime/Pi side effects; absent, malformed, unsupported, mismatch, and unknown identities fail distinctly. Added same-version spoof, malformed/unreadable, wrapper, modified-tree, project-self-mint, and version-only narrowing cases. Current official docs rechecked at df018b6020181d4245575fba006361ab69a1408b; immutable npm 0.84.2 identity recorded. No implementation or new board element.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-d5fc0c, pid=4322, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Cycle-four review is tightly bounded to authoritative Pi package identity, pre-side-effect enforcement, spoof and same-version mutant rejection, while confirming the retained managed identity and DFlash gates; Sol medium is sufficient for the final independent verdict."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-four review is tightly bounded to authoritative Pi package identity, pre-side-effect enforcement, spoof and same-version mutant rejection, while confirming the retained managed identity and DFlash gates; Sol medium is sufficient for the final independent verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-1852d9, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-1852d9)
Reviewer cycle 4: changes requested -> analysis. The catalog verifies the npm package tree, but the selected dist/cli.js uses #!/usr/bin/env node and the contract does not bind the Node interpreter or sanitize loader-affecting environment. A production-faithful NODE_OPTIONS=--require preload rewrote managed provider/model argv without changing any verified package byte. This is a composed-artifact bypass path; add exact execution-closure identity, environment binding, verification/use protection, and real-entry negative/narrowing cases. See TASK-260817-2h8hn4_reviewer-verdict-cycle-4.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-1852d9, pid=15323, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Cycle-five rework is bounded to the managed execution closure: absolute verified Node host, loader-environment sanitization, dependency verification, point-of-use recheck, and an explicit same-user concurrency threat boundary; Sol medium is sufficient to amend the accepted contract and tests."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-five rework is bounded to the managed execution closure: absolute verified Node host, loader-environment sanitization, dependency verification, point-of-use recheck, and an explicit same-user concurrency threat boundary; Sol medium is sufficient to amend the accepted contract and tests.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-79bc76, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-79bc76)
Rework cycle 5: replaced the package-only managed Pi gate with an exact official standalone execution-closure contract. Initial production catalog selects Pi v0.84.2 darwin-arm64 release asset c996e888...; full 217-file release-tree manifest and native entrypoint are verified initially and immediately before Pi spawn. Managed launch rejects loader-affecting environment names, executes only the captured canonical standalone absolute path, refuses npm/shebang Pi, and records an explicit same-UID/kernel threat boundary. Added real-entry preload, hostile Node/PATH, dependency-substitution, tree-swap, environment-change, and narrowing-mutant scenarios. Updated downstream implementation/operator AC and outcome resource; no new task or diagram is justified.
Cycle 5 validation: official v0.84.2 darwin-arm64 release checksum, extracted 217-file canonical tree digest, and standalone binary digest all match the contract; source/outcome SHA-256 parity is 10762c3c307cff76c9ee53c2608ed9178831b5a51fd9939f56c95060123cb1e3. task-board validate, git diff --check, and Markdown whitespace checks pass. Product code/build tests are not applicable to this no-implementation architecture task.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-79bc76, pid=16760, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Cycle-five review is bounded to the official standalone Pi execution closure, loader-environment rejection, point-of-use tree recheck, and the explicit same-UID race boundary while retaining prior gates; Sol medium is sufficient for the final independent acceptance verdict."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-five review is bounded to the official standalone Pi execution closure, loader-environment rejection, point-of-use tree recheck, and the explicit same-UID race boundary while retaining prior gates; Sol medium is sufficient for the final independent acceptance verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-400da7, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-400da7)
Reviewer cycle 5: changes requested -> analysis. Qwen readiness accepts liveness plus /v1/models but has no nonce, PID, socket ownership, or equivalent proof that the selected runtime child owns the listener. A live non-binding child plus a foreign loopback listener satisfies every written Qwen gate and can reach Pi launch, contradicting the no-attach ownership contract. Negative shape: bypass path around the check. See TASK-260817-2h8hn4_reviewer-verdict-cycle-5.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-400da7, pid=22384, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Cycle-six rework is narrowly scoped to a universal child-bound runtime attestation protocol for Qwen and Muse, including foreign-listener, replay, nonce, PID, model, and lifecycle negatives; Sol medium is sufficient to unify the existing readiness contract."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-six rework is narrowly scoped to a universal child-bound runtime attestation protocol for Qwen and Muse, including foreign-listener, replay, nonce, PID, model, and lifecycle negatives; Sol medium is sufficient to unify the existing readiness contract.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-10b3f2, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-10b3f2)
Cycle 6 rework closes reviewer F1. The decision now requires one exact agents-infra.runtime-attestation.v1 contract for every managed profile: fresh 256-bit nonce, direct-child PID, exact model, byte-sorted exact capabilities, timestamp, and Qwen dflash=null; Muse carries the exact active DFlash target/draft object. /v1/models is discovery only. Absence, unreadable observation, invalid/replayed/mismatched evidence, and foreign-listener plus live non-binding-child paths refuse before Pi. Updated downstream implementation and operator AC/checklists. Official Pi models/settings/usage docs rechecked on 2026-08-17. Validation: .temp/TASK-260817-2h8hn4/validation-06.log; board valid; git diff --check clean; research/outcome SHA-256 f170355332267a620029748eb9b77db1499f9242d9aa51636d491ea14163b9b4.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-10b3f2, pid=24973, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Cycle-six review is tightly bounded to the universal nonce/direct-child-PID runtime ownership attestation, Qwen foreign-listener refusal, and retained standalone Pi execution closure; Sol medium is sufficient for the final independent contract verdict."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-six review is tightly bounded to the universal nonce/direct-child-PID runtime ownership attestation, Qwen foreign-listener refusal, and retained standalone Pi execution closure; Sol medium is sufficient for the final independent contract verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-f2cf8a, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-f2cf8a)
Reviewer cycle 6: changes requested -> analysis. F1 forged or self-minted evidence: the arbitrary selected runtime/direct-child adapter receives the expected nonce, PID/model/capability/target/draft values and may mint the exact JSON treated as authoritative while proxying to an unowned or target-only backend. Add a trusted authority/ownership mechanism and a real-entry perfect-attestation proxy negative plus config/env-echo narrowing mutant. See TASK-260817-2h8hn4_reviewer-verdict-cycle-6.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-f2cf8a, pid=26731, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/high","text":"Cycle-seven rework must remove the impossible self-attestation claim, define project runtime executables as explicit trusted code policy, preserve owned spawn/lifecycle and pre-existing-listener refusal within a same-user threat boundary, and report DFlash as configured rather than independently verified; Sol high is justified by the cross-cutting contract simplification after six review cycles."}
spawn selection rationale for gpt-5.6-sol/high: Cycle-seven rework must remove the impossible self-attestation claim, define project runtime executables as explicit trusted code policy, preserve owned spawn/lifecycle and pre-existing-listener refusal within a same-user threat boundary, and report DFlash as configured rather than independently verified; Sol high is justified by the cross-cutting contract simplification after six review cycles.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-e4f792, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-e4f792)
Cycle 7 architecture rework: replaced self-minted HTTP/runtime attestation v1 with agents-infra-owned adapter authority. The launcher reserves the public listener; only the captured agents-infra executable is the direct adapter child; backend code is admitted through an immutable execution-closure catalog and runs as an owned descendant on private transport; v2 attestation arrives over a private inherited control pipe only after a compiled backend-specific observer proves initialized model/tool/DFlash state. Project TOML cannot provide adapter/observer code, digests, endpoints, or authority. Current DFlash docs expose no independent active-state endpoint, so no reviewed observer means unsupported/unknown diagnostics and pre-side-effect launch refusal. Updated implementation and operator-doc AC with the perfectly formed self-minted proxy attack and config/env-echo narrowing mutant. Official Pi main rechecked at 10acee6045e9025a22dff7e5220ed0d7538f12aa.
Cycle 7 validation: source/outcome SHA-256 parity fb68aa37dde6b26695a5fdc99f9178927099673f2af141aef8f3737ed6f5bf1d; task-board validate reports no issues; git diff --check and Markdown trailing-whitespace checks pass. No product build/test applies because this task explicitly produces architecture research only. The existing decision -> launcher implementation -> alias/docs dependency chain remains the smallest complete board.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-e4f792, pid=28308, exit=0)
Orchestrator rejected cycle-7 producer output before review: it replaced the requested practical trusted-runtime boundary with a compiled adapter/backend observer catalog, made the requested Muse profile unsupported absent a future observer-compatible catalog entry, and materially expanded the implementation beyond the user goal. Rework must treat the explicitly configured absolute runtime executable/argv as trusted project code policy; guarantee launcher spawn/process cleanup, isolated Pi state, exact loopback/readiness checks, and pre-existing-listener refusal within the stated same-UID race boundary; report DFlash provenance as configured/requested and optional runtime-reported, never independently verified. Do not require or invent an agents-infra backend catalog/observer/adapter to launch the two requested profiles.
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/high","text":"Cycle-eight is a constrained architecture correction with an explicit precondition: remove the rejected catalog/observer expansion and restore a truthful trusted-project-runtime boundary while retaining Pi safety gates; Sol high is appropriate to simplify the full artifact consistently after the prior producer divergence."}
spawn selection rationale for gpt-5.6-sol/high: Cycle-eight is a constrained architecture correction with an explicit precondition: remove the rejected catalog/observer expansion and restore a truthful trusted-project-runtime boundary while retaining Pi safety gates; Sol high is appropriate to simplify the full artifact consistently after the prior producer divergence.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-c8ae74, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-c8ae74)
Cycle 8 replaces the cycle-7 backend-catalog/compiled-observer/internal-proxy/private-attestation design with the mandatory practical trust boundary. The revised outcome preserves exact managed Pi identity and argv-parser gates, treats reviewed runtime.executable plus literal argv as trusted executable policy, limits launcher claims to reproducible lifecycle/readiness/state facts, and updates both downstream tasks. The older authoritative-DFlash-attestation checklist sentence is superseded by the attached cycle-8 directive and contract sections 7-12.
Cycle-8 validation: official Pi docs/source pinned at 10acee6045e9025a22dff7e5220ed0d7538f12aa with hashes in section 3; TOML example parses; task-board validate reports no issues; git diff --check and Markdown trailing-whitespace checks pass. Research/outcome and both downstream precondition copies have identical SHA-256 ef14211dacbc627260c0e4dfc40cc4eea87b04d01c1d5ddf523f75628624a47b. No product build/test applies to this no-implementation architecture task. Existing decision -> implementation -> alias/docs dependency chain remains the smallest complete board; no diagram or new element justified.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-c8ae74, pid=35049, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/high","text":"Cycle-eight review must validate the explicitly mandated practical trust boundary, retained Pi identity/argv protections, truthful non-claims, and launchability of both requested profiles without reopening rejected secure-supervisor scope; Sol high is justified for a final cross-cutting contract verdict."}
spawn selection rationale for gpt-5.6-sol/high: Cycle-eight review must validate the explicitly mandated practical trust boundary, retained Pi identity/argv protections, truthful non-claims, and launchability of both requested profiles without reopening rejected secure-supervisor scope; Sol high is justified for a final cross-cutting contract verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-5dbc94, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-5dbc94)
Reviewer cycle 8: changes requested -> analysis. F1 capability claim that does not reproduce: the current artifact preserves a 217-file count and opaque tree-manifest digest but drops the deterministic manifest algorithm/catalog payload required to implement and attack the exact managed Pi execution-closure gate. Restore the complete canonicalization contract, persist authoritative catalog content, and add a canonicalization-narrowing real-entry case. See TASK-260817-2h8hn4_reviewer-verdict-cycle-8.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-5dbc94, pid=38141, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Cycle-nine rework is strictly limited to restoring the deterministic 217-file Pi standalone manifest contract and persisted catalog payload plus a canonicalization-narrowing test, without changing the accepted cycle-eight trust boundary; Sol medium is sufficient."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-nine rework is strictly limited to restoring the deterministic 217-file Pi standalone manifest contract and persisted catalog payload plus a canonicalization-narrowing test, without changing the accepted cycle-eight trust boundary; Sol medium is sufficient.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-acb047, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-acb047)
Cycle 9 resolves reviewer F1 without new board scope: restored deterministic Pi release-root selection, exact 217-record catalog encoding, exhaustive derived directory inventory, path/name/type/link/mode rules, initial and point-of-use verification, and named canonicalization-narrowing acceptance. Added authoritative manifest outcome and identical downstream preconditions; official pinned docs and release checksum reproduced.
Cycle-9 validation evidence attached as TASK-260817-2h8hn4_cycle9-verification.md. Decision/source/downstream parity SHA-256 be8c3482ee78685f6b65d5bfa9802893a1666a032bca8df834fa7935688793d3; manifest parity SHA-256 2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b; TOML parse, official hashes, catalog structure/modes, task-board validate, and git diff --check pass. No product build applies to No implementation scope.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-acb047, pid=41814, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Cycle-nine review is bounded to deterministic Pi release-tree catalog reproducibility, persisted 217-record manifest parity, canonicalization-narrowing evidence, and preservation of the accepted practical runtime boundary; Sol medium is sufficient for the final verdict."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-nine review is bounded to deterministic Pi release-tree catalog reproducibility, persisted 217-record manifest parity, canonicalization-narrowing evidence, and preservation of the accepted practical runtime boundary; Sol medium is sufficient for the final verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-da31f2, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-da31f2)
Reviewer cycle 9: changes requested -> analysis. F1 bypass path around the check: profile names are only non-empty TOML keys but are inserted raw into the managed state path. Traversal escapes the project/profile cache root and normalized aliases share models.json/session.lock, defeating isolation and independent locks. Define an injective safe state key or narrow name domain, prove containment/collision resistance before side effects, add production-entry traversal/normalization negatives plus a narrowing mutant, and propagate downstream. See TASK-260817-2h8hn4_reviewer-verdict-cycle-9.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-da31f2, pid=44645, exit=0)
spawn selection rationale tuple: {"role":"solution-architect","pair":"gpt-5.6-sol/medium","text":"Cycle-ten rework is narrowly limited to injective SHA-256 profile state keys, cache-root containment, path-resolution failure semantics, traversal/collision negatives, and downstream propagation; Sol medium is sufficient without changing the accepted launch architecture."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-ten rework is narrowly limited to injective SHA-256 profile state keys, cache-root containment, path-resolution failure semantics, traversal/collision negatives, and downstream propagation; Sol medium is sufficient without changing the accepted launch architecture.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [analyst] solution-architect (codex) (run=RUN-260817-72b4ad, max_parallel=20)
spawn run started: [analyst] solution-architect (codex) (run=RUN-260817-72b4ad)
Cycle 10 resolves reviewer F1 without expanding the board: raw profile text is no longer a path component. Exact decoded-name UTF-8 bytes map to lowercase SHA-256 state keys; anchored no-follow cache operations, collision detection, containment, and partial/read failure refusal occur before side effects. Added TestPiLaunchProfileStateKeyIsolation with traversal/separator/dot/absolute-looking/Unicode/case/normalization cases and raw/lossy narrowing mutants. Updated both downstream task AC/checklists and synchronized all decision resources. Official Pi docs at pinned revision 10acee6045e9025a22dff7e5220ed0d7538f12aa reverified. Decision/resource SHA-256 b9d92598b5cb92c5d32a434318cfbe056dd37dd1961ba85220b10c785efbfb2d; task-board validate and git diff --check pass. Evidence: TASK-260817-2h8hn4_cycle10-verification.md.
agent completed: [analyst] solution-architect (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-72b4ad, pid=46514, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/medium","text":"Cycle-ten review is bounded to exact UTF-8 SHA-256 profile state keys, anchored cache-root containment, traversal/collision/path-failure negatives, manifest retention, and the accepted practical runtime boundary; Sol medium is sufficient for the final architecture verdict."}
spawn selection rationale for gpt-5.6-sol/medium: Cycle-ten review is bounded to exact UTF-8 SHA-256 profile state keys, anchored cache-root containment, traversal/collision/path-failure negatives, manifest retention, and the accepted practical runtime boundary; Sol medium is sufficient for the final architecture verdict.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-14-gccf0daf; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260817-584591, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260817-584591)
Reviewer cycle 10: accepted. Official Pi pinned-document hashes reproduced; exact TOML/precedence/diagnostics and cycle-8 trusted-runtime boundary verified; profile-state traversal, collision, lossy-normalization, failed-read, separator, identity, foreign-listener, DFlash self-report, cleanup, and manifest-canonicalization negative shapes are covered through production-entry acceptance requirements. The 217-record manifest reproduced byte-for-byte, downstream decision copies are identical, dependencies are correct, task-board validate and git diff --check pass. Evidence: TASK-260817-2h8hn4_reviewer-verdict-cycle-10.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260817-584591, pid=48360, exit=0)

## Precondition Resources
- [official-pi-models-doc](file://TASK-260817-2h8hn4/official-pi-models-doc) — Official Pi custom-model contract.
- [official-pi-settings-doc](file://TASK-260817-2h8hn4/official-pi-settings-doc) — Official Pi global and project settings contract.
- [official-pi-usage-doc](file://TASK-260817-2h8hn4/official-pi-usage-doc) — Official Pi CLI model selection and environment contract.
- [cycle-8-trust-boundary-directive.md](file://TASK-260817-2h8hn4/cycle-8-trust-boundary-directive.md) — Mandatory practical trust boundary; rejects cycle-7 backend catalog and preserves launchability of Qwen and Muse profiles.

## Outcome Resources
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-08bda1.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-08bda1.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_pi-local-model-launch-contract.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_pi-local-model-launch-contract.md) — Cycle-10 Pi launch contract with exact UTF-8 profile state keys, anchored cache containment, deterministic Pi catalog, and practical trusted-runtime boundary
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-252f62.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-252f62.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-1.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-1.md) — Reviewer changes-requested verdict with official-source and negative-evidence findings
- [TASK-260817-2h8hn4_pi-story-plan.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_pi-story-plan.md) — Saved three-phase Pi story plan referenced by the architecture decision.
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-f98913.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-f98913.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-3efee7.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-3efee7.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-2.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-2.md) — Reviewer cycle 2 changes-requested verdict: Pi parser separator bypass
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-ef6255.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-ef6255.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-2de4a8.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-2de4a8.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-3.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-3.md) — Reviewer cycle 3 changes-requested verdict: Pi parser identity compatibility gate is absent
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-d5fc0c.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-d5fc0c.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-1852d9.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-1852d9.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-4.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-4.md) — Reviewer cycle 4 changes-requested verdict: verified package is not bound to the executed Node program
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-79bc76.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-79bc76.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-400da7.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-400da7.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-5.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-5.md) — Reviewer cycle 5 changes-requested verdict: Qwen readiness is not bound to the owned runtime child
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-10b3f2.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-10b3f2.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-f2cf8a.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-f2cf8a.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-6.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-6.md) — Reviewer cycle 6 changes-requested verdict: runtime attestation authority is self-minted
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-e4f792.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-e4f792.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-c8ae74.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-c8ae74.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-5dbc94.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-5dbc94.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-8.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-8.md) — Reviewer cycle 8 changes-requested verdict: managed Pi tree identity catalog is not reproducible from authoritative task evidence
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-acb047.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-acb047.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_pi-v0.84.2-darwin-arm64-tree-manifest.txt](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_pi-v0.84.2-darwin-arm64-tree-manifest.txt) — Authoritative 217-record canonical manifest for the supported Pi v0.84.2 darwin-arm64 standalone release tree
- [TASK-260817-2h8hn4_cycle9-verification.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_cycle9-verification.md) — Cycle-9 official-source, catalog reproduction, parity, TOML, board, and diff verification evidence
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-da31f2.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-da31f2.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-9.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-9.md) — Reviewer cycle 9 changes-requested verdict: raw profile names escape or alias managed state paths
- [TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-72b4ad.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-analyst--solution-architect--codex-_RUN-260817-72b4ad.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_cycle10-verification.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_cycle10-verification.md) — Cycle-10 official-source, profile-state-key, downstream propagation, parity, and board validation evidence
- [TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-584591.log](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_spawn-log_-reviewer--reviewer--codex-_RUN-260817-584591.log) — System spawn log captured by task-board
- [TASK-260817-2h8hn4_reviewer-verdict-cycle-10.md](file://TASK-260817-2h8hn4/TASK-260817-2h8hn4_reviewer-verdict-cycle-10.md) — Reviewer cycle 10 accepted verdict with official-source, manifest reproduction, downstream parity, and gate-defeat evidence

## Created
2026-08-17T10:15:27Z

## Last Update
2026-08-17T12:38:48Z

## Assigned To
[reviewer] reviewer (codex)
