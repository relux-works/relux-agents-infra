# TASK-260830-12n20p — agents-infra vs agents-management contract audit

## Scope and baseline

- `agents-infra`: `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` (Story worktree, zero commits behind local `main`).
- `skill-agents-management`: shipped tag `v0.3.0` at `3bec0ba`. Current `main` is `c5ee9f6`; the post-tag code delta is the local-runtime status consumer, not a change to the System/Vendor/BuildPlan shapes audited here.
- The assignment brief describes the older `v0.2.0` shape. `v0.3.0` already ships **seven** system plugins (`antigravity`, `claude-code`, `codex`, `gemini-cli`, `muse`, `pi`, `qwen-code`) and **five** vendor plugins (`alibaba`, `anthropic`, `google`, `local-models`, `openai`). The `pi` system and `local-models` vendor landed in `v0.3.0`; they are not future work anymore (`skill-agents-management@v0.3.0:docs/architecture.md:15-53,183-223`; `pkg/agentic/systems/pi/pi.go:37-68`; `pkg/vendorplugin/vendors/local-models/vendor.go:8-59`).

All agents-management citations below are to `v0.3.0`, and all agents-infra citations are to the worktree commit above.

## Verdict map

Legend: **covered** = the shipped contract already owns the reusable fact; **contract change** = the reusable fact cannot be represented without changing skill-agents-management at source; **stays here** = workstation/product/process ownership genuinely belongs to agents-infra.

