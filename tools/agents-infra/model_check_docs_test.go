package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/relux-agents-infra/tools/agents-infra/internal/infra"
)

func TestBoundedModelCheckREADMEContractPinsSafetyAndExitSemantics(t *testing.T) {
	readme := normalizedOperatorDoc(t, filepath.Join(sourceRepoRoot(t), "README.md"))
	wants := []string{
		"#### Bounded model behavior checks",
		"agents-infra model-check --target ENTRYPOINT --prompt TEXT --output-dir DIR",
		"Both flags are repeatable.",
		"With no expectations, exit `0` proves only a clean, complete managed lifecycle, not a particular model behavior.",
		"The checker creates or secures `DIR` as `0700` and writes four new regular files as `0600`.",
		"It refuses to overwrite any of these names",
		"| `events.jsonl` | Raw Pi JSONL provider/tool event stream",
		"| `stderr.log` | Raw managed runtime and Pi stderr",
		"| `summary.json` | Schema-1 machine-readable outcome",
		"| `summary.txt` | Deterministic human-readable rendering",
		fmt.Sprintf("the final response is capped at %d bytes", infra.ModelCheckFinalResponseBytes),
		"`--expect-tool read` proves that an exact-name `read` tool event occurred; it does not prove which file was read.",
		"completed, non-error `read` of the installed `relux-agents-infra/SKILL.md`",
		"A response marker alone is self-reported evidence, not proof of the tool target.",
		"A failed, partial, or malformed read is not legitimate absence",
		fmt.Sprintf(
			"The default managed-execution deadline is `%s`; accepted Go duration values are `%s` through `%s`.",
			modelCheckDocDuration(infra.DefaultModelCheckDeadline),
			modelCheckDocDuration(infra.MinimumModelCheckDeadline),
			modelCheckDocDuration(infra.MaximumModelCheckDeadline),
		),
		"runs bounded TERM-to-SIGKILL cleanup for its owned Pi and runtime process groups",
		"Evidence files remain for the operator; the checker does not delete them.",
		"`deadline_ms`, `duration_ms`, `timed_out`, both process-group cleanup states, and `cleanup_confirmed`.",
		fmt.Sprintf("| `%d` | Complete valid stream, managed cleanup confirmed, no failed tools, and every supplied expectation met. |", infra.ModelCheckExitSuccess),
		fmt.Sprintf("| `%d` | Target/launch/validation/assistant/managed-cleanup failure. Early option validation may fail before summary artifacts exist. |", infra.ModelCheckExitExecutionFailed),
		fmt.Sprintf("| `%d` | Managed-execution deadline expired. The summary separately reports whether cleanup was confirmed. |", infra.ModelCheckExitTimeout),
		fmt.Sprintf("| `%d` | Provider JSONL is malformed or an otherwise successful process produced an incomplete agent lifecycle. |", infra.ModelCheckExitMalformedStream),
		fmt.Sprintf("| `%d` | One or more `--expect-tool` or `--expect-text` assertions were not observed. |", infra.ModelCheckExitExpectationFailed),
		fmt.Sprintf("| `%d` | A tool execution completed with `isError=true`; this takes precedence over unmet expectations. |", infra.ModelCheckExitToolFailure),
		"Because the checker always supplies Pi `--approve`, its tool calls execute unattended inside the caller's project",
	}
	requireModelCheckDocFragments(t, "README.md", readme, wants)
}

func TestReluxAgentsInfraSkillPinsBoundedModelCheckerWorkflow(t *testing.T) {
	skill := normalizedOperatorDoc(t, filepath.Join(sourceRepoRoot(t), "SKILL.md"))
	wants := []string{
		"### Bounded model checker",
		"deployment smokes, tool-use checks, and skill-discovery checks",
		"agents-infra setup global --source-dir /path/to/relux-agents-infra",
		"agents-infra verify global",
		"qwen-infra --print-config",
		"--target qwen-infra",
		"--output-dir .temp/model-check/qwen-skill-discovery-01",
		"Always choose a fresh task-scoped output directory because existing evidence is never overwritten",
		"forces JSON print/no-session mode and Pi `--approve`, so tool calls execute unattended in the caller's project",
		fmt.Sprintf(
			"The default deadline is `%s`, with an accepted Go duration range of `%s..%s`.",
			modelCheckDocDuration(infra.DefaultModelCheckDeadline),
			modelCheckDocDuration(infra.MinimumModelCheckDeadline),
			modelCheckDocDuration(infra.MaximumModelCheckDeadline),
		),
		fmt.Sprintf("it is exit `%d`", infra.ModelCheckExitTimeout),
		fmt.Sprintf(
			"Other exits are `%d` pass, `%d` execution/validation/cleanup failure, `%d` malformed/incomplete JSONL, `%d` unmet expectations, and `%d` failed tool execution.",
			infra.ModelCheckExitSuccess,
			infra.ModelCheckExitExecutionFailed,
			infra.ModelCheckExitMalformedStream,
			infra.ModelCheckExitExpectationFailed,
			infra.ModelCheckExitToolFailure,
		),
		"[bounded-check contract](README.md#bounded-model-behavior-checks)",
		"`events.jsonl` and `stderr.log` as sensitive raw mode-`0600` evidence",
		"sanitized `summary.json` and `summary.txt`",
		"A `read` expectation proves only that a read tool ran",
		"completed, non-error read whose argument resolves to the installed `relux-agents-infra/SKILL.md`",
		"A response marker alone is self-reported evidence",
		"failed/partial read is failure or unknown, never absence",
	}
	requireModelCheckDocFragments(t, "SKILL.md", skill, wants)
}

func requireModelCheckDocFragments(t *testing.T, name, document string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(document, want) {
			t.Errorf("%s missing bounded model-check contract fragment %q", name, want)
		}
	}
}

func modelCheckDocDuration(value time.Duration) string {
	if value%time.Minute == 0 {
		return fmt.Sprintf("%dm", value/time.Minute)
	}
	return value.String()
}
