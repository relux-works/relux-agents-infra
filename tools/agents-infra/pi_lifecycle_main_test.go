//go:build !windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

// Production call site: runPi -> runPiLifecycleCLI. Status, dry-run, and
// confirmed retirement must resolve configuration and external filesystem
// evidence without looking up or launching Pi or its configured runtime.
func TestRunPiLifecycleOperatorIsNonLaunchingAndProjectsExactPlan(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(configPath))
	mustWrite(t, configPath, mainTestPiConfig("/definitely/not/a/runtime", 18021))
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/definitely/no/provider/bin")

	canonical, err := infra.CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := infra.ResolvePiStatePaths(cache, canonical, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := infra.CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	seedMainLifecycleRoot(t, paths)
	legacy := filepath.Join(paths.LogsDir, "legacy-cli.jsonl")
	if err := os.WriteFile(legacy, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	statusOutput := captureStdout(t, func() {
		if err := runPi([]string{"lifecycle", "status", "--project", project, "--profile", "profile", "--json"}); err != nil {
			t.Fatalf("lifecycle status: %v", err)
		}
	})
	var status infra.PiLifecycleLogStatus
	if err := json.Unmarshal([]byte(statusOutput), &status); err != nil {
		t.Fatal(err)
	}
	if !status.ScanComplete || status.LegacyCount != 1 || status.WithinPolicy || status.SoakReady {
		t.Fatalf("status=%+v", status)
	}

	dryRunOutput := captureStdout(t, func() {
		if err := runPi([]string{"lifecycle", "retire-legacy", "--project", project, "--profile", "profile", "--dry-run", "--json"}); err != nil {
			t.Fatalf("legacy dry-run: %v", err)
		}
	})
	var plan infra.PiLegacyRetirementPlan
	if err := json.Unmarshal([]byte(dryRunOutput), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.PlanHash == "" || len(plan.Candidates) != 1 || plan.Candidates[0].Path != "logs/legacy-cli.jsonl" {
		t.Fatalf("plan=%+v", plan)
	}
	if _, err := os.Lstat(legacy); err != nil {
		t.Fatalf("dry-run mutated legacy evidence: %v", err)
	}

	confirmOutput := captureStdout(t, func() {
		if err := runPi([]string{"lifecycle", "retire-legacy", "--project", project, "--profile", "profile", "--confirm", plan.PlanHash, "--json"}); err != nil {
			t.Fatalf("legacy confirmation: %v", err)
		}
	})
	var result infra.PiLegacyRetirementResult
	if err := json.Unmarshal([]byte(confirmOutput), &result); err != nil {
		t.Fatal(err)
	}
	if result.PlanHash != plan.PlanHash || result.RetiredCount != 1 || !result.Status.SoakReady {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
		t.Fatalf("confirmed legacy candidate survived: %v", err)
	}
}

// Production call site: runPi -> runPiLifecycleCLI ->
// PiLifecycleOperatorStatus. Foreign legacy-directory evidence must remain
// externally visible and must independently refuse whole-scan policy health;
// it is not interchangeable with an absent legacy candidate.
func TestRunPiLifecycleStatusRefusesForeignEvidence(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(configPath))
	mustWrite(t, configPath, mainTestPiConfig("/definitely/not/a/runtime", 18021))
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/definitely/no/provider/bin")

	canonical, err := infra.CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := infra.ResolvePiStatePaths(cache, canonical, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := infra.CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	seedMainLifecycleRoot(t, paths)
	foreign := filepath.Join(paths.LogsDir, "foreign-cli.jsonl")
	want := []byte("preserve foreign evidence\n")
	if err := os.WriteFile(foreign, want, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(foreign, 0o640); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		if err := runPi([]string{"lifecycle", "status", "--project", project, "--profile", "profile", "--json"}); err != nil {
			t.Fatalf("lifecycle status: %v", err)
		}
	})
	var status infra.PiLifecycleLogStatus
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		t.Fatal(err)
	}
	if !status.ScanComplete || status.ForeignCount != 1 || status.ForeignBytes != int64(len(want)) || status.LegacyCount != 0 || status.UnknownCount != 0 || status.WithinPolicy || status.SoakReady {
		t.Fatalf("foreign evidence published whole-scan health: %+v", status)
	}
	got, err := os.ReadFile(foreign)
	if err != nil || string(got) != string(want) {
		t.Fatalf("status mutated foreign evidence: data=%q err=%v", got, err)
	}
}

// Production call site: runPi -> runPiLifecycleCLI ->
// PiLifecycleOperatorStatus. The opaque continuation must advance the real
// bounded directory cursor; every continuation page remains a lower bound and
// can never publish whole-scan health.
func TestRunPiLifecycleStatusPaginatesWithoutLaunching(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(configPath))
	mustWrite(t, configPath, mainTestPiConfig("/definitely/not/a/runtime", 18021))
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/definitely/no/provider/bin")
	canonical, err := infra.CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := infra.ResolvePiStatePaths(cache, canonical, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := infra.CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	seedMainLifecycleRoot(t, paths)
	for index := 0; index < 700; index++ {
		name := filepath.Join(paths.LogsDir, fmt.Sprintf("legacy-%04d.jsonl", index))
		if err := os.WriteFile(name, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	continuation := ""
	for page := 0; page < 8; page++ {
		args := []string{"lifecycle", "status", "--project", project, "--profile", "profile", "--json"}
		if continuation != "" {
			args = append(args, "--continuation", continuation)
		}
		var statusErr error
		output := captureStdout(t, func() { statusErr = runPi(args) })
		var status infra.PiLifecycleLogStatus
		if err := json.Unmarshal([]byte(output), &status); err != nil {
			t.Fatal(err)
		}
		if status.ScanComplete || status.WithinPolicy || status.SoakReady || page > 0 && !status.LowerBound {
			t.Fatalf("page %d published whole-scan health: %+v", page, status)
		}
		if statusErr == nil {
			if status.Continuation != "" || !status.LowerBound {
				t.Fatalf("final continuation page=%+v", status)
			}
			return
		}
		var launch *infra.PiLaunchError
		if !errors.As(statusErr, &launch) || launch.Code != "lifecycle_log_scan_exhausted" || status.Continuation == "" || status.Continuation == continuation {
			t.Fatalf("page %d did not advance bounded cursor: status=%+v err=%v", page, status, statusErr)
		}
		continuation = status.Continuation
	}
	t.Fatal("lifecycle status continuation did not converge")
}

func seedMainLifecycleRoot(t *testing.T, paths infra.PiStatePaths) {
	t.Helper()
	if err := os.Mkdir(paths.LifecycleLogsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(paths.LifecycleLogsRoot, "entries"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"foreground.lock", "retention.lock"} {
		if err := os.WriteFile(filepath.Join(paths.LifecycleLogsRoot, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{
		"generation.json":        "{\"schema_version\":1,\"generation\":0,\"state\":\"even\",\"scope\":\"aggregate\"}\n",
		"legacy-generation.json": "{\"schema_version\":1,\"generation\":0,\"state\":\"even\",\"scope\":\"legacy\"}\n",
	} {
		if err := os.WriteFile(filepath.Join(paths.LifecycleLogsRoot, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
