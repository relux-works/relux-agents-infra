package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	SharedRuntimeResourceStatusSchema                      = "agents-infra.pi.shared-runtime.resource-status.v1"
	sharedRuntimeResourceObservationSchema                 = "agents-infra.pi.shared-runtime.resource-observation.v1"
	sharedRuntimeResourceObservationMaxBytes               = 16 * 1024
	sharedRuntimeResourceObservationMaxTimeoutMilliseconds = 30_000
)

type SharedRuntimeResourceState string
type SharedRuntimeMemoryState string
type SharedRuntimeInferenceState string
type SharedRuntimeAdmissionState string

const (
	SharedRuntimeResourceHealthy   SharedRuntimeResourceState = "healthy"
	SharedRuntimeResourceBusy      SharedRuntimeResourceState = "busy"
	SharedRuntimeResourcePressured SharedRuntimeResourceState = "pressured"
	SharedRuntimeResourceDraining  SharedRuntimeResourceState = "draining"
	SharedRuntimeResourceUnknown   SharedRuntimeResourceState = "unknown"

	SharedRuntimeMemoryHealthy   SharedRuntimeMemoryState = "healthy"
	SharedRuntimeMemoryPressured SharedRuntimeMemoryState = "pressured"
	SharedRuntimeMemoryUnknown   SharedRuntimeMemoryState = "unknown"

	SharedRuntimeInferenceIdle    SharedRuntimeInferenceState = "idle"
	SharedRuntimeInferenceBusy    SharedRuntimeInferenceState = "busy"
	SharedRuntimeInferenceUnknown SharedRuntimeInferenceState = "unknown"

	SharedRuntimeAdmissionAdmitted    SharedRuntimeAdmissionState = "admitted"
	SharedRuntimeAdmissionRefused     SharedRuntimeAdmissionState = "refused"
	SharedRuntimeAdmissionNotEnforced SharedRuntimeAdmissionState = "not-enforced"
)

type SharedRuntimeLoadedModelMemoryStatus struct {
	State           SharedRuntimeMemoryState `json:"state"`
	Bytes           *uint64                  `json:"bytes,omitempty"`
	PressureAtBytes int                      `json:"pressure_at_bytes,omitempty"`
	RecoveryAtBytes int                      `json:"recovery_at_bytes,omitempty"`
	Reason          string                   `json:"reason,omitempty"`
}

type SharedRuntimeInferenceStatus struct {
	State  SharedRuntimeInferenceState `json:"state"`
	Reason string                      `json:"reason,omitempty"`
}

type SharedRuntimeResourcePolicyStatus struct {
	Mode                 string `json:"mode"`
	EvictionGraceSeconds int    `json:"eviction_grace_seconds"`
	PressureAction       string `json:"pressure_action,omitempty"`
	UnknownAction        string `json:"unknown_action,omitempty"`
	BusyAction           string `json:"busy_action,omitempty"`
}

// SharedRuntimeResourceStatus is the versioned consumer handoff for resource
// admission. State is one of healthy, busy, pressured, draining, or unknown.
// The independent facts remain explicit so a consumer never has to infer busy
// or pressure from leases, restart counts, or broker lifecycle state.
type SharedRuntimeResourceStatus struct {
	Schema            string                               `json:"schema"`
	State             SharedRuntimeResourceState           `json:"state"`
	Source            string                               `json:"source"`
	ObservedAt        *time.Time                           `json:"observed_at,omitempty"`
	LoadedModelMemory SharedRuntimeLoadedModelMemoryStatus `json:"loaded_model_memory"`
	Inference         SharedRuntimeInferenceStatus         `json:"inference"`
	Admission         SharedRuntimeAdmissionState          `json:"admission"`
	Reason            string                               `json:"reason,omitempty"`
	Policy            SharedRuntimeResourcePolicyStatus    `json:"policy"`
}

type sharedRuntimeProviderMemoryFact struct {
	State  string  `json:"state"`
	Bytes  *uint64 `json:"bytes,omitempty"`
	Reason string  `json:"reason,omitempty"`
}

type sharedRuntimeProviderInferenceFact struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type sharedRuntimeProviderResourceObservation struct {
	Schema            string                             `json:"schema"`
	Model             string                             `json:"model"`
	LoadedModelMemory sharedRuntimeProviderMemoryFact    `json:"loaded_model_memory"`
	Inference         sharedRuntimeProviderInferenceFact `json:"inference"`
}

func disabledSharedRuntimeResourceStatus(sharing PiRuntimeSharing, brokerState, source string) SharedRuntimeResourceStatus {
	state := SharedRuntimeResourceUnknown
	admission := SharedRuntimeAdmissionNotEnforced
	if brokerState == "draining" {
		state = SharedRuntimeResourceDraining
		admission = SharedRuntimeAdmissionRefused
	}
	return SharedRuntimeResourceStatus{
		Schema:            SharedRuntimeResourceStatusSchema,
		State:             state,
		Source:            source,
		LoadedModelMemory: SharedRuntimeLoadedModelMemoryStatus{State: SharedRuntimeMemoryUnknown, Reason: "resource_pressure_disabled"},
		Inference:         SharedRuntimeInferenceStatus{State: SharedRuntimeInferenceUnknown, Reason: "resource_pressure_disabled"},
		Admission:         admission,
		Reason:            "resource_pressure_disabled",
		Policy:            SharedRuntimeResourcePolicyStatus{Mode: sharing.ResourcePressureMode},
	}
}

