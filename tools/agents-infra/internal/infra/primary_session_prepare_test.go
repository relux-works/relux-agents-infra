package infra

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPreparePrimarySessionCodexPreservesAbsentProjectConfig(t *testing.T) {
	project := preparedRuntimeFixture(t)
	report, err := PreparePrimarySession(
		"codex",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !report.LocalRuntimePresent || !report.CodexProjectRendered || report.CodexConfigGenerated {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Artifacts) != 3 {
		t.Fatalf("artifacts = %#v", report.Artifacts)
	}
	if states := []string{
		report.Artifacts[0].State,
		report.Artifacts[1].State,
		report.Artifacts[2].State,
	}; !reflect.DeepEqual(states, []string{"rendered", "rendered", "absent"}) {
		t.Fatalf("artifact states = %#v", states)
	}
	if report.Artifacts[0].SHA256 == "" || report.Artifacts[1].SHA256 == "" || report.Artifacts[2].SHA256 != "" {
		t.Fatalf("artifacts = %#v", report.Artifacts)
	}
	if _, err := os.Stat(filepath.Join(project, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("prepare changed absent project config: %v", err)
	}
}

func TestPreparePrimarySessionCodexPreservesManagedProjectConfigBytes(t *testing.T) {
	project := preparedRuntimeFixture(t)
	layout, err := LocalLayout("", project)
	if err != nil {
		t.Fatal(err)
	}
	if err := setupCodex(layout, CodexConfigModeLocal, nil); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(project, ".codex", "config.toml")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	report, err := PreparePrimarySession(
		"codex",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("managed project config changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if !report.CodexConfigGenerated || report.Artifacts[2].State != "preserved" || report.Artifacts[2].SHA256 == "" {
		t.Fatalf("report = %#v", report)
	}
}

func TestPreparePrimarySessionCodexPreservesCustomProjectConfigBytes(t *testing.T) {
	project := preparedRuntimeFixture(t)
	configPath := filepath.Join(project, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	before := []byte("model = \"custom-project-model\"\n[profiles.local]\nmodel = \"custom-profile\"\n")
	if err := os.WriteFile(configPath, before, 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := PreparePrimarySession(
		"codex",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("custom project config changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if report.CodexConfigGenerated || report.Artifacts[2].State != "preserved" || report.Artifacts[2].SHA256 == "" {
		t.Fatalf("report = %#v", report)
	}
}

func TestPreparePrimarySessionCodexPreservesProjectConfigSymlink(t *testing.T) {
	project := preparedRuntimeFixture(t)
	configPath := filepath.Join(project, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join("..", ".agents", ".configs", "codex-config.toml")
	if err := os.Symlink(target, configPath); err != nil {
		t.Fatal(err)
	}

	report, err := PreparePrimarySession(
		"codex",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Readlink(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if after != target {
		t.Fatalf("project config symlink target changed: got %q, want %q", after, target)
	}
	if report.CodexConfigGenerated || report.Artifacts[2].State != "preserved" || report.Artifacts[2].Target != target {
		t.Fatalf("report = %#v", report)
	}
}

func TestPreparePrimarySessionClaudeRefreshesManagedProjectSurface(t *testing.T) {
	project := preparedRuntimeFixture(t)
	report, err := PreparePrimarySession(
		"claude",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !report.LocalRuntimePresent ||
		!report.ClaudeEntrypointRendered ||
		!report.ClaudeInstructionsLinked ||
		!report.ClaudeSettingsLinked {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Artifacts) != 3 {
		t.Fatalf("artifacts = %#v", report.Artifacts)
	}
	states := []string{report.Artifacts[0].State, report.Artifacts[1].State, report.Artifacts[2].State}
	if !reflect.DeepEqual(states, []string{"rendered", "linked", "linked"}) {
		t.Fatalf("artifact states = %#v", states)
	}
	if report.Artifacts[0].SHA256 == "" || report.Artifacts[1].Target == "" || report.Artifacts[2].Target == "" {
		t.Fatalf("artifacts = %#v", report.Artifacts)
	}
}

func TestPreparePrimarySessionWithoutLocalRuntimeIsExplicitNoop(t *testing.T) {
	project := t.TempDir()
	report, err := PreparePrimarySession(
		"codex",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.LocalRuntimePresent || len(report.Artifacts) != 0 {
		t.Fatalf("report = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(project, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("prepare created a provider surface without a local runtime: %v", err)
	}
}

func TestPreparePrimarySessionTreatsConfigOnlyAncestorAsNoop(t *testing.T) {
	configRoot := t.TempDir()
	configPath := filepath.Join(configRoot, ".agents", ".configs", "project-config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("[agents.codex.primary_session]\nmodel = \"gpt-test\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(configRoot, "nested", "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := PreparePrimarySession(
		"codex",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.LocalRuntimePresent || report.RuntimeProjectDir != "" || len(report.Artifacts) != 0 {
		t.Fatalf("config-only ancestor was treated as an installed runtime: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(configRoot, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("prepare changed a config-only ancestor: %v", err)
	}
}

func TestPreparePrimarySessionSkipsConfigOnlyAncestorForInstalledRuntime(t *testing.T) {
	runtimeRoot := preparedRuntimeFixture(t)
	configRoot := filepath.Join(runtimeRoot, "teams")
	configPath := filepath.Join(configRoot, ".agents", ".configs", "project-config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("[agents.codex.primary_session]\nmodel = \"gpt-test\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(configRoot, "nested", "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := PreparePrimarySession(
		"codex",
		project,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	canonicalRuntimeRoot, err := filepath.EvalSymlinks(runtimeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !report.LocalRuntimePresent || report.RuntimeProjectDir != canonicalRuntimeRoot {
		t.Fatalf("report roots = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(configRoot, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("prepare changed the nearer config-only ancestor: %v", err)
	}
}

func TestPreparePrimarySessionUsesNearestInstalledAncestorRuntime(t *testing.T) {
	project := preparedRuntimeFixture(t)
	nested := filepath.Join(project, "tools", "feature")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := PreparePrimarySession(
		"codex",
		nested,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	canonicalProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	canonicalNested, err := filepath.EvalSymlinks(nested)
	if err != nil {
		t.Fatal(err)
	}
	if report.ProjectDir != canonicalNested || report.RuntimeProjectDir != canonicalProject {
		t.Fatalf("report roots = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(canonicalProject, "AGENTS.md")); err != nil {
		t.Fatalf("ancestor runtime instructions were not prepared: %v", err)
	}
	if _, err := os.Stat(filepath.Join(canonicalProject, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("ancestor runtime config was unexpectedly created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(canonicalNested, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("nested provider surface was unexpectedly created: %v", err)
	}
}

func TestPreparePrimarySessionDoesNotTreatHomeRuntimeAsProjectRuntime(t *testing.T) {
	home := preparedRuntimeFixture(t)
	nested := filepath.Join(home, "src", "plain-project")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	report, err := PreparePrimarySession(
		"codex",
		nested,
		ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.LocalRuntimePresent || report.RuntimeProjectDir != "" || len(report.Artifacts) != 0 {
		t.Fatalf("home runtime leaked into project preparation: %#v", report)
	}
}

func preparedRuntimeFixture(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	for _, dir := range []string{
		filepath.Join(project, ".agents", ".configs"),
		filepath.Join(project, ".agents", ".instructions"),
		filepath.Join(project, ".agents", ".rules"),
		filepath.Join(project, ".agents", "skills", "example"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(project, ".agents", ".configs", "codex-config.toml"): `model = "gpt-test"

[profiles.fast]
model = "gpt-fast"
`,
		filepath.Join(project, ".agents", ".configs", "claude-settings.json"): "{}\n",
		filepath.Join(project, ".agents", ".instructions", "AGENTS.md"):       "# Managed instructions\n",
		filepath.Join(project, ".agents", ".rules", "default.rules"):          "allow\n",
		filepath.Join(project, ".agents", "skills", "example", "SKILL.md"):    "# Example\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	layout, err := LocalLayout("", project)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRuntimeReceipt(layout); err != nil {
		t.Fatal(err)
	}
	return project
}

// Prepare never fans out skills: it creates no provider skills surface on a
// runtime without one, and leaves a pre-existing provider skills surface
// byte- and target-identical on a runtime with one.
func TestPreparePrimarySessionLeavesSkillSurfacesUntouched(t *testing.T) {
	for _, provider := range []string{"claude", "codex"} {
		t.Run(provider+"/absent", func(t *testing.T) {
			project := preparedRuntimeFixture(t)
			if err := os.RemoveAll(filepath.Join(project, ".agents", "skills")); err != nil {
				t.Fatal(err)
			}

			report, err := PreparePrimarySession(
				provider,
				project,
				ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
			)
			if err != nil {
				t.Fatalf("prepare failed on a runtime with no skills tree: %v", err)
			}
			if !report.LocalRuntimePresent {
				t.Fatalf("report = %#v", report)
			}

			providerDir := ".claude"
			if provider == "codex" {
				providerDir = ".codex"
			}
			if _, err := os.Lstat(filepath.Join(project, providerDir, "skills")); !os.IsNotExist(err) {
				t.Fatalf("prepare invented a provider skills surface: %v", err)
			}
		})
		t.Run(provider+"/present", func(t *testing.T) {
			project := preparedRuntimeFixture(t)
			providerDir := ".claude"
			if provider == "codex" {
				providerDir = ".codex"
			}
			skillsDir := filepath.Join(project, providerDir, "skills")
			if err := os.MkdirAll(skillsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			customSkill := filepath.Join(skillsDir, "custom", "SKILL.md")
			if err := os.MkdirAll(filepath.Dir(customSkill), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(customSkill, []byte("custom skill\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			linkTarget := filepath.Join(project, ".agents", "skills", "example")
			if err := os.Symlink(linkTarget, filepath.Join(skillsDir, "example")); err != nil {
				t.Fatal(err)
			}

			report, err := PreparePrimarySession(
				provider,
				project,
				ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
			)
			if err != nil {
				t.Fatal(err)
			}
			if !report.LocalRuntimePresent {
				t.Fatalf("report = %#v", report)
			}
			data, err := os.ReadFile(customSkill)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "custom skill\n" {
				t.Fatalf("prepare rewrote provider skill content: %q", string(data))
			}
			target, err := os.Readlink(filepath.Join(skillsDir, "example"))
			if err != nil {
				t.Fatal(err)
			}
			if target != linkTarget {
				t.Fatalf("prepare rewrote provider skill link: got %q, want %q", target, linkTarget)
			}
		})
	}
}

// A fresh residual runtime carries no instruction inputs — setup no longer
// distributes them — so prepare creates the missing scaffold entrypoints
// itself and still reports real artifacts.
func TestPreparePrimarySessionCreatesMissingInstructionScaffold(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			project := preparedRuntimeFixture(t)
			if err := os.RemoveAll(filepath.Join(project, ".agents", ".instructions")); err != nil {
				t.Fatal(err)
			}

			report, err := PreparePrimarySession(
				provider,
				project,
				ChildLaunchCompositionProducer{Version: "test", Commit: "abc123"},
			)
			if err != nil {
				t.Fatalf("prepare failed on a runtime with no instruction inputs: %v", err)
			}
			if !report.LocalRuntimePresent || len(report.Artifacts) == 0 {
				t.Fatalf("report = %#v", report)
			}
			for _, name := range []string{"AGENTS.md", "INSTRUCTIONS.md"} {
				data, err := os.ReadFile(filepath.Join(project, ".agents", ".instructions", name))
				if err != nil {
					t.Fatalf("prepare did not create scaffold %s: %v", name, err)
				}
				if len(data) == 0 {
					t.Fatalf("prepare created an empty scaffold %s", name)
				}
			}
			for _, artifact := range report.Artifacts {
				if artifact.State == "absent" {
					continue
				}
				if artifact.Target == "" && artifact.SHA256 == "" {
					t.Fatalf("scaffolded artifact carries no evidence: %#v", artifact)
				}
			}
		})
	}
}
