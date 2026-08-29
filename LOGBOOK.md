# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-08-25

### 2022 — Qwen Thinking Is Native, Pi Yolo Is Not
- MILESTONE: Canonical Qwen non-`off` reasoning now requires a reasoning-capable Pi profile with `qwen-chat-template`; source-backed `medium` composes to native `--thinking medium`.
- FIX: `agents.pi.primary_session.yolo_mode` composes with nearest-field precedence, defaults safely to false, and rejects true before executable lookup or launch with `pi_yolo_mode_unsupported` because pinned Pi exposes no unattended tool policy.
- EVIDENCE: Full Go tests, vet, Darwin and Windows builds, and non-launching compose/print checks exit 0; narrowing the yolo gate and Qwen thinking-format gate makes their production-entrypoint tests fail with exit 1 before exact restoration.
- SCOPE: `TASK-260825-kpky8f`; no Pi runtime launch or external model execution.

### 2009 — Pi Approve Is Project Trust, Not Yolo
- FINDING: Pinned Pi `0.84.2` parses `--approve` into `projectTrustOverride`; its docs define the flag only as one-run trust for project-local files and expose no native unattended tool-execution policy.
- FINDING: Qwen thinking is native when the generated model has `reasoning=true`, `compat.thinkingFormat=qwen-chat-template`, and launch argv selects `--thinking medium`; Pi then sends `chat_template_kwargs.enable_thinking=true` with `preserve_thinking=true`.
- DECISION: `agents.pi.primary_session.yolo_mode=true` must fail before executable lookup or launch; it must never map to `--approve`. Omitted/false remains the compatible safe policy.
- SCOPE: `TASK-260825-kpky8f`; Pi config parsing, primary-session compose/launch gates, Qwen target validation, tests, README, and `SKILL.md`.

### 1449 — Model-check Operator Contract Is Executably Pinned
- REGRESSION: The bounded model-check README and skill sections could be deleted while the full uncached Go suite remained green, so unattended `--approve`, evidence, timeout, cleanup, and exit semantics had no drift guard.
- FIX: Dedicated doc-contract tests pin both sections, derive deadline and exit fragments from production constants, and cover exact artifacts, modes, overwrite refusal, read-evidence limits, cleanup, and exit-5 precedence.
- EVIDENCE: Narrowing the README precedence clause and the skill's failed-read-versus-absence rule made their named uncached tests fail with exit 1; byte-for-byte restoration and the unmutated tests then passed.
- SCOPE: `TASK-260825-39ycg2`; no model-check runtime behavior changed.

### 1409 — Qwen Skill Read Proved Before Deadline
- FINDING: Installed `agents-infra model-check` observed a completed non-error `read` of `$HOME/.agents/skills/relux-agents-infra/SKILL.md` through the real `qwen-infra` target.
- ANOMALY: The `5m` smoke timed out with exit `2` after `300192ms`; the response marker was unmet and the valid event stream remained incomplete.
- EVIDENCE: Both owned process-group cleanup states are `confirmed`; sanitized outcome belongs to `TASK-260825-39ycg2`, while raw JSONL/stderr remain local mode-`0600` evidence.
- DECISION: Skill discovery/read is proven; final-response behavior is not. Do not treat the absent marker as no skill read or the successful read as an overall passing check.

### 1340 — Load Fixtures Must Own Their Process Cleanup
- ANOMALY: Reviewer cycle 2 left eight busy-loop load-fixture processes in one process group after timing-flake diagnosis.
- EVIDENCE: Orchestrator terminated the exact fixture PIDs and verified the process group empty before rework continued.
- DECISION: Do not reproduce the unbounded load loop; cold-start stability uses an early PID marker, 2s deadline headroom, and six uncached production-entrypoint repetitions.
- SCOPE: Review fixture only; no production process leak observed.

### 1338 — Exit-Code Interfaces Need Nominal Ownership
- REGRESSION: `main` matched any `ExitCode() int`, so provider `*exec.ExitError` values changed established Codex, Claude, and target CLI exits outside model-check scope.
- FIX: `tools/agents-infra/main.go` recognizes only `*infra.ModelCheckFailure`; a production-binary Codex child exiting 42 must leave the wrapper at legacy exit 1.
- EVIDENCE: Reintroducing the structural interface makes `TestMainKeepsProviderChildFailuresAtLegacyExitOne` fail with observed exit 42.
- SCOPE: `TASK-260825-rtmcsw`; model-check retains its documented exits 1–5.

### 1338 — Cleanup Attestation Must Observe Live OS State
- ROOT CAUSE: Synthetic pending/failed report tests proved only the evaluator; a constant `processGroupCleanupState = confirmed` producer survived the suite.
- FIX: `tools/agents-infra/internal/infra/pi_test.go` probes a live process group as failed and the same reaped group as confirmed; the cold-start fixture writes its PID before heavy imports.
- EVIDENCE: The constant-confirmed mutant fails the named producer test; the readiness deadline case passes 6/6 uncached runs.
- SCOPE: `TASK-260825-rtmcsw`; no external model or runtime download.

### 1300 — Model Check Gates Need Boundary Proof
- REGRESSION: A fixed prompt and universally true cleanup attestation both survived the original positive production suite.
- FIX: `tools/agents-infra/model_check_main_test.go` captures the provider request body; `internal/infra/model_check_test.go` pins pending/failed cleanup to exit 1.
- EVIDENCE: Narrowed prompt, cleanup, deadline, raw-overwrite, and non-managed-target mutants each fail a named production or lifecycle test.
- SCOPE: `TASK-260825-rtmcsw`; no external model download.

### 1210 — Bounded Checks Need Process-Group Evidence
- ROOT CAUSE: Managed `RunPi` bounded only signal and runtime cleanup; normal Pi completion waited for the leader and exposed no machine-readable proof that either owned process group was gone.
- FIX: `tools/agents-infra/internal/infra/pi_launch_posix.go:239` routes Pi and runtime through bounded TERM-to-SIGKILL group cleanup; `internal/infra/model_check.go:144` records sanitized cleanup state with every behavior-check outcome.
- EVIDENCE: Production-binary timeout fixture leaves both an ignore-TERM `bash` descendant and runtime descendant; the command exits with the timeout code, reports cleanup confirmed, and direct PID probes return `ESRCH` for every recorded process.
- SCOPE: `TASK-260825-rtmcsw`; no duplicate shell launcher or external model download.

## 2026-08-24

### 2152 — Local Setup Honors Provider-Owned Skill Links
- ROOT CAUSE: `managedSkillLinkFailures` filtered provider surfaces by source-managed names only in global mode, so local setup contradicted the documented ownership boundary and rejected preserved external packages such as `mac-infra`.
- FIX: `tools/agents-infra/internal/infra/skill_link_validation.go` now derives managed names from `.agents/.skills` for both modes; production CLI tests preserve unmanaged links while still refusing a narrowed managed-name gate.
- EVIDENCE: Full Go suite, vet, build, source/global/local setup, and both verifies exit 0; the observed `casual-talks` external links remain unchanged.
- SCOPE: `TASK-260824-2a4gk3`; no relaxation inside source-managed skill packages.

### 2151 — Casual Talks Qwen Target Runs Text And Tools
- MILESTONE: Installed `openai-infra`, `anthropic-infra`, and `qwen-infra` resolve the configured target tuples in `/Users/alexis/src/casual-talks`; print/compose remain non-launching and preserve config bytes.
- EVIDENCE: Real Qwen/Pi run on mlx-lm 0.31.3 emits `TEXT_RESPONSE_OK`, completes successful `write` and `read` tool results on a task-scoped 39-byte file, emits `TOOL_ROUNDTRIP_OK`, and exits 0.
- FINDING: Runtime bound only `127.0.0.1:18011`; post-run listener, runtime process, and profile-lock holder checks all return expected absence.
- SCOPE: `TASK-260824-2a4gk3`; no secrets or arbitrary environment values persisted.

### 2109 — Absolute MLX Identity Unblocks Project Target Rollout
- DECISION: Operator selected `/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit` as the exact Qwen target/profile identity because real `mlx_lm.server` uses that resolved path for load, requests, and `/v1/models` readiness.
- MILESTONE: `TASK-260824-1jjze0` atomically rewrote all 121 in-scope project configs after a real-server exact-ID plus completion gate; 363 post-apply production alias composes and rollback-readiness verification exit 0.
- FINDING: Per-file backup/current/candidate hashes and raw MCP comparisons pass for all 121 configs; unrelated status lines across 116 Git worktrees remain unchanged.
- SCOPE: Task-only script and evidence under `.temp/TASK-260824-1jjze0`; no runtime migration path added.

### 2051 — Canonical Qwen Selector Is Not An MLX Runtime Identity
- ROOT CAUSE: Real `mlx_lm.server --model Qwen3.8-27B-MLX-8bit` treats the required canonical model ID as a Hugging Face repo, receives `RepositoryNotFound`, and keeps an empty HTTP shell alive; the local weights live at a differently named absolute path.
- FINDING: MLX reports a local model at `/v1/models` by resolved absolute path; managed Pi correctly reaches that route as `base_url + readiness_path`, but `waitPiRuntimeReady` requires the inventory ID to equal the canonical profile model exactly.
- EVIDENCE: `TASK-260824-1jjze0` full dry-run validates 121 candidates through 363 production alias compose calls; real apply gate exits 3 before writes, records zero writes, and post-refusal hashes match all 121 sources.
- BLOCKED: Recursive rollout requires an architecture decision that separates public target model identity from the MLX load selector/readiness identity, or changes the canonical target/runtime; proxy/symlink hacks would violate the managed-Pi boundary.

### 2013 — Hosted Identity Locks Cross The Wrapper Delimiter
- FIX: `lockCodexTargetArguments` and `lockClaudeTargetArguments` now scan identity selectors through the first wrapper `--` and stop only at the second provider operand boundary; Pi keeps its first-`--` message boundary.
- FIX: Ordinary hosted alias launches no longer dump the resolved launch plan to stderr; diagnostics remain explicit `--print-config` behavior.
- EVIDENCE: Production `runTarget` negatives reject eight post-delimiter model/reasoning/profile selector classes before provider execution or config mutation; narrowing both locks back to first-delimiter termination reddens every case with `-count=1`.
- SCOPE: `TASK-260824-2o4zq8` reviewer rework in `tools/agents-infra/internal/infra/canonical_target.go`, `tools/agents-infra/main.go`, and focused tests.

### 2002 — A Wrapper `--` Is Not An Operand Boundary
- ROOT CAUSE: `lockCodexTargetArguments`/`lockClaudeTargetArguments` in `tools/agents-infra/internal/infra/canonical_target.go` return the remaining argv raw on a literal `--`, treating it as an operand boundary; `parseCodexWrapperArgs` (`internal/infra/codex_launch.go:563`) and `parseClaudeWrapperArgs` (`internal/infra/claude_launch.go:414`) instead consume that `--` as a wrapper delimiter and keep parsing every following token as a provider flag.
- FINDING: The canonical identity lock is bypassable at the production `runTarget` entrypoint. With a recording fake provider and `err=nil`: `openai-infra -- exec -- --profile work` launches Codex with `--profile work`, which contract Section 3.6 says must always fail `target_identity_conflict`, while `--print-config` still reports `effective_profile_source: native`; `anthropic-infra -- -- --model other` launches Claude with `--model claude-opus-5 --effort high --model other`, and last-wins means `other` is what runs while the plan reports `claude-opus-5`. Same shape for Claude `--effort` and Codex `--model-reasoning-effort`.
- FINDING: Pi is unaffected — its `--` is a real message-operand boundary and the Pi composer refuses flag-like operands (`unsafe Pi message operand "--model"`).
- FINDING: No test passes `--` through any of the three lock functions, so that branch is entirely uncovered; the suite is green around a live bypass path.
- SCOPE: `TASK-260824-2o4zq8` review verdict, routed to `to-dev`.

### 1941 — Worktree Pi Fixtures Must Resolve The Common Checkout
- ROOT CAUSE: `officialPiAsset` looked only under the active Story worktree `.temp`, while the reviewed Pi asset lived under the primary checkout; required Qwen acceptance tests reported package success by skipping.
- FIX: `tools/agents-infra/main_test.go` and `internal/infra/pi_test.go` now derive a primary-checkout fallback from the worktree `.git` gitdir; production Qwen compose and all canonical Qwen tests report `PASS`, not `SKIP`.
- FINDING: Package-level `go test` success is insufficient evidence for external-asset acceptance tests unless their named verbose result is checked.
- SCOPE: `TASK-260824-2o4zq8` test infrastructure and Qwen target evidence.

