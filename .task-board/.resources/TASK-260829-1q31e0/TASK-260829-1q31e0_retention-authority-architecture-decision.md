# TASK-260829-1q31e0 retention authority architecture decision — revision 2

Decision date: 2026-08-29, Europe/Moscow.

## Decision

Retain the accepted trusted-effective-UID architecture and dedicated
profile-scoped lifecycle namespace. Add two contracts that were missing from
revision 1:

1. every foreground and read-only operation has explicit time and work
   budgets, and read-only status does not acquire the writer lock;
2. existing installations get a separate, explicit, two-phase legacy
   retirement command. It never adopts or deletes legacy evidence
   automatically.

Implementation is split into two atomic Tasks under
`STORY-260829-1xvrur`: the existing aggregate retention engine and a dependent
operator retirement flow. This is the smallest board that lets the retention
engine remain one ownership boundary without burying a destructive operator
workflow inside the same acceptance unit.

No daemon, privileged helper, broker ownership, Session Manager ownership,
zero-persistence mode, or live model/runtime/service/socket/endpoint is added.

## Exact baseline and source evidence

The source baseline remains exact current trunk
`6d051f54440d36e3ca3d132f8d9d1e78d46289de`; the assigned Story worktree is
zero commits behind local `main`. Its uncommitted lifecycle candidate is prior
producer evidence and was not modified or treated as landed source by this
architecture pass.

- Current-trunk `pi_state.go` derives per-run state below
  `runs/<run-state-key>` and uses non-blocking profile locks. Per-run state is
  intentionally unsuitable as the profile aggregate retention boundary.
- Current-trunk `pi_session_log.go` creates a random mode-`0600` JSONL file in
  the selected state `logs/` directory and appends without aggregate
  retention. The rejected candidate adds an unbounded aggregate scan and
  writer lock; it demonstrates the exact liveness gap but is not authority.
- `main.go` already separates non-launching operator commands (`runtime
  status|stop|quarantine`) from launch commands. A top-level
  `agents-infra lifecycle-logs ...` command follows that pattern without
  overloading Pi provider arguments.
- Shared-runtime locks use `LOCK_NB`; this source already treats busy ownership
  as a typed refusal rather than permission to block forever.
- Managed Pi remains Darwin/arm64-only. Linux and Windows requirements are
  compile-safe unsupported behavior, not new product implementations.

## Threat model

The effective UID is the managed-cache security principal. A malicious process
running as that UID is outside this feature's integrity threat model because it
can already rewrite or unlink the files directly. Random IDs, strict envelopes,
generation records, manifests, and confirmation hashes protect against
accidental collision, stale or corrupt evidence, path substitution, and buggy
scanner widening. They are not claimed to be unforgeable against the trusted
UID.

The implementation still protects or fails closed against other UIDs,
symlinks, hard links, unexpected types/modes/links, device/inode replacement,
partial publication, malformed or unreadable control records, concurrent run
IDs, scanner narrowing, and recursive deletion.

## Namespace and ownership contract

All new lifecycle logs use one profile root; per-run `agent/`, `sessions/`,
`client.lock`, and related state remain isolated.

```text
<profile-root>/lifecycle-logs/
  foreground.lock
  retention.lock
  generation.json
  entries/
    <UTC-start>-<128-bit-random-id>/
      record.json
      active.lock
      log.jsonl
  retirements/
    <plan-id>.result.json
```

An entry is managed and deletable only when descriptor-relative `O_NOFOLLOW`
inspection establishes the exact directory and three-child inventory, strict
record schema, modes, link counts, basename/record ID agreement, and
device/inode identities. `active.lock` must be acquired non-blocking before an
inactive entry can be deleted. Removal renames only a proven entry to the exact
`.deleting-<id>` tombstone form, unlinks only the three named children, removes
the empty directory, and fsyncs each parent. There is no recursive removal.

