# Lockstep plan — review record

Companion to `260830_agents-management-lockstep-release-and-rollback.md`
(`TASK-260830-s5ro4e`). This file holds what a reviewer needs and an operator does not:
revision history, the finding that drove each revision, the evidence index, and the lessons.
The operating plan does not depend on it.

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
