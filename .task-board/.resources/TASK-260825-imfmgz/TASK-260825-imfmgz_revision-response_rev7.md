# TASK-260825-imfmgz — revision 7 response to review `RUN-260825-9d5cff`

Producer run: `RUN-260825-34ba50`
Deliverable: `.research/260825_shared-local-runtime-broker.spec.md`, revision 7
Platform for all evidence: darwin 25.5.0 arm64, go1.25.5

## What the review found, in one sentence

The revision-6 decoder was correct and its **proof** did not bind anything to
it: two rules stated as classes were tested with one sample each, and two
narrowed decoders satisfy every revision-6 row while admitting frames the
specification refuses.

## Item-by-item closure

### Item 1 — a same-value duplicate row

**Closed.** §12.2.30c gains **P22.G**, and it is exhaustive rather than a single
row: the allowed set is closed and finite, so **all five** members are duplicated
carrying their **own valid value**, one row each. Every row asserts
`protocol_violation`, `frame_duplicate_field`, the named member, and a
`kern.procargs2` poll until the pid is gone showing it never carried
`runtime.executable`.

Why the valid value rather than any value: a duplicate carrying a *wrong* value
is refused by step 5's equality comparison whether or not a duplicate clause
exists. That is precisely how the value-sensitive narrowing survived revision 6.

Evidence: probe `P22.G_duplicate_same_value_{schema,protocol_version,runtime_key,launcher_pid,exec_plan_digest}`.

### Item 2 — a narrowed mutant that rejects duplicates only when values differ

**Closed, and a second narrowing was added on the same clause.**

- `dup_only_if_values_differ` reddens all five P22.G rows and is green on **the
  entire revision-6 table**, including both differing-value order rows. That is
  the reviewer's finding reproduced at the production entry point.
- `dup_only_protocol_version` is the same defect one axis over — the revision-6
  rows all duplicated `protocol_version`, so a gate applied to that member alone
  also survives them. It reddens the four non-`protocol_version` P22.G rows. This
  is why the member dimension is covered exhaustively rather than by a
  representative.

### Item 3 — prove the unknown-name class, not one sample

**Closed by near-miss class.** The name space is infinite, so §12.2.30c's
**P22.H** covers it by classes, and every class after the anchor exists because
exactly one narrowed mutant reddens on it:

| Unknown member | Narrowed mutant it reddens |
| --- | --- |
| `caller_chosen_field` | none — the `RUN-260825-a8a4ef` regression anchor, labelled as such rather than presented as class evidence |
| `future_extension` | `unknown_only_caller_chosen_field` |
| `Schema` carrying `schema`'s **valid** value | `unknown_case_folded` |
| `exec_plan_digest_v2` | `unknown_prefix_allowed` |
| `Schema` carrying a **wrong** value | none — recorded explicitly as non-discriminating |

**A measured correction, and the reason the last two rows read the way they do.**
The case-variant row did not work as first written, and the first run reddened on
exactly that. Go's `encoding/json` matches an object key to a struct field by
exact tag first and by **case-folded name** second, so `Schema` is not discarded
like `future_extension` — it is absorbed into `schema` and overwrites it. With an
arbitrary value the equality comparison stops it and the clause under test is
never exercised; only the member's own valid value reaches step 5 with everything
equal. Both variants are kept, the wrong-value one asserted **as a mismatch** and
labelled non-discriminating, exactly as the missing-member row is.

This is revision 6's duplicate-order lesson one level down: **an attack that a
different gate happens to stop is not evidence for the gate under test.** B12
step 4 now states the underlying requirement — the shape gate runs on the member
names as they appeared in the bytes, never on what a case-folding structure
decode resolved them to — and §12.3 carries that as its own mutant.

### Item 4 — update the sections, rerun uncached, record the matrix

**Closed.** §6.2 B12 step 4 pins both rules as classes and states the
case-folding fact. §12.2.30c gains P22.G, P22.H and P22.I and splits its mutant
table into **deletions** (prove the clause is present) and **narrowings** (prove
the class it covers). §12.3 gains six entries. §12.4 requires the recorded names
to be the ones in the bytes. §9 gains the general form. §16.1 is the closure map
for this revision; the earlier subsections renumbered to 16.2–16.6 and every
cross-reference to them was updated. `TASK-260825-lsojra` receives checklist
items 40–44.

Reruns, all uncached, all by this producer:

| Suite | Result |
| --- | ---: |
| probe module rev7 (P21, P22.A–I, rev4 attack rerun) | 109 pass / 0 fail |
| probe modules rev2, rev3, rev4, rev5, rev6 | all `ok` |
| `tools/agents-infra` `go build ./...` / `go vet ./...` | OK / OK |
| `go test ./internal/...` | attachments 1.508s, infra 107.029s |
| `go test .` | 71.575s |

The repository suite is a **no-regression control**, not a test of the delta: the
delta is the specification and `LOGBOOK.md`.

### Item 5 — logbook

**Closed.** `LOGBOOK.md`, entry `2005 — A Rule Stated As A Class Cannot Be Proved
By A Sample`.

## No protocol bump, deliberately

The verdict did not request one and §9's rule does not produce one: the
launcher's gate set is byte-identical to revision 6, so there is no client a bump
would protect. Bumping here would be as wrong as skipping it was in revisions 5
and 6, and §16.1 records that reading explicitly so the next revision does not
have to re-derive it.

## Attack first — ranked, for the next review

1. **The unknown-name class is still covered by classes, not exhaustively, and
   it cannot be otherwise.** Three near-misses is a judgement call. A reviewer
   who can name a fourth narrowing that survives all four rows — a length check,
   a trailing-whitespace tolerance, a Unicode normalisation — has the same
   finding again, one class further out. The honest position is that this
   dimension is unbounded and the mutant list is the claim, not the coverage.
2. **`caller_chosen_field` has no unique mutant.** It is retained for regression
   continuity and is labelled as carrying no class evidence. If a reviewer holds
   that a row without a discriminating mutant should not be in a table at all,
   that is a defensible position and the row should go.
3. **The case-folding fact is Go-specific and is now normative.** B12 step 4 is
   written to bind any implementation, but the *reason* it is written was
   measured on one standard library. An implementation in another language may
   have a decoder whose absorption rule differs, and the row that discriminates
   there may not be a case variant at all.
4. **P22 still models the launcher at the syscall layer.** §12.4 is what makes it
   production evidence and `TASK-260825-lsojra` owes it. This is the same gap
   that produced the revision-3 and revision-5 F1s, and it does not close until
   the implementation exists.
5. **The duplicate clause is proved exhaustively over the *members* but only
   over one *arity*.** Every row duplicates a member twice. A gate that refuses
   the second occurrence and admits the third is not excluded by anything here,
   though the specified rule plainly forbids it.
6. **`dup_only_protocol_version` and `unknown_only_caller_chosen_field` are the
   same mutant shape** — sample-one-value-of-a-dimension — applied to two
   clauses. If a reviewer shows a third dimension in step 4 that admits the same
   shape and is untested, that is the next finding.
