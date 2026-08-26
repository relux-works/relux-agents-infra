package infra

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type PiPrimarySessionSource struct {
	Profile         *string
	PiCompatibility *string
	YoloMode        *bool
}

type PiPrimarySessionPolicy struct {
	Profile         PiPolicyStringValue
	PiCompatibility PiPolicyStringValue
	YoloMode        PiPolicyBoolValue
}

type PiPolicyStringValue struct {
	Value   string
	Source  string
	Present bool
}

type PiPolicyBoolValue struct {
	Value   bool
	Source  string
	Present bool
}

type PiProfile struct {
	Provider              string
	Model                 string
	BaseURL               string
	API                   string
	Reasoning             bool
	Input                 []string
	ContextWindow         int
	MaxTokens             int
	Thinking              string
	RequestedCapabilities []string
	Compat                PiCompat
	Runtime               PiRuntime
	Source                string
}

type PiCompat struct {
	SupportsDeveloperRole   *bool
	SupportsReasoningEffort *bool
	SupportsUsageStreaming  *bool
	SupportsFinishReason    *bool
	MaxTokensField          *string
	ThinkingFormat          *string
}

type PiRuntime struct {
	Executable             string
	Argv                   []string
	ReadinessPath          string
	StartupTimeoutSeconds  int
	ShutdownTimeoutSeconds int
	DFlash                 *PiDFlash
	Sharing                *PiRuntimeSharing
}

type PiRuntimeSharing struct {
	Mode                      string `json:"mode"`
	LingerSeconds             int    `json:"linger_seconds"`
	MaxLeases                 int    `json:"max_leases"`
	HeartbeatIntervalSeconds  int    `json:"heartbeat_interval_seconds"`
	LeaseStaleSeconds         int    `json:"lease_stale_seconds"`
	BrokerStartTimeoutSeconds int    `json:"broker_start_timeout_seconds"`
}

type PiDFlash struct {
	TargetModel string
	DraftModel  string
	TargetArgv  []string
	DraftArgv   []string
}

func parsePiConfig(agents map[string]any, path string) (PiPrimarySessionSource, map[string]PiProfile, error) {
	pi, present, err := projectConfigTable(agents, "pi", "agents.pi")
	if err != nil {
		return PiPrimarySessionSource{}, nil, projectConfigFieldError(path, "agents.pi", err)
	}
	if !present {
		return PiPrimarySessionSource{}, map[string]PiProfile{}, nil
	}
	if err := rejectUnknownFields(pi, "agents.pi", "primary_session", "profiles"); err != nil {
		return PiPrimarySessionSource{}, nil, projectConfigFieldError(path, errField(err), err)
	}

	var primary PiPrimarySessionSource
	primaryTable, primaryPresent, err := projectConfigTable(pi, "primary_session", piPrimarySessionField)
	if err != nil {
		return primary, nil, projectConfigFieldError(path, piPrimarySessionField, err)
	}
	if primaryPresent {
		if err := rejectUnknownFields(primaryTable, piPrimarySessionField, "profile", "pi_compatibility", "yolo_mode"); err != nil {
			return primary, nil, projectConfigFieldError(path, errField(err), err)
		}
		primary.Profile, err = optionalNonEmptyString(primaryTable, "profile")
		if err != nil {
			return primary, nil, projectConfigFieldError(path, piPrimarySessionField+".profile", err)
		}
		primary.PiCompatibility, err = optionalNonEmptyString(primaryTable, "pi_compatibility")
		if err != nil {
			return primary, nil, projectConfigFieldError(path, piPrimarySessionField+".pi_compatibility", err)
		}
		primary.YoloMode, err = optionalBool(primaryTable, "yolo_mode")
		if err != nil {
			return primary, nil, projectConfigFieldError(path, piPrimarySessionField+".yolo_mode", err)
		}
		if primary.Profile == nil && primary.PiCompatibility == nil && primary.YoloMode == nil {
			return primary, nil, projectConfigFieldError(path, piPrimarySessionField, errors.New("table must contain at least one supported field"))
		}
	}

	profiles := map[string]PiProfile{}
	profilesTable, profilesPresent, err := projectConfigTable(pi, "profiles", "agents.pi.profiles")
	if err != nil {
		return primary, nil, projectConfigFieldError(path, "agents.pi.profiles", err)
	}
	if !profilesPresent {
		return primary, profiles, nil
	}
	for name, raw := range profilesTable {
		if name == "" {
			return primary, nil, projectConfigFieldError(path, "agents.pi.profiles", errors.New("profile name must not be empty"))
		}
		table, ok := raw.(map[string]any)
		if !ok {
			return primary, nil, projectConfigFieldError(path, "agents.pi.profiles."+name, fmt.Errorf("expected table, got %T", raw))
		}
		profile, parseErr := parsePiProfile(table, path, name)
		if parseErr != nil {
			return primary, nil, parseErr
		}
		profiles[name] = profile
	}
	return primary, profiles, nil
}

