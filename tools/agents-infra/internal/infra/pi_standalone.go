package infra

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	PiStandaloneLaunchPlanContract      = "agents-infra.pi-standalone-launch-plan"
	PiStandaloneLaunchPlanSchemaVersion = 1
	PiStandaloneResultContract          = "agents-infra.pi-standalone-result"
)

var piStandaloneAllowedTools = map[string]bool{
	"read":  true,
	"bash":  true,
	"edit":  true,
	"write": true,
	"grep":  true,
	"find":  true,
	"ls":    true,
}

// PiStandaloneRequest selects the separately authorized, non-interactive Pi
// path. Entrypoint is used by qwen-infra; an empty entrypoint selects the
// effective primary managed Pi profile for the direct pi-infra command.
// ClientRunID is an internal test/lifecycle seam and is never caller CLI input.
type PiStandaloneRequest struct {
	Prompt          string
	Entrypoint      string
	ExpectedProfile string
	ClientRunID     string
}

type PiStandaloneToolAuthorization struct {
	Mode                 string   `json:"mode"`
	Effective            bool     `json:"effective"`
	Scope                string   `json:"scope"`
	YoloSource           string   `json:"yolo_source"`
	AllowlistSource      string   `json:"allowlist_source"`
	AllowedTools         []string `json:"allowed_tools"`
	NativeEnforcement    string   `json:"native_enforcement"`
	ExtensionDiscovery   string   `json:"extension_discovery"`
	ProjectTrust         string   `json:"project_trust"`
	RPCDirectBash        string   `json:"rpc_direct_bash"`
	PrivilegeBoundary    string   `json:"privilege_boundary"`
	ArgumentOwnership    string   `json:"argument_ownership"`
	HumanApprovalOrStdin string   `json:"human_approval_or_stdin"`
	TaskBoardAdapter     string   `json:"task_board_adapter"`
}

type PiStandaloneStatePlan struct {
	Isolation   string `json:"isolation"`
	Persistence string `json:"persistence"`
}

type PiStandaloneLaunchPlan struct {
	Contract          string                         `json:"contract"`
	SchemaVersion     int                            `json:"schema_version"`
	Status            string                         `json:"status"`
	Producer          ChildLaunchCompositionProducer `json:"producer"`
	ProjectDir        string                         `json:"project_dir"`
	Entrypoint        string                         `json:"entrypoint,omitempty"`
	Target            *PrimarySessionTarget          `json:"target,omitempty"`
	Executable        string                         `json:"executable"`
	Argv              []string                       `json:"argv"`
	Profile           PrimarySessionResolvedString   `json:"profile"`
	Model             PrimarySessionResolvedString   `json:"model"`
	Reasoning         PrimarySessionResolvedString   `json:"reasoning"`
	PiCompatibility   PrimarySessionResolvedString   `json:"pi_compatibility"`
	ToolAuthorization PiStandaloneToolAuthorization  `json:"tool_authorization"`
	State             PiStandaloneStatePlan          `json:"state"`
	RuntimeMode       string                         `json:"runtime_mode"`
	Runtime           PiRuntimePlan                  `json:"runtime"`
	LifecycleLogs     PiLifecycleLogPlan             `json:"lifecycle_logs"`
	PiIdentity        PiExecutionIdentity            `json:"pi_identity"`
	Sources           []PrimarySessionSource         `json:"sources"`
}

// PiStandaloneFailure deliberately carries only a machine code. The wrapped
// diagnostic remains available to in-process callers through Unwrap, while
// the CLI surface never prints prompts, environment values, or raw child
// output as an error message.
type PiStandaloneFailure struct {
	Code string
	Err  error
}

func (e *PiStandaloneFailure) Error() string {
	return fmt.Sprintf(`{"contract":%q,"schema_version":1,"status":"error","error":{"code":%q}}`, PiStandaloneResultContract, e.Code)
}

func (e *PiStandaloneFailure) Unwrap() error { return e.Err }

func WrapPiStandaloneFailure(err error) error {
	if err == nil {
		return nil
	}
	code := "pi_standalone_failed"
	var launch *PiLaunchError
	var target *CanonicalTargetError
	var shared *SharedRuntimeError
	switch {
	case errors.As(err, &launch):
		code = launch.Code
	case errors.As(err, &target):
		code = target.Code
	case errors.As(err, &shared):
		code = shared.Code
	}
	return &PiStandaloneFailure{Code: code, Err: err}
}

func validatePiStandalonePolicy(policy PiStandaloneSessionPolicy) error {
	if !policy.YoloMode.Present || !policy.YoloMode.Value {
		return piError("pi_tool_authorization_required", errors.New("standalone Pi tool execution requires agents.pi.standalone_session.yolo_mode=true"))
	}
	if !policy.ToolAllowlist.Present {
		return piError("pi_tool_allowlist_required", errors.New("standalone Pi tool execution requires an explicit tool_allowlist"))
	}
	if len(policy.ToolAllowlist.Value) == 0 {
		return piError("pi_tool_allowlist_invalid", errors.New("standalone Pi tool_allowlist must not be empty"))
	}
	seen := map[string]bool{}
	for _, tool := range policy.ToolAllowlist.Value {
		if tool == "" || !piStandaloneAllowedTools[tool] {
			return piError("pi_tool_allowlist_invalid", errors.New("standalone Pi tool_allowlist contains an empty or unsupported pinned tool name"))
		}
		if seen[tool] {
			return piError("pi_tool_allowlist_invalid", errors.New("standalone Pi tool_allowlist contains a duplicate tool name"))
		}
		seen[tool] = true
	}
	return nil
}

