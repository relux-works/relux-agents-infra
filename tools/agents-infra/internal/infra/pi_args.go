package infra

import (
	"errors"
	"fmt"
	"strings"
)

type PiArgumentPlan struct {
	ProfileOverride *string
	Argv            []string
	DiagnosticArgv  []string
}

// applyPiPrimarySessionYolo turns the shared primary-session yolo policy into
// Pi's one-run project trust override. Pi does not prompt for individual tool
// calls; --approve only suppresses the project-local resource trust prompt so
// AGENTS.md, skills, extensions, and other reviewed project inputs can load.
func applyPiPrimarySessionYolo(args []string, policy PiPrimarySessionPolicy) ([]string, error) {
	result := append([]string(nil), args...)
	if !policy.YoloMode.Present || !policy.YoloMode.Value {
		return result, nil
	}
	approved := false
	for _, token := range result {
		if token == "--" {
			break
		}
		switch token {
		case "--approve", "-a":
			approved = true
		case "--no-approve", "-na":
			return nil, piError("invalid_provider_arguments", fmt.Errorf("%s.yolo_mode=true from %s conflicts with %s", piPrimarySessionField, policy.YoloMode.Source, token))
		}
	}
	if approved {
		return result, nil
	}
	return append([]string{"--approve"}, result...), nil
}

func ExtractPiProfileOverride(args []string) (*string, error) {
	var selected *string
	set := func(value string) error {
		if value == "" {
			return piError("invalid_provider_arguments", errors.New("--profile requires a non-empty value"))
		}
		if selected != nil && *selected != value {
			return piError("invalid_provider_arguments", errors.New("conflicting --profile selections"))
		}
		clone := value
		selected = &clone
		return nil
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		if strings.HasPrefix(args[i], "--profile=") {
			if err := set(strings.TrimPrefix(args[i], "--profile=")); err != nil {
				return nil, err
			}
			continue
		}
		if args[i] == "--profile" {
			if i+1 >= len(args) || args[i+1] == "--" {
				return nil, piError("invalid_provider_arguments", errors.New("--profile requires a value before the operand delimiter"))
			}
			i++
			if err := set(args[i]); err != nil {
				return nil, err
			}
		}
	}
	return selected, nil
}

var piKnownValueOptions = map[string]bool{
	"--provider": true, "--model": true, "--api-key": true, "--thinking": true,
	"--system-prompt": true, "--append-system-prompt": true, "--name": true,
	"--session": true, "--session-id": true, "--fork": true, "--session-dir": true,
	"--models": true, "--tools": true, "--exclude-tools": true, "--extension": true,
	"--skill": true, "--prompt-template": true, "--theme": true, "--use-theme": true,
	"--mode": true, "--export": true, "--tui-mode": true,
}

var piKnownBooleanOptions = map[string]bool{
	"--approve": true, "--no-approve": true, "--continue": true, "--no-tools": true,
	"--resume": true, "--no-session": true, "--no-builtin-tools": true, "--no-extensions": true,
	"--no-skills": true, "--no-prompt-templates": true, "--no-themes": true,
	"--no-context-files": true, "--verbose": true, "--offline": true, "--help": true, "--version": true,
}

var piShortValueOptions = map[string]bool{"-n": true, "-t": true, "-xt": true, "-e": true}
var piShortBooleanOptions = map[string]bool{"-h": true, "-v": true, "-c": true, "-r": true, "-nt": true, "-nbt": true, "-ne": true, "-ns": true, "-np": true, "-nc": true, "-a": true, "-na": true}
var piThinkingLevels = map[string]bool{"off": true, "minimal": true, "low": true, "medium": true, "high": true, "xhigh": true, "max": true}

