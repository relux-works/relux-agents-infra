package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	plan.ProjectConfigs = projectConfig.Sources
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
	model                bool
	modelValue           *codexExplicitValue
	reasoningEffort      bool
	reasoningEffortValue *codexExplicitValue
	profile              bool
	profileSource        string
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
			normalized = append(normalized, arg)
			if index+1 >= len(args) {
				continue
			}
			index++
			candidate := directCodexExplicitValue(args[index], "cli:"+arg)
			keep, err := acceptCodexExplicitValue("model", &selections.modelValue, candidate)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, args[index])
			} else {
				normalized = normalized[:len(normalized)-1]
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
		case arg == "--profile" || arg == "-p":
			selections.profile = true
			if selections.profileSource == "" {
				selections.profileSource = "cli:" + arg
			}
			normalized = append(normalized, arg)
			if index+1 < len(args) {
				index++
				normalized = append(normalized, args[index])
			}
		case strings.HasPrefix(arg, "--profile=") || strings.HasPrefix(arg, "-p="):
			selections.profile = true
			if selections.profileSource == "" {
				option, _, _ := strings.Cut(arg, "=")
				selections.profileSource = "cli:" + option
			}
			normalized = append(normalized, arg)
		case arg == "-c" || arg == "--config":
			if index+1 >= len(args) {
				normalized = append(normalized, arg)
				continue
			}
			index++
			keep, err := normalizeCodexConfigOverride(args[index], "cli:"+arg, &selections)
			if err != nil {
				return nil, codexExplicitSelections{}, err
			}
			if keep {
				normalized = append(normalized, arg, args[index])
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
		default:
			normalized = append(normalized, arg)
		}
	}
	return normalized, selections, nil
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

func loadCompositeMCPEnablement(ancestors []string, globalProjectConfigPath, section string) ([]string, map[string][]string, []CodexProjectConfigSource, error) {
	if section != "mcp" {
		return nil, nil, nil, fmt.Errorf("unsupported project config MCP section %q", section)
	}
	composite, err := loadCompositeProjectConfig(ancestors, globalProjectConfigPath)
	if err != nil {
		return nil, nil, nil, err
	}
	return composite.EnabledOrder, composite.EnabledBy, composite.Sources, nil
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
