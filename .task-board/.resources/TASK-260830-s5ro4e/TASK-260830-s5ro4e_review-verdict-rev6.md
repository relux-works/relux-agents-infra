# TASK-260830-s5ro4e review verdict — revision 6 (document revision 7)

Verdict: **changes requested**. Route to `analysis` for a corrected plan and a
new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-6` revision 6, base
`0425fb7bfb90fabe6552ea1a5b7a78c05adecd6c`, candidate tree
`48b1def75ea400d6a3c03782df0999dd54cec81f`, repository delta **empty**.
Reviewer run `RUN-260830-c2a992`. Production sources read at
`skill-project-management` `f1319eff`, `relux-agents-infra` `bb857fe5`.

## On the empty repository delta

The delta is empty because the base OID *is* the producer's own commit. Commit
`0425fb7` ("plan(lockstep): revision 7") adds
`.research/260830_agents-management-lockstep-release-and-rollback.md` (2001
lines) and 69 lines to `LOGBOOK.md`, and the CR snapshot was taken with that
commit already at the branch tip. So this is not a producer that changed
nothing: it is a snapshot artifact. I reviewed `5c9b4e4..0425fb7`, which is the
only commit between the revision-5 base and this one, and confirmed it touches
exactly those two files — no Go, shell, module or configuration file, no
release, tag, install or migration.

The plan file in the commit is byte-identical to the board resource
`TASK-260830-s5ro4e_lockstep-release-and-rollback-plan.md`.

## Revision 7 is a real improvement, and the three revision-6 findings are closed

I reproduced this rather than accepting it. Fences re-extracted from the
committed document with the producer's `extract-fences.py`: 39 blocks, all parse
(`bash -n` + `zsh -n` on 36 `bash` fences, `zsh -n` on 3 `zsh` fences). The
producer's validator run against that fresh extraction:
**74 pass, 0 fail, exit 0 under `bash`; 74 pass, 0 fail, exit 0 under `zsh`.**

Every A-row derivation reproduces against the real production installers: 5
task-board executables, 2 agents-infra executables, 8 `SKILL_REPOS` keys, 9
release roles equal to installed `~/.roles`, 38-name task-board environment set,
19-name agents-infra set, indirect-expansion probe empty on both (exit 1),
exactly one `source` line, `258:install_go 259:install_lldb_mcp`.

I also checked the dispositions P2 leans on, which the document asserts rather
than derives, and they hold on production text: task-board `BIN_DIR="$HOME/.local/bin"`
at `:16` is unconditional (so genuinely not an external input),
`AGENTS_SKILLS_DIR`/`CLAUDE_SKILLS_DIR`/`CODEX_SKILLS_DIR` at `:816-818` are
unconditional top-level assignments, agents-infra `BIN_DIR="${BIN_DIR:-…}"` and
`CONFIG_DIR="${AGENTS_INFRA_CONFIG_DIR:-}"` are external and both are `env -u`'d
by the Step-3/6 invocation, and the ten `TASK_BOARD_*` names in P2's set are
exactly the ten `-u` names in the Step-2/5 invocation. No lowercase-named
environment read exists in either installer today.

F1 (`install_go`), F2 (truncating mirror) and F3 (one override shape) are closed,
and B/C are honest attacks: they run the revision-6 guard verbatim against the
same mutant on real production text and show it admitting what P1/P2 refuse,
with narrowing controls. The `LOGBOOK.md` correction has the right shape —
entry `1920` is added, `1815` and `1529` carry explicit `SUPERSEDED by …` lines
with reasons, and no false claim stands as a current entry.

The rejection is not a regression. It is that the window enumeration is still
incomplete, one level deeper than last time, and that the surface it misses is
live on the target host right now.

## Findings

### F1 — High: `tb-sessiond` is a long-lived daemon with its own CLI↔daemon protocol version. The plan models neither the daemon nor the skew it creates, and R1 cannot recover the downgrade direction.

The plan mentions `tb-sessiond` eleven times. Every one is either the name in a
derived executable list, `--version` in the snapshot, or the single sentence at
`:1390`: "Provider children and already-exec'd CLI/sessiond processes continue."
There is no model of the daemon's lifetime, no window bounding it, and no
precondition on it. That is the gap.

**It is a persistent daemon, and two instances are live on this host now**, both
launched from the install directory the migration replaces, both bound to boards
this migration does not touch:

```text
42553 /Users/alexis/.local/bin/tb-sessiond --board-dir /Users/alexis/src/mac-infra/.task-board
61270 /Users/alexis/.local/bin/tb-sessiond --board-dir /Users/alexis/src/casual-talks/.task-board
```

**The CLI and the daemon negotiate a versioned control protocol.**
`tools/board-cli/internal/sessionmanager/types.go:13` defines
`controlProtocolVersion = 4`, and `client.go:714-735` compares it against the
running daemon's:

- `actual < controlProtocolVersion` → `ManagerProtocolUpgradeError`, and
  `ConnectOrStart` (`client.go:128,157`) responds by calling
  `takeOverOutdatedManager`, which **fences and restarts the running daemon**.
  This is the production connect path, not a helper: `ConnectOrStart` is the
  caller, and the takeover is reached from it directly.
- `actual > controlProtocolVersion` → a hard refusal with no recovery path:
  `"Session Manager protocol v%d is newer than this wrapper supports (v%d);
  update task-board before retrying"`. `takeOverOutdatedManager` refuses
  explicitly in that direction — `"refusing protocol takeover without an older
  daemon version"`.

