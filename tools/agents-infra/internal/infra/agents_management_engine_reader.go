package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/relux-works/skill-agents-management/pkg/inferenceengine"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
)

// sharedRuntimeObservationTTL bounds how long a single Process-B status read
// may back a launch decision. It is deliberately short: a stale attestation
// must never authorize a later, unrelated launch.
const sharedRuntimeObservationTTL = 30 * time.Second

// SharedRuntimeSanitizedEngineObservationReader is agents-infra's concrete,
// bounded observation source for the registry-installed engine adapter. It
// performs one real status read of the agents-infra-owned Process-B broker,
// then derives the closed sanitized fact set from that live status plus the
// already-resolved profile configuration. It never starts, stops, signals,
// or otherwise mutates Process B; SharedRuntimeStatusReport is the same
// read-only production entry point `agents-infra runtime status` uses.
type SharedRuntimeSanitizedEngineObservationReader struct {
	ProjectDir string
	HomeDir    string
	CacheRoot  string
	HTTPClient *http.Client
	Profile    PiProfile
}

// NewSharedRuntimeSanitizedEngineObservationReader builds the reader for one
// already-resolved Pi profile. The profile is captured once at trusted
// assembly time; ReadSanitizedEngineObservation never re-resolves it from
// caller-supplied input.
func NewSharedRuntimeSanitizedEngineObservationReader(projectDir, homeDir, cacheRoot string, profile PiProfile) *SharedRuntimeSanitizedEngineObservationReader {
	return &SharedRuntimeSanitizedEngineObservationReader{ProjectDir: projectDir, HomeDir: homeDir, CacheRoot: cacheRoot, Profile: profile}
}

func (r *SharedRuntimeSanitizedEngineObservationReader) ReadSanitizedEngineObservation(ctx context.Context, query vendorplugin.EngineObservationQuery) (SanitizedEngineObservation, error) {
	if err := ctx.Err(); err != nil {
		return SanitizedEngineObservation{}, err
	}
	report, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{
		ProjectDir: r.ProjectDir, Profile: string(query.Profile), HomeDir: r.HomeDir, CacheRoot: r.CacheRoot, HTTPClient: r.HTTPClient,
	})
	if err != nil {
		return SanitizedEngineObservation{}, fmt.Errorf("agents-infra: read shared Pi runtime status: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return SanitizedEngineObservation{}, err
	}
	if !sharedRuntimeAttestedAndReady(report) {
		return SanitizedEngineObservation{}, fmt.Errorf("agents-infra: shared Pi runtime engine is not attested and ready (broker_state=%s)", report.Broker.State)
	}
	facts, err := sanitizedEngineFactsFromSharedRuntimeStatus(report, r.Profile)
	if err != nil {
		return SanitizedEngineObservation{}, err
	}
	now := time.Now()
	return SanitizedEngineObservation{
		Contract: SanitizedEngineObservationContract, SchemaVersion: SanitizedEngineObservationSchemaVersion,
		Engine: query.Engine, Runtime: query.Runtime, Model: query.Model, Profile: query.Profile,
		ObservedAt: now, ValidUntil: now.Add(sharedRuntimeObservationTTL), Facts: facts,
	}, nil
}

// sharedRuntimeAttestedAndReady reports whether a live Process-B status read
// is trustworthy enough to derive sanitized facts from: the broker must be
// actively serving, and readiness must have been positively matched at least
// once.
func sharedRuntimeAttestedAndReady(report SharedRuntimeStatus) bool {
	return report.Broker.State == "serving" && report.LastReadinessMatch != nil
}

