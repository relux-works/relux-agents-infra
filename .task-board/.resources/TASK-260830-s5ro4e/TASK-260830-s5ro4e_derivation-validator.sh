#!/usr/bin/env bash
# Attack the plan's derivation and restore construction. Runs under bash and
# zsh. Every function under test is extracted verbatim from the plan document
# named by $1, never retyped, so a probe cannot pass against a copy the plan
# does not contain.
#
#   validate-revision6.sh PLAN.md OUTDIR
#
# Exit 0 only when every probe reached its required verdict.

PLAN="$1"
OUT="$2"
HERE="$(cd "$(dirname "$0")" && pwd)"
EXTRACT="$HERE/extract-plan-code.py"
REAL_PM=/Users/alexis/src/relux-works/skill-project-management
REAL_AI=/Users/alexis/src/relux-works/relux-agents-infra
ORIGINAL_HOME="$HOME"

rm -rf "$OUT"; mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"
FAILURES=0
PROBES=0

say() { printf '%s\n' "$*"; }

record() {
  # record NAME EXPECTED ACTUAL
  PROBES=$((PROBES + 1))
  if [ "$2" = "$3" ]; then
    printf 'PASS | %-52s | expected=%s actual=%s\n' "$1" "$2" "$3"
  else
    printf 'FAIL | %-52s | expected=%s actual=%s\n' "$1" "$2" "$3"
    FAILURES=$((FAILURES + 1))
  fi
}

load_fn() {
  # load_fn NAME -> defines it, or returns 3 when the revision lacks it
  _text="$(python3 "$EXTRACT" "$PLAN" fn "$1")" || return 3
  eval "$_text"
}

load_block() {
  python3 "$EXTRACT" "$PLAN" block "$1"
}

have_fn() {
  python3 "$EXTRACT" "$PLAN" fn "$1" >/dev/null 2>&1
}

# --------------------------------------------------------------------------
# Sandbox: a disposable HOME plus two disposable Git repositories carrying the
# real production installer text, so every derivation is exercised against the
# bytes production actually ships.
# --------------------------------------------------------------------------
build_sandbox() {
  SB="$OUT/$1"
  rm -rf "$SB"
  mkdir -p "$SB/repos/pm/scripts/lib" "$SB/repos/ai/scripts"
  export HOME="$SB/home"
  mkdir -p "$HOME/.local/bin" "$HOME/.config/task-board" "$HOME/.roles" \
    "$HOME/.agents/skills" "$HOME/.claude/skills" "$HOME/.codex/skills" \
    "$HOME/Library/Application Support/agents-infra" "$HOME/.local/state"

  cp "$REAL_PM/scripts/setup.sh" "$SB/repos/pm/scripts/setup.sh"
  cp "$REAL_PM/scripts/lib/agents-infra-compose.zsh" "$SB/repos/pm/scripts/lib/"
  cp "$REAL_AI/scripts/setup.sh" "$SB/repos/ai/scripts/setup.sh"

  for role in $(ls -1A "$REAL_PM/.roles"); do
    mkdir -p "$SB/repos/pm/.roles/$role"
    printf 'role %s v1\n' "$role" > "$SB/repos/pm/.roles/$role/ROLE.md"
    mkdir -p "$HOME/.roles/$role"
    printf 'role %s v1\n' "$role" > "$HOME/.roles/$role/ROLE.md"
  done

  # Managed skills on the sandbox host: the registered skill plus every
  # SKILL_REPOS dependency, in the already-installed shape the real host has.
  for skill in project-management $(awk '/^SKILL_REPOS=\(/{f=1;next} f&&/^\)/{f=0} f&&NF>=2{print $1}' "$SB/repos/pm/scripts/setup.sh"); do
    mkdir -p "$HOME/.agents/skills/$skill"
    printf 'skill %s v1\n' "$skill" > "$HOME/.agents/skills/$skill/SKILL.md"
    ln -sfn "$HOME/.agents/skills/$skill" "$HOME/.claude/skills/$skill"
    ln -sfn "$HOME/.agents/skills/$skill" "$HOME/.codex/skills/$skill"
  done

  for repo in pm ai; do
    git -C "$SB/repos/$repo" init -q
    git -C "$SB/repos/$repo" add -A
    git -C "$SB/repos/$repo" -c user.email=probe@example.invalid \
      -c user.name=probe commit -q -m baseline
  done
  PM_COMMIT="$(git -C "$SB/repos/pm" rev-parse --short=12 HEAD)"
  AI_COMMIT="$(git -C "$SB/repos/ai" rev-parse --short=12 HEAD)"

  for name in task-board tb-sessiond task-board-tui openai-board \
    anthropic-board; do
    printf '#!/bin/sh\nprintf "%s 0.0.0-test (commit %s, built now)\\n"\n' \
      "$name" "$PM_COMMIT" > "$HOME/.local/bin/$name"
    chmod +x "$HOME/.local/bin/$name"
  done
  # The agents-infra stub answers `version`, and accepts the setup/verify calls
  # the restore makes, recording that they happened.
  cat > "$HOME/.local/bin/agents-infra" <<STUB
#!/bin/sh
if [ "\$1" = version ]; then
  printf 'agents-infra 0.0.0-test (commit $AI_COMMIT, built now)\n'
  exit 0
fi
printf '%s\n' "\$*" >> "\$HOME/agents-infra-calls.log"
exit 0
STUB
  chmod +x "$HOME/.local/bin/agents-infra"
  printf '#!/bin/sh\nexit 0\n' > "$HOME/.local/bin/model-harness"
  chmod +x "$HOME/.local/bin/model-harness"

  cat > "$HOME/.config/task-board/install.json" <<JSON
{
  "repoPath": "$SB/repos/pm",
  "installedSkillPath": "$HOME/.agents/skills/project-management",
  "binDir": "$HOME/.local/bin"
}
JSON
  cat > "$HOME/Library/Application Support/agents-infra/install.json" <<JSON
{
  "repoPath": "$SB/repos/ai",
  "binDir": "$HOME/.local/bin",
  "platform": "darwin",
  "arch": "arm64",
  "version": "0.0.0-test",
  "commit": "$AI_COMMIT",
  "buildDate": "2026-08-30T00:00:00Z"
}
JSON
}

