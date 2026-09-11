# Lockstep migration runbook — the executable form of Part III

Companion to `260830_agents-management-lockstep-release-and-rollback.md`
(`TASK-260830-s5ro4e`). The plan holds every decision, precondition, expected result,
window, refusal and limit; this file holds the four scripts the plan's Part II and Part III
name, so that the operating document stays readable under pressure and the thing you
execute is a saved script rather than shell pasted out of prose mid-incident.

**Every script here is UNTESTED against a real installed pair.** See the plan's **P8** and
*Limits*. Save each under `$RECOVERY_ROOT` and run it from there; do not edit a refusal out
of one to make a step proceed — each refusal has a matching entry in the plan's *Stop
conditions*.

| ID | Script | Used by |
| --- | --- | --- |
| RB-1 | `snapshot-brew.sh` | P6, before and after every agents-infra install |
| RB-2 | `snapshot.sh` | Step 0 (`current-pair`) and after Step 3 (`bridge-pair`) |
| RB-3 | `r1.sh` | rollback of Steps 2 and 5 |
| RB-4 | `r2.sh` | rollback of Steps 3 and 6 |

## RB-1 — `snapshot-brew.sh` (P6)

Invoked as `bash snapshot-brew.sh <destination>`. A **top-level script, not a function**:
under `set -euo pipefail` at script scope a failed read aborts it and publishes no file, so
a failed read can never become a measured absence. Never add `|| true`.

```bash
set -euo pipefail
out="$1"; test ! -e "$out"                  # never overwrite an earlier good snapshot
rm -f "$out.partial"; trap 'rm -f "$out.partial"' EXIT
{ brew --prefix; brew list --formula; brew list --versions llvm; brew --prefix llvm; } > "$out.partial"
prefix="$(brew --prefix)"; llvm_prefix="$(brew --prefix llvm)"
for t in "$prefix/bin/lldb-mcp" "$prefix/bin/lldb-mcp.agents-infra.bak" \
         "$llvm_prefix/bin/lldb-mcp" "$llvm_prefix/bin/lldb"; do
  if [ -e "$t" ] || [ -L "$t" ]; then stat -f '%N|%HT|%z|%m|%Sp' "$t"; shasum -a 256 "$t"
  else printf 'ABSENT|%s\n' "$t"; fi
done >> "$out.partial"
mv -f "$out.partial" "$out"
```

## RB-2 — `snapshot.sh` (Part III snapshot)

