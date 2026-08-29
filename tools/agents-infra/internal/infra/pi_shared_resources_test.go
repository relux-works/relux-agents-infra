//go:build darwin

package infra

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func testSharedRuntimeResourcePolicy() PiRuntimeResourcePressure {
	return PiRuntimeResourcePressure{
		ObservationPath: "/agents-infra/resources", ObservationTimeoutMilliseconds: 100,
		PressureThresholdBytes: 1000, RecoveryThresholdBytes: 800, EvictionGraceSeconds: 0,
		PressureAction: "refuse-new-drain-idle", UnknownAction: "refuse-new", BusyAction: "observe",
	}
}

func sharedRuntimeProviderTestProfileTOML(name, runtime string, port int) string {
	return strings.Replace(sharedPiProfileTOML(name, runtime, port),
		`resource_pressure_mode = "disabled"`,
		`resource_pressure_mode = "provider"

[agents.pi.profiles."profile".runtime.sharing.resource_pressure]
observation_path = "/agents-infra/resources"
observation_timeout_milliseconds = 100
pressure_threshold_bytes = 1000
recovery_threshold_bytes = 800
eviction_grace_seconds = 0
pressure_action = "refuse-new-drain-idle"
unknown_action = "refuse-new"
busy_action = "observe"`, 1)
}

func observedSharedRuntimeResources(bytesValue uint64, inference string) sharedRuntimeProviderResourceObservation {
	return sharedRuntimeProviderResourceObservation{
		Schema: sharedRuntimeResourceObservationSchema, Model: "Qwen-test",
		LoadedModelMemory: sharedRuntimeProviderMemoryFact{State: "observed", Bytes: &bytesValue},
		Inference:         sharedRuntimeProviderInferenceFact{State: inference},
	}
}

func TestSharedRuntimeProviderObservationIsBoundedStrictAndVersioned(t *testing.T) {
	policy := testSharedRuntimeResourcePolicy()
	fixtureData, err := os.ReadFile("testdata/shared-runtime-resource-observation-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var valid sharedRuntimeProviderResourceObservation
	decoder := json.NewDecoder(bytes.NewReader(fixtureData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&valid); err != nil || requireSharedRuntimeResourceJSONEOF(decoder) != nil {
		t.Fatalf("versioned provider fixture is invalid: %v", err)
	}
	tests := []struct {
		name    string
		body    any
		status  int
		wantErr bool
	}{
		{name: "versioned observed facts", body: valid},
		{name: "wrong schema", body: func() any { value := valid; value.Schema = "v0"; return value }(), wantErr: true},
		{name: "wrong model", body: func() any { value := valid; value.Model = "other"; return value }(), wantErr: true},
		{name: "unknown field", body: map[string]any{"schema": sharedRuntimeResourceObservationSchema, "model": "Qwen-test", "loaded_model_memory": map[string]any{"state": "observed", "bytes": 900}, "inference": map[string]any{"state": "idle"}, "lease_count": 0}, wantErr: true},
		{name: "read failure is not unknown", status: http.StatusServiceUnavailable, body: map[string]any{"error": "unavailable"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != policy.ObservationPath {
					t.Fatalf("observation path=%q", request.URL.Path)
				}
				status := tc.status
				if status == 0 {
					status = http.StatusOK
				}
				writer.WriteHeader(status)
				_ = json.NewEncoder(writer).Encode(tc.body)
			}))
			defer server.Close()
			observation, err := observeSharedRuntimeProviderResources(context.Background(), server.Client(), server.URL+"/v1", "Qwen-test", policy)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("invalid provider observation admitted: %#v", observation)
				}
				return
			}
			if err != nil || observation.Schema != sharedRuntimeResourceObservationSchema || observation.Inference.State != "busy" {
				t.Fatalf("observation=%#v err=%v", observation, err)
			}
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(strings.Repeat("x", sharedRuntimeResourceObservationMaxBytes+1)))
	}))
	defer server.Close()
	if _, err := observeSharedRuntimeProviderResources(context.Background(), server.Client(), server.URL+"/v1", "Qwen-test", policy); err == nil {
		t.Fatal("oversized provider observation was admitted")
	}
}

func TestSharedRuntimeResourceStatusConsumerFixtureV1(t *testing.T) {
	status, pressure := classifySharedRuntimeResources(observedSharedRuntimeResources(900, "busy"), testSharedRuntimeResourcePolicy(), "serving", false, time.Unix(100, 0).UTC())
	if pressure {
		t.Fatal("healthy fixture unexpectedly latched pressure")
	}
	got, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	want, err := os.ReadFile("testdata/shared-runtime-resource-status-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("consumer fixture drifted\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// Production call site: SharedRuntimeStatusReport must refuse an attested
// broker status response when the resources evidence is absent. Falling back
// to the report's configured-policy default would launder missing evidence.
func TestSharedRuntimeStatusReportRefusesAttestedStatusWithoutResources(t *testing.T) {
	fixture := newSharedAttestationFixture(t)
	served := make(chan error, 1)
	go func() {
		connection, err := fixture.listener.AcceptUnix()
		if err != nil {
			served <- err
			return
		}
		defer connection.Close()
		reader := bufio.NewReaderSize(connection, sharedRuntimeMaxFrameBytes+1)
		if _, err := readSharedWireMessage(reader); err != nil {
			served <- err
			return
		}
		effective := fixture.resolved.Sharing
		if err := writeSharedWireMessage(connection, sharedWireMessage{
			Type: "hello_ok", ProtocolVersion: SharedRuntimeProtocolVersion,
			RuntimeKey: fixture.resolved.RuntimeKey, ProfileDigest: fixture.resolved.ProfileDigest,
			Broker: &fixture.broker, Runtime: &fixture.record, EffectiveSharing: &effective,
		}); err != nil {
			served <- err
			return
		}
		request, err := readSharedWireMessage(reader)
		if err != nil {
			served <- err
			return
		}
		if request.Type != "status" {
			served <- errors.New("production entry did not request broker status")
			return
		}
		served <- writeSharedWireMessage(connection, sharedWireMessage{
			Type: "status", State: "serving", Stage: fixture.record.Stage,
			Broker: &fixture.broker, Runtime: &fixture.record, EffectiveSharing: &effective,
		})
	}()

	_, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{
		ProjectDir: fixture.resolved.Project, HomeDir: fixture.resolved.HomeDir,
		CacheRoot: fixture.resolved.Paths.CanonicalCacheRoot, Profile: fixture.resolved.ProfileName,
	})
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) || shared.Code != "protocol_violation" {
		t.Fatalf("absent attested resources error=%v want protocol_violation", err)
	}
	select {
	case err := <-served:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("fake attested broker did not serve the status response")
	}
}

