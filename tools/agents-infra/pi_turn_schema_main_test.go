//go:build darwin && arm64

package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
)

type schemaOneDocument struct {
	Contract      string `json:"contract"`
	SchemaVersion int    `json:"schema_version"`
	Status        string `json:"status"`
	FinalText     string `json:"final_text"`
	Error         struct {
		Code managementpi.TurnResultCode `json:"code"`
	} `json:"error"`
}

func decodeSchemaOneRefusal(t *testing.T, output string, runErr error) schemaOneDocument {
	t.Helper()
	var failure *infra.PiTurnProcessAError
	if !errors.As(runErr, &failure) {
		t.Fatalf("schema-1 entry returned %#v, want a typed Process-A failure", runErr)
	}
	var document schemaOneDocument
	if err := json.Unmarshal([]byte(output), &document); err != nil {
		t.Fatalf("schema-1 stdout = %q: %v", output, err)
	}
	if document.Contract != managementpi.TurnResultContract || document.SchemaVersion != 1 || document.Status != "error" {
		t.Fatalf("schema-1 refusal envelope = %#v", document)
	}
	if strings.Count(strings.TrimSpace(output), "\n") != 0 {
		t.Fatalf("schema-1 refusal emitted more than one document: %q", output)
	}
	return document
}

// Production call site: runPi -> runPiStandaloneCLI -> runPiTurnSchema1CLI. The
// schema-1 result writer is installed the moment `--result-schema 1` is
// positively parsed, so every later outer-request refusal is still reported as
// exactly one bounded document with the exact closed code and never leaks the
// prompt or the supplied profile.
func TestSchemaOneOuterRequestRefusalsAreExactAndSanitized(t *testing.T) {
	secret := "secret prompt bytes"
	tests := []struct {
		name string
		args []string
		code managementpi.TurnResultCode
		exit int
	}{
		{"trailing operand", []string{"spawn", "--profile", "p", "--prompt", secret, "--deadline", "30m", "--result-schema", "1", "leftover"}, managementpi.TurnCodeRequestInvalid, 1},
		{"deadline over the bound", []string{"spawn", "--profile", "p", "--prompt", secret, "--deadline", "30m1ns", "--result-schema", "1"}, managementpi.TurnCodeRequestInvalid, 1},
		{"deadline at zero", []string{"spawn", "--profile", "p", "--prompt", secret, "--deadline", "0", "--result-schema", "1"}, managementpi.TurnCodeRequestInvalid, 1},
		{"unknown inner flag", []string{"spawn", "--profile", "p", "--prompt", secret, "--tools", "bash", "--result-schema", "1"}, managementpi.TurnCodeRequestInvalid, 1},
		{"repeated result schema", []string{"spawn", "--profile", "p", "--prompt", secret, "--result-schema", "1", "--result-schema", "1"}, managementpi.TurnCodeRequestInvalid, 1},
		{"prompt swallows the selector", []string{"spawn", "--profile", "p", "--prompt", "--result-schema=1"}, managementpi.TurnCodeRequestInvalid, 1},
		{"missing profile", []string{"spawn", "--prompt", secret, "--result-schema", "1"}, managementpi.TurnCodeProfileMissing, 1},
		{"equals-form selector without profile", []string{"spawn", "--prompt", secret, "--result-schema=1"}, managementpi.TurnCodeProfileMissing, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var runErr error
			output := captureStdout(t, func() { runErr = runPi(test.args) })
			document := decodeSchemaOneRefusal(t, output, runErr)
			if document.Error.Code != test.code {
				t.Fatalf("schema-1 code = %q, want %q", document.Error.Code, test.code)
			}
			var failure *infra.PiTurnProcessAError
			errors.As(runErr, &failure)
			if failure.ExitCode() != test.exit {
				t.Fatalf("schema-1 exit = %d, want %d", failure.ExitCode(), test.exit)
			}
			if strings.Contains(output, secret) || strings.Contains(output, "\"p\"") {
				t.Fatalf("schema-1 refusal leaked caller input: %q", output)
			}
		})
	}
}