```bash
set -euo pipefail
PM_STATE="$HOME/.config/task-board/install.json"
AI_STATE="$HOME/Library/Application Support/agents-infra/install.json"   # Darwin; else
# "${XDG_CONFIG_HOME:-$HOME/.config}/agents-infra/install.json" per agents-infra setup.sh:64-73.
j() { python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))[sys.argv[2]])' "$1" "$2"; }
PM_REPO="$(j "$PM_STATE" repoPath)"; AI_REPO="$(j "$AI_STATE" repoPath)"
test "$(j "$PM_STATE" binDir)" = "$(j "$AI_STATE" binDir)"; BIN_DIR="$(j "$PM_STATE" binDir)"
test -d "$PM_REPO" && test -d "$AI_REPO"
RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/current-pair"
MANIFEST="$RECOVERY_ROOT/manifest"; test ! -e "$RECOVERY_ROOT"
mkdir -p "$RECOVERY_ROOT/bin" "$MANIFEST" "$RECOVERY_ROOT/pm/roles" "$RECOVERY_ROOT/pm/skills" \
  "$RECOVERY_ROOT/pm/claude-skills" "$RECOVERY_ROOT/pm/codex-skills" "$RECOVERY_ROOT/ai" \
  "$RECOVERY_ROOT/sources"
printf '%s\n' "$BIN_DIR"  > "$MANIFEST/bin-dir.txt"
printf '%s\n' "$PM_STATE" > "$MANIFEST/pm-state-path.txt"
printf '%s\n' "$AI_STATE" > "$MANIFEST/ai-state-path.txt"
# Same filesystem, so every restore is an atomic rename, not a copy that can fail halfway.
device="$(stat -f %d "$RECOVERY_ROOT")"
for t in "$BIN_DIR" "$HOME/.agents/skills" "$HOME/.roles" "$HOME/.claude/skills" \
         "$HOME/.codex/skills" "$(dirname "$PM_STATE")" "$(dirname "$AI_STATE")"; do
  test -d "$t"; test "$(stat -f %d "$t")" = "$device"
done
{ sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' "$PM_REPO/scripts/setup.sh"
  sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' "$AI_REPO/scripts/setup.sh"
} > "$MANIFEST/binaries.txt"; test -s "$MANIFEST/binaries.txt"
while IFS= read -r n; do test -n "$n" || continue
  test -f "$BIN_DIR/$n" && test ! -L "$BIN_DIR/$n"
  cp -p "$BIN_DIR/$n" "$RECOVERY_ROOT/bin/$n"; cmp -s "$BIN_DIR/$n" "$RECOVERY_ROOT/bin/$n"
done < "$MANIFEST/binaries.txt"
ls -1A "$BIN_DIR" > "$MANIFEST/bin-inventory.txt"
shasum -a 256 "$RECOVERY_ROOT/bin"/* > "$RECOVERY_ROOT/bin.sha256"
# Recovery sources at the commit each saved executable itself reports: this makes the recovery
# baseline and the recovery binary the same pair by construction, with no revision written here.
TB_V="$("$RECOVERY_ROOT/bin/task-board" --version)"
SD_V="$("$RECOVERY_ROOT/bin/tb-sessiond" --version)"
AI_V="$("$RECOVERY_ROOT/bin/agents-infra" version)"
printf '%s\n%s\n%s\n' "$TB_V" "$SD_V" "$AI_V" > "$RECOVERY_ROOT/versions.txt"
commit_of() { printf '%s\n' "$1" | sed -nE \
  -e 's/.*\(commit ([0-9a-f]{7,40}),.*/\1/p' -e 's/.* commit=([0-9a-f]{7,40})( .*)?$/\1/p'; }
TB_C="$(commit_of "$TB_V")"; SD_C="$(commit_of "$SD_V")"; AI_C="$(commit_of "$AI_V")"
test -n "$TB_C" && test -n "$SD_C" && test -n "$AI_C"
test "$TB_C" = "$SD_C"       # one repo ships both; a divergence is an incoherent install
TB_SOURCE="$RECOVERY_ROOT/sources/skill-project-management"
AI_SOURCE="$RECOVERY_ROOT/sources/relux-agents-infra"
# Absolute destinations: `git -C repo worktree add` resolves a relative path against the
# repository, not the caller, and still exits zero.
git -C "$PM_REPO" worktree add --detach "$TB_SOURCE" "$(git -C "$PM_REPO" rev-parse --verify "${TB_C}^{commit}")"
git -C "$AI_REPO" worktree add --detach "$AI_SOURCE" "$(git -C "$AI_REPO" rev-parse --verify "${AI_C}^{commit}")"
test -z "$(git -C "$TB_SOURCE" status --porcelain)"; test -z "$(git -C "$AI_SOURCE" status --porcelain)"
git -C "$TB_SOURCE" rev-parse HEAD > "$MANIFEST/pm-source-commit.txt"
git -C "$AI_SOURCE" rev-parse HEAD > "$MANIFEST/ai-source-commit.txt"
# Re-read the artifact set from those exact sources: if a repo HEAD moved and changed it, the
# copy above is not the installed release's artifact set.
{ sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' "$TB_SOURCE/scripts/setup.sh"
  sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' "$AI_SOURCE/scripts/setup.sh"
} > "$MANIFEST/binaries-from-source.txt"
diff "$MANIFEST/binaries.txt" "$MANIFEST/binaries-from-source.txt"
ls -1A "$HOME/.roles" > "$MANIFEST/roles.txt"
while IFS= read -r r; do test -n "$r" || continue
  test -d "$HOME/.roles/$r"; rsync -a --delete "$HOME/.roles/$r/" "$RECOVERY_ROOT/pm/roles/$r/"
done < "$MANIFEST/roles.txt"
# Skills: the registered skill plus every SKILL_REPOS dependency, in all three trees. Every
# recorded kind gets a staged operand, or R1 aborts mid-restore.
{ sed -nE 's/^[[:space:]]*local skill_name="([^"]+)".*/\1/p' "$TB_SOURCE/scripts/setup.sh"
  sed -n '/^SKILL_REPOS=(/,/^)/p' "$TB_SOURCE/scripts/setup.sh" \
    | sed -nE 's/^[[:space:]]+([A-Za-z0-9._-]+)[[:space:]]+".*/\1/p'
} > "$MANIFEST/skills.txt"; test -s "$MANIFEST/skills.txt"
: > "$MANIFEST/skill-state.txt"
while IFS= read -r s; do
  test -n "$s" || continue; case "$s" in *"|"*) exit 76 ;; esac
  a="$HOME/.agents/skills/$s"
  if   [ -L "$a" ]; then line="$s|symlink|$(readlink "$a")"
  elif [ -d "$a" ]; then line="$s|directory|"; rsync -a --delete "$a/" "$RECOVERY_ROOT/pm/skills/$s/"
  elif [ -e "$a" ]; then exit 77
  else                   line="$s|absent|"; fi
  for spec in "$HOME/.claude/skills|claude-skills" "$HOME/.codex/skills|codex-skills"; do
    root="${spec%%|*}"; stage="${spec##*|}"
    if   [ -L "$root/$s" ]; then line="$line|symlink|$(readlink "$root/$s")"
    elif [ -d "$root/$s" ]; then line="$line|directory|"
         rsync -a --delete "$root/$s/" "$RECOVERY_ROOT/pm/$stage/$s/"
    elif [ -e "$root/$s" ]; then exit 77
    else                         line="$line|absent|"; fi
  done
  printf '%s\n' "$line" >> "$MANIFEST/skill-state.txt"
done < "$MANIFEST/skills.txt"
# Restorability, asserted at the door rather than during R1. R1 step 0 runs this same function
# against the artifact it consumes; neither may rely on the other having run.
snapshot_restorable() {   # $1 label  $2 kind  $3 recorded link  $4 staged tree
  case "$2" in
    symlink)   test -n "$3" || { printf 'UNRESTORABLE %s symlink with no target\n' "$1" >&2; return 1; } ;;
    directory) test -d "$4" || { printf 'UNRESTORABLE %s directory with no staged tree\n' "$1" >&2; return 1; } ;;
    absent)    : ;;
    *)         printf 'UNRESTORABLE %s unrecognised kind %s\n' "$1" "$2" >&2; return 1 ;;
  esac
}
while IFS='|' read -r s ak al ck cl xk xl; do test -n "$s" || continue
  snapshot_restorable "$s:.agents" "$ak" "$al" "$RECOVERY_ROOT/pm/skills/$s"
  snapshot_restorable "$s:.claude" "$ck" "$cl" "$RECOVERY_ROOT/pm/claude-skills/$s"
  snapshot_restorable "$s:.codex"  "$xk" "$xl" "$RECOVERY_ROOT/pm/codex-skills/$s"
done < "$MANIFEST/skill-state.txt"
# Both machine-scoped install states. The agents-infra one records repoPath, which the installed
# binary uses to resolve its own source tree.
install -m 0644 "$PM_STATE" "$RECOVERY_ROOT/pm/install.json"
install -m 0644 "$AI_STATE" "$RECOVERY_ROOT/ai/install.json"
```