func parsePiProfile(table map[string]any, path, name string) (PiProfile, error) {
	field := "agents.pi.profiles." + name
	if err := rejectUnknownFields(table, field,
		"provider", "model", "base_url", "api", "reasoning", "input", "context_window", "max_tokens", "thinking", "requested_capabilities", "compat", "runtime"); err != nil {
		return PiProfile{}, projectConfigFieldError(path, errField(err), err)
	}
	var p PiProfile
	var err error
	if p.Provider, err = requiredString(table, "provider"); err != nil {
		return p, projectConfigFieldError(path, field+".provider", err)
	}
	if err := validatePiProvider(p.Provider); err != nil {
		return p, projectConfigFieldError(path, field+".provider", err)
	}
	if p.Model, err = requiredString(table, "model"); err != nil {
		return p, projectConfigFieldError(path, field+".model", err)
	}
	if p.BaseURL, err = requiredString(table, "base_url"); err != nil {
		return p, projectConfigFieldError(path, field+".base_url", err)
	}
	if err := validatePiBaseURL(p.BaseURL); err != nil {
		return p, projectConfigFieldError(path, field+".base_url", err)
	}
	if p.API, err = requiredString(table, "api"); err != nil || p.API != "openai-completions" {
		if err == nil {
			err = errors.New("must equal openai-completions")
		}
		return p, projectConfigFieldError(path, field+".api", err)
	}
	if p.Reasoning, err = requiredBool(table, "reasoning"); err != nil {
		return p, projectConfigFieldError(path, field+".reasoning", err)
	}
	if p.Input, err = requiredStringArray(table, "input"); err != nil || !equalStrings(p.Input, []string{"text"}) {
		if err == nil {
			err = errors.New("must equal [\"text\"]")
		}
		return p, projectConfigFieldError(path, field+".input", err)
	}
	if p.ContextWindow, err = requiredPositiveInt(table, "context_window"); err != nil {
		return p, projectConfigFieldError(path, field+".context_window", err)
	}
	if p.MaxTokens, err = requiredPositiveInt(table, "max_tokens"); err != nil || p.MaxTokens > p.ContextWindow {
		if err == nil {
			err = errors.New("must not exceed context_window")
		}
		return p, projectConfigFieldError(path, field+".max_tokens", err)
	}
	if p.Thinking, err = requiredString(table, "thinking"); err != nil {
		return p, projectConfigFieldError(path, field+".thinking", err)
	}
	if !piThinkingLevels[p.Thinking] {
		return p, projectConfigFieldError(path, field+".thinking", errors.New("must be a documented Pi thinking level"))
	}
	if !p.Reasoning && p.Thinking != "off" {
		return p, projectConfigFieldError(path, field+".reasoning", errors.New("must be true when thinking is not off"))
	}
	if p.RequestedCapabilities, err = requiredStringArray(table, "requested_capabilities"); err != nil {
		return p, projectConfigFieldError(path, field+".requested_capabilities", err)
	}
	compatTable, present, tableErr := projectConfigTable(table, "compat", field+".compat")
	if tableErr != nil || !present {
		if tableErr == nil {
			tableErr = errors.New("required table is absent")
		}
		return p, projectConfigFieldError(path, field+".compat", tableErr)
	}
	if p.Compat, err = parsePiCompat(compatTable, field+".compat"); err != nil {
		return p, projectConfigFieldError(path, errField(err), err)
	}
	runtimeTable, present, tableErr := projectConfigTable(table, "runtime", field+".runtime")
	if tableErr != nil || !present {
		if tableErr == nil {
			tableErr = errors.New("required table is absent")
		}
		return p, projectConfigFieldError(path, field+".runtime", tableErr)
	}
	if p.Runtime, err = parsePiRuntime(runtimeTable, field+".runtime"); err != nil {
		return p, projectConfigFieldError(path, errField(err), err)
	}
	if err := validatePiRuntimeEndpointArgv(p.Runtime.Argv, p.BaseURL); err != nil {
		return p, projectConfigFieldError(path, field+".runtime.argv", err)
	}
	wantCaps := []string{"text", "tools"}
	if p.Runtime.DFlash != nil {
		wantCaps = []string{"dflash", "text", "tools"}
	}
	if !equalStrings(p.RequestedCapabilities, wantCaps) {
		return p, projectConfigFieldError(path, field+".requested_capabilities", fmt.Errorf("must equal %q", wantCaps))
	}
	if p.Runtime.DFlash != nil && p.Runtime.DFlash.TargetModel != p.Model {
		return p, projectConfigFieldError(path, field+".runtime.dflash.target_model", errors.New("must equal profile model"))
	}
	return p, nil
}

