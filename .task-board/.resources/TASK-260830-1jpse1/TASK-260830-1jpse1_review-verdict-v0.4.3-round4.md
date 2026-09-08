# TASK-260830-1jpse1 review verdict — v0.4.3 round 4

## Verdict

**ACCEPTED.** The two-sided raw-plugin refusal enumeration is complete at the
public `v0.4.3` tag, all five narrowed mutants named across the review rounds
are killed by their production-entry negatives, and task-board still builds,
tests, and queries its authoritative board against the public module without a
module replace.

## Independent error-path enumeration

I derived this list directly from `pkg/plugin/registry.go` at public `v0.4.3`,
before comparing it with the refreshed matrix.

`Registry.RegisterAll` has exactly nine error-producing classes:

1. nil registry receiver;
2. nil or typed-nil plugin;
3. unstable declaration across the two reads;
4. invalid declaration (failed normalization or a non-normalized spelling for
   the plugin or one of its dependency refs);
5. repeated dependency ID in one declaration;
6. duplicate plugin ID against the existing graph or current batch;
7. missing dependency;
8. dependency whose registered kind cannot satisfy the declared kind;
9. dependency cycle.

`Registry.Resolve` has exactly three error-producing classes:

1. nil registry receiver;
2. invalid plugin ID returned by `normalize`;
3. normalized but unregistered plugin ID.

The dependency materialization loop adds no fourth resolution error path:
`plugins` and its declarations are private registry state, and assignment to
that state occurs only after `validateEdges` and `detectCycle` accept the whole
candidate graph. `Lookup` and `Declaration` deliberately expose boolean
absence, not additional `Resolve` error branches.

The independently derived 9 registration + 3 resolution classes are exactly
the 12 raw-plugin rows in the v0.4.3 matrix. I cannot name a missing row on
either side.

## Gate-defeat evidence

Source was downloaded with `GOWORK=off GOPROXY=https://proxy.golang.org go mod
download -json github.com/relux-works/skill-agents-management@v0.4.3` and copied
to isolated scratch cases. Each test used `-count=1`. All mutants were
compile-clean and failed on the named test:

| Narrowed gate | Named production-entry test | Result |
| --- | --- | --- |
| same-kind duplicate admitted | `TestRegisterRefusesSameKindDuplicate` | killed, exit 1 |
| multi-node self-cycle admitted | `TestBuildMultiNodePlanRefusesASelfCycle` | killed, exit 1 |
| repeated non-self dependency admitted | `TestRegisterAllRefusesRepeatedNonSelfDependency` | killed, exit 1 |
| explicit non-primary node with empty binary admitted | `TestBuildMultiNodePlanRefusesExplicitNodeWithEmptyBinary` | killed, exit 1 |
| `Resolve` nil guard narrowed to `r == nil && id == ""` | `TestNilRegistryRefusesResolution` | killed, exit 1 |

The exact public-tag affected-package baseline also passed:

```text
go test -mod=mod ./pkg/plugin ./pkg/agentic ./pkg/vendorplugin -count=1
ok github.com/relux-works/skill-agents-management/pkg/plugin
ok github.com/relux-works/skill-agents-management/pkg/agentic
ok github.com/relux-works/skill-agents-management/pkg/vendorplugin
```

I accepted, but did not represent as my own rerun, the producer's attached
full `vet`, build, `./...`, regress, race, and 37/37 mutation evidence in
`TASK-260830-1jpse1_v0.4.3-results.md` and
`TASK-260830-1jpse1_v0.4.3-mutants.tsv`.

## Public consumer and release hygiene

- Public module origin: `refs/tags/v0.4.3`, hash
  `82db6b1774fa90b4fb2943ec5375a71d4ca9b964`, checksum
  `h1:3UhUWp+bCw+vy7qmGadxcHhG+CHpdecg6Lko/IgagHg=`.
- Local annotated tag resolves to the same hash as the public Go proxy origin.
- `git verify-commit 82db6b1...` and `git verify-tag v0.4.3`, using the configured
  human ECDSA public key in an explicit scratch `allowedSignersFile`, both
  reported a good `git` signature for `alexis@relux.works`.
- In a clean detached clone at task-board consumer commit
  `e4022da4371f99b26cd6173e16e560116d85362d`, the scratch `go.mod` required
  `skill-agents-management v0.4.3` and had no replace for that module.
- `go test -mod=mod ./internal/spawn -count=1` passed in that clean clone.
- The task-board CLI built successfully. `go version -m` reports public
  `skill-agents-management v0.4.3` with the checksum above, and the built CLI
  successfully queried the authoritative board (`status:reviewing`).

An earlier archive-only consumer attempt was rejected as evidence because it
lacked `.git` for a source guard and inherited a service-tier variable. Its raw
scratch log contained session-like environment values and was removed. The
clean detached-clone rerun above is the counted result.

Rollback remains explicit and credible: pin consumers to `v0.3.0`; there is no
persisted board, runtime, limit-state, or schema migration, and published tags
are not rewritten.

## Empty Change Request delta

Revision 4 has `repository_delta=empty`, and that is correct for this leaf. The
owning implementation, tests, refusal matrix, and release live in the separate
`skill-agents-management` repository and are already published as v0.4.3. This
agents-infra Story worktree is the tracking and consumer-validation surface;
the acceptance criteria explicitly identify task-board as the constrained
consumer and require no agents-infra contract migration. Adding a local file
here would duplicate or work around the owning package instead of delivering
the scoped release. The new board outcome evidence, not a repository patch, is
therefore the intended revision-4 deliverable.
