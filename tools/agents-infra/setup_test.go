package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSetupLocalAcceptsPrimaryFlagsAfterProjectDirectory(t *testing.T) {
	source := sourceRepoRoot(t)
	project := t.TempDir()

	captureStdout(t, func() {
		if err := runSetup([]string{
			"local",
			project,
			"--source-dir", source,
			"--codex-primary-model", "gpt-5.6-terra",
			"--codex-primary-reasoning-effort", "xhigh",
			"--codex-yolo-mode=false",
		}); err != nil {
			t.Fatalf("runSetup: %v", err)
		}
	})

	path := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	for _, want := range []string{
		"[agents.codex.primary_session]",
		"model = 'gpt-5.6-terra'",
		"reasoning_effort = 'xhigh'",
		"yolo_mode = false",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("project config missing %q:\n%s", want, data)
		}
	}
}

func TestRunSetupRejectsNonBooleanCodexYoloMode(t *testing.T) {
	err := runSetup([]string{
		"local",
		t.TempDir(),
		"--source-dir", sourceRepoRoot(t),
		"--codex-yolo-mode=yes",
	})
	if err == nil || !strings.Contains(err.Error(), "expected true or false") {
		t.Fatalf("runSetup error = %v, want strict boolean validation", err)
	}
}

func TestRunSetupGlobalRejectsPrimarySessionFlags(t *testing.T) {
	home := t.TempDir()
	err := runSetup([]string{
		"global",
		"--source-dir", sourceRepoRoot(t),
		"--home-dir", home,
		"--codex-primary-model", "gpt-5.6-terra",
	})
	if err == nil || !strings.Contains(err.Error(), "local-only") {
		t.Fatalf("runSetup global error = %v, want local-only rejection", err)
	}
	if _, statErr := os.Lstat(filepath.Join(home, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("runSetup global mutated destination before rejection: %v", statErr)
	}
}

func TestRunSetupLocalRejectsPrimarySessionFlagsForHomeTarget(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "set",
			args: []string{"--codex-primary-model", "gpt-5.6-terra"},
		},
		{
			name: "clear",
			args: []string{"--clear-codex-primary-session"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			path := filepath.Join(home, ".agents", ".configs", "project-config.toml")
			args := []string{"local", home, "--source-dir", sourceRepoRoot(t)}
			args = append(args, test.args...)

			err := runSetup(args)
			if err == nil {
				t.Fatal("runSetup unexpectedly accepted the ignored global project-config path")
			}
			for _, want := range []string{path, "field agents.codex.primary_session", "global"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("runSetup error %q missing %q", err, want)
				}
			}
			if _, statErr := os.Lstat(filepath.Join(home, ".agents")); !os.IsNotExist(statErr) {
				t.Fatalf("runSetup mutated HOME before rejection: %v", statErr)
			}
		})
	}
}

func sourceRepoRoot(t *testing.T) string {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root, err := filepath.Abs(filepath.Join(workingDir, "..", ".."))
	if err != nil {
		t.Fatalf("Abs(source repo): %v", err)
	}
	return root
}
