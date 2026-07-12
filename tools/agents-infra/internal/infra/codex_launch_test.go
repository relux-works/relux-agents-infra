package infra

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildCodexLaunchPlanComposesAncestorConfigsAndProvenance(t *testing.T) {
	t.Setenv("JIRA_TOKEN", "must-not-appear-in-diagnostics")
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)

	mustMkdir(t, filepath.Join(home, ".agents", ".configs"))
	mustWrite(t, filepath.Join(home, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://global.example/figma\"\n")
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"figma\"]\n")
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://parent.example/figma\"\n")
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	mustWrite(t, filepath.Join(child, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"jira\", \"figma\"]\n")
	mustWrite(t, filepath.Join(child, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.jira]\nurl = \"https://child.example/jira\"\nbearer_token_env_var = \"JIRA_TOKEN\"\n")

	plan, err := BuildCodexLaunchPlan(child, home, []string{"-d", "-"})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}

	if len(plan.MCPServers) != 2 {
		t.Fatalf("MCPServers = %#v, want 2 entries", plan.MCPServers)
	}
	if plan.MCPServers[0].Name != "figma" || plan.MCPServers[0].URL != "https://parent.example/figma" {
		t.Fatalf("figma server = %#v", plan.MCPServers[0])
	}
	if len(plan.MCPServers[0].EnabledBy) != 2 {
		t.Fatalf("figma EnabledBy = %#v, want parent and child configs", plan.MCPServers[0].EnabledBy)
	}
	if plan.MCPServers[1].Name != "jira" || plan.MCPServers[1].BearerTokenEnvVar != "JIRA_TOKEN" {
		t.Fatalf("jira server = %#v", plan.MCPServers[1])
	}

	wantArgs := []string{
		"-c", "mcp_servers.figma.url=\"https://parent.example/figma\"",
		"-c", "mcp_servers.jira.url=\"https://child.example/jira\"",
		"-c", "mcp_servers.jira.bearer_token_env_var=\"JIRA_TOKEN\"",
		codexDangerouslyBypassApprovalsAndSandbox,
		"-",
	}
	if !reflect.DeepEqual(plan.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", plan.Args, wantArgs)
	}

	rendered := RenderCodexLaunchPlan(plan)
	if strings.Contains(rendered, "must-not-appear-in-diagnostics") {
		t.Fatalf("rendered plan leaked bearer token environment value:\n%s", rendered)
	}
	for _, want := range []string{
		"agents-infra codex config",
		"enabled_mcp=figma",
		"enabled_mcp=jira,figma",
		"definition: " + filepath.Join(parent, ".agents", ".configs", "codex-mcp-servers.toml"),
		"definition: " + filepath.Join(child, ".agents", ".configs", "codex-mcp-servers.toml"),
		"-d => " + codexDangerouslyBypassApprovalsAndSandbox,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan missing %q:\n%s", want, rendered)
		}
	}
}

