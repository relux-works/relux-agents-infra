package modelharness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func Run(plan Plan) error {
	if err := inspectExecutable(plan.Executable); err != nil {
		return err
	}
	command := exec.Command(plan.Executable, plan.Argv...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s profile %q: %w", plan.Mode, plan.Profile, err)
	}
	return nil
}

func Doctor(plan Plan, stdout, stderr io.Writer) error {
	if err := inspectExecutable(plan.Executable); err != nil {
		return err
	}
	if plan.Mode != "ssh" {
		_, err := fmt.Fprintf(stdout, "profile=%s mode=%s executable=%s status=ok\n", plan.Profile, plan.Mode, plan.Executable)
		return err
	}
	if plan.Remote == nil {
		return errors.New("ssh plan is missing remote details")
	}
	remote := plan.Remote
	remoteRender := remoteCommand(remote.Executable, "render", remote.Profile, remote.Config, remote.Host, remote.Port, true)
	command := exec.Command(plan.Executable,
		"-T",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		remote.Target,
		remoteRender,
	)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("render remote profile %q on %s: %w", remote.Profile, remote.Target, err)
	}
	var remotePlan Plan
	decoder := json.NewDecoder(&output)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&remotePlan); err != nil {
		return fmt.Errorf("decode remote launch plan: %w", err)
	}
	wantEndpoint := fmt.Sprintf("http://%s:%d/v1", remote.Host, remote.Port)
	if remotePlan.Contract != Contract || remotePlan.SchemaVersion != SchemaVersion || remotePlan.Profile != remote.Profile || remotePlan.Endpoint != wantEndpoint {
		return errors.New("remote launch plan identity does not match the configured profile")
	}
	_, err := fmt.Fprintf(stdout, "profile=%s mode=ssh target=%s remote_profile=%s status=ok\n", plan.Profile, remote.Target, remote.Profile)
	return err
}

func inspectExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect executable %s: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("executable %s must be an executable regular file", path)
	}
	return nil
}