## RB-3 — `r1.sh` (Part III rollback R1)

Requires `RECOVERY_ROOT` and `PM_RELEASE` in the environment, and `snapshot_restorable()`
from RB-2 (identical text; neither may rely on the other having run).

```bash
set -euo pipefail
MANIFEST="$RECOVERY_ROOT/manifest"; BIN_DIR="$(cat "$MANIFEST/bin-dir.txt")"
# --- 0. Admissibility, BEFORE anything is mutated: a snapshot R1 cannot fully restore produces
#        no partial run at all. snapshot_restorable() is the snapshot's, verbatim.
while IFS='|' read -r s ak al ck cl xk xl; do test -n "$s" || continue
  snapshot_restorable "$s:.agents" "$ak" "$al" "$RECOVERY_ROOT/pm/skills/$s"
  snapshot_restorable "$s:.claude" "$ck" "$cl" "$RECOVERY_ROOT/pm/claude-skills/$s"
  snapshot_restorable "$s:.codex"  "$xk" "$xl" "$RECOVERY_ROOT/pm/codex-skills/$s"
done < "$MANIFEST/skill-state.txt"
while IFS= read -r n; do test -n "$n" || continue
  test -f "$RECOVERY_ROOT/bin/$n" || { printf 'UNRESTORABLE bin/%s missing\n' "$n" >&2; exit 1; }
done < "$MANIFEST/binaries.txt"
while IFS= read -r r; do test -n "$r" || continue
  test -d "$RECOVERY_ROOT/pm/roles/$r" || { printf 'UNRESTORABLE roles/%s missing\n' "$r" >&2; exit 1; }
done < "$MANIFEST/roles.txt"
test -f "$RECOVERY_ROOT/pm/install.json"

# --- 0d. Stop every daemon this rollback would otherwise strand, BEFORE any executable is
#         replaced, while the INSTALLED CLI still matches the daemons the release started.
#         Nothing here escalates to SIGKILL: a daemon that does not exit is a STOP.
rc=0; pgrep -fl tb-sessiond > "$RECOVERY_ROOT/daemons-before-r1.txt" || rc=$?
case "$rc" in
  0) : ;;
  1) printf 'NO-LIVE-DAEMON\n' > "$RECOVERY_ROOT/daemons-before-r1.txt" ;;
  *) printf 'DAEMON-CENSUS-FAILED rc=%s\n' "$rc" >&2; exit 1 ;;
esac
sed -nE 's/.*--board-dir[ =]([^ ]+).*/\1/p' "$RECOVERY_ROOT/daemons-before-r1.txt" \
  | sort -u > "$RECOVERY_ROOT/daemon-boards-before-r1.txt"
# A remote-board daemon carries no --board-dir and is not modelled by this plan.
grep -q -- '--remote-origin' "$RECOVERY_ROOT/daemons-before-r1.txt" && {
  printf 'STOP remote-board daemon present; disposition it by hand before R1\n' >&2; exit 1; }
while IFS= read -r board; do
  test -n "$board" || continue
  tag="$(printf '%s' "$board" | tr / _)"
  st="$RECOVERY_ROOT/r1-status-$tag.json"; lst="$RECOVERY_ROOT/r1-list-$tag.json"
  task-board --board-dir "$board" --json session status > "$st"   # Dial-based: starts nothing,
  task-board --board-dir "$board" --json session list   > "$lst"  # restarts nothing
  running="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1])).get("running"))' "$st")"
  case "$running" in
    False|false) continue ;;                     # nothing holds this board
    True|true)   : ;;
    *) printf 'STOP unreadable running flag for %s\n' "$board" >&2; exit 1 ;;
  esac
  pid="$(python3  -c 'import json,sys;print(json.load(open(sys.argv[1]))["pid"])' "$st")"
  fp="$(python3   -c 'import json,sys;print(json.load(open(sys.argv[1]))["board_fingerprint"])' "$st")"
  sess="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["session_count"])' "$st")"
  quar="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["quarantined_count"])' "$st")"
  # Strict: `session list` is Count()+QuarantinedCount() rows (manager.go:959-981) and every row
  # carries a plain non-omitempty `attached_clients` int (types.go:57). An absent key, a missing
  # or non-list `sessions`, or a row count that disagrees with status is UNKNOWN -- never a
  # measured zero, which would refuse nothing and then be recorded below as a fact nobody read.
  att="$(python3 -c 'import json,sys
d=json.load(open(sys.argv[1])); rows=d["sessions"]
if not isinstance(rows,list): raise SystemExit("sessions is not a list")
if len(rows)!=int(sys.argv[2]): raise SystemExit("%d rows vs status %s"%(len(rows),sys.argv[2]))
print(sum(int(r["attached_clients"]) for r in rows))' "$lst" "$((sess+quar))")" \
    || { printf 'STOP attached_clients for %s is unknown, not zero\n' "$board" >&2; exit 1; }
  # An attached client is somebody`s running agent, and stopping the daemon closes the proxy
  # listener it talks through -- its owner`s decision, never a rollback side effect.
  [ "$att" = "0" ] || { printf 'STOP %s has %s attached client(s)\n' "$board" "$att" >&2; exit 1; }
  ep="$(task-board --board-dir "$board" --json session status \
        | python3 -c 'import json,sys;print(json.load(sys.stdin)["pid"])')"
  test "$ep" = "$pid" || { printf 'STOP pid moved for %s\n' "$board" >&2; exit 1; }
  ps -p "$pid" -o command= | grep -q tb-sessiond \
    || { printf 'STOP pid %s is not a tb-sessiond\n' "$pid" >&2; exit 1; }
  kill -TERM "$pid"
  stopped=no; i=0
  while [ "$i" -lt 30 ]; do
    ps -p "$pid" > /dev/null 2>&1 || { stopped=yes; break; }
    sleep 1; i=$((i+1))
  done
  test "$stopped" = yes || { printf 'STOP daemon %s on %s did not exit in 30s; do NOT SIGKILL, escalate\n' \
    "$pid" "$board" >&2; exit 1; }
  # Deferred cleanup removes socket, endpoint and token, so the next ConnectOrStart sees
  # ErrManagerNotRunning and launches rather than taking over.
  after="$(task-board --board-dir "$board" --json session status \
           | python3 -c 'import json,sys;print(json.load(sys.stdin).get("running"))')"
  case "$after" in False|false) : ;;
    *) printf 'STOP %s still reports a running daemon after stop\n' "$board" >&2; exit 1 ;; esac
  # Recorded only after the stop is CONFIRMED: step 5 asserts against this file.
  printf 'r1-0d board=%s pid=%s fingerprint=%s sessions=%s attached=%s\n' \
    "$board" "$pid" "$fp" "$sess" "$att" >> "$RECOVERY_ROOT/r1-daemons-stopped.txt"
