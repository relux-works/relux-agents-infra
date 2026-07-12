package infra

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupLocalWithoutPrimaryFlagsPreservesProjectConfigByteForByte(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := []byte("# project comment\r\n[mcp]\r\nenabled_servers = [\"figma\"] # keep MCP\r\n\r\n[unknown.table]\r\nvalue = \"keep\"\r\n")
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	// Even an accidental source-side template must not overwrite project state.
	mustWrite(t, filepath.Join(source, ".configs", projectConfigFileName), "[mcp]\nenabled_servers = []\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	assertFileBytes(t, path, original)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s): %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("project config mode = %o, want 640", got)
	}
}

func TestSetupLocalWithoutPrimaryFlagsRejectsInvalidConfigBeforeSync(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := []byte("[agents.codex.primary_session]\nyolo_mode = \"false\"\n")
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{Layout: layout})
	if err == nil {
		t.Fatal("Setup unexpectedly accepted invalid project config")
	}
	for _, want := range []string{path, "field " + codexPrimaryYoloModeField} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Setup error %q missing %q", err, want)
		}
	}
	assertFileBytes(t, path, original)
	assertNoPath(t, filepath.Join(project, ".agents", ".instructions"))
	assertNoPath(t, filepath.Join(project, ".claude"))
}

func TestSetupLocalRejectsInvalidAncestorConfigBeforeTargetMutation(t *testing.T) {
	source := seedSourceRepo(t)
	parent := t.TempDir()
	project := filepath.Join(parent, "nested", "project")
	mustMkdir(t, project)
	ancestorPath := filepath.Join(parent, ".agents", ".configs", projectConfigFileName)
	mustMkdir(t, filepath.Dir(ancestorPath))
	ancestor := []byte("[agents.codex.primary_session]\nmodel = 56\n")
	if err := os.WriteFile(ancestorPath, ancestor, 0o640); err != nil {
		t.Fatalf("WriteFile(%s): %v", ancestorPath, err)
	}
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{
		Layout:              layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{YoloMode: boolPointer(false)},
	})
	if err == nil {
		t.Fatal("Setup unexpectedly accepted invalid ancestor project config")
	}
	for _, want := range []string{ancestorPath, "field " + codexPrimaryModelField} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Setup error %q missing %q", err, want)
		}
	}
	assertFileBytes(t, ancestorPath, ancestor)
	assertNoPath(t, filepath.Join(project, ".agents"))
}

func TestSetupLocalRejectsPrimarySessionMutationAtIgnoredGlobalPath(t *testing.T) {
	tests := []struct {
		name     string
		setup    CodexPrimarySessionSetup
		original []byte
	}{
		{
			name:  "set",
			setup: CodexPrimarySessionSetup{Model: stringPointer("gpt-5.6-terra")},
		},
		{
			name:     "clear",
			setup:    CodexPrimarySessionSetup{Clear: true},
			original: []byte("[agents.codex.primary_session]\nmodel = \"global-model\"\n"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := seedSourceRepo(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			path := filepath.Join(home, ".agents", ".configs", projectConfigFileName)
			if test.original != nil {
				mustMkdir(t, filepath.Dir(path))
				if err := os.WriteFile(path, test.original, 0o640); err != nil {
					t.Fatalf("WriteFile(%s): %v", path, err)
				}
			}
			layout, err := LocalLayout(source, home)
			if err != nil {
				t.Fatalf("LocalLayout: %v", err)
			}

			err = Setup(Options{Layout: layout, PrimarySessionSetup: test.setup})
			if err == nil {
				t.Fatal("Setup unexpectedly accepted the ignored global project-config path")
			}
			for _, want := range []string{path, "field " + codexPrimarySessionField, "global"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("Setup error %q missing %q", err, want)
				}
			}
			if test.original == nil {
				assertNoPath(t, path)
			} else {
				assertFileBytes(t, path, test.original)
			}
			assertNoPath(t, filepath.Join(home, ".agents", ".instructions"))
			assertNoPath(t, filepath.Join(home, ".claude"))
			assertNoPath(t, filepath.Join(home, ".codex"))
			assertNoPath(t, filepath.Join(home, ".local"))
		})
	}
}

