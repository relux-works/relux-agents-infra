//go:build !windows

package modelharness

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// shutdownSignals are the signals a service supervisor or a scripted stop
// sends at a standalone `model-harness run`. Before these were handled the
// harness died on the first one with zero bytes written while the runtime it
// started was reparented to pid 1 and kept holding its port.
func shutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
}

// configureRunProcess puts the runtime child in its own process group so the
// harness can stop the runtime *and* anything the runtime spawned with a single
// delivery, instead of stopping only the process it happens to hold a handle
// for.
func configureRunProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// signalRunProcessGroup delivers sig to the whole child process group.
//
// The negative sign is the entire point: `syscall.Kill(pid, ...)` reaches the
// direct child alone, which is exactly the shape that leaves a grandchild
// holding the listening socket after the harness is gone.
func signalRunProcessGroup(command *exec.Cmd, sig os.Signal) error {
	if command.Process == nil {
		return nil
	}
	unixSignal, ok := sig.(syscall.Signal)
	if !ok {
		unixSignal = syscall.SIGTERM
	}
	return syscall.Kill(-command.Process.Pid, unixSignal)
}

// killRunProcessGroup is the escalation, and the fatal-marker path's stop.
func killRunProcessGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}
	return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
}

// runProcessGroupStopped asks the kernel whether the runtime's process group is
// empty. Signal 0 performs the permission and existence checks without
// delivering anything, and aimed at -pgid it answers for every member rather
// than for the one process the harness holds a handle for.
//
// The three answers are different facts and are kept apart. ESRCH is the only
// one that means "empty". EPERM means the group still has a member that is not
// ours to signal, which is the opposite of stopped. Anything else is a failed
// read, and a failed read is not an absence: it is reported as an error so the
// caller escalates instead of attesting a stop it cannot see.
func runProcessGroupStopped(command *exec.Cmd) (bool, error) {
	if command.Process == nil {
		return true, nil
	}
	err := syscall.Kill(-command.Process.Pid, 0)
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, syscall.ESRCH):
		return true, nil
	case errors.Is(err, syscall.EPERM):
		return false, nil
	default:
		return false, fmt.Errorf("inspect process group %d: %w", command.Process.Pid, err)
	}
}