done < "$RECOVERY_ROOT/daemon-boards-before-r1.txt"

# --- 1. Executables: restore what the snapshot holds, quarantine what the release added.
#        Anything else in BIN_DIR is reported, not removed: this plan owns the release's
#        artifacts, not the operator's PATH.
sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' "$PM_RELEASE/scripts/setup.sh" \
  | sort -u > /tmp/r1-release-bins.$$
sort -u "$MANIFEST/binaries.txt" > /tmp/r1-snapshot-bins.$$
while IFS= read -r n; do test -n "$n" || continue
  if grep -qxF "$n" /tmp/r1-snapshot-bins.$$; then
    install -m 0755 "$RECOVERY_ROOT/bin/$n" "$BIN_DIR/.$n.rollback.$$"
    mv -f "$BIN_DIR/.$n.rollback.$$" "$BIN_DIR/$n"
  elif [ -e "$BIN_DIR/$n" ]; then mv -f "$BIN_DIR/$n" "$BIN_DIR/.$n.superseded.$$"; fi
done < /tmp/r1-release-bins.$$
ls -1A "$BIN_DIR" | grep -vxF -f "$MANIFEST/bin-inventory.txt" \
  > "$RECOVERY_ROOT/unexpected-bin-entries.txt" || true

# --- 2. Skills to their exact recorded kind in all three trees, then removal of any skill this
#        release's SKILL_REPOS added.
restore_entry() {   # $1 path  $2 kind  $3 link target  $4 staged tree
  d="$(dirname "$1")"; b="$(basename "$1")"
  case "$2" in
    symlink)   test -n "$3"; rm -rf "$1"; ln -sfn "$3" "$1" ;;
    directory) test -d "$4"; rsync -a --delete "$4/" "$d/.$b.rollback.$$/"
               [ -e "$1" ] || [ -L "$1" ] && mv "$1" "$d/.$b.failed.$$"
               mv "$d/.$b.rollback.$$" "$1" ;;
    absent)    if [ -e "$1" ] || [ -L "$1" ]; then mv "$1" "$d/.$b.superseded.$$"; fi ;;
    *)         printf 'UNKNOWN-KIND %s %s\n' "$1" "$2" >&2; return 1 ;;
  esac
}
while IFS='|' read -r s ak al ck cl xk xl; do test -n "$s" || continue
  restore_entry "$HOME/.agents/skills/$s" "$ak" "$al" "$RECOVERY_ROOT/pm/skills/$s"
  restore_entry "$HOME/.claude/skills/$s" "$ck" "$cl" "$RECOVERY_ROOT/pm/claude-skills/$s"
  restore_entry "$HOME/.codex/skills/$s"  "$xk" "$xl" "$RECOVERY_ROOT/pm/codex-skills/$s"
