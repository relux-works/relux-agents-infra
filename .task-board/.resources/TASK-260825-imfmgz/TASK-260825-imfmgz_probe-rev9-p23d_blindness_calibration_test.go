package probe7

// P23.D / P23.E - revision 9, closing review RUN-260825-86b7d5 finding F1.
//
// WHAT THE REVIEW FOUND. Revision 8's `unknown_by_wire_form` mutant was supposed
// to be the one mutant that fails in the ADMITTING-NOTHING direction: match the
// allowlist against the bytes a key was written as instead of against the name
// it decodes to, so a valid frame whose key is spelled `"schema"` is
// refused. It was wired through `isAllowedName(m.name, mutant)`, which rebuilds
// a `frameMember` from the decoded name alone and clears `wire`. The empty wire
// string matches no allowed literal, so the mutant refused every member of every
// frame. It was a reject-all fake.
//
// The harness reported it as KILLED anyway, because P23.B's kill condition was
// "disagrees with the oracle on at least one frame" and a reject-all mutant
// disagrees with the oracle on every accepted frame. Its 48 witnesses were all
// ordinary valid frames, beginning `arity/schema/x1`. And its `blind` label was
// a boolean in a table: nothing executed the claim.
//
// THE GENERAL DEFECT, which is the part worth carrying forward: a mutation
// harness's own controls are gates, and this task's evidence rule applies to
// them exactly as it applies to the launcher. "The mutant was killed" is a
// positive result about the harness. Nothing in revision 8 would have failed if
// a mutant were the wrong mutant. Three things fix that, and all three are
// structural rather than another row:
//
//	1. ANTI-REJECT-ALL (P23.B). Whatever a mutant narrows, it must still admit
//	   the plain valid frame. A mutant that refuses it is not a narrowing of a
//	   refusal clause; it is a broken mutant. This alone reddens the exact defect.
//	2. DECLARED DIRECTION (P23.B). Each mutant declares whether it disagrees with
//	   the oracle by ADMITTING frames the specification refuses or by OVER-
//	   REFUSING frames it must admit. The measured witness set must match. A
//	   reject-all fake declared `admits` fails; the same fake declared
//	   `over_refuses` is then caught by rule 1.
//	3. MEASURED BLINDNESS (this file). `blindTo` names the hand-written baselines
//	   a mutant is green on, and P23.D measures the truth and requires equality.
//	   Over-claiming fails, and so does under-claiming - the second matters
//	   because revision 8 under-claimed too: it called
//	   `dup_only_exactly_two_total` and `unknown_allow_over_32` blind to "the
//	   entire revision-7 table AND both RUN-260825-c71188 rows", when those two
//	   mutants exist precisely BECAUSE the c71188 rows catch them.
//
// WHAT "GREEN ON A BASELINE" MEANS HERE, stated precisely because the previous
// revision's version of this claim was prose. A mutant is green on baseline B
// iff for every row in B the mutant's DECODER VERDICT - accept, refusal reason
// and named member - is identical to the specified decoder's. That is strictly
// stronger than "the P22 test row still passes": a divergence the later equality
// comparison happens to mask still counts as a divergence here. Strictly
// stronger in the safe direction, because decoder agreement on a row implies the
// launcher's whole downstream path agrees on it - same `fr`, same comparisons,
// same exit. So `blindTo` containing B is a sound basis for "no row in B catches
// this mutant", and the converse is deliberately not claimed:
// `unknown_case_folded` diverges at the decoder on the row
// `unknown_case_variant_wrong_value` while the P22 production row stays green,
// which is exactly why that row is documented as non-discriminating.

import (
	"fmt"
	"strings"
	"testing"
)

// ---- the hand-written baselines --------------------------------------------

const (
	baselineRev6  = "rev6"
	baselineRev7  = "rev7"
	baselineC7118 = "review_c71188"
)

var allBaselines = []string{baselineRev6, baselineRev7, baselineC7118}

// legacyRow is one frame that some earlier revision or review WROTE BY HAND.
// Every row here already exists as an assertion in P22 or in the reviewer's
// attack files; this inventory is the same frames gathered so blindness can be
// computed instead of asserted.
type legacyRow struct {
	baseline string
	id       string
	build    func(t *testing.T, valid []byte) []byte
	origin   string
}

