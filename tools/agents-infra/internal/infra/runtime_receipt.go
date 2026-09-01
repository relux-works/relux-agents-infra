package infra

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// runtimeReceiptFileName marks a runtime that a setup run installed *and*
// verified. Nothing else may write it: syncRepo skips it (see shouldSkip) so a
// source tree cannot ship a receipt of its own, and Setup deletes any existing
// one before it touches the destination. A destination therefore carries a
// receipt only while it is the finished product of a run that passed its
// postconditions — the existence of a .agents directory proves nothing.
const runtimeReceiptFileName = ".agents-infra-install.json"

// runtimeReceiptSchema is bumped when the receipt contract changes; an unknown
// schema is treated as no receipt rather than trusted.
const runtimeReceiptSchema = 1

// RuntimeReceipt records which destination a completed setup run produced.
type RuntimeReceipt struct {
	Schema    int    `json:"schema"`
	Mode      Mode   `json:"mode"`
	AgentsDir string `json:"agentsDir"`
	SourceDir string `json:"sourceDir"`
	BinDir    string `json:"binDir"`
}

// RuntimeVerificationError reports that an installed runtime cannot be trusted,
// naming every failed check rather than only the first.
type RuntimeVerificationError struct {
	AgentsDir string
	Failures  []string
}

func (e *RuntimeVerificationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "installed agent runtime at %s is not usable:", e.AgentsDir)
	for _, failure := range e.Failures {
		b.WriteString("\n  " + failure)
	}
	b.WriteString("\nRerun agents-infra setup against a complete relux-agents-infra source tree.")
	return b.String()
}

func runtimeReceiptPath(agentsDir string) string {
	return filepath.Join(agentsDir, runtimeReceiptFileName)
}

// invalidateRuntimeReceipt drops the receipt before a run mutates anything, so
// a run that fails part way through cannot leave the previous run's receipt
// vouching for a half-rewritten runtime.
func invalidateRuntimeReceipt(agentsDir string) error {
	if agentsDir == "" {
		return nil
	}
	err := os.Remove(runtimeReceiptPath(agentsDir))
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("invalidate install receipt %s: %w", runtimeReceiptPath(agentsDir), err)
}

func writeRuntimeReceipt(layout Layout) error {
	agentsDir, err := filepath.Abs(layout.AgentsDir)
	if err != nil {
		return fmt.Errorf("resolve agents dir for install receipt: %w", err)
	}
	receipt := RuntimeReceipt{
		Schema:    runtimeReceiptSchema,
		Mode:      layout.Mode,
		AgentsDir: agentsDir,
		SourceDir: absOrSelf(layout.SourceDir),
		BinDir:    absOrSelf(layout.BinDir),
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("encode install receipt: %w", err)
	}
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return fmt.Errorf("create agents dir for install receipt: %w", err)
	}
	if err := os.WriteFile(runtimeReceiptPath(agentsDir), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write install receipt: %w", err)
	}
	return nil
}

