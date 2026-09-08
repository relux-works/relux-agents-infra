# skill-agents-management v0.5.0 — what the first consumer found

For `TASK-260830-mf1xwk` (task-board migration) and `TASK-260831-1pfnxx` to reuse.
Written by the first repository that actually compiled against `v0.5.0` and shipped a
consumption of it. Items 1-8 are the first run's findings, carried forward verbatim in
substance; items 9-12 are what the second run learned while landing the narrowed scope.

Each item says whether it is a **contract gap** (the library cannot express something a
consumer needs) or a **trap** (the library can express it, and a consumer that reads it
naively gets it wrong).

## Contract gaps

**1. The `pi` system plugin and `local-models` vendor describe agents-infra from the
outside.** `pi.ResolveBinary` resolves the binary **`agents-infra`** on PATH;
`pi/preflight.go` reads readiness through `localruntime.NewCLIStatusReader()`, which shells
out to `agents-infra runtime status --json`; `vendorplugin/vendors/local-models`'s `Pointer`
is an absolute path to an agents-infra checkout plus an `agents.pi.profiles.<name>` key
inside it. Perfect for task-board, which drives agents-infra as a subprocess. Circular for
agents-infra itself, which would re-exec and shell out to itself. **This is not a gap a
library patch closes** — it is a statement about which side of the boundary a consumer is
on. Ask that question first. Tracked as `TASK-260831-1pfnxx`.

**2. Composition is validate-only.** `System.ValidateComposition(Composition) error` refuses
a prefix the caller already built; `Composition{Prefix, Servers}` is documented as "already
composed by the caller". codex's and claude's validators are refusal surfaces with no
producer behind them. Any consumer that *produces* an MCP prefix keeps its own per-system
producer branch, and calling `ValidateComposition` afterwards is additive validation that
leaves the branch in place.

**3. `agentic.Plan` is single-variant.** One `Mode`, one `Binary`, one `Argv`. A consumer
that needs interactive + managed-host + managed-client argv from one build hits audit change
**#3, which is not shipped in v0.5.0**. claude declares only `LaunchModeExec` and
`LaunchModeDryRun`, and the omission of `LaunchModeManagedSession` is deliberate and
documented — declaring it "would be a capability claim with no construction behind it".

**4. `Plan.Nodes` is a dependency graph, not a variant set.** `BuildMultiNodePlan` orders
*dependency processes* (engine, sidecar) dependency-first. It is easy to mistake for #3 and
it is not #3.

**5. Runtime coverage is the six frozen rows only.** `agy`, `claude`, `codex`, `gemini`,
`muse`, `qwen`. Lookup of runtime `pi` → `found=false`; `local-models` → `found=false`. A
`pi`-based target resolves to nothing on the runtime plane, even though a `pi` **system**
plugin exists. System plugins and runtime bindings are separate vocabularies with separate
coverage.

## Traps

**6. `qwen` is overloaded — never map by name.** Runtime `qwen` = qwen-code × alibaba.
agents-infra's `qwen-infra` = Pi + a local Qwen profile. Unrelated. The plugin-plane vendor
for local Pi models is `local-models`, not `qwen`. agents-infra pins this in
`TestAgentsInfraVendorLabelsAreNotAgenticSystemIdentities`.

**7. Registry membership is wider than any consumer's admitted set.** v0.5.0 registers seven
systems: `antigravity`, `claude-code`, `codex`, `gemini-cli`, `muse`, `pi`, `qwen-code`.
agents-infra admits three environments. **Driving an admission gate off `Registry.IDs()`
weakens the gate.** The safe shape is a conjunction — the consumer's own admitted list AND a
registry lookup — with the consumer's list checked first and alone. agents-infra's
`resolveAdmittedEnvironmentSystemID` is a worked example, and mutant M-a in its results
artifact shows what the widened version breaks.

**8. Depending on the library forces `go 1.21.0 => 1.25.5`.** It declares `go 1.25.5`.

**9. A consumer's own internal labels are not system ids.** agents-infra's provider label for
the `claude-code` environment is `claude`, and no system is registered under `claude`. Any
identity resolution driven off an internal label instead of the boundary identifier looks up
a name nobody registers. Pin it: `TestProviderLabelIsNotAnAgenticSystemID`.

**10. Register-time normalization does not make a consumer's identifiers safe.** The library
guarantees a plugin's `ID()` is its own normal form. It guarantees nothing about the strings
a consumer feeds `Lookup`, and `Lookup` reports **not-found** for an id that fails to
normalize — an unnormalizable id and an unregistered id are the same answer. A consumer that
wants those distinguished must call `NormalizeSystemID` itself first, and should also assert
`normalized == raw` so one system never keys consumer-side state under two spellings.

## Mechanical notes

**11. `v0.5.0` is public and checksum-DB verified.** `proxy.golang.org` answers `200` for
`@v/v0.5.0.info`; `sum.golang.org` hashes land in `go.sum` under default `GOPROXY`/`GOSUMDB`
with no `GOPRIVATE`. No credential setup is needed to depend on it.

**12. Importing the system plugins is inert.** `init` hands the registry a declaration and
nothing more; `pi.New(localruntime.NewCLIStatusReader())` builds a reader that performs no
work until `Status` is called. Linking `agentic` plus the three system plugins into
agents-infra cost **+79,632 bytes (+0.70%)** and does not link `cobra`, which belongs to the
library's own CLI. So even a consumer on the wrong side of gap 1 can safely link the plugins
for identity — it just must not call them, and should enforce that mechanically rather than
by convention (agents-infra does, in `agentic_identity_guard_test.go`).