| agents-infra behavior | Verdict | Evidence in agents-infra | Evidence in shipped agents-management | Required ownership/action |
| --- | --- | --- | --- | --- |
| Hard-coded agentic-system/vendor/model identity and pairing in `[agents.targets]` | **covered** for reusable identity/admission; the target-file UX stays here | `project_config.go:36-49,269-321,380-420` declares vendor/environment/model/reasoning and hard-codes only `openai×codex`, `anthropic×claude-code`, `qwen×pi`. | `pkg/agentic/system.go:36-84,434-537` owns normalized System IDs/capabilities; `pkg/vendorplugin/registry.go:208-279,316-358` validates vendor→system dependencies and runtime declarations; `pkg/vendorplugin/spawn.go:128-190` resolves runtime/model/effort and builds the plan. | agents-infra should eventually consume the registry rather than keep a second reusable pair/model validator. TOML discovery, target names, source provenance and entrypoint mapping remain agents-infra. |
| Model selection and reasoning-effort validation | **covered** | `project_config.go:297-301,399-409` validates model/reasoning locally; `canonical_target.go:232-250,293-333` locks CLI args to configured identity. | `pkg/vendorplugin/vendor.go:457-499,534-660` owns per-model effort vocabularies/system membership; `pkg/vendorplugin/spawn.go:137-190` selects and validates model/effort before dispatch; `pkg/agentic/plan.go:77-146` refuses unsupported/missing transport. | Move reusable model/effort admission to skill-agents-management consumption. Do **not** equate Pi `thinking` with vendor reasoning effort: agents-infra checks Pi profile `Thinking`, while the shipped Pi system declares `EffortTransportNone` (`pkg/agentic/systems/pi/pi.go:70-90`). Pi thinking remains a profile/harness option until a distinct typed contract exists. |
| Pi as an agentic system / Process-A wrapper | **covered in v0.3.0** | Production entrypoint is `main.go:475-509`; it calls `RunPi`, whose config/profile selection and wrapper launch begin at `pi_launch_posix.go:47-150`. | `pkg/agentic/systems/pi/pi.go:1-26,37-137` is the real System plugin; `binary.go:7-17` resolves `agents-infra`; `args.go:10-40` emits `pi --profile <profile> --`; `env.go:7-17` carries caller CWD; `stdin.go:11-29` carries the assignment. | No new System-contract change is needed to express Pi. Downstream work should consume the shipped plugin, not re-create it. The still-unpinned Pi turn grammar is explicitly open in the shipped plugin (`args.go:22-34`), not something to guess here. |
| Local-model vendor catalog, runtime/model→agents-infra project/profile pointer, availability and launch preflight | **covered in v0.3.0** | agents-infra profiles supply provider/model/endpoint/runtime data and build Pi catalog/sidecar facts (`pi_plan.go:17-72,95-116,119-237`). | `pkg/vendorplugin/vendors/local-models/config.go:30-107,128-177` provides typed absent/malformed/valid config and project/profile pointers; `vendor.go:8-59` is the vendor; `pkg/agentic/systems/pi/preflight.go:17-74` reads status and fails closed; `pkg/vendorplugin/spawn.go:174-190` runs generic Preflightable before BuildPlan. | The reusable plugin-plane catalog/pointer/status read belongs in skill-agents-management and is already shipped. Runtime process ownership remains agents-infra (explicitly stated by `pkg/agentic/systems/pi/pi.go:6-12` and `docs/architecture.md:199-206`). |
| Basic observable launch value: one binary, argv, env, stdin, workdir and home | **covered** | Primary plans expose executable/argv/project and resolved fields (`primary_session_launch_plan.go:24-47,71-155`); actual hosted launch uses the plan at `main.go:710-718`. | `pkg/agentic/plan.go:17-26,77-186` builds exactly binary/argv/env/stdin/workdir/home and enforces capabilities; `pkg/agentic/system.go:323-434` supplies typed launch inputs. | Use BuildLaunch/BuildPlan for the reusable single-process harness plan. Process creation remains in agents-infra/consumer. |
| MCP composition grammar for Codex/Claude | **covered** for grammar validation and plan transport | `child_launch_composition.go:147-197` translates enabled server definitions into Codex TOML pairs or Claude `--mcp-config` JSON. | `pkg/agentic/system.go:221-268,323-405,434-537` carries Composition and delegates validation; Codex declares TOML grammar (`pkg/agentic/systems/codex/codex.go:92-114`) and Claude declares MCP JSON (`pkg/agentic/systems/claude/claude.go:100-133`). | Grammar ownership is already in system plugins. |
| Discovering hierarchical project config and MCP registries; enabled-order merge; definition/source provenance; required env-name list; versioned JSON success/error envelopes | **stays here** | `child_launch_composition.go:11-70,94-197` defines the external schema, canonicalizes project dir, reads ancestor/global sources, resolves definitions, records provenance/env names and emits agent-specific prefix; `main.go:721-785` enforces schema/version and emits error envelopes. | `agentic.Composition` contains only prefix plus name/transport/bearer metadata (`pkg/agentic/system.go:221-268`), and BuildPlan only validates a composition already supplied (`pkg/agentic/plan.go:137-151`). It has no file discovery, source paths, enabled-by provenance, producer version or JSON envelope. | No contract change is required: these are agents-infra's configuration/distribution API around the plugin plan. Keep them here and feed the resolved Composition into the system plugin. |
| Primary-session policy precedence, provenance, interactive plan, and total managed host/client split (Codex app-server vs Claude/Pi PTY) | **contract change** for the reusable launch-plan shape; source discovery/policy stays here | `primary_session_launch_plan.go:24-155,204-341,344-420` returns three variants plus field provenance and partitions every Codex token into host/client/session policy; `pi_plan.go:119-237` adds Pi PTY and sidecar/capability details. | `agentic.Plan` is one mode/one argv (`pkg/agentic/plan.go:17-26`); `LaunchMode` has exec/dry-run/managed-session but no simultaneous interactive/managed-host/managed-client variants (`pkg/agentic/system.go:86-130`). Claude intentionally does **not** declare managed-session (`pkg/agentic/systems/claude/claude.go:90-105`). | If the single plugin plane must own these reusable harness variants, extend skill-agents-management with typed named variants/host-client-policy partition (and Claude/Pi PTY support) at source. Do not bolt this onto agents-infra as another shadow adapter. Hierarchical policy precedence and provenance remain here. |
| Canonical aliases `openai-infra`, `anthropic-infra`, `qwen-infra` and their target lock | **stays here** | `project_config.go:30-49,324-347` defines the installed aliases; `canonical_target.go:91-102,156-196,255-333` resolves mappings and refuses identity drift; `main.go:669-718` launches them. | Runtime declarations model system×vendor pairs, not installed workstation aliases (`pkg/vendorplugin/runtime.go:81-142`; `registry.go:316-358`). | Keep alias naming, config ancestry and CLI-argument lock here; resolve the underlying runtime/model through agents-management. Important semantic mismatch: `qwen-infra` means **Pi + local Qwen profile**, while shipped runtime `qwen` means **qwen-code × alibaba** (`pkg/vendorplugin/runtime.go:321-324`). The alias must not be mapped to runtime `qwen`. |
| `vendor=qwen` in canonical target vs shipped local-model vendor identity | **covered underlying fact, but current names disagree** | `project_config.go:30-34,380-420` requires the user-facing vendor string `qwen` for `environment=pi`. | Shipped local Pi models are owned by vendor `local-models`, not `alibaba`/`qwen` (`pkg/vendorplugin/vendors/local-models/vendor.go:8-59`; `docs/architecture.md:44-53,183-206`). | Preserve `qwen` only as agents-infra product/entrypoint label or migrate config explicitly; plugin-plane vendor identity must be `local-models`. Never silently treat the two as the same VendorID. |
| Pi profile materialization: `models.json`, compaction settings, state paths, executable identity, endpoint/readiness, dFlash and capability evidence | **contract change** for a reusable local-model engine/sidecar node; file/process mechanics stay here | `pi_plan.go:17-116,119-237` builds catalog, state, engine plan, sidecars and capability fields; `pi_launch_posix.go:141-228,256-304` verifies identity, writes state/catalog, starts the model runtime and waits for readiness. | System/Vendor launch ends at one Process-A Plan (`pkg/agentic/plan.go:17-26`); the two-layer architecture only admits system and vendor nodes (`docs/architecture.md:13-82`). The local-model plugin explicitly delegates Process B to agents-infra (`docs/architecture.md:199-206`). | In skill-agents-management, add the future generic local-model/engine plugin kind and graph edges needed to describe the engine node and sidecar plan. Do not move start/stop/signal/readiness execution out of agents-infra merely to fit the graph. |
| Model-harness profile implicitly selects inference engine executable/argv, local vs SSH transport, stress and restart supervision | **contract change** for plugin identity/graph/typed engine plan; execution stays here | `internal/modelharness/config.go:26-82,98-170,203-280` expands a named profile into local executable/argv or SSH forwarding plus policies; `run.go:16-104` starts and supervises it. | A runtime is currently only `(agentic system × vendor)` (`pkg/vendorplugin/runtime.go:81-127`), vendor registration requires systems first (`pkg/vendorplugin/registry.go:208-260`), and `agentic.Plan` has no engine/sidecar/dependency node (`pkg/agentic/plan.go:17-26`). | This is the precise general-graph gap: skill-agents-management needs a third/general plugin node for local inference engines and declared dependency edges, plus a typed contribution to a multi-node launch plan. Keep OS process/SSH/supervision implementation in agents-infra unless a later ownership decision explicitly moves it. |
| Two-layer dependency direction | **contract change only for the future engine node; no existing agents-infra code depends on the old direction** | agents-infra does not import skill-agents-management and independently hard-codes targets/profiles (`project_config.go:36-49,380-420`), so no current code relies on vendor→system registration order. Its local engine is a profile/process resource used by Pi. | Vendor registration refuses any model naming an unregistered system (`pkg/vendorplugin/registry.go:208-260`); architecture makes vendor→system the only direction (`docs/architecture.md:44-53`). | A general graph will not break an agents-infra dependency because none exists. It must preserve the current vendor→system edge semantics while adding engine nodes/edges; otherwise the Pi system and local-model vendor have no non-cyclic place to reference the same Process-B kind. |
| Pi shared-runtime broker: election, authenticated lease protocol, path confinement, process identity, restart/backoff/quarantine, operator status/stop | **stays here** | `pi_shared_protocol.go:26-99,122-268` defines auth/version/digests/confined paths; `pi_shared_supervision.go:14-134` owns restart/quarantine state; production operator/launcher commands are `main.go:553-630`; process lifecycle is `pi_launch_posix.go:149-304` and shared-runtime files. | The shipped Pi plugin states it never starts/stops/signals Process B (`pkg/agentic/systems/pi/pi.go:6-12`); architecture repeats that this authority belongs entirely to agents-infra (`docs/architecture.md:199-206`) and places process-starting preflights/exec mechanics on the consumer side (`docs/architecture.md:234-257`). | Keep all broker lifecycle and OS attestation here. agents-management may read sanitized status through `pkg/localruntime`; it must not become the broker. |
| Pi standalone worker authorization/trust/deadline/state isolation | **stays here** | `pi_standalone.go:14-97` declares the allowlist and authorization/state contract; `main.go:512-550` owns deadline and CLI surface; `pi_launch_posix.go:77-88,123-139,152-167` applies separate policy and isolated state. | `LaunchRequest` has prompt/model/effort/profile/env/run/goal/budget/tier/composition but no workstation authorization/trust policy (`pkg/agentic/system.go:323-405`); architecture keeps roles/policy and process execution with the consumer (`docs/architecture.md:234-257`). | Keep security/product policy here. The Pi system plugin should receive only the already-authorized plan inputs. Add a shared contract later only if another consumer genuinely needs the same authorization semantics. |
| Bounded `agents-infra model-check` launch and behavior evidence | **covered** for the reusable runtime/model launch value and Pi preflight seam; checker orchestration **stays here**; **no contract-shape change** | The top-level dispatch and typed exit translation are `main.go:37-40,46-76,94-133`. `RunModelCheck` resolves the canonical target, forces the real JSON/no-session Pi invocation, requires managed Pi, starts `RunPi` under a deadline, captures raw output and evaluates it (`model_check.go:22-35,146-243,245-283`). It refuses overwrite and creates mode-`0600` artifacts (`model_check.go:304-327`), validates JSONL lifecycle/tool pairing (`model_check.go:330-441`), evaluates final-text/tool expectations and cleanup with typed exits (`model_check.go:517-576`), and emits bounded sanitized summaries (`model_check.go:618-713`). The built CLI is attacked at the production call site by missing tool/text, earlier non-final text, failed tool, malformed stream, readiness/run timeout, invalid deadline, overwrite and non-managed-target negatives (`model_check_main_test.go:90-350,441-468`). | Shipped `Plan` already represents binary/argv/env/stdin/workdir/home (`pkg/agentic/plan.go:9-26`), while `BuildLaunch` resolves the runtime/model/effort, runs optional non-dry-run `Preflightable`, then returns `BuildPlan` (`pkg/vendorplugin/spawn.go:103-190`); Pi supplies that preflight and fails closed on read failure or unattested state (`pkg/agentic/systems/pi/preflight.go:17-74`). The contract explicitly ends at a launch value and leaves process execution/fault injection to the consumer (`docs/architecture.md:234-257`). The current Pi plugin deliberately stops after the pinned wrapper prefix and leaves `<turn args>` open (`pkg/agentic/systems/pi/args.go:10-40`; `docs/architecture.md:217-223`). | Change **nothing in the skill-agents-management contract shape** for model-check: `Plan.Argv` can carry the exact checker invocation once the already-named Pi turn grammar is pinned. A later migration may complete that existing Pi plugin implementation/parity item; it must not move target restriction, deadline/cleanup, raw evidence, JSONL interpretation, expectations, sanitization, overwrite refusal or exit semantics out of agents-infra. `BuildPlan` should not become a test runner or evidence policy. |
| Installing/syncing instructions, symlinks, runtime receipts, preparing rendered project surfaces, then spawning a process | **stays here** | `infra.go:134-215` syncs/links/verifies installation; `main.go:450-472,710-718,827-879` prepares surfaces and starts processes. | agents-management explicitly ends at a launch value; consumers own processes and process-starting preparation (`docs/architecture.md:234-257`; `pkg/agentic/plan.go:17-26`). | No plugin-contract change. |