func legacyRows() []legacyRow {
	return []legacyRow{
		// -- rev6: the rows revision 6 wrote to close review RUN-260825-a8a4ef ---
		{baselineRev6, "valid_control", func(t *testing.T, b []byte) []byte { return b },
			"P22.A - the all-valid control every refusal row depends on"},
		{baselineRev6, "unknown_caller_chosen_field",
			func(t *testing.T, b []byte) []byte { return withMember(b, `"caller_chosen_field":"ignored"`) },
			"P22.B / P22.F - review RUN-260825-a8a4ef's first attack frame"},
		{baselineRev6, "dup_wrong_first",
			func(t *testing.T, b []byte) []byte { return prepMember(b, `"protocol_version":999`) },
			"P22.C / P22.F - review RUN-260825-a8a4ef's second attack frame"},
		{baselineRev6, "dup_wrong_last",
			func(t *testing.T, b []byte) []byte { return withMember(b, `"protocol_version":999`) },
			"P22.C - the reverse duplicate order, against a first-wins decoder"},
		{baselineRev6, "missing_runtime_key",
			func(t *testing.T, b []byte) []byte { return dropMember(t, b, "runtime_key") },
			"P22.D - the absent-member row"},
		{baselineRev6, "trailing_object",
			func(t *testing.T, b []byte) []byte {
				return append(append([]byte{}, b...), []byte(`{"protocol_version":999}`)...)
			},
			"P22.E - bytes after the frame"},

		// -- rev7: the rows revision 7 wrote to close review RUN-260825-9d5cff --
		{baselineRev7, "valid_control", func(t *testing.T, b []byte) []byte { return b },
			"P22.I - the control the narrowing rerun depends on"},
		{baselineRev7, "dup_same_schema", dupSame("schema"), "P22.G - same-value duplicate, exhaustive over the closed five"},
		{baselineRev7, "dup_same_protocol_version", dupSame("protocol_version"), "P22.G / P22.I - review RUN-260825-9d5cff's first surviving narrowing"},
		{baselineRev7, "dup_same_runtime_key", dupSame("runtime_key"), "P22.G"},
		{baselineRev7, "dup_same_launcher_pid", dupSame("launcher_pid"), "P22.G"},
		{baselineRev7, "dup_same_exec_plan_digest", dupSame("exec_plan_digest"), "P22.G"},
		{baselineRev7, "unknown_future_extension", unknownNamed("future_extension"),
			"P22.H / P22.I - review RUN-260825-9d5cff's second surviving narrowing"},
		{baselineRev7, "unknown_case_variant", unknownCaseVariantOf("Schema", "schema"),
			"P22.H - a case variant carrying the allowed member's own valid value"},
		{baselineRev7, "unknown_case_variant_wrong_value", unknownNamed("Schema"),
			"P22.H - the same case variant with a wrong value; recorded as non-discriminating at the production entry"},
		{baselineRev7, "unknown_prefix_variant", unknownNamed("exec_plan_digest_v2"),
			"P22.H - an allowed name used as a prefix"},

		// -- review_c71188: the two frames review RUN-260825-c71188 minted -------
		{baselineC7118, "valid_control", func(t *testing.T, b []byte) []byte { return b },
			"the control - without it a reject-all mutant would measure as blind to this baseline"},
		{baselineC7118, "arity_three_protocol_version",
			func(t *testing.T, b []byte) []byte {
				return dupSameValue(t, dupSameValue(t, b, "protocol_version"), "protocol_version")
			},
			"review RUN-260825-c71188 - a third occurrence, defeating count == 2"},
		{baselineC7118, "unknown_name_33_bytes", unknownNamed(nameOfLength(33)),
			"review RUN-260825-c71188 - a 33-byte unknown name, defeating length <= 32"},
	}
}

// legacyDivergences returns, per baseline, the rows on which `mutant` produces a
// different decoder verdict than the specified decoder.
func legacyDivergences(t *testing.T, valid []byte, mutant string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, row := range legacyRows() {
		frame := row.build(t, valid)
		wantAccept, wantReason, wantField := decoderVerdict(frame, "")
		gotAccept, gotReason, gotField := decoderVerdict(frame, mutant)
		if gotAccept == wantAccept && gotReason == wantReason && gotField == wantField {
			continue
		}
		detail := fmt.Sprintf("%s (specified: %s, mutant: %s)", row.id,
			verdictString(wantAccept, wantReason, wantField),
			verdictString(gotAccept, gotReason, gotField))
		out[row.baseline] = append(out[row.baseline], detail)
	}
	return out
}

