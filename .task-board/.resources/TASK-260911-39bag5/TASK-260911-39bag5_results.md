# TASK-260911-39bag5 — derive environment admission and dispatch from the agentic registry

## Change

Files touched (all within the declared scope):

- `tools/agents-infra/internal/infra/agents_management_registry.go` — added the single
  `launchableSystems` declaration (`LaunchableSystem{Environment, Provider, ResolvesExecutable}`),
  its accessors (`LaunchableEnvironments`, `LaunchableProviders`, `ProviderForLaunchableEnvironment`,
  `launchableSystemForEnvironment`, `launchableSystemForProvider`), and the two production
  admission entries `ValidateLaunchableEnvironment` / `ValidateLaunchableProvider`, each of which
  cross-checks the declaration against `agentic.Default` (via an injectable `agenticLookup` for
  testability, defaulting to `agentic.Default.Lookup`). Blank-imports `pkg/agentic/systems/{codex,claude}`
  so those two register into `agentic.Default`; pi was already registered through the existing
  `managementpi` import.
- `tools/agents-infra/internal/infra/project_config.go` — `validateProjectTarget`'s environment
  admission now calls `ValidateLaunchableEnvironment` instead of an inline
  `containsString([]string{"codex","claude-code","pi"}, ...)` + hardcoded error string. Wording is
  unchanged (`"must be one of codex, claude-code, pi"`, golden-tested).
- `tools/agents-infra/internal/infra/canonical_target.go` — `canonicalProviderForEnvironment` now
  reads `ProviderForLaunchableEnvironment` instead of duplicating the environment→provider switch;
  `CanonicalProviderForEntrypoint` and `lockCanonicalTargetArguments` now reference the named
  `launchableEnvironment*`/`launchableProvider*` constants instead of restating the bare string
  literals (their own switch keys — entrypoint alias names, or an already-validated environment —
  are a different, legitimate domain and are kept as literals).
- `tools/agents-infra/internal/infra/primary_session_launch_plan.go` — `BuildPrimarySessionLaunchPlan`'s
  provider admission now calls `ValidateLaunchableProvider` (wording unchanged:
  `unsupported provider %q`); the pi-specific "does this provider resolve its own executable"
  branch now reads `launchableSystemForProvider(provider).ResolvesExecutable` instead of `provider != "pi"`;
  the builder dispatch switch uses the named provider constants instead of bare literals.
- `tools/agents-infra/internal/infra/agents_management_registry_test.go` (new) — golden-text,
  admission, mutant-resistance and source-scan tests (see below).

Deliberately **out of scope** (left untouched): `project_config.go`'s `loadPrimaryProviderProjectConfig`
/`parseProjectConfigForProvider` provider checks (`"codex"`/`"claude"`, no `"pi"`) and
`canonical_target.go`'s hosted-vs-local branches (`target.Environment != "pi"`, `provider == "codex" || provider == "claude"`).
These are a genuinely different, narrower 2-of-3 concept (primary-session-provider config /
hosted-vs-local bookkeeping) that must **not** gain "pi" — folding them into the 3-way
launchable-systems declaration would be a behavior change, which the task explicitly forbids.

## Why registry cross-check, not just the declaration

