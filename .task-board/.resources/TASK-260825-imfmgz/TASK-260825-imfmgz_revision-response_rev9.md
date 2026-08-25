# TASK-260825-imfmgz — revision 9 response to review `RUN-260825-86b7d5`

Producer run: `RUN-260825-fc8f60`
Deliverable: `.research/260825_shared-local-runtime-broker.spec.md`, committed
`77ab287` on `task-board/story/STORY-260825-1r7z9o`; logbook entry 2245 in the
same commit. All four spec payloads verified byte-identical at sha256
`964148a9a2b77c89b8d9e2abcfd91c38c4dcb9613598d43daf71fa6453202dbb`.

## The short version

**The finding was real, and it was in my evidence about my evidence.** Revision
8 shipped eighteen mutants, each required to be killed by the generated corpus.
One of them was not the mutant its row described: `unknown_by_wire_form` was
wired through `isAllowedName(m.name, mutant)`, which rebuilds a `frameMember`
from the decoded name and therefore clears `wire`. The empty wire string matched
no allowed literal, so it refused every member of every frame. It was reported
KILLED by 48 witnesses, and all 48 were ordinary **valid** frames.

Revision 8's kill condition was `total > 0`. A reject-all fake satisfies that on
every accepted frame. The witness count was not merely non-zero, it was an order
of magnitude larger than any other narrowing's, and no rule in the harness was in
a position to read that as the tell it was.

**The general form, and the reason this revision is short:** a mutation harness's
controls are gates, and this task's own evidence rule applies to them exactly as
it applies to the launcher. "The mutant was killed" is a positive result *about
the harness*. Nothing in revision 8 would have failed if a mutant were the wrong
mutant — which is precisely the shape this project refuses in product code, one
level up.

No specified behaviour changed. `protocol_version` stays **6** for the fourth
revision running, and this time that is measured rather than asserted.

## Item-by-item closure

### F1 — the wire-form mutant discards the wire form

One line in `p21CheckKeys`: `isAllowedName(m.name, mutant)` became
`isAllowedMember(m, mutant)`. The whole member reaches the predicate now.

Measured consequence, and it is the number the review predicted: the mutant's
corpus witnesses moved from **48 over-refusals of valid frames** to **6**, all
on the *accepted* side — `encoding/escaped_{schema, protocol_version,
runtime_key, launcher_pid, exec_plan_digest}` and `encoding/all_escaped`. Zero
admitting witnesses, which is correct: the mutant is stricter than the
specification, never looser.

Revision 8's stated conclusion about that dimension — *a gate proved only by
inputs it must reject cannot be shown to admit what it must admit* — was right.
The number attached to it was measuring something else.

### Rework 1 — preserve the complete `frameMember`

Done, as above. `P23.F/the_corrected_mutant_is_not_reject_all` asserts both
halves directly: the plain valid frame is accepted under the mutant, and the
byte-equivalent all-escaped valid frame is refused `frame_unknown_field(schema)`.

### Rework 2 — a discriminating production-entry control

Done, and deliberately generalised past the one mutant. P23.C already carried the
escaped-valid → over-refusal half. **P23.E** adds the plain-valid → `execve` half
for **all eighteen** mutants, one process each at the real `runtime-launch` entry
point — because the mutant that failed was not one anybody would have singled
out, and a control written only for the mutant already known to be broken is the
row-by-row arrangement three earlier reviews already defeated.

P23.E carries its own negative: `control_reject_all_probe_MUST_redden` drives the
same entry point with `reject_all_probe` and requires
`protocol_violation/frame_unknown_field`. Without it, P23.E is eighteen positive
assertions and nothing shows it would ever fail.

### Rework 3 — make `blind` executable evidence

Done, and made **bidirectional**, because revision 8 was wrong in both
directions.

