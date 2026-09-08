#!/bin/zsh
# Narrowing-mutant gate. Each mutant admits exactly ONE class the guard must
# reject; a mutant is proved only when the named test exits non-zero.
set -u
ROOT="$(cd "$(dirname "$0")/../../tools/agents-infra" && pwd)"
cd "$ROOT" || exit 99
LOG="$ROOT/../../.temp/TASK-260830-y6infr/mutants.log"
: > "$LOG"

BACKUP="$(mktemp -d)"
restore() {
  for f in "$@"; do
    if [ -f "$BACKUP/$(basename $f)" ]; then cp "$BACKUP/$(basename $f)" "$f"; else rm -f "$f"; fi
  done
}
save() { for f in "$@"; do [ -f "$f" ] && cp "$f" "$BACKUP/$(basename $f)"; done; }

run_mutant() {
  local name="$1"; shift
  local pkg="$1"; shift
  local test="$1"; shift
  go test "$pkg" -run "$test" -count=1 >/dev/null 2>&1
  local code=$?
  if [ $code -ne 0 ]; then
    echo "MUTANT_KILLED name=$name test=$test exit=$code" >> "$LOG"
  else
    echo "MUTANT_SURVIVED name=$name test=$test exit=$code  <<< GUARD DOES NOT BITE" >> "$LOG"
  fi
}

O=internal/infra/agents_management_observer.go
P=internal/infra/agents_management_process_a.go
R=internal/infra/agents_management_registry.go
T=internal/infra/pi_turn_result.go
S=internal/infra/pi_standalone.go
B=internal/infra/agents_management_boundary_test.go
LIVE=internal/infra/agents_management_observer_live.go
HELPER=internal/infra/agents_management_second_parser.go
save $O $P $R $T $S $B

# 1. Called-helper bypass: a NEW same-package file does the live read and is
#    called from ObserveEngine. A fixed-filename guard would not see it.
cat > $LIVE <<'GO'
package infra

import osalias "os"

func readLiveEngineDetail() string {
	if data, err := osalias.ReadFile(osalias.Getenv("HOME") + "/.agents/.configs/project-config.toml"); err == nil {
		return string(data)
	}
	return ""
}
GO
perl -0pi -e 's/\tsnapshot, err := a\.reader\.ReadSanitizedEngineObservation\(ctx, query\)/\t_ = readLiveEngineDetail()\n\tsnapshot, err := a.reader.ReadSanitizedEngineObservation(ctx, query)/' $O
run_mutant "observation-plane-called-helper-live-read" ./internal/infra TestObservationPlaneCannotReachLiveRuntime

# 2. Narrow the DISCOVERY (not the import check) to a fixed filename list while
#    mutant 1 is still applied. The guard must stop biting, proving that
#    package-complete plane discovery is what closes the bypass.
perl -0pi -e 's/\t\tif entry\.IsDir\(\) \|\| !strings\.HasSuffix\(name, "\.go"\) \|\| strings\.HasSuffix\(name, "_test\.go"\) \{/\t\tif entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "agents_management_observer_live.go" {/' $B
go test ./internal/infra -run TestObservationPlaneCannotReachLiveRuntime -count=1 >/dev/null 2>&1
code=$?
if [ $code -eq 0 ]; then
  echo "DISCOVERY_NARROWING_ADMITTED name=fixed-filename-discovery exit=0  (bypass admitted, as required to prove discovery carries the guard)" >> "$LOG"
else
  echo "DISCOVERY_NARROWING_UNEXPECTED exit=$code" >> "$LOG"
fi
restore $B
rm -f $LIVE
restore $O

# 3. Second schema-1 classifier on a reached helper path.
cat > $HELPER <<'GO'
package infra

import (
	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
)

func secondClassifierPath(input managementpi.TurnResultInput) (managementpi.TurnResult, error) {
	return managementpi.ValidateTurnResult(input)
}
GO
perl -0pi -e 's/\treturn managementpi\.ValidateTurnResult\(input\)/\tif input.StdoutTruncated {\n\t\treturn secondClassifierPath(input)\n\t}\n\treturn managementpi.ValidateTurnResult(input)/' $P
run_mutant "consumer-second-classifier-on-helper-path" ./internal/infra TestConsumerPlaneCannotParseAroundSoleClassifier
rm -f $HELPER
restore $P

