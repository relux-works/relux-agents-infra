package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

func TestRunComposeEmitsOneV1DocumentWithoutProviderExecutable(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), "[mcp]\nenabled_servers = [\"figma\"]\n")
	mustWrite(t, filepath.Join(configDir, "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://mcp.figma.com/mcp\"\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())

	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := run([]string{"compose", "--agent", agent, "--project", project, "--schema-version", "1", "--json"}); err != nil {
					t.Fatalf("run compose: %v", err)
				}
			})
			var composition infra.ChildLaunchComposition
			decodeSingleJSONDocument(t, output, &composition)
			if composition.Contract != infra.ChildLaunchCompositionContract || composition.SchemaVersion != 1 || composition.Status != "ok" || composition.Agent != agent {
				t.Fatalf("composition envelope = %#v", composition)
			}
			if composition.Producer.Version == "" || composition.Producer.Commit == "" {
				t.Fatalf("composition producer metadata = %#v", composition.Producer)
			}
			if agent == "codex" && !reflect.DeepEqual(composition.ArgvPrefix, []string{"-c", "mcp_servers.figma.url=\"https://mcp.figma.com/mcp\""}) {
				t.Fatalf("Codex ArgvPrefix = %#v", composition.ArgvPrefix)
			}
			if agent == "claude" && (len(composition.ArgvPrefix) != 2 || composition.ArgvPrefix[0] != "--mcp-config") {
				t.Fatalf("Claude ArgvPrefix = %#v", composition.ArgvPrefix)
			}
		})
	}
}

func TestRunComposeUnsupportedSchemaVersionEmitsSafeV1ErrorEnvelope(t *testing.T) {
	project := t.TempDir()
	var composeErr error
	output := captureStdout(t, func() {
		composeErr = runCompose([]string{"--agent", "claude", "--project", project, "--schema-version", "2", "--json"})
	})
	if composeErr == nil {
		t.Fatal("runCompose succeeded for unsupported schema version")
	}
	var envelope infra.ChildLaunchCompositionErrorEnvelope
	decodeSingleJSONDocument(t, output, &envelope)
	if envelope.Contract != infra.ChildLaunchCompositionContract || envelope.SchemaVersion != 1 || envelope.Status != "error" || envelope.Agent != "claude" || envelope.Error.Code != "unsupported_schema_version" {
		t.Fatalf("error envelope = %#v", envelope)
	}
}

func TestRunComposeInvalidConfigEmitsSafeErrorEnvelope(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), "[mcp\nenabled_servers = [\"figma\"]\n")
	t.Setenv("HOME", home)

	var composeErr error
	output := captureStdout(t, func() {
		composeErr = runCompose([]string{"--agent", "codex", "--project", project, "--schema-version", "1", "--json"})
	})
	if composeErr == nil {
		t.Fatal("runCompose succeeded with invalid config")
	}
	var envelope infra.ChildLaunchCompositionErrorEnvelope
	decodeSingleJSONDocument(t, output, &envelope)
	if envelope.Error.Code != "invalid_project_configuration" {
		t.Fatalf("error envelope = %#v", envelope)
	}
	if strings.Contains(output, "parse TOML") || strings.Contains(output, "enabled_servers") {
		t.Fatalf("machine error envelope leaked human diagnostics: %s", output)
	}
}

func TestRunAttachmentsUsesCallerCWDEnv(t *testing.T) {
	project := t.TempDir()
	other := t.TempDir()
	manifestDir := filepath.Join(project, ".temp")
	mustMkdir(t, manifestDir)
	manifestPath := filepath.Join(manifestDir, "agents-attachments-manifest.json")
	mustWrite(t, manifestPath, `{"attachments":[{"id":"photo","name":"photo.png","mime_type":"image/png","local_path":"/tmp/photo.png"}]}`)
	t.Setenv(callerCWDEnv, project)

	originalCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(other); err != nil {
		t.Fatalf("Chdir(%s): %v", other, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalCWD)
	})

	output := captureStdout(t, func() {
		if err := runAttachments([]string{"path", "photo"}); err != nil {
			t.Fatalf("runAttachments: %v", err)
		}
	})
	if strings.TrimSpace(output) != "/tmp/photo.png" {
		t.Fatalf("output = %q", output)
	}
}

func TestAgentsInfraModuleHasNoTaskBoardDependency(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(data), "task-board") || strings.Contains(string(data), "skill-project-management") {
		t.Fatalf("go.mod contains a task-board dependency:\n%s", data)
	}
}

