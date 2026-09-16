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
	modelAvailabilityPolicyFixture     = "retry the preferred model before choosing an autonomous fallback"
	forcedFitPolicyFixture             = "do not fake an impossible platform model with flags, stubs, or mocks"
	imageIntakeWorkflowFixture         = "agents-attachments stage-images"
	dirtyCheckoutPolicyFixture         = "validate in a task-scoped worktree before integrating a reviewed patch"
	externalCILocalMirrorPolicySection = `### External-CI local mirror fallback

* Use a local mirror only when hosted CI cannot execute repository steps for a verified external cause that the agent cannot repair. Establish the cause from hosted provider evidence; an absent, failed, partial, malformed, or inconclusive status read is not proof of an external failure and does not authorize the fallback.
* Reproduce every affected hosted job from an exact, clean checkout of the PR-head commit. Run the matching commands with the same toolchain versions, environment variables and non-secret configuration, dependent services, and target platform, architecture, device, or runtime. When an exact match is objectively unavailable, document the difference and why the chosen substitute is equivalent for the behavior under test; otherwise report the job as unverified.
* Preserve auditable evidence for each mirrored job: PR-head SHA, hosted job name, external failure cause, local platform and target, tool versions, environment assumptions, services, exact commands, and every exit code. Redact secrets without omitting the fact that the corresponding configuration was present.
* Repository-caused code, test, build, configuration, or integration failures remain blocking whether they appear in hosted CI or the local mirror. A passing mirror never overrides a repository failure and never converts an unknown hosted result into success.
* Never forge, synthesize, edit, or otherwise misrepresent a hosted check or commit status. Local evidence supplements the unavailable hosted execution; it does not satisfy a required hosted status, authorize self-review, or bypass remote review, merge queues, or branch protection. The hosting platform's real review record and protection rules remain authoritative, and a required hosted check keeps the merge blocked until it reports legitimately or an authorized repository owner changes the rule.`
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
	// The deprecation guard must run before any build step, or a deprecated
	// `agents-infra codex` invocation would write the build output before the
	// Go dispatcher could refuse it.
	for _, entrypoint := range []string{"codex", "claude", "openai-infra", "anthropic-infra", "openai-dange", "anthropic-dange"} {
		message, ok := DeprecatedProviderMessage(entrypoint)
		if !ok {
			t.Fatalf("DeprecatedProviderMessage(%q) missing", entrypoint)
		}
		if !strings.Contains(body, escapeCmdEcho(message)) {
			t.Fatalf("windows wrapper body missing deprecation message for %s: %q", entrypoint, body)
		}
	}
	for _, want := range []string{`if "%~1"=="codex"`, `if "%~1"=="claude"`, `if "%~1"=="target"`, `if "%~1"=="target-yolo"`, `if "%~2"=="openai-infra"`, `if "%~2"=="anthropic-infra"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("windows wrapper body missing deprecation guard %q: %q", want, body)
		}
	}
	if strings.Contains(body, "qwen-infra") {
		t.Fatalf("windows wrapper body must not intercept the live qwen-infra entrypoint: %q", body)
	}
	guard := strings.Index(body, `if "%~1"=="codex"`)
	mkdir := strings.Index(body, "mkdir")
	build := strings.Index(body, "go build")
	if guard < 0 || mkdir < 0 || build < 0 || guard > mkdir || guard > build {
		t.Fatalf("windows deprecation guard must precede mkdir/go build (guard=%d mkdir=%d build=%d): %q", guard, mkdir, build, body)
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
	// The deprecation guard must run before mkdir/go build, or a deprecated
	// subcommand would write the build output before refusing.
	for _, entrypoint := range []string{"codex", "claude", "openai-infra", "anthropic-infra", "openai-dange", "anthropic-dange"} {
		message, ok := DeprecatedProviderMessage(entrypoint)
		if !ok {
			t.Fatalf("DeprecatedProviderMessage(%q) missing", entrypoint)
		}
		if !strings.Contains(body, posixShellQuote(message)) {
			t.Fatalf("unix wrapper body missing deprecation message for %s: %q", entrypoint, body)
		}
	}
	for _, want := range []string{`case "${1:-}" in`, "codex)", "claude)", "target)", "target-yolo)", "openai-infra)", "anthropic-infra)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("unix wrapper body missing deprecation guard %q: %q", want, body)
		}
	}
	if strings.Contains(body, "qwen-infra") {
		t.Fatalf("unix wrapper body must not intercept the live qwen-infra entrypoint: %q", body)
	}
	guard := strings.Index(body, `case "${1:-}" in`)
	mkdir := strings.Index(body, "mkdir -p")
	build := strings.Index(body, "go build")
	if guard < 0 || mkdir < 0 || build < 0 || guard > mkdir || guard > build {
		t.Fatalf("unix deprecation guard must precede mkdir/go build (guard=%d mkdir=%d build=%d): %q", guard, mkdir, build, body)
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

func TestPiInfraWrapperBodyForWindowsUsesExactSiblingTarget(t *testing.T) {
	body := piInfraWrapperBody("windows", "agents-infra.exe")
	for _, want := range []string{
		`if not exist "%DIR%agents-infra.exe"`,
		`"%DIR%agents-infra.exe" pi %*`,
		"exit /b %ERRORLEVEL%",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("windows pi-infra wrapper missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "agents-infra pi %*") {
		t.Fatalf("windows pi-infra wrapper must not fall back through PATH:\n%s", body)
	}
}

func TestDeprecatedCanonicalWrappersRefuseWithoutDelegation(t *testing.T) {
	for _, entrypoint := range []string{"openai-infra", "anthropic-infra"} {
		message, ok := DeprecatedProviderMessage(entrypoint)
		if !ok {
			t.Fatalf("DeprecatedProviderMessage(%q) missing", entrypoint)
		}
		for _, goos := range []string{"darwin", "windows"} {
			t.Run(entrypoint+"/"+goos, func(t *testing.T) {
				body := canonicalTargetWrapperBody(entrypoint, goos, "agents-infra")
				if goos == "windows" {
					if !strings.Contains(body, escapeCmdEcho(message)) || !strings.Contains(body, "1>&2") || !strings.Contains(body, "exit /b 1") {
						t.Fatalf("windows %s wrapper must print the exact message to stderr and exit 1:\n%s", entrypoint, body)
					}
				} else {
					if !strings.Contains(body, posixShellQuote(message)) || !strings.Contains(body, ">&2") || !strings.Contains(body, "exit 1") {
						t.Fatalf("unix %s wrapper must print the exact message to stderr and exit 1:\n%s", entrypoint, body)
					}
				}
				for _, unwanted := range []string{"TARGET", "exec ", "go build", "mkdir", "target " + entrypoint, "127"} {
					if strings.Contains(body, unwanted) {
						t.Fatalf("deprecated %s wrapper must not delegate, build, or check a sibling (found %q):\n%s", entrypoint, unwanted, body)
					}
				}
			})
		}
	}
}

func TestLiveCanonicalWrapperStillDelegates(t *testing.T) {
	posix := canonicalTargetWrapperBody("qwen-infra", "darwin", "agents-infra")
	if !strings.Contains(posix, `exec "$TARGET" target qwen-infra "$@"`) {
		t.Fatalf("unix qwen-infra wrapper must still delegate:\n%s", posix)
	}
	if strings.Contains(posix, "deprecated") {
		t.Fatalf("unix qwen-infra wrapper must not carry a deprecation notice:\n%s", posix)
	}
	windows := canonicalTargetWrapperBody("qwen-infra", "windows", "agents-infra.cmd")
	if !strings.Contains(windows, `"target qwen-infra`) && !strings.Contains(windows, `target qwen-infra`) {
		t.Fatalf("windows qwen-infra wrapper must still delegate:\n%s", windows)
	}
	if strings.Contains(windows, "deprecated") {
		t.Fatalf("windows qwen-infra wrapper must not carry a deprecation notice:\n%s", windows)
	}
}

