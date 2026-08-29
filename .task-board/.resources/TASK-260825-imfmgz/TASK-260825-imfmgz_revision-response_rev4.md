# Revision 4 response — TASK-260825-imfmgz

Producer run: `RUN-260825-59c781`
Responding to review: `RUN-260825-969723` (changes requested → `analysis`)
Deliverable: `.research/260825_shared-local-runtime-broker.spec.md`, 1707 → 2412
lines, committed `30e03eb` on `task-board/story/STORY-260825-1r7z9o`.

## Verdict on the review

All four findings were real. Three were blocking and none was repairable inside
the mechanism it broke — each was a process acting on something it could not see,
and each is closed by removing the action rather than sharpening the inspection.

Two of the three fixes then opened defects of their own, which this revision
found and closed as F5 and F7. Both are recorded here rather than quietly
absorbed, because the pattern is now three revisions old: the fix for one
ordering problem creates the next one, and the only thing that catches it is
reproducing the defect before fixing it.

## Closures

### F1 (blocking) — the launcher could not derive its target or its deadline

The review was exactly right, including about why the probes hid it: revision 3's
P10-P12 launcher received `PROBE_TARGET`, `PROBE_TARGET_ARG` and
`PROBE_AUTH_TIMEOUT_MS` through its environment, a capability with no normative
counterpart. That is the "capability claim that does not reproduce" shape.

Closed by **recomposition, not by a new channel**. B9 now spawns
`runtime-launch --runtime-key HEX --profile-project DIR --profile NAME`, and B12
step 2 makes the launcher re-run the same full profile composition the broker ran
and derive `runtime.executable`, the exact `runtime.argv`,
`runtime.startup_timeout_seconds` and the expected `exec_plan_digest` itself,
**before** the frame is read — because the frame's read deadline is one of the
values composition produces. The binding to the broker's plan is `runtime_key`
equality: §5.1 already puts executable, argc, every argv token and both runtime
timeouts inside `profile_digest`, so two processes that compute the same key hold
the same plan by construction.

Fail-closed outcomes are enumerated: missing arguments and unreadable/non-composing
projects ⇒ `runtime_launch_identity_unresolvable`; a divergent key ⇒
`runtime_launch_identity_mismatch`; a truncated frame ⇒ `protocol_violation`; a
foreign pid, wrong version or divergent `exec_plan_digest` ⇒
`runtime_authorization_mismatch`. None execs.

cwd and environment are stated to be **not launcher inputs at all** — the broker
sets both as process attributes at B9 and `execve` in place preserves them.

Putting the plan in the frame would also have worked and is rejected in §14 with
its reason: it makes the pipe a command channel, so whoever holds the write end
chooses *what* runs rather than only *whether*.

Evidence: probe P18.A drives the real shape and reads the resulting exec path and
exact argv back from `kern.procargs2` on the same pid; P18.B/D/E refuse with a
poll proving the pid never carried the target; **P18.C is paired with a mutant**
that treats the failed composition as satisfied and does exec, so the refusal is
a distinction rather than a launcher that never runs anything. Tests 12.2.30,
12.1.5; four mutants; a §12.4 launcher clause that now requires naming the
production call site.

### F2 (blocking) — a timed-out starter could end a broker serving a peer

Real, and it needs no attacker. Closed by **deletion**: §6.1 step 4d now signals
nothing. It selects a refusal code from facts the client owns and reports the
broker's state as `unknown` — never `starting`, which was the inference.

The spec enumerates why nothing is stranded rather than asserting it: every
B5-B13 step is independently bounded and each fatal outcome exits; a broker at
`serving` with no lease is bounded by the first-lease grace; a broker with a
lease is not this client's to end; a stopped one is an operator matter and a
`SIGTERM` would not have reached it anyway. §3.1 promotes the first-lease grace
from tidy-up to the only bound, and §6.3 now states the price of keeping the
broker a real child beside its benefit.

Evidence: probe P19.A reproduces the revocation end to end; P19.B is the fix;
**P19.C is the crux** — A's `wait4(WNOHANG)` returns the identical answer in both
runs, so the proxy cannot discriminate; P19.D is the control that removing the
authority strands nothing. Test 12.2.29 with the narrowing mutant that restores
the live-child inference; a §12.4 clause requiring the signal seam to be
instrumented; acceptance item 8.

A startup-only cancellation capability was considered and rejected in §14: it
must be revoked atomically at exactly the `serving` transition as observed by a
different process, which is the same cross-process inference class that broke
revisions 2 and 3.

### F3 (blocking) — ownership began before identity, and the gap was called impossible

The refusal was correctly fail-closed; the impossibility claim was false, and a
false impossibility claim is worse than a named limitation because it stops
anyone handling the case.

Closed in three parts. **Narrow it**: §6.0 renumbers the broker steps — the
detachment gate is hoisted to B1, B3 reads the predecessor record, and B4
publishes the elected-owner record as the broker's first side effect. The window
is now one `renameat` with no blocking call in it, instead of spanning full
composition, reclamation and preflight. **Name it**: `starting-unverified` is a
state in §3.1, in `status`'s state list, and in §8. **Bound it**: §10.2 replaces
the impossibility claim with a re-poll for either resolution — the record
appearing or the lock freeing — and only then refuses, printing the diagnosis and
the candidate holders with their `p_stat`.

It still refuses to signal, and §9 says why without hedging: probe P16.D produces
two candidates identical on uid, binary inode, argv and `p_stat` while exactly
one holds the lock, and `flock` cannot name its holder on darwin. A set is all
there is; signalling one member of a set is guessing. An operator with `lsof` can
do better and is told so.

