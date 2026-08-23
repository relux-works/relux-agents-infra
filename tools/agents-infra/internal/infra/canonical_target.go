package infra

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	PrimarySessionErrorUnknownEntrypoint      = "unknown_entrypoint"
	PrimarySessionErrorUnknownTarget          = "unknown_target"
	PrimarySessionErrorInvalidTarget          = "invalid_target"
	PrimarySessionErrorTargetIdentityConflict = "target_identity_conflict"
)

type TargetErrorContext struct {
	Entrypoint string `json:"entrypoint,omitempty"`
	Target     string `json:"target,omitempty"`
	Profile    string `json:"profile,omitempty"`
	Field      string `json:"field,omitempty"`
	Source     string `json:"source,omitempty"`
}

// CanonicalTargetError is safe to render on both human and machine startup
// surfaces. Context contains identities and field/source provenance only.
type CanonicalTargetError struct {
	Code        string
	Context     TargetErrorContext
	Remediation string
	Err         error
}

func (e *CanonicalTargetError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %v", e.Code, e.Err)
	var context []string
	if e.Context.Entrypoint != "" {
		context = append(context, "entrypoint="+strconv.Quote(e.Context.Entrypoint))
	}
	if e.Context.Target != "" {
		context = append(context, "target="+strconv.Quote(e.Context.Target))
	}
	if e.Context.Profile != "" {
		context = append(context, "profile="+strconv.Quote(e.Context.Profile))
	}
	if e.Context.Field != "" {
		context = append(context, "field="+strconv.Quote(e.Context.Field))
	}
	if e.Context.Source != "" {
		context = append(context, "source="+strconv.Quote(e.Context.Source))
	}
	if len(context) > 0 {
		b.WriteString(" (")
		b.WriteString(strings.Join(context, ", "))
		b.WriteString(")")
	}
	if e.Remediation != "" {
		b.WriteString(". Remediation: ")
		b.WriteString(e.Remediation)
	}
	return b.String()
}

func (e *CanonicalTargetError) Unwrap() error { return e.Err }

type ResolvedCanonicalTarget struct {
	Entrypoint        ProjectEntrypoint
	Target            ProjectTarget
	Profile           *PiProfile
	EffectiveProvider string
	EffectiveEndpoint string
}

func canonicalProviderForEnvironment(environment string) string {
	switch environment {
	case "codex":
		return "codex"
	case "claude-code":
		return "claude"
	case "pi":
		return "pi"
	default:
		return ""
	}
}

func CanonicalProviderForEntrypoint(entrypoint string) string {
	switch entrypoint {
	case "openai-infra":
		return "codex"
	case "anthropic-infra":
		return "claude"
	case "qwen-infra":
		return "pi"
	default:
		return ""
	}
}

func wrapCanonicalConfigurationError(err error) error {
	var configErr *ProjectConfigurationError
	if !errors.As(err, &configErr) {
		return err
	}
	return &CanonicalTargetError{
		Code: PrimarySessionErrorInvalidProjectConfiguration,
		Context: TargetErrorContext{
			Field:  configErr.Field,
			Source: configErr.Source,
		},
		Remediation: "correct the named project-config.toml field; agents-infra will not rewrite configuration automatically",
		Err:         configErr,
	}
}

func loadCanonicalTargetConfig(projectDir, homeDir string) (compositeProjectConfig, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return compositeProjectConfig{}, err
		}
	}
	homeDir, err := filepath.Abs(homeDir)
	if err != nil {
		return compositeProjectConfig{}, err
	}
	composite, err := loadCompositeProjectConfig(
		ancestorDirsRootFirst(projectDir),
		filepath.Join(homeDir, ".agents", ".configs", projectConfigFileName),
	)
	if err != nil {
		return compositeProjectConfig{}, wrapCanonicalConfigurationError(err)
	}
	if err := validateComposedCanonicalTargets(composite); err != nil {
		return compositeProjectConfig{}, err
	}
	return composite, nil
}

// ValidateCanonicalProjectConfiguration is used by setup/verify before they
// attest an installed alias surface. Legacy-only projects remain valid.
func ValidateCanonicalProjectConfiguration(projectDir, homeDir string) error {
	canonical, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return err
	}
	_, err = loadCanonicalTargetConfig(canonical, homeDir)
	return err
}

