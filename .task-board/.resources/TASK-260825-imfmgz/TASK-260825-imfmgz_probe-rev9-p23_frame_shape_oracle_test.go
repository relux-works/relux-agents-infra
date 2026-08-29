package probe7

// P23 - revision 8. The frame's shape gate is bound to a PROPERTY, not to a list
// of frames somebody thought of.
//
// Three reviews in a row found the same shape of defect, each one class further
// out, and the reason is structural rather than a series of oversights:
//
//	RUN-260825-a8a4ef  five equality comparisons closed nothing; the decoder had
//	                   already discarded the evidence. Revision 6 moved the gate
//	                   to the decoder.
//	RUN-260825-9d5cff  the decoder was right and the PROOF sampled it: one
//	                   duplicate value pair and one unknown name. Two narrowed
//	                   decoders kept every row green - "refuse a duplicate only
//	                   when the values differ", "refuse only that one name".
//	                   Revision 7 added exhaustive members and near-miss classes.
//	RUN-260825-c71188  still sampled, on two dimensions nobody had named: every
//	                   revision-7 duplicate row had arity exactly two, and every
//	                   revision-7 unknown name was at most 19 bytes. "Refuse only
//	                   a count of two" and "apply the allowlist only up to 32
//	                   bytes" keep the whole revision-7 table green and reach
//	                   execve.
//
// Adding two more rows would lose again, one dimension further out. The rows are
// not the problem: naming the admitted class in advance is. P23 stops doing that.
//
// THE ORACLE DIFFERENTIAL. The gate's decision is one predicate over one input:
// the decoded top-level member-name multiset M is accepted iff it equals the
// closed allowed set with multiplicity one. That predicate is decidable and
// tiny, so this test computes it INDEPENDENTLY - from the list of members the
// generator constructed, never by parsing the bytes and never by calling the
// decoder - and then requires the decoder to agree with it on every frame in a
// generated corpus. The expected verdict is never written down next to a frame.
//
// What that buys is the thing a row table cannot buy: a narrowed gate is caught
// because it DISAGREES WITH THE ORACLE, not because someone predicted the class
// it admits. The two RUN-260825-c71188 narrowings are killed here by generated
// frames, not by rows written to answer them.
//
// The corpus sweeps the decision's input structure rather than a list of
// attacks. Five dimensions, because the predicate has exactly five degrees of
// freedom:
//
//	name identity   every allowed name, every near-miss derivation of it, and
//	                random names over the printable byte space
//	occurrence      every allowed member at counts 0, 2, 3, 4, 5, 8 and 13, plus
//	                multi-member and random combinations
//	position        the offending member first, middle, last; a repeat adjacent
//	                to its original and separated from it
//	encoding        the same decoded name written plainly and as \u escapes, on
//	                both the accepted and the refused side
//	length          names from 0 to 1024 bytes, including every power-of-two
//	                boundary and each boundary +/- 1
//
// MUTANT KILL WITNESSES. The mutant table is no longer the coverage claim; it is
// the harness's own calibration. Every mutant must be killed BY THE CORPUS, and
// the test reports which generated frame killed it. A mutant that no generated
// frame kills fails this test as a coverage hole - reported by the harness
// rather than discovered by the next reviewer.
//
// REVISION 9, closing review RUN-260825-86b7d5 F1. Revision 8 stopped there, and
// stopping there is what let a BROKEN mutant be credited with a kill: a
// membership predicate that refused every member disagreed with the oracle on
// 48 frames, all of them ordinary valid ones, and `total > 0` accepted that as
// proof. "The mutant was killed" is a positive result about the harness, and
// this task's evidence rule applies to the harness's gates too. Three rules
// follow, all executed rather than asserted:
//
//	rule 1  every mutant must still ADMIT the plain valid frame (P23.B, and again
//	        at the production entry in P23.E)
//	rule 2  every mutant declares the SIDE it disagrees on, and the measured
//	        witness set must be entirely on that side (P23.B)
//	rule 3  `blindTo` is MEASURED against the hand-written row inventory, and an
//	        over-claim and an under-claim both fail (P23.D)
//
// P23.F reproduces the revision-8 defect as `reject_all_probe` and shows all
// three rules reddening on it. Measured by those rules: four mutants are blind
// to every hand-written baseline, ten to some baseline but not all, and four are
// caught by a row that already existed. Revision 8's "seven blind to everything"
// was wrong in both directions and is corrected in the table.
//
// WHAT THIS STILL DOES NOT PROVE, stated because the previous three revisions
// each claimed more than they had: a generated corpus is still a subset of an
// infinite input space. A narrowing whose admitted class is disjoint from every
// dimension swept here survives P23 exactly as `count == 2` survived revision 7.
// The corpus bounds the sample; it does not bound the space. What bounds the
// space is the STRUCTURAL obligation in spec section 12.4 - the production
// launcher must record the decoded multiset it decided on, and the decision must
// be an equality against a compiled constant set rather than a predicate over a
// count, a length, or any other quantity derived from the frame. P23 and that
// obligation are two halves of one claim and neither is evidence alone.

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

// ---- the generated corpus ---------------------------------------------------