func unavailableSharedRuntimeResourceStatus(sharing PiRuntimeSharing, brokerState, source string) SharedRuntimeResourceStatus {
	if sharing.ResourcePressureMode == "disabled" && sharing.ResourcePressure == nil {
		return disabledSharedRuntimeResourceStatus(sharing, brokerState, source)
	}
	if sharing.ResourcePressureMode == "provider" && sharing.ResourcePressure != nil {
		status := failedSharedRuntimeResourceStatus(*sharing.ResourcePressure, brokerState, "broker_resource_observation_unavailable")
		status.Source = source
		return status
	}
	return unknownSharedRuntimeResourcePolicyStatus(brokerState, source)
}

func unknownSharedRuntimeResourcePolicyStatus(brokerState, source string) SharedRuntimeResourceStatus {
	state := SharedRuntimeResourceUnknown
	if brokerState == "draining" {
		state = SharedRuntimeResourceDraining
	}
	const reason = "resource_pressure_policy_unknown"
	return SharedRuntimeResourceStatus{
		Schema:            SharedRuntimeResourceStatusSchema,
		State:             state,
		Source:            source,
		LoadedModelMemory: SharedRuntimeLoadedModelMemoryStatus{State: SharedRuntimeMemoryUnknown, Reason: reason},
		Inference:         SharedRuntimeInferenceStatus{State: SharedRuntimeInferenceUnknown, Reason: reason},
		Admission:         SharedRuntimeAdmissionRefused,
		Reason:            reason,
		Policy:            SharedRuntimeResourcePolicyStatus{Mode: "unknown"},
	}
}

func failedSharedRuntimeResourceStatus(policy PiRuntimeResourcePressure, brokerState, reason string) SharedRuntimeResourceStatus {
	state := SharedRuntimeResourceUnknown
	if brokerState == "draining" {
		state = SharedRuntimeResourceDraining
	}
	return SharedRuntimeResourceStatus{
		Schema: SharedRuntimeResourceStatusSchema,
		State:  state,
		Source: "provider-observation-failed",
		LoadedModelMemory: SharedRuntimeLoadedModelMemoryStatus{
			State: SharedRuntimeMemoryUnknown, PressureAtBytes: policy.PressureThresholdBytes,
			RecoveryAtBytes: policy.RecoveryThresholdBytes, Reason: reason,
		},
		Inference: SharedRuntimeInferenceStatus{State: SharedRuntimeInferenceUnknown, Reason: reason},
		Admission: SharedRuntimeAdmissionRefused,
		Reason:    reason,
		Policy:    sharedRuntimeResourcePolicyStatus(policy),
	}
}

func sharedRuntimeResourcePolicyStatus(policy PiRuntimeResourcePressure) SharedRuntimeResourcePolicyStatus {
	return SharedRuntimeResourcePolicyStatus{
		Mode: "provider", EvictionGraceSeconds: policy.EvictionGraceSeconds,
		PressureAction: policy.PressureAction, UnknownAction: policy.UnknownAction,
		BusyAction: policy.BusyAction,
	}
}

// sharedRuntimeResourcePolicyMatches is the single lease-admission
// compatibility boundary between a caller's configured policy and the policy
// fixed by the live broker. Resource-pressure policy is intentionally strict:
// any mode or table difference refuses acquisition instead of silently
// weakening, strengthening, or otherwise changing the caller's reviewed gate.
func sharedRuntimeResourcePolicyMatches(configured *PiRuntimeSharing, effective PiRuntimeSharing) bool {
	if configured == nil || configured.ResourcePressureMode != effective.ResourcePressureMode {
		return false
	}
	switch effective.ResourcePressureMode {
	case "disabled":
		return configured.ResourcePressure == nil && effective.ResourcePressure == nil
	case "provider":
		return configured.ResourcePressure != nil && effective.ResourcePressure != nil &&
			*configured.ResourcePressure == *effective.ResourcePressure
	default:
		return false
	}
}