func validateComposedCanonicalTargets(composite compositeProjectConfig) error {
	var targetNames []string
	for name := range composite.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)
	for _, name := range targetNames {
		target := composite.Targets[name]
		if target.Environment != "pi" {
			continue
		}
		if err := validateResolvedPiTarget(target, composite.PiProfiles); err != nil {
			return err
		}
	}
	var entrypointNames []string
	for entrypoint := range composite.Entrypoints {
		entrypointNames = append(entrypointNames, entrypoint)
	}
	sort.Strings(entrypointNames)
	for _, entrypoint := range entrypointNames {
		mapping := composite.Entrypoints[entrypoint]
		target, ok := composite.Targets[mapping.TargetName]
		if !ok {
			return &CanonicalTargetError{
				Code:        PrimarySessionErrorUnknownTarget,
				Context:     TargetErrorContext{Entrypoint: entrypoint, Target: mapping.TargetName, Field: entrypointsField + "." + entrypoint, Source: mapping.Source},
				Remediation: "define the exact target name in [agents.targets] or correct this entrypoint mapping",
				Err:         fmt.Errorf("entrypoint references an unknown canonical target"),
			}
		}
		if target.Vendor != canonicalEntrypointVendors[entrypoint] {
			return &CanonicalTargetError{
				Code:        PrimarySessionErrorInvalidTarget,
				Context:     TargetErrorContext{Entrypoint: entrypoint, Target: target.Name, Field: entrypointsField + "." + entrypoint, Source: mapping.Source},
				Remediation: "map the entrypoint to a target with the required vendor identity",
				Err:         fmt.Errorf("entrypoint vendor does not match its target vendor"),
			}
		}
	}
	return nil
}

func validateResolvedPiTarget(target ProjectTarget, profiles map[string]PiProfile) error {
	profileName := ""
	if target.Profile != nil {
		profileName = *target.Profile
	}
	profile, ok := profiles[profileName]
	if !ok {
		return &CanonicalTargetError{
			Code:        PrimarySessionErrorInvalidTarget,
			Context:     TargetErrorContext{Target: target.Name, Profile: profileName, Field: targetsField + "." + target.Name + ".profile", Source: target.Source},
			Remediation: "define the referenced complete managed Pi profile or select an existing exact profile name",
			Err:         errors.New("canonical Qwen target references an unknown managed Pi profile"),
		}
	}
	mismatch := func(field, remediation string) error {
		return &CanonicalTargetError{
			Code:        PrimarySessionErrorInvalidTarget,
			Context:     TargetErrorContext{Target: target.Name, Profile: profileName, Field: targetsField + "." + target.Name + "." + field, Source: target.Source},
			Remediation: remediation,
			Err:         errors.New("canonical Qwen target assertion does not match the selected managed Pi profile"),
		}
	}
	if profile.API != "openai-completions" {
		return mismatch("profile", "select a managed Pi profile whose api is exactly openai-completions")
	}
	if target.Model != profile.Model {
		return mismatch("model", "set target model to the selected profile model exactly")
	}
	if target.Reasoning != profile.Thinking {
		return mismatch("reasoning", "set target reasoning to the selected profile thinking value exactly")
	}
	if target.ProfileProvider != nil && *target.ProfileProvider != profile.Provider {
		return mismatch("profile_provider", "remove the optional assertion or set it to the selected profile provider exactly")
	}
	if target.Endpoint != nil && *target.Endpoint != profile.BaseURL {
		return mismatch("endpoint", "remove the optional assertion or set it to the selected profile base_url exactly")
	}
	return nil
}