func validatePiStandaloneRequest(request PiStandaloneRequest, callerArgs []string) error {
	if len(callerArgs) != 0 {
		return piError("pi_standalone_conflicting_arguments", errors.New("standalone Pi does not accept caller-controlled provider arguments"))
	}
	if strings.TrimSpace(request.Prompt) == "" {
		return piError("pi_standalone_prompt_invalid", errors.New("standalone Pi prompt must not be empty"))
	}
	if strings.IndexByte(request.Prompt, 0) >= 0 {
		return piError("pi_standalone_prompt_invalid", errors.New("standalone Pi prompt cannot contain NUL"))
	}
	if request.Entrypoint != "" && request.Entrypoint != "qwen-infra" {
		return piError("pi_standalone_entrypoint_invalid", errors.New("standalone Pi admits only the qwen-infra canonical entrypoint"))
	}
	return nil
}

func BuildStandalonePiArguments(callerArgs []string, profile PiProfile, policy PiStandaloneSessionPolicy, prompt string) (PiArgumentPlan, error) {
	request := PiStandaloneRequest{Prompt: prompt}
	if err := validatePiStandaloneRequest(request, callerArgs); err != nil {
		return PiArgumentPlan{}, err
	}
	if err := validatePiStandalonePolicy(policy); err != nil {
		return PiArgumentPlan{}, err
	}
	const promptSlot = "agents-infra-standalone-prompt-slot"
	managed := []string{
		"--no-approve",
		"--no-extensions",
		"--tools", strings.Join(policy.ToolAllowlist.Value, ","),
		"--mode", "json",
		"--no-session",
		"--print",
		"--", promptSlot,
	}
	plan, err := BuildManagedPiArguments(managed, "", profile)
	if err != nil {
		return PiArgumentPlan{}, err
	}
	if len(plan.Argv) == 0 || len(plan.DiagnosticArgv) == 0 || plan.Argv[len(plan.Argv)-1] != promptSlot || plan.DiagnosticArgv[len(plan.DiagnosticArgv)-1] != promptSlot {
		return PiArgumentPlan{}, piError("pi_standalone_argument_invariant_failed", errors.New("standalone Pi prompt operand was not composed canonically"))
	}
	plan.Argv = append(plan.Argv[:len(plan.Argv)-1], "--", prompt)
	plan.DiagnosticArgv[len(plan.DiagnosticArgv)-1] = "<prompt>"
	return plan, nil
}

func piStandaloneToolAuthorization(policy PiStandaloneSessionPolicy) PiStandaloneToolAuthorization {
	return PiStandaloneToolAuthorization{
		Mode:                 "unattended_allowlist",
		Effective:            true,
		Scope:                "standalone_session",
		YoloSource:           policy.YoloMode.Source,
		AllowlistSource:      policy.ToolAllowlist.Source,
		AllowedTools:         append([]string(nil), policy.ToolAllowlist.Value...),
		NativeEnforcement:    "pi_cli_strict_allowlist",
		ExtensionDiscovery:   "disabled",
		ProjectTrust:         "declined",
		RPCDirectBash:        "not_exposed",
		PrivilegeBoundary:    "calling_user",
		ArgumentOwnership:    "agents-infra",
		HumanApprovalOrStdin: "not_required",
		TaskBoardAdapter:     "deferred_not_implemented",
	}
}

func resolvePiStandaloneSelection(composite compositeProjectConfig, request PiStandaloneRequest) (string, string, *PrimarySessionTarget, error) {
	if request.Entrypoint != "" {
		if err := validateComposedCanonicalTargets(composite); err != nil {
			return "", "", nil, err
		}
		resolved, err := resolveCanonicalTargetFromComposite(request.Entrypoint, composite)
		if err != nil {
			return "", "", nil, err
		}
		if resolved.Target.Environment != "pi" || resolved.Target.Profile == nil {
			return "", "", nil, &CanonicalTargetError{
				Code:        PrimarySessionErrorInvalidTarget,
				Context:     TargetErrorContext{Entrypoint: request.Entrypoint, Target: resolved.Target.Name, Field: targetsField + "." + resolved.Target.Name + ".environment", Source: resolved.Target.Source},
				Remediation: "use qwen-infra with an admitted qwen/pi canonical target",
				Err:         errors.New("standalone Pi requires a canonical Pi target"),
			}
		}
		selected := *resolved.Target.Profile
		if request.ExpectedProfile != "" && request.ExpectedProfile != selected {
			return "", "", nil, piError("pi_profile_mismatch", errors.New("standalone Pi profile assertion does not match the resolved profile"))
		}
		return selected, resolved.Target.Source, primarySessionTargetFromResolution(resolved), nil
	}
	if !composite.PiPrimarySession.Profile.Present {
		return "", "", nil, piError("unknown_pi_profile", errors.New("standalone pi-infra requires an effective managed primary Pi profile"))
	}
	selected := composite.PiPrimarySession.Profile.Value
	if request.ExpectedProfile != "" && request.ExpectedProfile != selected {
		return "", "", nil, piError("pi_profile_mismatch", errors.New("standalone Pi profile assertion does not match the resolved profile"))
	}
	return selected, composite.PiPrimarySession.Profile.Source, nil, nil
}