func absOrSelf(path string) string {
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// VerifyInstalledRuntime is the postcondition a caller can re-run at any time:
// the destination must carry a receipt minted for *this* location, the
// installed tree must carry every asset the runtime dereferences, and the
// generated launcher must point at a source that can actually build it.
//
// A receipt alone is not evidence — it is checked alongside the live artifacts
// precisely so a copied or hand-written receipt cannot stand in for them.
func VerifyInstalledRuntime(layout Layout) error {
	if layout.Mode == ModeLocal {
		if err := ValidateCanonicalProjectConfiguration(layout.RootDir, ""); err != nil {
			return err
		}
	}
	agentsDir := absOrSelf(layout.AgentsDir)
	failures := verifyRuntimeReceipt(layout, agentsDir)
	failures = append(failures, runtimeArtifactFailures(layout)...)
	if len(failures) == 0 {
		return nil
	}
	return &RuntimeVerificationError{AgentsDir: agentsDir, Failures: failures}
}

func verifyRuntimeReceipt(layout Layout, agentsDir string) []string {
	path := runtimeReceiptPath(agentsDir)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []string{fmt.Sprintf("no completed-install receipt at %s; the runtime was never installed, or the run that wrote it did not finish", path)}
	}
	if err != nil {
		return []string{fmt.Sprintf("unreadable install receipt %s: %v", path, err)}
	}
	var receipt RuntimeReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return []string{fmt.Sprintf("malformed install receipt %s", path)}
	}
	var failures []string
	if receipt.Schema != runtimeReceiptSchema {
		failures = append(failures, fmt.Sprintf("install receipt %s has unsupported schema %d (expected %d)", path, receipt.Schema, runtimeReceiptSchema))
	}
	// A receipt that names another destination was copied, not earned.
	if !samePath(receipt.AgentsDir, agentsDir) {
		failures = append(failures, fmt.Sprintf("install receipt %s was minted for %s, not %s", path, receipt.AgentsDir, agentsDir))
	}
	if layout.Mode != "" && receipt.Mode != "" && receipt.Mode != layout.Mode {
		failures = append(failures, fmt.Sprintf("install receipt %s records %s setup, not %s", path, receipt.Mode, layout.Mode))
	}
	return failures
}

// verifyRuntimeArtifacts is the postcondition Setup itself runs, before it is
// allowed to mint a receipt.
func verifyRuntimeArtifacts(layout Layout) error {
	failures := runtimeArtifactFailures(layout)
	if len(failures) == 0 {
		return nil
	}
	return &RuntimeVerificationError{AgentsDir: absOrSelf(layout.AgentsDir), Failures: failures}
}

func runtimeArtifactFailures(layout Layout) []string {
	var failures []string
	agentsDir := absOrSelf(layout.AgentsDir)
	if info, err := os.Stat(agentsDir); err != nil || !info.IsDir() {
		return []string{fmt.Sprintf("no installed runtime directory at %s", agentsDir)}
	}
	for _, asset := range sourceAssets {
		if !pathExists(filepath.Join(agentsDir, asset.path)) {
			failures = append(failures, fmt.Sprintf("installed runtime is missing %s", asset.label()))
		}
	}
	if failure := piCatalogManifestFailure(agentsDir); failure != "" {
		failures = append(failures, failure)
	}
	failures = append(failures, managedSkillLinkFailures(layout)...)
	failures = append(failures, piInfraLauncherFailures(layout)...)
	failures = append(failures, canonicalTargetLauncherFailures(layout)...)
	failures = append(failures, directProviderYoloLauncherFailures(layout)...)
	return append(failures, launcherBackendFailures(layout)...)
}

func directProviderYoloLauncherFailures(layout Layout) []string {
	goos := runtime.GOOS
	targetName := piInfraTargetName(layout.Mode, goos)
	var failures []string
	for _, launcher := range directProviderYoloLaunchers {
		aliasPath := filepath.Join(layout.BinDir, canonicalTargetWrapperName(launcher.name, goos))
		wantBody := directProviderYoloWrapperBody(launcher, goos, targetName)
		info, err := os.Lstat(aliasPath)
		switch {
		case os.IsNotExist(err):
			failures = append(failures, fmt.Sprintf("no generated %s launcher at %s", launcher.name, aliasPath))
			continue
		case err != nil:
			failures = append(failures, fmt.Sprintf("cannot inspect %s launcher %s: %v", launcher.name, aliasPath, err))
			continue
		case !info.Mode().IsRegular():
			failures = append(failures, fmt.Sprintf("%s launcher %s is not a regular file", launcher.name, aliasPath))
			continue
		}
		body, err := os.ReadFile(aliasPath)
		switch {
		case err != nil:
			failures = append(failures, fmt.Sprintf("unreadable %s launcher %s: %v", launcher.name, aliasPath, err))
			continue
		case string(body) != wantBody:
			failures = append(failures, fmt.Sprintf("%s launcher %s has drifted from direct provider YOLO dispatch", launcher.name, aliasPath))
			continue
		}
		if goos != "windows" && info.Mode().Perm() != 0o755 {
			failures = append(failures, fmt.Sprintf("%s launcher %s mode is %04o, want 0755", launcher.name, aliasPath, info.Mode().Perm()))
		}
	}
	return failures
}

