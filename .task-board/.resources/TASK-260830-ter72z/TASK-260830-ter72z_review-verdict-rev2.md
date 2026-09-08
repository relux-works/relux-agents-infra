# TASK-260830-ter72z review verdict — CR revision 2

## Verdict

**Changes requested**; route to `to-dev`.

The rework closes the old direct `Observer` parameter and adds the required three result labels, but the central observed-or-refused invariant remains bypassable. A caller can register its own public `Engine`, return an exported `ObserveValue` built from caller data, and have `ResolveObserved` expose that value to consumers. The generic structured-value contract also admits a readiness proxy that explicitly says weights are not resident.

## Empty Change Request delta

`CR-TASK-260830-ter72z-2` revision 2 has `repository_delta=empty`. The exact Story-worktree diff from `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` to `f6c67993fb17426a984262ca7e99e6f5765eecdb` exits `0` with zero paths; both refs resolve to tree `f6c67993fb17426a984262ca7e99e6f5765eecdb`.

That emptiness is correct for this leaf: the requested specification belongs to the separate `skill-agents-management` repository. The current owning-repository delivery is PR #8 at signed rework commit `8f4a8483abefed9671eb048a2547bb3553f2950c`, not a local agents-infra delta. The empty CR is not the reason for rejection.

## Owning-repository identity and pristine validation

- GitHub PR #8 is open, non-draft, mergeable, based on `main` at `75b105291011ac8988b714a86a38cf9f56771e13`, and its remote head equals `8f4a8483abefed9671eb048a2547bb3553f2950c`.
- The rework commit parent is `4fa4e25c4c7cd314643d021dc0c8cf576c38c543`; tree is `d7804019c005e9b48a9e6642605cf56de1bb74b7`.
- Local `git verify-commit` succeeds with a good SSH signature for `alexis@relux.works`, ECDSA fingerprint `SHA256:60fPOOw38n4bW1moyoPfQ/TZIRwPFLgYf7BfR49npaU`.
- An archive-only fixture was indexed from the exact commit paths; `git write-tree` reproduced `d7804019c005e9b48a9e6642605cf56de1bb74b7`.
- `go test ./pkg/inferenceengine -count=1` passed.
- `go test -race ./pkg/inferenceengine -count=1` passed.
- `go test ./pkg/inferenceengine -cover -count=1` passed at `83.8%` statement coverage.
- `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1` passed on the exact indexed tree.
- Go formatting and whitespace checks passed.

## What matches the task

- The closed inventory includes argv spelling, reasoning stream field, health/readiness, weight artifact shape, mmap-aware memory accounting, speculative decoding, load/unload, inference-busy, memory-pressure sequencing, and model-harness local/SSH/stress/restart facts, each with task provenance.
- The result vocabulary exposes `ObservedValue`, `ObservedAbsent`, and `NotObserved`; failed reads do not fall back to absence.
- Every declared rule structurally requires `refuse` for read-failed, malformed, and unsupported derivation.
- `ModelHarnessExpansion.ExecutionOwner` is pinned to `agents-infra`; no OS process, SSH, readiness polling, or supervision implementation moved into `skill-agents-management`.
- The producer's compile-clean unsupported-refusal narrowing was reproduced. Removing only `Unsupported != refuse` made `TestResolveObservedRefusesAnIncompleteOrFallbackContract/unsupported_is_silently_dropped` exit `1` because `ResolveObserved` returned nil instead of `ErrContractInvalid`; restoring the pristine source made the same test pass.

## Blocking finding F1 — forged or self-minted canonical values still reach consumers

Standard negative shape: **forged or self-minted evidence**.

Production path:

1. `pkg/plugin.Registry.Register` (`pkg/plugin/registry.go:74-119`) publicly accepts and stores any caller implementation of `plugin.Plugin`.
2. `inferenceengine.ResolveObserved` (`pkg/inferenceengine/contract.go:280-343`) type-asserts that stored object to public `Engine` and calls its `DeriveObservation` at line 304.
3. Public `ObserveValue` (`contract.go:236-243`) accepts the engine/caller raw value, stamps metadata from the caller-supplied rule, and returns a sealed result.
4. `ResolveObserved` revalidates only matching metadata plus canonical syntax; it cannot establish that the value came from the observed process.

An external-package reviewer test used only these public entry points: it registered an `Engine` whose `DeriveObservation` returned `ObserveValue(rule, `["--ctx-size","999999"]`)`. `ResolveObserved` returned nil error and exposed that exact value. The test exited `1` with:

```text
ResolveObserved admitted caller-supplied canonical argv through a caller-registered Engine: ["--ctx-size","999999"]
```

Moving the minting API from a direct `Observer` argument to a caller-registered `Engine` changes the label, not the trust boundary.

## Blocking finding F2 — readiness semantics are not enforced

Standard negative shapes: **forged or self-minted evidence** and **property inferred from a proxy signal**.

`ValueContractCanonicalObject` (`contract.go:447-459`) accepts any non-empty canonical JSON object for health, readiness, artifact, memory, capability, load/unload, busy, pressure, forwarding, stress, and restart facts. It has no fact-specific schema or semantic validation.

A second external-package reviewer test returned this canonical object for `FactReadiness`:

```json
{"endpoint_answering":true,"weights_resident":false}
```

`ResolveObserved` accepted it and exposed it as `ObservedValue`. The test exited `1` with:

```text
ResolveObserved admitted endpoint-answering as readiness while weights were not resident
```

This is the exact measured difference the specification says must be refused: an endpoint may answer before weights are resident. The same open object grammar leaves the memory-accounting, artifact-shape, speculative-decoding, load/unload, inference-busy, and pressure-sequencing semantics equally unproven.

## Blocking finding F3 — the advertised gate has no production caller

Standard negative shape: **check present but uncalled from production**.

Repository-wide Go search at the exact tree found `ResolveObserved` only in `pkg/inferenceengine/contract.go` and tests. A corresponding search of agents-infra found no production call. The docs call it a production gate, but no composed agents-infra path invokes it, so the execution-owner boundary and derivation path are not under production-entry evidence.

## Required rework

1. Bind observation derivation to a concrete agents-infra-owned composition path that a consumer cannot replace by registering an arbitrary public `Engine`; keep all OS process, SSH, polling, and supervision execution in agents-infra.
2. Give each structured fact a closed typed/fact-specific value contract. In particular, readiness must reject endpoint-answering without weight residency; memory accounting must express the mapping-aware method; load/unload/busy/pressure must express their state and ordering rather than merely non-empty JSON.
3. Drive negative tests through the actual agents-infra composition/consumer entry point. The tests must register or supply evidence exactly as production does, then prove that canonical self-minted argv, false readiness, forged absence, read-failure-as-absence, and unsupported derivation are refused.
4. Preserve the reproduced unsupported-refusal narrowing mutant and add narrow mutants for the trusted composition gate and fact-specific readiness validation.
5. Record this repeated trust-boundary regression in the owning repository's `LOGBOOK.md`. This reviewer did not mutate either reviewed repository because the reviewer role is read-only.

This is ordinary implementation rework, not a Stop-The-Line platform or product decision.
