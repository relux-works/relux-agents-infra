#!/usr/bin/env bash
# Adversarial validation for document revision 7.
# Read-only against production installers. Never executes an installer.
# Usage: validate-revision7.sh <shell-label>
set -uo pipefail

PM_SRC="${PM_SRC:-/Users/alexis/src/relux-works/skill-project-management}"
AI_SRC="${AI_SRC:-/Users/alexis/src/relux-works/relux-agents-infra}"
SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/rev7-XXXXXX")"
trap 'rm -rf "$SANDBOX"' EXIT

pass=0; fail=0
ok()   { pass=$((pass+1)); printf 'PASS  %s\n' "$1"; }
bad()  { fail=$((fail+1)); printf 'FAIL  %s -- %s\n' "$1" "${2:-}"; }
check(){ if [ "$2" = "$3" ]; then ok "$1 ($2)"; else bad "$1" "expected [$3] got [$2]"; fi; }

# ---------- the exact commands the plan gives the operator ----------
pm_bins()   { sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' "$1/scripts/setup.sh"; }
ai_bins()   { sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' "$1/scripts/setup.sh"; }
skill_repos(){ sed -n '/^SKILL_REPOS=(/,/^)/p' "$1/scripts/setup.sh" \
                | sed -nE 's/^[[:space:]]+([A-Za-z0-9._-]+)[[:space:]]+".*/\1/p'; }
pm_envset() { grep -ohE '\$\{?[A-Z][A-Z0-9_]*' "$1/scripts/setup.sh" \
                "$1/scripts/lib/agents-infra-compose.zsh" | sed -E 's/^\$\{?//' | sort -u; }
ai_envset() { grep -ohE '\$\{?[A-Z][A-Z0-9_]*' "$1/scripts/setup.sh" \
                | sed -E 's/^\$\{?//' | sort -u; }
indirect()  { grep -nE 'printenv|(^|[^_[:alnum:]])eval([^_[:alnum:]]|$)|\$\{\(P\)|\$\{!|\[\[ -v ' "$@"; }
extract_commit() { printf '%s\n' "$1" | sed -nE \
  -e 's/.*\(commit ([0-9a-f]{7,40}),.*/\1/p' \
  -e 's/.* commit=([0-9a-f]{7,40})( .*)?$/\1/p'; }

# ---------- the guard the plan REPLACED, kept verbatim as a control ----------
rev6_extract_function_body() (
  set -euo pipefail
  file="$1"; function_name="$2"
  test -f "$file" || exit 70
  body=""
  if ! body="$(awk -v want="$function_name" '
    $0 ~ "^" want "\\(\\) \\{" { inside = 1 }
    inside                     { print }
    inside && $0 == "}"        { exit }
  ' "$file")"; then exit 70; fi
  test -n "$body" || exit 70
  printf '%s\n' "$body"
)
rev6_mirror_unchanged() (
  set -euo pipefail
  b=""; c=""
  if ! b="$(rev6_extract_function_body "$1" "$3")"; then exit 70; fi
  if ! c="$(rev6_extract_function_body "$2" "$3")"; then exit 70; fi
  if ! test "$b" = "$c"; then exit 73; fi
  printf 'MIRROR_UNCHANGED|%s\n' "$3"
)
rev6_env_names() { grep -oE '\$\{[A-Z_]+:-' "$1" | sed -E 's/\$\{([A-Z_]+):-/\1/' | sort -u; }

echo "=== A. Reproduce the plan's recorded derivations against production ==="
check "A1 pm executables count"  "$(pm_bins "$PM_SRC" | wc -l | tr -d ' ')" "5"
check "A2 pm executables"        "$(pm_bins "$PM_SRC" | tr '\n' ' ')" \
      "task-board tb-sessiond task-board-tui openai-board anthropic-board "
check "A3 ai executables"        "$(ai_bins "$AI_SRC" | tr '\n' ' ')" "agents-infra model-harness "
check "A4 skill_repos count"     "$(skill_repos "$PM_SRC" | wc -l | tr -d ' ')" "8"
check "A5 release roles count"   "$(ls -1A "$PM_SRC/.roles" | wc -l | tr -d ' ')" "9"
check "A6 pm env-set size"       "$(pm_envset "$PM_SRC" | wc -l | tr -d ' ')" "38"
check "A7 ai env-set size"       "$(ai_envset "$AI_SRC" | wc -l | tr -d ' ')" "19"
# The two names revision 6's classifier arrangement mishandled must be present.
if pm_envset "$PM_SRC" | grep -qx HOME; then ok "A8 pm env-set contains bare-\$HOME name"
else bad "A8 pm env-set contains HOME"; fi
if ai_envset "$AI_SRC" | grep -qx AGENTS_INFRA_SKIP_LLDB_MCP && \
   ai_envset "$AI_SRC" | grep -qx BIN_DIR; then ok "A9 ai env-set contains SKIP flag and BIN_DIR"
else bad "A9 ai env-set contains SKIP flag and BIN_DIR"; fi
indirect "$PM_SRC/scripts/setup.sh" "$PM_SRC/scripts/lib/agents-infra-compose.zsh" >/dev/null 2>&1
check "A10 indirect probe pm (expect no match => exit 1)" "$?" "1"
indirect "$AI_SRC/scripts/setup.sh" >/dev/null 2>&1
check "A11 indirect probe ai (expect no match => exit 1)" "$?" "1"
check "A12 pm sourced-file count" \
      "$(grep -c '^source \|^\. ' "$PM_SRC/scripts/setup.sh")" "1"
check "A13 install_skills is ungated in setup_main" \
      "$(sed -n '/^setup_main() {/,/^}/p' "$PM_SRC/scripts/setup.sh" | grep -c '^  install_skills$')" "1"
check "A14 install_go is the first stage in setup_main" \
      "$(sed -n '/^setup_main() {/,/^}/p' "$PM_SRC/scripts/setup.sh" \
         | grep -nE '^  (install_go|install_binary)' | head -1 | sed 's/.*  //')" "install_go"
check "A15 install_go precedes install_lldb_mcp in the ai call sequence" \
      "$(grep -nE '^(install_go|install_lldb_mcp)$' "$AI_SRC/scripts/setup.sh" | tr '\n' ' ')" \
      "258:install_go 259:install_lldb_mcp "

echo
echo "=== B. P1 vs the revision-6 mirror guard, on the exact F2 case ==="
mkdir -p "$SANDBOX/saved/scripts/lib" "$SANDBOX/rel/scripts/lib"
cp "$PM_SRC/scripts/setup.sh" "$SANDBOX/saved/scripts/setup.sh"
cp "$PM_SRC/scripts/lib/agents-infra-compose.zsh" "$SANDBOX/saved/scripts/lib/"
cp -R "$SANDBOX/saved/scripts/." "$SANDBOX/rel/scripts/"

# F2 mutant: change write_install_state AFTER the heredoc's column-0 '}'.
python3 - "$SANDBOX/rel/scripts/setup.sh" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
old='''EOF

  green "Install state: $state_path"'''
new='''EOF

  mkdir -p "$HOME/.config/task-board-v2"
  cp "$state_path" "$HOME/.config/task-board-v2/install.json"
  green "Install state: $state_path"'''
assert old in s, "anchor missing"
open(p,'w').write(s.replace(old,new,1))
PY
rev6_mirror_unchanged "$SANDBOX/saved/scripts/setup.sh" "$SANDBOX/rel/scripts/setup.sh" \
  write_install_state >/dev/null 2>&1
r6=$?
diff -ru "$SANDBOX/saved/scripts" "$SANDBOX/rel/scripts" >/dev/null 2>&1
p1=$?
check "B1 rev6 mirror guard on the F2 mutant (fail-open: 0 == admitted)" "$r6" "0"
check "B2 P1 diff -ru on the same mutant (1 == refused)"                 "$p1" "1"

# F2 second half: top-level *_SKILLS_DIR assignment moved, no function body changed.
cp -R "$SANDBOX/saved/scripts/." "$SANDBOX/rel/scripts/"
perl -pi -e 's|^AGENTS_SKILLS_DIR="\$HOME/\.agents/skills"$|AGENTS_SKILLS_DIR="$HOME/.agents/skills-moved"|' \
  "$SANDBOX/rel/scripts/setup.sh"
grep -qE '^AGENTS_SKILLS_DIR=".*skills-moved"$' "$SANDBOX/rel/scripts/setup.sh" \
  || bad "B3-setup" "top-level mutant not applied"
r6b=0
for fn in install_skills write_install_state install_roles register_skill; do
  rev6_mirror_unchanged "$SANDBOX/saved/scripts/setup.sh" "$SANDBOX/rel/scripts/setup.sh" \
    "$fn" >/dev/null 2>&1 || r6b=$?
done
diff -ru "$SANDBOX/saved/scripts" "$SANDBOX/rel/scripts" >/dev/null 2>&1
p1b=$?
check "B3 rev6 mirrors on a top-level assignment mutant (0 == all four admitted)" "$r6b" "0"
check "B4 P1 diff -ru on the same mutant (1 == refused)"                          "$p1b" "1"

# Narrowing control: an unchanged tree must PASS P1, not just "always fail".
cp -R "$SANDBOX/saved/scripts/." "$SANDBOX/rel/scripts/"
diff -ru "$SANDBOX/saved/scripts" "$SANDBOX/rel/scripts" >/dev/null 2>&1
check "B5 P1 on an identical tree (0 == admitted; narrowing control)" "$?" "0"

echo
echo "=== C. P2 vs the revision-6 override derivation, all four shapes ==="
declare -a SHAPES=(
  'if [ -n "$TASK_BOARD_FUTURE_FLAG" ]; then :; fi|bare-\$VAR'
  'OTHER_FLAG=${OTHER_FLAG-0}|\${X-}'
  ': ${THIRD_FLAG:=on}|\${X:=}'
  'FOURTH=${FOURTH_FLAG:-x}|\${X:-} (control)'
)
for entry in "${SHAPES[@]}"; do
  inj="${entry%%|*}"; label="${entry##*|}"
  cp "$PM_SRC/scripts/setup.sh" "$SANDBOX/rel/scripts/setup.sh"
  printf '%s\n' "$inj" >> "$SANDBOX/rel/scripts/setup.sh"
  # revision 6's derivation + classifier: does an unknown name reach it?
  if rev6_env_names "$SANDBOX/rel/scripts/setup.sh" \
       | grep -qE '^(TASK_BOARD_FUTURE_FLAG|OTHER_FLAG|THIRD_FLAG|FOURTH_FLAG)$'; then
    r6c=refused; else r6c=admitted; fi
  if diff <(pm_envset "$SANDBOX/saved") <(pm_envset "$SANDBOX/rel") >/dev/null 2>&1; then
    p2c=admitted; else p2c=refused; fi
  check "C-rev6 $label" "$r6c" "$([ "$label" = '\${X:-} (control)' ] && echo refused || echo admitted)"
  check "C-P2   $label" "$p2c" "refused"
done
# Narrowing control: unmodified release must be admitted by P2.
cp "$PM_SRC/scripts/setup.sh" "$SANDBOX/rel/scripts/setup.sh"
diff <(pm_envset "$SANDBOX/saved") <(pm_envset "$SANDBOX/rel") >/dev/null 2>&1
check "C-P2 unmodified release (0 == admitted; narrowing control)" "$?" "0"
# Indirect probe must fire on an injected eval.
printf 'eval "$SOMETHING"\n' >> "$SANDBOX/rel/scripts/setup.sh"
indirect "$SANDBOX/rel/scripts/setup.sh" >/dev/null 2>&1
check "C-P2 indirect probe on injected eval (0 == fired)" "$?" "0"

echo
echo "=== D. P6 formula-list snapshot vs the revision-6 LLDB target set ==="
# Simulate a brew mutation by an unmodelled stage: formula 'go' appears.
printf 'llvm\nripgrep\n'      > "$SANDBOX/formulae-before.txt"
printf 'go\nllvm\nripgrep\n'  > "$SANDBOX/formulae-after.txt"
# revision 6 recorded only: llvm installed-yes/no, llvm versions, llvm prefix,
# and four file paths. None of them changes when 'go' is installed.
printf 'LLVM_INSTALLED|llvm 23.1.0\nLLVM_PREFIX|/opt/homebrew/opt/llvm\nABSENT|x\n' \
  > "$SANDBOX/rev6-before.txt"
cp "$SANDBOX/rev6-before.txt" "$SANDBOX/rev6-after.txt"
cmp -s "$SANDBOX/rev6-before.txt" "$SANDBOX/rev6-after.txt"
check "D1 rev6 target set sees a brew install go (0 == identical == blind)" "$?" "0"
diff "$SANDBOX/formulae-before.txt" "$SANDBOX/formulae-after.txt" >/dev/null 2>&1
check "D2 P6 formula list sees the same mutation (1 == difference)" "$?" "1"
diff "$SANDBOX/formulae-before.txt" "$SANDBOX/formulae-before.txt" >/dev/null 2>&1
check "D3 P6 on an unmutated formula list (0; narrowing control)" "$?" "0"

echo
echo "=== E. P3 and P4 ==="
# P3 red when go is not resolvable in the invocation environment.
mkdir -p "$SANDBOX/emptybin"
( PATH="$SANDBOX/emptybin"; command -v go >/dev/null 2>&1 ) ; check "E1 P3 with go absent from PATH (1 == refused)" "$?" "1"
( command -v go >/dev/null 2>&1 ) ; check "E2 P3 on the real host (0 == satisfied)" "$?" "0"

# P4 loop, driven against a disposable HOME.
p4_check() { # $1 = HOME, $2 = skills file
  local refused=0 skill target
  while IFS= read -r skill; do
    [ -n "$skill" ] || continue
    target="$1/.agents/skills/$skill"
    if   [ -L "$target" ]; then refused=1
    elif [ -d "$target" ]; then :
    elif [ -e "$target" ]; then refused=1
    else                        refused=1
    fi
  done < "$2"
  return "$refused"
}
H="$SANDBOX/home"; mkdir -p "$H/.agents/skills"
{ echo project-management; skill_repos "$PM_SRC"; } > "$SANDBOX/skills.txt"
while IFS= read -r s; do mkdir -p "$H/.agents/skills/$s"; done < "$SANDBOX/skills.txt"
p4_check "$H" "$SANDBOX/skills.txt"; check "E3 P4 all real directories (0 == OK)" "$?" "0"
rm -rf "$H/.agents/skills/swiftui"; ln -s /tmp "$H/.agents/skills/swiftui"
p4_check "$H" "$SANDBOX/skills.txt"; check "E4 P4 one symlinked skill (1 == REFUSE)" "$?" "1"
rm -f "$H/.agents/skills/swiftui"
p4_check "$H" "$SANDBOX/skills.txt"; check "E5 P4 one absent skill (1 == REFUSE)" "$?" "1"
mkdir -p "$H/.agents/skills/swiftui"
p4_check "$H" "$SANDBOX/skills.txt"; check "E6 P4 after materialising (0 == OK; narrowing control)" "$?" "0"

echo
echo "=== F. Version-string commit extraction ==="
check "F1 task-board shape" \
  "$(extract_commit 'task-board version 0.24.3-172-g063197b1 (commit 063197b1, built 2026-08-30T06:56:20Z)')" \
  "063197b1"
check "F2 agents-infra shape" \
  "$(extract_commit 'agents-infra v1.6.1-103-g4270549 commit=4270549 build_date=2026-08-30T10:01:14Z')" \
  "4270549"
check "F3 unparseable version yields empty (caller's test -n refuses)" \
  "$(extract_commit 'agents-infra (unknown build)')" ""

echo
echo "=== G. R1 rollback, executed verbatim from the plan document ==="
# The R1 block is extracted from the plan, not retyped. It runs against a
# disposable HOME with fake artifacts. No installer is executed.
# Located by content, so a line-number shift in the document cannot silently
# point this suite at the wrong block or at nothing.
R1_BLOCK="$(grep -l 'r1-release-bins' "$(dirname "$0")"/fences/*.bash | head -1)"
if [ -n "$R1_BLOCK" ] && [ -f "$R1_BLOCK" ]; then ok "G0 R1 fence located ($(basename "$R1_BLOCK"))"
else bad "G0" "R1 fence not found in extracted fences"; R1_BLOCK=/dev/null; fi

P6_BLOCK="$(grep -l 'brew list --formula' "$(dirname "$0")"/fences/*.bash | tail -1)"
if [ -n "$P6_BLOCK" ] && [ -f "$P6_BLOCK" ]; then ok "G0b P6 script located ($(basename "$P6_BLOCK"))"
else bad "G0b" "P6 fence not found"; P6_BLOCK=/dev/null; fi

G="$SANDBOX/g"; export HOME="$G/home"
RECOVERY_ROOT="$G/recovery"; MANIFEST="$RECOVERY_ROOT/manifest"
PM_RELEASE="$G/release"
mkdir -p "$HOME/.local/bin" "$HOME/.roles" "$HOME/.agents/skills" \
         "$HOME/.claude/skills" "$HOME/.codex/skills" "$HOME/.config/task-board" \
         "$RECOVERY_ROOT/bin" "$MANIFEST" "$RECOVERY_ROOT/pm/roles" \
         "$RECOVERY_ROOT/pm/skills" "$PM_RELEASE/scripts" "$PM_RELEASE/.roles"

# --- installed "old" state -------------------------------------------------
printf '%s\n' "$HOME/.local/bin" > "$MANIFEST/bin-dir.txt"
printf '%s\n' "$HOME/.config/task-board/install.json" > "$MANIFEST/pm-state-path.txt"
for b in task-board tb-sessiond task-board-tui openai-board anthropic-board; do
  printf 'OLD-%s\n' "$b" > "$HOME/.local/bin/$b"; chmod +x "$HOME/.local/bin/$b"
  cp -p "$HOME/.local/bin/$b" "$RECOVERY_ROOT/bin/$b"
  printf '%s\n' "$b" >> "$MANIFEST/binaries.txt"
done
ls -1A "$HOME/.local/bin" > "$MANIFEST/bin-inventory.txt"

for r in developer reviewer tester; do
  mkdir -p "$HOME/.roles/$r"; printf 'OLD-%s\n' "$r" > "$HOME/.roles/$r/role.md"
  rsync -aq --delete "$HOME/.roles/$r/" "$RECOVERY_ROOT/pm/roles/$r/"
  printf '%s\n' "$r" >> "$MANIFEST/roles.txt"
done

for s in project-management swiftui; do
  mkdir -p "$HOME/.agents/skills/$s"; printf 'v1\n' > "$HOME/.agents/skills/$s/SKILL.md"
  rsync -aq --delete "$HOME/.agents/skills/$s/" "$RECOVERY_ROOT/pm/skills/$s/"
  ln -sfn "$HOME/.agents/skills/$s" "$HOME/.claude/skills/$s"
  ln -sfn "$HOME/.agents/skills/$s" "$HOME/.codex/skills/$s"
  printf '%s\n' "$s" >> "$MANIFEST/skills.txt"
  printf '%s|directory||symlink|%s|symlink|%s\n' "$s" \
    "$HOME/.agents/skills/$s" "$HOME/.agents/skills/$s" >> "$MANIFEST/skill-state.txt"
done
printf '{"repoPath":"/old/repo","binDir":"%s"}\n' "$HOME/.local/bin" \
  > "$HOME/.config/task-board/install.json"
install -m 0644 "$HOME/.config/task-board/install.json" "$RECOVERY_ROOT/pm/install.json"

# --- a release that ADDS a binary, a role and a skill, and REWRITES one ----
{
  echo '  install_binary "task-board" "$CLI_BINARY"'
  echo '  install_binary "tb-sessiond" "$SESSIOND_BINARY"'
  echo '  install_binary "task-board-tui" "$TUI_BINARY"'
  echo '  install_binary "openai-board" "$X"'
  echo '  install_binary "anthropic-board" "$X"'
  echo '  install_binary "gemini-board" "$X"'          # ADDED
  echo '  local skill_name="project-management"'
  echo 'SKILL_REPOS=('
  echo '  swiftui   "git@example:swiftui.git"'
  echo '  brand-new "git@example:brand-new.git"'       # ADDED
  echo ')'
} > "$PM_RELEASE/scripts/setup.sh"
for r in developer reviewer tester brand-new-role; do mkdir -p "$PM_RELEASE/.roles/$r"; done

# simulate what the release did to the host
printf 'NEW-gemini\n' > "$HOME/.local/bin/gemini-board"; chmod +x "$HOME/.local/bin/gemini-board"
for b in task-board tb-sessiond; do printf 'NEW-%s\n' "$b" > "$HOME/.local/bin/$b"; done
mkdir -p "$HOME/.roles/brand-new-role"; printf 'NEW\n' > "$HOME/.roles/brand-new-role/role.md"
printf 'NEW-developer\n' > "$HOME/.roles/developer/role.md"
mkdir -p "$HOME/.agents/skills/brand-new"; printf 'new\n' > "$HOME/.agents/skills/brand-new/SKILL.md"
ln -sfn "$HOME/.agents/skills/brand-new" "$HOME/.claude/skills/brand-new"
ln -sfn "$HOME/.agents/skills/brand-new" "$HOME/.codex/skills/brand-new"
printf 'v2-from-release\n' > "$HOME/.agents/skills/swiftui/SKILL.md"
printf '{"repoPath":"/release/candidate","binDir":"%s"}\n' "$HOME/.local/bin" \
  > "$HOME/.config/task-board/install.json"
# an unrelated binary the operator owns
printf 'MINE\n' > "$HOME/.local/bin/my-tool"; chmod +x "$HOME/.local/bin/my-tool"

# --- run R1 verbatim -------------------------------------------------------
( export RECOVERY_ROOT MANIFEST PM_RELEASE HOME; bash "$R1_BLOCK" ) >"$G/r1.log" 2>&1
check "G1 R1 exit" "$?" "0"

check "G2 restored task-board"        "$(cat "$HOME/.local/bin/task-board")"  "OLD-task-board"
check "G3 restored tb-sessiond"       "$(cat "$HOME/.local/bin/tb-sessiond")" "OLD-tb-sessiond"
check "G4 release-added binary quarantined" \
      "$(test -e "$HOME/.local/bin/gemini-board" && echo present || echo absent)" "absent"
check "G5 operator's unrelated binary untouched" \
      "$(cat "$HOME/.local/bin/my-tool")" "MINE"
check "G6 unrelated binary reported, not removed" \
      "$(grep -cx my-tool "$RECOVERY_ROOT/unexpected-bin-entries.txt")" "1"
check "G7 rewritten dependency skill restored" \
      "$(cat "$HOME/.agents/skills/swiftui/SKILL.md")" "v1"
check "G8 release-added skill removed from .agents" \
      "$(test -e "$HOME/.agents/skills/brand-new" && echo present || echo absent)" "absent"
check "G9 release-added skill link removed from .claude" \
      "$(test -e "$HOME/.claude/skills/brand-new" || test -L "$HOME/.claude/skills/brand-new" && echo present || echo absent)" "absent"
check "G10 release-added skill link removed from .codex" \
      "$(test -e "$HOME/.codex/skills/brand-new" || test -L "$HOME/.codex/skills/brand-new" && echo present || echo absent)" "absent"
check "G11 rewritten role restored"   "$(cat "$HOME/.roles/developer/role.md")" "OLD-developer"
check "G12 untouched role survives"   "$(cat "$HOME/.roles/tester/role.md")"    "OLD-tester"
check "G13 release-added role removed" \
      "$(test -e "$HOME/.roles/brand-new-role" && echo present || echo absent)" "absent"
check "G14 install state restored" \
      "$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["repoPath"])' \
         "$HOME/.config/task-board/install.json")" "/old/repo"
check "G15 preserved link kind for a snapshot skill" \
      "$(test -L "$HOME/.claude/skills/swiftui" && echo symlink || echo other)" "symlink"

echo
echo "--- P6 script, run read-only against the real Homebrew installation ---"
P6D="$SANDBOX/p6"; mkdir -p "$P6D"
bash "$P6_BLOCK" "$P6D/a.txt" >/dev/null 2>&1
check "H1 P6 fresh run on the real host" "$?" "0"
check "H2 P6 snapshot is non-trivial (whole formula list + 4 targets)" \
      "$([ "$(wc -l < "$P6D/a.txt")" -gt 100 ] && echo large || echo small)" "large"
bash "$P6_BLOCK" "$P6D/b.txt" >/dev/null 2>&1
diff "$P6D/a.txt" "$P6D/b.txt" >/dev/null 2>&1
check "H3 two consecutive clean runs compare identical (narrowing control)" "$?" "0"
bash "$P6_BLOCK" "$P6D/a.txt" >/dev/null 2>&1
check "H4 P6 refuses to overwrite an existing snapshot (non-zero)" \
      "$([ "$?" -ne 0 ] && echo refused || echo admitted)" "refused"
check "H5 and the earlier good snapshot survived that refusal" \
      "$([ -s "$P6D/a.txt" ] && echo intact || echo destroyed)" "intact"
( PATH=/usr/bin:/bin bash "$P6_BLOCK" "$P6D/nobrew.txt" ) >/dev/null 2>&1
check "H6 P6 with brew unreachable (non-zero; a failed read is not an absence)" \
      "$([ "$?" -ne 0 ] && echo refused || echo admitted)" "refused"
check "H7 and it published nothing" \
      "$([ -e "$P6D/nobrew.txt" ] && echo published || echo nothing)" "nothing"
check "H8 and it left no partial to be mistaken for a snapshot" \
      "$([ -e "$P6D/nobrew.txt.partial" ] && echo partial || echo clean)" "clean"
grep -vx go "$P6D/a.txt" > "$P6D/c.txt"
diff "$P6D/a.txt" "$P6D/c.txt" >/dev/null 2>&1
check "H9 removing one formula from the snapshot is a visible difference" "$?" "1"

# --- control: R1 is idempotent, so G16's refusal is attributable ----------
( export RECOVERY_ROOT MANIFEST PM_RELEASE HOME; bash "$R1_BLOCK" ) >"$G/r1-again.log" 2>&1
check "G17 R1 re-run on an already-restored state (0; isolates G16's cause)" "$?" "0"

# --- negative: a missing snapshot executable must abort, not half-restore --
cp -p "$RECOVERY_ROOT/bin/tb-sessiond" "$G/tb-sessiond.keep"
rm -f "$RECOVERY_ROOT/bin/tb-sessiond"
printf 'NEW-tb-sessiond\n' > "$HOME/.local/bin/tb-sessiond"
( export RECOVERY_ROOT MANIFEST PM_RELEASE HOME; bash "$R1_BLOCK" ) >"$G/r1-missing.log" 2>&1
check "G18 R1 refuses when a snapshot executable is missing (non-zero)" \
      "$([ "$?" -ne 0 ] && echo refused || echo admitted)" "refused"
check "G19 and it did not report success by leaving the release binary in place" \
      "$(cat "$HOME/.local/bin/tb-sessiond")" "NEW-tb-sessiond"
cp -p "$G/tb-sessiond.keep" "$RECOVERY_ROOT/bin/tb-sessiond"
( export RECOVERY_ROOT MANIFEST PM_RELEASE HOME; bash "$R1_BLOCK" ) >"$G/r1-fixed.log" 2>&1
check "G20 R1 green again once the snapshot is complete (narrowing control)" "$?" "0"
check "G21 and tb-sessiond is restored" "$(cat "$HOME/.local/bin/tb-sessiond")" "OLD-tb-sessiond"

# --- negative: an unrecognised snapshot kind must abort R1 -----------------
printf 'bogus-skill|WEIRD||symlink|/x|symlink|/x\n' >> "$MANIFEST/skill-state.txt"
( export RECOVERY_ROOT MANIFEST PM_RELEASE HOME; bash "$R1_BLOCK" ) >"$G/r1-bad.log" 2>&1
check "G16 R1 refuses an unrecognised snapshot kind (non-zero)" \
      "$([ "$?" -ne 0 ] && echo refused || echo admitted)" "refused"

echo
echo "shell=$1 pass=$pass fail=$fail"
test "$fail" -eq 0
