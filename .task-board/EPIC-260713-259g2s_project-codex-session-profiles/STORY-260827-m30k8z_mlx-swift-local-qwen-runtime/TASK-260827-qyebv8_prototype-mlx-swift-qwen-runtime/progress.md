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
- TASK-260827-2h39ya
- TASK-260827-2q77g8
- TASK-260827-2v13w8

## Checklist
- [x] Swift prototype builds on macOS with pinned official mlx-swift and mlx-swift-lm revisions recorded verbatim
- [x] Prototype loads the exact configured local Qwen model and reports load time and resident memory
- [x] Bounded non-streaming and streaming chat-completion smokes pass over the OpenAI-compatible contract used by the Pi profile
- [x] Bounded tool-call smoke returns a well-formed tool_calls payload; reasoning-content behavior recorded
- [x] Models listing plus startup, readiness and shutdown lifecycle answer as the model-harness managed contract expects
- [x] Unsupported model, tokenizer, chat-template or sampler behavior is recorded as an explicit named gap list, not silently worked around
- [x] Python mlx-lm remains the default local runtime; no default profile or config switch is made
- [x] Exact build, run and smoke commands with their outputs are attached as an outcome resource
- [x] Code written per task description and AC
- [x] Relevant tests written for new or changed behavior and passing
- [x] Gating, refusing, validating, authorizing, or attesting behavior covered by negative tests that fail when the gate admits what it must reject, with the production call site named
- [x] Lint clean
- [x] Relevant build/validation commands run after changes and build not broken
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/xhigh","text":"Greenfield MLX Swift LM runtime prototype against unfamiliar Swift/Metal APIs with tool-call, streaming and lifecycle parity requirements needs the strongest admitted Claude pair; cheaper pairs would raise rework and review cost."}
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/xhigh","text":"Greenfield MLX Swift LM runtime prototype against unfamiliar Swift and Metal APIs with tool-call, streaming and lifecycle parity requirements needs the strongest admitted Claude pair; cheaper pairs would raise rework and review cost."}
spawn selection rationale for claude-opus-5/xhigh: Greenfield MLX Swift LM runtime prototype against unfamiliar Swift and Metal APIs with tool-call, streaming and lifecycle parity requirements needs the strongest admitted Claude pair; cheaper pairs would raise rework and review cost.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260827-5156c5, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260827-5156c5)
Prototype built and green: swift build -c release exit=0, swift test -c release exit=0 (73 tests/8 suites), swift format lint --strict exit=0, go build/vet/gofmt exit=0. Ten gate-narrowing mutants all CAUGHT. Lifecycle smoke 17/17 PASS with zero GPU memory (unloadable fixture). BLOCKING FOR FULL-MODEL SMOKE: the shared Python mlx_lm.server for the same model is resident (pid 78541, 23.1 GiB RSS, 43 GiB wired) with one actively heartbeating lease from client_pid 12116; only ~9 GiB free, so the 27.5 GiB MLX Swift load cannot be attempted without destabilising another sessions live runtime. Polling for the lease to clear.
READY FOR REVIEW with one unproven area, stated explicitly.

DELIVERED: tools/mlx-swift-runtime-prototype (SwiftPM, 37 files). MLX Swift LM runtime serving the configured local Qwen model over the Pi profile OpenAI-compatible contract. Pinned exact: mlx-swift 0.31.6 (0bb916c67f4b9e5c682cbe02a42c701c93ab5021), mlx-swift-lm 3.31.4 (bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57), swift-transformers 1.3.3, swift-nio 2.99.0; full graph in committed Package.resolved.

GREEN (real exit codes, standalone processes): swift build -c release 0; swift test -c release 0 (77 tests/8 suites); swift format lint --strict 0; go build 0; go vet 0; gofmt -l 0; go test ./... -count=1 0.

NEGATIVE EVIDENCE: 12 narrowing mutants applied to production code, all CAUGHT. M1 was NOT caught on first attempt (identity tests missed prefix/substring/case neighbours); test cases added, then M1/M1b/M1c caught. M9/M10 cover the new install-sync exclusion.

