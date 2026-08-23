package infra

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	PrimarySessionLaunchPlanContract      = "agents-infra.primary-session-launch-plan"
	PrimarySessionLaunchPlanSchemaVersion = 1

	PrimarySessionManagedHostKindCodexAppServer = "codex-app-server"
	PrimarySessionManagedHostKindClaudePTY      = "claude-pty"

	PrimarySessionErrorUnsupportedSchemaVersion    = "unsupported_schema_version"
	PrimarySessionErrorInvalidProjectConfiguration = "invalid_project_configuration"
	PrimarySessionErrorInvalidProviderArguments    = "invalid_provider_arguments"
	PrimarySessionErrorProviderExecutableNotFound  = "provider_executable_not_found"
)

// PrimarySessionLaunchPlan is the non-launching serialization of exactly the
// launch plan `agents-infra codex` or `agents-infra claude` would execute for
// a primary session: same policy precedence, same executable resolution, same
// argument ordering. An external session manager selects a launch variant and
// owns the resulting process; agents-infra performs no launch and stays
// board-agnostic.
type PrimarySessionLaunchPlan struct {
	Contract           string                         `json:"contract"`
	SchemaVersion      int                            `json:"schema_version"`
	Status             string                         `json:"status"`
	Producer           ChildLaunchCompositionProducer `json:"producer"`
	Provider           string                         `json:"provider"`
	Target             *PrimarySessionTarget          `json:"target,omitempty"`
	ProjectDir         string                         `json:"project_dir"`
	Executable         string                         `json:"executable"`
	LaunchVariants     PrimarySessionLaunchVariants   `json:"launch_variants"`
	Resolved           PrimarySessionResolved         `json:"resolved"`
	RequiredEnvNames   []string                       `json:"required_env_names"`
	Sources            []PrimarySessionSource         `json:"sources"`
	Pi                 *PiLaunchPlanDetails           `json:"pi,omitempty"`
	Sidecars           *PrimarySessionSidecars        `json:"sidecars,omitempty"`
	Capabilities       *PiCapabilityPlan              `json:"capabilities,omitempty"`
	targetProviderArgs []string
}

func (p PrimarySessionLaunchPlan) TargetProviderArgs() []string {
	return append([]string(nil), p.targetProviderArgs...)
}

type PrimarySessionTarget struct {
	Entrypoint       string  `json:"entrypoint"`
	EntrypointSource string  `json:"entrypoint_source"`
	Name             string  `json:"name"`
	Source           string  `json:"source"`
	Vendor           string  `json:"vendor"`
	Environment      string  `json:"environment"`
	Model            string  `json:"model"`
	Reasoning        string  `json:"reasoning"`
	Profile          *string `json:"profile"`
	ProfileProvider  *string `json:"profile_provider"`
	Endpoint         *string `json:"endpoint"`
}

type PrimarySessionSidecars struct {
	LocalModel PiRuntimePlan `json:"local_model"`
}

type PrimarySessionLaunchVariants struct {
	Interactive   PrimarySessionInteractiveVariant   `json:"interactive"`
	ManagedHost   PrimarySessionManagedHostVariant   `json:"managed_host"`
	ManagedClient PrimarySessionManagedClientVariant `json:"managed_client"`
}

// PrimarySessionInteractiveVariant carries the exact argv the launching
// wrapper would pass to the provider executable for a terminal session.
type PrimarySessionInteractiveVariant struct {
	Argv []string `json:"argv"`
}

// PrimarySessionManagedHostVariant carries the argv for a manager-owned
// provider host process. For codex-app-server the argv is derived from the
// same normalized interactive argv the launcher would exec: every
// config-level global option class (arbitrary -c/--config overrides,
// --enable, --disable, --strict-config, --profile, --oss, --local-provider,
// --search, --dangerously-bypass-hook-trust) keeps its relative order,
// --model converts to its -c model override, and the manager appends
// `--listen <url>` at launch time. Session policy stays out of this argv —
// the interactive bypass flag, --sandbox/--ask-for-approval selections, and
// -c sandbox_mode=/approval_policy= overrides are reflected only in the
// resolved block and applied per thread through the app-server RPC. For
// claude-pty the argv is the full interactive composition run under a
// manager-owned PTY.
type PrimarySessionManagedHostVariant struct {
	Kind string   `json:"kind"`
	Argv []string `json:"argv"`
}

