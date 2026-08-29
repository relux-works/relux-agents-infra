//go:build darwin

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

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
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
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
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
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
