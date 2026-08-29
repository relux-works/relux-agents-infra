# Revision 2 response — TASK-260825-imfmgz

Producer run: `RUN-260825-f23f2d`
Responding to: review `RUN-260825-626994` (changes requested)
Deliverable: `.research/260825_shared-local-runtime-broker.spec.md`, revision 2
(771 → 1164 lines), committed `049fb20` on `task-board/story/STORY-260825-1r7z9o`

The review was correct on every count, including both blocking findings. This
document records what changed and, more usefully, what a re-reviewer should
attack in revision 2.

## 1. Findings closed

The full map is section 16 of the spec itself. Summary:

| # | Change | Probe |
| --- | --- | --- |
| F1 | §6 step 5's blocking `LOCK_EX` replaced by a single bounded loop racing a connect poll against `LOCK_EX\|LOCK_NB`; `shared_runtime_peer_start_timeout` added; §3.1 declares `ENOENT` + `EWOULDBLOCK` the normal observation during a peer's startup | P7 |
| F2 | Gate split: `fstat` descriptor 3 for `broker.lock` inode equality, **plus** a second-descriptor `LOCK_EX\|LOCK_NB`; §9 and §10 restate what it proves | P1, P2 |
| F3 | New §5.4: starter's policy governs, announced in `hello_ok`, reported by `status` and in the lease-limit refusal | — |
| F4 | New gate 3b on the broker's start-time-recorded binary identity; §8 split into two upgrade rows; protocol-version bump made normative | P5 |
| F5 | `--profile-source PATH` → `--profile-project DIR`, full composition re-run once at startup | — |
| F6 | Mid-handshake EOF folded into the bounded drain retry, `protocol_violation` still non-retryable | — |
| F7 | 190 → 189 | — |
| F8 | *Self-found.* "Reparented to init within milliseconds" was false; corrected, and the corrected relationship is used for `broker_start_failed` | P6 |

## 2. Probes

`.temp/TASK-260825-imfmgz/probe-rev2/`, output attached as
`TASK-260825-imfmgz_probe-rev2-results.log`. darwin 25.5.0 arm64, go1.25.5,
`golang.org/x/sys v0.30.0`. P1–P7 pass.

The review's core process criticism was that revision 1 specified two mechanisms
that were never probed — `flock` self-inspection and the startup race — while
loudly probing the ones that happened to work. Revision 2 probes every mechanism
it replaces them with, and P7 is deliberately shaped as a **negative**: it runs
revision 1's algorithm and requires it to fail, then runs the revised one and
requires it to succeed. A fix that cannot demonstrate the bug it fixes is not
evidence.

Two facts fell out of the probes that are now in the spec:

- On darwin, `close(2)` on a descriptor another thread is blocked in
  `flock(LOCK_EX)` on **itself blocks**. A client that abandons a blocking lock
  acquisition cannot even release its descriptor. Independent second reason not
  to block on the lock. (Found by a 25s test timeout whose stack showed the
  close parked in `syscall.Close`.)
- `t.TempDir()` overflowed `sun_path` while writing P7, which is the same
  103-byte limit §5.2 documents — observed for real rather than derived.

## 3. What a re-reviewer should attack in revision 2

Ranked by how much would break if the assumption is wrong.

1. **The wait loop's deadline branch is claimed to have exactly two outcomes**
   (§6 step 4d). The argument is that (b) runs every iteration and resolves to
   "acquired" or `EWOULDBLOCK`, so a client is always either the starter or has
   seen a peer. If there is a third reachable state, the spec has a dead branch
   or an unspecified one.
2. **Gate 3b is an announced value.** §7.2 argues it cannot admit anything gate 3
   would reject, because both must pass and gate 3 is kernel-sourced. Attack that
   composition — it is the only place revision 2 accepts self-reported evidence.
3. **The `broker_lock_not_inherited` gate still cannot prove the holder is
   *this* process.** §9 says so explicitly and §10 leans on the port preflight
   and the bind conflict instead. Attack whether that composition really closes
   the ad-hoc case, or whether the honest conclusion is to drop the gate.
4. **The starter keeps its broker as a real child** so step 4c can reap it. This
   leaves at most one unreaped broker child per client after attach, cleaned up
   only when the client exits. Bounded, but attack whether it is bounded in every
   path.
5. **Single timeout knob for both roles.** §4 and §6 argue
   `broker_start_timeout_seconds >= runtime.startup_timeout_seconds` makes one
   knob correct for a waiter. Attack the case where a peer's runtime startup
   legitimately exceeds the waiter's own configured value — the two clients may
   have different sharing tables (§5.4), and `broker_start_timeout_seconds` is
   read from the *caller's* config, not the effective policy.
6. **`--profile-project DIR` re-runs composition in the broker.** Attack whether
   the broker can reproduce the client's digest in every composition shape the
   existing resolver supports, given the self-verification is fatal.

Item 5 is the one I am least sure of and did not resolve: the waiter's deadline
comes from its own config while the runtime startup it is waiting on was
launched under the starter's. I chose the caller's value deliberately — a client
must be able to bound its own wait without first connecting to learn the
effective policy, and connecting is exactly what it cannot do yet. The residual
is that a waiter with a short timeout gives up on a peer's legitimately slow
start and reports `shared_runtime_peer_start_timeout`, which is at least the
correct, non-destructive outcome: it starts nothing and names the live broker.
Recorded here rather than smoothed over.

## 4. Board

No board element was created, removed, or re-scoped. The decomposition the review
verified (`imfmgz → lsojra → 1lc8o7`) is unchanged and still correct: the revision
is confined to the existing spec deliverable.

Three consequences were propagated, because leaving them would hand a developer
the revision the review rejected:

- Both siblings' **precondition copies** of the spec were updated to revision 2
  and verified byte-identical to the committed file. They are separate stored
  payloads, not links; updating the producer's copy alone would have left
  revision 1 in front of the implementer.
- `TASK-260825-lsojra` gained three checklist items: gate 3b (its item 7 says
  "12-gate", now 13), §5.4 effective sharing policy (previously uncovered by any
  item), and the §6 wait loop with the deterministic-window requirement for test
  12.1.2.
- `TASK-260825-1lc8o7` gained two: acceptance items 5 (the second spawn crossed
  the startup window) and 8 (effective linger read from `runtime status`), both
  new in revision 2 and neither covered by an existing item.

Each traces to a named spec section. No element was added for symmetry or
process, and nothing beyond these was invented.
