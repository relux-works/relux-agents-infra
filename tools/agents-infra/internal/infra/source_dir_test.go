package infra

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// launcherBackendFixtureMain is the smallest program that is actually an
// agents-infra CLI: it answers `version` the way runVersion does. A fixture
// that only compiles would model a runtime the launcher cannot use, so it
// cannot stand in for one that it can.
const launcherBackendFixtureMain = `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("agents-infra fixture commit=none build_date=none")
		return
	}
	fmt.Fprintln(os.Stderr, "usage: agents-infra version")
	os.Exit(2)
}
`

// seedLauncherBackend writes the Go module the generated agents-infra launcher
// builds. It is part of the source contract, not decoration: without it the
// installed runtime's agents-infra command cannot start.
func seedLauncherBackend(t *testing.T, dir string) {
	t.Helper()
	mustMkdir(t, filepath.Join(dir, "tools", "agents-infra"))
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "go.mod"), "module example.com/agents-infra\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "main.go"), launcherBackendFixtureMain)
}

func writeSourceTree(t *testing.T, dir string) string {
	t.Helper()
	mustMkdir(t, filepath.Join(dir, ".instructions"))
	mustMkdir(t, filepath.Join(dir, ".configs"))
	mustMkdir(t, filepath.Join(dir, ".rules"))
	mustWrite(t, filepath.Join(dir, ".instructions", "INSTRUCTIONS.md"), "# Instructions\n")
	mustWrite(t, filepath.Join(dir, ".instructions", "AGENTS.md"), "# Agents\n")
	seedLauncherBackend(t, dir)
	return dir
}

func writeInstallState(t *testing.T, configDir, repoPath string) {
	t.Helper()
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, installStateFileName), `{"repoPath":"`+repoPath+`","binDir":"/tmp/bin"}`)
}

func TestResolveSourceDirUsesInstallStateRepoPathWithoutExplicitPath(t *testing.T) {
	home := t.TempDir()
	source := writeSourceTree(t, t.TempDir())
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)

	resolved, err := ResolveSourceDir(SourceDirRequest{
		Mode:              ModeLocal,
		ConfigDirOverride: configDir,
		HomeDir:           home,
		TargetAgentsDir:   filepath.Join(t.TempDir(), ".agents"),
	})
	if err != nil {
		t.Fatalf("ResolveSourceDir: %v", err)
	}
	if resolved != source {
		t.Fatalf("resolved = %q, want install-state repoPath %q", resolved, source)
	}
}

func TestResolveSourceDirFallsBackToInstalledRuntime(t *testing.T) {
	home := t.TempDir()
	runtimeDir := writeSourceTree(t, filepath.Join(home, ".agents"))

	resolved, err := ResolveSourceDir(SourceDirRequest{
		Mode:              ModeLocal,
		ConfigDirOverride: filepath.Join(home, "config-without-state"),
		HomeDir:           home,
		TargetAgentsDir:   filepath.Join(t.TempDir(), ".agents"),
	})
	if err != nil {
		t.Fatalf("ResolveSourceDir: %v", err)
	}
	if resolved != runtimeDir {
		t.Fatalf("resolved = %q, want installed runtime %q", resolved, runtimeDir)
	}
}

func TestResolveSourceDirSkipsStaleInstallStateRepoPath(t *testing.T) {
	home := t.TempDir()
	runtimeDir := writeSourceTree(t, filepath.Join(home, ".agents"))
	configDir := filepath.Join(home, "config")
	stale := filepath.Join(t.TempDir(), "moved-away")
	writeInstallState(t, configDir, stale)

	resolved, err := ResolveSourceDir(SourceDirRequest{
		Mode:              ModeLocal,
		ConfigDirOverride: configDir,
		HomeDir:           home,
		TargetAgentsDir:   filepath.Join(t.TempDir(), ".agents"),
	})
	if err != nil {
		t.Fatalf("ResolveSourceDir: %v", err)
	}
	if resolved != runtimeDir {
		t.Fatalf("resolved = %q, want installed runtime %q after a stale repoPath", resolved, runtimeDir)
	}
}

// Negative: an explicitly requested source that is not an agents-infra tree must
// fail loudly, even when a perfectly good fallback is available. Silently using
// the fallback would install something the caller never asked for.
func TestResolveSourceDirRefusesWrongExplicitSourceInsteadOfFallingBack(t *testing.T) {
	home := t.TempDir()
	writeSourceTree(t, filepath.Join(home, ".agents"))
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, writeSourceTree(t, t.TempDir()))

	wrong := t.TempDir()
	mustMkdir(t, filepath.Join(wrong, "src"))

	for _, test := range []struct {
		name    string
		request SourceDirRequest
		origin  string
	}{
		{
			name:    "flag",
			request: SourceDirRequest{Mode: ModeLocal, Flag: wrong},
			origin:  string(SourceDirOriginFlag),
		},
		{
			name:    "env",
			request: SourceDirRequest{Mode: ModeLocal, Env: wrong},
			origin:  string(SourceDirOriginEnv),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := test.request
			request.ConfigDirOverride = configDir
			request.HomeDir = home
			request.TargetAgentsDir = filepath.Join(t.TempDir(), ".agents")

			resolved, err := ResolveSourceDir(request)
			if err == nil {
				t.Fatalf("ResolveSourceDir accepted a wrong explicit source and resolved %q", resolved)
			}
			var sourceErr *SourceDirError
			if !errors.As(err, &sourceErr) {
				t.Fatalf("error type = %T, want *SourceDirError", err)
			}
			for _, want := range []string{test.origin, wrong, ".instructions/INSTRUCTIONS.md", ".configs"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q missing %q", err, want)
				}
			}
		})
	}
}

