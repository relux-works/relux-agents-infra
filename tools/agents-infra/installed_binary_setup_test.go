package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildInstalledBinary produces the same artifact scripts/setup.sh drops into
// ~/.local/bin: a plain binary with no source path baked into it.
func buildInstalledBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "agents-infra")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	return binary
}

// runInstalledBinary invokes the built binary the way a globally installed
// launcher is invoked: no --source-dir, no AGENTS_INFRA_SOURCE_DIR, and a home
// that only carries what an install actually leaves behind.
func runInstalledBinary(t *testing.T, binary, home, configDir string, args ...string) (string, error) {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Env = append(os.Environ(),
		"HOME="+home,
		"AGENTS_INFRA_CONFIG_DIR="+configDir,
		"AGENTS_INFRA_SOURCE_DIR=",
		"AGENTS_INFRA_CALLER_CWD=",
	)
	output, err := command.CombinedOutput()
	return string(output), err
}

func writeInstallState(t *testing.T, configDir, repoPath string) {
	t.Helper()
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, "install.json"), `{"repoPath":"`+repoPath+`","binDir":"`+filepath.Join(repoPath, "bin")+`"}`)
}

func TestInstalledBinarySetupLocalResolvesSourceFromInstallState(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, sourceRepoRoot(t))
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	for _, want := range []string{
		filepath.Join(project, ".agents", ".instructions", "INSTRUCTIONS.md"),
		filepath.Join(project, ".claude", "instructions"),
		filepath.Join(project, "AGENTS.md"),
	} {
		if _, statErr := os.Lstat(want); statErr != nil {
			t.Fatalf("installed binary setup did not produce %s: %v\n%s", want, statErr, output)
		}
	}
}

// seedRuntimeSource writes a complete agents-infra source tree: the instruction
// entrypoints, the config and rules trees, and the Go module the generated
// launcher builds. Anything less is not a source tree, it is a tree that looks
// like one — see the marker-valid negatives below.
func seedRuntimeSource(t *testing.T, dir string) string {
	t.Helper()
	mustMkdir(t, filepath.Join(dir, ".instructions"))
	mustMkdir(t, filepath.Join(dir, ".configs"))
	mustMkdir(t, filepath.Join(dir, ".rules"))
	mustWrite(t, filepath.Join(dir, ".instructions", "INSTRUCTIONS.md"), "# Instructions\n")
	mustWrite(t, filepath.Join(dir, ".instructions", "AGENTS.md"), "# Agents\n")
	mustMkdir(t, filepath.Join(dir, "tools", "agents-infra"))
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "go.mod"), "module example.com/agents-infra\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "main.go"), "package main\n\nfunc main() {}\n")
	return dir
}

func TestInstalledBinarySetupLocalResolvesInstalledRuntimeWithoutInstallState(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	seedRuntimeSource(t, filepath.Join(home, ".agents"))
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, filepath.Join(home, "config-without-state"), "setup", "local", project)
	if err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	marker := filepath.Join(project, ".agents", ".instructions", "INSTRUCTIONS.md")
	if _, statErr := os.Lstat(marker); statErr != nil {
		t.Fatalf("installed binary setup did not sync the installed runtime: %v\n%s", statErr, output)
	}
	// A run that reports success must leave a runtime that verifies, otherwise
	// exit zero is the only evidence there is.
	if verifyOutput, verifyErr := runInstalledBinary(t, binary, home, filepath.Join(home, "config-without-state"), "verify", "local", project); verifyErr != nil {
		t.Fatalf("installed binary could not verify the runtime it just installed: %v\n%s", verifyErr, verifyOutput)
	}
}

// Negative: the source tree carries every historical marker — both instruction
// entrypoints, .configs, .rules — but not the Go module the generated launcher
// builds. Setup used to exit zero here, print a full install log, and mint a
// launcher that failed on first use. It must now refuse, and must not leave a
// runtime behind for a caller to mistake for a working one.
func TestInstalledBinarySetupLocalRefusesMarkerValidSourceWithoutLauncherBackend(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, t.TempDir())
	if err := os.RemoveAll(filepath.Join(source, "tools")); err != nil {
		t.Fatalf("RemoveAll(tools): %v", err)
	}
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err == nil {
		t.Fatalf("installed binary accepted a source without the launcher backend\n%s", output)
	}
	for _, want := range []string{
		"tools/agents-infra/go.mod",
		"tools/agents-infra/main.go",
		"generated agents-infra launcher builds",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("failure output missing %q:\n%s", want, output)
		}
	}
	assertNoFalselyUsableRuntime(t, binary, home, configDir, project, output)
}

