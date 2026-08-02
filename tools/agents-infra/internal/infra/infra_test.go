package infra

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

const (
	modelAvailabilityPolicyFixture = "retry the preferred model before choosing an autonomous fallback"
	forcedFitPolicyFixture         = "do not fake an impossible platform model with flags, stubs, or mocks"
	imageIntakeWorkflowFixture     = "agents-attachments stage-images"
)

func TestLocalLayout(t *testing.T) {
	layout, err := LocalLayout("/src/repo", "/tmp/project")
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if layout.AgentsDir != "/tmp/project/.agents" {
		t.Fatalf("AgentsDir = %q", layout.AgentsDir)
	}
	if layout.ClaudeDir != "/tmp/project/.claude" {
		t.Fatalf("ClaudeDir = %q", layout.ClaudeDir)
	}
	if layout.CodexDir != "/tmp/project/.codex" {
		t.Fatalf("CodexDir = %q", layout.CodexDir)
	}
}

func TestCLIWrapperNameForWindows(t *testing.T) {
	if got := cliWrapperName("windows"); got != "agents-infra.cmd" {
		t.Fatalf("cliWrapperName(windows) = %q", got)
	}
}

func TestCLIWrapperBodyForWindows(t *testing.T) {
	body := cliWrapperBody("windows", `C:\src\relux-agents-infra`, `C:\project\.local\bin\.agents-infra-build\agents-infra-local.exe`)
	if !strings.Contains(body, "AGENTS_INFRA_SOURCE_DIR=C:\\src\\relux-agents-infra") {
		t.Fatalf("windows wrapper body missing source dir: %q", body)
	}
	if !strings.Contains(body, `set "AGENTS_INFRA_BINARY=C:\project\.local\bin\.agents-infra-build\agents-infra-local.exe"`) {
		t.Fatalf("windows wrapper body does not build into the target: %q", body)
	}
	if strings.Contains(body, `%AGENTS_INFRA_SOURCE_DIR%\.temp`) {
		t.Fatalf("windows wrapper body still writes into the source checkout: %q", body)
	}
	if !strings.Contains(body, "AGENTS_INFRA_CALLER_CWD=%CD%") {
		t.Fatalf("windows wrapper body missing caller cwd preservation: %q", body)
	}
	if !strings.Contains(body, `go build -o "%AGENTS_INFRA_BINARY%" .`) {
		t.Fatalf("windows wrapper body missing go build invocation: %q", body)
	}
	if !strings.Contains(body, `"%AGENTS_INFRA_BINARY%" %*`) {
		t.Fatalf("windows wrapper body missing built binary invocation: %q", body)
	}
	if !strings.Contains(body, "exit /b %ERRORLEVEL%") {
		t.Fatalf("windows wrapper body missing exit-code propagation: %q", body)
	}
}

func TestCLIWrapperBodyForUnixPreservesCallerCWD(t *testing.T) {
	body := cliWrapperBody("darwin", `/src/relux-agents-infra`, `/project/.local/bin/.agents-infra-build/agents-infra-local`)
	if !strings.Contains(body, `export AGENTS_INFRA_SOURCE_DIR="/src/relux-agents-infra"`) {
		t.Fatalf("unix wrapper body missing source dir export: %q", body)
	}
	if !strings.Contains(body, `AGENTS_INFRA_BINARY="/project/.local/bin/.agents-infra-build/agents-infra-local"`) {
		t.Fatalf("unix wrapper body does not build into the target: %q", body)
	}
	if strings.Contains(body, `$AGENTS_INFRA_SOURCE_DIR/.temp`) {
		t.Fatalf("unix wrapper body still writes into the source checkout: %q", body)
	}
	if !strings.Contains(body, "AGENTS_INFRA_CALLER_CWD=$(pwd)") {
		t.Fatalf("unix wrapper body missing caller cwd capture: %q", body)
	}
	if !strings.Contains(body, "export AGENTS_INFRA_CALLER_CWD") {
		t.Fatalf("unix wrapper body missing caller cwd export: %q", body)
	}
	if !strings.Contains(body, `go build -o "$AGENTS_INFRA_BINARY" .`) || !strings.Contains(body, `exec "$AGENTS_INFRA_BINARY" "$@"`) {
		t.Fatalf("unix wrapper body should build and execute the Go binary: %q", body)
	}
}

func TestAgentsAttachmentsWrapperBodyForWindowsPropagatesSelectedLauncherExit(t *testing.T) {
	body := agentsAttachmentsWrapperBody("windows")
	for _, want := range []string{
		"if exist \"%DIR%agents-infra.cmd\" (\r\n  \"%DIR%agents-infra.cmd\" attachments %*\r\n  exit /b\r\n)",
		"if exist \"%DIR%agents-infra.exe\" (\r\n  \"%DIR%agents-infra.exe\" attachments %*\r\n  exit /b\r\n)",
		"agents-infra attachments %*\r\nexit /b\r\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("windows agents-attachments wrapper missing %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{
		"& exit /b %ERRORLEVEL%",
		"exit /b %ERRORLEVEL%",
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("windows agents-attachments wrapper contains stale-errorlevel pattern %q:\n%s", unwanted, body)
		}
	}
	if strings.Index(body, "agents-infra.cmd") > strings.Index(body, "agents-infra.exe") {
		t.Fatalf("windows wrapper should prefer sibling .cmd before .exe:\n%s", body)
	}
}

