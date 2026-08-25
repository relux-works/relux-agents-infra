package probe4

// P16 - the elected-owner window, and why `starting-unverified` is honest.
//
// Review RUN-260825-969723 finding F3: revision 3 acquired `broker.lock` at B1
// and published the owner record only at B5, after full profile composition,
// predecessor reclamation and port preflight. A broker frozen anywhere in that
// interval held the lock with no record, and `stop --force` called that state
// impossible and refused. The impossibility claim was false.
//
// Revision 4 does two things, and this probe tests both:
//
//   P16.A  The window is narrowed to ONE atomic write: the broker publishes its
//          elected-owner record immediately after winning the election, before
//          the session-leader-independent work. Everything expensive now happens
//          AFTER the record exists.
//   P16.B  The residual window is REPORTABLE. A frozen holder is `SSTOP` in the
//          kernel process table, so `status` can say "a stopped candidate holds
//          this lock" instead of "absent", and instead of guessing a pid to kill.
//   P16.C  Reporting discriminates. A lock holder whose argv is not a broker
//          invocation for this runtime key is NOT listed as a candidate, so the
//          candidate set is evidence rather than a process listing.
//   P16.D  The candidate set can hold MORE THAN ONE element while the lock is
//          held exactly once. This is why section 10.2 still refuses to signal:
//          a loser frozen between the election and its own exit is
//          indistinguishable from the winner.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func init() {
	helpers["p16_broker"] = p16Broker
	helpers["p16_foreign"] = p16Foreign
}

// p16Broker is the revision-4 broker prologue: B1 session-leader gate (skipped
// here, P9 already covers it), B2 election, B3 read predecessor, B4 publish.
func p16Broker() {
	dir := os.Getenv("P4_DIR")
	lock := filepath.Join(dir, "broker.lock")
	state := filepath.Join(dir, "broker-state.json")

	fd, err := unix.Open(lock, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		os.Exit(31)
	}
	if os.Getenv("P4_STOP_BEFORE_ELECT") == "1" {
		_ = os.WriteFile(filepath.Join(dir, "preelect."+strconv.Itoa(os.Getpid())), []byte("1"), 0o600)
		_ = unix.Kill(os.Getpid(), unix.SIGSTOP)
	}
	// B2 - election.
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		os.Exit(9) // EXIT_ELECTION_LOST
	}
	_ = os.WriteFile(filepath.Join(dir, "elected."+strconv.Itoa(os.Getpid())), []byte("1"), 0o600)

	if os.Getenv("P4_STOP_AFTER_ELECT") == "1" {
		// Freeze inside the B2->B4 window, exactly the F3 shape.
		_ = unix.Kill(os.Getpid(), unix.SIGSTOP)
	}

	// B4 - publish the elected-owner record. One atomic write.
	tmp := state + ".tmp"
	_ = os.WriteFile(tmp, []byte(`{"stage":"elected","broker_pid":`+strconv.Itoa(os.Getpid())+`}`), 0o600)
	_ = os.Rename(tmp, state)

	blockForever() // hold the lock until killed
}

// p16Foreign holds the same lock with an argv that is not a broker invocation.
func p16Foreign() {
	dir := os.Getenv("P4_DIR")
	fd, err := unix.Open(filepath.Join(dir, "broker.lock"), unix.O_RDWR|unix.O_CREAT|unix.O_CLOEXEC, 0o600)
	if err != nil {
		os.Exit(31)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		os.Exit(9)
	}
	_ = os.WriteFile(filepath.Join(dir, "foreign.up"), []byte("1"), 0o600)
	blockForever()
}

func selfExeIdentity(t *testing.T) (dev uint64, ino uint64) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var st unix.Stat_t
	if err := unix.Stat(exe, &st); err != nil {
		t.Fatal(err)
	}
	return uint64(st.Dev), st.Ino
}

func lockHeld(t *testing.T, lock string) bool {
	t.Helper()
	fd, err := unix.Open(lock, unix.O_RDWR|unix.O_CREAT|unix.O_CLOEXEC, 0o600)
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	defer unix.Close(fd)
	err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
	if err == unix.EWOULDBLOCK {
		return true // EWOULDBLOCK: some open file description holds it
	}
	if err != nil {
		t.Fatalf("flock: %v", err)
	}
	_ = unix.Flock(fd, unix.LOCK_UN)
	return false
}