### 1842 — Hosted Alias Profile Provenance Stays Provider-Specific
- DECISION: Hosted alias model/reasoning provenance comes from the target, but `resolved.profile` retains provider semantics: Claude is `not_applicable`; a Codex alias is `native` because explicit `--profile` and `-c profile=` are identity conflicts.
- DECISION: Codex `-c model=` and `-c model_reasoning_effort=` are identity selectors alongside their dedicated flags; exact repeats pass and divergences fail.
- FINDING: Contract evidence now creates clause-removal mutants and requires the actual validator to reject them; absent markers fail before mutation.
- SCOPE: `TASK-260824-3rl3ws` revision-3 rework for reviewer findings G1-G2.

### 1824 — Vendor Config Migration Is Operator-Only
- DECISION: Parse, compose, setup, verify, target dispatch, and launch never rewrite project config; invalid canonical targets fail before side effects with source/field identity and corrective action.
- DECISION: The recursive all-project rewrite is a separate task-scoped operator rollout, never installed product behavior or a startup fallback.
- SCOPE: Contract Sections 5-7 and `TASK-260824-1jjze0`, per owner rollout clarification.

### 1823 — Vendor Targets Separate Canonical And Pi Namespaces
- DECISION: Codex target reasoning remains a non-empty provider-owned string; Claude targets admit only efforts the composer can prove effective; Pi targets retain the owned thinking enum.
- DECISION: Pi profile `provider` is an operator namespace, not Qwen vendor evidence. Optional `profile_provider` asserts the existing label without forcing legacy profile mutation.
- FINDING: Pi `resolved.model` remains provider-qualified; target model stays unqualified, with explicit profile-provider/model and endpoint invariants in alias plans.
- SCOPE: `TASK-260824-3rl3ws` contract rework after reviewer findings F1-F5.

### 1815 — Managed Codex Sync Preserves User State
- ROOT CAUSE: `tools/agents-infra/internal/infra/infra.go:274` copied global Codex config wholesale and skipped the local copy wholesale; repeat setup therefore either erased user trust/TUI state or retained withdrawn managed defaults.
- FIX: `Setup -> syncRepo -> syncManagedCodexConfig` now refreshes source-owned defaults, merges installed `projects`/`notice` and non-fast custom profiles, removes `profiles.fast`, and refuses malformed installed TOML without replacement.
- EVIDENCE: Production-path global/local resync tests preserve trust, notice, and primary-session state while installing `service_tier = "default"`; malformed-existing-config coverage attacks the fallback boundary.

### 1753 — Fast-Profile Removal Meets Runtime Preservation Boundary
- FINDING: Source `.configs/codex-config.toml` and both installed managed copies retain `[profiles.fast]`; the source-repository local copy also retains the withdrawn `service_tier = "fast"` state.
- FINDING: Global installed config contains user-added trust entries and model selection absent from source; `setup global` would replace them from source, while `setup local` intentionally preserves existing `.agents/.configs/codex-config.toml` bytes.
- DECISION: Remove the source profile and its README advertisement with production `Setup` regression coverage; validate supported installs on isolated destinations and defer live runtime synchronization until accepted source integration can preserve user-owned state explicitly.

### 1751 — Vendor Targets Stay Additive And Identity-Locked
- DECISION: Canonical `[agents.targets]` plus `[agents.entrypoints]` is a strict alias-only launch path; declaring it does not alter direct `agents-infra codex|claude|pi` precedence.
- DECISION: Hosted target identity overrides matching legacy model policy; Qwen/Pi reuses an atomic `[agents.pi.profiles]` definition and validates target model/reasoning/endpoint as exact assertions over that profile.
- FINDING: This avoids both unsafe legacy fallback after target resolution fails and duplicated Pi runtime/endpoint policy while keeping existing project configs valid.
- SCOPE: `STORY-260824-1yr6m0`, contract outcome `TASK-260824-3rl3ws_vendor-target-contract.md`.

## 2026-08-18

### 0504 — 503 Retry Accepted; Configured-Timeout Magnitude Still Unbound
- FINDING: Cycle-2 review of `BUG-260818-jreo1p` reddens four mutants on `tools/agents-infra/internal/infra/pi_launch_posix.go:352` — `deadline.Reset(timeout)` on 503, deleting both in-loop liveness checks, widening `== 503` to `>= 500`, and deleting the 503 branch. A fifth survives: `time.NewTimer(timeout * 10)` keeps every readiness test green (the timeout subtest just runs 10.13s instead of 1.13s), so the suite binds "terminates on some deadline", not "on the configured `startup_timeout_seconds`".
- EVIDENCE: Installed `~/.local/bin/agents-infra pi` run under `env -i` against Python readiness fixtures with `startup_timeout_seconds = 3`: persistent 503 → `runtime readiness timed out` after 26 polls in 3.17s; 502 → `invalid readiness response: status=502` after 1 poll in 0.26s; runtime PID gone in both. Production behavior honors the configured value; only the regression binding is missing.
- FINDING: `~/.local/bin/agents-infra` is byte-identical to the current source only when rebuilt with the ldflags `scripts/setup.sh:176` stamps (`-X main.Version/Commit/BuildDate`). A plain `-trimpath` rebuild differs by 96 bytes; the symbol-table delta is exactly those three strings. Cycle one's plain-`-trimpath` provenance method no longer holds and must not be repeated as-is.
- NOTE: The controlled Qwen smoke does not traverse a 503 window — its llama.cpp bound port 18011 only after `model loaded` at 13.97s. 503 semantics are bound by the Go production-entry tests and the installed-binary run, not by the live smoke.
- STATUS: Accepted; reviewer supplied no `commit_ack`. Follow-up: assert elapsed time in the timeout subtest.

### 0448 — Controlled Qwen Cache Reproduces Text And Tool Smokes
- ROOT CAUSE: The three 0442 timeouts used ordinary HOME, while the reviewed 17,923,394,624-byte target and 931,146,432-byte mmproj live under `local-models/.temp/TASK-260817-300nun/live-smoke/home`.
- EVIDENCE: Under the original `env -i` task-HOME boundary, installed project `pi-infra` text and tool smokes both exit 0. JSON assertions prove exact `QWEN_TEXT_OK`; exactly one `bash` call with `printf QWEN_TOOL_VALUE=42`; exact successful result; and final `QWEN_TOOL_OK:42`. Port/process cleanup exit 0.
- STATUS: Live gate reproduced; `BUG-260818-jreo1p` rework ready for review. Raw logs attached as `BUG-260818-jreo1p_qwen-controlled-smokes.tar.gz`.

### 0444 — Qwen Re-Smoke Timeout Is Non-Blocking For Readiness Test Rework
- DECISION: Orchestrator directive `RUN-260818-fe202a:nudge:b5a4e1` identifies the three 120s timeouts as a cache-path mismatch: this re-smoke path does not expose the prior task-scoped cached weights. Preserve the failures and cleanup evidence; do not supersede accepted `local-models` `TASK-260817-300nun` live text/tool evidence or block the additive test rework.
- STATUS: No further live attempts. `BUG-260818-jreo1p` proceeds to review on isolated red mutants, green production tests/build/vet/setup/verify, and zero orphan/listener evidence.

### 0442 — Qwen Readiness No Longer Reproduces Prior Live Smoke
- ANOMALY: After `./setup.sh` plus local setup/verify, three installed `/Users/alexis/src/local-models/.local/bin/pi-infra` text smokes reached `runtime_readiness_timeout` at the configured 120s; the tool smoke was not run because exact-model readiness never succeeded.
- EVIDENCE: All three runs started the reviewed llama.cpp child and emitted its loopback server warnings. Each returned exit 1, released port 18011, and left no matching llama process; memory remained 77% free and `ollama ps` was empty.
- BLOCKED: Earlier `BUG-260818-jreo1p_results.md` reports passing Qwen text/tool smokes without an attached log. Current deployment evidence cannot reproduce that gate; runtime/model readiness must be restored or the operator must explicitly revise the 120s profile timeout before new live acceptance evidence can be produced.

### 0442 — Readiness Termination Mutants Isolated After Concurrency Nudge
- FIX: `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry` now drives production `RunPi` through persistent-503 timeout and child-exit-during-503 cases, asserting at least one exact-503 poll, owned PID reap, and profile-lock release.
- EVIDENCE: In `.temp/BUG-260818-jreo1p/mutant-repo`, resetting the deadline on every 503 times out the named Go test (exit 1); deleting both in-loop child-liveness checks returns `runtime_readiness_timeout` instead of `runtime_exited_early` (exit 1). Restored source matches baseline and shared source byte-for-byte; isolated green test and orphan checks exit 0.
- ANOMALY: The orchestrator's isolation nudge arrived after two equivalent mutants had already run briefly in the shared dirty checkout. They were restored byte-for-byte and later repeated in isolation; subsequent bootstrap reinstalled the restored source. Shared production-file mutants remain forbidden while concurrent local-model runs exist.

### 0418 — 503 Retry Loop Shipped With Both Termination Bounds Untested
- FINDING: `BUG-260818-jreo1p` binds the two branches its AC names — widening `== http.StatusServiceUnavailable` to `>= 500` reddens the 502 subtest, deleting the branch reddens the 503 subtest — but the *other* two clauses of the same AC sentence ("until timeout", "while the owned child remains alive") have zero coverage. Injecting `deadline.Reset(timeout)` into the 503 branch of `tools/agents-infra/internal/infra/pi_launch_posix.go:385` makes the retry infinite and `go test ./internal/infra` still returns `ok 70.703s`. Deleting both in-loop liveness guards (`case <-childWait.done` and `child.Signal(0)`) leaves it green at `ok 71.280s`.
- ROOT CAUSE OF THE GAP: no fixture in the package ever sustains 503 — `writePiSequencedReadinessServer` flips to 200 on request 3, `writePiReadinessServer` only answers 200. `TestPiLaunchRefusesForeignReadyListenerForDeadChild` serves 200, so it exercises the post-readiness check at `pi_launch_posix.go:176`, never the in-loop one.
- DECISION: A fix that converts a fatal branch into a retry loop makes the loop's *termination* bounds newly load-bearing. Bind them in the same change, not just the branch that was edited. Code correct by inspection is not code bound by evidence.
- FINDING: `~/.local/bin/agents-infra` is byte-identical to a `-trimpath` rebuild of current source with the recorded ldflags (sha256 `6e74c363c2fcea21b56efe72f1f355738f83f32971b589ac7e2e5265ff4d837d`). Reproducing the installed artifact from source is a cheap, exact way to prove an installed runtime carries a fix — better than mtime comparison.
- ANOMALY: A concurrent reviewer (`TASK-260817-300nun`, `RUN-260818-8ca9ef`) reported `deadline.Reset(timeout)` at `pi_launch_posix.go:385` as production code. It never was — that line is the unbounded-retry mutant this review injected and removed. Two agents mutating one shared dirty checkout produced a false finding that would have sent a producer to fix a nonexistent line. Concurrent reviewers must isolate mutants in a private copy, or serialize.
- STATUS: `BUG-260818-jreo1p` routed `to-dev`; rework is additive test coverage only, no production-path change required. Verdict in `BUG-260818-jreo1p_reviewer-verdict.md`.

### 0401 — Managed Pi Waits Through llama.cpp Loading 503
- ROOT CAUSE: `waitPiRuntimeReady` treated every non-200 response as fatal; llama.cpp build 10470 binds the exact loopback endpoint and returns HTTP 503 while loading a cached 27B model, so production `RunPi` killed its owned runtime before readiness.
- FIX: `tools/agents-infra/internal/infra/pi_launch_posix.go` retries exact 503 while the child remains alive and the configured startup deadline remains active; every other non-200 and every read/parse/model mismatch still fails closed.
- EVIDENCE: `TestPiLaunchReadinessRetriesOnlyServiceUnavailableAtProductionEntry` drives `RunPi`: 503,503,200 spawns Pi only after request 3; 502 stops after request 1. Widening the retry to all 5xx makes the 502 branch fail. Focused/full tests, build, vet, bootstrap, local sync, and installed verification exit 0.
- STATUS: `BUG-260818-jreo1p` implementation ready for review handoff; discovered by `TASK-260817-300nun` live Qwen smoke.

