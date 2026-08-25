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
	if _, err := os.Stat(status.Paths.Root); !os.IsNotExist(err) {
		t.Fatalf("runtime status created shared state: %v", err)
	}
}
