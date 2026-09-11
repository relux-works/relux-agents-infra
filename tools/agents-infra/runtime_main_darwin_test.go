//go:build darwin

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

type productionRuntimeAuthorizationFrame struct {
	Schema          string `json:"schema"`
	ProtocolVersion int    `json:"protocol_version"`
	RuntimeKey      string `json:"runtime_key"`
	LauncherPID     int    `json:"launcher_pid"`
	ExecPlanDigest  string `json:"exec_plan_digest"`
}

// Production call site: runRuntime's runtime-launch dispatch in main.go,
// exercised through the built agents-infra binary with caller-minted fd 3
// authorization evidence.
func TestProductionRuntimeLaunchRefusesModelOriginEnvironment(t *testing.T) {
	binary := buildInstalledBinary(t)
	home, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	project := t.TempDir()
	marker := filepath.Join(t.TempDir(), "runtime-started")
	runtimeExecutable := filepath.Join(t.TempDir(), "runtime")
	mustWrite(t, runtimeExecutable, "#!/bin/sh\nprintf launched > "+strconv.Quote(marker)+"\n")
	if err := os.Chmod(runtimeExecutable, 0o755); err != nil {
		t.Fatal(err)
	}
	port := 18033
	runtimeArgv := []string{"serve", "--model", "Model", "--host", "127.0.0.1", "--port", strconv.Itoa(port)}
	body := mainTestPiConfig(runtimeExecutable, port) + `
[agents.pi.profiles.profile.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
max_segment_bytes = 1048576
max_segments = 7
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
resource_pressure_mode = "disabled"
`
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(configPath))
	mustWrite(t, configPath, body)
	status, err := infra.SharedRuntimeStatusReport(infra.SharedRuntimeOperatorOptions{
		ProjectDir: project,
		HomeDir:    home,
		CacheRoot:  cache,
		Profile:    "profile",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := infra.CreateSharedRuntimeTree(status.Paths); err != nil {
		t.Fatal(err)
	}
	execPlanDigest := infra.SharedRuntimeExecPlanDigest(infra.PiProfile{Runtime: infra.PiRuntime{
		Executable:            runtimeExecutable,
		Argv:                  runtimeArgv,
		StartupTimeoutSeconds: 5,
	}}, status.Paths.RuntimeCWD)

	launch := func(t *testing.T, extraEnv ...string) (string, error) {
		t.Helper()
		readEnd, writeEnd, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command(binary, "runtime", "runtime-launch", "--runtime-key", status.RuntimeKey, "--profile-project", project, "--profile", "profile")
		command.Dir = status.Paths.RuntimeCWD
		command.Env = append([]string{"HOME=" + home}, extraEnv...)
		command.ExtraFiles = []*os.File{readEnd}
		var output bytes.Buffer
		command.Stdout = &output
		command.Stderr = &output
		if err := command.Start(); err != nil {
			_ = readEnd.Close()
			_ = writeEnd.Close()
			t.Fatal(err)
		}
		_ = readEnd.Close()
		frame := productionRuntimeAuthorizationFrame{
			Schema:          "agents-infra.pi.shared-runtime.auth.v1",
			ProtocolVersion: infra.SharedRuntimeProtocolVersion,
			RuntimeKey:      status.RuntimeKey,
			LauncherPID:     command.Process.Pid,
			ExecPlanDigest:  execPlanDigest,
		}
		if err := json.NewEncoder(writeEnd).Encode(frame); err != nil {
			_ = writeEnd.Close()
			_ = command.Process.Kill()
			_ = command.Wait()
			t.Fatal(err)
		}
		if err := writeEnd.Close(); err != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			t.Fatal(err)
		}
		waitErr := command.Wait()
		return output.String(), waitErr
	}

	t.Run("clean control", func(t *testing.T) {
		output, err := launch(t)
		if err != nil {
			t.Fatalf("clean runtime-launch failed: %v\n%s", err, output)
		}
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("clean runtime-launch did not exec target: %v", err)
		}
	})
	for _, name := range []string{"HF_ENDPOINT", "MODEL_ENDPOINT"} {
		t.Run(name, func(t *testing.T) {
			_ = os.Remove(marker)
			secret := "https://caller-origin.invalid/" + strings.ToLower(name)
			output, err := launch(t, name+"="+secret)
			want := "runtime-affecting environment name " + strconv.Quote(name) + " is denied"
			if err == nil || strings.TrimSpace(output) != want {
				t.Fatalf("production runtime-launch admitted %s: err=%v output=%q", name, err, output)
			}
			if strings.Contains(output, secret) {
				t.Fatalf("production runtime-launch leaked %s value: %q", name, output)
			}
			if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
				t.Fatalf("production runtime-launch executed target with %s: %v", name, statErr)
			}
		})
	}
}

func TestRunRuntimeStatusJSONIsAbsentAndSideEffectFree(t *testing.T) {
	project := t.TempDir()
	home, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(configPath))
	body := mainTestPiConfig("/bin/echo", 18021) + `
[agents.pi.profiles.profile.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
max_segment_bytes = 1048576
max_segments = 7
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
resource_pressure_mode = "disabled"
`
	mustWrite(t, configPath, body)
	t.Setenv("HOME", home)
	output := captureStdout(t, func() {
		if err := runRuntime([]string{"status", "--project", project, "--profile", "profile", "--json"}); err != nil {
			t.Fatalf("runtime status: %v", err)
		}
	})
	var status infra.SharedRuntimeStatus
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		t.Fatal(err)
	}
	if status.Broker.State != "absent" || status.Sharing.Configured.Mode != "shared" || status.Sharing.Effective != nil {
		t.Fatalf("absent status=%#v", status)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"restart_count", "restart_not_before", "quarantined_until",
		"last_readiness_match", "half_open",
	} {
		if _, present := fields[name]; !present {
			t.Fatalf("runtime status JSON omitted %q: %s", name, output)
		}
	}
	if _, err := os.Stat(status.Paths.Root); !os.IsNotExist(err) {
		t.Fatalf("runtime status created shared state: %v", err)
	}
}

func TestRunRuntimeManualQuarantineRoundTripUsesResolvedLedger(t *testing.T) {
	project := t.TempDir()
	home, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(configPath))
	body := mainTestPiConfig("/bin/echo", 18022) + `
[agents.pi.profiles.profile.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
max_segment_bytes = 1048576
max_segments = 7
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
resource_pressure_mode = "disabled"
`
	mustWrite(t, configPath, body)
	t.Setenv("HOME", home)
	for _, tc := range []struct {
		command string
		want    bool
	}{{"quarantine", true}, {"unquarantine", false}} {
		output := captureStdout(t, func() {
			if err := runRuntime([]string{tc.command, "--project", project, "--profile", "profile"}); err != nil {
				t.Fatalf("runtime %s: %v", tc.command, err)
			}
		})
		var ledger infra.SharedRuntimeRestartLedger
		if err := json.Unmarshal([]byte(output), &ledger); err != nil {
			t.Fatal(err)
		}
		if ledger.ManualQuarantine != tc.want {
			t.Fatalf("runtime %s manual_quarantine=%t want=%t", tc.command, ledger.ManualQuarantine, tc.want)
		}
	}
}
