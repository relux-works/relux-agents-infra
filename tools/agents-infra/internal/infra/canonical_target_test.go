package infra

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func canonicalHostedTargetsTOML() string {
	return "[agents.targets.\"openai-sol-high\"]\n" +
		"vendor = \"openai\"\n" +
		"environment = \"codex\"\n" +
		"model = \"gpt-5.6-sol\"\n" +
		"reasoning = \"high\"\n\n" +
		"[agents.targets.\"anthropic-opus-high\"]\n" +
		"vendor = \"anthropic\"\n" +
		"environment = \"claude-code\"\n" +
		"model = \"claude-opus-5\"\n" +
		"reasoning = \"high\"\n\n" +
		"[agents.entrypoints]\n" +
		"openai-infra = \"openai-sol-high\"\n" +
		"anthropic-infra = \"anthropic-opus-high\"\n"
}

func writeCanonicalConfig(t *testing.T, project, body string) string {
	t.Helper()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(path))
	mustWrite(t, path, body)
	return path
}

func canonicalErrorCode(t *testing.T, err error) string {
	t.Helper()
	var targetErr *CanonicalTargetError
	if !errors.As(err, &targetErr) {
		t.Fatalf("error = %#v, want CanonicalTargetError", err)
	}
	return targetErr.Code
}

func TestCanonicalTargetsComposeAtomicallyRootToCWD(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)
	parentPath := writeCanonicalConfig(t, parent, "[agents.targets.shared]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"parent-model\"\nreasoning=\"parent-reasoning\"\n[agents.entrypoints]\nopenai-infra=\"shared\"\n")
	childPath := writeCanonicalConfig(t, child, "[agents.targets.shared]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"child-model\"\nreasoning=\"child-reasoning\"\n")

	resolved, err := ResolveCanonicalTarget("openai-infra", child, home)
	if err != nil {
		t.Fatalf("ResolveCanonicalTarget: %v", err)
	}
	if resolved.Target.Model != "child-model" || resolved.Target.Reasoning != "child-reasoning" || resolved.Target.Source != childPath {
		t.Fatalf("atomic target = %#v", resolved.Target)
	}
	if resolved.Entrypoint.Source != parentPath {
		t.Fatalf("entrypoint source = %q, want %q", resolved.Entrypoint.Source, parentPath)
	}

	mustWrite(t, childPath, "[agents.targets.shared]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"child-model\"\n")
	_, err = ResolveCanonicalTarget("openai-infra", child, home)
	if err == nil || !strings.Contains(err.Error(), "agents.targets.shared.reasoning") {
		t.Fatalf("nearer partial target merged with ancestor: %v", err)
	}
}

func TestCanonicalTargetParserRejectsEveryCrossVendorEnvironmentPair(t *testing.T) {
	admitted := map[string]bool{"openai/codex": true, "anthropic/claude-code": true, "qwen/pi": true}
	for _, vendor := range []string{"openai", "anthropic", "qwen"} {
		for _, environment := range []string{"codex", "claude-code", "pi"} {
			pair := vendor + "/" + environment
			if admitted[pair] {
				continue
			}
			t.Run(strings.ReplaceAll(pair, "/", "_"), func(t *testing.T) {
				body := "[agents.targets.bad]\nvendor=\"" + vendor + "\"\nenvironment=\"" + environment + "\"\nmodel=\"Model\"\nreasoning=\"high\"\nprofile=\"profile\"\n"
				_, err := parseProjectConfig([]byte(body), "/project/.agents/.configs/project-config.toml")
				if err == nil || !strings.Contains(err.Error(), "agents.targets.bad.environment") || !strings.Contains(err.Error(), "not admitted") {
					t.Fatalf("cross-pair %s error = %v", pair, err)
				}
			})
		}
	}
}