func startHelper(t *testing.T, name, dir string, extraEnv ...string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-p4-helper="+name)
	cmd.Env = append(os.Environ(), helperEnv+"="+name, "P4_DIR="+dir)
	cmd.Env = append(cmd.Env, extraEnv...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(unix.SIGCONT)
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	return cmd
}

func brokerArgvMatch(argv []string) bool {
	for _, a := range argv {
		if a == "-p4-helper=p16_broker" {
			return true
		}
	}
	return false
}

func TestP16_ElectedOwnerWindow(t *testing.T) {
	dev, ino := selfExeIdentity(t)

	t.Run("A_frozen_between_election_and_publish", func(t *testing.T) {
		dir := shortTempDir(t)
		cmd := startHelper(t, "p16_broker", dir, "P4_STOP_AFTER_ELECT=1")
		pid := cmd.Process.Pid

		if !waitFile(filepath.Join(dir, "elected."+strconv.Itoa(pid)), 5*time.Second) {
			t.Fatal("helper never won the election")
		}
		if !waitStat(pid, pStatSSTOP, 5*time.Second) {
			st, err := ProcStat(pid)
			t.Fatalf("helper not SSTOP: stat=%d err=%v", st, err)
		}

		// The exact F3 state: lock held, record absent.
		if !lockHeld(t, filepath.Join(dir, "broker.lock")) {
			t.Fatal("lock not held by the frozen owner")
		}
		if exists(filepath.Join(dir, "broker-state.json")) {
			t.Fatal("record present; this subtest must observe the window")
		}

		// P16.B - the state is REPORTABLE, not absent.
		st, err := ProcStat(pid)
		if err != nil || st != pStatSSTOP {
			t.Fatalf("kernel cannot report the frozen holder: stat=%d err=%v", st, err)
		}
		cands, err := CandidateHolders(dev, ino, brokerArgvMatch)
		if err != nil {
			t.Fatalf("candidate enumeration failed: %v", err)
		}
		found := false
		for _, c := range cands {
			if c.Pid == pid {
				found = true
			}
		}
		if !found {
			t.Fatalf("frozen holder pid %d absent from candidate set %v", pid, cands)
		}
		t.Logf("P16.A/B: lock held, record absent, holder pid=%d p_stat=SSTOP, candidates=%d", pid, len(cands))

		// Bounded recovery: the re-poll in section 10.2 resolves on SIGCONT.
		if err := cmd.Process.Signal(unix.SIGCONT); err != nil {
			t.Fatal(err)
		}
		if !waitFile(filepath.Join(dir, "broker-state.json"), 5*time.Second) {
			t.Fatal("record never appeared after SIGCONT")
		}
		if !lockHeld(t, filepath.Join(dir, "broker.lock")) {
			t.Fatal("lock released across SIGCONT")
		}
		t.Log("P16.A: after SIGCONT the record appears and the lock is still held - the operator re-poll resolves")
	})

	t.Run("B_control_running_owner_publishes", func(t *testing.T) {
		dir := shortTempDir(t)
		cmd := startHelper(t, "p16_broker", dir)
		pid := cmd.Process.Pid
		if !waitFile(filepath.Join(dir, "broker-state.json"), 5*time.Second) {
			t.Fatal("unstopped owner never published")
		}
		st, err := ProcStat(pid)
		if err != nil {
			t.Fatal(err)
		}
		if st == pStatSSTOP {
			t.Fatal("control owner reported SSTOP; the SSTOP signal would not discriminate")
		}
		t.Logf("P16.B control: running owner p_stat=%d (not SSTOP), record published", st)
	})

	t.Run("C_foreign_lock_holder_is_not_a_candidate", func(t *testing.T) {
		dir := shortTempDir(t)
		cmd := startHelper(t, "p16_foreign", dir)
		if !waitFile(filepath.Join(dir, "foreign.up"), 5*time.Second) {
			t.Fatal("foreign holder never started")
		}
		if !lockHeld(t, filepath.Join(dir, "broker.lock")) {
			t.Fatal("foreign holder does not hold the lock")
		}
		cands, err := CandidateHolders(dev, ino, brokerArgvMatch)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cands {
			if c.Pid == cmd.Process.Pid {
				t.Fatalf("foreign holder %d listed as a broker candidate: argv=%v", c.Pid, c.Argv)
			}
		}
		t.Logf("P16.C: lock held by pid %d, candidate set excludes it - reporting discriminates", cmd.Process.Pid)
	})

	t.Run("D_candidate_set_can_exceed_one_so_signalling_stays_refused", func(t *testing.T) {
		dir := shortTempDir(t)
		winner := startHelper(t, "p16_broker", dir, "P4_STOP_AFTER_ELECT=1")
		if !waitStat(winner.Process.Pid, pStatSSTOP, 5*time.Second) {
			t.Fatal("winner never froze")
		}
		// A second candidate that will LOSE the election, frozen before it can
		// exit. Same uid, same binary, same argv shape as the winner.
		loser := startHelper(t, "p16_broker", dir, "P4_STOP_BEFORE_ELECT=1")
		if !waitStat(loser.Process.Pid, pStatSSTOP, 5*time.Second) {
			t.Fatal("second candidate never froze")
		}
		cands, err := CandidateHolders(dev, ino, brokerArgvMatch)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, c := range cands {
			if c.Pid == winner.Process.Pid || c.Pid == loser.Process.Pid {
				n++
			}
		}
		if n < 2 {
			t.Fatalf("expected both candidates visible, saw %d", n)
		}
		t.Logf("P16.D: lock held exactly once, %d indistinguishable candidates - `stop --force` must report, never signal", n)
	})
}
