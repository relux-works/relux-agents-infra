package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const codexDangerouslyBypassApprovalsAndSandbox = "--dangerously-bypass-approvals-and-sandbox"

type CodexLaunchPlan struct {
	StartDir                 string
	HomeDir                  string
	BaseCodexConfigPath      string
	BaseCodexConfigPresent   bool
	ProjectConfigs           []CodexProjectConfigSource
	RegistrySources          []CodexMCPRegistrySource
	MCPServers               []CodexMCPLaunchServer
	PrimarySession           CodexPrimarySessionPolicy
	PrimarySessionResolution CodexPrimarySessionResolution
	ConfigArgs               []string
	UserArgs                 []string
	Args                     []string
	PrintConfig              bool
	WrapperExpandedShortcuts []CodexWrapperShortcut
	ExplicitProfile          bool
	ExplicitProfileValue     string
	ExplicitProfileSource    string
	RemoteAuthTokenEnvName   string
	ExplicitSandbox          bool
	ExplicitSandboxValue     string
	ExplicitSandboxSource    string
	ExplicitApproval         bool
	ExplicitApprovalValue    string
	ExplicitApprovalSource   string
}

type CodexPrimarySessionApplication string

const (
	CodexPrimarySessionNotConfigured       CodexPrimarySessionApplication = "not_configured"
	CodexPrimarySessionApplied             CodexPrimarySessionApplication = "applied"
	CodexPrimarySessionSuppressedByCLI     CodexPrimarySessionApplication = "suppressed_by_explicit_cli"
	CodexPrimarySessionSuppressedByProfile CodexPrimarySessionApplication = "suppressed_by_explicit_profile"
)

// CodexPrimarySessionResolution records the invocation-level primary-session
// decision. ProjectValue and ProjectSource preserve the composed project
// policy even when an explicit CLI value or profile suppresses its application.
type CodexPrimarySessionResolution struct {
	Model           CodexPrimarySessionStringResolution
	ReasoningEffort CodexPrimarySessionStringResolution
	YoloMode        CodexPrimarySessionBoolResolution
}

type CodexPrimarySessionStringResolution struct {
	EffectiveValue      string
	EffectiveValueKnown bool
	EffectiveSource     string
	ProjectConfigured   bool
	ProjectValue        string
	ProjectSource       string
	ProjectApplication  CodexPrimarySessionApplication
}

type CodexPrimarySessionBoolResolution struct {
	EffectiveValue     bool
	EffectiveSource    string
	ProjectConfigured  bool
	ProjectValue       bool
	ProjectSource      string
	ProjectApplication CodexPrimarySessionApplication
}

type CodexProjectConfigSource struct {
	Path           string
	EnabledServers []string
	PrimarySession CodexPrimarySessionSource
}

type CodexMCPRegistrySource struct {
	Path        string
	Scope       string
	ServerNames []string
}

type CodexMCPLaunchServer struct {
	Name              string
	URL               string
	BearerTokenEnvVar string
	Command           string
	Args              []string
	DefinitionSource  string
	EnabledBy         []string
}

type CodexWrapperShortcut struct {
	From string
	To   string
}

type codexMCPDefinition struct {
	Server codexMCPServer
	Source string
}

