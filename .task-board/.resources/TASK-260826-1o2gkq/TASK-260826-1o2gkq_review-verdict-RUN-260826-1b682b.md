# TASK-260826-1o2gkq — reviewer verdict, CR rev2 (`story_final`)

- Reviewer run: `RUN-260826-1b682b` (claude-opus-5/high)
- Change Request: `CR-TASK-260826-1o2gkq-2` revision `2`, state `ready`
- Verdict: **ACCEPTED**, with one confirmed Story-level finding routed to the orchestrator (F1).
- Everything below was executed by this run unless a row says otherwise.

## 0. What rev2 actually is, and what changed since rev1

Revision 1 was the `task_delta` proof of the fresh-base merge and was accepted by
`RUN-260826-8e0561`. Revision 2 re-packages the **same, unchanged** commit as the
Story's final Change Request. The tip did not move: `c40f0c2`, tree `f92c9b2`.

One discrepancy is worth naming, because the producer's own evidence document is
wrong about it. `TASK-260826-1o2gkq_story-final-republication.md` predicts rev2
will publish with "an empty repository delta", and argues an empty delta is
*required* because "any non-board product patch would be drift from accepted CR
rev5". That reasoning conflates two different bases. rev1 was scoped to the
element's own commits and was legitimately empty. rev2 is based at the Story's
fork point from trunk (`e70f953` = current `main`), so its delta is the complete
Story delta — 25 paths, 11304 insertions. That is the correct shape for a
story_final revision and it is *better* for the AC "a new story_final Change
Request is independently reviewable": an empty patch would have been
un-reviewable. The published artifact is right; the prediction in the evidence
doc is wrong. No action needed beyond not trusting that paragraph.

`kind` is not readable from a child run — `task-board` exposes no CR read verb and
the CR record is not in `.task-board/` on disk. I therefore report it as
**inferred, not read**: base = the Story branch's fork point from `main`, delta =
the complete Story delta, integration scope `STORY-260825-1r7z9o`. That shape is
story_final's and is not task_delta's (rev1, the task_delta, was empty).

## 1. Change Request integrity — reconstructed, not trusted

The patch resource was verified to reproduce the declared candidate tree from the
declared base, in a scratch extraction outside the reviewed worktree:

```
git archive e70f953 | tar -x -C $T ; git -C $T init ; git -C $T apply <patch>
git -C $T add -A --force ; git -C $T write-tree
  -> f92c9b2c2fd0ca11ac00f7fe4d479a7264f6a698
```

| Claim | Method | Result |
| --- | --- | --- |
| patch sha256 | `shasum -a 256` | `21d5f4b4…c212415e3` — matches declared |
| patch path count | `grep -c '^diff --git'` | 25 — matches declared |
| `git apply --check` against base | exit | 0 |
| base + patch == candidate tree | `write-tree` | `f92c9b2…` exact |
| candidate tree == branch HEAD tree | `git rev-parse HEAD^{tree}` | `f92c9b2…` |
| declared base == fork point | `git merge-base main HEAD` | `e70f953…` |

## 2. Acceptance criteria — every one verified independently

| AC | Method | Result |
| --- | --- | --- |
| current `main` is an ancestor | `git merge-base --is-ancestor main HEAD` | exit 0 |
| the merge is real, not a squash | `git log -1 --format=%P c40f0c2` | two parents: `91adc73` (accepted rev5 tip) + `e70f953` (main) |
| every non-LOGBOOK product blob identical to accepted rev5 | `git diff --name-only 40a83fe f92c9b2 \| grep -v '^\.task-board/'` | **empty** — zero non-board paths differ, `LOGBOOK.md` included |
| rev5 candidate tree is what it claims | `git rev-parse 91adc73^{tree}` | `40a83fe…` |
| LOGBOOK additive vs trunk | `git diff --numstat main HEAD -- LOGBOOK.md` | `168  0` — zero deletions, so main's file is a line-subsequence of HEAD's |
| LOGBOOK preserves the accepted side | `git diff 91adc73 HEAD -- LOGBOOK.md` | empty — byte-identical |
| `integration_base_moved` closed | `git merge-tree --write-tree main HEAD` | prints `f92c9b2…` = HEAD tree exactly, exit 0 → integration is a clean fast-forward |
| worktree clean | `git status --porcelain` | empty |

### 2b. Trunk preservation — attacked separately

Tree-identity with rev5 would *also* pass if the merge had silently taken "ours"
and reverted trunk-only work, so identity alone is not enough. I enumerated every
non-board path `main` changed between the old fork point `b3cb845` and `e70f953`
(18 paths) and compared each against HEAD:

- **12 byte-identical to main** (`SKILL.md`, `canonical_target*.go`, `model_check*.go`, `pi_platform_windows.go`, `pi_run_report.go`, the four root `*_test.go` docs tests, …)
- **4 pure supersets, zero deletions**: `LOGBOOK.md` (+168 −0), `pi_launch_posix.go` (+3 −0), `pi_plan.go` (+22 −0), `pi_test.go` (+88 −0), `main.go` (+103 −0)
- **2 with a single deletion each, both strict extensions of the trunk line**, inspected by hand:
  - `README.md` −1: the `agents-infra` tool-table row is replaced by a row that retains every main-side command and adds `runtime status` / `runtime stop`.
  - `pi_config.go` −1: `rejectUnknownFields(…, "dflash")` becomes `rejectUnknownFields(…, "dflash", "sharing")`. `dflash` survives; the allowlist widens by exactly the new feature field.

`pi_state.go`'s 24 deletions do not appear here because `main` never touched that
file since the fork point — they are rev5 refactors of merge-base code, not trunk
reverts. **Nothing trunk-side was lost.**

## 3. Validation I ran myself

All of it in the scratch reconstruction, whose tree was asserted equal to
`f92c9b2…` before and after every mutation. Go 1.25.5, darwin/arm64.

| Command | Exit | Time |
| --- | ---: | ---: |
| `go build ./...` | 0 | 1.2s |
| `gofmt -l .` | 0 | empty output |
| `go vet ./...` (configured landing gate) | 0 | — |
| focused shared-runtime suite, run 1 / 2 / 3 | 0 / 0 / 0 | 34.2s / 34.1s / 34.3s |
| focused shared-runtime `-race` | 0 | 63.6s |
| `go test .` (root pkg) | 0 | 150.8s |
| `go test ./internal/attachments` | 0 | 3.6s |
| `go test ./internal/infra` run 1 | **1** | 181.3s |
| `go test ./internal/infra` run 2 | 0 | 196.5s |

`go test ./... -count=1` is the configured landing gate; I ran it split into its
three packages to stay inside a bounded call. Nothing was accepted from the
producer's attached logs.

### The one red, and why it is a false red

```
--- FAIL: TestSharedRuntimeLauncherRejectsAuthorizationChannelGuardBypassesAtProductionEntry (15.35s)
    --- FAIL: …/descriptor_three_is_a_socket_rather_than_a_fifo (15.01s)
        pi_shared_launcher_test.go:433: plain valid authorization did not reach the production launcher target
```

That message comes from `requireSharedLauncherValidControl` (`:449`), which polls
for a marker under `sharedLauncherPositiveControlTimeout` (15s wall clock). It is
the **positive control** — the assertion that the harness reaches the production
target at all — not a gate admitting something it must reject. It passes 3/3 in
isolation at 1.34–1.43s and fails only when the whole package runs in parallel on
a loaded machine. This is the same wall-clock flake family that
`RUN-260826-8e0561` recorded (they hit the sibling
`ComparesEveryAuthorizationValueAtProductionEntry/valid`) and that rev-verdicts
`6697f3 §8`, `ae2fa5 §6.4`, `c56491 §6.1`, `6b7916 rec4` already carry.

It is provably not introduced by this element: the tree is byte-identical to
accepted rev5 on every non-board path. This element cannot fix it without
violating its own byte-identity AC.

