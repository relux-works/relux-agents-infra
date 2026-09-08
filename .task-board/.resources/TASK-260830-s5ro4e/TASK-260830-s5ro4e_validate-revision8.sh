#!/usr/bin/env bash
# Adversarial validation for document revision 8.
# Read-only against production sources. Never executes an installer, never
# signals a live process, never mutates an installed artifact.
# Usage: validate-revision8.sh <shell-label>
set -uo pipefail

PM_SRC="${PM_SRC:-/Users/alexis/src/relux-works/skill-project-management}"
AI_SRC="${AI_SRC:-/Users/alexis/src/relux-works/relux-agents-infra}"
SM="$PM_SRC/tools/board-cli/internal/sessionmanager"
CMD="$PM_SRC/tools/board-cli/cmd"
PLAN="${PLAN:-$(dirname "$0")/../../.research/260830_agents-management-lockstep-release-and-rollback.md}"
REAL_HOME="$HOME"
SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/rev8-XXXXXX")"
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
echo "=== D. FIXTURES, not a run of the document's P6 script ==="
# Revision-7 review F3: this section was described as running the document's own
# commands. It does not. Every operand below is written by printf here. It shows
# what the two SHAPES of snapshot can and cannot see; the document's actual P6
# block is executed in section H, against real Homebrew, and H9 is the row that
# measures the substantive property.
# Simulate a brew mutation by an unmodelled stage: formula 'go' appears.
printf 'llvm\nripgrep\n'      > "$SANDBOX/formulae-before.txt"
printf 'go\nllvm\nripgrep\n'  > "$SANDBOX/formulae-after.txt"
# revision 6 recorded only: llvm installed-yes/no, llvm versions, llvm prefix,
# and four file paths. None of them changes when 'go' is installed.
printf 'LLVM_INSTALLED|llvm 23.1.0\nLLVM_PREFIX|/opt/homebrew/opt/llvm\nABSENT|x\n' \
  > "$SANDBOX/rev6-before.txt"
cp "$SANDBOX/rev6-before.txt" "$SANDBOX/rev6-after.txt"
cmp -s "$SANDBOX/rev6-before.txt" "$SANDBOX/rev6-after.txt"
check "D1 fixture: rev6-shaped target set is identical across the mutation" "$?" "0"
diff "$SANDBOX/formulae-before.txt" "$SANDBOX/formulae-after.txt" >/dev/null 2>&1
check "D2 fixture: formula-list shape differs across the same mutation" "$?" "1"
diff "$SANDBOX/formulae-before.txt" "$SANDBOX/formulae-before.txt" >/dev/null 2>&1
check "D3 fixture: unmutated formula list compares identical (narrowing control)" "$?" "0"

echo
echo "=== E. P3, and P4 driven as the document's own fences ==="
# P3 red when go is not resolvable in the invocation environment.
mkdir -p "$SANDBOX/emptybin"
( PATH="$SANDBOX/emptybin"; command -v go >/dev/null 2>&1 ) ; check "E1 P3 with go absent from PATH (1 == refused)" "$?" "1"
( command -v go >/dev/null 2>&1 ) ; check "E2 P3 on the real host (0 == satisfied)" "$?" "0"

# Revision-7 review F3: revision 7 defined its own p4_check() function and
# claimed the document's P4 was driven. It was not. Both P4 fences are now
# located by content in the extracted fences and executed verbatim, including
# the manual append the document requires of the operator.
FENCES="$(dirname "$0")/fences"
P4A="$(grep -l 'release-skill-repos.txt' "$FENCES"/*.bash | head -1)"
P4B="$(grep -l 'REFUSE %-24s' "$FENCES"/*.bash | head -1)"
if [ -f "$P4A" ]; then ok "E0a P4a fence located ($(basename "$P4A"))"; else bad "E0a" "P4a fence not found"; P4A=/dev/null; fi
if [ -f "$P4B" ]; then ok "E0b P4b fence located ($(basename "$P4B"))"; else bad "E0b" "P4b fence not found"; P4B=/dev/null; fi

