package infra

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	projectConfigParseField          = "project_config"
	codexPrimarySessionField         = "agents.codex.primary_session"
	codexPrimaryModelField           = codexPrimarySessionField + ".model"
	codexPrimaryReasoningEffortField = codexPrimarySessionField + ".reasoning_effort"
	codexPrimaryYoloModeField        = codexPrimarySessionField + ".yolo_mode"
	claudePrimarySessionField        = "agents.claude.primary_session"
	claudePrimaryModelField          = claudePrimarySessionField + ".model"
	claudePrimaryYoloModeField       = claudePrimarySessionField + ".yolo_mode"
	piPrimarySessionField            = "agents.pi.primary_session"
	piStandaloneSessionField         = "agents.pi.standalone_session"
	targetsField                     = "agents.targets"
	entrypointsField                 = "agents.entrypoints"
)

var canonicalEntrypointVendors = map[string]string{
	"openai-infra":    "openai",
	"anthropic-infra": "anthropic",
	"qwen-infra":      "qwen",
}

// ProjectTarget is one atomic canonical vendor/environment/model identity.
// Source identifies the single project config that supplied the whole
// definition; target fields never merge across ancestors.
type ProjectTarget struct {
	Name            string
	Source          string
	Vendor          string
	Environment     string
	Model           string
	Reasoning       string
	Profile         *string
	ProfileProvider *string
	Endpoint        *string
}

type ProjectEntrypoint struct {
	Name       string
	TargetName string
	Source     string
}

// CodexPrimarySessionPolicy is the root-to-leaf composition of all discovered
// [agents.codex.primary_session] tables. Present distinguishes an omitted field
// from an explicitly configured zero value, notably yolo_mode=false.
type CodexPrimarySessionPolicy struct {
	Model           CodexPrimarySessionStringValue
	ReasoningEffort CodexPrimarySessionStringValue
	YoloMode        CodexPrimarySessionBoolValue
}

type CodexPrimarySessionStringValue struct {
	Value   string
	Source  string
	Present bool
}

type CodexPrimarySessionBoolValue struct {
	Value   bool
	Source  string
	Present bool
}

// CodexPrimarySessionSource preserves the fields contributed by one project
// config. Pointer presence is intentional so false remains distinguishable
// from an omitted yolo_mode.
type CodexPrimarySessionSource struct {
	Model           *string
	ReasoningEffort *string
	YoloMode        *bool
}

// ClaudePrimarySessionPolicy is the root-to-leaf composition of all discovered
// [agents.claude.primary_session] tables. Present distinguishes an omitted
// field from an explicitly configured zero value, notably yolo_mode=false.
type ClaudePrimarySessionPolicy struct {
	Model    ClaudePrimarySessionStringValue
	YoloMode ClaudePrimarySessionBoolValue
}

type ClaudePrimarySessionStringValue struct {
	Value   string
	Source  string
	Present bool
}

type ClaudePrimarySessionBoolValue struct {
	Value   bool
	Source  string
	Present bool
}

// ClaudePrimarySessionSource preserves the fields contributed by one project
// config. Pointer presence distinguishes an omitted field from an explicitly
// configured value, notably yolo_mode=false.
type ClaudePrimarySessionSource struct {
	Model    *string
	YoloMode *bool
}

type parsedProjectConfig struct {
	EnabledMCPServers    []string
	PrimarySession       CodexPrimarySessionSource
	ClaudePrimarySession ClaudePrimarySessionSource
	PiPrimarySession     PiPrimarySessionSource
	PiStandaloneSession  PiStandaloneSessionSource
	PiProfiles           map[string]PiProfile
	Targets              map[string]ProjectTarget
	Entrypoints          map[string]string
}

