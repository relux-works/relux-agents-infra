package infra

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Claude's reasoning_effort is parsed like the codex field: present values
// are normalised to the provider's lowercase vocabulary, and the table stays
// an explicit-field contract.
func TestParseProjectConfigAcceptsClaudeReasoningEffort(t *testing.T) {
	path := "/project/.agents/.configs/project-config.toml"
	config, err := parseProjectConfig([]byte("[agents.claude.primary_session]\nreasoning_effort = \"High\"\n"), path)
	if err != nil {
		t.Fatalf("parseProjectConfig: %v", err)
	}
	if config.ClaudePrimarySession.ReasoningEffort == nil || *config.ClaudePrimarySession.ReasoningEffort != "high" {
		t.Fatalf("reasoning effort = %#v, want normalised \"high\"", config.ClaudePrimarySession.ReasoningEffort)
	}
	if config.ClaudePrimarySession.Model != nil || config.ClaudePrimarySession.YoloMode != nil {
		t.Fatalf("unrelated fields populated: %#v", config.ClaudePrimarySession)
	}
}

// The project reasoning effort reaches the composed argv as --effort right
// after the model and before the yolo flag; an explicit CLI --effort
// suppresses it, so the argv never carries two effort selections.
func TestBuildClaudeLaunchPlanResolvesProjectReasoningEffort(t *testing.T) {
	tests := []struct {
		name            string
		primaryConfig   string
		args            []string
		wantArgs        []string
		wantSource      string
		wantValue       string
		wantApplication ClaudePrimarySessionApplication
	}{
		{
			name:            "project effort applied after model before yolo",
			primaryConfig:   "model = \"claude-opus-5\"\nreasoning_effort = \"high\"\nyolo_mode = true\n",
			args:            []string{"inspect"},
			wantArgs:        []string{"--model", "claude-opus-5", "--effort", "high", claudeDangerouslySkipPermissions, "inspect"},
			wantValue:       "high",
			wantApplication: ClaudePrimarySessionApplied,
		},
		{
			name:            "explicit effort suppresses the project value",
			primaryConfig:   "model = \"claude-opus-5\"\nreasoning_effort = \"high\"\n",
			args:            []string{"--effort", "low", "inspect"},
			wantArgs:        []string{"--model", "claude-opus-5", "--effort", "low", "inspect"},
			wantSource:      "cli:--effort",
			wantValue:       "low",
			wantApplication: ClaudePrimarySessionSuppressedByCLI,
		},
		{
			name:            "explicit attached effort suppresses the project value",
			primaryConfig:   "reasoning_effort = \"xhigh\"\n",
			args:            []string{"--effort=medium"},
			wantArgs:        []string{"--effort=medium"},
			wantSource:      "cli:--effort",
			wantValue:       "medium",
			wantApplication: ClaudePrimarySessionSuppressedByCLI,
		},
		{
			name:            "no project effort composes nothing",
			primaryConfig:   "model = \"claude-opus-5\"\n",
			args:            []string{"inspect"},
			wantArgs:        []string{"--model", "claude-opus-5", "inspect"},
			wantSource:      "native",
			wantApplication: ClaudePrimarySessionNotConfigured,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			start := t.TempDir()
			mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
			configPath := filepath.Join(start, ".agents", ".configs", projectConfigFileName)
			mustWrite(t, configPath, "[agents.claude.primary_session]\n"+tt.primaryConfig)

			plan, err := BuildClaudeLaunchPlan(start, home, tt.args)
			if err != nil {
				t.Fatalf("BuildClaudeLaunchPlan: %v", err)
			}
			if !reflect.DeepEqual(plan.Args, tt.wantArgs) {
				t.Fatalf("Args = %#v, want %#v", plan.Args, tt.wantArgs)
			}
			if got := countArg(plan.Args, "--effort"); got > 1 {
				t.Fatalf("--effort count = %d in %#v, want at most one", got, plan.Args)
			}
			resolution := plan.PrimarySessionResolution.ReasoningEffort
			if resolution.ProjectApplication != tt.wantApplication {
				t.Fatalf("effort resolution = %#v, want application %q", resolution, tt.wantApplication)
			}
			if tt.wantApplication != ClaudePrimarySessionNotConfigured && resolution.ProjectSource != configPath {
				t.Fatalf("project source = %q, want %q", resolution.ProjectSource, configPath)
			}
			wantSource := tt.wantSource
			if wantSource == "" {
				wantSource = configPath
			}
			if resolution.EffectiveSource != wantSource {
				t.Fatalf("effective source = %q, want %q", resolution.EffectiveSource, wantSource)
			}
			if resolution.EffectiveValue != tt.wantValue || resolution.EffectiveValueKnown != (tt.wantValue != "") {
				t.Fatalf("effective value = %q (known %t), want %q", resolution.EffectiveValue, resolution.EffectiveValueKnown, tt.wantValue)
			}
		})
	}
}