func TestAgentsAttachmentsWrapperBodyForUnixDelegatesToSiblingOrPath(t *testing.T) {
	body := agentsAttachmentsWrapperBody("darwin")
	for _, want := range []string{
		`TARGET="$DIR/agents-infra"`,
		"TARGET=agents-infra",
		`"$TARGET" attachments "$@"`,
		`exit "$STATUS"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("unix agents-attachments wrapper missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "python") {
		t.Fatalf("unix agents-attachments wrapper should not mention Python:\n%s", body)
	}
}

func TestAgentsAttachmentsUnixWrapperPreservesGoRunUsageExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell launcher test")
	}
	dir := t.TempDir()
	launcher := filepath.Join(dir, "agents-attachments")
	delegated := filepath.Join(dir, "agents-infra")
	mustWrite(t, launcher, agentsAttachmentsWrapperBody(runtime.GOOS))
	mustWrite(t, delegated, "#!/usr/bin/env sh\nprintf '%s\\n' 'Usage: agents-attachments list' >&2\nprintf '%s\\n' 'exit status 2' >&2\nexit 1\n")
	if err := os.Chmod(launcher, 0o755); err != nil {
		t.Fatalf("Chmod(%s): %v", launcher, err)
	}
	if err := os.Chmod(delegated, 0o755); err != nil {
		t.Fatalf("Chmod(%s): %v", delegated, err)
	}

	command := exec.Command(launcher)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	err := command.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("launcher error = %v, want exit code 2", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Fatalf("launcher exit code = %d, want 2; stderr:\n%s", exitErr.ExitCode(), stderr.String())
	}
	if strings.Contains(stderr.String(), "exit status 2") {
		t.Fatalf("launcher leaked go run status trailer:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage: agents-attachments list") {
		t.Fatalf("launcher stderr missing delegated usage:\n%s", stderr.String())
	}
}

func TestSetupLocalCreatesInstalledRuntime(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	var logs bytes.Buffer
	if err := Setup(Options{Layout: layout, Stdout: &logs}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertExists(t, filepath.Join(project, ".agents", ".instructions", "INSTRUCTIONS.md"))
	assertNoPath(t, filepath.Join(project, ".agents", ".git"))
	assertSymlink(t, filepath.Join(project, ".agents", "skills", "pdf"), filepath.Join(project, ".agents", ".skills", "pdf"))
	assertSymlink(t, filepath.Join(project, ".claude", "instructions"), filepath.Join(project, ".agents", ".instructions"))
	assertSymlink(t, filepath.Join(project, ".claude", "skills", "pdf"), filepath.Join(project, ".agents", "skills", "pdf"))
	assertRenderedInstructions(t, filepath.Join(project, ".codex", "AGENTS.md"))
	assertRenderedInstructions(t, filepath.Join(project, "AGENTS.md"))
	assertFileContains(t, filepath.Join(project, ".codex", "AGENTS.md"), modelAvailabilityPolicyFixture)
	assertFileContains(t, filepath.Join(project, "AGENTS.md"), modelAvailabilityPolicyFixture)
	assertFileContains(t, filepath.Join(project, ".agents", ".instructions", "INSTRUCTIONS_WORKFLOW.md"), modelAvailabilityPolicyFixture)
	assertFileContains(t, filepath.Join(project, ".codex", "AGENTS.md"), forcedFitPolicyFixture)
	assertFileContains(t, filepath.Join(project, "AGENTS.md"), forcedFitPolicyFixture)
	assertFileContains(t, filepath.Join(project, ".agents", ".instructions", "INSTRUCTIONS_WORKFLOW.md"), forcedFitPolicyFixture)
	assertFileContains(t, filepath.Join(project, ".codex", "AGENTS.md"), imageIntakeWorkflowFixture)
	assertFileContains(t, filepath.Join(project, "AGENTS.md"), imageIntakeWorkflowFixture)
	assertFileContains(t, filepath.Join(project, ".agents", ".instructions", "INSTRUCTIONS_ATTACHMENTS.md"), imageIntakeWorkflowFixture)
	assertSymlink(t, filepath.Join(project, ".codex", "skills", "pdf"), filepath.Join(project, ".agents", "skills", "pdf"))
	assertNoPath(t, filepath.Join(project, ".agents", ".scripts", "agents-attachments"))
	assertRegularFile(t, filepath.Join(project, ".local", "bin", "agents-attachments"))
	assertFileContains(t, filepath.Join(project, ".local", "bin", "agents-attachments"), `"$TARGET" attachments "$@"`)
	assertFileNotContains(t, filepath.Join(project, ".local", "bin", "agents-attachments"), "python")

	launcher := filepath.Join(project, ".local", "bin", "agents-infra")
	data, err := os.ReadFile(launcher)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", launcher, err)
	}
	if !strings.Contains(string(data), source) {
		t.Fatalf("launcher does not reference source repo: %q", string(data))
	}

	claudeEntry := filepath.Join(project, ".claude", "CLAUDE.md")
	entry, err := os.ReadFile(claudeEntry)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", claudeEntry, err)
	}
	if !strings.Contains(string(entry), "@instructions/INSTRUCTIONS.md") {
		t.Fatalf("CLAUDE.md should reference Claude runtime instructions: %q", string(entry))
	}
}

func TestSetupRemovesStaleRepoSkillSelfLinks(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	staleLink := filepath.Join(project, ".agents", "skills", "legacy-agents-infra")
	mustMkdir(t, filepath.Dir(staleLink))
	if err := os.Symlink(layout.AgentsDir, staleLink); err != nil {
		t.Fatalf("Symlink(%s): %v", staleLink, err)
	}
	staleClaudeLink := filepath.Join(project, ".claude", "skills", "legacy-agents-infra")
	mustMkdir(t, filepath.Dir(staleClaudeLink))
	if err := os.Symlink(staleLink, staleClaudeLink); err != nil {
		t.Fatalf("Symlink(%s): %v", staleClaudeLink, err)
	}
	staleCodexLink := filepath.Join(project, ".codex", "skills", "legacy-agents-infra")
	mustMkdir(t, filepath.Dir(staleCodexLink))
	if err := os.Symlink(staleLink, staleCodexLink); err != nil {
		t.Fatalf("Symlink(%s): %v", staleCodexLink, err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, staleLink)
	assertNoPath(t, staleClaudeLink)
	assertNoPath(t, staleCodexLink)
	assertSymlink(t, filepath.Join(project, ".agents", "skills", repoSkillName), filepath.Join(project, ".agents"))
}

func TestRefreshLinksKeepsCanonicalRepoSkillWhenStaleSelfLinkCannotBeRemoved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission smoke is Unix-only")
	}
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	skillsDir := filepath.Join(project, ".agents", "skills")
	staleLink := filepath.Join(skillsDir, "legacy-agents-infra")
	if err := os.Symlink(layout.AgentsDir, staleLink); err != nil {
		t.Fatalf("Symlink(%s): %v", staleLink, err)
	}
	if err := os.Chmod(skillsDir, 0o555); err != nil {
		t.Fatalf("Chmod(%s): %v", skillsDir, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(skillsDir, 0o755)
	})

	var logs bytes.Buffer
	if err := RefreshLinks(Options{Layout: layout, Stdout: &logs}); err != nil {
		t.Fatalf("RefreshLinks should tolerate permission-denied stale cleanup: %v\nlogs:\n%s", err, logs.String())
	}

	assertSymlink(t, filepath.Join(skillsDir, repoSkillName), filepath.Join(project, ".agents"))
	assertSymlink(t, staleLink, filepath.Join(project, ".agents"))
	assertNoPath(t, filepath.Join(project, ".claude", "skills", "legacy-agents-infra"))
	assertNoPath(t, filepath.Join(project, ".codex", "skills", "legacy-agents-infra"))
	if !strings.Contains(logs.String(), "Skipped stale repo skill link") {
		t.Fatalf("expected stale-link skip log, got:\n%s", logs.String())
	}
}

func TestRefreshLinksReplacesLegacyPythonAttachmentsHelper(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	legacyScript := filepath.Join(project, ".agents", ".scripts", "agents-attachments")
	mustMkdir(t, filepath.Dir(legacyScript))
	mustWrite(t, legacyScript, "#!/usr/bin/env python3\n")
	mustMkdir(t, filepath.Join(project, ".local", "bin"))
	legacyLink := filepath.Join(project, ".local", "bin", "agents-attachments")
	if err := os.Remove(legacyLink); err != nil {
		t.Fatalf("Remove generated agents-attachments launcher: %v", err)
	}
	if err := os.Symlink(legacyScript, legacyLink); err != nil {
		t.Fatalf("Symlink legacy agents-attachments: %v", err)
	}

	if err := RefreshLinks(Options{Layout: layout}); err != nil {
		t.Fatalf("RefreshLinks: %v", err)
	}

	assertNoPath(t, legacyScript)
	assertRegularFile(t, filepath.Join(project, ".local", "bin", "agents-attachments"))
	assertFileContains(t, filepath.Join(project, ".local", "bin", "agents-attachments"), "agents-infra")
	assertFileContains(t, filepath.Join(project, ".local", "bin", "agents-attachments"), "attachments")
	assertFileNotContains(t, filepath.Join(project, ".local", "bin", "agents-attachments"), "python")
}

func TestSyncSkipsGitAndTemp(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".agents", ".git"))
	assertNoPath(t, filepath.Join(project, ".agents", ".temp"))
	assertNoPath(t, filepath.Join(project, ".agents", ".gitignore"))
	assertNoPath(t, filepath.Join(project, ".agents", ".task-board"))
	assertNoPath(t, filepath.Join(project, ".agents", "task-board.config.json"))
}

func TestSyncSkipsNestedGitMetadata(t *testing.T) {
	source := seedSourceRepo(t)
	mustMkdir(t, filepath.Join(source, ".skills", "pdf", ".git"))
	mustWrite(t, filepath.Join(source, ".skills", "pdf", ".git", "config"), "nested")
	mustWrite(t, filepath.Join(source, ".skills", "pdf", ".gitignore"), "nested-ignore")
	mustMkdir(t, filepath.Join(source, ".skills", "pdf", "examples", ".git"))
	mustWrite(t, filepath.Join(source, ".skills", "pdf", "examples", ".git", "HEAD"), "ref: refs/heads/main")

	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".agents", ".skills", "pdf", ".git"))
	assertNoPath(t, filepath.Join(project, ".agents", ".skills", "pdf", ".gitignore"))
	assertNoPath(t, filepath.Join(project, ".agents", ".skills", "pdf", "examples", ".git"))
}

func TestSyncSkipsSourceLocalRuntimeDirs(t *testing.T) {
	source := seedSourceRepo(t)
	mustMkdir(t, filepath.Join(source, ".agents", ".configs"))
	mustWrite(t, filepath.Join(source, ".agents", ".configs", "codex-config.toml"), "nested")
	mustMkdir(t, filepath.Join(source, ".claude", "skills"))
	mustWrite(t, filepath.Join(source, ".claude", "settings.json"), "nested")
	mustMkdir(t, filepath.Join(source, ".codex", "skills"))
	mustWrite(t, filepath.Join(source, ".codex", "config.toml"), "nested")
	mustMkdir(t, filepath.Join(source, ".local", "bin"))
	mustWrite(t, filepath.Join(source, ".local", "bin", "agents-infra"), "nested")
	mustMkdir(t, filepath.Join(source, ".planning"))
	mustWrite(t, filepath.Join(source, ".planning", "plan.md"), "nested")
	mustMkdir(t, filepath.Join(source, ".relux"))
	mustWrite(t, filepath.Join(source, ".relux", "state.json"), "nested")

	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".agents", ".agents"))
	assertNoPath(t, filepath.Join(project, ".agents", ".claude"))
	assertNoPath(t, filepath.Join(project, ".agents", ".codex"))
	assertNoPath(t, filepath.Join(project, ".agents", ".local"))
	assertNoPath(t, filepath.Join(project, ".agents", ".planning"))
	assertNoPath(t, filepath.Join(project, ".agents", ".relux"))
}

func TestDoctor(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	report := mustDoctor(t, layout)
	if !report.AgentsGitFree || !report.ClaudeLinked || report.CodexLinked || !report.CodexRendered || !report.CodexProjectRendered || report.CodexConfigPresent || report.CodexConfigLinked || report.CodexConfigGenerated || report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "global" || len(report.CodexMCPEnabled) != 0 || !report.CodexPrimaryConfigValid || report.CodexPrimarySession.Model.Present || report.CodexPrimarySession.ReasoningEffort.Present || report.CodexPrimarySession.YoloMode.Present || !report.HelpersLinked || !report.InfraSkillLink {
		t.Fatalf("unexpected doctor report: %+v", report)
	}
}

func TestDoctorReportsComposedPrimarySessionAndMCPPolicy(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	parentConfig := filepath.Join(parent, ".agents", ".configs", projectConfigFileName)
	mustWrite(t, parentConfig, `
[mcp]
enabled_servers = ["figma"]

[agents.codex.primary_session]
model = "parent-model"
yolo_mode = true
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	childConfig := filepath.Join(child, ".agents", ".configs", projectConfigFileName)
	mustWrite(t, childConfig, `
[mcp]
enabled_servers = ["lldb", "figma"]

[agents.codex.primary_session]
reasoning_effort = "xhigh"
yolo_mode = false
`)

	layout, err := LocalLayout("", child)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	report, err := Doctor(layout)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if !report.CodexPrimaryConfigValid {
		t.Fatalf("CodexPrimaryConfigValid = false: %+v", report)
	}
	if !reflect.DeepEqual(report.CodexMCPEnabled, []string{"figma", "lldb"}) {
		t.Fatalf("CodexMCPEnabled = %#v, want composed order", report.CodexMCPEnabled)
	}
	if got := report.CodexPrimarySession.Model; !got.Present || got.Value != "parent-model" || got.Source != parentConfig {
		t.Fatalf("primary model = %#v, want inherited parent value", got)
	}
	if got := report.CodexPrimarySession.ReasoningEffort; !got.Present || got.Value != "xhigh" || got.Source != childConfig {
		t.Fatalf("primary reasoning effort = %#v, want child value", got)
	}
	if got := report.CodexPrimarySession.YoloMode; !got.Present || got.Value || got.Source != childConfig {
		t.Fatalf("primary yolo mode = %#v, want explicit child false", got)
	}
}

func TestDoctorIgnoresHomeProjectConfigWithoutProjectOptIn(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "work", "project")
	mustMkdir(t, project)
	mustMkdir(t, filepath.Join(home, ".agents", ".configs"))
	mustWrite(t, filepath.Join(home, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "must-not-apply"
yolo_mode = true
`)
	t.Setenv("HOME", home)

	layout, err := LocalLayout("", project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	report, err := Doctor(layout)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if report.CodexPrimarySession.Model.Present || report.CodexPrimarySession.YoloMode.Present {
		t.Fatalf("home project config was treated as project opt-in: %+v", report.CodexPrimarySession)
	}
}

func TestDoctorFailsClosedOnInvalidComposedProjectConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "parent-model"
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	invalidConfig := filepath.Join(child, ".agents", ".configs", projectConfigFileName)
	mustWrite(t, invalidConfig, `
[agents.codex.primary_session]
yolo_mode = "false"
`)

	layout, err := LocalLayout("", child)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	report, err := Doctor(layout)
	if err == nil {
		t.Fatal("Doctor succeeded with invalid child project config")
	}
	if report.CodexPrimaryConfigValid {
		t.Fatalf("CodexPrimaryConfigValid = true after error: %+v", report)
	}
	if !strings.Contains(err.Error(), invalidConfig) || !strings.Contains(err.Error(), codexPrimaryYoloModeField) {
		t.Fatalf("Doctor error = %q, want source path and field", err)
	}
}

func TestDoctorDetectsProjectLocalCodexConfigShadowing(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	mustMkdir(t, filepath.Join(project, ".codex"))
	mustWrite(t, filepath.Join(project, ".codex", "config.toml"), "model = \"gpt-5.4\"\n")

	report := mustDoctor(t, layout)
	if !report.CodexConfigPresent {
		t.Fatalf("expected local Codex config to be present: %+v", report)
	}
	if !report.CodexConfigShadowsGlobal {
		t.Fatalf("expected local Codex config to shadow global config: %+v", report)
	}
	if report.CodexConfigLinked {
		t.Fatalf("custom local Codex config should not be reported as linked: %+v", report)
	}
	if report.CodexConfigEffective != "project-local" {
		t.Fatalf("CodexConfigEffective = %q, want project-local", report.CodexConfigEffective)
	}
}

func TestSetupGlobalDoesNotInstallCLIWrapper(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(home, ".local", "bin", "agents-infra"))
	assertNoPath(t, filepath.Join(home, ".local", "bin", "agents-infra.cmd"))
}

func TestSetupGlobalRemovesStaleProjectConfig(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}
	staleConfig := filepath.Join(home, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(staleConfig))
	mustWrite(t, staleConfig, "[mcp]\nenabled_servers = [\"figma\"]\n")

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, staleConfig)
}

func TestSetupRemovesGeneratedArtifacts(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	mustMkdir(t, filepath.Join(project, ".agents", ".rules"))
	mustMkdir(t, filepath.Join(project, ".claude"))
	mustMkdir(t, filepath.Join(project, ".codex", "rules"))
	mustMkdir(t, filepath.Join(project, ".local", "bin"))

	mustWrite(t, filepath.Join(project, ".agents", ".rules", "default.rules.bak.1"), "stale")
	mustWrite(t, filepath.Join(project, ".agents", ".DS_Store"), "junk")
	mustWrite(t, filepath.Join(project, ".claude", "settings.json.bak.1"), "stale")
	mustWrite(t, filepath.Join(project, ".codex", "rules", "default.rules.bak.1"), "stale")
	mustWrite(t, filepath.Join(project, ".local", "bin", "agents-infra.bak.1"), "stale")

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".agents", ".rules", "default.rules.bak.1"))
	assertNoPath(t, filepath.Join(project, ".agents", ".DS_Store"))
	assertNoPath(t, filepath.Join(project, ".claude", "settings.json.bak.1"))
	assertNoPath(t, filepath.Join(project, ".codex", "rules", "default.rules.bak.1"))
	assertNoPath(t, filepath.Join(project, ".local", "bin", "agents-infra.bak.1"))
}

func TestSetupReplacesManagedPathsWithoutBackups(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	mustMkdir(t, filepath.Join(project, ".claude"))
	mustMkdir(t, filepath.Join(project, ".codex", "rules"))
	mustMkdir(t, filepath.Join(project, ".local", "bin"))

	mustWrite(t, filepath.Join(project, ".claude", "settings.json"), "custom")
	if err := os.Symlink(filepath.Join(project, ".agents", ".configs", "codex-config.toml"), filepath.Join(project, ".codex", "config.toml")); err != nil {
		t.Fatalf("Symlink(project codex config): %v", err)
	}
	mustWrite(t, filepath.Join(project, ".codex", "rules", "default.rules"), "custom")
	mustWrite(t, filepath.Join(project, ".local", "bin", "agents-infra"), "#!/bin/sh\nexit 0\n")

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertSymlink(t, filepath.Join(project, ".claude", "settings.json"), filepath.Join(project, ".agents", ".configs", "claude-settings.json"))
	assertNoPath(t, filepath.Join(project, ".codex", "config.toml"))
	assertSymlink(t, filepath.Join(project, ".codex", "rules", "default.rules"), filepath.Join(project, ".agents", ".rules", "default.rules"))
	assertNoGeneratedArtifacts(t, project)
}

func TestSetupLocalPreservesCustomProjectCodexConfig(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	mustMkdir(t, filepath.Join(project, ".codex"))
	mustWrite(t, filepath.Join(project, ".codex", "config.toml"), "model = \"gpt-5.4\"\n")

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertFileContains(t, filepath.Join(project, ".codex", "config.toml"), "gpt-5.4")
	report := mustDoctor(t, layout)
	if !report.CodexConfigPresent || !report.CodexConfigShadowsGlobal {
		t.Fatalf("custom project Codex config should be preserved and reported as shadowing: %+v", report)
	}
}

func TestSetupLocalPreservesExistingNativeAgentConfigsOnResync(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout, CodexConfigMode: CodexConfigModeLocal}); err != nil {
		t.Fatalf("initial Setup: %v", err)
	}

	codexConfig := filepath.Join(project, ".agents", ".configs", "codex-config.toml")
	claudeSettings := filepath.Join(project, ".agents", ".configs", "claude-settings.json")
	mustWrite(t, codexConfig, "model = \"gpt-5.6-terra\"\nmodel_reasoning_effort = \"xhigh\"\n")
	mustWrite(t, claudeSettings, "{\n  \"model\": \"claude-sonnet-5\",\n  \"permissions\": {\"defaultMode\": \"bypassPermissions\"}\n}\n")

	mustWrite(t, filepath.Join(source, ".configs", "codex-config.toml"), "model = \"source-default-overwrite\"\n")
	mustWrite(t, filepath.Join(source, ".configs", "claude-settings.json"), "{\"model\":\"source-default-overwrite\"}\n")
	mustWrite(t, filepath.Join(source, ".configs", "codex-mcp-servers.toml"), `[servers.updated]
url = "https://example.test/mcp"
`)

	if err := Setup(Options{Layout: layout, CodexConfigMode: CodexConfigModeLocal}); err != nil {
		t.Fatalf("second Setup: %v", err)
	}

	assertFileContains(t, codexConfig, "gpt-5.6-terra")
	assertFileNotContains(t, codexConfig, "source-default-overwrite")
	assertFileContains(t, claudeSettings, "claude-sonnet-5")
	assertFileContains(t, claudeSettings, "bypassPermissions")
	assertFileNotContains(t, claudeSettings, "source-default-overwrite")
	assertFileContains(t, filepath.Join(project, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.updated]")
	assertFileContains(t, filepath.Join(project, ".codex", "config.toml"), "gpt-5.6-terra")
	assertFileNotContains(t, filepath.Join(project, ".codex", "config.toml"), "source-default-overwrite")
	assertSymlink(t, filepath.Join(project, ".claude", "settings.json"), claudeSettings)
}

func TestSetupLocalProjectMCPOptInInstallsCodexLocalLauncher(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	mustWrite(t, filepath.Join(project, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"figma\"]\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".codex", "config.toml"))
	launcherPath := filepath.Join(project, ".local", "bin", "codex-local")
	assertFileContains(t, launcherPath, generatedCodexConfigMarker)
	assertFileContains(t, launcherPath, "exec \"$DIR/agents-infra\" codex \"$@\"")
	assertFileNotContains(t, launcherPath, "mcp_servers.figma.url")

	report := mustDoctor(t, layout)
	if report.CodexConfigPresent || report.CodexConfigLinked || report.CodexConfigGenerated || report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "global" {
		t.Fatalf("project MCP opt-in should not create project-local Codex config: %+v", report)
	}
	if len(report.CodexMCPEnabled) != 1 || report.CodexMCPEnabled[0] != "figma" {
		t.Fatalf("CodexMCPEnabled = %#v, want [figma]", report.CodexMCPEnabled)
	}
}

func TestSetupSyncsSafariMCPRegistryDefinition(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	registryPath := filepath.Join(project, ".agents", ".configs", "codex-mcp-servers.toml")
	assertFileContains(t, registryPath, "[servers.safari]")
	assertFileContains(t, registryPath, "command = \"/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver\"")
	assertFileContains(t, registryPath, "args = [\"--mcp\"]")
}

func TestSetupLocalRemovesGeneratedCodexConfigAndLauncherWhenMCPOptInRemoved(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".codex"))
	mustMkdir(t, filepath.Join(project, ".local", "bin"))
	mustWrite(t, filepath.Join(project, ".codex", "config.toml"), generatedCodexConfigMarker+"\n[mcp_servers.figma]\nurl = \"https://mcp.figma.com/mcp\"\n")
	mustWrite(t, filepath.Join(project, ".local", "bin", "codex-local"), "#!/usr/bin/env sh\n"+generatedCodexConfigMarker+"\nexec codex \"$@\"\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".codex", "config.toml"))
	assertNoPath(t, filepath.Join(project, ".local", "bin", "codex-local"))
	report := mustDoctor(t, layout)
	if report.CodexConfigPresent || report.CodexConfigGenerated || report.CodexConfigShadowsGlobal {
		t.Fatalf("generated Codex config should be removed without MCP opt-in: %+v", report)
	}
}

func TestSetupLocalMCPOptInPreservesCustomCodexConfig(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	mustMkdir(t, filepath.Join(project, ".codex"))
	mustWrite(t, filepath.Join(project, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"figma\"]\n")
	mustWrite(t, filepath.Join(project, ".codex", "config.toml"), "model = \"custom\"\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertFileContains(t, filepath.Join(project, ".codex", "config.toml"), "model = \"custom\"")
	assertFileContains(t, filepath.Join(project, ".local", "bin", "codex-local"), "agents-infra\" codex")
	assertFileNotContains(t, filepath.Join(project, ".local", "bin", "codex-local"), "mcp_servers.figma.url")
	data, err := os.ReadFile(filepath.Join(project, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile(custom config): %v", err)
	}
	if strings.Contains(string(data), "[mcp_servers.figma]") {
		t.Fatalf("custom config should not be rewritten with MCP opt-in: %q", string(data))
	}
}

func TestSetupLocalUnknownMCPOptInDefersValidationToLaunchTime(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	mustWrite(t, filepath.Join(project, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"missing\"]\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup should defer unknown MCP validation to launch time: %v", err)
	}
	report := mustDoctor(t, layout)
	if len(report.CodexMCPEnabled) != 1 || report.CodexMCPEnabled[0] != "missing" {
		t.Fatalf("CodexMCPEnabled = %#v, want [missing]", report.CodexMCPEnabled)
	}
}

func TestSetupLocalGlobalCodexConfigModeRemovesCustomProjectCodexConfig(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	mustMkdir(t, filepath.Join(project, ".codex"))
	mustWrite(t, filepath.Join(project, ".codex", "config.toml"), "model = \"gpt-5.4\"\n")

	if err := Setup(Options{Layout: layout, CodexConfigMode: CodexConfigModeGlobal}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".codex", "config.toml"))
	report := mustDoctor(t, layout)
	if report.CodexConfigPresent || report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "global" {
		t.Fatalf("global Codex config mode should leave global config authoritative: %+v", report)
	}
}

func TestSetupLocalLocalCodexConfigModeRendersProjectSafeCodexConfig(t *testing.T) {
	source := seedSourceRepo(t)
	sourceConfigPath := filepath.Join(source, ".configs", "codex-config.toml")
	sourceConfig := `model = "gpt-5.6-terra"
model_reasoning_effort = "xhigh"
service_tier = "fast"
feature_flags = ["alpha", "beta"]
inline_policy = { enabled = true, retries = 3 }
released_at = 2026-07-23T10:30:00Z

[profiles.fast]
model = "gpt-5.6-terra"
model_reasoning_effort = "high"

[projects."/tmp/example"]
trust_level = "trusted"

[notice]
hide_rate_limit_model_nudge = true

[[hooks]]
name = "first"
args = ["one", "two"]

[[hooks]]
name = "second"
args = []
`
	mustWrite(t, sourceConfigPath, sourceConfig)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	mustMkdir(t, filepath.Join(project, ".codex"))
	legacyTarget := filepath.Join(project, ".agents", ".configs", "codex-config.toml")
	if err := os.Symlink(legacyTarget, filepath.Join(project, ".codex", "config.toml")); err != nil {
		t.Fatalf("Symlink(legacy project Codex config): %v", err)
	}

	if err := Setup(Options{Layout: layout, CodexConfigMode: CodexConfigModeLocal}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	projectConfigPath := filepath.Join(project, ".codex", "config.toml")
	info, err := os.Lstat(projectConfigPath)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", projectConfigPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("project-local Codex config should be rendered, got symlink: %s", projectConfigPath)
	}
	assertFileContains(t, projectConfigPath, generatedCodexConfigMarker)
	assertFileNotContains(t, projectConfigPath, "[profiles.fast]")

	installedConfigData, err := os.ReadFile(filepath.Join(project, ".agents", ".configs", "codex-config.toml"))
	if err != nil {
		t.Fatalf("ReadFile(installed Codex config): %v", err)
	}
	projectConfigData, err := os.ReadFile(projectConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(project Codex config): %v", err)
	}
	var wantDocument map[string]any
	if err := toml.Unmarshal(installedConfigData, &wantDocument); err != nil {
		t.Fatalf("Unmarshal(installed Codex config): %v", err)
	}
	delete(wantDocument, "profiles")
	var gotDocument map[string]any
	if err := toml.Unmarshal(projectConfigData, &gotDocument); err != nil {
		t.Fatalf("Unmarshal(project Codex config): %v", err)
	}
	if !reflect.DeepEqual(gotDocument, wantDocument) {
		t.Fatalf("rendered project Codex config changed valid settings:\ngot:  %#v\nwant: %#v", gotDocument, wantDocument)
	}

	report := mustDoctor(t, layout)
	if !report.CodexConfigPresent || report.CodexConfigLinked || !report.CodexConfigGenerated || !report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "project-local" {
		t.Fatalf("local Codex config mode should install project-local config: %+v", report)
	}
}

func TestSetupLocalLocalCodexConfigModeRejectsMalformedSourceWithoutClobberingGeneratedConfig(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout, CodexConfigMode: CodexConfigModeLocal}); err != nil {
		t.Fatalf("initial Setup: %v", err)
	}
	projectConfigPath := filepath.Join(project, ".codex", "config.toml")
	before, err := os.ReadFile(projectConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(initial project Codex config): %v", err)
	}

	installedConfigPath := filepath.Join(project, ".agents", ".configs", "codex-config.toml")
	mustWrite(t, installedConfigPath, "model = \"broken\"\n[profiles.fast\n")
	err = RefreshLinks(Options{Layout: layout, CodexConfigMode: CodexConfigModeLocal})
	if err == nil {
		t.Fatal("RefreshLinks succeeded with malformed installed Codex config")
	}
	if !strings.Contains(err.Error(), "parse installed Codex config") || !strings.Contains(err.Error(), installedConfigPath) {
		t.Fatalf("unexpected malformed Codex config error: %v", err)
	}

	after, readErr := os.ReadFile(projectConfigPath)
	if readErr != nil {
		t.Fatalf("ReadFile(project Codex config after failed refresh): %v", readErr)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("failed refresh changed project Codex config:\nbefore: %q\nafter:  %q", string(before), string(after))
	}
	temporaryFiles, globErr := filepath.Glob(filepath.Join(project, ".codex", ".config.toml.tmp-*"))
	if globErr != nil {
		t.Fatalf("Glob(temporary Codex configs): %v", globErr)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("failed refresh left temporary Codex configs: %#v", temporaryFiles)
	}
}

func TestSetupRejectsUnknownCodexConfigMode(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{Layout: layout, CodexConfigMode: CodexConfigMode("bogus")})
	if err == nil {
		t.Fatal("expected unknown Codex config mode to fail")
	}
	if !strings.Contains(err.Error(), "unknown Codex config mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetupGlobalLinksCodexConfig(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertSymlink(t, filepath.Join(home, ".codex", "config.toml"), filepath.Join(home, ".agents", ".configs", "codex-config.toml"))
	assertFileContains(t, filepath.Join(home, ".codex", "config.toml"), "hide_rate_limit_model_nudge = true")
	assertFileContains(t, filepath.Join(home, ".codex", "config.toml"), "[profiles.fast]")
	report := mustDoctor(t, layout)
	if !report.CodexConfigPresent || !report.CodexConfigLinked || report.CodexConfigGenerated || report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "global" {
		t.Fatalf("unexpected global Codex config doctor report: %+v", report)
	}
	assertFileNotContains(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.figma]")
}

func TestSetupPreservesExistingPublicSkillsRegistryEntries(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	mustMkdir(t, filepath.Join(project, ".agents", "skills", "public-skill"))
	mustWrite(t, filepath.Join(project, ".agents", "skills", "public-skill", "SKILL.md"), "public")

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertExists(t, filepath.Join(project, ".agents", "skills", "public-skill", "SKILL.md"))
	assertSymlink(t, filepath.Join(project, ".agents", "skills", "pdf"), filepath.Join(project, ".agents", ".skills", "pdf"))
}

func TestSetupScrubsStaleNestedGitMetadataFromInstalledRuntime(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	mustMkdir(t, filepath.Join(project, ".agents", ".skills", "pdf", ".git"))
	mustWrite(t, filepath.Join(project, ".agents", ".skills", "pdf", ".git", "config"), "stale")
	mustWrite(t, filepath.Join(project, ".agents", ".skills", "pdf", ".gitignore"), "stale-ignore")
	mustMkdir(t, filepath.Join(project, ".agents", ".skills", "pdf", "vendor", ".git"))
	mustWrite(t, filepath.Join(project, ".agents", ".skills", "pdf", "vendor", ".git", "HEAD"), "stale-head")

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".agents", ".skills", "pdf", ".git"))
	assertNoPath(t, filepath.Join(project, ".agents", ".skills", "pdf", ".gitignore"))
	assertNoPath(t, filepath.Join(project, ".agents", ".skills", "pdf", "vendor", ".git"))
}

func TestSetupLocalPreservesProjectAgentsSourceBeforeRendering(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".instructions"))
	mustWrite(t, filepath.Join(project, ".agents", ".instructions", "PROJECT.md"), "project instructions\n")
	mustWrite(t, filepath.Join(project, "AGENTS.md"), "# Project\n\n@./.agents/.instructions/PROJECT.md\n\nlocal body\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertExists(t, filepath.Join(project, ".agents", ".instructions", "AGENTS.project.md"))
	assertRenderedInstructions(t, filepath.Join(project, "AGENTS.md"))
	assertFileContains(t, filepath.Join(project, "AGENTS.md"), "project instructions")
	assertFileContains(t, filepath.Join(project, "AGENTS.md"), "local body")
}

// seedGitCheckout writes git metadata a real checkout carries, rather than an
// empty .git directory. The difference is not cosmetic: `go build` stamps VCS
// information, so a directory git refuses to read fails the build the generated
// launcher runs — and a fixture that models a broken checkout would be testing
// that, not what it claims to test.
func seedGitCheckout(t *testing.T, root string) {
	t.Helper()
	gitDir := filepath.Join(root, ".git")
	mustMkdir(t, filepath.Join(gitDir, "objects"))
	mustMkdir(t, filepath.Join(gitDir, "refs", "heads"))
	mustWrite(t, filepath.Join(gitDir, "HEAD"), "ref: refs/heads/main\n")
	mustWrite(t, filepath.Join(gitDir, "config"), "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n")
}

func seedSourceRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".instructions"))
	mustMkdir(t, filepath.Join(root, ".configs"))
	mustMkdir(t, filepath.Join(root, ".rules"))
	mustMkdir(t, filepath.Join(root, ".scripts"))
	mustMkdir(t, filepath.Join(root, ".skills", "skill-creator"))
	mustMkdir(t, filepath.Join(root, ".skills", "pdf"))
	seedLauncherBackend(t, root)
	mustMkdir(t, filepath.Join(root, ".temp"))
	mustMkdir(t, filepath.Join(root, ".task-board"))
	seedGitCheckout(t, root)

	mustWrite(t, filepath.Join(root, ".instructions", "INSTRUCTIONS.md"), "# Global Instructions\n\n@~/.agents/.instructions/INSTRUCTIONS_PLATFORM.md\n@~/.agents/.instructions/INSTRUCTIONS_ATTACHMENTS.md\n@~/.agents/.instructions/INSTRUCTIONS_WORKFLOW.md\n")
	mustWrite(t, filepath.Join(root, ".instructions", "AGENTS.md"), "# Global Instructions\n\n@~/.agents/.instructions/INSTRUCTIONS_PLATFORM.md\n@~/.agents/.instructions/INSTRUCTIONS_ATTACHMENTS.md\n@~/.agents/.instructions/INSTRUCTIONS_WORKFLOW.md\n")
	mustWrite(t, filepath.Join(root, ".instructions", "INSTRUCTIONS_PLATFORM.md"), "platform instructions\n")
	mustWrite(t, filepath.Join(root, ".instructions", "INSTRUCTIONS_ATTACHMENTS.md"), imageIntakeWorkflowFixture+"\n")
	mustWrite(t, filepath.Join(root, ".instructions", "INSTRUCTIONS_WORKFLOW.md"), modelAvailabilityPolicyFixture+"\n"+forcedFitPolicyFixture+"\n")
	mustWrite(t, filepath.Join(root, ".configs", "claude-settings.json"), "{}")
	mustWrite(t, filepath.Join(root, ".configs", "codex-config.toml"), "model = \"gpt-5.5\"\n\n[profiles.fast]\nmodel = \"gpt-5.5\"\n\n[notice]\nhide_rate_limit_model_nudge = true\n")
	mustWrite(t, filepath.Join(root, ".configs", "codex-mcp-servers.toml"), `[servers.figma]
url = "https://mcp.figma.com/mcp"

[servers.lldb]
command = "lldb-mcp"

[servers.safari]
command = "/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver"
args = ["--mcp"]
`)
	mustWrite(t, filepath.Join(root, ".rules", "default.rules"), "allow")
	mustWrite(t, filepath.Join(root, ".skills", "skill-creator", "SKILL.md"), "creator")
	mustWrite(t, filepath.Join(root, ".skills", "pdf", "SKILL.md"), "pdf")
	mustWrite(t, filepath.Join(root, ".gitignore"), "ignored")
	mustWrite(t, filepath.Join(root, ".temp", "junk.txt"), "junk")
	mustWrite(t, filepath.Join(root, ".task-board", "README.md"), "board")
	mustWrite(t, filepath.Join(root, "task-board.config.json"), "{}")
	return root
}

func mustDoctor(t *testing.T, layout Layout) Report {
	t.Helper()
	report, err := Doctor(layout)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	return report
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNoPath(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil {
		t.Fatalf("expected %s to be absent", path)
	}
}

func assertSymlink(t *testing.T, path, target string) {
	t.Helper()
	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("Readlink(%s): %v", path, err)
	}
	if got != target {
		t.Fatalf("%s -> %s, want %s", path, got, target)
	}
}

func assertRegularFile(t *testing.T, path string) {
	t.Helper()
	st, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("expected regular file %s to exist: %v", path, err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected %s to be a regular file, got symlink", path)
	}
	if st.IsDir() {
		t.Fatalf("expected %s to be a regular file, got directory", path)
	}
}

func assertRenderedInstructions(t *testing.T, path string) {
	t.Helper()
	st, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("expected rendered instructions %s to exist: %v", path, err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected rendered instructions %s to be a regular file, got symlink", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	body := string(data)
	if !strings.Contains(body, generatedInstructionsMarker) {
		t.Fatalf("rendered instructions missing generated marker: %q", body)
	}
	if strings.Contains(body, "@~/.agents/") {
		t.Fatalf("rendered instructions contain unresolved home include: %q", body)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q: %q", path, want, string(data))
	}
}

func assertFileNotContains(t *testing.T, path, unwanted string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if strings.Contains(string(data), unwanted) {
		t.Fatalf("%s contains unwanted %q: %q", path, unwanted, string(data))
	}
}

func assertNoGeneratedArtifacts(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if base == ".DS_Store" || strings.Contains(base, ".bak.") {
			t.Fatalf("unexpected generated artifact left behind: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s): %v", root, err)
	}
}
