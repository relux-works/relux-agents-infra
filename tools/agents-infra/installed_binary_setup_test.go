package main

import (
	"encoding/json"
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
		filepath.Join(project, ".agents", ".instructions", "INSTRUCTIONS.md"),
		filepath.Join(project, ".claude", "instructions"),
		filepath.Join(project, "AGENTS.md"),
	} {
		if _, statErr := os.Lstat(want); statErr != nil {
			t.Fatalf("installed binary setup did not produce %s: %v\n%s", want, statErr, output)
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

func TestInstalledBinarySetupLocalScrubsLiteralSourceDirAndAvoidsRepoSkillCycle(t *testing.T) {
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
		t.Skipf("cannot create narrowing-control skill symlink: %v", err)
	}
	nestedSafeTarget := filepath.Join(source, ".skills", "nested-safe", "target")
	mustMkdir(t, nestedSafeTarget)
	mustWrite(t, filepath.Join(nestedSafeTarget, "SKILL.md"), "# nested safe target\n")
	if err := os.Symlink("target", filepath.Join(source, ".skills", "nested-safe", "link")); err != nil {
		t.Skipf("cannot create nested narrowing-control skill symlink: %v", err)
	}
	dagDir := filepath.Join(source, ".skills", "contained-dag")
	mustMkdir(t, dagDir)
	for _, name := range []string{"left", "right"} {
		if err := os.Symlink(filepath.Join("..", "safe-target"), filepath.Join(dagDir, name)); err != nil {
			t.Skipf("cannot create contained DAG skill symlink: %v", err)
		}
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
	repoSkillLink := filepath.Join(agentsDir, "skills", "relux-agents-infra")
	wantTarget := filepath.Join(agentsDir, ".skills", "relux-agents-infra")
	rawTarget, err := os.Readlink(repoSkillLink)
	if err != nil {
		t.Fatalf("Readlink(%s): %v", repoSkillLink, err)
	}
	if !filepath.IsAbs(rawTarget) {
		rawTarget = filepath.Join(filepath.Dir(repoSkillLink), rawTarget)
	}
	if got, want := filepath.Clean(rawTarget), filepath.Clean(wantTarget); got != want {
		t.Fatalf("repository skill target = %s, want %s", got, want)
	}
	assertContainedAcyclicSymlinks(t, agentsDir)
	if _, err := os.Stat(filepath.Join(agentsDir, ".skills", "safe-link", "SKILL.md")); err != nil {
		t.Fatalf("safe contained source skill link did not survive setup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentsDir, ".skills", "nested-safe", "link", "SKILL.md")); err != nil {
		t.Fatalf("nested safe contained source skill link did not survive setup: %v", err)
	}
	for _, name := range []string{"left", "right"} {
		if _, err := os.Stat(filepath.Join(agentsDir, ".skills", "contained-dag", name, "SKILL.md")); err != nil {
			t.Fatalf("contained DAG skill link %s did not survive setup: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repoSkillLink, "SKILL.md")); err != nil {
		t.Fatalf("materialized repository skill is not readable through production link: %v", err)
	}

	find := exec.Command("find", "-L", agentsDir, "-maxdepth", "8", "-print")
	findOutput, findErr := find.CombinedOutput()
	if findErr != nil {
		t.Fatalf("recursive-safe inspection failed: %v\n%s", findErr, findOutput)
	}
	if strings.Contains(string(findOutput), "skills/relux-agents-infra/skills/relux-agents-infra") {
		t.Fatalf("recursive inspection re-entered the repository skill through itself:\n%s", findOutput)
	}
}

func TestInstalledBinarySetupLocalRefusesContainedTransitiveSourceSkillCycleBeforeDestinationMutation(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
	mustMkdir(t, filepath.Join(source, ".skills"))
	cycleTarget := filepath.Join(source, "cycle-target")
	mustMkdir(t, cycleTarget)
	probe := filepath.Join(source, ".skills", "transitive-cycle-probe")
	if err := os.Symlink(filepath.Join("..", "cycle-target"), probe); err != nil {
		t.Skipf("cannot create transitive source skill link: %v", err)
	}
	if err := os.Symlink(filepath.Join("..", ".skills", "transitive-cycle-probe"), filepath.Join(cycleTarget, "back")); err != nil {
		t.Skipf("cannot close transitive source skill cycle: %v", err)
	}
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()
	preserved := filepath.Join(project, ".agents", "destination-must-remain-untouched")
	mustMkdir(t, filepath.Dir(preserved))
	mustWrite(t, preserved, "sentinel")

	output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
	if err == nil || !strings.Contains(output, "source skill links are not safe to materialize") || !strings.Contains(output, "transitive symlink cycle") {
		t.Fatalf("setup local accepted contained transitive source skill cycle: %v\n%s", err, output)
	}
	if got := string(mustReadFile(t, preserved)); got != "sentinel" {
		t.Fatalf("setup mutated destination before refusing transitive source cycle: %q", got)
	}
}

func TestInstalledBinarySetupLocalRefusesUnsafeSourceSkillLinksBeforeDestinationMutation(t *testing.T) {
	binary := buildInstalledBinary(t)
	for _, test := range []struct {
		name       string
		probe      string
		target     func(source, outside string) string
		wantOutput string
	}{
		{
			name:  "absolute escape",
			probe: "unsafe-probe",
			target: func(_, outside string) string {
				return outside
			},
			wantOutput: "absolute and would escape",
		},
		{
			name:  "ancestor cycle",
			probe: "unsafe-probe",
			target: func(_, _ string) string {
				return ".."
			},
			wantOutput: "points to itself or an ancestor",
		},
		{
			name:  "nested absolute escape",
			probe: filepath.Join("nested-probe", "unsafe-probe"),
			target: func(_, outside string) string {
				return outside
			},
			wantOutput: "absolute and would escape",
		},
		{
			name:  "nested ancestor cycle",
			probe: filepath.Join("nested-probe", "unsafe-probe"),
			target: func(_, _ string) string {
				return filepath.Join("..", "..")
			},
			wantOutput: "points to itself or an ancestor",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
			skillsDir := filepath.Join(source, ".skills")
			mustMkdir(t, skillsDir)
			outside := filepath.Join(t.TempDir(), "outside-runtime")
			mustMkdir(t, outside)
			probe := filepath.Join(skillsDir, test.probe)
			mustMkdir(t, filepath.Dir(probe))
			if err := os.Symlink(test.target(source, outside), probe); err != nil {
				t.Skipf("cannot create source skill symlink: %v", err)
			}
			configDir := filepath.Join(home, "config")
			writeInstallState(t, configDir, source)
			project := t.TempDir()
			preserved := filepath.Join(project, ".agents", "destination-must-remain-untouched")
			mustMkdir(t, filepath.Dir(preserved))
			mustWrite(t, preserved, "sentinel")

			output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project)
			if err == nil || !strings.Contains(output, "source skill links are not safe to materialize") || !strings.Contains(output, test.wantOutput) {
				t.Fatalf("setup local did not refuse unsafe source skill link: %v\n%s", err, output)
			}
			if got := string(mustReadFile(t, preserved)); got != "sentinel" {
				t.Fatalf("setup mutated destination before refusing unsafe source link: %q", got)
			}
		})
	}
}

func TestInstalledBinaryVerifyLocalRefusesUnsafeManagedSkillLinkDrift(t *testing.T) {
	binary := buildInstalledBinary(t)
	for _, test := range []struct {
		name       string
		probe      string
		target     func(project, outside string) string
		wantOutput string
	}{
		{
			name:  "absolute escape",
			probe: "unsafe-probe",
			target: func(_, outside string) string {
				return outside
			},
			wantOutput: "escapes runtime containment",
		},
		{
			name:  "ancestor cycle",
			probe: "unsafe-probe",
			target: func(_, _ string) string {
				return ".."
			},
			wantOutput: "points to itself or an ancestor",
		},
		{
			name:  "nested absolute escape",
			probe: filepath.Join("nested-probe", "unsafe-probe"),
			target: func(_, outside string) string {
				return outside
			},
			wantOutput: "escapes runtime containment",
		},
		{
			name:  "nested ancestor cycle",
			probe: filepath.Join("nested-probe", "unsafe-probe"),
			target: func(_, _ string) string {
				return filepath.Join("..", "..")
			},
			wantOutput: "points to itself or an ancestor",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
			configDir := filepath.Join(home, "config")
			writeInstallState(t, configDir, source)
			project := t.TempDir()
			if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
				t.Fatalf("setup local control: %v\n%s", err, output)
			}
			outside := filepath.Join(t.TempDir(), "outside-runtime")
			mustMkdir(t, outside)
			probe := filepath.Join(project, ".agents", ".skills", test.probe)
			mustMkdir(t, filepath.Dir(probe))
			if err := os.Symlink(test.target(project, outside), probe); err != nil {
				t.Skipf("cannot create installed skill symlink: %v", err)
			}

			output, err := runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
			if err == nil || !strings.Contains(output, test.wantOutput) {
				t.Fatalf("verify local accepted unsafe managed skill link: %v\n%s", err, output)
			}
		})
	}
}

func TestInstalledBinaryVerifyLocalRefusesContainedTransitiveManagedSkillCycle(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()
	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local control: %v\n%s", err, output)
	}
	agentsDir := filepath.Join(project, ".agents")
	cycleTarget := filepath.Join(agentsDir, "cycle-target")
	mustMkdir(t, cycleTarget)
	probe := filepath.Join(agentsDir, ".skills", "transitive-cycle-probe")
	if err := os.Symlink(filepath.Join("..", "cycle-target"), probe); err != nil {
		t.Skipf("cannot create transitive installed skill link: %v", err)
	}
	if err := os.Symlink(filepath.Join("..", ".skills", "transitive-cycle-probe"), filepath.Join(cycleTarget, "back")); err != nil {
		t.Skipf("cannot close transitive installed skill cycle: %v", err)
	}

	output, err := runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
	if err == nil || !strings.Contains(output, "transitive symlink cycle") {
		t.Fatalf("verify local accepted contained transitive managed skill cycle: %v\n%s", err, output)
	}
}