// gm is one top-level member as the GENERATOR constructed it: the key literal it
// wrote into the bytes, the name a JSON decoder yields for that literal, and the
// raw value. `decoded` is ground truth by construction - it is never obtained by
// parsing, so the oracle below shares no code with the decoder under test.
type gm struct {
	wire    string
	decoded string
	val     string
}

func quoted(name string) string {
	b, err := json.Marshal(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func member(name, val string) gm { return gm{wire: quoted(name), decoded: name, val: val} }

// escapedMember writes the SAME decoded name using a \u escape for its first
// rune. The decoded name is unchanged, so the oracle treats it as that member -
// which is the specified rule, and which `unknown_by_wire_form` gets wrong in
// the admitting-nothing direction and `dup_keyed_on_wire_form` gets wrong in the
// admitting-something direction.
func escapedMember(name, val string) gm {
	r := []rune(name)
	wire := fmt.Sprintf(`"\u%04x%s"`, r[0], string(r[1:]))
	return gm{wire: wire, decoded: name, val: val}
}

func buildObject(ms []gm) []byte {
	var b strings.Builder
	b.WriteByte('{')
	for i, m := range ms {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(m.wire)
		b.WriteByte(':')
		b.WriteString(m.val)
	}
	b.WriteByte('}')
	return []byte(b.String())
}

// corpusFrame carries the bytes and the ground truth the generator used to build
// them. `expect*` is NOT written by hand: it is derived by oracleVerdict from
// `members`, or declared by the structural generator for inputs that are not one
// JSON object at all.
type corpusFrame struct {
	id      string
	dim     string
	bytes   []byte
	members []gm
	// structural inputs are not built from a member list; they declare their own
	// verdict because "these bytes are not one object" is a fact about the bytes.
	structural   bool
	structAccept bool
	structReason string
}

// oracleVerdict is the independent decision procedure. It reads the generator's
// own member list - never the bytes, never the decoder - and applies the rule
// exactly as spec section 6.2 B12 step 4 states it: the multiset must equal the
// closed five with multiplicity one, and the reported reason is the FIRST
// difference in the specified order (unknown in stream order, then duplicate in
// stream order, then missing in field order).
func oracleVerdict(c corpusFrame) (accept bool, reason, field string) {
	if c.structural {
		return c.structAccept, c.structReason, ""
	}
	allowed := map[string]bool{}
	for _, f := range frameFields {
		allowed[f] = true
	}
	for _, m := range c.members {
		if !allowed[m.decoded] {
			return false, "frame_unknown_field", m.decoded
		}
	}
	seen := map[string]bool{}
	for _, m := range c.members {
		if seen[m.decoded] {
			return false, "frame_duplicate_field", m.decoded
		}
		seen[m.decoded] = true
	}
	for _, f := range frameFields {
		if !seen[f] {
			return false, "frame_missing_field", f
		}
	}
	return true, "", ""
}

// baseMembers is the accepted frame, as a member list.
func baseMembers(t *testing.T, valid []byte) []gm {
	t.Helper()
	out := make([]gm, 0, len(frameFields))
	for _, f := range frameFields {
		out = append(out, member(f, string(rawValueOf(t, valid, f))))
	}
	return out
}

func valOf(ms []gm, name string) string {
	for _, m := range ms {
		if m.decoded == name {
			return m.val
		}
	}
	return `"ignored"`
}

func withoutMember(ms []gm, name string) []gm {
	out := make([]gm, 0, len(ms))
	for _, m := range ms {
		if m.decoded != name {
			out = append(out, m)
		}
	}
	return out
}

func frameOf(id, dim string, ms []gm) corpusFrame {
	return corpusFrame{id: id, dim: dim, bytes: buildObject(ms), members: ms}
}

// nameOfLength builds a deterministic unknown name of exactly n bytes that can
// never collide with an allowed one.
func nameOfLength(n int) string {
	if n == 0 {
		return ""
	}
	const alphabet = "qwxyz0123456789"
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(alphabet[i%len(alphabet)])
	}
	return b.String()
}

// buildCorpus generates every frame P23 decides on. Nothing here records an
// expected outcome: the oracle derives it.
func buildCorpus(t *testing.T, valid []byte) []corpusFrame {
	t.Helper()
	base := baseMembers(t, valid)
	var corpus []corpusFrame

	add := func(id, dim string, ms []gm) {
		corpus = append(corpus, frameOf(id, dim, ms))
	}

	// -- dimension: occurrence ------------------------------------------------
	// Every allowed member at every count on a ladder that runs well past any
	// arity a hand-written gate would special-case. Count 1 is the accepted
	// shape and is generated too, so the corpus is not a refuse-everything set.
	for _, f := range frameFields {
		for _, k := range []int{0, 1, 2, 3, 4, 5, 8, 13} {
			v := valOf(base, f)
			// Rebuild in the base order with the member repeated k times in place.
			var out []gm
			for _, m := range base {
				if m.decoded != f {
					out = append(out, m)
					continue
				}
				for i := 0; i < k; i++ {
					out = append(out, member(f, v))
				}
			}
			add(fmt.Sprintf("arity/%s/x%d", f, k), "occurrence", out)
		}
	}
	// Several members repeated at once, and the whole frame repeated.
	{
		var out []gm
		for _, m := range base {
			out = append(out, m)
			if m.decoded == "schema" || m.decoded == "runtime_key" {
				out = append(out, m, m)
			}
		}
		add("arity/two_members_x3", "occurrence", out)
	}
	for _, k := range []int{2, 3, 7} {
		var out []gm
		for i := 0; i < k; i++ {
			out = append(out, base...)
		}
		add(fmt.Sprintf("arity/whole_frame_x%d", k), "occurrence", out)
	}

	// -- dimension: position --------------------------------------------------
	// A repeat adjacent to its original, and one separated from it, for every
	// allowed member; plus an unknown member first, middle and last. Every
	// duplicate any review or revision ever minted was separated.
	for _, f := range frameFields {
		v := valOf(base, f)
		var adj []gm
		for _, m := range base {
			adj = append(adj, m)
			if m.decoded == f {
				adj = append(adj, member(f, v))
			}
		}
		add("position/adjacent_repeat/"+f, "position", adj)
		add("position/separated_repeat/"+f, "position", append(append([]gm{}, base...), member(f, v)))
	}
	{
		u := member("caller_chosen_field", `"ignored"`)
		add("position/unknown_first", "position", append([]gm{u}, base...))
		mid := append([]gm{}, base[:2]...)
		mid = append(mid, u)
		mid = append(mid, base[2:]...)
		add("position/unknown_middle", "position", mid)
		add("position/unknown_last", "position", append(append([]gm{}, base...), u))
	}

	// -- dimension: length ----------------------------------------------------
	// Unknown names across the whole plausible range, with every power-of-two
	// boundary and each boundary +/- 1, because "apply the allowlist up to N"
	// is the narrowing RUN-260825-c71188 found and N is not knowable in advance.
	for _, n := range []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65,
		127, 128, 129, 255, 256, 257, 511, 512, 1023, 1024} {
		add(fmt.Sprintf("length/unknown_%dB", n), "length",
			append(append([]gm{}, base...), member(nameOfLength(n), `"ignored"`)))
	}

	// -- dimension: name identity ---------------------------------------------
	// Near-miss derivations of every allowed name, each carrying THAT MEMBER'S
	// OWN VALID VALUE so the equality comparison can never be what stops it.
	// (Revision 7's measured lesson: an attack a different gate happens to stop
	// is not evidence for the gate under test.)
	derivations := []struct {
		key string
		fn  func(string) string
	}{
		{"upper", strings.ToUpper},
		{"title", func(s string) string { return strings.ToUpper(s[:1]) + s[1:] }},
		{"suffix_v2", func(s string) string { return s + "_v2" }},
		{"suffix_digit", func(s string) string { return s + "2" }},
		{"prefixed", func(s string) string { return "x" + s }},
		{"truncated", func(s string) string { return s[:len(s)-1] }},
		{"trailing_space", func(s string) string { return s + " " }},
		{"leading_space", func(s string) string { return " " + s }},
		{"trailing_nul", func(s string) string { return s + "\x00" }},
		{"dashed", func(s string) string { return strings.ReplaceAll(s, "_", "-") }},
		{"doubled", func(s string) string { return s + s }},
	}
	for _, f := range frameFields {
		v := valOf(base, f)
		for _, d := range derivations {
			name := d.fn(f)
			if name == f {
				continue
			}
			add("identity/"+d.key+"/"+f, "identity",
				append(append([]gm{}, base...), member(name, v)))
		}
		// A homoglyph: an allowed name with one ASCII letter replaced by the
		// Cyrillic character that renders identically. Every name any review or
		// revision has minted is pure ASCII.
		if i := strings.IndexByte(f, 'a'); i >= 0 {
			name := f[:i] + "а" + f[i+1:]
			add("identity/homoglyph/"+f, "identity",
				append(append([]gm{}, base...), member(name, v)))
		}
	}

	// -- dimension: encoding --------------------------------------------------
	// The same decoded name written plainly and as a \u escape, on BOTH sides of
	// the verdict. The accepted side matters as much as the refused side: a gate
	// that decides on the wire form refuses a frame it must admit, and no
	// refuse-only corpus can see that.
	for _, f := range frameFields {
		v := valOf(base, f)
		var esc []gm
		for _, m := range base {
			if m.decoded == f {
				esc = append(esc, escapedMember(f, v))
				continue
			}
			esc = append(esc, m)
		}
		add("encoding/escaped_"+f, "encoding", esc)
		// plain + escaped forms of the same member: one decoded name, twice.
		add("encoding/escaped_repeat/"+f, "encoding",
			append(append([]gm{}, base...), escapedMember(f, v)))
	}
	{
		var all []gm
		for _, m := range base {
			all = append(all, escapedMember(m.decoded, m.val))
		}
		add("encoding/all_escaped", "encoding", all)
	}
	// An unknown name written escaped, so the refused side covers encoding too.
	add("encoding/escaped_unknown", "encoding",
		append(append([]gm{}, base...), escapedMember("caller_chosen_field", `"ignored"`)))

	// -- dimension: order -----------------------------------------------------
	// The five members once each in several orders: all must be accepted.
	perms := [][]int{
		{0, 1, 2, 3, 4}, {4, 3, 2, 1, 0}, {2, 0, 4, 1, 3}, {1, 3, 0, 4, 2},
		{3, 4, 1, 2, 0}, {0, 4, 3, 1, 2}, {2, 3, 4, 0, 1}, {4, 0, 1, 3, 2},
	}
	for i, p := range perms {
		var out []gm
		for _, idx := range p {
			out = append(out, base[idx])
		}
		add(fmt.Sprintf("order/perm_%d", i), "order", out)
	}

	// -- dimension: random ----------------------------------------------------
	// Seeded so the corpus is identical on every run. Random names over the
	// printable byte space, and random combinations of every violation kind -
	// including combinations that violate NOTHING, which the oracle then accepts.
	rng := rand.New(rand.NewSource(8))
	const printable = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !#$%&()*+,-./:;<=>?@[]^_`{|}~"
	randName := func(n int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteByte(printable[rng.Intn(len(printable))])
		}
		return b.String()
	}
	for i := 0; i < 128; i++ {
		name := randName(1 + rng.Intn(64))
		if isAllowedName(name, "") {
			continue
		}
		add(fmt.Sprintf("random/unknown_%d", i), "random",
			append(append([]gm{}, base...), member(name, `"ignored"`)))
	}
	for i := 0; i < 96; i++ {
		out := append([]gm{}, base...)
		switch rng.Intn(4) {
		case 0: // drop a member
			out = withoutMember(out, frameFields[rng.Intn(len(frameFields))])
		case 1: // repeat a member k times
			f := frameFields[rng.Intn(len(frameFields))]
			for k := 1 + rng.Intn(5); k > 0; k-- {
				out = append(out, member(f, valOf(base, f)))
			}
		case 2: // add unknown members
			for j := 1 + rng.Intn(3); j > 0; j-- {
				n := randName(1 + rng.Intn(48))
				if isAllowedName(n, "") {
					continue
				}
				out = append(out, member(n, `"ignored"`))
			}
		case 3: // shuffle only - the oracle must ACCEPT this one
		}
		rng.Shuffle(len(out), func(a, b int) { out[a], out[b] = out[b], out[a] })
		add(fmt.Sprintf("random/mixed_%d", i), "random", out)
	}

	// -- dimension: structural ------------------------------------------------
	// Inputs that are not one JSON object. These declare their verdict because
	// it is a fact about the bytes rather than about a member multiset.
	valObj := string(buildObject(base))
	structural := []struct {
		id     string
		bytes  string
		accept bool
		reason string
	}{
		{"array", `[1,2,3]`, false, "frame_not_single_object"},
		{"string", `"schema"`, false, "frame_not_single_object"},
		{"number", `42`, false, "frame_not_single_object"},
		{"null", `null`, false, "frame_not_single_object"},
		{"bare_true", `true`, false, "frame_not_single_object"},
		{"empty_input", ``, false, "frame_unparseable"},
		{"empty_object", `{}`, false, "frame_missing_field"},
		{"truncated", valObj[:len(valObj)-1], false, "frame_unparseable"},
		{"two_objects", valObj + valObj, false, "frame_not_single_object"},
		{"object_then_garbage", valObj + `xyz`, false, "frame_not_single_object"},
		{"object_then_number", valObj + `7`, false, "frame_not_single_object"},
		{"trailing_whitespace", valObj + "  \n\t", true, ""},
		{"leading_whitespace", "  \n\t" + valObj, true, ""},
		{"pretty_printed", prettyOf(base), true, ""},
	}
	for _, s := range structural {
		// empty_object is decided by the member rule, not by the byte rule, so it
		// goes through the member path with an empty list.
		if s.id == "empty_object" {
			add("structural/empty_object", "structural", nil)
			continue
		}
		if s.id == "trailing_whitespace" || s.id == "leading_whitespace" || s.id == "pretty_printed" {
			corpus = append(corpus, corpusFrame{
				id: "structural/" + s.id, dim: "structural", bytes: []byte(s.bytes),
				structural: true, structAccept: true,
			})
			continue
		}
		corpus = append(corpus, corpusFrame{
			id: "structural/" + s.id, dim: "structural", bytes: []byte(s.bytes),
			structural: true, structAccept: false, structReason: s.reason,
		})
	}
	return corpus
}

