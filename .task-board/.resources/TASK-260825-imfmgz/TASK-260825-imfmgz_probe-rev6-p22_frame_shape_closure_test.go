package probe6

// P22 - the authorization frame's field set is closed at the DECODER.
//
// Review RUN-260825-a8a4ef finding F1, BLOCKING. Revision 5 declared the frame's
// field set closed at exactly five members and enforced it with five equality
// comparisons over a Go struct decoded by json.Unmarshal. Go discards unknown
// object members silently and resolves duplicate keys last-wins, so the reviewer
// drove the same launcher entry point with
//
//	(1) all five valid fields plus an unknown sixth `caller_chosen_field`
//	(2) `protocol_version: 999` followed by the valid `protocol_version: 5`
//
// and both reached execve. Every one of revision 5's five equality rows stayed
// green throughout: they were all true. The absent gate was the one no comparison
// can express - "these are the only bytes the frame carries".
//
// P22 is that gate and its evidence:
//
//	P22.A  Control. A valid five-key frame execve's, and the decoded key multiset
//	       the launcher recorded is exactly the five allowed keys once each. This
//	       is section 12.4's comparison-set obligation discharged on what the
//	       decoder SAW rather than on the struct members the implementation
//	       already knows about.
//	P22.B  An unknown sixth member refuses `protocol_violation`/
//	       `frame_unknown_field` naming the member, and never carries the target.
//	       Two mutants reach execve: `unknown_ignored` (the clause deleted) and
//	       `shape_gate_deleted` (revision 5's launcher verbatim). `dup_ignored`
//	       must still refuse it, so the clause is proved to be its own gate.
//	P22.C  A duplicate key refuses in BOTH orders. `dup_ignored` reaches execve on
//	       both, while `shape_gate_deleted` reaches execve on the last-wins order
//	       only - which is precisely why one order is not evidence.
//	P22.D  A missing member refuses naming it. This row deliberately does NOT
//	       discriminate the shape gate: the equality comparison catches a zero
//	       value too, so `shape_gate_deleted` still refuses. The test asserts that
//	       fact rather than hiding it, and asserts that the clause's own mutant
//	       changes the REASON without ever reaching execve.
//	P22.E  Content after the object refuses `frame_not_single_object`. Its mutant
//	       reaches execve; the row proves the clause is evaluated, and claims no
//	       more than that (see the comment on the row).
//	P22.F  The reviewer's two attack frames, rerun verbatim in shape against the
//	       corrected launcher: both refuse, the valid control still execve's, and
//	       both reproduce the original defeat under `shape_gate_deleted`.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const target = "/bin/sleep"

// rawFrame writes arbitrary bytes to the authorization pipe. Every attack below
// needs bytes a p21Frame value cannot express, which is the whole subject.
func (r *p21Run) rawFrame(t *testing.T, b []byte) {
	t.Helper()
	if _, err := r.w.Write(b); err != nil {
		t.Fatalf("write frame: %v", err)
	}
}

func validBytes(t *testing.T, prof p21Profile, key string, pid int, cwd string) []byte {
	t.Helper()
	b, err := json.Marshal(validFrame(prof, key, pid, cwd))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// withMember appends a member to an encoded object without going through the
// struct, so the launcher sees a key the struct has no field for.
func withMember(b []byte, member string) []byte {
	return append(append([]byte{}, b[:len(b)-1]...), []byte(","+member+"}")...)
}

// prepMember puts a member FIRST, before the valid ones.
func prepMember(b []byte, member string) []byte {
	return append([]byte("{"+member+","), b[1:]...)
}

// dropMember removes a member from an encoded object.
func dropMember(t *testing.T, b []byte, name string) []byte {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	delete(m, name)
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

type shapeCase struct {
	name   string
	frame  func(t *testing.T, base []byte) []byte
	reason string
	field  string
}

func p22Env(mutant, keylog string) []string {
	var env []string
	if mutant != "" {
		env = append(env, "P5_MUTANT="+mutant)
	}
	if keylog != "" {
		env = append(env, "P6_KEYLOG="+keylog)
	}
	return env
}

// runShape drives the real launcher entry point with `frame` and returns the
// refusal, or nil if the pid reached execve on the composed target.
func runShape(t *testing.T, dir, key, cwd string, prof p21Profile, mutant string, frame []byte) (*refusal, *p21Run) {
	t.Helper()
	run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env(mutant, "")})
	run.rawFrame(t, frame)
	if _, ok := run.becameTarget(target); ok {
		return nil, run
	}
	run.waitExit(t)
	ref := run.refusal(t)
	// "not running the runtime now" is not "never ran it": only the poll that
	// never observed the target can say that, and every refusing row below is
	// paired with it.
	if run.everCarried(target) {
		t.Fatalf("refused %+v but the pid carried %s at some point", ref, target)
	}
	return &ref, run
}