func BuildCodexLaunchPlan(startDir, homeDir string, args []string) (CodexLaunchPlan, error) {
	parsed, err := parseCodexWrapperArgs(args)
	if err != nil {
		return CodexLaunchPlan{}, err
	}
	if startDir == "" {
		startDir, err = os.Getwd()
		if err != nil {
			return CodexLaunchPlan{}, fmt.Errorf("resolve cwd: %w", err)
		}
	}
	startDir, err = filepath.Abs(startDir)
	if err != nil {
		return CodexLaunchPlan{}, fmt.Errorf("resolve start dir: %w", err)
	}
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return CodexLaunchPlan{}, fmt.Errorf("resolve home dir: %w", err)
		}
	}
	homeDir, err = filepath.Abs(homeDir)
	if err != nil {
		return CodexLaunchPlan{}, fmt.Errorf("resolve home dir: %w", err)
	}

	plan := CodexLaunchPlan{
		StartDir:                 startDir,
		HomeDir:                  homeDir,
		BaseCodexConfigPath:      filepath.Join(homeDir, ".codex", "config.toml"),
		BaseCodexConfigPresent:   pathExists(filepath.Join(homeDir, ".codex", "config.toml")),
		UserArgs:                 parsed.codexArgs,
		PrintConfig:              parsed.printConfig,
		WrapperExpandedShortcuts: parsed.expandedShortcuts,
	}

	ancestors := ancestorDirsRootFirst(startDir)
	globalProjectConfigPath := filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName)
	projectConfig, err := loadCompositeProjectConfig(ancestors, globalProjectConfigPath)
	if err != nil {
		return CodexLaunchPlan{}, err
	}
	for _, source := range projectConfig.Sources {
		plan.ProjectConfigs = append(plan.ProjectConfigs, CodexProjectConfigSource{
			Path:           source.Path,
			EnabledServers: append([]string(nil), source.EnabledServers...),
			PrimarySession: cloneCodexPrimarySessionSource(source.CodexPrimarySession),
		})
	}
	plan.PrimarySession = projectConfig.PrimarySession

	definitions, registrySources, err := loadCompositeMCPRegistry(homeDir, ancestors)
	if err != nil {
		return CodexLaunchPlan{}, err
	}
	plan.RegistrySources = registrySources

	for _, name := range projectConfig.EnabledOrder {
		def, ok := definitions[name]
		if !ok {
			return CodexLaunchPlan{}, fmt.Errorf("MCP server %q is enabled by %s but no definition was found in codex-mcp-servers.toml registries", name, strings.Join(projectConfig.EnabledBy[name], ", "))
		}
		if err := validateCodexMCPDefinition(name, def); err != nil {
			return CodexLaunchPlan{}, err
		}
		server := CodexMCPLaunchServer{
			Name:              name,
			URL:               def.Server.URL,
			BearerTokenEnvVar: def.Server.BearerTokenEnvVar,
			Command:           def.Server.Command,
			Args:              append([]string(nil), def.Server.Args...),
			DefinitionSource:  def.Source,
			EnabledBy:         append([]string(nil), projectConfig.EnabledBy[name]...),
		}
		plan.MCPServers = append(plan.MCPServers, server)
		plan.ConfigArgs = append(plan.ConfigArgs, codexMCPConfigArgs(server)...)
	}
	primaryResolution, primaryArgs := resolveCodexPrimarySession(plan.PrimarySession, parsed)
	plan.PrimarySessionResolution = primaryResolution
	plan.ConfigArgs = append(plan.ConfigArgs, primaryArgs...)
	plan.Args = append(append([]string(nil), plan.ConfigArgs...), plan.UserArgs...)
	plan.ExplicitProfile = parsed.explicit.profile
	plan.ExplicitProfileValue = parsed.explicit.profileValue
	plan.ExplicitProfileSource = parsed.explicit.profileSource
	plan.RemoteAuthTokenEnvName = parsed.explicit.remoteAuthTokenEnvName
	if selection := parsed.explicit.sandbox; selection != nil {
		plan.ExplicitSandbox = true
		plan.ExplicitSandboxValue = selection.value
		plan.ExplicitSandboxSource = selection.source
	}
	if selection := parsed.explicit.approval; selection != nil {
		plan.ExplicitApproval = true
		plan.ExplicitApprovalValue = selection.value
		plan.ExplicitApprovalSource = selection.source
	}
	return plan, nil
}

func validateCodexMCPDefinition(name string, def codexMCPDefinition) error {
	hasURL := def.Server.URL != ""
	hasCommand := def.Server.Command != ""
	switch {
	case !hasURL && !hasCommand:
		return fmt.Errorf("MCP server %q is defined by %s but is missing url or command", name, def.Source)
	case hasURL && hasCommand:
		return fmt.Errorf("MCP server %q is defined by %s with both url and command", name, def.Source)
	}
	if !hasURL && def.Server.BearerTokenEnvVar != "" {
		return fmt.Errorf("MCP server %q is defined by %s with bearer_token_env_var but no url", name, def.Source)
	}
	if !hasCommand && len(def.Server.Args) > 0 {
		return fmt.Errorf("MCP server %q is defined by %s with args but no command", name, def.Source)
	}
	return nil
}

func codexMCPConfigArgs(server CodexMCPLaunchServer) []string {
	if server.URL != "" {
		args := []string{"-c", fmt.Sprintf("mcp_servers.%s.url=%q", server.Name, server.URL)}
		if server.BearerTokenEnvVar != "" {
			args = append(args, "-c", fmt.Sprintf("mcp_servers.%s.bearer_token_env_var=%q", server.Name, server.BearerTokenEnvVar))
		}
		return args
	}

	args := []string{"-c", fmt.Sprintf("mcp_servers.%s.command=%q", server.Name, server.Command)}
	if len(server.Args) > 0 {
		args = append(args, "-c", fmt.Sprintf("mcp_servers.%s.args=%s", server.Name, formatTOMLStringArray(server.Args)))
	}
	return args
}

