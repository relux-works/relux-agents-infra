# TASK-260916-38vqh4 — Inventory and deprecation plan

Ready for review. Research only; no repository or host configuration changes.

## Scope and decision

Inventory is pinned to worktree HEAD `459742ea67e3c6b84169520b92d74fe7f73e3002`, matching the brief's control revision. Paths below are relative to that repository unless explicitly absolute. The installed binary version in the brief (`main-7a0a24c`) is not treated as source parity. No upstream refresh, branch manipulation, installation, or launch was performed.

Decision to unblock: where to cut agents-infra's context/launcher responsibilities without breaking task-board's composition and preparation interfaces. Recommendation: deprecate direct proprietary-provider execution immediately; remove ordinary setup instruction/skill distribution and bundled MCP registry distribution; preserve the compatibility implementations actually required by compose/prepare. Full deletion of the v1 preparation renderer requires a coordinated consumer migration, not a fabricated success report.

Research bound: one research leaf, 60 minutes, one report with source references (no archives). Frozen precondition: existing schema-version 1 compose/prepare contracts, including their semantic checks, rather than a new wire format. Exit: inventory, replacement evidence, concrete edits and the compatibility limitation recorded. First production slice: implementation task in STORY-260916-1vt3x2 introduces deprecation dispatch and negative executable tests, then separates setup from v1 compatibility code. No second research prerequisite is recommended.

## Important findings

1. **MCP composition is not launcher-only code.** `internal/infra/child_launch_composition.go:94` loads the project config and registry and uses helpers in `codex_launch.go` and `claude_launch.go`. Primary-session composition calls `BuildCodexLaunchPlan` and `BuildClaudeLaunchPlan` (`primary_session_launch_plan.go:275,459`). Deleting these files breaks task-board even if direct launchers become stubs.
2. **v1 prepare requires real instruction artifacts.** `primary_session_prepare.go:114` regenerates Codex instructions; `:142` regenerates Claude entrypoint/links. The external task-board validator explicitly requires these files, states and evidence. A residual-only report under unchanged schema 1 is rejected. Retain a narrowly scoped v1 compatibility path until the consumer supports the residual contract. Do not report no local runtime for an installed runtime or stamp false rendered booleans true.
3. **The LLDB wrapper requested by the keep list is absent in this revision.** `.scripts/` contains only setup-symlinks.sh and setup-pdf-tools.sh. No production LLDB implementation or registry entry was found in tools/agents-infra, scripts, .scripts or .configs. README.md:2547 explicitly says there is no bootstrap. Preserve user-provided stdio definitions through compose; do not invent or restore an LLDB installer as part of this change. This is a bounded repository finding, not verification of third-party LLDB availability.
4. **B5 proves managed homes/MCP/skills, not blanket context injection parity.** B5:696–716 reports no generated AGENTS.md/CLAUDE.md and no Claude system-prompt key unless explicitly requested. Do not claim that `curator run` reproduces arbitrary project `@` include semantics by default.
5. `openai-infra`/`anthropic-infra` are separate dispatch paths but the same primary-provider launch surface for this migration. Also cover `openai-dange`/`anthropic-dange` (`target-yolo`) to avoid an undeclared bypass.

## Evidence key (Curator B5)

B5 = `/Users/administrator/Developer/ReluxWorks/curator/curator/.task-board/.resources/TASK-260908-yl5x3k/TASK-260908-yl5x3k_onboarding-evidence-rev2.md`.

| Key | Exact evidence | What it establishes / bound |
| --- | --- | --- |
| E1 | B5:318–347 | `curator profile install git@github.com:relux-works/relux-root-context.git --directory packages/relux-root-context-ivan --range '^1.0' --use --takeover`, exit 0; profile activation switches native scopes. Install does not itself provision managed homes. |
| E2 | B5:655–666; 1063–1083 | Pinned agents-attachments/pdf/skill-creator links in managed homes; Claude/Codex skills current, Pi skills current. Not proof that every old local skill was imported. |
| E3 | B5:668–694; 893–908 | Curator renders Claude JSON and Codex TOML for figma/safari; Codex MCP listing exit 0. Safari binary unresolved and Figma OAuth incomplete. |
| E4 | B5:696–732; 1092–1098 | Context packages pin the instruction bytes from this exact infra revision; fragments replace native entrypoint ownership. No materialized instruction entrypoints; Claude prompt injection needs explicit option. Arbitrary local includes/default Codex context delivery are not established by this evidence. |
| E5 | B5:751–865; 937–941 | First `curator run claude_code|codex_cli|pi -- --version` provisions homes (exit 0); real Codex prompt exit 0; Claude and Pi prompts exit 1 for auth. `curator env resolve <env> --repair` is named as explicit repair. |
| E6 | B5:31–51; 1052–1104 | Prior doctor/native-state evidence and post-PR72 managed-home status. Accepted attached evidence, not rerun here. |

