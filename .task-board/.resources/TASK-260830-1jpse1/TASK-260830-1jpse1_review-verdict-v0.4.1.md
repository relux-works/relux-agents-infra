# TASK-260830-1jpse1 review verdict — v0.4.1 rework

## Verdict

**CHANGES REQUESTED** — route `TASK-260830-1jpse1` to `to-dev`.

The v0.4.1 rework kills the two narrowed mutants that caused the previous
rejection, but the new plugin and multi-node validation surface still does not
satisfy the task's negative-evidence gate. Two additional compile-clean
narrowed mutants survive the shipped package suites and admit states the
production entry points explicitly classify as invalid.

This is ordinary implementation/test rework, not Stop-The-Line and not a
human-only blocker.

## Candidate and empty repository delta

- Change Request: `CR-TASK-260830-1jpse1-2`, revision 2.
- Infra candidate: base `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`,
  candidate tree `f6c67993fb17426a984262ca7e99e6f5765eecdb`, zero
  changed paths.
- The empty `relux-agents-infra` delta is correct for this leaf. The authorized
  implementation owner is `relux-works/skill-agents-management`; adding an
  infra workaround would violate that ownership boundary.
- Exact owning-repo tag `v0.4.1` dereferences to
  `b40c415620f0a5d359e2322455b05fcd31f1870c`, parented directly on v0.4.0.
  Its repository delta is tests and release documentation only; v0.4.0
  production graph behavior is unchanged.

## Confirmed rework

The two previously demonstrated gates now have class-sensitive tests:

- Narrowing duplicate rejection to different-kind collisions makes
  `TestRegisterRefusesSameKindDuplicate` fail with `error = <nil>`.
- Admitting only a direct multi-node self edge makes
  `TestBuildMultiNodePlanRefusesASelfCycle` fail with `error = <nil>` while
  retaining the broader cycle guard.

The exact unmutated v0.4.1 targeted packages also pass:

```text
go test -mod=mod ./pkg/plugin ./pkg/agentic ./pkg/inferenceengine ./pkg/vendorplugin -count=1
ok pkg/plugin
ok pkg/agentic
ok pkg/inferenceengine
ok pkg/vendorplugin
```

## Blocking finding 1 — repeated dependency refusal remains unproven

Shape: **negative tests for gates / narrowing, not deleting**.

Production call site: `plugin.Registry.Register` / `RegisterAll` ->
`validateDeclaration`, `pkg/plugin/registry.go:164`.

A compile-clean mutant narrowed `ErrDuplicateDependency` to repeated
self-dependencies only. Ordinary repeated edges to another registered plugin
were then accepted. The complete shipped plugin package suite remained green:

```text
go test -mod=mod ./pkg/plugin -count=1
ok pkg/plugin
```

A reviewer attack through `Registry.RegisterAll` then observed the bypass:

```text
RegisterAll(repeated dependency) error = <nil>, want ErrDuplicateDependency
```

No shipped test names `plugin.ErrDuplicateDependency`, so the suite cannot
distinguish the intended gate from this strictly smaller one.

## Blocking finding 2 — invalid explicit process nodes remain unproven

Shape: **negative tests for gates / narrowing, not deleting**.

Production call site: `agentic.BuildMultiNodePlan` -> `validatePlanNode`,
`pkg/agentic/multinode.go:120`.

A compile-clean mutant retained empty-binary refusal for the implicit primary
node but admitted an explicit inference-engine/sidecar node with no binary. The
complete shipped agentic package suite remained green:

```text
go test -mod=mod ./pkg/agentic -count=1
ok pkg/agentic
```

A reviewer attack through `BuildMultiNodePlan` then observed the bypass:

```text
BuildMultiNodePlan(empty explicit binary) error = <nil>, want ErrPlanInvalid
```

This is directly on the new engine/sidecar launch-plan surface. No shipped test
names `agentic.ErrPlanInvalid`.

## Corroborating audit gap

Exact-tag test search finds no production-entry tests naming
`plugin.ErrNilPlugin`, `plugin.ErrInvalidDeclaration`,
`plugin.ErrUnstableDeclaration`, `plugin.ErrDuplicateDependency`,
`plugin.ErrPluginNotRegistered`, or `agentic.ErrPlanInvalid`. One test per
branch is not required, but every new refusal class needs a class-sensitive
negative. The two surviving narrowed mutants establish a real gap rather than
a coverage-count preference.

The v0.4.1 board outcome reports nine killed narrowed mutants, while the
owning-repo `LOGBOOK.md` entry reports eight. Rework must reconcile the factual
count and record this newly demonstrated gap in that owning-repo logbook.

## Required rework

1. Add production-entry negatives that kill the demonstrated repeated-edge
   mutant and explicit-node-empty-binary mutant.
2. Complete the promised audit of the remaining raw plugin and typed-plan
   refusal classes with narrowed, class-sensitive mutants; do not rely on
   positive-path or delete-only evidence.
3. Record the finding and corrected mutant count in the owning repository's
   `LOGBOOK.md`. This reviewer did not edit it because the reviewer role is
   read-only.
4. Do not rewrite published `v0.4.1`; release the reviewed repair as a new patch
   version and repeat exact-tag full/golden checks plus unchanged task-board
   consumer test/build/query verification.
5. Keep the infra repository delta empty unless a genuine consumer change is
   required.

## Evidence scope

Commands were run against a `git archive v0.4.1` scratch snapshot under the
Story worktree. Mutants and reviewer-only attack tests lived exclusively under
`.temp/TASK-260830-1jpse1/`; neither repository's tracked files were modified.
The producer's attached full/race/vet/regress/consumer results were inspected;
this reviewer independently reran the exact-tag targeted package suite and the
four bounded mutant checks described above.