`record.json` is a strict, bounded control document. In addition to the entry
ID, UTC start, run-state key, and log identity, it stores `committed_bytes` and
`committed_records`. Append writes only at `committed_bytes`, syncs the log,
then atomically replaces and fsyncs the record. Recovery truncates an exact
known log tail only when the file is larger than `committed_bytes`; a shorter,
replaced, unreadable, or otherwise divergent log is unknown and preserved.

## Required configuration and validation

The profile `lifecycle_log_retention` table has no numeric defaults. Every
field below is required, positive, parsed overflow-safely, and included with
source provenance in status:

```toml
[agents.pi.profiles.<name>.lifecycle_log_retention]
max_count = <positive integer>
max_bytes = <positive integer>              # aggregate committed JSONL bytes
max_age_seconds = <positive integer>
create_timeout_ms = <positive integer>
append_timeout_ms = <positive integer>
close_timeout_ms = <positive integer>
status_timeout_ms = <positive integer>
maintenance_timeout_ms = <positive integer>
max_scan_entries = <positive integer>
max_scan_control_bytes = <positive integer>
max_mutations_per_operation = <positive integer>
```

Control documents have a fixed 4 KiB encoded-size ceiling. Configuration
validation requires `max_scan_entries >= max_count +
max_mutations_per_operation + 2` and checks that the corresponding 4 KiB
control-byte product fits `int64` and `max_scan_control_bytes`. This guarantees
that a healthy managed namespace is fully inspectable while still bounding
corrupt or foreign inventory. `max_bytes` counts committed `log.jsonl` bytes;
status separately reports complete managed-envelope bytes.

The extra fields are not tuning defaults. They are explicit product policy,
so an operator can see exactly how long a launch/event/status may wait and how
much corrupt inventory one call will inspect.

## Clock, cancellation, and filesystem boundary

Each operation receives a context and an injected clock. Its effective deadline
is the earlier of the caller deadline and `start + <operation>_timeout_ms`.
Elapsed budgets use the monotonic component of the clock; age retention uses
UTC wall time. Tests inject both clocks and never sleep.

Lock acquisition always uses `LOCK_NB` plus clock-driven bounded retry. The
deadline and cancellation are checked before and after each retry, directory
entry, bounded control-file read, identity revalidation, and exact mutation.
No retry survives the effective deadline. A budget or cancellation result is a
stable typed refusal, never absence or health.

This is a cooperative userspace bound between filesystem syscalls. A single
kernel syscall cannot be cancelled portably after entry. The product contract
therefore requires the already-canonical user cache to be a local filesystem;
an unresponsive kernel/filesystem is a platform failure, not a condition that
Go can turn into a hard real-time guarantee. No operation adds network I/O.

## Lock priority and starvation contract

Foreground create, append, and close-maintenance acquire
`foreground.lock` shared, then `retention.lock` exclusive, both within their
own deadline. Explicit maintenance attempts `foreground.lock` exclusive and
non-blocking once; if any foreground holder or waiter is present it returns
`lifecycle_log_maintenance_busy` without mutation. Once maintenance owns the
gate, its total deadline and mutation cap bound how long new foreground calls
can wait.

Read-only status never acquires either lock. It uses the generation-fenced
snapshot below. Therefore repeated status calls cannot starve launch or append.
Repeated hostile maintenance invocations are a same-UID denial of service and
outside the threat model, but every individual invocation remains bounded and
cannot jump ahead of an already admitted foreground holder.

## Generation-fenced read-only status

Every aggregate mutation occurs under `retention.lock`:

1. atomically publish an odd generation with operation kind and fsync it;
2. perform the bounded mutation;
3. atomically publish the next even generation and fsync it.

Status reads and validates an even generation, performs the same
descriptor-relative scanner in read-only mode, then rereads the generation. It
publishes a coherent result only when both records are the same even generation
and all scanned identities still match. Odd, changed, missing-in-an-existing
root, malformed, or unreadable generation evidence yields
`observation=unknown`, `within_policy=false`, and `soak_ready=false`.