func RenderCodexLaunchPlan(plan CodexLaunchPlan) string {
	var out strings.Builder
	out.WriteString("agents-infra codex config\n")
	fmt.Fprintf(&out, "cwd: %s\n", plan.StartDir)
	if plan.BaseCodexConfigPresent {
		fmt.Fprintf(&out, "base_codex_config: %s\n", plan.BaseCodexConfigPath)
	} else {
		fmt.Fprintf(&out, "base_codex_config: %s (missing)\n", plan.BaseCodexConfigPath)
	}

	out.WriteString("project_configs:\n")
	if len(plan.ProjectConfigs) == 0 {
		out.WriteString("  - (none)\n")
	} else {
		for _, source := range plan.ProjectConfigs {
			if len(source.EnabledServers) == 0 {
				fmt.Fprintf(&out, "  - %s: enabled_mcp=(none)\n", source.Path)
			} else {
				fmt.Fprintf(&out, "  - %s: enabled_mcp=%s\n", source.Path, strings.Join(source.EnabledServers, ","))
			}
		}
	}

	renderCodexPrimarySessionResolution(&out, plan.PrimarySessionResolution)

	out.WriteString("mcp_registries:\n")
	if len(plan.RegistrySources) == 0 {
		out.WriteString("  - (none)\n")
	} else {
		for _, source := range plan.RegistrySources {
			if len(source.ServerNames) == 0 {
				fmt.Fprintf(&out, "  - %s (%s): servers=(none)\n", source.Path, source.Scope)
			} else {
				fmt.Fprintf(&out, "  - %s (%s): servers=%s\n", source.Path, source.Scope, strings.Join(source.ServerNames, ","))
			}
		}
	}

	out.WriteString("enabled_mcp:\n")
	if len(plan.MCPServers) == 0 {
		out.WriteString("  - (none)\n")
	} else {
		for _, server := range plan.MCPServers {
			fmt.Fprintf(&out, "  - %s\n", server.Name)
			fmt.Fprintf(&out, "    enabled_by: %s\n", strings.Join(server.EnabledBy, ", "))
			fmt.Fprintf(&out, "    definition: %s\n", server.DefinitionSource)
			if server.URL != "" {
				fmt.Fprintf(&out, "    url: %s\n", server.URL)
				if server.BearerTokenEnvVar != "" {
					fmt.Fprintf(&out, "    bearer_token_env_var: %s\n", server.BearerTokenEnvVar)
				}
			} else {
				fmt.Fprintf(&out, "    command: %s\n", server.Command)
				if len(server.Args) > 0 {
					fmt.Fprintf(&out, "    args: %s\n", formatTOMLStringArray(server.Args))
				}
			}
		}
	}

	out.WriteString("wrapper_expansions:\n")
	if len(plan.WrapperExpandedShortcuts) == 0 {
		out.WriteString("  - (none)\n")
	} else {
		for _, shortcut := range plan.WrapperExpandedShortcuts {
			fmt.Fprintf(&out, "  - %s => %s\n", shortcut.From, shortcut.To)
		}
	}

	out.WriteString("codex_args:\n")
	if len(plan.Args) == 0 {
		out.WriteString("  - (none)\n")
	} else {
		for _, arg := range plan.Args {
			fmt.Fprintf(&out, "  - %s\n", strconv.Quote(arg))
		}
	}
	return out.String()
}

func renderCodexPrimarySessionResolution(out *strings.Builder, resolution CodexPrimarySessionResolution) {
	out.WriteString("primary_session:\n")
	renderCodexPrimarySessionStringResolution(out, "model", resolution.Model)
	renderCodexPrimarySessionStringResolution(out, "reasoning_effort", resolution.ReasoningEffort)

	fmt.Fprintln(out, "  yolo_mode:")
	fmt.Fprintf(out, "    effective_value: %t\n", resolution.YoloMode.EffectiveValue)
	fmt.Fprintf(out, "    effective_source: %s\n", resolution.YoloMode.EffectiveSource)
	if resolution.YoloMode.ProjectConfigured {
		fmt.Fprintf(out, "    project_value: %t\n", resolution.YoloMode.ProjectValue)
		fmt.Fprintf(out, "    project_source: %s\n", resolution.YoloMode.ProjectSource)
	} else {
		fmt.Fprintln(out, "    project_value: (absent)")
		fmt.Fprintln(out, "    project_source: (none)")
	}
	fmt.Fprintf(out, "    project_application: %s\n", resolution.YoloMode.ProjectApplication)
}

func renderCodexPrimarySessionStringResolution(out *strings.Builder, field string, resolution CodexPrimarySessionStringResolution) {
	fmt.Fprintf(out, "  %s:\n", field)
	if resolution.EffectiveValueKnown {
		fmt.Fprintf(out, "    effective_value: %s\n", strconv.Quote(resolution.EffectiveValue))
	} else {
		fmt.Fprintln(out, "    effective_value: (codex-native)")
	}
	fmt.Fprintf(out, "    effective_source: %s\n", resolution.EffectiveSource)
	if resolution.ProjectConfigured {
		fmt.Fprintf(out, "    project_value: %s\n", strconv.Quote(resolution.ProjectValue))
		fmt.Fprintf(out, "    project_source: %s\n", resolution.ProjectSource)
	} else {
		fmt.Fprintln(out, "    project_value: (absent)")
		fmt.Fprintln(out, "    project_source: (none)")
	}
	fmt.Fprintf(out, "    project_application: %s\n", resolution.ProjectApplication)
}

func formatTOMLStringArray(values []string) string {
	var out strings.Builder
	out.WriteString("[")
	for i, value := range values {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(strconv.Quote(value))
	}
	out.WriteString("]")
	return out.String()
}

type parsedCodexWrapperArgs struct {
	codexArgs         []string
	printConfig       bool
	dangerRequested   bool
	dangerSource      string
	explicit          codexExplicitSelections
	expandedShortcuts []CodexWrapperShortcut
}

