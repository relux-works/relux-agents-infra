package infra

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const claudeDangerouslySkipPermissions = "--dangerously-skip-permissions"

type ClaudeLaunchPlan struct {
	StartDir                 string
	HomeDir                  string
	ProjectConfigs           []ClaudeProjectConfigSource
	RegistrySources          []CodexMCPRegistrySource
	MCPServers               []ClaudeMCPLaunchServer
	MCPConfigJSON            string
	PrimarySession           ClaudePrimarySessionPolicy
	PrimarySessionResolution ClaudePrimarySessionResolution
	ConfigArgs               []string
	UserArgs                 []string
	Args                     []string
	PrintConfig              bool
	WrapperExpandedShortcuts []CodexWrapperShortcut
}

type ClaudeProjectConfigSource struct {
	Path           string
	EnabledServers []string
	PrimarySession ClaudePrimarySessionSource
}

type ClaudePrimarySessionApplication string

const (
	ClaudePrimarySessionNotConfigured   ClaudePrimarySessionApplication = "not_configured"
	ClaudePrimarySessionApplied         ClaudePrimarySessionApplication = "applied"
	ClaudePrimarySessionSuppressedByCLI ClaudePrimarySessionApplication = "suppressed_by_explicit_cli"
)

// ClaudePrimarySessionResolution records the invocation-level Claude model
// decision. ProjectValue and ProjectSource preserve the composed project
// policy even when an explicit Claude CLI model suppresses its application.
type ClaudePrimarySessionResolution struct {
	Model ClaudePrimarySessionStringResolution
}

type ClaudePrimarySessionStringResolution struct {
	EffectiveValue      string
	EffectiveValueKnown bool
	EffectiveSource     string
	ProjectConfigured   bool
	ProjectValue        string
	ProjectSource       string
	ProjectApplication  ClaudePrimarySessionApplication
}

type ClaudeMCPLaunchServer struct {
	Name              string
	URL               string
	BearerTokenEnvVar string
	Command           string
	Args              []string
	DefinitionSource  string
	EnabledBy         []string
}