Charged to the producer, same as last cycle: the evidence doc reports
`go test ./... -count=1` exit 0 from **one** run of a gate this reviewer measured
at 1 fail / 2 pass across three package-level runs. The producer did disclose the
flake honestly ("one fresh green sample, not evidence the historical flake no
longer exists"), which is the right posture — but the table row still reads as an
unqualified 0.

## 4. Gates attacked, not read — F1, a confirmed new finding

The previous chain (`TASK-260826-fcu5pe`, runs `6697f3` / `ae2fa5` / `c56491` /
`6b7916`) attacked the executable-identity gates by narrowing them to **`Ino`
only, `Dev` dropped**, and rev5 answered by adding `*_same_inode_wrong_device`
witnesses. Those witnesses work. **Nobody ever attacked the mirror direction.**

I narrowed the same gates to **`Dev` only, `Ino` dropped** — i.e. an executable
with a *different inode on the same filesystem* is admitted:

| Production site | Gate | Mutant | Suite | Result |
| --- | --- | --- | --- | --- |
| `pi_shared_broker_darwin.go:836` | client exec identity in `attestClient`, called from `handleConnection:718` | drop `\|\| clientExec.Ino != …Ino` | full `./internal/infra` | **SURVIVED**, exit 0, 136.5s |
| `pi_shared_client_darwin.go:341` | peer exec identity | drop `\|\| peerIdentity.Ino != ownIdentity.Ino` | full `./internal/infra` | **SURVIVED**, exit 0, 171.5s |
| `pi_shared_operator_darwin.go:363` | broker-candidate scan | drop `\|\| identity.Ino != ownIdentity.Ino` | (same run as above) | **SURVIVED** |
| `pi_shared_operator_darwin.go:470` | force-stop broker exec | drop `\|\| brokerIdentity.Ino != ownIdentity.Ino` | full `./internal/infra` | KILLED — `TestSharedRuntimeForceStopRefusesForgedBrokerIdentityWithoutSignal` (`pi_shared_integration_test.go:944`) |

The single kill looks **incidental rather than designed**: that test forges a
record pointing at a genuinely different binary, which happens to sit on the same
device, so the inode half is what catches it. There is no witness named for this
class.

Root cause, and it is mechanical:

```
$ grep -rn 'Dev++|Dev +=|\.Dev = '   *_test.go   ->  5 hits
$ grep -rn 'Ino++|Ino +=|\.Ino = '   *_test.go   ->  0 hits
```

Five `Dev`-perturbing witnesses, **zero** `Ino`-perturbing witnesses, against four
production `Dev != … || Ino != …` comparisons. Every one of those gates has
exactly one covered half.

The shipped code is **correct** — this is a negative-evidence hole, not a live
vulnerability. But it is the more dangerous half to leave uncovered: a
cross-device inode collision (the covered case) is exotic, while substituting a
different binary on the same volume (the uncovered case) is the ordinary attack,
and the whole point of these gates is to refuse exactly that.

Second, weaker probe, same shape:
`pi_shared_broker_darwin.go:840`, `hello.ProtocolVersion != SharedRuntimeProtocolVersion`
narrowed to `>` — refuses newer versions, admits every older one, including `0`,
which is the zero value of an omitted field. It survived the focused shared-runtime
suite (exit 0, 34.8s). The witness table only carries `SharedRuntimeProtocolVersion + 1`
(currently 6), so nothing covers the below-current direction. **Focused-suite
evidence only — I did not run this one against the full package**, so treat it as
a strong lead, not a proven full-package survivor.

### Why F1 does not make this `changes_requested`

Fixing F1 means editing rev5 test blobs. This element's AC is *"every non-LOGBOOK
product blob remains identical to accepted CR rev5"*. Routing to `to-dev` would
order a developer to do something that fails the element's own acceptance
criteria — a forced fit, and the element would bounce straight back. The element
delivered precisely what it was asked for, and every one of its ACs is verified
above.

F1 belongs to the Story, not to this leaf. **Recommendation to the orchestrator:**
before integrating `STORY-260825-1r7z9o`, open one leaf authorized to touch rev5
test blobs that adds `*_same_device_wrong_inode` witnesses for
`pi_shared_broker_darwin.go:836`, `pi_shared_client_darwin.go:341`, and
`pi_shared_operator_darwin.go:363` (and names the class explicitly at `:470`
rather than relying on the incidental kill), plus a
`ProtocolVersion = SharedRuntimeProtocolVersion - 1` witness. Or integrate with
the gap consciously recorded. That is the orchestrator's call, not mine.

## 5. Findings

- **F1 — NEW, CONFIRMED.** The `Dev`/`Ino` executable-identity gates are covered on
  one half only. Narrowing three of four production sites to `Dev`-only survives
  the full `./internal/infra` package. Zero `Ino`-perturbing witnesses exist in the
  suite. Shipped code correct; evidence one-sided. Story-scoped, not fixable here.
- **F2 — carried, unchanged.** The 15s wall-clock positive control in
  `requireSharedLauncherValidControl` is load-flaky (1 fail / 2 pass at package
  scope, 3/3 pass in isolation). Needs an event-driven control from a task
  authorized to touch rev5 test blobs.
- **F3 — producer evidence, minor.** The story-final republication doc predicts
  rev2 will have an empty repository delta and argues an empty delta is required.
  Published rev2 is `present`/25 paths, which is correct for a fork-point-based
  story_final and better satisfies the "independently reviewable" AC. That
  paragraph is wrong; the artifact is right.
- **F4 — producer evidence, minor.** `go test ./... -count=1` exit 0 is reported
  from a single run of a gate this reviewer measured as 1-in-3 flaky at package
  scope. Single-sample evidence on a known-flaky gate. Disclosed honestly in prose,
  overstated in the table.
- **rev1 F2 is now CLOSED.** Last cycle the DoD item "Publish a new story_final
  Change Request" was ticked with no such CR in existence. Revision 2 now exists
  with a story-scoped, independently reviewable 25-path delta that this run
  reconstructed from its own patch resource.

## 6. Scope and honesty notes

- I modified nothing in the reviewed worktree. Every mutant ran in a scratch
  extraction under `/tmp`; the reviewed worktree was `git status --porcelain`-clean
  before and after, still at `c40f0c2` / tree `f92c9b2`.
- Scratch tree identity was re-asserted as `f92c9b2…` after each mutant restore.
- `kind = story_final` is **inferred from the published base and scope shape, not
  read.** No child-side read verb for CR records exists.
- The `-race` variant of the full package was not run; only the focused
  shared-runtime `-race` set (exit 0, 63.6s).
- I did not re-run `go test ./...` as a single invocation; I ran its three packages
  separately for bounded-call reasons. Coverage is equivalent; wall-clock
  contention across packages is not.