// ProjectConfigSource records all policy contributed by one project config.
// Launcher-specific plans copy only their provider's policy from this source.
type ProjectConfigSource struct {
	Path                 string
	EnabledServers       []string
	CodexPrimarySession  CodexPrimarySessionSource
	ClaudePrimarySession ClaudePrimarySessionSource
	PiPrimarySession     PiPrimarySessionSource
	PiStandaloneSession  PiStandaloneSessionSource
	PiProfiles           map[string]PiProfile
	Targets              map[string]ProjectTarget
	Entrypoints          map[string]string
}

type compositeProjectConfig struct {
	EnabledOrder         []string
	EnabledBy            map[string][]string
	Sources              []ProjectConfigSource
	PrimarySession       CodexPrimarySessionPolicy
	ClaudePrimarySession ClaudePrimarySessionPolicy
	PiPrimarySession     PiPrimarySessionPolicy
	PiStandaloneSession  PiStandaloneSessionPolicy
	PiProfiles           map[string]PiProfile
	Targets              map[string]ProjectTarget
	Entrypoints          map[string]ProjectEntrypoint
}

func loadCompositeProjectConfig(ancestors []string, globalProjectConfigPath string) (compositeProjectConfig, error) {
	composite := compositeProjectConfig{
		EnabledBy:   map[string][]string{},
		PiProfiles:  map[string]PiProfile{},
		Targets:     map[string]ProjectTarget{},
		Entrypoints: map[string]ProjectEntrypoint{},
	}
	enabledSeen := map[string]bool{}

	for _, dir := range ancestors {
		path := filepath.Join(dir, ".agents", ".configs", projectConfigFileName)
		if globalProjectConfigPath != "" && samePath(path, globalProjectConfigPath) {
			continue
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return compositeProjectConfig{}, projectConfigFieldError(path, projectConfigParseField, fmt.Errorf("read project config: %w", err))
		}
		config, err := parseProjectConfig(data, path)
		if err != nil {
			return compositeProjectConfig{}, err
		}

		composite.Sources = append(composite.Sources, ProjectConfigSource{
			Path:                 path,
			EnabledServers:       append([]string(nil), config.EnabledMCPServers...),
			CodexPrimarySession:  cloneCodexPrimarySessionSource(config.PrimarySession),
			ClaudePrimarySession: cloneClaudePrimarySessionSource(config.ClaudePrimarySession),
			PiPrimarySession:     clonePiPrimarySessionSource(config.PiPrimarySession),
			PiStandaloneSession:  clonePiStandaloneSessionSource(config.PiStandaloneSession),
			PiProfiles:           clonePiProfiles(config.PiProfiles),
			Targets:              cloneProjectTargets(config.Targets),
			Entrypoints:          cloneStringMap(config.Entrypoints),
		})
		composeCodexPrimarySession(&composite.PrimarySession, config.PrimarySession, path)
		composeClaudePrimarySession(&composite.ClaudePrimarySession, config.ClaudePrimarySession, path)
		composePiPrimarySession(&composite.PiPrimarySession, config.PiPrimarySession, path)
		composePiStandaloneSession(&composite.PiStandaloneSession, config.PiStandaloneSession, path)
		for name, profile := range config.PiProfiles {
			profile.Source = path
			composite.PiProfiles[name] = profile
		}
		for name, target := range config.Targets {
			target.Source = path
			composite.Targets[name] = target
		}
		for name, targetName := range config.Entrypoints {
			composite.Entrypoints[name] = ProjectEntrypoint{Name: name, TargetName: targetName, Source: path}
		}

		for _, name := range config.EnabledMCPServers {
			if !isBareTOMLKey(name) {
				return compositeProjectConfig{}, fmt.Errorf("MCP server name %q in %s is not a supported TOML bare key", name, path)
			}
			if !enabledSeen[name] {
				composite.EnabledOrder = append(composite.EnabledOrder, name)
				enabledSeen[name] = true
			}
			composite.EnabledBy[name] = append(composite.EnabledBy[name], path)
		}
	}

	return composite, nil
}

