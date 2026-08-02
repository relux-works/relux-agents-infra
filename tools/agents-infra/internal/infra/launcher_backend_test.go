package infra

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// breakLauncherBackendBuild narrows the source tree instead of deleting from
// it: both launcher-backend assets stay exactly where the contract names them,
// go.mod is untouched, and only the property the launcher actually depends on —
// that `go build .` completes — is broken. A check that passes here is checking
// for file names, not for a runtime that can start.
func breakLauncherBackendBuild(t *testing.T, sourceDir string) {
	t.Helper()
	mustWrite(t, filepath.Join(sourceDir, "tools", "agents-infra", "main.go"),
		"package main\n\nimport _ \"example.com/agents-infra/internal/missing\"\n\nfunc main() {}\n")
	assertLauncherBackendAssetsIntact(t, sourceDir)
}

// breakLauncherBackendStartup narrows one step further than
// breakLauncherBackendBuild: every named asset is present *and* `go build .`
// succeeds. Only the property the consumer ultimately depends on — that the
// binary the launcher execs starts — is broken. A check that passes here is
// checking that a compiler ran, not that a CLI exists.
func breakLauncherBackendStartup(t *testing.T, sourceDir string) {
	t.Helper()
	mustWrite(t, filepath.Join(sourceDir, "tools", "agents-infra", "main.go"),
		"package main\n\nimport \"os\"\n\nfunc main() { os.Exit(42) }\n")
	assertLauncherBackendAssetsIntact(t, sourceDir)
	assertLauncherBackendCompiles(t, sourceDir)
}

// breakLauncherBackendIdentity keeps a program that builds and exits zero, and
// removes only the claim that it is agents-infra. This is the shape a capability
// claim has to reject: exit zero from *something* is not evidence that the CLI
// the launcher promises is there.
func breakLauncherBackendIdentity(t *testing.T, sourceDir string) {
	t.Helper()
	mustWrite(t, filepath.Join(sourceDir, "tools", "agents-infra", "main.go"),
		"package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"ok\") }\n")
	assertLauncherBackendAssetsIntact(t, sourceDir)
	assertLauncherBackendCompiles(t, sourceDir)
}

func assertLauncherBackendAssetsIntact(t *testing.T, sourceDir string) {
	t.Helper()
	for _, asset := range sourceAssets {
		if !asset.launcherBackend {
			continue
		}
		if !pathExists(filepath.Join(sourceDir, asset.path)) {
			t.Fatalf("the negative removed %s; it must narrow the source, not empty it", asset.path)
		}
	}
}

// assertLauncherBackendCompiles is what keeps a startup negative from silently
// degrading into the weaker build negative it is meant to go past.
func assertLauncherBackendCompiles(t *testing.T, sourceDir string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "probe")
	build := exec.Command("go", "build", "-o", out, ".")
	build.Dir = launcherBackendModuleDir(sourceDir)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("the startup negative must still compile, otherwise it only re-tests the build gate: %v\n%s", err, output)
	}
}

// Negative: the source carries every named asset and still cannot produce a
// working launcher. Setup must refuse — before it writes anything — rather than
// install a runtime whose agents-infra command fails on first use.
func TestSetupRefusesSourceWhoseLauncherBackendCannotBuild(t *testing.T) {
	source := seedSourceRepo(t)
	breakLauncherBackendBuild(t, source)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{Layout: layout, Stdout: io.Discard})
	if err == nil {
		t.Fatal("Setup installed a runtime whose launcher backend does not build")
	}
	var sourceErr *SourceDirError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error %v is not a typed SourceDirError", err)
	}
	for _, want := range []string{
		"go build",
		launcherBackendModuleDir(source),
		"internal/missing",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not name %q", err, want)
		}
	}
	// Failing eventually is not the property under test: the launcher would do
	// that on first use, after the destination had been written and attested.
	if _, statErr := os.Lstat(layout.AgentsDir); !os.IsNotExist(statErr) {
		t.Fatalf("setup wrote the destination before refusing an unbuildable backend: %v", statErr)
	}
	if verifyErr := VerifyInstalledRuntime(layout); verifyErr == nil {
		t.Fatal("a refused setup left a runtime that verifies as usable")
	}
}

