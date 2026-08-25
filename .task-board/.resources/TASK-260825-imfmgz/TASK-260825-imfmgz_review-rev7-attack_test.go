package probe7

import (
	"encoding/json"
	"strings"
	"testing"
)

func dupThreeSameValue(t *testing.T, b []byte, name string) []byte {
	t.Helper()
	v := rawValueOf(t, b, name)
	member := `"` + name + `":` + string(v)
	return withMember(withMember(b, member), member)
}

func TestReviewRev7DuplicateArityNarrowingSurvivesNamedRows(t *testing.T) {
	dir, prof := p21Project(t, "67", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	rows := []struct {
		name  string
		field string
		frame func(*testing.T, []byte) []byte
	}{
		{"wrong_then_valid", "protocol_version", func(t *testing.T, b []byte) []byte { return prepMember(b, `"protocol_version":999`) }},
		{"valid_then_wrong", "protocol_version", func(t *testing.T, b []byte) []byte { return withMember(b, `"protocol_version":999`) }},
		{"same_schema", "schema", dupSame("schema")},
		{"same_protocol_version", "protocol_version", dupSame("protocol_version")},
		{"same_runtime_key", "runtime_key", dupSame("runtime_key")},
		{"same_launcher_pid", "launcher_pid", dupSame("launcher_pid")},
		{"same_exec_plan_digest", "exec_plan_digest", dupSame("exec_plan_digest")},
	}

	for _, row := range rows {
		row := row
		t.Run("existing_"+row.name+"_stays_green", func(t *testing.T) {
			run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env("dup_only_exactly_two_total", "")})
			frame := row.frame(t, validBytes(t, prof, key, run.pid, cwd))
			run.rawFrame(t, frame)
			if _, ok := run.becameTarget(target); ok {
				t.Fatalf("arity mutant unexpectedly admitted existing row %s", row.name)
			}
			run.waitExit(t)
			got := run.refusal(t)
			if got.Code != "protocol_violation" || got.Reason != "frame_duplicate_field" || got.Field != row.field {
				t.Fatalf("existing row %s changed under arity mutant: %+v", row.name, got)
			}
		})
	}

	t.Run("third_occurrence_reaches_execve", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env("dup_only_exactly_two_total", "")})
		frame := dupThreeSameValue(t, validBytes(t, prof, key, run.pid, cwd), "protocol_version")
		run.rawFrame(t, frame)
		if id, ok := run.becameTarget(target); !ok {
			run.waitExit(t)
			t.Fatalf("three-occurrence attack refused %+v; expected narrowed mutant to admit it", run.refusal(t))
		} else {
			t.Logf("BYPASS: arity-two-only duplicate mutant admitted three occurrences to execve %v", id.Argv)
		}
	})
}

func TestReviewRev7UnknownLengthNarrowingSurvivesNamedClasses(t *testing.T) {
	dir, prof := p21Project(t, "68", 4000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	for _, u := range unknownClasses {
		u := u
		t.Run("existing_"+u.key+"_stays_green", func(t *testing.T) {
			run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env("unknown_allow_over_32", "")})
			frame := u.frame(t, validBytes(t, prof, key, run.pid, cwd))
			run.rawFrame(t, frame)
			if _, ok := run.becameTarget(target); ok {
				t.Fatalf("length mutant unexpectedly admitted existing row %s", u.key)
			}
			run.waitExit(t)
			got := run.refusal(t)
			if u.key == "unknown_case_variant_wrong_value" {
				if got.Code != "protocol_violation" || got.Reason != "frame_unknown_field" {
					t.Fatalf("existing wrong-value case row changed: %+v", got)
				}
				return
			}
			if got.Code != "protocol_violation" || got.Reason != "frame_unknown_field" || got.Field != u.name {
				t.Fatalf("existing row %s changed under length mutant: %+v", u.key, got)
			}
		})
	}

	t.Run("long_unknown_member_reaches_execve", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: p22Env("unknown_allow_over_32", "")})
		longName := strings.Repeat("x", 33)
		member, err := json.Marshal(longName)
		if err != nil {
			t.Fatal(err)
		}
		frame := withMember(validBytes(t, prof, key, run.pid, cwd), string(member)+`:"ignored"`)
		run.rawFrame(t, frame)
		if id, ok := run.becameTarget(target); !ok {
			run.waitExit(t)
			t.Fatalf("long-name attack refused %+v; expected narrowed mutant to admit it", run.refusal(t))
		} else {
			t.Logf("BYPASS: length-limited unknown gate admitted %d-byte name to execve %v", len(longName), id.Argv)
		}
	})
}
