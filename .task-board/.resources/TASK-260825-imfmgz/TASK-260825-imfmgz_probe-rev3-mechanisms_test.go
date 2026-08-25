package probe3

// Revision-3 mechanism probes P8-P11.
//
// Revision 2 was refused by an independent review on two blocking findings:
//
//   F1  a broker SIGKILLed after spawning the runtime but before publishing its
//       durable identity record leaves a live runtime that reclamation reads as
//       "nothing to reclaim";
//   F2  a hand-invoked second broker can hold descriptor 3 on the correct
//       broker.lock inode while an incumbent holds the lock, so the
//       second-descriptor gate passes on someone else's lock.
//
// Revision 3 answers both by inverting two orderings rather than by adding a
// check. The broker acquires broker.lock ITSELF (no inheritance, nothing to
// infer), and the runtime is created as an unauthorized launcher that cannot
// become the runtime until its identity record is durable. Every mechanism that
// carries either inversion is probed here before it is specified.

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const (
	exitElectionLost = 71 // broker lost the broker.lock election, no side effect
	exitUnauthorized = 72 // launcher saw EOF: its broker died before authorizing
	exitAuthTimeout  = 73 // launcher waited out its authorization deadline
	exitAuthMismatch = 74 // authorization frame did not name this launcher
)

// ---------------------------------------------------------------------------
// P8 - the broker acquires broker.lock itself; election is a kernel fact
// ---------------------------------------------------------------------------

// TestP8BrokerSelfAcquiredElection replaces revision 2's starter-holds-then-
// hands-off-the-lock design. Each broker candidate's FIRST action is
// LOCK_EX|LOCK_NB on broker.lock. There is no inherited descriptor, so there is
// no "does this fd hold the lock" question to answer, which is precisely the
// question revision 2 could not answer and F2 exploited.
func TestP8BrokerSelfAcquiredElection(t *testing.T) {
	dir := shortTempDir(t)
	lockPath := filepath.Join(dir, "broker.lock")
	sockPath := filepath.Join(dir, "b.sock")

	const racers = 8
	type outcome struct {
		pid  int
		code int
	}

	launch := func(n int) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestBrokerCandidateChild")
		cmd.Env = append(os.Environ(),
			"PROBE_CHILD=broker-candidate",
			"PROBE_LOCK="+lockPath,
			"PROBE_SOCK="+sockPath,
			"PROBE_HOLD_MS=900",
			"PROBE_TAG="+strconv.Itoa(n),
		)
		return cmd
	}

	cmds := make([]*exec.Cmd, racers)
	for i := 0; i < racers; i++ {
		cmds[i] = launch(i)
		if err := cmds[i].Start(); err != nil {
			t.Fatalf("racer %d: %v", i, err)
		}
	}

	outcomes := make([]outcome, racers)
	for i, c := range cmds {
		err := c.Wait()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("racer %d: %v", i, err)
		}
		outcomes[i] = outcome{pid: c.Process.Pid, code: code}
	}

	winners, lost := 0, 0
	for _, o := range outcomes {
		switch o.code {
		case 0:
			winners++
		case exitElectionLost:
			lost++
		default:
			t.Fatalf("racer pid=%d unexpected exit %d", o.pid, o.code)
		}
	}
	if winners != 1 {
		t.Fatalf("P8.A FAIL: %d winners of %d racers, want exactly 1", winners, racers)
	}
	if lost != racers-1 {
		t.Fatalf("P8.A FAIL: %d election_lost, want %d", lost, racers-1)
	}
	t.Logf("P8.A OK: %d racers, exactly 1 acquired broker.lock, %d exited election_lost(%d)",
		racers, lost, exitElectionLost)

	// A loser must have produced no side effect at all: the whole point of
	// moving the election into the broker is that a losing broker exits before
	// reclamation, port preflight, runtime creation, and bind.
	if exists(sockPath) {
		t.Fatalf("P8.B FAIL: a losing candidate created the rendezvous socket")
	}
	t.Logf("P8.B OK: no rendezvous socket exists after %d losing candidates", lost)

	// Promotion is free: the lock is released by the kernel when the winner
	// dies, so a later candidate simply wins. Revision 2 needed a dedicated
	// waiter-promotion path for this.
	late := launch(99)
	if err := late.Start(); err != nil {
		t.Fatal(err)
	}
	if err := late.Wait(); err != nil {
		t.Fatalf("P8.C FAIL: candidate after the winner exited did not win: %v", err)
	}
	t.Logf("P8.C OK: after the holder exited, a fresh candidate acquired the lock (promotion needs no special case)")
}

