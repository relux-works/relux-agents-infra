#!/bin/zsh
# Revision-5 validator.
# Usage: validate-revision5.zsh <plan.md> <scratch-root> <agents-infra-source-worktree>
# Exits 0 only if every contract holds. Designed so that the negative probes are
# RED against the revision-4 plan and GREEN against revision 5.
set -euo pipefail

plan="$1"
root="$2"
source_wt="$3"
rm -rf "$root"; mkdir -p "$root"

# Extract a function definition verbatim from the plan by name. Handles both
# braced `name() {` and subshell `name() (` bodies.
extract_fn() {
  awk -v fn="$1" '
    $0 ~ "^" fn "\\(\\) [({]$" { emit = 1 }
    emit { print }
    emit && ($0 == "}" || $0 == ")") { exit }
  ' "$plan"
}

for fn in snapshot_lldb_surface capture_lldb_surface guarded_agents_infra_install \
          extract_reported_commit require_saved_binary_source_identity; do
  body="$(extract_fn "$fn")"
  if [[ -z "$body" ]]; then print -r -- "MISSING FUNCTION: $fn"; exit 64; fi
  eval "$body"
done
# materialize_recovery_source exists only in revision 5.
mrs_body="$(extract_fn materialize_recovery_source || true)"
have_mrs=0
if [[ -n "$mrs_body" ]]; then eval "$mrs_body"; have_mrs=1; fi

fixture="$root/fixture"; mkdir -p "$fixture/scripts"
cat > "$fixture/scripts/setup.sh" <<'INNER'
#!/bin/sh
echo ran >> "$GUARD_FIXTURE_LOG"
if [ -n "${GUARD_FIXTURE_FORCE_FAILURE:-}" ]; then exit 9; fi
exit 0
INNER
chmod 0755 "$fixture/scripts/setup.sh"

mkbrew() { # $1=mode -> creates $root/brew-$1/brew
  d="$root/brew-$1"; mkdir -p "$d"
  cat > "$d/brew" <<INNER
#!/bin/sh
mode="$1"
case "\$*" in
  "--prefix") echo /opt/homebrew; exit 0 ;;
  "list --formula")
    case "\$mode" in
      listfail) echo "Error: unreadable" >&2; exit 1 ;;
      notinstalled) printf 'git\\njq\\n'; exit 0 ;;
      *) printf 'git\\nllvm\\njq\\n'; exit 0 ;;
    esac ;;
  "list --versions llvm")
    case "\$mode" in
      versionsfail) echo "Error: unreadable" >&2; exit 1 ;;
      versionsempty) exit 0 ;;
      *) echo "llvm 23.1.0"; exit 0 ;;
    esac ;;
  "--prefix llvm")
    case "\$mode" in
      prefixfail) echo "Error: unreadable" >&2; exit 1 ;;
      *) echo /opt/homebrew/opt/llvm; exit 0 ;;
    esac ;;
esac
exit 1
INNER
  chmod 0755 "$d/brew"
}
for m in listfail notinstalled versionsfail versionsempty prefixfail ok; do mkbrew "$m"; done

fail=0
report() { print -r -- "$1=$2"; }

# ---- R1: read failure must refuse (RED on revision 4) ----
r1=0
( export PATH="$root/brew-listfail:$PATH"; snapshot_lldb_surface >"$root/r1.txt" ) || r1=$?
report reader_llvm_read_failure_exit "$r1"
test "$r1" -ne 0 || { print -r -- "FAIL R1: failed read returned zero"; fail=1 }
! grep -q '^ABSENT|/bin/' "$root/r1.txt" 2>/dev/null || {
  print -r -- "FAIL R1: failed read rendered as an absence under an empty prefix"; fail=1 }

# ---- R2: same failure, invoked as an `if` condition (the errexit-suppression shape) ----
r2=0
( export PATH="$root/brew-listfail:$PATH"
  set -euo pipefail
  if snapshot_lldb_surface >"$root/r2.txt"; then exit 0; else exit $?; fi ) || r2=$?
report reader_if_context_exit "$r2"
test "$r2" -ne 0 || { print -r -- "FAIL R2: caller if-context suppressed the read failure"; fail=1 }

# ---- R3: brew absent entirely must refuse ----
mkdir -p "$root/empty-bin"
r3=0
( PATH="$root/empty-bin:/usr/bin:/bin"; snapshot_lldb_surface >"$root/r3.txt" ) || r3=$?
report reader_brew_absent_exit "$r3"
test "$r3" -ne 0 || { print -r -- "FAIL R3: missing brew accepted"; fail=1 }

