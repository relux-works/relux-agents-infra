package probe4

// P20 - the two shapes review finding F4 said test 12.2.22 had conflated.
//
// Revision 3's test 22 started "a second broker holding a legitimately
// inherited and held lock descriptor" and expected it to reach the port
// preflight and the rendezvous bind. Revision 3 inherits no lock descriptor and
// its election exits before either point, so the test could not assert the
// production path at all. F4 asked for two separate shapes:
//
//   P20.A  A second broker against a serving incumbent loses the ELECTION and
//          leaves the incumbent untouched. It never reaches preflight or bind.
//          (The election itself is P8.A/P8.D; what P20.A adds is the assertion
//          that the incumbent's bound socket INODE is byte-identical afterwards.)
//   P20.B  A contender that appears AFTER the winner's stale-inode cleanup and
//          BEFORE its bind. The winner must exit broker_rendezvous_bind_conflict
//          and must NOT unlink the contender. Asserted on dev/ino, not on
//          existence: an unlink-and-rebind would leave a socket at the path too.
//   P20.C  Control - with the path free, the same bind succeeds, so P20.B is not
//          passing because bind always fails.

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func inode(t *testing.T, path string) (uint64, uint64, bool) {
	t.Helper()
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return 0, 0, false
	}
	return uint64(st.Dev), st.Ino, true
}

// p20BrokerTail is the revision-4 broker from the election through the bind,
// with the stale-inode cleanup and the bind as separate observable steps.
func p20BrokerTail(t *testing.T, dir string, betweenCleanupAndBind func()) (exitCode int, ln net.Listener) {
	t.Helper()
	lockPath := filepath.Join(dir, "broker.lock")
	sock := filepath.Join(dir, "b.sock")

	fd, err := unix.Open(lockPath, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unix.Close(fd) })
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return 9, nil // EXIT_ELECTION_LOST, before any side effect
	}

	// B6 tail - only a broker holding the lock may unlink a stale inode, and it
	// does so HERE, never at bind time.
	if _, _, ok := inode(t, sock); ok {
		if c, derr := net.Dial("unix", sock); derr != nil {
			_ = os.Remove(sock) // stale: nobody is accepting
		} else {
			_ = c.Close()
		}
	}

	if betweenCleanupAndBind != nil {
		betweenCleanupAndBind()
	}

	// B15 - bind, fail closed, never unlink.
	l, berr := net.Listen("unix", sock)
	if berr != nil {
		return 44, nil // broker_rendezvous_bind_conflict
	}
	return 0, l
}

func TestP20_BindConflictNeverDisplacesAnIncumbent(t *testing.T) {
	t.Run("A_second_broker_loses_the_election_and_never_reaches_bind", func(t *testing.T) {
		dir := shortTempDir(t)
		incumbent := startHelper(t, "p19_broker", dir, "P4_READY_AFTER=10ms")
		if !waitFile(filepath.Join(dir, "serving"), 5*time.Second) {
			t.Fatal("incumbent never bound")
		}
		// The incumbent holds broker.lock through a helper of its own; emulate
		// that by taking the lock here in a separate process-held descriptor.
		holder := startHelper(t, "p16_broker", dir)
		if !waitFile(filepath.Join(dir, "broker-state.json"), 5*time.Second) {
			t.Fatal("lock holder never published")
		}
		dev0, ino0, ok := inode(t, filepath.Join(dir, "b.sock"))
		if !ok {
			t.Fatal("incumbent socket missing")
		}

		code, ln := p20BrokerTail(t, dir, nil)
		if ln != nil {
			ln.Close()
		}
		if code != 9 {
			t.Fatalf("second broker exit = %d, want EXIT_ELECTION_LOST (9)", code)
		}
		dev1, ino1, ok := inode(t, filepath.Join(dir, "b.sock"))
		if !ok || dev0 != dev1 || ino0 != ino1 {
			t.Fatalf("incumbent socket inode changed: %d/%d -> %d/%d ok=%v", dev0, ino0, dev1, ino1, ok)
		}
		if !Alive(holder.Process.Pid) || !Alive(incumbent.Process.Pid) {
			t.Fatal("incumbent processes disturbed")
		}
		t.Logf("P20.A: election lost before cleanup or bind; incumbent socket inode %d/%d unchanged", dev1, ino1)
	})

	t.Run("B_contender_after_cleanup_before_bind_is_never_unlinked", func(t *testing.T) {
		dir := shortTempDir(t)
		sock := filepath.Join(dir, "b.sock")
		var devC, inoC uint64
		var contender net.Listener

		code, ln := p20BrokerTail(t, dir, func() {
			// A same-uid process binds the path in the window the winner
			// believed was its own.
			var err error
			contender, err = net.Listen("unix", sock)
			if err != nil {
				t.Fatal(err)
			}
			devC, inoC, _ = inode(t, sock)
		})
		if ln != nil {
			ln.Close()
		}
		if contender != nil {
			defer contender.Close()
		}
		if code != 44 {
			t.Fatalf("winner exit = %d, want broker_rendezvous_bind_conflict (44)", code)
		}
		dev1, ino1, ok := inode(t, sock)
		if !ok {
			t.Fatal("contender socket was unlinked")
		}
		if dev1 != devC || ino1 != inoC {
			t.Fatalf("contender inode replaced: %d/%d -> %d/%d - the winner unlinked and rebound", devC, inoC, dev1, ino1)
		}
		t.Logf("P20.B: bind conflict refused, contender inode %d/%d byte-identical - asserted on the inode, not on existence", dev1, ino1)
	})

	t.Run("C_control_free_path_binds", func(t *testing.T) {
		dir := shortTempDir(t)
		code, ln := p20BrokerTail(t, dir, nil)
		if code != 0 || ln == nil {
			t.Fatalf("control bind failed: exit = %d", code)
		}
		defer ln.Close()
		t.Log("P20.C control: the same tail binds when the path is free - P20.B is a distinction, not a constant refusal")
	})
}
