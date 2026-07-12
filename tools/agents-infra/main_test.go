package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
`)
	mustMkdir(t, filepath.Join(child, ".agents", ".configs"))
	childConfig := filepath.Join(child, ".agents", ".configs", "project-config.toml")
	mustWrite(t, childConfig, `
[mcp]
enabled_servers = ["lldb", "figma"]

[agents.codex.primary_session]
reasoning_effort = "xhigh"
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
	}
	for key, wantValue := range want {
		if got := fields[key]; got != wantValue {
			t.Fatalf("%s = %q, want %q:\n%s", key, got, wantValue, output)
		}
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