func TestInstalledBinaryVerifyLocalInspectsEveryManagedSkillSurface(t *testing.T) {
	binary := buildInstalledBinary(t)
	const managedRepoSkillName = "relux-agents-infra"
	home := t.TempDir()
	source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()
	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local control: %v\n%s", err, output)
	}
	outside := filepath.Join(t.TempDir(), "outside-runtime")
	mustMkdir(t, outside)
	for _, surface := range []struct {
		root string
		name string
	}{
		{root: filepath.Join(project, ".agents", ".skills"), name: "surface-escape-probe"},
		{root: filepath.Join(project, ".agents", "skills"), name: managedRepoSkillName},
		{root: filepath.Join(project, ".claude", "skills"), name: managedRepoSkillName},
		{root: filepath.Join(project, ".codex", "skills"), name: managedRepoSkillName},
	} {
		probe := filepath.Join(surface.root, surface.name)
		if err := os.Remove(probe); err != nil && !os.IsNotExist(err) {
			t.Fatalf("remove existing managed skill link %s: %v", probe, err)
		}
		if err := os.Symlink(outside, probe); err != nil {
			t.Skipf("cannot create installed skill symlink: %v", err)
		}
		output, err := runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
		if err == nil || !strings.Contains(output, probe) || !strings.Contains(output, "escapes runtime containment") {
			t.Fatalf("verify local did not inspect managed surface %s: %v\n%s", surface.root, err, output)
		}
		if err := os.Remove(probe); err != nil {
			t.Fatalf("remove surface probe %s: %v", probe, err)
		}
		if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
			t.Fatalf("setup local did not restore managed skill surface %s: %v\n%s", surface.root, err, output)
		}
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

