#!/usr/bin/env bash
# TASK-260830-s5ro4e revision 12 validation. Drives text EXTRACTED FROM THE SHIPPED DOCUMENTS,
# never a hand-typed copy. Signals no real process: pgrep is stubbed and every daemon stopped
# here is a fake this script spawned. Reads no credential, token or environment value.
cd "$(git rev-parse --show-toplevel)"
PLAN=.research/260830_agents-management-lockstep-release-and-rollback.md
RB=.research/260830_agents-management-lockstep-release-and-rollback_runbook.md
W="$(mktemp -d)"; trap 'rm -rf "$W"' EXIT
fails=0
ck(){ if [ "$2" = "$3" ]; then printf 'PASS  %-56s %s\n' "$1" "$2"
      else printf 'FAIL  %-56s got=%s want=%s\n' "$1" "$2" "$3"; fails=$((fails+1)); fi; }

echo "=== 0. Extraction: the probes drive the shipped text, verbatim ==="
python3 - "$W" "$PLAN" "$RB" <<'PY'
import sys,re
W,plan,rb=sys.argv[1:4]
p=open(plan).read(); r=open(rb).read()
sec=p[p.index("## P2 —"):p.index("## P3 —")]
f=re.search(r"```bash\n(.*?)\n```",sec,re.S).group(1)
h=f[:f.index("# Steps 2 and 5")]
open(W+"/p2-tb.sh","w").write(h+f[f.index("# Steps 2 and 5"):f.index("# Steps 3 and 6")])
open(W+"/p2-ai.sh","w").write(h+f[f.index("# Steps 3 and 6"):])
s=r[r.index("## RB-3"):r.index("## RB-4")]
g=re.search(r"```bash\n(.*?)\n```",s,re.S).group(1)
i=g.index("# --- 0d."); j=g.index('done < "$RECOVERY_ROOT/daemon-boards-before-r1.txt"')+len('done < "$RECOVERY_ROOT/daemon-boards-before-r1.txt"')
open(W+"/r1-0d.sh","w").write("set -euo pipefail\n"+g[i:j]+"\n")
print("p2-fence-verbatim=%s r1-0d-verbatim=%s" % (f in p, g[i:j] in r))
PY
ck "P2 fence + 0d slice are verbatim substrings" \
   "$(python3 -c "
plan=open('$PLAN').read(); rb=open('$RB').read()
tb=open('$W/p2-tb.sh').read(); zd=open('$W/r1-0d.sh').read().split(chr(10),1)[1]
print('yes' if (tb[tb.index('# Steps 2'):] in plan and zd.rstrip(chr(10)) in rb) else 'no')")" yes
cp "$W/r1-0d.sh" "$W/r1-0d.orig.sh"

echo
echo "=== 1. F1 — P2, shipped fence, both shells, real installer text ==="
TB=/Users/alexis/src/relux-works/skill-project-management
AI=/Users/alexis/src/relux-works/relux-agents-infra
mkdir -p "$W/rr" "$W/tb-old/scripts/lib" "$W/ai-old/scripts"
cp "$TB/scripts/setup.sh" "$W/tb-old/scripts/"; cp "$TB/scripts/lib/agents-infra-compose.zsh" "$W/tb-old/scripts/lib/"
cp "$AI/scripts/setup.sh" "$W/ai-old/scripts/"
cp -R "$W/tb-old" "$W/tb-new"; cp -R "$W/ai-old" "$W/ai-new"
printf '\n: ${BRAND_NEW_COMPOSE_OVERRIDE:=y}\n' >> "$W/tb-new/scripts/lib/agents-infra-compose.zsh"
printf '\nNEW_UNDECIDED_INPUT="${SOME_BRAND_NEW_OVERRIDE:-x}"\n' >> "$W/ai-new/scripts/setup.sh"
p2(){ RECOVERY_ROOT="$W/rr" SAVED_SOURCE="$2" RELEASE="$3" "$1" "$4" 2>&1; }
sig(){ printf '%s' "$1" | grep -c 'P2-SET-OK' ; }
for sh in bash zsh; do
  o="$(p2 $sh "$W/tb-old" "$W/tb-old" "$W/p2-tb.sh")"; ck "[$sh] tb clean pair -> sigil"  "$(printf '%s' "$o"|tr -s ' '|head -1)" "P2-SET-OK 38"
  o="$(p2 $sh "$W/tb-old" "$W/tb-new" "$W/p2-tb.sh")"; ck "[$sh] tb override in sourced file -> no sigil" "$(sig "$o")" 0
  ck "[$sh] tb  ... and the new name is reported" "$(printf '%s' "$o"|grep -c BRAND_NEW_COMPOSE_OVERRIDE)" 1
  o="$(p2 $sh "$W/ai-old" "$W/ai-old" "$W/p2-ai.sh")"; ck "[$sh] ai clean pair -> sigil"  "$(printf '%s' "$o"|tr -s ' '|head -1)" "P2-SET-OK 19"
  o="$(p2 $sh "$W/ai-old" "$W/ai-new" "$W/p2-ai.sh")"; ck "[$sh] ai injected override -> no sigil" "$(sig "$o")" 0
  ck "[$sh] ai  ... and the new name is reported" "$(printf '%s' "$o"|grep -c SOME_BRAND_NEW_OVERRIDE)" 1
  o="$(p2 $sh "$W/ai-old" "$W/ai-new" "$W/p2-tb.sh")"; ck "[$sh] F1 shape: named operand absent -> refusal" "$(printf '%s' "$o"|grep -c 'STOP P2 unreadable operand')" 1
  ck "[$sh] F1 shape:  ... and no sigil" "$(sig "$o")" 0
