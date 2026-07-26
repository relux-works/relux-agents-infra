package infra

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

const (
	PrimarySessionPreparationContract      = "agents-infra.primary-session-preparation"
	PrimarySessionPreparationSchemaVersion = 1

	PrimarySessionPreparationErrorUnsupportedSchemaVersion = "unsupported_schema_version"
	PrimarySessionPreparationErrorRenderFailed             = "render_failed"
)

type PrimarySessionPreparationReport struct {
	Contract                 string                         `json:"contract"`
	SchemaVersion            int                            `json:"schema_version"`
	Status                   string                         `json:"status"`
	Producer                 ChildLaunchCompositionProducer `json:"producer"`
	Provider                 string                         `json:"provider"`
	ProjectDir               string                         `json:"project_dir"`
	RuntimeProjectDir        string                         `json:"runtime_project_dir,omitempty"`
	LocalRuntimePresent      bool                           `json:"local_runtime_present"`
	CodexProjectRendered     bool                           `json:"codex_project_rendered"`
	CodexConfigGenerated     bool                           `json:"codex_config_generated"`
	ClaudeEntrypointRendered bool                           `json:"claude_entrypoint_rendered"`
	ClaudeInstructionsLinked bool                           `json:"claude_instructions_linked"`
	ClaudeSettingsLinked     bool                           `json:"claude_settings_linked"`
	Artifacts                []PrimarySessionArtifact       `json:"artifacts"`
}

type PrimarySessionArtifact struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	State  string `json:"state"`
	SHA256 string `json:"sha256,omitempty"`
	Target string `json:"target,omitempty"`
}

type PrimarySessionPreparationErrorEnvelope struct {
	Contract      string                         `json:"contract"`
	SchemaVersion int                            `json:"schema_version"`
	Status        string                         `json:"status"`
	Producer      ChildLaunchCompositionProducer `json:"producer"`
	Provider      string                         `json:"provider"`
	ProjectDir    string                         `json:"project_dir"`
	Error         PrimarySessionPreparationError `json:"error"`
}

type PrimarySessionPreparationError struct {
	Code string `json:"code"`
}

func NewPrimarySessionPreparationErrorEnvelope(
	provider string,
	projectDir string,
	producer ChildLaunchCompositionProducer,
	code string,
) PrimarySessionPreparationErrorEnvelope {
	return PrimarySessionPreparationErrorEnvelope{
		Contract:      PrimarySessionPreparationContract,
		SchemaVersion: PrimarySessionPreparationSchemaVersion,
		Status:        "error",
		Producer:      producer,
		Provider:      provider,
		ProjectDir:    projectDir,
		Error:         PrimarySessionPreparationError{Code: code},
	}
}

