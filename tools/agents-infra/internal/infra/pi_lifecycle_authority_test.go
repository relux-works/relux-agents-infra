//go:build !windows

package infra

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Production call site: openPiSessionLog -> recoverPiLifecycle ->
// recoverPiLifecycleClose. A close record is exact only when ClosedAt is the
// issued odd operation's StartedAt; a merely well-formed timestamp is forged.
func TestPiLifecycleRecoveryRefusesForgedCloseLookingEvidence(t *testing.T) {
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
// recoverPiLifecycleDelete. The recovery path must retain exact tombstone
// directory authority before it performs any bounded unlink.
func TestPiLifecycleDeleteRecoveryPreservesNarrowedTombstoneAuthority(t *testing.T) {
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
	tombstone := ".deleting-" + id
	tombstonePath := filepath.Join(entries, tombstone)
	if err := os.Rename(filepath.Join(entries, entry), tombstonePath); err != nil {
		t.Fatal(err)
	}
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	if _, err := beginPiLifecycleOperation(rootFD, even, "delete", entry, tombstone, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tombstonePath, 0o755); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)

	next, err := openPiSessionLog(context.Background(), paths, policy)
	if next != nil {
		defer next.close(context.Background())
	}
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("narrowed tombstone authority admitted: next=%v err=%v", next != nil, err)
	}
	if _, statErr := os.Lstat(tombstonePath); statErr != nil {
		t.Fatalf("unproven tombstone evidence was not preserved: %v", statErr)
	}
}

// Production call site: openPiSessionLog -> recoverPiLifecycle ->
// recoverPiLifecycleDelete -> revalidatePiLifecycleDeleteDirectory. A
// mode-correct replacement tombstone is not the directory identity recorded
// by the odd delete generation, even when all three child identities survive.
func TestPiLifecycleDeleteRecoveryPreservesModeCorrectTombstoneSubstitution(t *testing.T) {
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
	tombstone := ".deleting-" + id
	tombstonePath := filepath.Join(entries, tombstone)
	if err := os.Rename(filepath.Join(entries, entry), tombstonePath); err != nil {
		t.Fatal(err)
	}
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	if _, err := beginPiLifecycleOperation(rootFD, even, "delete", entry, tombstone, 0, 0, 0); err != nil {
		unix.Close(rootFD)
		t.Fatal(err)
	}

	preservedPath := filepath.Join(filepath.Dir(paths.LifecycleLogsRoot), "preserved-delete-tombstone")
	if err := os.Rename(tombstonePath, preservedPath); err != nil {
		unix.Close(rootFD)
		t.Fatal(err)
	}
	if err := os.Mkdir(tombstonePath, 0o700); err != nil {
		unix.Close(rootFD)
		t.Fatal(err)
	}
	for _, name := range []string{"record.json", "log.jsonl", "active.lock"} {
		if err := os.Rename(filepath.Join(preservedPath, name), filepath.Join(tombstonePath, name)); err != nil {
			unix.Close(rootFD)
			t.Fatal(err)
		}
	}
	var preserved, replacement unix.Stat_t
	if err := unix.Stat(preservedPath, &preserved); err != nil {
		unix.Close(rootFD)
		t.Fatal(err)
	}
	if err := unix.Stat(tombstonePath, &replacement); err != nil {
		unix.Close(rootFD)
		t.Fatal(err)
	}
	if preserved.Dev == replacement.Dev && preserved.Ino == replacement.Ino {
		unix.Close(rootFD)
		t.Fatal("test did not substitute the tombstone inode")
	}
	if preserved.Mode != replacement.Mode || preserved.Uid != replacement.Uid {
		unix.Close(rootFD)
		t.Fatalf("replacement is not generic-authority equivalent: preserved mode/uid=%#o/%d replacement=%#o/%d", preserved.Mode, preserved.Uid, replacement.Mode, replacement.Uid)
	}
	unix.Close(rootFD)

	next, recoveryErr := openPiSessionLog(context.Background(), paths, policy)
	if next != nil {
		defer next.close(context.Background())
	}
	if !piErrorIs(recoveryErr, "lifecycle_log_evidence_unknown") {
		t.Fatalf("mode-correct substituted tombstone admitted: next=%v err=%v", next != nil, recoveryErr)
	}
	if _, err := os.Lstat(tombstonePath); err != nil {
		t.Fatalf("unproven replacement tombstone was not preserved: %v", err)
	}
	for _, name := range []string{"record.json", "log.jsonl", "active.lock"} {
		if _, err := os.Lstat(filepath.Join(tombstonePath, name)); err != nil {
			t.Fatalf("replacement tombstone child %s was not preserved: %v", name, err)
		}
	}
}

