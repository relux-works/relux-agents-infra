//go:build !windows

package infra

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func validPiProfileTOML(name, runtime string, port int, dflash bool) string {
	extraCaps := `requested_capabilities = ["text", "tools"]`
	dflashBlock := ""
	argv := fmt.Sprintf(`["serve", "--model", "Model", "--host", "127.0.0.1", "--port", "%d"]`, port)
	if dflash {
		extraCaps = `requested_capabilities = ["dflash", "text", "tools"]`
		argv = fmt.Sprintf(`["serve", "--model", "Model", "--draft", "Draft", "--host", "127.0.0.1", "--port", "%d"]`, port)
		dflashBlock = fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.dflash]
target_model = "Model"
draft_model = "Draft"
target_argv = ["--model", "Model"]
draft_argv = ["--draft", "Draft"]
`, name)
	}
	return fmt.Sprintf(`
[agents.pi.primary_session]
profile = %q
pi_compatibility = %q

[agents.pi.profiles.%q]
provider = "local-provider"
model = "Model"
base_url = "http://127.0.0.1:%d/v1"
api = "openai-completions"
reasoning = false
input = ["text"]
context_window = 8192
max_tokens = 1024
thinking = "off"
%s

[agents.pi.profiles.%q.compat]
supports_developer_role = false
supports_reasoning_effort = false
supports_usage_in_streaming = true
supports_finish_reason = true
max_tokens_field = "max_tokens"

[agents.pi.profiles.%q.runtime]
executable = %q
argv = %s
readiness_path = "/models"
startup_timeout_seconds = 5
shutdown_timeout_seconds = 2
%s`, name, PiCompatibilityV0842DarwinARM64, name, port, extraCaps, name, name, runtime, argv, dflashBlock)
}

func reasoningPiProfileTOML(name, runtime string, port int) string {
	body := validPiProfileTOML(name, runtime, port, false)
	body = strings.Replace(body, `reasoning = false`, `reasoning = true`, 1)
	body = strings.Replace(body, `thinking = "off"`, `thinking = "medium"`, 1)
	body = strings.Replace(body, `max_tokens_field = "max_tokens"`, "max_tokens_field = \"max_tokens\"\nthinking_format = \"qwen-chat-template\"", 1)
	return body
}

func compactionPiProfileTOML(name, runtime string, port int) string {
	return validPiProfileTOML(name, runtime, port, false) + fmt.Sprintf(`
[agents.pi.profiles.%q.compaction]
enabled = true
compact_at_tokens = 6144
keep_recent_tokens = 2048
`, name)
}

func TestParsePiPolicyExactSchemaAndMuseDFlash(t *testing.T) {
	for _, dflash := range []bool{false, true} {
		cfg, err := parseProjectConfig([]byte(validPiProfileTOML("profile", "/bin/echo", 18011, dflash)), "/project/config.toml")
		if err != nil {
			t.Fatalf("parse dflash=%t: %v", dflash, err)
		}
		p := cfg.PiProfiles["profile"]
		if (p.Runtime.DFlash != nil) != dflash {
			t.Fatalf("DFlash present=%t, want %t", p.Runtime.DFlash != nil, dflash)
		}
		wantCaps := []string{"text", "tools"}
		if dflash {
			wantCaps = []string{"dflash", "text", "tools"}
		}
		if !reflect.DeepEqual(p.RequestedCapabilities, wantCaps) {
			t.Fatalf("capabilities=%q want=%q", p.RequestedCapabilities, wantCaps)
		}
	}
}

func TestParsePiCompactionIsStrictAndProfileScoped(t *testing.T) {
	without, err := parseProjectConfig([]byte(validPiProfileTOML("profile", "/bin/echo", 18011, false)), "/project/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if without.PiProfiles["profile"].Compaction != nil {
		t.Fatal("absent compaction table changed the Pi-native defaults")
	}

	body := compactionPiProfileTOML("profile", "/bin/echo", 18011)
	with, err := parseProjectConfig([]byte(body), "/project/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	want := &PiCompaction{Enabled: true, CompactAtTokens: 6144, ReserveTokens: 2048, KeepRecentTokens: 2048}
	if got := with.PiProfiles["profile"].Compaction; !reflect.DeepEqual(got, want) {
		t.Fatalf("compaction=%#v want=%#v", got, want)
	}
	legacyBody := strings.Replace(body, "compact_at_tokens = 6144", "reserve_tokens = 2048", 1)
	legacy, err := parseProjectConfig([]byte(legacyBody), "/project/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if got := legacy.PiProfiles["profile"].Compaction; !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy reserve compaction=%#v want=%#v", got, want)
	}

	tests := map[string]string{
		"unknown field":               strings.Replace(body, "enabled = true", "enabled = true\nsurprise = 1", 1),
		"missing enabled":             strings.Replace(body, "enabled = true\n", "", 1),
		"missing threshold":           strings.Replace(body, "compact_at_tokens = 6144\n", "", 1),
		"both threshold forms":        strings.Replace(body, "compact_at_tokens = 6144", "compact_at_tokens = 6144\nreserve_tokens = 2048", 1),
		"missing kept":                strings.Replace(body, "keep_recent_tokens = 2048\n", "", 1),
		"compact leaves short output": strings.Replace(body, "compact_at_tokens = 6144", "compact_at_tokens = 7680", 1),
		"compact reaches window":      strings.Replace(body, "compact_at_tokens = 6144", "compact_at_tokens = 8192", 1),
		"kept reaches compact":        strings.Replace(body, "keep_recent_tokens = 2048", "keep_recent_tokens = 6144", 1),
	}
	for name, invalid := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseProjectConfig([]byte(invalid), "/project/config.toml"); err == nil {
				t.Fatal("strict compaction parser admitted invalid input")
			}
		})
	}
}

func sharedPiProfileTOML(name, runtime string, port int) string {
	return validPiProfileTOML(name, runtime, port, false) + fmt.Sprintf(`
[agents.pi.profiles.%q.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 8
heartbeat_interval_seconds = 15
lease_stale_seconds = 60
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
`, name)
}

func TestParsePiRuntimeSharingIsStrictAndOptIn(t *testing.T) {
	exclusive, err := parseProjectConfig([]byte(validPiProfileTOML("profile", "/bin/echo", 18011, false)), "/project/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if exclusive.PiProfiles["profile"].Runtime.Sharing != nil {
		t.Fatal("absent runtime.sharing changed the exclusive profile")
	}

	sharedBody := sharedPiProfileTOML("profile", "/bin/echo", 18011)
	shared, err := parseProjectConfig([]byte(sharedBody), "/project/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	want := &PiRuntimeSharing{
		Mode: "shared", LingerSeconds: 0, MaxLeases: 8,
		HeartbeatIntervalSeconds: 15, LeaseStaleSeconds: 60, BrokerStartTimeoutSeconds: 40,
		RestartLimit: 3, RestartInitialBackoffSeconds: 1, RestartMaxBackoffSeconds: 4,
		StableRunSeconds: 10, QuarantineSeconds: 30,
	}
	if got := shared.PiProfiles["profile"].Runtime.Sharing; !reflect.DeepEqual(got, want) {
		t.Fatalf("sharing=%#v want=%#v", got, want)
	}

	tests := map[string]string{
		"unknown field":                  strings.Replace(sharedBody, `mode = "shared"`, "mode = \"shared\"\nsurprise = 1", 1),
		"missing mode":                   strings.Replace(sharedBody, "mode = \"shared\"\n", "", 1),
		"missing linger":                 strings.Replace(sharedBody, "linger_seconds = 0\n", "", 1),
		"missing max leases":             strings.Replace(sharedBody, "max_leases = 8\n", "", 1),
		"missing heartbeat":              strings.Replace(sharedBody, "heartbeat_interval_seconds = 15\n", "", 1),
		"missing stale":                  strings.Replace(sharedBody, "lease_stale_seconds = 60\n", "", 1),
		"missing broker timeout":         strings.Replace(sharedBody, "broker_start_timeout_seconds = 40\n", "", 1),
		"missing restart limit":          strings.Replace(sharedBody, "restart_limit = 3\n", "", 1),
		"missing initial backoff":        strings.Replace(sharedBody, "restart_initial_backoff_seconds = 1\n", "", 1),
		"missing maximum backoff":        strings.Replace(sharedBody, "restart_max_backoff_seconds = 4\n", "", 1),
		"missing stable run":             strings.Replace(sharedBody, "stable_run_seconds = 10\n", "", 1),
		"missing quarantine":             strings.Replace(sharedBody, "quarantine_seconds = 30\n", "", 1),
		"unknown mode":                   strings.Replace(sharedBody, `mode = "shared"`, `mode = "automatic"`, 1),
		"negative linger":                strings.Replace(sharedBody, "linger_seconds = 0", "linger_seconds = -1", 1),
		"zero max leases":                strings.Replace(sharedBody, "max_leases = 8", "max_leases = 0", 1),
		"heartbeat equals stale":         strings.Replace(sharedBody, "heartbeat_interval_seconds = 15", "heartbeat_interval_seconds = 60", 1),
		"broker timeout below bound":     strings.Replace(sharedBody, "broker_start_timeout_seconds = 40", "broker_start_timeout_seconds = 36", 1),
		"zero restart limit":             strings.Replace(sharedBody, "restart_limit = 3", "restart_limit = 0", 1),
		"zero initial backoff":           strings.Replace(sharedBody, "restart_initial_backoff_seconds = 1", "restart_initial_backoff_seconds = 0", 1),
		"maximum below initial":          strings.Replace(sharedBody, "restart_initial_backoff_seconds = 1", "restart_initial_backoff_seconds = 5", 1),
		"maximum reaches broker timeout": strings.Replace(sharedBody, "restart_max_backoff_seconds = 4", "restart_max_backoff_seconds = 40", 1),
		"zero stable run":                strings.Replace(sharedBody, "stable_run_seconds = 10", "stable_run_seconds = 0", 1),
		"zero quarantine":                strings.Replace(sharedBody, "quarantine_seconds = 30", "quarantine_seconds = 0", 1),
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseProjectConfig([]byte(body), "/project/config.toml"); err == nil {
				t.Fatal("strict runtime.sharing parser admitted invalid input")
			}
		})
	}
}

// Production call site: RunPi -> loadCompositeProjectConfig -> parsePiRuntime.
// Every invalid policy must be refused before provider lookup, runtime launch,
// or creation of the shared-runtime tree and restart ledger.
func TestRunPiRejectsSupervisionSecondsThatOverflowEffectiveDurations(t *testing.T) {
	tooLarge := strconv.FormatInt(maxTimeDurationSeconds+1, 10)
	tooLargeLeaseStale := strconv.FormatInt(maxSharedRuntimeLeaseStaleSeconds+1, 10)
	replace := func(old, replacement string) func(string) string {
		return func(body string) string { return strings.Replace(body, old, replacement, 1) }
	}
	tests := []struct {
		name      string
		mutate    func(string) string
		wantField string
		wantBound string
	}{
		{name: "runtime startup", mutate: replace("startup_timeout_seconds = 5", "startup_timeout_seconds = "+tooLarge), wantField: "runtime.startup_timeout_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "runtime shutdown", mutate: replace("shutdown_timeout_seconds = 2", "shutdown_timeout_seconds = "+tooLarge), wantField: "runtime.shutdown_timeout_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "linger", mutate: replace("linger_seconds = 0", "linger_seconds = "+tooLarge), wantField: "sharing.linger_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "heartbeat interval", mutate: replace("heartbeat_interval_seconds = 15", "heartbeat_interval_seconds = "+tooLarge), wantField: "sharing.heartbeat_interval_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "doubled lease stale", mutate: replace("lease_stale_seconds = 60", "lease_stale_seconds = "+tooLargeLeaseStale), wantField: "sharing.lease_stale_seconds", wantBound: strconv.FormatInt(maxSharedRuntimeLeaseStaleSeconds, 10)},
		{name: "broker start", mutate: replace("broker_start_timeout_seconds = 40", "broker_start_timeout_seconds = "+tooLarge), wantField: "sharing.broker_start_timeout_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "restart initial backoff", mutate: replace("restart_initial_backoff_seconds = 1", "restart_initial_backoff_seconds = "+tooLarge), wantField: "sharing.restart_initial_backoff_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "restart maximum backoff", mutate: replace("restart_max_backoff_seconds = 4", "restart_max_backoff_seconds = "+tooLarge), wantField: "sharing.restart_max_backoff_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "stable run", mutate: replace("stable_run_seconds = 10", "stable_run_seconds = "+tooLarge), wantField: "sharing.stable_run_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{name: "quarantine", mutate: replace("quarantine_seconds = 30", "quarantine_seconds = "+tooLarge), wantField: "sharing.quarantine_seconds", wantBound: strconv.FormatInt(maxTimeDurationSeconds, 10)},
		{
			name: "handoff sum",
			mutate: replace(
				"shutdown_timeout_seconds = 2",
				"shutdown_timeout_seconds = "+strconv.FormatInt(maxTimeDurationSeconds-1, 10),
			),
			wantField: "sharing.linger_seconds",
			wantBound: "runtime shutdown + 2 seconds",
		},
		{
			name: "broker ordering sum",
			mutate: func(body string) string {
				half := strconv.FormatInt(maxTimeDurationSeconds/2+1, 10)
				body = strings.Replace(body, "startup_timeout_seconds = 5", "startup_timeout_seconds = "+half, 1)
				body = strings.Replace(body, "shutdown_timeout_seconds = 2", "shutdown_timeout_seconds = "+half, 1)
				return strings.Replace(body, "broker_start_timeout_seconds = 40", "broker_start_timeout_seconds = "+strconv.FormatInt(maxTimeDurationSeconds, 10), 1)
			},
			wantField: "sharing.broker_start_timeout_seconds",
			wantBound: "runtime startup + shutdown + 30 seconds",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, home := t.TempDir(), t.TempDir()
			cache := filepath.Join(t.TempDir(), "cache")
			writePiProjectConfig(t, project, test.mutate(sharedPiProfileTOML("profile", "/bin/echo", 18011)))
			providerLookup := false
			err := RunPi(RunPiOptions{
				ProjectDir: project,
				HomeDir:    home,
				CacheRoot:  cache,
				Environ:    []string{},
				LookPath: func(string) (string, error) {
					providerLookup = true
					return "/bin/false", nil
				},
			})
			if piErrorCode(err) != "invalid_project_configuration" {
				t.Fatalf("RunPi error=%v code=%q", err, piErrorCode(err))
			}
			if !strings.Contains(err.Error(), test.wantField) || !strings.Contains(err.Error(), test.wantBound) {
				t.Fatalf("RunPi error lacks field/bound: %v", err)
			}
			if providerLookup {
				t.Fatal("invalid supervision duration reached provider lookup")
			}
			if _, statErr := os.Stat(cache); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("invalid supervision duration mutated runtime state: %v", statErr)
			}
		})
	}
}

func TestPiPrintConfigReportsSharedRuntimeWithoutInspectingOrCreatingIt(t *testing.T) {
	piRoot := officialPiAsset(t)
	project := t.TempDir()
	home, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	t.Setenv("HOME", home)
	writePiProjectConfig(t, project, sharedPiProfileTOML("profile", "/bin/echo", 18011))

	plan, err := BuildPrimarySessionLaunchPlan("pi", project, home, nil, ChildLaunchCompositionProducer{}, func(string) (string, error) {
		return filepath.Join(piRoot, "pi"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Pi == nil || plan.Pi.SharedRuntime == nil {
		t.Fatalf("shared runtime diagnostics absent: %#v", plan.Pi)
	}
	shared := plan.Pi.SharedRuntime
	if shared.Mode != "shared" || shared.RuntimeKey == "" || shared.ProfileDigest == "" || shared.Broker.Observed != "not-inspected" {
		t.Fatalf("shared runtime diagnostics incomplete: %#v", shared)
	}
	if plan.Pi.Runtime == nil || plan.Pi.Runtime.Ownership != "broker-owned-process-group" {
		t.Fatalf("runtime ownership=%#v", plan.Pi.Runtime)
	}
	if _, err := os.Stat(filepath.Join(cache, "agents-infra", "pi-runtimes")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("print config created or inspected shared runtime state: %v", err)
	}
}

func TestPiPrintConfigReportsManagedCompactionWithoutCreatingState(t *testing.T) {
	piRoot := officialPiAsset(t)
	project, home := t.TempDir(), t.TempDir()
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	t.Setenv("HOME", home)
	writePiProjectConfig(t, project, compactionPiProfileTOML("profile", "/bin/echo", 18011))

	plan, err := BuildPrimarySessionLaunchPlan("pi", project, home, nil, ChildLaunchCompositionProducer{}, func(string) (string, error) {
		return filepath.Join(piRoot, "pi"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Pi == nil || plan.Pi.Settings == nil || plan.Pi.Settings.Compaction == nil {
		t.Fatalf("managed compaction diagnostics absent: %#v", plan.Pi)
	}
	if got := plan.Pi.Settings.Compaction; !reflect.DeepEqual(got, &PiCompaction{Enabled: true, CompactAtTokens: 6144, ReserveTokens: 2048, KeepRecentTokens: 2048}) {
		t.Fatalf("compaction diagnostics=%#v", got)
	}
	if plan.Pi.Settings.Path != plan.Pi.State.SettingsJSON {
		t.Fatalf("settings path=%q state=%q", plan.Pi.Settings.Path, plan.Pi.State.SettingsJSON)
	}
	if _, err := os.Stat(plan.Pi.State.Root); !os.IsNotExist(err) {
		t.Fatalf("print-config created managed state: %v", err)
	}
}

func TestParsePiPolicyRejectsMalformedUnsafeUnknownAndNarrowedInputs(t *testing.T) {
	base := validPiProfileTOML("profile", "/bin/echo", 18011, false)
	tests := map[string]string{
		"unknown profile field":  strings.Replace(base, `provider = "local-provider"`, "provider = \"local-provider\"\nattestation_endpoint = \"http://127.0.0.1\"", 1),
		"localhost endpoint":     strings.Replace(base, "127.0.0.1:18011", "localhost:18011", 1),
		"wildcard endpoint":      strings.Replace(base, "127.0.0.1:18011", "0.0.0.0:18011", 1),
		"wildcard runtime bind":  strings.Replace(base, `"--host", "127.0.0.1"`, `"--host", "0.0.0.0"`, 1),
		"runtime port drift":     strings.Replace(base, `"--port", "18011"`, `"--port", "19011"`, 1),
		"missing runtime host":   strings.Replace(base, `, "--host", "127.0.0.1"`, "", 1),
		"missing runtime port":   strings.Replace(base, `, "--port", "18011"`, "", 1),
		"attached runtime host":  strings.Replace(base, `"--host", "127.0.0.1"`, `"--host=127.0.0.1"`, 1),
		"duplicate runtime port": strings.Replace(base, `"--port", "18011"`, `"--port", "18011", "--port", "18011"`, 1),
		"IPv6 endpoint":          strings.Replace(base, "127.0.0.1:18011", "[::1]:18011", 1),
		"remote endpoint":        strings.Replace(base, "127.0.0.1:18011", "192.0.2.1:18011", 1),
		"userinfo endpoint":      strings.Replace(base, "http://127.0.0.1", "http://user@127.0.0.1", 1),
		"query endpoint":         strings.Replace(base, "/v1\"", "/v1?x=1\"", 1),
		"fragment endpoint":      strings.Replace(base, "/v1\"", "/v1#x\"", 1),
		"relative executable":    strings.Replace(base, `executable = "/bin/echo"`, `executable = "runtime"`, 1),
		"empty runtime argv":     strings.Replace(base, `argv = ["serve", "--model", "Model", "--host", "127.0.0.1", "--port", "18011"]`, `argv = []`, 1),
		"NUL runtime argv":       strings.Replace(base, `argv = ["serve", "--model", "Model", "--host", "127.0.0.1", "--port", "18011"]`, `argv = ["\u0000"]`, 1),
		"wrong api":              strings.Replace(base, `api = "openai-completions"`, `api = "openai-responses"`, 1),
		"non-reasoning medium":   strings.Replace(base, `thinking = "off"`, `thinking = "medium"`, 1),
		"wrong yolo type":        strings.Replace(base, "pi_compatibility = "+strconv.Quote(PiCompatibilityV0842DarwinARM64), "pi_compatibility = "+strconv.Quote(PiCompatibilityV0842DarwinARM64)+"\nyolo_mode = \"true\"", 1),
		"capability overclaim":   strings.Replace(base, `["text", "tools"]`, `["dflash", "text", "tools"]`, 1),
		"provider unicode slash": strings.Replace(base, "local-provider", "local∕provider", 1),
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseProjectConfig([]byte(body), "/project/config.toml"); err == nil {
				t.Fatal("production parseProjectConfig accepted narrowed/unsafe Pi policy")
			}
		})
	}
}

func TestPiPrimaryYoloSafeDefaultAndNearestFalseMask(t *testing.T) {
	piRoot := officialPiAsset(t)
	home, parent := t.TempDir(), t.TempDir()
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	t.Setenv("HOME", home)
	lookPath := func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil }

	plan, err := BuildPrimarySessionLaunchPlan("pi", child, home, []string{"--version"}, ChildLaunchCompositionProducer{}, lookPath)
	if err != nil {
		t.Fatalf("safe default compose: %v", err)
	}
	if plan.Resolved.Yolo.Value || plan.Resolved.Yolo.Source != "default" || !reflect.DeepEqual(plan.LaunchVariants.Interactive.Argv, []string{"--version"}) {
		t.Fatalf("safe default changed direct Pi behavior: yolo=%#v argv=%#v", plan.Resolved.Yolo, plan.LaunchVariants.Interactive.Argv)
	}

	parentBody := strings.Replace(
		validPiProfileTOML("profile", "/bin/echo", 18011, false),
		"pi_compatibility = "+strconv.Quote(PiCompatibilityV0842DarwinARM64),
		"pi_compatibility = "+strconv.Quote(PiCompatibilityV0842DarwinARM64)+"\nyolo_mode = true",
		1,
	)
	writePiProjectConfig(t, parent, parentBody)
	childConfig := filepath.Join(child, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(childConfig))
	mustWrite(t, childConfig, "[agents.pi.primary_session]\nyolo_mode = false\n")
	plan, err = BuildPrimarySessionLaunchPlan("pi", child, home, nil, ChildLaunchCompositionProducer{}, lookPath)
	if err != nil {
		t.Fatalf("nearest false mask: %v", err)
	}
	canonicalChildConfig, err := filepath.EvalSymlinks(childConfig)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Resolved.Yolo.Value || !samePath(plan.Resolved.Yolo.Source, canonicalChildConfig) {
		t.Fatalf("nearest false did not mask ancestor true: %#v", plan.Resolved.Yolo)
	}
}

// Production call sites: BuildPrimarySessionLaunchPlan (non-launching compose)
// and RunPi (direct launch). Pi has no per-tool approval policy, so primary
// yolo means one-run project trust and must compose exactly one --approve.
func TestPiYoloTrueComposesProjectTrust(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	config := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(config))
	mustWrite(t, config, "[agents.pi.primary_session]\nyolo_mode = true\n")
	plan, err := BuildPrimarySessionLaunchPlan("pi", project, home, []string{"--approve"}, ChildLaunchCompositionProducer{}, func(string) (string, error) { return "/bin/echo", nil })
	if err != nil {
		t.Fatalf("compose yolo: %v", err)
	}
	if !plan.Resolved.Yolo.Value || !samePath(plan.Resolved.Yolo.Source, config) {
		t.Fatalf("resolved yolo = %#v", plan.Resolved.Yolo)
	}
	if got := strings.Join(plan.LaunchVariants.Interactive.Argv, " "); got != "--approve" {
		t.Fatalf("interactive argv = %q, want exactly one --approve", got)
	}
	if _, err := applyPiPrimarySessionYolo([]string{"--no-approve"}, PiPrimarySessionPolicy{YoloMode: PiPolicyBoolValue{Value: true, Present: true, Source: config}}); piErrorCode(err) != "invalid_provider_arguments" {
		t.Fatalf("conflicting no-approve error = %v", err)
	}
}

func TestPiProcessWriterPreservesFileDescriptors(t *testing.T) {
	if got := piProcessWriter(new(sync.Mutex), os.Stdout); got != os.Stdout {
		t.Fatalf("terminal file descriptor was wrapped as %T", got)
	}
	buffer := new(bytes.Buffer)
	if _, ok := piProcessWriter(new(sync.Mutex), buffer).(*piSynchronizedWriter); !ok {
		t.Fatal("non-file writer did not keep synchronized fan-in")
	}
}

func TestPiSessionLogsAreDistinctContainedAndPrivate(t *testing.T) {
	cache := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	mustMkdir(t, project)
	state, err := ResolvePiStatePaths(cache, project, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(state); err != nil {
		t.Fatal(err)
	}
	first, err := openPiSessionLog(state)
	if err != nil {
		t.Fatal(err)
	}
	first.event("pi_started", map[string]any{"pid": 42, "foreground": true})
	firstPath := first.path
	first.close()
	second, err := openPiSessionLog(state)
	if err != nil {
		t.Fatal(err)
	}
	secondPath := second.path
	second.close()
	if firstPath == secondPath {
		t.Fatalf("per-launch logs collided at %s", firstPath)
	}
	for _, path := range []string{firstPath, secondPath} {
		if filepath.Dir(path) != state.LogsDir {
			t.Fatalf("log escaped contained directory: %s", path)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("log mode = %04o, want 0600", info.Mode().Perm())
		}
	}
	data, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"event":"pi_started"`, `"foreground":true`, `"pid":42`} {
		if !bytes.Contains(data, []byte(want)) {
			t.Fatalf("session log %q missing %q", data, want)
		}
	}
}

