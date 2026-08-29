//go:build darwin

package infra

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// These constants are deliberately independent of sharedRuntimeAuthFields.
// The generator and oracle own their record; the production decoder owns its
// compiled set. A drift between them must make the differential red.
var oracleAuthFields = []string{"schema", "protocol_version", "runtime_key", "launcher_pid", "exec_plan_digest"}

type authGeneratedMember struct {
	wire    string
	decoded string
	value   string
}

type authCorpusFrame struct {
	id               string
	dimension        string
	bytes            []byte
	members          []authGeneratedMember
	structural       bool
	structuralAccept bool
	structuralReason string
}

type authShapeVerdict struct {
	accept bool
	reason string
	field  string
}

func authQuoted(name string) string {
	data, err := json.Marshal(name)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func authMember(name, value string) authGeneratedMember {
	return authGeneratedMember{wire: authQuoted(name), decoded: name, value: value}
}

func authEscapedMember(name, value string) authGeneratedMember {
	runes := []rune(name)
	return authGeneratedMember{wire: fmt.Sprintf(`"\u%04x%s"`, runes[0], string(runes[1:])), decoded: name, value: value}
}

func authBuildObject(members []authGeneratedMember) []byte {
	var result strings.Builder
	result.WriteByte('{')
	for index, member := range members {
		if index > 0 {
			result.WriteByte(',')
		}
		result.WriteString(member.wire)
		result.WriteByte(':')
		result.WriteString(member.value)
	}
	result.WriteByte('}')
	return []byte(result.String())
}

func authBaseMembers() []authGeneratedMember {
	return []authGeneratedMember{
		authMember("schema", authQuoted(sharedRuntimeAuthSchema)),
		authMember("protocol_version", fmt.Sprintf("%d", SharedRuntimeProtocolVersion)),
		authMember("runtime_key", authQuoted(strings.Repeat("a", 64))),
		authMember("launcher_pid", "4242"),
		authMember("exec_plan_digest", authQuoted(strings.Repeat("b", 64))),
	}
}

func authValueOf(members []authGeneratedMember, name string) string {
	for _, member := range members {
		if member.decoded == name {
			return member.value
		}
	}
	return `"ignored"`
}

func authFrame(id, dimension string, members []authGeneratedMember) authCorpusFrame {
	return authCorpusFrame{id: id, dimension: dimension, bytes: authBuildObject(members), members: members}
}

func authNameOfLength(length int) string {
	if length == 0 {
		return ""
	}
	const alphabet = "qwxyz0123456789"
	var result strings.Builder
	for index := 0; index < length; index++ {
		result.WriteByte(alphabet[index%len(alphabet)])
	}
	return result.String()
}

func authOracleVerdict(frame authCorpusFrame) authShapeVerdict {
	if frame.structural {
		return authShapeVerdict{accept: frame.structuralAccept, reason: frame.structuralReason}
	}
	allowed := map[string]bool{}
	for _, name := range oracleAuthFields {
		allowed[name] = true
	}
	for _, member := range frame.members {
		if !allowed[member.decoded] {
			return authShapeVerdict{reason: "frame_unknown_field", field: member.decoded}
		}
	}
	seen := map[string]bool{}
	for _, member := range frame.members {
		if seen[member.decoded] {
			return authShapeVerdict{reason: "frame_duplicate_field", field: member.decoded}
		}
		seen[member.decoded] = true
	}
	for _, name := range oracleAuthFields {
		if !seen[name] {
			return authShapeVerdict{reason: "frame_missing_field", field: name}
		}
	}
	return authShapeVerdict{accept: true}
}

func authProductionVerdict(raw []byte) authShapeVerdict {
	_, _, err := decodeSharedRuntimeAuthorizationFrame(raw)
	if err == nil {
		return authShapeVerdict{accept: true}
	}
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) {
		return authShapeVerdict{reason: "unexpected_error"}
	}
	return authShapeVerdict{reason: shared.Reason, field: shared.MismatchField}
}