Every REMOVE row below names these replacements. E4 is an ownership migration with a documented parity limit, not a green runtime attestation.

## Command inventory

For compactness, `main.go`, `cmd/...` and `internal/...` in these tables are under `tools/agents-infra/`; unqualified infra implementation filenames such as `infra.go` are under `tools/agents-infra/internal/infra/`. Bootstrap table shorthand `setup.sh`/`setup.ps1` means `scripts/setup.sh`/`scripts/setup.ps1`.

| Surface and source | Class | Implementation disposition |
| --- | --- | --- |
| setup global — main.go:165; infra.go:135 | KEEP | Residual runtime installer, config merge/settings/rules/helpers/receipt. Remove normal instruction/skills/MCP distribution in component rows below. |
| setup local + config/policy flags — main.go:165; project_config_setup.go:1 | KEEP | Keep path recursion refusal, atomic project settings, preserve/global/local config selection, Pi assets. Local instruction scaffold retained only for explicit v1 compatibility preparation. |
| refresh-links — main.go:264; infra.go:284 | KEEP | Refresh residual settings/rules/helpers, not normal context/skill fan-out. |
| doctor global/local — main.go:319; infra.go:303 | KEEP | Report residual installation and project policy; old instruction/skill booleans must be observational/legacy, never a residual health requirement. |
| verify global/local — main.go:296; runtime_receipt.go:111,163 | KEEP | Validate residual source assets, completed-install receipt, helpers/backend/catalog. Remove skill topology requirements once copying stops. |
| compose child — main.go:1023; child_launch_composition.go:94 | KEEP | Exact v1 contract, MCP-only argv, required env names, provenance, strict errors and empty arrays. Keep legacy registry reader for caller-supplied registries. |
| compose --mode primary-session --agent codex/claude/pi and --entrypoint — main.go:1090,1188; primary_session_launch_plan.go:207 | KEEP | Non-launching task-board API, provider policies, argv partitioning, canonical targets and MCP compatibility. Deprecation must not intercept this path. |
| prepare --agent codex/claude — main.go:1129; primary_session_prepare.go:78 | KEEP | Exact v1 API and real evidence. Its instruction rendering remains a compatibility exception pending consumer migration (see below). Skill fan-out is not required by the consumer artifact contract. |
| attachments list/show/path/materialize/stage-images — main.go:145; internal/attachments/attachments.go:1 | KEEP | Manifest/env/fallback, caller CWD and exit codes remain. |
| codex — main.go:436 | DEPRECATE | Replace direct execution, preparation and diagnostic branch with deterministic exit-1 migration error. |
| claude — main.go:465 | DEPRECATE | Same, points to claude_code. |
| pi / --print-config / spawn / turn / lifecycle status / retire-legacy — main.go:494,537,602,659,727 | KEEP | Preserve local-model ownership, contained state, typed turn contract, lifecycle refusal checks, native pass-through policy. |
| runtime status/stop — main.go:799 | KEEP | Broker inspection/stop, force/deadline behavior. |
| runtime quarantine — main.go:854–869 | KEEP | Pi local-model runtime residual; SetSharedRuntimeManualQuarantine sets operator quarantine. |
| runtime unquarantine — main.go:854–869 | KEEP | Pi local-model runtime residual; the same dispatcher clears operator quarantine. |
| runtime broker — main.go:870–883 | KEEP | Pi local-model runtime residual; RunSharedRuntimeBroker retains required runtime-key/profile-project/profile validation. |
| runtime runtime-launch — main.go:870–885 | KEEP | Pi local-model runtime residual; RunSharedRuntimeLauncher retains the same required identity/profile arguments. |
| target qwen-infra and spawn — main.go:928,957; canonical_target.go:285 | KEEP | Pi targets and locked model/environment/profile semantics. |
| target openai-infra/anthropic-infra — main.go:928,957 | DEPRECATE | Same outward launch responsibility as codex/claude, despite different canonical resolver. Block before config reads/argument parsing/launch. |
| target-yolo openai-infra/anthropic-infra — main.go:945 | DEPRECATE | Undocumented run switch at main.go:82; direct aliases must not bypass deprecation. No automatic transfer of unsafe flags. |
| version/--version — main.go:1220; help/-h/--help — main.go:53,1273 | KEEP | Version metadata stays; update usage/deprecation text. Preserve unrelated existing help exit behavior. |
| model-check — main.go:103 | KEEP | Bounded Pi behavior evidence and target checks. Its production validator rejects non-Pi environments (internal/infra/model_check.go:276); preserve that refusal and Pi behavior. |
| model-harness version/run/render/doctor/stress/help — cmd/model-harness/main.go:25,43,97 | KEEP | Separate executable and local-runtime harness remain installed. |

