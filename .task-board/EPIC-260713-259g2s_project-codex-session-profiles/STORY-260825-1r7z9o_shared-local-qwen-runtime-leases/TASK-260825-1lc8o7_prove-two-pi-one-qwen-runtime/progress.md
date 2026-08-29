## Status
closed

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- TASK-260825-lsojra

## Blocks
- (none)

## Checklist
- [ ] Document orchestrator-driven shared Qwen runtime workflow
- [ ] Automated acceptance harness uses two independent tracked spawns rather than one shell
- [ ] Real evidence proves two RUN IDs, two Pi processes, one Qwen runtime, and overlap
- [ ] Final lease release proves bounded cleanup with no listener or owned process left
- [ ] Opt exactly one deployed profile into shared mode with linger_seconds=0 for the proof and record the config delta; migrating other project configs stays out of scope per spec section 15
- [ ] Record spec section 13 item 5: the second tracked spawn crossed the broker startup window. Its start stamp precedes the runtime /v1/models first-ready stamp in broker.log, and its own log shows at least one EWOULDBLOCK wait-loop iteration before it attached. A run that misses the window does not prove reuse and must be repeated
- [ ] Record spec section 13 item 8: sharing.effective.linger_seconds = 0 read from agents-infra runtime status --json inside the overlap window, with the broker pid that fixed it. Without it, a teardown after the final release is equally consistent with a peer-started broker and an unrelated runtime death
- [ ] SUPERSEDES item 6. Spec revision 3 window evidence is not an EWOULDBLOCK observation - the client never opens broker.lock. Record instead: the second clients own log shows at least one broker child that exited EXIT_ELECTION_LOST before it attached, and its start stamp precedes the runtime /v1/models first-ready stamp in broker.log (section 13 item 5)
- [ ] Record spec section 13 item 4 publish-before-run evidence: the attested runtime pid start time must be byte-equal to the runtime.start_time the broker published in broker-state.json BEFORE that process became mlx_lm.server. That equality is what proves the record preceded the runtime rather than followed it
- [ ] Section 13 has EIGHT acceptance items in revisions 2 and 3, not seven. Any note or harness that enumerates seven is stale and must cover item 8 bounded cleanup attributed to the effective linger
- [ ] SUPERSEDES item 10. Spec section 13 has NINE acceptance items in revision 4, not eight. A new item 8 was inserted and the bounded-cleanup item is now item 9, so item 7 above should read section 13 item 9
- [ ] Record spec section 13 item 8, new in revision 4: neither run client signalled the broker. Both client logs must show no signal sent to any broker pid, and the status snapshot must show the broker pid unchanged from the one that first appeared in broker-state.json. If either spawn hit its broker_start_timeout, its refusal must report broker.state = unknown and the other run lease must be intact in the next snapshot. Under revision 3 the second spawn timeout would have terminated the first spawn broker
- [ ] CORRECTS item 9 step label. The publish-before-run evidence is section 6.2 B10 in revision 4 numbering, not B8. Item 9 must additionally record that runtime.post_exec.argv and runtime.exec_plan_digest match what the LAUNCHER independently composed, read from runtime.log - the broker never tells the launcher what to run, so the run does not prove item 4 unless both processes reached the same plan from the same project directory

## Notes
Acceptance is defined item-by-item in spec section 13 of the linked precondition resource. All seven items are required: two distinct RUN handles, distinct client pgids with no shared parent shell, two Pi pids with distinct session dirs, exactly one attested mlx_lm.server pid whose kernel start time predates the second client, an agents-infra runtime status --json snapshot taken inside the overlap showing lease_count=2 with both RUN ids, independent successful outputs, and bounded cleanup on all six surfaces after both terminal states. A single-shell pi-and-pi run is explicitly not acceptance. Checklist 5 covers the one-profile shared-mode opt-in; migrating other configs is out of scope per spec section 15.
Closed as superseded by the user-confirmed scope on 2026-08-26. Local Qwen/Pi is not yet a task-board runtime, so a two-tracked-RUN proof would falsely require the deferred board adapter. The current acceptance proof lives in TASK-260826-3i0lwe and uses two independent standalone qwen-infra/Pi spawn processes sharing the broker runtime. A task-board-specific two-RUN proof must be created later with the adapter integration.

## Precondition Resources
- [TASK-260825-1lc8o7_shared-runtime-broker.spec.md](file://TASK-260825-1lc8o7/TASK-260825-1lc8o7_shared-runtime-broker.spec.md) — Precondition: shared local runtime broker specification, revision 9 (byte-identical to the producer copy)

## Outcome Resources
(none)

## Created
2026-08-25T11:33:30Z

## Last Update
2026-08-25T22:33:29Z