// Negative: the module compiles. Every asset is present, `go build .` exits
// zero, and the program it produces exits 42 on every invocation. A build-only
// attestation accepts this and mints a receipt for a runtime whose first
// command fails.
func TestSetupRefusesSourceWhoseLauncherBackendBuildsButDoesNotStart(t *testing.T) {
	source := seedSourceRepo(t)
	breakLauncherBackendStartup(t, source)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{Layout: layout, Stdout: io.Discard})
	if err == nil {
		t.Fatal("Setup installed a runtime whose launcher backend builds but cannot start")
	}
	var sourceErr *SourceDirError
	if !errors.As(err, &sourceErr) {
		t.Fatalf("error %v is not a typed SourceDirError", err)
	}
	for _, want := range []string{launcherStartupProbeArg, "exit status 42"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not name %q", err, want)
		}
	}
	if _, statErr := os.Lstat(layout.AgentsDir); !os.IsNotExist(statErr) {
		t.Fatalf("setup wrote the destination before refusing a backend that does not start: %v", statErr)
	}
	if verifyErr := VerifyInstalledRuntime(layout); verifyErr == nil {
		t.Fatal("a refused setup left a runtime that verifies as usable")
	}
}

// Negative: the module builds and its program exits zero — it is simply not
// agents-infra. Exit zero from an unrelated program is the cheapest forgery of
// a working CLI, and the probe has to reject it.
func TestSetupRefusesSourceWhoseLauncherBackendIsNotAgentsInfra(t *testing.T) {
	source := seedSourceRepo(t)
	breakLauncherBackendIdentity(t, source)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{Layout: layout, Stdout: io.Discard})
	if err == nil {
		t.Fatal("Setup accepted a launcher backend that is not the agents-infra CLI")
	}
	if !strings.Contains(err.Error(), "does not identify it as agents-infra") {
		t.Fatalf("refusal %q does not say the binary failed to identify itself", err)
	}
	if _, statErr := os.Lstat(layout.AgentsDir); !os.IsNotExist(statErr) {
		t.Fatalf("setup wrote the destination before refusing a foreign backend: %v", statErr)
	}
}

