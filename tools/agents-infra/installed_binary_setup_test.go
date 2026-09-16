package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
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
		filepath.Join(project, ".agents", ".agents-infra-install.json"),
		filepath.Join(project, ".agents", ".configs", "claude-settings.json"),
		filepath.Join(project, ".agents", ".rules", "default.rules"),
		filepath.Join(project, ".claude", "settings.json"),
		filepath.Join(project, ".local", "bin", "agents-infra"),
	} {
		if _, statErr := os.Lstat(want); statErr != nil {
			t.Fatalf("installed binary setup did not produce %s: %v\n%s", want, statErr, output)
		}
	}
	for _, absent := range []string{
		filepath.Join(project, ".agents", ".instructions"),
		filepath.Join(project, ".agents", ".skills"),
		filepath.Join(project, ".claude", "instructions"),
		filepath.Join(project, "AGENTS.md"),
	} {
		if _, statErr := os.Lstat(absent); !os.IsNotExist(statErr) {
			t.Fatalf("installed binary setup distributed retired surface %s: %v\n%s", absent, statErr, output)
		}
	}
}

// Production call site: the installed binary dispatches setup local through
// runSetup into infra.Setup. If resolveLocalSetupLayout is removed or narrowed
// to lexical equality, these cases enter syncRepo and create project/.agents
// (recursively for the equality cases), so the no-mutation assertions kill the
// mutant rather than merely proving a helper exists.
func TestInstalledBinarySetupLocalRefusesRecursiveSourceBeforeFilesystemMutation(t *testing.T) {
	binary := buildInstalledBinary(t)
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tests := []struct {
		name string
		args func(t *testing.T, source, project string) (string, string)
	}{
		{
			name: "equal",
			args: func(_ *testing.T, source, _ string) (string, string) { return source, source },
		},
		{
			name: "relative paths",
			args: func(t *testing.T, source, _ string) (string, string) {
				relative, relErr := filepath.Rel(workingDir, source)
				if relErr != nil {
					t.Fatalf("Rel(%s, %s): %v", workingDir, source, relErr)
				}
				return filepath.Join(relative, "."), relative
			},
		},
		{
			name: "trailing separators",
			args: func(_ *testing.T, source, _ string) (string, string) {
				return source + string(filepath.Separator), source + string(filepath.Separator) + "."
			},
		},
		{
			name: "symlink alias",
			args: func(t *testing.T, source, _ string) (string, string) {
				alias := filepath.Join(t.TempDir(), "source-alias")
				if linkErr := os.Symlink(source, alias); linkErr != nil {
					t.Skipf("cannot create source alias: %v", linkErr)
				}
				return alias, source
			},
		},
		{
			name: "source contains project",
			args: func(_ *testing.T, source, project string) (string, string) { return source, project },
		},
	}
	if runtime.GOOS == "darwin" {
		tests = append(tests, struct {
			name string
			args func(t *testing.T, source, project string) (string, string)
		}{
			name: "case insensitive equality",
			args: func(t *testing.T, source, _ string) (string, string) {
				upper := strings.ToUpper(source)
				if _, statErr := os.Stat(upper); statErr != nil {
					t.Skip("test volume is case-sensitive")
				}
				return upper, source
			},
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := seedRuntimeSource(t, t.TempDir())
			project := source
			if test.name == "source contains project" {
				project = filepath.Join(source, "nested-project")
				mustMkdir(t, project)
			}
			sourceArg, projectArg := test.args(t, source, project)
			home := t.TempDir()
			configDir := filepath.Join(home, "config")

			output, runErr := runInstalledBinary(
				t, binary, home, configDir,
				"setup", "local", projectArg,
				"--source-dir", sourceArg,
				"--claude-yolo-mode=true",
			)
			if runErr == nil {
				t.Fatalf("installed production setup local accepted recursive source:\n%s", output)
			}
			resolvedSource, resolveErr := filepath.EvalSymlinks(source)
			if resolveErr != nil {
				t.Fatalf("EvalSymlinks(source): %v", resolveErr)
			}
			resolvedProject, resolveErr := filepath.EvalSymlinks(project)
			if resolveErr != nil {
				t.Fatalf("EvalSymlinks(project): %v", resolveErr)
			}
			for _, want := range []string{
				"refusing setup local",
				"resolved source directory",
				resolvedSource,
				"resolved project directory",
				resolvedProject,
				"syncing would copy the source into its own project-local destination",
			} {
				if !strings.Contains(output, want) {
					t.Fatalf("setup local refusal %q missing %q", output, want)
				}
			}
			for _, path := range []string{
				filepath.Join(project, ".agents"),
				filepath.Join(project, ".claude"),
				filepath.Join(project, ".codex"),
				filepath.Join(project, ".local"),
				filepath.Join(project, "AGENTS.md"),
			} {
				if _, statErr := os.Lstat(path); !os.IsNotExist(statErr) {
					t.Fatalf("refused setup mutated %s: %v", path, statErr)
				}
			}
		})
	}
}

