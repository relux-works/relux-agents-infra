# TASK-260830-s5ro4e review verdict — CR revision 8 (document revision 9)

Verdict: **changes requested**. Route to `analysis` for a corrected plan and a
new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-8` revision 8, base
`f083f6669634445ce71746745c61bf1e2e593839`, candidate tree
`f7872fb33959078ca31d9491eea408bdb3fd02f4`, repository delta **present** —
and the delta is one stray file, see F2. Reviewer run `RUN-260830-f0bcd7`.
Production sources read at `skill-project-management` `f1319eff`.

## On the repository delta

The base OID **is** the producer's own commit again. `f083f66` ("plan(lockstep):
revision 9") is the real work: `.research/260830_agents-management-lockstep-release-and-rollback.md`
(+436/-66 across 489 changed lines) and `LOGBOOK.md` (+13/-2). No Go, shell,
module or configuration file. No release, tag, install, migration, Homebrew
change or installed-runtime mutation. I reviewed `cbe69f3..f083f66` as the real
delta. The committed document is byte-identical to the board resource
`TASK-260830-s5ro4e_lockstep-release-and-rollback-plan.md` (3077 lines).

The snapshot's own `repository_delta` is `val-zsh.log`, a one-line file
containing `zsh: can't open input file: ./validate-revision9.sh` — the residue
of a validator run launched from the wrong directory. That is F2.

## Revision 9 closes both revision-8 findings, and I reproduced that

Not accepted from the producer — re-derived.

- **F1 (the newer-daemon refusal is unreachable) — closed, and closed correctly.**
  I re-read `client.go`, `stale_takeover.go` and every call site at `f1319eff`
  rather than trusting section L. `managerProtocolCompatibility` has exactly two
  call sites (`stale_takeover.go:51`, `:127`), **neither inside**
  `takeOverUnresponsiveManager` (0 occurrences in its body). `ConnectOrStart`
  carries one `errors.As` against `*ManagerProtocolUpgradeError`; the newer
  branch's bare `fmt.Errorf` matches nothing and is discarded. All three
  production callers pass a non-nil launch closure (`cmd/session.go:366`,
  `cmd/codex_manager.go:92`, `cmd/claude_manager.go:115`), so `launch == nil` is
  dead. `takeOverUnresponsiveManager` dials with `dialManagerStatus`, and the
  three-branch table in Part I matches the function line for line:
  `StartupHealthy` → `return client, nil, nil`; not healthy and past
  `startupHealthyDeadline` → `terminateStaleInstance`; lock not held →
  `(nil, nil, nil)`. `sessionmanager.Connect` has exactly one `cmd/` occurrence,
  the function value at `cmd/context_security.go:75`. Zero test files name the
  newer-than-wrapper refusal, while the two older-direction tests exist by name —
  L21's gap is real and not a grep artefact.
  The correction landed in all six required places, and the two facts the
  producer added (`Connect` via `context_security.go`, and
  `takeOverManagerForProtocolUpgrade` after the fact) narrow the claim rather
  than widen it. "Unreachable from every production `ConnectOrStart` call site"
  is the accurate form and is what the document now says.
- **F2 (section J read revision 7 from `HEAD`) — closed.** Pinned at `0425fb7`
  with `J0b` asserting the pin is not `HEAD`; L28 pinned at `cbe69f3`. I
  re-extracted 43 fences from the committed document with the shipped
  `extract-fences.py` (all parse: `bash -n` + `zsh -n` on 40 `bash` fences,
  `zsh -n` on 3 `zsh` fences) and ran the shipped validator unmodified:
  **204 pass / 0 fail, exit 0 under `bash`; 204 / 0, exit 0 under `zsh`.**
  Revision 8's failure mode — red the moment its own subject is committed — does
  not recur.
- **The `LOGBOOK.md` correction is a correction, not a contradiction beside the
  old claim.** New entry `2410` states the root cause, and entry `2350`'s
  downgrade FINDING carries an inline **SUPERSEDED by 2410** clause naming which
  three reads were true-but-insufficient. Same treatment previously applied to
  `1815` and `1529`.

The rejection below is not a regression. It is the same failure mode as
revisions 1, 7 and 8, one level further down: the census is right and the
instruction derived from it is not.