func TestDeprecatedDirectProviderYoloWrappersRefuseWithoutDelegation(t *testing.T) {
	for _, launcher := range directProviderYoloLaunchers {
		message, ok := DeprecatedProviderMessage(launcher.name)
		if !ok {
			t.Fatalf("DeprecatedProviderMessage(%q) missing", launcher.name)
		}
		for _, goos := range []string{"darwin", "windows"} {
			t.Run(launcher.name+"/"+goos, func(t *testing.T) {
				body := directProviderYoloWrapperBody(launcher, goos, "agents-infra")
				if goos == "windows" {
					if !strings.Contains(body, escapeCmdEcho(message)) || !strings.Contains(body, "exit /b 1") {
						t.Fatalf("windows %s wrapper must print the exact message and exit 1:\n%s", launcher.name, body)
					}
				} else {
					if !strings.Contains(body, posixShellQuote(message)) || !strings.Contains(body, "exit 1") {
						t.Fatalf("unix %s wrapper must print the exact message and exit 1:\n%s", launcher.name, body)
					}
				}
				for _, unwanted := range []string{"TARGET", "exec ", "go build", "target-yolo", "127"} {
					if strings.Contains(body, unwanted) {
						t.Fatalf("deprecated %s wrapper must not delegate or build (found %q):\n%s", launcher.name, unwanted, body)
					}
				}
			})
		}
	}
}

