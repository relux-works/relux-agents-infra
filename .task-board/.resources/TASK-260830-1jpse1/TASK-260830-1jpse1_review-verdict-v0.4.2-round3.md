# TASK-260830-1jpse1 review verdict — v0.4.2 round 3

## Verdict

**CHANGES REQUESTED** — route `TASK-260830-1jpse1` to `to-dev`.

The four required narrowed mutants die and the public release/consumer checks pass, but the claimed exhaustive refusal mapping omits one error-producing production path in its explicitly stated raw registration/resolution scope. A compile-clean narrowed mutant admits that path while the complete shipped package suite remains green.

This is ordinary implementation/test rework, not Stop-The-Line and not a human-only blocker.

## Candidate and empty repository delta

- Change Request: `CR-TASK-260830-1jpse1-3`, revision 3.
- Infra candidate: base `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree `f6c67993fb17426a984262ca7e99e6f5765eecdb`, zero changed paths.
- The empty `relux-agents-infra` delta is correct for this leaf. Implementation, release and rework belong to `relux-works/skill-agents-management`; an infra workaround would violate the stated ownership boundary.
- The verdict is changes requested because the owning-repository refusal proof is incomplete, not because this Change Request needs an infra file change.

## Completeness audit

I enumerated the error values independently from the public `v0.4.2` source and compared them to the 35-row matrix rather than treating row count as proof.

- `plugin.Registry.Register` / `RegisterAll`: anonymous nil-registry error plus `ErrNilPlugin`, `ErrInvalidDeclaration`, `ErrUnstableDeclaration`, `ErrDuplicatePlugin`, `ErrDuplicateDependency`, `ErrMissingDependency`, `ErrUnsatisfiableDeclaration`, and `ErrDependencyCycle`. All are represented.
- `plugin.Registry.Resolve`: anonymous nil-resolve error, `ErrInvalidDeclaration`, and `ErrPluginNotRegistered`. `ErrInvalidDeclaration` and `ErrPluginNotRegistered` are represented; the nil-resolve error/path is absent.
- `agentic.BuildMultiNodePlan`: `ErrPlanInvalid`, `ErrDuplicatePlanNode`, `ErrPlanDependencyMissing`, and `ErrPlanDependencyCycle`. All are represented.
- `vendorplugin.Registry.Register`, including model validation, compatibility sync and graph registration: the reachable vendor/model/plugin error values are represented. The source explains the graph errors that cannot be isolated through valid public vendor input.

### Blocking finding — nil `Resolve` refusal is outside the claimed exhaustive matrix

Shape: **negative tests for gates / narrowing, not deleting / production call site**.

Production call site: `pkg/plugin.(*Registry).Resolve`, `pkg/plugin/registry.go:253`.

The matrix and `docs/shipped-state.md` explicitly claim exhaustive coverage of raw plugin registration **and resolution**, but the nil-receiver guard in `Resolve` has no row and no shipped negative. The matrix's `plugin-nil-registry` row attacks `Register -> RegisterAll`; it does not drive `Resolve`, whose guard and returned error are separate.

I narrowed only the `Resolve` guard from:

```go
if r == nil {
```

to:

```go
if r == nil && id == "" {
```

This preserves refusal for one nil-receiver input while admitting `(*plugin.Registry)(nil).Resolve("missing-engine")`. The complete shipped plugin suite remained green:

```text
go test -mod=mod ./pkg/plugin -count=1
ok github.com/relux-works/skill-agents-management/pkg/plugin
```

This is the missing row requested by the round-3 brief. It disproves the completeness claim without disputing any of the 35 present rows.

## Required four-mutant attack

The public-tag harness was inspected and rerun with `-count=1`. Each named production-entry negative failed with exit 1 against a compile-clean narrowing:

| Mutant | Named negative | Result |
| --- | --- | --- |
| Same-kind duplicate admitted | `TestRegisterRefusesSameKindDuplicate` | killed |
| Multi-node direct self-cycle admitted | `TestBuildMultiNodePlanRefusesASelfCycle` | killed |
| Repeated non-self dependency admitted | `TestRegisterAllRefusesRepeatedNonSelfDependency` | killed |
| Explicit process node with empty binary admitted | `TestBuildMultiNodePlanRefusesExplicitNodeWithEmptyBinary` | killed |

The complete shipped matrix also reported `35/35` killed. That confirms every present row, but does not cover the missing `Resolve` row above.

## Public release and consumer verification

- Public tag `v0.4.2` cloned at commit `59d5f30980323ae16d103b8298294cf9fcb2e52d`; annotated tag object `f3340a27d78db83db7e9feb55bbc781ce0e43bb6` peels to that commit.
- `go mod download -json github.com/relux-works/skill-agents-management@v0.4.2` resolved origin hash `59d5f309...` and sum `h1:n90cJuKafOZNXrTlCWn5PYpKplJEWtd6yG410iC7I00=`.
- Independent `ssh-keygen -Y check-novalidate -n git` checks validated both commit and tag signatures with the same ECDSA fingerprint. GitHub reports `unknown_key`, consistent with the public key not being registered there; this does not invalidate the cryptographic signatures.
- Rollback is stated as pinning consumers to `v0.3.0`.
- Task-board consumer commit `e4022da4371f99b26cd6173e16e560116d85362d` required public `v0.4.2` with no replace for `skill-agents-management`.
- `env -u TASK_BOARD_CODEX_SERVICE_TIER -u TASK_BOARD_DIR go test -mod=mod ./internal/spawn -count=1`: pass.
- `go build -mod=mod`: pass.
- Candidate task-board binary queried the authoritative board successfully and read `TASK-260830-1jpse1` at `reviewing`.
- Honest diagnostic: the first consumer test run inherited `TASK_BOARD_CODEX_SERVICE_TIER=default` from the reviewer session and failed the isolation test that requires this variable absent for a Claude child. The clean-environment rerun passed; this was an ambient test-harness contaminant, not a v0.4.2 failure.

## Other direct validation

- Public exact-tag `go test -mod=mod ./... -count=1`: pass, including launch-surface goldens.
- `make vet`: pass.
- `make regress`: pass.

## Required rework

1. Add a production-entry negative for nil `(*plugin.Registry).Resolve`, with a non-empty requested ID, and add the corresponding compile-clean narrowed mutant to the matrix.
2. Reconcile the advertised count and `docs/shipped-state.md` claim with the expanded mapping.
3. Record this round-3 finding in the owning repository's append-only `LOGBOOK.md`; this reviewer did not modify tracked files because the reviewer role is read-only.
4. Do not rewrite `v0.4.2`; publish the proof repair as a later patch and repeat public-tag plus unchanged task-board consumer verification.
5. Keep the infra repository delta empty unless a genuine consumer change becomes necessary.

## Evidence scope

All source inspection, public-tag checkout, mutants and consumer work occurred under `.temp/review-TASK-260830-1jpse1/`. No tracked repository file was modified. The `logbook` skill required recording the finding, but reviewer read-only constraints prohibit editing the owning repository; the explicit producer action above carries that requirement forward.