// PrimarySessionManagedClientVariant carries every interactive-argv token
// that is not host argv and not session policy, in interactive order: thread
// and client semantics such as -C/--cd, --add-dir, -i/--image, --no-alt-screen,
// --remote/--remote-auth-token-env, subcommands with their flags, prompt text,
// and everything after `--`. The session manager applies these on its client
// or thread layer (for example thread cwd, writable roots, initial prompt,
// session selection) and must fail closed on any token it cannot represent
// rather than silently ignoring it. The three-way split is total: every
// interactive token appears in managed_host.argv, in managed_client.argv, or
// as a session-policy field in the resolved block, so nothing is silently
// dropped on the managed path. For claude-pty the whole interactive argv runs
// in the managed host PTY, so this fragment is always empty.
type PrimarySessionManagedClientVariant struct {
	Argv []string `json:"argv"`
}

type PrimarySessionResolved struct {
	Model           PrimarySessionResolvedString  `json:"model"`
	Reasoning       PrimarySessionResolvedString  `json:"reasoning"`
	Yolo            PrimarySessionResolvedBool    `json:"yolo"`
	Sandbox         PrimarySessionResolvedString  `json:"sandbox"`
	Profile         PrimarySessionResolvedString  `json:"profile"`
	Approval        PrimarySessionResolvedString  `json:"approval"`
	MCP             PrimarySessionResolvedMCP     `json:"mcp"`
	PiCompatibility PrimarySessionResolvedString  `json:"pi_compatibility,omitempty"`
	ProfileProvider *PrimarySessionResolvedString `json:"profile_provider,omitempty"`
	Endpoint        *PrimarySessionResolvedString `json:"endpoint,omitempty"`
}

// PrimarySessionResolvedString reports one resolved policy field with
// provenance. Value is null when the provider's native configuration decides
// (source "native"), when the field does not exist for the provider (source
// "not_applicable"), or when an explicit CLI selection suppressed the
// composed value without stating a new one.
type PrimarySessionResolvedString struct {
	Value  *string `json:"value"`
	Source string  `json:"source"`
}

type PrimarySessionResolvedBool struct {
	Value  bool   `json:"value"`
	Source string `json:"source"`
}

type PrimarySessionResolvedMCP struct {
	Servers []ChildLaunchCompositionMCPServer `json:"servers"`
	Sources []string                          `json:"sources"`
}

type PrimarySessionSource struct {
	Kind           string   `json:"kind"`
	Path           string   `json:"path"`
	Scope          string   `json:"scope,omitempty"`
	EnabledServers []string `json:"enabled_servers,omitempty"`
}

type PrimarySessionLaunchPlanErrorEnvelope struct {
	Contract      string                         `json:"contract"`
	SchemaVersion int                            `json:"schema_version"`
	Status        string                         `json:"status"`
	Producer      ChildLaunchCompositionProducer `json:"producer"`
	Provider      string                         `json:"provider"`
	ProjectDir    string                         `json:"project_dir"`
	Error         PrimarySessionLaunchPlanError  `json:"error"`
}

type PrimarySessionLaunchPlanError struct {
	Code        string              `json:"code"`
	Context     *TargetErrorContext `json:"context,omitempty"`
	Remediation string              `json:"remediation,omitempty"`
}

func NewPrimarySessionLaunchPlanErrorEnvelope(provider, projectDir string, producer ChildLaunchCompositionProducer, code string) PrimarySessionLaunchPlanErrorEnvelope {
	return PrimarySessionLaunchPlanErrorEnvelope{
		Contract:      PrimarySessionLaunchPlanContract,
		SchemaVersion: PrimarySessionLaunchPlanSchemaVersion,
		Status:        "error",
		Producer:      producer,
		Provider:      provider,
		ProjectDir:    projectDir,
		Error:         PrimarySessionLaunchPlanError{Code: code},
	}
}

func NewCanonicalTargetLaunchPlanErrorEnvelope(provider, projectDir string, producer ChildLaunchCompositionProducer, targetErr *CanonicalTargetError) PrimarySessionLaunchPlanErrorEnvelope {
	envelope := NewPrimarySessionLaunchPlanErrorEnvelope(provider, projectDir, producer, targetErr.Code)
	context := targetErr.Context
	envelope.Error.Context = &context
	envelope.Error.Remediation = targetErr.Remediation
	return envelope
}

