package infra

import (
	"errors"
	"fmt"
	"strings"

	"github.com/relux-works/skill-agents-management/pkg/agentic"
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
	"github.com/relux-works/skill-agents-management/pkg/inferenceengine"
	"github.com/relux-works/skill-agents-management/pkg/localruntime"
	"github.com/relux-works/skill-agents-management/pkg/plugin"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
	localmodels "github.com/relux-works/skill-agents-management/pkg/vendorplugin/vendors/local-models"

	// Blank-imported so their init() registers "codex" and "claude-code" into
	// agentic.Default. Pi registers the same way through the managementpi
	// import above, which this file already uses for its constructor. Without
	// these two, ValidateLaunchableEnvironment's registry cross-check would
	// have nothing to find codex or claude-code under, and would fail closed
	// on every launch — the opposite of the "registered but unlaunchable"
	// refusal the cross-check exists to produce for a system such as
	// gemini-cli, which this program deliberately never imports.
	_ "github.com/relux-works/skill-agents-management/pkg/agentic/systems/claude"
	_ "github.com/relux-works/skill-agents-management/pkg/agentic/systems/codex"
)

const defaultPiInferenceEngineID plugin.ID = "mlx"

const (
	launchableEnvironmentCodex      = "codex"
	launchableEnvironmentClaudeCode = "claude-code"
	launchableEnvironmentPi         = "pi"

	launchableProviderCodex  = "codex"
	launchableProviderClaude = "claude"
	launchableProviderPi     = "pi"
)

// LaunchableSystem pairs the environment identifier agents-infra's own
// project configuration and canonical-target resolution use
// (agents.targets.*.environment) with the provider label agents-infra's
// primary-session dispatch and preparation surfaces use for the very same
// agentic system.
//
// This table is the ONLY place the admitted agentic-system identifiers may be
// enumerated as a set. Every admission and dispatch site in
// project_config.go, canonical_target.go and primary_session_launch_plan.go
// reads it through the accessors below instead of restating "codex",
// "claude-code", "claude" or "pi" as a literal list or a switch keyed
// directly on those strings.
//
// agentic.Default — the compatibility registry skill-agents-management's
// system plugins self-register into via their own init() — is WIDER than
// this table: it also carries gemini-cli, muse, qwen-code and antigravity,
// none of which this program has a launcher for. Registry membership is
// therefore necessary but never sufficient for admission: both
// ValidateLaunchableEnvironment and ValidateLaunchableProvider require the
// identifier to be declared here AND registered, under that exact spelling,
// before admitting it.
type LaunchableSystem struct {
	// Environment is the agents-infra project-config spelling and must equal
	// a registered agentic.SystemID exactly, byte for byte — not merely its
	// normalized form, so that a case or alias variant of an admitted id
	// (for example "Codex" or "claude") is refused rather than silently
	// folded onto its canonical spelling.
	Environment string
	// Provider is agents-infra's own primary-session dispatch label: the
	// spelling BuildPrimarySessionLaunchPlan selects a builder with and, for
	// a hosted system, the executable name exec.LookPath resolves. It
	// predates and differs from the agentic system id for Claude
	// (claude-code / claude); the two are declared together here rather than
	// derived from one another because nothing after registration recovers
	// one spelling from the other.
	Provider string
	// ResolvesExecutable reports whether this provider's launch plan resolves
	// its executable through exec.LookPath. Pi is the one declared system for
	// which this is false: it has no primary-session executable of its own.
	ResolvesExecutable bool
}

// launchableSystems is the single declaration named above.
var launchableSystems = []LaunchableSystem{
	{Environment: launchableEnvironmentCodex, Provider: launchableProviderCodex, ResolvesExecutable: true},
	{Environment: launchableEnvironmentClaudeCode, Provider: launchableProviderClaude, ResolvesExecutable: true},
	{Environment: launchableEnvironmentPi, Provider: launchableProviderPi, ResolvesExecutable: false},
}

// LaunchableEnvironments returns the admitted agents.targets.*.environment
// identifiers, in declaration order. The slice is freshly built on every
// call so a caller cannot reorder or grow the declaration through it.
func LaunchableEnvironments() []string {
	out := make([]string, len(launchableSystems))
	for i, system := range launchableSystems {
		out[i] = system.Environment
	}
	return out
}

