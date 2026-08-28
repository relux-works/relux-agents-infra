//go:build darwin

package infra

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type SharedRuntimeBrokerOptions struct {
	RuntimeKey     string
	ProfileProject string
	ProfileName    string
	HomeDir        string
	CacheRoot      string
	Environ        []string
	HTTPClient     *http.Client
	Signals        <-chan os.Signal
}

type sharedWireMessage struct {
	Type                string                      `json:"type"`
	Code                string                      `json:"code,omitempty"`
	Reason              string                      `json:"reason,omitempty"`
	ProtocolVersion     int                         `json:"protocol_version,omitempty"`
	ClientPID           int                         `json:"client_pid,omitempty"`
	ClientExec          FileIdentity                `json:"client_exec,omitempty"`
	RuntimeKey          string                      `json:"runtime_key,omitempty"`
	ProfileDigest       string                      `json:"profile_digest,omitempty"`
	ProjectKey          string                      `json:"project_key,omitempty"`
	ProfileKey          string                      `json:"profile_key,omitempty"`
	RunID               string                      `json:"run_id,omitempty"`
	ConfiguredSharing   *PiRuntimeSharing           `json:"sharing_configured,omitempty"`
	EffectiveSharing    *PiRuntimeSharing           `json:"sharing_effective,omitempty"`
	Broker              *SharedBrokerIdentity       `json:"broker,omitempty"`
	Runtime             *SharedRuntimeProcessRecord `json:"runtime,omitempty"`
	Lease               *SharedLeaseRecord          `json:"lease,omitempty"`
	Leases              []SharedLeaseStatus         `json:"leases,omitempty"`
	LeaseID             string                      `json:"lease_id,omitempty"`
	LeaseCount          int                         `json:"lease_count,omitempty"`
	ClientKey           string                      `json:"client_key,omitempty"`
	RequestedAt         time.Time                   `json:"requested_at,omitempty"`
	Force               bool                        `json:"force,omitempty"`
	TimeoutSeconds      int                         `json:"timeout_seconds,omitempty"`
	State               string                      `json:"state,omitempty"`
	Stage               string                      `json:"stage,omitempty"`
	EffectiveMaxLeases  int                         `json:"effective_max_leases,omitempty"`
	ConfiguredMaxLeases int                         `json:"configured_max_leases,omitempty"`
	BrokerPID           int                         `json:"broker_pid,omitempty"`
	BrokerStartTime     *ProcessStartTime           `json:"broker_start_time,omitempty"`
}

type SharedLeaseStatus struct {
	SharedLeaseRecord
	State        string        `json:"state"`
	Age          time.Duration `json:"age"`
	HeartbeatAge time.Duration `json:"heartbeat_age"`
}

type sharedBrokerLock struct {
	file *os.File
}

func openSharedBrokerLock(path string) (*sharedBrokerLock, error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	file := os.NewFile(uintptr(fd), path)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Nlink != 1 || stat.Mode&0o777 != 0o600 {
		if err == nil {
			err = errors.New("broker.lock must be a mode-0600 single-link regular file")
		}
		file.Close()
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, &SharedRuntimeError{Code: "broker_election_lost", Err: err}
		}
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	return &sharedBrokerLock{file: file}, nil
}

func (lock *sharedBrokerLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	_ = unix.Flock(int(lock.file.Fd()), unix.LOCK_UN)
	return lock.file.Close()
}