E="$SANDBOX/e"; mkdir -p "$E/recovery" "$E/home/.agents/skills"
# P4a verbatim, against the real production release tree.
( PM_RELEASE="$PM_SRC" RECOVERY_ROOT="$E/recovery" bash "$P4A" ) >/dev/null 2>&1
check "E3a P4a fence exit" "$?" "0"
check "E3b P4a derived 8 SKILL_REPOS keys" \
      "$(wc -l < "$E/recovery/release-skill-repos.txt" | tr -d ' ')" "8"
# The manual step the document requires and revision 7 never exercised.
printf 'project-management\n' >> "$E/recovery/release-skill-repos.txt"
check "E3c after the operator's append, nine names" \
      "$(wc -l < "$E/recovery/release-skill-repos.txt" | tr -d ' ')" "9"

p4b() { ( HOME="$E/home" RECOVERY_ROOT="$E/recovery" bash "$P4B" ); }
while IFS= read -r sname; do mkdir -p "$E/home/.agents/skills/$sname"; done \
  < "$E/recovery/release-skill-repos.txt"
check "E4 P4b all nine real directories: OK lines" "$(p4b | grep -c '^OK')" "9"
check "E5 P4b all nine real directories: REFUSE lines" "$(p4b | grep -c '^REFUSE')" "0"
rm -rf "$E/home/.agents/skills/swiftui"; ln -s /tmp "$E/home/.agents/skills/swiftui"
check "E6 P4b one symlinked skill (the clone branch) REFUSEs" \
      "$(p4b | grep -c '^REFUSE')" "1"
check "E7 and names the shape it saw" \
      "$(p4b | grep '^REFUSE' | grep -c 'symlink')" "1"
rm -f "$E/home/.agents/skills/swiftui"
check "E8 P4b one absent skill REFUSEs" "$(p4b | grep -c '^REFUSE')" "1"
check "E9 and names it absent" "$(p4b | grep '^REFUSE' | grep -c 'absent')" "1"
printf 'x' > "$E/home/.agents/skills/swiftui"
check "E10 P4b a plain file REFUSEs" "$(p4b | grep -c '^REFUSE')" "1"
rm -f "$E/home/.agents/skills/swiftui"; mkdir -p "$E/home/.agents/skills/swiftui"
check "E11 P4b after re-materialising: no REFUSE (narrowing control)" \
      "$(p4b | grep -c '^REFUSE')" "0"
check "E12 and nine OK again (narrowing control)" "$(p4b | grep -c '^OK')" "9"

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
# The R1 block is located in the extracted fences BY CONTENT and run with bash,
# not retyped, against a disposable HOME with fake artifacts. No installer is
# executed. Locating by content is what stops a line-number shift in the
# document from silently pointing this suite at the wrong block or at nothing.
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
         "$RECOVERY_ROOT/pm/skills" "$RECOVERY_ROOT/pm/claude-skills" \
         "$RECOVERY_ROOT/pm/codex-skills" "$PM_RELEASE/scripts" "$PM_RELEASE/.roles"

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

# --- F2: R1 must refuse a snapshot it cannot restore BEFORE it mutates ------
# The control is revision 7's own R1 text, recovered from the committed
# document at HEAD and extracted the same way, so "red on the shipped text" is
# measured rather than asserted.
echo
# Section G exported a disposable HOME. Everything below reads the real host.
export HOME="$REAL_HOME"
echo "=== J. The recorded-but-unrestorable kind (revision-7 F2) ==="
OLDF="$SANDBOX/fences-rev7"; mkdir -p "$OLDF"
if git -C "$(dirname "$PLAN")/.." show "HEAD:.research/$(basename "$PLAN")" \
     > "$SANDBOX/plan-rev7.md" 2>/dev/null && [ -s "$SANDBOX/plan-rev7.md" ]; then
  python3 "$(dirname "$0")/extract-fences.py" "$SANDBOX/plan-rev7.md" "$OLDF" >/dev/null
  R1_OLD="$(grep -l 'r1-release-bins' "$OLDF"/*.bash | head -1)"
  if [ -f "$R1_OLD" ]; then ok "J0 revision-7 R1 recovered from HEAD ($(basename "$R1_OLD"))"
  else bad "J0" "revision-7 R1 fence not found"; R1_OLD=/dev/null; fi
else
  bad "J0" "could not read the revision-7 document from HEAD"; R1_OLD=/dev/null
fi

