//go:build windows

package infra

import (
	"errors"
	"os/exec"
	"time"

	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
)

func configureProcessACommand(*exec.Cmd) {}

func processAExit(err error) managementpi.ProcessAExit {
	if err == nil {
		return managementpi.ProcessAExit{}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return managementpi.ProcessAExit{Code: exitErr.ExitCode()}
	}
	return managementpi.ProcessAExit{Code: -1}
}

func stopProcessA(command *exec.Cmd, waited <-chan error, timeout time.Duration) (error, managementpi.ProcessACleanupOutcome) {
	if command.Process == nil || command.Process.Kill() != nil {
		return nil, managementpi.ProcessACleanupFailed
	}
	select {
	case err := <-waited:
		return err, managementpi.ProcessACleanupSucceeded
	case <-time.After(timeout):
		return nil, managementpi.ProcessACleanupFailed
	}
}