func TestBuildCodexLaunchPlanIgnoresHomeAgentsProjectConfigWithoutProjectOptIn(t *testing.T) {
	home := t.TempDir()
	start := filepath.Join(home, "project", "subdir")
	mustMkdir(t, start)
	mustMkdir(t, filepath.Join(home, ".agents", ".configs"))
	mustWrite(t, filepath.Join(home, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://global.example/figma\"\n")
	mustWrite(t, filepath.Join(home, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"figma\"]\n")

	plan, err := BuildCodexLaunchPlan(start, home, []string{"exec", "hello"})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if len(plan.MCPServers) != 0 || len(plan.ConfigArgs) != 0 {
		t.Fatalf("home agents registry/config should not enable MCP without project opt-in: %+v", plan)
	}
	wantArgs := []string{"exec", "hello"}
	if !reflect.DeepEqual(plan.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", plan.Args, wantArgs)
	}
	if !strings.Contains(RenderCodexLaunchPlan(plan), "enabled_mcp:\n  - (none)") {
		t.Fatalf("rendered plan should report no enabled MCP:\n%s", RenderCodexLaunchPlan(plan))
	}
}

func TestBuildCodexLaunchPlanSupportsStdioMCPServers(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(home, ".agents", ".configs"))
	mustWrite(t, filepath.Join(home, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.lldb]\ncommand = \"lldb-mcp\"\nargs = [\"--socket\", \"auto\"]\n")
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	mustWrite(t, filepath.Join(start, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"lldb\"]\n")

	plan, err := BuildCodexLaunchPlan(start, home, nil)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}

	if len(plan.MCPServers) != 1 {
		t.Fatalf("MCPServers = %#v, want 1 entry", plan.MCPServers)
	}
	server := plan.MCPServers[0]
	if server.Name != "lldb" || server.Command != "lldb-mcp" || !reflect.DeepEqual(server.Args, []string{"--socket", "auto"}) {
		t.Fatalf("lldb server = %#v", server)
	}

	wantArgs := []string{
		"-c", "mcp_servers.lldb.command=\"lldb-mcp\"",
		"-c", "mcp_servers.lldb.args=[\"--socket\", \"auto\"]",
	}
	if !reflect.DeepEqual(plan.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", plan.Args, wantArgs)
	}

	rendered := RenderCodexLaunchPlan(plan)
	for _, want := range []string{
		"enabled_mcp:\n  - lldb",
		"command: lldb-mcp",
		"args: [\"--socket\", \"auto\"]",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan missing %q:\n%s", want, rendered)
		}
	}
}

func TestBuildCodexLaunchPlanSupportsSafariMCPOptIn(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	safariCommand := "/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver"
	mustMkdir(t, filepath.Join(home, ".agents", ".configs"))
	mustWrite(t, filepath.Join(home, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.safari]\ncommand = \""+safariCommand+"\"\nargs = [\"--mcp\"]\n")

	plainPlan, err := BuildCodexLaunchPlan(start, home, nil)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan without opt-in: %v", err)
	}
	if len(plainPlan.MCPServers) != 0 {
		t.Fatalf("Safari should not be enabled from registry alone: %#v", plainPlan.MCPServers)
	}

	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	mustWrite(t, filepath.Join(start, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"safari\"]\n")

	plan, err := BuildCodexLaunchPlan(start, home, nil)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if len(plan.MCPServers) != 1 {
		t.Fatalf("MCPServers = %#v, want 1 entry", plan.MCPServers)
	}
	server := plan.MCPServers[0]
	if server.Name != "safari" || server.Command != safariCommand || !reflect.DeepEqual(server.Args, []string{"--mcp"}) {
		t.Fatalf("safari server = %#v", server)
	}

	wantArgs := []string{
		"-c", "mcp_servers.safari.command=\"/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver\"",
		"-c", "mcp_servers.safari.args=[\"--mcp\"]",
	}
	if !reflect.DeepEqual(plan.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", plan.Args, wantArgs)
	}

	rendered := RenderCodexLaunchPlan(plan)
	for _, want := range []string{
		"enabled_mcp:\n  - safari",
		"command: " + safariCommand,
		"args: [\"--mcp\"]",
		"mcp_servers.safari.command=\\\"/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver\\\"",
		"mcp_servers.safari.args=[\\\"--mcp\\\"]",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan missing %q:\n%s", want, rendered)
		}
	}
}

func TestBuildCodexLaunchPlanPrintConfigStopsWrapperParsingAfterSeparator(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()

	plan, err := BuildCodexLaunchPlan(start, home, []string{"--print-config", "--", "-d"})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !plan.PrintConfig {
		t.Fatalf("PrintConfig = false, want true")
	}
	wantArgs := []string{"-d"}
	if !reflect.DeepEqual(plan.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", plan.Args, wantArgs)
	}
	if len(plan.WrapperExpandedShortcuts) != 0 {
		t.Fatalf("WrapperExpandedShortcuts = %#v, want none", plan.WrapperExpandedShortcuts)
	}
}

