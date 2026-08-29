# TASK-260825-imfmgz — revision 9 board delta and gap self-verification

Producer run: `RUN-260825-fc8f60`, closing review `RUN-260825-86b7d5` F1.

## Board delta

**No element was created, removed, or re-scoped.** The board shape is unchanged
since revision 1:

```
TASK-260825-imfmgz  (design, this task)
  └─ blocks TASK-260825-lsojra  (implementation)
       └─ blocks TASK-260825-1lc8o7  (acceptance proof)
```

`task-board validate` passes.

### Resources refreshed

The specification payload is a **separate copy** on each element, not a link, so
updating only the producer copy would leave revision 8 in front of the
implementer. All three copies plus the working tree are byte-identical at sha256
`964148a9a2b77c89b8d9e2abcfd91c38c4dcb9613598d43daf71fa6453202dbb`:

| Element | Resource | Type |
| --- | --- | --- |
| `TASK-260825-imfmgz` | `TASK-260825-imfmgz_shared-runtime-broker.spec.md` | outcome |
| `TASK-260825-lsojra` | `TASK-260825-lsojra_shared-runtime-broker.spec.md` | precondition |
| `TASK-260825-1lc8o7` | `TASK-260825-1lc8o7_shared-runtime-broker.spec.md` | precondition |

### Checklist items added

Six items on `TASK-260825-lsojra` (51–56), each citing a named spec section:

| # | Cites | What it obliges |
| --- | --- | --- |
| 51 | §12.3.1 rule 1 | every shipped shape-gate mutant must still admit the plain valid frame, at the decoder **and** at the production entry, one process per mutant. **Extends** the proof obligation in items 48 and 42; does not supersede them |
| 52 | §12.3.1 rule 2 | every mutant declares its disagreement side and must disagree only there |
| 53 | §12.3.1 rule 3 | blindness measured against a named row inventory, failing on over-claim and under-claim alike, with a corpus-only witness proved absent by bytes |
| 54 | §12.3.1 final paragraph | ship `reject_all_probe` as a non-gate probe and assert all three rules redden on it |
| 55 | §9, §16.1 | items 38, 44 and 50 **stand**: no protocol bump, and revision 9 measures the claim over 417 frames |
| 56 | §12.3, §16.1 | the three measured corrections an implementer must not re-derive from the older prose |

Items 51 and 53 are phrased as **extending** items 42 and 48 rather than
superseding them, and that distinction is deliberate: revision 9 did not change
what the decoder must do or what the corpus must cover. It added obligations on
the *mutants that prove it*. An implementer reading items 42, 48, 51 and 53
together gets four non-contradicting obligations.

`TASK-260825-1lc8o7` receives **no** item. §13, the acceptance proof, is
untouched by revision 9. The one candidate — "record that the shipped mutant
table passed its own validity rules" — restates a precondition of `lsojra`'s
success rather than anything the two-spawn acceptance run observes, and §13's
proof is about broker ownership and lease lifecycle, not about the launcher's
frame decoder.

## Justified-gap self-verification

Revision 9 creates no board element beyond the literal spec, so the
`Justified gap` rule has nothing to license. It is still applied to the two
additions considered and rejected, because the rule's value is in the rejections.

### Rejected — a separate task for "harness validity rules"

**Proposed:** a new leaf under the story owning §12.3.1's three mutant
obligations and the `reject_all_probe` negative, on the argument that they are
test-infrastructure work rather than broker implementation.

**Checked against:** §12.3 (the mutant table has been `lsojra`'s obligation since
revision 3), §12.4 (the wiring proof, likewise), §15 scope boundaries, and the
review's own rework item 5, which says *"propagate only the resulting concrete
implementation/evidence obligation to `TASK-260825-lsojra`; no new board leaf is
justified by this finding."*

**Result: rejected.** There is no system gap. The mutants already belong to
`lsojra`'s deliverable; §12.3.1 constrains how they must be built, which is a
property of work already owned. Splitting it would create an element that cannot
be completed independently of the element it was split from, and the review
explicitly declined it.

### Rejected — a research task on generalising the harness rules

**Proposed:** research whether §12.3.1's three rules should be lifted into a
reusable mutation-testing contract for the repository, since the finding
("a mutation harness's controls are gates") is not specific to this frame
decoder.

**Checked against:** §15 scope boundaries, which bound this task to the shared
runtime broker; §14 rejected alternatives; and the story's own scope, which is a
runtime owner and lease protocol.

**Result: rejected.** The question is genuinely open, but it is open *outside*
this story. Nothing in this specification is blocked on the answer, and a
research task whose result cannot change any decision inside `lsojra` or
`1lc8o7` is ceremony. The general lesson is recorded where general lessons for
this repository belong — `LOGBOOK.md` entry 2245 — and that is the whole of the
propagation it earns here.

## Traceability

Every checklist item added by this revision cites a spec section by number.
Every spec section revision 9 touched — §12.3 including the new §12.3.1,
§12.2.30c's P23 probe table and blindness prose, the document header, and §16.1
— is a section that already existed as `lsojra`'s obligation. Revision 9
introduces no requirement that is not traceable to review `RUN-260825-86b7d5`
finding F1 and its five rework items.
