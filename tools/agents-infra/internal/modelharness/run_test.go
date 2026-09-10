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

// TestDoctorPassesPinnedNonEditableInstall drives the production model-harness
// doctor entry point: a profile declaring pinned_distribution must pass when
// the installed backend is a non-editable git checkout at the pinned commit.
func TestDoctorPassesPinnedNonEditableInstall(t *testing.T) {
	sitePackages := t.TempDir()
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{
		"url": "https://github.com/relux-works/mlx-lm.git",
		"vcs_info": {"vcs": "git", "commit_id": "`+testPinnedCommit+`", "requested_revision": "`+testPinnedCommit+`"}
	}`)
	plan := Plan{
		Profile:    "qwen-local",
		Mode:       "local",
		Executable: "/bin/echo",
		PinnedDistribution: &PinnedDistribution{
			Package:      "mlx_lm",
			SitePackages: sitePackages,
			Commit:       testPinnedCommit,
		},
	}
	var stdout, stderr bytes.Buffer
	if err := Doctor(plan, &stdout, &stderr); err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if !strings.Contains(stdout.String(), "status=ok") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

// TestDoctorRefusesEditablePinnedInstall is the negative control named by the
// task: model-harness doctor must fail, through the same production Doctor
// entry point, when the pinned distribution is installed editable.
func TestDoctorRefusesEditablePinnedInstall(t *testing.T) {
	sitePackages := t.TempDir()
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{
		"url": "file:///Users/alexis/src/relux-works/mlx-lm",
		"dir_info": {"editable": true}
	}`)
	plan := Plan{
		Profile:    "qwen-local",
		Mode:       "local",
		Executable: "/bin/echo",
		PinnedDistribution: &PinnedDistribution{
			Package:      "mlx_lm",
			SitePackages: sitePackages,
			Commit:       testPinnedCommit,
		},
	}
	var stdout, stderr bytes.Buffer
	err := Doctor(plan, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "editable") {
		t.Fatalf("error = %v, want editable-install refusal", err)
	}
	if strings.Contains(stdout.String(), "status=ok") {
		t.Fatalf("stdout must not report status=ok on refusal: %q", stdout.String())
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