# 4. Identity branch inserted into the generic consumer plane.
perl -0pi -e 's/\tif runner == nil \{/\tif request.Runtime == "qwen-infra" {\n\t\trunner = OSProcessATurnRunner{}\n\t}\n\tif runner == nil {/' $P
run_mutant "generic-plane-qwen-identity-branch" ./internal/infra TestGenericPlanesContainNoIdentityBranch
restore $P

# 5. Assembly dispatches on identity instead of only refusing on it.
perl -0pi -e 's/\tprofileName := \*resolved\.Target\.Profile/\tprofileName := *resolved.Target.Profile\n\tif resolved.Target.Vendor == "qwen" {\n\t\tprofileName = profileName + "-alibaba"\n\t}/' $R
run_mutant "assembly-identity-dispatch" ./internal/infra TestConcreteAssemblyDeclaresIdentityWithoutDispatchingOnIt
restore $R

# 6. Generic consumer plane becomes a Process-B broker reader.
perl -0pi -e 's/\tif runner == nil \{/\t_, _ = SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{})\n\tif runner == nil {/' $P
run_mutant "consumer-plane-reads-process-b-lifecycle" ./internal/infra TestProcessBLifecycleStaysOwnedByAgentsInfraAndOffTheGenericPlanes
restore $P

# 7. Narrow the schema-1 translator: admit tool failure as success.
perl -0pi -e 's/\tif toolFailed \{\n\t\treturn managementpi\.TurnCodeToolFailed, ""\n\t\}/\tif toolFailed \&\& finalText == "" {\n\t\treturn managementpi.TurnCodeToolFailed, ""\n\t}/' $T
run_mutant "translator-admits-tool-failure-as-success" ./internal/infra TestPiTurnTranslatorClassPrecedence
restore $T

# 8. Narrow the exact-profile assertion to case-insensitive matching.
perl -0pi -e 's/if request\.ExpectedProfile != "" \&\& request\.ExpectedProfile != selected \{/if request.ExpectedProfile != "" \&\& !strings.EqualFold(request.ExpectedProfile, selected) {/g' $S
run_mutant "profile-assertion-case-folds" . TestExactProfileAssertionRefusesEveryNonIdenticalProfile
restore $S

# 9. Narrow the consumer capture: report a clean exit for any signalled child.
perl -0pi -e 's/\t\tExit:            processAExit\(waitErr\),/\t\tExit:            managementpi.ProcessAExit{Code: 0},/' $P
run_mutant "consumer-reports-clean-exit-regardless" ./internal/infra TestConsumerRefusesExitDocumentDisagreementInsteadOfLaundering
restore $P

# 10. Narrow the adapter: clamp a stale validity interval into a live one.
perl -0pi -e 's/\t\tValidUntil:    snapshot\.ValidUntil,/\t\tValidUntil:    time.Now().Add(time.Minute),/' $O
run_mutant "adapter-clamps-stale-observation-live" ./internal/infra TestPiPluginGraphBuildLaunchRefusesForgedObservationBeforePreflight
restore $O

# 11. Narrow the identity gate: forward the query identity instead of the response's.
perl -0pi -e 's/\t\tEngine:        snapshot\.Engine,\n\t\tRuntime:       snapshot\.Runtime,\n\t\tModel:         snapshot\.Model,\n\t\tProfile:       snapshot\.Profile,/\t\tEngine:        query.Engine,\n\t\tRuntime:       query.Runtime,\n\t\tModel:         query.Model,\n\t\tProfile:       query.Profile,/' $O
run_mutant "adapter-launders-drifted-identity-into-query-identity" ./internal/infra TestPiPluginGraphBuildLaunchRefusesForgedObservationBeforePreflight
restore $O

# 12. Turn/message lifecycle (reviewer F1, CR-TASK-260830-y6infr-2 revision 2):
#     narrow the turn_start guard to admit a duplicate turn_start while the
#     first turn is still open.
perl -0pi -e 's/case "turn_start":\n\t\t\tif !sawAgentStart \|\| sawAgentEnd \|\| turnOpen \{\n\t\t\t\treturn "", false, errors\.New\("invalid Pi turn start"\)/case "turn_start":\n\t\t\tif !sawAgentStart || sawAgentEnd {\n\t\t\t\treturn "", false, errors.New("invalid Pi turn start")/' $T
run_mutant "lifecycle-admits-duplicate-turn-start" ./internal/infra "TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations/duplicate_turn_start"
restore $T