func parseProjectConfig(data []byte, path string) (parsedProjectConfig, error) {
	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, projectConfigParseField, fmt.Errorf("parse TOML (including field %s and %s): %w", codexPrimarySessionField, claudePrimarySessionField, err))
	}

	var config parsedProjectConfig
	mcp, present, err := projectConfigTable(document, "mcp", "mcp")
	if err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, "mcp", err)
	}
	if present {
		config.EnabledMCPServers, err = projectConfigStringArray(mcp, "enabled_servers")
		if err != nil {
			return parsedProjectConfig{}, projectConfigFieldError(path, "mcp.enabled_servers", err)
		}
	}

	agents, present, err := projectConfigTable(document, "agents", "agents")
	if err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, "agents", err)
	}
	if !present {
		return config, nil
	}
	config.PrimarySession, err = parseCodexPrimarySession(agents, path)
	if err != nil {
		return parsedProjectConfig{}, err
	}
	config.ClaudePrimarySession, err = parseClaudePrimarySession(agents, path)
	if err != nil {
		return parsedProjectConfig{}, err
	}
	config.PiPrimarySession, config.PiStandaloneSession, config.PiProfiles, err = parsePiConfig(agents, path)
	if err != nil {
		return parsedProjectConfig{}, err
	}
	config.Targets, err = parseProjectTargets(agents, path)
	if err != nil {
		return parsedProjectConfig{}, err
	}
	config.Entrypoints, err = parseProjectEntrypoints(agents, path)
	if err != nil {
		return parsedProjectConfig{}, err
	}
	return config, nil
}

func parseProjectTargets(agents map[string]any, path string) (map[string]ProjectTarget, error) {
	targets := map[string]ProjectTarget{}
	table, present, err := projectConfigTable(agents, "targets", targetsField)
	if err != nil {
		return nil, projectConfigFieldError(path, targetsField, err)
	}
	if !present {
		return targets, nil
	}
	for name, raw := range table {
		field := targetsField + "." + name
		if name == "" {
			return nil, projectConfigFieldError(path, targetsField, errors.New("target name must not be empty"))
		}
		definition, ok := raw.(map[string]any)
		if !ok {
			return nil, projectConfigFieldError(path, field, fmt.Errorf("expected table, got %T", raw))
		}
		if err := rejectUnknownFields(definition, field, "vendor", "environment", "model", "reasoning", "profile", "profile_provider", "endpoint"); err != nil {
			return nil, projectConfigFieldError(path, errField(err), err)
		}
		target := ProjectTarget{Name: name}
		if target.Vendor, err = requiredTargetString(definition, "vendor", false); err != nil {
			return nil, projectConfigFieldError(path, field+".vendor", err)
		}
		if target.Environment, err = requiredTargetString(definition, "environment", false); err != nil {
			return nil, projectConfigFieldError(path, field+".environment", err)
		}
		if target.Model, err = requiredTargetString(definition, "model", true); err != nil {
			return nil, projectConfigFieldError(path, field+".model", err)
		}
		if target.Reasoning, err = requiredTargetString(definition, "reasoning", true); err != nil {
			return nil, projectConfigFieldError(path, field+".reasoning", err)
		}
		if target.Profile, err = optionalExactNonEmptyString(definition, "profile"); err != nil {
			return nil, projectConfigFieldError(path, field+".profile", err)
		}
		if target.ProfileProvider, err = optionalExactNonEmptyString(definition, "profile_provider"); err != nil {
			return nil, projectConfigFieldError(path, field+".profile_provider", err)
		}
		if target.Endpoint, err = optionalExactNonEmptyString(definition, "endpoint"); err != nil {
			return nil, projectConfigFieldError(path, field+".endpoint", err)
		}
		if err := validateProjectTarget(target); err != nil {
			var fieldErr *piFieldError
			if errors.As(err, &fieldErr) {
				return nil, projectConfigFieldError(path, field+"."+fieldErr.field, fieldErr.err)
			}
			return nil, projectConfigFieldError(path, field, err)
		}
		targets[name] = target
	}
	return targets, nil
}

