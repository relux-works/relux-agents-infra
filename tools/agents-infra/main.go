package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/relux-works/alexis-agents-infra/tools/agents-infra/internal/infra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

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
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	layout, err := resolveLayout(mode, *sourceDir, *homeDir, *projectDir, fs.Args())
	if err != nil {
		return err
	}
	return infra.Setup(infra.Options{
		Layout:          layout,
		NoSync:          *noSync,
		CodexConfigMode: infra.CodexConfigMode(*codexConfigMode),
		Stdout:          os.Stdout,
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
	report := infra.Doctor(layout)
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
	fmt.Fprintf(os.Stdout, "codex_config_effective: %s\n", report.CodexConfigEffective)
	if report.Layout.Mode == infra.ModeLocal {
		fmt.Fprintf(os.Stdout, "codex_config_shadowing_global: %t\n", report.CodexConfigShadowsGlobal)
		if report.CodexConfigShadowsGlobal {
			if report.CodexConfigLinked {
				fmt.Fprintf(os.Stdout, "codex_config_action: managed project-local .codex/config.toml is active; use --codex-config=global to remove it if unintended\n")
			} else {
				fmt.Fprintf(os.Stdout, "codex_config_action: custom project-local .codex/config.toml overrides global Codex config; remove it if unintended\n")
			}
		}
	}
	fmt.Fprintf(os.Stdout, "helpers_linked: %t\n", report.HelpersLinked)
	fmt.Fprintf(os.Stdout, "infra_skill_link: %t\n", report.InfraSkillLink)
	return nil
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
  agents-infra setup local [PROJECT_DIR] [--source-dir DIR] [--project-dir DIR] [--no-sync] [--codex-config preserve|global|local]
  agents-infra refresh-links --agents-dir DIR --claude-dir DIR --codex-dir DIR --bin-dir DIR [--mode global|local] [--codex-config preserve|global|local]
  agents-infra doctor global [--home-dir DIR]
  agents-infra doctor local [PROJECT_DIR] [--project-dir DIR]`
}
