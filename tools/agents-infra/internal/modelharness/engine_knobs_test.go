package modelharness

import (
	"reflect"
	"strings"
	"testing"
)

func containsToken(argv []string, token string) bool {
	for _, candidate := range argv {
		if candidate == token {
			return true
		}
	}
	return false
}

// TestResolveDefaultsEngineToMLXLM proves the backward-compat golden path:
// a profile that never mentions engine resolves as mlx-lm, matching every
// profile deployed before this axis existed.
func TestResolveDefaultsEngineToMLXLM(t *testing.T) {
	config := writeConfig(t, `
[profiles.qwen-local]
mode = "local"
executable = "/bin/echo"
argv = ["serve", "--host", "{host}", "--port", "{port}"]
`)
	plan, err := Resolve(config, "qwen-local", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if plan.Engine != "mlx-lm" {
		t.Fatalf("engine = %q, want default mlx-lm", plan.Engine)
	}
}

// TestResolveEngineIsNeverInferredFromExecutable pins the design invariant
// named in the task's acceptance criteria directly: two profiles that launch
// the identical executable path resolve different knob translations purely
// because their declared engine field differs, and a profile pointing at an
// executable named after a different engine's binary still resolves under
// its declared engine.
func TestResolveEngineIsNeverInferredFromExecutable(t *testing.T) {
	config := writeConfig(t, `
[profiles.declared-mlx-lm]
mode = "local"
engine = "mlx-lm"
executable = "/opt/homebrew/bin/llama-server"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.declared-mlx-lm.knobs]
prefill_chunk_tokens = "2048"

[profiles.declared-llama-cpp]
mode = "local"
engine = "llama-cpp"
executable = "/opt/homebrew/bin/llama-server"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.declared-llama-cpp.knobs]
prefill_chunk_tokens = "2048"
`)
	mlxPlan, err := Resolve(config, "declared-mlx-lm", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("Resolve(declared-mlx-lm): %v", err)
	}
	if !containsToken(mlxPlan.Argv, "--prefill-step-size") {
		t.Fatalf("executable named llama-server must not force llama-cpp translation: argv=%#v", mlxPlan.Argv)
	}

	llamaPlan, err := Resolve(config, "declared-llama-cpp", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("Resolve(declared-llama-cpp): %v", err)
	}
	if !containsToken(llamaPlan.Argv, "--ubatch-size") {
		t.Fatalf("declared engine must drive translation regardless of executable: argv=%#v", llamaPlan.Argv)
	}
}

func TestResolveTranslatesKnobsPerEngine(t *testing.T) {
	tests := []struct {
		name   string
		engine string
		knobs  string
		want   []string
	}{
		{
			name:   "mlx-lm bounded kv, prefill, reasoning",
			engine: "mlx-lm",
			knobs: `
kv_context_tokens = "76800"
prefill_chunk_tokens = "2048"
reasoning_effort = "medium"
`,
			want: []string{"--max-kv-size", "76800", "--prefill-step-size", "2048", "--chat-template-args", `{"reasoning_effort": "medium"}`},
		},
		{
			name:   "mlx-lm unbounded kv omits the flag",
			engine: "mlx-lm",
			knobs:  `kv_context_tokens = "unbounded"`,
			want:   nil,
		},
		{
			name:   "mlx-swift bounded kv and reasoning",
			engine: "mlx-swift",
			knobs: `
kv_context_tokens = "32768"
reasoning_effort = "xhigh"
`,
			want: []string{"--max-kv-size", "32768", "--reasoning-effort", "xhigh"},
		},
		{
			name:   "llama-cpp ctx-size, ubatch-size, reasoning, speculative off",
			engine: "llama-cpp",
			knobs: `
kv_context_tokens = "8192"
prefill_chunk_tokens = "2048"
reasoning_effort = "low"
speculative_decoding = "off"
`,
			want: []string{"--ctx-size", "8192", "--ubatch-size", "2048", "--reasoning-effort", "low"},
		},
		{
			name:   "llama-cpp mtp speculative decoding",
			engine: "llama-cpp",
			knobs:  `speculative_decoding = "mtp"`,
			want:   []string{"--spec-type", "draft-mtp"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "`+test.engine+`"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.p.knobs]
`+test.knobs+`
`)
			plan, err := Resolve(config, "p", "127.0.0.1", 18011)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			base := []string{"--host", "127.0.0.1", "--port", "18011"}
			want := append(append([]string(nil), base...), test.want...)
			if !reflect.DeepEqual(plan.Argv, want) {
				t.Fatalf("argv = %#v, want %#v", plan.Argv, want)
			}
		})
	}
}

func TestResolveTranslatesModelToTrailingFlag(t *testing.T) {
	config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "mlx-lm"
executable = "/bin/echo"
model = "/models/Qwen"
argv = ["--host", "{host}", "--port", "{port}"]
`)
	plan, err := Resolve(config, "p", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := []string{"--host", "127.0.0.1", "--port", "18011", "--model", "/models/Qwen"}
	if !reflect.DeepEqual(plan.Argv, want) {
		t.Fatalf("argv = %#v, want %#v", plan.Argv, want)
	}
	if plan.Model != "/models/Qwen" {
		t.Fatalf("plan.Model = %q", plan.Model)
	}
}

// TestResolveRejectsUnknownEngine is the first of the three negative tests
// the task's acceptance criteria names explicitly.
func TestResolveRejectsUnknownEngine(t *testing.T) {
	config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "vllm"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]
`)
	_, err := Resolve(config, "p", "127.0.0.1", 18011)
	if err == nil || !strings.Contains(err.Error(), `unknown engine "vllm"`) {
		t.Fatalf("error = %v, want unknown engine refusal naming the engine", err)
	}
}

// TestResolveRejectsUnexpressibleKnobValue is the second named negative test:
// llama-cpp's --ctx-size has no unbounded form, so kv_context_tokens =
// "unbounded" must refuse naming the knob and the engine, not silently fall
// back to a finite default or to the mlx-lm behaviour of omitting the flag.
func TestResolveRejectsUnexpressibleKnobValue(t *testing.T) {
	config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "llama-cpp"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.p.knobs]
kv_context_tokens = "unbounded"
`)
	_, err := Resolve(config, "p", "127.0.0.1", 18011)
	if err == nil {
		t.Fatal("expected refusal, got nil error")
	}
	for _, want := range []string{`"kv_context_tokens"`, `"llama-cpp"`} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want it to name %s", err, want)
		}
	}
}