# Build a fixture whose ONLY unusual property is the recorded kind under test.
build_fixture() {   # $1 = target dir, $2 = .claude kind for the skill, $3 = stage it?
  local F="$1" kind="$2" stage="$3"
  rm -rf "$F"
  mkdir -p "$F/home/.local/bin" "$F/home/.roles/dev" "$F/home/.agents/skills/swiftui" \
           "$F/home/.claude/skills" "$F/home/.codex/skills" "$F/home/.config/task-board" \
           "$F/rec/bin" "$F/rec/manifest" "$F/rec/pm/roles/dev" "$F/rec/pm/skills/swiftui" \
           "$F/rec/pm/claude-skills" "$F/rec/pm/codex-skills" "$F/rel/scripts" "$F/rel/.roles/dev"
  printf '%s\n' "$F/home/.local/bin" > "$F/rec/manifest/bin-dir.txt"
  printf '%s\n' "$F/home/.config/task-board/install.json" > "$F/rec/manifest/pm-state-path.txt"
  for b in task-board tb-sessiond; do
    printf 'OLD-%s\n' "$b" > "$F/rec/bin/$b"
    printf 'NEW-%s\n' "$b" > "$F/home/.local/bin/$b"; chmod +x "$F/home/.local/bin/$b"
    printf '%s\n' "$b" >> "$F/rec/manifest/binaries.txt"
  done
  ls -1A "$F/home/.local/bin" > "$F/rec/manifest/bin-inventory.txt"
  printf 'OLD-ROLE\n' > "$F/rec/pm/roles/dev/role.md"
  printf 'NEW-ROLE\n' > "$F/home/.roles/dev/role.md"
  printf 'dev\n' > "$F/rec/manifest/roles.txt"
  printf 'v1\n' > "$F/rec/pm/skills/swiftui/SKILL.md"
  printf 'v2\n' > "$F/home/.agents/skills/swiftui/SKILL.md"
  printf 'swiftui\n' > "$F/rec/manifest/skills.txt"
  # ~/.claude entry: a REAL DIRECTORY, which install_skills rm -rf's and which
  # the snapshot is allowed to record. ~/.codex stays a symlink (control).
  mkdir -p "$F/home/.claude/skills/swiftui"; printf 'claude-v1\n' > "$F/home/.claude/skills/swiftui/SKILL.md"
  ln -sfn "$F/home/.agents/skills/swiftui" "$F/home/.codex/skills/swiftui"
  printf 'swiftui|directory||%s||symlink|%s\n' "$kind" "$F/home/.agents/skills/swiftui" \
    > "$F/rec/manifest/skill-state.txt"
  if [ "$stage" = stage ]; then
    rsync -aq --delete "$F/home/.claude/skills/swiftui/" "$F/rec/pm/claude-skills/swiftui/"
  fi
  printf '{"repoPath":"/old/repo","binDir":"%s"}\n' "$F/home/.local/bin" \
    > "$F/rec/manifest/../pm/install.json"
  printf '{"repoPath":"/release/candidate","binDir":"%s"}\n' "$F/home/.local/bin" \
    > "$F/home/.config/task-board/install.json"
  { echo '  install_binary "task-board" "$X"'
    echo '  install_binary "tb-sessiond" "$X"'
    echo '  local skill_name="project-management"'
    echo 'SKILL_REPOS=('; echo '  swiftui "git@example:swiftui.git"'; echo ')'
  } > "$F/rel/scripts/setup.sh"
}
run_r1() {   # $1 = block, $2 = fixture dir
  ( export HOME="$2/home" RECOVERY_ROOT="$2/rec" MANIFEST="$2/rec/manifest" \
           PM_RELEASE="$2/rel"; bash "$1" ) >"$2/r1.log" 2>&1
}

# J1/J2: revision 7's R1 against a recorded `directory` link-tree kind.
build_fixture "$SANDBOX/j1" directory nostage
run_r1 "$R1_OLD" "$SANDBOX/j1"; j1=$?
check "J1 revision-7 R1 on a recorded directory kind (non-zero == aborted)" \
      "$([ "$j1" -ne 0 ] && echo aborted || echo completed)" "aborted"
check "J2 and it had ALREADY mutated: executables restored before it aborted" \
      "$(cat "$SANDBOX/j1/home/.local/bin/task-board")" "OLD-task-board"