func TestInstalledBinaryVerifyLocalRecursesEveryManagedSkillPackage(t *testing.T) {
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	source := seedRuntimeSource(t, filepath.Join(home, ".agents"))
	configDir := filepath.Join(home, "config")
	writeInstallState(t, configDir, source)
	project := t.TempDir()
	if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
		t.Fatalf("setup local control: %v\n%s", err, output)
	}
	out := filepath.Join(t.TempDir(), "outside-runtime")
	mustMkdir(t, out)
	for _, packageDir := range []string{
		filepath.Join(project, ".agents", ".skills", "relux-agents-infra"),
		filepath.Join(project, ".agents", "skills", "relux-agents-infra"),
		filepath.Join(project, ".claude", "skills", "relux-agents-infra"),
		filepath.Join(project, ".codex", "skills", "relux-agents-infra"),
	} {
		info, err := os.Lstat(packageDir)
		if err != nil {
			t.Fatalf("Lstat(%s): %v", packageDir, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(packageDir); err != nil {
				t.Fatalf("remove managed package link %s: %v", packageDir, err)
			}
			mustMkdir(t, packageDir)
		}
		probe := filepath.Join(packageDir, "nested-probe", "escape")
		mustMkdir(t, filepath.Dir(probe))
		if err := os.Symlink(out, probe); err != nil {
			t.Skipf("cannot create nested installed skill symlink: %v", err)
		}
		output, err := runInstalledBinary(t, binary, home, configDir, "verify", "local", project)
		if err == nil || !strings.Contains(output, probe) || !strings.Contains(output, "escapes runtime containment") {
			t.Fatalf("verify local did not recurse through managed package %s: %v\n%s", packageDir, err, output)
		}
		if output, err := runInstalledBinary(t, binary, home, configDir, "setup", "local", project); err != nil {
			t.Fatalf("setup local did not restore managed package after drift probe: %v\n%s", err, output)
		}
	}
}

