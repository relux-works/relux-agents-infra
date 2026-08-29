//go:build !windows

package infra

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

var piStateKeyPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type PiStatePaths struct {
	CanonicalCacheRoot string `json:"canonical_cache_root"`
	ProjectStateKey    string `json:"project_state_key"`
	ProfileStateKey    string `json:"profile_state_key"`
	RunStateKey        string `json:"run_state_key,omitempty"`
	Root               string `json:"root"`
	AgentDir           string `json:"agent_dir"`
	SessionsDir        string `json:"sessions_dir"`
	LogsDir            string `json:"logs_dir"`
	ModelsJSON         string `json:"models_json"`
	SettingsJSON       string `json:"settings_json"`
	Lock               string `json:"lock"`
	components         []string
}

func exactStateKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

var piStateKey = exactStateKey

func ResolvePiStatePaths(cacheRoot, canonicalProject, profileName string) (PiStatePaths, error) {
	return resolvePiClientStatePaths(cacheRoot, canonicalProject, profileName, "")
}

func ResolvePiClientStatePaths(cacheRoot, canonicalProject, profileName, runID string) (PiStatePaths, error) {
	return resolvePiClientStatePaths(cacheRoot, canonicalProject, profileName, runID)
}

func resolvePiClientStatePaths(cacheRoot, canonicalProject, profileName, runID string) (PiStatePaths, error) {
	if cacheRoot == "" {
		var err error
		cacheRoot, err = os.UserCacheDir()
		if err != nil {
			return PiStatePaths{}, piError("profile_state_path_invalid", fmt.Errorf("resolve user cache directory: %w", err))
		}
	}
	abs, err := filepath.Abs(cacheRoot)
	if err != nil {
		return PiStatePaths{}, piError("profile_state_path_invalid", err)
	}
	canonical, err := filepath.EvalSymlinks(filepath.Clean(abs))
	if err != nil {
		return PiStatePaths{}, piError("profile_state_path_invalid", fmt.Errorf("canonicalize cache root: %w", err))
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("cache root is not a directory")
		}
		return PiStatePaths{}, piError("profile_state_path_invalid", err)
	}
	projectKey, profileKey := piStateKey(canonicalProject), piStateKey(profileName)
	if !piStateKeyPattern.MatchString(projectKey) || !piStateKeyPattern.MatchString(profileKey) {
		return PiStatePaths{}, piError("profile_state_path_invalid", errors.New("state key is not lowercase SHA-256 hex"))
	}
	components := []string{"agents-infra", "pi", projectKey, profileKey}
	runKey := ""
	lockName := "session.lock"
	if runID != "" {
		runKey = exactStateKey("agents-infra.pi.run.v1\x00" + runID)
		if !piStateKeyPattern.MatchString(runKey) {
			return PiStatePaths{}, piError("profile_state_path_invalid", errors.New("run state key is not lowercase SHA-256 hex"))
		}
		components = append(components, "runs", runKey)
		lockName = "client.lock"
	}
	suffix := filepath.Join(components...)
	if filepath.IsAbs(suffix) || len(splitCleanSuffix(suffix)) != len(components) {
		return PiStatePaths{}, piError("profile_state_path_invalid", errors.New("managed state suffix is invalid"))
	}
	root := filepath.Join(canonical, suffix)
	rel, err := filepath.Rel(canonical, root)
	if err != nil || rel != suffix {
		return PiStatePaths{}, piError("profile_state_path_invalid", errors.New("managed state escaped canonical cache root"))
	}
	paths := PiStatePaths{CanonicalCacheRoot: canonical, ProjectStateKey: projectKey, ProfileStateKey: profileKey, RunStateKey: runKey, Root: root, AgentDir: filepath.Join(root, "agent"), SessionsDir: filepath.Join(root, "sessions"), LogsDir: filepath.Join(root, "logs"), ModelsJSON: filepath.Join(root, "agent", "models.json"), SettingsJSON: filepath.Join(root, "agent", "settings.json"), Lock: filepath.Join(root, lockName), components: components}
	if err := validateExistingPiStateComponents(paths); err != nil {
		return PiStatePaths{}, piError("profile_state_path_invalid", err)
	}
	return paths, nil
}

