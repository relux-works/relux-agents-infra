package infra

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildChildLaunchCompositionCodexIsDeterministicMCPOnlyAndSecretSafe(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	project := filepath.Join(parent, "child")
	parentConfigDir := filepath.Join(parent, ".agents", ".configs")
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, parentConfigDir)
	mustMkdir(t, configDir)
	parentConfig := filepath.Join(parentConfigDir, projectConfigFileName)
	projectConfig := filepath.Join(configDir, projectConfigFileName)
	parentRegistry := filepath.Join(parentConfigDir, "codex-mcp-servers.toml")
	registry := filepath.Join(configDir, "codex-mcp-servers.toml")
	mustWrite(t, parentConfig, `
[mcp]
enabled_servers = ["jira"]
`)
	mustWrite(t, projectConfig, `
[mcp]
enabled_servers = ["lldb", "jira"]

[agents.codex.primary_session]
model = "must-not-enter-child-composition"
reasoning_effort = "must-not-enter-child-composition"
yolo_mode = true

[agents.claude.primary_session]
model = "must-not-enter-child-composition"
yolo_mode = true
`)
	mustWrite(t, parentRegistry, `
[servers.jira]
url = "https://jira.example/mcp"
bearer_token_env_var = "JIRA_TOKEN"
`)
	mustWrite(t, registry, `
[servers.lldb]
command = "lldb-mcp"
args = ["--socket", "auto"]
`)
	t.Setenv("JIRA_TOKEN", "must-not-be-read-or-serialized")
	producer := ChildLaunchCompositionProducer{Version: "v1.7.0", Commit: "abc123"}

	first, err := BuildChildLaunchComposition("codex", project, home, producer)
	if err != nil {
		t.Fatalf("BuildChildLaunchComposition: %v", err)
	}
	second, err := BuildChildLaunchComposition("codex", project, home, producer)
	if err != nil {
		t.Fatalf("second BuildChildLaunchComposition: %v", err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first composition: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second composition: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("composition is not deterministic:\n%s\n%s", firstJSON, secondJSON)
	}

	wantArgs := []string{
		"-c", "mcp_servers.jira.url=\"https://jira.example/mcp\"",
		"-c", "mcp_servers.jira.bearer_token_env_var=\"JIRA_TOKEN\"",
		"-c", "mcp_servers.lldb.command=\"lldb-mcp\"",
		"-c", "mcp_servers.lldb.args=[\"--socket\", \"auto\"]",
	}
	if !reflect.DeepEqual(first.ArgvPrefix, wantArgs) {
		t.Fatalf("ArgvPrefix = %#v, want %#v", first.ArgvPrefix, wantArgs)
	}
	if !reflect.DeepEqual(first.RequiredEnvVars, []string{"JIRA_TOKEN"}) {
		t.Fatalf("RequiredEnvVars = %#v", first.RequiredEnvVars)
	}
	if got := first.MCPServers; len(got) != 2 || got[0].Name != "jira" || got[0].Transport != "http" || got[0].BearerTokenEnvVar != "JIRA_TOKEN" || got[1].Name != "lldb" || got[1].Transport != "stdio" {
		t.Fatalf("MCPServers = %#v", got)
	}
	canonicalParent := filepath.Dir(first.ProjectDir)
	canonicalParentConfig := filepath.Join(canonicalParent, ".agents", ".configs", projectConfigFileName)
	canonicalProjectConfig := filepath.Join(first.ProjectDir, ".agents", ".configs", projectConfigFileName)
	canonicalParentRegistry := filepath.Join(canonicalParent, ".agents", ".configs", "codex-mcp-servers.toml")
	canonicalRegistry := filepath.Join(first.ProjectDir, ".agents", ".configs", "codex-mcp-servers.toml")
	if !reflect.DeepEqual(first.Sources.ProjectConfigs, []string{canonicalParentConfig, canonicalProjectConfig}) || !reflect.DeepEqual(first.Sources.Registries, []string{canonicalParentRegistry, canonicalRegistry}) {
		t.Fatalf("Sources = %#v", first.Sources)
	}
	if !reflect.DeepEqual(first.MCPServers[0].EnabledBy, []string{canonicalParentConfig, canonicalProjectConfig}) {
		t.Fatalf("jira EnabledBy = %#v", first.MCPServers[0].EnabledBy)
	}
	for _, forbidden := range []string{
		"must-not-enter-child-composition",
		"must-not-be-read-or-serialized",
		`"primary_session"`,
		`"model"`,
		`"reasoning_effort"`,
		`"yolo_mode"`,
		`"user_args"`,
		"--model",
		"--dangerously-bypass-approvals-and-sandbox",
		"--dangerously-skip-permissions",
	} {
		if strings.Contains(string(firstJSON), forbidden) {
			t.Fatalf("composition leaked excluded value %q:\n%s", forbidden, firstJSON)
		}
	}
}