PROVEN WITHOUT WEIGHTS: preflight exit 0 on the exact model — qwen3_5 registered in MLXVLM+MLXLLM at 3.31.4, model config.json decodes into MLXVLM.Qwen35Configuration, tokenizer loads, chat template renders 260 tokens with tools, prompt ends inside an open <think> block, tool format xml_function. Lifecycle smoke 17/17. model-harness run managed path: direct-child ownership, forwarded JSON events, readiness answer, SIGTERM group shutdown, port released.

REGRESSION FOUND AND FIXED AT SOURCE: the new .build directory broke setup local (shouldSkip uses a deny-list, not .gitignore; mode-0400 bundle resource -> permission denied). Fixed in internal/infra/infra.go:467 with source_sync_build_artifacts_test.go.

NOT PROVEN (checklist items 2,3,4,5,8): full-model load and generation smokes. The model needs ~27.5 GiB mostly-wired; the host already holds one copy (Python mlx_lm.server pid 78542, 23.1 GiB RSS, 40-43 GiB wired, 5.6-10.7 GiB free across six samples) under an actively heartbeating lease owned by pid 12116 = the operator interactive agents-infra target qwen-infra -- --continue session in zellij. Not stopped. scripts/smoke.sh is written and syntax-checked; it needs a host not already serving this model.

DEFAULTS UNCHANGED: model-harness.toml and project-config.toml untouched; prototype profile is task-scoped only; binary not installed. Python mlx-lm remains the rollback path.

NOT COMMITTED: version_control.confirm=true, so changes are left in the story worktree for owner confirmation.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260827-5156c5, pid=20674, exit=0)
No Change Request revision was published for TASK-260827-qyebv8 (handoff_unsatisfied): the board is not at to-review
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Resuming the blocked prototype run to finish the full-model load and generation smokes now that host memory is free; the code already exists and is green, so this is measurement and evidence capture rather than fresh design."}
spawn selection rationale for claude-opus-5/high: Resuming the blocked prototype run to finish the full-model load and generation smokes now that host memory is free; the code already exists and is green, so this is measurement and evidence capture rather than fresh design.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260827-5add7a, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260827-5add7a)
RESUME RUN (RUN-260827-5add7a) FINDING — swift build produces a non-runnable binary. Host memory is now free (35.4 GiB free, 0 mlx_lm.server processes, agents-infra runtime status: broker absent, leases 0), so the full-model smoke was attempted. It failed at model load, NOT on memory: MLX aborted with "Failed to load the default metallib. library not found (x4)" at Cmlx/mlx-c/mlx/c/stream.cpp:115. Root cause is upstream and documented: mlx-swift README states verbatim "SwiftPM (command line) cannot build the Metal shaders so the ultimate build has to be done via Xcode" and "Although SwiftPM (command line) cannot build the Metal shaders, xcodebuild can". Confirmed in the 0.31.6 checkout: Cmlx declares no resources, no .metallib exists anywhere under .build, the PrepareMetalShaders plugin referenced by Package.swift:265 no longer ships, and device.cpp resolves default.metallib from a mlx-swift_Cmlx.bundle it expects Xcode to emit. So the prototype README build instruction (swift build -c release) yields a binary that can never load a model. Lifecycle design is vindicated by the same run: the listening event fired and the port bound BEFORE the load, so the managed readiness poll saw a live socket rather than a refused connection. Switching the build to the documented xcodebuild path and re-running the full-model smoke.
RESUME RUN COMPLETE — all 15 checklist items now proven. READY FOR REVIEW.

FULL-MODEL SMOKE OK (25 PASS / 0 FAIL) against the real 28 GB Qwen through model-harness run: load 6.977s, physical footprint 28,261.4 MiB, MLX active/peak 29,501,612,496 B, first 503 at 2s, ready at 8s. Non-streaming (usage 17+34=51, marker, reasoning split out), streaming (35 frames / 34 chunks, one finish frame, one usage frame, [DONE], no </think> leak), tool-call (finish_reason=tool_calls, well-formed payload with call_ id), all four live refusals, SIGTERM exit 143 in 1s, port released, stopped event. Lifecycle smoke 0 failures. Preflight exit 0.

