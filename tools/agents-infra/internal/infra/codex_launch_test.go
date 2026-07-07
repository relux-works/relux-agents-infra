package infra

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildCodexLaunchPlanComposesAncestorConfigsAndProvenance(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)

	mustMkdir(t, filepath.Join(home, ".agents", ".configs"))
	mustWrite(t, filepath.Join(home, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://global.example/figma\"\n")
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", "project-config.toml"), "[codex.mcp]\nenabled_servers = [\"figma\"]\n")
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://parent.example/figma\"\n")
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	mustWrite(t, filepath.Join(child, ".agents", ".configs", "project-config.toml"), "[codex.mcp]\nenabled_servers = [\"jira\", \"figma\"]\n")
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
	mustWrite(t, filepath.Join(home, ".agents", ".configs", "project-config.toml"), "[codex.mcp]\nenabled_servers = [\"figma\"]\n")

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
	mustWrite(t, filepath.Join(start, ".agents", ".configs", "project-config.toml"), "[codex.mcp]\nenabled_servers = [\"lldb\"]\n")

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
	mustWrite(t, filepath.Join(start, ".agents", ".configs", "project-config.toml"), "[codex.mcp]\nenabled_servers = [\"safari\"]\n")

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
	mustWrite(t, projectConfig, "[codex.mcp]\nenabled_servers = [\"missing\"]\n")

	_, err := BuildCodexLaunchPlan(start, home, nil)
	if err == nil {
		t.Fatal("expected unknown enabled MCP to fail")
	}
	if !strings.Contains(err.Error(), "MCP server \"missing\"") || !strings.Contains(err.Error(), projectConfig) {
		t.Fatalf("unexpected error: %v", err)
	}
}
