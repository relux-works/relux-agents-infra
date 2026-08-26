package modelharness

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	Contract      = "model-harness.launch-plan"
	SchemaVersion = 1
	loopbackHost  = "127.0.0.1"
)

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Document struct {
	Profiles map[string]Profile `toml:"profiles"`
}

type Profile struct {
	Mode             string   `toml:"mode"`
	Executable       string   `toml:"executable,omitempty"`
	Argv             []string `toml:"argv,omitempty"`
	SSHExecutable    string   `toml:"ssh_executable,omitempty"`
	SSHTarget        string   `toml:"ssh_target,omitempty"`
	RemoteExecutable string   `toml:"remote_executable,omitempty"`
	RemoteConfig     string   `toml:"remote_config,omitempty"`
	RemoteProfile    string   `toml:"remote_profile,omitempty"`
	RemoteHost       string   `toml:"remote_host,omitempty"`
	RemotePort       int      `toml:"remote_port,omitempty"`
}

type Plan struct {
	Contract      string      `json:"contract"`
	SchemaVersion int         `json:"schema_version"`
	Config        string      `json:"config"`
	Profile       string      `json:"profile"`
	Mode          string      `json:"mode"`
	Executable    string      `json:"executable"`
	Argv          []string    `json:"argv"`
	Endpoint      string      `json:"endpoint"`
	Remote        *RemotePlan `json:"remote,omitempty"`
}