// PrimarySessionComposeError carries the machine error code for the JSON
// error envelope alongside the human diagnostic.
type PrimarySessionComposeError struct {
	Code string
	Err  error
}

func (e *PrimarySessionComposeError) Error() string { return e.Err.Error() }

func (e *PrimarySessionComposeError) Unwrap() error { return e.Err }

// BuildPrimarySessionLaunchPlan builds the primary-session contract for one
// provider without launching anything. lookPath resolves the provider
// executable and defaults to exec.LookPath; tests inject a fake.
func BuildPrimarySessionLaunchPlan(provider, projectDir, homeDir string, userArgs []string, producer ChildLaunchCompositionProducer, lookPath func(string) (string, error)) (PrimarySessionLaunchPlan, error) {
	if provider != "codex" && provider != "claude" && provider != "pi" {
		return PrimarySessionLaunchPlan{}, &PrimarySessionComposeError{
			Code: PrimarySessionErrorInvalidProjectConfiguration,
			Err:  fmt.Errorf("unsupported provider %q", provider),
		}
	}
	canonicalProjectDir, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return PrimarySessionLaunchPlan{}, &PrimarySessionComposeError{
			Code: PrimarySessionErrorInvalidProjectConfiguration,
			Err:  err,
		}
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	executable := ""
	if provider != "pi" {
		executable, err = lookPath(provider)
		if err != nil {
			return PrimarySessionLaunchPlan{}, &PrimarySessionComposeError{
				Code: PrimarySessionErrorProviderExecutableNotFound,
				Err:  fmt.Errorf("find %s executable: %w", provider, err),
			}
		}
	}

	result := PrimarySessionLaunchPlan{
		Contract:         PrimarySessionLaunchPlanContract,
		SchemaVersion:    PrimarySessionLaunchPlanSchemaVersion,
		Status:           "ok",
		Producer:         producer,
		Provider:         provider,
		ProjectDir:       canonicalProjectDir,
		Executable:       executable,
		RequiredEnvNames: []string{},
		Sources:          []PrimarySessionSource{},
	}
	result.Resolved.MCP.Servers = []ChildLaunchCompositionMCPServer{}
	result.Resolved.MCP.Sources = []string{}

	switch provider {
	case "codex":
		err = buildCodexPrimarySessionLaunchPlan(&result, canonicalProjectDir, homeDir, userArgs)
	case "claude":
		err = buildClaudePrimarySessionLaunchPlan(&result, canonicalProjectDir, homeDir, userArgs)
	case "pi":
		err = buildPiPrimarySessionLaunchPlan(&result, canonicalProjectDir, homeDir, userArgs, lookPath)
	}
	if err != nil {
		code := PrimarySessionErrorInvalidProjectConfiguration
		var argErr *ProviderArgumentError
		if errors.As(err, &argErr) {
			code = PrimarySessionErrorInvalidProviderArguments
		}
		var piErr *PiLaunchError
		if errors.As(err, &piErr) {
			code = piErr.Code
		}
		return PrimarySessionLaunchPlan{}, &PrimarySessionComposeError{
			Code: code,
			Err:  err,
		}
	}
	return result, nil
}