// Production call site: openPiSessionLog -> recoverPiLifecycle ->
// recoverPiLifecycleDelete. Replacing a recognized child after the odd delete
// generation is issued must not turn its name into unlink authority.
func TestPiLifecycleDeleteRecoveryPreservesSubstitutedChildAuthority(t *testing.T) {
	for name, mutate := range map[string]func(*testing.T, string){
		"record-mode-narrowed": func(t *testing.T, path string) {
			if err := os.Chmod(path, 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"log-inode-substituted": func(t *testing.T, path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("substituted\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"active-type-substituted": func(t *testing.T, path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("log.jsonl", path); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
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
			tombstone := ".deleting-" + id
			tombstonePath := filepath.Join(entries, tombstone)
			if err := os.Rename(filepath.Join(entries, entry), tombstonePath); err != nil {
				t.Fatal(err)
			}
			rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
			if err != nil {
				t.Fatal(err)
			}
			even := readPiLifecycleGenerationForTest(t, rootFD)
			if _, err := beginPiLifecycleOperation(rootFD, even, "delete", entry, tombstone, 0, 0, 0); err != nil {
				t.Fatal(err)
			}
			childName := map[string]string{
				"record-mode-narrowed":    "record.json",
				"log-inode-substituted":   "log.jsonl",
				"active-type-substituted": "active.lock",
			}[name]
			childPath := filepath.Join(tombstonePath, childName)
			mutate(t, childPath)
			unix.Close(rootFD)

			next, err := openPiSessionLog(context.Background(), paths, policy)
			if next != nil {
				defer next.close(context.Background())
			}
			if !piErrorIs(err, "lifecycle_log_evidence_unknown") {
				t.Fatalf("changed %s authority admitted: next=%v err=%v", childName, next != nil, err)
			}
			if _, statErr := os.Lstat(childPath); statErr != nil {
				t.Fatalf("unproven changed child was not preserved: %v", statErr)
			}
		})
	}
}

// Production call site: openPiSessionLog -> recoverPiLifecycle ->
// recoverPiLifecycleDelete. Substitution after the tombstone revalidation but
// before the bounded name-based unlink must preserve both the held original
// and the replacement, leave the generation odd, and return typed unknown.
func TestPiLifecycleDeleteRecoveryPreservesChildSubstitutedImmediatelyBeforeUnlink(t *testing.T) {
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
	tombstone := ".deleting-" + id
	tombstonePath := filepath.Join(entries, tombstone)
	if err := os.Rename(filepath.Join(entries, entry), tombstonePath); err != nil {
		t.Fatal(err)
	}
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	if _, err := beginPiLifecycleOperation(rootFD, even, "delete", entry, tombstone, 0, 0, 0); err != nil {
		unix.Close(rootFD)
		t.Fatal(err)
	}
	unix.Close(rootFD)

	replacement := []byte("delete-recovery-substitution\n")
	substituted := false
	originalHook := piLifecycleBeforeDeleteChildUnlink
	piLifecycleBeforeDeleteChildUnlink = func(entryFD int, name string) error {
		if name != "log.jsonl" || substituted {
			return nil
		}
		substituted = true
		if err := unix.Renameat(entryFD, name, entryFD, "log.jsonl.original"); err != nil {
			return err
		}
		fd, err := unix.Openat(entryFD, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
		if err != nil {
			return err
		}
		if _, err := unix.Write(fd, replacement); err != nil {
			unix.Close(fd)
			return err
		}
		return unix.Close(fd)
	}
	t.Cleanup(func() { piLifecycleBeforeDeleteChildUnlink = originalHook })

	next, recoveryErr := openPiSessionLog(context.Background(), paths, policy)
	if next != nil {
		defer next.close(context.Background())
	}
	if !substituted {
		t.Fatal("scheduling seam did not reach the production unlink boundary")
	}
	if !piErrorIs(recoveryErr, "lifecycle_log_evidence_unknown") {
		t.Errorf("substitution did not return stable typed unknown: next=%v err=%v", next != nil, recoveryErr)
	}
	if _, err := os.Lstat(filepath.Join(tombstonePath, "log.jsonl.original")); err != nil {
		t.Fatalf("held original child was not preserved: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(tombstonePath, "log.jsonl"))
	if errors.Is(err, syscall.ENOENT) {
		t.Error("substituted child was unlinked after its identity check")
		return
	}
	if err != nil || string(data) != string(replacement) {
		t.Fatalf("substituted evidence changed: data=%q err=%v", data, err)
	}
}

// Production call site: openPiSessionLog -> recoverPiLifecycle ->
// recoverPiLifecycleDelete. Evidence appearing after the bounded child scan
// must make the final tombstone removal a typed unknown, never leak raw
// ENOTEMPTY or authorize a new writer.
func TestPiLifecycleDeleteRecoveryTypesFinalTombstoneRemovalFailure(t *testing.T) {
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
	tombstone := ".deleting-" + id
	tombstonePath := filepath.Join(entries, tombstone)
	if err := os.Rename(filepath.Join(entries, entry), tombstonePath); err != nil {
		t.Fatal(err)
	}
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	if _, err := beginPiLifecycleOperation(rootFD, even, "delete", entry, tombstone, 0, 0, 0); err != nil {
		unix.Close(rootFD)
		t.Fatal(err)
	}
	unix.Close(rootFD)

	inserted := false
	originalHook := piLifecycleBeforeDeleteChildUnlink
	piLifecycleBeforeDeleteChildUnlink = func(entryFD int, name string) error {
		if name != "active.lock" || inserted {
			return nil
		}
		inserted = true
		fd, err := unix.Openat(entryFD, "late-evidence", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
		if err != nil {
			return err
		}
		return unix.Close(fd)
	}
	t.Cleanup(func() { piLifecycleBeforeDeleteChildUnlink = originalHook })

	next, recoveryErr := openPiSessionLog(context.Background(), paths, policy)
	if next != nil {
		defer next.close(context.Background())
	}
	if !inserted {
		t.Fatal("scheduling seam did not reach the final child unlink boundary")
	}
	if !piErrorIs(recoveryErr, "lifecycle_log_evidence_unknown") {
		t.Fatalf("final tombstone removal leaked an untyped failure: next=%v err=%v", next != nil, recoveryErr)
	}
	if _, err := os.Lstat(filepath.Join(tombstonePath, "late-evidence")); err != nil {
		t.Fatalf("late evidence was not preserved: %v", err)
	}
}

// Production call sites: PiLifecycleStatus -> readPiLifecycleGenerationPair
// and openPiSessionLog -> recoverPiLifecycle. An even record must carry no
// residual operation authority on either read or next-writer paths.
func TestPiLifecycleMalformedEvenGenerationIsUnknownToStatusAndNextWriter(t *testing.T) {
	for name, mutate := range map[string]func(*piLifecycleGeneration){
		"started-at":       func(value *piLifecycleGeneration) { value.StartedAt = "2026-01-01T00:00:00Z" },
		"committed-before": func(value *piLifecycleGeneration) { value.CommittedBefore = 1 },
		"records-before":   func(value *piLifecycleGeneration) { value.RecordsBefore = 1 },
		"append-bytes":     func(value *piLifecycleGeneration) { value.AppendBytes = 1 },
		"delete-authority": func(value *piLifecycleGeneration) {
			value.DeleteLog = &piLifecycleDeleteIdentity{Device: 1, Inode: 1, Mode: uint32(unix.S_IFREG | 0o600), UID: uint32(os.Geteuid()), Links: 1}
		},
	} {
		t.Run(name, func(t *testing.T) {
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
			mutate(&generation)
			writePiLifecycleGenerationForTest(t, rootFD, generation)
			unix.Close(rootFD)

			status, statusErr := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
			if !piErrorIs(statusErr, "lifecycle_log_evidence_unknown") || status.WithinPolicy || status.SoakReady {
				t.Fatalf("malformed even generation published health: status=%+v err=%v", status, statusErr)
			}
			next, nextErr := openPiSessionLog(context.Background(), paths, policy)
			if next != nil {
				defer next.close(context.Background())
			}
			if !piErrorIs(nextErr, "lifecycle_log_evidence_unknown") {
				t.Fatalf("malformed even generation admitted next writer: next=%v err=%v", next != nil, nextErr)
			}
		})
	}
}

// Production call site: openPiSessionLog -> recoverPiLifecycle. Odd records
// must satisfy the timestamp and exact operation-kind field contract before
// any recovery branch is allowed to interpret them.
func TestPiLifecycleMalformedOddGenerationIsUnknownToStatusAndNextWriter(t *testing.T) {
	for name, build := range map[string]func(*testing.T, int, *piSessionLog, piLifecycleGeneration) piLifecycleGeneration{
		"malformed-started-at": func(t *testing.T, rootFD int, log *piSessionLog, even piLifecycleGeneration) piLifecycleGeneration {
			odd, err := beginPiLifecycleOperation(rootFD, even, "close", log.entry, "", log.record.CommittedBytes, log.record.CommittedRecords, 0)
			if err != nil {
				t.Fatal(err)
			}
			odd.StartedAt = "not-a-timestamp"
			return odd
		},
		"close-staging-authority": func(t *testing.T, rootFD int, log *piSessionLog, even piLifecycleGeneration) piLifecycleGeneration {
			odd, err := beginPiLifecycleOperation(rootFD, even, "close", log.entry, "", log.record.CommittedBytes, log.record.CommittedRecords, 0)
			if err != nil {
				t.Fatal(err)
			}
			odd.StagingName = ".creating-" + log.record.EntryID
			return odd
		},
		"zero-byte-append": func(t *testing.T, rootFD int, log *piSessionLog, even piLifecycleGeneration) piLifecycleGeneration {
			odd, err := beginPiLifecycleOperation(rootFD, even, "append", log.entry, "", log.record.CommittedBytes, log.record.CommittedRecords, 1)
			if err != nil {
				t.Fatal(err)
			}
			odd.AppendBytes = 0
			return odd
		},
		"delete-untrusted-child-uid": func(t *testing.T, rootFD int, log *piSessionLog, even piLifecycleGeneration) piLifecycleGeneration {
			odd, err := beginPiLifecycleOperation(rootFD, even, "delete", log.entry, ".deleting-"+log.record.EntryID, 0, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			odd.DeleteLog.UID++
			return odd
		},
	} {
		t.Run(name, func(t *testing.T) {
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
			odd := build(t, rootFD, log, even)
			writePiLifecycleGenerationForTest(t, rootFD, odd)
			unix.Close(rootFD)

			status, statusErr := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
			if !piErrorIs(statusErr, "lifecycle_log_evidence_unknown") || status.WithinPolicy || status.SoakReady {
				t.Fatalf("malformed odd generation published health: status=%+v err=%v", status, statusErr)
			}
			next, nextErr := openPiSessionLog(context.Background(), paths, policy)
			if next != nil {
				defer next.close(context.Background())
			}
			if !piErrorIs(nextErr, "lifecycle_log_evidence_unknown") {
				t.Fatalf("malformed odd generation admitted next writer: next=%v err=%v", next != nil, nextErr)
			}
		})
	}
}

func writePiLifecycleGenerationForTest(t *testing.T, rootFD int, generation piLifecycleGeneration) {
	t.Helper()
	encoded, err := encodePiLifecycleControl(generation)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePiLifecycleControlAtomic(rootFD, "generation.json", encoded); err != nil {
		t.Fatal(err)
	}
}