## Every Go package

`go list ./...` returned exactly 5 packages (exit 0). No package is wholesale removable.

| Package/source | Class | Rationale |
| --- | --- | --- |
| main — main.go:1 | KEEP | CLI dispatcher retained; direct provider branches deprecated. |
| cmd/model-harness — cmd/model-harness/main.go:1 | KEEP | Harness executable. |
| internal/attachments — internal/attachments/attachments.go:1 | KEEP | Attachment contract/helper. |
| internal/infra — internal/infra/infra.go:1 | KEEP (split responsibilities) | Retain Pi, config, runtime receipts, target and task-board contracts; component-level removals follow. |
| internal/modelharness — internal/modelharness/config.go:1 | KEEP | Profile/engine knob/pinned distribution/run/stress/process implementation. |

All `internal/infra/pi_*.go`, `agents_management_*.go`, `model_check.go`, `canonical_target.go`, `project_config*.go`, `launcher_backend.go`, `runtime_receipt.go`, and `source_dir.go` remain, subject only to removal of obsolete setup requirements. This includes platform-specific broker/client/process/terminal code and the pinned Pi release manifest (`source_dir.go:81`). External Go modules are dependencies, not extra local packages; do not prune them merely because a launcher is deprecated.

## Setup components and inputs

| Component/source | Class | Rationale and replacement for each REMOVE |
| --- | --- | --- |
| Ordinary global instruction copy — infra.go:411,428 | REMOVE | Curator profile install + managed launcher context packages (E1,E4,E5); preserve original files as migration source until compatibility exits. |
| Ordinary local scaffold and instruction links — infra.go:472,846,887 | REMOVE | Curator ownership (E1,E4,E5), but do not remove private v1 prepare support in first slice. Local custom instructions are not proven migrated. |
| Ordinary `@` rendering / provider entrypoint generation — infra.go:1548,1562,1580,1585,1614,1638,1678,1693 | REMOVE | Profile context composition/launcher fragments (E4,E5), bounded by missing default injection proof. Retain these functions for v1 prepare only, not setup/refresh/direct launch. |
| Repo-skill materialization and `.skills`→skills→provider fan-out — infra.go:693,733,755,787,835,846,887,972 | REMOVE | Profile skill members and pinned managed-home links (E1,E2,E5). Do not delete unmanaged or Curator-owned installed paths. |
| Fan-out-only validation — skill_link_validation.go:12,29 | REMOVE | Ownership moves to Curator managed-home skills (E2). Remove validation calls only after source skill copying/fan-out are removed; do not leave unsafe copying without validation. |
| Bundled `.configs/codex-mcp-servers.toml:1` figma/safari definitions and automatic copy by syncRepo | REMOVE | Curator profile MCP rendering (E1,E3,E5). Preserve pre-existing caller registry files used by compose. No parity claim for safari command path (B5 uses safaridriver-mcp; source uses STP absolute path). |
| MCP composition inside executable codex/claude branches — main.go:436,465; codex_launch.go:114; claude_launch.go:106 | REMOVE | `curator run codex_cli|claude_code` dispatch from managed homes (E3,E5). Keep shared pure builders/readers for compose. |
| Legacy registry parse/load — infra.go:990,1024,1033; codex_launch.go:1012,1032 | KEEP | Compatibility backend for compose; deleting it now violates keep requirement. Registry distribution removed, registry read contract retained. |
| `.configs/claude-settings.json:1`, settings link — infra.go:874 | KEEP | Explicit residual ownership. Split link operation out of setupClaude. Preserve local custom settings according to infra.go:504. |
| `.configs/codex-config.toml:1`, merge/render — codex_config.go:13,75,110; infra.go:919 | KEEP | Preserve existing merge semantics and local config mode; no new Curator seed overwrite. |
| `.configs/templates/qwen-3.8-27b-mlx-8bit.project-config.toml:1` | KEEP | Pi local-model profile template. |
| Project-config primary policy/targets/Pi/MCP enable list — project_config.go:1 | KEEP | Used by task-board contracts and Pi. No broad TOML schema pruning in this release. |
| `.rules/default.rules:1` and rule links — infra.go:955 | KEEP | Residual Codex rules. |
| `.instructions/AGENTS.md:1` and INSTRUCTIONS.md:1 | REMOVE from normal install | E1,E4,E5; source files retained for v1 compatibility and Curator provenance, not native-home distribution. |
| INSTRUCTIONS_PLATFORM/STRUCTURE/BROWSER_AUTOMATION/REMOTE_AGENTS/TOOLS/SKILLS/SKILL_TRIGGERS/DIAGRAMS/TESTING/WORKFLOW/DOCS/STYLE `.md:1` each | REMOVE from normal install | Entire 12-module set moves to profile context packages (E4:727–732). Keep source bytes initially; future deletion needs source-of-truth migration, since B5 says change bytes at this source and re-cut. |
| `.instructions/INSTRUCTIONS_ATTACHMENTS.md:1` | KEEP contract document; REMOVE automatic injection | Helper/manifest remains; Curator attachment context replaces distribution (E4:1093; E2 agents-attachments). Keep source reference or relocate to docs with link updates in a later compatible change. |
| Source resolution, recursion gates, skip/scrub, install receipt — infra.go:211,533,582,638; source_dir.go:300; runtime_receipt.go:58,69 | KEEP | Narrow installed asset requirements; preserve refusal/error behavior and source/backend integrity. |
| agents-infra wrapper/build backend — infra.go:1466 | KEEP | Required by installed residual commands. |
| agents-attachments wrappers/legacy cleanup — infra.go:1338,1352 | KEEP | Backwards-compatible attachment helper. |
| pi-infra/qwen-infra wrappers — infra.go:1285,1202 | KEEP | Pi launchers. |
| openai-infra/anthropic-infra and -dange wrappers — infra.go:1191,1202,1247 | DEPRECATE | Keep wrappers for one release; central dispatch returns migration error. |
| `<project>/.local/bin/codex-local` local wrapper — infra.go:116–132 (BinDir at :131), :1414–1452 | DEPRECATE | It forwards to the deprecated CLI; preserve a clear error instead of silently invoking raw Codex. |
| lldb-mcp wrapper | KEEP if user supplied; absent here | No code to retain/create. Local stdio definition compatibility remains. README.md:2547–2558 and exhaustive production-tree search bound the finding. |

