package infra

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
)

// CodexPrimarySessionSetup describes an explicit setup-local mutation. Pointer
// presence distinguishes an omitted flag from an explicitly supplied zero
// value, notably --codex-yolo-mode=false.
type CodexPrimarySessionSetup struct {
	Model           *string
	ReasoningEffort *string
	YoloMode        *bool
	Clear           bool
}

func (setup CodexPrimarySessionSetup) requested() bool {
	return setup.Clear || setup.Model != nil || setup.ReasoningEffort != nil || setup.YoloMode != nil
}

// ClaudePrimarySessionSetup describes an explicit setup-local mutation for the
// Claude-only primary-session model policy.
type ClaudePrimarySessionSetup struct {
	Model *string
	Clear bool
}

func (setup ClaudePrimarySessionSetup) requested() bool {
	return setup.Clear || setup.Model != nil
}

type projectConfigAtomicWriter func(path string, data []byte, mode fs.FileMode) error
type projectConfigFileReplacer func(source, destination string) error

type preparedProjectConfigWrite struct {
	path           string
	original       []byte
	updated        []byte
	mode           fs.FileMode
	existed        bool
	changed        bool
	codexMutation  CodexPrimarySessionSetup
	claudeMutation ClaudePrimarySessionSetup
	atomicWrite    projectConfigAtomicWriter
}

func prepareCodexPrimarySessionSetup(
	layout Layout,
	setup CodexPrimarySessionSetup,
	atomicWrite projectConfigAtomicWriter,
) (*preparedProjectConfigWrite, error) {
	return preparePrimarySessionSetup(layout, setup, ClaudePrimarySessionSetup{}, atomicWrite)
}

func preparePrimarySessionSetup(
	layout Layout,
	codexSetup CodexPrimarySessionSetup,
	claudeSetup ClaudePrimarySessionSetup,
	atomicWrite projectConfigAtomicWriter,
) (*preparedProjectConfigWrite, error) {
	if layout.Mode != ModeLocal {
		if codexSetup.requested() {
			return nil, fmt.Errorf("field %s: Codex primary-session setup flags are local-only", codexPrimarySessionField)
		}
		if claudeSetup.requested() {
			return nil, fmt.Errorf("field %s: Claude primary-session setup flags are local-only", claudePrimarySessionField)
		}
		return nil, nil
	}

	path := filepath.Join(layout.AgentsDir, ".configs", projectConfigFileName)
	requested := codexSetup.requested() || claudeSetup.requested()
	globalPath, err := globalProjectConfigPathForSetup()
	if err != nil {
		return nil, err
	}
	if requested && samePath(path, globalPath) {
		field := codexPrimarySessionField
		if !codexSetup.requested() {
			field = claudePrimarySessionField
		}
		return nil, projectConfigFieldError(
			path,
			field,
			fmt.Errorf("target resolves to the ignored global project-config path %s", globalPath),
		)
	}

	parsed, err := loadTargetProjectConfigForSetup(layout, globalPath)
	if err != nil {
		return nil, err
	}
	if !requested {
		return nil, nil
	}

	if codexSetup.Clear && (codexSetup.Model != nil || codexSetup.ReasoningEffort != nil || codexSetup.YoloMode != nil) {
		return nil, projectConfigFieldError(
			path,
			codexPrimarySessionField,
			fmt.Errorf("--clear-codex-primary-session conflicts with primary-session set flags"),
		)
	}
	if codexSetup.Model != nil && strings.TrimSpace(*codexSetup.Model) == "" {
		return nil, projectConfigFieldError(path, codexPrimaryModelField, fmt.Errorf("supplied value must be a non-empty string"))
	}
	if codexSetup.ReasoningEffort != nil && strings.TrimSpace(*codexSetup.ReasoningEffort) == "" {
		return nil, projectConfigFieldError(path, codexPrimaryReasoningEffortField, fmt.Errorf("supplied value must be a non-empty string"))
	}
	if claudeSetup.Clear && claudeSetup.Model != nil {
		return nil, projectConfigFieldError(
			path,
			claudePrimarySessionField,
			fmt.Errorf("--clear-claude-primary-session conflicts with primary-session set flags"),
		)
	}
	if claudeSetup.Model != nil && strings.TrimSpace(*claudeSetup.Model) == "" {
		return nil, projectConfigFieldError(path, claudePrimaryModelField, fmt.Errorf("supplied value must be a non-empty string"))
	}

	original, mode, existed, err := readProjectConfigForSetup(path)
	if err != nil {
		return nil, err
	}
	updated := append([]byte(nil), original...)
	if codexSetup.requested() {
		updated, err = renderCodexPrimarySessionSetup(updated, path, parsed, codexSetup)
		if err != nil {
			return nil, err
		}
	}
	if claudeSetup.requested() {
		updatedConfig, parseErr := parseProjectConfig(updated, path)
		if parseErr != nil {
			return nil, fmt.Errorf("render project config %s: %w", path, parseErr)
		}
		updated, err = renderClaudePrimarySessionSetup(updated, path, updatedConfig, claudeSetup)
		if err != nil {
			return nil, err
		}
	}
	if _, err := parseProjectConfig(updated, path); err != nil {
		return nil, fmt.Errorf("render project config %s: %w", path, err)
	}
	if atomicWrite == nil {
		atomicWrite = writeProjectConfigAtomically
	}
	return &preparedProjectConfigWrite{
		path:           path,
		original:       original,
		updated:        updated,
		mode:           mode,
		existed:        existed,
		changed:        !bytes.Equal(original, updated),
		codexMutation:  codexSetup,
		claudeMutation: claudeSetup,
		atomicWrite:    atomicWrite,
	}, nil
}