func TestP22_FrameShapeIsClosedAtTheDecoder(t *testing.T) {
	dir, prof := p21Project(t, "63", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)
	base := func(t *testing.T) []byte { return validBytes(t, prof, key, 0, cwd) }
	_ = base

	// ---- P22.A -------------------------------------------------------------
	t.Run("P22.A_control_execs_and_records_the_decoded_key_multiset", func(t *testing.T) {
		keylog := filepath.Join(shortTempDir(t), "keys.json")
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env("", keylog)})
		run.authorize(t, validFrame(prof, key, run.pid, cwd))
		id, ok := run.becameTarget(target)
		if !ok {
			t.Fatal("the all-valid control did not reach execve; every refusal row below would be satisfied by a launcher that refuses everything")
		}
		if got := strings.Join(id.Argv, " "); got != target+" 63" {
			t.Fatalf("argv %q", got)
		}
		b, err := os.ReadFile(keylog)
		if err != nil {
			t.Fatalf("no decoder evidence written: %v", err)
		}
		var ev shapeEvidence
		if err := json.Unmarshal(b, &ev); err != nil {
			t.Fatal(err)
		}
		got := append([]string{}, ev.Keys...)
		sort.Strings(got)
		want := p21ComparedFields()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("decoded key multiset %v, compared set %v: section 12.4 requires these to be the same set", ev.Keys, ev.Compared)
		}
		if strings.Join(ev.Compared, ",") != strings.Join(want, ",") {
			t.Fatalf("compared set %v != %v", ev.Compared, want)
		}
		t.Logf("PASS control: execve %v; decoded keys %v == compared fields %v", id.Argv, ev.Keys, ev.Compared)
	})

	// ---- P22.B / C / D / E: the refusal table ------------------------------
	cases := []shapeCase{
		{
			name:   "P22.B_unknown_sixth_member",
			frame:  func(t *testing.T, b []byte) []byte { return withMember(b, `"caller_chosen_field":"ignored"`) },
			reason: "frame_unknown_field",
			field:  "caller_chosen_field",
		},
		{
			name:   "P22.C_duplicate_wrong_then_correct",
			frame:  func(t *testing.T, b []byte) []byte { return prepMember(b, `"protocol_version":999`) },
			reason: "frame_duplicate_field",
			field:  "protocol_version",
		},
		{
			name:   "P22.C_duplicate_correct_then_wrong",
			frame:  func(t *testing.T, b []byte) []byte { return withMember(b, `"protocol_version":999`) },
			reason: "frame_duplicate_field",
			field:  "protocol_version",
		},
		{
			name:   "P22.D_missing_member",
			frame:  func(t *testing.T, b []byte) []byte { return dropMember(t, b, "runtime_key") },
			reason: "frame_missing_field",
			field:  "runtime_key",
		},
		{
			name: "P22.E_content_after_the_object",
			frame: func(t *testing.T, b []byte) []byte {
				return append(append([]byte{}, b...), []byte(`{"protocol_version":999}`)...)
			},
			reason: "frame_not_single_object",
			field:  "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
			frame := c.frame(t, validBytes(t, prof, key, run.pid, cwd))
			run.rawFrame(t, frame)
			if _, ok := run.becameTarget(target); ok {
				t.Fatalf("%s reached execve", c.name)
			}
			run.waitExit(t)
			ref := run.refusal(t)
			if ref.Code != "protocol_violation" || ref.Reason != c.reason {
				t.Fatalf("got %+v, want protocol_violation/%s", ref, c.reason)
			}
			if ref.Field != c.field {
				t.Fatalf("got field %q, want %q: a refusal that does not name the offending member cannot tell these rows apart", ref.Field, c.field)
			}
			if run.everCarried(target) {
				t.Fatalf("%s refused but the pid carried %s", c.name, target)
			}
			t.Logf("PASS %s: %+v, pid %d never carried %s", c.name, ref, run.pid, target)
		})
	}
}

