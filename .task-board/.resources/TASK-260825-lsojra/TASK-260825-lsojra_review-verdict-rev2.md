# TASK-260825-lsojra — reviewer verdict, Change Request rev2

- Reviewer run: `RUN-260825-e2bfd4` (claude-opus-5/high)
- Change Request: `CR-TASK-260825-lsojra-2` revision 2
- Branch `task-board/story/STORY-260825-1r7z9o`, worktree HEAD `8518d45`, **4 commits behind `main`**
- **Verdict: ACCEPTED.** Blocking finding F1 from reviewer run `RUN-260825-bd12e8` is closed, and the
  closure was proved by narrowing, not by reading the diff.

## 1. On `repository_delta=empty`

The empty delta is a **CR base-capture artifact, not an empty producer run**, and this is the second
consecutive round it has appeared on this element. Verified from the object store, not from the
producer's account:

| Fact | Value |
| --- | --- |
| CR base OID | `8518d4509c7feabe8aef2175825b194e065e979a` |
| CR candidate tree OID | `eedf0fa9525e1cd873953a1f7a40e0f69c2260cf` |
| `git rev-parse 8518d45^{tree}` | `eedf0fa9525e1cd873953a1f7a40e0f69c2260cf` — **identical** |
| `git rev-parse 8518d45^` | `fc0c69e` — the previously reviewed candidate |

The base was captured **at** the producer's rework commit rather than before it, so base tree ==
candidate tree by construction. The producer did change repository files. The real reviewable delta
is `fc0c69e..8518d45`: 6 files, +226/-47, of which two are production
(`pi_shared_protocol.go`, `pi_shared_launcher_darwin.go`), three are tests, one is `LOGBOOK.md`.
That delta is what I reviewed. Accepting here is **not** accepting a no-op run; the board mechanics
are what produced the empty patch, and the orchestrator should treat the base-capture behaviour on
this element as a known defect of the CR snapshot, not as a signal about the work.

## 2. What F1 was, and what the repair does

F1: the spec §12.3.1 rule-1 production-entry control was **inert**. `TestMain` installed a seam at
`sharedAuthorizationDecode` that ran the specified decoder on the delivered bytes and then evaluated
the selected mutant against a **constant harness-built plain frame**. The 18 one-process-per-mutant
runs therefore executed the unmutated gate, and `reject_all_probe` was not selectable at the
production entry at all.

The repair removes the launcher-level seam entirely — `RunSharedRuntimeLauncher` now calls the
concrete `decodeSharedRuntimeAuthorizationFrame` — and moves the single test seam down to
`sharedAuthorizationShapeDecision`, whose input is produced by the **production tokenizer** and
carries each member's decoded name, raw wire spelling and raw value. The calibration was rewired the
same way: `authMutantVerdictFromProductionBytes` now tokenizes the actual corpus bytes through
`tokenizeSharedAuthorizationFrame` before calling any mutant predicate.

## 3. Discriminating evidence I produced myself

Every measurement below was run by this reviewer against the delivered tree, in a scratch copy of the
module under `.temp/RUN-260825-e2bfd4/probe/`. The worktree was not modified (`git status` clean).

### M1 — the F1 wiring reproduced verbatim must redden the new tests

I re-installed the pre-repair wiring inside the delivered code: run the specified gate on the
delivered bytes, then evaluate the mutant against a constant harness-built plain frame.

| Test | Under delivered wiring | Under reproduced F1 wiring |
| --- | --- | --- |
| `…DriveProductionLauncherGate/unknown_ignored_admits_the_delivered_unknown_member` | PASS | **FAIL** |
| `…DriveProductionLauncherGate/wire_membership_over-refuses_an_escaped_valid_frame` | PASS | **FAIL** |
| `…DriveProductionLauncherGate/wire_keyed_duplicate_admits_decoded_duplicate` | PASS | **FAIL** |

The three new production-entry regressions are therefore discriminating, not positive-path-only.

### M1b — `reject_all_probe` selectability (closes P-B)

Withdrawing `reject_all_probe` from the `TestMain` entry allowlist makes
`TestSharedRuntimeRejectAllProbeReddensAtProductionEntry` fail with
`unknown shape mutant "reject_all_probe"` — i.e. exactly the P-B symptom the previous review
reported. With the delivered allowlist the probe refuses the plain valid frame at the real entry as
`protocol_violation` / `frame_unknown_field` / `mismatch_field=schema`, with a never-exec proof.

### M2 / M2b — the P-C defect class is now caught, and provably was not before

I registered a probe mutant `probe_unsupplied_dimension` whose predicate decides on
`frame.dimension`, a quantity the production decode path never supplies. This is the
`RUN-260825-86b7d5` F1 shape verbatim.

