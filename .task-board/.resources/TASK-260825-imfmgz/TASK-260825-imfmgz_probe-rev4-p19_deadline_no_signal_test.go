package probe4

// P19 - a client's acquisition deadline is not shutdown authority.
//
// Review RUN-260825-969723 finding F2, BLOCKING: revision 3's client step 4d
// read `broker_child != none` as "my broker is still STARTING" and sent it
// SIGTERM. But section 6.3 keeps the broker the starter's child for its ENTIRE
// life, not only while starting. So the interleaving
//
//   A's last connect() returns ENOENT
//   -> the broker finishes startup and binds
//   -> independent client B connects and takes a lease
//   -> A reaches its deadline, sees a live child, and SIGTERMs it
//
// revokes B's lease with no operator stop. The proxy signal cannot distinguish
// `starting` from `serving`, and revision 3 acted on it anyway.
//
//   P19.A  NEGATIVE, run first: the revision-3 action reproduces the revocation.
//   P19.B  Revision 4: A signals nothing at its deadline. The broker keeps
//          serving and B's lease survives.
//   P19.C  The discriminating fact: in BOTH runs A's own wait4(WNOHANG) reports
//          the identical "still running" answer. The signal is the same; only
//          the action differs. That is why the fix is to stop acting on it, not
//          to inspect it harder.
//   P19.D  Control: nothing keeps a broker alive forever. With no lease ever
//          granted the first-lease grace still drains it, so P19.B is not
//          "never shut anything down".

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func init() { helpers["p19_broker"] = p19Broker }

// p19Broker binds only after a scripted readiness delay - the rendezvous socket
// exists if and only if the broker is serving, exactly as spec section 3.1
// requires - then serves connections as leases until SIGTERM or until the
// first-lease grace expires.
func p19Broker() {
	dir := os.Getenv("P4_DIR")
	sock := filepath.Join(dir, "b.sock")
	readyAfter, _ := time.ParseDuration(env("P4_READY_AFTER", "300ms"))
	grace, _ := time.ParseDuration(env("P4_GRACE", "0"))

	time.Sleep(readyAfter)

	ln, err := net.Listen("unix", sock)
	if err != nil {
		os.Exit(41)
	}
	_ = os.WriteFile(filepath.Join(dir, "serving"), []byte("1"), 0o600)

	var leases int64
	sig := make(chan os.Signal, 1)
	notifySignal(sig)

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			atomic.AddInt64(&leases, 1)
			go func(c net.Conn) {
				defer c.Close()
				w := bufio.NewWriter(c)
				_, _ = w.WriteString("lease\n")
				_ = w.Flush()
				buf := make([]byte, 1)
				_, _ = c.Read(buf) // held until the peer closes
				atomic.AddInt64(&leases, -1)
			}(c)
		}
	}()

	if grace > 0 {
		deadline := time.After(grace)
		for {
			select {
			case <-sig:
				_ = ln.Close()
				_ = os.Remove(sock)
				os.Exit(0)
			case <-deadline:
				if atomic.LoadInt64(&leases) == 0 {
					_ = os.WriteFile(filepath.Join(dir, "grace-drained"), []byte("1"), 0o600)
					_ = ln.Close()
					_ = os.Remove(sock)
					os.Exit(0)
				}
				deadline = nil
			}
		}
	}
	<-sig
	// Drain: this is what revokes every peer lease.
	_ = ln.Close()
	_ = os.Remove(sock)
	os.Exit(0)
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// p19Client is the client wait loop reduced to what F2 is about: attach until a
// deadline, then take (or refuse to take) the revision-3 action.
type p19Client struct {
	pid          int
	childStillUp bool
	refusal      string
}