// Negative: a tree whose instruction entrypoint pulls in modules it does not
// ship passes every file-existence marker and then fails half way through the
// render, after the destination has already been rewritten. The include closure
// has to be proven up front.
func TestInstalledBinarySetupLocalRefusesSourceWithUnshippedInstructionInclude(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, t.TempDir())
	mustWrite(t, filepath.Join(source, ".instructions", "AGENTS.md"),
		"# Agents\n\n@~/.agents/.instructions/INSTRUCTIONS_MISSING.md\n")
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err == nil {
		t.Fatalf("installed binary accepted a source with an unshipped instruction include\n%s", output)
	}
	if !strings.Contains(output, "INSTRUCTIONS_MISSING.md") {
		t.Fatalf("failure output does not name the missing instruction module:\n%s", output)
	}
	// Failing eventually is not the property under test — the render would do
	// that on its own, after the destination had already been rewritten. The
	// closure has to be proven before anything is written.
	if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("setup rewrote the destination before rejecting an unshippable include: %v", statErr)
	}
	assertNoFalselyUsableRuntime(t, binary, home, configDir, project, output)
}

// Negative: a destination that carries a complete-looking tree but was never
// completed by a verified run must not verify. This is the partial-write shape:
// the directory exists, the assets are there, and none of that is evidence that
// a run finished.
func TestInstalledBinaryVerifyLocalRefusesRuntimeWithoutCompletedInstall(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	project := t.TempDir()
	seedRuntimeSource(t, filepath.Join(project, ".agents"))

	output, err := runInstalledBinary(t, binary, home, filepath.Join(home, "config-without-state"), "verify", "local", project)
	if err == nil {
		t.Fatalf("verify accepted a runtime no completed run produced\n%s", output)
	}
	if !strings.Contains(output, "no completed-install receipt") {
		t.Fatalf("failure output does not name the missing receipt:\n%s", output)
	}
}

// assertNoFalselyUsableRuntime is the check a refusal owes its caller: whatever
// the run wrote before it stopped, the destination must not pass verification.
func assertNoFalselyUsableRuntime(t *testing.T, binary, home, configDir, project, setupOutput string) {
	t.Helper()
	verifyOutput, verifyErr := runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
	if verifyErr == nil {
		t.Fatalf("a refused setup left a runtime that verifies as usable\nsetup:\n%s\nverify:\n%s", setupOutput, verifyOutput)
	}
}

// Negative: with nothing to resolve, the installed binary must fail and name the
// candidates it tried instead of quietly setting up an empty runtime.
func TestInstalledBinarySetupLocalRefusesWhenNoSourceIsResolvable(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, filepath.Join(home, "config-without-state"), "setup", "local", project)
	if err == nil {
		t.Fatalf("installed binary set up a project without any resolvable source tree\n%s", output)
	}
	for _, want := range []string{
		"source dir is required",
		"install state repoPath",
		"installed runtime",
		"--source-dir DIR",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("failure output missing %q:\n%s", want, output)
		}
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("refused setup still wrote into the project: %v", statErr)
	}
}

// Negative: an explicit but wrong --source-dir must not silently fall back to
// the perfectly good install state sitting next to it.
func TestInstalledBinarySetupLocalRefusesWrongExplicitSourceDir(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, sourceRepoRoot(t))
	project := t.TempDir()
	wrong := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project, "--source-dir", wrong)
	if err == nil {
		t.Fatalf("installed binary accepted a wrong --source-dir\n%s", output)
	}
	for _, want := range []string{wrong, ".instructions/INSTRUCTIONS.md", ".configs"} {
		if !strings.Contains(output, want) {
			t.Fatalf("failure output missing %q:\n%s", want, output)
		}
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("refused setup still wrote into the project: %v", statErr)
	}
}
