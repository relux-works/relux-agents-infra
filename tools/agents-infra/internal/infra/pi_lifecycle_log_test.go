//go:build !windows

package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestParsePiLifecycleRetentionRequiresEveryExplicitOverflowSafeBudget(t *testing.T) {
	body := validPiProfileTOML("profile", "/bin/echo", 18011, false)
	parsed, err := parseProjectConfig([]byte(body), "config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.PiProfiles["profile"].LifecycleLogRetention; got != testPiLifecyclePolicy() {
		t.Fatalf("retention=%+v want=%+v", got, testPiLifecyclePolicy())
	}
	for _, line := range []string{
		"max_count = 8\n", "max_bytes = 1048576\n", "max_envelope_bytes = 2097152\n",
		"max_age_seconds = 4838400\n", "create_timeout_seconds = 5\n",
		"append_timeout_seconds = 5\n", "close_timeout_seconds = 5\n",
		"status_timeout_seconds = 5\n", "maintenance_timeout_seconds = 5\n",
		"max_scan_entries = 512\n", "max_scan_control_bytes = 262144\n",
		"max_mutations_per_operation = 8\n",
	} {
		t.Run(strings.Fields(line)[0], func(t *testing.T) {
			_, err := parseProjectConfig([]byte(strings.Replace(body, line, "", 1)), "config.toml")
			if err == nil || !strings.Contains(err.Error(), strings.Fields(line)[0]) {
				t.Fatalf("missing explicit %s admitted: %v", strings.Fields(line)[0], err)
			}
		})
	}

	exact := strings.NewReplacer(
		"max_count = 8", "max_count = 2",
		"max_bytes = 1048576", "max_bytes = 1000",
		"max_envelope_bytes = 2097152", "max_envelope_bytes = 9192",
		"max_scan_entries = 512", "max_scan_entries = 33",
		"max_scan_control_bytes = 262144", "max_scan_control_bytes = 45056",
		"max_mutations_per_operation = 8", "max_mutations_per_operation = 5",
	).Replace(body)
	if _, err := parseProjectConfig([]byte(exact), "config.toml"); err != nil {
		t.Fatalf("exact retention formulas refused: %v", err)
	}
	for name, narrowed := range map[string]string{
		"envelope": strings.Replace(exact, "max_envelope_bytes = 9192", "max_envelope_bytes = 9191", 1),
		"entries":  strings.Replace(exact, "max_scan_entries = 33", "max_scan_entries = 32", 1),
		"control":  strings.Replace(exact, "max_scan_control_bytes = 45056", "max_scan_control_bytes = 45055", 1),
	} {
		t.Run("narrow-"+name, func(t *testing.T) {
			if _, err := parseProjectConfig([]byte(narrowed), "config.toml"); err == nil {
				t.Fatalf("narrowed %s formula admitted", name)
			}
		})
	}
}

// Production call site: RunPi -> loadCompositeProjectConfig ->
// parsePiLifecycleLogRetention. Every policy member is mandatory and refusal
// must precede executable lookup and any cache mutation.
func TestRunPiRejectsMissingLifecycleRetentionPolicyBeforeProviderOrState(t *testing.T) {
	for _, field := range []string{
		"max_count = 8\n",
		"max_envelope_bytes = 2097152\n",
		"max_scan_control_bytes = 262144\n",
	} {
		t.Run(strings.Fields(field)[0], func(t *testing.T) {
			project, home := t.TempDir(), t.TempDir()
			cache := filepath.Join(t.TempDir(), "cache")
			writePiProjectConfig(t, project, strings.Replace(validPiProfileTOML("profile", "/bin/echo", 18011, false), field, "", 1))
			lookedUp := false
			err := RunPi(RunPiOptions{
				ProjectDir: project,
				HomeDir:    home,
				CacheRoot:  cache,
				LookPath: func(string) (string, error) {
					lookedUp = true
					return "/bin/false", nil
				},
			})
			if piErrorCode(err) != "invalid_project_configuration" || !strings.Contains(err.Error(), strings.Fields(field)[0]) {
				t.Fatalf("RunPi admitted absent %s: %v", strings.Fields(field)[0], err)
			}
			if lookedUp {
				t.Fatal("lifecycle policy refusal reached executable lookup")
			}
			if _, statErr := os.Stat(cache); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("lifecycle policy refusal mutated cache: %v", statErr)
			}
		})
	}
}

