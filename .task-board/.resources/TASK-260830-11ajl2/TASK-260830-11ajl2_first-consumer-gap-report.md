# TASK-260830-11ajl2 — first-consumer gap report (agents-infra → skill-agents-management)

**Status: BLOCKED.** The task cannot land as specified against
`skill-agents-management v0.5.0` without either contract additions that require a new
agents-management release — which the accepted lockstep plan gates behind a new
independent acceptance — or behaviour changes the task explicitly forbids.

No production code was changed. Baselines were measured; the dependency was probed in a
throwaway copy under `/tmp`, never in the worktree.

## Baseline (worktree `9db0526`, story branch, clean)

| Gate | Command | Exit |
| --- | --- | ---: |
| build | `GOWORK=off go build ./...` | 0 |
| vet | `GOWORK=off go vet ./...` | 0 |
| tests | `GOWORK=off go test ./... -count=1` | 0 |

Suite is green: `agents-infra` 103.560s, `internal/infra` 181.062s,
`internal/attachments` 3.131s, `internal/modelharness` 16.188s;
`cmd/model-harness` has no test files. Test-function counts:
88 / 357 / 13 / 24 = **482 total**.

## Mechanical facts, verified

Adding the dependency works and forces a Go floor bump:

```
$ go get github.com/relux-works/skill-agents-management@v0.5.0
go: upgraded go 1.21.0 => 1.25.5
go: added github.com/relux-works/skill-agents-management v0.5.0
```

A probe binary compiled against v0.5.0, blank-importing all seven system plugins, reports:

```
registered agentic systems: [antigravity claude-code codex gemini-cli muse pi qwen-code]
agents-infra admitted environments: [codex claude-code pi]
--- runtimes (system x vendor) ---
  runtime=agy      system=antigravity  vendor=google
  runtime=claude   system=claude-code  vendor=anthropic
  runtime=codex    system=codex        vendor=openai
  runtime=gemini   system=gemini-cli   vendor=google
  runtime=muse     system=muse         vendor=
  runtime=qwen     system=qwen-code    vendor=alibaba
  lookup runtime "pi" -> found=false
  lookup runtime "local-models" -> found=false
--- pi system ResolveBinary -> "" err=launchenv: "agents-infra" not found in the launch environment's PATH
```

Three things fall out of that output, and each one blocks a different piece of the task.

## Gap 1 — the shipped `pi` plugin describes agents-infra from the OUTSIDE

This is the structural finding, and it is not a gap a library patch closes.

- `pkg/agentic/systems/pi/binary.go` — `ResolveBinary` resolves **`agents-infra`** on the
  launch environment's PATH (proven by the probe's error text above).
- `pkg/agentic/systems/pi/args.go:10-40` — emits `pi --profile <profile> --`.
- `pkg/agentic/systems/pi/preflight.go` — reads status through
  `localruntime.NewCLIStatusReader()`, which shells out to
  `agents-infra runtime status --json` (`pkg/localruntime/client.go:110-124`).
- `pkg/vendorplugin/vendors/local-models/config.go:36-50` — its `Pointer` is an absolute
  path to an **agents-infra checkout** plus an `agents.pi.profiles.<name>` key inside that
  checkout's `project-config.toml`.

If agents-infra consumed these plugins to implement its own primary-session launch it
would re-exec itself, shell out to itself for readiness, and read a config file pointing
back at itself. The plugins are a caller-side description of how an external consumer
(task-board) drives agents-infra as a subprocess. The dependency direction is inverted for
this consumer, by design — `pkg/agentic/systems/pi/pi.go:6-12` and the library's
`docs/architecture.md:199-206` both state Process-B authority stays in agents-infra.

## Gap 2 — no producer-side composition contract (blocks the composition branch)

`child_launch_composition.go:186-197` is the hardcoded branch the task names:

```go
if agent == "codex" {
    composition.ArgvPrefix = append(composition.ArgvPrefix, codexMCPConfigArgs(server)...)
} else {
    claudeServers[name] = claudeMCPConfigServer(ClaudeMCPLaunchServer(server))
}
```