func TestParsePiMuseRequiresExactUniqueNonOverlappingArgvSubsequences(t *testing.T) {
	base := validPiProfileTOML("profile", "/bin/echo", 18011, true)
	tests := []string{
		strings.Replace(base, `"--model", "Model", "--draft"`, `"--model", "other", "--draft"`, 1),
		strings.Replace(base, `"--draft", "Draft", "--host"`, `"--draft", "Draft", "--draft", "Draft", "--host"`, 1),
		strings.Replace(base, `target_argv = ["--model", "Model"]`, `target_argv = ["--model", "wrong"]`, 1),
	}
	for i, body := range tests {
		if _, err := parseProjectConfig([]byte(body), "config"); err == nil {
			t.Fatalf("Muse narrowing %d admitted", i)
		}
	}
}

func TestPiProfilesReplaceAtomicallyAcrossProjectConfigs(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)
	parentPath := filepath.Join(parent, ".agents", ".configs", projectConfigFileName)
	childPath := filepath.Join(child, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(parentPath))
	mustMkdir(t, filepath.Dir(childPath))
	mustWrite(t, parentPath, validPiProfileTOML("profile", "/bin/echo", 18011, false))
	mustWrite(t, childPath, `[agents.pi.profiles.profile]
provider = "replacement-only"
`)
	_, err := loadCompositeProjectConfig(ancestorDirsRootFirst(child), "")
	if err == nil {
		t.Fatal("loadCompositeProjectConfig field-merged a partial child Pi profile")
	}
}