done < "$MANIFEST/skill-state.txt"
{ sed -nE 's/^[[:space:]]*local skill_name="([^"]+)".*/\1/p' "$PM_RELEASE/scripts/setup.sh"
  sed -n '/^SKILL_REPOS=(/,/^)/p' "$PM_RELEASE/scripts/setup.sh" \
    | sed -nE 's/^[[:space:]]+([A-Za-z0-9._-]+)[[:space:]]+".*/\1/p'
} | sort -u > /tmp/r1-release-skills.$$
comm -23 /tmp/r1-release-skills.$$ <(sort -u "$MANIFEST/skills.txt") > /tmp/r1-added-skills.$$
while IFS= read -r s; do test -n "$s" || continue
  restore_entry "$HOME/.agents/skills/$s" absent "" ""
  restore_entry "$HOME/.claude/skills/$s" absent "" ""
  restore_entry "$HOME/.codex/skills/$s"  absent "" ""
done < /tmp/r1-added-skills.$$

# --- 3. Roles: restore every snapshot role, quarantine every role the release added.
while IFS= read -r r; do test -n "$r" || continue
  rsync -a --delete "$RECOVERY_ROOT/pm/roles/$r/" "$HOME/.roles/.$r.rollback.$$/"
  [ -e "$HOME/.roles/$r" ] && mv "$HOME/.roles/$r" "$HOME/.roles/.$r.failed.$$"
  mv "$HOME/.roles/.$r.rollback.$$" "$HOME/.roles/$r"
