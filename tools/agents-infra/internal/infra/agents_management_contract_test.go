package infra

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/skill-agents-management/pkg/agentic"
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
	"github.com/relux-works/skill-agents-management/pkg/inferenceengine"
	"github.com/relux-works/skill-agents-management/pkg/localruntime"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
	localmodels "github.com/relux-works/skill-agents-management/pkg/vendorplugin/vendors/local-models"
)

type fakeManagementStatusReader struct {
	calls int
}

func (r *fakeManagementStatusReader) Status(context.Context, localruntime.StatusQuery) (localruntime.Status, error) {
	r.calls++
	return localruntime.Status{BrokerState: "absent", BrokerSource: localruntime.SourceDetermined}, nil
}

type recordingSanitizedObservationReader struct {
	calls  int
	mutate func(*SanitizedEngineObservation)
	err    error
}

func (r *recordingSanitizedObservationReader) ReadSanitizedEngineObservation(_ context.Context, query vendorplugin.EngineObservationQuery) (SanitizedEngineObservation, error) {
	r.calls++
	if r.err != nil {
		return SanitizedEngineObservation{}, r.err
	}
	now := time.Now()
	observation := SanitizedEngineObservation{
		Contract: SanitizedEngineObservationContract, SchemaVersion: SanitizedEngineObservationSchemaVersion,
		Engine: query.Engine, Runtime: query.Runtime, Model: query.Model, Profile: query.Profile,
		ObservedAt: now.Add(-time.Second), ValidUntil: now.Add(time.Minute), Facts: goodSanitizedEngineFacts(),
	}
	if r.mutate != nil {
		r.mutate(&observation)
	}
	return observation, nil
}

func goodSanitizedEngineFacts() []SanitizedEngineFact {
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
	facts := make([]SanitizedEngineFact, 0, len(inferenceengine.MeasuredFacts()))
	for _, definition := range inferenceengine.MeasuredFacts() {
		if definition.Fact == inferenceengine.FactSSHForwarding {
			facts = append(facts, SanitizedEngineFact{Fact: definition.Fact, Outcome: inferenceengine.OutcomeObservedAbsent})
			continue
		}
		facts = append(facts, SanitizedEngineFact{Fact: definition.Fact, Outcome: inferenceengine.OutcomeObservedValue, Value: values[definition.Fact]})
	}
	return facts
}