# ---- R4: llvm genuinely not installed is a MEASURED absence, not a refusal ----
r4=0
( export PATH="$root/brew-notinstalled:$PATH"; snapshot_lldb_surface >"$root/r4.txt" ) || r4=$?
report reader_llvm_not_installed_exit "$r4"
test "$r4" -eq 0 || { print -r -- "FAIL R4: legitimate absence refused"; fail=1 }
grep -q '^LLVM_NOT_INSTALLED|' "$root/r4.txt" || { print -r -- "FAIL R4: state not recorded"; fail=1 }
! grep -q '^ABSENT|/bin/' "$root/r4.txt" || { print -r -- "FAIL R4: probed collapsed /bin path"; fail=1 }

# ---- R5: llvm installed but a detail read is empty/failing refuses ----
r5=0
( export PATH="$root/brew-versionsempty:$PATH"; snapshot_lldb_surface >"$root/r5.txt" ) || r5=$?
report reader_ambiguous_exit "$r5"
test "$r5" -ne 0 || { print -r -- "FAIL R5: ambiguous llvm state accepted"; fail=1 }

r5b=0
( export PATH="$root/brew-prefixfail:$PATH"; snapshot_lldb_surface >"$root/r5b.txt" ) || r5b=$?
report reader_llvm_prefix_failure_exit "$r5b"
test "$r5b" -ne 0 || { print -r -- "FAIL R5b: failed llvm prefix read accepted"; fail=1 }
! grep -q '^ABSENT|/bin/' "$root/r5b.txt" 2>/dev/null || {
  print -r -- "FAIL R5b: failed prefix read rendered as an absence"; fail=1 }

r5c=0
( export PATH="$root/brew-versionsfail:$PATH"; snapshot_lldb_surface >"$root/r5c.txt" ) || r5c=$?
report reader_llvm_versions_failure_exit "$r5c"
test "$r5c" -ne 0 || { print -r -- "FAIL R5c: failed llvm versions read accepted"; fail=1 }

# ---- R6: end-to-end production guard with an unreadable llvm surface ----
export GUARD_FIXTURE_LOG="$root/e2e-installer.log"; : > "$GUARD_FIXTURE_LOG"
r6=0
( export PATH="$root/brew-listfail:$PATH"
  guarded_agents_infra_install "$fixture" "$root/e2e-guard" ) || r6=$?
report e2e_guard_failed_read_exit "$r6"
test "$r6" -ne 0 || { print -r -- "FAIL R6: guard admitted a failed read"; fail=1 }
test ! -e "$root/e2e-guard/passed" || { print -r -- "FAIL R6: passed created on failed read"; fail=1 }
test "$(wc -l < "$GUARD_FIXTURE_LOG" | tr -d ' ')" = 0 || {
  print -r -- "FAIL R6: installer ran despite an unusable before snapshot"; fail=1 }

# ---- R7: positive path, real reader on the real host ----
export GUARD_FIXTURE_LOG="$root/pos-installer.log"; : > "$GUARD_FIXTURE_LOG"
r7=0
guarded_agents_infra_install "$fixture" "$root/pos-guard" || r7=$?
report e2e_guard_real_host_exit "$r7"
test "$r7" -eq 0 || { print -r -- "FAIL R7: real-host positive path refused"; fail=1 }
test -f "$root/pos-guard/passed" || { print -r -- "FAIL R7: no passed marker"; fail=1 }

# ---- R8: stub-driven orchestration statuses (delta / installer failure) ----
print -r -- 0 > "$root/snapcount"
snapshot_lldb_surface() {
  n=$(cat "$root/snapcount"); n=$((n + 1)); print -r -- "$n" > "$root/snapcount"
  print -r -- "FULL|state-$n"
}
export GUARD_FIXTURE_LOG="$root/delta-installer.log"; : > "$GUARD_FIXTURE_LOG"
r8=0
guarded_agents_infra_install "$fixture" "$root/delta-guard" || r8=$?
report guard_delta_exit "$r8"
test "$r8" -ne 0 || { print -r -- "FAIL R8: surface delta accepted"; fail=1 }
grep -q '^cmp_exit=1$' "$root/delta-guard/status.txt" || { print -r -- "FAIL R8: cmp status"; fail=1 }