func TestPiComposeNearestSelectionCanUseOrReplaceAncestorProfile(t *testing.T) {
	piRoot := officialPiAsset(t)
	parent, home := t.TempDir(), t.TempDir()
	child := filepath.Join(parent, "child")
	mustMkdir(t, child)
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	t.Setenv("HOME", home)
	parentBody := validPiProfileTOML("ancestor-a", "/bin/echo", 18027, false)
	parentBody += "\n" + piProfileSection(t, validPiProfileTOML("ancestor-b", "/bin/echo", 18028, false))
	writePiProjectConfig(t, parent, parentBody)
	childConfig := filepath.Join(child, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(childConfig))
	mustWrite(t, childConfig, `[agents.pi.primary_session]
profile = "ancestor-b"
`)
	lookPath := func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil }
	plan, err := BuildPrimarySessionLaunchPlan("pi", child, home, nil, ChildLaunchCompositionProducer{}, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	canonicalChildConfig, err := filepath.EvalSymlinks(childConfig)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Pi == nil || plan.Pi.LogicalProfile != "ancestor-b" || plan.Resolved.Profile.Source != canonicalChildConfig {
		t.Fatalf("nearest selection did not select complete ancestor profile: %#v", plan)
	}

	replacement := validPiProfileTOML("ancestor-b", "/bin/echo", 18029, false)
	replacement = strings.Replace(replacement, `provider = "local-provider"`, `provider = "child-provider"`, 1)
	replacement = strings.Replace(replacement, `model = "Model"`, `model = "ChildModel"`, 1)
	replacement = strings.ReplaceAll(replacement, `"Model"`, `"ChildModel"`)
	mustWrite(t, childConfig, replacement)
	plan, err = BuildPrimarySessionLaunchPlan("pi", child, home, nil, ChildLaunchCompositionProducer{}, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Resolved.Model.Value == nil || *plan.Resolved.Model.Value != "child-provider/ChildModel" {
		t.Fatalf("child complete profile did not atomically replace ancestor: %#v", plan.Resolved.Model)
	}
}

func TestManagedPiArgvBridgePrecedenceRedactionAndOperandBoundary(t *testing.T) {
	p := mustParsedPiProfile(t, false)
	plan, err := BuildManagedPiArguments([]string{"--provider=local-provider", "--model", "local-provider/Model:high", "--thinking=high", "--api-key=super-secret", "--approve", "--", "ordinary prompt"}, "profile", p)
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := []string{"--provider", "local-provider", "--model", "Model", "--thinking", "high"}
	if !reflect.DeepEqual(plan.Argv[:6], wantPrefix) {
		t.Fatalf("argv prefix=%q want=%q", plan.Argv[:6], wantPrefix)
	}
	if strings.Contains(strings.Join(plan.Argv, "\x00"), "\x00--\x00") {
		t.Fatalf("fake separator forwarded: %q", plan.Argv)
	}
	if strings.Contains(strings.Join(plan.DiagnosticArgv, " "), "super-secret") || !strings.Contains(strings.Join(plan.DiagnosticArgv, " "), "<redacted>") {
		t.Fatalf("secret diagnostic=%q", plan.DiagnosticArgv)
	}
	if plan.Argv[len(plan.Argv)-1] != "ordinary prompt" {
		t.Fatalf("operand bytes not preserved: %q", plan.Argv)
	}
}

func TestManagedPiArgvBridgeRejectsIdentityLookalikesAndUnsafeSuffix(t *testing.T) {
	p := mustParsedPiProfile(t, false)
	tests := [][]string{
		{"--provider", "other"},
		{"--model", "local-provider∕Model"},
		{"--model", "Model:high", "--thinking", "low"},
		{"--api-key="},
		{"--api-key", ""},
		{"--session-dir", "/tmp/global-sessions"},
		{"--session", "/tmp/global-session.jsonl"},
		{"--session", `..\global-session.jsonl`},
		{"--fork", "../global-session.jsonl"},
		{"--fork", `..\global-session`},
		{"--export", "/tmp/global-session.jsonl"},
		{"--", "--provider"},
		{"--", "@file"},
		{"--", "ok", "--"},
		{"--unknown"},
		{"--unknown", "--", "prompt"},
	}
	for _, args := range tests {
		if _, err := BuildManagedPiArguments(args, "profile", p); err == nil {
			t.Fatalf("BuildManagedPiArguments accepted %q", args)
		}
	}
}

func TestManagedPiArgvBridgeKeepsIsolatedSessionModes(t *testing.T) {
	p := mustParsedPiProfile(t, false)
	for _, args := range [][]string{{"--continue"}, {"--resume"}, {"--session", "partial-id"}, {"--fork", "partial-id"}} {
		plan, err := BuildManagedPiArguments(args, "profile", p)
		if err != nil {
			t.Fatalf("BuildManagedPiArguments(%q): %v", args, err)
		}
		if !strings.Contains(strings.Join(plan.Argv, "\x00"), strings.Join(args, "\x00")) {
			t.Fatalf("isolated session mode missing from argv: args=%q argv=%q", args, plan.Argv)
		}
	}
}

func TestPiLaunchProfileStateKeyIsolation(t *testing.T) {
	names := []string{"/", `\`, ".", "..", "../qwen", "nested/../qwen", "/absolute/profile", "unicode∕slash", "unicode∖backslash", "qwen", "QWEN", "é", "e\u0301"}
	project, home := t.TempDir(), t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	t.Setenv("HOME", home)
	configPath := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(configPath))
	body := validPiProfileTOML(names[0], "/bin/echo", 18011, false)
	for _, name := range names[1:] {
		profileBody := validPiProfileTOML(name, "/bin/echo", 18011, false)
		marker := strings.Index(profileBody, "[agents.pi.profiles")
		if marker < 0 {
			t.Fatal("profile fixture marker missing")
		}
		body += "\n" + profileBody[marker:]
	}
	mustWrite(t, configPath, body)
	piRoot := officialPiAsset(t)
	lookPath := func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil }
	seen := map[string]string{}
	var locks []*PiProfileLock
	defer func() {
		for _, lock := range locks {
			_ = lock.Close()
		}
	}()
	for _, name := range names {
		plan, err := BuildPrimarySessionLaunchPlan("pi", project, home, []string{"--profile", name}, ChildLaunchCompositionProducer{}, lookPath)
		if err != nil {
			t.Fatalf("production compose profile %q: %v", name, err)
		}
		if plan.Pi == nil || plan.Pi.State == nil {
			t.Fatalf("production compose omitted state for %q", name)
		}
		paths := *plan.Pi.State
		want := sha256.Sum256([]byte(name))
		wantKey := hex.EncodeToString(want[:])
		if paths.ProfileStateKey != wantKey {
			t.Fatalf("profile %q key=%s want exact UTF-8 %s", name, paths.ProfileStateKey, wantKey)
		}
		if old, ok := seen[paths.ProfileStateKey]; ok {
			t.Fatalf("byte-distinct profiles %q and %q share a key", old, name)
		}
		seen[paths.ProfileStateKey] = name
		rel, err := filepath.Rel(paths.CanonicalCacheRoot, paths.Root)
		rawComponent := false
		for _, component := range splitCleanSuffix(rel) {
			if component == name {
				rawComponent = true
			}
		}
		if err != nil || rawComponent || len(splitCleanSuffix(rel)) != 4 {
			t.Fatalf("raw profile leaked or containment narrowed: name=%q rel=%q err=%v", name, rel, err)
		}
		if err := CreatePiStateTree(paths); err != nil {
			t.Fatal(err)
		}
		lock, err := AcquirePiProfileLock(paths)
		if err != nil {
			t.Fatalf("independent lock %q: %v", name, err)
		}
		locks = append(locks, lock)
		if _, err := AcquirePiProfileLock(paths); piErrorCode(err) != "pi_profile_busy" {
			t.Fatalf("same profile lock was not exclusive for %q: %v", name, err)
		}
	}
	original := piStateKey
	piStateKey = func(string) string { return strings.Repeat("a", 64) }
	defer func() { piStateKey = original }()
	_, err := BuildPrimarySessionLaunchPlan("pi", project, home, []string{"--profile", names[0]}, ChildLaunchCompositionProducer{}, lookPath)
	if piErrorCode(err) != "profile_state_key_collision" {
		t.Fatalf("production compose collision err=%v", err)
	}
}

func TestPiLaunchRefusesInvalidManagedStatePaths(t *testing.T) {
	piRoot := officialPiAsset(t)
	for _, shape := range []string{"missing-cache-root", "symlink-component", "non-directory-component"} {
		t.Run(shape, func(t *testing.T) {
			project, home := t.TempDir(), t.TempDir()
			cache := filepath.Join(t.TempDir(), "cache")
			if shape != "missing-cache-root" {
				mustMkdir(t, cache)
				component := filepath.Join(cache, "agents-infra")
				if shape == "symlink-component" {
					if err := os.Symlink(t.TempDir(), component); err != nil {
						t.Fatal(err)
					}
				} else {
					mustWrite(t, component, "not-a-directory")
				}
			}
			writePiProjectConfig(t, project, validPiProfileTOML("profile", "/bin/echo", 18011, false))
			err := RunPi(RunPiOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Environ: []string{"HOME=" + home}, LookPath: func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil }})
			if piErrorCode(err) != "profile_state_path_invalid" {
				t.Fatalf("production state refusal shape=%s err=%v", shape, err)
			}
		})
	}
}

func TestPiLaunchRefusesStatePostCreateRevalidationFailure(t *testing.T) {
	piRoot := officialPiAsset(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	writePiProjectConfig(t, project, validPiProfileTOML("profile", "/bin/echo", 18025, false))
	original := piRevalidateStateDir
	piRevalidateStateDir = func(int) error { return syscall.EIO }
	defer func() { piRevalidateStateDir = original }()
	err := runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "profile_state_path_invalid" {
		t.Fatalf("production state post-create revalidation err=%v", err)
	}
	canonical, err := CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := ResolvePiStatePaths(cache, canonical, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(paths.Lock); !os.IsNotExist(err) {
		t.Fatalf("post-create revalidation failure acquired lock: %v", err)
	}
}

func TestPiLaunchRefusesHardLinkedSessionLock(t *testing.T) {
	piRoot := officialPiAsset(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	writePiProjectConfig(t, project, validPiProfileTOML("profile", "/bin/echo", 18032, false))
	canonical, err := CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := ResolvePiStatePaths(cache, canonical, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "external-lock")
	mustWrite(t, external, "lock")
	if err := os.Link(external, paths.Lock); err != nil {
		t.Fatal(err)
	}
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "profile_state_path_invalid" {
		t.Fatalf("production hard-linked lock err=%v", err)
	}
}

func TestGeneratedPiCatalogHasFixedNonSecretCredentialSurface(t *testing.T) {
	body, err := GeneratePiModelsJSON(mustParsedPiProfile(t, false))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{"$", "!", "headers", "command", "environment"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated models.json contains forbidden %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, `"apiKey": "agents-infra-local"`) {
		t.Fatalf("missing fixed dummy key: %s", text)
	}
}

func TestWritePiCompactionSettingsPreservesUnrelatedPreferences(t *testing.T) {
	cache := t.TempDir()
	paths, err := ResolvePiStatePaths(cache, "/project", "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	existing := []byte("{\n  \"lastChangelogVersion\": \"0.84.2\",\n  \"theme\": \"dark\",\n  \"compaction\": {\"enabled\": false}\n}\n")
	if err := os.WriteFile(paths.SettingsJSON, existing, 0o600); err != nil {
		t.Fatal(err)
	}
	want := &PiCompaction{Enabled: true, CompactAtTokens: 8192, ReserveTokens: 4096, KeepRecentTokens: 2048}
	if err := WritePiCompactionSettings(paths, want); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(paths.SettingsJSON)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		LastChangelogVersion string `json:"lastChangelogVersion"`
		Theme                string `json:"theme"`
		Compaction           struct {
			Enabled          bool `json:"enabled"`
			ReserveTokens    int  `json:"reserveTokens"`
			KeepRecentTokens int  `json:"keepRecentTokens"`
		} `json:"compaction"`
	}
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	if document.LastChangelogVersion != "0.84.2" || document.Theme != "dark" || !document.Compaction.Enabled || document.Compaction.ReserveTokens != want.ReserveTokens || document.Compaction.KeepRecentTokens != want.KeepRecentTokens || bytes.Contains(content, []byte("compactAtTokens")) {
		t.Fatalf("merged settings=%#v", document)
	}
	if info, err := os.Stat(paths.SettingsJSON); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("settings mode=%v err=%v", info.Mode().Perm(), err)
	}

	malformed := []byte("not-json\n")
	if err := os.WriteFile(paths.SettingsJSON, malformed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WritePiCompactionSettings(paths, want); piErrorCode(err) != "pi_settings_invalid" {
		t.Fatalf("malformed settings error=%v", err)
	}
	unchanged, err := os.ReadFile(paths.SettingsJSON)
	if err != nil || !bytes.Equal(unchanged, malformed) {
		t.Fatalf("malformed settings changed: %q err=%v", unchanged, err)
	}
}

func TestPiMuseDiagnosticsRemainConfiguredUnverified(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	cache := filepath.Join(home, "Library", "Caches")
	mustMkdir(t, cache)
	config := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(config))
	mustWrite(t, config, validPiProfileTOML("profile", "/bin/echo", 18011, true))
	t.Setenv("HOME", home)
	piRoot := officialPiAsset(t)
	plan, err := BuildPrimarySessionLaunchPlan("pi", project, home, nil, ChildLaunchCompositionProducer{}, func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil })
	if err != nil {
		t.Fatal(err)
	}
	if plan.Pi == nil || plan.Pi.DFlash == nil || plan.Pi.DFlash.Status != "configured-unverified" {
		t.Fatalf("DFlash diagnostics=%#v", plan.Pi)
	}
	if !reflect.DeepEqual(plan.Pi.Capabilities.Requested, []string{"dflash", "text", "tools"}) || len(plan.Pi.Capabilities.Verified) != 0 || plan.Pi.Capabilities.Verification != "not-claimed" {
		t.Fatalf("capability diagnostics=%#v", plan.Pi.Capabilities)
	}
	body, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"attestation", "nonce", "backend_catalog", `"verified":["`} {
		if bytes.Contains(body, []byte(forbidden)) {
			t.Fatalf("diagnostics invented authority %q: %s", forbidden, body)
		}
	}
}

func TestPiExecutionEnvironmentRejectsLoaderNamesAndDuplicates(t *testing.T) {
	for _, env := range [][]string{{"PATH=/bin", "PATH=/usr/bin"}, {"DYLD_INSERT_LIBRARIES=x"}, {"NODE_OPTIONS=x"}, {"BUN_CONFIG=x"}, {"LD_PRELOAD=x"}} {
		if err := ValidatePiExecutionEnvironment(env); piErrorCode(err) != "pi_execution_environment_invalid" {
			t.Fatalf("env=%q err=%v", env, err)
		}
	}
}

func TestPiExecutionEnvironmentRejectsLlamaArgsWithoutExposingValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "LLAMA_ARG_MODEL", value: "secret-model-path"},
		{name: "LLAMA_ARG_CTX_SIZE", value: "secret-context-size"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePiExecutionEnvironment([]string{"HOME=/tmp", tc.name + "=" + tc.value})
			if piErrorCode(err) != "pi_execution_environment_invalid" {
				t.Fatalf("environment name %s was admitted: %v", tc.name, err)
			}
			want := fmt.Sprintf("runtime-affecting environment name %q is denied", tc.name)
			if err.Error() != want {
				t.Fatalf("refusal=%q want=%q", err, want)
			}
			if strings.Contains(err.Error(), tc.value) {
				t.Fatalf("refusal exposed environment value: %q", err)
			}
		})
	}
}

