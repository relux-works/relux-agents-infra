//go:build darwin

package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type standalonePiHelperInfo struct {
	PID         int      `json:"pid"`
	Argv        []string `json:"argv"`
	AgentDir    string   `json:"agent_dir"`
	SessionsDir string   `json:"sessions_dir"`
	StdinEOF    bool     `json:"stdin_eof"`
}

type standaloneWorkerResult struct {
	name string
	err  error
}

func waitStandalonePiHelperInfo(t *testing.T, path string, results <-chan standaloneWorkerResult) standalonePiHelperInfo {
	t.Helper()
	var info standalonePiHelperInfo
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && json.Unmarshal(data, &info) == nil && info.PID > 0 {
			return info
		}
		select {
		case result := <-results:
			t.Fatalf("standalone worker %s exited before publishing process/state evidence: %v", result.name, result.err)
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("standalone Pi helper did not publish process/state evidence")
	return standalonePiHelperInfo{}
}

// Production call site: RunPi -> runSharedPiSession in
// pi_shared_client_darwin.go. Readable stdin and a forced-positive terminal
// probe prove shared standalone workers bypass both interactive attachments.
func TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "pi-standalone-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home, err := os.MkdirTemp("/tmp", "ph-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimePIDFile := filepath.Join(testRoot, "runtime.pid")
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, "profile", runtimeExecutable, port, []string{"serve", "--model", "Model", "--pid-file", runtimePIDFile}, 8)
	body = strings.Replace(body, `reasoning = false`, `reasoning = true`, 1)
	body = strings.Replace(body, `thinking = "off"`, `thinking = "medium"`, 1)
	body = strings.Replace(body, `max_tokens_field = "max_tokens"`, "max_tokens_field = \"max_tokens\"\nthinking_format = \"qwen-chat-template\"", 1)
	body += standalonePiPolicyTOML("true", `["read", "bash", "edit", "write"]`)
	body += canonicalQwenTargetTOML(true, false)
	body += `
[agents.pi.profiles.profile.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 40
`
	writePiProjectConfig(t, project, body)
	piRoot := officialPiAsset(t)

	infoPaths := []string{filepath.Join(testRoot, "pi-a.json"), filepath.Join(testRoot, "pi-b.json")}
	releasePaths := []string{filepath.Join(testRoot, "release-a"), filepath.Join(testRoot, "release-b")}
	originalPiCommand := piExecCommand
	var invocation atomic.Int32
	piExecCommand = func(_ string, argv ...string) *exec.Cmd {
		index := int(invocation.Add(1)) - 1
		if index >= len(infoPaths) {
			index = len(infoPaths) - 1
		}
		args := []string{standalonePiHelperArg, infoPaths[index], releasePaths[index]}
		args = append(args, argv...)
		return exec.Command(os.Args[0], args...)
	}
	t.Cleanup(func() { piExecCommand = originalPiCommand })
	originalTerminalFDProbe := piTerminalFDProbe
	var terminalProbeCalls atomic.Int32
	piTerminalFDProbe = func(io.Reader) (int, bool) {
		terminalProbeCalls.Add(1)
		return 0, true
	}
	t.Cleanup(func() { piTerminalFDProbe = originalTerminalFDProbe })

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	t.Cleanup(cancel)
	results := make(chan standaloneWorkerResult, 2)
	start := func(name, prompt, runID string) {
		go func() {
			err := RunPi(RunPiOptions{
				ProjectDir: project,
				HomeDir:    home,
				CacheRoot:  cache,
				Environ:    []string{"HOME=" + home, "PATH=/usr/bin:/bin", "TASK_BOARD_RUN_ID=RUN-forged-board-id"},
				LookPath:   func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil },
				Context:    ctx,
				Stdin:      strings.NewReader("shared-readable-stdin-witness-" + name),
				Standalone: &PiStandaloneRequest{Prompt: prompt, Entrypoint: "qwen-infra", ClientRunID: runID},
			})
			results <- standaloneWorkerResult{name: name, err: err}
		}()
	}
	start("a", "standalone worker a", "standalone-a")
	first := waitStandalonePiHelperInfo(t, infoPaths[0], results)
	start("b", "standalone worker b", "standalone-b")
	second := waitStandalonePiHelperInfo(t, infoPaths[1], results)

	if first.PID == second.PID || first.AgentDir == second.AgentDir || first.SessionsDir == second.SessionsDir || first.AgentDir == "" || second.AgentDir == "" {
		t.Fatalf("standalone Pi processes/state are not independent: first=%#v second=%#v", first, second)
	}
	firstPGID, firstPGErr := syscall.Getpgid(first.PID)
	secondPGID, secondPGErr := syscall.Getpgid(second.PID)
	if firstPGErr != nil || secondPGErr != nil || firstPGID != first.PID || secondPGID != second.PID || firstPGID == secondPGID {
		t.Fatalf("standalone Pi process groups are not independent: first=%d/%d err=%v second=%d/%d err=%v", first.PID, firstPGID, firstPGErr, second.PID, secondPGID, secondPGErr)
	}
	if calls := terminalProbeCalls.Load(); calls != 0 {
		t.Fatalf("shared standalone workers attempted interactive terminal detection %d time(s)", calls)
	}
	for index, info := range []standalonePiHelperInfo{first, second} {
		if !info.StdinEOF {
			t.Fatalf("worker %d inherited readable stdin: %#v", index, info)
		}
		joined := strings.Join(info.Argv, "\x00")
		for _, required := range []string{"--no-approve", "--no-extensions", "--tools", "read,bash,edit,write", "--mode", "json", "--no-session", "--print", "--thinking", "medium"} {
			if !containsExactString(info.Argv, required) {
				t.Fatalf("worker %d argv lacks %q: %#v", index, required, info.Argv)
			}
		}
		if strings.Contains(joined, "rpc") || containsExactString(info.Argv, "--approve") || containsExactString(info.Argv, "--extension") || containsExactString(info.Argv, "-e") {
			t.Fatalf("worker %d exposed an authorization/extension/RPC bypass: %#v", index, info.Argv)
		}
	}

	status, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile"})
	if err != nil || status.Runtime == nil || len(status.Leases) != 2 {
		t.Fatalf("two standalone leases were not visible: status=%#v err=%v", status, err)
	}
	runtimePID := status.Runtime.PID
	publishedRuntimePID, readErr := os.ReadFile(runtimePIDFile)
	if readErr != nil || strings.TrimSpace(string(publishedRuntimePID)) != strconv.Itoa(runtimePID) {
		t.Fatalf("workers did not reuse the verified runtime: status_pid=%d file=%q err=%v", runtimePID, publishedRuntimePID, readErr)
	}

	if err := syscall.Kill(first.PID, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	firstResult := <-results
	if firstResult.name != "a" || firstResult.err == nil {
		t.Fatalf("crashed standalone worker result = %#v", firstResult)
	}
	waitForSharedTest(t, 8*time.Second, func() bool {
		current, statusErr := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile"})
		return statusErr == nil && current.Runtime != nil && current.Runtime.PID == runtimePID && len(current.Leases) == 1
	}, "crashed worker lease was not released while the peer runtime stayed live")
	if observation, inspectErr := inspectSharedProcessKernel(runtimePID); inspectErr != nil || !observation.live() {
		t.Fatalf("final release of crashed worker stopped the live peer runtime: observation=%#v err=%v", observation, inspectErr)
	}

	if err := os.WriteFile(releasePaths[1], []byte("release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondResult := <-results
	if secondResult.name != "b" || secondResult.err != nil {
		t.Fatalf("live standalone worker result = %#v", secondResult)
	}
	waitForSharedTest(t, 10*time.Second, func() bool {
		_, present, _ := readSharedBrokerRecord(status.Paths.BrokerState)
		observation, inspectErr := inspectSharedProcessKernel(runtimePID)
		return !present && (sharedProcessGone(inspectErr) || (inspectErr == nil && !observation.live()))
	}, "last standalone release did not reap the shared runtime")
}

// Production call site: RunPi -> BuildStandalonePiArguments in
// pi_launch_posix.go. Primary-session project trust is deliberately enabled in
// this composed policy so the launched child proves standalone owns the only
// approval posture it receives.
func TestRunPiStandaloneNeverInheritsPrimarySessionProjectTrust(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "pi-standalone-primary-yolo-")
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
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, "profile", runtimeExecutable, port, []string{"serve", "--model", "Model"}, 8)
	body = strings.Replace(body, `reasoning = false`, `reasoning = true`, 1)
	body = strings.Replace(body, `thinking = "off"`, `thinking = "medium"`, 1)
	body = strings.Replace(body, `max_tokens_field = "max_tokens"`, "max_tokens_field = \"max_tokens\"\nthinking_format = \"qwen-chat-template\"", 1)
	body = strings.Replace(body, "[agents.pi.primary_session]\n", "[agents.pi.primary_session]\nyolo_mode = true\n", 1)
	body += standalonePiPolicyTOML("true", `["read", "bash", "edit", "write"]`)
	body += canonicalQwenTargetTOML(true, false)
	writePiProjectConfig(t, project, body)
	piRoot := officialPiAsset(t)
	infoPath := filepath.Join(testRoot, "pi-primary-yolo.json")
	releasePath := filepath.Join(testRoot, "release-primary-yolo")

	originalPiCommand := piExecCommand
	piExecCommand = func(_ string, argv ...string) *exec.Cmd {
		args := []string{standalonePiHelperArg, infoPath, releasePath}
		args = append(args, argv...)
		return exec.Command(os.Args[0], args...)
	}
	t.Cleanup(func() { piExecCommand = originalPiCommand })

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	t.Cleanup(cancel)
	results := make(chan standaloneWorkerResult, 1)
	go func() {
		err := RunPi(RunPiOptions{
			ProjectDir: project,
			HomeDir:    home,
			CacheRoot:  cache,
			Environ:    []string{"HOME=" + home, "PATH=/usr/bin:/bin"},
			LookPath:   func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil },
			Context:    ctx,
			Standalone: &PiStandaloneRequest{Prompt: "primary yolo isolation worker", Entrypoint: "qwen-infra", ClientRunID: "standalone-primary-yolo"},
		})
		results <- standaloneWorkerResult{name: "primary-yolo", err: err}
	}()
	info := waitStandalonePiHelperInfo(t, infoPath, results)
	noApproveCount := 0
	for _, arg := range info.Argv {
		switch arg {
		case "--no-approve":
			noApproveCount++
		case "--approve", "-a", "-na":
			t.Fatalf("standalone worker received non-owned approval posture %q: %#v", arg, info.Argv)
		}
	}
	if noApproveCount != 1 {
		t.Fatalf("standalone worker --no-approve count = %d, want exactly 1: %#v", noApproveCount, info.Argv)
	}
	if err := os.WriteFile(releasePath, []byte("release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result := <-results; result.err != nil {
		t.Fatalf("standalone worker result = %#v", result)
	}
}