### 0205 — Exact-Name Gates Need A Case Control, Not Just A Suffix Control
- FINDING: `BUG-260818-1s1lka`'s `LLAMA_API_KEY` gate survives a `strings.EqualFold` widening mutant across the *entire* `internal/infra` package. `TestPiExecutionEnvironmentAcceptsExactCleanEnvironment` carries a lowercase case-sensitivity control for every sibling exact name (`hf_endpoint`, `model_endpoint`, `ggml_backend_path`) but not for `llama_api_key`; `LLAMA_API_KEY_SUFFIX` only pins the prefix/family dimension. `SKILL.md:329`, shipped by that same change, claims lookalikes stay admitted — a doc claim with no test in the case dimension.
- DECISION: Accepted anyway. A case-insensitive regression over-refuses a name `getenv` never reads, so it is a usability regression rather than an auth bypass, and it falsifies no AC. One-line follow-up: add `"llama_api_key=case-sensitive-lookalike"` to that clean list.
- FINDING: The broadening-mutant rule from 0119 needs two axes, not one. For an exact-name gate the family/prefix axis and the case axis are independent bounds; `LLAMA_API_KEY_SUFFIX` reddens the first and says nothing about the second. Add both controls whenever a new exact name joins that list.
- ANOMALY: The `llama-b10470` tree used at 0119 to verify the `GGML_BACKEND_PATH` premise against the shipped binary is gone from this host (`find / -maxdepth 6 -name llama-server` empty). The `LLAMA_API_KEY` -> `--api-key` premise therefore rests on the task description and the prior cycle's artifacts, not on a re-read of the artifact. Reported as unknown rather than inferred; whether b10470 exposes another ambient-auth env name (e.g. one backing `--api-key-file`) is likewise unestablished and is separate work, since repo policy requires an established runtime effect before adding a name.
- STATUS: `BUG-260818-1s1lka` accepted by review (`BUG-260818-1s1lka_review-verdict.md`); routed `to-review` for the commit-owning mover, since a reviewer archetype must not supply `commit_ack`.

### 0129 — Managed Pi Refuses Ambient llama.cpp API Authentication
- ROOT CAUSE: llama.cpp build 10470 maps inherited `LLAMA_API_KEY` to `--api-key`, while managed Pi profiles declare neither that option nor matching generated credentials; ambient process state could silently change runtime authentication.
- FIX: `tools/agents-infra/internal/infra/pi_catalog.go:295` refuses exact `LLAMA_API_KEY` at `RunPi`'s shared environment boundary before managed state or runtime spawn and reports only the name; `HF_TOKEN`, cache variables, exact-name lookalikes, and unrelated names remain admitted.
- EVIDENCE: Narrowing the gate to empty values reddens the helper, production `RunPi`, bootstrap-global alias, and project-local wrapper tests for non-empty `LLAMA_API_KEY`; restored focused/full tests, vet, build, bootstrap, and global/local verification exit 0.
- STATUS: `BUG-260818-1s1lka` implementation ready for review handoff.

### 0119 — b10470 Ships Exactly One GGML Env Name, Which Bounds The Gate Both Ways
- FINDING: `libggml.0.20.1.dylib` in `~/.local/share/llama.cpp/llama-b10470` imports `_getenv`, `_dlopen`, and `_dlsym`, and `GGML_BACKEND_PATH` is the *only* uppercase `GGML_*` env-shaped literal anywhere in that tree. `GGML_METAL_PATH` does not exist in the build at all. The exact-name policy in `tools/agents-infra/internal/infra/pi_catalog.go:295` is therefore complete for this build, and refusing a speculative `GGML_*` prefix gate is grounded in the artifact, not in caution.
- DECISION: Verify a docs premise against the shipped binary, not against the task description that introduced it. `llama --version` plus `strings`/`nm` on the ggml dylib settles it in seconds; `README.md:604` and `SKILL.md:320` state build 10470 as fact and now have that backing.
- FINDING: A narrowing mutant alone under-tests an exact-name gate. Adding `"GGML_"` to the loader prefix list in the same function reddens the `GGML_METAL_PATH` clean control on the in-process lifecycle test *and* both installed launcher surfaces — so the suite pins the upper bound of the class too, not just its lower bound. Reviewers should run the broadening mutant next to the narrowing one whenever a gate is deliberately scoped narrower than its name family.
- FINDING: Mutating `RunPi`'s gate order (moving `ValidatePiExecutionEnvironment` after `CreatePiStateTree`) reddens `environment refusal created managed state` for *every* member of the refusal table, not only the new one. Ordering is genuinely asserted rather than incidentally satisfied.
- ANOMALY: `TestPiLaunchForwardsSignalsThenCleansRuntime` fails in any rsync copy of the tree that reaches the Pi asset through a symlink — `officialPiAsset` uses `filepath.Abs` while `identity.Entrypoint` resolves symlinks, so the spawn-path assertion mismatches. Copy artifact only; green in the real checkout. Budget for it when reviewing in a disposable copy.
- STATUS: `BUG-260818-76hkcb` accepted by review (`BUG-260818-76hkcb_review-verdict.md`); routed `to-review` for the commit-owning mover, since a reviewer archetype must not supply `commit_ack`.

### 0105 — Managed Pi Denies Exact GGML Backend Loader Path
- ROOT CAUSE: llama.cpp build 10470 passes inherited `GGML_BACKEND_PATH` to `dlopen()` during backend discovery, bypassing the intent of managed `DYLD_*`/`LD_*` loader-injection gates.
- FIX: `tools/agents-infra/internal/infra/pi_catalog.go:295` refuses exact `GGML_BACKEND_PATH` at `RunPi`'s shared environment boundary before state or runtime spawn and reports only the name; other `GGML_*` remain admitted absent an established runtime effect.
- EVIDENCE: Production and bootstrap-global/project-local launcher negatives fail under a `GGML_BACKEND_PATH_V2` narrowing mutant; restored controls reach runtime backend initialization with `GGML_METAL_PATH`; full tests, vet, bootstrap, and global/local verification exit 0.
- STATUS: `BUG-260818-76hkcb` implementation ready for review handoff.

### 0010 — `strings(1)` Is A False-Negative Proxy For Go Gate Literals
- FINDING: `strings ~/.local/bin/agents-infra | grep HF_ENDPOINT` returns zero hits on a binary that demonstrably refuses `HF_ENDPOINT`. Go compiles short literal string equality (`name == "HF_ENDPOINT"`, 11 bytes) into a length check plus immediate-constant comparisons, so the literal never lands in rodata. Literals reached through a slice or a format string (`LLAMA_ARG_`, `runtime-affecting …`) do appear, which makes the proxy look reliable right up to the point it lies.
- IMPACT: The installed-binary staleness check used in `BUG-260817-161m6u` (LOGBOOK 2310) would have reported this gate as absent from a correctly built binary. Do not use `strings` to establish whether an installed launcher carries a gate; run the gate with a positive clean control instead.
- EVIDENCE: `BUG-260817-2bh9nk` review — real `~/.local/bin/pi-infra` (target sha256 `9859d5…a058`) under `env -i`: clean control reaches runtime spawn and creates the runtime marker; `HF_ENDPOINT` and `MODEL_ENDPOINT` each exit 1 with `runtime-affecting environment name "<NAME>" is denied`, no value in output, no runtime child.
- STATUS: `BUG-260817-2bh9nk` accepted by review; awaiting the commit-owning mover for the final `done` transition.

## 2026-08-17

### 2349 — Managed Pi Pins Model-Origin Environment
- ROOT CAUSE: llama.cpp build 10470 honors `HF_ENDPOINT` and `MODEL_ENDPOINT` during `-hf` resolution, so inherited values could redirect reviewed model IDs to an unreviewed origin without changing managed argv.
- FIX: `tools/agents-infra/internal/infra/pi_catalog.go:295` refuses both exact names at the shared `RunPi` environment gate before state or runtime spawn; diagnostics expose names only. Tokens, cache variables, and case-sensitive lookalikes remain separate policy questions.
- EVIDENCE: Clean, denial, no-value-leak, production-entry, bootstrap-global, and project-local wrapper tests pass; HF-only and MODEL-only narrowing mutants each fail the opposite endpoint's unit and production `RunPi` cases.
- STATUS: `BUG-260817-2bh9nk` implementation and validation ready for review handoff.

### 2323 — Installed-Launcher Behavioral Proof Beats Binary String Parity
- FINDING: A managed-profile probe of the installed `~/.local/bin/pi-infra` with a clean control is a complete ordering proof: the control walks argv plan, environment gate, Pi identity, runtime identity and creates `Caches/agents-infra/pi/<state-key>/…` before spawning the runtime child, while `LLAMA_ARG_MODEL`/`LLAMA_ARG_CTX_SIZE` runs stop with only `Caches/` present. Without that control, "no state was created" is indistinguishable from failing early for an unrelated reason.
- CONSTRAINT: `DYLD_*` cannot be probed through the `pi-infra` shim — macOS SIP strips it when exec'ing `/usr/bin/env sh`, so that gate stays covered at source level only.
- EVIDENCE: `BUG-260817-161m6u` cycle-2 review — installed binary rebuilt by `./setup.sh` (SHA-256 `df62cd…f42c3` → `3cd24e…a0d`); denied probes name only the variable and leak no value; README delete/narrow/ordering mutants all redden `TestPiOperatorContractDocumentsCycle10Boundary`; `pi_catalog.go` delete and narrow-to-`LLAMA_ARG_MODEL` mutants redden validator and production-entry tests.
- STATUS: `BUG-260817-161m6u` accepted by review; awaiting the commit-owning mover for the final `done` transition.

### 2310 — `setup global` Never Refreshes The Bootstrap-Owned CLI Binary
- FINDING: `agents-infra setup global` syncs source into `~/.agents` and prints `Skipping local CLI wrapper install for global setup; bootstrap owns ~/.local/bin/agents-infra`, so the compiled binary that `~/.local/bin/pi-infra` execs keeps running pre-change code. Only the repo bootstrap `./setup.sh` (`scripts/setup.sh:182` `go build -trimpath`) rebuilds it.
- FINDING: `verify global` inspects the runtime tree, not the executable, so a green `setup global` + `verify global` pair is a proxy signal and does not establish that the installed launcher carries a new gate.
- EVIDENCE: `BUG-260817-161m6u` review — `strings ~/.local/bin/agents-infra` (mtime 21:11) has `DYLD_`, `pi_execution_environment_invalid`, `PI_CODING_AGENT_DIR` but zero `LLAMA_ARG_`/`runtime-affecting`; a fresh `go build -trimpath` of the same source has both.
- NOTE: The generated project-local wrapper is a shell script that rebuilds from `AGENTS_INFRA_SOURCE_DIR` on every invocation, so local installs do pick changes up — the asymmetry is global-only.
- STATUS: `BUG-260817-161m6u` routed to `to-dev`; source gate and its mutants are sound, installed global runtime refresh and a README contract fragment in `pi_operator_docs_test.go` are the outstanding rework.

### 2253 — Managed Pi Denies Inherited llama.cpp Argument Environment
- ROOT CAUSE: `RunPi` passed inherited `LLAMA_ARG_*` variables to llama.cpp; options absent from reviewed runtime argv could therefore change model or runtime parameters.
- FIX: `tools/agents-infra/internal/infra/pi_catalog.go:284` refuses the entire case-insensitive `LLAMA_ARG_*` namespace before managed state or runtime spawn and reports only the quoted variable name.
- EVIDENCE: Production `RunPi` negatives cover `LLAMA_ARG_MODEL` and `LLAMA_ARG_CTX_SIZE`; narrowing the gate to `LLAMA_ARG_M` admits the second variable and makes both validator and production-entry tests fail.
- STATUS: `BUG-260817-161m6u` source tests/build/vet and installed global/local setup verification pass; ready for review handoff.