func authBuildCorpus() []authCorpusFrame {
	base := authBaseMembers()
	corpus := []authCorpusFrame{}
	add := func(id, dimension string, members []authGeneratedMember) {
		corpus = append(corpus, authFrame(id, dimension, members))
	}

	for _, field := range oracleAuthFields {
		for _, count := range []int{0, 1, 2, 3, 4, 5, 8, 13} {
			members := []authGeneratedMember{}
			for _, member := range base {
				if member.decoded != field {
					members = append(members, member)
					continue
				}
				for index := 0; index < count; index++ {
					members = append(members, authMember(field, member.value))
				}
			}
			add(fmt.Sprintf("occurrence/%s/x%d", field, count), "occurrence", members)
		}
	}
	multiple := []authGeneratedMember{}
	for _, member := range base {
		multiple = append(multiple, member)
		if member.decoded == "schema" || member.decoded == "runtime_key" {
			multiple = append(multiple, member, member)
		}
	}
	add("occurrence/two_members_x3", "occurrence", multiple)
	for _, count := range []int{2, 3, 7} {
		members := []authGeneratedMember{}
		for index := 0; index < count; index++ {
			members = append(members, base...)
		}
		add(fmt.Sprintf("occurrence/whole_x%d", count), "occurrence", members)
	}

	for _, field := range oracleAuthFields {
		value := authValueOf(base, field)
		adjacent := []authGeneratedMember{}
		for _, member := range base {
			adjacent = append(adjacent, member)
			if member.decoded == field {
				adjacent = append(adjacent, authMember(field, value))
			}
		}
		add("position/adjacent/"+field, "position", adjacent)
		separated := append(append([]authGeneratedMember{}, base...), authMember(field, value))
		add("position/separated/"+field, "position", separated)
	}
	unknown := authMember("caller_chosen_field", `"ignored"`)
	add("position/unknown_first", "position", append([]authGeneratedMember{unknown}, base...))
	middle := append([]authGeneratedMember{}, base[:2]...)
	middle = append(middle, unknown)
	middle = append(middle, base[2:]...)
	add("position/unknown_middle", "position", middle)
	add("position/unknown_last", "position", append(append([]authGeneratedMember{}, base...), unknown))

	for _, length := range []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257, 511, 512, 1023, 1024} {
		members := append(append([]authGeneratedMember{}, base...), authMember(authNameOfLength(length), `"ignored"`))
		add(fmt.Sprintf("length/unknown_%dB", length), "length", members)
	}

	derivations := []struct {
		name string
		make func(string) string
	}{
		{"upper", strings.ToUpper},
		{"title", func(value string) string { return strings.ToUpper(value[:1]) + value[1:] }},
		{"suffix_v2", func(value string) string { return value + "_v2" }},
		{"suffix_digit", func(value string) string { return value + "2" }},
		{"prefixed", func(value string) string { return "x" + value }},
		{"truncated", func(value string) string { return value[:len(value)-1] }},
		{"trailing_space", func(value string) string { return value + " " }},
		{"leading_space", func(value string) string { return " " + value }},
		{"trailing_nul", func(value string) string { return value + "\x00" }},
		{"dashed", func(value string) string { return strings.ReplaceAll(value, "_", "-") }},
		{"doubled", func(value string) string { return value + value }},
	}
	for _, field := range oracleAuthFields {
		for _, derivation := range derivations {
			name := derivation.make(field)
			if name == field {
				continue
			}
			members := append(append([]authGeneratedMember{}, base...), authMember(name, authValueOf(base, field)))
			add("identity/"+derivation.name+"/"+field, "identity", members)
		}
		if index := strings.IndexByte(field, 'a'); index >= 0 {
			name := field[:index] + "а" + field[index+1:]
			members := append(append([]authGeneratedMember{}, base...), authMember(name, authValueOf(base, field)))
			add("identity/homoglyph/"+field, "identity", members)
		}
	}

	for _, field := range oracleAuthFields {
		escaped := []authGeneratedMember{}
		for _, member := range base {
			if member.decoded == field {
				escaped = append(escaped, authEscapedMember(field, member.value))
			} else {
				escaped = append(escaped, member)
			}
		}
		add("encoding/escaped_"+field, "encoding", escaped)
		add("encoding/escaped_repeat/"+field, "encoding", append(append([]authGeneratedMember{}, base...), authEscapedMember(field, authValueOf(base, field))))
	}
	allEscaped := []authGeneratedMember{}
	for _, member := range base {
		allEscaped = append(allEscaped, authEscapedMember(member.decoded, member.value))
	}
	add("encoding/all_escaped", "encoding", allEscaped)
	add("encoding/escaped_unknown", "encoding", append(append([]authGeneratedMember{}, base...), authEscapedMember("caller_chosen_field", `"ignored"`)))

	permutations := [][]int{{0, 1, 2, 3, 4}, {4, 3, 2, 1, 0}, {2, 0, 4, 1, 3}, {1, 3, 0, 4, 2}, {3, 4, 1, 2, 0}, {0, 4, 3, 1, 2}, {2, 3, 4, 0, 1}, {4, 0, 1, 3, 2}}
	for index, permutation := range permutations {
		members := []authGeneratedMember{}
		for _, memberIndex := range permutation {
			members = append(members, base[memberIndex])
		}
		add(fmt.Sprintf("position/permutation_%d", index), "position", members)
	}

	random := rand.New(rand.NewSource(8))
	const printable = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !#$%&()*+,-./:;<=>?@[]^_`{|}~"
	randomName := func(length int) string {
		var result strings.Builder
		for index := 0; index < length; index++ {
			result.WriteByte(printable[random.Intn(len(printable))])
		}
		return result.String()
	}
	isOracleAllowed := func(name string) bool {
		for _, allowed := range oracleAuthFields {
			if name == allowed {
				return true
			}
		}
		return false
	}
	for index := 0; index < 128; index++ {
		name := randomName(1 + random.Intn(64))
		if isOracleAllowed(name) {
			continue
		}
		add(fmt.Sprintf("random/unknown_%d", index), "random", append(append([]authGeneratedMember{}, base...), authMember(name, `"ignored"`)))
	}
	for index := 0; index < 96; index++ {
		members := append([]authGeneratedMember{}, base...)
		switch random.Intn(4) {
		case 0:
			drop := oracleAuthFields[random.Intn(len(oracleAuthFields))]
			filtered := []authGeneratedMember{}
			for _, member := range members {
				if member.decoded != drop {
					filtered = append(filtered, member)
				}
			}
			members = filtered
		case 1:
			field := oracleAuthFields[random.Intn(len(oracleAuthFields))]
			for count := 1 + random.Intn(5); count > 0; count-- {
				members = append(members, authMember(field, authValueOf(base, field)))
			}
		case 2:
			for count := 1 + random.Intn(3); count > 0; count-- {
				name := randomName(1 + random.Intn(48))
				if !isOracleAllowed(name) {
					members = append(members, authMember(name, `"ignored"`))
				}
			}
		case 3:
		}
		random.Shuffle(len(members), func(left, right int) { members[left], members[right] = members[right], members[left] })
		add(fmt.Sprintf("random/mixed_%d", index), "random", members)
	}

	valid := string(authBuildObject(base))
	structural := []struct {
		id     string
		body   string
		accept bool
		reason string
	}{
		{"array", `[1,2,3]`, false, "frame_not_single_object"},
		{"string", `"schema"`, false, "frame_not_single_object"},
		{"number", `42`, false, "frame_not_single_object"},
		{"null", `null`, false, "frame_not_single_object"},
		{"true", `true`, false, "frame_not_single_object"},
		{"empty", ``, false, "frame_unparseable"},
		{"truncated", valid[:len(valid)-1], false, "frame_unparseable"},
		{"two_objects", valid + valid, false, "frame_not_single_object"},
		{"object_garbage", valid + `xyz`, false, "frame_not_single_object"},
		{"object_number", valid + `7`, false, "frame_not_single_object"},
		{"trailing_whitespace", valid + "  \n\t", true, ""},
		{"leading_whitespace", "  \n\t" + valid, true, ""},
		{"pretty", authPrettyObject(base), true, ""},
	}
	add("structural/empty_object", "structural", nil)
	for _, shape := range structural {
		corpus = append(corpus, authCorpusFrame{id: "structural/" + shape.id, dimension: "structural", bytes: []byte(shape.body), structural: true, structuralAccept: shape.accept, structuralReason: shape.reason})
	}
	return corpus
}

