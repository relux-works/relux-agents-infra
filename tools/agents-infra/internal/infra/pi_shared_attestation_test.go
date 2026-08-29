//go:build darwin

package infra

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

type sharedAttestationFixture struct {
	resolved sharedResolvedProfile
	state    PiStatePaths
	listener *net.UnixListener
	runtime  *exec.Cmd
	broker   SharedBrokerIdentity
	record   SharedRuntimeProcessRecord
}

func newSharedAttestationFixture(t *testing.T, extraRuntimeArgv ...string) sharedAttestationFixture {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	project := filepath.Join(root, "project")
	home := filepath.Join(root, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	portListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := portListener.Addr().(*net.TCPAddr).Port
	portListener.Close()
	target := buildSharedFakeRuntime(t, root)
	configuredArgv := []string{"serve", "--model", "Model"}
	body := validPiProfileWithArgv(t, "profile", target, port, configuredArgv, 2)
	body += `
[agents.pi.profiles.profile.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
max_segment_bytes = 1048576
max_segments = 7
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 35
`
	writePiProjectConfig(t, project, body)
	resolved, err := resolveSharedProfile(project, home, cache, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(resolved.Paths); err != nil {
		t.Fatal(err)
	}
	state, err := ResolvePiClientStatePaths(cache, resolved.Project, resolved.ProfileName, "RUN-attestation")
	if err != nil {
		t.Fatal(err)
	}
	runtimeArgv := append(append([]string(nil), resolved.Profile.Runtime.Argv...), extraRuntimeArgv...)
	runtime := exec.Command(target, runtimeArgv...)
	runtime.Dir = resolved.Paths.RuntimeCWD
	runtime.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if runtime.ProcessState == nil {
			_ = syscall.Kill(-runtime.Process.Pid, syscall.SIGKILL)
			_ = runtime.Wait()
		}
	})
	waitForSharedTest(t, 10*time.Second, func() bool {
		return checkSharedRuntimeModel(nil, resolved.Profile.BaseURL+resolved.Profile.Runtime.ReadinessPath, "Model") == nil
	}, "attestation fixture runtime did not become ready")
	runtimeObservation, err := inspectSharedProcess(runtime.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	targetIdentity, err := fileIdentity(target)
	if err != nil {
		t.Fatal(err)
	}
	self, err := inspectSharedProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	_, selfIdentity, err := ownResolvedExecutableIdentity()
	if err != nil {
		t.Fatal(err)
	}
	broker := SharedBrokerIdentity{
		PID: self.PID, PGID: self.PGID, SID: self.SID, StartTime: self.StartTime,
		UID: self.UID, ExecPath: self.ExecPath, Argv: append([]string(nil), self.Argv...),
		ExecutableIdentity: selfIdentity,
	}
	record := SharedRuntimeProcessRecord{
		PID: runtimeObservation.PID, PGID: runtimeObservation.PGID, StartTime: runtimeObservation.StartTime,
		UID: runtimeObservation.UID, PostExec: ProcessExecShape{ExecPath: runtimeObservation.ExecPath, Argv: append([]string(nil), runtimeObservation.Argv...)},
		CWD: resolved.Paths.RuntimeCWD, Endpoint: resolved.Profile.BaseURL,
		ExecutableIdentity: targetIdentity, ExecPlanDigest: SharedRuntimeExecPlanDigest(resolved.Profile, resolved.Paths.RuntimeCWD), Stage: "ready",
	}
	address, err := net.ResolveUnixAddr("unix", resolved.Paths.RendezvousSocket)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.ListenUnix("unix", address)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(resolved.Paths.RendezvousSocket, 0o600); err != nil {
		listener.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return sharedAttestationFixture{resolved: resolved, state: state, listener: listener, runtime: runtime, broker: broker, record: record}
}

func (fixture sharedAttestationFixture) answerOnce(t *testing.T, mutate func(*sharedWireMessage)) <-chan error {
	t.Helper()
	result := make(chan error, 1)
	go func() {
		connection, err := fixture.listener.AcceptUnix()
		if err != nil {
			result <- err
			return
		}
		defer connection.Close()
		reader := bufio.NewReaderSize(connection, sharedRuntimeMaxFrameBytes+1)
		if _, err := readSharedWireMessage(reader); err != nil {
			result <- err
			return
		}
		effective := fixture.resolved.Sharing
		response := sharedWireMessage{
			Type: "hello_ok", ProtocolVersion: SharedRuntimeProtocolVersion,
			RuntimeKey: fixture.resolved.RuntimeKey, ProfileDigest: fixture.resolved.ProfileDigest,
			Broker: &fixture.broker, Runtime: &fixture.record, EffectiveSharing: &effective,
		}
		if mutate != nil {
			mutate(&response)
		}
		result <- writeSharedWireMessage(connection, response)
	}()
	return result
}

func requireSharedAttestationCode(t *testing.T, err error, code string) {
	t.Helper()
	var shared *SharedRuntimeError
	if errors.As(err, &shared) {
		if shared.Code != code {
			t.Fatalf("error code=%q want=%q err=%v", shared.Code, code, err)
		}
		return
	}
	var launch *PiLaunchError
	if errors.As(err, &launch) && launch.Code == code {
		return
	}
	t.Fatalf("error=%v want code=%q", err, code)
}

func TestConnectAndAttestSharedRuntimeRejectsEveryGateDeleteAndNarrowWitness(t *testing.T) {
	wrongModelClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"object":"list","data":[{"id":"Wrong"}]}`)), Header: make(http.Header)}, nil
	})}
	tests := []struct {
		name           string
		code           string
		mutateSystem   func(*sharedRuntimeAttestationDependencies, sharedAttestationFixture)
		mutateResponse func(*sharedWireMessage)
		httpClient     *http.Client
	}{
		{
			name: "peer uid root narrowing", code: "broker_peer_untrusted",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, _ sharedAttestationFixture) {
				original := system.peerIdentity
				system.peerIdentity = func(connection *net.UnixConn) (uint32, int, error) {
					_, pid, err := original(connection)
					return 0, pid, err
				}
			},
		},
		{
			name: "peer zombie narrowing", code: "broker_identity_unavailable",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, _ sharedAttestationFixture) {
				original := system.inspectProcess
				system.inspectProcess = func(pid int) (sharedProcessObservation, error) {
					observation, err := original(pid)
					if pid == os.Getpid() {
						observation.PStat = darwinProcessStateZombie
					}
					return observation, err
				}
			},
		},
		{
			name: "broker executable same inode wrong device", code: "broker_executable_identity_mismatch",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, _ sharedAttestationFixture) {
				original := system.processExecIdentity
				system.processExecIdentity = func(observation sharedProcessObservation) (FileIdentity, error) {
					identity, err := original(observation)
					identity.Dev++
					return identity, err
				}
			},
		},
		{
			name: "broker executable same device wrong inode", code: "broker_executable_identity_mismatch",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, _ sharedAttestationFixture) {
				original := system.processExecIdentity
				system.processExecIdentity = func(observation sharedProcessObservation) (FileIdentity, error) {
					identity, err := original(observation)
					identity.Ino++
					return identity, err
				}
			},
		},
		{
			name: "broker build same inode wrong device", code: "broker_build_identity_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.Broker.ExecutableIdentity.Dev++
			},
		},
		{
			name: "broker start time zero", code: "broker_identity_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.Broker.StartTime = ProcessStartTime{}
			},
		},
		{
			name: "future protocol version range narrowing", code: "broker_protocol_version_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.ProtocolVersion = SharedRuntimeProtocolVersion + 1
			},
		},
		{
			name: "past protocol version range narrowing", code: "broker_protocol_version_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.ProtocolVersion = SharedRuntimeProtocolVersion - 1
			},
		},
		{
			name: "empty runtime key", code: "shared_runtime_identity_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.RuntimeKey = ""
			},
		},
		{
			name: "profile digest", code: "shared_runtime_profile_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.ProfileDigest = strings.Repeat("0", 64)
			},
		},
		{
			name: "empty profile digest", code: "shared_runtime_profile_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.ProfileDigest = ""
			},
		},
		{
			name: "empty endpoint", code: "shared_runtime_endpoint_mismatch",
			mutateResponse: func(response *sharedWireMessage) {
				response.Runtime.Endpoint = ""
			},
		},
		{
			name: "runtime executable same inode wrong device", code: "runtime_executable_invalid",
			mutateResponse: func(response *sharedWireMessage) {
				response.Runtime.ExecutableIdentity.Dev++
			},
		},
		{
			name: "runtime process uid root narrowing", code: "runtime_identity_mismatch",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, fixture sharedAttestationFixture) {
				original := system.inspectProcess
				system.inspectProcess = func(pid int) (sharedProcessObservation, error) {
					observation, err := original(pid)
					if pid == fixture.runtime.Process.Pid {
						observation.UID = 0
					}
					return observation, err
				}
			},
		},
		{
			name: "runtime process start time zero", code: "runtime_identity_mismatch",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, fixture sharedAttestationFixture) {
				original := system.inspectProcess
				system.inspectProcess = func(pid int) (sharedProcessObservation, error) {
					observation, err := original(pid)
					if pid == fixture.runtime.Process.Pid {
						observation.StartTime = ProcessStartTime{}
					}
					return observation, err
				}
			},
		},
		{
			name: "runtime process empty executable path", code: "runtime_identity_mismatch",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, fixture sharedAttestationFixture) {
				original := system.inspectProcess
				system.inspectProcess = func(pid int) (sharedProcessObservation, error) {
					observation, err := original(pid)
					if pid == fixture.runtime.Process.Pid {
						observation.ExecPath = ""
					}
					return observation, err
				}
			},
		},
		{
			name: "runtime process argv", code: "runtime_identity_mismatch",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, fixture sharedAttestationFixture) {
				original := system.inspectProcess
				system.inspectProcess = func(pid int) (sharedProcessObservation, error) {
					observation, err := original(pid)
					if pid == fixture.runtime.Process.Pid {
						observation.Argv = append(observation.Argv, "--forged")
					}
					return observation, err
				}
			},
		},
		{
			name: "runtime zombie narrowing", code: "runtime_exited_early",
			mutateSystem: func(system *sharedRuntimeAttestationDependencies, fixture sharedAttestationFixture) {
				original := system.inspectProcess
				system.inspectProcess = func(pid int) (sharedProcessObservation, error) {
					observation, err := original(pid)
					if pid == fixture.runtime.Process.Pid {
						observation.PStat = darwinProcessStateZombie
					}
					return observation, err
				}
			},
		},
		{
			name: "hello effective sharing absent", code: "protocol_violation",
			mutateResponse: func(response *sharedWireMessage) {
				response.EffectiveSharing = nil
			},
		},
		{name: "model discovery", code: "runtime_model_unavailable", httpClient: wrongModelClient},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newSharedAttestationFixture(t)
			originalSystem := sharedRuntimeAttestationSystem
			if testCase.mutateSystem != nil {
				mutated := originalSystem
				testCase.mutateSystem(&mutated, fixture)
				sharedRuntimeAttestationSystem = mutated
				t.Cleanup(func() { sharedRuntimeAttestationSystem = originalSystem })
			}
			served := fixture.answerOnce(t, testCase.mutateResponse)
			var attested *sharedAttestedRuntime
			var err error
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				attested, err = connectAndAttestSharedRuntime(fixture.resolved, fixture.state, "RUN-attestation", testCase.httpClient, time.Second)
			}()
			if recovered != nil {
				t.Fatalf("attestation gate panicked instead of refusing: %v", recovered)
			}
			if attested != nil {
				attested.close()
			}
			requireSharedAttestationCode(t, err, testCase.code)
			select {
			case <-served:
			case <-time.After(time.Second):
				t.Fatal("fixture broker did not observe the refused connection")
			}
		})
	}
}

func TestConnectAndAttestSharedRuntimeExactProtocolVersionReportsOnlyThePassedGateSet(t *testing.T) {
	fixture := newSharedAttestationFixture(t)
	served := fixture.answerOnce(t, nil)
	attested, err := connectAndAttestSharedRuntime(fixture.resolved, fixture.state, "RUN-attestation", nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer attested.close()
	requireExactSharedRuntimeAttestation(t, attested.gates)
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}

func requireExactSharedRuntimeAttestation(t *testing.T, outcomes []SharedRuntimeGateOutcome) {
	t.Helper()
	want := []string{
		"peer_uid", "peer_pid_liveness", "broker_executable", "broker_build",
		"broker_start_time", "protocol_version", "runtime_key", "profile_digest",
		"endpoint", "runtime_executable", "runtime_process", "runtime_liveness",
		"model_discovery",
	}
	if len(outcomes) != len(want) {
		t.Fatalf("attestation outcomes=%#v want exact gates=%q", outcomes, want)
	}
	for index, outcome := range outcomes {
		if outcome.Gate != want[index] || outcome.Outcome != "passed" || outcome.Source != "attested" {
			t.Fatalf("attestation outcome[%d]=%#v want gate=%q passed/attested", index, outcome, want[index])
		}
	}
}

func TestSharedRuntimeForceStopCannotBypassReachableBrokerAttestation(t *testing.T) {
	fixture := newSharedAttestationFixture(t)
	served := fixture.answerOnce(t, func(response *sharedWireMessage) { response.ProfileDigest = strings.Repeat("0", 64) })
	_, err := StopSharedRuntime(SharedRuntimeOperatorOptions{
		ProjectDir: fixture.resolved.Project, HomeDir: fixture.resolved.HomeDir,
		CacheRoot: fixture.resolved.Paths.CanonicalCacheRoot, Profile: fixture.resolved.ProfileName,
	}, true, time.Second)
	requireSharedAttestationCode(t, err, "shared_runtime_profile_mismatch")
	if err := <-served; err != nil {
		t.Fatal(err)
	}
	if observation, err := inspectSharedProcessKernel(fixture.runtime.Process.Pid); err != nil || !observation.live() {
		t.Fatalf("force stop bypassed failed attestation: observation=%#v err=%v", observation, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