# A release tree that adds one role, one dependency skill and one executable.
build_release_tree() {
  RELEASE="$SB/release-pm"
  rm -rf "$RELEASE"
  cp -R "$SB/repos/pm" "$RELEASE"
  rm -rf "$RELEASE/.git"
  mkdir -p "$RELEASE/.roles/release-only-role"
  printf 'role release-only-role v1\n' \
    > "$RELEASE/.roles/release-only-role/ROLE.md"
  python3 - "$RELEASE/scripts/setup.sh" <<'INJECT'
import sys, pathlib
p = pathlib.Path(sys.argv[1])
lines = p.read_text().split(chr(10))
out, inside, added_skill, added_bin = [], False, False, False
for line in lines:
    if line.startswith("SKILL_REPOS=("):
        inside = True
        out.append(line)
        continue
    if inside and line.startswith(")"):
        out.append('  release-only-skill "git@github.com:relux-works/skill-release-only.git"')
        added_skill = True
        inside = False
        out.append(line)
        continue
    out.append(line)
    if line.strip().startswith('install_binary "anthropic-board"'):
        out.append('  install_binary "release-only-bin" "$BOARD_LAUNCHER_DIR/release-only-bin"')
        added_bin = True
assert added_skill and added_bin, (added_skill, added_bin)
p.write_text(chr(10).join(out))
INJECT
}

# Simulate what that release's install did to the sandbox host.
apply_release_effects() {
  mkdir -p "$HOME/.roles/release-only-role"
  printf 'role release-only-role v1\n' \
    > "$HOME/.roles/release-only-role/ROLE.md"
  mkdir -p "$HOME/.agents/skills/release-only-skill"
  printf 'skill release-only-skill v1\n' \
    > "$HOME/.agents/skills/release-only-skill/SKILL.md"
  ln -sfn "$HOME/.agents/skills/release-only-skill" \
    "$HOME/.claude/skills/release-only-skill"
  ln -sfn "$HOME/.agents/skills/release-only-skill" \
    "$HOME/.codex/skills/release-only-skill"
  printf '#!/bin/sh\nexit 0\n' > "$HOME/.local/bin/release-only-bin"
  chmod +x "$HOME/.local/bin/release-only-bin"
  # The release also rewrote an existing dependency skill and an executable.
  printf 'skill go-testing-tools v2-from-release\n' \
    > "$HOME/.agents/skills/go-testing-tools/SKILL.md"
  printf '#!/bin/sh\necho v2-from-release\n' > "$HOME/.local/bin/task-board"
  chmod +x "$HOME/.local/bin/task-board"
  # And it rewrote both machine-scoped install states to name itself.
  python3 - "$HOME/.config/task-board/install.json" "$RELEASE" <<'PY'
import json, sys
p = sys.argv[1]
state = json.load(open(p))
state["repoPath"] = sys.argv[2]
json.dump(state, open(p, "w"), indent=2)
PY
}