func parsePiCompat(table map[string]any, field string) (PiCompat, error) {
	if err := rejectUnknownFields(table, field, "supports_developer_role", "supports_reasoning_effort", "supports_usage_in_streaming", "supports_finish_reason", "max_tokens_field", "thinking_format"); err != nil {
		return PiCompat{}, err
	}
	var c PiCompat
	var err error
	if c.SupportsDeveloperRole, err = optionalBool(table, "supports_developer_role"); err != nil {
		return c, fieldError(field+".supports_developer_role", err)
	}
	if c.SupportsReasoningEffort, err = optionalBool(table, "supports_reasoning_effort"); err != nil {
		return c, fieldError(field+".supports_reasoning_effort", err)
	}
	if c.SupportsUsageStreaming, err = optionalBool(table, "supports_usage_in_streaming"); err != nil {
		return c, fieldError(field+".supports_usage_in_streaming", err)
	}
	if c.SupportsFinishReason, err = optionalBool(table, "supports_finish_reason"); err != nil {
		return c, fieldError(field+".supports_finish_reason", err)
	}
	if c.MaxTokensField, err = optionalNonEmptyString(table, "max_tokens_field"); err != nil {
		return c, fieldError(field+".max_tokens_field", err)
	}
	if c.ThinkingFormat, err = optionalNonEmptyString(table, "thinking_format"); err != nil {
		return c, fieldError(field+".thinking_format", err)
	}
	return c, nil
}

func parsePiRuntime(table map[string]any, field string) (PiRuntime, error) {
	if err := rejectUnknownFields(table, field, "executable", "argv", "readiness_path", "startup_timeout_seconds", "shutdown_timeout_seconds", "dflash", "sharing"); err != nil {
		return PiRuntime{}, err
	}
	var r PiRuntime
	var err error
	if r.Executable, err = requiredString(table, "executable"); err != nil {
		return r, fieldError(field+".executable", err)
	}
	if strings.IndexByte(r.Executable, 0) >= 0 || !filepath.IsAbs(r.Executable) {
		return r, fieldError(field+".executable", errors.New("must be an absolute NUL-free path"))
	}
	if r.Argv, err = requiredStringArray(table, "argv"); err != nil || len(r.Argv) == 0 {
		if err == nil {
			err = errors.New("must not be empty")
		}
		return r, fieldError(field+".argv", err)
	}
	for _, token := range r.Argv {
		if token == "" || strings.IndexByte(token, 0) >= 0 {
			return r, fieldError(field+".argv", errors.New("tokens must be non-empty and NUL-free"))
		}
	}
	if r.ReadinessPath, err = requiredString(table, "readiness_path"); err != nil || r.ReadinessPath != "/models" {
		if err == nil {
			err = errors.New("must equal /models")
		}
		return r, fieldError(field+".readiness_path", err)
	}
	if r.StartupTimeoutSeconds, err = requiredPositiveInt(table, "startup_timeout_seconds"); err != nil {
		return r, fieldError(field+".startup_timeout_seconds", err)
	}
	if r.ShutdownTimeoutSeconds, err = requiredPositiveInt(table, "shutdown_timeout_seconds"); err != nil {
		return r, fieldError(field+".shutdown_timeout_seconds", err)
	}
	if raw, ok := table["dflash"]; ok {
		dt, ok := raw.(map[string]any)
		if !ok {
			return r, fieldError(field+".dflash", fmt.Errorf("expected table, got %T", raw))
		}
		d, parseErr := parsePiDFlash(dt, field+".dflash", r.Argv)
		if parseErr != nil {
			return r, parseErr
		}
		r.DFlash = &d
	}
	if raw, ok := table["sharing"]; ok {
		sharingTable, ok := raw.(map[string]any)
		if !ok {
			return r, fieldError(field+".sharing", fmt.Errorf("expected table, got %T", raw))
		}
		sharing, parseErr := parsePiRuntimeSharing(sharingTable, field+".sharing", r)
		if parseErr != nil {
			return r, parseErr
		}
		r.Sharing = &sharing
	}
	return r, nil
}

