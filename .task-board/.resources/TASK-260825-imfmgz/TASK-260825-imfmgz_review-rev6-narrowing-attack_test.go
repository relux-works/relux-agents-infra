package reviewrev6

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

var allowedFields = []string{
	"schema",
	"protocol_version",
	"runtime_key",
	"launcher_pid",
	"exec_plan_digest",
}

type narrowedGate struct {
	unknownNameOnly          string
	duplicatesOnlyIfDifferent bool
}

func verdict(raw []byte, mutant narrowedGate) string {
	r := bytes.NewReader(raw)
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return "frame_unparseable"
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return "frame_not_single_object"
	}

	allowed := map[string]bool{}
	for _, key := range allowedFields {
		allowed[key] = true
	}
	seen := map[string]json.RawMessage{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return "frame_unparseable"
		}
		key, ok := tok.(string)
		if !ok {
			return "frame_unparseable"
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return "frame_unparseable"
		}
		if !allowed[key] {
			if mutant.unknownNameOnly == "" || mutant.unknownNameOnly == key {
				return "frame_unknown_field"
			}
			continue
		}
		if previous, duplicate := seen[key]; duplicate {
			if !mutant.duplicatesOnlyIfDifferent || !bytes.Equal(previous, value) {
				return "frame_duplicate_field"
			}
		}
		seen[key] = value
	}
	if _, err := dec.Token(); err != nil {
		return "frame_unparseable"
	}
	rest, err := io.ReadAll(io.MultiReader(dec.Buffered(), r))
	if err != nil {
		return "frame_unparseable"
	}
	if len(bytes.TrimSpace(rest)) != 0 {
		return "frame_not_single_object"
	}
	for _, key := range allowedFields {
		if _, ok := seen[key]; !ok {
			return "frame_missing_field"
		}
	}
	return "accepted"
}

func baseFrame(versionFields string) []byte {
	return []byte(fmt.Sprintf(`{"schema":"agents-infra.pi.shared-runtime.auth.v1",%s,"runtime_key":"key","launcher_pid":42,"exec_plan_digest":"digest"}`, versionFields))
}

func TestP22TableDoesNotProveExactlyOnceForEqualValues(t *testing.T) {
	mutant := narrowedGate{duplicatesOnlyIfDifferent: true}
	cases := map[string]struct {
		raw  []byte
		want string
	}{
		"valid":              {baseFrame(`"protocol_version":6`), "accepted"},
		"unknown_tested_name": {append(baseFrame(`"protocol_version":6`)[:len(baseFrame(`"protocol_version":6`))-1], []byte(`,"caller_chosen_field":"ignored"}`)...), "frame_unknown_field"},
		"wrong_then_valid":   {baseFrame(`"protocol_version":999,"protocol_version":6`), "frame_duplicate_field"},
		"valid_then_wrong":   {baseFrame(`"protocol_version":6,"protocol_version":999`), "frame_duplicate_field"},
		"missing":            {[]byte(`{"schema":"agents-infra.pi.shared-runtime.auth.v1","protocol_version":6,"launcher_pid":42,"exec_plan_digest":"digest"}`), "frame_missing_field"},
		"trailing":           {append(baseFrame(`"protocol_version":6`), []byte(`{"protocol_version":999}`)...), "frame_not_single_object"},
	}
	for name, tc := range cases {
		if got := verdict(tc.raw, mutant); got != tc.want {
			t.Fatalf("P22 row %s reddened under narrowed duplicate gate: got %s want %s", name, got, tc.want)
		}
	}

	attack := baseFrame(`"protocol_version":6,"protocol_version":6`)
	if got := verdict(attack, mutant); got != "accepted" {
		t.Fatalf("attack did not survive narrowed gate: got %s", got)
	}
	if got := verdict(attack, narrowedGate{}); got != "frame_duplicate_field" {
		t.Fatalf("full exactly-once gate should refuse the attack: got %s", got)
	}
	t.Log("SURVIVING MUTANT: reject duplicate fields only when their values differ; every P22 row stays green, but protocol_version=6 repeated twice is admitted")
}

func TestP22TableDoesNotProveTheUnknownNameClass(t *testing.T) {
	mutant := narrowedGate{unknownNameOnly: "caller_chosen_field"}
	tested := append(baseFrame(`"protocol_version":6`)[:len(baseFrame(`"protocol_version":6`))-1], []byte(`,"caller_chosen_field":"ignored"}`)...)
	if got := verdict(tested, mutant); got != "frame_unknown_field" {
		t.Fatalf("the named P22 unknown row reddened: got %s", got)
	}

	attack := append(baseFrame(`"protocol_version":6`)[:len(baseFrame(`"protocol_version":6`))-1], []byte(`,"future_extension":"ignored"}`)...)
	if got := verdict(attack, mutant); got != "accepted" {
		t.Fatalf("alternate unknown key did not survive narrowed gate: got %s", got)
	}
	if got := verdict(attack, narrowedGate{}); got != "frame_unknown_field" {
		t.Fatalf("closed allowed-key set should refuse alternate unknown key: got %s", got)
	}
	t.Log("SURVIVING MUTANT: reject only the one unknown member named by P22; its row stays green, but another unknown member is admitted")
}