// LaunchableProviders returns the admitted primary-session provider labels,
// in declaration order.
func LaunchableProviders() []string {
	out := make([]string, len(launchableSystems))
	for i, system := range launchableSystems {
		out[i] = system.Provider
	}
	return out
}

// launchableSystemForEnvironment returns the declared row for environment,
// and false when environment is not one of launchableSystems.
func launchableSystemForEnvironment(environment string) (LaunchableSystem, bool) {
	for _, system := range launchableSystems {
		if system.Environment == environment {
			return system, true
		}
	}
	return LaunchableSystem{}, false
}

// launchableSystemForProvider returns the declared row for provider, and
// false when provider is not one of launchableSystems.
func launchableSystemForProvider(provider string) (LaunchableSystem, bool) {
	for _, system := range launchableSystems {
		if system.Provider == provider {
			return system, true
		}
	}
	return LaunchableSystem{}, false
}

// ProviderForLaunchableEnvironment returns the provider label declared for
// environment, and false when environment is not admitted. It is the single
// source canonicalProviderForEnvironment reads from, replacing what used to
// be a second, independently maintained switch over the same three strings.
func ProviderForLaunchableEnvironment(environment string) (string, bool) {
	system, ok := launchableSystemForEnvironment(environment)
	return system.Provider, ok
}

// agenticLookup is the shape of the registry cross-check every admission
// entry runs through: agentic.Default.Lookup in production, and an isolated
// agentic.NewRegistry()'s Lookup in a test that needs to prove the check is
// load-bearing without being able to mutate the real global registry.
type agenticLookup func(agentic.SystemID) (agentic.System, bool)

// ValidateLaunchableEnvironment is the production entry every admission path
// that accepts an agents.targets.*.environment value calls. It refuses any
// identifier that is not BOTH declared in launchableSystems AND registered,
// under that exact spelling, in the real agentic compatibility registry.
//
// The agentic registry is wider than what this program can launch, so
// registry membership alone never admits a system: a registered-but-
// unlaunchable id such as gemini-cli is refused here precisely because it is
// absent from launchableSystems. The declaration is not trusted on its own
// either — a typo naming a system nothing actually registered, or a change
// that dropped the registry cross-check, is caught here instead of silently
// admitting a launch nothing downstream can dispatch.
func ValidateLaunchableEnvironment(environment string) error {
	return validateLaunchableEnvironment(environment, agentic.Default.Lookup)
}

func validateLaunchableEnvironment(environment string, lookup agenticLookup) error {
	admitted := LaunchableEnvironments()
	if !containsString(admitted, environment) {
		return fmt.Errorf("must be one of %s", strings.Join(admitted, ", "))
	}
	if _, ok := lookup(agentic.SystemID(environment)); !ok {
		return fmt.Errorf("must be one of %s", strings.Join(admitted, ", "))
	}
	return nil
}

// ValidateLaunchableProvider is the production entry every admission path
// that accepts a primary-session provider label calls. The same declared-
// AND-registered rule as ValidateLaunchableEnvironment applies, cross-checked
// through the provider's declared environment spelling — the spelling the
// agentic registry actually keys on.
func ValidateLaunchableProvider(provider string) error {
	return validateLaunchableProvider(provider, agentic.Default.Lookup)
}

func validateLaunchableProvider(provider string, lookup agenticLookup) error {
	system, ok := launchableSystemForProvider(provider)
	if !ok {
		return fmt.Errorf("unsupported provider %q", provider)
	}
	if _, ok := lookup(agentic.SystemID(system.Environment)); !ok {
		return fmt.Errorf("unsupported provider %q", provider)
	}
	return nil
}

// PiPluginGraph is the trusted assembly result consumed by BuildLaunch. The
// registry owns the adapter; launch callers receive no observation setter or
// per-request evidence field.
type PiPluginGraph struct {
	Registry   *vendorplugin.Registry
	Runtime    vendorplugin.RuntimeID
	Model      vendorplugin.ModelID
	Profile    string
	Engine     plugin.Ref
	Provenance PiPluginGraphProvenance
}

// PiPluginGraphProvenance is the exact canonical-target identity the graph was
// assembled from: the explicit entrypoint, the target it maps to, the selected
// profile, and the profile-derived effective provider and endpoint, each with
// the configuration source that declared it. It is copied from the resolution
// and never re-derived from model names, argv, or plugin rows.
type PiPluginGraphProvenance struct {
	Entrypoint       string
	EntrypointSource string
	Target           string
	TargetSource     string
	Vendor           string
	Environment      string
	Model            string
	Profile          string
	ProfileSource    string
	Provider         string
	Endpoint         string
}

