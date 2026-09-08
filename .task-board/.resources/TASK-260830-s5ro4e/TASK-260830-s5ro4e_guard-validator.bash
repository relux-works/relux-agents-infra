#!/bin/bash
# Same contracts as validate-revision5.zsh, driven under bash (the shell the
# plan's code fences declare). Proves the guards do not depend on shell options.
set -uo pipefail
plan="$1"; root="$2"
rm -rf "$root"; mkdir -p "$root"

extract_fn() {
  awk -v fn="$1" '
    $0 ~ "^" fn "\\(\\) [({]$" { emit = 1 }
    emit { print }
    emit && ($0 == "}" || $0 == ")") { exit }
  ' "$plan"
}
for fn in snapshot_lldb_surface capture_lldb_surface guarded_agents_infra_install \
          extract_reported_commit require_saved_binary_source_identity; do
  b="$(extract_fn "$fn")"; [ -n "$b" ] || { echo "MISSING $fn"; exit 64; }
  eval "$b"
done
mrs="$(extract_fn materialize_recovery_source)"
have_mrs=0; [ -n "$mrs" ] && { eval "$mrs"; have_mrs=1; }

mkbrew() {
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
for m in listfail notinstalled versionsfail prefixfail ok; do mkbrew "$m"; done

fixture="$root/fixture"; mkdir -p "$fixture/scripts"
printf '#!/bin/sh\necho ran >> "$GUARD_FIXTURE_LOG"\nexit 0\n' > "$fixture/scripts/setup.sh"
chmod 0755 "$fixture/scripts/setup.sh"

fail=0
chk() { # name expected_nonzero actual
  echo "$1=$3"
  if [ "$2" = nonzero ] && [ "$3" -eq 0 ]; then echo "FAIL $1 (expected nonzero)"; fail=1; fi
  if [ "$2" = zero ] && [ "$3" -ne 0 ]; then echo "FAIL $1 (expected zero)"; fail=1; fi
}

r=0; ( export PATH="$root/brew-listfail:$PATH"; snapshot_lldb_surface >"$root/a" ) 2>/dev/null || r=$?
chk bash_reader_read_failure nonzero "$r"
r=0; ( export PATH="$root/brew-listfail:$PATH"; set -euo pipefail
       if snapshot_lldb_surface >"$root/b"; then exit 0; else exit $?; fi ) 2>/dev/null || r=$?
chk bash_reader_if_context nonzero "$r"
r=0; ( PATH="$root/empty:/usr/bin:/bin"; snapshot_lldb_surface >"$root/c" ) 2>/dev/null || r=$?
chk bash_reader_brew_absent nonzero "$r"
r=0; ( export PATH="$root/brew-prefixfail:$PATH"; snapshot_lldb_surface >"$root/d" ) 2>/dev/null || r=$?
chk bash_reader_prefix_failure nonzero "$r"
r=0; ( export PATH="$root/brew-versionsfail:$PATH"; snapshot_lldb_surface >"$root/e" ) 2>/dev/null || r=$?
chk bash_reader_versions_failure nonzero "$r"
r=0; ( export PATH="$root/brew-notinstalled:$PATH"; snapshot_lldb_surface >"$root/f" ) 2>/dev/null || r=$?
chk bash_reader_measured_absence zero "$r"
grep -q '^LLVM_NOT_INSTALLED|' "$root/f" || { echo "FAIL bash measured-absence state"; fail=1; }
grep -q '^ABSENT|/bin/' "$root/f" && { echo "FAIL bash collapsed /bin probe"; fail=1; }

export GUARD_FIXTURE_LOG="$root/log"; : > "$GUARD_FIXTURE_LOG"
r=0; ( export PATH="$root/brew-listfail:$PATH"
       guarded_agents_infra_install "$fixture" "$root/g" ) 2>/dev/null || r=$?
chk bash_e2e_guard_failed_read nonzero "$r"
[ ! -e "$root/g/passed" ] || { echo "FAIL bash passed created"; fail=1; }
[ "$(wc -l < "$GUARD_FIXTURE_LOG" | tr -d ' ')" = 0 ] || { echo "FAIL bash installer ran"; fail=1; }

export GUARD_FIXTURE_LOG="$root/log2"; : > "$GUARD_FIXTURE_LOG"
r=0; guarded_agents_infra_install "$fixture" "$root/h" 2>/dev/null || r=$?
chk bash_e2e_guard_real_host zero "$r"

if [ "$have_mrs" -eq 1 ]; then
  ai=/Users/alexis/src/relux-works/relux-agents-infra
  v="$(/Users/alexis/.local/bin/agents-infra version)"
  r=0; got="$(materialize_recovery_source "$ai" "$root/src" "$v" 2>/dev/null)" || r=$?
  chk bash_derive_from_installed zero "$r"
  if [ "$r" -eq 0 ]; then
    exp="$(git -C "$ai" rev-parse "$(extract_reported_commit "$v")^{commit}")"
    [ "$got" = "$exp" ] || { echo "FAIL bash derived OID"; fail=1; }
    git -C "$ai" worktree remove --force "$root/src" >/dev/null 2>&1
  fi
  r=0; materialize_recovery_source "$ai" "$root/s2" 'agents-infra version unknown' >/dev/null 2>&1 || r=$?
  chk bash_derive_unparseable nonzero "$r"
  r=0; materialize_recovery_source "$ai" "$root/s3" 'agents-infra v9.9.9-1-gdeadbee commit=deadbee' >/dev/null 2>&1 || r=$?
  chk bash_derive_unresolvable nonzero "$r"
else
  echo "FAIL materialize_recovery_source absent"; fail=1
fi
echo "overall_fail=$fail"; exit "$fail"
