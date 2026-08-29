//go:build darwin

package infra

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type SharedRuntimeOperatorOptions struct {
	ProjectDir string
	Profile    string
	HomeDir    string
	CacheRoot  string
	HTTPClient *http.Client
}

type SharedRuntimeGateOutcome struct {
	Gate    string `json:"gate"`
	Outcome string `json:"outcome"`
	Source  string `json:"source"`
}

type SharedRuntimeCandidate struct {
	PID       int              `json:"pid"`
	PStat     string           `json:"p_stat"`
	StartTime ProcessStartTime `json:"start_time"`
	Argv      []string         `json:"argv"`
}

type SharedRuntimeBrokerStatus struct {
	State            string                   `json:"state"`
	Stage            string                   `json:"stage,omitempty"`
	Source           string                   `json:"source"`
	PID              int                      `json:"pid,omitempty"`
	StartTime        *ProcessStartTime        `json:"start_time,omitempty"`
	UptimeSeconds    float64                  `json:"uptime_seconds,omitempty"`
	CandidateHolders []SharedRuntimeCandidate `json:"candidate_holders,omitempty"`
}

type SharedRuntimeProcessStatus struct {
	Source    string           `json:"source"`
	PID       int              `json:"pid"`
	PGID      int              `json:"pgid"`
	StartTime ProcessStartTime `json:"start_time"`
	ExecPath  string           `json:"exec_path"`
	Argv      []string         `json:"argv"`
	CWD       string           `json:"cwd"`
	Endpoint  string           `json:"endpoint"`
}

type SharedRuntimeSharingStatus struct {
	Mode       string            `json:"mode"`
	Configured PiRuntimeSharing  `json:"configured"`
	Effective  *PiRuntimeSharing `json:"effective,omitempty"`
	FixedByPID int               `json:"fixed_by_broker_pid,omitempty"`
	FixedAt    *ProcessStartTime `json:"fixed_by_broker_start_time,omitempty"`
}

type SharedRuntimeStatus struct {
	RuntimeKey    string                      `json:"runtime_key"`
	ProfileDigest string                      `json:"profile_digest"`
	Endpoint      string                      `json:"endpoint"`
	Sharing       SharedRuntimeSharingStatus  `json:"sharing"`
	Broker        SharedRuntimeBrokerStatus   `json:"broker"`
	Runtime       *SharedRuntimeProcessStatus `json:"runtime,omitempty"`
	Leases        []SharedLeaseStatus         `json:"leases"`
	Attestation   []SharedRuntimeGateOutcome  `json:"attestation"`
	Paths         SharedRuntimePaths          `json:"paths"`
}

type SharedRuntimeStopResult struct {
	State             string `json:"state"`
	BrokerPID         int    `json:"broker_pid,omitempty"`
	RuntimePID        int    `json:"runtime_pid,omitempty"`
	BrokerTerminated  bool   `json:"broker_terminated"`
	RuntimeTerminated bool   `json:"runtime_terminated"`
}

func SharedRuntimeStatusReport(options SharedRuntimeOperatorOptions) (SharedRuntimeStatus, error) {
	resolved, state, err := resolveSharedRuntimeOperator(options)
	if err != nil {
		return SharedRuntimeStatus{}, err
	}
	report := newSharedRuntimeStatus(resolved)
	attested, err := connectAndAttestSharedRuntime(resolved, state, "", options.HTTPClient, time.Second)
	if err == nil {
		defer attested.close()
		if err := writeSharedWireMessage(attested.connection, sharedWireMessage{Type: "status"}); err != nil {
			return SharedRuntimeStatus{}, sharedRuntimeError("broker_unreachable", err)
		}
		message, err := readSharedWireMessage(attested.reader)
		if err != nil {
			return SharedRuntimeStatus{}, sharedRuntimeError("broker_unreachable", err)
		}
		if message.Type != "status" || message.Broker == nil || message.Runtime == nil || message.EffectiveSharing == nil {
			return SharedRuntimeStatus{}, sharedRuntimeError("protocol_violation", errors.New("broker status response is incomplete"))
		}
		report.Broker = sharedBrokerStatus(message.State, message.Stage, "attested", *message.Broker)
		report.Runtime = sharedRuntimeProcessStatus(message.Runtime, "attested")
		report.Leases = append([]SharedLeaseStatus(nil), message.Leases...)
		report.Attestation = append([]SharedRuntimeGateOutcome(nil), attested.gates...)
		report.Sharing.Effective = message.EffectiveSharing
		report.Sharing.FixedByPID = message.Broker.PID
		report.Sharing.FixedAt = &message.Broker.StartTime
		return report, nil
	}
	if !isSharedBrokerUnreachable(err) {
		return SharedRuntimeStatus{}, err
	}
	return sharedRuntimeRecordStatus(resolved, report)
}