type codexExplicitSelections struct {
	model                  bool
	modelValue             *codexExplicitValue
	reasoningEffort        bool
	reasoningEffortValue   *codexExplicitValue
	profile                bool
	profileSource          string
	profileValue           string
	remoteAuthTokenEnv     bool
	remoteAuthTokenEnvName string
	sandbox                *codexPolicySelection
	approval               *codexPolicySelection
	// Last -c/--config override per policy key, tracked independently of the
	// winning selection: Codex deserializes only the last override per key
	// (earlier repeats are masked by last-wins), and a typed flag does not
	// mask an invalid override (probe-verified exit 1 on
	// `codex exec --sandbox read-only -c 'sandbox_mode="banana"'`).
	sandboxConfigLast  *codexExplicitValue
	approvalConfigLast *codexExplicitValue
}

// ProviderArgumentError marks a pass-through provider argument the provider's
// own parser would reject, so callers can distinguish argument problems from
// invalid project configuration.
type ProviderArgumentError struct {
	msg string
}

func (e *ProviderArgumentError) Error() string { return e.msg }

func providerArgErrorf(format string, args ...any) error {
	return &ProviderArgumentError{msg: fmt.Sprintf(format, args...)}
}

// Provider-validated policy value domains, probe-verified on codex-cli
// 0.145.0. The typed clap flags and the -c/--config deserialization accept
// different approval sets: on-failure and granular are config-only variants
// the typed --ask-for-approval flag rejects with exit 2.
var (
	codexSandboxPolicyValues        = []string{"read-only", "workspace-write", "danger-full-access"}
	codexApprovalFlagPolicyValues   = []string{"untrusted", "on-request", "never"}
	codexApprovalConfigPolicyValues = []string{"untrusted", "on-failure", "on-request", "granular", "never"}
)

// codexPolicySelection records one explicit pass-through sandbox or approval
// selection. Codex resolves the typed CLI flag after `-c` key overrides, so a
// flag selection wins over a config selection regardless of argument order,
// while repeated `-c` overrides keep provider last-wins semantics. Repeated
// flags are rejected exactly like the Codex clap parser rejects them.
type codexPolicySelection struct {
	value    string
	source   string
	fromFlag bool
}

func (s *codexExplicitSelections) recordPolicyFlag(field **codexPolicySelection, flagName, value string, allowed []string) error {
	if *field != nil && (*field).fromFlag {
		return providerArgErrorf("the Codex argument %s cannot be used multiple times", flagName)
	}
	// The Codex clap parser validates every typed policy value against its
	// enum (probe-verified exit 2, "invalid value 'banana'"), so an unknown
	// value fails closed instead of composing an argv Codex rejects at launch.
	if !containsString(allowed, value) {
		return providerArgErrorf("invalid value %s for the Codex argument %s (possible values: %s)", strconv.Quote(value), flagName, strings.Join(allowed, ", "))
	}
	*field = &codexPolicySelection{value: value, source: "cli:" + flagName, fromFlag: true}
	return nil
}

// codexProfileNamePattern mirrors the plain profile-name parser of the Codex
// CONFIG_PROFILE_V2 value: one or more ASCII letters, digits, dashes, or
// underscores. Probe-verified on codex-cli 0.145.0: empty, dot, slash, space,
// plus, at, tilde, comma, colon, backslash, equals, and non-ASCII values all
// exit 2 with "invalid --profile value ...; pass a plain name such as `work`",
// while values such as a_b, a-b, ab1, A_B, 123, -ab, _ab, and ab- are accepted.
var codexProfileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// recordProfileFlag registers one explicit --profile/-p occurrence in any
// spelling. The Codex clap parser rejects a repeated profile option at
// runtime (probe-verified exit 2, "the argument '--profile
// <CONFIG_PROFILE_V2>' cannot be used multiple times"; an `app-server --help`
// parser probe is insufficient because help short-circuits this validation)
// and validates the value against its plain profile-name syntax, so both a
// second occurrence and a name outside that domain fail closed instead of
// composing an argv Codex rejects at launch. Whether the named profile exists
// stays provider-native config resolution; only parser syntax is mirrored.
func (s *codexExplicitSelections) recordProfileFlag(flagName, value string) error {
	if s.profile {
		return providerArgErrorf("the Codex argument %s cannot be used multiple times", flagName)
	}
	if !codexProfileNamePattern.MatchString(value) {
		return providerArgErrorf("invalid value %s for the Codex argument %s; pass a plain profile name such as work", strconv.Quote(value), flagName)
	}
	s.profile = true
	return nil
}

