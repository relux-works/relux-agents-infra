package infra

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fakePrimarySessionLookPath(t *testing.T) func(string) (string, error) {
	t.Helper()
	return func(name string) (string, error) {
		return "/fake/bin/" + name, nil
	}
}

func writePrimarySessionFixture(t *testing.T, project string) (configPath, registryPath string) {
	t.Helper()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	configPath = filepath.Join(configDir, projectConfigFileName)
	registryPath = filepath.Join(configDir, "codex-mcp-servers.toml")
	mustWrite(t, configPath, `
[mcp]
enabled_servers = ["jira", "lldb"]

[agents.codex.primary_session]
model = "gpt-5.2-codex"
reasoning_effort = "high"
yolo_mode = true

[agents.claude.primary_session]
model = "claude-opus-4-8"
yolo_mode = true
`)
	mustWrite(t, registryPath, `
[servers.jira]
url = "https://jira.example/mcp"
bearer_token_env_var = "JIRA_TOKEN"

[servers.lldb]
command = "lldb-mcp"
args = ["--socket", "auto"]
`)
	return configPath, registryPath
}

func canonicalPrimarySessionFixturePaths(t *testing.T, plan PrimarySessionLaunchPlan) (configPath, registryPath string) {
	t.Helper()
	configDir := filepath.Join(plan.ProjectDir, ".agents", ".configs")
	return filepath.Join(configDir, projectConfigFileName), filepath.Join(configDir, "codex-mcp-servers.toml")
}

func TestBuildPrimarySessionLaunchPlanCodexParityAndManagedHost(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)
	t.Setenv("JIRA_TOKEN", "must-not-be-read-or-serialized")
	producer := ChildLaunchCompositionProducer{Version: "v2.0.0", Commit: "abc123"}
	userArgs := []string{"resume", "--last"}

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, userArgs, producer, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Contract != PrimarySessionLaunchPlanContract || plan.SchemaVersion != 1 || plan.Status != "ok" || plan.Provider != "codex" {
		t.Fatalf("plan envelope = %#v", plan)
	}
	if plan.Executable != "/fake/bin/codex" {
		t.Fatalf("Executable = %q", plan.Executable)
	}

	launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}

	configPath, _ := canonicalPrimarySessionFixturePaths(t, plan)
	wantInteractive := []string{
		"-c", "mcp_servers.jira.url=\"https://jira.example/mcp\"",
		"-c", "mcp_servers.jira.bearer_token_env_var=\"JIRA_TOKEN\"",
		"-c", "mcp_servers.lldb.command=\"lldb-mcp\"",
		"-c", "mcp_servers.lldb.args=[\"--socket\", \"auto\"]",
		"--model", "gpt-5.2-codex",
		"-c", "model_reasoning_effort=\"high\"",
		codexDangerouslyBypassApprovalsAndSandbox,
		"resume", "--last",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, wantInteractive) {
		t.Fatalf("interactive argv = %#v, want %#v", plan.LaunchVariants.Interactive.Argv, wantInteractive)
	}

	wantManaged := []string{
		"-c", "mcp_servers.jira.url=\"https://jira.example/mcp\"",
		"-c", "mcp_servers.jira.bearer_token_env_var=\"JIRA_TOKEN\"",
		"-c", "mcp_servers.lldb.command=\"lldb-mcp\"",
		"-c", "mcp_servers.lldb.args=[\"--socket\", \"auto\"]",
		"-c", "model=\"gpt-5.2-codex\"",
		"-c", "model_reasoning_effort=\"high\"",
		"app-server",
	}
	if plan.LaunchVariants.ManagedHost.Kind != PrimarySessionManagedHostKindCodexAppServer {
		t.Fatalf("managed host kind = %q", plan.LaunchVariants.ManagedHost.Kind)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, wantManaged) {
		t.Fatalf("managed host argv = %#v, want %#v", plan.LaunchVariants.ManagedHost.Argv, wantManaged)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, []string{"resume", "--last"}) {
		t.Fatalf("managed client argv = %#v, want session-selection tokens", plan.LaunchVariants.ManagedClient.Argv)
	}

	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "gpt-5.2-codex" || plan.Resolved.Model.Source != configPath {
		t.Fatalf("resolved model = %#v", plan.Resolved.Model)
	}
	if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != "high" || plan.Resolved.Reasoning.Source != configPath {
		t.Fatalf("resolved reasoning = %#v", plan.Resolved.Reasoning)
	}
	if !plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != configPath {
		t.Fatalf("resolved yolo = %#v", plan.Resolved.Yolo)
	}
	if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != "danger-full-access" || plan.Resolved.Sandbox.Source != configPath {
		t.Fatalf("resolved sandbox = %#v", plan.Resolved.Sandbox)
	}
	if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "never" || plan.Resolved.Approval.Source != configPath {
		t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
	}
	if plan.Resolved.Profile.Value != nil || plan.Resolved.Profile.Source != "native" {
		t.Fatalf("resolved profile = %#v", plan.Resolved.Profile)
	}
	if len(plan.Resolved.MCP.Servers) != 2 || plan.Resolved.MCP.Servers[0].Name != "jira" || plan.Resolved.MCP.Servers[0].Transport != "http" || plan.Resolved.MCP.Servers[1].Name != "lldb" || plan.Resolved.MCP.Servers[1].Transport != "stdio" {
		t.Fatalf("resolved MCP servers = %#v", plan.Resolved.MCP.Servers)
	}
	if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"JIRA_TOKEN"}) {
		t.Fatalf("RequiredEnvNames = %#v", plan.RequiredEnvNames)
	}

	repeated, err := BuildPrimarySessionLaunchPlan("codex", project, home, userArgs, producer, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("repeated BuildPrimarySessionLaunchPlan: %v", err)
	}
	firstJSON, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	secondJSON, err := json.Marshal(repeated)
	if err != nil {
		t.Fatalf("marshal repeated plan: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("plan is not deterministic:\n%s\n%s", firstJSON, secondJSON)
	}
	if strings.Contains(string(firstJSON), "must-not-be-read-or-serialized") {
		t.Fatalf("plan leaked secret env value:\n%s", firstJSON)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexExplicitCLISuppressesProjectPolicy(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--model", "gpt-6", "-d"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "gpt-6" || plan.Resolved.Model.Source != "cli:--model" {
		t.Fatalf("resolved model = %#v", plan.Resolved.Model)
	}
	if !plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != "wrapper:-d" {
		t.Fatalf("resolved yolo = %#v", plan.Resolved.Yolo)
	}
	managed := strings.Join(plan.LaunchVariants.ManagedHost.Argv, " ")
	if !strings.Contains(managed, "model=\"gpt-6\"") {
		t.Fatalf("managed argv missing explicit model: %#v", plan.LaunchVariants.ManagedHost.Argv)
	}
	if strings.Contains(managed, codexDangerouslyBypassApprovalsAndSandbox) {
		t.Fatalf("managed argv must not carry the interactive bypass flag: %#v", plan.LaunchVariants.ManagedHost.Argv)
	}
	interactive := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
	if !strings.Contains(interactive, codexDangerouslyBypassApprovalsAndSandbox) {
		t.Fatalf("interactive argv missing bypass flag: %#v", plan.LaunchVariants.Interactive.Argv)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexExplicitProfile(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--profile", "speed"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Resolved.Profile.Value == nil || *plan.Resolved.Profile.Value != "speed" || plan.Resolved.Profile.Source != "cli:--profile" {
		t.Fatalf("resolved profile = %#v", plan.Resolved.Profile)
	}
	if plan.Resolved.Model.Value != nil || plan.Resolved.Model.Source != "cli:--profile" {
		t.Fatalf("resolved model with explicit profile = %#v", plan.Resolved.Model)
	}
	argv := plan.LaunchVariants.ManagedHost.Argv
	if len(argv) < 3 || argv[len(argv)-3] != "--profile" || argv[len(argv)-2] != "speed" || argv[len(argv)-1] != "app-server" {
		t.Fatalf("managed argv = %#v, want --profile speed before app-server", argv)
	}
}

func TestBuildPrimarySessionLaunchPlanClaudeParity(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)
	producer := ChildLaunchCompositionProducer{Version: "v2.0.0", Commit: "abc123"}
	userArgs := []string{"--continue"}

	plan, err := BuildPrimarySessionLaunchPlan("claude", project, home, userArgs, producer, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Provider != "claude" || plan.Executable != "/fake/bin/claude" {
		t.Fatalf("plan = %#v", plan)
	}

	launch, err := BuildClaudeLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildClaudeLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}
	if plan.LaunchVariants.ManagedHost.Kind != PrimarySessionManagedHostKindClaudePTY {
		t.Fatalf("managed host kind = %q", plan.LaunchVariants.ManagedHost.Kind)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, plan.LaunchVariants.Interactive.Argv) {
		t.Fatalf("claude managed host argv must equal interactive argv:\n%#v\n%#v", plan.LaunchVariants.ManagedHost.Argv, plan.LaunchVariants.Interactive.Argv)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, []string{}) {
		t.Fatalf("claude managed client argv = %#v, want empty", plan.LaunchVariants.ManagedClient.Argv)
	}

	interactive := plan.LaunchVariants.Interactive.Argv
	if len(interactive) < 5 || interactive[0] != "--mcp-config" {
		t.Fatalf("interactive argv = %#v", interactive)
	}
	joined := strings.Join(interactive, " ")
	if !strings.Contains(joined, "--model claude-opus-4-8") || !strings.Contains(joined, claudeDangerouslySkipPermissions) || !strings.Contains(joined, "--continue") {
		t.Fatalf("interactive argv = %#v", interactive)
	}

	configPath, _ := canonicalPrimarySessionFixturePaths(t, plan)
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "claude-opus-4-8" || plan.Resolved.Model.Source != configPath {
		t.Fatalf("resolved model = %#v", plan.Resolved.Model)
	}
	if !plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != configPath {
		t.Fatalf("resolved yolo = %#v", plan.Resolved.Yolo)
	}
	if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "bypass-permissions" {
		t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
	}
	if plan.Resolved.Reasoning.Value != nil || plan.Resolved.Reasoning.Source != "native" {
		t.Fatalf("resolved reasoning = %#v, want native", plan.Resolved.Reasoning)
	}
	for name, resolved := range map[string]PrimarySessionResolvedString{
		"sandbox": plan.Resolved.Sandbox,
		"profile": plan.Resolved.Profile,
	} {
		if resolved.Value != nil || resolved.Source != "not_applicable" {
			t.Fatalf("resolved %s = %#v, want not_applicable", name, resolved)
		}
	}
	if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"JIRA_TOKEN"}) {
		t.Fatalf("RequiredEnvNames = %#v", plan.RequiredEnvNames)
	}
}