check "J3 and steps 6-7 never ran: the role is still the release's" \
      "$(cat "$SANDBOX/j1/home/.roles/dev/role.md")" "NEW-ROLE"
check "J4 and the install state is still the release's" \
      "$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["repoPath"])' \
         "$SANDBOX/j1/home/.config/task-board/install.json")" "/release/candidate"

# J5-J8: revision 8's R1 on the same fixture -- refusal BEFORE any mutation.
build_fixture "$SANDBOX/j2" directory nostage
run_r1 "$R1_BLOCK" "$SANDBOX/j2"; j2=$?
check "J5 revision-8 R1 on the same snapshot (non-zero == refused)" \
      "$([ "$j2" -ne 0 ] && echo refused || echo admitted)" "refused"
check "J6 and it refused at the door: executables untouched" \
      "$(cat "$SANDBOX/j2/home/.local/bin/task-board")" "NEW-task-board"
check "J7 and the role is untouched" \
      "$(cat "$SANDBOX/j2/home/.roles/dev/role.md")" "NEW-ROLE"
check "J8 and it said which entry it could not restore" \
      "$(grep -c 'UNRESTORABLE swiftui:.claude directory' "$SANDBOX/j2/r1.log")" "1"

# J9-J12: revision 8's R1 with the tree staged -- restores it (narrowing control).
build_fixture "$SANDBOX/j3" directory stage
run_r1 "$R1_BLOCK" "$SANDBOX/j3"
check "J9 revision-8 R1 with the directory staged (0 == admitted; control)" "$?" "0"
check "J10 and the link-tree entry is a directory again, not a symlink" \
      "$([ -d "$SANDBOX/j3/home/.claude/skills/swiftui" ] && \
         [ ! -L "$SANDBOX/j3/home/.claude/skills/swiftui" ] && echo directory || echo other)" "directory"
check "J11 and its content is the snapshot's" \
      "$(cat "$SANDBOX/j3/home/.claude/skills/swiftui/SKILL.md")" "claude-v1"
check "J12 and the rest of the rollback completed" \
      "$(cat "$SANDBOX/j3/home/.roles/dev/role.md")" "OLD-ROLE"

# J13-J15: the CLASS, not the instance -- a symlink kind with no recorded target.
build_fixture "$SANDBOX/j4" symlink nostage
run_r1 "$R1_BLOCK" "$SANDBOX/j4"; j4=$?
check "J13 revision-8 R1 on a symlink kind with an empty target (refused)" \
      "$([ "$j4" -ne 0 ] && echo refused || echo admitted)" "refused"
check "J14 and nothing was mutated" \
      "$(cat "$SANDBOX/j4/home/.local/bin/task-board")" "NEW-task-board"
# Revision 7 on the same input does not abort -- it FAILS OPEN. `ln -sfn ""`
# exits 0 and creates a dangling symlink, so R1 reports a successful rollback
# while destroying the tree it was restoring. Worse than F2's abort, same class.
run_r1 "$R1_OLD" "$SANDBOX/j4"; j4o=$?
check "J15 revision-7 R1 on the same input (0 == reported success)" "$j4o" "0"
check "J16 and it left a dangling symlink with an empty target" \
      "$([ -L "$SANDBOX/j4/home/.claude/skills/swiftui" ] && \
         [ -z "$(readlink "$SANDBOX/j4/home/.claude/skills/swiftui")" ] && \
         [ ! -e "$SANDBOX/j4/home/.claude/skills/swiftui" ] && echo dangling || echo other)" "dangling"
check "J17 and the content it was restoring is gone" \
      "$([ -f "$SANDBOX/j4/home/.claude/skills/swiftui/SKILL.md" ] && echo present || echo destroyed)" "destroyed"

# J16-J18: a missing saved ROLE -- never covered by any earlier probe.
build_fixture "$SANDBOX/j5" symlink stage
printf 'swiftui|directory||symlink|%s|symlink|%s\n' \
  "$SANDBOX/j5/home/.agents/skills/swiftui" "$SANDBOX/j5/home/.agents/skills/swiftui" \
  > "$SANDBOX/j5/rec/manifest/skill-state.txt"
rm -rf "$SANDBOX/j5/rec/pm/roles/dev"
run_r1 "$R1_BLOCK" "$SANDBOX/j5"; j5=$?
check "J18 revision-8 R1 with a saved role missing (refused)" \
      "$([ "$j5" -ne 0 ] && echo refused || echo admitted)" "refused"