func TestPiExecutionEnvironmentRejectsModelOriginOverridesWithoutExposingValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "HF_ENDPOINT", value: "https://unreviewed-hf-origin.invalid"},
		{name: "MODEL_ENDPOINT", value: "https://unreviewed-model-origin.invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePiExecutionEnvironment([]string{"HOME=/tmp", tc.name + "=" + tc.value})
			if piErrorCode(err) != "pi_execution_environment_invalid" {
				t.Fatalf("model-origin environment name %s was admitted: %v", tc.name, err)
			}
			want := fmt.Sprintf("runtime-affecting environment name %q is denied", tc.name)
			if err.Error() != want {
				t.Fatalf("refusal=%q want=%q", err, want)
			}
			if strings.Contains(err.Error(), tc.value) {
				t.Fatalf("refusal exposed model-origin environment value: %q", err)
			}
		})
	}
}

func TestPiExecutionEnvironmentRejectsExactGGMLBackendPathWithoutExposingValue(t *testing.T) {
	const secret = "/unreviewed/backend/secret"
	err := ValidatePiExecutionEnvironment([]string{"HOME=/tmp", "GGML_BACKEND_PATH=" + secret})
	if piErrorCode(err) != "pi_execution_environment_invalid" {
		t.Fatalf("GGML_BACKEND_PATH was admitted: %v", err)
	}
	const want = `runtime-affecting environment name "GGML_BACKEND_PATH" is denied`
	if err.Error() != want {
		t.Fatalf("refusal=%q want=%q", err, want)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("refusal exposed GGML_BACKEND_PATH value: %q", err)
	}
}