func TestBuildChildLaunchCompositionClaudeUsesOneMCPConfigArgument(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, projectConfigFileName), `
[mcp]
enabled_servers = ["jira", "lldb"]

[agents.claude.primary_session]
model = "excluded-primary-model"
yolo_mode = true
`)
	mustWrite(t, filepath.Join(configDir, "codex-mcp-servers.toml"), `
[servers.jira]
url = "https://jira.example/mcp"
bearer_token_env_var = "JIRA_TOKEN"

[servers.lldb]
command = "lldb-mcp"
args = ["--socket", "auto"]
`)

	composition, err := BuildChildLaunchComposition("claude", project, home, ChildLaunchCompositionProducer{Version: "dev", Commit: "unknown"})
	if err != nil {
		t.Fatalf("BuildChildLaunchComposition: %v", err)
	}
	repeated, err := BuildChildLaunchComposition("claude", project, home, ChildLaunchCompositionProducer{Version: "dev", Commit: "unknown"})
	if err != nil {
		t.Fatalf("repeated BuildChildLaunchComposition: %v", err)
	}
	if !reflect.DeepEqual(composition, repeated) {
		t.Fatalf("Claude composition is not deterministic:\n%#v\n%#v", composition, repeated)
	}
	if len(composition.ArgvPrefix) != 2 || composition.ArgvPrefix[0] != "--mcp-config" {
		t.Fatalf("ArgvPrefix = %#v, want exactly one --mcp-config pair", composition.ArgvPrefix)
	}
	var payload struct {
		MCPServers map[string]claudeMCPConfigEntry `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(composition.ArgvPrefix[1]), &payload); err != nil {
		t.Fatalf("decode Claude MCP payload: %v", err)
	}
	if got := payload.MCPServers["jira"]; got.Type != "http" || got.URL != "https://jira.example/mcp" || got.Headers["Authorization"] != "Bearer ${JIRA_TOKEN}" {
		t.Fatalf("jira payload = %#v", got)
	}
	if got := payload.MCPServers["lldb"]; got.Type != "stdio" || got.Command != "lldb-mcp" || !reflect.DeepEqual(got.Args, []string{"--socket", "auto"}) {
		t.Fatalf("lldb payload = %#v", got)
	}
	encoded, err := json.Marshal(composition)
	if err != nil {
		t.Fatalf("marshal composition: %v", err)
	}
	for _, forbidden := range []string{"excluded-primary-model", "--model", "--dangerously-skip-permissions"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("composition leaked excluded Claude primary field %q:\n%s", forbidden, encoded)
		}
	}
}

func TestBuildChildLaunchCompositionNoMCPUsesRequiredEmptyArraysForBothProviders(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			composition, err := BuildChildLaunchComposition(agent, project, home, ChildLaunchCompositionProducer{Version: "dev", Commit: "unknown"})
			if err != nil {
				t.Fatalf("BuildChildLaunchComposition: %v", err)
			}
			encoded, err := json.Marshal(composition)
			if err != nil {
				t.Fatalf("marshal composition: %v", err)
			}
			for _, want := range []string{`"argv_prefix":[]`, `"mcp_servers":[]`, `"required_env_vars":[]`, `"project_configs":[]`, `"registries":[]`} {
				if !strings.Contains(string(encoded), want) {
					t.Fatalf("no-MCP composition missing %s: %s", want, encoded)
				}
			}
		})
	}
}

func TestBuildChildLaunchCompositionRejectsInvalidProjectConfiguration(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, projectConfigFileName), "[mcp\nenabled_servers = [\"jira\"]\n")

	if _, err := BuildChildLaunchComposition("codex", project, home, ChildLaunchCompositionProducer{}); err == nil {
		t.Fatal("BuildChildLaunchComposition succeeded with invalid project configuration")
	}
}

// Removing the bundled registry distribution exposes project opt-ins with no
// definition anywhere. Composition must fail closed with the explicit
// missing-definition error — never silently drop the server or emit an empty
// success.
func TestBuildChildLaunchCompositionRejectsEnabledServerWithoutDefinition(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	configPath := filepath.Join(configDir, projectConfigFileName)
	mustWrite(t, configPath, "[mcp]\nenabled_servers = [\"figma\"]\n")

	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			_, err := BuildChildLaunchComposition(agent, project, home, ChildLaunchCompositionProducer{Version: "dev", Commit: "unknown"})
			if err == nil {
				t.Fatalf("BuildChildLaunchComposition(%s) succeeded with no figma definition", agent)
			}
			for _, want := range []string{`"figma"`, configPath, "no definition was found"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("missing-definition error = %q, want it to name %q", err, want)
				}
			}
		})
	}
}