Four consequences, each load-bearing:

1. **The plan's instruction is contradicted by the product.** Steps 0, 3 and 6
   say "Do not restart or kill them" about existing process images. For
   `tb-sessiond` the operator does not decide that: if `0.25.0` bumps
   `controlProtocolVersion`, the first new-CLI command against any board with a
   live pre-bump daemon restarts that daemon automatically, draining its
   provider hosts. The plan asserts a property of the host that production
   overrides.
2. **The skew is a window, and it is not named.** `W2` closes "when installed
   verification plus the live canary succeeds". The binary on disk is new at
   that point; the daemons for `mac-infra` and `casual-talks` are still the old
   image, and stay so until somebody next runs a command against those boards —
   an interval with no bound, no snapshot, and no entry in Part V. The AC
   requires every step at which the board runs against a contract version it
   does not support to be named with duration and impact. A daemon holding an
   older embedded `agents-management` behind a live control socket is exactly
   that, and it is the only such surface the plan does not name.
3. **The installed canary cannot see it.** The installer's `verify` stage reads
   `"$installed_sessiond" --version` — the file, not any running process
   (`scripts/setup.sh:495-501`). A canary on the migration's own board will
   either meet no daemon or trigger the upgrade takeover for that board only.
   Neither outcome says anything about the other two live daemons.
4. **R1 does not restore a working board in the downgrade direction, and its
   canary does not test it.** After R1 puts `0.24.3` back, a daemon that started
   from `0.25.0` during the window keeps running the newer protocol.
   `managerProtocolCompatibility` refuses it outright and `takeOverOutdatedManager`
   refuses to replace it, so the restored board cannot use that board until the
   newer daemon is stopped — which the plan forbids. The R1 recovery canary is
   `task-board q project_config(view=spawn-preflight, …)`, a read that need not
   reach the session manager at all, so it can go green over exactly this
   failure. That is a rollback whose stated success criterion does not exercise
   the state the rollback leaves behind.

Required rework:

1. Model `tb-sessiond` in Part I as what it is: a per-board daemon whose lifetime
   is independent of the installed file, with `controlProtocolVersion` named as
   part of the contract surface this migration sequences.
2. Add a precondition of the same fail-closed shape as P3/P4: enumerate the live
   daemons (`pgrep -fl tb-sessiond`) and their boards **before** the window, and
   state the required disposition for each. A release that does not change
   `controlProtocolVersion` and one that does are different steps.