### 2132 — Pi Endpoint Gate Survives Attack; llama Env Override Refuted
- FINDING: Real runtime `llama-b10470` exports `LLAMA_ARG_HOST`/`LLAMA_ARG_PORT` and `pi_launch_posix.go:142` passes the inherited environment through unfiltered, but driving the binary directly proves CLI wins — it logs `LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host`. Not a bypass of the argv endpoint gate.
- FINDING: `validatePiRuntimeEndpointArgv` runs at config-parse time (`pi_config.go:207`), so it still refuses divergence when Pi is absent from `PATH` and compose would otherwise report `managed:false`. No unmanaged-fallback bypass.
- FINDING: Narrowing mutants prove both bounds, not just gate presence — unbinding the port value reddens `runtime_port_drift`, unbinding the host value reddens `wildcard_runtime_bind`, and relaxing exactly-one to at-least-one reddens `duplicate_runtime_port`.
- ANOMALY: `mainTestOfficialPiAsset` calls `t.Skipf`, so both production-entry endpoint negatives silently SKIP and the package reports `ok` when the gitignored `.temp/TASK-260817-2h8hn4` Pi asset is absent. Story-wide fixture convention (helper is not in `HEAD`, backs 6 tests), not introduced by this bug.
- NOTE: `--reuse-port` is not refused in runtime argv; `preflightPiListener` binds without `SO_REUSEPORT`, so an occupied port is still caught, but a later same-port `SO_REUSEPORT` process could share the declared endpoint.
- STATUS: `BUG-260817-2lpkfh` reviewer verdict accepted; awaiting commit-owning mover for the `done` transition with `commit_ack`.

### 2109 — Managed Pi Endpoint Claim Bound To Runtime Argv
- ROOT CAUSE: `tools/agents-infra/internal/infra/pi_config.go` validated `base_url` independently while accepting arbitrary non-empty runtime argv, so compose could report loopback port `18011` while launching `--host 0.0.0.0` or `--port 19011`.
- FIX: Managed profiles require exactly one spaced `--host 127.0.0.1` and one spaced `--port <base_url-port>` pair before compose, diagnostics, or launch can succeed.
- EVIDENCE: Production `runCompose` and setup-generated `.local/bin/agents-infra compose` negatives reject wildcard and port-drift mutants; the exact loopback control preserves literal argv.
- STATUS: `BUG-260817-2lpkfh` implementation validation in progress.

### 2050 — Skill-Link Gate Proves Graph Acyclicity
- ROOT CAUSE: Per-link containment could not detect cycles formed through multiple individually contained directory links.
- FIX: `tools/agents-infra/internal/infra/skill_link_validation.go:88` now follows contained directory-link targets with DFS `visiting`/`done` state; re-entry is refused while shared completed targets remain a valid DAG.
- EVIDENCE: Source and installed production-entry transitive-cycle negatives fail under a per-link-only narrowing mutant and pass after restoration; focused/full/vet/build plus pristine/source/local-models setup, verify, and `find -L` gates exit 0.
- STATUS: `BUG-260817-3nk7yf` cycle-5 implementation ready for review handoff.

### 2038 — Per-Link Validation Misses Contained Transitive Skill Cycle
- REGRESSION: `tools/agents-infra/internal/infra/skill_link_validation.go:116` validates each symlink independently; two individually contained links can still form a traversal cycle through separate directories.
- EVIDENCE: Source-built production `setup local` and `verify local` both exited 0 for `.skills/transitive-cycle-probe -> ../cycle-target` plus `cycle-target/back -> ../.skills/transitive-cycle-probe`; installed `rsync -aL` reported `directory cycle` on both graph edges while the focused production suite passed.
- STATUS: `BUG-260817-3nk7yf` review routes to rework; enforce graph acyclicity across setup-owned skill traversal and add a production-entry transitive-cycle negative.

### 2026 — Recursive Skill-Link Containment Gate Closed
- ROOT CAUSE: `tools/agents-infra/internal/infra/skill_link_validation.go:53` inspected only top-level entries while `syncRepo` copied nested symlinks verbatim.
- FIX: `inspectSkillLinkTree` walks physical managed package directories without following links and validates every encountered link; global provider ownership remains filtered by top-level managed package name.
- EVIDENCE: Installed-binary nested escape/cycle negatives fail under a top-level-only narrowing mutant and pass after restoration; focused/full/vet/build plus pristine/source/local-models setup and verify gates pass.
- STATUS: `BUG-260817-3nk7yf` cycle-3 implementation ready for review handoff.

### 2010 — Nested Skill Symlinks Bypass Setup Containment Gate
- REGRESSION: `tools/agents-infra/internal/infra/skill_link_validation.go:53` delegates to a top-level-only scanner; symlinks nested under an ordinary `.skills/<name>/` directory are copied but never validated.
- EVIDENCE: Source-built production `setup local` and `verify local` both exited 0 for nested absolute escape and ancestor-cycle inputs; installed escape resolved outside `.agents`, and installed cycle resolved to the `.agents` ancestor.
- STATUS: `BUG-260817-3nk7yf` review routes to rework; recursively validate every copied source/runtime symlink while preserving the explicitly scoped global user-managed boundary.

### 2003 — Setup Skill-Link Gate Owns Top-Level Managed Surfaces
- ROOT CAUSE: `syncRepo` copied top-level source `.skills` symlinks verbatim, while setup and `verify local` had no containment or ancestor-cycle postcondition.
- FIX: `tools/agents-infra/internal/infra/skill_link_validation.go` rejects unsafe source links before destination mutation and verifies top-level links across `.agents/.skills`, `.agents/skills`, `.claude/skills`, and `.codex/skills`; global provider checks are limited to setup-managed names.
- DECISION: Do not recursively claim ownership of symlinks inside external skill packages or unrelated global user-managed links; production `./setup.sh` exposed that false ownership boundary.
- EVIDENCE: Installed-binary escape/cycle negatives and four-surface drift attacks pass; source install, pristine/source/local-models setup+verify, and recursive `find -L` pass.

### 1930 — Setup Skill Containment Gate Has Source-Symlink Bypass
- REGRESSION: `tools/agents-infra/internal/infra/infra.go:307` preserves source `.skills` symlinks without containment checks; production `setup local` materializes escaping and ancestor/self-cycle links under `.agents/.skills`.
- EVIDENCE: Installed-binary setup and subsequent `verify local` both exited 0 for `.skills/escape-probe -> <outside-runtime>` and `.skills/cycle-probe -> ..`; `find -L` reached the external marker through the installed runtime.
- STATUS: `BUG-260817-3nk7yf` review routes to rework; reject or safely omit non-contained/cyclic skill links through production setup and verify, with escaping and ancestor-cycle negative coverage.

### 1921 — Setup Runtime Materialization Cycles Removed
- ROOT CAUSE: `syncRepo` admitted legacy literal `$AGENTS_INFRA_SOURCE_DIR` and nested `.temp` trees from installed sources; `ensureRepoSkillLinks` linked `skills/relux-agents-infra` to its own `.agents` ancestor.
- FIX: `tools/agents-infra/internal/infra/infra.go` excludes and scrubs those generated artifacts, materializes `SKILL.md` plus `README.md` under `.skills/relux-agents-infra`, and links only to that contained package.
- EVIDENCE: Production setup regression fails under literal-copy and ancestor-link narrowing mutants; focused/full tests, vet/build, global/source/local-models setup+verify, and recursive `find -L` inspection pass.
- STATUS: `BUG-260817-3nk7yf` implementation ready for review handoff.

### 1853 — Pi Alias Type And Mode Drift Gate Closed
- ROOT CAUSE: Alias setup compared bytes before pathname type/mode, and verification used `os.Stat`; byte-identical symlinks inherited their target's regular-file identity while `0644` aliases skipped repair.
- FIX: `tools/agents-infra/internal/infra/infra.go` requires regular-file type, exact body, and `0755` before the alias up-to-date branch; `runtime_receipt.go` uses `os.Lstat` for both alias and sibling target before reading or launching either path.
- EVIDENCE: Production `setup local` repairs mode and byte-identical symlink alias drift; production `verify local` refuses symlink alias and sibling target. Focused production and verifier suites exit 0.
- STATUS: `TASK-260817-3a0zr3` reviewer cycle-1 findings resolved pending full gates and review handoff.

### 1848 — Pi Alias Symlink Drift Bypasses Verify
- REGRESSION: `tools/agents-infra/internal/infra/runtime_receipt.go:190` uses `os.Stat` for the managed `pi-infra` launcher, so replacing the installed regular file with a symlink to byte-identical external content passes `verify local`.
- EVIDENCE: Production `setup local` created the alias; after symlink substitution, production `verify local` exited 0. Mode-only drift is detected, but `installPiInfraLauncher` treats matching bytes at `0644` as up to date and setup fails instead of repairing the drift promised by README/SKILL.
- STATUS: `TASK-260817-3a0zr3` reviewer routes to rework; reject symlink/non-regular alias identity at setup and verify, repair all documented managed alias drift, and retain production-entry negative/narrowing coverage.

### 1829 — Setup Now Owns Pi Alias And Catalog Integrity
- FINDING: The authoritative 217-record Pi manifest was a launcher `go:embed` input but was not part of setup source/runtime receipt validation, so setup and verify could accept manifest drift before a managed launch exercised the catalog.
- FIX: `tools/agents-infra/internal/infra/source_dir.go` and `runtime_receipt.go` require the exact manifest SHA-256; both setup modes install an exact sibling-only `pi-infra`, and verification refuses missing/drifted alias bytes, mode, target, or manifest.
- EVIDENCE: Production installed-binary global setup preserves alias caller cwd and post-separator argv; production setup/verify negatives refuse alias and catalog drift.
- STATUS: `TASK-260817-3a0zr3` implementation ready for full validation and review handoff.

### 1807 — Pi Signal Gate Uses Deterministic Child Handshake
- ROOT CAUSE: The cycle-5 regression used an external Python Pi fixture and a fixed startup marker; under focused race-suite load its interpreter scheduling occasionally exhausted the unrelated `2s` shutdown window and produced `signal: killed`.
- FIX: `tools/agents-infra/internal/infra/pi_test.go` now drives production `RunPi` with the race-instrumented Go test binary and writes readiness only after `signal.Notify` is active; non-timeout Python readiness fixtures use `10s` startup budgets.
- EVIDENCE: Signal lifecycle passes 20/20 under `-race`; the exact focused Pi race suite passes 3/3 with SIGINT/SIGTERM forwarding, graceful exit, runtime-group cleanup, and lock release intact.

### 1758 — Pi Signal Cleanup Race Gate Is Flaky Under Suite Load
- REGRESSION: Required `go test -race ./internal/infra -run 'Test.*Pi' -count=1` failed in `TestPiLaunchForwardsSignalsThenCleansRuntime/terminated`; `RunPi` returned `signal: killed` after the Pi child missed graceful termination before timeout.
- EVIDENCE: First focused-suite run failed; the isolated signal test then passed 10/10 and a full focused rerun passed, so observed reproduction is 1/12 rather than stable green.
- STATUS: `TASK-260817-ccpnlm` reviewer cycle 5 routes to rework; make the lifecycle test/implementation deterministic under suite load and retain the closed managed `--export` isolation gate.

### 1748 — Managed Pi Export Bypass Closed
- ROOT CAUSE: Pinned Pi handles `--export` before isolated agent/session initialization and reads its source path directly, so forwarding the flag bypassed the existing session-location guards.
- FIX: Managed argument composition now refuses `--export` before state, lock, socket, or process side effects; ordinary Pi native passthrough remains unchanged.
- EVIDENCE: Production `runPi` drives a global-session sentinel through the refusal, verifies `invalid_provider_arguments`, unchanged bytes, no runtime launch, and an empty managed cache root; the focused unit and production tests pass under `go test -race`.

### 1739 — Managed Pi Export Escapes Session Isolation
- REGRESSION: `tools/agents-infra/internal/infra/pi_args.go:57` forwards `--export` without the managed session-path guard, so a caller can read an arbitrary session beneath `~/.pi/agent` despite hash-contained session state.
- EVIDENCE: Source-built production `agents-infra pi --print-config --export <global-session>` exits 0 and emits the global path in both Pi argv surfaces; pinned Pi calls `exportFromFile(parsed.export, ...)` before normal session creation.
- STATUS: `TASK-260817-ccpnlm` reviewer cycle 4 routes to rework; reject managed export or prove its source is anchored inside the generated session root, with a production-entry negative and a safe narrowing control.