func ResolveCanonicalTarget(entrypoint, projectDir, homeDir string) (ResolvedCanonicalTarget, error) {
	composite, err := loadCanonicalTargetConfig(projectDir, homeDir)
	if err != nil {
		return ResolvedCanonicalTarget{}, err
	}
	mapping, ok := composite.Entrypoints[entrypoint]
	if !ok {
		return ResolvedCanonicalTarget{}, &CanonicalTargetError{
			Code:        PrimarySessionErrorUnknownEntrypoint,
			Context:     TargetErrorContext{Entrypoint: entrypoint, Field: entrypointsField + "." + entrypoint},
			Remediation: "add an explicit mapping for this installed alias in [agents.entrypoints]",
			Err:         errors.New("canonical entrypoint is not configured; legacy provider policy is not a fallback"),
		}
	}
	target := composite.Targets[mapping.TargetName]
	resolved := ResolvedCanonicalTarget{Entrypoint: mapping, Target: target}
	if target.Environment == "pi" {
		profile := composite.PiProfiles[*target.Profile]
		resolved.Profile = &profile
		resolved.EffectiveProvider = profile.Provider
		resolved.EffectiveEndpoint = profile.BaseURL
	}
	return resolved, nil
}

func targetIdentityConflict(resolved ResolvedCanonicalTarget, field, remediation string) error {
	return &CanonicalTargetError{
		Code:        PrimarySessionErrorTargetIdentityConflict,
		Context:     TargetErrorContext{Entrypoint: resolved.Entrypoint.Name, Target: resolved.Target.Name, Field: field, Source: resolved.Target.Source},
		Remediation: remediation,
		Err:         errors.New("explicit provider arguments diverge from the configured canonical target identity"),
	}
}

func BuildCanonicalTargetLaunchPlan(entrypoint, projectDir, homeDir string, userArgs []string, producer ChildLaunchCompositionProducer, lookPath func(string) (string, error)) (PrimarySessionLaunchPlan, error) {
	canonicalProjectDir, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return PrimarySessionLaunchPlan{}, err
	}
	resolved, err := ResolveCanonicalTarget(entrypoint, canonicalProjectDir, homeDir)
	if err != nil {
		return PrimarySessionLaunchPlan{}, err
	}
	lockedArgs, err := lockCanonicalTargetArguments(resolved, userArgs)
	if err != nil {
		var targetErr *CanonicalTargetError
		if errors.As(err, &targetErr) {
			return PrimarySessionLaunchPlan{}, err
		}
		var argErr *ProviderArgumentError
		if errors.As(err, &argErr) {
			return PrimarySessionLaunchPlan{}, &PrimarySessionComposeError{Code: PrimarySessionErrorInvalidProviderArguments, Err: err}
		}
		return PrimarySessionLaunchPlan{}, err
	}
	provider := canonicalProviderForEnvironment(resolved.Target.Environment)
	plan, err := BuildPrimarySessionLaunchPlan(provider, canonicalProjectDir, homeDir, lockedArgs, producer, lookPath)
	if err != nil {
		return PrimarySessionLaunchPlan{}, err
	}
	plan.targetProviderArgs = append([]string(nil), lockedArgs...)
	plan.Target = primarySessionTargetFromResolution(resolved)
	if provider == "codex" || provider == "claude" {
		plan.Resolved.Model = resolvedStringValue(resolved.Target.Model, resolved.Target.Source)
		plan.Resolved.Reasoning = resolvedStringValue(resolved.Target.Reasoning, resolved.Target.Source)
		plan.Resolved.Profile = PrimarySessionResolvedString{Source: map[string]string{"codex": "native", "claude": "not_applicable"}[provider]}
		plan.Resolved.ProfileProvider = &PrimarySessionResolvedString{Source: "not_applicable"}
		plan.Resolved.Endpoint = &PrimarySessionResolvedString{Source: "not_applicable"}
	} else {
		profile := *resolved.Profile
		plan.Resolved.Profile = resolvedStringValue(*resolved.Target.Profile, resolved.Target.Source)
		plan.Resolved.ProfileProvider = primarySessionResolvedStringPointer(profile.Provider, profile.Source)
		plan.Resolved.Endpoint = primarySessionResolvedStringPointer(profile.BaseURL, profile.Source)
	}
	return plan, nil
}

func primarySessionTargetFromResolution(resolved ResolvedCanonicalTarget) *PrimarySessionTarget {
	return &PrimarySessionTarget{
		Entrypoint:       resolved.Entrypoint.Name,
		EntrypointSource: resolved.Entrypoint.Source,
		Name:             resolved.Target.Name,
		Source:           resolved.Target.Source,
		Vendor:           resolved.Target.Vendor,
		Environment:      resolved.Target.Environment,
		Model:            resolved.Target.Model,
		Reasoning:        resolved.Target.Reasoning,
		Profile:          cloneStringPointer(resolved.Target.Profile),
		ProfileProvider:  cloneStringPointer(resolved.Target.ProfileProvider),
		Endpoint:         cloneStringPointer(resolved.Target.Endpoint),
	}
}

