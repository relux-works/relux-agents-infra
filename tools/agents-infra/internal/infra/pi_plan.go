package infra

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const PrimarySessionManagedHostKindPiPTY = "pi-pty"

type PiLaunchPlanDetails struct {
	Managed               bool                 `json:"managed"`
	LogicalProfile        string               `json:"logical_profile,omitempty"`
	ProfileSource         string               `json:"profile_source,omitempty"`
	PiCompatibilitySource string               `json:"pi_compatibility_source,omitempty"`
	State                 *PiStatePaths        `json:"state,omitempty"`
	ModelsJSON            PiGeneratedCatalog   `json:"models_json,omitempty"`
	PiIdentity            *PiExecutionIdentity `json:"pi_identity,omitempty"`
	Runtime               *PiRuntimePlan       `json:"runtime,omitempty"`
	SharedRuntime         *PiSharedRuntimePlan `json:"shared_runtime,omitempty"`
	Capabilities          *PiCapabilityPlan    `json:"capabilities,omitempty"`
	DFlash                *PiDFlashPlan        `json:"dflash,omitempty"`
}
type PiSharedRuntimePlan struct {
	Mode          string             `json:"mode"`
	RuntimeKey    string             `json:"runtime_key"`
	ProfileDigest string             `json:"profile_digest"`
	Paths         SharedRuntimePaths `json:"paths"`
	Configured    PiRuntimeSharing   `json:"configured"`
	Broker        struct {
		Observed string `json:"observed"`
	} `json:"broker"`
}

