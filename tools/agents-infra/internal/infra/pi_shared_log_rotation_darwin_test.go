//go:build darwin

package infra

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Production call site: startUnauthorizedRuntime ->
// startUnauthorizedRuntimeWithDependencies -> openSharedRotatingLog ->
// newSharedRotatingLogWriter -> sharedFilesystemLogSink.Prune.
func TestStartUnauthorizedRuntimeRefusesOversizedRetainedArchiveBeforeCommandStart(t *testing.T) {
	directory := t.TempDir()
	runtimeLog := filepath.Join(directory, "runtime.log")
	archive := runtimeLog + ".00000000000000000001.20260829T120000.000000000Z"
	if err := os.WriteFile(runtimeLog, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(runtimeLog, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(archive, 0o600); err != nil {
		t.Fatal(err)
	}

	resolved := sharedResolvedProfile{
		Project:     directory,
		ProfileName: "profile",
		RuntimeKey:  strings.Repeat("a", 64),
		Sharing: PiRuntimeSharing{
			MaxSegmentBytes: 4,
			MaxSegments:     2,
		},
		Paths: SharedRuntimePaths{
			RuntimeCWD: directory,
			RuntimeLog: runtimeLog,
		},
	}
	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC)}
	startCalls := 0
	command, wait, authorizationWriter, err := startUnauthorizedRuntimeWithDependencies(resolved, nil, clock, func(*exec.Cmd) error {
		startCalls++
		return errors.New("command start must not be reached")
	})
	if err == nil {
		t.Fatal("startUnauthorizedRuntime admitted retained archives above max_segment_bytes * max_segments")
	}
	if startCalls != 0 {
		t.Fatalf("runtime command start side effect calls=%d want=0", startCalls)
	}
	if command != nil || wait != nil || authorizationWriter != nil {
		t.Fatalf("refused start returned command=%v wait=%v authorization_writer=%v", command, wait, authorizationWriter)
	}
	var runtimeErr *SharedRuntimeError
	if !errors.As(err, &runtimeErr) || runtimeErr.Code != "shared_runtime_state_path_invalid" {
		t.Fatalf("error=%v want shared_runtime_state_path_invalid", err)
	}
	if !strings.Contains(err.Error(), "managed log archive exceeds max_segment_bytes") {
		t.Fatalf("error=%v want oversized archive refusal", err)
	}
	for path, want := range map[string]int64{runtimeLog: 1, archive: 10} {
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("preserved %s: %v", filepath.Base(path), statErr)
		}
		if info.Size() != want {
			t.Fatalf("preserved %s size=%d want=%d", filepath.Base(path), info.Size(), want)
		}
	}
}

func TestOpenSharedRotatingLogRefusesManagedArchiveSymlinkWithoutTouchingForeignTarget(t *testing.T) {
	directory := t.TempDir()
	runtimeLog := filepath.Join(directory, "runtime.log")
	foreign := filepath.Join(directory, "foreign.log")
	archive := runtimeLog + ".00000000000000000001.20260829T120000.000000000Z"
	if err := os.WriteFile(runtimeLog, []byte("active"), 0o600); err != nil {
		t.Fatal(err)
	}
	wantForeign := []byte("foreign-output")
	if err := os.WriteFile(foreign, wantForeign, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, archive); err != nil {
		t.Fatal(err)
	}

	clock := &fakeSharedLogClock{now: time.Date(2026, 8, 29, 14, 0, 0, 0, time.UTC)}
	if _, err := openSharedRotatingLog(runtimeLog, 64, 2, clock); err == nil {
		t.Fatal("openSharedRotatingLog admitted a managed archive symlink")
	}
	gotForeign, err := os.ReadFile(foreign)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotForeign) != string(wantForeign) {
		t.Fatalf("foreign target changed: got=%q want=%q", gotForeign, wantForeign)
	}
	info, err := os.Lstat(archive)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("managed archive symlink was replaced: mode=%v", info.Mode())
	}
}
