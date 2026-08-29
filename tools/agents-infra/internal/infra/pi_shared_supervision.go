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
	if ledger.HalfOpen || ledger.RestartCount >= policy.RestartLimit {
		until := now.UTC().Add(time.Duration(policy.QuarantineSeconds) * time.Second)
		ledger.QuarantinedUntil = &until
		ledger.HalfOpen = false
		ledger.RestartNotBefore = nil
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
	notBefore := now.UTC().Add(delay)
	ledger.RestartNotBefore = &notBefore
	return sharedRuntimeRestartDecision{Backoff: delay}
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
