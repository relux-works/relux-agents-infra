# TASK-260825-lsojra review verdict — CHANGES REQUESTED

- Reviewer run: `RUN-260825-bd12e8` (claude-opus-5/high)
- Change Request: `CR-TASK-260825-lsojra-1` revision 1
- Base OID / candidate tree: `fc0c69e` / `e9315121b4bc78f823d0ebf23f6745d278292562`
- Verdict: **changes_requested → `to-dev`**

## On `repository_delta=empty`

The empty delta is an artifact of when the Change Request captured its base, not
an empty producer run. `fc0c69e^{tree}` **is** the candidate tree, and the
producer run `RUN-260825-304bfa` ran 18:17Z–20:05Z while commits `a94e67e`
(22:43 +0300), `6146f24` (22:51) and `fc0c69e` (22:56) landed on
`task-board/story/STORY-260825-1r7z9o` — inside that window. The reviewable work
is the committed range, ~12.4k lines across 35 files versus `main`, and this
review was conducted against the tree at `fc0c69e`, not against a zero-path
patch. So the delta is not the finding; what the delta contains is.

## Finding F1 — BLOCKING. The §12.3.1 rule-1 production-entry control is a no-op

`§12.3.1` rule 1 requires every shape-gate mutant to be checked "at the decoder
(P23.B) **and** at the production `runtime-launch` entry point, one process per
mutant (P23.E). The production check is not optional: it is where a reviewer
reproduces it."

`TestSharedRuntimeEveryShapeMutantAdmitsPlainValidFrameAtProductionEntry`
(`internal/infra/pi_shared_launcher_test.go:370`) does spawn one process per
mutant. What those processes run is not the mutant.
`TestMain` (`internal/infra/pi_shared_integration_test.go:56-63`) installs:

```go
sharedAuthorizationDecode = func(data []byte) (...) {
    frame, evidence, decodeErr := decodeSharedRuntimeAuthorizationFrame(data)   // the SPECIFIED decoder
    if decodeErr != nil { return frame, evidence, decodeErr }
    verdict := authMutantVerdict(authFrame("production/plain-valid", "control", authBaseMembers()), mutantName)
    ...
}
```

The mutant verdict is computed against a **constant, harness-built frame**. The
bytes that arrived on descriptor 3 are decided by the unmutated production gate.
The mutant is never the decoder.

Reproduced, three probes, all against the real `runtime runtime-launch` entry in
a forked process (probe sources: `TASK-260825-lsojra_review-probe.go.txt`):

- **P-A — the mutant is not installed.** Drove the entry under
  `AGENTS_INFRA_SHARED_SHAPE_MUTANT=unknown_ignored` with a frame carrying
  `future_extension`. That mutant, by its own row, ignores unknown members, so an
  installed mutant must admit the frame and reach `execve`. Observed instead:
  `{"code":"protocol_violation","reason":"frame_unknown_field","mismatch_field":"future_extension"}`,
  `reached_execve=false`. The specified gate ran; the mutant did not.
- **P-B — §12.3.1's own negative never reaches the production entry.**
  `reject_all_probe` appears only in `pi_shared_shape_oracle_test.go` (lines 470,
  680, 681, 686, 699) — the decoder model. Driving it at the production entry is
  impossible: `TestMain`'s allowlist over `authShapeMutants` rejects the name and
  the process exits 1 with `unknown shape mutant "reject_all_probe"`, which is
  not the gate refusing anything. Spec line 3028 requires "rule 1 reddens at the
  decoder **and at the production entry**"; only the first half exists.
- **P-C — the control is structurally blind to the exact defect it closes.**
  Registered `wire_cleared_probe`, review `RUN-260825-86b7d5`'s F1 verbatim: a
  membership predicate that consults a field the production decode path does not
  supply (`member.wire != ""`; `sharedAuthDecodeEvidence.DecodedKeys` is
  `[]string` and carries no wire spelling at all). Under an honest wiring it
  refuses everything production can hand it and rule 1 must redden. Observed:
  `--- PASS: .../wire_cleared_probe (0.33s)`. Green.

