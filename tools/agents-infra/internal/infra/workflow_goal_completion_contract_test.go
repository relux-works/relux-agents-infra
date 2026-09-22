package infra

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var primaryGoalCompletionClauses = []string{
	"task_board_run_id is absent",
	"task-board goal get",
	"every requirement",
	"persisted delivery evidence",
	"required review",
	"producer-only to-review",
	"running child",
	"pending review/check/landing",
	"blocked work",
	"human wait",
	"interrupted session",
	"native completion banner",
	"does not qualify and issues no clear",
	"same completion turn",
	"task-board goal clear --if-revision n",
	"fresh task-board goal get",
	"expected cleared successor",
	"predecessor",
	"read error is not absence",
	"cas conflict",
	"whole changed objective",
	"duplicate successor",
	"native condition completion",
	"interruption",
	"cancellation",
	"permitted in addition to the at-most-one requirement-actualization write per user turn",
	"session exit alone still never clears a goal",
}

func normalizedWorkflowContract(text string) string {
	text = strings.NewReplacer("`", "", "\n", " ").Replace(strings.ToLower(text))
	return strings.Join(strings.Fields(text), " ")
}

func primaryGoalCompletionContractFailure(text string) string {
	normalized := normalizedWorkflowContract(text)
	for _, clause := range primaryGoalCompletionClauses {
		if !strings.Contains(normalized, clause) {
			return clause
		}
	}
	return ""
}

func workflowInstructionsPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	return filepath.Clean(filepath.Join(
		filepath.Dir(thisFile), "..", "..", "..", "..", ".instructions", "INSTRUCTIONS_WORKFLOW.md",
	))
}

// This proves the source installed for both primary providers contains every
// instruction-only completion clause; it does not claim task-board state is
// stored or interpreted by agents-infra.
func TestWorkflowInstructionsCarryPrimaryGoalCompletionTurnContract(t *testing.T) {
	data, err := os.ReadFile(workflowInstructionsPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if missing := primaryGoalCompletionContractFailure(string(data)); missing != "" {
		t.Fatalf("workflow instructions missing primary-goal completion clause %q", missing)
	}
	t.Logf("primary-goal completion instruction coverage: %d/%d clauses", len(primaryGoalCompletionClauses), len(primaryGoalCompletionClauses))
}

// These controls narrow one refusal and reverse failure/absence while keeping
// the same words. Both must redden the source-text gate.
func TestWorkflowInstructionsRejectNarrowedPrimaryGoalCompletionContracts(t *testing.T) {
	data, err := os.ReadFile(workflowInstructionsPath(t))
	if err != nil {
		t.Fatal(err)
	}
	baseline := normalizedWorkflowContract(string(data))
	for _, mutant := range []struct {
		name string
		from string
		to   string
	}{
		{name: "running child admitted", from: "running child", to: "active offspring"},
		{name: "read failure laundered as absence", from: "read error is not absence", to: "absence is not read error"},
	} {
		t.Run(mutant.name, func(t *testing.T) {
			mutated := strings.Replace(baseline, mutant.from, mutant.to, 1)
			if mutated == baseline {
				t.Fatal("mutant did not reach workflow instructions")
			}
			if missing := primaryGoalCompletionContractFailure(mutated); missing == "" {
				t.Fatal("completion contract gate admitted narrowing mutant")
			}
		})
	}
}
