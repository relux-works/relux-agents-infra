//go:build !windows

package infra

import (
	"bytes"
	"crypto/sha256"
	"debug/macho"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"
)

const (
	PiCompatibilityV0842DarwinARM64 = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65"
	piCatalogEntrypointSHA256       = "d5de3fe32f9e109324f32d6e393554fb2ce10bbc82e8ff935ab2e072f5e2f044"
)

var piCatalogManifestSHA256 = "2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b"

//go:embed pi-v0.84.2-darwin-arm64-tree-manifest.txt
var piCatalogManifest []byte

type PiLaunchError struct {
	Code string
	Err  error
}

func (e *PiLaunchError) Error() string     { return e.Err.Error() }
func (e *PiLaunchError) Unwrap() error     { return e.Err }
func piError(code string, err error) error { return &PiLaunchError{Code: code, Err: err} }

type PiExecutionIdentity struct {
	Compatibility     string `json:"compatibility"`
	AssetSHA256       string `json:"asset_sha256"`
	Host              string `json:"host"`
	ReleaseRoot       string `json:"release_root"`
	Entrypoint        string `json:"entrypoint"`
	ManifestSHA256    string `json:"manifest_sha256"`
	EntrypointSHA256  string `json:"entrypoint_sha256"`
	FileCount         int    `json:"file_count"`
	ObservedState     string `json:"observed_state"`
	PointOfUseRecheck bool   `json:"point_of_use_recheck"`
}

type piManifestRecord struct{ Path, SHA256 string }

func compiledPiManifest() ([]piManifestRecord, map[string]bool, error) {
	sum := sha256.Sum256(piCatalogManifest)
	if hex.EncodeToString(sum[:]) != piCatalogManifestSHA256 {
		return nil, nil, errors.New("compiled Pi manifest digest is inconsistent")
	}
	if len(piCatalogManifest) == 0 || piCatalogManifest[len(piCatalogManifest)-1] != '\n' {
		return nil, nil, errors.New("compiled Pi manifest must end in LF")
	}
	lines := bytes.Split(piCatalogManifest[:len(piCatalogManifest)-1], []byte{'\n'})
	if len(lines) != 217 {
		return nil, nil, fmt.Errorf("compiled Pi manifest has %d records, want 217", len(lines))
	}
	records := make([]piManifestRecord, 0, len(lines))
	dirs := map[string]bool{".": true}
	previous := ""
	for i, line := range lines {
		if len(line) < 68 || line[64] != ' ' || line[65] != ' ' || line[66] != '.' || line[67] != '/' {
			return nil, nil, fmt.Errorf("compiled Pi manifest record %d has invalid encoding", i+1)
		}
		digest, rel := string(line[:64]), string(line[68:])
		if _, err := hex.DecodeString(digest); err != nil || digest != strings.ToLower(digest) {
			return nil, nil, fmt.Errorf("compiled Pi manifest record %d has invalid digest", i+1)
		}
		if err := validatePiCatalogPath(rel); err != nil {
			return nil, nil, fmt.Errorf("compiled Pi manifest record %d: %w", i+1, err)
		}
		if previous != "" && bytes.Compare([]byte(previous), []byte(rel)) >= 0 {
			return nil, nil, fmt.Errorf("compiled Pi manifest is not strictly byte sorted at %q", rel)
		}
		previous = rel
		records = append(records, piManifestRecord{Path: rel, SHA256: digest})
		for dir := filepath.ToSlash(filepath.Dir(rel)); dir != "."; dir = filepath.ToSlash(filepath.Dir(dir)) {
			dirs[dir] = true
		}
	}
	return records, dirs, nil
}

func validatePiCatalogPath(rel string) error {
	if rel == "" || !utf8.ValidString(rel) || strings.ContainsAny(rel, "\x00\r\n\\") || strings.HasPrefix(rel, "/") {
		return errors.New("invalid UTF-8 catalog path")
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." {
			return errors.New("invalid catalog path component")
		}
	}
	if filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel))) != rel {
		return errors.New("catalog path is not byte-preserving and relative")
	}
	return nil
}