func primarySessionResolvedStringPointer(value, source string) *PrimarySessionResolvedString {
	resolved := resolvedStringValue(value, source)
	return &resolved
}

func RenderCanonicalTargetLaunchPlan(plan PrimarySessionLaunchPlan) string {
	if plan.Target == nil {
		return ""
	}
	target := plan.Target
	optional := func(value *string) string {
		if value == nil {
			return "<not-configured>"
		}
		return *value
	}
	var b strings.Builder
	fmt.Fprintf(&b, "entrypoint: %s\n", target.Entrypoint)
	fmt.Fprintf(&b, "entrypoint_source: %s\n", target.EntrypointSource)
	fmt.Fprintf(&b, "target: %s\n", target.Name)
	fmt.Fprintf(&b, "target_source: %s\n", target.Source)
	fmt.Fprintf(&b, "vendor: %s\n", target.Vendor)
	fmt.Fprintf(&b, "environment: %s\n", target.Environment)
	fmt.Fprintf(&b, "model: %s\n", target.Model)
	fmt.Fprintf(&b, "reasoning: %s\n", target.Reasoning)
	fmt.Fprintf(&b, "profile: %s\n", optional(target.Profile))
	fmt.Fprintf(&b, "profile_provider_assertion: %s\n", optional(target.ProfileProvider))
	fmt.Fprintf(&b, "endpoint_assertion: %s\n", optional(target.Endpoint))
	fmt.Fprintf(&b, "effective_launch_provider: %s\n", plan.Provider)
	resolved := func(name string, value PrimarySessionResolvedString) {
		fmt.Fprintf(&b, "%s: %s\n", name, optional(value.Value))
		fmt.Fprintf(&b, "%s_source: %s\n", name, value.Source)
	}
	resolved("effective_model", plan.Resolved.Model)
	resolved("effective_reasoning", plan.Resolved.Reasoning)
	resolved("effective_profile", plan.Resolved.Profile)
	if plan.Resolved.ProfileProvider != nil {
		resolved("effective_profile_provider", *plan.Resolved.ProfileProvider)
	}
	if plan.Resolved.Endpoint != nil {
		resolved("effective_endpoint", *plan.Resolved.Endpoint)
	}
	b.WriteString("provider_argv:\n")
	for _, arg := range redactCanonicalDiagnosticArgs(plan.LaunchVariants.Interactive.Argv) {
		fmt.Fprintf(&b, "  - %s\n", strconv.Quote(arg))
	}
	return b.String()
}

func redactCanonicalDiagnosticArgs(args []string) []string {
	result := append([]string(nil), args...)
	for i := 0; i < len(result); i++ {
		switch {
		case result[i] == "--api-key" && i+1 < len(result):
			i++
			result[i] = "<redacted>"
		case strings.HasPrefix(result[i], "--api-key="):
			result[i] = "--api-key=<redacted>"
		}
	}
	return result
}

func lockCanonicalTargetArguments(resolved ResolvedCanonicalTarget, args []string) ([]string, error) {
	switch resolved.Target.Environment {
	case "codex":
		return lockCodexTargetArguments(resolved, args)
	case "claude-code":
		return lockClaudeTargetArguments(resolved, args)
	case "pi":
		return lockPiTargetArguments(resolved, args)
	default:
		return nil, targetIdentityConflict(resolved, targetsField+"."+resolved.Target.Name+".environment", "select an admitted target environment")
	}
}

func requireExactTargetValue(resolved ResolvedCanonicalTarget, field, got, want string) error {
	if got == want {
		return nil
	}
	return targetIdentityConflict(resolved, field, "remove the conflicting selector or repeat the configured target value exactly")
}

