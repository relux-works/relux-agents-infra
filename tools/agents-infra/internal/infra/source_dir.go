package infra

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SourceDirEnv is the environment override naming the agents-infra source tree.
const SourceDirEnv = "AGENTS_INFRA_SOURCE_DIR"

// ConfigDirEnv is the environment override for the machine-scoped config dir
// that holds the install state written by scripts/setup.sh.
const ConfigDirEnv = "AGENTS_INFRA_CONFIG_DIR"

const installStateFileName = "install.json"

// sourceAsset is one path a usable agents-infra source tree must carry, paired
// with the runtime component that dereferences it. The contract is derived from
// what setup installs, not from a hand-picked set of recognisable file names: a
// tree that satisfies a label but lacks an asset the installed runtime needs is
// not a usable source, it is a tree that happens to look like one.
type sourceAsset struct {
	path     string
	consumer string
	// launcherBackend marks assets the generated local agents-infra launcher
	// builds at run time, so they are checked against the launcher's recorded
	// source dir as well as against the source tree itself.
	launcherBackend bool
}

func (a sourceAsset) label() string {
	return fmt.Sprintf("%s (%s)", filepath.ToSlash(a.path), a.consumer)
}

// sourceAssets are the paths every usable agents-infra source tree carries.
// Both a repo checkout and an installed .agents runtime provide them, so setup
// can be bootstrapped from either without a host-specific path.
var sourceAssets = []sourceAsset{
	{
		path:     filepath.Join(".instructions", "INSTRUCTIONS.md"),
		consumer: "Claude instructions entrypoint",
	},
	{
		path:     filepath.Join(".instructions", "AGENTS.md"),
		consumer: "rendered Codex instructions entrypoint",
	},
	{
		path:     ".configs",
		consumer: "linked agent config tree",
	},
	{
		path:     ".rules",
		consumer: "linked agent rules tree",
	},
	{
		path:     "SKILL.md",
		consumer: "materialized relux-agents-infra skill package",
	},
	{
		path:     "README.md",
		consumer: "relux-agents-infra skill reference",
	},
	// installCLIWrapper generates a launcher that builds this module; a source
	// without it mints a runtime whose agents-infra command cannot start.
	{
		path:            filepath.Join("tools", "agents-infra", "go.mod"),
		consumer:        "Go module the generated agents-infra launcher builds",
		launcherBackend: true,
	},
	{
		path:            filepath.Join("tools", "agents-infra", "main.go"),
		consumer:        "Go module the generated agents-infra launcher builds",
		launcherBackend: true,
	},
	{
		path:     filepath.Join("tools", "agents-infra", "internal", "infra", "pi-v0.84.2-darwin-arm64-tree-manifest.txt"),
		consumer: "authoritative 217-record managed Pi release-tree catalog",
	},
}

const piCatalogManifestSourceSHA256 = "2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b"

func piCatalogManifestFailure(sourceDir string) string {
	path := filepath.Join(sourceDir, "tools", "agents-infra", "internal", "infra", "pi-v0.84.2-darwin-arm64-tree-manifest.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("authoritative managed Pi release-tree catalog %s cannot be read completely: %v", path, err)
	}
	got := fmt.Sprintf("%x", sha256.Sum256(data))
	if got != piCatalogManifestSourceSHA256 {
		return fmt.Sprintf("authoritative managed Pi release-tree catalog %s has drifted: sha256 %s, want %s", path, got, piCatalogManifestSourceSHA256)
	}
	return ""
}

// SourceDirOrigin names where a source-tree candidate came from.
type SourceDirOrigin string

const (
	SourceDirOriginFlag             SourceDirOrigin = "--source-dir"
	SourceDirOriginEnv              SourceDirOrigin = SourceDirEnv
	SourceDirOriginInstallState     SourceDirOrigin = "install state repoPath"
	SourceDirOriginInstalledRuntime SourceDirOrigin = "installed runtime"
	SourceDirOriginLayout           SourceDirOrigin = "layout source dir"
)

// SourceDirAttempt records one candidate considered while resolving the source
// tree, including why it was rejected.
type SourceDirAttempt struct {
	Origin  SourceDirOrigin
	Path    string
	Missing []string
	Reason  string
}

func (a SourceDirAttempt) usable() bool {
	return a.Path != "" && a.Reason == "" && len(a.Missing) == 0
}

