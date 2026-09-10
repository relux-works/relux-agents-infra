//go:build !windows

package infra

import (
	"bytes"
	"fmt"
	"testing"
)

type piKVBoundAlertReport struct {
	requestID      string
	maxKVSize      int
	observedTokens int
}

func newPiKVBoundAlertRecorder() (*[]piKVBoundAlertReport, func(string, int, int)) {
	reports := new([]piKVBoundAlertReport)
	return reports, func(requestID string, maxKVSize, observedTokens int) {
		*reports = append(*reports, piKVBoundAlertReport{requestID, maxKVSize, observedTokens})
	}
}

// Production call site: RunPi wraps the runtime child's merged
// stdout/stderr writer in newPiKVBoundAlertWriter (pi_launch_posix.go),
// reporting through sessionLog.event so an operator sees a truncated-context
// request in the Pi session log without reading raw server output.
func TestPiKVBoundAlertWriterReportsMatchAndForwardsVerbatim(t *testing.T) {
	var target bytes.Buffer
	reports, report := newPiKVBoundAlertRecorder()
	writer := newPiKVBoundAlertWriter(&target, report)

	line := "2026-09-11 19:30:00,123 - WARNING - kv_cache_bound_exceeded request_id=chatcmpl-abc123 max_kv_size=76800 observed_tokens=76910\n"
	other := "127.0.0.1 - - [11/Sep/2026 19:30:00] \"POST /v1/completions HTTP/1.1\" 200 -\n"
	if _, err := writer.Write([]byte(other + line)); err != nil {
		t.Fatal(err)
	}

	if target.String() != other+line {
		t.Fatalf("forwarded bytes=%q, want verbatim input", target.String())
	}
	if len(*reports) != 1 {
		t.Fatalf("reports=%v, want exactly one", *reports)
	}
	got := (*reports)[0]
	want := piKVBoundAlertReport{requestID: "chatcmpl-abc123", maxKVSize: 76800, observedTokens: 76910}
	if got != want {
		t.Fatalf("report=%+v, want %+v", got, want)
	}
}

// Removing the carry buffer (or resetting it fully on every Write) makes
// this fail while the single-write case above still passes, so it narrows
// specifically the across-boundary accumulation, not mere reachability.
func TestPiKVBoundAlertWriterMatchesAcrossWriteBoundary(t *testing.T) {
	var target bytes.Buffer
	reports, report := newPiKVBoundAlertRecorder()
	writer := newPiKVBoundAlertWriter(&target, report)

	full := "kv_cache_bound_exceeded request_id=chatcmpl-split max_kv_size=8 observed_tokens=41\n"
	head, tail := full[:40], full[40:]

	if _, err := writer.Write([]byte(head)); err != nil {
		t.Fatal(err)
	}
	if len(*reports) != 0 {
		t.Fatalf("partial line already reported: %v", *reports)
	}
	if _, err := writer.Write([]byte(tail)); err != nil {
		t.Fatal(err)
	}
	if len(*reports) != 1 {
		t.Fatalf("reports=%v, want exactly one after the line completes", *reports)
	}
	want := piKVBoundAlertReport{requestID: "chatcmpl-split", maxKVSize: 8, observedTokens: 41}
	if (*reports)[0] != want {
		t.Fatalf("report=%+v, want %+v", (*reports)[0], want)
	}
	if target.String() != full {
		t.Fatalf("forwarded bytes=%q, want verbatim %q", target.String(), full)
	}
}

// Negative control, narrowing the gate: a line that merely contains the
// substring "kv_cache_bound_exceeded" without the structured
// request_id/max_kv_size/observed_tokens fields must not report. A mutant
// that loosens the match to a bare substring.Contains check admits this
// line and fails here while the positive tests above still pass.
func TestPiKVBoundAlertWriterIgnoresNearMissLine(t *testing.T) {
	cases := []string{
		"kv_cache_bound_exceeded for request chatcmpl-abc123\n",
		"kv_cache_bound_exceeded request_id=chatcmpl-abc max_kv_size=notanumber observed_tokens=10\n",
		"127.0.0.1 - - [11/Sep/2026 19:30:00] \"POST /v1/completions HTTP/1.1\" 200 -\n",
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			var target bytes.Buffer
			reports, report := newPiKVBoundAlertRecorder()
			writer := newPiKVBoundAlertWriter(&target, report)

			if _, err := writer.Write([]byte(line)); err != nil {
				t.Fatal(err)
			}
			if len(*reports) != 0 {
				t.Fatalf("near-miss line reported: %v", *reports)
			}
			if target.String() != line {
				t.Fatalf("forwarded bytes=%q, want verbatim %q", target.String(), line)
			}
		})
	}
}

// A request that fits within the bound produces no server-side warning at
// all, so the writer must not fire on ordinary output volume either.
func TestPiKVBoundAlertWriterReportsNothingForOrdinaryOutput(t *testing.T) {
	var target bytes.Buffer
	reports, report := newPiKVBoundAlertRecorder()
	writer := newPiKVBoundAlertWriter(&target, report)

	for i := 0; i < 50; i++ {
		line := fmt.Sprintf("127.0.0.1 - - [11/Sep/2026 19:30:%02d] \"POST /v1/completions HTTP/1.1\" 200 -\n", i)
		if _, err := writer.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	if len(*reports) != 0 {
		t.Fatalf("reports=%v, want none for ordinary output", *reports)
	}
}

func TestPiKVBoundAlertWriterReportsEachMatchingLineOnce(t *testing.T) {
	var target bytes.Buffer
	reports, report := newPiKVBoundAlertRecorder()
	writer := newPiKVBoundAlertWriter(&target, report)

	lines := "kv_cache_bound_exceeded request_id=chatcmpl-1 max_kv_size=8 observed_tokens=9\n" +
		"kv_cache_bound_exceeded request_id=chatcmpl-2 max_kv_size=8 observed_tokens=12\n"
	if _, err := writer.Write([]byte(lines)); err != nil {
		t.Fatal(err)
	}
	if len(*reports) != 2 {
		t.Fatalf("reports=%v, want exactly two", *reports)
	}
	if (*reports)[0].requestID != "chatcmpl-1" || (*reports)[1].requestID != "chatcmpl-2" {
		t.Fatalf("reports=%v, want distinct request ids in order", *reports)
	}
}

func TestNewPiKVBoundAlertWriterReturnsNilForNilTarget(t *testing.T) {
	_, report := newPiKVBoundAlertRecorder()
	if writer := newPiKVBoundAlertWriter(nil, report); writer != nil {
		t.Fatalf("writer=%v, want nil for a nil target", writer)
	}
}
