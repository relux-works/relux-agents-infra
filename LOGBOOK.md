# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

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
