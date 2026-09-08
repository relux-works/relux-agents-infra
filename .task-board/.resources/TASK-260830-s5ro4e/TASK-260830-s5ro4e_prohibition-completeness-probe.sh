#!/usr/bin/env bash
# TASK-260830-s5ro4e review (document revision 9) -- is the ConnectOrStart-class
# prohibition, which is revision 9's entire mitigation for the downgrade
# direction, completely and correctly enumerated?
# Read-only. No process signalled, no installer run, no board mutated.
set -uo pipefail
PM=${PM:-/Users/alexis/src/relux-works/skill-project-management}
CLI="$PM/tools/board-cli"
PLAN=${PLAN:?}
pass=0; fail=0
ck(){ if [ "$2" = "$3" ]; then pass=$((pass+1)); printf 'PASS  %s (%s)\n' "$1" "$2";
      else fail=$((fail+1)); printf 'FAIL  %s -- expected [%s] got [%s]\n' "$1" "$3" "$2"; fi; }

echo "=== production source pin ==="
echo "skill-project-management HEAD = $(git -C "$PM" rev-parse HEAD)"
echo

echo "=== A. the real ConnectOrStart-class command set ==="
ck "A1 connectOrStartSessionManager call sites (excl. its own def)" \
   "$(grep -rn 'connectOrStartSessionManager(' --include='*.go' "$CLI" | grep -v _test.go | grep -vc 'func connectOrStartSessionManager')" "7"
ck "A2 session.go:151 belongs to sessionDoctorCmd" \
   "$(awk 'NR>=143 && NR<=152 && /Use:/{print $2}' "$CLI/cmd/session.go" | tr -d '",')" "doctor"
ck "A3 session.go:199 belongs to sessionReclaimContextsCmd" \
   "$(awk 'NR>=172 && NR<=200 && /Use:/{print $2}' "$CLI/cmd/session.go" | tr -d '",')" "reclaim-contexts"
ck "A4 session.go:245 belongs to sessionContextCompletenessCmd" \
   "$(awk 'NR>=226 && NR<=246 && /Use:/{print $2}' "$CLI/cmd/session.go" | tr -d '",')" "context-completeness"
ck "A5 session_context_economics.go declares 'context-economics'" \
   "$(grep -c 'Use:   "context-economics"' "$CLI/cmd/session_context_economics.go")" "1"
ck "A6 ... and reaches connectOrStartSessionManager" \
   "$(grep -c 'connectOrStartSessionManager(' "$CLI/cmd/session_context_economics.go")" "1"
ck "A7 session_context_retention.go declares 'sweep-contexts'" \
   "$(grep -c 'Use:   "sweep-contexts"' "$CLI/cmd/session_context_retention.go")" "1"
ck "A8 ... and reaches connectOrStartSessionManager" \
   "$(grep -c 'connectOrStartSessionManager(' "$CLI/cmd/session_context_retention.go")" "1"
ck "A9 both are registered under sessionCmd (they are 'session <x>')" \
   "$(grep -ch 'sessionCmd.AddCommand' "$CLI/cmd/session_context_economics.go" "$CLI/cmd/session_context_retention.go" | paste -sd+ - | bc)" "2"

echo
echo "=== B. 'session logs' is NOT ConnectOrStart-class -- it is not even a Dial ==="
ck "B1 sessionLogsCmd body contains no connectOrStartSessionManager" \
   "$(awk '/^var sessionLogsCmd/,/^}/' "$CLI/cmd/session.go" | grep -c 'connectOrStartSessionManager')" "0"
ck "B2 ... and no sessionmanager Dial either" \
   "$(awk '/^var sessionLogsCmd/,/^}/' "$CLI/cmd/session.go" | grep -cE '\bDial\(')" "0"
ck "B3 it streams layout.OperatorLogFile from disk" \
   "$(awk '/^var sessionLogsCmd/,/^}/' "$CLI/cmd/session.go" | grep -c 'OperatorLogFile')" "1"

echo
echo "=== C. a spawn does not have to be goal-bound to reach ConnectOrStart ==="
ck "C1 requiresManagedSpawnSession is true on ContextCtxID alone (|| not &&)" \
   "$(awk '/^func requiresManagedSpawnSession/,/^}/' "$CLI/cmd/codex_goal_spawn.go" | grep -c 'cfg.LaunchGoal != nil || strings.TrimSpace(cfg.ContextCtxID) != ""')" "1"
ck "C2 spawn.go materializes a context whenever contextPlan.Active" \
   "$(grep -c 'if contextPlan.Active {' "$CLI/cmd/spawn.go")" "1"
ck "C3 that path reaches connectOrStartSessionManager via spawn_context_lifecycle.go" \
   "$(grep -c 'connectOrStartSessionManager(parent, layout)' "$CLI/cmd/spawn_context_lifecycle.go")" "1"

echo
echo "=== D. what the plan's operator-facing prohibition actually names ==="
for t in "reclaim-contexts" "session doctor" "context-completeness" "context-economics" "sweep-contexts"; do
  printf 'INFO  plan mentions of %-22s : %s\n' "$t" "$(grep -c -- "$t" "$PLAN")"
done
ck "D1 plan never names 'context-completeness'"  "$(grep -c -- 'context-completeness' "$PLAN")" "0"
ck "D2 plan never names 'context-economics'"     "$(grep -c -- 'context-economics'    "$PLAN")" "0"
ck "D3 plan never names 'sweep-contexts'"        "$(grep -c -- 'sweep-contexts'       "$PLAN")" "0"
ck "D4 NARROWING CONTROL: it does name 'reclaim-contexts', so D1-D3 are gaps not grep artefacts" \
   "$(test "$(grep -c -- 'reclaim-contexts' "$PLAN")" -ge 5 && echo yes || echo no)" "yes"
ck "D5 the Part I call-site table mislabels session.go:151,199,245 as (logs, doctor, reclaim-contexts)" \
   "$(grep -c '`cmd/session.go:151,199,245` (`logs`, `doctor`, `reclaim-contexts`)' "$PLAN")" "1"
ck "D6 the prohibition names a non-existent command \`session context\` (real name: context-completeness)" \
   "$(grep -c '\`session context\`' "$PLAN")" "3"
ck "D7 the plan claims Dial-based session status reads are the ONLY safe ones" \
   "$(grep -c 'are the only reads that\|are the only thing that is' "$PLAN")" "2"
ck "D8 all prohibition sentences say goal-bound \`spawn\` only" \
   "$(grep -c 'goal-bound `spawn`' "$PLAN")" "3"
ck "D9 NARROWING CONTROL: the census sentences DO say 'or writable-context' -- so D8 is the prohibition narrowing, not the plan being unaware" \
   "$(grep -c 'goal-bound or writable-context\|goal-bound and writable-context' "$PLAN")" "3"

echo
echo "=== E. narrowing control: the plan's own census DID see the two files ==="
ck "E1 Part I table cites cmd/session_context_economics.go:58" \
   "$(grep -c 'cmd/session_context_economics.go:58' "$PLAN")" "1"
ck "E2 Part I table cites cmd/session_context_retention.go:50" \
   "$(grep -c 'cmd/session_context_retention.go:50' "$PLAN")" "1"
echo "     -> the file:line census is complete; the operator instruction derived from it is not."

echo
printf 'pass=%d fail=%d\n' "$pass" "$fail"
test "$fail" -eq 0