## Bootstrap scripts: every step

These scripts bootstrap binaries and then invoke setup; they do not independently implement instruction composition. Keep their residual purpose, change the delegated setup behavior above. Do not run installers from a hosted task-board session.

| Step | POSIX source | PowerShell source | Class / action |
| --- | --- | --- | --- |
| Defaults/options/output helpers | scripts/setup.sh:5,19,27,39 | scripts/setup.ps1:1,8,20 | KEEP bin/PDF flags, platform paths and diagnostics. |
| Resolve config/install state dir | setup.sh:61 | setup.ps1:13 | KEEP. |
| Ensure/install Go | setup.sh:72 | setup.ps1:28 | KEEP (required for both executables). |
| Git version/commit/date ldflags | setup.sh:91 | setup.ps1:46 | KEEP. |
| Build agents-infra + model-harness | setup.sh:99 | setup.ps1:74 | KEEP. |
| Install both binaries | setup.sh:110,116 | setup.ps1:85 | KEEP. |
| Write install.json source/bin/platform metadata | setup.sh:126 | setup.ps1:93 | KEEP. |
| Optional PDF toolchain | setup.sh:181; .scripts/setup-pdf-tools.sh:1 | setup.ps1:151 | KEEP optional local tools; Windows remains warning-only. B5 skill install is not proof of installing pandoc/poppler. |
| PATH advice/update | setup.sh:153 | setup.ps1:108 | KEEP. |
| Verify files, both version commands | setup.sh:161 | setup.ps1:126 | KEEP; use exact native exit status. |
| Invoke setup global then doctor global | setup.sh:168 | setup.ps1:138 | KEEP residual setup; no implicit Curator takeover. Add `verify global` to test actual residual postconditions if installer verification is revised. |
| Main execution order | setup.sh:173 | setup.ps1:143 | KEEP; update descriptive text. |
| Compatibility symlink helper | .scripts/setup-symlinks.sh:13 | n/a | KEEP delegates to narrowed refresh-links. |

PowerShell currently relies on `$ErrorActionPreference` around native Go/CLI programs and pipes checks to Out-Null. Implementation should explicitly check `$LASTEXITCODE` after each native gate; an exit 1 must not print a successful installation. This is an adjacent installer correctness risk, not evidence of a failure observed here.

## Exact deprecation behavior