func VerifyPiExecutionIdentity(piPath, compatibility string) (PiExecutionIdentity, error) {
	if compatibility != PiCompatibilityV0842DarwinARM64 || runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return PiExecutionIdentity{}, piError("pi_compatibility_unsupported", fmt.Errorf("unsupported Pi compatibility %q on %s/%s", compatibility, runtime.GOOS, runtime.GOARCH))
	}
	abs, err := filepath.Abs(piPath)
	if err != nil {
		return PiExecutionIdentity{}, piError("pi_execution_identity_unavailable", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return PiExecutionIdentity{}, piError("pi_execution_identity_unavailable", fmt.Errorf("resolve Pi entrypoint: %w", err))
	}
	resolved = filepath.Clean(resolved)
	root := filepath.Dir(resolved)
	if filepath.Base(resolved) != "pi" {
		return PiExecutionIdentity{}, piError("pi_execution_identity_malformed", errors.New("standalone Pi entrypoint must be named pi"))
	}
	if err := rejectSymlinkComponents(root); err != nil {
		return PiExecutionIdentity{}, piError("pi_execution_identity_malformed", err)
	}
	records, expectedDirs, err := compiledPiManifest()
	if err != nil {
		return PiExecutionIdentity{}, piError("pi_execution_identity_malformed", err)
	}
	expectedFiles := map[string]piManifestRecord{}
	for _, r := range records {
		expectedFiles[r.Path] = r
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return PiExecutionIdentity{}, piError("pi_execution_identity_unavailable", err)
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return PiExecutionIdentity{}, piError("pi_execution_identity_unavailable", errors.New("Pi root stat is unavailable"))
	}
	observedFiles := map[string]bool{}
	observedDirs := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			observedDirs[rel] = true
		} else if err := validatePiCatalogPath(rel); err != nil {
			return fmt.Errorf("observed path %q: %w", rel, err)
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("stat unavailable for %s", rel)
		}
		if uint64(st.Dev) != uint64(rootStat.Dev) {
			return fmt.Errorf("mount crossing at %s", rel)
		}
		mode := info.Mode()
		if entry.IsDir() {
			observedDirs[rel] = true
			if mode.Perm() != 0o755 {
				return fmt.Errorf("directory %s mode %04o, want 0755", rel, mode.Perm())
			}
			if !expectedDirs[rel] {
				return fmt.Errorf("extra directory %s", rel)
			}
			return nil
		}
		if !mode.IsRegular() {
			return fmt.Errorf("non-regular entry %s", rel)
		}
		if st.Nlink != 1 {
			return fmt.Errorf("entry %s link count %d, want 1", rel, st.Nlink)
		}
		record, found := expectedFiles[rel]
		if !found {
			return fmt.Errorf("extra file %s", rel)
		}
		wantMode := fs.FileMode(0o644)
		if rel == "pi" || rel == "examples/extensions/doom-overlay/doom/build.sh" || rel == "examples/extensions/doom-overlay/doom/build/doom.wasm" || rel == "native/darwin/prebuilds/darwin-arm64/darwin-modifiers.node" {
			wantMode = 0o755
		}
		if mode.Perm() != wantMode {
			return fmt.Errorf("file %s mode %04o, want %04o", rel, mode.Perm(), wantMode)
		}
		digest, hashErr := sha256File(path)
		if hashErr != nil {
			return hashErr
		}
		if digest != record.SHA256 {
			return fmt.Errorf("file %s digest mismatch", rel)
		}
		observedFiles[rel] = true
		return nil
	})
	if err != nil {
		return PiExecutionIdentity{}, classifyPiTreeError(err)
	}
	if len(observedFiles) != len(expectedFiles) || len(observedDirs) != len(expectedDirs) {
		return PiExecutionIdentity{}, piError("pi_execution_identity_mismatch", fmt.Errorf("Pi tree inventory mismatch: files %d/%d directories %d/%d", len(observedFiles), len(expectedFiles), len(observedDirs), len(expectedDirs)))
	}
	for path := range expectedFiles {
		if !observedFiles[path] {
			return PiExecutionIdentity{}, piError("pi_execution_identity_mismatch", fmt.Errorf("missing file %s", path))
		}
	}
	for path := range expectedDirs {
		if !observedDirs[path] {
			return PiExecutionIdentity{}, piError("pi_execution_identity_mismatch", fmt.Errorf("missing directory %s", path))
		}
	}
	entryDigest, err := sha256File(resolved)
	if err != nil {
		return PiExecutionIdentity{}, piError("pi_execution_identity_unavailable", err)
	}
	if entryDigest != piCatalogEntrypointSHA256 {
		return PiExecutionIdentity{}, piError("pi_execution_identity_mismatch", errors.New("Pi entrypoint digest mismatch"))
	}
	m, err := macho.Open(resolved)
	if err != nil {
		return PiExecutionIdentity{}, piError("pi_execution_identity_malformed", fmt.Errorf("Pi entrypoint is not Mach-O: %w", err))
	}
	defer m.Close()
	if m.Cpu != macho.CpuArm64 {
		return PiExecutionIdentity{}, piError("pi_execution_identity_mismatch", fmt.Errorf("Pi entrypoint CPU is %v, want arm64", m.Cpu))
	}
	return PiExecutionIdentity{Compatibility: compatibility, AssetSHA256: "c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65", Host: "darwin/arm64", ReleaseRoot: root, Entrypoint: resolved, ManifestSHA256: piCatalogManifestSHA256, EntrypointSHA256: entryDigest, FileCount: len(records), ObservedState: "verified", PointOfUseRecheck: true}, nil
}