func TestPiExecutionEnvironmentRejectsExactLlamaAPIKeyWithoutExposingValue(t *testing.T) {
	const secret = "unreviewed-llama-api-key"
	err := ValidatePiExecutionEnvironment([]string{"HOME=/tmp", "LLAMA_API_KEY=" + secret})
	if piErrorCode(err) != "pi_execution_environment_invalid" {
		t.Fatalf("LLAMA_API_KEY was admitted: %v", err)
	}
	const want = `runtime-affecting environment name "LLAMA_API_KEY" is denied`
	if err.Error() != want {
		t.Fatalf("refusal=%q want=%q", err, want)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("refusal exposed LLAMA_API_KEY value: %q", err)
	}
}

func TestPiExecutionEnvironmentAcceptsExactCleanEnvironment(t *testing.T) {
	clean := []string{
		"HOME=/tmp",
		"PATH=/usr/bin:/bin",
		"TERM=xterm-256color",
		"HF_HOME=/tmp/hf-cache",
		"HUGGINGFACE_HUB_CACHE=/tmp/huggingface-hub-cache",
		"TRANSFORMERS_CACHE=/tmp/transformers-cache",
		"HF_TOKEN=credential-treated-separately",
		"HUGGING_FACE_HUB_TOKEN=credential-treated-separately",
		"LLAMA_API_KEY_SUFFIX=not-the-exact-auth-control",
		"UNRELATED_SERVICE_API_KEY=unrelated-control",
		"hf_endpoint=case-sensitive-lookalike",
		"model_endpoint=case-sensitive-lookalike",
		"ggml_backend_path=case-sensitive-lookalike",
		"GGML_BACKEND_PATH_SUFFIX=not-the-loader-control",
		"GGML_METAL_PATH=unestablished-effect",
	}
	if err := ValidatePiExecutionEnvironment(clean); err != nil {
		t.Fatalf("clean environment %q rejected: %v", clean, err)
	}
}

func TestPiLaunchRejectsLoaderAndInboundPiEnvironmentBeforeState(t *testing.T) {
	piRoot := officialPiAsset(t)
	for name, env := range map[string][]string{
		"duplicate":           {"HOME=/tmp", "PATH=/bin", "PATH=/usr/bin"},
		"loader":              {"HOME=/tmp", "DYLD_INSERT_LIBRARIES=x"},
		"llama model":         {"HOME=/tmp", "LLAMA_ARG_MODEL=secret-model-path"},
		"llama absent option": {"HOME=/tmp", "LLAMA_ARG_CTX_SIZE=secret-context-size"},
		"hf model origin":     {"HOME=/tmp", "HF_ENDPOINT=https://secret-hf-origin.invalid"},
		"model origin":        {"HOME=/tmp", "MODEL_ENDPOINT=https://secret-model-origin.invalid"},
		"ggml backend path":   {"HOME=/tmp", "GGML_BACKEND_PATH=/secret/backend/path"},
		"llama api key":       {"HOME=/tmp", "LLAMA_API_KEY=secret-llama-api-key"},
		"inbound agent dir":   {"HOME=/tmp", "PI_CODING_AGENT_DIR=/tmp/agent"},
		"inbound sessions":    {"HOME=/tmp", "PI_CODING_AGENT_SESSION_DIR=/tmp/sessions"},
	} {
		t.Run(name, func(t *testing.T) {
			project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
			writePiProjectConfig(t, project, validPiProfileTOML("profile", "/bin/echo", 18030, false))
			err := RunPi(RunPiOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Environ: env, LookPath: func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil }})
			if piErrorCode(err) != "pi_execution_environment_invalid" {
				t.Fatalf("production environment shape=%s err=%v", name, err)
			}
			if strings.Contains(err.Error(), "secret-model-path") || strings.Contains(err.Error(), "secret-context-size") || strings.Contains(err.Error(), "secret-hf-origin") || strings.Contains(err.Error(), "secret-model-origin") || strings.Contains(err.Error(), "secret/backend/path") || strings.Contains(err.Error(), "secret-llama-api-key") {
				t.Fatalf("production RunPi refusal exposed an inherited environment value: %q", err)
			}
			entries, readErr := os.ReadDir(cache)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("environment refusal created managed state: %v", entries)
			}
		})
	}
}

func TestPiLaunchCleanEnvironmentReachesRuntimeBackendInitializationAndPreservesGlobalState(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := "/usr/bin/python3"
	if _, err := os.Stat(python); err != nil {
		t.Skipf("python fixture unavailable: %v", err)
	}
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	globalDir := filepath.Join(home, ".pi", "agent")
	mustMkdir(t, globalDir)
	globalSentinels := map[string][]byte{
		"models.json":          []byte("global-models-sentinel\n"),
		"settings.json":        []byte("global-settings-sentinel\n"),
		"auth.json":            []byte("global-auth-sentinel\n"),
		"trusted-folders.json": []byte("global-trust-sentinel\n"),
	}
	for name, sentinel := range globalSentinels {
		if err := os.WriteFile(filepath.Join(globalDir, name), sentinel, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pidFile := filepath.Join(t.TempDir(), "runtime.pid")
	script := filepath.Join(t.TempDir(), "runtime.py")
	scriptBody := `import http.server, json, os, subprocess, sys
port=int(sys.argv[1]); model=sys.argv[2]; pidfile=sys.argv[3]
descendant=subprocess.Popen(["/bin/sleep","60"])
open(pidfile,"w").write(str(os.getpid())+"\n"+str(descendant.pid)+"\n")
class H(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
    body=json.dumps({"object":"list","data":[{"id":model}]}).encode()
    self.send_response(200); self.send_header("Content-Type","application/json"); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`
	if err := os.WriteFile(script, []byte(scriptBody), 0o600); err != nil {
		t.Fatal(err)
	}
	config := compactionPiProfileTOML("profile", python, port)
	config = strings.Replace(config, fmt.Sprintf(`["serve", "--model", "Model", "--host", "127.0.0.1", "--port", "%d"]`, port), fmt.Sprintf(`[%q, %q, "Model", %q, "--host", "127.0.0.1", "--port", %q]`, script, strconv.Itoa(port), pidFile, strconv.Itoa(port)), 1)
	configPath := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(configPath))
	mustWrite(t, configPath, config)
	var stdout, stderr bytes.Buffer
	err = RunPi(RunPiOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Args: []string{"--version"}, Environ: []string{"HOME=" + home, "PATH=/usr/bin:/bin", "HF_TOKEN=credential-treated-separately", "HF_HOME=/tmp/hf-cache", "HUGGINGFACE_HUB_CACHE=/tmp/huggingface-hub-cache", "TRANSFORMERS_CACHE=/tmp/transformers-cache", "LLAMA_API_KEY_SUFFIX=not-the-exact-auth-control", "UNRELATED_SERVICE_API_KEY=unrelated-control", "GGML_METAL_PATH=unestablished-control"}, Stdout: &stdout, Stderr: &stderr, LookPath: func(name string) (string, error) { return filepath.Join(piRoot, "pi"), nil }})
	if err != nil {
		t.Fatalf("production RunPi lifecycle: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	for name, sentinel := range globalSentinels {
		got, err := os.ReadFile(filepath.Join(globalDir, name))
		if err != nil || !bytes.Equal(got, sentinel) {
			t.Fatalf("global Pi state %s changed: bytes=%q err=%v", name, got, err)
		}
	}
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("clean environment did not reach runtime backend initialization: %v", err)
	}
	for _, line := range strings.Fields(string(pidBytes)) {
		pid, _ := strconv.Atoi(line)
		if err := syscall.Kill(pid, syscall.Signal(0)); !errors.Is(err, syscall.ESRCH) {
			t.Fatalf("owned runtime group member survived Pi exit: pid=%d err=%v", pid, err)
		}
	}
	canonicalProject, err := CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := ResolvePiStatePaths(cache, canonicalProject, "profile")
	if err != nil {
		t.Fatal(err)
	}
	localModels, err := os.ReadFile(paths.ModelsJSON)
	if err != nil || !bytes.Contains(localModels, []byte("agents-infra-local")) {
		t.Fatalf("isolated models missing: %v %s", err, localModels)
	}
	settings, err := os.ReadFile(paths.SettingsJSON)
	if err != nil || !bytes.Contains(settings, []byte(`"reserveTokens": 2048`)) || !bytes.Contains(settings, []byte(`"keepRecentTokens": 2048`)) {
		t.Fatalf("isolated compaction settings missing: %v %s", err, settings)
	}
}