func prettyOf(ms []gm) string {
	var b strings.Builder
	b.WriteString("{\n")
	for i, m := range ms {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("  ")
		b.WriteString(m.wire)
		b.WriteString(" : ")
		b.WriteString(m.val)
	}
	b.WriteString("\n}\n")
	return b.String()
}

// decoderVerdict runs the decoder under test and reports its shape decision.
func decoderVerdict(b []byte, mutant string) (accept bool, reason, field string) {
	_, fault := p21DecodeFrame(b, mutant)
	if fault == nil {
		return true, "", ""
	}
	return false, fault.reason, fault.field
}

// ---- P23.A: the specified decoder agrees with the oracle everywhere ---------

func TestP23A_SpecifiedDecoderAgreesWithTheOracle(t *testing.T) {
	_, prof := p21Project(t, "70", 4000)
	valid := validBytes(t, prof, p21Key(prof), 4242, "/tmp")
	corpus := buildCorpus(t, valid)

	dims := map[string]int{}
	accepted, refused := 0, 0
	for _, c := range corpus {
		wantAccept, wantReason, wantField := oracleVerdict(c)
		dims[c.dim]++
		if wantAccept {
			accepted++
		} else {
			refused++
		}
		gotAccept, gotReason, gotField := decoderVerdict(c.bytes, "")
		if gotAccept != wantAccept {
			t.Fatalf("%s: decoder accept=%v, oracle accept=%v; bytes=%s",
				c.id, gotAccept, wantAccept, truncate(c.bytes))
		}
		if wantAccept {
			continue
		}
		if gotReason != wantReason {
			t.Fatalf("%s: decoder reason %q, oracle reason %q; bytes=%s",
				c.id, gotReason, wantReason, truncate(c.bytes))
		}
		// The structural rows carry no member to name.
		if !c.structural && gotField != wantField {
			t.Fatalf("%s: decoder named %q, oracle named %q", c.id, gotField, wantField)
		}
	}
	if accepted == 0 || refused == 0 {
		t.Fatalf("corpus is one-sided: %d accepted, %d refused - a gate proved only by refusals is satisfied by one that refuses everything", accepted, refused)
	}
	keys := make([]string, 0, len(dims))
	for k := range dims {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, dims[k]))
	}
	t.Logf("PASS P23.A: %d generated frames, %d accepted / %d refused, dimensions %s",
		len(corpus), accepted, refused, strings.Join(parts, " "))
	t.Logf("the specified decoder agrees with the independent oracle on accept/refuse, on the refusal reason, and on the named member, for every frame")
}

