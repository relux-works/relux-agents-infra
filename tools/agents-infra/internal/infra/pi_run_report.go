package infra

// PiRunReport is optional execution evidence produced by RunPi. It records
// only lifecycle state; process identifiers and command arguments are omitted
// so callers can safely carry the report into sanitized summaries.
type PiRunReport struct {
	Managed                      bool     `json:"managed"`
	DeadlineExceeded             bool     `json:"deadline_exceeded"`
	PiProcessGroupCleanup        string   `json:"pi_process_group_cleanup"`
	RuntimeProcessGroupCleanup   string   `json:"runtime_process_group_cleanup"`
	CleanupConfirmed             bool     `json:"cleanup_confirmed"`
	SessionLog                   string   `json:"session_log,omitempty"`
	LifecyclePolicySource        string   `json:"lifecycle_policy_source,omitempty"`
	LifecycleAggregateRoot       string   `json:"lifecycle_aggregate_root,omitempty"`
	LifecycleAggregateGeneration uint64   `json:"lifecycle_aggregate_generation"`
	LifecycleLegacyGeneration    uint64   `json:"lifecycle_legacy_generation"`
	LifecycleScanComplete        bool     `json:"lifecycle_scan_complete"`
	LifecycleScanExhausted       bool     `json:"lifecycle_scan_exhausted"`
	LifecycleContinuation        string   `json:"lifecycle_continuation,omitempty"`
	LifecyclePageScope           string   `json:"lifecycle_page_scope,omitempty"`
	LifecycleLowerBound          bool     `json:"lifecycle_lower_bound"`
	LifecycleScanEntries         int      `json:"lifecycle_scan_entries"`
	LifecycleScanControlBytes    int      `json:"lifecycle_scan_control_bytes"`
	LifecycleWithinPolicy        bool     `json:"lifecycle_within_policy"`
	LifecycleSoakReady           bool     `json:"lifecycle_soak_ready"`
	LifecycleManagedCount        int      `json:"lifecycle_managed_count"`
	LifecycleManagedBytes        int64    `json:"lifecycle_managed_bytes"`
	LifecycleEnvelopeBytes       int64    `json:"lifecycle_envelope_bytes"`
	LifecycleActiveCount         int      `json:"lifecycle_active_count"`
	LifecycleExpiredCount        int      `json:"lifecycle_expired_count"`
	LifecycleLegacyCount         int      `json:"lifecycle_legacy_count"`
	LifecycleLegacyBytes         int64    `json:"lifecycle_legacy_bytes"`
	LifecycleForeignCount        int      `json:"lifecycle_foreign_count"`
	LifecycleForeignBytes        int64    `json:"lifecycle_foreign_bytes"`
	LifecycleUnknownCount        int      `json:"lifecycle_unknown_count"`
	LifecycleOldest              string   `json:"lifecycle_oldest,omitempty"`
	LifecycleNewest              string   `json:"lifecycle_newest,omitempty"`
	LifecycleRecoveryCount       int      `json:"lifecycle_recovery_count"`
	LifecyclePrunedCount         int      `json:"lifecycle_pruned_count"`
	LifecycleDroppedCount        int      `json:"lifecycle_dropped_count"`
	LifecycleErrors              []string `json:"lifecycle_errors,omitempty"`
}

func recordPiLifecycleStatus(report *PiRunReport, status PiLifecycleLogStatus) {
	if report == nil {
		return
	}
	report.LifecyclePolicySource = status.PolicySource
	report.LifecycleAggregateRoot = status.AggregateRoot
	report.LifecycleAggregateGeneration = status.AggregateGeneration
	report.LifecycleLegacyGeneration = status.LegacyGeneration
	report.LifecycleScanComplete = status.ScanComplete
	report.LifecycleScanExhausted = status.ScanExhausted
	report.LifecycleContinuation = status.Continuation
	report.LifecyclePageScope = status.PageScope
	report.LifecycleLowerBound = status.LowerBound
	report.LifecycleScanEntries = status.ScanEntries
	report.LifecycleScanControlBytes = status.ScanControlBytes
	report.LifecycleWithinPolicy = status.WithinPolicy
	report.LifecycleSoakReady = status.SoakReady
	report.LifecycleManagedCount = status.ManagedCount
	report.LifecycleManagedBytes = status.ManagedCommittedBytes
	report.LifecycleEnvelopeBytes = status.ManagedEnvelopeBytes
	report.LifecycleActiveCount = status.ActiveCount
	report.LifecycleExpiredCount = status.ExpiredCount
	report.LifecycleLegacyCount = status.LegacyCount
	report.LifecycleLegacyBytes = status.LegacyBytes
	report.LifecycleForeignCount = status.ForeignCount
	report.LifecycleForeignBytes = status.ForeignBytes
	report.LifecycleUnknownCount = status.UnknownCount
	report.LifecycleOldest = status.Oldest
	report.LifecycleNewest = status.Newest
	report.LifecycleRecoveryCount = status.RecoveryCount
	report.LifecyclePrunedCount = status.PrunedCount
	report.LifecycleDroppedCount = status.DroppedCount
	report.LifecycleErrors = append([]string(nil), status.Errors...)
}

func newPiRunReport() PiRunReport {
	return PiRunReport{
		PiProcessGroupCleanup:      "not_started",
		RuntimeProcessGroupCleanup: "not_started",
	}
}

func finishPiRunReport(report *PiRunReport) {
	if report == nil {
		return
	}
	if !report.Managed {
		report.PiProcessGroupCleanup = "not_managed"
		report.RuntimeProcessGroupCleanup = "not_managed"
		report.CleanupConfirmed = true
		return
	}
	report.CleanupConfirmed = isConfirmedCleanupState(report.PiProcessGroupCleanup) &&
		isConfirmedCleanupState(report.RuntimeProcessGroupCleanup)
}

func isConfirmedCleanupState(state string) bool {
	return state == "confirmed" || state == "confirmed_after_sigkill" || state == "not_started"
}
