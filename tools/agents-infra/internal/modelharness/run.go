package modelharness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"
)

// runShutdownGrace is how long a signalled runtime is given to exit on its own
// before the harness escalates to SIGKILL, and runShutdownKillGrace is how long
// the group is then given to be reaped. They are variables so the escalation
// path is testable without a ten-second test.
var (
	runShutdownGrace     = 10 * time.Second
	runShutdownKillGrace = 5 * time.Second
)

// runShutdownPollInterval is how often the harness re-asks the kernel whether
// the runtime's process group is empty. `exec.Cmd.Wait` answers for the direct
// child only, so it is not an answer about the group.
var runShutdownPollInterval = 50 * time.Millisecond

func Run(plan Plan) error {
	return run(plan, os.Stdout, os.Stderr, time.Sleep)
}

func run(plan Plan, stdout, stderr io.Writer, sleep func(time.Duration)) error {
	if err := inspectExecutable(plan.Executable); err != nil {
		return err
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, shutdownSignals()...)
	defer signal.Stop(signals)
	return runWithSignals(plan, stdout, stderr, sleep, signals)
}

// runWithSignals is the seam the shutdown tests drive. Production reaches it
// only through run, with a channel fed by signal.Notify.
func runWithSignals(plan Plan, stdout, stderr io.Writer, sleep func(time.Duration), signals <-chan os.Signal) error {
	if plan.Supervision == nil {
		return runOnce(plan, stdout, stderr, signals)
	}
	return runSupervised(plan, stdout, stderr, sleep, signals)
}

func runOnce(plan Plan, stdout, stderr io.Writer, signals <-chan os.Signal) error {
	command := exec.Command(plan.Executable, plan.Argv...)
	command.Stdin = os.Stdin
	command.Stdout = stdout
	command.Stderr = stderr
	configureRunProcess(command)
	if err := command.Start(); err != nil {
		return fmt.Errorf("run %s profile %q: %w", plan.Mode, plan.Profile, err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("run %s profile %q: %w", plan.Mode, plan.Profile, err)
		}
		return nil
	case sig := <-signals:
		_, err := shutdownRuntime(plan, command, done, stderr, sig)
		return err
	}
}