func authPrettyObject(members []authGeneratedMember) string {
	var result strings.Builder
	result.WriteString("{\n")
	for index, member := range members {
		if index > 0 {
			result.WriteString(",\n")
		}
		result.WriteString("  " + member.wire + " : " + member.value)
	}
	result.WriteString("\n}\n")
	return result.String()
}

func TestSharedRuntimeAuthorizationShapeOracleDifferential(t *testing.T) {
	corpus := authBuildCorpus()
	accepted, refused := 0, 0
	dimensions := map[string]int{}
	for _, frame := range corpus {
		want := authOracleVerdict(frame)
		got := authProductionVerdict(frame.bytes)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: production=%#v oracle=%#v raw=%q", frame.id, got, want, frame.bytes)
		}
		dimensions[frame.dimension]++
		if want.accept {
			accepted++
		} else {
			refused++
		}
	}
	if len(corpus) != 398 || accepted == 0 || refused == 0 {
		t.Fatalf("corpus shape count=%d accepted=%d refused=%d dimensions=%v", len(corpus), accepted, refused, dimensions)
	}
	t.Logf("P23.A production decoder agrees with the independent oracle on 398 frames: %d accepted / %d refused", accepted, refused)
}

type authShapeMutant struct {
	name      string
	direction string
	blindTo   []string
}

