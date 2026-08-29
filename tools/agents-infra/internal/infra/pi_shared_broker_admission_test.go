//go:build darwin

package infra

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

type sharedBrokerAdmissionFixture struct {
	server      *sharedBrokerServer
	hello       sharedWireMessage
	observation sharedProcessObservation
	identity    FileIdentity
}

func newSharedBrokerAdmissionFixture(t *testing.T) sharedBrokerAdmissionFixture {
	t.Helper()
	_, _, _, resolved := newSharedIntegrationProfile(t)
	identity := FileIdentity{Dev: 7, Ino: 11, Size: 13, Mode: 0o755}
	observation := sharedProcessObservation{
		PID:       4242,
		PGID:      4242,
		SID:       4242,
		UID:       uint32(os.Geteuid()),
		StartTime: ProcessStartTime{Seconds: 17, Microseconds: 19},
		ExecPath:  "/test/shared-client",
		Argv:      []string{"/test/shared-client"},
	}
	record := &SharedBrokerRecord{
		Stage: "serving",
		State: "serving",
		Broker: SharedBrokerIdentity{
			PID: observation.PID, PGID: observation.PGID, SID: observation.SID,
			UID: observation.UID, StartTime: observation.StartTime,
			ExecPath: observation.ExecPath, Argv: append([]string(nil), observation.Argv...),
			ExecutableIdentity: identity,
		},
		RuntimeKey: resolved.RuntimeKey, ProfileDigest: resolved.ProfileDigest,
		Runtime: &SharedRuntimeProcessRecord{PID: 5252, Endpoint: resolved.Profile.BaseURL},
	}
	ledger := newSharedRuntimeRestartLedger(resolved.RuntimeKey, resolved.ProfileDigest)
	server := newSharedBrokerServer(resolved, record, &ledger, nil)
	server.admission = sharedBrokerAdmissionDependencies{
		peerIdentity: func(*net.UnixConn) (uint32, int, error) {
			return observation.UID, observation.PID, nil
		},
		inspectProcess: func(int) (sharedProcessObservation, error) {
			return observation, nil
		},
		processExecIdentity: func(sharedProcessObservation) (FileIdentity, error) {
			return identity, nil
		},
	}
	sharing := resolved.Sharing
	return sharedBrokerAdmissionFixture{
		server: server, observation: observation, identity: identity,
		hello: sharedWireMessage{
			Type: "hello", ProtocolVersion: SharedRuntimeProtocolVersion,
			ClientPID: observation.PID, RuntimeKey: resolved.RuntimeKey,
			ProfileDigest: resolved.ProfileDigest, RunID: "RUN-broker-admission",
			ConfiguredSharing: &sharing,
		},
	}
}

func TestSharedBrokerAttestClientRejectsEveryGateDeleteAndNarrowWitness(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		mutateSystem func(*sharedBrokerAdmissionDependencies)
		mutateHello  func(*sharedWireMessage)
	}{
		{
			name: "peer uid root narrowing", code: "broker_peer_untrusted",
			mutateSystem: func(system *sharedBrokerAdmissionDependencies) {
				original := system.peerIdentity
				system.peerIdentity = func(connection *net.UnixConn) (uint32, int, error) {
					_, pid, err := original(connection)
					return 0, pid, err
				}
			},
		},
		{
			name: "announced client pid zero narrowing", code: "broker_identity_mismatch",
			mutateHello: func(hello *sharedWireMessage) { hello.ClientPID = 0 },
		},
		{
			name: "client zombie narrowing", code: "broker_identity_unavailable",
			mutateSystem: func(system *sharedBrokerAdmissionDependencies) {
				original := system.inspectProcess
				system.inspectProcess = func(pid int) (sharedProcessObservation, error) {
					observation, err := original(pid)
					observation.PStat = darwinProcessStateZombie
					return observation, err
				}
			},
		},
		{
			name: "client executable same inode wrong device", code: "broker_executable_identity_mismatch",
			mutateSystem: func(system *sharedBrokerAdmissionDependencies) {
				original := system.processExecIdentity
				system.processExecIdentity = func(observation sharedProcessObservation) (FileIdentity, error) {
					identity, err := original(observation)
					identity.Dev++
					return identity, err
				}
			},
		},
		{
			name: "client executable same device wrong inode", code: "broker_executable_identity_mismatch",
			mutateSystem: func(system *sharedBrokerAdmissionDependencies) {
				original := system.processExecIdentity
				system.processExecIdentity = func(observation sharedProcessObservation) (FileIdentity, error) {
					identity, err := original(observation)
					identity.Ino++
					return identity, err
				}
			},
		},
		{
			name: "future protocol version range narrowing", code: "broker_protocol_version_mismatch",
			mutateHello: func(hello *sharedWireMessage) { hello.ProtocolVersion = SharedRuntimeProtocolVersion + 1 },
		},
		{
			name: "past protocol version range narrowing", code: "broker_protocol_version_mismatch",
			mutateHello: func(hello *sharedWireMessage) { hello.ProtocolVersion = SharedRuntimeProtocolVersion - 1 },
		},
		{
			name: "empty runtime key", code: "shared_runtime_identity_mismatch",
			mutateHello: func(hello *sharedWireMessage) { hello.RuntimeKey = "" },
		},
		{
			name: "empty profile digest", code: "shared_runtime_profile_mismatch",
			mutateHello: func(hello *sharedWireMessage) { hello.ProfileDigest = "" },
		},
	}

	t.Run("exact protocol version control", func(t *testing.T) {
		fixture := newSharedBrokerAdmissionFixture(t)
		observation, err := fixture.server.attestClient(nil, fixture.hello)
		if err != nil || !reflect.DeepEqual(observation, fixture.observation) {
			t.Fatalf("valid broker admission observation=%#v err=%v", observation, err)
		}
	})

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newSharedBrokerAdmissionFixture(t)
			if testCase.mutateSystem != nil {
				testCase.mutateSystem(&fixture.server.admission)
			}
			if testCase.mutateHello != nil {
				testCase.mutateHello(&fixture.hello)
			}
			observation, err := fixture.server.attestClient(nil, fixture.hello)
			requireSharedAttestationCode(t, err, testCase.code)
			if !reflect.DeepEqual(observation, sharedProcessObservation{}) {
				t.Fatalf("refused client returned observation=%#v", observation)
			}
			if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
				t.Fatalf("refused client was granted %d leases", leaseCount)
			}
		})
	}
}