func TestBuildPrimarySessionLaunchPlanEmptyProjectEmitsRequiredEmptyCollections(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			plan, err := BuildPrimarySessionLaunchPlan(provider, project, home, nil, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if err != nil {
				t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
			}
			encoded, err := json.Marshal(plan)
			if err != nil {
				t.Fatalf("marshal plan: %v", err)
			}
			for _, want := range []string{`"argv":[]`, `"servers":[]`, `"required_env_names":[]`, `"sources":[]`} {
				if !strings.Contains(string(encoded), want) {
					t.Fatalf("plan missing %s: %s", want, encoded)
				}
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanProviderExecutableNotFound(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	_, err := BuildPrimarySessionLaunchPlan("codex", project, home, nil, ChildLaunchCompositionProducer{}, func(string) (string, error) {
		return "", errors.New("executable file not found")
	})
	if err == nil {
		t.Fatal("BuildPrimarySessionLaunchPlan succeeded without provider executable")
	}
	var composeErr *PrimarySessionComposeError
	if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorProviderExecutableNotFound {
		t.Fatalf("error = %#v", err)
	}
}

func TestBuildPrimarySessionLaunchPlanRejectsUnknownProvider(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	_, err := BuildPrimarySessionLaunchPlan("gemini", project, home, nil, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err == nil {
		t.Fatal("BuildPrimarySessionLaunchPlan accepted unknown provider")
	}
}

func TestBuildPrimarySessionLaunchPlanInvalidProjectConfiguration(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, projectConfigFileName), "[mcp\nenabled_servers = [\"jira\"]\n")

	_, err := BuildPrimarySessionLaunchPlan("codex", project, home, nil, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err == nil {
		t.Fatal("BuildPrimarySessionLaunchPlan succeeded with invalid project configuration")
	}
	var composeErr *PrimarySessionComposeError
	if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorInvalidProjectConfiguration {
		t.Fatalf("error = %#v", err)
	}
}

func writePrimarySessionNoYoloFixture(t *testing.T, project string) {
	t.Helper()
	configDir := filepath.Join(project, ".agents", ".configs")
	mustMkdir(t, configDir)
	mustWrite(t, filepath.Join(configDir, projectConfigFileName), `
[agents.codex.primary_session]
model = "gpt-5.2-codex"

[agents.claude.primary_session]
model = "claude-opus-4-8"
`)
}

func TestBuildPrimarySessionLaunchPlanCodexNativePolicyFlagsResolve(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionNoYoloFixture(t, project)
	userArgs := []string{"--sandbox", "read-only", "--ask-for-approval", "on-request"}

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, userArgs, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != "read-only" || plan.Resolved.Sandbox.Source != "cli:--sandbox" {
		t.Fatalf("resolved sandbox = %#v", plan.Resolved.Sandbox)
	}
	if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "on-request" || plan.Resolved.Approval.Source != "cli:--ask-for-approval" {
		t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
	}
	launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}
	joined := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
	if !strings.Contains(joined, "--sandbox read-only") || !strings.Contains(joined, "--ask-for-approval on-request") {
		t.Fatalf("interactive argv lost native policy flags: %#v", plan.LaunchVariants.Interactive.Argv)
	}
	managed := strings.Join(plan.LaunchVariants.ManagedHost.Argv, " ")
	if strings.Contains(managed, "--sandbox") || strings.Contains(managed, "--ask-for-approval") {
		t.Fatalf("managed argv must keep sandbox/approval out of the app-server host: %#v", plan.LaunchVariants.ManagedHost.Argv)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexNativePolicyFlagForms(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name           string
		args           []string
		sandbox        string
		sandboxSource  string
		approval       string
		approvalSource string
	}{
		{name: "long_equals", args: []string{"--sandbox=workspace-write", "--ask-for-approval=never"}, sandbox: "workspace-write", sandboxSource: "cli:--sandbox", approval: "never", approvalSource: "cli:--ask-for-approval"},
		{name: "short_spaced", args: []string{"-s", "read-only", "-a", "untrusted"}, sandbox: "read-only", sandboxSource: "cli:-s", approval: "untrusted", approvalSource: "cli:-a"},
		{name: "short_equals", args: []string{"-s=danger-full-access", "-a=never"}, sandbox: "danger-full-access", sandboxSource: "cli:-s", approval: "never", approvalSource: "cli:-a"},
		{name: "short_attached", args: []string{"-sread-only", "-aon-request"}, sandbox: "read-only", sandboxSource: "cli:-s", approval: "on-request", approvalSource: "cli:-a"},
		{name: "config_overrides", args: []string{"-c", `sandbox_mode="workspace-write"`, "-c", `approval_policy="on-request"`}, sandbox: "workspace-write", sandboxSource: "cli:-c sandbox_mode", approval: "on-request", approvalSource: "cli:-c approval_policy"},
		{name: "config_last_wins", args: []string{"-c", `sandbox_mode="read-only"`, "-c", `sandbox_mode="workspace-write"`}, sandbox: "workspace-write", sandboxSource: "cli:-c sandbox_mode"},
		{name: "flag_beats_config_before", args: []string{"--sandbox", "read-only", "-c", `sandbox_mode="workspace-write"`}, sandbox: "read-only", sandboxSource: "cli:--sandbox"},
		{name: "flag_beats_config_after", args: []string{"-c", `sandbox_mode="workspace-write"`, "--sandbox", "read-only"}, sandbox: "read-only", sandboxSource: "cli:--sandbox"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if err != nil {
				t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
			}
			if testCase.sandbox != "" {
				if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != testCase.sandbox || plan.Resolved.Sandbox.Source != testCase.sandboxSource {
					t.Fatalf("resolved sandbox = %#v, want %q from %q", plan.Resolved.Sandbox, testCase.sandbox, testCase.sandboxSource)
				}
			}
			if testCase.approval != "" {
				if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != testCase.approval || plan.Resolved.Approval.Source != testCase.approvalSource {
					t.Fatalf("resolved approval = %#v, want %q from %q", plan.Resolved.Approval, testCase.approval, testCase.approvalSource)
				}
			}
			launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, testCase.args)
			if err != nil {
				t.Fatalf("BuildCodexLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
				t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanCodexExplicitPolicySuppressesProjectYolo(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--sandbox", "read-only"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != "cli:--sandbox" {
		t.Fatalf("resolved yolo = %#v, want suppressed by cli:--sandbox", plan.Resolved.Yolo)
	}
	if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != "read-only" || plan.Resolved.Sandbox.Source != "cli:--sandbox" {
		t.Fatalf("resolved sandbox = %#v", plan.Resolved.Sandbox)
	}
	if plan.Resolved.Approval.Value != nil || plan.Resolved.Approval.Source != "native" {
		t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
	}
	joined := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
	if strings.Contains(joined, codexDangerouslyBypassApprovalsAndSandbox) {
		t.Fatalf("interactive argv still composes the bypass flag Codex would reject next to --sandbox: %#v", plan.LaunchVariants.Interactive.Argv)
	}
	launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, []string{"--sandbox", "read-only"})
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}
	if launch.PrimarySessionResolution.YoloMode.ProjectApplication != CodexPrimarySessionSuppressedByCLI {
		t.Fatalf("yolo project application = %#v", launch.PrimarySessionResolution.YoloMode)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexNativePolicyErrors(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "duplicate_sandbox", args: []string{"--sandbox", "read-only", "--sandbox", "workspace-write"}, want: "cannot be used multiple times"},
		{name: "duplicate_sandbox_equal_values", args: []string{"-s", "read-only", "--sandbox=read-only"}, want: "cannot be used multiple times"},
		{name: "duplicate_approval", args: []string{"-a", "never", "-a", "on-request"}, want: "cannot be used multiple times"},
		{name: "danger_conflicts_with_sandbox", args: []string{"-d", "--sandbox", "read-only"}, want: "cannot be used with --sandbox"},
		{name: "danger_conflicts_with_approval", args: []string{codexDangerouslyBypassApprovalsAndSandbox, "-a", "never"}, want: "cannot be used with --ask-for-approval"},
		{name: "sandbox_missing_value", args: []string{"--sandbox"}, want: "a value is required"},
		{name: "sandbox_flag_like_value", args: []string{"--sandbox", "--model"}, want: "a value is required"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			_, err := BuildPrimarySessionLaunchPlan("codex", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want contains %q", err, testCase.want)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanClaudeNativePolicyFlagsResolve(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionNoYoloFixture(t, project)
	userArgs := []string{"--effort", "xhigh", "--permission-mode", "plan"}

	plan, err := BuildPrimarySessionLaunchPlan("claude", project, home, userArgs, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != "xhigh" || plan.Resolved.Reasoning.Source != "cli:--effort" {
		t.Fatalf("resolved reasoning = %#v", plan.Resolved.Reasoning)
	}
	if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "plan" || plan.Resolved.Approval.Source != "cli:--permission-mode" {
		t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
	}
	launch, err := BuildClaudeLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildClaudeLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}
	joined := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
	if !strings.Contains(joined, "--effort xhigh") || !strings.Contains(joined, "--permission-mode plan") {
		t.Fatalf("interactive argv lost native policy flags: %#v", plan.LaunchVariants.Interactive.Argv)
	}
}

func TestBuildPrimarySessionLaunchPlanClaudeNativePolicyForms(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name      string
		args      []string
		reasoning string
		approval  string
	}{
		{name: "equals_forms", args: []string{"--effort=high", "--permission-mode=acceptEdits"}, reasoning: "high", approval: "acceptEdits"},
		{name: "duplicates_last_wins", args: []string{"--effort", "high", "--effort=xhigh", "--permission-mode", "plan", "--permission-mode", "acceptEdits"}, reasoning: "xhigh", approval: "acceptEdits"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			plan, err := BuildPrimarySessionLaunchPlan("claude", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if err != nil {
				t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
			}
			if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != testCase.reasoning || plan.Resolved.Reasoning.Source != "cli:--effort" {
				t.Fatalf("resolved reasoning = %#v, want %q", plan.Resolved.Reasoning, testCase.reasoning)
			}
			if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != testCase.approval || plan.Resolved.Approval.Source != "cli:--permission-mode" {
				t.Fatalf("resolved approval = %#v, want %q", plan.Resolved.Approval, testCase.approval)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanClaudePermissionModeSuppressesProjectYolo(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)

	plan, err := BuildPrimarySessionLaunchPlan("claude", project, home, []string{"--permission-mode", "plan"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != "cli:--permission-mode" {
		t.Fatalf("resolved yolo = %#v, want suppressed by cli:--permission-mode", plan.Resolved.Yolo)
	}
	if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "plan" || plan.Resolved.Approval.Source != "cli:--permission-mode" {
		t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
	}
	joined := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
	if strings.Contains(joined, claudeDangerouslySkipPermissions) {
		t.Fatalf("interactive argv still composes skip-permissions, which overrides the explicit mode at runtime: %#v", plan.LaunchVariants.Interactive.Argv)
	}
	launch, err := BuildClaudeLaunchPlan(plan.ProjectDir, home, []string{"--permission-mode", "plan"})
	if err != nil {
		t.Fatalf("BuildClaudeLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}
	if launch.PrimarySessionResolution.YoloMode.ProjectApplication != ClaudePrimarySessionSuppressedByCLI {
		t.Fatalf("yolo project application = %#v", launch.PrimarySessionResolution.YoloMode)
	}
}

func TestBuildPrimarySessionLaunchPlanClaudeExplicitDangerOverridesPermissionMode(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()

	plan, err := BuildPrimarySessionLaunchPlan("claude", project, home, []string{"-d", "--permission-mode", "plan"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if !plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != "wrapper:-d" {
		t.Fatalf("resolved yolo = %#v", plan.Resolved.Yolo)
	}
	if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "bypass-permissions" || plan.Resolved.Approval.Source != "wrapper:-d" {
		t.Fatalf("resolved approval = %#v, want bypass-permissions winning over explicit mode", plan.Resolved.Approval)
	}
	joined := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
	if !strings.Contains(joined, claudeDangerouslySkipPermissions) || !strings.Contains(joined, "--permission-mode plan") {
		t.Fatalf("interactive argv = %#v", plan.LaunchVariants.Interactive.Argv)
	}
}

func TestBuildPrimarySessionLaunchPlanClaudeNativePolicyErrors(t *testing.T) {
	home := t.TempDir()
	for _, testCase := range []struct {
		name string
		args []string
		want string
	}{
		{name: "effort_missing_value", args: []string{"--effort"}, want: "--effort requires a value"},
		{name: "permission_mode_missing_value", args: []string{"--permission-mode"}, want: "--permission-mode requires a value"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			_, err := BuildPrimarySessionLaunchPlan("claude", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want contains %q", err, testCase.want)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanCodexManagedHostPreservesAppServerGlobals(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)
	userArgs := []string{
		"-c", `service_tier="fast"`,
		"--enable", "web_search",
		"--disable", "response_storage",
		"--strict-config",
		"--sandbox", "read-only",
		"-c", "approval_policy=never",
		"resume", "--last",
	}

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, userArgs, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}

	wantManaged := []string{
		"-c", "mcp_servers.jira.url=\"https://jira.example/mcp\"",
		"-c", "mcp_servers.jira.bearer_token_env_var=\"JIRA_TOKEN\"",
		"-c", "mcp_servers.lldb.command=\"lldb-mcp\"",
		"-c", "mcp_servers.lldb.args=[\"--socket\", \"auto\"]",
		"-c", "model=\"gpt-5.2-codex\"",
		"-c", "model_reasoning_effort=\"high\"",
		"-c", `service_tier="fast"`,
		"--enable", "web_search",
		"--disable", "response_storage",
		"--strict-config",
		"app-server",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, wantManaged) {
		t.Fatalf("managed host argv = %#v, want %#v", plan.LaunchVariants.ManagedHost.Argv, wantManaged)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, []string{"resume", "--last"}) {
		t.Fatalf("managed client argv = %#v, want session-selection tokens", plan.LaunchVariants.ManagedClient.Argv)
	}
	if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != "read-only" || plan.Resolved.Sandbox.Source != "cli:--sandbox" {
		t.Fatalf("resolved sandbox = %#v", plan.Resolved.Sandbox)
	}
	if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != "never" || plan.Resolved.Approval.Source != "cli:-c approval_policy" {
		t.Fatalf("resolved approval = %#v", plan.Resolved.Approval)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexManagedHostGlobalArgForms(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)
	userArgs := []string{
		"--model=gpt-6",
		"--enable=web_search",
		`-c=service_tier="fast"`,
		"--config", "sandbox_mode=read-only",
		"--profile=speed",
		"resume",
	}

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, userArgs, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}

	wantManaged := []string{
		"-c", "mcp_servers.jira.url=\"https://jira.example/mcp\"",
		"-c", "mcp_servers.jira.bearer_token_env_var=\"JIRA_TOKEN\"",
		"-c", "mcp_servers.lldb.command=\"lldb-mcp\"",
		"-c", "mcp_servers.lldb.args=[\"--socket\", \"auto\"]",
		"-c", "model=\"gpt-6\"",
		"--enable=web_search",
		`-c=service_tier="fast"`,
		"--profile=speed",
		"app-server",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, wantManaged) {
		t.Fatalf("managed host argv = %#v, want %#v", plan.LaunchVariants.ManagedHost.Argv, wantManaged)
	}
	if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != "read-only" || plan.Resolved.Sandbox.Source != "cli:--config sandbox_mode" {
		t.Fatalf("resolved sandbox = %#v", plan.Resolved.Sandbox)
	}
}

func TestCodexManagedArgvSplitsHostPolicyAndClientClasses(t *testing.T) {
	interactive := []string{
		"--model", "gpt-5.2-codex",
		codexDangerouslyBypassApprovalsAndSandbox,
		"-c", `service_tier="fast"`,
		"--config", `approval_policy="never"`,
		"-sread-only",
		"-a=untrusted",
		"--sandbox=workspace-write",
		"--ask-for-approval", "on-request",
		"--enable", "web_search",
		"--disable=response_storage",
		"--strict-config",
		"-p", "speed",
		"exec", "--json",
		"--",
		"prompt text", "--strict-config",
	}
	host, client := codexManagedArgvSplit(interactive)
	wantHost := []string{
		"-c", "model=\"gpt-5.2-codex\"",
		"-c", `service_tier="fast"`,
		"--enable", "web_search",
		"--disable=response_storage",
		"--strict-config",
		"-p", "speed",
		"app-server",
	}
	if !reflect.DeepEqual(host, wantHost) {
		t.Fatalf("host argv = %#v, want %#v", host, wantHost)
	}
	wantClient := []string{
		"exec", "--json",
		"--",
		"prompt text", "--strict-config",
	}
	if !reflect.DeepEqual(client, wantClient) {
		t.Fatalf("client argv = %#v, want %#v", client, wantClient)
	}
}

// Every top-level Codex option class the provider parser accepts must land in
// host argv, in client argv, or in resolved session policy — never be silently
// dropped. This is the reviewer-reproduced class set from codex-cli 0.145.0.
func TestCodexManagedArgvSplitIsTotalOverAcceptedGlobalClasses(t *testing.T) {
	interactive := []string{
		"--oss",
		"--local-provider", "ollama",
		"--dangerously-bypass-hook-trust",
		"-C", "/tmp",
		"--add-dir", "/tmp",
		"--search",
		"--no-alt-screen",
		"-i", "shot.png",
		"--remote", "ws://127.0.0.1:4500",
		"--remote-auth-token-env", "CODEX_REMOTE_TOKEN",
		"-mgpt-5.4",
		"-pspeed",
		"-sread-only",
		"resume", "--last",
	}
	host, client := codexManagedArgvSplit(interactive)
	wantHost := []string{
		"--oss",
		"--local-provider", "ollama",
		"--dangerously-bypass-hook-trust",
		"--search",
		"-c", "model=\"gpt-5.4\"",
		"-pspeed",
		"app-server",
	}
	if !reflect.DeepEqual(host, wantHost) {
		t.Fatalf("host argv = %#v, want %#v", host, wantHost)
	}
	wantClient := []string{
		"-C", "/tmp",
		"--add-dir", "/tmp",
		"--no-alt-screen",
		"-i", "shot.png",
		"--remote", "ws://127.0.0.1:4500",
		"--remote-auth-token-env", "CODEX_REMOTE_TOKEN",
		"resume", "--last",
	}
	if !reflect.DeepEqual(client, wantClient) {
		t.Fatalf("client argv = %#v, want %#v", client, wantClient)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexAttachedModelProfileForms(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	userArgs := []string{"-mgpt-5.4", "-pspeed"}

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, userArgs, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, []string{"-mgpt-5.4", "-pspeed"}) {
		t.Fatalf("interactive argv = %#v", plan.LaunchVariants.Interactive.Argv)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "gpt-5.4" || plan.Resolved.Model.Source != "cli:-m" {
		t.Fatalf("resolved model = %#v, want gpt-5.4 from cli:-m", plan.Resolved.Model)
	}
	if plan.Resolved.Profile.Value == nil || *plan.Resolved.Profile.Value != "speed" || plan.Resolved.Profile.Source != "cli:-p" {
		t.Fatalf("resolved profile = %#v, want speed from cli:-p", plan.Resolved.Profile)
	}
	wantManaged := []string{"-c", "model=\"gpt-5.4\"", "-pspeed", "app-server"}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, wantManaged) {
		t.Fatalf("managed host argv = %#v, want %#v", plan.LaunchVariants.ManagedHost.Argv, wantManaged)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, []string{}) {
		t.Fatalf("managed client argv = %#v, want empty", plan.LaunchVariants.ManagedClient.Argv)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexAttachedFormsFollowExplicitSelectionRules(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writePrimarySessionFixture(t, project)

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"-mgpt-5.4", "-mgpt-5.4"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	// The attached form suppresses the composed project model exactly like the
	// spaced form, and an equal duplicate normalizes to one occurrence.
	joined := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
	if strings.Contains(joined, "gpt-5.2-codex") {
		t.Fatalf("project model not suppressed by attached CLI model: %#v", plan.LaunchVariants.Interactive.Argv)
	}
	if strings.Count(joined, "-mgpt-5.4") != 1 {
		t.Fatalf("equal attached duplicates not normalized: %#v", plan.LaunchVariants.Interactive.Argv)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "gpt-5.4" || plan.Resolved.Model.Source != "cli:-m" {
		t.Fatalf("resolved model = %#v", plan.Resolved.Model)
	}

	// Conflicting values across attached and spaced forms fail closed exactly
	// like the spaced forms alone.
	_, err = BuildPrimarySessionLaunchPlan("codex", project, home, []string{"-mgpt-5.4", "--model", "other"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err == nil || !strings.Contains(err.Error(), "conflicting explicit Codex values") {
		t.Fatalf("conflicting attached/spaced model error = %v", err)
	}
}

func TestBuildPrimarySessionLaunchPlanCodexManagedClientPreservesThreadAndClientTokens(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	userArgs := []string{
		"--oss",
		"--local-provider", "ollama",
		"--dangerously-bypass-hook-trust",
		"-C", "/tmp",
		"--add-dir", "/tmp",
		"--search",
		"--no-alt-screen",
		"resume", "--last",
	}

	plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, userArgs, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, userArgs)
	if err != nil {
		t.Fatalf("BuildCodexLaunchPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
		t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
	}
	wantHost := []string{
		"--oss",
		"--local-provider", "ollama",
		"--dangerously-bypass-hook-trust",
		"--search",
		"app-server",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedHost.Argv, wantHost) {
		t.Fatalf("managed host argv = %#v, want %#v", plan.LaunchVariants.ManagedHost.Argv, wantHost)
	}
	wantClient := []string{
		"-C", "/tmp",
		"--add-dir", "/tmp",
		"--no-alt-screen",
		"resume", "--last",
	}
	if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, wantClient) {
		t.Fatalf("managed client argv = %#v, want %#v", plan.LaunchVariants.ManagedClient.Argv, wantClient)
	}
}

func TestBuildPrimarySessionLaunchPlanProviderArgumentErrorsUseDedicatedCode(t *testing.T) {
	home := t.TempDir()
	for _, testCase := range []struct {
		name     string
		provider string
		args     []string
	}{
		{name: "codex_duplicate_sandbox", provider: "codex", args: []string{"-s", "read-only", "-s", "workspace-write"}},
		{name: "codex_missing_model_value", provider: "codex", args: []string{"--model"}},
		{name: "codex_missing_short_model_value", provider: "codex", args: []string{"-m"}},
		{name: "codex_flag_like_model_value", provider: "codex", args: []string{"--model", "-c", "foo=bar"}},
		{name: "codex_missing_profile_value", provider: "codex", args: []string{"--profile"}},
		{name: "codex_missing_short_profile_value", provider: "codex", args: []string{"-p"}},
		{name: "codex_repeated_profile", provider: "codex", args: []string{"--profile", "fast", "-p", "slow"}},
		{name: "codex_repeated_equal_profile", provider: "codex", args: []string{"--profile=fast", "--profile=fast"}},
		{name: "codex_missing_config_value", provider: "codex", args: []string{"-c"}},
		{name: "codex_missing_enable_value", provider: "codex", args: []string{"--enable"}},
		{name: "codex_missing_disable_value", provider: "codex", args: []string{"--disable"}},
		{name: "codex_missing_local_provider_value", provider: "codex", args: []string{"--local-provider"}},
		{name: "codex_invalid_sandbox_value", provider: "codex", args: []string{"--sandbox", "banana"}},
		{name: "codex_invalid_approval_value", provider: "codex", args: []string{"-a", "on-failure"}},
		{name: "codex_invalid_config_sandbox_value", provider: "codex", args: []string{"-c", `sandbox_mode="banana"`}},
		{name: "codex_invalid_config_approval_value", provider: "codex", args: []string{"-c", `approval_policy="banana"`}},
		{name: "claude_missing_permission_mode_value", provider: "claude", args: []string{"--permission-mode"}},
		{name: "claude_invalid_permission_mode_value", provider: "claude", args: []string{"--permission-mode", "banana"}},
		{name: "claude_missing_model_value", provider: "claude", args: []string{"--model"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			_, err := BuildPrimarySessionLaunchPlan(testCase.provider, project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			var composeErr *PrimarySessionComposeError
			if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorInvalidProviderArguments {
				t.Fatalf("error = %#v, want code %q", err, PrimarySessionErrorInvalidProviderArguments)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanCodexPolicyValueDomains(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name           string
		args           []string
		wantErr        string
		sandbox        string
		sandboxSource  string
		approval       string
		approvalSource string
	}{
		// Typed clap flags validate every spelling against the flag enum
		// (probe exit 2 on codex-cli 0.145.0).
		{name: "sandbox_flag_spaced_invalid", args: []string{"--sandbox", "banana"}, wantErr: "invalid value \"banana\" for the Codex argument --sandbox"},
		{name: "sandbox_flag_equals_invalid", args: []string{"--sandbox=banana"}, wantErr: "invalid value \"banana\" for the Codex argument --sandbox"},
		{name: "sandbox_flag_short_attached_invalid", args: []string{"-sbanana"}, wantErr: "invalid value \"banana\" for the Codex argument -s"},
		{name: "sandbox_flag_short_equals_invalid", args: []string{"-s=banana"}, wantErr: "invalid value \"banana\" for the Codex argument -s"},
		{name: "sandbox_flag_case_sensitive", args: []string{"--sandbox", "Read-Only"}, wantErr: "invalid value \"Read-Only\" for the Codex argument --sandbox"},
		{name: "approval_flag_spaced_invalid", args: []string{"--ask-for-approval", "banana"}, wantErr: "invalid value \"banana\" for the Codex argument --ask-for-approval"},
		{name: "approval_flag_equals_invalid", args: []string{"--ask-for-approval=banana"}, wantErr: "invalid value \"banana\" for the Codex argument --ask-for-approval"},
		{name: "approval_flag_short_attached_invalid", args: []string{"-abanana"}, wantErr: "invalid value \"banana\" for the Codex argument -a"},
		// on-failure and granular are config-only variants; the typed flag
		// rejects them (probe exit 2).
		{name: "approval_flag_rejects_config_only_variant", args: []string{"--ask-for-approval", "on-failure"}, wantErr: "invalid value \"on-failure\" for the Codex argument --ask-for-approval"},
		// -c/--config overrides validate against the config deserialization
		// domain: last override per key wins, earlier repeats are masked, and
		// a typed flag does not mask an invalid override (probe exit 1).
		{name: "config_sandbox_invalid", args: []string{"-c", `sandbox_mode="banana"`}, wantErr: "unknown variant \"banana\" for the Codex config override sandbox_mode"},
		{name: "config_sandbox_invalid_masked_by_last_wins", args: []string{"-c", `sandbox_mode="banana"`, "-c", `sandbox_mode="read-only"`}, sandbox: "read-only", sandboxSource: "cli:-c sandbox_mode"},
		{name: "config_sandbox_invalid_winner", args: []string{"-c", `sandbox_mode="read-only"`, "-c", `sandbox_mode="banana"`}, wantErr: "unknown variant \"banana\" for the Codex config override sandbox_mode"},
		{name: "config_sandbox_invalid_not_masked_by_flag", args: []string{"--sandbox", "read-only", "-c", `sandbox_mode="banana"`}, wantErr: "unknown variant \"banana\" for the Codex config override sandbox_mode"},
		{name: "config_sandbox_non_string", args: []string{"-c", "sandbox_mode=true"}, wantErr: "unknown variant \"true\" for the Codex config override sandbox_mode"},
		{name: "config_sandbox_unquoted_valid", args: []string{"-c", "sandbox_mode=read-only"}, sandbox: "read-only", sandboxSource: "cli:-c sandbox_mode"},
		{name: "config_approval_invalid", args: []string{"--config", `approval_policy="banana"`}, wantErr: "unknown variant \"banana\" for the Codex config override approval_policy"},
		{name: "config_approval_accepts_on_failure", args: []string{"-c", `approval_policy="on-failure"`}, approval: "on-failure", approvalSource: "cli:-c approval_policy"},
		{name: "config_approval_accepts_granular", args: []string{"-c", `approval_policy="granular"`}, approval: "granular", approvalSource: "cli:-c approval_policy"},
		{name: "config_equals_spelling_invalid", args: []string{`--config=sandbox_mode="banana"`}, wantErr: "unknown variant \"banana\" for the Codex config override sandbox_mode"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, testCase.wantErr)
				}
				var composeErr *PrimarySessionComposeError
				if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorInvalidProviderArguments {
					t.Fatalf("error = %#v, want code %q", err, PrimarySessionErrorInvalidProviderArguments)
				}
				// The launcher path shares the parser, so it fails closed on
				// the same invocation instead of launching an argv the
				// provider rejects.
				if _, launchErr := BuildCodexLaunchPlan(project, home, testCase.args); launchErr == nil || !strings.Contains(launchErr.Error(), testCase.wantErr) {
					t.Fatalf("launcher error = %v, want contains %q", launchErr, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
			}
			if testCase.sandbox != "" {
				if plan.Resolved.Sandbox.Value == nil || *plan.Resolved.Sandbox.Value != testCase.sandbox || plan.Resolved.Sandbox.Source != testCase.sandboxSource {
					t.Fatalf("resolved sandbox = %#v, want %q from %q", plan.Resolved.Sandbox, testCase.sandbox, testCase.sandboxSource)
				}
			}
			if testCase.approval != "" {
				if plan.Resolved.Approval.Value == nil || *plan.Resolved.Approval.Value != testCase.approval || plan.Resolved.Approval.Source != testCase.approvalSource {
					t.Fatalf("resolved approval = %#v, want %q from %q", plan.Resolved.Approval, testCase.approval, testCase.approvalSource)
				}
			}
			launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, testCase.args)
			if err != nil {
				t.Fatalf("BuildCodexLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
				t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanClaudePermissionModeDomain(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "spaced_invalid", args: []string{"--permission-mode", "banana"}},
		{name: "equals_invalid", args: []string{"--permission-mode=banana"}},
		// Commander validates every occurrence, so a later valid mode does
		// not mask an earlier invalid one (probe exit 1).
		{name: "invalid_first_occurrence", args: []string{"--permission-mode", "banana", "--permission-mode", "plan"}},
		{name: "case_sensitive", args: []string{"--permission-mode", "Plan"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			_, err := BuildPrimarySessionLaunchPlan("claude", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			var composeErr *PrimarySessionComposeError
			if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorInvalidProviderArguments {
				t.Fatalf("error = %#v, want code %q", err, PrimarySessionErrorInvalidProviderArguments)
			}
			if !strings.Contains(err.Error(), "--permission-mode value") || !strings.Contains(err.Error(), "is invalid") {
				t.Fatalf("error = %v, want permission-mode domain diagnostic", err)
			}
			// The launcher path shares the parser and fails closed on the
			// same invocation Claude itself rejects with exit 1.
			if _, launchErr := BuildClaudeLaunchPlan(project, home, testCase.args); launchErr == nil {
				t.Fatalf("launcher accepted %v", testCase.args)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanClaudeEffortFallbackAndCanonicalization(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name           string
		args           []string
		reasoningValue string
		wantNative     bool
		argvContains   string
	}{
		// Claude ignores an unknown effort with a warning and applies its own
		// default (probe exit 0), so the token stays in argv but no effective
		// value is claimed.
		{name: "unknown_effort_reports_native", args: []string{"--effort", "banana"}, wantNative: true, argvContains: "--effort banana"},
		{name: "unknown_effort_equals_reports_native", args: []string{"--effort=banana"}, wantNative: true, argvContains: "--effort=banana"},
		// Effort matches case-insensitively (probe: no warning for HIGH or
		// xHiGh), so a recognized token canonicalizes to its domain value.
		{name: "mixed_case_effort_canonicalizes", args: []string{"--effort", "HIGH"}, reasoningValue: "high", argvContains: "--effort HIGH"},
		{name: "mixed_case_xhigh_canonicalizes", args: []string{"--effort", "xHiGh"}, reasoningValue: "xhigh"},
		// Last occurrence wins, including its recognized-ness.
		{name: "last_unknown_wins_over_valid", args: []string{"--effort", "high", "--effort", "banana"}, wantNative: true},
		{name: "last_valid_wins_over_unknown", args: []string{"--effort", "banana", "--effort=xhigh"}, reasoningValue: "xhigh"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			plan, err := BuildPrimarySessionLaunchPlan("claude", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if err != nil {
				t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
			}
			if testCase.wantNative {
				if plan.Resolved.Reasoning.Value != nil || plan.Resolved.Reasoning.Source != "native" {
					t.Fatalf("resolved reasoning = %#v, want provider-native fallback", plan.Resolved.Reasoning)
				}
			} else {
				if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != testCase.reasoningValue || plan.Resolved.Reasoning.Source != "cli:--effort" {
					t.Fatalf("resolved reasoning = %#v, want %q from cli:--effort", plan.Resolved.Reasoning, testCase.reasoningValue)
				}
			}
			if testCase.argvContains != "" {
				joined := strings.Join(plan.LaunchVariants.Interactive.Argv, " ")
				if !strings.Contains(joined, testCase.argvContains) {
					t.Fatalf("interactive argv %#v lost the pass-through effort token %q", plan.LaunchVariants.Interactive.Argv, testCase.argvContains)
				}
			}
			launch, err := BuildClaudeLaunchPlan(plan.ProjectDir, home, testCase.args)
			if err != nil {
				t.Fatalf("BuildClaudeLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
				t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
			}
			if launch.ExplicitEffortRecognized == testCase.wantNative {
				t.Fatalf("launcher ExplicitEffortRecognized = %t, want %t", launch.ExplicitEffortRecognized, !testCase.wantNative)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanCodexProfileNameDomain(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name    string
		args    []string
		wantErr string
		profile string
		source  string
	}{
		// The Codex clap parser validates CONFIG_PROFILE_V2 values against its
		// plain profile-name syntax in every spelling (probe exit 2 on
		// codex-cli 0.145.0, "invalid --profile value ...; pass a plain name
		// such as `work`").
		{name: "equals_empty", args: []string{"--profile="}, wantErr: "invalid value \"\" for the Codex argument --profile"},
		{name: "short_equals_empty", args: []string{"-p="}, wantErr: "invalid value \"\" for the Codex argument -p"},
		{name: "equals_slash", args: []string{"--profile=foo/bar"}, wantErr: "invalid value \"foo/bar\" for the Codex argument --profile"},
		{name: "equals_space", args: []string{"--profile=a b"}, wantErr: "invalid value \"a b\" for the Codex argument --profile"},
		{name: "spaced_dot", args: []string{"--profile", "a.b"}, wantErr: "invalid value \"a.b\" for the Codex argument --profile"},
		{name: "attached_dot", args: []string{"-p."}, wantErr: "invalid value \".\" for the Codex argument -p"},
		{name: "spaced_non_ascii", args: []string{"--profile", "работа"}, wantErr: "invalid value \"работа\" for the Codex argument --profile"},
		{name: "equals_embedded_equals", args: []string{"--profile=a=b"}, wantErr: "invalid value \"a=b\" for the Codex argument --profile"},
		// Error precedence mirrors the provider (probe exit 2 for each):
		// a missing value reports before multiplicity, multiplicity reports
		// before the second occurrence's invalid value.
		{name: "repeat_then_missing_value", args: []string{"--profile", "ok", "--profile"}, wantErr: "a value is required for the Codex argument --profile"},
		{name: "repeat_then_invalid_value", args: []string{"--profile", "ok", "--profile", "foo/bar"}, wantErr: "cannot be used multiple times"},
		{name: "invalid_first_occurrence", args: []string{"--profile", "foo/bar", "--profile", "ok"}, wantErr: "invalid value \"foo/bar\" for the Codex argument --profile"},
		// Values inside the plain-name domain are accepted in every spelling
		// (probe exit 0), including leading dash/underscore via = and attached
		// forms; profile existence stays provider-native resolution.
		{name: "spaced_valid", args: []string{"--profile", "work"}, profile: "work", source: "cli:--profile"},
		{name: "equals_valid_mixed", args: []string{"--profile=A_b-1"}, profile: "A_b-1", source: "cli:--profile"},
		{name: "equals_leading_dash", args: []string{"--profile=-ab"}, profile: "-ab", source: "cli:--profile"},
		{name: "attached_valid", args: []string{"-p_ab"}, profile: "_ab", source: "cli:-p"},
		{name: "digits_only", args: []string{"-p=123"}, profile: "123", source: "cli:-p"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, testCase.wantErr)
				}
				var composeErr *PrimarySessionComposeError
				if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorInvalidProviderArguments {
					t.Fatalf("error = %#v, want code %q", err, PrimarySessionErrorInvalidProviderArguments)
				}
				// The launcher path shares the parser, so it fails closed on
				// the same invocation instead of launching an argv the
				// provider rejects.
				if _, launchErr := BuildCodexLaunchPlan(project, home, testCase.args); launchErr == nil || !strings.Contains(launchErr.Error(), testCase.wantErr) {
					t.Fatalf("launcher error = %v, want contains %q", launchErr, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
			}
			if plan.Resolved.Profile.Value == nil || *plan.Resolved.Profile.Value != testCase.profile || plan.Resolved.Profile.Source != testCase.source {
				t.Fatalf("resolved profile = %#v, want %q from %q", plan.Resolved.Profile, testCase.profile, testCase.source)
			}
			launch, err := BuildCodexLaunchPlan(plan.ProjectDir, home, testCase.args)
			if err != nil {
				t.Fatalf("BuildCodexLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, launch.Args) {
				t.Fatalf("interactive argv parity broken:\ncompose: %#v\nlaunch:  %#v", plan.LaunchVariants.Interactive.Argv, launch.Args)
			}
		})
	}
}

func TestBuildPrimarySessionLaunchPlanCodexRemoteAuthTokenEnv(t *testing.T) {
	home := t.TempDir()

	t.Run("spaced_form_surfaces_env_name", func(t *testing.T) {
		project := t.TempDir()
		t.Setenv("CODEX_REMOTE_TOKEN", "must-not-be-read-or-serialized")
		args := []string{"--remote", "ws://127.0.0.1:4500", "--remote-auth-token-env", "CODEX_REMOTE_TOKEN"}
		plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		if err != nil {
			t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
		}
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"CODEX_REMOTE_TOKEN"}) {
			t.Fatalf("RequiredEnvNames = %#v, want the remote auth env name", plan.RequiredEnvNames)
		}
		if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, args) {
			t.Fatalf("managed client argv = %#v, want %#v", plan.LaunchVariants.ManagedClient.Argv, args)
		}
		planJSON, err := json.Marshal(plan)
		if err != nil {
			t.Fatalf("marshal plan: %v", err)
		}
		if strings.Contains(string(planJSON), "must-not-be-read-or-serialized") {
			t.Fatalf("plan leaked remote auth token value:\n%s", planJSON)
		}
	})

	t.Run("equals_form_surfaces_env_name", func(t *testing.T) {
		project := t.TempDir()
		plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--remote-auth-token-env=CODEX_REMOTE_TOKEN"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		if err != nil {
			t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
		}
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"CODEX_REMOTE_TOKEN"}) {
			t.Fatalf("RequiredEnvNames = %#v, want the remote auth env name", plan.RequiredEnvNames)
		}
		if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, []string{"--remote-auth-token-env=CODEX_REMOTE_TOKEN"}) {
			t.Fatalf("managed client argv = %#v", plan.LaunchVariants.ManagedClient.Argv)
		}
	})

	t.Run("deduplicates_against_mcp_bearer_names", func(t *testing.T) {
		project := t.TempDir()
		writePrimarySessionFixture(t, project)
		plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--remote-auth-token-env", "JIRA_TOKEN"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		if err != nil {
			t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
		}
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"JIRA_TOKEN"}) {
			t.Fatalf("RequiredEnvNames = %#v, want deduplicated [JIRA_TOKEN]", plan.RequiredEnvNames)
		}
	})

	t.Run("orders_mcp_names_before_remote_name", func(t *testing.T) {
		project := t.TempDir()
		writePrimarySessionFixture(t, project)
		plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--remote-auth-token-env", "CODEX_REMOTE_TOKEN"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		if err != nil {
			t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
		}
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{"JIRA_TOKEN", "CODEX_REMOTE_TOKEN"}) {
			t.Fatalf("RequiredEnvNames = %#v, want [JIRA_TOKEN CODEX_REMOTE_TOKEN]", plan.RequiredEnvNames)
		}
	})

	t.Run("pass_through_tokens_are_not_interpreted", func(t *testing.T) {
		// The first -- is the wrapper separator the arg parser consumes; the
		// second is Codex's own separator, so everything after it is prompt
		// text that must never be read as an environment option.
		project := t.TempDir()
		args := []string{"--", "--", "--remote-auth-token-env", "PROMPT_TEXT_NOT_AN_OPTION"}
		wantPassThrough := []string{"--", "--remote-auth-token-env", "PROMPT_TEXT_NOT_AN_OPTION"}
		plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		if err != nil {
			t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
		}
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{}) {
			t.Fatalf("RequiredEnvNames = %#v, want empty for pass-through tokens", plan.RequiredEnvNames)
		}
		if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, wantPassThrough) {
			t.Fatalf("interactive argv = %#v, want %#v", plan.LaunchVariants.Interactive.Argv, wantPassThrough)
		}
		if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, wantPassThrough) {
			t.Fatalf("managed client argv = %#v, want pass-through tokens preserved", plan.LaunchVariants.ManagedClient.Argv)
		}
	})

	t.Run("empty_name_contributes_no_requirement", func(t *testing.T) {
		// Codex accepts --remote-auth-token-env= (probe exit 0); an empty name
		// references no environment variable.
		project := t.TempDir()
		plan, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--remote-auth-token-env="}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		if err != nil {
			t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
		}
		if !reflect.DeepEqual(plan.RequiredEnvNames, []string{}) {
			t.Fatalf("RequiredEnvNames = %#v, want empty", plan.RequiredEnvNames)
		}
		if !reflect.DeepEqual(plan.LaunchVariants.ManagedClient.Argv, []string{"--remote-auth-token-env="}) {
			t.Fatalf("managed client argv = %#v", plan.LaunchVariants.ManagedClient.Argv)
		}
	})

	t.Run("repeated_occurrence_fails_closed", func(t *testing.T) {
		// Probe exit 2 on codex-cli 0.145.0: "the argument
		// '--remote-auth-token-env <ENV_VAR>' cannot be used multiple times".
		project := t.TempDir()
		args := []string{"--remote-auth-token-env", "A_TOKEN", "--remote-auth-token-env=B_TOKEN"}
		_, err := BuildPrimarySessionLaunchPlan("codex", project, home, args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		var composeErr *PrimarySessionComposeError
		if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorInvalidProviderArguments {
			t.Fatalf("error = %#v, want code %q", err, PrimarySessionErrorInvalidProviderArguments)
		}
		if !strings.Contains(err.Error(), "cannot be used multiple times") {
			t.Fatalf("error = %v, want multiplicity diagnostic", err)
		}
		if _, launchErr := BuildCodexLaunchPlan(project, home, args); launchErr == nil {
			t.Fatalf("launcher accepted repeated --remote-auth-token-env")
		}
	})

	t.Run("missing_value_fails_closed", func(t *testing.T) {
		// Probe exit 2: "a value is required for '--remote-auth-token-env
		// <ENV_VAR>' but none was supplied".
		project := t.TempDir()
		_, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--remote-auth-token-env"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
		var composeErr *PrimarySessionComposeError
		if !errors.As(err, &composeErr) || composeErr.Code != PrimarySessionErrorInvalidProviderArguments {
			t.Fatalf("error = %#v, want code %q", err, PrimarySessionErrorInvalidProviderArguments)
		}
		if !strings.Contains(err.Error(), "a value is required for the Codex argument --remote-auth-token-env") {
			t.Fatalf("error = %v, want missing-value diagnostic", err)
		}
	})
}
