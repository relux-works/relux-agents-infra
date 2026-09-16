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

// Deprecated canonical aliases return the migration error before any parsing,
// resolution, or provider side effect. Identity locking for these entrypoints
// now lives only in the pure builder and the compose contract (see
// TestRunComposeCanonicalErrorsCarrySafeActionableContext); runTarget itself
// must not reach resolution at all.
func TestRunTargetDeprecatedCanonicalAliasesExitBeforeResolution(t *testing.T) {
	home, project, binDir := t.TempDir(), t.TempDir(), t.TempDir()
	record := filepath.Join(t.TempDir(), "provider-record")
	configPath := writeMainCanonicalConfig(t, project, mainCanonicalHostedTOML())
	mustWrite(t, filepath.Join(binDir, "codex"), "#!/bin/sh\npwd > \""+record+"\"\nprintf '%s\\n' \"$@\" >> \""+record+"\"\n")
	mustWrite(t, filepath.Join(binDir, "claude"), "#!/bin/sh\npwd > \""+record+"\"\nprintf '%s\\n' \"$@\" >> \""+record+"\"\n")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	t.Setenv(callerCWDEnv, project)

	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		entrypoint string
		args       []string
		want       string
	}{
		{entrypoint: "openai-infra", args: []string{"--", "--model", "gpt-5.6-sol", "exec", "inspect"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{entrypoint: "anthropic-infra", args: []string{"--", "--model", "other"}, want: "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
		{entrypoint: "openai-infra", args: []string{"--print-config"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{entrypoint: "openai-infra", args: []string{"spawn", "--prompt", "x"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
	} {
		t.Run(testCase.entrypoint+"/"+strings.Join(testCase.args, "_"), func(t *testing.T) {
			_ = os.Remove(record)
			args := append([]string{testCase.entrypoint}, testCase.args...)
			err := runTarget(args)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("runTarget(%q) = %v, want exactly %q", args, err, testCase.want)
			}
			if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
				t.Fatalf("deprecated alias reached provider side effect: %v", statErr)
			}
		})
	}
	after, err := os.ReadFile(configPath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("deprecated alias rewrote project config: err=%v before=%q after=%q", err, before, after)
	}
}

const revision5DirectProviderYoloCallSiteMarker = "--agents-infra-direct-provider-yolo-call-site"

// The target-yolo dispatcher is deprecated along with the canonical aliases.
// It reports the installed dange surface name (the wrappers reach it with the
// canonical entrypoint name) before any parsing or provider side effects.
func TestRunDirectProviderYoloTargetIsDeprecatedBeforeParsing(t *testing.T) {
	home, project, binDir := t.TempDir(), t.TempDir(), t.TempDir()
	record := filepath.Join(t.TempDir(), "provider-record")
	writeMainCanonicalConfig(t, project, mainCanonicalHostedTOML())
	for _, provider := range []string{"codex", "claude"} {
		mustWrite(t, filepath.Join(binDir, provider), "#!/bin/sh\necho launched > \""+record+"\"\n")
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)
	t.Setenv(callerCWDEnv, project)

	for _, testCase := range []struct {
		entrypoint string
		args       []string
		want       string
	}{
		{entrypoint: "openai-infra", args: []string{"-d", "--danger", "--yolo", "--print-config", "inspect token"}, want: "openai-dange is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{entrypoint: "anthropic-infra", args: []string{"-d", "--danger", "--yolo", "--print-config", "inspect token"}, want: "anthropic-dange is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
		{entrypoint: "openai-infra", args: nil, want: "openai-dange is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
	} {
		t.Run(testCase.entrypoint, func(t *testing.T) {
			_ = os.Remove(record)
			args := append([]string{testCase.entrypoint}, testCase.args...)
			err := runDirectProviderYoloTarget(args)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("runDirectProviderYoloTarget(%q) = %v, want exactly %q", args, err, testCase.want)
			}
			if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
				t.Fatalf("deprecated target-yolo reached provider side effect: %v", statErr)
			}
		})
	}
}

func TestRunDirectProviderYoloTargetRejectsMissingAndUnsupportedEntrypoints(t *testing.T) {
	for _, args := range [][]string{nil, {"qwen-infra"}, {"forged-infra"}} {
		if err := runDirectProviderYoloTarget(args); err == nil {
			t.Fatalf("runDirectProviderYoloTarget(%q) unexpectedly succeeded", args)
		}
	}
}

// Deprecated canonical aliases return the migration error for every argument
// shape — danger flags, forged markers, identity selectors — because the guard
// runs before FlagSet parsing and canonical resolution.
func TestRunTargetDeprecatedAliasesIgnoreArgumentShape(t *testing.T) {
	home, project, binDir := t.TempDir(), t.TempDir(), t.TempDir()
	recordDir := t.TempDir()
	configPath := writeMainCanonicalConfig(t, project, mainCanonicalHostedTOML())
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
		want       string
	}{
		{name: "Codex leading danger", entrypoint: "openai-infra", provider: "codex", args: []string{"-d", "--model", "caller-model"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Codex danger", entrypoint: "openai-infra", provider: "codex", args: []string{"--danger", "exec", "inspect"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Codex forged marker", entrypoint: "openai-infra", provider: "codex", args: []string{revision5DirectProviderYoloCallSiteMarker, "--model", "caller-model"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Codex forged marker assignment", entrypoint: "openai-infra", provider: "codex", args: []string{revision5DirectProviderYoloCallSiteMarker + "=forged", "--model", "caller-model"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Codex identity selector", entrypoint: "openai-infra", provider: "codex", args: []string{"--", "exec", "--", "--model", "other"}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Codex config identity", entrypoint: "openai-infra", provider: "codex", args: []string{"--", "exec", "--", "-c", `model="other"`}, want: "openai-infra is deprecated and no longer launches Codex. Use 'curator run codex_cli -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Claude leading danger", entrypoint: "anthropic-infra", provider: "claude", args: []string{"-d", "inspect"}, want: "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Claude forged marker", entrypoint: "anthropic-infra", provider: "claude", args: []string{revision5DirectProviderYoloCallSiteMarker, "--model", "caller-model"}, want: "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Claude identity selector", entrypoint: "anthropic-infra", provider: "claude", args: []string{"--", "--model", "other"}, want: "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
		{name: "Claude reasoning selector", entrypoint: "anthropic-infra", provider: "claude", args: []string{"--", "--effort", "low"}, want: "anthropic-infra is deprecated and no longer launches Claude Code. Use 'curator run claude_code -- <args>'. This entrypoint will be removed in the next release."},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			record := filepath.Join(recordDir, testCase.provider)
			_ = os.Remove(record)
			args := append([]string{testCase.entrypoint}, testCase.args...)
			err := runTarget(args)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("runTarget(%q) = %v, want exactly %q", args, err, testCase.want)
			}
			if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
				t.Fatalf("deprecated alias reached %s provider: %v", testCase.provider, statErr)
			}
			after, readErr := os.ReadFile(configPath)
			if readErr != nil || string(after) != string(before) {
				t.Fatalf("deprecated alias rewrote config: err=%v before=%q after=%q", readErr, before, after)
			}
		})
	}
}
