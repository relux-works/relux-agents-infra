//go:build !windows

package modelharness

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// This review-only test narrows the production fixture: the grandchild stays
// in the managed process group, ignores SIGTERM, and closes every harness pipe.
// The contract still requires the group and its listening port to be gone
// before model-harness reports a completed stop.
func TestReviewerShutdownWaitsForTheWholeGroupNotOnlyTheDirectChild(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("requires python3")
	}
	dir := t.TempDir()
	childPID := filepath.Join(dir, "child.pid")
	grandchildPID := filepath.Join(dir, "grandchild.pid")
	port := reserveEphemeralPort(t)
	grandchildProgram := fmt.Sprintf(
		"import signal, socket, time\n"+
			"signal.signal(signal.SIGTERM, signal.SIG_IGN)\n"+
			"sock = socket.socket()\n"+
			"sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)\n"+
			"sock.bind(('127.0.0.1', %d))\n"+
			"sock.listen()\n"+
			"while True: time.sleep(1)\n", port)
	script := fmt.Sprintf(
		"trap 'exit 0' TERM\n"+
			"( exec %s -c %s ) </dev/null >/dev/null 2>&1 &\n"+
			"printf '%%s' \"$!\" > %s\n"+
			"printf '%%s' \"$$\" > %s\n"+
			"while :; do sleep 1; done\n",
		shellQuote(python), shellQuote(grandchildProgram), shellQuote(grandchildPID), shellQuote(childPID))
	plan := shellPlan("detached-port-holder", script)

	var child, grandchild int
	reapFixture(t, &child, &grandchild)
	signals := make(chan os.Signal, 1)
	var stdout, stderr lockedBuffer
	done := make(chan error, 1)
	go func() { done <- runWithSignals(plan, &stdout, &stderr, func(time.Duration) {}, signals) }()

	child = waitForPIDFile(t, childPID)
	grandchild = waitForPIDFile(t, grandchildPID)
	waitForListener(t, port)
	signals <- syscall.SIGTERM
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("harness shutdown failed before checking the group: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("harness did not return after direct child handled SIGTERM")
	}

	grandchildAlive := syscall.Kill(grandchild, 0) == nil
	listener, listenErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if listener != nil {
		_ = listener.Close()
	}
	if grandchildAlive || listenErr != nil {
		t.Fatalf("harness attested a stopped process group while its detached member survived: grandchild_alive=%v port_bind_error=%v stderr=%q", grandchildAlive, listenErr, stderr.String())
	}
}