func TestBuildCodexLaunchPlanFailsOnUnknownEnabledMCP(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	projectConfig := filepath.Join(start, ".agents", ".configs", "project-config.toml")
	mustWrite(t, projectConfig, "[mcp]\nenabled_servers = [\"missing\"]\n")

	_, err := BuildCodexLaunchPlan(start, home, nil)
	if err == nil {
		t.Fatal("expected unknown enabled MCP to fail")
	}
	if !strings.Contains(err.Error(), "MCP server \"missing\"") || !strings.Contains(err.Error(), projectConfig) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildCodexLaunchPlanLeavesPrimarySessionArgsNativeWhenUnconfigured(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()

	tests := []struct {
		name string
		args []string
	}{
		{name: "interactive", args: []string{"help me"}},
		{name: "exec", args: []string{"exec", "inspect this repo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := BuildCodexLaunchPlan(start, home, tt.args)
			if err != nil {
				t.Fatalf("BuildCodexLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.Args, tt.args) {
				t.Fatalf("Args = %#v, want unchanged %#v", plan.Args, tt.args)
			}
		})
	}
}

func TestBuildCodexLaunchPlanComposesPrimarySessionByFieldForInteractiveAndExec(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "parent-model"
reasoning_effort = "high"
yolo_mode = true
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	mustWrite(t, filepath.Join(child, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "child-model"
yolo_mode = false
`)
	parentConfig := filepath.Join(parent, ".agents", ".configs", projectConfigFileName)
	childConfig := filepath.Join(child, ".agents", ".configs", projectConfigFileName)

	prefix := []string{"--model", "child-model", "-c", `model_reasoning_effort="high"`}
	tests := []struct {
		name string
		args []string
	}{
		{name: "interactive", args: []string{"help me"}},
		{name: "exec", args: []string{"exec", "inspect this repo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := BuildCodexLaunchPlan(child, home, tt.args)
			if err != nil {
				t.Fatalf("BuildCodexLaunchPlan: %v", err)
			}
			want := append(append([]string(nil), prefix...), tt.args...)
			if !reflect.DeepEqual(plan.Args, want) {
				t.Fatalf("Args = %#v, want %#v", plan.Args, want)
			}
			if countArg(plan.Args, codexDangerouslyBypassApprovalsAndSandbox) != 0 {
				t.Fatalf("child yolo_mode=false did not mask parent true: %#v", plan.Args)
			}
			if got := plan.PrimarySession.Model; !got.Present || got.Value != "child-model" || got.Source != childConfig {
				t.Fatalf("composed model = %#v, want child value and provenance", got)
			}
			if got := plan.PrimarySession.ReasoningEffort; !got.Present || got.Value != "high" || got.Source != parentConfig {
				t.Fatalf("composed reasoning effort = %#v, want parent value and provenance", got)
			}
			if got := plan.PrimarySession.YoloMode; !got.Present || got.Value || got.Source != childConfig {
				t.Fatalf("composed yolo mode = %#v, want present child false and provenance", got)
			}
			resolution := plan.PrimarySessionResolution
			if got := resolution.Model; !got.EffectiveValueKnown || got.EffectiveValue != "child-model" || got.EffectiveSource != childConfig || got.ProjectApplication != CodexPrimarySessionApplied {
				t.Fatalf("model resolution = %#v, want applied child project value", got)
			}
			if got := resolution.ReasoningEffort; !got.EffectiveValueKnown || got.EffectiveValue != "high" || got.EffectiveSource != parentConfig || got.ProjectApplication != CodexPrimarySessionApplied {
				t.Fatalf("reasoning resolution = %#v, want applied parent project value", got)
			}
			if got := resolution.YoloMode; got.EffectiveValue || got.EffectiveSource != childConfig || !got.ProjectConfigured || got.ProjectValue || got.ProjectApplication != CodexPrimarySessionApplied {
				t.Fatalf("yolo resolution = %#v, want applied explicit child false", got)
			}
		})
	}
}

func TestRenderCodexLaunchPlanReportsPrimarySessionResolution(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "parent-model"
reasoning_effort = "high"
yolo_mode = true
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	mustWrite(t, filepath.Join(child, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "child-model"
yolo_mode = false
`)
	parentConfig := filepath.Join(parent, ".agents", ".configs", projectConfigFileName)
	childConfig := filepath.Join(child, ".agents", ".configs", projectConfigFileName)

	plan, err := BuildCodexLaunchPlan(child, home, []string{"--print-config", "exec", "inspect"})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	rendered := RenderCodexLaunchPlan(plan)
	for _, want := range []string{
		"project_configs:\n  - " + parentConfig,
		"  - " + childConfig,
		"primary_session:",
		"  model:\n    effective_value: \"child-model\"\n    effective_source: " + childConfig,
		"    project_value: \"child-model\"\n    project_source: " + childConfig + "\n    project_application: applied",
		"  reasoning_effort:\n    effective_value: \"high\"\n    effective_source: " + parentConfig,
		"  yolo_mode:\n    effective_value: false\n    effective_source: " + childConfig,
		"    project_value: false\n    project_source: " + childConfig + "\n    project_application: applied",
		"codex_args:\n  - \"--model\"\n  - \"child-model\"\n  - \"-c\"\n  - \"model_reasoning_effort=\\\"high\\\"\"",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan missing %q:\n%s", want, rendered)
		}
	}
}

func TestBuildCodexLaunchPlanPrimarySessionResolutionReportsExplicitSuppression(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	configPath := filepath.Join(start, ".agents", ".configs", projectConfigFileName)
	mustWrite(t, configPath, `
[agents.codex.primary_session]
model = "project-model"
reasoning_effort = "xhigh"
yolo_mode = false
`)

	plan, err := BuildCodexLaunchPlan(start, home, []string{
		"--model", "cli-model",
		"-c", `model_reasoning_effort="medium"`,
		"--yolo",
		"exec", "inspect",
	})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}

	if got := plan.PrimarySessionResolution.Model; !got.EffectiveValueKnown || got.EffectiveValue != "cli-model" || got.EffectiveSource != "cli:--model" || got.ProjectSource != configPath || got.ProjectApplication != CodexPrimarySessionSuppressedByCLI {
		t.Fatalf("model resolution = %#v, want explicit CLI suppression", got)
	}
	if got := plan.PrimarySessionResolution.ReasoningEffort; !got.EffectiveValueKnown || got.EffectiveValue != "medium" || got.EffectiveSource != "cli:-c model_reasoning_effort" || got.ProjectSource != configPath || got.ProjectApplication != CodexPrimarySessionSuppressedByCLI {
		t.Fatalf("reasoning resolution = %#v, want explicit CLI suppression", got)
	}
	if got := plan.PrimarySessionResolution.YoloMode; !got.EffectiveValue || got.EffectiveSource != "wrapper:--yolo" || got.ProjectValue || got.ProjectSource != configPath || got.ProjectApplication != CodexPrimarySessionSuppressedByCLI {
		t.Fatalf("yolo resolution = %#v, want wrapper CLI suppression", got)
	}

	rendered := RenderCodexLaunchPlan(plan)
	for _, want := range []string{
		"effective_value: \"cli-model\"\n    effective_source: cli:--model",
		"effective_value: \"medium\"\n    effective_source: cli:-c model_reasoning_effort",
		"effective_value: true\n    effective_source: wrapper:--yolo",
		"project_application: suppressed_by_explicit_cli",
		"wrapper_expansions:\n  - --yolo => " + codexDangerouslyBypassApprovalsAndSandbox,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan missing %q:\n%s", want, rendered)
		}
	}
}

