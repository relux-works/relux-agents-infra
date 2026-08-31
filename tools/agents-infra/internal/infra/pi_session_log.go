//go:build !windows

package infra

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	piLifecycleSchemaVersion = 1
	piLifecycleControlLimit  = 4096
)

var (
	piLifecycleNow       = time.Now
	piLifecycleRandom    = rand.Reader
	piLifecycleEntryName = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{9}Z-[0-9a-f]{32}$`)
	// piLifecycleBeforeDeleteChildUnlink is a deterministic scheduling seam
	// for production-entry substitution tests. Production leaves it nil.
	piLifecycleBeforeDeleteChildUnlink func(entryFD int, name string) error
)

type piSessionLog struct {
	mu        sync.Mutex
	paths     PiStatePaths
	policy    PiLifecycleLogRetention
	entry     string
	entryFD   int
	file      *os.File
	active    *os.File
	record    piLifecycleRecord
	path      string
	closed    bool
	appendErr error
}

type piSessionLogRecord struct {
	Timestamp string         `json:"timestamp"`
	Event     string         `json:"event"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type piLifecycleRecord struct {
	SchemaVersion    int    `json:"schema_version"`
	EntryID          string `json:"entry_id"`
	CreatedAt        string `json:"created_at"`
	ClosedAt         string `json:"closed_at,omitempty"`
	CommittedBytes   int64  `json:"committed_bytes"`
	CommittedRecords uint64 `json:"committed_records"`
	DirectoryDevice  uint64 `json:"directory_device"`
	DirectoryInode   uint64 `json:"directory_inode"`
	LogDevice        uint64 `json:"log_device"`
	LogInode         uint64 `json:"log_inode"`
	ActiveDevice     uint64 `json:"active_device"`
	ActiveInode      uint64 `json:"active_inode"`
}

type piLifecycleGeneration struct {
	SchemaVersion   int                        `json:"schema_version"`
	Generation      uint64                     `json:"generation"`
	State           string                     `json:"state"`
	Scope           string                     `json:"scope"`
	OperationID     string                     `json:"operation_id,omitempty"`
	OperationKind   string                     `json:"operation_kind,omitempty"`
	StartedAt       string                     `json:"started_at,omitempty"`
	EntryName       string                     `json:"entry_name,omitempty"`
	StagingName     string                     `json:"staging_name,omitempty"`
	DeleteDir       *piLifecycleDeleteIdentity `json:"delete_directory,omitempty"`
	DeleteRecord    *piLifecycleDeleteIdentity `json:"delete_record,omitempty"`
	DeleteLog       *piLifecycleDeleteIdentity `json:"delete_log,omitempty"`
	DeleteActive    *piLifecycleDeleteIdentity `json:"delete_active,omitempty"`
	CommittedBefore int64                      `json:"committed_before,omitempty"`
	RecordsBefore   uint64                     `json:"records_before,omitempty"`
	AppendBytes     int64                      `json:"append_bytes,omitempty"`
	Recovered       int                        `json:"recovered,omitempty"`
	Pruned          int                        `json:"pruned,omitempty"`
}

type piLifecycleDeleteIdentity struct {
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
	Mode   uint32 `json:"mode"`
	UID    uint32 `json:"uid"`
	Links  uint64 `json:"links"`
}

type piLifecycleEnvelope struct {
	Name          string
	Record        piLifecycleRecord
	RecordBytes   int64
	LogBytes      int64
	EnvelopeBytes int64
	Active        bool
}

type piLifecycleBudget struct {
	policy       PiLifecycleLogRetention
	entries      int
	controlBytes int
	mutations    int
	controlLimit int
}

type piLifecycleContinuation struct {
	SchemaVersion       int    `json:"schema_version"`
	ProjectKey          string `json:"project_key"`
	ProfileKey          string `json:"profile_key"`
	PolicyDigest        string `json:"policy_digest"`
	AggregateGeneration uint64 `json:"aggregate_generation"`
	LegacyGeneration    uint64 `json:"legacy_generation"`
	RootDevice          uint64 `json:"root_device"`
	RootInode           uint64 `json:"root_inode"`
	RootChangeNsec      int64  `json:"root_change_nsec"`
	Phase               string `json:"phase"`
	Offset              int64  `json:"offset"`
	ParentOffset        int64  `json:"parent_offset,omitempty"`
	RunName             string `json:"run_name,omitempty"`
	DirectoryDevice     uint64 `json:"directory_device"`
	DirectoryInode      uint64 `json:"directory_inode"`
	DirectoryChangeNsec int64  `json:"directory_change_nsec"`
	ParentDevice        uint64 `json:"parent_device,omitempty"`
	ParentInode         uint64 `json:"parent_inode,omitempty"`
	ParentChangeNsec    int64  `json:"parent_change_nsec,omitempty"`
}

type piLifecycleDirent struct {
	Name   string
	Cookie int64
}

type piLifecycleDirectoryIdentity struct {
	Device     uint64
	Inode      uint64
	ChangeNsec int64
}

func openPiSessionLog(ctx context.Context, paths PiStatePaths, policy PiLifecycleLogRetention) (*piSessionLog, error) {
	opCtx, cancel := piLifecycleOperationContext(ctx, policy.CreateTimeoutSeconds)
	defer cancel()
	rootFD, foreground, err := openAndLockPiLifecycleRoot(opCtx, paths)
	if err != nil {
		return nil, err
	}
	defer unix.Close(rootFD)
	defer foreground.Close()
	retention, err := acquirePiLifecycleLock(opCtx, rootFD, "retention.lock")
	if err != nil {
		return nil, err
	}
	defer retention.Close()
	budget := &piLifecycleBudget{policy: policy}
	generation, err := recoverPiLifecycle(opCtx, rootFD, budget)
	if err != nil {
		return nil, err
	}
	legacyGeneration, err := readPiLifecycleLegacyGeneration(rootFD, nil)
	if err != nil || legacyGeneration.State != "even" {
		if err == nil {
			err = errors.New("explicit legacy retirement is incomplete")
		}
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	envelopes, err := scanPiLifecycleEntries(opCtx, rootFD, budget, nil)
	if err != nil {
		return nil, err
	}
	maintenanceCtx, maintenanceCancel := piLifecycleOperationContext(ctx, policy.MaintenanceTimeoutSeconds)
	defer maintenanceCancel()
	if err := prunePiLifecycleForCreate(maintenanceCtx, rootFD, policy, budget, &generation, envelopes); err != nil {
		return nil, err
	}
	return createPiLifecycleEntry(opCtx, paths, rootFD, policy, budget, &generation)
}

func (log *piSessionLog) event(ctx context.Context, name string, fields map[string]any) (result error) {
	if log == nil {
		return piError("lifecycle_log_evidence_unknown", errors.New("lifecycle log is absent"))
	}
	data, err := json.Marshal(piSessionLogRecord{Timestamp: piLifecycleNow().UTC().Format(time.RFC3339Nano), Event: name, Fields: fields})
	if err != nil {
		return piError("lifecycle_log_append_refused", err)
	}
	data = append(data, '\n')
	log.mu.Lock()
	defer func() {
		if result != nil && log.appendErr == nil {
			log.appendErr = result
		}
		log.mu.Unlock()
	}()
	if log.closed || log.file == nil || log.active == nil {
		return piError("lifecycle_log_append_refused", errors.New("lifecycle log is closed"))
	}
	opCtx, cancel := piLifecycleOperationContext(ctx, log.policy.AppendTimeoutSeconds)
	defer cancel()
	rootFD, foreground, err := openAndLockPiLifecycleRoot(opCtx, log.paths)
	if err != nil {
		return err
	}
	defer unix.Close(rootFD)
	defer foreground.Close()
	retention, err := acquirePiLifecycleLock(opCtx, rootFD, "retention.lock")
	if err != nil {
		return err
	}
	defer retention.Close()
	budget := &piLifecycleBudget{policy: log.policy}
	generation, err := recoverPiLifecycle(opCtx, rootFD, budget)
	if err != nil {
		return err
	}
	current, recordBytes, err := revalidatePiLifecycleWriter(log)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	envelopes, err := scanPiLifecycleEntries(opCtx, rootFD, budget, nil)
	if err != nil {
		return err
	}
	var committed, envelope int64
	for _, item := range envelopes {
		committed += item.LogBytes
		envelope += item.EnvelopeBytes
	}
	next := current
	next.CommittedBytes += int64(len(data))
	if next.CommittedBytes < current.CommittedBytes || next.CommittedRecords == ^uint64(0) {
		return piError("lifecycle_log_budget_exhausted", errors.New("lifecycle committed counters overflow"))
	}
	next.CommittedRecords++
	nextBytes, err := encodePiLifecycleControl(next)
	if err != nil {
		return err
	}
	committedAfter := committed + int64(len(data))
	envelopeAfter := envelope + int64(len(data)) + int64(len(nextBytes)) - recordBytes
	if committedAfter < committed || envelopeAfter < envelope || committedAfter > int64(log.policy.MaxBytes) || envelopeAfter > int64(log.policy.MaxEnvelopeBytes) {
		return piError("lifecycle_log_retention_refused", errors.New("append exceeds committed or managed-envelope byte policy"))
	}
	operation, err := beginPiLifecycleOperation(rootFD, generation, "append", log.entry, "", current.CommittedBytes, current.CommittedRecords, int64(len(data)))
	if err != nil {
		return err
	}
	if err := piLifecycleCheck(opCtx); err != nil {
		return err
	}
	if _, _, err := revalidatePiLifecycleWriter(log); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	if n, err := log.file.WriteAt(data, current.CommittedBytes); err != nil || n != len(data) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return piError("lifecycle_log_append_refused", err)
	}
	if err := log.file.Sync(); err != nil {
		return piError("lifecycle_log_append_refused", err)
	}
	if err := writePiLifecycleControlAtomic(log.entryFD, "record.json", nextBytes); err != nil {
		return err
	}
	if err := finishPiLifecycleOperation(rootFD, operation, 0, 0); err != nil {
		return err
	}
	log.record = next
	return nil
}

// close releases the active lock and descriptors regardless of maintenance
// availability, so a timed-out close can never pin an active reservation.
func (log *piSessionLog) close(ctx context.Context) error {
	if log == nil {
		return nil
	}
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.closed {
		return nil
	}
	log.closed = true
	result := log.appendErr
	opCtx, cancel := piLifecycleOperationContext(ctx, log.policy.CloseTimeoutSeconds)
	rootFD, foreground, err := openAndLockPiLifecycleRoot(opCtx, log.paths)
	if err == nil {
		retention, lockErr := acquirePiLifecycleLock(opCtx, rootFD, "retention.lock")
		if lockErr == nil {
			budget := &piLifecycleBudget{policy: log.policy}
			generation, recoveryErr := recoverPiLifecycle(opCtx, rootFD, budget)
			if recoveryErr == nil {
				current, _, validationErr := revalidatePiLifecycleWriter(log)
				if validationErr == nil {
					operation, beginErr := beginPiLifecycleOperation(rootFD, generation, "close", log.entry, "", current.CommittedBytes, current.CommittedRecords, 0)
					if beginErr == nil {
						current.ClosedAt = operation.StartedAt
						encoded, encodeErr := encodePiLifecycleControl(current)
						if encodeErr == nil {
							if syncErr := log.file.Sync(); syncErr == nil {
								if writeErr := writePiLifecycleControlAtomic(log.entryFD, "record.json", encoded); writeErr == nil {
									if finishErr := finishPiLifecycleOperation(rootFD, operation, 0, 0); finishErr != nil {
										result = finishErr
									} else {
										log.record = current
									}
								} else {
									result = writeErr
								}
							} else {
								result = syncErr
							}
						} else {
							result = encodeErr
						}
					} else {
						result = beginErr
					}
				} else {
					result = piError("lifecycle_log_evidence_unknown", validationErr)
				}
			} else {
				result = recoveryErr
			}
			_ = retention.Close()
		} else {
			result = lockErr
		}
		_ = foreground.Close()
		_ = unix.Close(rootFD)
	} else {
		result = err
	}
	cancel()
	if log.active != nil {
		_ = syscall.Flock(int(log.active.Fd()), syscall.LOCK_UN)
		if closeErr := log.active.Close(); result == nil {
			result = closeErr
		}
		log.active = nil
	}
	if log.file != nil {
		if closeErr := log.file.Close(); result == nil {
			result = closeErr
		}
		log.file = nil
	}
	if log.entryFD >= 0 {
		if closeErr := unix.Close(log.entryFD); result == nil {
			result = closeErr
		}
		log.entryFD = -1
	}
	return result
}

func openAndLockPiLifecycleRoot(ctx context.Context, paths PiStatePaths) (int, *os.File, error) {
	profileFD, err := openPiAggregateProfileRoot(paths)
	if err != nil {
		return -1, nil, piError("profile_state_path_invalid", err)
	}
	defer unix.Close(profileFD)
	rootFD, err := unix.Openat(profileFD, "lifecycle-logs", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	created := false
	if errors.Is(err, syscall.ENOENT) {
		mkErr := unix.Mkdirat(profileFD, "lifecycle-logs", 0o700)
		if mkErr != nil && !errors.Is(mkErr, syscall.EEXIST) {
			return -1, nil, piError("profile_state_path_invalid", mkErr)
		}
		created = mkErr == nil
		rootFD, err = unix.Openat(profileFD, "lifecycle-logs", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	}
	if err != nil {
		return -1, nil, piError("profile_state_path_invalid", err)
	}
	if _, err := validatePiLifecycleDirAt(profileFD, "lifecycle-logs", rootFD); err != nil {
		unix.Close(rootFD)
		return -1, nil, piError("lifecycle_log_evidence_unknown", err)
	}
	if err := ensurePiLifecycleLockFile(rootFD, "foreground.lock", created); err != nil {
		unix.Close(rootFD)
		return -1, nil, err
	}
	foreground, err := acquirePiLifecycleLock(ctx, rootFD, "foreground.lock")
	if err != nil {
		unix.Close(rootFD)
		return -1, nil, err
	}
	if err := initializePiLifecycleRoot(rootFD, created); err != nil {
		foreground.Close()
		unix.Close(rootFD)
		return -1, nil, err
	}
	return rootFD, foreground, nil
}

func initializePiLifecycleRoot(rootFD int, created bool) error {
	if err := ensurePiLifecycleLockFile(rootFD, "retention.lock", created); err != nil {
		return err
	}
	entriesFD, err := unix.Openat(rootFD, "entries", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) && created {
		if err = unix.Mkdirat(rootFD, "entries", 0o700); err == nil {
			entriesFD, err = unix.Openat(rootFD, "entries", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		}
	}
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", fmt.Errorf("entries directory: %w", err))
	}
	if _, err := validatePiLifecycleDirAt(rootFD, "entries", entriesFD); err != nil {
		unix.Close(entriesFD)
		return piError("lifecycle_log_evidence_unknown", err)
	}
	unix.Close(entriesFD)
	zeroAggregate := piLifecycleGeneration{SchemaVersion: piLifecycleSchemaVersion, Generation: 0, State: "even", Scope: "aggregate"}
	zeroLegacy := piLifecycleLegacyGeneration{SchemaVersion: piLifecycleSchemaVersion, Generation: 0, State: "even", Scope: "legacy"}
	for _, item := range []struct {
		name  string
		value any
	}{{"generation.json", zeroAggregate}, {"legacy-generation.json", zeroLegacy}} {
		fd, openErr := unix.Openat(rootFD, item.name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(openErr, syscall.ENOENT) && created {
			encoded, encodeErr := encodePiLifecycleControl(item.value)
			if encodeErr != nil {
				return encodeErr
			}
			if writeErr := writePiLifecycleControlAtomic(rootFD, item.name, encoded); writeErr != nil {
				return writeErr
			}
			continue
		}
		if openErr != nil {
			return piError("lifecycle_log_evidence_unknown", fmt.Errorf("%s: %w", item.name, openErr))
		}
		unix.Close(fd)
	}
	return nil
}

func ensurePiLifecycleLockFile(rootFD int, name string, create bool) error {
	flags := unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW
	if create {
		flags |= unix.O_CREAT | unix.O_EXCL
	}
	fd, err := unix.Openat(rootFD, name, flags, 0o600)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(fd)
	return validatePiLifecycleFile(fd, 0, true)
}

func acquirePiLifecycleLock(ctx context.Context, rootFD int, name string) (*os.File, error) {
	fd, err := unix.Openat(rootFD, name, unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	file := os.NewFile(uintptr(fd), name)
	if err := validatePiLifecycleFile(fd, 0, true); err != nil {
		file.Close()
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	for {
		if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return file, nil
		} else if !errors.Is(err, syscall.EWOULDBLOCK) {
			file.Close()
			return nil, piError("lifecycle_log_evidence_unknown", err)
		}
		select {
		case <-ctx.Done():
			file.Close()
			return nil, piError("lifecycle_log_lock_timeout", ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func validatePiLifecycleDir(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Mode&0o777 != 0o700 || st.Uid != uint32(os.Geteuid()) {
		return errors.New("lifecycle directory must be mode-0700 and owned by the trusted effective UID")
	}
	return nil
}

func validatePiLifecycleDirAt(parentFD int, name string, fd int) (unix.Stat_t, error) {
	var descriptor, path unix.Stat_t
	if err := validatePiLifecycleDir(fd); err != nil {
		return descriptor, err
	}
	if err := unix.Fstat(fd, &descriptor); err != nil {
		return descriptor, err
	}
	if err := unix.Fstatat(parentFD, name, &path, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return descriptor, err
	}
	if descriptor.Dev != path.Dev || descriptor.Ino != path.Ino || descriptor.Mode != path.Mode || descriptor.Uid != path.Uid {
		return descriptor, errors.New("lifecycle directory path identity changed")
	}
	return descriptor, nil
}

func piLifecycleIdentityFromStat(st unix.Stat_t) piLifecycleDirectoryIdentity {
	return piLifecycleDirectoryIdentity{Device: uint64(st.Dev), Inode: st.Ino, ChangeNsec: piLifecycleStatChangeNsec(st)}
}

func openPiLifecycleDirectoryAt(parentFD int, name string) (int, piLifecycleDirectoryIdentity, error) {
	fd, err := unix.Openat(parentFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, piLifecycleDirectoryIdentity{}, err
	}
	st, err := validatePiLifecycleDirAt(parentFD, name, fd)
	if err != nil {
		unix.Close(fd)
		return -1, piLifecycleDirectoryIdentity{}, err
	}
	return fd, piLifecycleIdentityFromStat(st), nil
}

func revalidatePiLifecycleDirectoryAt(parentFD int, name string, fd int, expected piLifecycleDirectoryIdentity) error {
	st, err := validatePiLifecycleDirAt(parentFD, name, fd)
	if err != nil {
		return err
	}
	if piLifecycleIdentityFromStat(st) != expected {
		return errors.New("lifecycle directory changed during bounded scan")
	}
	return nil
}

func piLifecycleContinuationMatchesDirectory(token piLifecycleContinuation, identity piLifecycleDirectoryIdentity) bool {
	return token.DirectoryDevice == identity.Device && token.DirectoryInode == identity.Inode && token.DirectoryChangeNsec == identity.ChangeNsec
}

func scanPiLifecycleDirectoryPage(ctx context.Context, fd int, cookie int64, budget *piLifecycleBudget, visit func(piLifecycleDirent) error) (int64, bool, error) {
	return scanPiLifecycleDirectoryPageSized(ctx, fd, cookie, budget, 1024, visit)
}

func scanPiLifecycleDirectoryPageSized(ctx context.Context, fd int, cookie int64, budget *piLifecycleBudget, bufferSize int, visit func(piLifecycleDirent) error) (int64, bool, error) {
	if cookie < 0 {
		return cookie, false, piError("lifecycle_log_evidence_unknown", errors.New("negative lifecycle directory cookie"))
	}
	readerFD, err := unix.Openat(fd, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return cookie, false, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(readerFD)
	if _, err := unix.Seek(readerFD, cookie, io.SeekStart); err != nil {
		return cookie, false, piError("lifecycle_log_evidence_unknown", err)
	}
	current := cookie
	for {
		if err := piLifecycleCheck(ctx); err != nil {
			return current, false, err
		}
		remaining := budget.policy.MaxScanEntries - budget.entries
		batch, complete, readErr := readPiLifecycleDirentBatch(readerFD, bufferSize)
		if readErr != nil {
			if errors.Is(readErr, syscall.EINVAL) && remaining < 43 {
				return current, false, piError("lifecycle_log_scan_exhausted", errors.New("max_scan_entries exhausted before the next directory record"))
			}
			return current, false, piError("lifecycle_log_evidence_unknown", readErr)
		}
		if complete {
			return current, true, nil
		}
		batchEnd, seekErr := unix.Seek(readerFD, 0, io.SeekCurrent)
		if seekErr != nil {
			return current, false, piError("lifecycle_log_evidence_unknown", seekErr)
		}
		for index := range batch {
			batch[index].Cookie = batchEnd
		}
		for range batch {
			if err := budget.chargeEntry(); err != nil {
				return current, false, err
			}
		}
		for _, entry := range batch {
			if err := visit(entry); err != nil {
				return current, false, err
			}
		}
		current = batchEnd
	}
}

func validatePiLifecycleFile(fd int, expectedSize int64, checkSize bool) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0o777 != 0o600 || st.Nlink != 1 || st.Uid != uint32(os.Geteuid()) {
		return errors.New("lifecycle file must be a mode-0600 single-link regular file owned by the trusted effective UID")
	}
	if checkSize && st.Size != expectedSize {
		return fmt.Errorf("lifecycle file size changed: got %d want %d", st.Size, expectedSize)
	}
	return nil
}

func openPiAggregateProfileRoot(paths PiStatePaths) (int, error) {
	rootFD, err := unix.Open(paths.CanonicalCacheRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, err
	}
	fd := rootFD
	for _, name := range []string{"agents-infra", "pi", paths.ProjectStateKey, paths.ProfileStateKey} {
		next, openErr := unix.Openat(fd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if fd != rootFD {
			unix.Close(fd)
		}
		if openErr != nil {
			unix.Close(rootFD)
			return -1, openErr
		}
		fd = next
	}
	unix.Close(rootFD)
	return fd, nil
}

func createPiLifecycleEntry(ctx context.Context, paths PiStatePaths, rootFD int, policy PiLifecycleLogRetention, budget *piLifecycleBudget, generation *piLifecycleGeneration) (*piSessionLog, error) {
	if err := budget.mutate(1); err != nil {
		return nil, err
	}
	var random [16]byte
	if _, err := io.ReadFull(piLifecycleRandom, random[:]); err != nil {
		return nil, piError("lifecycle_log_create_refused", err)
	}
	id := hex.EncodeToString(random[:])
	entry := piLifecycleNow().UTC().Format("20060102T150405.000000000Z") + "-" + id
	staging := ".creating-" + id
	operation, err := beginPiLifecycleOperation(rootFD, *generation, "create", entry, staging, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	entriesFD, err := unix.Openat(rootFD, "entries", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, piError("lifecycle_log_create_refused", err)
	}
	defer unix.Close(entriesFD)
	if err := unix.Mkdirat(entriesFD, staging, 0o700); err != nil {
		return nil, piError("lifecycle_log_create_refused", err)
	}
	stagingFD, err := unix.Openat(entriesFD, staging, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, piError("lifecycle_log_create_refused", err)
	}
	defer func() {
		if stagingFD >= 0 {
			unix.Close(stagingFD)
		}
	}()
	var dirStat unix.Stat_t
	if err := unix.Fstat(stagingFD, &dirStat); err != nil {
		return nil, piError("lifecycle_log_create_refused", err)
	}
	activeFD, err := unix.Openat(stagingFD, "active.lock", unix.O_CREAT|unix.O_EXCL|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, piError("lifecycle_log_create_refused", err)
	}
	active := os.NewFile(uintptr(activeFD), filepath.Join(paths.LifecycleLogsRoot, "entries", entry, "active.lock"))
	if err := validatePiLifecycleFile(activeFD, 0, true); err != nil {
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	if err := syscall.Flock(activeFD, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	logFD, err := unix.Openat(stagingFD, "log.jsonl", unix.O_CREAT|unix.O_EXCL|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	logFile := os.NewFile(uintptr(logFD), filepath.Join(paths.LifecycleLogsRoot, "entries", entry, "log.jsonl"))
	var logStat, activeStat unix.Stat_t
	if err := unix.Fstat(logFD, &logStat); err != nil {
		logFile.Close()
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	if err := unix.Fstat(activeFD, &activeStat); err != nil {
		logFile.Close()
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	record := piLifecycleRecord{
		SchemaVersion: piLifecycleSchemaVersion, EntryID: id,
		CreatedAt:       piLifecycleNow().UTC().Format(time.RFC3339Nano),
		DirectoryDevice: uint64(dirStat.Dev), DirectoryInode: dirStat.Ino,
		LogDevice: uint64(logStat.Dev), LogInode: logStat.Ino,
		ActiveDevice: uint64(activeStat.Dev), ActiveInode: activeStat.Ino,
	}
	encoded, err := encodePiLifecycleControl(record)
	if err != nil {
		logFile.Close()
		active.Close()
		return nil, err
	}
	if int64(len(encoded)) > int64(policy.MaxEnvelopeBytes) {
		logFile.Close()
		active.Close()
		return nil, piError("lifecycle_log_retention_refused", errors.New("new envelope exceeds max_envelope_bytes"))
	}
	if err := writePiLifecycleControlExclusive(stagingFD, "record.json", encoded); err != nil {
		logFile.Close()
		active.Close()
		return nil, err
	}
	if err := unix.Fsync(stagingFD); err != nil {
		logFile.Close()
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	if err := piLifecycleCheck(ctx); err != nil {
		logFile.Close()
		active.Close()
		return nil, err
	}
	if err := unix.Renameat(entriesFD, staging, entriesFD, entry); err != nil {
		logFile.Close()
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	if err := unix.Fsync(entriesFD); err != nil {
		logFile.Close()
		active.Close()
		return nil, piError("lifecycle_log_create_refused", err)
	}
	if err := finishPiLifecycleOperation(rootFD, operation, 0, 0); err != nil {
		logFile.Close()
		active.Close()
		return nil, err
	}
	stagingFD = -1
	entryFD, err := unix.Openat(entriesFD, entry, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		logFile.Close()
		active.Close()
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	return &piSessionLog{paths: paths, policy: policy, entry: entry, entryFD: entryFD, file: logFile, active: active, record: record, path: logFile.Name()}, nil
}

func revalidatePiLifecycleWriter(log *piSessionLog) (piLifecycleRecord, int64, error) {
	encoded, _, err := readPiLifecycleControl(log.entryFD, "record.json", nil)
	if err != nil {
		return piLifecycleRecord{}, 0, err
	}
	var record piLifecycleRecord
	if err := decodePiLifecycleControl(encoded, &record); err != nil {
		return record, 0, err
	}
	if err := validatePiLifecycleRecord(record, log.entry); err != nil {
		return record, 0, err
	}
	var dirStat, logStat, activeStat unix.Stat_t
	if err := unix.Fstat(log.entryFD, &dirStat); err != nil {
		return record, 0, err
	}
	if err := unix.Fstat(int(log.file.Fd()), &logStat); err != nil {
		return record, 0, err
	}
	if err := unix.Fstat(int(log.active.Fd()), &activeStat); err != nil {
		return record, 0, err
	}
	if uint64(dirStat.Dev) != record.DirectoryDevice || dirStat.Ino != record.DirectoryInode || uint64(logStat.Dev) != record.LogDevice || logStat.Ino != record.LogInode || uint64(activeStat.Dev) != record.ActiveDevice || activeStat.Ino != record.ActiveInode || logStat.Size != record.CommittedBytes {
		return record, 0, errors.New("lifecycle writer identity or committed boundary changed")
	}
	for _, proof := range []struct {
		name string
		st   unix.Stat_t
	}{{"log.jsonl", logStat}, {"active.lock", activeStat}} {
		var pathStat unix.Stat_t
		if err := unix.Fstatat(log.entryFD, proof.name, &pathStat, unix.AT_SYMLINK_NOFOLLOW); err != nil || pathStat.Dev != proof.st.Dev || pathStat.Ino != proof.st.Ino {
			if err == nil {
				err = errors.New("lifecycle writer path identity changed")
			}
			return record, 0, err
		}
	}
	return record, int64(len(encoded)), nil
}

func piLifecycleDeleteIdentityFromStat(st unix.Stat_t) piLifecycleDeleteIdentity {
	return piLifecycleDeleteIdentity{
		Device: uint64(st.Dev), Inode: st.Ino, Mode: uint32(st.Mode),
		UID: st.Uid, Links: uint64(st.Nlink),
	}
}

func piLifecycleDeleteIdentityMatchesStat(identity *piLifecycleDeleteIdentity, st unix.Stat_t) bool {
	return identity != nil && *identity == piLifecycleDeleteIdentityFromStat(st)
}

func capturePiLifecycleDeleteChild(parentFD int, name string) (*piLifecycleDeleteIdentity, error) {
	fd, err := unix.Openat(parentFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)
	if err := validatePiLifecycleFile(fd, 0, false); err != nil {
		return nil, err
	}
	var descriptor, path unix.Stat_t
	if err := unix.Fstat(fd, &descriptor); err != nil {
		return nil, err
	}
	if err := unix.Fstatat(parentFD, name, &path, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return nil, err
	}
	identity := piLifecycleDeleteIdentityFromStat(descriptor)
	if identity != piLifecycleDeleteIdentityFromStat(path) {
		return nil, errors.New("delete child path identity changed while authority was captured")
	}
	return &identity, nil
}

func capturePiLifecycleDeleteAuthority(rootFD int, entry, staging string) (*piLifecycleDeleteIdentity, *piLifecycleDeleteIdentity, *piLifecycleDeleteIdentity, *piLifecycleDeleteIdentity, error) {
	entriesFD, _, err := openPiLifecycleDirectoryAt(rootFD, "entries")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer unix.Close(entriesFD)
	entryFD := -1
	selected := ""
	for _, name := range []string{entry, staging} {
		fd, openErr := unix.Openat(entriesFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			continue
		}
		if openErr != nil {
			if entryFD >= 0 {
				unix.Close(entryFD)
			}
			return nil, nil, nil, nil, openErr
		}
		if entryFD >= 0 {
			unix.Close(fd)
			unix.Close(entryFD)
			return nil, nil, nil, nil, errors.New("delete source and tombstone both exist")
		}
		entryFD, selected = fd, name
	}
	if entryFD < 0 {
		return nil, nil, nil, nil, errors.New("delete source authority is absent")
	}
	defer unix.Close(entryFD)
	dirStat, err := validatePiLifecycleDirAt(entriesFD, selected, entryFD)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	names, err := readAllPiLifecycleNames(entryFD, nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "active.lock,log.jsonl,record.json" {
		return nil, nil, nil, nil, errors.New("delete source must contain the exact managed child set")
	}
	record, err := capturePiLifecycleDeleteChild(entryFD, "record.json")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	log, err := capturePiLifecycleDeleteChild(entryFD, "log.jsonl")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	active, err := capturePiLifecycleDeleteChild(entryFD, "active.lock")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	directory := piLifecycleDeleteIdentityFromStat(dirStat)
	return &directory, record, log, active, nil
}

func beginPiLifecycleOperation(rootFD int, current piLifecycleGeneration, kind, entry, staging string, committed int64, records uint64, appendBytes int64) (piLifecycleGeneration, error) {
	if current.State != "even" || current.Generation&1 != 0 || current.Generation > ^uint64(0)-2 {
		return piLifecycleGeneration{}, piError("lifecycle_log_evidence_unknown", errors.New("aggregate generation is not a usable even value"))
	}
	var nonce [16]byte
	if _, err := io.ReadFull(piLifecycleRandom, nonce[:]); err != nil {
		return piLifecycleGeneration{}, piError("lifecycle_log_operation_refused", err)
	}
	operation := piLifecycleGeneration{
		SchemaVersion: piLifecycleSchemaVersion, Generation: current.Generation + 1,
		State: "odd", Scope: "aggregate", OperationID: hex.EncodeToString(nonce[:]),
		OperationKind: kind, StartedAt: piLifecycleNow().UTC().Format(time.RFC3339Nano),
		EntryName: entry, StagingName: staging, CommittedBefore: committed,
		RecordsBefore: records, AppendBytes: appendBytes,
		Recovered: current.Recovered, Pruned: current.Pruned,
	}
	if kind == "delete" {
		deleteDir, deleteRecord, deleteLog, deleteActive, captureErr := capturePiLifecycleDeleteAuthority(rootFD, entry, staging)
		if captureErr != nil {
			return piLifecycleGeneration{}, piError("lifecycle_log_evidence_unknown", captureErr)
		}
		operation.DeleteDir, operation.DeleteRecord = deleteDir, deleteRecord
		operation.DeleteLog, operation.DeleteActive = deleteLog, deleteActive
	}
	if err := validatePiLifecycleGeneration(operation, "aggregate"); err != nil {
		return piLifecycleGeneration{}, piError("lifecycle_log_operation_refused", err)
	}
	encoded, err := encodePiLifecycleControl(operation)
	if err != nil {
		return piLifecycleGeneration{}, err
	}
	if err := writePiLifecycleControlAtomic(rootFD, "generation.json", encoded); err != nil {
		return piLifecycleGeneration{}, err
	}
	return operation, nil
}

func completedPiLifecycleGeneration(operation piLifecycleGeneration, recovered, pruned int) piLifecycleGeneration {
	operation.Generation++
	operation.State = "even"
	operation.OperationID, operation.OperationKind, operation.StartedAt = "", "", ""
	operation.EntryName, operation.StagingName = "", ""
	operation.DeleteDir, operation.DeleteRecord = nil, nil
	operation.DeleteLog, operation.DeleteActive = nil, nil
	operation.CommittedBefore, operation.RecordsBefore, operation.AppendBytes = 0, 0, 0
	operation.Recovered += recovered
	operation.Pruned += pruned
	return operation
}

func finishPiLifecycleOperation(rootFD int, operation piLifecycleGeneration, recovered, pruned int) error {
	operation = completedPiLifecycleGeneration(operation, recovered, pruned)
	encoded, err := encodePiLifecycleControl(operation)
	if err != nil {
		return err
	}
	return writePiLifecycleControlAtomic(rootFD, "generation.json", encoded)
}

func recoverPiLifecycle(ctx context.Context, rootFD int, budget *piLifecycleBudget) (piLifecycleGeneration, error) {
	encoded, _, err := readPiLifecycleControl(rootFD, "generation.json", budget)
	if err != nil {
		return piLifecycleGeneration{}, piError("lifecycle_log_evidence_unknown", err)
	}
	var generation piLifecycleGeneration
	if err := decodePiLifecycleControl(encoded, &generation); err != nil {
		return generation, piError("lifecycle_log_evidence_unknown", err)
	}
	if err := validatePiLifecycleGeneration(generation, "aggregate"); err != nil {
		return generation, piError("lifecycle_log_evidence_unknown", err)
	}
	generation, err = recoverPiLifecycleGenerationTemp(rootFD, budget, generation)
	if err != nil {
		return generation, err
	}
	if generation.State == "even" {
		return generation, nil
	}
	if _, err := scanPiLifecycleEntries(ctx, rootFD, budget, &generation); err != nil {
		return generation, piError("lifecycle_log_evidence_unknown", err)
	}
	switch generation.OperationKind {
	case "create":
		err = recoverPiLifecycleCreate(rootFD, budget, generation)
	case "append":
		err = recoverPiLifecycleAppend(rootFD, budget, generation)
	case "close":
		err = recoverPiLifecycleClose(rootFD, budget, generation)
	case "delete":
		err = recoverPiLifecycleDelete(rootFD, budget, generation)
	default:
		err = piError("lifecycle_log_evidence_unknown", errors.New("unrecognized odd-generation operation"))
	}
	if err != nil {
		return generation, err
	}
	if _, err := scanPiLifecycleEntries(ctx, rootFD, budget, nil); err != nil {
		return generation, piError("lifecycle_log_evidence_unknown", err)
	}
	if err := finishPiLifecycleOperation(rootFD, generation, 1, 0); err != nil {
		return generation, err
	}
	generation = completedPiLifecycleGeneration(generation, 1, 0)
	return generation, nil
}

func recoverPiLifecycleGenerationTemp(rootFD int, budget *piLifecycleBudget, current piLifecycleGeneration) (piLifecycleGeneration, error) {
	temp := piLifecycleAtomicTempName("generation.json")
	encoded, _, err := readPiLifecycleControl(rootFD, temp, budget)
	if errors.Is(err, syscall.ENOENT) {
		return current, nil
	}
	if err != nil {
		return current, piError("lifecycle_log_evidence_unknown", err)
	}
	var candidate piLifecycleGeneration
	if err := decodePiLifecycleControl(encoded, &candidate); err != nil {
		return current, piError("lifecycle_log_evidence_unknown", err)
	}
	if err := validatePiLifecycleGeneration(candidate, "aggregate"); err != nil {
		return current, piError("lifecycle_log_evidence_unknown", err)
	}
	if current.State == "even" {
		if current.Generation == ^uint64(0) || candidate.State != "odd" || candidate.Generation != current.Generation+1 || candidate.Recovered != current.Recovered || candidate.Pruned != current.Pruned {
			return current, piError("lifecycle_log_evidence_unknown", errors.New("aggregate generation temp is not the exact next odd operation"))
		}
		if err := budget.mutate(1); err != nil {
			return current, err
		}
		if err := unix.Renameat(rootFD, temp, rootFD, "generation.json"); err != nil {
			return current, piError("lifecycle_log_evidence_unknown", err)
		}
		if err := unix.Fsync(rootFD); err != nil {
			return current, piError("lifecycle_log_evidence_unknown", err)
		}
		return candidate, nil
	}
	originalPruned := 0
	if current.OperationKind == "delete" {
		originalPruned = 1
	}
	expected := completedPiLifecycleGeneration(current, 0, originalPruned)
	expectedBytes, encodeErr := encodePiLifecycleControl(expected)
	if encodeErr != nil || !bytes.Equal(encoded, expectedBytes) {
		return current, piError("lifecycle_log_evidence_unknown", errors.New("aggregate generation temp is not the exact operation completion"))
	}
	if err := budget.mutate(1); err != nil {
		return current, err
	}
	if err := unix.Unlinkat(rootFD, temp, 0); err != nil {
		return current, piError("lifecycle_log_evidence_unknown", err)
	}
	if err := unix.Fsync(rootFD); err != nil {
		return current, piError("lifecycle_log_evidence_unknown", err)
	}
	return current, nil
}

func recoverPiLifecycleCreate(rootFD int, budget *piLifecycleBudget, generation piLifecycleGeneration) error {
	entriesFD, err := unix.Openat(rootFD, "entries", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entriesFD)
	finalFD, finalErr := unix.Openat(entriesFD, generation.EntryName, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if finalErr == nil {
		unix.Close(finalFD)
		return nil
	}
	if !errors.Is(finalErr, syscall.ENOENT) {
		return piError("lifecycle_log_evidence_unknown", finalErr)
	}
	stagingFD, stagingErr := unix.Openat(entriesFD, generation.StagingName, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(stagingErr, syscall.ENOENT) {
		return nil
	}
	if stagingErr != nil {
		return piError("lifecycle_log_evidence_unknown", stagingErr)
	}
	defer unix.Close(stagingFD)
	names, err := readAllPiLifecycleNames(stagingFD, budget)
	if err != nil {
		return err
	}
	sort.Strings(names)
	if strings.Join(names, ",") == "active.lock,log.jsonl,record.json" {
		if _, err := scanOnePiLifecycleEnvelope(entriesFD, generation.StagingName, budget, true, nil); err != nil {
			return err
		}
		if err := budget.mutate(1); err != nil {
			return err
		}
		if err := unix.Renameat(entriesFD, generation.StagingName, entriesFD, generation.EntryName); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		return unix.Fsync(entriesFD)
	}
	for _, name := range names {
		if name != "record.json" && name != "active.lock" && name != "log.jsonl" {
			return piError("lifecycle_log_evidence_unknown", errors.New("create partial contains foreign evidence"))
		}
	}
	for _, name := range []string{"record.json", "log.jsonl", "active.lock"} {
		if piLifecycleContainsString(names, name) {
			if err := budget.mutate(1); err != nil {
				return err
			}
			if err := unix.Unlinkat(stagingFD, name, 0); err != nil {
				return piError("lifecycle_log_evidence_unknown", err)
			}
		}
	}
	if err := budget.mutate(1); err != nil {
		return err
	}
	if err := unix.Unlinkat(entriesFD, generation.StagingName, unix.AT_REMOVEDIR); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	return nil
}

func recoverPiLifecycleAppend(rootFD int, budget *piLifecycleBudget, generation piLifecycleGeneration) error {
	entriesFD, entryFD, err := openPiLifecycleEntry(rootFD, generation.EntryName)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entriesFD)
	defer unix.Close(entryFD)
	encoded, _, err := readPiLifecycleControl(entryFD, "record.json", budget)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	var record piLifecycleRecord
	if err := decodePiLifecycleControl(encoded, &record); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	tempName := piLifecycleAtomicTempName("record.json")
	tempBytes, _, tempErr := readPiLifecycleControl(entryFD, tempName, budget)
	hasTemp := tempErr == nil
	if tempErr != nil && !errors.Is(tempErr, syscall.ENOENT) {
		return piError("lifecycle_log_evidence_unknown", tempErr)
	}
	var tempRecord piLifecycleRecord
	if hasTemp {
		if err := decodePiLifecycleControl(tempBytes, &tempRecord); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
	}
	logFD, err := unix.Openat(entryFD, "log.jsonl", unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(logFD)
	var st unix.Stat_t
	if err := unix.Fstat(logFD, &st); err != nil || uint64(st.Dev) != record.LogDevice || st.Ino != record.LogInode {
		return piError("lifecycle_log_evidence_unknown", errors.New("append recovery log identity changed"))
	}
	wantAfter := generation.CommittedBefore + generation.AppendBytes
	if record.CommittedBytes == wantAfter && record.CommittedRecords == generation.RecordsBefore+1 && st.Size == wantAfter {
		if hasTemp {
			return piError("lifecycle_log_evidence_unknown", errors.New("committed append retained an impossible record temp"))
		}
		return nil
	}
	if record.CommittedBytes != generation.CommittedBefore || record.CommittedRecords != generation.RecordsBefore || st.Size < generation.CommittedBefore || st.Size > wantAfter {
		return piError("lifecycle_log_evidence_unknown", errors.New("append recovery boundary is ambiguous"))
	}
	if hasTemp {
		expected := record
		expected.CommittedBytes = wantAfter
		expected.CommittedRecords = generation.RecordsBefore + 1
		if tempRecord != expected || st.Size != wantAfter {
			return piError("lifecycle_log_evidence_unknown", errors.New("append record temp is not the exact operation result"))
		}
		if err := budget.mutate(1); err != nil {
			return err
		}
		if err := unix.Renameat(entryFD, tempName, entryFD, "record.json"); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		return unix.Fsync(entryFD)
	}
	if st.Size > generation.CommittedBefore {
		if err := budget.mutate(1); err != nil {
			return err
		}
		if err := unix.Ftruncate(logFD, generation.CommittedBefore); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		return unix.Fsync(logFD)
	}
	return nil
}

func recoverPiLifecycleClose(rootFD int, budget *piLifecycleBudget, generation piLifecycleGeneration) error {
	entriesFD, entryFD, err := openPiLifecycleEntry(rootFD, generation.EntryName)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entriesFD)
	defer unix.Close(entryFD)
	encoded, _, err := readPiLifecycleControl(entryFD, "record.json", budget)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	var record piLifecycleRecord
	if err := decodePiLifecycleControl(encoded, &record); err != nil || validatePiLifecycleRecord(record, generation.EntryName) != nil || record.CommittedBytes != generation.CommittedBefore || record.CommittedRecords != generation.RecordsBefore {
		return piError("lifecycle_log_evidence_unknown", errors.New("close recovery record is ambiguous"))
	}
	tempName := piLifecycleAtomicTempName("record.json")
	tempBytes, _, tempErr := readPiLifecycleControl(entryFD, tempName, budget)
	if errors.Is(tempErr, syscall.ENOENT) {
		if record.ClosedAt == "" || record.ClosedAt == generation.StartedAt {
			return nil
		}
		return piError("lifecycle_log_evidence_unknown", errors.New("close recovery record does not prove this operation"))
	}
	if tempErr != nil {
		return piError("lifecycle_log_evidence_unknown", tempErr)
	}
	var tempRecord piLifecycleRecord
	if err := decodePiLifecycleControl(tempBytes, &tempRecord); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	expected := record
	expected.ClosedAt = generation.StartedAt
	if tempRecord != expected || record.ClosedAt != "" {
		return piError("lifecycle_log_evidence_unknown", errors.New("close record temp is not the exact operation result"))
	}
	if err := budget.mutate(1); err != nil {
		return err
	}
	if err := unix.Renameat(entryFD, tempName, entryFD, "record.json"); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	return unix.Fsync(entryFD)
}

func revalidatePiLifecycleDeleteDirectory(entriesFD int, tombstone string, entryFD int, expected *piLifecycleDeleteIdentity) error {
	descriptor, err := validatePiLifecycleDirAt(entriesFD, tombstone, entryFD)
	if err != nil {
		return err
	}
	observed := piLifecycleDeleteIdentityFromStat(descriptor)
	if expected == nil || expected.Device != observed.Device || expected.Inode != observed.Inode || expected.Mode != observed.Mode || expected.UID != observed.UID {
		return fmt.Errorf("delete tombstone authority changed: expected=%+v observed=%+v", expected, observed)
	}
	return nil
}

func openAndValidatePiLifecycleDeleteChild(entryFD int, name string, expected *piLifecycleDeleteIdentity) (int, bool, error) {
	fd, err := unix.Openat(entryFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return -1, false, nil
	}
	if err != nil {
		return -1, false, err
	}
	if err := revalidatePiLifecycleDeleteChild(entryFD, name, fd, expected); err != nil {
		unix.Close(fd)
		return -1, false, err
	}
	return fd, true, nil
}

func revalidatePiLifecycleDeleteChild(entryFD int, name string, childFD int, expected *piLifecycleDeleteIdentity) error {
	if err := validatePiLifecycleFile(childFD, 0, false); err != nil {
		return err
	}
	var descriptor, path unix.Stat_t
	if err := unix.Fstat(childFD, &descriptor); err != nil {
		return err
	}
	if err := unix.Fstatat(entryFD, name, &path, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if !piLifecycleDeleteIdentityMatchesStat(expected, descriptor) || !piLifecycleDeleteIdentityMatchesStat(expected, path) {
		return errors.New("delete child authority changed immediately before unlink")
	}
	return nil
}

func recoverPiLifecycleDelete(rootFD int, budget *piLifecycleBudget, generation piLifecycleGeneration) error {
	entriesFD, _, err := openPiLifecycleDirectoryAt(rootFD, "entries")
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entriesFD)
	entryFD, err := unix.Openat(entriesFD, generation.StagingName, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil
	}
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entryFD)
	if err := revalidatePiLifecycleDeleteDirectory(entriesFD, generation.StagingName, entryFD, generation.DeleteDir); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	names, err := readAllPiLifecycleNames(entryFD, budget)
	if err != nil {
		return err
	}
	for _, name := range names {
		if name != "record.json" && name != "active.lock" && name != "log.jsonl" {
			return piError("lifecycle_log_evidence_unknown", errors.New("delete tombstone contains foreign evidence"))
		}
	}
	for _, child := range []struct {
		name     string
		expected *piLifecycleDeleteIdentity
	}{{"record.json", generation.DeleteRecord}, {"log.jsonl", generation.DeleteLog}, {"active.lock", generation.DeleteActive}} {
		childFD, present, err := openAndValidatePiLifecycleDeleteChild(entryFD, child.name, child.expected)
		if err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		if !present {
			continue
		}
		if err := budget.mutate(1); err != nil {
			unix.Close(childFD)
			return err
		}
		if err := revalidatePiLifecycleDeleteDirectory(entriesFD, generation.StagingName, entryFD, generation.DeleteDir); err != nil {
			unix.Close(childFD)
			return piError("lifecycle_log_evidence_unknown", err)
		}
		if piLifecycleBeforeDeleteChildUnlink != nil {
			if err := piLifecycleBeforeDeleteChildUnlink(entryFD, child.name); err != nil {
				unix.Close(childFD)
				return piError("lifecycle_log_evidence_unknown", err)
			}
		}
		if err := revalidatePiLifecycleDeleteChild(entryFD, child.name, childFD, child.expected); err != nil {
			unix.Close(childFD)
			return piError("lifecycle_log_evidence_unknown", err)
		}
		if err := unix.Unlinkat(entryFD, child.name, 0); err != nil {
			unix.Close(childFD)
			return piError("lifecycle_log_evidence_unknown", err)
		}
		unix.Close(childFD)
	}
	if err := budget.mutate(1); err != nil {
		return err
	}
	if err := revalidatePiLifecycleDeleteDirectory(entriesFD, generation.StagingName, entryFD, generation.DeleteDir); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	if err := unix.Unlinkat(entriesFD, generation.StagingName, unix.AT_REMOVEDIR); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	return nil
}

func prunePiLifecycleForCreate(ctx context.Context, rootFD int, policy PiLifecycleLogRetention, budget *piLifecycleBudget, generation *piLifecycleGeneration, envelopes []piLifecycleEnvelope) error {
	sort.Slice(envelopes, func(i, j int) bool { return envelopes[i].Name > envelopes[j].Name })
	var activeCount int
	var activeCommitted, activeEnvelope int64
	for _, item := range envelopes {
		if item.Active {
			activeCount++
			activeCommitted += item.LogBytes
			activeEnvelope += item.EnvelopeBytes
		}
	}
	if activeCount >= policy.MaxCount || activeCommitted > int64(policy.MaxBytes) || activeEnvelope+piLifecycleControlLimit > int64(policy.MaxEnvelopeBytes) {
		return piError("lifecycle_log_retention_refused", errors.New("active lifecycle reservations exhaust retention policy"))
	}
	remainingCount := policy.MaxCount - activeCount - 1
	remainingBytes := int64(policy.MaxBytes) - activeCommitted
	remainingEnvelope := int64(policy.MaxEnvelopeBytes) - activeEnvelope - piLifecycleControlLimit
	cutoff := piLifecycleNow().Add(-time.Duration(policy.MaxAgeSeconds) * time.Second)
	var remove []piLifecycleEnvelope
	for _, item := range envelopes {
		if item.Active {
			continue
		}
		created, parseErr := time.Parse(time.RFC3339Nano, item.Record.CreatedAt)
		keep := parseErr == nil && !created.Before(cutoff) && remainingCount > 0 && item.LogBytes <= remainingBytes && item.EnvelopeBytes <= remainingEnvelope
		if keep {
			remainingCount--
			remainingBytes -= item.LogBytes
			remainingEnvelope -= item.EnvelopeBytes
		} else {
			remove = append(remove, item)
		}
	}
	if len(remove) > policy.MaxMutationsPerOperation {
		return piError("lifecycle_log_mutation_exhausted", errors.New("deterministic prune exceeds max_mutations_per_operation"))
	}
	for _, item := range remove {
		if err := piLifecycleCheck(ctx); err != nil {
			return err
		}
		tombstone := ".deleting-" + item.Name[strings.LastIndexByte(item.Name, '-')+1:]
		operation, err := beginPiLifecycleOperation(rootFD, *generation, "delete", item.Name, tombstone, 0, 0, 0)
		if err != nil {
			return err
		}
		entriesFD, entryFD, err := openPiLifecycleEntry(rootFD, item.Name)
		if err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		activeFD, err := unix.Openat(entryFD, "active.lock", unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if err == nil {
			err = syscall.Flock(activeFD, syscall.LOCK_EX|syscall.LOCK_NB)
		}
		if err != nil {
			if activeFD >= 0 {
				unix.Close(activeFD)
			}
			unix.Close(entryFD)
			unix.Close(entriesFD)
			return piError("lifecycle_log_retention_refused", errors.New("inactive pruning candidate became active or unreadable"))
		}
		_ = syscall.Flock(activeFD, syscall.LOCK_UN)
		unix.Close(activeFD)
		unix.Close(entryFD)
		if err := budget.mutate(1); err != nil {
			unix.Close(entriesFD)
			return err
		}
		if err := unix.Renameat(entriesFD, item.Name, entriesFD, tombstone); err != nil {
			unix.Close(entriesFD)
			return piError("lifecycle_log_evidence_unknown", err)
		}
		unix.Close(entriesFD)
		if err := recoverPiLifecycleDelete(rootFD, budget, operation); err != nil {
			return err
		}
		if err := finishPiLifecycleOperation(rootFD, operation, 0, 1); err != nil {
			return err
		}
		generation.Generation = operation.Generation + 1
		generation.State = "even"
	}
	return nil
}

func scanPiLifecycleEntries(ctx context.Context, rootFD int, budget *piLifecycleBudget, recovery *piLifecycleGeneration) ([]piLifecycleEnvelope, error) {
	if err := piLifecycleCheck(ctx); err != nil {
		return nil, err
	}
	if err := validatePiLifecycleDir(rootFD); err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	rootNames, err := readAllPiLifecycleNames(rootFD, budget)
	if err != nil {
		return nil, err
	}
	knownRoot := map[string]bool{"foreground.lock": true, "retention.lock": true, "generation.json": true, "legacy-generation.json": true, "entries": true}
	for _, name := range rootNames {
		if !knownRoot[name] {
			return nil, piError("lifecycle_log_evidence_unknown", fmt.Errorf("foreign aggregate-root entry %q", name))
		}
	}
	if len(rootNames) != len(knownRoot) {
		return nil, piError("lifecycle_log_evidence_unknown", errors.New("aggregate root is missing a fixed entry"))
	}
	entriesFD, err := unix.Openat(rootFD, "entries", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entriesFD)
	if _, err := validatePiLifecycleDirAt(rootFD, "entries", entriesFD); err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	names, err := readAllPiLifecycleNames(entriesFD, budget)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	result := make([]piLifecycleEnvelope, 0, len(names))
	for _, name := range names {
		if err := piLifecycleCheck(ctx); err != nil {
			return nil, err
		}
		managed := piLifecycleEntryName.MatchString(name)
		partial := strings.HasPrefix(name, ".creating-") || strings.HasPrefix(name, ".deleting-")
		if !managed && !(recovery != nil && partial && name == recovery.StagingName) {
			return nil, piError("lifecycle_log_evidence_unknown", fmt.Errorf("unproven aggregate entry %q", name))
		}
		if partial {
			continue
		}
		envelope, err := scanOnePiLifecycleEnvelope(entriesFD, name, budget, false, recovery)
		if err != nil {
			return nil, err
		}
		result = append(result, envelope)
	}
	return result, nil
}

func scanOnePiLifecycleEnvelope(entriesFD int, name string, budget *piLifecycleBudget, staging bool, recovery *piLifecycleGeneration) (piLifecycleEnvelope, error) {
	entryFD, err := unix.Openat(entriesFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entryFD)
	if err := validatePiLifecycleDir(entryFD); err != nil {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	names, err := readAllPiLifecycleNames(entryFD, budget)
	if err != nil {
		return piLifecycleEnvelope{}, err
	}
	sort.Strings(names)
	strictChildren := strings.Join(names, ",") == "active.lock,log.jsonl,record.json"
	tempChildren := strings.Join(names, ",") == ".record.json.tmp,active.lock,log.jsonl,record.json"
	allowRecordTemp := recovery != nil && (recovery.OperationKind == "append" || recovery.OperationKind == "close") && recovery.EntryName == name
	if !strictChildren && !(allowRecordTemp && tempChildren) {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", errors.New("managed envelope must contain exactly record.json, active.lock, and log.jsonl"))
	}
	if tempChildren {
		if _, _, err := readPiLifecycleControl(entryFD, piLifecycleAtomicTempName("record.json"), budget); err != nil {
			if piErrorIs(err, "lifecycle_log_scan_exhausted") {
				return piLifecycleEnvelope{}, err
			}
			return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
		}
	}
	encoded, recordStat, err := readPiLifecycleControl(entryFD, "record.json", budget)
	if err != nil {
		if piErrorIs(err, "lifecycle_log_scan_exhausted") {
			return piLifecycleEnvelope{}, err
		}
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	var record piLifecycleRecord
	if err := decodePiLifecycleControl(encoded, &record); err != nil {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	validationName := name
	if staging {
		validationName = "00000000T000000.000000000Z-" + strings.TrimPrefix(name, ".creating-")
	}
	if err := validatePiLifecycleRecord(record, validationName); err != nil {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	var dirStat unix.Stat_t
	if err := unix.Fstat(entryFD, &dirStat); err != nil || uint64(dirStat.Dev) != record.DirectoryDevice || dirStat.Ino != record.DirectoryInode {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", errors.New("managed envelope directory identity changed"))
	}
	logFD, err := unix.Openat(entryFD, "log.jsonl", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(logFD)
	activeFD, err := unix.Openat(entryFD, "active.lock", unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(activeFD)
	var logStat, activeStat unix.Stat_t
	if err := unix.Fstat(logFD, &logStat); err != nil {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
	}
	allowAppendPartial := recovery != nil && recovery.OperationKind == "append" && recovery.EntryName == name && record.CommittedBytes == recovery.CommittedBefore && logStat.Size >= recovery.CommittedBefore && logStat.Size <= recovery.CommittedBefore+recovery.AppendBytes
	if validatePiLifecycleFile(logFD, record.CommittedBytes, !allowAppendPartial) != nil || uint64(logStat.Dev) != record.LogDevice || logStat.Ino != record.LogInode {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", errors.New("managed log identity or committed size changed"))
	}
	if err := unix.Fstat(activeFD, &activeStat); err != nil || validatePiLifecycleFile(activeFD, 0, true) != nil || uint64(activeStat.Dev) != record.ActiveDevice || activeStat.Ino != record.ActiveInode {
		return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", errors.New("managed active-lock identity changed"))
	}
	active := false
	if err := syscall.Flock(activeFD, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return piLifecycleEnvelope{}, piError("lifecycle_log_evidence_unknown", err)
		}
		active = true
	} else {
		_ = syscall.Flock(activeFD, syscall.LOCK_UN)
	}
	return piLifecycleEnvelope{Name: name, Record: record, RecordBytes: recordStat.Size, LogBytes: logStat.Size, EnvelopeBytes: recordStat.Size + logStat.Size + activeStat.Size, Active: active}, nil
}

func validatePiLifecycleRecord(record piLifecycleRecord, name string) error {
	if record.SchemaVersion != piLifecycleSchemaVersion || len(record.EntryID) != 32 || name != "" && !strings.HasSuffix(name, "-"+record.EntryID) {
		return errors.New("lifecycle record identity is invalid")
	}
	if _, err := hex.DecodeString(record.EntryID); err != nil || record.CommittedBytes < 0 {
		return errors.New("lifecycle record counters or random ID are invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, record.CreatedAt); err != nil {
		return errors.New("lifecycle record created_at is invalid")
	}
	if record.ClosedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, record.ClosedAt); err != nil {
			return errors.New("lifecycle record closed_at is invalid")
		}
	}
	return nil
}

func validatePiLifecycleStoredDeleteIdentity(identity *piLifecycleDeleteIdentity, directory bool) error {
	if identity == nil || identity.Device == 0 || identity.Inode == 0 || identity.UID != uint32(os.Geteuid()) || identity.Links == 0 {
		return errors.New("delete lifecycle identity is absent or untrusted")
	}
	wantType, wantMode := uint32(unix.S_IFREG), uint32(0o600)
	if directory {
		wantType, wantMode = uint32(unix.S_IFDIR), uint32(0o700)
	}
	if identity.Mode&uint32(unix.S_IFMT) != wantType || identity.Mode&0o777 != wantMode {
		return errors.New("delete lifecycle identity has the wrong type or mode")
	}
	if !directory && identity.Links != 1 {
		return errors.New("delete lifecycle child is not a single-link regular file")
	}
	return nil
}

func piLifecycleGenerationHasDeleteAuthority(generation piLifecycleGeneration) bool {
	return generation.DeleteDir != nil || generation.DeleteRecord != nil || generation.DeleteLog != nil || generation.DeleteActive != nil
}

func piLifecycleGenerationEntryID(entry string) (string, bool) {
	if !piLifecycleEntryName.MatchString(entry) {
		return "", false
	}
	separator := strings.LastIndexByte(entry, '-')
	if separator < 0 {
		return "", false
	}
	return entry[separator+1:], true
}

func validatePiLifecycleGeneration(generation piLifecycleGeneration, scope string) error {
	if generation.SchemaVersion != piLifecycleSchemaVersion || generation.Scope != scope || generation.State != "even" && generation.State != "odd" || (generation.State == "even") != (generation.Generation&1 == 0) {
		return errors.New("lifecycle generation schema, scope, state, or parity is invalid")
	}
	if generation.Recovered < 0 || generation.Pruned < 0 || generation.CommittedBefore < 0 || generation.AppendBytes < 0 {
		return errors.New("lifecycle generation counters are invalid")
	}
	if generation.State == "even" {
		if generation.OperationID != "" || generation.OperationKind != "" || generation.StartedAt != "" || generation.EntryName != "" || generation.StagingName != "" || generation.CommittedBefore != 0 || generation.RecordsBefore != 0 || generation.AppendBytes != 0 || piLifecycleGenerationHasDeleteAuthority(generation) {
			return errors.New("even lifecycle generation carries operation authority")
		}
		return nil
	}
	if scope != "aggregate" {
		return errors.New("only the aggregate lifecycle generation may be odd")
	}
	if len(generation.OperationID) != 32 {
		return errors.New("odd lifecycle generation operation ID is invalid")
	}
	if _, err := hex.DecodeString(generation.OperationID); err != nil {
		return errors.New("odd lifecycle generation operation ID is invalid")
	}
	started, err := time.Parse(time.RFC3339Nano, generation.StartedAt)
	if err != nil || started.UTC().Format(time.RFC3339Nano) != generation.StartedAt {
		return errors.New("odd lifecycle generation started_at is invalid")
	}
	entryID, ok := piLifecycleGenerationEntryID(generation.EntryName)
	if !ok {
		return errors.New("odd lifecycle generation entry name is invalid")
	}
	withoutDeleteAuthority := func() error {
		if piLifecycleGenerationHasDeleteAuthority(generation) {
			return errors.New("non-delete lifecycle generation carries delete authority")
		}
		return nil
	}
	switch generation.OperationKind {
	case "create":
		if generation.StagingName != ".creating-"+entryID || generation.CommittedBefore != 0 || generation.RecordsBefore != 0 || generation.AppendBytes != 0 {
			return errors.New("create lifecycle generation fields are invalid")
		}
		return withoutDeleteAuthority()
	case "append":
		if generation.StagingName != "" || generation.AppendBytes <= 0 {
			return errors.New("append lifecycle generation fields are invalid")
		}
		return withoutDeleteAuthority()
	case "close":
		if generation.StagingName != "" || generation.AppendBytes != 0 {
			return errors.New("close lifecycle generation fields are invalid")
		}
		return withoutDeleteAuthority()
	case "delete":
		if generation.StagingName != ".deleting-"+entryID || generation.CommittedBefore != 0 || generation.RecordsBefore != 0 || generation.AppendBytes != 0 {
			return errors.New("delete lifecycle generation fields are invalid")
		}
		if err := validatePiLifecycleStoredDeleteIdentity(generation.DeleteDir, true); err != nil {
			return err
		}
		for _, identity := range []*piLifecycleDeleteIdentity{generation.DeleteRecord, generation.DeleteLog, generation.DeleteActive} {
			if err := validatePiLifecycleStoredDeleteIdentity(identity, false); err != nil {
				return err
			}
		}
		return nil
	default:
		return errors.New("odd lifecycle generation operation kind is invalid")
	}
}

func readAllPiLifecycleNames(fd int, budget *piLifecycleBudget) ([]string, error) {
	dup, err := unix.Openat(fd, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	file := os.NewFile(uintptr(dup), "lifecycle-directory")
	defer file.Close()
	var names []string
	for {
		batch, readErr := file.Readdirnames(1)
		for _, name := range batch {
			if budget != nil {
				if err := budget.chargeEntry(); err != nil {
					return nil, err
				}
			}
			names = append(names, name)
		}
		if errors.Is(readErr, io.EOF) {
			return names, nil
		}
		if readErr != nil {
			return nil, piError("lifecycle_log_evidence_unknown", readErr)
		}
	}
}

func readPiLifecycleControl(dirFD int, name string, budget *piLifecycleBudget) ([]byte, unix.Stat_t, error) {
	fd, err := unix.Openat(dirFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, unix.Stat_t{}, err
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	var before, after, pathStat unix.Stat_t
	if err := unix.Fstat(fd, &before); err != nil {
		return nil, before, err
	}
	if err := validatePiLifecycleFile(fd, 0, false); err != nil {
		return nil, before, err
	}
	content, err := io.ReadAll(io.LimitReader(file, piLifecycleControlLimit+1))
	if budget != nil {
		if chargeErr := budget.chargeControl(len(content)); chargeErr != nil {
			return nil, before, chargeErr
		}
	}
	if err != nil {
		return nil, before, err
	}
	if len(content) > piLifecycleControlLimit {
		return nil, before, errors.New("control file exceeds 4096 bytes")
	}
	if err := unix.Fstat(fd, &after); err != nil || before.Dev != after.Dev || before.Ino != after.Ino || before.Size != after.Size {
		return nil, before, errors.New("control file changed while read")
	}
	if err := unix.Fstatat(dirFD, name, &pathStat, unix.AT_SYMLINK_NOFOLLOW); err != nil || pathStat.Dev != after.Dev || pathStat.Ino != after.Ino {
		return nil, before, errors.New("control file path identity changed while read")
	}
	return content, before, nil
}

func encodePiLifecycleControl(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	encoded = append(encoded, '\n')
	if len(encoded) > piLifecycleControlLimit {
		return nil, piError("lifecycle_log_control_oversize", errors.New("control document exceeds 4096 bytes"))
	}
	return encoded, nil
}

func decodePiLifecycleControl(content []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("control document has trailing JSON")
	}
	return nil
}

func writePiLifecycleControlExclusive(dirFD int, name string, content []byte) error {
	fd, err := unix.Openat(dirFD, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	file := os.NewFile(uintptr(fd), name)
	if n, err := file.Write(content); err != nil || n != len(content) {
		file.Close()
		if err == nil {
			err = io.ErrShortWrite
		}
		return piError("lifecycle_log_evidence_unknown", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return piError("lifecycle_log_evidence_unknown", err)
	}
	if err := file.Close(); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	return unix.Fsync(dirFD)
}

func writePiLifecycleControlAtomic(dirFD int, name string, content []byte) error {
	temp := piLifecycleAtomicTempName(name)
	if err := writePiLifecycleControlExclusive(dirFD, temp, content); err != nil {
		return err
	}
	if err := unix.Renameat(dirFD, temp, dirFD, name); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	if err := unix.Fsync(dirFD); err != nil {
		return piError("lifecycle_log_evidence_unknown", err)
	}
	return nil
}

func piLifecycleAtomicTempName(name string) string {
	return "." + name + ".tmp"
}

func openPiLifecycleEntry(rootFD int, name string) (int, int, error) {
	entriesFD, err := unix.Openat(rootFD, "entries", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, -1, err
	}
	entryFD, err := unix.Openat(entriesFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		unix.Close(entriesFD)
		return -1, -1, err
	}
	return entriesFD, entryFD, nil
}

func (budget *piLifecycleBudget) chargeEntry() error {
	budget.entries++
	if budget.entries > budget.policy.MaxScanEntries {
		return piError("lifecycle_log_scan_exhausted", errors.New("max_scan_entries exhausted"))
	}
	return nil
}

func (budget *piLifecycleBudget) chargeControl(count int) error {
	limit := budget.controlLimit
	if limit == 0 {
		limit = budget.policy.MaxScanControlBytes
	}
	if count < 0 || limit < count || budget.controlBytes > limit-count {
		return piError("lifecycle_log_scan_exhausted", errors.New("max_scan_control_bytes exhausted"))
	}
	budget.controlBytes += count
	return nil
}

func (budget *piLifecycleBudget) mutate(count int) error {
	if count < 0 || budget.mutations > budget.policy.MaxMutationsPerOperation-count {
		return piError("lifecycle_log_mutation_exhausted", errors.New("max_mutations_per_operation exhausted"))
	}
	budget.mutations += count
	return nil
}

func piLifecycleOperationContext(parent context.Context, seconds int) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, time.Duration(seconds)*time.Second)
}

func piLifecycleCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return piError("lifecycle_log_deadline_exceeded", ctx.Err())
	default:
		return nil
	}
}

func piProcessIdentityFields(pid int) map[string]any {
	fields := map[string]any{"pid": pid}
	if pgid, err := syscall.Getpgid(pid); err == nil {
		fields["pgid"] = pgid
	}
	return fields
}

func piLifecycleContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func piLifecyclePolicyDigest(policy PiLifecycleLogRetention) string {
	encoded, _ := json.Marshal(policy)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func encodePiLifecycleContinuation(token piLifecycleContinuation) string {
	encoded, _ := json.Marshal(token)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodePiLifecycleContinuation(encoded string) (piLifecycleContinuation, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(data) > piLifecycleControlLimit {
		return piLifecycleContinuation{}, errors.New("invalid lifecycle continuation encoding")
	}
	var token piLifecycleContinuation
	if err := decodePiLifecycleControl(data, &token); err != nil {
		return token, err
	}
	if token.SchemaVersion != piLifecycleSchemaVersion || token.Offset < 0 {
		return token, errors.New("invalid lifecycle continuation schema")
	}
	return token, nil
}

func piLifecycleDirectoryCursor(phase string, cookie int64, identity piLifecycleDirectoryIdentity) piLifecycleContinuation {
	return piLifecycleContinuation{
		SchemaVersion: piLifecycleSchemaVersion,
		Phase:         phase, Offset: cookie,
		DirectoryDevice: identity.Device, DirectoryInode: identity.Inode,
		DirectoryChangeNsec: identity.ChangeNsec,
	}
}

func scanPiLifecycleEntriesPage(ctx context.Context, rootFD int, budget *piLifecycleBudget, token *piLifecycleContinuation) ([]piLifecycleEnvelope, *piLifecycleContinuation, error) {
	if err := validatePiLifecycleDir(rootFD); err != nil {
		return nil, nil, piError("lifecycle_log_evidence_unknown", err)
	}
	if token == nil {
		rootNames, err := readAllPiLifecycleNames(rootFD, budget)
		if err != nil {
			return nil, nil, err
		}
		knownRoot := map[string]bool{"foreground.lock": true, "retention.lock": true, "generation.json": true, "legacy-generation.json": true, "entries": true}
		for _, name := range rootNames {
			if !knownRoot[name] {
				return nil, nil, piError("lifecycle_log_evidence_unknown", fmt.Errorf("foreign aggregate-root entry %q", name))
			}
		}
		if len(rootNames) != len(knownRoot) {
			return nil, nil, piError("lifecycle_log_evidence_unknown", errors.New("aggregate root is missing a fixed entry"))
		}
	} else if token.Phase != "aggregate" {
		return nil, nil, piError("lifecycle_log_evidence_unknown", errors.New("invalid aggregate continuation phase"))
	}
	entriesFD, identity, err := openPiLifecycleDirectoryAt(rootFD, "entries")
	if err != nil {
		return nil, nil, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(entriesFD)
	cookie := int64(0)
	if token != nil {
		if !piLifecycleContinuationMatchesDirectory(*token, identity) {
			return nil, nil, piError("lifecycle_log_evidence_unknown", errors.New("aggregate continuation directory changed"))
		}
		cookie = token.Offset
	}
	var envelopes []piLifecycleEnvelope
	nextCookie, complete, scanErr := scanPiLifecycleDirectoryPageSized(ctx, entriesFD, cookie, budget, 96, func(entry piLifecycleDirent) error {
		if !piLifecycleEntryName.MatchString(entry.Name) {
			return piError("lifecycle_log_evidence_unknown", fmt.Errorf("unproven aggregate entry %q", entry.Name))
		}
		envelope, err := scanOnePiLifecycleEnvelope(entriesFD, entry.Name, budget, false, nil)
		if err != nil {
			return err
		}
		envelopes = append(envelopes, envelope)
		return nil
	})
	if err := revalidatePiLifecycleDirectoryAt(rootFD, "entries", entriesFD, identity); err != nil {
		return envelopes, nil, piError("lifecycle_log_evidence_unknown", err)
	}
	if scanErr != nil {
		if piErrorIs(scanErr, "lifecycle_log_scan_exhausted") {
			cursor := piLifecycleDirectoryCursor("aggregate", nextCookie, identity)
			return envelopes, &cursor, scanErr
		}
		return envelopes, nil, scanErr
	}
	if !complete {
		cursor := piLifecycleDirectoryCursor("aggregate", nextCookie, identity)
		return envelopes, &cursor, piError("lifecycle_log_scan_exhausted", errors.New("aggregate scan requires continuation"))
	}
	return envelopes, nil, nil
}

func addPiLifecycleEnvelopeStatus(status *PiLifecycleLogStatus, policy PiLifecycleLogRetention, envelopes []piLifecycleEnvelope) {
	cutoff := piLifecycleNow().Add(-time.Duration(policy.MaxAgeSeconds) * time.Second)
	for _, item := range envelopes {
		status.ManagedCount++
		status.ManagedCommittedBytes += item.LogBytes
		status.ManagedEnvelopeBytes += item.EnvelopeBytes
		if item.Active {
			status.ActiveCount++
		}
		created, _ := time.Parse(time.RFC3339Nano, item.Record.CreatedAt)
		if created.Before(cutoff) {
			status.ExpiredCount++
		}
		if status.Oldest == "" || item.Record.CreatedAt < status.Oldest {
			status.Oldest = item.Record.CreatedAt
		}
		if item.Record.CreatedAt > status.Newest {
			status.Newest = item.Record.CreatedAt
		}
	}
}

// PiLifecycleStatus takes no writer lock and never repairs state.
func PiLifecycleStatus(ctx context.Context, paths PiStatePaths, policy PiLifecycleLogRetention, policySource, continuation string) (PiLifecycleLogStatus, error) {
	status := PiLifecycleLogStatus{PolicySource: policySource, AggregateRoot: paths.LifecycleLogsRoot, Policy: policy, PageScope: "whole-profile"}
	opCtx, cancel := piLifecycleOperationContext(ctx, policy.StatusTimeoutSeconds)
	defer cancel()
	profileFD, err := openPiAggregateProfileRoot(paths)
	if err != nil {
		return piLifecycleUnknownStatus(status, err)
	}
	defer unix.Close(profileFD)
	rootFD, rootStat, entriesProofFD, entriesStat, err := openPiLifecycleStatusAuthority(profileFD)
	if err != nil {
		return piLifecycleUnknownStatus(status, err)
	}
	defer unix.Close(rootFD)
	defer unix.Close(entriesProofFD)
	budget := &piLifecycleBudget{policy: policy}
	aggregateStart, legacyStart, err := readPiLifecycleGenerationPair(rootFD, budget)
	if err != nil {
		return piLifecycleUnknownStatus(status, err)
	}
	status.AggregateGeneration, status.LegacyGeneration = aggregateStart.Generation, legacyStart.Generation
	status.RecoveryCount, status.PrunedCount = aggregateStart.Recovered, aggregateStart.Pruned
	if aggregateStart.State != "even" || legacyStart.State != "even" {
		return piLifecycleUnknownStatus(status, errors.New("odd lifecycle generation"))
	}
	finalControlReserve := budget.controlBytes
	budget.controlLimit = policy.MaxScanControlBytes - finalControlReserve
	if budget.controlLimit < budget.controlBytes {
		budget.controlLimit = budget.controlBytes
	}
	var token *piLifecycleContinuation
	if continuation != "" {
		decoded, tokenErr := decodePiLifecycleContinuation(continuation)
		if tokenErr != nil || decoded.ProjectKey != paths.ProjectStateKey || decoded.ProfileKey != paths.ProfileStateKey || decoded.PolicyDigest != piLifecyclePolicyDigest(policy) || decoded.AggregateGeneration != aggregateStart.Generation || decoded.LegacyGeneration != legacyStart.Generation || decoded.RootDevice != uint64(rootStat.Dev) || decoded.RootInode != rootStat.Ino || decoded.RootChangeNsec != piLifecycleStatChangeNsec(rootStat) {
			return piLifecycleUnknownStatus(status, errors.New("stale or foreign lifecycle continuation"))
		}
		token = &decoded
		status.PageScope, status.LowerBound = decoded.Phase, true
	}

	var cursor *piLifecycleContinuation
	var scanErr error
	if token == nil || token.Phase == "aggregate" {
		envelopes, aggregateCursor, err := scanPiLifecycleEntriesPage(opCtx, rootFD, budget, token)
		addPiLifecycleEnvelopeStatus(&status, policy, envelopes)
		cursor, scanErr = aggregateCursor, err
		if scanErr == nil {
			cursor, scanErr = scanPiLifecycleLegacyPage(opCtx, profileFD, budget, &status, nil)
		}
	} else {
		cursor, scanErr = scanPiLifecycleLegacyPage(opCtx, profileFD, budget, &status, token)
	}

	budget.controlLimit = 0
	aggregateFinal, legacyFinal, finalErr := readPiLifecycleGenerationPair(rootFD, budget)
	status.ScanEntries, status.ScanControlBytes = budget.entries, budget.controlBytes
	if finalErr != nil {
		if piErrorIs(finalErr, "lifecycle_log_scan_exhausted") {
			status.ScanExhausted, status.LowerBound = true, true
			status.UnknownCount++
			status.Errors = append(status.Errors, finalErr.Error())
			return status, finalErr
		}
		return piLifecycleUnknownStatus(status, finalErr)
	}
	if aggregateFinal != aggregateStart || !reflect.DeepEqual(legacyFinal, legacyStart) || aggregateFinal.State != "even" || legacyFinal.State != "even" {
		return piLifecycleUnknownStatus(status, errors.New("lifecycle generation changed during status scan"))
	}
	if err := revalidatePiLifecycleStatusAuthority(profileFD, rootFD, rootStat, entriesProofFD, entriesStat); err != nil {
		return piLifecycleUnknownStatus(status, err)
	}
	if scanErr != nil {
		if piErrorIs(scanErr, "lifecycle_log_scan_exhausted") && cursor != nil {
			cursor.ProjectKey, cursor.ProfileKey = paths.ProjectStateKey, paths.ProfileStateKey
			cursor.PolicyDigest = piLifecyclePolicyDigest(policy)
			cursor.AggregateGeneration, cursor.LegacyGeneration = aggregateStart.Generation, legacyStart.Generation
			cursor.RootDevice, cursor.RootInode = uint64(rootStat.Dev), rootStat.Ino
			cursor.RootChangeNsec = piLifecycleStatChangeNsec(rootStat)
			status.ScanExhausted, status.LowerBound = true, true
			status.UnknownCount++
			status.Continuation = encodePiLifecycleContinuation(*cursor)
			status.PageScope = cursor.Phase
			status.Errors = append(status.Errors, scanErr.Error())
			status.WithinPolicy, status.SoakReady = false, false
			return status, scanErr
		}
		return piLifecycleUnknownStatus(status, scanErr)
	}
	status.ScanComplete = continuation == ""
	status.LowerBound = continuation != ""
	status.WithinPolicy = continuation == "" && status.UnknownCount == 0 && status.LegacyCount == 0 && status.ForeignCount == 0 && status.ManagedCount <= policy.MaxCount && status.ManagedCommittedBytes <= int64(policy.MaxBytes) && status.ManagedEnvelopeBytes <= int64(policy.MaxEnvelopeBytes) && status.ExpiredCount == 0
	status.SoakReady = status.WithinPolicy
	return status, nil
}

func openPiLifecycleStatusAuthority(profileFD int) (int, unix.Stat_t, int, unix.Stat_t, error) {
	rootFD, err := unix.Openat(profileFD, "lifecycle-logs", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, unix.Stat_t{}, -1, unix.Stat_t{}, err
	}
	rootStat, err := validatePiLifecycleDirAt(profileFD, "lifecycle-logs", rootFD)
	if err != nil {
		unix.Close(rootFD)
		return -1, unix.Stat_t{}, -1, unix.Stat_t{}, err
	}
	entriesFD, err := unix.Openat(rootFD, "entries", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		unix.Close(rootFD)
		return -1, unix.Stat_t{}, -1, unix.Stat_t{}, err
	}
	entriesStat, err := validatePiLifecycleDirAt(rootFD, "entries", entriesFD)
	if err != nil {
		unix.Close(entriesFD)
		unix.Close(rootFD)
		return -1, unix.Stat_t{}, -1, unix.Stat_t{}, err
	}
	return rootFD, rootStat, entriesFD, entriesStat, nil
}

func revalidatePiLifecycleStatusAuthority(profileFD, rootFD int, rootStat unix.Stat_t, entriesFD int, entriesStat unix.Stat_t) error {
	rootFinal, err := validatePiLifecycleDirAt(profileFD, "lifecycle-logs", rootFD)
	if err != nil || rootFinal.Dev != rootStat.Dev || rootFinal.Ino != rootStat.Ino || piLifecycleStatChangeNsec(rootFinal) != piLifecycleStatChangeNsec(rootStat) {
		return errors.New("aggregate root authority changed during status scan")
	}
	entriesFinal, err := validatePiLifecycleDirAt(rootFD, "entries", entriesFD)
	if err != nil || entriesFinal.Dev != entriesStat.Dev || entriesFinal.Ino != entriesStat.Ino || piLifecycleStatChangeNsec(entriesFinal) != piLifecycleStatChangeNsec(entriesStat) {
		return errors.New("aggregate entries authority changed during status scan")
	}
	return nil
}

func piLifecycleUnknownStatus(status PiLifecycleLogStatus, err error) (PiLifecycleLogStatus, error) {
	status.UnknownCount++
	status.Errors = append(status.Errors, err.Error())
	status.WithinPolicy, status.SoakReady = false, false
	return status, piError("lifecycle_log_evidence_unknown", err)
}

func readPiLifecycleGenerationPair(rootFD int, budget *piLifecycleBudget) (piLifecycleGeneration, piLifecycleLegacyGeneration, error) {
	encoded, _, err := readPiLifecycleControl(rootFD, "generation.json", budget)
	if err != nil {
		if piErrorIs(err, "lifecycle_log_scan_exhausted") {
			return piLifecycleGeneration{}, piLifecycleLegacyGeneration{}, err
		}
		return piLifecycleGeneration{}, piLifecycleLegacyGeneration{}, piError("lifecycle_log_evidence_unknown", err)
	}
	var aggregate piLifecycleGeneration
	if err := decodePiLifecycleControl(encoded, &aggregate); err != nil {
		return aggregate, piLifecycleLegacyGeneration{}, piError("lifecycle_log_evidence_unknown", err)
	}
	if err := validatePiLifecycleGeneration(aggregate, "aggregate"); err != nil {
		return aggregate, piLifecycleLegacyGeneration{}, piError("lifecycle_log_evidence_unknown", err)
	}
	legacy, err := readPiLifecycleLegacyGeneration(rootFD, budget)
	if err != nil {
		if piErrorIs(err, "lifecycle_log_scan_exhausted") {
			return aggregate, legacy, err
		}
		return aggregate, legacy, piError("lifecycle_log_evidence_unknown", err)
	}
	return aggregate, legacy, nil
}

var errPiLifecyclePageStop = errors.New("lifecycle page stop")

func scanPiLifecycleLogDirectoryPage(ctx context.Context, parentFD int, name string, budget *piLifecycleBudget, status *PiLifecycleLogStatus, cookie int64, expected *piLifecycleDirectoryIdentity) (*piLifecycleContinuation, error) {
	logsFD, identity, err := openPiLifecycleDirectoryAt(parentFD, name)
	if errors.Is(err, syscall.ENOENT) {
		if expected != nil {
			return nil, piError("lifecycle_log_evidence_unknown", errors.New("continued legacy log directory disappeared"))
		}
		return nil, nil
	}
	if err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(logsFD)
	if expected != nil && identity != *expected {
		return nil, piError("lifecycle_log_evidence_unknown", errors.New("continued legacy log directory changed"))
	}
	nextCookie, complete, scanErr := scanPiLifecycleDirectoryPage(ctx, logsFD, cookie, budget, func(entry piLifecycleDirent) error {
		var st unix.Stat_t
		if err := unix.Fstatat(logsFD, entry.Name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		if st.Mode&unix.S_IFMT == unix.S_IFREG && st.Mode&0o777 == 0o600 && st.Nlink == 1 && st.Uid == uint32(os.Geteuid()) && strings.HasSuffix(entry.Name, ".jsonl") {
			status.LegacyCount++
			status.LegacyBytes += st.Size
		} else {
			status.ForeignCount++
			if st.Mode&unix.S_IFMT == unix.S_IFREG {
				status.ForeignBytes += st.Size
			}
		}
		return nil
	})
	if err := revalidatePiLifecycleDirectoryAt(parentFD, name, logsFD, identity); err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	if scanErr != nil {
		if piErrorIs(scanErr, "lifecycle_log_scan_exhausted") {
			cursor := piLifecycleDirectoryCursor("legacy-profile-logs", nextCookie, identity)
			return &cursor, scanErr
		}
		return nil, scanErr
	}
	if !complete {
		cursor := piLifecycleDirectoryCursor("legacy-profile-logs", nextCookie, identity)
		return &cursor, piError("lifecycle_log_scan_exhausted", errors.New("legacy log scan requires continuation"))
	}
	return nil, nil
}

func scanPiLifecycleLegacyPage(ctx context.Context, profileFD int, budget *piLifecycleBudget, status *PiLifecycleLogStatus, token *piLifecycleContinuation) (*piLifecycleContinuation, error) {
	phase := ""
	if token != nil {
		phase = token.Phase
	}
	if phase == "" || phase == "legacy-profile-logs" {
		cookie := int64(0)
		var expected *piLifecycleDirectoryIdentity
		if phase == "legacy-profile-logs" {
			cookie = token.Offset
			value := piLifecycleDirectoryIdentity{Device: token.DirectoryDevice, Inode: token.DirectoryInode, ChangeNsec: token.DirectoryChangeNsec}
			expected = &value
		}
		cursor, err := scanPiLifecycleLogDirectoryPage(ctx, profileFD, "logs", budget, status, cookie, expected)
		if err != nil {
			return cursor, err
		}
		phase = "legacy-runs"
	}

	runsFD, runsIdentity, err := openPiLifecycleDirectoryAt(profileFD, "runs")
	if errors.Is(err, syscall.ENOENT) {
		if token != nil && (token.Phase == "legacy-runs" || token.Phase == "legacy-run-logs") {
			return nil, piError("lifecycle_log_evidence_unknown", errors.New("continued runs directory disappeared"))
		}
		return nil, nil
	}
	if err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	defer unix.Close(runsFD)
	runsCookie := int64(0)
	if phase == "legacy-runs" && token != nil {
		expected := piLifecycleDirectoryIdentity{Device: token.DirectoryDevice, Inode: token.DirectoryInode, ChangeNsec: token.DirectoryChangeNsec}
		if runsIdentity != expected {
			return nil, piError("lifecycle_log_evidence_unknown", errors.New("continued runs directory changed"))
		}
		runsCookie = token.Offset
	} else if phase == "legacy-run-logs" {
		expectedRuns := piLifecycleDirectoryIdentity{Device: token.ParentDevice, Inode: token.ParentInode, ChangeNsec: token.ParentChangeNsec}
		if runsIdentity != expectedRuns || !piStateKeyPattern.MatchString(token.RunName) {
			return nil, piError("lifecycle_log_evidence_unknown", errors.New("continued run-log parent changed"))
		}
		runFD, runIdentity, openErr := openPiLifecycleDirectoryAt(runsFD, token.RunName)
		if openErr != nil {
			return nil, piError("lifecycle_log_evidence_unknown", openErr)
		}
		expectedLogs := piLifecycleDirectoryIdentity{Device: token.DirectoryDevice, Inode: token.DirectoryInode, ChangeNsec: token.DirectoryChangeNsec}
		cursor, scanErr := scanPiLifecycleLogDirectoryPage(ctx, runFD, "logs", budget, status, token.Offset, &expectedLogs)
		if revalidateErr := revalidatePiLifecycleDirectoryAt(runsFD, token.RunName, runFD, runIdentity); scanErr == nil && revalidateErr != nil {
			scanErr = piError("lifecycle_log_evidence_unknown", revalidateErr)
		}
		unix.Close(runFD)
		if scanErr != nil {
			if cursor != nil {
				cursor.Phase = "legacy-run-logs"
				cursor.RunName = token.RunName
				cursor.ParentOffset = token.ParentOffset
				cursor.ParentDevice, cursor.ParentInode, cursor.ParentChangeNsec = runsIdentity.Device, runsIdentity.Inode, runsIdentity.ChangeNsec
			}
			return cursor, scanErr
		}
		runsCookie = token.ParentOffset
	} else if phase != "legacy-runs" {
		return nil, piError("lifecycle_log_evidence_unknown", errors.New("invalid legacy continuation phase"))
	}

	var nestedCursor *piLifecycleContinuation
	nextCookie, complete, scanErr := scanPiLifecycleDirectoryPageSized(ctx, runsFD, runsCookie, budget, 96, func(entry piLifecycleDirent) error {
		if !piStateKeyPattern.MatchString(entry.Name) {
			status.ForeignCount++
			return nil
		}
		runFD, runIdentity, err := openPiLifecycleDirectoryAt(runsFD, entry.Name)
		if err != nil {
			return piError("lifecycle_log_evidence_unknown", err)
		}
		cursor, logErr := scanPiLifecycleLogDirectoryPage(ctx, runFD, "logs", budget, status, 0, nil)
		if revalidateErr := revalidatePiLifecycleDirectoryAt(runsFD, entry.Name, runFD, runIdentity); logErr == nil && revalidateErr != nil {
			logErr = piError("lifecycle_log_evidence_unknown", revalidateErr)
		}
		unix.Close(runFD)
		if cursor != nil {
			cursor.Phase = "legacy-run-logs"
			cursor.RunName = entry.Name
			cursor.ParentOffset = entry.Cookie
			cursor.ParentDevice, cursor.ParentInode, cursor.ParentChangeNsec = runsIdentity.Device, runsIdentity.Inode, runsIdentity.ChangeNsec
			nestedCursor = cursor
			return errPiLifecyclePageStop
		}
		return logErr
	})
	if err := revalidatePiLifecycleDirectoryAt(profileFD, "runs", runsFD, runsIdentity); err != nil {
		return nil, piError("lifecycle_log_evidence_unknown", err)
	}
	if errors.Is(scanErr, errPiLifecyclePageStop) {
		return nestedCursor, piError("lifecycle_log_scan_exhausted", errors.New("run log scan requires continuation"))
	}
	if scanErr != nil {
		if piErrorIs(scanErr, "lifecycle_log_scan_exhausted") {
			cursor := piLifecycleDirectoryCursor("legacy-runs", nextCookie, runsIdentity)
			return &cursor, scanErr
		}
		return nil, scanErr
	}
	if !complete {
		cursor := piLifecycleDirectoryCursor("legacy-runs", nextCookie, runsIdentity)
		return &cursor, piError("lifecycle_log_scan_exhausted", errors.New("runs scan requires continuation"))
	}
	return nil, nil
}

func piLifecycleStatChangeNsec(st unix.Stat_t) int64 {
	return st.Ctim.Sec*int64(time.Second) + st.Ctim.Nsec
}

func piErrorIs(err error, code string) bool {
	var launch *PiLaunchError
	return errors.As(err, &launch) && launch.Code == code
}
