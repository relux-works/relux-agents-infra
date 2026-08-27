package modelharness

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunSupervisedRestartsAfterFatalOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	counter := filepath.Join(t.TempDir(), "counter")
	plan := supervisedShellPlan(counter, `
count=0
if test -f "$1"; then count=$(cat "$1"); fi
count=$((count + 1))
printf '%s' "$count" > "$1"
if test "$count" -eq 1; then
  printf 'fatal-prefix Resource limit (499000) exceeded fatal-suffix\n' >&2
  kill -STOP $$
fi
exit 0
`)
	var stdout, stderr bytes.Buffer
	if err := run(plan, &stdout, &stderr, func(time.Duration) {}); err != nil {
		t.Fatalf("run: %v", err)
	}
	got, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "2" {
		t.Fatalf("launch count=%q want 2", got)
	}
	if !strings.Contains(stderr.String(), "restarting profile \"test\"") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRunSupervisedStopsAfterRestartBudget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh")
	}
	counter := filepath.Join(t.TempDir(), "counter")
	plan := supervisedShellPlan(counter, `
count=0
if test -f "$1"; then count=$(cat "$1"); fi
count=$((count + 1))
printf '%s' "$count" > "$1"
printf 'Resource limit (499000) exceeded\n' >&2
kill -STOP $$
`)
	plan.Supervision.MaxRestarts = 1
	var stdout, stderr bytes.Buffer
	err := run(plan, &stdout, &stderr, func(time.Duration) {})
	if err == nil || !strings.Contains(err.Error(), "restart budget exhausted") {
		t.Fatalf("error=%v", err)
	}
	got, readErr := os.ReadFile(counter)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "2" {
		t.Fatalf("launch count=%q want 2", got)
	}
}

func supervisedShellPlan(counter, script string) Plan {
	return Plan{
		Profile:    "test",
		Mode:       "local",
		Executable: "/bin/sh",
		Argv:       []string{"-c", script, "model-harness-test", counter},
		Supervision: &SupervisionPolicy{
			FatalOutputSubstrings:    []string{"Resource limit (499000) exceeded"},
			MaxRestarts:              3,
			RestartWindowSeconds:     60,
			RestartDelayMilliseconds: 1,
		},
	}
}