func TestPiLaunchSerializesRuntimeOutputFanIn(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	pidFile := filepath.Join(t.TempDir(), "runtime.pid")
	script := filepath.Join(t.TempDir(), "dual-stream-runtime.py")
	scriptBody := `import http.server, json, os, sys, threading
port=int(sys.argv[1]); pidfile=sys.argv[2]
open(pidfile,"w").write(str(os.getpid()))
def emit(fd, marker):
  for _ in range(2000): os.write(fd, marker)
threading.Thread(target=emit,args=(1,b"runtime-stdout\n"),daemon=True).start()
threading.Thread(target=emit,args=(2,b"runtime-stderr\n"),daemon=True).start()
class H(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
    body=json.dumps({"object":"list","data":[{"id":"Model"}]}).encode()
    self.send_response(200); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`
	mustWrite(t, script, scriptBody)
	config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), pidFile}, 2)
	writePiProjectConfig(t, project, config)
	var runtimeOutput bytes.Buffer
	err = RunPi(RunPiOptions{
		ProjectDir: project,
		HomeDir:    home,
		CacheRoot:  cache,
		Args:       []string{"--version"},
		Environ:    []string{"HOME=" + home, "PATH=/usr/bin:/bin"},
		Stderr:     &runtimeOutput,
		LookPath:   func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil },
	})
	if err != nil {
		t.Fatalf("production RunPi dual-stream fan-in: %v", err)
	}
	output := runtimeOutput.String()
	if !strings.Contains(output, "runtime-stdout") || !strings.Contains(output, "runtime-stderr") {
		t.Fatalf("runtime output fan-in lost a stream: stdout=%t stderr=%t", strings.Contains(output, "runtime-stdout"), strings.Contains(output, "runtime-stderr"))
	}
	assertRecordedPIDsGone(t, pidFile)
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchMusePassesExactTargetDraftArgv(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	record := filepath.Join(t.TempDir(), "muse-argv.json")
	pidFile := filepath.Join(t.TempDir(), "muse.pid")
	script := filepath.Join(t.TempDir(), "muse-runtime.py")
	scriptBody := `import http.server, json, os, sys
args=sys.argv
def value(flag): return args[args.index(flag)+1]
port=int(value("--port")); target=value("--model")
open(value("--record"),"w").write(json.dumps(args))
open(value("--pid-file"),"w").write(str(os.getpid()))
class H(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
    body=json.dumps({"object":"list","data":[{"id":target}]}).encode()
    self.send_response(200); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`
	mustWrite(t, script, scriptBody)
	argv := []string{script, "--model", "Model", "--draft", "Draft", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--record", record, "--pid-file", pidFile}
	config := validPiProfileWithArgvMode(t, "profile", python, port, argv, 2, true)
	writePiProjectConfig(t, project, config)
	if err := runPiFixture(project, home, cache, piRoot, nil); err != nil {
		t.Fatalf("production Muse launch: %v", err)
	}
	body, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, argv) {
		t.Fatalf("Muse runtime argv=%q want=%q", got, argv)
	}
	assertRecordedPIDsGone(t, pidFile)
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchRefusesOccupiedListenerBeforeRuntime(t *testing.T) {
	piRoot := officialPiAsset(t)
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	runtime := filepath.Join(t.TempDir(), "runtime-sentinel")
	mustWrite(t, runtime, "#!/bin/sh\necho launched > \"$0.launched\"\n")
	if err := os.Chmod(runtime, 0o755); err != nil {
		t.Fatal(err)
	}
	writePiProjectConfig(t, project, validPiProfileTOML("profile", runtime, port, false))
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "runtime_listener_occupied" {
		t.Fatalf("production occupied listener err=%v", err)
	}
	if _, statErr := os.Stat(runtime + ".launched"); !os.IsNotExist(statErr) {
		t.Fatalf("occupied listener launched runtime: %v", statErr)
	}
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchRefusesIndeterminateListenerCheck(t *testing.T) {
	piRoot := officialPiAsset(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	runtime := filepath.Join(t.TempDir(), "runtime-sentinel")
	mustWrite(t, runtime, "#!/bin/sh\necho launched > \"$0.launched\"\n")
	if err := os.Chmod(runtime, 0o755); err != nil {
		t.Fatal(err)
	}
	writePiProjectConfig(t, project, validPiProfileTOML("profile", runtime, 18023, false))
	original := piListen
	piListen = func(string, string) (net.Listener, error) {
		return nil, &net.OpError{Op: "listen", Net: "tcp4", Err: syscall.EACCES}
	}
	defer func() { piListen = original }()
	err := runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "runtime_listener_check_failed" {
		t.Fatalf("production indeterminate listener err=%v", err)
	}
	if _, statErr := os.Stat(runtime + ".launched"); !os.IsNotExist(statErr) {
		t.Fatalf("indeterminate listener launched runtime: %v", statErr)
	}
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchReadinessRefusesMalformedMismatchAndDeadChild(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	for name, body := range map[string]string{"malformed": `{}`, "mismatch": `{"object":"list","data":[{"id":"model-case-mismatch"}]}`} {
		t.Run(name, func(t *testing.T) {
			project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
			listener, err := netListenLoopback()
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			listener.Close()
			pidFile := filepath.Join(t.TempDir(), "runtime.pid")
			script := writePiReadinessServer(t, false)
			config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), body, pidFile}, 10)
			writePiProjectConfig(t, project, config)
			err = runPiFixture(project, home, cache, piRoot, nil)
			want := "runtime_readiness_invalid"
			if name == "mismatch" {
				want = "runtime_model_unavailable"
			}
			if piErrorCode(err) != want {
				t.Fatalf("err=%v want=%s", err, want)
			}
			assertRecordedPIDsGone(t, pidFile)
			assertPiLockReleased(t, project, cache, "profile")
		})
	}

	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	foreignPID := filepath.Join(t.TempDir(), "foreign.pid")
	serverScript := writePiReadinessServer(t, false)
	launcherScript := filepath.Join(t.TempDir(), "launch-foreign.py")
	launcherBody := `import socket, subprocess, sys, time
p=subprocess.Popen([sys.executable,sys.argv[1],sys.argv[2],sys.argv[3],sys.argv[4]], start_new_session=True)
deadline=time.time()+5
while time.time()<deadline:
  try:
    s=socket.create_connection(("127.0.0.1",int(sys.argv[2])),.1); s.close(); break
  except OSError: time.sleep(.02)
sys.exit(0)
`
	mustWrite(t, launcherScript, launcherBody)
	config := validPiProfileWithArgv(t, "profile", python, port, []string{launcherScript, serverScript, strconv.Itoa(port), `{"object":"list","data":[{"id":"Model"}]}`, foreignPID}, 10)
	writePiProjectConfig(t, project, config)
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "runtime_exited_early" {
		t.Fatalf("production ready foreign listener compensated for dead selected child: %v", err)
	}
	if pidBytes, readErr := os.ReadFile(foreignPID); readErr == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchReadinessRetriesOnlyServiceUnavailableAtProductionEntry(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	for _, tc := range []struct {
		name      string
		status    int
		wantCode  string
		wantCount string
	}{
		{name: "service unavailable then ready", status: http.StatusServiceUnavailable, wantCount: "3"},
		{name: "bad gateway", status: http.StatusBadGateway, wantCode: "runtime_readiness_invalid", wantCount: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
			listener, err := netListenLoopback()
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			listener.Close()
			pidFile := filepath.Join(t.TempDir(), "runtime.pid")
			countFile := filepath.Join(t.TempDir(), "readiness-count")
			script := writePiSequencedReadinessServer(t)
			config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), strconv.Itoa(tc.status), countFile, pidFile}, 10)
			writePiProjectConfig(t, project, config)

			err = runPiFixture(project, home, cache, piRoot, nil)
			if got := piErrorCode(err); got != tc.wantCode {
				t.Fatalf("production readiness error code=%q want=%q err=%v", got, tc.wantCode, err)
			}
			count, readErr := os.ReadFile(countFile)
			if readErr != nil || strings.TrimSpace(string(count)) != tc.wantCount {
				t.Fatalf("readiness request count=%q want=%q err=%v", count, tc.wantCount, readErr)
			}
			assertRecordedPIDsGone(t, pidFile)
			assertPiLockReleased(t, project, cache, "profile")
		})
	}
}

func TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	for _, tc := range []struct {
		name      string
		exitAfter bool
		wantCode  string
	}{
		{name: "times out while runtime remains alive", wantCode: "runtime_readiness_timeout"},
		{name: "refuses after owned runtime exits", exitAfter: true, wantCode: "runtime_exited_early"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
			listener, err := netListenLoopback()
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			listener.Close()
			pidFile := filepath.Join(t.TempDir(), "runtime.pid")
			countFile := filepath.Join(t.TempDir(), "readiness-count")
			script := writePiPersistentUnavailableServer(t, tc.exitAfter)
			config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), countFile, pidFile}, 1)
			writePiProjectConfig(t, project, config)

			err = runPiFixture(project, home, cache, piRoot, nil)
			if got := piErrorCode(err); got != tc.wantCode {
				t.Fatalf("production readiness error code=%q want=%q err=%v", got, tc.wantCode, err)
			}
			count, readErr := os.ReadFile(countFile)
			requests, parseErr := strconv.Atoi(strings.TrimSpace(string(count)))
			if readErr != nil || parseErr != nil || requests < 1 {
				t.Fatalf("runtime bound was reached without polling exact 503: count=%q read=%v parse=%v", count, readErr, parseErr)
			}
			assertRecordedPIDsGone(t, pidFile)
			assertPiLockReleased(t, project, cache, "profile")
		})
	}
}

