//go:build !windows

package infra

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func canonicalQwenTargetTOML(assertProvider, assertEndpoint bool) string {
	body := "\n[agents.targets.\"qwen-mlx-8bit\"]\n" +
		"vendor=\"qwen\"\n" +
		"environment=\"pi\"\n" +
		"model=\"Model\"\n" +
		"reasoning=\"medium\"\n" +
		"profile=\"profile\"\n"
	if assertProvider {
		body += "profile_provider=\"local-provider\"\n"
	}
	if assertEndpoint {
		body += "endpoint=\"http://127.0.0.1:18011/v1\"\n"
	}
	return body + "[agents.entrypoints]\nqwen-infra=\"qwen-mlx-8bit\"\n"
}

func canonicalQwenProject(t *testing.T, body string) (project, home, configPath, piPath string) {
	t.Helper()
	project, home = t.TempDir(), t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	piRoot := officialPiAsset(t)
	piPath = filepath.Join(piRoot, "pi")
	configPath = writeCanonicalConfig(t, project, reasoningPiProfileTOML("profile", "/bin/echo", 18011)+body)
	return
}

func TestCanonicalQwenPlanProvesProfileDerivedIdentityAndEndpointInvariants(t *testing.T) {
	project, home, configPath, piPath := canonicalQwenProject(t, canonicalQwenTargetTOML(true, true))
	plan, err := BuildCanonicalTargetLaunchPlan("qwen-infra", project, home,
		[]string{"--model", "local-provider/Model:medium", "--provider=local-provider", "--thinking", "medium", "--endpoint", "http://127.0.0.1:18011/v1", "--api-key=secret"},
		ChildLaunchCompositionProducer{}, func(string) (string, error) { return piPath, nil })
	if err != nil {
		t.Fatalf("BuildCanonicalTargetLaunchPlan: %v", err)
	}
	if plan.Target == nil || plan.Target.Model != "Model" || plan.Target.Profile == nil || *plan.Target.Profile != "profile" {
		t.Fatalf("target = %#v", plan.Target)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "local-provider/Model" || !samePath(plan.Resolved.Model.Source, configPath) {
		t.Fatalf("profile-qualified model = %#v", plan.Resolved.Model)
	}
	if plan.Resolved.ProfileProvider == nil || plan.Resolved.ProfileProvider.Value == nil || *plan.Resolved.ProfileProvider.Value != "local-provider" || !samePath(plan.Resolved.ProfileProvider.Source, configPath) {
		t.Fatalf("profile provider = %#v", plan.Resolved.ProfileProvider)
	}
	if plan.Resolved.Endpoint == nil || plan.Resolved.Endpoint.Value == nil || *plan.Resolved.Endpoint.Value != "http://127.0.0.1:18011/v1" || !samePath(plan.Resolved.Endpoint.Source, configPath) {
		t.Fatalf("profile endpoint = %#v", plan.Resolved.Endpoint)
	}
	if plan.Pi == nil || plan.Pi.Runtime == nil || plan.Pi.Runtime.Endpoint != *plan.Resolved.Endpoint.Value {
		t.Fatalf("runtime endpoint invariant = %#v", plan.Pi)
	}
	if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != "medium" || !samePath(plan.Resolved.Reasoning.Source, configPath) {
		t.Fatalf("profile reasoning = %#v", plan.Resolved.Reasoning)
	}
	wantArgs := []string{"--provider", "local-provider", "--model", "Model", "--thinking", "medium", "--api-key", "<redacted>"}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, wantArgs) {
		t.Fatalf("normalized Pi argv = %#v, want %#v", plan.LaunchVariants.Interactive.Argv, wantArgs)
	}
	resolved, err := ResolveCanonicalTarget("qwen-infra", project, home)
	if err != nil {
		t.Fatalf("ResolveCanonicalTarget: %v", err)
	}
	modelsJSON, err := GeneratePiModelsJSON(*resolved.Profile)
	if err != nil {
		t.Fatalf("GeneratePiModelsJSON: %v", err)
	}
	var models piModelsDocument
	if err := json.Unmarshal(modelsJSON, &models); err != nil {
		t.Fatalf("decode generated models.json: %v", err)
	}
	generated := models.Providers["local-provider"].Models[0]
	if !generated.Reasoning || generated.Compat["thinkingFormat"] != "qwen-chat-template" {
		t.Fatalf("generated Pi model cannot activate Qwen thinking: %#v", generated)
	}
	rendered := RenderCanonicalTargetLaunchPlan(plan)
	for _, want := range []string{"effective_model: local-provider/Model", "effective_profile_provider: local-provider", "effective_endpoint: http://127.0.0.1:18011/v1", `  - "--api-key"`, `  - "<redacted>"`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered plan missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "secret") {
		t.Fatalf("rendered plan leaked API key:\n%s", rendered)
	}
}

