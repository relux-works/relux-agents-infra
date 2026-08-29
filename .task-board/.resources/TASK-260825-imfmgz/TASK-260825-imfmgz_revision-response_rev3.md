# Revision 3 response — TASK-260825-imfmgz

Producer run: `RUN-260825-a6390b`
Responding to: review `RUN-260825-ac68af` (revision 2, *changes requested*)
Deliverable: `.research/260825_shared-local-runtime-broker.spec.md`, 1164 → 1665 lines
Probe evidence: `TASK-260825-imfmgz_probe-rev3-results.log` plus four sources

## 0. Verdict on the review

Both blocking findings were real, and neither was repairable inside the
mechanism it broke. F2 asked `flock` a question it cannot answer. F1 wrote the
evidence after the thing the evidence describes. Revision 3 therefore inverts
two orderings rather than adding two checks.

| Finding | Revision 2 | Revision 3 |
| --- | --- | --- |
| F2 | client takes `broker.lock`, hands the descriptor to the broker, broker infers the holder | broker takes the lock **itself**; the client never opens the file |
| F1 | spawn the runtime, then publish its identity | publish the identity, then authorize a process that cannot become the runtime before that |

Both inversions delete code rather than add it. The withdrawn gate
(`broker_lock_not_inherited`), the two-part descriptor check, and the dedicated
waiter-promotion path are all gone.

## 1. F1 — unrecorded live runtime after a broker `SIGKILL`

**Closed by publish-before-run (§6.2 B5–B10, §6.4).**

The broker now creates the runtime process as an *unauthorized launcher* — its
own executable, `agents-infra runtime runtime-launch`, blocked on a read of a
pipe whose write end only the broker holds. It publishes the durable record
naming that pid, with **both** the pre-exec and post-exec identity shapes, and
only then writes the authorization frame. The launcher `execve`s the real runtime
in place.

The two facts this rests on were probed, not assumed:

- **P10**: `execve` preserves pid, pgid and process **start time byte-for-byte**
  (`1787662475.779311` before and after) and replaces only exec path and argv. A
  record written before the runtime existed therefore still identifies it after.
- **P11**: if the broker dies first, the launcher's read returns EOF and it exits
  **without ever exec'ing**. The write end does not leak through `ExtraFiles`, or
  EOF would never arrive — P11.B asserts exactly that.

§6.4 states the result as a closed enumeration over every kill instant, and there
is no row in which a live runtime exists with no record naming it.

**The defect is reproduced before the fix.** P12.A runs revision 2's ordering and
requires the reviewer's forbidden state to appear: runtime alive,
`broker-state.json` absent, `broker.lock` free, and reclamation's verdict on the
absent record is `no-record` while that runtime is still serving. P12.B/C/D then
require the state to be unreachable under the new ordering. Test 12.2.24 is that
same pair at the production entry point.

One consequence worth naming: "absent record ⇒ nothing to reclaim" is now
**derived** rather than assumed, because B5 precedes B7. Revision 2 could not make
that claim, which is precisely what F1 exploited.

## 2. F2 — the lock gate is bypassable during a peer's startup

**Closed by moving the election into the broker (§6.2 B1) and replacing the
ad-hoc gate with a self-inspectable one (§6.2 B2).**

The broker's first action is `flock(broker.lock, LOCK_EX|LOCK_NB)` on a
descriptor it opened in its own process. A loser exits `EXIT_ELECTION_LOST`
before reading, killing, preflighting or binding. There is no descriptor 3, no
inheritance, and nothing to infer.

- **P8.D** replays the reviewer's exact four-condition shape — correct-inode but
  unlocked descriptor 3, a peer holding the lock, the runtime port free, no
  rendezvous socket — and gets `election_lost` with no port taken, no socket
  bound and the incumbent's lock still held. **P8.E** is the discriminating
  control: the same entry point with no incumbent acquires.
- **P8.A**: 8 concurrent candidates ⇒ exactly 1 winner, 7 clean losers.
- **P8.C**: waiter promotion now needs no code at all — the kernel frees the lock
  and the next candidate simply wins.

The ad-hoc-invocation gate is replaced by `getsid(0) == getpid()`
(`broker_not_session_leader`), a direct kernel fact about the calling process.
**P9** shows it distinguishes a `Setsid` child from a shell-launched one and from
a plainly forked one.

§10.3 states plainly what this does *not* do: a deliberate `setsid` invocation
with a genuine recomputed key is admitted, and that is correct — it is the same
binary serving the same profile. Revision 2's attempt to refuse it is the
mechanism that broke.

## 3. F3 — `stop --force` against a live broker with no socket

**Closed by §10.2, in six normative steps.** It is implementable now only because
the owner record exists from B5 rather than from readiness.

The broker is verified against the kernel — pid, non-zombie, uid, start time,
exec identity, exact argv — before anything is signalled, and the runtime is then
reclaimed through the **same** B4 subroutine a broker uses, not a second
implementation of it. `shared_runtime_owner_unidentifiable` covers a held lock
with no record; `broker_stop_identity_mismatch` covers an unverifiable one; an
unreadable read signals nothing.

**P14**: A refuses and leaves the live broker running, B stops and reclaims
without the socket ever being needed, C is the empty-state control.

## 4. F4 — divergent `broker_start_timeout_seconds`

