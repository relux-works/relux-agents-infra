//go:build windows

package infra

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
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
	Schema             string     `json:"schema"`
	RuntimeKey         string     `json:"runtime_key"`
	ProfileDigest      string     `json:"profile_digest"`
	RestartCount       int        `json:"restart_count"`
	RestartNotBefore   *time.Time `json:"restart_not_before"`
	QuarantinedUntil   *time.Time `json:"quarantined_until"`
	LastReadinessMatch *time.Time `json:"last_readiness_match"`
	ManualQuarantine   bool       `json:"manual_quarantine"`
	HalfOpen           bool       `json:"half_open"`
}

func SharedRuntimeProfileDigest(profile PiProfile) string {
	sum := sha256.Sum256([]byte(profile.Provider + "\x00" + profile.Model + "\x00" + profile.BaseURL))
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
	RuntimeKey         string                      `json:"runtime_key"`
	ProfileDigest      string                      `json:"profile_digest"`
	RestartCount       int                         `json:"restart_count"`
	RestartNotBefore   *time.Time                  `json:"restart_not_before"`
	QuarantinedUntil   *time.Time                  `json:"quarantined_until"`
	LastReadinessMatch *time.Time                  `json:"last_readiness_match"`
	ManualQuarantine   bool                        `json:"manual_quarantine"`
	HalfOpen           bool                        `json:"half_open"`
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