func assertContainedAcyclicSymlinks(t *testing.T, root string) {
	t.Helper()
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("Abs(%s): %v", root, err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", root, err)
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink == 0 {
			return nil
		}
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		target, err = filepath.Abs(target)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			t.Fatalf("managed skill symlink escapes runtime: %s -> %s", path, target)
		}
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		pathRel, err := filepath.Rel(target, pathAbs)
		if err == nil && pathRel != ".." && !strings.HasPrefix(pathRel, ".."+string(os.PathSeparator)) {
			t.Fatalf("managed skill symlink points to its own ancestor: %s -> %s", path, target)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inspect managed symlinks under %s: %v", root, err)
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

// Production call sites: setup local installs dange -> generated agents-infra
// target-yolo -> runDirectProviderYoloTarget -> provider exec. This matrix
// fails against revision 2's caller-flag rejection, revision 3's leading--d
// origin inference, and revision 5's forgeable argv marker: dange routes accept
// the matrix, while canonical target aliases refuse danger and forged markers.
func TestInstalledLocalProviderAliasesScopeImplicitFlagForwardingToDangeChain(t *testing.T) {
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

	fakeBin := t.TempDir()
	recordDir := t.TempDir()
	for _, provider := range []string{"codex", "claude"} {
		record := filepath.Join(recordDir, provider)
		mustWrite(t, filepath.Join(fakeBin, provider), "#!/bin/sh\nprintf '%s\\0' \"$@\" > \""+record+"\"\n")
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
		name       string
		alias      string
		provider   string
		nativeFlag string
		args       []string
		wantCaller []string
	}{
		{name: "openai caller short danger", alias: "openai-dange", provider: "codex", nativeFlag: "--dangerously-bypass-approvals-and-sandbox", args: []string{"-d", "exec", "space value", "Հայերեն"}, wantCaller: []string{"exec", "space value", "Հայերեն"}},
		{name: "openai caller danger", alias: "openai-dange", provider: "codex", nativeFlag: "--dangerously-bypass-approvals-and-sandbox", args: []string{"--danger", "exec", "line 1\nline 2"}, wantCaller: []string{"exec", "line 1\nline 2"}},
		{name: "openai caller yolo", alias: "openai-dange", provider: "codex", nativeFlag: "--dangerously-bypass-approvals-and-sandbox", args: []string{"--yolo", "exec", "tab\tvalue"}, wantCaller: []string{"exec", "tab\tvalue"}},
		{name: "openai caller model", alias: "openai-dange", provider: "codex", nativeFlag: "--dangerously-bypass-approvals-and-sandbox", args: []string{"--model", "gpt-5.6-sol", "exec", "", "inspect token"}, wantCaller: []string{"--model", "gpt-5.6-sol", "exec", "", "inspect token"}},
		{name: "openai redundant danger", alias: "openai-dange", provider: "codex", nativeFlag: "--dangerously-bypass-approvals-and-sandbox", args: []string{"-d", "--danger", "--yolo", "--dangerously-bypass-approvals-and-sandbox", "exec", "all danger"}, wantCaller: []string{"exec", "all danger"}},
		{name: "anthropic caller short danger", alias: "anthropic-dange", provider: "claude", nativeFlag: "--dangerously-skip-permissions", args: []string{"-d", "space value", "Հայերեն"}, wantCaller: []string{"space value", "Հայերեն"}},
		{name: "anthropic caller danger", alias: "anthropic-dange", provider: "claude", nativeFlag: "--dangerously-skip-permissions", args: []string{"--danger", "line 1\nline 2"}, wantCaller: []string{"line 1\nline 2"}},
		{name: "anthropic caller yolo", alias: "anthropic-dange", provider: "claude", nativeFlag: "--dangerously-skip-permissions", args: []string{"--yolo", "tab\tvalue"}, wantCaller: []string{"tab\tvalue"}},
		{name: "anthropic caller model", alias: "anthropic-dange", provider: "claude", nativeFlag: "--dangerously-skip-permissions", args: []string{"--model", "claude-opus-5", "", "inspect token"}, wantCaller: []string{"--model", "claude-opus-5", "", "inspect token"}},
		{name: "anthropic redundant danger", alias: "anthropic-dange", provider: "claude", nativeFlag: "--dangerously-skip-permissions", args: []string{"-d", "--danger", "--yolo", "--dangerously-skip-permissions", "all danger"}, wantCaller: []string{"all danger"}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			command := exec.Command(filepath.Join(project, ".local", "bin", testCase.alias), testCase.args...)
			command.Dir = project
			command.Env = installedAliasEnv()
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("installed %s: %v\n%s", testCase.alias, err, output)
			}
			data, err := os.ReadFile(filepath.Join(recordDir, testCase.provider))
			if err != nil {
				t.Fatal(err)
			}
			args := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
			count := 0
			for _, arg := range args {
				if arg == testCase.nativeFlag {
					count++
				}
				if arg == revision5DirectProviderYoloCallSiteMarker {
					t.Fatalf("retired revision-5 marker reached provider argv: %#v", args)
				}
			}
			if count != 1 {
				t.Fatalf("provider argv danger count = %d, want 1: %#v", count, args)
			}
			if !orderedSubsequence(args, testCase.wantCaller) {
				t.Fatalf("provider argv did not preserve caller non-danger bytes/order: got %#v want subsequence %#v", args, testCase.wantCaller)
			}
		})
	}

	canonicalCases := []struct {
		alias    string
		provider string
	}{
		{alias: "openai-infra", provider: "codex"},
		{alias: "anthropic-infra", provider: "claude"},
	}
	for _, testCase := range canonicalCases {
		refusals := []struct {
			name string
			args []string
		}{
			{name: "caller leading d", args: []string{"-d", "--model", "caller-model"}},
			{name: "forged revision 5 marker", args: []string{revision5DirectProviderYoloCallSiteMarker, "--model", "caller-model"}},
			{name: "forged marker-like assignment", args: []string{revision5DirectProviderYoloCallSiteMarker + "=forged", "--model", "caller-model"}},
		}
		for _, refusal := range refusals {
			t.Run(testCase.alias+" refuses "+refusal.name, func(t *testing.T) {
				record := filepath.Join(recordDir, testCase.provider)
				if err := os.Remove(record); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				command := exec.Command(filepath.Join(project, ".local", "bin", testCase.alias), refusal.args...)
				command.Dir = project
				command.Env = installedAliasEnv()
				output, err := command.CombinedOutput()
				if err == nil || !strings.Contains(string(output), "flag provided but not defined") {
					t.Fatalf("installed canonical alias accepted %s: err=%v\n%s", refusal.name, err, output)
				}
				if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
					t.Fatalf("canonical refusal reached provider side effect: %v", statErr)
				}
			})
		}
	}
}

func orderedSubsequence(got, want []string) bool {
	next := 0
	for _, arg := range got {
		if next < len(want) && arg == want[next] {
			next++
		}
	}
	return next == len(want)
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