func TestBuildCodexLaunchPlanPrimarySessionResolutionReportsProfileSuppression(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	configPath := filepath.Join(start, ".agents", ".configs", projectConfigFileName)
	mustWrite(t, configPath, `
[agents.codex.primary_session]
model = "project-model"
reasoning_effort = "high"
yolo_mode = true
`)

	plan, err := BuildCodexLaunchPlan(start, home, []string{"--profile", "fast", "exec", "inspect"})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	for field, got := range map[string]CodexPrimarySessionStringResolution{
		"model":            plan.PrimarySessionResolution.Model,
		"reasoning_effort": plan.PrimarySessionResolution.ReasoningEffort,
	} {
		if got.EffectiveValueKnown || got.EffectiveSource != "cli:--profile" || got.ProjectSource != configPath || got.ProjectApplication != CodexPrimarySessionSuppressedByProfile {
			t.Fatalf("%s resolution = %#v, want explicit profile suppression", field, got)
		}
	}
	if got := plan.PrimarySessionResolution.YoloMode; !got.EffectiveValue || got.EffectiveSource != configPath || got.ProjectApplication != CodexPrimarySessionApplied {
		t.Fatalf("yolo resolution = %#v, want profile-independent project yolo", got)
	}

	rendered := RenderCodexLaunchPlan(plan)
	if got := strings.Count(rendered, "project_application: suppressed_by_explicit_profile"); got != 2 {
		t.Fatalf("profile suppression count = %d, want 2:\n%s", got, rendered)
	}
	if !strings.Contains(rendered, "effective_value: (codex-native)\n    effective_source: cli:--profile") {
		t.Fatalf("rendered plan missing Codex-native profile evidence:\n%s", rendered)
	}
}