func rejectSymlinkComponents(path string) error {
	volume := filepath.VolumeName(path)
	rest := strings.TrimPrefix(path, volume)
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(rest, string(filepath.Separator)), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink component %s", current)
		}
	}
	return nil
}
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func classifyPiTreeError(err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return piError("pi_execution_identity_unavailable", err)
	}
	return piError("pi_execution_identity_mismatch", err)
}

func ValidatePiExecutionEnvironment(environ []string) error {
	seen := map[string]bool{}
	for _, item := range environ {
		name, _, ok := strings.Cut(item, "=")
		if !ok || name == "" {
			return piError("pi_execution_environment_invalid", errors.New("malformed environment entry"))
		}
		if seen[name] {
			return piError("pi_execution_environment_invalid", fmt.Errorf("duplicate environment name %q", name))
		}
		seen[name] = true
		if name == "HF_ENDPOINT" || name == "MODEL_ENDPOINT" || name == "GGML_BACKEND_PATH" || name == "LLAMA_API_KEY" {
			return piError("pi_execution_environment_invalid", fmt.Errorf("runtime-affecting environment name %q is denied", name))
		}
		upper := strings.ToUpper(name)
		for _, prefix := range []string{"DYLD_", "LD_", "NODE_", "BUN_", "LLAMA_ARG_"} {
			if strings.HasPrefix(upper, prefix) {
				return piError("pi_execution_environment_invalid", fmt.Errorf("runtime-affecting environment name %q is denied", name))
			}
		}
	}
	return nil
}

func piIdentityEqual(a, b PiExecutionIdentity) bool {
	return a.Compatibility == b.Compatibility && a.AssetSHA256 == b.AssetSHA256 && a.Host == b.Host && a.ReleaseRoot == b.ReleaseRoot && a.Entrypoint == b.Entrypoint && a.ManifestSHA256 == b.ManifestSHA256 && a.EntrypointSHA256 == b.EntrypointSHA256 && a.FileCount == b.FileCount && a.ObservedState == b.ObservedState && a.PointOfUseRecheck == b.PointOfUseRecheck
}

func piCatalogPaths() []string {
	records, _, _ := compiledPiManifest()
	paths := make([]string, len(records))
	for i, r := range records {
		paths[i] = r.Path
	}
	sort.Strings(paths)
	return paths
}
