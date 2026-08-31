// This file carries the part of the shutdown surface that has to hold on every
// platform, and it carries no build constraint on purpose: the process-group
// tests in run_shutdown_posix_test.go call syscall.Kill, which does not exist
// on Windows, so `GOOS=windows go test -c ./internal/modelharness` fails to
// build the package's tests at all unless the POSIX-only body is constrained
// and something compilable is left behind.

package modelharness

import (
	"os"
	"os/exec"
	"testing"
)

// TestShutdownSignalsHandleTheConsoleStop pins the one signal every platform
// has to answer. Windows has no SIGTERM or SIGHUP, but a console stop and Ctrl-C
// both arrive as os.Interrupt, and on POSIX syscall.SIGINT is that same value.
// A shutdownSignals that does not list it means `signal.Notify` never installs a
// handler, the process dies on the default action with nothing written, and the
// runtime it started is orphaned — blocker B7 from TASK-260828-28gdmq exactly.
func TestShutdownSignalsHandleTheConsoleStop(t *testing.T) {
	signals := shutdownSignals()
	if len(signals) == 0 {
		t.Fatal("shutdownSignals is empty; run would install no handler and die on the default action")
	}
	for _, signal := range signals {
		if signal == os.Interrupt {
			return
		}
	}
	t.Fatalf("shutdownSignals %v does not include os.Interrupt; a console stop would kill the harness and orphan the runtime", signals)
}

// TestRunProcessGroupStoppedOnAnUnstartedCommand fixes the boundary the
// shutdown loop leans on. There is no process, so there is nothing left to wait
// for, and that has to be reported as stopped rather than as an error — on
// Windows too, where the answer covers the direct child alone.
func TestRunProcessGroupStoppedOnAnUnstartedCommand(t *testing.T) {
	stopped, err := runProcessGroupStopped(&exec.Cmd{})
	if err != nil {
		t.Fatalf("inspecting an unstarted command returned %v", err)
	}
	if !stopped {
		t.Fatal("an unstarted command has no group left to wait for; want stopped")
	}
}