// Production call site: SharedRuntimeStatusReport -> sharedRuntimeRecordStatus.
// A stale persisted record is unverified, but its effective sharing provenance
// must still agree with the resource-policy view. Caller configuration must not
// be laundered into an effective disabled or provider enforcement claim.
func TestSharedRuntimeStatusReportRecordDerivedResourcePolicyUsesCoherentProvenance(t *testing.T) {
	tests := []struct {
		name               string
		configuredProvider bool
		recordSharing      func(PiRuntimeSharing) *PiRuntimeSharing
		wantEffectiveMode  string
		wantPolicyMode     string
		wantReason         string
		wantAdmission      SharedRuntimeAdmissionState
	}{
		{
			name: "configured disabled and record effective provider",
			recordSharing: func(configured PiRuntimeSharing) *PiRuntimeSharing {
				policy := testSharedRuntimeResourcePolicy()
				configured.ResourcePressureMode = "provider"
				configured.ResourcePressure = &policy
				return &configured
			},
			wantEffectiveMode: "provider", wantPolicyMode: "provider",
			wantReason: "broker_resource_observation_unavailable", wantAdmission: SharedRuntimeAdmissionRefused,
		},
		{
			name:               "configured provider and record effective disabled",
			configuredProvider: true,
			recordSharing: func(configured PiRuntimeSharing) *PiRuntimeSharing {
				configured.ResourcePressureMode = "disabled"
				configured.ResourcePressure = nil
				return &configured
			},
			wantEffectiveMode: "disabled", wantPolicyMode: "disabled",
			wantReason: "resource_pressure_disabled", wantAdmission: SharedRuntimeAdmissionNotEnforced,
		},
		{
			name:           "pre extension record has unknown effective policy",
			recordSharing:  func(PiRuntimeSharing) *PiRuntimeSharing { return nil },
			wantPolicyMode: "unknown", wantReason: "resource_pressure_policy_unknown",
			wantAdmission: SharedRuntimeAdmissionRefused,
		},
		{
			name: "partial record provider policy is unknown",
			recordSharing: func(configured PiRuntimeSharing) *PiRuntimeSharing {
				configured.ResourcePressureMode = "provider"
				configured.ResourcePressure = nil
				return &configured
			},
			wantEffectiveMode: "provider", wantPolicyMode: "unknown",
			wantReason: "resource_pressure_policy_unknown", wantAdmission: SharedRuntimeAdmissionRefused,
		},
		{
			name: "partial record disabled policy is unknown",
			recordSharing: func(configured PiRuntimeSharing) *PiRuntimeSharing {
				policy := testSharedRuntimeResourcePolicy()
				configured.ResourcePressureMode = "disabled"
				configured.ResourcePressure = &policy
				return &configured
			},
			wantEffectiveMode: "disabled", wantPolicyMode: "unknown",
			wantReason: "resource_pressure_policy_unknown", wantAdmission: SharedRuntimeAdmissionRefused,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			project, home, cache, resolved := newSharedIntegrationProfile(t)
			if testCase.configuredProvider {
				writePiProjectConfig(t, project, sharedRuntimeProviderTestProfileTOML("profile", "/bin/echo", 18011))
				var err error
				resolved, err = resolveSharedProfile(project, home, cache, "profile")
				if err != nil {
					t.Fatal(err)
				}
			}
			record := SharedBrokerRecord{
				Stage: "serving", State: "serving", ProtocolVersion: SharedRuntimeProtocolVersion,
				Broker:            SharedBrokerIdentity{PID: 99_999_999},
				RuntimeKeyClaimed: resolved.RuntimeKey, RuntimeKey: resolved.RuntimeKey,
				ProfileDigest: resolved.ProfileDigest,
				Sharing:       testCase.recordSharing(resolved.Sharing),
			}
			if err := writeSharedJSONAtomic(resolved.Paths.BrokerState, record); err != nil {
				t.Fatal(err)
			}

			status, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{
				ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile",
			})
			if err != nil {
				t.Fatal(err)
			}
			if status.Broker.State != "unverified-stale" || status.Broker.Source != "record-derived-unverified" {
				t.Fatalf("record broker provenance=%#v", status.Broker)
			}
			if status.Resources.Source != "record-derived-unverified" || status.Resources.ObservedAt != nil ||
				status.Resources.LoadedModelMemory.State != SharedRuntimeMemoryUnknown ||
				status.Resources.Inference.State != SharedRuntimeInferenceUnknown {
				t.Fatalf("record resource observation was guessed: %#v", status.Resources)
			}
			if status.Resources.Policy.Mode != testCase.wantPolicyMode || status.Resources.Reason != testCase.wantReason || status.Resources.Admission != testCase.wantAdmission {
				t.Fatalf("record resource policy=%#v want mode=%q reason=%q admission=%q", status.Resources, testCase.wantPolicyMode, testCase.wantReason, testCase.wantAdmission)
			}
			if testCase.wantEffectiveMode == "" {
				if status.Sharing.Effective != nil {
					t.Fatalf("pre-extension record invented effective sharing: %#v", status.Sharing.Effective)
				}
			} else if status.Sharing.Effective == nil || status.Sharing.Effective.ResourcePressureMode != testCase.wantEffectiveMode {
				t.Fatalf("record effective sharing=%#v want mode=%q", status.Sharing.Effective, testCase.wantEffectiveMode)
			}
		})
	}
}