func BuildManagedPiArguments(args []string, configuredProfile string, profile PiProfile) (PiArgumentPlan, error) {
	delimiter := -1
	for i, token := range args {
		if token == "--" {
			if delimiter >= 0 {
				return PiArgumentPlan{}, piError("unsafe_pi_operand", errors.New("repeated wrapper operand delimiter"))
			}
			delimiter = i
		}
	}
	prefix, suffix := args, []string{}
	if delimiter >= 0 {
		prefix, suffix = args[:delimiter], args[delimiter+1:]
	}
	for _, token := range suffix {
		if token == "" || strings.HasPrefix(token, "-") || strings.HasPrefix(token, "@") {
			return PiArgumentPlan{}, piError("unsafe_pi_operand", fmt.Errorf("unsafe Pi message operand %q", token))
		}
	}

	var profileSelection, providerSelection, modelSelection, thinkingSelection *string
	forward := []string{}
	diagnostic := []string{}
	set := func(dst **string, value, name string) error {
		if value == "" {
			return piError("invalid_provider_arguments", fmt.Errorf("%s requires a non-empty value", name))
		}
		if *dst != nil && **dst != value {
			return piError("invalid_provider_arguments", fmt.Errorf("conflicting %s selections", name))
		}
		clone := value
		*dst = &clone
		return nil
	}
	for i := 0; i < len(prefix); i++ {
		token := prefix[i]
		if token == "" {
			return PiArgumentPlan{}, piError("invalid_provider_arguments", errors.New("empty Pi argument"))
		}
		name, equalValue, hasEqual := strings.Cut(token, "=")
		isWrapper := name == "--profile" || name == "--provider" || name == "--model" || name == "--thinking" || name == "--api-key"
		if hasEqual && isWrapper {
			if equalValue == "" {
				return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("%s requires a non-empty value", name))
			}
			var target **string
			switch name {
			case "--profile":
				target = &profileSelection
			case "--provider":
				target = &providerSelection
			case "--model":
				target = &modelSelection
			case "--thinking":
				target = &thinkingSelection
			}
			if name != "--api-key" {
				if err := set(target, equalValue, name); err != nil {
					return PiArgumentPlan{}, err
				}
			}
			if name != "--profile" {
				forward = append(forward, name, equalValue)
				diagnostic = append(diagnostic, name, equalValue)
				if name == "--api-key" {
					diagnostic[len(diagnostic)-1] = "<redacted>"
				}
			}
			continue
		}
		if hasEqual && (piKnownValueOptions[name] || piKnownBooleanOptions[name] || name == "--print" || name == "--list-models") {
			return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("known Pi option %s does not accept equals form", name))
		}
		if token == "--profile" || token == "--provider" || token == "--model" || token == "--thinking" || token == "--api-key" {
			if i+1 >= len(prefix) {
				return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("%s requires a value before the operand delimiter", token))
			}
			i++
			value := prefix[i]
			if value == "" {
				return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("%s requires a non-empty value", token))
			}
			if token != "--api-key" {
				var target **string
				switch token {
				case "--profile":
					target = &profileSelection
				case "--provider":
					target = &providerSelection
				case "--model":
					target = &modelSelection
				case "--thinking":
					target = &thinkingSelection
				}
				if err := set(target, value, token); err != nil {
					return PiArgumentPlan{}, err
				}
			}
			if token != "--profile" {
				forward = append(forward, token, value)
				diagnostic = append(diagnostic, token, value)
				if token == "--api-key" {
					diagnostic[len(diagnostic)-1] = "<redacted>"
				}
			}
			continue
		}
		if piKnownValueOptions[token] {
			if i+1 >= len(prefix) {
				return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("%s requires a value before the operand delimiter", token))
			}
			value := prefix[i+1]
			if err := validateManagedPiStateArgument(token, value); err != nil {
				return PiArgumentPlan{}, err
			}
			forward = append(forward, token, value)
			diagnostic = append(diagnostic, token, value)
			i++
			continue
		}
		if piKnownBooleanOptions[token] {
			forward = append(forward, token)
			diagnostic = append(diagnostic, token)
			continue
		}
		if token == "--print" || token == "-p" {
			forward = append(forward, token)
			diagnostic = append(diagnostic, token)
			if i+1 < len(prefix) && !strings.HasPrefix(prefix[i+1], "@") && (!strings.HasPrefix(prefix[i+1], "-") || strings.HasPrefix(prefix[i+1], "---")) {
				forward = append(forward, prefix[i+1])
				diagnostic = append(diagnostic, prefix[i+1])
				i++
			}
			continue
		}
		if token == "--list-models" {
			forward = append(forward, token)
			diagnostic = append(diagnostic, token)
			if i+1 < len(prefix) && !strings.HasPrefix(prefix[i+1], "-") && !strings.HasPrefix(prefix[i+1], "@") {
				forward = append(forward, prefix[i+1])
				diagnostic = append(diagnostic, prefix[i+1])
				i++
			}
			continue
		}
		if piShortValueOptions[token] {
			if i+1 >= len(prefix) {
				return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("%s requires a value before the operand delimiter", token))
			}
			forward = append(forward, token, prefix[i+1])
			diagnostic = append(diagnostic, token, prefix[i+1])
			i++
			continue
		}
		if piShortBooleanOptions[token] {
			forward = append(forward, token)
			diagnostic = append(diagnostic, token)
			continue
		}
		if strings.HasPrefix(token, "--") {
			if hasEqual {
				forward = append(forward, token)
				diagnostic = append(diagnostic, token)
				continue
			}
			if i+1 >= len(prefix) || strings.HasPrefix(prefix[i+1], "-") || strings.HasPrefix(prefix[i+1], "@") {
				return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("unknown Pi option %q must use --name=value or a complete flag/value pair", token))
			}
			forward = append(forward, token, prefix[i+1])
			diagnostic = append(diagnostic, token, prefix[i+1])
			i++
			continue
		}
		if strings.HasPrefix(token, "-") {
			return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("unknown Pi short option %q", token))
		}
		return PiArgumentPlan{}, piError("unsafe_pi_operand", fmt.Errorf("managed Pi operands require the wrapper -- delimiter: %q", token))
	}
	if profileSelection != nil && *profileSelection != configuredProfile { /* CLI override is resolved by caller before this validation. */
	}
	if providerSelection != nil && *providerSelection != profile.Provider {
		return PiArgumentPlan{}, piError("managed_profile_identity_mismatch", fmt.Errorf("provider %q does not equal managed provider", *providerSelection))
	}
	effectiveThinking := profile.Thinking
	modelThinking, err := validateManagedModelSelection(modelSelection, profile)
	if err != nil {
		return PiArgumentPlan{}, err
	}
	if thinkingSelection != nil {
		if !piThinkingLevels[*thinkingSelection] {
			return PiArgumentPlan{}, piError("invalid_provider_arguments", fmt.Errorf("invalid thinking level %q", *thinkingSelection))
		}
		effectiveThinking = *thinkingSelection
	}
	if modelThinking != "" && thinkingSelection != nil && modelThinking != *thinkingSelection {
		return PiArgumentPlan{}, piError("invalid_provider_arguments", errors.New("model thinking suffix conflicts with --thinking"))
	}
	if modelThinking != "" {
		effectiveThinking = modelThinking
	}

	// Remove explicit identity selections and emit exactly one canonical managed
	// provider/model/thinking triple before every other option and operand.
	forward, diagnostic = stripPiIdentityOptions(forward, diagnostic)
	canonical := []string{"--provider", profile.Provider, "--model", profile.Model, "--thinking", effectiveThinking}
	argv := append(append(canonical, forward...), suffix...)
	diag := append(append([]string(nil), canonical...), diagnostic...)
	diag = append(diag, suffix...)
	return PiArgumentPlan{ProfileOverride: profileSelection, Argv: argv, DiagnosticArgv: diag}, nil
}

