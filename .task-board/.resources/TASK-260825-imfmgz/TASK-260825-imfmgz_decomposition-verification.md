# Decomposition and gap verification — STORY-260825-1r7z9o

Task: `TASK-260825-imfmgz` (solution-architect)
Date: 2026-08-25

## 1. Board shape: unchanged, and that is the finding

The story already carries the smallest decomposition that maps its scope:

| Element | Deliverable | Story requirement it implements |
| --- | --- | --- |
| `TASK-260825-imfmgz` | The specification | "Broker lifetime, lease recovery, final-release cleanup, status, explicit stop, and negative identity/concurrency tests are documented" |
| `TASK-260825-lsojra` | The implementation | "Separate task-board/agent orchestrator spawn-runners in distinct process groups must acquire leases automatically … reuse one broker-owned MLX runtime" |
| `TASK-260825-1lc8o7` | Docs plus the real acceptance proof | "An orchestrator can issue two independent tracked agent spawns … both overlap while using exactly one verified Qwen model-server PID" |

Dependencies are already linked as `imfmgz -> lsojra -> 1lc8o7`, and each phase
has exactly one deliverable. **No new story, task, or research element was
created.** Every clause of the story scope maps onto an existing element:

| Story scope clause | Covered by |
| --- | --- |
| terminal-independent broker | `lsojra` checklist 1 → spec §3, §6 |
| leases acquired automatically on the normal spawn path | `lsojra` checklist 2 → spec §5.3, §6, §7 |
| distinct process groups, independent spawn-runners | `lsojra` checklist 3 → spec §5.3, §6; proved by `1lc8o7` §13.2 |
| reuse one verified loopback model server | `lsojra` checklist 1, 7 → spec §7.2 |
| Pi sessions/state isolated | `lsojra` checklist 3 → spec §5.3 |
| release leases on tracked child termination | `lsojra` checklist 2, 4 → spec §7.4 |
| reject arbitrary or identity-mismatched listeners | `lsojra` checklist 7 → spec §7.2, §11, §12.2 |
| stop/reap only after the final lease or an operator stop | `lsojra` checklist 4 → spec §7.5, §10 |
| documentation and the two-spawn proof | `1lc8o7` → spec §13 |

Four checklist items were added to `lsojra` and one to `1lc8o7` rather than
creating new elements. Per the role rules, a task-local gate belongs in that
task's checklist; splitting the config schema, the attestation chain, the CLI
surface, or the mutant discipline into separate board elements would have been
symmetry, not scope.

## 2. Justified gap — the only addition beyond the literal story text

**Missing piece.** The story requires that same-profile spawns share a runtime
but never states how a profile becomes shareable. Without an explicit switch the
implementation must choose between two silent behaviors, and one of them changes
every existing deployment.

**Requirement left incomplete.** Story scope: "Separate task-board/agent
orchestrator spawn-runners … must acquire leases automatically when launching
the same managed Qwen profile." "The same managed Qwen profile" has no
shareability attribute today, so "automatically" is undefined for the 121
project configurations already in the fleet (`LOGBOOK.md`, 2026-08-24 2109).

**Consequence of leaving it open.** Defaulting to shared would silently convert
every existing managed profile from "one exclusive owner, refuse any occupied
listener" to "attach to a broker", changing the meaning of configurations no one
reviewed for sharing. Defaulting to exclusive with no switch would make the
story's own acceptance proof unreachable.

**How the addition closes it.** Spec §4 adds
`[agents.pi.profiles."NAME".runtime.sharing]` with `mode` defaulting to
`exclusive` when the table is absent, every field required when it is present,
and unknown fields failing closed — matching the existing profile schema
discipline. Existing configs keep today's behavior byte-for-byte; the acceptance
profile opts in explicitly (`1lc8o7` checklist 5).

