package probe6

// Revision 6's decoder boundary: the authorization frame's field set is closed
// at the DECODER, not at the comparison.
//
// Review RUN-260825-a8a4ef finding F1 showed why the distinction is the whole
// point. Revision 5 stated a closed five-field set and enforced it with five
// equality comparisons over a Go struct. `json.Unmarshal` ignores unknown object
// members and keeps the last of duplicate keys, so a frame carrying a sixth
// member, or carrying `protocol_version` twice, satisfied all five comparisons
// while transporting a value the launcher never looked at. A comparison can only
// answer "is the value I decoded the one I expect"; it cannot answer "were these
// the only bytes in the frame". That second question is the decoder's.

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
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

// p21FrameKeys token-streams one top-level JSON object and returns its member
// names in stream order, duplicates retained. It refuses anything that is not
// exactly one object, and anything following that object.
func p21FrameKeys(b []byte, mutant string) ([]string, *shapeFault) {
	r := bytes.NewReader(b)
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return nil, &shapeFault{reason: "frame_unparseable"}
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, &shapeFault{reason: "frame_not_single_object"}
	}
	var keys []string
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
		keys = append(keys, k)
		// Decode consumes the whole value, nested or not, so the loop only ever
		// sees top-level member names.
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, &shapeFault{reason: "frame_unparseable"}
		}
	}
	rest, err := io.ReadAll(io.MultiReader(dec.Buffered(), r))
	if err != nil {
		return nil, &shapeFault{reason: "frame_unparseable"}
	}
	if len(bytes.TrimSpace(rest)) > 0 && mutant != "trailing_ignored" {
		return nil, &shapeFault{reason: "frame_not_single_object"}
	}
	return keys, nil
}

// p21CheckKeys applies the closed-set rule to a decoded key multiset. The order
// of the three verdicts is fixed so a test can state which one a given frame
// must produce: unknown members first in stream order, then duplicates in stream
// order, then missing members in frameFields order.
//
// The `mutant` argument disables exactly one clause each, so every clause has a
// mutant that reddens its own row and leaves the other rows green. A mutant that
// reddens two rows would prove only that "some gate exists here".
func p21CheckKeys(keys []string, mutant string) *shapeFault {
	allowed := map[string]bool{}
	for _, f := range frameFields {
		allowed[f] = true
	}
	if mutant != "unknown_ignored" {
		for _, k := range keys {
			if !allowed[k] {
				return &shapeFault{reason: "frame_unknown_field", field: k}
			}
		}
	}
	if mutant != "dup_ignored" && mutant != "dup_ignored_first_wins" {
		seen := map[string]int{}
		for _, k := range keys {
			seen[k]++
			if allowed[k] && seen[k] == 2 {
				return &shapeFault{reason: "frame_duplicate_field", field: k}
			}
		}
	}
	if mutant != "missing_ignored" {
		present := map[string]bool{}
		for _, k := range keys {
			present[k] = true
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
	keys, fault := p21FrameKeys(b, mutant)
	p21LastKeys = keys
	if fault != nil {
		return fr, fault
	}
	if fault := p21CheckKeys(keys, mutant); fault != nil {
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
