# TASK-260830-11ajl2 — identity-only consumption of skill-agents-management v0.5.0

Second run, against the narrowed scope. The first run's gap report stands unchanged;
nothing it found was worked around. This run implements option B exactly: agents-infra
takes the dependency and consumes it for **identity only**, and every hardcoded
`codex` / `claude` / `pi` branch the audit marked "stays here" is still there.

Worktree `6fd5e5d`, story branch `task-board/story/STORY-260830-37bq03`, base not moved.

## What landed

| File | Change |
| --- | --- |
| `tools/agents-infra/go.mod` | `+ github.com/relux-works/skill-agents-management v0.5.0`; `go 1.21.0` → `go 1.25.5` |
| `tools/agents-infra/go.sum` | two entries, checksum-DB verified |
| `internal/infra/agentic_identity.go` | **new**, 103 lines — the whole consumption surface |
| `internal/infra/project_config.go` | 2 lines: the inline environment literal becomes a call into the gate above |
| `internal/infra/agentic_identity_test.go` | **new**, 7 tests |
| `internal/infra/agentic_identity_guard_test.go` | **new**, 2 tests |
| `README.md` | the Go floor and its two failure modes, stated where the toolchain requirement already was |

Nothing else. No production file other than `project_config.go` changed, and no existing
test file was touched at all.

## The consumption surface, exactly

`resolveAdmittedEnvironmentSystemID(registry, environment)` — the only place agents-infra
reaches the library. It uses four names from `pkg/agentic`: `NormalizeSystemID`,
`SystemID`, `Registry`, `Default`. Nothing else.

Production call site: **`internal/infra/project_config.go:384`**, inside
`validateProjectTarget`, which `parseProjectTargets` calls for every configured target —
the sole admission gate for `agents.targets.*`.

The gate is a **conjunction**, in this order:

1. the environment is in `admittedTargetEnvironments` (`codex`, `claude-code`, `pi`) — checked first and alone;
2. it normalizes through `agentic.NormalizeSystemID`;
3. it normalizes **to itself**, so one system never keys state under two spellings;
4. `Registry.Lookup` finds a registered plugin under that id.

The environment name is used as the identifier directly. agents-infra's three environments
and the plugin ids they name are the same three strings (`codex`, `claude-code`, `pi`), and
writing that correspondence down as a lookup table would be a second binding that can drift
from the first. The equality is **asserted** by step 3 + step 4 rather than assumed.

### What is deliberately NOT consumed

- **The admitted set is not derived from the registry.** The plugin plane registers seven
  systems; agents-infra launches three. Deriving admission from `Registry.IDs()` would widen
  the gate to `gemini-cli`, `muse` and `antigravity` targets this program has no launcher
  for. That is the gate weakening the first run's report named, and mutant M-a below proves
  the test suite catches it.
- **Vendors are not cross-checked.** agents-infra's vendor `qwen` means Pi driving a local
  Qwen profile; the plugin plane's nearest spelling is `qwen-code` × alibaba, an unrelated
  system, and there is no `pi × local-models` runtime in v0.5.0 to mean what agents-infra
  means. `TestAgentsInfraVendorLabelsAreNotAgenticSystemIdentities` pins that they must not
  be matched by name.
- **No behaviour, only names.** The three plugin packages are blank-imported so their `init`
  functions register a declaration; agents-infra never calls `ResolveBinary`, `BuildPlan`,
  `Preflight` or `ValidateComposition`. That is not taste — the shipped `pi` plugin resolves
  the binary `agents-infra` and reads readiness by shelling out to
  `agents-infra runtime status --json`, so calling it from inside agents-infra would make the
  program re-exec and shell out to itself. `TestProductionCodeImportsAgentsManagementForIdentityOnly`
  and `TestIdentityFileUsesOnlyTheIdentitySurfaceOfPkgAgentic` enforce this mechanically over
  every production `.go` file in the module.

### What stayed hardcoded, on purpose

- `child_launch_composition.go:186-197` — the codex/claude MCP prefix branch. v0.5.0
  `ValidateComposition` refuses a prefix the caller already built; there is no producer
  contract, so the branch can only be bypassed, not removed.
- `primary_session_launch_plan.go:249-256` — per-environment plan builders. `agentic.Plan`
  is single-variant; agents-infra needs interactive + managed-host + managed-client at once.
- `canonical_target.go:417-427` — per-environment CLI-argument lock, an audit "stays here".