// TestResolveRejectsKnobOnlyValidForAnotherEngine is the third named negative
// test: speculative_decoding has no launch flag on mlx-lm or mlx-swift, so
// setting it there must refuse naming the knob and the engine rather than
// dropping it.
func TestResolveRejectsKnobOnlyValidForAnotherEngine(t *testing.T) {
	for _, engine := range []string{"mlx-lm", "mlx-swift"} {
		t.Run(engine, func(t *testing.T) {
			config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "`+engine+`"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.p.knobs]
speculative_decoding = "ngram"
`)
			_, err := Resolve(config, "p", "127.0.0.1", 18011)
			if err == nil {
				t.Fatal("expected refusal, got nil error")
			}
			for _, want := range []string{`"speculative_decoding"`, `"` + engine + `"`, "is not valid for engine"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want it to contain %q", err, want)
				}
			}
		})
	}
}

func TestResolveRejectsInvalidReasoningEffort(t *testing.T) {
	config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "llama-cpp"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.p.knobs]
reasoning_effort = "extreme"
`)
	_, err := Resolve(config, "p", "127.0.0.1", 18011)
	if err == nil || !strings.Contains(err.Error(), "must be one of low, medium, xhigh") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveRejectsNonPositivePrefillChunkTokens(t *testing.T) {
	config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "mlx-lm"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"]

[profiles.p.knobs]
prefill_chunk_tokens = "0"
`)
	_, err := Resolve(config, "p", "127.0.0.1", 18011)
	if err == nil || !strings.Contains(err.Error(), "must be a positive integer") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveRejectsDuplicateModelDeclaration(t *testing.T) {
	config := writeConfig(t, `
[profiles.p]
mode = "local"
engine = "mlx-lm"
executable = "/bin/echo"
model = "/models/Qwen"
argv = ["--model", "/models/Qwen", "--host", "{host}", "--port", "{port}"]
`)
	_, err := Resolve(config, "p", "127.0.0.1", 18011)
	if err == nil || !strings.Contains(err.Error(), "argv must not declare --model when model is set") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveRejectsSSHProfileDeclaringEngineModelOrKnobs(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "engine",
			body: `
[profiles.remote]
mode = "ssh"
ssh_target = "host"
engine = "llama-cpp"
remote_executable = "/bin/model-harness"
remote_profile = "tiny"
remote_host = "127.0.0.1"
remote_port = 18011
`,
		},
		{
			name: "model",
			body: `
[profiles.remote]
mode = "ssh"
ssh_target = "host"
model = "/models/Qwen"
remote_executable = "/bin/model-harness"
remote_profile = "tiny"
remote_host = "127.0.0.1"
remote_port = 18011
`,
		},
		{
			name: "knobs",
			body: `
[profiles.remote]
mode = "ssh"
ssh_target = "host"
remote_executable = "/bin/model-harness"
remote_profile = "tiny"
remote_host = "127.0.0.1"
remote_port = 18011

[profiles.remote.knobs]
reasoning_effort = "medium"
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := writeConfig(t, test.body)
			_, err := Resolve(config, "remote", "127.0.0.1", 18011)
			if err == nil || !strings.Contains(err.Error(), "ssh mode cannot declare engine, model, or knobs") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