`agentic.Default` is the compatibility registry `skill-agents-management`'s system plugins
self-register into via `init()`. It is **wider** than what agents-infra can launch — it also
registers `gemini-cli`, `muse`, `qwen-code` and `antigravity` when those packages are imported
anywhere in the binary. `ValidateLaunchableEnvironment`/`ValidateLaunchableProvider` therefore
require BOTH: declared in `launchableSystems` AND registered under that exact spelling in
`agentic.Default`. Exact-string matching (not `agentic.NormalizeSystemID`'s folded form) is what
makes a case or alias variant ("Codex", "claude") fail closed instead of being silently folded
onto its canonical spelling and admitted.

## AC coverage — n of m driven through the production entry

| AC row | Production call site | Test |
|---|---|---|
| Unregistered identifier refused | `ValidateLaunchableEnvironment` | `TestValidateLaunchableEnvironmentGoldenText` ("other") |
| Registered-but-unlaunchable system (gemini-cli) refused | `ValidateLaunchableEnvironment` | `TestValidateLaunchableEnvironmentRefusesRegisteredButUnlaunchableSystem` — blank-imports `pkg/agentic/systems/gemini` in the test binary only, confirms it IS in `agentic.Default`, then confirms refusal |
| Case/alias variant refused | `ValidateLaunchableEnvironment` | `TestValidateLaunchableEnvironmentRefusesCaseAndAliasVariants` (Codex, CODEX, Claude-Code, claude, CLAUDE-CODE, Pi, padded) |
| Existing CLI wording preserved (golden) | `validateProjectTarget` → `ValidateLaunchableEnvironment` | `TestValidateLaunchableEnvironmentGoldenText`, `TestValidateLaunchableProviderGoldenText` |
| Provider-side admission (dispatch path) | `BuildPrimarySessionLaunchPlan` → `ValidateLaunchableProvider` | `TestValidateLaunchableProviderRefusesUnadmittedProvider`, plus pre-existing `TestBuildPrimarySessionLaunchPlanRejectsUnknownProvider` |
| Existing project-config fixtures / installed-binary setup tests unchanged | `parseProjectConfig`, compiled `openai-infra`/`anthropic-infra` alias binaries | `TestCanonicalTargetParserFieldAndReasoningDomainsFailClosed`, `TestCanonicalTargetParserRejectsEveryCrossVendorEnvironmentPair`, `TestCanonicalAliasReachesSiblingTargetAndPreservesCWDArgv`, `TestDirectProviderYoloAliasesDelegateOnceAndPreserveCWDArgv`, `TestCanonicalAliasRefusesMissingAndNonRegularSibling` — all still green |

7 of 7 AC rows driven through the two named production entries (`ValidateLaunchableEnvironment`,
`ValidateLaunchableProvider`) plus the pre-existing golden suites that exercise them transitively.

## Mutant evidence

| Mutant | Narrows to | Named test that fails | Bound stated by survival |
|---|---|---|---|
| Drop the `lookup(...)` call inside `validateLaunchableEnvironment`, keep only declaration membership | Admits any string in `launchableSystems` regardless of registry state | `TestValidateLaunchableEnvironmentRegistryCrossCheckIsLoadBearing` | none observed — mutant fails |
| Drop the `lookup(...)` call inside `validateLaunchableProvider` | Admits any provider in `launchableSystems` regardless of registry state | `TestValidateLaunchableProviderRegistryCrossCheckIsLoadBearing` | none observed — mutant fails |
| Admit an identifier absent from `launchableSystems` merely because some registry carries it | Registry membership alone becomes sufficient | `TestValidateLaunchableEnvironmentRegistryCrossCheckIsLoadBearing` (second half, `gemini-cli` registered in an isolated full registry) | none observed — mutant fails |
| Re-add `[]string{"codex", "claude-code", "pi"}` literal in `project_config.go`, or the equivalent case/condition literals in `canonical_target.go` / `primary_session_launch_plan.go` | Restates the admission/dispatch set inline instead of routing through `launchableSystems` | `TestLaunchableSystemSourceHasNoReintroducedLiteralList` (source-scan guard, mirrors `skill-agents-management`'s own `singlesource_guard_test.go` pattern) | Stated bound: this text-scan guard alone matches only the specific historical literal spelling it was written against; see the next row for the mutant that a reformatted or content-different literal at the same call site produces, and the behavioral test that still catches it |
| At the `validateProjectTarget` call site, stop calling `ValidateLaunchableEnvironment` and inline a differently-formatted, differently-CONTENTED literal instead (e.g. a four-item, no-space list that also admits `"gemini-cli"`) — this PRESERVES the "codex"/"claude-code"/"pi" tokens the text-scan guard looks for (in different formatting/superset form) so the static checker alone would miss it | Admits a registered-but-unlaunchable environment at the real production call site | `TestValidateProjectTargetRefusesRegisteredButUnlaunchableEnvironmentAtTheRealCallSite` — drives `validateProjectTarget` itself (not `ValidateLaunchableEnvironment` in isolation) with `gemini-cli` genuinely registered in the test binary | none observed — mutant fails; this is the behavioral suite closing the gap the static-only checker leaves open, per the "gate that inspects source text" DoD item |

Applying this last mutant surfaced a real finding worth recording: `validateProjectTarget` has a
SECOND, coincidental safety net downstream — the vendor/environment pairing `switch`'s `default`
arm refuses any pair it does not explicitly admit, including `openai/gemini-cli`. A first version
of this test asserted only `err != nil` and passed against the mutant for the wrong reason (the
downstream switch caught it, not the environment gate). The test now asserts the environment
gate's own wording (`"must be one of codex, claude-code, pi"`) rather than mere error presence,
which is what makes it actually distinguish "this gate refused it" from "something downstream
happened to refuse it too" — confirmed by re-running it against the mutant (failed, wrong message)
and against the restored code (passed).

All five mutants were applied by hand against a scratch copy and confirmed to fail their named
test before being discarded; the committed code has none of them.

## Verification run

- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l` on all touched files — clean (no output)
- `go test ./internal/infra/... -run 'TestValidateLaunchable|TestLaunchableSystem' -v` — all new tests pass
- `go test . -run 'TestCanonicalAlias|TestDirectProviderYolo|TestParseProjectConfig|TestCanonicalTarget|TestBuildPrimarySessionLaunchPlan|TestValidateLaunchable|TestLaunchableSystem' -v` — all pass (includes the installed-binary alias/setup tests, which exercise real compiled/symlinked binaries)
- `go test ./... -count=1` (full tools/agents-infra module, all packages) — **all green**, including `internal/infra` (253.6s). An earlier full-suite run surfaced one failure in
  `TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`, unrelated to
  any file this task touches; it was reproduced as passing both in isolation and in a second full
  run against the same (this task's) code, confirming it was a pre-existing timing flake under
  parallel load, not a regression from this change.
- No `go.mod`/`go.sum` change.