## What BuildPlan does not do

`agentic.BuildPlan` does: normalize/lookup one System, gate launch mode and effort transport, gate optional goal/budget/tier/composition, resolve one binary, build one argv/env/stdin value, and resolve home (`skill-agents-management@v0.3.0:pkg/agentic/plan.go:77-186`).

agents-infra launch composition additionally does all of the following:

1. canonical project and ancestor/global config discovery;
2. whole-target precedence and source provenance;
3. MCP registry discovery, enabled-order merge and required environment-name reporting;
4. producer/version/schema/status/error envelopes;
5. canonical alias→target resolution and conflict refusal;
6. simultaneous interactive, managed-host and managed-client variants with total token partition;
7. Pi catalog/settings/state/identity/sidecar/capability materialization;
8. process preparation and actual execution;
9. bounded model-behavior execution, raw-evidence persistence, JSONL lifecycle/tool interpretation, expectation evaluation, sanitization, cleanup verdicts and typed checker exits.

Items 1–5, 8 and 9 are agents-infra consumer/distribution responsibilities. Item 6 is a reusable System-plan contract gap. Item 7 splits: the reusable engine identity/sidecar graph is a skill-agents-management gap; state materialization and process lifecycle stay in agents-infra. `model-check` does not add another plan-shape gap: `Plan.Argv` is sufficient, while the exact Pi `<turn args>` remain an already-declared plugin implementation/parity open item rather than evidence-runner contract surface.