// Production call site: RunPi -> openPiSessionLog -> scanPiLifecycleEntries.
// Foreign aggregate evidence must refuse before runtime Start or Pi spawn.
func TestRunPiLifecycleFilesystemAuthorityRefusesBeforeRuntimeSideEffect(t *testing.T) {
	piRoot := officialPiAsset(t)
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	marker := filepath.Join(t.TempDir(), "runtime-started")
	runtime := filepath.Join(t.TempDir(), "runtime.sh")
	mustWrite(t, runtime, "#!/bin/sh\ntouch \""+marker+"\"\n")
	if err := os.Chmod(runtime, 0o700); err != nil {
		t.Fatal(err)
	}
	writePiProjectConfig(t, project, validPiProfileTOML("profile", runtime, 18011, false))
	canonical, err := CanonicalProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := ResolvePiStatePaths(cache, canonical, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	seed, err := openPiSessionLog(context.Background(), paths, testPiLifecyclePolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(paths.LifecycleLogsRoot, "foreign"), "preserve")
	err = runPiFixture(project, home, cache, piRoot, nil)
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("RunPi admitted foreign lifecycle evidence: %v", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runtime side effect became reachable: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(paths.LifecycleLogsRoot, "foreign")); err != nil || string(data) != "preserve" {
		t.Fatalf("foreign evidence was changed: data=%q err=%v", data, err)
	}
}

func TestPiLifecycleAggregateRootIsProfileWideAndEnvelopeIsStrict(t *testing.T) {
	cache, project := t.TempDir(), filepath.Join(t.TempDir(), "project")
	mustMkdir(t, project)
	a, err := ResolvePiClientStatePaths(cache, project, "profile", "run-a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ResolvePiClientStatePaths(cache, project, "profile", "run-b")
	if err != nil {
		t.Fatal(err)
	}
	if a.Root == b.Root || a.LifecycleLogsRoot != b.LifecycleLogsRoot || a.ProfileRoot != b.ProfileRoot {
		t.Fatalf("per-run roots or aggregate roots are wrong: a=%+v b=%+v", a, b)
	}
	if err := CreatePiStateTree(a); err != nil {
		t.Fatal(err)
	}
	log, err := openPiSessionLog(context.Background(), a, testPiLifecyclePolicy())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(log.path, filepath.Join(a.LifecycleLogsRoot, "entries")+string(filepath.Separator)) {
		t.Fatalf("lifecycle log escaped aggregate root: %s", log.path)
	}
	children, err := os.ReadDir(filepath.Dir(log.path))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, child := range children {
		names = append(names, child.Name())
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "active.lock,log.jsonl,record.json" {
		t.Fatalf("envelope children=%q", names)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, err := PiLifecycleStatus(context.Background(), a, testPiLifecyclePolicy(), "test-policy", "")
	if err != nil {
		t.Fatal(err)
	}
	if !status.ScanComplete || !status.WithinPolicy || !status.SoakReady || status.ManagedCount != 1 || status.ActiveCount != 0 {
		t.Fatalf("closed aggregate status=%+v", status)
	}
}

func TestPiLifecycleConcurrentAdmissionRefusesBeforeSecondEnvelope(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	policy.MaxCount = 1
	policy.MaxMutationsPerOperation = 1
	policy.MaxScanEntries = 13
	policy.MaxScanControlBytes = 24576
	policy.MaxEnvelopeBytes = policy.MaxBytes + piLifecycleControlLimit
	first, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer first.close(context.Background())
	if _, err := openPiSessionLog(context.Background(), paths, policy); !piErrorIs(err, "lifecycle_log_retention_refused") {
		t.Fatalf("second active admission error=%v", err)
	}
	entries, err := os.ReadDir(filepath.Join(paths.LifecycleLogsRoot, "entries"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("refused admission created %d entries", len(entries))
	}
}

func TestPiLifecycleOddAppendRecoversExactCommittedBoundary(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.event(context.Background(), "committed", map[string]any{"value": 1}); err != nil {
		t.Fatal(err)
	}
	entry, committed, records := log.entry, log.record.CommittedBytes, log.record.CommittedRecords
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(paths.LifecycleLogsRoot, "entries", entry, "log.jsonl")
	file, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("partial"); err != nil {
		t.Fatal(err)
	}
	file.Close()
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	genBytes, _, err := readPiLifecycleControl(rootFD, "generation.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var even piLifecycleGeneration
	if err := decodePiLifecycleControl(genBytes, &even); err != nil {
		t.Fatal(err)
	}
	odd, err := beginPiLifecycleOperation(rootFD, even, "append", entry, "", committed, records, int64(len("partial")))
	if err != nil || odd.State != "odd" {
		t.Fatalf("publish odd append: generation=%+v err=%v", odd, err)
	}
	unix.Close(rootFD)
	recovered, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatalf("next writer failed exact recovery: %v", err)
	}
	defer recovered.close(context.Background())
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != committed {
		t.Fatalf("recovered size=%d want committed=%d", info.Size(), committed)
	}
}

func TestPiLifecycleOddAppendRecognizesFullyCommittedPhase(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	beforeBytes, beforeRecords := log.record.CommittedBytes, log.record.CommittedRecords
	if err := log.event(context.Background(), "committed", nil); err != nil {
		t.Fatal(err)
	}
	appendBytes := log.record.CommittedBytes - beforeBytes
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	if _, err := beginPiLifecycleOperation(rootFD, even, "append", log.entry, "", beforeBytes, beforeRecords, appendBytes); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)
	next, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatalf("fully committed append was not recognized: %v", err)
	}
	defer next.close(context.Background())
	if info, err := os.Stat(log.path); err != nil || info.Size() != log.record.CommittedBytes {
		t.Fatalf("fully committed append was changed: size=%v err=%v", info, err)
	}
	_ = log.close(context.Background())
}

func TestPiLifecycleOddCloseRecoversBeforeAndAfterRecordPublication(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-record", true: "after-record"}[published], func(t *testing.T) {
			paths := newPiLifecycleTestState(t)
			policy := testPiLifecyclePolicy()
			log, err := openPiSessionLog(context.Background(), paths, policy)
			if err != nil {
				t.Fatal(err)
			}
			rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
			if err != nil {
				t.Fatal(err)
			}
			even := readPiLifecycleGenerationForTest(t, rootFD)
			odd, err := beginPiLifecycleOperation(rootFD, even, "close", log.entry, "", log.record.CommittedBytes, log.record.CommittedRecords, 0)
			if err != nil {
				t.Fatal(err)
			}
			if published {
				record := log.record
				record.ClosedAt = odd.StartedAt
				encoded, err := encodePiLifecycleControl(record)
				if err != nil {
					t.Fatal(err)
				}
				if err := writePiLifecycleControlAtomic(log.entryFD, "record.json", encoded); err != nil {
					t.Fatal(err)
				}
			}
			unix.Close(rootFD)
			next, err := openPiSessionLog(context.Background(), paths, policy)
			if err != nil {
				t.Fatalf("recover close phase: %v", err)
			}
			_ = next.close(context.Background())
			_ = log.close(context.Background())
		})
	}
}