func globalProjectConfigPathForSetup() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir for project config setup: %w", err)
	}
	homeDir, err = filepath.Abs(homeDir)
	if err != nil {
		return "", fmt.Errorf("resolve home dir for project config setup: %w", err)
	}
	return filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName), nil
}

func loadTargetProjectConfigForSetup(layout Layout, globalPath string) (parsedProjectConfig, error) {
	composite, err := loadCompositeProjectConfig(
		ancestorDirsRootFirst(layout.RootDir),
		globalPath,
	)
	if err != nil {
		return parsedProjectConfig{}, err
	}
	targetPath := filepath.Join(layout.AgentsDir, ".configs", projectConfigFileName)
	for _, source := range composite.Sources {
		if !samePath(source.Path, targetPath) {
			continue
		}
		return parsedProjectConfig{
			EnabledMCPServers:    append([]string(nil), source.EnabledServers...),
			PrimarySession:       cloneCodexPrimarySessionSource(source.CodexPrimarySession),
			ClaudePrimarySession: cloneClaudePrimarySessionSource(source.ClaudePrimarySession),
		}, nil
	}
	return parsedProjectConfig{}, nil
}

func readProjectConfigForSetup(path string) ([]byte, fs.FileMode, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, 0o644, false, nil
	}
	if err != nil {
		return nil, 0, false, projectConfigFieldError(path, codexPrimarySessionField, fmt.Errorf("stat: %w", err))
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, 0, false, projectConfigFieldError(path, codexPrimarySessionField, fmt.Errorf("expected a regular project config file"))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, false, projectConfigFieldError(path, codexPrimarySessionField, fmt.Errorf("read: %w", err))
	}
	return data, info.Mode().Perm(), true, nil
}