func TestSharedRuntimeResourceClassificationPreservesUnknownAndHysteresis(t *testing.T) {
	policy := testSharedRuntimeResourcePolicy()
	now := time.Unix(100, 0).UTC()
	status, pressure := classifySharedRuntimeResources(observedSharedRuntimeResources(700, "idle"), policy, "serving", false, now)
	if status.State != SharedRuntimeResourceHealthy || status.Admission != SharedRuntimeAdmissionAdmitted || pressure {
		t.Fatalf("healthy classification=%#v latched=%t", status, pressure)
	}
	status, pressure = classifySharedRuntimeResources(observedSharedRuntimeResources(1000, "idle"), policy, "serving", pressure, now)
	if status.State != "pressured" || status.Admission != "refused" || !pressure {
		t.Fatalf("pressure classification=%#v latched=%t", status, pressure)
	}
	status, pressure = classifySharedRuntimeResources(observedSharedRuntimeResources(900, "busy"), policy, "serving", pressure, now)
	if status.State != "pressured" || !pressure {
		t.Fatalf("hysteresis gap cleared pressure: %#v latched=%t", status, pressure)
	}
	status, pressure = classifySharedRuntimeResources(observedSharedRuntimeResources(800, "busy"), policy, "serving", pressure, now)
	if status.State != "busy" || status.Inference.State != "busy" || status.Admission != "admitted" || pressure {
		t.Fatalf("recovery classification=%#v latched=%t", status, pressure)
	}
	unknown := observedSharedRuntimeResources(700, "idle")
	unknown.LoadedModelMemory = sharedRuntimeProviderMemoryFact{State: "unknown", Reason: "provider_does_not_report_memory"}
	status, pressure = classifySharedRuntimeResources(unknown, policy, "serving", pressure, now)
	if status.State != "unknown" || status.Admission != "refused" || pressure {
		t.Fatalf("unknown fact was guessed: %#v latched=%t", status, pressure)
	}
	status, _ = classifySharedRuntimeResources(observedSharedRuntimeResources(700, "idle"), policy, "draining", false, now)
	if status.State != "draining" || status.Admission != "refused" {
		t.Fatalf("draining classification=%#v", status)
	}
}

func configureSharedBrokerResourceFixture(fixture *sharedBrokerAdmissionFixture, observations ...sharedRuntimeProviderResourceObservation) {
	policy := testSharedRuntimeResourcePolicy()
	fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
	fixture.server.resolved.Sharing.ResourcePressure = &policy
	fixture.hello.ConfiguredSharing = &fixture.server.resolved.Sharing
	index := 0
	fixture.server.resources = sharedBrokerResourceDependencies{
		observe: func(context.Context, *http.Client, string, string, PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
			if index >= len(observations) {
				return sharedRuntimeProviderResourceObservation{}, errors.New("fake provider fixture exhausted")
			}
			observation := observations[index]
			index++
			return observation, nil
		},
		now: func() time.Time { return time.Unix(200, 0).UTC() },
	}
}

func startSharedBrokerFixtureConnection(t *testing.T, fixture *sharedBrokerAdmissionFixture) (*net.UnixConn, *bufio.Reader, <-chan struct{}) {
	t.Helper()
	serverConnection, clientConnection := sharedBrokerUnixConnectionPair(t)
	wrapped := &sharedBrokerConnection{connection: serverConnection, reader: bufio.NewReaderSize(serverConnection, sharedRuntimeMaxFrameBytes+1)}
	fixture.server.mu.Lock()
	fixture.server.connections[wrapped] = true
	fixture.server.mu.Unlock()
	done := make(chan struct{})
	go func() {
		fixture.server.handleConnection(wrapped)
		close(done)
	}()
	reader := bufio.NewReaderSize(clientConnection, sharedRuntimeMaxFrameBytes+1)
	if err := writeSharedWireMessage(clientConnection, fixture.hello); err != nil {
		t.Fatal(err)
	}
	if message, err := readSharedWireMessage(reader); err != nil || message.Type != "hello_ok" {
		t.Fatalf("hello response=%#v err=%v", message, err)
	}
	return clientConnection, reader, done
}

