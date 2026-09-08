# TASK-260830-s5ro4e — review verdict, document revision 11 (CR revision 9)

**Verdict: CHANGES REQUESTED.** The compression did *not* lose a safety property — that
was the question I was asked first, and the answer is clean. It introduced two new ones,
both in material that revision 11 rewrote, both demonstrated by driving the shipped text
rather than reading it.

Reviewed: `6e26e0b` (`.research/260830_agents-management-lockstep-release-and-rollback.md`
762 lines, `_runbook.md` 340, `_review-record.md` 162), against `f083f66` (revision 9,
3077 lines).

## On the empty repository delta

The Change Request reports `repository_delta=empty` because its base OID `6e26e0b` is the
commit that *contains* revision 11. The work is real and reviewable as
`git diff f083f66 6e26e0b` — 3559 lines changed in the plan, plus two new companion files.
This is a snapshot-timing artifact, not a producer that changed nothing, so the emptiness
is not itself a finding either way.

## The compression audit — the primary question, and it passes

Classified every deletion from revision 9 as elaboration or property.

**Properties, all still present as a precondition with one command and one expected
result, a named prerequisite, or a stated limit:**

- All 17 census derivations survive as *live* derivations: P5's two `sed`s (rows 1, 4-6),
  snapshot `roles.txt` (2, 3), `repoPath`/`binDir` from both install states (7, 8),
  `w2/w3/w5/w6-*.txt` stage capture (9, 15), the `SKILL_REPOS` read (10, 11), the
  agents-infra install state (12), P6's whole formula list (13), P2's sigil set (14),
  staged link trees + `snapshot_restorable` (16), P7's protocol derivation (17).
- The four literal categories and why each is safe: collapsed into the two rules in
  *How to use it* plus the `git describe --tags --exact-match` assertions. The
  derived-not-literal principle is applied **harder** than in revision 9 — the `W2`
  planning-time stage enumeration and the `ConnectOrStart` command family both stopped
  being lists and became `sed`/`grep` against the release.
- Every step keeps version, must-remain-working capabilities, rollback commands, window
  and an UNTESTED status. Every window keeps duration and impact. `W2d`'s corrected
  "the impact is silence" survives verbatim in substance.
- The corrected consumption census survives with the direct-import claim in the correct
  direction: "**Absence of direct imports is not an adapter-removal gate**", and Step 4
  repeats "A zero direct-import count cannot satisfy this gate." No creep back.
- Stop conditions: revision 9's list plus five new ones for step 0d. Strictly wider.
- Limits: all five, plus Limit 6, with 1/2/3/4/6 collapsed into P8.

**UNTESTED labels are in the documents, not only in the handoff.** Plan: `Status:
UNTESTED (P8)` on R1; "**R2 is more unknown than R1** … no execution evidence of any
kind"; the four-row exercised/real/status table; UNTESTED in every Part IV step row;
Limits 1. Runbook header: "**Every script here is UNTESTED against a real installed
pair.**" Confirmed.

**The downgrade direction took the required route.** Not a third prose prohibition: the
dangerous state is defined as a state rather than a command family — "*a live daemon
speaking a newer protocol than the `task-board` binary on disk*" — and R1 **step 0d**
stops the daemon through its own SIGTERM/`Drain` shutdown arm before any executable
moves, so the combination never exists. Step 0d refuses attached clients, never escalates
to SIGKILL, and R1 step 5 asserts the state is still gone. Prevention (P7c) still runs
first and 0d is explicitly not a licence to skip it. This is the engineering answer.

**Things that moved landed somewhere real.** The four scripts are in the runbook and match
the plan's prose refusals; I extracted all four fences and diffed them against the
producer's own probe copies — verbatim, so their evidence was driven against the shipped
text. The revision narrative, census and lessons are in the review record. PRE-1…6 name an
owner each.

## F1 — BLOCKING. P2 fails open under zsh for Steps 3 and 6

Revision 9 had **two** P2 commands: task-board over `setup.sh` plus the one file it
sources by name, and agents-infra over **`setup.sh` alone**. Revision 11 merged them into
one generic command that globs `"$…/scripts/lib/"*.zsh` on both sides.

`relux-agents-infra` has no `scripts/lib/` directory. Under zsh — the operator's shell,
and the shell both installers declare — an unmatched glob aborts the whole command, so
`set_of` never runs on either side, `diff` compares two empty streams, and the probe grep
never executes.

Driven against the real agents-infra installer with one injected undecided override
`NEW_UNDECIDED_INPUT="${SOME_BRAND_NEW_OVERRIDE:-x}"` in the release copy:

| Shell | stdout | exits | Verdict |
| --- | --- | --- | --- |
| bash | `> SOME_BRAND_NEW_OVERRIDE` | `diff-exit=1`, `grep-exit=2` | stops the step — correct |
| zsh | *(none)* | `diff-exit=0`, `grep-exit=1` | **passes**, having read nothing |

zsh's output is *exactly* the plan's stated expected result — "no output from either (the
second exits 1 from grep finding nothing)". An operator following the written expectation
gets a green from a precondition that read zero bytes. This is the failed-read-as-absence
shape the plan itself forbids in its own second global rule, in one of only two
preconditions guarding the agents-infra installer's input surface.

