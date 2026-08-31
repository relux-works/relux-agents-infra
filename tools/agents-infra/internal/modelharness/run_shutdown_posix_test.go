//go:build !windows

// The process-group shutdown contract is POSIX-only: signalling a group, and
// asking the kernel whether that group is empty, have no equivalent here on
// Windows, where the harness manages the direct child alone. Keeping these
// tests behind the same constraint as run_process_posix.go is what lets
// `GOOS=windows go test -c ./internal/modelharness` compile; the
// platform-agnostic part of the shutdown surface is tested in
// run_shutdown_test.go, which has no constraint.
package modelharness

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The lifecycle contract these tests hold `model-harness run` to is the one
// TASK-260828-28gdmq recorded as blocker B7: a directed SIGTERM at a standalone
// `model-harness run` used to leave the harness dead with zero bytes written
// while the runtime it started was reparented to pid 1 and kept holding its
// listening port. A service supervisor or a scripted stop sends exactly that
// signal, and llama-server is stopped that way like every other managed
// runtime, so the property has to hold for the group and not just for the
// process the harness holds a handle for.

// TestRunSignalledShutdownStopsTheWholeProcessGroup is the narrowing test.
//
// The runtime child spawns a grandchild — a real runtime forks helpers, and a
// shell-wrapped launch is a grandchild by construction — and the assertion is
// that the grandchild is gone too. Signalling `command.Process.Pid` instead of
// `-command.Process.Pid` still stops the child and still returns cleanly, so a
// test that only checked the direct child or the harness exit status would pass
// against that mutant. This one fails.
func TestRunSignalledShutdownStopsTheWholeProcessGroup(t *testing.T) {
	dir := t.TempDir()
	childPID := filepath.Join(dir, "child.pid")
	grandchildPID := filepath.Join(dir, "grandchild.pid")
	plan := shellPlan("group-shutdown", groupScript(childPID, grandchildPID))
	var child, grandchild int
	reapFixture(t, &child, &grandchild)

	signals := make(chan os.Signal, 1)
	var stdout, stderr lockedBuffer
	done := make(chan error, 1)
	go func() { done <- runWithSignals(plan, &stdout, &stderr, func(time.Duration) {}, signals) }()

	child = waitForPIDFile(t, childPID)
	grandchild = waitForPIDFile(t, grandchildPID)
	if child == grandchild {
		t.Fatalf("child and grandchild are the same process (%d); the fixture proves nothing", child)
	}

	signals <- syscall.SIGTERM
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("signalled shutdown returned %v; a supervisor stop the harness completed is not a failure", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("harness did not return after SIGTERM")
	}

	assertProcessGone(t, "runtime child", child)
	assertProcessGone(t, "runtime grandchild", grandchild)

	// The group must stop on the FORWARDED signal, not on the SIGKILL
	// escalation behind it. Without this the escalation masks the forwarding:
	// a mutant that signals only the direct child still ends with a dead group,
	// because `exec.Cmd.Wait` blocks on the pipes the grandchild inherited, the
	// grace period expires, and the unmutated group kill cleans up what the
	// forwarded signal should have reached. That is a runtime SIGKILLed on
	// every stop, which is not the contract.
	if strings.Contains(stderr.String(), "did not exit within") {
		t.Fatalf("the group survived the forwarded signal and had to be killed; stderr=%q", stderr.String())
	}

	// The lifecycle record B7 found missing. Zero bytes written was half the
	// finding: nothing recorded that a stop had been asked for or completed.
	for _, want := range []string{"received terminated", "stopping profile \"group-shutdown\"", "stopped after terminated"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr does not record %q; stderr=%q", want, stderr.String())
		}
	}
}