func lockCodexTargetArguments(resolved ResolvedCanonicalTarget, args []string) ([]string, error) {
	out := []string{"--model", resolved.Target.Model, "-c", fmt.Sprintf("model_reasoning_effort=%s", strconv.Quote(resolved.Target.Reasoning))}
	wrapperDelimiterSeen := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			// The hosted wrapper consumes the first delimiter and continues
			// parsing provider-native selectors after it. A second delimiter
			// reaches Codex itself and is its operand boundary.
			out = append(out, arg)
			if wrapperDelimiterSeen {
				return append(out, args[i+1:]...), nil
			}
			wrapperDelimiterSeen = true
			continue
		}
		take := func(name string) (string, error) {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", providerArgErrorf("a value is required for the Codex argument %s", name)
			}
			i++
			return args[i], nil
		}
		switch {
		case arg == "--model" || arg == "-m":
			value, err := take(arg)
			if err != nil {
				return nil, err
			}
			if err := requireExactTargetValue(resolved, "provider_args.model", value, resolved.Target.Model); err != nil {
				return nil, err
			}
		case strings.HasPrefix(arg, "--model=") || strings.HasPrefix(arg, "-m="):
			_, value, _ := strings.Cut(arg, "=")
			if err := requireExactTargetValue(resolved, "provider_args.model", value, resolved.Target.Model); err != nil {
				return nil, err
			}
		case strings.HasPrefix(arg, "-m") && len(arg) > 2:
			if err := requireExactTargetValue(resolved, "provider_args.model", strings.TrimPrefix(arg, "-m"), resolved.Target.Model); err != nil {
				return nil, err
			}
		case arg == "--model-reasoning-effort":
			value, err := take(arg)
			if err != nil {
				return nil, err
			}
			if err := requireExactTargetValue(resolved, "provider_args.reasoning", value, resolved.Target.Reasoning); err != nil {
				return nil, err
			}
		case strings.HasPrefix(arg, "--model-reasoning-effort="):
			if err := requireExactTargetValue(resolved, "provider_args.reasoning", strings.TrimPrefix(arg, "--model-reasoning-effort="), resolved.Target.Reasoning); err != nil {
				return nil, err
			}
		case arg == "--profile" || arg == "-p":
			if _, err := take(arg); err != nil {
				return nil, err
			}
			return nil, targetIdentityConflict(resolved, "provider_args.profile", "canonical hosted targets do not admit a Codex profile; remove the profile selector")
		case strings.HasPrefix(arg, "--profile=") || strings.HasPrefix(arg, "-p=") || (strings.HasPrefix(arg, "-p") && len(arg) > 2):
			return nil, targetIdentityConflict(resolved, "provider_args.profile", "canonical hosted targets do not admit a Codex profile; remove the profile selector")
		case arg == "-c" || arg == "--config":
			value, err := take(arg)
			if err != nil {
				return nil, err
			}
			keep, err := lockCodexTargetConfigOverride(resolved, value)
			if err != nil {
				return nil, err
			}
			if keep {
				out = append(out, arg, value)
			}
		case strings.HasPrefix(arg, "-c=") || strings.HasPrefix(arg, "--config="):
			_, value, _ := strings.Cut(arg, "=")
			keep, err := lockCodexTargetConfigOverride(resolved, value)
			if err != nil {
				return nil, err
			}
			if keep {
				out = append(out, arg)
			}
		default:
			out = append(out, arg)
		}
	}
	return out, nil
}

func lockCodexTargetConfigOverride(resolved ResolvedCanonicalTarget, override string) (bool, error) {
	key, raw, ok := strings.Cut(override, "=")
	if !ok {
		return true, nil
	}
	switch strings.TrimSpace(key) {
	case "model":
		value := configCodexExplicitValue(raw, "canonical-target model").effective
		return false, requireExactTargetValue(resolved, "provider_args.model", value, resolved.Target.Model)
	case "model_reasoning_effort":
		value := configCodexExplicitValue(raw, "canonical-target reasoning").effective
		return false, requireExactTargetValue(resolved, "provider_args.reasoning", value, resolved.Target.Reasoning)
	case "profile":
		return false, targetIdentityConflict(resolved, "provider_args.profile", "canonical hosted targets do not admit a Codex profile; remove the profile config override")
	default:
		return true, nil
	}
}