func commitPreparedProjectConfig(prepared *preparedProjectConfigWrite, out io.Writer) error {
	if prepared == nil || !prepared.changed {
		return nil
	}
	field := codexPrimarySessionField
	if !prepared.codexMutation.requested() && prepared.claudeMutation.requested() {
		field = claudePrimarySessionField
	}
	current, _, existed, err := readProjectConfigForSetup(prepared.path)
	if err != nil {
		return err
	}
	if existed != prepared.existed || !bytes.Equal(current, prepared.original) {
		return projectConfigFieldError(
			prepared.path,
			field,
			fmt.Errorf("project config changed while setup was running; refusing to overwrite it"),
		)
	}
	if err := prepared.atomicWrite(prepared.path, prepared.updated, prepared.mode); err != nil {
		return projectConfigFieldError(prepared.path, field, fmt.Errorf("atomic write: %w", err))
	}
	if prepared.codexMutation.requested() {
		if prepared.codexMutation.Clear {
			logf(out, "Cleared Codex primary session from %s", prepared.path)
		} else {
			logf(out, "Updated Codex primary session in %s", prepared.path)
		}
	}
	if prepared.claudeMutation.requested() {
		if prepared.claudeMutation.Clear {
			logf(out, "Cleared Claude primary session from %s", prepared.path)
		} else {
			logf(out, "Updated Claude primary session in %s", prepared.path)
		}
	}
	return nil
}

func renderCodexPrimarySessionSetup(
	data []byte,
	path string,
	parsed parsedProjectConfig,
	setup CodexPrimarySessionSetup,
) ([]byte, error) {
	location, err := locateCodexPrimarySessionTable(data)
	if err != nil {
		return nil, projectConfigFieldError(path, codexPrimarySessionField, err)
	}
	present := codexPrimarySessionSourcePresent(parsed.PrimarySession)
	if present && !location.present {
		return nil, projectConfigFieldError(
			path,
			codexPrimarySessionField,
			fmt.Errorf("setup requires the explicit [%s] table form", codexPrimarySessionField),
		)
	}

	if setup.Clear {
		if !present {
			return append([]byte(nil), data...), nil
		}
		return clearCodexPrimarySessionTable(data, location)
	}
	if !location.present {
		return appendCodexPrimarySessionTable(data, setup)
	}
	return updateCodexPrimarySessionTable(data, path, parsed.PrimarySession, location, setup)
}

func renderClaudePrimarySessionSetup(
	data []byte,
	path string,
	parsed parsedProjectConfig,
	setup ClaudePrimarySessionSetup,
) ([]byte, error) {
	location, err := locateClaudePrimarySessionTable(data)
	if err != nil {
		return nil, projectConfigFieldError(path, claudePrimarySessionField, err)
	}
	present := claudePrimarySessionSourcePresent(parsed.ClaudePrimarySession)
	if present && !location.present {
		return nil, projectConfigFieldError(
			path,
			claudePrimarySessionField,
			fmt.Errorf("setup requires the explicit [%s] table form", claudePrimarySessionField),
		)
	}

	if setup.Clear {
		if !present {
			return append([]byte(nil), data...), nil
		}
		return clearCodexPrimarySessionTable(data, location)
	}
	if !location.present {
		return appendClaudePrimarySessionTable(data, setup)
	}
	return updateClaudePrimarySessionTable(data, path, parsed.ClaudePrimarySession, location, setup)
}

func codexPrimarySessionSourcePresent(source CodexPrimarySessionSource) bool {
	return source.Model != nil || source.ReasoningEffort != nil || source.YoloMode != nil
}

func claudePrimarySessionSourcePresent(source ClaudePrimarySessionSource) bool {
	return source.Model != nil
}

type textSpan struct {
	start int
	end   int
}

type primarySessionFieldLocation struct {
	value textSpan
	line  textSpan
}

type primarySessionTableLocation struct {
	present bool
	header  textSpan
	fields  map[string]primarySessionFieldLocation
}

func locateCodexPrimarySessionTable(data []byte) (primarySessionTableLocation, error) {
	return locatePrimarySessionTable(
		data,
		[]string{"agents", "codex", "primary_session"},
		map[string]bool{"model": true, "reasoning_effort": true, "yolo_mode": true},
	)
}

func locateClaudePrimarySessionTable(data []byte) (primarySessionTableLocation, error) {
	return locatePrimarySessionTable(
		data,
		[]string{"agents", "claude", "primary_session"},
		map[string]bool{"model": true},
	)
}