func TestCanonicalTargetParserFieldAndReasoningDomainsFailClosed(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{name: "targets wrong type", body: "agents.targets = \"bad\"", wantField: "agents.targets"},
		{name: "target wrong type", body: "[agents.targets]\nbad = \"bad\"", wantField: "agents.targets.bad"},
		{name: "missing vendor", body: "[agents.targets.bad]\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.vendor"},
		{name: "missing environment", body: "[agents.targets.bad]\nvendor=\"openai\"\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.environment"},
		{name: "missing model", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nreasoning=\"high\"", wantField: "agents.targets.bad.model"},
		{name: "missing reasoning", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"m\"", wantField: "agents.targets.bad.reasoning"},
		{name: "empty vendor", body: "[agents.targets.bad]\nvendor=\"\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.vendor"},
		{name: "empty environment", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"\"\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.environment"},
		{name: "empty model", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"\"\nreasoning=\"high\"", wantField: "agents.targets.bad.model"},
		{name: "wrong vendor type", body: "[agents.targets.bad]\nvendor=7\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.vendor"},
		{name: "wrong environment type", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=7\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.environment"},
		{name: "wrong model type", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=7\nreasoning=\"high\"", wantField: "agents.targets.bad.model"},
		{name: "wrong reasoning type", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=7", wantField: "agents.targets.bad.reasoning"},
		{name: "wrong profile type", body: "[agents.targets.bad]\nvendor=\"qwen\"\nenvironment=\"pi\"\nmodel=\"m\"\nreasoning=\"off\"\nprofile=7", wantField: "agents.targets.bad.profile"},
		{name: "wrong provider assertion type", body: "[agents.targets.bad]\nvendor=\"qwen\"\nenvironment=\"pi\"\nmodel=\"m\"\nreasoning=\"off\"\nprofile=\"p\"\nprofile_provider=7", wantField: "agents.targets.bad.profile_provider"},
		{name: "wrong endpoint type", body: "[agents.targets.bad]\nvendor=\"qwen\"\nenvironment=\"pi\"\nmodel=\"m\"\nreasoning=\"off\"\nprofile=\"p\"\nendpoint=7", wantField: "agents.targets.bad.endpoint"},
		{name: "whitespace model", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"  \"\nreasoning=\"high\"", wantField: "agents.targets.bad.model"},
		{name: "empty codex reasoning", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"\"", wantField: "agents.targets.bad.reasoning"},
		{name: "unknown vendor", body: "[agents.targets.bad]\nvendor=\"other\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.vendor"},
		{name: "unknown environment", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"other\"\nmodel=\"m\"\nreasoning=\"high\"", wantField: "agents.targets.bad.environment"},
		{name: "unknown field", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"high\"\nextra=true", wantField: "agents.targets.bad.extra"},
		{name: "claude reasoning", body: "[agents.targets.bad]\nvendor=\"anthropic\"\nenvironment=\"claude-code\"\nmodel=\"m\"\nreasoning=\"banana\"", wantField: "agents.targets.bad.reasoning"},
		{name: "pi reasoning", body: "[agents.targets.bad]\nvendor=\"qwen\"\nenvironment=\"pi\"\nmodel=\"m\"\nreasoning=\"banana\"\nprofile=\"p\"", wantField: "agents.targets.bad.reasoning"},
		{name: "hosted profile", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"future-effort\"\nprofile=\"p\"", wantField: "agents.targets.bad.profile"},
		{name: "hosted profile provider", body: "[agents.targets.bad]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"future-effort\"\nprofile_provider=\"p\"", wantField: "agents.targets.bad.profile_provider"},
		{name: "hosted endpoint", body: "[agents.targets.bad]\nvendor=\"anthropic\"\nenvironment=\"claude-code\"\nmodel=\"m\"\nreasoning=\"high\"\nendpoint=\"https://example.test/v1\"", wantField: "agents.targets.bad.endpoint"},
		{name: "pi missing profile", body: "[agents.targets.bad]\nvendor=\"qwen\"\nenvironment=\"pi\"\nmodel=\"m\"\nreasoning=\"off\"", wantField: "agents.targets.bad.profile"},
		{name: "pi relative endpoint", body: "[agents.targets.bad]\nvendor=\"qwen\"\nenvironment=\"pi\"\nmodel=\"m\"\nreasoning=\"off\"\nprofile=\"p\"\nendpoint=\"/v1\"", wantField: "agents.targets.bad.endpoint"},
		{name: "entrypoints wrong type", body: "agents.entrypoints = []", wantField: "agents.entrypoints"},
		{name: "unknown entrypoint key", body: "[agents.entrypoints]\nother=\"target\"", wantField: "agents.entrypoints.other"},
		{name: "entrypoint wrong value type", body: "[agents.entrypoints]\nopenai-infra=7", wantField: "agents.entrypoints.openai-infra"},
		{name: "entrypoint empty", body: "[agents.entrypoints]\nopenai-infra=\"\"", wantField: "agents.entrypoints.openai-infra"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseProjectConfig([]byte(testCase.body), "/project/.agents/.configs/project-config.toml")
			if err == nil || !strings.Contains(err.Error(), testCase.wantField) {
				t.Fatalf("error = %v, want field %s", err, testCase.wantField)
			}
		})
	}

	if _, err := parseProjectConfig([]byte("[agents.targets.ok]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"m\"\nreasoning=\"future-provider-token\""), "/project/config.toml"); err != nil {
		t.Fatalf("non-empty future Codex reasoning was locally enumerated: %v", err)
	}
}