func lockClaudeTargetArguments(resolved ResolvedCanonicalTarget, args []string) ([]string, error) {
	out := []string{"--model", resolved.Target.Model, "--effort", resolved.Target.Reasoning}
	wrapperDelimiterSeen := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			// The hosted wrapper consumes the first delimiter. Claude sees a
			// second delimiter, so only that one ends identity selection.
			out = append(out, arg)
			if wrapperDelimiterSeen {
				return append(out, args[i+1:]...), nil
			}
			wrapperDelimiterSeen = true
			continue
		}
		take := func(name string) (string, error) {
			if i+1 >= len(args) {
				return "", providerArgErrorf("the Claude argument %s requires a value", name)
			}
			i++
			return args[i], nil
		}
		switch {
		case arg == "--model":
			value, err := take(arg)
			if err != nil {
				return nil, err
			}
			if err := requireExactTargetValue(resolved, "provider_args.model", value, resolved.Target.Model); err != nil {
				return nil, err
			}
		case strings.HasPrefix(arg, "--model="):
			if err := requireExactTargetValue(resolved, "provider_args.model", strings.TrimPrefix(arg, "--model="), resolved.Target.Model); err != nil {
				return nil, err
			}
		case arg == "--effort":
			value, err := take(arg)
			if err != nil {
				return nil, err
			}
			if err := requireExactTargetValue(resolved, "provider_args.reasoning", value, resolved.Target.Reasoning); err != nil {
				return nil, err
			}
		case strings.HasPrefix(arg, "--effort="):
			if err := requireExactTargetValue(resolved, "provider_args.reasoning", strings.TrimPrefix(arg, "--effort="), resolved.Target.Reasoning); err != nil {
				return nil, err
			}
		default:
			out = append(out, arg)
		}
	}
	return out, nil
}

func lockPiTargetArguments(resolved ResolvedCanonicalTarget, args []string) ([]string, error) {
	profile := *resolved.Profile
	out := []string{"--profile", *resolved.Target.Profile, "--provider", profile.Provider, "--model", resolved.Target.Model, "--thinking", resolved.Target.Reasoning}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return append(out, args[i:]...), nil
		}
		take := func(name string) (string, error) {
			if i+1 >= len(args) {
				return "", piError("invalid_provider_arguments", fmt.Errorf("%s requires a value", name))
			}
			i++
			return args[i], nil
		}
		name, equalValue, hasEqual := strings.Cut(arg, "=")
		if hasEqual && containsString([]string{"--profile", "--provider", "--model", "--thinking", "--endpoint"}, name) {
			if err := checkPiTargetCoordinate(resolved, profile, name, equalValue); err != nil {
				return nil, err
			}
			continue
		}
		if containsString([]string{"--profile", "--provider", "--model", "--thinking", "--endpoint"}, arg) {
			value, err := take(arg)
			if err != nil {
				return nil, err
			}
			if err := checkPiTargetCoordinate(resolved, profile, arg, value); err != nil {
				return nil, err
			}
			continue
		}
		out = append(out, arg)
	}
	return out, nil
}

func checkPiTargetCoordinate(resolved ResolvedCanonicalTarget, profile PiProfile, selector, value string) error {
	switch selector {
	case "--profile":
		return requireExactTargetValue(resolved, "provider_args.profile", value, *resolved.Target.Profile)
	case "--provider":
		return requireExactTargetValue(resolved, "provider_args.profile_provider", value, profile.Provider)
	case "--thinking":
		return requireExactTargetValue(resolved, "provider_args.reasoning", value, resolved.Target.Reasoning)
	case "--endpoint":
		return requireExactTargetValue(resolved, "provider_args.endpoint", value, profile.BaseURL)
	case "--model":
		allowed := []string{
			resolved.Target.Model,
			profile.Provider + "/" + resolved.Target.Model,
			resolved.Target.Model + ":" + resolved.Target.Reasoning,
			profile.Provider + "/" + resolved.Target.Model + ":" + resolved.Target.Reasoning,
		}
		if containsString(allowed, value) {
			return nil
		}
		return targetIdentityConflict(resolved, "provider_args.model", "use the exact target model, optional selected profile provider, and optional matching thinking suffix")
	}
	return nil
}
