# Vendor / Environment / Model Target Contract

Status: proposed architecture for review  
Task: `TASK-260824-3rl3ws`  
Parent Story: `STORY-260824-1yr6m0`

## 1. Requirements and decisions

- **R1 — Separated identity.** A target has an explicit vendor, agent
  environment (harness), model, reasoning level, and optional endpoint/profile
  coordinates.
- **R2 — Required targets.** The supported v1 tuples are OpenAI / Codex /
  `gpt-5.6-sol` / `high`; Anthropic / Claude Code / `claude-opus-5` / `high`;
  and Qwen / Pi / a managed local `Qwen3.8-27B-MLX-8bit` profile / `off`.
- **R3 — Required entrypoints.** `openai-infra`, `anthropic-infra`, and
  `qwen-infra` are installed aliases whose target selection comes from project
  configuration. The wrappers contain only their entrypoint name and sibling
  `agents-infra` location; they do not contain a vendor, harness, model,
  reasoning level, profile, or endpoint.
- **R4 — Compatibility.** Existing `[agents.codex]`, `[agents.claude]`, and
  `[agents.pi]` configurations and the direct `agents-infra codex|claude|pi`
  commands keep their current behavior.
- **R5 — Diagnostics.** Alias `--print-config` and primary-session `compose`
  expose the selected target and provenance without launching a provider.
- **R6 — Fail closed.** Missing, malformed, ambiguous, cross-vendor, or
  identity-divergent target configuration is an error and never becomes legacy
  fallback or policy absence. The runtime reports precise, actionable,
  non-secret startup diagnostics and never repairs or rewrites configuration.

Decision: canonical targets are an additive strict launch path. They do not
silently become defaults for direct legacy launchers. This preserves existing
behavior while allowing explicit migration one entrypoint at a time.

## 2. Canonical TOML schema

Canonical policy lives in every discovered
`.agents/.configs/project-config.toml` under two new tables:

```toml
[agents.targets."openai-sol-high"]
vendor = "openai"
environment = "codex"
model = "gpt-5.6-sol"
reasoning = "high"

[agents.targets."anthropic-opus-high"]
vendor = "anthropic"
environment = "claude-code"
model = "claude-opus-5"
reasoning = "high"

[agents.targets."qwen-mlx-8bit"]
vendor = "qwen"
environment = "pi"
model = "Qwen3.8-27B-MLX-8bit"
reasoning = "off"
profile = "qwen-3.8-27b-mlx-8bit"
profile_provider = "local-qwen"
endpoint = "http://127.0.0.1:18011/v1"

[agents.entrypoints]
"openai-infra" = "openai-sol-high"
"anthropic-infra" = "anthropic-opus-high"
"qwen-infra" = "qwen-mlx-8bit"
```

The Qwen target reuses, rather than duplicates, the existing complete managed
Pi profile:

```toml
[agents.pi.primary_session]
pi_compatibility = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65"

[agents.pi.profiles."qwen-3.8-27b-mlx-8bit"]
provider = "local-qwen"
model = "Qwen3.8-27B-MLX-8bit"
base_url = "http://127.0.0.1:18011/v1"
api = "openai-completions"
reasoning = false
thinking = "off"
# The remaining existing managed Pi profile and runtime fields are required.
```

### 2.1 Target names

`<target-name>` is a non-empty TOML key. Identity is the exact decoded UTF-8
string: no trimming, case folding, Unicode normalization, or path use. A target
definition is atomic; fields from definitions of the same name in different
ancestor configs never merge.

### 2.2 Target fields

Every `[agents.targets.<target-name>]` accepts exactly these fields:

| Field | Required | Value domain | Meaning |
| --- | --- | --- | --- |
| `vendor` | yes | `openai`, `anthropic`, `qwen` | Product/API vendor identity. |
| `environment` | yes | `codex`, `claude-code`, `pi` | Agent harness used to launch the target. |
| `model` | yes | non-empty exact string | Provider/profile model identity. Whitespace-only is invalid. |
| `reasoning` | yes | see matrix below | Harness-native reasoning/thinking selection. |
| `profile` | conditional | non-empty exact UTF-8 string | Existing managed Pi profile identity. |
| `profile_provider` | no | non-empty exact UTF-8 string | Assertion over the selected managed Pi profile's operator-defined provider namespace. |
| `endpoint` | no | validated absolute URL | Assertion over the selected managed Pi profile endpoint. |