// shutdownRuntime owns the supervisor-signal path.
//
// It forwards the signal to the runtime's whole process group, waits for the
// group to become empty, escalates to SIGKILL if it does not, and authors a
// lifecycle record either way. A signalled stop that completes is a clean stop:
// it returns nil, so a supervisor that sends SIGTERM gets exit 0 rather than a
// failure it did not cause.
//
// "The group is empty" is asked of the kernel, not inferred from
// `exec.Cmd.Wait`. Wait reports the one process the harness holds a handle for,
// and a runtime helper that was spawned into the same group and redirected its
// inherited stdout/stderr away is invisible to it: the child is reaped, Wait
// returns, and a harness that reported a stopped group there would attest a
// finished stop while a detached member still held the runtime's port. That is
// the attestation half of blocker B7 from TASK-260828-28gdmq, whose other half
// was the harness orphaning the runtime outright.
//
// The first return value is how the runtime child itself ended, which the
// supervision loop needs in order to tell an operator stop apart from a crash;
// the second is whether the stop completed.
func shutdownRuntime(plan Plan, command *exec.Cmd, done <-chan error, stderr io.Writer, sig os.Signal) (error, error) {
	group := command.Process.Pid
	fmt.Fprintf(stderr, "model-harness: received %v; stopping profile %q process group %d\n", sig, plan.Profile, group)
	if err := signalRunProcessGroup(command, sig); err != nil {
		fmt.Fprintf(stderr, "model-harness: forwarding %v to profile %q process group %d failed: %v\n", sig, plan.Profile, group, err)
	}
	timer := time.NewTimer(runShutdownGrace)
	defer timer.Stop()
	poll := time.NewTicker(runShutdownPollInterval)
	defer poll.Stop()

	waited := done
	var childErr error
	childReaped := false
	escalated := false
	var inspectErr error

	// settled is the only thing allowed to authorise a "stopped" record: the
	// child reaped *and* the kernel reporting the group empty. An inspection
	// that failed is not an empty group — it is an unknown one, and it keeps
	// the loop running towards the escalation instead of being read as absence.
	settled := func() bool {
		if !childReaped {
			return false
		}
		stopped, err := runProcessGroupStopped(command)
		inspectErr = err
		if err != nil {
			return false
		}
		return stopped
	}
	report := func() (error, error) {
		fmt.Fprintf(stderr, "model-harness: profile %q process group %d stopped after %v (child: %v)\n", plan.Profile, group, sig, childErr)
		return childErr, nil
	}

	for {
		select {
		case waitErr := <-waited:
			childErr = waitErr
			childReaped = true
			// A nil channel never fires again; the child is reaped exactly once.
			waited = nil
			if settled() {
				return report()
			}
			if inspectErr != nil {
				fmt.Fprintf(stderr, "model-harness: profile %q runtime child exited (%v) but process group %d could not be inspected: %v\n", plan.Profile, childErr, group, inspectErr)
				continue
			}
			fmt.Fprintf(stderr, "model-harness: profile %q runtime child exited (%v) but process group %d is not empty; waiting for the rest of the group\n", plan.Profile, childErr, group)
		case <-poll.C:
			if settled() {
				return report()
			}
		case <-timer.C:
			if escalated {
				err := fmt.Errorf("run %s profile %q: process group %d did not exit after SIGKILL", plan.Mode, plan.Profile, group)
				if inspectErr != nil {
					err = fmt.Errorf("run %s profile %q: process group %d state is unknown after SIGKILL: %w", plan.Mode, plan.Profile, group, inspectErr)
				}
				fmt.Fprintf(stderr, "model-harness: %v\n", err)
				return childErr, err
			}
			escalated = true
			fmt.Fprintf(stderr, "model-harness: profile %q process group %d did not exit within %s of %v; killing it\n", plan.Profile, group, runShutdownGrace, sig)
			if err := killRunProcessGroup(command); err != nil {
				fmt.Fprintf(stderr, "model-harness: killing profile %q process group %d failed: %v\n", plan.Profile, group, err)
			}
			timer.Reset(runShutdownKillGrace)
		}
	}
}

type runAttemptResult struct {
	err         error
	fatalMarker string
	// signalled is non-nil when a supervisor signal ended the attempt. Such an
	// attempt is never restarted: the operator asked for a stop, not a replacement.
	signalled os.Signal
	// signalErr is the outcome of the group shutdown that signalled triggered.
	signalErr error
}

func runSupervised(plan Plan, stdout, stderr io.Writer, sleep func(time.Duration), signals <-chan os.Signal) error {
	policy := plan.Supervision
	restarts := make([]time.Time, 0, policy.MaxRestarts)
	for {
		result := runSupervisedAttempt(plan, stdout, stderr, policy.FatalOutputSubstrings, signals)
		// An operator stop is never a restartable failure. Falling through here
		// hands a child that exited non-zero under SIGTERM to restart_on_failure,
		// which relaunches the runtime the operator just asked to stop.
		if result.signalled != nil {
			return result.signalErr
		}
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

func runSupervisedAttempt(plan Plan, stdout, stderr io.Writer, markers []string, signals <-chan os.Signal) runAttemptResult {
	fatal := make(chan string, 1)
	command := exec.Command(plan.Executable, plan.Argv...)
	command.Stdin = os.Stdin
	command.Stdout = newFatalOutputWriter(stdout, markers, fatal)
	command.Stderr = newFatalOutputWriter(stderr, markers, fatal)
	configureRunProcess(command)
	if err := command.Start(); err != nil {
		return runAttemptResult{err: err}
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()

	select {
	case sig := <-signals:
		// The child's own exit is reported verbatim in err. A runtime that exits
		// non-zero when it is asked to stop is a normal stop, not a crash, and
		// the loop below is what has to draw that line.
		childErr, shutdownErr := shutdownRuntime(plan, command, done, stderr, sig)
		return runAttemptResult{signalled: sig, err: childErr, signalErr: shutdownErr}
	case marker := <-fatal:
		// The condemned runtime's whole group goes, not just the process the
		// harness holds a handle for.
		killErr := killRunProcessGroup(command)
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