func TestPiLifecycleOddCreateRecoversPublishedAndStrictSubsetPhases(t *testing.T) {
	for _, complete := range []bool{true, false} {
		t.Run(map[bool]string{true: "complete", false: "strict-subset"}[complete], func(t *testing.T) {
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
			staging := ".creating-" + id
			entries := filepath.Join(paths.LifecycleLogsRoot, "entries")
			if err := os.Rename(filepath.Join(entries, entry), filepath.Join(entries, staging)); err != nil {
				t.Fatal(err)
			}
			if !complete {
				if err := os.Remove(filepath.Join(entries, staging, "record.json")); err != nil {
					t.Fatal(err)
				}
			}
			rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
			if err != nil {
				t.Fatal(err)
			}
			even := readPiLifecycleGenerationForTest(t, rootFD)
			if _, err := beginPiLifecycleOperation(rootFD, even, "create", entry, staging, 0, 0, 0); err != nil {
				t.Fatal(err)
			}
			unix.Close(rootFD)
			next, err := openPiSessionLog(context.Background(), paths, policy)
			if err != nil {
				t.Fatalf("recover create phase: %v", err)
			}
			defer next.close(context.Background())
			if _, err := os.Lstat(filepath.Join(entries, staging)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("create staging survived recovery: %v", err)
			}
			_, finalErr := os.Stat(filepath.Join(entries, entry))
			if complete && finalErr != nil {
				t.Fatalf("complete create was not published: %v", finalErr)
			}
			if !complete && !errors.Is(finalErr, os.ErrNotExist) {
				t.Fatalf("strict create subset was not removed: %v", finalErr)
			}
		})
	}
}