func TestCodexLocalLauncherRefusesWithoutDelegation(t *testing.T) {
	message, ok := DeprecatedProviderMessage("codex")
	if !ok {
		t.Fatal("DeprecatedProviderMessage(codex) missing")
	}
	body := codexLocalLauncherBody()
	if !strings.Contains(body, generatedCodexConfigMarker) {
		t.Fatalf("codex-local shim must keep its generated marker for the opt-out lifecycle:\n%s", body)
	}
	if !strings.Contains(body, posixShellQuote(message)) || !strings.Contains(body, "exit 1") {
		t.Fatalf("codex-local shim must print the CLI codex message and exit 1:\n%s", body)
	}
	for _, unwanted := range []string{"exec ", "agents-infra\" codex", "DIR=", "go build"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("codex-local shim must not delegate or build (found %q):\n%s", unwanted, body)
		}
	}
}

func TestPiInfraUnixWrapperPreservesCallerCWDAndEveryArgument(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell launcher test")
	}
	dir := t.TempDir()
	launcher := filepath.Join(dir, "pi-infra")
	target := filepath.Join(dir, "agents-infra")
	mustWrite(t, launcher, piInfraWrapperBody(runtime.GOOS, "agents-infra"))
	mustWrite(t, target, "#!/usr/bin/env sh\nprintf 'cwd=<%s>\\n' \"$PWD\"\nfor arg in \"$@\"; do printf 'arg=<%s>\\n' \"$arg\"; done\n")
	for _, path := range []string{launcher, target} {
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatalf("Chmod(%s): %v", path, err)
		}
	}
	caller := filepath.Join(t.TempDir(), "caller with spaces")
	mustMkdir(t, caller)
	args := []string{"--profile", "qwen", "--", "ordinary prompt", "--post-separator", "@literal"}
	command := exec.Command(launcher, args...)
	command.Dir = caller
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("pi-infra: %v\n%s", err, output)
	}
	wantLines := []string{"cwd=<" + caller + ">", "arg=<pi>"}
	for _, arg := range args {
		wantLines = append(wantLines, "arg=<"+arg+">")
	}
	if got, want := strings.TrimSpace(string(output)), strings.Join(wantLines, "\n"); got != want {
		t.Fatalf("delegation output:\n%s\nwant:\n%s", got, want)
	}
}

func TestPiInfraUnixWrapperRefusesMissingSiblingTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell launcher test")
	}
	dir := t.TempDir()
	launcher := filepath.Join(dir, "pi-infra")
	mustWrite(t, launcher, piInfraWrapperBody(runtime.GOOS, "agents-infra"))
	if err := os.Chmod(launcher, 0o755); err != nil {
		t.Fatalf("Chmod(%s): %v", launcher, err)
	}
	output, err := exec.Command(launcher, "--", "prompt").CombinedOutput()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 127 {
		t.Fatalf("missing target error = %v, output:\n%s", err, output)
	}
	if !strings.Contains(string(output), "missing managed target") || !strings.Contains(string(output), filepath.Join(dir, "agents-infra")) {
		t.Fatalf("missing-target refusal lacks exact target:\n%s", output)
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

	// Retired distribution surfaces stay absent even though the source carries
	// them: instructions, skills, and the bundled MCP registry.
	assertNoPath(t, filepath.Join(project, ".agents", ".instructions"))
	assertNoPath(t, filepath.Join(project, ".agents", ".skills"))
	assertNoPath(t, filepath.Join(project, ".agents", "skills"))
	assertNoPath(t, filepath.Join(project, ".agents", ".configs", "codex-mcp-servers.toml"))
	assertNoPath(t, filepath.Join(project, ".claude", "instructions"))
	assertNoPath(t, filepath.Join(project, ".claude", "CLAUDE.md"))
	assertNoPath(t, filepath.Join(project, ".codex", "AGENTS.md"))
	assertNoPath(t, filepath.Join(project, "AGENTS.md"))
	assertNoPath(t, filepath.Join(project, ".agents", ".git"))
	// Residual surfaces install.
	assertSymlink(t, filepath.Join(project, ".claude", "settings.json"), filepath.Join(project, ".agents", ".configs", "claude-settings.json"))
	assertSymlink(t, filepath.Join(project, ".codex", "rules", "default.rules"), filepath.Join(project, ".agents", ".rules", "default.rules"))
	assertNoPath(t, filepath.Join(project, ".agents", ".scripts", "agents-attachments"))
	assertRegularFile(t, filepath.Join(project, ".local", "bin", "agents-attachments"))
	assertFileContains(t, filepath.Join(project, ".local", "bin", "agents-attachments"), `"$TARGET" attachments "$@"`)
	assertFileNotContains(t, filepath.Join(project, ".local", "bin", "agents-attachments"), "python")
	assertRegularFile(t, filepath.Join(project, ".local", "bin", piInfraWrapperName(runtime.GOOS)))
	assertRegularFile(t, filepath.Join(project, ".local", "bin", canonicalTargetWrapperName("openai-dange", runtime.GOOS)))
	assertRegularFile(t, filepath.Join(project, ".local", "bin", canonicalTargetWrapperName("anthropic-dange", runtime.GOOS)))
	assertFileContains(t, filepath.Join(project, ".local", "bin", piInfraWrapperName(runtime.GOOS)), "agents-infra")
	assertFileContains(t, filepath.Join(project, ".local", "bin", piInfraWrapperName(runtime.GOOS)), " pi ")

	launcher := filepath.Join(project, ".local", "bin", "agents-infra")
	data, err := os.ReadFile(launcher)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", launcher, err)
	}
	if !strings.Contains(string(data), source) {
		t.Fatalf("launcher does not reference source repo: %q", string(data))
	}
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime: %v", err)
	}
}

