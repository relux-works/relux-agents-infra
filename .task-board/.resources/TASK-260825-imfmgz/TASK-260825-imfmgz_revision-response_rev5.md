# Revision 5 response to review `RUN-260825-668303`

Task: `TASK-260825-imfmgz`  ·  Producer run: `RUN-260825-e7aee1`
Deliverable: `.research/260825_shared-local-runtime-broker.spec.md`, 2431 → 2741 lines
Probes: `.temp/TASK-260825-imfmgz/probe-rev5/` (module `probe5`)

The review found one blocking issue and it was real. Revision 5 is deliberately
narrow: no ownership, recovery, lease or acceptance semantics are reopened, no
broker step is renumbered, and no board element is created or re-scoped.

## F1 — the launcher protocol-version gate existed only in the specification

**What the review demonstrated.** B11 transmitted `schema` and
`protocol_version`; B12 step 4 required version equality before `execve`; the
refusal table promised `runtime_authorization_mismatch`. The attached P18
launcher parsed `Protocol` and compared only pid, key and digest. A frame with
the correct pid, runtime key and plan digest and `protocol_version = 999`
reached `execve` on the same pid. No P18 case varied that dimension and the
mutant table held no version mutant, so a green P18 could not support the claim.

**What changed, and why it is not a one-line fix.** The defect was not a wrong
comparison — it was a field that was transmitted, parsed, and then not used.
Fixing only the version comparison leaves the next transmitted-and-ignored field
for the next reviewer, so revision 5 closes the class:

- **§6.2 B11** states the frame's **exact five fields** and both compiled
  constants, and declares the field set **closed**: an implementation may not
  carry a field the launcher ignores and may not read one it does not compare.
  Extending the frame is a protocol bump.
- **§6.2 B12 step 4** becomes an ordered five-row table — `schema`,
  `protocol_version`, `launcher_pid`, `runtime_key`, `exec_plan_digest` — each
  compared by equality, each refusing `runtime_authorization_mismatch` with
  `mismatch_field` naming which diverged. Two rules are stated explicitly because
  both are ways to look like a check without being one: **comparisons are
  equalities, never compatibility ranges**, and **an absent field is a divergence,
  not a default**.
- **§9** adds why the launcher's version skew is **reachable rather than
  hypothetical**: B9 spawns the broker's own executable *by path*, so an in-place
  `rename`-over upgrade landing between a broker's start and its B9 produces a
  new-build launcher authorized by an old-build broker. That is the launcher-side
  twin of the stale-broker case gate 3b exists for, and the launcher has no
  handshake with which to run gate 3b. Neither `schema` nor `protocol_version` is
  redundant with `runtime_key`/`exec_plan_digest`: a release that changes the
  record layout, the frame fields, the argv scrubbing rules or the gate set —
  each of which §9 already requires a bump for — can leave both digests identical
  while changing what the frame means.
- **§8, §10.3, §11** carry the consequences: two new failure-matrix rows, the
  refusal payloads, and the statement that `runtime-launch` still deliberately
  admits a hand-invoked launcher with a genuine project and a self-minted valid
  frame — bounded because recomposition, not the frame, decides *what* runs.

### Review item 4 — is `schema` also a gate?

**Bound, not dropped.** It is gate 4a with the same refusal, the same
`mismatch_field` discipline and its own mutant. The same upgrade skew that makes
the version gate reachable makes a frame *shape* change reachable, and a
transmitted-but-unchecked field is exactly the defect F1 named.

The two fields answer two questions, and revision 5 is its own illustration: it
bumps `protocol_version` `4 → 5` because the launcher's gate set changed, and
leaves `schema` at `…auth.v1` because the five fields and their encoding did not.
`schema` says *how to read these bytes*; `protocol_version` says *what the broker
that wrote them does*.

### Review items 1-3 — the evidence

New tests **12.2.30a** and **12.2.30b**; corrected **12.2.18b**; eight new
narrowing mutants in §12.3; a new obligation in §12.4 that the instrumented
launcher run record **which fields were compared** and fail if a transmitted
field was read without comparison — because coverage is not inferable from a
green suite, which is the whole of this finding.

Probes P21.A-F, all passing on darwin 25.5.0/arm64, go1.25.5:

| Probe | Shape | Result |
| --- | --- | --- |
| P21.A | wholly valid frame | `execve`, kernel shows composed exec path and exact argv on the same pid — the control every refusal is measured against |
| P21.B | `protocol_version = 999` | refuses naming `protocol_version`, pid never carries the target. **Delete** mutant (the revision-4 launcher verbatim) reaches `execve`; **narrow to `>= 1`** mutant *also* reaches `execve` while still refusing `0` |
| P21.C | foreign `schema` | refuses naming `schema`; `schema_unchecked` mutant reaches `execve` |
| P21.D | five rows, one field bent each | each refuses naming that field and never carries the target; the all-correct sixth row `execve`s; `unnamed_field` mutant keeps every exit code green while emptying every field name |
| P21.E | absent fd3, non-FIFO fd3, EOF, deadline, truncated frame | four unauthorized shapes held apart by `reason`, one `protocol_violation`; `collapse_fd3` mutant reports an absent descriptor as a dead broker under the same exit code |
| P21.F | silent broker | armed bound fires at 705ms against a 700ms timeout; `deadline_ignored` mutant still blocked after 4s; `deadline_unavailable` reached by labelled fault injection |

**The reviewer's attack is rerun verbatim in shape** against the corrected
launcher (`review_rev4_attack_rerun_test.go`): the same entry point, the same
caller-minted frame, the same single varied field. It now refuses
`runtime_authorization_mismatch(protocol_version)` and the pid never carries
`/bin/sleep`, paired with the valid-version control that still reaches `execve`.