check "J19 and it refused before touching the executables" \
      "$(cat "$SANDBOX/j5/home/.local/bin/task-board")" "NEW-task-board"
run_r1 "$R1_OLD" "$SANDBOX/j5"; j5o=$?
check "J20 revision-7 R1 on the same input restored executables first, then died" \
      "$([ "$j5o" -ne 0 ] && [ "$(cat "$SANDBOX/j5/home/.local/bin/task-board")" = OLD-task-board ] \
         && echo aborted-after-mutating || echo other)" "aborted-after-mutating"

# J19: narrowing control -- a well-formed snapshot is still admitted.
build_fixture "$SANDBOX/j6" symlink stage
printf 'swiftui|directory||symlink|%s|symlink|%s\n' \
  "$SANDBOX/j6/home/.agents/skills/swiftui" "$SANDBOX/j6/home/.agents/skills/swiftui" \
  > "$SANDBOX/j6/rec/manifest/skill-state.txt"
run_r1 "$R1_BLOCK" "$SANDBOX/j6"
check "J21 revision-8 R1 on a well-formed snapshot (0; narrowing control)" "$?" "0"
check "J22 and it restored" "$(cat "$SANDBOX/j6/home/.local/bin/task-board")" "OLD-task-board"

# J21: the snapshot block carries the same admissibility text as R1.
SNAP_BLOCK="$(grep -l 'test ! -e "\$RECOVERY_ROOT"' "$(dirname "$0")"/fences/*.bash | head -1)"
if [ -f "$SNAP_BLOCK" ]; then ok "J23a snapshot fence located ($(basename "$SNAP_BLOCK"))"
else bad "J21a" "snapshot fence not found"; SNAP_BLOCK=/dev/null; fi
check "J23 snapshot and R1 both define snapshot_restorable with all four arms" \
      "$(for f in "$SNAP_BLOCK" "$R1_BLOCK"; do
           grep -c 'snapshot_restorable()' "$f"; done | tr '\n' ' ')" "1 1 "
check "J24 and both stage/consume the two link-tree trees" \
      "$(for f in "$SNAP_BLOCK" "$R1_BLOCK"; do
           grep -c 'codex-skills' "$f" >/dev/null && echo yes; done | tr '\n' ' ')" "yes yes "

echo
echo "=== K. The daemon census, driven as the document's own P7a fence ==="
P7A="$(grep -l 'daemons-before.txt' "$(dirname "$0")"/fences/*.bash | head -1)"
if [ -f "$P7A" ]; then ok "K0 P7a fence located ($(basename "$P7A"))"
else bad "K0" "P7a fence not found"; P7A=/dev/null; fi
K="$SANDBOX/k"; mkdir -p "$K/bin" "$K/rec"
mkpgrep() {   # $1 = exit code, $2 = stdout
  printf '#!/bin/sh\nprintf %%s "$2"\nexit %s\n' "$1" > "$K/bin/pgrep"
  printf '#!/bin/sh\n%s\nexit %s\n' "$2" "$1" > "$K/bin/pgrep"; chmod +x "$K/bin/pgrep"
}
runp7a() { ( export PATH="$K/bin:$PATH" RECOVERY_ROOT="$K/rec"; rm -f "$K/rec/daemons-before.txt"; \
             bash "$P7A" ) >"$K/out.txt" 2>&1; }
mkpgrep 0 'printf "111 /x/tb-sessiond --board-dir /b/one\n222 /x/tb-sessiond --board-dir /b/two\n"'
runp7a; check "K1 pgrep exit 0 (daemons present)" "$?" "0"
check "K2 and both lines were recorded" "$(wc -l < "$K/rec/daemons-before.txt" | tr -d ' ')" "2"
mkpgrep 1 'true'
runp7a; check "K3 pgrep exit 1 is a LEGITIMATE ABSENCE, not a failure" "$?" "0"
check "K4 and it is recorded as such" "$(cat "$K/rec/daemons-before.txt")" "NO-LIVE-DAEMON"
mkpgrep 2 'echo "pgrep: bad option" >&2'
runp7a; k5=$?
check "K5 pgrep exit 2 is a FAILED READ and stops the step" \
      "$([ "$k5" -ne 0 ] && echo refused || echo admitted)" "refused"