## Evidence

Every command below was run as a standalone process from
`tools/agents-infra`, with `GOWORK=off`. Real exit codes.

| Gate | Command | Exit |
| --- | --- | ---: |
| build | `GOWORK=off go build ./...` | 0 |
| vet | `GOWORK=off go vet ./...` | 0 |
| gofmt | `gofmt -l .` (module, excluding `.temp`) | 0, no output |
| full suite | `GOWORK=off go test ./... -count=1` | 0 |
| root pkg re-run after the README edit | `GOWORK=off go test . -count=1` | 0 |

Package timings from the full run: `agents-infra` 109.196s, `internal/infra` 183.567s,
`internal/attachments` 1.521s, `internal/modelharness` 15.578s; `cmd/model-harness` has no
test files.

**Lint**: this repo configures no golangci-lint or staticcheck — there is no config file and
neither binary is installed. The lint gate that exists here is `gofmt` + `go vet`, both
clean and reported above. Nothing else was run, and nothing else is claimed.

### The 482 are intact, by name and not just by count

Baseline was regenerated from `HEAD` into a throwaway tree (`git archive HEAD | tar -x`) and
both lists compared:

```
baseline (HEAD):  482
after:            491
removed:          0        <- comm -23, empty
added:            9        <- the new test functions, listed below
```

Per package after the change: `.` 88 (unchanged), `internal/infra` 366 (357 + 9),
`internal/attachments` 13 (unchanged), `internal/modelharness` 24 (unchanged).

The nine added:

```
TestAdmittedEnvironmentGateFailsClosedOnANilRegistry
TestAdmittedEnvironmentGateRefusesASystemThePlaneDoesNotRegister
TestAdmittedEnvironmentsResolveToRegisteredAgenticSystems
TestAgentsInfraVendorLabelsAreNotAgenticSystemIdentities
TestIdentityFileUsesOnlyTheIdentitySurfaceOfPkgAgentic
TestParseProjectConfigRefusesAdmittedEnvironmentsThePluginPlaneRejects
TestParseProjectConfigRefusesRegisteredSystemsThatAgentsInfraDoesNotAdmit
TestProductionCodeImportsAgentsManagementForIdentityOnly
TestProviderLabelIsNotAnAgenticSystemID
```

`git status` confirms no existing test file was modified: the only changed tracked files are
`README.md`, `go.mod`, `go.sum`, `project_config.go`, plus three added files.

### Mutants — the gate rejects, and rejects the right class

Four mutants, each applied to the pristine file, run, and reverted. Full output in
`TASK-260830-11ajl2_mutants.log`.

| # | Mutant | Kind | Tests that went red | Exit |
| --- | --- | --- | --- | ---: |
| M-a | drop the admitted-list check, leaving registry membership as the gate | **widening** | `TestParseProjectConfigRefusesRegisteredSystemsThatAgentsInfraDoesNotAdmit` — all four subtests (`antigravity`, `gemini-cli`, `muse`, `qwen-code`) | 1 |
| M-b | cross-check only `codex`, return early for the rest | **narrowing** | `TestParseProjectConfigRefusesAdmittedEnvironmentsThePluginPlaneRejects` (3 subtests) + `TestAdmittedEnvironmentGateRefusesASystemThePlaneDoesNotRegister` | 1 |
| M-c | drop the `pi` blank import | plane narrowing | new: `TestAdmittedEnvironmentsResolveToRegisteredAgenticSystems`, `TestProductionCodeImportsAgentsManagementForIdentityOnly`; **existing**: `TestCanonicalTargetParserRejectsEveryCrossVendorEnvironmentPair/{openai_pi,anthropic_pi}`, `TestCanonicalTargetParserFieldAndReasoningDomainsFailClosed/{pi_reasoning,pi_missing_profile,pi_relative_endpoint}` | 1 |
| M-d | add `agentic.BuildPlan` to the identity file | surface widening | `TestIdentityFileUsesOnlyTheIdentitySurfaceOfPkgAgentic` | 1 |

M-b is the narrowing mutant the evidence contract asks for: it does not delete the
cross-check, it shrinks the class the cross-check covers from three systems to one, and the
suite still catches it.

M-c matters for a different reason. It proves the cross-check is **load-bearing in
production, not only in its own tests** — dropping one plugin import makes five
pre-existing `canonical_target` tests fail, because every project config naming `pi` is now
refused at parse time. That is the loud failure this gate exists to produce.