const (
	authAdmits      = "admits"
	authOverRefuses = "over_refuses"
	authBaseline6   = "rev6"
	authBaseline7   = "rev7"
	authBaselineC   = "review_c71188"
)

var authAllBaselines = []string{authBaseline6, authBaseline7, authBaselineC}

var authShapeMutants = []authShapeMutant{
	{"unknown_ignored", authAdmits, nil},
	{"dup_ignored", authAdmits, nil},
	{"dup_ignored_first_wins", authAdmits, nil},
	{"trailing_ignored", authAdmits, []string{authBaseline7, authBaselineC}},
	{"missing_ignored", authAdmits, []string{authBaseline7, authBaselineC}},
	{"shape_gate_deleted", authAdmits, nil},
	{"dup_only_if_values_differ", authAdmits, []string{authBaseline6}},
	{"dup_only_protocol_version", authAdmits, []string{authBaseline6, authBaselineC}},
	{"unknown_only_caller_chosen_field", authAdmits, []string{authBaseline6}},
	{"unknown_case_folded", authAdmits, []string{authBaseline6, authBaselineC}},
	{"unknown_prefix_allowed", authAdmits, []string{authBaseline6, authBaselineC}},
	{"dup_only_exactly_two_total", authAdmits, []string{authBaseline6, authBaseline7}},
	{"unknown_allow_over_32", authAdmits, []string{authBaseline6, authBaseline7}},
	{"dup_only_when_separated", authAdmits, []string{authBaseline6, authBaselineC}},
	{"unknown_ascii_only", authAdmits, authAllBaselines},
	{"unknown_nonempty_only", authAdmits, authAllBaselines},
	{"dup_keyed_on_wire_form", authAdmits, authAllBaselines},
	{"unknown_by_wire_form", authOverRefuses, authAllBaselines},
}

func authMutantAllowed(member authGeneratedMember, mutant string) bool {
	switch mutant {
	case "unknown_ignored":
		return true
	case "unknown_only_caller_chosen_field":
		return member.decoded != "caller_chosen_field"
	case "unknown_case_folded":
		for _, allowed := range oracleAuthFields {
			if strings.EqualFold(member.decoded, allowed) {
				return true
			}
		}
		return false
	case "unknown_prefix_allowed":
		for _, allowed := range oracleAuthFields {
			if strings.HasPrefix(member.decoded, allowed) {
				return true
			}
		}
		return false
	case "unknown_allow_over_32":
		if len(member.decoded) > 32 {
			return true
		}
	case "unknown_ascii_only":
		for _, character := range member.decoded {
			if character > 127 {
				return true
			}
		}
	case "unknown_nonempty_only":
		if member.decoded == "" {
			return true
		}
	case "unknown_by_wire_form":
		for _, allowed := range oracleAuthFields {
			if member.wire == authQuoted(allowed) {
				return true
			}
		}
		return false
	case "reject_all_probe":
		return false
	}
	for _, allowed := range oracleAuthFields {
		if member.decoded == allowed {
			return true
		}
	}
	return false
}

