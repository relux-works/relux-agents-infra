package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

func mainCanonicalOpenAITOML() string {
	return "[agents.targets.openai]\n" +
		"vendor=\"openai\"\n" +
		"environment=\"codex\"\n" +
		"model=\"gpt-5.6-sol\"\n" +
		"reasoning=\"high\"\n" +
		"[agents.entrypoints]\n" +
		"openai-infra=\"openai\"\n"
}

func mainCanonicalHostedTOML() string {
	return "[agents.targets.openai]\n" +
		"vendor=\"openai\"\n" +
		"environment=\"codex\"\n" +
		"model=\"gpt-5.6-sol\"\n" +
		"reasoning=\"high\"\n" +
		"[agents.targets.anthropic]\n" +
		"vendor=\"anthropic\"\n" +
		"environment=\"claude-code\"\n" +
		"model=\"claude-opus-5\"\n" +
		"reasoning=\"high\"\n" +
		"[agents.entrypoints]\n" +
		"openai-infra=\"openai\"\n" +
		"anthropic-infra=\"anthropic\"\n"
}

func writeMainCanonicalConfig(t *testing.T, project, body string) string {
	t.Helper()
	path := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, filepath.Dir(path))
	mustWrite(t, path, body)
	return path
}

func TestRunComposeCanonicalEntrypointEmitsAliasPlan(t *testing.T) {
	home, project, binDir := t.TempDir(), t.TempDir(), t.TempDir()
	configPath := writeMainCanonicalConfig(t, project, mainCanonicalOpenAITOML())
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	output := captureStdout(t, func() {
		if err := runCompose([]string{"--mode", "primary-session", "--entrypoint", "openai-infra", "--project", project, "--schema-version", "1", "--json", "--", "--model", "gpt-5.6-sol"}); err != nil {
			t.Fatalf("runCompose: %v", err)
		}
	})
	var plan infra.PrimarySessionLaunchPlan
	decodeSingleJSONDocument(t, output, &plan)
	canonicalConfigPath, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Provider != "codex" || plan.Target == nil || plan.Target.Entrypoint != "openai-infra" || plan.Target.Source != canonicalConfigPath {
		t.Fatalf("canonical plan = %#v", plan)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "gpt-5.6-sol" || plan.Resolved.Model.Source != canonicalConfigPath {
		t.Fatalf("resolved model = %#v", plan.Resolved.Model)
	}
}

func TestRunComposePrimarySessionRequiresExactlyOneSelector(t *testing.T) {
	project := t.TempDir()
	for name, args := range map[string][]string{
		"neither": {"--mode", "primary-session", "--project", project, "--schema-version", "1", "--json"},
		"both":    {"--mode", "primary-session", "--agent", "codex", "--entrypoint", "openai-infra", "--project", project, "--schema-version", "1", "--json"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := runCompose(args); err == nil || !strings.Contains(err.Error(), "exactly one") {
				t.Fatalf("selector error = %v", err)
			}
		})
	}
	if err := runCompose([]string{"--entrypoint", "openai-infra", "--project", project, "--schema-version", "1", "--json"}); err == nil || !strings.Contains(err.Error(), "child compose") {
		t.Fatalf("child entrypoint error = %v", err)
	}
}