// Negative: the installed tree is untouched and every launcher-backend path is
// still present; only the module's ability to build is gone. Verification must
// reject it, because that is the state the launcher will meet.
func TestVerifyInstalledRuntimeRefusesLauncherWhoseBackendStoppedBuilding(t *testing.T) {
	source := seedSourceRepo(t)
	layout := setupVerifiedLocalRuntime(t, source)

	breakLauncherBackendBuild(t, source)

	err := VerifyInstalledRuntime(layout)
	if err == nil {
		t.Fatal("VerifyInstalledRuntime accepted a launcher whose backend no longer builds")
	}
	for _, want := range []string{"agents-infra launcher", "cannot start", "go build", "internal/missing"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

// Negative: the same runtime, and the backend still compiles — the binary it
// produces no longer runs. Verification must reject it.
func TestVerifyInstalledRuntimeRefusesLauncherWhoseBackendStoppedStarting(t *testing.T) {
	source := seedSourceRepo(t)
	layout := setupVerifiedLocalRuntime(t, source)

	breakLauncherBackendStartup(t, source)

	err := VerifyInstalledRuntime(layout)
	if err == nil {
		t.Fatal("VerifyInstalledRuntime accepted a launcher whose backend builds but does not start")
	}
	for _, want := range []string{"agents-infra launcher", "cannot start", launcherStartupProbeArg, "exit status 42"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

// Negative: this is the reviewer's output-path bypass, stated as a test. The
// source is complete and buildable, the installed tree is untouched, and only
// the destination the launcher writes on every invocation is unavailable. A
// verification that builds to its own temporary directory passes here while the
// launcher still fails on first use.
func TestVerifyInstalledRuntimeRefusesWhenTheLauncherOutputPathIsUnwritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission smoke is Unix-only")
	}
	source := seedSourceRepo(t)
	layout := setupVerifiedLocalRuntime(t, source)

	outputDir := filepath.Dir(cliLocalBinaryPath(layout, runtime.GOOS))
	if err := os.RemoveAll(outputDir); err != nil {
		t.Fatalf("RemoveAll(%s): %v", outputDir, err)
	}
	if err := os.Chmod(layout.BinDir, 0o555); err != nil {
		t.Fatalf("Chmod(%s): %v", layout.BinDir, err)
	}
	t.Cleanup(func() { _ = os.Chmod(layout.BinDir, 0o755) })

	err := VerifyInstalledRuntime(layout)
	if err == nil {
		t.Fatal("VerifyInstalledRuntime accepted a launcher whose build output path cannot be written")
	}
	for _, want := range []string{"cannot start", outputDir} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

// The bound, stated as a test: global setup generates no launcher — the
// bootstrap owns ~/.local/bin/agents-infra — so it makes no claim about a build
// and must not start requiring one. Without this, "gate everything" would look
// indistinguishable from "gate what the runtime does".
func TestSetupGlobalDoesNotRequireALauncherBackendBuild(t *testing.T) {
	source := seedSourceRepo(t)
	breakLauncherBackendBuild(t, source)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("global Setup refused a source it does not build: %v", err)
	}
}

// The failure text is the whole point of choosing the real operation over a
// file list: it has to name what the build could not resolve, not report that
// something went wrong somewhere.
func TestLauncherStartupFailureNamesWhatTheBuildCouldNotResolve(t *testing.T) {
	source := seedSourceRepo(t)
	breakLauncherBackendBuild(t, source)

	failure := launcherBackendSourceFailure(ModeLocal, source)
	if failure == "" {
		t.Fatal("launcherBackendSourceFailure accepted a module that does not build")
	}
	for _, want := range []string{launcherBackendModuleDir(source), "internal/missing"} {
		if !strings.Contains(failure, want) {
			t.Fatalf("failure %q does not name %q", failure, want)
		}
	}
}

// The generated launcher must not depend on writing into the source checkout.
// That path is shared by every project installed from the same source, and it
// is the one the reviewer made unavailable to show that a passing verification
// still yielded a launcher that could not run.
func TestGeneratedLauncherBuildsIntoTheTargetNotTheSource(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	launcher := filepath.Join(layout.BinDir, cliWrapperName(runtime.GOOS))
	body, err := os.ReadFile(launcher)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", launcher, err)
	}
	binaryPath, ok := generatedCLIWrapperBinaryPath(string(body))
	if !ok {
		t.Fatalf("launcher %s records no build output path:\n%s", launcher, body)
	}
	if !dirContains(project, binaryPath) {
		t.Fatalf("launcher builds to %s, which is outside the target %s", binaryPath, project)
	}
	if dirContains(source, binaryPath) {
		t.Fatalf("launcher builds into the shared source checkout at %s", binaryPath)
	}
}

// End to end through the artifact a user actually invokes. The source's old
// build output directory — the reviewer's exact bypass — is made unwritable
// before anything runs, and the whole source tree is compared before and after:
// installing a project's runtime and then running its launcher must leave the
// shared checkout byte-for-byte alone.
//
// Blocking the path is not enough on its own. A run that had already written
// the artifact there would find the directory entry present and succeed on the
// second pass, so the snapshot is what states the property.
func TestGeneratedLauncherRunsWithoutWritingIntoTheSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the generated sh launcher and chmod are Unix-only")
	}
	source := seedSourceRepo(t)
	blockedOutput := filepath.Join(source, ".temp", "bin")
	mustMkdir(t, blockedOutput)
	if err := os.Chmod(blockedOutput, 0o555); err != nil {
		t.Fatalf("Chmod(%s): %v", blockedOutput, err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedOutput, 0o755) })
	before := sourceTreeSnapshot(t, source)

	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup could not install a runtime with the source's old output path blocked: %v", err)
	}

	launcher := filepath.Join(layout.BinDir, cliWrapperName(runtime.GOOS))
	output, err := exec.Command(launcher, launcherStartupProbeArg).CombinedOutput()
	if err != nil {
		t.Fatalf("generated launcher %s could not run with a read-only source output dir: %v\n%s", launcher, err, output)
	}
	if !strings.Contains(string(output), launcherStartupIdentity) {
		t.Fatalf("generated launcher did not answer as agents-infra:\n%s", output)
	}

	after := sourceTreeSnapshot(t, source)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("installing and running the runtime wrote into the source tree:\nbefore:\n%s\nafter:\n%s",
			strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
}

// sourceTreeSnapshot lists the source tree, skipping .git: `go build` stamps VCS
// information, so git churn inside its own metadata is expected and is not a
// write by agents-infra.
func sourceTreeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return entries
}

// Verifying a source must not mutate it: the launcher's output lives in the
// target, and the source-preflight probe writes to a throwaway directory.
func TestLauncherStartupCheckDoesNotWriteIntoTheSourceTree(t *testing.T) {
	source := seedSourceRepo(t)
	before := treeSnapshot(t, filepath.Join(source, "tools"))

	if failure := launcherBackendSourceFailure(ModeLocal, source); failure != "" {
		t.Fatalf("launcherBackendSourceFailure on a usable module: %s", failure)
	}

	after := treeSnapshot(t, filepath.Join(source, "tools"))
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("the check wrote into the source tree:\nbefore:\n%s\nafter:\n%s",
			strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
}

// setupVerifiedLocalRuntime installs a runtime and proves the same check accepts
// it, so a later failure is the break under test and not a gate that refuses
// everything.
func setupVerifiedLocalRuntime(t *testing.T, source string) Layout {
	t.Helper()
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime after a successful Setup: %v", err)
	}
	return layout
}

func treeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return entries
}
