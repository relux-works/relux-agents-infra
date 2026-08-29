# TASK-260825-imfmgz review verdict — Change Request revision 9

Reviewer run: `RUN-260825-dab008`
Change Request: `CR-TASK-260825-imfmgz-9`, revision `9`
Verdict: **accepted**

## Decision

Revision 9 closes review `RUN-260825-86b7d5` F1 and its five rework items.
No material correctness, architecture, traceability, or evidence defect remains.
The specification is implementation-ready for `TASK-260825-lsojra`, and the
acceptance leaf `TASK-260825-1lc8o7` retains the exact two-independent-spawn
proof required by section 13.

## Empty repository delta

The Change Request reports `repository_delta=empty`, and the exact CR diff from
base `77ab2870902ca1dbc52d3aa6b82f03d0c9f2fc77` to candidate tree
`bd8210d766a6848baac390d6e3fcdc299dc7ed63` has zero paths. This is the correct
outcome for this revision's CR, not missing delivery: the producer had already
committed the revision-9 specification and logbook record as `77ab287` before
the CR snapshot, so that commit is the CR base. The task's deliverable is the
task-scoped specification outcome; its attached payload, the committed
`.research/260825_shared-local-runtime-broker.spec.md`, and both downstream
precondition copies are byte-identical at SHA-256
`964148a9a2b77c89b8d9e2abcfd91c38c4dcb9613598d43daf71fa6453202dbb`.

For audit, the actual revision-8 to revision-9 repository change is commit
`77ab287` over `60a0dc0`: only the specification and `LOGBOOK.md` changed
(215 insertions, 54 deletions), and `git diff --check 60a0dc0..77ab287` passed.
No repository mutation is missing from the empty revision-9 candidate.

## Gate-defeat evidence

1. Extracted `TASK-260825-imfmgz_probe-rev9-module.tar.gz` into a new empty
   directory and ran, with the test cache cleared:

   ```text
   GOPROXY=off GOFLAGS=-mod=vendor go test -count=1 -v ./...
   PASS
   ok taskboard.local/imfmgz/rev8probe 189.772s
   ```

   This independently reproduces the self-contained/offline property. P23.A
   checks the independently generated 398-frame oracle differential; P23.B
   validates all 18 mutants and their disagreement direction; P23.D measures
   blindness against 19 named legacy rows; P23.E drives one real
   `runtime-launch` process per mutant; P23.F verifies the repaired wiring and
   the 417-frame no-behaviour-change claim.

2. Reinserted the exact defeated revision-8 shape in a separate scratch copy:
   `reject_all_probe`, declared as an admitting narrowing and blind to all three
   baselines. The focused uncached run exited `1` as required, with three
   independently named failures:

   - P23.B: `REJECT-ALL MUTANT` because the plain valid frame was refused;
   - P23.D: `OVER-CLAIM` because five revision-6 rows catch it;
   - P23.E: `REJECT-ALL MUTANT` at the production entry because the valid frame
     did not reach `execve`.

   This defeats the attractive old result directly: a large kill count from a
   reject-all fake can no longer satisfy the harness, and the false blindness
   label is executable rather than prose.

3. The corrected `unknown_by_wire_form` mutant is load-bearing on the accepted
   side: the plain valid frame reaches `execve`, the escaped-key valid class is
   over-refused with six witnesses, and the same decoder/production entry is
   named by the wiring obligation. The review found no bypass through duplicate
   order, arity, unknown-name length, case folding, prefix matching, wire-form
   identity, trailing content, or missing evidence.

## Specification and board fit

- States, transitions, persistent records, AF_UNIX transport, broker/runtime
  process boundaries, single-flight election, RUN-bound state, connection-bound
  leases, crash recovery, final-release shutdown, status/stop, and typed
  refusals are normative and internally consistent.
- Arbitrary-listener refusal remains fail-closed through executable/build,
  protocol, runtime-key/profile, endpoint, runtime-process, liveness, and model
  gates. Same-uid/socket-ownership limits are stated rather than overclaimed.
- Section 13 requires two separate `task-board spawn --background` RUN handles,
  distinct client/Pi ancestry, overlap across startup, one attested Qwen pid,
  two simultaneous leases, and bounded cleanup after both terminal states.
- Board decomposition remains the smallest useful chain:
  `imfmgz -> lsojra -> 1lc8o7`. Revision 9 adds only the six concrete harness
  obligations to the implementation leaf and correctly adds nothing to the
  unchanged acceptance leaf. Both downstream precondition resources carry the
  exact revision-9 bytes.
- The general harness-gate finding is persisted in `LOGBOOK.md` entry 2245.

## Validation note

The producer's attached repository no-regression control reports `go build`,
`go vet`, and all three `tools/agents-infra` test packages green. I accepted it
as a baseline because revision 9 changes specification/logbook text, not shipped
Go code, and independently reran the revision-specific probe above.

`task-board validate` currently reports one concurrent board-wide
`PARENT_STATUS_MISMATCH` on unrelated `STORY-260825-7oqacp`; it names no element
in this Story and does not contradict this task's dependency, resource, or
checklist structure.

Accepted handoff: record with `accept_cr(TASK-260825-imfmgz, revision=9,
evidence=TASK-260825-imfmgz_review-verdict.md)`. The reviewer supplies no
`commit_ack`; the Orchestrator owns checkpoint/integration and the final `done`
transition.
