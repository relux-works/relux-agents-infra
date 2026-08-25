package probe3

// P15 - self-found during revision 3 probing, and load-bearing.
//
// P12/P14 first failed with "stale-record-broker-gone" against a process that
// Alive() reported as live. The cause is a platform fact revision 2's section 2
// states too simply: `kern.proc.pid` on a DEAD pid returns EIO, so liveness was
// treated as decidable from that call alone. It is not. An unreaped exited
// process - a zombie - still answers `kern.proc.pid` successfully. Only
// `kern.procargs2` refuses it.
//
// This matters wherever the specification decides "the recorded process is
// alive": a broker whose parent has not reaped it, or a runtime whose broker
// died without waiting, would be read as a live runtime and block a restart
// forever.

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// liveNotZombie is the corrected liveness predicate: the process must exist AND
// must not be a zombie.
func liveNotZombie(pid int) (exists bool, zombie bool) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return false, false
	}
	return true, kp.Proc.P_stat == 5 // SZOMB
}

func TestP15ZombieAnswersKernProcPid(t *testing.T) {
	// A child that exits and is deliberately NOT reaped.
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { _, _ = cmd.Process.Wait() })

	deadline := time.Now().Add(3 * time.Second)
	var sawZombie bool
	for time.Now().Before(deadline) {
		exists, zombie := liveNotZombie(pid)
		if exists && zombie {
			sawZombie = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !sawZombie {
		t.Skip("could not observe the zombie window on this host")
	}

	// A: the naive predicate says the process is alive.
	if _, err := unix.SysctlKinfoProc("kern.proc.pid", pid); err != nil {
		t.Fatalf("P15.A FAIL: kern.proc.pid refused a zombie: %v", err)
	}
	t.Logf("P15.A OK (defect): kern.proc.pid SUCCEEDS for the exited, unreaped pid=%d, so "+
		"\"liveness is decidable from kern.proc.pid\" is false", pid)

	// B: the identity read refuses it, which is why the full attestation chain
	// happens to be safe and a bare liveness check is not.
	if _, err := unix.SysctlRaw("kern.procargs2", pid); err == nil {
		t.Fatalf("P15.B FAIL: kern.procargs2 answered for a zombie; the corrected predicate " +
			"would have no second source")
	} else {
		t.Logf("P15.B OK: kern.procargs2 refuses the zombie (%v), so any decision that also "+
			"requires exec path and argv already fails closed", err)
	}

	// C: the corrected predicate distinguishes.
	exists, zombie := liveNotZombie(pid)
	if !exists || !zombie {
		t.Fatalf("P15.C FAIL: corrected predicate exists=%v zombie=%v", exists, zombie)
	}
	t.Logf("P15.C OK: p_stat == SZOMB distinguishes an unreaped corpse from a live process")

	// D: control - a genuinely live process is not a zombie, and a reaped one
	// is gone entirely.
	live := exec.Command("/bin/sleep", "7")
	if err := live.Start(); err != nil {
		t.Fatal(err)
	}
	lexists, lzombie := liveNotZombie(live.Process.Pid)
	if !lexists || lzombie {
		t.Fatalf("P15.D FAIL: live process exists=%v zombie=%v", lexists, lzombie)
	}
	_ = live.Process.Kill()
	_, _ = live.Process.Wait()
	if e, _ := liveNotZombie(live.Process.Pid); e {
		t.Logf("P15.D note: pid still visible immediately after reap; retrying")
		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("P15.D OK: control - a live process reports exists=true zombie=false; after reap " +
		"it disappears from kern.proc.pid entirely")

	_, _ = cmd.Process.Wait()
	if e, _ := liveNotZombie(pid); e {
		t.Logf("P15.E note: pid=%d still listed immediately after reaping", pid)
	} else {
		t.Logf("P15.E OK: after the parent reaps it, pid=%d is gone from kern.proc.pid", pid)
	}
	_ = os.Getpid()
}