// Setup scrubs legacy artifacts and distributes no skills: source .skills
// content — including hostile links that the old validator refused — is left
// behind entirely, while the residual runtime still installs and verifies.
func TestInstalledBinarySetupLocalScrubsLegacyArtifactsAndDistributesNoSkills(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
	legacyArtifact := filepath.Join(source, "$AGENTS_INFRA_SOURCE_DIR", ".temp", "bin")
	mustMkdir(t, legacyArtifact)
	mustWrite(t, filepath.Join(legacyArtifact, "agents-infra-local"), "legacy build output")
	nestedScratch := filepath.Join(source, "tools", "agents-infra", ".temp", "legacy-runtime")
	mustMkdir(t, nestedScratch)
	mustWrite(t, filepath.Join(nestedScratch, "stale"), "must not materialize")

	safeSkillTarget := filepath.Join(source, ".skills", "safe-target")
	mustMkdir(t, safeSkillTarget)
	mustWrite(t, filepath.Join(safeSkillTarget, "SKILL.md"), "# safe target\n")
	if err := os.Symlink("safe-target", filepath.Join(source, ".skills", "safe-link")); err != nil {
		t.Skipf("cannot create control skill symlink: %v", err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(source, ".skills", "escape-probe")); err != nil {
		t.Skipf("cannot create hostile source skill symlink: %v", err)
	}
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	literalDir := filepath.Join(project, ".agents", "$AGENTS_INFRA_SOURCE_DIR")
	if _, statErr := os.Lstat(literalDir); !os.IsNotExist(statErr) {
		t.Fatalf("setup retained literal source-dir artifact %s: %v", literalDir, statErr)
	}
	installedNestedScratch := filepath.Join(project, ".agents", "tools", "agents-infra", ".temp")
	if _, statErr := os.Lstat(installedNestedScratch); !os.IsNotExist(statErr) {
		t.Fatalf("setup retained nested source scratch %s: %v", installedNestedScratch, statErr)
	}

	agentsDir := filepath.Join(project, ".agents")
	for _, distributed := range []string{
		filepath.Join(agentsDir, ".skills"),
		filepath.Join(agentsDir, "skills"),
		filepath.Join(agentsDir, ".skills", "relux-agents-infra"),
	} {
		if _, statErr := os.Lstat(distributed); !os.IsNotExist(statErr) {
			t.Fatalf("setup distributed skill surface %s: %v", distributed, statErr)
		}
	}
	if output, err := runInstalledBinary(t, binary, home, configDir, "verify", "local", project); err != nil {
		t.Fatalf("verify local: %v\n%s", err, output)
	}
}

// Setup distributes no instructions and no bundled MCP registry even when the
// source carries them, while residual config, rules, and helpers install and
// verify. A pre-existing caller registry is preserved byte-identical: sync
// only writes paths it walks from source.
func TestInstalledBinarySetupLocalDistributesNoInstructionsOrBundledRegistry(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
	mustWrite(t, filepath.Join(source, ".instructions", "AGENTS.md"), "# Source instructions marker\n")
	mustWrite(t, filepath.Join(source, ".configs", "codex-mcp-servers.toml"), "[servers.bundled]\nurl = \"https://bundled.example/mcp\"\n")
	mustWrite(t, filepath.Join(source, ".configs", "claude-settings.json"), "{}\n")
	mustWrite(t, filepath.Join(source, ".rules", "default.rules"), "allow\n")
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()
	callerRegistry := filepath.Join(project, ".agents", ".configs", "codex-mcp-servers.toml")
	mustMkdir(t, filepath.Dir(callerRegistry))
	mustWrite(t, callerRegistry, "[servers.caller]\nurl = \"https://caller.example/mcp\"\n")

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	agentsDir := filepath.Join(project, ".agents")
	for _, distributed := range []string{
		filepath.Join(agentsDir, ".instructions"),
		filepath.Join(project, ".codex", "AGENTS.md"),
		filepath.Join(project, "AGENTS.md"),
		filepath.Join(project, ".claude", "CLAUDE.md"),
	} {
		if _, statErr := os.Lstat(distributed); !os.IsNotExist(statErr) {
			t.Fatalf("setup distributed instruction surface %s: %v", distributed, statErr)
		}
	}
	if data, err := os.ReadFile(callerRegistry); err != nil || !strings.Contains(string(data), "servers.caller") || strings.Contains(string(data), "servers.bundled") {
		t.Fatalf("caller registry not preserved byte-identical: err=%v data=%q", err, data)
	}
	for _, residual := range []string{
		filepath.Join(agentsDir, ".configs", "claude-settings.json"),
		filepath.Join(agentsDir, ".rules", "default.rules"),
		filepath.Join(project, ".claude", "settings.json"),
		filepath.Join(project, ".codex", "rules", "default.rules"),
		filepath.Join(project, ".local", "bin", "agents-attachments"),
		filepath.Join(project, ".local", "bin", "agents-infra"),
	} {
		if _, statErr := os.Lstat(residual); statErr != nil {
			t.Fatalf("residual surface %s missing: %v\n%s", residual, statErr, output)
		}
	}
	if output, err := runInstalledBinary(t, binary, home, configDir, "verify", "local", project); err != nil {
		t.Fatalf("verify local: %v\n%s", err, output)
	}
}

func TestInstalledBinarySetupAndVerifyLocalPreserveUnmanagedProviderSkillLinks(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()
	externalSkill := filepath.Join(t.TempDir(), "mac-infra")
	mustMkdir(t, externalSkill)
	mustWrite(t, filepath.Join(externalSkill, "SKILL.md"), "external provider-owned skill\n")

	for _, providerSkills := range []string{
		filepath.Join(project, ".claude", "skills"),
		filepath.Join(project, ".codex", "skills"),
	} {
		mustMkdir(t, providerSkills)
		if err := os.Symlink(externalSkill, filepath.Join(providerSkills, "mac-infra")); err != nil {
			t.Skipf("cannot create provider-owned skill symlink: %v", err)
		}
	}

	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local refused provider-owned skill links: %v\n%s", err, output)
	}
	if output, err := runInstalledBinary(t, binary, home, configDir, "verify", "local", project); err != nil {
		t.Fatalf("verify local refused provider-owned skill links: %v\n%s", err, output)
	}
	for _, providerSkills := range []string{
		filepath.Join(project, ".claude", "skills"),
		filepath.Join(project, ".codex", "skills"),
	} {
		link := filepath.Join(providerSkills, "mac-infra")
		target, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("Readlink(%s): %v", link, err)
		}
		if target != externalSkill {
			t.Fatalf("provider-owned skill link %s changed: got %q, want %q", link, target, externalSkill)
		}
	}
}

