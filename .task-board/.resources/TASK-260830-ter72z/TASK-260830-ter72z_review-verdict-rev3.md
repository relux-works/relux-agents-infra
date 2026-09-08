# TASK-260830-ter72z review verdict — CR revision 3

## Verdict

**Changes requested**; route to `to-dev`.

Revision 3 structurally closes the revision-2 caller-minted observation path and
corrects the documentation so `ResolveContract` is not presented as a runtime
production gate. It also rejects the named readiness, mmap-accounting, lifecycle,
busy-state, and pressure-ordering attacks. However, two supposedly closed
fact-specific JSON schemas still treat absent evidence as a valid explicit zero
value. The specification therefore does not yet satisfy the closed-contract
requirement.

## Empty Change Request delta

`CR-TASK-260830-ter72z-3` revision 3 has `repository_delta=empty`. The exact
agents-infra base and candidate resolve to the same tree
`f6c67993fb17426a984262ca7e99e6f5765eecdb`, and the diff has zero paths.

That emptiness is correct for this leaf. The specification belongs to the
separate `skill-agents-management` repository, and the reviewed delivery is PR
#8 at head `58a063c72ffc298023219d660f1cf7167bc2cb59`. The empty agents-infra CR is
not a review finding.

## Owning-repository identity and validation

- GitHub PR #8 is open, non-draft, mergeable, based on `main` at
  `75b105291011ac8988b714a86a38cf9f56771e13`; the remote PR head is exactly
  `58a063c72ffc298023219d660f1cf7167bc2cb59`.
- The reviewed commit tree is
  `8d680445a54473f76c5f8b4ee65357cbfdb9b749`; an alternate-index
  materialization reproduced that exact tree.
- `git verify-commit` cryptographically reports a good SSH signature with the
  same ECDSA fingerprint verified for revision 2. This checkout has no
  `allowedSignersFile`, so it cannot independently bind a principal; GitHub
  likewise reports `unknown_key`. No unsigned or changed head was substituted.
- On an isolated Git checkout of that exact commit:
  `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`, `make vet`,
  `make regress`, and `make build BIN=../agents-management` all passed.
- `go test ./pkg/inferenceengine -count=1` and
  `go test -race ./pkg/inferenceengine -count=1` passed.
- `make contract-mutants` killed all three compile-clean narrowed mutants:
  `trusted-composition-interface`, `readiness-requires-residency`, and
  `unsupported-refusal`.
- An initial full-suite run in an archive-only fixture failed only because the
  build integration test correctly detected the fixture had no Git index. The
  rerun in the exact-commit Git checkout passed; this was a reviewer-fixture
  failure, not a candidate failure.

## F1 — caller-minted evidence: closed

Standard negative shapes: **forged or self-minted evidence** and **bypass path
around the check**.

The reviewer reproduced the revision-2 public-entry attack by registering an
engine with an extra `DeriveObservation(Fact) string` method returning canonical
`["--ctx-size","999999"]`. `ResolveContract` called the method zero times and
returned no result/value/observation field.

A second public-entry attack wrapped/delegated to a legitimate engine contract,
canonicalized a value through public `ValidateCandidateValue`, and exposed that
value from the wrapper's extra derivation method. Neither wrapper nor delegate
derivation ran, and the value did not appear in the resolved declaration.
Shape validation remains public, but the code and docs now correctly say it is
not provenance or admission.

## F2 — proxy facts: partially closed, one blocking negative remains

The reviewer independently confirmed these refusals through
`ValidateCandidateValue`:

- readiness proxy `endpoint_answering=true, weights_resident=false`;
- mmap weights with naive `mach-physical-footprint` accounting;
- load and busy presence without their required state/transition semantics;
- pressure ordering that skips unload consultation.

Blocking finding — standard negative shape: **absent evidence treated as
satisfied**.

`decodeStrict` forbids unknown fields but cannot distinguish an omitted ordinary
Go zero-valued field from an explicitly supplied zero value. Two public-entry
attacks therefore succeeded with nil errors:

```text
FactSpeculativeDecoding input {"capability":"supported"}
=> {"capability":"supported","active":false}

FactRestartSupervisionPolicy input
{"mode":"bounded-backoff","backoff_seconds":[1,2,4]}
=> {"mode":"bounded-backoff","max_restarts":0,"backoff_seconds":[1,2,4]}
```

The first invents the required runtime-observed active state; the second invents
the required restart bound. Both schemas claim those fields as part of the
fact-specific contract, and the restart validator's own error says
`max_restarts` is required. Absence is being laundered into a legitimate fact.

Production/public call site attacked:
`pkg/inferenceengine.ValidateCandidateValue`, implemented in
`pkg/inferenceengine/contract.go` around lines 351-359; the affected plain value
fields are around lines 524-527 and 565-568.

## F3 — uncalled production gate: closed by honest scope

Exact-tree searches in both `skill-agents-management` and `agents-infra` found
no production caller of `ResolveObserved`, `ResolveContract`, or
`ValidateCandidateValue`. `ResolveObserved` is removed except for historical
LOGBOOK entries. The package, README, SKILL, architecture, consuming guide, and
shipped-state docs state that runtime observation admission is not implemented
in this revision and that `ResolveContract` validates declaration only. This is
the correct outcome for a specification leaf and no longer advertises an
uncalled production guard.

## Ownership and required measured knobs

- All 17 measured facts are present with task provenance: exact context/prefill
  argv spelling, reasoning stream field, health/readiness, artifact shape,
  mapping-aware memory, speculative decoding, load/unload, inference busy,
  pressure sequencing, and model-harness local executable/argv, SSH forwarding,
  stress, and restart policy.
- Every rule requires refusal for read failure, malformed evidence, and an
  unsupported derivation.
- `ModelHarnessExpansion.ExecutionOwner` and the package constant remain
  exactly `agents-infra`. No OS process, SSH, readiness polling, or supervision
  execution moved into `skill-agents-management`.

## Required rework

1. Make every semantically required field distinguishable from absence. At
   minimum, reject omitted `speculative-decoding.active` and omitted
   `restart-supervision.max_restarts`; audit the other structured schemas for
   valid zero values that omission can currently synthesize.
2. Add external-package negative tests through `ValidateCandidateValue` for
   omitted required fields. Name the **absent evidence treated as satisfied**
   shape.
3. Add a compile-clean narrowed mutant that disables the required-key check for
   at least one zero-valid field and require the named test to fail.
4. Record this revision-3 missing-field regression in the owning repository's
   `LOGBOOK.md`. The reviewer did not modify either repository because the
   reviewer role is read-only.

This is ordinary implementation rework, not a Stop-The-Line boundary.