# ==========================================================================
say "== revision under test: $PLAN"
say "== shell: $(ps -o comm= -p $$ 2>/dev/null || echo unknown)"
say ""

# --------------------------------------------------------------------------
# P1..P6 — derivation helpers against the real production installer text.
# --------------------------------------------------------------------------
if have_fn derive_pm_binary_names && have_fn derive_ai_binary_names \
  && have_fn derive_pm_managed_skill_names && have_fn derive_pm_role_names \
  && have_fn extract_function_body && have_fn derive_pm_setup_files \
  && have_fn derive_pm_setup_stages; then
  load_fn derive_pm_binary_names
  load_fn derive_ai_binary_names
  load_fn derive_pm_managed_skill_names
  load_fn derive_pm_role_names
  load_fn extract_function_body
  load_fn derive_pm_setup_files
  load_fn derive_pm_setup_stages

  build_sandbox derivations

  got="$(derive_pm_binary_names "$SB/repos/pm" | tr '\n' ' ')"; st=$?
  record "derive_pm_binary_names on production text" \
    "task-board tb-sessiond task-board-tui openai-board anthropic-board " "$got"

  got="$(derive_ai_binary_names "$SB/repos/ai" | tr '\n' ' ')"
  record "derive_ai_binary_names on production text" \
    "agents-infra model-harness " "$got"

  got="$(derive_pm_managed_skill_names "$SB/repos/pm" | wc -l | tr -d ' ')"
  record "derive_pm_managed_skill_names counts 1+8" "9" "$got"

  got="$(derive_pm_managed_skill_names "$SB/repos/pm" | head -1)"
  record "derived skill set starts with the registered skill" \
    "project-management" "$got"

  got="$(derive_pm_role_names "$SB/repos/pm" | wc -l | tr -d ' ')"
  expected="$(ls -1A "$REAL_PM/.roles" | wc -l | tr -d ' ')"
  record "derive_pm_role_names matches the .roles directory" \
    "$expected" "$got"

  # The finding this revision exists for: the stage sequence must expose the
  # unconditional install_skills stage.
  stages="$(derive_pm_setup_stages "$SB/repos/pm")"
  if printf '%s\n' "$stages" | grep -qx install_skills; then got=present
  else got=absent; fi
  record "derived W2/W5 stage sequence contains install_skills" "present" "$got"

  got="$(printf '%s\n' "$stages" | grep -n . | grep -E ':(install_skills|verify)$' | cut -d: -f1 | tr '\n' ' ')"
  record "install_skills precedes verify in the derived sequence" "13 14 " "$got"

  # Failed reads must refuse, not report an empty satisfied set.
  derive_pm_binary_names "$SB/nonexistent" >/dev/null 2>&1
  record "derive_pm_binary_names refuses an unreadable tree" "70" "$?"
  derive_pm_role_names "$SB/nonexistent" >/dev/null 2>&1
  record "derive_pm_role_names refuses an unreadable tree" "70" "$?"
  derive_pm_managed_skill_names "$SB/nonexistent" >/dev/null 2>&1
  record "derive_pm_managed_skill_names refuses unreadable tree" "70" "$?"
  cp "$SB/repos/pm/scripts/setup.sh" "$SB/setup-backup.sh"
  grep -v 'install_binary "' "$SB/setup-backup.sh" > "$SB/repos/pm/scripts/setup.sh"
  derive_pm_binary_names "$SB/repos/pm" >/dev/null 2>&1
  record "derive_pm_binary_names refuses an empty derived set" "70" "$?"
  cp "$SB/setup-backup.sh" "$SB/repos/pm/scripts/setup.sh"
else
  record "derivation helpers present in this revision" "present" "absent"
fi

