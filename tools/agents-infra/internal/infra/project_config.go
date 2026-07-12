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

type parsedProjectConfig struct {
	EnabledMCPServers []string
	PrimarySession    CodexPrimarySessionSource
}

type compositeProjectConfig struct {
	EnabledOrder   []string
	EnabledBy      map[string][]string
	Sources        []CodexProjectConfigSource
	PrimarySession CodexPrimarySessionPolicy
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

		composite.Sources = append(composite.Sources, CodexProjectConfigSource{
			Path:           path,
			EnabledServers: append([]string(nil), config.EnabledMCPServers...),
			PrimarySession: cloneCodexPrimarySessionSource(config.PrimarySession),
		})
		composeCodexPrimarySession(&composite.PrimarySession, config.PrimarySession, path)

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
			codexPrimarySessionField,
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
	codex, present, err := projectConfigTable(agents, "codex", "agents.codex")
	if err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, "agents.codex", err)
	}
	if !present {
		return config, nil
	}
	primary, present, err := projectConfigTable(codex, "primary_session", codexPrimarySessionField)
	if err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, codexPrimarySessionField, err)
	}
	if !present {
		return config, nil
	}

	config.PrimarySession.Model, err = projectConfigNonEmptyString(primary, "model")
	if err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, codexPrimaryModelField, err)
	}
	config.PrimarySession.ReasoningEffort, err = projectConfigNonEmptyString(primary, "reasoning_effort")
	if err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, codexPrimaryReasoningEffortField, err)
	}
	config.PrimarySession.YoloMode, err = projectConfigBool(primary, "yolo_mode")
	if err != nil {
		return parsedProjectConfig{}, projectConfigFieldError(path, codexPrimaryYoloModeField, err)
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
		return parsedProjectConfig{}, projectConfigFieldError(
			path,
			codexPrimarySessionField+"."+unsupported[0],
			errors.New("unsupported field"),
		)
	}
	if config.PrimarySession.Model == nil && config.PrimarySession.ReasoningEffort == nil && config.PrimarySession.YoloMode == nil {
		return parsedProjectConfig{}, projectConfigFieldError(
			path,
			codexPrimarySessionField,
			errors.New("table must contain at least one supported field"),
		)
	}
	return config, nil
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

func cloneCodexPrimarySessionSource(source CodexPrimarySessionSource) CodexPrimarySessionSource {
	return CodexPrimarySessionSource{
		Model:           cloneStringPointer(source.Model),
		ReasoningEffort: cloneStringPointer(source.ReasoningEffort),
		YoloMode:        cloneBoolPointer(source.YoloMode),
	}
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