func buildCodexPrimarySessionLaunchPlan(result *PrimarySessionLaunchPlan, projectDir, homeDir string, userArgs []string) error {
	plan, err := BuildCodexLaunchPlan(projectDir, homeDir, userArgs)
	if err != nil {
		return err
	}

	result.LaunchVariants.Interactive.Argv = emptyIfNil(plan.Args)
	hostArgv, clientArgv := codexManagedArgvSplit(plan.Args)
	result.LaunchVariants.ManagedHost = PrimarySessionManagedHostVariant{
		Kind: PrimarySessionManagedHostKindCodexAppServer,
		Argv: hostArgv,
	}
	result.LaunchVariants.ManagedClient.Argv = clientArgv

	result.Resolved.Model = resolvedStringFromCodex(plan.PrimarySessionResolution.Model)
	result.Resolved.Reasoning = resolvedStringFromCodex(plan.PrimarySessionResolution.ReasoningEffort)
	yolo := plan.PrimarySessionResolution.YoloMode
	result.Resolved.Yolo = PrimarySessionResolvedBool{Value: yolo.EffectiveValue, Source: yolo.EffectiveSource}
	switch {
	case yolo.EffectiveValue:
		result.Resolved.Sandbox = resolvedStringValue("danger-full-access", yolo.EffectiveSource)
		result.Resolved.Approval = resolvedStringValue("never", yolo.EffectiveSource)
	default:
		if plan.ExplicitSandbox {
			result.Resolved.Sandbox = resolvedStringValue(plan.ExplicitSandboxValue, plan.ExplicitSandboxSource)
		} else {
			result.Resolved.Sandbox = PrimarySessionResolvedString{Source: "native"}
		}
		if plan.ExplicitApproval {
			result.Resolved.Approval = resolvedStringValue(plan.ExplicitApprovalValue, plan.ExplicitApprovalSource)
		} else {
			result.Resolved.Approval = PrimarySessionResolvedString{Source: "native"}
		}
	}
	if plan.ExplicitProfile {
		result.Resolved.Profile = resolvedStringValue(plan.ExplicitProfileValue, plan.ExplicitProfileSource)
	} else {
		result.Resolved.Profile = PrimarySessionResolvedString{Source: "native"}
	}

	for _, source := range plan.ProjectConfigs {
		result.Sources = append(result.Sources, PrimarySessionSource{
			Kind:           "project-config",
			Path:           source.Path,
			EnabledServers: append([]string(nil), source.EnabledServers...),
		})
	}
	appendPrimarySessionRegistrySources(result, plan.RegistrySources)
	for _, server := range plan.MCPServers {
		appendPrimarySessionMCPServer(result, ChildLaunchCompositionMCPServer{
			Name:              server.Name,
			Transport:         mcpTransport(server.URL),
			DefinitionSource:  server.DefinitionSource,
			EnabledBy:         append([]string(nil), server.EnabledBy...),
			BearerTokenEnvVar: server.BearerTokenEnvVar,
		})
	}
	// A valid --remote-auth-token-env selection names the environment variable
	// holding the remote bearer token; the contract carries the name only,
	// after the MCP bearer-token names and de-duplicated against them. The
	// wrapper parser recorded it before any `--`, so pass-through prompt text
	// is never interpreted as an environment reference, and Codex's accepted
	// empty name contributes no requirement.
	if plan.RemoteAuthTokenEnvName != "" {
		appendPrimarySessionRequiredEnvName(result, plan.RemoteAuthTokenEnvName)
	}
	return nil
}

