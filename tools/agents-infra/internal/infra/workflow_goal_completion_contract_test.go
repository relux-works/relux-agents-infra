package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	workflowCompletionContractStart = "<!-- primary-goal-completion-turn-contract:v1:start -->"
	workflowCompletionContractEnd   = "<!-- primary-goal-completion-turn-contract:v1:end -->"
	workflowCompletionContractText  = `
A primary parent with no TASK_BOARD_RUN_ID, before declaring the exact board
primary objective delivered, reads task-board goal get. It verifies every
requirement of that exact objective against persisted delivery evidence, and
required review, check, and landing obligations must already be satisfied.
Producer-only to-review, a running child, pending review/check/landing,
blocked remaining work, a human wait, an interrupted session, or a native
completion banner does not qualify and issues no clear.

On that same completion turn, it runs task-board goal clear --if-revision N
--reason "Objective delivered; <evidence refs>", then runs a separate fresh
task-board goal get. Readback must verify no active primary goal, the expected
cleared successor carrying the delivery-evidence reason and predecessor, and
the immutable previous revision.

The ledger is never described as cleared before successful readback. A read
error is not absence. Read, native-clear acknowledgement, storage, CAS, and
readback failures are surfaced truthfully with the active head and recovery
evidence preserved. On a CAS conflict, the parent rereads and reassesses the
whole changed objective instead of blindly clearing the new revision. If
another actor already cleared the same delivered objective, it verifies that
recorded transition and appends no duplicate successor.

Native condition completion, turn end, detach, stop, usage limit, interruption,
and cancellation never trigger this clear on their own. A primary-goal clear
accepts neither a board task nor a spawned-run goal.

This evidenced completion clear is permitted in addition to the existing
at-most-one-requirement-actualization-write-per-user-turn cap, including when
one turn creates or updates and then delivers the objective. A wording-only
restatement does not clear anything. Session exit alone still never clears a
goal.`
)

func normalizedWorkflowContract(text string) string {
	text = strings.ReplaceAll(text, "`", "")
	return strings.Join(strings.Fields(text), " ")
}

func workflowCompletionContractBlock(text string) (string, error) {
	if strings.Count(text, workflowCompletionContractStart) != 1 ||
		strings.Count(text, workflowCompletionContractEnd) != 1 {
		return "", fmt.Errorf("workflow instructions must carry exactly one completion-turn contract marker pair")
	}
	scopeAt := strings.Index(text, "## Primary Parent Goal Actualization")
	if scopeAt < 0 {
		return "", fmt.Errorf("workflow instructions are missing the Primary Parent Goal Actualization section")
	}
	scopeEnd := strings.Index(text[scopeAt:], "\n---")
	if scopeEnd < 0 {
		return "", fmt.Errorf("Primary Parent Goal Actualization section has no boundary")
	}
	scope := text[scopeAt : scopeAt+scopeEnd]
	start := strings.Index(scope, workflowCompletionContractStart)
	end := strings.Index(scope, workflowCompletionContractEnd)
	if start < 0 || end < 0 || end <= start {
		return "", fmt.Errorf("completion-turn contract markers are outside Primary Parent Goal Actualization")
	}
	return scope[start+len(workflowCompletionContractStart) : end], nil
}

func primaryGoalCompletionContractFailure(text string) string {
	block, err := workflowCompletionContractBlock(text)
	if err != nil {
		return err.Error()
	}
	if normalizedWorkflowContract(block) != normalizedWorkflowContract(workflowCompletionContractText) {
		return "the marked operative completion-turn block differs from v1"
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

// This proves the source workflow module contains the exact operative
// completion block in its Primary Parent Goal Actualization section; it does
// not claim that agents-infra setup installs the module or stores board state.
func TestWorkflowInstructionsCarryPrimaryGoalCompletionTurnContract(t *testing.T) {
	data, err := os.ReadFile(workflowInstructionsPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if failure := primaryGoalCompletionContractFailure(string(data)); failure != "" {
		t.Fatalf("workflow instructions violate the primary-goal completion contract: %s", failure)
	}
	t.Log("primary-goal completion instruction coverage: 1/1 exact operative section")
}

// This is the token-inventory-semantic-bypass regression from the first
// project-management review. The mutant keeps the old searched phrase inside
// the marked block but reverses the operative sentence, which must be rejected.
func TestWorkflowInstructionsRejectTokenPreservingCompletionContractReversals(t *testing.T) {
	data, err := os.ReadFile(workflowInstructionsPath(t))
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(data)
	for _, mutant := range []struct {
		name string
		from string
		to   string
	}{
		{
			name: "non-delivery qualifies while refusal phrase survives as legacy text",
			from: "does not qualify and issues no clear.",
			to:   "qualifies and issues a clear. The legacy phrase does not qualify and issues no clear is retained only for search compatibility.",
		},
	} {
		t.Run(mutant.name, func(t *testing.T) {
			mutated := strings.Replace(baseline, mutant.from, mutant.to, 1)
			if mutated == baseline {
				t.Fatal("mutant did not reach workflow instructions")
			}
			if failure := primaryGoalCompletionContractFailure(mutated); failure == "" {
				t.Fatal("completion contract gate admitted a token-preserving semantic reversal")
			}
		})
	}
}
