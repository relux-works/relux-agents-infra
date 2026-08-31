//go:build !windows

package infra

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

const sharedRuntimeRestartLedgerSchema = "agents-infra.pi.shared-runtime.restart-ledger.v1"

// sharedRuntimeFailureHistoryLimit bounds the failure/backoff evidence a
// ledger retains. Without a bound, a multi-week unattended run accumulates
// one entry per failed restart attempt forever; this caps that growth and
// evicts the oldest entry first, keeping only the most recent attempts.
const sharedRuntimeFailureHistoryLimit = 20

// SharedRuntimeStatusContractVersion is the current published version of the
// SharedRuntimeStatus wire contract emitted by "runtime status --json". Bump
// it whenever a change removes or redefines the meaning of an existing
// field; purely additive fields never require a bump. A consumer compiled
// against version N understands contract versions
// SharedRuntimeStatusMinSupportedContractVersion..N inclusive and must
// refuse anything outside that range through DecodeSharedRuntimeStatus
// rather than parsing the fields it recognises from an unsupported version
// and silently ignoring the rest.
const SharedRuntimeStatusContractVersion = 1

// SharedRuntimeStatusMinSupportedContractVersion is the oldest contract
// version this build still understands. It only advances when an older
// version is deliberately retired, never implicitly.
const SharedRuntimeStatusMinSupportedContractVersion = 1

type SharedRuntimeFailureEvent struct {
	OccurredAt       time.Time  `json:"occurred_at"`
	RestartCount     int        `json:"restart_count"`
	BackoffSeconds   int        `json:"backoff_seconds,omitempty"`
	Quarantined      bool       `json:"quarantined"`
	QuarantinedUntil *time.Time `json:"quarantined_until,omitempty"`
}