// TestRunSignalledShutdownEscalatesToKill covers the runtime that takes SIGTERM
// and does not leave. Without the escalation the harness would wait forever and
// the supervisor's stop would never complete.
func TestRunSignalledShutdownEscalatesToKill(t *testing.T) {
	restoreGrace := runShutdownGrace
	restoreKill := runShutdownKillGrace
	runShutdownGrace = 300 * time.Millisecond
	runShutdownKillGrace = 10 * time.Second
	t.Cleanup(func() {
		runShutdownGrace = restoreGrace
		runShutdownKillGrace = restoreKill
	})

	dir := t.TempDir()
	childPID := filepath.Join(dir, "child.pid")
	grandchildPID := filepath.Join(dir, "grandchild.pid")
	// trap '' TERM makes the child ignore SIGTERM outright.
	plan := shellPlan("stubborn", "trap '' TERM INT HUP\n"+groupScript(childPID, grandchildPID))
	var child, grandchild int
	reapFixture(t, &child, &grandchild)

	signals := make(chan os.Signal, 1)
	var stdout, stderr lockedBuffer
	done := make(chan error, 1)
	go func() { done <- runWithSignals(plan, &stdout, &stderr, func(time.Duration) {}, signals) }()

	child = waitForPIDFile(t, childPID)
	grandchild = waitForPIDFile(t, grandchildPID)

	signals <- syscall.SIGTERM
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("escalated shutdown returned %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("harness did not return after escalating to SIGKILL")
	}
	assertProcessGone(t, "stubborn child", child)
	assertProcessGone(t, "stubborn grandchild", grandchild)
	if !strings.Contains(stderr.String(), "did not exit within") {
		t.Fatalf("escalation was not recorded; stderr=%q", stderr.String())
	}
}

// TestRunSupervisedShutdownDoesNotRestart separates a stop from a failure. The
// supervision policy replaces a runtime that died on its own; it must not
// replace one the operator asked to stop, or `systemctl stop` becomes an
// infinite relaunch.
func TestRunSupervisedShutdownDoesNotRestart(t *testing.T) {
	dir := t.TempDir()
	childPID := filepath.Join(dir, "child.pid")
	grandchildPID := filepath.Join(dir, "grandchild.pid")
	launches := filepath.Join(dir, "launches")
	// The child exits non-zero under SIGTERM, which is what a real runtime does.
	// That is exactly the exit restart_on_failure is configured to replace, so
	// the supervision loop has to tell "the operator stopped it" apart from
	// "it fell over" on the same exit status.
	plan := shellPlan("supervised-stop",
		fmt.Sprintf("printf 'x' >> %s\n", shellQuote(launches))+
			"trap 'exit 143' TERM\n"+groupScript(childPID, grandchildPID))
	var child, grandchild int
	reapFixture(t, &child, &grandchild)
	plan.Supervision = &SupervisionPolicy{
		FatalOutputSubstrings:    []string{"Resource limit ("},
		RestartOnFailure:         true,
		MaxRestarts:              3,
		RestartWindowSeconds:     60,
		RestartDelayMilliseconds: 1,
	}

	signals := make(chan os.Signal, 1)
	var stdout, stderr lockedBuffer
	done := make(chan error, 1)
	go func() { done <- runWithSignals(plan, &stdout, &stderr, func(time.Duration) {}, signals) }()

	child = waitForPIDFile(t, childPID)
	grandchild = waitForPIDFile(t, grandchildPID)
	signals <- syscall.SIGTERM
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("supervised signalled shutdown returned %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("supervised harness did not return after SIGTERM")
	}
	assertProcessGone(t, "supervised child", child)
	assertProcessGone(t, "supervised grandchild", grandchild)
	if strings.Contains(stderr.String(), "restarting profile") {
		t.Fatalf("a signalled stop was treated as a runtime failure and restarted; stderr=%q", stderr.String())
	}
	launched, err := os.ReadFile(launches)
	if err != nil {
		t.Fatal(err)
	}
	if string(launched) != "x" {
		t.Fatalf("runtime was launched %d times after one stop; want 1", len(launched))
	}
}

