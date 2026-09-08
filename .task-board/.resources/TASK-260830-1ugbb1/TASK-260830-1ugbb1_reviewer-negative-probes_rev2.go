//go:build !windows

package infra

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Production call site: openPiSessionLog -> recoverPiLifecycle ->
// recoverPiLifecycleClose. A close record is exact only when ClosedAt is the
// odd operation's StartedAt; a merely well-formed timestamp is forged evidence.
func TestReviewerForgedCloseLookingEvidenceIsRefused(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer log.close(context.Background())

	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	odd, err := beginPiLifecycleOperation(rootFD, even, "close", log.entry, "", log.record.CommittedBytes, log.record.CommittedRecords, 0)
	if err != nil {
		t.Fatal(err)
	}
	forged := log.record
	forged.ClosedAt = time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC).Format(time.RFC3339Nano)
	if forged.ClosedAt == odd.StartedAt {
		t.Fatal("probe timestamp unexpectedly equals operation timestamp")
	}
	encoded, err := encodePiLifecycleControl(forged)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePiLifecycleControlAtomic(log.entryFD, "record.json", encoded); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)

	next, err := openPiSessionLog(context.Background(), paths, policy)
	if next != nil {
		defer next.close(context.Background())
	}
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("forged close-looking record admitted by production recovery: next=%v err=%v", next != nil, err)
	}
}

// Production call site: openPiSessionLog -> recoverPiLifecycle ->
// recoverPiLifecycleDelete. A narrowed tombstone directory is unproven and
// must survive; child names alone are not deletion authority.
func TestReviewerDeleteRecoveryPreservesNarrowedTombstoneAuthority(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	entry, id := log.entry, log.record.EntryID
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries := filepath.Join(paths.LifecycleLogsRoot, "entries")
	tombstoneName := ".deleting-" + id
	tombstonePath := filepath.Join(entries, tombstoneName)
	if err := os.Rename(filepath.Join(entries, entry), tombstonePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tombstonePath, 0o755); err != nil {
		t.Fatal(err)
	}
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	if _, err := beginPiLifecycleOperation(rootFD, even, "delete", entry, tombstoneName, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)

	next, err := openPiSessionLog(context.Background(), paths, policy)
	if next != nil {
		defer next.close(context.Background())
	}
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("narrowed tombstone authority was deleted and new work admitted: next=%v err=%v", next != nil, err)
	}
	if _, statErr := os.Lstat(tombstonePath); statErr != nil {
		t.Fatalf("unproven tombstone evidence was not preserved: %v", statErr)
	}
}

// Production call site: PiLifecycleStatus -> readPiLifecycleGenerationPair.
// An even generation carrying residual operation fields is malformed, not a
// healthy absence of an in-flight operation.
func TestReviewerEvenGenerationWithResidualOperationAuthorityIsUnknown(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	generation := readPiLifecycleGenerationForTest(t, rootFD)
	generation.StartedAt = "2026-01-01T00:00:00Z"
	generation.CommittedBefore = 1
	generation.RecordsBefore = 1
	generation.AppendBytes = 1
	encoded, err := encodePiLifecycleControl(generation)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePiLifecycleControlAtomic(rootFD, "generation.json", encoded); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)

	status, err := PiLifecycleStatus(context.Background(), paths, policy, "review", "")
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") || status.WithinPolicy || status.SoakReady {
		t.Fatalf("malformed even generation published health: status=%+v err=%v", status, err)
	}
}
