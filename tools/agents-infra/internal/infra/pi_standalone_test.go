//go:build darwin

package infra

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func standalonePiPolicyTOML(yolo string, tools string) string {
	return "\n[agents.pi.standalone_session]\n" +
		"yolo_mode = " + yolo + "\n" +
		"tool_allowlist = " + tools + "\n"
}

func validStandaloneQwenConfig(runtime string, port int, shared bool) string {
	body := reasoningPiProfileTOML("profile", runtime, port) +
		standalonePiPolicyTOML("true", `["read", "bash", "edit", "write"]`) +
		canonicalQwenTargetTOML(true, true)
	if shared {
		body += `
[agents.pi.profiles.profile.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 8
max_segment_bytes = 1048576
max_segments = 7
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
resource_pressure_mode = "disabled"
`
	}
	return body
}

func TestBuildStandalonePiArgumentsOwnsExactAuthorizationAndMediumReasoning(t *testing.T) {
	config, err := parseProjectConfig([]byte(validStandaloneQwenConfig("/bin/echo", 18011, false)), "/project/.agents/.configs/project-config.toml")
	if err != nil {
		t.Fatal(err)
	}
	policy := PiStandaloneSessionPolicy{}
	composePiStandaloneSession(&policy, config.PiStandaloneSession, "/project/.agents/.configs/project-config.toml")
	plan, err := BuildStandalonePiArguments(nil, config.PiProfiles["profile"], policy, "write the requested sentinel")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--provider", "local-provider",
		"--model", "Model",
		"--thinking", "medium",
		"--no-approve",
		"--no-extensions",
		"--tools", "read,bash,edit,write",
		"--mode", "json",
		"--no-session",
		"--print",
		"--",
		"write the requested sentinel",
	}
	if !reflect.DeepEqual(plan.Argv, want) {
		t.Fatalf("standalone argv = %#v, want %#v", plan.Argv, want)
	}
	if plan.DiagnosticArgv[len(plan.DiagnosticArgv)-1] != "<prompt>" || strings.Contains(strings.Join(plan.DiagnosticArgv, "\x00"), "requested sentinel") {
		t.Fatalf("diagnostic argv exposed prompt: %#v", plan.DiagnosticArgv)
	}
	for _, forbidden := range []string{"--approve", "--extension", "-e", "rpc"} {
		if containsExactString(plan.Argv, forbidden) {
			t.Fatalf("standalone argv exposed forbidden control %q: %#v", forbidden, plan.Argv)
		}
	}
}

func TestStandalonePiAuthorizationRejectsNarrowedAndInvalidAllowlists(t *testing.T) {
	baseConfig, err := parseProjectConfig([]byte(reasoningPiProfileTOML("profile", "/bin/echo", 18011)), "/project/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	profile := baseConfig.PiProfiles["profile"]
	tests := []struct {
		name   string
		policy PiStandaloneSessionPolicy
		code   string
	}{
		{name: "absent yolo", policy: PiStandaloneSessionPolicy{}, code: "pi_tool_authorization_required"},
		{name: "false yolo", policy: PiStandaloneSessionPolicy{YoloMode: PiPolicyBoolValue{Present: true}}, code: "pi_tool_authorization_required"},
		{name: "absent allowlist", policy: PiStandaloneSessionPolicy{YoloMode: PiPolicyBoolValue{Present: true, Value: true}}, code: "pi_tool_allowlist_required"},
		{name: "empty allowlist", policy: PiStandaloneSessionPolicy{YoloMode: PiPolicyBoolValue{Present: true, Value: true}, ToolAllowlist: PiPolicyStringListValue{Present: true}}, code: "pi_tool_allowlist_invalid"},
		{name: "duplicate allowlist", policy: PiStandaloneSessionPolicy{YoloMode: PiPolicyBoolValue{Present: true, Value: true}, ToolAllowlist: PiPolicyStringListValue{Present: true, Value: []string{"read", "read"}}}, code: "pi_tool_allowlist_invalid"},
		{name: "future builtin narrowing", policy: PiStandaloneSessionPolicy{YoloMode: PiPolicyBoolValue{Present: true, Value: true}, ToolAllowlist: PiPolicyStringListValue{Present: true, Value: []string{"read", "powershell"}}}, code: "pi_tool_allowlist_invalid"},
		{name: "wildcard", policy: PiStandaloneSessionPolicy{YoloMode: PiPolicyBoolValue{Present: true, Value: true}, ToolAllowlist: PiPolicyStringListValue{Present: true, Value: []string{"*"}}}, code: "pi_tool_allowlist_invalid"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := BuildStandalonePiArguments(nil, profile, testCase.policy, "safe prompt")
			if piErrorCode(err) != testCase.code {
				t.Fatalf("error code = %q, want %q; err=%v", piErrorCode(err), testCase.code, err)
			}
		})
	}
}

