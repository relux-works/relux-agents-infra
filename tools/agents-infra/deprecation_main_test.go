package main

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Deprecated entrypoint contract: one exact line on stderr, empty stdout,
// exit 1, no provider launch, no Curator auto-exec, no filesystem changes, and
// no config reads or provider resolution. Guards run before any parsing, so
// every argument shape — including --print-config, --help, danger flags,
// malformed provider args, and spawn — receives the same message.
var deprecatedEntrypointMessages = map[string]string{
	"codex":           "agents-infra codex is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release.",
	"claude":          "agents-infra claude is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release.",
	"openai-infra":    "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release.",
	"anthropic-infra": "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release.",
	"openai-dange":    "openai-dange is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release.",
	"anthropic-dange": "anthropic-dange is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release.",
}

func TestDeprecatedProviderErrorMessagesAreExact(t *testing.T) {
	for entrypoint, want := range deprecatedEntrypointMessages {
		t.Run(entrypoint, func(t *testing.T) {
			err := deprecatedProviderError(entrypoint)
			if err == nil || err.Error() != want {
				t.Fatalf("deprecatedProviderError(%q) = %v, want exactly %q", entrypoint, err, want)
			}
			if strings.Contains(want, "\n") {
				t.Fatalf("deprecation message must be one line: %q", want)
			}
		})
	}
	if err := deprecatedProviderError("qwen-infra"); err == nil || !strings.Contains(err.Error(), "unknown deprecated provider entrypoint") {
		t.Fatalf("live entrypoint must not resolve to a deprecation message: %v", err)
	}
}

func TestRunCodexAndClaudeAreDeprecatedForEveryArgumentShape(t *testing.T) {
	argShapes := [][]string{
		nil,
		{"--print-config"},
		{"--help"},
		{"-d", "--danger", "--yolo"},
		{"--model", "{bad", "--", "exec", "inspect"},
		{"spawn", "--prompt", "x"},
	}
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			for _, args := range argShapes {
				var err error
				if provider == "codex" {
					err = runCodex(args)
				} else {
					err = runClaude(args)
				}
				if err == nil || err.Error() != deprecatedEntrypointMessages[provider] {
					t.Fatalf("run%s(%q) = %v, want exactly %q", strings.Title(provider), args, err, deprecatedEntrypointMessages[provider])
				}
			}
		})
	}
}