func truncate(b []byte) string {
	if len(b) <= 220 {
		return string(b)
	}
	return string(b[:200]) + fmt.Sprintf("...(%d bytes)", len(b))
}

// ---- P23.B: every mutant is killed BY THE CORPUS ----------------------------

// shapeMutant records what each mutant is and where it came from.
//
// Revision 9 replaces revision 8's `blind bool` with two fields that are
// CHECKED rather than asserted, because review RUN-260825-86b7d5 found a mutant
// whose table row said one thing and whose code did another:
//
//	blindTo    the hand-written baselines this mutant is green on. P23.D measures
//	           the truth against the row inventory in
//	           p23d_blindness_calibration_test.go and fails on an over-claim AND
//	           on an under-claim. Revision 8 did both: it called the wire-form
//	           mutant blind when it was a reject-all fake, and it called the two
//	           RUN-260825-c71188 mutants blind to the c71188 rows when those rows
//	           are exactly what catches them.
//	direction  how this mutant disagrees with the oracle on the corpus: it either
//	           ADMITS frames the specification refuses, or it OVER-REFUSES frames
//	           the specification admits. P23.B requires the measured witness set
//	           to land on the declared side and to be empty on the other. A
//	           reject-all fake declared `admits` fails that; declared
//	           `over_refuses` it is caught by the plain-valid control instead.
type shapeMutant struct {
	name      string
	kind      string // deletion | narrowing
	origin    string
	direction string // admits | over_refuses
	blindTo   []string
	note      string
}