The library **validates** a composition prefix and never produces one:
`System.ValidateComposition(c Composition) error`, with `Composition{Prefix, Servers}`
documented as "already composed by the caller". codex's validator
(`pkg/agentic/systems/codex/composition.go`) and claude's
(`pkg/agentic/systems/claude/composition.go` → `internal/mcpjson.Validate`) are refusal
surfaces only.

So the *grammar* is owned by the plugins, but the *producer* is not expressible. Calling
`ValidateComposition` after building the prefix is additive validation that leaves the
branch in place — "merely bypassed", which the DoD rejects. Removing the branch needs a
producer-side composition contribution that v0.5.0 does not have.

This matches the audit's verdict for this surface: **stays here**
(`TASK-260830-12n20p_agents-infra-contract-audit.md`).

## Gap 3 — `agentic.Plan` cannot express the primary-session launch shape

`primary_session_launch_plan.go` returns three variants from one build: interactive argv,
managed-host argv (codex app-server), and managed-client argv, plus per-field provenance.

`agentic.Plan` has exactly one `Mode`, one `Binary`, one `Argv`. `Plan.Nodes` /
`BuildMultiNodePlan` model **dependency processes** (engine, sidecar) ordered
dependency-first — not alternative variants of one primary process. And claude declares
only `LaunchModeExec` + `LaunchModeDryRun`, with the omission documented deliberately in
`pkg/agentic/systems/claude/claude.go:90-105`: declaring `LaunchModeManagedSession`
"would be a capability claim with no construction behind it."

This is **audit change #3** — "extend the plan model beyond one Process-A `Plan` to typed
named variants/dependencies/sidecars". It is **not shipped in v0.5.0, and no task in
STORY-260830-37bq03 owns it.** The story shipped audit changes #1 (generalized graph,
TASK-260830-1jpse1) and #2 (inference-engine kind, TASK-260830-ter72z) only.

## Gap 4 — the target vocabulary has nothing to resolve to

`project_config.go:380-420` admits exactly three pairs: `openai/codex`,
`anthropic/claude-code`, `qwen/pi`. Mapping them onto the plugin plane:

| agents-infra pair | plugin-plane runtime | Available in v0.5.0? |
| --- | --- | --- |
| `openai` / `codex` | `codex` = codex × openai | yes |
| `anthropic` / `claude-code` | `claude` = claude-code × anthropic | yes |
| `qwen` / `pi` | would be `pi` × `local-models` | **no — neither runtime exists** |

The only `qwen` runtime is `qwen-code × alibaba`, and the audit is explicit that
`qwen-infra` (Pi + local Qwen profile) **must not** be mapped to it, and that agents-infra's
vendor label `qwen` must never be silently equated with plugin vendor `local-models`.

Declaring `pi × local-models` is exactly the work TASK-260830-y6infr owns, and it needs a
library release.

## Why DoD item 3 is false by construction at v0.5.0

> "Adding an environment afterwards requires no agents-infra change; prove with a narrowing mutant"

Two agents-infra switches must gain a case for any new environment, and neither is
expressible through v0.5.0:

- `canonical_target.go:417-427` `lockCanonicalTargetArguments` → per-environment CLI
  identity lock (`lockCodexTargetArguments` etc.). The audit assigns this to agents-infra
  ("keep alias naming, config ancestry and CLI-argument lock here").
- `primary_session_launch_plan.go:249-256` → per-environment `buildXPrimarySessionLaunchPlan`,
  which needs Gap 3's unshipped variant contract.

Worse, the obvious "generalization" is a **gate weakening**: driving the environment
admission gate off registry membership widens it from the 3 admitted environments to the 7
registered systems (proven above), admitting `gemini-cli`, `muse` and `antigravity` targets
for which agents-infra has no launcher at all. A narrowing mutant on that gate would not
survive review, and it is a behaviour change besides.

## Why "fix it in the library and release" is not available to me

The task's standing instruction is to fix contract gaps in `skill-agents-management` and
release. The accepted lockstep plan
(`.research/260830_agents-management-lockstep-release-and-rollback.md`, rev. 11/12) forbids
that without a new human acceptance:

- "**This plan must receive a new independent acceptance before any breaking change,
  release, installation or tag in this sequence.**"