## Six originally shipped systems vs agents-infra

| v0.2 system plugin | agents-infra overlap | Disagreement / scheduling consequence |
| --- | --- | --- |
| `codex` | Direct primary launch, MCP composition, model/effort/yolo resolution and managed app-server split (`codex_launch.go`; `primary_session_launch_plan.go:275-420`). | System plugin covers Exec/DryRun/ManagedSession and the one-process plan (`pkg/agentic/systems/codex/codex.go:78-145`). agents-infra additionally has primary-session policy/provenance and host/client split; this is the named contract gap, not evidence the Codex plugin is missing. |
| `claude-code` | Direct primary launch, MCP JSON, model/effort/yolo resolution and manager-owned PTY (`claude_launch.go`; `primary_session_launch_plan.go:459-524`). | Shipped Claude plugin deliberately has no ManagedSession mode (`pkg/agentic/systems/claude/claude.go:90-105`); agents-infra's PTY variant therefore requires a contract extension if it moves into the shared plane. |
| `qwen-code` | None. | `qwen-infra` is not this plugin; it launches Pi against a local Qwen profile. Do not schedule a qwen-code migration based on the alias name. Shipped qwen-code uses stdin effort transport (`pkg/agentic/systems/qwen/qwen.go:88-130`). |
| `gemini-cli` | None. | No conflict observed because agents-infra exposes no Gemini launch/compose path (`main.go:46-83`; child compose only Codex/Claude at `main.go:740-743`). |
| `antigravity` | None. | No conflict observed; agents-infra has no Antigravity launch path. Its preflighted executable requirement stays a consumer preflight in the shipped contract (`pkg/agentic/systems/agy/agy.go:65-110`). |
| `muse` | No Muse agent harness; only Pi runtime dFlash/config may mention models. | No harness overlap. Shipped `muse` is a system-only runtime with unresolved vendor authority (`pkg/vendorplugin/runtime.go:339-344`), which must not be confused with an inference-engine executable/profile in agents-infra. |

