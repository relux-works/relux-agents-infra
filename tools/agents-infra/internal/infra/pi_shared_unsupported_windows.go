//go:build windows

package infra

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type SharedRuntimeError struct {
	Code string `json:"code"`
	Err  error  `json:"-"`
}

func (e *SharedRuntimeError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *SharedRuntimeError) Unwrap() error { return e.Err }

func SharedRuntimeExitCode(err error) (int, bool) {
	var shared *SharedRuntimeError
	return 1, errors.As(err, &shared)
}

type SharedRuntimePaths struct {
	RuntimeKey    string `json:"runtime_key"`
	RestartLedger string `json:"restart_ledger"`
}

type SharedRuntimeRestartLedger struct {
	Schema             string                      `json:"schema"`
	RuntimeKey         string                      `json:"runtime_key"`
	ProfileDigest      string                      `json:"profile_digest"`
	RestartCount       int                         `json:"restart_count"`
	RestartNotBefore   *time.Time                  `json:"restart_not_before"`
	QuarantinedUntil   *time.Time                  `json:"quarantined_until"`
	LastReadinessMatch *time.Time                  `json:"last_readiness_match"`
	ManualQuarantine   bool                        `json:"manual_quarantine"`
	HalfOpen           bool                        `json:"half_open"`
	FailureHistory     []SharedRuntimeFailureEvent `json:"failure_history"`
}

// SharedRuntimeStatusContractVersion and SharedRuntimeStatusMinSupportedContractVersion
// mirror the !windows contract in pi_shared_supervision.go; see that file for
// the compatibility rule. Windows never produces a real payload, but any
// consumer built for this platform must see the same contract.
const SharedRuntimeStatusContractVersion = 1
const SharedRuntimeStatusMinSupportedContractVersion = 1

type SharedRuntimeFailureEvent struct {
	OccurredAt       time.Time  `json:"occurred_at"`
	RestartCount     int        `json:"restart_count"`
	BackoffSeconds   int        `json:"backoff_seconds,omitempty"`
	Quarantined      bool       `json:"quarantined"`
	QuarantinedUntil *time.Time `json:"quarantined_until,omitempty"`
}