- Its Decision is "Release task-board first, then agents-infra, while both compile **exact
  `v0.5.0`**." Every row of the compatibility matrix is keyed to exact `v0.5.0`.
- `v0.6.0` is reserved as a *conditional* breaking release, permitted "only after exact
  released-source and linked-binary inventories prove neither released consumer needs" the
  compatibility adapter — a condition about adapter removal, not about adding plan variants.

Closing Gaps 2, 3 and 4 requires an agents-management version that agents-infra then
compiles, which contradicts "both compile exact v0.5.0" and occupies or reorders reserved
versions. Revising an accepted plan is a human decision.

## Board-level blockers (independent of the above)

The mandated first command was **refused**:

```
$ task-board m 'set_status(TASK-260830-11ajl2, status=development)'
cannot set TASK-260830-11ajl2 to development — blocked by:
  TASK-260830-y6infr (status: development)
  TASK-260830-mf1xwk (status: backlog): blocked by unfinished dependencies
```

- **TASK-260830-y6infr** owns the pi-plugin consumption and the `pi × local-models`
  runtime that Gap 4 needs. Its spawn **failed**: `agent completed: exit=1`, codex
  provider limit exhausted, next retry **Sep 6**. It is parked in `development` while not
  progressing.
- **TASK-260830-mf1xwk** is listed as a blocker of this task, but this task's brief says it
  runs *before* the board migration ("which is exactly why this runs before the task-board
  migration rather than after"). **That dependency edge appears backwards.** Per the brief
  I did not start or modify mf1xwk.

## The list the board migration should reuse

1. The `pi` system plugin and `local-models` vendor describe agents-infra from the outside
   (binary = `agents-infra`, status via `agents-infra runtime status --json`, config pointer
   = an agents-infra checkout). Fine for task-board, circular for agents-infra.
2. Composition is validate-only. Any consumer that *produces* an MCP prefix keeps its own
   per-system producer branch.
3. `agentic.Plan` is single-variant. Any consumer needing simultaneous
   interactive/managed-host/managed-client variants hits audit change #3, unshipped.
4. `Plan.Nodes` is a dependency graph, not a variant set — easy to mistake for #3.
5. Runtime coverage is the six frozen rows only. `pi` and `local-models` have plugins but
   **no runtime binding**, so a `pi`-based target resolves to nothing.
6. `qwen` is overloaded: runtime `qwen` = qwen-code × alibaba, unrelated to `qwen-infra`
   = Pi + local Qwen profile. Never map by name.
7. Registry membership (7) is wider than any consumer's admitted set. Driving an admission
   gate off `Registry.IDs()` weakens the gate.
8. Depending on the library forces `go 1.21.0 => 1.25.5`.

## Decision needed

One of:

- **A — schedule the missing contract work.** Create the task for audit change #3 (typed
  named plan variants + managed-host/client partition) plus a producer-side composition
  contribution and the `pi × local-models` runtime, re-accept the lockstep plan for the
  extra release, then re-run this task. Highest cost, and the only option that satisfies
  the DoD as written.
- **B — narrow this task to what v0.5.0 honestly supports** and rewrite the DoD: agents-infra
  takes the dependency and consumes the library for *identity* only (`SystemID`
  normalization, registry lookup as a cross-check), while composition production,
  target/alias resolution and primary-session variants stay in agents-infra per the audit's
  own "stays here" verdicts. The hardcoded `codex`/`claude`/`pi` branches **remain**, so
  DoD items 1, 3 and 4 must be struck.
- **C — unblock the ordering first.** Fix the `mf1xwk → 11ajl2` edge direction, and let
  y6infr land the pi plugin consumption and the `pi × local-models` runtime (it retries
  Sep 6 or needs a non-codex agent), then re-scope this task to the codex/claude surfaces
  only.

**Recommendation: B, then A.** The audit already classified three of this task's four
surfaces as "stays here" and only one as "contract change". The DoD was written against the
`v0.2.0` shape the brief describes, which the audit had already overturned. Option B makes
the task honest immediately and defers the one genuine contract gap (#3) to a task that
names it.