func validateExistingPiStateComponents(paths PiStatePaths) error {
	rootFD, err := unix.Open(paths.CanonicalCacheRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer unix.Close(rootFD)
	fd := rootFD
	for _, name := range statePathComponents(paths) {
		next, openErr := unix.Openat(fd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if fd != rootFD {
			unix.Close(fd)
		}
		if errors.Is(openErr, syscall.ENOENT) {
			return nil
		}
		if openErr != nil {
			return openErr
		}
		fd = next
	}
	defer func() {
		if fd != rootFD {
			unix.Close(fd)
		}
	}()
	for _, name := range []string{"agent", "sessions", "logs"} {
		child, openErr := unix.Openat(fd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			continue
		}
		if openErr != nil {
			return openErr
		}
		unix.Close(child)
	}
	return nil
}

func ValidatePiStateKeyCollisions(profiles map[string]PiProfile) error {
	seen := map[string]string{}
	for name := range profiles {
		key := piStateKey(name)
		if other, ok := seen[key]; ok && other != name {
			return piError("profile_state_key_collision", fmt.Errorf("profiles %q and %q share state key %s", other, name, key))
		}
		seen[key] = name
	}
	return nil
}

func splitCleanSuffix(path string) []string {
	var out []string
	for path != "." && path != string(filepath.Separator) {
		dir, base := filepath.Split(path)
		if base == "" || base == "." || base == ".." {
			return nil
		}
		out = append([]string{base}, out...)
		path = filepath.Clean(dir)
	}
	return out
}

// CreatePiStateTree creates every managed component relative to an already
// opened canonical-cache-root descriptor. O_NOFOLLOW plus post-open fstat makes
// component replacement/symlink attacks refusals rather than path traversal.
func CreatePiStateTree(paths PiStatePaths) error {
	rootFD, err := unix.Open(paths.CanonicalCacheRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piError("profile_state_path_invalid", err)
	}
	defer unix.Close(rootFD)
	components := statePathComponents(paths)
	fd := rootFD
	for _, component := range components {
		next, openErr := openOrCreateDirAt(fd, component)
		if fd != rootFD {
			unix.Close(fd)
		}
		if openErr != nil {
			return piError("profile_state_path_invalid", openErr)
		}
		fd = next
	}
	defer func() {
		if fd != rootFD {
			unix.Close(fd)
		}
	}()
	for _, childName := range []string{"agent", "sessions", "logs"} {
		child, openErr := openOrCreateDirAt(fd, childName)
		if openErr != nil {
			return piError("profile_state_path_invalid", openErr)
		}
		unix.Close(child)
	}
	return nil
}

func statePathComponents(paths PiStatePaths) []string {
	if len(paths.components) > 0 {
		return append([]string(nil), paths.components...)
	}
	components := []string{"agents-infra", "pi", paths.ProjectStateKey, paths.ProfileStateKey}
	if paths.RunStateKey != "" {
		components = append(components, "runs", paths.RunStateKey)
	}
	return components
}

func openOrCreateDirAt(parent int, name string) (int, error) {
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		if err = unix.Mkdirat(parent, name, 0o700); err != nil && !errors.Is(err, syscall.EEXIST) {
			return -1, err
		}
		fd, err = unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	}
	if err != nil {
		return -1, err
	}
	if err := piRevalidateStateDir(fd); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

var piRevalidateStateDir = func(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		return errors.New("managed state component is not a directory")
	}
	return nil
}

type PiProfileLock struct{ file *os.File }

func AcquirePiProfileLock(paths PiStatePaths) (*PiProfileLock, error) {
	rootFD, err := openPiProfileRoot(paths)
	if err != nil {
		return nil, piError("profile_state_path_invalid", err)
	}
	defer unix.Close(rootFD)
	lockName := filepath.Base(paths.Lock)
	if lockName != "session.lock" && lockName != "client.lock" {
		return nil, piError("profile_state_path_invalid", errors.New("unexpected client lock name"))
	}
	fd, err := unix.Openat(rootFD, lockName, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, piError("profile_state_path_invalid", err)
	}
	f := os.NewFile(uintptr(fd), paths.Lock)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 {
		if err == nil {
			err = errors.New("session lock must be a single-link regular file")
		}
		f.Close()
		return nil, piError("profile_state_path_invalid", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, piError("pi_profile_busy", err)
		}
		return nil, piError("profile_state_path_invalid", err)
	}
	return &PiProfileLock{file: f}, nil
}
func (l *PiProfileLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return l.file.Close()
}

func WritePiModelsJSON(paths PiStatePaths, content []byte) error {
	rootFD, err := openPiProfileRoot(paths)
	if err != nil {
		return piError("profile_state_path_invalid", err)
	}
	defer unix.Close(rootFD)
	dirFD, err := unix.Openat(rootFD, "agent", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piError("profile_state_path_invalid", err)
	}
	defer unix.Close(dirFD)
	return writePiAgentFileAtomic(dirFD, "models.json", content)
}

func WritePiCompactionSettings(paths PiStatePaths, compaction *PiCompaction) error {
	if compaction == nil {
		return nil
	}
	rootFD, err := openPiProfileRoot(paths)
	if err != nil {
		return piError("profile_state_path_invalid", err)
	}
	defer unix.Close(rootFD)
	dirFD, err := unix.Openat(rootFD, "agent", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return piError("profile_state_path_invalid", err)
	}
	defer unix.Close(dirFD)
	document := map[string]json.RawMessage{}
	existing, present, err := readPiAgentFile(dirFD, "settings.json")
	if err != nil {
		return piError("pi_settings_invalid", err)
	}
	if present {
		if err := json.Unmarshal(existing, &document); err != nil {
			return piError("pi_settings_invalid", fmt.Errorf("decode managed Pi settings: %w", err))
		}
		if document == nil {
			return piError("pi_settings_invalid", errors.New("managed Pi settings must be a JSON object"))
		}
	}
	nativeCompaction := struct {
		Enabled          bool `json:"enabled"`
		ReserveTokens    int  `json:"reserveTokens"`
		KeepRecentTokens int  `json:"keepRecentTokens"`
	}{
		Enabled:          compaction.Enabled,
		ReserveTokens:    compaction.ReserveTokens,
		KeepRecentTokens: compaction.KeepRecentTokens,
	}
	encodedCompaction, err := json.Marshal(nativeCompaction)
	if err != nil {
		return piError("pi_settings_invalid", err)
	}
	document["compaction"] = encodedCompaction
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return piError("pi_settings_invalid", err)
	}
	content = append(content, '\n')
	if err := writePiAgentFileAtomic(dirFD, "settings.json", content); err != nil {
		return err
	}
	return nil
}