// recordRemoteAuthTokenEnv registers one explicit --remote-auth-token-env
// occurrence. The Codex clap parser rejects a repeated occurrence
// (probe-verified exit 2, "the argument '--remote-auth-token-env <ENV_VAR>'
// cannot be used multiple times" on codex-cli 0.145.0), so a second one fails
// closed. The recorded value is the environment variable NAME the contract
// surfaces in required_env_names; the value behind it is never read. Codex
// accepts an empty name (probe exit 0 on --remote-auth-token-env=), which
// simply yields no environment requirement.
func (s *codexExplicitSelections) recordRemoteAuthTokenEnv(flagName, value string) error {
	if s.remoteAuthTokenEnv {
		return providerArgErrorf("the Codex argument %s cannot be used multiple times", flagName)
	}
	s.remoteAuthTokenEnv = true
	s.remoteAuthTokenEnvName = value
	return nil
}

func (s *codexExplicitSelections) recordPolicyConfig(field **codexPolicySelection, value codexExplicitValue) {
	if *field != nil && (*field).fromFlag {
		return
	}
	*field = &codexPolicySelection{value: value.effective, source: value.source}
}

// validatePolicyConfigDomains mirrors the config deserialization Codex runs
// at startup: only the last -c/--config override per policy key is
// deserialized, an unknown or non-string value fails the launch (probe
// exit 1, "unknown variant"/"invalid type"), and a typed policy flag does not
// mask that failure. All probe-verified on codex-cli 0.145.0.
func (s *codexExplicitSelections) validatePolicyConfigDomains() error {
	if err := validateCodexPolicyConfigValue(s.sandboxConfigLast, "sandbox_mode", codexSandboxPolicyValues); err != nil {
		return err
	}
	return validateCodexPolicyConfigValue(s.approvalConfigLast, "approval_policy", codexApprovalConfigPolicyValues)
}

func validateCodexPolicyConfigValue(value *codexExplicitValue, key string, allowed []string) error {
	if value == nil {
		return nil
	}
	// A non-string TOML value keeps its raw text as the effective string
	// ("true", "3"), which is never a domain member, so it fails closed the
	// same way Codex rejects it ("invalid type ... expected string").
	if !containsString(allowed, value.effective) {
		return providerArgErrorf("unknown variant %s for the Codex config override %s (expected one of: %s)", strconv.Quote(value.effective), key, strings.Join(allowed, ", "))
	}
	return nil
}

func (s *codexExplicitSelections) policySelectionSource() string {
	if s.sandbox != nil {
		return s.sandbox.source
	}
	if s.approval != nil {
		return s.approval.source
	}
	return ""
}

type codexExplicitValue struct {
	comparable any
	display    string
	effective  string
	source     string
}

func parseCodexWrapperArgs(args []string) (parsedCodexWrapperArgs, error) {
	var parsed parsedCodexWrapperArgs
	passThrough := false
	for _, arg := range args {
		if arg == codexDangerouslyBypassApprovalsAndSandbox {
			parsed.dangerRequested = true
			if parsed.dangerSource == "" {
				parsed.dangerSource = "cli:" + codexDangerouslyBypassApprovalsAndSandbox
			}
			continue
		}
		if passThrough {
			parsed.codexArgs = append(parsed.codexArgs, arg)
			continue
		}
		switch arg {
		case "--":
			passThrough = true
		case "--print-config":
			parsed.printConfig = true
		case "-d", "--danger", "--yolo":
			parsed.dangerRequested = true
			if parsed.dangerSource == "" {
				parsed.dangerSource = "wrapper:" + arg
			}
			parsed.expandedShortcuts = append(parsed.expandedShortcuts, CodexWrapperShortcut{
				From: arg,
				To:   codexDangerouslyBypassApprovalsAndSandbox,
			})
		default:
			parsed.codexArgs = append(parsed.codexArgs, arg)
		}
	}
	normalizedArgs, explicit, err := normalizeCodexExplicitSelections(parsed.codexArgs)
	if err != nil {
		return parsedCodexWrapperArgs{}, err
	}
	// The Codex clap parser rejects the bypass flag alongside the typed
	// sandbox/approval flags, so an explicit danger request combined with an
	// explicit policy flag can never launch; fail closed with the same rule.
	if parsed.dangerRequested {
		if explicit.sandbox != nil && explicit.sandbox.fromFlag {
			return parsedCodexWrapperArgs{}, providerArgErrorf("the Codex argument %s cannot be used with --sandbox", codexDangerouslyBypassApprovalsAndSandbox)
		}
		if explicit.approval != nil && explicit.approval.fromFlag {
			return parsedCodexWrapperArgs{}, providerArgErrorf("the Codex argument %s cannot be used with --ask-for-approval", codexDangerouslyBypassApprovalsAndSandbox)
		}
	}
	// Config-level policy domains are checked after every clap-level rule,
	// mirroring Codex order: clap parses the full command line first and the
	// -c overrides only fail later, when config deserialization runs at
	// startup.
	if err := explicit.validatePolicyConfigDomains(); err != nil {
		return parsedCodexWrapperArgs{}, err
	}
	parsed.codexArgs = normalizedArgs
	parsed.explicit = explicit
	return parsed, nil
}