func TestBuildCodexLaunchPlanPrimarySessionResolutionReportsNativeDefaults(t *testing.T) {
	plan, err := BuildCodexLaunchPlan(t.TempDir(), t.TempDir(), []string{"--print-config", "exec", "inspect"})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	for field, got := range map[string]CodexPrimarySessionStringResolution{
		"model":            plan.PrimarySessionResolution.Model,
		"reasoning_effort": plan.PrimarySessionResolution.ReasoningEffort,
	} {
		if got.EffectiveValueKnown || got.EffectiveSource != "native" || got.ProjectConfigured || got.ProjectApplication != CodexPrimarySessionNotConfigured {
			t.Fatalf("%s resolution = %#v, want native/unconfigured", field, got)
		}
	}
	if got := plan.PrimarySessionResolution.YoloMode; got.EffectiveValue || got.EffectiveSource != "default" || got.ProjectConfigured || got.ProjectApplication != CodexPrimarySessionNotConfigured {
		t.Fatalf("yolo resolution = %#v, want safe default", got)
	}

	rendered := RenderCodexLaunchPlan(plan)
	for _, want := range []string{
		"project_configs:\n  - (none)",
		"effective_value: (codex-native)\n    effective_source: native",
		"  yolo_mode:\n    effective_value: false\n    effective_source: default",
		"project_value: (absent)\n    project_source: (none)\n    project_application: not_configured",
		"codex_args:\n  - \"exec\"\n  - \"inspect\"",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan missing %q:\n%s", want, rendered)
		}
	}
}

func TestBuildCodexLaunchPlanPrimarySessionExplicitPrecedence(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	mustWrite(t, filepath.Join(start, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "project-model"
reasoning_effort = "xhigh"
yolo_mode = true
`)

	projectModel := []string{"--model", "project-model"}
	projectEffort := []string{"-c", `model_reasoning_effort="xhigh"`}
	danger := []string{codexDangerouslyBypassApprovalsAndSandbox}
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "long model overrides model only",
			args: []string{"--model", "cli-model", "exec", "prompt"},
			want: joinArgs(projectEffort, danger, []string{"--model", "cli-model", "exec", "prompt"}),
		},
		{
			name: "short model overrides model only",
			args: []string{"exec", "-m", "cli-model", "prompt"},
			want: joinArgs(projectEffort, danger, []string{"exec", "-m", "cli-model", "prompt"}),
		},
		{
			name: "top level config model overrides model only",
			args: []string{"exec", "-c", `model="cli-model"`, "prompt"},
			want: joinArgs(projectEffort, danger, []string{"exec", "-c", `model="cli-model"`, "prompt"}),
		},
		{
			name: "top level config effort overrides effort only",
			args: []string{"-c", `model_reasoning_effort="medium"`, "exec", "prompt"},
			want: joinArgs(projectModel, danger, []string{"-c", `model_reasoning_effort="medium"`, "exec", "prompt"}),
		},
		{
			name: "nested config model does not override project model",
			args: []string{"-c", `profiles.fast.model="cli-model"`, "prompt"},
			want: joinArgs(projectModel, projectEffort, danger, []string{"-c", `profiles.fast.model="cli-model"`, "prompt"}),
		},
		{
			name: "profile suppresses model and effort but not yolo",
			args: []string{"--profile", "fast", "exec", "prompt"},
			want: joinArgs(danger, []string{"--profile", "fast", "exec", "prompt"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := BuildCodexLaunchPlan(start, home, tt.args)
			if err != nil {
				t.Fatalf("BuildCodexLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.Args, tt.want) {
				t.Fatalf("Args = %#v, want %#v", plan.Args, tt.want)
			}
		})
	}
}

func TestBuildCodexLaunchPlanNormalizesEqualExplicitSelections(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	mustWrite(t, filepath.Join(start, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "project-model"
reasoning_effort = "xhigh"
yolo_mode = false
`)

	args := []string{
		"--model", "cli-model",
		"-m", "cli-model",
		"-c", `model='cli-model'`,
		"-c", `model_reasoning_effort="high"`,
		"--config", `model_reasoning_effort='high'`,
		"exec", "prompt",
	}
	plan, err := BuildCodexLaunchPlan(start, home, args)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	want := []string{
		"--model", "cli-model",
		"-c", `model_reasoning_effort="high"`,
		"exec", "prompt",
	}
	if !reflect.DeepEqual(plan.Args, want) {
		t.Fatalf("Args = %#v, want normalized %#v", plan.Args, want)
	}
}

func TestBuildCodexLaunchPlanRejectsConflictingExplicitSelections(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantField string
	}{
		{
			name:      "model flags",
			args:      []string{"--model", "first", "-m", "second"},
			wantField: "model",
		},
		{
			name:      "model flag and top level config",
			args:      []string{"-c", `model="first"`, "--model", "second"},
			wantField: "model",
		},
		{
			name:      "reasoning config",
			args:      []string{"-c", `model_reasoning_effort="high"`, "--config", `model_reasoning_effort="medium"`},
			wantField: "model_reasoning_effort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildCodexLaunchPlan(t.TempDir(), t.TempDir(), tt.args)
			if err == nil {
				t.Fatal("expected conflicting explicit values to fail")
			}
			if !strings.Contains(err.Error(), "conflicting explicit Codex values") || !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("error = %q, want explicit conflict for %q", err, tt.wantField)
			}
		})
	}
}

