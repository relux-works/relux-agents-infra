package probe

// Revision-2 probes for TASK-260825-imfmgz.
//
// Each probe corresponds to a mechanism that the revised specification states
// normatively. Nothing here is assumed; the raw output is the evidence.
//
//   P1  second-descriptor flock detects a held lock (F2 mechanism)
//   P2  inherited-fd fstat identity + in-child self-inspection (F2 mechanism)
//   P3  AF_UNIX bind onto an existing socket inode (rendezvous bind conflict)
//   P4  connect() to a socket inode whose listener is gone (F1 loop input)
//   P5  rename-over upgrade: stat(path) resolves to the NEW inode (F4)
//   P6  setsid child stays a child; wait4(WNOHANG) sees its exit (F1 step c)

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func errStr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

// ---------------------------------------------------------------- P1

func TestP1SecondDescriptorDetectsHeldFlock(t *testing.T) {
	dir := t.TempDir()
	held := filepath.Join(dir, "broker.lock")
	unheld := filepath.Join(dir, "other.lock")

	f1, err := os.OpenFile(held, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()
	if err := unix.Flock(int(f1.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("primary acquire: %v", err)
	}
	t.Logf("P1.0 primary LOCK_EX|LOCK_NB on fd1: err=%s (held from here on)", errStr(nil))

	// A: the naive check the ORIGINAL spec implied.
	errA := unix.Flock(int(f1.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	t.Logf("P1.A re-lock the SAME inherited fd while held: err=%s  => %s",
		errStr(errA), verdict(errA == nil, "ALWAYS PASSES - gate is inert", "distinguishes"))

	// B: the mechanism the revised spec states.
	f2, err := os.OpenFile(held, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	errB := unix.Flock(int(f2.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	t.Logf("P1.B SECOND descriptor, same process, lock IS held: err=%s  => %s",
		errStr(errB), verdict(errB == unix.EWOULDBLOCK, "EWOULDBLOCK - held is detectable", "UNEXPECTED"))

	// C: control - the negative case must be distinguishable.
	f3, err := os.OpenFile(unheld, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f3.Close()
	errC := unix.Flock(int(f3.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	t.Logf("P1.C CONTROL second descriptor, lock NOT held: err=%s  => %s",
		errStr(errC), verdict(errC == nil, "acquires - unheld is distinguishable", "UNEXPECTED"))

	// D: closing the probing descriptor must not disturb the real lock.
	f2.Close()
	f4, err := os.OpenFile(held, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f4.Close()
	errD := unix.Flock(int(f4.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	t.Logf("P1.D after closing the probing fd, re-probe: err=%s  => %s",
		errStr(errD), verdict(errD == unix.EWOULDBLOCK, "lock survived the probe", "PROBE RELEASED THE LOCK"))

	if errA != nil || errB != unix.EWOULDBLOCK || errC != nil || errD != unix.EWOULDBLOCK {
		t.Fatalf("P1 unexpected: A=%v B=%v C=%v D=%v", errA, errB, errC, errD)
	}
}

func verdict(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

// ---------------------------------------------------------------- P2
// The child re-execs this same test binary with PROBE_CHILD=lockcheck and an
// inherited descriptor at index 3, exactly as the spec's ExtraFiles handoff.

func TestP2InheritedFdIdentityAndSelfInspection(t *testing.T) {
	if os.Getenv("PROBE_CHILD") != "" {
		return
	}
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "broker.lock")
	decoyPath := filepath.Join(dir, "decoy.lock")

	for _, c := range []struct {
		name   string
		locked bool
		fd     string // "lock", "decoy", "none"
		expect string
	}{
		{"held-and-correct-fd", true, "lock", "OK"},
		{"UNLOCKED-fd (ad-hoc invocation)", false, "lock", "broker_lock_not_inherited"},
		{"fd points at a DIFFERENT file", true, "decoy", "broker_lock_not_inherited"},
		{"no descriptor inherited", true, "none", "broker_lock_not_inherited"},
	} {
		f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if c.locked {
			if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
				t.Fatalf("%s: parent lock: %v", c.name, err)
			}
		}
		d, err := os.OpenFile(decoyPath, os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestP2Child", "-test.v")
		cmd.Env = append(os.Environ(), "PROBE_CHILD=lockcheck", "PROBE_LOCK_PATH="+lockPath)
		switch c.fd {
		case "lock":
			cmd.ExtraFiles = []*os.File{f}
		case "decoy":
			cmd.ExtraFiles = []*os.File{d}
		}
		out, _ := cmd.CombinedOutput()
		got := grepMarker(string(out), "CHILD-RESULT=")
		t.Logf("P2 %-34s -> %-28s %s", c.name, got,
			verdict(got == c.expect, "as specified", "MISMATCH, expected "+c.expect))
		if got != c.expect {
			t.Fatalf("P2 %s: got %q want %q\n%s", c.name, got, c.expect, out)
		}
		f.Close()
		d.Close()
	}
}

func TestP2Child(t *testing.T) {
	if os.Getenv("PROBE_CHILD") != "lockcheck" {
		t.Skip("parent-driven only")
	}
	fmt.Printf("CHILD-RESULT=%s\n", childLockGate(os.Getenv("PROBE_LOCK_PATH")))
}

// childLockGate is the revised spec's broker_lock_not_inherited gate, verbatim
// in shape: (1) fstat the inherited descriptor and require dev/ino equality
// with a fresh open of broker.lock, (2) prove the lock is held via a SECOND
// descriptor.
func childLockGate(lockPath string) string {
	inherited := os.NewFile(3, "broker.lock")
	if inherited == nil {
		return "broker_lock_not_inherited"
	}
	var st unix.Stat_t
	if err := unix.Fstat(3, &st); err != nil {
		return "broker_lock_not_inherited"
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		return "broker_lock_not_inherited"
	}
	var want unix.Stat_t
	if err := unix.Stat(lockPath, &want); err != nil {
		return "shared_runtime_state_path_invalid"
	}
	if st.Dev != want.Dev || st.Ino != want.Ino {
		return "broker_lock_not_inherited"
	}
	probe, err := os.OpenFile(lockPath, os.O_RDWR, 0o600)
	if err != nil {
		return "shared_runtime_state_path_invalid"
	}
	defer probe.Close()
	switch err := unix.Flock(int(probe.Fd()), unix.LOCK_EX|unix.LOCK_NB); err {
	case unix.EWOULDBLOCK:
		return "OK"
	case nil:
		_ = unix.Flock(int(probe.Fd()), unix.LOCK_UN)
		return "broker_lock_not_inherited"
	default:
		return "shared_runtime_state_path_invalid"
	}
}

func grepMarker(s, prefix string) string {
	for _, line := range splitLines(s) {
		if len(line) > len(prefix) && line[:len(prefix)] == prefix {
			return line[len(prefix):]
		}
	}
	return "<no marker>"
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, trimCR(cur))
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, trimCR(cur))
}

func trimCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}

// ---------------------------------------------------------------- P3

func TestP3BindOntoExistingSocketInode(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "b.sock")

	l1, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l1.Close()

	_, errLive := net.Listen("unix", sock)
	t.Logf("P3.A bind onto a LIVE listener's path: err=%s => %s", errStr(errLive),
		verdict(errLive != nil, "refused - a second broker cannot displace a live one", "ADMITTED"))

	// Leave a STALE inode behind: Go unlinks on close by default, so opt out.
	l1.(*net.UnixListener).SetUnlinkOnClose(false)
	l1.Close()
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("stale socket inode not left behind: %v", err)
	}
	_, errStale := net.Listen("unix", sock)
	t.Logf("P3.B bind onto a STALE socket inode: err=%s => %s", errStr(errStale),
		verdict(errStale != nil, "refused - stale inode must be unlinked under the lock first", "ADMITTED"))

	if errLive == nil || errStale == nil {
		t.Fatal("P3: bind must never silently take over an existing socket path")
	}
}

// ---------------------------------------------------------------- P4

func TestP4ConnectClassification(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "b.sock")

	_, errAbsent := net.Dial("unix", sock)
	t.Logf("P4.A connect, no inode at all: err=%s => %s", errStr(errAbsent),
		verdict(isErrno(errAbsent, unix.ENOENT), "ENOENT - 'no broker yet'", "UNEXPECTED"))

	lDead, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	lDead.(*net.UnixListener).SetUnlinkOnClose(false)
	lDead.Close()
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("stale socket inode not left behind: %v", err)
	}
	_, errDead := net.Dial("unix", sock)
	t.Logf("P4.B connect, inode present but nobody listening: err=%s => %s", errStr(errDead),
		verdict(isErrno(errDead, unix.ECONNREFUSED), "ECONNREFUSED - 'stale inode, reclaim'", "UNEXPECTED"))

	os.Remove(sock)
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	c, errLive := net.Dial("unix", sock)
	if errLive == nil {
		c.Close()
	}
	t.Logf("P4.C connect to a live listener: err=%s => %s", errStr(errLive),
		verdict(errLive == nil, "connects", "UNEXPECTED"))

	if !isErrno(errAbsent, unix.ENOENT) || !isErrno(errDead, unix.ECONNREFUSED) || errLive != nil {
		t.Fatal("P4: connect classification is load-bearing for the section 6 wait loop")
	}
}

func isErrno(err error, want unix.Errno) bool {
	if err == nil {
		return false
	}
	type unwrapper interface{ Unwrap() error }
	for e := err; e != nil; {
		if errno, ok := e.(syscall.Errno); ok {
			return errno == syscall.Errno(want)
		}
		u, ok := e.(unwrapper)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// ---------------------------------------------------------------- P5
// F4: does gate 3 (stat the peer's recorded exec PATH) survive a rename-over
// in-place upgrade? It must not, and the probe must say so out loud.

func TestP5RenameOverUpgradeDefeatsPathStat(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "agents-infra")
	if err := os.WriteFile(installed, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var oldSt unix.Stat_t
	if err := unix.Stat(installed, &oldSt); err != nil {
		t.Fatal(err)
	}

	// A running broker recorded its own binary identity at startup.
	recordedDev, recordedIno := oldSt.Dev, oldSt.Ino

	// Operator upgrades in place: write a new file, rename over the same path.
	staged := filepath.Join(dir, ".agents-infra.new")
	if err := os.WriteFile(staged, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(staged, installed); err != nil {
		t.Fatal(err)
	}

	var newSt unix.Stat_t
	if err := unix.Stat(installed, &newSt); err != nil {
		t.Fatal(err)
	}

	gate3Passes := oldStPathNow(installed) == statKey(newSt) // both sides stat the same path
	t.Logf("P5.A old inode=%d:%d  new inode=%d:%d  (rename-over changed the inode: %v)",
		recordedDev, recordedIno, newSt.Dev, newSt.Ino, recordedIno != newSt.Ino)
	t.Logf("P5.B GATE 3 (stat peer's recorded exec PATH vs client stats its own PATH): passes=%v => %s",
		gate3Passes, verdict(gate3Passes, "STALE BROKER ADMITTED - review finding F4 confirmed", "refused"))

	// The revised gate 3b: the broker announces the identity it recorded AT ITS
	// OWN STARTUP; the client compares that against its own current binary.
	gate3bPasses := recordedIno == newSt.Ino && recordedDev == newSt.Dev
	t.Logf("P5.C GATE 3b (broker's START-TIME-RECORDED inode vs client's current inode): passes=%v => %s",
		gate3bPasses, verdict(!gate3bPasses, "stale broker REFUSED - gate 3b closes F4", "still admitted"))

	if !gate3Passes {
		t.Fatal("P5: expected gate 3 to pass under rename-over; F4's premise would be wrong")
	}
	if gate3bPasses {
		t.Fatal("P5: gate 3b must refuse a broker running the pre-upgrade inode")
	}
}

func oldStPathNow(p string) string {
	var st unix.Stat_t
	if err := unix.Stat(p, &st); err != nil {
		return "err"
	}
	return statKey(st)
}

func statKey(st unix.Stat_t) string { return fmt.Sprintf("%d:%d", st.Dev, st.Ino) }

// ---------------------------------------------------------------- P6
// F1 step (c) + the "reparented to init within milliseconds" claim.

func TestP6SetsidChildRemainsReapableChild(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 7")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Logf("P6.0 started setsid child pid=%d (our pid=%d)", pid, os.Getpid())

	var ws unix.WaitStatus
	var seen bool
	var code int
	deadline := time.Now().Add(3 * time.Second)
	polls := 0
	for time.Now().Before(deadline) {
		polls++
		wpid, err := unix.Wait4(pid, &ws, unix.WNOHANG, nil)
		if err != nil {
			t.Fatalf("wait4: %v", err)
		}
		if wpid == pid {
			seen, code = true, ws.ExitStatus()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("P6.A wait4(WNOHANG) after Setsid: reaped=%v exit=%d polls=%d => %s",
		seen, code, polls, verdict(seen && code == 7,
			"a setsid child is STILL our child - startup failure is detectable, and 'reparented to init immediately' is FALSE",
			"NOT REAPABLE"))

	// Session id: setsid did detach the session, which is what terminal
	// independence actually rests on.
	cmd2 := exec.Command("/bin/sleep", "31")
	cmd2.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd2.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd2.Process.Kill(); _ = cmd2.Wait() }()
	sid, err := unix.Getsid(cmd2.Process.Pid)
	ownSid, _ := unix.Getsid(os.Getpid())
	t.Logf("P6.B child sid=%d (err=%s) our sid=%d => %s", sid, errStr(err), ownSid,
		verdict(err == nil && sid != ownSid && sid == cmd2.Process.Pid,
			"child is its own session leader - terminal independence holds", "UNEXPECTED"))
	ppid := parentOf(cmd2.Process.Pid)
	t.Logf("P6.C child ppid=%d, our pid=%d => %s", ppid, os.Getpid(),
		verdict(ppid == os.Getpid(),
			"still parented to the starter until the starter exits", "already reparented"))

	if !seen || code != 7 {
		t.Fatal("P6: the starter must be able to detect its broker's startup failure")
	}
	if err != nil || sid != cmd2.Process.Pid {
		t.Fatal("P6: setsid must make the child a session leader")
	}
	if ppid != os.Getpid() {
		t.Fatal("P6: expected the setsid child to remain our child")
	}
}

func parentOf(pid int) int {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return -1
	}
	return int(kp.Eproc.Ppid)
}