Evidence: P16.A produces the exact state and shows `SIGCONT` resolving it with
the lock still held; P16.B is the running-owner control that makes `SSTOP`
discriminate; P16.C shows the candidate filter excludes a foreign holder; P16.D
is why the refusal stays. Tests 12.2.31 (with a post-B4 `SIGSTOP` control so the
limitation is scoped to the window, not to the operator surface) and 12.2.32
(pauses after B1-B6); three mutants; a §12.4 `status` clause.

### F4 — test 12.2.22 specified the withdrawn revision-2 path

Split as asked. **12.2.22a**: a second broker against a serving incumbent exits
`EXIT_ELECTION_LOST` at B2 and the test asserts it reached neither preflight nor
bind, with the incumbent's socket **inode** byte-identical afterwards.
**12.2.22b**: a deterministic contender between B6's stale-inode cleanup and
B15's bind produces `broker_rendezvous_bind_conflict`, asserted on the
contender's `dev`/`ino` rather than on the path existing — an unlink-and-rebind
would satisfy the weaker assertion. P20.C is the free-path control. Probes
P20.A/B/C; two mutants, one of which requires the weaker assertion to stop
reddening.

## Self-found

### F5 — the F3 fix reintroduced the second review's F1

`broker-state.json` is one file. Moving the owner record forward to B4 means the
successor overwrites the predecessor's record **before** B6 reclaims the runtime
that record names, so a `SIGKILL` between them leaves a live runtime nothing
names. Closed by carrying the predecessor's block forward verbatim as
`inherited-unreclaimed`, and by making a malformed read fatal at B3 — before B4
writes — so a failed read can never destroy a predecessor's block either.

Found by writing the probe before believing the fix. P17.A reproduces the loss,
P17.B reclaims at the same kill instant, P17.C is the control that reclamation
does not kill whatever it finds, P17.D covers the failed read. §6.4 gains the
B4-B6 row; test 12.2.24a; two mutants.

### F6 — the detachment gate ran after the election

Harmless in revision 3 only because the election had no side effect either. Once
B3 reads and B4 writes, a broker that won the election and then failed the gate
would have overwritten a predecessor's record on its way out. Hoisted to B1; it
is a pure `getsid` call with nothing to lose by running first.

### F7 — "removes its own record and exits" stopped being safe

The second door the F3 fix opened. Every fatal outcome said the broker removes
its own record, which was safe only while that record could never name a runtime
the broker had not created. Since B4 it can: a `--runtime-key` mismatch at B5 or
an occupied port at B8 would delete a record still naming a live predecessor.

Closed by a normative **fatal-exit discipline** table in §6.2 with one rule
underneath it: *the record may only stop naming a process once that process is
proven gone.* Leave it intact while a block is unreclaimed; after B9 reap first
and remove second; and never remove it when the reap failed. §7.5's normal drain
already obeyed the ordering. §8 gains two rows; test 12.2.33 with a
no-inherited-block control so the rule discriminates; two mutants.

## Evidence

- Probes **P16-P20** on darwin 25.5.0 arm64, go1.25.5, `golang.org/x/sys v0.30.0`.
  All pass. Source and raw output attached; source also at
  `.temp/TASK-260825-imfmgz/probe-rev4/`.
- **Deliberate negatives**: P17.A, P18.C and P19.A each reproduce the defect
  before the fix. **Discriminating controls**: P16.B, P16.C, P17.C, P18.C's
  fallback mutant, P18.A, P19.D, P20.C — so no gate here is proved by always
  refusing.
- Revision-3 probes P8-P15 rerun by this producer: pass, 3.580s.
- Repository suite rerun by this producer, not inherited: `go build` OK,
  `go vet` OK, `go test -count=1 ./internal/...` (attachments 0.993s, infra
  88.472s) and `go test -count=1 .` (57.198s) all pass. The repository delta is
  the spec and the logbook, so this is a regression baseline rather than a test
  of the delta.
- Board: no element created, removed or re-scoped. All four spec payloads
  verified byte-identical at SHA-256 `5109a92261712c89...8453`.

## Attack revision 4 first, ranked

1. **§6.4 is again claimed closed**, now over fifteen steps. The two places to
   look are named in that section: between B10's two-step rewrite and B11's
   write, and between B4's `renameat` and B6's first `kern.proc.pid`.
2. **`starting-unverified` is a named limitation, not a closure.** "One atomic
   write with no blocking call in it" is a claim about an implementation that
   does not exist yet. An allocation, a log write or a path revalidation inserted
   between B2 and B4 widens the window silently.
3. **Composition now runs twice per runtime** — broker at B5, launcher at B12 —
   reconciled only by `runtime_key` equality. Sound exactly as long as §5.1's
   digest covers every value the launcher acts on. A field added to `runtime.*`
   without being added to the digest breaks it.
4. **`exec_plan_digest` is claimed only as a consistency check** (§9). Any later
   text that treats it as authorization is a defect.
5. **The fatal-exit discipline is a table**, and tables drift from code. A new
   fatal outcome that lands in none of its five rows is an unreclaimed runtime
   waiting to happen.
6. **The client now has no authority over the broker it forked at all.** The
   bound rests entirely on the first-lease grace plus every B5-B13 step being
   independently bounded. A step added there without a bound reopens it.
7. **B4 publishes `runtime_key_claimed` before B5 verifies it.** Argued safe
   because it authorizes nothing and lives under a key-derived path; any consumer
   reading it as verified is a defect.
8. **The probes are syscall-layer models in a separate module.** §12.4 is what
   makes them production evidence, and `TASK-260825-lsojra` owes it — which is
   exactly the gap F1 caught in revision 3.