Recommended first-release behavior (decision, not existing behavior): write one line to stderr, no stdout, **exit 1**, launch nothing, make no filesystem changes, perform no config reads or provider resolution. Use existing main error path (main.go:30) so no new exit-code plumbing is needed.

Messages, including punctuation:

- `agents-infra codex is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release.`
- `agents-infra claude is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release.`
- `openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release.`
- `anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release.`

For target-yolo use the same provider message with `openai-dange` / `anthropic-dange` in its first clause. `agents-infra target <name>` reports the canonical alias message. The local `<project>/.local/bin/codex-local` wrapper reports the CLI codex message (LocalLayout BinDir: infra.go:131; filename: :1418; delegation: :1441–1452). Do not auto-exec Curator or forward danger flags.

Keep the shim's current conditional lifecycle in setupCodexLocalLauncher (infra.go:1414–1438): local setup with nonempty project MCP opt-ins creates or preserves `<project>/.local/bin/codex-local`; without opt-ins, remove only the recognized generated shim through removeGeneratedCodexLocalLauncher (:1455), preserving unrelated user files. Removing bundled registry distribution does not change this opt-in decision. The shim delegates to the deprecated codex branch and therefore returns its stderr message and exit 1 without resolving registry definitions or launching a provider.

**`--print-config` does not survive on deprecated launch entrypoints**: it receives the same exit-1 message, including combined or malformed provider args. Keep non-launching `compose --mode primary-session … --json` diagnostics for task-board and `pi`/qwen diagnostics. Do not conflate preserving a pure launch-plan builder with keeping a deprecated command operational.

`openai-infra` and `anthropic-infra` wrappers call `target`, not `runCodex/runClaude` (infra.go:1231). Canonical provider mapping (canonical_target.go:83) resolves Codex/Claude; main.go:957 prepares and execs their primary-session plans. Thus they are the same user-facing responsibility but not identical functions. Guard `runTarget` immediately after selecting those names, before the `spawn` branch; guard `runDirectProviderYoloTarget` similarly. Keep `BuildCanonicalTargetLaunchPlan` callable by compose and Pi contracts.

## Compatibility evidence and exact edit sequence

External consumer inspected at `/Users/administrator/Developer/ReluxWorks/skill-project-management`, HEAD `b2e26c038ddfe38392aa86412eb066fd12d45462` (source observation; installed task-board binary parity unverified). Prefix TB below = `tools/board-cli/` in that repo.

- TB `internal/spawn/launch_composition.go:735` invokes `agents-infra compose --agent … --project … --schema-version 1 --json` and bounds output.
- TB `cmd/infra_prepare.go:58` invokes prepare; `:93` decodes/validates before launch.
- TB `internal/sessionmanager/launch_plan.go:203,244,279–313` requires real local-runtime instruction states/artifacts, and forbids a purported no-op with runtime/artifact fields. Remaining validation checks path, state, hash/target and duplicate/missing artifacts through :383.
- TB `cmd/codex_manager.go:184` and `cmd/claude_manager.go:219` consume primary-session compose. Therefore keep primary-session builders, not only child MCP composition.

### Slice A: safe in this repository