DEFECT FOUND AND FIXED — the prototype README told you to build with swift build, which produces a binary that can NEVER load a model. mlx-swift README says it verbatim: SwiftPM (command line) cannot build the Metal shaders. Confirmed in the pinned 0.31.6 checkout: Cmlx declares no resources, no .metallib anywhere under .build, PrepareMetalShaders referenced at Package.swift:265 no longer ships. Correct path is xcodebuild with BOTH -skipPackagePluginValidation (CudaBuild plugin) and -skipMacroValidation (MLXHuggingFaceMacros), plus the separately downloaded Metal Toolchain (~688 MB, installed this run). Build docs corrected in the prototype README and the root README tools table.

NEW GATE — MetalShaderLibraryCheck refuses the serve path BEFORE the listener binds, so a swift-build product exits 2 with an actionable message instead of aborting mid-load with a bound port. Verified at the real entry point, not only in unit tests: the surviving .build/release binary exits 2 and leaves no listener on 18019. The gate treats an unreadable search root as undetermined and does NOT refuse on it, because failing to read a directory is not evidence of absence.

SECOND REGRESSION FIXED — xcodebuild writes a 2.6 GB DerivedData tree into the source dir, repeating the .build hazard that broke setup local. DerivedData added to shouldSkip (infra.go), two tests, and to the prototype .gitignore.

MUTATION EVIDENCE — 6 mutants, all CAUGHT: M1 unreadable-counts-as-absence, M2 admit-stops-throwing, M3 bundle-dir-accepted-instead-of-file, M4 executable-dir-dropped-from-search, Go-A delete-DerivedData-exclusion, Go-B narrow-to-strings.Contains. M4 was NOT caught on the first attempt — the original test asserted against defaultSearchRoots(), where a test hosts bundle URL and executable directory coincide, so the mutant was unobservable; extracted a pure composeSearchRoots and asserted there. Reported because a test that cannot fail is not evidence.

RECORDED, NOT WORKED AROUND — resident_bytes read 14,056.9 / 10,774.2 / 2,650.2 MiB across three identical loads while physical footprint stayed within 16 MiB and MLX active bytes was byte-identical; size hosts from the footprint, not the resident figure. reasoning is published under reasoning, not reasoning_content.

GATES (real exit codes, standalone): swift build 0; swift test 0 (92 tests / 9 suites, was 77); swift format lint --strict 0; xcodebuild BUILD SUCCEEDED with 0 error lines; preflight 0; smoke.sh 0; lifecycle-smoke.sh 0; go build 0; go vet 0; gofmt -l 0 files; go test ./internal/infra/... 0.

DEFAULTS UNCHANGED — model-harness.toml and project-config.toml untouched; the prototype profile lives only in .temp/TASK-260827-qyebv8/ and is passed with --config; nothing installed, no binary on PATH. Python mlx-lm remains the default local runtime and rollback path.

NOT COMMITTED — version_control.confirm=true, so changes are left in the story worktree for owner confirmation.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260827-5add7a, pid=26099, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Independent-provider review of a Change Request whose central claims are build-toolchain correctness and measured runtime evidence; a different model family reduces the chance of inheriting the producer's blind spots."}
spawn selection rationale for gpt-5.6-sol/xhigh: Independent-provider review of a Change Request whose central claims are build-toolchain correctness and measured runtime evidence; a different model family reduces the chance of inheriting the producer's blind spots.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260827-4e72d6, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260827-4e72d6)
agent completed: [reviewer] reviewer (codex) (exit=1)
spawn run completed: codex (run=RUN-260827-4e72d6, pid=93296, exit=1)
spawn autonomous recovery: run RUN-260827-4e72d6 queued successor RUN-260827-bd02c2 (attempt 1/3, model=gpt-5.6-sol): spawned agent exited with code 1
spawn run started: [reviewer] reviewer (codex) (run=RUN-260827-bd02c2)
Reviewer revision 1: CHANGES REQUESTED. F1 max_tokens=1e300 aborts the production decoder (exit 133) instead of returning a refusal. F2 a directory named default.metallib passes the production Main startup gate and binds the listener (forged evidence). F3 remove the 6,180,616-byte generated default.profraw and ignore profile output. F4 refresh or supersede the stale primary results artifact. Evidence: TASK-260827-qyebv8_review-verdict.md and TASK-260827-qyebv8_review-attack-probes.md.
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260827-bd02c2, pid=99648, exit=0)
spawn selection rationale tuple: {"role":"developer","pair":"claude-opus-5/high","text":"Rework of four reviewer findings on code this producer lineage already owns: two admission-path defects needing total numeric conversion and regular-file gating with named negative tests, plus artifact hygiene."}
spawn selection rationale for claude-opus-5/high: Rework of four reviewer findings on code this producer lineage already owns: two admission-path defects needing total numeric conversion and regular-file gating with named negative tests, plus artifact hygiene.
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260827-a388ff, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260827-a388ff)
REWORK COMPLETE (RUN-260827-a388ff) — all four reviewer findings fixed and attacked. READY FOR REVIEW.

