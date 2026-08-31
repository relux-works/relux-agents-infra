//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

type productionModelCheckSummary struct {
	SchemaVersion   int    `json:"schema_version"`
	Status          string `json:"status"`
	ExitCode        int    `json:"exit_code"`
	ProcessExitCode *int   `json:"process_exit_code"`
	TimedOut        bool   `json:"timed_out"`
	DeadlineMS      int64  `json:"deadline_ms"`
	DurationMS      int64  `json:"duration_ms"`
	Target          struct {
		Entrypoint  string `json:"entrypoint"`
		Name        string `json:"name"`
		Environment string `json:"environment"`
		Provider    string `json:"provider"`
		Model       string `json:"model"`
	} `json:"target"`
	EventCounts map[string]int `json:"event_counts"`
	ToolCalls   []struct {
		Name      string `json:"name"`
		Completed bool   `json:"completed"`
		Failed    bool   `json:"failed"`
	} `json:"tool_calls"`
	ToolFailures           int    `json:"tool_failures"`
	FinalResponse          string `json:"final_response"`
	FinalResponseTruncated bool   `json:"final_response_truncated"`
	Expectations           []struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
		Met   bool   `json:"met"`
	} `json:"expectations"`
	EventStream struct {
		Valid    bool   `json:"valid"`
		Complete bool   `json:"complete"`
		Error    string `json:"error,omitempty"`
	} `json:"event_stream"`
	ManagedRuntime struct {
		Managed                    bool   `json:"managed"`
		PiProcessGroupCleanup      string `json:"pi_process_group_cleanup"`
		RuntimeProcessGroupCleanup string `json:"runtime_process_group_cleanup"`
		CleanupConfirmed           bool   `json:"cleanup_confirmed"`
	} `json:"managed_runtime"`
	Errors []string `json:"errors"`
}

func TestMainKeepsProviderChildFailuresAtLegacyExitOne(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	project := t.TempDir()
	binDir := t.TempDir()
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 42\n")

	command := exec.Command(binary, "codex")
	command.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+binDir+":/usr/bin:/bin",
		callerCWDEnv+"="+project,
	)
	err := command.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("production codex launch error = %v, want *exec.ExitError", err)
	}
	if got := exitErr.ExitCode(); got != 1 {
		t.Fatalf("production CLI exit = %d, want legacy exit 1 for provider child failure", got)
	}
}

