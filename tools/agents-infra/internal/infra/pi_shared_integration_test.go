//go:build darwin

package infra

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	sharedLeaseHelperEnv  = "AGENTS_INFRA_SHARED_LEASE_HELPER"
	sharedLockHelperEnv   = "AGENTS_INFRA_SHARED_LOCK_HELPER"
	sharedCandidateHelper = "AGENTS_INFRA_SHARED_CANDIDATE_HELPER"
	sharedLockPathEnv     = "AGENTS_INFRA_SHARED_LOCK_PATH"
	sharedTestProjectEnv  = "AGENTS_INFRA_SHARED_TEST_PROJECT"
	sharedTestHomeEnv     = "AGENTS_INFRA_SHARED_TEST_HOME"
	sharedTestRunIDEnv    = "AGENTS_INFRA_SHARED_TEST_RUN_ID"
	sharedAuthEvidenceEnv = "AGENTS_INFRA_SHARED_AUTH_EVIDENCE"
	sharedSetNonblockFail = "AGENTS_INFRA_SHARED_SET_NONBLOCK_FAIL"
	sharedFirstGraceEnv   = "AGENTS_INFRA_SHARED_FIRST_GRACE_MS"
	sharedShapeMutantEnv  = "AGENTS_INFRA_SHARED_SHAPE_MUTANT"
	sharedTestProfileName = "profile"
	standalonePiHelperArg = "standalone-pi-helper"
)