F1 out-of-range max_tokens aborted the runtime: ChatCompletionRequest.optionalInt now converts with Int(exactly:) and throws a new numberOutOfRange decoding error, which Router.chatCompletions already maps to 400 invalid_body. Proven at the LIVE ready HTTP entry on the real 28 GB model, not only in unit tests: 1e300, -1e300 and 2^63 each return 400 with the out-of-range message AND a following /health returns 200, because a bounded 400 from a process that then dies proves nothing. Control: max_tokens 0 still returns invalid_max_tokens, so the gate rejects unrepresentable values rather than merely large ones. These checks are now permanent in scripts/smoke.sh.

F2 the Metal gate accepted a directory as forged evidence: new FileObjectProbe.swift uses POSIX stat instead of Foundation, because the distinctions that matter are the ones Foundation blurs — stat follows symlinks (a link to a real library is admitted, a dangling one is not) and its errno separates ENOENT (proven absence) from EACCES (never absence). inspect gained libraryNotAFile(path:kind:), classify carries it into absent(searched:rejected:), and the refusal names what is actually there instead of saying not found. An unreadable library path still yields undetermined and does NOT refuse. New scripts/metallib-gate-probe.sh drives the COMPOSED EXECUTABLE with a forged directory and a dangling symlink: exit 2, no listener ever bound, 10 PASS / 0 FAIL. Its fourth case plants a real file and requires the gate to ADMIT, so a blanket-refusing gate could not pass the probe.

F3 default.profraw (6,180,616 B) deleted; *.profraw and *.profdata added to the package .gitignore. Candidate subtree is now 41 paths with no .build, DerivedData, .temp or profile output.

F4 TASK-260827-qyebv8_results.md rewritten: single current status statement, revision-3 gate table, section 5 replaced by the green full-model results, blocker artifact explicitly labelled historical. New artifact TASK-260827-qyebv8_review-rework-rev2.md carries the per-finding detail.

HARDENING BEYOND THE FOUR FINDINGS — ModelDirectoryCheck.observe had the identical forged-evidence shape one level up: it matched required files by NAME IN A DIRECTORY LISTING, so mkdir config.json or a directory named *.safetensors passed admission on the same startup path. Closed in the same change rather than left for a third round: entries and weights are stat-probed, a new notRegularFiles observation names what was found, per-entry EACCES yields unreadable rather than incomplete, and admitsSymlinkedFiles keeps Hugging Face style symlinked trees working.

MUTATION EVIDENCE — 6 narrowing mutants, all CAUGHT: M-A existence-only probe; M-B first stat failure laundered into absence; M-B2 every stat failure laundered into absence; M-C S_IFDIR classified as regular file (the exact reviewer defect); M-D out-of-range clamped instead of refused; M-E Int(exactly:) reverted to Int(_:) — killed the TEST PROCESS with signal 5, reproducing the reviewer exit 133; M-F only the positive side of the range gated. M-A and M-C were additionally driven through the composed executable and reproduced the reviewer observation exactly (listener bound on 28117), absent once restored.

TWO SELF-REPORTED WEAKNESSES, BOTH FIXED — (1) M-B did not fail the Metal test it should have owned, because the lstat fallback still returned unreadable in that fixture; M-B2 was written to remove both guards and confirm it. (2) The first version of unreadableEntryIsNotMissing chmod-0 the model directory, which makes contentsOfDirectory fail first, so the per-entry probe was never reached — a test that could not fail for its stated reason. Rewritten to symlink a required file into a 0o000 subdirectory, keeping the directory listable, and it now asserts the listing is still readable so the precondition itself is checked.