func TestSetupLocalLeavesProjectInstructionSpaceUntouched(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	instructionsDir := filepath.Join(project, ".agents", ".instructions")
	mustMkdir(t, instructionsDir)
	mustWrite(t, filepath.Join(instructionsDir, "AGENTS.md"), "# Local Codex Instructions\n\n@PROJECT.md\n")
	mustWrite(t, filepath.Join(instructionsDir, "INSTRUCTIONS.md"), "# Local Claude Instructions\n\n@PROJECT.md\n")
	mustWrite(t, filepath.Join(instructionsDir, "PROJECT.md"), "project-owned instructions\n")
	mustWrite(t, filepath.Join(instructionsDir, "INSTRUCTIONS_WORKFLOW.md"), "project-owned workflow override\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	before := map[string]string{}
	for _, name := range []string{"AGENTS.md", "INSTRUCTIONS.md", "PROJECT.md", "INSTRUCTIONS_WORKFLOW.md"} {
		data, err := os.ReadFile(filepath.Join(instructionsDir, name))
		if err != nil {
			t.Fatal(err)
		}
		before[name] = string(data)
	}

	for i := 0; i < 2; i++ {
		if err := Setup(Options{Layout: layout}); err != nil {
			t.Fatalf("Setup run %d: %v", i+1, err)
		}
	}

	for name, want := range before {
		data, err := os.ReadFile(filepath.Join(instructionsDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("setup rewrote project instruction file %s", name)
		}
	}
	assertNoPath(t, filepath.Join(project, ".codex", "AGENTS.md"))
	assertNoPath(t, filepath.Join(project, "AGENTS.md"))
	assertNoPath(t, filepath.Join(project, ".claude", "CLAUDE.md"))
	assertNoPath(t, filepath.Join(project, ".claude", "instructions"))
}

func TestSetupAndRefreshLinksLeaveSkillSurfacesUntouched(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	// Pre-seed user- and Curator-owned skill surfaces, including the stale
	// self-link shape setup used to garbage-collect. Setup owns none of them
	// now and must leave every byte and link target identical.
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
	managedLooking := filepath.Join(project, ".agents", ".skills", "pdf", "SKILL.md")
	mustMkdir(t, filepath.Dir(managedLooking))
	mustWrite(t, managedLooking, "user-owned skill content\n")

	snapshot := map[string]string{}
	for _, link := range []string{staleLink, staleClaudeLink, staleCodexLink} {
		target, err := os.Readlink(link)
		if err != nil {
			t.Fatal(err)
		}
		snapshot[link] = target
	}
	skillBytes, err := os.ReadFile(managedLooking)
	if err != nil {
		t.Fatal(err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := RefreshLinks(Options{Layout: layout}); err != nil {
		t.Fatalf("RefreshLinks: %v", err)
	}

	for link, want := range snapshot {
		target, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("Readlink(%s): %v", link, err)
		}
		if target != want {
			t.Fatalf("skill link %s changed: got %q, want %q", link, target, want)
		}
	}
	after, err := os.ReadFile(managedLooking)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(skillBytes) {
		t.Fatalf("installed skill content changed")
	}
	assertNoPath(t, filepath.Join(project, ".agents", "skills", repoSkillName))
	assertNoPath(t, filepath.Join(project, ".agents", ".skills", repoSkillName))
	assertNoPath(t, filepath.Join(project, ".claude", "skills", repoSkillName))
	assertNoPath(t, filepath.Join(project, ".codex", "skills", repoSkillName))
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime: %v", err)
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
	// Setup installs the residual runtime only. The instruction and skill
	// link booleans are legacy observations of native-home state setup no
	// longer manages, so they report false on a fresh residual install while
	// helpers and config policy keep reporting healthy.
	if !report.AgentsGitFree || report.ClaudeLinked || report.CodexLinked || report.CodexRendered || report.CodexProjectRendered || report.CodexConfigPresent || report.CodexConfigLinked || report.CodexConfigGenerated || report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "global" || len(report.CodexMCPEnabled) != 0 || !report.CodexPrimaryConfigValid || report.CodexPrimarySession.Model.Present || report.CodexPrimarySession.ReasoningEffort.Present || report.CodexPrimarySession.YoloMode.Present || !report.HelpersLinked || report.InfraSkillLink {
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
	seedGlobalAgentsInfraTarget(t, layout)

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertRegularFile(t, filepath.Join(home, ".local", "bin", "agents-infra"))
	assertNoPath(t, filepath.Join(home, ".local", "bin", "agents-infra.cmd"))
	assertRegularFile(t, filepath.Join(home, ".local", "bin", "pi-infra"))
	assertRegularFile(t, filepath.Join(home, ".local", "bin", "openai-infra"))
	assertRegularFile(t, filepath.Join(home, ".local", "bin", "anthropic-infra"))
	assertRegularFile(t, filepath.Join(home, ".local", "bin", "qwen-infra"))
	assertRegularFile(t, filepath.Join(home, ".local", "bin", "openai-dange"))
	assertRegularFile(t, filepath.Join(home, ".local", "bin", "anthropic-dange"))
	assertNoPath(t, filepath.Join(home, ".agents", ".instructions"))
	assertNoPath(t, filepath.Join(home, ".agents", ".skills"))
	assertNoPath(t, filepath.Join(home, ".codex", "AGENTS.md"))
	assertSymlink(t, filepath.Join(home, ".claude", "settings.json"), filepath.Join(home, ".agents", ".configs", "claude-settings.json"))
	assertSymlink(t, filepath.Join(home, ".codex", "rules", "default.rules"), filepath.Join(home, ".agents", ".rules", "default.rules"))
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime: %v", err)
	}
}

func TestSetupGlobalDistributesNoInstructionsSkillsOrRegistry(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}
	seedGlobalAgentsInfraTarget(t, layout)

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	for _, distributed := range []string{
		filepath.Join(home, ".agents", ".instructions"),
		filepath.Join(home, ".agents", ".skills"),
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, ".agents", ".configs", "codex-mcp-servers.toml"),
		filepath.Join(home, ".claude", "CLAUDE.md"),
		filepath.Join(home, ".claude", "instructions"),
		filepath.Join(home, ".codex", "AGENTS.md"),
	} {
		assertNoPath(t, distributed)
	}
	assertSymlink(t, filepath.Join(home, ".claude", "settings.json"), filepath.Join(home, ".agents", ".configs", "claude-settings.json"))
	assertSymlink(t, filepath.Join(home, ".codex", "config.toml"), filepath.Join(home, ".agents", ".configs", "codex-config.toml"))
	assertSymlink(t, filepath.Join(home, ".codex", "rules", "default.rules"), filepath.Join(home, ".agents", ".rules", "default.rules"))
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime: %v", err)
	}
}

