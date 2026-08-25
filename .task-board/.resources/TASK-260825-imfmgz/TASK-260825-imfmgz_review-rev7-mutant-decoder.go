package probe7

// Revision 6's decoder boundary: the authorization frame's field set is closed
// at the DECODER, not at the comparison. Revision 7 leaves that boundary exactly
// where revision 6 put it and changes only what the evidence is able to prove
// about it.
//
// Review RUN-260825-a8a4ef finding F1 showed why the decoder/comparison
// distinction is the whole point. Revision 5 stated a closed five-field set and
// enforced it with five equality comparisons over a Go struct. `json.Unmarshal`
// ignores unknown object members and keeps the last of duplicate keys, so a
// frame carrying a sixth member, or carrying `protocol_version` twice, satisfied
// all five comparisons while transporting a value the launcher never looked at.
// A comparison can only answer "is the value I decoded the one I expect"; it
// cannot answer "were these the only bytes in the frame". That second question
// is the decoder's.
//
// Review RUN-260825-9d5cff then found that the revision-6 PROOF did not bind the
// decoder to the rule it had just written down. Two narrowed gates satisfied
// every named revision-6 row while admitting frames the specification refuses:
//
//	(1) refuse a duplicate only when the two decoded values DIFFER. Both
//	    revision-6 duplicate rows carried 999 and 6, so both stayed green - yet
//	    `protocol_version` repeated twice with the same valid value was admitted,
//	    which is not "exactly once".
//	(2) refuse only the one unknown name the row happened to sample
//	    (`caller_chosen_field`). That row stayed green while any other unknown
//	    member, `future_extension` among them, was admitted.
//
// Neither is caught by a delete-only mutant: deleting a clause proves the clause
// exists, and says nothing about the class it covers. Revision 7 therefore
// carries FOUR narrowed mutants alongside the deletions, and the rows that
// redden them:
//
//	dup_only_if_values_differ         value-sensitive duplicate gate
//	dup_only_protocol_version         field-sampled duplicate gate
//	unknown_only_caller_chosen_field  name-sampled unknown gate
//	unknown_case_folded               case-insensitive allowlist
//	unknown_prefix_allowed            prefix-tolerant allowlist
//
// The duplicate dimension is proved EXHAUSTIVELY rather than by sample, because
// the allowed set is closed and finite: all five members are duplicated with
// their own valid value, one row each. The unknown-name dimension is infinite,
// so it is proved by named near-miss classes, each paired with a narrowed mutant
// that only that class reddens.

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"strings"
)

// frameFields is the closed set, in the order the refusal reports a missing
// member. An implementation may not add to it without a protocol version bump
// (spec section 9).
var frameFields = []string{
	"schema",
	"protocol_version",
	"runtime_key",
	"launcher_pid",
	"exec_plan_digest",
}

type shapeFault struct {
	reason string
	field  string
}

// frameMember is one top-level member as the decoder saw it: the name, and the
// raw bytes of its value. The value is retained for exactly one reason - the
// narrowed mutant `dup_only_if_values_differ` needs it in order to BE that
// mutant. The specified gate never consults it: a duplicate is a duplicate
// whatever it carries.
type frameMember struct {
	name string
	raw  json.RawMessage
}

// shapeEvidence is what the launcher records for the section 12.4 wiring proof:
// the key multiset the DECODER observed, alongside the fields the launcher
// compared. A green suite cannot establish either set by itself - revision 5's
// was green while the decoded multiset held six keys and the compared set held
// five.
type shapeEvidence struct {
	Keys     []string `json:"decoded_keys"`
	Compared []string `json:"compared_fields"`
}

// p21LastKeys is the decoded key multiset from the most recent frame, in stream
// order and with duplicates retained. Retaining duplicates is the point: a
// deduplicating record would report exactly the shape the attack forges.
var p21LastKeys []string

func p21ComparedFields() []string {
	out := append([]string{}, frameFields...)
	sort.Strings(out)
	return out
}

func memberNames(ms []frameMember) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.name)
	}
	return out
}