func piPluginGraphProvenance(resolved ResolvedCanonicalTarget) PiPluginGraphProvenance {
	provenance := PiPluginGraphProvenance{
		Entrypoint:       resolved.Entrypoint.Name,
		EntrypointSource: resolved.Entrypoint.Source,
		Target:           resolved.Target.Name,
		TargetSource:     resolved.Target.Source,
		Vendor:           resolved.Target.Vendor,
		Environment:      resolved.Target.Environment,
		Model:            resolved.Target.Model,
		Provider:         resolved.EffectiveProvider,
		Endpoint:         resolved.EffectiveEndpoint,
	}
	if resolved.Target.Profile != nil {
		provenance.Profile = *resolved.Target.Profile
	}
	if resolved.Profile != nil {
		provenance.ProfileSource = resolved.Profile.Source
	}
	return provenance
}

// BuildPiPluginGraph consumes an already-resolved canonical Pi target. The
// agents-infra target vendor remains a product label; the plugin-plane vendor
// is always local-models, and the registered agentic system is always Pi.
func BuildPiPluginGraph(project string, resolved ResolvedCanonicalTarget, status localruntime.StatusReader, observations SanitizedEngineObservationReader) (PiPluginGraph, error) {
	if status == nil {
		return PiPluginGraph{}, errors.New("agents-infra: local runtime status reader is required")
	}
	if resolved.Target.Environment != "pi" || resolved.Target.Profile == nil || resolved.Profile == nil {
		return PiPluginGraph{}, errors.New("agents-infra: canonical target is not a resolved Pi profile")
	}
	profileName := *resolved.Target.Profile
	profile := *resolved.Profile
	if resolved.Target.Model != profile.Model {
		return PiPluginGraph{}, errors.New("agents-infra: resolved Pi target model contradicts the selected profile")
	}
	if resolved.Target.ProfileProvider != nil && *resolved.Target.ProfileProvider != profile.Provider {
		return PiPluginGraph{}, errors.New("agents-infra: resolved Pi target provider assertion contradicts the selected profile")
	}
	if resolved.EffectiveProvider != "" && resolved.EffectiveProvider != profile.Provider {
		return PiPluginGraph{}, errors.New("agents-infra: resolved Pi target effective provider contradicts the selected profile")
	}
	if resolved.EffectiveEndpoint != "" && resolved.EffectiveEndpoint != profile.BaseURL {
		return PiPluginGraph{}, errors.New("agents-infra: resolved Pi target effective endpoint contradicts the selected profile")
	}
	runtimeID, err := vendorplugin.NormalizeRuntimeID(resolved.Target.Name)
	if err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: canonical target cannot name a plugin runtime: %w", err)
	}
	modelID := vendorplugin.ModelID(resolved.Target.Model)
	if err := vendorplugin.ValidateModelID(modelID); err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: canonical target cannot name a plugin model: %w", err)
	}
	engine := plugin.Ref{ID: defaultPiInferenceEngineID, Kind: inferenceengine.Kind}
	adapter, err := NewSanitizedEngineObservationAdapter(engine, observations)
	if err != nil {
		return PiPluginGraph{}, err
	}

	systems := agentic.NewRegistry()
	if err := systems.Register(managementpi.New(status)); err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: register Pi system plugin: %w", err)
	}
	registry, err := vendorplugin.NewRegistryWithEngineObservationAdapters(systems, adapter)
	if err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: construct vendor registry: %w", err)
	}
	if err := registry.RegisterPlugin(inferenceengine.NewConfigured(engine.ID)); err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: register inference engine: %w", err)
	}

	cacheBudget := cloneInt64Pointer(profile.CacheBudgetBytes)
	config := localmodels.Config{
		InferenceEngines: []plugin.ID{engine.ID},
		Runtimes: []localmodels.RuntimeEntry{{
			ID:     runtimeID,
			System: "pi",
			Engine: engine,
			Models: map[vendorplugin.ModelID]localmodels.ModelEntry{
				modelID: {
					Description:         "Local model selected by the canonical agents-infra Pi target",
					Publisher:           profile.Publisher,
					Family:              profile.Family,
					Lifecycle:           vendorplugin.LifecycleCurrent,
					ContextWindowTokens: profile.ContextWindow,
					CacheBudgetBytes:    cacheBudget,
					EffortSupport:       agentic.EffortSupportNone,
					Engine:              engine,
					Pointer: localmodels.Pointer{
						AgentsInfraProject: project,
						AgentsInfraProfile: profileName,
					},
				},
			},
		}},
	}
	if err := registry.Register(localmodels.New(config, localmodels.WithStatusReader(status))); err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: register local-models vendor: %w", err)
	}
	if err := registry.DeclareRuntime(vendorplugin.RuntimeDeclaration{
		ID:     runtimeID,
		System: "pi",
		Vendor: localmodels.VendorID,
		Broker: vendorplugin.BrokerProvenance{
			Checked: []string{"agents-infra canonical target"},
			Found:   "resolved canonical Pi target",
		},
		Engine: engine,
	}); err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: declare canonical Pi runtime: %w", err)
	}
	return PiPluginGraph{Registry: registry, Runtime: runtimeID, Model: modelID, Profile: profileName, Engine: engine, Provenance: piPluginGraphProvenance(resolved)}, nil
}