3. Derive whether `0.25.0`/`0.25.1` change `controlProtocolVersion` from the
   release tree at execution time — `grep -n 'controlProtocolVersion *=' …/types.go`
   against `$SAVED_SOURCE` and `$RELEASE` — and make a change stop the step for a
   decision, rather than discovering it when a daemon is fenced mid-window.
4. Name the daemon-skew window in Part V for `W2` and `W5`, with its real
   duration semantics (until each board's daemon is next touched, unbounded) and
   its impact on the capability list this plan says must keep working.
5. Give R1 a documented disposition for a newer-protocol daemon left alive by an
   abandoned release, and make the R1 canary reach the session manager rather
   than only a config read.
6. State the in-flight behaviour honestly: for `tb-sessiond` the answer is not
   "processes continue" but "the CLI will restart them, on its own schedule,
   when their protocol is older".

I did **not** establish whether `0.25.0` will change `controlProtocolVersion` —
that tag does not exist. That is precisely why it belongs in the plan as a
derived precondition rather than as an assumption in either direction.

### F2 — Medium: the snapshot legally records a `directory` kind for the `~/.claude` and `~/.codex` link trees, saves no tree for it, and R1 then aborts mid-rollback

The Part III snapshot classifies each link-tree entry into three kinds:

```bash
if   [ -L "$root/$skill" ]; then line="$line|symlink|$(readlink "$root/$skill")"
elif [ -d "$root/$skill" ]; then line="$line|directory|"
elif [ -e "$root/$skill" ]; then exit 77
else                             line="$line|absent|"
```

`directory` is a recognised, recorded kind — but only `~/.agents/skills/<skill>`
is ever staged with `rsync`. R1 then calls
`restore_entry "$HOME/.claude/skills/$skill" "$c_kind" "$c_link" ""` with an
empty staged tree, and the `directory` arm begins `test -d "$4"`. Under
`set -euo pipefail` that aborts R1.

Driven against a disposable `HOME` with the R1 block extracted verbatim from the
document (`fences/block23-line1065.bash`), one skill, control and attack
differing only in the recorded kind:

| Recorded `~/.claude` kind | Executable | Role | Install state `repoPath` |
| --- | --- | --- | --- |
| `symlink` (today's shape — control) | `OLD` restored | `OLD-ROLE` restored | `/old/repo` restored |
| `directory` | `OLD` restored | **`NEW-ROLE`** — release's | **`/release/candidate`** — release's |

R1 aborts inside the skill loop: executables are already restored, and **steps 6
and 7 never run**, so the host is left with the abandoned release's roles and its
install state. Nothing detects this before the window — P4 inspects only
`~/.agents/skills`, and only for the release's own skill set; no precondition
looks at the link trees' kinds at all.

Reachability, measured now: all 18 link-tree entries on this host are symlinks,
so it is not reachable today. That is why this is Medium and not High. But the
plan's own standard is the one that convicts it — Part I states that measured
host state is "an observation of one host at one time, not a property of the
plan", and census item 11 derives the kind precisely so the rollback does not
depend on the host shape. Production models the state as possible too:
`install_skills`'s already-installed branch `rm -rf`s a real directory in either
link tree. So the plan derives a kind its rollback cannot consume.

G16 proves an *unrecognised* kind aborts R1. `directory` is a *recognised* kind
that also aborts, and no probe covers it.

Required rework: either stage the link-tree directory content in the snapshot, or
refuse a `directory` kind at snapshot time — before the window, where a refusal
is cheap — or bring the link trees under P4. Any of the three, plus a probe that
is red on the current R1 text.

### F3 — Low: two of the six evidence sections describe running the document's own commands when they ran a reimplementation

Section B/C/D is introduced as: "Every row runs the **revision-6 guard verbatim**
and the **revision-7 precondition** against the same mutant." True for B and C —
I re-ran both and they do exactly that on real production text. False for D:
D1 `cmp`s two files written by the same `printf` (identical by construction, so
"blind" is a tautology, not a measurement of the revision-6 reader), and D2/D3
`diff` two other `printf` literals rather than running the P6 script. The
substantive property is established elsewhere — H1/H2/H9 run the real P6 block
against real Homebrew and show a formula difference is visible — so no conclusion
is wrong; the method claim is.

Section E is titled "P3 and P4 driven against a disposable HOME". P3 is driven.
P4 is not: the validator defines `p4_check`, a function returning an exit status,
while the document's P4 is a print loop that emits `OK`/`REFUSE` and always exits
0. The classification logic is faithfully mirrored, and a print-and-read
precondition is the declared form of revision 7 — but "the document's P4 was
driven" is not what happened, and the manual step the document requires (the
operator appending `project-management` to `release-skill-repos.txt`) is never
exercised.

Section G's prose says the R1 block "was extracted from this markdown file by
line offset", while the evidence preamble says the suite locates R1 and P6 "by
content, not by line number". The validator does the latter
(`grep -l 'r1-release-bins' fences/*.bash`). Sloppy rather than false, but in a
document on its seventh revision for evidence-honesty reasons, the description of
what was run should match what was run.

## Confirmed independently — reproduced, not accepted from the producer

- Fences re-extracted from the committed document: 39 blocks, all parse under
  every applicable shell. Validator: 74/0 under `bash`, 74/0 under `zsh`.
- All fifteen A-row derivations, re-run by hand against production text, return
  exactly what the document records.
- P2's asserted dispositions (`BIN_DIR` unconditional in task-board, external in
  agents-infra and `env -u`'d; the three `*_SKILLS_DIR` top-level and
  unconditional; the ten `TASK_BOARD_*` names matching the ten `-u` names)
  all hold on production text.
