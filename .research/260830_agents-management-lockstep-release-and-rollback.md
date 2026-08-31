# Agents-management lockstep release and rollback plan

Task `TASK-260830-s5ro4e` · document revision 11 = `CR-TASK-260830-s5ro4e-10` ·
2026-08-31 · **operating document**

Revision history, review findings, the reasoning behind each decision and the
validation-evidence index are in the companion review record,
`260830_agents-management-lockstep-release-and-rollback_review-record.md`, and the four
scripts Part II and Part III name are in the companion runbook,
`260830_agents-management-lockstep-release-and-rollback_runbook.md`, delivered with this
plan. Nothing here depends on having read the review record; the runbook is the thing you
execute, and every refusal it performs is also named here, in *Stop conditions*.

Revision 11 is a size correction: revision 10 was correct and 3585 lines long, and a
rollback path nobody can read under pressure protects nothing.
Every safety property of revision 10 survives below as a precondition with one exact
command and one stated expected result, as a named prerequisite (PRE-1…6), or as a stated
limit. None was deleted to make the budget. **This plan must receive a new independent
acceptance before any breaking change, release, installation or tag in this sequence.**

**How to use it.** Read *Decision*, *Capabilities* and *The daemon* before the day. Run the
Part II preconditions for the step you are taking; any one not producing its stated result
stops the step. Run the step from Part IV. On failure run R1 or R2 from Part III, then the
canary. Read *Stop conditions* and *Limits* before improvising — several states here have
no procedure. Two rules hold everywhere: **a value written in this document is never a
comparison operand** (every check compares two live values, one from the saved installed
pair's own source and one from the release; recorded numbers are dated observations), and
**a failed read is never an absence** (a non-zero exit, unparseable output, or an empty
result where a set was expected is `unknown` and stops the step).

## Decision

Use public `skill-agents-management v0.5.0` as the compatibility bridge. Release task-board
first, then agents-infra, while both compile exact `v0.5.0`. Permit `v0.6.0` to remove a
compatibility adapter only after exact released-source and linked-binary inventories prove
neither released consumer needs it. Repin task-board first, then agents-infra, to the
accepted immutable `v0.6.0`.

The running programs do not dynamically load the Go module, but `v0.5.0` compatibility
calls reach the plugin graph and inference engine transitively (measured below). Safety
rests on immutable module versions and checksums in each released binary; unmodified
compatibility tests against the exact public module; a removal manifest proved against both
released consumers; out-of-place candidate canaries; and cutover isolation with the previous
pair kept as an absolute-path routing authority until the installed canary succeeds. There
is no planned *dynamic module-version* mismatch. There are four mixed-install windows and
two daemon windows, all in Part V; none is called safe or zero-duration.

| Reserved version | Role |
| --- | --- |
| `skill-agents-management v0.5.0` at `b74f758a90a422304d55460422831da80e3d6cc8`, `h1:yy/j8YgKrXLWcb6zzoAld79a7iamTvOI5eZd/9MaAIk=` | already released; the bridge |
| `skill-project-management 0.25.0` | first task-board release compiled with exact `v0.5.0` |
| `relux-agents-infra v1.7.0` | first agents-infra release compiled with exact `v0.5.0` |
| `skill-agents-management v0.6.0` | conditional breaking release |
| `skill-project-management 0.25.1` | task-board repin to exact `v0.6.0` |
| `relux-agents-infra v1.7.1` | agents-infra repin to exact `v0.6.0` |

If any reserved version is occupied before execution, stop and revise this reviewed plan.
Never move or replace a published tag. No release build may use an untagged module commit, a
mutable tag, `@latest`, an agents-management `replace`, or a Go workspace override.

## Measured consumption surface

Clean detached worktree at task-board `063197b10e02cbacabfba4c192d16fa310f70eb7`, only
`go.mod`/`go.sum` changed to pin public `v0.5.0`, `GOWORK=off`, `-mod=mod`, no replace, Go
1.25.5 darwin/arm64.

| Instrument | Result | Establishes |
| --- | ---: | --- |
| direct tracked-source import census | 25 production / 36 test files, 13 package families | task-board imports compatibility packages but directly imports neither `pkg/plugin` nor `pkg/inferenceengine` |
| `go list -deps ./...` | 20 agents-management packages | the composed graph includes `pkg/plugin`, `pkg/inferenceengine` and `.../engines/mlx` despite zero direct imports |
| `go tool nm` on the candidate binary | 555 module symbols | 161 `pkg/agentic`, 32 `pkg/plugin`, 153 `pkg/vendorplugin`, 7 `pkg/inferenceengine`, 190 `pkg/providerlimits`, including `plugin.(*Registry).RegisterAll`, `inferenceengine.init`, `vendorplugin.productionEngineFactSource.ReadEngineFacts` |
| production-path inspection + candidate preflight | exit 0 | the reach is operational, not test-only or dead source |

The production path is unguarded by build tags: `internal/spawn/launch_plan.go` calls
`agentic.NewRegistry`/`Register`/`BuildPlan`; `pkg/agentic/registry.go` maps that into
`plugin.Registry.Register`; four blank-imported vendor packages register through
`vendorplugin.Register` at `init`; `internal/spawn/models.go:307-332` reads
`vendorplugin.Default` during package init, before any CLI command runs; and
`pkg/vendorplugin/engine.go` calls the concrete inference-engine contract. Reproduced gate:
`GOWORK=off go test -mod=mod ./internal/spawn -count=1` → `ok … 10.341s`, exit 0. **Absence
of direct imports is not an adapter-removal gate**; that gate is Step 4's.

## Compatibility matrix

| task-board | Embedded | agents-infra | Status |
| --- | --- | --- | --- |
| `0.24.3-172-g063197b1` | `v0.2.0` | `v1.6.1-103-g4270549` | pair observed installed while planning; the recovery baseline is derived from the installed artifact at execution time, not from this row |
| scratch `063197b1` | `v0.5.0` | unchanged | composed tests/build/preflight green; not released |
| `0.25.0` | `v0.5.0` | old installed | required Step-2 transition; must pass the installed canary |
| `0.25.0` | `v0.5.0` | `v1.7.0`/`v0.5.0` | stable bridge pair |
| `0.25.0` | `v0.5.0` | `v1.7.0`, `v0.6.0` published | still supported: immutable embedded pins |
| `0.25.1` | `v0.6.0` | `v1.7.0`/`v0.5.0` | required Step-5 transition; the external compose seam must be canaried before cutover |
| `0.25.1` | `v0.6.0` | `v1.7.1`/`v0.6.0` | target pair |

## Capabilities that must remain working

At every step an authoritative board path must perform: spawn preflight and model/provider
selection; run-record creation, child launch, observe/status/directives and terminal
outcome; outcome resource add/update/read; producer handoff, reviewer spawn, reviewer
verdict, `accept_cr`, rework routing, and Story-workspace and Change-Request reads.

Goal-bound and writable-context spawn additionally needs a **usable session plane**:
`launchTrackedSpawnRun` (`cmd/codex_goal_spawn.go:50-56`) routes those runs through the
board's `tb-sessiond`, so a board whose daemon speaks a protocol the CLI does not accept has
lost `spawn` even though every file on disk is correct. P7 and the R1 canary cover that; a
`project_config` read does not. During any window the default PATH surface is not
authoritative — the saved pair is, and this prefix makes a saved `task-board` resolve the
saved `tb-sessiond` and saved `agents-infra` rather than a partially installed sibling:

```bash
PATH="$RECOVERY_ROOT/bin:$PATH" "$RECOVERY_ROOT/bin/task-board" q --format compact \
  'project_config(view=spawn-preflight, role=developer, agent=codex, task_class=research)'
```

# Part I — What the operator must know

Both installers are `#!/usr/bin/env zsh`, read at `skill-project-management` `f1319eff` and
`relux-agents-infra` `bb857fe5`. **Nothing here is a comparison operand**: P1 compares the
release against the saved source, and each step derives the release's own stage list into
`w2/w3/w5/w6-*.txt`. Three stages decide a window's risk:

| Stage | Where | Hazard | Removed by |
| --- | --- | --- | --- |
| `install_go` | both installers, **before every managed binary**; task-board `:131-146`, agents-infra `:75-91` | `brew install go` when `go` is not on `PATH`, `exit 1` otherwise. **No skip flag exists** | **P3** — its non-mutating first branch becomes the only reachable one |
| `install_lldb_mcp` | agents-infra `:93-171` | `brew install llvm`; rewrites `$(brew --prefix)/bin/lldb-mcp`; may write `…lldb-mcp.agents-infra.bak` | `AGENTS_INFRA_SKIP_LLDB_MCP=1`, which returns at `:94-97` before any read. Mandatory in Steps 3/6 |
| `install_skills` | task-board `:840-888`, **unconditional** | for each `SKILL_REPOS` entry not already a real directory: `git clone` from the network at an arbitrary remote HEAD, after `rm -f` of a stale symlink — so a failed clone **leaves the skill deleted** | **P4** — every managed skill is already a real directory before the window |

`install_binary` (`:195-210`) writes a new inode and `mv`s it into place, so no destination
is ever missing mid-install. The task-board installer also targets `~/.agents/skills`,
`~/.roles` and the machine-scoped install state, and maintains a `~/.claude/skills` and
`~/.codex/skills` entry per skill; two skills (`core-data`, `swiftui`) get a root `SKILL.md`
symlink via `SKILL_NESTED`.

## The daemon: `tb-sessiond`

`install_binary "tb-sessiond"` replaces a **file**. What that file becomes when it runs is a
long-lived per-board daemon whose lifetime is independent of the file's, which owns that
board's provider hosts and session records, and which speaks a **versioned control protocol**
to the CLI. It is the one surface here where a window is opened by a process, not a file.
Facts re-read from production text at `f1319eff` on 2026-08-31:

| Fact | Source |
| --- | --- |
| `controlProtocolVersion = 4`, a compile-time constant. `--version` does not print it; it is derivable only from the source tree a build came from | `internal/sessionmanager/types.go:11-13` |
| the daemon reports `protocol_version`, `daemon_version`, `pid`, `board_fingerprint`, `session_count`; per-session `attached_clients` | `types.go:114-128`, `:57` |
| the daemon binary is resolved **through `PATH`** by bare name — which is what makes the recovery-PATH prefix work | `cmd/session.go:26,480-484` (`exec.LookPath`) |
| `tb-sessiond` installs its own `SIGTERM`/`os.Interrupt` handler, and that context is what `daemon.Serve` selects on | `cmd/tb-sessiond/main.go:137` |
| SIGTERM takes the **same** `server.Shutdown` + deferred `Manager.Close` + lock/endpoint/token/socket cleanup arm as `Drain`. **No CLI subcommand exposes `Drain`**, so the signal is the only reachable door — and it is a supported one | `daemon.go:82-249`, `control.go:36-47` |
| `Manager.Close` releases manager-owned transports only; doc comment: *"Provider hosts intentionally survive daemon replacement and are reconciled from their durable records by the next instance."* Its body iterates `m.proxies` and nothing else | `manager.go:1068-1081` |
| `LoadAndReconcile` rebuilds sessions and hosts from durable records, first thing after the next instance takes the lock | `manager.go:193-249` |

**CLI newer than daemon:** `ConnectOrStart` takes over automatically — `Drain` first, then a
fenced SIGTERM/SIGKILL if the old daemon does not release its lock (`client.go:145-166`,
`stale_takeover.go:71-163`). The operator does not choose when.

**Daemon newer than CLI: unprotected, not refused.** The refusal text at `client.go:725-731`
is a plain `fmt.Errorf` while the older-daemon branch returns a typed error, and
`ConnectOrStart`'s single `errors.As` matches only the typed one — so the newer-daemon error
is discarded and control reaches `takeOverUnresponsiveManager` (`stale_takeover.go:168-266`),
which dials with the **unchecked** `dialManagerStatus`. The older CLI therefore either
receives a live client and issues v4 calls against a v5 daemon, or SIGTERM/SIGKILLs it —
decided by that daemon's `startup_healthy`, unknowable for a build that does not exist.
PRE-1 is the fix and it belongs to the owning repository.

**The one dangerous state, and the ordering that removes it.** The dangerous state is exactly
one combination: *a live daemon speaking a newer protocol than the `task-board` binary on
disk.* It is not a property of any command, so it is not addressed by telling an operator
which commands to avoid — two earlier revisions tried that and enumerated the command family
wrongly. Stop the daemon **before** the executables are replaced and let the next command
start it **after**, and the combination never exists. That ordering is **R1 step 0d**, and it
costs three things:

- **Attached clients are disconnected.** `Manager.Close` closes the per-session proxy listener
  (`manager.go:596,1035-1058`) — concretely a managed `codex`/`claude` wrapper mid-run. The
  provider process survives and is reconciled, so the *session* is not lost, but the client
  must re-attach. Step 0d therefore **refuses** a board reporting any `attached_clients` rather
  than disconnecting somebody else's running agent to make a rollback tidy. That field is read
  **strictly**: an absent key, a non-list, or a row count disagreeing with
  `session_count + quarantined_count` is `unknown` and stops R1 — never a zero that stops
  nothing and is then recorded as a fact nobody read (Limit 7).
- **Durable-record skew is untouched.** `validateSessionRecord` (`record_store.go:315-321`)
  fails closed on an unknown record contract, so a `-v3` record is quarantined by name
  (`manager.go:205-221`) — loud, per-session, not resumable by the restored daemon. If
  `0.25.0` instead keeps `-v2` and adds fields, the lenient `json.Unmarshal`
  (`record_store.go:89`) drops them on the next save. Which one it does is **unknown**
  (Limit 6).
- **It is UNTESTED.** Every fact above is derived and probed; deriving a mechanism is not
  rehearsing it. **P8** is how that label gets cleared.

Prevention still comes first: **P7c** derives the constant on both sides before the window
and stops the step if it changed; step 0d makes a bump recoverable instead of one-way, and is
not a licence to skip P7. **A bump is likely, not hypothetical** — `git log -L` on the
constant's own line gives `df4201dc` (2026-07-24) v1, `445b391e` (07-26) `2`, `f8ba3268`
(07-28) `3`, `d66bfea8` (07-28) `4`: four values in five days.

**Image skew exists even when the protocol does not change.** A daemon started before an
install keeps executing its own task-board image, with that image's embedded
agents-management, until it restarts. Measured read-only 2026-08-30 via
`task-board --board-dir <board> --json session status`:

| Board | PID | `daemon_version` | `protocol_version` | Sessions |
| --- | ---: | --- | ---: | ---: |
| `~/src/mac-infra/.task-board` | 42553 | `0.24.3-112-ged48781` | 4 | 31 |
| `~/src/casual-talks/.task-board` | 61270 | `0.24.3-124-g2a527a8` | 4 | 2 |
| this migration's board | — | not running (exit 0 — a legitimate absence) | — | 0 |

The installed file is `0.24.3-172-g063197b1`: 60 and 48 commits of drift, 8 and 5 days old.
That window (`Wimg`) is open **before this migration starts**, and this migration's own board
has no daemon — so an installed canary here observes nothing about the two boards that do.

# Part II — Preconditions

Each is one command with a stated expected result. `$SAVED_SOURCE` is the recovery worktree
the snapshot created at the commit the *installed* binary reports; `$RELEASE` is the release
tree. Both operands are therefore live. **Run every Part II command under `bash`**, never the
operator's interactive `zsh`: the two disagree on unmatched globs and word splitting, and a
precondition whose verdict depends on which shell typed it is not a precondition.

## P1 — installer text unchanged from the source this plan analysed

```bash
diff -ru "$SAVED_SOURCE/scripts" "$RELEASE/scripts"; echo "exit=$?"
```

**Expected: no output, `exit=0`.** If non-empty, answer in writing, attached to the step
evidence: does it change `setup_main` or the top-level call sequence (what the windows are
defined from)? `install_go`, `install_lldb_mcp` or `install_skills` (the three stages modelled
by behaviour)? `AGENTS_SKILLS_DIR`/`CLAUDE_SKILLS_DIR`/`CODEX_SKILLS_DIR` (`:816-818`),
`BIN_DIR`, or the destinations in `register_skill`/`install_roles`/`write_install_state` (what
the snapshot and rollback aim at)? Does it add a `source` line, pulling in a file P1 and P2
were not scoped over? **Any "yes" stops the step**: Part I is re-derived and this plan gets a
new review, because a release that changes the installer's boundary needs a plan that
describes the new boundary.

## P2 — no environment variable this plan has not decided

Match the `$` sigil, not an expansion operator, so `$X`, `${X}`, `${X:-}`, `${X-}`, `${X:=}`
are all seen. Each installer gets its **own named operand list**: task-board is `setup.sh` plus
the one file it sources (`scripts/lib/agents-infra-compose.zsh`, `:35`); agents-infra is
`setup.sh` alone and has no `scripts/lib/` at all, so one shared `scripts/lib/`\*`.zsh` glob
aborts the whole command under zsh — `set_of` never runs, `diff` compares two empty streams, the
probe `grep` never executes, and the failure prints byte-for-byte what a pass prints. `set_of`
therefore **refuses** an operand it cannot read, and P2 passes only on a `P2-SET-OK` sigil.

```bash
set_of() {   # an unread set is not an empty set
  for f in "$@"; do test -r "$f" || { echo "STOP P2 unreadable operand $f" >&2; return 2; }; done
  grep -ohE '\$\{?[A-Z][A-Z0-9_]*' "$@" | sed -E 's/^\$\{?//' | sort -u
}
o="$RECOVERY_ROOT/p2-old.txt"; n="$RECOVERY_ROOT/p2-new.txt"
L=scripts/lib/agents-infra-compose.zsh    # task-board only; agents-infra has no scripts/lib

# Steps 2 and 5, task-board -- setup.sh plus the one file it sources, both named:
set_of "$SAVED_SOURCE/scripts/setup.sh" "$SAVED_SOURCE/$L" > "$o" &&
set_of "$RELEASE/scripts/setup.sh"      "$RELEASE/$L"      > "$n" &&
diff "$o" "$n" && test -s "$n" && echo P2-SET-OK $(wc -l < "$n")
grep -nE 'printenv|(^|[^_[:alnum:]])eval([^_[:alnum:]]|$)|\$\{\(P\)|\$\{!|\[\[ -v ' \
  "$RELEASE/scripts/setup.sh" "$RELEASE/$L"

# Steps 3 and 6, agents-infra -- setup.sh and nothing else:
set_of "$SAVED_SOURCE/scripts/setup.sh" > "$o" && set_of "$RELEASE/scripts/setup.sh" > "$n" &&
diff "$o" "$n" && test -s "$n" && echo P2-SET-OK $(wc -l < "$n")
grep -nE 'printenv|(^|[^_[:alnum:]])eval([^_[:alnum:]]|$)|\$\{\(P\)|\$\{!|\[\[ -v ' \
  "$RELEASE/scripts/setup.sh"
```

**Expected: one `P2-SET-OK <n>` line and no output from the `grep`, which exits 1 finding
nothing** (planning-time `n`: 38 task-board, 19 agents-infra; both `grep`s observed empty).
**No sigil stops the step** — a differing set, an empty set, or an unreadable operand. A name
only in the release is an input nobody decided about; a `grep` match means the environment
surface is no longer readable by inspection. The set is deliberately unfiltered:
`BIN_DIR` is *both* self-assigned and an external override, so any "names the script assigns
itself" filter is wrong. Each external name's disposition is the install invocation itself
(Part IV); `HOME`, `PATH`, `PWD`, `TMPDIR`, `XDG_CONFIG_HOME` stay real, and `SKILL_DIR`/
`SOURCE_DIR` are computed from `$0`.

## P3 — `go` already resolves, so no installer can install it inside a window

```bash
command -v go && go version && brew list --formula | grep -qx go && echo PRECONDITION-P3-OK
```

**Expected: an absolute path, a version line, `PRECONDITION-P3-OK`.** `install_go` runs first
in both installers and has no skip flag, so only making its non-mutating branch reachable
prevents a Homebrew mutation inside the window. If `go` does not resolve, install it **outside
every window**, then re-run; the check must pass in the same environment the installer runs in,
and `env -u` of `TASK_BOARD_*` does not touch `PATH`, so it does. Under P3 an interruption
inside `install_go` changes nothing and the step is re-run from the top. Without P3, an
interrupted `brew install go` damages Homebrew's own state, which this plan cannot restore:
report `unknown/failed` and do not resume until the Homebrew owner repairs it.

## P4 — every managed skill is already a real directory

The list is read from the release's own map, so a ninth skill is covered without editing this
document. An empty result is a failed read, not an empty map.

```bash
sed -n '/^SKILL_REPOS=(/,/^)/p' "$PM_RELEASE/scripts/setup.sh" \
  | sed -nE 's/^[[:space:]]+([A-Za-z0-9._-]+)[[:space:]]+".*/\1/p' \
  | tee "$RECOVERY_ROOT/release-skill-repos.txt"        # 8 keys at planning time
echo project-management >> "$RECOVERY_ROOT/release-skill-repos.txt"
while IFS= read -r skill; do
  t="$HOME/.agents/skills/$skill"
  if   [ -L "$t" ]; then printf 'REFUSE %-24s symlink\n' "$skill"
  elif [ -d "$t" ]; then printf 'OK     %-24s directory\n' "$skill"
  elif [ -e "$t" ]; then printf 'REFUSE %-24s not a directory\n' "$skill"
  else                   printf 'REFUSE %-24s absent\n' "$skill"; fi
done < "$RECOVERY_ROOT/release-skill-repos.txt"
```

**Expected: nine `OK`, no `REFUSE`.** A `REFUSE` means the installer would clone from the
network inside the window, with skill deletion as its failure mode. Materialise that skill from
its own repository beforehand, under your own SSH agent. **Do not let the release installer do
it.**

## P5 — the release's artifact sets match the sets the snapshot holds

```bash
# Both halves always: the released one from its release tree, the other from the snapshot source.
PM_SRC="$RECOVERY_ROOT/sources/skill-project-management"   # Steps 2/5: PM_SRC="$PM_RELEASE"
AI_SRC="$RECOVERY_ROOT/sources/relux-agents-infra"         # Steps 3/6: AI_SRC="$AI_RELEASE"
{ sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' "$PM_SRC/scripts/setup.sh"
  sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' "$AI_SRC/scripts/setup.sh"
} | sort -u > "$RECOVERY_ROOT/p5-binaries.txt"; test -s "$RECOVERY_ROOT/p5-binaries.txt"
ls -1A "$PM_SRC/.roles" | sort -u > "$RECOVERY_ROOT/p5-roles.txt"
diff <(sort -u "$RECOVERY_ROOT/manifest/binaries.txt") "$RECOVERY_ROOT/p5-binaries.txt"
diff <(sort -u "$RECOVERY_ROOT/manifest/roles.txt")    "$RECOVERY_ROOT/p5-roles.txt"
```

Observed at planning time: 5 task-board executables, 2 agents-infra executables, 9 roles, 9
skills. **Expected: no output from either `diff`.** A name only in the release is an artifact
R1/R2 must quarantine; a name only in the snapshot is one the release stopped installing. Both
are permitted and handled in Part III — discovering the difference afterwards is not.

## P6 — Homebrew and the LLDB surface unchanged across an agents-infra window

Save **RB-1** from the runbook as `"$RECOVERY_ROOT/snapshot-brew.sh"`. It is a **top-level
script, not a function**: under `set -euo pipefail` at script scope a failed read aborts it and
publishes no file, so a failed read can never become a measured absence. It refuses an existing
destination file,
writes through a `.partial` and `mv`s only on success, and captures `brew --prefix`, the
**whole** formula list, `brew list --versions llvm`, `brew --prefix llvm`, and for each of
the four LLDB targets either `stat -f '%N|%HT|%z|%m|%Sp'` plus `shasum -a 256`, or an
explicit `ABSENT|` line.

Run it immediately before and after each agents-infra install, into distinct filenames, then
`diff` them. **Expected: no output, `exit=0`, both files present and non-empty.** A missing file
is `unknown/failed`, not "identical". Taking the **whole formula list** is what makes a
`brew install go` by any stage visible. `brew list --versions llvm` exits 1 when llvm is
genuinely absent, which aborts the block — correct for this host, which has llvm; a host without
it needs a reviewed variant, **not** an appended `|| true`. A difference means an installer stage
mutated Homebrew inside the window: report `unknown/failed`, withhold every new default-PATH
composition, and do not claim `W3`/`W6` rollback success. **This plan does not restore Homebrew
and does not claim to.** Pre-existing anomaly on this host: `$llvm_prefix/bin/lldb-mcp` and
`$llvm_prefix/bin/lldb` are `ABSENT` — a baseline, not a migration result, and not repaired here.

## P7 — the live daemons, and the protocol both sides speak

Applies to **Steps 2 and 5**, the two that replace `tb-sessiond`.

```bash
# P7a - census. A pgrep exit other than 0 or 1 is a failed census: unknown, never "no daemons".
rc=0; pgrep -fl tb-sessiond > "$RECOVERY_ROOT/daemons-before.txt" || rc=$?
case "$rc" in
  0) : ;;
  1) printf 'NO-LIVE-DAEMON\n' > "$RECOVERY_ROOT/daemons-before.txt" ;;
  *) printf 'DAEMON-CENSUS-FAILED rc=%s\n' "$rc" >&2; exit 1 ;;
esac
sed -nE 's/.*--board-dir[ =]([^ ]+).*/\1/p' "$RECOVERY_ROOT/daemons-before.txt" \
  | sort -u > "$RECOVERY_ROOT/daemon-boards.txt"
# P7b - read each daemon without starting or restarting one: both subcommands use Dial
# (client.go:70), which launches nothing.
while IFS= read -r board; do
  task-board --board-dir "$board" --json session status
  task-board --board-dir "$board" --json session list
done < "$RECOVERY_ROOT/daemon-boards.txt"
# P7c - derive controlProtocolVersion on both sides.
grep -n 'controlProtocolVersion *=' "$SAVED_SOURCE/tools/board-cli/internal/sessionmanager/types.go"
grep -n 'controlProtocolVersion *=' "$PM_RELEASE/tools/board-cli/internal/sessionmanager/types.go"
```

Record per board: `protocol_version`, `daemon_version`, `pid`, `board_fingerprint`,
`session_count`, summed `attached_clients`. **P7c expects exactly one line from each side and
the two values equal**; an empty result is a failed read — the constant may have moved — and is
`unknown`, not "no protocol".

| P7c result | Required disposition, recorded per board before the window |
| --- | --- |
| **Equal** — protocol-transparent but **not** image-transparent (`Wimg`) | each board with a live daemon is either **quiesced** (`session_count` 0 and the daemon stopped by its owner, outside every window) **or explicitly excluded** from goal-bound and writable-context spawn until its daemon next exits. "We will see" is not a disposition; a live daemon with no recorded disposition stops the step |
| **Different** | **stop the step for a decision, before the window.** The two safe forms: **(a)** a release that does not change the constant, or **(b)** every live daemon quiesced and stopped by its owner first. (b) is a cross-board operation this plan has no authority to perform — obtain it in writing from the owners of every board holding a daemon and record it with the step evidence. If a board is stranded anyway, R1 step 0d makes the rollback possible rather than one-way; a fallback, not a licence to skip (a) or (b) |

The set of commands reaching the daemon-starting path is **derived, never restated here**,
because a future release can extend it without touching this document. Diagnostic only — no
procedure in this plan depends on it. At `f1319eff` it yields 9 call sites plus the definition
at `cmd/session.go:360`:

```bash
grep -rn --include='*.go' 'connectOrStartSessionManager\|ConnectOrStartAndAttach' \
  "$PM_RELEASE/tools/board-cli/cmd" | grep -v '_test.go'
```

## P8 — the rollback has been rehearsed on a disposable copy

**Every rollback here is UNTESTED until this precondition is satisfied, and it is currently NOT
SATISFIED.** Required before Step 2, on a disposable `HOME` and a disposable board, with real
built artifacts and not stand-ins: take the Part III snapshot end to end (including
`git worktree add`, the `cmp` loop and the same-device assertions); install a real
`0.25.0`-class build; start a `tb-sessiond` and open at least one session; run **R1 including
step 0d** and the R1 canary; run **R2** including `setup global`/`verify global`/`doctor global`;
run both installers under exactly the Part IV `env -u` invocations; record every real exit code.

This is also the only way to settle Limits 1, 2, 3, 4 and 6, which is why it is one item and not
five. **If the rehearsal is not authorised, the release proceeds with P7 prevention as the first
line and every rollback labelled UNTESTED — a decision for the release owner to take explicitly,
not a default.** This planning task is forbidden to perform it.

## Named prerequisites and NOT IMPLEMENTED

| # | Requirement | Owner | Status |
| --- | --- | --- | --- |
| PRE-1 | **Fail closed when an older CLI meets a newer daemon on a `ConnectOrStart` path.** `managerProtocolCompatibility` returns a *typed* error on the `actual > controlProtocolVersion` branch as it already does on `<`; `ConnectOrStart` matches that type and returns it instead of discarding it; `takeOverUnresponsiveManager` runs the compatibility check before returning a client or signalling a PID. Ships with negative tests that fail when a newer daemon is admitted — today `grep -rl 'is newer than this wrapper supports' *_test.go` finds nothing, while the older direction has two named tests | `skill-project-management` | **NOT IMPLEMENTED.** R1 step 0d removes the state this guard would protect *for this plan's own rollback only*; it does nothing about a skew somebody else creates |
| PRE-2 | Host-wide interlock preventing any other agent, wrapper or human from invoking the installed `task-board` by plain name during `W2`/`W5` | `skill-project-management` | **NOT IMPLEMENTED.** Withholding the default PATH protects only work this operator starts. Mitigation: P7's per-board disposition |
| PRE-3 | Mechanically classify a P1 diff as "touches a modelled boundary" or not | this plan | **NOT IMPLEMENTED.** Mitigation: P1's four questions answered in writing and attached, so the judgement is reviewable after the fact |
| PRE-4 | Detect a concurrent `brew` invocation racing the P6 before-snapshot | this plan | **NOT IMPLEMENTED.** Mitigation: run releases when no other Homebrew work is in flight; a P6 difference is authoritative |
| PRE-5 | Verify the Steps 2/5 `env -u` list still covers every `TASK_BOARD_*` name P2 reports | this plan | **NOT IMPLEMENTED.** P2's diff catches an added name, not a forgotten `-u`. Compare by eye and record it |
| PRE-6 | Rehearse the rollbacks against real artifacts | release owner | **P8** |

# Part III — Snapshot and rollback

Take the snapshot before Step 2 into `current-pair`, and again after both `0.25.0` and `v1.7.0`
pass their canaries into a fresh `bridge-pair`; the script is identical, only `RECOVERY_ROOT`
changes. **Never overwrite a prior snapshot.** These commands copy no credential, cookie, token,
keychain data or environment value. Nothing here names a binary, role, skill, repository path or
install directory this plan decided — every set is read at execution time from the artifact that
owns it.

## Snapshot

**RB-2** in the companion runbook. In order, refusing at each step rather than continuing:

1. read `binDir` and `repoPath` from both machine-scoped install states; require the two
   `binDir` values equal and both repositories present;
2. create `$RECOVERY_ROOT` only if it does not already exist, and require every destination
   (`$BIN_DIR`, `~/.agents/skills`, `~/.roles`, `~/.claude/skills`, `~/.codex/skills` and
   both state directories) to be on the **same device**, so every restore is an atomic
   rename rather than a copy that can fail halfway;
3. derive the executable set from the *installed pair's own installers*, copy each and `cmp`
   it, and record `ls -1A "$BIN_DIR"` plus the sha256 set;
4. read `--version` from each saved executable, extract its commit, require task-board and
   `tb-sessiond` to report the **same** commit — a divergence is an incoherent install —
   and `git worktree add --detach` each source at that commit into `$RECOVERY_ROOT/sources`.
   This is what makes the recovery baseline and the recovery binary the same pair by
   construction, with no revision written into this document;
5. re-derive the executable set from those exact worktrees and `diff` it against step 3: a
   repository HEAD that moved and changed the set means the copy is not the installed
   release's artifact set;
6. save every `~/.roles` entry, and every skill (the registered one plus every `SKILL_REPOS`
   dependency) across all three trees, recording `symlink|directory|absent` per tree and
   staging a tree for every `directory`;
7. assert every recorded kind is restorable — a symlink has a target, a directory has a
   staged tree — and refuse the snapshot at the door if not;
8. copy both machine-scoped install states last.

Observed counts at planning time: `binaries.txt` 7, `roles.txt` 9, `skills.txt` 9,
`skill-state.txt` 9 — observations; step 5's `diff`, not a count, is the check. A parse failure,
an unresolvable commit, a dirty source worktree, a divergent task-board/tb-sessiond commit, a
failed `cmp`, a changed derived set or any missing executable **blocks Step 2**, and is never
treated as an absence or a usable baseline.

## R1 — task-board surface (rolls back Steps 2 and 5)

`RECOVERY_ROOT` is `current-pair` for Step 2, `bridge-pair` for Step 5; `PM_RELEASE` is the
release tree that was installed.

**RB-3** in the companion runbook, in this order:

- **0 — Admissibility, before anything is mutated.** Every recorded skill kind must be
  restorable, and every saved executable, role and install state present. A snapshot R1
  cannot fully restore produces no partial run at all.
- **0d — Stop every daemon this rollback would otherwise strand — before any executable is
  replaced**, while the installed CLI still matches the daemons the release started. Census
  with `pgrep -fl tb-sessiond` (an exit other than 0 or 1 is a failed census, `unknown`);
  **stop** if a remote-board daemon is present. Per board: read `session status` and
  `session list` (both `Dial`-based, so they start and restart nothing); skip a board
  reporting `running:false`; **refuse** a board with any `attached_clients`, indexed strictly and
  cross-checked against `session_count + quarantined_count` (Limit 7); re-identify the
  PID against the live status and `ps` before signalling; `kill -TERM`; wait up to 30 s —
  **never SIGKILL**, a daemon that does not exit is a stop and an escalation to its owner;
  assert the board then reports `running:false`; record the board **only after** the stop is
  confirmed.
- **1 — Executables.** Restore what the snapshot holds, quarantine what the release added as
  `.<name>.superseded.$$`, and *report* anything else that appeared in `$BIN_DIR` rather than
  removing it: this plan owns the release's artifacts, not the operator's PATH.
- **2 — Skills.** Restore each to its exact recorded kind in all three trees, then remove any
  skill this release's `SKILL_REPOS` added.
- **3 — Roles.** Restore every snapshot role, quarantine every role the release added.
- **4 — Install state last.**
- **5 — Assert the state 0d removed is still gone.** An assumption that a state was removed is
  worth nothing next to a read that it is gone; a board that has a daemon again acquired it
  *during* R1, and that is a stop.

**R1 canary.** `project_config` alone is not enough — it need never touch a daemon, so it can go
green over exactly the failure this has to detect.

```bash
PATH="$RECOVERY_ROOT/bin:$PATH" "$RECOVERY_ROOT/bin/task-board"  --version
PATH="$RECOVERY_ROOT/bin:$PATH" "$RECOVERY_ROOT/bin/tb-sessiond" --version
PATH="$RECOVERY_ROOT/bin:$PATH" "$RECOVERY_ROOT/bin/task-board" q --format compact \
  'project_config(view=spawn-preflight, role=developer, agent=codex, task_class=research)'
while IFS= read -r board; do test -n "$board" || continue
  PATH="$RECOVERY_ROOT/bin:$PATH" \
    "$RECOVERY_ROOT/bin/task-board" --board-dir "$board" --json session status
done < "$RECOVERY_ROOT/daemon-boards-after-r1.txt"
# then a live spawn -> outcome -> handoff -> reviewer route canary, recovery-prefixed
```

`"running":false` with exit 0 is a **legitimate absence**: the next command starts a daemon from
the restored image via `PATH`. A non-zero exit or unparseable output is `unknown` and R1 is
**not** green. A `protocol_version` **equal** to the restored build's `controlProtocolVersion`
(P7c, read from `$RECOVERY_ROOT/sources/skill-project-management`) is the rolled-back state. A
**greater** one should be unreachable after step 0d, and step 5 fails closed if a daemon came
back; if it is observed anyway, R1 is **failed, not partially green** — do not run the live canary
against that board and do not signal it, because the version-matched CLI step 0d needed is no
longer installed and the safe stop is gone. Escalate to that board's owner.

**R1's scope.** It restores files — executables, skills, roles, install state — and for provider
children and already-exec'd CLI processes that is correct and intended. For `tb-sessiond` it also
*removes* a process, through the daemon's own shutdown path, before the files move. Two limits it
does not repair: a board whose owner will not detach a live client cannot be rolled back on this
pass, and durable session records rewritten under a new record contract are quarantined by name
rather than resumed. A partial run of R1 is a rollback failure, never a successful half-restore.
Status: **UNTESTED** (P8).

## R2 — agents-infra managed runtime (rolls back Steps 3 and 6)

**RB-4** in the companion runbook: restore the two agents-infra executables the release's
own `BINARY_NAME`/`MODEL_HARNESS_BINARY_NAME` name (quarantining any the release added),
re-mint the runtime with `agents-infra setup global --source-dir` pointed at the saved
source worktree — with `AGENTS_INFRA_SOURCE_DIR` **unset**, since it would override
`--source-dir`'s resolution order — then `verify global` and `doctor global`, then the
machine-scoped install state last, because the installed binary resolves its own source tree
through it.

R2 copies no `~/.agents` wholesale and prints no config contents. **R2 is more unknown than R1** —
R1's logic has been exercised against a disposable `HOME` with stand-in artifacts, R2 has **no
execution evidence of any kind**, because `setup global` mutates an installed runtime. Rehearse R2
first. Afterwards the restored install state names the original repository path while the receipt
`setup global` wrote names the recovery source worktree; that divergence is deliberate and must be
recorded with the rollback evidence, because **deleting `$RECOVERY_ROOT` after a rollback leaves a
runtime whose receipt names a path that no longer exists.** Retain it until a clean reinstall
re-mints both.

| Procedure | Logic exercised? | Run against a real pair? | Status |
| --- | --- | --- | --- |
| R1 including step 0 admissibility | yes — disposable `HOME`, stand-in artifacts | no | **UNTESTED operationally** |
| R1 step 0d daemon stop | derived from production text and probed; never signalled a live daemon | no | **UNTESTED** |
| R2 | **no** | no | **UNTESTED** |
| Snapshot block | derivations and version parsing only | no | **UNTESTED** |

# Part IV — Release order

| Step | Repository / version | Must remain working | Window | Rollback | Status |
| --- | --- | --- | --- | --- | --- |
| **0** freeze | whatever pair is installed; the operator reads no version from this document (planning-time observation: `0.24.3-172-g063197b1`/`v0.2.0` and `v1.6.1-103-g4270549`) | everything: read, preflight, one real spawn, outcome handoff, reviewer route on the installed pair | none | nothing installed changes; on snapshot failure stop and discard only the new snapshot after inspecting it | **UNTESTED** |
| **1** hold `v0.5.0` | `skill-agents-management v0.5.0` at the reserved commit/checksum; publishes nothing | installed board keeps spawning and routing with embedded `v0.2.0` | none at runtime; the scratch repin is source-only | `go mod edit -require=…@v0.2.0`, `GOFLAGS=-mod=mod GOWORK=off go mod tidy`, re-run the candidate gate | **PARTIALLY TESTED** (source build/test only) |
| **2** task-board `0.25.0` | `skill-project-management 0.25.0`, exact public `v0.5.0`; agents-infra unchanged | saved `current-pair` performs every spawn/route/resource/CR operation for the whole window | **`W2`** | **R1** with `RECOVERY_ROOT=…/current-pair`, `PM_RELEASE=…/skill-project-management-0.25.0`, then the R1 canary | **UNTESTED** |
| **3** agents-infra `v1.7.0` | `relux-agents-infra v1.7.0`, exact public `v0.5.0`; task-board stays `0.25.0` | `0.25.0` preflights, spawns, composes through agents-infra, attaches an outcome and routes to a reviewer, before and after; agents-infra full tests/build/verify plus the real target/compose/Pi refusal gates on the exact release head | **`W3`** | **R2** with `…/current-pair`, `AI_RELEASE=…/relux-agents-infra-v1.7.0`, then the recovery-prefixed canary; if task-board routing is red, also R1 | **UNTESTED** |
| **4** allow `v0.6.0` | `skill-agents-management v0.6.0` candidate | installed `0.25.0`/`v1.7.0` continues all spawn and route with embedded `v0.5.0` | none at runtime; publishing repins nothing | before publication: repin the consumer candidate to `v0.5.0`, `go mod tidy`, `go test ./... -count=1`. After a bad tag is published **never delete or move it** — keep consumers on `v0.5.0` and publish a reviewed forward `v0.6.1` | **UNTESTED**; `v0.6.0` does not exist |
| **5** task-board `0.25.1` | `skill-project-management 0.25.1`, exact released `v0.6.0`; agents-infra stays `v1.7.0`/`v0.5.0` | full candidate and installed spawn/route canaries **including the external agents-infra compose seam** | **`W5`** | **R1** with `…/bridge-pair`, then the R1 canary plus `PATH="$RECOVERY_ROOT/bin:$PATH" env -u AGENTS_INFRA_SOURCE_DIR "$RECOVERY_ROOT/bin/agents-infra" verify global` | **UNTESTED** |
| **6** agents-infra `v1.7.1` | `relux-agents-infra v1.7.1`, exact released `v0.6.0`; task-board stays `0.25.1` | agents-infra full gates and the installed spawn/compose/outcome/handoff/reviewer canary, reviewer route included | **`W6`** | **R2** against `…/bridge-pair`, then the canary; if the Step-5 seam canary fails, R1 against `bridge-pair` too | **UNTESTED** |

Forward fixes always use new immutable patch releases (`0.25.2`, `v1.7.2`); never rewrite
`0.25.1`, `v1.7.1` or `v0.6.0`. **A half-restored pair is a recovery state, never a release
success**: if one half restores and the other does not, keep `PATH="$RECOVERY_ROOT/bin:$PATH"` and
admit no new default-PATH work until both are green.

**Step 0 specifics.** Run **P7a and P7b here** and record the census with the snapshot, so Step 2's
disposition has a baseline to be a disposition *about*. The snapshot is admissible only when each
saved executable's reported commit resolves to the HEAD of its clean detached worktree and, in a
disposable `HOME`, the saved `agents-infra` binary runs `setup global --source-dir` against its own
derived worktree then `verify global`, both exit 0. An installed executable whose commit does not
resolve **blocks Step 0**: repair or reinstall a resolvable pair, never substitute a nearby
revision. Do not begin Step 2 while a run is at a handoff/acceptance mutation boundary.

**Steps 2 and 5.** Both run P1–P5, P7 and P8 outside the window, against `current-pair`/`0.25.0`
and `bridge-pair`/`0.25.1` respectively. **P7 is re-run at Step 5, never inherited:** Step 5
replaces `tb-sessiond` again, and a protocol unchanged at Step 2 says nothing about Step 5. Because
Step 5's `$SAVED_SOURCE` is the `bridge-pair` source, its P1 compares `0.25.1`'s installer against
`0.25.0`'s.

```bash
RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/current-pair"      # bridge-pair at Step 5
PM_RELEASE="$HOME/.local/state/TASK-260830-s5ro4e/release-candidates/skill-project-management-0.25.0"
SAVED_SOURCE="$RECOVERY_ROOT/sources/skill-project-management"
test "$(git -C "$PM_RELEASE" describe --tags --exact-match)" = "0.25.0"
sed -n '/^setup_main() {/,/^}/p' "$PM_RELEASE/scripts/setup.sh" \
  > "$RECOVERY_ROOT/w2-setup-main.txt"    # what W2 is defined from: the release's own text
env -u TASK_BOARD_PROJECT_DIR -u TASK_BOARD_CONFIGURE_PROJECT \
    -u TASK_BOARD_BOOTSTRAP_LOCAL_AGENTS -u TASK_BOARD_MODE -u TASK_BOARD_DIR \
    -u TASK_BOARD_REMOTE -u TASK_BOARD_USER -u TASK_BOARD_ID \
    -u TASK_BOARD_TLS_NO_VERIFY -u TASK_BOARD_SETUP_SOURCE_ONLY \
  "$PM_RELEASE/scripts/setup.sh"
```

A **bare** machine-scoped install is mandatory and no flag is permitted: `--configure-project`
additionally runs `configure_project`, `check_agents_infra_compose` and `bootstrap_local_agents`
inside the same window and writes a project config this plan does not model. `W2`/`W5` open at the
**first `install_binary` call** — not at the start of the script, since `install_go` precedes it and
mutates nothing under P3 — and close at installed-canary or R1-canary success; the stages inside are
the sequence in `w2/w5-setup-main.txt`, and the duration is **measured**, not assumed sub-second.

In-flight runs stay on their existing process images: a run reaching a new CLI mutation during the
window may see a partial default install, so retry that mutation through the recovery-prefixed old
pair — do not reparent, rewrite or kill it. An interruption inside `install_go` changes nothing under
P3; one inside `install_skills` leaves a skill tree or a link partial, and recovery authority for
that surface is R1, so the run resumes only after the R1 canary is green. **`tb-sessiond` does not
follow that rule:** replacing the file leaves every running daemon alive on its old image, which is
`Wimg` when P7c found the protocol unchanged, and is a step P7 already stopped when it found it
changed. Note the asymmetry — withholding the default PATH protects work *this* operator starts, not
another agent or human invoking the installed `task-board` by plain name (PRE-2).

**Steps 3 and 6.** Both run P1, P2, P3, P5 and the **P6 before-snapshot** outside the window; P7 does
not apply, since `tb-sessiond` is not in this release's artifact set.

```bash
AI_RELEASE="$HOME/.local/state/TASK-260830-s5ro4e/release-candidates/relux-agents-infra-v1.7.0"
SAVED_SOURCE="$RECOVERY_ROOT/sources/relux-agents-infra"
test "$(git -C "$AI_RELEASE" describe --tags --exact-match)" = "v1.7.0"
sed -n '/^install_go$/,$p' "$AI_RELEASE/scripts/setup.sh" > "$RECOVERY_ROOT/w3-call-sequence.txt"
bash "$RECOVERY_ROOT/snapshot-brew.sh" "$RECOVERY_ROOT/brew-before-step3.txt"
AGENTS_INFRA_SKIP_LLDB_MCP=1 env -u BIN_DIR -u AGENTS_INFRA_CONFIG_DIR \
  "$AI_RELEASE/scripts/setup.sh"
bash "$RECOVERY_ROOT/snapshot-brew.sh" "$RECOVERY_ROOT/brew-after-step3.txt"
diff "$RECOVERY_ROOT/brew-before-step3.txt" "$RECOVERY_ROOT/brew-after-step3.txt"; echo "exit=$?"
```

That is the whole invocation, no arguments. `--with-pdf-tools` is forbidden — it runs a separate
toolchain installer this plan does not model. `BIN_DIR` must be unset so the install directory is
the one both install states record, and `AGENTS_INFRA_CONFIG_DIR` must be unset so
`resolve_config_dir` picks the path the snapshot saved. `AGENTS_INFRA_SKIP_LLDB_MCP=1` is mandatory,
not an operator option: LLDB bootstrap is unrelated to this migration and is deferred to a separate
reviewed maintenance operation. Retain both brew files, the installer exit code and the measured
duration; a difference, a missing file or an empty file withholds every new default-PATH composition
**even when the installer exited 0**.

`W3`/`W6` start when `agents-infra` is replaced (`install_go` and the skipped `install_lldb_mcp`
precede it and mutate nothing) and cover `model-harness`, install state, receipt invalidation,
`.agents` source sync, Claude/Codex links, helper/target launchers, runtime verification and receipt
write; they end at installed-canary or R2-canary success, with the duration recorded. Provider
children, primary sessions, Pi runtimes and existing LLDB MCP processes keep their images and
leases — do not restart them, and withhold new composition until the P6 comparison, installed
verification and canary all pass. A live daemon resolves the programs it launches at exec time
through the `PATH` it inherited, so a child launched by an old daemon after this step can pick up the
**new** `agents-infra`; nothing in this step proves that combination, which is why the withholding
runs to the canary and not merely to the installer's exit. An interruption inside `install_binary`
leaves one executable replaced and the other not; the new-inode `mv` means no destination is ever
missing, and R2 restores both. After Step 3 is green, capture the fresh `bridge-pair` snapshot before
any Step-5 mutation.

**Step 4 — the removal gate.** `v0.5.0` labels these bridges load-bearing and the composed
task-board reaches all of them: `agentic.Registry`/`NewRegistry`/`Register`/
`RegisterWithDependencies`/`Lookup`/`Graph`/`BuildPlan` plus the `System`, `LaunchRequest`,
`LaunchMode`, `Plan` shapes; `vendorplugin.Registry`/`Default`/`Register`/`DeclareRuntime`/
`RuntimeDeclarationOf`/`Lookup`/`Graph` plus `Vendor`, `Model`, `RuntimeDeclaration`,
`BrokerProvenance` and the pricing/lifecycle/effort shapes; and the adapter calls those trigger —
`plugin.Registry.Register`, graph topological sync, inference-engine init and
`productionEngineFactSource.ReadEngineFacts`. `0.25.0` and `v1.7.0` do not exist yet, so their final
inventories cannot be guessed here. Before a `v0.6.0` candidate may be approved the release owner
creates a versioned removal manifest listing every method, exported type/field, init registration,
linked symbol family and production call path proposed for removal, then for **both** exact released
consumer tags: (1) create clean detached worktrees of `0.25.0` and `v1.7.0`; (2) prove their
`go.mod`/`go.sum` use public exact `v0.5.0`, no replace, no workspace override; (3) record direct
imports, transitive packages and `go tool nm` symbols; (4) map every manifest item to a consumer call
site or prove it unused, via type-check/build failures and production-entry tests; (5) repin scratch
consumers to the exact `v0.6.0` candidate and run full tests, builds and real launch/compose
canaries; (6) **negatively narrow** each removed adapter in a private candidate — both released
consumers must remain green because they no longer require it, and a delete-only result is
insufficient; (7) require `required_by_0.25.0=false` **and** `required_by_v1.7.0=false` for every
item. `unknown`, a failed read, a partial inventory, a surviving symbol, an init-only reach or a
production-path reach **blocks `v0.6.0`**. A zero direct-import count cannot satisfy this gate.

# Part V — Named windows

| Window | Duration | Impact | Avoidance / survival |
| --- | --- | --- | --- |
| `v0.5.0` published, installed board embeds `v0.2.0`; and `v0.6.0` published while both consumers embed `v0.5.0` | `0s` dynamic mismatch | none from publication alone | exact immutable embedded pins; no `@latest` |
| **`W2`** task-board `0.25.0` install | first `install_binary` call → installed or R1 canary; **measured, unbounded in advance** | the default surface may mix every executable, the registered skill, every `SKILL_REPOS` dependency with both link-tree entries, roles and install state; new default spawn/route may fail | withhold the default surface; P4 removes the clone branch, P3 the `install_go` brew branch; saved `current-pair` with recovery PATH is the authority |
| Step-3 and Step-6 Homebrew/LLDB bypass guard | P6 before-snapshot → successful after-comparison; measured | no accepted mutation; a failed read or difference makes Homebrew/LLDB state **unknown** | mandatory `AGENTS_INFRA_SKIP_LLDB_MCP=1` plus P3; the whole formula list is snapshotted; stop on any difference |
| **`W3`** agents-infra `v1.7.0` install | first managed-binary replacement → installed or R2 canary; measured | both agents-infra executables, the machine-scoped install state, `.agents`, links/helpers and the receipt may mix; new composition may fail | saved `current-pair`; existing children keep their images; retry later mutations through the saved pair |
| `0.25.0` installed, `v1.7.0` not yet | until the Step-3 canary | supported **only** if the Step-2 live external-seam canary passed | an explicitly tested pair, not inferred independence |
| **`W2d`** daemon protocol skew, **only if `0.25.0` changes `controlProtocolVersion`** | opens when the new pair is on disk; closes **per board** when that board's daemon is next reached by a daemon-starting command — **no upper bound**, since an untouched board keeps its old daemon indefinitely | **until then nothing visibly fails; the impact is silence.** The CLI is the newer side, so the takeover succeeds and is not reported. **After** it, that board carries a daemon newer than the snapshot's CLI, and the restored CLI does not refuse it — it drives it unchecked or fences it, decided by `startup_healthy` | **avoided first:** P7c stops the step if the constant changed; if it must change, every live daemon is quiesced and stopped by its owner beforehand. **Then survivable:** R1 step 0d stops a stranded daemon through its own SIGTERM shutdown *before* any executable moves, so the state is never created. Step 0d refuses a board with attached clients and is **UNTESTED** |
| **`Wimg`** daemon image skew, **whether or not the protocol changes** | opens when `tb-sessiond` is replaced; closes per board when that daemon next exits. **Unbounded — already open on this host**, daemons 60 and 48 commits behind, 8 and 5 days old | each live daemon keeps executing the old task-board image and its old embedded agents-management while the CLI on disk executes the new one. Protocol equality proves the transport is compatible and nothing about how two embedded contract versions compose across it. 31 and 2 sessions held at planning time | P7's recorded per-board disposition: quiesced and stopped by its owner outside the window, **or** explicitly excluded from goal-bound and writable-context spawn until its daemon exits. A live daemon with no recorded disposition stops the step |
| **`W5`** task-board `0.25.1` install | as `W2`, measured | as `W2` | `bridge-pair` authority, same P3/P4, full R1 |
| `0.25.1`/`v0.6.0` with agents-infra `v1.7.0`/`v0.5.0` | until the Step-6 canary | cross-version embedded libraries behind one external seam | allowed only after the Step-5 real spawn/compose/route canary |
| **`W6`** agents-infra `v1.7.1` install | as `W3`, measured | as `W3` | `bridge-pair`; restore one half, canary, then both if the intermediate pair is red |

`W2d` and `Wimg` repeat identically for Step 5, against `bridge-pair` and the `0.25.0`→`0.25.1`
comparison. Two windows only *fail* to exist because of a precondition, listed so nobody removes the
precondition: **`install_go`** in both installers (without P3, an unbounded Homebrew mutation inside
the release window, with no snapshot and no restore) and **the `install_skills` clone branch**
(without P4, a network fetch of an unrelated skill at an arbitrary remote HEAD, whose failure mode
deletes the skill). The forbidden mode is running an installer from a breaking consumer candidate
before its exact tests, removal manifest, out-of-place canary and recovery snapshot are green; its
impact would be loss of new spawn/route authority for an unbounded time, and this plan never accepts
it.

# Stop conditions

Stop and restore the last accepted pair if:

- any precondition P1–P8 does not produce its stated expected result;
- any manifest item proposed for `v0.6.0` is still required or `unknown` for either exact released
  consumer;
- any read or inventory is failed, partial or malformed;
- a candidate requires `replace`, a workspace override, a mutable lookup or a shadow graph;
- a real spawn succeeds but resource/handoff/reviewer routing fails;
- the saved absolute pair cannot route during a cutover failure;
- an in-flight run would need to be killed, reparented or rewritten;
- one restored half is green but the pair canary is red;
- a P1 diff touches a modelled boundary — the plan is re-derived and re-reviewed, never patched at
  the console;
- `controlProtocolVersion` differs between the saved installed source and the release, at Step 2 or
  Step 5. **P7 cannot clear this stop by being re-run:** it needs either a release that does not
  change the constant, or a written per-board decision from the owners of every board holding a live
  daemon, obtained before the window;
- a live daemon exists on any board with no recorded P7 disposition;
- the daemon census fails to read — a `pgrep` exit other than 0 or 1, or a `session status` that
  exits non-zero or returns unparseable output. A failed census is `unknown`, never "no daemons";
- **R1 step 0d cannot stop a daemon it must stop:** a board reports `attached_clients` greater than
  zero; the daemon does not exit within 30 s of SIGTERM — **do not escalate to SIGKILL, escalate to
  its owner**; the PID moves or does not resolve to a `tb-sessiond`; a remote-board daemon is
  present; or the board still reports `running` after the stop;
- after a rollback, any board reports a `protocol_version` greater than the restored build's, or a
  board step 0d stopped has a daemon again. Either means a daemon started **during** the restore,
  from an image this rollback did not choose. R1 is failed, not partially green, and the
  version-matched CLI step 0d needed is gone — escalate to that board's owner, do not retry.

# Limits — what more writing cannot fix

Each is a state that **cannot be made safe by writing; it must be exercised on a disposable copy
first.** These are results, not gaps.

1. **R1 and R2 have never been run against a real installed pair.** R1's logic was driven hard, but
   every artifact it moved was a text file standing in for a binary, a skill tree or a role — no
   multi-megabyte `rsync`, no real permission edge. R2 has no execution evidence at all.
2. **The protocol takeover has never been observed.** What fencing a v4 daemon holding 31 real
   sessions does to those sessions is **unknown**.
3. **The installers have never been run under exactly the Part IV `env -u` invocations.** They are
   quoted, derived and reviewed; not executed.
4. **Whether a protocol-5 `tb-sessiond` populates `startup_healthy`** — the single field deciding
   whether a restored older CLI receives an unchecked live client or SIGKILLs a daemon holding live
   sessions. It is a property of an unreleased build. Step 0d makes it moot *for this plan's own
   rollback*; it still decides a skew somebody else creates.
5. **What a *fenced* daemon's provider children do.** For the **graceful** stop the answer is derived
   from production text and is no longer unknown (`Manager.Close` preserves provider hosts,
   `LoadAndReconcile` reconciles them) and step 0d takes that path deliberately. The fenced path stays
   **UNKNOWN** and is now an escalation, not a step in any procedure.
6. **Whether `0.25.0` changes the durable session-record contract.** A `-v3` record is quarantined by
   name — loud, per-session, unrecoverable by the restored daemon. A kept `-v2` with added fields is
   dropped silently on the next save. The two shapes are opposite and neither can be chosen from
   `f1319eff`.

7. **A cross-protocol `attached_clients` read cannot be trusted at all.** Step 0d's strict read
   closes only the shell half. Against a **protocol-mismatched** daemon — the one situation 0d
   exists for — the Go client's lenient `json.Unmarshal` can zero a renamed or dropped field
   before the shell sees it, so a well-formed `attached_clients: 0` may be a decode artifact
   rather than a measurement, and no shell-level check distinguishes them. PRE-1 is the
   structural fix; until it ships, a zero from a daemon whose `protocol_version` differs from
   the CLI's is **unread**, and that board is dispositioned by its owner, not stopped by R1.

**Limits 1, 2, 3, 4 and 6 are one rehearsal, and it is P8:** build `0.25.0`, install it into a
disposable `HOME`, start a daemon on a disposable board, downgrade with step 0d, and watch. If that
rehearsal is not authorised, P7's prevention is the first line and step 0d is an **UNTESTED**
fallback — better than a prohibition an operator has to remember, and not the same as a tested
rollback. This plan's acceptance criterion, reviewed and accepted before the first breaking change,
is met by review; its **operational** readiness is not, and no revision of this document can supply
it.

# References

Read at `skill-project-management` `f1319eff` and `relux-agents-infra` `bb857fe5`; the daemon facts
re-verified at `f1319eff` on 2026-08-31.

- task-board `internal/spawn/launch_plan.go:157-164,332-342`, `models.go:307-332`
- agents-management `pkg/agentic/registry.go:120-128,170-183`,
  `pkg/vendorplugin/registry.go:252-337,363-388`, `pkg/vendorplugin/engine.go:9-11,31-54`
- project-management installer `scripts/setup.sh:16,35,131-146,195-210,816-818,822-831,834-838,840-888,891-930`
- agents-infra installer `scripts/setup.sh:5-18,64-73,75-91,93-171,255-271`;
  `tools/agents-infra/internal/infra/infra.go:134-215`; `.../source_dir.go:12-19`
- session manager `types.go:11-13,57,114-128`;
  `client.go:70-96,112-126,128-165,228-253,340-346,677,714-735`;
  `stale_takeover.go:25-41,43-59,71-79,86-163,168-266`; `daemon.go:82-249`; `control.go:36-47`;
  `manager.go:193-249,596,1035-1058,1068-1081`; `record_store.go:19-20,67-115,315-321`;
  `builder_gateway_manager.go:319-352`
- task-board `cmd/tb-sessiond/main.go:137,139`;
  `cmd/session.go:26,34-110,126-141,151,199,245,360-372,480-506`;
  `cmd/session_context_economics.go:58`; `cmd/session_context_retention.go:50`;
  `cmd/spawn_context_lifecycle.go:292`; `cmd/codex_goal_spawn.go:35-44,50-56,74-79`;
  `cmd/codex_manager.go:92`; `cmd/claude_manager.go:115`; `cmd/context_security.go:75`
- companion: `260830_agents-management-lockstep-release-and-rollback_review-record.md`