done < "$MANIFEST/roles.txt"
comm -23 <(ls -1A "$PM_RELEASE/.roles" | sort -u) <(sort -u "$MANIFEST/roles.txt") \
  > /tmp/r1-added-roles.$$
while IFS= read -r r; do test -n "$r" || continue
  [ -e "$HOME/.roles/$r" ] && mv "$HOME/.roles/$r" "$HOME/.roles/.$r.superseded.$$"
done < /tmp/r1-added-roles.$$

# --- 4. Install state last.
pmdir="$(dirname "$(cat "$MANIFEST/pm-state-path.txt")")"
install -m 0644 "$RECOVERY_ROOT/pm/install.json" "$pmdir/.install.json.rollback.$$"
mv -f "$pmdir/.install.json.rollback.$$" "$(cat "$MANIFEST/pm-state-path.txt")"

# --- 5. Assert the state 0d removed is still gone: an assumption that a state was removed is
#        worth nothing next to a read that it is gone. A daemon that reappears here was started
#        DURING R1 -- by an agent, a cron or a human.
rc=0; pgrep -fl tb-sessiond > "$RECOVERY_ROOT/daemons-after-r1.txt" || rc=$?
case "$rc" in
  0) : ;;
  1) printf 'NO-LIVE-DAEMON\n' > "$RECOVERY_ROOT/daemons-after-r1.txt" ;;
  *) printf 'DAEMON-CENSUS-FAILED rc=%s\n' "$rc" >&2; exit 1 ;;