### 1731 — Managed Pi Session Path Bypass Closed
- ROOT CAUSE: Pinned Pi treats `--session-dir` as an environment override and classifies `--session`/`--fork` values containing `/`, `\`, or ending in `.jsonl` as direct filesystem paths, bypassing hash-contained session state.
- FIX: `tools/agents-infra/internal/infra/pi_args.go` rejects the directory override and path-shaped selectors for managed profiles while retaining ID lookup, `--continue`, and `--resume` inside `PI_CODING_AGENT_SESSION_DIR`.
- EVIDENCE: Production `runPi` negatives preserve global-session bytes and prove zero runtime/state side effects across absolute, `.jsonl`, and backslash narrowing shapes; focused race tests pass.

### 1723 — Managed Pi Session Isolation Is CLI-Bypassable
- REGRESSION: `tools/agents-infra/internal/infra/pi_args.go:54` forwards managed `--session-dir`; pinned Pi gives that flag precedence over `PI_CODING_AGENT_SESSION_DIR`, so callers can redirect session reads/writes into the normal `~/.pi/agent` tree.
- EVIDENCE: Production `agents-infra pi --print-config --session-dir <HOME>/.pi/agent/sessions` exits 0 and emits the global path in both final Pi argv surfaces despite the hash-contained state path.
- STATUS: `TASK-260817-ccpnlm` reviewer cycle 3 routes to rework; managed launch must reject or safely neutralize state-location overrides and retain a production-entry negative isolation test.

### 1710 — Pi Session Output Fan-In Serialized Through Reap
- ROOT CAUSE: Serializing only runtime stdout/stderr left Pi stderr concurrent on the same caller writer; process-group disappearance also allowed `RunPi` to return before `Cmd.Wait` drained runtime pipes.
- FIX: `tools/agents-infra/internal/infra/pi_launch_posix.go` shares one mutex across every live Pi/runtime output stream and retains multi-consumer child completion until cleanup observes `Cmd.Wait`.
- EVIDENCE: Reviewer reproduction and `TestPiLaunchSerializesRuntimeOutputFanIn` both pass under `go test -race`; the dual-stream fixture would race if either cross-process serialization or post-reap drain waiting were narrowed away.

### 1702 — Pi Runtime Output Fan-In Races
- REGRESSION: `RunPi` assigns runtime stdout and stderr to one arbitrary `io.Writer`; `os/exec` writes through two goroutines and races on `bytes.Buffer`.
- EVIDENCE: `go test -race ./internal/infra -run '^TestPiLaunchOwnedRuntimeLifecycleAndGlobalStatePreservation$' -count=1` exits 1 with concurrent `bytes.Buffer.ReadFrom` writes from `tools/agents-infra/internal/infra/pi_launch_posix.go:142-143`.
- STATUS: `TASK-260817-ccpnlm` reviewer cycle 2 routes to rework; serialize production output fan-in and retain a dual-stream race regression.

### 1645 — Pi Rework Drives Security Gates Through Production
- FIX: `tools/agents-infra/internal/infra/pi_args.go` rejects every bare unknown long option and empty managed credential value; only self-contained `--name=value` and complete pre-delimiter flag/value pairs survive.
- FIX: Exact readiness refuses redirects; isolated state uses random atomic catalog temporaries, single-link regular lock files, and post-open directory revalidation.
- EVIDENCE: Named profile-state and catalog narrowing cases now enter through production compose; listener, readiness, spawn, point-of-use mutation, signal, shutdown, literal-argv, environment, and lock-release attacks enter through `RunPi`.
- STATUS: Reviewer cycle-1 bypass reproduced red before rework and now refuses in a source-built binary while both permitted unknown-option forms compose without side effects.

### 1615 — Pi Bare Unknown Argv Gate Bypassed
- REGRESSION: Managed production diagnostics accept a bare unknown long option and emit it in final Pi argv, although the cycle-10 contract permits only `--name=value` or a complete flag/value pair.
- ROOT CAUSE: `tools/agents-infra/internal/infra/pi_args.go:226` refuses the bare form only when suffix operands exist; the no-suffix path forwards it.
- EVIDENCE: `agents-infra pi --print-config --unknown` exited 0 with `status:"ok"`; existing coverage tests only `--unknown -- prompt`, leaving a bypass path.
- STATUS: `TASK-260817-ccpnlm` reviewer cycle routes to rework; add real-entry refusal and narrowing evidence for the exact permitted unknown-option forms.

### 1557 — Pi Cleanup No Longer Depends on Single-Consumer Exit Evidence
- ROOT CAUSE: Readiness could consume the runtime child's buffered `Wait` result; cleanup then waited on the already-empty channel and masked `runtime_exited_early` as a shutdown timeout.
- FIX: `tools/agents-infra/internal/infra/pi_launch_posix.go` now terminates and verifies the owned process group by PGID existence, using the child-result channel only as optional reap evidence.
- EVIDENCE: Positive real-entry lifecycle proves runtime-group reap and global Pi-state preservation; readiness negatives keep dead-child refusal independent of a ready endpoint.

### 1530 — Pi Profile State Uses Hash-Only Contained Paths
- ROOT CAUSE: Exact TOML profile identity was interpolated as a raw path component, so traversal escaped the cache root and normalized aliases shared state and locks.
- DECISION: Derive `profile_state_key` as lowercase SHA-256 of exact decoded-name UTF-8 bytes; never normalize, case-fold, clean, or sanitize before hashing.
- FIX: `TASK-260817-2h8hn4` cycle 10 requires anchored no-follow cache containment, collision and partial-read refusal before side effects, independent locks for byte-distinct names, and a production-entry raw/lossy-key narrowing test.
- SCOPE: `.research/260817_pi-local-model-launch-contract.md`; downstream `TASK-260817-ccpnlm` and `TASK-260817-3a0zr3` AC and preconditions.

### 1523 — Pi Profile Name Escapes Managed State Root
- FINDING: `TASK-260817-2h8hn4` cycle-9 schema accepts every non-empty profile key, while lifecycle derives `.../<sha256(canonical-project)>/<profile>/` with the raw profile name.
- EVIDENCE: Quoted TOML profile `../../../../../../.pi/agent` parses and normalizes outside the project/profile cache root; distinct names `qwen` and `nested/../qwen` normalize to the same state and lock path.
- STATUS: Review cycle 9 requires analysis rework: constrain profile names or derive the filesystem component from an injective safe encoding/hash, prove containment and collision resistance at the production entry, and add traversal/separator/normalization negatives before any lock or file write.

### 1515 — Pi Release-Tree Catalog Made Reproducible
- FIX: `TASK-260817-2h8hn4` cycle 9 restores the exact 217-record manifest, bytewise path ordering, record encoding, exhaustive prefix-closure inventory, entry-type/link policy, and exact permission map for Pi v0.84.2 darwin-arm64.
- EVIDENCE: Official asset SHA-256 `c996e888...` independently regenerates manifest SHA-256 `2f68ab1b...`; extracted tree has 217 regular files, 34 directories including root, no symlinks/other types, 4 files at `0755`, 213 files at `0644`.
- DECISION: Managed Pi identity compares the indivisible compiled catalog before side effects and again immediately before Pi spawn; opaque digest-only or regular-files-only approximations are insufficient.

### 1508 — Pi Catalog Digest Lost Its Canonicalization Contract
- REGRESSION: Cycle 8 retains the 217-file count and release-tree digest but drops the bytewise sort, record encoding, path/type rules, and complete catalog payload that made the managed Pi execution-closure check reproducible in cycle 5.
- ROOT CAUSE: Trust-boundary simplification removed unrelated Pi catalog-generation details while promising that the exact managed Pi identity gate remained unchanged.
- STATUS: `TASK-260817-2h8hn4` review cycle 8 requires analysis rework; restore a task-scoped deterministic manifest algorithm plus authoritative catalog content/digest evidence and a narrowing case that changes canonicalization while leaving the asset/entrypoint hashes intact.

### 1500 — Pi Runtime Boundary Reduced to Reproducible Claims
- DECISION: Reviewed project TOML `runtime.executable` plus literal argv is trusted executable policy; agents-infra reproduces absolute no-shell spawn, loopback preflight/readiness, direct-child liveness, process-group cleanup, isolated Pi state, and no intentional attach/fallback.
- DECISION: Qwen text/tools and Muse DFlash are requested/configured, never independently verified. Muse acceptance adds exact target/draft argv, exact target readiness, and operator Pi smoke/benchmark evidence.
- SCOPE: `TASK-260817-2h8hn4` cycle 8 rejects the cycle-7 backend catalog, compiled observer, internal proxy, and attestation API while retaining exact managed Pi identity and argv-parser gates.

### 1449 — Runtime Authority Moved Outside Project Backend
- DECISION: Managed Pi uses the verified agents-infra executable as the only direct adapter child and public-listener owner; runtime attestation v2 travels over an inherited private control pipe that backend descendants never receive.
- DECISION: Project TOML may select only immutable backend-catalog entries. Each entry fixes backend execution closure, argv grammar, private transport, compiled observer, and provable capability contracts; projects cannot supply adapter/observer code, digests, endpoints, or authority.
- FINDING: Current DFlash documentation exposes launch configuration but no independent initialized-engine active-state endpoint. Muse launch remains fail-closed unless a reviewed backend-specific observer is compiled into the catalog; diagnostics report unsupported with capability unknown.
- EVIDENCE: `TASK-260817-2h8hn4` cycle 7 adds a production-entry self-minted proxy attack and a narrowing mutant that replaces authoritative observation with config/environment echo.

### 1437 — Runtime Attestation Authority Is Self-Minted
- FINDING: `agents-infra.runtime-attestation.v1` validates a nonce, direct-child PID, model, and capability fields, but the arbitrary project-selected runtime or adapter that receives those expected values is also allowed to mint the JSON claim.
- ROOT CAUSE: The contract defines exact response shape without an independent authority anchor for socket/process ownership or active DFlash state; a direct-child proxy can own the public listener, echo the expected attestation, and forward inference to a foreign or target-only backend.
- STATUS: `TASK-260817-2h8hn4` review cycle 6 requires analysis rework with a trusted attestation authority/ownership mechanism and a production-entry self-minted proxy negative case.

### 1433 — Managed Runtime Ownership Uses One Child-Bound Attestation
- DECISION: Every managed Pi runtime must return `agents-infra.runtime-attestation.v1` bound to a fresh 256-bit launcher nonce, exact direct-child PID, exact model, and exact byte-sorted capability set; Qwen uses `dflash: null`, Muse adds the exact active DFlash target/draft object.
- ROOT CAUSE: Child liveness and `/v1/models` allowed a foreign listener to satisfy Qwen readiness while the selected non-binding child stayed alive; a preflight port check cannot close the check-to-bind race.
- STATUS: `TASK-260817-2h8hn4` cycle 6 decision and downstream implementation/operator AC now require absent, unreadable, malformed, stale, replayed, wrong-nonce/PID/model/capability, and foreign-listener refusals through the real launcher.

### 1426 — Pi Qwen Readiness Is Not Child-Bound
- FINDING: The managed Qwen contract accepts `/v1/models` from the configured loopback origin without a nonce, PID, or other proof that the selected runtime child owns the listener.
- ROOT CAUSE: Child-bound nonce/PID attestation exists only for the Muse DFlash profile; process liveness plus model readiness cannot distinguish the new child from a pre-existing listener.
- STATUS: `TASK-260817-2h8hn4` review cycle 5 requires analysis rework; bind readiness to the owned child and add a production-entry foreign-listener bypass plus narrowing evidence.

### 1422 — Managed Pi Uses Verified Standalone Execution Closure
- ROOT CAUSE: Byte-exact npm Pi verification left the `#!/usr/bin/env node` host, installed dependencies, `PATH`, and loader environment outside the gate; `NODE_OPTIONS=--require` could rewrite managed argv before Pi parsed it.
- DECISION: Managed profiles admit only a compiled-catalog official standalone release tree, initially Pi `v0.84.2` darwin-arm64; npm/shebang Pi remains native-passthrough-only.
- DECISION: Direct launch rejects loader-affecting environment names before side effects, invokes the verified canonical standalone path, and repeats full tree/environment-name verification immediately before Pi spawn.
- SCOPE: `.research/260817_pi-local-model-launch-contract.md`, `TASK-260817-2h8hn4` review cycle 5; downstream `TASK-260817-ccpnlm` and `TASK-260817-3a0zr3` AC updated.