func locatePrimarySessionTable(data []byte, target []string, supportedFields map[string]bool) (primarySessionTableLocation, error) {
	location := primarySessionTableLocation{fields: map[string]primarySessionFieldLocation{}}
	var parser unstable.Parser
	parser.Reset(data)
	var currentTable []string

	for parser.NextExpression() {
		expression := parser.Expression()
		switch expression.Kind {
		case unstable.Table, unstable.ArrayTable:
			parts, ranges := unstableNodeKey(expression)
			currentTable = parts
			if expression.Kind != unstable.Table || !equalStringSlices(parts, target) {
				continue
			}
			if location.present {
				return primarySessionTableLocation{}, fmt.Errorf("multiple explicit primary-session tables")
			}
			if len(ranges) == 0 {
				return primarySessionTableLocation{}, fmt.Errorf("primary-session table has no key range")
			}
			location.present = true
			location.header = physicalLineSpan(data, ranges[0].start, ranges[len(ranges)-1].end)
		case unstable.KeyValue:
			if !location.present || !equalStringSlices(currentTable, target) {
				continue
			}
			parts, ranges := unstableNodeKey(expression)
			if len(parts) != 1 || len(ranges) != 1 {
				continue
			}
			field := parts[0]
			if !supportedFields[field] {
				continue
			}
			valueRange := expression.Value().Raw
			value := textSpan{start: int(valueRange.Offset), end: int(valueRange.Offset + valueRange.Length)}
			location.fields[field] = primarySessionFieldLocation{
				value: value,
				line:  physicalLineSpan(data, ranges[0].start, value.end),
			}
		}
	}
	if err := parser.Error(); err != nil {
		return primarySessionTableLocation{}, fmt.Errorf("locate primary-session table: %w", err)
	}
	return location, nil
}