// sanitizedEngineFactsFromSharedRuntimeStatus derives the closed 17-fact set
// from a real attested Process-B status plus the already-resolved profile.
// It refuses (returns an error, deriving nothing) rather than inventing a
// value it cannot back with real data.
func sanitizedEngineFactsFromSharedRuntimeStatus(report SharedRuntimeStatus, profile PiProfile) ([]SanitizedEngineFact, error) {
	weightArtifact, weightFormat, err := sharedRuntimeWeightArtifactFact(profile.Runtime.Argv)
	if err != nil {
		return nil, err
	}
	memoryAccounting, err := sharedRuntimeMemoryAccountingFact(report, weightFormat)
	if err != nil {
		return nil, err
	}
	inferenceBusy, busy, err := sharedRuntimeInferenceBusyFact(report)
	if err != nil {
		return nil, err
	}
	pressureSequence, err := sharedRuntimeMemoryPressureSequenceFact(report)
	if err != nil {
		return nil, err
	}

	facts := make([]SanitizedEngineFact, 0, len(inferenceengine.MeasuredFacts()))
	for _, definition := range inferenceengine.MeasuredFacts() {
		switch definition.Fact {
		case inferenceengine.FactContextArgv:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(profile.Runtime.Argv)))
		case inferenceengine.FactPrefillArgv:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(profile.Runtime.Argv)))
		case inferenceengine.FactReasoningStreamField:
			if profile.Reasoning {
				facts = append(facts, observedValueFact(definition.Fact, sharedRuntimeReasoningStreamField))
			} else {
				facts = append(facts, absentFact(definition.Fact))
			}
		case inferenceengine.FactHealth:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(struct {
				Endpoint string `json:"endpoint"`
				Healthy  bool   `json:"healthy"`
			}{Endpoint: profile.Runtime.ReadinessPath, Healthy: true})))
		case inferenceengine.FactReadiness:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(struct {
				WeightsResident bool `json:"weights_resident"`
			}{WeightsResident: true})))
		case inferenceengine.FactWeightArtifact:
			facts = append(facts, observedValueFact(definition.Fact, weightArtifact))
		case inferenceengine.FactMemoryAccounting:
			facts = append(facts, observedValueFact(definition.Fact, memoryAccounting))
		case inferenceengine.FactSpeculativeDecoding:
			capable := profile.Runtime.DFlash != nil
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(struct {
				Capable bool `json:"capable"`
				Active  bool `json:"active"`
			}{Capable: capable, Active: capable && busy})))
		case inferenceengine.FactLoadState:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(struct {
				State           string `json:"state"`
				WeightsResident bool   `json:"weights_resident"`
			}{State: "loaded", WeightsResident: true})))
		case inferenceengine.FactUnloadState:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(struct {
				State           string `json:"state"`
				WeightsResident bool   `json:"weights_resident"`
			}{State: "unloaded", WeightsResident: false})))
		case inferenceengine.FactInferenceBusy:
			facts = append(facts, observedValueFact(definition.Fact, inferenceBusy))
		case inferenceengine.FactMemoryPressureSequence:
			facts = append(facts, observedValueFact(definition.Fact, pressureSequence))
		case inferenceengine.FactLocalExecutable:
			facts = append(facts, observedValueFact(definition.Fact, profile.Runtime.Executable))
		case inferenceengine.FactLocalArgv:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(profile.Runtime.Argv)))
		case inferenceengine.FactSSHForwarding:
			facts = append(facts, absentFact(definition.Fact))
		case inferenceengine.FactStressPolicy:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(sharedRuntimeStressPolicy(profile.Runtime.Sharing))))
		case inferenceengine.FactRestartSupervisionPolicy:
			facts = append(facts, observedValueFact(definition.Fact, mustEncodeJSON(sharedRuntimeRestartPolicy(profile.Runtime.Sharing))))
		default:
			return nil, fmt.Errorf("agents-infra: unknown measured fact %q", definition.Fact)
		}
	}
	return facts, nil
}

// sharedRuntimeReasoningStreamField is the field path agents-infra's
// OpenAI-compatible local runtimes use to stream reasoning deltas.
const sharedRuntimeReasoningStreamField = "delta.reasoning_content"

func observedValueFact(fact inferenceengine.Fact, value string) SanitizedEngineFact {
	return SanitizedEngineFact{Fact: fact, Outcome: inferenceengine.OutcomeObservedValue, Value: value}
}

func absentFact(fact inferenceengine.Fact) SanitizedEngineFact {
	return SanitizedEngineFact{Fact: fact, Outcome: inferenceengine.OutcomeObservedAbsent}
}

func mustEncodeJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("agents-infra: encode sanitized engine fact: %v", err))
	}
	return string(encoded)
}

func sharedRuntimeWeightArtifactFact(argv []string) (value, format string, err error) {
	modelPath := ""
	for index, token := range argv {
		if (token == "--model" || token == "-m") && index+1 < len(argv) {
			modelPath = argv[index+1]
			break
		}
		if after, ok := strings.CutPrefix(token, "--model="); ok {
			modelPath = after
			break
		}
	}
	if modelPath == "" || !filepath.IsAbs(modelPath) {
		return "", "", errors.New("agents-infra: profile runtime argv does not name an absolute --model path")
	}
	modelPath = filepath.Clean(modelPath)
	if strings.EqualFold(filepath.Ext(modelPath), ".gguf") {
		return mustEncodeJSON(struct {
			Format    string `json:"format"`
			ModelPath string `json:"model_path"`
		}{Format: "gguf", ModelPath: modelPath}), "gguf", nil
	}
	directory := modelPath
	if info, statErr := os.Stat(modelPath); statErr == nil && !info.IsDir() {
		directory = filepath.Dir(modelPath)
	}
	return mustEncodeJSON(struct {
		Format     string `json:"format"`
		ModelPath  string `json:"model_path"`
		ConfigPath string `json:"config_path"`
	}{Format: "safetensors", ModelPath: modelPath, ConfigPath: filepath.Join(directory, "config.json")}), "safetensors", nil
}