// Production call site: sharedBrokerServer.handleConnection -> acquireLease.
// The live broker must refuse before provider observation or lease reservation
// whenever its fixed resource-pressure policy differs from the caller's
// configured policy. This covers a provider-to-disabled downgrade, a narrowed
// threshold, and absent caller evidence.
func TestSharedBrokerProductionResourcePolicyMismatchRefusesBeforeObservationOrLease(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*sharedBrokerAdmissionFixture) *PiRuntimeSharing
	}{
		{
			name: "provider caller cannot use disabled broker",
			configure: func(fixture *sharedBrokerAdmissionFixture) *PiRuntimeSharing {
				fixture.server.resolved.Sharing.ResourcePressureMode = "disabled"
				fixture.server.resolved.Sharing.ResourcePressure = nil
				configured := fixture.server.resolved.Sharing
				policy := testSharedRuntimeResourcePolicy()
				configured.ResourcePressureMode = "provider"
				configured.ResourcePressure = &policy
				return &configured
			},
		},
		{
			name: "stricter caller threshold cannot be narrowed",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.PressureThresholdBytes = 900
			}),
		},
		{
			name: "recovery threshold only differs",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.RecoveryThresholdBytes = 799
			}),
		},
		{
			name: "observation path only differs",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.ObservationPath = "/agents-infra/resources-v2"
			}),
		},
		{
			name: "observation timeout only differs",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.ObservationTimeoutMilliseconds++
			}),
		},
		{
			name: "eviction grace only differs",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.EvictionGraceSeconds++
			}),
		},
		{
			name: "pressure action only differs",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.PressureAction += "-v2"
			}),
		},
		{
			name: "unknown action only differs",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.UnknownAction += "-v2"
			}),
		},
		{
			name: "busy action only differs",
			configure: configuredSharedRuntimeResourcePolicyDifference(func(policy *PiRuntimeResourcePressure) {
				policy.BusyAction += "-v2"
			}),
		},
		{
			name: "disabled caller cannot use provider broker",
			configure: func(fixture *sharedBrokerAdmissionFixture) *PiRuntimeSharing {
				effectivePolicy := testSharedRuntimeResourcePolicy()
				fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
				fixture.server.resolved.Sharing.ResourcePressure = &effectivePolicy
				configured := fixture.server.resolved.Sharing
				configured.ResourcePressureMode = "disabled"
				configured.ResourcePressure = nil
				return &configured
			},
		},
		{
			name: "absent caller policy is not treated as compatible",
			configure: func(fixture *sharedBrokerAdmissionFixture) *PiRuntimeSharing {
				fixture.server.resolved.Sharing.ResourcePressureMode = "disabled"
				fixture.server.resolved.Sharing.ResourcePressure = nil
				return nil
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newSharedBrokerAdmissionFixture(t)
			fixture.server.resolved.Profile.Model = "Qwen-test"
			configured := testCase.configure(&fixture)
			fixture.hello.ConfiguredSharing = configured
			effective := fixture.server.resolved.Sharing
			fixture.server.record.Sharing = &effective
			runtimePID := fixture.server.record.Runtime.PID

			var observations atomic.Int32
			fixture.server.resources = sharedBrokerResourceDependencies{
				observe: func(context.Context, *http.Client, string, string, PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
					observations.Add(1)
					return observedSharedRuntimeResources(950, "busy"), nil
				},
				now: func() time.Time { return time.Unix(500, 0).UTC() },
			}

			connection, reader, done := startSharedBrokerFixtureConnection(t, &fixture)
			if err := writeSharedWireMessage(connection, sharedWireMessage{Type: "acquire", ClientKey: "policy-mismatch"}); err != nil {
				t.Fatal(err)
			}
			refusal, err := readSharedWireMessage(reader)
			if err != nil {
				t.Fatal(err)
			}
			_ = connection.Close()
			<-done

			if refusal.Type != "refused" || refusal.Code != "shared_runtime_resource_policy_mismatch" || refusal.Reason != "configured_resource_pressure_policy_differs" || refusal.MismatchField != "sharing.resource_pressure" {
				t.Fatalf("policy mismatch response=%#v", refusal)
			}
			if !reflect.DeepEqual(refusal.ConfiguredSharing, configured) || refusal.EffectiveSharing == nil || !reflect.DeepEqual(*refusal.EffectiveSharing, effective) {
				t.Fatalf("policy mismatch provenance configured=%#v effective=%#v", refusal.ConfiguredSharing, refusal.EffectiveSharing)
			}
			if observations.Load() != 0 {
				t.Fatalf("policy mismatch reached provider observation %d times", observations.Load())
			}
			if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 || fixture.server.record.Runtime.PID != runtimePID {
				t.Fatalf("policy mismatch changed ownership: leases=%d runtime=%#v", leaseCount, fixture.server.record.Runtime)
			}
		})
	}
}

func configuredSharedRuntimeResourcePolicyDifference(mutate func(*PiRuntimeResourcePressure)) func(*sharedBrokerAdmissionFixture) *PiRuntimeSharing {
	return func(fixture *sharedBrokerAdmissionFixture) *PiRuntimeSharing {
		effectivePolicy := testSharedRuntimeResourcePolicy()
		fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
		fixture.server.resolved.Sharing.ResourcePressure = &effectivePolicy
		configured := fixture.server.resolved.Sharing
		configuredPolicy := effectivePolicy
		mutate(&configuredPolicy)
		configured.ResourcePressure = &configuredPolicy
		return &configured
	}
}

// Production call site: sharedBrokerServer.handleConnection -> acquireLease.
// Pressure refuses only the new lease; the existing connection-bound lease and
// single broker-owned runtime survive, then the same runtime admits after the
// provider reports memory at the configured recovery threshold.
func TestSharedBrokerProductionPressureRefusalPreservesLeaseAndRecoversWithoutDuplicateRuntime(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	configureSharedBrokerResourceFixture(&fixture,
		observedSharedRuntimeResources(700, "idle"),
		observedSharedRuntimeResources(1000, "busy"),
		observedSharedRuntimeResources(800, "idle"),
	)
	runtimePID := fixture.server.record.Runtime.PID

	first, firstReader, firstDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(first, sharedWireMessage{Type: "acquire", ClientKey: "owner-a"}); err != nil {
		t.Fatal(err)
	}
	firstLease, err := readSharedWireMessage(firstReader)
	if err != nil || firstLease.Type != "lease" || firstLease.Runtime.PID != runtimePID {
		t.Fatalf("first lease=%#v err=%v", firstLease, err)
	}

	pressured, pressuredReader, pressuredDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(pressured, sharedWireMessage{Type: "acquire", ClientKey: "owner-b"}); err != nil {
		t.Fatal(err)
	}
	refusal, err := readSharedWireMessage(pressuredReader)
	if err != nil || refusal.Type != "refused" || refusal.Code != "shared_runtime_resource_pressure" || refusal.Resources == nil || refusal.Resources.State != "pressured" {
		t.Fatalf("pressure refusal=%#v err=%v", refusal, err)
	}
	_ = pressured.Close()
	<-pressuredDone
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 1 || fixture.server.record.Runtime.PID != runtimePID {
		t.Fatalf("pressure changed ownership: leases=%d runtime=%#v", leaseCount, fixture.server.record.Runtime)
	}

	recovered, recoveredReader, recoveredDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(recovered, sharedWireMessage{Type: "acquire", ClientKey: "owner-c"}); err != nil {
		t.Fatal(err)
	}
	recoveredLease, err := readSharedWireMessage(recoveredReader)
	if err != nil || recoveredLease.Type != "lease" || recoveredLease.Runtime.PID != runtimePID {
		t.Fatalf("recovered lease=%#v err=%v", recoveredLease, err)
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 2 {
		t.Fatalf("recovery lease count=%d want=2", leaseCount)
	}
	_ = first.Close()
	_ = recovered.Close()
	<-firstDone
	<-recoveredDone
}

// Production call site: sharedBrokerServer.handleConnection -> acquireLease.
// A provider read failure is unknown, never absence or healthy pressure.
func TestSharedBrokerProductionUnknownResourceObservationRefusesLease(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	policy := testSharedRuntimeResourcePolicy()
	fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
	fixture.server.resolved.Sharing.ResourcePressure = &policy
	fixture.hello.ConfiguredSharing = &fixture.server.resolved.Sharing
	fixture.server.resources = sharedBrokerResourceDependencies{
		observe: func(context.Context, *http.Client, string, string, PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
			return sharedRuntimeProviderResourceObservation{}, errors.New("fake read failure")
		},
		now: time.Now,
	}
	connection, reader, done := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(connection, sharedWireMessage{Type: "acquire", ClientKey: "unknown"}); err != nil {
		t.Fatal(err)
	}
	refusal, err := readSharedWireMessage(reader)
	if err != nil || refusal.Type != "refused" || refusal.Code != "shared_runtime_resource_unknown" || refusal.Resources == nil || refusal.Resources.State != "unknown" {
		t.Fatalf("unknown refusal=%#v err=%v", refusal, err)
	}
	_ = connection.Close()
	<-done
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
		t.Fatalf("unknown observation granted %d leases", leaseCount)
	}
}