func unstableNodeKey(node *unstable.Node) ([]string, []textSpan) {
	var parts []string
	var ranges []textSpan
	iterator := node.Key()
	for iterator.Next() {
		key := iterator.Node()
		parts = append(parts, string(key.Data))
		ranges = append(ranges, textSpan{
			start: int(key.Raw.Offset),
			end:   int(key.Raw.Offset + key.Raw.Length),
		})
	}
	return parts, ranges
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func physicalLineSpan(data []byte, contentStart, contentEnd int) textSpan {
	start := 0
	if newline := bytes.LastIndexByte(data[:contentStart], '\n'); newline >= 0 {
		start = newline + 1
	}
	end := len(data)
	if newline := bytes.IndexByte(data[contentEnd:], '\n'); newline >= 0 {
		end = contentEnd + newline + 1
	}
	return textSpan{start: start, end: end}
}

type textEdit struct {
	span        textSpan
	replacement []byte
}

func clearCodexPrimarySessionTable(data []byte, location primarySessionTableLocation) ([]byte, error) {
	edits := []textEdit{{span: location.header}}
	for _, field := range location.fields {
		edits = append(edits, textEdit{span: field.line})
	}
	return applyTextEdits(data, edits)
}

func updateCodexPrimarySessionTable(
	data []byte,
	path string,
	current CodexPrimarySessionSource,
	location primarySessionTableLocation,
	setup CodexPrimarySessionSetup,
) ([]byte, error) {
	var edits []textEdit
	missing := CodexPrimarySessionSetup{}

	if setup.Model != nil && (current.Model == nil || *current.Model != *setup.Model) {
		if field, ok := location.fields["model"]; ok {
			value, err := encodeProjectConfigString(*setup.Model)
			if err != nil {
				return nil, projectConfigFieldError(path, codexPrimaryModelField, err)
			}
			edits = append(edits, textEdit{span: field.value, replacement: value})
		} else if current.Model != nil {
			return nil, projectConfigFieldError(path, codexPrimaryModelField, fmt.Errorf("field is not directly editable in the explicit table"))
		} else {
			missing.Model = setup.Model
		}
	}
	if setup.ReasoningEffort != nil && (current.ReasoningEffort == nil || *current.ReasoningEffort != *setup.ReasoningEffort) {
		if field, ok := location.fields["reasoning_effort"]; ok {
			value, err := encodeProjectConfigString(*setup.ReasoningEffort)
			if err != nil {
				return nil, projectConfigFieldError(path, codexPrimaryReasoningEffortField, err)
			}
			edits = append(edits, textEdit{span: field.value, replacement: value})
		} else if current.ReasoningEffort != nil {
			return nil, projectConfigFieldError(path, codexPrimaryReasoningEffortField, fmt.Errorf("field is not directly editable in the explicit table"))
		} else {
			missing.ReasoningEffort = setup.ReasoningEffort
		}
	}
	if setup.YoloMode != nil && (current.YoloMode == nil || *current.YoloMode != *setup.YoloMode) {
		if field, ok := location.fields["yolo_mode"]; ok {
			edits = append(edits, textEdit{span: field.value, replacement: []byte(strconv.FormatBool(*setup.YoloMode))})
		} else if current.YoloMode != nil {
			return nil, projectConfigFieldError(path, codexPrimaryYoloModeField, fmt.Errorf("field is not directly editable in the explicit table"))
		} else {
			missing.YoloMode = setup.YoloMode
		}
	}

	if missing.requested() {
		insertAt := location.header.end
		for _, field := range location.fields {
			if field.line.end > insertAt {
				insertAt = field.line.end
			}
		}
		body, err := renderCodexPrimarySessionFields(missing, detectProjectConfigNewline(data))
		if err != nil {
			return nil, projectConfigFieldError(path, codexPrimarySessionField, err)
		}
		if insertAt > 0 && data[insertAt-1] != '\n' {
			body = append([]byte(detectProjectConfigNewline(data)), body...)
		}
		edits = append(edits, textEdit{span: textSpan{start: insertAt, end: insertAt}, replacement: body})
	}
	if len(edits) == 0 {
		return append([]byte(nil), data...), nil
	}
	return applyTextEdits(data, edits)
}

func updateClaudePrimarySessionTable(
	data []byte,
	path string,
	current ClaudePrimarySessionSource,
	location primarySessionTableLocation,
	setup ClaudePrimarySessionSetup,
) ([]byte, error) {
	if setup.Model == nil || (current.Model != nil && *current.Model == *setup.Model) {
		return append([]byte(nil), data...), nil
	}
	if field, ok := location.fields["model"]; ok {
		value, err := encodeProjectConfigString(*setup.Model)
		if err != nil {
			return nil, projectConfigFieldError(path, claudePrimaryModelField, err)
		}
		return applyTextEdits(data, []textEdit{{span: field.value, replacement: value}})
	}
	if current.Model != nil {
		return nil, projectConfigFieldError(path, claudePrimaryModelField, fmt.Errorf("field is not directly editable in the explicit table"))
	}

	insertAt := location.header.end
	for _, field := range location.fields {
		if field.line.end > insertAt {
			insertAt = field.line.end
		}
	}
	body, err := renderClaudePrimarySessionFields(setup, detectProjectConfigNewline(data))
	if err != nil {
		return nil, projectConfigFieldError(path, claudePrimarySessionField, err)
	}
	if insertAt > 0 && data[insertAt-1] != '\n' {
		body = append([]byte(detectProjectConfigNewline(data)), body...)
	}
	return applyTextEdits(data, []textEdit{{span: textSpan{start: insertAt, end: insertAt}, replacement: body}})
}

func appendCodexPrimarySessionTable(data []byte, setup CodexPrimarySessionSetup) ([]byte, error) {
	newline := detectProjectConfigNewline(data)
	result := append([]byte(nil), data...)
	if len(result) > 0 {
		if !bytes.HasSuffix(result, []byte(newline)) {
			result = append(result, newline...)
		}
		if !bytes.HasSuffix(result, []byte(newline+newline)) {
			result = append(result, newline...)
		}
	}
	result = append(result, []byte("["+codexPrimarySessionField+"]"+newline)...)
	fields, err := renderCodexPrimarySessionFields(setup, newline)
	if err != nil {
		return nil, err
	}
	return append(result, fields...), nil
}

func appendClaudePrimarySessionTable(data []byte, setup ClaudePrimarySessionSetup) ([]byte, error) {
	newline := detectProjectConfigNewline(data)
	result := append([]byte(nil), data...)
	if len(result) > 0 {
		if !bytes.HasSuffix(result, []byte(newline)) {
			result = append(result, newline...)
		}
		if !bytes.HasSuffix(result, []byte(newline+newline)) {
			result = append(result, newline...)
		}
	}
	result = append(result, []byte("["+claudePrimarySessionField+"]"+newline)...)
	fields, err := renderClaudePrimarySessionFields(setup, newline)
	if err != nil {
		return nil, err
	}
	return append(result, fields...), nil
}

func renderCodexPrimarySessionFields(setup CodexPrimarySessionSetup, newline string) ([]byte, error) {
	var body strings.Builder
	if setup.Model != nil {
		value, err := encodeProjectConfigString(*setup.Model)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&body, "model = %s%s", value, newline)
	}
	if setup.ReasoningEffort != nil {
		value, err := encodeProjectConfigString(*setup.ReasoningEffort)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&body, "reasoning_effort = %s%s", value, newline)
	}
	if setup.YoloMode != nil {
		fmt.Fprintf(&body, "yolo_mode = %s%s", strconv.FormatBool(*setup.YoloMode), newline)
	}
	return []byte(body.String()), nil
}