func TestCanonicalQwenSelectedProfileStillRequiresPublisher(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	writeProviderIsolationFixture(t, project, `
[agents.targets.qwen]
vendor = "qwen"
environment = "pi"
model = "Model"
reasoning = "off"
profile = "qwen-bsim"

[agents.entrypoints]
qwen-infra = "qwen"
`)

	_, err := BuildCanonicalTargetLaunchPlan("qwen-infra", project, home, nil, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err == nil || !strings.Contains(err.Error(), "agents.pi.profiles.qwen-bsim.publisher") {
		t.Fatalf("error = %v, want canonical Qwen selected publisher field", err)
	}
}

func TestCanonicalQwenAcceptsOperatorProviderWithoutLiteralLocalQwen(t *testing.T) {
	project, home, _, _ := canonicalQwenProject(t, canonicalQwenTargetTOML(false, false))
	resolved, err := ResolveCanonicalTarget("qwen-infra", project, home)
	if err != nil {
		t.Fatalf("operator-defined provider without assertion: %v", err)
	}
	if resolved.EffectiveProvider != "local-provider" || resolved.Target.ProfileProvider != nil {
		t.Fatalf("provider resolution = %#v", resolved)
	}
}

func TestCanonicalQwenProfileAssertionsFailClosed(t *testing.T) {
	baseProfile := reasoningPiProfileTOML("profile", "/bin/echo", 18011)
	baseTarget := canonicalQwenTargetTOML(true, true)
	tests := []struct {
		name      string
		body      string
		wantCode  string
		wantField string
	}{
		{name: "non openai api", body: strings.Replace(baseProfile, `api = "openai-completions"`, `api = "anthropic-messages"`, 1) + baseTarget, wantCode: PrimarySessionErrorInvalidProjectConfiguration, wantField: "agents.pi.profiles.profile.api"},
		{name: "model mismatch", body: baseProfile + strings.Replace(baseTarget, `model="Model"`, `model="Other"`, 1), wantCode: PrimarySessionErrorInvalidTarget, wantField: "agents.targets.qwen-mlx-8bit.model"},
		{name: "reasoning mismatch", body: baseProfile + strings.Replace(baseTarget, `reasoning="medium"`, `reasoning="off"`, 1), wantCode: PrimarySessionErrorInvalidTarget, wantField: "agents.targets.qwen-mlx-8bit.reasoning"},
		{name: "profile cannot reason", body: strings.Replace(baseProfile, `reasoning = true`, `reasoning = false`, 1) + baseTarget, wantCode: PrimarySessionErrorInvalidProjectConfiguration, wantField: "agents.pi.profiles.profile.reasoning"},
		{name: "missing qwen thinking format", body: strings.Replace(baseProfile, "thinking_format = \"qwen-chat-template\"\n", "", 1) + baseTarget, wantCode: PrimarySessionErrorInvalidTarget, wantField: "agents.pi.profiles.profile.compat.thinking_format"},
		{name: "wrong qwen thinking format", body: strings.Replace(baseProfile, `thinking_format = "qwen-chat-template"`, `thinking_format = "qwen"`, 1) + baseTarget, wantCode: PrimarySessionErrorInvalidTarget, wantField: "agents.pi.profiles.profile.compat.thinking_format"},
		{name: "provider assertion mismatch", body: baseProfile + strings.Replace(baseTarget, `profile_provider="local-provider"`, `profile_provider="other"`, 1), wantCode: PrimarySessionErrorInvalidTarget, wantField: "agents.targets.qwen-mlx-8bit.profile_provider"},
		{name: "endpoint assertion mismatch", body: baseProfile + strings.Replace(baseTarget, `endpoint="http://127.0.0.1:18011/v1"`, `endpoint="http://127.0.0.1:18012/v1"`, 1), wantCode: PrimarySessionErrorInvalidTarget, wantField: "agents.targets.qwen-mlx-8bit.endpoint"},
		{name: "unknown profile", body: baseProfile + strings.Replace(baseTarget, `profile="profile"`, `profile="missing"`, 1), wantCode: PrimarySessionErrorInvalidTarget, wantField: "agents.targets.qwen-mlx-8bit.profile"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			project, home := t.TempDir(), t.TempDir()
			writeCanonicalConfig(t, project, testCase.body)
			_, err := ResolveCanonicalTarget("qwen-infra", project, home)
			if canonicalErrorCode(t, err) != testCase.wantCode || !strings.Contains(err.Error(), testCase.wantField) {
				t.Fatalf("error = %v, want %s and %s", err, testCase.wantCode, testCase.wantField)
			}
		})
	}
}