func verdictString(accept bool, reason, field string) string {
	if accept {
		return "accept"
	}
	if field == "" {
		return "refuse/" + reason
	}
	return "refuse/" + reason + "(" + field + ")"
}

// ---- P23.D: blindness is measured, never declared ---------------------------

func TestP23D_BlindnessIsMeasuredNotDeclared(t *testing.T) {
	_, prof := p21Project(t, "73", 4000)
	valid := validBytes(t, prof, p21Key(prof), 4242, "/tmp")
	corpus := buildCorpus(t, valid)

	// The inventory must be non-empty on every baseline, or "green on baseline B"
	// would be vacuously true and every mutant would measure as blind to it.
	counts := map[string]int{}
	for _, row := range legacyRows() {
		counts[row.baseline]++
	}
	for _, b := range allBaselines {
		if counts[b] < 2 {
			t.Fatalf("baseline %s has %d rows: a baseline with no rows, or with only a control, makes blindness vacuous", b, counts[b])
		}
	}
	t.Logf("hand-written baselines: %s", func() string {
		var parts []string
		for _, b := range allBaselines {
			parts = append(parts, fmt.Sprintf("%s=%d rows", b, counts[b]))
		}
		return strings.Join(parts, " ")
	}())

	for _, mu := range shapeMutants {
		mu := mu
		t.Run(mu.name, func(t *testing.T) {
			div := legacyDivergences(t, valid, mu.name)

			measured := map[string]bool{}
			for _, b := range allBaselines {
				if len(div[b]) == 0 {
					measured[b] = true
				}
			}
			declared := map[string]bool{}
			for _, b := range mu.blindTo {
				declared[b] = true
			}

			for _, b := range allBaselines {
				switch {
				case declared[b] && !measured[b]:
					t.Fatalf("OVER-CLAIM: mutant %s is declared blind to baseline %s, but %d of its rows catch it: %s",
						mu.name, b, len(div[b]), strings.Join(div[b], "; "))
				case !declared[b] && measured[b]:
					t.Fatalf("UNDER-CLAIM: mutant %s is NOT declared blind to baseline %s, yet every row in %s produces the same verdict under it. The table understates what the corpus is carrying; say so or add a row that catches it.",
						mu.name, b, b)
				}
			}

			for _, b := range allBaselines {
				if measured[b] {
					t.Logf("BLIND to %s: all %d hand-written rows produce the identical decoder verdict under this mutant", b, counts[b])
					continue
				}
				t.Logf("caught by %s: %s", b, strings.Join(div[b], "; "))
			}

			if len(mu.blindTo) == 0 {
				t.Log("not blind to any hand-written baseline: an existing row already catches it")
				return
			}

			// A mutant blind to a baseline has to be killed by something else, and
			// the corpus is the something else. Report the witness - this is the
			// per-mutant version of the claim revision 8 made once, in prose, for a
			// group of seven.
			//
			// The witness must be a frame no hand-written row already contains,
			// which is checked by bytes rather than assumed from the two generators
			// being separate code. Otherwise "the corpus kills what the rows miss"
			// could be satisfied by the corpus having regenerated one of the rows.
			handWritten := map[string]bool{}
			for _, row := range legacyRows() {
				handWritten[string(row.build(t, valid))] = true
			}
			var witness string
			for _, c := range corpus {
				wantAccept, _, _ := oracleVerdict(c)
				gotAccept, gotReason, _ := decoderVerdict(c.bytes, mu.name)
				if gotAccept == wantAccept {
					continue
				}
				if handWritten[string(c.bytes)] {
					continue
				}
				if gotAccept {
					witness = fmt.Sprintf("%s [%s] admitted by the mutant, refused by the oracle", c.id, c.dim)
				} else {
					witness = fmt.Sprintf("%s [%s] accepted by the oracle, refused %s by the mutant", c.id, c.dim, gotReason)
				}
				break
			}
			if witness == "" {
				t.Fatalf("mutant %s is blind to %s and NO generated frame outside the hand-written inventory kills it: the corpus does not bind the class it narrows",
					mu.name, strings.Join(mu.blindTo, "+"))
			}
			t.Logf("corpus-only witness: %s", witness)
		})
	}

	// The summary is a claim in its own right and it is computed, not typed.
	fully, partly, none := 0, 0, 0
	for _, mu := range shapeMutants {
		switch len(mu.blindTo) {
		case len(allBaselines):
			fully++
		case 0:
			none++
		default:
			partly++
		}
	}
	if fully == 0 {
		t.Fatal("no mutant is blind to every hand-written baseline: the calibration does not show that the corpus generalizes past the rows someone wrote")
	}
	t.Logf("PASS P23.D: %d mutants - %d blind to every hand-written baseline, %d blind to some, %d caught by an existing row; every label measured against %d rows",
		len(shapeMutants), fully, partly, none, len(legacyRows()))
}