func managementGraphFixture(t *testing.T, observation *recordingSanitizedObservationReader, processA string) (PiPluginGraph, *fakeManagementStatusReader) {
	t.Helper()
	profile := "local-qwen"
	cacheBudget := int64(6_442_450_944)
	status := &fakeManagementStatusReader{}
	resolved := ResolvedCanonicalTarget{
		Target:  ProjectTarget{Name: "qwen-infra", Vendor: "qwen", Environment: "pi", Model: "qwen-3.8-27b-mlx-8bit", Profile: &profile},
		Profile: &PiProfile{Provider: "local-qwen", Publisher: "alibaba", Family: "qwen", Model: "qwen-3.8-27b-mlx-8bit", ContextWindow: 4096, CacheBudgetBytes: &cacheBudget},
	}
	graph, err := BuildPiPluginGraph("/repo", resolved, status, observation)
	if err != nil {
		t.Fatalf("BuildPiPluginGraph: %v", err)
	}
	if processA == "" {
		processA = filepath.Join(t.TempDir(), "agents-infra")
		if err := os.WriteFile(processA, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return graph, status
}

// Production call site: vendorplugin.BuildLaunch over the real Pi and
// local-models plugins assembled by BuildPiPluginGraph.
func TestPiPluginGraphBuildLaunchMatchesAcceptedSurfaceAndIdentity(t *testing.T) {
	observations := &recordingSanitizedObservationReader{}
	graph, status := managementGraphFixture(t, observations, "")
	processA := filepath.Join(t.TempDir(), "agents-infra")
	if err := os.WriteFile(processA, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	request := graph.SpawnRequest([]byte("inspect the repository"), "/repo", []string{"PATH=" + filepath.Dir(processA)})
	plan, err := vendorplugin.BuildLaunch(context.Background(), graph.Registry, request, agentic.LaunchModeExec)
	if err != nil {
		t.Fatalf("BuildLaunch: %v", err)
	}
	want := []string{"pi", "spawn", "--profile", graph.Profile, "--prompt", "inspect the repository", "--deadline", "30m", "--result-schema", "1"}
	if !reflect.DeepEqual(plan.Argv, want) || plan.Stdin.Attached || len(plan.Stdin.Bytes) != 0 {
		t.Fatalf("plan = argv %#v stdin %#v, want exact Process-A argv and EOF stdin", plan.Argv, plan.Stdin)
	}
	if filepath.Base(plan.Binary) != "agents-infra" || observations.calls != 1 || status.calls != 1 {
		t.Fatalf("binary=%q observation calls=%d status calls=%d", plan.Binary, observations.calls, status.calls)
	}
	if graph.Runtime == "qwen" || graph.Runtime == "qwen-code" {
		t.Fatalf("qwen-infra aliased shipped qwen runtime: %q", graph.Runtime)
	}
	if caps := managementpi.New(status).Capabilities(); caps.EffortTransport != agentic.EffortTransportNone {
		t.Fatalf("Pi thinking was equated with vendor effort: %q", caps.EffortTransport)
	}
	runtime, err := graph.Registry.ResolveRuntime(graph.Runtime)
	if err != nil {
		t.Fatalf("ResolveRuntime: %v", err)
	}
	models := runtime.Vendor.Models()
	if len(models) != 1 {
		t.Fatalf("generic models = %#v", models)
	}
	model := models[0]
	if model.Publisher != "alibaba" || model.Family != "qwen" || model.CacheBudgetBytes == nil || *model.CacheBudgetBytes != 6_442_450_944 {
		t.Fatalf("generic model facts = %#v", model)
	}
	provenance, err := plan.ConsumerProvenance()
	if err != nil || provenance.Publisher != "alibaba" || provenance.Family != "qwen" {
		t.Fatalf("consumer provenance = %#v, %v", provenance, err)
	}
}

// Production call site: BuildPiPluginGraph. A caller cannot cross-wire a
// target/model/provider coordinate around the already-resolved Pi profile.
func TestBuildPiPluginGraphRefusesResolvedProfileProviderCrossWires(t *testing.T) {
	profileName := "profile"
	provider := "local-qwen"
	base := ResolvedCanonicalTarget{
		Target:            ProjectTarget{Name: "runtime", Vendor: "qwen", Environment: "pi", Model: "model", Profile: &profileName, ProfileProvider: &provider},
		Profile:           &PiProfile{Provider: provider, Publisher: "alibaba", Family: "qwen", Model: "model", ContextWindow: 4096},
		EffectiveProvider: provider,
	}
	tests := []struct {
		name   string
		mutate func(*ResolvedCanonicalTarget)
	}{
		{"model cross-wire", func(r *ResolvedCanonicalTarget) { r.Target.Model = "other" }},
		{"profile provider assertion cross-wire", func(r *ResolvedCanonicalTarget) { other := "other"; r.Target.ProfileProvider = &other }},
		{"effective provider cross-wire", func(r *ResolvedCanonicalTarget) { r.EffectiveProvider = "other" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := base
			profile := *base.Profile
			resolved.Profile = &profile
			test.mutate(&resolved)
			if _, err := BuildPiPluginGraph("/repo", resolved, &fakeManagementStatusReader{}, &recordingSanitizedObservationReader{}); err == nil {
				t.Fatal("cross-wired resolved profile was admitted")
			}
		})
	}
}

// Production call site: BuildPiPluginGraph. A Qwen-shaped model and context
// are not cache-budget evidence; profile absence must remain nil in the
// released generic local-model fact.
func TestBuildPiPluginGraphPreservesMissingCacheBudgetWithoutInference(t *testing.T) {
	profileName := "qwen-3.8-27b-mlx-8bit"
	resolved := ResolvedCanonicalTarget{
		Target: ProjectTarget{
			Name:        "qwen-infra",
			Vendor:      "qwen",
			Environment: "pi",
			Model:       "Qwen3.8-27B-MLX-8bit",
			Profile:     &profileName,
		},
		Profile: &PiProfile{
			Provider:      "local-qwen",
			Publisher:     "alibaba",
			Family:        "qwen",
			Model:         "Qwen3.8-27B-MLX-8bit",
			ContextWindow: 131072,
		},
	}
	graph, err := BuildPiPluginGraph("/repo", resolved, &fakeManagementStatusReader{}, &recordingSanitizedObservationReader{})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := graph.Registry.ResolveRuntime(graph.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	models := runtime.Vendor.Models()
	if len(models) != 1 || models[0].CacheBudgetBytes != nil {
		t.Fatalf("missing profile cache budget was inferred: %#v", models)
	}
}

func TestPiPluginGraphBuildLaunchRefusesForgedObservationBeforePreflight(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*SanitizedEngineObservation)
		want   error
	}{
		{"wrong contract", func(o *SanitizedEngineObservation) { o.Contract = "forged" }, vendorplugin.ErrEngineObservationVersion},
		{"stale", func(o *SanitizedEngineObservation) { o.ValidUntil = time.Now().Add(-time.Second) }, vendorplugin.ErrEngineObservationStale},
		{"identity drift", func(o *SanitizedEngineObservation) { o.Profile = "other" }, vendorplugin.ErrEngineObservationIdentity},
		{"wrong engine", func(o *SanitizedEngineObservation) { o.Engine.ID = "other" }, vendorplugin.ErrEngineObservationIdentity},
		{"malformed missing fact", func(o *SanitizedEngineObservation) { o.Facts = o.Facts[1:] }, inferenceengine.ErrObservationMalformed},
		{"unsupported fact", func(o *SanitizedEngineObservation) {
			o.Facts[0].Outcome = inferenceengine.OutcomeNotObserved
			o.Facts[0].Cause = inferenceengine.NotObservedUnsupported
		}, inferenceengine.ErrObservationUnsupported},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observations := &recordingSanitizedObservationReader{mutate: test.mutate}
			graph, status := managementGraphFixture(t, observations, "")
			pathDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(pathDir, "agents-infra"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			_, err := vendorplugin.BuildLaunch(context.Background(), graph.Registry, graph.SpawnRequest([]byte("prompt"), "/repo", []string{"PATH=" + pathDir}), agentic.LaunchModeExec)
			if !errors.Is(err, test.want) {
				t.Fatalf("BuildLaunch error = %v, want %v", err, test.want)
			}
			if observations.calls != 1 || status.calls != 0 {
				t.Fatalf("observation calls=%d preflight calls=%d, want 1/0", observations.calls, status.calls)
			}
		})
	}
}

func TestPiPluginGraphRefusesMissingEvidenceAndCallerProfileConflict(t *testing.T) {
	profile := "local-qwen"
	resolved := ResolvedCanonicalTarget{
		Target:  ProjectTarget{Name: "qwen-infra", Vendor: "qwen", Environment: "pi", Model: "model", Profile: &profile},
		Profile: &PiProfile{Provider: "local", Model: "model", ContextWindow: 4096},
	}
	if _, err := BuildPiPluginGraph("/repo", resolved, &fakeManagementStatusReader{}, nil); err == nil {
		t.Fatal("missing sanitized observation source was admitted")
	}
	observations := &recordingSanitizedObservationReader{}
	graph, status := managementGraphFixture(t, observations, "")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agents-infra"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	request := graph.SpawnRequest([]byte("prompt"), "/repo", []string{"PATH=" + dir})
	request.Profile = "caller-forged-profile"
	_, err := vendorplugin.BuildLaunch(context.Background(), graph.Registry, request, agentic.LaunchModeExec)
	if !errors.Is(err, localmodels.ErrProfileConflict) || observations.calls != 0 || status.calls != 0 {
		t.Fatalf("profile conflict err=%v observation=%d preflight=%d", err, observations.calls, status.calls)
	}
}

func TestProcessAConsumerUsesSoleClassifierAtRealChildLaunch(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "fake-process-b-observed")
	processA := filepath.Join(dir, "agents-infra")
	script := `#!/bin/sh
: > "$FAKE_PROCESS_B_MARKER"
printf '%s\n' '{"contract":"agents-infra.pi-turn-result","schema_version":1,"status":"ok","final_text":"accepted"}'
`
	if err := os.WriteFile(processA, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	observations := &recordingSanitizedObservationReader{}
	graph, _ := managementGraphFixture(t, observations, processA)
	request := graph.SpawnRequest([]byte("prompt"), dir, []string{"PATH=" + dir, "FAKE_PROCESS_B_MARKER=" + marker})
	result, err := BuildAndRunPiTurn(context.Background(), graph.Registry, request, OSProcessATurnRunner{})
	if err != nil {
		t.Fatalf("BuildAndRunPiTurn: %v", err)
	}
	if result.Class != managementpi.TurnResultOK || result.FinalText != "accepted" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("fake Process-B lifecycle was not composed through Process A: %v", err)
	}
}

func TestProcessAConsumerCannotParseAroundSoleClassifier(t *testing.T) {
	source, err := os.ReadFile("agents_management_process_a.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if strings.Count(text, "managementpi.ValidateTurnResult(input)") != 1 {
		t.Fatalf("production consumer must call the sole classifier exactly once")
	}
	for _, forbidden := range []string{"encoding/json", "json.Unmarshal", "json.NewDecoder"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("production consumer contains permissive parser %q", forbidden)
		}
	}
}