func TestResolveSourceDirNamesEveryCandidateWhenNothingIsUsable(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, "config")

	_, err := ResolveSourceDir(SourceDirRequest{
		Mode:              ModeLocal,
		ConfigDirOverride: configDir,
		HomeDir:           home,
		TargetAgentsDir:   filepath.Join(t.TempDir(), ".agents"),
	})
	if err == nil {
		t.Fatal("ResolveSourceDir resolved a source tree from an empty host")
	}
	for _, want := range []string{
		string(SourceDirOriginFlag),
		string(SourceDirOriginEnv),
		string(SourceDirOriginInstallState),
		string(SourceDirOriginInstalledRuntime),
		filepath.Join(home, ".agents"),
		"--source-dir DIR",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestResolveSourceDirRejectsCandidateContainingTheDestination(t *testing.T) {
	home := t.TempDir()
	runtimeDir := writeSourceTree(t, filepath.Join(home, ".agents"))

	_, err := ResolveSourceDir(SourceDirRequest{
		Mode:              ModeGlobal,
		ConfigDirOverride: filepath.Join(home, "config-without-state"),
		HomeDir:           home,
		TargetAgentsDir:   runtimeDir,
	})
	if err == nil {
		t.Fatal("ResolveSourceDir accepted a source tree that contains its own destination")
	}
	if !strings.Contains(err.Error(), "would sync into itself") {
		t.Fatalf("error %q does not name the self-sync rejection", err)
	}
}

func TestResolveSourceDirPlatformInstallStateLocations(t *testing.T) {
	t.Run("darwin", func(t *testing.T) {
		home := t.TempDir()
		source := writeSourceTree(t, t.TempDir())
		writeInstallState(t, filepath.Join(home, "Library", "Application Support", "agents-infra"), source)

		resolved, err := ResolveSourceDir(SourceDirRequest{Mode: ModeLocal, GOOS: "darwin", HomeDir: home})
		if err != nil {
			t.Fatalf("ResolveSourceDir: %v", err)
		}
		if resolved != source {
			t.Fatalf("resolved = %q, want %q", resolved, source)
		}
	})

	t.Run("xdg", func(t *testing.T) {
		home := t.TempDir()
		xdg := t.TempDir()
		source := writeSourceTree(t, t.TempDir())
		writeInstallState(t, filepath.Join(xdg, "agents-infra"), source)

		resolved, err := ResolveSourceDir(SourceDirRequest{Mode: ModeLocal, GOOS: "linux", XDGConfigHome: xdg, HomeDir: home})
		if err != nil {
			t.Fatalf("ResolveSourceDir: %v", err)
		}
		if resolved != source {
			t.Fatalf("resolved = %q, want %q", resolved, source)
		}
	})
}

// Negative: Setup is the production call site that copies a tree into a
// destination. It must refuse a missing or non-agents-infra source instead of
// syncing whatever the caller handed it.
func TestSetupRefusesUnusableSourceDirWithoutTouchingDestination(t *testing.T) {
	unrelated := t.TempDir()
	mustWrite(t, filepath.Join(unrelated, "README.md"), "not agents-infra\n")

	for _, test := range []struct {
		name      string
		sourceDir string
		want      []string
	}{
		{
			name:      "missing",
			sourceDir: "",
			want:      []string{"source dir is required", "not set"},
		},
		{
			name:      "wrong tree",
			sourceDir: unrelated,
			want:      []string{"not a usable agents-infra source tree", ".instructions/INSTRUCTIONS.md", ".configs"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			project := t.TempDir()
			layout, err := LocalLayout(test.sourceDir, project)
			if err != nil {
				t.Fatalf("LocalLayout: %v", err)
			}

			err = Setup(Options{Layout: layout, NoSync: true, Stdout: io.Discard})
			if err == nil {
				t.Fatal("Setup accepted an unusable source dir")
			}
			for _, want := range test.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q missing %q", err, want)
				}
			}
			if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
				t.Fatalf("Setup mutated the destination before rejecting the source: %v", statErr)
			}
		})
	}
}

func TestSetupRefusesSourceDirContainingItsOwnDestination(t *testing.T) {
	project := writeSourceTree(t, t.TempDir())
	layout, err := LocalLayout(project, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{Layout: layout, NoSync: true, Stdout: io.Discard})
	if err == nil {
		t.Fatal("Setup accepted a source dir that contains its own destination")
	}
	if !strings.Contains(err.Error(), "would sync into itself") {
		t.Fatalf("error %q does not name the self-sync rejection", err)
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("Setup mutated the destination before rejecting the source: %v", statErr)
	}
}