const (
	dirAdmits      = "admits"
	dirOverRefuses = "over_refuses"
)

// The `blindTo` values below are MEASURED, not chosen: P23.D recomputes each one
// from the row inventory and fails if the table disagrees. Revision 9's measured
// correction to revision 8's prose is visible in the two RUN-260825-c71188 rows,
// which are blind to rev6 and rev7 and NOT to the baseline they were minted for.
var shapeMutants = []shapeMutant{
	{"unknown_ignored", "deletion", "revision 6", dirAdmits, nil,
		"the unknown-member clause removed"},
	{"dup_ignored", "deletion", "revision 6", dirAdmits, nil,
		"the duplicate clause removed, last-wins value decode"},
	{"dup_ignored_first_wins", "deletion", "revision 6", dirAdmits, nil,
		"the duplicate clause removed, first-wins value decode"},
	{"trailing_ignored", "deletion", "revision 6", dirAdmits, []string{baselineRev7, baselineC7118},
		"the single-object clause removed"},
	{"missing_ignored", "deletion", "revision 6", dirAdmits, []string{baselineRev7, baselineC7118},
		"the absent-member clause removed"},
	{"shape_gate_deleted", "deletion", "revision 5 verbatim", dirAdmits, nil,
		"the whole gate removed - json.Unmarshal into five fields"},
	{"dup_only_if_values_differ", "narrowing", "review RUN-260825-9d5cff", dirAdmits, []string{baselineRev6},
		"a repeat is refused only when the two values differ"},
	{"dup_only_protocol_version", "narrowing", "revision 7", dirAdmits, []string{baselineRev6, baselineC7118},
		"the duplicate rule applied to one sampled member"},
	{"unknown_only_caller_chosen_field", "narrowing", "review RUN-260825-9d5cff", dirAdmits, []string{baselineRev6},
		"one sampled name refused, every other admitted"},
	{"unknown_case_folded", "narrowing", "revision 7", dirAdmits, []string{baselineRev6, baselineC7118},
		"the allowlist compared case-insensitively"},
	{"unknown_prefix_allowed", "narrowing", "revision 7", dirAdmits, []string{baselineRev6, baselineC7118},
		"the allowlist matched by prefix"},
	{"dup_only_exactly_two_total", "narrowing", "review RUN-260825-c71188", dirAdmits, []string{baselineRev6, baselineRev7},
		"a repeat is refused only at a total count of exactly two"},
	{"unknown_allow_over_32", "narrowing", "review RUN-260825-c71188", dirAdmits, []string{baselineRev6, baselineRev7},
		"the allowlist applied only to names of at most 32 bytes"},
	// MEASURED CORRECTION to revision 8, found by P23.D on its first run and not
	// by reasoning: revision 8 called this mutant blind to every hand-written
	// row. It is not. `dup_same_exec_plan_digest` appends the repeat at the end
	// of the object, and `exec_plan_digest` is the LAST member in the struct's
	// marshal order, so that one row - alone among the five same-value duplicate
	// rows - puts the repeat ADJACENT to its original and catches this mutant.
	// The blindness is to rev6 and to the c71188 rows only.
	{"dup_only_when_separated", "narrowing", "revision 8", dirAdmits, []string{baselineRev6, baselineC7118},
		"a repeat is refused only when it is not adjacent to its original"},
	{"unknown_ascii_only", "narrowing", "revision 8", dirAdmits, allBaselines,
		"the allowlist applied only to pure-ASCII names"},
	{"unknown_nonempty_only", "narrowing", "revision 8", dirAdmits, allBaselines,
		"the allowlist skipped for a zero-length name"},
	{"dup_keyed_on_wire_form", "narrowing", "revision 8", dirAdmits, allBaselines,
		"repeats identified by the bytes of the key rather than by the decoded name"},
	{"unknown_by_wire_form", "narrowing", "revision 8", dirOverRefuses, allBaselines,
		"membership decided on the wire form - refuses a VALID frame whose key is escaped. The ONLY over-refusing mutant in the table, and the one review RUN-260825-86b7d5 found miswired into a reject-all fake"},
}