### 1406 — Pi Managed Launch Bound to Immutable Package Identity
- DECISION: Managed `agents-infra pi` policy selects a read-only compatibility catalog entry compiled into the agents-infra release; project TOML cannot mint digests or extend the catalog.
- FINDING: Published `@earendil-works/pi-coding-agent@0.84.2` is bound by npm SRI, tarball SHA-256, exact `dist/cli.js` and `dist/cli/args.js` hashes, and a 972-file canonical package manifest digest; semver alone is non-authoritative.
- DECISION: Direct launch statically verifies the canonical package tree before lock/file/runtime/Pi side effects. Absent, malformed, unsupported, mismatch, and unknown identities remain distinct fail-closed states; non-launching diagnostics never execute Pi/npm/Node/Bun to discover identity.
- SCOPE: `.research/260817_pi-local-model-launch-contract.md`, `TASK-260817-2h8hn4` review cycle 4.

### 1358 — Pi Argv Contract Lacks Runtime Identity Gate
- FINDING: `TASK-260817-2h8hn4` proves argv normalization against Pi commit `a1bc0ec79010887210cc7de28714d72c78577dab` (`@earendil-works/pi-coding-agent` 0.84.2), but the proposed launcher resolves an arbitrary `pi` executable and never establishes that its parser grammar matches the pinned snapshot.
- ROOT CAUSE: Parser-dependent option arities, separator handling, equal forms, and extension-flag consumption are specified without a supported Pi version/build identity, compatibility probe, or deterministic parser catalog selected from verified identity.
- STATUS: Review cycle 3 requires analysis rework; bind managed launch to exact verified Pi identity or define a deterministic generated compatibility catalog with mismatch/unknown refusal and production-entry negative evidence before runtime start.

### 1351 — Pi Separator Is Not End-of-Options
- ROOT CAUSE: Pi `parseArgs()` at revision `a1bc0ec79010887210cc7de28714d72c78577dab` treats literal `--` as an unknown flag and continues parsing later model, credential, thinking, and trust options.
- DECISION: Managed `agents-infra pi` strips its wrapper-only delimiter, forwards only exact safe operands, rejects suffix tokens beginning with `-` or `@`, prevents value consumption across the removed boundary, and normalizes recognized `--flag=value` wrapper forms to Pi's spaced argv.
- STATUS: `TASK-260817-2h8hn4` contract and production-entry negative/mutant scenarios revised; no runtime may start on unsafe or ambiguous operand boundaries.

### 1337 — Pi Managed Identity and DFlash Gates Defined
- DECISION: Managed Pi profiles generate one provider/model catalog identity; CLI `--provider` and `--model` are accepted only when they resolve byte-for-byte to that identity. Different endpoint-exposed IDs, patterns, mismatches, and separator lookalikes refuse before runtime start.
- DECISION: DFlash launch requires a runtime-owned JSON attestation bound to a fresh 256-bit launcher nonce, direct child PID, fresh timestamp, and exact target/draft identities; absent, unreadable, malformed, false, stale, or mismatched evidence terminates runtime before Pi starts.
- STATUS: `TASK-260817-2h8hn4` contract and production-entry negative scenarios revised; `unknown` remains non-launching diagnostics only.

### 1332 — Pi Local Profile Override and DFlash Attestation Gaps
- FINDING: The draft Pi contract generates one provider/model entry but permits CLI provider/model overrides without defining how an overridden selection enters the generated catalog or remains bound to the selected runtime profile.
- FINDING: The DFlash profile requires fail-closed capability proof but defines no authoritative attestation source; its negative scenario permits `unknown`, allowing absent evidence to reach the permissive branch.
- STATUS: `TASK-260817-2h8hn4` requires analysis rework before implementation; make override materialization/refusal exact and define a launch-time DFlash attestation gate with negative production-entry evidence.

### 1325 — Pi Local Profiles Require Isolated Agent State
- FINDING: Pi custom providers/models are global-agent-dir data (`models.json`), while project settings use `.pi/settings.json`; project settings alone cannot register local models.
- DECISION: `agents-infra pi` selected local profiles use `PI_CODING_AGENT_DIR`/`PI_CODING_AGENT_SESSION_DIR` under agents-infra cache, an atomic generated local-only catalog, a loopback managed runtime child, and no reads/writes under `~/.pi/agent`.
- ANOMALY: Public DFlash documentation checked 2026-08-17 lists Qwen 3.5/3.6 targets but not the task labels `Qwen 3.8 27B` or `Muse Glimmer 30B`; profile artifact/runtime strings remain exact operator inputs and must not be guessed.
- SCOPE: `.research/260817_pi-local-model-launch-contract.md`, `TASK-260817-2h8hn4`.

## 2026-08-10

### 1829 — Local Verify Does Not Prove Source Freshness
- FINDING: `agents-infra verify local /Users/alexis/src/voice` passed while its installed `INSTRUCTIONS_TESTING.md` lacked the two newest Android spawn/device-state clauses present in the source and global runtime.
- ROOT CAUSE: Local verification proves the installed runtime is internally usable; it does not compare that runtime with the current source tree.
- STATUS: `TASK-260810-1drz2j` resynced the project runtime, proved direct source-to-installed/rendered parity, and passed independent review.

## 2026-08-04

### 1714 — Primary Launch Overrode Explicit Codex Config Mode
- ROOT CAUSE: `PreparePrimarySession` called `setupCodex(..., CodexConfigModeLocal, ...)` before every direct or session-manager Codex launch, recreating `.codex/config.toml` after an explicit `setup local --codex-config=global` and shadowing the user-level config.
- FIX: primary preparation now refreshes only Codex instructions, skills, and rules. It preserves an absent, managed, custom, or linked project config exactly and reports the config artifact as `absent` or `preserved`.
- EVIDENCE: expected-red tests reproduced creation, managed-byte replacement, and custom-config replacement; focused tests cover absent, managed-file, custom-file, symlink, and ancestor-runtime states. `go test ./... -count=1`, `go vet ./...`, `go build ./...`, formatting, and diff checks pass.
- LIVE VERIFICATION: installed the source build globally, refreshed `/Users/alexis/src/voice` with `--codex-config=global`, then launched `codexD` and `claudeD`. Codex used `gpt-5.6-sol`/`xhigh`/YOLO, Claude used `claude-fable-5`/YOLO, both bounded prompts completed without launch warnings, and `.codex/config.toml` remained absent after both launches.
- STATUS: Source fix and verification complete; committed as `b10d3b7` after owner authorization.

## 2026-07-25

### 0210 — Profile Name Domain and Remote Auth Env Metadata (0155 Pair Resolved)
- FIX: `recordProfileFlag` validates every `--profile`/`-p` spelling (spaced, `=`, attached) against the Codex plain profile-name syntax `^[A-Za-z0-9_-]+$` before recording it, failing closed via `ProviderArgumentError` → `invalid_provider_arguments`. Probes on codex-cli 0.145.0: empty, `foo/bar`, `a b`, `.`, `a.b`, `a=b`, `a+b`, `a@b`, `a~b`, `a,b`, `a:b`, backslash, and non-ASCII all exit 2 with "invalid --profile value ...; pass a plain name such as `work`"; `a_b`, `a-b`, `ab1`, `A_B`, `123`, `-ab`, `_ab`, `ab-` exit 0. Error precedence mirrors the provider exactly (probed): missing value reports before multiplicity, multiplicity reports before the second occurrence's invalid value, first-occurrence invalid value reports as invalid value. Profile existence stays provider-native config resolution.
- FIX: The shared Codex parser recognizes `--remote-auth-token-env` (spaced and `=` forms; root-command flag, probe-verified). A valid non-empty name surfaces in `required_env_names` after MCP bearer-token names, de-duplicated against them; the environment value is never read or serialized. Codex's accepted empty name (`--remote-auth-token-env=`, probe exit 0) contributes no requirement; a repeated occurrence ("cannot be used multiple times", probe exit 2) and a missing value fail closed identically in compose and the launchers. Tokens after the provider `--` stay uninterpreted; the argv tokens remain preserved in `managed_client.argv`.
- EVIDENCE: Probes in `.temp/TASK-260724-35d94i/rework7/`. All six reviewer repros re-run on the fixed binary: the five profile shapes exit 1 with the safe schema-v1 `invalid_provider_arguments` envelope; the remote-auth invocation emits `required_env_names:["CODEX_REMOTE_TOKEN"]` with the client argv intact. New infra coverage: profile-domain table (17 cases incl. launcher parity), remote-auth-env suite (8 cases incl. MCP dedup, ordering, post-`--`, secret-leak guard), 6 normalize-table cases; CLI: 7 fail-closed envelope cases plus a 2-case positive suite with MCP-plus-remote dedup. Full tests/vet/gofmt/diff-check green; child schema-v1 byte-identical to a HEAD-built binary for both providers on the MCP fixture; `go list -deps` has no task-board dependency.

### 0155 — Codex Profile Domain Still Diverges from Provider
- REGRESSION: Primary-session compose returns `status:"ok"` and serializes invalid Codex profile names as effective in `resolved.profile` and `managed_host.argv`.
- EVIDENCE: Codex CLI 0.145.0 rejects `--profile=`, `-p=`, `--profile=foo/bar`, `--profile="a b"`, and `-p.` with exit 2; the current compose binary accepts all five with exit 0.
- ROOT CAUSE: `tools/agents-infra/internal/infra/codex_launch.go:455` validates profile multiplicity but not the provider's plain-name value domain before `primary_session_launch_plan.go:253` serializes the selection.
- STATUS: `TASK-260724-35d94i` requires rework; validate every supported profile spelling in the shared builder and add contract/CLI regression coverage.

### 0155 — Required Env Metadata Omits Remote Auth Reference
- REGRESSION: A valid `--remote-auth-token-env CODEX_REMOTE_TOKEN` is preserved in `managed_client.argv`, but the primary-session contract emits `required_env_names:[]`.
- ROOT CAUSE: `tools/agents-infra/internal/infra/primary_session_launch_plan.go:472` collects only composed MCP bearer-token references and does not collect provider user-argument environment references.
- STATUS: `TASK-260724-35d94i` requires rework so valid managed-client environment references appear by name, never value, in `required_env_names`.

### 0252 — Policy Value Domains Validated Against Providers (0126 Resolved)
- FIX: The shared launch-plan parsers validate native policy values against probe-verified provider domains before serializing them as effective. Codex typed flags (`--sandbox`/`-s`, `--ask-for-approval`/`-a` in spaced/`=`/attached forms) validate against the clap enums (`read-only|workspace-write|danger-full-access`; `untrusted|on-request|never`); `-c`/`--config` `sandbox_mode=`/`approval_policy=` overrides validate against the config deserialization domains, where `on-failure` and `granular` are approval variants the typed flag rejects. Claude `--permission-mode` validates every occurrence against the case-sensitive commander choices (`acceptEdits|auto|bypassPermissions|manual|dontAsk|plan`). All rejects fail closed via `ProviderArgumentError` → `invalid_provider_arguments` in compose and identically in the launchers.
- FIX: Claude `--effort` mirrors the provider exactly instead of failing closed: values match case-insensitively and canonicalize to lowercase (`HIGH` → effective `high`); an unknown value is NOT rejected — Claude launches, warns, and applies its own default — so the token stays in argv but `resolved.reasoning` reports the provider-native fallback (`value:null, source:"native"`) via the new `ExplicitEffortRecognized` plan field.
- DECISION: Config-domain validation mirrors Codex deserialization semantics precisely: only the last `-c` override per policy key is validated (earlier repeats are masked by last-wins, probe exit 0), and a typed flag does not mask an invalid `-c` override (probe `codex exec --sandbox read-only -c 'sandbox_mode="banana"'` exit 1). Non-string TOML values (`sandbox_mode=true`) fail closed like the provider's "invalid type" reject. `model_reasoning_effort` was probed and is NOT config-validated by codex 0.145.0 (`"banana"` exit 0), so it stays unvalidated — no domain exists to enforce.
- EVIDENCE: Probes on codex-cli 0.145.0 / Claude Code 2.1.217 in `.temp/TASK-260724-35d94i/rework6/` (probe-01..27). All four reviewer repros re-run on the fixed binary: exit 1 + safe schema-v1 `invalid_provider_arguments` envelope; `--effort banana` composes ok with native reasoning. Child schema-v1 byte-identical to a HEAD-built binary on the MCP fixture; no-launch sentinel proof; full tests/vet/gofmt/diff-check green; no task-board dependency.

