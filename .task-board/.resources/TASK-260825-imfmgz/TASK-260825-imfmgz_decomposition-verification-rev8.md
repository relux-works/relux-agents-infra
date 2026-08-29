# TASK-260825-imfmgz — revision 8 board delta and gap self-verification

Producer run: `RUN-260825-753d00`, closing review `RUN-260825-c71188`.

## Board delta

**No board element was created, removed, or re-scoped.** The story's shape is
unchanged:

```
TASK-260825-imfmgz  (design, this task)
        ↓ blocks
TASK-260825-lsojra  (implement)
        ↓ blocks
TASK-260825-1lc8o7  (acceptance proof)
```

Six checklist items added to `TASK-260825-lsojra` (45–50), each citing a named
spec section and stating what it supersedes or extends rather than silently
contradicting it — the pattern revisions 5, 6 and 7 established, and the reason
an implementer reading fifty items can still tell which rule is live:

| Item | Cites | Relationship to earlier items |
| --- | --- | --- |
| 45 | §6.2 B12 step 4 — the multiset equality | **Supersedes the rule statement** in 36 and 41. Behaviour is byte-identical since revision 6; what changes is that the decision is one equality and the three clauses are reporting |
| 46 | §6.2 B12 step 4, first normative rule | New. Forbids predicates over derived quantities, naming each shape a review or revision has actually seen admit a forbidden frame |
| 47 | §6.2 B12 step 4, second normative rule | New, and the one genuinely new *specified detail*: names are decoded, not wire bytes, in both directions |
| 48 | §12.2.30c, §12.3 | **Supersedes the proof obligation** in 40 and 42. Oracle differential, generated corpus, mutants-as-calibration, harness-reported coverage holes |
| 49 | §12.4 | **Extends** 39 and 43. The structural obligation and the shared call site |
| 50 | §9 | States that **38 and 44 stand** — no seventh version — so an implementer does not expect a bump after five revisions of version items |

`TASK-260825-1lc8o7` receives **no item**, for the same reason as revisions 6 and
7: §13 is untouched by this revision. The one candidate — "record that the
launcher's decoded multiset was the five allowed keys once each" — restates a
precondition of 1lc8o7's own success, because a B12 step-4 refusal means no
runtime exists and the acceptance proof cannot be produced at all. An item that
cannot fail independently of the item it sits under is not an item.

## Additions considered and rejected

Both were checked against the spec's own answers and its out-of-scope list before
being rejected, per decomposition rule 3.

| Candidate | Verdict | Sections checked |
| --- | --- | --- |
| A research task on whether the shape gate should be a schema validator (JSON Schema, CDDL) rather than hand-written | **Rejected — the spec already answers it.** §6.2 B12 step 4 now specifies the decision as a multiset equality against a compiled constant; a schema validator is one possible implementation of that and needs no research to permit. §14 already records the rejected alternative of putting the plan in the frame, on the same "the launcher recomputes rather than trusts" principle. Nothing is unresolved | §6.2 B12, §9, §14, §15 |
| A task to make `mismatch_field` distinguish "named the empty member" from "named nothing", closing the degenerate zero-length assertion | **Rejected — invented scope, and the spec's threat model says why.** §9 puts same-uid forgery out of the threat model: the refusal payload is a diagnostic for an operator, not an authenticated channel. The zero-length row is already carried by its `reason`, by the never-carried-the-target poll and by its mutant, so the gate is proved; only one *assertion* is degenerate, and it is labelled as such in both the spec and the probe. Changing the wire shape of a refusal to strengthen a test assertion inverts the dependency | §9, §11, §12.2.30c, §15 |

## Justified-gap record

**None required.** Revision 8 adds no element and no scope beyond the review's
four rework items and its F2. The one thing it specifies that the literal review
did not ask for — the decoded-name rule in §6.2 B12 step 4 — is not a gap-closing
*element*; it is a normative answer to an ambiguity discovered while building the
evidence, inside a section this revision was already rewriting, and it changes no
frame's verdict under the reading the shipped decoder already implements. It is
recorded in §16.1 as self-found rather than presented as a review item.

## Traceability

| Spec section changed | Requirement it implements | Review item |
| --- | --- | --- |
| §6.2 B12 step 4 | AC "typed refusals"; DoD "arbitrary-listener refusal through … attestation" | F1 (a) |
| §8 out-of-contract-member row | AC "failure recovery" | F1, consistency |
| §11 `protocol_violation` row | AC "typed refusals" | F1, consistency |
| §12.2.30c | AC "tests"; DoD "gate, refusal, validation, authorization and attestation behaviour attacked, not read" | F1 (b), items 1–3 |
| §12.3 | same | F1 (b) |
| §12.4 | AC "tests"; DoD "solution fits project architecture" | F1 (b), the structural half |
| §2 probe table | AC "security assumptions" — probed, not assumed | F1, F2 |
| §16.1 | the closure map this document indexes | F1, F2 |

## Verification performed before writing any of the above

- read the review verdict and both attack logs in full;
- read the reviewer's reconstructed probe module, and reused it rather than
  rebuilding one, so the revision-8 evidence runs the code the review ran;
- reproduced both bypasses before changing anything — they are P23.C rows now;
- confirmed the **revision-7 decoder needed no change** by running the oracle
  differential against it: 398 frames, agreement on verdict, reason and member.
  That is why no behaviour changed and no version bumped, and it is a measured
  fact rather than an assumption;
- checked §9, §14 and §15 before both rejections above;
- `task-board validate` green.