func (a SourceDirAttempt) describe() string {
	switch {
	case a.Path == "":
		reason := a.Reason
		if reason == "" {
			reason = "not provided"
		}
		return fmt.Sprintf("%s: %s", a.Origin, reason)
	case len(a.Missing) > 0:
		return fmt.Sprintf("%s %s: missing %s", a.Origin, a.Path, strings.Join(a.Missing, ", "))
	case a.Reason != "":
		return fmt.Sprintf("%s %s: %s", a.Origin, a.Path, a.Reason)
	default:
		return fmt.Sprintf("%s %s: usable", a.Origin, a.Path)
	}
}

// SourceDirError reports that setup has no usable agents-infra source tree. It
// always names the candidates that were considered and what each one lacked.
type SourceDirError struct {
	Mode     Mode
	Attempts []SourceDirAttempt
}

func (e *SourceDirError) Error() string {
	var b strings.Builder
	if len(e.Attempts) == 1 && e.Attempts[0].Path != "" {
		fmt.Fprintf(&b, "source dir is not a usable agents-infra source tree for %s setup: %s", e.modeLabel(), e.Attempts[0].describe())
	} else {
		fmt.Fprintf(&b, "source dir is required for %s setup and no usable agents-infra source tree was found:", e.modeLabel())
		for _, attempt := range e.Attempts {
			b.WriteString("\n  " + attempt.describe())
		}
	}
	b.WriteString("\nA usable source tree contains " + strings.Join(assetLabels(), ", ") + ",\nplus every instruction module its entrypoints include, and its tools/agents-infra\nmodule must be one `go build .` completes.")
	b.WriteString("\nPass --source-dir DIR or set " + SourceDirEnv + " to a relux-agents-infra checkout or an installed .agents runtime.")
	return b.String()
}

func (e *SourceDirError) modeLabel() string {
	if e.Mode == "" {
		return "agents-infra"
	}
	return string(e.Mode)
}

func assetLabels() []string {
	labels := make([]string, 0, len(sourceAssets))
	for _, asset := range sourceAssets {
		labels = append(labels, asset.label())
	}
	return labels
}

// MissingSourceDirAssets reports what a candidate source tree lacks: any
// required asset, plus any instruction module its entrypoints include but do
// not ship. An empty or non-directory path is missing everything.
//
// The include walk exists because the marker files themselves are trivially
// forgeable — an entrypoint that pulls in modules the tree does not carry
// installs a runtime that fails the first time it is rendered.
func MissingSourceDirAssets(dir string) []string {
	if dir == "" {
		return assetLabels()
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return assetLabels()
	}
	var missing []string
	for _, asset := range sourceAssets {
		if !pathExists(filepath.Join(dir, asset.path)) {
			missing = append(missing, asset.label())
		}
	}
	return append(missing, missingInstructionIncludes(dir)...)
}

// instructionEntrypoints are the instruction files setup renders; every module
// they include has to exist in the source tree for the render to succeed.
var instructionEntrypoints = []string{
	filepath.Join(".instructions", "INSTRUCTIONS.md"),
	filepath.Join(".instructions", "AGENTS.md"),
}

// missingInstructionIncludes resolves the @include closure of the instruction
// entrypoints against the source tree and reports every referenced module the
// tree does not carry. It mirrors resolveInstructionInclude, with ~/.agents/
// pointing at the source tree, because that is what the destination becomes.
// References outside the tree (absolute paths, other ~/ paths) name host state
// setup does not install and are left to the render step.
func missingInstructionIncludes(sourceDir string) []string {
	var missing []string
	seen := map[string]bool{}
	var walk func(path string)
	walk = func(path string) {
		if seen[path] {
			return
		}
		seen[path] = true
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		for _, line := range strings.SplitAfter(string(data), "\n") {
			ref, ok := parseInstructionInclude(line)
			if !ok {
				continue
			}
			resolved, ok := resolveSourceInstructionInclude(ref, filepath.Dir(path), sourceDir)
			if !ok {
				continue
			}
			if !pathExists(resolved) {
				rel, relErr := filepath.Rel(sourceDir, resolved)
				if relErr != nil {
					rel = resolved
				}
				missing = append(missing, fmt.Sprintf("%s (included by %s)", filepath.ToSlash(rel), filepath.ToSlash(ref)))
				continue
			}
			walk(resolved)
		}
	}
	for _, entrypoint := range instructionEntrypoints {
		walk(filepath.Join(sourceDir, entrypoint))
	}
	return missing
}