// ValidateExplicitPiEntrypoint is the provider-local gate every Pi consumer
// passes before canonical resolution: the entrypoint must be selected
// explicitly by the caller. An empty selection is refused as unknown_entrypoint
// here, before any configuration is read, so that neither a unique configured
// target, a model name, a vendor label, argv, nor the legacy provider policy
// can stand in for the missing selection.
func ValidateExplicitPiEntrypoint(entrypoint string) error {
	if strings.TrimSpace(entrypoint) == "" || strings.TrimSpace(entrypoint) != entrypoint {
		return &CanonicalTargetError{
			Code:        PrimarySessionErrorUnknownEntrypoint,
			Context:     TargetErrorContext{Field: entrypointsField},
			Remediation: "select one configured [agents.entrypoints] alias explicitly, for example --target qwen-infra",
			Err:         errors.New("canonical Pi entrypoint must be selected explicitly; a unique configured target, a model name, or legacy provider policy is not a fallback"),
		}
	}
	return nil
}

// ResolvePiPluginGraph resolves the canonical Pi target and assembles the
// trusted plugin graph for it. A nil status or observations reader is filled
// with agents-infra's real production reader for that already-resolved
// profile: localruntime.NewCLIStatusReader for preflight, and
// SharedRuntimeSanitizedEngineObservationReader for the sanitized engine
// observation. Tests that need a fake reader still pass one explicitly.
func ResolvePiPluginGraph(projectDir, homeDir, entrypoint string, status localruntime.StatusReader, observations SanitizedEngineObservationReader) (PiPluginGraph, error) {
	if err := ValidateExplicitPiEntrypoint(entrypoint); err != nil {
		return PiPluginGraph{}, err
	}
	project, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return PiPluginGraph{}, err
	}
	resolved, err := ResolveCanonicalTarget(entrypoint, project, homeDir)
	if err != nil {
		return PiPluginGraph{}, err
	}
	if resolved.Target.Environment != "pi" {
		return PiPluginGraph{}, &CanonicalTargetError{
			Code:        PrimarySessionErrorInvalidTarget,
			Context:     TargetErrorContext{Entrypoint: resolved.Entrypoint.Name, Target: resolved.Target.Name, Field: targetsField + "." + resolved.Target.Name + ".environment", Source: resolved.Target.Source},
			Remediation: "select an entrypoint whose target declares environment = \"pi\"",
			Err:         fmt.Errorf("canonical entrypoint selects environment %q, not the managed Pi environment", resolved.Target.Environment),
		}
	}
	if status == nil {
		status = localruntime.NewCLIStatusReader()
	}
	if observations == nil {
		if resolved.Profile == nil {
			return PiPluginGraph{}, errors.New("agents-infra: canonical target has no resolved Pi profile for engine observation")
		}
		observations = NewSharedRuntimeSanitizedEngineObservationReader(project, homeDir, "", *resolved.Profile)
	}
	return BuildPiPluginGraph(project, resolved, status, observations)
}

func (g PiPluginGraph) SpawnRequest(prompt []byte, workDir string, environ []string) vendorplugin.SpawnRequest {
	return vendorplugin.SpawnRequest{
		Runtime: g.Runtime,
		Model:   g.Model,
		Engine:  g.Engine,
		Prompt:  append([]byte(nil), prompt...),
		WorkDir: workDir,
		Profile: g.Profile,
		Env:     append([]string(nil), environ...),
	}
}
