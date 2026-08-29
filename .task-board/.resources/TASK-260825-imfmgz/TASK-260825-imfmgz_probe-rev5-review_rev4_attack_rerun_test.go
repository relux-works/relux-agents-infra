package probe5

// The revision-4 reviewer's attack, rerun verbatim in shape against revision 5.
//
// RUN-260825-668303 minted a frame with the correct launcher pid, the correct
// runtime key and the correct exec plan digest but `protocol_version = 999`, and
// reported:
//
//	DEFEATED: protocol_version=999 reached execve on pid 26294 argv=[/bin/sleep 41]
//
// Same entry point, same caller-minted frame, same single varied field. The
// assertion is inverted because the launcher now performs the comparison the
// specification always required: the attack must be refused, it must be refused
// by name, and the pid must never have carried the runtime's exec path.
//
// The positive control in the same test is what keeps the refusal meaningful: an
// otherwise identical frame carrying the launcher's own protocol version must
// still reach `execve`. Without it, a launcher that refused everything would
// pass.

import (
	"testing"
)

func TestReviewRev4Attack_WrongProtocolVersionIsNowRefused(t *testing.T) {
	dir, prof := p21Project(t, "41", 5000)
	key := p21Key(prof)
	cwd := realPath(t, dir)

	t.Run("attack_wrong_version_refused", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		f := validFrame(prof, key, run.pid, cwd)
		f.Protocol = 999
		run.authorize(t, f)

		if code := run.waitExit(t); code != exitAuthorizationMismatch {
			t.Fatalf("exit = %d, want runtime_authorization_mismatch (%d)", code, exitAuthorizationMismatch)
		}
		got := run.refusal(t)
		if got.Code != "runtime_authorization_mismatch" || got.Field != "protocol_version" {
			t.Fatalf("refusal = %+v, want runtime_authorization_mismatch on protocol_version", got)
		}
		if run.everCarried("/bin/sleep") {
			t.Fatalf("REPRODUCED: protocol_version=999 still reached execve on pid %d", run.pid)
		}
		t.Logf("REFUSED: protocol_version=999 on pid %d ⇒ %s(%s); the pid never carried /bin/sleep", run.pid, got.Code, got.Field)
	})

	t.Run("control_correct_version_still_execs", func(t *testing.T) {
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		run.authorize(t, validFrame(prof, key, run.pid, cwd))
		id, ok := run.becameTarget("/bin/sleep")
		if !ok {
			t.Fatal("the valid-version control did not exec; the refusal above discriminates nothing")
		}
		t.Logf("CONTROL: protocol_version=%d on pid %d ⇒ execve %v", authProtocol, run.pid, id.Argv)
	})
}