func TestSetupLocalRejectsPrimarySessionMutationThroughGlobalPathAlias(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	alias := filepath.Join(t.TempDir(), "home-alias")
	if err := os.Symlink(home, alias); err != nil {
		t.Skipf("cannot create directory symlink for path-alias coverage: %v", err)
	}
	t.Setenv("HOME", home)
	layout, err := LocalLayout(source, alias)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	targetPath := filepath.Join(alias, ".agents", ".configs", projectConfigFileName)
	globalPath := filepath.Join(home, ".agents", ".configs", projectConfigFileName)

	err = Setup(Options{
		Layout:              layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{YoloMode: boolPointer(false)},
	})
	if err == nil {
		t.Fatal("Setup unexpectedly accepted an alias of the ignored global project-config path")
	}
	for _, want := range []string{targetPath, "field " + codexPrimarySessionField, "global"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Setup error %q missing %q", err, want)
		}
	}
	assertNoPath(t, globalPath)
	assertNoPath(t, filepath.Join(home, ".agents"))
}

func TestSetupLocalMergesSuppliedPrimaryFieldsAndPreservesOmittedContent(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := `# project header
[mcp]
enabled_servers = ["figma"] # keep MCP comment

[unknown.table]
owner = "keep-me"

[agents.codex.primary_session] # keep table comment
model = "old-model" # keep model comment
yolo_mode = true # keep yolo comment

# keep comment before following table
[following]
value = 42
`
	mustMkdir(t, filepath.Dir(path))
	mustWrite(t, path, original)
	customCodexConfig := filepath.Join(project, ".codex", "config.toml")
	mustMkdir(t, filepath.Dir(customCodexConfig))
	mustWrite(t, customCodexConfig, "model = \"custom-project-model\"\n")
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	reasoning := "xhigh"
	yolo := false

	if err := Setup(Options{
		Layout: layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{
			ReasoningEffort: &reasoning,
			YoloMode:        &yolo,
		},
	}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	updated := readFileString(t, path)
	for _, preserved := range []string{
		"# project header\n[mcp]\nenabled_servers = [\"figma\"] # keep MCP comment",
		"[unknown.table]\nowner = \"keep-me\"",
		"[agents.codex.primary_session] # keep table comment",
		"model = \"old-model\" # keep model comment",
		"# keep comment before following table\n[following]\nvalue = 42",
	} {
		if !strings.Contains(updated, preserved) {
			t.Fatalf("updated project config did not preserve %q:\n%s", preserved, updated)
		}
	}
	if !strings.Contains(updated, "yolo_mode = false # keep yolo comment") {
		t.Fatalf("explicit false did not update yolo_mode in place:\n%s", updated)
	}
	if !strings.Contains(updated, "reasoning_effort = 'xhigh'\n") {
		t.Fatalf("missing reasoning_effort insertion:\n%s", updated)
	}
	parsed, err := parseProjectConfig([]byte(updated), path)
	if err != nil {
		t.Fatalf("parseProjectConfig(updated): %v", err)
	}
	if parsed.PrimarySession.Model == nil || *parsed.PrimarySession.Model != "old-model" {
		t.Fatalf("omitted model was not preserved: %+v", parsed.PrimarySession)
	}
	if parsed.PrimarySession.ReasoningEffort == nil || *parsed.PrimarySession.ReasoningEffort != reasoning {
		t.Fatalf("reasoning_effort = %+v, want %q", parsed.PrimarySession.ReasoningEffort, reasoning)
	}
	if parsed.PrimarySession.YoloMode == nil || *parsed.PrimarySession.YoloMode {
		t.Fatalf("yolo_mode = %+v, want explicit false", parsed.PrimarySession.YoloMode)
	}
	assertFileBytes(t, customCodexConfig, []byte("model = \"custom-project-model\"\n"))
}

func TestSetupLocalCreatesPrimarySessionProjectConfig(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	model := "gpt-5.6-terra"
	yolo := false

	if err := Setup(Options{
		Layout: layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{
			Model:    &model,
			YoloMode: &yolo,
		},
	}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	updated := readFileString(t, path)
	for _, want := range []string{
		"[agents.codex.primary_session]",
		"model = 'gpt-5.6-terra'",
		"yolo_mode = false",
	} {
		if !strings.Contains(updated, want) {
			t.Fatalf("created project config missing %q:\n%s", want, updated)
		}
	}
}

func TestRenderPrimarySessionUpdatesExistingStringsAndAddsYolo(t *testing.T) {
	path := filepath.Join(t.TempDir(), projectConfigFileName)
	original := []byte(`[agents.codex.primary_session]
model = "old-model" # model formatting
reasoning_effort = "medium" # effort formatting
`)
	parsed, err := parseProjectConfig(original, path)
	if err != nil {
		t.Fatalf("parseProjectConfig(original): %v", err)
	}
	model := "new-model"
	reasoning := "xhigh"
	yolo := false

	updated, err := renderCodexPrimarySessionSetup(original, path, parsed, CodexPrimarySessionSetup{
		Model:           &model,
		ReasoningEffort: &reasoning,
		YoloMode:        &yolo,
	})
	if err != nil {
		t.Fatalf("renderCodexPrimarySessionSetup: %v", err)
	}
	body := string(updated)
	for _, want := range []string{
		"model = 'new-model' # model formatting",
		"reasoning_effort = 'xhigh' # effort formatting",
		"yolo_mode = false\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("updated primary session missing %q:\n%s", want, body)
		}
	}
}

func TestRenderPrimarySessionPreservesCRLFWhenAddingField(t *testing.T) {
	path := filepath.Join(t.TempDir(), projectConfigFileName)
	original := []byte("[agents.codex.primary_session]\r\nmodel = \"gpt-5.6-terra\"\r\n")
	parsed, err := parseProjectConfig(original, path)
	if err != nil {
		t.Fatalf("parseProjectConfig(original): %v", err)
	}
	reasoning := "xhigh"

	updated, err := renderCodexPrimarySessionSetup(original, path, parsed, CodexPrimarySessionSetup{
		ReasoningEffort: &reasoning,
	})
	if err != nil {
		t.Fatalf("renderCodexPrimarySessionSetup: %v", err)
	}
	if !bytes.Contains(updated, []byte("\r\nreasoning_effort = 'xhigh'\r\n")) {
		t.Fatalf("added field did not use CRLF: %q", updated)
	}
	if bytes.Contains(bytes.ReplaceAll(updated, []byte("\r\n"), nil), []byte("\n")) {
		t.Fatalf("updated project config contains a lone LF: %q", updated)
	}
}

func TestSetupLocalClearAbsentPrimarySessionDoesNotCreateProjectConfig(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{
		Layout:              layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{Clear: true},
		projectConfigAtomicWriter: func(string, []byte, os.FileMode) error {
			t.Fatal("atomic writer called while clearing an absent primary session")
			return nil
		},
	}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(project, ".agents", ".configs", projectConfigFileName)); !os.IsNotExist(err) {
		t.Fatalf("clear created an absent project config: %v", err)
	}
}