// BuildClaudeLaunchPlan mirrors BuildCodexLaunchPlan: it walks the same
// ancestor .agents/.configs/project-config.toml files, reads the same
// shared [mcp] enabled_servers opt-in and the same codex-mcp-servers.toml
// registries, but renders the result as a Claude Code --mcp-config JSON
// payload instead of Codex `-c` overrides. The opt-in list is intentionally
// agent-agnostic — one list drives which servers agents-infra codex and
// agents-infra claude each compose into their own format.
func BuildClaudeLaunchPlan(startDir, homeDir string, args []string) (ClaudeLaunchPlan, error) {
	parsed, err := parseClaudeWrapperArgs(args)
	if err != nil {
		return ClaudeLaunchPlan{}, err
	}
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return ClaudeLaunchPlan{}, fmt.Errorf("resolve cwd: %w", err)
		}
	}
	startDir, err = filepath.Abs(startDir)
	if err != nil {
		return ClaudeLaunchPlan{}, fmt.Errorf("resolve start dir: %w", err)
	}
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return ClaudeLaunchPlan{}, fmt.Errorf("resolve home dir: %w", err)
		}
	}
	homeDir, err = filepath.Abs(homeDir)
	if err != nil {
		return ClaudeLaunchPlan{}, fmt.Errorf("resolve home dir: %w", err)
	}

	plan := ClaudeLaunchPlan{
		StartDir:                 startDir,
		HomeDir:                  homeDir,
		UserArgs:                 parsed.claudeArgs,
		PrintConfig:              parsed.printConfig,
		WrapperExpandedShortcuts: parsed.expandedShortcuts,
	}

	ancestors := ancestorDirsRootFirst(startDir)
	globalProjectConfigPath := filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName)
	projectConfig, err := loadCompositeProjectConfig(ancestors, globalProjectConfigPath)
	if err != nil {
		return ClaudeLaunchPlan{}, err
	}
	plan.PrimarySession = projectConfig.ClaudePrimarySession
	for _, source := range projectConfig.Sources {
		plan.ProjectConfigs = append(plan.ProjectConfigs, ClaudeProjectConfigSource{
			Path:           source.Path,
			EnabledServers: append([]string(nil), source.EnabledServers...),
			PrimarySession: cloneClaudePrimarySessionSource(source.ClaudePrimarySession),
		})
	}

	definitions, registrySources, err := loadCompositeMCPRegistry(homeDir, ancestors)
	if err != nil {
		return ClaudeLaunchPlan{}, err
	}
	plan.RegistrySources = registrySources

	mcpServers := map[string]claudeMCPConfigServer{}
	for _, name := range projectConfig.EnabledOrder {
		def, ok := definitions[name]
		if !ok {
			return ClaudeLaunchPlan{}, fmt.Errorf("MCP server %q is enabled by %s but no definition was found in codex-mcp-servers.toml registries", name, strings.Join(projectConfig.EnabledBy[name], ", "))
		}
		if err := validateCodexMCPDefinition(name, def); err != nil {
			return ClaudeLaunchPlan{}, err
		}
		server := ClaudeMCPLaunchServer{
			Name:              name,
			URL:               def.Server.URL,
			BearerTokenEnvVar: def.Server.BearerTokenEnvVar,
			Command:           def.Server.Command,
			Args:              append([]string(nil), def.Server.Args...),
			DefinitionSource:  def.Source,
			EnabledBy:         append([]string(nil), projectConfig.EnabledBy[name]...),
		}
		plan.MCPServers = append(plan.MCPServers, server)
		mcpServers[name] = claudeMCPConfigServer(server)
	}

	if len(mcpServers) > 0 {
		configJSON, err := marshalClaudeMCPConfig(mcpServers)
		if err != nil {
			return ClaudeLaunchPlan{}, fmt.Errorf("encode Claude MCP config: %w", err)
		}
		plan.MCPConfigJSON = configJSON
		plan.ConfigArgs = []string{"--mcp-config", configJSON}
	}
	primaryResolution, primaryArgs := resolveClaudePrimarySession(plan.PrimarySession, parsed)
	plan.PrimarySessionResolution = primaryResolution
	plan.ConfigArgs = append(plan.ConfigArgs, primaryArgs...)
	plan.Args = append(append([]string(nil), plan.ConfigArgs...), plan.UserArgs...)
	return plan, nil
}

type claudeMCPConfigServer ClaudeMCPLaunchServer