// TestModelHarnessRunReleasesPortOnDirectedSIGTERM drives the shipped binary
// with a real signal, because everything above feeds the channel by hand and so
// cannot prove that `run` installs a handler at all. Deleting the
// `signal.Notify` call leaves every seam test above passing and fails here.
//
// It also asserts the consequence the operator actually cares about: the port
// the runtime's group held is bindable again once the harness has returned.
func TestModelHarnessRunReleasesPortOnDirectedSIGTERM(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the model-harness binary")
	}
	netcat, err := exec.LookPath("nc")
	if err != nil {
		t.Skip("requires nc to hold a port from a shell grandchild")
	}

	dir := t.TempDir()
	binary := filepath.Join(dir, "model-harness")
	build := exec.Command("go", "build", "-o", binary, "../../cmd/model-harness")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build model-harness: %v", err)
	}

	port := reserveEphemeralPort(t)
	childPID := filepath.Join(dir, "child.pid")
	grandchildPID := filepath.Join(dir, "grandchild.pid")
	script := fmt.Sprintf(
		"%s -l %d >/dev/null 2>&1 &\nprintf '%%s' \"$!\" > %s\nprintf '%%s' \"$$\" > %s\nwhile true; do sleep 0.2; done\n",
		shellQuote(netcat), port, shellQuote(grandchildPID), shellQuote(childPID))
	// The launcher config is the production input: `model-harness run` reads a
	// profile out of it rather than taking an executable on the command line.
	config := filepath.Join(dir, "config.toml")
	configBody := "[profiles.port-holder]\nmode = \"local\"\nexecutable = \"/bin/sh\"\nargv = [" +
		strings.Join([]string{
			tomlString("-c"),
			tomlString(script),
			tomlString("model-harness-test"),
			tomlString("{host}"),
			tomlString("{port}"),
		}, ", ") + "]\n"
	if err := os.WriteFile(config, []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}

	harness := exec.Command(binary, "run", "port-holder", "--config", config, "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	var harnessErr lockedBuffer
	harness.Stderr = &harnessErr
	harness.Stdout = &harnessErr
	if err := harness.Start(); err != nil {
		t.Fatalf("start harness: %v", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- harness.Wait() }()
	t.Cleanup(func() {
		if harness.Process != nil {
			_ = harness.Process.Kill()
		}
		// The runtime child is the group leader here, so this is the pid whose
		// group is safe to aim at. The harness itself is not a group leader and
		// a negative signal at its pid could reach an unrelated group.
		for _, path := range []string{childPID, grandchildPID} {
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if pid, convErr := strconv.Atoi(strings.TrimSpace(string(raw))); convErr == nil && pid > 0 {
				_ = syscall.Kill(-pid, syscall.SIGKILL)
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})

	child := waitForPIDFile(t, childPID)
	grandchild := waitForPIDFile(t, grandchildPID)
	waitForListener(t, port)

	if err := syscall.Kill(harness.Process.Pid, syscall.SIGTERM); err != nil {
		t.Fatalf("signal harness: %v", err)
	}
	select {
	case err := <-exited:
		if err != nil {
			t.Fatalf("harness exited %v after SIGTERM; want a clean stop. output=%q", err, harnessErr.String())
		}
	case <-time.After(60 * time.Second):
		t.Fatalf("harness did not exit after SIGTERM; output=%q", harnessErr.String())
	}

	assertProcessGone(t, "runtime child", child)
	assertProcessGone(t, "port-holding grandchild", grandchild)
	if strings.Contains(harnessErr.String(), "did not exit within") {
		t.Fatalf("the shipped binary had to SIGKILL a group that should have stopped on SIGTERM; output=%q", harnessErr.String())
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("port %d is still held after the harness stopped: %v; output=%q", port, err, harnessErr.String())
	}
	_ = listener.Close()
	if !strings.Contains(harnessErr.String(), "stopping profile \"port-holder\"") {
		t.Fatalf("the shipped binary wrote no lifecycle record for the stop; output=%q", harnessErr.String())
	}
}

// lockedBuffer exists because `exec.Cmd` copies child output into a non-*os.File
// writer from its own goroutine, while `shutdownRuntime` writes lifecycle
// records into the same writer from the caller's. In production both are fd 2
// and the kernel serialises them; in a test they are one bytes.Buffer and must
// be locked. Without this the harness's own records are silently lost and every
// assertion about them becomes a coin flip.
type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// reapFixture guarantees the fixture group is gone when the test ends, however
// the test ended. Without it a mutation run that breaks shutdown leaves the
// fixture shells running after `go test` reports, which is how leaked runtime
// fixtures got into this repository's history before.
func reapFixture(t *testing.T, pids ...*int) {
	t.Cleanup(func() {
		for _, pid := range pids {
			if pid == nil || *pid <= 0 {
				continue
			}
			_ = syscall.Kill(-*pid, syscall.SIGKILL)
			_ = syscall.Kill(*pid, syscall.SIGKILL)
		}
	})
}

// groupScript keeps a runtime child alive with a grandchild beside it, both
// recording their pids so the test can assert on the group rather than on the
// one process the harness holds.
func groupScript(childPID, grandchildPID string) string {
	return fmt.Sprintf(
		"( while true; do sleep 0.2; done ) &\nprintf '%%s' \"$!\" > %s\nprintf '%%s' \"$$\" > %s\nwhile true; do sleep 0.2; done\n",
		shellQuote(grandchildPID), shellQuote(childPID))
}

func shellPlan(profile, script string) Plan {
	return Plan{
		Profile:    profile,
		Mode:       "local",
		Executable: "/bin/sh",
		Argv:       []string{"-c", script, "model-harness-test"},
	}
}

func tomlString(value string) string {
	replaced := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`).Replace(value)
	return `"` + replaced + `"`
}

func waitForPIDFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil {
			if pid, convErr := strconv.Atoi(strings.TrimSpace(string(raw))); convErr == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no pid was written to %s", path)
	return 0
}

func waitForListener(t *testing.T, port int) {
	t.Helper()
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("nothing ever listened on %s; the fixture never held the port", address)
}

// assertProcessGone distinguishes a dead process from an unreadable answer.
// syscall.Kill with signal 0 reports ESRCH for "no such process" and EPERM for
// "alive but not yours"; only the first is death, and an EPERM would be a
// failed read rather than an absence.
func assertProcessGone(t *testing.T, label string, pid int) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		last = syscall.Kill(pid, 0)
		if errors.Is(last, syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s (pid %d) is still alive after the harness stopped: kill(pid, 0) = %v", label, pid, last)
}

func reserveEphemeralPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

// The rest of this file is the regression for the second half of blocker B7:
// the harness attesting a finished stop it could not see.
//
// `shutdownRuntime` used to report "process group N stopped" the instant
// `exec.Cmd.Wait` returned. Wait answers for the direct child alone, so a
// runtime helper spawned into the same group that redirected its inherited
// stdout/stderr away is invisible to it. The harness printed a stopped group,
// returned exit 0, and left that member alive holding the runtime's port —
// which is precisely the state an operator reads the exit code to rule out.

const (
	portHolderPortEnv    = "MODEL_HARNESS_TEST_PORT_HOLDER_PORT"
	portHolderPIDFileEnv = "MODEL_HARNESS_TEST_PORT_HOLDER_PID_FILE"
)

// TestPortHolderHelperProcess is not a test. It is the fixture the two tests
// below need, and it has to be a compiled program rather than a shell line
// because all three of its properties matter at once: it ignores SIGTERM, it
// holds a real loopback listener, and it is started with its stdio redirected
// away from the pipes exec.Cmd.Wait watches. A shell subshell cannot do the
// first while a separate binary holds the socket — the signal would reach the
// binary and free the port.
func TestPortHolderHelperProcess(t *testing.T) {
	port := os.Getenv(portHolderPortEnv)
	if port == "" {
		t.Skip("helper-process entry point; only runs when the fixture env is set")
	}
	// Notify without ever draining is how a process ignores a signal in Go: the
	// runtime installs a handler, so the default terminate action never fires.
	signal.Notify(make(chan os.Signal, 8), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "port holder could not listen on %s: %v\n", port, err)
		os.Exit(1)
	}
	if path := os.Getenv(portHolderPIDFileEnv); path != "" {
		if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "port holder could not record its pid: %v\n", err)
			os.Exit(1)
		}
	}
	for {
		connection, err := listener.Accept()
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		_ = connection.Close()
	}
}

// portHolderScript is the runtime fixture: a child that exits promptly and
// politely on SIGTERM, and one group member beside it that does not.
func portHolderScript(t *testing.T, childPID, holderPID string, port int) string {
	t.Helper()
	helper, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(
		"%s=%d %s=%s %s -test.run=TestPortHolderHelperProcess >/dev/null 2>&1 </dev/null &\n"+
			"printf '%%s' \"$$\" > %s\n"+
			"trap 'exit 143' TERM\n"+
			"while true; do sleep 0.2; done\n",
		portHolderPortEnv, port,
		portHolderPIDFileEnv, shellQuote(holderPID),
		shellQuote(helper),
		shellQuote(childPID))
}

// TestRunSignalledShutdownWaitsForADetachedGroupMember drives the seam.
//
// Two mutants were run against it and both fail here. Returning on Wait alone —
// the revision this fixes — reports a stopped group while the holder is still
// bound to the port. Keeping the group check but reporting stopped once the
// grace expires, instead of escalating to the group, ends the same way: the
// holder ignores SIGTERM, so only the SIGKILL escalation ever clears it.
func TestRunSignalledShutdownWaitsForADetachedGroupMember(t *testing.T) {
	restoreGrace := runShutdownGrace
	restoreKill := runShutdownKillGrace
	runShutdownGrace = 400 * time.Millisecond
	runShutdownKillGrace = 10 * time.Second
	t.Cleanup(func() {
		runShutdownGrace = restoreGrace
		runShutdownKillGrace = restoreKill
	})

	dir := t.TempDir()
	childPID := filepath.Join(dir, "child.pid")
	holderPID := filepath.Join(dir, "holder.pid")
	port := reserveEphemeralPort(t)
	plan := shellPlan("detached-member", portHolderScript(t, childPID, holderPID, port))
	var child, holder int
	reapFixture(t, &child, &holder)

	signals := make(chan os.Signal, 1)
	var stdout, stderr lockedBuffer
	done := make(chan error, 1)
	go func() { done <- runWithSignals(plan, &stdout, &stderr, func(time.Duration) {}, signals) }()

	child = waitForPIDFile(t, childPID)
	holder = waitForPIDFile(t, holderPID)
	waitForListener(t, port)

	signals <- syscall.SIGTERM
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("signalled shutdown returned %v; stderr=%q", err, stderr.String())
		}
	case <-time.After(60 * time.Second):
		t.Fatalf("harness did not return after SIGTERM; stderr=%q", stderr.String())
	}

	// No polling window here on purpose. The harness has already claimed the
	// stop finished; anything still alive at this instant makes that claim
	// false, and a retry loop would launder it into a pass.
	assertProcessGoneNow(t, "runtime child", child, stderr.String())
	assertProcessGoneNow(t, "detached port holder", holder, stderr.String())
	assertPortFree(t, port, stderr.String())

	if !strings.Contains(stderr.String(), "is not empty") {
		t.Fatalf("the harness never recorded that the group outlived its child; stderr=%q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "did not exit within") {
		t.Fatalf("a member that ignores SIGTERM must be escalated to, and the escalation recorded; stderr=%q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "stopped after terminated") {
		t.Fatalf("no stopped record was written once the group was actually empty; stderr=%q", stderr.String())
	}
}

// TestModelHarnessRunDoesNotReportStoppedWhileAGroupMemberHoldsThePort is the
// same property at the production entry point: the shipped `model-harness run`
// binary, a profile read out of a launcher config, and a real directed SIGTERM.
// Everything the seam test proves is proved here about the exit code an
// operator's supervisor actually reads.
func TestModelHarnessRunDoesNotReportStoppedWhileAGroupMemberHoldsThePort(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the model-harness binary and waits out the shipped shutdown grace")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "model-harness")
	build := exec.Command("go", "build", "-o", binary, "../../cmd/model-harness")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build model-harness: %v", err)
	}

	port := reserveEphemeralPort(t)
	childPID := filepath.Join(dir, "child.pid")
	holderPID := filepath.Join(dir, "holder.pid")
	script := portHolderScript(t, childPID, holderPID, port)
	config := filepath.Join(dir, "config.toml")
	configBody := "[profiles.detached-member]\nmode = \"local\"\nexecutable = \"/bin/sh\"\nargv = [" +
		strings.Join([]string{
			tomlString("-c"),
			tomlString(script),
			tomlString("model-harness-test"),
			tomlString("{host}"),
			tomlString("{port}"),
		}, ", ") + "]\n"
	if err := os.WriteFile(config, []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}

	harness := exec.Command(binary, "run", "detached-member", "--config", config, "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	var harnessErr lockedBuffer
	harness.Stderr = &harnessErr
	harness.Stdout = &harnessErr
	if err := harness.Start(); err != nil {
		t.Fatalf("start harness: %v", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- harness.Wait() }()
	t.Cleanup(func() {
		if harness.Process != nil {
			_ = harness.Process.Kill()
		}
		for _, path := range []string{childPID, holderPID} {
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if pid, convErr := strconv.Atoi(strings.TrimSpace(string(raw))); convErr == nil && pid > 0 {
				_ = syscall.Kill(-pid, syscall.SIGKILL)
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})

	child := waitForPIDFile(t, childPID)
	holder := waitForPIDFile(t, holderPID)
	waitForListener(t, port)

	if err := syscall.Kill(harness.Process.Pid, syscall.SIGTERM); err != nil {
		t.Fatalf("signal harness: %v", err)
	}
	select {
	case err := <-exited:
		if err != nil {
			t.Fatalf("harness exited %v after SIGTERM; want a clean stop. output=%q", err, harnessErr.String())
		}
	case <-time.After(90 * time.Second):
		t.Fatalf("harness did not exit after SIGTERM; output=%q", harnessErr.String())
	}

	assertProcessGoneNow(t, "runtime child", child, harnessErr.String())
	assertProcessGoneNow(t, "detached port holder", holder, harnessErr.String())
	assertPortFree(t, port, harnessErr.String())
	if !strings.Contains(harnessErr.String(), "stopped after terminated") {
		t.Fatalf("the shipped binary reported exit 0 without a stopped record; output=%q", harnessErr.String())
	}
}

// assertProcessGoneNow is assertProcessGone without the grace. It is used after
// the harness has already reported the stop finished, where a wait-and-retry
// would be evidence about the operating system rather than about the harness.
func assertProcessGoneNow(t *testing.T, label string, pid int, record string) {
	t.Helper()
	err := syscall.Kill(pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return
	}
	t.Fatalf("%s (pid %d) was alive at the instant the harness reported the group stopped: kill(pid, 0) = %v; record=%q", label, pid, err, record)
}

// assertPortFree is the consequence an operator cares about: the next start of
// the same profile must be able to bind. A stop that returns 0 while the port
// is still held is the failure this whole file exists for.
func assertPortFree(t *testing.T, port int, record string) {
	t.Helper()
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("port %d is still held after the harness reported the group stopped: %v; record=%q", port, err, record)
	}
	_ = listener.Close()
}