**Where the fix lives:** in P2, in the plan. Either restore revision 9's per-installer
form (agents-infra: `setup.sh` alone), or make the operand set explicit and fail closed
when it is empty. Also state which shell the Part II preconditions are run under — that is
the root enabler, and it is currently unstated anywhere in the document.

## F2 — BLOCKING. R1 step 0d's attached-clients refusal admits an absent field

`att` is computed as `sum(int(r.get("attached_clients") or 0) …)` over
`d if isinstance(d,list) else d.get("sessions") or []`. Every other field the step reads
is indexed strictly (`["pid"]`, `["board_fingerprint"]`) or validated with an explicit
stop arm (`running`). The one field the third-party refusal depends on is not.

Isolated against the shipped `r1-0d.sh` with a live fake daemon whose `ps` command matches,
status reporting `session_count: 4`:

| `session list` payload | Result |
| --- | --- |
| `{"sessions":[{"attached_clients":0}]}` | terminated — control, correct |
| `{"sessions":[{"attached_clients":2}]}` | `STOP /b1 has 2 attached client(s)` — correct |
| `{"sessions":[{"session_id":"s1"},{"session_id":"s2"}]}` | **SIGTERM sent**, recorded `attached=0` |
| `{"items":[{"attached_clients":9}]}` | **SIGTERM sent**, recorded `attached=0` |
| `{"sessions":null}` | **SIGTERM sent**, recorded `attached=0` |

Three absent/malformed shapes are read as a measured zero, the daemon is stopped, and
`r1-daemons-stopped.txt` records `attached=0` as a fact that was never read — fabricated
evidence in the rollback record, in the step whose whole point is "recorded **only after**
the stop is confirmed".

I checked reachability rather than asserting it. With a matched pair this cannot fire:
`printSessionList` (`cmd/session.go:585-591`) always emits `{"sessions": [...]}` and
`SessionView.AttachedClients` (`types.go:57`) is a plain `int` with no `omitempty`. The
exposed path is the one 0d exists for — a **protocol-mismatched** board, where the Go
client's lenient decode can zero a renamed or dropped field before the shell ever sees it.
The shell-level half is fixable here; the Go-level half is not, and belongs next to PRE-1
or as a seventh Limit rather than being left implied.

**Where the fix lives:** index the field strictly so a missing key is a non-zero exit and
`set -euo pipefail` stops R1, require `sessions` to be present, and cross-check
`len(rows)` against status `session_count` — a row count that disagrees is `unknown`, not
zero. Then say plainly, in Limits, that a cross-protocol `attached_clients` read cannot be
trusted at all.

## What I checked that held up

- **`set -e` and the `test && action` / `grep -q … && { … exit 1; }` idioms.** Driven in
  both bash and zsh: the four shapes in RB-2/RB-3 do not abort on the false branch. The
  producer's E9 reached the same result by measurement; I reproduced it independently
  rather than taking it.
- **The daemon census has no third launch shape.** `launchSessionManagerDaemon`
  (`cmd/session.go:480-506`) passes either `--remote-origin …` or `--board-dir …` and
  nothing else, so 0d's `sed`-plus-`--remote-origin` pair covers production. A synthetic
  `tb-sessiond` line with neither is silently skipped (`rc=0`, nothing stopped), but I
  could not reach that shape from production and am recording it as defensive-only, not
  as a finding.
- **Producer probes ran against the shipped text.** All four runbook fences byte-match
  `rb1…rb4.sh`, and `r1-0d.sh` is a verbatim slice of `rb3-r1.sh`. E4's seven refusals,
  E5's positive control and E6's two narrowing mutants are real negative evidence.

## Minor, not blocking, fix while you are in there

- Step 3's *Must remain working* cell dropped "agents-infra full tests/build/verify plus
  real target/compose/Pi refusal gates on the exact release head", which Step 6's cell
  kept. Restore the symmetry.
- Step 5's rollback lost the `env -u AGENTS_INFRA_SOURCE_DIR` prefix on
  `agents-infra verify global` that revision 9 carried; RB-4 still has it for R2's own
  calls.
- P5's `diff <(sort …/binaries.txt) <(sort release-set.txt)` still names a file none of
  the three preceding commands produces. Pre-existing since revision 9, not a compression
  regression, but the AC asks for a concrete command sequence.

## Not the producer's fault, and not a reason to re-open the compression

Both findings are in text revision 11 wrote, not in text it deleted. The 800-line budget
did what it was meant to do: I read this plan end to end and could hold it, which is not
true of revision 9. Keep the form. Two fixes and this is done.

## Evidence

- `TASK-260830-s5ro4e_review-evidence-rev11.log` — F1 under both shells with the injected
  override; F2's five isolated payloads with the daemon's real fate each time.
- Reproduce: `.temp/review-rev11/{rev-isolate.sh,rev/run.sh}` against the extracted
  `r1-0d.sh`; `/tmp/p2t/p2.sh` for F1.
- No migration step, release, install or tag was performed. No live daemon was signalled;
  every process stopped in these probes was a fake spawned by the probe itself. No
  credential, token or environment value was read, printed or persisted.