check "K6 and it did not publish NO-LIVE-DAEMON over the failure" \
      "$(cat "$K/rec/daemons-before.txt" 2>/dev/null | grep -c 'NO-LIVE-DAEMON')" "0"
check "K7 and it named the failure" "$(grep -c 'DAEMON-CENSUS-FAILED rc=2' "$K/out.txt")" "1"
mkpgrep 3 'echo "pgrep: fatal" >&2'
runp7a; k8=$?
check "K8 pgrep exit 3 likewise stops the step (class, not one code)" \
      "$([ "$k8" -ne 0 ] && echo refused || echo admitted)" "refused"
# Narrowing control: the real pgrep on the real host.
( export RECOVERY_ROOT="$K/rec"; rm -f "$K/rec/daemons-before.txt"; bash "$P7A" ) >/dev/null 2>&1
check "K9 real pgrep on the target host (0 == daemons present; narrowing control)" "$?" "0"
check "K10 and it saw both live daemons" \
      "$(grep -c 'tb-sessiond' "$K/rec/daemons-before.txt")" "2"
# The board extraction the document performs on that output.
sed -nE 's/.*--board-dir[ =]([^ ]+).*/\1/p' "$K/rec/daemons-before.txt" | sort -u > "$K/boards.txt"
check "K11 board paths extracted from the daemons' own argv" \
      "$(wc -l < "$K/boards.txt" | tr -d ' ')" "2"

echo
echo "=== I. tb-sessiond contract facts, derived from production text ==="
check "I1 controlProtocolVersion is a compile-time constant" \
      "$(grep -cE '^\tcontrolProtocolVersion = [0-9]+$' "$SM/types.go")" "1"
check "I2 its current value" \
      "$(sed -nE 's/^\tcontrolProtocolVersion = ([0-9]+)$/\1/p' "$SM/types.go")" "4"
check "I3 --version does not report it (so it must come from source)" \
      "$(~/.local/bin/task-board --version | grep -ci 'protocol')" "0"
check "I4 the daemon reports protocol_version over its status endpoint" \
      "$(grep -c 'ProtocolVersion *int *.json:"protocol_version,omitempty"' "$SM/types.go")" "1"
check "I5 managerProtocolCompatibility has an older-daemon branch" \
      "$(sed -n '/^func managerProtocolCompatibility/,/^}/p' "$SM/client.go" \
         | grep -c 'actual < controlProtocolVersion')" "1"
check "I6 and a newer-daemon branch" \
      "$(sed -n '/^func managerProtocolCompatibility/,/^}/p' "$SM/client.go" \
         | grep -c 'actual > controlProtocolVersion')" "1"
check "I7 the newer-daemon branch is a refusal with no recovery" \
      "$(sed -n '/^func managerProtocolCompatibility/,/^}/p' "$SM/client.go" \
         | grep -c 'is newer than this wrapper supports')" "1"
check "I8 takeOverOutdatedManager refuses the downgrade direction explicitly" \
      "$(grep -c 'refusing protocol takeover without an older daemon version' "$SM/stale_takeover.go")" "1"
check "I9 ConnectOrStart reaches the takeover from production, not a helper" \
      "$(sed -n '/^func ConnectOrStart(/,/^}/p' "$SM/client.go" \
         | grep -c 'takeOverOutdatedManager(ctx, layout, upgrade)')" "1"
check "I10 the takeover drains before it fences" \
      "$(sed -n '/^func takeOverManagerForProtocolUpgrade/,/^}/p' "$SM/stale_takeover.go" \
         | grep -c 'client.Drain(ctx)')" "1"
check "I11 and falls through to a fenced terminate when it cannot drain" \
      "$(sed -n '/^func takeOverManagerForProtocolUpgrade/,/^}/p' "$SM/stale_takeover.go" \
         | grep -c 'terminateStaleInstance(ctx, layout, recorded)')" "1"
check "I12 Drain is reachable from NO cobra command (so no operator can drain)" \
      "$(grep -rl '\.Drain(' "$CMD" 2>/dev/null | wc -l | tr -d ' ')" "0"