func TestStandalonePiNearestFalseMasksInheritedAuthorization(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	child := filepath.Join(root, "nested")
	if err := os.MkdirAll(child, 0o700); err != nil {
		t.Fatal(err)
	}
	writePiProjectConfig(t, root, validStandaloneQwenConfig("/bin/echo", 18011, false))
	writePiProjectConfig(t, child, "[agents.pi.standalone_session]\nyolo_mode = false\n")
	lookedUp := false
	err := RunPi(RunPiOptions{
		ProjectDir: child,
		HomeDir:    home,
		Environ:    []string{"HOME=" + home},
		LookPath: func(string) (string, error) {
			lookedUp = true
			return "", errors.New("must not be reached")
		},
		Standalone: &PiStandaloneRequest{Prompt: "safe prompt", Entrypoint: "qwen-infra"},
	})
	if piErrorCode(err) != "pi_tool_authorization_required" || lookedUp {
		t.Fatalf("nearest false did not mask inherited yolo authorization: code=%q looked_up=%t err=%v", piErrorCode(err), lookedUp, err)
	}
}

func TestRunPiStandaloneMalformedPolicyFailsClosedBeforeExecutableLookup(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		policy string
	}{
		{name: "wrong allowlist type", policy: "[agents.pi.standalone_session]\nyolo_mode = true\ntool_allowlist = \"read\"\n"},
		{name: "unknown field", policy: "[agents.pi.standalone_session]\nyolo_mode = true\ntool_allowlist = [\"read\"]\nextensions = true\n"},
		{name: "empty table", policy: "[agents.pi.standalone_session]\n"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			project, home := t.TempDir(), t.TempDir()
			body := reasoningPiProfileTOML("profile", "/bin/echo", 18011) + testCase.policy + canonicalQwenTargetTOML(true, true)
			writePiProjectConfig(t, project, body)
			lookedUp := false
			err := RunPi(RunPiOptions{
				ProjectDir: project,
				HomeDir:    home,
				Environ:    []string{"HOME=" + home},
				LookPath: func(string) (string, error) {
					lookedUp = true
					return "", errors.New("must not be reached")
				},
				Standalone: &PiStandaloneRequest{Prompt: "safe prompt", Entrypoint: "qwen-infra"},
			})
			if piErrorCode(err) != "invalid_project_configuration" || lookedUp {
				t.Fatalf("malformed standalone policy did not fail closed: code=%q looked_up=%t err=%v", piErrorCode(err), lookedUp, err)
			}
		})
	}
}

func TestStandalonePiNamedPromptAllowsLeadingFlagAndFileMarkers(t *testing.T) {
	profile := PiProfile{Provider: "local-provider", Model: "Model", Thinking: "medium"}
	policy := PiStandaloneSessionPolicy{
		YoloMode:      PiPolicyBoolValue{Value: true, Present: true},
		ToolAllowlist: PiPolicyStringListValue{Value: []string{"read"}, Present: true},
	}
	for _, prompt := range []string{"--approve", "@/tmp/injected-prompt"} {
		plan, err := BuildStandalonePiArguments(nil, profile, policy, prompt)
		if err != nil || plan.Argv[len(plan.Argv)-1] != prompt || plan.Argv[len(plan.Argv)-2] != "--" {
			t.Fatalf("named prompt %q was not preserved behind --: plan=%#v err=%v", prompt, plan, err)
		}
	}
	for _, prompt := range []string{"safe\x00suffix", "   "} {
		if _, err := BuildStandalonePiArguments(nil, profile, policy, prompt); piErrorCode(err) != "pi_standalone_prompt_invalid" {
			t.Fatalf("prompt %q was admitted: %v", prompt, err)
		}
	}
}

func TestRunPiStandaloneRefusesCallerAuthorizationAndRPCFlagsBeforeExecutableLookup(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	writePiProjectConfig(t, project, validStandaloneQwenConfig("/bin/echo", 18011, false))
	for _, args := range [][]string{
		{"--tools", "read"}, {"-t", "read"}, {"--exclude-tools", "bash"}, {"-xt", "bash"},
		{"--no-tools"}, {"-nt"}, {"--no-builtin-tools"}, {"-nbt"},
		{"--extension", "/tmp/replacement.ts"}, {"-e", "/tmp/replacement.ts"}, {"--no-extensions"}, {"-ne"},
		{"--approve"}, {"-a"}, {"--no-approve"}, {"-na"}, {"--mode", "rpc"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			lookedUp := false
			err := RunPi(RunPiOptions{
				ProjectDir: project,
				HomeDir:    home,
				Args:       args,
				Environ:    []string{"HOME=" + home},
				LookPath: func(string) (string, error) {
					lookedUp = true
					return "", errors.New("must not be reached")
				},
				Standalone: &PiStandaloneRequest{Prompt: "safe prompt", Entrypoint: "qwen-infra"},
			})
			if piErrorCode(err) != "pi_standalone_conflicting_arguments" || lookedUp {
				t.Fatalf("caller flags reached lookup: code=%q looked_up=%t err=%v", piErrorCode(err), lookedUp, err)
			}
		})
	}
}