snapshot_lldb_surface() { print -r -- 'FULL|stable' }
export GUARD_FIXTURE_LOG="$root/instfail-installer.log"; : > "$GUARD_FIXTURE_LOG"
export GUARD_FIXTURE_FORCE_FAILURE=1
r9=0
guarded_agents_infra_install "$fixture" "$root/instfail-guard" || r9=$?
unset GUARD_FIXTURE_FORCE_FAILURE
report guard_installer_failure_exit "$r9"
test "$r9" -ne 0 || { print -r -- "FAIL R9: installer failure accepted"; fail=1 }
grep -q '^install_exit=9$' "$root/instfail-guard/status.txt" || { print -r -- "FAIL R9: status"; fail=1 }

# ---- R10..R13: F2 recovery-baseline derivation ----
if test "$have_mrs" -eq 0; then
  print -r -- "FAIL F2: materialize_recovery_source absent (baseline still hard-coded)"
  fail=1
  report baseline_derived 0
else
  report baseline_derived 1
  ai_repo="/Users/alexis/src/relux-works/relux-agents-infra"
  real_version="$(/Users/alexis/.local/bin/agents-infra version)"

  r10=0
  derived="$(materialize_recovery_source "$ai_repo" "$root/derived-src" "$real_version" 2>/dev/null)" || r10=$?
  report derive_from_installed_exit "$r10"
  test "$r10" -eq 0 || { print -r -- "FAIL R10: could not derive from installed artifact"; fail=1 }
  if test "$r10" -eq 0; then
    reported="$(extract_reported_commit "$real_version")"
    expected="$(git -C "$ai_repo" rev-parse "${reported}^{commit}")"
    test "$derived" = "$expected" || { print -r -- "FAIL R10: derived OID mismatch"; fail=1 }
    test "$(git -C "$root/derived-src" rev-parse HEAD)" = "$expected" || {
      print -r -- "FAIL R10: worktree HEAD is not the reported commit"; fail=1 }
    require_saved_binary_source_identity agents-infra "$root/derived-src" "$real_version" >/dev/null || {
      print -r -- "FAIL R10: derived pair fails its own identity gate"; fail=1 }
    git -C "$ai_repo" worktree remove --force "$root/derived-src" >/dev/null 2>&1 || true
  fi

  r11=0
  materialize_recovery_source "$ai_repo" "$root/unparsed-src" 'agents-infra version unknown' \
    >/dev/null 2>&1 || r11=$?
  report derive_unparseable_exit "$r11"
  test "$r11" -ne 0 || { print -r -- "FAIL R11: unparseable version produced a baseline"; fail=1 }
  test ! -e "$root/unparsed-src" || { print -r -- "FAIL R11: worktree created anyway"; fail=1 }

  r12=0
  materialize_recovery_source "$ai_repo" "$root/unknown-src" \
    'agents-infra v9.9.9-1-gdeadbee commit=deadbee' >/dev/null 2>&1 || r12=$?
  report derive_unresolvable_exit "$r12"
  test "$r12" -ne 0 || { print -r -- "FAIL R12: unresolvable commit produced a baseline"; fail=1 }
  test ! -e "$root/unknown-src" || { print -r -- "FAIL R12: worktree created anyway"; fail=1 }
fi

# ---- R14: no maintained baseline literal survives in the executable blocks ----
lit=$(awk '/^```bash$/{inb=1;next} /^```$/{inb=0;next} inb' "$plan" \
      | grep -cE '(063197b10e02cbacabfba4c192d16fa310f70eb7|4270549dd17c010599e2083bf3ec7672af60ea29)' || true)
report hardcoded_baseline_literals_in_code "$lit"
test "$lit" -eq 0 || { print -r -- "FAIL R14: executable blocks still pin a literal baseline"; fail=1 }

# ---- R15: step contracts intact ----
test "$(grep -c '^### Step [0-6] —' "$plan")" -eq 7
for step_number in {0..6}; do
  section="$(awk -v step="$step_number" '
    $0 ~ "^### Step " step " —" { emit = 1 }
    emit && $0 ~ "^### Step " && $0 !~ "^### Step " step " —" { exit }
    emit { print }' "$plan")"
  for pat in '^Repository/version:' '^Required board capability:' 'spawn' 'route' \
             '^In-flight runs:' '^Rollback command' '^Rollback status:'; do
    print -r -- "$section" | grep -q "$pat" || {
      print -r -- "FAIL R15: step $step_number missing $pat"; fail=1 }
  done
  print -r -- "$section" | grep -Eq '^(Mismatch window|Mixed-install window)' || {
    print -r -- "FAIL R15: step $step_number missing window"; fail=1 }
done
report release_step_contracts 7

print -r -- "overall_fail=$fail"
exit "$fail"
