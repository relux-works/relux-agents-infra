# TASK-260830-1jpse1 re-review verdict — CHANGES REQUESTED

## Verdict

Route `TASK-260830-1jpse1` to `to-dev`. The general graph architecture and consumer compatibility are substantially correct, but the new refusal/validation surface does not satisfy the task's explicit negative-evidence gate. Two compile-clean narrowed mutants survive the shipped suites and admit states the production entry points must reject.

This is ordinary implementation rework, not Stop-The-Line and not a human-only blocker.

## Candidate and ownership boundary

- Change Request: `CR-TASK-260830-1jpse1-1`, revision 1.
- Infra repository delta: empty. `git diff 5c9b4e4f7a88e1eb937b80851af522e4fa4b066f f6c67993fb17426a984262ca7e99e6f5765eecdb` has zero paths.
- Empty is the correct repository shape for this leaf: implementation is owned and released by `relux-works/skill-agents-management`; adding compensating `relux-agents-infra` code would violate the stated ownership boundary. The verdict is changes requested because owning-repo release evidence is incomplete, not because this worktree needs a non-empty patch.
- Published tag `v0.4.0` is an annotated tag whose local and current `origin` refs dereference to exact commit `96ea38fa06bf69a97c3782096cbc5e779ff13877`.

## Confirmed implementation properties

- `pkg/plugin.Registry` stores opaque string `Kind` data and contains no kind enum, allowlist, layer number, kind dispatch, or import of `pkg/inferenceengine`.
- `Declaration{ID, Kind, Dependencies}` owns edge direction. Existing tests exercise reverse system-to-vendor direction plus arbitrary `agent-environment`, `weight-artifact`, `artifact-store`, and `sidecar` kinds.
- Registration has named errors for missing dependencies, kind mismatch/unsatisfiable declarations, and cycles, and mutations are atomic.
- Existing System/Vendor compatibility adapters preserve vendor-to-system edges; the source-compatible launch surface and full exact-tag suite pass.
- `pkg/inferenceengine.Kind` is the first new kind without a registry contract edit.
- `agentic.BuildMultiNodePlan` adds typed process nodes and preserves legacy primary `Plan` fields.
- Task-board consumer commit `e4022da4` passes unchanged `internal/spawn` tests and CLI build against downloaded `github.com/relux-works/skill-agents-management@v0.4.0` when run with `-mod=mod`.

## Blocking finding 1 — same-kind duplicate refusal is unproven

Shape: **negative tests for gates / narrowing, not deleting**.

Production call site: `pkg/plugin.(*Registry).RegisterAll`, `pkg/plugin/registry.go:106`.

A compile-clean mutant narrowed duplicate refusal to only the different-kind case. Same-ID, same-kind registration then overwrote the existing plugin, while this shipped command remained green:

```text
go test ./pkg/plugin ./pkg/agentic ./pkg/vendorplugin -count=1
ok pkg/plugin
ok pkg/agentic
ok pkg/vendorplugin
```

A reviewer attack test registering `sidecar/metrics` twice then failed on that mutant:

```text
second same-kind Register error = <nil>, want ErrDuplicatePlugin
```

The current suite has no assertion naming `plugin.ErrDuplicatePlugin`; it cannot distinguish the shipped gate from this narrower, invalid one.

## Blocking finding 2 — multi-node self-cycle refusal is unproven

Shape: **negative tests for gates / narrowing, not deleting**.

Production call site: `agentic.BuildMultiNodePlan` -> `planNodeOrder`, `pkg/agentic/multinode.go:142`.

A compile-clean mutant admitted only a one-node self-cycle while preserving refusal of the tested two-node engine/sidecar cycle. The shipped command remained green:

```text
go test ./pkg/agentic -count=1
ok pkg/agentic
```

A reviewer attack test with `engine -> engine` then failed on that mutant:

```text
BuildMultiNodePlan(self-cycle) error = <nil>, want ErrPlanDependencyCycle
```

The existing cycle test proves one cycle shape only. It does not prove the stated cycle bound.

## Corroborating coverage gap

The new tests do not name these raw-graph refusal errors at all: `ErrNilPlugin`, `ErrInvalidDeclaration`, `ErrUnstableDeclaration`, `ErrDuplicatePlugin`, `ErrDuplicateDependency`, and `ErrPluginNotRegistered`. Multi-node tests likewise do not name `ErrPlanInvalid` or `ErrDuplicatePlanNode`. Not every branch needs one test per line, but every new validation class needs a negative driven through the production entry point; the two surviving mutants show the current set is insufficient in fact, not merely by count.

## Validation run directly by this reviewer

| Scope | Command | Result |
| --- | --- | --- |
| Exact tag targeted | `go test ./pkg/plugin ./pkg/inferenceengine ./pkg/agentic ./pkg/vendorplugin -count=1` | pass |
| Exact tag full | `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1` | pass after initializing Git metadata in the archive snapshot |
| Exact tag static | `make vet` | pass |
| Exact tag regression/goldens | `make regress` | pass |
| Consumer | task-board `e4022da4`: `go test -mod=mod ./internal/spawn -count=1` | pass |
| Consumer build | task-board `e4022da4`: `go build -mod=mod .` | pass |
| Duplicate narrowed mutant | shipped plugin/agentic/vendor suites | **survived (invalid green)** |
| Self-cycle narrowed mutant | shipped agentic suite | **survived (invalid green)** |
| Reviewer attack tests | duplicate same-kind; self-cycle | both fail against their mutant, proving the missing coverage |

Honest diagnostics: the first full-suite attempt failed because an archive has no `.git` and an integration test invokes `git check-ignore`; rerun after scratch-only `git init` passed. The first consumer attempt hit intentionally stale vendoring after changing only the scratch dependency version; the supported `-mod=mod` lane passed.

## Required rework

1. Add production-entry negative tests that kill both demonstrated narrowed mutants: same-ID/same-kind duplicate registration and a one-node self-cycle.
2. Cover the remaining new registration and typed-plan refusal classes with class-sensitive negative tests, especially invalid/unstable declarations, duplicate dependencies, duplicate plan nodes, and invalid process nodes. Prefer narrowed mutants over delete-only checks.
3. Record this negative-evidence gap in the owning repository's `LOGBOOK.md`; reviewer read-only constraints prohibit modifying it in this run.
4. Because `v0.4.0` is already published and must not be rewritten, release the reviewed repair as a new patch version, then repeat exact-tag full/golden checks and unchanged task-board consumer verification.
5. Keep the infra repository delta empty unless a genuine consumer change becomes necessary; do not add a workaround here.

## Non-blocking architecture note

The compatibility vendor adapter currently derives only vendor-to-system edges; docs describe a direct vendor-to-engine edge as optional. The accepted transitive order `engine -> system -> vendor` works, and the raw graph can express a direct cross-kind edge, so this note is not a separate rejection. If the product contract requires `local-models` itself to declare a direct engine edge, add an explicit source-compatible vendor dependency API during rework rather than relying on a second raw graph.