**Self-verification performed before the addition.** Sections read:
`STORY-260825-1r7z9o` description, scope, and AC; all three child task scopes
and ACs; `.research/260817_pi-local-model-launch-contract.md` §4 and §4.1 (the
TOML contract and its "unknown fields fail closed" rule), §7 (process ownership
and lifecycle), §9 (security and threat boundary, including its explicit
out-of-scope list), §10 (failure semantics), and §11 (rejected alternatives).
Result: the cycle-8 contract neither answers shareability nor excludes it. Its
out-of-scope list covers a malicious runtime, runtime self-attestation, the
post-preflight bind race, same-uid mutation, model acquisition/conversion/
benchmarking, and DFlash attestation — none of which is this. The addition is
one field group inside an existing, already-governed schema, not new scope.

**Additions considered and rejected as invented scope.** Socket-ownership
attestation via `proc_pidfdinfo` was rejected: cycle-8 §9 already places the
post-preflight bind race out of the threat model, so it would close an
already-accepted residual risk at the cost of a cgo or raw-syscall build
dependency. It is recorded in spec §9 and §14 as future hardening with a
reserved refusal code, and no board element was created for it. A research task
for it was also rejected: the spec decides the baseline, so no implementation
decision is waiting on an answer.

## 3. Research tasks

None created. Every question the specification had to answer was either resolved
from the existing contract and source, or settled empirically by the probe in
`TASK-260825-imfmgz_platform-probe.log` — darwin process identity through
`sysctl`, `LOCAL_PEERCRED`/`LOCAL_PEERPID`, the `sun_path` limit, and `flock`
handoff across `fork`/`exec`. Per the role rules, a research task is only
justified for a question the spec genuinely leaves open; none remained.

## 4. Diagrams

None produced. The state machine and lease lifecycle are two small graphs that
are clearer inline in spec §3 than as separate rendered artifacts, and the
process/IPC topology is fully described by the path and message tables in §5 and
§7.3. A rendered diagram would have added an artifact without adding clarity.

---

## 5. Revision 2 addendum (`RUN-260825-f23f2d`)

Sections 1–4 above were verified against the board by review `RUN-260825-626994`
and remain accurate. Revision 2 changed the spec, not the decomposition.

**No board element was created, removed, or re-scoped.** `imfmgz → lsojra →
1lc8o7` is unchanged. The review's own conclusion was that the rework is
specification revision, not board work, and nothing in closing F1–F8 introduced
a deliverable that an existing element does not already own.

**Five checklist items were added**, each because revision 2 introduced a
normative requirement that no existing item covered. These are not new scope;
they are traceability for scope that already sits inside `lsojra` and `1lc8o7`,
and without them a developer would implement the revision the review rejected.

| Element | Item | Spec requirement it traces to |
| --- | --- | --- |
| `lsojra` | gate 3b, correcting item 7's "12-gate" to 13 | §7.2 gate 3b, §8 in-place-upgrade row |
| `lsojra` | effective sharing policy | §5.4 (new section) |
| `lsojra` | §6 wait loop and the deterministic window in test 12.1.2 | §6 step 4, §12.1.2 |
| `1lc8o7` | second spawn crossed the startup window | §13 item 5 (new) |
| `1lc8o7` | effective linger read from `runtime status` | §13 item 8 (revised) |

**Precondition resources were refreshed.** Both siblings store their own copy of
the spec rather than a link, so the producer-side update did not reach them.
Both were re-uploaded and verified byte-identical to the committed revision 2.

**Nothing was added beyond this.** The candidates considered and rejected:

- A separate task for the `broker.lock` gate rework. Rejected: it is one gate
  inside `lsojra` checklist item 1, and splitting it would create a board element
  smaller than its own coordination cost.
