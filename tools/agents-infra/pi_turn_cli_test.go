//go:build !windows

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
	"github.com/relux-works/skill-agents-management/pkg/agentic"
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
	"github.com/relux-works/skill-agents-management/pkg/inferenceengine"
	"github.com/relux-works/skill-agents-management/pkg/localruntime"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
)

// runPiTurnCLI is the real production consumer/parent entry point wired at
// `agents-infra pi turn`. These cases only exercise argument validation,
// which must refuse before infra.ResolvePiPluginGraph or any launch effect.
func TestRunPiTurnCLIRefusesInvalidArgumentsBeforeAnyGraphResolution(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"--bogus"}},
		{"positional argument", []string{"--target", "qwen-infra", "--prompt", "hi", "extra"}},
		{"zero deadline", []string{"--target", "qwen-infra", "--prompt", "hi", "--deadline", "0s"}},
		{"negative deadline", []string{"--target", "qwen-infra", "--prompt", "hi", "--deadline", "-1m"}},
		{"deadline over the 30m ceiling", []string{"--target", "qwen-infra", "--prompt", "hi", "--deadline", "31m"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := runPiTurnCLI(test.args); err == nil {
				t.Fatal("invalid pi turn arguments were admitted")
			}
		})
	}
}

// The top-level `pi` dispatcher must route "turn" to the real production
// consumer entry point rather than to standalone Process-A spawn handling.
func TestRunPiDispatchesTurnSubcommandToPiTurnCLI(t *testing.T) {
	err := runPi([]string{"turn", "--deadline", "0s"})
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("runPi([turn ...]) = %v, want the pi turn deadline refusal", err)
	}
}

// piTurnFakeStatusReader stands in for localruntime.CLIStatusReader. It never
// consults a broker, socket, or process.
type piTurnFakeStatusReader struct {
	mu    sync.Mutex
	calls int
}

func (r *piTurnFakeStatusReader) Status(context.Context, localruntime.StatusQuery) (localruntime.Status, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return localruntime.Status{BrokerState: "absent", BrokerSource: localruntime.SourceDetermined}, nil
}

// piTurnFakeObservationReader stands in for the shared-runtime sanitized
// engine observation reader with a complete, in-window observation.
type piTurnFakeObservationReader struct {
	mu    sync.Mutex
	calls int
}

func (r *piTurnFakeObservationReader) ReadSanitizedEngineObservation(_ context.Context, query vendorplugin.EngineObservationQuery) (infra.SanitizedEngineObservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	now := time.Now()
	return infra.SanitizedEngineObservation{
		Contract: infra.SanitizedEngineObservationContract, SchemaVersion: infra.SanitizedEngineObservationSchemaVersion,
		Engine: query.Engine, Runtime: query.Runtime, Model: query.Model, Profile: query.Profile,
		ObservedAt: now.Add(-time.Second), ValidUntil: now.Add(time.Minute), Facts: piTurnGoodEngineFacts(),
	}, nil
}

