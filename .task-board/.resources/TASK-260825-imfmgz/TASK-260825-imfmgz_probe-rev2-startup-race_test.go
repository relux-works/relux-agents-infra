package probe

// P7 - review finding F1, made executable.
//
// Models section 6's startup algorithm for a SECOND client that arrives while
// the first client's broker is still bringing the runtime up. The broker holds
// broker.lock for its whole serving life and binds the rendezvous socket only
// after the runtime is ready.
//
//   OLD (spec revision 1): connect -> ENOENT -> LOCK_EX *blocking*
//   NEW (spec revision 2): bounded loop racing a connect poll against LOCK_EX|LOCK_NB
//
// The old algorithm must be shown to fail, or the fix is unjustified.

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type brokerSim struct {
	lockFile  *os.File
	listener  net.Listener
	listenErr chan error
}

// shortTempDir keeps the rendezvous path inside sun_path. t.TempDir() embeds the
// full subtest name and overflows it - which is exactly the limit section 5.2
// documents, observed here for real.
func shortTempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "p7")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// startBrokerSim takes broker.lock and holds it for its whole life, binding the
// rendezvous socket only after readyAfter elapses. This is section 3.1's rule:
// "the socket is bound only after the runtime is ready and attested".
func startBrokerSim(t *testing.T, lockPath, sockPath string, readyAfter time.Duration) *brokerSim {
	t.Helper()
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("broker could not take broker.lock: %v", err)
	}
	b := &brokerSim{lockFile: f, listenErr: make(chan error, 1)}
	go func() {
		time.Sleep(readyAfter)
		l, err := net.Listen("unix", sockPath)
		if err != nil {
			b.listenErr <- err
			return
		}
		b.listener = l
		close(b.listenErr)
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	return b
}

func (b *brokerSim) stop() {
	if b.listener != nil {
		b.listener.Close()
	}
	b.lockFile.Close() // releases broker.lock, as a dying broker does
}

// clientOld is section 6 steps 4-5 as written in revision 1.
func clientOld(lockPath, sockPath string, deadline time.Duration) (string, time.Duration) {
	start := time.Now()

	// Step 4: attach attempt.
	if c, err := net.Dial("unix", sockPath); err == nil {
		c.Close()
		return "attached", time.Since(start)
	}

	// Step 5: LOCK_EX with a deadline, BLOCKING. There is no path back to step 4.
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return "state_path_invalid", time.Since(start)
	}

	done := make(chan error, 1)
	go func() { done <- unix.Flock(int(f.Fd()), unix.LOCK_EX) }()
	select {
	case err := <-done:
		f.Close()
		if err != nil {
			return "state_path_invalid", time.Since(start)
		}
		return "became_starter", time.Since(start)
	case <-time.After(deadline):
		// NOTE: f is deliberately NOT closed here. close(2) on a descriptor that
		// another thread is blocked in flock(LOCK_EX) on BLOCKS on darwin until
		// the flock returns - verified by a 25s go test timeout whose stack shows
		// goroutine 8 parked in syscall.Close. A client that abandons a blocking
		// flock therefore cannot even release its descriptor cleanly; it can only
		// exit the process. This is a second, independent reason the revised
		// section 6 uses LOCK_EX|LOCK_NB in a poll loop rather than a blocking
		// acquire with a deadline.
		return "broker_start_timeout", time.Since(start)
	}
}

// clientNew is section 6's revised wait loop: one bounded loop that races the
// connect poll against a NON-BLOCKING lock attempt.
func clientNew(lockPath, sockPath string, deadline time.Duration) (string, time.Duration) {
	start := time.Now()
	end := start.Add(deadline)
	backoff := 10 * time.Millisecond
	const backoffCap = 200 * time.Millisecond
	starter := false
	sawPeerHoldingLock := false

	for {
		// (a) attach attempt, every iteration - not just the first.
		if c, err := net.Dial("unix", sockPath); err == nil {
			c.Close()
			if starter {
				return "attached_as_starter", time.Since(start)
			}
			return "attached", time.Since(start)
		}

		// (b) single-flight, non-blocking, only while we are not already the starter.
		if !starter {
			f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
			if err != nil {
				return "state_path_invalid", time.Since(start)
			}
			switch err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err {
			case nil:
				// We are the starter: reclaim, fork the broker, keep looping.
				starter = true
				// (the fd stays open; the real starter hands it to the broker)
			case unix.EWOULDBLOCK:
				// A peer's broker exists or is starting. Keep polling connect.
				sawPeerHoldingLock = true
				f.Close()
			default:
				f.Close()
				return "state_path_invalid", time.Since(start)
			}
		}

		if !time.Now().Before(end) {
			if starter {
				return "broker_start_timeout", time.Since(start)
			}
			if sawPeerHoldingLock {
				return "shared_runtime_peer_start_timeout", time.Since(start)
			}
			return "broker_start_timeout", time.Since(start)
		}
		time.Sleep(backoff)
		if backoff < backoffCap {
			backoff = backoff * 3 / 2
		}
	}
}

