package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/attachments"
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
		code := 1
		if code, ok := attachments.ExitCode(err); ok {
			os.Exit(code)
		}
		var modelCheckFailure *infra.ModelCheckFailure
		if errors.As(err, &modelCheckFailure) {
			code = modelCheckFailure.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
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
	case "verify":
		return runVerify(args[1:])
	case "compose":
		return runCompose(args[1:])
	case "prepare":
		return runPrepare(args[1:])
	case "attachments":
		return runAttachments(args[1:])
	case "codex":
		return runCodex(args[1:])
	case "claude":
		return runClaude(args[1:])
	case "pi":
		return runPi(args[1:])
	case "target":
		return runTarget(args[1:])
	case "model-check":
		return runModelCheck(args[1:])
	case "version", "--version":
		return runVersion()
	case "help", "-h", "--help":
		return usageError()
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText())
	}
}

type repeatedStringFlag []string

func (f *repeatedStringFlag) String() string { return strings.Join(*f, ",") }
func (f *repeatedStringFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func runModelCheck(args []string) error {
	fs := flag.NewFlagSet("model-check", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	target := fs.String("target", "", "configured canonical entrypoint, for example qwen-infra")
	prompt := fs.String("prompt", "", "behavior-check prompt")
	outputDir := fs.String("output-dir", "", "explicit evidence output directory")
	deadline := fs.Duration("deadline", infra.DefaultModelCheckDeadline, "bounded execution deadline")
	var expectedTools repeatedStringFlag
	var expectedText repeatedStringFlag
	fs.Var(&expectedTools, "expect-tool", "required tool name; repeat for multiple expectations")
	fs.Var(&expectedText, "expect-text", "required final-response substring; repeat for multiple expectations")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("model-check does not accept positional arguments: %q", fs.Args())
	}
	startDir := os.Getenv(callerCWDEnv)
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve model-check caller cwd: %w", err)
		}
	}
	summary, err := infra.RunModelCheck(infra.ModelCheckOptions{
		ProjectDir:    startDir,
		Target:        *target,
		Prompt:        *prompt,
		OutputDir:     *outputDir,
		Deadline:      *deadline,
		ExpectedTools: append([]string(nil), expectedTools...),
		ExpectedText:  append([]string(nil), expectedText...),
		Environ:       os.Environ(),
		Producer:      infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit},
	})
	if summary.SchemaVersion != 0 {
		fmt.Fprint(os.Stdout, infra.RenderModelCheckSummary(summary))
	}
	return err
}