1. **main.go:** replace runCodex/runClaude bodies with errors above. Add a small deprecation helper for fixed surface/provider mappings. Add guards in runTarget/runDirectProviderYoloTarget before any parsing or side effects. Update usageText. Leave runCompose/runPrepare and pure builders working.
2. **infra.go setup split:** extract `setupClaudeSettings`, `setupCodexRules` and existing config-mode operation from setupClaude/setupCodexWithConfig. RefreshLinks calls residual helpers, setupHelpers and installCLIWrapper. Remove ensureRepoSkillLinks/fan-out calls. Remove `.skills` and `skills` copies in syncRepo at the same time as removing validateSourceSkillLinks/managedSkillLinkFailures setup/verify requirements. Preserve installed user/Curator files; no recursive deletion of native homes.
3. **Instruction split:** move writeClaudeEntrypoint/writeCodexEntrypoints and rendering/include helpers into `legacy_prepare_instructions.go`, called only by PreparePrimarySession's v1 path. Prepare invokes those plus residual rules/settings, never skill fan-out. Remove ordinary setup/refresh rendering and global instruction copy. Keep local scaffold creation reachable from v1 prepare for newly installed residual runtimes, so prepare can produce the honest required files; preserve existing custom project inputs. Retain source instruction bytes for that compatibility path. Document this exception explicitly rather than claiming total instruction removal.
4. **MCP split:** stop shipping/copying bundled codex-mcp-servers.toml. Extract `loadCompositeMCPRegistry`, `mergeMCPRegistry`, `codexMCPServer`, parsing/validation, codexMCPConfigArgs and marshalClaudeMCPConfig into `mcp_contract_compat.go` if useful for ownership clarity; preserve behavior and caller-provided global/local registries. Keep BuildCodexLaunchPlan/BuildClaudeLaunchPlan, argument normalizers, primary-policy resolution and rendering helpers as required by primary-session compose; delete only functions with no remaining references. Removing bundled definitions can expose existing enabled_servers entries with no definition: return the existing explicit error, never silently drop servers.
5. **source_dir.go/runtime_receipt.go:** sourceAssets no longer requires SKILL.md/README solely for skill packaging, or global instruction entrypoints solely for setup. Keep config/rules/backend/Pi catalog requirements. Keep receipt invalidation, validation, selected source refusal and recursion checks. For v1 prepare's inputs, validate at point of compatibility use; do not weaken residual runtime verification or accept malformed read as absence. Adjust tests/helpers for the smaller installed asset set.
6. **Diagnostics:** Doctor keeps fields if external consumers may use them, describes them as legacy observations; health requirements and docs must not insist native homes are infra-managed. B5 doctor false/false is not a reason to reclaim ownership.
7. **Cleanup only after reference check:** ensureRepoSkillLinks, materializeRepoSkill, stale/dangling skill repair and shouldSkipSkillFanout can be deleted once no callers remain. skill_link_validation.go can go only with all copying/fan-out removed. Keep generic filesystem helpers and skip/scrub routines used by Pi/setup. Keep .instructions source files and attachment contract initially; do not delete all codex_launch.go/claude_launch.go.

### Slice B: coordinated retirement, not part of the safe first deletion

Options: (a) retain isolated v1 compatibility renderer (recommended now; larger retained code, stable consumers); (b) coordinate task-board with a new residual prepare contract/version and Curator-managed launch semantics, then delete renderer and registry backend (clean ownership, cross-repository deployment); (c) silently return no-op/empty MCP/fake hashes (rejected: changes contract or falsifies evidence).

Full instruction renderer deletion is gated on TB ValidatePrimarySessionPreparationReport and invocation negotiation accepting an honest residual state. Full registry-reader deletion is gated on child/primary composition obtaining equivalent MCP data from an agreed Curator adapter, including project opt-ins, provenance, required env names and strict errors. B5 contains no such adapter proof. Do not substitute empty arrays for configured servers. This is a concrete implementation boundary; research can hand off the safe slice without claiming full retirement is currently possible. If implementation is required to delete all compatibility code in the same release, the exact architecture decision needed is coordinated task-board contract migration versus retaining v1 compatibility.

## Tests to retain, replace, add

Use existing Go test infrastructure; no new harness framework.

| Test files | Exact change |
| --- | --- |
| main_test.go:700,1153,1181,1381,1397,1417,1470,1477,1528,1579,1629 | Retain compose/prepare schema/error/argv-policy/secret tests. Replace parity with deprecated direct execution by parity against pure builders. |
| main_test.go:1204 `TestDirectLaunchAndPrepareCommandRenderIdenticalProviderArtifacts` | Replace with prepare-only real artifact verification and separate no-side-effect deprecated executable tests. |
| internal/infra/child_launch_composition_test.go:11,118,178,200 | Retain deterministic MCP-only argv, Claude JSON, empty-array and invalid-project negative tests. Add missing-definition fixture after bundled registry removal. |
| internal/infra/primary_session_prepare_test.go:10–347 | Keep config bytes/symlink/absence preservation, ancestor and no-runtime checks. Keep instruction artifact assertions in v1 compatibility; replace skill refresh expectations with skill non-mutation. |
| internal/infra/codex_launch_test.go, claude_launch_test.go, primary_session_launch_plan_test.go, canonical_target_test.go | Keep pure builder/policy/MCP tests; these remain production code through compose. Retire only direct-execution assumptions. |
| internal/infra/infra_test.go, source_dir_test.go, source_sync_build_artifacts_test.go, runtime_receipt_test.go | Replace ordinary instruction/skill installation expectations with absence/non-mutation; keep recursion/failed reads/backend/catalog/receipt postcondition negatives. Change source fixtures deliberately, not broad removal of refusal tests. |
| installed_binary_setup_test.go:308,338,409,475,502,544,586; agents_management_boundary_test.go | Remove or rewrite skill-fan-out topology expectations only when surface no longer copied; keep unrelated boundary gates. Retain recursion and backend refusal tests at :105,1174,1206,1265,1293,1377. Replace missing instruction include setup expectation at :1347 with prepare compatibility error test. |
| installed_binary_setup_test.go:814; canonical_target_main_test.go | Replace provider-launch success with executable deprecation cases; keep qwen target paths and compose entrypoint tests. |
| internal/infra/infra_test.go:1069,1122–1133 | Retain/adapt TestSetupLocalProjectMCPOptInInstallsCodexLocalLauncher and no-opt-in/removal tests for the actual `<project>/.local/bin/codex-local` shim; executable deprecation tests invoke that path. |
| All pi_* tests, internal/modelharness tests, attachments tests, shipped_codex_policy_test.go, setup_test.go | Retain; adjust only shared setup fixtures/docs assertions affected by smaller installer. |