func authMutantVerdict(frame authCorpusFrame, mutant string) authShapeVerdict {
	if frame.structural {
		if mutant == "trailing_ignored" && (strings.Contains(frame.id, "two_objects") || strings.Contains(frame.id, "object_garbage") || strings.Contains(frame.id, "object_number") || strings.Contains(frame.id, "trailing_object")) {
			return authShapeVerdict{accept: true}
		}
		return authOracleVerdict(frame)
	}
	if mutant == "shape_gate_deleted" {
		return authShapeVerdict{accept: true}
	}
	for _, member := range frame.members {
		if !authMutantAllowed(member, mutant) {
			return authShapeVerdict{reason: "frame_unknown_field", field: member.decoded}
		}
	}
	if mutant != "dup_ignored" && mutant != "dup_ignored_first_wins" {
		switch mutant {
		case "dup_only_exactly_two_total":
			counts := map[string]int{}
			for _, member := range frame.members {
				counts[member.decoded]++
			}
			for _, member := range frame.members {
				if counts[member.decoded] == 2 {
					return authShapeVerdict{reason: "frame_duplicate_field", field: member.decoded}
				}
			}
		case "dup_only_when_separated":
			for index, member := range frame.members {
				for prior := 0; prior < index; prior++ {
					if frame.members[prior].decoded == member.decoded && prior != index-1 {
						return authShapeVerdict{reason: "frame_duplicate_field", field: member.decoded}
					}
				}
			}
		default:
			seen := map[string]authGeneratedMember{}
			for _, member := range frame.members {
				key := member.decoded
				if mutant == "dup_keyed_on_wire_form" {
					key = member.wire
				}
				first, duplicate := seen[key]
				if !duplicate {
					seen[key] = member
					continue
				}
				refuse := true
				if mutant == "dup_only_if_values_differ" {
					refuse = strings.TrimSpace(first.value) != strings.TrimSpace(member.value)
				}
				if mutant == "dup_only_protocol_version" {
					refuse = member.decoded == "protocol_version"
				}
				if refuse {
					return authShapeVerdict{reason: "frame_duplicate_field", field: member.decoded}
				}
			}
		}
	}
	if mutant != "missing_ignored" {
		present := map[string]bool{}
		for _, member := range frame.members {
			present[member.decoded] = true
		}
		for _, allowed := range oracleAuthFields {
			if !present[allowed] {
				return authShapeVerdict{reason: "frame_missing_field", field: allowed}
			}
		}
	}
	return authShapeVerdict{accept: true}
}

func authProductionMutantVerdict(input sharedAuthShapeInput, mutant string) sharedAuthShapeVerdict {
	members := make([]authGeneratedMember, 0, len(input.Members))
	for _, member := range input.Members {
		members = append(members, authGeneratedMember{
			wire:    member.WireName,
			decoded: member.DecodedName,
			value:   string(member.Value),
		})
	}
	frame := authCorpusFrame{
		id:               "production/delivered",
		bytes:            authBuildObject(members),
		members:          members,
		structural:       input.StructuralErr != nil,
		structuralReason: input.StructuralReason,
	}
	if input.CompleteObject && input.StructuralReason == "frame_not_single_object" {
		frame.id = "production/trailing_object"
	}
	verdict := authMutantVerdict(frame, mutant)
	return sharedAuthShapeVerdict{
		Accepted:      verdict.accept,
		Reason:        verdict.reason,
		MismatchField: verdict.field,
	}
}

func authMutantVerdictFromProductionBytes(raw []byte, mutant string) authShapeVerdict {
	_, members, completeObject, structuralReason, structuralErr := tokenizeSharedAuthorizationFrame(raw)
	verdict := authProductionMutantVerdict(sharedAuthShapeInput{
		Members:          members,
		StructuralReason: structuralReason,
		StructuralErr:    structuralErr,
		CompleteObject:   completeObject,
	}, mutant)
	return authShapeVerdict{accept: verdict.Accepted, reason: verdict.Reason, field: verdict.MismatchField}
}