func readPiAgentFile(dirFD int, name string) ([]byte, bool, error) {
	fd, err := unix.Openat(dirFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return nil, false, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 {
		return nil, false, errors.New("managed Pi settings must be a single-link regular file")
	}
	const maxSettingsBytes = 1 << 20
	content, err := io.ReadAll(io.LimitReader(f, maxSettingsBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(content) > maxSettingsBytes {
		return nil, false, errors.New("managed Pi settings exceed 1 MiB")
	}
	return content, true, nil
}

func writePiAgentFileAtomic(dirFD int, name string, content []byte) error {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return piError("profile_state_path_invalid", err)
	}
	temp := "." + name + ".tmp-" + strconv.Itoa(os.Getpid()) + "-" + hex.EncodeToString(nonce[:])
	fd, err := unix.Openat(dirFD, temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return piError("profile_state_path_invalid", err)
	}
	f := os.NewFile(uintptr(fd), temp)
	writeErr := error(nil)
	if n, err := f.Write(content); err != nil || n != len(content) {
		if err == nil {
			err = fmt.Errorf("partial %s write", name)
		}
		writeErr = err
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = unix.Unlinkat(dirFD, temp, 0)
		return piError("profile_state_path_invalid", writeErr)
	}
	if err := unix.Renameat(dirFD, temp, dirFD, name); err != nil {
		_ = unix.Unlinkat(dirFD, temp, 0)
		return piError("profile_state_path_invalid", err)
	}
	if err := unix.Fsync(dirFD); err != nil {
		return piError("profile_state_path_invalid", err)
	}
	return nil
}

func openPiProfileRoot(paths PiStatePaths) (int, error) {
	rootFD, err := unix.Open(paths.CanonicalCacheRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, err
	}
	fd := rootFD
	for _, name := range statePathComponents(paths) {
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
