//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

// Production call site: runCompose -> BuildCanonicalTargetLaunchPlan ->
// buildPiPrimarySessionLaunchPlan. Keeping the profile and target in separate
// ancestor configs proves that effective provider/endpoint provenance cannot
// be narrowed to the optional target assertions.
func TestRunComposeCanonicalQwenUsesProfileDerivedEffectiveCoordinates(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	project := filepath.Join(root, "nested")
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	mustMkdir(t, project)
	profileBody := mainTestPiConfig("/bin/echo", 18011)
	profileBody = strings.Replace(profileBody, `reasoning = false`, `reasoning = true`, 1)
	profileBody = strings.Replace(profileBody, `thinking = "off"`, `thinking = "medium"`, 1)
	profileBody = strings.Replace(profileBody, `supports_developer_role = false`, "supports_developer_role = false\nsupports_reasoning_effort = false\nthinking_format = \"qwen-chat-template\"", 1)
	profileConfig := writeMainCanonicalConfig(t, root, profileBody)
	targetConfig := writeMainCanonicalConfig(t, project, `[agents.targets.qwen]
vendor = "qwen"
environment = "pi"
model = "Model"
reasoning = "medium"
profile = "profile"
profile_provider = "local-provider"
endpoint = "http://127.0.0.1:18011/v1"

[agents.entrypoints]
qwen-infra = "qwen"
`)
	profileBefore, err := os.ReadFile(profileConfig)
	if err != nil {
		t.Fatal(err)
	}
	targetBefore, err := os.ReadFile(targetConfig)
	if err != nil {
		t.Fatal(err)
	}
	canonicalProfileConfig, err := filepath.EvalSymlinks(profileConfig)
	if err != nil {
		t.Fatal(err)
	}
	canonicalTargetConfig, err := filepath.EvalSymlinks(targetConfig)
	if err != nil {
		t.Fatal(err)
	}
	piRoot := mainTestOfficialPiAsset(t)
	t.Setenv("HOME", home)
	t.Setenv("PATH", piRoot)

	output := captureStdout(t, func() {
		if err := runCompose([]string{"--mode", "primary-session", "--entrypoint", "qwen-infra", "--project", project, "--schema-version", "1", "--json", "--", "--model", "local-provider/Model:medium"}); err != nil {
			t.Fatalf("runCompose: %v", err)
		}
	})
	var plan infra.PrimarySessionLaunchPlan
	decodeSingleJSONDocument(t, output, &plan)
	if plan.Target == nil || plan.Target.Model != "Model" || plan.Target.Reasoning != "medium" || plan.Target.Source != canonicalTargetConfig {
		t.Fatalf("target = %#v", plan.Target)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "local-provider/Model" || plan.Resolved.Model.Source != canonicalProfileConfig {
		t.Fatalf("qualified model = %#v", plan.Resolved.Model)
	}
	if plan.Resolved.Reasoning.Value == nil || *plan.Resolved.Reasoning.Value != "medium" || plan.Resolved.Reasoning.Source != canonicalProfileConfig {
		t.Fatalf("reasoning = %#v, want medium with profile source", plan.Resolved.Reasoning)
	}
	wantArgvPrefix := []string{"--provider", "local-provider", "--model", "Model", "--thinking", "medium"}
	if len(plan.LaunchVariants.Interactive.Argv) < len(wantArgvPrefix) || !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv[:len(wantArgvPrefix)], wantArgvPrefix) {
		t.Fatalf("Pi native reasoning argv = %#v, want prefix %#v", plan.LaunchVariants.Interactive.Argv, wantArgvPrefix)
	}
	if plan.Resolved.ProfileProvider == nil || plan.Resolved.ProfileProvider.Value == nil || *plan.Resolved.ProfileProvider.Value != "local-provider" || plan.Resolved.ProfileProvider.Source != canonicalProfileConfig {
		t.Fatalf("profile provider = %#v", plan.Resolved.ProfileProvider)
	}
	if plan.Resolved.Endpoint == nil || plan.Resolved.Endpoint.Value == nil || *plan.Resolved.Endpoint.Value != "http://127.0.0.1:18011/v1" || plan.Resolved.Endpoint.Source != canonicalProfileConfig {
		t.Fatalf("endpoint = %#v", plan.Resolved.Endpoint)
	}
	if plan.Pi == nil || plan.Pi.Runtime == nil || plan.Pi.Runtime.Endpoint != *plan.Resolved.Endpoint.Value {
		t.Fatalf("runtime endpoint invariant = %#v", plan.Pi)
	}
	for path, before := range map[string][]byte{profileConfig: profileBefore, targetConfig: targetBefore} {
		after, err := os.ReadFile(path)
		if err != nil || string(after) != string(before) {
			t.Fatalf("Qwen compose rewrote %s: err=%v before=%q after=%q", path, err, before, after)
		}
	}
	for _, path := range []string{plan.Pi.State.Lock, plan.Pi.State.ModelsJSON} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("Qwen compose created runtime state %s: %v", path, err)
		}
	}
}