func sharedRuntimeMemoryAccountingFact(report SharedRuntimeStatus, weightFormat string) (string, error) {
	if report.Resources.LoadedModelMemory.State != SharedRuntimeMemoryHealthy && report.Resources.LoadedModelMemory.State != SharedRuntimeMemoryPressured {
		return "", fmt.Errorf("agents-infra: loaded-model memory is not observed (state=%s); configure resource_pressure_mode=provider to observe it", report.Resources.LoadedModelMemory.State)
	}
	if report.Resources.LoadedModelMemory.Bytes == nil {
		return "", errors.New("agents-infra: loaded-model memory state is observed without a byte count")
	}
	method := "resident-bytes"
	includesMapped := false
	if weightFormat == "gguf" {
		method, includesMapped = "mmap-aware", true
	}
	return mustEncodeJSON(struct {
		Method                string `json:"method"`
		Bytes                 uint64 `json:"bytes"`
		IncludesMappedWeights bool   `json:"includes_mapped_weights"`
	}{Method: method, Bytes: *report.Resources.LoadedModelMemory.Bytes, IncludesMappedWeights: includesMapped}), nil
}

func sharedRuntimeInferenceBusyFact(report SharedRuntimeStatus) (string, bool, error) {
	switch report.Resources.Inference.State {
	case SharedRuntimeInferenceIdle:
		return mustEncodeJSON(struct {
			Busy bool `json:"busy"`
		}{Busy: false}), false, nil
	case SharedRuntimeInferenceBusy:
		return mustEncodeJSON(struct {
			Busy bool `json:"busy"`
		}{Busy: true}), true, nil
	default:
		return "", false, fmt.Errorf("agents-infra: inference activity is not observed (state=%s)", report.Resources.Inference.State)
	}
}

func sharedRuntimeMemoryPressureSequenceFact(report SharedRuntimeStatus) (string, error) {
	pressure := "normal"
	action := "none"
	switch report.Resources.State {
	case SharedRuntimeResourceHealthy, SharedRuntimeResourceBusy:
		pressure, action = "normal", "none"
	case SharedRuntimeResourcePressured:
		pressure, action = "critical", "unload-idle"
	case SharedRuntimeResourceDraining:
		pressure, action = "critical", "refuse"
	default:
		return "", fmt.Errorf("agents-infra: memory pressure sequence is not observed (state=%s)", report.Resources.State)
	}
	return mustEncodeJSON(struct {
		Pressure  string                 `json:"pressure"`
		Consulted []inferenceengine.Fact `json:"consulted"`
		Action    string                 `json:"action"`
	}{
		Pressure: pressure,
		Consulted: []inferenceengine.Fact{
			inferenceengine.FactLoadState, inferenceengine.FactUnloadState, inferenceengine.FactInferenceBusy,
		},
		Action: action,
	}), nil
}

func sharedRuntimeStressPolicy(sharing *PiRuntimeSharing) struct {
	Enabled        bool `json:"enabled"`
	MaxConcurrency int  `json:"max_concurrency"`
} {
	if sharing == nil || sharing.Mode != "shared" {
		return struct {
			Enabled        bool `json:"enabled"`
			MaxConcurrency int  `json:"max_concurrency"`
		}{Enabled: false, MaxConcurrency: 1}
	}
	concurrency := sharing.MaxLeases
	if concurrency < 1 {
		concurrency = 1
	}
	return struct {
		Enabled        bool `json:"enabled"`
		MaxConcurrency int  `json:"max_concurrency"`
	}{Enabled: true, MaxConcurrency: concurrency}
}

func sharedRuntimeRestartPolicy(sharing *PiRuntimeSharing) struct {
	MaxAttempts      int `json:"max_attempts"`
	InitialBackoffMS int `json:"initial_backoff_ms"`
	MaxBackoffMS     int `json:"max_backoff_ms"`
} {
	if sharing == nil {
		return struct {
			MaxAttempts      int `json:"max_attempts"`
			InitialBackoffMS int `json:"initial_backoff_ms"`
			MaxBackoffMS     int `json:"max_backoff_ms"`
		}{MaxAttempts: 0, InitialBackoffMS: 1000, MaxBackoffMS: 1000}
	}
	initial := sharing.RestartInitialBackoffSeconds * 1000
	if initial < 1 {
		initial = 1
	}
	max := sharing.RestartMaxBackoffSeconds * 1000
	if max < initial {
		max = initial
	}
	return struct {
		MaxAttempts      int `json:"max_attempts"`
		InitialBackoffMS int `json:"initial_backoff_ms"`
		MaxBackoffMS     int `json:"max_backoff_ms"`
	}{MaxAttempts: sharing.RestartLimit, InitialBackoffMS: initial, MaxBackoffMS: max}
}