type authLegacyRow struct {
	baseline string
	id       string
	frame    authCorpusFrame
}

func authLegacyRows() []authLegacyRow {
	base := authBaseMembers()
	valid := authFrame("valid", "legacy", base)
	duplicate := func(field, value string) authCorpusFrame {
		return authFrame("dup_"+field, "legacy", append(append([]authGeneratedMember{}, base...), authMember(field, value)))
	}
	unknown := func(name, source string) authCorpusFrame {
		return authFrame("unknown_"+name, "legacy", append(append([]authGeneratedMember{}, base...), authMember(name, authValueOf(base, source))))
	}
	missing := []authGeneratedMember{}
	for _, member := range base {
		if member.decoded != "runtime_key" {
			missing = append(missing, member)
		}
	}
	trailing := authCorpusFrame{id: "trailing_object", structural: true, structuralReason: "frame_not_single_object", bytes: append(append([]byte{}, valid.bytes...), valid.bytes...)}
	rows := []authLegacyRow{
		{authBaseline6, "valid", valid},
		{authBaseline6, "unknown_caller", unknown("caller_chosen_field", "schema")},
		{authBaseline6, "dup_wrong_first", duplicate("protocol_version", "999")},
		{authBaseline6, "dup_wrong_last", duplicate("protocol_version", "999")},
		{authBaseline6, "missing_runtime_key", authFrame("missing", "legacy", missing)},
		{authBaseline6, "trailing_object", trailing},
		{authBaseline7, "valid", valid},
	}
	for _, field := range oracleAuthFields {
		rows = append(rows, authLegacyRow{authBaseline7, "dup_same_" + field, duplicate(field, authValueOf(base, field))})
	}
	rows = append(rows,
		authLegacyRow{authBaseline7, "unknown_future", unknown("future_extension", "schema")},
		authLegacyRow{authBaseline7, "unknown_case", unknown("Schema", "schema")},
		authLegacyRow{authBaseline7, "unknown_case_wrong_value", authFrame("case_wrong", "legacy", append(append([]authGeneratedMember{}, base...), authMember("Schema", `"wrong"`)))},
		authLegacyRow{authBaseline7, "unknown_prefix", unknown("exec_plan_digest_v2", "exec_plan_digest")},
		authLegacyRow{authBaselineC, "valid", valid},
	)
	three := append(append([]authGeneratedMember{}, base...), authMember("protocol_version", authValueOf(base, "protocol_version")), authMember("protocol_version", authValueOf(base, "protocol_version")))
	rows = append(rows,
		authLegacyRow{authBaselineC, "arity_three", authFrame("arity_three", "legacy", three)},
		authLegacyRow{authBaselineC, "unknown_33", unknown(authNameOfLength(33), "schema")},
	)
	return rows
}