func StopSharedRuntime(options SharedRuntimeOperatorOptions, force bool, timeout time.Duration) (SharedRuntimeStopResult, error) {
	if timeout <= 0 {
		return SharedRuntimeStopResult{}, sharedRuntimeError("invalid_runtime_stop_timeout", errors.New("timeout must be positive"))
	}
	resolved, state, err := resolveSharedRuntimeOperator(options)
	if err != nil {
		return SharedRuntimeStopResult{}, err
	}
	attested, connectErr := connectAndAttestSharedRuntime(resolved, state, "", options.HTTPClient, minSharedDuration(timeout, 2*time.Second))
	if connectErr == nil {
		defer attested.close()
		if err := writeSharedWireMessage(attested.connection, sharedWireMessage{Type: "stop", Force: force, TimeoutSeconds: int(timeout.Seconds())}); err != nil {
			return SharedRuntimeStopResult{}, sharedRuntimeError("broker_unreachable", err)
		}
		response, err := readSharedWireMessage(attested.reader)
		if err != nil {
			return SharedRuntimeStopResult{}, sharedRuntimeError("broker_unreachable", err)
		}
		if response.Type == "refused" {
			details := map[string]any{"lease_count": response.LeaseCount, "leases": response.Leases}
			return SharedRuntimeStopResult{}, &SharedRuntimeError{Code: response.Code, Reason: response.Reason, Details: details, Err: errors.New("broker refused stop")}
		}
		if response.Type != "stopping" {
			return SharedRuntimeStopResult{}, sharedRuntimeError("protocol_violation", fmt.Errorf("unexpected stop response %q", response.Type))
		}
		deadline := time.Now().Add(timeout)
		if err := waitRecordedBrokerGone(attested.broker, deadline); err != nil {
			return SharedRuntimeStopResult{}, err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return SharedRuntimeStopResult{}, sharedRuntimeError("runtime_shutdown_timeout", errors.New("operator stop deadline elapsed before runtime verification"))
		}
		if err := waitRecordedRuntimeGone(&attested.runtime, remaining); err != nil {
			return SharedRuntimeStopResult{}, err
		}
		remaining = time.Until(deadline)
		if remaining <= 0 {
			return SharedRuntimeStopResult{}, sharedRuntimeError("runtime_shutdown_timeout", errors.New("operator stop deadline elapsed before endpoint verification"))
		}
		if err := waitSharedEndpointFree(resolved.Profile.BaseURL, remaining); err != nil {
			return SharedRuntimeStopResult{}, err
		}
		return SharedRuntimeStopResult{State: "absent", BrokerPID: attested.broker.PID, RuntimePID: attested.runtime.PID, BrokerTerminated: true, RuntimeTerminated: true}, nil
	}
	if !isSharedBrokerUnreachable(connectErr) {
		return SharedRuntimeStopResult{}, connectErr
	}
	if !force {
		return SharedRuntimeStopResult{}, sharedRuntimeError("shared_runtime_broker_unreachable_use_force", connectErr)
	}
	return forceStopUnreachableSharedRuntime(resolved, timeout)
}

func resolveSharedRuntimeOperator(options SharedRuntimeOperatorOptions) (sharedResolvedProfile, PiStatePaths, error) {
	resolved, err := resolveSharedProfile(options.ProjectDir, options.HomeDir, options.CacheRoot, options.Profile)
	if err != nil {
		return sharedResolvedProfile{}, PiStatePaths{}, err
	}
	state, err := ResolvePiStatePaths(options.CacheRoot, resolved.Project, resolved.ProfileName)
	if err != nil {
		return sharedResolvedProfile{}, PiStatePaths{}, err
	}
	return resolved, state, nil
}

func newSharedRuntimeStatus(resolved sharedResolvedProfile) SharedRuntimeStatus {
	return SharedRuntimeStatus{
		RuntimeKey: resolved.RuntimeKey, ProfileDigest: resolved.ProfileDigest,
		Endpoint: resolved.Profile.BaseURL, Paths: resolved.Paths,
		Sharing: SharedRuntimeSharingStatus{Mode: resolved.Sharing.Mode, Configured: resolved.Sharing},
		Broker:  SharedRuntimeBrokerStatus{State: "absent", Source: "determined"},
		Leases:  []SharedLeaseStatus{}, Attestation: []SharedRuntimeGateOutcome{},
	}
}

