//go:build darwin

package infra

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func testSharedRuntimeSupervisionPolicy() PiRuntimeSharing {
	return PiRuntimeSharing{MaxSegmentBytes: 1024, MaxSegments: 2, RestartLimit: 4, RestartInitialBackoffSeconds: 2, RestartMaxBackoffSeconds: 5, StableRunSeconds: 9, QuarantineSeconds: 30}
}

func shortSharedRuntimeCache(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp("/tmp", "sr-ledger-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	return directory
}

func TestSharedRuntimeRestartLedgerPersistsAcrossBrokerInstancesAtResolvedPath(t *testing.T) {
	cache := shortSharedRuntimeCache(t)
	key := exactStateKey("restart-ledger")
	paths, err := ResolveSharedRuntimePaths(cache, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(paths); err != nil {
		t.Fatal(err)
	}
	ledger := newSharedRuntimeRestartLedger(key, "profile")
	ledger.RestartCount = 3
	sharedRuntimeRecordReadiness(&ledger, time.Unix(100, 0))
	if err := writeSharedRuntimeRestartLedger(paths.RestartLedger, ledger); err != nil {
		t.Fatal(err)
	}
	restartedBrokerLedger, err := readSharedRuntimeRestartLedger(paths.RestartLedger, key, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if restartedBrokerLedger.RestartCount != 3 || restartedBrokerLedger.LastReadinessMatch == nil || !restartedBrokerLedger.LastReadinessMatch.Equal(time.Unix(100, 0)) {
		t.Fatalf("persisted ledger=%#v", restartedBrokerLedger)
	}
}

func TestSharedRuntimeRestartPolicyIsBoundedStableAndHalfOpen(t *testing.T) {
	policy := testSharedRuntimeSupervisionPolicy()
	ledger := newSharedRuntimeRestartLedger("runtime", "profile")
	now := time.Unix(1000, 0).UTC()
	wants := []time.Duration{2 * time.Second, 4 * time.Second, 5 * time.Second}
	for index, want := range wants {
		decision := sharedRuntimeRecordFailure(&ledger, policy, now)
		if decision.Quarantined || decision.Backoff != want {
			t.Fatalf("failure %d decision=%#v want backoff=%s", index+1, decision, want)
		}
	}
	decision := sharedRuntimeRecordFailure(&ledger, policy, now)
	if !decision.Quarantined || ledger.QuarantinedUntil == nil || !ledger.QuarantinedUntil.Equal(now.Add(30*time.Second)) {
		t.Fatalf("quarantine decision=%#v ledger=%#v", decision, ledger)
	}
	if err := sharedRuntimeBeginAttempt(&ledger, now.Add(29*time.Second)); sharedRuntimeErrorCode(err) != "shared_runtime_quarantined" {
		t.Fatalf("active quarantine admitted: %v", err)
	}
	if err := sharedRuntimeBeginAttempt(&ledger, now.Add(30*time.Second)); err != nil || !ledger.HalfOpen || ledger.QuarantinedUntil != nil {
		t.Fatalf("automatic half-open failed: ledger=%#v err=%v", ledger, err)
	}
	decision = sharedRuntimeRecordFailure(&ledger, policy, now.Add(31*time.Second))
	if !decision.Quarantined {
		t.Fatalf("failed half-open did not re-quarantine: %#v", decision)
	}
	sharedRuntimeResetStableRun(&ledger)
	if ledger.RestartCount != 0 || ledger.HalfOpen {
		t.Fatalf("stable run did not reset ledger: %#v", ledger)
	}
	if got := sharedRuntimeRecordFailure(&ledger, policy, now).Backoff; got != 2*time.Second {
		t.Fatalf("post-stable backoff=%s want=2s", got)
	}
}

func TestSharedRuntimeRestartBackoffClampsBeforeDurationOverflow(t *testing.T) {
	policy := PiRuntimeSharing{
		MaxSegmentBytes:              1024,
		MaxSegments:                  2,
		RestartLimit:                 3,
		RestartInitialBackoffSeconds: int(maxTimeDurationSeconds - 1),
		RestartMaxBackoffSeconds:     int(maxTimeDurationSeconds),
		QuarantineSeconds:            1,
	}
	ledger := newSharedRuntimeRestartLedger("runtime", "profile")
	ledger.RestartCount = 1
	decision := sharedRuntimeRecordFailure(&ledger, policy, time.Unix(1000, 0).UTC())
	want := time.Duration(maxTimeDurationSeconds) * time.Second
	if decision.Quarantined || decision.Backoff != want || decision.Backoff <= 0 {
		t.Fatalf("overflowing exponential backoff decision=%#v want=%s", decision, want)
	}
}

func TestSharedRuntimeManualQuarantinePersistsAndRefusesAttempt(t *testing.T) {
	cache := shortSharedRuntimeCache(t)
	key := exactStateKey("manual-quarantine")
	paths, err := ResolveSharedRuntimePaths(cache, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(paths); err != nil {
		t.Fatal(err)
	}
	ledger, err := setSharedRuntimeManualQuarantine(paths, key, "profile", true)
	if err != nil {
		t.Fatal(err)
	}
	if !ledger.ManualQuarantine || sharedRuntimeErrorCode(sharedRuntimeBeginAttempt(&ledger, time.Now())) != "shared_runtime_quarantined" {
		t.Fatalf("manual quarantine admitted: %#v", ledger)
	}
	loaded, err := readSharedRuntimeRestartLedger(paths.RestartLedger, key, "profile")
	if err != nil || !loaded.ManualQuarantine {
		t.Fatalf("persisted manual quarantine=%#v err=%v", loaded, err)
	}
	loaded, err = setSharedRuntimeManualQuarantine(paths, key, "profile", false)
	if err != nil || loaded.ManualQuarantine {
		t.Fatalf("manual clear=%#v err=%v", loaded, err)
	}
}

func sharedRuntimeErrorCode(err error) string {
	var shared *SharedRuntimeError
	if errors.As(err, &shared) {
		return shared.Code
	}
	return ""
}

func TestSharedRuntimeStatusJSONCarriesLifecycleFacts(t *testing.T) {
	now := time.Unix(2000, 0).UTC()
	report := SharedRuntimeStatus{
		RestartCount: 2, RestartNotBefore: &now, QuarantinedUntil: &now,
		LastReadinessMatch: &now, HalfOpen: true,
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		`"restart_count":2`, `"restart_not_before":`, `"quarantined_until":`,
		`"last_readiness_match":`, `"half_open":true`,
	} {
		if !bytes.Contains(data, []byte(field)) {
			t.Fatalf("status JSON %s lacks %s", data, field)
		}
	}
	if bytes.Contains(data, []byte(`"last_failure`)) {
		t.Fatalf("status JSON invented deferred last-failure evidence: %s", data)
	}
}

func TestSharedRuntimeStatusServingAfterRestartHasNoInferredBackoff(t *testing.T) {
	readyAt := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	report := SharedRuntimeStatus{
		RestartCount: 2, LastReadinessMatch: &readyAt, HalfOpen: true,
		Broker:  SharedRuntimeBrokerStatus{State: "ready", Source: "attested"},
		Runtime: &SharedRuntimeProcessStatus{Source: "attested", Endpoint: "http://127.0.0.1:18011/v1"},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"restart_not_before":null`)) {
		t.Fatalf("serving status inferred backoff from restart_count: %s", data)
	}
}

func TestSharedRuntimeStatusWireFixturesAreAdditiveAndTimestampStrict(t *testing.T) {
	const preExtension = `{"restart_count":2,"quarantined_until":null,"last_readiness_match":"2026-08-29T12:00:00Z","manual_quarantine":false}`
	var legacy SharedRuntimeStatus
	if err := json.Unmarshal([]byte(preExtension), &legacy); err != nil {
		t.Fatalf("decode pre-extension fixture: %v", err)
	}
	if legacy.RestartNotBefore != nil || legacy.HalfOpen {
		t.Fatalf("pre-extension fixture fabricated new facts: %#v", legacy)
	}
	var legacyFields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(preExtension), &legacyFields); err != nil {
		t.Fatal(err)
	}
	if _, present := legacyFields["restart_not_before"]; present {
		t.Fatalf("pre-extension fixture unexpectedly contains restart_not_before: %s", preExtension)
	}

	const postExtension = `{"restart_count":2,"restart_not_before":"2026-08-29T12:00:04Z","quarantined_until":null,"last_readiness_match":"2026-08-29T12:00:00Z","manual_quarantine":false,"half_open":true}`
	var widened SharedRuntimeStatus
	if err := json.Unmarshal([]byte(postExtension), &widened); err != nil {
		t.Fatalf("decode post-extension fixture: %v", err)
	}
	wantDeadline := time.Date(2026, 8, 29, 12, 0, 4, 0, time.UTC)
	if widened.RestartNotBefore == nil || !widened.RestartNotBefore.Equal(wantDeadline) || !widened.HalfOpen {
		t.Fatalf("post-extension facts=%#v want deadline=%s half_open=true", widened, wantDeadline)
	}
	var widenedFields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(postExtension), &widenedFields); err != nil {
		t.Fatal(err)
	}
	if _, present := widenedFields["restart_not_before"]; !present {
		t.Fatalf("post-extension fixture omitted restart_not_before: %s", postExtension)
	}

	for _, fixture := range []string{
		`{"restart_not_before":"not-a-timestamp"}`,
		`{"quarantined_until":"not-a-timestamp"}`,
		`{"last_readiness_match":"not-a-timestamp"}`,
	} {
		var status SharedRuntimeStatus
		if err := json.Unmarshal([]byte(fixture), &status); err == nil {
			t.Fatalf("malformed timestamp fixture decoded: %s", fixture)
		}
	}
}

// Production call site: SharedRuntimeStatusReport. It copies the persisted
// deadline and half-open fact; a historical restart count never mints backoff.
func TestSharedRuntimeStatusReportPublishesPersistedDeadlineWithoutInference(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	readyAt := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	ledger := newSharedRuntimeRestartLedger(resolved.RuntimeKey, resolved.ProfileDigest)
	ledger.RestartCount = 2
	ledger.LastReadinessMatch = &readyAt
	ledger.HalfOpen = true
	if err := writeSharedRuntimeRestartLedger(resolved.Paths.RestartLedger, ledger); err != nil {
		t.Fatal(err)
	}
	options := SharedRuntimeOperatorOptions{
		ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile",
	}
	report, err := SharedRuntimeStatusReport(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.RestartCount != 2 || report.RestartNotBefore != nil || !report.HalfOpen {
		t.Fatalf("historical restart count was inferred as backoff: %#v", report)
	}

	deadline := time.Date(2026, 8, 29, 12, 0, 4, 0, time.UTC)
	ledger.RestartNotBefore = &deadline
	ledger.HalfOpen = false
	if err := writeSharedRuntimeRestartLedger(resolved.Paths.RestartLedger, ledger); err != nil {
		t.Fatal(err)
	}
	report, err = SharedRuntimeStatusReport(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.RestartNotBefore == nil || !report.RestartNotBefore.Equal(deadline) || report.HalfOpen {
		t.Fatalf("persisted restart facts changed in status: %#v", report)
	}
}

// Production call site: SharedRuntimeStatusReport. A malformed ledger is a
// failed read, not an absent ledger with zero-valued lifecycle facts.
func TestSharedRuntimeStatusRefusesMalformedRestartLedgerTimestamps(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	valid := map[string]any{
		"schema": sharedRuntimeRestartLedgerSchema, "runtime_key": resolved.RuntimeKey,
		"profile_digest": resolved.ProfileDigest, "restart_count": 2,
		"restart_not_before": nil, "quarantined_until": nil,
		"last_readiness_match": nil, "manual_quarantine": false, "half_open": false,
	}
	fixtures := []struct {
		name string
		data []byte
	}{{name: "truncated", data: []byte(`{"schema":`)}}
	for _, field := range []string{"restart_not_before", "quarantined_until", "last_readiness_match"} {
		record := make(map[string]any, len(valid))
		for key, value := range valid {
			record[key] = value
		}
		record[field] = "not-a-timestamp"
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		fixtures = append(fixtures, struct {
			name string
			data []byte
		}{name: field, data: data})
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			if err := os.WriteFile(resolved.Paths.RestartLedger, fixture.data, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{
				ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile",
			})
			if sharedRuntimeErrorCode(err) != "shared_runtime_state_unreadable" {
				t.Fatalf("malformed ledger was treated as absence: %v", err)
			}
		})
	}
}

// Production call site: sharedBrokerServer.serve. Runtime death is the event
// that persists the cross-broker failure count and quarantine decision.
func TestSharedBrokerServePersistsStableResetAndFailedHalfOpen(t *testing.T) {
	for _, tc := range []struct {
		name           string
		readyAgo       time.Duration
		halfOpen       bool
		restartCount   int
		wantCount      int
		wantCode       string
		wantQuarantine bool
	}{
		{name: "stable run resets before next failure", readyAgo: 2 * time.Second, restartCount: 3, wantCount: 1, wantCode: "runtime_exited_early"},
		{name: "failed half-open re-quarantines", halfOpen: true, restartCount: 3, wantCount: 4, wantCode: "shared_runtime_quarantined", wantQuarantine: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newSharedBrokerAdmissionFixture(t)
			fixture.server.resolved.Sharing.RestartLimit = 4
			fixture.server.resolved.Sharing.RestartInitialBackoffSeconds = 1
			fixture.server.resolved.Sharing.RestartMaxBackoffSeconds = 2
			fixture.server.resolved.Sharing.StableRunSeconds = 1
			fixture.server.resolved.Sharing.QuarantineSeconds = 30
			readyAt := time.Now().UTC().Add(-tc.readyAgo)
			fixture.server.readyAt = readyAt
			fixture.server.record.ReadyAt = &readyAt
			fixture.server.ledger.RestartCount = tc.restartCount
			fixture.server.ledger.HalfOpen = tc.halfOpen
			if err := writeSharedRuntimeRestartLedger(fixture.server.resolved.Paths.RestartLedger, *fixture.server.ledger); err != nil {
				t.Fatal(err)
			}
			root := shortSharedRuntimeCache(t)
			listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: root + "/serve.sock", Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			fixture.server.listener = listener
			runtimeWait := &piProcessWait{done: make(chan struct{}), err: errors.New("fixture runtime died")}
			close(runtimeWait.done)
			err = fixture.server.serve(runtimeWait, make(chan os.Signal))
			_ = listener.Close()
			if sharedRuntimeErrorCode(err) != tc.wantCode {
				t.Fatalf("serve error=%v want code=%s", err, tc.wantCode)
			}
			ledger, err := readSharedRuntimeRestartLedger(fixture.server.resolved.Paths.RestartLedger, fixture.server.resolved.RuntimeKey, fixture.server.resolved.ProfileDigest)
			if err != nil {
				t.Fatal(err)
			}
			if ledger.RestartCount != tc.wantCount || (ledger.QuarantinedUntil != nil) != tc.wantQuarantine {
				t.Fatalf("persisted serve ledger=%#v", ledger)
			}
		})
	}
}

// Production call site: sharedRuntimeRecordFailure, invoked by the broker on
// every failed restart attempt. Without appendSharedRuntimeFailureEvent's
// trim, this test goes red: FailureHistory grows to exactly `total` entries
// instead of staying at the bound.
func TestSharedRuntimeFailureHistoryIsBoundedAndEvictsOldestFirst(t *testing.T) {
	policy := testSharedRuntimeSupervisionPolicy()
	policy.RestartLimit = 1000 // stay in the backoff branch; quarantine is covered separately
	ledger := newSharedRuntimeRestartLedger("runtime", "profile")
	now := time.Unix(1000, 0).UTC()
	const total = sharedRuntimeFailureHistoryLimit + 5
	for i := 0; i < total; i++ {
		sharedRuntimeRecordFailure(&ledger, policy, now.Add(time.Duration(i)*time.Second))
	}
	if got := len(ledger.FailureHistory); got != sharedRuntimeFailureHistoryLimit {
		t.Fatalf("failure history len=%d want=%d (bound not enforced): %#v", got, sharedRuntimeFailureHistoryLimit, ledger.FailureHistory)
	}
	wantFirstRetained := total - sharedRuntimeFailureHistoryLimit + 1
	for index, event := range ledger.FailureHistory {
		if want := wantFirstRetained + index; event.RestartCount != want {
			t.Fatalf("failure history[%d].restart_count=%d want=%d, eviction order broken (oldest must be evicted first): %#v", index, event.RestartCount, want, ledger.FailureHistory)
		}
	}
	if last := ledger.FailureHistory[len(ledger.FailureHistory)-1].RestartCount; last != total {
		t.Fatalf("newest failure history entry restart_count=%d want=%d", last, total)
	}
}

// Production call site: SharedRuntimeStatusReport. The published status must
// carry the same bounded evidence as the ledger, not a wider or narrower view.
func TestSharedRuntimeStatusReportSurfacesBoundedFailureHistory(t *testing.T) {
	project, home, cache, resolved := newSharedIntegrationProfile(t)
	policy := testSharedRuntimeSupervisionPolicy()
	policy.RestartLimit = 1000
	ledger := newSharedRuntimeRestartLedger(resolved.RuntimeKey, resolved.ProfileDigest)
	now := time.Unix(1000, 0).UTC()
	const total = sharedRuntimeFailureHistoryLimit + 7
	for i := 0; i < total; i++ {
		sharedRuntimeRecordFailure(&ledger, policy, now.Add(time.Duration(i)*time.Second))
	}
	if err := writeSharedRuntimeRestartLedger(resolved.Paths.RestartLedger, ledger); err != nil {
		t.Fatal(err)
	}
	report, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{
		ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: "profile",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(report.FailureHistory); got != sharedRuntimeFailureHistoryLimit {
		t.Fatalf("status failure_history len=%d want=%d", got, sharedRuntimeFailureHistoryLimit)
	}
	if got := report.FailureHistory[len(report.FailureHistory)-1].RestartCount; got != total {
		t.Fatalf("status failure_history newest restart_count=%d want=%d", got, total)
	}
}

func TestSharedRuntimeStatusJSONCarriesContractVersion(t *testing.T) {
	report := SharedRuntimeStatus{ContractVersion: SharedRuntimeStatusContractVersion}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`"contract_version":%d`, SharedRuntimeStatusContractVersion)
	if !bytes.Contains(data, []byte(want)) {
		t.Fatalf("status JSON missing %s: %s", want, data)
	}
}

// Production call site: DecodeSharedRuntimeStatus, the sanctioned entry point
// wired into "runtime status --json" and any future out-of-repo consumer.
// Without the version gate, this test goes red: the payload with
// contract_version 2 decodes cleanly instead of being refused.
func TestDecodeSharedRuntimeStatusRefusesUnsupportedContractVersion(t *testing.T) {
	base := SharedRuntimeStatus{
		ContractVersion: SharedRuntimeStatusContractVersion,
		RuntimeKey:      "runtime", ProfileDigest: "digest",
		Leases: []SharedLeaseStatus{}, Attestation: []SharedRuntimeGateOutcome{},
		FailureHistory: []SharedRuntimeFailureEvent{},
	}
	validData, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSharedRuntimeStatus(validData)
	if err != nil {
		t.Fatalf("supported contract version refused: %v", err)
	}
	if decoded.RuntimeKey != "runtime" {
		t.Fatalf("supported contract version decode lost fields: %#v", decoded)
	}

	for _, future := range []int{SharedRuntimeStatusContractVersion + 1, SharedRuntimeStatusContractVersion + 50} {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(validData, &fields); err != nil {
			t.Fatal(err)
		}
		versionData, err := json.Marshal(future)
		if err != nil {
			t.Fatal(err)
		}
		fields["contract_version"] = versionData
		fields["runtime_key"], err = json.Marshal(fmt.Sprintf("future-runtime-%d", future))
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		status, err := DecodeSharedRuntimeStatus(data)
		var shared *SharedRuntimeError
		if !errors.As(err, &shared) || shared.Code != "shared_runtime_status_unsupported_contract_version" {
			t.Fatalf("future contract version %d admitted: status=%#v err=%v", future, status, err)
		}
		if status.RuntimeKey != "" || status.ContractVersion != 0 {
			t.Fatalf("refused contract version %d still parsed recognised fields: %#v", future, status)
		}
		if got := shared.Details["observed_contract_version"]; got != future {
			t.Fatalf("refusal for version %d did not name the observed version: %#v", future, shared.Details)
		}
		if got := shared.Details["supported_contract_version_min"]; got != SharedRuntimeStatusMinSupportedContractVersion {
			t.Fatalf("refusal did not name supported min: %#v", shared.Details)
		}
		if got := shared.Details["supported_contract_version_max"]; got != SharedRuntimeStatusContractVersion {
			t.Fatalf("refusal did not name supported max: %#v", shared.Details)
		}
	}

	for _, malformed := range []int{0, -1} {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(validData, &fields); err != nil {
			t.Fatal(err)
		}
		versionData, err := json.Marshal(malformed)
		if err != nil {
			t.Fatal(err)
		}
		fields["contract_version"] = versionData
		data, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeSharedRuntimeStatus(data); sharedRuntimeErrorCode(err) != "shared_runtime_status_unsupported_contract_version" {
			t.Fatalf("out-of-range contract version %d admitted: %v", malformed, err)
		}
	}
}

func TestSharedRuntimeStatusWireFixturesCarryFailureHistoryAdditively(t *testing.T) {
	const preExtension = `{"contract_version":1,"restart_count":2,"quarantined_until":null,"last_readiness_match":"2026-08-29T12:00:00Z","manual_quarantine":false}`
	var legacy SharedRuntimeStatus
	if err := json.Unmarshal([]byte(preExtension), &legacy); err != nil {
		t.Fatalf("decode pre-extension fixture: %v", err)
	}
	if legacy.FailureHistory != nil {
		t.Fatalf("pre-extension fixture fabricated failure history: %#v", legacy.FailureHistory)
	}

	const postExtension = `{"contract_version":1,"restart_count":2,"quarantined_until":null,"last_readiness_match":"2026-08-29T12:00:00Z","manual_quarantine":false,"failure_history":[{"occurred_at":"2026-08-29T11:59:00Z","restart_count":1,"backoff_seconds":2,"quarantined":false},{"occurred_at":"2026-08-29T12:00:00Z","restart_count":2,"quarantined":true,"quarantined_until":"2026-08-29T12:05:00Z"}]}`
	var widened SharedRuntimeStatus
	if err := json.Unmarshal([]byte(postExtension), &widened); err != nil {
		t.Fatalf("decode post-extension fixture: %v", err)
	}
	if len(widened.FailureHistory) != 2 || widened.FailureHistory[0].BackoffSeconds != 2 || !widened.FailureHistory[1].Quarantined {
		t.Fatalf("post-extension failure history not decoded: %#v", widened.FailureHistory)
	}
	if widened.FailureHistory[1].QuarantinedUntil == nil {
		t.Fatalf("post-extension quarantine deadline dropped: %#v", widened.FailureHistory[1])
	}
}