**Closed by strengthening the schema, not by adding a rule.** §4 now requires
`broker_start_timeout_seconds >= runtime.startup_timeout_seconds +
runtime.shutdown_timeout_seconds + 30`. Both terms are inside `profile_digest`,
so every client authorized to share this runtime resolves the same two values,
and the `+ 30` covers composition, reclamation, fork, record writes and poll
granularity.

The consequence is the answer the review asked for: the **smallest admissible
value is identical for every authorized client and is already an upper bound on
a correct startup**. Divergence above it can only mean one client is more
patient, never that a correct peer start is refused. Where the values do differ,
`shared_runtime_peer_start_timeout` and `runtime status` carry both.

## 5. F5 — stale process-boundary and board text

§1 item 2 now states the exact boundary: own session, process group, controlling
terminal and descriptors from birth; OS parentage to the starting client until
that client exits; and why keeping it a real child is load-bearing rather than
incidental.

Board text: `add_checklist_item` is the only checklist mutation the CLI exposes —
there is no way to edit an existing item's text. The stale items are therefore
**superseded by explicit items that name them**, on both siblings, rather than
silently corrected: `lsojra` items 13 and 20, `1lc8o7` items 8 and 10. An
implementer reads a direct instruction not to implement item 12, not an ordering
puzzle.

## 6. Self-found in revision 3

**F6 — `kern.proc.pid` is not a liveness test.** Revision 2 §2 read its EIO on a
dead pid as making liveness decidable. It is decidable only once the corpse has
been *reaped*: an exited, unreaped process answers successfully. This was found
the honest way — it broke P14 in draft, where a `SIGKILL`ed broker kept reporting
as live. **P15** establishes it with `p_stat == SZOMB` as the discriminator and a
live-process control. Every liveness decision in the spec now requires existence
**and** `p_stat != SZOMB` (§1 item 8, §6.2 B4, §7.2 gates 2 and 11, §10.2, §11).
Left unfixed, one zombie would have blocked every future runtime start.

**F7 — nothing bounded a broker whose client never attached.** `lingering` is
entered only on the one-to-zero lease transition, so a broker forked by a client
that then died or timed out would hold the runtime forever. §3.1 adds the
**first-lease grace**: drain after `broker_start_timeout_seconds` from B12. It is
a derivation of an existing field, not a new knob.

**A third, caught during self-review rather than by a probe.** Step 4d originally
had a timed-out client read the owner record and kill the runtime group, which
contradicts §7.4's "a client never signals a runtime it does not own". Corrected:
the client `SIGTERM`s its own broker and lets the broker's drain path reap the
runtime; a broker that must be `SIGKILL`ed leaves a *recorded* orphan, which the
next broker's B4 reclaims. Same path as any other broker death, no special case.

**A fourth, caught the same way.** The first draft of §6.1 capped broker attempts
at 8. That silently breaks late waiter promotion: a client that exhausts its
attempts early cannot start a broker when the incumbent dies late in the window.
The cap is removed; `election_backoff` (2s doubling to 30s) bounds the rate
instead, and resets when `broker-state.json` names a broker pid that is not live.
Test 12.1.4b and two mutants cover it.

## 7. Attack this first, in revision 3

Ranked by how much of the design rests on it:

1. **The §6.4 enumeration is claimed to be closed.** Every reachable kill instant
   is meant to be in one of six rows. If a seventh exists — in particular around
   B8's two-step record rewrite, or between B9's write and the launcher's read —
   the F1 fix is incomplete.
2. **The pre-exec/post-exec pair is the whole of orphan identification.** P10
   proves start time survives `execve` on darwin 25.5.0 arm64. If any managed
   runtime is launched through something that is not a plain `execve` in place —
   a wrapper that forks, a Python entry point that re-execs — the recorded pid
   stops being the runtime's pid and reclamation silently degrades to "stale".
3. **`broker_not_session_leader` proves detachment, not provenance**, and §9 says
   so. Anyone who reads it as an authorization gate is reading more than is
   claimed.
4. **The backoff-reset read is the client's only use of `broker-state.json`.** It
   is argued safe because it only decides *when* the next broker is offered, and
   the B1 election still adjudicates. If any implementation lets that read
   authorize anything, it is a new bypass.
5. **First-lease grace uses `broker_start_timeout_seconds` for a second purpose.**
   Chosen because it is already the bound on the same interval, but it is a
   reuse, and a very patient configuration keeps an unattached broker alive for
   that long.
6. **The launcher's authorization frame is deliberately not a secret.** §9 states
   the reasoning; if same-uid forgery is ever brought into the threat model, that
   decision must be revisited, not patched.
7. **Probes model the mechanisms, not the production code.** Every P8–P15 claim
   is about a syscall-level shape. §12.4 requires the same properties to be
   proved through the real entry points; until `lsojra` does that, the mechanism
   is proved and the wiring is not.

## 8. Evidence

- `go test -count=1 ./...` in `.temp/TASK-260825-imfmgz/probe-rev3/`: P8–P15 all
  pass, 3.16s. Two of them (P12.A, P15) are deliberately negatives that assert a
  defect reproduces.
- Repository suite on this tree, rerun by this producer, not inherited:
  `go build ./...` OK, `go vet ./...` OK,
  `go test -count=1 ./internal/...` — `attachments` 2.952s, `infra` 102.252s,
  `go test -count=1 .` — 73.986s. All three packages pass.
- The repository delta of this revision is the specification file only; no Go
  code in `tools/agents-infra` changed, so the suite result is a regression
  baseline rather than a test of the delta, and is reported as such.