// TestDeprecatedEntrypointsExitOneWithNoSideEffects drives the real compiled
// binary, every installed deprecated-alias wrapper, and the local cliWrapper
// with deprecated subcommands. Fake provider and Curator executables write a
// sentinel if anything launches them; the project and home trees are
// snapshotted without exclusions before and after each invocation.
//
// The fixture proves the refusal happens before any build: the cached wrapper
// build output is deleted after setup, and a failing `go` shadows the real
// toolchain on PATH during every probe. A wrapper that still built would exit
// 73 with "unexpected-build" and recreate the build directory instead of
// exiting 1 with the migration notice.
func TestDeprecatedEntrypointsExitOneWithNoSideEffects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable deprecation test; in-process guard tests cover Windows dispatch")
	}
	binary := buildInstalledBinary(t)
	home := t.TempDir()
	project := t.TempDir()

	// The codex-local shim only exists with a nonempty project MCP opt-in.
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	mustWrite(t, filepath.Join(project, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"figma\"]\n")
	setup := exec.Command(binary, "setup", "local", project, "--source-dir", sourceRepoRoot(t))
	setup.Env = append(os.Environ(), "HOME="+home, "AGENTS_INFRA_CONFIG_DIR="+filepath.Join(home, "config"))
	setup.Env = append(setup.Env, sharedGoCacheEnv(t)...)
	if output, err := setup.CombinedOutput(); err != nil {
		t.Fatalf("setup local: %v\n%s", err, output)
	}
	for _, wrapper := range []string{"openai-infra", "anthropic-infra", "openai-dange", "anthropic-dange", "codex-local", "agents-infra"} {
		if _, err := os.Stat(filepath.Join(project, ".local", "bin", wrapper)); err != nil {
			t.Fatalf("installed wrapper %s missing: %v", wrapper, err)
		}
	}

	fakeBin := t.TempDir()
	sentinel := filepath.Join(t.TempDir(), "launch-sentinel")
	for _, name := range []string{"codex", "claude", "curator"} {
		mustWrite(t, filepath.Join(fakeBin, name), "#!/bin/sh\necho \""+name+" launched: $@\" >> \""+sentinel+"\"\nexit 0\n")
		if err := os.Chmod(filepath.Join(fakeBin, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Installed only after setup (which needs the real toolchain): any probe
	// that still shells out to `go build` fails here instead of rebuilding.
	mustWrite(t, filepath.Join(fakeBin, "go"), "#!/bin/sh\necho unexpected-build >&2\nexit 73\n")
	if err := os.Chmod(filepath.Join(fakeBin, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Delete the cached wrapper build output setup left behind: a wrapper
	// that rebuilds before refusing would recreate this directory, which the
	// exclusion-free snapshot below would catch as a filesystem change.
	if err := os.RemoveAll(filepath.Join(project, ".local", "bin", ".agents-infra-build")); err != nil {
		t.Fatalf("remove cached wrapper build output: %v", err)
	}
	deprecationEnv := func() []string {
		environ := append(os.Environ(),
			"HOME="+home,
			"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
			"AGENTS_INFRA_CONFIG_DIR="+filepath.Join(home, "config"),
			"AGENTS_INFRA_SOURCE_DIR=",
			"AGENTS_INFRA_CALLER_CWD=",
			"GOTELEMETRY=off",
		)
		return append(environ, sharedGoCacheEnv(t)...)
	}

	argShapes := [][]string{
		nil,
		{"--print-config"},
		{"--help"},
		{"-d", "--danger", "--yolo"},
		{"--model", "{bad", "--", "exec", "inspect"},
		{"spawn", "--prompt", "x"},
	}
	directCases := []struct {
		name string
		argv []string
		want string
	}{
		{name: "codex", argv: []string{"codex"}, want: deprecatedEntrypointMessages["codex"]},
		{name: "claude", argv: []string{"claude"}, want: deprecatedEntrypointMessages["claude"]},
		{name: "target openai-infra", argv: []string{"target", "openai-infra"}, want: deprecatedEntrypointMessages["openai-infra"]},
		{name: "target anthropic-infra", argv: []string{"target", "anthropic-infra"}, want: deprecatedEntrypointMessages["anthropic-infra"]},
		{name: "target-yolo openai-infra", argv: []string{"target-yolo", "openai-infra"}, want: deprecatedEntrypointMessages["openai-dange"]},
		{name: "target-yolo anthropic-infra", argv: []string{"target-yolo", "anthropic-infra"}, want: deprecatedEntrypointMessages["anthropic-dange"]},
	}
	for _, testCase := range directCases {
		for _, args := range argShapes {
			argv := append(append([]string{}, testCase.argv...), args...)
			t.Run(testCase.name+"/"+fmt.Sprint(argv), func(t *testing.T) {
				before := snapshotTree(t, project, home)
				stdout, stderr, exitCode := runDeprecationProbe(t, binary, project, deprecationEnv(), argv...)
				assertDeprecationOutcome(t, testCase.want, stdout, stderr, exitCode)
				assertNoDeprecationSideEffects(t, project, home, sentinel, before)
			})
		}
	}

	wrapperShapes := [][]string{nil, {"--print-config", "-d", "--danger"}}
	wrapperCases := []struct {
		name string
		want string
	}{
		{name: "openai-infra", want: deprecatedEntrypointMessages["openai-infra"]},
		{name: "anthropic-infra", want: deprecatedEntrypointMessages["anthropic-infra"]},
		{name: "openai-dange", want: deprecatedEntrypointMessages["openai-dange"]},
		{name: "anthropic-dange", want: deprecatedEntrypointMessages["anthropic-dange"]},
		{name: "codex-local", want: deprecatedEntrypointMessages["codex"]},
	}
	for _, testCase := range wrapperCases {
		for _, args := range wrapperShapes {
			t.Run("wrapper/"+testCase.name+"/"+fmt.Sprint(args), func(t *testing.T) {
				before := snapshotTree(t, project, home)
				stdout, stderr, exitCode := runDeprecationProbe(t, filepath.Join(project, ".local", "bin", testCase.name), project, deprecationEnv(), args...)
				assertDeprecationOutcome(t, testCase.want, stdout, stderr, exitCode)
				assertNoDeprecationSideEffects(t, project, home, sentinel, before)
			})
		}
	}

	// The local cliWrapper must refuse deprecated subcommands before it
	// creates its build directory or invokes the toolchain: same assertions,
	// same failing-go/absent-build fixture as the alias wrappers.
	cliWrapper := filepath.Join(project, ".local", "bin", "agents-infra")
	for _, testCase := range directCases {
		for _, args := range wrapperShapes {
			argv := append(append([]string{}, testCase.argv...), args...)
			t.Run("cliWrapper/"+testCase.name+"/"+fmt.Sprint(argv), func(t *testing.T) {
				before := snapshotTree(t, project, home)
				stdout, stderr, exitCode := runDeprecationProbe(t, cliWrapper, project, deprecationEnv(), argv...)
				assertDeprecationOutcome(t, testCase.want, stdout, stderr, exitCode)
				assertNoDeprecationSideEffects(t, project, home, sentinel, before)
			})
		}
	}
}

func runDeprecationProbe(t *testing.T, executable, dir string, env []string, argv ...string) (string, string, int) {
	t.Helper()
	command := exec.Command(executable, argv...)
	command.Dir = dir
	command.Env = env
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("probe %s %q did not exit normally: %v", executable, argv, err)
		}
		exitCode = exitErr.ExitCode()
	}
	return stdout.String(), stderr.String(), exitCode
}

func assertDeprecationOutcome(t *testing.T, want, stdout, stderr string, exitCode int) {
	t.Helper()
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1 (stdout=%q stderr=%q)", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if stderr != want+"\n" {
		t.Fatalf("stderr = %q, want exactly %q", stderr, want+"\n")
	}
}

func assertNoDeprecationSideEffects(t *testing.T, project, home, sentinel string, before map[string]string) {
	t.Helper()
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		data, _ := os.ReadFile(sentinel)
		t.Fatalf("deprecated entrypoint launched a provider or Curator: %q", data)
	}
	after := snapshotTree(t, project, home)
	if len(after) != len(before) {
		t.Fatalf("filesystem changed: %d entries before, %d after", len(before), len(after))
	}
	for path, hash := range before {
		if after[path] != hash {
			t.Fatalf("filesystem changed at %s", path)
		}
	}
}

// snapshotTree hashes every file, symlink target, and directory entry under
// the given roots with no exclusions. Deprecated entrypoints must not write
// anywhere — including the wrapper build output dir — so a rebuild before
// refusing shows up here as a before/after mismatch.
func snapshotTree(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			key := root + string(os.PathSeparator) + relative
			info, err := entry.Info()
			if err != nil {
				return err
			}
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				snapshot[key] = "link:" + target
			case info.IsDir():
				snapshot[key] = "dir"
			default:
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				sum := sha256.Sum256(data)
				snapshot[key] = fmt.Sprintf("file:%x:%d", sum[:], info.Mode().Perm())
			}
			return nil
		})
		if err != nil {
			t.Fatalf("snapshot %s: %v", root, err)
		}
	}
	return snapshot
}