type RemotePlan struct {
	Target     string `json:"target"`
	Executable string `json:"executable"`
	Config     string `json:"config,omitempty"`
	Profile    string `json:"profile"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
}

func DefaultConfigPath() (string, error) {
	if configured := os.Getenv("MODEL_HARNESS_CONFIG"); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", errors.New("MODEL_HARNESS_CONFIG must be an absolute path")
		}
		return filepath.Clean(configured), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "model-harness", "config.toml"), nil
}

func Resolve(configPath, profileName, host string, port int) (Plan, error) {
	if configPath == "" {
		var err error
		configPath, err = DefaultConfigPath()
		if err != nil {
			return Plan{}, err
		}
	}
	if !filepath.IsAbs(configPath) {
		return Plan{}, errors.New("config path must be absolute")
	}
	configPath = filepath.Clean(configPath)
	if !profileNamePattern.MatchString(profileName) {
		return Plan{}, fmt.Errorf("invalid profile name %q", profileName)
	}
	if host != loopbackHost {
		return Plan{}, fmt.Errorf("host must equal %s", loopbackHost)
	}
	if port < 1 || port > 65535 {
		return Plan{}, errors.New("port must be between 1 and 65535")
	}
	document, err := loadDocument(configPath)
	if err != nil {
		return Plan{}, err
	}
	profile, ok := document.Profiles[profileName]
	if !ok {
		return Plan{}, fmt.Errorf("unknown profile %q in %s", profileName, configPath)
	}
	profile = normalizeProfile(profile)
	if err := validateProfile(profileName, profile); err != nil {
		return Plan{}, err
	}
	plan := Plan{
		Contract:      Contract,
		SchemaVersion: SchemaVersion,
		Config:        configPath,
		Profile:       profileName,
		Mode:          profile.Mode,
		Endpoint:      "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/v1",
	}
	switch profile.Mode {
	case "local":
		plan.Executable = profile.Executable
		plan.Argv = substituteEndpoint(profile.Argv, host, port)
	case "ssh":
		plan.Executable = profile.SSHExecutable
		plan.Remote = &RemotePlan{
			Target:     profile.SSHTarget,
			Executable: profile.RemoteExecutable,
			Config:     profile.RemoteConfig,
			Profile:    profile.RemoteProfile,
			Host:       profile.RemoteHost,
			Port:       profile.RemotePort,
		}
		remoteRun := remoteCommand(profile.RemoteExecutable, "run", profile.RemoteProfile, profile.RemoteConfig, profile.RemoteHost, profile.RemotePort, false)
		forward := net.JoinHostPort(host, strconv.Itoa(port)) + ":" + net.JoinHostPort(profile.RemoteHost, strconv.Itoa(profile.RemotePort))
		plan.Argv = []string{
			"-tt",
			"-o", "BatchMode=yes",
			"-o", "ExitOnForwardFailure=yes",
			"-o", "ServerAliveInterval=15",
			"-o", "ServerAliveCountMax=3",
			"-L", forward,
			profile.SSHTarget,
			remoteRun,
		}
	default:
		return Plan{}, fmt.Errorf("profile %q has unsupported mode %q", profileName, profile.Mode)
	}
	return plan, nil
}

func EncodePlan(w io.Writer, plan Plan) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(plan)
}

func loadDocument(path string) (Document, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Document{}, fmt.Errorf("inspect config %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Document{}, fmt.Errorf("config %s must be a regular non-symlink file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return Document{}, fmt.Errorf("open config %s: %w", path, err)
	}
	defer file.Close()
	var document Document
	decoder := toml.NewDecoder(file).DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return Document{}, fmt.Errorf("decode config %s: %w", path, err)
	}
	if len(document.Profiles) == 0 {
		return Document{}, fmt.Errorf("config %s has no profiles", path)
	}
	return document, nil
}

func validateProfile(name string, profile Profile) error {
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q", name)
	}
	switch profile.Mode {
	case "local":
		if err := validateAbsoluteExecutable("executable", profile.Executable); err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		if len(profile.Argv) == 0 {
			return fmt.Errorf("profile %q: argv must not be empty", name)
		}
		hostCount, portCount := 0, 0
		for _, token := range profile.Argv {
			if token == "" || strings.IndexByte(token, 0) >= 0 {
				return fmt.Errorf("profile %q: argv tokens must be non-empty and NUL-free", name)
			}
			switch token {
			case "{host}":
				hostCount++
			case "{port}":
				portCount++
			default:
				if strings.Contains(token, "{host}") || strings.Contains(token, "{port}") {
					return fmt.Errorf("profile %q: endpoint placeholders must be whole argv tokens", name)
				}
			}
		}
		if hostCount != 1 || portCount != 1 {
			return fmt.Errorf("profile %q: argv must contain exactly one {host} and one {port} token", name)
		}
		if profile.SSHExecutable != "" || profile.SSHTarget != "" || profile.RemoteExecutable != "" || profile.RemoteConfig != "" || profile.RemoteProfile != "" || profile.RemoteHost != "" || profile.RemotePort != 0 {
			return fmt.Errorf("profile %q: local mode cannot declare remote fields", name)
		}
	case "ssh":
		if profile.Executable != "" || len(profile.Argv) != 0 {
			return fmt.Errorf("profile %q: ssh mode cannot declare executable or argv", name)
		}
		if profile.SSHExecutable == "" {
			profile.SSHExecutable = "/usr/bin/ssh"
		}
		if err := validateAbsoluteExecutable("ssh_executable", profile.SSHExecutable); err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		if !profileNamePattern.MatchString(profile.SSHTarget) {
			return fmt.Errorf("profile %q: ssh_target must be a simple host alias", name)
		}
		if err := validateAbsoluteExecutable("remote_executable", profile.RemoteExecutable); err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		if profile.RemoteConfig != "" && !filepath.IsAbs(profile.RemoteConfig) {
			return fmt.Errorf("profile %q: remote_config must be absolute", name)
		}
		if !profileNamePattern.MatchString(profile.RemoteProfile) {
			return fmt.Errorf("profile %q: invalid remote_profile", name)
		}
		if profile.RemoteHost != loopbackHost {
			return fmt.Errorf("profile %q: remote_host must equal %s", name, loopbackHost)
		}
		if profile.RemotePort < 1 || profile.RemotePort > 65535 {
			return fmt.Errorf("profile %q: remote_port must be between 1 and 65535", name)
		}
	default:
		return fmt.Errorf("profile %q: mode must equal local or ssh", name)
	}
	return nil
}

func normalizeProfile(profile Profile) Profile {
	if profile.Mode == "ssh" && profile.SSHExecutable == "" {
		profile.SSHExecutable = "/usr/bin/ssh"
	}
	return profile
}

func validateAbsoluteExecutable(field, value string) error {
	if value == "" || !filepath.IsAbs(value) || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("%s must be an absolute NUL-free path", field)
	}
	return nil
}

func substituteEndpoint(argv []string, host string, port int) []string {
	out := make([]string, len(argv))
	for index, token := range argv {
		switch token {
		case "{host}":
			out[index] = host
		case "{port}":
			out[index] = strconv.Itoa(port)
		default:
			out[index] = token
		}
	}
	return out
}

func remoteCommand(executable, action, profile, config, host string, port int, jsonOutput bool) string {
	argv := []string{executable, action, profile}
	if config != "" {
		argv = append(argv, "--config", config)
	}
	argv = append(argv, "--host", host, "--port", strconv.Itoa(port))
	if jsonOutput {
		argv = append(argv, "--json")
	}
	quoted := make([]string, len(argv))
	for index, token := range argv {
		quoted[index] = shellQuote(token)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
