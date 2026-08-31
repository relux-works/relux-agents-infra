//go:build windows

package modelharness

import (
	"os"
	"os/exec"
)

// shutdownSignals is os.Interrupt only: Windows has no SIGTERM or SIGHUP, and
// a service stop arrives as a console control event Go surfaces as
// os.Interrupt.
func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}

// configureRunProcess is a no-op: process groups here are a job-object
// question, which this harness does not manage. Windows shutdown therefore
// stops the direct child only, and that limitation is stated rather than
// papered over.
func configureRunProcess(command *exec.Cmd) {}

func signalRunProcessGroup(command *exec.Cmd, sig os.Signal) error {
	if command.Process == nil {
		return nil
	}
	return command.Process.Kill()
}

func killRunProcessGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}
	return command.Process.Kill()
}

// runProcessGroupStopped can only answer for the direct child. This harness
// manages no job object here, so it holds nothing that would let it observe or
// stop what the runtime spawned; the reaped child is the whole of what it
// controls. The POSIX group contract is not silently claimed on this platform —
// the limitation stated in configureRunProcess is the same one here.
func runProcessGroupStopped(command *exec.Cmd) (bool, error) {
	return true, nil
}