# 13. Drop the turnOpen requirement everywhere it gates a turn-scoped event
#     (message_start, message_update, message_end, turn_end all independently
#     require an open turn). Missing turn_start is admitted only when every
#     one of these redundant defense-in-depth checks is narrowed together;
#     narrowing any strict subset still leaves a neighboring check standing.
perl -0pi -e 's/ \|\| !turnOpen//g' $T
run_mutant "lifecycle-admits-message-events-without-open-turn" ./internal/infra "TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations/missing_turn_start"
restore $T

# 14. Drop the messageOpen requirement everywhere it gates a message-content
#     event (message_update and message_end both independently require an
#     open message). Missing message_start is admitted only when both are
#     narrowed together.
perl -0pi -e 's/ \|\| !messageOpen//g' $T
run_mutant "lifecycle-admits-message-content-without-open-message" ./internal/infra "TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations/missing_message_start"
restore $T

# 15. Narrow the turn_end guard to admit turn_end without an open turn
#     (duplicate turn_end), leaving its messageOpen check intact.
perl -0pi -e 's/case "turn_end":\n\t\t\tif !sawAgentStart \|\| sawAgentEnd \|\| !turnOpen \|\| messageOpen \{\n\t\t\t\treturn "", false, errors\.New\("invalid Pi turn end"\)/case "turn_end":\n\t\t\tif !sawAgentStart || sawAgentEnd || messageOpen {\n\t\t\t\treturn "", false, errors.New("invalid Pi turn end")/' $T
run_mutant "lifecycle-admits-turn-end-without-turn-start" ./internal/infra "TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations/turn_end_without_turn_start"
restore $T

# 16. Narrow the turn_end guard to admit turn_end while its message is still
#     open, leaving its turnOpen check intact. The dangling messageOpen=true
#     survives past agent_end, so the trailing structural EOF check (the same
#     defense-in-depth backstop mutant 13 relies on) is narrowed in the same
#     way to isolate this transition's own guard from that backstop.
perl -0pi -e 's/case "turn_end":\n\t\t\tif !sawAgentStart \|\| sawAgentEnd \|\| !turnOpen \|\| messageOpen \{\n\t\t\t\treturn "", false, errors\.New\("invalid Pi turn end"\)/case "turn_end":\n\t\t\tif !sawAgentStart || sawAgentEnd || !turnOpen {\n\t\t\t\treturn "", false, errors.New("invalid Pi turn end")/' $T
perl -0pi -e 's/ \|\| turnOpen \|\| messageOpen \|\| len\(openTools\)/ || turnOpen || len(openTools)/' $T
run_mutant "lifecycle-admits-turn-end-with-message-open" ./internal/infra "TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations/turn_end_while_message_open"
restore $T

# 17. Narrow the agent_end guard to admit agent_end while a turn is still
#     open, leaving turn_end's own guards untouched. The dangling
#     turnOpen=true survives to EOF, so the same trailing structural check is
#     narrowed in the same way to isolate this transition's own guard.
perl -0pi -e 's/case "agent_end":\n\t\t\tif !sawAgentStart \|\| sawAgentEnd \|\| turnOpen \{\n\t\t\t\treturn "", false, errors\.New\("invalid Pi agent end"\)/case "agent_end":\n\t\t\tif !sawAgentStart || sawAgentEnd {\n\t\t\t\treturn "", false, errors.New("invalid Pi agent end")/' $T
perl -0pi -e 's/ \|\| turnOpen \|\| messageOpen \|\| len\(openTools\)/ || messageOpen || len(openTools)/' $T
run_mutant "lifecycle-admits-agent-end-with-turn-open" ./internal/infra "TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations/agent_end_while_turn_open"
restore $T

# 18. Narrow the message_start guard to admit a duplicate message_start while
#     the first message is still open.
perl -0pi -e 's/case "message_start":\n\t\t\tif !sawAgentStart \|\| sawAgentEnd \|\| !turnOpen \|\| messageOpen \{\n\t\t\t\treturn "", false, errors\.New\("invalid Pi message start"\)/case "message_start":\n\t\t\tif !sawAgentStart || sawAgentEnd || !turnOpen {\n\t\t\t\treturn "", false, errors.New("invalid Pi message start")/' $T
run_mutant "lifecycle-admits-duplicate-message-start" ./internal/infra "TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations/duplicate_message_start"
restore $T

echo "--- restored tree ---" >> "$LOG"
git -C "$ROOT" status --short -- internal/infra >> "$LOG" 2>&1
cat "$LOG"