// DecodeSharedRuntimeStatus is the sanctioned entry point for any consumer
// reading a SharedRuntimeStatus payload produced by "runtime status --json".
// It refuses a payload declaring a contract version outside
// [SharedRuntimeStatusMinSupportedContractVersion, SharedRuntimeStatusContractVersion]
// before trusting any other field, naming both the observed version and the
// supported range. It never falls back to parsing the fields it recognises
// from an unsupported version and dropping the rest.
func DecodeSharedRuntimeStatus(data []byte) (SharedRuntimeStatus, error) {
	var probe struct {
		ContractVersion int `json:"contract_version"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return SharedRuntimeStatus{}, sharedRuntimeError("shared_runtime_status_undecodable", err)
	}
	if probe.ContractVersion < SharedRuntimeStatusMinSupportedContractVersion || probe.ContractVersion > SharedRuntimeStatusContractVersion {
		return SharedRuntimeStatus{}, &SharedRuntimeError{
			Code: "shared_runtime_status_unsupported_contract_version",
			Details: map[string]any{
				"observed_contract_version":      probe.ContractVersion,
				"supported_contract_version_min": SharedRuntimeStatusMinSupportedContractVersion,
				"supported_contract_version_max": SharedRuntimeStatusContractVersion,
			},
			Err: fmt.Errorf("shared runtime status contract version %d is not supported; this build supports %d..%d",
				probe.ContractVersion, SharedRuntimeStatusMinSupportedContractVersion, SharedRuntimeStatusContractVersion),
		}
	}
	var status SharedRuntimeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return SharedRuntimeStatus{}, sharedRuntimeError("shared_runtime_status_undecodable", err)
	}
	return status, nil
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

type sharedRuntimeRestartDecision struct {
	Backoff     time.Duration
	Quarantined bool
}

func newSharedRuntimeRestartLedger(runtimeKey, profileDigest string) SharedRuntimeRestartLedger {
	return SharedRuntimeRestartLedger{Schema: sharedRuntimeRestartLedgerSchema, RuntimeKey: runtimeKey, ProfileDigest: profileDigest}
}

func readSharedRuntimeRestartLedger(path, runtimeKey, profileDigest string) (SharedRuntimeRestartLedger, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newSharedRuntimeRestartLedger(runtimeKey, profileDigest), nil
	}
	if err != nil {
		return SharedRuntimeRestartLedger{}, sharedRuntimeError("shared_runtime_state_unreadable", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var ledger SharedRuntimeRestartLedger
	if err := decoder.Decode(&ledger); err != nil || requireJSONEOF(decoder) != nil {
		return SharedRuntimeRestartLedger{}, sharedRuntimeError("shared_runtime_state_unreadable", errors.New("restart ledger is malformed"))
	}
	if ledger.Schema != sharedRuntimeRestartLedgerSchema || ledger.RuntimeKey != runtimeKey || ledger.ProfileDigest != profileDigest || ledger.RestartCount < 0 {
		return SharedRuntimeRestartLedger{}, sharedRuntimeError("shared_runtime_state_unreadable", errors.New("restart ledger identity or counters differ"))
	}
	return ledger, nil
}

func writeSharedRuntimeRestartLedger(path string, ledger SharedRuntimeRestartLedger) error {
	return writeSharedJSONAtomic(path, ledger)
}

func sharedRuntimeBeginAttempt(ledger *SharedRuntimeRestartLedger, now time.Time) error {
	if ledger.ManualQuarantine {
		return &SharedRuntimeError{Code: "shared_runtime_quarantined", Details: map[string]any{"manual": true}, Err: errors.New("shared runtime is manually quarantined")}
	}
	if ledger.QuarantinedUntil == nil {
		return nil
	}
	if now.Before(*ledger.QuarantinedUntil) {
		return &SharedRuntimeError{Code: "shared_runtime_quarantined", Details: map[string]any{"quarantined_until": *ledger.QuarantinedUntil}, Err: errors.New("shared runtime quarantine is active")}
	}
	ledger.QuarantinedUntil = nil
	ledger.RestartNotBefore = nil
	ledger.HalfOpen = true
	return nil
}

func sharedRuntimeRecordReadiness(ledger *SharedRuntimeRestartLedger, now time.Time) {
	matched := now.UTC()
	ledger.LastReadinessMatch = &matched
	ledger.RestartNotBefore = nil
}

func sharedRuntimeResetStableRun(ledger *SharedRuntimeRestartLedger) {
	ledger.RestartCount = 0
	ledger.HalfOpen = false
}

func sharedRuntimeRecordFailure(ledger *SharedRuntimeRestartLedger, policy PiRuntimeSharing, now time.Time) sharedRuntimeRestartDecision {
	ledger.RestartCount++
	occurredAt := now.UTC()
	if ledger.HalfOpen || ledger.RestartCount >= policy.RestartLimit {
		until := occurredAt.Add(time.Duration(policy.QuarantineSeconds) * time.Second)
		ledger.QuarantinedUntil = &until
		ledger.HalfOpen = false
		ledger.RestartNotBefore = nil
		appendSharedRuntimeFailureEvent(ledger, SharedRuntimeFailureEvent{
			OccurredAt: occurredAt, RestartCount: ledger.RestartCount,
			Quarantined: true, QuarantinedUntil: &until,
		})
		return sharedRuntimeRestartDecision{Quarantined: true}
	}
	delay := time.Duration(policy.RestartInitialBackoffSeconds) * time.Second
	maximum := time.Duration(policy.RestartMaxBackoffSeconds) * time.Second
	for count := 1; count < ledger.RestartCount && delay < maximum; count++ {
		if delay > maximum-delay {
			delay = maximum
			break
		}
		delay *= 2
	}
	notBefore := occurredAt.Add(delay)
	ledger.RestartNotBefore = &notBefore
	appendSharedRuntimeFailureEvent(ledger, SharedRuntimeFailureEvent{
		OccurredAt: occurredAt, RestartCount: ledger.RestartCount,
		BackoffSeconds: int(delay / time.Second),
	})
	return sharedRuntimeRestartDecision{Backoff: delay}
}

// appendSharedRuntimeFailureEvent enforces sharedRuntimeFailureHistoryLimit
// by evicting the oldest retained event first (FIFO), so the ledger's
// failure/backoff evidence never grows past the bound regardless of how many
// restart attempts a long unattended run accumulates.
func appendSharedRuntimeFailureEvent(ledger *SharedRuntimeRestartLedger, event SharedRuntimeFailureEvent) {
	ledger.FailureHistory = append(ledger.FailureHistory, event)
	if len(ledger.FailureHistory) > sharedRuntimeFailureHistoryLimit {
		overflow := len(ledger.FailureHistory) - sharedRuntimeFailureHistoryLimit
		ledger.FailureHistory = append([]SharedRuntimeFailureEvent(nil), ledger.FailureHistory[overflow:]...)
	}
}

func sharedRuntimeRestartDelay(ledger SharedRuntimeRestartLedger, now time.Time) time.Duration {
	if ledger.RestartNotBefore == nil || !now.Before(*ledger.RestartNotBefore) {
		return 0
	}
	return ledger.RestartNotBefore.Sub(now)
}

func setSharedRuntimeManualQuarantine(paths SharedRuntimePaths, runtimeKey, profileDigest string, enabled bool) (SharedRuntimeRestartLedger, error) {
	ledger, err := readSharedRuntimeRestartLedger(paths.RestartLedger, runtimeKey, profileDigest)
	if err != nil {
		return SharedRuntimeRestartLedger{}, err
	}
	ledger.ManualQuarantine = enabled
	if enabled {
		ledger.HalfOpen = false
		ledger.RestartNotBefore = nil
	} else {
		ledger.QuarantinedUntil = nil
		ledger.HalfOpen = false
	}
	if err := writeSharedRuntimeRestartLedger(paths.RestartLedger, ledger); err != nil {
		return SharedRuntimeRestartLedger{}, fmt.Errorf("write manual shared runtime quarantine: %w", err)
	}
	return ledger, nil
}
