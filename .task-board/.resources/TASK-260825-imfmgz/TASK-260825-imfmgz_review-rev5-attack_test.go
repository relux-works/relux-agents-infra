package probe5

// Reviewer attack for revision 5's claimed closed authorization-frame field
// set. The production-shaped probe launcher uses ordinary json.Unmarshal into
// p21Frame, which silently ignores unknown object members and accepts duplicate
// keys by retaining the last value. Both inputs therefore carry data that the
// launcher does not compare, despite B11/B12 and tests 12.2.30a/12.4 requiring
// exactly five fields and no carried-but-uncompared field.

import (
	"encoding/json"
	"testing"
)

func TestReviewRev5Attack_ClosedFieldSetIsNotEnforced(t *testing.T) {
	dir, prof := p21Project(t, "61", 5000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	t.Run("unknown_sixth_field_reaches_execve", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		base, err := json.Marshal(validFrame(prof, key, run.pid, cwd))
		if err != nil {
			t.Fatal(err)
		}
		frame := append(append([]byte{}, base[:len(base)-1]...), []byte(`,"caller_chosen_field":"ignored"}`)...)
		if _, err := run.w.Write(frame); err != nil {
			t.Fatal(err)
		}
		if id, ok := run.becameTarget("/bin/sleep"); !ok {
			t.Fatal("unknown sixth field was refused; attack did not reproduce")
		} else {
			t.Logf("DEFEATED: unknown sixth field was ignored and pid %d reached execve %v", run.pid, id.Argv)
		}
	})

	t.Run("duplicate_wrong_then_correct_version_reaches_execve", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		base, err := json.Marshal(validFrame(prof, key, run.pid, cwd))
		if err != nil {
			t.Fatal(err)
		}
		frame := append([]byte(`{"protocol_version":999,`), base[1:]...)
		if _, err := run.w.Write(frame); err != nil {
			t.Fatal(err)
		}
		if id, ok := run.becameTarget("/bin/sleep"); !ok {
			t.Fatal("duplicate protocol_version was refused; attack did not reproduce")
		} else {
			t.Logf("DEFEATED: first protocol_version=999 was ignored and pid %d reached execve %v", run.pid, id.Argv)
		}
	})
}