func TestSetupGlobalRemovesStaleProjectConfig(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}
	seedGlobalAgentsInfraTarget(t, layout)
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
	mustWrite(t, codexConfig, `model = "user-overrides-managed-model"
service_tier = "fast"

[profiles.fast]
model = "gpt-5.5"

[profiles.custom]
model = "custom-model"

[projects."/user/trusted"]
trust_level = "trusted"

[notice]
hide_full_access_warning = true
`)
	mustWrite(t, claudeSettings, "{\n  \"model\": \"claude-sonnet-5\",\n  \"permissions\": {\"defaultMode\": \"bypassPermissions\"}\n}\n")
	projectConfig := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	projectConfigState := "[agents.codex.primary_session]\nmodel = \"primary-model\"\nyolo_mode = false\n"
	mustWrite(t, projectConfig, projectConfigState)

	mustWrite(t, filepath.Join(source, ".configs", "codex-config.toml"), `model = "source-default-overwrite"
service_tier = "default"

[projects."/source/trusted"]
trust_level = "trusted"

[notice]
hide_rate_limit_model_nudge = true
`)
	mustWrite(t, filepath.Join(source, ".configs", "claude-settings.json"), "{\"model\":\"source-default-overwrite\"}\n")
	mustWrite(t, filepath.Join(source, ".configs", "codex-mcp-servers.toml"), `[servers.updated]
url = "https://example.test/mcp"
`)
	callerRegistry := filepath.Join(project, ".agents", ".configs", "codex-mcp-servers.toml")
	mustWrite(t, callerRegistry, "[servers.caller]\nurl = \"https://caller.example/mcp\"\n")

	if err := Setup(Options{Layout: layout, CodexConfigMode: CodexConfigModeLocal}); err != nil {
		t.Fatalf("second Setup: %v", err)
	}

	assertFileContains(t, codexConfig, "source-default-overwrite")
	assertFileContains(t, codexConfig, "service_tier = 'default'")
	assertFileNotContains(t, codexConfig, "user-overrides-managed-model")
	assertFileNotContains(t, codexConfig, "[profiles.fast]")
	assertFileContains(t, codexConfig, "[profiles.custom]")
	assertFileContains(t, codexConfig, "[projects.'/user/trusted']")
	assertFileContains(t, codexConfig, "[projects.'/source/trusted']")
	assertFileContains(t, codexConfig, "hide_full_access_warning = true")
	assertFileContains(t, codexConfig, "hide_rate_limit_model_nudge = true")
	assertFileContains(t, claudeSettings, "claude-sonnet-5")
	assertFileContains(t, claudeSettings, "bypassPermissions")
	assertFileNotContains(t, claudeSettings, "source-default-overwrite")
	assertFileContains(t, callerRegistry, "[servers.caller]")
	assertFileNotContains(t, callerRegistry, "[servers.updated]")
	assertFileContains(t, filepath.Join(project, ".codex", "config.toml"), "source-default-overwrite")
	assertFileNotContains(t, filepath.Join(project, ".codex", "config.toml"), "[profiles.custom]")
	assertSymlink(t, filepath.Join(project, ".claude", "settings.json"), claudeSettings)
	report := mustDoctor(t, layout)
	if !report.CodexConfigPresent || !report.CodexConfigGenerated || !report.CodexPrimarySession.Model.Present || report.CodexPrimarySession.Model.Value != "primary-model" || !report.CodexPrimarySession.YoloMode.Present || report.CodexPrimarySession.YoloMode.Value {
		t.Fatalf("unexpected local doctor report after managed config migration: %+v", report)
	}
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime: %v", err)
	}
	if got, err := os.ReadFile(projectConfig); err != nil {
		t.Fatalf("ReadFile(%s): %v", projectConfig, err)
	} else if string(got) != projectConfigState {
		t.Fatalf("primary-session policy changed during resync:\ngot:  %q\nwant: %q", string(got), projectConfigState)
	}
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
	codexMessage, ok := DeprecatedProviderMessage("codex")
	if !ok {
		t.Fatal("DeprecatedProviderMessage(codex) missing")
	}
	assertFileContains(t, launcherPath, generatedCodexConfigMarker)
	assertFileContains(t, launcherPath, posixShellQuote(codexMessage))
	assertFileContains(t, launcherPath, "exit 1")
	assertFileNotContains(t, launcherPath, "exec \"$DIR/agents-infra\" codex")
	assertFileNotContains(t, launcherPath, "mcp_servers.figma.url")

	report := mustDoctor(t, layout)
	if report.CodexConfigPresent || report.CodexConfigLinked || report.CodexConfigGenerated || report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "global" {
		t.Fatalf("project MCP opt-in should not create project-local Codex config: %+v", report)
	}
	if len(report.CodexMCPEnabled) != 1 || report.CodexMCPEnabled[0] != "figma" {
		t.Fatalf("CodexMCPEnabled = %#v, want [figma]", report.CodexMCPEnabled)
	}
}