func BuildPiStandaloneLaunchPlan(projectDir, homeDir string, request PiStandaloneRequest, producer ChildLaunchCompositionProducer, lookPath func(string) (string, error)) (PiStandaloneLaunchPlan, error) {
	canonicalProject, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	if err := validatePiStandaloneRequest(request, nil); err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return PiStandaloneLaunchPlan{}, err
		}
	}
	composite, err := loadCompositeProjectConfig(ancestorDirsRootFirst(canonicalProject), filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName))
	if err != nil {
		return PiStandaloneLaunchPlan{}, piError("invalid_project_configuration", err)
	}
	if err := validatePiStandalonePolicy(composite.PiStandaloneSession); err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	profileName, profileSource, target, err := resolvePiStandaloneSelection(composite, request)
	if err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	if !composite.PiPrimarySession.PiCompatibility.Present {
		return PiStandaloneLaunchPlan{}, piError("invalid_project_configuration", errors.New("standalone managed Pi requires agents.pi.primary_session.pi_compatibility"))
	}
	profile, ok := composite.PiProfiles[profileName]
	if !ok {
		return PiStandaloneLaunchPlan{}, piError("unknown_pi_profile", errors.New("standalone Pi selected an unknown managed profile"))
	}
	if err := ValidatePiStateKeyCollisions(composite.PiProfiles); err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	argsPlan, err := BuildStandalonePiArguments(nil, profile, composite.PiStandaloneSession, request.Prompt)
	if err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	piPath, err := lookPath("pi")
	if err != nil {
		return PiStandaloneLaunchPlan{}, piError("provider_executable_not_found", err)
	}
	identity, err := VerifyPiExecutionIdentity(piPath, composite.PiPrimarySession.PiCompatibility.Value)
	if err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	sharing := "exclusive"
	ownership := "direct-child-process-group"
	if profile.Runtime.Sharing != nil {
		sharing = profile.Runtime.Sharing.Mode
		if sharing == "shared" {
			ownership = "shared-runtime-lease-broker"
		}
	}
	state, err := ResolvePiStatePaths("", canonicalProject, profileName)
	if err != nil {
		return PiStandaloneLaunchPlan{}, err
	}
	plan := PiStandaloneLaunchPlan{
		Contract:          PiStandaloneLaunchPlanContract,
		SchemaVersion:     PiStandaloneLaunchPlanSchemaVersion,
		Status:            "ok",
		Producer:          producer,
		ProjectDir:        canonicalProject,
		Entrypoint:        request.Entrypoint,
		Target:            target,
		Executable:        identity.Entrypoint,
		Argv:              argsPlan.DiagnosticArgv,
		Profile:           resolvedStringValue(profileName, profileSource),
		Model:             resolvedStringValue(profile.Provider+"/"+profile.Model, profile.Source),
		Reasoning:         resolvedStringValue(profile.Thinking, profile.Source),
		PiCompatibility:   resolvedStringValue(composite.PiPrimarySession.PiCompatibility.Value, composite.PiPrimarySession.PiCompatibility.Source),
		ToolAuthorization: piStandaloneToolAuthorization(composite.PiStandaloneSession),
		State:             PiStandaloneStatePlan{Isolation: "per-process-random-run-key", Persistence: "disabled"},
		RuntimeMode:       sharing,
		LifecycleLogs:     PiLifecycleLogPlan{PolicySource: profile.Source, AggregateRoot: state.LifecycleLogsRoot, Policy: profile.LifecycleLogRetention, Status: "not-inspected"},
		Runtime: PiRuntimePlan{
			Executable:             profile.Runtime.Executable,
			Argv:                   append([]string(nil), profile.Runtime.Argv...),
			Source:                 profile.Source,
			Endpoint:               profile.BaseURL,
			ReadinessURL:           profile.BaseURL + profile.Runtime.ReadinessPath,
			StartupTimeoutSeconds:  profile.Runtime.StartupTimeoutSeconds,
			ShutdownTimeoutSeconds: profile.Runtime.ShutdownTimeoutSeconds,
			ExecutableState:        staticExecutableState(profile.Runtime.Executable),
			Ownership:              ownership,
		},
		PiIdentity: identity,
	}
	for _, source := range composite.Sources {
		plan.Sources = append(plan.Sources, PrimarySessionSource{Kind: "project_config", Path: source.Path})
	}
	return plan, nil
}

func newPiStandaloneRunID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", piError("pi_standalone_state_identity_failed", err)
	}
	return "standalone-" + hex.EncodeToString(random), nil
}