func parsePiRuntimeSharing(table map[string]any, field string, runtime PiRuntime) (PiRuntimeSharing, error) {
	if err := rejectUnknownFields(table, field, "mode", "linger_seconds", "max_leases", "heartbeat_interval_seconds", "lease_stale_seconds", "broker_start_timeout_seconds"); err != nil {
		return PiRuntimeSharing{}, err
	}
	var sharing PiRuntimeSharing
	var err error
	if sharing.Mode, err = requiredString(table, "mode"); err != nil {
		return sharing, fieldError(field+".mode", err)
	}
	if sharing.Mode != "exclusive" && sharing.Mode != "shared" {
		return sharing, fieldError(field+".mode", errors.New("must equal exclusive or shared"))
	}
	if sharing.LingerSeconds, err = requiredNonNegativeInt(table, "linger_seconds"); err != nil {
		return sharing, fieldError(field+".linger_seconds", err)
	}
	if sharing.MaxLeases, err = requiredPositiveInt(table, "max_leases"); err != nil {
		return sharing, fieldError(field+".max_leases", err)
	}
	if sharing.HeartbeatIntervalSeconds, err = requiredPositiveInt(table, "heartbeat_interval_seconds"); err != nil {
		return sharing, fieldError(field+".heartbeat_interval_seconds", err)
	}
	if sharing.LeaseStaleSeconds, err = requiredPositiveInt(table, "lease_stale_seconds"); err != nil {
		return sharing, fieldError(field+".lease_stale_seconds", err)
	}
	if sharing.HeartbeatIntervalSeconds >= sharing.LeaseStaleSeconds {
		return sharing, fieldError(field+".heartbeat_interval_seconds", errors.New("must be less than lease_stale_seconds"))
	}
	if sharing.BrokerStartTimeoutSeconds, err = requiredPositiveInt(table, "broker_start_timeout_seconds"); err != nil {
		return sharing, fieldError(field+".broker_start_timeout_seconds", err)
	}
	minimum := runtime.StartupTimeoutSeconds + runtime.ShutdownTimeoutSeconds + 30
	if sharing.BrokerStartTimeoutSeconds < minimum {
		return sharing, fieldError(field+".broker_start_timeout_seconds", fmt.Errorf("must be at least runtime startup + shutdown + 30 seconds (%d)", minimum))
	}
	return sharing, nil
}

