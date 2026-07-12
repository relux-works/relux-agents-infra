package infra

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseProjectConfigReadsTypedPrimarySessionAndMCP(t *testing.T) {
	path := "/project/.agents/.configs/project-config.toml"
	config, err := parseProjectConfig([]byte(`
[mcp]
enabled_servers = [
  "figma",
  "lldb",
]

[agents.codex.primary_session]
model = "gpt-project"
reasoning_effort = "xhigh"
yolo_mode = false

[agents.claude.primary_session]
model = "claude-project"

[unrelated]
preserved = "and ignored by this reader"
`), path)
	if err != nil {
		t.Fatalf("parseProjectConfig: %v", err)
	}
	if !reflect.DeepEqual(config.EnabledMCPServers, []string{"figma", "lldb"}) {
		t.Fatalf("EnabledMCPServers = %#v", config.EnabledMCPServers)
	}
	if config.PrimarySession.Model == nil || *config.PrimarySession.Model != "gpt-project" {
		t.Fatalf("Model = %#v", config.PrimarySession.Model)
	}
	if config.PrimarySession.ReasoningEffort == nil || *config.PrimarySession.ReasoningEffort != "xhigh" {
		t.Fatalf("ReasoningEffort = %#v", config.PrimarySession.ReasoningEffort)
	}
	if config.PrimarySession.YoloMode == nil || *config.PrimarySession.YoloMode {
		t.Fatalf("YoloMode = %#v, want present false", config.PrimarySession.YoloMode)
	}
	if config.ClaudePrimarySession.Model == nil || *config.ClaudePrimarySession.Model != "claude-project" {
		t.Fatalf("Claude model = %#v", config.ClaudePrimarySession.Model)
	}
}

func TestParseProjectConfigRejectsInvalidClaudePrimarySession(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{
			name:      "wrong claude table type",
			body:      "[agents]\nclaude = false",
			wantField: "agents.claude",
		},
		{
			name:      "empty model",
			body:      "[agents.claude.primary_session]\nmodel = \"  \"",
			wantField: claudePrimaryModelField,
		},
		{
			name:      "unsupported field",
			body:      "[agents.claude.primary_session]\nmodel = \"claude-opus-4-6\"\nyolo_mode = true",
			wantField: claudePrimarySessionField + ".yolo_mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/project/.agents/.configs/project-config.toml"
			_, err := parseProjectConfig([]byte(tt.body), path)
			if err == nil {
				t.Fatal("expected invalid Claude primary session to fail")
			}
			if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("error = %q, want path %q and field %q", err, path, tt.wantField)
			}
		})
	}
}

func TestParseProjectConfigRejectsWrongPrimarySessionTableTypes(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{name: "agents", body: `agents = "invalid"`, wantField: "agents"},
		{name: "codex", body: "[agents]\ncodex = false", wantField: "agents.codex"},
		{name: "primary session", body: "[agents.codex]\nprimary_session = []", wantField: codexPrimarySessionField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/project/.agents/.configs/project-config.toml"
			_, err := parseProjectConfig([]byte(tt.body), path)
			if err == nil {
				t.Fatal("expected wrong table type to fail")
			}
			if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("error = %q, want path %q and field %q", err, path, tt.wantField)
			}
		})
	}
}

func TestParseProjectConfigRejectsWrongMCPEnablementTypes(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "not an array", body: "[mcp]\nenabled_servers = \"figma\""},
		{name: "non-string element", body: "[mcp]\nenabled_servers = [\"figma\", 42]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/project/.agents/.configs/project-config.toml"
			_, err := parseProjectConfig([]byte(tt.body), path)
			if err == nil {
				t.Fatal("expected wrong MCP enablement type to fail")
			}
			if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "mcp.enabled_servers") {
				t.Fatalf("error = %q, want path %q and mcp.enabled_servers", err, path)
			}
		})
	}
}