`p23d_blindness_calibration_test.go` carries an inventory of the **19
hand-written rows** in three named baselines — `rev6` (the six rows revision 6
wrote for review `RUN-260825-a8a4ef`), `rev7` (the ten rows revision 7 wrote for
review `RUN-260825-9d5cff`), `review_c71188` (the two frames that review minted,
plus a control) — each row already existing as an assertion in P22 or in a
reviewer attack file. P23.D recomputes, for every mutant, the set of baselines it
is green on, and requires it to **equal** the declared `blindTo`. An over-claim
fails. An under-claim fails too.

*Green on a baseline* is defined as: for every row, the mutant's decoder verdict
— accept, refusal `reason` **and** named member — is identical to the specified
decoder's. That is strictly stronger than "the row's test still passes", and
stronger in the safe direction: decoder agreement on a row implies the whole
downstream launcher path agrees on it, same `fr`, same comparisons, same exit.
The converse is deliberately not claimed, and §12.3.1 says so —
`unknown_case_folded` diverges at the decoder on `unknown_case_variant_wrong_value`
while the P22 production row stays green, which is exactly why that row has been
documented as non-discriminating since revision 7.

Each baseline must carry more than its valid control, or blindness on it would be
vacuous, and P23.D checks that too. A blind mutant additionally owes a
**corpus-only** witness, proved absent from the inventory **by bytes** rather
than assumed from the two generators being separate code.

**Two measured corrections, both produced by P23.D on its first run rather than
by reasoning:**

1. `dup_only_exactly_two_total` and `unknown_allow_over_32` are **not** blind to
   `review_c71188`. Those rows are precisely what caught them — the mutants exist
   because that review minted an arity-3 frame and a 33-byte name. Revision 8's
   prose said "green on the entire revision-7 table **and on both
   `RUN-260825-c71188` rows**", which is self-contradictory on its face and which
   nothing executed.
2. `dup_only_when_separated` is caught by `dup_same_exec_plan_digest`. The
   same-value duplicate rows **append** the repeat, and `exec_plan_digest` is the
   **last** member in the frame's field order, so that one row of the five places
   the two occurrences *adjacent*. I had reasoned that every hand-written
   duplicate was separated; measurement said otherwise on the first run, and I
   corrected the declaration rather than the measurement.

Final measured distribution: **four** mutants blind to all three baselines
(`unknown_ascii_only`, `unknown_nonempty_only`, `dup_keyed_on_wire_form`,
`unknown_by_wire_form`), **ten** blind to some baseline but not all, **four**
caught by a row that already existed. Revision 8 claimed seven blind to
everything.

### Rework 4 — repackage, rerun uncached offline, update every count

Done. Every number in the specification that came from the broken mutant is
corrected: 48 → 6 witnesses, seven blind → four, 155 → 200 assertions. The spec's
own history is annotated rather than rewritten: §16.2's revision-8 rows carry an
explicit *"corrected by revision 9"* note beside the counts that are now known
wrong.

- `TASK-260825-imfmgz_probe-rev9-module.tar.gz` — verified identical to the tree
  the results log was produced from, `vendor/` included.
- `TASK-260825-imfmgz_probe-rev9-results.log` — **200 passes, 0 failures,
  190.619s**, `go clean -testcache` first, `GOPROXY=off GOFLAGS=-mod=vendor`, on
  darwin 26.5.1 arm64 go1.25.5.
- `TASK-260825-imfmgz_probe-rev9-selfcontained-rerun.log` — the packaging
  property preserved: extracted into an empty directory, cache cleared, offline:
  `ok 191.522s`.
- `TASK-260825-imfmgz_rev9-repo-baseline.log` — `tools/agents-infra` build, vet
  and all three test packages, as a **no-regression control**, not a test of the
  delta. The delta is the specification and `LOGBOOK.md`.

The review asked for no protocol bump *if the specified decoder's admitted set is
unchanged*. That is now a measurement, not a claim — see the self-found item
below.

### Rework 5 — propagate only the concrete obligation to `TASK-260825-lsojra`

