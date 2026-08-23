package infra

import (
	"fmt"
	"io"
	"os"

	"github.com/pelletier/go-toml/v2"
)

const managedCodexConfigRelativePath = ".configs/codex-config.toml"

func syncManagedCodexConfig(sourcePath, destinationPath string, mode os.FileMode) error {
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source-managed Codex config %s: %w", sourcePath, err)
	}
	installedData, err := os.ReadFile(destinationPath)
	if os.IsNotExist(err) {
		if err := writeProjectConfigAtomically(destinationPath, sourceData, mode); err != nil {
			return fmt.Errorf("install source-managed Codex config %s: %w", destinationPath, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read existing managed Codex config %s: %w", destinationPath, err)
	}

	var sourceDocument map[string]any
	if err := toml.Unmarshal(sourceData, &sourceDocument); err != nil {
		return fmt.Errorf("parse source-managed Codex config %s: %w", sourcePath, err)
	}
	var installedDocument map[string]any
	if err := toml.Unmarshal(installedData, &installedDocument); err != nil {
		return fmt.Errorf("parse existing managed Codex config %s: %w", destinationPath, err)
	}

	// Codex mutates these user-level tables while it runs. Preserve installed
	// trust decisions and TUI acknowledgement state while source remains the
	// authority for model, reasoning, service tier, and other managed defaults.
	for _, key := range []string{"projects", "notice"} {
		merged, err := mergeCodexConfigTable(sourceDocument, installedDocument, key, "")
		if err != nil {
			return fmt.Errorf("merge existing managed Codex config %s: %w", destinationPath, err)
		}
		if len(merged) == 0 {
			delete(sourceDocument, key)
		} else {
			sourceDocument[key] = merged
		}
	}

	// Profiles are user-level configuration. Carry forward custom profiles, but
	// never retain the withdrawn fast profile from an older managed install.
	profiles, err := mergeCodexConfigTable(sourceDocument, installedDocument, "profiles", "fast")
	if err != nil {
		return fmt.Errorf("merge existing managed Codex config %s: %w", destinationPath, err)
	}
	if len(profiles) == 0 {
		delete(sourceDocument, "profiles")
	} else {
		sourceDocument["profiles"] = profiles
	}

	mergedData, err := toml.Marshal(sourceDocument)
	if err != nil {
		return fmt.Errorf("encode merged managed Codex config %s: %w", destinationPath, err)
	}
	if err := writeProjectConfigAtomically(destinationPath, mergedData, mode); err != nil {
		return fmt.Errorf("write merged managed Codex config %s: %w", destinationPath, err)
	}
	return nil
}

func mergeCodexConfigTable(sourceDocument, installedDocument map[string]any, key, excludedKey string) (map[string]any, error) {
	sourceTable, err := codexConfigTable(sourceDocument, key)
	if err != nil {
		return nil, fmt.Errorf("source field %s: %w", key, err)
	}
	installedTable, err := codexConfigTable(installedDocument, key)
	if err != nil {
		return nil, fmt.Errorf("installed field %s: %w", key, err)
	}
	merged := make(map[string]any, len(sourceTable)+len(installedTable))
	for name, value := range sourceTable {
		if name != excludedKey {
			merged[name] = value
		}
	}
	for name, value := range installedTable {
		if name != excludedKey {
			merged[name] = value
		}
	}
	return merged, nil
}

func codexConfigTable(document map[string]any, key string) (map[string]any, error) {
	raw, found := document[key]
	if !found {
		return nil, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected table, got %T", raw)
	}
	return table, nil
}

func renderProjectCodexConfig(sourcePath, destinationPath string, out io.Writer) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read installed Codex config %s: %w", sourcePath, err)
	}

	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse installed Codex config %s: %w", sourcePath, err)
	}
	delete(document, "profiles")

	rendered, err := toml.Marshal(document)
	if err != nil {
		return fmt.Errorf("render project-local Codex config from %s: %w", sourcePath, err)
	}
	rendered = append([]byte(generatedCodexConfigMarker+"\n"), rendered...)
	if err := writeProjectConfigAtomically(destinationPath, rendered, 0o644); err != nil {
		return fmt.Errorf("write project-local Codex config %s: %w", destinationPath, err)
	}
	logf(out, "Rendered project-local Codex config without user-level profiles: %s", destinationPath)
	return nil
}