esac
sed -nE 's/.*--board-dir[ =]([^ ]+).*/\1/p' "$RECOVERY_ROOT/daemons-after-r1.txt" \
  | sort -u > "$RECOVERY_ROOT/daemon-boards-after-r1.txt"
if [ -f "$RECOVERY_ROOT/r1-daemons-stopped.txt" ]; then
  awk '{sub(/^board=/,"",$2); print $2}' "$RECOVERY_ROOT/r1-daemons-stopped.txt" \
    | sort -u > "$RECOVERY_ROOT/r1-stopped-boards.txt"
  comm -12 "$RECOVERY_ROOT/r1-stopped-boards.txt" "$RECOVERY_ROOT/daemon-boards-after-r1.txt" \
    | grep -q . && { printf 'STOP a board stopped by step 0d has a daemon again\n' >&2; exit 1; }
fi
rm -f /tmp/r1-*.$$
```

## RB-4 — `r2.sh` (Part III rollback R2)

Requires `RECOVERY_ROOT` and `AI_RELEASE` in the environment.

```bash
set -euo pipefail
MANIFEST="$RECOVERY_ROOT/manifest"; BIN_DIR="$(cat "$MANIFEST/bin-dir.txt")"
sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' \
  "$AI_RELEASE/scripts/setup.sh" | sort -u > /tmp/r2-release-bins.$$
while IFS= read -r n; do test -n "$n" || continue
  if grep -qxF "$n" "$MANIFEST/binaries.txt"; then
    install -m 0755 "$RECOVERY_ROOT/bin/$n" "$BIN_DIR/.$n.rollback.$$"
    mv -f "$BIN_DIR/.$n.rollback.$$" "$BIN_DIR/$n"
  elif [ -e "$BIN_DIR/$n" ]; then mv -f "$BIN_DIR/$n" "$BIN_DIR/.$n.superseded.$$"; fi
done < /tmp/r2-release-bins.$$
# Re-mint from the exact saved source. AGENTS_INFRA_SOURCE_DIR must not be set: it would
# override --source-dir's resolution order.
env -u AGENTS_INFRA_SOURCE_DIR -u AGENTS_INFRA_CONFIG_DIR \
  "$RECOVERY_ROOT/bin/agents-infra" setup global \
  --source-dir "$RECOVERY_ROOT/sources/relux-agents-infra"
env -u AGENTS_INFRA_SOURCE_DIR "$RECOVERY_ROOT/bin/agents-infra" verify global
env -u AGENTS_INFRA_SOURCE_DIR "$RECOVERY_ROOT/bin/agents-infra" doctor global
# Install state last: the installed binary resolves its own source tree through it.
aidir="$(dirname "$(cat "$MANIFEST/ai-state-path.txt")")"
install -m 0644 "$RECOVERY_ROOT/ai/install.json" "$aidir/.install.json.rollback.$$"
mv -f "$aidir/.install.json.rollback.$$" "$(cat "$MANIFEST/ai-state-path.txt")"
rm -f /tmp/r2-release-bins.$$
```