func piTurnGoodEngineFacts() []infra.SanitizedEngineFact {
	values := map[inferenceengine.Fact]string{
		inferenceengine.FactContextArgv:              `["--ctx-size","4096"]`,
		inferenceengine.FactPrefillArgv:              `["--ubatch-size","512"]`,
		inferenceengine.FactReasoningStreamField:     "delta.reasoning_content",
		inferenceengine.FactHealth:                   `{"endpoint":"/health","healthy":true}`,
		inferenceengine.FactReadiness:                `{"weights_resident":true}`,
		inferenceengine.FactWeightArtifact:           `{"format":"safetensors","model_path":"/models/weights.safetensors","config_path":"/models/config.json"}`,
		inferenceengine.FactMemoryAccounting:         `{"method":"resident-bytes","bytes":4096,"includes_mapped_weights":true}`,
		inferenceengine.FactSpeculativeDecoding:      `{"capable":true,"active":false}`,
		inferenceengine.FactLoadState:                `{"state":"loaded","weights_resident":true}`,
		inferenceengine.FactUnloadState:              `{"state":"unloaded","weights_resident":false}`,
		inferenceengine.FactInferenceBusy:            `{"busy":false}`,
		inferenceengine.FactMemoryPressureSequence:   `{"pressure":"normal","consulted":["load-state","unload-state","inference-busy"],"action":"none"}`,
		inferenceengine.FactLocalExecutable:          "/opt/agents-infra/bin/mlx-server",
		inferenceengine.FactLocalArgv:                `["--port","8080"]`,
		inferenceengine.FactStressPolicy:             `{"enabled":true,"max_concurrency":2}`,
		inferenceengine.FactRestartSupervisionPolicy: `{"max_attempts":3,"initial_backoff_ms":100,"max_backoff_ms":1000}`,
	}
	facts := make([]infra.SanitizedEngineFact, 0, len(inferenceengine.MeasuredFacts()))
	for _, definition := range inferenceengine.MeasuredFacts() {
		if definition.Fact == inferenceengine.FactSSHForwarding {
			facts = append(facts, infra.SanitizedEngineFact{Fact: definition.Fact, Outcome: inferenceengine.OutcomeObservedAbsent})
			continue
		}
		facts = append(facts, infra.SanitizedEngineFact{Fact: definition.Fact, Outcome: inferenceengine.OutcomeObservedValue, Value: values[definition.Fact]})
	}
	return facts
}

// piTurnFakeProcessA is the fake Process-A boundary: it records the exact
// plan vendorplugin.BuildLaunch produced and answers with a schema-1 ok
// document. No child process, runtime, or model is ever started.
type piTurnFakeProcessA struct {
	mu    sync.Mutex
	calls int
	plans []agentic.Plan
}

func (r *piTurnFakeProcessA) RunProcessATurn(_ context.Context, plan agentic.Plan) (managementpi.TurnResultInput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.plans = append(r.plans, plan)
	return managementpi.TurnResultInput{
		Stdout:       []byte(`{"contract":"agents-infra.pi-turn-result","schema_version":1,"status":"ok","final_text":"accepted"}`),
		Exit:         managementpi.ProcessAExit{Code: 0},
		Intervention: managementpi.TurnInterventionNone,
		Cleanup:      managementpi.ProcessACleanupNotRequired,
	}, nil
}

type piTurnFixture struct {
	project      string
	home         string
	targetConfig string
	status       *piTurnFakeStatusReader
	observations *piTurnFakeObservationReader
	processA     *piTurnFakeProcessA
	stdout       *bytes.Buffer
	env          []string
}

// piTurnCanonicalTargets is the canonical-only shape this bug is about: the
// only valid Qwen target is reachable through the explicit qwen-infra alias
// and nothing else. A second, unmapped Pi target makes "guess the Pi target"
// ambiguous, and a codex alias makes a wrong-environment selection possible.
// The alias vocabulary itself is closed, so ambiguity lives in targets.
const piTurnCanonicalTargets = `[agents.targets.qwen-mlx]
vendor = "qwen"
environment = "pi"
model = "Model"
reasoning = "medium"
profile = "profile"
profile_provider = "local-provider"
endpoint = "http://127.0.0.1:18011/v1"

[agents.targets.qwen-alt]
vendor = "qwen"
environment = "pi"
model = "Model"
reasoning = "medium"
profile = "profile"

[agents.targets.openai-sol]
vendor = "openai"
environment = "codex"
model = "gpt-5.6-sol"
reasoning = "high"

[agents.entrypoints]
qwen-infra = "qwen-mlx"
openai-infra = "openai-sol"
`

