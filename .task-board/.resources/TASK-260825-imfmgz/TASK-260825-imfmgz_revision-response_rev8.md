# TASK-260825-imfmgz — revision 8 response to review `RUN-260825-c71188`

Producer run: `RUN-260825-753d00`
Deliverable: `.research/260825_shared-local-runtime-broker.spec.md`, revision 8
Protocol version: **unchanged at 6** — see "Why no bump" below.

## The finding, and why revision 8 is not another table of rows

The review was right, and it was right for the third time in a row about the same
thing. Its two decoders keep **every** named revision-7 row green and reach
`execve`:

| Narrowing | Why every revision-7 row stayed green |
| --- | --- |
| refuse a repeat only at a total occurrence count of exactly **two** | every duplicate frame revisions 6 and 7 minted has arity two |
| apply the allowlist only to names of at most **32 bytes** | every unknown name revisions 6 and 7 minted is at most nineteen bytes |

Revision 7 answered the *previous* review by adding rows. This review then built
two narrowings on dimensions those rows did not vary. Adding two more rows loses
again, one dimension further out, and this task has now spent three revisions
demonstrating exactly that. **The rows were never the problem. Naming the
admitted class in advance is.**

So revision 8 changes two things, and they are halves of one claim.

### (a) The rule is stated constructively — §6.2 B12 step 4

The gate is no longer three clauses to be satisfied. It is one decision:

> `M` is admitted **iff** `M == {schema, protocol_version, runtime_key,
> launcher_pid, exec_plan_digest}` as a multiset.

with the three clauses demoted to a **classification of the difference** for the
refusal `reason` — which is what they always were, and which is precisely why
they were narrowable one at a time. Three clauses have three bounds to establish
and each can be narrowed independently; one equality against a compiled constant
has one, and it is not narrowable without changing the constant.

Two normative rules follow:

- **the decision may not be expressed as a predicate over any quantity derived
  from the frame** — not an occurrence count, a name length, a position, a
  folded or prefix match, or the bytes of a key. Every one of those is a
  plausible implementation of one clause and every one admits frames the equality
  refuses. An implementation that computes `M` and compares it to a constant
  cannot express any of them;
- **a member name is the name a JSON decoder yields, compared by byte equality**,
  not the bytes it was written as. See "Self-found" below.

### (b) The proof is a differential, not a table — §12.2.30c, probe P23

The predicate is small enough to implement twice, so it is. P23 carries an
**independent oracle** that reads the generator's own record of what it built —
never the bytes, never the decoder — and the decoder must agree with it on every
frame of a generated corpus, on accept/refuse, on the refusal `reason` **and** on
the named member. **No expected outcome is written next to any frame.**

That is the whole difference: a narrowed gate is caught because it *disagrees
with the oracle*, not because someone predicted the class it admits. The review's
two narrowings are killed here by generated frames, not by rows written to answer
them.

The mutant table inverts with it. It is no longer the coverage claim; it is the
harness's **calibration**. Each mutant must be killed *by the corpus*, the harness
reports which frame killed it, and **a mutant nothing kills fails the suite** as a
coverage hole reported by the harness rather than found by the next reviewer.

## Item-by-item closure

| Rework item | Where | Evidence |
| --- | --- | --- |
| 1. Bind "more than once" beyond arity two; cover member × arity or add a property obligation | §6.2 B12 step 4 (equality, no arity to special-case); §12.2.30c occurrence dimension | P23.A sweeps counts 0, 1, 2, 3, 4, 5, 8, 13 for **every** member plus multi-member and whole-object repetition — 44 frames. P23.B kills `dup_only_exactly_two_total` with **55** witnesses, first `arity/schema/x3`. P23.C drives arity 3 and arity 5 at the production entry |
| 2. Extend the open-name proof with length boundaries past the sample range; add the length narrowing; state the generated coverage rather than calling four names a class | §12.2.30c length + random dimensions, with the dimension table stating what is swept and why | P23.A sweeps 0, 1, 2, 3 and every power of two from 8 to 1024 with each boundary ±1 (26 frames), plus 128 seeded random names over the printable byte space. P23.B kills `unknown_allow_over_32` with **77** witnesses, first `length/unknown_33B`. P23.C drives 33 and 1024 bytes at the production entry |
| 3. Rerun uncached with the valid frame still reaching `execve`; attach a self-contained module | — | 155 passing assertions, 0 failures, 186.9s, cache cleared first. The all-valid control `execve`s in P22.A, P22.F, P22.I and P23.C. Archive verified by extracting into an empty directory and running `GOPROXY=off`: `ok 186.7s` |
| 4. Propagate only concrete new implementation obligations; no new board leaf unless scope changed | — | Five checklist items added to `TASK-260825-lsojra`; no element created, removed or re-scoped; `TASK-260825-1lc8o7` receives none, for the reason recorded in the decomposition verification |
| F2. Attach `go.mod`, `go.sum`, TestMain/helper source and the identity helper | `TASK-260825-imfmgz_probe-rev8-module.tar.gz` | One archive with `go.mod`, `go.sum`, the **vendored** dependency, `helper_main_test.go`, `identity.go`, the launcher, the decoder and all three probe suites, plus a README with the one command that runs it |

## Self-found, and it would have been the next finding

Nothing in revisions 6 or 7 said whether a member name is the **decoded** name or
the bytes it was written as. Both are implementable and they disagree:

- `{"\u0073chema": …}` — a **valid** frame under the decoded reading, an unknown
  member under the wire reading;
- `"schema"` and `"\u0073chema"` in one object — the **same member twice** under
  the decoded reading, two members under the wire reading.

§6.2 B12 step 4 now makes the decoded reading normative. Two mutants cover the two
directions, and one of them is why this matters more than it looks:
`unknown_by_wire_form` **admits nothing extra** — it is *stricter* — and it
**refuses a valid frame**. Forty-eight generated frames catch it. **No corpus
built only from frames that must be refused ever could**, and every corpus this
task has built before revision 8 was exactly that. That is a class of defect
three revisions of evidence were structurally unable to see.

## Why no bump

`protocol_version` stays at **6**. §9's rule read the other way is the reason: the
version gate is enforced by the *reading* side, so what a bump protects is the
client that would otherwise attach to a broker applying a different gate set. The
set of frames the launcher admits did not move — P23.A confirms the revision-7
decoder already implements the revision-8 equality, on all 398 frames, including
`reason` and named member. A bump with no gate-set change protects no client and
would invalidate every deployed pairing for nothing.

## Evidence

| Artifact | What it is |
| --- | --- |
| `TASK-260825-imfmgz_probe-rev8-module.tar.gz` | the self-contained module: `go.mod`, `go.sum`, vendored `golang.org/x/sys`, helper harness, identity helper, launcher, decoder with all 18 mutants, P21/P22/P23 |
| `TASK-260825-imfmgz_probe-rev8-results.log` | full uncached `-v` run: 155 passes, 0 failures, 186.9s, darwin 26.5.1 arm64, go1.25.5 |
| `TASK-260825-imfmgz_probe-rev8-selfcontained-rerun.log` | vendored/offline run, and the clean-extract run that discharges F2 the way a reviewer reproduces it |
| `TASK-260825-imfmgz_probe-rev8-earlier-rerun.log` | revision 2-6 probe modules reconstructed from board resources and rerun uncached, all green |
| `TASK-260825-imfmgz_rev8-repo-baseline.log` | `go build`, `go vet`, `go test ./internal/...` and `go test .` — a **no-regression control**, not a test of the delta; the repository delta is the spec and `LOGBOOK.md` |
| `TASK-260825-imfmgz_probe-rev8-p23_frame_shape_oracle_test.go` | P23 loose, for reading |
| `TASK-260825-imfmgz_probe-rev8-frame_shape_test.go` | the decoder and every mutant, loose, for reading |

## Attack first, in revision 8

Ranked. The first two are where I think a reviewer should actually go.

1. **The corpus is still a sample, and §12.2.30c says so.** A narrowing whose
   admitted class is disjoint from all seven dimensions survives P23 exactly as
   `count == 2` survived revision 7. The counterweight is structural — B12 step 4
   forbids predicates over derived quantities and §12.4 requires the production
   launcher to record the multiset it decided on — but that is a *specification*
   obligation discharged by `TASK-260825-lsojra`, not by anything running here.
   If a reviewer finds an eighth dimension, the honest reading is that the
   structural half is carrying more weight than the empirical half, not that P23
   needs a row for it.
2. **The oracle and the decoder could share an assumption.** The oracle reads the
   generator's member list rather than parsing bytes, which is the strongest
   independence available inside one module — but both live in one package, and
   the generator's idea of what it built is itself a model. A frame whose bytes
   decode to a multiset different from the one the generator recorded would fool
   *both*. The structural rows partly cover this; a reviewer minting bytes by
   hand and checking them against both is the real test.
3. **P23 models the launcher at the syscall layer**, like every probe before it.
   §12.4 is what makes it production evidence and `lsojra` owes it — the same gap
   behind the revision-3, revision-5 and revision-7 F1s. P23.C's cross-layer
   agreement assertion narrows it (the layer-1 verdict must predict the
   production outcome on every frame driven) but agreement on nine frames is not
   identity.
4. **Seven mutants are labelled "blind" and I wrote five of them.** The claim is
   narrower than it may read: those five were reachable from the *generator's
   dimensions*, and the dimensions were chosen from the structure of the decision
   rather than from a list of attacks. The claim is "the corpus covers the
   decision's input structure", not "no narrowing survives". A reviewer who holds
   that a producer-authored blind mutant is weaker calibration than a
   reviewer-authored one is right, and the review's own two are in the table for
   exactly that reason.
5. **The zero-length-member row's field assertion is degenerate** —
   `mismatch_field` is `omitempty`, so `""` cannot distinguish "named the empty
   member" from "named nothing". It is labelled in both the spec and the probe and
   is carried by its `reason` and its mutant. A reviewer may hold that the refusal
   should carry an explicit presence marker instead.
6. **The seeded random dimension is 224 of 398 frames** and could be doing more
   work than it looks — or less. It is reproducible, but a reviewer should check
   that its generator can actually reach the near-miss shapes the deterministic
   dimensions cover, rather than mostly producing obviously-unknown names.
7. **The revision-4 evidence cannot be rerun from its attachments** (`notifySignal`
   is defined nowhere) and I reconstructed one line to run it. That rerun is
   labelled a reconstruction. A reviewer may reasonably hold that revision 4's
   probe claims are therefore unverified until that module is reattached whole.