// ---- P23.E: the anti-reject-all control at the PRODUCTION entry -------------

// TestP23E_EveryMutantStillExecsThePlainValidFrame is review RUN-260825-86b7d5's
// rework item 2, generalized past the one mutant that failed it.
//
// P23.B enforces the same invariant at the decoder in microseconds. This runs it
// where it is production evidence: the real `runtime-launch` entry point, one
// process per mutant, plain valid frame in, `execve` on the composed target out.
// A reject-all mutant reddens here, and nothing else does - every mutant in the
// table narrows or deletes a REFUSAL clause, and no such change can turn an
// accepted frame away.
//
// It is deliberately every mutant and not only the seven P23.C carries rows for.
// The defect this closes was in a mutant nobody would have singled out.
func TestP23E_EveryMutantStillExecsThePlainValidFrame(t *testing.T) {
	dir, prof := p21Project(t, "74", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	for _, mu := range shapeMutants {
		mu := mu
		t.Run(mu.name, func(t *testing.T) {
			run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env(mu.name, "")})
			run.rawFrame(t, validBytes(t, prof, key, run.pid, cwd))
			id, exec := run.becameTarget(target)
			if !exec {
				run.waitExit(t)
				t.Fatalf("REJECT-ALL MUTANT: %s (%s, %s) refused the plain valid frame at the production entry with %+v. Every kill this mutant is credited with is satisfied by a launcher that refuses everything.",
					mu.name, mu.kind, mu.origin, run.refusal(t))
			}
			t.Logf("CONTROL: mutant %s still execve'd the plain valid frame %v", mu.name, id.Argv)
		})
	}

	// And the control's own control. Every assertion above is positive - each
	// mutant DID exec - so on its own this test is exactly the positive-path-only
	// evidence the task's evidence rule refuses. `reject_all_probe` is the
	// revision-8 defect reproduced, and it must redden here, at the production
	// entry, for the same reason and with the same refusal a reviewer would see.
	t.Run("control_reject_all_probe_MUST_redden", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env("reject_all_probe", "")})
		run.rawFrame(t, validBytes(t, prof, key, run.pid, cwd))
		if id, exec := run.becameTarget(target); exec {
			t.Fatalf("the reject-all probe reached execve %v: this test cannot tell a broken mutant from a working one", id.Argv)
		}
		run.waitExit(t)
		ref := run.refusal(t)
		if ref.Code != "protocol_violation" || ref.Reason != "frame_unknown_field" {
			t.Fatalf("got %+v, want protocol_violation/frame_unknown_field", ref)
		}
		t.Logf("REDDENS: the reject-all probe refused the plain valid frame %+v - every row above would have been satisfied by this, and revision 8 shipped a mutant that was exactly this", ref)
	})

	t.Logf("PASS P23.E: all %d mutants admit the plain valid frame at the production entry, and the reject-all probe does not", len(shapeMutants))
}