// resolveSourceInstructionInclude maps one @include reference onto the source
// tree. It reports false for references that name host state rather than an
// asset the source is expected to ship.
func resolveSourceInstructionInclude(ref, baseDir, sourceDir string) (string, bool) {
	const agentsHomePrefix = "~/.agents/"
	switch {
	case strings.HasPrefix(ref, agentsHomePrefix):
		return filepath.Join(sourceDir, filepath.FromSlash(strings.TrimPrefix(ref, agentsHomePrefix))), true
	case strings.HasPrefix(ref, "~/"), filepath.IsAbs(ref):
		return "", false
	default:
		resolved := filepath.Join(baseDir, filepath.FromSlash(ref))
		if !dirContains(sourceDir, resolved) {
			return "", false
		}
		return resolved, true
	}
}

// SourceDirRequest describes one source-tree resolution for a setup run.
type SourceDirRequest struct {
	Mode Mode
	// Flag is the --source-dir value; an explicit value is never silently
	// replaced by a discovered fallback.
	Flag string
	// Env is the AGENTS_INFRA_SOURCE_DIR value.
	Env string
	// ConfigDirOverride is AGENTS_INFRA_CONFIG_DIR.
	ConfigDirOverride string
	// XDGConfigHome is XDG_CONFIG_HOME, used on non-darwin hosts.
	XDGConfigHome string
	// GOOS selects the platform install-state location; empty means runtime.GOOS.
	GOOS string
	// HomeDir enables the install-state and installed-runtime fallbacks.
	HomeDir string
	// TargetAgentsDir is the directory setup syncs into. A candidate that would
	// contain it is rejected instead of syncing a tree into itself.
	TargetAgentsDir string
}

// ResolveSourceDir picks the agents-infra source tree for a setup run.
//
// Order: --source-dir, AGENTS_INFRA_SOURCE_DIR, the repoPath recorded by the
// installer in the machine-scoped install state, then the installed .agents
// runtime. An explicitly requested source is validated and never falls back, so
// a wrong path fails loudly instead of syncing something unrelated.
func ResolveSourceDir(req SourceDirRequest) (string, error) {
	if explicit := explicitSourceCandidate(req); explicit.Path != "" {
		attempt := evaluateSourceDirCandidate(explicit, req)
		if attempt.usable() {
			return attempt.Path, nil
		}
		return "", &SourceDirError{Mode: req.Mode, Attempts: []SourceDirAttempt{attempt}}
	}

	attempts := []SourceDirAttempt{
		{Origin: SourceDirOriginFlag},
		{Origin: SourceDirOriginEnv, Reason: "not set"},
	}
	for _, candidate := range discoveredSourceCandidates(req) {
		attempt := evaluateSourceDirCandidate(candidate, req)
		if attempt.usable() {
			return attempt.Path, nil
		}
		attempts = append(attempts, attempt)
	}
	return "", &SourceDirError{Mode: req.Mode, Attempts: attempts}
}

func explicitSourceCandidate(req SourceDirRequest) SourceDirAttempt {
	if strings.TrimSpace(req.Flag) != "" {
		return SourceDirAttempt{Origin: SourceDirOriginFlag, Path: req.Flag}
	}
	if strings.TrimSpace(req.Env) != "" {
		return SourceDirAttempt{Origin: SourceDirOriginEnv, Path: req.Env}
	}
	return SourceDirAttempt{}
}

func discoveredSourceCandidates(req SourceDirRequest) []SourceDirAttempt {
	statePath := installStatePath(req)
	stateCandidate := SourceDirAttempt{Origin: SourceDirOriginInstallState}
	switch {
	case statePath == "":
		stateCandidate.Reason = "no machine-scoped config dir to read"
	default:
		repoPath, err := readInstallStateRepoPath(statePath)
		switch {
		case err != nil:
			stateCandidate.Reason = fmt.Sprintf("%s: %v", statePath, err)
		case repoPath == "":
			stateCandidate.Reason = fmt.Sprintf("%s has no repoPath", statePath)
		default:
			stateCandidate.Path = repoPath
		}
	}

	runtimeCandidate := SourceDirAttempt{Origin: SourceDirOriginInstalledRuntime}
	if req.HomeDir == "" {
		runtimeCandidate.Reason = "no home directory to search"
	} else {
		runtimeCandidate.Path = filepath.Join(req.HomeDir, ".agents")
	}

	return []SourceDirAttempt{stateCandidate, runtimeCandidate}
}