func TestCanonicalQwenCompositeModelAndCoordinatesAreIdentityLocked(t *testing.T) {
	project, home, _, piPath := canonicalQwenProject(t, canonicalQwenTargetTOML(false, false))
	exact := [][]string{
		{"--model", "Model"},
		{"--model", "local-provider/Model"},
		{"--model", "Model:medium"},
		{"--model", "local-provider/Model:medium"},
		{"--profile=profile", "--provider", "local-provider", "--thinking=medium"},
	}
	for _, args := range exact {
		if _, err := BuildCanonicalTargetLaunchPlan("qwen-infra", project, home, args, ChildLaunchCompositionProducer{}, func(string) (string, error) { return piPath, nil }); err != nil {
			t.Fatalf("exact selector %v: %v", args, err)
		}
	}
	conflicts := [][]string{
		{"--model", "Model:high"},
		{"--model", "local-provider/Model:high"},
		{"--model", "other/Model:medium"},
		{"--provider", "other"},
		{"--thinking", "off"},
		{"--profile", "other"},
		{"--endpoint", "http://127.0.0.1:18012/v1"},
	}
	for _, args := range conflicts {
		_, err := BuildCanonicalTargetLaunchPlan("qwen-infra", project, home, args, ChildLaunchCompositionProducer{}, func(string) (string, error) { return piPath, nil })
		if canonicalErrorCode(t, err) != PrimarySessionErrorTargetIdentityConflict {
			t.Fatalf("conflicting selector %v error = %v", args, err)
		}
	}
}

func TestCanonicalQwenDelimiterRemainsMessageOperandBoundary(t *testing.T) {
	project, home, _, piPath := canonicalQwenProject(t, canonicalQwenTargetTOML(false, false))
	_, err := BuildCanonicalTargetLaunchPlan("qwen-infra", project, home,
		[]string{"--", "--model", "other"},
		ChildLaunchCompositionProducer{}, func(string) (string, error) { return piPath, nil })
	if err == nil || !strings.Contains(err.Error(), `unsafe Pi message operand "--model"`) {
		t.Fatalf("Pi delimiter stopped being a message operand boundary: %v", err)
	}
}

func TestCanonicalQwenDeclarationsDoNotChangeDirectPiPrecedence(t *testing.T) {
	project, home, _, piPath := canonicalQwenProject(t, canonicalQwenTargetTOML(false, false))
	plan, err := BuildPrimarySessionLaunchPlan("pi", project, home, []string{"--profile=profile"}, ChildLaunchCompositionProducer{}, func(string) (string, error) { return piPath, nil })
	if err != nil {
		t.Fatalf("direct Pi: %v", err)
	}
	if plan.Resolved.Profile.Value == nil || *plan.Resolved.Profile.Value != "profile" || plan.Resolved.Profile.Source != "cli" || plan.Target != nil || plan.Resolved.ProfileProvider != nil || plan.Resolved.Endpoint != nil {
		t.Fatalf("direct Pi precedence changed: %#v", plan)
	}
}

func TestDoctorReportsCanonicalQwenTargetAndProfileProvenance(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	project := filepath.Join(root, "nested")
	mustMkdir(t, project)
	profileConfig := writeCanonicalConfig(t, root, reasoningPiProfileTOML("profile", "/bin/echo", 18011))
	targetConfig := writeCanonicalConfig(t, project, canonicalQwenTargetTOML(false, false))
	t.Setenv("HOME", home)
	layout, err := LocalLayout("", project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	report, err := Doctor(layout)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if len(report.CanonicalTargets) != 1 {
		t.Fatalf("canonical targets = %#v", report.CanonicalTargets)
	}
	target := report.CanonicalTargets[0]
	if target.Entrypoint != "qwen-infra" || !samePath(target.TargetSource, targetConfig) || !samePath(target.ProfileSource, profileConfig) || target.ProfileProvider != "local-provider" || target.Endpoint != "http://127.0.0.1:18011/v1" {
		t.Fatalf("doctor canonical target = %#v", target)
	}
}