A missing aggregate root is `absent` only after the profile root and bounded
legacy locations were read successfully. A read error is never mapped to
absence. Active-lock state may change during a scan; either state is a valid
instantaneous observation, while any associated file mutation changes the
generation and invalidates the snapshot.

## Bounded scanner and continuation

One scanner implementation serves create, append, close/recovery, status, and
legacy inventory. Modes select aggregate-only or aggregate-plus-legacy roots;
the ownership classifier is identical.

- At most `max_scan_entries` directory entries and
  `max_scan_control_bytes` control bytes are inspected in one operation.
- JSONL payloads are statted, not read, during ordinary status/admission.
- The scanner stops before exceeding either cap or the effective deadline.
- A truncated scan reports observed counts/bytes as lower bounds,
  `scan_complete=false`, an explicit exhaustion reason, and an opaque
  continuation token. It never reports healthy policy.
- On Darwin the token contains a versioned directory cookie plus directory
  device/inode/change-time facts. It is navigation state, not authority. A
  changed directory invalidates the token and restarts inventory; it never
  broadens deletion scope.
- `--print-config` and compose return one bounded page. The operator
  `agents-infra lifecycle-logs status` command accepts the opaque continuation
  token for further read-only pages.

Pruning is planned only after a complete aggregate scan. If the required plan
exceeds `max_mutations_per_operation`, create refuses with
`lifecycle_log_maintenance_required` before creating a log. If a later exact
unlink fails after safe progress, admission still refuses, reports the precise
partial result, and the next bounded operation converges from a fresh scan.
No unproven entry is touched.

## Operation semantics

| Operation | Bound and lock behavior | Budget/failure result |
| --- | --- | --- |
| Create (`RunPi` and `runSharedPiSession` -> `openPiSessionLogAt`) | Shared foreground gate; exclusive retention lock; complete aggregate scan before recovery/prune/publication | Timeout/cancel/scan exhaustion refuses launch before the lifecycle entry is admitted |
| Append (`piSessionLog.event`) | In-process mutex; shared foreground gate; exclusive retention lock; fd/path/record/active identity revalidated before reservation and immediately before append | Drops the whole record, increments `dropped_records`, leaves file size unchanged, publishes typed error and unknown footprint |
| Close (`piSessionLog.close`) | Syncs/closes the log and releases `active.lock` regardless of maintenance availability; bounded prune is best effort under the foreground/retention locks | Never blocks process cleanup indefinitely; skipped/failed prune is reported and recovered by a later operation |
| Status (`--print-config`, compose, run report, `lifecycle-logs status`) | No lock; one generation-fenced bounded scan page; no mutation | Returns unknown with reason/cursor on concurrent mutation, timeout, cancellation, read failure, or scan cap |
| Explicit retirement apply | Exclusive foreground gate attempted once; exclusive retention lock; prevalidates a complete bounded plan page before any unlink | Busy/changed/unreadable evidence refuses; crash or deadline after exact deletes reports resumable safe progress |

Create/recovery recognizes exact `.creating-<id>` and `.deleting-<id>` states.
A held active lock preserves the entry. A free, complete staging envelope is
published; a free strict subset containing only launcher-defined children is
removed exactly; unexpected or unreadable evidence is unknown and preserved.
A deletion tombstone is resumed only when its remaining exact inventory and
record identity are still proven.

## Deterministic retention

Active reservations consume count and committed-byte budgets and are never
evicted. Inactive proven entries sort newest first by `(started_at, entry_id)`.
The scanner keeps the newest prefix satisfying count, committed JSONL bytes,
complete-envelope bytes, and age; deletions execute oldest first. An inactive
entry is expired only when `started_at < now-max_age`. Active exhaustion
returns `lifecycle_log_retention_exhausted`.