func normalizeCodexExplicitSelections(args []string) ([]string, codexExplicitSelections, error) {
	var normalized []string
	var selections codexExplicitSelections

	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			normalized = append(normalized, args[index:]...)
			break
		}
		switch {
		case arg == "--model" || arg == "-m":
			selections.model = true
			value, consumed, err := takeCodexOptionValue(args, index, arg)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			index += consumed
			keep, err := acceptCodexExplicitValue("model", &selections.modelValue, directCodexExplicitValue(value, "cli:"+arg))
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, arg, value)
			}
		case strings.HasPrefix(arg, "--model=") || strings.HasPrefix(arg, "-m="):
			selections.model = true
			option, value, _ := strings.Cut(arg, "=")
			keep, err := acceptCodexExplicitValue("model", &selections.modelValue, directCodexExplicitValue(value, "cli:"+option))
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, arg)
			}
		case strings.HasPrefix(arg, "-m") && len(arg) > 2:
			// Attached short form -mVALUE, accepted by the Codex parser.
			selections.model = true
			keep, err := acceptCodexExplicitValue("model", &selections.modelValue, directCodexExplicitValue(strings.TrimPrefix(arg, "-m"), "cli:-m"))
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, arg)
			}
		case arg == "--profile" || arg == "-p":
			value, consumed, err := takeCodexOptionValue(args, index, arg)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if err := selections.recordProfileFlag(arg, value); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			index += consumed
			selections.profileSource = "cli:" + arg
			selections.profileValue = value
			normalized = append(normalized, arg, value)
		case strings.HasPrefix(arg, "--profile=") || strings.HasPrefix(arg, "-p="):
			option, value, _ := strings.Cut(arg, "=")
			if err := selections.recordProfileFlag(option, value); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			selections.profileSource = "cli:" + option
			selections.profileValue = value
			normalized = append(normalized, arg)
		case strings.HasPrefix(arg, "-p") && len(arg) > 2:
			// Attached short form -pVALUE, accepted by the Codex parser.
			value := strings.TrimPrefix(arg, "-p")
			if err := selections.recordProfileFlag("-p", value); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			selections.profileSource = "cli:-p"
			selections.profileValue = value
			normalized = append(normalized, arg)
		case arg == "--remote-auth-token-env":
			value, consumed, err := takeCodexOptionValue(args, index, arg)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if err := selections.recordRemoteAuthTokenEnv(arg, value); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			index += consumed
			normalized = append(normalized, arg, value)
		case strings.HasPrefix(arg, "--remote-auth-token-env="):
			if err := selections.recordRemoteAuthTokenEnv("--remote-auth-token-env", strings.TrimPrefix(arg, "--remote-auth-token-env=")); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			normalized = append(normalized, arg)
		case arg == "--sandbox" || arg == "-s":
			value, consumed, err := takeCodexOptionValue(args, index, arg)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if err := selections.recordPolicyFlag(&selections.sandbox, arg, value, codexSandboxPolicyValues); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			normalized = append(normalized, args[index:index+1+consumed]...)
			index += consumed
		case strings.HasPrefix(arg, "--sandbox=") || strings.HasPrefix(arg, "-s=") || (strings.HasPrefix(arg, "-s") && len(arg) > 2 && !strings.HasPrefix(arg, "-s=")):
			flagName, value := splitCodexPolicyFlag(arg, "--sandbox", "-s")
			if err := selections.recordPolicyFlag(&selections.sandbox, flagName, value, codexSandboxPolicyValues); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			normalized = append(normalized, arg)
		case arg == "--ask-for-approval" || arg == "-a":
			value, consumed, err := takeCodexOptionValue(args, index, arg)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if err := selections.recordPolicyFlag(&selections.approval, arg, value, codexApprovalFlagPolicyValues); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			normalized = append(normalized, args[index:index+1+consumed]...)
			index += consumed
		case strings.HasPrefix(arg, "--ask-for-approval=") || strings.HasPrefix(arg, "-a=") || (strings.HasPrefix(arg, "-a") && len(arg) > 2 && !strings.HasPrefix(arg, "-a=")):
			flagName, value := splitCodexPolicyFlag(arg, "--ask-for-approval", "-a")
			if err := selections.recordPolicyFlag(&selections.approval, flagName, value, codexApprovalFlagPolicyValues); err != nil {
				return nil, codexExplicitSelections{}, err
			}
			normalized = append(normalized, arg)
		case arg == "-c" || arg == "--config":
			value, consumed, err := takeCodexOptionValue(args, index, arg)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			index += consumed
			keep, err := normalizeCodexConfigOverride(value, "cli:"+arg, &selections)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, arg, value)
			}
		case strings.HasPrefix(arg, "-c="):
			keep, err := normalizeCodexConfigOverride(strings.TrimPrefix(arg, "-c="), "cli:-c", &selections)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, arg)
			}
		case strings.HasPrefix(arg, "--config="):
			keep, err := normalizeCodexConfigOverride(strings.TrimPrefix(arg, "--config="), "cli:--config", &selections)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, arg)
			}
		case arg == "--enable" || arg == "--disable" || arg == "--local-provider":
			// Value-taking config-level globals that pass through untouched;
			// the Codex clap parser still requires one non-flag value per
			// occurrence, so a missing value fails closed here instead of
			// composing an argv Codex rejects at launch.
			value, consumed, err := takeCodexOptionValue(args, index, arg)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			index += consumed
			normalized = append(normalized, arg, value)
		default:
			normalized = append(normalized, arg)
		}
	}
	return normalized, selections, nil
}

