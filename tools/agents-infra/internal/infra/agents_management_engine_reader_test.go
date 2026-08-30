package infra

import (
	"testing"
	"time"

	"github.com/relux-works/skill-agents-management/pkg/inferenceengine"
)

func goodSharedRuntimeStatusFixture() SharedRuntimeStatus {
	bytesValue := uint64(4096)
	return SharedRuntimeStatus{
		Broker: SharedRuntimeBrokerStatus{State: "serving"},
		Resources: SharedRuntimeResourceStatus{
			State:             SharedRuntimeResourceHealthy,
			LoadedModelMemory: SharedRuntimeLoadedModelMemoryStatus{State: SharedRuntimeMemoryHealthy, Bytes: &bytesValue},
			Inference:         SharedRuntimeInferenceStatus{State: SharedRuntimeInferenceIdle},
		},
	}
}

func goodSharedRuntimeProfileFixture() PiProfile {
	return PiProfile{
		Reasoning: true,
		Runtime: PiRuntime{
			Executable:    "/opt/agents-infra/bin/mlx-server",
			Argv:          []string{"--model", "/models/weights", "--ctx-size", "4096"},
			ReadinessPath: "/models",
			Sharing: &PiRuntimeSharing{
				Mode: "shared", MaxLeases: 4,
				RestartLimit: 3, RestartInitialBackoffSeconds: 1, RestartMaxBackoffSeconds: 4,
			},
		},
	}
}

// sanitizedEngineFactsFromSharedRuntimeStatus is the pure mapping the real
// SharedRuntimeSanitizedEngineObservationReader delegates to after a live
// status read. This proves the mapping itself satisfies the exact upstream
// contract, independent of transport.
func TestSharedRuntimeEngineFactsSatisfyUpstreamContract(t *testing.T) {
	facts, err := sanitizedEngineFactsFromSharedRuntimeStatus(goodSharedRuntimeStatusFixture(), goodSharedRuntimeProfileFixture())
	if err != nil {
		t.Fatalf("sanitizedEngineFactsFromSharedRuntimeStatus: %v", err)
	}
	if len(facts) != len(inferenceengine.MeasuredFacts()) {
		t.Fatalf("fact count = %d, want %d", len(facts), len(inferenceengine.MeasuredFacts()))
	}
	readings := make([]inferenceengine.Reading, 0, len(facts))
	for _, fact := range facts {
		var outcome inferenceengine.ReadOutcome
		switch fact.Outcome {
		case inferenceengine.OutcomeObservedValue:
			outcome = inferenceengine.ReadValue(fact.Value)
		case inferenceengine.OutcomeObservedAbsent:
			outcome = inferenceengine.ReadAbsent()
		default:
			outcome = inferenceengine.ReadFailure(fact.Cause, fact.Detail)
		}
		readings = append(readings, inferenceengine.NewReading(fact.Fact, outcome))
	}
	if _, err := inferenceengine.ValidateReadings("mlx", inferenceengine.EngineKindNativeTransformer, readings); err != nil {
		t.Fatalf("ValidateReadings rejected the real mapping: %v", err)
	}
}

// Negative: a profile that never requests reasoning must not fabricate a
// reasoning-stream-field value.
func TestSharedRuntimeEngineFactsReportReasoningStreamFieldAbsentWhenNotRequested(t *testing.T) {
	profile := goodSharedRuntimeProfileFixture()
	profile.Reasoning = false
	facts, err := sanitizedEngineFactsFromSharedRuntimeStatus(goodSharedRuntimeStatusFixture(), profile)
	if err != nil {
		t.Fatalf("sanitizedEngineFactsFromSharedRuntimeStatus: %v", err)
	}
	for _, fact := range facts {
		if fact.Fact == inferenceengine.FactReasoningStreamField {
			if fact.Outcome != inferenceengine.OutcomeObservedAbsent {
				t.Fatalf("reasoning-stream-field = %#v, want observed-absent", fact)
			}
			return
		}
	}
	t.Fatal("reasoning-stream-field fact missing")
}