func TestNormalizeCodexExplicitSelectionsSupportsNativeFormsAndBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		want          []string
		wantModel     bool
		wantReasoning bool
		wantProfile   bool
	}{
		{
			name: "equals forms and equal duplicates",
			args: []string{
				"--model=cli-model",
				"-m=cli-model",
				`-c=model='cli-model'`,
				`--config=model_reasoning_effort='high'`,
				`-c=model_reasoning_effort="high"`,
				"--profile=fast",
				"-p=fast",
				"prompt",
			},
			want: []string{
				"--model=cli-model",
				`--config=model_reasoning_effort='high'`,
				"--profile=fast",
				"-p=fast",
				"prompt",
			},
			wantModel:     true,
			wantReasoning: true,
			wantProfile:   true,
		},
		{
			name:        "separate short profile",
			args:        []string{"-p", "fast", "prompt"},
			want:        []string{"-p", "fast", "prompt"},
			wantProfile: true,
		},
		{
			name:      "missing model value remains native",
			args:      []string{"--model"},
			want:      []string{"--model"},
			wantModel: true,
		},
		{
			name: "missing config value remains native",
			args: []string{"-c"},
			want: []string{"-c"},
		},
		{
			name: "unrelated config forms remain native",
			args: []string{"--config", "features.search=true", "-c=features.other=false"},
			want: []string{"--config", "features.search=true", "-c=features.other=false"},
		},
		{
			name: "codex separator stops selection parsing",
			args: []string{"--", "--model", "prompt-text"},
			want: []string{"--", "--model", "prompt-text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, selections, err := normalizeCodexExplicitSelections(tt.args)
			if err != nil {
				t.Fatalf("normalizeCodexExplicitSelections: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalized args = %#v, want %#v", got, tt.want)
			}
			if selections.model != tt.wantModel || selections.reasoningEffort != tt.wantReasoning || selections.profile != tt.wantProfile {
				t.Fatalf("selections = %#v, want model=%t reasoning=%t profile=%t", selections, tt.wantModel, tt.wantReasoning, tt.wantProfile)
			}
		})
	}
}

