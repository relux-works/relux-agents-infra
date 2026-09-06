package infra

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The ceiling belongs to task-board, not the infra TOML parser. Invoke its
// production preflight on the shipped file, without reconstructing policy in a
// fixture or inheriting the developer's board/run selection. Missing tooling is
// a failure: skipping would silently remove coverage of the shipped authority.
func TestShippedCodexPolicyAdmitsAstraWithMediumCeilingAndSolFallback(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "task-board", "q", "project_config(view=spawn-preflight, role=developer, agent=codex)")
	cmd.Dir = t.TempDir()
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TASK_BOARD_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env,
		"TASK_BOARD_CONFIG="+filepath.Join(root, "task-board.config.json"),
		"TASK_BOARD_DIR="+t.TempDir())
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("task-board production preflight: %v", err)
	}
	var report struct {
		Ceiling struct {
			Configured bool   `json:"configured"`
			Effort     string `json:"reasoning_effort"`
			Criterion  string `json:"reasoning_effort_criterion"`
			Resolution string `json:"resolution_model"`
			Pairs      struct {
				Models []struct {
					ID      string   `json:"id"`
					Efforts []string `json:"efforts"`
				} `json:"models"`
			} `json:"admitted_pairs"`
		} `json:"resolved_role_ceiling"`
	}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatal(err)
	}
	c := report.Ceiling
	if !c.Configured || c.Effort != "medium" || c.Criterion != "less_or_equal" || c.Resolution != "gpt-5.6-sol" {
		t.Fatalf("shipped Codex ceiling = %+v; want configured medium/less_or_equal, resolution gpt-5.6-sol", c)
	}
	got := map[string][]string{}
	for _, model := range c.Pairs.Models {
		got[model.ID] = model.Efforts
	}
	// Exact equality proves both admission through medium and exclusion of high
	// and every other unadmitted effort/model, using the production resolver.
	want := map[string][]string{
		"gpt-6-astra": {"low", "medium"},
		"gpt-5.6-sol": {"low", "medium"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shipped admitted pairs = %v, want %v", got, want)
	}
	t.Run("native_fallback", TestSetupGlobalPreservesRepositorySolFallbackWithReasoningEffort)
}