func TestP7ConcurrentSecondClientDuringBrokerStartup(t *testing.T) {
	const runtimeStartup = 700 * time.Millisecond
	const clientDeadline = 3 * time.Second

	t.Run("OLD algorithm starves the second client", func(t *testing.T) {
		dir := shortTempDir(t)
		lock, sock := filepath.Join(dir, "broker.lock"), filepath.Join(dir, "b.sock")
		b := startBrokerSim(t, lock, sock, runtimeStartup)
		defer b.stop()

		outcome, took := clientOld(lock, sock, clientDeadline)
		t.Logf("P7.A OLD: outcome=%-22s after=%v => %s", outcome, took.Round(10*time.Millisecond),
			verdict(outcome == "broker_start_timeout",
				"F1 CONFIRMED - a healthy broker was serving and the client still failed",
				"did not reproduce"))
		if outcome != "broker_start_timeout" {
			t.Fatalf("F1 did not reproduce: got %q", outcome)
		}
		if c, err := net.Dial("unix", sock); err == nil {
			c.Close()
			t.Logf("P7.B the broker WAS connectable at the moment the client gave up")
		} else {
			t.Fatalf("broker never became connectable, test is invalid: %v", err)
		}
	})

	t.Run("NEW algorithm attaches to the peer's broker", func(t *testing.T) {
		dir := shortTempDir(t)
		lock, sock := filepath.Join(dir, "broker.lock"), filepath.Join(dir, "b.sock")
		b := startBrokerSim(t, lock, sock, runtimeStartup)
		defer b.stop()

		outcome, took := clientNew(lock, sock, clientDeadline)
		if err := <-b.listenErr; err != nil {
			t.Fatalf("broker sim never bound, test is invalid: %v", err)
		}
		t.Logf("P7.C NEW: outcome=%-22s after=%v => %s", outcome, took.Round(10*time.Millisecond),
			verdict(outcome == "attached", "attaches once the peer's broker binds", "REGRESSION"))
		if outcome != "attached" {
			t.Fatalf("revised loop failed: %q", outcome)
		}
		if took < runtimeStartup {
			t.Fatalf("attached before the broker could have bound: %v", took)
		}
	})

	t.Run("NEW algorithm still elects exactly one starter", func(t *testing.T) {
		dir := shortTempDir(t)
		lock, sock := filepath.Join(dir, "broker.lock"), filepath.Join(dir, "b.sock")
		// No broker at all: N clients race; exactly one must win the lock.
		const n = 8
		type res struct {
			outcome string
		}
		out := make(chan res, n)
		for i := 0; i < n; i++ {
			go func() {
				o, _ := clientNew(lock, sock, 400*time.Millisecond)
				out <- res{o}
			}()
		}
		starters, peers := 0, 0
		for i := 0; i < n; i++ {
			r := <-out
			switch r.outcome {
			case "broker_start_timeout":
				starters++ // won the lock, no broker was actually forked in the model
			case "shared_runtime_peer_start_timeout":
				peers++
			default:
				t.Fatalf("unexpected outcome %q", r.outcome)
			}
		}
		t.Logf("P7.D NEW: %d client(s) became starter, %d waited on the peer => %s",
			starters, peers, verdict(starters == 1 && peers == n-1,
				"single-flight preserved, and the losers are typed distinctly", "SINGLE-FLIGHT BROKEN"))
		if starters != 1 || peers != n-1 {
			t.Fatalf("single-flight broken: starters=%d peers=%d", starters, peers)
		}
	})

	t.Run("NEW algorithm promotes a waiter when the starter's broker dies", func(t *testing.T) {
		dir := shortTempDir(t)
		lock, sock := filepath.Join(dir, "broker.lock"), filepath.Join(dir, "b.sock")
		// A broker takes the lock and never binds, then dies at 300ms.
		b := startBrokerSim(t, lock, sock, time.Hour)
		go func() { time.Sleep(300 * time.Millisecond); b.stop() }()

		outcome, took := clientNew(lock, sock, 3*time.Second)
		t.Logf("P7.E NEW: outcome=%-22s after=%v => %s", outcome, took.Round(10*time.Millisecond),
			verdict(outcome == "broker_start_timeout" && took > 300*time.Millisecond,
				"the waiter acquired the freed lock and became the new starter",
				"waiter never got promoted"))
		if outcome != "broker_start_timeout" || took < 300*time.Millisecond {
			t.Fatalf("waiter was not promoted: %q after %v", outcome, took)
		}
	})
}
