package main

import (
	"strings"
	"testing"
)

// runPiTurnCLI is the real production consumer/parent entry point wired at
// `agents-infra pi turn`. These cases only exercise argument validation,
// which must refuse before infra.ResolvePiPluginGraph or any launch effect.
func TestRunPiTurnCLIRefusesInvalidArgumentsBeforeAnyGraphResolution(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"--bogus"}},
		{"positional argument", []string{"--prompt", "hi", "extra"}},
		{"zero deadline", []string{"--prompt", "hi", "--deadline", "0s"}},
		{"negative deadline", []string{"--prompt", "hi", "--deadline", "-1m"}},
		{"deadline over the 30m ceiling", []string{"--prompt", "hi", "--deadline", "31m"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := runPiTurnCLI(test.args); err == nil {
				t.Fatal("invalid pi turn arguments were admitted")
			}
		})
	}
}

// The top-level `pi` dispatcher must route "turn" to the real production
// consumer entry point rather than to standalone Process-A spawn handling.
func TestRunPiDispatchesTurnSubcommandToPiTurnCLI(t *testing.T) {
	err := runPi([]string{"turn", "--deadline", "0s"})
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("runPi([turn ...]) = %v, want the pi turn deadline refusal", err)
	}
}