func runAttachments(args []string) error {
	if callerCWD := os.Getenv(callerCWDEnv); callerCWD != "" {
		originalCWD, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("read current directory: %w", err)
		}
		if err := os.Chdir(callerCWD); err != nil {
			return fmt.Errorf("restore caller cwd for attachments: %w", err)
		}
		defer func() {
			_ = os.Chdir(originalCWD)
		}()
	}
	return attachments.Run(args, attachments.Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    os.Getenv,
	})
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
	fs.Func("claude-yolo-mode", "persistent Claude yolo mode for this project: true or false", func(value string) error {
		var parsed bool
		switch value {
		case "true":
			parsed = true
		case "false":
			parsed = false
		default:
			return fmt.Errorf("expected true or false")
		}
		claudePrimarySessionSetup.YoloMode = &parsed
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

	layout, err := resolveLayout(mode, *homeDir, *projectDir, positionals)
	if err != nil {
		return err
	}
	layout.SourceDir, err = resolveSetupSourceDir(layout, *sourceDir)
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

// runVerify re-checks the postcondition Setup enforces, so a caller that only
// sees an exit code and a directory can still tell an installed runtime apart
// from a directory that merely looks like one.
func runVerify(args []string) error {
	if len(args) == 0 {
		return errors.New("verify requires mode: global or local")
	}
	mode := args[0]
	fs := flag.NewFlagSet("verify "+mode, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	homeDir := fs.String("home-dir", "", "home directory for global verify")
	projectDir := fs.String("project-dir", "", "project directory for local verify")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	layout, err := resolveLayout(mode, *homeDir, *projectDir, fs.Args())
	if err != nil {
		return err
	}
	if err := infra.VerifyInstalledRuntime(layout); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "verified %s agent runtime: %s\n", layout.Mode, layout.AgentsDir)
	return nil
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
	layout, err := resolveLayout(mode, *homeDir, *projectDir, fs.Args())
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
			fmt.Fprintf(os.Stdout, "claude_primary_yolo_mode: %t\n", report.ClaudePrimarySession.YoloMode.Value)
			fmt.Fprintf(os.Stdout, "claude_primary_yolo_mode_source: %s\n", claudePrimaryBoolSource(report.ClaudePrimarySession.YoloMode))
		}
		for _, target := range report.CanonicalTargets {
			prefix := "canonical_" + strings.ReplaceAll(target.Entrypoint, "-", "_")
			fmt.Fprintf(os.Stdout, "%s_target: %s\n", prefix, target.Name)
			fmt.Fprintf(os.Stdout, "%s_entrypoint_source: %s\n", prefix, target.EntrypointSource)
			fmt.Fprintf(os.Stdout, "%s_target_source: %s\n", prefix, target.TargetSource)
			fmt.Fprintf(os.Stdout, "%s_vendor: %s\n", prefix, target.Vendor)
			fmt.Fprintf(os.Stdout, "%s_environment: %s\n", prefix, target.Environment)
			fmt.Fprintf(os.Stdout, "%s_model: %s\n", prefix, target.Model)
			fmt.Fprintf(os.Stdout, "%s_reasoning: %s\n", prefix, target.Reasoning)
			fmt.Fprintf(os.Stdout, "%s_profile: %s\n", prefix, target.Profile)
			fmt.Fprintf(os.Stdout, "%s_profile_source: %s\n", prefix, target.ProfileSource)
			fmt.Fprintf(os.Stdout, "%s_profile_provider: %s\n", prefix, target.ProfileProvider)
			fmt.Fprintf(os.Stdout, "%s_profile_provider_source: %s\n", prefix, target.ProfileSource)
			fmt.Fprintf(os.Stdout, "%s_endpoint: %s\n", prefix, target.Endpoint)
			fmt.Fprintf(os.Stdout, "%s_endpoint_source: %s\n", prefix, target.ProfileSource)
		}
		if report.CodexConfigShadowsGlobal {
			if report.CodexConfigGenerated {
				fmt.Fprintf(os.Stdout, "codex_config_action: managed project-local .codex/config.toml is active; rendered from the installed Codex config without user-level profiles; use --codex-config=global to remove it if unintended\n")
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

func claudePrimaryBoolSource(value infra.ClaudePrimarySessionBoolValue) string {
	if value.Present {
		return value.Source
	}
	return "default"
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
	if _, err := infra.PreparePrimarySession(
		"codex",
		plan.StartDir,
		infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit},
	); err != nil {
		return fmt.Errorf("prepare Codex project surface: %w", err)
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
	if _, err := infra.PreparePrimarySession(
		"claude",
		plan.StartDir,
		infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit},
	); err != nil {
		return fmt.Errorf("prepare Claude project surface: %w", err)
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

func runPi(args []string) error {
	startDir := os.Getenv(callerCWDEnv)
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	printConfig := false
	filtered := make([]string, 0, len(args))
	beforeDelimiter := true
	for _, token := range args {
		if beforeDelimiter && token == "--" {
			beforeDelimiter = false
			filtered = append(filtered, token)
			continue
		}
		if beforeDelimiter && token == "--print-config" {
			printConfig = true
			continue
		}
		filtered = append(filtered, token)
	}
	if printConfig {
		plan, err := infra.BuildPrimarySessionLaunchPlan("pi", startDir, "", filtered, infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit}, nil)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(plan)
	}
	return infra.RunPi(infra.RunPiOptions{ProjectDir: startDir, Args: filtered, Environ: os.Environ(), Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr})
}

func runTarget(args []string) error {
	if len(args) == 0 {
		return errors.New("target requires an entrypoint name")
	}
	entrypoint := args[0]
	fs := flag.NewFlagSet("target "+entrypoint, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	printConfig := fs.Bool("print-config", false, "resolve and print the canonical target without launching")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	startDir := os.Getenv(callerCWDEnv)
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve target caller cwd: %w", err)
		}
	}
	producer := infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit}
	plan, err := infra.BuildCanonicalTargetLaunchPlan(entrypoint, startDir, "", fs.Args(), producer, nil)
	if err != nil {
		return err
	}
	if *printConfig {
		fmt.Fprint(os.Stdout, infra.RenderCanonicalTargetLaunchPlan(plan))
		return nil
	}
	if plan.Provider == "pi" {
		return infra.RunPi(infra.RunPiOptions{
			ProjectDir: plan.ProjectDir,
			Args:       plan.TargetProviderArgs(),
			Environ:    os.Environ(),
			Stdin:      os.Stdin,
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
		})
	}
	if _, err := infra.PreparePrimarySession(plan.Provider, plan.ProjectDir, producer); err != nil {
		return fmt.Errorf("prepare canonical %s target project surface: %w", plan.Provider, err)
	}
	cmd := exec.Command(plan.Executable, plan.LaunchVariants.Interactive.Argv...)
	cmd.Dir = plan.ProjectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCompose(args []string) error {
	fs := flag.NewFlagSet("compose", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	mode := fs.String("mode", "child", "composition mode: child or primary-session")
	agent := fs.String("agent", "", "agent provider: codex, claude, or pi")
	entrypoint := fs.String("entrypoint", "", "canonical vendor entrypoint")
	projectDir := fs.String("project", "", "project directory used for composition")
	schemaVersion := fs.Int("schema-version", 0, "composition contract schema version")
	jsonOutput := fs.Bool("json", false, "emit one JSON contract document")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*jsonOutput {
		return fmt.Errorf("compose requires --json")
	}
	switch *mode {
	case "child":
		if *entrypoint != "" {
			return fmt.Errorf("child compose does not accept --entrypoint")
		}
		if *agent != "codex" && *agent != "claude" {
			return fmt.Errorf("child compose requires --agent codex or claude")
		}
	case "primary-session":
		if (*agent == "") == (*entrypoint == "") {
			return fmt.Errorf("primary-session compose requires exactly one of --agent or --entrypoint")
		}
		if *entrypoint != "" {
			return runComposeCanonicalTarget(*entrypoint, *projectDir, *schemaVersion, fs.Args())
		}
		if *agent != "codex" && *agent != "claude" && *agent != "pi" {
			return fmt.Errorf("primary-session compose requires --agent codex, claude, or pi")
		}
		return runComposePrimarySession(*agent, *projectDir, *schemaVersion, fs.Args())
	default:
		return fmt.Errorf("compose requires --mode child or --mode primary-session")
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("compose does not accept positional arguments: %q", fs.Args())
	}
	canonicalProjectDir, err := infra.CanonicalProjectDir(*projectDir)
	if err != nil {
		return err
	}
	producer := infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit}
	if *schemaVersion != infra.ChildLaunchCompositionSchemaVersion {
		envelope := infra.NewChildLaunchCompositionErrorEnvelope(*agent, canonicalProjectDir, producer, "unsupported_schema_version")
		if err := json.NewEncoder(os.Stdout).Encode(envelope); err != nil {
			return fmt.Errorf("encode compose error envelope: %w", err)
		}
		return fmt.Errorf("unsupported child launch composition schema version %d", *schemaVersion)
	}

	composition, err := infra.BuildChildLaunchComposition(*agent, canonicalProjectDir, "", producer)
	if err != nil {
		envelope := infra.NewChildLaunchCompositionErrorEnvelope(*agent, canonicalProjectDir, producer, "invalid_project_configuration")
		if encodeErr := json.NewEncoder(os.Stdout).Encode(envelope); encodeErr != nil {
			return fmt.Errorf("encode compose error envelope: %w", encodeErr)
		}
		return fmt.Errorf("compose project MCP configuration: %w", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(composition); err != nil {
		return fmt.Errorf("encode child launch composition: %w", err)
	}
	return nil
}

func runComposeCanonicalTarget(entrypoint, projectDir string, schemaVersion int, userArgs []string) error {
	producer := infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit}
	canonicalProjectDir, err := infra.CanonicalProjectDir(projectDir)
	if err != nil {
		return err
	}
	provider := infra.CanonicalProviderForEntrypoint(entrypoint)
	if schemaVersion != infra.PrimarySessionLaunchPlanSchemaVersion {
		envelope := infra.NewPrimarySessionLaunchPlanErrorEnvelope(provider, canonicalProjectDir, producer, infra.PrimarySessionErrorUnsupportedSchemaVersion)
		if err := json.NewEncoder(os.Stdout).Encode(envelope); err != nil {
			return fmt.Errorf("encode canonical target error envelope: %w", err)
		}
		return fmt.Errorf("unsupported primary-session launch plan schema version %d", schemaVersion)
	}
	plan, err := infra.BuildCanonicalTargetLaunchPlan(entrypoint, canonicalProjectDir, "", userArgs, producer, nil)
	if err != nil {
		var targetErr *infra.CanonicalTargetError
		var envelope infra.PrimarySessionLaunchPlanErrorEnvelope
		if errors.As(err, &targetErr) {
			envelope = infra.NewCanonicalTargetLaunchPlanErrorEnvelope(provider, canonicalProjectDir, producer, targetErr)
		} else {
			code := infra.PrimarySessionErrorInvalidProjectConfiguration
			var composeErr *infra.PrimarySessionComposeError
			if errors.As(err, &composeErr) {
				code = composeErr.Code
			}
			envelope = infra.NewPrimarySessionLaunchPlanErrorEnvelope(provider, canonicalProjectDir, producer, code)
		}
		if encodeErr := json.NewEncoder(os.Stdout).Encode(envelope); encodeErr != nil {
			return fmt.Errorf("encode canonical target error envelope: %w", encodeErr)
		}
		return fmt.Errorf("compose canonical target launch plan: %w", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(plan); err != nil {
		return fmt.Errorf("encode canonical target launch plan: %w", err)
	}
	return nil
}

func runPrepare(args []string) error {
	fs := flag.NewFlagSet("prepare", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agent := fs.String("agent", "", "agent provider: codex or claude")
	projectDir := fs.String("project", "", "project directory to prepare")
	schemaVersion := fs.Int("schema-version", 0, "preparation contract schema version")
	jsonOutput := fs.Bool("json", false, "emit one JSON contract document")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *agent != "codex" && *agent != "claude" {
		return fmt.Errorf("prepare requires --agent codex or --agent claude")
	}
	if !*jsonOutput {
		return fmt.Errorf("prepare requires --json")
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("prepare does not accept positional arguments: %q", fs.Args())
	}
	canonicalProjectDir, err := infra.CanonicalProjectDir(*projectDir)
	if err != nil {
		return err
	}
	producer := infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit}
	if *schemaVersion != infra.PrimarySessionPreparationSchemaVersion {
		envelope := infra.NewPrimarySessionPreparationErrorEnvelope(
			*agent,
			canonicalProjectDir,
			producer,
			infra.PrimarySessionPreparationErrorUnsupportedSchemaVersion,
		)
		if err := json.NewEncoder(os.Stdout).Encode(envelope); err != nil {
			return fmt.Errorf("encode prepare error envelope: %w", err)
		}
		return fmt.Errorf("unsupported primary-session preparation schema version %d", *schemaVersion)
	}
	report, err := infra.PreparePrimarySession(*agent, canonicalProjectDir, producer)
	if err != nil {
		envelope := infra.NewPrimarySessionPreparationErrorEnvelope(
			*agent,
			canonicalProjectDir,
			producer,
			infra.PrimarySessionPreparationErrorRenderFailed,
		)
		if encodeErr := json.NewEncoder(os.Stdout).Encode(envelope); encodeErr != nil {
			return fmt.Errorf("encode prepare error envelope: %w", encodeErr)
		}
		return fmt.Errorf("prepare primary-session project surface: %w", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		return fmt.Errorf("encode primary-session preparation report: %w", err)
	}
	return nil
}

// runComposePrimarySession emits the non-launching primary-session launch
// plan. Provider user args are everything after the compose flags (typically
// separated with `--`) and pass through the same wrapper parsing as
// `agents-infra codex|claude`.
func runComposePrimarySession(provider, projectDir string, schemaVersion int, userArgs []string) error {
	producer := infra.ChildLaunchCompositionProducer{Version: Version, Commit: Commit}
	canonicalProjectDir, err := infra.CanonicalProjectDir(projectDir)
	if err != nil {
		return err
	}
	if schemaVersion != infra.PrimarySessionLaunchPlanSchemaVersion {
		envelope := infra.NewPrimarySessionLaunchPlanErrorEnvelope(provider, canonicalProjectDir, producer, infra.PrimarySessionErrorUnsupportedSchemaVersion)
		if err := json.NewEncoder(os.Stdout).Encode(envelope); err != nil {
			return fmt.Errorf("encode primary-session error envelope: %w", err)
		}
		return fmt.Errorf("unsupported primary-session launch plan schema version %d", schemaVersion)
	}
	plan, err := infra.BuildPrimarySessionLaunchPlan(provider, canonicalProjectDir, "", userArgs, producer, nil)
	if err != nil {
		code := infra.PrimarySessionErrorInvalidProjectConfiguration
		var composeErr *infra.PrimarySessionComposeError
		if errors.As(err, &composeErr) {
			code = composeErr.Code
		}
		envelope := infra.NewPrimarySessionLaunchPlanErrorEnvelope(provider, canonicalProjectDir, producer, code)
		if encodeErr := json.NewEncoder(os.Stdout).Encode(envelope); encodeErr != nil {
			return fmt.Errorf("encode primary-session error envelope: %w", encodeErr)
		}
		return fmt.Errorf("compose primary-session launch plan: %w", err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(plan); err != nil {
		return fmt.Errorf("encode primary-session launch plan: %w", err)
	}
	return nil
}

func runVersion() error {
	fmt.Fprintf(os.Stdout, "agents-infra %s commit=%s build_date=%s\n", Version, Commit, BuildDate)
	return nil
}

// resolveSetupSourceDir resolves the source tree a setup run copies from. The
// installed binary carries no source path of its own, so an explicit flag or
// environment override is followed by the installer's recorded repo path and
// the installed .agents runtime.
func resolveSetupSourceDir(layout infra.Layout, sourceDirFlag string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	return infra.ResolveSourceDir(infra.SourceDirRequest{
		Mode:              layout.Mode,
		Flag:              sourceDirFlag,
		Env:               os.Getenv(infra.SourceDirEnv),
		ConfigDirOverride: os.Getenv(infra.ConfigDirEnv),
		XDGConfigHome:     os.Getenv("XDG_CONFIG_HOME"),
		GOOS:              runtime.GOOS,
		HomeDir:           homeDir,
		TargetAgentsDir:   layout.AgentsDir,
	})
}

func resolveLayout(mode, homeDir, projectDir string, positional []string) (infra.Layout, error) {
	switch mode {
	case string(infra.ModeGlobal):
		if homeDir == "" {
			var err error
			homeDir, err = os.UserHomeDir()
			if err != nil {
				return infra.Layout{}, fmt.Errorf("resolve home dir: %w", err)
			}
		}
		return infra.GlobalLayout("", homeDir)
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
		return infra.LocalLayout("", projectDir)
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
  agents-infra setup local [PROJECT_DIR] [--source-dir DIR] [--project-dir DIR] [--no-sync] [--codex-config preserve|global|local] [--codex-primary-model MODEL] [--codex-primary-reasoning-effort EFFORT] [--codex-yolo-mode=true|false] [--clear-codex-primary-session] [--claude-primary-model MODEL] [--claude-yolo-mode=true|false] [--clear-claude-primary-session]
  agents-infra refresh-links --agents-dir DIR --claude-dir DIR --codex-dir DIR --bin-dir DIR [--mode global|local] [--codex-config preserve|global|local]
  agents-infra doctor global [--home-dir DIR]
  agents-infra doctor local [PROJECT_DIR] [--project-dir DIR]
  agents-infra verify global [--home-dir DIR]
  agents-infra verify local [PROJECT_DIR] [--project-dir DIR]
  agents-infra compose --agent codex|claude --project DIR --schema-version 1 --json
  agents-infra compose --mode primary-session --agent codex|claude|pi --project DIR --schema-version 1 --json [-- PROVIDER_ARGS...]
  agents-infra compose --mode primary-session --entrypoint openai-infra|anthropic-infra|qwen-infra --project DIR --schema-version 1 --json [-- PROVIDER_ARGS...]
  agents-infra prepare --agent codex|claude --project DIR --schema-version 1 --json
  agents-infra attachments list|show|path|materialize|stage-images [...]
  agents-infra codex [--print-config] [-d|--danger|--yolo] [--] [CODEX_ARGS...]
  agents-infra claude [--print-config] [-d|--danger|--yolo] [--] [CLAUDE_ARGS...]
  agents-infra pi [--print-config] [--profile NAME] [PI_ARGS...] [-- MESSAGE...]
  agents-infra target ENTRYPOINT [--print-config] [-- PROVIDER_ARGS...]
  agents-infra model-check --target ENTRYPOINT --prompt TEXT --output-dir DIR [--deadline DURATION] [--expect-tool NAME] [--expect-text TEXT]

Source tree resolution for setup (first usable wins):
  1. --source-dir DIR
  2. ` + infra.SourceDirEnv + `
  3. repoPath from the installer's machine-scoped install.json
  4. the installed ~/.agents runtime
An explicit --source-dir or ` + infra.SourceDirEnv + ` is never replaced by a
discovered fallback; an unusable one fails and names what is missing.`
}
