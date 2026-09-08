# TASK-260829-1q31e0 lifecycle retention architecture — revision 3

Decision date: 2026-08-30, Europe/Moscow.

## Verdict

The existing two-Task Story is the smallest development-ready decomposition.
No daemon, helper, research, documentation-only, soak-only, or review Task is
added. Task A owns one profile aggregate retention state machine. Dependent
Task B owns the explicitly confirmed legacy-deletion CLI.

Implementation remains a conditional NO-GO until
`TASK-260829-ivybt9` is independently accepted, integrated, merged into the
protected branch, and a fresh managed retention workspace records that exact
post-pressure commit as its selected base. Current `6d051f54440d36e3ca3d132f8d9d1e78d46289de`
and the dirty revision-5 retention workspace are historical evidence only.

Revision 3 closes all six implementation-blocking ambiguities identified by
the post-pressure replay preflight.

## Source and requirement boundary

- Current local `HEAD`, `main`, and cached `origin/main` were all
  `6d051f54440d36e3ca3d132f8d9d1e78d46289de` during this decision.
- Current source derives standalone/shared client state below
  `runs/<run-key>` in `tools/agents-infra/internal/infra/pi_state.go`; the
  aggregate lifecycle root must therefore be a first-class profile-root path,
  not `PiStatePaths.LogsDir` from a client run.
- Current `piSessionLog.event` and `close` have no retention, cancellation, or
  durable committed-byte protocol. Task A replaces this lifecycle boundary
  through the real `RunPi` and `runSharedPiSession` call sites.
- Pressure replacement `TASK-260829-ivybt9` was `reviewing` when this decision
  was written. Retention must preserve its six overlapping paths and rerun its
  accepted production pressure slice.
- Requirement traceability: lifecycle retention AC1-AC5; architecture revision
  2 sections Required configuration, Generation-fenced status, Bounded scanner
  and continuation, Operation semantics, Explicit legacy retirement workflow,
  Gap G1, and Gap G2; post-pressure replay holes 1-6.

## Namespace and generation records

Task A creates one profile aggregate root, independent of run state:

```text
<profile-root>/lifecycle-logs/
  foreground.lock
  retention.lock
  generation.json
  legacy-generation.json
  entries/
    <UTC-start>-<128-bit-random-id>/
      record.json
      active.lock
      log.jsonl
```

Task B does not add an internal `retirements/` history. Plans and results are
bounded, operator-selected external artifacts.

`generation.json` fences aggregate entry publication, append commits, recovery,
and pruning. `legacy-generation.json` fences Task B legacy deletion. Task A
creates both as strict even generation zero records when it creates a new
aggregate root, so absence inside an existing root is unknown rather than a
legacy fallback.

Each record is strict, versioned, at most 4 KiB, mode `0600`, single-link, and
atomically replaced plus parent-fsynced. It carries generation, parity/state,
operation ID, operation kind, scope (`aggregate` or `legacy`), start time, and
the bounded recovery identity needed by that operation. The generation number
increments monotonically without wrap; exhaustion is a typed refusal.

Status reads both generations before scanning and again after scanning. A
whole-profile healthy result requires both pairs to be identical and even.
Aggregate admission depends only on a stable aggregate generation, so an
interrupted explicit legacy retirement cannot indefinitely block foreground Pi
launches; top-level status remains unknown and `soak_ready=false` until the
legacy generation is recovered.

## Hole 1 — odd aggregate generation recovery

The next Task-A writer that obtains the foreground and retention locks owns
bounded recovery before starting a new operation. It does not publish another
odd generation first.

1. Strictly read the odd record and perform one complete bounded aggregate
   scan. Missing, malformed, unreadable, identity-changing, over-budget, or
   unrecognized evidence returns `lifecycle_log_evidence_unknown`, preserves
   every unproven path, leaves the generation odd, and admits no create/append.
2. Recover only exact protocol states named by the odd operation:
   - exact free `.creating-<id>` complete envelopes are published; exact free
     launcher-defined strict subsets are removed child-by-child;
   - exact `.deleting-<id>` tombstones resume named child unlinks and empty-dir
     removal without recursion;
   - an exact log longer than `record.json.committed_bytes` is truncated to the
     committed boundary; a shorter, replaced, linked, wrong-mode, or unreadable
     log is unknown and preserved;
   - an already-published create, fully removed tombstone, or fully committed
     append is recognized as completed only from the strict remaining state.
3. Recovery checks caller cancellation, monotonic deadline, scan caps, and the
   mutation cap before and after every bounded step. Safe partial progress may
   remain under the same odd generation; the next writer resumes it. A new
   foreground operation never follows partial recovery.