func TestSetupLocalClearRemovesOnlyPrimarySessionTable(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := `# top-level comment
[mcp]
enabled_servers = ["figma"]

[agents.codex.primary_session]
model = "gpt-5.6-terra"
# comment inside the removed table remains user content
yolo_mode = false

[unrelated]
keep = "yes"
`
	mustMkdir(t, filepath.Dir(path))
	mustWrite(t, path, original)
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{
		Layout:              layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{Clear: true},
	}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	updated := readFileString(t, path)
	for _, removed := range []string{"[agents.codex.primary_session]", "model =", "yolo_mode ="} {
		if strings.Contains(updated, removed) {
			t.Fatalf("clear left %q behind:\n%s", removed, updated)
		}
	}
	for _, preserved := range []string{
		"# top-level comment\n[mcp]\nenabled_servers = [\"figma\"]",
		"# comment inside the removed table remains user content",
		"[unrelated]\nkeep = \"yes\"",
	} {
		if !strings.Contains(updated, preserved) {
			t.Fatalf("clear did not preserve %q:\n%s", preserved, updated)
		}
	}
	parsed, err := parseProjectConfig([]byte(updated), path)
	if err != nil {
		t.Fatalf("parseProjectConfig(cleared): %v", err)
	}
	if codexPrimarySessionSourcePresent(parsed.PrimarySession) {
		t.Fatalf("primary session still present after clear: %+v", parsed.PrimarySession)
	}
}

