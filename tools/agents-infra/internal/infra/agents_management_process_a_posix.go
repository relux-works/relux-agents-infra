//go:build !windows

package infra

import (
	"errors"
	"os/exec"
	"syscall"
	"time"

	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
)

func configureProcessACommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func processAExit(err error) managementpi.ProcessAExit {
	exit := managementpi.ProcessAExit{}
	if err == nil {
		return exit
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		exit.Code = -1
		return exit
	}
	exit.Code = exitErr.ExitCode()
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
		exit.Signaled = status.Signaled()
	}
	return exit
}

func stopProcessA(command *exec.Cmd, waited <-chan error, timeout time.Duration) (error, managementpi.ProcessACleanupOutcome) {
	if command.Process == nil {
		return nil, managementpi.ProcessACleanupFailed
	}
	_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-waited:
		return err, managementpi.ProcessACleanupSucceeded
	case <-timer.C:
	}
	_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	timer.Reset(timeout)
	select {
	case err := <-waited:
		return err, managementpi.ProcessACleanupSucceeded
	case <-timer.C:
		return nil, managementpi.ProcessACleanupFailed
	}
}