### 0126 — Primary-Session Policy Values Diverge from Provider Resolution
- REGRESSION: Primary-session compose returns `status:"ok"` and reports invalid native policy values as effective: Codex `--sandbox banana` / `--ask-for-approval banana`; Claude `--permission-mode banana` / `--effort banana`.
- EVIDENCE: Codex CLI 0.145.0 rejects the typed values with exit 2; Claude Code 2.1.217 rejects the permission mode with exit 1 and warns that unknown effort is ignored in favor of its default. Compose exits 0 with `resolved.sandbox`, `resolved.approval`, or `resolved.reasoning` equal to `banana`.
- ROOT CAUSE: `codex_launch.go` records typed policy strings without provider value validation; `claude_launch.go` does the same for permission mode and effort, so serialized provenance does not represent the provider's effective policy.
- STATUS: `TASK-260724-35d94i` requires rework; validate or otherwise normalize provider policy domains in the shared launch-plan builders and add contract/CLI coverage for invalid values.

### 0210 — Malformed Provider Arguments Fail Closed (0103 Resolved)
- FIX: `normalizeCodexExplicitSelections` fails closed with `ProviderArgumentError` (→ `invalid_provider_arguments`) for every recognized value-taking Codex option whose spaced value is missing or flag-like: `--model`/`-m`, `--profile`/`-p`, `-c`/`--config`, `--enable`, `--disable`, `--local-provider`, joining the policy flags that already had the rule (`takeCodexPolicyFlagValue` generalized to `takeCodexOptionValue`). A repeated `--profile`/`-p` occurrence in any spelling, even with equal values, also fails closed via `recordProfileFlag`.
- FIX: The Claude wrapper rejects a trailing `--model` without a value, mirroring the existing `--effort`/`--permission-mode` rule.
- EVIDENCE: Runtime probes on Codex CLI 0.145.0: each trailing option exits 2 "a value is required"; `--model -c foo=bar` exits 2 (flag-like value); `--profile a --profile b` and equal-value repeats exit 2 "cannot be used multiple times" (an `app-server --help` probe is insufficient — help short-circuits this validation). Claude exits 1 "option '--model <model>' argument missing". Compose now exits nonzero with the safe `invalid_provider_arguments` envelope for all these shapes, and `agents-infra codex --model` fails closed with the same parser. Client-fragment tokens stay deliberately unvalidated: they remain visible in `managed_client.argv` and the consumer fails closed by contract, so no allow-list drift.
- STATUS: Full tests/vet/gofmt/diff-check green; child schema-v1 byte-identical to a HEAD-built binary on the MCP fixture; happy-path primary-session plans unchanged.

### 0142 — Primary-Session Managed Split Made Total (0035/0036 Resolved)
- FIX: `codexManagedArgvSplit` replaces `codexManagedHostArgv` with a total three-way classification: config-level globals (`-c`/`--config`, `--enable`, `--disable`, `--strict-config`, `--profile` in all forms, `--oss`, `--local-provider`, `--search`, `--dangerously-bypass-hook-trust`) go to `managed_host.argv`; sandbox/approval policy stays resolved-only; every other token (`-C`/`--cd`, `--add-dir`, `-i`/`--image`, `--no-alt-screen`, `--remote*`, subcommands, prompt text, post-`--`) lands in the new `managed_client.argv` fragment in interactive order. Unrecognized future flags route to the client fragment instead of being dropped, closing the class.
- FIX: `normalizeCodexExplicitSelections` now parses attached `-mVALUE`/`-pVALUE`; they resolve with `cli:-m`/`cli:-p` provenance, suppress project-config model, and convert/preserve in host argv exactly like their spaced and `=` siblings.
- DECISION: Attached forms join the existing explicit-selection rules (equal duplicates normalize, conflicting explicit model values fail closed) for uniformity across all spellings of one field. Parser-only probes show codex 0.145.0 accepts even conflicting repeats (`-m a -m b` exit 0, last-wins); the wrapper keeps its stricter deterministic-provenance contract uniformly rather than per-form.
- EVIDENCE: Reviewer repros re-run live: globals compose now emits host `[--oss --local-provider ollama --dangerously-bypass-hook-trust --search app-server]` + client `[-C /tmp --add-dir /tmp --no-alt-screen resume --last]`; attached compose resolves model/profile with `cli:-m`/`cli:-p` and host `[-c model="gpt-5.4" -pspeed app-server]`. Composed host argv parser-probed exit 0. Child schema-v1 output byte-identical to a HEAD-built binary. Full tests/vet/gofmt/diff-check green; no task-board dependency.

### 0103 — Primary-Session Missing-Value Token Is Silently Dropped
- REGRESSION: Codex 0.145.0 rejects `--model` without a value with exit 2, but primary-session compose returns `status:"ok"`.
- ROOT CAUSE: `codex_launch.go:513` records a valueless model as explicit instead of returning `ProviderArgumentError`; `primary_session_launch_plan.go:317` then consumes the flag without routing it to host, client, or a resolved value.
- EVIDENCE: Compose emits interactive `--model`, managed host ending at `app-server`, empty managed client, and `resolved.model={value:null,source:"explicit_cli"}`.
- STATUS: `TASK-260724-35d94i` requires rework; enforce fail-closed missing-value handling and cover every value-taking managed-split class.

### 0036 — Primary-Session Attached Model/Profile Parity Gap
- REGRESSION: Codex 0.145.0 accepts attached short forms `-mMODEL` and `-pPROFILE`, but `normalizeCodexExplicitSelections` and `codexManagedHostArgv` do not classify them.
- EVIDENCE: Compose for an empty external project preserves `-mgpt-5.4 -pspeed` in `interactive.argv` while emitting `managed_host.argv=["app-server"]` and reporting both model/profile as native; `codex -mgpt-5.4 app-server --help` and `codex -pspeed app-server --help` both exit 0.
- STATUS: `TASK-260724-35d94i` requires rework so every provider-accepted model/profile form has correct provenance and managed-host semantics.

### 0035 — Primary-Session Managed Host Still Drops Codex Globals
- REGRESSION: `codexManagedHostArgv` preserves only a partial global-option set; Codex 0.145.0 accepts `--oss`, `--local-provider`, `--dangerously-bypass-hook-trust`, `-C`/`--cd`, `--add-dir`, `--search`, and `--no-alt-screen` before `app-server`, but compose drops all of them from `managed_host.argv`.
- EVIDENCE: One compose probe retained every option in `interactive.argv` and emitted only model/reasoning plus `app-server`; parser-only `codex <option> app-server --help` probes exited 0 for all listed classes.
- SCOPE: No separate managed client fragment or structured resolved field preserves the dropped session semantics, so a Session Manager would have to duplicate Codex argument parsing.
- STATUS: Prior 0023 “lossless host/client split” claim is incomplete; `TASK-260724-35d94i` routes to rework with focused coverage for every supported argument class.

### 0023 — Primary-Session Managed Host Global Args Resolved
- FIX: `codexManagedHostArgv` now derives `managed_host.argv` from the same normalized interactive argv instead of rebuilding a whitelist: arbitrary `-c`/`--config` overrides, `--enable`, `--disable`, `--strict-config`, and `--profile` keep their relative order, `--model` converts to its `-c model=` override, and the argv ends with `app-server`.
- DECISION: The host/client split is explicit per argument class — session policy (bypass flag, `--sandbox`/`--ask-for-approval`, `-c sandbox_mode=`/`approval_policy=`) stays out of host argv and lives in `resolved` for per-thread RPC application; subcommands, subcommand flags, and prompt/`--` tokens are client-only.
- EVIDENCE: The reviewer repro (`-c service_tier="fast" --enable web_search --strict-config`) now preserves all three arguments in `managed_host.argv` in order; managed-host contract tests (`TestBuildPrimarySessionLaunchPlanCodexManagedHostPreservesAppServerGlobals`, `TestCodexManagedHostArgvSplitsHostAndClientClasses`, plus the compose CLI test) cover preservation, forms, and the split.
- STATUS: Full `go test ./... -count=1`, `go vet`, `go build`, and `gofmt -l` green; child schema-v1 contract untouched (zero diff vs HEAD); README managed-host bullet updated to the derive-from-interactive rule; `TASK-260724-35d94i` second rework round complete.

## 2026-07-24

### 1858 — Primary-Session Native Policy Reflection Resolved
- FIX: Codex `--sandbox`/`-s`, `--ask-for-approval`/`-a` (spaced/`=`/attached forms) and `-c sandbox_mode=`/`-c approval_policy=` overrides now resolve into the primary-session contract with `cli:` provenance; a typed flag wins over `-c` regardless of order and repeated `-c` keeps last-wins, matching Codex resolution order.
- FIX: Claude `--effort` and `--permission-mode` resolve into `resolved.reasoning`/`resolved.approval` with commander last-wins duplicate semantics; absent effort now reports source `native` instead of `not_applicable`.
- DECISION: An explicit sandbox/approval/permission-mode selection suppresses project-config `yolo_mode` (`suppressed_by_explicit_cli`) — Codex clap rejects the bypass flag next to a typed policy flag (probe-verified on 0.145.0), and Claude `--dangerously-skip-permissions` silently overrides an explicit mode (probe-verified on 2.1.217: plan-mode YES/NO control pair).
- DECISION: Provider-parser-invalid pass-through args (repeated Codex policy flags, missing values, bypass+flag combos) fail closed with new error code `invalid_provider_arguments` instead of masquerading as project-config errors.
- STATUS: Focused contract/parity tests plus full suite, vet, gofmt, child byte-parity, and live compose smokes all green; `TASK-260724-35d94i` rework round.

### 1739 — Primary-Session Managed Host Drops Codex Global Args
- REGRESSION: `tools/agents-infra/internal/infra/primary_session_launch_plan.go:209` reconstructs `managed_host.argv` from MCP, model, reasoning, and profile only; valid provider user args accepted by `codex app-server` such as `-c service_tier="fast"`, `--enable web_search`, and `--strict-config` remain in `interactive.argv` but disappear from the managed host.
- EVIDENCE: A schema-v1 compose probe preserved all three arguments in `interactive.argv` while emitting only model/reasoning plus `app-server` for the host; Codex CLI 0.145.0 accepted the same global arguments before `app-server --help`.
- STATUS: `TASK-260724-35d94i` routes to rework; preserve host-compatible global arguments and ordering, keep client-only/session policy handling explicit, and add managed-host parity coverage.

### 1711 — Primary-Session Native Policy Parity Gap
- REGRESSION: `primary_session_launch_plan.go:227` reports Codex sandbox/approval as native whenever yolo is false, even when interactive argv explicitly carries `--sandbox` and `--ask-for-approval`; the managed app-server argv omits those flags and loses the requested thread policy.
- REGRESSION: `primary_session_launch_plan.go:278` marks Claude reasoning not applicable and approval native even when argv carries supported `--effort` and `--permission-mode` selections.
- EVIDENCE: `TASK-260724-35d94i` reviewer probes against Codex 0.145.0 and Claude Code 2.1.217 reproduce both mismatches; full Go tests remain green because native policy flags lack contract-parity coverage.
- STATUS: Route to implementation rework; resolve native selections with provenance and add Codex/Claude parity tests.

## 2026-07-23

### 1306 — Project-Safe Codex Config Rendering
- ROOT CAUSE: Local mode symlinked the full installed Codex config, exposing the user-level-only top-level `profiles` table at project scope.
- DECISION: `tools/agents-infra/internal/infra/codex_config.go` removes only top-level `profiles`, preserves all other valid TOML semantics, and reuses the platform-backed atomic writer.
- FIX: Local setup now renders a managed regular config; global setup retains the full symlinked config and profiles.
- STATUS: Focused/full tests, vet, build, coverage, setup smoke, doctor smoke, and cross-provider review pass.

## 2026-07-21

### 1357 — Standard Service Tier Sync Preserves Global User State
- FINDING: Source Codex template used `fast`; global config already used `default` and included trusted-project entries absent from the source template.
- DECISION: Set the source default to `default`, leave global runtime content untouched, and regenerate casual-talks' managed runtime through `agents-infra setup local --codex-config=global`.
- STATUS: Source, global effective config, and casual-talks runtime verify Standard; the retained `fast` profile is for explicit model/reasoning selection, not service-tier switching.