// Production call site: main -> run -> runModelCheck ->
// infra.RunModelCheck -> infra.RunPi. Every case invokes a freshly built
// agents-infra binary against a configured canonical qwen-infra target.
func TestModelCheckProductionEntrypoint(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("the pinned managed Pi production identity is darwin/arm64")
	}
	python := "/usr/bin/python3"
	if _, err := os.Stat(python); err != nil {
		t.Skipf("Python fixture runtime unavailable: %v", err)
	}
	binary := buildInstalledBinary(t)
	piRoot := mainTestOfficialPiAsset(t)
	runtimeScript := writeModelCheckRuntime(t)

	t.Run("happy path persists raw and sanitized evidence", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "happy")
		mustWrite(t, filepath.Join(fixture.project, "fixture.txt"), "fixture input\n")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture,
			"--expect-tool", "read",
			"--expect-text", "MODEL_CHECK_OK",
		)
		if exitCode != 0 {
			t.Fatalf("model-check exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.SchemaVersion != 1 || summary.Status != "passed" || summary.ExitCode != 0 || summary.ProcessExitCode == nil || *summary.ProcessExitCode != 0 || summary.TimedOut {
			t.Fatalf("outcome summary = %#v", summary)
		}
		if summary.DeadlineMS != int64((5*time.Minute)/time.Millisecond) || summary.DurationMS <= 0 {
			t.Fatalf("deadline/duration = %d/%d", summary.DeadlineMS, summary.DurationMS)
		}
		if summary.Target.Entrypoint != "qwen-infra" || summary.Target.Name != "qwen" || summary.Target.Environment != "pi" || summary.Target.Provider != "local-provider" || summary.Target.Model != "Model" {
			t.Fatalf("target summary = %#v", summary.Target)
		}
		if summary.EventCounts["session"] != 1 || summary.EventCounts["agent_end"] != 1 || summary.ToolFailures != 0 || len(summary.ToolCalls) != 1 || summary.ToolCalls[0].Name != "read" || !summary.ToolCalls[0].Completed || summary.ToolCalls[0].Failed {
			t.Fatalf("event/tool summary = counts=%v calls=%#v failures=%d", summary.EventCounts, summary.ToolCalls, summary.ToolFailures)
		}
		if len(summary.Expectations) != 2 || !summary.Expectations[0].Met || !summary.Expectations[1].Met {
			t.Fatalf("expectations = %#v", summary.Expectations)
		}
		if !summary.EventStream.Valid || !summary.EventStream.Complete || !summary.ManagedRuntime.Managed || !summary.ManagedRuntime.CleanupConfirmed {
			t.Fatalf("stream/runtime summary = %#v %#v", summary.EventStream, summary.ManagedRuntime)
		}
		if strings.Contains(summary.FinalResponse, modelCheckFixtureSecret) || strings.Contains(stdout, modelCheckFixtureSecret) || strings.Contains(stderr, modelCheckFixtureSecret) {
			t.Fatalf("sanitized surfaces leaked fixture secret: summary=%q stdout=%q stderr=%q", summary.FinalResponse, stdout, stderr)
		}
		events := mustReadFile(t, filepath.Join(fixture.outputDir, "events.jsonl"))
		if !bytes.Contains(events, []byte(modelCheckFixtureSecret)) {
			t.Fatalf("raw event artifact did not preserve provider bytes: %s", events)
		}
		requests := mustReadFile(t, fixture.requests)
		if !bytes.Contains(requests, []byte(modelCheckFixturePrompt)) {
			t.Fatalf("provider request did not receive caller prompt %q: %s", modelCheckFixturePrompt, requests)
		}
		for _, name := range []string{"events.jsonl", "stderr.log", "summary.json", "summary.txt"} {
			info, err := os.Stat(filepath.Join(fixture.outputDir, name))
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
				t.Fatalf("artifact %s: info=%v err=%v", name, info, err)
			}
		}
		assertModelCheckPIDsGone(t, fixture.runtimePIDs)
	})

	t.Run("final response is bounded in sanitized evidence", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "long-text")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture)
		if exitCode != 0 {
			t.Fatalf("model-check exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if !summary.FinalResponseTruncated || len(summary.FinalResponse) != infra.ModelCheckFinalResponseBytes {
			t.Fatalf("bounded response = bytes=%d truncated=%t", len(summary.FinalResponse), summary.FinalResponseTruncated)
		}
		events := mustReadFile(t, filepath.Join(fixture.outputDir, "events.jsonl"))
		if len(events) <= infra.ModelCheckFinalResponseBytes {
			t.Fatalf("raw artifact unexpectedly bounded to %d bytes", len(events))
		}
		assertModelCheckPIDsGone(t, fixture.runtimePIDs)
	})

	t.Run("missing expected tool refuses", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "text")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture,
			"--expect-tool", "read",
		)
		if exitCode != infra.ModelCheckExitExpectationFailed {
			t.Fatalf("expectation exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitExpectationFailed, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.Status != "failed" || summary.ExitCode != infra.ModelCheckExitExpectationFailed || len(summary.Expectations) != 1 || summary.Expectations[0].Kind != "tool" || summary.Expectations[0].Met {
			t.Fatalf("expectation summary = %#v", summary)
		}
	})

	t.Run("missing expected text refuses", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "text")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture,
			"--expect-text", "ABSENT_TEXT",
		)
		if exitCode != infra.ModelCheckExitExpectationFailed {
			t.Fatalf("expectation exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitExpectationFailed, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.Status != "failed" || summary.ExitCode != infra.ModelCheckExitExpectationFailed || len(summary.Expectations) != 1 || summary.Expectations[0].Kind != "text" || summary.Expectations[0].Met {
			t.Fatalf("expectation summary = %#v", summary)
		}
	})

	t.Run("expected text is limited to the final assistant response", func(t *testing.T) {
		const earlierText = "EARLIER_ASSISTANT_TEXT_MUST_NOT_SATISFY_FINAL_EXPECTATION"
		fixture := newModelCheckFixture(t, runtimeScript, "earlier-text")
		mustWrite(t, filepath.Join(fixture.project, "fixture.txt"), "fixture input\n")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture,
			"--prompt", earlierText,
			"--expect-text", earlierText,
		)
		if exitCode != infra.ModelCheckExitExpectationFailed {
			t.Fatalf("final-response expectation exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitExpectationFailed, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if len(summary.Expectations) != 1 || summary.Expectations[0].Kind != "text" || summary.Expectations[0].Met {
			t.Fatalf("final-response expectation summary = %#v", summary.Expectations)
		}
		if strings.Contains(summary.FinalResponse, earlierText) {
			t.Fatalf("earlier assistant text leaked into final response: %q", summary.FinalResponse)
		}
		if events := mustReadFile(t, filepath.Join(fixture.outputDir, "events.jsonl")); !bytes.Contains(events, []byte(earlierText)) {
			t.Fatalf("fixture did not place earlier text in the event stream: %s", events)
		}
	})

	t.Run("failed tool execution refuses", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "failed-tool")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture)
		if exitCode != infra.ModelCheckExitToolFailure {
			t.Fatalf("tool failure exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitToolFailure, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.ToolFailures != 1 || len(summary.ToolCalls) != 1 || !summary.ToolCalls[0].Completed || !summary.ToolCalls[0].Failed || !summary.EventStream.Valid || !summary.EventStream.Complete {
			t.Fatalf("tool failure summary = %#v", summary)
		}
	})

	t.Run("malformed JSONL refuses instead of treating it as absence", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "malformed")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture)
		if exitCode != infra.ModelCheckExitMalformedStream {
			t.Fatalf("malformed exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitMalformedStream, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.EventStream.Valid || summary.EventStream.Error == "" || summary.ExitCode != infra.ModelCheckExitMalformedStream {
			t.Fatalf("malformed summary = %#v", summary)
		}
	})

	t.Run("deadline override bounds runtime readiness", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "slow-ready")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture, "--deadline", "2s")
		if exitCode != infra.ModelCheckExitTimeout {
			t.Fatalf("readiness timeout exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitTimeout, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.Status != "timed_out" || !summary.TimedOut || summary.DeadlineMS != 2000 || !summary.ManagedRuntime.CleanupConfirmed || summary.ManagedRuntime.PiProcessGroupCleanup != "not_started" {
			t.Fatalf("readiness timeout summary = %#v", summary)
		}
		assertModelCheckPIDsGone(t, fixture.runtimePIDs)
	})

	t.Run("deadline override terminates both owned process groups", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "timeout")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture, "--deadline", "2s")
		if exitCode != infra.ModelCheckExitTimeout {
			t.Fatalf("timeout exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitTimeout, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.Status != "timed_out" || !summary.TimedOut || summary.DeadlineMS != 2000 || !summary.ManagedRuntime.Managed || !summary.ManagedRuntime.CleanupConfirmed {
			t.Fatalf("timeout summary = %#v", summary)
		}
		if summary.ManagedRuntime.PiProcessGroupCleanup == "failed" || summary.ManagedRuntime.RuntimeProcessGroupCleanup == "failed" {
			t.Fatalf("timeout cleanup = %#v", summary.ManagedRuntime)
		}
		assertModelCheckPIDsGone(t, fixture.runtimePIDs)
		assertModelCheckPIDsGone(t, fixture.toolPIDs)
	})

	t.Run("out of range deadline refuses before launch", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "text")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture, "--deadline", "2h")
		if exitCode != infra.ModelCheckExitExecutionFailed || stdout != "" || !strings.Contains(stderr, "deadline must be between 1ms and 30m0s") {
			t.Fatalf("deadline refusal exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
		}
		if _, err := os.Stat(fixture.runtimePIDs); !os.IsNotExist(err) {
			t.Fatalf("deadline refusal launched managed runtime: %v", err)
		}
	})

	t.Run("zero deadline refuses before launch", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "text")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture, "--deadline", "0")
		if exitCode != infra.ModelCheckExitExecutionFailed || stdout != "" || !strings.Contains(stderr, "deadline must be between 1ms and 30m0s") {
			t.Fatalf("zero deadline refusal exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
		}
		if _, err := os.Stat(fixture.runtimePIDs); !os.IsNotExist(err) {
			t.Fatalf("zero deadline refusal launched managed runtime: %v", err)
		}
	})

	t.Run("existing evidence refuses overwrite before launch", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "text")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture)
		if exitCode != 0 {
			t.Fatalf("first model-check exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
		}
		before := mustReadFile(t, filepath.Join(fixture.outputDir, "summary.json"))
		stdout, stderr, exitCode = runModelCheckProduction(t, binary, piRoot, fixture)
		if exitCode != infra.ModelCheckExitExecutionFailed || stdout != "" || !strings.Contains(stderr, "refuses to overwrite existing artifact events.jsonl") {
			t.Fatalf("overwrite refusal exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
		}
		after := mustReadFile(t, filepath.Join(fixture.outputDir, "summary.json"))
		if !bytes.Equal(before, after) {
			t.Fatal("overwrite refusal modified prior summary evidence")
		}
		assertModelCheckPIDsGone(t, fixture.runtimePIDs)
	})

	t.Run("preexisting raw events alone refuse overwrite", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "text")
		mustMkdir(t, fixture.outputDir)
		eventsPath := filepath.Join(fixture.outputDir, "events.jsonl")
		mustWrite(t, eventsPath, "prior-run-event\n")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture)
		if exitCode != infra.ModelCheckExitExecutionFailed || stdout != "" || !strings.Contains(stderr, "refuses to overwrite existing artifact events.jsonl") {
			t.Fatalf("raw-event refusal exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
		}
		if got := string(mustReadFile(t, eventsPath)); got != "prior-run-event\n" {
			t.Fatalf("raw-event refusal modified prior evidence: %q", got)
		}
		if _, err := os.Stat(fixture.runtimePIDs); !os.IsNotExist(err) {
			t.Fatalf("raw-event refusal launched managed runtime: %v", err)
		}
	})

	t.Run("non-managed canonical target refuses without inventing stream failure", func(t *testing.T) {
		fixture := newModelCheckFixture(t, runtimeScript, "text")
		stdout, stderr, exitCode := runModelCheckProduction(t, binary, piRoot, fixture, "--target", "openai-infra")
		if exitCode != infra.ModelCheckExitExecutionFailed {
			t.Fatalf("non-managed target exit=%d want=%d\nstdout:\n%s\nstderr:\n%s", exitCode, infra.ModelCheckExitExecutionFailed, stdout, stderr)
		}
		summary := readProductionModelCheckSummary(t, fixture.outputDir)
		if summary.Status != "failed" || summary.ExitCode != infra.ModelCheckExitExecutionFailed || summary.EventStream.Complete || summary.EventStream.Error != "" {
			t.Fatalf("non-managed target summary = %#v", summary)
		}
		if len(summary.Errors) != 1 || !strings.Contains(summary.Errors[0], "target must resolve to the managed Pi environment") {
			t.Fatalf("non-managed target reasons = %#v", summary.Errors)
		}
		if _, err := os.Stat(fixture.runtimePIDs); !os.IsNotExist(err) {
			t.Fatalf("non-managed target launched managed runtime: %v", err)
		}
	})
}