func TestSetupLocalValidationFailuresLeaveProjectConfigByteIdentical(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		setup     CodexPrimarySessionSetup
		wantField string
	}{
		{
			name:      "invalid TOML",
			body:      "[mcp\nenabled_servers = []\n",
			setup:     CodexPrimarySessionSetup{Model: stringPointer("gpt-5.6-terra")},
			wantField: projectConfigParseField,
		},
		{
			name:      "wrong existing yolo type",
			body:      "[agents.codex.primary_session]\nyolo_mode = \"false\"\n",
			setup:     CodexPrimarySessionSetup{Model: stringPointer("gpt-5.6-terra")},
			wantField: codexPrimaryYoloModeField,
		},
		{
			name:      "wrong existing model type",
			body:      "[agents.codex.primary_session]\nmodel = 56\n",
			setup:     CodexPrimarySessionSetup{YoloMode: boolPointer(false)},
			wantField: codexPrimaryModelField,
		},
		{
			name:      "wrong existing reasoning type",
			body:      "[agents.codex.primary_session]\nreasoning_effort = true\n",
			setup:     CodexPrimarySessionSetup{YoloMode: boolPointer(false)},
			wantField: codexPrimaryReasoningEffortField,
		},
		{
			name:      "empty model flag",
			body:      "[mcp]\nenabled_servers = []\n",
			setup:     CodexPrimarySessionSetup{Model: stringPointer("  ")},
			wantField: codexPrimaryModelField,
		},
		{
			name:      "empty reasoning flag",
			body:      "[mcp]\nenabled_servers = []\n",
			setup:     CodexPrimarySessionSetup{ReasoningEffort: stringPointer("")},
			wantField: codexPrimaryReasoningEffortField,
		},
		{
			name:      "clear conflicts with set",
			body:      "[agents.codex.primary_session]\nmodel = \"existing\"\n",
			setup:     CodexPrimarySessionSetup{Clear: true, Model: stringPointer("replacement")},
			wantField: codexPrimarySessionField,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := seedSourceRepo(t)
			project := t.TempDir()
			path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
			mustMkdir(t, filepath.Dir(path))
			original := []byte(test.body)
			if err := os.WriteFile(path, original, 0o640); err != nil {
				t.Fatalf("WriteFile(%s): %v", path, err)
			}
			layout, err := LocalLayout(source, project)
			if err != nil {
				t.Fatalf("LocalLayout: %v", err)
			}

			err = Setup(Options{Layout: layout, PrimarySessionSetup: test.setup})
			if err == nil {
				t.Fatal("Setup unexpectedly succeeded")
			}
			for _, want := range []string{path, "field " + test.wantField} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("Setup error %q missing %q", err, want)
				}
			}
			assertFileBytes(t, path, original)
		})
	}
}

func TestSetupGlobalRejectsPrimarySessionFlagsBeforeSync(t *testing.T) {
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}

	err = Setup(Options{
		Layout:              layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{Model: stringPointer("gpt-5.6-terra")},
	})
	if err == nil || !strings.Contains(err.Error(), "local-only") || !strings.Contains(err.Error(), codexPrimarySessionField) {
		t.Fatalf("Setup global error = %v, want local-only field error", err)
	}
	if _, statErr := os.Lstat(filepath.Join(home, ".agents")); !os.IsNotExist(statErr) {
		t.Fatalf("global setup mutated destination before rejection: %v", statErr)
	}
}

func TestSetupLocalAtomicWriteFailurePreservesOriginal(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := []byte("[mcp]\nenabled_servers = [\"figma\"]\n")
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	writeCalls := 0

	err = Setup(Options{
		Layout:              layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{Model: stringPointer("gpt-5.6-terra")},
		projectConfigAtomicWriter: func(string, []byte, os.FileMode) error {
			writeCalls++
			return errors.New("simulated atomic replacement failure")
		},
	})
	if err == nil {
		t.Fatal("Setup unexpectedly succeeded")
	}
	for _, want := range []string{path, "field " + codexPrimarySessionField, "simulated atomic replacement failure"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Setup error %q missing %q", err, want)
		}
	}
	if writeCalls != 1 {
		t.Fatalf("atomic writer calls = %d, want 1", writeCalls)
	}
	assertFileBytes(t, path, original)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s): %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("project config mode = %o, want 640", got)
	}
}