func evaluateSourceDirCandidate(candidate SourceDirAttempt, req SourceDirRequest) SourceDirAttempt {
	if candidate.Path == "" {
		return candidate
	}
	if abs, err := filepath.Abs(candidate.Path); err == nil {
		candidate.Path = abs
	}
	// Global setup can source from the installed runtime itself, so reject that
	// cycle while selecting a candidate. Local setup performs the stronger
	// source-versus-project canonical identity check at the start of Setup,
	// where both fully resolved roots are available for the diagnostic.
	if req.Mode != ModeLocal && req.TargetAgentsDir != "" && dirContains(candidate.Path, req.TargetAgentsDir) {
		candidate.Reason = fmt.Sprintf("would sync into itself; it contains the destination %s", req.TargetAgentsDir)
		return candidate
	}
	candidate.Missing = MissingSourceDirAssets(candidate.Path)
	if len(candidate.Missing) == 0 {
		candidate.Reason = piCatalogManifestFailure(candidate.Path)
	}
	return candidate
}

func installStatePath(req SourceDirRequest) string {
	dir := configDir(req)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, installStateFileName)
}

func configDir(req SourceDirRequest) string {
	if strings.TrimSpace(req.ConfigDirOverride) != "" {
		return req.ConfigDirOverride
	}
	if req.GOOS == "darwin" {
		if req.HomeDir == "" {
			return ""
		}
		return filepath.Join(req.HomeDir, "Library", "Application Support", "agents-infra")
	}
	if strings.TrimSpace(req.XDGConfigHome) != "" {
		return filepath.Join(req.XDGConfigHome, "agents-infra")
	}
	if req.HomeDir == "" {
		return ""
	}
	return filepath.Join(req.HomeDir, ".config", "agents-infra")
}

func readInstallStateRepoPath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("no install state")
	}
	if err != nil {
		return "", fmt.Errorf("unreadable install state")
	}
	var state struct {
		RepoPath string `json:"repoPath"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("malformed install state")
	}
	return strings.TrimSpace(state.RepoPath), nil
}

// dirContains reports whether outer is inner or one of its ancestors.
func dirContains(outer, inner string) bool {
	outerAbs, err := filepath.Abs(outer)
	if err != nil {
		return false
	}
	innerAbs, err := filepath.Abs(inner)
	if err != nil {
		return false
	}
	outerAbs = filepath.Clean(outerAbs)
	innerAbs = filepath.Clean(innerAbs)
	if outerAbs == innerAbs {
		return true
	}
	rel, err := filepath.Rel(outerAbs, innerAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// requireLayoutSourceDir is the gate every Setup run passes before it copies a
// tree into a destination: a missing or non-agents-infra source is refused with
// a typed error naming what is absent.
func requireLayoutSourceDir(layout Layout) error {
	attempt := SourceDirAttempt{Origin: SourceDirOriginLayout, Path: layout.SourceDir}
	if attempt.Path == "" {
		attempt.Reason = "not set"
		return &SourceDirError{Mode: layout.Mode, Attempts: []SourceDirAttempt{attempt}}
	}
	if layout.AgentsDir != "" && dirContains(attempt.Path, layout.AgentsDir) {
		attempt.Reason = fmt.Sprintf("would sync into itself; it contains the destination %s", layout.AgentsDir)
		return &SourceDirError{Mode: layout.Mode, Attempts: []SourceDirAttempt{attempt}}
	}
	if missing := MissingSourceDirAssets(attempt.Path); len(missing) > 0 {
		attempt.Missing = missing
		return &SourceDirError{Mode: layout.Mode, Attempts: []SourceDirAttempt{attempt}}
	}
	if failure := piCatalogManifestFailure(attempt.Path); failure != "" {
		attempt.Reason = failure
		return &SourceDirError{Mode: layout.Mode, Attempts: []SourceDirAttempt{attempt}}
	}
	// The assets above are names; the launcher runs a build. Proving the build
	// here, before the destination is touched, is what keeps a module the
	// launcher cannot compile from being installed and then attested.
	if failure := launcherBackendSourceFailure(layout.Mode, attempt.Path); failure != "" {
		attempt.Reason = failure
		return &SourceDirError{Mode: layout.Mode, Attempts: []SourceDirAttempt{attempt}}
	}
	return nil
}