// takeCodexOptionValue consumes the spaced value of a recognized value-taking
// Codex option. The Codex clap parser does not accept flag-like tokens as
// option values and requires one value per occurrence (probe-verified exit 2,
// "a value is required ... but none was supplied", for --model, --profile,
// -c/--config, --enable, --disable, --local-provider, and the policy flags),
// so both gaps fail closed instead of composing an argv Codex would reject.
func takeCodexOptionValue(args []string, index int, flagName string) (string, int, error) {
	if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
		return "", 0, providerArgErrorf("a value is required for the Codex argument %s", flagName)
	}
	return args[index+1], 1, nil
}

func splitCodexPolicyFlag(arg, longName, shortName string) (flagName, value string) {
	if rest, ok := strings.CutPrefix(arg, longName+"="); ok {
		return longName, rest
	}
	if rest, ok := strings.CutPrefix(arg, shortName+"="); ok {
		return shortName, rest
	}
	return shortName, strings.TrimPrefix(arg, shortName)
}

func normalizeCodexConfigOverride(value, source string, selections *codexExplicitSelections) (bool, error) {
	key, rawValue, ok := strings.Cut(value, "=")
	if !ok {
		return true, nil
	}
	switch strings.TrimSpace(key) {
	case "model":
		selections.model = true
		return acceptCodexExplicitValue(
			"model",
			&selections.modelValue,
			configCodexExplicitValue(rawValue, source+" model"),
		)
	case "model_reasoning_effort":
		selections.reasoningEffort = true
		return acceptCodexExplicitValue(
			"model_reasoning_effort",
			&selections.reasoningEffortValue,
			configCodexExplicitValue(rawValue, source+" model_reasoning_effort"),
		)
	case "sandbox_mode":
		value := configCodexExplicitValue(rawValue, source+" sandbox_mode")
		selections.sandboxConfigLast = &value
		selections.recordPolicyConfig(&selections.sandbox, value)
	case "approval_policy":
		value := configCodexExplicitValue(rawValue, source+" approval_policy")
		selections.approvalConfigLast = &value
		selections.recordPolicyConfig(&selections.approval, value)
	}
	return true, nil
}

func directCodexExplicitValue(value, source string) codexExplicitValue {
	return codexExplicitValue{
		comparable: value,
		display:    strconv.Quote(value),
		effective:  value,
		source:     source,
	}
}

func configCodexExplicitValue(value, source string) codexExplicitValue {
	trimmed := strings.TrimSpace(value)
	parsed := map[string]any{}
	if err := toml.Unmarshal([]byte("value = "+trimmed), &parsed); err == nil {
		if parsedValue, ok := parsed["value"]; ok {
			effective := trimmed
			if stringValue, ok := parsedValue.(string); ok {
				effective = stringValue
			}
			return codexExplicitValue{
				comparable: parsedValue,
				display:    strconv.Quote(trimmed),
				effective:  effective,
				source:     source,
			}
		}
	}
	return codexExplicitValue{
		comparable: trimmed,
		display:    strconv.Quote(trimmed),
		effective:  trimmed,
		source:     source,
	}
}

func acceptCodexExplicitValue(field string, current **codexExplicitValue, candidate codexExplicitValue) (bool, error) {
	if *current == nil {
		copy := candidate
		*current = &copy
		return true, nil
	}
	if reflect.DeepEqual((*current).comparable, candidate.comparable) {
		return false, nil
	}
	return false, fmt.Errorf(
		"conflicting explicit Codex values for field %s: %s and %s",
		field,
		(*current).display,
		candidate.display,
	)
}

func resolveCodexPrimarySession(policy CodexPrimarySessionPolicy, parsed parsedCodexWrapperArgs) (CodexPrimarySessionResolution, []string) {
	resolution := CodexPrimarySessionResolution{
		Model: resolveCodexPrimarySessionString(
			policy.Model,
			parsed.explicit.model,
			parsed.explicit.modelValue,
			parsed.explicit.profile,
			parsed.explicit.profileSource,
		),
		ReasoningEffort: resolveCodexPrimarySessionString(
			policy.ReasoningEffort,
			parsed.explicit.reasoningEffort,
			parsed.explicit.reasoningEffortValue,
			parsed.explicit.profile,
			parsed.explicit.profileSource,
		),
		YoloMode: resolveCodexPrimarySessionYolo(policy.YoloMode, parsed),
	}

	var args []string
	if resolution.Model.ProjectApplication == CodexPrimarySessionApplied {
		args = append(args, "--model", policy.Model.Value)
	}
	if resolution.ReasoningEffort.ProjectApplication == CodexPrimarySessionApplied {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%s", strconv.Quote(policy.ReasoningEffort.Value)))
	}
	if resolution.YoloMode.EffectiveValue {
		args = append(args, codexDangerouslyBypassApprovalsAndSandbox)
	}
	return resolution, args
}