func TestBuildCodexLaunchPlanEmitsExactlyOneDangerousFlag(t *testing.T) {
	tests := []struct {
		name          string
		primaryConfig string
		args          []string
		wantArgs      []string
		wantShortcuts int
	}{
		{
			name:          "project and duplicate explicit requests",
			primaryConfig: "yolo_mode = true\n",
			args:          []string{"-d", "--yolo", codexDangerouslyBypassApprovalsAndSandbox, "exec", "prompt"},
			wantArgs:      []string{codexDangerouslyBypassApprovalsAndSandbox, "exec", "prompt"},
			wantShortcuts: 2,
		},
		{
			name:          "explicit request without project yolo",
			primaryConfig: "yolo_mode = false\n",
			args:          []string{"--danger", "prompt"},
			wantArgs:      []string{codexDangerouslyBypassApprovalsAndSandbox, "prompt"},
			wantShortcuts: 1,
		},
		{
			name:          "false emits no flag",
			primaryConfig: "yolo_mode = false\n",
			args:          []string{"prompt"},
			wantArgs:      []string{"prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			start := t.TempDir()
			mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
			mustWrite(t, filepath.Join(start, ".agents", ".configs", projectConfigFileName), "[agents.codex.primary_session]\n"+tt.primaryConfig)

			plan, err := BuildCodexLaunchPlan(start, home, tt.args)
			if err != nil {
				t.Fatalf("BuildCodexLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.Args, tt.wantArgs) {
				t.Fatalf("Args = %#v, want %#v", plan.Args, tt.wantArgs)
			}
			if got := countArg(plan.Args, codexDangerouslyBypassApprovalsAndSandbox); got > 1 {
				t.Fatalf("dangerous flag count = %d in %#v, want at most one", got, plan.Args)
			}
			if len(plan.WrapperExpandedShortcuts) != tt.wantShortcuts {
				t.Fatalf("WrapperExpandedShortcuts = %#v, want %d", plan.WrapperExpandedShortcuts, tt.wantShortcuts)
			}
		})
	}
}

func TestBuildCodexLaunchPlanRejectsMalformedPrimarySessionConfig(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{
			name:      "empty model",
			body:      "[agents.codex.primary_session]\nmodel = \"\"\n",
			wantField: "agents.codex.primary_session.model",
		},
		{
			name:      "blank effort",
			body:      "[agents.codex.primary_session]\nreasoning_effort = \"   \"\n",
			wantField: "agents.codex.primary_session.reasoning_effort",
		},
		{
			name:      "wrong model type",
			body:      "[agents.codex.primary_session]\nmodel = 42\n",
			wantField: "agents.codex.primary_session.model",
		},
		{
			name:      "wrong yolo type",
			body:      "[agents.codex.primary_session]\nyolo_mode = \"true\"\n",
			wantField: "agents.codex.primary_session.yolo_mode",
		},
		{
			name:      "duplicate conflicting model",
			body:      "[agents.codex.primary_session]\nmodel = \"first\"\nmodel = \"second\"\n",
			wantField: "agents.codex.primary_session",
		},
		{
			name:      "empty primary table",
			body:      "[agents.codex.primary_session]\n",
			wantField: "agents.codex.primary_session",
		},
		{
			name:      "unsupported primary field",
			body:      "[agents.codex.primary_session]\nreasoning = \"high\"\n",
			wantField: "agents.codex.primary_session.reasoning",
		},
		{
			name:      "malformed table",
			body:      "[agents.codex.primary_session\nmodel = \"broken\"\n",
			wantField: "project_config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			start := t.TempDir()
			mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
			path := filepath.Join(start, ".agents", ".configs", projectConfigFileName)
			mustWrite(t, path, tt.body)

			_, err := BuildCodexLaunchPlan(start, home, nil)
			if err == nil {
				t.Fatal("expected malformed primary session config to fail")
			}
			if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("error = %q, want source path %q and field %q", err, path, tt.wantField)
			}
		})
	}
}

func TestBuildClaudeLaunchPlanDoesNotApplyCodexPrimarySession(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	mustWrite(t, filepath.Join(start, ".agents", ".configs", projectConfigFileName), `
[agents.codex.primary_session]
model = "codex-only"
reasoning_effort = "high"
yolo_mode = true
`)

	args := []string{"--resume"}
	plan, err := BuildClaudeLaunchPlan(start, home, args)
	if err != nil {
		t.Fatalf("BuildClaudeLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.Args, args) {
		t.Fatalf("Claude Args = %#v, want unchanged %#v", plan.Args, args)
	}
}

func countArg(args []string, want string) int {
	count := 0
	for _, arg := range args {
		if arg == want {
			count++
		}
	}
	return count
}

func joinArgs(parts ...[]string) []string {
	var joined []string
	for _, part := range parts {
		joined = append(joined, part...)
	}
	return joined
}