type claudeMCPConfigEntry struct {
	Type    string            `json:"type"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
}

func marshalClaudeMCPConfig(servers map[string]claudeMCPConfigServer) (string, error) {
	entries := make(map[string]claudeMCPConfigEntry, len(servers))
	for name, server := range servers {
		if server.URL != "" {
			entry := claudeMCPConfigEntry{Type: "http", URL: server.URL}
			if server.BearerTokenEnvVar != "" {
				entry.Headers = map[string]string{
					"Authorization": fmt.Sprintf("Bearer ${%s}", server.BearerTokenEnvVar),
				}
			}
			entries[name] = entry
			continue
		}
		entries[name] = claudeMCPConfigEntry{
			Type:    "stdio",
			Command: server.Command,
			Args:    server.Args,
		}
	}
	payload, err := json.Marshal(struct {
		MCPServers map[string]claudeMCPConfigEntry `json:"mcpServers"`
	}{MCPServers: entries})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func RenderClaudeLaunchPlan(plan ClaudeLaunchPlan) string {
	var out strings.Builder
	out.WriteString("agents-infra claude config\n")
	fmt.Fprintf(&out, "cwd: %s\n", plan.StartDir)

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

	renderClaudePrimarySessionResolution(&out, plan.PrimarySessionResolution)

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

	out.WriteString("claude_args:\n")
	if len(plan.Args) == 0 {
		out.WriteString("  - (none)\n")
	} else {
		for _, arg := range plan.Args {
			fmt.Fprintf(&out, "  - %s\n", strconv.Quote(arg))
		}
	}
	return out.String()
}

func renderClaudePrimarySessionResolution(out *strings.Builder, resolution ClaudePrimarySessionResolution) {
	out.WriteString("primary_session:\n")
	out.WriteString("  model:\n")
	if resolution.Model.EffectiveValueKnown {
		fmt.Fprintf(out, "    effective_value: %s\n", strconv.Quote(resolution.Model.EffectiveValue))
	} else {
		fmt.Fprintln(out, "    effective_value: (claude-native)")
	}
	fmt.Fprintf(out, "    effective_source: %s\n", resolution.Model.EffectiveSource)
	if resolution.Model.ProjectConfigured {
		fmt.Fprintf(out, "    project_value: %s\n", strconv.Quote(resolution.Model.ProjectValue))
		fmt.Fprintf(out, "    project_source: %s\n", resolution.Model.ProjectSource)
	} else {
		fmt.Fprintln(out, "    project_value: (absent)")
		fmt.Fprintln(out, "    project_source: (none)")
	}
	fmt.Fprintf(out, "    project_application: %s\n", resolution.Model.ProjectApplication)
}

type parsedClaudeWrapperArgs struct {
	claudeArgs          []string
	printConfig         bool
	explicitModel       bool
	explicitModelValue  string
	explicitModelSource string
	expandedShortcuts   []CodexWrapperShortcut
}

func parseClaudeWrapperArgs(args []string) (parsedClaudeWrapperArgs, error) {
	var parsed parsedClaudeWrapperArgs
	passThrough := false
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if passThrough {
			parsed.claudeArgs = append(parsed.claudeArgs, arg)
			continue
		}
		switch {
		case arg == "--":
			passThrough = true
		case arg == "--print-config":
			parsed.printConfig = true
		case arg == "-d" || arg == "--danger" || arg == "--yolo":
			parsed.claudeArgs = append(parsed.claudeArgs, claudeDangerouslySkipPermissions)
			parsed.expandedShortcuts = append(parsed.expandedShortcuts, CodexWrapperShortcut{
				From: arg,
				To:   claudeDangerouslySkipPermissions,
			})
		case arg == "--model":
			parsed.claudeArgs = append(parsed.claudeArgs, arg)
			parsed.explicitModel = true
			parsed.explicitModelSource = "cli:--model"
			if index+1 < len(args) {
				index++
				parsed.explicitModelValue = args[index]
				parsed.claudeArgs = append(parsed.claudeArgs, args[index])
			}
		case strings.HasPrefix(arg, "--model="):
			parsed.claudeArgs = append(parsed.claudeArgs, arg)
			parsed.explicitModel = true
			parsed.explicitModelSource = "cli:--model"
			parsed.explicitModelValue = strings.TrimPrefix(arg, "--model=")
		default:
			parsed.claudeArgs = append(parsed.claudeArgs, arg)
		}
	}
	return parsed, nil
}

func resolveClaudePrimarySession(policy ClaudePrimarySessionPolicy, parsed parsedClaudeWrapperArgs) (ClaudePrimarySessionResolution, []string) {
	resolution := ClaudePrimarySessionResolution{
		Model: ClaudePrimarySessionStringResolution{
			EffectiveSource:    "native",
			ProjectConfigured:  policy.Model.Present,
			ProjectValue:       policy.Model.Value,
			ProjectSource:      policy.Model.Source,
			ProjectApplication: ClaudePrimarySessionNotConfigured,
		},
	}
	if parsed.explicitModel {
		resolution.Model.EffectiveSource = parsed.explicitModelSource
		if resolution.Model.EffectiveSource == "" {
			resolution.Model.EffectiveSource = "explicit_cli"
		}
		if parsed.explicitModelValue != "" {
			resolution.Model.EffectiveValue = parsed.explicitModelValue
			resolution.Model.EffectiveValueKnown = true
		}
		if policy.Model.Present {
			resolution.Model.ProjectApplication = ClaudePrimarySessionSuppressedByCLI
		}
		return resolution, nil
	}
	if !policy.Model.Present {
		return resolution, nil
	}
	resolution.Model.EffectiveValue = policy.Model.Value
	resolution.Model.EffectiveValueKnown = true
	resolution.Model.EffectiveSource = policy.Model.Source
	resolution.Model.ProjectApplication = ClaudePrimarySessionApplied
	return resolution, []string{"--model", policy.Model.Value}
}
