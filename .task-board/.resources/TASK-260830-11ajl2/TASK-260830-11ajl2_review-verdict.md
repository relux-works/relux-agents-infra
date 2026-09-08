# TASK-260830-11ajl2 — review verdict: ACCEPTED

Reviewer run `RUN-260831-781ec1`. Head under review `8c7b771` on
`task-board/story/STORY-260830-37bq03`, base not moved
(`git diff --name-status $(git merge-base main HEAD) HEAD` = the commit's own files).
Host darwin 25.5.0 arm64, `go1.25.5`, all commands `GOWORK=off` from
`tools/agents-infra`. Nothing in the reviewed worktree was modified: every mutant
below was applied to a throwaway copy under `.temp/RUN-260831-781ec1/mut/`, and
`git status --short` is empty at the end of this run.

## Verdict

Accepted. The admission gate is at least as strict as the literal it replaced, the
consumption is genuinely identity-only, the Go floor is stated accurately and its
two branches reproduce, and no pre-existing test was edited. Four findings are
recorded below; none of them changes production behaviour and none is rework.

## What I re-ran rather than accepted from the producer

| Gate | Command | Result |
| --- | --- | ---: |
| build | `go build ./...` | exit 0 |
| vet | `go vet ./...` | exit 0 |
| gofmt | `gofmt -l .` | no output |
| full suite | `go test ./... -count=1` | exit 0, 4 packages green, 247s |

Suite log: `TASK-260830-11ajl2_review-suite.log`. Package timings on this host:
`agents-infra` 84.2s, `internal/infra` 145.5s, `internal/modelharness` 14.9s,
`internal/attachments` 2.3s.

## 1. Can a fourth environment be admitted? No, by two independent bounds

Attacked, not read. Mutants applied one at a time to the pristine file in the
throwaway copy:

| # | Mutant | Kind | Caught by | Exit |
| --- | --- | --- | --- | ---: |
| R1 | delete the `admittedTargetEnvironments` check, leaving registry membership as the gate | **widening** | `TestParseProjectConfigRefusesRegisteredSystemsThatAgentsInfraDoesNotAdmit` — all 4 subtests (`antigravity`, `gemini-cli`, `muse`, `qwen-code`) | 1 |
| R2 | delete the gate call site in `validateProjectTarget` entirely | delete-only | the above 4 **plus** `TestParseProjectConfigRefusesAdmittedEnvironmentsThePluginPlaneRejects` (3 subtests) | 1 |
| R3 | cross-check `codex` only, return the raw id for the rest | **narrowing** | `TestParseProjectConfigRefusesAdmittedEnvironmentsThePluginPlaneRejects` (3), `TestAdmittedEnvironmentGateRefusesASystemThePlaneDoesNotRegister`, `TestAdmittedEnvironmentGateFailsClosedOnANilRegistry` | 1 |
| R6 | drop the `pi` blank import | plane narrowing | 2 new tests **plus** pre-existing `TestCanonicalQwenProfileAssertionsFailClosed` (7 subtests), `TestCanonicalTargetParserRejectsEveryCrossVendorEnvironmentPair/{openai_pi,anthropic_pi}`, `TestCanonicalTargetParserFieldAndReasoningDomainsFailClosed/{pi_reasoning,pi_missing_profile,pi_relative_endpoint}`, `TestDoctorReportsCanonicalQwenTargetAndProfileProvenance`, `TestBuildStandalonePiArgumentsOwnsExactAuthorizationAndMediumReasoning`, `TestStandalonePiNearestFalseMasksInheritedAuthorization`, `TestRunPiStandaloneRefusesCallerAuthorizationAndRPCFlagsBeforeExecutableLookup` (10 subtests) | 1 |

R3 is the narrowing proof the evidence contract asks for: it does not delete the
cross-check, it shrinks the covered class from three systems to one, and three
separate tests still go red. R6 is the load-bearing-in-production proof, and it is
stronger than the producer reported — dropping one plugin import turns **far more
than five** pre-existing tests red across two packages, because every project
config naming `pi` is refused at parse time.

Two structural facts confirmed by reading, not inferring:

- `validateProjectTarget` is the sole caller and `parseProjectTargets` its sole
  entry; `ProjectTarget` is constructed in exactly one place
  (`project_config.go:290`) and every construction goes through the gate. There
  is no second parser, no CLI-flag path, and no doctor path that synthesizes a
  target around it.
- Even the R1 widening does **not** admit a fourth environment end to end: the
  vendor/environment pair switch at `project_config.go:398-421` is a second,
  independent closed set (`openai/codex`, `anthropic/claude-code`, `qwen/pi`).
  Under R1 the registered-but-unadmitted systems were still refused — with the
  wrong reason, which is what the test caught. Admitting a fourth environment for
  real needs coordinated edits in four places, which is a feature, not a bypass.

Fail-closed behaviour verified against the library source, not assumed:
`Registry.Lookup` returns `(nil, false)` on a nil receiver and on an id that fails
to normalize (`pkg/agentic/registry.go`), so
`TestAdmittedEnvironmentGateFailsClosedOnANilRegistry` is pinning real behaviour
rather than a panic. The refusal wording `must be one of codex, claude-code, pi`
is byte-identical to the literal it replaced, so the CLI surface is unchanged.

## 2. Identity only — genuinely

- Only two production files changed at all: the new `agentic_identity.go` and two
  lines of `project_config.go`. `canonical_target.go`,
  `primary_session_launch_plan.go`, `child_launch_composition.go`,
  `model_check.go` and every launch file are untouched in the diff. Composition
  production, target/alias resolution, primary-session variants and model-check
  stay in agents-infra, as the ratified scope requires.
- The import guard holds in both directions. A second production file importing
  `pkg/agentic`, and any file importing `pkg/localruntime`, are both refused
  (mutant R5, two distinct errors).
- The dangerous behaviour surfaces are caught. `sys.ValidateComposition(agentic.Composition{})`
  inside the identity file fails `TestIdentityFileUsesOnlyTheIdentitySurfaceOfPkgAgentic`
  (mutant R4b), and the same holds for `ResolveBinary`, `Argv`, `Stdin` and
  `BuildPlan`: each needs an `agentic`-typed argument, so calling one forces a
  disallowed selector into the file. The re-exec / self-shell-out hazard the gap
  report named is structurally unreachable.

## 3. The Go floor, both branches reproduced

Neither branch can be produced on this host by running an older Go, since the host
is exactly at the floor. I reproduced the **mechanism** instead — a `go.mod`
directive above the running toolchain — which is the identical code path:

```
$ printf 'module floorprobe\n\ngo 1.99.0\n' > go.mod
$ GOTOOLCHAIN=local go build ./...
go: go.mod requires go >= 1.99.0 (running go 1.25.5; GOTOOLCHAIN=local)

$ GOTOOLCHAIN=auto go build ./...
go: downloading go1.99.0 (darwin/arm64)
go: download go1.99.0 for darwin/arm64: toolchain not available
```

`go env GOTOOLCHAIN` is `auto` by default. So README:149's two claims — hard
refusal under `local`, toolchain download (network) under `auto` — are accurate.
The substitution is 1.99.0 for "any host below 1.25.5"; the branch selection in
cmd/go depends only on floor > running, not on the specific versions.

Blast radius checked independently:

- `find . -name go.mod` → **one** module. No second module inherits the floor.
- No `.github`, no `.gitlab-ci.yml`, no `.circleci`. No hosted CI to break.
- `grep` for any `go-version` / `go 1.1x` / `go 1.2x` pin across `*.yml`, `*.yaml`,
  `*.sh`, `*.md`, `*.toml`, `Makefile` → **no hits**. Nothing anywhere asserts the
  old floor.
- No `toolchain` directive in `go.mod`, so `auto` picks the floor version itself.

## 4. No test edited

Verified across the whole branch, not just the commit:
`git diff --name-status $(git merge-base main HEAD) HEAD -- '*_test.go'` returns
exactly two `A` entries and no `M`. Test-function name sets diffed between `HEAD~1`
and `HEAD`: **0 removed, 9 added**, the nine being the new identity and guard
tests. My absolute counts are 485 → 494 against the producer's 482 → 491 because I
count every `func Test` in `*_test.go` including the build-tagged
`runtime_main_darwin_test.go`; the delta and the removal set are identical, so the
claim stands under both methods.

## The checklist question — decided

All four removals are correct. Nothing should be restored.

1. **`Hardcoded codex/claude/pi agent branches removed, not merely bypassed`** —
   correct to remove. The ratified description says the branches remain, and the
   item forbids the only mechanism v0.5.0 offers.
2. **`Adding an environment afterwards requires no agents-infra change; prove with a narrowing mutant`**
   — correct to remove, and the reasoning is **sound, not a rationalisation**. I
   checked it against the code rather than the argument: a fourth environment
   needs a case in `canonicalProviderForEnvironment` (`canonical_target.go:78-89`),
   in `lockCanonicalTargetArguments` (`canonical_target.go:415-425`), in the
   vendor/environment pair switch (`project_config.go:398-421`) and a
   `buildXPrimarySessionLaunchPlan` (`primary_session_launch_plan.go:249-256`).
   Every one of those is agents-infra-owned behaviour the audit assigns here, and
   v0.5.0 offers no producer contract for any of them — its `Argv` builds argv for
   a launch the plugin owns, while `lock*TargetArguments` validates user-supplied
   args against a configured target, which is the opposite direction. The
   empirical half is decisive: the only generalisation that satisfies the item is
   exactly mutant R1, and R1 is the gate weakening, from 3 admitted environments
   to 7 registered systems. Keeping the item would have demanded the change the
   rest of the task exists to refuse.
3. **`...hardcoded branches are gone`** — correct; the surviving half became its
   own item and the rest is (1) again.
4. **`Every contract gap fixed in skill-agents-management and released`** —
   correct to remove; forbidden by the accepted lockstep plan, which requires a
   fresh human acceptance before any release. The gaps are recorded in
   `TASK-260830-11ajl2_consumer-gaps.md` and the structural one is
   `TASK-260831-1pfnxx`.

A producer editing its own DoD is normally a self-serving act. Here it is
acceptable: the removals were recorded verbatim with reasons before review, the
mismatch originated in an orchestrator-side rewrite that the producer did not
author, and the run explicitly handed the decision to review instead of taking it.

## Findings recorded, none blocking

**F1 — the identity guard's narrowing half is narrower than its own comment.**
`TestIdentityFileUsesOnlyTheIdentitySurfaceOfPkgAgentic` inspects `agentic.X`
selector expressions only. Behaviour reached through a method on a `Registry.Lookup`
result evades it: I added `sys.Capabilities()` to `resolveAdmittedEnvironmentSystemID`
(mutant R4) and **both guard tests stayed green**. The guard's comment claims it
fails on "BuildPlan, System, Composition or anything else that would make
agents-infra a consumer of the plugins' behaviour"; "anything else" is not what it
enforces. Severity is low and bounded: only `ID()` and `Capabilities()` have
signatures free of `agentic` types, the other five interface methods are caught
(R4b), and the import guard confines any `System` value to `agentic_identity.go`
alone. Worth tightening the comment, or the guard, when that file next changes.

**F2 — test-function counts are method-dependent.** 482 and 485 are both correct
answers to different questions. Whoever reuses the number should state the method;
the invariant that matters (0 removed) is method-independent.

**F3 — `admittedTargetEnvironments` is a mutable package var.** It has to be, for
the append-and-restore subtests. No production code writes it, and `internal/infra`
has zero `t.Parallel()` calls, so the test mutation is race-free today. A future
parallel test in this package would make it flaky.

**F4 — `SKILL.md:57` states the Go toolchain requirement without a version.**
README:149 carries the floor; the skill doc does not. Cosmetic, optional.

## Handoff

This is a reviewer-archetype run, so it records acceptance evidence and does not
supply `commit_ack`. The commit-owning mover integrates the scope and makes the
final transition. Two orchestrator-level items the producer surfaced and I did not
resolve, because they are not this run's to decide:

- `TASK-260830-y6infr` still blocks this task on the board. The edge was correct
  for the original scope; the narrowed scope needs neither the pi plugin
  consumption nor the `pi × local-models` runtime y6infr owns.
- `TASK-260830-mf1xwk` was not started, read for write, or modified by this
  review.
