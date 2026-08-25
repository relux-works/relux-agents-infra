//go:build !darwin && !windows

package infra

import (
	"errors"
	"net/http"
	"time"
)

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
	RuntimeKey    string                      `json:"runtime_key"`
	ProfileDigest string                      `json:"profile_digest"`
	Broker        SharedRuntimeBrokerStatus   `json:"broker"`
	Sharing       SharedRuntimeSharingStatus  `json:"sharing"`
	Runtime       *SharedRuntimeProcessStatus `json:"runtime,omitempty"`
	Leases        []any                       `json:"leases"`
}

type SharedRuntimeStopResult struct {
	State             string `json:"state"`
	BrokerPID         int    `json:"broker_pid,omitempty"`
	RuntimePID        int    `json:"runtime_pid,omitempty"`
	BrokerTerminated  bool   `json:"broker_terminated"`
	RuntimeTerminated bool   `json:"runtime_terminated"`
}

func unsupportedSharedRuntimePlatform() error {
	return sharedRuntimeError("shared_runtime_platform_unsupported", errors.New("shared Pi runtimes require Darwin process attestation"))
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

func runSharedPiSession(RunPiOptions, string, string, PiProfile, PiArgumentPlan, PiExecutionIdentity, runtimeExecutableIdentity) error {
	return unsupportedSharedRuntimePlatform()
}