func TestP23B_EveryMutantIsKilledByTheCorpus(t *testing.T) {
	_, prof := p21Project(t, "71", 4000)
	valid := validBytes(t, prof, p21Key(prof), 4242, "/tmp")
	corpus := buildCorpus(t, valid)

	blindKilled := 0
	for _, mu := range shapeMutants {
		mu := mu
		t.Run(mu.name, func(t *testing.T) {
			type witness struct {
				id, dim, detail string
			}
			var admits, overRefuses []witness
			for _, c := range corpus {
				wantAccept, wantReason, _ := oracleVerdict(c)
				gotAccept, gotReason, _ := decoderVerdict(c.bytes, mu.name)
				switch {
				case gotAccept && !wantAccept:
					admits = append(admits, witness{c.id, c.dim,
						fmt.Sprintf("oracle refuses %s, mutant accepts", wantReason)})
				case !gotAccept && wantAccept:
					overRefuses = append(overRefuses, witness{c.id, c.dim,
						fmt.Sprintf("oracle accepts, mutant refuses %s", gotReason)})
				}
			}
			total := len(admits) + len(overRefuses)
			if total == 0 {
				t.Fatalf("COVERAGE HOLE: mutant %s (%s, %s) disagrees with the oracle on no generated frame. The corpus does not bind the class this mutant narrows, and a reviewer will find it next.", mu.name, mu.kind, mu.origin)
			}

			// RULE 1, revision 9, closing review RUN-260825-86b7d5 F1. Every mutant
			// in this table narrows or deletes a REFUSAL clause, and no such change
			// can turn away a frame the specification admits - except the one whose
			// whole point is to decide membership on the wrong thing, and even that
			// one admits the PLAIN valid frame. A mutant that refuses it is not a
			// narrowing at all; it is broken, and every kill credited to it is
			// satisfied by a launcher that refuses everything.
			//
			// Revision 8's kill condition was `total > 0`, which this shape passes
			// with a large witness count entirely on the wrong side of the verdict.
			if accept, reason, field := decoderVerdict(valid, mu.name); !accept {
				t.Fatalf("REJECT-ALL MUTANT: %s (%s, %s) refuses the plain valid frame - %s(%s). It is not the mutant its row describes, and its %d witnesses prove nothing about the class it claims to narrow.",
					mu.name, mu.kind, mu.origin, reason, field, total)
			}

			// RULE 2: the mutant disagrees on the side its row declares, and does
			// not disagree on the other. Direction is not decoration - an admitting
			// narrowing and an over-refusing one are opposite defects, and a table
			// that does not distinguish them cannot notice when a mutant turns into
			// the other kind.
			switch mu.direction {
			case dirAdmits:
				if len(admits) == 0 {
					t.Fatalf("mutant %s declares direction %q but admits nothing the oracle refuses; its %d witnesses are all over-refusals, which is the opposite defect", mu.name, mu.direction, len(overRefuses))
				}
				if len(overRefuses) != 0 {
					t.Fatalf("mutant %s declares direction %q but ALSO over-refuses %d frames the oracle admits (first: %s). A mutant that fails in both directions cannot show which class it binds.", mu.name, mu.direction, len(overRefuses), overRefuses[0].id)
				}
			case dirOverRefuses:
				if len(overRefuses) == 0 {
					t.Fatalf("mutant %s declares direction %q but over-refuses nothing", mu.name, mu.direction)
				}
				if len(admits) != 0 {
					t.Fatalf("mutant %s declares direction %q but admits %d frames the oracle refuses (first: %s)", mu.name, mu.direction, len(admits), admits[0].id)
				}
			default:
				t.Fatalf("mutant %s declares no direction", mu.name)
			}
			show := func(label string, ws []witness) {
				for i, w := range ws {
					if i == 3 {
						t.Logf("  ... and %d more %s witnesses", len(ws)-3, label)
						break
					}
					t.Logf("  %s witness: %s [%s] - %s", label, w.id, w.dim, w.detail)
				}
			}
			t.Logf("KILLED %s (%s, %s, direction %s): %d admitting + %d over-refusing witnesses", mu.name, mu.kind, mu.origin, mu.direction, len(admits), len(overRefuses))
			show("admits", admits)
			show("over-refuses", overRefuses)
			if len(mu.blindTo) == len(allBaselines) {
				blindKilled++
				t.Logf("  BLIND to every hand-written baseline (%s); the generated corpus kills it. P23.D is what MEASURES that claim - this line only reports it.", strings.Join(mu.blindTo, ", "))
			} else if len(mu.blindTo) > 0 {
				t.Logf("  blind to %s only; some other hand-written baseline catches it (P23.D names the row)", strings.Join(mu.blindTo, ", "))
			}
			t.Log("  " + mu.note)
		})
	}
	if blindKilled == 0 {
		t.Fatal("no fully blind mutant was exercised: the calibration does not show that the corpus generalizes beyond the rows someone wrote")
	}
	t.Logf("PASS P23.B: %d mutants, all killed by generated frames, all admitting the plain valid frame, all disagreeing on their declared side only; %d blind to every hand-written baseline", len(shapeMutants), blindKilled)
}