// DecodeSharedRuntimeStatus mirrors the !windows implementation in
// pi_shared_supervision.go.
func DecodeSharedRuntimeStatus(data []byte) (SharedRuntimeStatus, error) {
	var probe struct {
		ContractVersion int `json:"contract_version"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return SharedRuntimeStatus{}, &SharedRuntimeError{Code: "shared_runtime_status_undecodable", Err: err}
	}
	if probe.ContractVersion < SharedRuntimeStatusMinSupportedContractVersion || probe.ContractVersion > SharedRuntimeStatusContractVersion {
		return SharedRuntimeStatus{}, &SharedRuntimeError{
			Code: "shared_runtime_status_unsupported_contract_version",
			Err: fmt.Errorf("shared runtime status contract version %d is not supported; this build supports %d..%d",
				probe.ContractVersion, SharedRuntimeStatusMinSupportedContractVersion, SharedRuntimeStatusContractVersion),
		}
	}
	var status SharedRuntimeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return SharedRuntimeStatus{}, &SharedRuntimeError{Code: "shared_runtime_status_undecodable", Err: err}
	}
	return status, nil
}

func SharedRuntimeProfileDigest(profile PiProfile) string {
	budget := "absent"
	if profile.CacheBudgetBytes != nil {
		budget = strconv.FormatInt(*profile.CacheBudgetBytes, 10)
	}
	sum := sha256.Sum256([]byte(profile.Provider + "\x00" + profile.Model + "\x00" + profile.BaseURL + "\x00" + budget))
	return hex.EncodeToString(sum[:])
}

func SharedRuntimeExecPlanDigest(profile PiProfile, cwd string) string {
	budget := "absent"
	if profile.CacheBudgetBytes != nil {
		budget = strconv.FormatInt(*profile.CacheBudgetBytes, 10)
	}
	sum := sha256.Sum256([]byte(profile.Runtime.Executable + "\x00" + cwd + "\x00" + budget))
	return hex.EncodeToString(sum[:])
}

func SharedRuntimeKey(profile PiProfile) (string, string) {
	digest := SharedRuntimeProfileDigest(profile)
	sum := sha256.Sum256([]byte("agents-infra.pi.shared-runtime.v1\x00" + profile.BaseURL + "\x00" + digest))
	return hex.EncodeToString(sum[:]), digest
}

func ResolveSharedRuntimePaths(string, string) (SharedRuntimePaths, error) {
	return SharedRuntimePaths{}, unsupportedSharedRuntimePlatform()
}

type SharedRuntimeBrokerOptions struct {
	RuntimeKey     string
	ProfileProject string
	ProfileName    string
	HomeDir        string
	CacheRoot      string
	Environ        []string
}

type SharedRuntimeLauncherOptions = SharedRuntimeBrokerOptions

type SharedRuntimeOperatorOptions struct {
	ProjectDir string
	Profile    string
	HomeDir    string
	CacheRoot  string
	HTTPClient *http.Client
}

type SharedRuntimeBrokerStatus struct {
	State string `json:"state"`
	PID   int    `json:"pid,omitempty"`
}

type SharedRuntimeProcessStatus struct {
	PID      int    `json:"pid"`
	Endpoint string `json:"endpoint"`
}

type SharedRuntimeSharingStatus struct {
	Configured PiRuntimeSharing  `json:"configured"`
	Effective  *PiRuntimeSharing `json:"effective,omitempty"`
}

type SharedRuntimeStatus struct {
	ContractVersion    int                         `json:"contract_version"`
	RuntimeKey         string                      `json:"runtime_key"`
	ProfileDigest      string                      `json:"profile_digest"`
	RestartCount       int                         `json:"restart_count"`
	RestartNotBefore   *time.Time                  `json:"restart_not_before"`
	QuarantinedUntil   *time.Time                  `json:"quarantined_until"`
	LastReadinessMatch *time.Time                  `json:"last_readiness_match"`
	ManualQuarantine   bool                        `json:"manual_quarantine"`
	HalfOpen           bool                        `json:"half_open"`
	FailureHistory     []SharedRuntimeFailureEvent `json:"failure_history"`
	Resources          SharedRuntimeResourceStatus `json:"resources"`
	Broker             SharedRuntimeBrokerStatus   `json:"broker"`
	Sharing            SharedRuntimeSharingStatus  `json:"sharing"`
	Runtime            *SharedRuntimeProcessStatus `json:"runtime,omitempty"`
	Leases             []any                       `json:"leases"`
}

func SetSharedRuntimeManualQuarantine(SharedRuntimeOperatorOptions, bool) (SharedRuntimeRestartLedger, error) {
	return SharedRuntimeRestartLedger{}, unsupportedSharedRuntimePlatform()
}

type SharedRuntimeStopResult struct {
	State             string `json:"state"`
	BrokerPID         int    `json:"broker_pid,omitempty"`
	RuntimePID        int    `json:"runtime_pid,omitempty"`
	BrokerTerminated  bool   `json:"broker_terminated"`
	RuntimeTerminated bool   `json:"runtime_terminated"`
}

func unsupportedSharedRuntimePlatform() error {
	return &SharedRuntimeError{Code: "shared_runtime_platform_unsupported", Err: errors.New("shared Pi runtimes require Darwin process attestation")}
}

func RunSharedRuntimeBroker(SharedRuntimeBrokerOptions) error {
	return unsupportedSharedRuntimePlatform()
}
func RunSharedRuntimeLauncher(SharedRuntimeLauncherOptions) error {
	return unsupportedSharedRuntimePlatform()
}
func SharedRuntimeStatusReport(SharedRuntimeOperatorOptions) (SharedRuntimeStatus, error) {
	return SharedRuntimeStatus{}, unsupportedSharedRuntimePlatform()
}
func StopSharedRuntime(SharedRuntimeOperatorOptions, bool, time.Duration) (SharedRuntimeStopResult, error) {
	return SharedRuntimeStopResult{}, unsupportedSharedRuntimePlatform()
}
