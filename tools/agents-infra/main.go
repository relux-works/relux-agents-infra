package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

const callerCWDEnv = "AGENTS_INFRA_CALLER_CWD"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	switch args[0] {
	case "setup":
		return runSetup(args[1:])
	case "refresh-links":
		return runRefreshLinks(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "codex":
		return runCodex(args[1:])
	case "claude":
		return runClaude(args[1:])
	case "version", "--version":
		return runVersion()
	case "help", "-h", "--help":
		return usageError()
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText())
	}
}

func runSetup(args []string) error {
	if len(args) == 0 {
		return errors.New("setup requires mode: global or local")
	}
	mode := args[0]
	fs := flag.NewFlagSet("setup "+mode, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	sourceDir := fs.String("source-dir", "", "source repo directory")
	noSync := fs.Bool("no-sync", false, "skip repo sync")
	homeDir := fs.String("home-dir", "", "home directory for global setup")
	projectDir := fs.String("project-dir", "", "project directory for local setup")
	codexConfigMode := fs.String("codex-config", string(infra.CodexConfigModePreserve), "Codex config handling for local setup: preserve, global, or local")
	var primarySessionSetup infra.CodexPrimarySessionSetup
	var claudePrimarySessionSetup infra.ClaudePrimarySessionSetup
	fs.Func("codex-primary-model", "primary Codex model for this project", func(value string) error {
		primarySessionSetup.Model = &value
		return nil
	})
	fs.Func("codex-primary-reasoning-effort", "primary Codex reasoning effort for this project", func(value string) error {
		primarySessionSetup.ReasoningEffort = &value
		return nil
	})
	fs.Func("codex-yolo-mode", "persistent Codex yolo mode for this project: true or false", func(value string) error {
		var parsed bool
		switch value {
		case "true":
			parsed = true
		case "false":
			parsed = false
		default:
			return fmt.Errorf("expected true or false")
		}
		primarySessionSetup.YoloMode = &parsed
		return nil
	})
	clearPrimarySession := fs.Bool("clear-codex-primary-session", false, "remove this project's primary Codex session table")
	fs.Func("claude-primary-model", "primary Claude model for this project", func(value string) error {
		claudePrimarySessionSetup.Model = &value
		return nil
	})
	clearClaudePrimarySession := fs.Bool("clear-claude-primary-session", false, "remove this project's primary Claude session table")

	parseArgs := args[1:]
	var leadingProjectDir string
	if mode == string(infra.ModeLocal) && len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		leadingProjectDir = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	positionals := fs.Args()
	if leadingProjectDir != "" {
		positionals = append([]string{leadingProjectDir}, positionals...)
	}
	if mode == string(infra.ModeGlobal) && len(positionals) > 0 {
		return fmt.Errorf("setup global does not accept positional project directories: %q", positionals[0])
	}
	if mode == string(infra.ModeLocal) && len(positionals) > 1 {
		return fmt.Errorf("setup local accepts one project directory, got %q", positionals)
	}
	primarySessionSetup.Clear = *clearPrimarySession
	claudePrimarySessionSetup.Clear = *clearClaudePrimarySession

	layout, err := resolveLayout(mode, *sourceDir, *homeDir, *projectDir, positionals)
	if err != nil {
		return err
	}
	return infra.Setup(infra.Options{
		Layout:                    layout,
		NoSync:                    *noSync,
		CodexConfigMode:           infra.CodexConfigMode(*codexConfigMode),
		PrimarySessionSetup:       primarySessionSetup,
		ClaudePrimarySessionSetup: claudePrimarySessionSetup,
		Stdout:                    os.Stdout,
	})
}

func runRefreshLinks(args []string) error {
	fs := flag.NewFlagSet("refresh-links", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agentsDir := fs.String("agents-dir", "", "installed agents directory")
	claudeDir := fs.String("claude-dir", "", "claude directory")
	codexDir := fs.String("codex-dir", "", "codex directory")
	binDir := fs.String("bin-dir", "", "helper bin directory")
	mode := fs.String("mode", string(infra.ModeGlobal), "layout mode: global or local")
	codexConfigMode := fs.String("codex-config", string(infra.CodexConfigModePreserve), "Codex config handling for local refresh: preserve, global, or local")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *agentsDir == "" || *claudeDir == "" || *codexDir == "" || *binDir == "" {
		return fmt.Errorf("refresh-links requires --agents-dir, --claude-dir, --codex-dir, and --bin-dir")
	}
	layout := infra.Layout{
		Mode:      infra.Mode(*mode),
		AgentsDir: *agentsDir,
		ClaudeDir: *claudeDir,
		CodexDir:  *codexDir,
		BinDir:    *binDir,
	}
	return infra.RefreshLinks(infra.Options{
		Layout:          layout,
		CodexConfigMode: infra.CodexConfigMode(*codexConfigMode),
		Stdout:          os.Stdout,
	})
}

func runDoctor(args []string) error {
	if len(args) == 0 {
		return errors.New("doctor requires mode: global or local")
	}
	mode := args[0]
	fs := flag.NewFlagSet("doctor "+mode, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	homeDir := fs.String("home-dir", "", "home directory for global doctor")
	projectDir := fs.String("project-dir", "", "project directory for local doctor")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	layout, err := resolveLayout(mode, "", *homeDir, *projectDir, fs.Args())
	if err != nil {
		return err
	}
	report, doctorErr := infra.Doctor(layout)
	fmt.Fprintf(os.Stdout, "mode: %s\n", report.Layout.Mode)
	fmt.Fprintf(os.Stdout, "agents_dir: %s\n", report.Layout.AgentsDir)
	fmt.Fprintf(os.Stdout, "claude_dir: %s\n", report.Layout.ClaudeDir)
	fmt.Fprintf(os.Stdout, "codex_dir: %s\n", report.Layout.CodexDir)
	fmt.Fprintf(os.Stdout, "bin_dir: %s\n", report.Layout.BinDir)
	fmt.Fprintf(os.Stdout, "git_free: %t\n", report.AgentsGitFree)
	fmt.Fprintf(os.Stdout, "claude_linked: %t\n", report.ClaudeLinked)
	fmt.Fprintf(os.Stdout, "codex_linked: %t\n", report.CodexLinked)
	fmt.Fprintf(os.Stdout, "codex_rendered: %t\n", report.CodexRendered)
	if report.Layout.Mode == infra.ModeLocal {
		fmt.Fprintf(os.Stdout, "codex_project_rendered: %t\n", report.CodexProjectRendered)
	}
	fmt.Fprintf(os.Stdout, "codex_config_present: %t\n", report.CodexConfigPresent)
	fmt.Fprintf(os.Stdout, "codex_config_linked: %t\n", report.CodexConfigLinked)
	fmt.Fprintf(os.Stdout, "codex_config_generated: %t\n", report.CodexConfigGenerated)
	fmt.Fprintf(os.Stdout, "codex_config_effective: %s\n", report.CodexConfigEffective)
	if report.Layout.Mode == infra.ModeLocal {
		fmt.Fprintf(os.Stdout, "codex_mcp_enabled: %s\n", strings.Join(report.CodexMCPEnabled, ","))
		fmt.Fprintf(os.Stdout, "codex_config_shadowing_global: %t\n", report.CodexConfigShadowsGlobal)
		fmt.Fprintf(os.Stdout, "codex_primary_config_valid: %t\n", report.CodexPrimaryConfigValid)
		if report.CodexPrimaryConfigValid {
			fmt.Fprintf(os.Stdout, "codex_primary_model: %s\n", report.CodexPrimarySession.Model.Value)
			fmt.Fprintf(os.Stdout, "codex_primary_model_source: %s\n", codexPrimaryStringSource(report.CodexPrimarySession.Model))
			fmt.Fprintf(os.Stdout, "codex_primary_reasoning_effort: %s\n", report.CodexPrimarySession.ReasoningEffort.Value)
			fmt.Fprintf(os.Stdout, "codex_primary_reasoning_effort_source: %s\n", codexPrimaryStringSource(report.CodexPrimarySession.ReasoningEffort))
			fmt.Fprintf(os.Stdout, "codex_primary_yolo_mode: %t\n", report.CodexPrimarySession.YoloMode.Value)
			fmt.Fprintf(os.Stdout, "codex_primary_yolo_mode_source: %s\n", codexPrimaryBoolSource(report.CodexPrimarySession.YoloMode))
		}
		fmt.Fprintf(os.Stdout, "claude_primary_config_valid: %t\n", report.ClaudePrimaryConfigValid)
		if report.ClaudePrimaryConfigValid {
			fmt.Fprintf(os.Stdout, "claude_primary_model: %s\n", report.ClaudePrimarySession.Model.Value)
			fmt.Fprintf(os.Stdout, "claude_primary_model_source: %s\n", claudePrimaryStringSource(report.ClaudePrimarySession.Model))
		}
		if report.CodexConfigShadowsGlobal {
			if report.CodexConfigGenerated {
				fmt.Fprintf(os.Stdout, "codex_config_action: generated project-local .codex/config.toml is active because local MCP opt-in is configured\n")
			} else if report.CodexConfigLinked {
				fmt.Fprintf(os.Stdout, "codex_config_action: managed project-local .codex/config.toml is active; use --codex-config=global to remove it if unintended\n")
			} else {
				fmt.Fprintf(os.Stdout, "codex_config_action: custom project-local .codex/config.toml overrides global Codex config; remove it if unintended\n")
			}
		}
	}
	fmt.Fprintf(os.Stdout, "helpers_linked: %t\n", report.HelpersLinked)
	fmt.Fprintf(os.Stdout, "infra_skill_link: %t\n", report.InfraSkillLink)
	return doctorErr
}

func codexPrimaryStringSource(value infra.CodexPrimarySessionStringValue) string {
	if value.Present {
		return value.Source
	}
	return "native"
}

func codexPrimaryBoolSource(value infra.CodexPrimarySessionBoolValue) string {
	if value.Present {
		return value.Source
	}
	return "default"
}

func claudePrimaryStringSource(value infra.ClaudePrimarySessionStringValue) string {
	if value.Present {
		return value.Source
	}
	return "native"
}

func runCodex(args []string) error {
	plan, err := infra.BuildCodexLaunchPlan(os.Getenv(callerCWDEnv), "", args)
	if err != nil {
		return err
	}
	rendered := infra.RenderCodexLaunchPlan(plan)
	if plan.PrintConfig {
		fmt.Fprint(os.Stdout, rendered)
		return nil
	}
	fmt.Fprint(os.Stderr, rendered)
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("find codex executable: %w", err)
	}
	cmd := exec.Command(codexPath, plan.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runClaude(args []string) error {
	plan, err := infra.BuildClaudeLaunchPlan(os.Getenv(callerCWDEnv), "", args)
	if err != nil {
		return err
	}
	rendered := infra.RenderClaudeLaunchPlan(plan)
	if plan.PrintConfig {
		fmt.Fprint(os.Stdout, rendered)
		return nil
	}
	fmt.Fprint(os.Stderr, rendered)
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("find claude executable: %w", err)
	}
	cmd := exec.Command(claudePath, plan.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runVersion() error {
	fmt.Fprintf(os.Stdout, "agents-infra %s commit=%s build_date=%s\n", Version, Commit, BuildDate)
	return nil
}

func resolveLayout(mode, sourceDir, homeDir, projectDir string, positional []string) (infra.Layout, error) {
	if sourceDir == "" {
		sourceDir = os.Getenv("AGENTS_INFRA_SOURCE_DIR")
	}
	switch mode {
	case string(infra.ModeGlobal):
		if homeDir == "" {
			var err error
			homeDir, err = os.UserHomeDir()
			if err != nil {
				return infra.Layout{}, fmt.Errorf("resolve home dir: %w", err)
			}
		}
		return infra.GlobalLayout(sourceDir, homeDir)
	case string(infra.ModeLocal):
		if projectDir == "" {
			if len(positional) > 0 {
				projectDir = positional[0]
			} else if callerCWD := os.Getenv(callerCWDEnv); callerCWD != "" {
				projectDir = callerCWD
			} else {
				projectDir = "."
			}
		}
		if sourceDir != "" {
			abs, err := filepath.Abs(sourceDir)
			if err == nil {
				sourceDir = abs
			}
		}
		return infra.LocalLayout(sourceDir, projectDir)
	default:
		return infra.Layout{}, fmt.Errorf("unknown mode %q", mode)
	}
}

func usageError() error {
	return errors.New(usageText())
}

func usageText() string {
	return `Usage:
  agents-infra version
  agents-infra setup global [--source-dir DIR] [--home-dir DIR] [--no-sync]
  agents-infra setup local [PROJECT_DIR] [--source-dir DIR] [--project-dir DIR] [--no-sync] [--codex-config preserve|global|local] [--codex-primary-model MODEL] [--codex-primary-reasoning-effort EFFORT] [--codex-yolo-mode=true|false] [--clear-codex-primary-session] [--claude-primary-model MODEL] [--clear-claude-primary-session]
  agents-infra refresh-links --agents-dir DIR --claude-dir DIR --codex-dir DIR --bin-dir DIR [--mode global|local] [--codex-config preserve|global|local]
  agents-infra doctor global [--home-dir DIR]
  agents-infra doctor local [PROJECT_DIR] [--project-dir DIR]
  agents-infra codex [--print-config] [-d|--danger|--yolo] [--] [CODEX_ARGS...]
  agents-infra claude [--print-config] [-d|--danger|--yolo] [--] [CLAUDE_ARGS...]`
}