The seventh `v0.3.0` system is `pi`; its overlap and ownership split are covered above.

## Precise source changes for the next contract task

Change **in skill-agents-management**:

1. Replace the fixed two-layer registry topology with a general plugin graph that retains the existing vendor→system dependency and validation semantics.
2. Add a generic local-model/inference-engine plugin kind (stable identity, capabilities/config schema, local/remote launch-plan contribution) so a named agents-infra/model-harness profile is no longer the implicit engine type.
3. Extend the plan model beyond one Process-A `Plan` to typed named variants/dependencies/sidecars, sufficient to represent interactive + managed host + managed client and a local-model engine sidecar without moving process execution.
4. Add corresponding fail-closed registration/graph/plan validation and negative tests at the real registry/BuildLaunch entry points (unknown dependency, cycle, wrong kind, missing engine, sidecar dropped, unsupported variant, and narrowing mutants).

`model-check` adds **no fifth contract change**. If a later consumer migration routes its launch through `BuildLaunch`, skill-agents-management must complete the already-named Pi turn-argument parity work in the existing Pi plugin (`pkg/agentic/systems/pi/args.go:10-40`); that is plugin implementation data carried by the existing `Plan.Argv`, not a new graph/plan/evidence abstraction.

Keep **in agents-infra**:

1. hierarchical `project-config.toml` discovery, canonical alias and source-provenance UX;
2. MCP registry discovery and versioned JSON composition/prepare envelopes;
3. setup/sync/symlink/runtime-receipt and project-surface preparation;
4. all process creation, PTY/app-server hosting, SSH invocation and supervision;
5. Pi state/catalog/settings materialization, executable/environment revalidation and readiness checks;
6. the entire shared-runtime broker/lease/attestation/restart/quarantine/operator plane;
7. standalone Pi authorization, trust and isolated state policy;
8. bounded `model-check` target policy, execution deadline and cleanup, raw artifact protection, lifecycle/tool parsing, expectations, sanitization and typed exit semantics.

