package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/modelharness"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "model-harness:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: model-harness run|render|doctor|version")
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Printf("model-harness %s (%s, %s)\n", Version, Commit, BuildDate)
		return nil
	case "run", "render", "doctor":
		return runProfileCommand(args[0], args[1:])
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runProfileCommand(action string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s requires a profile", action)
	}
	profile := args[0]
	flags := flag.NewFlagSet(action, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	config := flags.String("config", "", "absolute model-harness config path")
	host := flags.String("host", "127.0.0.1", "local endpoint host")
	port := flags.Int("port", 0, "local endpoint port")
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *port == 0 {
		return errors.New("--port is required")
	}
	plan, err := modelharness.Resolve(*config, profile, *host, *port)
	if err != nil {
		return err
	}
	switch action {
	case "run":
		if *jsonOutput {
			return errors.New("run does not accept --json")
		}
		return modelharness.Run(plan)
	case "render":
		if !*jsonOutput {
			return errors.New("render requires --json")
		}
		return modelharness.EncodePlan(os.Stdout, plan)
	case "doctor":
		if *jsonOutput {
			return errors.New("doctor does not accept --json")
		}
		return modelharness.Doctor(plan, os.Stdout, os.Stderr)
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}

func printUsage() {
	fmt.Println(`Usage:
  model-harness run PROFILE --host 127.0.0.1 --port PORT [--config PATH]
  model-harness render PROFILE --host 127.0.0.1 --port PORT --json [--config PATH]
  model-harness doctor PROFILE --host 127.0.0.1 --port PORT [--config PATH]
  model-harness version`)
}
