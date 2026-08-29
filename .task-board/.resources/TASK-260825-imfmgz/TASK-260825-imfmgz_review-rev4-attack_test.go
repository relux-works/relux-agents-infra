package probe4

import (
	"testing"
	"time"
)

// TestReviewAttackWrongProtocolVersionIsAccepted defeats the revision-4
// launcher model with a frame that has the correct caller-mintable pid, runtime
// key, and plan digest but an incompatible protocol version. Section 6.2 B12
// requires protocol-version equality before execve. The attached P18 launcher
// parses Protocol but never compares it, so the target is incorrectly reached.
func TestReviewAttackWrongProtocolVersionIsAccepted(t *testing.T) {
	dir, prof := p18Project(t, "41")
	key := p18Key(prof)
	run := p18Start(t, dir, dir, key)
	run.authorize(t, p18Frame{
		Schema:      "agents-infra.pi.shared-runtime.auth.v1",
		Protocol:    999,
		RuntimeKey:  key,
		LauncherPid: run.pid,
		PlanDigest:  p18PlanDigest(prof, realPath(t, dir)),
	})

	for i := 0; i < 400; i++ {
		if got, err := Identify(run.pid); err == nil && got.Exe == "/bin/sleep" {
			t.Logf("DEFEATED: protocol_version=999 reached execve on pid %d argv=%v", run.pid, got.Argv)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("wrong-version frame was refused; the reported bypass did not reproduce")
}