func TestPiLaunchRuntimeSpawnFailureReleasesLock(t *testing.T) {
	piRoot := officialPiAsset(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtime := filepath.Join(t.TempDir(), "invalid-runtime")
	mustWrite(t, runtime, "not an executable image")
	if err := os.Chmod(runtime, 0o755); err != nil {
		t.Fatal(err)
	}
	writePiProjectConfig(t, project, validPiProfileTOML("profile", runtime, port, false))
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "runtime_start_failed" {
		t.Fatalf("production runtime spawn failure err=%v", err)
	}
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchRejectsAbsentDirectoryAndNonExecutableRuntime(t *testing.T) {
	piRoot := officialPiAsset(t)
	nonExecutable := filepath.Join(t.TempDir(), "runtime")
	mustWrite(t, nonExecutable, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(nonExecutable, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		path string
		code string
	}{
		"absent":         {path: filepath.Join(t.TempDir(), "absent"), code: "runtime_executable_not_found"},
		"directory":      {path: t.TempDir(), code: "runtime_executable_invalid"},
		"non-executable": {path: nonExecutable, code: "runtime_executable_invalid"},
	} {
		t.Run(name, func(t *testing.T) {
			project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
			writePiProjectConfig(t, project, validPiProfileTOML("profile", tc.path, 18026, false))
			err := runPiFixture(project, home, cache, piRoot, nil)
			if piErrorCode(err) != tc.code {
				t.Fatalf("production runtime path shape=%s err=%v", name, err)
			}
			entries, readErr := os.ReadDir(cache)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("runtime path refusal created managed state: %v", entries)
			}
		})
	}
}

func TestPiLaunchRefusesRuntimeExecutableDisappearanceBeforeSpawn(t *testing.T) {
	piRoot := officialPiAsset(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtime := filepath.Join(t.TempDir(), "runtime")
	mustWrite(t, runtime, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(runtime, 0o755); err != nil {
		t.Fatal(err)
	}
	writePiProjectConfig(t, project, validPiProfileTOML("profile", runtime, port, false))
	original := piListen
	piListen = func(network, address string) (net.Listener, error) {
		if err := os.Remove(runtime); err != nil {
			return nil, err
		}
		return original(network, address)
	}
	defer func() { piListen = original }()
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "runtime_executable_invalid" {
		t.Fatalf("production disappearing runtime err=%v", err)
	}
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchPassesRuntimeShellMetacharactersLiterally(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	record := filepath.Join(t.TempDir(), "literal-argv")
	sideEffect := filepath.Join(t.TempDir(), "must-not-exist")
	token := "; touch " + sideEffect
	pidFile := filepath.Join(t.TempDir(), "runtime.pid")
	script := filepath.Join(t.TempDir(), "literal-runtime.py")
	scriptBody := `import http.server, json, os, sys
port=int(sys.argv[1]); record=sys.argv[2]; token=sys.argv[3]; pidfile=sys.argv[4]
open(record,"w").write(token)
open(pidfile,"w").write(str(os.getpid()))
class H(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
    body=json.dumps({"object":"list","data":[{"id":"Model"}]}).encode()
    self.send_response(200); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`
	mustWrite(t, script, scriptBody)
	config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), record, token, pidFile}, 10)
	writePiProjectConfig(t, project, config)
	if err := runPiFixture(project, home, cache, piRoot, nil); err != nil {
		t.Fatalf("literal runtime argv launch: %v", err)
	}
	got, err := os.ReadFile(record)
	if err != nil || string(got) != token {
		t.Fatalf("runtime argv was not literal: got=%q err=%v", got, err)
	}
	if _, err := os.Stat(sideEffect); !os.IsNotExist(err) {
		t.Fatalf("runtime argv was interpreted by a shell: %v", err)
	}
	assertRecordedPIDsGone(t, pidFile)
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchPiSpawnFailureCleansRuntimeAndReleasesLock(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	pidFile := filepath.Join(t.TempDir(), "runtime.pid")
	script := writePiReadinessServer(t, false)
	config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), `{"object":"list","data":[{"id":"Model"}]}`, pidFile}, 10)
	writePiProjectConfig(t, project, config)
	original := piExecCommand
	piExecCommand = func(string, ...string) *exec.Cmd { return exec.Command(filepath.Join(t.TempDir(), "missing-pi")) }
	defer func() { piExecCommand = original }()
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "pi_start_failed" {
		t.Fatalf("production Pi spawn failure err=%v", err)
	}
	assertRecordedPIDsGone(t, pidFile)
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchPointOfUseCatalogMutationCleansRuntime(t *testing.T) {
	piRoot := cloneOfficialPiAsset(t)
	python := requirePython(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	pidFile := filepath.Join(t.TempDir(), "runtime.pid")
	script := filepath.Join(t.TempDir(), "mutating-runtime.py")
	scriptBody := `import http.server, json, os, sys
port=int(sys.argv[1]); mutate=sys.argv[2]; pidfile=sys.argv[3]
open(pidfile,"w").write(str(os.getpid()))
os.chmod(mutate,0o600)
class H(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
    body=json.dumps({"object":"list","data":[{"id":"Model"}]}).encode()
    self.send_response(200); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`
	mustWrite(t, script, scriptBody)
	config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), filepath.Join(piRoot, "CHANGELOG.md"), pidFile}, 10)
	writePiProjectConfig(t, project, config)
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "pi_execution_identity_changed" {
		t.Fatalf("production point-of-use Pi mutation err=%v", err)
	}
	assertRecordedPIDsGone(t, pidFile)
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiLaunchForwardsSignalsThenCleansRuntime(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
			listener, err := netListenLoopback()
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			listener.Close()
			runtimePID := filepath.Join(t.TempDir(), "runtime.pid")
			runtimeScript := writePiReadinessServer(t, false)
			config := validPiProfileWithArgv(t, "profile", python, port, []string{runtimeScript, strconv.Itoa(port), `{"object":"list","data":[{"id":"Model"}]}`, runtimePID}, 10)
			writePiProjectConfig(t, project, config)
			started, received := filepath.Join(t.TempDir(), "pi.started"), filepath.Join(t.TempDir(), "pi.signal")
			captured := make(chan struct {
				path string
				argv []string
			}, 1)
			original := piExecCommand
			piExecCommand = func(path string, argv ...string) *exec.Cmd {
				captured <- struct {
					path string
					argv []string
				}{path: path, argv: append([]string(nil), argv...)}
				return exec.Command(os.Args[0], "-test.run=^TestPiSignalChildProcess$", "--", started, received)
			}
			defer func() { piExecCommand = original }()
			signals := make(chan os.Signal, 1)
			result := make(chan error, 1)
			go func() {
				result <- RunPi(RunPiOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Args: []string{"--version"}, Environ: []string{"HOME=" + home, "PATH=/usr/bin:/bin"}, LookPath: func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil }, Signals: signals})
			}()
			waitForFile(t, started, 5*time.Second)
			signals <- sig
			select {
			case err := <-result:
				if err != nil {
					t.Fatalf("signal launch result: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("signal launch did not exit")
			}
			invocation := <-captured
			if invocation.path != filepath.Join(piRoot, "pi") || !filepath.IsAbs(invocation.path) {
				t.Fatalf("managed Pi spawn path=%q", invocation.path)
			}
			body, err := os.ReadFile(received)
			if err != nil || strings.TrimSpace(string(body)) != strconv.Itoa(int(sig)) {
				t.Fatalf("Pi did not receive %v: body=%q err=%v", sig, body, err)
			}
			assertRecordedPIDsGone(t, runtimePID)
			assertPiLockReleased(t, project, cache, "profile")
		})
	}
}