# --------------------------------------------------------------------------
# P7 — install-state reads: a failed read is not an absence.
# --------------------------------------------------------------------------
if have_fn read_install_state_field; then
  load_fn read_install_state_field
  got="$(read_install_state_field "$HOME/.config/task-board/install.json" repoPath)"
  record "read_install_state_field reads repoPath" "$SB/repos/pm" "$got"
  read_install_state_field "$HOME/nope.json" repoPath >/dev/null 2>&1
  record "read_install_state_field refuses a missing file" "70" "$?"
  printf 'not json' > "$SB/broken.json"
  read_install_state_field "$SB/broken.json" repoPath >/dev/null 2>&1
  record "read_install_state_field refuses malformed JSON" "70" "$?"
  printf '{"binDir": "/x"}' > "$SB/partial.json"
  read_install_state_field "$SB/partial.json" repoPath >/dev/null 2>&1
  record "read_install_state_field refuses a missing key" "70" "$?"
  if read_install_state_field "$HOME/nope.json" repoPath >/dev/null 2>&1; then
    got=accepted; else got=refused; fi
  record "refusal survives an if-condition caller" "refused" "$got"
else
  record "read_install_state_field present in this revision" "present" "absent"
fi

# --------------------------------------------------------------------------
# P8 — the dependency-skill precondition.
# --------------------------------------------------------------------------
if have_fn require_dependency_skills_materialized; then
  load_fn require_dependency_skills_materialized
  require_dependency_skills_materialized "$SB/repos/pm" >/dev/null 2>&1
  record "precondition passes when all skills are real dirs" "0" "$?"
  mv "$HOME/.agents/skills/go-testing-tools" "$SB/moved-skill"
  ln -sfn "$SB/moved-skill" "$HOME/.agents/skills/go-testing-tools"
  require_dependency_skills_materialized "$SB/repos/pm" >/dev/null 2>&1
  record "precondition refuses a symlinked dependency skill" "74" "$?"
  rm -f "$HOME/.agents/skills/go-testing-tools"
  require_dependency_skills_materialized "$SB/repos/pm" >/dev/null 2>&1
  record "precondition refuses an absent dependency skill" "74" "$?"
  cp -R "$SB/moved-skill" "$HOME/.agents/skills/go-testing-tools"
  require_dependency_skills_materialized "$SB/repos/pm" >/dev/null 2>&1
  record "precondition passes again once materialized" "0" "$?"
  require_dependency_skills_materialized "$SB/nonexistent" >/dev/null 2>&1
  record "precondition refuses when the map cannot be read" "70" "$?"
else
  record "require_dependency_skills_materialized present" "present" "absent"
fi

# --------------------------------------------------------------------------
# P9 — mirror guard.
# --------------------------------------------------------------------------
if have_fn require_mirrored_function_unchanged; then
  load_fn require_mirrored_function_unchanged
  require_mirrored_function_unchanged "$SB/repos/pm/scripts/setup.sh" \
    "$SB/repos/pm/scripts/setup.sh" install_skills >/dev/null 2>&1
  record "mirror guard accepts an unchanged install_skills" "0" "$?"
  sed 's/^install_skills() {/install_skills() {\n  : changed/' \
    "$SB/repos/pm/scripts/setup.sh" > "$SB/changed-setup.sh"
  require_mirrored_function_unchanged "$SB/repos/pm/scripts/setup.sh" \
    "$SB/changed-setup.sh" install_skills >/dev/null 2>&1
  record "mirror guard refuses a changed install_skills" "73" "$?"
  require_mirrored_function_unchanged "$SB/repos/pm/scripts/setup.sh" \
    "$SB/nonexistent.sh" install_skills >/dev/null 2>&1
  record "mirror guard refuses an unreadable candidate" "70" "$?"
  require_mirrored_function_unchanged "$SB/repos/ai/scripts/setup.sh" \
    "$SB/repos/ai/scripts/setup.sh" install_lldb_mcp >/dev/null 2>&1
  record "mirror guard reads install_lldb_mcp" "0" "$?"
  require_mirrored_function_unchanged "$SB/repos/pm/scripts/setup.sh" \
    "$SB/repos/pm/scripts/setup.sh" no_such_function >/dev/null 2>&1
  record "mirror guard refuses an absent function" "70" "$?"
else
  record "require_mirrored_function_unchanged present" "present" "absent"
fi

