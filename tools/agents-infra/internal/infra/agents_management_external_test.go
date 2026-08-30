package infra_test

import (
	"context"

	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
)

// This external-package declaration intentionally compiles against the exact
// accepted public candidate. It prevents tests from leaning on internal or
// copied observer/turn types.
var (
	_ vendorplugin.EngineObservationAdapter = externalAdapter{}
	_                                       = vendorplugin.NewRegistryWithEngineObservationAdapters
	_                                       = managementpi.ValidateTurnResult
	_                                       = managementpi.TurnResultInput{
		Exit:         managementpi.ProcessAExit{Code: 0},
		Intervention: managementpi.TurnInterventionNone,
		Cleanup:      managementpi.ProcessACleanupNotRequired,
	}
)

type externalAdapter struct{}

func (externalAdapter) EngineObservationAdapterDeclaration() vendorplugin.EngineObservationAdapterDeclaration {
	return vendorplugin.EngineObservationAdapterDeclaration{}
}

func (externalAdapter) ObserveEngine(context.Context, vendorplugin.EngineObservationQuery) (vendorplugin.EngineObservation, error) {
	return vendorplugin.EngineObservation{}, nil
}