Done. No board element created, removed or re-scoped. Six checklist items added
to `lsojra`, all citing §12.3.1. `1lc8o7` receives none: §13 is untouched.
Recorded in `TASK-260825-imfmgz_decomposition-verification-rev9.md`.

### Self-found — the no-bump claim was prose for three revisions

§9 makes a protocol bump depend on whether the **specified** decoder's admitted
set moved. Revisions 7, 8 and now 9 each assert it did not. Until now that was an
argument.

Revision 8's wiring is retained under the name `rev8_unknown_wiring` — explicitly
not a gate mutant — and P23.F requires it to agree with revision 9's on the whole
generated corpus **and** every hand-written row, in verdict, reason and named
member: **417 frames**, all agreeing. The F1 repair is confined to mutant
selection, so §9 leaves no client a bump would protect and `protocol_version`
stays 6.

## What is normative now (spec §12.3.1)

Three obligations on every shape-gate mutant, all executed:

1. **It must still admit the plain valid frame.** Every mutant narrows or deletes
   a *refusal* clause, and no such change can turn away a frame the specification
   admits — including the wire-form mutant, whose defect is on the escaped
   spelling only. Checked at the decoder (P23.B) and at the production entry
   (P23.E).
2. **It must declare the side it disagrees on and disagree only there.** Exactly
   one mutant over-refuses; seventeen admit. A mutant that has silently become
   the opposite defect no longer reports a large kill count.
3. **Blindness is measured, never declared.** Over-claim and under-claim fail
   alike; a blind mutant owes a corpus-only witness identified by bytes.

Plus `reject_all_probe`, the review's defect reproduced on purpose, with the
suite required to show all three rules reddening on it (P23.F). Without that,
§12.3.1 is positive-path-only evidence about the harness — the same defect one
level up.

## Attack first, ranked

1. **The inventory is itself hand-written.** P23.D measures blindness against 19
   rows I chose. If a legacy assertion exists that is not in the inventory, a
   `blindTo` claim can still be an over-claim against the real row set. Every row
   is traceable to a P22 case or a reviewer attack file and the origins are named
   in the source, but nothing *derives* the inventory from the P22 tables — the
   two could drift. The honest statement of the claim is "blind to these 19
   named rows", and that is what the spec says.
2. **Decoder-layer greenness is a proxy for test greenness.** §12.3.1 argues the
   implication (decoder agreement ⇒ the whole downstream path agrees) and it
   holds because the launcher's remaining steps are a pure function of `fr`. A
   reviewer who disagrees with that argument should attack it directly: it is the
   only thing standing between a cheap 342-evaluation measurement and 342
   process spawns.
3. **P23.E asserts an invariant I reasoned about, not one I derived.** "No
   refusal-clause narrowing can turn away a valid frame" is true of the eighteen
   mutants in the table. It is not a theorem about mutants in general, and a
   future mutant that legitimately over-refuses the plain frame would have to
   weaken rule 1 rather than fail it. State the exception if one is ever needed;
   do not delete the rule.
4. **The direction field is declared, and rule 2 only checks consistency with the
   corpus.** A mutant whose declared direction and measured witnesses agree can
   still be the wrong mutant in some other way — rule 2 catches a reject-all fake
   and a direction flip, not an arbitrary miswiring. Rules 1 and 3 are what
   narrow the remaining space, and the three together are the claim.
5. **`reject_all_probe` is one broken shape.** It is the shape this review found,
   reproduced faithfully, and P23.F shows all three rules reddening on it. It is
   not a proof that the rules catch every possible broken mutant.
6. **Revision 2-6 probe modules were not rerun by this producer.** They are
   untouched since revision 8, where they were reconstructed and rerun green, and
   the revision-9 delta is confined to the revision-9 module. Stated rather than
   quietly omitted.
7. **The Go module path is still `taskboard.local/imfmgz/rev8probe` and the
   package is still `probe7`.** Both are carried forward unchanged on purpose, so
   an archive diff against revision 8 is exactly the evidence delta and nothing
   else.