// TestP22_Mutants is the discriminating half. Each clause of the shape gate has
// a mutant that deletes exactly that clause, and the table below records which
// rows each mutant reddens and which it leaves green. A mutant that reddened
// every row would prove only that "some gate exists here".
func TestP22_Mutants(t *testing.T) {
	dir, prof := p21Project(t, "64", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	frames := map[string]func(t *testing.T, b []byte) []byte{
		"unknown":         func(t *testing.T, b []byte) []byte { return withMember(b, `"caller_chosen_field":"ignored"`) },
		"dup_wrong_first": func(t *testing.T, b []byte) []byte { return prepMember(b, `"protocol_version":999`) },
		"dup_wrong_last":  func(t *testing.T, b []byte) []byte { return withMember(b, `"protocol_version":999`) },
		"missing":         func(t *testing.T, b []byte) []byte { return dropMember(t, b, "runtime_key") },
		"trailing": func(t *testing.T, b []byte) []byte {
			return append(append([]byte{}, b...), []byte(`{"protocol_version":999}`)...)
		},
	}

	// want: "exec" ⇒ the mutant admits the frame to execve (the clause is proved
	// load-bearing for that row); otherwise the expected code/reason pair.
	type expect struct{ code, reason string }
	table := []struct {
		mutant string
		want   map[string]expect
		note   string
	}{
		{
			mutant: "unknown_ignored",
			want: map[string]expect{
				"unknown":         {"exec", ""},
				"dup_wrong_first": {"protocol_violation", "frame_duplicate_field"},
				"trailing":        {"protocol_violation", "frame_not_single_object"},
			},
			note: "deleting the unknown-member clause admits the reviewer's first attack and leaves the other clauses standing",
		},
		{
			mutant: "dup_ignored",
			want: map[string]expect{
				// Go's json.Unmarshal keeps the LAST duplicate, so deleting the
				// clause admits the reviewer's exact frame...
				"dup_wrong_first": {"exec", ""},
				// ...and refuses the reverse order, because last-wins hands 999 to
				// the equality gate. A suite that tested only this order would show
				// a green "duplicate keys are refused" with the clause deleted.
				"dup_wrong_last": {"runtime_authorization_mismatch", "frame_field_differs"},
				"unknown":        {"protocol_violation", "frame_unknown_field"},
			},
			note: "deleting the duplicate clause admits the reviewer's second attack under a last-wins value decoder, while the unknown clause still refuses - so the two are separate gates",
		},
		{
			mutant: "dup_ignored_first_wins",
			want: map[string]expect{
				// The mirror image, and the reason both orders are mandatory: an
				// equally ordinary first-wins decoder admits precisely the order
				// last-wins refuses.
				"dup_wrong_last":  {"exec", ""},
				"dup_wrong_first": {"runtime_authorization_mismatch", "frame_field_differs"},
			},
			note: "a first-wins value decoder with the duplicate clause deleted admits the OTHER order: neither duplicate order is evidence by itself, and the shape gate is what makes the launcher independent of the dedup rule",
		},
		{
			mutant: "trailing_ignored",
			want: map[string]expect{
				"trailing": {"exec", ""},
				"unknown":  {"protocol_violation", "frame_unknown_field"},
			},
			note: "deleting the single-object clause admits bytes after the frame",
		},
		{
			mutant: "missing_ignored",
			// NOT "exec". The equality comparison catches the zero value, so this
			// clause changes what the refusal SAYS and never what it admits. The
			// row is recorded as such rather than dressed up as a bypass.
			want: map[string]expect{
				"missing": {"runtime_authorization_mismatch", "frame_field_differs"},
			},
			note: "deleting the missing-member clause never reaches execve: it degrades the refusal from frame_missing_field to a mismatch on a zero value",
		},
		{
			mutant: "shape_gate_deleted",
			want: map[string]expect{
				// revision 5's launcher verbatim: the reviewer's exact two defeats
				"unknown":         {"exec", ""},
				"dup_wrong_first": {"exec", ""},
				// last-wins keeps 999 here, so this order refused even in revision
				// 5. One duplicate order is not evidence; this is why.
				"dup_wrong_last": {"runtime_authorization_mismatch", "frame_field_differs"},
				// and the honest half: the missing row never discriminated the
				// shape gate at all.
				"missing": {"runtime_authorization_mismatch", "frame_field_differs"},
			},
			note: "revision 5's decoder verbatim, reproducing both reviewer defeats and showing exactly which rows do and do not discriminate it",
		},
	}

	for _, row := range table {
		row := row
		t.Run(row.mutant, func(t *testing.T) {
			for frameName, want := range row.want {
				frameName, want := frameName, want
				t.Run(frameName, func(t *testing.T) {
					run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env(row.mutant, "")})
					run.rawFrame(t, frames[frameName](t, validBytes(t, prof, key, run.pid, cwd)))
					id, exec := run.becameTarget(target)
					if want.code == "exec" {
						if !exec {
							run.waitExit(t)
							t.Fatalf("mutant %s did not admit %s (got %+v); the clause it deletes is not proved load-bearing", row.mutant, frameName, run.refusal(t))
						}
						t.Logf("REDDENS: mutant %s admitted %s to execve %v", row.mutant, frameName, id.Argv)
						return
					}
					if exec {
						t.Fatalf("mutant %s admitted %s, but this row is expected to keep refusing", row.mutant, frameName)
					}
					run.waitExit(t)
					ref := run.refusal(t)
					if ref.Code != want.code || ref.Reason != want.reason {
						t.Fatalf("mutant %s / %s: got %+v, want %s/%s", row.mutant, frameName, ref, want.code, want.reason)
					}
					if run.everCarried(target) {
						t.Fatalf("mutant %s / %s refused but carried %s", row.mutant, frameName, target)
					}
					t.Logf("GREEN: mutant %s / %s still refuses %+v", row.mutant, frameName, ref)
				})
			}
			t.Log(row.note)
		})
	}
}

