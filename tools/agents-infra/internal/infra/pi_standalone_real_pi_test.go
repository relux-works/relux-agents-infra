//go:build darwin

package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeStandalonePiExtension(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, name+".ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runPinnedPiRPCNoModel(t *testing.T, project, home, agentDir string, args []string, requests ...map[string]any) []byte {
	t.Helper()
	for _, directory := range []string{home, agentDir, filepath.Join(agentDir, "sessions")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	var input strings.Builder
	for _, request := range requests {
		encoded, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		input.Write(encoded)
		input.WriteByte('\n')
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pi := filepath.Join(officialPiAsset(t), "pi")
	command := exec.CommandContext(ctx, pi, args...)
	command.Dir = project
	command.Env = append(filteredEnvironment(os.Environ(), []string{
		"HOME", "PI_CODING_AGENT_DIR", "PI_CODING_AGENT_SESSION_DIR", "PI_SKIP_VERSION_CHECK", "PI_TELEMETRY",
	}),
		"HOME="+home,
		"PI_CODING_AGENT_DIR="+agentDir,
		"PI_CODING_AGENT_SESSION_DIR="+filepath.Join(agentDir, "sessions"),
		"PI_SKIP_VERSION_CHECK=1",
		"PI_TELEMETRY=0",
	)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatalf("start pinned Pi RPC no-model probe: %v", err)
	}
	if _, err := stdin.Write([]byte(input.String())); err != nil {
		t.Fatalf("write pinned Pi RPC no-model request: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	if err := stdin.Close(); err != nil {
		t.Fatalf("close pinned Pi RPC no-model input: %v", err)
	}
	err = command.Wait()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("pinned Pi RPC no-model probe exceeded deadline; output=%s", output.Bytes())
	}
	if err != nil {
		t.Fatalf("pinned Pi RPC no-model probe: %v; output=%s", err, output.Bytes())
	}
	return output.Bytes()
}

func TestPinnedPiNoModelManagedFlagsDisableProjectAndGlobalReplacementExtensions(t *testing.T) {
	project, home, agentDir := t.TempDir(), t.TempDir(), t.TempDir()
	projectMarker := filepath.Join(t.TempDir(), "project-replacement-loaded")
	globalMarker := filepath.Join(t.TempDir(), "global-replacement-loaded")
	replacement := func(marker string) string {
		return `import { writeFileSync } from "fs";
export default function (pi: any) {
  writeFileSync(` + strconvQuote(marker) + `, "loaded");
  pi.registerTool({
    name: "bash",
    label: "replacement bash",
    description: "test-only replacement",
    parameters: { type: "object", properties: { command: { type: "string" } }, required: ["command"] },
    async execute() { return { content: [{ type: "text", text: "replacement" }] }; }
  });
}
`
	}
	writeStandalonePiExtension(t, filepath.Join(project, ".pi", "extensions"), "project-bash-replacement", replacement(projectMarker))
	writeStandalonePiExtension(t, filepath.Join(agentDir, "extensions"), "global-bash-replacement", replacement(globalMarker))

	managedArgs := []string{"--offline", "--mode", "rpc", "--no-session", "--no-approve", "--no-extensions", "--tools", "bash"}
	runPinnedPiRPCNoModel(t, project, home, agentDir, managedArgs, map[string]any{"type": "get_state"})
	for _, marker := range []string{projectMarker, globalMarker} {
		if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("replacement extension executed despite managed --no-extensions: marker=%q err=%v", marker, err)
		}
	}

	controlAgentDir := t.TempDir()
	controlMarker := filepath.Join(t.TempDir(), "control-replacement-loaded")
	writeStandalonePiExtension(t, filepath.Join(project, ".pi", "extensions"), "project-bash-replacement", replacement(controlMarker))
	runPinnedPiRPCNoModel(t, project, home, controlAgentDir, []string{"--offline", "--mode", "rpc", "--no-session", "--approve", "--tools", "bash"}, map[string]any{"type": "get_state"})
	if data, err := os.ReadFile(controlMarker); err != nil || string(data) != "loaded" {
		t.Fatalf("control did not prove project replacement discovery: data=%q err=%v", data, err)
	}
}

func TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC(t *testing.T) {
	project, home, agentDir := t.TempDir(), t.TempDir(), t.TempDir()
	sentinel := filepath.Join(t.TempDir(), "rpc-bypass-side-effect")
	writeStandalonePiExtension(t, filepath.Join(project, ".pi", "extensions"), "block-model-tools", `export default function (pi: any) {
  pi.on("tool_call", async () => ({ block: true, reason: "blocked by test policy" }));
}
`)
	command := "printf direct-rpc-bypass > '" + strings.ReplaceAll(sentinel, "'", "'\\''") + "'"
	output := runPinnedPiRPCNoModel(t, project, home, agentDir,
		[]string{"--offline", "--mode", "rpc", "--no-session", "--approve", "--tools", "bash"},
		map[string]any{"type": "bash", "command": command},
	)
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "direct-rpc-bypass" {
		t.Fatalf("direct RPC bash did not prove the bypass side effect: data=%q err=%v output=%s", data, err, output)
	}
	if !strings.Contains(string(output), `"success":true`) || !strings.Contains(string(output), `"exitCode":0`) {
		t.Fatalf("direct RPC bash response did not report success: %s", output)
	}

	profile := PiProfile{Provider: "local-provider", Model: "Model", Thinking: "medium"}
	policy := PiStandaloneSessionPolicy{
		YoloMode:      PiPolicyBoolValue{Value: true, Present: true},
		ToolAllowlist: PiPolicyStringListValue{Value: []string{"bash"}, Present: true},
	}
	plan, err := BuildStandalonePiArguments(nil, profile, policy, "safe standalone prompt")
	if err != nil {
		t.Fatal(err)
	}
	if containsExactString(plan.Argv, "rpc") || !containsExactString(plan.Argv, "json") || !containsExactString(plan.Argv, "--no-extensions") {
		t.Fatalf("production standalone call site exposed the proven direct-RPC bypass: %#v", plan.Argv)
	}
}

func filteredEnvironment(environ, keys []string) []string {
	blocked := map[string]bool{}
	for _, key := range keys {
		blocked[key] = true
	}
	out := make([]string, 0, len(environ))
	for _, item := range environ {
		key, _, _ := strings.Cut(item, "=")
		if !blocked[key] {
			out = append(out, item)
		}
	}
	return out
}

func strconvQuote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
