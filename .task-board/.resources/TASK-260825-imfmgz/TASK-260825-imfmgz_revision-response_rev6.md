# TASK-260825-imfmgz revision 6 — response to review `RUN-260825-a8a4ef`

Verdict answered: **changes requested**, one blocking finding with five rework
items. All five are closed. Nothing else in the specification was re-opened.

Deliverable: `.research/260825_shared-local-runtime-broker.spec.md` at revision 6,
committed as `4a12a67` on `task-board/story/STORY-260825-1r7z9o`.

## F1 — the claimed closed field set was not closed

The review is correct and the finding is a clean instance of the
**bypass-around-the-check** shape. Revision 5 declared the authorization frame's
field set closed at five members and then enforced that closure with five
equality comparisons over a Go struct. Unknown object members are discarded and
duplicate keys resolve to one value, so a frame carrying a sixth member, or
carrying `protocol_version` twice, satisfied every comparison while transporting
data the launcher never looked at. All five rows were green, and all five were
*true*.

The rule was not wrong. It was on the wrong side of the decode. A comparison
answers *is this value the one I expect*; it cannot answer *were these the only
bytes the frame carried*, because the decode that precedes it has already thrown
that evidence away. Revision 6 therefore does not fix five comparisons — it moves
the closure to the only place that can hold it.

### Item 1 — specify the decoder boundary

`§6.2 B12` gains **step 4**, before any value comparison and long before
`execve`: the bytes read at step 3 must be *exactly one JSON object, carrying
each of the five allowed keys exactly once, with nothing after it*. Every
violation exits `protocol_violation` — the code the review recommended — carrying
a `reason` and, where one member is at fault, that member's name:

| Violation | `reason` |
| --- | --- |
| Not a JSON object, or content after it closes | `frame_not_single_object` |
| A member outside the closed five | `frame_unknown_field` |
| An allowed member more than once | `frame_duplicate_field` |
| An allowed member absent | `frame_missing_field` |
| Unreadable as one object | `frame_unparseable` |

The verdict order is normative (unknown → duplicate → missing) so a frame that
violates two rules has one specified refusal a test can name. The comparisons
become **step 5** and `execve` **step 6**; the header records the renumbering the
way `§6.0` records revision 4's.

The specification also states *how* to decode, because the wrong decode is the
defect: enumerate the object's member names with duplicates retained and apply
the table to that multiset. Decoding into a five-member structure and inspecting
its fields cannot satisfy the rule — a structure with five members can never
report a sixth.

`§11` rewrites the `protocol_violation` row with its five reasons and separates
it from `runtime_authorization_mismatch`, which is now reached only for frames
whose *shape* step 4 already accepted. `§8` gains the out-of-contract-member row.
`§9` restates the rule as "no field uncompared **and** no field uncounted", with
the general form underneath it: **a check can only be as strong as the evidence
still present when it runs.**

### Item 2 — both duplicate orders, and the unknown-member negative

New test **12.2.30c**, driven with **raw bytes** rather than a serialized
structure — the whole subject is frames a five-member structure cannot express,
so a test that builds its input from one proves nothing. Five refusal rows
(unknown member, duplicate wrong-then-valid, duplicate valid-then-wrong, absent
member, appended object), each asserting code, `reason`, named member, and paired
with a `kern.procargs2` poll proving the pid **never** carried
`runtime.executable`. The discriminating control is the all-valid frame, which
must `execve`.

**Both duplicate orders turned out to be load-bearing for a reason the review
named and the run then sharpened.** The producer's first draft of the mutant
table expected one deleted-clause mutant to admit both orders. The run
contradicted it, and the contradiction is the point: Go's last-wins decode admits
`{"protocol_version": 999, …valid…}` and *refuses* the reverse, because last-wins
hands the equality gate the valid value in one order and `999` in the other. A
first-wins decoder admits exactly the reverse. Each order is refused by whichever
dedup rule the implementation did not choose — so a suite testing a single order
shows a green "duplicates are refused" against a launcher with no duplicate rule
at all, on either decoder. The table now carries **two** duplicate mutants, one
per dedup rule, and each order is proved load-bearing against the rule that would
otherwise hide it.

12.2.30a is annotated with what it cannot establish: it varies values in a
well-shaped frame, so a launcher passing all six of its rows may still be
discarding members it never saw. That split is why the review's attack passed
every row of it.

### Item 3 — discriminating decoder mutants

Six, one per clause plus revision 5's launcher verbatim, each recorded in `§12.3`
with what it must redden **and what it must leave green**:

| Mutant | Reddens | Stays green |
| --- | --- | --- |
| Delete the unknown-member clause | unknown row → `execve` | duplicate, single-object rows |
| Delete the duplicate clause, last-wins decode | wrong-then-valid → `execve` | unknown row; valid-then-wrong (equality catches it) |
| Delete the duplicate clause, first-wins decode | valid-then-wrong → `execve` | wrong-then-valid (equality catches it) |
| Delete the single-object clause | appended-object row → `execve` | unknown row |
| Delete the whole gate — revision 5's decoder | unknown and wrong-then-valid → `execve` | — |
| Record the multiset deduplicated | the control's multiset assertion | — |

