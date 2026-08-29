//go:build windows

package modelharness

import "os/exec"

func configureStressProcess(command *exec.Cmd) {}

func stopStressProcess(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}
	return command.Process.Kill()
}