func renderClaudePrimarySessionFields(setup ClaudePrimarySessionSetup, newline string) ([]byte, error) {
	if setup.Model == nil {
		return nil, fmt.Errorf("model is required when rendering a Claude primary-session table")
	}
	value, err := encodeProjectConfigString(*setup.Model)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("model = %s%s", value, newline)), nil
}

func encodeProjectConfigString(value string) ([]byte, error) {
	encoded, err := toml.Marshal(struct {
		Value string `toml:"value"`
	}{Value: value})
	if err != nil {
		return nil, fmt.Errorf("encode string: %w", err)
	}
	line := strings.TrimSpace(string(encoded))
	_, scalar, ok := strings.Cut(line, "=")
	if !ok {
		return nil, fmt.Errorf("encode string: missing scalar value")
	}
	return []byte(strings.TrimSpace(scalar)), nil
}

func detectProjectConfigNewline(data []byte) string {
	if newline := bytes.IndexByte(data, '\n'); newline > 0 && data[newline-1] == '\r' {
		return "\r\n"
	}
	return "\n"
}

func applyTextEdits(data []byte, edits []textEdit) ([]byte, error) {
	sort.Slice(edits, func(left, right int) bool {
		if edits[left].span.start == edits[right].span.start {
			return edits[left].span.end > edits[right].span.end
		}
		return edits[left].span.start > edits[right].span.start
	})
	updated := append([]byte(nil), data...)
	lastStart := len(data) + 1
	for _, edit := range edits {
		if edit.span.start < 0 || edit.span.end < edit.span.start || edit.span.end > len(data) {
			return nil, fmt.Errorf("invalid project config edit range %d:%d", edit.span.start, edit.span.end)
		}
		if edit.span.end > lastStart {
			return nil, fmt.Errorf("overlapping project config edit range %d:%d", edit.span.start, edit.span.end)
		}
		next := make([]byte, 0, len(updated)-(edit.span.end-edit.span.start)+len(edit.replacement))
		next = append(next, updated[:edit.span.start]...)
		next = append(next, edit.replacement...)
		next = append(next, updated[edit.span.end:]...)
		updated = next
		lastStart = edit.span.start
	}
	return updated, nil
}

func writeProjectConfigAtomically(path string, data []byte, mode fs.FileMode) error {
	return writeProjectConfigAtomicallyWithReplace(path, data, mode, replaceProjectConfigFile)
}

func writeProjectConfigAtomicallyWithReplace(
	path string,
	data []byte,
	mode fs.FileMode,
	replace projectConfigFileReplacer,
) error {
	if replace == nil {
		return fmt.Errorf("replace project config: replacement function is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	defer temporary.Close()

	if err := temporary.Chmod(mode.Perm()); err != nil {
		return fmt.Errorf("set temporary file mode: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := replace(temporaryPath, path); err != nil {
		return fmt.Errorf("replace project config: %w", err)
	}
	return nil
}