func parsePiDFlash(table map[string]any, field string, runtimeArgv []string) (PiDFlash, error) {
	if err := rejectUnknownFields(table, field, "target_model", "draft_model", "target_argv", "draft_argv"); err != nil {
		return PiDFlash{}, err
	}
	var d PiDFlash
	var err error
	if d.TargetModel, err = requiredString(table, "target_model"); err != nil {
		return d, fieldError(field+".target_model", err)
	}
	if d.DraftModel, err = requiredString(table, "draft_model"); err != nil {
		return d, fieldError(field+".draft_model", err)
	}
	if d.TargetArgv, err = requiredStringArray(table, "target_argv"); err != nil || len(d.TargetArgv) == 0 {
		if err == nil {
			err = errors.New("must not be empty")
		}
		return d, fieldError(field+".target_argv", err)
	}
	if d.DraftArgv, err = requiredStringArray(table, "draft_argv"); err != nil || len(d.DraftArgv) == 0 {
		if err == nil {
			err = errors.New("must not be empty")
		}
		return d, fieldError(field+".draft_argv", err)
	}
	if d.TargetArgv[len(d.TargetArgv)-1] != d.TargetModel {
		return d, fieldError(field+".target_argv", errors.New("terminal token must equal target_model"))
	}
	if d.DraftArgv[len(d.DraftArgv)-1] != d.DraftModel {
		return d, fieldError(field+".draft_argv", errors.New("terminal token must equal draft_model"))
	}
	ti, tc := exactSubsequence(runtimeArgv, d.TargetArgv)
	di, dc := exactSubsequence(runtimeArgv, d.DraftArgv)
	if tc != 1 || dc != 1 {
		return d, fieldError(field, errors.New("target_argv and draft_argv must each occur exactly once in runtime.argv"))
	}
	if ti < di+len(d.DraftArgv) && di < ti+len(d.TargetArgv) {
		return d, fieldError(field, errors.New("target_argv and draft_argv must not overlap"))
	}
	return d, nil
}

func validatePiBaseURL(raw string) error {
	_, _, err := piBaseURLEndpoint(raw)
	return err
}

func piBaseURLEndpoint(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/v1" || u.RawPath != "" {
		return "", "", errors.New("must be exact http://127.0.0.1:<port>/v1 without user info, query, fragment, or path encoding")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || host != "127.0.0.1" {
		return "", "", errors.New("must use exact IPv4 loopback with an explicit port")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
		return "", "", errors.New("must use a canonical non-zero explicit port")
	}
	if raw != "http://127.0.0.1:"+port+"/v1" {
		return "", "", errors.New("URL normalization is not permitted")
	}
	return host, port, nil
}

func validatePiRuntimeEndpointArgv(argv []string, baseURL string) error {
	host, port, err := piBaseURLEndpoint(baseURL)
	if err != nil {
		return fmt.Errorf("cannot bind runtime argv to invalid base_url: %w", err)
	}
	hostFlags, portFlags := 0, 0
	for i, token := range argv {
		switch {
		case token == "--host":
			hostFlags++
			if i+1 >= len(argv) || argv[i+1] != host {
				return fmt.Errorf("--host must be followed by exact base_url host %q", host)
			}
		case strings.HasPrefix(token, "--host="):
			return errors.New("--host must use one exact spaced token pair")
		case token == "--port":
			portFlags++
			if i+1 >= len(argv) || argv[i+1] != port {
				return fmt.Errorf("--port must be followed by exact base_url port %q", port)
			}
		case strings.HasPrefix(token, "--port="):
			return errors.New("--port must use one exact spaced token pair")
		}
	}
	if hostFlags != 1 || portFlags != 1 {
		return fmt.Errorf("must contain exactly one --host %s and one --port %s pair", host, port)
	}
	return nil
}

func validatePiProvider(value string) error {
	if value == "" {
		return errors.New("must not be empty")
	}
	for _, r := range value {
		if unicode.IsSpace(r) || strings.ContainsRune("/:*?[]\\", r) || strings.ContainsRune("∕⁄／⧸∖＼﹨：∶", r) {
			return errors.New("contains a separator, glob, whitespace, or Unicode lookalike")
		}
	}
	return nil
}

type piFieldError struct {
	field string
	err   error
}