func TestSetupDoesNotSyncBundledMCPRegistryDefinition(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	callerRegistry := filepath.Join(project, ".agents", ".configs", "codex-mcp-servers.toml")
	mustMkdir(t, filepath.Dir(callerRegistry))
	mustWrite(t, callerRegistry, "[servers.caller]\nurl = \"https://caller.example/mcp\"\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// The source carries a safari definition; it must not reach the install.
	data, err := os.ReadFile(callerRegistry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[servers.caller]") || strings.Contains(string(data), "[servers.safari]") {
		t.Fatalf("caller registry not preserved byte-identical: %q", string(data))
	}
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
	codexRefusal, ok := DeprecatedProviderMessage("codex")
	if !ok {
		t.Fatal("DeprecatedProviderMessage(codex) missing")
	}
	assertFileContains(t, filepath.Join(project, ".local", "bin", "codex-local"), posixShellQuote(codexRefusal))
	assertFileNotContains(t, filepath.Join(project, ".local", "bin", "codex-local"), "agents-infra\" codex")
	assertFileNotContains(t, filepath.Join(project, ".local", "bin", "codex-local"), "mcp_servers.figma.url")
	data, err := os.ReadFile(filepath.Join(project, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile(custom config): %v", err)
	}
	if strings.Contains(string(data), "[mcp_servers.figma]") {
		t.Fatalf("custom config should not be rewritten with MCP opt-in: %q", string(data))
	}
}

func TestSetupLocalUnknownMCPOptInDefersValidationToComposeTime(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	mustWrite(t, filepath.Join(project, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"missing\"]\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup should defer unknown MCP validation to compose time: %v", err)
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
	seedGlobalAgentsInfraTarget(t, layout)

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertSymlink(t, filepath.Join(home, ".codex", "config.toml"), filepath.Join(home, ".agents", ".configs", "codex-config.toml"))
	assertFileContains(t, filepath.Join(home, ".codex", "config.toml"), "hide_rate_limit_model_nudge = true")
	assertFileContains(t, filepath.Join(home, ".codex", "config.toml"), "service_tier = \"default\"")
	assertFileNotContains(t, filepath.Join(home, ".codex", "config.toml"), "[profiles.fast]")
	report := mustDoctor(t, layout)
	if !report.CodexConfigPresent || !report.CodexConfigLinked || report.CodexConfigGenerated || report.CodexConfigShadowsGlobal || report.CodexConfigEffective != "global" {
		t.Fatalf("unexpected global Codex config doctor report: %+v", report)
	}
	assertFileNotContains(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.figma]")
}

// Production call site: Setup -> setupCodex -> syncManagedCodexConfig.
// This binds the repository-managed Codex config to the installed native config
// that an openai-board parent session reads when no explicit project pin wins.
func TestSetupGlobalPreservesRepositorySolFallbackWithReasoningEffort(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repositoryConfigPath := filepath.Join(workingDir, "..", "..", "..", "..", ".configs", "codex-config.toml")
	repositoryConfig, err := os.ReadFile(repositoryConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", repositoryConfigPath, err)
	}

	source := seedSourceRepo(t)
	mustWrite(t, filepath.Join(source, ".configs", "codex-config.toml"), string(repositoryConfig))
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}
	seedGlobalAgentsInfraTarget(t, layout)

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	installedPath := filepath.Join(home, ".codex", "config.toml")
	installedConfig, err := os.ReadFile(installedPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", installedPath, err)
	}
	var document struct {
		Model           string `toml:"model"`
		ReasoningEffort string `toml:"model_reasoning_effort"`
	}
	if err := toml.Unmarshal(installedConfig, &document); err != nil {
		t.Fatalf("Unmarshal(%s): %v", installedPath, err)
	}
	if document.Model != "gpt-5.6-sol" || document.ReasoningEffort != "xhigh" {
		t.Fatalf("installed Codex pin = %q/%q, want gpt-5.6-sol/xhigh", document.Model, document.ReasoningEffort)
	}
}

func TestSetupGlobalMigratesManagedCodexConfigPreservingUserState(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}
	seedGlobalAgentsInfraTarget(t, layout)
	existingConfigPath := filepath.Join(home, ".agents", managedCodexConfigRelativePath)
	mustMkdir(t, filepath.Dir(existingConfigPath))
	mustWrite(t, existingConfigPath, `model = "old-managed-model"
service_tier = "fast"

[profiles.fast]
model = "gpt-5.5"

[profiles.custom]
model = "custom-model"

[projects."/user/trusted"]
trust_level = "trusted"

[notice]
hide_rate_limit_model_nudge = false
hide_full_access_warning = true
`)

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	configPath := filepath.Join(home, ".codex", "config.toml")
	assertSymlink(t, configPath, existingConfigPath)
	assertFileContains(t, configPath, "model = 'gpt-5.6-sol'")
	assertFileContains(t, configPath, "model_context_window = 272000")
	assertFileContains(t, configPath, "model_auto_compact_token_limit = 245000")
	assertFileContains(t, configPath, "service_tier = 'default'")
	assertFileNotContains(t, configPath, "old-managed-model")
	assertFileNotContains(t, configPath, "[profiles.fast]")
	assertFileContains(t, configPath, "[profiles.custom]")
	assertFileContains(t, configPath, "[projects.'/user/trusted']")
	assertFileContains(t, configPath, "hide_rate_limit_model_nudge = false")
	assertFileContains(t, configPath, "hide_full_access_warning = true")
	report := mustDoctor(t, layout)
	if !report.CodexConfigPresent || !report.CodexConfigLinked || report.CodexConfigEffective != "global" {
		t.Fatalf("unexpected global doctor report after managed config migration: %+v", report)
	}
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime: %v", err)
	}
}