## Answers to the schedule questions

- **Can Pi be expressed under the shipped contract?** Yes. It is already shipped as a real System plugin in `v0.3.0`, with generic Preflightable integration and a local-models vendor. No further Pi-specific System seam is required. Its real turn-argument grammar remains explicitly unpinned/open.
- **What does agents-infra composition do beyond BuildPlan?** Config/registry discovery, precedence/provenance, versioned external envelopes, canonical target locking, simultaneous managed variants, Pi sidecar/engine facts, preparation and process execution.
- **Would a general graph break an agents-infra dependency on two-layer direction?** No current dependency exists: agents-infra does not consume the registry. The graph must preserve vendor→system semantics and add engine nodes; that removes a representational block rather than breaking current code.
- **Which original six overlap?** Codex and Claude overlap directly; qwen-code, Gemini, Antigravity and Muse do not. The `qwen-infra` alias is Pi, not qwen-code. The new seventh plugin Pi overlaps directly and is already shipped.
- **Does `model-check` require a new `BuildPlan` contract?** No. Runtime/model selection and Pi preflight fit the shipped `BuildLaunch`/`Plan` seam; the exact Pi checker argv is blocked only by the already-open Pi turn-grammar parity item. Starting the managed lifecycle and judging its JSONL/tool/text/cleanup evidence are deliberately consumer responsibilities and stay in agents-infra.

## Top-level command census

The audit was re-enumerated from the exact `run` switch (`main.go:46-83`), not from remembered launch paths. Only `model-check` was absent from the prior launch map; it is now classified above. Every other top-level command maps as follows:

| Top-level command | Launch classification | Where this audit covers it |
| --- | --- | --- |
| `setup`, `refresh-links`, `doctor`, `verify` | No provider/agent launch; installation, link repair and diagnostics (`main.go:156-386`). | “Installing/syncing instructions…” stays here. |
| `compose` | Explicitly non-launching plan/config serialization (`main.go:721-825,882-915`). | MCP composition, primary-session plan and provenance rows. |
| `prepare` | Explicitly non-launching project-surface mutation (`main.go:827-879`). | “Installing/syncing instructions…” stays here. |
| `attachments` | No agent launch; delegates attachment intake to the attachments package (`main.go:136-153`). | Outside the agentic-system/vendor/launch map; named here so it is not mistaken for an omitted launch consumer. |
| `codex`, `claude` | Direct provider launches after preparation (`main.go:417-472`). | Primary-session policy/variant and preparation rows. |
| `pi` | Direct managed Pi or standalone worker launch (`main.go:475-550`). | Pi System, local-model, standalone policy and broker rows. |
| `runtime` | `status`/operator actions plus internal broker/runtime process launch (`main.go:553-630`). | Pi shared-runtime broker row. |
| `target` | Canonical alias resolution followed by Pi or provider launch (`main.go:669-718`). | Canonical alias/target-lock row. |
| `model-check` | Distinct managed launch **and** output evaluator (`main.go:94-133`). | New bounded model-check row above. |
| `version`, `--version`, `help`, `-h`, `--help` | Metadata/usage only (`main.go:77-80,918-920`). | No launch behavior. |

## Read-only discipline and verification note

No tracked file in either repository was changed. This report is a task-scoped scratch artifact for board attachment. Because the task changes no production behavior, no new tests are applicable; existing gate tests are validation evidence for the cited shipped contract, not deliverables of this audit.

Validation run directly, with real exit codes:

| Repository | Command | Exit | Result |
| --- | --- | ---: | --- |
| agents-infra | `go test ./internal/infra ./internal/modelharness` | 0 | PASS (`internal/infra` 516.851s; `internal/modelharness` 1.230s) |
| agents-infra | `go build ./...` | 0 | PASS |
| agents-infra | `go vet ./internal/infra ./internal/modelharness` | 0 | PASS |
| skill-agents-management | `go test ./pkg/agentic/... ./pkg/vendorplugin/...` | 0 | PASS, including all seven systems and local-models |
| skill-agents-management | `go build ./...` | 0 | PASS |
| skill-agents-management | `go vet ./pkg/agentic/... ./pkg/vendorplugin/...` | 0 | PASS |