func RunSharedRuntimeBroker(options SharedRuntimeBrokerOptions) (result error) {
	// B1 is intentionally the first operation in this entry point.
	sessionID, err := unix.Getsid(0)
	if err != nil || sessionID != os.Getpid() {
		if err == nil {
			err = fmt.Errorf("getsid(0)=%d getpid()=%d", sessionID, os.Getpid())
		}
		return sharedRuntimeError("broker_not_session_leader", err)
	}

	paths, err := ResolveSharedRuntimePaths(options.CacheRoot, options.RuntimeKey)
	if err != nil {
		return err
	}
	lock, err := openSharedBrokerLock(paths.BrokerLock)
	if err != nil {
		return err
	}
	defer lock.Close()

	predecessor, predecessorPresent, err := readSharedBrokerRecord(paths.BrokerState)
	if err != nil {
		return err
	}
	self, err := inspectSharedProcess(os.Getpid())
	if err != nil {
		return sharedRuntimeError("broker_identity_unavailable", err)
	}
	_, selfFile, err := ownResolvedExecutableIdentity()
	if err != nil {
		return sharedRuntimeError("broker_identity_unavailable", err)
	}
	record := SharedBrokerRecord{
		Stage:           "elected",
		State:           "starting",
		ProtocolVersion: SharedRuntimeProtocolVersion,
		Broker: SharedBrokerIdentity{
			PID: self.PID, PGID: self.PGID, SID: self.SID, StartTime: self.StartTime,
			UID: self.UID, ExecPath: self.ExecPath, Argv: append([]string(nil), self.Argv...),
			ExecutableIdentity: selfFile,
		},
		RuntimeKeyClaimed: options.RuntimeKey,
	}
	if predecessorPresent && predecessor.Runtime != nil {
		inherited := *predecessor.Runtime
		inherited.Stage = "inherited-unreclaimed"
		record.Runtime = &inherited
	}
	if err := writeSharedJSONAtomic(paths.BrokerState, record); err != nil {
		return err
	}
	recordPublished := true
	runtimeOwned := false
	runtimeReaped := false
	defer func() {
		if result == nil || !recordPublished {
			return
		}
		if record.Runtime != nil && !runtimeReaped {
			return
		}
		if !runtimeOwned || runtimeReaped {
			_ = os.Remove(paths.BrokerState)
		}
	}()

	resolved, err := resolveSharedProfile(options.ProfileProject, options.HomeDir, options.CacheRoot, options.ProfileName)
	if err != nil {
		return err
	}
	if resolved.RuntimeKey != options.RuntimeKey {
		return sharedRuntimeError("shared_runtime_identity_mismatch", errors.New("broker recomputed a different runtime key"))
	}
	if record.Runtime != nil {
		if err := reclaimSharedRuntime(record.Runtime, resolved.Profile.Runtime.ShutdownTimeoutSeconds); err != nil {
			return err
		}
		record.Runtime = nil
		runtimeReaped = true
		if err := writeSharedJSONAtomic(paths.BrokerState, record); err != nil {
			return err
		}
	}
	if err := removeSharedLeaseMirrors(paths.LeasesDir); err != nil {
		return err
	}
	if err := removeStaleRendezvous(paths.RendezvousSocket); err != nil {
		return err
	}
	record.Stage = "composed"
	record.RuntimeKey = resolved.RuntimeKey
	record.ProfileDigest = resolved.ProfileDigest
	record.Endpoint = resolved.Profile.BaseURL
	effective := resolved.Sharing
	record.Sharing = &effective
	if err := writeSharedJSONAtomic(paths.BrokerState, record); err != nil {
		return err
	}
	ledger, err := readSharedRuntimeRestartLedger(paths.RestartLedger, resolved.RuntimeKey, resolved.ProfileDigest)
	if err != nil {
		return err
	}
	if err := sharedRuntimeBeginAttempt(&ledger, time.Now().UTC()); err != nil {
		return err
	}
	if err := writeSharedRuntimeRestartLedger(paths.RestartLedger, ledger); err != nil {
		return err
	}
	if delay := sharedRuntimeRestartDelay(ledger, time.Now().UTC()); delay > 0 {
		timer := time.NewTimer(delay)
		<-timer.C
	}
	if err := preflightPiListener(resolved.Profile.BaseURL); err != nil {
		var launch *PiLaunchError
		if errors.As(err, &launch) {
			return sharedRuntimeError(launch.Code, err)
		}
		return err
	}

	runtimeReady := false
	defer func() {
		if result == nil || runtimeReady {
			return
		}
		decision := sharedRuntimeRecordFailure(&ledger, resolved.Sharing, time.Now().UTC())
		if ledgerErr := writeSharedRuntimeRestartLedger(paths.RestartLedger, ledger); ledgerErr != nil {
			result = ledgerErr
			return
		}
		if decision.Quarantined {
			result = &SharedRuntimeError{Code: "shared_runtime_quarantined", Details: map[string]any{"restart_count": ledger.RestartCount, "quarantined_until": ledger.QuarantinedUntil}, Err: result}
		}
	}()
	runtimeCommand, runtimeWait, authorizationWriter, err := startUnauthorizedRuntime(resolved, options.Environ)
	if err != nil {
		return err
	}
	runtimeOwned = true
	runtimeReaped = false
	cleanupRuntime := func() error {
		cleanupErr := terminateProcessGroup(runtimeCommand.Process.Pid, runtimeWait, time.Duration(resolved.Profile.Runtime.ShutdownTimeoutSeconds)*time.Second)
		if cleanupErr == nil {
			runtimeReaped = true
		}
		return cleanupErr
	}
	defer func() {
		if !runtimeReaped {
			_ = cleanupRuntime()
		}
	}()

	launcherObservation, err := waitForSharedProcessIdentity(runtimeCommand.Process.Pid, 2*time.Second)
	if err != nil {
		authorizationWriter.Close()
		return sharedRuntimeError("runtime_start_failed", err)
	}
	runtimeFile, err := fileIdentity(resolved.Profile.Runtime.Executable)
	if err != nil {
		authorizationWriter.Close()
		return piError("runtime_executable_invalid", err)
	}
	execPlanDigest := SharedRuntimeExecPlanDigest(resolved.Profile, resolved.Paths.RuntimeCWD)
	record.Runtime = &SharedRuntimeProcessRecord{
		PID: launcherObservation.PID, PGID: launcherObservation.PGID,
		StartTime: launcherObservation.StartTime, UID: launcherObservation.UID,
		PreExec:  ProcessExecShape{ExecPath: launcherObservation.ExecPath, Argv: append([]string(nil), launcherObservation.Argv...)},
		PostExec: ProcessExecShape{ExecPath: resolved.Profile.Runtime.Executable, Argv: append([]string{resolved.Profile.Runtime.Executable}, resolved.Profile.Runtime.Argv...)},
		CWD:      resolved.Paths.RuntimeCWD, Endpoint: resolved.Profile.BaseURL,
		ExecutableIdentity: runtimeFile, ExecPlanDigest: execPlanDigest, Stage: "authorizing",
	}
	if err := writeSharedJSONAtomic(paths.BrokerState, record); err != nil {
		authorizationWriter.Close()
		return err
	}
	frame := sharedRuntimeAuthorizationFrame{
		Schema: sharedRuntimeAuthSchema, ProtocolVersion: SharedRuntimeProtocolVersion,
		RuntimeKey: resolved.RuntimeKey, LauncherPID: runtimeCommand.Process.Pid,
		ExecPlanDigest: execPlanDigest,
	}
	if err := writeAuthorizationFrame(authorizationWriter, frame); err != nil {
		authorizationWriter.Close()
		return sharedRuntimeError("protocol_violation", err)
	}
	if err := authorizationWriter.Close(); err != nil {
		return sharedRuntimeError("protocol_violation", err)
	}
	wantModel := resolved.Profile.Model
	if resolved.Profile.Runtime.DFlash != nil {
		wantModel = resolved.Profile.Runtime.DFlash.TargetModel
	}
	if err := waitPiRuntimeReady(context.Background(), options.HTTPClient, resolved.Profile.BaseURL+resolved.Profile.Runtime.ReadinessPath, wantModel, runtimeCommand.Process, runtimeWait, time.Duration(resolved.Profile.Runtime.StartupTimeoutSeconds)*time.Second); err != nil {
		return err
	}
	readyAt := time.Now().UTC()
	runtimeReady = true
	sharedRuntimeRecordReadiness(&ledger, readyAt)
	if err := writeSharedRuntimeRestartLedger(paths.RestartLedger, ledger); err != nil {
		return err
	}
	record.State = "serving"
	record.Runtime.Stage = "running"
	record.ReadyAt = &readyAt
	if err := writeSharedJSONAtomic(paths.BrokerState, record); err != nil {
		return err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: paths.RendezvousSocket, Net: "unix"})
	if err != nil {
		return sharedRuntimeError("broker_rendezvous_bind_conflict", err)
	}
	if err := os.Chmod(paths.RendezvousSocket, 0o600); err != nil {
		listener.Close()
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}

	server := newSharedBrokerServer(resolved, &record, &ledger, listener)
	serveErr := server.serve(runtimeWait, options.Signals)
	listener.Close()
	if server.forcedStop() {
		server.closeConnections("lease_revoked", "operator_force_stop")
	} else {
		server.closeConnections("shutting_down", "broker_draining")
	}
	cleanupErr := cleanupRuntime()
	if cleanupErr != nil {
		return cleanupErr
	}
	if err := cleanupSharedRuntimeState(paths); err != nil {
		return err
	}
	recordPublished = false
	if serveErr != nil {
		return serveErr
	}
	return nil
}