GATES (real exit codes, standalone processes, no pipes): swift build -c release 0; swift test -c release 0 (103 tests / 9 suites, was 92); swift format lint --recursive --strict 0; xcodebuild BUILD SUCCEEDED with 0 error lines; bash -n on all three scripts 0; preflight 0; scripts/smoke.sh 0 (SMOKE OK, 36 PASS / 0 FAIL); scripts/lifecycle-smoke.sh 0; scripts/metallib-gate-probe.sh 0; go build ./... 0; go vet ./internal/infra/... 0; gofmt -l on both changed Go files 0 no output; go test ./internal/infra/... -count=1 0 in 124.040s.

FULL-MODEL SMOKE (rev 3): load 6.738s, physical footprint 29,633,484,064 B (28,260.7 MiB), first 503 at 2s, ready at 8s, streaming 35 frames / 34 chunks, finish_reason=tool_calls with a well-formed call_ payload, SIGTERM exit 143 in 1s, port released, stopped event. The hardened gates did not break the real path: preflight metal_shader_library passed on the xcodebuild product and the model loaded and generated.

REAL-PATH REGRESSION CHECK — the real DerivedData default.metallib is a regular file and every required file in the configured model directory is a regular file, verified before tightening either gate.

LOGBOOK — three entries added: the two forged-evidence gates, the out-of-range abort, and the test-that-cannot-fail finding.

DEFAULTS UNCHANGED — model-harness.toml and project-config.toml untouched, the prototype profile lives only in .temp/TASK-260827-qyebv8/ and is passed with --config, nothing installed on PATH. Python mlx-lm remains the default local runtime and rollback path.

