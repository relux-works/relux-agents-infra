package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/modelharness"
)

// TestREADMEQwenLocalProfileResolvesWithDefaultEngine is the golden
// compatibility test: the deployed `qwen-local` profile documented in
// README.md carries no `engine` field. Adding the engine axis must not change
// its resolved plan — engine defaults to `mlx-lm` and the raw executable/argv
// path is untouched.
func TestREADMEQwenLocalProfileResolvesWithDefaultEngine(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join(sourceRepoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	profile := documentedTOMLBlock(t, string(readme), "[profiles.qwen-local]")
	if strings.Contains(profile, "engine") {
		t.Fatalf("the deployed qwen-local profile must stay engine-field-free to prove the default path: %s", profile)
	}

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := modelharness.Resolve(path, "qwen-local", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("the documented qwen-local profile does not resolve: %v", err)
	}
	if plan.Engine != "mlx-lm" {
		t.Fatalf("engine = %q, want default mlx-lm", plan.Engine)
	}
	want := []string{
		"-c", "from mlx_lm.server import main; main()",
		"--model", "/models/Qwen",
		"--host", "127.0.0.1",
		"--port", "18011",
	}
	if !reflect.DeepEqual(plan.Argv, want) {
		t.Fatalf("argv = %#v, want %#v (adding the engine axis must not change this plan)", plan.Argv, want)
	}
}

// TestREADMEEngineKnobProfilesResolveAsDocumented takes the mlx-lm and
// llama-cpp declarative knob examples out of README.md and hands those exact
// bytes to the production resolver, so the documented per-engine knob
// translation table is an artifact the build checks rather than prose next to
// the code.
func TestREADMEEngineKnobProfilesResolveAsDocumented(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join(sourceRepoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	document := string(readme)

	tests := []struct {
		marker  string
		profile string
		want    []string
	}{
		{
			marker:  "[profiles.qwen-local-mlx-lm]",
			profile: "qwen-local-mlx-lm",
			want: []string{
				"-c", "from mlx_lm.server import main; main()",
				"--host", "127.0.0.1",
				"--port", "18011",
				"--model", "/models/Qwen",
				"--max-kv-size", "76800",
				"--prefill-step-size", "2048",
				"--chat-template-args", `{"reasoning_effort": "medium"}`,
			},
		},
		{
			marker:  "[profiles.qwen-local-llama-cpp]",
			profile: "qwen-local-llama-cpp",
			want: []string{
				"--host", "127.0.0.1",
				"--port", "18011",
				"--no-webui",
				"--model", "/absolute/path/to/model.gguf",
				"--ctx-size", "8192",
				"--ubatch-size", "2048",
				"--reasoning-effort", "medium",
				"--spec-type", "ngram-mod",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.profile, func(t *testing.T) {
			block := documentedTOMLBlock(t, document, test.marker)
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(block), 0o600); err != nil {
				t.Fatal(err)
			}
			plan, err := modelharness.Resolve(path, test.profile, "127.0.0.1", 18011)
			if err != nil {
				t.Fatalf("the documented %s profile does not resolve: %v", test.profile, err)
			}
			if !reflect.DeepEqual(plan.Argv, test.want) {
				t.Fatalf("argv = %#v, want %#v", plan.Argv, test.want)
			}
		})
	}
}

// TestREADMECrossReferencesTheEngineAdapterSpec pins the doc obligation from
// TASK-260830-2cgim0's acceptance criteria: the profile documentation must
// name the canonical-knob-set spec it translates, not just describe knobs
// that happen to match it.
func TestREADMECrossReferencesTheEngineAdapterSpec(t *testing.T) {
	readme := normalizedOperatorDoc(t, filepath.Join(sourceRepoRoot(t), "README.md"))
	const spec = ".research/260831_engine-adapter-contract-and-canonical-knob-set.spec.md"
	if !strings.Contains(readme, spec) {
		t.Fatalf("README.md no longer cross-references %s", spec)
	}
	specPath := filepath.Join(sourceRepoRoot(t), spec)
	if _, err := os.Stat(specPath); err != nil {
		t.Fatalf("cross-referenced spec is missing: %v", err)
	}
}
