#!/usr/bin/env bash
# TASK-260830-s5ro4e review (revision 8 / CR revision 7)
# Question: is the "daemon newer than CLI" refusal that Part I, P7 and R1 step 8
# rest on reachable from any production call site?
# Read-only. No process signalled, no repository modified, no installer run.
set -uo pipefail
PM="${PM:-/Users/alexis/src/relux-works/skill-project-management}"
B="$PM/tools/board-cli"; SM="$B/internal/sessionmanager"
pass=0; fail=0
ck(){ if [ "$2" = "$3" ]; then pass=$((pass+1)); printf 'PASS  %s (%s)\n' "$1" "$2";
      else fail=$((fail+1)); printf 'FAIL  %s -- expected [%s] got [%s]\n' "$1" "$3" "$2"; fi; }

echo "=== source identity ==="
git -C "$PM" log --oneline -1

echo
echo "=== 1. the newer-daemon branch returns an UNTYPED error ==="
ck "N1 newer branch exists" \
  "$(sed -n '714,740p' "$SM/client.go" | grep -c 'is newer than this wrapper supports')" 1
ck "N2 and returns fmt.Errorf, not a ManagerProtocolUpgradeError" \
  "$(awk '/^func managerProtocolCompatibility/,/^}/' "$SM/client.go" \
     | awk '/actual > controlProtocolVersion/,/^	}/' | grep -c 'return fmt.Errorf')" 1
ck "N3 while the older branch returns the typed error" \
  "$(awk '/^func managerProtocolCompatibility/,/^}/' "$SM/client.go" \
     | awk '/actual < controlProtocolVersion/,/^	}/' | grep -c 'ManagerProtocolUpgradeError')" 1

echo
echo "=== 2. ConnectOrStart only branches on the TYPED error ==="
ck "N4 ConnectOrStart matches only ManagerProtocolUpgradeError" \
  "$(awk '/^func ConnectOrStart\(/,/^}/' "$SM/client.go" | grep -c 'errors.As(err, &upgrade)')" 1
ck "N5 no other errors.As / error inspection in ConnectOrStart" \
  "$(awk '/^func ConnectOrStart\(/,/^}/' "$SM/client.go" | grep -c 'errors\.\(As\|Is\)(')" 1
ck "N6 the else-block that observes the compat error returns nothing for the untyped case" \
  "$(awk '/^func ConnectOrStart\(/,/^}/' "$SM/client.go" \
     | awk '/} else {/,/^	}$/' | grep -c 'return nil, upgrade')" 2
ck "N7 the only launch==nil early return is ErrManagerNotRunning" \
  "$(awk '/^func ConnectOrStart\(/,/^}/' "$SM/client.go" \
     | grep -A1 'if launch == nil' | grep -c 'ErrManagerNotRunning')" 1
ck "N8 so control falls through to takeOverUnresponsiveManager" \
  "$(awk '/^func ConnectOrStart\(/,/^}/' "$SM/client.go" | grep -c 'takeOverUnresponsiveManager(ctx, layout)')" 2

echo
echo "=== 3. every production ConnectOrStart caller passes a NON-nil launch ==="
for f in cmd/session.go cmd/codex_manager.go cmd/claude_manager.go; do
  n=$(grep -A4 'sessionmanager.ConnectOrStart' "$B/$f" | grep -c 'func() error')
  ck "N9 $f passes launch" "$n" 1
done
ck "N10a sessionmanager.Connect (launch-less, compat-checked) CALL sites under cmd/" \
  "$(grep -rn 'sessionmanager\.Connect(' --include='*.go' "$B/cmd" | wc -l | tr -d ' ')" 0
ck "N10b it appears under cmd/ only as a passed function value" \
  "$(grep -rn 'sessionmanager\.Connect,' --include='*.go' "$B/cmd" | wc -l | tr -d ' ')" 1

echo
echo "=== 4. takeOverUnresponsiveManager performs NO compatibility check ==="
ck "N11 managerProtocolCompatibility call sites in production" \
  "$(grep -rn 'managerProtocolCompatibility(' --include='*.go' "$SM" | grep -v _test.go | grep -vc 'func managerProtocolCompatibility')" 2
ck "N12 neither is inside takeOverUnresponsiveManager" \
  "$(awk '/^func takeOverUnresponsiveManager/,/^}/' "$SM/stale_takeover.go" | grep -c 'managerProtocolCompatibility')" 0
ck "N13 it dials with dialManagerStatus (unchecked), not dialHealthyManager" \
  "$(awk '/^func takeOverUnresponsiveManager/,/^}/' "$SM/stale_takeover.go" | grep -c 'dialManagerStatus(ctx, layout)')" 1
ck "N14 dialManagerStatus contains no compatibility check" \
  "$(awk '/^func dialManagerStatus/,/^}/' "$SM/stale_takeover.go" | grep -c 'managerProtocolCompatibility')" 0
ck "N15 and a StartupHealthy holder is returned to the caller as a live client" \
  "$(awk '/^func takeOverUnresponsiveManager/,/^}/' "$SM/stale_takeover.go" \
     | grep -A2 'if status.StartupHealthy' | grep -c 'return client, nil, nil')" 1
ck "N16 while a non-StartupHealthy holder falls through to terminateStaleInstance" \
  "$(awk '/^func takeOverUnresponsiveManager/,/^}/' "$SM/stale_takeover.go" | grep -c 'terminateStaleInstance(ctx, layout, recorded)')" 1
ck "N17 terminateStaleInstance signals the process (graceful then forced)" \
  "$(awk '/^func terminateStaleInstance/,/^}/' "$SM/stale_takeover.go" | grep -c 'requestProcessTermination\|forceProcessTermination')" 2
ck "N18 ConnectOrStart returns that healthy client directly" \
  "$(awk '/^func ConnectOrStart\(/,/^}/' "$SM/client.go" | grep -c 'return healthyClient, nil')" 2

echo
echo "=== 5. the direction is untested ==="
ck "N19 test FILES naming a newer-than-wrapper daemon refusal" \
  "$(grep -rl 'is newer than this wrapper supports' "$SM"/*_test.go 2>/dev/null | wc -l | tr -d ' ')" 0
ck "N20 while the OLDER direction has a dedicated test" \
  "$(grep -c 'func TestConnectOrStartUpgradesHealthyOutdatedProtocolHolder' "$SM/stale_takeover_unix_test.go")" 1
ck "N21 and the not-StartupHealthy terminate path has one too" \
  "$(grep -c 'func TestConnectOrStartTakesOverHolderWithoutStartupHealthy' "$SM/stale_takeover_unix_test.go")" 1

echo
echo "=== 6. the plan does not name any of this ==="
PLAN="${PLAN:?set PLAN}"
for t in takeOverUnresponsiveManager unresponsive StartupHealthy startup_healthy 'launch == nil' dialManagerStatus; do
  ck "N22 plan mentions '$t'" "$(grep -c -- "$t" "$PLAN")" 0
done

echo
printf 'probe pass=%d fail=%d\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
