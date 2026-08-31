//go:build darwin

package infra

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Production call sites exercised without wall-clock sleeps or live runtime
// contact: openPiSessionLog/PiLifecycleStatus, restart-ledger read/write and
// crash decisions, sharedBrokerServer.statusSnapshotLocked, and
// classifySharedRuntimeResources. The fake timeline composes hourly, daily,
// and eight-week lifecycle pressure with host reload, backend health loss,
// stale leases, ledger corruption, quarantine, half-open recovery, and memory
// pressure hysteresis.
func TestPiRetentionPlaneDeterministicEightWeekCrashLeaseReloadPressureSoak(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	policy.MaxCount = 8
	policy.MaxMutationsPerOperation = 8
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base
	originalNow := piLifecycleNow
	t.Cleanup(func() { piLifecycleNow = originalNow })
	piLifecycleNow = func() time.Time { return current }

	seedClosedPiLifecycleEntry(t, paths, policy)
	legacyPath := filepath.Join(paths.LogsDir, "pre-retention-legacy.jsonl")
	if err := os.WriteFile(legacyPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "soak")
	if err != nil {
		t.Fatal(err)
	}
	retired, err := PiLegacyRetire(context.Background(), paths, policy, "soak", plan.PlanHash)
	if err != nil || !retired.Status.SoakReady {
		t.Fatalf("initial retirement=%+v err=%v", retired, err)
	}

	pressurePolicy := testSharedRuntimeResourcePolicy()
	sharing := PiRuntimeSharing{
		Mode: "shared", MaxLeases: 4, LeaseStaleSeconds: 6 * 60 * 60,
		RestartLimit: 3, RestartInitialBackoffSeconds: 60 * 60,
		RestartMaxBackoffSeconds: 4 * 60 * 60, StableRunSeconds: 12 * 60 * 60,
		QuarantineSeconds: 24 * 60 * 60, ResourcePressureMode: "provider",
		ResourcePressure: &pressurePolicy,
	}
	ledgerPath := filepath.Join(t.TempDir(), "restart", "ledger.json")
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o700); err != nil {
		t.Fatal(err)
	}
	ledger := newSharedRuntimeRestartLedger("runtime-key", "profile-digest")
	server := &sharedBrokerServer{
		resolved: sharedResolvedProfile{Sharing: sharing}, state: "serving",
		leases: map[string]*SharedLeaseRecord{},
	}

	appendManaged := func(label string) {
		t.Helper()
		log, err := openPiSessionLog(context.Background(), paths, policy)
		if err != nil {
			t.Fatalf("%s open: %v", label, err)
		}
		if err := log.event(context.Background(), "soak", map[string]any{"step": label}); err != nil {
			t.Fatalf("%s append: %v", label, err)
		}
		if err := log.close(context.Background()); err != nil {
			t.Fatalf("%s close: %v", label, err)
		}
		status, err := PiLifecycleStatus(context.Background(), paths, policy, "soak", "")
		if err != nil || !status.ScanComplete || !status.WithinPolicy || !status.SoakReady || status.ManagedCount > policy.MaxCount {
			t.Fatalf("%s status=%+v err=%v", label, status, err)
		}
	}

	for hour := 1; hour <= 24; hour++ {
		current = base.Add(time.Duration(hour) * time.Hour)
		appendManaged("hour")
	}

	for crash := 1; crash <= sharing.RestartLimit; crash++ {
		decision := sharedRuntimeRecordFailure(&ledger, sharing, current)
		if crash < sharing.RestartLimit {
			if decision.Quarantined || decision.Backoff <= 0 || sharedRuntimeRestartDelay(ledger, current) != decision.Backoff {
				t.Fatalf("crash %d decision=%+v ledger=%+v", crash, decision, ledger)
			}
			current = current.Add(decision.Backoff)
		} else if !decision.Quarantined || ledger.QuarantinedUntil == nil {
			t.Fatalf("crash loop did not quarantine: decision=%+v ledger=%+v", decision, ledger)
		}
		if err := writeSharedRuntimeRestartLedger(ledgerPath, ledger); err != nil {
			t.Fatal(err)
		}
		ledger, err = readSharedRuntimeRestartLedger(ledgerPath, "runtime-key", "profile-digest")
		if err != nil {
			t.Fatalf("host restart reload %d: %v", crash, err)
		}
	}
	current = ledger.QuarantinedUntil.Add(time.Second)
	if err := sharedRuntimeBeginAttempt(&ledger, current); err != nil || !ledger.HalfOpen {
		t.Fatalf("post-quarantine half-open=%+v err=%v", ledger, err)
	}
	sharedRuntimeRecordReadiness(&ledger, current)
	sharedRuntimeResetStableRun(&ledger)
	if ledger.RestartCount != 0 || ledger.HalfOpen || ledger.LastReadinessMatch == nil {
		t.Fatalf("stable backend readiness did not reset crash loop: %+v", ledger)
	}

	for day := 2; day <= 14; day++ {
		current = base.Add(time.Duration(day) * 24 * time.Hour)
		appendManaged("day")
		lease := &SharedLeaseRecord{LeaseID: "lease", GrantedAt: current.Add(-8 * time.Hour), LastHeartbeatAt: current.Add(-7 * time.Hour)}
		server.leases[lease.LeaseID] = lease
		_, leases := server.statusSnapshotLocked(current)
		if len(leases) != 1 || leases[0].State != "held(stale)" {
			t.Fatalf("day %d stale lease health=%+v", day, leases)
		}
		lease.LastHeartbeatAt = current
		_, leases = server.statusSnapshotLocked(current)
		if leases[0].State != "held" {
			t.Fatalf("day %d renewed lease health=%+v", day, leases)
		}
		delete(server.leases, lease.LeaseID)
		if err := writeSharedRuntimeRestartLedger(ledgerPath, ledger); err != nil {
			t.Fatal(err)
		}
		ledger, err = readSharedRuntimeRestartLedger(ledgerPath, "runtime-key", "profile-digest")
		if err != nil {
			t.Fatalf("day %d reload: %v", day, err)
		}
	}

	validLedger := ledger
	if err := os.WriteFile(ledgerPath, []byte("{malformed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSharedRuntimeRestartLedger(ledgerPath, "runtime-key", "profile-digest"); err == nil {
		t.Fatal("corrupt restart ledger was treated as absence")
	} else {
		var shared *SharedRuntimeError
		if !errors.As(err, &shared) || shared.Code != "shared_runtime_state_unreadable" {
			t.Fatalf("corrupt ledger error=%v", err)
		}
	}
	if err := writeSharedRuntimeRestartLedger(ledgerPath, validLedger); err != nil {
		t.Fatal(err)
	}

	pressureLatched := false
	for week := 1; week <= 8; week++ {
		current = base.Add(time.Duration(week) * 7 * 24 * time.Hour)
		appendManaged("week")
		healthy, next := classifySharedRuntimeResources(soakResourceObservation(700, "idle"), pressurePolicy, "serving", pressureLatched, current)
		if healthy.State != SharedRuntimeResourceHealthy || next {
			t.Fatalf("week %d healthy resources=%+v latched=%t", week, healthy, next)
		}
		pressured, next := classifySharedRuntimeResources(soakResourceObservation(1000, "idle"), pressurePolicy, "serving", next, current)
		if pressured.State != SharedRuntimeResourcePressured || pressured.Admission != SharedRuntimeAdmissionRefused || !next {
			t.Fatalf("week %d pressure resources=%+v latched=%t", week, pressured, next)
		}
		unknown := sharedRuntimeProviderResourceObservation{
			Schema: SharedRuntimeResourceStatusSchema, Model: "Model",
			LoadedModelMemory: sharedRuntimeProviderMemoryFact{State: "unknown", Reason: "backend_reload_corruption"},
			Inference:         sharedRuntimeProviderInferenceFact{State: "unknown", Reason: "backend_reload_corruption"},
		}
		lost, _ := classifySharedRuntimeResources(unknown, pressurePolicy, "serving", next, current)
		if lost.State != SharedRuntimeResourceUnknown || lost.Admission != SharedRuntimeAdmissionRefused {
			t.Fatalf("week %d backend health loss was guessed: %+v", week, lost)
		}
		recovered, pressureLatched := classifySharedRuntimeResources(soakResourceObservation(800, "busy"), pressurePolicy, "serving", next, current)
		if recovered.State != SharedRuntimeResourceBusy || pressureLatched {
			t.Fatalf("week %d pressure recovery=%+v latched=%t", week, recovered, pressureLatched)
		}
	}

	status, err := PiLifecycleStatus(context.Background(), paths, policy, "soak", "")
	if err != nil || !status.ScanComplete || !status.WithinPolicy || !status.SoakReady || status.ManagedCount != policy.MaxCount || status.LegacyCount != 0 || status.UnknownCount != 0 {
		t.Fatalf("eight-week final status=%+v err=%v", status, err)
	}
}

func soakResourceObservation(bytes uint64, inference string) sharedRuntimeProviderResourceObservation {
	return sharedRuntimeProviderResourceObservation{
		Schema: SharedRuntimeResourceStatusSchema, Model: "Model",
		LoadedModelMemory: sharedRuntimeProviderMemoryFact{State: "observed", Bytes: &bytes},
		Inference:         sharedRuntimeProviderInferenceFact{State: inference},
	}
}
