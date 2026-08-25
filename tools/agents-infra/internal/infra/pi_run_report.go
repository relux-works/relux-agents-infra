package infra

// PiRunReport is optional execution evidence produced by RunPi. It records
// only lifecycle state; process identifiers and command arguments are omitted
// so callers can safely carry the report into sanitized summaries.
type PiRunReport struct {
	Managed                    bool   `json:"managed"`
	DeadlineExceeded           bool   `json:"deadline_exceeded"`
	PiProcessGroupCleanup      string `json:"pi_process_group_cleanup"`
	RuntimeProcessGroupCleanup string `json:"runtime_process_group_cleanup"`
	CleanupConfirmed           bool   `json:"cleanup_confirmed"`
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