The anti-widening test is proven from both sides:

- widening the **plugin plane** does not widen the gate — the test binary deliberately
  registers four systems production does not link (`agy`, `gemini`, `muse`, `qwen`), and
  each is still refused by the real `parseProjectConfig` entry point;
- widening the **admitted list** does not widen the gate either — three subtests append an
  environment to `admittedTargetEnvironments` and drive `parseProjectConfig`, and the plugin
  plane refuses each one (unregistered, second spelling, not an identifier).

Both halves of the conjunction are required. Neither alone admits a target.

## The Go floor bump: 1.21.0 → 1.25.5

Forced, not chosen. `skill-agents-management` declares `go 1.25.5`, so `go get` raised ours:

```
go: upgraded go 1.21.0 => 1.25.5
go: added github.com/relux-works/skill-agents-management v0.5.0
```

What it touches, checked:

- **CI: none exists.** There is no `.github`, no `.gitlab-ci.yml`, no `.circleci` in this
  repository. Nothing hosted to update, and nothing hosted to be broken. Gates are local.
- **`scripts/setup.sh` `install_go()` checks presence, never version.** It runs
  `command -v go` and, on macOS with Homebrew, `brew install go` when missing. A host that
  already has an old Go passes that check and then meets the floor at build time instead.
  Left as is — it is not wrong, and changing installer policy is outside this task.
- **The launcher builds on every invocation**, so the floor is a *runtime* requirement, not
  just a build-time one (`internal/infra/launcher_backend.go:73-92`, README "the generated
  launcher runs `go build .` … on every invocation").
- **The two failure modes, reproduced** rather than asserted. In a throwaway copy of `HEAD`
  with the directive raised above the installed toolchain:

  ```
  $ GOTOOLCHAIN=local go build ./...
  go: go.mod requires go >= 1.99.0 (running go 1.25.5; GOTOOLCHAIN=local)

  $ GOTOOLCHAIN=auto go build ./...
  go: downloading go1.99.0 (darwin/arm64)
  go: download go1.99.0 for darwin/arm64: toolchain not available
  ```

  So: below the floor with `GOTOOLCHAIN=local` → hard refusal; with the default
  `GOTOOLCHAIN=auto` → a toolchain download, which needs network. Both are new exposure at
  1.25.5 that 1.21.0 did not have in practice.
- **`launcherStartupFailure` fails closed on it.** It captures `CombinedOutput` and reports
  "the generated agents-infra launcher runs `go build …` on every invocation, and that build
  fails", carrying the toolchain error verbatim. A host below the floor gets a refusal that
  names the cause, not a broken install.
- **Module fetchability: public, no credentials.**
  `https://proxy.golang.org/github.com/relux-works/skill-agents-management/@v/v0.5.0.info`
  answers `200`, and `go.sum` carries `sum.golang.org`-verified hashes under the default
  `GOPROXY=https://proxy.golang.org,direct` / `GOSUMDB=sum.golang.org` with no `GOPRIVATE`
  set. No new authentication requirement for the per-invocation launcher build; a first
  build on a cold host does need network, as it already did for the three existing
  dependencies.
- **Binary size**: 11,427,442 → 11,507,074 bytes, +79,632 (+0.70%). `cobra` is not linked —
  it belongs to the library's own CLI, which we do not import.
- **No test or script asserts the `go` directive**, so nothing else moved with it.

## Scope kept

- No new agents-management release, no tag, no version other than exact `v0.5.0`.
- No hardcoded branch removed.
- `TASK-260830-mf1xwk` not started, not modified, not read for write.
- Story branch base not moved.
- No credential, token, cookie or keychain value printed, exported or persisted.

## Two things a reviewer should decide

1. **The board checklist was stale relative to the ratified scope.** The task description and
   AC were rewritten for option B, but `progress.md` still carried the four struck DoD items.
   They are listed verbatim in `TASK-260830-11ajl2_checklist-realignment.md` along with what
   replaced them. If any of those removals is not what was intended, the artifact is enough
   to restore them exactly.
2. **`TASK-260830-y6infr` still blocks this task on the board.** That edge was correct for
   the original scope — y6infr owns the pi plugin consumption and the `pi × local-models`
   runtime that the old DoD needed. The narrowed scope needs neither. y6infr is itself parked
   (codex provider limit, retry Sep 6). Whether to drop the edge is an orchestrator call, not
   one this run made.