// TestP23F_TheHarnessCatchesABrokenMutant is the negative test for the three
// rules above. Without it, P23.B/D/E are themselves positive-path-only evidence:
// they pass, and nothing shows they would fail.
//
// `reject_all_probe` is not a mutant of the shape gate. It is the exact defect
// review RUN-260825-86b7d5 found - a predicate that refuses every member - and
// it is here so the harness's own gates can be shown to redden on it.
func TestP23F_TheHarnessCatchesABrokenMutant(t *testing.T) {
	_, prof := p21Project(t, "75", 4000)
	valid := validBytes(t, prof, p21Key(prof), 4242, "/tmp")
	corpus := buildCorpus(t, valid)

	const broken = "reject_all_probe"

	t.Run("rule_1_anti_reject_all", func(t *testing.T) {
		if accept, reason, _ := decoderVerdict(valid, broken); accept {
			t.Fatal("the broken probe accepted the plain valid frame; it is not reproducing the defect")
		} else {
			t.Logf("REDDENS: the plain valid frame is refused %s under %s - P23.B rule 1 and P23.E both fail on this", reason, broken)
		}
	})

	t.Run("rule_2_declared_direction", func(t *testing.T) {
		var admits, overRefuses int
		for _, c := range corpus {
			wantAccept, _, _ := oracleVerdict(c)
			gotAccept, _, _ := decoderVerdict(c.bytes, broken)
			switch {
			case gotAccept && !wantAccept:
				admits++
			case !gotAccept && wantAccept:
				overRefuses++
			}
		}
		if admits != 0 {
			t.Fatalf("the broken probe admits %d frames; a reject-all defect admits none", admits)
		}
		if overRefuses == 0 {
			t.Fatal("the broken probe over-refuses nothing")
		}
		// This is the arithmetic that made the revision-8 defect invisible: the
		// kill count is large and entirely on the wrong side of the verdict.
		t.Logf("REDDENS: %d admitting + %d over-refusing witnesses. Declared `admits`, rule 2 fails on the empty admitting set; declared `over_refuses`, rule 1 fails. Revision 8's kill condition - total > 0 - passed on exactly this shape.",
			admits, overRefuses)
	})

	t.Run("rule_3_measured_blindness", func(t *testing.T) {
		div := legacyDivergences(t, valid, broken)
		var caught []string
		for _, b := range allBaselines {
			if len(div[b]) > 0 {
				caught = append(caught, fmt.Sprintf("%s(%d rows)", b, len(div[b])))
			}
		}
		if len(caught) != len(allBaselines) {
			t.Fatalf("the broken probe is not caught by every baseline: %v", caught)
		}
		t.Logf("REDDENS: caught by %s, so any blindTo claim at all would be an OVER-CLAIM. The revision-8 table declared blind=true for the real defect and nothing executed the claim.",
			strings.Join(caught, " "))
	})

	t.Run("the_corrected_mutant_is_not_reject_all", func(t *testing.T) {
		// The other half: the repaired `unknown_by_wire_form` must accept the plain
		// frame and refuse the escaped one, so the row proves a wire-form gate
		// rather than a broken predicate.
		if accept, reason, field := decoderVerdict(valid, "unknown_by_wire_form"); !accept {
			t.Fatalf("the repaired mutant still refuses the plain valid frame: %s(%s)", reason, field)
		}
		var escaped []gm
		for _, m := range baseMembers(t, valid) {
			escaped = append(escaped, escapedMember(m.decoded, m.val))
		}
		frame := buildObject(escaped)
		if accept, _, _ := decoderVerdict(frame, ""); !accept {
			t.Fatal("the specified decoder refuses the all-escaped frame; the rule is byte equality on the DECODED name")
		}
		accept, reason, field := decoderVerdict(frame, "unknown_by_wire_form")
		if accept {
			t.Fatal("the repaired mutant admits the all-escaped frame; it is no longer deciding on the wire form")
		}
		t.Logf("DISCRIMINATES: plain valid -> accept, all-escaped valid -> refuse %s(%s). The mutant now refuses exactly one class instead of everything.", reason, field)
	})

	t.Run("the_repair_does_not_change_the_specified_decoder", func(t *testing.T) {
		// F1's repair is one line inside `p21CheckKeys`. Whether it moved the
		// SPECIFIED decoder's admitted set is a question of fact, and section 9
		// makes a protocol bump depend on the answer, so it is measured rather
		// than argued: `rev8_unknown_wiring` is revision 8's line verbatim, and it
		// must agree with the specified decoder on the whole corpus and on every
		// hand-written row - verdict, reason and named member.
		checked := 0
		disagree := func(what string, b []byte) {
			t.Helper()
			wa, wr, wf := decoderVerdict(b, "")
			ga, gr, gf := decoderVerdict(b, "rev8_unknown_wiring")
			if wa != ga || wr != gr || wf != gf {
				t.Fatalf("%s: revision 9 says %s, revision 8's wiring says %s - the repair moved the specified decoder and section 9 would require a protocol bump",
					what, verdictString(wa, wr, wf), verdictString(ga, gr, gf))
			}
			checked++
		}
		for _, c := range corpus {
			disagree(c.id, c.bytes)
		}
		for _, row := range legacyRows() {
			disagree(row.baseline+"/"+row.id, row.build(t, valid))
		}
		t.Logf("NO BEHAVIOUR CHANGE: revision 8's wiring and revision 9's agree on all %d frames (verdict, reason and named member). The repair is confined to mutant selection, so protocol_version stays 6 for a fourth revision.", checked)
	})

	t.Log("the harness's own gates are gates: this test is what shows they would fail")
}