func p19WaitLoop(t *testing.T, sock string, child int, deadline time.Duration, signalAtDeadline bool) p19Client {
	t.Helper()
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if c, err := net.Dial("unix", sock); err == nil {
			c.Close()
			return p19Client{pid: child, childStillUp: true, refusal: "attached"}
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Step 4d. The observation both revisions make, identically:
	var ws unix.WaitStatus
	wpid, err := unix.Wait4(child, &ws, unix.WNOHANG, nil)
	stillUp := err == nil && wpid == 0
	out := p19Client{pid: child, childStillUp: stillUp}
	if stillUp && signalAtDeadline {
		// REVISION 3: infer "still starting" from "still alive" and terminate.
		_ = unix.Kill(child, unix.SIGTERM)
		out.refusal = "broker_start_timeout (child SIGTERMed)"
	} else {
		// REVISION 4: report the observation, act on nothing.
		out.refusal = "broker_start_timeout (child state unknown to this client)"
	}
	return out
}

func TestP19_ClientDeadlineIsNotShutdownAuthority(t *testing.T) {
	run := func(t *testing.T, signalAtDeadline bool) (peerAlive bool, brokerAlive bool, obs p19Client) {
		dir := shortTempDir(t)
		sock := filepath.Join(dir, "b.sock")

		// Client A forks the broker, which becomes connectable AFTER A's own
		// deadline has already elapsed.
		cmd := startHelper(t, "p19_broker", dir, "P4_READY_AFTER=400ms")
		child := cmd.Process.Pid

		// A's deadline is deliberately shorter than the startup.
		obs = p19WaitLoop(t, sock, child, 150*time.Millisecond, false)
		if obs.refusal == "attached" {
			t.Skip("A attached; the window was missed - rerun")
		}

		// The broker now finishes startup, and INDEPENDENT client B takes a lease.
		if !waitFile(filepath.Join(dir, "serving"), 5*time.Second) {
			t.Fatal("broker never reached serving")
		}
		peer, err := net.Dial("unix", sock)
		if err != nil {
			t.Fatalf("peer B could not take a lease: %v", err)
		}
		defer peer.Close()
		br := bufio.NewReader(peer)
		if line, err := br.ReadString('\n'); err != nil || line != "lease\n" {
			t.Fatalf("peer B lease handshake: %q %v", line, err)
		}

		// Only NOW does A execute step 4d, with B's lease already live.
		obs = p19WaitLoop(t, sock+".never", child, 10*time.Millisecond, signalAtDeadline)

		time.Sleep(400 * time.Millisecond)
		_ = peer.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		buf := make([]byte, 1)
		_, rerr := peer.Read(buf)
		peerAlive = rerr != nil && isTimeout(rerr) // still open == read timed out
		brokerAlive = Alive(child)
		return
	}

	t.Run("A_negative_revision3_action_revokes_a_peer_lease", func(t *testing.T) {
		peerAlive, brokerAlive, obs := run(t, true)
		if !obs.childStillUp {
			t.Fatal("child was not observed alive; the negative did not reproduce")
		}
		if peerAlive {
			t.Fatal("peer lease survived; the revision-3 defect did not reproduce")
		}
		t.Logf("P19.A NEGATIVE reproduced: A observed child_still_up=%v, took the revision-3 action (%s), peer lease REVOKED, broker_alive=%v",
			obs.childStillUp, obs.refusal, brokerAlive)
	})

	t.Run("B_revision4_no_signal_preserves_the_peer_lease", func(t *testing.T) {
		peerAlive, brokerAlive, obs := run(t, false)
		if !obs.childStillUp {
			t.Fatal("child was not observed alive; the two runs are not comparable")
		}
		if !peerAlive {
			t.Fatal("peer lease lost without any signal")
		}
		if !brokerAlive {
			t.Fatal("broker died without any signal")
		}
		t.Logf("P19.B: A observed the IDENTICAL child_still_up=%v and signalled nothing (%s); peer lease intact, broker alive",
			obs.childStillUp, obs.refusal)
	})

	t.Run("D_control_first_lease_grace_still_drains_an_unused_broker", func(t *testing.T) {
		dir := shortTempDir(t)
		cmd := startHelper(t, "p19_broker", dir, "P4_READY_AFTER=50ms", "P4_GRACE=400ms")
		if !waitFile(filepath.Join(dir, "serving"), 5*time.Second) {
			t.Fatal("broker never reached serving")
		}
		if !waitFile(filepath.Join(dir, "grace-drained"), 5*time.Second) {
			t.Fatal("broker with no lease was never bounded")
		}
		if !waitGone(cmd.Process.Pid, 5*time.Second) {
			t.Fatal("broker did not exit after the grace")
		}
		if exists(filepath.Join(dir, "b.sock")) {
			t.Fatal("rendezvous socket left behind")
		}
		t.Log("P19.D control: removing the client's shutdown authority does not leave brokers alive forever - the first-lease grace bounds the case A's SIGTERM was pretending to cover")
	})
}

func isTimeout(err error) bool {
	type timeouter interface{ Timeout() bool }
	te, ok := err.(timeouter)
	return ok && te.Timeout()
}