func TestRunCodexPrintConfigUsesCallerCWDEnv(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	appDir := filepath.Join(project, "apps", "mobile", "app")
	mustMkdir(t, appDir)
	mustMkdir(t, filepath.Join(appDir, ".agents", ".configs"))
	mustWrite(t, filepath.Join(appDir, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"figma\"]\n")
	mustWrite(t, filepath.Join(appDir, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://mcp.figma.com/mcp\"\n")

	t.Setenv("HOME", home)
	t.Setenv(callerCWDEnv, appDir)

	output := captureStdout(t, func() {
		if err := runCodex([]string{"--print-config"}); err != nil {
			t.Fatalf("runCodex: %v", err)
		}
	})

	for _, want := range []string{
		"cwd: " + appDir,
		"enabled_mcp=figma",
		"enabled_mcp:\n  - figma",
		"mcp_servers.figma.url=",
		"https://mcp.figma.com/mcp",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("print-config missing %q:\n%s", want, output)
		}
	}
}

func TestRunCodexPrintConfigEmitsSafariMCPCommandAndArgs(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	appDir := filepath.Join(project, "apps", "web")
	safariCommand := "/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver"
	mustMkdir(t, appDir)
	mustMkdir(t, filepath.Join(appDir, ".agents", ".configs"))
	mustWrite(t, filepath.Join(appDir, ".agents", ".configs", "project-config.toml"), "[mcp]\nenabled_servers = [\"safari\"]\n")
	mustWrite(t, filepath.Join(appDir, ".agents", ".configs", "codex-mcp-servers.toml"), "[servers.safari]\ncommand = \""+safariCommand+"\"\nargs = [\"--mcp\"]\n")

	t.Setenv("HOME", home)
	t.Setenv(callerCWDEnv, appDir)

	output := captureStdout(t, func() {
		if err := runCodex([]string{"--print-config"}); err != nil {
			t.Fatalf("runCodex: %v", err)
		}
	})

	for _, want := range []string{
		"cwd: " + appDir,
		"enabled_mcp=safari",
		"enabled_mcp:\n  - safari",
		"command: " + safariCommand,
		"args: [\"--mcp\"]",
		"mcp_servers.safari.command=\\\"/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver\\\"",
		"mcp_servers.safari.args=[\\\"--mcp\\\"]",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("print-config missing %q:\n%s", want, output)
		}
	}
}

func TestRunCodexPrintConfigReportsPrimarySessionDiagnosticsWithoutLaunching(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	parentConfig := filepath.Join(parent, ".agents", ".configs", "project-config.toml")
	mustWrite(t, parentConfig, `
[agents.codex.primary_session]
model = "parent-model"
reasoning_effort = "high"
yolo_mode = true
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	childConfig := filepath.Join(child, ".agents", ".configs", "project-config.toml")
	mustWrite(t, childConfig, `
[agents.codex.primary_session]
model = "child-model"
yolo_mode = false
`)
	t.Setenv("HOME", home)
	t.Setenv(callerCWDEnv, child)

	output := captureStdout(t, func() {
		if err := runCodex([]string{
			"--print-config",
			"--model", "cli-model",
			"--profile", "fast",
			"--yolo",
			"exec", "inspect",
		}); err != nil {
			t.Fatalf("runCodex: %v", err)
		}
	})

	for _, want := range []string{
		"project_configs:\n  - " + parentConfig,
		"  - " + childConfig,
		"effective_value: \"cli-model\"\n    effective_source: cli:--model",
		"project_value: \"child-model\"\n    project_source: " + childConfig + "\n    project_application: suppressed_by_explicit_cli",
		"  reasoning_effort:\n    effective_value: (codex-native)\n    effective_source: cli:--profile",
		"project_value: \"high\"\n    project_source: " + parentConfig + "\n    project_application: suppressed_by_explicit_profile",
		"  yolo_mode:\n    effective_value: true\n    effective_source: wrapper:--yolo",
		"project_value: false\n    project_source: " + childConfig + "\n    project_application: suppressed_by_explicit_cli",
		"wrapper_expansions:\n  - --yolo => --dangerously-bypass-approvals-and-sandbox",
		"codex_args:\n  - \"--dangerously-bypass-approvals-and-sandbox\"\n  - \"--model\"\n  - \"cli-model\"\n  - \"--profile\"\n  - \"fast\"\n  - \"exec\"\n  - \"inspect\"",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("print-config missing %q:\n%s", want, output)
		}
	}
}

func TestRunClaudePrintConfigReportsIndependentPrimarySessionDiagnostics(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	mustWrite(t, filepath.Join(parent, ".agents", ".configs", "project-config.toml"), `
[agents.codex.primary_session]
model = "codex-parent"
reasoning_effort = "xhigh"
yolo_mode = true

[agents.claude.primary_session]
model = "claude-parent"
yolo_mode = true
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	childConfig := filepath.Join(child, ".agents", ".configs", "project-config.toml")
	mustWrite(t, childConfig, `
[agents.codex.primary_session]
model = "codex-child"
yolo_mode = false

[agents.claude.primary_session]
model = "claude-child"
yolo_mode = false
`)
	t.Setenv("HOME", home)
	t.Setenv(callerCWDEnv, child)

	output := captureStdout(t, func() {
		if err := runClaude([]string{"--print-config", "--model", "claude-cli", "-p", "inspect"}); err != nil {
			t.Fatalf("runClaude: %v", err)
		}
	})
	for _, want := range []string{
		"project_configs:\n  - " + filepath.Join(parent, ".agents", ".configs", "project-config.toml"),
		"  - " + childConfig,
		"effective_value: \"claude-cli\"\n    effective_source: cli:--model",
		"project_value: \"claude-child\"\n    project_source: " + childConfig + "\n    project_application: suppressed_by_explicit_cli",
		"  yolo_mode:\n    effective_value: false\n    effective_source: " + childConfig,
		"    project_value: false\n    project_source: " + childConfig + "\n    project_application: applied",
		"claude_args:\n  - \"--model\"\n  - \"claude-cli\"\n  - \"-p\"\n  - \"inspect\"",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("print-config missing %q:\n%s", want, output)
		}
	}
	for _, unexpected := range []string{"codex-child", "xhigh", "--dangerously-bypass-approvals-and-sandbox"} {
		if strings.Contains(output, unexpected) {
			t.Fatalf("Claude print-config leaked Codex policy %q:\n%s", unexpected, output)
		}
	}
}

func TestRunDoctorLocalReportsComposedPrimarySessionDiagnostics(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, filepath.Join(parent, ".agents", ".configs"))
	parentConfig := filepath.Join(parent, ".agents", ".configs", "project-config.toml")
	mustWrite(t, parentConfig, `
[mcp]
enabled_servers = ["figma"]

[agents.codex.primary_session]
model = "parent-model"
yolo_mode = true

[agents.claude.primary_session]
model = "claude-parent"
yolo_mode = true
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	childConfig := filepath.Join(child, ".agents", ".configs", "project-config.toml")
	mustWrite(t, childConfig, `
[mcp]
enabled_servers = ["lldb", "figma"]

[agents.codex.primary_session]
reasoning_effort = "xhigh"
yolo_mode = false

[agents.claude.primary_session]
model = "claude-child"
yolo_mode = false
`)
	mustMkdir(t, filepath.Join(child, ".codex"))
	mustWrite(t, filepath.Join(child, ".codex", "config.toml"), "model = \"legacy-local\"\n")
	t.Setenv("HOME", home)

	output := captureStdout(t, func() {
		if err := runDoctor([]string{"local", child}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})
	fields := parseKeyValueOutput(output)
	want := map[string]string{
		"codex_mcp_enabled":                     "figma,lldb",
		"codex_config_shadowing_global":         "true",
		"codex_primary_config_valid":            "true",
		"codex_primary_model":                   "parent-model",
		"codex_primary_model_source":            parentConfig,
		"codex_primary_reasoning_effort":        "xhigh",
		"codex_primary_reasoning_effort_source": childConfig,
		"codex_primary_yolo_mode":               "false",
		"codex_primary_yolo_mode_source":        childConfig,
		"claude_primary_config_valid":           "true",
		"claude_primary_model":                  "claude-child",
		"claude_primary_model_source":           childConfig,
		"claude_primary_yolo_mode":              "false",
		"claude_primary_yolo_mode_source":       childConfig,
	}
	for key, wantValue := range want {
		if got := fields[key]; got != wantValue {
			t.Fatalf("%s = %q, want %q:\n%s", key, got, wantValue, output)
		}
	}
}

func TestRunDoctorLocalReportsAbsentPrimarySessionDefaults(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)

	output := captureStdout(t, func() {
		if err := runDoctor([]string{"local", project}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})
	fields := parseKeyValueOutput(output)
	want := map[string]string{
		"codex_primary_config_valid":            "true",
		"codex_primary_model":                   "",
		"codex_primary_model_source":            "native",
		"codex_primary_reasoning_effort":        "",
		"codex_primary_reasoning_effort_source": "native",
		"codex_primary_yolo_mode":               "false",
		"codex_primary_yolo_mode_source":        "default",
		"claude_primary_config_valid":           "true",
		"claude_primary_model":                  "",
		"claude_primary_model_source":           "native",
		"claude_primary_yolo_mode":              "false",
		"claude_primary_yolo_mode_source":       "default",
	}
	for key, wantValue := range want {
		if got := fields[key]; got != wantValue {
			t.Fatalf("%s = %q, want %q:\n%s", key, got, wantValue, output)
		}
	}
}

func TestRunDoctorLocalReportsRenderedProjectSafeCodexConfig(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".codex"))
	mustWrite(t, filepath.Join(project, ".codex", "config.toml"), "# Generated by agents-infra; do not edit directly.\nmodel = \"gpt-5.6-terra\"\n")
	t.Setenv("HOME", home)

	output := captureStdout(t, func() {
		if err := runDoctor([]string{"local", project}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})
	fields := parseKeyValueOutput(output)
	want := map[string]string{
		"codex_config_present":          "true",
		"codex_config_linked":           "false",
		"codex_config_generated":        "true",
		"codex_config_effective":        "project-local",
		"codex_config_shadowing_global": "true",
	}
	for key, wantValue := range want {
		if got := fields[key]; got != wantValue {
			t.Fatalf("%s = %q, want %q:\n%s", key, got, wantValue, output)
		}
	}
	if !strings.Contains(output, "rendered from the installed Codex config without user-level profiles") {
		t.Fatalf("doctor action does not explain the managed project-safe config:\n%s", output)
	}
}

func TestRunDoctorLocalFailsClosedOnInvalidPrimarySessionConfig(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustWrite(t, configPath, `
[agents.codex.primary_session]
yolo_mode = "false"
`)
	t.Setenv("HOME", home)

	var doctorErr error
	output := captureStdout(t, func() {
		doctorErr = runDoctor([]string{"local", project})
	})
	if doctorErr == nil {
		t.Fatal("runDoctor succeeded with invalid project config")
	}
	if !strings.Contains(doctorErr.Error(), configPath) || !strings.Contains(doctorErr.Error(), "agents.codex.primary_session.yolo_mode") {
		t.Fatalf("runDoctor error = %q, want source path and field", doctorErr)
	}
	fields := parseKeyValueOutput(output)
	if got := fields["codex_primary_config_valid"]; got != "false" {
		t.Fatalf("codex_primary_config_valid = %q, want false:\n%s", got, output)
	}
	if _, ok := fields["codex_primary_model"]; ok {
		t.Fatalf("invalid config should not emit partial primary values:\n%s", output)
	}
}

func TestRunDoctorLocalReportsInvalidClaudePrimarySession(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustWrite(t, configPath, `
[agents.codex.primary_session]
model = "gpt-5.6-terra"

[agents.claude.primary_session]
model = ""
`)
	t.Setenv("HOME", home)

	var doctorErr error
	output := captureStdout(t, func() {
		doctorErr = runDoctor([]string{"local", project})
	})
	if doctorErr == nil || !strings.Contains(doctorErr.Error(), configPath) || !strings.Contains(doctorErr.Error(), "agents.claude.primary_session.model") {
		t.Fatalf("runDoctor error = %q, want Claude field validation", doctorErr)
	}
	fields := parseKeyValueOutput(output)
	if got := fields["claude_primary_config_valid"]; got != "false" {
		t.Fatalf("claude_primary_config_valid = %q, want false:\n%s", got, output)
	}
	if _, ok := fields["claude_primary_model"]; ok {
		t.Fatalf("invalid Claude config should not emit partial Claude values:\n%s", output)
	}
}

func TestRunDoctorLocalFailsClosedOnMalformedProjectTOML(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustWrite(t, configPath, "[agents.codex.primary_session\nmodel = \"broken\"\n")
	t.Setenv("HOME", home)

	var doctorErr error
	output := captureStdout(t, func() {
		doctorErr = runDoctor([]string{"local", project})
	})
	if doctorErr == nil {
		t.Fatal("runDoctor succeeded with malformed project TOML")
	}
	if !strings.Contains(doctorErr.Error(), configPath) || !strings.Contains(doctorErr.Error(), "field project_config") {
		t.Fatalf("runDoctor error = %q, want source path and parse field", doctorErr)
	}
	fields := parseKeyValueOutput(output)
	if got := fields["codex_primary_config_valid"]; got != "false" {
		t.Fatalf("codex_primary_config_valid = %q, want false:\n%s", got, output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stdout = write
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := write.Close(); err != nil {
		t.Fatalf("Close stdout pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, read); err != nil {
		t.Fatalf("Copy stdout pipe: %v", err)
	}
	if err := read.Close(); err != nil {
		t.Fatalf("Close stdout pipe reader: %v", err)
	}
	return buf.String()
}

func decodeSingleJSONDocument(t *testing.T, output string, destination any) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	if err := decoder.Decode(destination); err != nil {
		t.Fatalf("decode JSON document: %v\n%s", err, output)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("stdout contained more than one JSON document: err=%v extra=%#v\n%s", err, extra, output)
	}
}

func parseKeyValueOutput(output string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[key] = strings.TrimPrefix(value, " ")
	}
	return fields
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func TestRunComposePrimarySessionEmitsOneV1DocumentWithParity(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	binDir := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), `
[mcp]
enabled_servers = ["figma"]

[agents.codex.primary_session]
model = "gpt-5.2-codex"
reasoning_effort = "high"
yolo_mode = true

[agents.claude.primary_session]
model = "claude-opus-4-8"
yolo_mode = true
`)
	mustWrite(t, filepath.Join(configDir, "codex-mcp-servers.toml"), "[servers.figma]\nurl = \"https://mcp.figma.com/mcp\"\n")
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	mustWrite(t, filepath.Join(binDir, "claude"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := run([]string{"compose", "--mode", "primary-session", "--agent", provider, "--project", project, "--schema-version", "1", "--json", "--", "--continue"}); err != nil {
					t.Fatalf("run compose primary-session: %v", err)
				}
			})
			var plan infra.PrimarySessionLaunchPlan
			decodeSingleJSONDocument(t, output, &plan)
			if plan.Contract != infra.PrimarySessionLaunchPlanContract || plan.SchemaVersion != 1 || plan.Status != "ok" || plan.Provider != provider {
				t.Fatalf("plan envelope = %#v", plan)
			}
			if plan.Producer.Version == "" || plan.Producer.Commit == "" {
				t.Fatalf("plan producer metadata = %#v", plan.Producer)
			}
			if plan.Executable != filepath.Join(binDir, provider) {
				t.Fatalf("Executable = %q, want %q", plan.Executable, filepath.Join(binDir, provider))
			}
			var launchArgs []string
			if provider == "codex" {
				launch, err := infra.BuildCodexLaunchPlan(plan.ProjectDir, home, []string{"--continue"})
				if err != nil {
					t.Fatalf("BuildCodexLaunchPlan: %v", err)
				}
				launchArgs = launch.Args
				if plan.LaunchVariants.ManagedHost.Kind != infra.PrimarySessionManagedHostKindCodexAppServer {
					t.Fatalf("managed host kind = %q", plan.LaunchVariants.ManagedHost.Kind)
				}
			} else {
				launch, err := infra.BuildClaudeLaunchPlan(plan.ProjectDir, home, []string{"--continue"})
				if err != nil {
					t.Fatalf("BuildClaudeLaunchPlan: %v", err)
				}
				launchArgs = launch.Args
				if plan.LaunchVariants.ManagedHost.Kind != infra.PrimarySessionManagedHostKindClaudePTY {
					t.Fatalf("managed host kind = %q", plan.LaunchVariants.ManagedHost.Kind)
				}
			}
			if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launchArgs) {
				t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launchArgs)
			}
			if plan.Resolved.Model.Value == nil || !plan.Resolved.Yolo.Value {
				t.Fatalf("resolved policy = %#v", plan.Resolved)
			}
		})
	}
}

func TestRunPiPrintConfigAndComposeAreNonLaunchingAndRedacted(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	runtimeSentinel := filepath.Join(t.TempDir(), "runtime-sentinel")
	mustWrite(t, runtimeSentinel, "#!/bin/sh\necho launched > \"$0.launched\"\n")
	if err := os.Chmod(runtimeSentinel, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), mainTestPiConfig(runtimeSentinel, 18021))
	piRoot := mainTestOfficialPiAsset(t)
	t.Setenv("PATH", piRoot)
	t.Setenv("HOME", home)
	t.Setenv(callerCWDEnv, project)

	printOutput := captureStdout(t, func() {
		if err := runPi([]string{"--print-config", "--api-key=secret", "--", "hello"}); err != nil {
			t.Fatalf("runPi print: %v", err)
		}
	})
	composeOutput := captureStdout(t, func() {
		if err := run([]string{"compose", "--mode", "primary-session", "--agent", "pi", "--project", project, "--schema-version", "1", "--json", "--", "--api-key=secret", "--", "hello"}); err != nil {
			t.Fatalf("compose Pi: %v", err)
		}
	})
	var printPlan, composePlan infra.PrimarySessionLaunchPlan
	decodeSingleJSONDocument(t, printOutput, &printPlan)
	decodeSingleJSONDocument(t, composeOutput, &composePlan)
	if printPlan.Provider != "pi" || printPlan.Pi == nil || !printPlan.Pi.Managed || printPlan.LaunchVariants.ManagedHost.Kind != infra.PrimarySessionManagedHostKindPiPTY {
		t.Fatalf("Pi plan=%#v", printPlan)
	}
	if strings.Contains(printOutput, "secret") || strings.Contains(composeOutput, "secret") {
		t.Fatal("Pi diagnostics leaked --api-key")
	}
	if !reflect.DeepEqual(printPlan.LaunchVariants, composePlan.LaunchVariants) || printPlan.Pi.ModelsJSON.SHA256 != composePlan.Pi.ModelsJSON.SHA256 {
		t.Fatalf("print/compose parity mismatch")
	}
	if _, err := os.Stat(runtimeSentinel + ".launched"); !os.IsNotExist(err) {
		t.Fatalf("diagnostics launched runtime: %v", err)
	}
	for _, path := range []string{printPlan.Pi.State.Lock, printPlan.Pi.State.ModelsJSON} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("diagnostics created %s: %v", path, err)
		}
	}
	if len(printPlan.Pi.Capabilities.Verified) != 0 || printPlan.Pi.Capabilities.Verification != "not-claimed" {
		t.Fatalf("capability overclaim: %#v", printPlan.Pi.Capabilities)
	}
}

// Production call site: runCompose -> BuildPrimarySessionLaunchPlan ->
// buildPiPrimarySessionLaunchPlan. These cases must fail if runtime argv is no
// longer bound to the exact endpoint declared by base_url.
func TestRunComposePiRefusesRuntimeEndpointDivergence(t *testing.T) {
	piRoot := mainTestOfficialPiAsset(t)
	base := mainTestPiConfig("/bin/echo", 18021)
	tests := []struct {
		name       string
		config     string
		wantReject bool
	}{
		{name: "exact loopback endpoint control", config: base},
		{name: "wildcard runtime bind", config: strings.Replace(base, `"--host", "127.0.0.1"`, `"--host", "0.0.0.0"`, 1), wantReject: true},
		{name: "runtime port drift", config: strings.Replace(base, `"--port", "18021"`, `"--port", "19021"`, 1), wantReject: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, home := t.TempDir(), t.TempDir()
			mustMkdir(t, filepath.Join(home, "Library", "Caches"))
			configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
			mustMkdir(t, filepath.Dir(configPath))
			mustWrite(t, configPath, test.config)
			t.Setenv("HOME", home)
			t.Setenv("PATH", piRoot)

			var composeErr error
			output := captureStdout(t, func() {
				composeErr = runCompose([]string{"--mode", "primary-session", "--agent", "pi", "--project", project, "--schema-version", "1", "--json"})
			})
			if test.wantReject {
				if composeErr == nil {
					t.Fatal("production compose admitted runtime/base_url endpoint divergence")
				}
				var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
				decodeSingleJSONDocument(t, output, &envelope)
				if envelope.Error.Code != infra.PrimarySessionErrorInvalidProjectConfiguration {
					t.Fatalf("endpoint divergence error=%#v", envelope.Error)
				}
				return
			}
			if composeErr != nil {
				t.Fatalf("exact endpoint control rejected: %v\n%s", composeErr, output)
			}
			var plan infra.PrimarySessionLaunchPlan
			decodeSingleJSONDocument(t, output, &plan)
			wantArgv := []string{"serve", "--model", "Model", "--host", "127.0.0.1", "--port", "18021"}
			if plan.Pi == nil || plan.Pi.Runtime == nil || plan.Pi.Runtime.Endpoint != "http://127.0.0.1:18021/v1" || !reflect.DeepEqual(plan.Pi.Runtime.Argv, wantArgv) {
				t.Fatalf("exact endpoint control plan=%#v", plan.Pi)
			}
		})
	}
}

func TestRunPiPrintConfigRejectsBareUnknownLongOption(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	runtimeSentinel := filepath.Join(t.TempDir(), "runtime-sentinel")
	mustWrite(t, runtimeSentinel, "#!/bin/sh\necho launched > \"$0.launched\"\n")
	if err := os.Chmod(runtimeSentinel, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), mainTestPiConfig(runtimeSentinel, 18022))
	t.Setenv("PATH", mainTestOfficialPiAsset(t))
	t.Setenv("HOME", home)
	t.Setenv(callerCWDEnv, project)

	if err := runPi([]string{"--print-config", "--unknown"}); err == nil {
		t.Fatal("production runPi admitted a forbidden bare unknown long option")
	}
	if _, err := os.Stat(runtimeSentinel + ".launched"); !os.IsNotExist(err) {
		t.Fatalf("rejected argv launched runtime: %v", err)
	}

	for _, args := range [][]string{{"--print-config", "--custom-extension=value"}, {"--print-config", "--custom-extension", "value"}} {
		output := captureStdout(t, func() {
			if err := runPi(args); err != nil {
				t.Fatalf("complete unknown option form %q: %v", args, err)
			}
		})
		var plan infra.PrimarySessionLaunchPlan
		decodeSingleJSONDocument(t, output, &plan)
		joined := strings.Join(plan.LaunchVariants.Interactive.Argv, "\x00")
		if !strings.Contains(joined, strings.Join(args[1:], "\x00")) {
			t.Fatalf("complete unknown option form was not preserved: args=%q argv=%q", args, plan.LaunchVariants.Interactive.Argv)
		}
	}
}

func TestRunPiRejectsManagedSessionPathOverridesBeforeSideEffects(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	runtimeSentinel := filepath.Join(t.TempDir(), "runtime-sentinel")
	mustWrite(t, runtimeSentinel, "#!/bin/sh\necho launched > \"$0.launched\"\n")
	if err := os.Chmod(runtimeSentinel, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), mainTestPiConfig(runtimeSentinel, 18023))
	globalSessions := filepath.Join(home, ".pi", "agent", "sessions")
	mustMkdir(t, globalSessions)
	globalSession := filepath.Join(globalSessions, "sentinel.jsonl")
	const sentinel = "global-session-sentinel\n"
	mustWrite(t, globalSession, sentinel)
	t.Setenv("PATH", mainTestOfficialPiAsset(t))
	t.Setenv("HOME", home)
	t.Setenv(callerCWDEnv, project)

	for name, args := range map[string][]string{
		"session directory":          {"--session-dir", globalSessions},
		"absolute session path":      {"--session", globalSession},
		"absolute fork path":         {"--fork", globalSession},
		"jsonl session path":         {"--session", "sentinel.jsonl"},
		"backslash-shaped fork path": {"--fork", `..\global-session`},
		"global session export":      {"--export", globalSession},
	} {
		t.Run(name, func(t *testing.T) {
			err := runPi(args)
			var launchErr *infra.PiLaunchError
			if !errors.As(err, &launchErr) || launchErr.Code != "invalid_provider_arguments" {
				t.Fatalf("production runPi(%q) error=%v", args, err)
			}
			if _, err := os.Stat(runtimeSentinel + ".launched"); !os.IsNotExist(err) {
				t.Fatalf("rejected session override launched runtime: %v", err)
			}
			got, err := os.ReadFile(globalSession)
			if err != nil || string(got) != sentinel {
				t.Fatalf("global session sentinel changed: bytes=%q err=%v", got, err)
			}
			entries, err := os.ReadDir(cache)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("session override refusal created managed state: %v", entries)
			}
		})
	}
}

func TestRunPiPrintConfigKeepsManagedContinueAndResumeContained(t *testing.T) {
	for _, mode := range []string{"--continue", "--resume"} {
		t.Run(mode, func(t *testing.T) {
			project, home := t.TempDir(), t.TempDir()
			mustMkdir(t, filepath.Join(home, "Library", "Caches"))
			configDir := filepath.Join(project, ".agents", ".configs")
			mustMkdir(t, configDir)
			runtimeSentinel := filepath.Join(t.TempDir(), "runtime-sentinel")
			mustWrite(t, runtimeSentinel, "#!/bin/sh\necho launched > \"$0.launched\"\n")
			if err := os.Chmod(runtimeSentinel, 0o755); err != nil {
				t.Fatal(err)
			}
			mustWrite(t, filepath.Join(configDir, "project-config.toml"), mainTestPiConfig(runtimeSentinel, 18024))
			t.Setenv("PATH", mainTestOfficialPiAsset(t))
			t.Setenv("HOME", home)
			t.Setenv(callerCWDEnv, project)

			output := captureStdout(t, func() {
				if err := runPi([]string{"--print-config", mode}); err != nil {
					t.Fatalf("runPi print %s: %v", mode, err)
				}
			})
			var plan infra.PrimarySessionLaunchPlan
			decodeSingleJSONDocument(t, output, &plan)
			if plan.Pi == nil || plan.Pi.State == nil || plan.Pi.State.SessionsDir == "" {
				t.Fatalf("managed state missing: %#v", plan.Pi)
			}
			if !strings.Contains(plan.Pi.State.SessionsDir, filepath.Join("agents-infra", "pi", plan.Pi.State.ProjectStateKey, plan.Pi.State.ProfileStateKey, "sessions")) {
				t.Fatalf("session path is not hash-contained: %q", plan.Pi.State.SessionsDir)
			}
			if !slices.Contains(plan.LaunchVariants.Interactive.Argv, mode) || slices.Contains(plan.LaunchVariants.Interactive.Argv, "--session-dir") {
				t.Fatalf("managed %s argv=%q", mode, plan.LaunchVariants.Interactive.Argv)
			}
			if _, err := os.Stat(runtimeSentinel + ".launched"); !os.IsNotExist(err) {
				t.Fatalf("print config launched runtime: %v", err)
			}
		})
	}
}

func TestRunPiNativePassthroughOnlyOnPolicyAbsence(t *testing.T) {
	project, bin := t.TempDir(), t.TempDir()
	record := filepath.Join(t.TempDir(), "record")
	pi := filepath.Join(bin, "pi")
	mustWrite(t, pi, "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > \"$PI_RECORD\"\n")
	if err := os.Chmod(pi, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("PI_RECORD", record)
	t.Setenv(callerCWDEnv, project)
	if err := runPi([]string{"native", "args"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	canonicalProject, err := infra.CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	want := canonicalProject + "\nnative\nargs\n"
	if string(body) != want {
		t.Fatalf("native passthrough=%q want=%q", body, want)
	}
	config := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(config))
	mustWrite(t, config, "[agents.pi.primary_session\nprofile = \"broken\"\n")
	if err := runPi([]string{"must-not-run"}); err == nil {
		t.Fatal("malformed Pi policy fell back to native passthrough")
	}
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	if err := runPi([]string{"must-not-run-on-read-failure"}); err == nil {
		t.Fatal("Pi policy read failure fell back to native passthrough")
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("Pi policy read failure executed native Pi: %v", err)
	}
}

func TestRunComposePiMalformedPolicyPrecedesMissingExecutable(t *testing.T) {
	project := t.TempDir()
	config := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(config))
	mustWrite(t, config, "[agents.pi.primary_session\nprofile = \"broken\"\n")
	t.Setenv("PATH", t.TempDir())
	var composeErr error
	output := captureStdout(t, func() {
		composeErr = runCompose([]string{"--mode", "primary-session", "--agent", "pi", "--project", project, "--schema-version", "1", "--json"})
	})
	if composeErr == nil {
		t.Fatal("malformed Pi policy composed")
	}
	var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
	decodeSingleJSONDocument(t, output, &envelope)
	if envelope.Error.Code != infra.PrimarySessionErrorInvalidProjectConfiguration {
		t.Fatalf("error precedence=%#v", envelope.Error)
	}
}

func mainTestPiConfig(runtime string, port int) string {
	return fmt.Sprintf(`
[agents.pi.primary_session]
profile = "profile"
pi_compatibility = %q
[agents.pi.profiles.profile]
provider = "local-provider"
model = "Model"
base_url = "http://127.0.0.1:%d/v1"
api = "openai-completions"
reasoning = false
input = ["text"]
context_window = 8192
max_tokens = 1024
thinking = "off"
requested_capabilities = ["text", "tools"]
[agents.pi.profiles.profile.compat]
supports_developer_role = false
[agents.pi.profiles.profile.runtime]
executable = %q
argv = ["serve", "--model", "Model", "--host", "127.0.0.1", "--port", "%d"]
readiness_path = "/models"
startup_timeout_seconds = 5
shutdown_timeout_seconds = 2
`, infra.PiCompatibilityV0842DarwinARM64, port, runtime, port)
}

func mainTestOfficialPiAsset(t *testing.T) string {
	t.Helper()
	root, _ := filepath.Abs("../../.temp/TASK-260817-2h8hn4/pi-standalone-darwin-arm64-0.84.2/pi")
	if _, err := os.Stat(filepath.Join(root, "pi")); err != nil {
		t.Skipf("official Pi asset unavailable: %v", err)
	}
	return root
}

func TestRunPrepareEmitsOneV1Document(t *testing.T) {
	project := preparedRuntimeFixtureMain(t)
	output := captureStdout(t, func() {
		if err := run([]string{
			"prepare",
			"--agent", "codex",
			"--project", project,
			"--schema-version", "1",
			"--json",
		}); err != nil {
			t.Fatalf("run prepare: %v", err)
		}
	})
	var report infra.PrimarySessionPreparationReport
	decodeSingleJSONDocument(t, output, &report)
	if report.Contract != infra.PrimarySessionPreparationContract ||
		report.SchemaVersion != 1 ||
		report.Status != "ok" ||
		report.Provider != "codex" ||
		!report.CodexProjectRendered ||
		report.CodexConfigGenerated ||
		len(report.Artifacts) != 3 ||
		report.Artifacts[2].Kind != "codex-config" ||
		report.Artifacts[2].State != "absent" {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunPrepareUnsupportedSchemaVersionEmitsSafeErrorEnvelope(t *testing.T) {
	project := t.TempDir()
	var prepareErr error
	output := captureStdout(t, func() {
		prepareErr = runPrepare([]string{
			"--agent", "claude",
			"--project", project,
			"--schema-version", "2",
			"--json",
		})
	})
	if prepareErr == nil {
		t.Fatal("runPrepare succeeded for unsupported schema")
	}
	var envelope infra.PrimarySessionPreparationErrorEnvelope
	decodeSingleJSONDocument(t, output, &envelope)
	if envelope.Contract != infra.PrimarySessionPreparationContract ||
		envelope.Status != "error" ||
		envelope.Error.Code != infra.PrimarySessionPreparationErrorUnsupportedSchemaVersion {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestDirectLaunchAndPrepareCommandRenderIdenticalProviderArtifacts(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	mustWrite(t, filepath.Join(binDir, "claude"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			launchProject := preparedRuntimeFixtureMain(t)
			prepareProject := preparedRuntimeFixtureMain(t)
			t.Setenv(callerCWDEnv, launchProject)
			var launchErr error
			if provider == "codex" {
				launchErr = runCodex(nil)
			} else {
				launchErr = runClaude(nil)
			}
			if launchErr != nil {
				t.Fatalf("direct launch: %v", launchErr)
			}

			output := captureStdout(t, func() {
				if err := runPrepare([]string{
					"--agent", provider,
					"--project", prepareProject,
					"--schema-version", "1",
					"--json",
				}); err != nil {
					t.Fatalf("prepare: %v", err)
				}
			})
			var report infra.PrimarySessionPreparationReport
			decodeSingleJSONDocument(t, output, &report)
			canonicalLaunchProject, err := filepath.EvalSymlinks(launchProject)
			if err != nil {
				t.Fatal(err)
			}
			canonicalPrepareProject := report.ProjectDir
			for _, artifact := range report.Artifacts {
				relative, err := filepath.Rel(canonicalPrepareProject, artifact.Path)
				if err != nil {
					t.Fatal(err)
				}
				otherPath := filepath.Join(canonicalLaunchProject, relative)
				if artifact.State == "absent" {
					if _, err := os.Lstat(otherPath); !os.IsNotExist(err) {
						t.Fatalf("%s direct launch changed absent artifact %s: %v", provider, relative, err)
					}
					if _, err := os.Lstat(artifact.Path); !os.IsNotExist(err) {
						t.Fatalf("%s prepare changed absent artifact %s: %v", provider, relative, err)
					}
					continue
				}
				if artifact.Target != "" {
					otherTarget, err := os.Readlink(otherPath)
					if err != nil {
						t.Fatal(err)
					}
					wantRelativeTarget, err := filepath.Rel(canonicalPrepareProject, artifact.Target)
					if err != nil {
						t.Fatal(err)
					}
					gotRelativeTarget, err := filepath.Rel(canonicalLaunchProject, otherTarget)
					if err != nil {
						t.Fatal(err)
					}
					if gotRelativeTarget != wantRelativeTarget {
						t.Fatalf("%s link target differs for %s: %s != %s", provider, relative, gotRelativeTarget, wantRelativeTarget)
					}
					continue
				}
				other, err := os.ReadFile(otherPath)
				if err != nil {
					t.Fatal(err)
				}
				prepared, err := os.ReadFile(artifact.Path)
				if err != nil {
					t.Fatal(err)
				}
				normalizedOther := strings.ReplaceAll(string(other), canonicalLaunchProject, "$PROJECT")
				normalizedPrepared := strings.ReplaceAll(string(prepared), canonicalPrepareProject, "$PROJECT")
				if normalizedOther != normalizedPrepared {
					t.Fatalf("%s artifact differs for %s", provider, relative)
				}
			}
			launchSurface := normalizedProviderSurface(t, canonicalLaunchProject, provider)
			preparedSurface := normalizedProviderSurface(t, canonicalPrepareProject, provider)
			if !reflect.DeepEqual(launchSurface, preparedSurface) {
				t.Fatalf(
					"%s full provider surface differs:\ndirect=%#v\nprepare=%#v",
					provider,
					launchSurface,
					preparedSurface,
				)
			}
		})
	}
}

func normalizedProviderSurface(t *testing.T, project, provider string) map[string]string {
	t.Helper()
	surfaceDir := filepath.Join(project, "."+provider)
	snapshot := map[string]string{}
	err := filepath.WalkDir(surfaceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			snapshot[relative] = "link:" + strings.ReplaceAll(target, project, "$PROJECT")
		case entry.IsDir():
			snapshot[relative] = "dir"
		default:
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			snapshot[relative] = "file:" + strings.ReplaceAll(string(data), project, "$PROJECT")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s surface: %v", provider, err)
	}
	if provider == "codex" {
		data, err := os.ReadFile(filepath.Join(project, "AGENTS.md"))
		if err != nil {
			t.Fatal(err)
		}
		snapshot["AGENTS.md"] = "file:" + strings.ReplaceAll(string(data), project, "$PROJECT")
	}
	return snapshot
}

func preparedRuntimeFixtureMain(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	for _, dir := range []string{
		filepath.Join(project, ".agents", ".configs"),
		filepath.Join(project, ".agents", ".instructions"),
		filepath.Join(project, ".agents", ".rules"),
		filepath.Join(project, ".agents", "skills", "example"),
	} {
		mustMkdir(t, dir)
	}
	mustWrite(t, filepath.Join(project, ".agents", ".configs", "codex-config.toml"), `model = "gpt-test"

[profiles.fast]
model = "gpt-fast"
`)
	mustWrite(t, filepath.Join(project, ".agents", ".configs", "claude-settings.json"), "{}\n")
	mustWrite(t, filepath.Join(project, ".agents", ".instructions", "AGENTS.md"), "# Managed instructions\n")
	mustWrite(t, filepath.Join(project, ".agents", ".rules", "default.rules"), "allow\n")
	mustWrite(t, filepath.Join(project, ".agents", "skills", "example", "SKILL.md"), "# Example\n")
	return project
}

func TestRunComposePrimarySessionUnsupportedSchemaVersionEmitsSafeErrorEnvelope(t *testing.T) {
	project := t.TempDir()
	var composeErr error
	output := captureStdout(t, func() {
		composeErr = runCompose([]string{"--mode", "primary-session", "--agent", "codex", "--project", project, "--schema-version", "2", "--json"})
	})
	if composeErr == nil {
		t.Fatal("runCompose succeeded for unsupported primary-session schema version")
	}
	var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
	decodeSingleJSONDocument(t, output, &envelope)
	if envelope.Contract != infra.PrimarySessionLaunchPlanContract || envelope.SchemaVersion != 1 || envelope.Status != "error" || envelope.Provider != "codex" || envelope.Error.Code != "unsupported_schema_version" {
		t.Fatalf("error envelope = %#v", envelope)
	}
}

func TestRunComposePrimarySessionMissingExecutableEmitsSafeErrorEnvelope(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())

	var composeErr error
	output := captureStdout(t, func() {
		composeErr = runCompose([]string{"--mode", "primary-session", "--agent", "claude", "--project", project, "--schema-version", "1", "--json"})
	})
	if composeErr == nil {
		t.Fatal("runCompose succeeded without provider executable")
	}
	var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
	decodeSingleJSONDocument(t, output, &envelope)
	if envelope.Error.Code != "provider_executable_not_found" {
		t.Fatalf("error envelope = %#v", envelope)
	}
}

func TestRunComposePrimarySessionMalformedProviderArgsFailClosed(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	mustWrite(t, filepath.Join(binDir, "claude"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	for _, testCase := range []struct {
		name     string
		provider string
		args     []string
	}{
		// The providers reject each shape at runtime (codex-cli 0.145.0
		// exit 2 at clap parse or exit 1 at config deserialization, Claude
		// Code 2.1.217 exit 1); the composed plan must fail closed instead
		// of emitting a lossy ok plan.
		{name: "codex_trailing_model", provider: "codex", args: []string{"--model"}},
		{name: "codex_repeated_profile", provider: "codex", args: []string{"--profile", "fast", "--profile", "slow"}},
		{name: "codex_empty_profile_value", provider: "codex", args: []string{"--profile="}},
		{name: "codex_empty_short_profile_value", provider: "codex", args: []string{"-p="}},
		{name: "codex_slash_profile_value", provider: "codex", args: []string{"--profile=foo/bar"}},
		{name: "codex_space_profile_value", provider: "codex", args: []string{"--profile", "a b"}},
		{name: "codex_attached_dot_profile_value", provider: "codex", args: []string{"-p."}},
		{name: "codex_repeated_remote_auth_token_env", provider: "codex", args: []string{"--remote-auth-token-env", "A_TOKEN", "--remote-auth-token-env=B_TOKEN"}},
		{name: "codex_missing_remote_auth_token_env_value", provider: "codex", args: []string{"--remote-auth-token-env"}},
		{name: "codex_invalid_sandbox_value", provider: "codex", args: []string{"--sandbox", "banana"}},
		{name: "codex_invalid_config_policy_value", provider: "codex", args: []string{"-c", `sandbox_mode="banana"`}},
		{name: "claude_trailing_model", provider: "claude", args: []string{"--model"}},
		{name: "claude_invalid_permission_mode", provider: "claude", args: []string{"--permission-mode", "banana"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			composeArgs := append([]string{"--mode", "primary-session", "--agent", testCase.provider, "--project", project, "--schema-version", "1", "--json", "--"}, testCase.args...)
			var composeErr error
			output := captureStdout(t, func() {
				composeErr = runCompose(composeArgs)
			})
			if composeErr == nil {
				t.Fatalf("runCompose succeeded for malformed provider args %v", testCase.args)
			}
			var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
			decodeSingleJSONDocument(t, output, &envelope)
			if envelope.Error.Code != infra.PrimarySessionErrorInvalidProviderArguments {
				t.Fatalf("error envelope = %#v, want code %q", envelope, infra.PrimarySessionErrorInvalidProviderArguments)
			}
			if envelope.Status != "error" || envelope.SchemaVersion != 1 {
				t.Fatalf("error envelope = %#v", envelope)
			}
		})
	}
}

func TestRunComposeRejectsUnknownMode(t *testing.T) {
	project := t.TempDir()
	if err := runCompose([]string{"--mode", "spawn", "--agent", "codex", "--project", project, "--schema-version", "1", "--json"}); err == nil {
		t.Fatal("runCompose accepted unknown mode")
	}
}

func TestRunComposePrimarySessionCodexManagedHostPreservesGlobalArgs(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	binDir := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), `
[agents.codex.primary_session]
model = "gpt-5.2-codex"
reasoning_effort = "high"
yolo_mode = true
`)
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	output := captureStdout(t, func() {
		if err := run([]string{"compose", "--mode", "primary-session", "--agent", "codex", "--project", project, "--schema-version", "1", "--json", "--", "-c", `service_tier="fast"`, "--enable", "web_search", "--strict-config"}); err != nil {
			t.Fatalf("run compose primary-session: %v", err)
		}
	})
	var plan infra.PrimarySessionLaunchPlan
	decodeSingleJSONDocument(t, output, &plan)

	wantInteractive := []string{
		"--model", "gpt-5.2-codex",
		"-c", "model_reasoning_effort=\"high\"",
		"--dangerously-bypass-approvals-and-sandbox",
		"-c", `service_tier="fast"`,
		"--enable", "web_search",
		"--strict-config",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, wantInteractive) {
		t.Fatalf("interactive argv = %#v, want %#v", plan.LaunchVariants.Interactive.Argv, wantInteractive)
	}
	wantManaged := []string{
		"-c", "model=\"gpt-5.2-codex\"",
		"-c", "model_reasoning_effort=\"high\"",
		"-c", `service_tier="fast"`,
		"--enable", "web_search",
		"--strict-config",
		"app-server",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, wantManaged) {
		t.Fatalf("managed host argv = %#v, want %#v", plan.LaunchVariants.ManagedHost.Argv, wantManaged)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, []string{}) {
		t.Fatalf("managed client argv = %#v, want empty", plan.LaunchVariants.ManagedClient.Argv)
	}
}

func TestRunComposePrimarySessionCodexRoutesAttachedFormsAndClientTokens(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	binDir := t.TempDir()
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	output := captureStdout(t, func() {
		if err := run([]string{
			"compose", "--mode", "primary-session", "--agent", "codex", "--project", project, "--schema-version", "1", "--json", "--",
			"-mgpt-5.4", "-pspeed",
			"--oss", "--local-provider", "ollama",
			"--dangerously-bypass-hook-trust",
			"-C", "/tmp", "--add-dir", "/tmp",
			"--search", "--no-alt-screen",
			"resume", "--last",
		}); err != nil {
			t.Fatalf("run compose primary-session: %v", err)
		}
	})
	var plan infra.PrimarySessionLaunchPlan
	decodeSingleJSONDocument(t, output, &plan)

	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "gpt-5.4" || plan.Resolved.Model.Source != "cli:-m" {
		t.Fatalf("resolved model = %#v", plan.Resolved.Model)
	}
	if plan.Resolved.Profile.Value == nil || *plan.Resolved.Profile.Value != "speed" || plan.Resolved.Profile.Source != "cli:-p" {
		t.Fatalf("resolved profile = %#v", plan.Resolved.Profile)
	}
	wantHost := []string{
		"-c", "model=\"gpt-5.4\"",
		"--oss", "--local-provider", "ollama",
		"--dangerously-bypass-hook-trust",
		"--search",
		"app-server",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, wantHost) {
		t.Fatalf("managed host argv = %#v, want %#v", plan.LaunchVariants.ManagedHost.Argv, wantHost)
	}
	wantClient := []string{
		"-pspeed",
		"-C", "/tmp", "--add-dir", "/tmp",
		"--no-alt-screen",
		"resume", "--last",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, wantClient) {
		t.Fatalf("managed client argv = %#v, want %#v", plan.LaunchVariants.ManagedClient.Argv, wantClient)
	}
}

func TestRunComposePrimarySessionResolvesNativePolicyFlags(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	binDir := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), `
[agents.codex.primary_session]
model = "gpt-5.2-codex"

[agents.claude.primary_session]
model = "claude-opus-4-8"
`)
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	mustWrite(t, filepath.Join(binDir, "claude"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	t.Run("codex", func(t *testing.T) {
		output := captureStdout(t, func() {
			if err := run([]string{"compose", "--mode", "primary-session", "--agent", "codex", "--project", project, "--schema-version", "1", "--json", "--", "--sandbox", "read-only", "--ask-for-approval", "on-request"}); err != nil {
				t.Fatalf("run compose primary-session: %v", err)
			}
		})
		var plan infra.PrimarySessionLaunchPlan
		decodeSingleJSONDocument(t, output, &plan)
		if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != "read-only" || plan.Resolved.Sandbox.Source != "cli:--sandbox" {
			t.Fatalf("resolved sandbox = %#v", plan.Resolved.Sandbox)
		}
		if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "on-request" || plan.Resolved.Approval.Source != "cli:--ask-for-approval" {
			t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
		}
	})
	t.Run("claude", func(t *testing.T) {
		output := captureStdout(t, func() {
			if err := run([]string{"compose", "--mode", "primary-session", "--agent", "claude", "--project", project, "--schema-version", "1", "--json", "--", "--effort", "xhigh", "--permission-mode", "plan"}); err != nil {
				t.Fatalf("run compose primary-session: %v", err)
			}
		})
		var plan infra.PrimarySessionLaunchPlan
		decodeSingleJSONDocument(t, output, &plan)
		if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != "xhigh" || plan.Resolved.Reasoning.Source != "cli:--effort" {
			t.Fatalf("resolved reasoning = %#v", plan.Resolved.Reasoning)
		}
		if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "plan" || plan.Resolved.Approval.Source != "cli:--permission-mode" {
			t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
		}
	})
}

func TestRunComposePrimarySessionCodexSurfacesRemoteAuthEnvName(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	binDir := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, "project-config.toml"), `
[mcp]
enabled_servers = ["jira"]
`)
	mustWrite(t, filepath.Join(configDir, "codex-mcp-servers.toml"), `
[servers.jira]
url = "https://jira.example/mcp"
bearer_token_env_var = "JIRA_TOKEN"
`)
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	t.Run("distinct_name_appends_after_mcp_names", func(t *testing.T) {
		output := captureStdout(t, func() {
			if err := run([]string{"compose", "--mode", "primary-session", "--agent", "codex", "--project", project, "--schema-version", "1", "--json", "--", "--remote", "ws://127.0.0.1:4500", "--remote-auth-token-env", "CODEX_REMOTE_TOKEN"}); err != nil {
				t.Fatalf("run compose primary-session: %v", err)
			}
		})
		var plan infra.PrimarySessionLaunchPlan
		decodeSingleJSONDocument(t, output, &plan)
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"JIRA_TOKEN", "CODEX_REMOTE_TOKEN"}) {
			t.Fatalf("required_env_names = %#v, want [JIRA_TOKEN CODEX_REMOTE_TOKEN]", plan.RequiredEnvNames)
		}
		wantClient := []string{"--remote", "ws://127.0.0.1:4500", "--remote-auth-token-env", "CODEX_REMOTE_TOKEN"}
		if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, wantClient) {
			t.Fatalf("managed client argv = %#v, want %#v", plan.LaunchVariants.ManagedClient.Argv, wantClient)
		}
	})

	t.Run("mcp_bearer_name_is_deduplicated", func(t *testing.T) {
		output := captureStdout(t, func() {
			if err := run([]string{"compose", "--mode", "primary-session", "--agent", "codex", "--project", project, "--schema-version", "1", "--json", "--", "--remote-auth-token-env=JIRA_TOKEN"}); err != nil {
				t.Fatalf("run compose primary-session: %v", err)
			}
		})
		var plan infra.PrimarySessionLaunchPlan
		decodeSingleJSONDocument(t, output, &plan)
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"JIRA_TOKEN"}) {
			t.Fatalf("required_env_names = %#v, want deduplicated [JIRA_TOKEN]", plan.RequiredEnvNames)
		}
	})
}