NOT COMMITTED — version_control.confirm=true, so changes are left in the story worktree for owner confirmation.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260827-a388ff, pid=11330, exit=0)
spawn selection rationale tuple: {"role":"reviewer","pair":"gpt-5.6-sol/xhigh","text":"Second-round review by the same independent provider that produced the round-1 findings, so closure is judged against the original reviewer's own standard and probes."}
spawn selection rationale for gpt-5.6-sol/xhigh: Second-round review by the same independent provider that produced the round-1 findings, so closure is judged against the original reviewer's own standard and probes.
spawn agent resolution: Agent selection: codex via explicit_override (preferred_agentic_system: mixed[claude,codex], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=codex; schema=1; producer=v1.6.1-44-gd91d6fc; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (codex) (run=RUN-260827-5606e8, max_parallel=20)
spawn run started: [reviewer] reviewer (codex) (run=RUN-260827-5606e8)
agent completed: [reviewer] reviewer (codex) (exit=0)
spawn run completed: codex (run=RUN-260827-5606e8, pid=26992, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260827-qyebv8_spawn-log_-implementer--developer--claude-_RUN-260827-5156c5.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_spawn-log_-implementer--developer--claude-_RUN-260827-5156c5.log) — System spawn log captured by task-board
- [TASK-260827-qyebv8_results.md](file://TASK-260827-qyebv8/TASK-260827-qyebv8_results.md) — MLX Swift LM Qwen runtime prototype summary (revision 3): current status, gate table, full-model smoke results, named gap list; points at the full-model and rework artifacts
- [TASK-260827-qyebv8_preflight.json](file://TASK-260827-qyebv8/TASK-260827-qyebv8_preflight.json) — Final preflight on the real model incl. metal_shader_library stage (exit 0)
- [TASK-260827-qyebv8_lifecycle-smoke.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_lifecycle-smoke.log) — Verbatim final lifecycle-smoke.sh output (0 failures, exit 0)
- [TASK-260827-qyebv8_managed-run.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_managed-run.log) — model-harness run managed-path transcript: direct-child ownership, forwarded JSON events, readiness answer and SIGTERM group shutdown
- [TASK-260827-qyebv8_blocker.md](file://TASK-260827-qyebv8/TASK-260827-qyebv8_blocker.md) — Blocker record: host memory contention with the operator live Qwen session, evidence, attempts made, and the exact input needed to finish items 2-5
- [TASK-260827-qyebv8_spawn-log_-implementer--developer--claude-_RUN-260827-5add7a.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_spawn-log_-implementer--developer--claude-_RUN-260827-5add7a.log) — System spawn log captured by task-board
- [TASK-260827-qyebv8_full-model-smoke.md](file://TASK-260827-qyebv8/TASK-260827-qyebv8_full-model-smoke.md) — Full-model load/streaming/tool-call/lifecycle evidence, swift-build metallib finding, xcodebuild fix, mutation results
- [TASK-260827-qyebv8_smoke-full-model.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_smoke-full-model.log) — Verbatim final scripts/smoke.sh output against the real Qwen model (SMOKE OK, 25 PASS / 0 FAIL)
- [TASK-260827-qyebv8_change-request_rev1.patch](file://TASK-260827-qyebv8/TASK-260827-qyebv8_change-request_rev1.patch) — Change Request CR-TASK-260827-qyebv8-1 revision 1 candidate patch (repository_delta=present, 44 changed paths)
- [TASK-260827-qyebv8_spawn-log_-reviewer--reviewer--codex-_RUN-260827-4e72d6.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_spawn-log_-reviewer--reviewer--codex-_RUN-260827-4e72d6.log) — System spawn log captured by task-board
- [TASK-260827-qyebv8_spawn-log_-reviewer--reviewer--codex-_RUN-260827-bd02c2.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_spawn-log_-reviewer--reviewer--codex-_RUN-260827-bd02c2.log) — System spawn log captured by task-board
- [TASK-260827-qyebv8_review-verdict.md](file://TASK-260827-qyebv8/TASK-260827-qyebv8_review-verdict.md) — Revision 1 reviewer verdict: changes requested with exact candidate identity, findings, reruns, and required rework
- [TASK-260827-qyebv8_review-attack-probes.md](file://TASK-260827-qyebv8/TASK-260827-qyebv8_review-attack-probes.md) — Reviewer negative probes reproducing max_tokens process abort and forged Metal-library gate admission
- [TASK-260827-qyebv8_spawn-log_-implementer--developer--claude-_RUN-260827-a388ff.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_spawn-log_-implementer--developer--claude-_RUN-260827-a388ff.log) — System spawn log captured by task-board
- [TASK-260827-qyebv8_review-rework-rev2.md](file://TASK-260827-qyebv8/TASK-260827-qyebv8_review-rework-rev2.md) — Revision 3 rework: F1-F4 fixes, live-entry proofs, 6 narrowing mutants, ModelDirectoryCheck hardening
- [TASK-260827-qyebv8_smoke-full-model-rev3.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_smoke-full-model-rev3.log) — Verbatim scripts/smoke.sh against the real Qwen after rework (SMOKE OK, 36 PASS / 0 FAIL) incl. live out-of-range max_tokens refusals
- [TASK-260827-qyebv8_metallib-gate-probe.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_metallib-gate-probe.log) — Forged-metallib probe at the composed executable entry point (10 PASS / 0 FAIL)
- [TASK-260827-qyebv8_lifecycle-smoke-rev3.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_lifecycle-smoke-rev3.log) — Lifecycle smoke after rework (0 failures)
- [TASK-260827-qyebv8_preflight-rev3.json](file://TASK-260827-qyebv8/TASK-260827-qyebv8_preflight-rev3.json) — Preflight on the real model after the hardened gates (exit 0, metal_shader_library passed)
- [TASK-260827-qyebv8_change-request_rev2.patch](file://TASK-260827-qyebv8/TASK-260827-qyebv8_change-request_rev2.patch) — Change Request CR-TASK-260827-qyebv8-2 revision 2 candidate patch (repository_delta=present, 45 changed paths)
- [TASK-260827-qyebv8_spawn-log_-reviewer--reviewer--codex-_RUN-260827-5606e8.log](file://TASK-260827-qyebv8/TASK-260827-qyebv8_spawn-log_-reviewer--reviewer--codex-_RUN-260827-5606e8.log) — System spawn log captured by task-board
- [TASK-260827-qyebv8_review-verdict-rev2.md](file://TASK-260827-qyebv8/TASK-260827-qyebv8_review-verdict-rev2.md) — Round 2 reviewer verdict with independent live, entry-point, and narrowing-mutant evidence

## Created
2026-08-27T10:29:10Z

## Last Update
2026-08-27T18:35:11Z

## Assigned To
[reviewer] reviewer (codex)