No create occurs after a truncated scan, incomplete prune, or unestablished
aggregate identity. Legacy evidence does not count as managed allowance and
does not automatically block new aggregate writes, but it keeps global health
and `soak_ready` false until explicitly retired.

## Status contract

Every status surface exposes:

- policy values and source;
- aggregate root, generation, observation, operation deadline, scan limits,
  `scan_complete`, exhaustion reason, continuation availability, and errors;
- managed count, committed JSONL bytes, complete-envelope bytes, active count,
  expired count, oldest/newest start;
- legacy, foreign, and unknown count/bytes with exact-vs-lower-bound markers;
- pruned count/bytes, recovered staging/tombstone count, dropped records, and
  last error in run reports;
- `within_policy` and `soak_ready`.

`within_policy` is true only for a complete stable aggregate snapshot satisfying
the three retention bounds with no aggregate unknown evidence. `soak_ready`
additionally requires a complete legacy scan with zero legacy lifecycle and
unknown evidence. Read failure, odd/changing generation, truncation, or budget
exhaustion is unknown and false for both.

Stable error codes are:

- `lifecycle_log_lock_timeout`
- `lifecycle_log_operation_cancelled`
- `lifecycle_log_scan_budget_exhausted`
- `lifecycle_log_operation_timeout`
- `lifecycle_log_maintenance_busy`
- `lifecycle_log_maintenance_required`
- `lifecycle_log_retention_exhausted`
- `lifecycle_log_evidence_unknown`
- `lifecycle_log_retirement_plan_changed`
- `lifecycle_log_retirement_confirmation_required`

## Explicit legacy retirement workflow

Legacy files are historical lifecycle-shaped JSONL files in exact canonical
`logs/` and `runs/<64-lowercase-hex>/logs/` roots. Ordinary retention/status
never adopts, moves, or deletes them. Unrelated, ambiguous, symlinked,
hard-linked, wrong-mode, unreadable, or identity-changing evidence is excluded
from the candidate set and preserved as foreign/unknown.

The operator flow is a separate top-level, non-launching CLI:

```text
agents-infra lifecycle-logs status \
  --project DIR --profile NAME [--continuation TOKEN] --json

agents-infra lifecycle-logs retire-legacy \
  --project DIR --profile NAME --dry-run --output PLAN.json

agents-infra lifecycle-logs retire-legacy \
  --project DIR --profile NAME --apply-plan PLAN.json \
  --confirm-plan-sha256 HEX --confirm-delete
```

Dry-run is the only plan-creation mode and never mutates state. It writes a new
mode-`0600` file with `O_EXCL`, strict schema/version, random plan ID, exact
project/profile keys, policy source, root identities, bounded candidate page,
per-candidate descriptor-relative source, device/inode, size, mode, link count,
and timestamps, plus lower-bound excluded evidence and continuation state. It
prints the plan SHA-256. The hash confirms that apply uses exactly the bytes the
operator reviewed; it is not a same-UID signature.

Apply requires all three of `--apply-plan`, exact
`--confirm-plan-sha256`, and `--confirm-delete`; there is no force flag or
implicit action. It strictly decodes the bounded plan, re-resolves the same
profile, obtains the maintenance gate, and revalidates the entire not-yet-absent
page before the first mutation. Any changed, unreadable, or ambiguous candidate
refuses the page without deletion. Exact candidates are unlinked by name from
their already-open legacy `logs` directory and that directory is fsynced. It
never removes a `logs` directory, run directory, or parent recursively.

The result is atomically recorded under `retirements/<plan-id>.result.json` and
emitted as JSON. It lists deleted, already-absent-after-prior-apply, refused,
and remaining candidates, exact bytes, continuation, and post-operation
status. Reapplying the same plan is idempotent. A crash after an unlink but
before the result update is recovered as `already_absent_unattested`; no new
path is inferred. Repeated bounded dry-run/apply pages eventually remove every
unchanged exact legacy candidate. A final complete status scan can then set
`within_policy=true` and `soak_ready=true` without manual filesystem surgery.

