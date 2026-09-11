# Lockstep plan — review record

Companion to `260830_agents-management-lockstep-release-and-rollback.md`
(`TASK-260830-s5ro4e`). This file holds what a reviewer needs and an operator does not:
revision history, the finding that drove each revision, the evidence index, and the lessons.
The operating plan does not depend on it.

## Revision 12 — two fail-open reads the compression introduced

The revision-11 review confirmed the compression lost **no** safety property and asked for two
fixes, both in text revision 11 had rewritten. Both are closed here; nothing else changed, and
the plan is still 800 lines.

**F1 — P2 failed open under zsh (Steps 3 and 6).** Revision 11 merged revision 9's two
per-installer P2 commands into one that globbed `"$…/scripts/lib/"`\*`.zsh` on both sides.
`relux-agents-infra` has no `scripts/lib/` at all, and under zsh an unmatched glob aborts the
whole command: `set_of` never ran, `diff` compared two empty streams, the probe `grep` never
executed, and the result was byte-for-byte the empty output the plan documented as a pass.

Reproduced against the real agents-infra installer with one injected undecided override
`NEW_UNDECIDED_INPUT="${SOME_BRAND_NEW_OVERRIDE:-x}"`:

| Shell | revision 11 | revision 12 |
| --- | --- | --- |
| bash | `SOME_BRAND_NEW_OVERRIDE` printed, `diff-exit=1` — stops | `SOME_BRAND_NEW_OVERRIDE` printed, no sigil — stops |
| zsh | *no output*, `diff-exit=0` — **passes having read nothing** | `SOME_BRAND_NEW_OVERRIDE` printed, no sigil — stops |

