#!/usr/bin/env bash
# Targeted red probe against revision 5's restore_project_management_surface.
# Revision 5 has no snapshot manifest, so its snapshot layout is built by hand
# exactly as its own snapshot block would have written it, and its restore is
# extracted verbatim and executed against the same host state the revision-6
# end-to-end probe uses.
PLAN="$1"; OUT="$2"
EXTRACT="$(cd "$(dirname "$0")" && pwd)/extract-plan-code.py"
REAL_PM=/Users/alexis/src/relux-works/skill-project-management
rm -rf "$OUT"; mkdir -p "$OUT"; OUT="$(cd "$OUT" && pwd)"
FAILURES=0
record() {
  if [ "$2" = "$3" ]; then printf 'PASS | %-46s | expected=%s actual=%s\n' "$1" "$2" "$3"
  else printf 'FAIL | %-46s | expected=%s actual=%s\n' "$1" "$2" "$3"; FAILURES=$((FAILURES+1)); fi
}

export HOME="$OUT/home"
mkdir -p "$HOME/.agents/skills" "$HOME/.claude/skills" "$HOME/.codex/skills" \
  "$HOME/.roles" "$HOME/.config/task-board"
for role in $(ls -1A "$REAL_PM/.roles"); do
  mkdir -p "$HOME/.roles/$role"; printf 'role %s v1\n' "$role" > "$HOME/.roles/$role/ROLE.md"
done
for skill in project-management $(awk '/^SKILL_REPOS=\(/{f=1;next} f&&/^\)/{f=0} f&&NF>=2{print $1}' "$REAL_PM/scripts/setup.sh"); do
  mkdir -p "$HOME/.agents/skills/$skill"
  printf 'skill %s v1\n' "$skill" > "$HOME/.agents/skills/$skill/SKILL.md"
  ln -sfn "$HOME/.agents/skills/$skill" "$HOME/.claude/skills/$skill"
  ln -sfn "$HOME/.agents/skills/$skill" "$HOME/.codex/skills/$skill"
done
printf '{"repoPath":"/x","installedSkillPath":"/y","binDir":"/z"}\n' \
  > "$HOME/.config/task-board/install.json"

# Revision 5's snapshot layout, built by hand from its own snapshot block.
RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/current-pair"
mkdir -p "$RECOVERY_ROOT/pm/roles"
rsync -a --delete "$HOME/.agents/skills/project-management/" \
  "$RECOVERY_ROOT/pm/project-management-skill/"
for role in $(ls -1A "$HOME/.roles"); do
  rsync -a --delete "$HOME/.roles/$role/" "$RECOVERY_ROOT/pm/roles/$role/"
done
cp "$HOME/.config/task-board/install.json" "$RECOVERY_ROOT/pm/install.json"

# What a release that adds a role and a dependency skill leaves behind.
mkdir -p "$HOME/.roles/release-only-role"
printf 'role release-only-role v1\n' > "$HOME/.roles/release-only-role/ROLE.md"
mkdir -p "$HOME/.agents/skills/release-only-skill"
printf 'skill release-only-skill v1\n' > "$HOME/.agents/skills/release-only-skill/SKILL.md"
ln -sfn "$HOME/.agents/skills/release-only-skill" "$HOME/.claude/skills/release-only-skill"
ln -sfn "$HOME/.agents/skills/release-only-skill" "$HOME/.codex/skills/release-only-skill"
printf 'skill go-testing-tools v2-from-release\n' \
  > "$HOME/.agents/skills/go-testing-tools/SKILL.md"

text="$(python3 "$EXTRACT" "$PLAN" fn restore_project_management_surface)" || {
  echo "restore_project_management_surface absent from $PLAN"; exit 3; }
eval "$text"
restore_project_management_surface > "$OUT/restore.log" 2>&1
record "revision-5 restore runs" "0" "$?"

[ -e "$HOME/.roles/release-only-role" ] && got=present || got=absent
record "release-added role removed" "absent" "$got"
[ -e "$HOME/.agents/skills/release-only-skill" ] && got=present || got=absent
record "release-added skill removed" "absent" "$got"
[ -L "$HOME/.claude/skills/release-only-skill" ] && got=present || got=absent
record "release-added ~/.claude link removed" "absent" "$got"
got="$(cat "$HOME/.agents/skills/go-testing-tools/SKILL.md")"
record "rewritten dependency skill restored" "skill go-testing-tools v1" "$got"
got="$(cat "$HOME/.roles/developer/ROLE.md")"
record "existing role restored" "role developer v1" "$got"

printf 'failures=%s\n' "$FAILURES"
[ "$FAILURES" -eq 0 ]