func startUnauthorizedRuntime(resolved sharedResolvedProfile, environ []string) (*exec.Cmd, *piProcessWait, *os.File, error) {
	return startUnauthorizedRuntimeWithDependencies(resolved, environ, sharedSystemLogClock{}, func(command *exec.Cmd) error {
		return command.Start()
	})
}

func startUnauthorizedRuntimeWithDependencies(resolved sharedResolvedProfile, environ []string, clock sharedLogClock, startCommand func(*exec.Cmd) error) (*exec.Cmd, *piProcessWait, *os.File, error) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, sharedRuntimeError("runtime_start_failed", err)
	}
	executable, _, err := ownResolvedExecutableIdentity()
	if err != nil {
		readEnd.Close()
		writeEnd.Close()
		return nil, nil, nil, sharedRuntimeError("runtime_start_failed", err)
	}
	logFile, err := openSharedRotatingLog(resolved.Paths.RuntimeLog, resolved.Sharing.MaxSegmentBytes, resolved.Sharing.MaxSegments, clock)
	if err != nil {
		readEnd.Close()
		writeEnd.Close()
		return nil, nil, nil, err
	}
	args := []string{"runtime", "runtime-launch", "--runtime-key", resolved.RuntimeKey, "--profile-project", resolved.Project, "--profile", resolved.ProfileName}
	command := exec.Command(executable, args...)
	command.Dir = resolved.Paths.RuntimeCWD
	if environ == nil {
		environ = os.Environ()
	}
	command.Env = scrubSharedRuntimeEnvironment(environ)
	command.Stdin = nil
	command.Stdout = logFile
	command.Stderr = logFile
	command.ExtraFiles = []*os.File{readEnd}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := startCommand(command); err != nil {
		readEnd.Close()
		writeEnd.Close()
		logFile.Close()
		return nil, nil, nil, sharedRuntimeError("runtime_start_failed", err)
	}
	readEnd.Close()
	return command, waitForPiProcessAndClose(command, logFile), writeEnd, nil
}