func TestRunPiStandaloneAuthorizationFailurePrecedesExecutableAndState(t *testing.T) {
	piRoot := officialPiAsset(t)
	for _, testCase := range []struct {
		name   string
		policy string
		code   string
	}{
		{name: "missing", code: "pi_tool_authorization_required"},
		{name: "false", policy: standalonePiPolicyTOML("false", `["read"]`), code: "pi_tool_authorization_required"},
		{name: "empty", policy: standalonePiPolicyTOML("true", `[]`), code: "pi_tool_allowlist_invalid"},
		{name: "unknown", policy: standalonePiPolicyTOML("true", `["powershell"]`), code: "pi_tool_allowlist_invalid"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
			body := reasoningPiProfileTOML("profile", "/bin/echo", 18011) + testCase.policy + canonicalQwenTargetTOML(true, true)
			writePiProjectConfig(t, project, body)
			lookedUp := false
			err := RunPi(RunPiOptions{
				ProjectDir: project,
				HomeDir:    home,
				CacheRoot:  cache,
				Environ:    []string{"HOME=" + home},
				LookPath: func(string) (string, error) {
					lookedUp = true
					return filepath.Join(piRoot, "pi"), nil
				},
				Standalone: &PiStandaloneRequest{Prompt: "safe prompt", Entrypoint: "qwen-infra"},
			})
			if piErrorCode(err) != testCase.code || lookedUp {
				t.Fatalf("authorization failure ordering: code=%q looked_up=%t err=%v", piErrorCode(err), lookedUp, err)
			}
			entries, readErr := os.ReadDir(cache)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("authorization refusal mutated state: entries=%v err=%v", entries, readErr)
			}
		})
	}
}

func TestBuildPiStandaloneLaunchPlanSeparatesTrustAuthorizationAndRuntimeLease(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	piRoot := officialPiAsset(t)
	configPath := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	writePiProjectConfig(t, project, validStandaloneQwenConfig("/bin/echo", 18011, true))
	plan, err := BuildPiStandaloneLaunchPlan(project, home, PiStandaloneRequest{Prompt: "safe prompt", Entrypoint: "qwen-infra"}, ChildLaunchCompositionProducer{Version: "test", Commit: "test"}, func(string) (string, error) {
		return filepath.Join(piRoot, "pi"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Contract != PiStandaloneLaunchPlanContract || plan.SchemaVersion != 1 || plan.Status != "ok" || plan.Entrypoint != "qwen-infra" {
		t.Fatalf("standalone plan identity = %#v", plan)
	}
	if plan.Reasoning.Value == nil || *plan.Reasoning.Value != "medium" || !samePath(plan.Reasoning.Source, configPath) {
		t.Fatalf("standalone reasoning = %#v", plan.Reasoning)
	}
	authorization := plan.ToolAuthorization
	if !authorization.Effective || authorization.ProjectTrust != "declined" || authorization.ExtensionDiscovery != "disabled" || authorization.RPCDirectBash != "not_exposed" || authorization.HumanApprovalOrStdin != "not_required" || authorization.TaskBoardAdapter != "deferred_not_implemented" {
		t.Fatalf("standalone authorization diagnostics = %#v", authorization)
	}
	if plan.RuntimeMode != "shared" || plan.Runtime.Ownership != "shared-runtime-lease-broker" || plan.State.Isolation != "per-process-random-run-key" || plan.State.Persistence != "disabled" {
		t.Fatalf("standalone runtime/state diagnostics = %#v / %#v / %#v", plan.RuntimeMode, plan.Runtime, plan.State)
	}
	joined := strings.Join(plan.Argv, "\x00")
	if strings.Contains(joined, "safe prompt") || !strings.Contains(joined, "<prompt>") {
		t.Fatalf("standalone plan prompt redaction failed: %#v", plan.Argv)
	}
}

func TestPiStandaloneFailureIsTypedAndSanitized(t *testing.T) {
	secret := "secret-prompt-never-render"
	wrapped := WrapPiStandaloneFailure(piError("pi_tool_authorization_required", errors.New(secret)))
	var failure *PiStandaloneFailure
	if !errors.As(wrapped, &failure) || failure.Code != "pi_tool_authorization_required" {
		t.Fatalf("typed failure = %#v", wrapped)
	}
	if strings.Contains(wrapped.Error(), secret) || !strings.Contains(wrapped.Error(), `"code":"pi_tool_authorization_required"`) {
		t.Fatalf("standalone failure was not sanitized and typed: %s", wrapped)
	}
}

func containsExactString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
