package infra

import (
	"errors"
	"fmt"
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
)

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
// [agents.claude.primary_session] tables. Claude accepts a model policy only;
// its reasoning and permission controls remain independent of Codex policy.
type ClaudePrimarySessionPolicy struct {
	Model ClaudePrimarySessionStringValue
}

type ClaudePrimarySessionStringValue struct {
	Value   string
	Source  string
	Present bool
}

// ClaudePrimarySessionSource preserves the model contributed by one project
// config. Pointer presence distinguishes an omitted model from a configured one.
type ClaudePrimarySessionSource struct {
	Model *string
}

type parsedProjectConfig struct {
	EnabledMCPServers    []string
	PrimarySession       CodexPrimarySessionSource
	ClaudePrimarySession ClaudePrimarySessionSource
}

// ProjectConfigSource records all policy contributed by one project config.
// Launcher-specific plans copy only their provider's policy from this source.
type ProjectConfigSource struct {
	Path                 string
	EnabledServers       []string
	CodexPrimarySession  CodexPrimarySessionSource
	ClaudePrimarySession ClaudePrimarySessionSource
}

type compositeProjectConfig struct {
	EnabledOrder         []string
	EnabledBy            map[string][]string
	Sources              []ProjectConfigSource
	PrimarySession       CodexPrimarySessionPolicy
	ClaudePrimarySession ClaudePrimarySessionPolicy
}

func loadCompositeProjectConfig(ancestors []string, globalProjectConfigPath string) (compositeProjectConfig, error) {
	composite := compositeProjectConfig{
		EnabledBy: map[string][]string{},
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
			return compositeProjectConfig{}, fmt.Errorf("read project config %s: %w", path, err)
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
		})
		composeCodexPrimarySession(&composite.PrimarySession, config.PrimarySession, path)
		composeClaudePrimarySession(&composite.ClaudePrimarySession, config.ClaudePrimarySession, path)

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
		return parsedProjectConfig{}, fmt.Errorf(
			"%s: field %s (including %s): parse TOML: %w",
			path,
			projectConfigParseField,
			codexPrimarySessionField+" and "+claudePrimarySessionField,
			err,
		)
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
	return config, nil
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
	var unsupported []string
	for key := range primary {
		if key != "model" {
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
	if model == nil {
		return ClaudePrimarySessionSource{}, projectConfigFieldError(
			path,
			claudePrimarySessionField,
			errors.New("table must contain at least one supported field"),
		)
	}
	return ClaudePrimarySessionSource{Model: model}, nil
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

func projectConfigFieldError(path, field string, err error) error {
	return fmt.Errorf("%s: field %s: %w", path, field, err)
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
}

func cloneCodexPrimarySessionSource(source CodexPrimarySessionSource) CodexPrimarySessionSource {
	return CodexPrimarySessionSource{
		Model:           cloneStringPointer(source.Model),
		ReasoningEffort: cloneStringPointer(source.ReasoningEffort),
		YoloMode:        cloneBoolPointer(source.YoloMode),
	}
}

func cloneClaudePrimarySessionSource(source ClaudePrimarySessionSource) ClaudePrimarySessionSource {
	return ClaudePrimarySessionSource{Model: cloneStringPointer(source.Model)}
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
