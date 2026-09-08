# TASK-260830-ter72z review verdict — CR revision 4

## Verdict

**Accepted** at exact owning-repository head
`37dc5c0b2abd82cd369042daab9087dfc845e880`.

Revision 4 closes the revision-3 **absent evidence treated as satisfied**
finding as a class, preserves legitimate explicit zero values, and does not
reopen the earlier **forged or self-minted evidence** path. The implementation
matches the specification acceptance criteria and ownership boundary.

## Empty Change Request delta

`CR-TASK-260830-ter72z-4` has `repository_delta=empty`. The agents-infra base
and candidate trees are both
`f6c67993fb17426a984262ca7e99e6f5765eecdb`; the exact diff has zero paths.

That emptiness is correct. This leaf's specification and implementation live in
the separate `skill-agents-management` repository. No agents-infra repository
file was supposed to change, so an empty agents-infra delta is the expected
delivery shape rather than missing work.

## Exact head, signature, and remote state

- Local and remote branch head: `37dc5c0b2abd82cd369042daab9087dfc845e880`.
- PR #8 still names that exact head and is non-merged. The PR was closed after
  publication as superseded by PR #9, which merged as `b74f758` and was released
  as signed tag `v0.5.0`.
- No `v0.4.4` remote tag exists. The proposed tag was not published before this
  review; its release plan has been superseded and this acceptance does not
  authorize publishing an older release after `v0.5.0`.
- `git verify-commit` against the existing task-scoped allowed-signers evidence
  reports a good `git` SSH signature for `alexis@relux.works`, ECDSA fingerprint
  `SHA256:60fPOOw38n4bW1moyoPfQ/TZIRwPFLgYf7BfR49npaU`.
- Author and committer remain `alexis <alexis@relux.works>`.

The PR supersession is a delivery-state anomaly, not a defect in the exact
candidate handed to this reviewer. The branch remains available at the reviewed
OID, and no stale or substituted head was tested.

## Independent public-entry attacks

The reviewer built an external-package scratch probe and drove the public entry
points rather than calling internal helpers.

### Required-field presence and null — closed as a class

`inferenceengine.ValidateCandidateValue` refused all 38 required fact-field
omissions and all 38 explicit `null` substitutions. Every refusal was
`ErrObservationMalformed` and named both the fact and the missing/null field.

The matrix included fields not named by prior reviews, including
`health.process_alive` and `profile-ssh-forwarding.remote_port`; both refused by
fact and field. This rules out a patch limited to the two demonstrated
instances.

The same 38-occurrence omission matrix run against parent
`58a063c72ffc298023219d660f1cf7167bc2cb59` reproduced exactly two nil-error
admissions:

- `speculative-decoding.active`
- `profile-restart-supervision-policy.max_restarts`

Thus the producer's claim that only those two fields previously admitted
omission reproduced independently.

### Explicit zero remains valid

Through the same public entry point:

- `speculative-decoding` accepted explicit `active:false` and refused omission;
- `profile-restart-supervision-policy` accepted explicit `max_restarts:0` and
  refused omission.

The fix distinguishes absence from a valid zero instead of banning zero values.

### Caller-minted observation remains refused

The reviewer registered a custom public `Engine`, canonicalized
`["--ctx-size","999999"]` through `ValidateCandidateValue`, and exposed it from
an extra `DeriveObservation` method. `plugin.Registry.Register` succeeded,
`ResolveContract` resolved the declaration, the derivation method was called
zero times, and the resolved type exposed no result/value/observation field.

Production/public call sites attacked:

- `pkg/inferenceengine.ValidateCandidateValue`
- `pkg/plugin.Registry.Register`
- `pkg/inferenceengine.ResolveContract`

## Contract scope and ownership

- The closed inventory contains all 17 measured facts with task provenance:
  exact context/prefill argv, reasoning stream field, health/readiness, weight
  shape, mapping-aware memory, speculative decoding, load/unload, busy state,
  pressure sequencing, and local executable/argv versus SSH forwarding with
  stress/restart policy.
- Every rule fixes `refuse` for read failure, malformed evidence, and an engine
  that cannot express the fact; there is no silent-drop branch.
- `ModelHarnessExpansion.ExecutionOwner` and the package constant remain exactly
  `agents-infra`.
- The candidate Go delta adds structured shape validation and tests only. It
  adds no OS process, signal, SSH, readiness polling, or supervision execution.
- Exact-tree production-symbol searches found no caller of `ResolveContract`,
  `ValidateCandidateValue`, or removed `ResolveObserved` in either repository.
  Candidate docs explicitly say the contract is unconsumed and
  `ResolveContract` is not a production gate.

## Validation on exact head

- Independent scratch public-entry head probe: pass.
- Independent parent omission replay: pass; exactly the two claimed pre-fix
  admissions reproduced.
- `go test -mod=mod ./pkg/inferenceengine -count=1`: pass.
- `go test -mod=mod -race ./pkg/inferenceengine -count=1`: pass.
- `go test -mod=mod -cover ./pkg/inferenceengine -count=1`: pass, 94.5%.
- `python3 .scripts/verify-inference-engine-contract.py`: pass; 5/5 narrowed,
  compile-clean mutants killed by named test failures.
- `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`: pass.
- `env -u TASK_BOARD_DIR go test -mod=mod -race ./... -count=1`: pass.
- `make vet`: pass.
- `make build BIN=.temp/TASK-260830-ter72z/reviewer-agents-management`: pass.
- `make regress`: pass.
- `git diff --check`: pass.

Mutation failures were behavioral, not compile failures:

- narrowed trusted composition failed `TestEngineTrustBoundaryIsDeclarationOnly`;
- narrowed readiness failed `TestReadinessRefusesEndpointAnsweringWithoutResidentWeights`;
- narrowed unsupported refusal failed the named silent-drop test;
- exempting only `active` or only `max_restarts` failed the corresponding
  explicit-zero-versus-omission subtest.

No repository code or documentation was modified by this reviewer. Scratch
probes and logs remained under `.temp/`.