func TestSharedBrokerAcquireLeaseRefusesDrainingBeforeGrant(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	fixture.server.state = "draining"
	lease, err := fixture.server.acquireLease(
		sharedWireMessage{Type: "acquire", ClientKey: "client"},
		fixture.hello,
		fixture.observation,
	)
	requireSharedAttestationCode(t, err, "shared_runtime_shutting_down")
	if lease != nil {
		t.Fatalf("draining broker granted lease=%#v", lease)
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
		t.Fatalf("draining broker retained %d leases", leaseCount)
	}
}

func TestSharedBrokerProductionConnectionRejectsWireFrameBoundWidening(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	serverConnection, clientConnection := sharedBrokerUnixConnectionPair(t)
	wrapped := &sharedBrokerConnection{
		connection: serverConnection,
		reader:     bufio.NewReaderSize(serverConnection, sharedRuntimeMaxFrameBytes+1),
	}
	fixture.server.connections[wrapped] = true
	done := make(chan struct{})
	go func() {
		fixture.server.handleConnection(wrapped)
		close(done)
	}()

	raw, err := marshalSharedBrokerOversizeHello(fixture.hello)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := clientConnection.Write(raw); err != nil {
		t.Fatal(err)
	}
	_ = clientConnection.SetReadDeadline(time.Now().Add(3 * time.Second))
	response, readErr := readSharedWireMessage(bufio.NewReaderSize(clientConnection, sharedRuntimeMaxFrameBytes+1))
	_ = clientConnection.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("broker handler did not finish after bounded-frame attack")
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	if response.Type != "refused" || response.Code != "protocol_violation" {
		t.Fatalf("oversize hello response=%#v", response)
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
		t.Fatalf("oversize hello was granted %d leases", leaseCount)
	}
}

// Production call site: sharedBrokerServer.handleConnection. Closing the real
// AF_UNIX peer without a release frame must run the handler defer and remove
// both the in-memory lease and its persisted mirror.
func TestSharedBrokerHandleConnectionReleasesLeaseAfterAbruptClientDeath(t *testing.T) {
	fixture := newSharedBrokerAdmissionFixture(t)
	serverConnection, clientConnection := sharedBrokerUnixConnectionPair(t)
	wrapped := &sharedBrokerConnection{
		connection: serverConnection,
		reader:     bufio.NewReaderSize(serverConnection, sharedRuntimeMaxFrameBytes+1),
	}
	fixture.server.connections[wrapped] = true
	done := make(chan struct{})
	go func() {
		fixture.server.handleConnection(wrapped)
		close(done)
	}()
	clientReader := bufio.NewReaderSize(clientConnection, sharedRuntimeMaxFrameBytes+1)
	if err := writeSharedWireMessage(clientConnection, fixture.hello); err != nil {
		t.Fatal(err)
	}
	if message, err := readSharedWireMessage(clientReader); err != nil || message.Type != "hello_ok" {
		t.Fatalf("hello response=%#v err=%v", message, err)
	}
	if err := writeSharedWireMessage(clientConnection, sharedWireMessage{Type: "acquire", ClientKey: "abrupt-client"}); err != nil {
		t.Fatal(err)
	}
	leaseMessage, err := readSharedWireMessage(clientReader)
	if err != nil || leaseMessage.Type != "lease" || leaseMessage.LeaseID == "" {
		t.Fatalf("lease response=%#v err=%v", leaseMessage, err)
	}
	mirror := filepath.Join(fixture.server.resolved.Paths.LeasesDir, leaseMessage.LeaseID+".json")
	if _, err := os.Stat(mirror); err != nil {
		t.Fatalf("lease mirror was not published: %v", err)
	}
	if err := clientConnection.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handleConnection did not observe abrupt client death")
	}
	if leaseCount, _ := fixture.server.leaseFacts(); leaseCount != 0 {
		t.Fatalf("handleConnection retained %d lease(s) after client death", leaseCount)
	}
	if _, err := os.Stat(mirror); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lease mirror survived abrupt client death: %v", err)
	}
}

func sharedBrokerUnixConnectionPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	path := root + "/broker.sock"
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	client, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	server, err := listener.AcceptUnix()
	if err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	return server, client
}

func marshalSharedBrokerOversizeHello(hello sharedWireMessage) ([]byte, error) {
	var buffer bytes.Buffer
	if err := writeSharedWireMessage(&buffer, hello); err != nil {
		return nil, err
	}
	plain := bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'})
	padding := bytes.Repeat([]byte{' '}, sharedRuntimeMaxFrameBytes-len(plain))
	raw := append(append(append([]byte(nil), plain[:len(plain)-1]...), padding...), '}', '\n')
	if len(raw) != sharedRuntimeMaxFrameBytes+1 {
		return nil, errors.New("oversize broker witness has the wrong length")
	}
	return raw, nil
}

func TestRunSharedRuntimeBrokerRefusesRecomputedRuntimeKeyAtProductionEntry(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	last := byte('0')
	if resolved.RuntimeKey[len(resolved.RuntimeKey)-1] == last {
		last = '1'
	}
	wrongKey := resolved.RuntimeKey[:len(resolved.RuntimeKey)-1] + string(last)
	wrongPaths, err := ResolveSharedRuntimePaths(cache, wrongKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(wrongPaths); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "runtime", "broker", "--runtime-key", wrongKey, "--profile-project", project, "--profile", "profile")
	command.Env = append(os.Environ(), "HOME="+home)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	output, err := command.CombinedOutput()
	if err == nil || !bytes.Contains(output, []byte(`"code":"shared_runtime_identity_mismatch"`)) {
		t.Fatalf("broker recomputed-key gate err=%v output=%s", err, output)
	}
	for _, path := range []string{wrongPaths.BrokerState, wrongPaths.RendezvousSocket} {
		if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("recomputed-key refusal left side effect at %s: %v", path, statErr)
		}
	}
}

func TestReclaimSharedRuntimeRejectsUIDAndPGIDGateNarrowingBeforeSignal(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root cannot express the uid==0 narrowing witness")
	}
	baseObservation := sharedProcessObservation{
		PID: 3131, PGID: 4141, UID: uint32(os.Geteuid()),
		StartTime: ProcessStartTime{Seconds: 23, Microseconds: 29},
		ExecPath:  "/test/runtime", Argv: []string{"/test/runtime", "serve"},
	}
	record := SharedRuntimeProcessRecord{
		PID: baseObservation.PID, PGID: baseObservation.PGID,
		UID: baseObservation.UID, StartTime: baseObservation.StartTime,
		PreExec:  ProcessExecShape{ExecPath: baseObservation.ExecPath, Argv: append([]string(nil), baseObservation.Argv...)},
		PostExec: ProcessExecShape{ExecPath: baseObservation.ExecPath, Argv: append([]string(nil), baseObservation.Argv...)},
		Endpoint: "http://127.0.0.1:1/v1",
	}
	tests := []struct {
		name   string
		mutate func(*sharedProcessObservation, *sharedProcessObservation)
	}{
		{
			name:   "reclaim uid root narrowing",
			mutate: func(kernel, _ *sharedProcessObservation) { kernel.UID = 0 },
		},
		{
			name:   "reclaim pgid zero narrowing",
			mutate: func(_, full *sharedProcessObservation) { full.PGID = 0 },
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			kernel, full := baseObservation, baseObservation
			testCase.mutate(&kernel, &full)
			signalled := false
			original := sharedRuntimeReclaimSystem
			sharedRuntimeReclaimSystem = sharedRuntimeReclaimDependencies{
				inspectKernel:  func(int) (sharedProcessObservation, error) { return kernel, nil },
				inspectProcess: func(int) (sharedProcessObservation, error) { return full, nil },
				kill: func(int, syscall.Signal) error {
					signalled = true
					return nil
				},
				waitGone:         func(*SharedRuntimeProcessRecord, time.Duration) error { return nil },
				waitEndpointFree: func(string, time.Duration) error { return nil },
			}
			t.Cleanup(func() { sharedRuntimeReclaimSystem = original })
			err := reclaimSharedRuntime(&record, 1)
			requireSharedAttestationCode(t, err, "shared_runtime_orphan_unidentifiable")
			if signalled {
				t.Fatal("untrusted reclaim evidence reached process-group signal")
			}
		})
	}
}