func TestSetupGlobalRejectsMalformedExistingCodexConfigWithoutReplacingIt(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}
	seedGlobalAgentsInfraTarget(t, layout)
	existingConfigPath := filepath.Join(home, ".agents", managedCodexConfigRelativePath)
	mustMkdir(t, filepath.Dir(existingConfigPath))
	existingConfig := []byte("model = \"user-model\"\n[projects.\"/user\"\ntrust_level = \"trusted\"\n")
	mustWrite(t, existingConfigPath, string(existingConfig))

	err = Setup(Options{Layout: layout})
	if err == nil {
		t.Fatal("Setup succeeded with malformed existing managed Codex config")
	}
	if !strings.Contains(err.Error(), "parse existing managed Codex config") || !strings.Contains(err.Error(), existingConfigPath) {
		t.Fatalf("unexpected malformed existing config error: %v", err)
	}
	after, readErr := os.ReadFile(existingConfigPath)
	if readErr != nil {
		t.Fatalf("ReadFile(%s): %v", existingConfigPath, readErr)
	}
	if !bytes.Equal(after, existingConfig) {
		t.Fatalf("failed migration replaced malformed existing config:\nbefore: %q\nafter:  %q", existingConfig, after)
	}
}

func TestSetupPreservesExistingSkillsContentWithoutManagingIt(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	mustMkdir(t, filepath.Join(project, ".agents", "skills", "public-skill"))
	mustWrite(t, filepath.Join(project, ".agents", "skills", "public-skill", "SKILL.md"), "public")
	mustMkdir(t, filepath.Join(project, ".agents", ".skills", "pdf"))
	mustWrite(t, filepath.Join(project, ".agents", ".skills", "pdf", "SKILL.md"), "user-owned pdf skill")

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertFileContains(t, filepath.Join(project, ".agents", "skills", "public-skill", "SKILL.md"), "public")
	assertFileContains(t, filepath.Join(project, ".agents", ".skills", "pdf", "SKILL.md"), "user-owned pdf skill")
	assertNoPath(t, filepath.Join(project, ".agents", "skills", repoSkillName))
	assertNoPath(t, filepath.Join(project, ".claude", "skills"))
	assertNoPath(t, filepath.Join(project, ".codex", "skills"))
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime: %v", err)
	}
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