func TestPiLifecycleOddDeleteResumesExactTombstoneWithoutRecursion(t *testing.T) {
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
	if err := os.Rename(filepath.Join(entries, entry), filepath.Join(entries, tombstone)); err != nil {
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
	if err := os.Remove(filepath.Join(entries, tombstone, "record.json")); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)
	next, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatalf("resume delete: %v", err)
	}
	defer next.close(context.Background())
	if _, err := os.Lstat(filepath.Join(entries, tombstone)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("delete tombstone survived recovery: %v", err)
	}
}

func TestPiLifecycleStatusIsLockFreeAndOddOrPagedEvidenceIsNeverHealthy(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	rootFD, foreground, err := openAndLockPiLifecycleRoot(context.Background(), paths)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	status, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if err != nil || time.Since(started) > time.Second || !status.ScanComplete {
		t.Fatalf("status queued on writer lock or failed: elapsed=%s status=%+v err=%v", time.Since(started), status, err)
	}
	foreground.Close()
	genBytes, _, err := readPiLifecycleControl(rootFD, "generation.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var even piLifecycleGeneration
	if err := decodePiLifecycleControl(genBytes, &even); err != nil {
		t.Fatal(err)
	}
	if _, err := beginPiLifecycleOperation(rootFD, even, "append", log.entry, "", log.record.CommittedBytes, log.record.CommittedRecords, 1); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)
	oddStatus, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") || oddStatus.WithinPolicy || oddStatus.SoakReady {
		t.Fatalf("odd generation laundered healthy: status=%+v err=%v", oddStatus, err)
	}
}

func TestPiLifecycleFreshRunEightWeekFakeClockStaysBounded(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	policy.MaxCount = 3
	policy.MaxMutationsPerOperation = 8
	policy.MaxScanEntries = 49
	policy.MaxScanControlBytes = 61440
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	originalNow := piLifecycleNow
	t.Cleanup(func() { piLifecycleNow = originalNow })
	current := base
	piLifecycleNow = func() time.Time { return current }
	for week := 0; week < 8; week++ {
		current = base.Add(time.Duration(week) * 7 * 24 * time.Hour)
		log, err := openPiSessionLog(context.Background(), paths, policy)
		if err != nil {
			t.Fatalf("week %d create: %v", week, err)
		}
		if err := log.event(context.Background(), "week", map[string]any{"week": week}); err != nil {
			t.Fatalf("week %d append: %v", week, err)
		}
		if err := log.close(context.Background()); err != nil {
			t.Fatalf("week %d close: %v", week, err)
		}
	}
	status, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if err != nil {
		t.Fatal(err)
	}
	if status.ManagedCount != policy.MaxCount || !status.ScanComplete || !status.WithinPolicy || !status.SoakReady {
		t.Fatalf("eight-week status=%+v", status)
	}
}