func TestProjectConfigReplacementFailurePreservesOriginalAndCleansTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, projectConfigFileName)
	original := []byte("[mcp]\nenabled_servers = [\"figma\"]\n")
	updated := []byte("[agents.codex.primary_session]\nmodel = \"gpt-5.6-terra\"\n")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	replaceCalls := 0

	err := writeProjectConfigAtomicallyWithReplace(
		path,
		updated,
		0o640,
		func(source, destination string) error {
			replaceCalls++
			if destination != path {
				t.Fatalf("replacement destination = %q, want %q", destination, path)
			}
			if filepath.Dir(source) != filepath.Dir(destination) {
				t.Fatalf("replacement paths are not in the same directory: %q -> %q", source, destination)
			}
			staged, readErr := os.ReadFile(source)
			if readErr != nil {
				t.Fatalf("ReadFile(staged %s): %v", source, readErr)
			}
			if !bytes.Equal(staged, updated) {
				t.Fatalf("staged bytes = %q, want %q", staged, updated)
			}
			return errors.New("simulated platform replacement failure")
		},
	)
	if err == nil || !strings.Contains(err.Error(), "simulated platform replacement failure") {
		t.Fatalf("writeProjectConfigAtomicallyWithReplace error = %v, want replacement failure", err)
	}
	if replaceCalls != 1 {
		t.Fatalf("replacement calls = %d, want 1", replaceCalls)
	}
	assertFileBytes(t, path, original)
	temporaryFiles, globErr := filepath.Glob(filepath.Join(directory, "."+projectConfigFileName+".tmp-*"))
	if globErr != nil {
		t.Fatalf("Glob temporary project configs: %v", globErr)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary project configs remain after replacement failure: %v", temporaryFiles)
	}
}