// p21FrameKeys token-streams one top-level JSON object and returns its members
// in stream order, duplicates retained. It refuses anything that is not exactly
// one object, and anything following that object.
func p21FrameKeys(b []byte, mutant string) ([]frameMember, *shapeFault) {
	r := bytes.NewReader(b)
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return nil, &shapeFault{reason: "frame_unparseable"}
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, &shapeFault{reason: "frame_not_single_object"}
	}
	var members []frameMember
	for {
		t, err := dec.Token()
		if err != nil {
			return nil, &shapeFault{reason: "frame_unparseable"}
		}
		if d, ok := t.(json.Delim); ok && d == '}' {
			break
		}
		k, ok := t.(string)
		if !ok {
			return nil, &shapeFault{reason: "frame_unparseable"}
		}
		// Decode consumes the whole value, nested or not, so the loop only ever
		// sees top-level member names.
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, &shapeFault{reason: "frame_unparseable"}
		}
		members = append(members, frameMember{name: k, raw: raw})
	}
	rest, err := io.ReadAll(io.MultiReader(dec.Buffered(), r))
	if err != nil {
		return nil, &shapeFault{reason: "frame_unparseable"}
	}
	if len(bytes.TrimSpace(rest)) > 0 && mutant != "trailing_ignored" {
		return nil, &shapeFault{reason: "frame_not_single_object"}
	}
	return members, nil
}

// isAllowedName answers whether a decoded member name is inside the closed five.
//
// The specified rule is BYTE EQUALITY against the five names, and revision 7
// says so here rather than leaving it implied, because "a member outside the
// closed five" is a class and the three mutants below are the three ways an
// implementation narrows it without noticing: sampling one name, folding case,
// and matching a prefix. Each is a plausible implementation, each keeps
// revision 6's single unknown row green, and each admits a member the frame may
// not carry.
func isAllowedName(k, mutant string) bool {
	switch mutant {
	case "unknown_only_caller_chosen_field":
		// MUTANT, NARROWED: refuse only the one name revision 6 sampled.
		return k != "caller_chosen_field"
	case "unknown_case_folded":
		// MUTANT, NARROWED: a case-insensitive allowlist admits `Schema`.
		for _, f := range frameFields {
			if strings.EqualFold(k, f) {
				return true
			}
		}
		return false
	case "unknown_prefix_allowed":
		// MUTANT, NARROWED: `strings.HasPrefix` instead of equality admits
		// `exec_plan_digest_v2`.
		for _, f := range frameFields {
			if strings.HasPrefix(k, f) {
				return true
			}
		}
		return false
	case "unknown_allow_over_32":
		// REVIEW MUTANT, NARROWED: reject unknown names only while their byte
		// length is at most 32. Every revision-7 P22.H sample is shorter, but a
		// longer unknown member is admitted even though the specified allowlist
		// is byte equality against the closed five at every length.
		if len(k) > 32 {
			return true
		}
	}
	for _, f := range frameFields {
		if k == f {
			return true
		}
	}
	return false
}

// duplicateRefused answers whether the SECOND occurrence of an allowed member is
// refused. The specified rule ignores both the field and the values: the second
// occurrence is the violation. The two mutants are the two ways that rule gets
// narrowed into something that still passes a single sampled row.
func duplicateRefused(m frameMember, first json.RawMessage, mutant string) bool {
	switch mutant {
	case "dup_only_if_values_differ":
		// MUTANT, NARROWED: `protocol_version: 6` twice is admitted, because the
		// two decoded values agree. Every differing-value row stays green.
		return !bytes.Equal(bytes.TrimSpace(first), bytes.TrimSpace(m.raw))
	case "dup_only_protocol_version":
		// MUTANT, NARROWED: the duplicate rule is applied only to the field the
		// revision-6 rows happened to duplicate. `schema` twice is admitted.
		return m.name == "protocol_version"
	}
	return true
}