// ---- P23.C: the reviewer's two exact bypasses, at the production entry ------

// TestP23C_ReviewRev7BypassesRefuseAtTheProductionEntry drives the real
// `runtime-launch` entry point. P23.A and P23.B decide at the decoder; this is
// what binds that decoder to the shipped path, and it carries the frames review
// RUN-260825-c71188 minted plus one frame per revision-8 blind mutant.
//
// Every row is checked twice and in both directions: the specified launcher must
// refuse it without ever carrying the target, and the narrowed mutant that
// admits it must reach execve - so no row is satisfied by a launcher that
// refuses everything. Each row additionally asserts that the decoder verdict
// from P23.A predicts the production outcome, which is what makes the two layers
// one gate rather than two coincidences.
//
// REVISION 9. The last row below - `all_names_escaped_is_a_VALID_frame` - is the
// only INVERTED one: the mutant must OVER-REFUSE a frame the specification
// admits. Review RUN-260825-86b7d5 showed that half a row is not a row. A mutant
// that refuses everything satisfies it too, and revision 8's mutant did exactly
// that. The missing half - plain valid frame -> execve UNDER THE SAME MUTANT -
// is P23.E, which runs it for all eighteen mutants rather than for this one,
// because the mutant that failed it was not one anybody would have singled out.
func TestP23C_ReviewRev7BypassesRefuseAtTheProductionEntry(t *testing.T) {
	dir, prof := p21Project(t, "72", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	rows := []struct {
		name   string
		mutant string
		reason string
		field  string
		invert bool // the mutant OVER-refuses a valid frame instead of admitting a bad one
		build  func(base []gm) []gm
		origin string
	}{
		{
			name: "arity_three_protocol_version", mutant: "dup_only_exactly_two_total",
			reason: "frame_duplicate_field", field: "protocol_version",
			origin: "review RUN-260825-c71188, the exact bypass",
			build: func(b []gm) []gm {
				v := valOf(b, "protocol_version")
				return append(append([]gm{}, b...), member("protocol_version", v), member("protocol_version", v))
			},
		},
		{
			name: "arity_five_schema", mutant: "dup_only_exactly_two_total",
			reason: "frame_duplicate_field", field: "schema",
			origin: "the same class, a different member and a higher count",
			build: func(b []gm) []gm {
				v := valOf(b, "schema")
				out := append([]gm{}, b...)
				for i := 0; i < 4; i++ {
					out = append(out, member("schema", v))
				}
				return out
			},
		},
		{
			name: "unknown_name_33_bytes", mutant: "unknown_allow_over_32",
			reason: "frame_unknown_field", field: nameOfLength(33),
			origin: "review RUN-260825-c71188, the exact bypass",
			build: func(b []gm) []gm {
				return append(append([]gm{}, b...), member(nameOfLength(33), `"ignored"`))
			},
		},
		{
			name: "unknown_name_1024_bytes", mutant: "unknown_allow_over_32",
			reason: "frame_unknown_field", field: nameOfLength(1024),
			origin: "the same class, far past any plausible cutoff",
			build: func(b []gm) []gm {
				return append(append([]gm{}, b...), member(nameOfLength(1024), `"ignored"`))
			},
		},
		{
			name: "adjacent_repeat_schema", mutant: "dup_only_when_separated",
			reason: "frame_duplicate_field", field: "schema",
			origin: "revision 8, blind to every hand-written row",
			build: func(b []gm) []gm {
				v := valOf(b, "schema")
				var out []gm
				for _, m := range b {
					out = append(out, m)
					if m.decoded == "schema" {
						out = append(out, member("schema", v))
					}
				}
				return out
			},
		},
		{
			name: "homoglyph_of_an_allowed_name", mutant: "unknown_ascii_only",
			reason: "frame_unknown_field", field: "schemа",
			origin: "revision 8, blind: every minted name so far is pure ASCII",
			build: func(b []gm) []gm {
				return append(append([]gm{}, b...), member("schemа", valOf(b, "schema")))
			},
		},
		{
			name: "zero_length_member_name", mutant: "unknown_nonempty_only",
			reason: "frame_unknown_field", field: "",
			origin: "revision 8, blind: no row has ever carried an empty name. NOTE: `mismatch_field` is `omitempty`, so the empty expectation here is indistinguishable from a launcher that did not name the member at all - this row is carried by the REASON, by the never-carried-the-target poll, and by the mutant reproducing the bypass, not by its field assertion",
			build: func(b []gm) []gm {
				return append(append([]gm{}, b...), member("", `"ignored"`))
			},
		},
		{
			name: "escaped_repeat_of_an_allowed_name", mutant: "dup_keyed_on_wire_form",
			reason: "frame_duplicate_field", field: "schema",
			origin: "revision 8, blind: one decoded name written two different ways",
			build: func(b []gm) []gm {
				return append(append([]gm{}, b...), escapedMember("schema", valOf(b, "schema")))
			},
		},
		{
			name: "all_names_escaped_is_a_VALID_frame", mutant: "unknown_by_wire_form",
			invert: true,
			origin: "revision 8, blind and in the other direction: the mutant refuses a frame that must be admitted. Its other half is P23.E - under this same mutant the PLAIN valid frame must still reach execve, which is what distinguishes a wire-form gate from the reject-all fake review RUN-260825-86b7d5 found here",
			build: func(b []gm) []gm {
				var out []gm
				for _, m := range b {
					out = append(out, escapedMember(m.decoded, m.val))
				}
				return out
			},
		},
	}

	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Log("origin: " + row.origin)

			// Cross-layer binding: what P23.A's decoder says about these exact
			// bytes must be what the production entry point does with them.
			predict := func(frame []byte, mutant string) bool {
				accept, _, _ := decoderVerdict(frame, mutant)
				return accept
			}

			t.Run("production_launcher", func(t *testing.T) {
				run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
				frame := buildObject(row.build(baseMembers(t, validBytes(t, prof, key, run.pid, cwd))))
				wantExec := row.invert
				if got := predict(frame, ""); got != wantExec {
					t.Fatalf("layer-1 decoder predicts accept=%v, this row expects %v: the two layers disagree before the run even starts", got, wantExec)
				}
				run.rawFrame(t, frame)
				id, exec := run.becameTarget(target)
				if wantExec {
					if !exec {
						run.waitExit(t)
						t.Fatalf("a VALID frame was refused by the specified launcher: %+v", run.refusal(t))
					}
					t.Logf("ACCEPTED at the production entry: execve %v", id.Argv)
					return
				}
				if exec {
					t.Fatalf("REGRESSION: %s reached execve %v", row.name, id.Argv)
				}
				run.waitExit(t)
				ref := run.refusal(t)
				if ref.Code != "protocol_violation" || ref.Reason != row.reason || ref.Field != row.field {
					t.Fatalf("got %+v, want protocol_violation/%s naming %q", ref, row.reason, row.field)
				}
				if run.everCarried(target) {
					t.Fatalf("refused %+v but the pid carried %s at some point", ref, target)
				}
				t.Logf("REFUSED at the production entry: %+v; pid %d never carried %s", ref, run.pid, target)
			})

			t.Run("narrowed_mutant_"+row.mutant, func(t *testing.T) {
				run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env(row.mutant, "")})
				frame := buildObject(row.build(baseMembers(t, validBytes(t, prof, key, run.pid, cwd))))
				wantExec := !row.invert
				if got := predict(frame, row.mutant); got != wantExec {
					t.Fatalf("layer-1 decoder under mutant %s predicts accept=%v, this row expects %v", row.mutant, got, wantExec)
				}
				run.rawFrame(t, frame)
				id, exec := run.becameTarget(target)
				if wantExec {
					if !exec {
						run.waitExit(t)
						t.Fatalf("mutant %s did not admit %s (%+v): the row does not prove the class is load-bearing", row.mutant, row.name, run.refusal(t))
					}
					t.Logf("BYPASS REPRODUCED: mutant %s admitted %s to execve %v", row.mutant, row.name, id.Argv)
					return
				}
				if exec {
					t.Fatalf("mutant %s admitted the valid frame; this row expects it to over-refuse", row.mutant)
				}
				run.waitExit(t)
				ref := run.refusal(t)
				t.Logf("OVER-REFUSAL REPRODUCED: mutant %s refused a valid frame %+v", row.mutant, ref)
			})
		})
	}

	// The specified launcher's own control. It runs ONLY the specified launcher,
	// which is precisely why it did not discriminate the revision-8 defect: no
	// mutant is under test here. P23.E is the per-mutant version.
	t.Run("all_valid_control_still_execs", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		run.authorize(t, validFrame(prof, key, run.pid, cwd))
		id, ok := run.becameTarget(target)
		if !ok {
			run.waitExit(t)
			t.Fatalf("the control no longer execs: %+v - every refusal row above would be satisfied by a launcher that refuses everything", run.refusal(t))
		}
		t.Logf("CONTROL: execve %v", id.Argv)
	})
}