func newPiTurnFixture(t *testing.T) piTurnFixture {
	t.Helper()
	home, root := t.TempDir(), t.TempDir()
	project := filepath.Join(root, "nested")
	mustMkdir(t, filepath.Join(home, "Library", "Caches"))
	mustMkdir(t, project)
	profileBody := mainTestPiConfig("/bin/echo", 18011)
	profileBody = strings.Replace(profileBody, `reasoning = false`, `reasoning = true`, 1)
	profileBody = strings.Replace(profileBody, `thinking = "off"`, `thinking = "medium"`, 1)
	profileBody = strings.Replace(profileBody, `supports_developer_role = false`, "supports_developer_role = false\nsupports_reasoning_effort = false\nthinking_format = \"qwen-chat-template\"", 1)
	writeMainCanonicalConfig(t, root, profileBody)
	targetConfig := writeMainCanonicalConfig(t, project, piTurnCanonicalTargets)
	canonicalTargetConfig, err := filepath.EvalSymlinks(targetConfig)
	if err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	mustWrite(t, filepath.Join(binDir, "agents-infra"), "#!/bin/sh\necho 'fake process A must never be executed' >&2\nexit 99\n")
	if err := os.Chmod(filepath.Join(binDir, "agents-infra"), 0o755); err != nil {
		t.Fatal(err)
	}
	return piTurnFixture{
		project:      project,
		home:         home,
		targetConfig: canonicalTargetConfig,
		status:       &piTurnFakeStatusReader{},
		observations: &piTurnFakeObservationReader{},
		processA:     &piTurnFakeProcessA{},
		stdout:       &bytes.Buffer{},
		env:          []string{"PATH=" + binDir, "HOME=" + home},
	}
}

func (f piTurnFixture) deps() piTurnDependencies {
	return piTurnDependencies{
		startDir:     f.project,
		homeDir:      f.home,
		environ:      f.env,
		status:       f.status,
		observations: f.observations,
		runner:       f.processA,
		stdout:       f.stdout,
	}
}

func (f piTurnFixture) assertNoChildOrModelEffect(t *testing.T) {
	t.Helper()
	if f.processA.calls != 0 {
		t.Fatalf("fake Process A was reached %d times before the refusal", f.processA.calls)
	}
	if f.status.calls != 0 || f.observations.calls != 0 {
		t.Fatalf("status calls=%d observation calls=%d, want no runtime or engine read before the refusal", f.status.calls, f.observations.calls)
	}
	if f.stdout.Len() != 0 {
		t.Fatalf("refusal wrote a turn result: %q", f.stdout.String())
	}
}

// Reproduction of the reported bug, preserved at the exact production resolver
// call site: ResolvePiPluginGraph with an empty entrypoint in a canonical-only
// project whose valid Qwen target is mapped only as qwen-infra must refuse
// unknown_entrypoint before BuildLaunch, Process A, or any reader is touched.
// The unique configured Pi-capable mapping must not be inferred.
func TestResolvePiPluginGraphRefusesEmptyEntrypointBeforeAnySideEffect(t *testing.T) {
	fixture := newPiTurnFixture(t)
	t.Setenv("HOME", fixture.home)
	_, err := infra.ResolvePiPluginGraph(fixture.project, fixture.home, "", fixture.status, fixture.observations)
	var targetErr *infra.CanonicalTargetError
	if !errors.As(err, &targetErr) || targetErr.Code != infra.PrimarySessionErrorUnknownEntrypoint {
		t.Fatalf("ResolvePiPluginGraph(empty entrypoint) = %v, want unknown_entrypoint", err)
	}
	if fixture.status.calls != 0 || fixture.observations.calls != 0 {
		t.Fatalf("empty entrypoint reached readers: status=%d observations=%d", fixture.status.calls, fixture.observations.calls)
	}
}