const (
	modelCheckFixturePrompt = "Run the requested bounded behavior check."
	modelCheckFixtureSecret = "fixture-api-token-super-secret-123456789"
)

type modelCheckFixture struct {
	home        string
	project     string
	outputDir   string
	runtimePIDs string
	toolPIDs    string
	requests    string
	binDir      string
}

func newModelCheckFixture(t *testing.T, runtimeScript, scenario string) modelCheckFixture {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	runtimePIDs := filepath.Join(t.TempDir(), "runtime-pids")
	toolPIDs := filepath.Join(t.TempDir(), "tool-pids")
	requests := filepath.Join(t.TempDir(), "provider-requests.jsonl")
	outputDir := filepath.Join(t.TempDir(), "evidence")
	binDir := t.TempDir()
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	config := fmt.Sprintf(`
[agents.pi.primary_session]
profile = "profile"
pi_compatibility = %q

[agents.pi.profiles.profile]
provider = "local-provider"
model = "Model"
base_url = "http://127.0.0.1:%d/v1"
api = "openai-completions"
reasoning = false
input = ["text"]
context_window = 8192
max_tokens = 1024
thinking = "off"
requested_capabilities = ["text", "tools"]

[agents.pi.profiles.profile.lifecycle_log_retention]
max_count = 8
max_bytes = 1048576
max_envelope_bytes = 2097152
max_age_seconds = 4838400
create_timeout_seconds = 5
append_timeout_seconds = 5
close_timeout_seconds = 5
status_timeout_seconds = 5
maintenance_timeout_seconds = 5
max_scan_entries = 512
max_scan_control_bytes = 262144
max_mutations_per_operation = 8

[agents.pi.profiles.profile.compat]
supports_developer_role = false

[agents.pi.profiles.profile.runtime]
executable = "/usr/bin/python3"
argv = [%q, "--host", "127.0.0.1", "--port", %q, "--scenario", %q, "--runtime-pids", %q, "--tool-pids", %q, "--events-path", %q, "--requests-path", %q]
readiness_path = "/models"
startup_timeout_seconds = 5
shutdown_timeout_seconds = 1

[agents.targets.qwen]
vendor = "qwen"
environment = "pi"
model = "Model"
reasoning = "off"
profile = "profile"

[agents.targets.openai]
vendor = "openai"
environment = "codex"
model = "gpt-5.6-sol"
reasoning = "high"

[agents.entrypoints]
qwen-infra = "qwen"
openai-infra = "openai"
`, infra.PiCompatibilityV0842DarwinARM64, port, runtimeScript, strconv.Itoa(port), scenario, runtimePIDs, toolPIDs, filepath.Join(outputDir, "events.jsonl"), requests)
	writeMainCanonicalConfig(t, project, config)
	return modelCheckFixture{
		home:        home,
		project:     project,
		outputDir:   outputDir,
		runtimePIDs: runtimePIDs,
		toolPIDs:    toolPIDs,
		requests:    requests,
		binDir:      binDir,
	}
}