The fix is revision 9's shape, not a tolerance: each installer gets its own **named** operand
list, `set_of` refuses an operand it cannot read, and P2 passes only on an explicit
`P2-SET-OK` sigil, so an empty or unread set can no longer look like a clean one. Part II now
also states the shell every precondition is run under — the root enabler, previously unstated.
Both fences were extracted from the shipped plan and driven in **both** shells across five
cases (identical trees, an override injected into task-board's *sourced* file, an override
injected into agents-infra's `setup.sh`, and the F1 shape of a named operand that is missing):
identical verdicts in bash and zsh, `P2-SET-OK 38`/`19` on the clean pairs.

**F2 — R1 step 0d's attached-clients refusal admitted an absent field.** `att` was
`sum(int(r.get("attached_clients") or 0) …)`, so an absent key, a missing `sessions` list or a
`null` read as a measured zero: the daemon was stopped and `r1-daemons-stopped.txt` recorded
`attached=0` as a fact nobody had read. It is now indexed strictly, `sessions` must be a list,
and `len(rows)` is cross-checked against `session_count + quarantined_count` — `List()` returns
`Count()` plus `QuarantinedCount()` rows (`manager.go:959-981`), so a disagreeing count is
`unknown`, not zero.

Driven against the 0d slice extracted verbatim from the shipped runbook, with a fake daemon the
probe spawned itself and a stubbed `pgrep` so no real process could ever enter the census:

| `session list` payload | revision 11 | revision 12 |
| --- | --- | --- |
| 4 rows, all `attached_clients:0` (control) | terminated | terminated, `attached=0` — a real read |
| a row with `attached_clients:2` | `STOP … 2 attached client(s)` | unchanged |
| rows with the key absent | **SIGTERM, recorded `attached=0`** | `KeyError` → STOP, never signalled |
| `{"items":[…]}`, no `sessions` | **SIGTERM, recorded `attached=0`** | `KeyError` → STOP, never signalled |
| `{"sessions":null}` | **SIGTERM, recorded `attached=0`** | STOP, never signalled |
| 4 rows against `session_count:1` | SIGTERM, recorded `sessions=1 attached=0` | STOP, never signalled |
| `quarantined_count` absent | n/a | `KeyError` → `set -e` stops R1 |
| `attached_clients:"two"` | n/a | `ValueError` → STOP, never signalled |

Two mutants bound it rather than only proving it exists. **Delete** — restoring revision 11's
lenient `.get(…) or 0` — reproduces all three of the reviewer's shapes, terminating the daemon
and recording a fabricated `attached=0`. **Narrow** — keeping strict indexing but removing only
the row-count cross-check — still catches the absent key but lets the 4-rows-versus-1 case
through, recording `sessions=1 attached=0`. So the two halves of the fix cover different
classes and neither is redundant.

The Go half is not fixable in shell and is not claimed to be: **Limit 7** now says plainly that
against a protocol-mismatched daemon a lenient `json.Unmarshal` can zero a renamed or dropped
field before the shell sees it, so a well-formed `attached_clients: 0` may be a decode artifact,
PRE-1 is the structural fix, and until it ships such a zero is **unread** and that board is its
owner's to disposition.

**Minor, as asked.** Step 3's *Must remain working* cell regains the agents-infra
tests/build/verify plus target/compose/Pi refusal gates that Step 6's cell kept; Step 5's
rollback regains the `env -u AGENTS_INFRA_SOURCE_DIR` prefix RB-4 still carries; P5 no longer
names `release-set.txt`, which nothing produced — it writes `p5-binaries.txt`/`p5-roles.txt`
from both halves and diffs those against the manifest.

**Still untested, unchanged.** R1 has never run against a real installed pair and R2 has no
execution evidence of any kind. Every UNTESTED label in the plan and the runbook header stands.
No migration step, release, install or tag was performed. No live daemon was signalled: every
process stopped in these probes was a fake the probe spawned, and `pgrep` was stubbed so the
real daemons on this host could not enter the census.

## Revision 11 — the size correction

Revision 10 was correct and unusable. Nine review cycles of real findings produced a
document that was more accurate and less followable each round:

| Revision | Plan lines | Shell functions in the document |
| ---: | ---: | ---: |
| 5 | 1990 | 12 |
| 9 | 3585 | 6 |
| 11 | **760** (plan) + 340 (runbook) | 4 scripts, in the runbook |

Revision 11 deletes no safety property. Each one is now either a precondition with one
command and one expected result (P1–P8), a named prerequisite (PRE-1…6), or a stated limit.
What left the operating document:

| What left | Where it went | Why |
| --- | --- | --- |
| Revision-by-revision narrative, the "what the review found and what changed" sections, the derived-versus-restated census, the review lessons | this file | it is reasoning about the plan, not instruction for the operator |
| The four verbatim scripts (P6 brew snapshot, Part III snapshot, R1, R2) | the runbook, delivered with the plan | they are executed, not read. Nobody pastes 157 lines of shell out of prose mid-incident; they run a saved script. Every refusal each script performs is still named in the plan — in the R1/R2 step lists and in *Stop conditions* — so the operator knows what should stop without opening the runbook |
| The `ConnectOrStart` command-family enumeration | replaced by a `grep` the operator runs against the release | a future release can extend the family without touching this document, and no procedure depends on the list any more |
| Per-step window definitions duplicated between Part IV and Part V | Part V only | restatement |

**P8 is new and is the honest form of what nine revisions could not fix.** The rehearsal was
always the answer; earlier revisions recorded it as a NOT IMPLEMENTED row and then kept
writing. It is now a precondition with a numbered procedure, so "UNTESTED" has a defined way
to become "tested", and a release that proceeds without it does so as an explicit decision by
the release owner rather than by default.

## Revision history

| Rev | Finding | Resolution |
| ---: | --- | --- |
| 3 | guards without `errexit` in an `if` context accepted failed reads | per-command status tests |
| 4 | a hard-coded recovery baseline that had already gone stale | baseline derived from the installed artifact at execution time |
| 5 | reader fail-opens: three failed `brew` calls produced a fabricated snapshot | fail-closed top-level snapshot script |
| 6 | `extract_function_body` compared a truncated prefix and admitted a changed `write_install_state`; the env-override classifier recognised one of at least four expansion shapes | whole-tree `diff -ru` (P1) and a `$`-sigil set diff (P2) |
| 7 | the harness had become unreviewed software whose own correctness was the larger risk | harness removed; every guard became an operator precondition |
| 8 | `tb-sessiond` modelled as a file, not a daemon; snapshot recorded a `directory` kind with no staged tree, so R1 aborted mid-rollback | daemon modelled from production text; P7; pre-mutation admissibility pass |
| 9 | the "daemon newer than CLI" refusal is unreachable from every production `ConnectOrStart` call site — the plan asserted a protection the product does not have | the downgrade direction restated as *unprotected*; PRE-1 named for the owning repository |
| 10 | the replacement prohibition enumerated the `ConnectOrStart` family both incorrectly and incompletely | prohibition deleted, not corrected: **R1 step 0d** stops the daemon before the executables move, so the dangerous state is never created |
| 11 | 3585 lines is itself a risk: a rollback path nobody can read under pressure protects nothing | this revision |

**The shape that recurred**, and the rule that follows: *the check is present but uncalled
from production.* The engine-kind contract, the External-CI policy gate, the comparison
instrumentation and the wrapper's protocol refusal were all guards that exist, read
correctly, and never run on the path that matters. For every guard a plan relies on, name the
production entry point that reaches it and probe the path, not the text.

Three more lessons worth keeping: a fix that adds machinery moves the defect into the
machinery (revisions 3–6, severity never fell); a plan that models artifacts will miss every
risk that lives in a process (revision 8 — enumerate what is *running*, not only what is
installed); and when writing about a binary you do not own, the plan may only *describe* — a
missing guard is a prerequisite for the owning repository, never a sentence that reads as
though the behaviour exists.

## Evidence for revision 11

Run in this task's worktree on 2026-08-31. Real exit codes; no migration step, install,
release or tag was performed, and both installers were read, never executed.

**E1 — the daemon facts the plan makes load-bearing, re-verified against production text**
at `skill-project-management` `f1319eff`: `controlProtocolVersion = 4`
(`types.go:13`), the `signal.NotifyContext(..., os.Interrupt, syscall.SIGTERM)` handler
(`cmd/tb-sessiond/main.go:137`), `Manager.Close`'s doc-comment invariant that provider hosts
survive and are reconciled (`manager.go:1068-1081`), and `AttachedClients`
`json:"attached_clients"` (`types.go:57`). Exit 0.

**E2 — the `ConnectOrStart` derivation the plan ships instead of an enumeration**, run
against production: 9 call sites plus the definition at `cmd/session.go:360`. Exit 0. This is
what replaces revision 9's wrong list; the plan states it is diagnostic only.

**E3 — `bash -n` on all four runbook scripts.** RB-1 (12 lines), RB-2 (105), RB-3 (156),
RB-4 (23): each **exit 0**.

**E4 — R1 step 0d refusals, driven against a stubbed `task-board` and `pgrep`**, not read.
Seven cases, each asserting both the exit code and the message:

| Case | Result |
| --- | --- |
| `pgrep` exits 2 (failed census) | rc=1, `DAEMON-CENSUS-FAILED` |
| `pgrep` exits 1 (genuine absence) | rc=0, completes |
| a `--remote-origin` daemon is present | rc=1, `STOP remote-board daemon` |
| board reports `running:false` | rc=0, completes without signalling |
| board's `running` flag is unreadable | rc=1, `STOP unreadable running` |
| board reports `attached_clients: 2` | rc=1, refuses |
| the PID does not resolve to a `tb-sessiond` | rc=1, refuses |

`PASS=7 FAIL=0`, suite exit 0.

**E5 — positive control: 0d's stop path driven end to end.** A fake daemon whose `ps`
command matches `tb-sessiond`, a stub reporting `running:true` for the two pre-kill reads and
`running:false` afterwards. Result: `SIGTERM` sent, the process terminated, the post-stop
assertion passed, `0D-COMPLETED`, rc=0, and the board recorded **after** confirmation —
`r1-0d board=/b1 pid=… fingerprint=fp sessions=4 attached=0`. Without this the seven refusals
above would be consistent with a script that always exits 1.

**E6 — narrowing mutants, because a deleted gate proves only that the gate exists.**

- Widening the attached-clients gate from `[ "$att" = "0" ]` to `[ "$att" -ge 0 ]`: the
  unmutated script refuses a board with 2 attached clients; the mutant **admits** it and
  proceeds to the PID check. The gate is load-bearing.
- Collapsing the census `case` so any `pgrep` failure writes `NO-LIVE-DAEMON`: the unmutated
  script exits 1 with `DAEMON-CENSUS-FAILED` on rc=2; the mutant **completes rc=0**. "A
  failed read is never an absence" is load-bearing, not decoration.

**E7 — P4's skill-kind loop, all four arms** against a disposable `HOME`: `directory` → `OK`;
`symlink` → `REFUSE` (the class P4 exists to catch, since the installer `rm -f`s a stale
symlink before a clone that may fail); a plain file → `REFUSE`; absent → `REFUSE`. Exit 0.

**E8 — `snapshot_restorable`, every arm.** Refuses a `symlink` with no recorded target, a
`directory` with no staged tree, and an unrecognised kind; accepts the three valid shapes.
Exit 0. This is the pass that turned revision 7's mid-rollback abort into a refusal at the
door.

**E9 — an `errexit` question raised and settled by measurement, not by reading.** The
scripts use `test && action` and `grep -q … && { … exit 1; }` at statement level under
`set -euo pipefail`. First reading suggested the normal (test-false) case would abort the
rollback. Driven directly: it does not — the failing command is not the one following the
final `&&`, so it is exempt; a bare `false` control **does** abort (rc=1), and a true
condition **does** fire the guard (rc=7). An intermediate probe that wrapped each shape in
`eval` reported the opposite, because `eval`'s own non-zero status is what errexit caught;
that probe measured the wrapper. Recorded because the wrong reading would have "fixed" four
correct guards.

Artifacts: `.temp/TASK-260830-s5ro4e/{errexit-probe.sh,errexit-probe3.sh,0d/,rb*.sh,sr.sh,p4home/}`.

## Evidence carried forward, unchanged

- The consumption census (25/36 files, 13 direct package families, 20 composed packages, 555
  linked symbols, the six-step production path, `./internal/spawn` green from a detached
  worktree). Unchanged since document revision 3 and independently reproduced by the
  revision-2 review; this revision makes no independent claim about it.
- The live-host measurements in the plan's Part I (two daemons at protocol 4 holding 31 and 2
  sessions, 60 and 48 commits behind the installed file; the pre-existing LLDB absence; nine
  roles; eight `SKILL_REPOS` trees, all real directories). Dated observations, re-measured by
  P3/P4/P6/P7 at execution time.
- The `controlProtocolVersion` history: four values in five days.

## What this revision did not do, stated as unknown

- **No rollback was rehearsed against a real installed pair.** E4–E8 drive logic against
  stubs and disposable directories. R2 still has no execution evidence of any kind. This is
  P8 and Limits 1.
- **The Part III snapshot block was not executed** against the real installed pair; its
  derivations were reproduced individually in earlier revisions and its version parsing
  probed, but `git worktree add`, the `cmp` loop and the same-device assertions have not run
  end to end.
- **No CLI/daemon protocol mismatch was staged.** E5 drives 0d's *mechanism* against a fake
  daemon; it does not show a real `tb-sessiond` releasing its lock, nor what happens to
  provider children. Limits 2, 4, 5, 6.
- **The `env -u` invocations were not executed**, so it is not established that the
  installers complete under exactly those environments. Limits 3.
- **`0.25.0` and `v1.7.0` do not exist**, so P7c's release-side operand, the `v0.6.0` removal
  manifest, and the durable-record contract question cannot be answered from any source
  available to this task.