func sharedRuntimeRecordStatus(resolved sharedResolvedProfile, report SharedRuntimeStatus) (SharedRuntimeStatus, error) {
	record, present, err := readSharedBrokerRecord(resolved.Paths.BrokerState)
	if err != nil {
		return SharedRuntimeStatus{}, err
	}
	if !present {
		held, lock, err := probeSharedBrokerLock(resolved.Paths)
		if err != nil {
			return SharedRuntimeStatus{}, err
		}
		if lock != nil {
			lock.Close()
		}
		if held {
			candidates, err := sharedRuntimeBrokerCandidates(resolved)
			if err != nil {
				return SharedRuntimeStatus{}, err
			}
			report.Broker = SharedRuntimeBrokerStatus{State: "starting-unverified", Source: "kernel-candidate-reporting-only", CandidateHolders: candidates}
		}
		return report, nil
	}
	state := record.State
	if state == "" {
		state = "starting"
	}
	if !sharedRecordedBrokerIsLive(record.Broker) {
		state = "unverified-stale"
	}
	report.Broker = sharedBrokerStatus(state, record.Stage, "record-derived-unverified", record.Broker)
	if record.Runtime != nil {
		report.Runtime = sharedRuntimeProcessStatus(record.Runtime, "record-derived-unverified")
	}
	if record.Sharing != nil {
		effective := *record.Sharing
		report.Sharing.Effective = &effective
		report.Sharing.FixedByPID = record.Broker.PID
		report.Sharing.FixedAt = &record.Broker.StartTime
	}
	leasing, err := readSharedLeaseMirrors(resolved.Paths.LeasesDir)
	if err != nil {
		return SharedRuntimeStatus{}, err
	}
	report.Leases = leasing
	return report, nil
}

func sharedBrokerStatus(state, stage, source string, broker SharedBrokerIdentity) SharedRuntimeBrokerStatus {
	start := broker.StartTime
	uptime := time.Since(time.Unix(start.Seconds, int64(start.Microseconds)*1000)).Seconds()
	if uptime < 0 {
		uptime = 0
	}
	return SharedRuntimeBrokerStatus{State: state, Stage: stage, Source: source, PID: broker.PID, StartTime: &start, UptimeSeconds: uptime}
}

func sharedRuntimeProcessStatus(runtime *SharedRuntimeProcessRecord, source string) *SharedRuntimeProcessStatus {
	if runtime == nil {
		return nil
	}
	return &SharedRuntimeProcessStatus{
		Source: source, PID: runtime.PID, PGID: runtime.PGID, StartTime: runtime.StartTime,
		ExecPath: runtime.PostExec.ExecPath, Argv: redactRuntimeDiagnosticArgv(runtime.PostExec.Argv),
		CWD: runtime.CWD, Endpoint: runtime.Endpoint,
	}
}

func redactRuntimeDiagnosticArgv(argv []string) []string {
	result := append([]string(nil), argv...)
	for index := range result {
		if index > 0 && (result[index-1] == "--api-key" || result[index-1] == "--token") {
			result[index] = "[redacted]"
			continue
		}
		for _, prefix := range []string{"--api-key=", "--token="} {
			if strings.HasPrefix(result[index], prefix) {
				result[index] = prefix + "[redacted]"
			}
		}
	}
	return result
}

func readSharedLeaseMirrors(directory string) ([]SharedLeaseStatus, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return []SharedLeaseStatus{}, nil
	}
	if err != nil {
		return nil, sharedRuntimeError("shared_runtime_state_unreadable", err)
	}
	now := time.Now()
	result := make([]SharedLeaseStatus, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, sharedRuntimeError("shared_runtime_state_unreadable", err)
		}
		var lease SharedLeaseRecord
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&lease); err != nil || requireJSONEOF(decoder) != nil {
			if err == nil {
				err = errors.New("content follows lease mirror")
			}
			return nil, sharedRuntimeError("shared_runtime_state_unreadable", err)
		}
		result = append(result, SharedLeaseStatus{SharedLeaseRecord: lease, State: "unverified", Age: now.Sub(lease.GrantedAt), HeartbeatAge: now.Sub(lease.LastHeartbeatAt)})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].LeaseID < result[right].LeaseID })
	return result, nil
}

func isSharedBrokerUnreachable(err error) bool {
	if isSharedConnectAbsence(err) {
		return true
	}
	var shared *SharedRuntimeError
	return errors.As(err, &shared) && (shared.Code == "broker_unreachable" || shared.Code == "shared_runtime_unavailable")
}

