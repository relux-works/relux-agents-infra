package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/relux-works/skill-agents-management/pkg/inferenceengine"
	"github.com/relux-works/skill-agents-management/pkg/plugin"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
)

const (
	SanitizedEngineObservationContract      = "agents-infra.sanitized-engine-observation"
	SanitizedEngineObservationSchemaVersion = 1
)

// SanitizedEngineFact is one closed inference-engine reading. It carries no
// process handle, environment value, credential, raw status payload, or
// lifecycle operation.
type SanitizedEngineFact struct {
	Fact    inferenceengine.Fact
	Outcome inferenceengine.Outcome
	Value   string
	Cause   inferenceengine.NotObservedCause
	Detail  string
}

// SanitizedEngineObservation is the versioned, value-only handoff from the
// agents-infra-owned Process-B observation plane to agents-management's
// immutable registry adapter. Identity is copied, never used to redirect a
// launch.
type SanitizedEngineObservation struct {
	Contract      string
	SchemaVersion int
	Engine        plugin.Ref
	Runtime       vendorplugin.RuntimeID
	Model         vendorplugin.ModelID
	Profile       string
	ObservedAt    time.Time
	ValidUntil    time.Time
	Facts         []SanitizedEngineFact
}

// SanitizedEngineObservationReader performs one bounded, read-only
// observation. The adapter invokes it exactly once for a BuildLaunch call.
type SanitizedEngineObservationReader interface {
	ReadSanitizedEngineObservation(context.Context, vendorplugin.EngineObservationQuery) (SanitizedEngineObservation, error)
}

type SanitizedEngineObservationReaderFunc func(context.Context, vendorplugin.EngineObservationQuery) (SanitizedEngineObservation, error)

func (f SanitizedEngineObservationReaderFunc) ReadSanitizedEngineObservation(ctx context.Context, query vendorplugin.EngineObservationQuery) (SanitizedEngineObservation, error) {
	return f(ctx, query)
}

// SanitizedEngineObservationAdapter is the concrete agents-infra adapter that
// the trusted assembly registers for its configured inference engine. Engine
// identity arrives as injected data, never as a literal on this launch path,
// so the adapter carries no runtime, model, or engine special case.
// Process-B ownership remains outside this type; it can only ask its injected
// reader for one bounded sanitized value.
type SanitizedEngineObservationAdapter struct {
	engine plugin.Ref
	reader SanitizedEngineObservationReader
}

func NewSanitizedEngineObservationAdapter(engine plugin.Ref, reader SanitizedEngineObservationReader) (*SanitizedEngineObservationAdapter, error) {
	if engine.ID == "" || engine.Kind != inferenceengine.Kind {
		return nil, fmt.Errorf("agents-infra: invalid inference-engine declaration %#v", engine)
	}
	if reader == nil {
		return nil, errors.New("agents-infra: sanitized engine observation reader is required")
	}
	return &SanitizedEngineObservationAdapter{engine: engine, reader: reader}, nil
}

func (a *SanitizedEngineObservationAdapter) EngineObservationAdapterDeclaration() vendorplugin.EngineObservationAdapterDeclaration {
	return vendorplugin.EngineObservationAdapterDeclaration{
		Contract:       vendorplugin.EngineObservationAdapterContract,
		SchemaVersion:  vendorplugin.EngineObservationAdapterSchemaVersion,
		Engine:         a.engine,
		EngineKind:     inferenceengine.EngineKindNativeTransformer,
		EngineContract: inferenceengine.ContractVersion,
	}
}

func (a *SanitizedEngineObservationAdapter) ObserveEngine(ctx context.Context, query vendorplugin.EngineObservationQuery) (vendorplugin.EngineObservation, error) {
	if err := ctx.Err(); err != nil {
		return vendorplugin.EngineObservation{}, err
	}
	snapshot, err := a.reader.ReadSanitizedEngineObservation(ctx, query)
	if err != nil {
		return vendorplugin.EngineObservation{}, err
	}
	if err := ctx.Err(); err != nil {
		return vendorplugin.EngineObservation{}, err
	}

	readings := make([]inferenceengine.Reading, 0, len(snapshot.Facts))
	for _, fact := range snapshot.Facts {
		var outcome inferenceengine.ReadOutcome
		switch fact.Outcome {
		case inferenceengine.OutcomeObservedValue:
			outcome = inferenceengine.ReadValue(fact.Value)
		case inferenceengine.OutcomeObservedAbsent:
			outcome = inferenceengine.ReadAbsent()
		case inferenceengine.OutcomeNotObserved:
			outcome = inferenceengine.ReadFailure(fact.Cause, fact.Detail)
		default:
			outcome = inferenceengine.ReadFailure(inferenceengine.NotObservedMalformed, "unknown sanitized outcome")
		}
		readings = append(readings, inferenceengine.NewReading(fact.Fact, outcome))
	}

	contract := vendorplugin.EngineObservationAdapterContract
	schema := vendorplugin.EngineObservationAdapterSchemaVersion
	if snapshot.Contract != SanitizedEngineObservationContract || snapshot.SchemaVersion != SanitizedEngineObservationSchemaVersion {
		contract = snapshot.Contract
		schema = snapshot.SchemaVersion
	}
	return vendorplugin.EngineObservation{
		Contract:      contract,
		SchemaVersion: schema,
		Engine:        snapshot.Engine,
		Runtime:       snapshot.Runtime,
		Model:         snapshot.Model,
		Profile:       snapshot.Profile,
		ObservedAt:    snapshot.ObservedAt,
		ValidUntil:    snapshot.ValidUntil,
		Readings:      readings,
	}, nil
}
