package infra

import (
	"errors"
	"testing"
)

// Production call site: RunPi populates each lifecycle state and defers
// finishPiRunReport; RunModelCheck then passes the report to
// evaluateModelCheckOutcome before publishing its cleanup attestation.
func TestModelCheckCleanupAttestationRefusesUnconfirmedStates(t *testing.T) {
	tests := []struct {
		name       string
		piState    string
		runtime    string
		confirmed  bool
		runErr     error
		wantStatus string
		wantExit   int
	}{
		{name: "both confirmed", piState: "confirmed", runtime: "confirmed", confirmed: true, wantStatus: "passed", wantExit: 0},
		{name: "sigkill still proves absence", piState: "confirmed_after_sigkill", runtime: "confirmed_after_sigkill", confirmed: true, wantStatus: "passed", wantExit: 0},
		{name: "nothing started needs no cleanup but execution fails", piState: "not_started", runtime: "not_started", confirmed: true, runErr: errors.New("runtime did not start"), wantStatus: "failed", wantExit: ModelCheckExitExecutionFailed},
		{name: "pending pi cleanup refuses", piState: "pending", runtime: "confirmed", wantStatus: "failed", wantExit: ModelCheckExitExecutionFailed},
		{name: "failed runtime cleanup refuses", piState: "confirmed", runtime: "failed", wantStatus: "failed", wantExit: ModelCheckExitExecutionFailed},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			report := PiRunReport{
				Managed:                    true,
				PiProcessGroupCleanup:      testCase.piState,
				RuntimeProcessGroupCleanup: testCase.runtime,
			}
			finishPiRunReport(&report)
			if report.CleanupConfirmed != testCase.confirmed {
				t.Fatalf("cleanup_confirmed=%t want=%t for pi=%q runtime=%q", report.CleanupConfirmed, testCase.confirmed, testCase.piState, testCase.runtime)
			}

			summary := ModelCheckSummary{ManagedRuntime: report}
			parsed := parsedModelCheckEvents{valid: true, complete: true}
			status, exitCode, reasons := evaluateModelCheckOutcome(summary, parsed, testCase.runErr)
			if status != testCase.wantStatus || exitCode != testCase.wantExit {
				t.Fatalf("outcome=%q/%d want=%q/%d reasons=%v", status, exitCode, testCase.wantStatus, testCase.wantExit, reasons)
			}
			if !testCase.confirmed && len(reasons) != 1 {
				t.Fatalf("unconfirmed cleanup reasons = %v", reasons)
			}
		})
	}
}