## Rejected alternatives

- A same-UID daemon, shared broker, or Session Manager owner does not create a
  stronger security principal and adds an availability dependency.
- A privileged helper/other UID creates real authority but is disproportionate
  to user-cache retention and outside the current installation contract.
- Passive HMACs, Keychain secrets, xattrs, ACLs, flags, birth time, modes, and
  hard links are still mintable or replaceable by the trusted UID.
- Zero persistence removes the retained diagnostics and makes count/age policy
  meaningless.
- Automatic legacy adoption, movement, deletion, or background cleanup is
  rejected. Only the explicit plan/hash/delete-confirmation flow may retire
  exact legacy evidence.
- Permanent unhealthy status was rejected because it gives existing clean
  installations no supported upgrade path.

## Board decomposition and dependencies

### Task A — aggregate lifecycle retention engine

Existing `TASK-260829-1q31e0` owns explicit configuration, dedicated aggregate
envelopes, generation-fenced bounded scanner, create/append/close/recovery,
deterministic pruning, production launch wiring, status/run reports, docs, and
adversarial/race/soak/compile gates.

### Task B — explicit legacy retirement operator flow

Add sibling `TASK-260829-2v7x1u:
provide-explicit-pi-lifecycle-legacy-retirement`. It owns the
non-launching status pagination and two-phase dry-run/apply CLI, plan/result
schemas, exact revalidation/unlink, resume semantics, docs, and negative tests.
It is blocked by Task A because it consumes Task A's scanner, locks, envelope
classification, policy, and diagnostics. No other dependency is needed.

No research, daemon, documentation-only, soak-only, or review Task is added.
Docs and validation remain local gates of their owning code Task.

## Requirement traceability

| Requirement | Task A | Task B |
| --- | --- | --- |
| Lifecycle AC1: explicit positive count/byte/age bounds | Required retention and operation-budget configuration | Reuses validated policy; refuses divergent plan policy |
| Lifecycle AC2: aggregate root and per-run isolation | Owns namespace and both production launch paths | Reads only exact aggregate/legacy roots; never merges session state |
| Lifecycle AC3: strict deletion authority and active preservation | Owns envelopes, locks, identity checks, exact non-recursive removal | Revalidates explicit legacy plan and exact non-recursive unlink |
| Lifecycle AC4: deterministic pruning and fail-closed evidence | Owns complete-scan-before-prune, generation snapshot, budget refusals | Preserves excluded/changed evidence and reports bounded resume state |
| Lifecycle AC5: status and week-scale proof | Owns same-scanner status and fresh-run eight-week soak | Proves upgraded clean profile reaches healthy/soak-ready after explicit retirement |
| Revision-required finding 1: bounded liveness | Owns explicit per-operation deadlines, work caps, cancellation, lock priority, and status generation fence | Uses maintenance deadline/cap and never waits ahead of foreground work |
| Revision-required finding 2: upgrade path | Reports legacy exactly or as bounded unknown; never auto-mutates | Owns reviewed plan, confirmation, deletion, resume, and final healthy status |

## Justified gap records

### Gap G1 — bounded operation liveness

- Missing piece: revision 1 specified an unbounded writer lock and full scan for
  create, append, close, recovery, prune, and status.
- Requirement enabled: hours/days/weeks stability, lifecycle AC2-AC5, and the
  explicit revision-required liveness gate.
- Consequence if omitted: a held lock or large corrupt/foreign inventory can
  hang launch, append, cleanup, and read-only status indefinitely.
- Closure: Task A adds required deadlines, work/mutation caps, monotonic
  cancellation, writer-priority gating, lock-free generation-fenced status,
  typed refusal, and production-path attacks.
- Self-check: lifecycle AC1-AC5, Task scope, architecture decision brief,
  architecture independent-review scope, revision-required findings, and all
  stated exclusions were checked. No section answers liveness already or puts
  bounded local filesystem work out of scope.