// PreparePrimarySession refreshes the installed provider-specific project
// surface without syncing source files and without launching a provider.
// Direct primary launchers and external session owners call this same function
// so their pre-launch filesystem effects cannot drift.
func PreparePrimarySession(
	provider string,
	projectDir string,
	producer ChildLaunchCompositionProducer,
) (PrimarySessionPreparationReport, error) {
	if provider != "codex" && provider != "claude" {
		return PrimarySessionPreparationReport{}, fmt.Errorf("unsupported provider %q", provider)
	}
	canonicalProjectDir, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return PrimarySessionPreparationReport{}, err
	}
	report := PrimarySessionPreparationReport{
		Contract:      PrimarySessionPreparationContract,
		SchemaVersion: PrimarySessionPreparationSchemaVersion,
		Status:        "ok",
		Producer:      producer,
		Provider:      provider,
		ProjectDir:    canonicalProjectDir,
		Artifacts:     []PrimarySessionArtifact{},
	}
	runtimeProjectDir, found, err := installedProjectRuntimeRoot(canonicalProjectDir)
	if err != nil {
		return report, err
	}
	if !found {
		return report, nil
	}
	layout, err := LocalLayout("", runtimeProjectDir)
	if err != nil {
		return report, err
	}
	report.LocalRuntimePresent = true
	report.RuntimeProjectDir = runtimeProjectDir

	switch provider {
	case "codex":
		if err := setupCodex(layout, CodexConfigModeLocal, nil); err != nil {
			return report, fmt.Errorf("prepare Codex project surface: %w", err)
		}
		report.CodexProjectRendered = isRenderedInstructionsFile(filepath.Join(layout.RootDir, "AGENTS.md"))
		report.CodexConfigGenerated = isGeneratedCodexConfigFile(filepath.Join(layout.CodexDir, "config.toml"))
		for _, artifact := range []struct {
			kind string
			path string
		}{
			{kind: "codex-instructions", path: filepath.Join(layout.CodexDir, "AGENTS.md")},
			{kind: "project-instructions", path: filepath.Join(layout.RootDir, "AGENTS.md")},
			{kind: "codex-config", path: filepath.Join(layout.CodexDir, "config.toml")},
		} {
			rendered, err := primarySessionArtifact(artifact.kind, artifact.path, "rendered")
			if err != nil {
				return report, err
			}
			report.Artifacts = append(report.Artifacts, rendered)
		}
		if !report.CodexProjectRendered || !report.CodexConfigGenerated {
			return report, fmt.Errorf("Codex project surface verification failed")
		}
	case "claude":
		if err := setupClaude(layout, nil); err != nil {
			return report, fmt.Errorf("prepare Claude project surface: %w", err)
		}
		report.ClaudeEntrypointRendered = isGeneratedClaudeEntrypointFile(filepath.Join(layout.ClaudeDir, "CLAUDE.md"))
		report.ClaudeInstructionsLinked = isLinkTo(
			filepath.Join(layout.ClaudeDir, "instructions"),
			filepath.Join(layout.AgentsDir, ".instructions"),
		)
		report.ClaudeSettingsLinked = isLinkTo(
			filepath.Join(layout.ClaudeDir, "settings.json"),
			filepath.Join(layout.AgentsDir, ".configs", "claude-settings.json"),
		)
		for _, artifact := range []struct {
			kind  string
			path  string
			state string
		}{
			{kind: "claude-entrypoint", path: filepath.Join(layout.ClaudeDir, "CLAUDE.md"), state: "rendered"},
			{kind: "claude-instructions", path: filepath.Join(layout.ClaudeDir, "instructions"), state: "linked"},
			{kind: "claude-settings", path: filepath.Join(layout.ClaudeDir, "settings.json"), state: "linked"},
		} {
			rendered, err := primarySessionArtifact(artifact.kind, artifact.path, artifact.state)
			if err != nil {
				return report, err
			}
			report.Artifacts = append(report.Artifacts, rendered)
		}
		if !report.ClaudeEntrypointRendered || !report.ClaudeInstructionsLinked || !report.ClaudeSettingsLinked {
			return report, fmt.Errorf("Claude project surface verification failed")
		}
	}
	return report, nil
}

func installedProjectRuntimeRoot(startDir string) (string, bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("resolve home directory: %w", err)
	}
	homeDir, err = filepath.Abs(homeDir)
	if err != nil {
		return "", false, fmt.Errorf("resolve home directory: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(homeDir); resolveErr == nil {
		homeDir = resolved
	}
	for current := startDir; ; current = filepath.Dir(current) {
		if filepath.Clean(current) == filepath.Clean(homeDir) {
			return "", false, nil
		}
		agentsDir := filepath.Join(current, ".agents")
		info, err := os.Stat(agentsDir)
		if err == nil {
			if !info.IsDir() {
				return "", false, fmt.Errorf("installed project runtime is not a directory: %s", agentsDir)
			}
			return current, true, nil
		}
		if !os.IsNotExist(err) {
			return "", false, fmt.Errorf("stat installed project runtime %s: %w", agentsDir, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
	}
}

func primarySessionArtifact(kind, path, state string) (PrimarySessionArtifact, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return PrimarySessionArtifact{}, fmt.Errorf("inspect prepared artifact %s: %w", path, err)
	}
	artifact := PrimarySessionArtifact{Kind: kind, Path: path, State: state}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return PrimarySessionArtifact{}, fmt.Errorf("read prepared artifact link %s: %w", path, err)
		}
		artifact.Target = target
		return artifact, nil
	}
	if !info.Mode().IsRegular() {
		return PrimarySessionArtifact{}, fmt.Errorf("prepared artifact is not a regular file or symlink: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return PrimarySessionArtifact{}, fmt.Errorf("read prepared artifact %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	artifact.SHA256 = fmt.Sprintf("%x", sum[:])
	return artifact, nil
}