| Calibration routing | Result for `probe_unsupplied_dimension` |
| --- | --- |
| Delivered (`authMutantVerdictFromProductionBytes`) | **FAIL — `coverage hole: corpus kills no frame`** |
| Pre-repair routing restored (evaluate on the generator's own frame record) | **PASS — `KILLED … by occurrence/schema/x0; admits=340`** |

A fabricated kill count of 340 from a predicate production never evaluates, turned into a hard
failure. That is the F1 class closed at the calibration half, measured rather than asserted.

### Adversarial sweep at the real `runtime-launch` entry (22 raw-byte frames)

Driven as raw bytes through descriptor 3 into the real production entry point, not as a serialized
struct. All 22 behaved exactly as revision 8/9 requires, each refusal carrying a never-exec proof
(`exec.marker` absent **and** `kern.procargs2` never showing the target exec path):

- **Admitted (4):** all five members with a leading `\u` escape; all five fully escaped rune-by-rune;
  spaces/tabs between every token; the plain valid frame.
- **Shape refusals (14):** `schema` + its escaped form → `frame_duplicate_field/schema`;
  `Schema` carrying schema's own valid value → `frame_unknown_field/Schema`; empty member name;
  `exec_plan_digest_v2` prefix; unknown member first **and** last; whole frame repeated inline;
  same-value duplicate of the last member; a nested 200-deep unknown value; two objects; valid five
  plus a trailing token; an array; `{}` → `frame_missing_field/schema`; a missing member.
- **Ordering (2, the ones that matter):** a duplicate whose second value is **wrong** refuses
  `frame_duplicate_field`, and an unknown member alongside a **wrong** `protocol_version` refuses
  `frame_unknown_field` — proving step 4 runs before step 5 and the shape gate is not reachable-only
  through value-correct frames.
- **Equality, not range (3):** `protocol_version` 5, 999, and `"6"` are each refused —
  `runtime_authorization_mismatch/protocol_version` for the two numbers, `frame_unparseable` for the
  string. No `>= N` admission.
- One probe of mine failed and was **my** error, not the implementation's: a frame with embedded
  newlines is correctly a `protocol_violation`, because the frame is newline-delimited on the wire.

## 4. Independently re-verified, not taken on report

- `go build ./...`, `go vet ./...`, `gofmt -l .` — all clean.
- `GOOS=linux` and `GOOS=windows` `go build ./...` — both succeed.
- `go test ./internal/infra -run '^(TestSharedRuntime|TestSharedAuthorization)' -count=1` → ok 13.4s
- `go test -race ./internal/infra -run '^(TestSharedRuntime|TestSharedAuthorization)' -count=1` → ok 27.8s
- `go test ./internal/... -count=1` → ok (attachments 1.0s, infra 134.7s)
- `go test . -count=1` → ok 72.2s. There is no `./cmd/...` in this module; root + the two internal
  packages are the whole of `./...`, and all three are green.
- 13-gate attestation chain read at its single production call site
  (`connectAndAttestSharedRuntime`): gate 3b `broker_build` present; `p_stat != SZOMB` enforced on
  the broker peer **and** the runtime; the client verifies runtime uid, start time, exec path, argv
  and `/v1/models` itself rather than trusting the announcement. Operator commands use the same
  chain. Unchanged by this delta and intact.
- `sharedAuthMultisetVerdict` decides by comparing the decoded multiset against the compiled
  `sharedRuntimeAuthFields` — the same constant `writeAuthorizationFrame` (B11) writes the frame
  from — with unknown/duplicate/missing used only to classify the reported reason. No predicate over
  a frame-derived quantity. `protocol_version` is 6.

## 5. Non-blocking observations (recorded, did not gate acceptance)

1. **The evidence record's structural fields are literals, not measurements.**
   `sharedAuthDecodeEvidence.DecisionCallSite` and `.ConstantFieldSet` are emitted unconditionally
   inside `decodeSharedRuntimeAuthorizationFrame`, so under any of the 18 mutants the record still
   claims the compiled constant set while a different decision is installed. `DecodedKeys` and
   `ComparedFields` **are** measured. The §12.4 structural obligation is carried by those two
   measured fields plus the mutation calibration, which is what §12.4 itself demands ("neither half
   is evidence alone") — but the two literal fields discriminate nothing on their own and should not
   be cited as if they did. Pre-existing, not introduced here.
2. **A reason-code reordering on doubly-invalid frames.** `tokenizeSharedAuthorizationFrame` now runs
   the five value decodes *before* the trailing-content check. A frame that is both value-unparseable
   and has trailing content now reports `frame_unparseable` where it previously reported
   `frame_not_single_object`. Both are refusals under the same `protocol_violation` code and no
   admitted frame changed — R6/R7 confirm trailing content is still refused — but the corpus does not
   contain that combination, so the 417-frame agreement test would not have caught a real inversion
   here.
3. **The worktree is 4 commits behind `main`, and trunk moved inside the same package.** Trunk added
   `model_check.go` and changed `pi_config.go`, `pi_launch_posix.go`, `canonical_target.go` and
   `main.go` — files this work also touches. **My green suites were measured at `8518d45` and do not
   predict a trunk-green result.** Integration is the orchestrator's step and the producer correctly
   did not rebase, but the integration must re-run the suite on the merged tree.

## 6. Why acceptance, and why no repository change was not the question here

The CR's `repository_delta=empty` is a snapshot artifact (§1), so the "did this leaf deliver nothing"
question does not arise on its own terms — there is a real 6-file delta and I reviewed it. Judged on
that delta: the single blocking finding is closed, the closure is proved by two independent
narrowings that redden the delivered tests and by one that shows the pre-repair harness manufacturing
a 340-frame kill count out of nothing, the gate itself survived a 22-frame raw-byte attack at its real
entry point including the ordering and equality cases, and every suite is green under my own hands.

Acceptance evidence is recorded for the commit-owning mover. This reviewer supplied no `commit_ack`.