4. After a complete scan proves that every exact partial is converged and no
   unknown aggregate evidence exists, atomically publish `odd+1` as even with
   recovery counters, fsync, and only then begin the requested operation with
   the next odd generation.

Crash tests cover failure before the first mutation, after each exact child
mutation or append sync/record replace, and immediately before the even write.
Status never repairs state and remains lock-free and unknown while aggregate
generation is odd.

## Hole 2 — legacy mutation fencing and recovery

Task B apply obtains the maintenance gate and retention lock, validates the
plan page, and publishes an odd `legacy-generation.json` before the first
unlink. Every status-visible legacy unlink occurs while that record is odd.
After the bounded result is durably published, apply publishes the next even
legacy generation.

An interrupted legacy apply is recovered only by a subsequent apply of the
same exact plan page or an explicit Task-B recovery path. It revalidates the
plan digest, root identities, and all remaining candidates; already absent
planned identities are reported as `already_absent_unattested`, never rebound
to a new inode. Ambiguous evidence leaves the legacy generation odd and
top-level status unknown. Task A can still admit aggregate work from its
independent stable generation, but cannot claim `soak_ready`.

Dry-run and status pagination are read-only and never alter either generation.
Aggregate and legacy generation reads are both part of the status control-byte
budget.

## Hole 3 — exact scanner units and continuation health

`max_scan_entries` counts every directory entry returned by every `readdir`
performed by the scanner, not logical envelopes. This includes the five fixed
aggregate-root entries, every child of `entries/`, and all three children of a
complete envelope. It also includes each traversed canonical legacy-log entry,
`runs/` entry, run `logs/` entry, staging entry, and tombstone entry. `.` and
`..` are not returned and are not charged. Opening or statting a known name is
not a directory-entry charge.

`max_scan_control_bytes` counts every byte actually returned while reading a
strict control document, including repeated generation reads. A bounded reader
may consume one sentinel byte to prove oversize; that byte is charged. JSONL
payloads are statted, not read, during scan/status.

Configuration validation performs these overflow-safe minimum checks for a
healthy aggregate plus one operation's recoverable partials:

```text
managed_slots = max_count + max_mutations_per_operation
required_scan_entries = 5 + 4 * managed_slots
required_scan_control_bytes = 4096 * (managed_slots + 4)
max_scan_entries >= required_scan_entries
max_scan_control_bytes >= required_scan_control_bytes
```

The four control reads reserve aggregate and legacy generation reads before and
after a status scan. Operations that read fewer controls still use the same
configured cap. All products and sums are checked before conversion.

A continuation token is versioned navigation state only. It binds project and
profile keys, policy digest, scan mode/phase, both starting generations,
directory device/inode/change facts, and the platform directory cookie. It
does not carry deletion authority or authoritative aggregate counters. Every
page reports exact page counters and explicit global lower bounds with
`scan_complete=false`, `within_policy=false`, and `soak_ready=false` when a
continuation is needed. Even a final continuation page cannot claim whole-scan
health from page-local or caller-carried counters. Health is published only by
a fresh scan from the beginning that completes within one operation and passes
both final generation rereads. Task B converges by deleting bounded first pages
and restarting inventory; after exact legacy evidence is gone, a fresh complete
scan proves the healthy upgrade state.

Changed generation, policy, directory identity, or cookie invalidates the
token and returns unknown without mutation.

## Hole 4 — managed-envelope byte authority

Add required positive `max_envelope_bytes` to
`lifecycle_log_retention`; it has no default. `max_bytes` remains aggregate
committed `log.jsonl` bytes. `max_envelope_bytes` is the aggregate logical byte
sum of each proven managed envelope's current `log.jsonl`, encoded
`record.json`, and `active.lock` file sizes. Directory metadata and filesystem
allocation blocks are reported separately when available and are not portable
policy authority.

Validation requires, overflow-safely:

```text
max_envelope_bytes >= max_bytes + 4096 * max_count
```

This permits all configured committed JSONL bytes plus the maximum strict
control record for every retained entry. Active reservations consume count,
committed-byte, and envelope-byte allowance. Pruning keeps the newest inactive
prefix satisfying all four authorities: count, committed JSONL bytes, managed
envelope bytes, and age. Status exposes configured and observed values
separately. Narrowing either byte authority by one must fail its exact-boundary
production test.

## Hole 5 — bounded external retirement artifacts

Remove the internal `retirements/<plan-id>.result.json` design. Task B requires
operator-selected `--output` for dry-run plans and `--result-output` for apply
results. Both targets must be outside the managed profile tree and are created
as strict mode-`0600`, single-link files using a same-directory temporary,
fsync, and no-replace publication. Existing targets refuse; there is no
overwrite, implicit archive, or managed history.