func parseProjectEntrypoints(agents map[string]any, path string) (map[string]string, error) {
	entrypoints := map[string]string{}
	table, present, err := projectConfigTable(agents, "entrypoints", entrypointsField)
	if err != nil {
		return nil, projectConfigFieldError(path, entrypointsField, err)
	}
	if !present {
		return entrypoints, nil
	}
	for name, raw := range table {
		field := entrypointsField + "." + name
		if _, ok := canonicalEntrypointVendors[name]; !ok {
			return nil, projectConfigFieldError(path, field, errors.New("unsupported entrypoint"))
		}
		value, ok := raw.(string)
		if !ok {
			return nil, projectConfigFieldError(path, field, fmt.Errorf("expected string, got %T", raw))
		}
		if value == "" {
			return nil, projectConfigFieldError(path, field, errors.New("must not be empty"))
		}
		entrypoints[name] = value
	}
	return entrypoints, nil
}

func requiredTargetString(table map[string]any, key string, rejectWhitespaceOnly bool) (string, error) {
	value, present := table[key]
	if !present {
		return "", errors.New("required field is absent")
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", value)
	}
	if stringValue == "" || (rejectWhitespaceOnly && strings.TrimSpace(stringValue) == "") {
		return "", errors.New("must be a non-empty string")
	}
	return stringValue, nil
}

func optionalExactNonEmptyString(table map[string]any, key string) (*string, error) {
	value, present := table[key]
	if !present {
		return nil, nil
	}
	stringValue, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", value)
	}
	if stringValue == "" {
		return nil, errors.New("must not be empty")
	}
	return &stringValue, nil
}

func validateProjectTarget(target ProjectTarget) error {
	if !containsString([]string{"openai", "anthropic", "qwen"}, target.Vendor) {
		return fieldError("vendor", errors.New("must be one of openai, anthropic, qwen"))
	}
	if !containsString([]string{"codex", "claude-code", "pi"}, target.Environment) {
		return fieldError("environment", errors.New("must be one of codex, claude-code, pi"))
	}
	profileForbidden := func() error {
		switch {
		case target.Profile != nil:
			return fieldError("profile", errors.New("is forbidden for hosted targets"))
		case target.ProfileProvider != nil:
			return fieldError("profile_provider", errors.New("is forbidden for hosted targets"))
		case target.Endpoint != nil:
			return fieldError("endpoint", errors.New("is forbidden for hosted targets"))
		}
		return nil
	}
	switch {
	case target.Vendor == "openai" && target.Environment == "codex":
		return profileForbidden()
	case target.Vendor == "anthropic" && target.Environment == "claude-code":
		if !containsString(claudeEffortValues, target.Reasoning) {
			return fieldError("reasoning", errors.New("must be one of low, medium, high, xhigh, max"))
		}
		return profileForbidden()
	case target.Vendor == "qwen" && target.Environment == "pi":
		if !piThinkingLevels[target.Reasoning] {
			return fieldError("reasoning", errors.New("must be a documented Pi thinking level"))
		}
		if target.Profile == nil {
			return fieldError("profile", errors.New("required field is absent"))
		}
		if target.Endpoint != nil {
			parsed, err := url.Parse(*target.Endpoint)
			if err != nil || !parsed.IsAbs() || parsed.Host == "" {
				return fieldError("endpoint", errors.New("must be an absolute URL with a host"))
			}
		}
		return nil
	default:
		return fieldError("environment", fmt.Errorf("vendor/environment pair %s/%s is not admitted", target.Vendor, target.Environment))
	}
}

