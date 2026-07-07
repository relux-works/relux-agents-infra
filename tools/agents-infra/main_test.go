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
	mustWrite(t, filepath.Join(appDir, ".agents", ".configs", "project-config.toml"), "[codex.mcp]\nenabled_servers = [\"figma\"]\n")
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
	mustWrite(t, filepath.Join(appDir, ".agents", ".configs", "project-config.toml"), "[codex.mcp]\nenabled_servers = [\"safari\"]\n")
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