func canonicalTargetLauncherFailures(layout Layout) []string {
	goos := runtime.GOOS
	targetName := piInfraTargetName(layout.Mode, goos)
	var failures []string
	for _, entrypoint := range canonicalTargetLauncherNames {
		aliasPath := filepath.Join(layout.BinDir, canonicalTargetWrapperName(entrypoint, goos))
		wantBody := canonicalTargetWrapperBody(entrypoint, goos, targetName)
		info, err := os.Lstat(aliasPath)
		switch {
		case os.IsNotExist(err):
			failures = append(failures, fmt.Sprintf("no generated %s launcher at %s", entrypoint, aliasPath))
			continue
		case err != nil:
			failures = append(failures, fmt.Sprintf("cannot inspect %s launcher %s: %v", entrypoint, aliasPath, err))
			continue
		case !info.Mode().IsRegular():
			failures = append(failures, fmt.Sprintf("%s launcher %s is not a regular file", entrypoint, aliasPath))
			continue
		}
		body, err := os.ReadFile(aliasPath)
		switch {
		case err != nil:
			failures = append(failures, fmt.Sprintf("unreadable %s launcher %s: %v", entrypoint, aliasPath, err))
			continue
		case string(body) != wantBody:
			failures = append(failures, fmt.Sprintf("%s launcher %s has drifted from canonical target dispatch", entrypoint, aliasPath))
			continue
		}
		if goos != "windows" && info.Mode().Perm() != 0o755 {
			failures = append(failures, fmt.Sprintf("%s launcher %s mode is %04o, want 0755", entrypoint, aliasPath, info.Mode().Perm()))
		}
	}
	return failures
}

func piInfraLauncherFailures(layout Layout) []string {
	goos := runtime.GOOS
	aliasPath := filepath.Join(layout.BinDir, piInfraWrapperName(goos))
	targetName := piInfraTargetName(layout.Mode, goos)
	wantBody := piInfraWrapperBody(goos, targetName)
	info, err := os.Lstat(aliasPath)
	switch {
	case os.IsNotExist(err):
		return []string{fmt.Sprintf("no generated pi-infra launcher at %s", aliasPath)}
	case err != nil:
		return []string{fmt.Sprintf("cannot inspect pi-infra launcher %s: %v", aliasPath, err)}
	case !info.Mode().IsRegular():
		return []string{fmt.Sprintf("pi-infra launcher %s is not a regular file", aliasPath)}
	}
	body, err := os.ReadFile(aliasPath)
	switch {
	case err != nil:
		return []string{fmt.Sprintf("unreadable pi-infra launcher %s: %v", aliasPath, err)}
	case string(body) != wantBody:
		return []string{fmt.Sprintf("pi-infra launcher %s has drifted from the managed %s target", aliasPath, targetName)}
	}
	if goos != "windows" && info.Mode().Perm() != 0o755 {
		return []string{fmt.Sprintf("pi-infra launcher %s mode is %04o, want 0755", aliasPath, info.Mode().Perm())}
	}
	targetPath := filepath.Join(layout.BinDir, targetName)
	targetInfo, err := os.Lstat(targetPath)
	switch {
	case os.IsNotExist(err):
		return []string{fmt.Sprintf("pi-infra launcher target is missing: %s", targetPath)}
	case err != nil:
		return []string{fmt.Sprintf("pi-infra launcher target cannot be read: %s: %v", targetPath, err)}
	case !targetInfo.Mode().IsRegular():
		return []string{fmt.Sprintf("pi-infra launcher target is not a regular file: %s", targetPath)}
	case goos != "windows" && targetInfo.Mode().Perm()&0o111 == 0:
		return []string{fmt.Sprintf("pi-infra launcher target is not executable: %s", targetPath)}
	}
	if layout.Mode == ModeGlobal && goos != "windows" {
		if failure := installedAgentsInfraStartupFailure(targetPath); failure != "" {
			return []string{fmt.Sprintf("pi-infra launcher target %s does not start as agents-infra: %s", targetPath, failure)}
		}
	}
	return nil
}