func TestInstalledBinarySetupGlobalPiInfraPreservesCWDArgvAndRefusesDrift(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX production alias test")
	}
	built := buildInstalledBinary(t)
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	mustMkdir(t, binDir)
	installed := filepath.Join(binDir, "agents-infra")
	binaryBytes, err := os.ReadFile(built)
	if err != nil {
		t.Fatalf("ReadFile(built binary): %v", err)
	}
	if err := os.WriteFile(installed, binaryBytes, 0o755); err != nil {
		t.Fatalf("WriteFile(installed binary): %v", err)
	}
	configDir := filepath.Join(home, "config")
	setupOutput, err := runInstalledBinary(t, installed, home, configDir, "setup", "global", "--source-dir", sourceRepoRoot(t))
	if err != nil {
		t.Fatalf("installed binary setup global: %v\n%s", err, setupOutput)
	}
	if verifyOutput, verifyErr := runInstalledBinary(t, installed, home, configDir, "verify", "global"); verifyErr != nil {
		t.Fatalf("installed binary verify global: %v\n%s", verifyErr, verifyOutput)
	}

	fakeBin := t.TempDir()
	fakePi := filepath.Join(fakeBin, "pi")
	mustWrite(t, fakePi, "#!/usr/bin/env sh\nprintf 'cwd=<%s>\\n' \"$PWD\"\nfor arg in \"$@\"; do printf 'arg=<%s>\\n' \"$arg\"; done\n")
	if err := os.Chmod(fakePi, 0o755); err != nil {
		t.Fatalf("Chmod(fake pi): %v", err)
	}
	caller := filepath.Join(t.TempDir(), "caller with spaces")
	mustMkdir(t, caller)
	canonicalCaller, err := filepath.EvalSymlinks(caller)
	if err != nil {
		t.Fatalf("EvalSymlinks(caller): %v", err)
	}
	args := []string{"--", "ordinary prompt", "--post-separator", "@literal"}
	command := exec.Command(filepath.Join(binDir, "pi-infra"), args...)
	command.Dir = caller
	command.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"AGENTS_INFRA_CONFIG_DIR="+configDir,
		"AGENTS_INFRA_SOURCE_DIR=",
		"AGENTS_INFRA_CALLER_CWD=",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("pi-infra production entry: %v\n%s", err, output)
	}
	wantLines := []string{"cwd=<" + canonicalCaller + ">"}
	for _, arg := range args {
		wantLines = append(wantLines, "arg=<"+arg+">")
	}
	if got, want := strings.TrimSpace(string(output)), strings.Join(wantLines, "\n"); got != want {
		t.Fatalf("production delegation output:\n%s\nwant:\n%s", got, want)
	}

	aliasPath := filepath.Join(binDir, "pi-infra")
	mustWrite(t, aliasPath, strings.ReplaceAll(string(mustReadFile(t, aliasPath)), "agents-infra", "other-infra"))
	verifyOutput, verifyErr := runInstalledBinary(t, installed, home, configDir, "verify", "global")
	if verifyErr == nil || !strings.Contains(verifyOutput, "pi-infra launcher") || !strings.Contains(verifyOutput, "has drifted") {
		t.Fatalf("verify global did not refuse alias drift: %v\n%s", verifyErr, verifyOutput)
	}
}