// Production call site: runPi -> runPiTurnCLI -> runPiTurn ->
// infra.ResolvePiPluginGraph -> infra.BuildAndRunPiTurn -> vendorplugin.BuildLaunch.
// A direct pi turn under canonical-only configuration must select the explicit
// qwen-infra target, reach the generic BuildLaunch/Process-A boundary with the
// exact profile, and keep target/profile/provider/model/endpoint provenance.
func TestRunPiTurnCanonicalOnlySelectsExplicitTargetAndReachesFakeProcessA(t *testing.T) {
	fixture := newPiTurnFixture(t)
	t.Setenv("HOME", fixture.home)
	err := runPiTurn([]string{"--target", "qwen-infra", "--prompt", "inspect the repository", "--deadline", "2m"}, fixture.deps())
	if err != nil {
		t.Fatalf("runPiTurn: %v", err)
	}
	if fixture.processA.calls != 1 {
		t.Fatalf("fake Process A calls = %d, want exactly one BuildLaunch/Process-A boundary crossing", fixture.processA.calls)
	}
	plan := fixture.processA.plans[0]
	wantArgv := []string{"pi", "spawn", "--profile", "profile", "--prompt", "inspect the repository", "--deadline", "30m", "--result-schema", "1"}
	if strings.Join(plan.Argv, "\x00") != strings.Join(wantArgv, "\x00") {
		t.Fatalf("Process-A argv = %#v, want %#v", plan.Argv, wantArgv)
	}
	if filepath.Base(plan.Binary) != "agents-infra" || plan.WorkDir != fixture.project {
		t.Fatalf("plan binary=%q workdir=%q", plan.Binary, plan.WorkDir)
	}
	if fixture.status.calls != 1 || fixture.observations.calls != 1 {
		t.Fatalf("status calls=%d observation calls=%d, want one preflight and one sanitized observation", fixture.status.calls, fixture.observations.calls)
	}
	var result managementpi.TurnResult
	decodeSingleJSONDocument(t, fixture.stdout.String(), &result)
	if result.Class != managementpi.TurnResultOK || result.FinalText != "accepted" {
		t.Fatalf("turn result = %#v, want ok/accepted", result)
	}

	graph, err := infra.ResolvePiPluginGraph(fixture.project, fixture.home, "qwen-infra", fixture.status, fixture.observations)
	if err != nil {
		t.Fatalf("ResolvePiPluginGraph: %v", err)
	}
	profileConfig, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(fixture.project), ".agents", ".configs", "project-config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	want := infra.PiPluginGraphProvenance{
		Entrypoint: "qwen-infra", EntrypointSource: fixture.targetConfig,
		Target: "qwen-mlx", TargetSource: fixture.targetConfig,
		Vendor: "qwen", Environment: "pi", Model: "Model",
		Profile: "profile", ProfileSource: profileConfig,
		Provider: "local-provider", Endpoint: "http://127.0.0.1:18011/v1",
	}
	if graph.Provenance != want {
		t.Fatalf("provenance = %#v, want %#v", graph.Provenance, want)
	}
	if graph.Profile != "profile" || string(graph.Model) != "Model" || graph.Runtime == "qwen" || graph.Runtime == "qwen-code" {
		t.Fatalf("graph identity = profile %q model %q runtime %q", graph.Profile, graph.Model, graph.Runtime)
	}
}

