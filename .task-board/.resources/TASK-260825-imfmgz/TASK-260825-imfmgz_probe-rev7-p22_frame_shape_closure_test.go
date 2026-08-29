package probe7

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
//
// Revision 7 adds P22.G and P22.H, and the reason is review RUN-260825-9d5cff.
// Everything above proves that each CLAUSE exists. None of it proved the CLASS
// each clause covers, and the reviewer built two narrowed decoders that keep
// every row above green while admitting frames the specification refuses:
//
//	P22.G  "exactly once" is proved for the whole closed set and independently of
//	       the values. Each of the five allowed members is duplicated with its OWN
//	       VALID value, one row each - the set is closed and finite, so this
//	       dimension is proved exhaustively rather than sampled. Two narrowed
//	       mutants live here: `dup_only_if_values_differ`, which is green on every
//	       revision-6 row and admits all five, and `dup_only_protocol_version`,
//	       which is green on the field revision 6 happened to duplicate and admits
//	       the other four.
//	P22.H  "outside the closed five" is a class, not the one name revision 6
//	       sampled. The name space is infinite, so it is covered by near-miss
//	       CLASSES, each with a narrowed mutant that only that class reddens:
//	       an unrelated second name (`unknown_only_caller_chosen_field`), a case
//	       variant of an allowed name (`unknown_case_folded`), and an allowed name
//	       used as a prefix (`unknown_prefix_allowed`). Together they pin the
//	       specified rule to byte equality against exactly five names.

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

// rawValueOf returns the raw encoded value of a member, so a duplicate can be
// appended carrying the member's OWN VALID value rather than a wrong one. That
// distinction is the whole of review RUN-260825-9d5cff's first finding: a gate
// that refuses a duplicate only when the values differ is green on every
// wrong-value row and admits this one.
func rawValueOf(t *testing.T, b []byte, name string) json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	v, ok := m[name]
	if !ok {
		t.Fatalf("valid frame has no member %q", name)
	}
	return v
}

// dupSameValue appends a second occurrence of an allowed member carrying the
// identical bytes the first one carries.
func dupSameValue(t *testing.T, b []byte, name string) []byte {
	t.Helper()
	return withMember(b, `"`+name+`":`+string(rawValueOf(t, b, name)))
}

// unknownNamed appends one unknown member with the given name and an
// uninteresting value.
func unknownNamed(name string) func(t *testing.T, b []byte) []byte {
	return func(t *testing.T, b []byte) []byte {
		return withMember(b, `"`+name+`":"ignored"`)
	}
}

// unknownCaseVariantOf appends an unknown member whose name is a case variant of
// `allowed`, carrying `allowed`'s OWN VALID value.
//
// The value is not decoration, and revision 7 measured why. Go's encoding/json
// matches an object key to a struct field by exact tag first and by CASE-FOLDED
// name second, so `"Schema"` is not discarded the way `"future_extension"` is -
// it is absorbed into the `schema` field and overwrites it. A case variant
// carrying a WRONG value is therefore caught by the equality comparison and
// proves nothing about the unknown-member clause; only a case variant carrying
// the RIGHT value reaches the comparisons with everything equal, so that the
// shape gate is the only thing left between it and `execve`.
//
// This is the same trap as revision 6's single duplicate order, one level down:
// an attack that a different gate happens to stop is not evidence for the gate
// under test.
func unknownCaseVariantOf(name, allowed string) func(t *testing.T, b []byte) []byte {
	return func(t *testing.T, b []byte) []byte {
		return withMember(b, `"`+name+`":`+string(rawValueOf(t, b, allowed)))
	}
}

func dupSame(name string) func(t *testing.T, b []byte) []byte {
	return func(t *testing.T, b []byte) []byte { return dupSameValue(t, b, name) }
}