func classifySharedRuntimeResources(observation sharedRuntimeProviderResourceObservation, policy PiRuntimeResourcePressure, brokerState string, pressureLatched bool, now time.Time) (SharedRuntimeResourceStatus, bool) {
	status := SharedRuntimeResourceStatus{
		Schema: SharedRuntimeResourceStatusSchema, Source: "provider-observed",
		ObservedAt: timePointer(now.UTC()), Admission: SharedRuntimeAdmissionAdmitted,
		Policy: sharedRuntimeResourcePolicyStatus(policy),
	}
	if observation.LoadedModelMemory.State == "unknown" {
		status.LoadedModelMemory = SharedRuntimeLoadedModelMemoryStatus{
			State: SharedRuntimeMemoryUnknown, PressureAtBytes: policy.PressureThresholdBytes,
			RecoveryAtBytes: policy.RecoveryThresholdBytes, Reason: observation.LoadedModelMemory.Reason,
		}
	} else {
		bytesValue := *observation.LoadedModelMemory.Bytes
		if pressureLatched {
			if bytesValue <= uint64(policy.RecoveryThresholdBytes) {
				pressureLatched = false
			}
		} else if bytesValue >= uint64(policy.PressureThresholdBytes) {
			pressureLatched = true
		}
		memoryState := SharedRuntimeMemoryHealthy
		if pressureLatched {
			memoryState = SharedRuntimeMemoryPressured
		}
		status.LoadedModelMemory = SharedRuntimeLoadedModelMemoryStatus{
			State: memoryState, Bytes: &bytesValue,
			PressureAtBytes: policy.PressureThresholdBytes,
			RecoveryAtBytes: policy.RecoveryThresholdBytes,
		}
	}
	status.Inference = SharedRuntimeInferenceStatus{State: SharedRuntimeInferenceState(observation.Inference.State), Reason: observation.Inference.Reason}
	switch {
	case brokerState == "draining":
		status.State, status.Admission, status.Reason = SharedRuntimeResourceDraining, SharedRuntimeAdmissionRefused, "broker_draining"
	case status.LoadedModelMemory.State == SharedRuntimeMemoryPressured:
		status.State, status.Admission, status.Reason = SharedRuntimeResourcePressured, SharedRuntimeAdmissionRefused, "loaded_model_memory_pressure"
	case status.LoadedModelMemory.State == SharedRuntimeMemoryUnknown || status.Inference.State == SharedRuntimeInferenceUnknown:
		status.State, status.Admission, status.Reason = SharedRuntimeResourceUnknown, SharedRuntimeAdmissionRefused, "resource_observation_unknown"
	case status.Inference.State == SharedRuntimeInferenceBusy:
		status.State = SharedRuntimeResourceBusy
	default:
		status.State = SharedRuntimeResourceHealthy
	}
	return status, pressureLatched
}

func observeSharedRuntimeProviderResources(ctx context.Context, client *http.Client, baseURL, model string, policy PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return sharedRuntimeProviderResourceObservation{}, err
	}
	endpoint.Path = policy.ObservationPath
	endpoint.RawPath = ""
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return sharedRuntimeProviderResourceObservation{}, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return sharedRuntimeProviderResourceObservation{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return sharedRuntimeProviderResourceObservation{}, fmt.Errorf("resource observation status %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, sharedRuntimeResourceObservationMaxBytes+1))
	if err != nil {
		return sharedRuntimeProviderResourceObservation{}, err
	}
	if len(data) > sharedRuntimeResourceObservationMaxBytes {
		return sharedRuntimeProviderResourceObservation{}, errors.New("resource observation exceeds 16384 bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var observation sharedRuntimeProviderResourceObservation
	if err := decoder.Decode(&observation); err != nil {
		return sharedRuntimeProviderResourceObservation{}, err
	}
	if err := requireSharedRuntimeResourceJSONEOF(decoder); err != nil {
		return sharedRuntimeProviderResourceObservation{}, err
	}
	if err := validateSharedRuntimeProviderObservation(observation, model); err != nil {
		return sharedRuntimeProviderResourceObservation{}, err
	}
	return observation, nil
}

func validateSharedRuntimeProviderObservation(observation sharedRuntimeProviderResourceObservation, model string) error {
	if observation.Schema != sharedRuntimeResourceObservationSchema {
		return errors.New("resource observation schema differs")
	}
	if observation.Model != model {
		return errors.New("resource observation model differs")
	}
	switch observation.LoadedModelMemory.State {
	case "observed":
		if observation.LoadedModelMemory.Bytes == nil || *observation.LoadedModelMemory.Bytes == 0 || observation.LoadedModelMemory.Reason != "" {
			return errors.New("observed loaded-model memory fact is incomplete")
		}
	case "unknown":
		if observation.LoadedModelMemory.Bytes != nil || observation.LoadedModelMemory.Reason == "" {
			return errors.New("unknown loaded-model memory fact is incomplete")
		}
	default:
		return errors.New("loaded-model memory state must equal observed or unknown")
	}
	switch observation.Inference.State {
	case "idle", "busy":
		if observation.Inference.Reason != "" {
			return errors.New("observed inference fact must not carry a reason")
		}
	case "unknown":
		if observation.Inference.Reason == "" {
			return errors.New("unknown inference fact requires a reason")
		}
	default:
		return errors.New("inference state must equal idle, busy, or unknown")
	}
	return nil
}

func timePointer(value time.Time) *time.Time { return &value }

func requireSharedRuntimeResourceJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("content follows resource observation")
		}
		return err
	}
	return nil
}