// TestP8FLockGateBypassShapeIsRefused is the reviewer's exact F2 attack replayed
// against revision 3. An incumbent broker is in `starting`: it holds the lock,
// the rendezvous socket is absent, and the runtime port is still free - the
// window in which revision 2's three-refusal composition admitted a second
// broker. The revision-3 candidate must lose on the lock alone.
func TestP8FLockGateBypassShapeIsRefused(t *testing.T) {
	dir := shortTempDir(t)
	lockPath := filepath.Join(dir, "broker.lock")
	sockPath := filepath.Join(dir, "b.sock")

	// Incumbent broker, mid-startup.
	incumbent, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer incumbent.Close()
	if err := unix.Flock(int(incumbent.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	// The port the runtime will eventually bind. Free right now, exactly as it
	// is during a real 27B model startup.
	probeLn, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probeLn.Addr().(*net.TCPAddr).Port
	probeLn.Close()

	// The attack hands the candidate a descriptor open on the CORRECT
	// broker.lock inode but not itself holding the lock. Under revision 2 this
	// passed the fstat-inode check, and the second-descriptor check returned
	// EWOULDBLOCK from the incumbent, so the gate passed on someone else's lock.
	unlockedFD3, err := os.OpenFile(lockPath, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer unlockedFD3.Close()

	cmd := exec.Command(os.Args[0], "-test.run=TestBrokerCandidateChild")
	cmd.Env = append(os.Environ(),
		"PROBE_CHILD=broker-candidate",
		"PROBE_LOCK="+lockPath,
		"PROBE_SOCK="+sockPath,
		"PROBE_PORT="+strconv.Itoa(port),
		"PROBE_HOLD_MS=200",
		"PROBE_TAG=attack",
	)
	cmd.ExtraFiles = []*os.File{unlockedFD3} // becomes descriptor 3 in the child
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	if code != exitElectionLost {
		t.Fatalf("P8.D FAIL: F2 attack shape exited %d, want election_lost(%d)\n%s",
			code, exitElectionLost, out)
	}
	if exists(sockPath) {
		t.Fatalf("P8.D FAIL: the refused candidate bound the rendezvous socket")
	}
	if ln, lerr := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port)); lerr != nil {
		t.Fatalf("P8.D FAIL: the refused candidate occupied the runtime port: %v", lerr)
	} else {
		ln.Close()
	}
	if err := unix.Flock(int(unlockedFD3.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
		t.Fatalf("P8.D FAIL: the incumbent no longer holds broker.lock")
	}
	t.Logf("P8.D OK: correct-inode/unlocked-fd3 + peer-holds-lock + free port + no socket "+
		"=> election_lost(%d), no runtime port taken, no socket bound, incumbent undisturbed",
		exitElectionLost)

	// Discriminating control: the same candidate, same code path, with no
	// incumbent, must WIN. Without this the test only proves the candidate
	// always refuses.
	incumbent.Close()
	ctl := exec.Command(os.Args[0], "-test.run=TestBrokerCandidateChild")
	ctl.Env = append(os.Environ(),
		"PROBE_CHILD=broker-candidate",
		"PROBE_LOCK="+lockPath,
		"PROBE_SOCK="+sockPath,
		"PROBE_HOLD_MS=50",
		"PROBE_TAG=control",
	)
	if err := ctl.Run(); err != nil {
		t.Fatalf("P8.E FAIL: control candidate with no incumbent did not win: %v", err)
	}
	t.Logf("P8.E OK: control - same entry point, no incumbent, acquires the lock (the gate discriminates)")
}

// TestBrokerCandidateChild models the revision-3 broker entry point: acquire
// broker.lock as the very first action, and exit election_lost on EWOULDBLOCK
// before any side effect.
func TestBrokerCandidateChild(t *testing.T) {
	if os.Getenv("PROBE_CHILD") != "broker-candidate" {
		t.Skip("child entry point")
	}
	f, err := os.OpenFile(os.Getenv("PROBE_LOCK"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		fmt.Println("lock open:", err)
		os.Exit(1)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		// Nothing has happened yet: no reclamation, no preflight, no runtime,
		// no bind. This is the whole of revision 3's answer to F2.
		fmt.Println("election_lost:", err)
		os.Exit(exitElectionLost)
	}
	// Won. Only now may a real broker touch anything.
	if sock := os.Getenv("PROBE_SOCK"); sock != "" && os.Getenv("PROBE_BIND") == "1" {
		ln, err := net.Listen("unix", sock)
		if err != nil {
			os.Exit(1)
		}
		defer ln.Close()
	}
	ms, _ := strconv.Atoi(os.Getenv("PROBE_HOLD_MS"))
	time.Sleep(time.Duration(ms) * time.Millisecond)
	os.Exit(0)
}

// ---------------------------------------------------------------------------
// P9 - the broker can prove its own detachment from the kernel
// ---------------------------------------------------------------------------

// TestP9SessionLeaderSelfGate probes the replacement for
// broker_lock_not_inherited. Revision 2's gate tried to infer a relationship
// between an inherited descriptor and a lock. This one asks the kernel a
// question about the calling process itself, and it discriminates.
func TestP9SessionLeaderSelfGate(t *testing.T) {
	run := func(name string, setsid bool, viaShell bool) (int, int) {
		var cmd *exec.Cmd
		if viaShell {
			cmd = exec.Command("/bin/sh", "-c",
				fmt.Sprintf("exec %q -test.run=TestSessionReportChild", os.Args[0]))
		} else {
			cmd = exec.Command(os.Args[0], "-test.run=TestSessionReportChild")
		}
		cmd.Env = append(os.Environ(), "PROBE_CHILD=session-report")
		if setsid {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		}
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var pid, sid int
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "SESSION ") {
				fmt.Sscanf(line, "SESSION pid=%d sid=%d", &pid, &sid)
			}
		}
		return pid, sid
	}

	pid, sid := run("setsid", true, false)
	if pid == 0 || pid != sid {
		t.Fatalf("P9.A FAIL: Setsid child pid=%d sid=%d, want equal", pid, sid)
	}
	t.Logf("P9.A OK: Setsid:true child is its own session leader (pid=%d sid=%d) => gate admits", pid, sid)

	pid, sid = run("shell", false, true)
	if pid == 0 || pid == sid {
		t.Fatalf("P9.B FAIL: shell-launched child pid=%d sid=%d, want different", pid, sid)
	}
	t.Logf("P9.B OK: hand-typed/shell-launched broker pid=%d sid=%d => gate refuses "+
		"broker_not_session_leader", pid, sid)

	pid, sid = run("direct", false, false)
	if pid == 0 || pid == sid {
		t.Fatalf("P9.C FAIL: directly forked child pid=%d sid=%d, want different", pid, sid)
	}
	t.Logf("P9.C OK: a broker forked WITHOUT Setsid also refuses (pid=%d sid=%d) - the gate "+
		"tests detachment, not provenance", pid, sid)
}

func TestSessionReportChild(t *testing.T) {
	if os.Getenv("PROBE_CHILD") != "session-report" {
		t.Skip("child entry point")
	}
	sid, err := unix.Getsid(0)
	if err != nil {
		fmt.Println("getsid:", err)
		os.Exit(1)
	}
	fmt.Printf("SESSION pid=%d sid=%d\n", os.Getpid(), sid)
	os.Exit(0)
}

// ---------------------------------------------------------------------------
// P10 - exec preserves the identity the broker published before authorizing
// ---------------------------------------------------------------------------

// TestP10ExecPreservesPidPgidStartTime is the load-bearing fact under
// publish-before-run. The broker publishes the runtime's pid, pgid and start
// time while the process is still an unauthorized launcher, and every later
// attestation and reclamation compares those recorded values against the
// runtime AFTER it has exec'd. If exec renewed the start time the record would
// be worthless.
func TestP10ExecPreservesPidPgidStartTime(t *testing.T) {
	dir := shortTempDir(t)
	authR, authW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer authW.Close()

	cmd := exec.Command(os.Args[0], "-test.run=TestLauncherChild")
	cmd.Env = append(os.Environ(),
		"PROBE_CHILD=launcher",
		"PROBE_TARGET=/bin/sleep",
		"PROBE_TARGET_ARG=13",
		"PROBE_AUTH_TIMEOUT_MS=5000",
		"PROBE_READY="+filepath.Join(dir, "ready"),
	)
	cmd.ExtraFiles = []*os.File{authR} // descriptor 3 in the child
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	authR.Close()
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = unix.Kill(-pid, unix.SIGKILL); _, _ = cmd.Process.Wait() })

	// Wait until the launcher exists and is blocked on authorization.
	deadline := time.Now().Add(3 * time.Second)
	for !exists(filepath.Join(dir, "ready")) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	before, err := Identify(pid)
	if err != nil {
		t.Fatalf("P10 FAIL: cannot identify the unauthorized launcher: %v", err)
	}
	pgidBefore, err := unix.Getpgid(pid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(before.Exe, filepath.Base(os.Args[0])) {
		t.Fatalf("P10 FAIL: pre-exec identity is %q, expected the launcher binary", before.Exe)
	}
	t.Logf("P10.A OK: before authorization pid=%d pgid=%d start=%s exe=%q (this is what the "+
		"broker publishes)", pid, pgidBefore, before.StartKey(), before.Exe)

	// Authorize. The launcher execs the real runtime in place.
	if _, err := fmt.Fprintf(authW, "AUTH pid=%d\n", pid); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(5 * time.Second)
	var after ProcIdentity
	for time.Now().Before(deadline) {
		after, err = Identify(pid)
		if err == nil && after.Exe == "/bin/sleep" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if after.Exe != "/bin/sleep" {
		t.Fatalf("P10 FAIL: launcher never became the runtime (exe=%q err=%v)", after.Exe, err)
	}
	pgidAfter, err := unix.Getpgid(pid)
	if err != nil {
		t.Fatal(err)
	}

	if after.Pid != before.Pid {
		t.Fatalf("P10 FAIL: pid changed across exec: %d -> %d", before.Pid, after.Pid)
	}
	if pgidAfter != pgidBefore {
		t.Fatalf("P10 FAIL: pgid changed across exec: %d -> %d", pgidBefore, pgidAfter)
	}
	if after.StartKey() != before.StartKey() {
		t.Fatalf("P10 FAIL: start time changed across exec: %s -> %s (publish-before-run "+
			"would be unimplementable)", before.StartKey(), after.StartKey())
	}
	if strings.Join(after.Argv, " ") == strings.Join(before.Argv, " ") {
		t.Fatalf("P10 FAIL: argv did not change across exec, so the probe did not exec")
	}
	t.Logf("P10.B OK: after exec pid=%d pgid=%d start=%s exe=%q argv=%q - pid, pgid and start "+
		"time byte-preserved, argv replaced", after.Pid, pgidAfter, after.StartKey(), after.Exe, after.Argv)
	t.Logf("P10.C OK: the record published BEFORE the runtime existed still identifies it "+
		"AFTER it exists; pre-exec argv %q and post-exec argv %q are the two admissible shapes",
		before.Argv, after.Argv)
}

// ---------------------------------------------------------------------------
// P11 - an unauthorized launcher cannot outlive its broker
// ---------------------------------------------------------------------------

// TestP11LauncherDiesOnAuthorizationEOF probes the guarantee that closes F1's
// window: if the broker dies before authorizing, the launcher never becomes the
// runtime. The write end must not leak into the launcher, or EOF never arrives.
func TestP11LauncherDiesOnAuthorizationEOF(t *testing.T) {
	dir := shortTempDir(t)
	authR, authW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestLauncherChild")
	cmd.Env = append(os.Environ(),
		"PROBE_CHILD=launcher",
		"PROBE_TARGET=/bin/sleep",
		"PROBE_TARGET_ARG=29",
		"PROBE_AUTH_TIMEOUT_MS=10000",
		"PROBE_READY="+filepath.Join(dir, "ready"),
	)
	cmd.ExtraFiles = []*os.File{authR}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	authR.Close()
	pid := cmd.Process.Pid

	deadline := time.Now().Add(3 * time.Second)
	for !exists(filepath.Join(dir, "ready")) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	id, err := Identify(pid)
	if err != nil {
		t.Fatalf("P11 FAIL: launcher not identifiable: %v", err)
	}
	if id.Exe == "/bin/sleep" {
		t.Fatalf("P11 FAIL: launcher exec'd the runtime without authorization")
	}

	// The broker dies. Its write end is the only one, so the launcher's read
	// returns EOF. This is the exact instant F1 exploited.
	authW.Close()

	err = cmd.Wait()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	if code != exitUnauthorized {
		t.Fatalf("P11.A FAIL: launcher exited %d, want unauthorized(%d) - it must not survive "+
			"its broker", code, exitUnauthorized)
	}
	if Alive(pid) {
		t.Fatalf("P11.A FAIL: launcher pid=%d still alive", pid)
	}
	t.Logf("P11.A OK: broker's write end closed => launcher exited unauthorized(%d) and never "+
		"exec'd the runtime", exitUnauthorized)
	t.Logf("P11.B OK: the authorization write end did not leak into the launcher through " +
		"ExtraFiles, or the read would never have returned EOF")
}

// TestP11LauncherRefusesForeignAuthorization proves the launcher's own gate
// discriminates rather than accepting any byte on the pipe.
func TestP11LauncherRefusesForeignAuthorization(t *testing.T) {
	dir := shortTempDir(t)
	authR, authW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer authW.Close()
	cmd := exec.Command(os.Args[0], "-test.run=TestLauncherChild")
	cmd.Env = append(os.Environ(),
		"PROBE_CHILD=launcher",
		"PROBE_TARGET=/bin/sleep",
		"PROBE_TARGET_ARG=31",
		"PROBE_AUTH_TIMEOUT_MS=5000",
		"PROBE_READY="+filepath.Join(dir, "ready"),
	)
	cmd.ExtraFiles = []*os.File{authR}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	authR.Close()
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// An authorization frame naming a different pid: the launcher must refuse
	// rather than exec. Absent this, any writer on the pipe is an authority.
	if _, err := fmt.Fprintf(authW, "AUTH pid=%d\n", pid+100000); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	}
	if code != exitAuthMismatch {
		t.Fatalf("P11.C FAIL: launcher exited %d on a frame naming another pid, want "+
			"auth_mismatch(%d)", code, exitAuthMismatch)
	}
	t.Logf("P11.C OK: authorization frame naming a foreign pid => auth_mismatch(%d), no exec",
		exitAuthMismatch)
	_ = dir
}