// unknownClasses are the near-miss classes that pin "a member outside the closed
// five" to byte equality against exactly those five names. Each row after the
// first is paired with a narrowed mutant in TestP22_Mutants that ONLY that row
// reddens; `caller_chosen_field` carries no unique mutant of its own and is
// retained as the RUN-260825-a8a4ef regression anchor, which is stated here
// rather than left for a reader to discover.
var unknownClasses = []struct {
	key   string
	name  string
	frame func(t *testing.T, b []byte) []byte
	class string
}{
	{"unknown", "caller_chosen_field", unknownNamed("caller_chosen_field"),
		"the name review RUN-260825-a8a4ef used; regression anchor, no unique mutant"},
	{"unknown_future_extension", "future_extension", unknownNamed("future_extension"),
		"an unrelated second name: reddens unknown_only_caller_chosen_field"},
	{"unknown_case_variant", "Schema", unknownCaseVariantOf("Schema", "schema"),
		"a case variant of an allowed name carrying that name's valid value: reddens unknown_case_folded"},
	{"unknown_case_variant_wrong_value", "Schema", unknownNamed("Schema"),
		"the same case variant carrying a WRONG value: refuses under every narrowed unknown mutant, but as runtime_authorization_mismatch from the equality gate, so it discriminates NOTHING here and is recorded as such"},
	{"unknown_prefix_variant", "exec_plan_digest_v2", unknownNamed("exec_plan_digest_v2"),
		"an allowed name used as a prefix: reddens unknown_prefix_allowed"},
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
			name:   "P22.G_duplicate_same_value_schema",
			frame:  dupSame("schema"),
			reason: "frame_duplicate_field",
			field:  "schema",
		},
		{
			name:   "P22.G_duplicate_same_value_protocol_version",
			frame:  dupSame("protocol_version"),
			reason: "frame_duplicate_field",
			field:  "protocol_version",
		},
		{
			name:   "P22.G_duplicate_same_value_runtime_key",
			frame:  dupSame("runtime_key"),
			reason: "frame_duplicate_field",
			field:  "runtime_key",
		},
		{
			name:   "P22.G_duplicate_same_value_launcher_pid",
			frame:  dupSame("launcher_pid"),
			reason: "frame_duplicate_field",
			field:  "launcher_pid",
		},
		{
			name:   "P22.G_duplicate_same_value_exec_plan_digest",
			frame:  dupSame("exec_plan_digest"),
			reason: "frame_duplicate_field",
			field:  "exec_plan_digest",
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

	// P22.H - the unknown-member CLASS. Generated from unknownClasses so the row
	// set and the mutant table below cannot drift apart.
	for _, u := range unknownClasses {
		t.Logf("P22.H class %q (%s): %s", u.name, u.key, u.class)
		cases = append(cases, shapeCase{
			name:   "P22.H_unknown_member_" + u.key,
			frame:  u.frame,
			reason: "frame_unknown_field",
			field:  u.name,
		})
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
		// revision 7: duplicates carrying the member's OWN VALID value, one per
		// allowed member, and the unknown-name near-miss classes.
		"dup_same_schema":           dupSame("schema"),
		"dup_same_protocol_version": dupSame("protocol_version"),
		"dup_same_runtime_key":      dupSame("runtime_key"),
		"dup_same_launcher_pid":     dupSame("launcher_pid"),
		"dup_same_exec_plan_digest": dupSame("exec_plan_digest"),

		"trailing": func(t *testing.T, b []byte) []byte {
			return append(append([]byte{}, b...), []byte(`{"protocol_version":999}`)...)
		},
	}

	// The unknown-member frames come from the same table the refusal rows are
	// generated from, so a class cannot exist in one place and not the other.
	for _, u := range unknownClasses {
		frames[u.key] = u.frame
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
				"unknown":                  {"exec", ""},
				"unknown_future_extension": {"exec", ""},
				"unknown_case_variant":     {"exec", ""},
				"unknown_prefix_variant":   {"exec", ""},
				// NOT "exec": with a wrong value the case variant is absorbed into
				// `schema` by Go's case-folded field matching and the equality gate
				// refuses it, so it discriminates nothing about this clause.
				"unknown_case_variant_wrong_value": {"runtime_authorization_mismatch", "frame_field_differs"},
				"dup_wrong_first":                  {"protocol_violation", "frame_duplicate_field"},
				"trailing":                         {"protocol_violation", "frame_not_single_object"},
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
			// REVISION 7, review RUN-260825-9d5cff finding F1 (1). NARROWED, not
			// deleted: the clause is present and refuses every duplicate whose two
			// values differ - which is every duplicate revision 6 tested. What it
			// admits is "exactly once" itself.
			mutant: "dup_only_if_values_differ",
			want: map[string]expect{
				"dup_same_schema":           {"exec", ""},
				"dup_same_protocol_version": {"exec", ""},
				"dup_same_runtime_key":      {"exec", ""},
				"dup_same_launcher_pid":     {"exec", ""},
				"dup_same_exec_plan_digest": {"exec", ""},
				// both revision-6 duplicate rows stay GREEN, which is the finding:
				// the mutant survives the entire revision-6 table.
				"dup_wrong_first": {"protocol_violation", "frame_duplicate_field"},
				"dup_wrong_last":  {"protocol_violation", "frame_duplicate_field"},
				"unknown":         {"protocol_violation", "frame_unknown_field"},
			},
			note: "a value-sensitive duplicate gate keeps every revision-6 row green and admits all five allowed members repeated with their own valid value: the duplicate rule is about the SECOND OCCURRENCE, never about what it carries",
		},
		{
			// REVISION 7. The same defect one axis over, and the reason the
			// duplicate dimension is proved exhaustively rather than sampled: the
			// revision-6 rows all duplicated protocol_version, so a gate that
			// checks only that field survives them too.
			mutant: "dup_only_protocol_version",
			want: map[string]expect{
				"dup_same_schema":           {"exec", ""},
				"dup_same_runtime_key":      {"exec", ""},
				"dup_same_launcher_pid":     {"exec", ""},
				"dup_same_exec_plan_digest": {"exec", ""},
				// green on the sampled field, in both revision-6 orders and with
				// the same value.
				"dup_same_protocol_version": {"protocol_violation", "frame_duplicate_field"},
				"dup_wrong_first":           {"protocol_violation", "frame_duplicate_field"},
			},
			note: "a field-sampled duplicate gate is green on protocol_version and admits the other four: the allowed set is closed and finite, so every member gets its own row rather than a representative",
		},
		{
			// REVISION 7, review RUN-260825-9d5cff finding F1 (2). NARROWED: the
			// clause refuses exactly the one name revision 6 sampled.
			mutant: "unknown_only_caller_chosen_field",
			want: map[string]expect{
				"unknown":                  {"protocol_violation", "frame_unknown_field"},
				"unknown_future_extension": {"exec", ""},
				"unknown_case_variant":     {"exec", ""},
				"unknown_prefix_variant":   {"exec", ""},
				// NOT "exec", and kept visible for the same reason the
				// missing-member row is: a table that drops its non-discriminating
				// rows reads as broader coverage than it has. With a wrong value the
				// case variant is absorbed into `schema` by Go's case-folded field
				// matching, so the equality gate refuses it and the unknown clause is
				// never what stopped it.
				"unknown_case_variant_wrong_value": {"runtime_authorization_mismatch", "frame_field_differs"},
			},
			note: "refusing one sampled unknown name keeps revision 6's only unknown row green and admits every other unknown member: one name is not the complement of the allowlist",
		},
		{
			// REVISION 7. A case-insensitive allowlist is the narrowing an
			// implementation reaches for without noticing, and only a case-variant
			// row catches it.
			mutant: "unknown_case_folded",
			want: map[string]expect{
				"unknown_case_variant":     {"exec", ""},
				"unknown":                  {"protocol_violation", "frame_unknown_field"},
				"unknown_future_extension": {"protocol_violation", "frame_unknown_field"},
				"unknown_prefix_variant":   {"protocol_violation", "frame_unknown_field"},
				// admitted by the mutant's folded allowlist, then caught by the
				// equality gate on the value it overwrote. Recorded, not omitted.
				"unknown_case_variant_wrong_value": {"runtime_authorization_mismatch", "frame_field_differs"},
			},
			note: "a case-folded allowlist admits Schema and nothing else in the table: the row exists because it is the only one that reddens this mutant",
		},
		{
			// REVISION 7. Prefix matching instead of equality, caught only by a
			// name that extends an allowed one.
			mutant: "unknown_prefix_allowed",
			want: map[string]expect{
				"unknown_prefix_variant":   {"exec", ""},
				"unknown":                  {"protocol_violation", "frame_unknown_field"},
				"unknown_future_extension": {"protocol_violation", "frame_unknown_field"},
				"unknown_case_variant":     {"protocol_violation", "frame_unknown_field"},
				// prefix matching is case-sensitive, so both case variants stay
				// outside the mutant's allowlist.
				"unknown_case_variant_wrong_value": {"protocol_violation", "frame_unknown_field"},
			},
			note: "a prefix-tolerant allowlist admits exec_plan_digest_v2 and nothing else in the table: name comparison is byte equality against exactly five names",
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
				"unknown":                  {"exec", ""},
				"unknown_future_extension": {"exec", ""},
				// revision 5's decoder absorbs a case variant into the field it
				// folds onto and never sees a sixth member at all.
				"unknown_case_variant": {"exec", ""},
				"dup_wrong_first":      {"exec", ""},
				// revision 5's decoder admits the same-value duplicate too, so
				// revision 7's new row is a regression row for it as well.
				"dup_same_protocol_version": {"exec", ""},
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

// TestP22I_ReviewerRev6NarrowingAttackRerun replays review RUN-260825-9d5cff's
// two surviving narrowed gates against the real launcher entry point.
//
// The reviewer's finding was not that the revision-6 decoder admitted anything -
// it did not - but that nothing in the revision-6 proof would have caught an
// implementation that narrowed either clause. This test is that proof, stated as
// the reviewer stated the attack: the exact frame each narrowed gate admits must
// refuse against the specified decoder, must reach `execve` under the
// corresponding narrowed mutant, and the valid control must still exec so that
// none of it is satisfied by a launcher that refuses everything.
func TestP22I_ReviewerRev6NarrowingAttackRerun(t *testing.T) {
	dir, prof := p21Project(t, "66", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	attacks := []struct {
		name   string
		frame  func(t *testing.T, b []byte) []byte
		mutant string
		reason string
		field  string
	}{
		{
			name:   "same_value_duplicate_admitted_by_a_value_sensitive_gate",
			frame:  dupSame("protocol_version"),
			mutant: "dup_only_if_values_differ",
			reason: "frame_duplicate_field",
			field:  "protocol_version",
		},
		{
			name:   "second_unknown_name_admitted_by_a_name_sampled_gate",
			frame:  unknownNamed("future_extension"),
			mutant: "unknown_only_caller_chosen_field",
			reason: "frame_unknown_field",
			field:  "future_extension",
		},
	}

	for _, a := range attacks {
		a := a
		t.Run(a.name, func(t *testing.T) {
			t.Run("specified_decoder_refuses", func(t *testing.T) {
				run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
				run.rawFrame(t, a.frame(t, validBytes(t, prof, key, run.pid, cwd)))
				if id, ok := run.becameTarget(target); ok {
					t.Fatalf("REGRESSION: the reviewer's frame reached execve %v", id.Argv)
				}
				run.waitExit(t)
				ref := run.refusal(t)
				if ref.Code != "protocol_violation" || ref.Reason != a.reason || ref.Field != a.field {
					t.Fatalf("got %+v, want protocol_violation/%s naming %s", ref, a.reason, a.field)
				}
				if run.everCarried(target) {
					t.Fatal("refused but the pid carried the target")
				}
				t.Logf("CLOSED: %+v", ref)
			})
			t.Run("narrowed_gate_admits_it", func(t *testing.T) {
				run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env(a.mutant, "")})
				run.rawFrame(t, a.frame(t, validBytes(t, prof, key, run.pid, cwd)))
				id, ok := run.becameTarget(target)
				if !ok {
					run.waitExit(t)
					t.Fatalf("mutant %s did not admit the frame (%+v): the row does not prove the class", a.mutant, run.refusal(t))
				}
				t.Logf("REDDENS: mutant %s admitted the frame to execve %v", a.mutant, id.Argv)
			})
		})
	}

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
