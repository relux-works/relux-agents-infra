//go:build !windows

package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// modelHarnessBackedPiProfileTOML builds a project-config.toml body whose Pi
// profile launches through `model-harness run qwen-local --config ...`, the
// same shape the deployed qwen-3.8-27b-mlx-8bit profile uses.
func modelHarnessBackedPiProfileTOML(name string, port, contextWindow int, harnessConfigPath string) string {
	body := validPiProfileTOML(name, "/usr/local/bin/model-harness", port, false)
	body = strings.Replace(body, "context_window = 8192", fmt.Sprintf("context_window = %d", contextWindow), 1)
	oldArgv := fmt.Sprintf(`["serve", "--model", "Model", "--host", "127.0.0.1", "--port", "%d"]`, port)
	newArgv := fmt.Sprintf(`["run", "qwen-local", "--config", %q, "--host", "127.0.0.1", "--port", "%d"]`, harnessConfigPath, port)
	body = strings.Replace(body, oldArgv, newArgv, 1)
	if !strings.Contains(body, newArgv) {
		panic("modelHarnessBackedPiProfileTOML: fixture argv substitution did not match")
	}
	return body
}

func writeModelHarnessConfig(t *testing.T, dir, name string, kvArgvSuffix string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	body := fmt.Sprintf(`
[profiles.qwen-local]
mode = "local"
executable = "/bin/echo"
argv = ["--host", "{host}", "--port", "{port}"%s]
`, kvArgvSuffix)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write model-harness config: %v", err)
	}
	return path
}

// Production call site: parseProjectConfig -> parsePiProfile ->
// validatePiModelHarnessKVBound, resolving the real referenced
// model-harness config through modelharness.Resolve. Both negative shapes
// name context_window and (where present) --max-kv-size in the refusal.
func TestPiProfileRefusesContextWindowAboveModelHarnessKVBoundAtProductionEntry(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing --max-kv-size entirely", func(t *testing.T) {
		harnessConfig := writeModelHarnessConfig(t, dir, "missing-kv.toml", "")
		body := modelHarnessBackedPiProfileTOML("profile", 18011, 75000, harnessConfig)
		_, err := parseProjectConfig([]byte(body), "/project/config.toml")
		if err == nil {
			t.Fatal("parseProjectConfig admitted a profile whose model-harness runtime has no --max-kv-size bound at all")
		}
		if !strings.Contains(err.Error(), "75000") || !strings.Contains(err.Error(), "--max-kv-size") {
			t.Fatalf("error does not name context_window and --max-kv-size: %v", err)
		}
	})

	t.Run("bound below context_window", func(t *testing.T) {
		harnessConfig := writeModelHarnessConfig(t, dir, "low-kv.toml", `, "--max-kv-size", "70000"`)
		body := modelHarnessBackedPiProfileTOML("profile", 18012, 75000, harnessConfig)
		_, err := parseProjectConfig([]byte(body), "/project/config.toml")
		if err == nil {
			t.Fatal("parseProjectConfig admitted context_window=75000 above --max-kv-size=70000")
		}
		if !strings.Contains(err.Error(), "75000") || !strings.Contains(err.Error(), "70000") {
			t.Fatalf("error does not name both context_window and --max-kv-size: %v", err)
		}
	})

	// Exact off-by-one at the boundary: narrows a mutated comparison such as
	// `bound+1 < contextWindow` (which would wrongly admit this exact gap)
	// down to the true bound < contextWindow rule.
	t.Run("bound exactly one below context_window", func(t *testing.T) {
		harnessConfig := writeModelHarnessConfig(t, dir, "off-by-one-kv.toml", `, "--max-kv-size", "74999"`)
		body := modelHarnessBackedPiProfileTOML("profile", 18016, 75000, harnessConfig)
		_, err := parseProjectConfig([]byte(body), "/project/config.toml")
		if err == nil {
			t.Fatal("parseProjectConfig admitted context_window=75000 above --max-kv-size=74999")
		}
		if !strings.Contains(err.Error(), "75000") || !strings.Contains(err.Error(), "74999") {
			t.Fatalf("error does not name both context_window and --max-kv-size: %v", err)
		}
	})
}

// Same production call site as the refusal test above, driven with pairs
// that must be admitted: an exact match and the deployed qwen-local pair
// (context_window 75000, --max-kv-size 76800) from
// /Users/alexis/src/.agents/.configs/model-harness.toml.
func TestPiProfileAdmitsContextWindowAtOrBelowModelHarnessKVBoundAtProductionEntry(t *testing.T) {
	dir := t.TempDir()

	t.Run("bound exactly equals context_window", func(t *testing.T) {
		harnessConfig := writeModelHarnessConfig(t, dir, "equal-kv.toml", `, "--max-kv-size", "75000"`)
		body := modelHarnessBackedPiProfileTOML("profile", 18013, 75000, harnessConfig)
		cfg, err := parseProjectConfig([]byte(body), "/project/config.toml")
		if err != nil {
			t.Fatalf("parseProjectConfig refused a consistent context_window/--max-kv-size pair: %v", err)
		}
		if cfg.PiProfiles["profile"].ContextWindow != 75000 {
			t.Fatalf("context_window=%d want 75000", cfg.PiProfiles["profile"].ContextWindow)
		}
	})

	t.Run("deployed pair: bound above context_window", func(t *testing.T) {
		harnessConfig := writeModelHarnessConfig(t, dir, "deployed-kv.toml", `, "--max-kv-size", "76800"`)
		body := modelHarnessBackedPiProfileTOML("profile", 18014, 75000, harnessConfig)
		if _, err := parseProjectConfig([]byte(body), "/project/config.toml"); err != nil {
			t.Fatalf("parseProjectConfig refused the deployed context_window=75000/--max-kv-size=76800 pair: %v", err)
		}
	})
}

// Locks in that the guard fires only for a runtime that is actually
// model-harness; every other existing Pi runtime shape (used pervasively by
// other tests in this package with executables like /bin/echo) must remain
// unaffected. This is the negative control for modelHarnessRunInvocation's
// executable-name gate: removing it would make every non-model-harness
// profile above pass through unchecked, which this test cannot detect by
// itself, but a false-positive match here would fail immediately.
func TestPiProfileModelHarnessKVBoundIgnoresNonModelHarnessRuntimes(t *testing.T) {
	body := validPiProfileTOML("profile", "/bin/echo", 18015, false)
	if _, err := parseProjectConfig([]byte(body), "/project/config.toml"); err != nil {
		t.Fatalf("non-model-harness runtime was rejected by the KV-bound guard: %v", err)
	}
}
