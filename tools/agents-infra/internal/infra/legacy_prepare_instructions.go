package infra

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// This file is the v1 prepare compatibility exception. Ordinary setup and
// refresh-links no longer distribute instructions: Curator owns instruction
// materialization for managed runtimes. The schema-version-1
// primary-session preparation contract consumed by task-board still requires
// real rendered instruction artifacts with honest states and hashes, so
// PreparePrimarySession — and only PreparePrimarySession — renders them here.
// Skill fan-out is not part of this path: prepare refreshes instructions,
// rules, and settings, and leaves every skill surface untouched. Full deletion
// of this renderer is gated on a coordinated consumer migration to a residual
// prepare contract; see the repository CHANGELOG.

const generatedClaudeEntrypoint = "# Claude Instructions\n\nLoad all instructions from the Claude runtime instructions directory:\n\n@instructions/INSTRUCTIONS.md\n"
const projectCodexInstructionScaffold = "# Project Instructions\n\n<!-- Add project-specific Codex instructions or relative @includes here. -->\n"
const projectClaudeInstructionScaffold = "# Project Instructions\n\n<!-- Add project-specific Claude instructions or relative @includes here. -->\n"

// prepareCodexProjectSurface refreshes the v1 Codex provider surface:
// rendered instruction entrypoints plus residual rules links. It never touches
// Codex config state or skill links.
func prepareCodexProjectSurface(layout Layout, out io.Writer) error {
	if err := writeCodexEntrypoints(layout, out); err != nil {
		return err
	}
	return setupCodexRules(layout, out)
}

// prepareClaudeProjectSurface refreshes the v1 Claude provider surface: the
// instruction link, the rendered entrypoint, and the residual settings link.
// It never touches skill links.
func prepareClaudeProjectSurface(layout Layout, out io.Writer) error {
	if err := createSymlink(filepath.Join(layout.AgentsDir, ".instructions"), filepath.Join(layout.ClaudeDir, "instructions"), out); err != nil {
		return err
	}
	if err := writeClaudeEntrypoint(layout); err != nil {
		return err
	}
	return setupClaudeSettings(layout, out)
}

func writeClaudeEntrypoint(layout Layout) error {
	return os.WriteFile(filepath.Join(layout.ClaudeDir, "CLAUDE.md"), []byte(generatedClaudeEntrypoint), 0o644)
}

func isGeneratedClaudeEntrypointFile(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && string(data) == generatedClaudeEntrypoint
}

type renderSource struct {
	path           string
	includeBaseDir string
}

func writeCodexEntrypoints(layout Layout, out io.Writer) error {
	source := defaultInstructionSource(layout)
	if layout.Mode == ModeLocal {
		projectSource, err := projectInstructionSource(layout, out)
		if err != nil {
			return err
		}
		source = projectSource
	}
	if err := writeRenderedInstructions(filepath.Join(layout.CodexDir, "AGENTS.md"), source, layout, out); err != nil {
		return err
	}
	if layout.Mode != ModeLocal {
		return nil
	}
	return writeRenderedInstructions(filepath.Join(layout.RootDir, "AGENTS.md"), source, layout, out)
}

func defaultInstructionSource(layout Layout) renderSource {
	path := filepath.Join(layout.AgentsDir, ".instructions", "AGENTS.md")
	return renderSource{path: path, includeBaseDir: filepath.Dir(path)}
}

func projectInstructionSource(layout Layout, out io.Writer) (renderSource, error) {
	preserved := filepath.Join(layout.AgentsDir, ".instructions", "AGENTS.project.md")
	if _, err := os.Stat(preserved); err == nil {
		return renderSource{path: preserved, includeBaseDir: layout.RootDir}, nil
	} else if !os.IsNotExist(err) {
		return renderSource{}, err
	}

	projectDoc := filepath.Join(layout.RootDir, "AGENTS.md")
	data, err := os.ReadFile(projectDoc)
	if os.IsNotExist(err) {
		return defaultInstructionSource(layout), nil
	}
	if err != nil {
		return renderSource{}, fmt.Errorf("read project AGENTS.md: %w", err)
	}
	if strings.Contains(string(data), generatedInstructionsMarker) {
		return defaultInstructionSource(layout), nil
	}
	if err := os.MkdirAll(filepath.Dir(preserved), 0o755); err != nil {
		return renderSource{}, err
	}
	if err := os.WriteFile(preserved, data, 0o644); err != nil {
		return renderSource{}, fmt.Errorf("preserve project AGENTS.md source: %w", err)
	}
	logf(out, "Preserved project AGENTS.md source: %s", preserved)
	return renderSource{path: preserved, includeBaseDir: layout.RootDir}, nil
}

