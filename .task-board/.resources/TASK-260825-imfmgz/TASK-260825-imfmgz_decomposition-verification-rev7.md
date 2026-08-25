# TASK-260825-imfmgz — revision 7 board delta and self-verification

Producer run: `RUN-260825-34ba50`

## Board delta

**No board element was created, removed or re-scoped.** The story keeps its three
leaves and its dependency links:

- `TASK-260825-imfmgz` — this specification (analysis → to-review)
- `TASK-260825-lsojra` — implementation, blocked by `imfmgz`
- `TASK-260825-1lc8o7` — acceptance proof, blocked by `lsojra`

The only board writes are five checklist items on `TASK-260825-lsojra`, items
40–44, and the refreshed revision-7 spec attached as a precondition on `lsojra`
and `1lc8o7`.

## Why no element was added

Revision 7 changes **no specified behaviour**. The decoder boundary, the refusal
table, the closed field set and `protocol_version = 6` are exactly as revision 6
left them. What changed is §12's evidence contract and two paragraphs of §6.2
B12 step 4 that pin two existing rules as classes. There is no new mechanism to
build, so there is no new deliverable, so there is no new task. Creating one
would be board growth without scope, which rule 1 of the decomposition contract
forbids.

Every item added traces to a concrete spec requirement:

| Item | Spec requirement it implements |
| --- | --- |
| 40 | §12.2.30c, P22.G and P22.H row sets |
| 41 | §6.2 B12 step 4, the two class-pinning paragraphs |
| 42 | §12.3, the five new narrowing mutants |
| 43 | §12.4, the recorded-names-in-the-bytes clause |
| 44 | §9's bump rule read as a non-bump, plus §12.2.30c's P22.I rerun |

Items 40 and 43 name what they supersede or extend (37 and 39) rather than
silently contradicting them, and item 44 states that item 38 **stands** — the
protocol version does not move — because an implementer reading five revisions of
version items in sequence would otherwise reasonably expect a sixth bump.

## `TASK-260825-1lc8o7` receives no item

Checked against the delta. §13, the acceptance proof, is untouched by revision 7:
two independent RUN handles and Pi pids, overlapping execution, one attested Qwen
runtime pid, and cleanup after both terminal states are all as the sixth review
accepted them. The only candidate item — "record that the launcher's decoder
refused nothing during the accepted run" — restates a precondition of that run's
success, since a refusal at B12 means no runtime exists and §13's proof cannot
be produced at all. It would be a checklist item that cannot fail independently.

## Out-of-scope self-verification

Sections checked before concluding that revision 7 adds no scope: §15 scope
boundaries, §14 rejected alternatives, §9's threat boundary, §13's acceptance
proof, and §16.2–16.6's records of what earlier revisions deliberately did not
touch.

Two additions were considered and **rejected**:

1. **A research task on whether the frame should be length-prefixed or otherwise
   framed at the byte level rather than as JSON.** Rejected against §14, which
   records the framing decision as a deliberate position, and against §9, which
   places same-uid forgery outside the threat model. The reviewer's finding is
   about the proof of a decode rule, not about the encoding. Nothing in the spec
   leaves this open, so rule 4 forbids the research task.
2. **A separate task for the narrowing-mutant suite**, split out of `lsojra`.
   Rejected against rule 1: the mutants are evidence for the decoder `lsojra`
   already owes, and a mutant suite with no implementation to mutate has no
   deliverable. It stays as checklist items on the task that builds the thing.

## Traceability of the revision itself

Every change in revision 7 cites review `RUN-260825-9d5cff` finding F1 and the
specific rework item it closes; §16.1 is the item-by-item map, and
`TASK-260825-imfmgz_revision-response_rev7.md` carries the same map with the
evidence attached to each row.