func waitForPiProcessAndClose(command *exec.Cmd, closer io.Closer) *piProcessWait {
	wait := &piProcessWait{done: make(chan struct{})}
	go func() {
		wait.err = command.Wait()
		if closeErr := closer.Close(); wait.err == nil {
			wait.err = closeErr
		}
		close(wait.done)
	}()
	return wait
}

func waitForSharedProcessIdentity(pid int, timeout time.Duration) (sharedProcessObservation, error) {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		observation, err := inspectSharedProcess(pid)
		if err == nil {
			return observation, nil
		}
		last = err
		time.Sleep(5 * time.Millisecond)
	}
	return sharedProcessObservation{}, last
}

func openSharedLog(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_APPEND|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Nlink != 1 || stat.Mode&0o777 != 0o600 {
		if err == nil {
			err = errors.New("managed log must be a mode-0600 single-link regular file")
		}
		unix.Close(fd)
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	return os.NewFile(uintptr(fd), path), nil
}

type sharedRuntimeReclaimDependencies struct {
	inspectKernel    func(int) (sharedProcessObservation, error)
	inspectProcess   func(int) (sharedProcessObservation, error)
	kill             func(int, syscall.Signal) error
	waitGone         func(*SharedRuntimeProcessRecord, time.Duration) error
	waitEndpointFree func(string, time.Duration) error
}

var sharedRuntimeReclaimSystem = sharedRuntimeReclaimDependencies{
	inspectKernel:    inspectSharedProcessKernel,
	inspectProcess:   inspectSharedProcess,
	kill:             syscall.Kill,
	waitGone:         waitRecordedRuntimeGone,
	waitEndpointFree: waitSharedEndpointFree,
}

func reclaimSharedRuntime(runtimeRecord *SharedRuntimeProcessRecord, shutdownTimeoutSeconds int) error {
	if runtimeRecord == nil {
		return nil
	}
	system := sharedRuntimeReclaimSystem
	observation, err := system.inspectKernel(runtimeRecord.PID)
	if err != nil {
		if sharedProcessGone(err) {
			return nil
		}
		return sharedRuntimeError("shared_runtime_orphan_unidentifiable", err)
	}
	if !observation.live() {
		return nil
	}
	if observation.UID != uint32(os.Geteuid()) {
		return sharedRuntimeError("shared_runtime_orphan_unidentifiable", errors.New("recorded runtime uid differs"))
	}
	if observation.StartTime != runtimeRecord.StartTime {
		return nil
	}
	full, err := system.inspectProcess(runtimeRecord.PID)
	if err != nil {
		return sharedRuntimeError("shared_runtime_orphan_unidentifiable", err)
	}
	if full.PGID != runtimeRecord.PGID {
		return sharedRuntimeError("shared_runtime_orphan_unidentifiable", errors.New("recorded runtime process group differs"))
	}
	shapeMatches := func(shape ProcessExecShape) bool {
		return full.ExecPath == shape.ExecPath && equalStrings(full.Argv, shape.Argv)
	}
	if !shapeMatches(runtimeRecord.PreExec) && !shapeMatches(runtimeRecord.PostExec) {
		return sharedRuntimeError("shared_runtime_orphan_unidentifiable", errors.New("runtime matches neither recorded identity shape"))
	}
	if runtimeRecord.PGID <= 0 {
		return sharedRuntimeError("shared_runtime_orphan_unidentifiable", errors.New("runtime process group is invalid"))
	}
	_ = system.kill(-runtimeRecord.PGID, syscall.SIGTERM)
	if system.waitGone(runtimeRecord, time.Duration(shutdownTimeoutSeconds)*time.Second) == nil {
		return system.waitEndpointFree(runtimeRecord.Endpoint, time.Second)
	}
	_ = system.kill(-runtimeRecord.PGID, syscall.SIGKILL)
	if err := system.waitGone(runtimeRecord, time.Second); err != nil {
		return err
	}
	return system.waitEndpointFree(runtimeRecord.Endpoint, time.Second)
}

func waitRecordedRuntimeGone(runtimeRecord *SharedRuntimeProcessRecord, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		observation, err := inspectSharedProcessKernel(runtimeRecord.PID)
		if sharedProcessGone(err) || (err == nil && !observation.live()) || (err == nil && observation.StartTime != runtimeRecord.StartTime) {
			return nil
		}
		if err != nil {
			return sharedRuntimeError("shared_runtime_orphan_unidentifiable", err)
		}
		if time.Now().After(deadline) {
			return sharedRuntimeError("runtime_shutdown_timeout", errors.New("runtime process remained live"))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func waitSharedEndpointFree(endpoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if err := preflightPiListener(endpoint); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return sharedRuntimeError("runtime_shutdown_timeout", errors.New("runtime endpoint remained occupied"))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func removeStaleRendezvous(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	return nil
}

func cleanupSharedRuntimeState(paths SharedRuntimePaths) error {
	if err := removeSharedLeaseMirrors(paths.LeasesDir); err != nil {
		return err
	}
	for _, path := range []string{paths.BrokerState, paths.RendezvousSocket} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return sharedRuntimeError("shared_runtime_state_path_invalid", err)
		}
	}
	return nil
}

type sharedBrokerEvent struct {
	kind  string
	force bool
}

type sharedBrokerConnection struct {
	connection *net.UnixConn
	reader     *bufio.Reader
	writerMu   sync.Mutex
	leaseID    string
}

func (connection *sharedBrokerConnection) send(message sharedWireMessage) error {
	connection.writerMu.Lock()
	defer connection.writerMu.Unlock()
	return writeSharedWireMessage(connection.connection, message)
}

type sharedBrokerAdmissionDependencies struct {
	peerIdentity        func(*net.UnixConn) (uint32, int, error)
	inspectProcess      func(int) (sharedProcessObservation, error)
	processExecIdentity func(sharedProcessObservation) (FileIdentity, error)
}

var sharedBrokerAdmissionSystem = sharedBrokerAdmissionDependencies{
	peerIdentity:        sharedUnixPeerIdentity,
	inspectProcess:      inspectSharedProcess,
	processExecIdentity: processExecIdentity,
}

type sharedBrokerServer struct {
	resolved    sharedResolvedProfile
	record      *SharedBrokerRecord
	ledger      *SharedRuntimeRestartLedger
	listener    *net.UnixListener
	mu          sync.Mutex
	state       string
	connections map[*sharedBrokerConnection]bool
	leases      map[string]*SharedLeaseRecord
	events      chan sharedBrokerEvent
	hadLease    bool
	forced      bool
	readyAt     time.Time
	admission   sharedBrokerAdmissionDependencies
}

var sharedFirstLeaseGraceDuration = func(sharing PiRuntimeSharing) time.Duration {
	return time.Duration(sharing.BrokerStartTimeoutSeconds) * time.Second
}

func newSharedBrokerServer(resolved sharedResolvedProfile, record *SharedBrokerRecord, ledger *SharedRuntimeRestartLedger, listener *net.UnixListener) *sharedBrokerServer {
	return &sharedBrokerServer{
		resolved: resolved, record: record, ledger: ledger, listener: listener, state: "serving",
		connections: map[*sharedBrokerConnection]bool{}, leases: map[string]*SharedLeaseRecord{},
		events: make(chan sharedBrokerEvent, 32), readyAt: dereferenceReadyAt(record.ReadyAt),
		admission: sharedBrokerAdmissionSystem,
	}
}

func dereferenceReadyAt(value *time.Time) time.Time {
	if value == nil {
		return time.Now()
	}
	return *value
}

func (server *sharedBrokerServer) serve(runtimeWait *piProcessWait, signals <-chan os.Signal) error {
	accepts := make(chan *net.UnixConn)
	acceptErrors := make(chan error, 1)
	go func() {
		for {
			connection, err := server.listener.AcceptUnix()
			if err != nil {
				acceptErrors <- err
				return
			}
			accepts <- connection
		}
	}()
	var ownedSignals chan os.Signal
	if signals == nil {
		ownedSignals = make(chan os.Signal, 2)
		signal.Notify(ownedSignals, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(ownedSignals)
		signals = ownedSignals
	}
	firstLeaseDelay := time.Until(server.readyAt.Add(sharedFirstLeaseGraceDuration(server.resolved.Sharing)))
	if firstLeaseDelay < 0 {
		firstLeaseDelay = 0
	}
	firstLeaseGrace := time.NewTimer(firstLeaseDelay)
	defer firstLeaseGrace.Stop()
	stableDelay := time.Until(server.readyAt.Add(time.Duration(server.resolved.Sharing.StableRunSeconds) * time.Second))
	if stableDelay < 0 {
		stableDelay = 0
	}
	stableRun := time.NewTimer(stableDelay)
	defer stableRun.Stop()
	stableRunChannel := stableRun.C
	var linger *time.Timer
	var lingerChannel <-chan time.Time
	for {
		select {
		case connection := <-accepts:
			wrapped := &sharedBrokerConnection{connection: connection, reader: bufio.NewReaderSize(connection, sharedRuntimeMaxFrameBytes+1)}
			server.mu.Lock()
			server.connections[wrapped] = true
			server.mu.Unlock()
			go server.handleConnection(wrapped)
		case err := <-acceptErrors:
			if server.currentState() == "draining" || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return sharedRuntimeError("broker_unreachable", err)
		case <-runtimeWait.done:
			if time.Since(server.readyAt) >= time.Duration(server.resolved.Sharing.StableRunSeconds)*time.Second {
				sharedRuntimeResetStableRun(server.ledger)
			}
			decision := sharedRuntimeRecordFailure(server.ledger, server.resolved.Sharing, time.Now().UTC())
			if err := writeSharedRuntimeRestartLedger(server.resolved.Paths.RestartLedger, *server.ledger); err != nil {
				return err
			}
			runtimeErr := fmt.Errorf("runtime exited while broker served: %v", runtimeWait.err)
			if decision.Quarantined {
				return &SharedRuntimeError{Code: "shared_runtime_quarantined", Details: map[string]any{"restart_count": server.ledger.RestartCount, "quarantined_until": server.ledger.QuarantinedUntil}, Err: runtimeErr}
			}
			return sharedRuntimeError("runtime_exited_early", runtimeErr)
		case <-stableRunChannel:
			sharedRuntimeResetStableRun(server.ledger)
			if err := writeSharedRuntimeRestartLedger(server.resolved.Paths.RestartLedger, *server.ledger); err != nil {
				return err
			}
			stableRunChannel = nil
		case <-signals:
			server.setDraining()
			return nil
		case event := <-server.events:
			switch event.kind {
			case "stop":
				server.mu.Lock()
				server.forced = event.force
				server.mu.Unlock()
				server.setDraining()
				return nil
			case "lease-change":
				leaseCount, hadLease := server.leaseFacts()
				if leaseCount > 0 {
					if linger != nil {
						if !linger.Stop() {
							select {
							case <-linger.C:
							default:
							}
						}
					}
					lingerChannel = nil
					server.setState("serving")
				} else if hadLease {
					server.setState("lingering")
					if server.resolved.Sharing.LingerSeconds == 0 {
						server.setDraining()
						return nil
					}
					linger = time.NewTimer(time.Duration(server.resolved.Sharing.LingerSeconds) * time.Second)
					lingerChannel = linger.C
				}
			}
		case <-firstLeaseGrace.C:
			_, hadLease := server.leaseFacts()
			if !hadLease {
				server.setDraining()
				return nil
			}
		case <-lingerChannel:
			server.setDraining()
			return nil
		}
	}
}

func (server *sharedBrokerServer) currentState() string {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.state
}

func (server *sharedBrokerServer) setState(state string) {
	server.mu.Lock()
	server.state = state
	server.record.State = state
	record := *server.record
	server.mu.Unlock()
	_ = writeSharedJSONAtomic(server.resolved.Paths.BrokerState, record)
}

func (server *sharedBrokerServer) setDraining() {
	server.setState("draining")
	_ = server.listener.Close()
}

func (server *sharedBrokerServer) leaseFacts() (int, bool) {
	server.mu.Lock()
	defer server.mu.Unlock()
	return len(server.leases), server.hadLease
}

func (server *sharedBrokerServer) forcedStop() bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.forced
}

func (server *sharedBrokerServer) closeConnections(messageType, reason string) {
	server.mu.Lock()
	connections := make([]*sharedBrokerConnection, 0, len(server.connections))
	for connection := range server.connections {
		connections = append(connections, connection)
	}
	server.mu.Unlock()
	for _, connection := range connections {
		_ = connection.send(sharedWireMessage{Type: messageType, LeaseID: connection.leaseID, Reason: reason})
		_ = connection.connection.Close()
	}
}

func (server *sharedBrokerServer) handleConnection(connection *sharedBrokerConnection) {
	defer func() {
		server.releaseConnectionLease(connection)
		server.mu.Lock()
		delete(server.connections, connection)
		server.mu.Unlock()
		connection.connection.Close()
	}()
	if err := connection.connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return
	}
	hello, err := readSharedWireMessage(connection.reader)
	if err != nil || hello.Type != "hello" {
		_ = connection.send(sharedWireMessage{Type: "refused", Code: "protocol_violation"})
		return
	}
	clientObservation, refusal := server.attestClient(connection.connection, hello)
	if refusal != nil {
		_ = sendSharedRefusal(connection, refusal)
		return
	}
	effective := server.resolved.Sharing
	broker := server.record.Broker
	runtimeRecord := server.record.Runtime
	if err := connection.send(sharedWireMessage{
		Type: "hello_ok", ProtocolVersion: SharedRuntimeProtocolVersion,
		RuntimeKey: server.resolved.RuntimeKey, ProfileDigest: server.resolved.ProfileDigest,
		EffectiveSharing: &effective, Broker: &broker, Runtime: runtimeRecord,
	}); err != nil {
		return
	}
	for {
		_ = connection.connection.SetReadDeadline(time.Now().Add(time.Duration(server.resolved.Sharing.LeaseStaleSeconds) * 2 * time.Second))
		message, err := readSharedWireMessage(connection.reader)
		if err != nil {
			return
		}
		switch message.Type {
		case "acquire":
			if connection.leaseID != "" {
				_ = connection.send(sharedWireMessage{Type: "refused", Code: "protocol_violation", Reason: "duplicate_acquire"})
				return
			}
			lease, refusal := server.acquireLease(message, hello, clientObservation)
			if refusal != nil {
				_ = sendSharedRefusal(connection, refusal)
				return
			}
			connection.leaseID = lease.LeaseID
			server.mu.Lock()
			server.leases[lease.LeaseID] = lease
			server.hadLease = true
			leaseCount := len(server.leases)
			server.mu.Unlock()
			if err := writeSharedJSONAtomic(filepath.Join(server.resolved.Paths.LeasesDir, lease.LeaseID+".json"), lease); err != nil {
				server.releaseConnectionLease(connection)
				_ = sendSharedRefusal(connection, err)
				return
			}
			server.notify("lease-change", false)
			if err := connection.send(sharedWireMessage{Type: "lease", Lease: lease, LeaseID: lease.LeaseID, LeaseCount: leaseCount, Runtime: server.record.Runtime}); err != nil {
				return
			}
		case "heartbeat":
			if message.LeaseID == "" || message.LeaseID != connection.leaseID {
				_ = connection.send(sharedWireMessage{Type: "refused", Code: "protocol_violation", Reason: "lease_identity_mismatch"})
				return
			}
			server.mu.Lock()
			lease := server.leases[connection.leaseID]
			if lease != nil {
				lease.LastHeartbeatAt = time.Now().UTC()
			}
			leaseCount := len(server.leases)
			server.mu.Unlock()
			if lease != nil {
				_ = writeSharedJSONAtomic(filepath.Join(server.resolved.Paths.LeasesDir, lease.LeaseID+".json"), lease)
			}
			if err := connection.send(sharedWireMessage{Type: "heartbeat_ok", LeaseID: connection.leaseID, LeaseCount: leaseCount}); err != nil {
				return
			}
		case "release":
			if message.LeaseID != connection.leaseID {
				_ = connection.send(sharedWireMessage{Type: "refused", Code: "protocol_violation", Reason: "lease_identity_mismatch"})
				return
			}
			server.releaseConnectionLease(connection)
			_ = connection.send(sharedWireMessage{Type: "released", LeaseID: message.LeaseID})
			return
		case "status":
			state, leases := server.statusSnapshot()
			if err := connection.send(sharedWireMessage{Type: "status", State: state, Stage: server.record.Stage, Leases: leases, LeaseCount: len(leases), Runtime: server.record.Runtime, Broker: &server.record.Broker, EffectiveSharing: server.record.Sharing}); err != nil {
				return
			}
		case "stop":
			_, leases := server.statusSnapshot()
			leaseCount := len(leases)
			if leaseCount > 0 && !message.Force {
				_ = connection.send(sharedWireMessage{Type: "refused", Code: "shared_runtime_leases_active", LeaseCount: leaseCount, Leases: leases})
				return
			}
			_ = connection.send(sharedWireMessage{Type: "stopping", LeaseCount: leaseCount})
			server.notify("stop", message.Force)
			return
		default:
			_ = connection.send(sharedWireMessage{Type: "refused", Code: "protocol_violation", Reason: "unknown_message"})
			return
		}
	}
}

func (server *sharedBrokerServer) attestClient(connection *net.UnixConn, hello sharedWireMessage) (sharedProcessObservation, error) {
	system := server.admission
	uid, pid, err := system.peerIdentity(connection)
	if err != nil || uid != uint32(os.Geteuid()) {
		if err == nil {
			err = errors.New("peer uid differs")
		}
		return sharedProcessObservation{}, sharedRuntimeError("broker_peer_untrusted", err)
	}
	if hello.ClientPID != pid {
		return sharedProcessObservation{}, sharedRuntimeError("broker_identity_mismatch", errors.New("hello client pid differs from LOCAL_PEERPID"))
	}
	observation, err := system.inspectProcess(pid)
	if err != nil || !observation.live() {
		if err == nil {
			err = errors.New("client is a zombie")
		}
		return sharedProcessObservation{}, sharedRuntimeError("broker_identity_unavailable", err)
	}
	clientExec, err := system.processExecIdentity(observation)
	if err != nil {
		return sharedProcessObservation{}, sharedRuntimeError("broker_identity_unavailable", err)
	}
	if clientExec.Dev != server.record.Broker.ExecutableIdentity.Dev || clientExec.Ino != server.record.Broker.ExecutableIdentity.Ino {
		return sharedProcessObservation{}, sharedRuntimeError("broker_executable_identity_mismatch", errors.New("client executable inode differs"))
	}
	if hello.ProtocolVersion != SharedRuntimeProtocolVersion {
		return sharedProcessObservation{}, sharedRuntimeError("broker_protocol_version_mismatch", errors.New("client protocol version differs"))
	}
	if hello.RuntimeKey != server.resolved.RuntimeKey {
		return sharedProcessObservation{}, sharedRuntimeError("shared_runtime_identity_mismatch", errors.New("client runtime key differs"))
	}
	if hello.ProfileDigest != server.resolved.ProfileDigest {
		return sharedProcessObservation{}, sharedRuntimeError("shared_runtime_profile_mismatch", errors.New("client profile digest differs"))
	}
	return observation, nil
}

func (server *sharedBrokerServer) acquireLease(message, hello sharedWireMessage, client sharedProcessObservation) (*SharedLeaseRecord, error) {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.state == "draining" {
		return nil, sharedRuntimeError("shared_runtime_shutting_down", errors.New("broker is draining"))
	}
	if len(server.leases) >= server.resolved.Sharing.MaxLeases {
		configured := server.resolved.Sharing.MaxLeases
		if hello.ConfiguredSharing != nil {
			configured = hello.ConfiguredSharing.MaxLeases
		}
		return nil, &SharedRuntimeError{Code: "shared_runtime_lease_limit", Details: map[string]any{
			"effective_max_leases":  server.resolved.Sharing.MaxLeases,
			"configured_max_leases": configured,
			"broker_pid":            server.record.Broker.PID,
			"broker_start_time":     server.record.Broker.StartTime,
		}, Err: errors.New("effective shared runtime lease limit reached")}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	now := time.Now().UTC()
	return &SharedLeaseRecord{
		LeaseID: hex.EncodeToString(nonce[:]), RunID: hello.RunID, ClientKey: message.ClientKey,
		ClientPID: client.PID, ClientStartTime: client.StartTime,
		RuntimeKey: server.resolved.RuntimeKey, ProfileDigest: server.resolved.ProfileDigest,
		Endpoint: server.resolved.Profile.BaseURL, RuntimePID: server.record.Runtime.PID,
		GrantedAt: now, LastHeartbeatAt: now,
	}, nil
}

func (server *sharedBrokerServer) releaseConnectionLease(connection *sharedBrokerConnection) {
	if connection.leaseID == "" {
		return
	}
	server.mu.Lock()
	_, present := server.leases[connection.leaseID]
	delete(server.leases, connection.leaseID)
	leaseID := connection.leaseID
	connection.leaseID = ""
	server.mu.Unlock()
	if present {
		_ = os.Remove(filepath.Join(server.resolved.Paths.LeasesDir, leaseID+".json"))
		server.notify("lease-change", false)
	}
}

func (server *sharedBrokerServer) notify(kind string, force bool) {
	select {
	case server.events <- sharedBrokerEvent{kind: kind, force: force}:
	default:
		go func() { server.events <- sharedBrokerEvent{kind: kind, force: force} }()
	}
}

func (server *sharedBrokerServer) statusSnapshot() (string, []SharedLeaseStatus) {
	server.mu.Lock()
	defer server.mu.Unlock()
	now := time.Now()
	leases := make([]SharedLeaseStatus, 0, len(server.leases))
	for _, lease := range server.leases {
		state := "held"
		heartbeatAge := now.Sub(lease.LastHeartbeatAt)
		if heartbeatAge > time.Duration(server.resolved.Sharing.LeaseStaleSeconds)*time.Second {
			state = "held(stale)"
		}
		leases = append(leases, SharedLeaseStatus{SharedLeaseRecord: *lease, State: state, Age: now.Sub(lease.GrantedAt), HeartbeatAge: heartbeatAge})
	}
	return server.state, leases
}

func sendSharedRefusal(connection *sharedBrokerConnection, err error) error {
	message := sharedWireMessage{Type: "refused", Code: "protocol_violation"}
	var shared *SharedRuntimeError
	if errors.As(err, &shared) {
		message.Code = shared.Code
		message.Reason = shared.Reason
		if shared.Details != nil {
			if value, ok := shared.Details["effective_max_leases"].(int); ok {
				message.EffectiveMaxLeases = value
			}
			if value, ok := shared.Details["configured_max_leases"].(int); ok {
				message.ConfiguredMaxLeases = value
			}
			if value, ok := shared.Details["broker_pid"].(int); ok {
				message.BrokerPID = value
			}
			if value, ok := shared.Details["broker_start_time"].(ProcessStartTime); ok {
				message.BrokerStartTime = &value
			}
		}
	}
	return connection.send(message)
}

func readSharedWireMessage(reader io.Reader) (sharedWireMessage, error) {
	buffered, ok := reader.(*bufio.Reader)
	if !ok {
		buffered = bufio.NewReaderSize(reader, sharedRuntimeMaxFrameBytes+1)
	}
	line, err := buffered.ReadBytes('\n')
	if err != nil {
		return sharedWireMessage{}, err
	}
	if len(line) > sharedRuntimeMaxFrameBytes {
		return sharedWireMessage{}, sharedRuntimeError("protocol_violation", errors.New("frame exceeds 65536 bytes"))
	}
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	var message sharedWireMessage
	if err := decoder.Decode(&message); err != nil {
		return sharedWireMessage{}, sharedRuntimeError("protocol_violation", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return sharedWireMessage{}, sharedRuntimeError("protocol_violation", err)
	}
	return message, nil
}

func writeSharedWireMessage(writer io.Writer, message sharedWireMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(data)+1 > sharedRuntimeMaxFrameBytes {
		return sharedRuntimeError("protocol_violation", errors.New("frame exceeds 65536 bytes"))
	}
	data = append(data, '\n')
	_, err = writer.Write(data)
	return err
}