## Findings

### F1 — High: the `ConnectOrStart`-class prohibition, which is now revision 9's entire mitigation for the downgrade direction, enumerates the command family both incorrectly and incompletely

Revision 9's central operational move is to replace a refusal it proved does not
exist with a prohibition: *"run no `ConnectOrStart`-class command against that
board, recovery-prefixed or not"*. The document says so five times, and defines
the family by enumeration each time — including inside the R1 rollback fence
itself (line 1753), where the list is what an operator reads while executing:

> `# command against it -- goal-bound spawn, session doctor, session logs, session`
> `# reclaim-contexts, session context, or the managed codex/claude wrappers --`

The `--` makes that gloss exhaustive, and it is the sole safety instruction for
the one state the plan says R1 cannot fix. **It is wrong on both ends.**

**Three real `ConnectOrStart`-class subcommands appear nowhere in the 3077-line
document.** All three call `connectOrStartSessionManager` and therefore reach
exactly the unchecked-client-or-SIGKILL behaviour revision 9 exists to warn
about:

| Command | Call site | Mentions in the plan |
| --- | --- | ---: |
| `task-board session context-completeness [CTX-ID]` | `cmd/session.go:245` | **0** |
| `task-board session context-economics` | `cmd/session_context_economics.go:58` | **0** |
| `task-board session sweep-contexts` | `cmd/session_context_retention.go:50` | **0** |

Narrowing control: `reclaim-contexts` appears 6 times and `session doctor` 8, so
those three zeros are gaps, not a grep artefact.

**And one command in the prohibition is not `ConnectOrStart`-class at all.**
`session logs` (`cmd/session.go:126-141`) neither starts nor dials a daemon: it
calls `streamSessionManagerLog(..., layout.OperatorLogFile, follow)` and reads a
file. Its body contains 0 `connectOrStartSessionManager` and 0 `Dial(`.

**The two errors have one cause, and the plan contains the evidence that
contradicts itself.** The Part I call-site table (line 561) is complete — it
cites `cmd/session_context_economics.go:58`, `cmd/session_context_retention.go:50`
and `cmd/spawn_context_lifecycle.go:292` by file and line. The same row then
glosses `cmd/session.go:151,199,245` as ``(`logs`, `doctor`, `reclaim-contexts`)``.
The real mapping is `:151` → `doctor`, `:199` → `reclaim-contexts`, `:245` →
`context-completeness`. That single mislabel carried `logs` (harmless,
over-inclusive) into every downstream prohibition sentence and dropped
`context-completeness` (real) out of them, and nothing ever walked the two
`session_context_*.go` file:line citations back to their command names. The
census saw them; the instruction does not carry them.

Also: three of the prohibition sentences write ``session context``, which is not
a command. The binary registers `context-completeness`. An operator typing what
the plan wrote gets `unknown command`, and a careful one who resolves it against
`--help` finds a name the plan never uses.

**Why this is load-bearing rather than a wording nit.** The plan's own framing
makes the enumeration the safety boundary — it states in two places that the
`Dial`-based reads are *"the only reads that are"* safe. The three omitted
commands are all report-shaped (`context-completeness` prints re-exploration
feedback, `context-economics` prints a cost report, `sweep-contexts` reports
expiring payloads), which is precisely the class an operator under R1 would
reach for while deciding what to do, believing them to be reads. Running one
against a board whose daemon is newer than the restored CLI produces the two
outcomes revision 9 documents: an unchecked live client issuing v4 calls at a v5
daemon, or SIGTERM/SIGKILL of a daemon holding live sessions. The plan says it
cannot choose between them and that the blast radius is UNKNOWN.

This is the same shape as revision 8's F1, moved one level down. There the guard
was present and uncalled; here the census is present and the derived instruction
is incomplete. Revision 9's own new lesson 6 — *"for every guard a plan relies
on, name the production entry point that reaches it, and probe the path"* — was
applied to the guard and not to the prohibition that replaced it.

Mechanically derived, 26 probes, 0 failures, attached as
`TASK-260830-s5ro4e_prohibition-completeness-probe.sh` / `.log`.

Required rework:

1. Correct the Part I call-site table gloss at line 561: `cmd/session.go:151,199,245`
   are `doctor`, `reclaim-contexts` and `context-completeness`. `session logs` is
   a file read and belongs with the `Dial`-free reads, not in the
   `ConnectOrStart` row.
2. Make every prohibition enumeration the complete family, derived from the
   `connectOrStartSessionManager` census rather than restated by hand — all five
   sites: Part I "Daemon newer than CLI" (594-595), Part II point 3 (1324-1326),
   the R1 fence comment (1753-1755), the R1 canary reading (1779-1781), and the
   revision summary (2799-2801). The family is `session doctor`,
   `session reclaim-contexts`, `session context-completeness`,
   `session context-economics`, `session sweep-contexts`, the managed
   `codex`/`claude` wrappers, and `spawn` — with the qualifier below.
3. Use the real command name `context-completeness`; drop `session context`.
4. Either drop `session logs` or keep it with an explicit note that it is
   included out of caution and is not in the family — do not let a wrong member
   stand as evidence the list was derived.
5. Narrow the `spawn` qualifier consistently. The prohibition sentences all say
   *goal-bound* `spawn`, but `requiresManagedSpawnSession`
   (`cmd/codex_goal_spawn.go:74-79`) is `cfg.LaunchGoal != nil ||
   strings.TrimSpace(cfg.ContextCtxID) != ""`, and `cmd/spawn.go:2175-2180`
   materializes a context whenever `contextPlan.Active`, reaching
   `connectOrStartSessionManager` through `cmd/spawn_context_lifecycle.go:292`.
   The plan's *census* sentences already say "goal-bound or writable-context
   spawn" (lines 564, 1299) — three times in total — so this is the prohibition
   narrowing what the document already knows, not the document being unaware.
   Make the prohibition say the same thing the census says.
6. Add the derivation to section L as a probe, not as prose: the complete
   `connectOrStartSessionManager` caller set (7 sites), each mapped to its cobra
   `Use:` string, with a control row proving a `Dial`-only command
   (`session status`, `session logs`) is correctly excluded. A hand-written
   enumeration of a safety boundary is the same class of evidence as a presence
   probe for a guard.

I did **not** run any of the three omitted commands, did not stage a protocol
mismatch, and signalled no process. F1 rests on the cobra command definitions,
the `connectOrStartSessionManager` call graph and the plan's own text.

### F2 — Low: the Change Request's repository delta is a stray artifact that would be committed on acceptance

`repository_delta: present` is entirely `val-zsh.log` at the branch root, one
line: `zsh: can't open input file: ./validate-revision9.sh`. It is the residue
of a validator invocation launched from the wrong working directory; the real
zsh run succeeded and is attached as
`TASK-260830-s5ro4e_validate-revision9-zsh.log` (13967 bytes, `pass=204 fail=0`).
Nothing in the plan references it. Accepting this revision hands the orchestrator
a scope whose only file is that leftover. Delete it before the next snapshot.

Not a defect in the plan and not evidence of anything false — the producer's
attached logs are honest records of runs that really happened, and I reproduced
both. Recorded so the next snapshot is clean.

## Confirmed independently — reproduced, not accepted from the producer

- 43 fences re-extracted from the committed document; all parse under every
  applicable shell. Shipped validator, run by me from a fresh copy against the
  committed document: **204 / 0, exit 0 under `bash`; 204 / 0, exit 0 under
  `zsh`.** Section J's `0425fb7` pin and L28's `cbe69f3` pin both resolve, and
  `J0b`/`L28f` assert the pin is not the document under test.
- Section L's reachability claims are accurate on production text at `f1319eff`.
  I re-derived L11-L12 (2 call sites, neither in `takeOverUnresponsiveManager`),
  L7-L9 (3 of 3 non-nil launch closures), L19-L20 (`Connect`: 1 `cmd/`
  occurrence, a function value), L15-L17 (the two takeover branches, `SIGTERM`
  then `SIGKILL`) and L21-L23 (0 test files naming the newer-daemon refusal, 2
  named tests for the older direction) myself rather than reading the rows.
- `W2d`'s impact cell is corrected in the right direction: in the older-daemon
  direction `ConnectOrStart` takes over and succeeds, the `Dial`-based commands
  never check compatibility, and the window's impact is silence. The revision-8
  self-contradiction is gone.
