# TASK-260830-y6infr — consume-accepted-pi-observer-and-turn-contract

Branch `task-board/story/STORY-260830-3imc9f`, worktree `.temp/STORY-260830-3imc9f/worktree`.
No live runtime, model, process, service, socket, network endpoint, status command,
or user HOME/configuration was contacted at any point.

## Pinned upstream

`github.com/relux-works/skill-agents-management v0.5.1-0.20260830114459-046baef11790`
— the immutable pseudo-version of the independently accepted commit
`046baef11790e93a7967230eec760c4563432270` (tree `454e2aae…`, reviewed patch SHA-256
`471ebb6c…`). No `replace`, no copied interface. `go mod verify`: all modules verified.
The upstream module declares `go 1.25.5`, which forces the same directive in
`tools/agents-infra/go.mod`; nothing else in the repo pinned an older Go.

Per AC7 this checkpoint pins the exact accepted commit. The stable semver tag and
the final pin remain owned by the dedicated delivery task.

## What this session added on top of the prior partial run

The previous spawned run on this task died on a provider usage limit (exit=1) and
left an unreviewed working tree. That tree built and its suite passed, but it had no
genericity guard, no no-live guard, no fail-closed plane discovery, no consumer-side
negative tests, and no mutation evidence. This session added those and fixed the two
defects they exposed.

### Defects the new guards found

1. **Identity literal on the generic launch path.** `MLXEngineObservationAdapter`
   named an engine identity in the plane that agents-management calls, even though
   it never branched on one. Renamed to `SanitizedEngineObservationAdapter`; the
   engine ref arrives only as injected data from the trusted assembly. The concrete
   `mlx` identity now exists exactly once, in `agents_management_registry.go`.
2. **`--profile` was schema-1 only.** The exact assertion is a property of the
   standalone entry point, not of the result schema. `agents-infra pi spawn --profile X …`
   was a CLI parse error on the legacy surface; it now asserts exactly there too.

### New production changes

- `main.go`: `--profile` added to the legacy standalone flag set, forwarded as
  `PiStandaloneRequest.ExpectedProfile`.
- `internal/infra/agents_management_observer.go`: adapter renamed; doc and error
  text carry no engine identity.
- `internal/infra/agents_management_registry.go`: updated to the renamed constructor.

### New tests

| File | Covers |
| --- | --- |
| `internal/infra/agents_management_boundary_test.go` | Fail-closed plane discovery (AST, all non-test sources incl. foreign build tags); observation plane cannot reach a live runtime; consumer plane cannot parse around the sole classifier; neither plane carries an identity literal; the assembly may refuse on identity but not dispatch on it; Process-B lifecycle stays owned by agents-infra and off both planes; contract tests contact no live state |
| `internal/infra/agents_management_consumer_test.go` | All 10 exact pre-child refusal codes through `BuildAndRunPiTurn` → `pi.ValidateTurnResult`; 9 exit/document disagreement shapes refused as `result-invalid`; cancellation after child start; pre-cancelled caller starts no child and performs no observation; dry run is observation- and preflight-free; `qwen-infra` never aliases shipped `qwen-code`/Alibaba; plan is metamorphic under identity renaming |
| `pi_turn_schema_main_test.go` | Schema-1 writer precedes every outer-request refusal with the exact code, exit, and one sanitized document; only the literal `1` selects schema 1; exact-profile assertion refuses every near-miss on both surfaces with a byte-exact sanitized document |

## Negative evidence — narrowing mutants

Each mutant admits exactly one class the guard must reject. `.temp/TASK-260830-y6infr/mutants.sh`
applies, runs the named test, records the real exit code, and restores the tree.
Full log: `.temp/TASK-260830-y6infr/mutants.log`.

| # | Mutant | Test | Exit |
| --- | --- | --- | ---: |
| 1 | Called-helper live read in a NEW same-package file, `os` imported under an alias, called from `ObserveEngine` | `TestObservationPlaneCannotReachLiveRuntime` | 1 |
| 2 | *Discovery* narrowed to a fixed filename list while mutant 1 is applied | same | **0 — bypass admitted, as required** |
| 3 | Second `ValidateTurnResult` on a reached helper path | `TestConsumerPlaneCannotParseAroundSoleClassifier` | 1 |
| 4 | `if request.Runtime == "qwen-infra"` inserted into the generic consumer plane | `TestGenericPlanesContainNoIdentityBranch` | 1 |
| 5 | Assembly dispatches on `Vendor == "qwen"` instead of only refusing | `TestConcreteAssemblyDeclaresIdentityWithoutDispatchingOnIt` | 1 |
| 6 | Consumer plane reads `SharedRuntimeStatusReport` | `TestProcessBLifecycleStaysOwnedByAgentsInfraAndOffTheGenericPlanes` | 1 |
| 7 | Translator admits tool failure as success when final text is present | `TestPiTurnTranslatorClassPrecedence` | 1 |
| 8 | Exact-profile assertion case-folds | `TestExactProfileAssertionRefusesEveryNonIdenticalProfile` | 1 |
| 9 | Consumer reports a clean exit regardless of the waited status | `TestConsumerRefusesExitDocumentDisagreementInsteadOfLaundering` | 1 |
| 10 | Adapter clamps a stale validity interval into a live one | `TestPiPluginGraphBuildLaunchRefusesForgedObservationBeforePreflight` | 1 |
| 11 | Adapter launders drifted response identity into the query identity | same | 1 |
| 12 | Contract test imports `net/http` | `TestContractTestsContactNoLiveRuntime` | 1 |