func TestInstalledBinarySetupLocalPiInfraRepairsModeAndSymlinkDrift(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX production alias type and mode test")
	}
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, sourceRepoRoot(t))
	project := t.TempDir()

	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	aliasPath := filepath.Join(project, ".local", "bin", "pi-infra")
	targetPath := filepath.Join(project, ".local", "bin", "agents-infra")

	if err := os.Chmod(aliasPath, 0o644); err != nil {
		t.Fatalf("Chmod(pi-infra): %v", err)
	}
	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local did not repair alias mode: %v\n%s", err, output)
	}
	assertRegularExecutable := func(path string) {
		t.Helper()
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("Lstat(%s): %v", path, err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o755 {
			t.Fatalf("%s mode = %v, want regular 0755", path, info.Mode())
		}
	}
	assertRegularExecutable(aliasPath)

	externalAlias := filepath.Join(t.TempDir(), "pi-infra-copy")
	if err := os.WriteFile(externalAlias, mustReadFile(t, aliasPath), 0o755); err != nil {
		t.Fatalf("WriteFile(external alias): %v", err)
	}
	if err := os.Remove(aliasPath); err != nil {
		t.Fatalf("Remove(pi-infra): %v", err)
	}
	if err := os.Symlink(externalAlias, aliasPath); err != nil {
		t.Fatalf("Symlink(pi-infra): %v", err)
	}
	verifyOutput, verifyErr := runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
	if verifyErr == nil || !strings.Contains(verifyOutput, "pi-infra launcher") || !strings.Contains(verifyOutput, "is not a regular file") {
		t.Fatalf("verify local accepted byte-identical symlink alias: %v\n%s", verifyErr, verifyOutput)
	}
	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local did not repair symlink alias: %v\n%s", err, output)
	}
	assertRegularExecutable(aliasPath)

	externalTarget := filepath.Join(t.TempDir(), "agents-infra-copy")
	if err := os.WriteFile(externalTarget, mustReadFile(t, targetPath), 0o755); err != nil {
		t.Fatalf("WriteFile(external target): %v", err)
	}
	if err := os.Remove(targetPath); err != nil {
		t.Fatalf("Remove(agents-infra): %v", err)
	}
	if err := os.Symlink(externalTarget, targetPath); err != nil {
		t.Fatalf("Symlink(agents-infra): %v", err)
	}
	verifyOutput, verifyErr = runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
	if verifyErr == nil || !strings.Contains(verifyOutput, "pi-infra launcher target is not a regular file") {
		t.Fatalf("verify local accepted byte-identical symlink target: %v\n%s", verifyErr, verifyOutput)
	}
}