// Negative matrix for the target-selection gate at the production call site
// runPiTurn. Each selection must be refused before ResolvePiPluginGraph reads a
// runtime or engine surface and before BuildLaunch or Process A runs, and no
// refusal may fall back to the legacy provider policy, a unique configured Pi
// target, or the last of several conflicting selections.
func TestRunPiTurnRefusesMissingAmbiguousWrongEnvironmentAndConflictingTargets(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode string
		wantText string
	}{
		{"missing selection", []string{"--prompt", "hi", "--deadline", "2m"}, "", "requires an explicit --target"},
		{"empty selection", []string{"--target", "", "--prompt", "hi", "--deadline", "2m"}, "", "requires an explicit --target"},
		{"whitespace selection", []string{"--target", " ", "--prompt", "hi", "--deadline", "2m"}, infra.PrimarySessionErrorUnknownEntrypoint, ""},
		{"ambiguous target name instead of alias", []string{"--target", "qwen-mlx", "--prompt", "hi", "--deadline", "2m"}, infra.PrimarySessionErrorUnknownEntrypoint, ""},
		{"ambiguous unmapped pi target", []string{"--target", "qwen-alt", "--prompt", "hi", "--deadline", "2m"}, infra.PrimarySessionErrorUnknownEntrypoint, ""},
		{"unknown alias", []string{"--target", "nope-infra", "--prompt", "hi", "--deadline", "2m"}, infra.PrimarySessionErrorUnknownEntrypoint, ""},
		{"wrong environment", []string{"--target", "openai-infra", "--prompt", "hi", "--deadline", "2m"}, infra.PrimarySessionErrorInvalidTarget, ""},
		{"conflicting selections", []string{"--target", "qwen-infra", "--target", "openai-infra", "--prompt", "hi", "--deadline", "2m"}, "", "conflicting selections are refused"},
		{"conflicting equal selections", []string{"--target=qwen-infra", "--target", "qwen-infra", "--prompt", "hi", "--deadline", "2m"}, "", "conflicting selections are refused"},
		{"conflicting with last valid", []string{"--target", "openai-infra", "--target", "qwen-infra", "--prompt", "hi", "--deadline", "2m"}, "", "conflicting selections are refused"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPiTurnFixture(t)
			t.Setenv("HOME", fixture.home)
			err := runPiTurn(test.args, fixture.deps())
			if err == nil {
				t.Fatal("target selection was admitted")
			}
			if test.wantCode != "" {
				var targetErr *infra.CanonicalTargetError
				if !errors.As(err, &targetErr) || targetErr.Code != test.wantCode {
					t.Fatalf("error = %v, want canonical target code %s", err, test.wantCode)
				}
			}
			if test.wantText != "" && !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("error = %v, want %q", err, test.wantText)
			}
			fixture.assertNoChildOrModelEffect(t)
		})
	}
}

// The wrong-environment refusal must name the offending target field and its
// source so the operator can fix configuration rather than guess, and it must
// never be silently rewritten into the unique Pi target.
func TestRunPiTurnWrongEnvironmentRefusalCarriesProvenance(t *testing.T) {
	fixture := newPiTurnFixture(t)
	t.Setenv("HOME", fixture.home)
	err := runPiTurn([]string{"--target", "openai-infra", "--prompt", "hi", "--deadline", "2m"}, fixture.deps())
	var targetErr *infra.CanonicalTargetError
	if !errors.As(err, &targetErr) {
		t.Fatalf("error = %v, want CanonicalTargetError", err)
	}
	want := infra.TargetErrorContext{Entrypoint: "openai-infra", Target: "openai-sol", Field: "agents.targets.openai-sol.environment", Source: fixture.targetConfig}
	if targetErr.Context != want {
		t.Fatalf("context = %#v, want %#v", targetErr.Context, want)
	}
	fixture.assertNoChildOrModelEffect(t)
}

// The explicit alias, not target uniqueness, is what selects: with two Pi
// targets configured the mapped alias still resolves to exactly the mapped
// target, and the unmapped Pi target is unreachable by any spelling.
func TestRunPiTurnSelectsOnlyTheMappedTargetWhenSeveralPiTargetsExist(t *testing.T) {
	fixture := newPiTurnFixture(t)
	t.Setenv("HOME", fixture.home)
	if err := runPiTurn([]string{"--target", "qwen-infra", "--prompt", "hi", "--deadline", "2m"}, fixture.deps()); err != nil {
		t.Fatalf("runPiTurn(qwen-infra): %v", err)
	}
	graph, err := infra.ResolvePiPluginGraph(fixture.project, fixture.home, "qwen-infra", fixture.status, fixture.observations)
	if err != nil {
		t.Fatalf("ResolvePiPluginGraph: %v", err)
	}
	if graph.Provenance.Target != "qwen-mlx" || graph.Provenance.Entrypoint != "qwen-infra" {
		t.Fatalf("provenance = %#v, want the mapped qwen-mlx target", graph.Provenance)
	}
	if fixture.processA.calls != 1 {
		t.Fatalf("fake Process A calls = %d", fixture.processA.calls)
	}
}