type PiGeneratedCatalog struct {
	Path   string `json:"path,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}
type PiRuntimePlan struct {
	Executable             string   `json:"executable"`
	Argv                   []string `json:"argv"`
	Source                 string   `json:"source"`
	Endpoint               string   `json:"endpoint"`
	ReadinessURL           string   `json:"readiness_url"`
	StartupTimeoutSeconds  int      `json:"startup_timeout_seconds"`
	ShutdownTimeoutSeconds int      `json:"shutdown_timeout_seconds"`
	ExecutableState        string   `json:"executable_state"`
	Ownership              string   `json:"ownership"`
}
type PiCapabilityPlan struct {
	Requested    []string `json:"requested"`
	Verified     []string `json:"verified"`
	Verification string   `json:"verification"`
}
type PiDFlashPlan struct {
	Status      string   `json:"status"`
	TargetModel string   `json:"target_model"`
	DraftModel  string   `json:"draft_model"`
	TargetArgv  []string `json:"target_argv"`
	DraftArgv   []string `json:"draft_argv"`
}

type piModelsDocument struct {
	Providers map[string]piModelsProvider `json:"providers"`
}
type piModelsProvider struct {
	BaseURL string          `json:"baseUrl"`
	APIKey  string          `json:"apiKey"`
	API     string          `json:"api"`
	Models  []piModelsModel `json:"models"`
}
type piModelsModel struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	API           string             `json:"api"`
	Reasoning     bool               `json:"reasoning"`
	Input         []string           `json:"input"`
	ContextWindow int                `json:"contextWindow"`
	MaxTokens     int                `json:"maxTokens"`
	Cost          map[string]float64 `json:"cost"`
	Compat        map[string]any     `json:"compat,omitempty"`
}

func GeneratePiModelsJSON(profile PiProfile) ([]byte, error) {
	compat := map[string]any{}
	if profile.Compat.SupportsDeveloperRole != nil {
		compat["supportsDeveloperRole"] = *profile.Compat.SupportsDeveloperRole
	}
	if profile.Compat.SupportsReasoningEffort != nil {
		compat["supportsReasoningEffort"] = *profile.Compat.SupportsReasoningEffort
	}
	if profile.Compat.SupportsUsageStreaming != nil {
		compat["supportsUsageInStreaming"] = *profile.Compat.SupportsUsageStreaming
	}
	if profile.Compat.SupportsFinishReason != nil {
		compat["supportsFinishReason"] = *profile.Compat.SupportsFinishReason
	}
	if profile.Compat.MaxTokensField != nil {
		compat["maxTokensField"] = *profile.Compat.MaxTokensField
	}
	if profile.Compat.ThinkingFormat != nil {
		compat["thinkingFormat"] = *profile.Compat.ThinkingFormat
	}
	doc := piModelsDocument{Providers: map[string]piModelsProvider{profile.Provider: {BaseURL: profile.BaseURL, APIKey: "agents-infra-local", API: profile.API, Models: []piModelsModel{{ID: profile.Model, Name: profile.Model, API: profile.API, Reasoning: profile.Reasoning, Input: append([]string(nil), profile.Input...), ContextWindow: profile.ContextWindow, MaxTokens: profile.MaxTokens, Cost: map[string]float64{"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0}, Compat: compat}}}}}
	return json.MarshalIndent(doc, "", "  ")
}

func buildPiPrimarySessionLaunchPlan(result *PrimarySessionLaunchPlan, projectDir, homeDir string, userArgs []string, lookPath func(string) (string, error)) error {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	composite, err := loadCompositeProjectConfig(ancestorDirsRootFirst(projectDir), filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName))
	if err != nil {
		return err
	}
	for _, source := range composite.Sources {
		result.Sources = append(result.Sources, PrimarySessionSource{Kind: "project_config", Path: source.Path})
	}
	if err := validatePiPrimarySessionYolo(composite.PiPrimarySession); err != nil {
		return err
	}
	override, err := ExtractPiProfileOverride(userArgs)
	if err != nil {
		return err
	}
	selected := ""
	selectedSource := ""
	if override != nil {
		selected, selectedSource = *override, "cli"
	} else if composite.PiPrimarySession.Profile.Present {
		selected, selectedSource = composite.PiPrimarySession.Profile.Value, composite.PiPrimarySession.Profile.Source
	}
	if selected == "" {
		piPath, findErr := lookPath("pi")
		if findErr != nil {
			return piError("provider_executable_not_found", findErr)
		}
		result.Executable = piPath
		result.LaunchVariants.Interactive.Argv = append([]string(nil), userArgs...)
		result.LaunchVariants.ManagedHost = PrimarySessionManagedHostVariant{Kind: PrimarySessionManagedHostKindPiPTY, Argv: append([]string(nil), userArgs...)}
		result.LaunchVariants.ManagedClient.Argv = []string{}
		result.Resolved.Model = PrimarySessionResolvedString{Source: "native"}
		result.Resolved.Reasoning = PrimarySessionResolvedString{Source: "native"}
		result.Resolved.Profile = PrimarySessionResolvedString{Source: "native"}
		result.Resolved.PiCompatibility = PrimarySessionResolvedString{Source: "native"}
		result.Resolved.Yolo = resolvedPiYolo(composite.PiPrimarySession)
		result.Resolved.Sandbox = PrimarySessionResolvedString{Source: "not_applicable"}
		result.Resolved.Approval = PrimarySessionResolvedString{Source: "not_applicable"}
		result.Pi = &PiLaunchPlanDetails{Managed: false}
		return nil
	}
	if !composite.PiPrimarySession.PiCompatibility.Present {
		return piError("invalid_project_configuration", errors.New("managed Pi profile requires agents.pi.primary_session.pi_compatibility"))
	}
	profile, ok := composite.PiProfiles[selected]
	if !ok {
		return piError("unknown_pi_profile", fmt.Errorf("unknown Pi profile %q", selected))
	}
	if err := ValidatePiStateKeyCollisions(composite.PiProfiles); err != nil {
		return err
	}
	argsPlan, err := BuildManagedPiArguments(userArgs, selected, profile)
	if err != nil {
		return err
	}
	state, err := ResolvePiStatePaths("", projectDir, selected)
	if err != nil {
		return err
	}
	models, err := GeneratePiModelsJSON(profile)
	if err != nil {
		return err
	}
	modelsSum := sha256.Sum256(models)
	execState := staticExecutableState(profile.Runtime.Executable)
	piPath, findErr := lookPath("pi")
	if findErr != nil {
		return piError("provider_executable_not_found", findErr)
	}
	result.Executable = piPath
	identity, err := VerifyPiExecutionIdentity(result.Executable, composite.PiPrimarySession.PiCompatibility.Value)
	if err != nil {
		return err
	}
	result.Executable = identity.Entrypoint
	result.LaunchVariants.Interactive.Argv = argsPlan.DiagnosticArgv
	result.LaunchVariants.ManagedHost = PrimarySessionManagedHostVariant{Kind: PrimarySessionManagedHostKindPiPTY, Argv: argsPlan.DiagnosticArgv}
	result.LaunchVariants.ManagedClient.Argv = []string{}
	result.Resolved.Model = resolvedStringValue(profile.Provider+"/"+profile.Model, profile.Source)
	result.Resolved.Reasoning = resolvedStringValue(profile.Thinking, profile.Source)
	result.Resolved.Profile = resolvedStringValue(selected, selectedSource)
	result.Resolved.PiCompatibility = resolvedStringValue(composite.PiPrimarySession.PiCompatibility.Value, composite.PiPrimarySession.PiCompatibility.Source)
	result.Resolved.Yolo = resolvedPiYolo(composite.PiPrimarySession)
	result.Resolved.Sandbox = PrimarySessionResolvedString{Source: "not_applicable"}
	result.Resolved.Approval = PrimarySessionResolvedString{Source: "not_applicable"}
	details := &PiLaunchPlanDetails{Managed: true, LogicalProfile: selected, ProfileSource: selectedSource, PiCompatibilitySource: composite.PiPrimarySession.PiCompatibility.Source, State: &state, ModelsJSON: PiGeneratedCatalog{Path: state.ModelsJSON, SHA256: hex.EncodeToString(modelsSum[:])}, PiIdentity: &identity,
		Runtime:      &PiRuntimePlan{Executable: profile.Runtime.Executable, Argv: append([]string(nil), profile.Runtime.Argv...), Source: profile.Source, Endpoint: profile.BaseURL, ReadinessURL: strings.TrimSuffix(profile.BaseURL, "/v1") + "/v1" + profile.Runtime.ReadinessPath, StartupTimeoutSeconds: profile.Runtime.StartupTimeoutSeconds, ShutdownTimeoutSeconds: profile.Runtime.ShutdownTimeoutSeconds, ExecutableState: execState, Ownership: "direct-child-process-group"},
		Capabilities: &PiCapabilityPlan{Requested: append([]string(nil), profile.RequestedCapabilities...), Verified: []string{}, Verification: "not-claimed"}}
	if profile.Runtime.Sharing != nil && profile.Runtime.Sharing.Mode == "shared" {
		runtimeKey, profileDigest := SharedRuntimeKey(profile)
		paths, err := ResolveSharedRuntimePaths("", runtimeKey)
		if err != nil {
			return err
		}
		shared := &PiSharedRuntimePlan{Mode: "shared", RuntimeKey: runtimeKey, ProfileDigest: profileDigest, Paths: paths, Configured: *profile.Runtime.Sharing}
		shared.Broker.Observed = "not-inspected"
		details.SharedRuntime = shared
		details.Runtime.Ownership = "broker-owned-process-group"
	}
	if profile.Runtime.DFlash != nil {
		d := profile.Runtime.DFlash
		details.DFlash = &PiDFlashPlan{Status: "configured-unverified", TargetModel: d.TargetModel, DraftModel: d.DraftModel, TargetArgv: append([]string(nil), d.TargetArgv...), DraftArgv: append([]string(nil), d.DraftArgv...)}
	}
	result.Pi = details
	result.Sidecars = &PrimarySessionSidecars{LocalModel: *details.Runtime}
	result.Capabilities = details.Capabilities
	return nil
}

func staticExecutableState(path string) string {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "absent"
	}
	if err != nil {
		return "unknown"
	}
	if !info.Mode().IsRegular() {
		return "non-regular"
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "non-executable"
	}
	f, err := os.Open(path)
	if err != nil {
		return "unreadable"
	}
	f.Close()
	return "present"
}

func resolvePiNativeExecutable() (string, error) {
	path, err := exec.LookPath("pi")
	if err != nil {
		return "", piError("provider_executable_not_found", err)
	}
	return path, nil
}