func TestSharedRuntimeAuthorizationMutantCalibrationAndHarnessNegatives(t *testing.T) {
	corpus := authBuildCorpus()
	plain := authFrame("plain", "control", authBaseMembers())
	legacy := authLegacyRows()
	if len(legacy) != 19 {
		t.Fatalf("legacy row inventory=%d want=19", len(legacy))
	}
	legacyBytes := map[string]bool{}
	for _, row := range legacy {
		legacyBytes[string(row.frame.bytes)] = true
	}
	for _, mutant := range authShapeMutants {
		t.Run(mutant.name, func(t *testing.T) {
			if verdict := authMutantVerdictFromProductionBytes(plain.bytes, mutant.name); !verdict.accept {
				t.Fatalf("rule 1: mutant refuses plain valid frame: %#v", verdict)
			}
			admits, overRefuses := []string{}, []string{}
			for _, frame := range corpus {
				want, got := authOracleVerdict(frame), authMutantVerdictFromProductionBytes(frame.bytes, mutant.name)
				if got.accept && !want.accept {
					admits = append(admits, frame.id)
				}
				if !got.accept && want.accept {
					overRefuses = append(overRefuses, frame.id)
				}
			}
			if len(admits)+len(overRefuses) == 0 {
				t.Fatal("coverage hole: corpus kills no frame")
			}
			if mutant.direction == authAdmits && (len(admits) == 0 || len(overRefuses) != 0) {
				t.Fatalf("rule 2 direction mismatch: admits=%v over_refuses=%v", admits, overRefuses)
			}
			if mutant.direction == authOverRefuses && (len(overRefuses) == 0 || len(admits) != 0) {
				t.Fatalf("rule 2 direction mismatch: admits=%v over_refuses=%v", admits, overRefuses)
			}

			divergences := map[string][]string{}
			for _, row := range legacy {
				want, got := authOracleVerdict(row.frame), authMutantVerdictFromProductionBytes(row.frame.bytes, mutant.name)
				if !reflect.DeepEqual(want, got) {
					divergences[row.baseline] = append(divergences[row.baseline], row.id)
				}
			}
			declared := map[string]bool{}
			for _, baseline := range mutant.blindTo {
				declared[baseline] = true
			}
			for _, baseline := range authAllBaselines {
				measuredBlind := len(divergences[baseline]) == 0
				if measuredBlind != declared[baseline] {
					t.Fatalf("rule 3 blindness mismatch for %s: declared=%t divergences=%v", baseline, declared[baseline], divergences[baseline])
				}
			}
			if len(mutant.blindTo) > 0 {
				witness := ""
				for _, frame := range corpus {
					if legacyBytes[string(frame.bytes)] {
						continue
					}
					if !reflect.DeepEqual(authOracleVerdict(frame), authMutantVerdictFromProductionBytes(frame.bytes, mutant.name)) {
						witness = frame.id
						break
					}
				}
				if witness == "" {
					t.Fatal("blind mutant has no corpus-only byte-distinct witness")
				}
			}
			first := admits
			if mutant.direction == authOverRefuses {
				first = overRefuses
			}
			t.Logf("KILLED %s by %s; admits=%d over_refuses=%d blind_to=%v", mutant.name, first[0], len(admits), len(overRefuses), mutant.blindTo)
		})
	}

	t.Run("reject_all_probe_reddens_all_harness_rules", func(t *testing.T) {
		if authMutantVerdictFromProductionBytes(plain.bytes, "reject_all_probe").accept {
			t.Fatal("rule 1 negative did not redden")
		}
		admits, over := 0, 0
		for _, frame := range corpus {
			want, got := authOracleVerdict(frame), authMutantVerdictFromProductionBytes(frame.bytes, "reject_all_probe")
			if got.accept && !want.accept {
				admits++
			}
			if !got.accept && want.accept {
				over++
			}
		}
		if admits != 0 || over == 0 {
			t.Fatalf("rule 2 negative shape changed: admits=%d over=%d", admits, over)
		}
		caught := map[string]bool{}
		for _, row := range legacy {
			if !reflect.DeepEqual(authOracleVerdict(row.frame), authMutantVerdictFromProductionBytes(row.frame.bytes, "reject_all_probe")) {
				caught[row.baseline] = true
			}
		}
		if len(caught) != 3 {
			t.Fatalf("rule 3 negative did not redden every baseline: %v", caught)
		}
	})

	if SharedRuntimeProtocolVersion != 7 {
		t.Fatalf("revision 9 protocol version=%d want=6", SharedRuntimeProtocolVersion)
	}
	checked := 0
	for _, frame := range append(corpus, func() []authCorpusFrame {
		rows := authLegacyRows()
		result := make([]authCorpusFrame, 0, len(rows))
		for _, row := range rows {
			result = append(result, row.frame)
		}
		return result
	}()...) {
		if got, want := authProductionVerdict(frame.bytes), authOracleVerdict(frame); !reflect.DeepEqual(got, want) {
			t.Fatalf("rev8_unknown_wiring disagreement on %s: production=%#v rev8=%#v", frame.id, got, want)
		}
		checked++
	}
	if checked != 417 {
		t.Fatalf("no-behaviour-change measurement=%d want=417", checked)
	}

	names := make([]string, 0, len(authShapeMutants))
	for _, mutant := range authShapeMutants {
		names = append(names, mutant.name)
	}
	sort.Strings(names)
	t.Logf("P23.B/D/F: 18 calibrated mutants, measured blindness, reject-all negative, and 417-frame revision wiring agreement: %s", strings.Join(names, ", "))
}
