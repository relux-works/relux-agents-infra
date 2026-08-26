//go:build !windows

package infra

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type piSessionLog struct {
	mu   sync.Mutex
	file *os.File
	path string
}

type piSessionLogRecord struct {
	Timestamp string         `json:"timestamp"`
	Event     string         `json:"event"`
	Fields    map[string]any `json:"fields,omitempty"`
}

func openPiSessionLog(paths PiStatePaths) (*piSessionLog, error) {
	rootFD, err := openPiProfileRoot(paths)
	if err != nil {
		return nil, piError("profile_state_path_invalid", err)
	}
	defer unix.Close(rootFD)
	logsFD, err := unix.Openat(rootFD, "logs", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, piError("profile_state_path_invalid", err)
	}
	defer unix.Close(logsFD)
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, piError("profile_state_path_invalid", err)
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(nonce[:]) + ".jsonl"
	fd, err := unix.Openat(logsFD, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, piError("profile_state_path_invalid", err)
	}
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Nlink != 1 || stat.Mode&0o777 != 0o600 {
		if err == nil {
			err = errors.New("session log must be a mode-0600 single-link regular file")
		}
		unix.Close(fd)
		return nil, piError("profile_state_path_invalid", err)
	}
	path := filepath.Join(paths.LogsDir, name)
	return &piSessionLog{file: os.NewFile(uintptr(fd), path), path: path}, nil
}

func (log *piSessionLog) event(name string, fields map[string]any) {
	if log == nil || log.file == nil {
		return
	}
	record := piSessionLogRecord{Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Event: name, Fields: fields}
	data, err := json.Marshal(record)
	if err != nil {
		return
	}
	data = append(data, '\n')
	log.mu.Lock()
	defer log.mu.Unlock()
	_, _ = log.file.Write(data)
}

func (log *piSessionLog) close() {
	if log == nil || log.file == nil {
		return
	}
	log.mu.Lock()
	defer log.mu.Unlock()
	_ = log.file.Sync()
	_ = log.file.Close()
	log.file = nil
}

func piProcessIdentityFields(pid int) map[string]any {
	fields := map[string]any{"pid": pid}
	if pgid, err := syscall.Getpgid(pid); err == nil {
		fields["pgid"] = pgid
	}
	return fields
}