// codexManagedArgvSplit derives the managed codex app-server host argv and
// the managed client argv from the same normalized interactive argv the
// launcher would exec. The classification is total — every interactive token
// is routed to exactly one destination, so nothing is silently dropped on the
// managed path:
//
//   - -c/--config overrides, --enable, --disable, --strict-config, --profile,
//     --oss, --local-provider, --search, --dangerously-bypass-hook-trust:
//     config-level globals the app-server host process consumes, kept in
//     their interactive order;
//   - --model (spaced, =, and attached -mVALUE): converted to its -c model
//     override because app-server has no --model flag;
//   - the interactive bypass flag, --sandbox/--ask-for-approval selections,
//     and -c sandbox_mode=/approval_policy= overrides: session policy the
//     manager applies per thread over the app-server RPC from the resolved
//     block, never argv;
//   - every other token (thread/client options such as -C/--cd, --add-dir,
//     -i/--image, --no-alt-screen, --remote, subcommands with their flags,
//     prompt text, and everything after --): managed client argv, in order.
//     Routing unrecognized tokens to the client fragment instead of an
//     allow-list keeps future provider flags visible to the consumer, which
//     must fail closed on tokens it cannot represent.
//
// The wrapper parser has already failed closed on a recognized value-taking
// option with a missing or flag-like value and on a repeated profile
// selection, so every value-taking global reaching this split carries its
// value; the value-absent branches below are defensive only.
func codexManagedArgvSplit(interactive []string) (host, client []string) {
	client = []string{}
	for index := 0; index < len(interactive); index++ {
		arg := interactive[index]
		switch {
		case arg == "--":
			client = append(client, interactive[index:]...)
			return append(host, "app-server"), client
		case arg == codexDangerouslyBypassApprovalsAndSandbox:
		case arg == "--model" || arg == "-m":
			if index+1 < len(interactive) {
				index++
				host = append(host, "-c", fmt.Sprintf("model=%s", strconv.Quote(interactive[index])))
			}
		case strings.HasPrefix(arg, "--model=") || strings.HasPrefix(arg, "-m="):
			_, value, _ := strings.Cut(arg, "=")
			host = append(host, "-c", fmt.Sprintf("model=%s", strconv.Quote(value)))
		case strings.HasPrefix(arg, "-m") && len(arg) > 2:
			host = append(host, "-c", fmt.Sprintf("model=%s", strconv.Quote(strings.TrimPrefix(arg, "-m"))))
		case arg == "-c" || arg == "--config":
			if index+1 >= len(interactive) {
				host = append(host, arg)
				continue
			}
			index++
			if !codexPolicyConfigOverride(interactive[index]) {
				host = append(host, arg, interactive[index])
			}
		case strings.HasPrefix(arg, "-c="):
			if !codexPolicyConfigOverride(strings.TrimPrefix(arg, "-c=")) {
				host = append(host, arg)
			}
		case strings.HasPrefix(arg, "--config="):
			if !codexPolicyConfigOverride(strings.TrimPrefix(arg, "--config=")) {
				host = append(host, arg)
			}
		case arg == "--profile" || arg == "-p":
			if index+1 < len(interactive) {
				client = append(client, arg, interactive[index+1])
				index++
			} else {
				client = append(client, arg)
			}
		case strings.HasPrefix(arg, "--profile=") || strings.HasPrefix(arg, "-p="),
			strings.HasPrefix(arg, "-p") && len(arg) > 2:
			client = append(client, arg)
		case arg == "--enable" || arg == "--disable" || arg == "--local-provider":
			host = append(host, arg)
			if index+1 < len(interactive) {
				index++
				host = append(host, interactive[index])
			}
		case strings.HasPrefix(arg, "--enable=") || strings.HasPrefix(arg, "--disable="),
			strings.HasPrefix(arg, "--local-provider="),
			arg == "--strict-config", arg == "--oss", arg == "--search",
			arg == "--dangerously-bypass-hook-trust":
			host = append(host, arg)
		case arg == "--sandbox" || arg == "-s" || arg == "--ask-for-approval" || arg == "-a":
			// The wrapper parser fails closed on a missing or flag-like
			// policy value, so a spaced value token always follows here.
			index++
		case strings.HasPrefix(arg, "--sandbox=") || strings.HasPrefix(arg, "--ask-for-approval="),
			strings.HasPrefix(arg, "-s") && len(arg) > 2,
			strings.HasPrefix(arg, "-a") && len(arg) > 2:
			// =-joined or attached policy selection, mirroring the wrapper
			// parser's classification: session policy via RPC.
		default:
			client = append(client, arg)
		}
	}
	return append(host, "app-server"), client
}

// codexPolicyConfigOverride reports whether a -c/--config override selects the
// per-thread sandbox or approval policy that the session manager applies over
// the app-server RPC instead of through host argv.
func codexPolicyConfigOverride(value string) bool {
	key, _, ok := strings.Cut(value, "=")
	if !ok {
		return false
	}
	switch strings.TrimSpace(key) {
	case "sandbox_mode", "approval_policy":
		return true
	}
	return false
}