- The `v0.6.0` removal gate names specific methods, types and adapter call paths
  and requires `required_by_0.25.0=false` / `required_by_v1.7.0=false` per item,
  with `unknown`, failed read, partial inventory, surviving symbol or init-only
  reach all blocking. Derived from invoked surfaces, not from absence of direct
  imports. Unchanged and still adequate.
- Every step, 0 through 6, states in-flight behaviour and carries a concrete
  rollback command sequence, including Step 5.
- Every operational rollback is labelled **UNTESTED**, and the R1/R2 asymmetry is
  stated honestly: R1's logic was exercised on fake artifacts, R2 has no
  execution evidence at all and the document says so in those words.
- The delta touches only `.research/…` and `LOGBOOK.md`. No release, tag,
  install, migration, Homebrew change or installed-runtime mutation.

## What I did not re-derive, stated as unknown rather than inferred

- I did **not** rebuild task-board against public `v0.5.0` and did not re-derive
  the 555-symbol count, the 20-package graph or the
  `GOWORK=off go test -mod=mod ./internal/spawn -count=1` exit 0. That section is
  unchanged since document revision 3 and was reproduced by the revision-2
  review; the document itself declines to re-claim it. I make no independent
  claim about it here.
- I did not execute either installer, and the operational rollbacks remain
  unrehearsed. Nothing in this review changes their **UNTESTED** status.
- I did not establish what `controlProtocolVersion` will be in `0.25.0` or
  `0.25.1`; those tags do not exist. F1 asks for a derived precondition, not for
  a guess.
- I did not exercise an actual CLI/daemon protocol mismatch end to end. F1 rests
  on the production source of the compatibility check and its call site, plus the
  measured fact that two daemons are live, not on a staged mismatch.

## Scope and safety

No release, tag, install, migration, Homebrew change or installed-runtime
mutation was performed. Both installers were read, never executed. All probes ran
under `.temp/TASK-260830-s5ro4e-review6/` and disposable `HOME`s in `$TMPDIR`. No
process belonging to another run was signalled — the two live `tb-sessiond`
daemons were observed with `pgrep` and left alone. No credential, token, cookie
or keychain value was read, printed or persisted.

## Evidence

- `.temp/TASK-260830-s5ro4e-review6/` — independently re-extracted `fences/`,
  `val-bash.log`, `val-zsh.log`, `review6-evidence.log`.
- Attached: `TASK-260830-s5ro4e_review-evidence-rev6.log`.