func TestCanonicalEntrypointResolutionNeverInfersOrFallsBack(t *testing.T) {
	home := t.TempDir()
	t.Run("missing mapping with multiple candidates", func(t *testing.T) {
		project := t.TempDir()
		writeCanonicalConfig(t, project, "[agents.targets.one]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"one\"\nreasoning=\"high\"\n[agents.targets.two]\nvendor=\"openai\"\nenvironment=\"codex\"\nmodel=\"two\"\nreasoning=\"high\"\n[agents.codex.primary_session]\nmodel=\"legacy\"\nreasoning_effort=\"high\"\n")
		_, err := ResolveCanonicalTarget("openai-infra", project, home)
		if canonicalErrorCode(t, err) != PrimarySessionErrorUnknownEntrypoint {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("unknown target", func(t *testing.T) {
		project := t.TempDir()
		writeCanonicalConfig(t, project, "[agents.entrypoints]\nopenai-infra=\"missing\"")
		_, err := ResolveCanonicalTarget("openai-infra", project, home)
		if canonicalErrorCode(t, err) != PrimarySessionErrorUnknownTarget || !strings.Contains(err.Error(), "Remediation:") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("alias vendor mismatch", func(t *testing.T) {
		project := t.TempDir()
		writeCanonicalConfig(t, project, "[agents.targets.wrong]\nvendor=\"anthropic\"\nenvironment=\"claude-code\"\nmodel=\"opus\"\nreasoning=\"high\"\n[agents.entrypoints]\nopenai-infra=\"wrong\"")
		_, err := ResolveCanonicalTarget("openai-infra", project, home)
		if canonicalErrorCode(t, err) != PrimarySessionErrorInvalidTarget {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestCanonicalHostedPlanLocksIdentityAndKeepsLegacyPolicyUnrelated(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configPath := writeCanonicalConfig(t, project, canonicalHostedTargetsTOML()+"\n[agents.codex.primary_session]\nmodel=\"legacy-model\"\nreasoning_effort=\"legacy-reasoning\"\nyolo_mode=true\n")
	plan, err := BuildCanonicalTargetLaunchPlan("openai-infra", project, home,
		[]string{"--model", "gpt-5.6-sol", "--model-reasoning-effort=high", "-c", `model="gpt-5.6-sol"`, "-c", `model_reasoning_effort="high"`, "exec", "inspect"},
		ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildCanonicalTargetLaunchPlan: %v", err)
	}
	wantPrefix := []string{codexDangerouslyBypassApprovalsAndSandbox, "--model", "gpt-5.6-sol", "-c", `model_reasoning_effort="high"`}
	if !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("canonical argv prefix = %#v, want %#v", plan.LaunchVariants.Interactive.Argv, wantPrefix)
	}
	canonicalConfigPath, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Target == nil || plan.Target.Source != canonicalConfigPath || plan.Provider != "codex" {
		t.Fatalf("target plan = %#v", plan)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "gpt-5.6-sol" || plan.Resolved.Model.Source != canonicalConfigPath {
		t.Fatalf("resolved model = %#v", plan.Resolved.Model)
	}
	if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != "high" || plan.Resolved.Reasoning.Source != canonicalConfigPath {
		t.Fatalf("resolved reasoning = %#v", plan.Resolved.Reasoning)
	}
	if !plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != canonicalConfigPath {
		t.Fatalf("unrelated legacy yolo policy was lost: %#v", plan.Resolved.Yolo)
	}
	if plan.Resolved.Profile.Source != "native" || plan.Resolved.Profile.Value != nil {
		t.Fatalf("canonical Codex profile = %#v", plan.Resolved.Profile)
	}
	if plan.Resolved.ProfileProvider == nil || plan.Resolved.ProfileProvider.Source != "not_applicable" || plan.Resolved.Endpoint == nil || plan.Resolved.Endpoint.Source != "not_applicable" {
		t.Fatalf("hosted target effective coordinates = %#v %#v", plan.Resolved.ProfileProvider, plan.Resolved.Endpoint)
	}
}

func TestCanonicalCodexSelectorsAcceptExactAndRefuseEveryDivergentForm(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writeCanonicalConfig(t, project, canonicalHostedTargetsTOML())
	if _, err := BuildCanonicalTargetLaunchPlan("openai-infra", project, home,
		[]string{"exec", "--", "--model", "gpt-5.6-sol", "--model-reasoning-effort", "high"},
		ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t)); err != nil {
		t.Fatalf("exact Codex repeats after wrapper delimiter: %v", err)
	}
	if _, err := BuildCanonicalTargetLaunchPlan("openai-infra", project, home,
		[]string{"exec", "--", "--", "--model", "operand-not-a-selector"},
		ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t)); err != nil {
		t.Fatalf("Codex provider operand boundary was over-locked: %v", err)
	}
	tests := []struct {
		name string
		args []string
	}{
		{name: "model flag", args: []string{"--model", "other"}},
		{name: "reasoning flag", args: []string{"--model-reasoning-effort", "low"}},
		{name: "config model", args: []string{"-c", `model="other"`}},
		{name: "config reasoning", args: []string{"-c", `model_reasoning_effort="low"`}},
		{name: "profile flag", args: []string{"--profile", "work"}},
		{name: "profile config", args: []string{"-c", `profile="work"`}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := BuildCanonicalTargetLaunchPlan("openai-infra", project, home, testCase.args, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
			if canonicalErrorCode(t, err) != PrimarySessionErrorTargetIdentityConflict {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestCanonicalClaudeSelectorsLockStrictReasoningWhileLegacyFallbackRemains(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	writeCanonicalConfig(t, project, canonicalHostedTargetsTOML())
	if _, err := BuildCanonicalTargetLaunchPlan("anthropic-infra", project, home, []string{"--model=claude-opus-5", "--effort", "high"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t)); err != nil {
		t.Fatalf("exact Claude repeats: %v", err)
	}
	if _, err := BuildCanonicalTargetLaunchPlan("anthropic-infra", project, home, []string{"--", "--model", "claude-opus-5", "--effort", "high"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t)); err != nil {
		t.Fatalf("exact Claude repeats after wrapper delimiter: %v", err)
	}
	if _, err := BuildCanonicalTargetLaunchPlan("anthropic-infra", project, home, []string{"--", "--", "--model", "operand-not-a-selector"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t)); err != nil {
		t.Fatalf("Claude provider operand boundary was over-locked: %v", err)
	}
	_, err := BuildCanonicalTargetLaunchPlan("anthropic-infra", project, home, []string{"--effort", "banana"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if canonicalErrorCode(t, err) != PrimarySessionErrorTargetIdentityConflict {
		t.Fatalf("strict alias effort error = %v", err)
	}
	legacy, err := BuildPrimarySessionLaunchPlan("claude", project, home, []string{"--effort", "banana"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil || legacy.Resolved.Reasoning.Source != "native" || legacy.Resolved.Reasoning.Value != nil {
		t.Fatalf("legacy Claude fallback changed: plan=%#v err=%v", legacy.Resolved.Reasoning, err)
	}
}

func TestCanonicalUnreadableConfigHasSourceFieldRemediationAndNoRewrite(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	configPath := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, configPath)
	_, err := ResolveCanonicalTarget("openai-infra", project, home)
	if canonicalErrorCode(t, err) != PrimarySessionErrorInvalidProjectConfiguration {
		t.Fatalf("error = %v", err)
	}
	var targetErr *CanonicalTargetError
	errors.As(err, &targetErr)
	if targetErr.Context.Source != configPath || targetErr.Context.Field != projectConfigParseField || targetErr.Remediation == "" {
		t.Fatalf("actionable context = %#v", targetErr)
	}
	info, statErr := os.Stat(configPath)
	if statErr != nil || !info.IsDir() {
		t.Fatalf("read failure rewrote config path: %v %#v", statErr, info)
	}
}

func TestLegacyPrimarySessionPlanOmitsAliasOnlyJSONFields(t *testing.T) {
	plan, err := BuildPrimarySessionLaunchPlan("codex", t.TempDir(), t.TempDir(), nil, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Target != nil || plan.Resolved.ProfileProvider != nil || plan.Resolved.Endpoint != nil {
		t.Fatalf("legacy plan gained alias-only fields: %#v", plan)
	}
}

func TestCanonicalDeclarationsDoNotChangeDirectCodexOrClaudePrecedence(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	writeCanonicalConfig(t, project, canonicalHostedTargetsTOML()+`
[agents.codex.primary_session]
model = "legacy-codex"
reasoning_effort = "legacy-reasoning"
[agents.claude.primary_session]
model = "legacy-claude"
`)

	codex, err := BuildPrimarySessionLaunchPlan("codex", project, home, []string{"--model", "explicit-codex", "--model-reasoning-effort", "explicit-reasoning"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("direct Codex: %v", err)
	}
	if codex.Resolved.Model.Value == nil || *codex.Resolved.Model.Value != "explicit-codex" || codex.Resolved.Model.Source != "cli:--model" || codex.Target != nil || codex.Resolved.ProfileProvider != nil || codex.Resolved.Endpoint != nil {
		t.Fatalf("direct Codex precedence changed: %#v", codex)
	}

	claude, err := BuildPrimarySessionLaunchPlan("claude", project, home, []string{"--model", "explicit-claude", "--effort", "xhigh"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("direct Claude: %v", err)
	}
	if claude.Resolved.Model.Value == nil || *claude.Resolved.Model.Value != "explicit-claude" || claude.Resolved.Model.Source != "cli:--model" || claude.Resolved.Reasoning.Value == nil || *claude.Resolved.Reasoning.Value != "xhigh" || claude.Target != nil || claude.Resolved.ProfileProvider != nil || claude.Resolved.Endpoint != nil {
		t.Fatalf("direct Claude precedence changed: %#v", claude)
	}
}