func parseCodexPrimarySession(agents map[string]any, path string) (CodexPrimarySessionSource, error) {
	codex, present, err := projectConfigTable(agents, "codex", "agents.codex")
	if err != nil {
		return CodexPrimarySessionSource{}, projectConfigFieldError(path, "agents.codex", err)
	}
	if !present {
		return CodexPrimarySessionSource{}, nil
	}
	primary, present, err := projectConfigTable(codex, "primary_session", codexPrimarySessionField)
	if err != nil {
		return CodexPrimarySessionSource{}, projectConfigFieldError(path, codexPrimarySessionField, err)
	}
	if !present {
		return CodexPrimarySessionSource{}, nil
	}

	var source CodexPrimarySessionSource
	source.Model, err = projectConfigNonEmptyString(primary, "model")
	if err != nil {
		return CodexPrimarySessionSource{}, projectConfigFieldError(path, codexPrimaryModelField, err)
	}
	source.ReasoningEffort, err = projectConfigNonEmptyString(primary, "reasoning_effort")
	if err != nil {
		return CodexPrimarySessionSource{}, projectConfigFieldError(path, codexPrimaryReasoningEffortField, err)
	}
	source.YoloMode, err = projectConfigBool(primary, "yolo_mode")
	if err != nil {
		return CodexPrimarySessionSource{}, projectConfigFieldError(path, codexPrimaryYoloModeField, err)
	}

	var unsupported []string
	for key := range primary {
		switch key {
		case "model", "reasoning_effort", "yolo_mode":
		default:
			unsupported = append(unsupported, key)
		}
	}
	if len(unsupported) > 0 {
		sort.Strings(unsupported)
		return CodexPrimarySessionSource{}, projectConfigFieldError(
			path,
			codexPrimarySessionField+"."+unsupported[0],
			errors.New("unsupported field"),
		)
	}
	if !codexPrimarySessionSourcePresent(source) {
		return CodexPrimarySessionSource{}, projectConfigFieldError(
			path,
			codexPrimarySessionField,
			errors.New("table must contain at least one supported field"),
		)
	}
	return source, nil
}

func parseClaudePrimarySession(agents map[string]any, path string) (ClaudePrimarySessionSource, error) {
	claude, present, err := projectConfigTable(agents, "claude", "agents.claude")
	if err != nil {
		return ClaudePrimarySessionSource{}, projectConfigFieldError(path, "agents.claude", err)
	}
	if !present {
		return ClaudePrimarySessionSource{}, nil
	}
	primary, present, err := projectConfigTable(claude, "primary_session", claudePrimarySessionField)
	if err != nil {
		return ClaudePrimarySessionSource{}, projectConfigFieldError(path, claudePrimarySessionField, err)
	}
	if !present {
		return ClaudePrimarySessionSource{}, nil
	}

	model, err := projectConfigNonEmptyString(primary, "model")
	if err != nil {
		return ClaudePrimarySessionSource{}, projectConfigFieldError(path, claudePrimaryModelField, err)
	}
	yoloMode, err := projectConfigBool(primary, "yolo_mode")
	if err != nil {
		return ClaudePrimarySessionSource{}, projectConfigFieldError(path, claudePrimaryYoloModeField, err)
	}
	var unsupported []string
	for key := range primary {
		switch key {
		case "model", "yolo_mode":
		default:
			unsupported = append(unsupported, key)
		}
	}
	if len(unsupported) > 0 {
		sort.Strings(unsupported)
		return ClaudePrimarySessionSource{}, projectConfigFieldError(
			path,
			claudePrimarySessionField+"."+unsupported[0],
			errors.New("unsupported field"),
		)
	}
	source := ClaudePrimarySessionSource{Model: model, YoloMode: yoloMode}
	if !claudePrimarySessionSourcePresent(source) {
		return ClaudePrimarySessionSource{}, projectConfigFieldError(
			path,
			claudePrimarySessionField,
			errors.New("table must contain at least one supported field"),
		)
	}
	return source, nil
}