check "I13 Dial performs no compatibility check (which is why it is the probe)" \
      "$(sed -n '/^func Dial(/,/^}/p' "$SM/client.go" | grep -c 'managerProtocolCompatibility')" "0"
check "I14 session status uses Dial, not ConnectOrStart" \
      "$(sed -n '/Use:   "status"/,/^}/p' "$CMD/session.go" | grep -c 'sessionmanager.Dial(layout)')" "1"
check "I15 the status decoder is lenient (no DisallowUnknownFields)" \
      "$(grep -c 'DisallowUnknownFields' "$SM/client.go")" "0"
check "I16 the daemon executable is resolved through PATH by bare name" \
      "$(grep -c 'exec.LookPath(sessionManagerDaemonExecutable)' "$CMD/session.go")" "1"
check "I17 and that bare name is tb-sessiond" \
      "$(sed -nE 's/^var sessionManagerDaemonExecutable = "(.*)"$/\1/p' "$CMD/session.go")" "tb-sessiond"
check "I18 the daemon's board comes from its own argv" \
      "$(grep -c '"--board-dir", layout.BoardKey.Canonical' "$CMD/session.go")" "1"
check "I19 goal-bound spawn reaches the daemon through ConnectOrStart" \
      "$(grep -c 'return connectOrStartSessionManager(ctx, layout)' "$CMD/codex_goal_spawn.go")" "1"
check "I20 and launchTrackedSpawnRun routes those runs there" \
      "$(sed -n '/^func launchTrackedSpawnRun/,/^}/p' "$CMD/codex_goal_spawn.go" \
         | grep -c 'launchManagedGoalBoundRun(cfg)')" "1"
# Protocol history: four distinct values in five days.
PROTO_VALUES="$(cd "$PM_SRC" && for c in d66bfea8 f8ba3268 445b391e df4201dc; do
  git show "${c}:tools/board-cli/internal/sessionmanager/types.go" \
    | sed -nE 's/.*controlProtocolVersion += +(.*)$/\1/p' | head -1
done | tr '\n' ' ')"
check "I21 the constant's four historical values" "$PROTO_VALUES" \
      '4 3 2 "task-board-session-manager-v1" '

echo
echo "--- live daemon census on the target host (read-only) ---"
DSTAT="$SANDBOX/daemon-status.jsonl"; : > "$DSTAT"
dboards="$(sed -nE 's/.*--board-dir[ =]([^ ]+).*/\1/p' "$K/rec/daemons-before.txt" | sort -u)"
dcount=0
while IFS= read -r b; do
  [ -n "$b" ] || continue
  task-board --board-dir "$b" --json session status >> "$DSTAT" 2>/dev/null || \
    { bad "I22 status read for $b"; continue; }
  dcount=$((dcount+1))
done <<< "$dboards"
check "I22 every live daemon answered session status" "$dcount" "2"
check "I23 both report protocol_version 4" \
      "$(grep -c '"protocol_version":4' "$DSTAT")" "2"
INSTALLED_TB="$(~/.local/bin/task-board --version | sed -nE 's/^task-board version ([^ ]+) .*/\1/p')"
check "I24 neither daemon's image equals the INSTALLED file's version" \
      "$(grep -c "\"daemon_version\":\"$INSTALLED_TB\"" "$DSTAT")" "0"
check "I25 both are running some older image (measured skew, today)" \
      "$(grep -c '"daemon_version":"0.24.3-1' "$DSTAT")" "2"
check "I26 and they hold sessions a takeover would drain or fence" \
      "$([ "$(sed -nE 's/.*"session_count":([0-9]+).*/\1/p' "$DSTAT" | paste -sd+ - | bc)" -gt 0 ] \
         && echo yes || echo no)" "yes"
# Narrowing control: an absent daemon is distinguishable from a failed read.
task-board --json session status > "$SANDBOX/own-board.json" 2>/dev/null
check "I27 this migration's own board: exit 0 (a read that succeeded)" "$?" "0"
check "I28 and reports running:false -- a legitimate absence, not unknown" \
      "$(grep -c '"running":false' "$SANDBOX/own-board.json")" "1"
check "I29 so an installed canary here could not observe either live daemon" \
      "$(grep -c 'protocol_version' "$SANDBOX/own-board.json")" "0"

echo
echo "shell=$1 pass=$pass fail=$fail"
test "$fail" -eq 0