Two results are recorded as **non-discriminating**, asserted rather than omitted:

- the absent-member clause admits **nothing** — the equality gate refuses a zero
  value on its own — so its mutant changes a refusal's `reason` and never reaches
  `execve`;
- `shape_gate_deleted` does **not** redden the missing-member row, for the same
  reason.

A mutant table that keeps only its discriminating rows reads as broader coverage
than it has, which is the failure mode this review found one level up.

### Item 4 — the wiring proof covers the decoded key multiset

`§12.4`'s comparison-set obligation was the same defect one level up, and the
specification now says so. Revision 5 required the instrumented run to fail if a
transmitted field went uncompared, but specified the compared set as the
launcher's five struct members — making the obligation satisfiable by an
implementation *structurally incapable* of detecting the violation it was asked
to report. It is now stated over the **decoded key multiset**, duplicates
retained, and must fail unless the multiset and the compared set are both exactly
the five allowed keys once each. Duplicates are retained in the record for the
same reason they are refused: a deduplicating record reports precisely the shape
the forged frame presents, which is why that is itself a `§12.3` mutant. The
evidence is written **before** the verdict is acted on, so a refusing run leaves
the same record a passing one does.

### Item 5 — logbook

`LOGBOOK.md`, entry *1905 — A Comparison Cannot Close A Field Set*, recording the
regression, the root cause, the measured duplicate-order finding that corrected
the first draft, the decision to record non-discriminating rows, and the version
bump.

## Protocol version

Bumped to `6`, on `§9`'s rule and not on a compatibility argument, for the second
revision running. The frame's bytes did not change again; the launcher's gate set
changed again. Skipping the bump because nothing moved on the wire is exactly the
exception that would make the version unreliable, and applying one rule twice to
two changes that differ is the only test a rule of this kind gets. What it
protects is a client attaching to a stale broker while assuming a gate set that
broker never applied.

## Evidence

- `TASK-260825-imfmgz_probe-rev6-results.log` — P21 and P22 on darwin 25.5.0
  arm64, go1.25.5. 28 passes, 0 failures.
- `TASK-260825-imfmgz_probe-rev6-p22_frame_shape_closure_test.go` — the P22 rows
  and the mutant table.
- `TASK-260825-imfmgz_probe-rev6-frame_shape_rev6_test.go` — the decoder boundary
  and its six mutants, including both dedup rules.
- `TASK-260825-imfmgz_probe-rev6-p21_frame_field_closure_test.go` — the corrected
  launcher, with `protocol_version` at 6.
- `TASK-260825-imfmgz_probe-rev6-earlier-rerun.log` — uncached rerun of the rev2,
  rev3, rev4 and rev5 probe modules; all pass.
- `TASK-260825-imfmgz_rev6-repo-baseline.log` — `go build`, `go vet`, and
  `go test -count=1 ./internal/... .` all pass in `tools/agents-infra`.

The reviewer's own attack module (`probe-rev5`) is retained unchanged and still
passes: it reproduces the defeat against revision 5's launcher, which revision 6
keeps in-module as the `shape_gate_deleted` mutant. The defect and its fix are
therefore measured against each other rather than across two modules.

## What a re-reviewer should attack first

Ranked by where this producer's confidence is thinnest:

1. **The single-object clause claims the least and should be checked for
   overclaim.** Same-uid forgery is out of the threat model (`§9`), so this clause
   closes no forgery. It is specified as making the launcher able to *state* what
   it received, and as failing closed on a partial-then-complete write. If that
   framing still reads as a security claim, it is wrong and should be narrowed
   further.
2. **The verdict order (unknown → duplicate → missing) is a choice, not a
   derivation.** A frame violating two rules has one specified refusal. Attack
   whether any pair of violations makes the specified order the wrong one to
   report.
3. **`frame_unparseable` is shared between the tokenizer and the value decode.**
   Both paths exit with it and the test does not hold them apart the way `§10.3`
   holds the four `runtime_launch_unauthorized` reasons apart. That is the same
   shape as the revision-4 finding, one level smaller.
4. **The multiset record is written by the launcher about itself.** It is
   evidence for `§12.4`, and a launcher that lies about what it decoded would
   satisfy it. The production obligation is instrumentation of the real entry
   point, which `TASK-260825-lsojra` owes; the probe cannot discharge it.
5. **P22 asserts refusals within a bounded poll.** Every refusing row waits out
   the 3s exec-path poll before concluding "never carried the target". A refusal
   that took longer than the poll would be scored as a refusal-with-no-exec
   either way.