// probeSharedBrokerLock reports a held existing lock. If it can take the lock,
// it returns the live descriptor so callers can keep the absence proof stable.
func probeSharedBrokerLock(paths SharedRuntimePaths) (held bool, acquired *sharedBrokerLock, result error) {
	if _, err := os.Stat(paths.Root); errors.Is(err, os.ErrNotExist) {
		return false, nil, nil
	} else if err != nil {
		return false, nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	lock, err := openSharedBrokerLock(paths.BrokerLock)
	if err == nil {
		return false, lock, nil
	}
	var shared *SharedRuntimeError
	if errors.As(err, &shared) && shared.Code == "broker_election_lost" {
		return true, nil, nil
	}
	return false, nil, err
}

func sharedRuntimeBrokerCandidates(resolved sharedResolvedProfile) ([]SharedRuntimeCandidate, error) {
	processes, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, sharedRuntimeError("broker_identity_unavailable", err)
	}
	_, ownIdentity, err := ownResolvedExecutableIdentity()
	if err != nil {
		return nil, sharedRuntimeError("broker_identity_unavailable", err)
	}
	result := []SharedRuntimeCandidate{}
	for _, process := range processes {
		pid := int(process.Proc.P_pid)
		if pid <= 0 || process.Eproc.Ucred.Uid != uint32(os.Geteuid()) || process.Proc.P_stat == darwinProcessStateZombie {
			continue
		}
		observation, err := inspectSharedProcess(pid)
		if err != nil || !observation.live() || !isBrokerArgvForRuntime(observation.Argv, resolved.RuntimeKey) {
			continue
		}
		identity, err := processExecIdentity(observation)
		if err != nil || identity.Dev != ownIdentity.Dev || identity.Ino != ownIdentity.Ino {
			continue
		}
		result = append(result, SharedRuntimeCandidate{PID: pid, PStat: darwinProcessStateName(observation.PStat), StartTime: observation.StartTime, Argv: append([]string(nil), observation.Argv...)})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].PID < result[right].PID })
	return result, nil
}

func isBrokerArgvForRuntime(argv []string, runtimeKey string) bool {
	if len(argv) < 4 || argv[1] != "runtime" || argv[2] != "broker" {
		return false
	}
	for index := 3; index+1 < len(argv); index++ {
		if argv[index] == "--runtime-key" {
			return argv[index+1] == runtimeKey
		}
	}
	return false
}

func darwinProcessStateName(state int8) string {
	switch state {
	case 1:
		return "SIDL"
	case 2:
		return "SRUN"
	case 3:
		return "SSLEEP"
	case 4:
		return "SSTOP"
	case 5:
		return "SZOMB"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", state)
	}
}

type sharedRuntimeOperatorDependencies struct {
	inspectProcess        func(int) (sharedProcessObservation, error)
	ownExecutableIdentity func() (string, FileIdentity, error)
	processExecIdentity   func(sharedProcessObservation) (FileIdentity, error)
	kill                  func(int, syscall.Signal) error
}

func sharedRecordedBrokerIsLive(record SharedBrokerIdentity) bool {
	observation, err := inspectSharedProcessKernel(record.PID)
	return err == nil && observation.live() && observation.UID == uint32(os.Geteuid()) && observation.StartTime == record.StartTime
}

func forceStopUnreachableSharedRuntime(resolved sharedResolvedProfile, timeout time.Duration) (SharedRuntimeStopResult, error) {
	deadline := time.Now().Add(timeout)
	for {
		record, present, err := readSharedBrokerRecord(resolved.Paths.BrokerState)
		if err != nil {
			return SharedRuntimeStopResult{}, err
		}
		if present {
			return stopRecordedSharedRuntime(resolved, record, deadline)
		}
		held, lock, err := probeSharedBrokerLock(resolved.Paths)
		if err != nil {
			return SharedRuntimeStopResult{}, err
		}
		if lock != nil {
			defer lock.Close()
			if err := cleanupSharedRuntimeState(resolved.Paths); err != nil {
				return SharedRuntimeStopResult{}, err
			}
			return SharedRuntimeStopResult{State: "absent"}, nil
		}
		if !held {
			return SharedRuntimeStopResult{State: "absent"}, nil
		}
		if !time.Now().Before(deadline) {
			candidates, candidatesErr := sharedRuntimeBrokerCandidates(resolved)
			if candidatesErr != nil {
				return SharedRuntimeStopResult{}, candidatesErr
			}
			return SharedRuntimeStopResult{}, &SharedRuntimeError{Code: "shared_runtime_owner_unidentifiable", Details: map[string]any{"candidate_holders": candidates, "operator_next_step": "inspect broker.lock with lsof; signal only after human identification"}, Err: errors.New("broker.lock remains held without an owner record")}
		}
		time.Sleep(minSharedDuration(100*time.Millisecond, time.Until(deadline)))
	}
}