func TestPreparedProjectConfigRefusesConcurrentOverwrite(t *testing.T) {
	project := t.TempDir()
	layout, err := LocalLayout(project, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := []byte("[mcp]\nenabled_servers = []\n")
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	prepared, err := prepareCodexPrimarySessionSetup(
		layout,
		CodexPrimarySessionSetup{Model: stringPointer("gpt-5.6-terra")},
		nil,
	)
	if err != nil {
		t.Fatalf("prepareCodexPrimarySessionSetup: %v", err)
	}
	concurrent := []byte("[mcp]\nenabled_servers = [\"figma\"]\n")
	if err := os.WriteFile(path, concurrent, 0o640); err != nil {
		t.Fatalf("WriteFile(concurrent %s): %v", path, err)
	}

	err = commitPreparedProjectConfig(prepared, nil)
	if err == nil || !strings.Contains(err.Error(), "changed while setup was running") {
		t.Fatalf("commitPreparedProjectConfig error = %v, want concurrent-change rejection", err)
	}
	assertFileBytes(t, path, concurrent)
}

func TestSetupLocalSamePrimaryValueAvoidsRewrite(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := []byte("[agents.codex.primary_session]\nmodel = \"same-model\" # preserve quoting\n")
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{
		Layout:              layout,
		PrimarySessionSetup: CodexPrimarySessionSetup{Model: stringPointer("same-model")},
		projectConfigAtomicWriter: func(string, []byte, os.FileMode) error {
			t.Fatal("atomic writer called for a byte-identical update")
			return nil
		},
	}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	assertFileBytes(t, path, original)
}

func TestSetupLocalClaudePrimarySessionMutatesOnlyClaudeTable(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := `# project header
[mcp]
enabled_servers = ["figma"] # keep MCP

[agents.codex.primary_session] # keep Codex comment
model = "gpt-5.6-terra" # keep Codex model
reasoning_effort = "xhigh"
yolo_mode = true

[agents.claude.primary_session] # keep Claude comment
model = "claude-old" # keep Claude model comment

[unrelated]
owner = "preserve"
`
	mustMkdir(t, filepath.Dir(path))
	mustWrite(t, path, original)
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	model := "claude-opus-4-6"
	if err := Setup(Options{
		Layout:                    layout,
		ClaudePrimarySessionSetup: ClaudePrimarySessionSetup{Model: &model},
	}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	updated := readFileString(t, path)
	for _, preserved := range []string{
		"[mcp]\nenabled_servers = [\"figma\"] # keep MCP",
		"[agents.codex.primary_session] # keep Codex comment\nmodel = \"gpt-5.6-terra\" # keep Codex model\nreasoning_effort = \"xhigh\"\nyolo_mode = true",
		"[unrelated]\nowner = \"preserve\"",
	} {
		if !strings.Contains(updated, preserved) {
			t.Fatalf("Claude-only mutation did not preserve %q:\n%s", preserved, updated)
		}
	}
	if !strings.Contains(updated, "model = 'claude-opus-4-6' # keep Claude model comment") {
		t.Fatalf("Claude model was not updated in place:\n%s", updated)
	}
	parsed, err := parseProjectConfig([]byte(updated), path)
	if err != nil {
		t.Fatalf("parseProjectConfig(updated): %v", err)
	}
	if parsed.PrimarySession.Model == nil || *parsed.PrimarySession.Model != "gpt-5.6-terra" || parsed.PrimarySession.YoloMode == nil || !*parsed.PrimarySession.YoloMode {
		t.Fatalf("Codex policy changed during Claude-only setup: %#v", parsed.PrimarySession)
	}
	if parsed.ClaudePrimarySession.Model == nil || *parsed.ClaudePrimarySession.Model != model {
		t.Fatalf("Claude policy = %#v, want %q", parsed.ClaudePrimarySession, model)
	}
}

func TestSetupLocalClearClaudePrimarySessionPreservesCodexAndOtherTOML(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	original := `[mcp]
enabled_servers = ["figma"]

[agents.codex.primary_session]
model = "gpt-5.6-terra"
reasoning_effort = "xhigh"
yolo_mode = true

[agents.claude.primary_session]
model = "claude-opus-4-6"
# preserved user comment from cleared table

[unrelated]
keep = "yes"
`
	mustMkdir(t, filepath.Dir(path))
	mustWrite(t, path, original)
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{
		Layout:                    layout,
		ClaudePrimarySessionSetup: ClaudePrimarySessionSetup{Clear: true},
	}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	updated := readFileString(t, path)
	for _, removed := range []string{"[agents.claude.primary_session]", "model = \"claude-opus-4-6\""} {
		if strings.Contains(updated, removed) {
			t.Fatalf("clear left %q behind:\n%s", removed, updated)
		}
	}
	for _, preserved := range []string{
		"[mcp]\nenabled_servers = [\"figma\"]",
		"[agents.codex.primary_session]\nmodel = \"gpt-5.6-terra\"\nreasoning_effort = \"xhigh\"\nyolo_mode = true",
		"# preserved user comment from cleared table",
		"[unrelated]\nkeep = \"yes\"",
	} {
		if !strings.Contains(updated, preserved) {
			t.Fatalf("Claude clear did not preserve %q:\n%s", preserved, updated)
		}
	}
	parsed, err := parseProjectConfig([]byte(updated), path)
	if err != nil {
		t.Fatalf("parseProjectConfig(updated): %v", err)
	}
	if claudePrimarySessionSourcePresent(parsed.ClaudePrimarySession) {
		t.Fatalf("Claude primary session still present: %#v", parsed.ClaudePrimarySession)
	}
	if parsed.PrimarySession.Model == nil || *parsed.PrimarySession.Model != "gpt-5.6-terra" {
		t.Fatalf("Codex policy changed during Claude clear: %#v", parsed.PrimarySession)
	}
}

func TestSetupLocalRejectsInvalidClaudePrimarySessionFlagsBeforeWrite(t *testing.T) {
	tests := []struct {
		name      string
		setup     ClaudePrimarySessionSetup
		wantField string
	}{
		{name: "empty model", setup: ClaudePrimarySessionSetup{Model: stringPointer("  ")}, wantField: claudePrimaryModelField},
		{name: "clear conflicts with model", setup: ClaudePrimarySessionSetup{Clear: true, Model: stringPointer("claude-opus-4-6")}, wantField: claudePrimarySessionField},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := seedSourceRepo(t)
			project := t.TempDir()
			path := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
			original := []byte("[agents.codex.primary_session]\nmodel = \"gpt-5.6-terra\"\n")
			mustMkdir(t, filepath.Dir(path))
			if err := os.WriteFile(path, original, 0o640); err != nil {
				t.Fatalf("WriteFile(%s): %v", path, err)
			}
			layout, err := LocalLayout(source, project)
			if err != nil {
				t.Fatalf("LocalLayout: %v", err)
			}
			err = Setup(Options{Layout: layout, ClaudePrimarySessionSetup: test.setup})
			if err == nil || !strings.Contains(err.Error(), "field "+test.wantField) {
				t.Fatalf("Setup error = %v, want field %s", err, test.wantField)
			}
			assertFileBytes(t, path, original)
		})
	}
}

func TestProjectConfigTextEditsRejectInvalidRanges(t *testing.T) {
	tests := []struct {
		name  string
		edits []textEdit
	}{
		{
			name:  "out of bounds",
			edits: []textEdit{{span: textSpan{start: 0, end: 4}}},
		},
		{
			name: "overlap",
			edits: []textEdit{
				{span: textSpan{start: 0, end: 2}},
				{span: textSpan{start: 1, end: 3}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := applyTextEdits([]byte("abc"), test.edits); err == nil {
				t.Fatal("applyTextEdits unexpectedly succeeded")
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s changed unexpectedly\nwant: %q\n got: %q", path, want, got)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}