# --------------------------------------------------------------------------
# P10 — environment-override classification.
# --------------------------------------------------------------------------
if have_fn require_env_overrides_classified; then
  load_fn derive_setup_env_override_names
  load_fn classify_pm_env_override
  load_fn classify_ai_env_override
  load_fn require_env_overrides_classified
  require_env_overrides_classified "$SB/repos/pm/scripts/setup.sh" \
    classify_pm_env_override >/dev/null 2>&1
  record "every pm override in production text is classified" "0" "$?"
  require_env_overrides_classified "$SB/repos/ai/scripts/setup.sh" \
    classify_ai_env_override >/dev/null 2>&1
  record "every ai override in production text is classified" "0" "$?"
  sed 's|^BOARD_MODE=.*|BOARD_MODE="${TASK_BOARD_FUTURE_FLAG:-local}"|' \
    "$SB/repos/pm/scripts/setup.sh" > "$SB/future-setup.sh"
  require_env_overrides_classified "$SB/future-setup.sh" \
    classify_pm_env_override >/dev/null 2>&1
  record "an unclassified new override refuses the step" "75" "$?"
  require_env_overrides_classified "$SB/nonexistent.sh" \
    classify_pm_env_override >/dev/null 2>&1
  record "classification refuses an unreadable installer" "70" "$?"
else
  record "require_env_overrides_classified present" "present" "absent"
fi

# --------------------------------------------------------------------------
# P11 — end to end: snapshot, a release that adds artifacts, rollback.
# --------------------------------------------------------------------------
if have_fn restore_project_management_surface \
  && python3 "$EXTRACT" "$PLAN" block 'PM_STATE="$HOME/.config/task-board/install.json"' >/dev/null 2>&1; then
  build_sandbox endtoend
  for fn_name in read_install_state_field derive_pm_binary_names \
    derive_ai_binary_names derive_pm_role_names derive_pm_managed_skill_names \
    derive_ai_config_dir extract_reported_commit \
    require_saved_binary_source_identity materialize_recovery_source \
    snapshot_bin_dir restore_binary restore_binaries_for_release \
    restore_roles_from_snapshot restore_skill_entry \
    restore_managed_skills_from_snapshot restore_install_state \
    restore_project_management_surface restore_agents_infra_surface; do
    load_fn "$fn_name"
  done

  snapshot_text="$(load_block 'PM_STATE="$HOME/.config/task-board/install.json"')"
  ( set -x; eval "$snapshot_text" ) > "$OUT/snapshot.log" 2>&1
  record "snapshot block runs against the sandbox pair" "0" "$?"

  RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/current-pair"
  got="$(cat "$RECOVERY_ROOT/manifest/binaries.txt" 2>/dev/null | wc -l | tr -d ' ')"
  record "manifest records the derived executable set" "7" "$got"
  got="$(cat "$RECOVERY_ROOT/manifest/skills.txt" 2>/dev/null | wc -l | tr -d ' ')"
  record "manifest records the derived managed-skill set" "9" "$got"
  if [ -f "$RECOVERY_ROOT/ai/install.json" ]; then got=present; else got=absent; fi
  record "snapshot saves the agents-infra install state" "present" "$got"

  build_release_tree
  apply_release_effects
  restore_project_management_surface "$RELEASE" > "$OUT/restore.log" 2>&1
  record "rollback runs" "0" "$?"

  if [ -e "$HOME/.roles/release-only-role" ]; then got=present; else got=absent; fi
  record "a role the release added is removed by rollback" "absent" "$got"
  if [ -e "$HOME/.agents/skills/release-only-skill" ]; then got=present; else got=absent; fi
  record "a skill the release added is removed by rollback" "absent" "$got"
  if [ -e "$HOME/.claude/skills/release-only-skill" ] \
    || [ -L "$HOME/.claude/skills/release-only-skill" ]; then got=present; else got=absent; fi
  record "its ~/.claude link is removed by rollback" "absent" "$got"
  if [ -e "$HOME/.codex/skills/release-only-skill" ] \
    || [ -L "$HOME/.codex/skills/release-only-skill" ]; then got=present; else got=absent; fi
  record "its ~/.codex link is removed by rollback" "absent" "$got"
  if [ -e "$HOME/.local/bin/release-only-bin" ]; then got=present; else got=absent; fi
  record "an executable the release added is removed" "absent" "$got"
  got="$(cat "$HOME/.agents/skills/go-testing-tools/SKILL.md")"
  record "a rewritten dependency skill is restored" \
    "skill go-testing-tools v1" "$got"
  got="$(cat "$HOME/.roles/developer/ROLE.md")"
  record "an existing role is restored" "role developer v1" "$got"
  got="$("$HOME/.local/bin/task-board" 2>/dev/null | sed 's/ (commit.*//')"
  record "a replaced executable is restored" "task-board 0.0.0-test" "$got"
  got="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["repoPath"])' "$HOME/.config/task-board/install.json")"
  record "the task-board install state is restored" "$SB/repos/pm" "$got"
  # Surviving roles and skills the release did not add stay untouched.
  got="$(ls -1A "$HOME/.roles" | grep -v '^\.' | wc -l | tr -d ' ')"
  expected="$(ls -1A "$REAL_PM/.roles" | wc -l | tr -d ' ')"
  record "no unrelated role is removed" "$expected" "$got"

  # agents-infra half: the install state must come back too.
  ai_state="$HOME/Library/Application Support/agents-infra/install.json"
  python3 - "$ai_state" <<'PY'