// Negative: a value other than exactly 1 is not a positive schema-1 selection.
// It must fall through to the unchanged legacy operator surface rather than
// silently producing a versioned document for a version nobody requested.
func TestSchemaOneSelectorRequiresExactlyVersionOne(t *testing.T) {
	for _, args := range [][]string{
		{"spawn", "--profile", "p", "--prompt", "x", "--result-schema", "2"},
		{"spawn", "--profile", "p", "--prompt", "x", "--result-schema=2"},
		{"spawn", "--profile", "p", "--prompt", "x", "--result-schema"},
	} {
		t.Run(strings.Join(args[4:], " "), func(t *testing.T) {
			if selectsPiTurnResultSchema1(args) {
				t.Fatalf("args %v were mistaken for a schema-1 selection", args)
			}
			var runErr error
			output := captureStdout(t, func() { runErr = runPi(args) })
			var turnFailure *infra.PiTurnProcessAError
			if errors.As(runErr, &turnFailure) {
				t.Fatalf("legacy surface emitted a schema-1 failure: %#v", turnFailure)
			}
			var failure *infra.PiStandaloneFailure
			if !errors.As(runErr, &failure) || failure.Code != "pi_standalone_cli_invalid" {
				t.Fatalf("legacy refusal = %#v", runErr)
			}
			if strings.Contains(output, managementpi.TurnResultContract) {
				t.Fatalf("legacy surface emitted a schema-1 document: %q", output)
			}
		})
	}
}

// Production call site: runPi -> runPiStandaloneCLI. `--profile` is an exact
// byte assertion over the runtime-resolved profile on both the schema-1 and the
// legacy surface. No aliasing, case folding, trimming, or fallback is admitted,
// and the refusal never echoes the supplied profile.
func TestExactProfileAssertionRefusesEveryNonIdenticalProfile(t *testing.T) {
	project, home := mainStandaloneQwenProject(t)
	t.Setenv(callerCWDEnv, project)
	t.Setenv("HOME", home)
	t.Setenv("PATH", mainTestOfficialPiAsset(t))

	resolved := captureStdout(t, func() {
		if err := runPi([]string{"spawn", "--prompt", "safe prompt", "--print-config"}); err != nil {
			t.Fatalf("baseline print-config: %v", err)
		}
	})
	var plan infra.PiStandaloneLaunchPlan
	decodeSingleJSONDocument(t, resolved, &plan)
	if plan.Profile.Value == nil || *plan.Profile.Value == "" {
		t.Fatalf("baseline plan carried no resolved profile: %#v", plan)
	}
	profile := *plan.Profile.Value

	t.Run("exact profile is accepted", func(t *testing.T) {
		if err := runPi([]string{"spawn", "--profile", profile, "--prompt", "safe prompt", "--print-config"}); err != nil {
			t.Fatalf("exact profile assertion refused: %v", err)
		}
	})

	near := []string{
		strings.ToUpper(profile),
		" " + profile,
		profile + " ",
		profile + "-2",
		strings.TrimSuffix(profile, profile[len(profile)-1:]),
	}
	for _, candidate := range near {
		if candidate == profile {
			continue
		}
		t.Run("legacy refuses "+candidate, func(t *testing.T) {
			err := runPi([]string{"spawn", "--profile", candidate, "--prompt", "safe prompt", "--print-config"})
			var failure *infra.PiStandaloneFailure
			if !errors.As(err, &failure) || failure.Code != "pi_profile_mismatch" {
				t.Fatalf("near-miss profile %q error = %#v", candidate, err)
			}
			const want = `{"contract":"agents-infra.pi-standalone-result","schema_version":1,"status":"error","error":{"code":"pi_profile_mismatch"}}`
			if err.Error() != want {
				t.Fatalf("legacy profile refusal = %s, want the exact sanitized document", err)
			}
		})
		t.Run("schema-1 refuses "+candidate, func(t *testing.T) {
			var runErr error
			output := captureStdout(t, func() {
				runErr = runPi([]string{"spawn", "--profile", candidate, "--prompt", "safe prompt", "--deadline", "30m", "--result-schema", "1"})
			})
			document := decodeSchemaOneRefusal(t, output, runErr)
			if document.Error.Code != managementpi.TurnCodeProfileMismatch {
				t.Fatalf("schema-1 near-miss code = %q, want %q", document.Error.Code, managementpi.TurnCodeProfileMismatch)
			}
			const want = `{"contract":"agents-infra.pi-turn-result","schema_version":1,"status":"error","error":{"code":"pi_turn_profile_mismatch"}}`
			if strings.TrimSpace(output) != want {
				t.Fatalf("schema-1 profile refusal = %q, want the exact sanitized document", output)
			}
		})
	}
}