// TestLauncherChild models the revision-3 `runtime runtime-launch` entry point.
// It is created by the broker in the runtime's own process group and cannot
// become the runtime until the broker authorizes it, which the broker does only
// after the runtime identity record naming this pid is durable.
func TestLauncherChild(t *testing.T) {
	if os.Getenv("PROBE_CHILD") != "launcher" {
		t.Skip("child entry point")
	}
	auth := os.NewFile(3, "authorization")
	if auth == nil {
		os.Exit(1)
	}
	if ready := os.Getenv("PROBE_READY"); ready != "" {
		_ = os.WriteFile(ready, []byte(strconv.Itoa(os.Getpid())), 0o600)
	}

	type readResult struct {
		n   int
		err error
		buf []byte
	}
	ch := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 512)
		n, err := auth.Read(buf)
		ch <- readResult{n: n, err: err, buf: buf}
	}()

	ms, _ := strconv.Atoi(os.Getenv("PROBE_AUTH_TIMEOUT_MS"))
	select {
	case r := <-ch:
		if r.n == 0 {
			// EOF: the broker died before authorizing. No runtime may exist.
			os.Exit(exitUnauthorized)
		}
		var want int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(r.buf[:r.n])), "AUTH pid=%d", &want); err != nil {
			os.Exit(exitAuthMismatch)
		}
		if want != os.Getpid() {
			os.Exit(exitAuthMismatch)
		}
		target := os.Getenv("PROBE_TARGET")
		argv := []string{target, os.Getenv("PROBE_TARGET_ARG")}
		// exec in place: pid, pgid and start time survive, argv is replaced.
		if err := syscall.Exec(target, argv, []string{}); err != nil {
			os.Exit(1)
		}
	case <-time.After(time.Duration(ms) * time.Millisecond):
		os.Exit(exitAuthTimeout)
	}
}