- `LOGBOOK.md` entry `2410` replaces rather than contradicts: `2350`'s downgrade
  FINDING carries an inline `SUPERSEDED by 2410` naming which reads were
  true-but-insufficient.
- The four consumption-surface instruments are separately derived and answer
  different questions — direct tracked-source imports (25 prod / 36 test files,
  13 package families), `go list -deps` (20 compiled packages), `go tool nm`
  (555 module symbols, broken down per package), and production-path inspection
  plus preflight. The document explicitly declines to re-claim the census in
  revision 9 and says so under "what this revision did not do".
- The `v0.6.0` removal gate names specific methods, types and adapter call paths,
  requires `required_by_0.25.0=false` **and** `required_by_v1.7.0=false` per
  manifest item, and blocks on `unknown`, failed read, partial inventory,
  surviving symbol, init-only reach or production-path reach. It requires
  negative narrowing of each removed adapter and calls a delete-only result
  insufficient. Unchanged since revision 5 and still adequate.
- Windows: `W2`, `W2d`, `W3`, `W5`, `W6`, `Wimg`, the two Homebrew/LLDB bypass
  guards and the three `0s` dynamic-mismatch rows. I looked for a fifth
  installer window — partially applied, interrupted, or a consumer repinned
  before its dependency published — and did not find one the table does not
  name. The gap I found is one level below a window: a member missing from the
  prohibition that the windows hand to the operator.
- Steps 0-6 each state in-flight behaviour, carry a concrete rollback command
  sequence (including Step 5: R1 with `RECOVERY_ROOT=…/bridge-pair` plus the
  recovery-PATH `agents-infra verify global`, and R2 if the agents-infra half is
  red), and carry a `Rollback status:` line. Every operational rollback is
  labelled **UNTESTED**; Step 1's is `PARTIALLY TESTED` and says for what.
- The delta touches only `.research/…` and `LOGBOOK.md`.

## What I did not re-derive, stated as unknown rather than inferred

- I did not rebuild task-board against public `v0.5.0` and make no claim about
  the 555-symbol count, the 20-package graph or the `./internal/spawn` gate.
  Unchanged since document revision 3.
- I did not execute either installer. The operational rollbacks remain
  unrehearsed and **UNTESTED**; nothing in this review changes that.
- I did not establish what `controlProtocolVersion` will be in `0.25.0` or
  `0.25.1`. Those tags do not exist.
- I did not establish whether a protocol-5 daemon would populate
  `startup_healthy`. It decides which of the two takeover branches fires, and it
  is unknown, not estimated. The document says the same.
- I did not run task-board's Go test suite, and I did not run any of the three
  omitted commands against any board.
- One branch I noticed and am reporting as unresolved rather than as a finding:
  in the third takeover outcome (lock not held) `ConnectOrStart` calls `launch()`
  and then re-enters `takeOverUnresponsiveManager` on every poll iteration, so
  that outcome can degrade into either of the other two inside the same command.
  The plan's table reports only the terminal `context deadline exceeded`.
  Whether the intermediate transition is reachable in practice depends on lock
  arbitration timing I did not measure.

## Scope and safety

No release, tag, install, migration, Homebrew change or installed-runtime
mutation was performed. Both installers were read, never executed. No process
was signalled and no other run's processes were touched; the validator's daemon
observations are `Dial`-based `session status` reads that start nothing. All
probes ran under `.temp/TASK-260830-s5ro4e-review9/` and disposable `HOME`s in
`$TMPDIR`. No credential, token, cookie or keychain value was read, printed or
persisted.

## Evidence

- `TASK-260830-s5ro4e_prohibition-completeness-probe.sh` / `.log` — 26 probes,
  0 failures, exit 0; the F1 derivation with narrowing controls at D4, D9, E1-E2.
- `TASK-260830-s5ro4e_review-evidence-rev8-bash.log` /
  `TASK-260830-s5ro4e_review-evidence-rev8-zsh.log` — my independent 204/0 runs
  of the shipped validator against the committed document.
- `.temp/TASK-260830-s5ro4e-review9/` — re-extracted `fences/` (43), both
  validator runs, the probe and its log.