### Gap G2 — explicit legacy retirement

- Missing piece: revision 1 preserves legacy forever while making it keep
  `soak_ready=false`; existing clean installations have no supported recovery.
- Requirement enabled: truthful status, week-scale stable upgrade behavior,
  and the explicit revision-required upgrade gate.
- Consequence if omitted: upgraded profiles remain permanently unhealthy or
  require undocumented manual filesystem deletion.
- Closure: Task B adds an explicit audited dry-run/plan/hash/confirm-delete
  workflow with bounded pages, descriptor-relative revalidation, exact unlink,
  resumability, and post-operation health evidence.
- Self-check: lifecycle AC1-AC5, Task scope, architecture decision brief,
  architecture independent-review scope, revision-required findings, and
  out-of-scope clauses were checked. The scope excludes *automatic* legacy
  adoption/deletion, not explicit operator retirement. No daemon, privileged
  helper, zero-persistence behavior, or live service contact is introduced.

## Exact adversarial and narrowing tests

Task A tests must name and drive production call sites:

- Hold `retention.lock`; `RunPi -> openPiSessionLogAt` refuses within
  `create_timeout_ms` before runtime/provider side effects.
- Hold the lock after opening a real log; `piSessionLog.event` returns within
  `append_timeout_ms`, drops the whole record, and leaves committed size
  unchanged. Narrowing the deadline check by one retry fails.
- Hold the lock during `piSessionLog.close`; process cleanup and active-lock
  release still occur within `close_timeout_ms`; prune is reported skipped.
- Hammer `--print-config`/compose status while launching/appending; status takes
  no lock, cannot starve foreground operations, and odd/changing generation is
  unknown.
- Populate more than `max_scan_entries` or
  `max_scan_control_bytes`; create refuses before prune/create, status returns
  bounded unknown, and every unvisited/foreign/active entry remains.
- Narrow either scan cap by one and prove the exact-boundary fixture fails;
  removing the cap alone is not accepted mutant evidence.
- Crash at each create, committed-append, and tombstoned-delete step; recovery
  only completes/removes exact proven state and preserves ambiguity.
- Mutate active mode/link/path/inode/record between reservation and append;
  the complete event is dropped and health is unknown.
- Fresh run IDs through exclusive, standalone, and shared production selection
  converge on one aggregate root; narrowing either caller to a run directory
  fails.
- Eight fake-clock weeks remain within count, committed JSONL bytes, managed
  envelope bytes, age, scan-work, and operation-time budgets with
  `soak_ready=true` in the clean fixture.

Task B tests must drive `runLifecycleLogs` CLI dispatch, not helper-only APIs:

- Dry-run never mutates and inventories only exact historical lifecycle
  candidates; symlink, hard-link, wrong-mode, unreadable, extra-child, changed,
  and unrelated evidence is preserved/reported.
- Apply without the exact plan hash or `--confirm-delete` refuses before the
  production unlink call. A narrowing mutant accepting any hash prefix fails.
- Replace/change one candidate after dry-run; whole prevalidation refuses with
  every candidate preserved.
- Exceed page entry/control-byte/deadline budgets; output is bounded,
  continuation is present, and no unplanned candidate is touched.
- Crash after the first exact unlink; reapplying the same plan reports the
  absent candidate and safely processes the remainder.
- Hold a foreground shared gate or run concurrent append/create; maintenance
  returns `lifecycle_log_maintenance_busy` and cannot starve foreground work.
- Retire all exact legacy pages from canonical and fresh run roots, then drive
  the production status surface and require a complete healthy/`soak_ready`
  result while foreign/ambiguous evidence still prevents that claim.

Both Tasks retain focused race tests, uncached relevant Go suites, `go vet`,
`go build`, gofmt/diff integrity, Darwin product tests, and Linux/Windows
compile gates. Neither test matrix contacts a live model/runtime/service,
socket, or endpoint.