// TestPiSignalChildProcess is the actual managed Pi child used by the signal
// lifecycle regression. It acknowledges readiness only after signal.Notify is
// installed, removing the Python interpreter/startup scheduling race that made
// the production forwarding gate flaky under the focused race suite.
func TestPiSignalChildProcess(t *testing.T) {
	delimiter := -1
	for i, arg := range os.Args {
		if arg == "--" {
			delimiter = i
			break
		}
	}
	if delimiter < 0 {
		return
	}
	if len(os.Args) != delimiter+3 {
		t.Fatalf("signal child arguments: %q", os.Args)
	}
	started, received := os.Args[delimiter+1], os.Args[delimiter+2]
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	if err := os.WriteFile(started, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	sig := <-signals
	if err := os.WriteFile(received, []byte(strconv.Itoa(int(sig.(syscall.Signal)))), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPiLaunchShutdownTimeoutKillsRuntimeGroupAndReleasesLock(t *testing.T) {
	piRoot := officialPiAsset(t)
	python := requirePython(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	pidFile := filepath.Join(t.TempDir(), "runtime.pid")
	script := writePiReadinessServer(t, true)
	config := validPiProfileWithArgv(t, "profile", python, port, []string{script, strconv.Itoa(port), `{"object":"list","data":[{"id":"Model"}]}`, pidFile}, 2)
	config = strings.Replace(config, "shutdown_timeout_seconds = 2", "shutdown_timeout_seconds = 1", 1)
	writePiProjectConfig(t, project, config)
	err = runPiFixture(project, home, cache, piRoot, nil)
	if piErrorCode(err) != "runtime_shutdown_timeout" {
		t.Fatalf("production shutdown escalation err=%v", err)
	}
	assertRecordedPIDsGone(t, pidFile)
	assertPiLockReleased(t, project, cache, "profile")
}

func TestPiRuntimeReadinessDoesNotFollowRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"Model"}]}`))
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusFound))
	defer redirect.Close()
	child, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	childWait := &piProcessWait{done: make(chan struct{})}
	if err := waitPiRuntimeReady(context.Background(), nil, redirect.URL, "Model", child, childWait, time.Second); piErrorCode(err) != "runtime_readiness_invalid" {
		t.Fatalf("readiness followed redirect away from exact configured URL: %v", err)
	}
}

func TestPiLaunchRejectsCatalogCanonicalizationNarrowing(t *testing.T) {
	source := officialPiAsset(t)
	root := filepath.Join(t.TempDir(), "pi")
	cmd := exec.Command("cp", "-cR", source, root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone official Pi asset: %v\n%s", err, output)
	}
	project, home := t.TempDir(), t.TempDir()
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	t.Setenv("HOME", home)
	writePiProjectConfig(t, project, validPiProfileTOML("profile", "/bin/echo", 18024, false))
	verify := func(wantOK bool) {
		t.Helper()
		_, err := BuildPrimarySessionLaunchPlan("pi", project, home, nil, ChildLaunchCompositionProducer{}, func(string) (string, error) { return filepath.Join(root, "pi"), nil })
		if wantOK && err != nil {
			t.Fatalf("official asset refused: %v", err)
		}
		if !wantOK && err == nil {
			t.Fatal("production BuildPrimarySessionLaunchPlan accepted narrowed catalog")
		}
	}
	verify(true)
	t.Run("extra directory", func(t *testing.T) {
		p := filepath.Join(root, "extra-empty")
		if err := os.Mkdir(p, 0o755); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(p)
		verify(false)
	})
	t.Run("extra file", func(t *testing.T) {
		p := filepath.Join(root, "extra-file")
		mustWrite(t, p, "extra")
		defer os.Remove(p)
		verify(false)
	})
	t.Run("missing non-entrypoint", func(t *testing.T) {
		p := filepath.Join(root, "docs", "models.md")
		backup := filepath.Join(t.TempDir(), "models.md")
		if err := os.Rename(p, backup); err != nil {
			t.Fatal(err)
		}
		defer os.Rename(backup, p)
		verify(false)
	})
	t.Run("non-entrypoint mode", func(t *testing.T) {
		p := filepath.Join(root, "CHANGELOG.md")
		if err := os.Chmod(p, 0o600); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(p, 0o644)
		verify(false)
	})
	t.Run("symlink type", func(t *testing.T) {
		p := filepath.Join(root, "CHANGELOG.md")
		backup := p + ".hold"
		if err := os.Rename(p, backup); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("README.md", p); err != nil {
			t.Fatal(err)
		}
		defer func() { os.Remove(p); os.Rename(backup, p) }()
		verify(false)
	})
	t.Run("hardlink alias", func(t *testing.T) {
		p := filepath.Join(root, "CHANGELOG.md")
		backup := p + ".hold"
		if err := os.Rename(p, backup); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(filepath.Join(root, "README.md"), p); err != nil {
			t.Fatal(err)
		}
		defer func() { os.Remove(p); os.Rename(backup, p) }()
		verify(false)
	})
	t.Run("path case", func(t *testing.T) {
		p := filepath.Join(root, "docs", "models.md")
		mutated := filepath.Join(root, "docs", "MODELS.md")
		if err := os.Rename(p, mutated); err != nil {
			t.Skipf("filesystem cannot express case mutation: %v", err)
		}
		defer os.Rename(mutated, p)
		entries, err := os.ReadDir(filepath.Dir(p))
		if err != nil {
			t.Fatal(err)
		}
		observed := false
		for _, entry := range entries {
			observed = observed || entry.Name() == "MODELS.md"
		}
		if !observed {
			t.Skip("filesystem did not preserve the case-only mutation")
		}
		verify(false)
	})
	t.Run("path normalization", func(t *testing.T) {
		p := filepath.Join(root, "assets", "clankolas.png")
		mutated := filepath.Join(root, "assets", "clankol\u00e1s.png")
		if err := os.Rename(p, mutated); err != nil {
			t.Skipf("filesystem cannot express Unicode path mutation: %v", err)
		}
		defer os.Rename(mutated, p)
		verify(false)
	})
	t.Run("manifest record ordering", func(t *testing.T) {
		original := append([]byte(nil), piCatalogManifest...)
		originalDigest := piCatalogManifestSHA256
		lines := bytes.Split(piCatalogManifest, []byte{'\n'})
		lines[0], lines[1] = lines[1], lines[0]
		piCatalogManifest = bytes.Join(lines, []byte{'\n'})
		sum := sha256.Sum256(piCatalogManifest)
		piCatalogManifestSHA256 = hex.EncodeToString(sum[:])
		defer func() { piCatalogManifest, piCatalogManifestSHA256 = original, originalDigest }()
		verify(false)
	})
	t.Run("manifest record encoding", func(t *testing.T) {
		original := append([]byte(nil), piCatalogManifest...)
		originalDigest := piCatalogManifestSHA256
		piCatalogManifest = bytes.Replace(piCatalogManifest, []byte("  ./"), []byte(" \t./"), 1)
		sum := sha256.Sum256(piCatalogManifest)
		piCatalogManifestSHA256 = hex.EncodeToString(sum[:])
		defer func() { piCatalogManifest, piCatalogManifestSHA256 = original, originalDigest }()
		verify(false)
	})
}

func TestPiComposeRejectsUnsupportedAbsentShebangAndCopiedStandalone(t *testing.T) {
	official := officialPiAsset(t)
	for name, setup := range map[string]func(*testing.T) (string, string){
		"unsupported compatibility": func(t *testing.T) (string, string) {
			body := strings.Replace(validPiProfileTOML("profile", "/bin/echo", 18031, false), PiCompatibilityV0842DarwinARM64, "unsupported-pi", 1)
			return filepath.Join(official, "pi"), body
		},
		"absent": func(t *testing.T) (string, string) {
			return filepath.Join(t.TempDir(), "missing-pi"), validPiProfileTOML("profile", "/bin/echo", 18031, false)
		},
		"npm shebang": func(t *testing.T) (string, string) {
			root := t.TempDir()
			path := filepath.Join(root, "pi")
			mustWrite(t, path, "#!/usr/bin/env node\n")
			return path, validPiProfileTOML("profile", "/bin/echo", 18031, false)
		},
		"copied standalone": func(t *testing.T) (string, string) {
			root := t.TempDir()
			path := filepath.Join(root, "pi")
			input, err := os.ReadFile(filepath.Join(official, "pi"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, input, 0o755); err != nil {
				t.Fatal(err)
			}
			return path, validPiProfileTOML("profile", "/bin/echo", 18031, false)
		},
	} {
		t.Run(name, func(t *testing.T) {
			piPath, body := setup(t)
			project, home := t.TempDir(), t.TempDir()
			mustMkdir(t, filepath.Join(home, "Library", "Caches"))
			t.Setenv("HOME", home)
			writePiProjectConfig(t, project, body)
			_, err := BuildPrimarySessionLaunchPlan("pi", project, home, nil, ChildLaunchCompositionProducer{}, func(string) (string, error) { return piPath, nil })
			if err == nil {
				t.Fatalf("production compose admitted %s Pi identity", name)
			}
			code := piErrorCode(err)
			if !strings.HasPrefix(code, "pi_compatibility_") && !strings.HasPrefix(code, "pi_execution_identity_") {
				t.Fatalf("unexpected Pi identity error for %s: %v", name, err)
			}
		})
	}
}

func mustParsedPiProfile(t *testing.T, dflash bool) PiProfile {
	t.Helper()
	cfg, err := parseProjectConfig([]byte(validPiProfileTOML("profile", "/bin/echo", 18011, dflash)), "config")
	if err != nil {
		t.Fatal(err)
	}
	return cfg.PiProfiles["profile"]
}
func piErrorCode(err error) string {
	var pe *PiLaunchError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}
func officialPiAsset(t *testing.T) string {
	t.Helper()
	repoRoot, _ := filepath.Abs("../../../..")
	candidates := []string{filepath.Join(repoRoot, ".temp", "TASK-260817-2h8hn4", "pi-standalone-darwin-arm64-0.84.2", "pi")}
	if primaryRoot := primaryCheckoutRootFromGitFile(filepath.Join(repoRoot, ".git")); primaryRoot != "" {
		candidates = append(candidates, filepath.Join(primaryRoot, ".temp", "TASK-260817-2h8hn4", "pi-standalone-darwin-arm64-0.84.2", "pi"))
	}
	for _, root := range candidates {
		if _, err := os.Stat(filepath.Join(root, "pi")); err == nil {
			return root
		}
	}
	t.Skipf("official Pi acceptance asset unavailable in %v", candidates)
	return ""
}

func primaryCheckoutRootFromGitFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(data))
	if !strings.HasPrefix(text, "gitdir:") {
		return ""
	}
	gitDir := strings.TrimSpace(strings.TrimPrefix(text, "gitdir:"))
	if gitDir == "" {
		return ""
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(filepath.Dir(path), gitDir)
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Clean(gitDir))))
}

func cloneOfficialPiAsset(t *testing.T) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "pi")
	cmd := exec.Command("cp", "-cR", officialPiAsset(t), destination)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone official Pi asset: %v\n%s", err, output)
	}
	return destination
}

func netListenLoopback() (net.Listener, error) { return net.Listen("tcp4", "127.0.0.1:0") }

func writePiProjectConfig(t *testing.T, project, body string) {
	t.Helper()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(path))
	mustWrite(t, path, body)
}

func validPiProfileWithArgv(t *testing.T, name, runtime string, port int, argv []string, startupTimeout int) string {
	return validPiProfileWithArgvMode(t, name, runtime, port, argv, startupTimeout, false)
}

func validPiProfileWithArgvMode(t *testing.T, name, runtime string, port int, argv []string, startupTimeout int, dflash bool) string {
	t.Helper()
	base := validPiProfileTOML(name, runtime, port, dflash)
	defaultArgv := fmt.Sprintf(`["serve", "--model", "Model", "--host", "127.0.0.1", "--port", "%d"]`, port)
	if dflash {
		defaultArgv = fmt.Sprintf(`["serve", "--model", "Model", "--draft", "Draft", "--host", "127.0.0.1", "--port", "%d"]`, port)
	}
	effectiveArgv := append([]string(nil), argv...)
	containsToken := func(want string) bool {
		for _, token := range effectiveArgv {
			if token == want {
				return true
			}
		}
		return false
	}
	if !containsToken("--host") {
		effectiveArgv = append(effectiveArgv, "--host", "127.0.0.1")
	}
	if !containsToken("--port") {
		effectiveArgv = append(effectiveArgv, "--port", strconv.Itoa(port))
	}
	encoded := make([]string, len(effectiveArgv))
	for i, token := range effectiveArgv {
		encoded[i] = strconv.Quote(token)
	}
	base = strings.Replace(base, defaultArgv, "["+strings.Join(encoded, ", ")+"]", 1)
	base = strings.Replace(base, "startup_timeout_seconds = 5", fmt.Sprintf("startup_timeout_seconds = %d", startupTimeout), 1)
	return base
}

func piProfileSection(t *testing.T, body string) string {
	t.Helper()
	marker := strings.Index(body, "[agents.pi.profiles")
	if marker < 0 {
		t.Fatal("profile fixture marker missing")
	}
	return body[marker:]
}

func requirePython(t *testing.T) string {
	t.Helper()
	python := "/usr/bin/python3"
	if _, err := os.Stat(python); err != nil {
		t.Skipf("python fixture unavailable: %v", err)
	}
	return python
}

func writePiReadinessServer(t *testing.T, ignoreTERM bool) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "readiness-runtime.py")
	ignore := ""
	if ignoreTERM {
		ignore = "signal.signal(signal.SIGTERM, signal.SIG_IGN)"
	}
	body := fmt.Sprintf(`import http.server, os, signal, sys
port=int(sys.argv[1]); response=sys.argv[2].encode(); pidfile=sys.argv[3]
open(pidfile,"w").write(str(os.getpid()))
%s
class H(http.server.BaseHTTPRequestHandler):
  def do_GET(self):
    self.send_response(200); self.send_header("Content-Length",str(len(response))); self.end_headers(); self.wfile.write(response)
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`, ignore)
	mustWrite(t, script, body)
	return script
}

func writePiSequencedReadinessServer(t *testing.T) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "sequenced-readiness-runtime.py")
	body := `import http.server, json, os, sys
port=int(sys.argv[1]); loading_status=int(sys.argv[2]); countfile=sys.argv[3]; pidfile=sys.argv[4]
open(pidfile,"w").write(str(os.getpid()))
class H(http.server.BaseHTTPRequestHandler):
  requests=0
  def do_GET(self):
    H.requests += 1
    open(countfile,"w").write(str(H.requests))
    if H.requests < 3:
      status=loading_status; body=b"loading"
    else:
      status=200; body=json.dumps({"object":"list","data":[{"id":"Model"}]}).encode()
    self.send_response(status); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`
	mustWrite(t, script, body)
	return script
}

func writePiPersistentUnavailableServer(t *testing.T, exitAfter bool) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "persistent-unavailable-runtime.py")
	exit := ""
	if exitAfter {
		exit = "threading.Timer(.02, lambda: os._exit(17)).start()"
	}
	body := fmt.Sprintf(`import http.server, os, sys, threading, time
port=int(sys.argv[1]); countfile=sys.argv[2]; pidfile=sys.argv[3]
open(pidfile,"w").write(str(os.getpid()))
parent=os.getppid()
def watch_parent():
  while os.getppid() == parent: time.sleep(.02)
  os._exit(18)
threading.Thread(target=watch_parent, daemon=True).start()
class H(http.server.BaseHTTPRequestHandler):
  requests=0
  def do_GET(self):
    H.requests += 1
    open(countfile,"w").write(str(H.requests))
    body=b"loading"
    self.send_response(503); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
    %s
  def log_message(self,*args): pass
http.server.HTTPServer(("127.0.0.1",port),H).serve_forever()
`, exit)
	mustWrite(t, script, body)
	return script
}

func runPiFixture(project, home, cache, piRoot string, args []string) error {
	if args == nil {
		args = []string{"--version"}
	}
	return RunPi(RunPiOptions{
		ProjectDir: project,
		HomeDir:    home,
		CacheRoot:  cache,
		Args:       args,
		Environ:    []string{"HOME=" + home, "PATH=/usr/bin:/bin"},
		LookPath:   func(string) (string, error) { return filepath.Join(piRoot, "pi"), nil },
	})
}

func assertRecordedPIDsGone(t *testing.T, pidFile string) {
	t.Helper()
	body, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read runtime pid file: %v", err)
	}
	for _, field := range strings.Fields(string(body)) {
		pid, err := strconv.Atoi(field)
		if err != nil {
			t.Fatalf("invalid recorded pid %q", field)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			err = syscall.Kill(pid, syscall.Signal(0))
			if errors.Is(err, syscall.ESRCH) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("owned process survived cleanup: pid=%d err=%v", pid, err)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func assertPiLockReleased(t *testing.T, project, cache, profile string) {
	t.Helper()
	canonical, err := CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := ResolvePiStatePaths(cache, canonical, profile)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := AcquirePiProfileLock(paths)
	if err != nil {
		t.Fatalf("profile lock was not released: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessGroupCleanupStateReflectsLiveAndReapedGroups(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exec /bin/sleep 60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	reaped := false
	t.Cleanup(func() {
		if reaped {
			return
		}
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})

	if got := processGroupCleanupState(pid, nil); got != "failed" {
		t.Fatalf("live process group cleanup state = %q, want failed", got)
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		t.Fatalf("kill process group: %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("SIGKILLed process unexpectedly exited successfully")
	}
	reaped = true

	if got := processGroupCleanupState(pid, nil); got != "confirmed" {
		t.Fatalf("reaped process group cleanup state = %q, want confirmed", got)
	}
	if got := processGroupCleanupState(pid, errors.New("SIGKILL escalation")); got != "confirmed_after_sigkill" {
		t.Fatalf("reaped escalated process group cleanup state = %q, want confirmed_after_sigkill", got)
	}
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
