# TASK-260830-1jpse1 — refusal-proof rework and v0.4.1 release

## Delivered

- Owning repository: `relux-works/skill-agents-management`.
- Signed commit: `b40c415620f0a5d359e2322455b05fcd31f1870c`, parented directly on v0.4.0.
- Pull request: <https://github.com/relux-works/skill-agents-management/pull/4>; complete remote diff matched the local diff byte-for-byte, comment-review verdict accepted the exact head, and GitHub recorded the exact reviewed commit as the merge commit. The repository had no configured status checks.
- Release: signed annotated tag `v0.4.1`, dereferencing to `b40c415620f0a5d359e2322455b05fcd31f1870c`.
- Public module: `go mod download -json github.com/relux-works/skill-agents-management@v0.4.1` resolved `refs/tags/v0.4.1` to the same commit with sum `h1:lAbFB8O/x04leOFMJljKdAGc4byTSS10inPsgiztpaY=`.

This rework changes tests and release documentation only. The accepted v0.4.0 production graph implementation is unchanged.

## Refusal bounds added

Production entry points now carry negative coverage for:

1. `plugin.Registry.Register`: same-id, same-kind `sidecar/metrics` duplicate returns `plugin.ErrDuplicatePlugin` and retains the first registration.
2. `plugin.Registry.RegisterAll`: an identical duplicate inside one batch returns `ErrDuplicatePlugin` atomically.
3. `plugin.Registry.Register` / `RegisterAll`: later missing dependencies and later arbitrary-kind mismatches are refused, including targets introduced by the same batch.
4. `plugin.Registry.RegisterAll`: self cycles and three-node cycles return `ErrDependencyCycle`; refused batches neither add new nodes nor disturb a pre-existing graph.
5. `agentic.Registry.RegisterWithDependencies`: missing and kind-mismatched prerequisites propagate the graph's named errors without publishing the compatibility system.
6. `vendorplugin.Registry.Register` -> `syncAgenticGraph`: equal-width shadows with a different dependency or kind are refused by the compatibility-sync check itself, not accidentally by a downstream graph check.
7. `agentic.BuildMultiNodePlan`: missing dependencies are refused from the implicit primary node and from later edge positions; duplicate explicit/implicit-primary node IDs are refused.
8. `agentic.BuildMultiNodePlan` -> `planNodeOrder`: `engine -> engine` and a three-node cycle return `ErrPlanDependencyCycle`.

## Mutation evidence

Nine compile-clean narrowed mutants were run with `-count=1`; each corresponding named test returned exit 1:

| Narrowed mutant | Named proof | Exit |
| --- | --- | ---: |
| Duplicate refusal limited to different-kind collisions | `TestRegisterRefusesSameKindDuplicate` | 1 |
| Multi-node DFS admits a direct self edge but still refuses the existing two-node cycle | `TestBuildMultiNodePlanRefusesASelfCycle` | 1 |
| Plugin edge validation checks only dependency index 0 | later missing/kind-mismatch tests | 1 |
| Kind mismatch refusal limited to the historical model-vendor actual kind | `TestRegisterAllRefusesALaterKindMismatchAgainstABatchNode` | 1 |
| Atomic assignment leaks a rejected batch only when the registry was already non-empty | `TestRegisterAllPreservesExistingGraphWhenANewBatchIsRefused` | 1 |
| Compatibility sync compares dependency count only | equal-width shadow test | 1 |
| Multi-node missing check skips the implicit primary node | primary-node missing-dependency subtest | 1 |
| Multi-node missing check inspects only dependency index 0 | later-dependency subtest | 1 |
| Plugin graph DFS admits a direct self edge while retaining two-node refusal | plugin self-cycle subtest | 1 |

One preliminary self-cycle probe used a predicate that did not actually admit the tested edge because the implicit primary node was already on the DFS stack; that diagnostic returned exit 0 and was not counted as a killed mutant. The corrected compile-clean self-edge narrowing above returned exit 1. Every production file was restored through the inverse patch before green validation.

## Green validation

Every command below ran directly and returned exit 0:

| Scope | Command/result | Exit |
| --- | --- | ---: |
| Baseline | `go test ./pkg/plugin ./pkg/agentic ./pkg/vendorplugin -count=1` | 0 |
| Targeted after rework | same targeted suite | 0 |
| Static validation | `make vet` | 0 |
| Build | `make build BIN=.temp/TASK-260830-1jpse1-build/agents-management` | 0 |
| Full suite and launch goldens | `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1` | 0 |
| Regression net | `make regress` | 0 |
| Targeted race | `env -u TASK_BOARD_DIR go test -race -mod=mod ./pkg/plugin ./pkg/agentic ./pkg/vendorplugin -count=1` | 0 |
| Formatting/diff | `gofmt` plus `git diff --check` | 0 |
| Candidate consumer | task-board `internal/spawn` at `e4022da4` against the exact local candidate | 0 |
| Candidate consumer build/query | task-board CLI build and read-only task query against the exact candidate | 0 |
| Commit signature | local SSH `git verify-commit b40c415...` | 0 |
| Tag signature | local SSH `git verify-tag v0.4.1` | 0 |
| Public module download | `@v0.4.1` from `proxy.golang.org` | 0 |
| Released consumer | task-board `internal/spawn` at `e4022da4` against downloaded `@v0.4.1` | 0 |
| Released consumer build/query | task-board CLI build and read-only task query against downloaded `@v0.4.1` | 0 |

GitHub reports the commit SSH signature as `unknown_key` because the public key is not registered as a GitHub signing key. Local cryptographic verification against the configured human public key passed for both commit and tag; no unsigned fallback was used.

## Version and rollback

`v0.4.1` is the refusal-proof test patch over v0.4.0 and preserves the complete v0.4.0 API and behavior surface. Task-board still compiles, tests and queries unchanged against it. Rollback remains a consumer pin to `v0.3.0`; neither v0.4.0 nor v0.4.1 migrates board, runtime or limit-state data. Published tags remain immutable. Native task-board graph adoption remains separately owned by `TASK-260830-mf1xwk`.
