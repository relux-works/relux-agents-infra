# Revision 3 attack evidence — TASK-260825-imfmgz

Reviewer run: `RUN-260825-969723`
Candidate: `CR-TASK-260825-imfmgz-3` revision 3

## A1 — launcher input closure fails

Normative producer/consumer comparison:

| Required by `runtime-launch` | Supplied by B7 argv | Supplied by B9 frame | Other normative source |
| --- | --- | --- | --- |
| `runtime.startup_timeout_seconds` | no | no | none |
| `runtime.executable` | no | no | none |
| exact `runtime.argv` | no | no | none |

B7 supplies only `--runtime-key`. B9 supplies schema, protocol version,
`runtime_key`, and `launcher_pid`. B10 nevertheless requires the missing timeout
and target values. A hash does not recover its preimage, so `runtime_key` cannot
be used to derive them.

The attached producer probes do not reproduce this protocol. Their launcher
gets `PROBE_TARGET`, `PROBE_TARGET_ARG`, and `PROBE_AUTH_TIMEOUT_MS` through its
environment. The normative B7/B9 contract specifies no equivalent transport or
binding to the B8 durable record.

Negative shape: **check present but uncalled from production / capability claim
that does not reproduce**. The publish-before-run mechanism works in the probe,
but the production entry point is not implementable from its declared inputs.

## A2 — client deadline can terminate a serving broker with a peer lease

Valid ordering under sections 6.1 and 6.2:

1. Client A starts broker child X.
2. A's step 4a `connect()` returns `ENOENT` immediately before its deadline.
3. X completes B12/B13 and becomes connectable.
4. Independent client B connects and obtains a lease.
5. A executes step 4d. `broker_child != none` is true because X remains A's
   child for its whole lifetime, not only while starting.
6. A infers X is still starting and sends it `SIGTERM`; X drains and revokes B.

No forbidden OS event or same-uid attack is needed. The proxy signal
`broker_child != none` cannot distinguish `starting` from `serving`. This path
terminates the runtime with a live lease even though there was no operator stop.

Negative shape: **bypass path around the final-release gate / prove or report
nothing**. A local acquisition deadline must not become shutdown authority over
a broker that may already serve another RUN.

## A3 — valid owner is called unidentifiable

B1 acquires `broker.lock` and declares the broker the runtime owner. B5 publishes
the owner record only after B2, full profile composition, and predecessor read /
reclamation. Therefore a valid broker paused anywhere in B1-B4 holds the lock
with no record.

The unreachable `stop --force` path says that `EWOULDBLOCK` plus no record is a
state no broker of this protocol can produce, refuses
`shared_runtime_owner_unidentifiable`, and signals nothing. A deterministic
`SIGSTOP` after B1 defeats both operator status and operator stop indefinitely.
The safe refusal is preferable to signalling an arbitrary pid, but it does not
satisfy the promised operator lifecycle and its impossibility claim is false.

Negative shape: **absence treated as proof / capability claim that does not
reproduce**. The design needs kernel-verifiable owner identity throughout the
ownership interval, or an explicit bounded recovery contract that does not call
the reachable state impossible.

## A4 — revision-2 test survived the revision-3 ownership rewrite

Section 12.2 test 22 starts a second broker with an inherited held lock and
expects it to reach `runtime_listener_occupied`, then
`broker_rendezvous_bind_conflict`. Revision 3 states that no descriptor is
inherited and B1 loses the election before preflight or bind. The test conflicts
with the normative production path and cannot prove the B13 no-unlink rule.

Replace it with two separate production shapes: the serving-incumbent case must
exit `EXIT_ELECTION_LOST`; a deterministic contender inserted after stale-inode
cleanup but before B13 must trigger `broker_rendezvous_bind_conflict` without
unlinking the contender.