// TestMain lets the test binary exercise the broker and runtime-launch entry
// points under the same executable inode as its client. The production broker
// still forks and execs these exact entry points; only argument routing belongs
// to the harness.
func TestMain(m *testing.M) {
	if len(os.Args) >= 4 && os.Args[1] == standalonePiHelperArg {
		stdinProbe := make([]byte, 1)
		count, stdinErr := os.Stdin.Read(stdinProbe)
		info := map[string]any{
			"pid":          os.Getpid(),
			"argv":         append([]string(nil), os.Args[4:]...),
			"agent_dir":    os.Getenv("PI_CODING_AGENT_DIR"),
			"sessions_dir": os.Getenv("PI_CODING_AGENT_SESSION_DIR"),
			"stdin_eof":    count == 0 && errors.Is(stdinErr, io.EOF),
		}
		data, err := json.Marshal(info)
		if err == nil {
			err = os.WriteFile(os.Args[2], data, 0o600)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for {
			if _, err := os.Stat(os.Args[3]); err == nil {
				os.Exit(0)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	if os.Getenv(sharedCandidateHelper) == "1" {
		fmt.Fprintln(os.Stdout, "candidate-ready")
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	if len(os.Args) >= 3 && os.Args[1] == "runtime" && (os.Args[2] == "broker" || os.Args[2] == "runtime-launch") {
		values, err := parseSharedTestInternalArgs(os.Args[3:])
		if err == nil {
			if mutantName := os.Getenv(sharedShapeMutantEnv); mutantName != "" {
				known := mutantName == "reject_all_probe"
				for _, mutant := range authShapeMutants {
					known = known || mutant.name == mutantName
				}
				if !known {
					fmt.Fprintf(os.Stderr, "unknown shape mutant %q\n", mutantName)
					os.Exit(1)
				}
				sharedAuthorizationShapeDecision = func(input sharedAuthShapeInput) sharedAuthShapeVerdict {
					return authProductionMutantVerdict(input, mutantName)
				}
			}
			if value := os.Getenv(sharedFirstGraceEnv); value != "" {
				milliseconds, parseErr := time.ParseDuration(value + "ms")
				if parseErr != nil {
					fmt.Fprintln(os.Stderr, parseErr)
					os.Exit(1)
				}
				sharedFirstLeaseGraceDuration = func(PiRuntimeSharing) time.Duration { return milliseconds }
			}
			if os.Getenv(sharedSetNonblockFail) == "1" {
				sharedRuntimeSetNonblock = func(int, bool) error { return errors.New("injected set-nonblock failure") }
			}
			if evidencePath := os.Getenv(sharedAuthEvidenceEnv); evidencePath != "" {
				sharedAuthEvidenceObserver = func(evidence sharedAuthDecodeEvidence) {
					data, _ := json.Marshal(evidence)
					_ = os.WriteFile(evidencePath, data, 0o600)
				}
			}
			options := SharedRuntimeLauncherOptions{
				RuntimeKey: values["--runtime-key"], ProfileProject: values["--profile-project"],
				ProfileName: values["--profile"], Environ: os.Environ(),
			}
			if os.Args[2] == "broker" {
				err = RunSharedRuntimeBroker(SharedRuntimeBrokerOptions{
					RuntimeKey: options.RuntimeKey, ProfileProject: options.ProfileProject,
					ProfileName: options.ProfileName, Environ: options.Environ,
				})
			} else {
				err = RunSharedRuntimeLauncher(options)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if code, ok := SharedRuntimeExitCode(err); ok {
				os.Exit(code)
			}
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func parseSharedTestInternalArgs(args []string) (map[string]string, error) {
	if len(args) != 6 {
		return nil, fmt.Errorf("expected three internal flag pairs, got %q", args)
	}
	values := map[string]string{}
	for index := 0; index < len(args); index += 2 {
		if args[index] != "--runtime-key" && args[index] != "--profile-project" && args[index] != "--profile" {
			return nil, fmt.Errorf("unexpected internal flag %q", args[index])
		}
		if values[args[index]] != "" || args[index+1] == "" {
			return nil, fmt.Errorf("duplicate or empty internal flag %q", args[index])
		}
		values[args[index]] = args[index+1]
	}
	return values, nil
}

func TestSharedRuntimeLeaseHelper(t *testing.T) {
	if os.Getenv(sharedLeaseHelperEnv) != "1" {
		t.Skip("subprocess helper")
	}
	project := os.Getenv(sharedTestProjectEnv)
	home := os.Getenv(sharedTestHomeEnv)
	runID := os.Getenv(sharedTestRunIDEnv)
	cache := filepath.Join(home, "Library", "Caches")
	resolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(resolved.Paths); err != nil {
		t.Fatal(err)
	}
	state, err := ResolvePiClientStatePaths(cache, resolved.Project, resolved.ProfileName, runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(state); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquirePiProfileLock(state)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	lease, err := acquireSharedRuntimeLease(resolved, state, runID, os.Environ(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.close()
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"lease_id": lease.lease.LeaseID, "runtime_pid": lease.runtime.PID,
		"client_pid": os.Getpid(), "state_root": state.Root,
	}); err != nil {
		t.Fatal(err)
	}
	input := make(chan struct{})
	go func() {
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		close(input)
	}()
	monitor := lease.monitor()
	select {
	case <-input:
		lease.close()
	case err := <-monitor:
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSharedRuntimeLockHolderHelper(t *testing.T) {
	if os.Getenv(sharedLockHelperEnv) != "1" {
		t.Skip("subprocess helper")
	}
	lock, err := openSharedBrokerLock(os.Getenv(sharedLockPathEnv))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, err := fmt.Fprintln(os.Stdout, "locked"); err != nil {
		t.Fatal(err)
	}
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

type sharedLeaseHelperProcess struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	info    struct {
		LeaseID    string `json:"lease_id"`
		RuntimePID int    `json:"runtime_pid"`
		ClientPID  int    `json:"client_pid"`
		StateRoot  string `json:"state_root"`
	}
	stderr *bytes.Buffer
	stdout *bytes.Buffer
	ready  chan error
}

func launchSharedLeaseHelper(t *testing.T, project, home, runID string) *sharedLeaseHelperProcess {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestSharedRuntimeLeaseHelper$", "-test.count=1")
	command.Env = append(os.Environ(),
		sharedLeaseHelperEnv+"=1", sharedTestProjectEnv+"="+project,
		sharedTestHomeEnv+"="+home, sharedTestRunIDEnv+"="+runID, "HOME="+home,
	)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := new(bytes.Buffer)
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	stdoutLog := new(bytes.Buffer)
	helper := &sharedLeaseHelperProcess{command: command, stdin: stdin, stderr: stderr, stdout: stdoutLog, ready: make(chan error, 1)}
	go func() { helper.ready <- json.NewDecoder(io.TeeReader(stdout, stdoutLog)).Decode(&helper.info) }()
	return helper
}

func awaitSharedLeaseHelper(t *testing.T, helper *sharedLeaseHelperProcess) {
	t.Helper()
	select {
	case err := <-helper.ready:
		if err != nil {
			_ = helper.command.Process.Kill()
			_ = helper.command.Wait()
			t.Fatalf("decode lease helper: %v; stdout=%s stderr=%s", err, helper.stdout.String(), helper.stderr.String())
		}
	case <-time.After(12 * time.Second):
		_ = helper.command.Process.Kill()
		_ = helper.command.Wait()
		t.Fatalf("lease helper did not acquire: stdout=%s stderr=%s", helper.stdout.String(), helper.stderr.String())
	}
}

func startSharedLeaseHelper(t *testing.T, project, home, runID string) *sharedLeaseHelperProcess {
	t.Helper()
	helper := launchSharedLeaseHelper(t, project, home, runID)
	awaitSharedLeaseHelper(t, helper)
	return helper
}

func (helper *sharedLeaseHelperProcess) release(t *testing.T) {
	t.Helper()
	if _, err := io.WriteString(helper.stdin, "release\n"); err != nil {
		t.Fatal(err)
	}
	_ = helper.stdin.Close()
	if err := helper.command.Wait(); err != nil {
		t.Fatalf("lease helper exit: %v; stderr=%s", err, helper.stderr.String())
	}
}

func TestSharedRuntimeProductionSingleFlightIndependentClientsCrashAndFinalRelease(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	pidFile := filepath.Join(testRoot, "runtime.pid")
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{"serve", "--model", "Model", "--pid-file", pidFile}, 8)
	body += fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 40
`, sharedTestProfileName)
	writePiProjectConfig(t, project, body)
	peerProject := filepath.Join(testRoot, "peer-project")
	if err := os.MkdirAll(peerProject, 0o700); err != nil {
		t.Fatal(err)
	}
	peerBody := strings.Replace(body, "max_leases = 4", "max_leases = 8", 1)
	writePiProjectConfig(t, peerProject, peerBody)

	first := startSharedLeaseHelper(t, project, home, "RUN-independent-a")
	t.Cleanup(func() {
		if first.command.ProcessState == nil {
			_ = first.command.Process.Kill()
			_ = first.command.Wait()
		}
	})
	resolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}
	loser, err := startSharedRuntimeBroker(resolved, append(os.Environ(), "HOME="+home))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case loserErr := <-loser.done:
		var exitError *exec.ExitError
		if !errors.As(loserErr, &exitError) || exitError.ExitCode() != sharedRuntimeExitElectionLost {
			t.Fatalf("competing broker did not exit election-lost: %v", loserErr)
		}
	case <-time.After(3 * time.Second):
		_ = loser.command.Process.Kill()
		_ = loser.command.Wait()
		t.Fatal("competing broker blocked instead of losing the nonblocking election")
	}

	second := startSharedLeaseHelper(t, peerProject, home, "RUN-independent-b")
	t.Cleanup(func() {
		if second.command.ProcessState == nil {
			_ = second.command.Process.Kill()
			_ = second.command.Wait()
		}
	})
	if first.info.RuntimePID != second.info.RuntimePID || first.info.LeaseID == second.info.LeaseID || first.info.ClientPID == second.info.ClientPID || first.info.StateRoot == second.info.StateRoot {
		t.Fatalf("clients did not share only the runtime: first=%#v second=%#v", first.info, second.info)
	}
	firstSID, _ := syscall.Getsid(first.info.ClientPID)
	secondSID, _ := syscall.Getsid(second.info.ClientPID)
	if firstSID != first.info.ClientPID || secondSID != second.info.ClientPID || firstSID == secondSID {
		t.Fatalf("client sessions are not independent: first=%d/%d second=%d/%d", first.info.ClientPID, firstSID, second.info.ClientPID, secondSID)
	}

	status, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: peerProject, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
	if err != nil {
		t.Fatal(err)
	}
	if status.Broker.State != "serving" || len(status.Leases) != 2 || status.Runtime == nil || status.Runtime.PID != first.info.RuntimePID {
		t.Fatalf("two-lease attested status=%#v", status)
	}
	requireExactSharedRuntimeAttestation(t, status.Attestation)
	if status.Sharing.Configured.MaxLeases != 8 || status.Sharing.Effective == nil || status.Sharing.Effective.MaxLeases != 4 || status.Sharing.FixedByPID != status.Broker.PID {
		t.Fatalf("starter policy was not reported as effective beside peer configuration: %#v", status.Sharing)
	}

	if err := first.command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = first.stdin.Close()
	if err := first.command.Wait(); err == nil {
		t.Fatal("crashed lease helper exited successfully")
	}
	waitForSharedTest(t, 5*time.Second, func() bool {
		current, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: peerProject, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
		return err == nil && len(current.Leases) == 1 && current.Runtime != nil && current.Runtime.PID == second.info.RuntimePID
	}, "broker did not release the crashed client while preserving the live lease")
	if observation, err := inspectSharedProcessKernel(second.info.RuntimePID); err != nil || !observation.live() {
		t.Fatalf("runtime died while second lease remained: observation=%#v err=%v", observation, err)
	}

	second.release(t)
	waitForSharedTest(t, 8*time.Second, func() bool {
		_, present, _ := readSharedBrokerRecord(status.Paths.BrokerState)
		observation, processErr := inspectSharedProcessKernel(second.info.RuntimePID)
		return !present && (sharedProcessGone(processErr) || (processErr == nil && !observation.live()))
	}, "final release did not reap the shared runtime and remove owner state")
}

func TestSharedRuntimeAcquisitionRetriesListenerHandoff(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	bodyForPIDFile := func(pidFile string) string {
		body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{"serve", "--model", "Model", "--pid-file", pidFile}, 5)
		return body + fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 1
max_leases = 4
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 40
`, sharedTestProfileName)
	}
	writePiProjectConfig(t, project, bodyForPIDFile(filepath.Join(testRoot, "runtime-a.pid")))
	firstResolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(firstResolved.Paths); err != nil {
		t.Fatal(err)
	}
	firstState, err := ResolvePiClientStatePaths(cache, project, sharedTestProfileName, "RUN-listener-handoff-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(firstState); err != nil {
		t.Fatal(err)
	}
	firstLease, err := acquireSharedRuntimeLease(firstResolved, firstState, "RUN-listener-handoff-a", append(os.Environ(), "HOME="+home), nil, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	firstRuntimePID := firstLease.runtime.PID
	firstLease.close()
	waitForSharedTest(t, 3*time.Second, func() bool {
		record, present, readErr := readSharedBrokerRecord(firstResolved.Paths.BrokerState)
		return readErr == nil && present && record.State == "lingering"
	}, "first runtime did not enter linger")

	writePiProjectConfig(t, project, bodyForPIDFile(filepath.Join(testRoot, "runtime-b.pid")))
	secondResolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}
	if firstResolved.RuntimeKey == secondResolved.RuntimeKey {
		t.Fatal("profile drift did not change the runtime key")
	}
	if err := CreateSharedRuntimeTree(secondResolved.Paths); err != nil {
		t.Fatal(err)
	}
	secondState, err := ResolvePiClientStatePaths(cache, project, sharedTestProfileName, "RUN-listener-handoff-b")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(secondState); err != nil {
		t.Fatal(err)
	}
	secondLease, err := acquireSharedRuntimeLease(secondResolved, secondState, "RUN-listener-handoff-b", append(os.Environ(), "HOME="+home), nil, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	secondRuntimePID := secondLease.runtime.PID
	logData, err := os.ReadFile(secondResolved.Paths.BrokerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(logData, []byte(`"code":"runtime_listener_occupied"`)) {
		t.Fatalf("broker log did not record the bounded listener handoff retry: %s", logData)
	}
	if secondRuntimePID == 0 || secondRuntimePID == firstRuntimePID {
		t.Fatal("listener handoff acquired no runtime")
	}
	secondLease.close()
	waitForSharedTest(t, 5*time.Second, func() bool {
		_, present, readErr := readSharedBrokerRecord(secondResolved.Paths.BrokerState)
		firstObservation, firstErr := inspectSharedProcessKernel(firstRuntimePID)
		secondObservation, secondErr := inspectSharedProcessKernel(secondRuntimePID)
		firstGone := sharedProcessGone(firstErr) || (firstErr == nil && !firstObservation.live())
		secondGone := sharedProcessGone(secondErr) || (secondErr == nil && !secondObservation.live())
		return readErr == nil && !present && firstGone && secondGone
	}, "listener handoff did not reap both runtime generations")
}

func TestSharedRuntimeAcquisitionRetriesExecutableUpgradeHandoff(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{"serve", "--model", "Model", "--pid-file", filepath.Join(testRoot, "runtime.pid")}, 5)
	body += fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 1
max_leases = 4
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 40
`, sharedTestProfileName)
	writePiProjectConfig(t, project, body)
	resolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(resolved.Paths); err != nil {
		t.Fatal(err)
	}
	state, err := ResolvePiClientStatePaths(cache, project, sharedTestProfileName, "RUN-executable-handoff")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(state); err != nil {
		t.Fatal(err)
	}

	oldExecutable := filepath.Join(testRoot, "old-agents-infra")
	oldBytes, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldExecutable, oldBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	oldBroker := exec.Command(oldExecutable, "runtime", "broker", "--runtime-key", resolved.RuntimeKey, "--profile-project", project, "--profile", sharedTestProfileName)
	oldBroker.Env = append(os.Environ(), "HOME="+home)
	oldBroker.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	oldOutput := new(bytes.Buffer)
	oldBroker.Stdout = oldOutput
	oldBroker.Stderr = oldOutput
	if err := oldBroker.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if oldBroker.ProcessState == nil {
			_ = syscall.Kill(-oldBroker.Process.Pid, syscall.SIGKILL)
			_ = oldBroker.Wait()
		}
	})
	var oldRuntimePID int
	waitForSharedTest(t, 10*time.Second, func() bool {
		record, present, readErr := readSharedBrokerRecord(resolved.Paths.BrokerState)
		if readErr != nil || !present || record.State != "serving" || record.Runtime == nil {
			return false
		}
		oldRuntimePID = record.Runtime.PID
		return true
	}, "old executable broker did not reach serving")

	oldStopped := make(chan error, 1)
	go func() {
		timer := time.NewTimer(350 * time.Millisecond)
		defer timer.Stop()
		<-timer.C
		if err := oldBroker.Process.Signal(syscall.SIGTERM); err != nil {
			oldStopped <- err
			return
		}
		oldStopped <- oldBroker.Wait()
	}()
	lease, err := acquireSharedRuntimeLease(resolved, state, "RUN-executable-handoff", append(os.Environ(), "HOME="+home), nil, context.Background())
	if err != nil {
		t.Fatalf("executable handoff acquisition: %v; old broker output=%s", err, oldOutput)
	}
	newRuntimePID := lease.runtime.PID
	if err := <-oldStopped; err != nil {
		t.Fatalf("old executable broker stop: %v; output=%s", err, oldOutput)
	}
	if newRuntimePID == 0 || newRuntimePID == oldRuntimePID {
		t.Fatalf("executable handoff runtime pid=%d old=%d", newRuntimePID, oldRuntimePID)
	}
	lease.close()
	waitForSharedTest(t, 5*time.Second, func() bool {
		_, present, readErr := readSharedBrokerRecord(resolved.Paths.BrokerState)
		oldObservation, oldErr := inspectSharedProcessKernel(oldRuntimePID)
		newObservation, newErr := inspectSharedProcessKernel(newRuntimePID)
		oldGone := sharedProcessGone(oldErr) || (oldErr == nil && !oldObservation.live())
		newGone := sharedProcessGone(newErr) || (newErr == nil && !newObservation.live())
		return readErr == nil && !present && oldGone && newGone
	}, "executable handoff did not reap both runtime generations")
}

func TestSharedRuntimeOperatorStopRefusesActiveLeasesThenForceDrains(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{"serve", "--model", "Model"}, 8)
	body += fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 40
`, sharedTestProfileName)
	writePiProjectConfig(t, project, body)
	first := startSharedLeaseHelper(t, project, home, "RUN-stop-a")
	second := startSharedLeaseHelper(t, project, home, "RUN-stop-b")
	for _, helper := range []*sharedLeaseHelperProcess{first, second} {
		helper := helper
		t.Cleanup(func() {
			if helper.command.ProcessState == nil {
				_ = helper.command.Process.Kill()
				_ = helper.command.Wait()
			}
		})
	}
	options := SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName}
	_, err = StopSharedRuntime(options, false, 3*time.Second)
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) || shared.Code != "shared_runtime_leases_active" {
		t.Fatalf("non-force stop error=%v want shared_runtime_leases_active", err)
	}
	if count, ok := shared.Details["lease_count"].(int); !ok || count != 2 {
		t.Fatalf("active-lease refusal details=%#v", shared.Details)
	}
	if observation, err := inspectSharedProcessKernel(first.info.RuntimePID); err != nil || !observation.live() {
		t.Fatalf("non-force stop disturbed runtime: observation=%#v err=%v", observation, err)
	}
	result, err := StopSharedRuntime(options, true, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "absent" || !result.BrokerTerminated || !result.RuntimeTerminated || result.RuntimePID != first.info.RuntimePID {
		t.Fatalf("force-stop result=%#v", result)
	}
	for _, helper := range []*sharedLeaseHelperProcess{first, second} {
		_ = helper.stdin.Close()
		if err := helper.command.Wait(); err == nil {
			t.Fatal("revoked lease helper exited successfully instead of surfacing broker termination")
		}
	}
	if observation, err := inspectSharedProcessKernel(first.info.RuntimePID); !sharedProcessGone(err) && (err != nil || observation.live()) {
		t.Fatalf("runtime survived force stop: observation=%#v err=%v", observation, err)
	}
}

func TestSharedRuntimeStarterPolicyGovernsLeaseLimitStatusAndCleanup(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "starter")
	peerProject := filepath.Join(testRoot, "peer")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, peerProject, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{"serve", "--model", "Model"}, 8)
	body += fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 1
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 40
`, sharedTestProfileName)
	writePiProjectConfig(t, project, body)
	writePiProjectConfig(t, peerProject, strings.Replace(body, "max_leases = 1", "max_leases = 16", 1))
	first := startSharedLeaseHelper(t, project, home, "RUN-effective-starter")
	t.Cleanup(func() {
		if first.command.ProcessState == nil {
			_ = first.command.Process.Kill()
			_ = first.command.Wait()
		}
	})
	peer, err := resolveSharedProfile(peerProject, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}
	state, err := ResolvePiClientStatePaths(cache, peer.Project, peer.ProfileName, "RUN-effective-peer")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(state); err != nil {
		t.Fatal(err)
	}
	_, err = acquireSharedRuntimeLease(peer, state, "RUN-effective-peer", append(os.Environ(), "HOME="+home), nil, nil)
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) || shared.Code != "shared_runtime_lease_limit" {
		t.Fatalf("second lease error=%v want shared_runtime_lease_limit", err)
	}
	if shared.Details["effective_max_leases"] != 1 || shared.Details["configured_max_leases"] != 16 || shared.Details["broker_pid"] == 0 || shared.Details["broker_start_time"] == nil {
		t.Fatalf("lease-limit evidence=%#v", shared.Details)
	}
	status, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: peerProject, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
	if err != nil {
		t.Fatal(err)
	}
	if status.Sharing.Configured.MaxLeases != 16 || status.Sharing.Effective == nil || status.Sharing.Effective.MaxLeases != 1 || status.Sharing.FixedByPID != status.Broker.PID || len(status.Leases) != 1 {
		t.Fatalf("effective policy status=%#v", status)
	}
	first.release(t)
	waitForSharedTest(t, 5*time.Second, func() bool {
		_, present, _ := readSharedBrokerRecord(peer.Paths.BrokerState)
		return !present
	}, "effective zero linger did not drain after final release")
}

func TestSharedRuntimeProductionStartupWindowElectsOneOfEight(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	readyFile := filepath.Join(testRoot, "ready")
	spawnLog := filepath.Join(testRoot, "spawns")
	body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{
		"serve", "--model", "Model", "--ready-file", readyFile, "--spawn-log", spawnLog,
	}, 15)
	body += fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 8
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 47
`, sharedTestProfileName)
	writePiProjectConfig(t, project, body)
	resolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}

	helpers := make([]*sharedLeaseHelperProcess, 0, 8)
	for index := 0; index < 8; index++ {
		helper := launchSharedLeaseHelper(t, project, home, fmt.Sprintf("RUN-race-%d", index))
		helpers = append(helpers, helper)
	}
	t.Cleanup(func() {
		for _, helper := range helpers {
			if helper.command.ProcessState == nil {
				_ = helper.command.Process.Kill()
				_ = helper.command.Wait()
			}
		}
	})
	waitForSharedTest(t, 10*time.Second, func() bool {
		data, readErr := os.ReadFile(spawnLog)
		return readErr == nil && bytes.Count(data, []byte("\n")) == 1
	}, "the elected broker did not start exactly one runtime")
	waitForSharedTest(t, 10*time.Second, func() bool {
		data, readErr := os.ReadFile(resolved.Paths.BrokerLog)
		return readErr == nil && bytes.Count(data, []byte("broker_election_lost")) >= 7
	}, "all seven peer brokers did not lose the election while readiness was held")
	if _, err := os.Stat(resolved.Paths.RendezvousSocket); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rendezvous became visible before readiness release: %v", err)
	}
	for _, helper := range helpers {
		select {
		case err := <-helper.ready:
			t.Fatalf("client completed before readiness release: %v; stderr=%s", err, helper.stderr.String())
		default:
		}
	}
	if err := os.WriteFile(readyFile, []byte("ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, helper := range helpers {
		awaitSharedLeaseHelper(t, helper)
	}
	runtimePID := helpers[0].info.RuntimePID
	for _, helper := range helpers {
		if helper.info.RuntimePID != runtimePID {
			t.Fatalf("startup race created multiple runtime pids: first=%d current=%d", runtimePID, helper.info.RuntimePID)
		}
	}
	status, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Leases) != 8 {
		t.Fatalf("race granted %d leases, want 8", len(status.Leases))
	}
	data, err := os.ReadFile(spawnLog)
	if err != nil || bytes.Count(data, []byte("\n")) != 1 {
		t.Fatalf("runtime spawn count after leases=%d err=%v", bytes.Count(data, []byte("\n")), err)
	}
	for _, helper := range helpers {
		helper.release(t)
	}
}

func TestSharedRuntimeFirstLeaseGraceDrainsAbandonedBroker(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{"serve", "--model", "Model"}, 8)
	body += fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 40
`, sharedTestProfileName)
	writePiProjectConfig(t, project, body)
	resolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(resolved.Paths); err != nil {
		t.Fatal(err)
	}
	child, err := startSharedRuntimeBroker(resolved, append(os.Environ(), "HOME="+home, sharedFirstGraceEnv+"=150"))
	if err != nil {
		t.Fatal(err)
	}
	var runtimePID int
	waitForSharedTest(t, 10*time.Second, func() bool {
		record, present, readErr := readSharedBrokerRecord(resolved.Paths.BrokerState)
		if readErr != nil || !present || record.State != "serving" || record.Runtime == nil {
			return false
		}
		runtimePID = record.Runtime.PID
		return true
	}, "abandoned broker did not reach serving")
	select {
	case err := <-child.done:
		if err != nil {
			t.Fatalf("first-lease grace broker exit: %v", err)
		}
	case <-time.After(4 * time.Second):
		_ = child.command.Process.Kill()
		_ = child.command.Wait()
		t.Fatal("first-lease grace did not bound the abandoned broker")
	}
	if _, present, err := readSharedBrokerRecord(resolved.Paths.BrokerState); err != nil || present {
		t.Fatalf("grace drain left owner record: present=%v err=%v", present, err)
	}
	if observation, err := inspectSharedProcessKernel(runtimePID); !sharedProcessGone(err) && (err != nil || observation.live()) {
		t.Fatalf("grace drain left runtime live: observation=%#v err=%v", observation, err)
	}
}

func buildSharedFakeRuntime(t *testing.T, root string) string {
	t.Helper()
	source := filepath.Join(root, "fake-runtime.go")
	executable := filepath.Join(root, "fake-runtime")
	body := `package main
import (
  "encoding/json"
  "flag"
  "net/http"
  "os"
  "strconv"
)
func main() {
  args := os.Args[1:]
  if len(args) > 0 && args[0] == "serve" { args = args[1:] }
  flags := flag.NewFlagSet("runtime", flag.ExitOnError)
  model := flags.String("model", "", "")
  host := flags.String("host", "", "")
  port := flags.Int("port", 0, "")
  pidFile := flags.String("pid-file", "", "")
  readyFile := flags.String("ready-file", "", "")
  spawnLog := flags.String("spawn-log", "", "")
  _ = flags.Parse(args)
  if *pidFile != "" { _ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0600) }
  if *spawnLog != "" {
    file, err := os.OpenFile(*spawnLog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
    if err != nil { panic(err) }
    _, _ = file.WriteString(strconv.Itoa(os.Getpid())+"\n")
    _ = file.Close()
  }
  http.HandleFunc("/v1/models", func(writer http.ResponseWriter, request *http.Request) {
    if *readyFile != "" {
      if _, err := os.Stat(*readyFile); err != nil { http.Error(writer, "not ready", http.StatusServiceUnavailable); return }
    }
    writer.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(writer).Encode(map[string]any{"object":"list", "data":[]map[string]string{{"id":*model}}})
  })
  if err := http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil); err != nil { panic(err) }
}
`
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", executable, source)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build fake shared runtime: %v\n%s", err, output)
	}
	return executable
}

func TestSharedRuntimeForceStopAbsentCleansStaleLeaseMirrors(t *testing.T) {
	project := t.TempDir()
	home, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	cache := filepath.Join(home, "Library", "Caches")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	writePiProjectConfig(t, project, sharedPiProfileTOML("profile", "/bin/echo", 18011))
	resolved, err := resolveSharedProfile(project, home, cache, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(resolved.Paths); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(resolved.Paths.LeasesDir, strings.Repeat("a", 32)+".json")
	if err := os.WriteFile(stale, []byte(`{"lease_id":"stale"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := StopSharedRuntime(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile"}, true, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "absent" {
		t.Fatalf("stop result=%#v", result)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale lease mirror survived absent cleanup: %v", err)
	}
}

func newSharedIntegrationProfile(t *testing.T) (project, home, cache string, resolved sharedResolvedProfile) {
	return newSharedIntegrationProfileAtPort(t, 18011)
}

func newSharedIntegrationProfileAtPort(t *testing.T, port int) (project, home, cache string, resolved sharedResolvedProfile) {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	project = filepath.Join(root, "project")
	home = filepath.Join(root, "home")
	cache = filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writePiProjectConfig(t, project, sharedPiProfileTOML("profile", "/bin/echo", port))
	resolved, err = resolveSharedProfile(project, home, cache, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(resolved.Paths); err != nil {
		t.Fatal(err)
	}
	return project, home, cache, resolved
}

func TestSharedRuntimeAcquisitionHonorsCancelledContextWithoutElection(t *testing.T) {
	_, _, cache, resolved := newSharedIntegrationProfile(t)
	state, err := ResolvePiClientStatePaths(cache, resolved.Project, resolved.ProfileName, "RUN-cancelled")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(state); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = acquireSharedRuntimeLease(resolved, state, "RUN-cancelled", os.Environ(), nil, ctx)
	if piErrorCode(err) != "pi_deadline_exceeded" || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled acquisition error=%v", err)
	}
	for _, path := range []string{resolved.Paths.BrokerState, resolved.Paths.RendezvousSocket, resolved.Paths.BrokerLog} {
		if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("cancelled acquisition started a broker side effect at %s: %v", path, statErr)
		}
	}
}

func TestSharedRuntimeCorruptRecordIsNeverAbsence(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	if err := os.WriteFile(resolved.Paths.BrokerState, []byte(`{"stage":`), 0o600); err != nil {
		t.Fatal(err)
	}
	options := SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile"}
	for _, operation := range []struct {
		name string
		run  func() error
	}{
		{name: "status", run: func() error { _, err := SharedRuntimeStatusReport(options); return err }},
		{name: "force_stop", run: func() error { _, err := StopSharedRuntime(options, true, 200*time.Millisecond); return err }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			err := operation.run()
			var shared *SharedRuntimeError
			if !errors.As(err, &shared) || shared.Code != "shared_runtime_state_unreadable" {
				t.Fatalf("error=%v want shared_runtime_state_unreadable", err)
			}
		})
	}
	state, err := ResolvePiClientStatePaths(cache, resolved.Project, resolved.ProfileName, "RUN-corrupt-record")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(state); err != nil {
		t.Fatal(err)
	}
	_, err = acquireSharedRuntimeLease(resolved, state, "RUN-corrupt-record", append(os.Environ(), "HOME="+home), nil, nil)
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) || shared.Code != "shared_runtime_state_unreadable" {
		t.Fatalf("lease acquisition error=%v want shared_runtime_state_unreadable", err)
	}
	if data, err := os.ReadFile(resolved.Paths.BrokerState); err != nil || string(data) != `{"stage":` {
		t.Fatalf("operator changed unreadable owner record: %q err=%v", data, err)
	}
}