Negative shapes: **positive-path-only evidence around a fake**, and **the check
present but not called from the path it claims to check**. The producer evidence
reports this as "one production runtime-launch process per mutant for the
plain-valid control" and as "reject-all harness negative" — the processes are
real, the mutants in them are not, and the negative is decoder-only.

I reran the producer's focused suite at `fc0c69e` unmodified:
`go test ./internal/infra -run 'TestSharedRuntimeAuthorizationShapeOracleDifferential|TestSharedRuntimeAuthorizationMutantCalibrationAndHarnessNegatives|TestSharedRuntimeEveryShapeMutantAdmitsPlainValidFrameAtProductionEntry|TestSharedRuntimeLauncher' -count=1`
→ `ok ... 4.337s`, exit 0. Green, with the control inert. That is the failure
mode revision 9 was written to prevent, one level up.

### What closing F1 requires

1. Install the selected mutant as the shape gate the production process actually
   decides with — the predicate must receive the members the production decoder
   produced from the delivered bytes, not a harness constant. P-A is the
   regression: under `unknown_ignored` a frame with an unknown member must reach
   `execve`.
2. Give the production decode path enough per-member fact for the predicate to
   be the mutant it claims to be. `unknown_by_wire_form` and
   `dup_keyed_on_wire_form` both decide on wire spelling, which the current
   evidence struct discards — that discard is why the constant was reachable for.
   Note the §12.4 constraint while doing this: the recorded multiset must stay
   the decoder-yielded names (item 47), so a wire form added for the mutants must
   not become what the gate or the record is keyed on.
3. Drive `reject_all_probe` at the production entry as a first-class selectable
   probe and assert it refuses **because the gate refused**, distinguishable from
   `unknown shape mutant`.
4. Re-measure the 18 rows after 1–3. Rule 2 direction and rule 3 blindness are
   currently measured against `authMutantVerdict` only; once the predicate sees
   production-derived members those numbers must be re-derived, not carried over.

## Secondary observations (not blocking on their own, fix while in there)

- **S1.** `passedSharedRuntimeGateOutcomes()` (`pi_shared_client_darwin.go:417`)
  emits a hardcoded 13×`{outcome: passed, source: attested}` list. The chain does
  return early on every failure, so reaching it implies all gates passed — but
  the reported outcome is a literal, not a record of what ran. If §10 status is
  read as gate evidence, record per-gate outcomes as they are decided.
- **S2.** The production decoder is reached through the package var
  `sharedAuthorizationDecode`, while the §12.2.30c differential decides against
  the concrete `decodeSharedRuntimeAuthorizationFrame`. That indirection is the
  seam F1 went through. The §12.4 shared-call-site obligation is currently met by
  the launcher writing its own `DecisionCallSite` string literal; consider
  proving identity rather than declaring it.

## What this review verified as sound

Not everything here is in doubt, and the rework should not disturb these:

- 13-gate client attestation with gate 3b (`broker_build_identity_mismatch`),
  kernel-verified broker pid/start-time, runtime uid/start-time/exec-path/argv
  and independent `/v1/models`, in one named production call site
  (`connectAndAttestSharedRuntime`), with `p_stat != SZOMB` on broker peer,
  runtime and process observation (`pi_shared_process_darwin.go:32`).
- The B12 step-4 shape gate itself: `tokenizeSharedAuthorizationFrame` enumerates
  member names via `json.Decoder.Token` with duplicates retained, and
  `sharedAuthMultisetVerdict` decides by multiset equality against the compiled
  `sharedRuntimeAuthFields`, with the unknown/duplicate/missing classification
  used only for `reason`/`mismatch_field`. No count-threshold, length, adjacency,
  case-folded or prefix predicate is in the decision. Escaped-key frames decode
  correctly in both directions.
- The 398-frame oracle differential agrees with the production decoder on
  verdict, reason and named member; the five equality comparisons and the four
  `runtime_launch_unauthorized` reasons are driven at the real entry point with
  a never-`execve` proof.
- `protocol_version` correctly stays 6 (items 38/44/50 stand).
- Top-level `agents-infra runtime status|stop|broker|runtime-launch` exists in
  production `main.go:510-567`.

The rework is F1 and the four steps under it. Nothing else needs reopening.
