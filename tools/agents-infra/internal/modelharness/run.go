package modelharness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func Run(plan Plan) error {
	return run(plan, os.Stdout, os.Stderr, time.Sleep)
}

func run(plan Plan, stdout, stderr io.Writer, sleep func(time.Duration)) error {
	if err := inspectExecutable(plan.Executable); err != nil {
		return err
	}
	if plan.Supervision == nil {
		return runOnce(plan, stdout, stderr)
	}
	return runSupervised(plan, stdout, stderr, sleep)
}

func runOnce(plan Plan, stdout, stderr io.Writer) error {
	command := exec.Command(plan.Executable, plan.Argv...)
	command.Stdin = os.Stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s profile %q: %w", plan.Mode, plan.Profile, err)
	}
	return nil
}

type runAttemptResult struct {
	err         error
	fatalMarker string
}

func runSupervised(plan Plan, stdout, stderr io.Writer, sleep func(time.Duration)) error {
	policy := plan.Supervision
	restarts := make([]time.Time, 0, policy.MaxRestarts)
	for {
		result := runSupervisedAttempt(plan, stdout, stderr, policy.FatalOutputSubstrings)
		if result.err == nil && result.fatalMarker == "" {
			return nil
		}
		shouldRestart := result.fatalMarker != "" || policy.RestartOnFailure
		if !shouldRestart {
			return fmt.Errorf("run %s profile %q: %w", plan.Mode, plan.Profile, result.err)
		}

		now := time.Now()
		windowStart := now.Add(-time.Duration(policy.RestartWindowSeconds) * time.Second)
		kept := restarts[:0]
		for _, restartedAt := range restarts {
			if !restartedAt.Before(windowStart) {
				kept = append(kept, restartedAt)
			}
		}
		restarts = kept
		if len(restarts) >= policy.MaxRestarts {
			return fmt.Errorf("run %s profile %q: restart budget exhausted after %d restarts in %ds: %w", plan.Mode, plan.Profile, len(restarts), policy.RestartWindowSeconds, supervisedAttemptError(result))
		}
		restarts = append(restarts, now)
		fmt.Fprintf(stderr, "model-harness: restarting profile %q after supervised runtime failure (%d/%d): %v\n", plan.Profile, len(restarts), policy.MaxRestarts, supervisedAttemptError(result))
		sleep(time.Duration(policy.RestartDelayMilliseconds) * time.Millisecond)
	}
}

func runSupervisedAttempt(plan Plan, stdout, stderr io.Writer, markers []string) runAttemptResult {
	fatal := make(chan string, 1)
	command := exec.Command(plan.Executable, plan.Argv...)
	command.Stdin = os.Stdin
	command.Stdout = newFatalOutputWriter(stdout, markers, fatal)
	command.Stderr = newFatalOutputWriter(stderr, markers, fatal)
	if err := command.Start(); err != nil {
		return runAttemptResult{err: err}
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()

	select {
	case marker := <-fatal:
		killErr := command.Process.Kill()
		waitErr := <-done
		if waitErr == nil {
			waitErr = killErr
		}
		return runAttemptResult{err: waitErr, fatalMarker: marker}
	case err := <-done:
		select {
		case marker := <-fatal:
			return runAttemptResult{err: err, fatalMarker: marker}
		default:
			return runAttemptResult{err: err}
		}
	}
}

func supervisedAttemptError(result runAttemptResult) error {
	if result.fatalMarker != "" {
		if result.err != nil {
			return fmt.Errorf("fatal output %q: %w", result.fatalMarker, result.err)
		}
		return fmt.Errorf("fatal output %q", result.fatalMarker)
	}
	return result.err
}

type fatalOutputWriter struct {
	target       io.Writer
	markers      []string
	notify       chan<- string
	maxMarkerLen int
	mu           sync.Mutex
	carry        string
	matched      bool
}

func newFatalOutputWriter(target io.Writer, markers []string, notify chan<- string) *fatalOutputWriter {
	maxMarkerLen := 0
	for _, marker := range markers {
		if len(marker) > maxMarkerLen {
			maxMarkerLen = len(marker)
		}
	}
	return &fatalOutputWriter{target: target, markers: markers, notify: notify, maxMarkerLen: maxMarkerLen}
}

func (w *fatalOutputWriter) Write(p []byte) (int, error) {
	n, err := w.target.Write(p)
	if n == 0 {
		return n, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.matched {
		return n, err
	}
	combined := w.carry + string(p[:n])
	for _, marker := range w.markers {
		if strings.Contains(combined, marker) {
			w.matched = true
			select {
			case w.notify <- marker:
			default:
			}
			return n, err
		}
	}
	carryLen := w.maxMarkerLen - 1
	if carryLen > len(combined) {
		carryLen = len(combined)
	}
	if carryLen > 0 {
		w.carry = combined[len(combined)-carryLen:]
	}
	return n, err
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
