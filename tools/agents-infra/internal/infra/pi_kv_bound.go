package infra

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/modelharness"
)

// validatePiModelHarnessKVBound enforces that a Pi profile's context_window
// never exceeds the active-generation KV cache bound of the model-harness
// profile it launches.
//
// The active generation's KV cache is a fixed-size ring once --max-kv-size is
// set: it does not refuse a request that no longer fits, it silently
// overwrites the oldest tokens, and the resulting truncated response is
// indistinguishable from one that fit cleanly. A context_window above the
// bound therefore fails silently rather than loudly, which is worse than
// refusing to start.
//
// This is checked against the real referenced model-harness config rather
// than a value copied by hand into project-config.toml, because the two
// files are edited by different tools (agents-infra vs. model-harness) and
// have already drifted once in production: the deployed qwen-3.8-27b-mlx-8bit
// profile ran with context_window=75000 while the referenced qwen-local
// model-harness profile carried no --max-kv-size at all, until an operator
// hand-patched it on 2026-08-30.
//
// --prompt-cache-bytes is not this bound and must not be read as one. It
// caps a *different* mechanism: the pool of stored prefix caches
// mlx_lm-relux.server reuses across requests, not the active generation's KV
// cache. A profile can carry a generous --prompt-cache-bytes and still run
// every single generation against an unbounded (or too-small) active KV
// cache; only --max-kv-size bounds that.
func validatePiModelHarnessKVBound(contextWindow int, baseURL string, runtime PiRuntime) error {
	harnessProfile, configPath, ok := modelHarnessRunInvocation(runtime.Executable, runtime.Argv)
	if !ok {
		return nil
	}
	host, port, err := piBaseURLEndpoint(baseURL)
	if err != nil {
		// Already reported by the base_url validator that runs before this one.
		return nil
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return nil
	}
	plan, err := modelharness.Resolve(configPath, harnessProfile, host, portNumber)
	if err != nil {
		return fmt.Errorf("resolve referenced model-harness profile %q: %w", harnessProfile, err)
	}
	if plan.Mode != "local" {
		// The KV bound lives on whatever host actually runs the model (for
		// example over ssh); a local config parse cannot observe it.
		return nil
	}
	bound, present, err := maxKVSizeFromArgv(plan.Argv)
	if err != nil {
		return fmt.Errorf("model-harness profile %q (%s) argv: %w", harnessProfile, plan.Config, err)
	}
	if !present {
		return fmt.Errorf("model-harness profile %q (%s) has no --max-kv-size bound but context_window is %d: an unbounded active-generation KV cache silently truncates instead of refusing", harnessProfile, plan.Config, contextWindow)
	}
	if bound < contextWindow {
		return fmt.Errorf("model-harness profile %q (%s) --max-kv-size %d is less than context_window %d: raise --max-kv-size to at least context_window, or lower context_window to at most --max-kv-size", harnessProfile, plan.Config, bound, contextWindow)
	}
	return nil
}

// modelHarnessRunInvocation recognizes a Pi runtime that launches a model
// through `model-harness run PROFILE [--config PATH] ...` and reports the
// referenced model-harness profile name and config path. configPath is
// empty when the invocation relies on model-harness's own default config
// resolution, matching modelharness.Resolve's own contract.
func modelHarnessRunInvocation(executable string, argv []string) (profile, configPath string, ok bool) {
	if filepath.Base(executable) != "model-harness" {
		return "", "", false
	}
	if len(argv) < 2 || argv[0] != "run" {
		return "", "", false
	}
	profile = argv[1]
	for i := 2; i+1 < len(argv); i++ {
		if argv[i] == "--config" {
			configPath = argv[i+1]
		}
	}
	return profile, configPath, true
}

// maxKVSizeFromArgv reads the exact --max-kv-size value model-harness would
// pass to the local runtime, without executing anything.
func maxKVSizeFromArgv(argv []string) (value int, present bool, err error) {
	for i, token := range argv {
		if token != "--max-kv-size" {
			continue
		}
		if present {
			return 0, false, errors.New("--max-kv-size must appear at most once")
		}
		if i+1 >= len(argv) {
			return 0, false, errors.New("--max-kv-size must be followed by a value")
		}
		n, convErr := strconv.Atoi(argv[i+1])
		if convErr != nil || n <= 0 {
			return 0, false, fmt.Errorf("--max-kv-size value %q must be a positive integer", argv[i+1])
		}
		value, present = n, true
	}
	return value, present, nil
}