done
echo "  (revision 11's glob form, for contrast, under zsh:)"
cat > "$W/old-p2.sh" <<'EOS'
set_of() { grep -ohE '\$\{?[A-Z][A-Z0-9_]*' "$@" | sed -E 's/^\$\{?//' | sort -u; }
diff <(set_of "$SAVED_SOURCE/scripts/setup.sh" "$SAVED_SOURCE/scripts/lib/"*.zsh) \
     <(set_of "$RELEASE/scripts/setup.sh"      "$RELEASE/scripts/lib/"*.zsh); echo "diff-exit=$?"
EOS
o="$(p2 zsh "$W/ai-old" "$W/ai-new" "$W/old-p2.sh")"
ck "MUTANT rev-11 glob under zsh reports the override" "$(printf '%s' "$o"|grep -c SOME_BRAND_NEW_OVERRIDE)" 0
ck "MUTANT rev-11 glob under zsh yields diff-exit=0"    "$(printf '%s' "$o"|grep -c 'diff-exit=0')" 1

echo
echo "=== 2. F2 — R1 step 0d, shipped slice, fake daemon, stubbed pgrep ==="
mkdir -p "$W/shim" "$W/fake"
printf '#!/bin/sh\nwhile :; do sleep 1; done\n' > "$W/fake/tb-sessiond"
printf '#!/bin/sh\n[ -s "$FAKE_PGREP" ] && cat "$FAKE_PGREP" && exit 0\nexit 1\n' > "$W/shim/pgrep"
cat > "$W/shim/task-board" <<'SHIM'
#!/bin/sh
case "$5" in
  status) if kill -0 "$FAKE_PID" 2>/dev/null; then cat "$FAKE_STATUS"
          else echo '{"running":false}'; fi ;;
  list)   cat "$FAKE_LIST" ;;
esac
SHIM
chmod +x "$W/fake/tb-sessiond" "$W/shim/pgrep" "$W/shim/task-board"
case0d(){ # $1 label $2 list-payload $3 status-template $4 want-fate $5 want-record
  local RR; RR="$(mktemp -d)"
  "$W/fake/tb-sessiond" --board-dir /b1 2>/dev/null & local fp=$!
  sleep 0.3
  printf '%s /bin/sh %s/fake/tb-sessiond --board-dir /b1\n' "$fp" "$W" > "$RR/pgrep.txt"
  printf '%s\n' "$2" > "$RR/list.json"; printf '%s\n' "${3//PID/$fp}" > "$RR/status.json"
  local msg; msg="$(RECOVERY_ROOT="$RR" FAKE_PID="$fp" FAKE_PGREP="$RR/pgrep.txt" FAKE_LIST="$RR/list.json" \
        FAKE_STATUS="$RR/status.json" PATH="$W/shim:$PATH" bash "$W/r1-0d.sh" 2>&1)"; local rc=$?
  local fate; if kill -0 "$fp" 2>/dev/null; then fate=alive; else fate=terminated; fi
  local rec; if [ -s "$RR/r1-daemons-stopped.txt" ]; then rec=recorded; else rec=none; fi
  ck "$1" "rc=$rc/$fate/$rec" "$4"
  printf '        %s\n' "$(printf '%s' "$msg"|grep -E '^(STOP|KeyError|ValueError|SystemExit)|Error:'|head -1)"
  kill -9 "$fp" 2>/dev/null; wait "$fp" 2>/dev/null; rm -rf "$RR"; }