func resolveCodexPrimarySessionString(
	project CodexPrimarySessionStringValue,
	explicit bool,
	explicitValue *codexExplicitValue,
	profile bool,
	profileSource string,
) CodexPrimarySessionStringResolution {
	resolution := CodexPrimarySessionStringResolution{
		EffectiveSource:    "native",
		ProjectConfigured:  project.Present,
		ProjectValue:       project.Value,
		ProjectSource:      project.Source,
		ProjectApplication: CodexPrimarySessionNotConfigured,
	}

	switch {
	case explicit:
		resolution.EffectiveSource = "explicit_cli"
		if explicitValue != nil {
			resolution.EffectiveValue = explicitValue.effective
			resolution.EffectiveValueKnown = true
			resolution.EffectiveSource = explicitValue.source
		}
		if project.Present {
			resolution.ProjectApplication = CodexPrimarySessionSuppressedByCLI
		}
	case profile:
		resolution.EffectiveSource = profileSource
		if resolution.EffectiveSource == "" {
			resolution.EffectiveSource = "explicit_profile"
		}
		if project.Present {
			resolution.ProjectApplication = CodexPrimarySessionSuppressedByProfile
		}
	case project.Present:
		resolution.EffectiveValue = project.Value
		resolution.EffectiveValueKnown = true
		resolution.EffectiveSource = project.Source
		resolution.ProjectApplication = CodexPrimarySessionApplied
	}

	return resolution
}

func resolveCodexPrimarySessionYolo(project CodexPrimarySessionBoolValue, parsed parsedCodexWrapperArgs) CodexPrimarySessionBoolResolution {
	resolution := CodexPrimarySessionBoolResolution{
		EffectiveSource:    "default",
		ProjectConfigured:  project.Present,
		ProjectValue:       project.Value,
		ProjectSource:      project.Source,
		ProjectApplication: CodexPrimarySessionNotConfigured,
	}

	if parsed.dangerRequested {
		resolution.EffectiveValue = true
		resolution.EffectiveSource = parsed.dangerSource
		if resolution.EffectiveSource == "" {
			resolution.EffectiveSource = "explicit_cli"
		}
		if project.Present {
			resolution.ProjectApplication = CodexPrimarySessionSuppressedByCLI
		}
		return resolution
	}
	// An explicit sandbox or approval selection suppresses project yolo_mode:
	// composing the bypass flag next to a typed policy flag is a Codex clap
	// conflict, and next to a `-c` policy override it would silently discard
	// the user's explicit selection at runtime.
	if project.Present && project.Value && parsed.explicit.policySelectionSource() != "" {
		resolution.EffectiveSource = parsed.explicit.policySelectionSource()
		resolution.ProjectApplication = CodexPrimarySessionSuppressedByCLI
		return resolution
	}
	if project.Present {
		resolution.EffectiveValue = project.Value
		resolution.EffectiveSource = project.Source
		resolution.ProjectApplication = CodexPrimarySessionApplied
	}
	return resolution
}

func ancestorDirsRootFirst(startDir string) []string {
	dir := filepath.Clean(startDir)
	var cwdFirst []string
	for {
		cwdFirst = append(cwdFirst, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	rootFirst := make([]string, 0, len(cwdFirst))
	for i := len(cwdFirst) - 1; i >= 0; i-- {
		rootFirst = append(rootFirst, cwdFirst[i])
	}
	return rootFirst
}

func loadCompositeMCPRegistry(homeDir string, ancestors []string) (map[string]codexMCPDefinition, []CodexMCPRegistrySource, error) {
	definitions := map[string]codexMCPDefinition{}
	var sources []CodexMCPRegistrySource

	globalPath := filepath.Join(homeDir, ".agents", ".configs", "codex-mcp-servers.toml")
	if err := mergeMCPRegistry(definitions, &sources, globalPath, "global"); err != nil {
		return nil, nil, err
	}
	for _, dir := range ancestors {
		path := filepath.Join(dir, ".agents", ".configs", "codex-mcp-servers.toml")
		if samePath(path, globalPath) {
			continue
		}
		if err := mergeMCPRegistry(definitions, &sources, path, "project"); err != nil {
			return nil, nil, err
		}
	}
	return definitions, sources, nil
}

func mergeMCPRegistry(definitions map[string]codexMCPDefinition, sources *[]CodexMCPRegistrySource, path, scope string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read Codex MCP registry %s: %w", path, err)
	}
	registry, err := parseCodexMCPRegistry(data, path)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(registry))
	for name, server := range registry {
		names = append(names, name)
		definitions[name] = codexMCPDefinition{
			Server: server,
			Source: path,
		}
	}
	sort.Strings(names)
	*sources = append(*sources, CodexMCPRegistrySource{
		Path:        path,
		Scope:       scope,
		ServerNames: names,
	})
	return nil
}
