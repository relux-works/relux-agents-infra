package infra

import (
	"errors"
	"fmt"

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
	Registry *vendorplugin.Registry
	Runtime  vendorplugin.RuntimeID
	Model    vendorplugin.ModelID
	Profile  string
	Engine   plugin.Ref
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
	runtimeID, err := vendorplugin.NormalizeRuntimeID(resolved.Target.Name)
	if err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: canonical target cannot name a plugin runtime: %w", err)
	}
	modelID := vendorplugin.ModelID(resolved.Target.Model)
	if err := vendorplugin.ValidateModelID(modelID); err != nil {
		return PiPluginGraph{}, fmt.Errorf("agents-infra: canonical target cannot name a plugin model: %w", err)
	}
	profileName := *resolved.Target.Profile
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

	profile := *resolved.Profile
	config := localmodels.Config{
		InferenceEngines: []plugin.ID{engine.ID},
		Runtimes: []localmodels.RuntimeEntry{{
			ID:     runtimeID,
			System: "pi",
			Engine: engine,
			Models: map[vendorplugin.ModelID]localmodels.ModelEntry{
				modelID: {
					Description:         "Local model selected by the canonical agents-infra Pi target",
					Publisher:           resolved.Target.Vendor,
					Family:              resolved.Target.Vendor,
					Lifecycle:           vendorplugin.LifecycleCurrent,
					ContextWindowTokens: profile.ContextWindow,
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
	return PiPluginGraph{Registry: registry, Runtime: runtimeID, Model: modelID, Profile: profileName, Engine: engine}, nil
}

// ResolvePiPluginGraph resolves the canonical Pi target and assembles the
// trusted plugin graph for it. A nil status or observations reader is filled
// with agents-infra's real production reader for that already-resolved
// profile: localruntime.NewCLIStatusReader for preflight, and
// SharedRuntimeSanitizedEngineObservationReader for the sanitized engine
// observation. Tests that need a fake reader still pass one explicitly.
func ResolvePiPluginGraph(projectDir, homeDir, entrypoint string, status localruntime.StatusReader, observations SanitizedEngineObservationReader) (PiPluginGraph, error) {
	project, err := CanonicalProjectDir(projectDir)
	if err != nil {
		return PiPluginGraph{}, err
	}
	resolved, err := ResolveCanonicalTarget(entrypoint, project, homeDir)
	if err != nil {
		return PiPluginGraph{}, err
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
