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
)

const defaultPiInferenceEngineID plugin.ID = "mlx"

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
