//go:build !windows

package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
)

func mainStandaloneQwenProject(t *testing.T) (project, home string) {
	t.Helper()
	project, home = t.TempDir(), t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	body := mainTestPiConfig("/bin/echo", 18011)
	body = strings.Replace(body, `reasoning = false`, `reasoning = true`, 1)
	body = strings.Replace(body, `thinking = "off"`, `thinking = "medium"`, 1)
	body = strings.Replace(body, `supports_developer_role = false`, "supports_developer_role = false\nsupports_reasoning_effort = false\nthinking_format = \"qwen-chat-template\"", 1)
	body += `
[agents.pi.standalone_session]
yolo_mode = true
tool_allowlist = ["read", "bash", "edit", "write"]

[agents.targets.qwen]
vendor = "qwen"
environment = "pi"
model = "Model"
reasoning = "medium"
profile = "profile"
profile_provider = "local-provider"
endpoint = "http://127.0.0.1:18011/v1"

[agents.entrypoints]
qwen-infra = "qwen"
`
	writeMainCanonicalConfig(t, project, body)
	return project, home
}

func TestSchemaOneWriterPrecedesProfileAndRequestRefusals(t *testing.T) {
	tests := []struct {
		name string
		args []string
		code managementpi.TurnResultCode
	}{
		{"missing profile", []string{"spawn", "--prompt", "secret", "--deadline", "30m", "--result-schema", "1"}, managementpi.TurnCodeProfileMissing},
		{"duplicate profile", []string{"spawn", "--profile", "a", "--profile", "b", "--prompt", "secret", "--deadline", "30m", "--result-schema", "1"}, managementpi.TurnCodeRequestInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var runErr error
			output := captureStdout(t, func() { runErr = runPi(test.args) })
			var failure *infra.PiTurnProcessAError
			if !errors.As(runErr, &failure) {
				t.Fatalf("runPi error = %#v", runErr)
			}
			var document struct {
				Contract      string `json:"contract"`
				SchemaVersion int    `json:"schema_version"`
				Status        string `json:"status"`
				Error         struct {
					Code managementpi.TurnResultCode `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(output), &document); err != nil {
				t.Fatalf("schema-1 output = %q: %v", output, err)
			}
			if document.Contract != managementpi.TurnResultContract || document.SchemaVersion != 1 || document.Status != "error" || document.Error.Code != test.code {
				t.Fatalf("schema-1 refusal = %#v", document)
			}
			if strings.Contains(output, "secret") || strings.Count(strings.TrimSpace(output), "\n") != 0 {
				t.Fatalf("schema-1 refusal leaked or emitted multiple documents: %q", output)
			}
		})
	}
}

// Production call site: runTarget -> runPiStandaloneCLI ->
// BuildPiStandaloneLaunchPlan. This proves the public qwen-infra command emits
// the managed unattended contract without starting Pi or revealing the prompt.
func TestRunTargetQwenStandalonePrintConfigOwnsAuthorizationAndPreservesReasoning(t *testing.T) {
	project, home := mainStandaloneQwenProject(t)
	t.Setenv(callerCWDEnv, project)
	t.Setenv("HOME", home)
	t.Setenv("PATH", mainTestOfficialPiAsset(t))
	secretPrompt := "task-scoped secret prompt"
	output := captureStdout(t, func() {
		if err := runTarget([]string{"qwen-infra", "spawn", "--prompt", secretPrompt, "--print-config"}); err != nil {
			t.Fatalf("runTarget standalone print-config: %v", err)
		}
	})
	var plan infra.PiStandaloneLaunchPlan
	decodeSingleJSONDocument(t, output, &plan)
	if plan.Contract != infra.PiStandaloneLaunchPlanContract || plan.Status != "ok" || plan.Reasoning.Value == nil || *plan.Reasoning.Value != "medium" {
		t.Fatalf("standalone qwen launch plan = %#v", plan)
	}
	wantTail := []string{"--no-approve", "--no-extensions", "--tools", "read,bash,edit,write", "--mode", "json", "--no-session", "--print", "<prompt>"}
	if len(plan.Argv) < len(wantTail) || !reflect.DeepEqual(plan.Argv[len(plan.Argv)-len(wantTail):], wantTail) {
		t.Fatalf("standalone managed argv = %#v, want tail %#v", plan.Argv, wantTail)
	}
	if plan.ToolAuthorization.ProjectTrust != "declined" || plan.ToolAuthorization.RPCDirectBash != "not_exposed" || plan.ToolAuthorization.TaskBoardAdapter != "deferred_not_implemented" {
		t.Fatalf("standalone authorization diagnostics = %#v", plan.ToolAuthorization)
	}
	if strings.Contains(output, secretPrompt) {
		t.Fatalf("standalone print-config leaked prompt: %s", output)
	}
}

// Production call site: runPi/runTarget -> runPiStandaloneCLI. Unknown Pi
// authorization, extension, mode, and positional shapes never reach RunPi.
func TestStandaloneCLIRefusesCallerOverridesWithSanitizedTypedFailure(t *testing.T) {
	secret := "caller secret prompt"
	for _, testCase := range []struct {
		name string
		run  func() error
	}{
		{name: "qwen tools override", run: func() error { return runTarget([]string{"qwen-infra", "spawn", "--prompt", secret, "--tools", "bash"}) }},
		{name: "qwen extension override", run: func() error {
			return runTarget([]string{"qwen-infra", "spawn", "--prompt", secret, "--extension", "/tmp/replacement.ts"})
		}},
		{name: "qwen rpc override", run: func() error { return runTarget([]string{"qwen-infra", "spawn", "--prompt", secret, "--mode", "rpc"}) }},
		{name: "direct pi positional override", run: func() error { return runPi([]string{"spawn", "--prompt", secret, "--", "--approve"}) }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.run()
			var failure *infra.PiStandaloneFailure
			if !errors.As(err, &failure) || failure.Code != "pi_standalone_cli_invalid" {
				t.Fatalf("standalone CLI override error = %#v", err)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "replacement.ts") {
				t.Fatalf("typed standalone CLI failure leaked caller input: %s", err)
			}
		})
	}
}

func TestStandaloneCLIAcceptsExactDeadlineBounds(t *testing.T) {
	project, home := mainStandaloneQwenProject(t)
	t.Setenv(callerCWDEnv, project)
	t.Setenv("HOME", home)
	t.Setenv("PATH", mainTestOfficialPiAsset(t))
	for _, deadline := range []string{"1ns", "30m"} {
		t.Run(deadline, func(t *testing.T) {
			if err := runTarget([]string{"qwen-infra", "spawn", "--prompt", "safe prompt", "--deadline", deadline, "--print-config"}); err != nil {
				t.Fatalf("exact standalone deadline bound %s refused: %v", deadline, err)
			}
		})
	}
}

// Production call site: runTarget -> runPiStandaloneCLI. Out-of-range values
// are refused before plan resolution or launch, and the public error contains
// only the typed code rather than the prompt or caller-supplied duration.
func TestStandaloneCLIRefusesOutOfRangeDeadlineWithSanitizedFailure(t *testing.T) {
	secretPrompt := "deadline prompt secret"
	for _, deadline := range []string{"-1ns", "0", "30m1ns"} {
		t.Run(deadline, func(t *testing.T) {
			err := runTarget([]string{"qwen-infra", "spawn", "--prompt", secretPrompt, "--deadline", deadline, "--print-config"})
			var failure *infra.PiStandaloneFailure
			if !errors.As(err, &failure) || failure.Code != "pi_standalone_deadline_invalid" {
				t.Fatalf("standalone deadline error = %#v", err)
			}
			if got := err.Error(); strings.Contains(got, secretPrompt) || strings.Contains(got, deadline) || got != `{"contract":"agents-infra.pi-standalone-result","schema_version":1,"status":"error","error":{"code":"pi_standalone_deadline_invalid"}}` {
				t.Fatalf("standalone deadline failure was not sanitized: %s", got)
			}
		})
	}
}