Unknown fields fail with `invalid_project_configuration` and the full field
path. No target field contains secrets.

### 2.3 Admitted v1 tuples

| Vendor | Environment | Reasoning domain | Profile | Profile provider / endpoint |
| --- | --- | --- | --- | --- |
| `openai` | `codex` | non-empty exact string; provider-owned | forbidden | forbidden |
| `anthropic` | `claude-code` | `low`, `medium`, `high`, `xhigh`, `max` | forbidden | forbidden |
| `qwen` | `pi` | `off`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max` | required | optional assertions |

No other vendor/environment pair is admitted in v1. Provider and model
availability remain provider/runtime facts; the target resolver validates the
schema and tuple, not a remote catalog. Codex reasoning is deliberately open:
the current legacy Codex policy accepts any non-empty string and Codex owns
model/effort compatibility, so the canonical path must not lose that
expressiveness or require an agents-infra release for a new Codex effort.

Claude reasoning is deliberately closed to the values that the current
composer can prove effective. Claude accepts an unknown `--effort` token but
ignores it and applies a provider-native default; such a token cannot satisfy a
strict target identity. The legacy Claude path retains that native-fallback
behavior. Pi reasoning is closed because agents-infra owns and validates the
documented Pi thinking-level domain.

For `qwen` / `pi`:

- `profile` selects an existing atomic `[agents.pi.profiles.<profile>]`.
- The profile must have `api = "openai-completions"`.
- Profile `provider` is an operator-defined Pi catalog namespace, not evidence
  of canonical vendor identity. No literal such as `local-qwen` is inferred or
  required. When target `profile_provider` is present, it must exactly equal
  profile `provider`; when absent, the referenced atomic profile supplies the
  effective value.
- Target `model` must exactly equal profile `model`.
- Target `reasoning` must exactly equal profile `thinking`.
- When target `endpoint` is present, it must exactly equal profile `base_url`.
- The effective endpoint is always profile `base_url`, even when the target
  omits the assertion.
- The existing managed Pi validator remains authoritative for the literal
  loopback `/v1` URL, runtime host/port agreement, runtime identity, profile
  completeness, and lifecycle safety.

### 2.4 Entrypoint fields

`[agents.entrypoints]` accepts exactly three scalar string keys in v1:

| Key | Target vendor required |
| --- | --- |
| `openai-infra` | `openai` |
| `anthropic-infra` | `anthropic` |
| `qwen-infra` | `qwen` |

Each value is an exact target name. Unknown keys, non-string/empty values,
unknown targets, or alias/vendor disagreement fail closed. The resolver never
chooses among targets by vendor and never infers an entrypoint mapping from a
legacy table.

## 3. Composition and precedence

Project configs are read root-to-cwd as today.

1. A nearer complete target definition atomically replaces the same target
   name from an ancestor.
2. A nearer explicit entrypoint mapping replaces the same alias mapping.
3. The selected mapping resolves its named target from the composed target map.
4. The target's environment selects the existing Codex, Claude, or Pi launch
   composer.
5. Target identity is applied above matching legacy primary-session selection.
   Unrelated legacy policy still applies: MCP and yolo/permission policy for
   Codex/Claude, and `pi_compatibility` plus the referenced profile definition
   for Pi.
6. Alias invocations are identity-locked. Explicit model, reasoning/thinking,
   profile, endpoint, or provider arguments may repeat the resolved value, but
   a conflicting value fails with `target_identity_conflict`; it does not
   override the configured target. For Pi, the lock is applied to the decoded
   `--model` grammar, not merely to its base string: `model`,
   `provider/model`, `model:thinking`, and `provider/model:thinking` are split
   into model, optional provider, and optional thinking coordinates. The model
   must equal target `model`, an explicit provider component must equal the
   selected profile's effective `provider`, and an explicit thinking suffix
   must equal target `reasoning`. A matching base with a divergent suffix is a
   conflict and is rejected before the legacy Pi composer can normalize it or
   let it override effective thinking. For Codex, `--model`,
   `--model-reasoning-effort`, `-c model=...`, and
   `-c model_reasoning_effort=...` are all identity selectors: exact repeats
   are accepted and divergent values fail. A canonical hosted target has no
   profile coordinate, so Codex `--profile` and `-c profile=...` always fail
   with `target_identity_conflict` on an alias invocation instead of selecting
   or asserting a profile.
7. Other provider-native arguments retain their existing parser, precedence,
   operand-boundary, and forwarding behavior.

The direct commands remain a separate compatibility path:

- `agents-infra codex` and `codex-local` retain explicit CLI/profile >
  `[agents.codex.primary_session]` > Codex-native precedence.
- `agents-infra claude` retains explicit CLI >
  `[agents.claude.primary_session]` > Claude-native precedence.
- `agents-infra pi` and `pi-infra` retain explicit `--profile` >
  `[agents.pi.primary_session].profile`, with existing managed-profile rules.
- Merely declaring targets or entrypoint mappings does not change those direct
  commands.

A malformed or unreadable contributing config, invalid target, missing target,
or invalid referenced Pi profile is a read/configuration failure, not target
absence. No legacy fallback is permitted after an alias was invoked.

## 4. Dispatch and installed aliases

The public dispatch operation is:

```text
agents-infra target <entrypoint-name> [--print-config] [-- provider-args...]
```

Each installed alias preserves caller cwd and argv and delegates only to its
exact sibling target:

```text
openai-infra     -> sibling agents-infra target openai-infra
anthropic-infra  -> sibling agents-infra target anthropic-infra
qwen-infra       -> sibling agents-infra target qwen-infra
```

The aliases never fall back through `PATH`. Setup repairs missing, changed,
non-regular, or non-executable aliases; setup postconditions and `verify`
reject remaining drift. An alias resolves configuration before provider/runtime
side effects. `--print-config` is non-launching.

For machine consumers, primary-session compose accepts exactly one selector:

```text
agents-infra compose --mode primary-session \
  --entrypoint openai-infra --project DIR --schema-version 1 --json \
  [-- provider-args...]