func buildClaudePrimarySessionLaunchPlan(result *PrimarySessionLaunchPlan, projectDir, homeDir string, userArgs []string) error {
	plan, err := BuildClaudeLaunchPlan(projectDir, homeDir, userArgs)
	if err != nil {
		return err
	}

	result.LaunchVariants.Interactive.Argv = emptyIfNil(plan.Args)
	result.LaunchVariants.ManagedHost = PrimarySessionManagedHostVariant{
		Kind: PrimarySessionManagedHostKindClaudePTY,
		Argv: emptyIfNil(plan.Args),
	}
	// The whole interactive composition runs inside the manager-owned PTY, so
	// no separate client fragment exists for Claude.
	result.LaunchVariants.ManagedClient.Argv = []string{}

	model := plan.PrimarySessionResolution.Model
	if model.EffectiveValueKnown {
		result.Resolved.Model = resolvedStringValue(model.EffectiveValue, model.EffectiveSource)
	} else {
		result.Resolved.Model = PrimarySessionResolvedString{Source: model.EffectiveSource}
	}
	if plan.ExplicitEffort && plan.ExplicitEffortRecognized {
		result.Resolved.Reasoning = resolvedStringValue(plan.ExplicitEffortValue, plan.ExplicitEffortSource)
	} else {
		// Either no --effort was passed, or its token is one Claude ignores
		// with a warning while applying its own default (probe-verified); in
		// both cases the provider's native configuration decides, so no
		// effective value is claimed.
		result.Resolved.Reasoning = PrimarySessionResolvedString{Source: "native"}
	}
	yolo := plan.PrimarySessionResolution.YoloMode
	result.Resolved.Yolo = PrimarySessionResolvedBool{Value: yolo.EffectiveValue, Source: yolo.EffectiveSource}
	result.Resolved.Sandbox = PrimarySessionResolvedString{Source: "not_applicable"}
	result.Resolved.Profile = PrimarySessionResolvedString{Source: "not_applicable"}
	switch {
	case yolo.EffectiveValue:
		// --dangerously-skip-permissions overrides an explicit
		// --permission-mode at Claude runtime (probe-verified), so an
		// effective yolo always resolves the approval policy.
		result.Resolved.Approval = resolvedStringValue("bypass-permissions", yolo.EffectiveSource)
	case plan.ExplicitPermissionMode:
		result.Resolved.Approval = resolvedStringValue(plan.ExplicitPermissionValue, plan.ExplicitPermissionSource)
	default:
		result.Resolved.Approval = PrimarySessionResolvedString{Source: "native"}
	}

	for _, source := range plan.ProjectConfigs {
		result.Sources = append(result.Sources, PrimarySessionSource{
			Kind:           "project-config",
			Path:           source.Path,
			EnabledServers: append([]string(nil), source.EnabledServers...),
		})
	}
	appendPrimarySessionRegistrySources(result, plan.RegistrySources)
	for _, server := range plan.MCPServers {
		appendPrimarySessionMCPServer(result, ChildLaunchCompositionMCPServer{
			Name:              server.Name,
			Transport:         mcpTransport(server.URL),
			DefinitionSource:  server.DefinitionSource,
			EnabledBy:         append([]string(nil), server.EnabledBy...),
			BearerTokenEnvVar: server.BearerTokenEnvVar,
		})
	}
	return nil
}

func appendPrimarySessionRegistrySources(result *PrimarySessionLaunchPlan, registries []CodexMCPRegistrySource) {
	for _, source := range registries {
		result.Sources = append(result.Sources, PrimarySessionSource{
			Kind:  "mcp-registry",
			Path:  source.Path,
			Scope: source.Scope,
		})
		result.Resolved.MCP.Sources = append(result.Resolved.MCP.Sources, source.Path)
	}
}

func appendPrimarySessionMCPServer(result *PrimarySessionLaunchPlan, server ChildLaunchCompositionMCPServer) {
	result.Resolved.MCP.Servers = append(result.Resolved.MCP.Servers, server)
	if server.BearerTokenEnvVar != "" {
		appendPrimarySessionRequiredEnvName(result, server.BearerTokenEnvVar)
	}
}

func appendPrimarySessionRequiredEnvName(result *PrimarySessionLaunchPlan, name string) {
	if !containsString(result.RequiredEnvNames, name) {
		result.RequiredEnvNames = append(result.RequiredEnvNames, name)
	}
}

func resolvedStringFromCodex(resolution CodexPrimarySessionStringResolution) PrimarySessionResolvedString {
	if resolution.EffectiveValueKnown {
		return resolvedStringValue(resolution.EffectiveValue, resolution.EffectiveSource)
	}
	return PrimarySessionResolvedString{Source: resolution.EffectiveSource}
}

func resolvedStringValue(value, source string) PrimarySessionResolvedString {
	return PrimarySessionResolvedString{Value: &value, Source: source}
}

func mcpTransport(url string) string {
	if url != "" {
		return "http"
	}
	return "stdio"
}

func emptyIfNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