func writeRenderedInstructions(target string, source renderSource, layout Layout, out io.Writer) error {
	rendered, err := renderInstructions(source, layout, map[string]bool{})
	if err != nil {
		return err
	}
	body := generatedInstructionsMarker + "\n" +
		fmt.Sprintf("<!-- Source: %s -->\n\n", filepath.ToSlash(source.path)) +
		rendered
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	if err := removeManagedPath(target, out); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write rendered instructions %s: %w", target, err)
	}
	logf(out, "Rendered Codex instructions: %s", target)
	return nil
}

func renderInstructions(source renderSource, layout Layout, stack map[string]bool) (string, error) {
	abs, err := filepath.Abs(source.path)
	if err != nil {
		return "", err
	}
	if stack[abs] {
		return "", fmt.Errorf("cyclic instruction include: %s", source.path)
	}
	data, err := os.ReadFile(source.path)
	if err != nil {
		return "", fmt.Errorf("read instruction file %s: %w", source.path, err)
	}

	stack[abs] = true
	defer delete(stack, abs)

	var out strings.Builder
	lines := strings.SplitAfter(string(data), "\n")
	for _, line := range lines {
		includeRef, ok := parseInstructionInclude(line)
		if !ok {
			out.WriteString(line)
			continue
		}
		includePath, err := resolveInstructionInclude(includeRef, source.includeBaseDir, layout)
		if err != nil {
			return "", err
		}
		rendered, err := renderInstructions(renderSource{path: includePath, includeBaseDir: filepath.Dir(includePath)}, layout, stack)
		if err != nil {
			return "", err
		}
		out.WriteString(rendered)
		if !strings.HasSuffix(rendered, "\n") {
			out.WriteString("\n")
		}
	}
	return out.String(), nil
}

func parseInstructionInclude(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "@") {
		return "", false
	}
	ref := strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
	if ref == "" || strings.ContainsAny(ref, " \t") {
		return "", false
	}
	if !strings.HasSuffix(strings.ToLower(ref), ".md") {
		return "", false
	}
	return ref, true
}

func resolveInstructionInclude(ref, baseDir string, layout Layout) (string, error) {
	const agentsHomePrefix = "~/.agents/"
	if strings.HasPrefix(ref, agentsHomePrefix) {
		return filepath.Join(layout.AgentsDir, filepath.FromSlash(strings.TrimPrefix(ref, agentsHomePrefix))), nil
	}
	if strings.HasPrefix(ref, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir for include %s: %w", ref, err)
		}
		return filepath.Join(home, filepath.FromSlash(strings.TrimPrefix(ref, "~/"))), nil
	}
	if filepath.IsAbs(ref) {
		return ref, nil
	}
	return filepath.Join(baseDir, filepath.FromSlash(ref)), nil
}

// ensureLocalInstructionScaffold creates the minimal project-owned instruction
// entrypoints a fresh residual local runtime needs before the v1 compatibility
// renderer can produce its required artifacts. It only creates missing files;
// existing project inputs are never overwritten.
func ensureLocalInstructionScaffold(layout Layout, out io.Writer) error {
	if layout.Mode != ModeLocal {
		return nil
	}
	instructionsDir := filepath.Join(layout.AgentsDir, ".instructions")
	if err := os.MkdirAll(instructionsDir, 0o755); err != nil {
		return fmt.Errorf("create project instruction directory: %w", err)
	}
	for name, body := range map[string]string{
		"AGENTS.md":       projectCodexInstructionScaffold,
		"INSTRUCTIONS.md": projectClaudeInstructionScaffold,
	} {
		path := filepath.Join(instructionsDir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("create project instruction scaffold %s: %w", path, err)
		}
		if _, err := io.WriteString(file, body); err != nil {
			_ = file.Close()
			return fmt.Errorf("write project instruction scaffold %s: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close project instruction scaffold %s: %w", path, err)
		}
		logf(out, "Created project instruction scaffold: %s", path)
	}
	return nil
}