func stopRecordedSharedRuntime(resolved sharedResolvedProfile, record *SharedBrokerRecord, deadline time.Time) (SharedRuntimeStopResult, error) {
	return stopRecordedSharedRuntimeWithDependencies(resolved, record, deadline, sharedRuntimeOperatorDependencies{
		inspectProcess:        inspectSharedProcess,
		ownExecutableIdentity: ownResolvedExecutableIdentity,
		processExecIdentity:   processExecIdentity,
		kill:                  syscall.Kill,
	})
}

func stopRecordedSharedRuntimeWithDependencies(resolved sharedResolvedProfile, record *SharedBrokerRecord, deadline time.Time, system sharedRuntimeOperatorDependencies) (SharedRuntimeStopResult, error) {
	brokerObservation, err := system.inspectProcess(record.Broker.PID)
	if err != nil || !brokerObservation.live() || brokerObservation.UID != uint32(os.Geteuid()) || brokerObservation.StartTime != record.Broker.StartTime || !equalStrings(brokerObservation.Argv, record.Broker.Argv) {
		if err == nil {
			err = errors.New("recorded broker kernel identity differs")
		}
		return SharedRuntimeStopResult{}, sharedRuntimeError("broker_stop_identity_mismatch", err)
	}
	_, ownIdentity, err := system.ownExecutableIdentity()
	if err != nil {
		return SharedRuntimeStopResult{}, sharedRuntimeError("broker_stop_identity_mismatch", err)
	}
	brokerIdentity, err := system.processExecIdentity(brokerObservation)
	if err != nil || brokerIdentity.Dev != ownIdentity.Dev || brokerIdentity.Ino != ownIdentity.Ino {
		if err == nil {
			err = errors.New("recorded broker executable differs from caller")
		}
		return SharedRuntimeStopResult{}, sharedRuntimeError("broker_stop_identity_mismatch", err)
	}
	_ = system.kill(record.Broker.PID, syscall.SIGTERM)
	if err := waitRecordedBrokerGone(record.Broker, deadline); err != nil {
		_ = system.kill(record.Broker.PID, syscall.SIGKILL)
		killDeadline := time.Now().Add(time.Second)
		if err := waitRecordedBrokerGone(record.Broker, killDeadline); err != nil {
			return SharedRuntimeStopResult{}, err
		}
	}
	var lock *sharedBrokerLock
	for {
		candidate, err := openSharedBrokerLock(resolved.Paths.BrokerLock)
		if err == nil {
			lock = candidate
			break
		}
		var shared *SharedRuntimeError
		if !errors.As(err, &shared) || shared.Code != "broker_election_lost" {
			return SharedRuntimeStopResult{}, err
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(minSharedDuration(20*time.Millisecond, time.Until(deadline)))
	}
	if lock == nil {
		return SharedRuntimeStopResult{}, sharedRuntimeError("runtime_shutdown_timeout", errors.New("broker lock was not released"))
	}
	defer lock.Close()
	runtimePID := 0
	runtimeTerminated := false
	if record.Runtime != nil {
		runtimePID = record.Runtime.PID
		if err := reclaimSharedRuntime(record.Runtime, resolved.Profile.Runtime.ShutdownTimeoutSeconds); err != nil {
			return SharedRuntimeStopResult{}, err
		}
		runtimeTerminated = true
	}
	if err := cleanupSharedRuntimeState(resolved.Paths); err != nil {
		return SharedRuntimeStopResult{}, err
	}
	return SharedRuntimeStopResult{State: "absent", BrokerPID: record.Broker.PID, RuntimePID: runtimePID, BrokerTerminated: true, RuntimeTerminated: runtimeTerminated}, nil
}

func waitRecordedBrokerGone(record SharedBrokerIdentity, deadline time.Time) error {
	for {
		observation, err := inspectSharedProcessKernel(record.PID)
		if sharedProcessGone(err) || (err == nil && (!observation.live() || observation.StartTime != record.StartTime)) {
			return nil
		}
		if err != nil {
			return sharedRuntimeError("broker_stop_identity_mismatch", err)
		}
		if !time.Now().Before(deadline) {
			return sharedRuntimeError("runtime_shutdown_timeout", errors.New("broker remained live"))
		}
		time.Sleep(minSharedDuration(20*time.Millisecond, time.Until(deadline)))
	}
}

func minSharedDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}