// Production call site: sharedBrokerServer.handleConnection status case. This
// is the protocol-v7 consumer handoff, not a helper-only serialization test.
func TestSharedBrokerProductionStatusPublishesVersionedBusyFact(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	configureSharedBrokerResourceFixture(&fixture, observedSharedRuntimeResources(700, "busy"))
	connection, reader, done := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(connection, sharedWireMessage{Type: "status"}); err != nil {
		t.Fatal(err)
	}
	status, err := readSharedWireMessage(reader)
	if err != nil || status.Type != "status" || status.Resources == nil || status.Resources.Schema != SharedRuntimeResourceStatusSchema || status.Resources.State != "busy" || status.Resources.Inference.State != "busy" {
		t.Fatalf("status handoff=%#v err=%v", status, err)
	}
	_ = connection.Close()
	<-done
}

// Production call sites: sharedBrokerServer.handleConnection status and
// acquire cases. A read-only status request may observe recovery facts, but it
// must publish the pressure admission state still enforced by the broker until
// acquireLease applies recovery and the serve-loop cancels pressure eviction.
func TestSharedBrokerProductionStatusCannotBypassLatchedPressureRecovery(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	configureSharedBrokerResourceFixture(&fixture,
		observedSharedRuntimeResources(800, "idle"),
		observedSharedRuntimeResources(800, "idle"),
	)
	fixture.server.resolved.Sharing.ResourcePressure.EvictionGraceSeconds = 1
	fixture.hello.ConfiguredSharing = &fixture.server.resolved.Sharing
	fixture.server.pressureLatched = true
	fixture.server.hadLease = true
	runtimePID := fixture.server.record.Runtime.PID

	root := shortSharedRuntimeCache(t)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: root + "/status-pressure.sock", Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	fixture.server.listener = listener
	runtimeWait := &piProcessWait{done: make(chan struct{})}
	signals := make(chan os.Signal, 1)
	serveDone := make(chan error, 1)
	go func() { serveDone <- fixture.server.serve(runtimeWait, signals) }()
	fixture.server.notify("resource-pressure", false)
	waitForSharedBrokerResourceCondition(t, "pressure eviction armed", func() bool {
		return fixture.server.currentState() == "pressured"
	})

	statusConnection, statusReader, statusDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(statusConnection, sharedWireMessage{Type: "status"}); err != nil {
		t.Fatal(err)
	}
	status, err := readSharedWireMessage(statusReader)
	if err != nil || status.Type != "status" || status.State != "pressured" || status.Resources == nil || status.Resources.State != "pressured" || status.Resources.Admission != "refused" {
		t.Fatalf("latched status bypassed enforced pressure: response=%#v err=%v", status, err)
	}
	if !fixture.server.resourcePressureLatched() || fixture.server.currentState() != "pressured" {
		t.Fatalf("status mutated pressure owner: latched=%t state=%s", fixture.server.resourcePressureLatched(), fixture.server.currentState())
	}
	_ = statusConnection.Close()
	<-statusDone

	recoveredConnection, recoveredReader, recoveredDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(recoveredConnection, sharedWireMessage{Type: "acquire", ClientKey: "recovered-owner"}); err != nil {
		t.Fatal(err)
	}
	lease, err := readSharedWireMessage(recoveredReader)
	if err != nil || lease.Type != "lease" || lease.Runtime == nil || lease.Runtime.PID != runtimePID {
		t.Fatalf("recovery did not reuse broker-owned runtime: response=%#v err=%v", lease, err)
	}
	waitForSharedBrokerResourceCondition(t, "recovery applied and eviction cancelled", func() bool {
		return !fixture.server.resourcePressureLatched() && fixture.server.currentState() == "serving"
	})
	select {
	case err := <-serveDone:
		t.Fatalf("cancelled pressure eviction still drained broker: %v", err)
	case <-time.After(1200 * time.Millisecond):
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 1 || fixture.server.record.Runtime.PID != runtimePID {
		t.Fatalf("recovery changed ownership: leases=%d runtime=%#v", leaseCount, fixture.server.record.Runtime)
	}

	signals <- syscall.SIGTERM
	if err := <-serveDone; err != nil {
		t.Fatal(err)
	}
	_ = recoveredConnection.Close()
	<-recoveredDone
}

// Production call site: sharedBrokerServer.handleConnection -> acquireLease.
// Observation order, rather than HTTP completion order, owns admission. An
// older recovery snapshot must not clear a newer pressure latch or receive a
// lease after the newer request has been refused.
func TestSharedBrokerProductionStaleRecoveryObservationCannotClearNewerPressure(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	policy := testSharedRuntimeResourcePolicy()
	policy.ObservationTimeoutMilliseconds = 2_000
	fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
	fixture.server.resolved.Sharing.ResourcePressure = &policy
	fixture.hello.ConfiguredSharing = &fixture.server.resolved.Sharing

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var calls atomic.Int32
	fixture.server.resources = sharedBrokerResourceDependencies{
		observe: func(ctx context.Context, _ *http.Client, _, _ string, _ PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
			switch calls.Add(1) {
			case 1:
				close(firstStarted)
				select {
				case <-releaseFirst:
					return observedSharedRuntimeResources(800, "idle"), nil
				case <-ctx.Done():
					return sharedRuntimeProviderResourceObservation{}, ctx.Err()
				}
			case 2:
				return observedSharedRuntimeResources(1000, "busy"), nil
			default:
				return sharedRuntimeProviderResourceObservation{}, errors.New("unexpected fake provider observation")
			}
		},
		now: func() time.Time { return time.Unix(300, 0).UTC() },
	}

	stale, staleReader, staleDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(stale, sharedWireMessage{Type: "acquire", ClientKey: "stale-recovery"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("older recovery observation did not start")
	}

	newer, newerReader, newerDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(newer, sharedWireMessage{Type: "acquire", ClientKey: "newer-pressure"}); err != nil {
		t.Fatal(err)
	}
	pressureRefusal, err := readSharedWireMessage(newerReader)
	if err != nil || pressureRefusal.Type != "refused" || pressureRefusal.Code != "shared_runtime_resource_pressure" {
		t.Fatalf("newer pressure response=%#v err=%v", pressureRefusal, err)
	}
	if !fixture.server.resourcePressureLatched() {
		t.Fatal("newer pressure observation did not latch pressure")
	}
	close(releaseFirst)

	staleRefusal, err := readSharedWireMessage(staleReader)
	if err != nil || staleRefusal.Type != "refused" || staleRefusal.Code != "shared_runtime_resource_unknown" || staleRefusal.Resources == nil || staleRefusal.Resources.Reason != "resource_observation_stale" {
		t.Fatalf("stale recovery observation was not refused as unknown: response=%#v err=%v", staleRefusal, err)
	}
	if !fixture.server.resourcePressureLatched() {
		t.Fatal("stale recovery observation cleared newer pressure")
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
		t.Fatalf("stale recovery observation granted %d leases", leaseCount)
	}

	_ = stale.Close()
	_ = newer.Close()
	<-staleDone
	<-newerDone
}

// Production call site: sharedBrokerServer.handleConnection -> acquireLease.
// A provider snapshot can still be superseded after its read completes. The
// generation is rechecked under the same lock that reserves the lease, closing
// the observe-to-grant bypass rather than relying only on completion ordering.
func TestSharedBrokerProductionSupersededAdmissionCannotGrantBeforeLeaseReservation(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	configureSharedBrokerResourceFixture(&fixture,
		observedSharedRuntimeResources(800, "idle"),
		observedSharedRuntimeResources(1000, "busy"),
	)

	beforeReservation := make(chan struct{})
	releaseReservation := make(chan struct{})
	fixture.server.resources.beforeLeaseReservation = func(generation uint64) {
		if generation != 1 {
			return
		}
		close(beforeReservation)
		<-releaseReservation
	}

	stale, staleReader, staleDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(stale, sharedWireMessage{Type: "acquire", ClientKey: "superseded-before-reservation"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-beforeReservation:
	case <-time.After(time.Second):
		t.Fatal("first admission did not reach the lease reservation boundary")
	}

	newer, newerReader, newerDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(newer, sharedWireMessage{Type: "acquire", ClientKey: "newer-pressure"}); err != nil {
		t.Fatal(err)
	}
	pressureRefusal, err := readSharedWireMessage(newerReader)
	if err != nil || pressureRefusal.Type != "refused" || pressureRefusal.Code != "shared_runtime_resource_pressure" {
		t.Fatalf("newer pressure response=%#v err=%v", pressureRefusal, err)
	}
	close(releaseReservation)

	staleRefusal, err := readSharedWireMessage(staleReader)
	if err != nil || staleRefusal.Type != "refused" || staleRefusal.Code != "shared_runtime_resource_unknown" || staleRefusal.Resources == nil || staleRefusal.Resources.Reason != "resource_observation_stale" {
		t.Fatalf("superseded admission crossed lease reservation: response=%#v err=%v", staleRefusal, err)
	}
	if !fixture.server.resourcePressureLatched() {
		t.Fatal("superseded admission cleared newer pressure")
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
		t.Fatalf("superseded admission reserved %d leases", leaseCount)
	}

	_ = stale.Close()
	_ = newer.Close()
	<-staleDone
	<-newerDone
}

// Production call sites: sharedBrokerServer.handleConnection status and
// acquire cases. A status response whose provider snapshot was superseded by
// a newer pressure decision must remain unknown/refused instead of publishing
// the stale recovery as healthy/admitted.
func TestSharedBrokerProductionStaleStatusObservationCannotLaunderNewerPressure(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	policy := testSharedRuntimeResourcePolicy()
	policy.ObservationTimeoutMilliseconds = 2_000
	fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
	fixture.server.resolved.Sharing.ResourcePressure = &policy
	fixture.hello.ConfiguredSharing = &fixture.server.resolved.Sharing

	statusStarted := make(chan struct{})
	releaseStatus := make(chan struct{})
	var calls atomic.Int32
	fixture.server.resources = sharedBrokerResourceDependencies{
		observe: func(ctx context.Context, _ *http.Client, _, _ string, _ PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
			switch calls.Add(1) {
			case 1:
				close(statusStarted)
				select {
				case <-releaseStatus:
					return observedSharedRuntimeResources(800, "idle"), nil
				case <-ctx.Done():
					return sharedRuntimeProviderResourceObservation{}, ctx.Err()
				}
			case 2:
				return observedSharedRuntimeResources(1000, "busy"), nil
			default:
				return sharedRuntimeProviderResourceObservation{}, errors.New("unexpected fake provider observation")
			}
		},
		now: func() time.Time { return time.Unix(400, 0).UTC() },
	}

	statusConnection, statusReader, statusDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(statusConnection, sharedWireMessage{Type: "status"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-statusStarted:
	case <-time.After(time.Second):
		t.Fatal("older status observation did not start")
	}

	pressure, pressureReader, pressureDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(pressure, sharedWireMessage{Type: "acquire", ClientKey: "newer-pressure"}); err != nil {
		t.Fatal(err)
	}
	pressureRefusal, err := readSharedWireMessage(pressureReader)
	if err != nil || pressureRefusal.Type != "refused" || pressureRefusal.Code != "shared_runtime_resource_pressure" {
		t.Fatalf("newer pressure response=%#v err=%v", pressureRefusal, err)
	}
	close(releaseStatus)

	status, err := readSharedWireMessage(statusReader)
	if err != nil || status.Type != "status" || status.Resources == nil || status.Resources.State != SharedRuntimeResourceUnknown || status.Resources.Admission != SharedRuntimeAdmissionRefused || status.Resources.Reason != "resource_observation_stale" {
		t.Fatalf("stale status laundered newer pressure: response=%#v err=%v", status, err)
	}
	if !fixture.server.resourcePressureLatched() {
		t.Fatal("stale status observation cleared newer pressure")
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
		t.Fatalf("status/pressure race granted %d leases", leaseCount)
	}

	_ = statusConnection.Close()
	_ = pressure.Close()
	<-statusDone
	<-pressureDone
}

// Production call sites: sharedBrokerServer.handleConnection status and
// acquire cases. A status observation that completed healthy must be
// revalidated together with the broker snapshot immediately before the wire
// response is built; a newer pressure latch cannot be published as healthy.
func TestSharedBrokerProductionPressureCannotSupersedeStatusBeforePublication(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	runtimePID := fixture.server.record.Runtime.PID
	configureSharedBrokerResourceFixture(&fixture,
		observedSharedRuntimeResources(800, "idle"),
		observedSharedRuntimeResources(1000, "busy"),
	)

	beforePublication := make(chan struct{})
	releasePublication := make(chan struct{})
	fixture.server.resources.beforeStatusPublication = func(decision sharedBrokerResourceDecision) {
		if decision.statusGeneration != 1 {
			return
		}
		close(beforePublication)
		<-releasePublication
	}

	statusConnection, statusReader, statusDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(statusConnection, sharedWireMessage{Type: "status"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-beforePublication:
	case <-time.After(time.Second):
		t.Fatal("status observation did not reach the publication boundary")
	}

	pressureConnection, pressureReader, pressureDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(pressureConnection, sharedWireMessage{Type: "acquire", ClientKey: "newer-pressure"}); err != nil {
		t.Fatal(err)
	}
	pressureRefusal, err := readSharedWireMessage(pressureReader)
	if err != nil || pressureRefusal.Type != "refused" || pressureRefusal.Code != "shared_runtime_resource_pressure" {
		t.Fatalf("newer pressure response=%#v err=%v", pressureRefusal, err)
	}
	close(releasePublication)

	status, err := readSharedWireMessage(statusReader)
	if err != nil || status.Type != "status" || status.Resources == nil || status.Resources.State != SharedRuntimeResourceUnknown || status.Resources.Admission != SharedRuntimeAdmissionRefused || status.Resources.Reason != "resource_observation_stale" {
		t.Fatalf("superseded status was not published as explicit unknown/refused: response=%#v err=%v", status, err)
	}
	if !fixture.server.resourcePressureLatched() {
		t.Fatal("newer pressure observation did not remain latched")
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 || fixture.server.record.Runtime.PID != runtimePID {
		t.Fatalf("status/pressure publication race changed ownership: leases=%d runtime=%#v", leaseCount, fixture.server.record.Runtime)
	}

	_ = statusConnection.Close()
	_ = pressureConnection.Close()
	<-statusDone
	<-pressureDone
}

// Production call sites: sharedBrokerServer.handleConnection status and
// acquire cases. Healthy diagnostic polls cannot advance lease-admission
// invalidation; an acquire paused at the reservation boundary must still grant
// after repeated healthy status responses.
func TestSharedBrokerProductionHealthyStatusPollingCannotStarveAdmission(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	runtimePID := fixture.server.record.Runtime.PID
	policy := testSharedRuntimeResourcePolicy()
	fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
	fixture.server.resolved.Sharing.ResourcePressure = &policy
	fixture.hello.ConfiguredSharing = &fixture.server.resolved.Sharing
	var observations atomic.Int32
	fixture.server.resources = sharedBrokerResourceDependencies{
		observe: func(context.Context, *http.Client, string, string, PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
			observations.Add(1)
			return observedSharedRuntimeResources(800, "idle"), nil
		},
		now: func() time.Time { return time.Unix(600, 0).UTC() },
	}

	beforeReservation := make(chan struct{})
	releaseReservation := make(chan struct{})
	fixture.server.resources.beforeLeaseReservation = func(generation uint64) {
		if generation != 1 {
			return
		}
		close(beforeReservation)
		<-releaseReservation
	}

	acquireConnection, acquireReader, acquireDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(acquireConnection, sharedWireMessage{Type: "acquire", ClientKey: "healthy-owner"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-beforeReservation:
	case <-time.After(time.Second):
		t.Fatal("healthy acquire did not reach the lease reservation boundary")
	}

	for poll := 0; poll < 4; poll++ {
		statusConnection, statusReader, statusDone := startSharedBrokerFixtureConnection(t, &fixture)
		if err := writeSharedWireMessage(statusConnection, sharedWireMessage{Type: "status"}); err != nil {
			t.Fatal(err)
		}
		status, err := readSharedWireMessage(statusReader)
		if err != nil || status.Type != "status" || status.Resources == nil || status.Resources.State != SharedRuntimeResourceHealthy || status.Resources.Admission != SharedRuntimeAdmissionAdmitted {
			t.Fatalf("healthy status poll %d response=%#v err=%v", poll+1, status, err)
		}
		_ = statusConnection.Close()
		<-statusDone
	}
	close(releaseReservation)

	lease, err := readSharedWireMessage(acquireReader)
	if err != nil || lease.Type != "lease" || lease.Runtime == nil || lease.Runtime.PID != runtimePID {
		t.Fatalf("healthy status polling starved admission: response=%#v err=%v", lease, err)
	}
	if observations.Load() != 5 {
		t.Fatalf("provider observations=%d want=5", observations.Load())
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 1 {
		t.Fatalf("healthy admission lease count=%d want=1", leaseCount)
	}

	_ = acquireConnection.Close()
	<-acquireDone
}

// Production call sites: sharedBrokerServer.handleConnection status and
// acquire cases. Separating healthy diagnostics from admission freshness must
// not weaken the safety edge: direct pressure observed by status invalidates a
// healthy acquire before it can reserve a lease.
func TestSharedBrokerProductionPressuredStatusInvalidatesPendingAdmission(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	policy := testSharedRuntimeResourcePolicy()
	fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
	fixture.server.resolved.Sharing.ResourcePressure = &policy
	fixture.hello.ConfiguredSharing = &fixture.server.resolved.Sharing
	runtimePID := fixture.server.record.Runtime.PID
	var observations atomic.Int32
	fixture.server.resources = sharedBrokerResourceDependencies{
		observe: func(context.Context, *http.Client, string, string, PiRuntimeResourcePressure) (sharedRuntimeProviderResourceObservation, error) {
			switch observations.Add(1) {
			case 1:
				return observedSharedRuntimeResources(800, "idle"), nil
			case 2:
				return observedSharedRuntimeResources(1000, "busy"), nil
			default:
				return sharedRuntimeProviderResourceObservation{}, errors.New("unexpected fake provider observation")
			}
		},
		now: func() time.Time { return time.Unix(700, 0).UTC() },
	}

	beforeReservation := make(chan struct{})
	releaseReservation := make(chan struct{})
	fixture.server.resources.beforeLeaseReservation = func(generation uint64) {
		if generation != 1 {
			return
		}
		close(beforeReservation)
		<-releaseReservation
	}

	acquireConnection, acquireReader, acquireDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(acquireConnection, sharedWireMessage{Type: "acquire", ClientKey: "pending-healthy-owner"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-beforeReservation:
	case <-time.After(time.Second):
		t.Fatal("healthy acquire did not reach the lease reservation boundary")
	}

	statusConnection, statusReader, statusDone := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(statusConnection, sharedWireMessage{Type: "status"}); err != nil {
		t.Fatal(err)
	}
	status, err := readSharedWireMessage(statusReader)
	if err != nil || status.Type != "status" || status.Resources == nil || status.Resources.State != SharedRuntimeResourcePressured || status.Resources.Admission != SharedRuntimeAdmissionRefused {
		t.Fatalf("pressure status response=%#v err=%v", status, err)
	}
	close(releaseReservation)

	refusal, err := readSharedWireMessage(acquireReader)
	if err != nil || refusal.Type != "refused" || refusal.Code != "shared_runtime_resource_unknown" || refusal.Resources == nil || refusal.Resources.Reason != "resource_observation_stale" {
		t.Fatalf("pressured status did not invalidate pending admission: response=%#v err=%v", refusal, err)
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 || fixture.server.record.Runtime.PID != runtimePID {
		t.Fatalf("pressured status changed ownership: leases=%d runtime=%#v", leaseCount, fixture.server.record.Runtime)
	}

	_ = acquireConnection.Close()
	_ = statusConnection.Close()
	<-acquireDone
	<-statusDone
}

func TestSharedBrokerProductionStatusPreservesDrainingOverLatchedPressure(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.resolved.Profile.Model = "Qwen-test"
	configureSharedBrokerResourceFixture(&fixture, observedSharedRuntimeResources(800, "idle"))
	fixture.server.state = "draining"
	fixture.server.pressureLatched = true

	connection, reader, done := startSharedBrokerFixtureConnection(t, &fixture)
	if err := writeSharedWireMessage(connection, sharedWireMessage{Type: "status"}); err != nil {
		t.Fatal(err)
	}
	status, err := readSharedWireMessage(reader)
	if err != nil || status.Type != "status" || status.State != "draining" || status.Resources == nil || status.Resources.State != "draining" || status.Resources.Admission != "refused" {
		t.Fatalf("draining precedence was bypassed: response=%#v err=%v", status, err)
	}
	if !fixture.server.resourcePressureLatched() || fixture.server.currentState() != "draining" {
		t.Fatalf("status mutated draining pressure owner: latched=%t state=%s", fixture.server.resourcePressureLatched(), fixture.server.currentState())
	}
	_ = connection.Close()
	<-done
}

func waitForSharedBrokerResourceCondition(t *testing.T, name string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", name)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestSharedBrokerPressureDrainPreservesLeaseThenEvictsAfterFinalRelease(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	policy := testSharedRuntimeResourcePolicy()
	fixture.server.resolved.Sharing.ResourcePressureMode = "provider"
	fixture.server.resolved.Sharing.ResourcePressure = &policy
	fixture.server.pressureLatched = true
	lease := &SharedLeaseRecord{LeaseID: "lease-a"}
	connection := &sharedBrokerConnection{leaseID: lease.LeaseID}
	fixture.server.leases[lease.LeaseID] = lease
	fixture.server.hadLease = true

	root := shortSharedRuntimeCache(t)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: root + "/pressure.sock", Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	fixture.server.listener = listener
	runtimeWait := &piProcessWait{done: make(chan struct{})}
	signals := make(chan os.Signal)
	done := make(chan error, 1)
	go func() { done <- fixture.server.serve(runtimeWait, signals) }()
	fixture.server.notify("resource-pressure", false)
	select {
	case err := <-done:
		t.Fatalf("pressure drained while lease was held: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if fixture.server.currentState() != "serving" {
		t.Fatalf("pressure changed state while leased: %s", fixture.server.currentState())
	}
	fixture.server.releaseConnectionLease(connection)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("final lease release did not trigger configured pressure drain")
	}
	if fixture.server.currentState() != "draining" {
		t.Fatalf("post-release state=%s want=draining", fixture.server.currentState())
	}
	_ = listener.Close()
}