func TestSharedRuntimeHeldLockWithoutRecordIsReportedAndNeverSignalled(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	command := exec.Command(os.Args[0], "-test.run=^TestSharedRuntimeLockHolderHelper$", "-test.count=1")
	command.Env = append(os.Environ(), sharedLockHelperEnv+"=1", sharedLockPathEnv+"="+resolved.Paths.BrokerLock)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := new(bytes.Buffer)
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = io.WriteString(stdin, "release\n")
		_ = stdin.Close()
		if command.ProcessState == nil {
			_ = command.Wait()
		}
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "locked\n" {
		t.Fatalf("lock helper not ready: line=%q err=%v stderr=%s", line, err, stderr.String())
	}
	options := SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile"}
	status, err := SharedRuntimeStatusReport(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Broker.State != "starting-unverified" {
		t.Fatalf("held lock without record reported as %q", status.Broker.State)
	}
	_, err = StopSharedRuntime(options, true, 150*time.Millisecond)
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) || shared.Code != "shared_runtime_owner_unidentifiable" {
		t.Fatalf("force stop error=%v want shared_runtime_owner_unidentifiable", err)
	}
	if observation, err := inspectSharedProcessKernel(command.Process.Pid); err != nil || !observation.live() {
		t.Fatalf("force stop signalled unnamed candidate: observation=%#v err=%v", observation, err)
	}
}

