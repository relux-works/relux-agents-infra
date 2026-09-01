package infra

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/skill-agents-management/pkg/agentic"
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
	qwensystem "github.com/relux-works/skill-agents-management/pkg/agentic/systems/qwen"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
	localmodels "github.com/relux-works/skill-agents-management/pkg/vendorplugin/vendors/local-models"
)

// fakeProcessA writes an executable that stands in for the real
// `agents-infra pi spawn ... --result-schema 1` Process A. Nothing here starts
// a live runtime, model, socket, or broker.
func fakeProcessA(t *testing.T, body string) (dir string, env []string) {
	t.Helper()
	dir = t.TempDir()
	path := filepath.Join(dir, "agents-infra")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, []string{"PATH=" + dir}
}

func consumerFixture(t *testing.T, body string) (PiPluginGraph, vendorplugin.SpawnRequest, *recordingSanitizedObservationReader) {
	t.Helper()
	dir, env := fakeProcessA(t, body)
	observations := &recordingSanitizedObservationReader{}
	graph, _ := managementGraphFixture(t, observations, "")
	return graph, graph.SpawnRequest([]byte("prompt"), dir, env), observations
}

const okTurnDocument = `{"contract":"agents-infra.pi-turn-result","schema_version":1,"status":"ok","final_text":"accepted"}`

func refusalTurnDocument(code managementpi.TurnResultCode) string {
	return fmt.Sprintf(`{"contract":"agents-infra.pi-turn-result","schema_version":1,"status":"error","error":{"code":%q}}`, code)
}

// Production call site: BuildAndRunPiTurn -> OSProcessATurnRunner ->
// pi.ValidateTurnResult. Every exact pre-child refusal that agents-infra's real
// standalone entry can emit must reach the consumer as ProcessARefused with the
// exact closed code, never as a child, tool, invalid, or success class.
func TestConsumerClassifiesEveryExactPreChildRefusalThroughSoleClassifier(t *testing.T) {
	codes := []managementpi.TurnResultCode{
		managementpi.TurnCodeRequestInvalid,
		managementpi.TurnCodeProfileMissing,
		managementpi.TurnCodeProfileUnknown,
		managementpi.TurnCodeProfileMismatch,
		managementpi.TurnCodeEnvironmentMalformed,
		managementpi.TurnCodeEnvironmentDenied,
		managementpi.TurnCodeConfigurationInvalid,
		managementpi.TurnCodeAuthorizationDenied,
		managementpi.TurnCodeIdentityInvalid,
		managementpi.TurnCodeRuntimeRefused,
	}
	for _, code := range codes {
		t.Run(string(code), func(t *testing.T) {
			graph, request, _ := consumerFixture(t, "printf '%s' '"+refusalTurnDocument(code)+"'\nexit 1\n")
			result, err := BuildAndRunPiTurn(context.Background(), graph.Registry, request, OSProcessATurnRunner{})
			if result.Class != managementpi.TurnResultProcessARefused || result.Code != code {
				t.Fatalf("result = %#v, want process-a-refused/%s", result, code)
			}
			if !errors.Is(err, managementpi.ErrTurnProcessARefused) {
				t.Fatalf("error = %v, want ErrTurnProcessARefused", err)
			}
			if result.FinalText != "" {
				t.Fatalf("refusal carried final text %q", result.FinalText)
			}
		})
	}
}

// Negative: the consumer must not launder an exit/document disagreement into a
// success, a previous result, or a preflight class. Each case narrows one
// admitted shape rather than deleting the classifier call.
func TestConsumerRefusesExitDocumentDisagreementInsteadOfLaundering(t *testing.T) {
	oversize := strings.Repeat("a", managementpi.TurnResultMaxStdoutBytes+16)
	tests := []struct {
		name string
		body string
	}{
		{"zero exit without document", "exit 0\n"},
		{"zero exit with error document", "printf '%s' '" + refusalTurnDocument(managementpi.TurnCodeChildFailed) + "'\nexit 0\n"},
		{"error exit with ok document", "printf '%s' '" + okTurnDocument + "'\nexit 1\n"},
		{"duplicate document", "printf '%s%s' '" + okTurnDocument + "' '" + okTurnDocument + "'\nexit 0\n"},
		{"trailing non-whitespace bytes", "printf '%s' '" + okTurnDocument + "trailing'\nexit 0\n"},
		{"unknown error code", "printf '%s' '" + refusalTurnDocument("pi_turn_invented_code") + "'\nexit 1\n"},
		{"unknown schema version", `printf '%s' '{"contract":"agents-infra.pi-turn-result","schema_version":2,"status":"ok","final_text":"x"}'` + "\nexit 0\n"},
		{"exit outside closed table", "printf '%s' '" + okTurnDocument + "'\nexit 7\n"},
		{"over the bounded capture", "printf '%s' '" + oversize + "'\nexit 0\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			graph, request, _ := consumerFixture(t, test.body)
			result, err := BuildAndRunPiTurn(context.Background(), graph.Registry, request, OSProcessATurnRunner{})
			if result.Class != managementpi.TurnResultInvalid || result.Code != managementpi.TurnCodeResultInvalid {
				t.Fatalf("result = %#v, want result-invalid", result)
			}
			if !errors.Is(err, managementpi.ErrTurnResultInvalid) {
				t.Fatalf("error = %v, want ErrTurnResultInvalid", err)
			}
		})
	}
}

