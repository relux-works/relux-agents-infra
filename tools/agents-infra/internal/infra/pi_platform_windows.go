//go:build windows

package infra

import (
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const PiCompatibilityV0842DarwinARM64 = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65"

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

func VerifyPiExecutionIdentity(string, string) (PiExecutionIdentity, error) {
	return PiExecutionIdentity{}, piError("pi_compatibility_unsupported", errors.New("managed Pi profiles are supported only on darwin/arm64"))
}
func ValidatePiExecutionEnvironment([]string) error { return nil }

type PiStatePaths struct {
	CanonicalCacheRoot string `json:"canonical_cache_root"`
	ProjectStateKey    string `json:"project_state_key"`
	ProfileStateKey    string `json:"profile_state_key"`
	Root               string `json:"root"`
	AgentDir           string `json:"agent_dir"`
	SessionsDir        string `json:"sessions_dir"`
	ModelsJSON         string `json:"models_json"`
	Lock               string `json:"lock"`
}

func ValidatePiStateKeyCollisions(map[string]PiProfile) error { return nil }
func ResolvePiStatePaths(string, string, string) (PiStatePaths, error) {
	return PiStatePaths{}, piError("pi_compatibility_unsupported", errors.New("managed Pi profiles are supported only on darwin/arm64"))
}

type RunPiOptions struct {
	ProjectDir string
	HomeDir    string
	CacheRoot  string
	Args       []string
	Environ    []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	LookPath   func(string) (string, error)
	HTTPClient *http.Client
	Signals    <-chan os.Signal
}

func RunPi(opts RunPiOptions) error {
	project, err := CanonicalProjectDir(opts.ProjectDir)
	if err != nil {
		return err
	}
	if opts.HomeDir == "" {
		opts.HomeDir, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	if opts.Environ == nil {
		opts.Environ = os.Environ()
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}
	composite, err := loadCompositeProjectConfig(ancestorDirsRootFirst(project), filepath.Join(opts.HomeDir, ".agents", ".configs", projectConfigFileName))
	if err != nil {
		return piError("invalid_project_configuration", err)
	}
	override, err := ExtractPiProfileOverride(opts.Args)
	if err != nil {
		return err
	}
	managed := override != nil || composite.PiPrimarySession.Profile.Present
	if managed {
		return piError("pi_compatibility_unsupported", errors.New("managed Pi profiles are supported only on darwin/arm64"))
	}
	path, err := opts.LookPath("pi")
	if err != nil {
		return piError("provider_executable_not_found", err)
	}
	cmd := exec.Command(path, opts.Args...)
	cmd.Dir = project
	cmd.Env = opts.Environ
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd.Run()
}