func validateManagedPiStateArgument(option, value string) error {
	switch option {
	case "--export":
		// Pinned Pi reads this path before it initializes the isolated agent and
		// session directories. Managed export would therefore be a direct read
		// outside the hash-contained session boundary.
		return piError("invalid_provider_arguments", errors.New("--export cannot read outside managed Pi session isolation"))
	case "--session-dir":
		return piError("invalid_provider_arguments", errors.New("--session-dir cannot override managed Pi session isolation"))
	case "--session", "--fork":
		// Pinned Pi treats any selector containing a slash or backslash, or
		// ending in .jsonl, as a filesystem path. Managed profiles permit only
		// ID lookup inside PI_CODING_AGENT_SESSION_DIR so arbitrary session
		// files cannot bypass the hash-contained state boundary.
		if strings.ContainsAny(value, `/\`) || strings.HasSuffix(value, ".jsonl") {
			return piError("invalid_provider_arguments", fmt.Errorf("%s filesystem paths cannot bypass managed Pi session isolation", option))
		}
	default:
		// The remaining pinned value options select model/UI/resource behavior;
		// none has pinned direct agent/session state semantics.
		return nil
	}
	return nil
}

func validateManagedModelSelection(selection *string, profile PiProfile) (string, error) {
	if selection == nil {
		return "", nil
	}
	value := *selection
	bases := []string{profile.Model, profile.Provider + "/" + profile.Model}
	for _, base := range bases {
		if value == base {
			return "", nil
		}
		if strings.HasPrefix(value, base+":") {
			suffix := strings.TrimPrefix(value, base+":")
			if piThinkingLevels[suffix] {
				return suffix, nil
			}
		}
	}
	return "", piError("managed_profile_identity_mismatch", fmt.Errorf("model %q does not equal managed provider/model identity", value))
}

func stripPiIdentityOptions(argv, diagnostic []string) ([]string, []string) {
	strip := func(in []string) []string {
		out := []string{}
		for i := 0; i < len(in); i++ {
			if (in[i] == "--provider" || in[i] == "--model" || in[i] == "--thinking") && i+1 < len(in) {
				i++
				continue
			}
			out = append(out, in[i])
		}
		return out
	}
	return strip(argv), strip(diagnostic)
}
