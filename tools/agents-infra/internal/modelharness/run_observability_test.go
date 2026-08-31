package modelharness

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestSupervisionMarkerGateRejectsNearMissOutput bounds the condemn decision in
// `newFatalOutputWriter`, reached in production from Run -> runSupervised ->
// runSupervisedAttempt. The positive tests in run_test.go only prove the gate is
// reachable; this one proves the class it condemns is the configured literal and
// not a looser substring of it.
//
// Two near misses are used so the bound holds at both ends of the match. Each
// shares one end of the configured marker and differs at the other, so a mutant
// that truncates the match from either side admits one of them and fails here.
// One near miss alone was not enough: with only the "exceeded" case, a
// head-truncating mutant (marker[14:]) survived.
func TestSupervisionMarkerGateRejectsNearMissOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	const marker = "RuntimeError: [metal::malloc] Resource limit ("
	cases := []struct {
		name string
		// line shares one end of marker verbatim and diverges at the other.
		line string
		// echo is a fragment of line that must survive forwarding.
		echo string
	}{
		{
			// Shares the whole head; diverges only at the marker's trailing open
			// paren. A mutant that truncates the match from the tail admits it.
			name: "tail differs",
			line: "RuntimeError: [metal::malloc] Resource limit exceeded",
			echo: "Resource limit exceeded",
		},
		{
			// Shares the whole tail including the open paren; diverges only in the
			// leading exception name. A mutant that truncates the match from the
			// head admits it.
			name: "head differs",
			line: "FatalError: [metal::malloc] Resource limit (499000)",
			echo: "FatalError:",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if strings.Contains(testCase.line, marker) {
				t.Fatalf("near miss %q contains the full marker; it is not a near miss", testCase.line)
			}
			plan := Plan{
				Profile:    "near-miss",
				Mode:       "local",
				Executable: "/bin/sh",
				Argv:       []string{"-c", `printf '%s\n' ` + shellQuote(testCase.line) + ` >&2; exit 0`},
				Supervision: &SupervisionPolicy{
					FatalOutputSubstrings:    []string{marker},
					RestartOnFailure:         false,
					MaxRestarts:              3,
					RestartWindowSeconds:     60,
					RestartDelayMilliseconds: 1,
				},
			}
			var stdout, stderr bytes.Buffer
			if err := run(plan, &stdout, &stderr, func(time.Duration) {}); err != nil {
				t.Fatalf("run: %v", err)
			}
			if strings.Contains(stderr.String(), "restarting profile") {
				t.Fatalf("near-miss output condemned the worker; stderr=%q", stderr.String())
			}
			if !strings.Contains(stderr.String(), testCase.echo) {
				t.Fatalf("child stderr was not forwarded; stderr=%q", stderr.String())
			}
		})
	}
}

// TestSupervisionMarkerGateMatchesAcrossWriteBoundary keeps the same gate from
// being narrowed by chunking: a runtime that flushes the marker in two writes is
// exactly as condemned as one that flushes it in a single write. Removing the
// carry in fatalOutputWriter makes this fail while the run_test.go cases still
// pass.
func TestSupervisionMarkerGateMatchesAcrossWriteBoundary(t *testing.T) {
	notify := make(chan string, 1)
	marker := "RuntimeError: [metal::malloc] Resource limit ("
	var sink bytes.Buffer
	writer := newFatalOutputWriter(&sink, []string{marker}, notify)
	head, tail := marker[:20], marker[20:]
	if _, err := writer.Write([]byte("noise " + head)); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-notify:
		t.Fatalf("partial marker condemned the worker: %q", got)
	default:
	}
	if _, err := writer.Write([]byte(tail + " 499000)\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-notify:
		if got != marker {
			t.Fatalf("marker=%q want %q", got, marker)
		}
	default:
		t.Fatal("split marker was not condemned")
	}
	if sink.String() != "noise "+marker+" 499000)\n" {
		t.Fatalf("forwarded bytes=%q", sink.String())
	}
}

// TestRunForwardsChildBytesVerbatimAndAddsNoRecords is the characterisation
// behind TASK-260828-28gdmq: `model-harness run` is a pipe, not a recorder. It
// forwards exactly what the child wrote and authors nothing of its own on a
// clean run, so nothing a child keeps off stdout/stderr can be recovered from
// harness-captured output. That fact is channel-independent and is what this
// test establishes. Of the candidate runtimes, only two were audited —
// mlx_lm.server and the Swift prototype — and both keep prompt and completion
// bodies on the HTTP socket; llama-server was never exercised and its behaviour
// is reported as unknown (blocker B8).
func TestRunForwardsChildBytesVerbatimAndAddsNoRecords(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	const outLine = "runtime: listening on 127.0.0.1:18081\n"
	const errLine = "127.0.0.1 - - [28/Aug/2026 13:19:46] \"POST /v1/chat/completions HTTP/1.1\" 200 -\n"
	plan := Plan{
		Profile:    "verbatim",
		Mode:       "local",
		Executable: "/bin/sh",
		Argv:       []string{"-c", `printf '` + outLine + `'; printf '` + errLine + `' >&2`},
	}
	var stdout, stderr bytes.Buffer
	if err := run(plan, &stdout, &stderr, func(time.Duration) {}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if stdout.String() != outLine {
		t.Fatalf("stdout=%q want %q", stdout.String(), outLine)
	}
	if stderr.String() != errLine {
		t.Fatalf("stderr=%q want %q", stderr.String(), errLine)
	}
}

// TestRunPersistsNothingItself pins the retention finding: `model-harness run`
// opens no capture sink of its own, so where forwarded output lands, how long it
// is kept, and how large it grows are entirely the caller's. It also guards the
// scope discipline of TASK-260828-28gdmq — a silently added log file would make
// the recorded gap untrue, and would fail here.
func TestRunPersistsNothingItself(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	plan := Plan{
		Profile:    "no-sink",
		Mode:       "local",
		Executable: "/bin/sh",
		Argv:       []string{"-c", `printf 'PROMPTNONCE\n'; printf 'COMPLETIONNONCE\n' >&2`},
	}
	var stdout, stderr bytes.Buffer
	if err := run(plan, &stdout, &stderr, func(time.Duration) {}); err != nil {
		t.Fatalf("run: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("run created %v; it must open no capture sink of its own", names)
	}
}