import json, sys
p = sys.argv[1]
state = json.load(open(p))
state["repoPath"] = "/abandoned/release/candidate"
json.dump(state, open(p, "w"), indent=2)
PY
  restore_agents_infra_surface "$SB/repos/ai" > "$OUT/restore-ai.log" 2>&1
  record "agents-infra rollback runs" "0" "$?"
  got="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["repoPath"])' "$ai_state")"
  record "the agents-infra install state is restored" "$SB/repos/ai" "$got"
  if grep -q 'setup global' "$HOME/agents-infra-calls.log" 2>/dev/null; then
    got=called; else got=absent; fi
  record "the saved agents-infra ran setup global" "called" "$got"
else
  record "restore_project_management_surface + snapshot block present" \
    "present" "absent"
fi

# --------------------------------------------------------------------------
# P13 — materialize_recovery_source refuses a destination it did not create.
# --------------------------------------------------------------------------
if have_fn materialize_recovery_source; then
  load_fn extract_reported_commit
  load_fn materialize_recovery_source
  build_sandbox worktree
  version="$("$HOME/.local/bin/agents-infra" version)"
  ( cd "$OUT" && materialize_recovery_source "$SB/repos/ai" \
    "relative-destination" "$version" ) >/dev/null 2>&1
  record "worktree refuses a misplaced relative destination" "1" "$?"
  materialize_recovery_source "$SB/repos/ai" "$SB/absolute-dest" "$version" \
    >/dev/null 2>&1
  record "worktree accepts an absolute destination" "0" "$?"
  materialize_recovery_source "$SB/repos/ai" "$SB/never" "no commit here" \
    >/dev/null 2>&1
  record "worktree refuses an unparseable version" "1" "$?"
  if [ -e "$SB/never" ]; then got=present; else got=absent; fi
  record "no worktree is created on refusal" "absent" "$got"
else
  record "materialize_recovery_source present" "present" "absent"
fi

# --------------------------------------------------------------------------
# P12 — LLDB guard covers the backup path production writes.
# --------------------------------------------------------------------------
if have_fn snapshot_lldb_surface; then
  load_fn snapshot_lldb_surface
  FAKE="$OUT/fakebrew"
  mkdir -p "$FAKE/bin" "$FAKE/prefix/bin"
  cat > "$FAKE/bin/brew" <<STUB
#!/bin/sh
case "\$1" in
  --prefix)
    if [ -n "\$2" ]; then printf '%s\n' "$FAKE/llvm"; else printf '%s\n' "$FAKE/prefix"; fi ;;
  list)
    if [ "\$2" = --formula ]; then printf 'llvm\n'; else printf 'llvm 23.1.0\n'; fi ;;
  *) exit 1 ;;
esac
STUB
  chmod +x "$FAKE/bin/brew"
  mkdir -p "$FAKE/llvm/bin"
  printf 'wrapper v1\n' > "$FAKE/prefix/bin/lldb-mcp"
  printf 'backup v1\n' > "$FAKE/prefix/bin/lldb-mcp.agents-infra.bak"
  OLD_PATH="$PATH"; PATH="$FAKE/bin:$PATH"; export PATH
  snapshot_lldb_surface > "$OUT/lldb-before.txt" 2>&1
  record "LLDB snapshot reads a fake Homebrew surface" "0" "$?"
  printf 'backup v2-mutated\n' > "$FAKE/prefix/bin/lldb-mcp.agents-infra.bak"
  snapshot_lldb_surface > "$OUT/lldb-after.txt" 2>&1
  cmp -s "$OUT/lldb-before.txt" "$OUT/lldb-after.txt"
  if [ $? -eq 0 ]; then got=identical; else got=different; fi
  record "a mutated .agents-infra.bak is detected" "different" "$got"
  PATH="$OLD_PATH"; export PATH
else
  record "snapshot_lldb_surface present" "present" "absent"
fi

export HOME="$ORIGINAL_HOME"
say ""
say "probes=$PROBES failures=$FAILURES"
[ "$FAILURES" -eq 0 ]