Plan and result schemas have bounded strings and at most one page of candidates
or `max_mutations_per_operation` result rows. Their maximum encoded size is an
overflow-safe product of the configured caps and fixed schema ceilings, is
exposed in dry-run/status diagnostics, and is enforced before publication.
Stdout emits the same bounded summary. Repeated invocations can create files
only at paths explicitly chosen by the operator; the profile-wide eight-week
footprint therefore contains no secondary retirement log system.

## Hole 6 — final per-candidate unlink revalidation

Apply performs whole-page prevalidation before its first deletion. Immediately
before every later unlink it additionally:

1. opens the planned candidate through the already-open legacy `logs` parent
   using `O_NOFOLLOW`;
2. compares fd and path device/inode, regular-file type, effective UID, exact
   mode, link count, size, and plan-bound timestamps to the reviewed plan;
3. closes the proof fd and calls only `unlinkat(parentFD, exactName, 0)` before
   the next cancellation/deadline boundary;
4. fsyncs the already-open parent after each successful bounded unlink.

A mismatch before the first unlink refuses the whole page with no deletion. A
mismatch after safe earlier progress stops immediately, preserves the changed
candidate and every later candidate, leaves resumable exact result evidence,
and returns `lifecycle_log_retirement_plan_changed`. Malicious mutation by the
trusted effective UID remains outside the integrity threat model; ordinary
concurrency, stale plans, and path substitution remain fail-closed.

## Development tasks and dependencies

### Task A — `TASK-260829-1q31e0`

Owns explicit policy including `max_envelope_bytes`, first-class aggregate
paths, both generation records, odd aggregate recovery, strict envelopes,
bounded scanner, create/append/close/prune, status types, exclusive/standalone/
shared wiring, docs, pressure composition, adversarial tests, races, fake-clock
eight-week proof, and platform compile gates.

Hard dependency: `TASK-260829-ivybt9`. Execution requires its accepted merged
post-pressure trunk and a fresh managed Story workspace from that exact OID.

### Task B — `TASK-260829-2v7x1u`

Owns production CLI pagination, dry-run plan, legacy generation fencing and
recovery, exact confirmation/apply, per-candidate final revalidation,
external-only bounded results, final clean-upgrade proof, docs, adversarial
tests, races, and platform compile gates.

Hard dependency: Task A only; the pressure dependency is transitive.

## Negative and narrowing gates added by revision 3

- Crash each aggregate generation phase and require exact recovery or a stable
  unknown refusal; deleting the final generation reread or narrowing recovery
  identity must fail named production tests.
- Crash Task B after a legacy unlink and before its even generation; status
  must be unknown/`soak_ready=false` until exact apply recovery.
- Count every envelope child and all four generation reads. Narrow either scan
  formula by one and require the exact-boundary production case to fail.
- Resume on a final continuation page and prove it cannot publish health;
  removing the fresh-from-start requirement must yield a false healthy result.
- Exercise exact `max_bytes` and `max_envelope_bytes` boundaries independently;
  narrowing either by one must fail.
- Repeated dry-run/apply runs must leave the managed profile root unchanged
  except for generation-fenced legacy source deletion; no result history may
  appear under the root.
- Swap a later candidate after whole-page validation and before its unlink;
  earlier exact progress may remain, but the replacement and all later evidence
  must survive.
- Rerun the accepted pressure production/race slice across all six overlapping
  files. Removing final pressure publication freshness or recoupling status to
  admission generation must fail the inherited named tests.

## Gap and out-of-scope self-verification

No new beyond-literal-spec Task is required. Holes 1-4 and 6 specify mechanisms
already necessary for lifecycle AC1-AC5 and Gap G1. Hole 5 narrows Gap G2's
result persistence so the explicit retirement workflow cannot create a second
unbounded artifact family. Lifecycle AC1-AC5, both revision-required findings,
architecture revision 2, the post-pressure overlap/replay gates, and the entire
out-of-scope list were checked. The decision adds no daemon, privileged helper,
automatic legacy adoption/deletion, zero-persistence mode, network behavior,
or live model/runtime/service/socket/endpoint contact.

## Evidence anomaly

The board precondition named `post-pressure-retention-replay-preflight.md`
contained only the literal path
`.temp/TASK-260829-1q31e0/post-pressure-retention-replay-preflight.md`; that
temporary file was absent. Its exact prior content was recovered read-only from
the local Codex rollout that displayed it before attachment. Revision 3 is
attached as real board-owned bytes so future runs do not depend on that missing
temporary path.