// TestP22F_ReviewerRev5AttackRerun replays review RUN-260825-a8a4ef's two frames
// verbatim in shape against the corrected launcher. Both must now refuse, and the
// valid control must still reach execve - a launcher that refused everything
// would satisfy the first half alone.
func TestP22F_ReviewerRev5AttackRerun(t *testing.T) {
	dir, prof := p21Project(t, "65", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	t.Run("unknown_sixth_field_now_refuses", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		b := validBytes(t, prof, key, run.pid, cwd)
		run.rawFrame(t, append(append([]byte{}, b[:len(b)-1]...), []byte(`,"caller_chosen_field":"ignored"}`)...))
		if id, ok := run.becameTarget(target); ok {
			t.Fatalf("REGRESSION: the reviewer's first attack still reaches execve %v", id.Argv)
		}
		run.waitExit(t)
		ref := run.refusal(t)
		if ref.Code != "protocol_violation" || ref.Reason != "frame_unknown_field" || ref.Field != "caller_chosen_field" {
			t.Fatalf("got %+v", ref)
		}
		if run.everCarried(target) {
			t.Fatal("refused but carried the target")
		}
		t.Logf("CLOSED: %+v", ref)
	})

	t.Run("duplicate_wrong_then_correct_version_now_refuses", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		b := validBytes(t, prof, key, run.pid, cwd)
		run.rawFrame(t, append([]byte(`{"protocol_version":999,`), b[1:]...))
		if id, ok := run.becameTarget(target); ok {
			t.Fatalf("REGRESSION: the reviewer's second attack still reaches execve %v", id.Argv)
		}
		run.waitExit(t)
		ref := run.refusal(t)
		if ref.Code != "protocol_violation" || ref.Reason != "frame_duplicate_field" || ref.Field != "protocol_version" {
			t.Fatalf("got %+v", ref)
		}
		if run.everCarried(target) {
			t.Fatal("refused but carried the target")
		}
		t.Logf("CLOSED: %+v", ref)
	})

	t.Run("valid_control_still_execs", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		run.authorize(t, validFrame(prof, key, run.pid, cwd))
		id, ok := run.becameTarget(target)
		if !ok {
			run.waitExit(t)
			t.Fatalf("the control no longer execs: %+v", run.refusal(t))
		}
		t.Logf("CONTROL: execve %v", id.Argv)
	})
}
