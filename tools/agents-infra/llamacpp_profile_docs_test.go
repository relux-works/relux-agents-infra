package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/modelharness"
)

// TestREADMELlamaCppProfileResolvesAsDocumented takes the llama.cpp profile out
// of README.md and hands those exact bytes to the production resolver, so the
// documented profile is an artifact the build checks rather than prose next to
// the code. A README profile that no longer resolves — a renamed field, a
// dropped placeholder, a flag moved out of the pinned set — fails here.
func TestREADMELlamaCppProfileResolvesAsDocumented(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join(sourceRepoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	profile := documentedTOMLBlock(t, string(readme), "[profiles.llamacpp-local]")

	// The resolver refuses a non-absolute or non-regular config path, so the
	// documented bytes are written out and resolved exactly as an operator's
	// own config would be.
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := modelharness.Resolve(path, "llamacpp-local", "127.0.0.1", 18011)
	if err != nil {
		t.Fatalf("the documented llama.cpp profile does not resolve: %v", err)
	}
	if plan.Endpoint != "http://127.0.0.1:18011/v1" {
		t.Fatalf("endpoint = %q", plan.Endpoint)
	}
	want := []string{
		"--model", "/absolute/path/to/model.gguf",
		"--host", "127.0.0.1",
		"--port", "18011",
		"--ctx-size", "8192",
		"--batch-size", "2048",
		"--ubatch-size", "2048",
		"--reasoning-effort", "medium",
		"--no-webui",
	}
	if !reflect.DeepEqual(plan.Argv, want) {
		t.Fatalf("rendered argv = %#v, want %#v", plan.Argv, want)
	}

	// The pinned launch conditions. Each is a condition the runtimes under
	// comparison do not default to the same way, so dropping one from the
	// documented profile silently changes what a llama.cpp run measures.
	for _, pinned := range []string{"--ctx-size", "--batch-size", "--ubatch-size", "--reasoning-effort"} {
		if !containsToken(plan.Argv, pinned) {
			t.Fatalf("documented profile no longer pins %s; argv=%#v", pinned, plan.Argv)
		}
	}
}

// TestREADMEDocumentsTheHarnessStopContract pins the lifecycle promises the
// implementation now makes. `model-harness run` is invoked by supervisors and
// scripts that only know what this document says.
func TestREADMEDocumentsTheHarnessStopContract(t *testing.T) {
	readme := normalizedOperatorDoc(t, filepath.Join(sourceRepoRoot(t), "README.md"))
	for _, want := range []string{
		"It places the runtime in its own process group",
		"forwards `SIGINT`, `SIGTERM` and `SIGHUP` to that whole group",
		"escalates to `SIGKILL` if the group has not exited within ten seconds",
		"exits `0` when a signalled stop completes",
		"A signalled stop is never treated as a restartable failure",
		"On Windows there is no process group here and only the direct child is stopped.",
		"`GET /v1/models` answers `503` until the weights are resident",
		"| stdout | empty; every line is on stderr |",
		"| HTTP access line (method, path, status) | **none at any verbosity** |",
		"that is an unknown outcome rather than a false success",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md no longer documents %q", want)
		}
	}
}

// documentedTOMLBlock returns the fenced toml block that contains marker.
func documentedTOMLBlock(t *testing.T, document, marker string) string {
	t.Helper()
	rest := document
	for {
		start := strings.Index(rest, "```toml\n")
		if start < 0 {
			break
		}
		rest = rest[start+len("```toml\n"):]
		end := strings.Index(rest, "```")
		if end < 0 {
			break
		}
		block := rest[:end]
		if strings.Contains(block, marker) {
			return block
		}
		rest = rest[end:]
	}
	t.Fatalf("README.md has no toml block containing %q", marker)
	return ""
}

func containsToken(argv []string, token string) bool {
	for _, candidate := range argv {
		if candidate == token {
			return true
		}
	}
	return false
}