func TestRunComposeCanonicalErrorsCarrySafeActionableContext(t *testing.T) {
	home, binDir := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\nexit 0\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	tests := []struct {
		name        string
		body        string
		providerArg []string
		wantCode    string
		wantField   string
	}{
		{name: "missing mapping", body: "[agents.codex.primary_session]\nmodel=\"legacy\"\n", wantCode: infra.PrimarySessionErrorUnknownEntrypoint, wantField: "agents.entrypoints.openai-infra"},
		{name: "unknown target", body: "[agents.entrypoints]\nopenai-infra=\"missing\"\n", wantCode: infra.PrimarySessionErrorUnknownTarget, wantField: "agents.entrypoints.openai-infra"},
		{name: "malformed field", body: "[agents.targets.openai]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=7\nreasoning=\"high\"\n[agents.entrypoints]\nopenai-infra=\"openai\"\n", wantCode: infra.PrimarySessionErrorInvalidProjectConfiguration, wantField: "agents.targets.openai.model"},
		{name: "identity conflict", body: mainCanonicalOpenAITOML(), providerArg: []string{"--", "--model", "other"}, wantCode: infra.PrimarySessionErrorTargetIdentityConflict, wantField: "provider_args.model"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			project := t.TempDir()
			configPath := writeMainCanonicalConfig(t, project, testCase.body)
			before, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"--mode", "primary-session", "--entrypoint", "openai-infra", "--project", project, "--schema-version", "1", "--json"}
			args = append(args, testCase.providerArg...)
			var composeErr error
			output := captureStdout(t, func() { composeErr = runCompose(args) })
			if composeErr == nil || !strings.Contains(composeErr.Error(), "Remediation:") {
				t.Fatalf("human error = %v", composeErr)
			}
			var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
			decodeSingleJSONDocument(t, output, &envelope)
			if envelope.Error.Code != testCase.wantCode || envelope.Error.Context == nil || envelope.Error.Context.Field != testCase.wantField || envelope.Error.Remediation == "" {
				t.Fatalf("error envelope = %#v", envelope)
			}
			after, err := os.ReadFile(configPath)
			if err != nil || string(after) != string(before) {
				t.Fatalf("compose rewrote project config: err=%v before=%q after=%q", err, before, after)
			}
		})
	}
}

func TestRunComposeCanonicalUnreadableSourceIsNotReportedAsMissingMapping(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	configPath := filepath.Join(project, ".agents", ".configs", "project-config.toml")
	mustMkdir(t, configPath)
	t.Setenv("HOME", home)
	var composeErr error
	output := captureStdout(t, func() {
		composeErr = runCompose([]string{"--mode", "primary-session", "--entrypoint", "openai-infra", "--project", project, "--schema-version", "1", "--json"})
	})
	if composeErr == nil || !strings.Contains(composeErr.Error(), "Remediation:") || strings.Contains(composeErr.Error(), infra.PrimarySessionErrorUnknownEntrypoint) {
		t.Fatalf("unreadable source error = %v", composeErr)
	}
	var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
	decodeSingleJSONDocument(t, output, &envelope)
	if envelope.Error.Code != infra.PrimarySessionErrorInvalidProjectConfiguration || envelope.Error.Context == nil || envelope.Error.Context.Field != "project_config" || envelope.Error.Context.Source == "" || envelope.Error.Remediation == "" {
		t.Fatalf("unreadable source envelope = %#v", envelope)
	}
	info, err := os.Stat(configPath)
	if err != nil || !info.IsDir() {
		t.Fatalf("compose rewrote unreadable source: %v %#v", err, info)
	}
}