S4='{"running":true,"pid":PID,"board_fingerprint":"fp","session_count":4,"quarantined_count":0}'
S1='{"running":true,"pid":PID,"board_fingerprint":"fp","session_count":1,"quarantined_count":0}'
Z0='{"sessions":[{"attached_clients":0},{"attached_clients":0},{"attached_clients":0},{"attached_clients":0}]}'
case0d "CONTROL 4 real zeros -> stops the daemon"      "$Z0" "$S4" "rc=0/terminated/recorded"
case0d "POSITIVE 2 attached clients -> refuses"        '{"sessions":[{"attached_clients":2},{"attached_clients":0},{"attached_clients":0},{"attached_clients":0}]}' "$S4" "rc=1/alive/none"
case0d "F2-a attached_clients key absent -> unknown"   '{"sessions":[{"session_id":"s1"},{"session_id":"s2"},{"session_id":"s3"},{"session_id":"s4"}]}' "$S4" "rc=1/alive/none"
case0d "F2-b no sessions key, items instead -> unknown" '{"items":[{"attached_clients":9}]}' "$S4" "rc=1/alive/none"
case0d "F2-c sessions null -> unknown"                 '{"sessions":null}' "$S4" "rc=1/alive/none"
case0d "F2-d 4 rows vs session_count 1 -> unknown"     "$Z0" "$S1" "rc=1/alive/none"
case0d "F2-e quarantined_count absent -> unknown"      '{"sessions":[{"attached_clients":0}]}' '{"running":true,"pid":PID,"board_fingerprint":"fp","session_count":1}' "rc=1/alive/none"
case0d "F2-f attached_clients is a string -> unknown"  '{"sessions":[{"attached_clients":"two"}]}' "$S1" "rc=1/alive/none"

echo
echo "=== 3. Mutants: prove the bound, do not only prove the gate exists ==="
python3 - "$W" <<'PY'
import sys; W=sys.argv[1]
s=open(W+"/r1-0d.orig.sh").read()
i=s.index('  att="$(python3 -c'); j=s.index("\n", s.index("is unknown, not zero", i))+1
open(W+"/r1-0d.sh","w").write(s[:i]+'''  att="$(python3 -c 'import json,sys
d=json.load(open(sys.argv[1])); rows=d if isinstance(d,list) else d.get("sessions") or []
print(sum(int(r.get("attached_clients") or 0) for r in rows))' "$lst")"
'''+s[j:])
PY
echo "  MUTANT-DELETE: revision 11's lenient .get(...) or 0 restored"
case0d "  M1 key absent -> admits a fabricated zero"   '{"sessions":[{"session_id":"s1"}]}' "$S4" "rc=0/terminated/recorded"
case0d "  M1 items instead -> admits a fabricated zero" '{"items":[{"attached_clients":9}]}' "$S4" "rc=0/terminated/recorded"
case0d "  M1 sessions null -> admits a fabricated zero" '{"sessions":null}' "$S4" "rc=0/terminated/recorded"
python3 - "$W" <<'PY'
import sys; W=sys.argv[1]
s=open(W+"/r1-0d.orig.sh").read(); i=s.index('if len(rows)!=int(sys.argv[2])'); j=s.index("\n",i)+1
open(W+"/r1-0d.sh","w").write(s[:i]+s[j:])
PY
echo "  MUTANT-NARROW: strict indexing kept, only the row-count cross-check removed"
case0d "  M2 key absent -> still caught by the strict index" '{"sessions":[{"session_id":"s1"}]}' "$S4" "rc=1/alive/none"
case0d "  M2 4 rows vs count 1 -> now admitted"              "$Z0" "$S1" "rc=0/terminated/recorded"
cp "$W/r1-0d.orig.sh" "$W/r1-0d.sh"

echo
echo "=== 4. Residuals and budget ==="
ck "no scripts/lib glob survives in either document" "$(grep -c 'lib/"\*\.zsh' $PLAN $RB | awk -F: '{s+=$2} END{print s}')" 0
ck "no release-set.txt, which nothing produced"      "$(grep -c 'release-set.txt' $PLAN | head -1)" 0
ck "no lenient attached_clients read survives"       "$(grep -c 'attached_clients") or 0' $RB)" 0
ck "Step 3 regained the agents-infra gate symmetry"  "$(grep -c 'agents-infra full tests/build/verify plus the real target/compose/Pi' $PLAN)" 1
ck "Step 5 regained env -u AGENTS_INFRA_SOURCE_DIR"  "$(grep -c 'env -u AGENTS_INFRA_SOURCE_DIR "\$RECOVERY_ROOT/bin/agents-infra" verify global` |' $PLAN)" 1
ck "Limit 7 present"                                 "$(grep -c 'cross-protocol .attached_clients. read cannot be trusted' $PLAN)" 1
ck "Part II names the shell it is run under"         "$(grep -c 'Run every Part II command under .bash' $PLAN)" 1
ck "plan is within the 800-line budget"              "$([ "$(grep -c '' $PLAN)" -le 800 ] && echo yes || echo no)" yes
printf '      plan=%s runbook=%s review-record=%s lines\n' "$(grep -c '' $PLAN)" "$(grep -c '' $RB)" "$(grep -c '' .research/260830_agents-management-lockstep-release-and-rollback_review-record.md)"

echo
if [ "$fails" -eq 0 ]; then echo "ALL CHECKS PASSED"; else echo "$fails CHECK(S) FAILED"; fi
exit "$fails"