func (e *piFieldError) Error() string          { return e.err.Error() }
func (e *piFieldError) Unwrap() error          { return e.err }
func fieldError(field string, err error) error { return &piFieldError{field: field, err: err} }
func errField(err error) string {
	var fe *piFieldError
	if errors.As(err, &fe) {
		return fe.field
	}
	return "agents.pi"
}
func rejectUnknownFields(table map[string]any, field string, allowed ...string) error {
	ok := map[string]bool{}
	for _, key := range allowed {
		ok[key] = true
	}
	var unknown []string
	for key := range table {
		if !ok[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return fieldError(field+"."+unknown[0], errors.New("unsupported field"))
	}
	return nil
}
func requiredString(table map[string]any, key string) (string, error) {
	p, err := optionalNonEmptyString(table, key)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", errors.New("required field is absent")
	}
	return *p, nil
}
func optionalNonEmptyString(table map[string]any, key string) (*string, error) {
	v, ok := table[key]
	if !ok {
		return nil, nil
	}
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", v)
	}
	if s == "" {
		return nil, errors.New("must not be empty")
	}
	return &s, nil
}
func requiredBool(table map[string]any, key string) (bool, error) {
	p, err := optionalBool(table, key)
	if err != nil {
		return false, err
	}
	if p == nil {
		return false, errors.New("required field is absent")
	}
	return *p, nil
}
func optionalBool(table map[string]any, key string) (*bool, error) {
	v, ok := table[key]
	if !ok {
		return nil, nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil, fmt.Errorf("expected boolean, got %T", v)
	}
	return &b, nil
}
func requiredPositiveInt(table map[string]any, key string) (int, error) {
	v, ok := table[key]
	if !ok {
		return 0, errors.New("required field is absent")
	}
	n, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("expected integer, got %T", v)
	}
	if n <= 0 || int64(int(n)) != n {
		return 0, errors.New("must be a positive platform integer")
	}
	return int(n), nil
}
func requiredNonNegativeInt(table map[string]any, key string) (int, error) {
	v, ok := table[key]
	if !ok {
		return 0, errors.New("required field is absent")
	}
	n, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("expected integer, got %T", v)
	}
	if n < 0 || int64(int(n)) != n {
		return 0, errors.New("must be a non-negative platform integer")
	}
	return int(n), nil
}
func requiredStringArray(table map[string]any, key string) ([]string, error) {
	v, ok := table[key]
	if !ok {
		return nil, errors.New("required field is absent")
	}
	a, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array of strings, got %T", v)
	}
	out := make([]string, len(a))
	for i, item := range a {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("element %d is %T, want string", i, item)
		}
		out[i] = s
	}
	return out, nil
}
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func exactSubsequence(haystack, needle []string) (int, int) {
	first, count := -1, 0
	if len(needle) == 0 || len(needle) > len(haystack) {
		return first, count
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if equalStrings(haystack[i:i+len(needle)], needle) {
			if first < 0 {
				first = i
			}
			count++
		}
	}
	return first, count
}

func composePiPrimarySession(policy *PiPrimarySessionPolicy, source PiPrimarySessionSource, path string) {
	if source.Profile != nil {
		policy.Profile = PiPolicyStringValue{Value: *source.Profile, Source: path, Present: true}
	}
	if source.PiCompatibility != nil {
		policy.PiCompatibility = PiPolicyStringValue{Value: *source.PiCompatibility, Source: path, Present: true}
	}
	if source.YoloMode != nil {
		policy.YoloMode = PiPolicyBoolValue{Value: *source.YoloMode, Source: path, Present: true}
	}
}
func clonePiPrimarySessionSource(s PiPrimarySessionSource) PiPrimarySessionSource {
	return PiPrimarySessionSource{Profile: cloneStringPointer(s.Profile), PiCompatibility: cloneStringPointer(s.PiCompatibility), YoloMode: cloneBoolPointer(s.YoloMode)}
}
func clonePiProfiles(in map[string]PiProfile) map[string]PiProfile {
	out := map[string]PiProfile{}
	for k, v := range in {
		v.Input = append([]string(nil), v.Input...)
		v.RequestedCapabilities = append([]string(nil), v.RequestedCapabilities...)
		v.Runtime.Argv = append([]string(nil), v.Runtime.Argv...)
		if v.Runtime.Sharing != nil {
			sharing := *v.Runtime.Sharing
			v.Runtime.Sharing = &sharing
		}
		out[k] = v
	}
	return out
}

func resolvedPiYolo(policy PiPrimarySessionPolicy) PrimarySessionResolvedBool {
	if policy.YoloMode.Present {
		return PrimarySessionResolvedBool{Value: policy.YoloMode.Value, Source: policy.YoloMode.Source}
	}
	return PrimarySessionResolvedBool{Value: false, Source: "default"}
}
