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
	// Setup builds the launcher backend, and an isolated HOME would otherwise
	// point the Go toolchain at an empty module cache inside a temp dir: the
	// build would go to the network and the caches would outlive the test. An
	// operator's HOME carries these; the harness has to model that, not invent
	// a colder machine than the one under test.
	command.Env = append(command.Env, sharedGoCacheEnv(t)...)
	output, err := command.CombinedOutput()
	return string(output), err
}

// sharedGoCacheEnv reports the Go cache locations of the machine running the
// tests, so a child process given an isolated HOME still builds against the
// caches a real operator would have.
func sharedGoCacheEnv(t *testing.T) []string {
	t.Helper()
	output, err := exec.Command("go", "env", "GOPATH", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		t.Fatalf("go env: %v", err)
	}
	values := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(values) != 3 {
		t.Fatalf("go env returned %d values, want 3: %q", len(values), output)
	}
	return []string{
		"GOPATH=" + values[0],
		"GOCACHE=" + values[1],
		"GOMODCACHE=" + values[2],
	}
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
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "main.go"), runnableLauncherBackendMain)
	return dir
}

// runnableLauncherBackendMain is the smallest program that is actually an
// agents-infra CLI: it answers `version` the way runVersion does. A fixture
// that merely compiles models a runtime the launcher cannot use, so it cannot
// stand in for one that it can.
const runnableLauncherBackendMain = `package main

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

// Negative: this is the same forgery one level deeper. The tree carries the
// real go.mod, go.sum and main.go — every path the contract names, from the
// actual module — and is missing the internal packages that module imports. A
// presence list cannot tell the two apart; `go build .` can, and that is what
// the generated launcher runs on every invocation.
func TestInstalledBinarySetupLocalRefusesLauncherBackendThatCannotBuild(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, t.TempDir())
	seedRealLauncherBackendWithoutItsPackages(t, source)
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err == nil {
		t.Fatalf("installed binary accepted a launcher backend that cannot build\n%s", output)
	}
	for _, want := range []string{
		"go build",
		filepath.Join(source, "tools", "agents-infra"),
		"internal/infra",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("failure output missing %q:\n%s", want, output)
		}
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("setup wrote the destination before rejecting an unbuildable backend: %v", statErr)
	}
	assertNoFalselyUsableRuntime(t, binary, home, configDir, project, output)
}

// seedRealLauncherBackendWithoutItsPackages narrows the module rather than
// removing it: the three files the contract and the receipt read stay, taken
// from the real repository, and only the packages `go build` needs are absent.
// A check that still passes here is checking names.
func seedRealLauncherBackendWithoutItsPackages(t *testing.T, source string) {
	t.Helper()
	realModule := filepath.Join(sourceRepoRoot(t), "tools", "agents-infra")
	target := filepath.Join(source, "tools", "agents-infra")
	mustMkdir(t, target)
	for _, name := range []string{"go.mod", "go.sum", "main.go"} {
		body, err := os.ReadFile(filepath.Join(realModule, name))
		if err != nil {
			t.Fatalf("read %s from the real module: %v", name, err)
		}
		mustWrite(t, filepath.Join(target, name), string(body))
	}
	for _, name := range []string{"go.mod", "main.go"} {
		if _, err := os.Lstat(filepath.Join(target, name)); err != nil {
			t.Fatalf("the negative must keep %s in place, not remove it: %v", name, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(target, "internal")); !os.IsNotExist(err) {
		t.Fatalf("the forged module must not carry internal packages: %v", err)
	}
}

// Negative: the forgery one level deeper again. Nothing is missing and nothing
// fails to compile — `go build .` exits zero over a complete module. The program
// it produces exits 42, which is what the launcher execs. A build-only
// attestation mints a receipt here and hands back a runtime whose very first
// command fails.
func TestInstalledBinarySetupLocalRefusesLauncherBackendThatBuildsButDoesNotStart(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, t.TempDir())
	seedLauncherBackendThatBuildsButDoesNotStart(t, source)
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err == nil {
		t.Fatalf("installed binary accepted a launcher backend that builds but does not start\n%s", output)
	}
	for _, want := range []string{"version", "exit status 42"} {
		if !strings.Contains(output, want) {
			t.Fatalf("failure output missing %q:\n%s", want, output)
		}
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("setup wrote the destination before rejecting a backend that does not start: %v", statErr)
	}
	assertNoFalselyUsableRuntime(t, binary, home, configDir, project, output)
}

// Negative, the "preserve" half of the same contract: a runtime that installed
// and verified cleanly must stop verifying once its launcher backend stops
// producing a binary that starts. A receipt is evidence of a run that passed,
// not a standing licence.
func TestInstalledBinaryVerifyLocalRefusesRuntimeWhoseLauncherStoppedStarting(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, t.TempDir())
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	// Sanity: the same check accepts the runtime it just installed, so the
	// failure below is the break and not a gate that refuses everything.
	if verifyOutput, verifyErr := runInstalledBinary(t, binary, home, configDir, "verify", "local", project); verifyErr != nil {
		t.Fatalf("verify rejected the runtime it just installed: %v\n%s", verifyErr, verifyOutput)
	}

	seedLauncherBackendThatBuildsButDoesNotStart(t, source)

	verifyOutput, verifyErr := runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
	if verifyErr == nil {
		t.Fatalf("verify preserved a usable verdict for a launcher that no longer starts\n%s", verifyOutput)
	}
	for _, want := range []string{"cannot start", "exit status 42"} {
		if !strings.Contains(verifyOutput, want) {
			t.Fatalf("failure output missing %q:\n%s", want, verifyOutput)
		}
	}
}

// seedLauncherBackendThatBuildsButDoesNotStart narrows the module to exactly one
// broken property. The self-check is the point: if this fixture ever stopped
// compiling it would quietly collapse into the weaker build negative above and
// the startup gate would go untested.
func seedLauncherBackendThatBuildsButDoesNotStart(t *testing.T, source string) {
	t.Helper()
	target := filepath.Join(source, "tools", "agents-infra")
	mustMkdir(t, target)
	mustWrite(t, filepath.Join(target, "go.mod"), "module example.com/agents-infra\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(target, "main.go"), "package main\n\nimport \"os\"\n\nfunc main() { os.Exit(42) }\n")
	probe := filepath.Join(t.TempDir(), "probe")
	build := exec.Command("go", "build", "-o", probe, ".")
	build.Dir = target
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("the startup negative must still compile, otherwise it only re-tests the build gate: %v\n%s", err, output)
	}
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