### 1232 — Local Attachment Launcher Preserves Go Exit Codes
- ROOT CAUSE: local `agents-infra` used `go run`, which converts a delegated process exit code such as usage `2` into `1` plus an `exit status 2` trailer.
- FIX: `tools/agents-infra/internal/infra/infra.go:947` preserves legacy trailers; `tools/agents-infra/internal/infra/infra.go:1069` builds and executes a local Go binary.
- EVIDENCE: subprocess coverage plus refreshed `/Users/alexis/src/casual-talks/.local/bin/agents-attachments` now return usage exit `2`.
- STATUS: Resolved.

### 1232 — Attachment Rollout Uses Legacy Thread-ID Suffix
- ROOT CAUSE: `tools/agents-infra/internal/attachments/attachments.go:733` previously matched a thread ID anywhere in a rollout basename and could select a newer unrelated session.
- FIX: require the legacy `rollout-*<thread-id>.jsonl` suffix before modification-time ordering.
- EVIDENCE: regression coverage creates competing files under a temporary HOME and selects only `rollout-run-needle.jsonl`.
- STATUS: Resolved.

### 1243 — Attachment Rollout Thread Selection Regression
- REGRESSION: `tools/agents-infra/internal/attachments/attachments.go:733` accepts any rollout basename containing `threadID`; a newer `rollout-…needle-unrelated.jsonl` wins over the legacy suffix-matching `rollout-…needle.jsonl`.
- EVIDENCE: `TASK-260721-2c1847` review artifact reproduces materialization from the unrelated session.
- STATUS: Route to rework; restore exact legacy thread-id filename matching and add regression coverage.

### 1219 — Attachment Usage Exit-Code Fix
- FIX: `tools/agents-infra/main.go` now preserves typed attachment helper exit codes at the process boundary.
- EVIDENCE: `/Users/alexis/.local/bin/agents-infra attachments` and `/Users/alexis/.local/bin/agents-attachments` both print usage and exit `2`.
- STATUS: Regression from 1217 resolved; re-review required for `TASK-260721-2c1847`.

### 1217 — Attachment Usage Exit-Code Regression
- REGRESSION: `tools/agents-infra/main.go:25` prints every returned error then exits `1`; `attachments.Run` marks usage as code `2`, but that signal is discarded.
- EVIDENCE: `go run . attachments` prints the Go helper's `exit code 2` error but exits `1`; legacy `.scripts/agents-attachments` returned `usage()` directly with `2`.
- STATUS: `TASK-260721-2c1847` requires a top-level usage-exit mapping and subprocess-level regression test before acceptance.

### 1211 — Windows Attachment Launcher Exit Regression
- REGRESSION: `tools/agents-infra/internal/infra/infra.go:945` joins each conditional launch with `& exit /b %ERRORLEVEL%`; CMD executes `exit` unconditionally and expands `%ERRORLEVEL%` before the delegated command.
- SCOPE: Installed `agents-attachments.cmd` cannot fall back to PATH when sibling launchers are absent and can return a stale success status after a Go-helper failure.
- STATUS: `TASK-260721-2c1847` routed to rework; add a Windows launcher-contract test and error-path coverage for the new attachment package.

## 2026-07-13

### 1804 — Claude Separator Review Correction
- FINDING: The prior separator concern is not a task regression: the detailed contract requires literal native danger input to be consumed while mirroring Codex, and `codex_launch.go:379` uses the same ordering.
- DECISION: Treat `--` as stopping wrapper-shortcut/selection parsing; retain native dangerous-flag de-duplication and provenance parity with Codex.
- STATUS: `TASK-260713-1soh7i` accepted after full Go validation and documentation audit.

### 1803 — Claude Yolo Separator Regression
- REGRESSION: `claude_launch.go:342` recognizes `--dangerously-skip-permissions` before checking `--`; a native argument after the separator suppresses `yolo_mode=false` as explicit CLI input.
- FINDING: `go run . claude --print-config -- --dangerously-skip-permissions` reports `effective_value: true` and `suppressed_by_explicit_cli`; the task requires only pre-separator arguments to participate.
- STATUS: `TASK-260713-1soh7i` routed to rework; add focused coverage for the native flag after `--`.

### 1757 — Claude Persistent Yolo Policy
- DECISION: `[agents.claude.primary_session].yolo_mode` has independent nearest-field precedence; explicit false masks inherited true and explicit Claude danger input suppresses project policy.
- FIX: Claude parse, launch resolution, print-config, setup, doctor, docs, and tests now mirror the Codex yolo contract with `--dangerously-skip-permissions` emitted at most once.
- STATUS: Full Go test, vet, build, 81.0% infra coverage, print-config/doctor smoke, and setup true→false→clear smoke pass.

### 1718 — Provider Session Policy Review Accepted
- MILESTONE: `TASK-260713-1bok5k` verified independent Claude provenance and no Codex model/reasoning/yolo leakage in both target repositories.
- STATUS: Uncached Go tests, vet, build, gofmt, diff-check, print-config, and doctor smokes passed.

### 1713 — Claude Primary Session Isolated from Codex Policy
- DECISION: `[agents.claude.primary_session]` owns only a non-empty Claude model; Codex model, reasoning effort, and yolo remain provider-local.
- FIX: `project_config.go`, `claude_launch.go`, setup, print-config, and doctor compose and report Claude provenance independently.
- STATUS: Full Go test, vet, build, and two-repository Claude print-config/doctor smokes pass with `claude-opus-4-6`.

### 1610 — Primary-Session Operator Contract Documented
- FIX: `README.md` and `SKILL.md` now document the primary-session TOML, independent nearest-field precedence, yolo scope, setup/clear, render, doctor, native fallback, and `.codex/config.toml` coexistence.
- FIX: Corrected the MCP example to `[mcp]`; task-board spawn-ceiling ownership is cross-linked without duplicating its policy.
- STATUS: CLI usage and primary-session Go tests were checked against the active implementation.

### 1608 — MCP Skill Example Used Retired Table Path
- FINDING: `SKILL.md` documented `[codex.mcp]`, while `project_config.go` reads the canonical `[mcp]` table.
- FIX: Primary-session documentation update will correct the MCP example alongside the current launcher contract.

### 1558 — Codex Alias Danger Moved to Project Policy
- FINDING: Active `codexD` is owned only by the user-authored `~/.zshrc:134`; no separate tracked alias definition exists, while `.instructions/INSTRUCTIONS_TOOLS.md:53` owns the shared documentation.
- FIX: Removed the alias-level `-d`; retained `agents-infra codex -d` as the documented explicit ad-hoc full-trust escape hatch.
- STATUS: Fresh zsh smokes in relux-agents-infra and skill-project-management render no wrapper danger expansion and exactly one final native danger argument from each target's `yolo_mode = true` project profile.

### 1537 — Sibling Board Has Pre-Existing Orphan Resources
- FINDING: skill-project-management `task-board validate` reports six tracked orphan resource files/directories dated February–April 2026.
- SCOPE: Target setup did not touch `.task-board`; its config checksum stayed byte-identical and contains no primary-session field.
- STATUS: Left unchanged as unrelated board debt; relux-agents-infra board validation passes.

### 1535 — Target Primary Codex Profiles Active
- MILESTONE: `agents-infra setup local` configured `gpt-5.6-terra`, `xhigh`, and `yolo_mode = true` in relux-agents-infra and skill-project-management.
- STATUS: Both doctor and print-config report target-local provenance, unchanged effective MCP enablement, and exactly one native danger flag in the primary Codex argv.

### 1534 — Concurrent Sibling Writes During Rollout
- ANOMALY: skill-project-management tracked files changed concurrently while target setup ran; `tools/board-cli/cmd/root.go` appeared after the before-state snapshot and board agents remained active on spawn-ceiling work.
- SCOPE: Setup's tracked delta there is the generated marker in `AGENTS.md`; its original bytes remain in `.agents/.instructions/AGENTS.project.md`.
- DECISION: Preservation evidence uses target config, task-board config, MCP registry, and rendered argv checksums instead of an unstable whole-worktree hash; concurrent edits were preserved.

### 1513 — Platform-Backed Project Config Replacement
- DECISION: `project_config_replace_posix.go` uses same-filesystem POSIX rename; `project_config_replace_windows.go` uses `github.com/natefinch/atomic.ReplaceFile` backed by `MoveFileExW(REPLACE_EXISTING|WRITE_THROUGH)`; unsupported targets fail closed.
- FIX: `project_config_setup.go:506` delegates final replacement; focused failure coverage proves original-byte and temporary-file preservation; Windows-only tests cover successful replace and delete-locked failure preservation.
- STATUS: Full/race/vet/coverage/build gates and Windows amd64/arm64 test compilation pass. Local Wine execution was unavailable because the cask requires sudo and the portable archive was Gatekeeper-killed; no platform protections were bypassed.

### 1455 — Windows Atomic Replace Contract Gap
- FINDING: `tools/agents-infra/internal/infra/project_config_setup.go:529` ends the guarded write with `os.Rename`; Go explicitly does not guarantee `Rename` atomicity on non-Unix platforms.
- SCOPE: Windows is supported through `scripts/setup.ps1`; a Windows cross-build proves compilation, not atomic replacement semantics.
- STATUS: `TASK-260713-4ihi4q` review routed to rework for a platform-backed atomic replace and Windows-specific evidence; Unix tests and black-box failure preservation pass.

### 1444 — Global Project-Config Collision Rejected
- REGRESSION: `setup local "$HOME"` accepted primary-session flags and wrote the global runtime config that project discovery intentionally ignores.
- ROOT CAUSE: `project_config_setup.go:56` did not enforce discovery's global-path exclusion, and lexical absolute-path equality missed filesystem aliases.
- FIX: `project_config_setup.go:56` rejects set/clear before sync; `infra.go:1200` compares existing ancestor identity plus unresolved suffix, with Windows case folding.
- STATUS: Exact set/clear, symlink-alias, CLI, full/race/vet/build, and two-repository smoke coverage pass.

### 1423 — Setup Flags After Project Path
- ROOT CAUSE: Go `flag.FlagSet` stops at the first positional argument; documented `setup local PROJECT --flag` calls left trailing flags unparsed and could skip the requested profile mutation.
- FIX: `tools/agents-infra/main.go` extracts the leading local project before flag parsing, rejects extra positionals, and preserves explicit `--codex-yolo-mode=false`; `tools/agents-infra/setup_test.go` covers the path-first form.
- STATUS: Set, partial update, no-flag byte preservation, diagnostics, and clear passed in disposable relux-agents-infra and skill-project-management worktrees.

### 1408 — Placeholder Setup Task Retired
- FIX: Closed `TASK-260713-22gp48` as an accidental placeholder duplicate; `TASK-260713-4ihi4q` remains the sole owner of local primary-session setup flags, atomic TOML merge, preservation, and tests.
- MILESTONE: Regenerated `STORY-260713-3vxko6` plan retains the closed duplicate only for audit and includes target profile rollout `TASK-260713-25bqi7` plus alias cleanup `TASK-260713-1ripj2`.
- STATUS: Active primary-session tasks have complete briefs, dependencies, and checklists; agents-infra Go tests, board validation, and diff checks pass.

### 1340 — agents-infra Owns Project Primary Codex Policy
- DECISION: `[agents.codex.primary_session]` in project `.agents/.configs/project-config.toml` owns optional model, reasoning effort, and `yolo_mode`; nearest ancestor field wins and explicit false masks inherited yolo.
- DECISION: task-board remains a separate consumer of `task-board.config.json -> spawn.ceilings` and does not provide primary-session defaults to agents-infra.
- SCOPE: `STORY-260713-3vxko6` decomposes launch composition, safe local setup, print-config/doctor evidence, docs, and disposable validation of relux-agents-infra plus skill-project-management.
- STATUS: Cross-repository contract and diagrams are linked as task preconditions from `TASK-260713-190sng` in skill-project-management.

## 2026-07-10

### 1155 — Board-Agnostic Image Intake
- DECISION: Image intake belongs in `.scripts/agents-attachments` as `stage-images`, not in board-specific scripts or resources.
- DECISION: Source-to-staged mappings persist redacted source labels plus content hashes; raw ICCID/IMSI/key-like labels are not written into staged filenames.
- SCOPE: `.scripts/agents-attachments`, `.instructions/INSTRUCTIONS_ATTACHMENTS.md`, `SKILL.md`, `README.md`, `tests/test_agents_attachments.py`.