// Cancellation after the child exists must reach the classifier as a recorded
// consumer intervention with a bounded cleanup outcome, and must survive the
// induced signal exit that carries no schema-1 document.
func TestConsumerRecordsCancellationAfterChildStartAndKeepsItAuthoritative(t *testing.T) {
	graph, request, _ := consumerFixture(t, "sleep 30\n")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(250 * time.Millisecond)
		cancel()
	}()
	result, err := BuildAndRunPiTurn(ctx, graph.Registry, request, OSProcessATurnRunner{})
	if result.Class != managementpi.TurnResultCancelled || result.Code != managementpi.TurnCodeCancelled {
		t.Fatalf("result = %#v, want cancelled", result)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if strings.Contains(err.Error(), request.Profile) || strings.Contains(err.Error(), "sleep") {
		t.Fatalf("cancellation error leaked launch detail: %v", err)
	}
}

// A pre-cancelled caller must never reach the child at all, and must never be
// laundered into a schema-1 protocol failure.
func TestConsumerRefusesPreCancelledCallerBeforeAnyChildEffect(t *testing.T) {
	dir, env := fakeProcessA(t, ": > \"$FAKE_PROCESS_A_MARKER\"\nexit 0\n")
	marker := filepath.Join(dir, "started")
	observations := &recordingSanitizedObservationReader{}
	graph, status := managementGraphFixture(t, observations, "")
	request := graph.SpawnRequest([]byte("prompt"), dir, append(env, "FAKE_PROCESS_A_MARKER="+marker))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := BuildAndRunPiTurn(ctx, graph.Registry, request, OSProcessATurnRunner{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("pre-cancelled call started Process A")
	}
	if observations.calls != 0 || status.calls != 0 {
		t.Fatalf("pre-cancelled call observed=%d preflight=%d, want 0/0", observations.calls, status.calls)
	}
}

// Dry run plans the exact same outer grammar with the literal prompt
// placeholder and must perform no observation, preflight, or prompt read.
func TestDryRunPlanIsObservationAndPreflightFree(t *testing.T) {
	dir, env := fakeProcessA(t, "exit 0\n")
	observations := &recordingSanitizedObservationReader{}
	graph, status := managementGraphFixture(t, observations, "")
	plan, err := vendorplugin.BuildLaunch(context.Background(), graph.Registry, graph.SpawnRequest([]byte("prompt"), dir, env), agentic.LaunchModeDryRun)
	if err != nil {
		t.Fatalf("dry-run BuildLaunch: %v", err)
	}
	want := []string{"pi", "spawn", "--profile", graph.Profile, "--prompt", "<prompt>", "--deadline", "30m", "--result-schema", "1"}
	if !reflect.DeepEqual(plan.Argv, want) {
		t.Fatalf("dry-run argv = %#v, want %#v", plan.Argv, want)
	}
	if observations.calls != 0 || status.calls != 0 {
		t.Fatalf("dry run observed=%d preflight=%d, want 0/0", observations.calls, status.calls)
	}
}

// AC5. qwen-infra is an agents-infra product label. It must resolve to the Pi
// agentic system over the local-models plugin-plane vendor and must never
// select the shipped qwen system, which is qwen-code by Alibaba.
func TestQwenInfraResolvesToPiAndLocalModelsNeverShippedQwen(t *testing.T) {
	dir, env := fakeProcessA(t, "exit 0\n")
	observations := &recordingSanitizedObservationReader{}
	graph, _ := managementGraphFixture(t, observations, "")
	plan, err := vendorplugin.BuildLaunch(context.Background(), graph.Registry, graph.SpawnRequest([]byte("prompt"), dir, env), agentic.LaunchModeExec)
	if err != nil {
		t.Fatalf("BuildLaunch: %v", err)
	}
	shipped := qwensystem.New().ID()
	if shipped != "qwen-code" {
		t.Fatalf("shipped qwen system identity drifted: %q", shipped)
	}
	if plan.System == shipped {
		t.Fatalf("qwen-infra aliased the shipped qwen system %q", shipped)
	}
	if plan.System != managementpi.New(&fakeManagementStatusReader{}).ID() {
		t.Fatalf("plan.System = %q, want the Pi system", plan.System)
	}
	if localmodels.VendorID == "qwen" || localmodels.VendorID == "alibaba" {
		t.Fatalf("plugin-plane vendor identity drifted to %q", localmodels.VendorID)
	}
	if filepath.Base(plan.Binary) == string(shipped) || filepath.Base(plan.Binary) == "qwen" {
		t.Fatalf("qwen-infra resolved the qwen executable: %q", plan.Binary)
	}
}

// AC6, behavioural half. Renaming every identity the canonical target carries
// must change the plan only in the renamed positions. A literal branch on any
// Pi/Qwen/MLX/model identity would break this equivalence.
func TestPlanIsMetamorphicUnderIdentityRenaming(t *testing.T) {
	build := func(t *testing.T, runtime, model, profileName string) (agentic.Plan, vendorplugin.EngineObservationQuery) {
		t.Helper()
		dir, env := fakeProcessA(t, "exit 0\n")
		var seen vendorplugin.EngineObservationQuery
		reader := SanitizedEngineObservationReaderFunc(func(_ context.Context, query vendorplugin.EngineObservationQuery) (SanitizedEngineObservation, error) {
			seen = query
			now := time.Now()
			return SanitizedEngineObservation{
				Contract: SanitizedEngineObservationContract, SchemaVersion: SanitizedEngineObservationSchemaVersion,
				Engine: query.Engine, Runtime: query.Runtime, Model: query.Model, Profile: query.Profile,
				ObservedAt: now.Add(-time.Second), ValidUntil: now.Add(time.Minute), Facts: goodSanitizedEngineFacts(),
			}, nil
		})
		resolved := ResolvedCanonicalTarget{
			Target:  ProjectTarget{Name: runtime, Vendor: "product-label", Environment: "pi", Model: model, Profile: &profileName},
			Profile: &PiProfile{Provider: "local", Publisher: "neutral-publisher", Family: "neutral-family", Model: model, ContextWindow: 4096},
		}
		graph, err := BuildPiPluginGraph("/repo", resolved, &fakeManagementStatusReader{}, reader)
		if err != nil {
			t.Fatalf("BuildPiPluginGraph(%s): %v", runtime, err)
		}
		plan, err := vendorplugin.BuildLaunch(context.Background(), graph.Registry, graph.SpawnRequest([]byte("prompt"), dir, env), agentic.LaunchModeExec)
		if err != nil {
			t.Fatalf("BuildLaunch(%s): %v", runtime, err)
		}
		return plan, seen
	}

	original, originalQuery := build(t, "qwen-infra", "qwen-3.8-27b-mlx-8bit", "local-qwen")
	renamed, renamedQuery := build(t, "neutral-infra", "neutral-model-42", "neutral-profile")

	if len(original.Argv) != len(renamed.Argv) {
		t.Fatalf("argv shape changed under renaming: %#v vs %#v", original.Argv, renamed.Argv)
	}
	for index := range original.Argv {
		if index == 3 {
			if original.Argv[index] != "local-qwen" || renamed.Argv[index] != "neutral-profile" {
				t.Fatalf("profile slot = %q/%q", original.Argv[index], renamed.Argv[index])
			}
			continue
		}
		if original.Argv[index] != renamed.Argv[index] {
			t.Fatalf("argv[%d] differs under renaming: %q vs %q", index, original.Argv[index], renamed.Argv[index])
		}
	}
	if original.System != renamed.System {
		t.Fatalf("system dispatch changed under renaming: %q vs %q", original.System, renamed.System)
	}
	if originalQuery.Engine != renamedQuery.Engine {
		t.Fatalf("engine selection changed under renaming: %#v vs %#v", originalQuery.Engine, renamedQuery.Engine)
	}
	if string(renamedQuery.Runtime) != "neutral-infra" || string(renamedQuery.Model) != "neutral-model-42" || renamedQuery.Profile != "neutral-profile" {
		t.Fatalf("observation query did not carry the renamed identity: %#v", renamedQuery)
	}
}