func TestSetupLeavesProjectAgentsSourceUntouchedAndPrepareRendersIt(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".instructions"))
	mustWrite(t, filepath.Join(project, ".agents", ".instructions", "PROJECT.md"), "project instructions\n")
	projectDoc := "# Project\n\n@./.agents/.instructions/PROJECT.md\n\nlocal body\n"
	mustWrite(t, filepath.Join(project, "AGENTS.md"), projectDoc)
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertNoPath(t, filepath.Join(project, ".agents", ".instructions", "AGENTS.project.md"))
	assertNoPath(t, filepath.Join(project, ".codex", "AGENTS.md"))
	data, err := os.ReadFile(filepath.Join(project, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != projectDoc {
		t.Fatalf("setup rewrote the hand-written project AGENTS.md: %q", string(data))
	}

	report, err := PreparePrimarySession("codex", project, ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"})
	if err != nil {
		t.Fatalf("PreparePrimarySession: %v", err)
	}
	if !report.CodexProjectRendered {
		t.Fatalf("report = %#v", report)
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
	mustWrite(t, filepath.Join(root, ".instructions", "INSTRUCTIONS_WORKFLOW.md"), modelAvailabilityPolicyFixture+"\n"+forcedFitPolicyFixture+"\n"+dirtyCheckoutPolicyFixture+"\n\n"+externalCILocalMirrorPolicySection+"\n")
	mustWrite(t, filepath.Join(root, ".configs", "claude-settings.json"), "{}")
	mustWrite(t, filepath.Join(root, ".configs", "codex-config.toml"), "model = \"gpt-5.6-sol\"\nmodel_context_window = 272000\nmodel_auto_compact_token_limit = 245000\nservice_tier = \"default\"\n\n[notice]\nhide_rate_limit_model_nudge = true\n")
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
	mustWrite(t, filepath.Join(root, "SKILL.md"), "# relux-agents-infra\n")
	mustWrite(t, filepath.Join(root, "README.md"), "# relux-agents-infra\n")
	mustWrite(t, filepath.Join(root, ".gitignore"), "ignored")
	mustWrite(t, filepath.Join(root, ".temp", "junk.txt"), "junk")
	mustWrite(t, filepath.Join(root, ".task-board", "README.md"), "board")
	mustWrite(t, filepath.Join(root, "task-board.config.json"), "{}")
	return root
}

func seedGlobalAgentsInfraTarget(t *testing.T, layout Layout) {
	t.Helper()
	target := filepath.Join(layout.BinDir, piInfraTargetName(ModeGlobal, runtime.GOOS))
	mustMkdir(t, filepath.Dir(target))
	body := "global agents-infra target fixture\n"
	if runtime.GOOS != "windows" {
		body = "#!/usr/bin/env sh\nprintf '%s\\n' 'agents-infra fixture commit=none build_date=none'\n"
	}
	mustWrite(t, target, body)
	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatalf("Chmod(%s): %v", target, err)
	}
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