```

`--agent` remains supported for legacy composition. Supplying both `--agent`
and `--entrypoint`, or neither, is an error. Child MCP-only composition remains
`--agent codex|claude` and does not accept `--entrypoint`.

## 5. Print and compose contract

Alias plans keep the existing
`agents-infra.primary-session-launch-plan` schema version 1. Existing fields
retain their meanings. In particular, top-level `provider` remains the launch
implementation key `codex`, `claude`, or `pi` for existing consumers.

Alias plans add a `target` object:

```json
{
  "target": {
    "entrypoint": "openai-infra",
    "entrypoint_source": "/project/.agents/.configs/project-config.toml",
    "name": "openai-sol-high",
    "source": "/project/.agents/.configs/project-config.toml",
    "vendor": "openai",
    "environment": "codex",
    "model": "gpt-5.6-sol",
    "reasoning": "high",
    "profile": null,
    "profile_provider": null,
    "endpoint": null
  }
}
```

For Qwen, `target.model` is the unqualified exact profile model ID,
`target.profile` is populated, and configured `target.profile_provider` and
`target.endpoint` assertions are populated or null. Existing
`resolved.model` retains its current Pi meaning and is provider-qualified:
`<profile.provider>/<profile.model>`. Alias plans additionally expose
`resolved.profile_provider` and `resolved.endpoint`; both use the selected
profile-definition source. The required Pi invariants are:

```text
target.model == selected_profile.model
resolved.profile_provider.value == selected_profile.provider
resolved.model.value == resolved.profile_provider.value + "/" + target.model
resolved.endpoint.value == selected_profile.base_url
resolved.endpoint.value == pi.runtime.endpoint
```

`resolved.reasoning` and `resolved.profile` retain their existing Pi meanings.
Target assertions retain target-definition provenance while effective Pi
coordinates retain profile-definition provenance. For hosted alias plans,
resolved model/reasoning sources are the target definition.
`resolved.profile_provider` and `resolved.endpoint` are Pi-only and use
`not_applicable`. `resolved.profile` keeps its existing per-provider meaning:
Claude uses `not_applicable`; Codex uses an explicit CLI source when a profile
is selected explicitly and `native` when Codex configuration decides. Because
canonical Codex alias invocations reject `--profile` and `-c profile=...`, an
alias plan reports the Codex profile coordinate as `native`, with no inferred
profile value. Legacy Codex `--agent` plans continue to report an accepted
explicit profile with its existing CLI provenance.

The `target` object and the alias-only `resolved.profile_provider` and
`resolved.endpoint` fields are emitted only for `--entrypoint`/vendor-alias
plans. Legacy `--agent` plans do not gain those fields. This preserves legacy
schema-v1 bytes while making the Qwen alias identity machine-checkable without
changing the existing Pi `resolved.model` representation or duplicating its
runtime endpoint fact.

Human `--print-config` renders the same information before the existing
provider-specific launch plan: entrypoint and source, target name and source,
vendor, environment, model, reasoning, profile, profile-provider assertion,
endpoint, effective launch provider, and provider argv. Secret values and
arbitrary environment values remain excluded.

Legacy `--agent` compose and direct `--print-config` output omit `target`; their
existing JSON/text remains backward compatible. Errors use the existing error
envelope with stable additions `unknown_entrypoint`, `unknown_target`,
`invalid_target`, and `target_identity_conflict` as applicable.

Every canonical-target error occurs before provider/runtime side effects and is
actionable on both output surfaces. Human diagnostics name the stable code, the
contributing config path, the exact dotted field path or entrypoint/target/
profile identity, and the corrective action. JSON errors retain the existing
envelope and may add safe `error.context` members (`entrypoint`, `target`,
`profile`, `field`, `source`) plus `error.remediation`; irrelevant members are
omitted. Diagnostics never include secret values, arbitrary TOML values, or
environment values. A read failure names the unreadable source and never uses
the remediation for an absent mapping.

## 6. Migration and compatibility

- There is no automatic rewrite, deletion, or semantic migration of legacy
  tables. Parse, compose, print, target dispatch, setup, verify, and launch are
  read/validate operations with respect to project config; none edits it.
- A legacy-only project remains valid and all existing direct commands behave
  as before.
- Vendor aliases are opt-in by configuration. Invoking an installed but
  unmapped alias fails `unknown_entrypoint`; it does not guess from legacy
  state or hardcode a target.
- Migration is additive: define canonical targets, map the three entrypoints,
  keep legacy tables for direct launch policy, and reuse the existing complete
  Pi profile by name. Its existing free-form `provider` label does not need to
  change; copy it into optional target `profile_provider` only when an explicit
  assertion is wanted.
- Canonical Codex targets retain the legacy non-empty-string reasoning domain.
  Canonical Claude targets accept only reasoning values whose effectiveness the
  current composer can establish; a legacy Claude invocation with an unknown
  effort remains valid and reports provider-native fallback, but cannot be
  migrated to a strict alias until that effort becomes recognized.
- Once alias diagnostics and deployment smokes pass, an operator may remove
  redundant hosted `model`/`reasoning` legacy fields in a separate reviewed
  edit. This is optional; leaving them is supported.
- `agents-infra doctor` reports canonical mappings and their resolved target
  coordinates with sources, while continuing to report legacy provider tables.

The separately tracked rollout is an operator action, not product/runtime
migration. `TASK-260824-1jjze0` may build and run a task-scoped one-time script
that inventories the agreed local roots, dry-runs, preserves each MCP section,
rewrites the remaining agent configuration to this contract, and validates
every changed file. That script is not installed by agents-infra, is not called
by setup/verify/compose/launch, and is not a fallback from a startup error.

## 7. Required negative evidence

Implementation tests must drive the production parse, compose, alias, setup,
verify, and launch entrypoints as applicable. Positive-only helper tests do not
establish the refusal contract.

Required negatives:

1. Every non-admitted vendor/environment cross-pair is refused.
2. Missing/empty/unknown fields and wrong TOML types are refused with the field
   path; unknown target fields are refused. An empty Codex reasoning string and
   out-of-domain Claude or Pi reasoning values are refused; a new non-empty
   Codex reasoning token is not locally rejected merely because agents-infra
   does not enumerate it.
3. Missing mapping, unknown target reference, alias/vendor mismatch, multiple
   inferred candidates, and simultaneous compose `--agent`/`--entrypoint` are
   refused without inference or fallback.
4. Hosted targets refuse `profile` and `endpoint`; Pi targets refuse missing or
   unknown profiles.
5. Qwen targets refuse non-OpenAI-compatible profile APIs and
   model/reasoning/profile-provider/endpoint assertion mismatches against the
   selected profile. A valid profile with a non-`local-qwen` provider label is
   accepted when the target omits that assertion or states the same label.
6. Conflicting explicit identity CLI selectors are refused; equal repeats are
   accepted and normalized. Pi `model:thinking` and
   `provider/model:thinking` tokens whose model matches but whose thinking
   suffix differs from target reasoning are refused; a provider-qualified token
   with a divergent provider is also refused. Codex aliases reject
   `--profile` and `-c profile=...`; their `--model`,
   `--model-reasoning-effort`, `-c model=...`, and
   `-c model_reasoning_effort=...` forms are each tested for an accepted exact
   repeat and a refused divergence.
7. A malformed, unreadable, or partially read canonical config never falls
   back to legacy configuration or provider-native launch. Runtime/setup/
   compose/verify leave project-config bytes unchanged and report the failing
   source and field plus a corrective action; an unreadable source is not
   reported as an absent mapping.
8. Installed aliases reach production target dispatch, preserve cwd/argv, and
   refuse a missing or non-regular sibling; narrowing verification to one alias
   must fail the other alias cases.
9. An unconfigured alias cannot launch even when a complete matching legacy
   table exists; this proves there is no hidden hardcoded/legacy bypass.
10. Legacy direct launch tests prove canonical target declarations do not alter
    existing Codex, Claude, or Pi precedence, including Claude's unknown-effort
    provider-native fallback.
11. Qwen alias compose proves the unqualified target model, effective Pi
    provider-qualified `resolved.model`, `resolved.profile_provider`, and both
    copies of the effective endpoint satisfy the Section 5 invariants. A
    narrowing mutant that sources either effective field from the target
    assertion instead of the selected profile must fail.
12. Startup-error tests assert stable code plus safe actionable context for a
    missing mapping, unknown target, malformed field, unreadable source, and
    identity conflict; each case proves no provider/runtime side effect and no
    project-config rewrite. Narrowing diagnostics to a generic code without the
    failing source/identity or remediation must fail.

## 8. Development decomposition and traceability

The current four-task Story is the smallest proportional decomposition after
the explicit rollout requirement to rewrite all local project target configs
was added:

| Board element | Deliverable | Requirements |
| --- | --- | --- |
| `TASK-260824-3rl3ws` | This architecture contract | R1-R6; resolves legacy/Pi/alias choices. |
| `TASK-260824-2o4zq8` | Parser, resolver, composition, aliases, diagnostics, setup/verify, tests, docs | Sections 2-7; R1, R3-R6. |
| `TASK-260824-1jjze0` | One-time recursive rewrite of local project configs, preserving each MCP section | Rollout requirement "rewrite all local project target configs"; R2-R3, R5-R6; Sections 2, 5-7. |
| `TASK-260824-2a4gk3` | Concrete casual-talks target configuration and real Qwen deployment smoke | Section 2 example and Section 7 deployment evidence; R2, R3, R5-R6. |

Dependencies are `3rl3ws -> 2o4zq8 -> 1jjze0 -> 2a4gk3`; deployment also keeps
its explicit implementation prerequisite. No extra Story, research,
documentation, diagram, or quality-gate task is justified: the architecture
question is resolved here, implementation includes its own docs and negative
tests, the rewrite task owns only the explicitly requested one-time rollout,
and deployment owns its live smoke.

### Gap and out-of-scope audit

The owner requirements, Story description/scope/AC, task acceptance criteria,
and the managed Pi operator contract's explicit exclusions were checked.
`TASK-260824-1jjze0` is beyond literal R1-R6 but is justified by the explicit
rollout requirement named in that task: without it, the requested replacement
of all in-scope local project agent configuration would not occur. It closes
that gap with a task-scoped, inventory-first, reversible one-time rewrite while
preserving MCP bytes. Section 6 and R4 were checked: the task does not add an
automatic agents-infra migration and therefore does not weaken additive legacy
compatibility. The managed Pi exclusions were checked: it does not acquire or
convert models, add provider adapters, automate benchmarks, introduce cloud
sync, or claim runtime attestation. No other beyond-literal element and no
unresolved research question is justified.