// launcherBackendFailures checks the artifact installCLIWrapper actually wrote:
// the launcher hardcodes a source dir and an output path, builds
// SOURCE/tools/agents-infra to that path on every invocation, and executes the
// result. A launcher whose source cannot be built, whose output path cannot be
// written, or whose built binary does not start is a runtime that reports
// itself installed and then fails on first use.
func launcherBackendFailures(layout Layout) []string {
	if layout.Mode == ModeGlobal {
		// Global setup does not generate a wrapper; the bootstrap owns it.
		return nil
	}
	if layout.BinDir == "" {
		return []string{"no bin dir recorded, so the generated agents-infra launcher cannot be checked"}
	}
	path := filepath.Join(layout.BinDir, cliWrapperName(runtime.GOOS))
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []string{fmt.Sprintf("no generated agents-infra launcher at %s", path)}
	}
	if err != nil {
		return []string{fmt.Sprintf("unreadable agents-infra launcher %s: %v", path, err)}
	}
	sourceDir, ok := generatedCLIWrapperSourceDir(string(body))
	if !ok {
		return []string{fmt.Sprintf("agents-infra launcher %s does not record a source dir", path)}
	}
	// Read the output path back out of the launcher instead of recomputing it.
	// A postcondition that assumes where the launcher builds is checking its own
	// assumption, and a build to some other destination proves nothing about the
	// one the consumer uses.
	binaryPath, ok := generatedCLIWrapperBinaryPath(string(body))
	if !ok {
		return []string{fmt.Sprintf("agents-infra launcher %s does not record the binary it builds and executes", path)}
	}
	var failures []string
	for _, asset := range sourceAssets {
		if !asset.launcherBackend {
			continue
		}
		if !pathExists(filepath.Join(sourceDir, asset.path)) {
			failures = append(failures, fmt.Sprintf("agents-infra launcher %s builds %s, which is missing %s", path, sourceDir, asset.label()))
		}
	}
	if len(failures) > 0 {
		// The named-asset failures above already say what is absent; running a
		// build that is guaranteed to fail would only repeat it less clearly.
		return failures
	}
	// Every path the launcher reads is present, which is not the same claim as
	// "the launcher works". The postcondition runs the launcher's whole
	// operation: build to the launcher's own output path, then start the result.
	if failure := launcherStartupFailure(sourceDir, binaryPath); failure != "" {
		return []string{fmt.Sprintf("agents-infra launcher %s cannot start: %s", path, failure)}
	}
	return nil
}

// generatedCLIWrapperSourceDir extracts the source dir cliWrapperBody baked
// into a generated launcher.
func generatedCLIWrapperSourceDir(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if value, ok := strings.CutPrefix(trimmed, "export AGENTS_INFRA_SOURCE_DIR="); ok {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return "", false
			}
			return unquoted, unquoted != ""
		}
		if value, ok := strings.CutPrefix(trimmed, `set "AGENTS_INFRA_SOURCE_DIR=`); ok {
			value = strings.TrimSuffix(value, `"`)
			return value, value != ""
		}
	}
	return "", false
}

// generatedCLIWrapperBinaryPath extracts the build output path cliWrapperBody
// baked into a generated launcher — the destination the launcher writes on
// every invocation and then executes.
func generatedCLIWrapperBinaryPath(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if value, ok := strings.CutPrefix(trimmed, `set "AGENTS_INFRA_BINARY=`); ok {
			value = strings.TrimSuffix(value, `"`)
			return value, value != ""
		}
		if value, ok := strings.CutPrefix(trimmed, "AGENTS_INFRA_BINARY="); ok {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return "", false
			}
			return unquoted, unquoted != ""
		}
	}
	return "", false
}
