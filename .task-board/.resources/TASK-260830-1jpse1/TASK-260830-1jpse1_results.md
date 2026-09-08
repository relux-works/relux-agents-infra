# TASK-260830-1jpse1 — implementation and release outcome

## Delivered

- Owning repository: `relux-works/skill-agents-management`.
- Reviewed signed commit: `96ea38fa06bf69a97c3782096cbc5e779ff13877`.
- Pull request: <https://github.com/relux-works/skill-agents-management/pull/3>.
- PR state: merged; recorded merge object is the exact reviewed commit (no squash/rebase rewrite).
- Release: signed annotated tag `v0.4.0`, dereferencing to `96ea38fa06bf69a97c3782096cbc5e779ff13877`.
- Module download evidence: `go mod download -json ...@v0.4.0` reported VCS origin `refs/tags/v0.4.0`, hash `96ea38f...`, sum `h1:T1N4VEoImtIZhNJITFL8oQV3ANIuvjSXabWKkYDFS6c=`.

The release adds:

1. `pkg/plugin.Registry`: kind-agnostic atomic graph registration, resolution and dependency-first ordering.
2. Named registration refusals for missing dependencies, unsatisfiable kind declarations and dependency cycles.
3. Source-compatible `agentic.Registry` and `vendorplugin.Registry` graph adapters. Existing System/Vendor implementations and registration calls are unchanged; vendor-to-system edges are derived from existing model declarations.
4. `pkg/inferenceengine.Kind` as the first new kind, without a registry contract edit.
5. `agentic.BuildMultiNodePlan` with typed primary, inference-engine and sidecar process nodes plus missing-edge/cycle validation.
6. Documentation, consumer guidance, rollback path and LOGBOOK decisions.

## Compatibility and graph semantics

- Existing vendor→system dependency meaning remains intact.
- Graph direction is declaration-owned: tests admit the reverse cross-kind direction in a separate graph.
- Pi and local-models can share one engine prerequisite in the non-cyclic order engine → system → vendor.
- A self-review negative found and fixed a same-id/same-kind shadow declaration that narrowed a system's engine dependencies; production `vendorplugin.Registry.Register` now requires exact declaration equality during compatibility sync.
- Existing six original system/four vendor regression registration paths and launch goldens remain green. The already-shipped Pi and conditional local-models suites also remain green.

## Release order and rollback

Executed order:

1. Implement additive graph and plan APIs behind compatibility adapters.
2. Run owning-repository tests, goldens, race, vet, build and regress.
3. Build and test task-board unchanged against the exact candidate source.
4. Publish reviewed signed commit to main and signed `v0.4.0` tag.
5. Download the public `@v0.4.0` module and repeat task-board spawn/build/query verification against the tag.

Rollback: consumers pin `v0.3.0`; v0.4.0 migrates no board, runtime or limit-state data. Published tags are immutable. Any repair ships as a later signed patch version. Native task-board graph adoption remains separately owned by `TASK-260830-mf1xwk`.

## Green validation evidence

Every command below ran directly and exited 0:

| Scope | Command | Exit |
| --- | --- | ---: |
| Graph TDD | `go test -mod=mod ./pkg/plugin ./pkg/agentic ./pkg/inferenceengine ./pkg/vendorplugin -count=1` | 0 |
| Race | `go test -race -mod=mod ./pkg/plugin ./pkg/agentic ./pkg/inferenceengine ./pkg/vendorplugin -count=1` | 0 |
| Lint/static | `make vet` | 0 |
| Build | `make build BIN=.temp/TASK-260830-1jpse1/agents-management` | 0 |
| Full suite | `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1` | 0 |
| Regression/goldens | `make regress` | 0 |
| Formatting/diff | `test -z "$(gofmt -l pkg internal tools)" && git diff --check` | 0 |
| Candidate consumer | task-board `internal/spawn` at consumer `e4022da4`, exact local candidate selected | 0 |
| Candidate consumer build | task-board CLI against exact local candidate | 0 |
| Released module download | `go mod download github.com/relux-works/skill-agents-management@v0.4.0` | 0 |
| Released consumer | task-board `internal/spawn` against downloaded `@v0.4.0` | 0 |
| Released consumer build | task-board CLI against downloaded `@v0.4.0` | 0 |
| Released consumer runtime | released-tag-built task-board read-only query on this task | 0 |
| Commit signature | local SSH `git verify-commit` for `96ea38f...` | 0 |
| Tag signature | local SSH `git verify-tag v0.4.0` | 0 |

GitHub reports the commit's SSH signature as `unknown_key` because that public key is not registered as a GitHub *signing* key, while local cryptographic verification against the configured human public key succeeds. No unsigned fallback was used.

## Honest red/diagnostic evidence

- Initial graph tests before production files existed: exit 1, expected TDD red.
- Shadow-declaration narrowing test before the exact-declaration fix: exit 1, expected mutant red; it demonstrated the bypass and passed after the production fix.
- First task-board compatibility attempt: exit 1 because ambient `TASK_BOARD_CODEX_SERVICE_TIER=default` polluted a consumer test; rerun with only that ambient variable removed passed.
- Workspace compatibility attempts: exit 1 because consumer test helpers invoke `go list -mod=mod`, which Go refuses under workspace mode. Validation switched to a temporary detached-worktree `replace`, then to the public tag; both supported modes passed.
- Initial `git commit -S`: exit 128 because Git defaulted to unavailable GPG material. No commit was created. The release used the configured SSH key and verified successfully.
- Process-substitution signature verification: exit 1 because Git could not consume the ephemeral allowed-signers path. Verification through a task-scoped allowed-signers file passed.

## Workspace note

The board Story worktree belongs to `relux-agents-infra`, while the authorized owning repository is `skill-agents-management`. Work was isolated in a task-scoped owning-repo worktree under this Story's `.temp/`; the sibling main checkouts and their dirty board state were not modified. The owning-repo release is already landed, so the agents-infra Change Request is expected to have an empty repository delta and this outcome resource is the durable handoff evidence.
