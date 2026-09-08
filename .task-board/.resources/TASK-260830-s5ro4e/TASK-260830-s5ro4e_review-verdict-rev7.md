# TASK-260830-s5ro4e review verdict — CR revision 7 (document revision 8)

Verdict: **changes requested**. Route to `analysis` for a corrected plan and a
new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-7` revision 7, base
`cbe69f399a24f7a3e61f1b84cc79ae9ed0b91bce`, candidate tree
`df9e827ad3801b4a5af53987128a3280a3d0d6f8`, repository delta **empty**.
Reviewer run `RUN-260830-5532c6`. Production sources read at
`skill-project-management` `f1319eff`.

## On the empty repository delta

Same snapshot artifact as revision 6, not a producer that changed nothing: the
base OID **is** the producer's own commit. `cbe69f3` ("plan(lockstep): revision
8") is the only commit between the revision-7 base and this one, and it touches
exactly two files — `.research/260830_agents-management-lockstep-release-and-rollback.md`
(+837/-60) and `LOGBOOK.md` (+13). No Go, shell, module or configuration file.
No release, tag, install, migration, Homebrew change or installed-runtime
mutation. I reviewed `0425fb7..cbe69f3` as the real delta. The committed
document is byte-identical to the board resource
`TASK-260830-s5ro4e_lockstep-release-and-rollback-plan.md`.

## Revision 8 closes all three revision-7 findings, and I reproduced that

Not accepted from the producer — re-derived. Fences re-extracted from the
committed document with the producer's `extract-fences.py`: **43 blocks**, all
parse (`bash -n` + `zsh -n` on 40 `bash` fences, `zsh -n` on 3 `zsh` fences).
The producer's validator, run by me against that fresh extraction:
**151 pass, 0 fail, exit 0 under `bash`; 151 pass, 0 fail, exit 0 under `zsh`.**

- **F1 (`tb-sessiond` unmodelled)** — Part I now models the daemon, P7 is a real
  fail-closed precondition (`pgrep` exit 1 vs exit 2/3 kept apart as absence vs
  failed read, driven verbatim in section K), P7c derives the constant from both
  source trees, Part V names `W2d` and `Wimg`, and R1 gains a daemon census and
  a canary that reaches the session manager. The contract reads are correct:
  `controlProtocolVersion = 4` at `types.go:12-13`; the four historical values
  `df4201dc`(v1)/`445b391e`(2)/`f8ba3268`(3)/`d66bfea8`(4) on 2026-07-24…07-28;
  the `Dial` and `ConnectOrStart` call-site tables match production exactly;
  `Drain` has zero references under `cmd/`; the daemon is resolved by bare name
  through `PATH`; goal-bound spawn reaches `ConnectOrStart` via
  `codex_goal_spawn.go:43`.
- **F2 (unrestorable snapshot kind)** — closed, and closed wider than reported.
  Section J, re-run pinned at `0425fb7`, drives revision 7's own R1 text against
  revision 8's on identical fixtures: rev 7 aborts *after* restoring executables
  (role and install state left as the abandoned release's), and on the
  neighbouring `case` arm exits **0** while destroying the tree it was restoring
  and leaving a dangling link; rev 8 refuses at the door with nothing mutated
  and names the entry. Five refusals with narrowing controls (J9-J12, J21-J22)
  that still admit a well-formed snapshot. Finding the fail-open one arm over
  from the reported defect is the right instinct.
- **F3 (evidence prose vs what ran)** — closed. D is labelled a fixture, E now
  locates both P4 fences by content and executes them verbatim including the
  manual append, G's prose matches its validator.

The rejection below is not a regression. It is the same failure mode as
revisions 1 and 7, one level deeper: the conclusion that carries the most weight
rests on a partial read of production text, and the branch that was not read
reverses it.

## Findings

### F1 — High: the "daemon newer than CLI" refusal is unreachable from every production `ConnectOrStart` call site. The plan asserts it as fact, and Part I, P7 and R1 step 8 all rest on it.

The document states, in three places, that the downgrade direction fails closed:

- Part I: *"`actual > controlProtocolVersion` … is a hard refusal … There is no
  recovery path in that direction."*
- P7, "Different" disposition, item 2: *"A restored older CLI cannot use that
  board's session plane, cannot drain the newer daemon and cannot replace it."*
- R1 step 8 / Part V `W2d`: *"nothing in the restored build can drain or replace
  the daemon"*, *"After the takeover, R1 cannot restore that board."*

**Production does not do this.** `managerProtocolCompatibility`
(`client.go:714-737`) returns the typed `*ManagerProtocolUpgradeError` for an
*older* daemon but a plain `fmt.Errorf` for a *newer* one. `ConnectOrStart`
(`client.go:128-200`) inspects the error with exactly one
`errors.As(err, &upgrade)` against that typed value. For the newer-daemon error
the match fails, **the error is discarded**, and control falls through to:

```go
if launch == nil { return nil, ErrManagerNotRunning }
healthyClient, takeover, err := takeOverUnresponsiveManager(ctx, layout)
if err != nil { return nil, err }
if healthyClient != nil { return healthyClient, nil }
```

Every production caller passes a non-nil `launch` — `cmd/session.go:366`,
`cmd/codex_manager.go:92`, `cmd/claude_manager.go:115` — so the `launch == nil`
arm that would surface the refusal is dead code in this binary.
`sessionmanager.Connect`, the launch-less compat-checked entry, has **zero call
sites** under `cmd/`; it appears once, as a function value passed into
`cmd/context_security.go:75`.

`takeOverUnresponsiveManager` (`stale_takeover.go:168-266`) performs **no
compatibility check**: it dials with `dialManagerStatus` (`:25-41`), the
unchecked variant, not `dialHealthyManager` (`:43-59`) which is the only place
`managerProtocolCompatibility` runs on this path. So the older CLI meets the
newer daemon and takes one of two branches, decided by a single field:

| `status.StartupHealthy` of the newer daemon | What the older CLI does |
| --- | --- |
| `true` | `return client, nil, nil` → `ConnectOrStart` returns **a live client to a daemon speaking a protocol it does not support**, no error, no check |
| `false` | falls through the recorded/alive/instance re-checks to `terminateStaleInstance` (`:265`) → `requestProcessTermination`, then a forced signal |

Both outcomes contradict the plan. The first is worse than the plan's model in
kind: the plan tells the operator the session plane is *lost* to the restored
CLI, so a wrong command produces a clean refusal. In fact the CLI proceeds and
issues v4 control calls against a v5 daemon. The second is worse in blast
radius: it is precisely the **out-of-band termination** the plan lists in its
own NOT IMPLEMENTED table as *"the only remaining move … its blast radius
decides whether that state is recoverable at all … UNKNOWN"* — arriving
automatically, on a daemon holding 31 sessions, instead of as a decision.

`StartupHealthy` is `startup_healthy` in `ManagerStatus` (`types.go:114-128`).
Whether a protocol-5 daemon still populates it is **unknown** — the status
shape is exactly what a protocol bump changes, and the decoder is lenient
(`client.go:677`), so a renamed or dropped field decodes as `false` and selects
the terminate branch. I state that as unknown, not as a prediction.

Mechanically derived, 30 probes, 0 failures, attached as
`TASK-260830-s5ro4e_downgrade-reachability-probe.sh` / `.log`.

**Why this is load-bearing rather than a footnote.** The plan's own design
creates the state: it runs the saved `0.24.3` pair through a recovery `PATH`
prefix as "the authority" throughout `W2`/`W5`, and R1 restores that pair while
a release-started daemon survives. A saved CLI invoked under the recovery prefix
against a board whose daemon the new CLI already replaced is not an exotic
scenario — it is the plan's primary mitigation running against its primary
hazard. The constant moved four times in five days; the live daemons on this
host are 8 and 5 days old.

**And the plan's own evidence encodes the blind spot.** Section I rows I5-I7
assert the newer branch *exists* in `managerProtocolCompatibility`. I9 checks
that the *older* takeover is reached from `ConnectOrStart`. Nothing checks
whether the newer branch is reachable from anything. That is the standard
negative shape "the check is present but uncalled from production", asserted as
a guarantee. The same asymmetry exists in task-board's tests:
`TestConnectOrStartUpgradesHealthyOutdatedProtocolHolder` and
`TestConnectOrStartTakesOverHolderWithoutStartupHealthy` cover the older
direction and the terminate branch; **no test file names the newer-than-wrapper
refusal at all.**

Required rework:

1. Correct Part I's "Daemon newer than CLI" paragraph to what production does:
   the refusal exists in `managerProtocolCompatibility` and is not surfaced by
   any production entry point; `ConnectOrStart` discards it and falls through to
   `takeOverUnresponsiveManager`, which returns an unchecked live client when the
   daemon reports `startup_healthy`, and otherwise fences it. Name
   `client.go:145-170`, `stale_takeover.go:168-266` and the `launch == nil`
   dead arm.
2. Restate the irreversibility section accordingly. "No rollback" is still the
   right conclusion; the reason is not "nothing can touch that daemon" but
   "the older CLI will drive or fence it without a compatibility check, which is
   not a rollback". Keep the blast radius of a fence as UNKNOWN, and say that
   the plan cannot choose whether it happens.
3. Fix P7's "Different" disposition item 2 and R1 step 8's `protocol_version`
   greater bullet. The instruction to escalate is right; the reason is wrong,
   and the prohibition must widen from "do not run the live spawn canary" to
   "run no `ConnectOrStart`-class command against that board" — `session
   doctor`, `logs`, `reclaim-contexts`, `session context` and the managed
   `codex`/`claude` wrappers, recovery-prefixed or not.
4. Fix Part V `W2d`'s impact cell, which is wrong in the other direction too:
   *"the board runs a daemon whose protocol the installed CLI refuses, so
   goal-bound `spawn`, `session doctor`, `logs`, `reclaim-contexts` and the
   managed wrappers fail on it"* contradicts the same cell's own "restarted
   without operator action". In the older-daemon direction nothing fails —
   `ConnectOrStart` takes over and succeeds, and the `Dial`-based commands never
   check compatibility at all. The window's impact is silence, not failure.
5. Add reachability probes to section I next to I5-I7: the newer branch's error
   type, `ConnectOrStart`'s single `errors.As`, the non-nil `launch` at every
   call site, and the absence of `managerProtocolCompatibility` inside
   `takeOverUnresponsiveManager`. A presence probe for a guard is not evidence
   the guard runs.
6. Correct `LOGBOOK.md` entry `2350`, which now persists the false claim as
   institutional memory — *"a rollback restores every file and still cannot use
   that board's session plane"*, supported by the true-but-insufficient "no
   cobra command reaches `Drain`". Apply the same `SUPERSEDED by …` treatment
   the producer correctly applied to entries `1815` and `1529`.

I did **not** stage a live protocol mismatch, did not build two task-board
images with different constants, and signalled no process. F1 rests on
production source, its call sites and their `launch` arguments — the same
standard section I uses — not on an observed takeover.

### F2 — Low: the attached validator's section J is not reproducible from the shipped artifact, because it reads revision 7 from `HEAD`

`val.sh:422` recovers the revision-7 rollback text with
`git show "HEAD:.research/…"`. That was correct when the producer ran it, with
revision 7 at `HEAD` and revision 8 still in the working tree. Now that revision
8 is committed, `HEAD` is revision 8 and section J compares the document with
itself: running the attached validator unmodified against the committed document
gives **146 pass / 5 fail**, the five being exactly J2, J15, J16, J17, J20 — the
revision-7 contrast rows. Pinning `0425fb7` restores 151/0 under both shells,
which is how I verified the substantive claim.

Not a false claim and not a defect in the plan's logic — the producer's attached
logs are honest records of a run that really happened. But an evidence script
that goes red the moment its own subject is committed cannot be re-run by the
next reviewer, and this document's whole argument is that evidence should be
reproducible rather than accepted. Pin the comparison to the revision-7 commit
by SHA.

## Confirmed independently — reproduced, not accepted from the producer

- 43 fences extracted from the committed document, all parse under every
  applicable shell. Producer validator: **151/0 under `bash`, 151/0 under
  `zsh`**, with section J pinned at `0425fb7`.
- The `tb-sessiond` contract reads in Part I and section I are accurate on
  production text at `f1319eff`, with one exception: the newer-daemon direction
  (F1). Every `Dial` and `ConnectOrStart` call site listed in the Part I table
  matches `grep` exactly; `Drain` has zero `cmd/` references; the constant's
  four historical values and dates reproduce from `git log -L`.
- Section J's contrast is real and is the strongest evidence in the document:
  revision 7's shipped rollback text both aborts mid-mutation on a recorded
  `directory` kind and **exits 0 while destroying the tree** on an empty symlink
  target; revision 8 refuses both before its first mutation, and still admits a
  well-formed snapshot.
- P7a's census keeps `pgrep` exit 1 (absence) and exit 2/3 (failed read) apart,
  driven as the document's own fence in section K with a stub `pgrep`.
- The `v0.6.0` removal gate names specific methods, types and adapter call paths
  with `required_by_0.25.0` / `required_by_v1.7.0` per item, and treats
  `unknown`, a failed read, a partial inventory, a surviving symbol and
  init-only reach as blocking. Unchanged since revision 5 and still adequate.
- Every step 0-6 states in-flight behaviour and carries a concrete rollback
  command sequence, including Step 5.
- Every operational rollback is labelled **UNTESTED**, and the R1/R2 asymmetry
  is stated in those words. The "What this revision did not do" section is
  unusually honest and I found nothing in it that overclaims.
- The delta touches only `.research/…` and `LOGBOOK.md`.

## What I did not re-derive, stated as unknown rather than inferred

- I did not rebuild task-board against public `v0.5.0` and make no claim about
  the 555-symbol count, the 20-package graph or the `./internal/spawn` gate.
  Unchanged since document revision 3; the document itself declines to re-claim
  it.
- I did not execute either installer. The operational rollbacks remain
  unrehearsed and **UNTESTED**; nothing in this review changes that.
- I did not establish what `controlProtocolVersion` will be in `0.25.0` or
  `0.25.1`. Those tags do not exist, which is why P7c is right to derive it.
- I did not establish whether a protocol-5 daemon would populate
  `startup_healthy`. That field decides which of F1's two branches fires, and it
  is unknown, not estimated.
- I did not run task-board's Go test suite.

## Scope and safety

No release, tag, install, migration, Homebrew change or installed-runtime
mutation was performed. Both installers were read, never executed. No process
was signalled: the two live `tb-sessiond` daemons were observed read-only by the
producer's validator (`session status`, which uses `Dial` and starts nothing) and
left alone. No other run's processes were touched. All probes ran under
`.temp/TASK-260830-s5ro4e-review7/` and disposable `HOME`s in `$TMPDIR`. No
credential, token, cookie or keychain value was read, printed or persisted.

## Evidence

- `TASK-260830-s5ro4e_downgrade-reachability-probe.sh` / `.log` — 30 probes,
  0 failures, the F1 derivation.
- `TASK-260830-s5ro4e_review-evidence-rev7-bash.log` /
  `TASK-260830-s5ro4e_review-evidence-rev7-zsh.log` — independent 151/0 runs.
- `.temp/TASK-260830-s5ro4e-review7/` — re-extracted `fences/`, unpinned run
  (146/5), pinned runs (151/0 ×2).