func runModelCheckProduction(t *testing.T, binary, piRoot string, fixture modelCheckFixture, extra ...string) (string, string, int) {
	t.Helper()
	args := []string{
		"model-check",
		"--target", "qwen-infra",
		"--prompt", modelCheckFixturePrompt,
		"--output-dir", fixture.outputDir,
	}
	args = append(args, extra...)
	command := exec.Command(binary, args...)
	command.Env = append(os.Environ(),
		"HOME="+fixture.home,
		"PATH="+fixture.binDir+":"+piRoot+":/usr/bin:/bin",
		callerCWDEnv+"="+fixture.project,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("start model-check production binary: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}
	return stdout.String(), stderr.String(), exitCode
}

func readProductionModelCheckSummary(t *testing.T, outputDir string) productionModelCheckSummary {
	t.Helper()
	data := mustReadFile(t, filepath.Join(outputDir, "summary.json"))
	var summary productionModelCheckSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("decode summary.json: %v\n%s", err, data)
	}
	return summary
}

func assertModelCheckPIDsGone(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pid evidence %s: %v", path, err)
	}
	for _, field := range strings.Fields(string(data)) {
		pid, err := strconv.Atoi(field)
		if err != nil {
			t.Fatalf("invalid pid %q in %s", field, path)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			err = syscall.Kill(pid, syscall.Signal(0))
			if errors.Is(err, syscall.ESRCH) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("owned process survived cleanup: pid=%d err=%v", pid, err)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func writeModelCheckRuntime(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model-check-runtime.py")
	mustWrite(t, path, `import os, sys

def value(name):
    return sys.argv[sys.argv.index(name) + 1]

runtime_pid_file = value("--runtime-pids")
open(runtime_pid_file, "w").write(f"{os.getpid()}\n")

import http.server, json, shlex, subprocess, time

port = int(value("--port"))
scenario = value("--scenario")
tool_pid_file = value("--tool-pids")
events_path = value("--events-path")
requests_path = value("--requests-path")

if scenario == "timeout":
    child = subprocess.Popen(["/bin/sh", "-c", "trap '' TERM; sleep 60"])
else:
    child = subprocess.Popen(["/bin/sleep", "60"])
with open(runtime_pid_file, "a") as runtime_pids:
    runtime_pids.write(f"{child.pid}\n")

if scenario == "malformed":
    watcher = '''import os, sys, time
path = sys.argv[1]
for _ in range(10000):
    try:
        body = open(path, "rb").read()
        if b'"type":"session"' in body:
            with open(path, "ab") as stream:
                stream.write(b"not-json\\n")
            break
    except FileNotFoundError:
        pass
    time.sleep(.001)
'''
    subprocess.Popen([sys.executable, "-c", watcher, events_path])

requests = 0

def chunk(delta, finish_reason=None):
    return {
        "id": "chatcmpl-fixture",
        "object": "chat.completion.chunk",
        "created": 1,
        "model": "Model",
        "choices": [{"index": 0, "delta": delta, "finish_reason": finish_reason}],
    }

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if scenario == "slow-ready":
            time.sleep(60)
            return
        body = json.dumps({"object": "list", "data": [{"id": "Model"}]}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        global requests
        requests += 1
        length = int(self.headers.get("Content-Length", "0"))
        request_body = self.rfile.read(length)
        with open(requests_path, "ab") as request_log:
            request_log.write(request_body + b"\n")
        if scenario == "timeout" and requests > 1:
            time.sleep(60)
            return
        if scenario in ("happy", "failed-tool", "timeout", "earlier-text") and requests == 1:
            if scenario in ("happy", "earlier-text"):
                name = "read"
                args = {"path": "fixture.txt"}
            elif scenario == "failed-tool":
                name = "read"
                args = {"path": "definitely-absent-model-check-fixture"}
            else:
                name = "bash"
                quoted = shlex.quote(tool_pid_file)
                command = f"echo $$ > {quoted}; (trap '' TERM; sleep 60) & echo $! >> {quoted}; wait"
                args = {"command": command}
            tool_delta = {
                "role": "assistant",
                "tool_calls": [{
                    "index": 0,
                    "id": "call_fixture",
                    "type": "function",
                    "function": {"name": name, "arguments": json.dumps(args)},
                }],
            }
            if scenario == "earlier-text":
                tool_delta["content"] = "EARLIER_ASSISTANT_TEXT_MUST_NOT_SATISFY_FINAL_EXPECTATION"
            self.send_stream([chunk(tool_delta), chunk({}, "tool_calls")])
            return
        text = "MODEL_CHECK_OK token=`+modelCheckFixtureSecret+`"
        if scenario == "long-text":
            text = "word " * 1200
        self.send_stream([chunk({"role": "assistant", "content": text}), chunk({}, "stop")])

    def send_stream(self, chunks):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.end_headers()
        for item in chunks:
            self.wfile.write(("data: " + json.dumps(item) + "\n\n").encode())
            self.wfile.flush()
        self.wfile.write(b"data: [DONE]\n\n")
        self.wfile.flush()

    def log_message(self, *args):
        pass

http.server.ThreadingHTTPServer(("127.0.0.1", port), Handler).serve_forever()
`)
	return path
}