func TestPiLifecycleAppendHonorsExactCommittedByteCap(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	record := piSessionLogRecord{Timestamp: "2026-01-01T00:00:00Z", Event: "cap", Fields: map[string]any{"v": 1}}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	policy := testPiLifecyclePolicy()
	policy.MaxCount = 1
	policy.MaxBytes = len(encoded) + 1
	policy.MaxEnvelopeBytes = policy.MaxBytes + piLifecycleControlLimit
	policy.MaxMutationsPerOperation = 1
	policy.MaxScanEntries = 13
	policy.MaxScanControlBytes = 24576
	originalNow := piLifecycleNow
	t.Cleanup(func() { piLifecycleNow = originalNow })
	piLifecycleNow = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer log.close(context.Background())
	if err := log.event(context.Background(), "cap", map[string]any{"v": 1}); err != nil {
		t.Fatalf("exact max_bytes append refused: %v", err)
	}
	if err := log.event(context.Background(), "over", nil); !piErrorIs(err, "lifecycle_log_retention_refused") {
		t.Fatalf("over max_bytes append error=%v", err)
	}
}

func TestPiLifecycleAppendHonorsExactEnvelopeByteCapIndependently(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer log.close(context.Background())
	originalNow := piLifecycleNow
	t.Cleanup(func() { piLifecycleNow = originalNow })
	piLifecycleNow = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	data, err := json.Marshal(piSessionLogRecord{Timestamp: piLifecycleNow().UTC().Format(time.RFC3339Nano), Event: "cap", Fields: map[string]any{"v": 1}})
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	next := log.record
	next.CommittedBytes += int64(len(data))
	next.CommittedRecords++
	nextRecord, err := encodePiLifecycleControl(next)
	if err != nil {
		t.Fatal(err)
	}
	log.policy.MaxBytes = 1 << 20
	log.policy.MaxEnvelopeBytes = len(data) + len(nextRecord)
	if err := log.event(context.Background(), "cap", map[string]any{"v": 1}); err != nil {
		t.Fatalf("exact max_envelope_bytes append refused: %v", err)
	}
	log.policy.MaxEnvelopeBytes--
	if err := log.event(context.Background(), "over", nil); !piErrorIs(err, "lifecycle_log_retention_refused") {
		t.Fatalf("narrowed max_envelope_bytes append error=%v", err)
	}
}