func projectConfigTable(parent map[string]any, key, field string) (map[string]any, bool, error) {
	value, present := parent[key]
	if !present {
		return nil, false, nil
	}
	table, ok := value.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("expected table for %s, got %T", field, value)
	}
	return table, true, nil
}

func projectConfigStringArray(table map[string]any, key string) ([]string, error) {
	value, present := table[key]
	if !present {
		return nil, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array of strings, got %T", value)
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		stringValue, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("expected array of strings, found %T element", item)
		}
		result = append(result, stringValue)
	}
	return result, nil
}

func projectConfigNonEmptyString(table map[string]any, key string) (*string, error) {
	value, present := table[key]
	if !present {
		return nil, nil
	}
	stringValue, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", value)
	}
	if strings.TrimSpace(stringValue) == "" {
		return nil, fmt.Errorf("must be a non-empty string")
	}
	return &stringValue, nil
}

func projectConfigBool(table map[string]any, key string) (*bool, error) {
	value, present := table[key]
	if !present {
		return nil, nil
	}
	boolValue, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf("expected boolean, got %T", value)
	}
	return &boolValue, nil
}

// ProjectConfigurationError keeps safe source/field provenance available to
// alias diagnostics without exposing arbitrary TOML values.
type ProjectConfigurationError struct {
	Source string
	Field  string
	Err    error
}

func (e *ProjectConfigurationError) Error() string {
	return fmt.Sprintf("%s: field %s: %v", e.Source, e.Field, e.Err)
}

func (e *ProjectConfigurationError) Unwrap() error { return e.Err }

func projectConfigFieldError(path, field string, err error) error {
	return &ProjectConfigurationError{Source: path, Field: field, Err: err}
}

func composeCodexPrimarySession(policy *CodexPrimarySessionPolicy, source CodexPrimarySessionSource, path string) {
	if source.Model != nil {
		policy.Model = CodexPrimarySessionStringValue{
			Value:   *source.Model,
			Source:  path,
			Present: true,
		}
	}
	if source.ReasoningEffort != nil {
		policy.ReasoningEffort = CodexPrimarySessionStringValue{
			Value:   *source.ReasoningEffort,
			Source:  path,
			Present: true,
		}
	}
	if source.YoloMode != nil {
		policy.YoloMode = CodexPrimarySessionBoolValue{
			Value:   *source.YoloMode,
			Source:  path,
			Present: true,
		}
	}
}

func composeClaudePrimarySession(policy *ClaudePrimarySessionPolicy, source ClaudePrimarySessionSource, path string) {
	if source.Model != nil {
		policy.Model = ClaudePrimarySessionStringValue{
			Value:   *source.Model,
			Source:  path,
			Present: true,
		}
	}
	if source.YoloMode != nil {
		policy.YoloMode = ClaudePrimarySessionBoolValue{
			Value:   *source.YoloMode,
			Source:  path,
			Present: true,
		}
	}
}

func cloneCodexPrimarySessionSource(source CodexPrimarySessionSource) CodexPrimarySessionSource {
	return CodexPrimarySessionSource{
		Model:           cloneStringPointer(source.Model),
		ReasoningEffort: cloneStringPointer(source.ReasoningEffort),
		YoloMode:        cloneBoolPointer(source.YoloMode),
	}
}

func cloneClaudePrimarySessionSource(source ClaudePrimarySessionSource) ClaudePrimarySessionSource {
	return ClaudePrimarySessionSource{
		Model:    cloneStringPointer(source.Model),
		YoloMode: cloneBoolPointer(source.YoloMode),
	}
}

func cloneProjectTargets(source map[string]ProjectTarget) map[string]ProjectTarget {
	result := make(map[string]ProjectTarget, len(source))
	for name, target := range source {
		target.Profile = cloneStringPointer(target.Profile)
		target.ProfileProvider = cloneStringPointer(target.ProfileProvider)
		target.Endpoint = cloneStringPointer(target.Endpoint)
		result[name] = target
	}
	return result
}

func cloneStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