func TestSharedRuntimeForceStopRefusesForgedBrokerIdentityWithoutSignal(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	command := exec.Command("/bin/sleep", "10")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	observation, err := inspectSharedProcess(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	record := SharedBrokerRecord{
		Stage: "serving", State: "serving", ProtocolVersion: SharedRuntimeProtocolVersion,
		Broker: SharedBrokerIdentity{
			PID: observation.PID, PGID: observation.PGID, SID: observation.SID,
			StartTime: observation.StartTime, UID: observation.UID,
			ExecPath: observation.ExecPath, Argv: append([]string(nil), observation.Argv...),
		},
		RuntimeKeyClaimed: resolved.RuntimeKey, RuntimeKey: resolved.RuntimeKey,
	}
	if err := writeSharedJSONAtomic(resolved.Paths.BrokerState, record); err != nil {
		t.Fatal(err)
	}
	_, err = StopSharedRuntime(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile"}, true, 200*time.Millisecond)
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) || shared.Code != "broker_stop_identity_mismatch" {
		t.Fatalf("force stop error=%v want broker_stop_identity_mismatch", err)
	}
	if current, err := inspectSharedProcessKernel(command.Process.Pid); err != nil || !current.live() {
		t.Fatalf("force stop signalled mismatched broker: observation=%#v err=%v", current, err)
	}
}

func TestSharedRuntimeForceStopRejectsRecordedBrokerIdentityNarrowingBeforeSignal(t *testing.T) {
	tests := []struct {
		name         string
		mutateRecord func(*SharedBrokerIdentity)
		mutateSystem func(*sharedRuntimeOperatorDependencies, int)
	}{
		{
			name: "pid reuse start time",
			mutateRecord: func(identity *SharedBrokerIdentity) {
				identity.StartTime.Microseconds++
			},
		},
		{
			name: "recorded argv",
			mutateRecord: func(identity *SharedBrokerIdentity) {
				identity.Argv = append(identity.Argv, "--forged")
			},
		},
		{
			name: "recorded uid root",
			mutateRecord: func(identity *SharedBrokerIdentity) {
				identity.UID = 0
			},
			mutateSystem: func(system *sharedRuntimeOperatorDependencies, pid int) {
				original := system.inspectProcess
				system.inspectProcess = func(candidatePID int) (sharedProcessObservation, error) {
					observation, err := original(candidatePID)
					if candidatePID == pid {
						observation.UID = 0
					}
					return observation, err
				}
			},
		},
		{
			name:         "broker executable same inode wrong device",
			mutateRecord: func(*SharedBrokerIdentity) {},
			mutateSystem: func(system *sharedRuntimeOperatorDependencies, pid int) {
				original := system.processExecIdentity
				system.processExecIdentity = func(observation sharedProcessObservation) (FileIdentity, error) {
					identity, err := original(observation)
					if observation.PID == pid {
						identity.Dev++
					}
					return identity, err
				}
			},
		},
		{
			name:         "broker executable same device wrong inode",
			mutateRecord: func(*SharedBrokerIdentity) {},
			mutateSystem: func(system *sharedRuntimeOperatorDependencies, pid int) {
				original := system.processExecIdentity
				system.processExecIdentity = func(observation sharedProcessObservation) (FileIdentity, error) {
					identity, err := original(observation)
					if observation.PID == pid {
						identity.Ino++
					}
					return identity, err
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.name == "recorded uid root" && os.Geteuid() == 0 {
				t.Skip("root cannot express the uid==0 narrowing witness")
			}
			_, _, _, resolved := newSharedIntegrationProfile(t)
			command, observation := launchSharedRecordedBrokerCandidate(t, resolved.Paths.BrokerLock)
			executableIdentity, err := processExecIdentity(observation)
			if err != nil {
				t.Fatal(err)
			}
			identity := SharedBrokerIdentity{
				PID: observation.PID, PGID: observation.PGID, SID: observation.SID,
				StartTime: observation.StartTime, UID: observation.UID,
				ExecPath: observation.ExecPath, Argv: append([]string(nil), observation.Argv...),
				ExecutableIdentity: executableIdentity,
			}
			testCase.mutateRecord(&identity)
			record := SharedBrokerRecord{
				Stage: "serving", State: "serving", ProtocolVersion: SharedRuntimeProtocolVersion,
				Broker: identity, RuntimeKeyClaimed: resolved.RuntimeKey, RuntimeKey: resolved.RuntimeKey,
			}
			if err := writeSharedJSONAtomic(resolved.Paths.BrokerState, record); err != nil {
				t.Fatal(err)
			}
			system := sharedRuntimeOperatorDependencies{
				inspectProcess:        inspectSharedProcess,
				ownExecutableIdentity: ownResolvedExecutableIdentity,
				processExecIdentity:   processExecIdentity,
				kill:                  syscall.Kill,
			}
			signalAttempted := false
			system.kill = func(int, syscall.Signal) error {
				signalAttempted = true
				return nil
			}
			if testCase.mutateSystem != nil {
				testCase.mutateSystem(&system, command.Process.Pid)
			}

			// This production-entry witness protects both signal sites in
			// stopRecordedSharedRuntime.
			_, err = stopRecordedSharedRuntimeWithDependencies(resolved, &record, time.Now().Add(200*time.Millisecond), system)
			var shared *SharedRuntimeError
			if !errors.As(err, &shared) || shared.Code != "broker_stop_identity_mismatch" {
				t.Fatalf("force stop error=%v want broker_stop_identity_mismatch", err)
			}
			if signalAttempted {
				t.Fatal("untrusted recorded broker identity reached the signal dependency")
			}
			if current, inspectErr := inspectSharedProcessKernel(command.Process.Pid); inspectErr != nil || !current.live() {
				t.Fatalf("force stop signalled recorded broker candidate: observation=%#v err=%v", current, inspectErr)
			}
		})
	}
}

func TestSharedRuntimeBrokerCandidatesRejectSameDeviceWrongInodeAtProductionEntry(t *testing.T) {
	_, _, _, resolved := newSharedIntegrationProfile(t)
	executable, ownIdentity, err := ownResolvedExecutableIdentity()
	if err != nil {
		t.Fatal(err)
	}
	validCommand, _ := launchSharedBrokerCandidateProcess(t, executable, resolved.RuntimeKey)

	copyPath := filepath.Join(t.TempDir(), "different-inode-candidate")
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(copyPath, data, 0o700); err != nil {
		t.Fatal(err)
	}
	copyIdentity, err := fileIdentity(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	if copyIdentity.Dev != ownIdentity.Dev || copyIdentity.Ino == ownIdentity.Ino {
		t.Fatalf("candidate fixture identity=%#v want device=%d inode!=%d", copyIdentity, ownIdentity.Dev, ownIdentity.Ino)
	}
	wrongInodeCommand, _ := launchSharedBrokerCandidateProcess(t, copyPath, resolved.RuntimeKey)

	candidates, err := sharedRuntimeBrokerCandidates(resolved)
	if err != nil {
		t.Fatal(err)
	}
	foundValid := false
	for _, candidate := range candidates {
		if candidate.PID == validCommand.Process.Pid {
			foundValid = true
		}
		if candidate.PID == wrongInodeCommand.Process.Pid {
			t.Fatalf("sharedRuntimeBrokerCandidates admitted same-device/wrong-inode candidate=%#v", candidate)
		}
	}
	if !foundValid {
		t.Fatalf("sharedRuntimeBrokerCandidates did not return exact-executable control pid=%d candidates=%#v", validCommand.Process.Pid, candidates)
	}
}

func launchSharedBrokerCandidateProcess(t *testing.T, executable, runtimeKey string) (*exec.Cmd, sharedProcessObservation) {
	t.Helper()
	command := exec.Command(executable, "runtime", "broker", "--runtime-key", runtimeKey)
	command.Env = append(os.Environ(), sharedCandidateHelper+"=1")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := new(bytes.Buffer)
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		if command.ProcessState == nil {
			_ = command.Wait()
		}
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "candidate-ready\n" {
		t.Fatalf("broker candidate not ready: line=%q err=%v stderr=%s", line, err, stderr.String())
	}
	observation, err := inspectSharedProcess(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	return command, observation
}

func TestSharedRuntimeStatusMarksReusedRecordedBrokerPIDUnverifiedStale(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	command, observation := launchSharedRecordedBrokerCandidate(t, resolved.Paths.BrokerLock)
	observation.StartTime.Microseconds++
	record := SharedBrokerRecord{
		Stage: "serving", State: "serving", ProtocolVersion: SharedRuntimeProtocolVersion,
		Broker: SharedBrokerIdentity{
			PID: observation.PID, PGID: observation.PGID, SID: observation.SID,
			StartTime: observation.StartTime, UID: observation.UID,
			ExecPath: observation.ExecPath, Argv: append([]string(nil), observation.Argv...),
		},
		RuntimeKeyClaimed: resolved.RuntimeKey, RuntimeKey: resolved.RuntimeKey,
	}
	if err := writeSharedJSONAtomic(resolved.Paths.BrokerState, record); err != nil {
		t.Fatal(err)
	}
	status, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{
		ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile",
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.Broker.State != "unverified-stale" || status.Broker.PID != command.Process.Pid {
		t.Fatalf("reused broker pid status=%#v want unverified-stale pid=%d", status.Broker, command.Process.Pid)
	}
	if current, inspectErr := inspectSharedProcessKernel(command.Process.Pid); inspectErr != nil || !current.live() {
		t.Fatalf("status inspection signalled recorded broker candidate: observation=%#v err=%v", current, inspectErr)
	}
}

func launchSharedRecordedBrokerCandidate(t *testing.T, lockPath string) (*exec.Cmd, sharedProcessObservation) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestSharedRuntimeLockHolderHelper$", "-test.count=1")
	command.Env = append(os.Environ(), sharedLockHelperEnv+"=1", sharedLockPathEnv+"="+lockPath)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := new(bytes.Buffer)
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = io.WriteString(stdin, "release\n")
		_ = stdin.Close()
		if command.ProcessState == nil {
			_ = command.Wait()
		}
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "locked\n" {
		t.Fatalf("recorded broker candidate not ready: line=%q err=%v stderr=%s", line, err, stderr.String())
	}
	observation, err := inspectSharedProcess(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	return command, observation
}

func TestSharedRuntimeAdHocBrokerSelfGateAndForeignListener(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	project, home, _, resolved := newSharedIntegrationProfileAtPort(t, port)
	args := []string{"runtime", "broker", "--runtime-key", resolved.RuntimeKey, "--profile-project", project, "--profile", "profile"}

	t.Run("not_session_leader", func(t *testing.T) {
		command := exec.Command(os.Args[0], args...)
		command.Env = append(os.Environ(), "HOME="+home)
		output, err := command.CombinedOutput()
		if err == nil || !bytes.Contains(output, []byte(`"code":"broker_not_session_leader"`)) {
			t.Fatalf("ad-hoc broker err=%v output=%s", err, output)
		}
		if _, err := os.Stat(resolved.Paths.BrokerState); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("B1 made a shared-state side effect: %v", err)
		}
	})

	t.Run("attractive_foreign_listener", func(t *testing.T) {
		command := exec.Command(os.Args[0], args...)
		command.Env = append(os.Environ(), "HOME="+home)
		command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		output, err := command.CombinedOutput()
		if err == nil || !bytes.Contains(output, []byte(`"code":"runtime_listener_occupied"`)) {
			t.Fatalf("broker foreign-listener gate err=%v output=%s", err, output)
		}
		if _, err := os.Stat(resolved.Paths.RendezvousSocket); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("broker bound rendezvous after listener refusal: %v", err)
		}
		if _, present, err := readSharedBrokerRecord(resolved.Paths.BrokerState); err != nil || present {
			t.Fatalf("broker left owner record without runtime: present=%v err=%v", present, err)
		}
	})
}

func TestSharedRuntimeReclamationRefusesUnknownShapeAndReapsVerifiedGroup(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	command := exec.Command("/bin/sleep", "10")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.ProcessState == nil {
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
			_ = command.Wait()
		}
	})
	observation, err := inspectSharedProcess(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	record := &SharedRuntimeProcessRecord{
		PID: observation.PID, PGID: observation.PGID, StartTime: observation.StartTime, UID: observation.UID,
		PreExec:  ProcessExecShape{ExecPath: observation.ExecPath, Argv: append([]string(nil), observation.Argv...)},
		PostExec: ProcessExecShape{ExecPath: observation.ExecPath, Argv: append([]string(nil), observation.Argv...)},
		Endpoint: fmt.Sprintf("http://127.0.0.1:%d/v1", port),
	}
	forged := *record
	forged.PreExec.Argv = []string{"/bin/sleep", "11"}
	forged.PostExec.Argv = []string{"/bin/sleep", "12"}
	err = reclaimSharedRuntime(&forged, 1)
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) || shared.Code != "shared_runtime_orphan_unidentifiable" {
		t.Fatalf("forged reclaim error=%v want shared_runtime_orphan_unidentifiable", err)
	}
	if current, err := inspectSharedProcessKernel(command.Process.Pid); err != nil || !current.live() {
		t.Fatalf("forged record signalled runtime: observation=%#v err=%v", current, err)
	}
	if err := reclaimSharedRuntime(record, 1); err != nil {
		t.Fatal(err)
	}
	if current, err := inspectSharedProcessKernel(command.Process.Pid); !sharedProcessGone(err) && (err != nil || current.live()) {
		t.Fatalf("verified runtime group survived reclamation: observation=%#v err=%v", current, err)
	}
	_ = command.Wait()
}

func waitForSharedTest(t *testing.T, timeout time.Duration, condition func() bool, failure string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal(failure)
}