Add `deprecation_main_test.go` using a compiled CLI, isolated HOME/project, fake provider and fake Curator executables that write sentinels. Table covers codex, claude, both target aliases, target-yolo aliases, installed wrappers and local Codex wrapper; args include no args, --print-config, --help, danger flags, malformed provider args and `spawn`. Assert actual exit 1, exact stderr, empty stdout, no sentinel and unchanged filesystem. Narrow the guard to one alias/provider in a temporary mutant and prove the corresponding other rows fail (do not only delete the whole guard).

Add real setup→prepare→TB decoder compatibility fixtures for Codex/Claude installed runtimes and genuine no-runtime/config-only ancestors. Test custom instruction preservation, invalid include/read error, artifact hash mismatch, forged success booleans, and missing artifact. Add setup negative fixtures showing skills/instructions/registry are not distributed even if present in source, while config/rules/Pi/helpers are. Curator-owned/user custom links must remain byte/target-identical. No live provider login or model download needed for these gates.

Implementation validation in bounded standalone calls: package main tests; infra compose/prepare/setup/receipt tests; remaining infra tests split by Pi/runtime groups; attachments; modelharness; platform build checks and actual Windows script check on Windows. Do not call the whole suite passing if a subset or platform is untested. Count tested deprecation entrypoint×argument cases and report the actual ratio, naming Windows/live-host blind spots. Installation validation must not run the host setup script inside a managed session.

## README and release plan

Rewrite README around:

1. Purpose/ownership: residual setup, local Pi runtime, attachments and task-board contracts. Curator owns contexts, skills and managed proprietary-provider launches.
2. Quick start: bootstrap residual tools, settings/config/rules; separate explicit Curator migration command and auth prerequisites. Remove native codex shell-function wrapping advice.
3. Migration/deprecation: exact commands/messages/exit 1, no --print-config, removal next release, legacy installed paths preserved.
4. Residual setup/source resolution/doctor/verify (current README:9–303), revised asset list and honest legacy field meanings.
5. Contracts: child compose (:304), primary-session compose and prepare (:335), Session Manager wrappers (:380), compatibility renderer/registry exception and consumer migration prerequisite. Do not present task-board wrappers as deprecated direct launchers.
6. Keep Pi operator/runtime/catalog/profiles/targets/model-check sections (:429–1831); mark only proprietary executable aliases deprecated, retain canonical identifiers for compose.
7. Keep meta-config policy composition (:1832), but explain MCP registry compatibility rather than advertise ordinary launcher composition.
8. Keep tooling, optional PDF, config merge/settings (:2074,:2282,:2326), attachments (:2599), rules (:2638).
9. Replace Instructions/Skills/How It Works/Adding Skills/Updating Instructions (:2228,:2249,:2644,:2722,:2738) with Curator ownership and evidence-bounded migration; retain source provenance/attachment contract links. Keep unrelated research and repository sections if still accurate.

Update SKILL.md concurrently (especially :1–90, :750–913, :969): remove obsolete setup/fan-out/native launcher recommendations, document deprecation and compatibility exception, retain Pi/config/attachments/contracts. Although no longer auto-materialized as a skill, this file remains agent-facing guidance until its distribution is migrated.

No CHANGELOG or VERSION file is tracked in this snapshot. Add CHANGELOG.md with an Unreleased breaking-change/deprecation entry and explicit next-release removal. Version values are injected by scripts/setup.sh:91 and setup.ps1:46; do not hardcode a fabricated release in main.go. Tags visible locally end at v1.6.1; propose v2.0.0 for this behavior-breaking release, with alias deletion in the following release. Release owner chooses/publishes the actual tag; no tag/commit/install is part of this research.

## Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| `prepare` schema 1 accepted wire shape but rejected semantics | Preserve real compatibility artifacts; cross-consumer negative validation before removal. |
| Empty compose output silently loses configured MCP | Keep registry reader and missing-definition refusal; no fake empty success. |
| Managed Curator homes versus residual native config | Do not infer managed-home paths or rewrite Curator seeds; residual native settings/config do not automatically configure managed homes. Document this ownership boundary. |
| B5 auth/context/safari gaps | State real exit-1 auth failures and context parity unknown; no onboarding success claim for Claude/Pi prompt paths. |
| `.instructions` is still root-context byte source | Retain source bytes until provenance/re-cut workflow moves; remove distribution rather than blindly deleting content. |
| Repeated setup restores old ownership or deletes user paths | Explicit sync exclusions, non-mutation tests, no automatic home cleanup/takeover. |
| Receipt/source validator still expects removed skill files | Change sourceAssets and skill checks in the same slice as sync exclusions; preserve backend/Pi catalog checks. |
| Wrapper aliases bypass deprecation | Guard all execution dispatch paths before parsing, test compiled wrappers; leave pure compose usable. |
| Platform drift | Exercise Windows separately; do not treat Darwin tests as Windows verification. |
| Unreviewed policy deletion damages Pi/task-board | Keep project config schema/builders and existing provider-policy negative tests. |

## Verification and evidence honesty

Prior research-run evidence retained from the original report (not rerun during Rework 1):

- `go list ./...` in tools/agents-infra — exit **0**, 5 local packages.
- `go test ./internal/infra -run 'TestBuildChildLaunchComposition|TestPreparePrimarySession' -count=1` — exit **0**. Existing contract behavior only, not proposed migration behavior.
- `go test . -run 'TestRunPrepare|TestRunComposePrimarySession|TestRunComposeRejectsUnknownMode' -count=1` — exit **0**. Existing CLI contract subset only.
- `git rev-parse HEAD` — exit **0**, source pin above; `git status --porcelain` — exit **0**, empty at research checkpoint.
- B5 read with numbered lines; task-board consumer source inspected directly. All B5 launches/install results are accepted prior evidence, not rerun here.

Report structure validation (standalone Python assertions) — exit **0**, 26/26 required topic markers present, artifact below 60,000 bytes; this is a document check, not a behavioral gate. Final `git status --porcelain` — exit **0**, empty.

No full suite, real provider launch, installer, Windows runtime validation, external availability check, or implementation test was run. Repository reads included an initial wrong board query (`task(...)`, exit 1, repaired with get) and a nonexistent task-board source glob (exit 1, repaired by locating skill-project-management); neither is a validation pass. No current-host doctor result is claimed.

## Research logbook — 2026-09-16

- FINDING: v1 task-board preparation validator requires instruction artifacts, so literal deletion of every renderer conflicts with the preservation requirement. Recommend safe ownership split now, coordinated contract migration for full deletion.
- FINDING: LLDB wrapper absent at 459742e; preserve caller-supplied stdio support, do not reintroduce unsupported installation based on brief wording.
- EVIDENCE LIMIT: B5 managed homes/skills/MCP current does not establish default instruction injection parity; Claude/Pi prompt auth checks are red.
- DECISION: direct provider aliases and print-config become explicit exit-1 stubs, while compose/prepare builders remain live compatibility APIs.
- STORAGE: this task-scoped logbook section and board note substitute for editing repository LOGBOOK.md because the brief explicitly requires read-only repository work and /tmp report storage. No standalone logbook CLI is installed on PATH.


## Rework 1 — source corrections, 2026-09-16

- Added all four omitted runtime dispatcher subcommands as KEEP: runtime quarantine, runtime unquarantine, runtime broker and runtime runtime-launch (main.go:854–885). Retain existing Pi shared-runtime tests: internal/infra/pi_shared_supervision_test.go:107 (manual quarantine), pi_shared_broker_admission_test.go:323 (broker production entry), and pi_shared_launcher_test.go:317–477 (launcher production refusals). Add dispatcher negative cases for missing required broker/launcher arguments and unexpected quarantine positional arguments when implementation changes that surface.
- Corrected both local-shim references to `<project>/.local/bin/codex-local`; verified LocalLayout BinDir at infra.go:131 and actual filename at :1418. The rework brief abbreviates the basename as codex, but source and reviewer verdict establish codex-local. The setup lifecycle and tests above now address that actual installed surface.
- Fact-checking: numbered dispatcher/Layout source reads and shim implementation read exited 0. No Go tests run in this report-only correction, as explicitly instructed; prior test evidence above is accepted historical evidence only. Reviewer full-suite attempt exited 143 (terminated), not a green result.
- Logbook: both P2 corrections addressed without repository edits; compatibility contracts, goldens, release policy and Curator evidence conclusions remain unchanged. This section records the brief/source filename discrepancy for implementation and review.