// A malformed project reasoning effort fails the plan before launch, naming
// the config path and the exact field.
func TestBuildClaudeLaunchPlanRejectsInvalidProjectReasoningEffort(t *testing.T) {
	home := t.TempDir()
	start := t.TempDir()
	mustMkdir(t, filepath.Join(start, ".agents", ".configs"))
	configPath := filepath.Join(start, ".agents", ".configs", projectConfigFileName)
	mustWrite(t, configPath, "[agents.claude.primary_session]\nreasoning_effort = \"ultra\"\n")
	_, err := BuildClaudeLaunchPlan(start, home, []string{"inspect"})
	if err == nil || !strings.Contains(err.Error(), configPath) || !strings.Contains(err.Error(), claudePrimaryReasoningEffortField) {
		t.Fatalf("BuildClaudeLaunchPlan error = %v, want source path and reasoning field", err)
	}
}

// The compose plan reports the applied project effort with the config path
// as its source, and keeps reporting the CLI source when the CLI won.
func TestPrimarySessionLaunchPlanReportsClaudeProjectReasoningEffort(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	mustMkdir(t, filepath.Join(project, ".agents", ".configs"))
	configPath := filepath.Join(project, ".agents", ".configs", projectConfigFileName)
	mustWrite(t, configPath, "[agents.claude.primary_session]\nmodel = \"claude-opus-5\"\nreasoning_effort = \"high\"\nyolo_mode = true\n")
	// The plan reports the symlink-resolved config path (macOS temp dirs).
	if resolved, err := filepath.EvalSymlinks(configPath); err == nil {
		configPath = resolved
	}

	applied, err := BuildPrimarySessionLaunchPlan("claude", project, home, nil, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan: %v", err)
	}
	if got := applied.Resolved.Reasoning; got.Value == nil || *got.Value != "high" || got.Source != configPath {
		t.Fatalf("resolved reasoning = %#v, want high from %s", got, configPath)
	}
	if want := []string{"--model", "claude-opus-5", "--effort", "high", claudeDangerouslySkipPermissions}; !reflect.DeepEqual(applied.LaunchVariants.ManagedHost.Argv, want) {
		t.Fatalf("managed host argv = %#v, want %#v", applied.LaunchVariants.ManagedHost.Argv, want)
	}

	suppressed, err := BuildPrimarySessionLaunchPlan("claude", project, home, []string{"--effort", "low"}, ChildLaunchCompositionProducer{}, fakePrimarySessionLookPath(t))
	if err != nil {
		t.Fatalf("BuildPrimarySessionLaunchPlan with CLI effort: %v", err)
	}
	if got := suppressed.Resolved.Reasoning; got.Value == nil || *got.Value != "low" || got.Source != "cli:--effort" {
		t.Fatalf("resolved reasoning = %#v, want low from cli:--effort", got)
	}
	if countArg(suppressed.LaunchVariants.ManagedHost.Argv, "--effort") != 1 {
		t.Fatalf("managed host argv carries a duplicated effort: %#v", suppressed.LaunchVariants.ManagedHost.Argv)
	}
}