Rework validation was then run directly against the production `model-check` entrypoint and the revised command census:

| Repository/artifact | Command | Exit | Result |
| --- | --- | ---: | --- |
| agents-infra | `go test . -run '^TestModelCheckProductionEntrypoint$' -count=1` | 0 | PASS; freshly built CLI exercised the complete positive and negative matrix in `model_check_main_test.go:90-350` (18.473s) |
| agents-infra | `go test ./internal/infra -run '^TestModelCheckCleanupAttestationRefusesUnconfirmedStates$' -count=1` | 0 | PASS; narrowed cleanup-attestation refusal (2.189s) |
| agents-infra | `go build ./...` | 0 | PASS |
| agents-infra | `go vet ./...` | 0 | PASS |
| agents-infra | `git diff --check` | 0 | PASS; repository worktree remains clean |
| revised audit | task-scoped Python assertion over all top-level command names plus the `model-check` ownership split | 0 | PASS (`audit-census-ok`) |
| revised audit | task-scoped Python trailing-whitespace assertion | 0 | PASS (`audit-markdown-ok`: 146 lines) |

Existing negative-evidence coverage exercised by those green suites (no new gate was introduced by this audit):

- Production `vendorplugin.BuildLaunch` (`pkg/vendorplugin/spawn.go:128-190`) is driven by redirect, missing/out-of-vocabulary effort, unknown model/system and unresolvable-runtime negatives in `pkg/vendorplugin/spawn_test.go:94-413`; the local Pi path is driven through the same entry point by preflight refusal/timeout tests in `pkg/vendorplugin/vendors/local-models/buildlaunch_test.go:94-160`.
- Production `agentic.BuildPlan` (`pkg/agentic/plan.go:77-186`) is driven by unknown system/mode, narrowed effort transport, unsupported parameters, composition and plugin-contract negatives in `pkg/agentic/plan_test.go:15-318`.
- Production canonical target parsing/locking (`project_config.go:269-420`; `canonical_target.go:293-333`) is attacked by cross-pair and divergent-selector negatives in `canonical_target_test.go:72,210`.
- Production standalone authorization (`RunPi` at `pi_launch_posix.go:47-150`) is driven by narrowed/invalid allowlist, malformed policy and caller-bypass negatives in `pi_standalone_test.go:82-225`.
- Production shared-runtime attestation/launcher paths are attacked by delete-and-narrow witnesses and absent/forged/bypass authorization inputs in `pi_shared_attestation_test.go:186-436` and `pi_shared_launcher_test.go:311-639`.

## Recovery-run verification

The successor run independently reread the shipped contract from a temporary archive of exact tag `v0.3.0` and re-enumerated all 18 selectors in the production `run` switch. It did not rely on the dirty `skill-agents-management` checkout or its README.

| Repository/artifact | Command | Exit | Result |
| --- | --- | ---: | --- |
| agents-infra | `go test . -run '^TestModelCheckProductionEntrypoint$' -count=1` | 0 | PASS; production CLI positive/negative behavior matrix. |
| agents-infra | `go test ./internal/infra -run '^TestModelCheckCleanupAttestationRefusesUnconfirmedStates$' -count=1` | 0 | PASS; cleanup attestation refusal. |
| agents-infra | `go build ./...` | 0 | PASS. |
| agents-infra | `go vet ./...` | 0 | PASS. |
| skill-agents-management `v0.3.0` archive | `go test ./pkg/agentic/... ./pkg/vendorplugin/... -count=1` | 0 | PASS; exact shipped contract, including `BuildPlan`, `BuildLaunch`, Pi preflight and their refusal tests. |
| skill-agents-management `v0.3.0` archive | `go build ./...` | 0 | PASS. |
| agents-infra | `go test ./internal/infra -run '^TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry$' -count=1` | 1 | FAIL twice: the 1s fixture did not observe the first 503 before its bound and the exit-after fixture timed out instead of observing early exit. This is the same unrelated full-suite failure recorded by Change Request revision 2; no code may be changed under this audit's read-only constraint. |

The red readiness test is reported as red and is not used as evidence for the `model-check` verdict. The production `model-check` entrypoint matrix and its cleanup-attestation negative are green; the audit changes no runtime behavior.