// Production call site: RunPi -> exclusive managed-runtime branch in
// pi_launch_posix.go. The non-empty reader and forced-positive terminal probe
// are discriminating witnesses: if either standalone guard is narrowed away,
// the helper reads a byte or the launch attempts interactive foreground
// attachment instead of preserving a closed-stdin process group.
func TestRunPiStandaloneExclusiveWorkerClosesReadableStdin(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "pi-standalone-exclusive-")
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
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	body := validPiProfileWithArgv(t, "profile", runtimeExecutable, port, []string{"serve", "--model", "Model"}, 8)
	body = strings.Replace(body, `reasoning = false`, `reasoning = true`, 1)
	body = strings.Replace(body, `thinking = "off"`, `thinking = "medium"`, 1)
	body = strings.Replace(body, `max_tokens_field = "max_tokens"`, "max_tokens_field = \"max_tokens\"\nthinking_format = \"qwen-chat-template\"", 1)
	body += standalonePiPolicyTOML("true", `["read", "bash", "edit", "write"]`)
	body += canonicalQwenTargetTOML(true, false)
	writePiProjectConfig(t, project, body)
	piRoot := officialPiAsset(t)
	infoPath := filepath.Join(testRoot, "pi-exclusive.json")
	releasePath := filepath.Join(testRoot, "release-exclusive")

	originalPiCommand := piExecCommand
	piExecCommand = func(_ string, argv ...string) *exec.Cmd {
		args := []string{standalonePiHelperArg, infoPath, releasePath}
		args = append(args, argv...)
		return exec.Command(os.Args[0], args...)
	}
	t.Cleanup(func() { piExecCommand = originalPiCommand })
	originalTerminalFDProbe := piTerminalFDProbe
	var terminalProbeCalls atomic.Int32
	piTerminalFDProbe = func(io.Reader) (int, bool) {
		terminalProbeCalls.Add(1)
		return 0, true
	}
	t.Cleanup(func() { piTerminalFDProbe = originalTerminalFDProbe })

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	t.Cleanup(cancel)
	results := make(chan standaloneWorkerResult, 1)
	go func() {
		err := RunPi(RunPiOptions{
			ProjectDir: project,
			HomeDir:    home,
			CacheRoot:  cache,
			Environ:    []string{"HOME=" + home, "PATH=/usr/bin:/bin"},
			LookPath:   func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil },
			Context:    ctx,
			Stdin:      strings.NewReader("exclusive-readable-stdin-witness"),
			Standalone: &PiStandaloneRequest{Prompt: "exclusive standalone worker", Entrypoint: "qwen-infra", ClientRunID: "standalone-exclusive"},
		})
		results <- standaloneWorkerResult{name: "exclusive", err: err}
	}()
	info := waitStandalonePiHelperInfo(t, infoPath, results)
	if !info.StdinEOF {
		t.Fatalf("exclusive standalone worker inherited readable stdin: %#v", info)
	}
	if calls := terminalProbeCalls.Load(); calls != 0 {
		t.Fatalf("exclusive standalone worker attempted interactive terminal detection %d time(s)", calls)
	}
	if err := os.WriteFile(releasePath, []byte("release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := <-results
	if result.err != nil {
		t.Fatalf("exclusive standalone worker result = %#v", result)
	}
}

func TestStandalonePiStateNeverDerivesFromTaskBoardRunID(t *testing.T) {
	project, cache := t.TempDir(), t.TempDir()
	canonical, err := CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	first, err := ResolvePiClientStatePaths(cache, canonical, "profile", "standalone-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolvePiClientStatePaths(cache, canonical, "profile", "standalone-b")
	if err != nil {
		t.Fatal(err)
	}
	board, err := ResolvePiClientStatePaths(cache, canonical, "profile", "RUN-forged-board-id")
	if err != nil {
		t.Fatal(err)
	}
	if first.RunStateKey == second.RunStateKey || first.RunStateKey == board.RunStateKey || second.RunStateKey == board.RunStateKey {
		t.Fatalf("standalone state identities collapsed onto board or peer state: %q %q %q", first.RunStateKey, second.RunStateKey, board.RunStateKey)
	}
	if strings.Contains(first.Root, "standalone-a") || strings.Contains(second.Root, "standalone-b") || strings.Contains(board.Root, "RUN-forged-board-id") {
		t.Fatalf("raw run identity leaked into state paths: %q %q %q", first.Root, second.Root, board.Root)
	}
}

func TestStandalonePiCrashFailureWrapIsSanitized(t *testing.T) {
	secret := "secret worker prompt"
	wrapped := WrapPiStandaloneFailure(fmt.Errorf("child failed after %s", secret))
	var failure *PiStandaloneFailure
	if !errors.As(wrapped, &failure) || strings.Contains(wrapped.Error(), secret) {
		t.Fatalf("standalone crash failure was not sanitized: %#v", wrapped)
	}
}