Mutant 2 is the load-bearing one: it proves the bypass is closed by
**package-complete plane discovery**, not by the import denial list. A guard that
trusted a filename list would have passed with the live read shipping.

## Gates — real exit codes

| Gate | Command | Exit |
| --- | --- | ---: |
| Format | `gofmt -l tools/agents-infra` (zero output) | 0 |
| Vet | `go vet ./...` | 0 |
| Build | `go build ./...` | 0 |
| Tests — infra | `go test ./internal/infra -count=1` (280.7s) | 0 |
| Tests — main + rest | `go test . ./internal/attachments ./internal/modelharness ./cmd/... -count=1` | 0 |
| Race — new infra surface | `go test -race ./internal/infra -run 'TestPiPluginGraph\|TestProcessA\|TestConsumer\|TestObservationPlane\|TestGenericPlanes\|TestConcreteAssembly\|TestProcessBLifecycle\|TestDryRun\|TestQwenInfra\|TestPlanIsMetamorphic\|TestPiTurn\|TestPinnedPiGrammar\|TestPiExecutionEnvironment' -count=1` | 0 |
| Race — new CLI surface | `go test -race . -run 'TestSchemaOne\|TestExactProfileAssertion\|TestStandaloneCLI\|TestRunTargetQwenStandalone' -count=1` | 0 |
| Cross-platform build | `GOOS/GOARCH` ∈ {windows,linux,darwin} × {amd64,arm64} `go build ./...` | 0 (all 5) |
| Cross-platform vet | same matrix, `go vet ./...` | 0 (all 5) |
| Module integrity | `go mod verify` | 0 |
| Mutation | `.temp/TASK-260830-y6infr/mutants.sh` | 12 mutants, all killed except the intentional discovery-narrowing control |

## Explicitly not run

- **Full `go test -race ./...`.** A single shell call in this headless run is bounded
  at ~10 minutes and the unraced suite alone is ~7 minutes. Race was run against the
  complete new and changed surface (both packages), not the whole tree. The
  unraced full suite passed for every package.
- **`make vet` / `make regress`.** This repository has no Makefile; the equivalent
  commands were run directly and are listed above.
- **Independent review, PR, and merge gates (part of AC8).** Those belong to the
  review and delivery roles; this task hands off at `to-review`.
- **Any live verification.** Out of scope by the task and the ADR. All acceptance is
  static files, fake readers, and fake Process-A children.

## Ownership and activation

Process-B election, lease, restart, quarantine, and rotation remain in agents-infra
and are unreachable from either generic plane (guarded, mutant 6). Process-A
cleanup signals only its own process group.

`infra.BuildAndRunPiTurn` is the real production child-launch/result call site and
is driven end to end by the tests through a fake Process A. It ships as a library
seam: no CLI verb activates it. Per the ADR's migration sequence, activation for the
local Qwen path is gated on the stable upstream release tag and the final pin, which
the dedicated delivery task owns.

## Immutable Change Request

`CR-TASK-260830-y6infr-1`, published as board outcomes `TASK-260830-y6infr_CR-1.patch`
and `TASK-260830-y6infr_CR-1.md`.

| Field | Value |
| --- | --- |
| Base commit | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| Patch bytes | 126159 |
| Patch SHA-256 | `105c3c4d3bb8b640f5d37e7f18aebcb478401d667455e49ee27efc71f14386f7` |
| `git diff --check` | exit 0 |
| Scope | 25 files, +2324 / -43 |

Reconstruction was verified, not asserted: a detached worktree at the base commit
took the patch and rebuilt.

| Reconstruction step | Exit |
| --- | ---: |
| `git apply --check --binary` at base | 0 |
| `git apply --binary` at base | 0 |
| `go build ./...` from the snapshot | 0 |
| Contract/guard tests from the snapshot | 0 |

The verification worktree was removed afterwards (`git worktree remove` exit 0).
