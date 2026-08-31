//go:build !windows

package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type RunPiOptions struct {
	ProjectDir string
	HomeDir    string
	CacheRoot  string
	Args       []string
	Environ    []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	LookPath   func(string) (string, error)
	HTTPClient *http.Client
	Signals    <-chan os.Signal
	Context    context.Context
	Report     *PiRunReport
	Standalone *PiStandaloneRequest
}

var (
	piListen          = net.Listen
	piExecCommand     = exec.Command
	piTerminalFDProbe = piTerminalFD
)

func RunPi(opts RunPiOptions) error {
	if opts.Report != nil {
		*opts.Report = newPiRunReport()
		defer finishPiRunReport(opts.Report)
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	project, err := CanonicalProjectDir(opts.ProjectDir)
	if err != nil {
		return err
	}
	if opts.HomeDir == "" {
		opts.HomeDir, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	if opts.Environ == nil {
		opts.Environ = os.Environ()
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}
	composite, err := loadCompositeProjectConfig(ancestorDirsRootFirst(project), filepath.Join(opts.HomeDir, ".agents", ".configs", projectConfigFileName))
	if err != nil {
		return piError("invalid_project_configuration", err)
	}
	effectiveArgs := opts.Args
	selected := ""
	if opts.Standalone != nil {
		if err := validatePiStandaloneRequest(*opts.Standalone, opts.Args); err != nil {
			return err
		}
		if err := validatePiStandalonePolicy(composite.PiStandaloneSession); err != nil {
			return err
		}
		selected, _, _, err = resolvePiStandaloneSelection(composite, *opts.Standalone)
		if err != nil {
			return err
		}
	} else {
		effectiveArgs, err = applyPiPrimarySessionYolo(opts.Args, composite.PiPrimarySession)
		if err != nil {
			return err
		}
		override, extractErr := ExtractPiProfileOverride(effectiveArgs)
		if extractErr != nil {
			return extractErr
		}
		if override != nil {
			selected = *override
		} else if composite.PiPrimarySession.Profile.Present {
			selected = composite.PiPrimarySession.Profile.Value
		}
	}
	piPath, err := opts.LookPath("pi")
	if err != nil {
		return piError("provider_executable_not_found", err)
	}
	if selected == "" {
		return runPiProcess(piPath, effectiveArgs, project, opts.Environ, opts.Stdin, opts.Stdout, opts.Stderr, false)
	}
	if opts.Report != nil {
		opts.Report.Managed = true
	}
	if !composite.PiPrimarySession.PiCompatibility.Present {
		return piError("invalid_project_configuration", errors.New("managed Pi profile requires pi_compatibility"))
	}
	profile, ok := composite.PiProfiles[selected]
	if !ok {
		return piError("unknown_pi_profile", fmt.Errorf("unknown Pi profile %q", selected))
	}
	if err := ValidatePiStateKeyCollisions(composite.PiProfiles); err != nil {
		return err
	}
	var argsPlan PiArgumentPlan
	if opts.Standalone != nil {
		argsPlan, err = BuildStandalonePiArguments(opts.Args, profile, composite.PiStandaloneSession, opts.Standalone.Prompt)
	} else {
		argsPlan, err = BuildManagedPiArguments(effectiveArgs, selected, profile)
	}
	if err != nil {
		return err
	}
	if err := ValidatePiExecutionEnvironment(opts.Environ); err != nil {
		return err
	}
	for _, item := range opts.Environ {
		name, _, _ := strings.Cut(item, "=")
		if name == "PI_CODING_AGENT_DIR" || name == "PI_CODING_AGENT_SESSION_DIR" || name == "PI_SKIP_VERSION_CHECK" || name == "PI_TELEMETRY" {
			return piError("pi_execution_environment_invalid", fmt.Errorf("inbound %s is forbidden for a managed profile", name))
		}
	}
	identity, err := VerifyPiExecutionIdentity(piPath, composite.PiPrimarySession.PiCompatibility.Value)
	if err != nil {
		return err
	}
	runtimeIdentity, err := inspectRuntimeExecutable(profile.Runtime.Executable)
	if err != nil {
		return err
	}
	if profile.Runtime.Sharing != nil && profile.Runtime.Sharing.Mode == "shared" {
		return runSharedPiSession(opts, project, selected, profile, argsPlan, identity, runtimeIdentity)
	}
	var state PiStatePaths
	if opts.Standalone != nil {
		runID := opts.Standalone.ClientRunID
		if runID == "" {
			runID, err = newPiStandaloneRunID()
			if err != nil {
				return err
			}
			standalone := *opts.Standalone
			standalone.ClientRunID = runID
			opts.Standalone = &standalone
		}
		state, err = ResolvePiClientStatePaths(opts.CacheRoot, project, selected, runID)
	} else {
		state, err = ResolvePiStatePaths(opts.CacheRoot, project, selected)
	}
	if err != nil {
		return err
	}
	if err := CreatePiStateTree(state); err != nil {
		return err
	}
	lock, err := AcquirePiProfileLock(state)
	if err != nil {
		return err
	}
	defer lock.Close()
	sessionLog, err := openPiSessionLog(opts.Context, state, profile.LifecycleLogRetention)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := sessionLog.close(opts.Context)
		status, statusErr := PiLifecycleStatus(context.Background(), state, profile.LifecycleLogRetention, profile.Source, "")
		if closeErr != nil {
			status.UnknownCount++
			status.Errors = append(status.Errors, closeErr.Error())
			status.WithinPolicy, status.SoakReady = false, false
		}
		if statusErr != nil && len(status.Errors) == 0 {
			status.UnknownCount++
			status.Errors = append(status.Errors, statusErr.Error())
		}
		recordPiLifecycleStatus(opts.Report, status)
	}()
	if opts.Report != nil {
		opts.Report.SessionLog = sessionLog.path
	}
	if err := sessionLog.event(opts.Context, "session_start", map[string]any{
		"project": project, "profile": selected, "provider": profile.Provider,
		"model": profile.Model, "thinking": profile.Thinking,
		"transcript_dir": state.SessionsDir,
	}); err != nil {
		return err
	}
	if opts.Stderr != nil {
		fmt.Fprintf(opts.Stderr, "agents-infra: Pi session log: %s\n", sessionLog.path)
	}
	models, err := GeneratePiModelsJSON(profile)
	if err != nil {
		return err
	}
	if err := WritePiModelsJSON(state, models); err != nil {
		return err
	}
	if err := WritePiCompactionSettings(state, profile.Compaction); err != nil {
		return err
	}
	if err := preflightPiListener(profile.BaseURL); err != nil {
		return err
	}
	if current, err := inspectRuntimeExecutable(profile.Runtime.Executable); err != nil || current != runtimeIdentity {
		if err == nil {
			err = errors.New("runtime executable identity changed")
		}
		return piError("runtime_executable_invalid", err)
	}
	if err := opts.Context.Err(); err != nil {
		if opts.Report != nil {
			opts.Report.DeadlineExceeded = errors.Is(err, context.DeadlineExceeded)
		}
		return piError("pi_deadline_exceeded", err)
	}

	runtimeCmd := exec.Command(profile.Runtime.Executable, profile.Runtime.Argv...)
	runtimeCmd.Dir = project
	runtimeCmd.Env = opts.Environ
	outputMu := new(sync.Mutex)
	runtimeOutput := newPiSynchronizedWriter(outputMu, opts.Stderr)
	runtimeCmd.Stdout = runtimeOutput
	runtimeCmd.Stderr = runtimeOutput
	runtimeCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := runtimeCmd.Start(); err != nil {
		_ = sessionLog.event(opts.Context, "runtime_start_failed", map[string]any{"error": err.Error()})
		return piError("runtime_start_failed", err)
	}
	if opts.Report != nil {
		opts.Report.RuntimeProcessGroupCleanup = "pending"
	}
	runtimeWait := waitForPiProcess(runtimeCmd)
	cleanupRuntime := func() error {
		err := terminateProcessGroup(runtimeCmd.Process.Pid, runtimeWait, time.Duration(profile.Runtime.ShutdownTimeoutSeconds)*time.Second)
		fields := map[string]any{"state": processGroupCleanupState(runtimeCmd.Process.Pid, err)}
		if err != nil {
			fields["error"] = err.Error()
		}
		_ = sessionLog.event(opts.Context, "runtime_cleanup", fields)
		if opts.Report != nil {
			opts.Report.RuntimeProcessGroupCleanup = fields["state"].(string)
		}
		return err
	}
	cleaned := false
	defer func() {
		if !cleaned {
			_ = cleanupRuntime()
		}
	}()
	if err := sessionLog.event(opts.Context, "runtime_started", piProcessIdentityFields(runtimeCmd.Process.Pid)); err != nil {
		cleanupErr := cleanupRuntime()
		cleaned = true
		if cleanupErr != nil {
			return cleanupErr
		}
		return err
	}
	wantModel := profile.Model
	if profile.Runtime.DFlash != nil {
		wantModel = profile.Runtime.DFlash.TargetModel
	}
	if err := waitPiRuntimeReady(opts.Context, opts.HTTPClient, profile.BaseURL+profile.Runtime.ReadinessPath, wantModel, runtimeCmd.Process, runtimeWait, time.Duration(profile.Runtime.StartupTimeoutSeconds)*time.Second); err != nil {
		_ = sessionLog.event(opts.Context, "runtime_readiness_failed", map[string]any{"error": err.Error()})
		if opts.Report != nil {
			opts.Report.DeadlineExceeded = errors.Is(err, context.DeadlineExceeded)
		}
		cleanupErr := cleanupRuntime()
		cleaned = true
		if cleanupErr != nil {
			return cleanupErr
		}
		return err
	}
	if err := sessionLog.event(opts.Context, "runtime_ready", map[string]any{"endpoint": profile.BaseURL, "model": wantModel}); err != nil {
		cleanupErr := cleanupRuntime()
		cleaned = true
		if cleanupErr != nil {
			return cleanupErr
		}
		return err
	}
	select {
	case <-runtimeWait.done:
		cleanupErr := cleanupRuntime()
		cleaned = true
		if cleanupErr != nil {
			return cleanupErr
		}
		return piError("runtime_exited_early", errors.New("runtime child exited after readiness"))
	default:
	}
	if err := runtimeCmd.Process.Signal(syscall.Signal(0)); err != nil {
		_ = cleanupRuntime()
		cleaned = true
		return piError("runtime_exited_early", err)
	}
	rechecked, verifyErr := VerifyPiExecutionIdentity(identity.Entrypoint, composite.PiPrimarySession.PiCompatibility.Value)
	if verifyErr != nil || !piIdentityEqual(identity, rechecked) {
		_ = cleanupRuntime()
		cleaned = true
		if verifyErr == nil {
			verifyErr = errors.New("Pi identity changed")
		}
		return piError("pi_execution_identity_changed", verifyErr)
	}
	if err := ValidatePiExecutionEnvironment(opts.Environ); err != nil {
		_ = cleanupRuntime()
		cleaned = true
		return piError("pi_execution_identity_changed", err)
	}
	managedEnv := append([]string(nil), opts.Environ...)
	managedEnv = append(managedEnv, "PI_CODING_AGENT_DIR="+state.AgentDir, "PI_CODING_AGENT_SESSION_DIR="+state.SessionsDir, "PI_SKIP_VERSION_CHECK=1", "PI_TELEMETRY=0")
	piCmd := piExecCommand(identity.Entrypoint, argsPlan.Argv...)
	piCmd.Dir = project
	piCmd.Env = managedEnv
	foreground := false
	if opts.Standalone == nil {
		piCmd.Stdin = opts.Stdin
		foreground = configurePiProcessTerminal(piCmd, opts.Stdin)
	} else {
		piCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	piCmd.Stdout = piProcessWriter(outputMu, opts.Stdout)
	piCmd.Stderr = piProcessWriter(outputMu, opts.Stderr)
	if err := piCmd.Start(); err != nil {
		_ = sessionLog.event(opts.Context, "pi_start_failed", map[string]any{"error": err.Error()})
		_ = cleanupRuntime()
		cleaned = true
		return piError("pi_start_failed", err)
	}
	piFields := piProcessIdentityFields(piCmd.Process.Pid)
	piFields["foreground"] = foreground
	if opts.Report != nil {
		opts.Report.PiProcessGroupCleanup = "pending"
	}
	piWait := waitForPiProcess(piCmd)
	piCleaned := false
	cleanupPi := func(first syscall.Signal) error {
		if piCleaned {
			return nil
		}
		err := terminateProcessGroupWithSignal(piCmd.Process.Pid, piWait, first, time.Duration(profile.Runtime.ShutdownTimeoutSeconds)*time.Second)
		fields := map[string]any{"signal": first.String(), "state": processGroupCleanupState(piCmd.Process.Pid, err)}
		if err != nil {
			fields["error"] = err.Error()
		}
		_ = sessionLog.event(opts.Context, "pi_cleanup", fields)
		piCleaned = true
		if opts.Report != nil {
			opts.Report.PiProcessGroupCleanup = fields["state"].(string)
		}
		return err
	}
	if err := sessionLog.event(opts.Context, "pi_started", piFields); err != nil {
		cleanupErr := cleanupPi(syscall.SIGTERM)
		_ = cleanupRuntime()
		cleaned = true
		if cleanupErr != nil {
			return cleanupErr
		}
		return err
	}
	signals := opts.Signals
	var ownedSignals chan os.Signal
	if signals == nil {
		ownedSignals = make(chan os.Signal, 2)
		signal.Notify(ownedSignals, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(ownedSignals)
		signals = ownedSignals
	}
	contextDone := opts.Context.Done()
	var result error
	select {
	case <-piWait.done:
		result = piWait.err
		fields := map[string]any{}
		if result != nil {
			fields["error"] = result.Error()
		}
		_ = sessionLog.event(opts.Context, "pi_exited", fields)
		select {
		case <-runtimeWait.done:
			result = piError("runtime_exited_early", fmt.Errorf("runtime child exited before Pi session ended: %v", runtimeWait.err))
		default:
			if liveErr := runtimeCmd.Process.Signal(syscall.Signal(0)); liveErr != nil {
				result = piError("runtime_exited_early", liveErr)
			}
		}
	case <-runtimeWait.done:
		_ = sessionLog.event(opts.Context, "runtime_exited_early", map[string]any{"error": fmt.Sprint(runtimeWait.err)})
		_ = cleanupPi(syscall.SIGTERM)
		cleanupErr := cleanupRuntime()
		cleaned = true
		if cleanupErr != nil {
			return cleanupErr
		}
		result = piError("runtime_exited_early", fmt.Errorf("runtime child exited during Pi session: %v", runtimeWait.err))
	case sig := <-signals:
		_ = sessionLog.event(opts.Context, "signal_received", map[string]any{"signal": sig.String()})
		forward := syscall.SIGTERM
		if received, ok := sig.(syscall.Signal); ok {
			forward = received
		}
		result = cleanupPi(forward)
	case <-contextDone:
		_ = sessionLog.event(context.Background(), "context_done", map[string]any{"error": opts.Context.Err().Error()})
		if opts.Report != nil {
			opts.Report.DeadlineExceeded = errors.Is(opts.Context.Err(), context.DeadlineExceeded)
		}
		_ = cleanupPi(syscall.SIGTERM)
		result = piError("pi_deadline_exceeded", opts.Context.Err())
	}
	if !piCleaned {
		if cleanupErr := cleanupPi(syscall.SIGTERM); cleanupErr != nil && result == nil {
			result = cleanupErr
		}
	}
	if !cleaned {
		if cleanupErr := cleanupRuntime(); cleanupErr != nil {
			cleaned = true
			return cleanupErr
		}
		cleaned = true
	}
	endFields := map[string]any{"status": "ok"}
	if result != nil {
		endFields["status"] = "error"
		endFields["error"] = result.Error()
	}
	_ = sessionLog.event(context.Background(), "session_end", endFields)
	return result
}

type piSynchronizedWriter struct {
	mu     *sync.Mutex
	writer io.Writer
}

type piProcessWait struct {
	done chan struct{}
	err  error
}

func waitForPiProcess(cmd *exec.Cmd) *piProcessWait {
	wait := &piProcessWait{done: make(chan struct{})}
	go func() {
		wait.err = cmd.Wait()
		close(wait.done)
	}()
	return wait
}

func newPiSynchronizedWriter(mu *sync.Mutex, writer io.Writer) io.Writer {
	if writer == nil {
		return nil
	}
	return &piSynchronizedWriter{mu: mu, writer: writer}
}

// Preserve terminal file descriptors for the interactive Pi child. Wrapping
// an *os.File in a generic io.Writer makes os/exec insert a pipe, which causes
// Pi to see stdout/stderr as non-TTY and exit immediately with status 0.
func piProcessWriter(mu *sync.Mutex, writer io.Writer) io.Writer {
	if file, ok := writer.(*os.File); ok {
		return file
	}
	return newPiSynchronizedWriter(mu, writer)
}

func configurePiProcessTerminal(cmd *exec.Cmd, stdin io.Reader) bool {
	if fd, ok := piTerminalFDProbe(stdin); ok {
		cmd.SysProcAttr = &syscall.SysProcAttr{Foreground: true, Ctty: fd}
		return true
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return false
}

func (w *piSynchronizedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(p)
}

type runtimeExecutableIdentity struct {
	Dev, Ino uint64
	Size     int64
	Mode     os.FileMode
}

func inspectRuntimeExecutable(path string) (runtimeExecutableIdentity, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return runtimeExecutableIdentity{}, piError("runtime_executable_not_found", err)
	}
	if err != nil {
		return runtimeExecutableIdentity{}, piError("runtime_executable_invalid", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return runtimeExecutableIdentity{}, piError("runtime_executable_invalid", errors.New("runtime executable must be a regular executable file"))
	}
	f, err := os.Open(path)
	if err != nil {
		return runtimeExecutableIdentity{}, piError("runtime_executable_invalid", err)
	}
	f.Close()
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return runtimeExecutableIdentity{}, piError("runtime_executable_invalid", errors.New("runtime stat identity unavailable"))
	}
	return runtimeExecutableIdentity{Dev: uint64(st.Dev), Ino: uint64(st.Ino), Size: info.Size(), Mode: info.Mode()}, nil
}

func preflightPiListener(baseURL string) error {
	u, _ := url.Parse(baseURL)
	listener, err := piListen("tcp4", u.Host)
	if err != nil {
		var op *net.OpError
		if errors.As(err, &op) && errors.Is(op.Err, syscall.EADDRINUSE) {
			return piError("runtime_listener_occupied", err)
		}
		return piError("runtime_listener_check_failed", err)
	}
	return listener.Close()
}

func waitPiRuntimeReady(ctx context.Context, client *http.Client, endpoint, model string, child *os.Process, childWait *piProcessWait, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = &http.Client{
			Timeout:   time.Second,
			Transport: &http.Transport{Proxy: nil},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return piError("pi_deadline_exceeded", ctx.Err())
		case <-childWait.done:
			return piError("runtime_exited_early", fmt.Errorf("runtime exited before readiness: %v", childWait.err))
		case <-deadline.C:
			return piError("runtime_readiness_timeout", errors.New("runtime readiness timed out"))
		case <-ticker.C:
			if err := child.Signal(syscall.Signal(0)); err != nil {
				return piError("runtime_exited_early", err)
			}
			request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if requestErr != nil {
				return piError("runtime_readiness_invalid", requestErr)
			}
			resp, err := client.Do(request)
			if err != nil {
				if ctx.Err() != nil {
					return piError("pi_deadline_exceeded", ctx.Err())
				}
				continue
			}
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			if readErr != nil {
				return piError("runtime_readiness_invalid", fmt.Errorf("invalid readiness response: status=%d read=%v", resp.StatusCode, readErr))
			}
			if resp.StatusCode == http.StatusServiceUnavailable {
				continue
			}
			if resp.StatusCode != http.StatusOK {
				return piError("runtime_readiness_invalid", fmt.Errorf("invalid readiness response: status=%d read=%v", resp.StatusCode, readErr))
			}
			var payload struct {
				Object string `json:"object"`
				Data   *[]struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &payload); err != nil || payload.Object != "list" || payload.Data == nil {
				return piError("runtime_readiness_invalid", errors.New("readiness response is not an OpenAI model list"))
			}
			for _, item := range *payload.Data {
				if item.ID == model {
					return nil
				}
			}
			return piError("runtime_model_unavailable", fmt.Errorf("exact model %q is absent", model))
		}
	}
}

func terminateProcessGroup(pid int, wait *piProcessWait, timeout time.Duration) error {
	return terminateProcessGroupWithSignal(pid, wait, syscall.SIGTERM, timeout)
}

func terminateProcessGroupWithSignal(pid int, wait *piProcessWait, first syscall.Signal, timeout time.Duration) error {
	if err := syscall.Kill(-pid, syscall.Signal(0)); errors.Is(err, syscall.ESRCH) {
		return waitForPiProcessDrain(wait, time.Second)
	}
	_ = syscall.Kill(-pid, first)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pid, syscall.Signal(0)); errors.Is(err, syscall.ESRCH) {
			return waitForPiProcessDrain(wait, time.Second)
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	if err := waitProcessGroupGone(pid, time.Second); err != nil {
		return err
	}
	if err := waitForPiProcessDrain(wait, time.Second); err != nil {
		return err
	}
	return piError("runtime_shutdown_timeout", errors.New("runtime group required SIGKILL"))
}

func processGroupCleanupState(pid int, cleanupErr error) string {
	err := syscall.Kill(-pid, syscall.Signal(0))
	if errors.Is(err, syscall.ESRCH) {
		if cleanupErr == nil {
			return "confirmed"
		}
		return "confirmed_after_sigkill"
	}
	return "failed"
}

func waitForPiProcessDrain(wait *piProcessWait, timeout time.Duration) error {
	select {
	case <-wait.done:
		return nil
	case <-time.After(timeout):
		return piError("runtime_shutdown_timeout", errors.New("runtime child could not be reaped and its output drained"))
	}
}
func waitProcessGroupGone(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Kill(-pid, syscall.Signal(0))
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if time.Now().After(deadline) {
			return piError("runtime_shutdown_timeout", errors.New("runtime process group still exists"))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func runPiProcess(path string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer, setGroup bool) error {
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if setGroup {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	return cmd.Run()
}