func TestPiLifecycleWriterRejectsActivePathSubstitutionAndPreservesEvidence(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	log, err := openPiSessionLog(context.Background(), paths, testPiLifecyclePolicy())
	if err != nil {
		t.Fatal(err)
	}
	original := log.path + ".preserved"
	if err := os.Rename(log.path, original); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(log.path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	err = log.event(context.Background(), "must-refuse", nil)
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("active path substitution admitted: %v", err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatalf("original evidence was not preserved: %v", err)
	}
	if _, err := os.Stat(log.path); err != nil {
		t.Fatalf("replacement evidence was not preserved: %v", err)
	}
	_ = log.close(context.Background())
}

func TestPiLifecycleFinalContinuationPageCannotPublishHealth(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < policy.MaxScanEntries+32; index++ {
		if err := os.WriteFile(filepath.Join(paths.LogsDir, fmt.Sprintf("legacy-%04d.jsonl", index)), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	status, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if !piErrorIs(err, "lifecycle_log_scan_exhausted") || status.Continuation == "" {
		t.Fatalf("production scan did not publish first continuation: status=%+v err=%v", status, err)
	}
	for page := 0; page < 32; page++ {
		previous := status.Continuation
		status, err = PiLifecycleStatus(context.Background(), paths, policy, "test", previous)
		if err == nil {
			if status.ScanComplete || !status.LowerBound || status.WithinPolicy || status.SoakReady || status.Continuation != "" {
				t.Fatalf("final continuation page published whole-scan health: %+v", status)
			}
			return
		}
		if !piErrorIs(err, "lifecycle_log_scan_exhausted") || status.Continuation == "" || status.Continuation == previous || status.ScanComplete || !status.LowerBound || status.WithinPolicy || status.SoakReady {
			t.Fatalf("continuation page %d did not advance safely: status=%+v err=%v", page, status, err)
		}
	}
	t.Fatal("continuation did not reach a final lower-bound page")
}

func TestPiLifecycleLegacyContinuationRejectsDirectoryMutation(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < policy.MaxScanEntries+32; index++ {
		if err := os.WriteFile(filepath.Join(paths.LogsDir, fmt.Sprintf("legacy-%04d.jsonl", index)), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	first, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if !piErrorIs(err, "lifecycle_log_scan_exhausted") || first.Continuation == "" {
		t.Fatalf("first legacy page=%+v err=%v", first, err)
	}
	if err := os.WriteFile(filepath.Join(paths.LogsDir, "legacy-mutated.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resumed, err := PiLifecycleStatus(context.Background(), paths, policy, "test", first.Continuation)
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") || resumed.WithinPolicy || resumed.SoakReady || resumed.UnknownCount == 0 {
		t.Fatalf("mutated continuation was admitted: status=%+v err=%v", resumed, err)
	}
}

func TestPiLifecycleRunLogContinuationAdvancesThroughProductionScanner(t *testing.T) {
	cache, project := t.TempDir(), filepath.Join(t.TempDir(), "project")
	mustMkdir(t, project)
	paths, err := ResolvePiClientStatePaths(cache, project, "profile", "run-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < policy.MaxScanEntries+32; index++ {
		if err := os.WriteFile(filepath.Join(paths.LogsDir, fmt.Sprintf("run-legacy-%04d.jsonl", index)), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	status, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if !piErrorIs(err, "lifecycle_log_scan_exhausted") || status.Continuation == "" || status.PageScope != "legacy-run-logs" {
		t.Fatalf("run-log scan did not publish its production cursor: status=%+v err=%v", status, err)
	}
	for page := 0; page < 32; page++ {
		previous := status.Continuation
		status, err = PiLifecycleStatus(context.Background(), paths, policy, "test", previous)
		if err == nil {
			if status.ScanComplete || !status.LowerBound || status.WithinPolicy || status.SoakReady {
				t.Fatalf("final run-log continuation claimed health: %+v", status)
			}
			return
		}
		if !piErrorIs(err, "lifecycle_log_scan_exhausted") || status.Continuation == "" || status.Continuation == previous {
			t.Fatalf("run-log page %d failed to advance: status=%+v err=%v", page, status, err)
		}
	}
	t.Fatal("run-log continuation did not converge")
}

func TestPiLifecycleContinuationRejectsPolicyAndGenerationChange(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < policy.MaxScanEntries+32; index++ {
		if err := os.WriteFile(filepath.Join(paths.LogsDir, fmt.Sprintf("legacy-%04d.jsonl", index)), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	first, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if !piErrorIs(err, "lifecycle_log_scan_exhausted") || first.Continuation == "" {
		t.Fatalf("first page=%+v err=%v", first, err)
	}
	changedPolicy := policy
	changedPolicy.MaxAgeSeconds++
	if status, err := PiLifecycleStatus(context.Background(), paths, changedPolicy, "test", first.Continuation); !piErrorIs(err, "lifecycle_log_evidence_unknown") || status.WithinPolicy || status.SoakReady {
		t.Fatalf("policy-changed token admitted: status=%+v err=%v", status, err)
	}
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	generation := readPiLifecycleGenerationForTest(t, rootFD)
	generation.Generation += 2
	encoded, err := encodePiLifecycleControl(generation)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePiLifecycleControlAtomic(rootFD, "generation.json", encoded); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)
	if status, err := PiLifecycleStatus(context.Background(), paths, policy, "test", first.Continuation); !piErrorIs(err, "lifecycle_log_evidence_unknown") || status.WithinPolicy || status.SoakReady {
		t.Fatalf("generation-changed token admitted: status=%+v err=%v", status, err)
	}
}

func TestPiLifecycleStatusRejectsNarrowedAggregateDirectoryAuthority(t *testing.T) {
	for _, target := range []string{".", "entries"} {
		t.Run(target, func(t *testing.T) {
			paths := newPiLifecycleTestState(t)
			log, err := openPiSessionLog(context.Background(), paths, testPiLifecyclePolicy())
			if err != nil {
				t.Fatal(err)
			}
			if err := log.close(context.Background()); err != nil {
				t.Fatal(err)
			}
			path := paths.LifecycleLogsRoot
			if target != "." {
				path = filepath.Join(path, target)
			}
			if err := os.Chmod(path, 0o755); err != nil {
				t.Fatal(err)
			}
			status, err := PiLifecycleStatus(context.Background(), paths, testPiLifecyclePolicy(), "test", "")
			if !piErrorIs(err, "lifecycle_log_evidence_unknown") || status.WithinPolicy || status.SoakReady || status.UnknownCount == 0 {
				t.Fatalf("status attested narrowed %s authority: status=%+v err=%v", target, status, err)
			}
		})
	}
}

func TestPiLifecycleOddAppendRecoversAtomicRecordTempPhase(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer log.close(context.Background())
	data := []byte("{\"timestamp\":\"2026-01-01T00:00:00Z\",\"event\":\"temp-phase\"}\n")
	rootFD, err := unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	even := readPiLifecycleGenerationForTest(t, rootFD)
	before := log.record
	if _, err := beginPiLifecycleOperation(rootFD, even, "append", log.entry, "", before.CommittedBytes, before.CommittedRecords, int64(len(data))); err != nil {
		t.Fatal(err)
	}
	if n, err := log.file.WriteAt(data, before.CommittedBytes); err != nil || n != len(data) {
		t.Fatalf("append phase write n=%d err=%v", n, err)
	}
	if err := log.file.Sync(); err != nil {
		t.Fatal(err)
	}
	after := before
	after.CommittedBytes += int64(len(data))
	after.CommittedRecords++
	encoded, err := encodePiLifecycleControl(after)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePiLifecycleControlExclusive(log.entryFD, piLifecycleAtomicTempName("record.json"), encoded); err != nil {
		t.Fatal(err)
	}
	unix.Close(rootFD)

	next, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatalf("next writer did not recover exact record temp: %v", err)
	}
	defer next.close(context.Background())
	if _, err := os.Lstat(filepath.Join(filepath.Dir(log.path), piLifecycleAtomicTempName("record.json"))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("record temp survived recovery: %v", err)
	}
	rootFD, err = unix.Open(paths.LifecycleLogsRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(rootFD)
	generation := readPiLifecycleGenerationForTest(t, rootFD)
	if generation.State != "even" || generation.Generation < even.Generation+2 || generation.Recovered == 0 {
		t.Fatalf("recovered generation=%+v", generation)
	}
}

func TestPiLifecycleRecoversAtomicGenerationTempPhases(t *testing.T) {
	for _, phase := range []string{"before-odd-rename", "before-even-rename"} {
		t.Run(phase, func(t *testing.T) {
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
			odd := piLifecycleGeneration{
				SchemaVersion: piLifecycleSchemaVersion, Generation: even.Generation + 1,
				State: "odd", Scope: "aggregate", OperationID: strings.Repeat("a", 32),
				OperationKind: "close", StartedAt: "2026-01-01T00:00:00Z",
				EntryName: log.entry, CommittedBefore: log.record.CommittedBytes,
				RecordsBefore: log.record.CommittedRecords, Recovered: even.Recovered, Pruned: even.Pruned,
			}
			if phase == "before-even-rename" {
				encoded, encodeErr := encodePiLifecycleControl(odd)
				if encodeErr != nil {
					t.Fatal(encodeErr)
				}
				if err := writePiLifecycleControlAtomic(rootFD, "generation.json", encoded); err != nil {
					t.Fatal(err)
				}
				odd = completedPiLifecycleGeneration(odd, 0, 0)
			}
			encoded, err := encodePiLifecycleControl(odd)
			if err != nil {
				t.Fatal(err)
			}
			if err := writePiLifecycleControlExclusive(rootFD, piLifecycleAtomicTempName("generation.json"), encoded); err != nil {
				t.Fatal(err)
			}
			unix.Close(rootFD)
			next, err := openPiSessionLog(context.Background(), paths, policy)
			if err != nil {
				t.Fatalf("next writer did not recover %s: %v", phase, err)
			}
			defer next.close(context.Background())
			if _, err := os.Lstat(filepath.Join(paths.LifecycleLogsRoot, piLifecycleAtomicTempName("generation.json"))); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("generation temp survived %s: %v", phase, err)
			}
		})
	}
}

func TestPiLifecycleGenerationReadFailureIsNotLaunderedAsAbsence(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	generation := filepath.Join(paths.LifecycleLogsRoot, "generation.json")
	preserved := generation + ".preserved"
	if err := os.Rename(generation, preserved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(preserved, generation); err != nil {
		t.Fatal(err)
	}
	status, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if !piErrorIs(err, "lifecycle_log_evidence_unknown") || status.UnknownCount == 0 || status.WithinPolicy || status.SoakReady {
		t.Fatalf("generation read failure laundered: status=%+v err=%v", status, err)
	}
	if _, err := os.Lstat(preserved); err != nil {
		t.Fatalf("generation evidence was not preserved: %v", err)
	}
}

func TestPiLifecycleExistingIncompleteRootIsUnknownAndNotRepaired(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	if err := os.Mkdir(paths.LifecycleLogsRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := openPiSessionLog(context.Background(), paths, testPiLifecyclePolicy()); !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("incomplete existing root was admitted: %v", err)
	}
	children, err := os.ReadDir(paths.LifecycleLogsRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 0 {
		t.Fatalf("incomplete root was mutated: %v", children)
	}
}

func TestPiLifecycleScannerHonorsExactAndNarrowedWorkBounds(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	baseline, err := PiLifecycleStatus(context.Background(), paths, policy, "test", "")
	if err != nil {
		t.Fatal(err)
	}
	exact := policy
	exact.MaxScanEntries = baseline.ScanEntries
	exact.MaxScanControlBytes = baseline.ScanControlBytes
	if status, err := PiLifecycleStatus(context.Background(), paths, exact, "test", ""); err != nil || !status.ScanComplete {
		t.Fatalf("exact scanner bounds refused: status=%+v err=%v", status, err)
	}
	for name, narrowed := range map[string]PiLifecycleLogRetention{
		"entries": func() PiLifecycleLogRetention { value := exact; value.MaxScanEntries--; return value }(),
		"control": func() PiLifecycleLogRetention { value := exact; value.MaxScanControlBytes--; return value }(),
	} {
		t.Run(name, func(t *testing.T) {
			status, err := PiLifecycleStatus(context.Background(), paths, narrowed, "test", "")
			if !piErrorIs(err, "lifecycle_log_scan_exhausted") || !status.ScanExhausted || status.WithinPolicy || status.SoakReady {
				t.Fatalf("narrowed %s bound admitted: status=%+v err=%v", name, status, err)
			}
		})
	}
}

func TestPiLifecycleCloseReleasesActiveLockWhenForegroundDeadlineExpires(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	foreground := lockPiLifecycleFile(t, filepath.Join(paths.LifecycleLogsRoot, "foreground.lock"))
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := log.close(ctx); !piErrorIs(err, "lifecycle_log_lock_timeout") {
		t.Fatalf("close under held foreground lock error=%v", err)
	}
	foreground.Close()
	active, err := os.OpenFile(filepath.Join(filepath.Dir(log.path), "active.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer active.Close()
	if err := syscall.Flock(int(active.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("close deadline left active reservation locked: %v", err)
	}
	_ = syscall.Flock(int(active.Fd()), syscall.LOCK_UN)
}

func newPiLifecycleTestState(t *testing.T) PiStatePaths {
	t.Helper()
	cache, project := t.TempDir(), filepath.Join(t.TempDir(), "project")
	mustMkdir(t, project)
	paths, err := ResolvePiStatePaths(cache, project, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	return paths
}

func requirePiLifecycleError(t *testing.T, err error, code string) {
	t.Helper()
	var launch *PiLaunchError
	if !errors.As(err, &launch) || launch.Code != code {
		t.Fatalf("error=%v code=%q want=%q", err, launch.Code, code)
	}
}

func lockPiLifecycleFile(t *testing.T, path string) *os.File {
	t.Helper()
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		t.Fatal(err)
	}
	return file
}

func readPiLifecycleGenerationForTest(t *testing.T, rootFD int) piLifecycleGeneration {
	t.Helper()
	encoded, _, err := readPiLifecycleControl(rootFD, "generation.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var generation piLifecycleGeneration
	if err := decodePiLifecycleControl(encoded, &generation); err != nil {
		t.Fatal(err)
	}
	return generation
}
