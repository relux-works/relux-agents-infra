//go:build darwin

package infra

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const sharedLogArchiveTimestampLayout = "20060102T150405.000000000Z"

type sharedFilesystemLogSink struct {
	path string
}

func (sink *sharedFilesystemLogSink) Open(_ time.Time) (io.WriteCloser, int64, error) {
	file, err := openSharedLog(sink.path)
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	return file, info.Size(), nil
}

func (sink *sharedFilesystemLogSink) Rotate(at time.Time) (io.WriteCloser, error) {
	archive, err := sink.nextArchivePath(at)
	if err != nil {
		return nil, err
	}
	if err := os.Rename(sink.path, archive); err != nil {
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	file, err := openSharedLog(sink.path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (sink *sharedFilesystemLogSink) Prune(maxSegmentBytes int64, maxSegments int) error {
	archives, _, err := sink.archives()
	if err != nil {
		return err
	}
	for _, archive := range archives {
		var stat unix.Stat_t
		statErr := unix.Lstat(archive, &stat)
		if statErr != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Mode&0o777 != 0o600 || stat.Nlink != 1 {
			if statErr == nil {
				statErr = fmt.Errorf("managed log archive must be a mode-0600 single-link regular file")
			}
			return sharedRuntimeError("shared_runtime_state_path_invalid", statErr)
		}
		if stat.Size > maxSegmentBytes {
			return sharedRuntimeError("shared_runtime_state_path_invalid", fmt.Errorf("managed log archive exceeds max_segment_bytes"))
		}
	}
	maxArchives := maxSegments - 1
	if excess := len(archives) - maxArchives; excess > 0 {
		for _, archive := range archives[:excess] {
			if removeErr := os.Remove(archive); removeErr != nil {
				return sharedRuntimeError("shared_runtime_state_path_invalid", removeErr)
			}
		}
	}
	return nil
}

func (sink *sharedFilesystemLogSink) nextArchivePath(at time.Time) (string, error) {
	_, highest, err := sink.archives()
	if err != nil {
		return "", err
	}
	if highest == ^uint64(0) {
		return "", sharedRuntimeError("shared_runtime_state_path_invalid", fmt.Errorf("runtime log archive sequence exhausted"))
	}
	sequence := highest + 1
	name := fmt.Sprintf("%s.%020d.%s", filepath.Base(sink.path), sequence, at.UTC().Format(sharedLogArchiveTimestampLayout))
	return filepath.Join(filepath.Dir(sink.path), name), nil
}

func (sink *sharedFilesystemLogSink) archives() ([]string, uint64, error) {
	entries, err := os.ReadDir(filepath.Dir(sink.path))
	if err != nil {
		return nil, 0, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	prefix := filepath.Base(sink.path) + "."
	archives := make([]string, 0)
	var highest uint64
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		remainder := strings.TrimPrefix(name, prefix)
		separator := strings.IndexByte(remainder, '.')
		if separator != 20 {
			continue
		}
		sequence, parseErr := strconv.ParseUint(remainder[:separator], 10, 64)
		if parseErr != nil {
			continue
		}
		if _, parseErr := time.Parse(sharedLogArchiveTimestampLayout, remainder[separator+1:]); parseErr != nil {
			continue
		}
		archives = append(archives, filepath.Join(filepath.Dir(sink.path), name))
		if sequence > highest {
			highest = sequence
		}
	}
	sort.Strings(archives)
	return archives, highest, nil
}

func openSharedRotatingLog(path string, maxSegmentBytes, maxSegments int, clock sharedLogClock) (*sharedRotatingLogWriter, error) {
	writer, err := newSharedRotatingLogWriter(&sharedFilesystemLogSink{path: path}, clock, int64(maxSegmentBytes), maxSegments)
	if err != nil {
		return nil, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	return writer, nil
}