// p21CheckKeys applies the closed-set rule to a decoded member multiset. The
// order of the three verdicts is fixed so a test can state which one a given
// frame must produce: unknown members first in stream order, then duplicates in
// stream order, then missing members in frameFields order.
//
// The `mutant` argument disables or NARROWS exactly one clause each, so every
// clause has mutants that redden their own rows and leave the other rows green.
// A mutant that reddens every row would prove only that "some gate exists here".
func p21CheckKeys(members []frameMember, mutant string) *shapeFault {
	if mutant != "unknown_ignored" {
		for _, m := range members {
			if !isAllowedName(m.name, mutant) {
				return &shapeFault{reason: "frame_unknown_field", field: m.name}
			}
		}
	}
	if mutant == "dup_only_exactly_two_total" {
		// REVIEW MUTANT, NARROWED: validate after decoding the multiset, but
		// reject only a total count of two. Every revision-7 duplicate row has
		// arity two, so it stays green; a third occurrence is admitted.
		counts := map[string]int{}
		for _, m := range members {
			if isAllowedName(m.name, mutant) {
				counts[m.name]++
			}
		}
		for _, m := range members {
			if counts[m.name] == 2 {
				return &shapeFault{reason: "frame_duplicate_field", field: m.name}
			}
		}
	} else if mutant != "dup_ignored" && mutant != "dup_ignored_first_wins" {
		seen := map[string]json.RawMessage{}
		for _, m := range members {
			if !isAllowedName(m.name, mutant) {
				continue
			}
			if first, dup := seen[m.name]; dup {
				if duplicateRefused(m, first, mutant) {
					return &shapeFault{reason: "frame_duplicate_field", field: m.name}
				}
				continue
			}
			seen[m.name] = m.raw
		}
	}
	if mutant != "missing_ignored" {
		present := map[string]bool{}
		for _, m := range members {
			present[m.name] = true
		}
		for _, f := range frameFields {
			if !present[f] {
				return &shapeFault{reason: "frame_missing_field", field: f}
			}
		}
	}
	return nil
}

// p21DecodeFrame is the launcher's whole decoder boundary.
//
// `shape_gate_deleted` is revision 5's launcher verbatim - a bare
// json.Unmarshal into the five-member struct - and is what the reviewer's two
// attack frames defeated.
func p21DecodeFrame(b []byte, mutant string) (p21Frame, *shapeFault) {
	var fr p21Frame
	if mutant == "shape_gate_deleted" {
		p21LastKeys = nil
		if json.Unmarshal(b, &fr) != nil {
			return fr, &shapeFault{reason: "frame_unparseable"}
		}
		return fr, nil
	}
	members, fault := p21FrameKeys(b, mutant)
	p21LastKeys = memberNames(members)
	if fault != nil {
		return fr, fault
	}
	if fault := p21CheckKeys(members, mutant); fault != nil {
		return fr, fault
	}
	// The shape is now known to be exactly the five allowed keys once each, so
	// the value decode can no longer discard anything - and, crucially, it no
	// longer matters WHICH duplicate-resolution rule the value decoder uses.
	//
	// That independence is the property `dup_ignored_first_wins` exists to prove.
	// Go's json.Unmarshal keeps the LAST of duplicate keys, so with the duplicate
	// clause deleted it admits `{"protocol_version":999, ...valid...}` and refuses
	// the reverse order. A first-wins decoder - an equally ordinary implementation
	// choice, and the one a hand-rolled streaming parser usually lands on - admits
	// exactly the reverse. Neither order is evidence on its own: each is refused
	// by whichever dedup rule the implementation did not choose, which is what
	// makes a single-order test look like a passing gate.
	if mutant == "dup_ignored_first_wins" {
		return p21FirstWins(b)
	}
	if mutant == "trailing_ignored" {
		// The natural shape of a decoder that does not check for trailing bytes:
		// decode the first value off the stream and never look at the rest.
		// json.Unmarshal would reject the extra object on its own, which would make
		// the mutant redden for the wrong reason.
		if json.NewDecoder(bytes.NewReader(b)).Decode(&fr) != nil {
			return fr, &shapeFault{reason: "frame_unparseable"}
		}
		return fr, nil
	}
	if json.Unmarshal(b, &fr) != nil {
		return fr, &shapeFault{reason: "frame_unparseable"}
	}
	return fr, nil
}

// p21FirstWins decodes retaining the FIRST occurrence of each duplicate key.
func p21FirstWins(b []byte) (p21Frame, *shapeFault) {
	var fr p21Frame
	r := bytes.NewReader(b)
	dec := json.NewDecoder(r)
	if _, err := dec.Token(); err != nil {
		return fr, &shapeFault{reason: "frame_unparseable"}
	}
	first := map[string]json.RawMessage{}
	for {
		t, err := dec.Token()
		if err != nil {
			return fr, &shapeFault{reason: "frame_unparseable"}
		}
		if d, ok := t.(json.Delim); ok && d == '}' {
			break
		}
		k, _ := t.(string)
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return fr, &shapeFault{reason: "frame_unparseable"}
		}
		if _, seen := first[k]; !seen {
			first[k] = raw
		}
	}
	flat, err := json.Marshal(first)
	if err != nil {
		return fr, &shapeFault{reason: "frame_unparseable"}
	}
	if json.Unmarshal(flat, &fr) != nil {
		return fr, &shapeFault{reason: "frame_unparseable"}
	}
	return fr, nil
}