### Review item 5 — P18.D/E wording

Corrected by **supersession rather than restatement**, recorded in §2's probe
table and in §12.4's closing note. P18.D asserted an exit code four gate-4
branches share; P18.E's absent-FIFO case asserted the code the EOF branch also
produces. Both claims named more than they exercised. 12.2.30a and 12.2.18b,
asserting `mismatch_field` and `reason`, carry them now.

## F2 — self-found: the read bound was stated, never armed

B12 step 3 said the launcher reads its frame "with a deadline". A descriptor
inherited across `fork`+`exec` arrives **blocking**, and a read deadline cannot
be armed on a blocking descriptor. Revision 4's launcher discarded the error from
arming it, so its read had **no bound at all** and its own deadline branch was
*unreachable rather than merely untested* — which is why no revision-4 case ever
exercised it. An implementation following revision 4 literally waits forever for
a broker that never writes.

B12 step 3 now requires the bound to be armed before the read and makes a failure
to arm it fatal (`deadline_unavailable`): a launcher that cannot bound its wait
must refuse, never wait. Test 12.2.30b; probe P21.F, whose deletion mutant is
still blocked after 4s on a 700ms bound. The arming-failure branch is reached by
**fault injection and labelled as such** in both the spec and the probe, because
gate 1 has already established a FIFO and a FIFO is pollable, so no input reaches
it.

## F3 — self-found: the probe harness could satisfy its own negative assertions

Revision 4's P18 recorded exec-path observations into a 64-slot channel with a
non-blocking send. Once sixty-four observations accumulated, later ones were
dropped — including, potentially, the one proving the launcher had `execve`d the
target. Every "never carried the exec path" assertion in P18 was satisfiable by a
full buffer. It did not bite in revision 4, where every refusing launcher exited
in milliseconds; it would have bitten P21.E and P21.F, whose refusals are
deliberately slow. Revision 5's harness records into an unbounded mutex-guarded
set. Recorded in §12.4.

## The protocol bump, and why the easy answer was rejected

Nothing in the frame changed, so a revision-4 broker and a revision-5 launcher
would exchange byte-identical bytes and a compatibility argument is available.
It was rejected. §9 says a change to **the gate set** bumps the version, and the
launcher's gate set changed. The version gate is enforced by the *reading* side,
so a stale build can never be caught by the check it does not have. What the bump
actually protects is the **client**: a revision-4 broker left running is a broker
whose runtime was authorized under a launcher gate set a revision-5 client would
otherwise assume had been applied. Gate 5 refuses it with
`broker_protocol_version_mismatch`, remedied by one `agents-infra runtime stop`.

## Evidence rerun by this producer, not inherited

- `probe-rev5`: `go vet` clean, `go test -count=1 -v ./...` all pass (6.1s).
- `probe-rev4` (P16-P20), `probe-rev3` (P8-P15), `probe-rev2` (P1-P7): all pass,
  uncached, `-count=1`.
- `tools/agents-infra`: `go build ./...` OK, `go vet ./...` OK,
  `go test -count=1 ./internal/...` (attachments 1.392s, infra 115.727s) and
  `go test -count=1 .` (72.879s) all pass. The revision-5 delta is the
  specification and the logbook, so this suite is a **regression baseline, not
  evidence for the delta**.

## Board

No element created, removed or re-scoped. All three spec payloads — the task
outcome resource and both siblings' precondition copies — refreshed to revision 5
and verified byte-identical to the committed file. Five checklist items added to
`TASK-260825-lsojra`, each citing a named spec section and the finding it closes.
`TASK-260825-1lc8o7` needs no revision-5 item: §13 is unchanged by this revision,
and inventing one would be board noise rather than traceability.

## Attack first, in revision 5

Ranked, for whoever reviews this next:

1. **The closed-field-set rule is a rule about code that does not exist yet.**
   §12.4 asks the instrumented run to fail if a transmitted field was read
   without being compared. That is an assertion about the *set* of comparisons
   performed, which is harder to instrument honestly than any outcome assertion
   in this document. If `TASK-260825-lsojra` cannot implement it as stated, the
   rule degrades to a code-review convention and F1's class reopens.
2. **`schema` equality is argued from an upgrade that also moves the version.**
   Every concrete skew scenario I can construct moves `protocol_version` too, so
   4a may be strictly redundant with 4b in practice. I bound it anyway, on the
   grounds that a transmitted-and-unchecked field is the defect itself — but if a
   reviewer can show 4a can never fire while 4b passes, the honest answer might be
   to remove the field rather than gate it.
3. **The version bump is argued to protect the client, not the launcher.** The
   chain is: stale broker → its runtime was authorized under a gate set the new
   client assumes → gate 5 refuses the broker. If a revision-4 broker can be
   attached to by a revision-5 client through any path that skips gate 5, the
   bump buys nothing and the reasoning in §16.1 is wrong.
4. **P21 models the launcher at the syscall layer**, like every probe before it.
   §12.4 is what makes it production evidence and `TASK-260825-lsojra` owes it —
   the same gap that produced the revision-3 F1 and this revision's F1.
5. **The four `runtime_launch_unauthorized` reasons are now normative surface.**
   They are refusal *payload*, not protocol messages, so nothing gates on them —
   but tests now assert them, which makes them a contract an implementation can
   drift from silently.
6. **`deadline_unavailable` is unreachable by input** and is proved by injection.
   That is stated plainly, but an injection-proved branch is weaker evidence than
   every other row in this document, and a reviewer should decide whether a branch
   no input reaches should exist at all.