- A research task for the §5.4 policy-divergence question. Rejected under the
  role rule that research is only for questions the spec leaves open: the review
  identified the gap and the spec now answers it (starter's policy governs), so
  no implementation decision is waiting on an answer.
- A task for socket-ownership attestation, reconsidered because gate 3's limits
  are now stated explicitly in §9. Rejected again for the same reason as in
  section 2 above: it is in the §9 out-of-threat-model list and §14's rejected
  alternatives, and revision 2 narrowed rather than widened that residual.
- Re-opening the `runtime_listener_attestation_unavailable` reservation. Rejected:
  it remains reserved and unimplemented, and revision 2 adds no code or
  diagnostic that claims listener ownership.

**Diagrams:** still none. Revision 2 added no topology; §6's algorithm is a
linear loop that reads more precisely as numbered steps than as a flowchart.

---

## Revision 4 addendum (`RUN-260825-59c781`, closing review `RUN-260825-969723`)

### Board delta: none

No board element was created, removed, re-scoped, or re-parented. The three-leaf
chain stands unchanged:

```
STORY-260825-1r7z9o
  └── TASK-260825-imfmgz  (spec)          blocks →  TASK-260825-lsojra
      └── TASK-260825-lsojra  (implement) blocks →  TASK-260825-1lc8o7
          └── TASK-260825-1lc8o7  (prove)
```

Revision 4 closed three blocking findings and three self-found defects entirely
inside the existing specification. Every one of them is a change to how an
existing deliverable behaves — the launcher's inputs, the client's deadline
branch, the broker's step order and its record — and none of them adds a
deliverable with its own acceptance criteria. Splitting any of them into a board
element would have created scope that no requirement asks for and that no
developer could pick up independently of `TASK-260825-lsojra`, whose scope
already reads "implement the shared runtime broker per the specification".

### Additions rejected on self-verification

Four were considered and refused before creation. Each was checked against the
story scope, the story AC, and section 15's out-of-scope list.

| Proposed | Verdict | Why |
| --- | --- | --- |
| A research task: "identify the `flock` holder pid on darwin" | **Rejected** | Not an open question. The revision-1 reviewer probe already established `F_GETLK` cannot see a BSD `flock`, and probe P16.D establishes that process enumeration yields a set rather than a holder. A research task here would re-derive a settled negative result |
| A task: "add `proc_pidfdinfo` socket-ownership attestation so the lock holder is provable" | **Rejected** | Section 14 already rejects socket-ownership attestation for the endpoint, and it does not answer the lock-holder question anyway. It would also require cgo, changing build constraints for a residual risk the story does not raise |
| A separate task for the `runtime-launch` recomposition | **Rejected** | It is one entry point of the binary `TASK-260825-lsojra` already owns, and it cannot be tested without the broker that authorizes it. Checklist item 25 carries it with a spec-section citation |
| A separate task for the operator `starting-unverified` surface | **Rejected** | `TASK-260825-lsojra` checklist item 8 already owns `agents-infra runtime status\|stop` per spec section 10. Revision 4 changes what those surfaces report, not who builds them; checklist item 27 carries the delta |

### Justified gap: none created

Revision 4 added no element beyond the literal spec, so no `Justified gap` record
was required. The three configuration/record fields it does add —
`stage`, `runtime_key_claimed`, `runtime.exec_plan_digest` — are inside the
specification's own record schema (section 6.2 B4 and B10), not new board scope,
and each closes a named review finding rather than anticipating a need.

### Traceability of the twelve checklist items added

Every item added in this revision cites a named spec section and, where it exists,
the review finding it closes and the probe that establishes it.

| Element | Item | Cites | Closes |
| --- | --- | --- | --- |
| `lsojra` | 21 | §6.0 renumbering table | reading gate for items 14-19 |
| `lsojra` | 22 | §6.2 B1/B2 order | self-found F6 |
| `lsojra` | 23 | §6.2 B3/B4, probe P17.A | review F3 + self-found F5 |
| `lsojra` | 24 | §6.2 fatal-exit discipline, test 12.2.33 | self-found F7 |
| `lsojra` | 25 | §6.2 B9/B12, probes P18.A-E | review F1 (blocking) |
| `lsojra` | 26 | §6.1 step 4d, §12.4, probe P19.C | review F2 (blocking) |
| `lsojra` | 27 | §10.1, §10.2, probe P16.D | review F3 (blocking) |
| `lsojra` | 28 | §9 protocol version bump | consequence of F1 + F3 |
| `lsojra` | 29 | §3.1 first-lease grace, probe P19.D | review F2 |
| `1lc8o7` | 11 | §13 item count (nine) | review F2 consequence |
| `1lc8o7` | 12 | §13 item 8 | review F2 (blocking) |
| `1lc8o7` | 13 | §13 item 4, §6.2 B10/B12 | review F1 + renumbering |

### Sibling precondition payloads

Both sibling copies are separate payloads rather than links, so updating only the
producer's copy would leave revision 3 in front of the implementer. Both were
refreshed and verified byte-identical to the committed file and to the task
outcome copy at SHA-256 `5109a92261712c89d717a5aeeb463a6c4f731e6f1639d415e475f1ab736a8453`.

---

## Revision 5 addendum — review `RUN-260825-668303`

### Board delta: none

No board element was created, removed, re-scoped or re-parented. The three-leaf
chain — specification → implementation → real two-spawn proof — is unchanged and
remains the smallest decomposition that maps the story's scope.

**Why no element was added.** The review's finding was that a normative gate in
an already-approved section had no evidence behind it. That is rework inside the
existing specification leaf, not new system scope: the gate was already specified,
already traced to the story's "reject arbitrary or identity-mismatched listeners"
clause, and already owned by `TASK-260825-lsojra` to implement. Creating a
"harden the authorization frame" task would have split one deliverable across two
elements and given the implementer two places to look for the same contract.

**Justified-gap check performed before deciding.** The candidate addition
considered and rejected was a separate research task on *whether the authorization
frame should carry a nonce or a secret* — the natural adjacent question once
forgery is discussed. Sections checked: §9 threat boundary (same-uid forgery is
explicitly outside the model, stated twice), §6.2 B12 ("the frame is not a secret
and is not claimed to be one"), §14 rejected alternatives (the plan-in-frame
variant is already rejected there with reasons), and §15 scope boundaries. The
specification **already answers** the question and states the answer as a
deliberate design position, so a research task would re-litigate a resolved
decision. Rejected; not created.

**Second candidate, also rejected.** An acceptance-side item on
`TASK-260825-1lc8o7` recording that the two clients and the broker agreed on
`protocol_version = 5`. Sections checked: §13 (nine acceptance items, unchanged by
revision 5) and §7.2 gate 5, which already requires that agreement for any lease
to exist at all — so the two-spawn proof cannot succeed without it. An item
restating a precondition of its own success is ceremony, not traceability.
Rejected; `1lc8o7` receives no revision-5 item.

### Traceability of the six checklist items added

| Element | Item | Cites | Closes |
| --- | --- | --- | --- |
| `lsojra` | 30 | §6.2 B11 closed field set, B12 step 4 table, probes P21.B-D | review F1 (blocking) |
| `lsojra` | 31 | §6.2 B12 step 3 arming rule, probe P21.F | self-found F2 |
| `lsojra` | 32 | §10.3, §11 refusal payloads, probe P21.E | review F1 item 5 |
| `lsojra` | 33 | §9 and §16.1 protocol bump to 5 | consequence of F1 |
| `lsojra` | 34 | tests 12.2.30a, 12.2.30b, 12.2.18b, §12.3 mutants | review F1 items 1-3 |
| `lsojra` | 35 | §12.4 launcher clause, second obligation | review F1, wiring |

Items 30 and 33 explicitly supersede earlier items (25 and 28) rather than
silently contradicting them, because `add_checklist_item` is the only checklist
mutation the CLI exposes and a stale item left unmarked is a trap for the
implementer.

### Sibling precondition payloads

Both sibling copies are separate payloads rather than links. Both were refreshed
to revision 5 and verified byte-identical to the committed file and to the task
outcome copy at SHA-256
`86a78926d40bdc677b333b0816fed79c854daa7c80b1d72139617795254107e9`, superseding
the `5109a922…8453` and `b631e658…c348` hashes quoted above.