func TestRunTargetDispatchPreservesCallerCWDAndLocksBeforeProviderSideEffects(t *testing.T) {
	home, project, binDir := t.TempDir(), t.TempDir(), t.TempDir()
	record := filepath.Join(t.TempDir(), "provider-record")
	writeMainCanonicalConfig(t, project, mainCanonicalOpenAITOML())
	mustMkdir(t, filepath.Join(project, ".agents", ".instructions"))
	mustMkdir(t, filepath.Join(project, ".agents", ".rules"))
	mustMkdir(t, filepath.Join(project, ".agents", "skills"))
	mustWrite(t, filepath.Join(project, ".agents", ".instructions", "AGENTS.md"), "# Test instructions\n")
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\npwd > \""+record+"\"\nprintf '%s\\n' \"$@\" >> \""+record+"\"\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	t.Setenv(callerCWDEnv, project)

	stderr := captureStderr(t, func() {
		if err := runTarget([]string{"openai-infra", "--", "--model", "gpt-5.6-sol", "exec", "inspect"}); err != nil {
			t.Fatalf("runTarget: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("ordinary target launch printed diagnostics to stderr:\n%s", stderr)
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	canonicalProject, _ := filepath.EvalSymlinks(project)
	if lines[0] != canonicalProject || !strings.Contains(string(data), "gpt-5.6-sol") || !strings.Contains(string(data), "inspect") {
		t.Fatalf("provider record = %q", data)
	}

	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	if err := runTarget([]string{"openai-infra", "--", "--model", "other"}); err == nil {
		t.Fatal("identity conflict launched")
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("identity conflict reached provider: %v", err)
	}

	legacyConfig := writeMainCanonicalConfig(t, project, "[agents.codex.primary_session]\nmodel=\"gpt-5.6-sol\"\nreasoning_effort=\"high\"\n")
	legacyBefore, err := os.ReadFile(legacyConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := runTarget([]string{"openai-infra", "--", "exec", "inspect"}); err == nil || !strings.Contains(err.Error(), infra.PrimarySessionErrorUnknownEntrypoint) {
		t.Fatalf("unconfigured alias error = %v", err)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("legacy-only configuration bypassed target resolution: %v", err)
	}
	legacyAfter, err := os.ReadFile(legacyConfig)
	if err != nil || string(legacyAfter) != string(legacyBefore) {
		t.Fatalf("unconfigured alias rewrote legacy config: err=%v before=%q after=%q", err, legacyBefore, legacyAfter)
	}
}

// Production call site: runTarget -> BuildCanonicalTargetLaunchPlan ->
// lockCanonicalTargetArguments. The first delimiter in provider args belongs
// to the hosted wrapper, so identity selectors after it remain active and must
// be locked before preparation or provider execution.
func TestRunTargetRejectsHostedIdentitySelectorsAfterWrapperDelimiter(t *testing.T) {
	home, project, binDir := t.TempDir(), t.TempDir(), t.TempDir()
	recordDir := t.TempDir()
	configPath := writeMainCanonicalConfig(t, project, mainCanonicalHostedTOML())
	for _, path := range []string{
		filepath.Join(project, ".agents", ".instructions"),
		filepath.Join(project, ".agents", ".rules"),
		filepath.Join(project, ".agents", "skills"),
	} {
		mustMkdir(t, path)
	}
	mustWrite(t, filepath.Join(project, ".agents", ".instructions", "AGENTS.md"), "# Test instructions\n")
	for _, provider := range []string{"codex", "claude"} {
		record := filepath.Join(recordDir, provider)
		mustWrite(t, filepath.Join(binDir, provider), "#!/bin/sh\nprintf '%s\\n' \"$@\" > \""+record+"\"\n")
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	t.Setenv(callerCWDEnv, project)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		entrypoint string
		provider   string
		args       []string
		wantField  string
	}{
		{name: "Codex model", entrypoint: "openai-infra", provider: "codex", args: []string{"exec", "--", "--model", "other"}, wantField: "provider_args.model"},
		{name: "Codex reasoning", entrypoint: "openai-infra", provider: "codex", args: []string{"exec", "--", "--model-reasoning-effort", "low"}, wantField: "provider_args.reasoning"},
		{name: "Codex config model", entrypoint: "openai-infra", provider: "codex", args: []string{"exec", "--", "-c", `model="other"`}, wantField: "provider_args.model"},
		{name: "Codex config reasoning", entrypoint: "openai-infra", provider: "codex", args: []string{"exec", "--", "-c", `model_reasoning_effort="low"`}, wantField: "provider_args.reasoning"},
		{name: "Codex profile", entrypoint: "openai-infra", provider: "codex", args: []string{"exec", "--", "--profile", "work"}, wantField: "provider_args.profile"},
		{name: "Codex config profile", entrypoint: "openai-infra", provider: "codex", args: []string{"exec", "--", "-c", `profile="work"`}, wantField: "provider_args.profile"},
		{name: "Claude model", entrypoint: "anthropic-infra", provider: "claude", args: []string{"--", "--model", "other"}, wantField: "provider_args.model"},
		{name: "Claude reasoning", entrypoint: "anthropic-infra", provider: "claude", args: []string{"--", "--effort", "low"}, wantField: "provider_args.reasoning"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			record := filepath.Join(recordDir, testCase.provider)
			_ = os.Remove(record)
			args := append([]string{testCase.entrypoint, "--"}, testCase.args...)
			err := runTarget(args)
			if err == nil || !strings.Contains(err.Error(), infra.PrimarySessionErrorTargetIdentityConflict) || !strings.Contains(err.Error(), testCase.wantField) {
				t.Fatalf("identity conflict = %v, want code and field %s", err, testCase.wantField)
			}
			if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
				t.Fatalf("identity conflict reached %s provider: %v", testCase.provider, statErr)
			}
			after, readErr := os.ReadFile(configPath)
			if readErr != nil || string(after) != string(before) {
				t.Fatalf("identity conflict rewrote config: err=%v before=%q after=%q", readErr, before, after)
			}
		})
	}
}
