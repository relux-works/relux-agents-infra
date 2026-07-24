package infra

import (
	"fmt"
	"io"
	"os"

	"github.com/pelletier/go-toml/v2"
)

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