// Installed canonical and dange wrappers still exist for one release as
// direct-refusal stubs: every wrapper reports its exact migration message,
// exits 1, and never reaches the provider — without building, delegating, or
// touching a sibling target. The fixture deletes the cached wrapper build
// output and shadows `go` with a failing stub, so a wrapper that still built
// would exit 73 instead of 1.
func TestInstalledProviderAliasesReportDeprecationWithoutLaunching(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX installed alias production test")
	}
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, sourceRepoRoot(t))
	project := t.TempDir()
	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	writeMainCanonicalConfig(t, project, mainCanonicalHostedTOML())
	if err := os.RemoveAll(filepath.Join(project, ".local", "bin", ".agents-infra-build")); err != nil {
		t.Fatalf("remove cached wrapper build output: %v", err)
	}

	fakeBin := t.TempDir()
	recordDir := t.TempDir()
	for _, provider := range []string{"codex", "claude"} {
		record := filepath.Join(recordDir, provider)
		mustWrite(t, filepath.Join(fakeBin, provider), "#!/bin/sh\nprintf '%s\\0' \"$@\" > \""+record+"\"\n")
	}
	mustWrite(t, filepath.Join(fakeBin, "go"), "#!/bin/sh\necho unexpected-build >&2\nexit 73\n")
	if err := os.Chmod(filepath.Join(fakeBin, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	installedAliasEnv := func() []string {
		environ := append(os.Environ(),
			"HOME="+home,
			"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
			"AGENTS_INFRA_CONFIG_DIR="+configDir,
			"AGENTS_INFRA_SOURCE_DIR=",
			"AGENTS_INFRA_CALLER_CWD=",
		)
		return append(environ, sharedGoCacheEnv(t)...)
	}
	tests := []struct {
		alias    string
		provider string
		want     string
	}{
		{alias: "openai-infra", provider: "codex", want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{alias: "anthropic-infra", provider: "claude", want: "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
		{alias: "openai-dange", provider: "codex", want: "openai-dange is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{alias: "anthropic-dange", provider: "claude", want: "anthropic-dange is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
	}
	argShapes := [][]string{
		nil,
		{"--print-config"},
		{"-d", "--danger", "--yolo", "--model", "caller-model"},
		{revision5DirectProviderYoloCallSiteMarker, "--model", "caller-model"},
	}
	for _, testCase := range tests {
		for _, args := range argShapes {
			t.Run(testCase.alias+"/"+fmt.Sprint(args), func(t *testing.T) {
				record := filepath.Join(recordDir, testCase.provider)
				if err := os.Remove(record); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				command := exec.Command(filepath.Join(project, ".local", "bin", testCase.alias), args...)
				command.Dir = project
				command.Env = installedAliasEnv()
				var stdout, stderr strings.Builder
				command.Stdout = &stdout
				command.Stderr = &stderr
				runErr := command.Run()
				exitErr, ok := runErr.(*exec.ExitError)
				if !ok || exitErr.ExitCode() != 1 {
					t.Fatalf("installed %s %q exit = %v, want exit 1 (stdout=%q stderr=%q)", testCase.alias, args, runErr, stdout.String(), stderr.String())
				}
				if stdout.String() != "" {
					t.Fatalf("installed %s %q stdout = %q, want empty", testCase.alias, args, stdout.String())
				}
				if stderr.String() != testCase.want+"\n" {
					t.Fatalf("installed %s %q stderr = %q, want exactly %q", testCase.alias, args, stderr.String(), testCase.want+"\n")
				}
				if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
					t.Fatalf("installed %s reached provider side effect: %v", testCase.alias, statErr)
				}
			})
		}
	}
}

// Production call sites: the bootstrap-installed global pi-infra alias and the
// setup-generated project-local pi-infra wrapper. Both must reach RunPi's
// managed execution-environment gate before the configured runtime executable
// can start.
func TestInstalledPiLaunchersRejectExactEnvironmentNamesBeforeRuntimeSpawn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("managed Pi launch is supported only on darwin/arm64")
	}
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	configDir := filepath.Join(home, "config")
	binDir := filepath.Join(home, ".local", "bin")
	mustMkdir(t, binDir)
	globalBinary := filepath.Join(binDir, "agents-infra")
	if err := os.WriteFile(globalBinary, mustReadFile(t, binary), 0o755); err != nil {
		t.Fatalf("install global binary fixture: %v", err)
	}
	if output, err := runInstalledBinary(t, globalBinary, home, configDir, "setup", "global", "--source-dir", sourceRepoRoot(t)); err != nil {
		t.Fatalf("setup global fixture: %v\n%s", err, output)
	}
	project := t.TempDir()
	if output, err := runInstalledBinary(t, globalBinary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local fixture: %v\n%s", err, output)
	}
	runtimeMarker := filepath.Join(t.TempDir(), "runtime-started")
	runtimeExecutable := filepath.Join(t.TempDir(), "runtime")
	mustWrite(t, runtimeExecutable, "#!/usr/bin/env sh\nprintf started > "+strconv.Quote(runtimeMarker)+"\n")
	if err := os.Chmod(runtimeExecutable, 0o755); err != nil {
		t.Fatalf("Chmod(runtime fixture): %v", err)
	}
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustWrite(t, configPath, mainTestPiConfig(runtimeExecutable, 18029))
	piRoot := mainTestOfficialPiAsset(t)

	surfaces := map[string]string{
		"bootstrap global alias": filepath.Join(binDir, "pi-infra"),
		"project local wrapper":  filepath.Join(project, ".local", "bin", "pi-infra"),
	}
	for surface, launcher := range surfaces {
		t.Run(surface+"/clean control", func(t *testing.T) {
			command := exec.Command(launcher)
			command.Dir = project
			command.Env = []string{
				"HOME=" + home,
				"PATH=" + piRoot + string(os.PathListSeparator) + os.Getenv("PATH"),
				"AGENTS_INFRA_CONFIG_DIR=" + configDir,
				"AGENTS_INFRA_SOURCE_DIR=",
				"HF_TOKEN=credential-treated-separately",
				"HF_HOME=/tmp/hf-cache",
				"HUGGINGFACE_HUB_CACHE=/tmp/huggingface-hub-cache",
				"TRANSFORMERS_CACHE=/tmp/transformers-cache",
				"LLAMA_API_KEY_SUFFIX=not-the-exact-auth-control",
				"llama_api_key=case-sensitive-lookalike",
				"UNRELATED_SERVICE_API_KEY=unrelated-control",
				"GGML_METAL_PATH=unestablished-control",
			}
			command.Env = append(command.Env, sharedGoCacheEnv(t)...)
			output, launchErr := command.CombinedOutput()
			if _, err := os.Stat(runtimeMarker); err != nil {
				t.Fatalf("installed %s clean control did not reach runtime backend initialization: launch=%v marker=%v\n%s", surface, launchErr, err, output)
			}
			if err := os.Remove(runtimeMarker); err != nil {
				t.Fatalf("remove runtime control marker: %v", err)
			}
		})
		for _, name := range []string{"HF_ENDPOINT", "MODEL_ENDPOINT", "GGML_BACKEND_PATH", "LLAMA_API_KEY"} {
			t.Run(surface+"/"+name, func(t *testing.T) {
				secret := "https://must-not-leak.invalid/" + strings.ToLower(name)
				command := exec.Command(launcher)
				command.Dir = project
				command.Env = []string{
					"HOME=" + home,
					"PATH=" + piRoot + string(os.PathListSeparator) + os.Getenv("PATH"),
					"AGENTS_INFRA_CONFIG_DIR=" + configDir,
					"AGENTS_INFRA_SOURCE_DIR=",
					name + "=" + secret,
				}
				command.Env = append(command.Env, sharedGoCacheEnv(t)...)
				output, err := command.CombinedOutput()
				if err == nil || !strings.Contains(string(output), "runtime-affecting environment name "+strconv.Quote(name)+" is denied") {
					t.Fatalf("installed %s admitted %s: %v\n%s", surface, name, err, output)
				}
				if strings.Contains(string(output), secret) {
					t.Fatalf("installed %s leaked %s value: %s", surface, name, output)
				}
				if _, statErr := os.Stat(runtimeMarker); !os.IsNotExist(statErr) {
					t.Fatalf("installed %s started runtime before refusing %s: %v", surface, name, statErr)
				}
			})
		}
	}
}

// Production call site: the setup-generated .local/bin/agents-infra compose
// entrypoint. This guards against an installed runtime carrying a parser that
// reports base_url while launching a wildcard or different-port backend.
func TestInstalledLocalAgentsInfraComposeRefusesPiRuntimeEndpointDivergence(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, sourceRepoRoot(t))
	project := t.TempDir()
	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("installed binary setup local: %v\n%s", err, output)
	}
	installed := filepath.Join(project, ".local", "bin", "agents-infra")
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	piRoot := mainTestOfficialPiAsset(t)
	base := mainTestPiConfig("/bin/echo", 18021)
	runCompose := func(config string) ([]byte, error) {
		t.Helper()
		mustWrite(t, configPath, config)
		command := exec.Command(installed, "compose", "--mode", "primary-session", "--agent", "pi", "--project", project, "--schema-version", "1", "--json")
		command.Env = append(os.Environ(),
			"HOME="+home,
			"PATH="+piRoot+string(os.PathListSeparator)+os.Getenv("PATH"),
			"AGENTS_INFRA_CONFIG_DIR="+configDir,
			"AGENTS_INFRA_SOURCE_DIR=",
			"AGENTS_INFRA_CALLER_CWD=",
		)
		command.Env = append(command.Env, sharedGoCacheEnv(t)...)
		return command.CombinedOutput()
	}

	controlOutput, err := runCompose(base)
	if err != nil {
		t.Fatalf("installed production compose rejected exact endpoint control: %v\n%s", err, controlOutput)
	}
	var control infra.PrimarySessionLaunchPlan
	if err := json.Unmarshal(controlOutput, &control); err != nil {
		t.Fatalf("decode installed control plan: %v\n%s", err, controlOutput)
	}
	wantArgv := []string{"serve", "--model", "Model", "--host", "127.0.0.1", "--port", "18021"}
	if control.Pi == nil || control.Pi.Runtime == nil || !slices.Equal(control.Pi.Runtime.Argv, wantArgv) {
		t.Fatalf("installed exact endpoint control plan=%#v", control.Pi)
	}

	for name, mutant := range map[string]string{
		"wildcard runtime bind": strings.Replace(base, `"--host", "127.0.0.1"`, `"--host", "0.0.0.0"`, 1),
		"runtime port drift":    strings.Replace(base, `"--port", "18021"`, `"--port", "19021"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			output, err := runCompose(mutant)
			if err == nil || !strings.Contains(string(output), `"code":"invalid_project_configuration"`) || !strings.Contains(string(output), ".runtime.argv") {
				t.Fatalf("installed production compose admitted endpoint divergence: %v\n%s", err, output)
			}
		})
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return data
}

// seedRuntimeSource writes a complete agents-infra source tree: the config and
// rules trees, the retained instruction sources, and the Go module the
// generated launcher builds. Anything less than the residual contract is not a
// source tree, it is a tree that looks like one — see the marker-valid
// negatives below.
func seedRuntimeSource(t *testing.T, dir string) string {
	t.Helper()
	mustMkdir(t, filepath.Join(dir, ".instructions"))
	mustMkdir(t, filepath.Join(dir, ".configs"))
	mustMkdir(t, filepath.Join(dir, ".rules"))
	mustWrite(t, filepath.Join(dir, ".instructions", "INSTRUCTIONS.md"), "# Instructions\n")
	mustWrite(t, filepath.Join(dir, ".instructions", "AGENTS.md"), "# Agents\n")
	mustWrite(t, filepath.Join(dir, "SKILL.md"), "# relux-agents-infra\n")
	mustWrite(t, filepath.Join(dir, "README.md"), "# relux-agents-infra\n")
	mustMkdir(t, filepath.Join(dir, "tools", "agents-infra"))
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "go.mod"), "module example.com/agents-infra\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "main.go"), runnableLauncherBackendMain)
	manifest, err := os.ReadFile(filepath.Join(sourceRepoRoot(t), "tools", "agents-infra", "internal", "infra", "pi-v0.84.2-darwin-arm64-tree-manifest.txt"))
	if err != nil {
		t.Fatalf("read authoritative Pi manifest fixture: %v", err)
	}
	mustMkdir(t, filepath.Join(dir, "tools", "agents-infra", "internal", "infra"))
	mustWrite(t, filepath.Join(dir, "tools", "agents-infra", "internal", "infra", "pi-v0.84.2-darwin-arm64-tree-manifest.txt"), string(manifest))
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
	marker := filepath.Join(project, ".agents", ".agents-infra-install.json")
	if _, statErr := os.Lstat(marker); statErr != nil {
		t.Fatalf("installed binary setup did not sync the installed runtime: %v\n%s", statErr, output)
	}
	// A run that reports success must leave a runtime that verifies, otherwise
	// exit zero is the only evidence there is.
	if verifyOutput, verifyErr := runInstalledBinary(t, binary, home, filepath.Join(home, "config-without-state"), "verify", "local", project); verifyErr != nil {
		t.Fatalf("installed binary could not verify the runtime it just installed: %v\n%s", verifyErr, verifyOutput)
	}
}

// Negative: the source tree carries every residual marker — .configs, .rules —
// but not the Go module the generated launcher builds. Setup used to exit
// zero here, print a full install log, and mint a launcher that failed on
// first use. It must now refuse, and must not leave a runtime behind for a
// caller to mistake for a working one.
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
	if _, err := os.Lstat(filepath.Join(target, "internal", "infra", "infra.go")); !os.IsNotExist(err) {
		t.Fatalf("the forged module must not carry the internal Go package even though it retains the required Pi manifest: %v", err)
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

// Setup no longer renders instructions, so a source tree with an unshipped
// include is accepted. The failure it used to prove up front now belongs to
// the v1 prepare compatibility path: prepare fails honestly on an installed
// instruction input it cannot render, naming the missing module.
func TestInstalledBinaryPrepareFailsHonestlyOnUnrenderableInstalledInclude(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, t.TempDir())
	mustWrite(t, filepath.Join(source, ".instructions", "AGENTS.md"),
		"# Agents\n\n@~/.agents/.instructions/INSTRUCTIONS_MISSING.md\n")
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()

	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local refused a source whose instruction closure it no longer validates: %v\n%s", err, output)
	}
	installedAgents := filepath.Join(project, ".agents", ".instructions", "AGENTS.md")
	mustMkdir(t, filepath.Dir(installedAgents))
	mustWrite(t, installedAgents, "# Project\n\n@~/.agents/.instructions/INSTRUCTIONS_MISSING.md\n")

	output, err := runInstalledBinary(t, binary, home, configDir, "prepare", "--agent", "codex", "--project", project, "--schema-version", "1", "--json")
	if err == nil {
		t.Fatalf("prepare accepted an installed instruction input it cannot render\n%s", output)
	}
	if !strings.Contains(output, "INSTRUCTIONS_MISSING.md") {
		t.Fatalf("prepare failure does not name the missing instruction module:\n%s", output)
	}
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
	for _, want := range []string{wrong, ".configs", ".rules", "tools/agents-infra/go.mod"} {
		if !strings.Contains(output, want) {
			t.Fatalf("failure output missing %q:\n%s", want, output)
		}
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("refused setup still wrote into the project: %v", statErr)
	}
}