// Negative: memory accounting can only be an ObservedValue when the live
// broker actually measured resident bytes. A disabled/unknown resource
// policy must refuse the whole observation rather than invent a byte count.
func TestSharedRuntimeEngineFactsRefuseWhenMemoryIsNotMeasured(t *testing.T) {
	report := goodSharedRuntimeStatusFixture()
	report.Resources.LoadedModelMemory = SharedRuntimeLoadedModelMemoryStatus{State: SharedRuntimeMemoryUnknown, Reason: "resource_pressure_disabled"}
	if _, err := sanitizedEngineFactsFromSharedRuntimeStatus(report, goodSharedRuntimeProfileFixture()); err == nil {
		t.Fatal("unmeasured loaded-model memory was admitted as an observed fact")
	}
}

// Negative: unknown inference activity must refuse rather than guess busy or
// idle.
func TestSharedRuntimeEngineFactsRefuseWhenInferenceActivityIsNotMeasured(t *testing.T) {
	report := goodSharedRuntimeStatusFixture()
	report.Resources.Inference = SharedRuntimeInferenceStatus{State: SharedRuntimeInferenceUnknown, Reason: "unavailable"}
	if _, err := sanitizedEngineFactsFromSharedRuntimeStatus(report, goodSharedRuntimeProfileFixture()); err == nil {
		t.Fatal("unmeasured inference activity was admitted as an observed fact")
	}
}

// Negative: a profile whose runtime argv never names an absolute --model
// path must refuse the weight-artifact fact rather than invent one.
func TestSharedRuntimeEngineFactsRefuseWhenNoAbsoluteModelPathIsConfigured(t *testing.T) {
	profile := goodSharedRuntimeProfileFixture()
	profile.Runtime.Argv = []string{"--ctx-size", "4096"}
	if _, err := sanitizedEngineFactsFromSharedRuntimeStatus(goodSharedRuntimeStatusFixture(), profile); err == nil {
		t.Fatal("missing --model argv was admitted as a weight artifact")
	}
}

// A GGUF weight path must select mmap-aware memory accounting to satisfy the
// upstream cross-fact invariant, never resident-bytes.
func TestSharedRuntimeEngineFactsSelectMMapAwareAccountingForGGUF(t *testing.T) {
	profile := goodSharedRuntimeProfileFixture()
	profile.Runtime.Argv = []string{"--model", "/models/weights.gguf"}
	facts, err := sanitizedEngineFactsFromSharedRuntimeStatus(goodSharedRuntimeStatusFixture(), profile)
	if err != nil {
		t.Fatalf("sanitizedEngineFactsFromSharedRuntimeStatus: %v", err)
	}
	readings := make([]inferenceengine.Reading, 0, len(facts))
	for _, fact := range facts {
		if fact.Fact == inferenceengine.FactWeightArtifact && fact.Outcome != inferenceengine.OutcomeObservedValue {
			t.Fatalf("weight artifact = %#v", fact)
		}
		var outcome inferenceengine.ReadOutcome
		if fact.Outcome == inferenceengine.OutcomeObservedAbsent {
			outcome = inferenceengine.ReadAbsent()
		} else {
			outcome = inferenceengine.ReadValue(fact.Value)
		}
		readings = append(readings, inferenceengine.NewReading(fact.Fact, outcome))
	}
	if _, err := inferenceengine.ValidateReadings("mlx", inferenceengine.EngineKindNativeTransformer, readings); err != nil {
		t.Fatalf("ValidateReadings rejected the GGUF mapping: %v", err)
	}
}

// The reader must refuse before ever deriving facts when Process B has not
// been attested as serving and ready, rather than deriving facts from a
// broker report it cannot trust.
func TestSharedRuntimeAttestedAndReadyRefusesEveryUntrustedShape(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		report SharedRuntimeStatus
		want   bool
	}{
		{"broker absent", SharedRuntimeStatus{Broker: SharedRuntimeBrokerStatus{State: "absent"}, LastReadinessMatch: &now}, false},
		{"broker starting", SharedRuntimeStatus{Broker: SharedRuntimeBrokerStatus{State: "starting"}, LastReadinessMatch: &now}, false},
		{"serving without readiness match", SharedRuntimeStatus{Broker: SharedRuntimeBrokerStatus{State: "serving"}}, false},
		{"serving and ready", SharedRuntimeStatus{Broker: SharedRuntimeBrokerStatus{State: "serving"}, LastReadinessMatch: &now}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sharedRuntimeAttestedAndReady(test.report); got != test.want {
				t.Fatalf("sharedRuntimeAttestedAndReady = %v, want %v", got, test.want)
			}
		})
	}
}
