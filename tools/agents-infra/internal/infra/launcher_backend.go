package infra

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// launcherBackendBuildTimeout bounds the verification build, so a wedged or
// offline toolchain surfaces as a refusal that says so rather than as a hang.
const launcherBackendBuildTimeout = 10 * time.Minute

// launcherStartupTimeout bounds the probe run. Answering `version` is a
// constant-time operation; a binary that cannot do it inside this window is not
// a launcher backend anyone can use.
const launcherStartupTimeout = 2 * time.Minute

// launcherStartupProbeArg is the argv the attestation drives the built binary
// with. It has to be a real command the CLI serves, bounded, side-effect free
// and stable across versions — `version` is the only one that is all four.
const launcherStartupProbeArg = "version"

// launcherStartupIdentity is what `agents-infra version` prints; see runVersion
// in the CLI's main package, which writes
//
//	agents-infra <version> commit=<commit> build_date=<date>
//
// Requiring it is what separates "some process ran and exited zero" from "the
// agents-infra CLI started".
const launcherStartupIdentity = "agents-infra "

// launcherBackendBuildOutputLines caps how much command output a refusal
// carries. The build names what it could not resolve in its first lines, and a
// refusal is only useful if a human can read it.
const launcherBackendBuildOutputLines = 20

// launcherBackendModuleDir is the Go module the generated local launcher builds
// on every invocation; see cliWrapperBody, which writes
//
//	cd "$AGENTS_INFRA_SOURCE_DIR/tools/agents-infra" && go build -o ... .
func launcherBackendModuleDir(sourceDir string) string {
	return filepath.Join(sourceDir, "tools", "agents-infra")
}

// launcherStartupFailure performs, end to end, what the generated launcher
// performs on every invocation — create the build output directory, build the
// module to that exact path, execute the result — and reports why it failed, or
// "" when the launcher can start.
//
// Each narrower claim this replaces was satisfied by a tree that then failed on
// first use. Checking that go.mod and main.go exist is a proxy for the build: a
// module can carry both names and still miss the packages `go build` needs.
// Checking that the module builds is a proxy for the program: a module can
// compile cleanly into a binary that exits non-zero the moment it is run, or
// into a binary that is not agents-infra at all. And building to some unrelated
// temporary directory is a proxy for the launcher's real output path: the
// build the consumer runs can fail on a destination this one never touched.
//
// So the attested property is the consumer's whole operation, including where
// its output lands and whether the result answers as the CLI it claims to be.
//
// Cost of that choice: setup and verify need a Go toolchain and pay for one
// build plus one bounded process start. The launcher already needs both on
// every invocation, so this adds no new requirement to the runtime — and
// because setup builds to the launcher's own output path, the launcher's first
// invocation is a cache hit against an artifact that is already there.
func launcherStartupFailure(sourceDir, binaryPath string) string {
	moduleDir := launcherBackendModuleDir(sourceDir)
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Sprintf("the generated agents-infra launcher builds %s with `go build`, but no Go toolchain is available: %v", moduleDir, err)
	}
	binaryPath = absOrSelf(binaryPath)
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return fmt.Sprintf("the generated agents-infra launcher builds to %s on every invocation, and that path cannot be prepared: %v", binaryPath, err)
	}

	buildCtx, cancelBuild := context.WithTimeout(context.Background(), launcherBackendBuildTimeout)
	defer cancelBuild()
	build := exec.CommandContext(buildCtx, goBin, "build", "-o", binaryPath, ".")
	build.Dir = moduleDir
	buildOutput, buildErr := build.CombinedOutput()
	switch {
	case buildCtx.Err() != nil:
		return fmt.Sprintf("`go build -o %s .` in %s did not finish within %s, so the generated agents-infra launcher cannot be shown to start", binaryPath, moduleDir, launcherBackendBuildTimeout)
	case buildErr != nil:
		return fmt.Sprintf("the generated agents-infra launcher runs `go build -o %s .` in %s on every invocation, and that build fails (%v):\n%s", binaryPath, moduleDir, buildErr, indentBuildOutput(buildOutput))
	}

	// Compiling is not starting. The launcher execs what it just built, so the
	// postcondition has to as well.
	runCtx, cancelRun := context.WithTimeout(context.Background(), launcherStartupTimeout)
	defer cancelRun()
	probe := exec.CommandContext(runCtx, binaryPath, launcherStartupProbeArg)
	probeOutput, probeErr := probe.CombinedOutput()
	switch {
	case runCtx.Err() != nil:
		return fmt.Sprintf("the generated agents-infra launcher execs %s, and `%s %s` did not answer within %s", binaryPath, binaryPath, launcherStartupProbeArg, launcherStartupTimeout)
	case probeErr != nil:
		return fmt.Sprintf("the generated agents-infra launcher execs %s, and `%s %s` fails (%v):\n%s", binaryPath, binaryPath, launcherStartupProbeArg, probeErr, indentBuildOutput(probeOutput))
	case !strings.Contains(string(probeOutput), launcherStartupIdentity):
		return fmt.Sprintf("the generated agents-infra launcher execs %s, and `%s %s` does not identify it as agents-infra (nothing matching %q in its output):\n%s", binaryPath, binaryPath, launcherStartupProbeArg, launcherStartupIdentity, indentBuildOutput(probeOutput))
	}
	return ""
}

// launcherBackendSourceFailure gates a candidate source tree on the runtime it
// would install, before Setup is allowed to write the destination. The target's
// bin dir does not exist yet at this point and must not be created by a run
// that is about to refuse, so the probe builds to a throwaway path; the
// launcher's real output path is attested by the postcondition, which runs
// against the launcher that was actually written.
//
// Global setup generates no launcher — the bootstrap owns
// ~/.local/bin/agents-infra — so it has nothing to build and nothing to claim
// about a build.
func launcherBackendSourceFailure(mode Mode, sourceDir string) string {
	if mode == ModeGlobal {
		return ""
	}
	outDir, err := os.MkdirTemp("", "agents-infra-launcher-backend-")
	if err != nil {
		return fmt.Sprintf("cannot build %s to check the generated agents-infra launcher: %v", launcherBackendModuleDir(sourceDir), err)
	}
	defer os.RemoveAll(outDir)
	return launcherStartupFailure(sourceDir, filepath.Join(outDir, cliLocalBinaryName(runtime.GOOS)))
}

// indentBuildOutput trims command output to its leading lines and indents it,
// so a refusal reads as one message rather than as an unattributed dump.
func indentBuildOutput(output []byte) string {
	trimmed := strings.TrimRight(string(output), "\n")
	if trimmed == "" {
		return "    (no output)"
	}
	lines := strings.Split(trimmed, "\n")
	truncated := false
	if len(lines) > launcherBackendBuildOutputLines {
		lines = lines[:launcherBackendBuildOutputLines]
		truncated = true
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString("    " + line + "\n")
	}
	if truncated {
		b.WriteString("    ...\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
